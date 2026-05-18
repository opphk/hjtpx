package middleware

import (
	"crypto/rand"
	"fmt"
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
	csrfSecurity      *service.CSRFSecurity
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
	enhancedCSRFTokenStore = make(map[string]*sessionCSRFData)
	enhancedCSRFStoreMu    sync.RWMutex
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
				sessionID := enhancedGenerateCSRFCSID(c)

				token, err := csrfSecurity.GenerateToken()
				if err == nil {
					enhancedCSRFStoreMu.Lock()
					enhancedCSRFTokenStore[sessionID] = &sessionCSRFData{
						Token:     token,
						ExpiresAt: time.Now().Add(cfg.TokenExpiration),
						Used:      false,
					}
					enhancedCSRFStoreMu.Unlock()

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

		sessionID := enhancedGenerateCSRFCSID(c)

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

		enhancedCSRFStoreMu.RLock()
		sessionData, exists := enhancedCSRFTokenStore[sessionID]
		enhancedCSRFStoreMu.RUnlock()

		if !exists || sessionData == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_session_invalid",
				"code":    "CSRF_SESSION_INVALID",
				"message": "Invalid or expired CSRF session",
			})
			return
		}

		if time.Now().After(sessionData.ExpiresAt) {
			enhancedCSRFStoreMu.Lock()
			delete(enhancedCSRFTokenStore, sessionID)
			enhancedCSRFStoreMu.Unlock()

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
			enhancedCSRFStoreMu.Lock()
			delete(enhancedCSRFTokenStore, sessionID)
			enhancedCSRFStoreMu.Unlock()
		}

		c.Set("csrf_verified", true)
		c.Set("csrf_session_id", sessionID)

		c.Next()
	}
}

func enhancedGenerateCSRFCSID(c *gin.Context) string {
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

	cspPolicy := buildCSPPolicy(cfg)

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

		c.Header("Content-Security-Policy", cspPolicy)

		c.Header("X-Content-Security-Policy", cspPolicy)

		c.Next()
	}
}

func buildCSPPolicy(cfg EnhancedCSPConfig) string {
	directives := []string{"default-src 'self'"}

	if len(cfg.ScriptSrc) > 0 {
		directives = append(directives, "script-src "+strings.Join(cfg.ScriptSrc, " "))
	}
	if len(cfg.StyleSrc) > 0 {
		directives = append(directives, "style-src "+strings.Join(cfg.StyleSrc, " "))
	}
	if len(cfg.ImgSrc) > 0 {
		directives = append(directives, "img-src "+strings.Join(cfg.ImgSrc, " "))
	}
	if len(cfg.FontSrc) > 0 {
		directives = append(directives, "font-src "+strings.Join(cfg.FontSrc, " "))
	}
	if len(cfg.ConnectSrc) > 0 {
		directives = append(directives, "connect-src "+strings.Join(cfg.ConnectSrc, " "))
	}
	if len(cfg.FrameSrc) > 0 {
		directives = append(directives, "frame-src "+strings.Join(cfg.FrameSrc, " "))
	}
	if len(cfg.ObjectSrc) > 0 {
		directives = append(directives, "object-src "+strings.Join(cfg.ObjectSrc, " "))
	}

	if cfg.ReportURI != "" {
		directives = append(directives, "report-uri "+cfg.ReportURI)
	}

	return strings.Join(directives, "; ")
}

func EnhancedSecurityHeadersMiddleware() gin.HandlerFunc {
	securityConfig := service.DefaultSecurityHeaders

	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", securityConfig.CSP)
		c.Header("Strict-Transport-Security", securityConfig.HSTS)
		c.Header("X-Frame-Options", securityConfig.XFrameOptions)
		c.Header("X-Content-Type-Options", securityConfig.XContentTypeOptions)
		c.Header("X-XSS-Protection", securityConfig.XXSSProtection)
		c.Header("Referrer-Policy", securityConfig.ReferrerPolicy)

		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("X-Download-Options", "noopen")
		c.Header("X-Request-ID", enhancedGenerateRequestID())

		c.Next()
	}
}

func enhancedGenerateRequestID() string {
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

func GetEnhancedCSRFSecurity() *service.CSRFSecurity {
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
	return service.SanitizeHTML(input)
}

func DetectXSS(input string) (bool, string) {
	initXSSSecurity()
	return xssSecurity.DetectXSS(input)
}

type EnhancedInputValidationMiddlewareConfig struct {
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

var defaultEnhancedInputValidationConfig = EnhancedInputValidationMiddlewareConfig{
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

func EnhancedInputValidationMiddleware(configs ...EnhancedInputValidationMiddlewareConfig) gin.HandlerFunc {
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
	// 检查URL是否为空
	if urlStr == "" {
		return false
	}
	
	// 检查是否包含协议
	urlLower := strings.ToLower(urlStr)
	
	// 阻止危险协议
	dangerousProtocols := []string{
		"dict://",
		"ldap://",
		"sftp://",
		"imap://",
		"pop3://",
		"telnet://",
		"tftp://",
		"gopher://",
		"file://",
	}
	
	for _, proto := range dangerousProtocols {
		if strings.Contains(urlLower, proto) {
			return true
		}
	}
	
	// 检查内网IP和主机名
	privatePatterns := []string{
		"127.0.0.1",
		"localhost",
		"0.0.0.0",
		"::1",
		"[::1]",
		"[::ffff:",
		"169.254.169.254",
		"metadata.google.internal",
		"metadata.azure.com",
		"metadata.openstack.org",
	}
	
	for _, pattern := range privatePatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}
	
	// 检查私有IP范围
	ipPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|169\.254\.)`),
		regexp.MustCompile(`(?i)(127\.|0\.0\.0\.0)`),
	}
	
	for _, pattern := range ipPatterns {
		if pattern.MatchString(urlStr) {
			return true
		}
	}
	
	// 检查十六进制编码的IP
	hexIPPattern := regexp.MustCompile(`(?i)(0x[a-f0-9]{8})`)
	if hexIPPattern.MatchString(urlStr) {
		return true
	}
	
	// 检查点分十进制IP
	ipPattern := regexp.MustCompile(`(?i)(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	matches := ipPattern.FindAllString(urlStr, -1)
	for _, ip := range matches {
		if isPrivateIP(ip) {
			return true
		}
	}
	
	// 检查允许的域名列表
	if len(cfg.AllowedDomains) > 0 {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return true
		}
		
		host := parsedURL.Hostname()
		allowed := false
		for _, domain := range cfg.AllowedDomains {
			if host == domain || strings.HasSuffix(host, "."+domain) {
				allowed = true
				break
			}
		}
		
		if !allowed {
			return true
		}
	}
	
	// 检查阻止的IP范围
	if len(cfg.BlockedIPRanges) > 0 {
		for _, blockedRange := range cfg.BlockedIPRanges {
			if strings.Contains(urlStr, blockedRange) {
				return true
			}
		}
	}
	
	return false
}

func isPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.",
		"192.168.",
		"127.",
		"0.",
		"169.254.",
	}
	
	for _, range_ := range privateRanges {
		if strings.Contains(ip, range_) {
			return true
		}
	}
	
	return false
}
