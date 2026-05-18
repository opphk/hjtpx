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
		dbRouter:    router,
		interval:   interval,
		enabled:    true,
		slaveStatus: make(map[int]*SlaveStatus),
		stopCh:     make(chan struct{}),
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
		}
	}
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
