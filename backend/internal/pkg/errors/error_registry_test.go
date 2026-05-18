package errors

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppErrorWithCategory(t *testing.T) {
	err := New(CodeInvalidParams, "test error")
	err = err.WithCategory(CategoryParameter)

	assert.Equal(t, CategoryParameter, err.Category)
}

func TestAppErrorWithSeverity(t *testing.T) {
	err := New(CodeInternalError, "internal error")
	err = err.WithSeverity(SeverityCritical)

	assert.Equal(t, SeverityCritical, err.Severity)
}

func TestAppErrorIsRetryable(t *testing.T) {
	tests := []struct {
		code     Code
		expected bool
	}{
		{CodeTokenExpired, true},
		{CodeDatabaseError, true},
		{CodeCacheError, true},
		{CodeExternalService, true},
		{CodeOperationFailed, true},
		{CodeOperationTimeout, true},
		{CodeRateLimited, true},
		{CodeTooManyRequest, true},
		{CodeCaptchaFailed, true},
		{CodeResourceLimit, true},
		{CodeInvalidParams, false},
		{CodeMissingParams, false},
		{CodeUnauthorized, false},
		{CodeNotFound, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			err := New(tt.code, "test")
			assert.Equal(t, tt.expected, err.Retryable)
		})
	}
}

func TestAppErrorGetLocalizedMessage(t *testing.T) {
	err := New(CodeInvalidParams, "invalid params")

	assert.Equal(t, "参数无效", err.GetLocalizedMessage("zh-CN"))
	assert.Equal(t, "Invalid parameters", err.GetLocalizedMessage("en-US"))
}

func TestGenerateErrorID(t *testing.T) {
	id1 := GenerateErrorID()
	id2 := GenerateErrorID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "ERR-")
}

func TestAppErrorToResponseWithLocale(t *testing.T) {
	err := New(CodeInvalidParams, "test error")
	err.Category = CategoryParameter
	err.Severity = SeverityWarning

	resp := err.ToResponseWithLocale("en-US")

	assert.Equal(t, CodeInvalidParams, resp["code"])
	assert.Equal(t, "Invalid parameters", resp["message"])
	assert.Equal(t, "parameter", resp["category"])
	assert.Equal(t, "warning", resp["severity"])
}

func TestGetErrorCodeInfo(t *testing.T) {
	info := GetErrorCodeInfo(CodeInvalidParams)

	assert.NotNil(t, info)
	assert.Equal(t, CodeInvalidParams, info.Code)
	assert.Equal(t, CategoryParameter, info.Category)
	assert.Equal(t, SeverityWarning, info.Severity)
}

func TestRegisterCustomErrorCode(t *testing.T) {
	customCode := Code(10001)
	err := RegisterCustomErrorCode(customCode, &ErrorCode{
		Category:   CategoryBusiness,
		Severity:   SeverityError,
		Message:    "自定义错误",
		MessageEn:  "Custom error",
		HTTPStatus: 500,
		Retryable:  true,
	})

	assert.NoError(t, err)

	info := GetErrorCodeInfo(customCode)
	assert.NotNil(t, info)
	assert.Equal(t, "自定义错误", info.Message)
}

func TestListErrorCodes(t *testing.T) {
	codes := ListErrorCodes(CategoryParameter)

	assert.NotEmpty(t, codes)
	for _, code := range codes {
		assert.Equal(t, CategoryParameter, code.Category)
	}
}

func TestExportErrorCodesJSON(t *testing.T) {
	data, err := ExportErrorCodesJSON()

	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var codes []map[string]interface{}
	err = json.Unmarshal(data, &codes)
	assert.NoError(t, err)
	assert.NotEmpty(t, codes)
}

func TestErrorStatisticsCollector(t *testing.T) {
	collector := GetErrorStatisticsCollector()
	collector.Reset()

	collector.RecordError(CodeInvalidParams)
	collector.RecordError(CodeInvalidParams)
	collector.RecordError(CodeNotFound)

	stats := collector.GetStats()
	assert.Equal(t, 2, len(stats))

	paramStats := collector.GetStatsByCategory(CategoryParameter)
	assert.Equal(t, 1, len(paramStats))
}

func TestNewFormattedError(t *testing.T) {
	vars := map[string]string{
		"field":  "email",
		"reason": "invalid format",
	}

	err := NewFormattedError(CodeInvalidParams, vars)

	assert.Contains(t, err.Message, "email")
	assert.Contains(t, err.Message, "invalid format")
}

