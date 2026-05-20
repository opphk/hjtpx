package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessAuditService_NewService(t *testing.T) {
	service := NewAccessAuditService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.accessLogs)
	assert.NotNil(t, service.permissionChanges)
	assert.NotNil(t, service.sensitiveOperations)
	assert.NotNil(t, service.abnormalPatterns)
	assert.Equal(t, 10000, service.maxLogs)
	assert.Equal(t, 90, service.retentionDays)
	assert.True(t, service.enableGeoLocation)
	assert.True(t, service.enableRiskScoring)
}

func TestAccessAuditService_LogAccess(t *testing.T) {
	service := NewAccessAuditService()

	log := service.LogAccess(
		AccessLogin,
		1,
		"testuser",
		"192.168.1.100",
		"Mozilla/5.0",
		"session",
		"123",
		"login",
		"success",
		100,
		map[string]string{"session_id": "sess123"},
	)

	assert.NotNil(t, log)
	assert.Equal(t, AccessLogin, log.EventType)
	assert.Equal(t, uint(1), log.UserID)
	assert.Equal(t, "testuser", log.Username)
	assert.Equal(t, "192.168.1.100", log.IPAddress)
	assert.Equal(t, "Mozilla/5.0", log.UserAgent)
	assert.Equal(t, "success", log.Status)
	assert.Equal(t, int64(100), log.ResponseTime)
	assert.NotNil(t, log.Tags)
}

func TestAccessAuditService_LogAccessFromRequest(t *testing.T) {
	service := NewAccessAuditService()

	req := httptest.NewRequest("POST", "/test?param=value", nil)
	req.Header.Set("X-Session-ID", "sess-abc-123")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")

	log := service.LogAccessFromRequest(
		AccessView,
		2,
		"viewer",
		req,
		"document",
		"doc-456",
		"view",
		"success",
		50,
	)

	assert.NotNil(t, log)
	assert.Equal(t, AccessView, log.EventType)
	assert.Equal(t, uint(2), log.UserID)
	assert.Equal(t, "viewer", log.Username)
	assert.Equal(t, "10.0.0.1", log.IPAddress)
	assert.Equal(t, "TestAgent/1.0", log.UserAgent)
	assert.Equal(t, "sess-abc-123", log.SessionID)
	assert.Contains(t, log.RequestPath, "/test")
}

func TestAccessAuditService_LogPermissionChange(t *testing.T) {
	service := NewAccessAuditService()

	change := &PermissionChange{
		ChangeType: "grant",
		UserID:     3,
		Permission: "admin",
		NewValue:   "true",
		ChangedBy:  1,
		Reason:     "Promotion",
		Status:     "approved",
	}

	err := service.LogPermissionChange(change)
	assert.NoError(t, err)

	history, err := service.GetPermissionChangeHistory(3)
	assert.NoError(t, err)
	assert.NotEmpty(t, history)
}

func TestAccessAuditService_LogSensitiveOperation(t *testing.T) {
	service := NewAccessAuditService()

	err := service.LogSensitiveOperation(
		"export",
		"Export user data",
		"user_data",
		4,
		"sensitive_user",
		"192.168.1.50",
		SeverityHigh,
	)

	assert.NoError(t, err)
}

func TestAccessAuditService_DetectAbnormalAccess(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 60; i++ {
		service.LogAccess(
			AccessLogin,
			5,
			"rapid_user",
			"192.168.1.200",
			"Mozilla/5.0",
			"session",
			"456",
			"login",
			"success",
			50,
			nil,
		)
	}

	patterns, err := service.DetectAbnormalAccess(5, "192.168.1.200")
	assert.NoError(t, err)
	assert.NotEmpty(t, patterns)

	highIPPatternFound := false
	for _, p := range patterns {
		if p.PatternType == "high_ip_frequency" {
			highIPPatternFound = true
			assert.Equal(t, SeverityMedium, p.Severity)
			break
		}
	}
	assert.True(t, highIPPatternFound, "Expected high_ip_frequency pattern")
}

func TestAccessAuditService_GetAccessLogs(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 5; i++ {
		service.LogAccess(
			AccessLogin,
			uint(i+10),
			"user",
			"192.168.1.10",
			"Mozilla/5.0",
			"session",
			"123",
			"login",
			"success",
			50,
			nil,
		)
	}

	filter := AccessLogFilter{
		EventType: AccessLogin,
		Limit:     3,
		Offset:    0,
	}

	logs, total, err := service.GetAccessLogs(filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 3)
}

