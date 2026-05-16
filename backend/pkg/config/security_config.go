package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SecurityConfig struct {
	CSRF                     CSRFConfig
	XSS                      XSSConfig
	Signature                SignatureConfig
	Crypto                   CryptoConfig
	RateLimit                RateLimitConfig
	Session                  SessionConfig
	Password                 PasswordConfig
	AllowedOrigins           []string
	AllowedMethods           []string
	AllowedHeaders           []string
	MaxRequestBodySize       int64
	EnableHSTS               bool
	HSTSMaxAge               int
	EnableCSP                bool
	ContentSecurityPolicy    string
}

type CSRFConfig struct {
	Enable             bool
	TokenLength        int
	TokenExpiration    time.Duration
	HeaderName         string
	FormFieldName      string
	CookieName         string
	SafeMethods        []string
	RequireValidation  bool
}

type XSSConfig struct {
	EnableLog        bool
	AllowedTags      []string
	BlockAttributes   bool
	MaxLength        int
	EnableSafeHTML   bool
}

type SignatureConfig struct {
	Enable              bool
	SecretKey           string
	Algorithm           string
	TimestampTolerance  time.Duration
	RequireTimestamp    bool
	RequireNonce        bool
	NonceCacheTTL       time.Duration
	SignatureHeader     string
	TimestampHeader     string
	NonceHeader        string
}

type CryptoConfig struct {
	AESKeySize         int
	HashAlgorithm      string
	PBKDF2Iterations  int
	SaltLength        int
	EnableKeyRotation bool
	KeyRotationPeriod time.Duration
}

type RateLimitConfig struct {
	Enable            bool
	RequestsPerSecond int
	BurstSize         int
	CleanupInterval   time.Duration
	EnablePerIP       bool
	EnablePerUser     bool
}

type SessionConfig struct {
	Timeout         time.Duration
	RefreshTimeout  time.Duration
	SecureCookie   bool
	HTTPOnly       bool
	SameSite       string
	SessionName    string
}

type PasswordConfig struct {
	MinLength       int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumber    bool
	RequireSpecial   bool
	MaxAttempts     int
	LockoutDuration time.Duration
	BcryptCost     int
}

var (
	securityConfigInstance *SecurityConfig
	securityConfigOnce     sync.Once
	securityConfigMutex   sync.RWMutex
)

