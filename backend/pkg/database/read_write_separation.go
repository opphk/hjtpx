package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBRouter struct {
	masterDB        *gorm.DB
	slaveDBs        []*gorm.DB
	slaveWeights    []int
	currentSlave    uint32
	enabled         bool
	loadBalanceMode string
	mu              sync.RWMutex
	healthChecker   *SlaveHealthChecker
	metrics         *RouterMetrics
}

type RouterMetrics struct {
	MasterQueries  atomic.Int64
	SlaveQueries   atomic.Int64
	FailedQueries  atomic.Int64
	SlaveSwitches  atomic.Int64
	LastSwitchTime atomic.Value
	AvgLatency     atomic.Int64
}

type SlaveHealthChecker struct {
	dbRouter      *DBRouter
	interval      time.Duration
	enabled       bool
	slaveStatus   map[int]*SlaveStatus
	mu            sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
	failoverEnabled bool
	maxFailCount    int
}

type SlaveStatus struct {
	Index     int
	Host      string
	Port      string
	Healthy   bool
	Latency   time.Duration
	LastCheck time.Time
	FailCount int
}

var router *DBRouter

func InitReadWriteSeparation(cfg *config.Config) error {
	if !cfg.Database.ReadWriteSeparation.Enabled {
		router = &DBRouter{
			enabled:         false,
			loadBalanceMode: cfg.Database.ReadWriteSeparation.LoadBalanceStrategy,
		}
		return nil
	}

	var err error

	masterDB, err := connectDB(
		cfg.Database.ReadWriteSeparation.Master.Host,
		cfg.Database.ReadWriteSeparation.Master.Port,
		cfg.Database.ReadWriteSeparation.Master.User,
		cfg.Database.ReadWriteSeparation.Master.Password,
		cfg.Database.ReadWriteSeparation.Master.DBName,
		cfg.Database.ReadWriteSeparation.Master.SSLMode,
	)
	if err != nil {
		return fmt.Errorf("failed to connect master database: %w", err)
	}

	slaveDBs := make([]*gorm.DB, 0)
	slaveWeights := make([]int, 0)

	for _, slaveCfg := range cfg.Database.ReadWriteSeparation.Slaves {
		slaveDB, err := connectDB(
			slaveCfg.Host,
			slaveCfg.Port,
			slaveCfg.User,
			slaveCfg.Password,
			slaveCfg.DBName,
			slaveCfg.SSLMode,
		)
		if err != nil {
			log.Printf("Warning: failed to connect slave database %s: %v", slaveCfg.Host, err)
			continue
		}
		slaveDBs = append(slaveDBs, slaveDB)
		slaveWeights = append(slaveWeights, slaveCfg.Weight)
	}

	router = &DBRouter{
		masterDB:        masterDB,
		slaveDBs:        slaveDBs,
		slaveWeights:    slaveWeights,
		enabled:         true,
		loadBalanceMode: cfg.Database.ReadWriteSeparation.LoadBalanceStrategy,
		currentSlave:    0,
		metrics:        &RouterMetrics{},
	}

	if cfg.Database.ReadWriteSeparation.AutoFailover {
		router.healthChecker = NewSlaveHealthChecker(router, 30*time.Second)
		router.healthChecker.Start()
	}

	log.Println("Read-write separation initialized successfully")
	return nil
}

func NewSlaveHealthChecker(router *DBRouter, interval time.Duration) *SlaveHealthChecker {
	return &SlaveHealthChecker{
		dbRouter:        router,
		interval:       interval,
		enabled:        true,
		slaveStatus:    make(map[int]*SlaveStatus),
		stopCh:        make(chan struct{}),
		failoverEnabled: true,
		maxFailCount:    3,
	}
}

func (shc *SlaveHealthChecker) Start() {
	shc.wg.Add(1)
	go shc.checkLoop()
}

func (shc *SlaveHealthChecker) Stop() {
	close(shc.stopCh)
	shc.wg.Wait()
}

func (shc *SlaveHealthChecker) SetFailoverEnabled(enabled bool) {
	shc.mu.Lock()
	defer shc.mu.Unlock()
	shc.failoverEnabled = enabled
}

func (shc *SlaveHealthChecker) SetMaxFailCount(count int) {
	shc.mu.Lock()
	defer shc.mu.Unlock()
	shc.maxFailCount = count
}

