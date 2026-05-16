package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/hjtpx/hjtpx/pkg/config"
)

var (
	Client     *redis.Client
	Cluster    *redis.ClusterClient
	ctx        = context.Background()
	clientMu   sync.RWMutex
	clusterMu  sync.RWMutex
)

type RedisClient struct {
	client *redis.Client
}

type PoolStats struct {
	TotalConns    uint32
	IdleConns     uint32
	StaleConns    uint32
	MetricsTotal  uint32
	MetricsHits   uint32
	MetricsMisses uint32
	MetricsTimeouts uint32
}

type Config struct {
	PoolSize        int
	MinIdleConns    int
	MaxIdleConns    int
	MaxRetries      int
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PoolTimeout     time.Duration
	PoolMaxIdle     int
	PoolMinIdle     int
	PoolMaxActive   int
}

var defaultConfig = &Config{
	PoolSize:        100,
	MinIdleConns:    10,
	MaxIdleConns:    50,
	MaxRetries:      3,
	DialTimeout:     5 * time.Second,
	ReadTimeout:     3 * time.Second,
	WriteTimeout:    3 * time.Second,
	PoolTimeout:     4 * time.Second,
	PoolMaxIdle:     50,
	PoolMinIdle:     10,
	PoolMaxActive:   100,
}

func GetContext() context.Context {
	return ctx
}

func ConnectRedis(cfg *config.RedisConfig) error {
	clientMu.Lock()
	defer clientMu.Unlock()

	Client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     100,
		MinIdleConns: 10,
		PoolTimeout:  4 * time.Second,
	})

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	return nil
}

func CloseRedis() error {
	clientMu.Lock()
	defer clientMu.Unlock()

	if Cluster != nil {
		if err := Cluster.Close(); err != nil {
			return err
		}
		Cluster = nil
	}

	if Client != nil {
		return Client.Close()
	}
	return nil
}

func GetClient() *redis.Client {
	clientMu.RLock()
	defer clientMu.RUnlock()
	return Client
}

func NewRedisClient(cfg *Config) (*RedisClient, error) {
	if cfg == nil {
		cfg = defaultConfig
	}

	client := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	return &RedisClient{
		client: client,
	}, nil
}

func (rc *RedisClient) SetPoolConfig(cfg *Config) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	if cfg == nil {
		cfg = defaultConfig
	}

	opts := &redis.Options{
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	}

	_ = opts

	return nil
}

func (rc *RedisClient) GetPoolStats() *PoolStats {
	if rc == nil || rc.client == nil {
		return &PoolStats{}
	}

	stats := rc.client.PoolStats()

	return &PoolStats{
		TotalConns:    stats.TotalConns,
		IdleConns:     stats.IdleConns,
		StaleConns:    stats.StaleConns,
		MetricsTotal:  stats.Hits + stats.Misses,
		MetricsHits:   stats.Hits,
		MetricsMisses: stats.Misses,
		MetricsTimeouts: stats.Timeouts,
	}
}

func (rc *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if rc == nil || rc.client == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	return rc.client.Get(ctx, key).Result()
}

func (rc *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return rc.client.Set(ctx, key, value, expiration).Err()
}

func (rc *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if len(keys) == 0 {
		return nil
	}
	return rc.client.Del(ctx, keys...).Err()
}

func (rc *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	if rc == nil || rc.client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return rc.client.Exists(ctx, keys...).Result()
}

func (rc *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return rc.client.Expire(ctx, key, expiration).Err()
}

func (rc *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	if rc == nil || rc.client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return rc.client.TTL(ctx, key).Result()
}

func (rc *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	if rc == nil || rc.client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return rc.client.Incr(ctx, key).Result()
}

func (rc *RedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	if rc == nil || rc.client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return rc.client.IncrBy(ctx, key, value).Result()
}

func (rc *RedisClient) Close() error {
	if rc == nil || rc.client == nil {
		return nil
	}
	return rc.client.Close()
}

type RedisCluster struct {
	client *redis.ClusterClient
}

func NewRedisCluster(addrs []string) (*RedisCluster, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses provided")
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		PoolSize: 100,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	clusterMu.Lock()
	Cluster = client
	clusterMu.Unlock()

	return &RedisCluster{client: client}, nil
}

func (rc *RedisCluster) Set(ctx context.Context, key string, value interface{}) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis cluster client is nil")
	}
	return rc.client.Set(ctx, key, value, 0).Err()
}

func (rc *RedisCluster) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis cluster client is nil")
	}
	return rc.client.Set(ctx, key, value, ttl).Err()
}

func (rc *RedisCluster) Get(ctx context.Context, key string) (string, error) {
	if rc == nil || rc.client == nil {
		return "", fmt.Errorf("redis cluster client is nil")
	}
	return rc.client.Get(ctx, key).Result()
}

func (rc *RedisCluster) Delete(ctx context.Context, keys ...string) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis cluster client is nil")
	}
	if len(keys) == 0 {
		return nil
	}
	return rc.client.Del(ctx, keys...).Err()
}

func (rc *RedisCluster) Exists(ctx context.Context, keys ...string) (int64, error) {
	if rc == nil || rc.client == nil {
		return 0, fmt.Errorf("redis cluster client is nil")
	}
	return rc.client.Exists(ctx, keys...).Result()
}

func (rc *RedisCluster) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if rc == nil || rc.client == nil {
		return fmt.Errorf("redis cluster client is nil")
	}
	return rc.client.Expire(ctx, key, expiration).Err()
}

func (rc *RedisCluster) TTL(ctx context.Context, key string) (time.Duration, error) {
	if rc == nil || rc.client == nil {
		return 0, fmt.Errorf("redis cluster client is nil")
	}
	return rc.client.TTL(ctx, key).Result()
}

func (rc *RedisCluster) Incr(ctx context.Context, key string) (int64, error) {
	if rc == nil || rc.client == nil {
		return 0, fmt.Errorf("redis cluster client is nil")
	}
	return rc.client.Incr(ctx, key).Result()
}

func (rc *RedisCluster) Close() error {
	clusterMu.Lock()
	defer clusterMu.Unlock()

	if rc == nil || rc.client == nil {
		return nil
	}

	err := rc.client.Close()
	Cluster = nil
	return err
}

func GetClusterClient() *redis.ClusterClient {
	clusterMu.RLock()
	defer clusterMu.RUnlock()
	return Cluster
}
