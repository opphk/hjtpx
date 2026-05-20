package service

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAccessAuditService_AllEventTypes(t *testing.T) {
	service := NewAccessAuditService()

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

	for _, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			log := service.LogAccess(
				eventType,
				1,
				"testuser",
				"192.168.1.100",
				"Mozilla/5.0",
				"session",
				"123",
				"action",
				"success",
				100,
				nil,
			)
			assert.NotNil(t, log)
			assert.Equal(t, eventType, log.EventType)
		})
	}
}

func TestAccessAuditService_AllSeverityLevels(t *testing.T) {
	service := NewAccessAuditService()

	severities := []AccessSeverity{
		SeverityInfo,
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	for _, severity := range severities {
		t.Run(string(severity), func(t *testing.T) {
			service.accessLogs = []*AccessAuditLog{}
			log := &AccessAuditLog{
				Timestamp: now(),
				EventType: AccessLogin,
				Severity:  severity,
				UserID:    1,
				Username: "testuser",
				Status:   "success",
			}
			service.accessLogs = append(service.accessLogs, log)
			assert.NotNil(t, log)
			assert.Equal(t, severity, log.Severity)
		})
	}
}

func TestAccessAuditService_ConcurrentLogAccess(t *testing.T) {
	service := NewAccessAuditService()

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(index int) {
			service.LogAccess(
				AccessLogin,
				uint(index),
				"user",
				"10.0.0.1",
				"UA",
				"session",
				"123",
				"login",
				"success",
				50,
				nil,
			)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	assert.GreaterOrEqual(t, len(service.accessLogs), 100)
}

func TestAccessAuditService_FilterCombinations(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 100, "user1", "10.0.0.1", "UA1", "session", "s1", "login", "success", 50, nil)
	service.LogAccess(AccessView, 200, "user2", "10.0.0.2", "UA2", "session", "s2", "view", "success", 50, nil)
	service.LogAccess(AccessLogin, 100, "user1", "10.0.0.3", "UA1", "session", "s3", "login", "failed", 50, nil)

	testCases := []struct {
		name   string
		filter AccessLogFilter
		expect int
	}{
		{
			name:   "Filter by UserID 100",
			filter: AccessLogFilter{UserID: 100},
			expect: 2,
		},
		{
			name:   "Filter by UserID 200",
			filter: AccessLogFilter{UserID: 200},
			expect: 1,
		},
		{
			name:   "Filter by IP 10.0.0.1",
			filter: AccessLogFilter{IPAddress: "10.0.0.1"},
			expect: 1,
		},
		{
			name:   "Filter by EventType Login",
			filter: AccessLogFilter{EventType: AccessLogin},
			expect: 2,
		},
		{
			name:   "Filter by Status failed",
			filter: AccessLogFilter{Status: "failed"},
			expect: 1,
		},
		{
			name:   "Combined UserID and EventType",
			filter: AccessLogFilter{UserID: 100, EventType: AccessLogin},
			expect: 2,
		},
		{
			name:   "Limit 1",
			filter: AccessLogFilter{Limit: 1},
			expect: 1,
		},
		{
			name:   "Offset 1",
			filter: AccessLogFilter{Offset: 1},
			expect: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs, _, err := service.GetAccessLogs(tc.filter)
			assert.NoError(t, err)
			assert.Len(t, logs, tc.expect)
		})
	}
}

func TestAccessAuditService_AlertThresholds(t *testing.T) {
	service := NewAccessAuditService()

	service.SetAlertThreshold("failed_login", 3)
	service.SetAlertThreshold("admin_action", 5)

	thresholds := service.GetAlertThresholds()

	assert.Equal(t, 3, thresholds["failed_login"])
	assert.Equal(t, 5, thresholds["admin_action"])
}

