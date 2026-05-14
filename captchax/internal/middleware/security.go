package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type SecurityConfig struct {
	HTTPSRedirect       bool
	FrameOptions        string
	ContentTypeOptions  string
	XSSProtection       string
	ContentSecurity     string
	ReferrerPolicy      string
	PermissionsPolicy   string
	StrictTransport     string
	CSRFEnabled         bool
	CSRFTokenLength     int
	CSRFTokenHeader     string
	CSRFFormField       string
	AllowedOrigins      []string
	TrustedOrigins      []string
}

var defaultSecurityConfig = &SecurityConfig{
	HTTPSRedirect:      true,
	FrameOptions:       "SAMEORIGIN",
	ContentTypeOptions: "nosniff",
	XSSProtection:      "1; mode=block",
	ReferrerPolicy:     "strict-origin-when-cross-origin",
	PermissionsPolicy:  "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
	StrictTransport:    "max-age=31536000; includeSubDomains",
	CSRFEnabled:        false,
	CSRFTokenLength:    32,
	CSRFTokenHeader:    "X-CSRF-Token",
	CSRFFormField:      "csrf_token",
	AllowedOrigins:     []string{},
	TrustedOrigins:     []string{},
}

func Security() gin.HandlerFunc {
	return SecurityWithConfig(defaultSecurityConfig)
}

func SecurityWithConfig(config *SecurityConfig) gin.HandlerFunc {
	if config.FrameOptions == "" {
		config.FrameOptions = defaultSecurityConfig.FrameOptions
	}
	if config.ContentTypeOptions == "" {
		config.ContentTypeOptions = defaultSecurityConfig.ContentTypeOptions
	}
	if config.XSSProtection == "" {
		config.XSSProtection = defaultSecurityConfig.XSSProtection
	}
	if config.ReferrerPolicy == "" {
		config.ReferrerPolicy = defaultSecurityConfig.ReferrerPolicy
	}
	if config.PermissionsPolicy == "" {
		config.PermissionsPolicy = defaultSecurityConfig.PermissionsPolicy
	}
	if config.StrictTransport == "" {
		config.StrictTransport = defaultSecurityConfig.StrictTransport
	}

	return func(c *gin.Context) {
		if config.HTTPSRedirect && c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") != "https" {
			if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
				secureURL := "https://" + c.Request.Host + c.Request.URL.String()
				c.Redirect(http.StatusMovedPermanently, secureURL)
				c.Abort()
				return
			}
		}

		c.Header("X-Frame-Options", config.FrameOptions)
		c.Header("X-Content-Type-Options", config.ContentTypeOptions)
		c.Header("X-XSS-Protection", config.XSSProtection)
		c.Header("Referrer-Policy", config.ReferrerPolicy)
		c.Header("Permissions-Policy", config.PermissionsPolicy)

		origin := c.GetHeader("Origin")
		if origin != "" && config.isOriginAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		c.Next()

		c.Header("X-Content-Type-Options", config.ContentTypeOptions)
		c.Header("Strict-Transport-Security", config.StrictTransport)
	}
}

func (s *SecurityConfig) isOriginAllowed(origin string) bool {
	for _, allowed := range s.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	for _, trusted := range s.TrustedOrigins {
		if trusted == origin {
			return true
		}
	}
	return false
}

func CSRFProtection() gin.HandlerFunc {
	return CSRFWithConfig(defaultSecurityConfig)
}

func CSRFWithConfig(config *SecurityConfig) gin.HandlerFunc {
	if config.CSRFTokenLength <= 0 {
		config.CSRFTokenLength = defaultSecurityConfig.CSRFTokenLength
	}
	if config.CSRFTokenHeader == "" {
		config.CSRFTokenHeader = defaultSecurityConfig.CSRFTokenHeader
	}
	if config.CSRFFormField == "" {
		config.CSRFFormField = defaultSecurityConfig.CSRFFormField
	}

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			csrfToken := generateCSRFToken(config.CSRFTokenLength)
			c.SetCookie("csrf_token", csrfToken, 3600, "/", "", false, true)
			c.Header("X-CSRF-Token", csrfToken)
			c.Set("csrf_token", csrfToken)
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = c.GetHeader("Referer")
		}

		if !isTrustedOrigin(origin, config.TrustedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "csrf: invalid origin",
			})
			return
		}

		tokenFromHeader := c.GetHeader(config.CSRFTokenHeader)
		tokenFromCookie, _ := c.Cookie("csrf_token")
		tokenFromForm := c.PostForm(config.CSRFFormField)

		validToken := tokenFromHeader
		if validToken == "" {
			validToken = tokenFromCookie
		}
		if validToken == "" {
			validToken = tokenFromForm
		}

		cookieToken, exists := c.Get("csrf_token")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "csrf: token not found",
			})
			return
		}

		if !secureCompare(validToken, cookieToken.(string)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "csrf: token mismatch",
			})
			return
		}

		newToken := generateCSRFToken(config.CSRFTokenLength)
		c.SetCookie("csrf_token", newToken, 3600, "/", "", false, true)
		c.Header("X-CSRF-Token", newToken)
		c.Set("csrf_token", newToken)

		c.Next()
	}
}

func generateCSRFToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, length)
	sum := 0
	for i := range token {
		hash := sha256.Sum256([]byte(string(rune(i)) + string(rune(sum))))
		token[i] = charset[int(hash[0])%len(charset)]
		sum += int(hash[0])
	}
	return base64.URLEncoding.EncodeToString(token)[:length]
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func isTrustedOrigin(origin string, trustedOrigins []string) bool {
	if origin == "" {
		return true
	}
	for _, trusted := range trustedOrigins {
		if trusted == origin {
			return true
		}
	}
	return len(trustedOrigins) == 0
}

func XSSProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Content-Security-Policy", "default-src 'self'")
		c.Header("X-WebKit-CSP", "default-src 'self'")
		c.Next()
	}
}
