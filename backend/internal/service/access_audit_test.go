package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewAccessAuditService(t *testing.T) {
	service := NewAccessAuditService()
	if service == nil {
		t.Error("NewAccessAuditService returned nil")
	}
	if service.accessLogs == nil {
		t.Error("accessLogs should be initialized")
	}
	if service.permissionChanges == nil {
		t.Error("permissionChanges should be initialized")
	}
	if service.alertThreshold == nil {
		t.Error("alertThreshold should be initialized")
	}
	if len(service.alertThreshold) == 0 {
		t.Error("alertThreshold should have default values")
	}
}

func TestAccessAuditService_LogAccess(t *testing.T) {
	service := NewAccessAuditService()

	log := service.LogAccess(
		AccessLogin,
		1,
		"testuser",
		"192.168.1.1",
		"Mozilla/5.0",
		"session",
		"session-123",
		"login",
		"success",
		100,
		nil,
	)

	if log == nil {
		t.Error("LogAccess returned nil")
	}
	if log.EventType != AccessLogin {
		t.Errorf("EventType mismatch: expected %s, got %s", AccessLogin, log.EventType)
	}
	if log.UserID != 1 {
		t.Errorf("UserID mismatch: expected 1, got %d", log.UserID)
	}
	if log.Username != "testuser" {
		t.Errorf("Username mismatch: expected testuser, got %s", log.Username)
	}
	if log.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress mismatch: expected 192.168.1.1, got %s", log.IPAddress)
	}
	if log.Status != "success" {
		t.Errorf("Status mismatch: expected success, got %s", log.Status)
	}
	if log.RiskScore <= 0 {
		t.Error("RiskScore should be greater than 0")
	}
	if len(log.Tags) == 0 {
		t.Error("Tags should not be empty")
	}
}

