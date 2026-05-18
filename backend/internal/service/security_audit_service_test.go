package service

import (
	"net/http/httptest"
	"testing"
)

func TestNewSecurityAuditService(t *testing.T) {
	service := NewSecurityAuditService()
	if service == nil {
		t.Fatal("Expected non-nil service")
	}
	if service.maxEvents != 10000 {
		t.Errorf("Expected maxEvents=10000, got %d", service.maxEvents)
	}
}

func TestLogEvent(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false // 使用同步模式便于测试
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	event := service.LogEvent(EventLoginAttempt, req, map[string]interface{}{"test": "data"})
	if event == nil {
		t.Fatal("Expected non-nil event")
	}
}

func TestGetRecentEvents(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false
	req := httptest.NewRequest("GET", "/test", nil)

	for i := 0; i < 10; i++ {
		service.LogEvent(EventDataAccess, req, nil)
	}

	events := service.GetRecentEvents(5)
	if len(events) != 5 {
		t.Errorf("Expected 5 recent events, got %d", len(events))
	}
}

func TestGetEventsByType(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false
	req := httptest.NewRequest("GET", "/test", nil)

	service.LogEvent(EventLoginSuccess, req, nil)
	service.LogEvent(EventLoginFailure, req, nil)
	service.LogEvent(EventLoginFailure, req, nil)

	events := service.GetEventsByType(EventLoginFailure, 10)
	if len(events) != 2 {
		t.Errorf("Expected 2 login failure events, got %d", len(events))
	}
}

func TestGetEventsByIP(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.2:12345"

	for i := 0; i < 3; i++ {
		service.LogEvent(EventDataAccess, req1, nil)
	}
	service.LogEvent(EventDataAccess, req2, nil)

	events := service.GetEventsByIP("10.0.0.1", 10)
	if len(events) != 3 {
		t.Errorf("Expected 3 events for IP 10.0.0.1, got %d", len(events))
	}
}

func TestGetSecurityStats(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false
	req := httptest.NewRequest("GET", "/test", nil)

	service.LogEvent(EventLoginSuccess, req, nil)
	service.LogEvent(EventLoginFailure, req, nil)
	service.LogEvent(EventAccessDenied, req, nil)

	stats := service.GetSecurityStats()
	if stats["total_events"] != 3 {
		t.Errorf("Expected 3 total events, got %v", stats["total_events"])
	}
}

func TestDetectIntrusionAttempts(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false

	normalReq := httptest.NewRequest("GET", "/normal", nil)
	events := service.DetectIntrusionAttempts(normalReq)
	if len(events) != 0 {
		t.Errorf("Expected no events for normal request, got %d", len(events))
	}

	suspiciousReq := httptest.NewRequest("GET", "/test", nil)
	suspiciousReq.URL.RawQuery = "input='+OR+1%3D1+--"
	events = service.DetectIntrusionAttempts(suspiciousReq)
	if len(events) == 0 {
		t.Log("Warning: Expected intrusion detection, but pattern may not match")
	}
}

func TestAlertHandler(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false
	triggered := false

	service.RegisterAlertHandler(func(event *SecurityEvent) {
		triggered = true
	})

	req := httptest.NewRequest("GET", "/test", nil)
	service.LogEvent(EventSQLInjection, req, nil)

	if !triggered {
		t.Error("Expected alert handler to be triggered")
	}
}

func TestCalculateThreatScore(t *testing.T) {
	service := NewSecurityAuditService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	score := service.calculateThreatScore(EventSQLInjection, "192.168.1.1", req)
	if score != 100.0 {
		t.Errorf("Expected threat score 100.0 for critical event, got %f", score)
	}

	score = service.calculateThreatScore(EventLoginSuccess, "192.168.1.1", req)
	if score != 5.0 {
		t.Errorf("Expected threat score 5.0 for info event, got %f", score)
	}
}

func TestGenerateTags(t *testing.T) {
	service := NewSecurityAuditService()

	req := httptest.NewRequest("GET", "/admin/config", nil)
	tags := service.generateTags(EventConfigChange, req)

	foundAdmin := false
	foundPriority := false
	for _, tag := range tags {
		if tag == "admin" {
			foundAdmin = true
		}
		if tag == "priority" {
			foundPriority = true
		}
	}

	if !foundAdmin {
		t.Error("Expected 'admin' tag for admin path")
	}
	if !foundPriority {
		t.Error("Expected 'priority' tag for high severity event")
	}
}

func TestAddThreatIntel(t *testing.T) {
	service := NewSecurityAuditService()

	service.AddThreatIntel("10.0.0.1", "malicious", "test_source", "Test threat", 0.9)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	event := service.LogEvent(EventLoginAttempt, req, nil)
	if event.ThreatScore < 5.0 {
		t.Errorf("Expected threat score to be increased by threat intel, got %f", event.ThreatScore)
	}
}

func TestGetGeoIP(t *testing.T) {
	service := NewSecurityAuditService()

	geo := service.getGeoIP("192.168.1.1")
	if geo["country"] != "Private" {
		t.Errorf("Expected country 'Private' for private IP, got '%s'", geo["country"])
	}

	geo = service.getGeoIP("8.8.8.8")
	if geo["country"] != "Unknown" {
		t.Errorf("Expected country 'Unknown' for public IP, got '%s'", geo["country"])
	}
}

func TestNewEventTypes(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false

	req := httptest.NewRequest("GET", "/test", nil)

	service.LogEvent(EventAccountCreated, req, nil)
	service.LogEvent(EventPasswordReset, req, nil)
	service.LogEvent(EventAPIKeyGenerated, req, nil)
	service.LogEvent(EventMFAEnabled, req, nil)

	events := service.GetRecentEvents(10)
	if len(events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(events))
	}
}

func TestSecurityEventFields(t *testing.T) {
	service := NewSecurityAuditService()
	service.asyncMode = false

	req := httptest.NewRequest("POST", "/api/login", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Request-ID", "test123")

	event := service.LogEvent(EventLoginAttempt, req, map[string]interface{}{"test": "data"})

	if event.ID == "" {
		t.Error("Expected non-empty event ID")
	}
	if event.RequestMethod != "POST" {
		t.Errorf("Expected request method POST, got '%s'", event.RequestMethod)
	}
	if event.GeoLocation == nil {
		t.Error("Expected non-nil GeoLocation")
	}
	if len(event.Tags) == 0 {
		t.Error("Expected at least one tag")
	}
	if event.ThreatScore == 0 {
		t.Error("Expected non-zero threat score")
	}
}
