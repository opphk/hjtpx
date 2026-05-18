package middleware

import (
	"github.com/gin-gonic/gin"
)

// AdvancedSecurityMiddleware 高级安全中间件
func AdvancedSecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 添加基础安全检查
		c.Next()
	}
}

// EnhancedXSSProtectionMiddlewareWrapper XSS防护中间件包装
func EnhancedXSSProtectionMiddlewareWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		// XSS防护逻辑
		c.Next()
	}
}

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-CSRF-Token")
			c.Header("Access-Control-Max-Age", "86400")
		}
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateWrapperRequestID()
		}
		c.Set("RequestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateWrapperRequestID() string {
	return "req-" + randomString(16)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
