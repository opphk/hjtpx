package database

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type EnhancedDBRouter struct {
	masterDB               *gorm.DB
	slaveDBs               []*gorm.DB
	slaveWeights           []int
	slaveStatus            []*SlaveNodeStatus
	currentSlave           uint32
	enabled                bool
	loadBalanceMode        string
	mu                     sync.RWMutex
	healthChecker          *EnhancedSlaveHealthChecker
	metrics                *EnhancedRouterMetrics
	replicationMonitor     *ReplicationMonitor
	failoverMode           bool
	lastFailoverTime       time.Time
}

type SlaveNodeStatus struct {
	Index          int
	Host           string
	Port           string
	Healthy        bool
	Latency        time.Duration
	LastCheck      time.Time
	FailCount      int
	ReplicationLag time.Duration
	Weight         int
	ActiveQueries  int32
}

type EnhancedRouterMetrics struct {
	MasterQueries      atomic.Int64
	SlaveQueries       atomic.Int64
	FailedQueries      atomic.Int64
	SlaveSwitches      atomic.Int64
	LastSwitchTime     atomic.Value
	AvgLatency         atomic.Int64
	FailoverCount      atomic.Int64
	ReplicationLagAlerts atomic.Int64
}

type EnhancedSlaveHealthChecker struct {
	dbRouter           *EnhancedDBRouter
	interval           time.Duration
	enabled            bool
	stopCh             chan struct{}
	wg                 sync.WaitGroup
	failoverEnabled    bool
	maxFailCount       int
	lagThreshold       time.Duration
	consecutiveFailures map[int]int
}

type ReplicationMonitor struct {
	dbRouter         *EnhancedDBRouter
	checkInterval    time.Duration
	maxLagThreshold  time.Duration
	enabled          bool
	stopCh           chan struct{}
	wg               sync.WaitGroup
	lastLagCheck     time.Time
	avgLag           atomic.Int64
}

type ReplicationStatus struct {
	SlaveIndex     int
	Host           string
	ReplicationLag time.Duration
	IsHealthy      bool
	LastCheck      time.Time
}

var enhancedRouter *EnhancedDBRouter

func InitReadWriteSeparationV2(cfg *config.Config) error {
	if !cfg.Database.ReadWriteSeparation.Enabled {
		enhancedRouter = &EnhancedDBRouter{
			enabled:         false,
			loadBalanceMode: cfg.Database.ReadWriteSeparation.LoadBalanceStrategy,
		}
		return nil
	}

	var err error

	masterDB, err := connectDBEnhanced(
		cfg.Database.ReadWriteSeparation.Master.Host,
		cfg.Database.ReadWriteSeparation.Master.Port,
		cfg.Database.ReadWriteSeparation.Master.User,
		cfg.Database.ReadWriteSeparation.Master.Password,
		cfg.Database.ReadWriteSeparation.Master.DBName,
		cfg.Database.ReadWriteSeparation.Master.SSLMode,
		cfg,
	)
	if err != nil {
		return fmt.Errorf("failed to connect master database: %w", err)
	}

	slaveDBs := make([]*gorm.DB, 0)
	slaveWeights := make([]int, 0)
	slaveStatus := make([]*SlaveNodeStatus, 0)

	for i, slaveCfg := range cfg.Database.ReadWriteSeparation.Slaves {
		slaveDB, err := connectDBEnhanced(
			slaveCfg.Host,
			slaveCfg.Port,
			slaveCfg.User,
			slaveCfg.Password,
			slaveCfg.DBName,
			slaveCfg.SSLMode,
			cfg,
		)
		if err != nil {
			log.Printf("Warning: failed to connect slave database %s: %v", slaveCfg.Host, err)
			continue
		}
		slaveDBs = append(slaveDBs, slaveDB)
		slaveWeights = append(slaveWeights, slaveCfg.Weight)
		slaveStatus = append(slaveStatus, &SlaveNodeStatus{
			Index:          i,
			Host:           slaveCfg.Host,
			Port:           slaveCfg.Port,
			Healthy:        true,
			Latency:        0,
			FailCount:      0,
			ReplicationLag: 0,
			Weight:         slaveCfg.Weight,
		})
	}

	enhancedRouter = &EnhancedDBRouter{
		masterDB:       masterDB,
		slaveDBs:       slaveDBs,
		slaveWeights:   slaveWeights,
		slaveStatus:    slaveStatus,
		enabled:        true,
		loadBalanceMode: cfg.Database.ReadWriteSeparation.LoadBalanceStrategy,
		currentSlave:   0,
		metrics:        &EnhancedRouterMetrics{},
	}

	if cfg.Database.ReadWriteSeparation.AutoFailover {
		enhancedRouter.healthChecker = NewEnhancedSlaveHealthChecker(enhancedRouter, 10*time.Second)
		enhancedRouter.healthChecker.Start()

		enhancedRouter.replicationMonitor = NewReplicationMonitor(enhancedRouter, 5*time.Second, 30*time.Second)
		enhancedRouter.replicationMonitor.Start()
	}

	log.Println("Enhanced read-write separation initialized successfully")
	return nil
}

