package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository"
	"github.com/hjtpx/hjtpx/pkg/models"
	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionInvalid  = errors.New("invalid session")
)

const (
	SessionPrefix    = "admin:session:"
	SessionTTL       = 24 * time.Hour
	SessionKeyPrefix = "admin:token:"
)

type SessionInfo struct {
	SessionID string
	AdminID   uint
	Username  string
	Role      string
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionService interface {
	CreateSession(ctx context.Context, adminID uint, username, role, ip, userAgent string) (*SessionInfo, error)
	GetSession(ctx context.Context, token string) (*SessionInfo, error)
	ValidateSession(ctx context.Context, token string) (*SessionInfo, error)
	RefreshSession(ctx context.Context, token string) (*SessionInfo, error)
	InvalidateSession(ctx context.Context, token string) error
	InvalidateAllSessions(ctx context.Context, adminID uint) error
	DeleteExpiredSessions(ctx context.Context) error
}

type sessionService struct {
	adminRepo    repository.AdminRepository
	redisClient  *goredis.Client
}

func NewSessionService(adminRepo repository.AdminRepository, redisClient *goredis.Client) SessionService {
	return &sessionService{
		adminRepo:   adminRepo,
		redisClient: redisClient,
	}
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (s *sessionService) CreateSession(ctx context.Context, adminID uint, username, role, ip, userAgent string) (*SessionInfo, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(SessionTTL)

	sessionInfo := &SessionInfo{
		SessionID: sessionID,
		AdminID:   adminID,
		Username:  username,
		Role:      role,
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	sessionKey := SessionKeyPrefix + sessionID
	if s.redisClient != nil {
		sessionData := fmt.Sprintf("%d:%s:%s:%s:%s", adminID, username, role, ip, userAgent)
		err := s.redisClient.Set(ctx, sessionKey, sessionData, SessionTTL).Err()
		if err != nil {
			return nil, fmt.Errorf("failed to store session in Redis: %w", err)
		}

		adminSessionsKey := fmt.Sprintf("admin:%d:sessions", adminID)
		s.redisClient.SAdd(ctx, adminSessionsKey, sessionID)
		s.redisClient.Expire(ctx, adminSessionsKey, SessionTTL)
	}

	return sessionInfo, nil
}

func (s *sessionService) GetSession(ctx context.Context, token string) (*SessionInfo, error) {
	sessionKey := SessionKeyPrefix + token

	if s.redisClient != nil {
		data, err := s.redisClient.Get(ctx, sessionKey).Result()
		if err == goredis.Nil {
			return nil, ErrSessionNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get session from Redis: %w", err)
		}

		parts := splitSessionData(data)
		if len(parts) >= 5 {
			var parsedAdminID uint
			fmt.Sscanf(parts[0], "%d", &parsedAdminID)
			return &SessionInfo{
				SessionID: token,
				AdminID:   parsedAdminID,
				Username:  parts[1],
				Role:      parts[2],
				IPAddress: parts[3],
				UserAgent: parts[4],
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(SessionTTL),
			}, nil
		}
		return nil, ErrSessionInvalid
	}

	return nil, ErrSessionNotFound
}

func splitSessionData(data string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == ':' && (i == 0 || data[i-1] != '\\') {
			part := data[start:i]
			part = replaceEscapedChars(part)
			parts = append(parts, part)
			start = i + 1
		}
	}
	if start < len(data) {
		part := data[start:]
		part = replaceEscapedChars(part)
		parts = append(parts, part)
	}
	return parts
}

func replaceEscapedChars(s string) string {
	result := make([]byte, len(s))
	j := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		result[j] = s[i]
		j++
	}
	return string(result[:j])
}

func (s *sessionService) ValidateSession(ctx context.Context, token string) (*SessionInfo, error) {
	session, err := s.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		s.InvalidateSession(ctx, token)
		return nil, ErrSessionExpired
	}

	return session, nil
}

func (s *sessionService) RefreshSession(ctx context.Context, token string) (*SessionInfo, error) {
	session, err := s.ValidateSession(ctx, token)
	if err != nil {
		return nil, err
	}

	sessionKey := SessionKeyPrefix + token
	if s.redisClient != nil {
		err := s.redisClient.Expire(ctx, sessionKey, SessionTTL).Err()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh session: %w", err)
		}
	}

	session.ExpiresAt = time.Now().Add(SessionTTL)
	return session, nil
}

func (s *sessionService) InvalidateSession(ctx context.Context, token string) error {
	sessionKey := SessionKeyPrefix + token

	if s.redisClient != nil {
		session, err := s.GetSession(ctx, token)
		if err == nil && session != nil {
			adminSessionsKey := fmt.Sprintf("admin:%d:sessions", session.AdminID)
			s.redisClient.SRem(ctx, adminSessionsKey, token)
		}

		err = s.redisClient.Del(ctx, sessionKey).Err()
		if err != nil && err != goredis.Nil {
			return fmt.Errorf("failed to invalidate session: %w", err)
		}
	}

	return nil
}

func (s *sessionService) InvalidateAllSessions(ctx context.Context, adminID uint) error {
	if s.redisClient != nil {
		adminSessionsKey := fmt.Sprintf("admin:%d:sessions", adminID)
		sessionIDs, err := s.redisClient.SMembers(ctx, adminSessionsKey).Result()
		if err != nil && err != goredis.Nil {
			return fmt.Errorf("failed to get admin sessions: %w", err)
		}

		for _, sessionID := range sessionIDs {
			sessionKey := SessionKeyPrefix + sessionID
			s.redisClient.Del(ctx, sessionKey)
		}

		s.redisClient.Del(ctx, adminSessionsKey)
	}

	return nil
}

func (s *sessionService) DeleteExpiredSessions(ctx context.Context) error {
	return nil
}

func CreateAdminLoginLog(adminRepo repository.AdminRepository, adminID uint, ip, userAgent, status, failReason string) error {
	if adminRepo == nil {
		return nil
	}
	log := &models.AdminLoginLog{
		AdminID:    adminID,
		IPAddress:  ip,
		UserAgent:  userAgent,
		Status:     status,
		FailReason: failReason,
	}
	return adminRepo.CreateLoginLog(context.Background(), log)
}
