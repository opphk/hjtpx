package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDynamicDefenseService_EvaluateRequest(t *testing.T) {
	service := NewDynamicDefenseService()
	ctx := context.Background()

	tests := []struct {
		name           string
		path           string
		expectedBlock  bool
		expectedThreat bool
	}{
		{
			name:           "Normal request",
			path:           "/api/users",
			expectedBlock:  false,
			expectedThreat: false,
		},
		{
			name:           "SQL Injection attempt",
			path:           "/api/search?q=1' OR '1'='1",
			expectedBlock:  true,
			expectedThreat: true,
		},
		{
			name:           "XSS attempt",
			path:           "/api/search?q=<script>alert(1)</script>",
			expectedBlock:  true,
			expectedThreat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.path, nil)
			req.RemoteAddr = "192.0.2.1:12345"

			result, err := service.EvaluateRequest(ctx, req)
			if err != nil {
				t.Errorf("EvaluateRequest() error = %v", err)
				return
			}

			if result.ShouldBlock != tt.expectedBlock {
				t.Errorf("EvaluateRequest() ShouldBlock = %v, want %v", result.ShouldBlock, tt.expectedBlock)
			}
		})
	}
}

func TestDynamicDefenseService_WAFRules(t *testing.T) {
	service := NewDynamicDefenseService()

	tests := []struct {
		name      string
		path      string
		shouldMatch bool
	}{
		{
			name:        "SQL Union pattern",
			path:        "/api?id=1 UNION SELECT * FROM users",
			shouldMatch: true,
		},
		{
			name:        "SQL OR pattern",
			path:        "/api?id=1 OR 1=1",
			shouldMatch: true,
		},
		{
			name:        "Script tag",
			path:        "/search?q=<script>alert(1)</script>",
			shouldMatch: true,
		},
		{
			name:        "Path traversal",
			path:        "/files/../../../etc/passwd",
			shouldMatch: true,
		},
		{
			name:        "Normal path",
			path:        "/api/users",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.path, nil)
			result := service.evaluateWAFRules(req)

			if result.Matched != tt.shouldMatch {
				t.Errorf("evaluateWAFRules() Matched = %v, want %v", result.Matched, tt.shouldMatch)
			}
		})
	}
}

func TestDynamicDefenseService_RateLimiting(t *testing.T) {
	service := NewDynamicDefenseService()

	ip := "192.0.2.100"

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
		req.RemoteAddr = ip + ":12345"

		result := service.evaluateRateLimit(ip)
		if result.ShouldLimit {
			t.Errorf("evaluateRateLimit() ShouldLimit = true on request %d, want false", i+1)
		}
	}

	for i := 0; i < 100; i++ {
		service.evaluateRateLimit(ip)
	}

	result := service.evaluateRateLimit(ip)
	if !result.ShouldLimit {
		t.Error("evaluateRateLimit() ShouldLimit = false after many requests, want true")
	}
}

