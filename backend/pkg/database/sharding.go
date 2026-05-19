package database

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type ShardingConfig struct {
	Enabled           bool
	ShardingKey       string
	ShardingAlgorithm string
	ShardCount        int
	VirtualNodes      int
	ShardPrefix       string
}

type ShardingStrategy struct {
	config   ShardingConfig
	shards   map[int]*gorm.DB
	mu       sync.RWMutex
	router   *ShardRouter
}

type ShardRouter struct {
	shards     []*gorm.DB
	vNodes     int
	totalSlots int
	vNodeMap   map[int]int
	mu         sync.RWMutex
}

type QueryRouter struct {
	sharding    *ShardingStrategy
	readWrite   *DBRouter
}

func NewShardingStrategy(cfg ShardingConfig) (*ShardingStrategy, error) {
	if !cfg.Enabled {
		return &ShardingStrategy{
			config: cfg,
			shards: make(map[int]*gorm.DB),
		}, nil
	}

	if cfg.ShardCount <= 0 {
		cfg.ShardCount = 4
	}

	if cfg.VirtualNodes <= 0 {
		cfg.VirtualNodes = 100
	}

	ss := &ShardingStrategy{
		config:   cfg,
		shards:   make(map[int]*gorm.DB),
		router:   newShardRouter(cfg.ShardCount, cfg.VirtualNodes),
	}

	return ss, nil
}

func newShardRouter(shardCount, vNodes int) *ShardRouter {
	sr := &ShardRouter{
		shards:     make([]*gorm.DB, shardCount),
		vNodes:     vNodes,
		totalSlots: shardCount * vNodes,
		vNodeMap:   make(map[int]int),
	}

	for i := 0; i < shardCount; i++ {
		for j := 0; j < vNodes; j++ {
			slot := hashSlot(i, j, shardCount)
			sr.vNodeMap[slot] = i
		}
	}

	return sr
}

func hashSlot(i, j, shardCount int) int {
	h := fnv.New32a()
	fmt.Fprintf(h, "%d:%d", i, j)
	return int(h.Sum32()) % (shardCount * 100)
}

func (sr *ShardRouter) RegisterShard(shardIndex int, db *gorm.DB) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if shardIndex >= 0 && shardIndex < len(sr.shards) {
		sr.shards[shardIndex] = db
	}
}

func (sr *ShardRouter) GetShard(shardingKey string) (*gorm.DB, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	if len(sr.shards) == 0 {
		return nil, fmt.Errorf("no shards available")
	}

	slot := hashShardingKey(shardingKey, sr.totalSlots)

	shardIndex := 0
	if idx, ok := sr.vNodeMap[slot]; ok {
		shardIndex = idx
	}

	if shardIndex >= len(sr.shards) || sr.shards[shardIndex] == nil {
		return nil, fmt.Errorf("shard %d not available", shardIndex)
	}

	return sr.shards[shardIndex], nil
}

func hashShardingKey(key string, slots int) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % slots
}

func (ss *ShardingStrategy) RegisterShard(shardIndex int, db *gorm.DB) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if shardIndex < 0 || shardIndex >= ss.config.ShardCount {
		return fmt.Errorf("invalid shard index: %d", shardIndex)
	}

	ss.shards[shardIndex] = db
	ss.router.RegisterShard(shardIndex, db)

	return nil
}

func (ss *ShardingStrategy) GetShard(shardingKey string) (*gorm.DB, error) {
	if !ss.config.Enabled {
		return nil, fmt.Errorf("sharding is disabled")
	}

	return ss.router.GetShard(shardingKey)
}

func (ss *ShardingStrategy) GetAllShards() []*gorm.DB {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	shards := make([]*gorm.DB, 0, len(ss.shards))
	for _, shard := range ss.shards {
		if shard != nil {
			shards = append(shards, shard)
		}
	}
	return shards
}

func (ss *ShardingStrategy) GetShardCount() int {
	return ss.config.ShardCount
}

func (ss *ShardingStrategy) IsEnabled() bool {
	return ss.config.Enabled
}

func (ss *ShardingStrategy) Close() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	var lastErr error
	for i, shard := range ss.shards {
		if shard != nil {
			if sqlDB, err := shard.DB(); err == nil {
				if err := sqlDB.Close(); err != nil {
					lastErr = err
					log.Printf("Error closing shard %d: %v", i, err)
				}
			}
		}
	}
	return lastErr
}

func NewQueryRouter(sharding *ShardingStrategy, readWrite *DBRouter) *QueryRouter {
	return &QueryRouter{
		sharding:  sharding,
		readWrite: readWrite,
	}
}

func (qr *QueryRouter) Read(ctx context.Context, query *gorm.DB, shardingKey string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if qr.sharding != nil && qr.sharding.IsEnabled() && shardingKey != "" {
		db, err = qr.sharding.GetShard(shardingKey)
		if err != nil {
			return nil, err
		}
	} else if qr.readWrite != nil && qr.readWrite.IsEnabled() {
		db = qr.readWrite.GetOptimalSlave()
	} else {
		return query.WithContext(ctx), nil
	}

	return db.WithContext(ctx), nil
}

