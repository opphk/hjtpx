package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type CSRFConfig struct {
	TokenLength     int
	TokenExpiration time.Duration
	HeaderName      string
	FormFieldName   string
	CookieName      string
	SafeMethods     []string
}

type CSRFStore interface {
	Store(token, sessionID string) error
	Verify(token, sessionID string) (bool, error)
	Delete(sessionID string) error
}

type CSRFRedisStore struct {
	expiration time.Duration
}

type CSRFMemoryStore struct {
	tokens map[string]map[string]time.Time
	mu     sync.RWMutex
}

var defaultCSRFConfig = CSRFConfig{
	TokenLength:     32,
	TokenExpiration: 1 * time.Hour,
	HeaderName:      "X-CSRF-Token",
	FormFieldName:   "csrf_token",
	CookieName:      "csrf_token",
	SafeMethods:     []string{"GET", "HEAD", "OPTIONS"},
}

func generateToken(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = byte((time.Now().UnixNano()%256 + int64(i)*17 + time.Now().UnixNano()/1000000%256) % 256)
	}
	hash := sha256.Sum256(b)
	return base64.URLEncoding.EncodeToString(hash[:])
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func NewCSRFRedisStore(expiration time.Duration) *CSRFRedisStore {
	return &CSRFRedisStore{expiration: expiration}
}

func (s *CSRFRedisStore) Store(token, sessionID string) error {
	if redis.Client == nil {
		return fmt.Errorf("redis client not available")
	}
	ctx := context.Background()
	hashedToken := hashToken(token)
	key := fmt.Sprintf("csrf:%s", sessionID)
	err := redis.Client.Set(ctx, key, hashedToken, s.expiration).Err()
	return err
}

func (s *CSRFRedisStore) Verify(token, sessionID string) (bool, error) {
	if redis.Client == nil {
		return false, fmt.Errorf("redis client not available")
	}
	ctx := context.Background()
	key := fmt.Sprintf("csrf:%s", sessionID)
	storedHash, err := redis.Client.Get(ctx, key).Result()
	if err != nil {
		return false, err
	}
	inputHash := hashToken(token)
	return storedHash == inputHash, nil
}

func (s *CSRFRedisStore) Delete(sessionID string) error {
	if redis.Client == nil {
		return nil
	}
	ctx := context.Background()
	key := fmt.Sprintf("csrf:%s", sessionID)
	return redis.Client.Del(ctx, key).Err()
}

func NewCSRFMemoryStore() *CSRFMemoryStore {
	store := &CSRFMemoryStore{
		tokens: make(map[string]map[string]time.Time),
	}
	go store.cleanupExpiredTokens()
	return store
}

func (s *CSRFMemoryStore) cleanupExpiredTokens() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for sessionID, tokenMap := range s.tokens {
			for token, expiry := range tokenMap {
				if now.After(expiry) {
					delete(tokenMap, token)
				}
			}
			if len(tokenMap) == 0 {
				delete(s.tokens, sessionID)
			}
		}
		s.mu.Unlock()
	}
}

func (s *CSRFMemoryStore) Store(token, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hashedToken := hashToken(token)
	if s.tokens[sessionID] == nil {
		s.tokens[sessionID] = make(map[string]time.Time)
	}
	s.tokens[sessionID][hashedToken] = time.Now().Add(defaultCSRFConfig.TokenExpiration)
	return nil
}

func (s *CSRFMemoryStore) Verify(token, sessionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hashedToken := hashToken(token)
	if tokenMap, ok := s.tokens[sessionID]; ok {
		if expiry, exists := tokenMap[hashedToken]; exists {
			if time.Now().Before(expiry) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *CSRFMemoryStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, sessionID)
	return nil
}

func generateSessionID(c *gin.Context) string {
	if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
		return sessionID
	}
	if sessionID := c.GetHeader("X-Forwarded-For"); sessionID != "" {
		parts := strings.Split(sessionID, ",")
		return strings.TrimSpace(parts[0]) + ":" + c.ClientIP()
	}
	return c.ClientIP() + ":" + c.GetHeader("User-Agent")
}

func isSafeMethod(method string, safeMethods []string) bool {
	for _, m := range safeMethods {
		if method == m {
			return true
		}
	}
	return false
}

func CSRFProtection(config ...CSRFConfig) gin.HandlerFunc {
	cfg := defaultCSRFConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	var store CSRFStore
	if redis.Client != nil {
		store = NewCSRFRedisStore(cfg.TokenExpiration)
	} else {
		store = NewCSRFMemoryStore()
	}

	return func(c *gin.Context) {
		method := c.Request.Method

		if isSafeMethod(method, cfg.SafeMethods) {
			if method == "GET" || method == "HEAD" {
				sessionID := generateSessionID(c)
				token := generateToken(cfg.TokenLength)

				err := store.Store(token, sessionID)
				if err == nil {
					c.Set("csrf_token", token)
					c.Set("csrf_session_id", sessionID)
					c.Header("X-CSRF-Token", token)
					c.SetCookie(
						cfg.CookieName,
						token,
						int(cfg.TokenExpiration.Seconds()),
						"/",
						"",
						false,
						true,
					)
				}
			}
			c.Next()
			return
		}

		sessionID := generateSessionID(c)

		var token string
		token = c.GetHeader(cfg.HeaderName)
		if token == "" {
			token = c.Query(cfg.FormFieldName)
		}
		if token == "" {
			token = c.PostForm(cfg.FormFieldName)
		}
		if token == "" {
			cookieToken, err := c.Cookie(cfg.CookieName)
			if err == nil {
				token = cookieToken
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf token missing",
				"message": "CSRF token is required for this request",
			})
			return
		}

		valid, err := store.Verify(token, sessionID)
		if err != nil || !valid {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf token invalid",
				"message": "Invalid or expired CSRF token",
			})
			return
		}

		err = store.Delete(sessionID)
		if err != nil {
			fmt.Printf("[CSRF] Warning: failed to delete used token: %v\n", err)
		}

		c.Next()
	}
}

func GetCSRFToken(c *gin.Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		return token.(string)
	}
	return ""
}
