package click

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheManager struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCacheManager(redisAddr string, redisPassword string, db int) (*CacheManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &CacheManager{
		client: client,
		ttl:    CacheExpireMinutes * time.Minute,
	}, nil
}

func (cm *CacheManager) Store(ctx context.Context, data *CaptchaData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal captcha data: %w", err)
	}

	key := cm.buildKey(data.ID)
	if err := cm.client.Set(ctx, key, jsonData, cm.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store captcha in redis: %w", err)
	}

	return nil
}

func (cm *CacheManager) Get(ctx context.Context, captchaID string) (*CaptchaData, error) {
	key := cm.buildKey(captchaID)

	jsonData, err := cm.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("captcha not found or expired")
		}
		return nil, fmt.Errorf("failed to get captcha from redis: %w", err)
	}

	var data CaptchaData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal captcha data: %w", err)
	}

	if time.Since(data.CreatedAt) > cm.ttl {
		cm.Delete(ctx, captchaID)
		return nil, fmt.Errorf("captcha expired")
	}

	return &data, nil
}

func (cm *CacheManager) Delete(ctx context.Context, captchaID string) error {
	key := cm.buildKey(captchaID)

	if err := cm.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete captcha from redis: %w", err)
	}

	return nil
}

func (cm *CacheManager) Exists(ctx context.Context, captchaID string) (bool, error) {
	key := cm.buildKey(captchaID)

	exists, err := cm.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check captcha existence: %w", err)
	}

	return exists > 0, nil
}

func (cm *CacheManager) buildKey(captchaID string) string {
	return fmt.Sprintf("captcha:click:%s", captchaID)
}

func (cm *CacheManager) Close() error {
	return cm.client.Close()
}

type MockCacheManager struct {
	data map[string]*CaptchaData
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		data: make(map[string]*CaptchaData),
	}
}

func (mcm *MockCacheManager) Store(ctx context.Context, data *CaptchaData) error {
	mcm.data[data.ID] = data
	return nil
}

func (mcm *MockCacheManager) Get(ctx context.Context, captchaID string) (*CaptchaData, error) {
	data, ok := mcm.data[captchaID]
	if !ok {
		return nil, fmt.Errorf("captcha not found or expired")
	}

	if time.Since(data.CreatedAt) > CacheExpireMinutes*time.Minute {
		delete(mcm.data, captchaID)
		return nil, fmt.Errorf("captcha expired")
	}

	return data, nil
}

func (mcm *MockCacheManager) Delete(ctx context.Context, captchaID string) error {
	delete(mcm.data, captchaID)
	return nil
}

func (mcm *MockCacheManager) Exists(ctx context.Context, captchaID string) (bool, error) {
	_, ok := mcm.data[captchaID]
	return ok, nil
}

func (mcm *MockCacheManager) Close() error {
	return nil
}

type CaptchaCache interface {
	Store(ctx context.Context, data *CaptchaData) error
	Get(ctx context.Context, captchaID string) (*CaptchaData, error)
	Delete(ctx context.Context, captchaID string) error
	Exists(ctx context.Context, captchaID string) (bool, error)
	Close() error
}