func LoadSecurityConfig() *SecurityConfig {
	securityConfigOnce.Do(func() {
		securityConfigInstance = &SecurityConfig{
			CSRF: CSRFConfig{
				Enable:            getEnvAsBoolSecurity("CSRF_ENABLE", true),
				TokenLength:       getEnvAsIntSecurity("CSRF_TOKEN_LENGTH", 32),
				TokenExpiration:   getEnvAsDurationSecurity("CSRF_TOKEN_EXPIRATION", 1*time.Hour),
				HeaderName:        getEnvSecurity("CSRF_HEADER_NAME", "X-CSRF-Token"),
				FormFieldName:     getEnvSecurity("CSRF_FORM_FIELD", "csrf_token"),
				CookieName:        getEnvSecurity("CSRF_COOKIE_NAME", "csrf_token"),
				SafeMethods:       getEnvAsSliceSecurity("CSRF_SAFE_METHODS", "GET,HEAD,OPTIONS", ","),
				RequireValidation: getEnvAsBoolSecurity("CSRF_REQUIRE_VALIDATION", true),
			},
			XSS: XSSConfig{
				EnableLog:       getEnvAsBoolSecurity("XSS_ENABLE_LOG", true),
				AllowedTags:     getEnvAsSliceSecurity("XSS_ALLOWED_TAGS", "p,br,strong,em,u,h1,h2,h3,h4,h5,h6,ul,ol,li,a,img", ","),
				BlockAttributes: getEnvAsBoolSecurity("XSS_BLOCK_ATTRIBUTES", false),
				MaxLength:       getEnvAsIntSecurity("XSS_MAX_LENGTH", 10000),
				EnableSafeHTML:  getEnvAsBoolSecurity("XSS_ENABLE_SAFE_HTML", true),
			},
			Signature: SignatureConfig{
				Enable:             getEnvAsBoolSecurity("SIGNATURE_ENABLE", true),
				SecretKey:          getEnvSecurity("SIGNATURE_SECRET_KEY", "default-secret-key-change-in-production"),
				Algorithm:          getEnvSecurity("SIGNATURE_ALGORITHM", "SHA256"),
				TimestampTolerance: getEnvAsDurationSecurity("SIGNATURE_TIMESTAMP_TOLERANCE", 5*time.Minute),
				RequireTimestamp:   getEnvAsBoolSecurity("SIGNATURE_REQUIRE_TIMESTAMP", true),
				RequireNonce:       getEnvAsBoolSecurity("SIGNATURE_REQUIRE_NONCE", true),
				NonceCacheTTL:      getEnvAsDurationSecurity("SIGNATURE_NONCE_CACHE_TTL", 24*time.Hour),
				SignatureHeader:    getEnvSecurity("SIGNATURE_HEADER", "X-Signature"),
				TimestampHeader:    getEnvSecurity("SIGNATURE_TIMESTAMP_HEADER", "X-Timestamp"),
				NonceHeader:       getEnvSecurity("SIGNATURE_NONCE_HEADER", "X-Nonce"),
			},
			Crypto: CryptoConfig{
				AESKeySize:         getEnvAsIntSecurity("CRYPTO_AES_KEY_SIZE", 32),
				HashAlgorithm:      getEnvSecurity("CRYPTO_HASH_ALGORITHM", "sha256"),
				PBKDF2Iterations:  getEnvAsIntSecurity("CRYPTO_PBKDF2_ITERATIONS", 100000),
				SaltLength:         getEnvAsIntSecurity("CRYPTO_SALT_LENGTH", 32),
				EnableKeyRotation:  getEnvAsBoolSecurity("CRYPTO_ENABLE_KEY_ROTATION", false),
				KeyRotationPeriod: getEnvAsDurationSecurity("CRYPTO_KEY_ROTATION_PERIOD", 90*24*time.Hour),
			},
			RateLimit: RateLimitConfig{
				Enable:            getEnvAsBoolSecurity("RATELIMIT_ENABLE", true),
				RequestsPerSecond: getEnvAsIntSecurity("RATELIMIT_RPS", 100),
				BurstSize:         getEnvAsIntSecurity("RATELIMIT_BURST", 200),
				CleanupInterval:   getEnvAsDurationSecurity("RATELIMIT_CLEANUP_INTERVAL", 5*time.Minute),
				EnablePerIP:       getEnvAsBoolSecurity("RATELIMIT_ENABLE_PER_IP", true),
				EnablePerUser:     getEnvAsBoolSecurity("RATELIMIT_ENABLE_PER_USER", true),
			},
			Session: SessionConfig{
				Timeout:        getEnvAsDurationSecurity("SESSION_TIMEOUT", 24*time.Hour),
				RefreshTimeout: getEnvAsDurationSecurity("SESSION_REFRESH_TIMEOUT", 30*time.Minute),
				SecureCookie:   getEnvAsBoolSecurity("SESSION_SECURE_COOKIE", true),
				HTTPOnly:       getEnvAsBoolSecurity("SESSION_HTTP_ONLY", true),
				SameSite:       getEnvSecurity("SESSION_SAME_SITE", "strict"),
				SessionName:    getEnvSecurity("SESSION_NAME", "session_id"),
			},
			Password: PasswordConfig{
				MinLength:         getEnvAsIntSecurity("PASSWORD_MIN_LENGTH", 8),
				RequireUppercase:  getEnvAsBoolSecurity("PASSWORD_REQUIRE_UPPERCASE", true),
				RequireLowercase:  getEnvAsBoolSecurity("PASSWORD_REQUIRE_LOWERCASE", true),
				RequireNumber:     getEnvAsBoolSecurity("PASSWORD_REQUIRE_NUMBER", true),
				RequireSpecial:    getEnvAsBoolSecurity("PASSWORD_REQUIRE_SPECIAL", true),
				MaxAttempts:       getEnvAsIntSecurity("PASSWORD_MAX_ATTEMPTS", 5),
				LockoutDuration:   getEnvAsDurationSecurity("PASSWORD_LOCKOUT_DURATION", 30*time.Minute),
				BcryptCost:        getEnvAsIntSecurity("PASSWORD_BCRYPT_COST", 12),
			},
			AllowedOrigins:        getEnvAsSliceSecurity("CORS_ALLOWED_ORIGINS", "*", ","),
			AllowedMethods:        getEnvAsSliceSecurity("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,PATCH,OPTIONS", ","),
			AllowedHeaders:        getEnvAsSliceSecurity("CORS_ALLOWED_HEADERS", "Origin,Content-Type,Accept,Authorization,X-CSRF-Token,X-Signature,X-Timestamp,X-Nonce", ","),
			MaxRequestBodySize:    getEnvAsInt64Security("MAX_REQUEST_BODY_SIZE", 10*1024*1024),
			EnableHSTS:            getEnvAsBoolSecurity("ENABLE_HSTS", true),
			HSTSMaxAge:            getEnvAsIntSecurity("HSTS_MAX_AGE", 31536000),
			EnableCSP:             getEnvAsBoolSecurity("ENABLE_CSP", true),
			ContentSecurityPolicy: getEnvSecurity("CSP_POLICY", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.bootcdn.net; style-src 'self' 'unsafe-inline' https://cdn.bootcdn.net; font-src 'self' https://cdn.bootcdn.net; img-src 'self' data: https:; connect-src 'self';"),
		}
	})

	return securityConfigInstance
}

func getEnvAsIntSecurity(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		result, err := strconv.Atoi(value)
		if err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvAsInt64Security(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		result, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvAsBoolSecurity(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		result, err := strconv.ParseBool(value)
		if err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvAsDurationSecurity(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvAsSliceSecurity(key, defaultValue, sep string) []string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return strings.Split(value, sep)
	}
	return strings.Split(defaultValue, sep)
}

func getEnvSecurity(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func GetSecurityConfig() *SecurityConfig {
	securityConfigMutex.RLock()
	defer securityConfigMutex.RUnlock()

	if securityConfigInstance == nil {
		return LoadSecurityConfig()
	}
	return securityConfigInstance
}

func UpdateSecurityConfig(config *SecurityConfig) {
	securityConfigMutex.Lock()
	defer securityConfigMutex.Unlock()
	securityConfigInstance = config
}

func (c *SecurityConfig) Validate() error {
	if c.CSRF.TokenLength < 16 {
		return fmt.Errorf("CSRF token length must be at least 16 characters")
	}

	if c.CSRF.TokenExpiration < 1*time.Minute {
		return fmt.Errorf("CSRF token expiration must be at least 1 minute")
	}

	if c.Signature.SecretKey == "" || c.Signature.SecretKey == "default-secret-key-change-in-production" {
		return fmt.Errorf("SIGNATURE_SECRET_KEY must be set and not be the default value in production")
	}

	if c.Crypto.AESKeySize != 16 && c.Crypto.AESKeySize != 24 && c.Crypto.AESKeySize != 32 {
		return fmt.Errorf("CRYPTO_AES_KEY_SIZE must be 16, 24, or 32")
	}

	if c.Crypto.PBKDF2Iterations < 10000 {
		return fmt.Errorf("CRYPTO_PBKDF2_ITERATIONS must be at least 10000")
	}

	if c.Password.MinLength < 6 {
		return fmt.Errorf("PASSWORD_MIN_LENGTH must be at least 6")
	}

	if c.Password.BcryptCost < 4 || c.Password.BcryptCost > 31 {
		return fmt.Errorf("PASSWORD_BCRYPT_COST must be between 4 and 31")
	}

	if c.RateLimit.RequestsPerSecond < 1 {
		return fmt.Errorf("RATELIMIT_RPS must be at least 1")
	}

	return nil
}

func (c *SecurityConfig) GetCSRFConfig() CSRFConfig {
	return c.CSRF
}

func (c *SecurityConfig) GetXSSConfig() XSSConfig {
	return c.XSS
}

func (c *SecurityConfig) GetSignatureConfig() SignatureConfig {
	return c.Signature
}

func (c *SecurityConfig) GetCryptoConfig() CryptoConfig {
	return c.Crypto
}

func (c *SecurityConfig) GetRateLimitConfig() RateLimitConfig {
	return c.RateLimit
}

func (c *SecurityConfig) GetSessionConfig() SessionConfig {
	return c.Session
}

func (c *SecurityConfig) GetPasswordConfig() PasswordConfig {
	return c.Password
}

func (c *SecurityConfig) IsProductionReady() bool {
	if c.Signature.SecretKey == "default-secret-key-change-in-production" {
		return false
	}

	if c.Password.BcryptCost < 10 {
		return false
	}

	if c.Crypto.PBKDF2Iterations < 50000 {
		return false
	}

	return true
}

func (c *SecurityConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"csrf":                    c.CSRF,
		"xss":                     c.XSS,
		"signature":               c.Signature,
		"crypto":                  c.Crypto,
		"rate_limit":              c.RateLimit,
		"session":                 c.Session,
		"password":                c.Password,
		"allowed_origins":         c.AllowedOrigins,
		"allowed_methods":         c.AllowedMethods,
		"allowed_headers":         c.AllowedHeaders,
		"max_request_body_size":   c.MaxRequestBodySize,
		"enable_hsts":             c.EnableHSTS,
		"hsts_max_age":            c.HSTSMaxAge,
		"enable_csp":              c.EnableCSP,
		"production_ready":        c.IsProductionReady(),
	}
}

type SecurityConfigUpdate struct {
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (c *SecurityConfig) UpdateField(update SecurityConfigUpdate) error {
	switch update.Field {
	case "csrf.enable":
		if v, ok := update.Value.(bool); ok {
			c.CSRF.Enable = v
		}
	case "csrf.token_length":
		if v, ok := update.Value.(int); ok {
			c.CSRF.TokenLength = v
		}
	case "xss.enable_log":
		if v, ok := update.Value.(bool); ok {
			c.XSS.EnableLog = v
		}
	case "signature.enable":
		if v, ok := update.Value.(bool); ok {
			c.Signature.Enable = v
		}
	case "rate_limit.enable":
		if v, ok := update.Value.(bool); ok {
			c.RateLimit.Enable = v
		}
	case "rate_limit.requests_per_second":
		if v, ok := update.Value.(int); ok {
			c.RateLimit.RequestsPerSecond = v
		}
	case "password.min_length":
		if v, ok := update.Value.(int); ok {
			c.Password.MinLength = v
		}
	case "session.timeout":
		if v, ok := update.Value.(string); ok {
			if d, err := time.ParseDuration(v); err == nil {
				c.Session.Timeout = d
			}
		}
	default:
		return fmt.Errorf("unknown field: %s", update.Field)
	}

	UpdateSecurityConfig(c)

	return nil
}