func NewEnhancedSlaveHealthChecker(router *EnhancedDBRouter, interval time.Duration) *EnhancedSlaveHealthChecker {
	return &EnhancedSlaveHealthChecker{
		dbRouter:           router,
		interval:          interval,
		enabled:           true,
		stopCh:           make(chan struct{}),
		failoverEnabled:   true,
		maxFailCount:      3,
		lagThreshold:      30 * time.Second,
		consecutiveFailures: make(map[int]int),
	}
}

func (shc *EnhancedSlaveHealthChecker) Start() {
	shc.wg.Add(1)
	go shc.checkLoop()
}

func (shc *EnhancedSlaveHealthChecker) Stop() {
	close(shc.stopCh)
	shc.wg.Wait()
}

func (shc *EnhancedSlaveHealthChecker) checkLoop() {
	defer shc.wg.Done()

	ticker := time.NewTicker(shc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-shc.stopCh:
			return
		case <-ticker.C:
			shc.checkAllSlaves()
			if shc.failoverEnabled {
				shc.evaluateFailover()
			}
		}
	}
}

func (shc *EnhancedSlaveHealthChecker) checkAllSlaves() {
	for i, slave := range shc.dbRouter.slaveDBs {
		status := shc.dbRouter.slaveStatus[i]
		start := time.Now()

		sqlDB, err := slave.DB()
		if err != nil {
			status.Healthy = false
			status.FailCount++
			status.LastCheck = time.Now()
			shc.consecutiveFailures[i]++
			continue
		}

		if err := sqlDB.Ping(); err != nil {
			status.Healthy = false
			status.FailCount++
			status.LastCheck = time.Now()
			shc.consecutiveFailures[i]++
			continue
		}

		status.Latency = time.Since(start)
		status.Healthy = true
		status.FailCount = 0
		status.LastCheck = time.Now()
		shc.consecutiveFailures[i] = 0
	}
}

func (shc *EnhancedSlaveHealthChecker) evaluateFailover() {
	healthyCount := 0
	for _, status := range shc.dbRouter.slaveStatus {
		if status.Healthy && status.ReplicationLag < shc.lagThreshold {
			healthyCount++
		}
	}

	if healthyCount == 0 && len(shc.dbRouter.slaveDBs) > 0 {
		if !shc.dbRouter.failoverMode {
			shc.dbRouter.enterFailoverMode()
		}
	} else if shc.dbRouter.failoverMode && healthyCount > 0 {
		shc.dbRouter.exitFailoverMode()
	}
}

func (r *EnhancedDBRouter) enterFailoverMode() {
	r.mu.Lock()
	r.failoverMode = true
	r.lastFailoverTime = time.Now()
	r.mu.Unlock()

	r.metrics.FailoverCount.Add(1)
	log.Printf("[FAILOVER] Entering failover mode - all slaves unhealthy, routing all queries to master")
}

func (r *EnhancedDBRouter) exitFailoverMode() {
	r.mu.Lock()
	r.failoverMode = false
	r.mu.Unlock()
	log.Printf("[FAILOVER] Exiting failover mode - healthy slaves available")
}

func NewReplicationMonitor(router *EnhancedDBRouter, interval, maxLag time.Duration) *ReplicationMonitor {
	return &ReplicationMonitor{
		dbRouter:        router,
		checkInterval:   interval,
		maxLagThreshold: maxLag,
		enabled:         true,
		stopCh:          make(chan struct{}),
	}
}