func TestAccessAuditService_PermissionChangeHistory(t *testing.T) {
	service := NewAccessAuditService()

	changes := []*PermissionChange{
		{UserID: 50, ChangeType: "grant", Permission: "read"},
		{UserID: 50, ChangeType: "grant", Permission: "write"},
		{UserID: 50, ChangeType: "revoke", Permission: "read"},
		{UserID: 51, ChangeType: "grant", Permission: "admin"},
	}

	for _, change := range changes {
		err := service.LogPermissionChange(change)
		assert.NoError(t, err)
	}

	history50, err := service.GetPermissionChangeHistory(50)
	assert.NoError(t, err)
	assert.Len(t, history50, 3)

	history51, err := service.GetPermissionChangeHistory(51)
	assert.NoError(t, err)
	assert.Len(t, history51, 1)

	history52, err := service.GetPermissionChangeHistory(52)
	assert.NoError(t, err)
	assert.Empty(t, history52)
}

func TestAccessAuditService_ExportFormats(t *testing.T) {
	service := NewAccessAuditService()

	service.LogAccess(AccessLogin, 1, "user", "10.0.0.1", "UA", "session", "123", "login", "success", 50, nil)

	jsonData, err := service.ExportAccessLogs("json", AccessLogFilter{})
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	assert.Contains(t, string(jsonData), "[")

	csvData, err := service.ExportAccessLogs("csv", AccessLogFilter{})
	assert.NoError(t, err)
	assert.NotEmpty(t, csvData)
	assert.Contains(t, string(csvData), "ID,Timestamp")

	unknownData, err := service.ExportAccessLogs("unknown", AccessLogFilter{})
	assert.NoError(t, err)
	assert.NotEmpty(t, unknownData)
}

func TestAccessAuditService_SensitivePatterns(t *testing.T) {
	service := NewAccessAuditService()

	assert.NotEmpty(t, service.sensitivePatterns)

	for _, pattern := range service.sensitivePatterns {
		assert.NotNil(t, pattern)
	}
}

func TestAccessAuditService_AccessStats(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 10; i++ {
		status := "success"
		if i%2 == 0 {
			status = "failed"
		}
		eventType := AccessLogin
		if i%3 == 0 {
			eventType = AccessView
		}
		service.LogAccess(
			eventType,
			uint(i%3),
			"user",
			"10.0.0.1",
			"UA",
			"session",
			"123",
			"action",
			status,
			int64(50+i*10),
			nil,
		)
	}

	stats, err := service.GetAccessStats(time.Time{}, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	assert.Greater(t, stats["total_access"].(int), 0)
	assert.Contains(t, stats["by_event_type"], "login")
	assert.Contains(t, stats["by_severity"], "info")
	assert.Contains(t, stats["by_status"], "success")
	assert.Greater(t, stats["unique_users_count"].(int), 0)
	assert.Greater(t, stats["unique_ips_count"].(int), 0)
	assert.Greater(t, stats["avg_response_time"].(float64), 0.0)
}

func TestAccessAuditService_UserAccessSummary(t *testing.T) {
	service := NewAccessAuditService()

	userID := uint(999)

	for i := 0; i < 5; i++ {
		eventType := AccessLogin
		if i%2 == 0 {
			eventType = AccessView
		}
		service.LogAccess(
			eventType,
			userID,
			"summary_user",
			"10.0.0.1",
			"UA",
			"session",
			"s"+string(rune(i)),
			"action",
			"success",
			50,
			nil,
		)
	}

	summary, err := service.GetUserAccessSummary(userID, 30)
	assert.NoError(t, err)
	assert.NotNil(t, summary)

	assert.Equal(t, userID, summary["user_id"])
	assert.Equal(t, 30, summary["period_days"])
	assert.Equal(t, 5, summary["total_access"])
	assert.Contains(t, summary["by_event_type"], "login")
	assert.Contains(t, summary["by_severity"], "info")
	assert.Equal(t, 1, summary["unique_ips_count"])
}

func TestAccessAuditService_AbnormalPatterns(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 150; i++ {
		service.ipAccessCounts["192.168.1.100"]++
	}

	patterns, err := service.DetectAbnormalAccess(1, "192.168.1.100")
	assert.NoError(t, err)
	assert.NotEmpty(t, patterns)

	pattern := patterns[0]
	assert.Equal(t, "high_ip_frequency", pattern.PatternType)
	assert.Equal(t, SeverityMedium, pattern.Severity)
	assert.Greater(t, pattern.RiskScore, 0.0)
}

func TestAccessAuditService_GetRecentAbnormalPatterns(t *testing.T) {
	service := NewAccessAuditService()

	for i := 0; i < 5; i++ {
		pattern := &AbnormalAccessPattern{
			PatternType:     "test_pattern",
			Description:     "Test description",
			LastOccurrence:  time.Now(),
			OccurrenceCount: i + 1,
		}
		service.abnormalPatterns = append(service.abnormalPatterns, pattern)
	}

	patterns := service.GetRecentAbnormalPatterns(3)
	assert.Len(t, patterns, 3)

	patterns = service.GetRecentAbnormalPatterns(10)
	assert.Len(t, patterns, 5)
}

func TestAccessAuditService_GeoLocationDetection(t *testing.T) {
	service := NewAccessAuditService()

	testCases := []struct {
		ip       string
		expected string
	}{
		{"10.0.0.1", "Private Network"},
		{"10.255.255.255", "Private Network"},
		{"172.16.0.1", "Private Network"},
		{"172.31.255.255", "Private Network"},
		{"192.168.0.1", "Private Network"},
		{"192.168.255.255", "Private Network"},
		{"8.8.8.8", "Unknown"},
		{"1.1.1.1", "Unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			geo := service.getGeoLocation(tc.ip)
			assert.Equal(t, tc.expected, geo)
		})
	}
}

