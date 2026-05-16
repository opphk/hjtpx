package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
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

var DefaultIPConfig = RateLimitConfig{MaxRequests: 100, WindowSecs: 60}
var DefaultUserConfig = RateLimitConfig{MaxRequests: 200, WindowSecs: 60}
var DefaultAppConfig = RateLimitConfig{MaxRequests: 500, WindowSecs: 60}

func NewRateLimitConfigFromConfig() *RateLimitConfig {
	cfg := config.GetConfig()
	return &RateLimitConfig{
		MaxRequests: cfg.RateLimit.DefaultLimit,
		WindowSecs:  cfg.RateLimit.WindowSecs,
	}
}

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

func (s *RateLimitService) CheckRateLimitWithBurst(ctx context.Context, key string, maxRequests int, windowSecs int, burstLimit int) (*RateLimitResult, error) {
	result := &RateLimitResult{
		Allowed:   true,
		Remaining: maxRequests - 1,
		ResetAt:   time.Now().Add(time.Duration(windowSecs) * time.Second),
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
		redis.Client.Expire(ctx, key, time.Duration(windowSecs)*time.Second)
	}

	effectiveLimit := maxRequests
	if int(count) <= burstLimit {
		result.Remaining = burstLimit - int(count) + 1
	}

	if int(count) > effectiveLimit {
		result.Allowed = false
		result.Remaining = 0
		return result, nil
	}

	result.Remaining = effectiveLimit - int(count)
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
	cfg := config.GetConfig()
	banCfg := cfg.Blacklist.BanDurations

	switch {
	case violationCount >= banCfg.Level5Count:
		return time.Duration(banCfg.Level5DurationHours) * time.Hour
	case violationCount >= banCfg.Level4Count:
		return time.Duration(banCfg.Level4DurationHours) * time.Hour
	case violationCount >= banCfg.Level3Count:
		return time.Duration(banCfg.Level3DurationHours) * time.Hour
	case violationCount >= banCfg.Level2Count:
		return time.Duration(banCfg.Level2DurationMins) * time.Minute
	case violationCount >= banCfg.Level1Count:
		return time.Duration(banCfg.Level1DurationMins) * time.Minute
	default:
		return 5 * time.Minute
	}
}

type RateLimitStrategy string

const (
	StrategyFixed    RateLimitStrategy = "fixed"
	StrategySliding RateLimitStrategy = "sliding"
	StrategyToken   RateLimitStrategy = "token"
)

func (s *RateLimitService) CheckRateLimitSliding(ctx context.Context, key string, maxRequests int, windowSecs int) (*RateLimitResult, error) {
	result := &RateLimitResult{
		Allowed:   true,
		Remaining: maxRequests - 1,
		ResetAt:   time.Now().Add(time.Duration(windowSecs) * time.Second),
	}

	if redis.Client == nil {
		return result, nil
	}

	now := time.Now().UnixMilli()
	windowStart := now - int64(windowSecs*1000)

	pipe := redis.Client.Pipeline()
	
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart))
	
	member := fmt.Sprintf("%d", now)
	pipe.ZAdd(ctx, key, goredis.Z{Score: float64(now), Member: member})
	
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, time.Duration(windowSecs)*time.Second)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to execute redis pipeline: %w", err)
	}

	count := countCmd.Val()
	
	if count >= int64(maxRequests) {
		result.Allowed = false
		result.Remaining = 0
		return result, nil
	}

	result.Remaining = maxRequests - int(count)
	return result, nil
}

