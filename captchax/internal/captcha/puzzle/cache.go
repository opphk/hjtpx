package puzzle

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
	ErrPuzzleNotFound  = errors.New("puzzle captcha not found")
	ErrPuzzleExpired    = errors.New("puzzle captcha expired")
	ErrPuzzleVerified  = errors.New("puzzle captcha already verified")
	ErrInvalidPiecePos = errors.New("invalid puzzle piece position")
)

type CacheManager struct {
	redis *cache.RedisClient
	cfg   *config.CaptchaConfig
}

type CacheData struct {
	ID           string      `json:"id"`
	TargetX      int         `json:"target_x"`
	TargetY      int         `json:"target_y"`
	TargetWidth  int         `json:"target_width"`
	TargetHeight int         `json:"target_height"`
	PieceShape   PuzzleShape `json:"piece_shape"`
	PieceSize    int         `json:"piece_size"`
	CreatedAt    int64       `json:"created_at"`
	Verified     bool        `json:"verified"`
	Attempts     int         `json:"attempts"`
}

func NewCacheManager(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *CacheManager {
	return &CacheManager{
		redis: redisClient,
		cfg:   cfg,
	}
}

func (cm *CacheManager) keyForID(id string) string {
	return fmt.Sprintf("captcha:puzzle:%s", id)
}

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

func (cm *CacheManager) Get(ctx context.Context, id string) (*CacheData, error) {
	key := cm.keyForID(id)

	dataStr, err := cm.redis.Get(ctx, key)
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, ErrPuzzleNotFound
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	var data CacheData
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	if cm.isExpired(&data) {
		_ = cm.Delete(ctx, id)
		return nil, ErrPuzzleExpired
	}

	return &data, nil
}

func (cm *CacheManager) Delete(ctx context.Context, id string) error {
	key := cm.keyForID(id)
	if err := cm.redis.Del(ctx, key); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

func (cm *CacheManager) MarkVerified(ctx context.Context, id string) error {
	data, err := cm.Get(ctx, id)
	if err != nil {
		return err
	}

	if data.Verified {
		return ErrPuzzleVerified
	}

	data.Verified = true

	return cm.Set(ctx, id, data)
}

func (cm *CacheManager) IncrementAttempts(ctx context.Context, id string) (int, error) {
	data, err := cm.Get(ctx, id)
	if err != nil {
		return 0, err
	}

	data.Attempts++
	if data.Attempts > cm.cfg.MaxAttempts && cm.cfg.MaxAttempts > 0 {
		_ = cm.Delete(ctx, id)
		return data.Attempts, fmt.Errorf("max attempts exceeded")
	}

	if err := cm.Set(ctx, id, data); err != nil {
		return data.Attempts, err
	}

	return data.Attempts, nil
}

func (cm *CacheManager) isExpired(data *CacheData) bool {
	if data.CreatedAt == 0 {
		return false
	}
	expiration := time.Duration(cm.cfg.ExpireMinutes) * time.Minute
	expirationTime := time.Unix(data.CreatedAt, 0).Add(expiration)
	return time.Now().After(expirationTime)
}

func (cm *CacheManager) Exists(ctx context.Context, id string) (bool, error) {
	key := cm.keyForID(id)
	count, err := cm.redis.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

func (cm *CacheManager) GetAttempts(ctx context.Context, id string) (int, error) {
	data, err := cm.Get(ctx, id)
	if err != nil {
		return 0, err
	}
	return data.Attempts, nil
}

func (cm *CacheManager) RemainingAttempts(ctx context.Context, id string) (int, error) {
	data, err := cm.Get(ctx, id)
	if err != nil {
		return 0, err
	}
	remaining := cm.cfg.MaxAttempts - data.Attempts
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}
