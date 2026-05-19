package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIAttackDetectorService_DetectAttack(t *testing.T) {
	service := NewAIAttackDetectorService()
	ctx := context.Background()

	tests := []struct {
		name          string
		path          string
		query         string
		wantAttack    bool
	}{
		{
			name:       "Normal request",
			path:       "/api/users",
			query:      "",
			wantAttack: false,
		},
		{
			name:       "SQL Injection",
			path:       "/api/search",
			query:      "id=1 UNION SELECT password FROM admin--",
			wantAttack: true,
		},
		{
			name:       "XSS attempt",
			path:       "/search",
			query:      "q=<script>alert('xss')</script>",
			wantAttack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.path+"?"+tt.query, nil)
			result, err := service.DetectAttack(ctx, req, "test-session")
			if err != nil {
				t.Errorf("DetectAttack() error = %v", err)
				return
			}

			if result.IsAttack != tt.wantAttack {
				t.Errorf("DetectAttack() IsAttack = %v, want %v", result.IsAttack, tt.wantAttack)
			}
		})
	}
}

func TestAIAttackDetectorService_EvaluateRules(t *testing.T) {
	service := NewAIAttackDetectorService()

	tests := []struct {
		name          string
		path          string
		query         string
		wantMatches   int
	}{
		{
			name:        "SQL Union match",
			path:        "/api",
			query:       "id=1 UNION SELECT * FROM users",
			wantMatches: 1,
		},
		{
			name:        "XSS script tag",
			path:        "/search",
			query:       "q=<script>alert(1)</script>",
			wantMatches: 1,
		},
		{
			name:        "No match",
			path:        "/api/users",
			query:       "id=1&name=test",
			wantMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.path+"?"+tt.query, nil)
			results := service.evaluateRules(req)

			if len(results) != tt.wantMatches {
				t.Errorf("evaluateRules() got %d matches, want %d", len(results), tt.wantMatches)
			}
		})
	}
}

func TestAIAttackDetectorService_BehavioralAnalysis(t *testing.T) {
	service := NewAIAttackDetectorService()

	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	result := service.performBehavioralAnalysis("192.0.2.1", "test-session", req)

	if result == nil {
		t.Error("performBehavioralAnalysis() returned nil")
	}
}

func TestAIAttackDetectorService_GetDetectionRules(t *testing.T) {
	service := NewAIAttackDetectorService()

	rules := service.GetDetectionRules()
	if len(rules) == 0 {
		t.Error("GetDetectionRules() returned empty list")
	}
}

func TestAIAttackDetectorService_AddDetectionRule(t *testing.T) {
	service := NewAIAttackDetectorService()

	initialCount := len(service.detectionRules)

	rule := &DetectionRule{
		Name:       "Test Rule",
		Pattern:    regexp.MustCompile(`(?i)test_pattern`),
		AttackType: AttackCategorySQLInjection,
		Severity:   3,
		Weight:     0.7,
	}

	err := service.AddDetectionRule(rule)
	if err != nil {
		t.Errorf("AddDetectionRule() error = %v", err)
	}

	rules := service.GetDetectionRules()
	if len(rules) != initialCount+1 {
		t.Errorf("GetDetectionRules() count = %d, want %d", len(rules), initialCount+1)
	}
}

func TestAIAttackDetectorService_MLPrediction(t *testing.T) {
	service := NewAIAttackDetectorService()

	req := httptest.NewRequest("GET", "http://example.com/api/test?q=normal", nil)
	seqResult := &SequenceAnalysisResult{IsAnomalous: false, Score: 0.1}
	behResult := &BehavioralAnalysisResult{IsAnomalous: false, AnomalyScore: 0.1}

	result := service.mlPredict(req, seqResult, behResult)

	if result == nil {
		t.Error("mlPredict() returned nil")
	}

	if result.AttackProbability < 0 || result.AttackProbability > 1 {
		t.Errorf("mlPredict() AttackProbability = %v, want between 0 and 1", result.AttackProbability)
	}
}

func TestAIAttackDetectorService_GetAttackStatistics(t *testing.T) {
	service := NewAIAttackDetectorService()

	stats := service.GetAttackStatistics()

	if stats == nil {
		t.Error("GetAttackStatistics() returned nil")
	}

	if stats["total_models"] == nil {
		t.Error("GetAttackStatistics() missing total_models")
	}

	if stats["total_rules"] == nil {
		t.Error("GetAttackStatistics() missing total_rules")
	}
}

func TestAIAttackDetectorService_UpdateDetectionRule(t *testing.T) {
	service := NewAIAttackDetectorService()

	updates := &DetectionRule{
		Severity: 5,
		Weight:   0.9,
		IsActive: true,
	}

	for id := range service.detectionRules {
		err := service.UpdateDetectionRule(id, updates)
		if err != nil {
			t.Errorf("UpdateDetectionRule() error = %v", err)
		}
		break
	}
}

func TestAIAttackDetectorService_GetActiveThreats(t *testing.T) {
	service := NewAIAttackDetectorService()

	threats := service.GetActiveThreats()
	if threats == nil {
		t.Error("GetActiveThreats() returned nil")
	}
}

func TestAIAttackDetectorService_ExportImport(t *testing.T) {
	service := NewAIAttackDetectorService()

	data, err := service.ExportModelConfig()
	if err != nil {
		t.Errorf("ExportModelConfig() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportModelConfig() returned empty data")
	}

	newService := NewAIAttackDetectorService()
	err = newService.ImportModelConfig(data)
	if err != nil {
		t.Errorf("ImportModelConfig() error = %v", err)
	}
}
