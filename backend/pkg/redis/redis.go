package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/hjtpx/hjtpx/pkg/config"
)

var Client *redis.Client
var ctx = context.Background()

const (
	CaptchaCacheTTL   = 5 * time.Minute
	UserCacheTTL      = 30 * time.Minute
	StatsCacheTTL     = 1 * time.Minute
	SessionCacheTTL   = 10 * time.Minute
	RateLimitCacheTTL = 1 * time.Minute
)

type CacheStats struct {
	Hits      int64
	Misses    int64
	Keys      int64
	MemUsage  int64
	Expries   int64
}

type CacheMetrics struct {
	TotalRequests int64
	CacheHits     int64
	CacheMisses   int64
	HitRate       float64
}

func ConnectRedis(cfg *config.RedisConfig) error {
	Client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     50,
		MinIdleConns: 10,
		PoolTimeout:  4 * time.Second,
	})

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	return nil
}

func CloseRedis() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

func GetClient() *redis.Client {
	return Client
}

func GetContext() context.Context {
	return ctx
}

func HealthCheck() error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return Client.Ping(ctx).Err()
}

func GetCacheStats() (CacheStats, error) {
	if Client == nil {
		return CacheStats{}, fmt.Errorf("redis client is nil")
	}

	_, err := Client.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return CacheStats{}, err
	}

	stats := CacheStats{}

	return stats, nil
}

func GetCacheMetrics() CacheMetrics {
	return CacheMetrics{
		TotalRequests: 0,
		CacheHits:    0,
		CacheMisses:  0,
		HitRate:      0,
	}
}

func WarmupCache() error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}

	stats, err := Client.Info(ctx, "keyspace").Result()
	if err != nil {
		return err
	}

	fmt.Printf("Redis cache warmup completed, keyspace info: %s\n", stats)
	return nil
}

func SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return Client.Set(ctx, key, value, ttl).Err()
}

func Get(key string) (string, error) {
	if Client == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	return Client.Get(ctx, key).Result()
}

func Set(key string, value interface{}) error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return Client.Set(ctx, key, value, 0).Err()
}

func Delete(key string) error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return Client.Del(ctx, key).Err()
}

func Exists(key string) (bool, error) {
	if Client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	result, err := Client.Exists(ctx, key).Result()
	return result > 0, err
}

func Incr(key string) (int64, error) {
	if Client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return Client.Incr(ctx, key).Result()
}

func Expire(key string, ttl time.Duration) error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return Client.Expire(ctx, key, ttl).Err()
}

func GetTTL(key string) (time.Duration, error) {
	if Client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return Client.TTL(ctx, key).Result()
}

func SetNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	if Client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	return Client.SetNX(ctx, key, value, ttl).Result()
}

func HSet(key string, values ...interface{}) error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return Client.HSet(ctx, key, values...).Err()
}

func HGet(key, field string) (string, error) {
	if Client == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	return Client.HGet(ctx, key, field).Result()
}

func HGetAll(key string) (map[string]string, error) {
	if Client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	return Client.HGetAll(ctx, key).Result()
}

func SAdd(key string, members ...interface{}) error {
	if Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return Client.SAdd(ctx, key, members...).Err()
}

func SMembers(key string) ([]string, error) {
	if Client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	return Client.SMembers(ctx, key).Result()
}

func IsMember(key string, member interface{}) (bool, error) {
	if Client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	return Client.SIsMember(ctx, key, member).Result()
}

func Pipeline() redis.Pipeliner {
	if Client == nil {
		return nil
	}
	return Client.Pipeline()
}

func ClosePipeline(pipe redis.Pipeliner) error {
	if pipe == nil {
		return nil
	}
	_, err := pipe.Exec(ctx)
	return err
}