func TestDynamicDefenseService_IPRangeWhitelist(t *testing.T) {
	service := NewDynamicDefenseService()

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"10.0.0.1 in whitelist", "10.0.0.1", true},
		{"192.168.1.1 in whitelist", "192.168.1.1", true},
		{"172.16.0.1 in whitelist", "172.16.0.1", true},
		{"Public IP not in whitelist", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.checkIPRangeWhitelist(tt.ip); got != tt.want {
				t.Errorf("checkIPRangeWhitelist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDynamicDefenseService_ThreatLevelCalculation(t *testing.T) {
	service := NewDynamicDefenseService()

	tests := []struct {
		name          string
		score         float64
		expectedLevel ThreatLevel
	}{
		{"Critical threat", 85, ThreatLevelCritical},
		{"High threat", 65, ThreatLevelHigh},
		{"Medium threat", 45, ThreatLevelMedium},
		{"Low threat", 25, ThreatLevelLow},
		{"No threat", 10, ThreatLevelNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.calculateThreatLevel(tt.score); got != tt.expectedLevel {
				t.Errorf("calculateThreatLevel() = %v, want %v", got, tt.expectedLevel)
			}
		})
	}
}

func TestDynamicDefenseService_PolicyManagement(t *testing.T) {
	service := NewDynamicDefenseService()

	policy := &DefensePolicy{
		Name:      "Test Policy",
		Priority:  50,
		IsActive:  true,
		Condition: &PolicyCondition{ThreatLevel: ThreatLevelMedium},
		Action:    ActionRateLimit,
		Duration:  30 * time.Minute,
	}

	err := service.CreatePolicy(policy)
	if err != nil {
		t.Errorf("CreatePolicy() error = %v", err)
	}

	policies := service.GetPolicies()
	if len(policies) == 0 {
		t.Error("GetPolicies() returned empty list")
	}

	policy.IsActive = false
	err = service.UpdatePolicy(policy.ID, policy)
	if err != nil {
		t.Errorf("UpdatePolicy() error = %v", err)
	}

	err = service.DeletePolicy(policy.ID)
	if err != nil {
		t.Errorf("DeletePolicy() error = %v", err)
	}
}

func TestDynamicDefenseService_WAFRuleManagement(t *testing.T) {
	service := NewDynamicDefenseService()

	initialCount := len(service.wafRules)

	rule := &WAFRule{
		Name:        "Test WAF Rule",
		Pattern:     regexp.MustCompile(`(?i)test_pattern`),
		Action:     ActionBlock,
		Severity:   3,
		Description: "Test rule",
	}

	err := service.AddWAFRule(rule)
	if err != nil {
		t.Errorf("AddWAFRule() error = %v", err)
	}

	rules := service.GetWAFRules()
	if len(rules) != initialCount+1 {
		t.Errorf("GetWAFRules() count = %d, want %d", len(rules), initialCount+1)
	}
}

func TestDynamicDefenseService_GeoBlocking(t *testing.T) {
	service := NewDynamicDefenseService()

	service.EnableGeoBlocking([]string{"XX", "YY"})

	if !service.geoBlocking.BlockedCountries["XX"] {
		t.Error("EnableGeoBlocking() did not block country XX")
	}

	service.DisableGeoBlocking([]string{"XX"})

	if service.geoBlocking.BlockedCountries["XX"] {
		t.Error("DisableGeoBlocking() did not unblock country XX")
	}
}

func TestDynamicDefenseService_IPBlacklist(t *testing.T) {
	service := NewDynamicDefenseService()

	err := service.AddIPToBlacklist("192.0.2.0/24")
	if err != nil {
		t.Errorf("AddIPToBlacklist() error = %v", err)
	}

	if !service.checkIPRangeBlacklist("192.0.2.100") {
		t.Error("checkIPRangeBlacklist() returned false for blacklisted IP")
	}
}

func TestDynamicDefenseService_DefenseState(t *testing.T) {
	service := NewDynamicDefenseService()

	state := service.GetDefenseState()
	if state == nil {
		t.Error("GetDefenseState() returned nil")
	}

	if state.CurrentLevel != ThreatLevelLow {
		t.Errorf("GetDefenseState() CurrentLevel = %v, want %v", state.CurrentLevel, ThreatLevelLow)
	}
}

func TestDynamicDefenseService_DefenseStatistics(t *testing.T) {
	service := NewDynamicDefenseService()

	stats := service.GetDefenseStatistics()

	if stats == nil {
		t.Error("GetDefenseStatistics() returned nil")
	}

	if stats["total_policies"] == nil {
		t.Error("GetDefenseStatistics() missing total_policies")
	}

	if stats["waf_rules"] == nil {
		t.Error("GetDefenseStatistics() missing waf_rules")
	}
}

func TestDynamicDefenseService_EnableDisable(t *testing.T) {
	service := NewDynamicDefenseService()

	service.Disable()
	if service.IsEnabled() {
		t.Error("Disable() did not disable service")
	}

	service.Enable()
	if !service.IsEnabled() {
		t.Error("Enable() did not enable service")
	}
}

func TestDynamicDefenseService_ExportImport(t *testing.T) {
	service := NewDynamicDefenseService()

	config, err := service.ExportConfiguration()
	if err != nil {
		t.Errorf("ExportConfiguration() error = %v", err)
	}

	if len(config) == 0 {
		t.Error("ExportConfiguration() returned empty config")
	}

	newService := NewDynamicDefenseService()
	err = newService.ImportConfiguration(config)
	if err != nil {
		t.Errorf("ImportConfiguration() error = %v", err)
	}
}
