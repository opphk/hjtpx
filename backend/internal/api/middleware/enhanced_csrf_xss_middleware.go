package middleware

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type EnhancedCSRFConfig struct {
	Enabled              bool
	TokenLength          int
	TokenExpiration      time.Duration
	HeaderName           string
	CookieName           string
	FormFieldName        string
	SafeMethods          []string
	DoubleSubmitCookie   bool
	RequireEncryption    bool
	RotateOnVerification bool
	ExcludePaths         []string
	RedisEnabled         bool
}

var defaultEnhancedCSRFConfig = EnhancedCSRFConfig{
	Enabled:              true,
	TokenLength:          32,
	TokenExpiration:      1 * time.Hour,
	HeaderName:           "X-CSRF-Token",
	CookieName:           "csrf_token",
	FormFieldName:        "csrf_token",
	SafeMethods:          []string{"GET", "HEAD", "OPTIONS", "TRACE"},
	DoubleSubmitCookie:   true,
	RequireEncryption:    false,
	RotateOnVerification: true,
	ExcludePaths:         []string{"/health", "/metrics", "/api/health"},
	RedisEnabled:         redis.Client != nil,
}

type EnhancedXSSConfig struct {
	Enabled              bool
	EnableHTMLSanitization bool
	EnableAttributeFiltering bool
	EnableURLValidation  bool
	EnableJSRemoval      bool
	AllowedTags          []string
	AllowedAttrs         []string
	MaxInputLength       int
	ExcludePaths         []string
	LogViolations        bool
}

var defaultEnhancedXSSConfig = EnhancedXSSConfig{
	Enabled:               true,
	EnableHTMLSanitization: true,
	EnableAttributeFiltering: true,
	EnableURLValidation:   true,
	EnableJSRemoval:       true,
	AllowedTags:           []string{"p", "br", "b", "i", "em", "strong", "a", "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6"},
	AllowedAttrs:          []string{"href", "title", "class", "id"},
	MaxInputLength:       10000,
	ExcludePaths:         []string{"/health", "/metrics", "/api/health"},
	LogViolations:        true,
}

type EnhancedCSPConfig struct {
	Enabled               bool
	DefaultSrc           []string
	ScriptSrc            []string
	StyleSrc             []string
	ImgSrc               []string
	FontSrc              []string
	ConnectSrc           []string
	FrameSrc             []string
	ObjectSrc            []string
	ReportURI            string
	EnableNonce          bool
	ExcludePaths         []string
}

var defaultEnhancedCSPConfig = EnhancedCSPConfig{
	Enabled:    true,
	DefaultSrc: []string{"'self'"},
	ScriptSrc:  []string{"'self'"},
	StyleSrc:   []string{"'self'", "'unsafe-inline'"},
	ImgSrc:     []string{"'self'", "data:", "https:"},
	FontSrc:    []string{"'self'"},
	ConnectSrc: []string{"'self'"},
	FrameSrc:   []string{"'none'"},
	ObjectSrc:  []string{"'none'"},
	EnableNonce: true,
	ExcludePaths: []string{"/health", "/metrics", "/api/health"},
}

var (
	csrfSecurity      *service.EnhancedCSRFSecurity
	csrfSecurityOnce  sync.Once
	xssSecurity       *service.XSSSecurity
	xssSecurityOnce   sync.Once
)

func initCSRFSecurity() {
	csrfSecurityOnce.Do(func() {
		csrfSecurity = service.NewCSRFSecurity(nil)
	})
}

func initXSSSecurity() {
	xssSecurityOnce.Do(func() {
		xssSecurity = service.NewXSSSecurity(nil)
	})
}

type sessionCSRFData struct {
	Token     string
	ExpiresAt time.Time
	Used      bool
}

var (
	csrfTokenStoreXSS = make(map[string]*sessionCSRFData)
	csrfStoreMuXSS    sync.RWMutex
)

