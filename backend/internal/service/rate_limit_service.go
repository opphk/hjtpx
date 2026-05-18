package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

type RateLimitService struct{}

func NewRateLimitService() *RateLimitService {
	return &RateLimitService{}
}

type RateLimitConfig struct {
	MaxRequests int
	WindowSecs  int
}

const (
	PrefixIPRateLimit    = "ratelimit:ip:"
	PrefixUserRateLimit  = "ratelimit:user:"
	PrefixAppRateLimit   = "ratelimit:app:"
	PrefixBlacklist      = "blacklist:"
	PrefixWhitelist      = "whitelist:"
	PrefixBan            = "ban:"
	PrefixViolation      = "violation:"
	PrefixFailedAttempts = "failed:"
)

var DefaultIPConfig = RateLimitConfig{MaxRequests: 60, WindowSecs: 60}
var DefaultUserConfig = RateLimitConfig{MaxRequests: 100, WindowSecs: 60}
var DefaultAppConfig = RateLimitConfig{MaxRequests: 200, WindowSecs: 60}

func (s *RateLimitService) CheckIPRateLimit(ctx context.Context, ip string, config *RateLimitConfig) (*RateLimitResult, error) {
	if config == nil {
		config = &DefaultIPConfig
	}
	key := PrefixIPRateLimit + ip
	return s.checkRateLimit(ctx, key, config)
}

func (s *RateLimitService) RecordViolation(ctx context.Context, identifier string, violationType string) error {
	if redis.Client == nil {
		return nil
	}
	key := PrefixViolation + identifier
	return redis.Client.Set(ctx, key, violationType, 24*time.Hour).Err()
}

func (s *RateLimitService) RecordFailedAttempt(ctx context.Context, identifier string) (int64, error) {
	if redis.Client == nil {
		return 0, nil
	}
	key := PrefixFailedAttempts + identifier
	count, err := redis.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	redis.Client.Expire(ctx, key, 1*time.Hour)
	return count, nil
}

func (s *RateLimitService) ClearFailedAttempts(ctx context.Context, identifier string) error {
	if redis.Client == nil {
		return nil
	}
	key := PrefixFailedAttempts + identifier
	return redis.Client.Del(ctx, key).Err()
}

func (s *RateLimitService) IsWhitelisted(ctx context.Context, identifier string, whitelistType string) (bool, error) {
	if redis.Client == nil {
		return false, nil
	}
	key := PrefixWhitelist + whitelistType + ":" + identifier
	exists, err := redis.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *RateLimitService) AddToWhitelist(ctx context.Context, identifier string, whitelistType string, ttl time.Duration) error {
	if redis.Client == nil {
		return nil
	}
	key := PrefixWhitelist + whitelistType + ":" + identifier
	return redis.Client.Set(ctx, key, "1", ttl).Err()
}

func (s *RateLimitService) RemoveFromWhitelist(ctx context.Context, identifier string, whitelistType string) error {
	if redis.Client == nil {
		return nil
	}
	key := PrefixWhitelist + whitelistType + ":" + identifier
	return redis.Client.Del(ctx, key).Err()
}

func (s *RateLimitService) CheckUserRateLimit(ctx context.Context, userID uint, config *RateLimitConfig) (*RateLimitResult, error) {
	if config == nil {
		config = &DefaultUserConfig
	}
	key := fmt.Sprintf("%s%d", PrefixUserRateLimit, userID)
	return s.checkRateLimit(ctx, key, config)
}

func (s *RateLimitService) CheckAppRateLimit(ctx context.Context, appID uint, config *RateLimitConfig) (*RateLimitResult, error) {
	if config == nil {
		config = &DefaultAppConfig
	}
	key := fmt.Sprintf("%s%d", PrefixAppRateLimit, appID)
	return s.checkRateLimit(ctx, key, config)
}

func (s *RateLimitService) checkRateLimit(ctx context.Context, key string, config *RateLimitConfig) (*RateLimitResult, error) {
	result := &RateLimitResult{
		Allowed:   true,
		Remaining: float64(config.MaxRequests - 1),
		ResetAt:   time.Now().Add(time.Duration(config.WindowSecs) * time.Second),
	}

	if redis.Client == nil {
		return result, nil
	}

	pipe := redis.Client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to execute redis pipeline: %w", err)
	}

	count := incrCmd.Val()
	ttl := ttlCmd.Val()

	if ttl > 0 {
		result.ResetAt = time.Now().Add(ttl)
	}

	if count == 1 {
		redis.Client.Expire(ctx, key, time.Duration(config.WindowSecs)*time.Second)
	}

	if int(count) > config.MaxRequests {
		result.Allowed = false
		result.Remaining = 0
		return result, nil
	}

	result.Remaining = float64(config.MaxRequests - int(count))
	return result, nil
}
