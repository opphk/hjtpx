package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIncrementRequestCount(t *testing.T) {
	ResetMetrics()
	initial := GetRequestCount()
	IncrementRequestCount()
	current := GetRequestCount()
	assert.Equal(t, initial+1, current)
}

func TestIncrementSuccessCount(t *testing.T) {
	ResetMetrics()
	initial := GetSuccessCount()
	IncrementSuccessCount()
	current := GetSuccessCount()
	assert.Equal(t, initial+1, current)
}

func TestIncrementFailureCount(t *testing.T) {
	ResetMetrics()
	initial := GetFailureCount()
	IncrementFailureCount()
	current := GetFailureCount()
	assert.Equal(t, initial+1, current)
}

func TestGetSuccessRate(t *testing.T) {
	ResetMetrics()

	assert.Equal(t, 100.0, GetSuccessRate())

	IncrementRequestCount()
	IncrementSuccessCount()
	IncrementRequestCount()
	IncrementSuccessCount()
	assert.Equal(t, 100.0, GetSuccessRate())

	ResetMetrics()
	IncrementRequestCount()
	IncrementSuccessCount()
	IncrementRequestCount()
	IncrementFailureCount()
	assert.Equal(t, 50.0, GetSuccessRate())
}

func TestGetUptime(t *testing.T) {
	uptime := GetUptime()
	assert.True(t, uptime >= 0)

	time.Sleep(1 * time.Millisecond)
	newUptime := GetUptime()
	assert.True(t, newUptime > uptime)
}

func TestResetMetrics(t *testing.T) {
	ResetMetrics()
	IncrementRequestCount()
	IncrementSuccessCount()
	IncrementFailureCount()

	assert.Greater(t, GetRequestCount(), uint64(0))
	assert.Greater(t, GetSuccessCount(), uint64(0))
	assert.Greater(t, GetFailureCount(), uint64(0))

	ResetMetrics()

	assert.Equal(t, uint64(0), GetRequestCount())
	assert.Equal(t, uint64(0), GetSuccessCount())
	assert.Equal(t, uint64(0), GetFailureCount())
}

func TestRecordCacheHit(t *testing.T) {
	RecordCacheHit()
}

func TestRecordCacheMiss(t *testing.T) {
	RecordCacheMiss()
}

func TestRecordCacheSize(t *testing.T) {
	RecordCacheSize(100)
}

func TestRecordCacheOperation(t *testing.T) {
	RecordCacheOperation("set")
	RecordCacheOperation("delete")
}

func TestRecordDBQueryDuration(t *testing.T) {
	RecordDBQueryDuration("select", 10*time.Millisecond)
	RecordDBQueryDuration("insert", 5*time.Millisecond)
}

func TestRecordDBError(t *testing.T) {
	RecordDBError()
}

func TestRecordBusinessError(t *testing.T) {
	RecordBusinessError("validation", "/api/user")
	RecordBusinessError("authorization", "/api/admin")
}

func TestRecordAuthSuccess(t *testing.T) {
	RecordAuthSuccess()
}

func TestRecordAuthFailure(t *testing.T) {
	RecordAuthFailure("invalid_token")
	RecordAuthFailure("wrong_password")
}

func TestRecordCaptchaSuccess(t *testing.T) {
	RecordCaptchaSuccess()
}

func TestRecordCaptchaFailure(t *testing.T) {
	RecordCaptchaFailure()
}

func TestRecordCaptchaBlocked(t *testing.T) {
	RecordCaptchaBlocked()
}

func TestRecordRateLimitAccepted(t *testing.T) {
	RecordRateLimitAccepted()
}

func TestRecordRateLimitRejected(t *testing.T) {
	RecordRateLimitRejected()
}

func TestRecordHealthCheck(t *testing.T) {
	RecordHealthCheck("success")
	RecordHealthCheck("failure")
}

func TestSetAvailability(t *testing.T) {
	SetAvailability(99.9)
	SetAvailability(100.0)
}

func TestGetMetricsSummary(t *testing.T) {
	summary := GetMetricsSummary()
	assert.Contains(t, summary, "http_requests_total")
	assert.Contains(t, summary, "http_success_rate")
	assert.Contains(t, summary, "uptime_seconds")
}