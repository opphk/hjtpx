package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/service"
)

// TenantMiddleware 租户隔离中间件
func TenantMiddleware() gin.HandlerFunc {
	tenantService := service.NewTenantService()

	return func(c *gin.Context) {
		var tenant *model.Tenant
		var err error

		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID != "" {
			tenant, err = tenantService.GetTenant(c.Request.Context(), parseUint(tenantID))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID"})
				return
			}
		} else {
			host := c.Request.Host
			parts := strings.Split(host, ".")
			if len(parts) >= 3 {
				subdomain := parts[0]
				tenant, err = tenantService.GetTenantBySubdomain(c.Request.Context(), subdomain)
				if err != nil || tenant == nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tenant not found"})
					return
				}
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tenant identifier required"})
				return
			}
		}

		if tenant == nil || tenant.Status != "active" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tenant not found or inactive"})
			return
		}

		ctx := context.WithValue(c.Request.Context(), service.TenantContextKey{}, tenant)
		c.Request = c.Request.WithContext(ctx)
		c.Set("tenant", tenant)

		c.Next()
	}
}

// AdminTenantMiddleware 管理员租户管理中间件
func AdminTenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// parseUint 简单的字符串转uint
func parseUint(s string) uint {
	var n uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + uint(c-'0')
		}
	}
	return n
}