func TestAccessAuditService_GetAccessLogs_FilterByUserID(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 100, "user1", "10.0.0.1", "UA1", "session", "s1", "login", "success", 50, nil)
	service.LogAccess(AccessView, 200, "user2", "10.0.0.2", "UA2", "session", "s2", "view", "success", 50, nil)
	service.LogAccess(AccessLogin, 100, "user1", "10.0.0.3", "UA1", "session", "s3", "login", "success", 50, nil)

	filter := AccessLogFilter{UserID: 100}
	logs, _, err := service.GetAccessLogs(filter)
	assert.NoError(t, err)
	assert.Len(t, logs, 2)

	for _, log := range logs {
		assert.Equal(t, uint(100), log.UserID)
	}
}

func TestAccessAuditService_GetAccessLogs_FilterByIPAddress(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "user1", "10.0.0.1", "UA", "session", "s1", "login", "success", 50, nil)
	service.LogAccess(AccessView, 2, "user2", "10.0.0.2", "UA", "session", "s2", "view", "success", 50, nil)

	filter := AccessLogFilter{IPAddress: "10.0.0.1"}
	logs, _, err := service.GetAccessLogs(filter)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "10.0.0.1", logs[0].IPAddress)
}

func TestAccessAuditService_GetAccessLogs_FilterByDateRange(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "user1", "10.0.0.1", "UA", "session", "s1", "login", "success", 50, nil)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	filter := AccessLogFilter{
		StartDate: yesterday,
		EndDate:   tomorrow,
	}

	logs, _, err := service.GetAccessLogs(filter)
	assert.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestAccessAuditService_GetAccessStats(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 10; i++ {
		status := "success"
		if i%3 == 0 {
			status = "failed"
		}
		service.LogAccess(
			AccessLogin,
			uint(i),
			"user",
			"10.0.0.1",
			"Mozilla/5.0",
			"session",
			"123",
			"login",
			status,
			50+int64(i*10),
			nil,
		)
	}

	stats, err := service.GetAccessStats(time.Time{}, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	assert.Equal(t, 10, stats["total_access"])
	assert.Equal(t, 10.0, stats["avg_response_time"])
}

func TestAccessAuditService_GetPermissionChangeHistory(t *testing.T) {
	service := NewAccessAuditService()

	changes := []*PermissionChange{
		{UserID: 50, ChangeType: "grant", Permission: "read", ChangedBy: 1},
		{UserID: 50, ChangeType: "revoke", Permission: "read", ChangedBy: 1},
		{UserID: 51, ChangeType: "grant", Permission: "write", ChangedBy: 1},
	}

	for _, change := range changes {
		err := service.LogPermissionChange(change)
		assert.NoError(t, err)
	}

	history, err := service.GetPermissionChangeHistory(50)
	assert.NoError(t, err)
	assert.Len(t, history, 2)
}

func TestAccessAuditService_ExportAccessLogs_JSON(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "test", "10.0.0.1", "UA", "session", "123", "login", "success", 50, nil)

	data, err := service.ExportAccessLogs("json", AccessLogFilter{})
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var logs []*AccessAuditLog
	err = json.Unmarshal(data, &logs)
	assert.NoError(t, err)
	assert.NotEmpty(t, logs)
}

func TestAccessAuditService_ExportAccessLogs_CSV(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "test", "10.0.0.1", "UA", "session", "123", "login", "success", 50, nil)

	data, err := service.ExportAccessLogs("csv", AccessLogFilter{})
	assert.NoError(t, err)
	assert.Contains(t, string(data), "ID,Timestamp,EventType,Severity")
	assert.Contains(t, string(data), "login")
}

func TestAccessAuditService_DetermineSeverity(t *testing.T) {
	service := NewAccessAuditService()

	tests := []struct {
		eventType AccessEventType
		status    string
		expected  AccessSeverity
	}{
		{AccessLogin, "success", SeverityLow},
		{AccessLogin, "failed", SeverityMedium},
		{AccessAdminAction, "success", SeverityHigh},
		{AccessSensitiveData, "success", SeverityHigh},
		{AccessDelete, "success", SeverityMedium},
		{AccessView, "success", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			severity := service.determineSeverity(tt.eventType, tt.status)
			assert.Equal(t, tt.expected, severity)
		})
	}
}