func TestAccessAuditService_SensitiveResourceTypes(t *testing.T) {
	service := NewAccessAuditService()

	sensitiveTypes := []string{
		"user",
		"password",
		"email",
		"phone",
		"address",
		"payment",
		"financial",
		"UserProfile",
		"PASSWORD",
		"CreditCard",
	}

	for _, rt := range sensitiveTypes {
		assert.True(t, service.isSensitiveResourceType(rt), "Expected %s to be sensitive", rt)
	}

	nonSensitive := []string{
		"article",
		"post",
		"comment",
		"file",
		"document",
		"content",
	}

	for _, rt := range nonSensitive {
		assert.False(t, service.isSensitiveResourceType(rt), "Expected %s to be non-sensitive", rt)
	}
}

func TestAccessAuditService_RiskScoreCalculation(t *testing.T) {
	service := NewAccessAuditService()

	service.ipAccessCounts["10.0.0.1"] = 35
	service.userAccessCounts[1] = 55

	score := service.calculateRiskScore(AccessSensitiveData, "10.0.0.1", 1, "user_data")

	assert.Greater(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)

	service.ipAccessCounts["10.0.0.1"] = 0
	service.userAccessCounts[1] = 0

	score = service.calculateRiskScore(AccessLogin, "10.0.0.1", 1, "document")

	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestAccessAuditService_TagGeneration(t *testing.T) {
	service := NewAccessAuditService()

	testCases := []struct {
		eventType   AccessEventType
		resourceType string
		status      string
		expectTags  []string
	}{
		{
			eventType:    AccessLogin,
			resourceType: "session",
			status:       "success",
			expectTags:   []string{"login"},
		},
		{
			eventType:    AccessSensitiveData,
			resourceType: "user_profile",
			status:       "success",
			expectTags:   []string{"sensitive_data", "gdpr", "pii"},
		},
		{
			eventType:    AccessAdminAction,
			resourceType: "admin_config",
			status:       "failed",
			expectTags:   []string{"admin_action", "admin", "failed"},
		},
		{
			eventType:    AccessLogin,
			resourceType: "password",
			status:       "success",
			expectTags:   []string{"login", "pii"},
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.eventType), func(t *testing.T) {
			tags := service.generateTags(tc.eventType, tc.resourceType, tc.status)
			for _, expectTag := range tc.expectTags {
				assert.Contains(t, tags, expectTag)
			}
		})
	}
}