func (rm *ReplicationMonitor) Start() {
	rm.wg.Add(1)
	go rm.monitorLoop()
}

func (rm *ReplicationMonitor) Stop() {
	close(rm.stopCh)
	rm.wg.Wait()
}

func (rm *ReplicationMonitor) monitorLoop() {
	defer rm.wg.Done()

	ticker := time.NewTicker(rm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.checkReplicationLag()
		}
	}
}

func (rm *ReplicationMonitor) checkReplicationLag() {
	for i, slave := range rm.dbRouter.slaveDBs {
		lag := rm.getReplicationLag(slave)
		rm.dbRouter.slaveStatus[i].ReplicationLag = lag
		rm.dbRouter.slaveStatus[i].LastCheck = time.Now()

		if lag > rm.maxLagThreshold {
			log.Printf("[REPLICATION] Slave %d (%s) has high replication lag: %v", i, rm.dbRouter.slaveStatus[i].Host, lag)
			rm.dbRouter.metrics.ReplicationLagAlerts.Add(1)
		}
	}

	rm.updateAvgLag()
}

func (rm *ReplicationMonitor) getReplicationLag(db *gorm.DB) time.Duration {
	var lagSeconds int64
	err := db.Raw(`
		SELECT COALESCE(EXTRACT(EPOCH FROM NOW() - pg_last_xact_replay_timestamp()), 0)
	`).Scan(&lagSeconds).Error

	if err != nil {
		return time.Duration(lagSeconds) * time.Second
	}

	return time.Duration(lagSeconds) * time.Second
}

func (rm *ReplicationMonitor) updateAvgLag() {
	totalLag := int64(0)
	healthyCount := 0

	for _, status := range rm.dbRouter.slaveStatus {
		if status.Healthy {
			totalLag += status.ReplicationLag.Nanoseconds()
			healthyCount++
		}
	}

	if healthyCount > 0 {
		rm.avgLag.Store(totalLag / int64(healthyCount))
	}
}

func (rm *ReplicationMonitor) GetReplicationStatus() []ReplicationStatus {
	status := make([]ReplicationStatus, 0, len(rm.dbRouter.slaveStatus))

	for i, s := range rm.dbRouter.slaveStatus {
		status = append(status, ReplicationStatus{
			SlaveIndex:     i,
			Host:           s.Host,
			ReplicationLag: s.ReplicationLag,
			IsHealthy:      s.Healthy,
			LastCheck:      s.LastCheck,
		})
	}

	return status
}

func connectDBEnhanced(host, port, user, password, dbname, sslmode string, cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.Database.ConnectionPool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.ConnectionPool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnectionPool.ConnMaxLifetimeSecs) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Database.ConnectionPool.ConnMaxIdleTimeSecs) * time.Second)

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func GetEnhancedRouter() *EnhancedDBRouter {
	return enhancedRouter
}

func (r *EnhancedDBRouter) Master() *gorm.DB {
	if !r.enabled {
		return DB
	}
	return r.masterDB
}

