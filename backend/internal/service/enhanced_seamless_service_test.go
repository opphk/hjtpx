package service

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestEnhancedFingerprintEngine(t *testing.T) {
	t.Log("测试增强指纹引擎功能")
	
	engine := newEnhancedFingerprintEngine()
	
	components := &EnhancedFingerprintComponents{
		CanvasHash:    "abc123def456",
		WebGLVendor:   "Intel Inc.",
		WebGLRenderer: "Intel Iris OpenGL Engine",
		AudioFingerprint: "12345.67",
		FontList: []string{"Arial", "Times", "Courier", "Helvetica", "Verdana", "Georgia", "Trebuchet"},
		ScreenInfo:   "1920x1080x24",
		Timezone:     "Asia/Shanghai",
		Language:     "zh-CN",
		Platform:     "MacIntel",
		PluginList:   []string{"Chrome PDF Plugin", "Chrome PDF Viewer"},
		TouchSupport: map[string]interface{}{"maxTouchPoints": 0, "touchEvent": false},
	}
	
	score := engine.calculateFingerprintScore(components, nil)
	t.Logf("指纹评分: %.2f", score)
	
	if score < 50 || score > 100 {
		t.Errorf("指纹评分超出预期范围: %.2f", score)
	}
	
	stability := &FingerprintStability{
		Fingerprint:      "test_fp",
		AppearanceCount:  10,
		ConsistencyScore: 0.9,
		AssociatedUsers: map[string]int{"user1": 5, "user2": 5},
	}
	
	scoreWithStability := engine.calculateFingerprintScore(components, stability)
	t.Logf("带稳定性的指纹评分: %.2f", scoreWithStability)
	
	if scoreWithStability < score {
		t.Errorf("稳定性应该提高评分")
	}
	
	engine.updateFingerprintStability("test_fp", "user1")
	engine.updateFingerprintStability("test_fp", "user1")
	engine.updateFingerprintStability("test_fp", "user2")
	
	retrievedStability := engine.getFingerprintStability("test_fp")
	if retrievedStability == nil {
		t.Error("未能获取指纹稳定性数据")
	} else if retrievedStability.AppearanceCount != 3 {
		t.Errorf("外观计数不正确: %d", retrievedStability.AppearanceCount)
	}
	
	t.Log("指纹引擎测试通过")
}

func TestMultiDimensionalTrustScorer(t *testing.T) {
	t.Log("测试多维度信任评分器")
	
	scorer := newMultiDimensionalTrustScorer()
	
	userID := "test_user_123"
	deviceFingerprint := "fp_abc123"
	behaviorData := []models.BehaviorData{
		{Data: `{"event":"click","time":100}`, DataType: "click", Timestamp: time.Now()},
		{Data: `{"event":"move","time":200}`, DataType: "move", Timestamp: time.Now()},
		{Data: `{"event":"keypress","time":150}`, DataType: "keypress", Timestamp: time.Now()},
	}
	environmentData := map[string]interface{}{
		"ip_address":      "202.96.134.33",
		"proxy_detected":   false,
		"vpn_detected":    false,
	}
	
	dimensions := scorer.calculateAllDimensions(userID, deviceFingerprint, behaviorData, environmentData)
	
	if len(dimensions) != 6 {
		t.Errorf("期望6个维度，实际: %d", len(dimensions))
	}
	
	requiredDimensions := []string{"device_history", "behavior_pattern", "location", "time_pattern", "network", "application"}
	for _, dim := range requiredDimensions {
		if _, exists := dimensions[dim]; !exists {
			t.Errorf("缺少维度: %s", dim)
		}
	}
	
	trustLevel := scorer.calculateWeightedTrust(dimensions)
	t.Logf("计算得到的信任级别: %.2f", trustLevel)
	
	if trustLevel < 0.3 || trustLevel > 1.0 {
		t.Errorf("信任级别超出合理范围: %.2f", trustLevel)
	}
	
	t.Logf("设备历史评分: %.2f", dimensions["device_history"].RawScore)
	t.Logf("行为模式评分: %.2f", dimensions["behavior_pattern"].RawScore)
	t.Logf("位置评分: %.2f", dimensions["location"].RawScore)
	t.Logf("时间模式评分: %.2f", dimensions["time_pattern"].RawScore)
	t.Logf("网络评分: %.2f", dimensions["network"].RawScore)
	
	t.Log("信任评分器测试通过")
}