func (shc *SlaveHealthChecker) checkLoop() {
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

func (shc *SlaveHealthChecker) evaluateFailover() {
	shc.mu.RLock()
	defer shc.mu.RUnlock()

	for i, status := range shc.slaveStatus {
		if status.FailCount >= shc.maxFailCount && status.Healthy {
			log.Printf("[HEALTH_CHECKER] 从库 %d 失败次数超过阈值(%d)，标记为不健康", i, shc.maxFailCount)
			shc.mu.RUnlock()
			shc.markUnhealthy(i)
			shc.mu.RLock()
		}
	}
}

func (shc *SlaveHealthChecker) markUnhealthy(index int) {
	shc.mu.Lock()
	defer shc.mu.Unlock()

	if status, exists := shc.slaveStatus[index]; exists {
		status.Healthy = false
		shc.slaveStatus[index] = status
	}

	log.Printf("[HEALTH_CHECKER] 从库 %d 已标记为不健康", index)
}

func (shc *SlaveHealthChecker) markHealthy(index int) {
	shc.mu.Lock()
	defer shc.mu.Unlock()

	if status, exists := shc.slaveStatus[index]; exists {
		status.Healthy = true
		status.FailCount = 0
		shc.slaveStatus[index] = status
	}

	log.Printf("[HEALTH_CHECKER] 从库 %d 已恢复健康", index)
}

func (shc *SlaveHealthChecker) checkAllSlaves() {
	shc.mu.Lock()
	defer shc.mu.Unlock()

	for i, slave := range shc.dbRouter.slaveDBs {
		status := &SlaveStatus{
			Index:     i,
			Healthy:   true,
			LastCheck: time.Now(),
		}

		sqlDB, err := slave.DB()
		if err != nil {
			status.Healthy = false
			status.FailCount++
		} else {
			start := time.Now()
			if err := sqlDB.Ping(); err != nil {
				status.Healthy = false
				status.FailCount++
			} else {
				status.Latency = time.Since(start)
			}
		}

		shc.slaveStatus[i] = status
	}
}

func (shc *SlaveHealthChecker) GetHealthySlaves() []int {
	shc.mu.RLock()
	defer shc.mu.RUnlock()

	var healthy []int
	for i, status := range shc.slaveStatus {
		if status.Healthy {
			healthy = append(healthy, i)
		}
	}
	return healthy
}

func (shc *SlaveHealthChecker) GetSlaveStatus() []*SlaveStatus {
	shc.mu.RLock()
	defer shc.mu.RUnlock()

	status := make([]*SlaveStatus, 0, len(shc.slaveStatus))
	for _, s := range shc.slaveStatus {
		status = append(status, s)
	}
	return status
}

func (r *DBRouter) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"master_queries":  r.metrics.MasterQueries.Load(),
		"slave_queries":   r.metrics.SlaveQueries.Load(),
		"failed_queries":  r.metrics.FailedQueries.Load(),
		"slave_switches":  r.metrics.SlaveSwitches.Load(),
		"avg_latency_ms":  r.metrics.AvgLatency.Load() / 1000000,
	}
}

func (r *DBRouter) RecordQuery(isMaster bool, latency time.Duration) {
	if isMaster {
		r.metrics.MasterQueries.Add(1)
	} else {
		r.metrics.SlaveQueries.Add(1)
	}

	total := r.metrics.MasterQueries.Load() + r.metrics.SlaveQueries.Load()
	avgLatency := (r.metrics.AvgLatency.Load()*(total-1) + latency.Nanoseconds()) / total
	r.metrics.AvgLatency.Store(avgLatency)
}

func (r *DBRouter) RecordFailure() {
	r.metrics.FailedQueries.Add(1)
}

func (r *DBRouter) RecordSlaveSwitch() {
	r.metrics.SlaveSwitches.Add(1)
	r.metrics.LastSwitchTime.Store(time.Now())
}

func (r *DBRouter) GetOptimalSlave() *gorm.DB {
	if !r.enabled || len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	if r.healthChecker != nil {
		healthySlaves := r.healthChecker.GetHealthySlaves()
		if len(healthySlaves) == 0 {
			return r.masterDB
		}

		var bestSlave *gorm.DB
		var minLatency time.Duration = time.Hour

		for _, idx := range healthySlaves {
			status := r.healthChecker.slaveStatus[idx]
			if status.Latency < minLatency {
				minLatency = status.Latency
				bestSlave = r.slaveDBs[idx]
			}
		}

		if bestSlave != nil {
			return bestSlave
		}
	}

	return r.Slave()
}

