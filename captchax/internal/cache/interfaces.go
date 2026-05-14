package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

type ReadThroughCache interface {
	Cache
	GetOrLoad(ctx context.Context, key string, loader func() ([]byte, error)) ([]byte, bool, error)
}

type WriteThroughCache interface {
	Cache
	SetOrInvalidate(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Invalidate(ctx context.Context, key string) error
}

type MetricsCollector interface {
	RecordHit()
	RecordMiss()
	RecordLatency(d time.Duration)
	RecordEviction()
}

type CacheStats struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	MemorySize int64
	ItemCount  int64
	HitRate    float64
}
