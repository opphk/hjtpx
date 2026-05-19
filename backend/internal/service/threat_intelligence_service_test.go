package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestThreatIntelligenceService_AnalyzeIP(t *testing.T) {
	service := NewThreatIntelligenceService()
	ctx := context.Background()

	tests := []struct {
		name     string
		ip       string
		wantErr  bool
	}{
		{
			name:    "Valid public IP",
			ip:      "8.8.8.8",
			wantErr: false,
		},
		{
			name:    "Private IP should return zero score",
			ip:      "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "Loopback IP",
			ip:      "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "Another public IP",
			ip:      "1.1.1.1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.AnalyzeIP(ctx, tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != nil {
				if result.IP != tt.ip {
					t.Errorf("AnalyzeIP() IP = %v, want %v", result.IP, tt.ip)
				}
			}
		})
	}
}

func TestThreatIntelligenceService_DetectThreatPattern(t *testing.T) {
	service := NewThreatIntelligenceService()
	ctx := context.Background()

	tests := []struct {
		name         string
		requestURI   string
		queryString  string
		wantPatterns int
	}{
		{
			name:         "SQL Injection attempt",
			requestURI:   "/api/users",
			queryString:  "id=1 UNION SELECT password FROM users--",
			wantPatterns: 1,
		},
		{
			name:         "XSS attempt",
			requestURI:   "/search",
			queryString:  "q=<script>alert('xss')</script>",
			wantPatterns: 1,
		},
		{
			name:         "Normal request",
			requestURI:   "/api/users",
			queryString:  "id=1",
			wantPatterns: 0,
		},
		{
			name:         "Path traversal attempt",
			requestURI:   "/files/../../../etc/passwd",
			queryString:  "",
			wantPatterns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.requestURI+"?"+tt.queryString, nil)
			patterns, err := service.DetectThreatPattern(ctx, req)
			if err != nil {
				t.Errorf("DetectThreatPattern() error = %v", err)
				return
			}
			if len(patterns) != tt.wantPatterns {
				t.Errorf("DetectThreatPattern() got %v patterns, want %v", len(patterns), tt.wantPatterns)
			}
		})
	}
}

func TestThreatIntelligenceService_MatchAttackSignature(t *testing.T) {
	service := NewThreatIntelligenceService()
	ctx := context.Background()

	tests := []struct {
		name          string
		payload       string
		wantSignature bool
	}{
		{
			name:          "SQL Union signature",
			payload:       "1 UNION SELECT username, password FROM admin",
			wantSignature: true,
		},
		{
			name:          "Normal payload",
			payload:       "hello world",
			wantSignature: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/test", strings.NewReader(tt.payload))
			signatures, err := service.MatchAttackSignature(ctx, req, tt.payload)
			if err != nil {
				t.Errorf("MatchAttackSignature() error = %v", err)
				return
			}
			if (len(signatures) > 0) != tt.wantSignature {
				t.Errorf("MatchAttackSignature() got signature = %v, want %v", len(signatures) > 0, tt.wantSignature)
			}
		})
	}
}

func TestThreatIntelligenceService_GetComprehensiveThreatAssessment(t *testing.T) {
	service := NewThreatIntelligenceService()
	ctx := context.Background()

	tests := []struct {
		name      string
		ip        string
		domain    string
		url       string
		wantScore bool
	}{
		{
			name:      "Assessment with IP only",
			ip:        "8.8.8.8",
			domain:    "",
			url:       "",
			wantScore: true,
		},
		{
			name:      "Assessment with all parameters",
			ip:        "1.1.1.1",
			domain:    "example.com",
			url:       "https://example.com/test",
			wantScore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetComprehensiveThreatAssessment(ctx, tt.ip, tt.domain, tt.url)
			if err != nil {
				t.Errorf("GetComprehensiveThreatAssessment() error = %v", err)
				return
			}
			if tt.wantScore && result.CombinedScore < 0 {
				t.Errorf("GetComprehensiveThreatAssessment() got invalid score = %v", result.CombinedScore)
			}
		})
	}
}

func TestThreatIntelligenceService_isPrivateIP(t *testing.T) {
	service := NewThreatIntelligenceService()

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"10.x.x.x range", "10.0.0.1", true},
		{"172.16.x.x range", "172.16.0.1", true},
		{"192.168.x.x range", "192.168.1.1", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"Public IP", "8.8.8.8", false},
		{"Another public IP", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.isPrivateIP(tt.ip); got != tt.want {
				t.Errorf("isPrivateIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestThreatIntelligenceService_AddCustomThreatPattern(t *testing.T) {
	service := NewThreatIntelligenceService()

	pattern := &ThreatPattern{
		ID:         "custom_test",
		Name:       "Custom Test Pattern",
		Type:       ThreatTypeWebAttack,
		Severity:   SeverityLevelHigh,
		Confidence: 0.95,
	}

	err := service.AddCustomThreatPattern("custom_pattern", pattern)
	if err != nil {
		t.Errorf("AddCustomThreatPattern() error = %v", err)
	}

	if len(service.threatPatterns) == 0 {
		t.Error("AddCustomThreatPattern() did not add pattern")
	}
}

func TestThreatIntelligenceService_GetThreatStatistics(t *testing.T) {
	service := NewThreatIntelligenceService()

	stats := service.GetThreatStatistics()

	if stats == nil {
		t.Error("GetThreatStatistics() returned nil")
	}

	if stats["total_feeds"] == nil {
		t.Error("GetThreatStatistics() missing total_feeds")
	}

	if stats["total_threat_patterns"] == nil {
		t.Error("GetThreatStatistics() missing total_threat_patterns")
	}
}

func TestThreatIntelligenceService_ValidateIOC(t *testing.T) {
	service := NewThreatIntelligenceService()

	tests := []struct {
		name    string
		iocType string
		iocValue string
		want    bool
	}{
		{"Valid IP", "ip", "192.0.2.1", true},
		{"Invalid IP", "ip", "invalid", false},
		{"Valid domain", "domain", "example.com", true},
		{"Valid URL", "url", "https://example.com", true},
		{"Invalid URL", "url", "not-a-url", false},
		{"Valid MD5 hash", "hash", "d41d8cd98f00b204e9800998ecf8427e", true},
		{"Invalid hash length", "hash", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := service.ValidateIOC(tt.iocType, tt.iocValue)
			if err != nil {
				t.Errorf("ValidateIOC() error = %v", err)
				return
			}
			if valid != tt.want {
				t.Errorf("ValidateIOC() = %v, want %v", valid, tt.want)
			}
		})
	}
}

func TestThreatIntelligenceService_ExportImport(t *testing.T) {
	service := NewThreatIntelligenceService()

	data, err := service.ExportThreatData("json")
	if err != nil {
		t.Errorf("ExportThreatData() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportThreatData() returned empty data")
	}

	newService := NewThreatIntelligenceService()
	err = newService.ImportThreatData(data, "json")
	if err != nil {
		t.Errorf("ImportThreatData() error = %v", err)
	}
}

func TestThreatIntelligenceService_UpdateThreatFeeds(t *testing.T) {
	service := NewThreatIntelligenceService()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.UpdateThreatFeeds(ctx)
	if err != nil {
		t.Errorf("UpdateThreatFeeds() error = %v", err)
	}
}