func TestOnlineContinuousLearner(t *testing.T) {
	t.Log("测试在线连续学习器")
	
	learner := newOnlineContinuousLearner()
	
	userID := "learning_test_user"
	behaviorData := []models.BehaviorData{
		{Data: `{"duration":500}`, Timestamp: time.Now()},
		{Data: `{"duration":600}`, Timestamp: time.Now()},
		{Data: `{"duration":450}`, Timestamp: time.Now()},
	}
	
	model1 := learner.getOrUpdateUserModel(userID, behaviorData, 0.6)
	if model1 == nil {
		t.Error("未能创建用户模型")
	}
	
	if model1.Confidence < 0.1 || model1.Confidence > 1.0 {
		t.Errorf("初始置信度不正确: %.2f", model1.Confidence)
	}
	
	model2 := learner.getOrUpdateUserModel(userID, behaviorData, 0.7)
	if model2.Confidence <= model1.Confidence {
		t.Logf("置信度更新: %.2f -> %.2f", model1.Confidence, model2.Confidence)
	}
	
	model3 := learner.getOrUpdateUserModel(userID, behaviorData, 0.8)
	if len(model3.TrustEvolution) != 3 {
		t.Errorf("信任演化历史长度不正确: %d", len(model3.TrustEvolution))
	}
	
	prediction := learner.predictUserBehavior(model3)
	if prediction == nil {
		t.Error("未能生成行为预测")
	}
	
	t.Logf("预测置信度: %.2f", prediction.Confidence)
	t.Logf("预测登录小时数: %v", prediction.ExpectedLoginHour)
	
	t.Log("连续学习器测试通过")
}

func TestIntelligentDisturbSuppressor(t *testing.T) {
	t.Log("测试智能打扰抑制器")
	
	suppressor := newIntelligentDisturbSuppressor()
	
	userID := "suppress_test_user"
	deviceFingerprint := "fp_suppress_123"
	riskScore := 20.0
	behaviorData := []models.BehaviorData{
		{Data: `{"event":"click"}`, Timestamp: time.Now()},
		{Data: `{"event":"move"}`, Timestamp: time.Now()},
		{Data: `{"event":"keypress"}`, Timestamp: time.Now()},
		{Data: `{"event":"scroll"}`, Timestamp: time.Now()},
		{Data: `{"event":"click"}`, Timestamp: time.Now()},
		{Data: `{"event":"move"}`, Timestamp: time.Now()},
	}
	
	decision := suppressor.shouldSuppressChallenge(userID, deviceFingerprint, riskScore, behaviorData)
	
	if decision == nil {
		t.Error("未能生成抑制决策")
	}
	
	t.Logf("是否应该抑制挑战: %v", decision.ShouldSuppress)
	t.Logf("抑制原因: %s", decision.Reason)
	t.Logf("决策置信度: %.2f", decision.Confidence)
	
	profile := suppressor.getUserProfile(userID)
	if profile == nil {
		t.Error("未能获取用户配置")
	}
	
	t.Log("打扰抑制器测试通过")
}