func EnhancedCSRFProtection(configs ...EnhancedCSRFConfig) gin.HandlerFunc {
	initCSRFSecurity()

	cfg := defaultEnhancedCSRFConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		method := c.Request.Method
		isSafeMethod := false
		for _, m := range cfg.SafeMethods {
			if method == m {
				isSafeMethod = true
				break
			}
		}

		if isSafeMethod {
			if method == "GET" || method == "HEAD" {
				sessionID := generateCSRFCSIDXSS(c)

				token, err := csrfSecurity.GenerateToken()
				if err == nil {
					csrfStoreMuXSS.Lock()
					csrfTokenStoreXSS[sessionID] = &sessionCSRFData{
						Token:     token,
						ExpiresAt: time.Now().Add(cfg.TokenExpiration),
						Used:      false,
					}
					csrfStoreMuXSS.Unlock()

					c.Set("csrf_token", token)
					c.Set("csrf_session_id", sessionID)
					c.Header(cfg.HeaderName, token)

					c.SetCookie(
						cfg.CookieName,
						token,
						int(cfg.TokenExpiration.Seconds()),
						"/",
						"",
						cfg.RequireEncryption,
						true,
					)
				}
			}
			c.Next()
			return
		}

		sessionID := generateCSRFCSIDXSS(c)

		var token string
		token = c.GetHeader(cfg.HeaderName)
		if token == "" {
			token = c.Query(cfg.FormFieldName)
		}
		if token == "" {
			token = c.PostForm(cfg.FormFieldName)
		}
		if token == "" {
			if cookieToken, err := c.Cookie(cfg.CookieName); err == nil {
				token = cookieToken
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_missing",
				"code":    "CSRF_TOKEN_MISSING",
				"message": "CSRF token is required for this request",
			})
			return
		}

		csrfStoreMuXSS.RLock()
		sessionData, exists := csrfTokenStoreXSS[sessionID]
		csrfStoreMuXSS.RUnlock()

		if !exists || sessionData == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_session_invalid",
				"code":    "CSRF_SESSION_INVALID",
				"message": "Invalid or expired CSRF session",
			})
			return
		}

		if time.Now().After(sessionData.ExpiresAt) {
			csrfStoreMuXSS.Lock()
			delete(csrfTokenStore, sessionID)
			csrfStoreMuXSS.Unlock()

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_expired",
				"code":    "CSRF_TOKEN_EXPIRED",
				"message": "CSRF token has expired",
			})
			return
		}

		valid := false

		if sessionData.Token == token {
			valid = true
		}

		if cfg.DoubleSubmitCookie {
			cookieToken, err := c.Cookie(cfg.CookieName)
			if err == nil && cookieToken == token {
				valid = true
			}
		}

		if !valid {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_invalid",
				"code":    "CSRF_TOKEN_INVALID",
				"message": "Invalid CSRF token",
			})
			return
		}

		if cfg.RotateOnVerification {
			csrfStoreMuXSS.Lock()
			delete(csrfTokenStore, sessionID)
			csrfStoreMuXSS.Unlock()
		}

		c.Set("csrf_verified", true)
		c.Set("csrf_session_id", sessionID)

		c.Next()
	}
}

func generateCSRFCSIDXSS(c *gin.Context) string {
	if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0]) + ":" + c.ClientIP()
	}

	return c.ClientIP() + ":" + c.GetHeader("User-Agent")
}

func EnhancedXSSProtectionMiddleware(configs ...EnhancedXSSConfig) gin.HandlerFunc {
	initXSSSecurity()

	cfg := defaultEnhancedXSSConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	xssConfig := &service.XSSSecurityConfig{
		EnableHTMLSanitization:   cfg.EnableHTMLSanitization,
		EnableAttributeFiltering:  cfg.EnableAttributeFiltering,
		EnableURLValidation:      cfg.EnableURLValidation,
		EnableJSRemoval:          cfg.EnableJSRemoval,
		AllowedTags:              cfg.AllowedTags,
		AllowedAttrs:             cfg.AllowedAttrs,
		MaxInputLength:           cfg.MaxInputLength,
	}

	xssSecurity := service.NewXSSSecurity(xssConfig)
	_ = xssSecurity // Mark as intentionally unused

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		c.Request.Header.Set("X-XSS-Protection", "1; mode=block")

		c.Next()
	}
}

func EnhancedCSPMiddleware(configs ...EnhancedCSPConfig) gin.HandlerFunc {
	cfg := defaultEnhancedCSPConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	cspConfig := service.ContentSecurityPolicyConfig{
		DefaultSrc:     cfg.DefaultSrc,
		ScriptSrc:      cfg.ScriptSrc,
		StyleSrc:       cfg.StyleSrc,
		ImgSrc:         cfg.ImgSrc,
		FontSrc:        cfg.FontSrc,
		ConnectSrc:     cfg.ConnectSrc,
		FrameSrc:       cfg.FrameSrc,
		ObjectSrc:      cfg.ObjectSrc,
		ReportURI:      cfg.ReportURI,
		EnableNonce:    cfg.EnableNonce,
	}

	policy := cspConfig.BuildPolicy()

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		c.Header("Content-Security-Policy", policy)

		c.Header("X-Content-Security-Policy", policy)

		c.Next()
	}
}

