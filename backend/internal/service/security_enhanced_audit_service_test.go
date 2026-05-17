package service

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSecurityEnhancedAuditService(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.anomalyPatterns)
	assert.NotNil(t, svc.complianceRules)
	assert.Greater(t, len(svc.anomalyPatterns), 0)
	assert.Greater(t, len(svc.complianceRules), 0)
}

func TestSecurityEnhancedAuditService_LogOperation(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("LogBasicOperation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		record := svc.LogOperation("data_access", "user_data", "GET", req, nil)
		assert.NotNil(t, record)
		assert.Equal(t, AuditCategoryOperation, record.Category)
		assert.Equal(t, AuditSeverityInfo, record.Severity)
		assert.Equal(t, "data_access", record.Operation)
		assert.Equal(t, "user_data", record.Resource)
	})

	t.Run("LogOperationWithDetails", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/upload", nil)
		req.RemoteAddr = "192.168.1.2:12345"

		details := map[string]interface{}{
			"file_size": 1024,
			"file_type": "image/png",
		}

		record := svc.LogOperation("file_upload", "file_storage", "POST", req, details)
		assert.NotNil(t, record)
		assert.Equal(t, "file_upload", record.Operation)
		assert.Equal(t, 1024, record.Details["file_size"])
	})
}

func TestSecurityEnhancedAuditService_LogSecurityEvent(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("LogCriticalEvent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.RemoteAddr = "192.168.1.3:12345"

		record := svc.LogSecurityEvent(AuditSeverityCritical, "unauthorized_access", req, nil)
		assert.NotNil(t, record)
		assert.Equal(t, AuditCategorySecurity, record.Category)
		assert.Equal(t, AuditSeverityCritical, record.Severity)
	})

	t.Run("LogWarningEvent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/sensitive", nil)
		req.RemoteAddr = "192.168.1.4:12345"

		record := svc.LogSecurityEvent(AuditSeverityWarning, "suspicious_access", req, nil)
		assert.NotNil(t, record)
		assert.Equal(t, AuditSeverityWarning, record.Severity)
	})
}

func TestSecurityEnhancedAuditService_LogAccessDenied(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("LogAccessDenied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/secret", nil)
		req.RemoteAddr = "192.168.1.5:12345"

		record := svc.LogAccessDenied("insufficient_permissions", req, nil)
		assert.NotNil(t, record)
		assert.Equal(t, AuditCategorySecurity, record.Category)
		assert.Equal(t, AuditStatusBlocked, record.Status)
		assert.Equal(t, "access_denied", record.Operation)
	})
}

func TestSecurityEnhancedAuditService_LogAnomaly(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("LogAnomalyDetected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.RemoteAddr = "192.168.1.6:12345"

		evidence := map[string]interface{}{
			"request_count": 100,
			"time_window":   "1m",
		}

		record := svc.LogAnomaly("rapid_requests", req, evidence)
		assert.NotNil(t, record)
		assert.Equal(t, AuditCategoryAnomaly, record.Category)
		assert.Equal(t, "anomaly_detected", record.Operation)
	})
}

func TestSecurityEnhancedAuditService_GetRecords(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("GetAllRecords", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.10:12345"
			svc.LogOperation("test_op", "test_resource", "GET", req, nil)
		}

		records := svc.GetRecords(nil)
		assert.GreaterOrEqual(t, len(records), 10)
	})

	t.Run("FilterByCategory", func(t *testing.T) {
		filter := &AuditFilter{Category: AuditCategoryOperation}
		records := svc.GetRecords(filter)
		for _, record := range records {
			assert.Equal(t, AuditCategoryOperation, record.Category)
		}
	})

	t.Run("FilterBySeverity", func(t *testing.T) {
		filter := &AuditFilter{Severity: AuditSeverityInfo}
		records := svc.GetRecords(filter)
		for _, record := range records {
			assert.Equal(t, AuditSeverityInfo, record.Severity)
		}
	})

	t.Run("FilterBySourceIP", func(t *testing.T) {
		ip := "192.168.1.100"
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip + ":12345"
		svc.LogOperation("test_ip", "test_resource", "GET", req, nil)

		filter := &AuditFilter{SourceIP: ip}
		records := svc.GetRecords(filter)
		for _, record := range records {
			assert.Equal(t, ip, record.SourceIP)
		}
	})

	t.Run("FilterByTimeRange", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		filter := &AuditFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		records := svc.GetRecords(filter)
		for _, record := range records {
			assert.True(t, record.Timestamp.After(startTime))
			assert.True(t, record.Timestamp.Before(endTime))
		}
	})

	t.Run("FilterWithLimit", func(t *testing.T) {
		filter := &AuditFilter{Limit: 5}
		records := svc.GetRecords(filter)
		assert.LessOrEqual(t, len(records), 5)
	})
}