func connectDB(host, port, user, password, dbname, sslmode string) (*gorm.DB, error) {
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

	cfg := config.GetConfig()
	sqlDB.SetMaxOpenConns(cfg.Database.ConnectionPool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.ConnectionPool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnectionPool.ConnMaxLifetimeSecs) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Database.ConnectionPool.ConnMaxIdleTimeSecs) * time.Second)

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func GetRouter() *DBRouter {
	return router
}

func (r *DBRouter) Master() *gorm.DB {
	if !r.enabled {
		return DB
	}
	return r.masterDB
}

func (r *DBRouter) Slave() *gorm.DB {
	if !r.enabled || len(r.slaveDBs) == 0 {
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
	default:
		return r.getSlaveRoundRobin()
	}
}

func (r *DBRouter) getSlaveRoundRobin() *gorm.DB {
	if len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	index := atomic.AddUint32(&r.currentSlave, 1) % uint32(len(r.slaveDBs))
	return r.slaveDBs[index]
}

func (r *DBRouter) getSlaveWeightedRoundRobin() *gorm.DB {
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

	randomVal := int(time.Now().UnixNano() % int64(totalWeight))
	currentWeight := 0

	for i, w := range r.slaveWeights {
		currentWeight += w
		if randomVal < currentWeight {
			return r.slaveDBs[i]
		}
	}

	return r.slaveDBs[0]
}

func (r *DBRouter) getSlaveRandom() *gorm.DB {
	if len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	index := int(time.Now().UnixNano() % int64(len(r.slaveDBs)))
	return r.slaveDBs[index]
}

func (r *DBRouter) Read(ctx context.Context) *gorm.DB {
	return r.Slave().WithContext(ctx)
}

func (r *DBRouter) Write(ctx context.Context) *gorm.DB {
	return r.Master().WithContext(ctx)
}

func (r *DBRouter) GetSlaveHealthStatus() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make([]map[string]interface{}, 0, len(r.slaveDBs))
	for i, slave := range r.slaveDBs {
		health := true
		sqlDB, err := slave.DB()
		if err != nil {
			health = false
		} else if err := sqlDB.Ping(); err != nil {
			health = false
		}

		status = append(status, map[string]interface{}{
			"index":  i,
			"health": health,
		})
	}
	return status
}

func (r *DBRouter) Close() error {
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

	return nil
}

func (r *DBRouter) IsEnabled() bool {
	return r.enabled
}

func (r *DBRouter) GetSlaveWithLeastConnections() *gorm.DB {
	if !r.enabled || len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	if r.healthChecker != nil {
		healthySlaves := r.healthChecker.GetHealthySlaves()
		if len(healthySlaves) == 0 {
			return r.masterDB
		}
	}

	minConns := -1
	var bestSlave *gorm.DB

	for i, slave := range r.slaveDBs {
		if r.healthChecker != nil {
			status := r.healthChecker.slaveStatus[i]
			if !status.Healthy {
				continue
			}
		}

		sqlDB, err := slave.DB()
		if err != nil {
			continue
		}

		stats := sqlDB.Stats()
		if minConns == -1 || stats.InUse < minConns {
			minConns = stats.InUse
			bestSlave = slave
		}
	}

	if bestSlave != nil {
		return bestSlave
	}

	return r.masterDB
}

func (r *DBRouter) GetSlaveByLatency() *gorm.DB {
	if !r.enabled || len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	if r.healthChecker == nil {
		return r.Slave()
	}

	healthySlaves := r.healthChecker.GetHealthySlaves()
	if len(healthySlaves) == 0 {
		return r.masterDB
	}

	var bestSlave *gorm.DB
	var minLatency time.Duration = time.Hour

	for _, idx := range healthySlaves {
		status := r.healthChecker.slaveStatus[idx]
		if status.Latency < minLatency {
			minLatency = status.Latency
			bestSlave = r.slaveDBs[idx]
		}
	}

	if bestSlave != nil {
		return bestSlave
	}

	return r.masterDB
}