func EnhancedSecurityHeadersMiddleware() gin.HandlerFunc {
	securityConfig := service.SecurityHeadersConfig{
		EnableCSP:              true,
		EnableHSTS:             true,
		HSTSMaxAge:             31536000,
		HSTSIncludeSubdomains:  true,
		HSTSPreload:           true,
		EnableXFrameOptions:    true,
		XFrameOptions:          "DENY",
		EnableXContentType:    true,
		XContentTypeOptions:   "nosniff",
		EnableXSSProtection:   true,
		XSSProtectionMode:     "block",
		EnableReferrerPolicy:  true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		EnablePermissionsPolicy: true,
		PermissionsPolicy:     "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
		EnableOtherHeaders:    true,
	}

	headers := service.BuildSecurityHeaders(securityConfig)

	return func(c *gin.Context) {
		for name, value := range headers {
			c.Header(name, value)
		}

		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("X-Download-Options", "noopen")
		c.Header("X-Request-ID", generateRequestIDXSS())

		c.Next()
	}
}

func generateRequestIDXSS() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func SetupEnhancedSecurityMiddleware(r *gin.Engine) {
	r.Use(EnhancedDDoSMiddleware())

	r.Use(EnhancedCSRFProtection())

	r.Use(EnhancedXSSProtectionMiddleware())

	r.Use(EnhancedCSPMiddleware())

	r.Use(EnhancedSecurityHeadersMiddleware())

	r.Use(ConnectionTrackingMiddlewareHandler(100, 60))

	r.Use(TrafficAnalysisMiddlewareHandler())

	r.Use(BehavioralAnalysisMiddlewareHandler())
}

func GetEnhancedCSRFSecurity() *service.EnhancedCSRFSecurity {
	initCSRFSecurity()
	return csrfSecurity
}

func GetEnhancedXSSSecurity() *service.XSSSecurity {
	initXSSSecurity()
	return xssSecurity
}

func SanitizeInput(input string) string {
	initXSSSecurity()
	return xssSecurity.SanitizeInput(input)
}

func SanitizeHTML(input string) string {
	initXSSSecurity()
	return xssSecurity.SanitizeHTML(input)
}

type EnhancedInputValidationConfig struct {
	Enabled         bool
	ValidateQuery   bool
	ValidateForm    bool
	ValidateJSON    bool
	ValidateHeaders bool
	MaxQueryParams  int
	MaxBodySize     int64
	ExcludePaths    []string
	LogViolations   bool
}

var defaultEnhancedInputValidationConfig = EnhancedInputValidationConfig{
	Enabled:         true,
	ValidateQuery:   true,
	ValidateForm:    true,
	ValidateJSON:    true,
	ValidateHeaders: false,
	MaxQueryParams:  50,
	MaxBodySize:     1024 * 1024 * 10,
	ExcludePaths:    []string{"/health", "/metrics"},
	LogViolations:   true,
}

func DetectXSS(input string) (bool, string) {
	initXSSSecurity()
	return xssSecurity.DetectXSS(input)
}

func EnhancedInputValidationMiddleware(configs ...EnhancedInputValidationConfig) gin.HandlerFunc {
	cfg := defaultEnhancedInputValidationConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	initXSSSecurity()

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		if cfg.MaxBodySize > 0 && c.Request.ContentLength > cfg.MaxBodySize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "body_too_large",
				"message": fmt.Sprintf("Request body exceeds maximum size of %d bytes", cfg.MaxBodySize),
			})
			return
		}

		if cfg.ValidateQuery {
			query := c.Request.URL.Query()
			if len(query) > cfg.MaxQueryParams {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":   "too_many_query_params",
					"message": fmt.Sprintf("Too many query parameters, maximum is %d", cfg.MaxQueryParams),
				})
				return
			}

			for key, values := range query {
				sanitizedKey := SanitizeInput(key)
				if sanitizedKey != key {
					delete(query, key)
					query[sanitizedKey] = values
				}

				for i, value := range values {
					sanitized := SanitizeInput(value)
					if sanitized != value {
						isXSS, pattern := DetectXSS(value)
						if isXSS && cfg.LogViolations {
							fmt.Printf("[XSS_DETECTED] Key: %s, Pattern: %s, IP: %s\n", key, pattern, c.ClientIP())
						}
						values[i] = sanitized
					}
				}
				query[key] = values
			}
			c.Request.URL.RawQuery = query.Encode()
		}

		if cfg.ValidateForm && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH") {
			if err := c.Request.ParseForm(); err == nil {
				for key, values := range c.Request.PostForm {
					sanitizedKey := SanitizeInput(key)
					if sanitizedKey != key {
						delete(c.Request.PostForm, key)
						c.Request.PostForm[sanitizedKey] = values
						key = sanitizedKey
					}

					for i, value := range values {
						sanitized := SanitizeHTML(value)
						if sanitized != value {
							isXSS, pattern := DetectXSS(value)
							if isXSS && cfg.LogViolations {
								fmt.Printf("[XSS_DETECTED] Form Field: %s, Pattern: %s, IP: %s\n", key, pattern, c.ClientIP())
							}
							values[i] = sanitized
						}
					}
					c.Request.PostForm[key] = values
				}
			}
		}

		c.Next()
	}
}

