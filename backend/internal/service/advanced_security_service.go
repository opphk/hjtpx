package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hjtpx/hjtpx/pkg/crypto"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken2    = errors.New("invalid token")
	ErrTokenExpired2    = errors.New("token expired")
	ErrPasswordTooWeak  = errors.New("password is too weak")
	ErrInvalidCSRFToken = errors.New("invalid CSRF token")
)

type SecurityService struct {
	redis         *redis.Client
	ctx           context.Context
	inputPatterns map[string]*regexp.Regexp
	xssPatterns   []*regexp.Regexp
}

func NewSecurityService(redisClient *redis.Client) *SecurityService {
	return &SecurityService{
		redis: redisClient,
		ctx:   context.Background(),
		inputPatterns: map[string]*regexp.Regexp{
			"sql_keywords": regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|onerror|onload)`),
			"sql_chars":    regexp.MustCompile(`(?i)(['";\\]|(\-\-)|(\/\*))`),
			"cmd_chars":    regexp.MustCompile(`(?i)(\|\||&&|;|` + "`" + `|\$\(|\$\{)`),
		},
		xssPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(<script[^>]*>.*?</script>)`),
			regexp.MustCompile(`(?i)(javascript:|on\w+\s*=)`),
			regexp.MustCompile(`(?i)(<iframe|<img|<svg|<link|<meta)`),
		},
	}
}

func (s *SecurityService) SanitizeInput(input string) string {
	sanitized := input
	for _, pattern := range s.inputPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, "")
	}
	return sanitized
}

func (s *SecurityService) SanitizeHTML(input string) string {
	sanitized := input

	sanitized = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)</script>`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)<img[^>]*>`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)<svg[^>]*>.*?</svg>`).ReplaceAllString(sanitized, "")

	sanitized = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(["'][^"']*["']|[^\s>]*)`).ReplaceAllString(sanitized, "")

	sanitized = html.EscapeString(sanitized)

	sanitized = template.HTMLEscapeString(sanitized)

	return sanitized
}

func (s *SecurityService) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters", ErrPasswordTooWeak)
	}

	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("%w: password must contain at least one uppercase letter", ErrPasswordTooWeak)
	}

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("%w: password must contain at least one lowercase letter", ErrPasswordTooWeak)
	}

	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("%w: password must contain at least one digit", ErrPasswordTooWeak)
	}

	if !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
		return fmt.Errorf("%w: password must contain at least one special character", ErrPasswordTooWeak)
	}

	commonPasswords := []string{"password", "12345678", "qwerty", "admin", "letmein", "welcome"}
	lowerPassword := strings.ToLower(password)
	for _, common := range commonPasswords {
		if strings.Contains(lowerPassword, common) {
			return fmt.Errorf("%w: password contains common pattern", ErrPasswordTooWeak)
		}
	}

	return nil
}

func (s *SecurityService) HashPassword(password string) (string, error) {
	err := s.ValidatePasswordStrength(password)
	if err != nil {
		return "", err
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (s *SecurityService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *SecurityService) UseParameterizedQueries() bool {
	return true
}

type ConfigEncryptor struct {
	key []byte
}

func NewConfigEncryptor(key string) *ConfigEncryptor {
	hash := sha256.Sum256([]byte(key))
	return &ConfigEncryptor{key: hash[:]}
}

func (e *ConfigEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *ConfigEncryptor) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

type JWTSecurity struct {
	redis     *redis.Client
	ctx       context.Context
	jwtSecret []byte
}

func NewJWTSecurity(redisClient *redis.Client, secret string) *JWTSecurity {
	return &JWTSecurity{
		redis:     redisClient,
		ctx:       context.Background(),
		jwtSecret: []byte(secret),
	}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (s *JWTSecurity) CreateTokenPair(userID int64) (*TokenPair, error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     now.Add(15 * time.Minute).Unix(),
		"iat":     now.Unix(),
		"type":    "access",
	})

	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     now.Add(7 * 24 * time.Hour).Unix(),
		"iat":     now.Unix(),
		"type":    "refresh",
		"jti":     generateSecureToken(),
	})

	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	if s.redis != nil {
		key := fmt.Sprintf("refresh_token:%d", userID)
		s.redis.Set(s.ctx, key, refreshTokenString, 7*24*time.Hour)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}, nil
}

func (s *JWTSecurity) RefreshTokenPair(refreshTokenString string) (*TokenPair, error) {
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken2, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken2
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken2
	}

	if claims["type"] != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid user_id in token")
	}
	userID := int64(userIDFloat)

	if s.redis != nil {
		storedToken, err := s.redis.Get(s.ctx, fmt.Sprintf("refresh_token:%d", userID)).Result()
		if err != nil || storedToken != refreshTokenString {
			return nil, ErrInvalidToken2
		}
	}

	return s.CreateTokenPair(userID)
}

func (s *JWTSecurity) RevokeToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return ErrInvalidToken2
	}

	claims := token.Claims.(jwt.MapClaims)
	exp := int64(claims["exp"].(float64))
	remaining := time.Until(time.Unix(exp, 0))

	if remaining > 0 && s.redis != nil {
		blacklistKey := fmt.Sprintf("blacklist:%s", crypto.HashSHA256([]byte(tokenString)))
		s.redis.Set(s.ctx, blacklistKey, "1", remaining)
	}

	return nil
}

func (s *JWTSecurity) IsTokenBlacklisted(tokenString string) bool {
	if s.redis == nil {
		return false
	}

	blacklistKey := fmt.Sprintf("blacklist:%s", crypto.HashSHA256([]byte(tokenString)))
	exists, err := s.redis.Exists(s.ctx, blacklistKey).Result()
	return err == nil && exists > 0
}

func (s *JWTSecurity) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	if s.IsTokenBlacklisted(tokenString) {
		return nil, ErrInvalidToken2
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return nil, ErrTokenExpired2
		}
		return nil, ErrInvalidToken2
	}

	if !token.Valid {
		return nil, ErrInvalidToken2
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken2
	}

	return &claims, nil
}

type CSRFSecurity struct {
	redis      *redis.Client
	ctx        context.Context
	expiration time.Duration
}

func NewCSRFSecurity(redisClient *redis.Client) *CSRFSecurity {
	return &CSRFSecurity{
		redis:      redisClient,
		ctx:        context.Background(),
		expiration: 24 * time.Hour,
	}
}

func (s *CSRFSecurity) GenerateToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	tokenStr := base64.URLEncoding.EncodeToString(token)

	if s.redis != nil {
		hashedToken := crypto.HashSHA256([]byte(tokenStr))
		key := fmt.Sprintf("csrf:%s", hashedToken)
		s.redis.Set(s.ctx, key, "1", s.expiration)
	}

	return tokenStr, nil
}

func (s *CSRFSecurity) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	if s.redis == nil {
		return false
	}

	hashedToken := crypto.HashSHA256([]byte(token))
	key := fmt.Sprintf("csrf:%s", hashedToken)

	exists, err := s.redis.Exists(s.ctx, key).Result()
	if err != nil || exists == 0 {
		return false
	}

	s.redis.Del(s.ctx, key)
	return true
}

type RequestValidator struct {
	validators map[string]*regexp.Regexp
}

func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		validators: map[string]*regexp.Regexp{
			"email": regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`),
			"phone": regexp.MustCompile(`^1[3-9]\d{9}$`),
			"ip":    regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`),
			"url":   regexp.MustCompile(`^https?://`),
			"uuid":  regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
		},
	}
}

