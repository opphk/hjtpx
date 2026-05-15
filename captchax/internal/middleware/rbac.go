package middleware

import (
	"net/http"
	"strings"

	"captchax/internal/model"
	"captchax/internal/repository"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type RBACMiddleware struct {
	adminRoleRepo *repository.AdminRoleRepo
}

func NewRBACMiddleware(adminRoleRepo *repository.AdminRoleRepo) *RBACMiddleware {
	return &RBACMiddleware{
		adminRoleRepo: adminRoleRepo,
	}
}

func (m *RBACMiddleware) RequirePermission(permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, exists := c.Get("admin_id")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		hasPermission, err := m.adminRoleRepo.HasPermission(c.Request.Context(), int64(adminID.(uint)), permissionCode)
		if err != nil || !hasPermission {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *RBACMiddleware) RequireRole(roleCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, exists := c.Get("admin_id")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		for _, roleCode := range roleCodes {
			hasRole, err := m.adminRoleRepo.HasRole(c.Request.Context(), int64(adminID.(uint)), roleCode)
			if err == nil && hasRole {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "insufficient role privileges")
		c.Abort()
		return
	}
}

func (m *RBACMiddleware) RequireAnyPermission(permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, exists := c.Get("admin_id")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		for _, permCode := range permissionCodes {
			hasPermission, err := m.adminRoleRepo.HasPermission(c.Request.Context(), int64(adminID.(uint)), permCode)
			if err == nil && hasPermission {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "insufficient permissions")
		c.Abort()
		return
	}
}

func (m *RBACMiddleware) RequireAllPermissions(permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, exists := c.Get("admin_id")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		for _, permCode := range permissionCodes {
			hasPermission, err := m.adminRoleRepo.HasPermission(c.Request.Context(), int64(adminID.(uint)), permCode)
			if err != nil || !hasPermission {
				response.Forbidden(c, "insufficient permissions")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func (m *RBACMiddleware) SuperAdminOnly() gin.HandlerFunc {
	return m.RequireRole(string(model.AdminRoleSuper))
}

func (m *RBACMiddleware) AdminOnly() gin.HandlerFunc {
	return m.RequireRole(string(model.AdminRoleSuper), string(model.AdminRoleAdmin))
}

func (m *RBACMiddleware) OperatorOrAbove() gin.HandlerFunc {
	return m.RequireRole(string(model.AdminRoleSuper), string(model.AdminRoleAdmin), string(model.AdminRoleUser))
}

func RequirePermission(permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		hasPermission, exists := c.Get("has_permission_" + permissionCode)
		if exists && hasPermission.(bool) {
			c.Next()
			return
		}

		adminID, exists := c.Get("admin_id")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		adminRoleRepo, ok := c.Get("admin_role_repo")
		if !ok {
			response.InternalError(c, "service unavailable")
			c.Abort()
			return
		}

		repo := adminRoleRepo.(*repository.AdminRoleRepo)
		has, err := repo.HasPermission(c.Request.Context(), int64(adminID.(uint)), permissionCode)
		if err != nil || !has {
			response.Forbidden(c, "insufficient permissions: "+permissionCode)
			c.Abort()
			return
		}

		c.Set("has_permission_"+permissionCode, true)
		c.Next()
	}
}

func RequireRole(roleCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, exists := c.Get("admin_id")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		adminRoleRepo, ok := c.Get("admin_role_repo")
		if !ok {
			response.InternalError(c, "service unavailable")
			c.Abort()
			return
		}

		repo := adminRoleRepo.(*repository.AdminRoleRepo)

		for _, roleCode := range roleCodes {
			has, err := repo.HasRole(c.Request.Context(), int64(adminID.(uint)), roleCode)
			if err == nil && has {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "insufficient role privileges")
		c.Abort()
		return
	}
}

func SuperAdminOnly() gin.HandlerFunc {
	return RequireRole("super_admin")
}

func AdminOnly() gin.HandlerFunc {
	return RequireRole("super_admin", "admin")
}

type PermissionChecker struct {
	repo *repository.AdminRoleRepo
}

func NewPermissionChecker(repo *repository.AdminRoleRepo) *PermissionChecker {
	return &PermissionChecker{repo: repo}
}

func (p *PermissionChecker) CheckPermission(c *gin.Context, permissionCode string) bool {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return false
	}

	has, err := p.repo.HasPermission(c.Request.Context(), int64(adminID.(uint)), permissionCode)
	return err == nil && has
}

func (p *PermissionChecker) CheckRole(c *gin.Context, roleCode string) bool {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return false
	}

	has, err := p.repo.HasRole(c.Request.Context(), int64(adminID.(uint)), roleCode)
	return err == nil && has
}

func (p *PermissionChecker) GetPermissions(c *gin.Context) ([]*model.Permission, error) {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return nil, nil
	}

	return p.repo.GetAdminPermissions(c.Request.Context(), int64(adminID.(uint)))
}

func (p *PermissionChecker) GetRoles(c *gin.Context) ([]*model.Role, error) {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return nil, nil
	}

	return p.repo.GetAdminRoles(c.Request.Context(), int64(adminID.(uint)))
}

func ParsePermissions(permissionStr string) []string {
	if permissionStr == "" {
		return []string{}
	}
	return strings.Split(permissionStr, ",")
}

func JoinPermissions(permissions []string) string {
	return strings.Join(permissions, ",")
}

func PermissionStringContains(permissionStr string, permissionCode string) bool {
	permissions := ParsePermissions(permissionStr)
	for _, p := range permissions {
		if strings.TrimSpace(p) == permissionCode {
			return true
		}
	}
	return false
}

func PermissionStringContainsAny(permissionStr string, permissionCodes []string) bool {
	for _, code := range permissionCodes {
		if PermissionStringContains(permissionStr, code) {
			return true
		}
	}
	return false
}

func PermissionStringContainsAll(permissionStr string, permissionCodes []string) bool {
	for _, code := range permissionCodes {
		if !PermissionStringContains(permissionStr, code) {
			return false
		}
	}
	return true
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, code int, message string) {
	c.JSON(statusCode, APIResponse{
		Code:    code,
		Message: message,
	})
}
