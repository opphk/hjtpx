package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
)

var (
	owaspService *service.OWASPService
	owaspOnce    = &sync.Once{}
)

func initOWASPService() {
	owaspOnce.Do(func() {
		owaspService = service.NewOWASPService()
	})
}

type OWASPConfig struct {
	Enabled           bool
	EnforceHeaders    bool
	EnforceHTTPS      bool
	BlockNonCompliant bool
}

var DefaultOWASPConfig = OWASPConfig{
	Enabled:           true,
	EnforceHeaders:    true,
	EnforceHTTPS:      false,
	BlockNonCompliant: false,
}

func OWASPSecurityMiddleware(config ...OWASPConfig) gin.HandlerFunc {
	initOWASPService()

	cfg := DefaultOWASPConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		if cfg.EnforceHeaders {
			setSecurityHeaders(c.Writer)
		}

		if cfg.EnforceHTTPS {
			if c.Request.TLS == nil && c.Request.Header.Get("X-Forwarded-Proto") != "https" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "HTTPS required",
					"code":  http.StatusForbidden,
				})
				return
			}
		}

		compliance := owaspService.CheckCompliance(c.Request)
		c.Set("owasp_compliance", compliance)

		if cfg.BlockNonCompliant && !compliance["compliant"].(bool) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":      "OWASP compliance check failed",
				"code":       http.StatusBadRequest,
				"compliance": compliance,
			})
			return
		}

		c.Set("owasp_service", owaspService)
		c.Next()
	}
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
}

func GetOWASPService() *service.OWASPService {
	initOWASPService()
	return owaspService
}
