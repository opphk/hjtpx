package database

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type PoolMonitor struct {
	mu            sync.RWMutex
	statsHistory  []PoolStatsRecord
	maxHistoryLen int
}

type PoolStatsRecord struct {
	Timestamp time.Time
	Stats     PoolStats
}

var poolMonitor *PoolMonitor

func InitConnectionPool(cfg *config.Config) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(cfg.Database.ConnectionPool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.ConnectionPool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnectionPool.ConnMaxLifetimeSecs) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Database.ConnectionPool.ConnMaxIdleTimeSecs) * time.Second)

	poolMonitor = &PoolMonitor{
		statsHistory:  make([]PoolStatsRecord, 0),
		maxHistoryLen: 1000,
	}

	if cfg.Database.Monitoring.EnableConnectionMetrics {
		go startPoolMonitoring(cfg)
	}

	log.Println("Connection pool configured successfully")
	return nil
}

func startPoolMonitoring(cfg *config.Config) {
	ticker := time.NewTicker(time.Duration(cfg.Database.Monitoring.MetricsIntervalSecs) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := GetPoolStats()
		if err != nil {
			log.Printf("Error getting pool stats: %v", err)
			continue
		}

		poolMonitor.mu.Lock()
		poolMonitor.statsHistory = append(poolMonitor.statsHistory, PoolStatsRecord{
			Timestamp: time.Now(),
			Stats:     *stats,
		})

		if len(poolMonitor.statsHistory) > poolMonitor.maxHistoryLen {
			poolMonitor.statsHistory = poolMonitor.statsHistory[1:]
		}
		poolMonitor.mu.Unlock()

		log.Printf("Pool stats: Open=%d, InUse=%d, Idle=%d, WaitCount=%d",
			stats.OpenConnections,
			stats.InUse,
			stats.Idle,
			stats.WaitCount,
		)
	}
}

func GetPoolStatsHistory() []PoolStatsRecord {
	if poolMonitor == nil {
		return nil
	}

	poolMonitor.mu.RLock()
	defer poolMonitor.mu.RUnlock()

	history := make([]PoolStatsRecord, len(poolMonitor.statsHistory))
	copy(history, poolMonitor.statsHistory)
	return history
}

func UpdateConnectionPool(maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	if maxOpen > 0 {
		sqlDB.SetMaxOpenConns(maxOpen)
	}
	if maxIdle > 0 {
		sqlDB.SetMaxIdleConns(maxIdle)
	}
	if maxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(maxLifetime)
	}
	if maxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(maxIdleTime)
	}

	log.Printf("Connection pool updated: MaxOpen=%d, MaxIdle=%d", maxOpen, maxIdle)
	return nil
}

func CheckConnectionHealth() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func GetRawDB() (*sql.DB, error) {
	return DB.DB()
}
