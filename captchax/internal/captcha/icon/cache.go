package icon

import (
	"captchax/config"
	"captchax/pkg/cache"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCaptchaNotFound = errors.New("captcha not found")
	ErrCaptchaExpired  = errors.New("captcha expired")
	ErrCaptchaVerified = errors.New("captcha already verified")
)

// CacheData stores captcha data in Redis
type CacheData struct {
	ID             string   `json:"id"`
	TargetIconIDs  []string `json:"target_icon_ids"`
	CreatedAt      int64    `json:"created_at"`
	Verified       bool     `json:"verified"`
}

// CacheManager handles Redis operations for icon captcha
type CacheManager struct {
	redis *cache.RedisClient
	cfg   *config.CaptchaConfig
}

// NewCacheManager creates a new CacheManager
func NewCacheManager(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *CacheManager {
	return &CacheManager{
		redis: redisClient,
		cfg:   cfg,
	}
}

// keyForID generates Redis key from captcha ID
func (cm *CacheManager) keyForID(id string) string {
	return fmt.Sprintf("captcha:icon:%s", id)
}

// Set stores captcha data in Redis
func (cm *CacheManager) Set(ctx context.Context, id string, data *CacheData) error {
	key := cm.keyForID(id)

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	expiration := time.Duration(cm.cfg.ExpireMinutes) * time.Minute
	if err := cm.redis.Set(ctx, key, dataBytes, expiration); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Get retrieves captcha data from Redis
func (cm *CacheManager) Get(ctx context.Context, id string) (*CacheData, error) {
	key := cm.keyForID(id)

	dataStr, err := cm.redis.Get(ctx, key)
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, ErrCaptchaNotFound
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	var data CacheData
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	if cm.isExpired(&data) {
		_ = cm.Delete(ctx, id)
		return nil, ErrCaptchaExpired
	}

	return &data, nil
}

// Delete removes captcha data from Redis
func (cm *CacheManager) Delete(ctx context.Context, id string) error {
	key := cm.keyForID(id)
	if err := cm.redis.Del(ctx, key); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

// MarkVerified marks captcha as verified
func (cm *CacheManager) MarkVerified(ctx context.Context, id string) error {
	data, err := cm.Get(ctx, id)
	if err != nil {
		return err
	}

	if data.Verified {
		return ErrCaptchaVerified
	}

	data.Verified = true
	return cm.Set(ctx, id, data)
}

// isExpired checks if captcha has expired
func (cm *CacheManager) isExpired(data *CacheData) bool {
	if data.CreatedAt == 0 {
		return false
	}
	expiration := time.Duration(cm.cfg.ExpireMinutes) * time.Minute
	expirationTime := time.Unix(data.CreatedAt, 0).Add(expiration)
	return time.Now().After(expirationTime)
}

// Exists checks if captcha exists in Redis
func (cm *CacheManager) Exists(ctx context.Context, id string) (bool, error) {
	key := cm.keyForID(id)
	count, err := cm.redis.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}
