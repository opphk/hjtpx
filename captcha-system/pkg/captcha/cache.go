package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheService struct {
	client *redis.Client
	prefix string
}

type CaptchaCacheData struct {
	ChallengeID string          `json:"challenge_id"`
	Type        string          `json:"type"`
	Data        json.RawMessage `json:"data"`
	Solution    json.RawMessage `json:"solution"`
	Difficulty  string          `json:"difficulty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type SessionCacheData struct {
	SessionID   string    `json:"session_id"`
	Fingerprint string    `json:"fingerprint"`
	IPAddress   string    `json:"ip_address"`
	Attempts    int       `json:"attempts"`
	Blocked     bool      `json:"blocked"`
	BlockedUntil time.Time `json:"blocked_until,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{
		client: client,
		prefix: "captcha:",
	}
}

func (s *CacheService) SetChallenge(ctx context.Context, challengeID string, data *CaptchaCacheData, expiration time.Duration) error {
	key := s.prefix + "challenge:" + challengeID
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal challenge data: %w", err)
	}
	return s.client.Set(ctx, key, jsonData, expiration).Err()
}

func (s *CacheService) GetChallenge(ctx context.Context, challengeID string) (*CaptchaCacheData, error) {
	key := s.prefix + "challenge:" + challengeID
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	var result CaptchaCacheData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal challenge data: %w", err)
	}
	return &result, nil
}

func (s *CacheService) DeleteChallenge(ctx context.Context, challengeID string) error {
	key := s.prefix + "challenge:" + challengeID
	return s.client.Del(ctx, key).Err()
}

func (s *CacheService) SetSession(ctx context.Context, sessionID string, data *SessionCacheData, expiration time.Duration) error {
	key := s.prefix + "session:" + sessionID
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}
	return s.client.Set(ctx, key, jsonData, expiration).Err()
}

func (s *CacheService) GetSession(ctx context.Context, sessionID string) (*SessionCacheData, error) {
	key := s.prefix + "session:" + sessionID
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var result SessionCacheData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}
	return &result, nil
}

func (s *CacheService) IncrementSessionAttempts(ctx context.Context, sessionID string) (int64, error) {
	key := s.prefix + "session:" + sessionID
	return s.client.HIncrBy(ctx, key, "attempts", 1).Result()
}

func (s *CacheService) SetRateLimit(ctx context.Context, key string, expiration time.Duration) error {
	limitKey := s.prefix + "ratelimit:" + key
	return s.client.Set(ctx, limitKey, "1", expiration).Err()
}

func (s *CacheService) CheckRateLimit(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, error) {
	limitKey := s.prefix + "ratelimit:" + key
	count, err := s.client.Incr(ctx, limitKey).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		s.client.Expire(ctx, limitKey, window)
	}

	return count <= int64(maxRequests), nil
}

func (s *CacheService) Close() error {
	return s.client.Close()
}