func TestAccessAuditService_UpdateAndCleanupCounts(t *testing.T) {
	service := NewAccessAuditService()

	service.updateAccessCounts("10.0.0.1", 100)
	service.updateAccessCounts("10.0.0.1", 100)
	service.updateAccessCounts("10.0.0.2", 101)

	assert.Equal(t, 2, service.ipAccessCounts["10.0.0.1"])
	assert.Equal(t, 1, service.ipAccessCounts["10.0.0.2"])
	assert.Equal(t, 2, service.userAccessCounts[100])
	assert.Equal(t, 1, service.userAccessCounts[101])

	service.ipAccessCounts["10.0.0.1"] = 10
	service.ipAccessCounts["10.0.0.2"] = 3
	service.ipAccessCounts["10.0.0.3"] = 2

	service.cleanupOldCounts()

	_, exists1 := service.ipAccessCounts["10.0.0.1"]
	assert.True(t, exists1)

	_, exists2 := service.ipAccessCounts["10.0.0.2"]
	assert.True(t, exists2)

	_, exists3 := service.ipAccessCounts["10.0.0.3"]
	assert.False(t, exists3)
}

func TestAccessAuditService_SeverityDetermination(t *testing.T) {
	service := NewAccessAuditService()

	testCases := []struct {
		eventType AccessEventType
		status    string
		expected  AccessSeverity
	}{
		{AccessLogin, "success", SeverityLow},
		{AccessLogin, "failed", SeverityMedium},
		{AccessLogout, "success", SeverityLow},
		{AccessView, "success", SeverityInfo},
		{AccessCreate, "success", SeverityInfo},
		{AccessUpdate, "success", SeverityInfo},
		{AccessDelete, "success", SeverityMedium},
		{AccessExport, "success", SeverityMedium},
		{AccessImport, "success", SeverityInfo},
		{AccessAdminAction, "success", SeverityHigh},
		{AccessSensitiveData, "success", SeverityHigh},
		{AccessConfigChange, "success", SeverityMedium},
		{AccessPermissionGrant, "success", SeverityHigh},
		{AccessPermissionRevoke, "success", SeverityHigh},
		{AccessRoleAssign, "success", SeverityHigh},
		{AccessRoleRemove, "success", SeverityHigh},
	}

	for _, tc := range testCases {
		t.Run(string(tc.eventType)+"_"+tc.status, func(t *testing.T) {
			severity := service.determineSeverity(tc.eventType, tc.status)
			assert.Equal(t, tc.expected, severity)
		})
	}
}

func TestAccessAuditService_AlertThresholdChecking(t *testing.T) {
	service := NewAccessAuditService()

	service.SetAlertThreshold("failed_login", 5)

	for i := 0; i < 4; i++ {
		service.ipAccessCounts["192.168.1.50"]++
	}

	service.checkAlertThreshold(AccessLogin, "192.168.1.50", 1)

	service.ipAccessCounts["192.168.1.50"]++
	service.checkAlertThreshold(AccessLogin, "192.168.1.50", 1)
}

func TestAccessAuditService_TriggerAlert(t *testing.T) {
	service := NewAccessAuditService()

	service.triggerAlert(AccessLogin, "192.168.1.100", 1, 10)
}

func TestAccessAuditService_MaxLogsEnforcement(t *testing.T) {
	service := &AccessAuditService{
		accessLogs:       make([]*AccessAuditLog, 0),
		permissionChanges: make([]*PermissionChange, 0),
		sensitiveOperations: make([]*SensitiveOperation, 0),
		abnormalPatterns:  make([]*AbnormalAccessPattern, 0),
		maxLogs:          5,
		retentionDays:     90,
		alertThreshold:   make(map[string]int),
		ipAccessCounts:   make(map[string]int),
		userAccessCounts: make(map[uint]int),
		timeWindow:        1 * time.Hour,
		enableGeoLocation: true,
		enableRiskScoring: true,
		sensitivePatterns: make([]*regexp.Regexp, 0),
	}

	for i := 0; i < 10; i++ {
		service.accessLogs = append(service.accessLogs, &AccessAuditLog{ID: uint(i)})
		if len(service.accessLogs) > service.maxLogs {
			service.accessLogs = service.accessLogs[len(service.accessLogs)-service.maxLogs:]
		}
	}

	assert.Len(t, service.accessLogs, 5)
	assert.Equal(t, uint(5), service.accessLogs[0].ID)
	assert.Equal(t, uint(9), service.accessLogs[4].ID)
}

