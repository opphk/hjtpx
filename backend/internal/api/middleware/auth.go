package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type UserClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

type AdminClaims struct {
	AdminID  uint   `json:"admin_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token := parts[1]

		if redis.Client != nil {
			ctx := c.Request.Context()
			loggedOut, err := redis.Client.Get(ctx, "logout:"+token).Result()
			if err == nil && loggedOut == "1" {
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		claims, err := jwt.ParseToken(token)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func AuthMiddlewareWithRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token := parts[1]

		if redis.Client != nil {
			ctx := c.Request.Context()
			loggedOut, err := redis.Client.Get(ctx, "logout:"+token).Result()
			if err == nil && loggedOut == "1" {
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		claims, err := jwt.ParseToken(token)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if len(allowedRoles) > 0 {
			hasRole := false
			for _, role := range allowedRoles {
				if role == claims.Username {
					hasRole = true
					break
				}
			}
			if !hasRole {
				response.Forbidden(c)
				c.Abort()
				return
			}
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func UserAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token := parts[1]

		if redis.Client != nil {
			ctx := c.Request.Context()
			loggedOut, err := redis.Client.Get(ctx, "user_logout:"+token).Result()
			if err == nil && loggedOut == "1" {
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		claims, err := jwt.ParseUserToken(token)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(uint)
	}
	if adminID, exists := c.Get("admin_id"); exists {
		return adminID.(uint)
	}
	return 0
}

func GetUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return ""
}

func GetAdminID(c *gin.Context) uint {
	if adminID, exists := c.Get("admin_id"); exists {
		return adminID.(uint)
	}
	return 0
}

func IsSuperAdmin(c *gin.Context) bool {
	if isSuper, exists := c.Get("is_super_admin"); exists {
		return isSuper.(bool)
	}
	return false
}

func GetRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}
