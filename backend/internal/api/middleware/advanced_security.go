package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type HTTPSConfig struct {
	Enabled       bool
	ExcludePaths  []string
	RedirectCode  int
}

var defaultHTTPSConfig = HTTPSConfig{
	Enabled:      true,
	RedirectCode: http.StatusMovedPermanently,
}

func HTTPSRedirect(config ...HTTPSConfig) gin.HandlerFunc {
	cfg := defaultHTTPSConfig
	if len(config) > 0 {
		cfg = config[0]
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

		if c.Request.TLS != nil {
			c.Next()
			return
		}

		proto := c.GetHeader("X-Forwarded-Proto")
		if proto == "https" {
			c.Next()
			return
		}

		proto = c.GetHeader("X-Forwarded-Scheme")
		if proto == "https" {
			c.Next()
			return
		}

		host := c.Request.Host
		if host == "" {
			host = c.Request.URL.Host
		}

		httpsURL := fmt.Sprintf("https://%s%s", host, c.Request.URL.String())

		c.Redirect(cfg.RedirectCode, httpsURL)
		c.Abort()
	}
}

type InputValidationMiddlewareConfig struct {
	Enabled       bool
	ValidateQuery bool
	ValidateForm  bool
	ValidateJSON  bool
	ExcludePaths  []string
	CustomRules   map[string]func(string) bool
}

var defaultInputValidationConfig = InputValidationMiddlewareConfig{
	Enabled:       true,
	ValidateQuery: true,
	ValidateForm:  true,
	ValidateJSON:  false,
}

func InputValidationMiddleware(config ...InputValidationMiddlewareConfig) gin.HandlerFunc {
	cfg := defaultInputValidationConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	securityService := service.NewSecurityService(nil)
	validator := service.NewRequestValidator()

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

		if cfg.ValidateQuery {
			for key, values := range c.Request.URL.Query() {
				for _, value := range values {
					sanitized := securityService.SanitizeInput(value)
					if sanitized != value {
						securityService.IncrementMetric("sql_injection")
					}

					if err := validator.Validate(key, sanitized, "required"); err == nil {
						c.Set(fmt.Sprintf("validated_%s", key), sanitized)
					}
				}
			}
		}

		if cfg.ValidateForm && (c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut) {
			if err := c.Request.ParseForm(); err == nil {
				for key, values := range c.Request.PostForm {
					for _, value := range values {
						sanitized := securityService.SanitizeInput(value)
						sanitized = securityService.SanitizeHTML(sanitized)

						if sanitized != value {
							securityService.IncrementMetric("sql_injection")
							securityService.IncrementMetric("xss")
						}

						if err := validator.Validate(key, sanitized, "required"); err == nil {
							c.Set(fmt.Sprintf("validated_%s", key), sanitized)
						}
					}
				}
			}
		}

		c.Next()
	}
}

type XSSProtectionConfig struct {
	Enabled      bool
	ExcludePaths []string
}

var defaultXSSProtectionConfig = XSSProtectionConfig{
	Enabled: true,
}

func XSSProtectionMiddleware(config ...XSSProtectionConfig) gin.HandlerFunc {
	cfg := defaultXSSProtectionConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	securityService := service.NewSecurityService(nil)

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

		if c.Request.Body != nil {
			body := make([]byte, 0)
			c.Request.Body.Read(body)
			c.Request.Body = newReadCloser(body)

			sanitized := securityService.SanitizeHTML(string(body))
			if sanitized != string(body) {
				securityService.IncrementMetric("xss")
			}

			c.Set("sanitized_body", sanitized)
		}

		c.Next()
	}
}

type readCloser struct {
	data []byte
	pos  int
}

func newReadCloser(data []byte) *readCloser {
	return &readCloser{data: data, pos: 0}
}

func (rc *readCloser) Read(p []byte) (n int, err error) {
	if rc.pos >= len(rc.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, rc.data[rc.pos:])
	rc.pos += n
	return n, nil
}

func (rc *readCloser) Close() error {
	return nil
}

type SecurityHeadersMiddlewareConfig struct {
	Enabled             bool
	CSP                 string
	HSTS                string
	XFrameOptions       string
	XContentTypeOptions string
	XSSProtection       string
	ReferrerPolicy      string
	PermissionsPolicy   string
	ExcludePaths        []string
}

var defaultSecurityHeadersConfig = SecurityHeadersMiddlewareConfig{
	Enabled:             true,
	CSP:                 "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' https:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
	HSTS:                "max-age=31536000; includeSubDomains; preload",
	XFrameOptions:       "DENY",
	XContentTypeOptions: "nosniff",
	XSSProtection:       "1; mode=block",
	ReferrerPolicy:      "strict-origin-when-cross-origin",
	PermissionsPolicy:   "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
}

func SecurityHeadersMiddleware(config ...SecurityHeadersMiddlewareConfig) gin.HandlerFunc {
	cfg := defaultSecurityHeadersConfig
	if len(config) > 0 {
		cfg = config[0]
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

		if cfg.CSP != "" {
			c.Header("Content-Security-Policy", cfg.CSP)
		}

		if cfg.HSTS != "" {
			c.Header("Strict-Transport-Security", cfg.HSTS)
		}

		if cfg.XFrameOptions != "" {
			c.Header("X-Frame-Options", cfg.XFrameOptions)
		}

		if cfg.XContentTypeOptions != "" {
			c.Header("X-Content-Type-Options", cfg.XContentTypeOptions)
		}

		if cfg.XSSProtection != "" {
			c.Header("X-XSS-Protection", cfg.XSSProtection)
		}

		if cfg.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", cfg.ReferrerPolicy)
		}

		if cfg.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", cfg.PermissionsPolicy)
		}

		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("X-Download-Options", "noopen")

		c.Next()
	}
}

