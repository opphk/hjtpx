package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// ErrorHandler 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			response.InternalServerError(c, err.Error())
		}
	}
}
