package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	goredis "github.com/redis/go-redis/v9"
)

const (
	CaptchaSessionPrefix = "captcha:session:"
	DefaultTTL          = 5 * time.Minute
)

type SessionCache struct {
	client *goredis.Client
}

func NewSessionCache() *SessionCache {
	return &SessionCache{
		client: nil,
	}
}

func (s *SessionCache) getKey(sessionID string) string {
	return CaptchaSessionPrefix + sessionID
}

func (s *SessionCache) Set(ctx context.Context, session *models.CaptchaSession) error {
	if s.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := s.getKey(session.SessionID)
	ttl := time.Until(session.ExpiredAt)
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	return s.client.Set(ctx, key, string(data), ttl).Err()
}

func (s *SessionCache) Get(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	key := s.getKey(sessionID)
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var session models.CaptchaSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *SessionCache) Delete(ctx context.Context, sessionID string) error {
	if s.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := s.getKey(sessionID)
	_, err := s.client.Del(ctx, key).Result()
	return err
}

func (s *SessionCache) Exists(ctx context.Context, sessionID string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("redis client not initialized")
	}

	key := s.getKey(sessionID)
	count, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SessionCache) Expire(ctx context.Context, sessionID string, ttl time.Duration) error {
	if s.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := s.getKey(sessionID)
	return s.client.Expire(ctx, key, ttl).Err()
}

func (s *SessionCache) TTL(ctx context.Context, sessionID string) (time.Duration, error) {
	if s.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	key := s.getKey(sessionID)
	return s.client.TTL(ctx, key).Result()
}

func (s *SessionCache) UpdateStatus(ctx context.Context, sessionID string, status string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.Status = status

	if status == "verified" {
		now := time.Now()
		session.VerifiedAt = &now
	}

	return s.Set(ctx, session)
}

func (s *SessionCache) IncrementVerifyCount(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.VerifyCount++

	ttl, err := s.TTL(ctx, sessionID)
	if err != nil || ttl <= 0 {
		ttl = DefaultTTL
	}

	return s.Set(ctx, session)
}

func (s *SessionCache) UpdateRiskScores(ctx context.Context, sessionID string, riskScore, traceScore, envScore float64) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.RiskScore = riskScore
	session.TraceScore = traceScore
	session.EnvScore = envScore

	return s.Set(ctx, session)
}

func (s *SessionCache) SetWithTTL(ctx context.Context, session *models.CaptchaSession, ttl time.Duration) error {
	if s.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := s.getKey(session.SessionID)
	return s.client.Set(ctx, key, string(data), ttl).Err()
}

func (s *SessionCache) CleanupExpired(ctx context.Context) (int64, error) {
	if s.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	pattern := CaptchaSessionPrefix + "*"
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, err
	}

	var deletedCount int64
	for _, key := range keys {
		ttl, err := s.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		if ttl <= 0 {
			_, err := s.client.Del(ctx, key).Result()
			if err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}
