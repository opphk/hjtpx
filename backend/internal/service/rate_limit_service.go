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

type RateLimitResult struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
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
		Remaining: config.MaxRequests - 1,
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

	result.Remaining = config.MaxRequests - int(count)
	return result, nil
}

func (s *RateLimitService) AddToBlacklist(ctx context.Context, identifier string, blacklistType string, duration time.Duration) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixBlacklist, blacklistType, identifier)
	if duration > 0 {
		return redis.Client.Set(ctx, key, "1", duration).Err()
	}
	return redis.Client.Set(ctx, key, "1", 0).Err()
}

func (s *RateLimitService) RemoveFromBlacklist(ctx context.Context, identifier string, blacklistType string) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixBlacklist, blacklistType, identifier)
	return redis.Client.Del(ctx, key).Err()
}

func (s *RateLimitService) IsBlacklisted(ctx context.Context, identifier string, blacklistType string) (bool, error) {
	if redis.Client == nil {
		return false, nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixBlacklist, blacklistType, identifier)
	result, err := redis.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (s *RateLimitService) AddToWhitelist(ctx context.Context, identifier string, whitelistType string, duration time.Duration) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixWhitelist, whitelistType, identifier)
	if duration > 0 {
		return redis.Client.Set(ctx, key, "1", duration).Err()
	}
	return redis.Client.Set(ctx, key, "1", 0).Err()
}

func (s *RateLimitService) RemoveFromWhitelist(ctx context.Context, identifier string, whitelistType string) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixWhitelist, whitelistType, identifier)
	return redis.Client.Del(ctx, key).Err()
}

func (s *RateLimitService) IsWhitelisted(ctx context.Context, identifier string, whitelistType string) (bool, error) {
	if redis.Client == nil {
		return false, nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixWhitelist, whitelistType, identifier)
	result, err := redis.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (s *RateLimitService) RecordViolation(ctx context.Context, identifier string, violationType string) (int, error) {
	if redis.Client == nil {
		return 0, nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixViolation, violationType, identifier)
	count, err := redis.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		redis.Client.Expire(ctx, key, 10*time.Minute)
	}
	return int(count), nil
}

func (s *RateLimitService) GetViolationCount(ctx context.Context, identifier string, violationType string) (int, error) {
	if redis.Client == nil {
		return 0, nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixViolation, violationType, identifier)
	result, err := redis.Client.Get(ctx, key).Int()
	if err != nil {
		return 0, nil
	}
	return result, nil
}

func (s *RateLimitService) ClearViolations(ctx context.Context, identifier string, violationType string) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixViolation, violationType, identifier)
	return redis.Client.Del(ctx, key).Err()
}

func (s *RateLimitService) BanIdentifier(ctx context.Context, identifier string, banType string, duration time.Duration) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixBan, banType, identifier)
	return redis.Client.Set(ctx, key, "1", duration).Err()
}

func (s *RateLimitService) UnbanIdentifier(ctx context.Context, identifier string, banType string) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixBan, banType, identifier)
	return redis.Client.Del(ctx, key).Err()
}

func (s *RateLimitService) IsBanned(ctx context.Context, identifier string, banType string) (bool, time.Duration, error) {
	if redis.Client == nil {
		return false, 0, nil
	}
	key := fmt.Sprintf("%s%s:%s", PrefixBan, banType, identifier)
	ttl, err := redis.Client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	return ttl > 0, ttl, nil
}

func (s *RateLimitService) RecordFailedAttempt(ctx context.Context, identifier string) (int, error) {
	if redis.Client == nil {
		return 0, nil
	}
	key := fmt.Sprintf("%s%s", PrefixFailedAttempts, identifier)
	count, err := redis.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		redis.Client.Expire(ctx, key, 15*time.Minute)
	}
	return int(count), nil
}

func (s *RateLimitService) GetFailedAttempts(ctx context.Context, identifier string) (int, error) {
	if redis.Client == nil {
		return 0, nil
	}
	key := fmt.Sprintf("%s%s", PrefixFailedAttempts, identifier)
	result, err := redis.Client.Get(ctx, key).Int()
	if err != nil {
		return 0, nil
	}
	return result, nil
}

func (s *RateLimitService) ClearFailedAttempts(ctx context.Context, identifier string) error {
	if redis.Client == nil {
		return nil
	}
	key := fmt.Sprintf("%s%s", PrefixFailedAttempts, identifier)
	return redis.Client.Del(ctx, key).Err()
}

func (s *RateLimitService) ShouldAutoBan(ctx context.Context, identifier string, violationType string, threshold int) (bool, error) {
	count, err := s.GetViolationCount(ctx, identifier, violationType)
	if err != nil {
		return false, err
	}
	return count >= threshold, nil
}

func (s *RateLimitService) AutoBan(ctx context.Context, identifier string, banType string) error {
	violationCount, _ := s.GetViolationCount(ctx, identifier, banType)
	duration := s.CalculateBanDuration(violationCount)
	return s.BanIdentifier(ctx, identifier, banType, duration)
}

func (s *RateLimitService) CalculateBanDuration(violationCount int) time.Duration {
	baseDuration := 5 * time.Minute
	switch {
	case violationCount >= 10:
		return 24 * time.Hour
	case violationCount >= 7:
		return 12 * time.Hour
	case violationCount >= 5:
		return 6 * time.Hour
	case violationCount >= 3:
		return 2 * time.Hour
	case violationCount >= 2:
		return 30 * time.Minute
	default:
		return baseDuration
	}
}
