package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdaptiveSecurityMiddleware_Disabled(t *testing.T) {
	cfg := MiddlewareConfig{
		Enabled: false,
	}

	handler := AdaptiveSecurityMiddleware(cfg)

	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code == http.StatusForbidden {
		t.Error("Disabled middleware should not block requests")
	}
}

func TestAdaptiveSecurityMiddleware_ExcludePaths(t *testing.T) {
	cfg := MiddlewareConfig{
		Enabled:     true,
		ExcludePaths: []string{"/api/v1/health"},
	}

	handler := AdaptiveSecurityMiddleware(cfg)

	req := httptest.NewRequest("GET", "http://example.com/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code == http.StatusForbidden {
		t.Error("Excluded path should not be blocked")
	}
}

func TestAdaptiveSecurityMiddleware_NormalRequest(t *testing.T) {
	cfg := MiddlewareConfig{
		Enabled:           true,
		ExcludePaths:      []string{},
		BlockOnAnyThreat:  false,
		ChallengeOnMedium: false,
	}

	handler := AdaptiveSecurityMiddleware(cfg)

	req := httptest.NewRequest("GET", "http://example.com/api/users", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code == http.StatusForbidden {
		t.Error("Normal request should not be blocked")
	}
}

func TestAdaptiveSecurityMiddleware_HealthCheck(t *testing.T) {
	cfg := DefaultAdaptiveSecurityConfig

	handler := AdaptiveSecurityMiddleware(cfg)

	req := httptest.NewRequest("GET", "http://example.com/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code == http.StatusForbidden {
		t.Error("Health check should be excluded by default")
	}
}

func TestGetAdaptiveSecurityService(t *testing.T) {
	service := GetAdaptiveSecurityService()

	if service == nil {
		t.Error("GetAdaptiveSecurityService() returned nil")
	}
}

func TestAdaptiveSecurityService_EnableDisable(t *testing.T) {
	service := GetAdaptiveSecurityService()

	service.Disable()
	if service.IsEnabled() {
		t.Error("Disable() did not disable service")
	}

	service.Enable()
	if !service.IsEnabled() {
		t.Error("Enable() did not enable service")
	}
}

func TestAdaptiveSecurityService_GetSecurityStatistics(t *testing.T) {
	service := GetAdaptiveSecurityService()

	stats := service.GetSecurityStatistics()

	if stats == nil {
		t.Error("GetSecurityStatistics() returned nil")
	}

	if stats["enabled"] == nil {
		t.Error("GetSecurityStatistics() missing enabled field")
	}

	if stats["config"] == nil {
		t.Error("GetSecurityStatistics() missing config field")
	}
}

func TestAdaptiveSecurityService_UpdateConfig(t *testing.T) {
	service := GetAdaptiveSecurityService()

	config := &AdaptiveSecurityConfig{
		ThreatIntelEnabled:    false,
		DynamicDefenseEnabled: true,
		IntegrationMode:      IntegrationModeParallel,
	}

	service.UpdateConfig(config)

	updatedConfig := service.GetConfig()
	if updatedConfig.ThreatIntelEnabled != false {
		t.Error("UpdateConfig() did not update ThreatIntelEnabled")
	}

	if updatedConfig.IntegrationMode != IntegrationModeParallel {
		t.Error("UpdateConfig() did not update IntegrationMode")
	}
}

func TestAdaptiveSecurityService_GetConfig(t *testing.T) {
	service := GetAdaptiveSecurityService()

	config := service.GetConfig()

	if config == nil {
		t.Error("GetConfig() returned nil")
	}
}

func TestAdaptiveSecurityService_GetIndividualServices(t *testing.T) {
	service := GetAdaptiveSecurityService()

	if service.GetThreatIntel() == nil {
		t.Error("GetThreatIntel() returned nil")
	}

	if service.GetDynamicDefense() == nil {
		t.Error("GetDynamicDefense() returned nil")
	}

	if service.GetAIAttackDetector() == nil {
		t.Error("GetAIAttackDetector() returned nil")
	}

	if service.GetHoneypot() == nil {
		t.Error("GetHoneypot() returned nil")
	}

	if service.GetAutoResponse() == nil {
		t.Error("GetAutoResponse() returned nil")
	}

	if service.GetBotDetection() == nil {
		t.Error("GetBotDetection() returned nil")
	}
}

func TestAdaptiveSecurityService_PerformHealthCheck(t *testing.T) {
	service := GetAdaptiveSecurityService()

	result := service.PerformHealthCheck()

	if result == nil {
		t.Error("PerformHealthCheck() returned nil")
	}

	if result.Healthy == false {
		t.Error("PerformHealthCheck() should return healthy status")
	}

	if len(result.Components) == 0 {
		t.Error("PerformHealthCheck() returned no components")
	}
}

func TestAdaptiveSecurityService_ExportImport(t *testing.T) {
	service := GetAdaptiveSecurityService()

	data, err := service.ExportConfiguration()
	if err != nil {
		t.Errorf("ExportConfiguration() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportConfiguration() returned empty data")
	}

	err = service.ImportConfiguration(data)
	if err != nil {
		t.Errorf("ImportConfiguration() error = %v", err)
	}
}

func TestProcessRequest(t *testing.T) {
	service := GetAdaptiveSecurityService()

	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"

	result := service.ProcessRequest(req)

	if result == nil {
		t.Error("ProcessRequest() returned nil")
	}

	if result.ProcessingTime < 0 {
		t.Error("ProcessRequest() returned negative processing time")
	}
}

func TestProcessSequential(t *testing.T) {
	service := GetAdaptiveSecurityService()
	ctx := context.Background()

	result := &AdaptiveSecurityResult{
		ThreatTypes:     []string{},
		Recommendations: []string{},
		ActionTaken:     []string{},
		ComponentsUsed:  []string{},
	}

	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	sessionID := "test-session"

	service.processSequential(ctx, req, sessionID, result)

	if len(result.ComponentsUsed) == 0 {
		t.Error("processSequential() did not use any components")
	}
}

func TestCalculateFinalScore(t *testing.T) {
	service := GetAdaptiveSecurityService()

	result := &AdaptiveSecurityResult{
		ThreatIntelScore:     30,
		DynamicDefenseScore:   40,
		AIAttackScore:        50,
		HoneypotTriggered:    false,
		ThreatTypes:         []string{},
		Recommendations:      []string{},
		ActionTaken:          []string{},
		ComponentsUsed:       []string{},
	}

	service.calculateFinalScore(result)

	if result.RiskScore < 0 {
		t.Error("calculateFinalScore() returned negative score")
	}
}

func TestCalculateFinalScore_HoneypotTriggered(t *testing.T) {
	service := GetAdaptiveSecurityService()

	result := &AdaptiveSecurityResult{
		ThreatIntelScore:     0,
		DynamicDefenseScore:   0,
		AIAttackScore:        0,
		HoneypotTriggered:    true,
		ThreatTypes:         []string{},
		Recommendations:      []string{},
		ActionTaken:          []string{},
		ComponentsUsed:       []string{},
	}

	service.calculateFinalScore(result)

	if result.RiskScore != 100 {
		t.Errorf("calculateFinalScore() should set score to 100 for honeypot trigger, got %v", result.RiskScore)
	}
}