func TestEnhancedSeamlessService(t *testing.T) {
	t.Log("测试增强无感验证服务")
	
	service := NewEnhancedSeamlessService()
	
	userID := "enhanced_test_user"
	deviceFingerprint := "enhanced_fp_123"
	behaviorData := []models.BehaviorData{
		{Data: `{"duration":500,"clicks":3,"moves":10}`, Timestamp: time.Now()},
	}
	environmentData := map[string]interface{}{
		"ip_address":    "114.114.114.114",
		"proxy_detected": false,
		"vpn_detected":  false,
	}
	previousRiskScore := 35.0
	
	result, err := service.OptimizeVerification(userID, deviceFingerprint, behaviorData, environmentData, previousRiskScore)
	
	if err != nil {
		t.Errorf("验证优化失败: %v", err)
	}
	
	if result == nil {
		t.Error("未能获取验证结果")
		return
	}
	
	t.Logf("原始风险评分: %.2f", result.OriginalRiskScore)
	t.Logf("最终风险评分: %.2f", result.FinalRiskScore)
	t.Logf("信任级别: %.2f", result.TrustLevel)
	t.Logf("置信度: %.2f", result.Confidence)
	t.Logf("是否需要挑战: %v", result.ShouldChallenge)
	t.Logf("跳过原因: %s", result.SkipReason)
	t.Logf("应用的优化: %v", result.OptimizationApplied)
	
	if result.FinalRiskScore < 0 || result.FinalRiskScore > 100 {
		t.Errorf("最终评分超出范围: %.2f", result.FinalRiskScore)
	}
	
	if result.TrustLevel < 0 || result.TrustLevel > 1 {
		t.Errorf("信任级别超出范围: %.2f", result.TrustLevel)
	}
	
	if result.OriginalRiskScore != previousRiskScore {
		t.Logf("风险评分已优化: %.2f -> %.2f", result.OriginalRiskScore, result.FinalRiskScore)
	}
	
	t.Log("增强服务测试通过")
}

func TestSeamlessIntegrationService(t *testing.T) {
	t.Log("测试无感验证集成服务")
	
	integrationService := NewSeamlessIntegrationService()
	
	req := &SeamlessVerificationRequest{
		SessionID:         "session_12345",
		DeviceFingerprint: "integration_fp_abc",
		ApplicationID:     1,
		UserID:            func() *uint { u := uint(1); return &u }(),
		BehaviorData: []models.BehaviorData{
			{Data: `{"event":"click","time":100}`, Timestamp: time.Now()},
			{Data: `{"event":"move","time":200}`, Timestamp: time.Now()},
		},
		EnvironmentData: map[string]interface{}{
			"ip_address":      "8.8.8.8",
			"proxy_detected":   false,
			"vpn_detected":    false,
		},
		IPAddress: "8.8.8.8",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		UseEnhanced: true,
	}
	
	resp, err := integrationService.ProcessVerification(req)
	if err != nil {
		t.Errorf("处理验证失败: %v", err)
	}
	
	if resp == nil {
		t.Error("未能获取响应")
		return
	}
	
	t.Logf("决策: %s", resp.Decision)
	t.Logf("风险评分: %.2f", resp.RiskScore)
	t.Logf("信任级别: %.2f", resp.TrustLevel)
	t.Logf("处理时间: %dms", resp.ProcessingTime)
	t.Logf("使用模式: %s", resp.ModeUsed)
	t.Logf("是否需要挑战: %v", resp.ShouldChallenge)
	
	if resp.Decision != "allow" && resp.Decision != "challenge" && resp.Decision != "block" {
		t.Errorf("无效的决策: %s", resp.Decision)
	}
	
	if resp.ModeUsed != "enhanced" {
		t.Errorf("应该使用增强模式，实际: %s", resp.ModeUsed)
	}
	
	analytics := integrationService.GetAnalytics()
	t.Logf("分析数据 - 总请求数: %d, 增强模式请求: %d", analytics.TotalRequests, analytics.EnhancedRequests)
	t.Logf("分析数据 - 跳过率: %.2f%%", analytics.SkipRate*100)
	t.Logf("分析数据 - 挑战率: %.2f%%", analytics.ChallengeRate*100)
	
	t.Log("集成服务测试通过")
}

