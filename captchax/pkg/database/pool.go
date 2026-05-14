package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"captchax/internal/log"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type RedisPoolConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type PoolStats struct {
	PostgresMaxOpenConns     int
	PostgresMaxIdleConns     int
	PostgresOpenConns        int
	PostgresIdleConns        int
	PostgresWaitCount        int64
	PostgresWaitDuration     time.Duration
	PostgresMaxIdleClosed    int64
	PostgresMaxLifetimeClosed int64
	RedisPoolSize            int
	RedisMinIdleConns        int
	RedisTotalConns          int
	RedisIdleConns           int
	RedisStaleConns          int
	RedisHits                int64
	RedisMisses              int64
	RedisTimeouts            int64
}

type HealthChecker struct {
	mu         sync.RWMutex
	postgresOK bool
	redisOK    bool
	lastCheck  time.Time
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		postgresOK: false,
		redisOK:    false,
		lastCheck:  time.Now(),
	}
}

func (h *HealthChecker) SetPostgres(ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.postgresOK = ok
	h.lastCheck = time.Now()
}

func (h *HealthChecker) SetRedis(ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.redisOK = ok
	h.lastCheck = time.Now()
}

func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.postgresOK && h.redisOK
}

func (h *HealthChecker) Status() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]interface{}{
		"postgres": h.postgresOK,
		"redis":    h.redisOK,
		"last_check": h.lastCheck,
	}
}

type MonitoredPostgres struct {
	*gorm.DB
	health   *HealthChecker
	poolStats PoolStats
	mu        sync.RWMutex
}

func NewMonitoredPostgres(dsn string, cfg *PostgresPoolConfig) (*MonitoredPostgres, error) {
	if cfg == nil {
		cfg = &PostgresPoolConfig{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 3 * time.Minute,
		}
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		PrepareStmt: true,
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	mp := &MonitoredPostgres{
		DB:     db,
		health: NewHealthChecker(),
	}
	mp.health.SetPostgres(true)

	go mp.monitorPostgres()

	return mp, nil
}

func (mp *MonitoredPostgres) monitorPostgres() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sqlDB, err := mp.DB.DB()
		if err != nil {
			mp.health.SetPostgres(false)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = sqlDB.PingContext(ctx)
		cancel()

		if err != nil {
			mp.health.SetPostgres(false)
			log.Default().Error("postgres health check failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			mp.health.SetPostgres(true)
		}
	}
}

func (mp *MonitoredPostgres) GetPoolStats() PoolStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	sqlDB, err := mp.DB.DB()
	stats := PoolStats{}
	if err != nil {
		return stats
	}

	postgresStats := sqlDB.Stats()
	stats.PostgresMaxOpenConns = postgresStats.MaxOpenConnections
	stats.PostgresMaxIdleConns = 0
	stats.PostgresOpenConns = postgresStats.OpenConnections
	stats.PostgresIdleConns = postgresStats.Idle
	stats.PostgresWaitCount = postgresStats.WaitCount
	stats.PostgresWaitDuration = postgresStats.WaitDuration
	stats.PostgresMaxIdleClosed = postgresStats.MaxIdleClosed
	stats.PostgresMaxLifetimeClosed = postgresStats.MaxLifetimeClosed

	return stats
}

func (mp *MonitoredPostgres) IsHealthy() bool {
	return mp.health.IsHealthy()
}

func (mp *MonitoredPostgres) HealthStatus() map[string]interface{} {
	return mp.health.Status()
}

func (mp *MonitoredPostgres) Close() error {
	sqlDB, err := mp.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

type MonitoredRedis struct {
	*redis.Client
	health    *HealthChecker
	poolStats PoolStats
	mu        sync.RWMutex
}

func NewMonitoredRedis(cfg *RedisPoolConfig) (*MonitoredRedis, error) {
	if cfg == nil {
		cfg = &RedisPoolConfig{
			PoolSize:    100,
			MinIdleConns: 10,
			PoolTimeout:  4 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		PoolTimeout:  cfg.PoolTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	mr := &MonitoredRedis{
		Client: client,
		health: NewHealthChecker(),
	}
	mr.health.SetRedis(true)

	go mr.monitorRedis()

	return mr, nil
}

func (mr *MonitoredRedis) monitorRedis() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := mr.Client.Ping(ctx).Err()
		cancel()

		if err != nil {
			mr.health.SetRedis(false)
			log.Default().Error("redis health check failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			mr.health.SetRedis(true)
		}
	}
}

func (mr *MonitoredRedis) GetPoolStats() PoolStats {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	stats := PoolStats{}

	redisStats := mr.Client.PoolStats()
	stats.RedisPoolSize = int(redisStats.TotalConns)
	stats.RedisMinIdleConns = 0
	stats.RedisTotalConns = int(redisStats.TotalConns)
	stats.RedisIdleConns = int(redisStats.IdleConns)
	stats.RedisStaleConns = int(redisStats.StaleConns)
	stats.RedisHits = int64(redisStats.Hits)
	stats.RedisMisses = int64(redisStats.Misses)
	stats.RedisTimeouts = int64(redisStats.Timeouts)

	return stats
}

func (mr *MonitoredRedis) IsHealthy() bool {
	return mr.health.IsHealthy()
}

func (mr *MonitoredRedis) HealthStatus() map[string]interface{} {
	return mr.health.Status()
}

func (mr *MonitoredRedis) Close() error {
	return mr.Client.Close()
}

type PoolManager struct {
	Postgres *MonitoredPostgres
	Redis    *MonitoredRedis
	mu       sync.RWMutex
}

func NewPoolManager() *PoolManager {
	return &PoolManager{}
}

func (pm *PoolManager) SetupPostgres(dsn string, cfg *PostgresPoolConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	db, err := NewMonitoredPostgres(dsn, cfg)
	if err != nil {
		return err
	}
	pm.Postgres = db
	return nil
}

func (pm *PoolManager) SetupRedis(cfg *RedisPoolConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	redis, err := NewMonitoredRedis(cfg)
	if err != nil {
		return err
	}
	pm.Redis = redis
	return nil
}

func (pm *PoolManager) GetAllStats() PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var stats PoolStats

	if pm.Postgres != nil {
		stats = pm.Postgres.GetPoolStats()
	}

	if pm.Redis != nil {
		redisStats := pm.Redis.GetPoolStats()
		stats.RedisPoolSize = redisStats.RedisPoolSize
		stats.RedisMinIdleConns = redisStats.RedisMinIdleConns
		stats.RedisTotalConns = redisStats.RedisTotalConns
		stats.RedisIdleConns = redisStats.RedisIdleConns
		stats.RedisStaleConns = redisStats.RedisStaleConns
		stats.RedisHits = redisStats.RedisHits
		stats.RedisMisses = redisStats.RedisMisses
		stats.RedisTimeouts = redisStats.RedisTimeouts
	}

	return stats
}

func (pm *PoolManager) HealthCheck() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.Postgres != nil && !pm.Postgres.IsHealthy() {
		return false
	}
	if pm.Redis != nil && !pm.Redis.IsHealthy() {
		return false
	}
	return true
}

func (pm *PoolManager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.Postgres != nil {
		pm.Postgres.Close()
	}
	if pm.Redis != nil {
		pm.Redis.Close()
	}
}

type ConnectionPool interface {
	Stats() PoolStats
	HealthCheck() bool
	Close() error
}
