package response

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PageInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginatedResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *PageMeta   `json:"meta,omitempty"`
}

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Errors  []FieldError `json:"errors,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Errors  []FieldError `json:"errors,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

func SuccessWithMeta(c *gin.Context, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
		Meta:    meta,
	})
}

func SuccessPaginated(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	meta := &PageMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
		Meta:    meta,
	})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func ErrorWithFields(c *gin.Context, httpStatus int, code int, message string, errors []FieldError) {
	c.JSON(httpStatus, ErrorResponse{
		Code:    code,
		Message: message,
		Errors:  errors,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, CodeInvalidParams, message)
}

func BadRequestWithFields(c *gin.Context, message string, errors []FieldError) {
	ErrorWithFields(c, http.StatusBadRequest, CodeInvalidParams, message, errors)
}

func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
}

func UnauthorizedWithMessage(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

func Forbidden(c *gin.Context) {
	Error(c, http.StatusForbidden, CodeForbidden, "forbidden")
}

func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	Error(c, http.StatusNotFound, CodeNotFound, message)
}

func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	Error(c, http.StatusInternalServerError, CodeServerError, message)
}

func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "too many requests"
	}
	Error(c, http.StatusTooManyRequests, CodeTooManyRequests, message)
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Code:    CodeSuccess,
		Message: "created",
		Data:    data,
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func getTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("trace_id"); exists {
		return traceID.(string)
	}
	return ""
}

func setCacheControl(c *gin.Context, maxAge time.Duration) {
	if maxAge > 0 {
		c.Header("Cache-Control", "public, max-age="+strconv.FormatInt(int64(maxAge.Seconds()), 10))
	} else {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	}
}

func SetCacheHeaders(c *gin.Context, maxAge time.Duration, isPublic bool) {
	if maxAge > 0 {
		cacheType := "private"
		if isPublic {
			cacheType = "public"
		}
		c.Header("Cache-Control", cacheType+", max-age="+strconv.FormatInt(int64(maxAge.Seconds()), 10))
		c.Header("Expires", time.Now().Add(maxAge).Format(http.TimeFormat))
	} else {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	}
}

func SetCORSHeaders(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Correlation-ID")
	c.Header("Access-Control-Max-Age", "86400")
}

func ValidateRequiredFields(data map[string]interface{}, required []string) []FieldError {
	var errors []FieldError

	for _, field := range required {
		if val, exists := data[field]; !exists || val == nil || val == "" {
			errors = append(errors, FieldError{
				Field:   field,
				Message: field + " is required",
			})
		}
	}

	return errors
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}

func HealthCheck(c *gin.Context, services map[string]string) {
	status := "healthy"
	httpStatus := http.StatusOK

	for _, v := range services {
		if v != "healthy" {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			break
		}
	}

	c.JSON(httpStatus, gin.H{
		"status":    status,
		"timestamp": time.Now(),
		"services":  services,
	})
}

const (
	CodeSuccess         = 0
	CodeInvalidParams   = 400
	CodeUnauthorized    = 401
	CodeForbidden       = 403
	CodeNotFound        = 404
	CodeServerError     = 500
	CodeTooManyRequests = 429
	CodeConflict        = 409
	CodeGone            = 410
	CodeUnprocessable   = 422
)

func Conflict(c *gin.Context, message string) {
	if message == "" {
		message = "resource conflict"
	}
	Error(c, http.StatusConflict, CodeConflict, message)
}

func Gone(c *gin.Context, message string) {
	if message == "" {
		message = "resource no longer available"
	}
	Error(c, http.StatusGone, CodeGone, message)
}

func UnprocessableEntity(c *gin.Context, message string) {
	if message == "" {
		message = "unprocessable entity"
	}
	Error(c, http.StatusUnprocessableEntity, CodeUnprocessable, message)
}