type SSRFProtectionConfig struct {
	Enabled            bool
	AllowedDomains     []string
	BlockedIPRanges    []string
	EnableDNSRebinding bool
	CheckLoopback      bool
	CheckPrivate       bool
}

var defaultSSRFConfig = SSRFProtectionConfig{
	Enabled:            true,
	AllowedDomains:     []string{},
	BlockedIPRanges:    []string{},
	EnableDNSRebinding: true,
	CheckLoopback:      true,
	CheckPrivate:       true,
}

func SSRFProtectionMiddleware(configs ...SSRFProtectionConfig) gin.HandlerFunc {
	cfg := defaultSSRFConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		urlStr := c.Query("url")
		if urlStr == "" {
			urlStr = c.PostForm("url")
		}

		if urlStr != "" {
			if isSSRFAttack(urlStr, cfg) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "ssrf_detected",
					"message": "SSRF attack detected",
				})
				return
			}
		}

		c.Next()
	}
}

func isSSRFAttack(urlStr string, cfg SSRFProtectionConfig) bool {
	ssrfPatterns := []string{
		"http://127.0.0.1",
		"http://localhost",
		"http://0.0.0.0",
		"http://[::]",
		"http://[::1]",
		"file://",
		"gopher://",
		"dict://",
		"ftp://",
	}

	privateIPRegex := regexp.MustCompile(`(?i)(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|169\.254\.|127\.)`)

	for _, pattern := range ssrfPatterns {
		if strings.Contains(urlStr, pattern) {
			return true
		}
	}

	if privateIPRegex.MatchString(urlStr) {
		return true
	}

	metadataEndpoints := []string{
		"metadata.google.internal",
		"metadata.azure.com",
		"169.254.169.254",
		"metadata.openstack.org",
	}

	for _, endpoint := range metadataEndpoints {
		if strings.Contains(urlStr, endpoint) {
			return true
		}
	}

	if cfg.Enabled {
		for _, blocked := range cfg.BlockedIPRanges {
			if strings.Contains(urlStr, blocked) {
				return true
			}
		}

		if cfg.CheckPrivate {
			host := urlStr
			if strings.HasPrefix(host, "http://") {
				host = strings.TrimPrefix(host, "http://")
			} else if strings.HasPrefix(host, "https://") {
				host = strings.TrimPrefix(host, "https://")
			}
			host = strings.Split(host, "/")[0]
			host = strings.Split(host, ":")[0]

			ip := net.ParseIP(host)
			if ip != nil {
				if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
					return true
				}
			}
		}
	}

	return false
}

type SSRFProtectionMiddlewareV2 struct {
	config     SSRFProtectionConfig
	validator  *ssrfValidator
}

type ssrfValidator struct {
	blockedRanges []*net.IPNet
	allowedDomains map[string]bool
}

func newSSRFValidator(cfg SSRFProtectionConfig) *ssrfValidator {
	v := &ssrfValidator{
		allowedDomains: make(map[string]bool),
	}

	for _, domain := range cfg.AllowedDomains {
		v.allowedDomains[strings.ToLower(domain)] = true
	}

	for _, cidr := range cfg.BlockedIPRanges {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil {
			v.blockedRanges = append(v.blockedRanges, ipnet)
		}
	}

	defaultBlocked := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"0.0.0.0/8",
	}
	for _, cidr := range defaultBlocked {
		if _, ipnet, err := net.ParseCIDR(cidr); err == nil {
			v.blockedRanges = append(v.blockedRanges, ipnet)
		}
	}

	return v
}

func (v *ssrfValidator) isBlocked(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return true
	}

	host := u.Hostname()
	if host == "" {
		return true
	}

	if v.allowedDomains[strings.ToLower(host)] {
		return false
	}

	ip := net.ParseIP(host)
	if ip != nil {
		for _, blocked := range v.blockedRanges {
			if blocked.Contains(ip) {
				return true
			}
		}
	}

	return false
}
