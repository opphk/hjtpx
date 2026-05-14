package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

var defaultCORSConfig = &CORSConfig{
	AllowOrigins: []string{"*"},
	AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
	AllowHeaders: []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"X-Requested-With",
		"X-CSRF-Token",
		"X-Request-ID",
	},
	ExposeHeaders:    []string{"Content-Length", "Content-Type"},
	AllowCredentials: true,
	MaxAge:           86400,
}

func CORS() gin.HandlerFunc {
	return CORSWithConfig(defaultCORSConfig)
}

func CORSWithConfig(config *CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if !isOriginAllowed(origin, config.AllowOrigins) {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
		c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Max-Age", formatInt(config.MaxAge))
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(allowed, origin) {
			return true
		}
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if negative {
		result = "-" + result
	}
	return result
}