type CSRFTokenMiddlewareConfig struct {
	Enabled        bool
	HeaderName     string
	FormFieldName  string
	CookieName     string
	ExcludePaths   []string
	SafeMethods    []string
	RedisEnabled   bool
}

var defaultCSRFTokenConfig = CSRFTokenMiddlewareConfig{
	Enabled:       true,
	HeaderName:    "X-CSRF-Token",
	FormFieldName: "_csrf",
	CookieName:    "csrf_token",
	SafeMethods:   []string{"GET", "HEAD", "OPTIONS", "TRACE"},
	RedisEnabled:  false,
}

var (
	csrfTokenStore = make(map[string]map[string]time.Time)
	csrfMu         sync.RWMutex
)

func CSRFTokenMiddleware(config ...CSRFTokenMiddlewareConfig) gin.HandlerFunc {
	cfg := defaultCSRFTokenConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	csrfSecurity := service.NewCSRFSecurity(nil)

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
				sessionID := generateCSRFCSID(c)

				token, err := csrfSecurity.GenerateToken()
				if err == nil {
					c.Set("csrf_token", token)
					c.Set("csrf_session_id", sessionID)
					c.Header("X-CSRF-Token", token)
					c.SetCookie(
						cfg.CookieName,
						token,
						int(24*time.Hour.Seconds()),
						"/",
						"",
						true,
						true,
					)

					csrfMu.Lock()
					if csrfTokenStore[sessionID] == nil {
						csrfTokenStore[sessionID] = make(map[string]time.Time)
					}
					hashedToken := hashCSRFToken(token)
					csrfTokenStore[sessionID][hashedToken] = time.Now().Add(24 * time.Hour)
					csrfMu.Unlock()
				}
			}
			c.Next()
			return
		}

		sessionID := generateCSRFCSID(c)

		token := c.GetHeader(cfg.HeaderName)
		if token == "" {
			token = c.Query(cfg.FormFieldName)
		}
		if token == "" {
			token = c.PostForm(cfg.FormFieldName)
		}
		if token == "" {
			if cookie, err := c.Cookie(cfg.CookieName); err == nil {
				token = cookie
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_missing",
				"message": "CSRF token is required",
			})
			return
		}

		valid := validateCSRFToken(sessionID, token)
		if !valid {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_invalid",
				"message": "Invalid or expired CSRF token",
			})
			return
		}

		invalidateCSRFToken(sessionID, token)

		c.Next()
	}
}

func generateCSRFCSID(c *gin.Context) string {
	if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0]) + ":" + c.ClientIP()
	}

	return c.ClientIP() + ":" + c.GetHeader("User-Agent")
}

func hashCSRFToken(token string) string {
	hash := service.NewConfigEncryptor(token)
	hashed, _ := hash.Encrypt(token)
	return hashed
}

func validateCSRFToken(sessionID, token string) bool {
	csrfMu.RLock()
	defer csrfMu.RUnlock()

	sessionTokens, ok := csrfTokenStore[sessionID]
	if !ok {
		return false
	}

	hashedToken := hashCSRFToken(token)
	expiry, ok := sessionTokens[hashedToken]
	if !ok {
		return false
	}

	if time.Now().After(expiry) {
		return false
	}

	return true
}

func invalidateCSRFToken(sessionID, token string) {
	csrfMu.Lock()
	defer csrfMu.Unlock()

	hashedToken := hashCSRFToken(token)
	delete(csrfTokenStore[sessionID], hashedToken)
}

type RateLimitMiddlewareConfig struct {
	Enabled       bool
	MaxRequests   int
	Window        time.Duration
	ExcludePaths  []string
	KeyFunc       func(*gin.Context) string
}

var defaultRateLimitConfig = RateLimitMiddlewareConfig{
	Enabled:     true,
	MaxRequests: 100,
	Window:      1 * time.Minute,
	KeyFunc: func(c *gin.Context) string {
		return c.ClientIP()
	},
}

func RateLimitMiddleware(config ...RateLimitMiddlewareConfig) gin.HandlerFunc {
	cfg := defaultRateLimitConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	type clientRequest struct {
		count    int
		resetAt  time.Time
	}

	requests := make(map[string]*clientRequest)
	var mu sync.RWMutex

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

		key := cfg.KeyFunc(c)

		mu.Lock()
		client, exists := requests[key]
		now := time.Now()

		if !exists || now.After(client.resetAt) {
			requests[key] = &clientRequest{
				count:   1,
				resetAt: now.Add(cfg.Window),
			}
			mu.Unlock()
			c.Next()
			return
		}

		client.count++
		if client.count > cfg.MaxRequests {
			retryAfter := int(time.Until(client.resetAt).Seconds())
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.MaxRequests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", client.resetAt.Unix()))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests",
				"retry_after": retryAfter,
			})
			mu.Unlock()
			return
		}

		remaining := cfg.MaxRequests - client.count
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", client.resetAt.Unix()))

		mu.Unlock()
		c.Next()
	}
}

func SetupSecurityMiddleware(r *gin.Engine) {
	// Disable HTTPS redirect for development environment
	r.Use(HTTPSRedirect(HTTPSConfig{Enabled: false}))

	r.Use(SecurityHeadersMiddleware())

	r.Use(CORS())

	r.Use(CSRFTokenMiddleware())

	r.Use(RateLimitMiddleware())

	r.Use(InputValidationMiddleware())

	r.Use(XSSProtectionMiddleware())
}