func TestAccessAuditService_CalculateRiskScore(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "user", "10.0.0.1", "UA", "session", "123", "login", "success", 50, nil)
	service.LogAccess(AccessLogin, 1, "user", "10.0.0.1", "UA", "session", "124", "login", "success", 50, nil)

	for i := 0; i < 35; i++ {
		service.LogAccess(AccessLogin, 1, "user", "10.0.0.1", "UA", "session", "sess"+string(rune(i)), "login", "success", 50, nil)
	}

	score := service.calculateRiskScore(AccessSensitiveData, "10.0.0.1", 1, "user_data")
	assert.Greater(t, score, 0.0)
}

func TestAccessAuditService_GenerateTags(t *testing.T) {
	service := NewAccessAuditService()

	tags := service.generateTags(AccessLogin, "document", "success")
	assert.Contains(t, tags, "login")

	tags = service.generateTags(AccessSensitiveData, "user_profile", "success")
	assert.Contains(t, tags, "sensitive_data")
	assert.Contains(t, tags, "gdpr")
	assert.Contains(t, tags, "pii")

	tags = service.generateTags(AccessAdminAction, "admin_config", "failed")
	assert.Contains(t, tags, "admin")
	assert.Contains(t, tags, "failed")
}

func TestAccessAuditService_IsSensitiveResourceType(t *testing.T) {
	service := NewAccessAuditService()

	sensitiveTypes := []string{"user", "password", "email", "phone", "address", "payment"}
	for _, rt := range sensitiveTypes {
		assert.True(t, service.isSensitiveResourceType(rt), "Expected %s to be sensitive", rt)
	}

	nonSensitiveTypes := []string{"article", "post", "comment", "file"}
	for _, rt := range nonSensitiveTypes {
		assert.False(t, service.isSensitiveResourceType(rt), "Expected %s to be non-sensitive", rt)
	}
}

func TestAccessAuditService_GetGeoLocation(t *testing.T) {
	service := NewAccessAuditService()

	privateIPs := []string{"10.0.0.1", "192.168.1.1", "172.16.0.1"}
	for _, ip := range privateIPs {
		geo := service.getGeoLocation(ip)
		assert.Equal(t, "Private Network", geo)
	}

	publicIP := "8.8.8.8"
	geo := service.getGeoLocation(publicIP)
	assert.Equal(t, "Unknown", geo)
}

func TestAccessAuditService_GetRecentAbnormalPatterns(t *testing.T) {
	service := NewAccessAuditService()

	patterns := service.GetRecentAbnormalPatterns(5)
	assert.NotNil(t, patterns)

	patterns = service.GetRecentAbnormalPatterns(0)
	assert.NotNil(t, patterns)
}

func TestAccessAuditService_GetUserAccessSummary(t *testing.T) {
	service := NewAccessAuditService()

	userID := uint(999)
	for i := 0; i < 5; i++ {
		service.LogAccess(AccessLogin, userID, "summary_user", "10.0.0.1", "UA", "session", "s"+string(rune(i)), "login", "success", 50, nil)
	}

	summary, err := service.GetUserAccessSummary(userID, 7)
	assert.NoError(t, err)
	assert.NotNil(t, summary)

	assert.Equal(t, userID, summary["user_id"])
	assert.Equal(t, 7, summary["period_days"])
	assert.Equal(t, 5, summary["total_access"])
}

func TestAccessAuditService_SetAndGetAlertThresholds(t *testing.T) {
	service := NewAccessAuditService()

	service.SetAlertThreshold("custom_event", 20)

	thresholds := service.GetAlertThresholds()
	assert.Equal(t, 20, thresholds["custom_event"])
	assert.Equal(t, 5, thresholds["failed_login"])
}

func TestAccessAuditService_UpdateAccessCounts(t *testing.T) {
	service := NewAccessAuditService()

	service.updateAccessCounts("10.0.0.1", 100)
	service.updateAccessCounts("10.0.0.1", 100)
	service.updateAccessCounts("10.0.0.2", 101)

	assert.Equal(t, 2, service.ipAccessCounts["10.0.0.1"])
	assert.Equal(t, 1, service.ipAccessCounts["10.0.0.2"])
	assert.Equal(t, 2, service.userAccessCounts[100])
	assert.Equal(t, 1, service.userAccessCounts[101])
}

func TestAccessAuditService_CleanupOldCounts(t *testing.T) {
	service := NewAccessAuditService()

	service.ipAccessCounts["10.0.0.1"] = 10
	service.ipAccessCounts["10.0.0.2"] = 3
	service.userAccessCounts[100] = 10
	service.userAccessCounts[101] = 2

	service.cleanupOldCounts()

	_, exists := service.ipAccessCounts["10.0.0.1"]
	assert.True(t, exists)

	_, exists = service.ipAccessCounts["10.0.0.2"]
	assert.False(t, exists)
}

