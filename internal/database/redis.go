package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hjtpx/internal/config"
	"hjtpx/internal/utils"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

type RedisCache struct {
	client *redis.Client
}

func InitRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	RedisClient = client
	utils.Info("Redis connection established at %s", cfg.Addr())

	return client, nil
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := rc.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

func (rc *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get cache: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	if err := rc.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

func (rc *RedisCache) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rc.Set(ctx, key, value, expiration)
}

func (rc *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := rc.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

func CloseRedis() error {
	if RedisClient == nil {
		return nil
	}

	if err := RedisClient.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	utils.Info("Redis connection closed")
	return nil
}

func GetRedisClient() *redis.Client {
	return RedisClient
}

func GetRedisCache() *RedisCache {
	return NewRedisCache(RedisClient)
}