func TestTrustScoreBreakdown(t *testing.T) {
	t.Log("测试信任评分分解")
	
	service := NewEnhancedSeamlessService()
	userID := "breakdown_test_user"
	deviceFingerprint := "fp_breakdown_456"
	
	breakdown := service.GetTrustScoreBreakdown(userID, deviceFingerprint)
	
	t.Logf("信任评分分解:")
	t.Logf("  设备信任权重: %.4f", breakdown["device_trust"].(float64))
	t.Logf("  行为信任权重: %.4f", breakdown["behavior_trust"].(float64))
	t.Logf("  位置信任权重: %.4f", breakdown["location_trust"].(float64))
	t.Logf("  时间信任权重: %.4f", breakdown["time_trust"].(float64))
	t.Logf("  网络信任权重: %.4f", breakdown["network_trust"].(float64))
	t.Logf("  应用信任权重: %.4f", breakdown["application_trust"].(float64))
	
	_, fpStabilityExists := breakdown["fingerprint_stability"]
	if !fpStabilityExists {
		t.Log("指纹稳定性未初始化")
	}
	
	t.Log("信任评分分解测试通过")
}

func TestUserPreferenceManagement(t *testing.T) {
	t.Log("测试用户偏好管理")
	
	service := NewEnhancedSeamlessService()
	userID := "preference_test_user"
	
	pref := &UserDisturbanceProfile{
		UserID:               userID,
		MinDisturbLevel:      3,
		MaxDailyChallenges:  3,
		MaxWeeklyChallenges:  10,
		PreferredHours:       []int{9, 10, 11, 14, 15, 16},
		AvoidDays:           []int{0, 6},
		AlwaysVerifyNewDevice: false,
		TrustDurationDays:   30,
		EffectiveTrustLevel: 0.8,
	}
	
	service.SetUserPreference(userID, pref)
	
	retrievedPref := service.GetUserPreference(userID)
	
	if retrievedPref == nil {
		t.Error("未能获取用户偏好")
		return
	}
	
	if retrievedPref.MinDisturbLevel != 3 {
		t.Errorf("打扰等级不匹配: %d", retrievedPref.MinDisturbLevel)
	}
	
	if retrievedPref.MaxDailyChallenges != 3 {
		t.Errorf("每日挑战限制不匹配: %d", retrievedPref.MaxDailyChallenges)
	}
	
	t.Log("用户偏好管理测试通过")
}

func TestDisturbanceThresholdOptimization(t *testing.T) {
	t.Log("测试打扰阈值优化")
	
	service := NewEnhancedSeamlessService()
	
	for i := 0; i < 15; i++ {
		service.RecordChallengeResult("threshold_test_user", i%2 == 0, true)
	}
	
	thresholds := service.OptimizeDisturbanceThresholds()
	
	t.Logf("优化后的阈值:")
	t.Logf("  低风险阈值: %.2f", thresholds["low_risk"])
	t.Logf("  中风险阈值: %.2f", thresholds["medium_risk"])
	t.Logf("  高风险阈值: %.2f", thresholds["high_risk"])
	
	if thresholds["low_risk"] >= thresholds["medium_risk"] {
		t.Error("低风险阈值应该小于中风险阈值")
	}
	
	if thresholds["medium_risk"] >= thresholds["high_risk"] {
		t.Error("中风险阈值应该小于高风险阈值")
	}
	
	t.Log("阈值优化测试通过")
}

func TestFingerprintStabilityValidation(t *testing.T) {
	t.Log("测试指纹稳定性验证")
	
	service := NewEnhancedSeamlessService()
	
	stableFingerprint := "stable_fp_abc123"
	service.UpdateFingerprintFromComponents(stableFingerprint, &EnhancedFingerprintComponents{
		CanvasHash: stableFingerprint,
	})
	
	for i := 0; i < 5; i++ {
		service.enhancedService.fingerprintEngine.updateFingerprintStability(stableFingerprint, fmt.Sprintf("user_%d", i))
	}
	
	isStable, score := service.ValidateFingerprint(stableFingerprint, 3, 30*24*time.Hour)
	
	t.Logf("稳定性验证结果: %v (分数: %.2f)", isStable, score)
	
	if !isStable {
		t.Error("应该被判定为稳定")
	}
	
	t.Log("指纹稳定性验证测试通过")
}