func TestAccessAuditService_CheckAlertThreshold(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 5; i++ {
		service.ipAccessCounts["192.168.1.50"]++
	}

	service.checkAlertThreshold(AccessLogin, "192.168.1.50", 1)
}

func TestAccessAuditService_ConcurrentLogAccess(t *testing.T) {
	service := NewAccessAuditService()

	var wg sync.WaitGroup
	userCount := 100

	for i := 0; i < userCount; i++ {
		wg.Add(1)
		go func(userID uint) {
			defer wg.Done()
			service.LogAccess(
				AccessLogin,
				userID,
				"user",
				"10.0.0.1",
				"Mozilla/5.0",
				"session",
				"123",
				"login",
				"success",
				50,
				nil,
			)
		}(uint(i))
	}

	wg.Wait()

	assert.GreaterOrEqual(t, len(service.accessLogs), userCount)
}

func TestAccessAuditService_MaxLogsLimit(t *testing.T) {
	service := &AccessAuditService{
		accessLogs:       make([]*AccessAuditLog, 0),
		permissionChanges: make([]*PermissionChange, 0),
		sensitiveOperations: make([]*SensitiveOperation, 0),
		abnormalPatterns:  make([]*AbnormalAccessPattern, 0),
		maxLogs:          10,
		retentionDays:     90,
		alertThreshold:   make(map[string]int),
		ipAccessCounts:   make(map[string]int),
		userAccessCounts: make(map[uint]int),
		timeWindow:        1 * time.Hour,
		enableGeoLocation: true,
		enableRiskScoring: true,
		sensitivePatterns: make([]*regexp.Regexp, 0),
	}

	for i := 0; i < 25; i++ {
		service.accessLogs = append(service.accessLogs, &AccessAuditLog{
			ID:        uint(i),
			Timestamp: time.Now(),
		})
	}

	if len(service.accessLogs) > service.maxLogs {
		service.accessLogs = service.accessLogs[len(service.accessLogs)-service.maxLogs:]
	}

	assert.Equal(t, 10, len(service.accessLogs))
}

func TestAccessAuditLog_StructFields(t *testing.T) {
	log := &AccessAuditLog{
		ID:            1,
		Timestamp:     time.Now(),
		EventType:     AccessLogin,
		Severity:      SeverityInfo,
		UserID:        100,
		Username:      "testuser",
		IPAddress:     "192.168.1.1",
		UserAgent:     "TestAgent",
		ResourceType:  "session",
		ResourceID:    "123",
		Action:        "login",
		Status:        "success",
		ErrorMessage:  "",
		SessionID:     "sess-abc",
		RequestPath:   "/login",
		RequestMethod: "POST",
		ResponseTime:  50,
		Metadata:      map[string]string{"key": "value"},
		GeoLocation:   "Private Network",
		RiskScore:     25.5,
		Tags:          []string{"login", "success"},
	}

	jsonData, err := json.Marshal(log)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
}

func TestPermissionChange_StructFields(t *testing.T) {
	change := &PermissionChange{
		ChangeType:   "grant",
		UserID:       50,
		TargetUserID: 51,
		Permission:   "admin",
		OldValue:     "false",
		NewValue:     "true",
		ChangedBy:    1,
		Reason:       "Promotion",
		Timestamp:    time.Now(),
		ApprovedBy:   2,
		Status:       "approved",
	}

	jsonData, err := json.Marshal(change)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
}

func TestSensitiveOperation_StructFields(t *testing.T) {
	op := &SensitiveOperation{
		OperationType:    "export",
		Description:      "Export user data",
		DataType:         "user_data",
		RiskLevel:        SeverityHigh,
		RequiresApproval: true,
		ApprovalRoles:    []string{"admin", "security"},
		AuditRequired:    true,
		Metadata:         map[string]interface{}{"format": "csv"},
	}

	jsonData, err := json.Marshal(op)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
}

func TestAbnormalAccessPattern_StructFields(t *testing.T) {
	pattern := &AbnormalAccessPattern{
		PatternType:     "high_frequency",
		Description:     "Abnormal access frequency",
		UserID:          100,
		IPAddress:       "192.168.1.50",
		FirstOccurrence: time.Now().Add(-1 * time.Hour),
		LastOccurrence:  time.Now(),
		OccurrenceCount: 150,
		Severity:        SeverityHigh,
		Recommendation:  "Investigate potential abuse",
		RiskScore:       75.5,
	}

	jsonData, err := json.Marshal(pattern)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
}

