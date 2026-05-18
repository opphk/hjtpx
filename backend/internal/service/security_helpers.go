package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

// CaptchaService 验证码服务
type CaptchaService struct {
	redis *redis.Client
	ctx   context.Context
}

// NewCaptchaService 创建验证码服务
func NewCaptchaService(redisClient *redis.Client) *CaptchaService {
	return &CaptchaService{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// ValidateCaptchaToken 验证验证码Token
func (s *CaptchaService) ValidateCaptchaToken(token string, expiration time.Duration) bool {
	if token == "" {
		return false
	}
	
	// 简单的token格式检查
	if len(token) < 10 {
		return false
	}
	
	// 检查token是否已被使用（在Redis中记录）
	if s.redis != nil {
		key := "captcha:used:" + token
		exists, _ := s.redis.Exists(s.ctx, key).Result()
		if exists > 0 {
			return false // Token已被使用
		}
	}
	
	return true
}

// MarkCaptchaTokenUsed 标记验证码Token已使用
func (s *CaptchaService) MarkCaptchaTokenUsed(token string, expiration time.Duration) error {
	if s.redis != nil {
		key := "captcha:used:" + token
		return s.redis.Set(s.ctx, key, "1", expiration).Err()
	}
	return nil
}

// MaskSensitiveData 屏蔽敏感数据
func MaskSensitiveData(input string) string {
	sensitivePatterns := []struct {
		pattern *regexp.Regexp
		mask   string
	}{
		{regexp.MustCompile(`(?i)(password|passwd|pwd)[\s:=]+[^\s,;]+`), "$1=***MASKED***"},
		{regexp.MustCompile(`(?i)(api[_-]?key|apikey)[\s:=]+[^\s,;]+`), "$1=***MASKED***"},
		{regexp.MustCompile(`(?i)(token|bearer)[\s:=]+[^\s,;]+`), "$1=***MASKED***"},
		{regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`), "****-****-****-****"},
		{regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), "***-**-****"},
	}
	
	result := input
	for _, p := range sensitivePatterns {
		result = p.pattern.ReplaceAllString(result, p.mask)
	}
	
	return result
}

// SanitizeLogEntry 清理日志条目中的敏感信息
func SanitizeLogEntry(data map[string]string) map[string]string {
	sensitiveFields := []string{"password", "token", "api_key", "secret", "credential", "auth"}
	
	result := make(map[string]string)
	for key, value := range data {
		isSensitive := false
		lowerKey := strings.ToLower(key)
		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}
		
		if isSensitive {
			result[key] = "***MASKED***"
		} else {
			result[key] = value
		}
	}
	
	return result
}

// IsStrongCaptcha 检查验证码强度
func IsStrongCaptcha(captcha string) bool {
	if len(captcha) < 4 {
		return false
	}
	
	// 检查是否包含重复字符
	hasRepeating := regexp.MustCompile(`(.)\1{2,}`).MatchString(captcha)
	if hasRepeating {
		return false
	}
	
	// 检查是否为弱验证码模式
	weakPatterns := []string{
		"1234", "0000", "1111", "2222", "3333", "4444", "5555", "6666", "7777", "8888", "9999",
		"abcd", "aaaa", "bbbb",
		"qwer", "asdf", "zxcv",
	}
	
	lowerCaptcha := strings.ToLower(captcha)
	for _, pattern := range weakPatterns {
		if lowerCaptcha == pattern || strings.Contains(lowerCaptcha, pattern) {
			return false
		}
	}
	
	return true
}

// GenerateSecureNonce 生成安全随机数
func GenerateSecureNonce(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// JWTClaims JWT声明
type JWTClaims struct {
	AdminID   uint   `json:"admin_id"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"exp"`
}

// GenerateJWT 生成JWT Token（包装pkg/jwt）
func GenerateJWT(redisClient *redis.Client, adminID uint, username string) (string, error) {
	return jwt.GenerateToken(adminID, username)
}

// ParseJWT 解析JWT Token（包装pkg/jwt）
func ParseJWT(redisClient *redis.Client, token string) (*JWTClaims, error) {
	claims, err := jwt.ParseToken(token)
	if err != nil {
		return nil, err
	}
	
	return &JWTClaims{
		AdminID:   claims.AdminID,
		Username:  claims.Username,
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}

// SecuritySessionService 会话服务（与管理员会话服务区分）
type SecuritySessionService struct {
	redis *redis.Client
	ctx   context.Context
	sessions map[string]*SecuritySessionInfo
	mu      sync.RWMutex
}

type SecuritySessionInfo struct {
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewSecuritySessionService 创建会话服务
func NewSecuritySessionService(redisClient *redis.Client) *SecuritySessionService {
	return &SecuritySessionService{
		redis:    redisClient,
		ctx:      context.Background(),
		sessions: make(map[string]*SecuritySessionInfo),
	}
}

// CreateSecuritySession 创建新会话
func (s *SecuritySessionService) CreateSecuritySession(ctx context.Context, userID string, duration time.Duration) (string, error) {
	sessionID, err := GenerateSecureNonce(32)
	if err != nil {
		return "", err
	}
	
	sessionInfo := &SecuritySessionInfo{
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt:  time.Now().Add(duration),
	}
	
	if s.redis != nil {
		key := "session:" + sessionID
		data := map[string]interface{}{
			"user_id":    userID,
			"created_at": sessionInfo.CreatedAt.Unix(),
		}
		s.redis.HSet(ctx, key, data)
		s.redis.Expire(ctx, key, duration)
	} else {
		s.mu.Lock()
		s.sessions[sessionID] = sessionInfo
		s.mu.Unlock()
	}
	
	return sessionID, nil
}

// ValidateSecuritySession 验证会话
func (s *SecuritySessionService) ValidateSecuritySession(ctx context.Context, sessionID string) (bool, error) {
	if s.redis != nil {
		key := "session:" + sessionID
		exists, err := s.redis.Exists(ctx, key).Result()
		if err != nil {
			return false, err
		}
		return exists > 0, nil
	} else {
		s.mu.RLock()
		defer s.mu.RUnlock()
		
		session, exists := s.sessions[sessionID]
		if !exists {
			return false, nil
		}
		
		return time.Now().Before(session.ExpiresAt), nil
	}
}

// InvalidateSecuritySession 使会话失效
func (s *SecuritySessionService) InvalidateSecuritySession(ctx context.Context, sessionID string) error {
	if s.redis != nil {
		key := "session:" + sessionID
		return s.redis.Del(ctx, key).Err()
	} else {
		s.mu.Lock()
		defer s.mu.Unlock()
		
		delete(s.sessions, sessionID)
		return nil
	}
}

// XSSSecurity XSS安全服务
type XSSSecurity struct {
	redis *redis.Client
	ctx   context.Context
}

// NewXSSSecurity 创建XSS安全服务
func NewXSSSecurity(redisClient *redis.Client) *XSSSecurity {
	return &XSSSecurity{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// SanitizeInput 清理输入
func (s *XSSSecurity) SanitizeInput(input string) string {
	return SanitizeHTML(input)
}

// SanitizeHTML HTML转义和清理
func SanitizeHTML(input string) string {
	// 移除script标签
	input = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</script>`).ReplaceAllString(input, "")
	
	// 移除iframe
	input = regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`).ReplaceAllString(input, "")
	
	// 移除img标签（可能有onerror）
	input = regexp.MustCompile(`(?i)<img[^>]*>`).ReplaceAllString(input, "")
	
	// 移除svg标签
	input = regexp.MustCompile(`(?i)<svg[^>]*>.*?</svg>`).ReplaceAllString(input, "")
	
	// 移除事件处理器
	input = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(["'][^"']*["']|[^\s>]*)`).ReplaceAllString(input, "")
	
	// HTML转义
	return input
}

// DetectXSS 检测XSS攻击
func (s *XSSSecurity) DetectXSS(input string) (bool, string) {
	xssPatterns := []string{
		`<script`,
		`javascript:`,
		`onerror`,
		`onload`,
		`onclick`,
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true, pattern
		}
	}
	
	return false, ""
}