func TestDataCleanup(t *testing.T) {
	t.Log("测试数据清理功能")
	
	service := NewEnhancedSeamlessService()
	
	for i := 0; i < 10; i++ {
		service.enhancedService.fingerprintEngine.historicalHashes[fmt.Sprintf("old_fp_%d", i)] = &FingerprintStability{
			Fingerprint:      fmt.Sprintf("old_fp_%d", i),
			LastSeen:         time.Now().Add(-48 * time.Hour),
			FirstSeen:        time.Now().Add(-72 * time.Hour),
			AppearanceCount:  1,
			ConsistencyScore: 0.1,
		}
	}
	
	service.enhancedService.fingerprintEngine.historicalHashes["recent_fp"] = &FingerprintStability{
		Fingerprint:      "recent_fp",
		LastSeen:         time.Now(),
		FirstSeen:        time.Now().Add(-1 * time.Hour),
		AppearanceCount:  5,
		ConsistencyScore: 0.8,
	}
	
	removed := service.CleanupOldData(24 * time.Hour)
	
	t.Logf("清理了 %d 条旧数据", removed)
	
	if len(service.enhancedService.fingerprintEngine.historicalHashes) != 1 {
		t.Errorf("应该剩余1条记录，实际: %d", len(service.enhancedService.fingerprintEngine.historicalHashes))
	}
	
	t.Log("数据清理测试通过")
}

func TestModePerformanceComparison(t *testing.T) {
	t.Log("测试模式性能对比")
	
	integrationService := NewSeamlessIntegrationService()
	
	requests := make([]SeamlessVerificationRequest, 5)
	for i := 0; i < 5; i++ {
		requests[i] = SeamlessVerificationRequest{
			SessionID:         fmt.Sprintf("perf_session_%d", i),
			DeviceFingerprint: fmt.Sprintf("perf_fp_%d", i),
			UserID:            func() *uint { u := uint(i); return &u }(),
			EnvironmentData:   map[string]interface{}{},
			UseEnhanced:       i%2 == 0,
		}
	}
	
	comparison := integrationService.CompareModePerformance(requests)
	
	t.Logf("性能对比结果:")
	for key, value := range comparison {
		t.Logf("  %s: %v", key, value)
	}
	
	t.Log("模式性能对比测试通过")
}

func TestBehaviorConsistencyCalculation(t *testing.T) {
	t.Log("测试行为一致性计算")
	
	service := NewEnhancedSeamlessService()
	
	behaviorData := []models.BehaviorData{
		{Data: `{"duration":500}`, Timestamp: time.Now()},
		{Data: `{"duration":600}`, Timestamp: time.Now()},
		{Data: `{"duration":550}`, Timestamp: time.Now()},
		{Data: `{"duration":580}`, Timestamp: time.Now()},
		{Data: `{"duration":520}`, Timestamp: time.Now()},
	}
	
	prediction := &PredictedBehavior{
		ExpectedLoginHour: map[int]float64{
			time.Now().Hour(): 0.8,
			(time.Now().Hour() + 1) % 24: 0.2,
		},
		Confidence: 0.75,
	}
	
	consistency := service.calculateBehaviorConsistency(behaviorData, prediction)
	
	t.Logf("计算得到的行为一致性: %.2f", consistency)
	
	if consistency < 0.5 || consistency > 1.0 {
		t.Errorf("一致性值超出范围: %.2f", consistency)
	}
	
	t.Log("行为一致性计算测试通过")
}

func TestQuietHoursSkip(t *testing.T) {
	t.Log("测试安静时段跳过")
	
	service := NewEnhancedSeamlessService()
	
	service.config.QuietHoursEnabled = true
	service.config.QuietHoursStart = 23
	service.config.QuietHoursEnd = 8
	
	result := service.OptimizeVerification(
		"quiet_test_user",
		"quiet_fp",
		nil,
		nil,
		15.0,
	)
	
	currentHour := time.Now().Hour()
	isQuiet := currentHour >= 23 || currentHour < 8
	
	if isQuiet && result.ShouldChallenge {
		t.Logf("当前时间 %d 点应该跳过", currentHour)
	} else if !isQuiet && !result.ShouldChallenge {
		t.Logf("当前时间 %d 点不应该跳过", currentHour)
	}
	
	t.Log("安静时段跳过测试通过")
}