func TestErrorAggregator(t *testing.T) {
	aggregator := NewErrorAggregator(10, time.Hour)
	aggregator.Reset()

	aggregator.Add(New(CodeInvalidParams, "error 1"))
	aggregator.Add(New(CodeInvalidParams, "error 2"))
	aggregator.Add(New(CodeNotFound, "error 3"))

	assert.Equal(t, 2, aggregator.GetCount(CodeInvalidParams))
	assert.Equal(t, 1, aggregator.GetCount(CodeNotFound))

	all := aggregator.GetAll()
	assert.Equal(t, 2, all[CodeInvalidParams])
	assert.Equal(t, 1, all[CodeNotFound])
}

func TestPanicRecorder(t *testing.T) {
	recorder := NewPanicRecorder(10)
	recorder.Reset()

	recorder.Record("test panic", []byte("stack trace"))

	assert.Equal(t, int64(1), recorder.GetCount())

	recent := recorder.GetRecent()
	assert.Len(t, recent, 1)
	assert.Equal(t, "test panic", recent[0].Message)
}

func TestRecoveryHandler(t *testing.T) {
	handler := NewRecoveryHandler()
	handler.SetMaxRetries(2)

	var panicOccurred bool
	handler.AddRecoveryHandler(func(p interface{}) interface{} {
		panicOccurred = true
		return nil
	})

	panicOccurred = false
	fn := func() error {
		panic("test panic")
	}

	err := handler.ExecuteWithRecovery(fn)
	assert.NoError(t, err)
	assert.True(t, panicOccurred)
}

func TestErrorContext(t *testing.T) {
	ctx := &ErrorContext{
		UserID:    "user123",
		RequestID: "req456",
		Endpoint:  "/api/test",
		Method:    "POST",
		Timestamp: time.Now(),
	}

	appErr := New(CodeInvalidParams, "test error")
	awareErr := WithContext(appErr, ctx)

	resp := awareErr.ToResponse()
	assert.NotNil(t, resp["context"])
}

func TestExceptionHandlerChain(t *testing.T) {
	chain := &ExceptionHandlerChain{
		handlers: make([]ExceptionHandler, 0),
	}

	handler := NewMetricsExceptionHandler()
	chain.AddHandler(handler)

	exception := &Exception{
		Type:      "test",
		Message:   "test error",
		Timestamp: time.Now(),
		ErrorID:  "ERR-123456-ABCDEF",
	}

	chain.Handle(exception)
	assert.GreaterOrEqual(t, handler.GetCount("test"), int64(1))
}

func TestErrorCategoryString(t *testing.T) {
	assert.Equal(t, "parameter", CategoryParameter.String())
	assert.Equal(t, "auth", CategoryAuth.String())
	assert.Equal(t, "resource", CategoryResource.String())
	assert.Equal(t, "system", CategorySystem.String())
	assert.Equal(t, "business", CategoryBusiness.String())
	assert.Equal(t, "rate_limit", CategoryRateLimit.String())
	assert.Equal(t, "security", CategorySecurity.String())
	assert.Equal(t, "unknown", CategoryUnknown.String())
}

func TestErrorSeverityString(t *testing.T) {
	assert.Equal(t, "info", SeverityInfo.String())
	assert.Equal(t, "warning", SeverityWarning.String())
	assert.Equal(t, "error", SeverityError.String())
	assert.Equal(t, "critical", SeverityCritical.String())
}

func TestExceptionCreation(t *testing.T) {
	err := New(CodeInternalError, "test error")
	exception := NewException(err, map[string]interface{}{
		"user_id": "123",
	})

	assert.Equal(t, "AppError", exception.Type)
	assert.Equal(t, "test error", exception.Message)
	assert.NotEmpty(t, exception.StackTrace)
	assert.NotEmpty(t, exception.ErrorID)
	assert.NotNil(t, exception.Context)
	assert.Equal(t, "123", exception.Context["user_id"])
}

func TestRecoveryHandlerWithNonRetryable(t *testing.T) {
	handler := NewRecoveryHandler()
	handler.SetMaxRetries(2)

	err := handler.ExecuteWithRetry(func() error {
		return New(CodeInvalidParams, "non-retryable error")
	})

	assert.Error(t, err)
}

func TestMetricsExceptionHandler(t *testing.T) {
	handler := NewMetricsExceptionHandler()

	handler.HandleException(&Exception{
		Type:      "test_type",
		Message:   "test message",
		Timestamp: time.Now(),
	})

	handler.HandleException(&Exception{
		Type:      "test_type",
		Message:   "another message",
		Timestamp: time.Now(),
	})

	handler.HandleException(&Exception{
		Type:      "other_type",
		Message:   "other message",
		Timestamp: time.Now(),
	})

	assert.Equal(t, int64(2), handler.GetCount("test_type"))
	assert.Equal(t, int64(1), handler.GetCount("other_type"))

	allCounts := handler.GetAllCounts()
	assert.Equal(t, int64(2), allCounts["test_type"])
	assert.Equal(t, int64(1), allCounts["other_type"])
}
