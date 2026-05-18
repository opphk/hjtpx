package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ErrorCategory int

const (
	CategoryUnknown ErrorCategory = iota
	CategoryParameter
	CategoryAuth
	CategoryResource
	CategorySystem
	CategoryBusiness
	CategoryRateLimit
	CategorySecurity
)

func (c ErrorCategory) String() string {
	switch c {
	case CategoryParameter:
		return "parameter"
	case CategoryAuth:
		return "auth"
	case CategoryResource:
		return "resource"
	case CategorySystem:
		return "system"
	case CategoryBusiness:
		return "business"
	case CategoryRateLimit:
		return "rate_limit"
	case CategorySecurity:
		return "security"
	default:
		return "unknown"
	}
}

type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

type ErrorCode struct {
	Code      Code
	Category  ErrorCategory
	Severity  ErrorSeverity
	Message   string
	MessageEn string
	HTTPStatus int
	Retryable bool
	Recoverable bool
}

var (
	errorCodeRegistry     map[Code]*ErrorCode
	errorCodeRegistryMu  sync.RWMutex
	errorCodeMapping      map[string]Code
)

func init() {
	errorCodeRegistry = make(map[Code]*ErrorCode)
	errorCodeMapping = make(map[string]Code)

	registerErrorCode(CodeSuccess, &ErrorCode{
		Code:         CodeSuccess,
		Category:     CategoryUnknown,
		Severity:     SeverityInfo,
		Message:      "操作成功",
		MessageEn:    "Success",
		HTTPStatus:   http.StatusOK,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeUnknown, &ErrorCode{
		Code:         CodeUnknown,
		Category:     CategoryUnknown,
		Severity:     SeverityError,
		Message:      "未知错误",
		MessageEn:    "Unknown error",
		HTTPStatus:   http.StatusInternalServerError,
		Retryable:    true,
		Recoverable:  false,
	})

	registerErrorCode(CodeInvalidParams, &ErrorCode{
		Code:         CodeInvalidParams,
		Category:     CategoryParameter,
		Severity:     SeverityWarning,
		Message:      "参数无效",
		MessageEn:    "Invalid parameters",
		HTTPStatus:   http.StatusBadRequest,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeMissingParams, &ErrorCode{
		Code:         CodeMissingParams,
		Category:     CategoryParameter,
		Severity:     SeverityWarning,
		Message:      "缺少参数",
		MessageEn:    "Missing required parameters",
		HTTPStatus:   http.StatusBadRequest,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeInvalidFormat, &ErrorCode{
		Code:         CodeInvalidFormat,
		Category:     CategoryParameter,
		Severity:     SeverityWarning,
		Message:      "格式错误",
		MessageEn:    "Invalid format",
		HTTPStatus:   http.StatusBadRequest,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeUnauthorized, &ErrorCode{
		Code:         CodeUnauthorized,
		Category:     CategoryAuth,
		Severity:     SeverityWarning,
		Message:      "未认证",
		MessageEn:    "Unauthorized",
		HTTPStatus:   http.StatusUnauthorized,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeTokenExpired, &ErrorCode{
		Code:         CodeTokenExpired,
		Category:     CategoryAuth,
		Severity:     SeverityWarning,
		Message:      "令牌已过期",
		MessageEn:    "Token expired",
		HTTPStatus:   http.StatusUnauthorized,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeTokenInvalid, &ErrorCode{
		Code:         CodeTokenInvalid,
		Category:     CategoryAuth,
		Severity:     SeverityWarning,
		Message:      "令牌无效",
		MessageEn:    "Invalid token",
		HTTPStatus:   http.StatusUnauthorized,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodePermissionDenied, &ErrorCode{
		Code:         CodePermissionDenied,
		Category:     CategoryAuth,
		Severity:     SeverityWarning,
		Message:      "权限不足",
		MessageEn:    "Permission denied",
		HTTPStatus:   http.StatusForbidden,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeNotFound, &ErrorCode{
		Code:         CodeNotFound,
		Category:     CategoryResource,
		Severity:     SeverityInfo,
		Message:      "资源不存在",
		MessageEn:    "Resource not found",
		HTTPStatus:   http.StatusNotFound,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeAlreadyExists, &ErrorCode{
		Code:         CodeAlreadyExists,
		Category:     CategoryResource,
		Severity:     SeverityWarning,
		Message:      "资源已存在",
		MessageEn:    "Resource already exists",
		HTTPStatus:   http.StatusConflict,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeResourceLimit, &ErrorCode{
		Code:         CodeResourceLimit,
		Category:     CategoryResource,
		Severity:     SeverityWarning,
		Message:      "资源受限",
		MessageEn:    "Resource limit exceeded",
		HTTPStatus:   http.StatusRequestEntityTooLarge,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeInternalError, &ErrorCode{
		Code:         CodeInternalError,
		Category:     CategorySystem,
		Severity:     SeverityError,
		Message:      "内部错误",
		MessageEn:    "Internal server error",
		HTTPStatus:   http.StatusInternalServerError,
		Retryable:    true,
		Recoverable:  false,
	})

	registerErrorCode(CodeDatabaseError, &ErrorCode{
		Code:         CodeDatabaseError,
		Category:     CategorySystem,
		Severity:     SeverityError,
		Message:      "数据库错误",
		MessageEn:    "Database error",
		HTTPStatus:   http.StatusInternalServerError,
		Retryable:    true,
		Recoverable:  false,
	})

	registerErrorCode(CodeCacheError, &ErrorCode{
		Code:         CodeCacheError,
		Category:     CategorySystem,
		Severity:     SeverityError,
		Message:      "缓存错误",
		MessageEn:    "Cache error",
		HTTPStatus:   http.StatusInternalServerError,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeExternalService, &ErrorCode{
		Code:         CodeExternalService,
		Category:     CategorySystem,
		Severity:     SeverityError,
		Message:      "外部服务错误",
		MessageEn:    "External service error",
		HTTPStatus:   http.StatusBadGateway,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeValidationFailed, &ErrorCode{
		Code:         CodeValidationFailed,
		Category:     CategoryBusiness,
		Severity:     SeverityWarning,
		Message:      "验证失败",
		MessageEn:    "Validation failed",
		HTTPStatus:   http.StatusBadRequest,
		Retryable:    false,
		Recoverable:  true,
	})

	registerErrorCode(CodeOperationFailed, &ErrorCode{
		Code:         CodeOperationFailed,
		Category:     CategoryBusiness,
		Severity:     SeverityError,
		Message:      "操作失败",
		MessageEn:    "Operation failed",
		HTTPStatus:   http.StatusInternalServerError,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeOperationTimeout, &ErrorCode{
		Code:         CodeOperationTimeout,
		Category:     CategoryBusiness,
		Severity:     SeverityError,
		Message:      "操作超时",
		MessageEn:    "Operation timeout",
		HTTPStatus:   http.StatusInternalServerError,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeRateLimited, &ErrorCode{
		Code:         CodeRateLimited,
		Category:     CategoryRateLimit,
		Severity:     SeverityWarning,
		Message:      "请求过于频繁",
		MessageEn:    "Rate limit exceeded",
		HTTPStatus:   http.StatusTooManyRequests,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeTooManyRequest, &ErrorCode{
		Code:         CodeTooManyRequest,
		Category:     CategoryRateLimit,
		Severity:     SeverityWarning,
		Message:      "请求过多",
		MessageEn:    "Too many requests",
		HTTPStatus:   http.StatusTooManyRequests,
		Retryable:    true,
		Recoverable:  true,
	})

	registerErrorCode(CodeSecurityRisk, &ErrorCode{
		Code:         CodeSecurityRisk,
		Category:     CategorySecurity,
		Severity:     SeverityCritical,
		Message:      "安全风险",
		MessageEn:    "Security risk detected",
		HTTPStatus:   http.StatusForbidden,
		Retryable:    false,
		Recoverable:  false,
	})

	registerErrorCode(CodeCaptchaFailed, &ErrorCode{
		Code:         CodeCaptchaFailed,
		Category:     CategorySecurity,
		Severity:     SeverityWarning,
		Message:      "验证码失败",
		MessageEn:    "Captcha verification failed",
		HTTPStatus:   http.StatusForbidden,
		Retryable:    true,
		Recoverable:  true,
	})
}

func registerErrorCode(code Code, info *ErrorCode) {
	errorCodeRegistryMu.Lock()
	defer errorCodeRegistryMu.Unlock()

	errorCodeRegistry[code] = info
	errorCodeMapping[info.Message] = code
	errorCodeMapping[info.MessageEn] = code
}

func GetErrorCodeInfo(code Code) *ErrorCode {
	errorCodeRegistryMu.RLock()
	defer errorCodeRegistryMu.RUnlock()

	if info, exists := errorCodeRegistry[code]; exists {
		return info
	}
	return nil
}

func RegisterCustomErrorCode(code Code, info *ErrorCode) error {
	if code < 10000 {
		return fmt.Errorf("custom error code must be >= 10000, got %d", code)
	}

	errorCodeRegistryMu.Lock()
	defer errorCodeRegistryMu.Unlock()

	if _, exists := errorCodeRegistry[code]; exists {
		return fmt.Errorf("error code %d already registered", code)
	}

	info.Code = code
	errorCodeRegistry[code] = info
	errorCodeMapping[info.Message] = code
	errorCodeMapping[info.MessageEn] = code

	return nil
}

func ListErrorCodes(category ErrorCategory) []*ErrorCode {
	errorCodeRegistryMu.RLock()
	defer errorCodeRegistryMu.RUnlock()

	var results []*ErrorCode
	for _, info := range errorCodeRegistry {
		if category == CategoryUnknown || info.Category == category {
			results = append(results, info)
		}
	}
	return results
}

func ExportErrorCodesJSON() ([]byte, error) {
	errorCodeRegistryMu.RLock()
	defer errorCodeRegistryMu.RUnlock()

	type ErrorCodeExport struct {
		Code        int    `json:"code"`
		Category    string `json:"category"`
		Severity    string `json:"severity"`
		Message     string `json:"message"`
		MessageEn   string `json:"message_en"`
		HTTPStatus  int    `json:"http_status"`
		Retryable   bool   `json:"retryable"`
		Recoverable bool   `json:"recoverable"`
	}

	var exports []ErrorCodeExport
	for code, info := range errorCodeRegistry {
		exports = append(exports, ErrorCodeExport{
			Code:        int(code),
			Category:    info.Category.String(),
			Severity:    info.Severity.String(),
			Message:     info.Message,
			MessageEn:   info.MessageEn,
			HTTPStatus:  info.HTTPStatus,
			Retryable:   info.Retryable,
			Recoverable: info.Recoverable,
		})
	}

	return json.MarshalIndent(exports, "", "  ")
}

func (e *AppError) WithCategory(category ErrorCategory) *AppError {
	return &AppError{
		Code:       e.Code,
		HttpStatus: e.HttpStatus,
		Message:    e.Message,
		Detail:     e.Detail,
		Err:        e.Err,
		Fields:     e.Fields,
		Category:   category,
	}
}

func (e *AppError) WithSeverity(severity ErrorSeverity) *AppError {
	return &AppError{
		Code:       e.Code,
		HttpStatus: e.HttpStatus,
		Message:    e.Message,
		Detail:     e.Detail,
		Err:        e.Err,
		Fields:     e.Fields,
		Severity:   severity,
	}
}

func (e *AppError) IsRetryable() bool {
	if info := GetErrorCodeInfo(e.Code); info != nil {
		return info.Retryable
	}
	return false
}

func (e *AppError) IsRecoverable() bool {
	if info := GetErrorCodeInfo(e.Code); info != nil {
		return info.Recoverable
	}
	return false
}

func (e *AppError) GetCategory() ErrorCategory {
	if info := GetErrorCodeInfo(e.Code); info != nil {
		return info.Category
	}
	return CategoryUnknown
}

func (e *AppError) GetSeverity() ErrorSeverity {
	if info := GetErrorCodeInfo(e.Code); info != nil {
		return info.Severity
	}
	return SeverityError
}

func (e *AppError) ToResponseWithLocale(locale string) map[string]interface{} {
	resp := map[string]interface{}{
		"code":    e.Code,
		"message": e.GetLocalizedMessage(locale),
	}

	if e.Detail != "" {
		resp["detail"] = e.Detail
	}
	if len(e.Fields) > 0 {
		resp["fields"] = e.Fields
	}
	if e.Category != CategoryUnknown {
		resp["category"] = e.Category.String()
	}
	if e.Severity != SeverityError {
		resp["severity"] = e.Severity.String()
	}

	return resp
}

func (e *AppError) GetLocalizedMessage(locale string) string {
	if info := GetErrorCodeInfo(e.Code); info != nil {
		if strings.HasPrefix(locale, "en") {
			return info.MessageEn
		}
		return info.Message
	}
	return e.Message
}

type ErrorTemplate struct {
	Code       Code
	Message    string
	Template   string
	Variables  []string
}

var errorTemplates = map[Code]*ErrorTemplate{
	CodeInvalidParams: {
		Code:       CodeInvalidParams,
		Message:    "参数无效",
		Template:   "参数 {{field}} 无效: {{reason}}",
		Variables:  []string{"field", "reason"},
	},
	CodeMissingParams: {
		Code:       CodeMissingParams,
		Message:    "缺少参数",
		Template:   "缺少必需参数: {{fields}}",
		Variables:  []string{"fields"},
	},
	CodeValidationFailed: {
		Code:       CodeValidationFailed,
		Message:    "验证失败",
		Template:   "验证失败: {{reason}}",
		Variables:  []string{"reason"},
	},
}

func FormatErrorMessage(template *ErrorTemplate, vars map[string]string) string {
	message := template.Template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		message = strings.ReplaceAll(message, placeholder, value)
	}
	return message
}

func NewFormattedError(code Code, vars map[string]string) *AppError {
	if template, exists := errorTemplates[code]; exists {
		return New(code, FormatErrorMessage(template, vars))
	}
	return New(code, CodeToMessage(code))
}

var errorIDPattern = regexp.MustCompile(`^ERR-\d{10}-[A-Z0-9]{6}$`)

func GenerateErrorID() string {
	timestamp := time.Now().Unix()
	randomPart := strings.ToUpper(randomString(6))
	return fmt.Sprintf("ERR-%d-%s", timestamp, randomPart)
}

func randomString(n int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func IsValidErrorID(id string) bool {
	return errorIDPattern.MatchString(id)
}

type ErrorStatistics struct {
	Code          Code           `json:"code"`
	Count         int64          `json:"count"`
	FirstOccurred time.Time      `json:"first_occurred"`
	LastOccurred  time.Time      `json:"last_occurred"`
	AverageRate   float64        `json:"average_rate"`
}

type ErrorStatisticsCollector struct {
	mu      sync.RWMutex
	stats   map[Code]*ErrorStatistics
}

var errorStatsCollector *ErrorStatisticsCollector

func GetErrorStatisticsCollector() *ErrorStatisticsCollector {
	if errorStatsCollector == nil {
		errorStatsCollector = &ErrorStatisticsCollector{
			stats: make(map[Code]*ErrorStatistics),
		}
	}
	return errorStatsCollector
}

func (c *ErrorStatisticsCollector) RecordError(code Code) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if stat, exists := c.stats[code]; exists {
		stat.Count++
		stat.LastOccurred = now

		duration := stat.LastOccurred.Sub(stat.FirstOccurred)
		if duration > 0 {
			stat.AverageRate = float64(stat.Count) / duration.Seconds()
		}
	} else {
		c.stats[code] = &ErrorStatistics{
			Code:          code,
			Count:         1,
			FirstOccurred: now,
			LastOccurred:  now,
			AverageRate:   0,
		}
	}
}

func (c *ErrorStatisticsCollector) GetStats() []*ErrorStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []*ErrorStatistics
	for _, stat := range c.stats {
		results = append(results, stat)
	}
	return results
}

func (c *ErrorStatisticsCollector) GetStatsByCategory(category ErrorCategory) []*ErrorStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []*ErrorStatistics
	for _, stat := range c.stats {
		if info := GetErrorCodeInfo(stat.Code); info != nil && info.Category == category {
			results = append(results, stat)
		}
	}
	return results
}

func (c *ErrorStatisticsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats = make(map[Code]*ErrorStatistics)
}
