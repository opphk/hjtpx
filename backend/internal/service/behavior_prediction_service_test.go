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

func TestPredictBehavior_Normal(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	prediction := service.PredictBehavior("normal_user")
	
	if prediction == nil {
		t.Error("PredictBehavior 返回了 nil")
	}
	if prediction.Action != "normal" {
		t.Errorf("期望动作为 normal, 实际得到 %s", prediction.Action)
	}
}

func TestPredictBehavior_Suspicious(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	prediction := service.PredictBehavior("suspicious_user")
	
	if prediction == nil {
		t.Error("PredictBehavior 返回了 nil")
	}
	if prediction.Action == "" {
		t.Error("动作为空")
	}
}

func TestAnalyzeSession_Success(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	analysis := service.AnalyzeSession(map[string]interface{}{
		"user_id":    "user-123",
		"session_id": "session-456",
		"duration":   300,
	})
	
	if analysis == nil {
		t.Error("AnalyzeSession 返回了 nil")
	}
	if analysis.Score < 0 || analysis.Score > 100 {
		t.Errorf("分数应该在 0-100 之间, 实际得到 %f", analysis.Score)
	}
}

func TestAnalyzeSession_Empty(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	analysis := service.AnalyzeSession(map[string]interface{}{})
	
	if analysis == nil {
		t.Error("AnalyzeSession 返回了 nil")
	}
}

func TestDetectAnomaly_NoAnomaly(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	anomaly := service.DetectAnomaly("normal_user")
	
	if anomaly == nil {
		t.Error("DetectAnomaly 返回了 nil")
	}
}

func TestDetectAnomaly_WithAnomaly(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	anomaly := service.DetectAnomaly("suspicious_user")
	
	if anomaly == nil {
		t.Error("DetectAnomaly 返回了 nil")
	}
}

func TestGetUserBehaviorProfile(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	profile := service.GetUserBehaviorProfile("user-123")
	
	if profile == nil {
		t.Error("GetUserBehaviorProfile 返回了 nil")
	}
	if profile.UserID != "user-123" {
		t.Errorf("期望用户ID为 user-123, 实际得到 %s", profile.UserID)
	}
}

func TestGetUserBehaviorProfile_NonExistent(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	profile := service.GetUserBehaviorProfile("non-existent-user")
	
	if profile == nil {
		t.Error("GetUserBehaviorProfile 应该返回默认profile")
	}
}

func TestUpdateUserBehaviorProfile(t *testing.T) {
	service := NewBehaviorPredictionService()
	
	profile := &BehaviorProfile{
		UserID:       "user-123",
		TotalActions: 100,
		RiskScore:    30.0,
		LastUpdated:  time.Now(),
	}
	
	err := service.UpdateUserBehaviorProfile(profile)
	if err != nil {
		t.Errorf("UpdateUserBehaviorProfile 失败: %v", err)
	}
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