func TestAccessAuditService_LogAccessWithMetadata(t *testing.T) {
	service := NewAccessAuditService()

	metadata := map[string]string{
		"session_id":   "sess-123",
		"request_path": "/api/users",
		"user_role":    "admin",
	}

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
		metadata,
	)

	assert.NotNil(t, log)
	assert.Equal(t, "sess-123", log.SessionID)
	assert.Contains(t, log.RequestPath, "/api/users")
	assert.Equal(t, "admin", log.Metadata["user_role"])
}

func TestAccessAuditService_LogSensitiveOperationVariations(t *testing.T) {
	service := NewAccessAuditService()

	operations := []struct {
		operationType string
		description   string
		dataType     string
		riskLevel    AccessSeverity
	}{
		{"export", "Export user data", "user_data", SeverityHigh},
		{"delete", "Delete user account", "user_account", SeverityCritical},
		{"view", "View sensitive document", "sensitive_document", SeverityMedium},
		{"modify", "Modify payment info", "payment_info", SeverityCritical},
	}

	for _, op := range operations {
		t.Run(op.operationType, func(t *testing.T) {
			err := service.LogSensitiveOperation(
				op.operationType,
				op.description,
				op.dataType,
				1,
				"user",
				"10.0.0.1",
				op.riskLevel,
			)
			assert.NoError(t, err)
		})
	}
}

func TestAccessAuditService_PermissionChangeVariations(t *testing.T) {
	service := NewAccessAuditService()

	changes := []*PermissionChange{
		{UserID: 1, ChangeType: "grant", Permission: "read", NewValue: "allowed"},
		{UserID: 2, ChangeType: "grant", Permission: "write", NewValue: "allowed"},
		{UserID: 3, ChangeType: "revoke", Permission: "admin", OldValue: "granted"},
		{UserID: 4, ChangeType: "update", Permission: "role", OldValue: "user", NewValue: "admin"},
	}

	for _, change := range changes {
		err := service.LogPermissionChange(change)
		assert.NoError(t, err)
	}
}

func TestAccessAuditService_DetectAbnormalPatterns_Detailed(t *testing.T) {
	service := NewAccessAuditService()

	service.ipAccessCounts["192.168.1.100"] = 60
	service.userAccessCounts[100] = 110

	patterns, err := service.DetectAbnormalAccess(100, "192.168.1.100")
	assert.NoError(t, err)
	assert.NotEmpty(t, patterns)

	hasHighIP := false
	hasHighUser := false

	for _, p := range patterns {
		if p.PatternType == "high_ip_frequency" {
			hasHighIP = true
			assert.Equal(t, "192.168.1.100", p.IPAddress)
		}
		if p.PatternType == "high_user_frequency" {
			hasHighUser = true
			assert.Equal(t, uint(100), p.UserID)
		}
	}

	assert.True(t, hasHighIP || hasHighUser)
}

func TestAccessAuditService_GetAccessStats_EmptyLogs(t *testing.T) {
	service := &AccessAuditService{
		accessLogs:       make([]*AccessAuditLog, 0),
		permissionChanges: make([]*PermissionChange, 0),
		sensitiveOperations: make([]*SensitiveOperation, 0),
		abnormalPatterns:  make([]*AbnormalAccessPattern, 0),
		mu:               sync.RWMutex{},
	}

	stats, err := service.GetAccessStats(time.Time{}, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, 0, stats["total_access"])
}

func TestAccessAuditService_GetUserAccessSummary_NoLogs(t *testing.T) {
	service := &AccessAuditService{
		accessLogs:       make([]*AccessAuditLog, 0),
		permissionChanges: make([]*PermissionChange, 0),
		sensitiveOperations: make([]*SensitiveOperation, 0),
		abnormalPatterns:  make([]*AbnormalAccessPattern, 0),
		mu:               sync.RWMutex{},
	}

	summary, err := service.GetUserAccessSummary(999, 7)
	assert.NoError(t, err)
	assert.Equal(t, 0, summary["total_access"])
}