func TestSecurityEnhancedAuditService_GetStatistics(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("InitialStatistics", func(t *testing.T) {
		stats := svc.GetStatistics()
		assert.Equal(t, 0, stats.TotalRecords)
	})

	t.Run("StatisticsAfterRecords", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.20:12345"
			svc.LogOperation("test", "resource", "GET", req, nil)
		}

		stats := svc.GetStatistics()
		assert.GreaterOrEqual(t, stats.TotalRecords, 5)
		assert.NotNil(t, stats.ByCategory)
		assert.NotNil(t, stats.BySeverity)
		assert.NotNil(t, stats.ByStatus)
	})
}

func TestSecurityEnhancedAuditService_AnomalyPatterns(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()

	t.Run("GetAnomalyPatterns", func(t *testing.T) {
		patterns := svc.GetAnomalyPatterns()
		assert.Greater(t, len(patterns), 0)

		for _, pattern := range patterns {
			assert.NotEmpty(t, pattern.PatternID)
			assert.NotEmpty(t, pattern.Name)
		}
	})

	t.Run("UpdateAnomalyPattern", func(t *testing.T) {
		err := svc.UpdateAnomalyPattern("rapid_requests", false)
		assert.NoError(t, err)

		patterns := svc.GetAnomalyPatterns()
		for _, pattern := range patterns {
			if pattern.PatternID == "rapid_requests" {
				assert.False(t, pattern.IsActive)
			}
		}

		err = svc.UpdateAnomalyPattern("rapid_requests", true)
		assert.NoError(t, err)
	})

	t.Run("UpdateNonExistentPattern", func(t *testing.T) {
		err := svc.UpdateAnomalyPattern("non_existent_pattern", true)
		assert.Error(t, err)
	})
}

func TestSecurityEnhancedAuditService_ComplianceRules(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()

	t.Run("GetComplianceRules", func(t *testing.T) {
		rules := svc.GetComplianceRules()
		assert.Greater(t, len(rules), 0)

		for _, rule := range rules {
			assert.NotEmpty(t, rule.ID)
			assert.NotEmpty(t, rule.Name)
			assert.NotEmpty(t, rule.Framework)
		}
	})

	t.Run("UpdateComplianceRule", func(t *testing.T) {
		err := svc.UpdateComplianceRule("gdpr_data_access", false)
		assert.NoError(t, err)

		rules := svc.GetComplianceRules()
		for _, rule := range rules {
			if rule.ID == "gdpr_data_access" {
				assert.False(t, rule.Enabled)
			}
		}

		err = svc.UpdateComplianceRule("gdpr_data_access", true)
		assert.NoError(t, err)
	})

	t.Run("UpdateNonExistentRule", func(t *testing.T) {
		err := svc.UpdateComplianceRule("non_existent_rule", true)
		assert.Error(t, err)
	})
}

func TestSecurityEnhancedAuditService_CheckCompliance(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()

	t.Run("ComplianceCheckWithContext", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/123", nil)
		req.RemoteAddr = "192.168.1.30:12345"

		context := map[string]interface{}{
			"logged":      true,
			"authorized":  true,
		}

		violations := svc.CheckCompliance(req, context)
		assert.NotNil(t, violations)
	})

	t.Run("ComplianceCheckWithUnloggedEvent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/sensitive", nil)
		req.RemoteAddr = "192.168.1.31:12345"

		context := map[string]interface{}{
			"logged":     false,
		}

		violations := svc.CheckCompliance(req, context)
		assert.Greater(t, len(violations), 0)
	})
}

func TestSecurityEnhancedAuditService_DetectAnomalies(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("DetectAnomalies", func(t *testing.T) {
		ip := "192.168.1.40"

		for i := 0; i < 20; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = ip + ":12345"
			svc.LogSecurityEvent(AuditSeverityWarning, "rapid_request", req, nil)
		}

		anomalies := svc.DetectAnomalies(ip, 5*time.Minute)
		assert.NotNil(t, anomalies)
	})

	t.Run("NoAnomalies", func(t *testing.T) {
		ip := "192.168.1.41"
		anomalies := svc.DetectAnomalies(ip, 5*time.Minute)
		assert.NotNil(t, anomalies)
	})
}

func TestSecurityEnhancedAuditService_RegisterHandlers(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()

	t.Run("RegisterAlertHandler", func(t *testing.T) {
		svc.RegisterAlertHandler(func(record *AuditRecord) {
		})
		assert.Greater(t, len(svc.alertHandlers), 0)
	})

	t.Run("RegisterAnomalyHandler", func(t *testing.T) {
		svc.RegisterAnomalyHandler(func(pattern *AnomalyPattern, evidence map[string]interface{}) {
		})
		assert.Greater(t, len(svc.anomalyHandlers), 0)
	})
}