func TestAccessAuditService_LogAccessFromRequest(t *testing.T) {
	service := NewAccessAuditService()

	req := httptest.NewRequest("POST", "/api/login", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("X-Session-ID", "session-456")
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	log := service.LogAccessFromRequest(
		AccessLogin,
		2,
		"testuser2",
		req,
		"user",
		"2",
		"login",
		"success",
		50,
	)

	if log == nil {
		t.Error("LogAccessFromRequest returned nil")
	}
	if log.UserAgent != "TestAgent/1.0" {
		t.Errorf("UserAgent mismatch: expected TestAgent/1.0, got %s", log.UserAgent)
	}
	if log.SessionID != "session-456" {
		t.Errorf("SessionID mismatch: expected session-456, got %s", log.SessionID)
	}
}

func TestAccessAuditService_LogPermissionChange(t *testing.T) {
	service := NewAccessAuditService()

	change := &PermissionChange{
		ChangeType:   "grant",
		UserID:       1,
		TargetUserID: 2,
		Permission:   "admin",
		OldValue:     "",
		NewValue:     "admin",
		ChangedBy:    1,
		Reason:       "Test permission change",
		Status:       "completed",
	}

	err := service.LogPermissionChange(change)
	if err != nil {
		t.Errorf("LogPermissionChange failed: %v", err)
	}

	if len(service.permissionChanges) != 1 {
		t.Errorf("permissionChanges count mismatch: expected 1, got %d", len(service.permissionChanges))
	}
}

func TestAccessAuditService_LogSensitiveOperation(t *testing.T) {
	service := NewAccessAuditService()

	err := service.LogSensitiveOperation(
		"export_data",
		"Exporting user data",
		"user_data",
		1,
		"testuser",
		"192.168.1.1",
		SeverityHigh,
	)

	if err != nil {
		t.Errorf("LogSensitiveOperation failed: %v", err)
	}

	if len(service.sensitiveOperations) != 1 {
		t.Errorf("sensitiveOperations count mismatch: expected 1, got %d", len(service.sensitiveOperations))
	}
}

func TestAccessAuditService_DetectAbnormalAccess(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 60; i++ {
		service.LogAccess(
			AccessLogin,
			1,
			"testuser",
			"192.168.1.100",
			"Mozilla/5.0",
			"session",
			"session-123",
			"login",
			"success",
			100,
			nil,
		)
	}

	patterns, err := service.DetectAbnormalAccess(1, "192.168.1.100")
	if err != nil {
		t.Errorf("DetectAbnormalAccess failed: %v", err)
	}

	if len(patterns) == 0 {
		t.Error("Should detect abnormal IP access pattern")
	}

	foundHighFrequency := false
	for _, p := range patterns {
		if p.PatternType == "high_ip_frequency" {
			foundHighFrequency = true
			break
		}
	}
	if !foundHighFrequency {
		t.Error("Should detect high_ip_frequency pattern")
	}
}

func TestAccessAuditService_GetAccessLogs(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 5; i++ {
		service.LogAccess(
			AccessLogin,
			uint(i+1),
			"testuser",
			"192.168.1.1",
			"Mozilla/5.0",
			"session",
			"session-123",
			"login",
			"success",
			100,
			nil,
		)
	}

	filter := AccessLogFilter{
		UserID: 1,
	}
	logs, total, err := service.GetAccessLogs(filter)
	if err != nil {
		t.Errorf("GetAccessLogs failed: %v", err)
	}

	if total != 1 {
		t.Errorf("Total count mismatch: expected 1, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("Logs count mismatch: expected 1, got %d", len(logs))
	}
	if len(logs) > 0 && logs[0].UserID != 1 {
		t.Error("Filtered log should have UserID 1")
	}
}

func TestAccessAuditService_GetAccessStats(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "user1", "192.168.1.1", "Mozilla/5.0", "session", "1", "login", "success", 100, nil)
	service.LogAccess(AccessLogin, 2, "user2", "192.168.1.2", "Mozilla/5.0", "session", "2", "login", "success", 150, nil)
	service.LogAccess(AccessView, 1, "user1", "192.168.1.1", "Mozilla/5.0", "page", "1", "view", "success", 50, nil)

	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now().Add(24 * time.Hour)

	stats, err := service.GetAccessStats(startDate, endDate)
	if err != nil {
		t.Errorf("GetAccessStats failed: %v", err)
	}

	if stats["total_access"] == nil {
		t.Error("total_access should not be nil")
	}
	if stats["by_event_type"] == nil {
		t.Error("by_event_type should not be nil")
	}
	if stats["unique_users_count"] == nil {
		t.Error("unique_users_count should not be nil")
	}
}

func TestAccessAuditService_ExportAccessLogs(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "testuser", "192.168.1.1", "Mozilla/5.0", "session", "1", "login", "success", 100, nil)

	filter := AccessLogFilter{}

	data, err := service.ExportAccessLogs("json", filter)
	if err != nil {
		t.Errorf("ExportAccessLogs (JSON) failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON export should return data")
	}
	if !strings.Contains(string(data), "login") {
		t.Error("JSON export should contain login event")
	}

	csvData, err := service.ExportAccessLogs("csv", filter)
	if err != nil {
		t.Errorf("ExportAccessLogs (CSV) failed: %v", err)
	}
	if len(csvData) == 0 {
		t.Error("CSV export should return data")
	}
	if !strings.Contains(string(csvData), "login") {
		t.Error("CSV export should contain login event")
	}
}

func TestAccessAuditService_GetPermissionChangeHistory(t *testing.T) {
	service := NewAccessAuditService()

	change := &PermissionChange{
		ChangeType:   "grant",
		UserID:       1,
		TargetUserID: 2,
		Permission:   "admin",
		Status:       "completed",
		ChangedBy:    3,
	}
	service.LogPermissionChange(change)

	history, err := service.GetPermissionChangeHistory(1)
	if err != nil {
		t.Errorf("GetPermissionChangeHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("History count mismatch: expected 1, got %d", len(history))
	}

	history2, _ := service.GetPermissionChangeHistory(3)
	if len(history2) != 0 {
		t.Error("User 3 should not have permission changes")
	}
}

func TestAccessAuditService_GetRecentAbnormalPatterns(t *testing.T) {
	service := NewAccessAuditService()

	patterns := service.GetRecentAbnormalPatterns(10)
	if patterns == nil {
		t.Error("GetRecentAbnormalPatterns should not return nil")
	}

	patterns2 := service.GetRecentAbnormalPatterns(0)
	if patterns2 == nil {
		t.Error("GetRecentAbnormalPatterns with limit 0 should not return nil")
	}
}

func TestAccessAuditService_GetUserAccessSummary(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 5; i++ {
		service.LogAccess(AccessLogin, 1, "testuser", "192.168.1.1", "Mozilla/5.0", "session", "1", "login", "success", 100, nil)
	}

	summary, err := service.GetUserAccessSummary(1, 7)
	if err != nil {
		t.Errorf("GetUserAccessSummary failed: %v", err)
	}

	if summary["user_id"] != 1 {
		t.Error("Summary should contain correct user_id")
	}
	if summary["total_access"] != 5 {
		t.Errorf("Total access mismatch: expected 5, got %v", summary["total_access"])
	}
}

func TestAccessAuditService_AlertThresholds(t *testing.T) {
	service := NewAccessAuditService()

	service.SetAlertThreshold("custom_event", 10)

	thresholds := service.GetAlertThresholds()
	if thresholds["custom_event"] != 10 {
		t.Errorf("Custom threshold not set correctly: expected 10, got %d", thresholds["custom_event"])
	}

	if thresholds["failed_login"] != 5 {
		t.Errorf("Default threshold not present: expected 5, got %d", thresholds["failed_login"])
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		xri      string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For header",
			xff:      "10.0.0.1, 192.168.1.1",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Real-IP header",
			xri:      "10.0.0.2",
			expected: "10.0.0.2",
		},
		{
			name:     "RemoteAddr only",
			remote:   "192.168.1.100:12345",
			expected: "192.168.1.100:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			if tt.remote != "" {
				req.RemoteAddr = tt.remote
			}

			ip := getClientIP(req)
			if !strings.HasPrefix(ip, tt.expected) {
				t.Errorf("getClientIP() = %v, want %v", ip, tt.expected)
			}
		})
	}
}

