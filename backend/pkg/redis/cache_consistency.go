package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type ConsistencyLevel int

const (
	ConsistencyLevelEventual ConsistencyLevel = iota
	ConsistencyLevelStrong
	ConsistencyLevelLinearizable
)

type CacheUpdateMode int

const (
	CacheUpdateModeWriteThrough CacheUpdateMode = iota
	CacheUpdateModeWriteBehind
	CacheUpdateModeRefreshAhead
)

type InvalidationStrategy int

const (
	InvalidationStrategyTimeBased InvalidationStrategy = iota
	InvalidationStrategyVersionBased
	InvalidationStrategyPubSub
)

type DistributedCacheConsistency struct {
	config           *ConsistencyConfig
	mu               sync.RWMutex
	localVersionMap  *sync.Map
	pubSubClient     *goredis.PubSub
	pubSubCtx        context.Context
	pubSubCancel     context.CancelFunc
	invalidationQueue chan *InvalidationMessage
	metrics          *ConsistencyMetrics
}

type ConsistencyConfig struct {
	Level              ConsistencyLevel
	UpdateMode         CacheUpdateMode
	InvalidationStrat  InvalidationStrategy
	InvalidationTTL    time.Duration
	VersionCheckTTL    time.Duration
	WriteBehindBatch   int
	WriteBehindInterval time.Duration
	PubSubChannel      string
}

type InvalidationMessage struct {
	Key      string
	Version  int64
	Type     string
	Source   string
	Time     time.Time
}

type ConsistencyMetrics struct {
	TotalWrites       atomic.Int64
	TotalReads        atomic.Int64
	ConflictCount     atomic.Int64
	InvalidationCount atomic.Int64
	PublishSuccess    atomic.Int64
	PublishFailures   atomic.Int64
	LocalHits         atomic.Int64
	LocalMisses       atomic.Int64
	LastSyncTime      atomic.Value
}

type CacheConsistencyStatus struct {
	Level         string
	UpdateMode    string
	Metrics       map[string]interface{}
	Connected     bool
	LastSyncTime  time.Time
}

var DefaultConsistencyConfig = &ConsistencyConfig{
	Level:              ConsistencyLevelEventual,
	UpdateMode:         CacheUpdateModeWriteThrough,
	InvalidationStrat:  InvalidationStrategyPubSub,
	InvalidationTTL:    5 * time.Minute,
	VersionCheckTTL:    10 * time.Second,
	WriteBehindBatch:   100,
	WriteBehindInterval: 5 * time.Second,
	PubSubChannel:      "cache:invalidation",
}