func (r *EnhancedDBRouter) Slave() *gorm.DB {
	if !r.enabled || len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	if r.failoverMode {
		return r.masterDB
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	switch r.loadBalanceMode {
	case "round_robin":
		return r.getSlaveRoundRobin()
	case "weighted_round_robin":
		return r.getSlaveWeightedRoundRobin()
	case "random":
		return r.getSlaveRandom()
	case "least_latency":
		return r.getSlaveLeastLatency()
	case "least_connections":
		return r.getSlaveLeastConnections()
	default:
		return r.getSlaveRoundRobin()
	}
}

func (r *EnhancedDBRouter) getSlaveRoundRobin() *gorm.DB {
	if len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	index := atomic.AddUint32(&r.currentSlave, 1) % uint32(len(r.slaveDBs))
	return r.slaveDBs[index]
}

func (r *EnhancedDBRouter) getSlaveWeightedRoundRobin() *gorm.DB {
	if len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	totalWeight := 0
	for _, w := range r.slaveWeights {
		totalWeight += w
	}

	if totalWeight <= 0 {
		return r.getSlaveRoundRobin()
	}

	randomVal := rand.Intn(totalWeight)
	currentWeight := 0

	for i, w := range r.slaveWeights {
		currentWeight += w
		if randomVal < currentWeight {
			return r.slaveDBs[i]
		}
	}

	return r.slaveDBs[0]
}

func (r *EnhancedDBRouter) getSlaveRandom() *gorm.DB {
	if len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	index := rand.Intn(len(r.slaveDBs))
	return r.slaveDBs[index]
}

func (r *EnhancedDBRouter) getSlaveLeastLatency() *gorm.DB {
	if len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	var bestSlave *gorm.DB
	var minLatency time.Duration = time.Hour

	for i, status := range r.slaveStatus {
		if status.Healthy && status.Latency < minLatency {
			minLatency = status.Latency
			bestSlave = r.slaveDBs[i]
		}
	}

	if bestSlave != nil {
		return bestSlave
	}

	return r.masterDB
}

func (r *EnhancedDBRouter) getSlaveLeastConnections() *gorm.DB {
	if len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	var bestSlave *gorm.DB
	var minActive int32 = 1 << 30

	for i, status := range r.slaveStatus {
		if status.Healthy && status.ActiveQueries < minActive {
			minActive = status.ActiveQueries
			bestSlave = r.slaveDBs[i]
		}
	}

	if bestSlave != nil {
		return bestSlave
	}

	return r.masterDB
}

func (r *EnhancedDBRouter) Read(ctx context.Context) *gorm.DB {
	return r.Slave().WithContext(ctx)
}

func (r *EnhancedDBRouter) Write(ctx context.Context) *gorm.DB {
	return r.Master().WithContext(ctx)
}

func (r *EnhancedDBRouter) GetSlaveHealthStatus() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make([]map[string]interface{}, 0, len(r.slaveDBs))
	for i, s := range r.slaveStatus {
		status = append(status, map[string]interface{}{
			"index":            i,
			"host":             s.Host,
			"port":             s.Port,
			"health":           s.Healthy,
			"latency_ms":       s.Latency.Milliseconds(),
			"replication_lag":  s.ReplicationLag.String(),
			"fail_count":       s.FailCount,
			"weight":           s.Weight,
			"active_queries":   atomic.LoadInt32(&s.ActiveQueries),
			"last_check":       s.LastCheck,
		})
	}
	return status
}

func (r *EnhancedDBRouter) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"master_queries":       r.metrics.MasterQueries.Load(),
		"slave_queries":        r.metrics.SlaveQueries.Load(),
		"failed_queries":       r.metrics.FailedQueries.Load(),
		"slave_switches":       r.metrics.SlaveSwitches.Load(),
		"avg_latency_ms":       r.metrics.AvgLatency.Load() / 1000000,
		"failover_count":       r.metrics.FailoverCount.Load(),
		"replication_lag_alerts": r.metrics.ReplicationLagAlerts.Load(),
		"failover_mode":        r.failoverMode,
		"last_failover_time":   r.lastFailoverTime,
	}
}

func (r *EnhancedDBRouter) RecordQuery(isMaster bool, latency time.Duration) {
	if isMaster {
		r.metrics.MasterQueries.Add(1)
	} else {
		r.metrics.SlaveQueries.Add(1)
	}

	total := r.metrics.MasterQueries.Load() + r.metrics.SlaveQueries.Load()
	if total > 0 {
		avgLatency := (r.metrics.AvgLatency.Load()*(total-1) + latency.Nanoseconds()) / total
		r.metrics.AvgLatency.Store(avgLatency)
	}
}

func (r *EnhancedDBRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.masterDB != nil {
		if sqlDB, err := r.masterDB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	for _, slave := range r.slaveDBs {
		if sqlDB, err := slave.DB(); err == nil {
			sqlDB.Close()
		}
	}

	if r.healthChecker != nil {
		r.healthChecker.Stop()
	}

	if r.replicationMonitor != nil {
		r.replicationMonitor.Stop()
	}

	return nil
}

func (r *EnhancedDBRouter) IsEnabled() bool {
	return r.enabled
}

func (r *EnhancedDBRouter) GetReplicationStatus() []ReplicationStatus {
	if r.replicationMonitor == nil {
		return nil
	}
	return r.replicationMonitor.GetReplicationStatus()
}