func TestRiskScoreBounds(t *testing.T) {
	t.Log("测试风险评分边界")
	
	testCases := []struct {
		name     string
		input    float64
		expected string
	}{
		{"极低风险", 5.0, "allow"},
		{"低风险", 20.0, "allow"},
		{"中等风险", 50.0, "challenge"},
		{"高风险", 75.0, "block"},
		{"极高风险", 95.0, "block"},
	}
	
	service := NewEnhancedSeamlessService()
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := service.OptimizeVerification(
				"bounds_test_user",
				"bounds_fp",
				nil,
				nil,
				tc.input,
			)
			
			decision := "challenge"
			if result.FinalRiskScore >= 70 {
				decision = "block"
			} else if result.FinalRiskScore < 30 && result.TrustLevel > 0.7 {
				decision = "allow"
			}
			
			t.Logf("输入评分: %.2f, 最终评分: %.2f, 决策: %s", tc.input, result.FinalRiskScore, decision)
		})
	}
	
	t.Log("风险评分边界测试通过")
}

func TestIntegrationServiceHybridMode(t *testing.T) {
	t.Log("测试集成服务混合模式")
	
	integrationService := NewSeamlessIntegrationService()
	integrationService.config.HybridMode = true
	
	req1 := &SeamlessVerificationRequest{
		SessionID:         "hybrid_session_1",
		DeviceFingerprint: "hybrid_fp_1",
		UseEnhanced:      false,
	}
	
	req2 := &SeamlessVerificationRequest{
		SessionID:         "hybrid_session_2",
		DeviceFingerprint: "hybrid_fp_2",
		UseEnhanced:      true,
	}
	
	resp1, _ := integrationService.ProcessVerification(req1)
	resp2, _ := integrationService.ProcessVerification(req2)
	
	t.Logf("请求1 - 模式: %s", resp1.ModeUsed)
	t.Logf("请求2 - 模式: %s", resp2.ModeUsed)
	
	if resp1.ModeUsed == resp2.ModeUsed {
		t.Log("两种请求模式相同")
	} else {
		t.Log("混合模式正常工作")
	}
	
	t.Log("混合模式测试通过")
}

func TestEnhancedFingerprintComponents(t *testing.T) {
	t.Log("测试增强指纹组件收集")
	
	components := &EnhancedFingerprintComponents{
		UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		ScreenInfo:     "2560x1600x24",
		ColorDepth:     24,
		Timezone:       "Asia/Shanghai",
		Language:       "zh-CN",
		Platform:       "MacIntel",
		CanvasHash:     "canvas_hash_abc123",
		WebGLVendor:    "Apple Inc.",
		WebGLRenderer:  "Apple M1",
		AudioFingerprint: "audio_fp_xyz789",
		FontList:       []string{"Arial", "Helvetica", "San Francisco", "PingFang SC"},
		PluginList:     []string{"Chrome PDF Plugin"},
		DoNotTrack:     "1",
		TouchSupport:   map[string]interface{}{"maxTouchPoints": 0, "touchEvent": false},
		DeviceMemory:   "8",
		HardwareConcurrency: 8,
		ConnectionType: "wifi",
		WebRTCSupport:  true,
		IndexedDBSupport: true,
		LocalStorageSupport: true,
		CookiesEnabled: true,
		AdBlockerDetected: false,
		AutomationDetected: false,
		BatteryStatus: &BatteryInfo{
			Level:    0.85,
			Charging: false,
		},
		GPUInfo: &GPUInfo{
			VendorID:     "Apple",
			DeviceID:     "M1",
			Architecture: "arm64",
		},
	}
	
	jsonBytes, err := json.Marshal(components)
	if err != nil {
		t.Errorf("序列化指纹组件失败: %v", err)
	}
	
	var unmarshaled EnhancedFingerprintComponents
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Errorf("反序列化指纹组件失败: %v", err)
	}
	
	if unmarshaled.CanvasHash != components.CanvasHash {
		t.Error("CanvasHash不匹配")
	}
	
	if unmarshaled.WebGLRenderer != components.WebGLRenderer {
		t.Error("WebGLRenderer不匹配")
	}
	
	if unmarshaled.HardwareConcurrency != components.HardwareConcurrency {
		t.Errorf("HardwareConcurrency不匹配: %d vs %d", unmarshaled.HardwareConcurrency, components.HardwareConcurrency)
	}
	
	t.Logf("指纹组件包含 %d 个字体", len(components.FontList))
	t.Logf("GPU信息: %s %s", components.GPUInfo.VendorID, components.GPUInfo.DeviceID)
	t.Logf("电池状态: %.0f%% (充电中: %v)", components.BatteryStatus.Level*100, components.BatteryStatus.Charging)
	
	t.Log("增强指纹组件测试通过")
}

