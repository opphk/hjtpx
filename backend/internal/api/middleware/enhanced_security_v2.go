package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha3"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SecurityV2Config struct {
	SecretKey          string
	Algorithm          string
	ReplayWindowSecs   int64
	NonceCacheSize     int
	TokenBucketRate    float64
	TokenBucketBurst   int
	TokenBucketRefillMs int64
	EnableReplay       bool
	EnableNonce        bool
	EnableRateLimit    bool
}

type EnhancedSecurityV2 struct {
	config     *SecurityV2Config
	nonceCache *nonceCache
	rateLimit  map[string]*tokenBucket
	mu         sync.RWMutex
}

type nonceCache struct {
	data map[string]bool
	mu   sync.RWMutex
}

func newNonceCache(size int) *nonceCache {
	return &nonceCache{
		data: make(map[string]bool, size),
	}
}

func (nc *nonceCache) get(nonce string) (bool, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	exists, ok := nc.data[nonce]
	return exists, ok
}

func (nc *nonceCache) set(nonce string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if len(nc.data) > 10000 {
		nc.data = make(map[string]bool)
	}
	nc.data[nonce] = true
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewEnhancedSecurityV2(config *SecurityV2Config) *EnhancedSecurityV2 {
	if config == nil {
		config = &SecurityV2Config{
			SecretKey:          "default-secret-key-change-in-production",
			Algorithm:          "HMAC-SHA3-256",
			ReplayWindowSecs:   300,
			NonceCacheSize:     10000,
			TokenBucketRate:    100,
			TokenBucketBurst:   200,
			TokenBucketRefillMs: 100,
			EnableReplay:       true,
			EnableNonce:        true,
			EnableRateLimit:    true,
		}
	}
	return &EnhancedSecurityV2{
		config:     config,
		nonceCache: newNonceCache(config.NonceCacheSize),
		rateLimit:  make(map[string]*tokenBucket),
	}
}

func (s *EnhancedSecurityV2) GenerateSignature(ctx context.Context, req *Request) (string, error) {
	stringToSign := s.buildStringToSign(req)

	hmacHash := hmac.New(func() hash.Hash { return sha3.New256() }, []byte(s.config.SecretKey))
	hmacHash.Write([]byte(stringToSign))
	return hex.EncodeToString(hmacHash.Sum(nil)), nil
}

func (s *EnhancedSecurityV2) buildStringToSign(req *Request) string {
	parts := []string{
		req.Method,
		req.Path,
		req.Timestamp,
		req.Nonce,
	}
	if req.Body != "" {
		parts = append(parts, req.Body)
	}
	return strings.Join(parts, "\n")
}

func (s *EnhancedSecurityV2) VerifySignature(ctx context.Context, req *Request, signature string) (bool, error) {
	expectedSig, err := s.GenerateSignature(ctx, req)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(expectedSig), []byte(signature)) == 1, nil
}

func (s *EnhancedSecurityV2) EnhancedReplayProtection(ctx context.Context, nonce string, timestamp int64) bool {
	if !s.config.EnableReplay {
		return true
	}

	now := time.Now().Unix()
	diff := math.Abs(float64(now - timestamp))
	if diff > float64(s.config.ReplayWindowSecs) {
		return false
	}

	return true
}

func (s *EnhancedSecurityV2) TokenBucketRateLimit(ctx context.Context, key string, rate float64, burst int) (bool, error) {
	if !s.config.EnableRateLimit {
		return true, nil
	}

	if rate <= 0 {
		rate = s.config.TokenBucketRate
	}
	if burst <= 0 {
		burst = s.config.TokenBucketBurst
	}

	s.mu.Lock()
	tb, exists := s.rateLimit[key]
	if !exists {
		tb = &tokenBucket{
			tokens:     float64(burst),
			lastRefill: time.Now(),
		}
		s.rateLimit[key] = tb
	}
	s.mu.Unlock()

	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Milliseconds()
	refillAmount := float64(elapsed) / float64(s.config.TokenBucketRefillMs) * rate
	tb.tokens = math.Min(float64(burst), tb.tokens+refillAmount)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true, nil
	}

	return false, nil
}

func (s *EnhancedSecurityV2) VersionRouting(ctx context.Context, path string) (string, error) {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	for _, part := range pathParts {
		if strings.HasPrefix(part, "v") {
			v := strings.TrimPrefix(part, "v")
			if _, err := strconv.Atoi(v); err == nil {
				return "v" + v, nil
			}
		}
	}
	return "v1", nil
}

func (s *EnhancedSecurityV2) GenerateNonce() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *EnhancedSecurityV2) GenerateTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func EnhancedSecurityMiddleware(security *EnhancedSecurityV2) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !security.config.EnableReplay && !security.config.EnableNonce && !security.config.EnableRateLimit {
			c.Next()
			return
		}

		_ = c.GetHeader("X-Signature")
		timestampStr := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")

		if timestampStr != "" {
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err == nil {
				if !security.EnhancedReplayProtection(c.Request.Context(), nonce, timestamp) {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "request expired or replay detected",
					})
					return
				}
			}
		}

		if nonce != "" {
			if exists, _ := security.nonceCache.get(nonce); exists {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "nonce already used",
				})
				return
			}
			security.nonceCache.set(nonce)
		}

		clientIP := c.ClientIP()
		allowed, err := security.TokenBucketRateLimit(c.Request.Context(), "ip:"+clientIP, 0, 0)
		if err == nil && !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

type Request struct {
	Method    string
	Path      string
	Body      string
	Timestamp string
	Nonce     string
}

type RateLimitResult struct {
	Allowed    bool
	Remaining  float64
	ResetAt    time.Time
	RetryAfter time.Duration
}

func (s *EnhancedSecurityV2) CheckRateLimit(ctx context.Context, key string) *RateLimitResult {
	result := &RateLimitResult{
		Allowed:   true,
		Remaining: float64(s.config.TokenBucketBurst),
		ResetAt:   time.Now(),
	}

	allowed, err := s.TokenBucketRateLimit(ctx, key, 0, 0)
	if err != nil {
		result.Allowed = true
		return result
	}

	result.Allowed = allowed
	if !allowed {
		result.Remaining = 0
		result.RetryAfter = time.Duration(s.config.TokenBucketRefillMs) * time.Millisecond
		result.ResetAt = time.Now().Add(result.RetryAfter)
	}

	return result
}

func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func HashPassword(password string) (string, error) {
	h := sha3.New256()
	h.Write([]byte(password))
	hash := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash), nil
}

func VerifyPassword(password, hash string) bool {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(passwordHash), []byte(hash)) == 1
}

type SecurityHeaders struct{}

func NewSecurityHeaders() *SecurityHeaders {
	return &SecurityHeaders{}
}

func (sh *SecurityHeaders) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}
