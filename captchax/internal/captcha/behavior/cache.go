package behavior

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"captchax/pkg/cache"
)

type CacheManager struct {
	redis      *cache.RedisClient
	expiration time.Duration
}

func NewCacheManager(redisClient *cache.RedisClient) *CacheManager {
	return &CacheManager{
		redis:      redisClient,
		expiration: 5 * time.Minute,
	}
}

func (cm *CacheManager) Set(ctx context.Context, token string, data *CaptchaData) error {
	if cm.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal captcha data: %w", err)
	}

	key := cm.buildKey(token)
	return cm.redis.Set(ctx, key, dataBytes, cm.expiration)
}

func (cm *CacheManager) Get(ctx context.Context, token string) (*CaptchaData, error) {
	if cm.redis == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	key := cm.buildKey(token)
	data, err := cm.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("captcha not found: %w", err)
	}

	var captchaData CaptchaData
	if err := json.Unmarshal([]byte(data), &captchaData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal captcha data: %w", err)
	}

	if cm.isExpired(&captchaData) {
		_ = cm.Delete(ctx, token)
		return nil, fmt.Errorf("captcha expired")
	}

	return &captchaData, nil
}

func (cm *CacheManager) Delete(ctx context.Context, token string) error {
	if cm.redis == nil {
		return nil
	}

	key := cm.buildKey(token)
	return cm.redis.Del(ctx, key)
}

func (cm *CacheManager) Exists(ctx context.Context, token string) (bool, error) {
	if cm.redis == nil {
		return false, fmt.Errorf("redis client not available")
	}

	key := cm.buildKey(token)
	data, err := cm.redis.Get(ctx, key)
	if err != nil {
		return false, nil
	}

	if len(data) == 0 {
		return false, nil
	}

	var captchaData CaptchaData
	if err := json.Unmarshal([]byte(data), &captchaData); err != nil {
		return false, nil
	}

	if cm.isExpired(&captchaData) {
		_ = cm.Delete(ctx, token)
		return false, nil
	}

	return true, nil
}

func (cm *CacheManager) buildKey(token string) string {
	return fmt.Sprintf("captcha:behavior:%s", token)
}

func (cm *CacheManager) isExpired(data *CaptchaData) bool {
	if data.CreatedAt == 0 {
		return false
	}

	expirationTime := time.Unix(data.CreatedAt, 0).Add(cm.expiration)
	return time.Now().After(expirationTime)
}

func (cm *CacheManager) SetExpiration(expiration time.Duration) {
	if expiration > 0 {
		cm.expiration = expiration
	}
}

func (cm *CacheManager) GetExpiration() time.Duration {
	return cm.expiration
}

func (cm *CacheManager) GetTTL(ctx context.Context, token string) (time.Duration, error) {
	if cm.redis == nil {
		return 0, fmt.Errorf("redis client not available")
	}

	key := cm.buildKey(token)
	return cm.redis.TTL(ctx, key)
}

func (cm *CacheManager) Refresh(ctx context.Context, token string) error {
	if cm.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	key := cm.buildKey(token)
	return cm.redis.Expire(ctx, key, cm.expiration)
}

type MockCacheManager struct {
	data map[string]*CaptchaData
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		data: make(map[string]*CaptchaData),
	}
}

func (mcm *MockCacheManager) Set(ctx context.Context, token string, captchaData *CaptchaData) error {
	mcm.data[token] = captchaData
	return nil
}

func (mcm *MockCacheManager) Get(ctx context.Context, token string) (*CaptchaData, error) {
	data, ok := mcm.data[token]
	if !ok {
		return nil, fmt.Errorf("captcha not found")
	}
	return data, nil
}

func (mcm *MockCacheManager) Delete(ctx context.Context, token string) error {
	delete(mcm.data, token)
	return nil
}

func (mcm *MockCacheManager) Exists(ctx context.Context, token string) (bool, error) {
	_, ok := mcm.data[token]
	return ok, nil
}

func (mcm *MockCacheManager) Clear() {
	mcm.data = make(map[string]*CaptchaData)
}

func (mcm *MockCacheManager) Size() int {
	return len(mcm.data)
}