func (qr *QueryRouter) Write(ctx context.Context, query *gorm.DB, shardingKey string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if qr.sharding != nil && qr.sharding.IsEnabled() && shardingKey != "" {
		db, err = qr.sharding.GetShard(shardingKey)
		if err != nil {
			return nil, err
		}
	} else if qr.readWrite != nil && qr.readWrite.IsEnabled() {
		db = qr.readWrite.Master()
	} else {
		return query.WithContext(ctx), nil
	}

	return db.WithContext(ctx), nil
}

func (qr *QueryRouter) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"sharding_enabled": qr.sharding != nil && qr.sharding.IsEnabled(),
		"read_write_enabled": qr.readWrite != nil && qr.readWrite.IsEnabled(),
	}
}

func (qr *QueryRouter) RecordCacheHit() {
}

func (qr *QueryRouter) RecordCacheMiss() {
}

func (qr *QueryRouter) RecordSlowQuery() {
}

type IndexOptimization struct {
	db          *gorm.DB
	autoAnalyze bool
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func NewIndexOptimization(db *gorm.DB, cfg *config.IndexOptimizationConfig) *IndexOptimization {
	return &IndexOptimization{
		db:          db,
		autoAnalyze: cfg.AutoAnalyzeEnabled,
		interval:    time.Duration(cfg.AutoAnalyzeIntervalHours) * time.Hour,
		stopCh:      make(chan struct{}),
	}
}

func (io *IndexOptimization) Start() {
	if !io.autoAnalyze || io.interval <= 0 {
		return
	}

	io.wg.Add(1)
	go func() {
		defer io.wg.Done()
		ticker := time.NewTicker(io.interval)
		defer ticker.Stop()

		for {
			select {
			case <-io.stopCh:
				return
			case <-ticker.C:
				io.analyze()
			}
		}
	}()
}

func (io *IndexOptimization) Stop() {
	close(io.stopCh)
	io.wg.Wait()
}

func (io *IndexOptimization) analyze() {
	if io.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sqlDB, err := io.db.DB()
	if err != nil {
		log.Printf("IndexOptimization: failed to get db instance: %v", err)
		return
	}

	if _, err := sqlDB.ExecContext(ctx, "ANALYZE"); err != nil {
		log.Printf("IndexOptimization: failed to analyze: %v", err)
	}
}

func (io *IndexOptimization) AnalyzeTable(tableName string) error {
	if io.db == nil {
		return fmt.Errorf("database not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	return io.db.WithContext(ctx).Exec(fmt.Sprintf("ANALYZE %s", tableName)).Error
}

func (io *IndexOptimization) AnalyzeIndex(indexName string) error {
	if io.db == nil {
		return fmt.Errorf("database not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	return io.db.WithContext(ctx).Exec(fmt.Sprintf("ANALYZE INDEX %s", indexName)).Error
}

type ShardingKeyGenerator struct {
	shardingKeyType string
}

func NewShardingKeyGenerator(keyType string) *ShardingKeyGenerator {
	return &ShardingKeyGenerator{
		shardingKeyType: keyType,
	}
}

func (g *ShardingKeyGenerator) GenerateFromUserID(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func (g *ShardingKeyGenerator) GenerateFromTenantID(tenantID string) string {
	return fmt.Sprintf("tenant:%s", tenantID)
}

func (g *ShardingKeyGenerator) GenerateFromTimeRange(start, end time.Time) string {
	return fmt.Sprintf("time:%s:%s", start.Format("20060102"), end.Format("20060102"))
}

func (g *ShardingKeyGenerator) GenerateComposite(keys ...string) string {
	return strings.Join(keys, ":")
}

type DatabaseHealthCheck struct {
	db       *gorm.DB
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
	healthy  bool
	mu       sync.RWMutex
}

func NewDatabaseHealthCheck(db *gorm.DB, interval time.Duration) *DatabaseHealthCheck {
	return &DatabaseHealthCheck{
		db:       db,
		interval: interval,
		stopCh:   make(chan struct{}),
		healthy:  true,
	}
}

func (hc *DatabaseHealthCheck) Start() {
	hc.wg.Add(1)
	go func() {
		defer hc.wg.Done()
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-hc.stopCh:
				return
			case <-ticker.C:
				hc.check()
			}
		}
	}()
}

func (hc *DatabaseHealthCheck) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *DatabaseHealthCheck) check() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB, err := hc.db.DB()
	if err != nil {
		hc.setHealthy(false)
		return
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		hc.setHealthy(false)
		return
	}

	hc.setHealthy(true)
}

func (hc *DatabaseHealthCheck) setHealthy(healthy bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.healthy = healthy
}

func (hc *DatabaseHealthCheck) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.healthy
}

func (hc *DatabaseHealthCheck) GetStatus() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return map[string]interface{}{
		"healthy": hc.healthy,
	}
}

func getEnv(key string, defaultValue string) string {
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	return defaultValue
}
