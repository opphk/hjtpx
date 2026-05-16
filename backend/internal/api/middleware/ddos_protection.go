package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	ddosProtectionService *service.DDoSProtectionService
	ddosProtectionOnce    = &sync.Once{}
)

func initDDOSProtection() {
	ddosProtectionOnce.Do(func() {
		ddosProtectionService = service.NewDDoSProtectionService()
	})
}

type DDOSProtectionConfig struct {
	Enabled         bool
	ExcludePaths    []string
	EnableWhitelist bool
}

var DefaultDDOSProtectionConfig = DDOSProtectionConfig{
	Enabled:         true,
	EnableWhitelist: false,
}

func DDOSProtectionMiddleware(config ...DDOSProtectionConfig) gin.HandlerFunc {
	initDDOSProtection()

	cfg := DefaultDDOSProtectionConfig
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
			if path == excluded || pathHasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		result := ddosProtectionService.CheckRequest(c.Request)
		c.Set("ddos_result", result)

		if !result.Allowed {
			response := gin.H{
				"error": "Request blocked",
				"code":  http.StatusTooManyRequests,
			}
			if result.RetryAfter > 0 {
				response["retry_after"] = result.RetryAfter
				c.Header("Retry-After", string(rune(result.RetryAfter)))
			}
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response)
			return
		}

		c.Next()
	}
}

func GetDDOSProtectionService() *service.DDoSProtectionService {
	initDDOSProtection()
	return ddosProtectionService
}
