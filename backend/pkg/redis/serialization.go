package redis

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"math"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type SerializationFormat int

const (
	SerializationJSON SerializationFormat = iota
	SerializationGob
	SerializationMsgpack
)

type Serializer interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type JSONSerializer struct{}

func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

func (js *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (js *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type GobSerializer struct {
	pool sync.Pool
}

func NewGobSerializer() *GobSerializer {
	return &GobSerializer{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

func (gs *GobSerializer) Marshal(v interface{}) ([]byte, error) {
	buf := gs.pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		gs.pool.Put(buf)
	}()

	encoder := gob.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (gs *GobSerializer) Unmarshal(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(v)
}

type OptimizedSerializer struct {
	jsonSerializer  *JSONSerializer
	gobSerializer   *GobSerializer
	defaultFormat  SerializationFormat
	compressionThreshold int
}

func NewOptimizedSerializer() *OptimizedSerializer {
	return &OptimizedSerializer{
		jsonSerializer:  NewJSONSerializer(),
		gobSerializer:   NewGobSerializer(),
		defaultFormat:  SerializationJSON,
		compressionThreshold: 1024,
	}
}

func (os *OptimizedSerializer) Marshal(v interface{}) ([]byte, error) {
	var data []byte
	var err error

	switch os.defaultFormat {
	case SerializationGob:
		data, err = os.gobSerializer.Marshal(v)
	default:
		data, err = os.jsonSerializer.Marshal(v)
	}

	if err != nil {
		return nil, err
	}

	if len(data) >= os.compressionThreshold {
		compressed, err := compress(data)
		if err == nil {
			return append([]byte{0x01}, compressed...), nil
		}
	}

	return append([]byte{0x00}, data...), nil
}

func (os *OptimizedSerializer) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}

	isCompressed := data[0] == 0x01
	payload := data[1:]

	if isCompressed {
		var err error
		payload, err = decompress(payload)
		if err != nil {
			return err
		}
	}

	switch os.defaultFormat {
	case SerializationGob:
		return os.gobSerializer.Unmarshal(payload, v)
	default:
		return os.jsonSerializer.Unmarshal(payload, v)
	}
}

func (os *OptimizedSerializer) SetDefaultFormat(format SerializationFormat) {
	os.defaultFormat = format
}

type CachedSerializer struct {
	serializer  Serializer
	cache       sync.Map
	maxCacheSize int
	mu          sync.RWMutex
}

func NewCachedSerializer(serializer Serializer, maxCacheSize int) *CachedSerializer {
	if maxCacheSize <= 0 {
		maxCacheSize = 1000
	}
	return &CachedSerializer{
		serializer:  serializer,
		maxCacheSize: maxCacheSize,
	}
}

func (cs *CachedSerializer) Marshal(v interface{}) ([]byte, error) {
	key := cs.getKey(v)
	if key != "" {
		if cached, ok := cs.cache.Load(key); ok {
			return cached.([]byte), nil
		}
	}

	data, err := cs.serializer.Marshal(v)
	if err != nil {
		return nil, err
	}

	if key != "" {
		cs.cache.Store(key, data)
		cs.evictIfNeeded()
	}

	return data, nil
}

func (cs *CachedSerializer) Unmarshal(data []byte, v interface{}) error {
	return cs.serializer.Unmarshal(data, v)
}

func (cs *CachedSerializer) getKey(v interface{}) string {
	return ""
}

func (cs *CachedSerializer) evictIfNeeded() {
}

type ConnectionPoolMonitor struct {
	client          goredis.Cmdable
	mu             sync.RWMutex
	collectionInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewConnectionPoolMonitor(client goredis.Cmdable, collectionInterval time.Duration) *ConnectionPoolMonitor {
	if collectionInterval <= 0 {
		collectionInterval = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	cpm := &ConnectionPoolMonitor{
		client:              client,
		collectionInterval:  collectionInterval,
		ctx:                 ctx,
		cancel:              cancel,
	}

	go cpm.startMonitoring()
	return cpm
}

func (cpm *ConnectionPoolMonitor) startMonitoring() {
	ticker := time.NewTicker(cpm.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cpm.ctx.Done():
			return
		case <-ticker.C:
			cpm.collectStats()
		}
	}
}

func (cpm *ConnectionPoolMonitor) collectStats() {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	if cpm.client == nil {
		return
	}

}

func (cpm *ConnectionPoolMonitor) GetStats() interface{} {
	cpm.mu.RLock()
	defer cpm.mu.RUnlock()
	return nil
}

func (cpm *ConnectionPoolMonitor) Close() {
	cpm.cancel()
}

type PerformanceOptimizer struct {
	serializer      *OptimizedSerializer
	poolMonitor     *ConnectionPoolMonitor
	batchOperator   *BatchOperator
}

func NewPerformanceOptimizer(client goredis.Cmdable) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		serializer:    NewOptimizedSerializer(),
		poolMonitor:   NewConnectionPoolMonitor(client, 10*time.Second),
		batchOperator: NewBatchOperator(client, 100),
	}
}

func (po *PerformanceOptimizer) GetSerializer() *OptimizedSerializer {
	return po.serializer
}

func (po *PerformanceOptimizer) GetBatchOperator() *BatchOperator {
	return po.batchOperator
}

func (po *PerformanceOptimizer) Close() {
	po.poolMonitor.Close()
}

type LazyLoader struct {
	cache     *EnhancedCache
	loader    func(ctx context.Context, key string) (interface{}, error)
	ttl       time.Duration
	mu        sync.Mutex
	loading   map[string]chan struct{}
}

func NewLazyLoader(cache *EnhancedCache, ttl time.Duration, loader func(ctx context.Context, key string) (interface{}, error)) *LazyLoader {
	return &LazyLoader{
		cache:   cache,
		loader:  loader,
		ttl:     ttl,
		loading: make(map[string]chan struct{}),
	}
}

func (ll *LazyLoader) Get(ctx context.Context, key string) (interface{}, error) {
	data, err := ll.cache.Get(ctx, key, nil)
	if err == nil {
		var result interface{}
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
	}

	ll.mu.Lock()
	if ch, ok := ll.loading[key]; ok {
		ll.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return ll.Get(ctx, key)
	}

	ch := make(chan struct{})
	ll.loading[key] = ch
	ll.mu.Unlock()

	defer func() {
		ll.mu.Lock()
		delete(ll.loading, key)
		close(ch)
		ll.mu.Unlock()
	}()

	value, err := ll.loader(ctx, key)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(value)
	if err == nil {
		ll.cache.Set(ctx, key, jsonData, &SetOptions{TTL: ll.ttl})
	}

	return value, nil
}

type PrefetchBuffer struct {
	buffer    []string
	maxSize   int
	mu        sync.Mutex
	onProcess func(keys []string)
}

func NewPrefetchBuffer(maxSize int, onProcess func(keys []string)) *PrefetchBuffer {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &PrefetchBuffer{
		buffer:    make([]string, 0, maxSize),
		maxSize:   maxSize,
		onProcess: onProcess,
	}
}

func (pb *PrefetchBuffer) Add(key string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.buffer = append(pb.buffer, key)

	if len(pb.buffer) >= pb.maxSize && pb.onProcess != nil {
		pb.onProcess(pb.buffer)
		pb.buffer = make([]string, 0, pb.maxSize)
	}
}

func (pb *PrefetchBuffer) Flush() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if len(pb.buffer) > 0 && pb.onProcess != nil {
		pb.onProcess(pb.buffer)
		pb.buffer = make([]string, 0, pb.maxSize)
	}
}

