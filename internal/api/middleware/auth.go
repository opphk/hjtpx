package middleware

import (
	"strings"

	"hjtpx/internal/config"
	"hjtpx/internal/utils"

	"github.com/gin-gonic/gin"
)

func Auth(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			utils.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("app_id", claims.AppID)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

func AdminAuth(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			utils.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		if claims.Role != "admin" {
			utils.Forbidden(c, "Admin access required")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("app_id", claims.AppID)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(uint)
	}
	return 0
}

func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}

func GetAppID(c *gin.Context) uint {
	if appID, exists := c.Get("app_id"); exists {
		return appID.(uint)
	}
	return 0
}

func InitAuthMiddleware(cfg config.JWTConfig) *utils.JWTManager {
	return utils.NewJWTManager(
		cfg.Secret,
		cfg.ExpirationTime(),
		cfg.Issuer,
	)
}