func TestSecurityEnhancedAuditService_ExportRecords(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("ExportToJSON", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.50:12345"
			svc.LogOperation("export_test", "resource", "GET", req, nil)
		}

		data, err := svc.ExportRecords("json", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("ExportToCSV", func(t *testing.T) {
		data, err := svc.ExportRecords("csv", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), "ID,Timestamp")
	})

	t.Run("ExportWithFilter", func(t *testing.T) {
		filter := &AuditFilter{Category: AuditCategoryOperation}
		data, err := svc.ExportRecords("json", filter)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}

func TestSecurityEnhancedAuditService_ConcurrentAccess(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()

	t.Run("ConcurrentLogging", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.60:12345"
				svc.LogOperation("concurrent_test", "resource", "GET", req, nil)
			}(i)
		}
		wg.Wait()

		stats := svc.GetStatistics()
		assert.GreaterOrEqual(t, stats.TotalRecords, 40)
	})
}

func TestSecurityEnhancedAuditService_DefaultPatterns(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()

	expectedPatterns := []string{
		"rapid_requests",
		"unusual_hours",
		"high_error_rate",
		"data_exfiltration",
		"privilege_escalation",
		"brute_force",
	}

	for _, patternID := range expectedPatterns {
		t.Run(patternID, func(t *testing.T) {
			pattern, exists := svc.anomalyPatterns[patternID]
			assert.True(t, exists, "Pattern %s should exist", patternID)
			assert.NotEmpty(t, pattern.Name)
			assert.NotEmpty(t, pattern.Description)
			assert.True(t, pattern.IsActive)
		})
	}
}

func TestSecurityEnhancedAuditService_DefaultComplianceRules(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()

	expectedRules := []struct {
		id        string
		framework string
	}{
		{"gdpr_data_access", "GDPR"},
		{"pci_data_protection", "PCI-DSS"},
		{"hipaa_audit_controls", "HIPAA"},
		{"sox_access_control", "SOX"},
	}

	for _, rule := range expectedRules {
		t.Run(rule.id, func(t *testing.T) {
			r, exists := svc.complianceRules[rule.id]
			assert.True(t, exists, "Rule %s should exist", rule.id)
			assert.Equal(t, rule.framework, r.Framework)
			assert.True(t, r.Enabled)
			assert.NotEmpty(t, r.Checks)
		})
	}
}

func TestAuditRecord_Timestamp(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("TimestampSetCorrectly", func(t *testing.T) {
		before := time.Now()
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.70:12345"
		record := svc.LogOperation("timestamp_test", "resource", "GET", req, nil)
		after := time.Now()

		assert.True(t, record.Timestamp.After(before.Add(-time.Second)))
		assert.True(t, record.Timestamp.Before(after.Add(time.Second)))
	})
}

func TestAuditRecord_ID(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("UniqueIDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.80:12345"
			record := svc.LogOperation("unique_id_test", "resource", "GET", req, nil)
			assert.False(t, ids[record.ID], "Duplicate ID found: %s", record.ID)
			ids[record.ID] = true
		}
	})

	t.Run("IDFormat", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.81:12345"
		record := svc.LogOperation("id_format_test", "resource", "GET", req, nil)
		assert.Contains(t, record.ID, "AUD-")
	})
}

func TestSecurityEnhancedAuditService_MatchesFilter(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.90:12345"
	svc.LogOperation("filter_test", "resource", "GET", req, nil)

	t.Run("CategoryMismatch", func(t *testing.T) {
		filter := &AuditFilter{Category: AuditCategorySecurity}
		records := svc.GetRecords(filter)
		for _, record := range records {
			assert.Equal(t, AuditCategorySecurity, record.Category)
		}
	})

	t.Run("OperationMismatch", func(t *testing.T) {
		filter := &AuditFilter{Operation: "non_existent_operation"}
		records := svc.GetRecords(filter)
		for _, record := range records {
			assert.Equal(t, "non_existent_operation", record.Operation)
		}
	})
}

func TestSecurityEnhancedAuditService_ComplianceViolation(t *testing.T) {
	svc := NewSecurityEnhancedAuditService()
	svc.asyncMode = false

	t.Run("LogComplianceViolation", func(t *testing.T) {
		violation := &ComplianceViolation{
			RuleID:       "test_rule",
			RuleName:     "Test Compliance Rule",
			Framework:    "TEST",
			Violation:    "Test violation",
			Severity:     AuditSeverityWarning,
			Timestamp:    time.Now(),
			Evidence:     map[string]interface{}{"test": "evidence"},
			Remediation: "Fix the issue",
		}

		record := svc.LogComplianceViolation(violation)
		assert.NotNil(t, record)
		assert.Equal(t, AuditCategoryCompliance, record.Category)
		assert.Equal(t, AuditSeverityWarning, record.Severity)
		assert.Equal(t, AuditStatusFailure, record.Status)
	})
}
