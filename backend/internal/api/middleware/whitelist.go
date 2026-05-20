package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
)

var whitelistService = service.NewRateLimitService()

const (
	WhitelistTypeIP   = "ip"
	WhitelistTypeUser = "user"
)

func IPWhitelistMiddleware(whitelistIPs []string) gin.HandlerFunc {
	whitelistMap := make(map[string]bool)
	for _, ip := range whitelistIPs {
		whitelistMap[ip] = true
	}

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

		if whitelistMap[ip] {
			c.Set("whitelisted", true)
			c.Next()
			return
		}

		isWhitelisted, err := whitelistService.IsWhitelisted(c.Request.Context(), ip, WhitelistTypeIP)
		if err == nil && isWhitelisted {
			c.Set("whitelisted", true)
			c.Next()
			return
		}

		c.Next()
	}
}

func UserWhitelistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.Next()
			return
		}

		userIDStr := formatUserIDForWhitelist(userID)
		isWhitelisted, err := whitelistService.IsWhitelisted(c.Request.Context(), userIDStr, WhitelistTypeUser)
		if err != nil {
			c.Next()
			return
		}

		if isWhitelisted {
			c.Set("whitelisted", true)
			c.Next()
			return
		}

		c.Next()
	}
}

func SkipRateLimitForWhitelist() gin.HandlerFunc {
	return func(c *gin.Context) {
		if whitelisted, exists := c.Get("whitelisted"); exists {
			if isWhitelisted, ok := whitelisted.(bool); ok && isWhitelisted {
				c.Header("X-RateLimit-Skip", "true")
			}
		}
		c.Next()
	}
}

func WhitelistBypassMiddleware(whitelistIPs []string) gin.HandlerFunc {
	whitelistMap := make(map[string]bool)
	for _, ip := range whitelistIPs {
		whitelistMap[ip] = true
	}

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

		if whitelistMap[ip] {
			c.Set("whitelisted", true)
			c.Set("rate_limit_bypass", true)
			c.Next()
			return
		}

		isWhitelisted, err := whitelistService.IsWhitelisted(c.Request.Context(), ip, WhitelistTypeIP)
		if err == nil && isWhitelisted {
			c.Set("whitelisted", true)
			c.Set("rate_limit_bypass", true)
			c.Next()
			return
		}

		userID := GetUserID(c)
		if userID > 0 {
			userIDStr := formatUserIDForWhitelist(userID)
			isUserWhitelisted, err := whitelistService.IsWhitelisted(c.Request.Context(), userIDStr, WhitelistTypeUser)
			if err == nil && isUserWhitelisted {
				c.Set("whitelisted", true)
				c.Set("rate_limit_bypass", true)
				c.Next()
				return
			}
		}

		c.Next()
	}
}

func CombinedWhitelistMiddleware(whitelistIPs []string) gin.HandlerFunc {
	whitelistMap := make(map[string]bool)
	for _, ip := range whitelistIPs {
		whitelistMap[ip] = true
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		if ip != "" {
			if whitelistMap[ip] {
				c.Set("whitelisted", true)
				c.Next()
				return
			}

			isIPWhitelisted, err := whitelistService.IsWhitelisted(c.Request.Context(), ip, WhitelistTypeIP)
			if err == nil && isIPWhitelisted {
				c.Set("whitelisted", true)
				c.Next()
				return
			}
		}

		userID := GetUserID(c)
		if userID > 0 {
			userIDStr := formatUserIDForWhitelist(userID)
			isUserWhitelisted, err := whitelistService.IsWhitelisted(c.Request.Context(), userIDStr, WhitelistTypeUser)
			if err == nil && isUserWhitelisted {
				c.Set("whitelisted", true)
				c.Next()
				return
			}
		}

		c.Next()
	}
}

func WhitelistChecker() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		isIPWhitelisted := false
		if ip != "" {
			isWhitelisted, _ := whitelistService.IsWhitelisted(c.Request.Context(), ip, WhitelistTypeIP)
			isIPWhitelisted = isWhitelisted
		}

		isUserWhitelisted := false
		userID := GetUserID(c)
		if userID > 0 {
			userIDStr := formatUserIDForWhitelist(userID)
			isWhitelisted, _ := whitelistService.IsWhitelisted(c.Request.Context(), userIDStr, WhitelistTypeUser)
			isUserWhitelisted = isWhitelisted
		}

		c.Set("ip_whitelisted", isIPWhitelisted)
		c.Set("user_whitelisted", isUserWhitelisted)

		if isIPWhitelisted || isUserWhitelisted {
			c.Set("whitelisted", true)
		}

		c.Next()
	}
}

func AddToWhitelist(c *gin.Context, identifier string, whitelistType string) error {
	return whitelistService.AddToWhitelist(c.Request.Context(), identifier, whitelistType, 0)
}

func RemoveFromWhitelist(c *gin.Context, identifier string, whitelistType string) error {
	return whitelistService.RemoveFromWhitelist(c.Request.Context(), identifier, whitelistType)
}

func formatUserIDForWhitelist(userID uint) string {
	return fmt.Sprintf("%d", userID)
}