func (r *DBRouter) ReadWithFallback(ctx context.Context, retries int) (*gorm.DB, error) {
	for i := 0; i < retries; i++ {
		slave := r.GetOptimalSlave()
		if slave == r.masterDB {
			return slave, nil
		}

		sqlDB, err := slave.DB()
		if err != nil {
			if i == retries-1 {
				return r.masterDB, fmt.Errorf("failed to get slave connection after %d retries", retries)
			}
			r.RecordFailure()
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}

		if err := sqlDB.Ping(); err != nil {
			if r.healthChecker != nil {
				for idx, s := range r.slaveDBs {
					if s == slave {
						r.healthChecker.markUnhealthy(idx)
						break
					}
				}
			}
			r.RecordFailure()
			r.RecordSlaveSwitch()

			if i == retries-1 {
				return r.masterDB, nil
			}
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}

		return slave.WithContext(ctx), nil
	}

	return r.masterDB.WithContext(ctx), nil
}

func (r *DBRouter) GetConnectionPoolStats() []map[string]interface{} {
	stats := make([]map[string]interface{}, 0, len(r.slaveDBs)+1)

	if r.masterDB != nil {
		sqlDB, err := r.masterDB.DB()
		if err == nil {
			dbStats := sqlDB.Stats()
			stats = append(stats, map[string]interface{}{
				"role":              "master",
				"max_open":          dbStats.MaxOpenConnections,
				"open":              dbStats.OpenConnections,
				"in_use":            dbStats.InUse,
				"idle":              dbStats.Idle,
				"wait_count":        dbStats.WaitCount,
				"wait_duration_ms":  dbStats.WaitDuration.Milliseconds(),
				"max_idle_closed":   dbStats.MaxIdleClosed,
				"max_lifetime_closed": dbStats.MaxLifetimeClosed,
			})
		}
	}

	for i, slave := range r.slaveDBs {
		healthy := true
		if r.healthChecker != nil {
			status := r.healthChecker.slaveStatus[i]
			healthy = status.Healthy
		}

		sqlDB, err := slave.DB()
		if err != nil {
			stats = append(stats, map[string]interface{}{
				"role":    "slave",
				"index":   i,
				"healthy": false,
				"error":   err.Error(),
			})
			continue
		}

		dbStats := sqlDB.Stats()
		latency := time.Duration(0)
		if r.healthChecker != nil {
			status := r.healthChecker.slaveStatus[i]
			latency = status.Latency
		}

		stats = append(stats, map[string]interface{}{
			"role":              "slave",
			"index":             i,
			"healthy":           healthy,
			"max_open":          dbStats.MaxOpenConnections,
			"open":              dbStats.OpenConnections,
			"in_use":            dbStats.InUse,
			"idle":              dbStats.Idle,
			"wait_count":        dbStats.WaitCount,
			"wait_duration_ms":  dbStats.WaitDuration.Milliseconds(),
			"max_idle_closed":   dbStats.MaxIdleClosed,
			"max_lifetime_closed": dbStats.MaxLifetimeClosed,
			"latency_ms":        latency.Milliseconds(),
		})
	}

	return stats
}

func (r *DBRouter) GetDetailedMetrics() map[string]interface{} {
	result := r.GetMetrics()

	result["slave_count"] = len(r.slaveDBs)
	result["strategy"] = r.loadBalanceMode
	result["enabled"] = r.enabled

	if r.healthChecker != nil {
		healthyCount := len(r.healthChecker.GetHealthySlaves())
		result["healthy_slaves"] = healthyCount
		result["unhealthy_slaves"] = len(r.slaveDBs) - healthyCount
	}

	result["connection_pool_stats"] = r.GetConnectionPoolStats()

	return result
}

func (r *DBRouter) FailoverToMaster() {
	log.Println("[DB_ROUTER] Manual failover to master initiated")
	r.mu.Lock()
	r.enabled = false
	r.mu.Unlock()
}

func (r *DBRouter) RestoreSlaves() {
	log.Println("[DB_ROUTER] Restoring slave connections")
	r.mu.Lock()
	r.enabled = true
	r.mu.Unlock()

	if r.healthChecker != nil {
		for i := range r.slaveDBs {
			r.healthChecker.markHealthy(i)
		}
	}
}

func (r *DBRouter) GetSlaveStatusDetailed() []*SlaveStatus {
	if r.healthChecker == nil {
		return nil
	}
	return r.healthChecker.GetSlaveStatus()
}

func (r *DBRouter) SetLoadBalanceMode(mode string) {
	r.mu.Lock()
	r.loadBalanceMode = mode
	r.mu.Unlock()
	log.Printf("[DB_ROUTER] Load balance mode changed to: %s", mode)
}
