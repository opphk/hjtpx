package handler

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type ErrorInfo struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
	Field       string `json:"field,omitempty"`
}

type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   ErrorInfo  `json:"error"`
	TraceID string     `json:"trace_id,omitempty"`
	Time    time.Time  `json:"time"`
}

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			handleError(c, http.StatusInternalServerError, err.Error())
		}
	}
}

func RecoveryWithCustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				fmt.Printf("[PANIC RECOVERED] %v\n%s\n", err, stack)

				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Success: false,
					Error: ErrorInfo{
						Code:    http.StatusInternalServerError,
						Message: "internal server error",
					},
					TraceID: getTraceID(c),
					Time:    time.Now(),
				})
			}
		}()

		c.Next()
	}
}

func NotFoundHandler(c *gin.Context) {
	requestID := ""
	if rid, exists := c.Get("request_id"); exists {
		requestID = rid.(string)
	}

	c.JSON(http.StatusNotFound, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:        http.StatusNotFound,
			Message:     "endpoint not found",
			Description: fmt.Sprintf("Cannot %s %s", c.Request.Method, c.Request.URL.Path),
		},
		TraceID: requestID,
		Time:    time.Now(),
	})
}

func MethodNotAllowedHandler(c *gin.Context) {
	requestID := ""
	if rid, exists := c.Get("request_id"); exists {
		requestID = rid.(string)
	}

	c.JSON(http.StatusMethodNotAllowed, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:        http.StatusMethodNotAllowed,
			Message:     "method not allowed",
			Description: fmt.Sprintf("Method %s is not allowed for this endpoint", c.Request.Method),
		},
		TraceID: requestID,
		Time:    time.Now(),
	})
}

func handleError(c *gin.Context, status int, message string) {
	requestID := ""
	if rid, exists := c.Get("request_id"); exists {
		requestID = rid.(string)
	}

	c.JSON(status, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:    status,
			Message: message,
		},
		TraceID: requestID,
		Time:    time.Now(),
	})
}

func getTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("request_id"); exists {
		return traceID.(string)
	}
	return ""
}

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

func BadRequestError(message string) *APIError {
	return NewAPIError(http.StatusBadRequest, message)
}

func UnauthorizedError(message string) *APIError {
	return NewAPIError(http.StatusUnauthorized, message)
}

func ForbiddenError(message string) *APIError {
	return NewAPIError(http.StatusForbidden, message)
}

func NotFoundError(message string) *APIError {
	return NewAPIError(http.StatusNotFound, message)
}

func ConflictError(message string) *APIError {
	return NewAPIError(http.StatusConflict, message)
}

func InternalServerError(message string) *APIError {
	return NewAPIError(http.StatusInternalServerError, message)
}

func TooManyRequestsError(message string) *APIError {
	return NewAPIError(http.StatusTooManyRequests, message)
}

func AbortWithError(c *gin.Context, err *APIError) {
	c.AbortWithStatusJSON(err.Code, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:    err.Code,
			Message: err.Message,
		},
		TraceID: getTraceID(c),
		Time:    time.Now(),
	})
}

func HandleServiceError(c *gin.Context, err error) {
	if apiErr, ok := err.(*APIError); ok {
		AbortWithError(c, apiErr)
		return
	}

	response.InternalServerError(c, "service error: "+err.Error())
}
