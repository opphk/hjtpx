package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var blacklistService = service.NewRateLimitService()

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

		isBlacklisted, err := blacklistService.IsBlacklisted(c.Request.Context(), ip, BlacklistTypeIP)
		if err != nil {
			c.Next()
			return
		}

		if isBlacklisted {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "IP已被禁止访问",
				"error":   "ip_blacklisted",
			})
			c.Abort()
			return
		}

		isBanned, remaining, err := blacklistService.IsBanned(c.Request.Context(), ip, BlacklistTypeIP)
		if err == nil && isBanned {
			c.Header("Retry-After", remaining.String())
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "IP已被临时封禁",
				"error":   "ip_banned",
				"retry_after": remaining.String(),
			})
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

		isBlacklisted, err := blacklistService.IsBlacklisted(c.Request.Context(), string(rune(userID)), BlacklistTypeUser)
		if err != nil {
			c.Next()
			return
		}

		if isBlacklisted {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "用户已被禁止访问",
				"error":   "user_blacklisted",
			})
			c.Abort()
			return
		}

		isBanned, remaining, err := blacklistService.IsBanned(c.Request.Context(), string(rune(userID)), BlacklistTypeUser)
		if err == nil && isBanned {
			c.Header("Retry-After", remaining.String())
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "用户已被临时封禁",
				"error":   "user_banned",
				"retry_after": remaining.String(),
			})
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
			isBlacklisted, _ := blacklistService.IsBlacklisted(c.Request.Context(), ip, BlacklistTypeIP)
			if isBlacklisted {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "IP已被禁止访问",
					"error":   "ip_blacklisted",
				})
				c.Abort()
				return
			}

			isBanned, remaining, _ := blacklistService.IsBanned(c.Request.Context(), ip, BlacklistTypeIP)
			if isBanned {
				c.Header("Retry-After", remaining.String())
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "IP已被临时封禁",
					"error":   "ip_banned",
					"retry_after": remaining.String(),
				})
				c.Abort()
				return
			}
		}

		userID := GetUserID(c)
		if userID > 0 {
			userIDStr := formatUserID(userID)
			isBlacklisted, _ := blacklistService.IsBlacklisted(c.Request.Context(), userIDStr, BlacklistTypeUser)
			if isBlacklisted {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "用户已被禁止访问",
					"error":   "user_blacklisted",
				})
				c.Abort()
				return
			}

			isBanned, remaining, _ := blacklistService.IsBanned(c.Request.Context(), userIDStr, BlacklistTypeUser)
			if isBanned {
				c.Header("Retry-After", remaining.String())
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "用户已被临时封禁",
					"error":   "user_banned",
					"retry_after": remaining.String(),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func AutoBanMiddleware(violationThreshold int, banType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		if identifier == "" {
			identifier = c.GetHeader("X-Forwarded-For")
			if identifier == "" {
				identifier = c.GetHeader("X-Real-IP")
			}
		}

		if identifier == "" {
			c.Next()
			return
		}

		shouldBan, err := blacklistService.ShouldAutoBan(c.Request.Context(), identifier, banType, violationThreshold)
		if err != nil {
			c.Next()
			return
		}

		if shouldBan {
			err := blacklistService.AutoBan(c.Request.Context(), identifier, banType)
			if err == nil {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "检测到异常行为，IP已被临时封禁",
					"error":   "auto_banned",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func RecordViolationAndAutoBanMiddleware(violationType string, threshold int, banType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		if identifier == "" {
			identifier = c.GetHeader("X-Forwarded-For")
			if identifier == "" {
				identifier = c.GetHeader("X-Real-IP")
			}
		}

		if identifier == "" {
			c.Next()
			return
		}

		_, _ = blacklistService.RecordViolation(c.Request.Context(), identifier, violationType)

		shouldBan, _ := blacklistService.ShouldAutoBan(c.Request.Context(), identifier, banType, threshold)
		if shouldBan {
			_ = blacklistService.AutoBan(c.Request.Context(), identifier, banType)
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "检测到异常行为，IP已被临时封禁",
				"error":   "auto_banned",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func AddToBlacklist(c *gin.Context, identifier string, blacklistType string, duration time.Duration) error {
	return blacklistService.AddToBlacklist(c.Request.Context(), identifier, blacklistType, duration)
}

func RemoveFromBlacklist(c *gin.Context, identifier string, blacklistType string) error {
	return blacklistService.RemoveFromBlacklist(c.Request.Context(), identifier, blacklistType)
}

func AddBan(c *gin.Context, identifier string, banType string, duration time.Duration) error {
	return blacklistService.BanIdentifier(c.Request.Context(), identifier, banType, duration)
}

func RemoveBan(c *gin.Context, identifier string, banType string) error {
	return blacklistService.UnbanIdentifier(c.Request.Context(), identifier, banType)
}

func formatUserID(userID uint) string {
	return fmt.Sprintf("%d", userID)
}
