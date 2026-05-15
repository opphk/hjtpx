package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		fmt.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n",
			start.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)

		if len(query) > 0 {
			fmt.Printf("  Query: %s\n", query)
		}

		if len(c.Errors) > 0 {
			fmt.Printf("  Errors: %v\n", c.Errors.String())
		}
	}
}