func (s *RateLimitService) GetRateLimitStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"ip_limits":      0,
		"user_limits":    0,
		"app_limits":     0,
		"whitelisted":    0,
		"blacklisted":    0,
		"banned":         0,
		"violations":     0,
		"failed_attempts": 0,
	}

	if redis.Client == nil {
		return stats, nil
	}

	patterns := []string{
		PrefixIPRateLimit + "*",
		PrefixUserRateLimit + "*",
		PrefixAppRateLimit + "*",
		PrefixWhitelist + "*",
		PrefixBlacklist + "*",
		PrefixBan + "*",
		PrefixViolation + "*",
		PrefixFailedAttempts + "*",
	}

	for i, pattern := range patterns {
		var count int64
		iter := redis.Client.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			count++
		}
		
		switch i {
		case 0:
			stats["ip_limits"] = count
		case 1:
			stats["user_limits"] = count
		case 2:
			stats["app_limits"] = count
		case 3:
			stats["whitelisted"] = count
		case 4:
			stats["blacklisted"] = count
		case 5:
			stats["banned"] = count
		case 6:
			stats["violations"] = count
		case 7:
			stats["failed_attempts"] = count
		}
	}

	return stats, nil
}

func (s *RateLimitService) CleanupExpiredData(ctx context.Context) error {
	if redis.Client == nil {
		return nil
	}

	cfg := config.GetConfig()
	retentionDays := time.Duration(cfg.Blacklist.RetentionDays) * 24 * time.Hour

	patterns := []string{
		PrefixViolation + "*",
		PrefixFailedAttempts + "*",
	}

	for _, pattern := range patterns {
		var cursor uint64
		for {
			keys, newCursor, err := redis.Client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}

			for _, key := range keys {
				ttl, _ := redis.Client.TTL(ctx, key).Result()
				if ttl < 0 {
					redis.Client.Del(ctx, key)
				}
			}

			cursor = newCursor
			if cursor == 0 {
				break
			}
		}
	}

	_ = retentionDays

	return nil
}

type RiskBasedRateLimitConfig struct {
	CriticalMaxRequests int
	CriticalWindowSecs int
	HighMaxRequests    int
	HighWindowSecs     int
	MediumMaxRequests  int
	MediumWindowSecs   int
	LowMaxRequests     int
	LowWindowSecs      int
	SafeMaxRequests    int
	SafeWindowSecs     int
}

func GetRiskBasedRateLimitConfig() *RiskBasedRateLimitConfig {
	cfg := config.GetConfig()
	return &RiskBasedRateLimitConfig{
		CriticalMaxRequests: cfg.RateLimit.RiskBased.CriticalMaxRequests,
		CriticalWindowSecs: cfg.RateLimit.RiskBased.CriticalWindowSecs,
		HighMaxRequests:    cfg.RateLimit.RiskBased.HighMaxRequests,
		HighWindowSecs:    cfg.RateLimit.RiskBased.HighWindowSecs,
		MediumMaxRequests: cfg.RateLimit.RiskBased.MediumMaxRequests,
		MediumWindowSecs:  cfg.RateLimit.RiskBased.MediumWindowSecs,
		LowMaxRequests:    80,
		LowWindowSecs:    60,
		SafeMaxRequests:   100,
		SafeWindowSecs:   60,
	}
}

func (s *RateLimitService) CheckRiskBasedRateLimit(ctx context.Context, ip string, riskLevel string) (*RateLimitResult, error) {
	cfg := GetRiskBasedRateLimitConfig()

	var maxRequests, windowSecs int

	switch riskLevel {
	case "critical":
		maxRequests = cfg.CriticalMaxRequests
		windowSecs = cfg.CriticalWindowSecs
	case "high":
		maxRequests = cfg.HighMaxRequests
		windowSecs = cfg.HighWindowSecs
	case "medium":
		maxRequests = cfg.MediumMaxRequests
		windowSecs = cfg.MediumWindowSecs
	case "low":
		maxRequests = cfg.LowMaxRequests
		windowSecs = cfg.LowWindowSecs
	default:
		maxRequests = cfg.SafeMaxRequests
		windowSecs = cfg.SafeWindowSecs
	}

	key := fmt.Sprintf("%s%s:%s", PrefixIPRateLimit, riskLevel, ip)
	config := &RateLimitConfig{
		MaxRequests: maxRequests,
		WindowSecs:  windowSecs,
	}

	return s.checkRateLimit(ctx, key, config)
}
