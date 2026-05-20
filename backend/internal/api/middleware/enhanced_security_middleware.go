package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type SecurityMiddlewareConfig struct {
	EnableSQLInjectionProtection bool
	EnableXSSProtection          bool
	EnableCSRFProtection         bool
	EnableRBAC                   bool
	EnableAuditLogging           bool
	ExcludePaths                 []string
}

var defaultSecurityMiddlewareConfig = SecurityMiddlewareConfig{
	EnableSQLInjectionProtection: true,
	EnableXSSProtection:          true,
	EnableCSRFProtection:         true,
	EnableRBAC:                   true,
	EnableAuditLogging:           true,
	ExcludePaths:                 []string{"/health", "/metrics", "/api/health"},
}

var (
	sqlProtection  *service.EnhancedSQLInjectionProtection
	xssProtection  *service.EnhancedXSSProtection
	csrfService    *service.EnhancedCSRFService
	rbacService    *service.EnhancedRBACService
	auditService   *service.SecurityAuditService
	securityOnce   sync.Once
)

func initSecurityServices() {
	securityOnce.Do(func() {
		sqlProtection = service.NewEnhancedSQLInjectionProtection(nil)
		xssProtection = service.NewEnhancedXSSProtection(nil)
		csrfService = service.NewEnhancedCSRFService(nil)
		rbacService = service.NewEnhancedRBACService(nil)
		auditService = service.NewSecurityAuditService(nil)
	})
}

func SecurityMiddleware(configs ...SecurityMiddlewareConfig) gin.HandlerFunc {
	initSecurityServices()

	cfg := defaultSecurityMiddlewareConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		if cfg.EnableSQLInjectionProtection {
			validateSQLInjection(c, cfg)
		}

		if cfg.EnableXSSProtection {
			validateXSS(c, cfg)
		}

		if cfg.EnableAuditLogging {
			logSecurityEvent(c, cfg)
		}

		c.Next()
	}
}

func validateSQLInjection(c *gin.Context, cfg SecurityMiddlewareConfig) {
	query := c.Request.URL.Query()
	for _, values := range query {
		for _, value := range values {
			valid, pattern, severity := sqlProtection.ValidateInput(value)
			if !valid {
				auditService.LogSecurityEvent(&service.AuditLogEntry{
					UserID:    getUserID(c),
					Action:    "sql_injection_attempt",
					Resource:  c.Request.URL.Path,
					Result:    "blocked",
					IPAddress: c.ClientIP(),
					UserAgent: c.GetHeader("User-Agent"),
					Details:   "Pattern: " + pattern,
					RiskScore: 1.0,
				})

				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "security_violation",
					"code":    "SQL_INJECTION_DETECTED",
					"message": "Potential SQL injection attack detected",
					"pattern": pattern,
					"severity": severity,
				})
				return
			}
		}
	}

	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		if err := c.Request.ParseForm(); err == nil {
			for _, values := range c.Request.PostForm {
				for _, value := range values {
					valid, pattern, severity := sqlProtection.ValidateInput(value)
					if !valid {
						auditService.LogSecurityEvent(&service.AuditLogEntry{
							UserID:    getUserID(c),
							Action:    "sql_injection_attempt",
							Resource:  c.Request.URL.Path,
							Result:    "blocked",
							IPAddress: c.ClientIP(),
							UserAgent: c.GetHeader("User-Agent"),
							Details:   "Pattern: " + pattern,
							RiskScore: 1.0,
						})

						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
							"error":   "security_violation",
							"code":    "SQL_INJECTION_DETECTED",
							"message": "Potential SQL injection attack detected",
							"pattern": pattern,
							"severity": severity,
						})
						return
					}
				}
			}
		}
	}
}

func validateXSS(c *gin.Context, cfg SecurityMiddlewareConfig) {
	query := c.Request.URL.Query()
	for _, values := range query {
		for _, value := range values {
			detected, pattern, severity := xssProtection.DetectXSS(value)
			if detected {
				auditService.LogSecurityEvent(&service.AuditLogEntry{
					UserID:    getUserID(c),
					Action:    "xss_attempt",
					Resource:  c.Request.URL.Path,
					Result:    "blocked",
					IPAddress: c.ClientIP(),
					UserAgent: c.GetHeader("User-Agent"),
					Details:   "Pattern: " + pattern,
					RiskScore: 1.0,
				})

				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "security_violation",
					"code":    "XSS_DETECTED",
					"message": "Potential XSS attack detected",
					"pattern": pattern,
					"severity": severity,
				})
				return
			}
		}
	}

	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		if err := c.Request.ParseForm(); err == nil {
			for _, values := range c.Request.PostForm {
				for _, value := range values {
					detected, pattern, severity := xssProtection.DetectXSS(value)
					if detected {
						auditService.LogSecurityEvent(&service.AuditLogEntry{
							UserID:    getUserID(c),
							Action:    "xss_attempt",
							Resource:  c.Request.URL.Path,
							Result:    "blocked",
							IPAddress: c.ClientIP(),
							UserAgent: c.GetHeader("User-Agent"),
							Details:   "Pattern: " + pattern,
							RiskScore: 1.0,
						})

						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
							"error":   "security_violation",
							"code":    "XSS_DETECTED",
							"message": "Potential XSS attack detected",
							"pattern": pattern,
							"severity": severity,
						})
						return
					}
				}
			}
		}
	}
}

func logSecurityEvent(c *gin.Context, cfg SecurityMiddlewareConfig) {
	c.Set("security_event_start", time.Now())
}

func getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return "anonymous"
}

type RBACMiddlewareConfig struct {
	RequiredPermission string
	AllowedRoles       []string
}

func RequirePermission(permission string) gin.HandlerFunc {
	initSecurityServices()

	return func(c *gin.Context) {
		role := getUserRole(c)

		if !rbacService.HasPermission(role, permission) {
			auditService.LogSecurityEvent(&service.AuditLogEntry{
				UserID:    getUserID(c),
				Action:    "unauthorized_access_attempt",
				Resource:  c.Request.URL.Path,
				Result:    "blocked",
				IPAddress: c.ClientIP(),
				UserAgent: c.GetHeader("User-Agent"),
				Details:   "Required permission: " + permission,
				RiskScore: 0.8,
			})

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "access_denied",
				"code":    "INSUFFICIENT_PERMISSIONS",
				"message": "You do not have permission to perform this action",
				"required": permission,
			})
			return
		}

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	initSecurityServices()

	return func(c *gin.Context) {
		userRole := getUserRole(c)

		allowed := false
		for _, role := range roles {
			if userRole == role {
				allowed = true
				break
			}
		}

		if !allowed {
			auditService.LogSecurityEvent(&service.AuditLogEntry{
				UserID:    getUserID(c),
				Action:    "unauthorized_access_attempt",
				Resource:  c.Request.URL.Path,
				Result:    "blocked",
				IPAddress: c.ClientIP(),
				UserAgent: c.GetHeader("User-Agent"),
				Details:   "Required roles: " + strings.Join(roles, ", "),
				RiskScore: 0.8,
			})

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "access_denied",
				"code":    "INSUFFICIENT_ROLE",
				"message": "You do not have the required role to perform this action",
				"required": roles,
			})
			return
		}

		c.Next()
	}
}

func getUserRole(c *gin.Context) string {
	if role, exists := c.Get("user_role"); exists {
		if r, ok := role.(string); ok {
			return r
		}
	}

	if roleHeader := c.GetHeader("X-User-Role"); roleHeader != "" {
		return roleHeader
	}

	return "guest"
}
