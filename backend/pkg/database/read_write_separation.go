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
	masterDB         *gorm.DB
	slaveDBs         []*gorm.DB
	slaveWeights     []int
	currentSlave     uint32
	enabled          bool
	loadBalanceMode  string
	mu               sync.RWMutex
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
	}

	log.Println("Read-write separation initialized successfully")
	return nil
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
