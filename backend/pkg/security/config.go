package security

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled         bool          `json:"enabled"`
	GlobalLimit     int           `json:"global_limit"`
	GlobalWindow    time.Duration `json:"global_window"`
	IPLimit        int           `json:"ip_limit"`
	IPWindow       time.Duration `json:"ip_window"`
	UserLimit      int           `json:"user_limit"`
	UserWindow     time.Duration `json:"user_window"`
	APIKeyLimit    int           `json:"api_key_limit"`
	APIKeyWindow   time.Duration `json:"api_key_window"`
	BurstSize      int           `json:"burst_size"`
	Algorithm       string        `json:"algorithm"` // fixed, sliding, token_bucket, leaky_bucket
}

// CSRFConfig CSRF配置
type CSRFConfigSecurity struct {
	Enabled        bool          `json:"enabled"`
	Secret         string        `json:"secret"`
	TokenLength    int           `json:"token_length"`
	TokenExpiry    time.Duration `json:"token_expiry"`
	CookieName     string        `json:"cookie_name"`
	HeaderName     string        `json:"header_name"`
	FormFieldName  string        `json:"form_field_name"`
	CookieSecure   bool          `json:"cookie_secure"`
	CookieHTTPOnly bool          `json:"cookie_http_only"`
	SameSite       string        `json:"same_site"`
	ExcludePaths   []string      `json:"exclude_paths"`
}

// XSSConfig XSS配置
type XSSConfigSecurity struct {
	Enabled          bool     `json:"enabled"`
	AllowedTags     []string `json:"allowed_tags"`
	AllowedAttrs    []string `json:"allowed_attrs"`
	AllowedProtocols []string `json:"allowed_protocols"`
	MaxInputLength  int      `json:"max_input_length"`
	EnableInputFilter  bool   `json:"enable_input_filter"`
	EnableOutputFilter bool   `json:"enable_output_filter"`
}

// IPConfig IP配置
type IPConfigSecurity struct {
	Whitelist     []string `json:"whitelist"`
	Blacklist     []string `json:"blacklist"`
	EnableWhitelist bool   `json:"enable_whitelist"`
	EnableBlacklist bool   `json:"enable_blacklist"`
	AutoBlockThreshold int `json:"auto_block_threshold"`
	AutoBlockDuration time.Duration `json:"auto_block_duration"`
}