func TestAccessAuditService_DetermineSeverity(t *testing.T) {
	service := NewAccessAuditService()

	tests := []struct {
		eventType AccessEventType
		status    string
		expected  AccessSeverity
	}{
		{AccessLogin, "success", SeverityLow},
		{AccessAdminAction, "success", SeverityHigh},
		{AccessSensitiveData, "success", SeverityHigh},
		{AccessDelete, "success", SeverityMedium},
		{AccessLogin, "failed", SeverityMedium},
		{AccessView, "success", SeverityInfo},
	}

	for _, tt := range tests {
		severity := service.determineSeverity(tt.eventType, tt.status)
		if severity != tt.expected {
			t.Errorf("determineSeverity(%s, %s) = %s, want %s",
				tt.eventType, tt.status, severity, tt.expected)
		}
	}
}

func TestAccessAuditService_CalculateRiskScore(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 35; i++ {
		service.LogAccess(AccessLogin, 1, "user1", "192.168.1.50", "Mozilla/5.0", "session", "1", "login", "success", 100, nil)
	}

	score := service.calculateRiskScore(AccessSensitiveData, "192.168.1.50", 1, "user_data")
	if score <= 0 {
		t.Error("Risk score should be greater than 0 for sensitive data access")
	}
}

func TestAccessAuditService_IsSensitiveResourceType(t *testing.T) {
	service := NewAccessAuditService()

	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"user", true},
		{"password", true},
		{"email_data", true},
		{"payment_info", true},
		{"financial_records", true},
		{"public_page", false},
		{"article", false},
	}

	for _, tt := range tests {
		result := service.isSensitiveResourceType(tt.resourceType)
		if result != tt.expected {
			t.Errorf("isSensitiveResourceType(%s) = %v, want %v",
				tt.resourceType, result, tt.expected)
		}
	}
}

func TestAccessAuditService_GenerateTags(t *testing.T) {
	service := NewAccessAuditService()

	tags := service.generateTags(AccessLogin, "session", "success")
	if len(tags) == 0 {
		t.Error("Tags should not be empty for login event")
	}

	tags = service.generateTags(AccessSensitiveData, "user_password", "success")
	foundSensitive := false
	for _, tag := range tags {
		if tag == "sensitive" || tag == "gdpr" || tag == "pii" {
			foundSensitive = true
			break
		}
	}
	if !foundSensitive {
		t.Error("Sensitive operation should have sensitive-related tags")
	}

	tags = service.generateTags(AccessAdminAction, "admin_panel", "failed")
	foundAdmin := false
	for _, tag := range tags {
		if tag == "admin" || tag == "failed" {
			foundAdmin = true
			break
		}
	}
	if !foundAdmin {
		t.Error("Admin action should have admin-related tags")
	}
}

func TestAccessAuditService_GetGeoLocation(t *testing.T) {
	service := NewAccessAuditService()

	tests := []struct {
		ip       string
		expected string
	}{
		{"10.0.0.1", "Private Network"},
		{"192.168.1.1", "Private Network"},
		{"172.16.0.1", "Private Network"},
		{"8.8.8.8", "Unknown"},
	}

	for _, tt := range tests {
		geo := service.getGeoLocation(tt.ip)
		if geo != tt.expected {
			t.Errorf("getGeoLocation(%s) = %s, want %s", tt.ip, geo, tt.expected)
		}
	}
}