func (v *RequestValidator) Validate(field, value, rule string) error {
	if rule == "required" && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}

	if value == "" {
		return nil
	}

	switch rule {
	case "email":
		if !v.validators["email"].MatchString(value) {
			return fmt.Errorf("%s must be a valid email address", field)
		}
	case "phone":
		if !v.validators["phone"].MatchString(value) {
			return fmt.Errorf("%s must be a valid phone number", field)
		}
	case "ip":
		if !v.validators["ip"].MatchString(value) {
			return fmt.Errorf("%s must be a valid IP address", field)
		}
	case "url":
		if !v.validators["url"].MatchString(value) {
			return fmt.Errorf("%s must be a valid URL", field)
		}
	case "uuid":
		if !v.validators["uuid"].MatchString(value) {
			return fmt.Errorf("%s must be a valid UUID", field)
		}
	case "numeric":
		if !regexp.MustCompile(`^\d+$`).MatchString(value) {
			return fmt.Errorf("%s must be numeric", field)
		}
	case "alpha":
		if !regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(value) {
			return fmt.Errorf("%s must contain only letters", field)
		}
	case "alphanumeric":
		if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(value) {
			return fmt.Errorf("%s must contain only letters and numbers", field)
		}
	}

	return nil
}

func (v *RequestValidator) ValidateMap(data map[string]string, rules map[string]string) map[string]string {
	errors := make(map[string]string)

	for field, rule := range rules {
		value, exists := data[field]
		if !exists {
			value = ""
		}

		if err := v.Validate(field, value, rule); err != nil {
			errors[field] = err.Error()
		}
	}

	return errors
}