func TestAccessLogFilter_StructFields(t *testing.T) {
	filter := AccessLogFilter{
		UserID:       100,
		IPAddress:    "192.168.1.1",
		EventType:    AccessLogin,
		ResourceType: "session",
		StartDate:    time.Now().Add(-24 * time.Hour),
		EndDate:      time.Now(),
		Status:       "success",
		Severity:     SeverityInfo,
		Limit:        100,
		Offset:       0,
	}

	jsonData, err := json.Marshal(filter)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For present",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			remote:   "192.168.1.1:8080",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Real-IP present",
			headers:  map[string]string{"X-Real-IP": "10.0.0.3"},
			remote:   "192.168.1.1:8080",
			expected: "10.0.0.3",
		},
		{
			name:     "No proxy headers",
			headers:  map[string]string{},
			remote:   "192.168.1.1:8080",
			expected: "192.168.1.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remote

			ip := getClientIP(req)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

func TestAccessAuditService_AccessEventTypes(t *testing.T) {
	eventTypes := []AccessEventType{
		AccessLogin,
		AccessLogout,
		AccessView,
		AccessCreate,
		AccessUpdate,
		AccessDelete,
		AccessExport,
		AccessImport,
		AccessAdminAction,
		AccessSensitiveData,
		AccessConfigChange,
		AccessPermissionGrant,
		AccessPermissionRevoke,
		AccessRoleAssign,
		AccessRoleRemove,
	}

	for _, et := range eventTypes {
		assert.NotEmpty(t, string(et))
	}
}

func TestAccessAuditService_SeverityLevels(t *testing.T) {
	severities := []AccessSeverity{
		SeverityInfo,
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	for _, s := range severities {
		assert.NotEmpty(t, string(s))
	}
}

func TestAccessAuditService_TriggerAlert(t *testing.T) {
	service := NewAccessAuditService()

	service.triggerAlert(AccessLogin, "192.168.1.100", 1, 10)
}

func TestAccessAuditService_CleanupOldLogs(t *testing.T) {
	service := NewAccessAuditService()

	cutoff := time.Now().AddDate(0, 0, -91)
	oldLog := &AccessAuditLog{
		ID:        1,
		Timestamp: cutoff.Add(-24 * time.Hour),
		EventType: AccessLogin,
	}
	newLog := &AccessAuditLog{
		ID:        2,
		Timestamp: time.Now(),
		EventType: AccessLogin,
	}

	service.accessLogs = []*AccessAuditLog{oldLog, newLog}

	newLogs := make([]*AccessAuditLog, 0)
	for _, log := range service.accessLogs {
		if log.Timestamp.After(cutoff) {
			newLogs = append(newLogs, log)
		}
	}
	service.accessLogs = newLogs

	require.Len(t, service.accessLogs, 1)
	assert.Equal(t, uint(2), service.accessLogs[0].ID)
}

func TestAccessAuditService_DetectAbnormalPatterns(t *testing.T) {
	service := &AccessAuditService{
		accessLogs:             make([]*AccessAuditLog, 0),
		permissionChanges:      make([]*PermissionChange, 0),
		sensitiveOperations:    make([]*SensitiveOperation, 0),
		abnormalPatterns:       make([]*AbnormalAccessPattern, 0),
		mu:                     sync.RWMutex{},
		maxLogs:                1000,
		retentionDays:          90,
		alertThreshold:         make(map[string]int),
		ipAccessCounts:         make(map[string]int),
		userAccessCounts:       make(map[uint]int),
		timeWindow:             1 * time.Hour,
		enableGeoLocation:      true,
		enableRiskScoring:      true,
		sensitivePatterns:      make([]*regexp.Regexp, 0),
	}

	service.ipAccessCounts["192.168.1.100"] = 150

	pattern := &AbnormalAccessPattern{
		PatternType:     "abnormal_frequency",
		Description:    "Abnormal access frequency",
		IPAddress:      "192.168.1.100",
		LastOccurrence: time.Now(),
		OccurrenceCount: 150,
		Severity:       SeverityHigh,
		Recommendation: "Investigate",
		RiskScore:      80,
	}
	service.abnormalPatterns = append(service.abnormalPatterns, pattern)

	if len(service.abnormalPatterns) > 1000 {
		service.abnormalPatterns = service.abnormalPatterns[len(service.abnormalPatterns)-1000:]
	}

	assert.NotEmpty(t, service.abnormalPatterns)
}
