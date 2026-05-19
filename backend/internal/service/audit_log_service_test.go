package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAuditLogService(t *testing.T) {
	service := NewAuditLogService()
	assert.NotNil(t, service)
}

func TestAuditLogEntry_Validation(t *testing.T) {
	entry := &AuditLogEntry{
		ID:           1,
		Timestamp:    time.Now(),
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
		Endpoint:     "/api/test",
		Method:       "GET",
		StatusCode:   200,
		ResponseTime: 100,
		Action:       "read",
		ResourceType: "test",
		ResourceID:   "1",
	}

	assert.Equal(t, uint(1), entry.ID)
	assert.Equal(t, "testuser", entry.Username)
	assert.Equal(t, "GET", entry.Method)
	assert.Equal(t, 200, entry.StatusCode)
}

func TestAuditLogService_LogRequest(t *testing.T) {
	service := NewAuditLogService()

	entry := &AuditLogEntry{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
		Endpoint:     "/api/test",
		Method:       "GET",
		StatusCode:   200,
		ResponseTime: 100,
		Action:       "read",
		ResourceType: "test",
		ResourceID:   "1",
	}

	err := service.LogRequest(entry)
	assert.NoError(t, err)
}

func TestAuditLogService_LogSecurityEvent(t *testing.T) {
	service := NewAuditLogService()

	err := service.LogSecurityEvent("test_event", "event description", 1, "127.0.0.1", map[string]interface{}{
		"key": "value",
	})
	assert.NoError(t, err)
}

func TestAuditLogService_LogAuthentication(t *testing.T) {
	service := NewAuditLogService()

	err := service.LogAuthentication("testuser", true, "127.0.0.1", "test-agent")
	assert.NoError(t, err)

	err = service.LogAuthentication("testuser", false, "127.0.0.1", "test-agent")
	assert.NoError(t, err)
}

func TestAuditLogService_LogAdminAction(t *testing.T) {
	service := NewAuditLogService()

	err := service.LogAdminAction(1, "admin", "create", "user", "1", `{"name":"test"}`)
	assert.NoError(t, err)
}

func TestAuditLogEntry_ErrorHandling(t *testing.T) {
	entry := &AuditLogEntry{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "127.0.0.1",
		Endpoint:     "/api/test",
		Method:       "GET",
		StatusCode:   500,
		ResponseTime: 500,
		Action:       "read",
		ResourceType: "test",
		ResourceID:   "1",
		Error:        "internal server error",
	}

	assert.Equal(t, 500, entry.StatusCode)
	assert.Equal(t, "internal server error", entry.Error)
}
