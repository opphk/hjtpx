package errors

import (
	"fmt"
	"net/http"
	"strings"
)

type Code int

const (
	CodeSuccess Code = 0

	CodeUnknown Code = 1000

	CodeInvalidParams Code = 1001
	CodeMissingParams Code = 1002
	CodeInvalidFormat Code = 1003

	CodeUnauthorized     Code = 2001
	CodeTokenExpired     Code = 2002
	CodeTokenInvalid     Code = 2003
	CodePermissionDenied Code = 2004

	CodeNotFound      Code = 3001
	CodeAlreadyExists Code = 3002
	CodeResourceLimit Code = 3003

	CodeInternalError   Code = 4001
	CodeDatabaseError   Code = 4002
	CodeCacheError      Code = 4003
	CodeExternalService Code = 4004

	CodeValidationFailed Code = 5001
	CodeOperationFailed  Code = 5002
	CodeOperationTimeout Code = 5003

	CodeRateLimited    Code = 6001
	CodeTooManyRequest Code = 6002

	CodeSecurityRisk  Code = 7001
	CodeCaptchaFailed Code = 7002
)

type AppError struct {
	Code       Code                   `json:"code"`
	HttpStatus int                    `json:"-"`
	Message    string                 `json:"message"`
	Detail     string                 `json:"detail,omitempty"`
	Err        error                  `json:"-"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithDetail(detail string) *AppError {
	return &AppError{
		Code:       e.Code,
		HttpStatus: e.HttpStatus,
		Message:    e.Message,
		Detail:     detail,
		Err:        e.Err,
		Fields:     e.Fields,
	}
}

func (e *AppError) WithField(key string, value interface{}) *AppError {
	fields := e.Fields
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields[key] = value

	return &AppError{
		Code:       e.Code,
		HttpStatus: e.HttpStatus,
		Message:    e.Message,
		Detail:     e.Detail,
		Err:        e.Err,
		Fields:     fields,
	}
}

func (e *AppError) WithFields(fields map[string]interface{}) *AppError {
	newFields := e.Fields
	if newFields == nil {
		newFields = make(map[string]interface{})
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &AppError{
		Code:       e.Code,
		HttpStatus: e.HttpStatus,
		Message:    e.Message,
		Detail:     e.Detail,
		Err:        e.Err,
		Fields:     newFields,
	}
}

func (e *AppError) Wrap(err error) *AppError {
	return &AppError{
		Code:       e.Code,
		HttpStatus: e.HttpStatus,
		Message:    e.Message,
		Detail:     e.Detail,
		Err:        err,
		Fields:     e.Fields,
	}
}

func (e *AppError) ToResponse() map[string]interface{} {
	resp := map[string]interface{}{
		"code":    e.Code,
		"message": e.Message,
	}
	if e.Detail != "" {
		resp["detail"] = e.Detail
	}
	if len(e.Fields) > 0 {
		resp["fields"] = e.Fields
	}
	return resp
}

func New(code Code, message string) *AppError {
	return &AppError{
		Code:       code,
		HttpStatus: CodeToHTTPStatus(code),
		Message:    message,
	}
}

func NewWithError(code Code, message string, err error) *AppError {
	return &AppError{
		Code:       code,
		HttpStatus: CodeToHTTPStatus(code),
		Message:    message,
		Err:        err,
	}
}

func Wrap(err error, code Code, message string) *AppError {
	if err == nil {
		return New(code, message)
	}

	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:       code,
			HttpStatus: CodeToHTTPStatus(code),
			Message:    message,
			Detail:     appErr.Message,
			Err:        err,
		}
	}

	return &AppError{
		Code:       code,
		HttpStatus: CodeToHTTPStatus(code),
		Message:    message,
		Err:        err,
	}
}

func WrapIfErr(err error, message string) error {
	if err == nil {
		return nil
	}
	return New(CodeInternalError, message).Wrap(err)
}

func CodeToHTTPStatus(code Code) int {
	switch code {
	case CodeSuccess:
		return http.StatusOK

	case CodeInvalidParams, CodeMissingParams, CodeInvalidFormat, CodeValidationFailed:
		return http.StatusBadRequest

	case CodeUnauthorized, CodeTokenExpired, CodeTokenInvalid:
		return http.StatusUnauthorized

	case CodePermissionDenied:
		return http.StatusForbidden

	case CodeNotFound:
		return http.StatusNotFound

	case CodeAlreadyExists:
		return http.StatusConflict

	case CodeRateLimited, CodeTooManyRequest:
		return http.StatusTooManyRequests

	case CodeResourceLimit:
		return http.StatusRequestEntityTooLarge

	case CodeInternalError, CodeDatabaseError, CodeCacheError:
		return http.StatusInternalServerError

	case CodeExternalService:
		return http.StatusBadGateway

	case CodeOperationFailed, CodeOperationTimeout:
		return http.StatusInternalServerError

	case CodeSecurityRisk, CodeCaptchaFailed:
		return http.StatusForbidden

	default:
		return http.StatusInternalServerError
	}
}

func CodeToMessage(code Code) string {
	switch code {
	case CodeSuccess:
		return "操作成功"

	case CodeUnknown:
		return "未知错误"

	case CodeInvalidParams:
		return "参数无效"
	case CodeMissingParams:
		return "缺少参数"
	case CodeInvalidFormat:
		return "格式错误"

	case CodeUnauthorized:
		return "未认证"
	case CodeTokenExpired:
		return "令牌已过期"
	case CodeTokenInvalid:
		return "令牌无效"
	case CodePermissionDenied:
		return "权限不足"

	case CodeNotFound:
		return "资源不存在"
	case CodeAlreadyExists:
		return "资源已存在"
	case CodeResourceLimit:
		return "资源受限"

	case CodeInternalError:
		return "内部错误"
	case CodeDatabaseError:
		return "数据库错误"
	case CodeCacheError:
		return "缓存错误"
	case CodeExternalService:
		return "外部服务错误"

	case CodeValidationFailed:
		return "验证失败"
	case CodeOperationFailed:
		return "操作失败"
	case CodeOperationTimeout:
		return "操作超时"

	case CodeRateLimited:
		return "请求过于频繁"
	case CodeTooManyRequest:
		return "请求过多"

	case CodeSecurityRisk:
		return "安全风险"
	case CodeCaptchaFailed:
		return "验证码失败"

	default:
		return "系统错误"
	}
}

func Is(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	if appErr, ok := err.(*AppError); ok {
		if targetAppErr, ok := target.(*AppError); ok {
			return appErr.Code == targetAppErr.Code
		}
	}

	return strings.Contains(err.Error(), target.Error())
}

func IsCode(err error, code Code) bool {
	if err == nil {
		return false
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}

	return false
}

func GetCode(err error) Code {
	if err == nil {
		return CodeSuccess
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}

	return CodeUnknown
}

func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.HttpStatus
	}

	return http.StatusInternalServerError
}

var (
	ErrInvalidParams = New(CodeInvalidParams, "参数无效")
	ErrMissingParams = New(CodeMissingParams, "缺少参数")
	ErrInvalidFormat = New(CodeInvalidFormat, "格式错误")

	ErrUnauthorized     = New(CodeUnauthorized, "未认证")
	ErrTokenExpired     = New(CodeTokenExpired, "令牌已过期")
	ErrTokenInvalid     = New(CodeTokenInvalid, "令牌无效")
	ErrPermissionDenied = New(CodePermissionDenied, "权限不足")

	ErrNotFound      = New(CodeNotFound, "资源不存在")
	ErrAlreadyExists = New(CodeAlreadyExists, "资源已存在")
	ErrResourceLimit = New(CodeResourceLimit, "资源受限")

	ErrInternal        = New(CodeInternalError, "内部错误")
	ErrDatabase        = New(CodeDatabaseError, "数据库错误")
	ErrCache           = New(CodeCacheError, "缓存错误")
	ErrExternalService = New(CodeExternalService, "外部服务错误")

	ErrValidationFailed = New(CodeValidationFailed, "验证失败")
	ErrOperationFailed  = New(CodeOperationFailed, "操作失败")
	ErrOperationTimeout = New(CodeOperationTimeout, "操作超时")

	ErrRateLimited    = New(CodeRateLimited, "请求过于频繁")
	ErrTooManyRequest = New(CodeTooManyRequest, "请求过多")

	ErrSecurityRisk  = New(CodeSecurityRisk, "安全风险")
	ErrCaptchaFailed = New(CodeCaptchaFailed, "验证码失败")
)
