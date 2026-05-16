package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				fmt.Printf("[Recovery] panic recovered:\n%v\n%s\n", err, stack)

				response.InternalServerError(c, "Internal Server Error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
