package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var blacklistService = service.NewBlacklistService()

type BlacklistOptions struct {
	BanDuration time.Duration
	Reason      string
}

const (
	BlacklistTypeIP   = "ip"
	BlacklistTypeUser = "user"
)

func IPBlacklistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		if ip == "" {
			c.Next()
			return
		}

		isBlacklisted, err := blacklistService.CheckBlacklist(ip, BlacklistTypeIP)
		if err != nil {
			c.Next()
			return
		}

		if isBlacklisted {
			response.Forbidden(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

func UserBlacklistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.Next()
			return
		}

		userIDStr := fmt.Sprintf("%d", userID)
		isBlacklisted, err := blacklistService.CheckBlacklist(userIDStr, BlacklistTypeUser)
		if err != nil {
			c.Next()
			return
		}

		if isBlacklisted {
			response.Forbidden(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

func CombinedBlacklistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		if ip != "" {
			isBlacklisted, _ := blacklistService.CheckBlacklist(ip, BlacklistTypeIP)
			if isBlacklisted {
				response.Forbidden(c)
				c.Abort()
				return
			}
		}

		userID := GetUserID(c)
		if userID > 0 {
			userIDStr := fmt.Sprintf("%d", userID)
			isBlacklisted, _ := blacklistService.CheckBlacklist(userIDStr, BlacklistTypeUser)
			if isBlacklisted {
				response.Forbidden(c)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}



func formatUserID(userID uint) string {
	return fmt.Sprintf("%d", userID)
}