func NewDistributedCacheConsistency(config *ConsistencyConfig) *DistributedCacheConsistency {
	if config == nil {
		config = DefaultConsistencyConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	dcc := &DistributedCacheConsistency{
		config:             config,
		localVersionMap:    &sync.Map{},
		pubSubCtx:          ctx,
		pubSubCancel:       cancel,
		invalidationQueue:  make(chan *InvalidationMessage, 1000),
		metrics:            &ConsistencyMetrics{},
	}

	if config.InvalidationStrat == InvalidationStrategyPubSub {
		go dcc.startPubSub()
	}

	go dcc.processInvalidationQueue()

	if config.UpdateMode == CacheUpdateModeWriteBehind {
		go dcc.startWriteBehindProcessor()
	}

	return dcc
}

func (dcc *DistributedCacheConsistency) startPubSub() {
	if Client == nil {
		return
	}

	dcc.pubSubClient = Client.Subscribe(dcc.pubSubCtx, dcc.config.PubSubChannel)

	go func() {
		ch := dcc.pubSubClient.Channel()
		for msg := range ch {
			var invMsg InvalidationMessage
			if err := json.Unmarshal([]byte(msg.Payload), &invMsg); err == nil {
				dcc.invalidationQueue <- &invMsg
			}
		}
	}()
}

func (dcc *DistributedCacheConsistency) processInvalidationQueue() {
	for msg := range dcc.invalidationQueue {
		dcc.processInvalidation(msg)
	}
}

func (dcc *DistributedCacheConsistency) processInvalidation(msg *InvalidationMessage) {
	dcc.mu.Lock()
	defer dcc.mu.Unlock()

	dcc.metrics.InvalidationCount.Add(1)

	switch msg.Type {
	case "invalidate":
		dcc.localVersionMap.Delete(msg.Key)
		if ec := GetEnhancedCache(); ec != nil {
			ec.Delete(context.Background(), msg.Key, &DeleteOptions{Level: CacheLevelL1})
		}
	case "update":
		if currentVersion, ok := dcc.localVersionMap.Load(msg.Key); !ok || currentVersion.(int64) < msg.Version {
			dcc.localVersionMap.Store(msg.Key, msg.Version)
			if ec := GetEnhancedCache(); ec != nil {
				ec.Delete(context.Background(), msg.Key, &DeleteOptions{Level: CacheLevelL1})
			}
		}
	}
}

func (dcc *DistributedCacheConsistency) startWriteBehindProcessor() {
	ticker := time.NewTicker(dcc.config.WriteBehindInterval)
	defer ticker.Stop()

	for range ticker.C {
		dcc.flushWriteBehind()
	}
}

func (dcc *DistributedCacheConsistency) flushWriteBehind() {
}

func (dcc *DistributedCacheConsistency) GetWithConsistency(ctx context.Context, key string) ([]byte, error) {
	dcc.metrics.TotalReads.Add(1)

	ec := GetEnhancedCache()
	if ec == nil {
		return nil, ErrCacheDisabled
	}

	switch dcc.config.Level {
	case ConsistencyLevelStrong:
		return dcc.getStrong(ctx, key)
	case ConsistencyLevelLinearizable:
		return dcc.getLinearizable(ctx, key)
	default:
		return dcc.getEventual(ctx, key)
	}
}

func (dcc *DistributedCacheConsistency) getEventual(ctx context.Context, key string) ([]byte, error) {
	ec := GetEnhancedCache()
	return ec.Get(ctx, key, nil)
}

func (dcc *DistributedCacheConsistency) getStrong(ctx context.Context, key string) ([]byte, error) {
	ec := GetEnhancedCache()

	localVersion, _ := dcc.localVersionMap.LoadOrStore(key, int64(0))
	remoteVersion, err := dcc.getRemoteVersion(ctx, key)
	if err != nil {
		return nil, err
	}

	if localVersion.(int64) == remoteVersion {
		val, err := ec.Get(ctx, key, &GetOptions{Level: CacheLevelL1})
		if err == nil {
			dcc.metrics.LocalHits.Add(1)
			return val, nil
		}
	}

	dcc.metrics.LocalMisses.Add(1)
	val, err := ec.Get(ctx, key, &GetOptions{Level: CacheLevelL2})
	if err == nil {
		dcc.localVersionMap.Store(key, remoteVersion)
	}
	return val, err
}

func (dcc *DistributedCacheConsistency) getLinearizable(ctx context.Context, key string) ([]byte, error) {
	if Client == nil {
		return nil, ErrCacheDisabled
	}

	lockKey := fmt.Sprintf("lock:linear:%s", key)
	lockTTL := 5 * time.Second

	acquired, err := Client.SetNX(ctx, lockKey, "1", lockTTL).Result()
	if err != nil {
		return nil, err
	}

	if !acquired {
		dcc.metrics.ConflictCount.Add(1)
		return nil, fmt.Errorf("conflict: key is locked")
	}
	defer Client.Del(ctx, lockKey)

	ec := GetEnhancedCache()
	val, err := ec.Get(ctx, key, &GetOptions{Level: CacheLevelL2})
	if err == nil {
		ec.Set(ctx, key, val, &SetOptions{Level: CacheLevelL1})
	}
	return val, err
}

func (dcc *DistributedCacheConsistency) SetWithConsistency(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	dcc.metrics.TotalWrites.Add(1)

	switch dcc.config.UpdateMode {
	case CacheUpdateModeWriteThrough:
		return dcc.setWriteThrough(ctx, key, value, ttl)
	case CacheUpdateModeWriteBehind:
		return dcc.setWriteBehind(ctx, key, value, ttl)
	case CacheUpdateModeRefreshAhead:
		return dcc.setRefreshAhead(ctx, key, value, ttl)
	default:
		return dcc.setWriteThrough(ctx, key, value, ttl)
	}
}

func (dcc *DistributedCacheConsistency) setWriteThrough(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ec := GetEnhancedCache()
	if ec == nil {
		return ErrCacheDisabled
	}

	vm := NewVersionManager()
	newVersion, _ := vm.Increment(key)

	opts := &SetOptions{
		Level:   CacheLevelBoth,
		TTL:     ttl,
		Version: newVersion,
	}

	if err := ec.Set(ctx, key, value, opts); err != nil {
		return err
	}

	dcc.localVersionMap.Store(key, newVersion)

	if dcc.config.InvalidationStrat == InvalidationStrategyPubSub {
		dcc.publishInvalidation(ctx, key, newVersion, "update")
	}

	return nil
}

func (dcc *DistributedCacheConsistency) setWriteBehind(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ec := GetEnhancedCache()
	if ec == nil {
		return ErrCacheDisabled
	}

	ec.Set(ctx, key, value, &SetOptions{Level: CacheLevelL1, TTL: ttl})

	return nil
}

func (dcc *DistributedCacheConsistency) setRefreshAhead(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return dcc.setWriteThrough(ctx, key, value, ttl)
}

func (dcc *DistributedCacheConsistency) DeleteWithConsistency(ctx context.Context, key string) error {
	ec := GetEnhancedCache()
	if ec == nil {
		return ErrCacheDisabled
	}

	if err := ec.Delete(ctx, key, &DeleteOptions{Level: CacheLevelBoth}); err != nil {
		return err
	}

	dcc.localVersionMap.Delete(key)

	if dcc.config.InvalidationStrat == InvalidationStrategyPubSub {
		dcc.publishInvalidation(ctx, key, 0, "invalidate")
	}

	return nil
}

func (dcc *DistributedCacheConsistency) publishInvalidation(ctx context.Context, key string, version int64, msgType string) {
	if Client == nil {
		return
	}

	msg := &InvalidationMessage{
		Key:     key,
		Version: version,
		Type:    msgType,
		Source:  "local",
		Time:    time.Now(),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		dcc.metrics.PublishFailures.Add(1)
		return
	}

	if err := Client.Publish(ctx, dcc.config.PubSubChannel, payload).Err(); err != nil {
		dcc.metrics.PublishFailures.Add(1)
	} else {
		dcc.metrics.PublishSuccess.Add(1)
	}
}

func (dcc *DistributedCacheConsistency) getRemoteVersion(ctx context.Context, key string) (int64, error) {
	if Client == nil {
		return 0, nil
	}

	versionKey := fmt.Sprintf("cache:version:%s", key)
	val, err := Client.Get(ctx, versionKey).Int64()
	if err == goredis.Nil {
		return 0, nil
	}
	return val, err
}

func (dcc *DistributedCacheConsistency) GetStatus() *CacheConsistencyStatus {
	dcc.mu.RLock()
	defer dcc.mu.RUnlock()

	levelStr := "eventual"
	switch dcc.config.Level {
	case ConsistencyLevelStrong:
		levelStr = "strong"
	case ConsistencyLevelLinearizable:
		levelStr = "linearizable"
	}

	modeStr := "write-through"
	switch dcc.config.UpdateMode {
	case CacheUpdateModeWriteBehind:
		modeStr = "write-behind"
	case CacheUpdateModeRefreshAhead:
		modeStr = "refresh-ahead"
	}

	metrics := map[string]interface{}{
		"total_writes":       dcc.metrics.TotalWrites.Load(),
		"total_reads":        dcc.metrics.TotalReads.Load(),
		"conflict_count":     dcc.metrics.ConflictCount.Load(),
		"invalidation_count": dcc.metrics.InvalidationCount.Load(),
		"publish_success":    dcc.metrics.PublishSuccess.Load(),
		"publish_failures":   dcc.metrics.PublishFailures.Load(),
		"local_hits":         dcc.metrics.LocalHits.Load(),
		"local_misses":       dcc.metrics.LocalMisses.Load(),
	}

	var lastSyncTime time.Time
	if lst := dcc.metrics.LastSyncTime.Load(); lst != nil {
		lastSyncTime = lst.(time.Time)
	}

	return &CacheConsistencyStatus{
		Level:        levelStr,
		UpdateMode:   modeStr,
		Metrics:      metrics,
		Connected:    Client != nil,
		LastSyncTime: lastSyncTime,
	}
}

func (dcc *DistributedCacheConsistency) Close() {
	if dcc.pubSubClient != nil {
		dcc.pubSubClient.Close()
	}
	if dcc.pubSubCancel != nil {
		dcc.pubSubCancel()
	}
	close(dcc.invalidationQueue)
}

var globalDistributedConsistency *DistributedCacheConsistency
var globalDistributedConsistencyOnce sync.Once

func InitDistributedCacheConsistency(config *ConsistencyConfig) {
	globalDistributedConsistencyOnce.Do(func() {
		globalDistributedConsistency = NewDistributedCacheConsistency(config)
	})
}

func GetDistributedCacheConsistency() *DistributedCacheConsistency {
	if globalDistributedConsistency == nil {
		InitDistributedCacheConsistency(nil)
	}
	return globalDistributedConsistency
}