type SecurityConfig struct {
	EnableHTTPSRedirect          bool
	EnableSecurityHeaders        bool
	EnableCSRFProtection         bool
	EnableRateLimit              bool
	EnableInputValidation        bool
	EnableSQLInjectionProtection bool
	EnableXSSProtection          bool
}

var DefaultSecurityConfig = SecurityConfig{
	EnableHTTPSRedirect:          true,
	EnableSecurityHeaders:        true,
	EnableCSRFProtection:         true,
	EnableRateLimit:              true,
	EnableInputValidation:        true,
	EnableSQLInjectionProtection: true,
	EnableXSSProtection:          true,
}

type SecurityPolicy struct {
	MaxLoginAttempts       int
	LockoutDuration        time.Duration
	PasswordMinLength      int
	PasswordRequireUpper   bool
	PasswordRequireLower   bool
	PasswordRequireDigit   bool
	PasswordRequireSpecial bool
	SessionTimeout         time.Duration
	CSRFTokenExpiration    time.Duration
}

var DefaultSecurityPolicy = SecurityPolicy{
	MaxLoginAttempts:       5,
	LockoutDuration:        15 * time.Minute,
	PasswordMinLength:      8,
	PasswordRequireUpper:   true,
	PasswordRequireLower:   true,
	PasswordRequireDigit:   true,
	PasswordRequireSpecial: true,
	SessionTimeout:         24 * time.Hour,
	CSRFTokenExpiration:    24 * time.Hour,
}

type SecurityMetrics struct {
	TotalRequests        int64 `json:"total_requests"`
	BlockedRequests      int64 `json:"blocked_requests"`
	SQLInjectionAttempts int64 `json:"sql_injection_attempts"`
	XSSAttempts          int64 `json:"xss_attempts"`
	CSRFAttempts         int64 `json:"csrf_attempts"`
	RateLimitedRequests  int64 `json:"rate_limited_requests"`
}

var securityMetrics = &SecurityMetrics{}

func (s *SecurityService) IncrementMetric(name string) {
	switch name {
	case "total":
		securityMetrics.TotalRequests++
	case "blocked":
		securityMetrics.BlockedRequests++
	case "sql_injection":
		securityMetrics.SQLInjectionAttempts++
	case "xss":
		securityMetrics.XSSAttempts++
	case "csrf":
		securityMetrics.CSRFAttempts++
	case "rate_limited":
		securityMetrics.RateLimitedRequests++
	}
}

func GetSecurityMetrics() *SecurityMetrics {
	return securityMetrics
}

func generateSecureToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type EncryptedConfig struct {
	Version     int    `json:"v"`
	Algorithm   string `json:"a"`
	Data        string `json:"d"`
	EncryptedAt int64  `json:"e"`
}

func (e *ConfigEncryptor) EncryptConfig(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	encrypted, err := e.Encrypt(string(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	config := EncryptedConfig{
		Version:     1,
		Algorithm:   "AES-256-GCM",
		Data:        encrypted,
		EncryptedAt: time.Now().Unix(),
	}

	resultJSON, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	return string(resultJSON), nil
}

func (e *ConfigEncryptor) DecryptConfig(data string, target interface{}) error {
	var config EncryptedConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	decrypted, err := e.Decrypt(config.Data)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	return json.Unmarshal([]byte(decrypted), target)
}
