package service

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestSeamlessOptimizationService(t *testing.T) {
	t.Log("测试无缝优化服务...")

	service := NewSeamlessOptimizationService()

	behaviorData := []models.BehaviorData{
		{
			Data:      `{"x": 100, "y": 200, "timestamp": 1234567890, "event": "click"}`,
			DataType:  "click",
			Timestamp: time.Now(),
		},
	}

	environmentData := map[string]interface{}{
		"ip_address": "192.168.1.1",
		"user_agent": "Mozilla/5.0",
	}

	result, err := service.OptimizeSeamlessVerification(
		"user_123",
		"device_fingerprint_abc",
		behaviorData,
		environmentData,
		45.0,
	)

	if err != nil {
		t.Errorf("优化失败: %v", err)
		return
	}

	if result == nil {
		t.Error("结果为空")
		return
	}

	t.Logf("优化结果 - 最终风险分数: %.2f, 需要挑战: %v, 优化应用: %v",
		result.FinalRiskScore,
		result.ShouldChallenge,
		result.OptimizationApplied)

	service.UpdateUserPattern("user_123", true, 3*time.Second, "device_fingerprint_abc", "Beijing")

	t.Log("无缝优化服务测试通过")
}

func TestEnhancedAdaptiveDifficultyService(t *testing.T) {
	t.Log("测试增强型自适应难度服务...")

	service := NewEnhancedAdaptiveDifficultyService()

	difficulty, recommendation := service.GetEnhancedDifficulty(
		"user_123",
		&DifficultyContext{
			HighRiskContext: false,
			TimeSensitive:  true,
		},
	)

	t.Logf("推荐难度: %s", difficulty)
	t.Logf("基础难度: %s", recommendation.BaseDifficulty)
	t.Logf("置信度: %.2f", recommendation.Confidence)
	t.Logf("推理: %s", recommendation.Reasoning)

	service.UpdateDifficultyWithResult("user_123", difficulty, true, 5*time.Second, "slider")

	analytics := service.GetUserAnalytics("user_123")
	if analytics != nil {
		t.Logf("用户分析 - 总尝试次数: %d, 成功次数: %d, 成功率: %.2f%%",
			analytics.TotalAttempts,
			analytics.SuccessCount,
			analytics.SuccessRate)
	}

	t.Log("增强型自适应难度服务测试通过")
}

func TestIntelligentRecommendationService(t *testing.T) {
	t.Log("测试智能推荐服务...")

	service := NewIntelligentRecommendationService()

	req := &RecommendationRequest{
		UserID:         "user_123",
		DeviceFingerprint: "device_abc",
		ApplicationID:   1,
		Context: &RecommendationContext{
			Action:    "login",
			SessionID: "session_123",
			TimeOfDay: 10,
		},
		UserProfile: &CaptchaUserProfile{
			UserID:        "user_123",
			SuccessRate:   85.0,
			AvgResponseTime: 5.0,
		},
	}

	result := service.GetRecommendation(req)

	if result == nil {
		t.Error("推荐结果为空")
		return
	}

	t.Logf("推荐方法: %s", result.RecommendedMethod)
	t.Logf("置信度: %.2f", result.Confidence)
	t.Logf("预计时间: %.2f秒", result.EstimatedTime)
	t.Logf("预计成功率: %.2f%%", result.EstimatedSuccessRate)
	t.Logf("推理: %s", result.Reasoning)

	if len(result.AlternativeMethods) > 0 {
		t.Logf("备选方案: %d个", len(result.AlternativeMethods))
	}

	service.UpdateUserProfile("user_123", result.RecommendedMethod, true, 4.5, nil)

	t.Log("智能推荐服务测试通过")
}

func TestBehaviorPredictionService(t *testing.T) {
	t.Log("测试行为预测服务...")

	service := NewBehaviorPredictionService()

	req := &PredictionRequest{
		UserID:    "user_123",
		SessionID: "session_abc",
		CurrentAction: &UserAction{
			ActionType: "click_submit",
			Timestamp: time.Now(),
			Duration:  500 * time.Millisecond,
			Success:   true,
		},
		RecentActions: []UserAction{
			{
				ActionType: "type_username",
				Timestamp:  time.Now().Add(-5 * time.Second),
				Duration:   2 * time.Second,
				Success:    true,
			},
			{
				ActionType: "click_password_field",
				Timestamp:  time.Now().Add(-3 * time.Second),
				Duration:   500 * time.Millisecond,
				Success:    true,
			},
			{
				ActionType: "type_password",
				Timestamp:  time.Now().Add(-2 * time.Second),
				Duration:   2 * time.Second,
				Success:    true,
			},
		},
		EnvironmentData: map[string]interface{}{
			"ip_address": "192.168.1.100",
			"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		},
	}

	result := service.PredictUserBehavior(req)

	if result == nil {
		t.Error("预测结果为空")
		return
	}

	t.Logf("预测意图: %s", result.PredictedIntent.Type)
	t.Logf("意图置信度: %.2f", result.PredictedIntent.Confidence)
	t.Logf("风险评分: %.2f", result.RiskAssessment.OverallRiskScore)
	t.Logf("风险等级: %s", result.RiskAssessment.RiskLevel)
	t.Logf("应拦截: %v", result.ShouldIntercept)
	t.Logf("推荐操作: %s", result.RecommendedAction)
	t.Logf("警告级别: %s", result.WarningLevel)

	if len(result.NextActions) > 0 {
		t.Logf("预测下一步操作: %v", result.NextActions)
	}

	t.Log("行为预测服务测试通过")
}

func TestIntegration(t *testing.T) {
	t.Log("测试服务集成...")

	seamlessService := NewSeamlessOptimizationService()
	adaptiveService := NewEnhancedAdaptiveDifficultyService()
	recommendService := NewIntelligentRecommendationService()
	predictionService := NewBehaviorPredictionService()

	userID := "user_integration_test"
	sessionID := "session_integration_123"

	predReq := &PredictionRequest{
		UserID:    userID,
		SessionID: sessionID,
		RecentActions: []UserAction{
			{ActionType: "page_load", Timestamp: time.Now(), Duration: 1 * time.Second},
			{ActionType: "click_login", Timestamp: time.Now(), Duration: 300 * time.Millisecond},
		},
	}

	prediction := predictionService.PredictUserBehavior(predReq)

	if prediction.ShouldIntercept {
		t.Log("检测到需要拦截，继续验证流程")
	}

	seamlessResult, _ := seamlessService.OptimizeSeamlessVerification(
		userID,
		"device_integration",
		[]models.BehaviorData{},
		nil,
		prediction.RiskAssessment.OverallRiskScore,
	)

	if !seamlessResult.ShouldChallenge {
		t.Log("无缝验证跳过，用户体验优化成功")
	} else {
		t.Log("需要展示验证码")

		difficulty, _ := adaptiveService.GetEnhancedDifficulty(userID, nil)
		t.Logf("自适应难度: %s", difficulty)

		recReq := &RecommendationRequest{
			UserID: userID,
			Context: &RecommendationContext{
				SessionID: sessionID,
				IsHighRisk: prediction.RiskAssessment.RiskLevel != "low",
			},
		}

		recResult := recommendService.GetRecommendation(recReq)
		t.Logf("智能推荐: %s (置信度: %.2f)", recResult.RecommendedMethod, recResult.Confidence)
	}

	seamlessService.UpdateUserPattern(userID, true, 4*time.Second, "device_integration", "TestCity")

	t.Log("服务集成测试通过")
}
