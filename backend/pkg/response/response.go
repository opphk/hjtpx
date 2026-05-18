package response

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, Response{
		Code:    httpStatus,
		Message: message,
	})
}

// ErrorWithLog 错误响应并记录日志
func ErrorWithLog(c *gin.Context, httpStatus int, message string, err error) {
	if err != nil {
		log.Printf("[ERROR] %s: %v (Path: %s, Method: %s)", message, err, c.Request.URL.Path, c.Request.Method)
	}
	Error(c, httpStatus, message)
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "unauthorized")
}

// Forbidden 403错误
func Forbidden(c *gin.Context) {
	Error(c, http.StatusForbidden, "forbidden")
}

// NotFound 404错误
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	Error(c, http.StatusNotFound, message)
}

// InternalServerError 500错误
func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	Error(c, http.StatusInternalServerError, message)
}

// TooManyRequests 429错误
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "too many requests"
	}
	Error(c, http.StatusTooManyRequests, message)
}

const (
	CodeSuccess         = 0
	CodeInvalidParams   = 400
	CodeUnauthorized    = 401
	CodeForbidden       = 403
	CodeNotFound        = 404
	CodeServerError     = 500
	CodeTooManyRequests = 429
)

func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}
