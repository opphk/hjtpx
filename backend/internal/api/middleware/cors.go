package middleware

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowedOrigins           []string
	AllowedOriginFunc        func(origin string) bool
	AllowCredentials         bool
	AllowMethods             []string
	AllowHeaders             []string
	ExposeHeaders            []string
	MaxAge                   int
	AllowPrivateNetwork      bool
	SkipDynamicOrigins       bool
}

var (
	corsConfig     *CORSConfig
	corsConfigOnce sync.Once
	corsConfigMu   sync.RWMutex
)

var defaultCORSConfig = CORSConfig{
	AllowedOrigins:     []string{"*"},
	AllowCredentials:  true,
	AllowMethods:      []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"},
	AllowHeaders:      []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With", "X-App-ID", "X-Signature", "X-Timestamp", "X-Nonce", "X-Request-ID"},
	ExposeHeaders:     []string{"Content-Length", "X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
	MaxAge:            86400,
	AllowPrivateNetwork: false,
	SkipDynamicOrigins: false,
}

func loadCORSConfig() *CORSConfig {
	corsConfigOnce.Do(func() {
		origins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if origins == "" {
			origins = "*"
		}
		
		var allowedOrigins []string
		if origins != "*" {
			allowedOrigins = strings.Split(origins, ",")
		} else {
			allowedOrigins = []string{"*"}
		}

		allowCredentials := true
		if val := os.Getenv("CORS_ALLOW_CREDENTIALS"); val != "" {
			allowCredentials, _ = strconv.ParseBool(val)
		}

		allowMethods := os.Getenv("CORS_ALLOW_METHODS")
		if allowMethods == "" {
			allowMethods = "GET,POST,PUT,DELETE,PATCH,OPTIONS,HEAD"
		}

		allowHeaders := os.Getenv("CORS_ALLOW_HEADERS")
		if allowHeaders == "" {
			allowHeaders = "Origin,Content-Type,Content-Length,Accept-Encoding,X-CSRF-Token,Authorization,accept,origin,Cache-Control,X-Requested-With,X-App-ID,X-Signature,X-Timestamp,X-Nonce,X-Request-ID"
		}

		exposeHeaders := os.Getenv("CORS_EXPOSE_HEADERS")
		if exposeHeaders == "" {
			exposeHeaders = "Content-Length,X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset"
		}

		maxAge := 86400
		if val := os.Getenv("CORS_MAX_AGE"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil {
				maxAge = parsed
			}
		}

		corsConfig = &CORSConfig{
			AllowedOrigins:    allowedOrigins,
			AllowCredentials:  allowCredentials,
			AllowMethods:      strings.Split(allowMethods, ","),
			AllowHeaders:      strings.Split(allowHeaders, ","),
			ExposeHeaders:     strings.Split(exposeHeaders, ","),
			MaxAge:            maxAge,
			AllowPrivateNetwork: false,
			SkipDynamicOrigins: false,
		}
	})
	return corsConfig
}

func GetCORSConfig() *CORSConfig {
	corsConfigMu.RLock()
	defer corsConfigMu.RUnlock()
	if corsConfig == nil {
		return loadCORSConfig()
	}
	return corsConfig
}

func isOriginAllowed(origin string, config *CORSConfig) bool {
	for _, allowed := range config.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		}
	}
	return false
}

func CORS(options ...CORSConfig) gin.HandlerFunc {
	config := defaultCORSConfig
	if len(options) > 0 {
		config = options[0]
	} else {
		config = *loadCORSConfig()
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" && isOriginAllowed(origin, &config) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			
			if config.AllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Vary", "Origin")
			}
		} else if config.AllowedOrigins[0] == "*" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if len(config.AllowMethods) > 0 {
			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
		}

		if len(config.AllowHeaders) > 0 {
			c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
		}

		if len(config.ExposeHeaders) > 0 {
			c.Writer.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
		}

		if config.MaxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		}

		if config.AllowPrivateNetwork {
			c.Writer.Header().Set("Access-Control-Allow-Private-Network", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
			c.Writer.Header().Set("X-Frame-Options", "DENY")
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func CORSWithConfig(config CORSConfig) gin.HandlerFunc {
	return CORS(config)
}

func RelaxedCORS() gin.HandlerFunc {
	relaxedConfig := CORSConfig{
		AllowedOrigins:    []string{"*"},
		AllowCredentials:  true,
		AllowMethods:      []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"},
		AllowHeaders:      []string{"*"},
		ExposeHeaders:     []string{"*"},
		MaxAge:            86400,
	}
	return CORS(relaxedConfig)
}

func StrictCORS() gin.HandlerFunc {
	originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
	} else {
		allowedOrigins = []string{}
	}

	strictConfig := CORSConfig{
		AllowedOrigins:    allowedOrigins,
		AllowCredentials:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		MaxAge:           3600,
	}
	return CORS(strictConfig)
}