func TestAccessAuditService_StructValidation(t *testing.T) {
	log := &AccessAuditLog{
		ID:            1,
		Timestamp:     now(),
		EventType:     AccessLogin,
		Severity:      SeverityInfo,
		UserID:        100,
		Username:      "user",
		IPAddress:     "10.0.0.1",
		UserAgent:     "UA",
		ResourceType:  "session",
		ResourceID:    "123",
		Action:        "login",
		Status:        "success",
		SessionID:     "sess",
		RequestPath:   "/",
		RequestMethod: "POST",
		ResponseTime:  50,
		RiskScore:     25.0,
		Tags:          []string{"tag1", "tag2"},
	}

	assert.Equal(t, uint(1), log.ID)
	assert.Equal(t, AccessLogin, log.EventType)
	assert.Equal(t, SeverityInfo, log.Severity)
	assert.Equal(t, uint(100), log.UserID)
	assert.Equal(t, int64(50), log.ResponseTime)
	assert.Equal(t, 25.0, log.RiskScore)
}

func TestPermissionChange_StructValidation(t *testing.T) {
	change := &PermissionChange{
		ChangeType:   "grant",
		UserID:        50,
		TargetUserID: 51,
		Permission:   "admin",
		OldValue:     "user",
		NewValue:     "admin",
		ChangedBy:    1,
		Reason:       "Promotion",
		Timestamp:    now(),
		ApprovedBy:   2,
		Status:       "approved",
	}

	assert.Equal(t, "grant", change.ChangeType)
	assert.Equal(t, uint(50), change.UserID)
	assert.Equal(t, uint(51), change.TargetUserID)
	assert.Equal(t, "admin", change.Permission)
}

func TestSensitiveOperation_StructValidation(t *testing.T) {
	op := &SensitiveOperation{
		OperationType:    "export",
		Description:      "Export user data",
		DataType:         "user_data",
		RiskLevel:        SeverityHigh,
		RequiresApproval: true,
		ApprovalRoles:    []string{"admin", "security"},
		AuditRequired:    true,
	}

	assert.Equal(t, "export", op.OperationType)
	assert.Equal(t, SeverityHigh, op.RiskLevel)
	assert.True(t, op.RequiresApproval)
	assert.True(t, op.AuditRequired)
	assert.Len(t, op.ApprovalRoles, 2)
}

func TestAbnormalAccessPattern_StructValidation(t *testing.T) {
	pattern := &AbnormalAccessPattern{
		PatternType:     "high_frequency",
		Description:     "Too many requests",
		UserID:          100,
		IPAddress:       "10.0.0.1",
		FirstOccurrence: now().Add(-1 * time.Hour),
		LastOccurrence:  now(),
		OccurrenceCount: 150,
		Severity:        SeverityHigh,
		Recommendation:  "Investigate",
		RiskScore:       75.0,
	}

	assert.Equal(t, "high_frequency", pattern.PatternType)
	assert.Equal(t, uint(100), pattern.UserID)
	assert.Equal(t, 150, pattern.OccurrenceCount)
	assert.Equal(t, SeverityHigh, pattern.Severity)
}

func TestAccessLogFilter_StructValidation(t *testing.T) {
	filter := AccessLogFilter{
		UserID:       100,
		IPAddress:    "10.0.0.1",
		EventType:    AccessLogin,
		ResourceType: "session",
		StartDate:    now().Add(-24 * time.Hour),
		EndDate:      now(),
		Status:       "success",
		Severity:     SeverityInfo,
		Limit:        100,
		Offset:       0,
	}

	assert.Equal(t, uint(100), filter.UserID)
	assert.Equal(t, AccessLogin, filter.EventType)
	assert.Equal(t, 100, filter.Limit)
}

func now() time.Time {
	return time.Now()
}