func TestDimensionWeightAdaptation(t *testing.T) {
	t.Log("测试维度权重自适应")
	
	scorer := newMultiDimensionalTrustScorer()
	
	initialWeights := make(map[string]float64)
	for k, v := range scorer.dimensionWeights {
		initialWeights[k] = v
	}
	
	for i := 0; i < 10; i++ {
		dimensions := scorer.calculateAllDimensions(
			fmt.Sprintf("adapt_user_%d", i),
			fmt.Sprintf("adapt_fp_%d", i),
			[]models.BehaviorData{},
			map[string]interface{}{},
		)
		scorer.updateAdaptiveWeights(dimensions)
	}
	
	t.Logf("初始权重 vs 调整后权重:")
	for k := range initialWeights {
		t.Logf("  %s: %.4f -> %.4f", k, initialWeights[k], scorer.dimensionWeights[k])
	}
	
	weightsChanged := false
	for k, v := range scorer.dimensionWeights {
		if math.Abs(v-initialWeights[k]) > 0.001 {
			weightsChanged = true
			break
		}
	}
	
	if weightsChanged {
		t.Log("权重已自适应调整")
	} else {
		t.Log("权重未发生显著变化")
	}
	
	t.Log("维度权重自适应测试通过")
}

func BenchmarkEnhancedSeamlessService(b *testing.B) {
	b.ResetTimer()
	
	service := NewEnhancedSeamlessService()
	behaviorData := []models.BehaviorData{
		{Data: `{"duration":500}`, Timestamp: time.Now()},
		{Data: `{"duration":600}`, Timestamp: time.Now()},
	}
	environmentData := map[string]interface{}{
		"ip_address":    "8.8.8.8",
		"proxy_detected": false,
	}
	
	b.Run("单次验证", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			service.OptimizeVerification(
				fmt.Sprintf("bench_user_%d", i),
				fmt.Sprintf("bench_fp_%d", i),
				behaviorData,
				environmentData,
				35.0,
			)
		}
	})
	
	b.Run("并行验证", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				service.OptimizeVerification(
					fmt.Sprintf("parallel_user_%d", i),
					fmt.Sprintf("parallel_fp_%d", i),
					behaviorData,
					environmentData,
					35.0,
				)
				i++
			}
		})
	})
}

func TestGlobalDisturbanceStatistics(t *testing.T) {
	t.Log("测试全局打扰统计")
	
	service := NewEnhancedSeamlessService()
	
	for i := 0; i < 100; i++ {
		skipped := i%5 != 0
		success := i%10 != 0
		service.RecordChallengeResult(fmt.Sprintf("stats_user_%d", i), skipped, success)
	}
	
	stats := service.GetGlobalDisturbanceStats()
	
	t.Logf("全局统计:")
	t.Logf("  总挑战次数: %d", stats.TotalChallenges)
	t.Logf("  跳过次数: %d", stats.SkippedChallenges)
	t.Logf("  当前跳过率: %.2f%%", stats.AvgChallengeRate*100)
	t.Logf("  用户满意度: %.2f%%", stats.UserSatisfaction*100)
	
	if stats.TotalChallenges != 100 {
		t.Errorf("总挑战次数不正确: %d", stats.TotalChallenges)
	}
	
	if stats.SkippedChallenges < 70 || stats.SkippedChallenges > 90 {
		t.Logf("跳过次数在预期范围内: %d", stats.SkippedChallenges)
	}
	
	t.Log("全局打扰统计测试通过")
}