// BruteForceConfigSecurity 暴力破解配置
type BruteForceConfigSecurity struct {
	Enabled       bool          `json:"enabled"`
	MaxAttempts   int           `json:"max_attempts"`
	LockoutTime  time.Duration `json:"lockout_time"`
	ResetAfter   time.Duration `json:"reset_after"`
	MaxIdentifiers int          `json:"max_identifiers"`
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Algorithm        string `json:"algorithm"` // aes-128, aes-256
	MasterKey        string `json:"master_key"`
	KeyRotationDays  int    `json:"key_rotation_days"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RateLimit  RateLimitConfig      `json:"rate_limit"`
	CSRF       CSRFConfigSecurity   `json:"csrf"`
	XSS        XSSConfigSecurity    `json:"xss"`
	IP         IPConfigSecurity     `json:"ip"`
	BruteForce BruteForceConfigSecurity `json:"brute_force"`
	Encryption EncryptionConfig      `json:"encryption"`
	
	mu sync.RWMutex
}

// GlobalSecurityConfig 全局安全配置
var GlobalSecurityConfig *SecurityConfig

var configLock sync.RWMutex

// LoadSecurityConfig 加载安全配置
func LoadSecurityConfig(filepath string) (*SecurityConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return getDefaultSecurityConfig(), nil
	}

	var config SecurityConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return getDefaultSecurityConfig(), nil
	}

	configLock.Lock()
	GlobalSecurityConfig = &config
	configLock.Unlock()

	return &config, nil
}

// SaveSecurityConfig 保存安全配置
func SaveSecurityConfig(filepath string, config *SecurityConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath[:len(filepath)-len(filepath)-len("security.json")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// GetSecurityConfig 获取安全配置
func GetSecurityConfig() *SecurityConfig {
	configLock.RLock()
	defer configLock.RUnlock()

	if GlobalSecurityConfig != nil {
		return GlobalSecurityConfig
	}

	return getDefaultSecurityConfig()
}

// UpdateSecurityConfig 更新安全配置
func UpdateSecurityConfig(config *SecurityConfig) {
	configLock.Lock()
	defer configLock.Unlock()
	GlobalSecurityConfig = config
}

// UpdateRateLimitConfig 更新限流配置
func (c *SecurityConfig) UpdateRateLimitConfig(config RateLimitConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RateLimit = config
}

// UpdateCSRFConfig 更新CSRF配置
func (c *SecurityConfig) UpdateCSRFConfig(config CSRFConfigSecurity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CSRF = config
}

// UpdateXSSConfig 更新XSS配置
func (c *SecurityConfig) UpdateXSSConfig(config XSSConfigSecurity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.XSS = config
}

// UpdateIPConfig 更新IP配置
func (c *SecurityConfig) UpdateIPConfig(config IPConfigSecurity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IP = config
}

// UpdateBruteForceConfig 更新暴力破解配置
func (c *SecurityConfig) UpdateBruteForceConfig(config BruteForceConfigSecurity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BruteForce = config
}

// UpdateEncryptionConfig 更新加密配置
func (c *SecurityConfig) UpdateEncryptionConfig(config EncryptionConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Encryption = config
}

// GetRateLimitConfig 获取限流配置
func (c *SecurityConfig) GetRateLimitConfig() RateLimitConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RateLimit
}

// GetCSRFConfig 获取CSRF配置
func (c *SecurityConfig) GetCSRFConfig() CSRFConfigSecurity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CSRF
}

// GetXSSConfig 获取XSS配置
func (c *SecurityConfig) GetXSSConfig() XSSConfigSecurity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.XSS
}

// GetIPConfig 获取IP配置
func (c *SecurityConfig) GetIPConfig() IPConfigSecurity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.IP
}

// GetBruteForceConfig 获取暴力破解配置
func (c *SecurityConfig) GetBruteForceConfig() BruteForceConfigSecurity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.BruteForce
}

// GetEncryptionConfig 获取加密配置
func (c *SecurityConfig) GetEncryptionConfig() EncryptionConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Encryption
}

// ValidateConfig 验证配置
func (c *SecurityConfig) ValidateConfig() error {
	if c.RateLimit.GlobalLimit <= 0 {
		return fmt.Errorf("global limit must be positive")
	}

	if c.RateLimit.GlobalWindow <= 0 {
		return fmt.Errorf("global window must be positive")
	}

	if c.BruteForce.MaxAttempts <= 0 {
		return fmt.Errorf("max attempts must be positive")
	}

	if c.BruteForce.LockoutTime <= 0 {
		return fmt.Errorf("lockout time must be positive")
	}

	if c.CSRF.TokenLength < 16 {
		return fmt.Errorf("token length must be at least 16")
	}

	if c.CSRF.Secret == "" {
		return fmt.Errorf("csrf secret is required")
	}

	return nil
}

// ApplyDefaultConfig 应用默认配置
func (c *SecurityConfig) ApplyDefaultConfig() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.RateLimit.GlobalLimit == 0 {
		c.RateLimit = RateLimitConfig{
			Enabled:       true,
			GlobalLimit:   1000,
			GlobalWindow:  time.Minute,
			IPLimit:       100,
			IPWindow:      time.Minute,
			UserLimit:     200,
			UserWindow:    time.Minute,
			BurstSize:     20,
			Algorithm:     "sliding",
		}
	}

	if c.CSRF.TokenLength == 0 {
		c.CSRF = CSRFConfigSecurity{
			Enabled:        true,
			TokenLength:    32,
			TokenExpiry:    24 * time.Hour,
			CookieName:     "csrf_token",
			HeaderName:     "X-CSRF-Token",
			FormFieldName:  "_csrf",
			CookieSecure:   true,
			CookieHTTPOnly: false,
			SameSite:       "strict",
			ExcludePaths:   []string{"/api/health", "/api/healthz"},
		}
	}

	if c.XSS.MaxInputLength == 0 {
		c.XSS = XSSConfigSecurity{
			Enabled:            true,
			AllowedTags:        []string{"p", "br", "b", "i", "u", "em", "strong", "a", "ul", "ol", "li"},
			AllowedAttrs:       []string{"href", "title", "alt", "class"},
			AllowedProtocols:   []string{"http", "https", "mailto"},
			MaxInputLength:     10000,
			EnableInputFilter:  true,
			EnableOutputFilter: true,
		}
	}

	if c.BruteForce.MaxAttempts == 0 {
		c.BruteForce = BruteForceConfigSecurity{
			Enabled:        true,
			MaxAttempts:    5,
			LockoutTime:    15 * time.Minute,
			ResetAfter:     30 * time.Minute,
			MaxIdentifiers: 10000,
		}
	}

	if c.Encryption.Algorithm == "" {
		c.Encryption = EncryptionConfig{
			Algorithm:       "aes-256",
			KeyRotationDays: 90,
		}
	}
}

// ExportConfig 导出配置
func (c *SecurityConfig) ExportConfig() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	configCopy := *c
	return json.MarshalIndent(&configCopy, "", "  ")
}

// ImportConfig 导入配置
func (c *SecurityConfig) ImportConfig(data []byte) error {
	var newConfig SecurityConfig
	if err := json.Unmarshal(data, &newConfig); err != nil {
		return err
	}

	if err := newConfig.ValidateConfig(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	*c = newConfig

	configLock.Lock()
	GlobalSecurityConfig = &newConfig
	configLock.Unlock()

	return nil
}

// CloneConfig 克隆配置
func (c *SecurityConfig) CloneConfig() *SecurityConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &SecurityConfig{
		RateLimit:  c.RateLimit,
		CSRF:       c.CSRF,
		XSS:        c.XSS,
		IP:         c.IP,
		BruteForce: c.BruteForce,
		Encryption: c.Encryption,
	}
}

// getDefaultSecurityConfig 获取默认安全配置
func getDefaultSecurityConfig() *SecurityConfig {
	config := &SecurityConfig{}
	config.ApplyDefaultConfig()
	return config
}

// NewSecurityConfig 创建新的安全配置
func NewSecurityConfig() *SecurityConfig {
	config := &SecurityConfig{}
	config.ApplyDefaultConfig()
	return config
}

// LoadConfigFromEnv 从环境变量加载配置
func LoadConfigFromEnv() *SecurityConfig {
	config := NewSecurityConfig()

	config.RateLimit.Enabled = getEnvBool("SECURITY_RATE_LIMIT_ENABLED", true)
	config.RateLimit.GlobalLimit = getEnvInt("SECURITY_RATE_LIMIT_GLOBAL", 1000)
	config.RateLimit.IPLimit = getEnvInt("SECURITY_RATE_LIMIT_IP", 100)

	config.CSRF.Enabled = getEnvBool("SECURITY_CSRF_ENABLED", true)
	config.CSRF.Secret = os.Getenv("SECURITY_CSRF_SECRET")

	config.XSS.Enabled = getEnvBool("SECURITY_XSS_ENABLED", true)

	config.BruteForce.Enabled = getEnvBool("SECURITY_BRUTE_FORCE_ENABLED", true)
	config.BruteForce.MaxAttempts = getEnvInt("SECURITY_BRUTE_FORCE_MAX_ATTEMPTS", 5)

	config.Encryption.MasterKey = os.Getenv("SECURITY_ENCRYPTION_KEY")

	return config
}

// Helper functions
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		_, err := fmt.Sscanf(val, "%d", &result)
		if err == nil {
			return result
		}
	}
	return defaultVal
}