func (pb *PrefetchBuffer) Size() int {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	return len(pb.buffer)
}

type AdaptiveTtlManager struct {
	defaultTtl    time.Duration
	minTtl        time.Duration
	maxTtl        time.Duration
	accessCounts  *sync.Map
	mu            sync.Mutex
}

func NewAdaptiveTtlManager(defaultTtl, minTtl, maxTtl time.Duration) *AdaptiveTtlManager {
	if minTtl <= 0 {
		minTtl = 1 * time.Minute
	}
	if maxTtl <= 0 {
		maxTtl = 24 * time.Hour
	}
	if defaultTtl < minTtl {
		defaultTtl = minTtl
	}
	if defaultTtl > maxTtl {
		defaultTtl = maxTtl
	}

	return &AdaptiveTtlManager{
		defaultTtl:   defaultTtl,
		minTtl:       minTtl,
		maxTtl:       maxTtl,
		accessCounts: &sync.Map{},
	}
}

func (atm *AdaptiveTtlManager) RecordAccess(key string) {
	val, _ := atm.accessCounts.LoadOrStore(key, int64(0))
	newCount := val.(int64) + 1
	atm.accessCounts.Store(key, newCount)
}

func (atm *AdaptiveTtlManager) GetTtl(key string) time.Duration {
	val, ok := atm.accessCounts.Load(key)
	if !ok {
		return atm.defaultTtl
	}

	count := val.(int64)
	ttl := time.Duration(float64(atm.defaultTtl) * (1 + math.Log10(float64(count+1))))

	if ttl < atm.minTtl {
		return atm.minTtl
	}
	if ttl > atm.maxTtl {
		return atm.maxTtl
	}
	return ttl
}

func (atm *AdaptiveTtlManager) Reset() {
	atm.accessCounts = &sync.Map{}
}

var (
	globalOptimizer *PerformanceOptimizer
	globalOptimizerOnce sync.Once
)

func InitPerformanceOptimizer(client goredis.Cmdable) {
	globalOptimizerOnce.Do(func() {
		globalOptimizer = NewPerformanceOptimizer(client)
	})
}

func GetPerformanceOptimizer() *PerformanceOptimizer {
	return globalOptimizer
}
