package service

import (
	"testing"
	"time"
)

func TestNewBehaviorPredictionService(t *testing.T) {
	service := NewBehaviorPredictionService()
	if service == nil {
		t.Error("NewBehaviorPredictionService 返回了 nil")
	}
}

func TestPredictUserBehavior_Normal(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	req := &PredictionRequest{
		UserID: "normal_user",
		RecentActions: []UserAction{
			{ActionType: "login", Timestamp: time.Now()},
			{ActionType: "view", Timestamp: time.Now()},
		},
	}
	
	prediction := service.PredictUserBehavior(req)
	
	if prediction == nil {
		t.Error("PredictUserBehavior 返回了 nil")
	}
}

func TestGetRiskProfile(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	profile := service.GetRiskProfile("user-123")
	
	if profile == nil {
		t.Error("GetRiskProfile 返回了 nil")
	}
}

func TestAddToWhitelist(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	service.AddToWhitelist("192.168.1.100", "ip", 1*time.Hour, "测试白名单")
}

func TestAddToBlacklist(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	service.AddToBlacklist("192.168.1.200", "ip", 24*time.Hour, "测试黑名单", "high")
}

func TestBehaviorPredictionResult_Fields(t *testing.T) {
	result := &BehaviorPredictionResult{
		Action:      "allow",
		Confidence:  0.95,
		RiskScore:   10.0,
		Timestamp:   time.Now(),
		Reasons:     []string{"normal_pattern"},
	}
	
	if result.Action != "allow" {
		t.Errorf("期望动作为 allow, 实际得到 %s", result.Action)
	}
	if result.Confidence != 0.95 {
		t.Errorf("期望置信度为 0.95, 实际得到 %f", result.Confidence)
	}
}

func TestSessionAnalysisResult_Fields(t *testing.T) {
	result := &SessionAnalysisResult{
		SessionID: "session-123",
		Score:     75.0,
		Duration:  300,
		Patterns:  []string{"normal"},
		Anomalies: []string{},
	}
	
	if result.SessionID != "session-123" {
		t.Errorf("期望会话ID为 session-123, 实际得到 %s", result.SessionID)
	}
	if result.Score != 75.0 {
		t.Errorf("期望分数为 75.0, 实际得到 %f", result.Score)
	}
}

func TestAnomalyDetectionResult_Fields(t *testing.T) {
	result := &AnomalyDetectionResult{
		UserID:     "user-123",
		IsAnomaly:  true,
		AnomalyType: "suspicious_pattern",
		Severity:   "high",
		Score:      85.0,
		DetectedAt: time.Now(),
	}
	
	if result.UserID != "user-123" {
		t.Errorf("期望用户ID为 user-123, 实际得到 %s", result.UserID)
	}
	if !result.IsAnomaly {
		t.Error("应该是异常")
	}
	if result.AnomalyType != "suspicious_pattern" {
		t.Errorf("期望异常类型为 suspicious_pattern, 实际得到 %s", result.AnomalyType)
	}
}

func TestPredictBehavior_Concurrent(t *testing.T) {
	service := NewBehaviorPredictionService()
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(userID string) {
			prediction := service.PredictBehavior(userID)
			if prediction == nil {
				t.Errorf("并发预测失败")
			}
			done <- true
		}("user-" + string(rune('0'+i)))
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPredictBehavior_EmptyUserID(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	prediction := service.PredictBehavior("")
	
	if prediction == nil {
		t.Error("PredictBehavior 应该处理空用户ID")
	}
}
