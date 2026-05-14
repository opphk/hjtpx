package risk

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type IPLimit struct {
	client         *redis.Client
	keyPrefix      string
	maxFailures    int
	criticalFailures int
	blockDuration  time.Duration
}

type IPLimitConfig struct {
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	KeyPrefix      string
	MaxFailures    int
	CriticalFailures int
	BlockDuration  time.Duration
}

func NewIPLimit(cfg *IPLimitConfig) (*IPLimit, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}

	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "captchax:ip:"
	}
	if cfg.MaxFailures == 0 {
		cfg.MaxFailures = 3
	}
	if cfg.CriticalFailures == 0 {
		cfg.CriticalFailures = 5
	}
	if cfg.BlockDuration == 0 {
		cfg.BlockDuration = 30 * time.Minute
	}

	return &IPLimit{
		client:         client,
		keyPrefix:      cfg.KeyPrefix,
		maxFailures:    cfg.MaxFailures,
		criticalFailures: cfg.CriticalFailures,
		blockDuration:  cfg.BlockDuration,
	}, nil
}

func (l *IPLimit) RecordAccess(ctx context.Context, ip string) error {
	key := l.keyPrefix + "access:" + ip

	pipe := l.client.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Hour)
	_, err := pipe.Exec(ctx)

	return err
}

func (l *IPLimit) RecordFailure(ctx context.Context, ip string) error {
	failKey := l.keyPrefix + "fail:" + ip
	countKey := l.keyPrefix + "count:" + ip

	pipe := l.client.Pipeline()

	incr := pipe.Incr(ctx, failKey)
	pipe.Expire(ctx, failKey, l.blockDuration)

	pipe.Set(ctx, countKey, incr.Val(), l.blockDuration)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	count, err := l.GetFailureCount(ctx, ip)
	if err != nil {
		return err
	}

	if count >= int64(l.criticalFailures) {
		return l.AddToBlacklist(ctx, ip)
	}

	return nil
}

func (l *IPLimit) RecordSuccess(ctx context.Context, ip string) error {
	pipe := l.client.Pipeline()

	pipe.Del(ctx, l.keyPrefix+"fail:"+ip)
	pipe.Del(ctx, l.keyPrefix+"access:"+ip)

	_, err := pipe.Exec(ctx)
	return err
}

func (l *IPLimit) GetAccessCount(ctx context.Context, ip string) (int64, error) {
	key := l.keyPrefix + "access:" + ip
	val, err := l.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	count, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (l *IPLimit) GetFailureCount(ctx context.Context, ip string) (int64, error) {
	key := l.keyPrefix + "fail:" + ip
	val, err := l.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	count, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (l *IPLimit) IsBlacklisted(ctx context.Context, ip string) (bool, error) {
	key := l.keyPrefix + "blacklist:" + ip
	exists, err := l.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (l *IPLimit) AddToBlacklist(ctx context.Context, ip string) error {
	key := l.keyPrefix + "blacklist:" + ip
	return l.client.Set(ctx, key, "1", l.blockDuration*2).Err()
}

func (l *IPLimit) RemoveFromBlacklist(ctx context.Context, ip string) error {
	key := l.keyPrefix + "blacklist:" + ip
	return l.client.Del(ctx, key).Err()
}

func (l *IPLimit) CheckIPRisk(ctx context.Context, ip string) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	isBlacklisted, err := l.IsBlacklisted(ctx, ip)
	if err == nil && isBlacklisted {
		score += 100
		factors = append(factors, RiskFactor{
			Name:   "ip_blacklisted",
			Weight: 100,
			Reason: "IP地址已在黑名单中",
		})
		return score, factors
	}

	failureCount, err := l.GetFailureCount(ctx, ip)
	if err == nil {
		if failureCount >= int64(l.criticalFailures) {
			score += 30
			factors = append(factors, RiskFactor{
				Name:   "excessive_failures_critical",
				Weight: 30,
				Reason: fmt.Sprintf("失败次数达到%d次，疑似恶意攻击", l.criticalFailures),
			})
		} else if failureCount >= int64(l.maxFailures) {
			score += 30
			factors = append(factors, RiskFactor{
				Name:   "excessive_failures",
				Weight: 30,
				Reason: fmt.Sprintf("失败次数达到%d次，风险升高", l.maxFailures),
			})
		}
	}

	accessCount, err := l.GetAccessCount(ctx, ip)
	if err == nil && accessCount > 100 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "high_frequency_access",
			Weight: 15,
			Reason: fmt.Sprintf("IP访问频率异常(%d次/小时)", accessCount),
		})
	}

	return score, factors
}

func (l *IPLimit) GetIPStats(ctx context.Context, ip string) (*IPStats, error) {
	failures, _ := l.GetFailureCount(ctx, ip)
	accesses, _ := l.GetAccessCount(ctx, ip)
	blacklisted, _ := l.IsBlacklisted(ctx, ip)

	return &IPStats{
		IP:           ip,
		Failures:     failures,
		AccessCount:  accesses,
		Blacklisted:  blacklisted,
		LastAccessed: time.Now(),
	}, nil
}

type IPStats struct {
	IP           string
	Failures     int64
	AccessCount  int64
	Blacklisted  bool
	LastAccessed time.Time
}

func (l *IPLimit) Close() error {
	return l.client.Close()
}
