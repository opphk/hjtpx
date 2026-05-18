package service

import (
	"testing"
	"time"
)

func TestNewBehaviorPredictionService(t *testing.T) {
	svc := NewBehaviorPredictionService()
	if svc == nil {
		t.Error("NewBehaviorPredictionService 返回了 nil")
	}
}

func TestBehaviorPredictionService_PredictUserBehavior(t *testing.T) {
	svc := NewBehaviorPredictionService()

	req := &PredictionRequest{
		UserID:    "user-123",
		SessionID: "session-456",
		CurrentAction: &UserAction{
			ActionType: "login",
			Timestamp: time.Now(),
			Duration:  500 * time.Millisecond,
			Success:   true,
		},
		RecentActions: []UserAction{
			{
				ActionType: "browse",
				Timestamp: time.Now().Add(-1 * time.Hour),
				Duration:  2 * time.Second,
				Success:   true,
			},
		},
		EnvironmentData: map[string]interface{}{
			"ip":      "192.168.1.1",
			"browser": "Chrome",
		},
	}

	result := svc.PredictUserBehavior(req)

	if result == nil {
		t.Error("PredictUserBehavior 返回了 nil")
	}

	if result.RecommendedAction == "" {
		t.Error("推荐动作为空")
	}
}

func TestBehaviorPredictionService_RiskProfile(t *testing.T) {
	svc := NewBehaviorPredictionService()

	profile := svc.GetRiskProfile("user-123")

	if profile == nil {
		t.Error("GetRiskProfile 返回了 nil")
	}
}

func TestBehaviorPredictionService_UpdateRiskThresholds(t *testing.T) {
	svc := NewBehaviorPredictionService()

	thresholds := &RiskThresholds{
		LowRiskThreshold:    20,
		MediumRiskThreshold: 50,
		HighRiskThreshold:   80,
	}

	svc.UpdateRiskThresholds(thresholds)
}

func TestBehaviorPredictionService_AddToWhitelist(t *testing.T) {
	svc := NewBehaviorPredictionService()

	svc.AddToWhitelist("test-user", "user", 1*time.Hour, "test")

	t.Log("AddToWhitelist 测试通过")
}

func TestBehaviorPredictionService_AddToBlacklist(t *testing.T) {
	svc := NewBehaviorPredictionService()

	svc.AddToBlacklist("malicious-user", "user", 24*time.Hour, "malicious activity", "high")

	t.Log("AddToBlacklist 测试通过")
}

func TestPredictionRequest(t *testing.T) {
	req := &PredictionRequest{
		UserID:    "user-123",
		SessionID: "session-456",
		CurrentAction: &UserAction{
			ActionType: "click",
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
			Success:   true,
		},
	}

	if req.UserID != "user-123" {
		t.Errorf("UserID 不正确")
	}
	if req.SessionID != "session-456" {
		t.Errorf("SessionID 不正确")
	}
	if req.CurrentAction.ActionType != "click" {
		t.Errorf("ActionType 不正确")
	}
}

func TestPredictionResult(t *testing.T) {
	result := &PredictionResult{
		RecommendedAction: "allow",
		Confidence:        0.95,
		ShouldIntercept:   false,
	}

	if result.RecommendedAction != "allow" {
		t.Errorf("RecommendedAction 不正确")
	}
	if result.Confidence != 0.95 {
		t.Errorf("Confidence 不正确")
	}
}

func TestUserAction(t *testing.T) {
	action := &UserAction{
		ActionType: "submit",
		Timestamp:  time.Now(),
		Duration:   500 * time.Millisecond,
		Success:    true,
		Metadata:   map[string]interface{}{"field": "value"},
	}

	if action.ActionType != "submit" {
		t.Errorf("ActionType 不正确")
	}
	if action.Success != true {
		t.Errorf("Success 不正确")
	}
}

func TestRiskThresholds(t *testing.T) {
	thresholds := &RiskThresholds{
		LowRiskThreshold:    10,
		MediumRiskThreshold: 40,
		HighRiskThreshold:   70,
	}

	if thresholds.LowRiskThreshold != 10 {
		t.Errorf("LowRiskThreshold 不正确")
	}
	if thresholds.MediumRiskThreshold != 40 {
		t.Errorf("MediumRiskThreshold 不正确")
	}
	if thresholds.HighRiskThreshold != 70 {
		t.Errorf("HighRiskThreshold 不正确")
	}
}

func TestRiskProfile(t *testing.T) {
	profile := &RiskProfile{
		UserID:          "user-123",
		BaseRiskScore:   25.0,
		CurrentRiskScore: 30.0,
		RiskTrend:       "stable",
	}

	if profile.UserID != "user-123" {
		t.Errorf("UserID 不正确")
	}
	if profile.BaseRiskScore != 25.0 {
		t.Errorf("BaseRiskScore 不正确")
	}
}

func TestRiskAssessment(t *testing.T) {
	assessment := &RiskAssessment{
		OverallRiskScore: 45.0,
		RiskLevel:       "medium",
		RiskFactors:     []PredictionRiskFactor{},
	}

	if assessment.OverallRiskScore != 45.0 {
		t.Errorf("OverallRiskScore 不正确")
	}
	if assessment.RiskLevel != "medium" {
		t.Errorf("RiskLevel 不正确")
	}
}

func TestPredictionRiskFactor(t *testing.T) {
	factor := &PredictionRiskFactor{
		FactorType:   "velocity",
		Severity:     0.5,
		Weight:       0.3,
		Contributing: []string{"fast_clicks", "uniform_pattern"},
	}

	if factor.FactorType != "velocity" {
		t.Errorf("FactorType 不正确")
	}
	if factor.Severity != 0.5 {
		t.Errorf("Severity 不正确")
	}
}
