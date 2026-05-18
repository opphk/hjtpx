package service

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewAdvancedRuleEngine(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}
	if len(engine.rules) == 0 {
		t.Error("Expected rules to be initialized")
	}
	if engine.statistics == nil {
		t.Error("Expected statistics to be initialized")
	}
	if engine.combinators == nil {
		t.Error("Expected combinators to be initialized")
	}
}

func TestAdvancedRuleEngineInitialization(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	expectedRules := []string{
		"trajectory_speed_too_fast",
		"trajectory_speed_too_slow",
		"trajectory_curvature_too_smooth",
		"trajectory_curvature_abnormal_jitter",
		"click_interval_too_short",
		"click_interval_too_long",
		"click_position_too_concentrated",
		"slider_release_precision_too_high",
		"captcha_attempt_frequency_too_high",
		"captcha_failure_rate_too_high",
		"device_fingerprint_duplicate",
		"ip_subnet_abnormal_access",
		"session_behavior_abnormal",
		"mouse_trajectory_mechanical",
		"keyboard_input_rhythm_abnormal",
		"combined_trajectory_anomaly",
		"combined_click_pattern_anomaly",
		"captcha_rapid_retry",
		"device_ip_session_correlation",
		"proxy_vpn_tor_detection",
		"timing_behavior_inconsistency",
	}

	for _, ruleName := range expectedRules {
		rule, exists := engine.rules[ruleName]
		if !exists {
			t.Errorf("Expected rule %s to exist", ruleName)
		}
		if rule != nil && !rule.Enabled {
			t.Errorf("Expected rule %s to be enabled by default", ruleName)
		}
	}
}

func TestTrajectorySpeedTooFast(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2000,
		},
	}

	result := engine.Evaluate(ctx)
	if len(result.TriggeredRules) == 0 {
		t.Error("Expected trajectory_speed_too_fast rule to trigger")
	}

	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "trajectory_speed_too_fast" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected trajectory_speed_too_fast to be in triggered rules")
	}
}

func TestTrajectorySpeedTooSlow(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 5,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "trajectory_speed_too_slow" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected trajectory_speed_too_slow to be in triggered rules")
	}
}

func TestTrajectoryCurvatureSmooth(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		TrajectoryFeatures: &TrajectoryFeatures{
			CurvatureAverage:  0.005,
			CurvatureVariance: 0.0005,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "trajectory_curvature_too_smooth" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected trajectory_curvature_too_smooth to be in triggered rules")
	}
}

func TestClickIntervalTooShort(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		ClickFeatures: &ClickFeatures{
			IntervalAverage: 20,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "click_interval_too_short" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected click_interval_too_short to be in triggered rules")
	}
}

func TestClickPositionConcentrated(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		ClickFeatures: &ClickFeatures{
			ClusteringScore:   0.95,
			PositionEntropy:    1.0,
			PositionVarianceX: 100,
			PositionVarianceY: 100,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "click_position_too_concentrated" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected click_position_too_concentrated to be in triggered rules")
	}
}

func TestSliderReleasePrecision(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		SliderFeatures: &AdvancedSliderFeatures{
			Precision:   0.99,
			Directness:  0.995,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "slider_release_precision_too_high" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected slider_release_precision_too_high to be in triggered rules")
	}
}

func TestCaptchaFailureRate(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		CaptchaFeatures: &CaptchaFeatures{
			AttemptCount: 5,
			FailureRate:  0.8,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "captcha_failure_rate_too_high" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected captcha_failure_rate_too_high to be in triggered rules")
	}
}

func TestCaptchaAttemptFrequency(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		CaptchaFeatures: &CaptchaFeatures{
			AttemptCount:     4,
			AttemptFrequency: 6,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "captcha_attempt_frequency_too_high" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected captcha_attempt_frequency_too_high to be in triggered rules")
	}
}

func TestProxyVPNTorDetection(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		IPFeatures: &IPFeatures{
			IsProxy: true,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "proxy_vpn_tor_detection" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected proxy_vpn_tor_detection to be in triggered rules")
	}
}

func TestKeyboardInputRhythm(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		KeyboardFeatures: &KeyboardFeatures{
			TypingSpeed:       16,
			HoldTimeVariance:  5,
			HoldTimeAverage:   40,
			Regularity:        0.99,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "keyboard_input_rhythm_abnormal" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected keyboard_input_rhythm_abnormal to be in triggered rules")
	}
}

func TestSessionBehaviorAbnormal(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		SessionFeatures: &SessionFeatures{
			Duration:        10 * time.Minute,
			InteractionCount: 1200,
			BounceRate:      0.95,
			FocusLossCount:  0,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "session_behavior_abnormal" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected session_behavior_abnormal to be in triggered rules")
	}
}

func TestCombinedTrajectoryAnomaly(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		TrajectoryFeatures: &TrajectoryFeatures{
			CurvatureAverage:   0.005,
			CurvatureVariance: 0.0005,
			SpeedConsistency:  0.99,
		},
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2000,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "combined_trajectory_anomaly" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected combined_trajectory_anomaly to be in triggered rules")
	}
}

func TestMouseTrajectoryMechanical(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		TrajectoryFeatures: &TrajectoryFeatures{
			SpeedConsistency:    0.99,
			CurvatureAverage:    0.003,
			Smoothness:          0.99,
			AccelerationVariance: 0.0005,
		},
		BehaviorFeatures: &BehaviorFeatures{},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "mouse_trajectory_mechanical" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected mouse_trajectory_mechanical to be in triggered rules")
	}
}

func TestRuleEnableDisable(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	err := engine.DisableRule("trajectory_speed_too_fast")
	if err != nil {
		t.Errorf("Unexpected error disabling rule: %v", err)
	}

	rule, exists := engine.rules["trajectory_speed_too_fast"]
	if !exists {
		t.Error("Expected rule to still exist")
	}
	if rule.Enabled {
		t.Error("Expected rule to be disabled")
	}

	err = engine.EnableRule("trajectory_speed_too_fast")
	if err != nil {
		t.Errorf("Unexpected error enabling rule: %v", err)
	}
	if !rule.Enabled {
		t.Error("Expected rule to be enabled")
	}
}

func TestRuleWeightConfiguration(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	err := engine.SetRuleWeight("trajectory_speed_too_fast", 50.0)
	if err != nil {
		t.Errorf("Unexpected error setting rule weight: %v", err)
	}

	rule, _ := engine.rules["trajectory_speed_too_fast"]
	if rule.Weight != 50.0 {
		t.Errorf("Expected weight 50.0, got %f", rule.Weight)
	}
}

func TestGetRulesByCategory(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	rules := engine.GetRulesByCategory("speed")
	if len(rules) == 0 {
		t.Error("Expected rules in speed category")
	}

	for _, rule := range rules {
		if rule.Category != "speed" {
			t.Errorf("Expected category speed, got %s", rule.Category)
		}
	}
}

func TestANDCombinator(t *testing.T) {
	combinator := &ANDCombinator{}

	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2000,
		},
	}

	rules := []*AdvancedDetectionRule{
		{
			Name: "test_rule1",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return ctx.BehaviorFeatures != nil && ctx.BehaviorFeatures.AvgSpeed > 1500
			},
			Weight: 30,
		},
		{
			Name: "test_rule2",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return ctx.BehaviorFeatures != nil && ctx.BehaviorFeatures.AvgSpeed > 1800
			},
			Weight: 30,
		},
	}

	result := combinator.Combine(rules, ctx)
	if !result {
		t.Error("Expected AND combinator to return true")
	}

	ctx.BehaviorFeatures.AvgSpeed = 1000
	result = combinator.Combine(rules, ctx)
	if result {
		t.Error("Expected AND combinator to return false")
	}
}

func TestORCombinator(t *testing.T) {
	combinator := &ORCombinator{}

	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 1000,
		},
	}

	rules := []*AdvancedDetectionRule{
		{
			Name: "test_rule1",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return ctx.BehaviorFeatures != nil && ctx.BehaviorFeatures.AvgSpeed > 1500
			},
			Weight: 30,
		},
		{
			Name: "test_rule2",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return ctx.BehaviorFeatures != nil && ctx.BehaviorFeatures.AvgSpeed > 500
			},
			Weight: 30,
		},
	}

	result := combinator.Combine(rules, ctx)
	if !result {
		t.Error("Expected OR combinator to return true")
	}

	ctx.BehaviorFeatures.AvgSpeed = 2000
	result = combinator.Combine(rules, ctx)
	if !result {
		t.Error("Expected OR combinator to return true")
	}
}

func TestThresholdCombinator(t *testing.T) {
	combinator := &ThresholdCombinator{MinTriggers: 2}

	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2000,
		},
	}

	rules := []*AdvancedDetectionRule{
		{
			Name: "test_rule1",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return ctx.BehaviorFeatures != nil && ctx.BehaviorFeatures.AvgSpeed > 1500
			},
			Weight: 30,
		},
		{
			Name: "test_rule2",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return ctx.BehaviorFeatures != nil && ctx.BehaviorFeatures.AvgSpeed > 1800
			},
			Weight: 30,
		},
		{
			Name: "test_rule3",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return ctx.BehaviorFeatures != nil && ctx.BehaviorFeatures.AvgSpeed > 100
			},
			Weight: 30,
		},
	}

	result := combinator.Combine(rules, ctx)
	if !result {
		t.Error("Expected Threshold combinator to return true")
	}

	combinator.MinTriggers = 4
	result = combinator.Combine(rules, ctx)
	if result {
		t.Error("Expected Threshold combinator to return false")
	}
}

func TestWeightedSumCombinator(t *testing.T) {
	combinator := &WeightedSumCombinator{Threshold: 0.6}

	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2000,
		},
	}

	rules := []*AdvancedDetectionRule{
		{
			Name: "test_rule1",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return true
			},
			Weight: 50,
		},
		{
			Name: "test_rule2",
			Condition: func(ctx *AdvancedDetectionContext) bool {
				return true
			},
			Weight: 50,
		},
	}

	result := combinator.Combine(rules, ctx)
	if !result {
		t.Error("Expected WeightedSum combinator to return true")
	}

	rules[0].Condition = func(ctx *AdvancedDetectionContext) bool {
		return false
	}
	rules[1].Condition = func(ctx *AdvancedDetectionContext) bool {
		return false
	}

	result = combinator.Combine(rules, ctx)
	if result {
		t.Error("Expected WeightedSum combinator to return false")
	}
}

func TestCreateCombinedRule(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	err := engine.CreateCombinedRule("custom_combined", []string{
		"trajectory_speed_too_fast",
		"trajectory_curvature_too_smooth",
	}, "AND")

	if err != nil {
		t.Errorf("Unexpected error creating combined rule: %v", err)
	}

	rule, exists := engine.rules["custom_combined"]
	if !exists {
		t.Error("Expected combined rule to exist")
	}
	if rule == nil {
		t.Fatal("Expected non-nil rule")
	}
	if rule.Category != "combined" {
		t.Errorf("Expected category combined, got %s", rule.Category)
	}
	if !rule.Enabled {
		t.Error("Expected combined rule to be enabled")
	}
}

func TestEvaluateNoTrigger(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 500,
		},
	}

	result := engine.Evaluate(ctx)

	if result.IsBot {
		t.Error("Expected IsBot to be false for normal behavior")
	}

	if len(result.TriggeredRules) > 5 {
		t.Errorf("Expected few triggered rules for normal behavior, got %d", len(result.TriggeredRules))
	}
}

func TestRuleStatistics(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2000,
		},
	}

	for i := 0; i < 10; i++ {
		engine.Evaluate(ctx)
	}

	stats := engine.GetStatistics()
	if stats == nil {
		t.Fatal("Expected non-nil statistics")
	}

	if stats.TotalEvaluations != 10 {
		t.Errorf("Expected 10 evaluations, got %d", stats.TotalEvaluations)
	}
}

func TestConcurrentEvaluation(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	var wg sync.WaitGroup
	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2000,
		},
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			engine.Evaluate(ctx)
		}()
	}

	wg.Wait()

	stats := engine.GetStatistics()
	if stats.TotalEvaluations != 100 {
		t.Errorf("Expected 100 evaluations, got %d", stats.TotalEvaluations)
	}
}

func TestExportConfiguration(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	config := engine.ExportConfiguration()
	if config == "" {
		t.Error("Expected non-empty configuration export")
	}

	if len(config) < 100 {
		t.Error("Expected detailed configuration export")
	}
}

func TestBotDetectionServiceWithAdvancedRules(t *testing.T) {
	service := NewBotDetectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	result := service.DetectBot(req, nil)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.RiskScore > 1.0 {
		t.Error("Expected risk score <= 1.0")
	}
}

func TestAdvancedRuleEngineWithNilContext(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	result := engine.Evaluate(nil)

	if result.TotalScore != 0 {
		t.Errorf("Expected zero total score for nil context, got %f", result.TotalScore)
	}

	if result.IsBot {
		t.Error("Expected IsBot to be false for nil context")
	}
}

func TestRuleCombinationTypes(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		BehaviorFeatures: &BehaviorFeatures{
			AvgSpeed: 2500,
		},
		TrajectoryFeatures: &TrajectoryFeatures{
			CurvatureAverage:  0.005,
			CurvatureVariance: 0.0005,
		},
	}

	result := engine.Evaluate(ctx)

	if len(result.TriggeredRules) < 2 {
		t.Errorf("Expected multiple triggered rules for combined test, got %d", len(result.TriggeredRules))
	}

	expectedCombined := false
	for _, rule := range result.TriggeredRules {
		if rule == "combined_trajectory_anomaly" {
			expectedCombined = true
			break
		}
	}
	if !expectedCombined {
		t.Error("Expected combined_trajectory_anomaly to be triggered")
	}
}

func TestHighSeverityRules(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	testCases := []struct {
		name     string
		context  *AdvancedDetectionContext
		expected string
	}{
		{
			name: "device_fingerprint_duplicate",
			context: &AdvancedDetectionContext{
				DeviceFeatures: &DeviceFeatures{
					FingerprintHash: "test_hash_123",
				},
				SessionFeatures: &SessionFeatures{},
			},
			expected: "device_fingerprint_duplicate",
		},
		{
			name: "ip_subnet_abnormal",
			context: &AdvancedDetectionContext{
				IPFeatures: &IPFeatures{
					IsProxy:         true,
					RequestFrequency: 150,
				},
				SessionFeatures: &SessionFeatures{
					UniqueSessions: 5,
				},
			},
			expected: "ip_subnet_abnormal_access",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.Evaluate(tc.context)
			found := false
			for _, rule := range result.TriggeredRules {
				if rule == tc.expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %s to be triggered", tc.expected)
			}
		})
	}
}

func TestRulePriority(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	rule, exists := engine.rules["trajectory_speed_too_fast"]
	if !exists {
		t.Fatal("Expected rule to exist")
	}

	if rule.Priority != 1 {
		t.Errorf("Expected priority 1 for high severity rule, got %d", rule.Priority)
	}
}

func TestRuleWeightBounds(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	err := engine.SetRuleWeight("trajectory_speed_too_fast", 150)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rule, _ := engine.rules["trajectory_speed_too_fast"]
	if rule.Weight != 100 {
		t.Errorf("Expected weight capped at 100, got %f", rule.Weight)
	}

	err = engine.SetRuleWeight("trajectory_speed_too_fast", -10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if rule.Weight != 0 {
		t.Errorf("Expected weight capped at 0, got %f", rule.Weight)
	}
}

func TestGetAllRules(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	rules := engine.GetAllRules()
	if len(rules) != len(engine.rules) {
		t.Errorf("Expected %d rules, got %d", len(engine.rules), len(rules))
	}
}

func TestIPSubnetAbnormalAccess(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		IPFeatures: &IPFeatures{
			IsProxy:          true,
			RequestFrequency: 120,
		},
		SessionFeatures: &SessionFeatures{
			UniqueSessions: 4,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "ip_subnet_abnormal_access" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ip_subnet_abnormal_access to be triggered")
	}
}

func TestTimingBehaviorInconsistency(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		TimeFeatures: &TimeFeatures{
			HourOfDay:        10,
			IsWeekend:        true,
			ActivityDuration: 15 * time.Second,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "timing_behavior_inconsistency" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected timing_behavior_inconsistency to be triggered")
	}
}

func TestCaptchaRapidRetry(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		CaptchaFeatures: &CaptchaFeatures{
			AttemptCount:     5,
			FailureRate:      0.8,
			AttemptFrequency: 7,
		},
	}

	result := engine.Evaluate(ctx)

	hasHighFailure := false
	hasHighFreq := false
	hasRapidRetry := false

	for _, rule := range result.TriggeredRules {
		if rule == "captcha_failure_rate_too_high" {
			hasHighFailure = true
		}
		if rule == "captcha_attempt_frequency_too_high" {
			hasHighFreq = true
		}
		if rule == "captcha_rapid_retry" {
			hasRapidRetry = true
		}
	}

	if !hasHighFailure || !hasHighFreq {
		t.Error("Expected individual captcha rules to trigger")
	}
	if !hasRapidRetry {
		t.Error("Expected captcha_rapid_retry combined rule to trigger")
	}
}

func TestDeviceIPSessionCorrelation(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		DeviceFeatures: &DeviceFeatures{
			FingerprintHash: "duplicate_hash_456",
		},
		IPFeatures: &IPFeatures{
			IsProxy:          true,
			RequestFrequency: 150,
		},
		SessionFeatures: &SessionFeatures{
			UniqueSessions: 6,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "device_ip_session_correlation" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected device_ip_session_correlation to be triggered")
	}
}

func TestClickIntervalTooLong(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		ClickFeatures: &ClickFeatures{
			IntervalAverage: 6000,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "click_interval_too_long" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected click_interval_too_long to be triggered")
	}
}

func TestTrajectoryCurvatureJitter(t *testing.T) {
	engine := NewAdvancedRuleEngine()

	ctx := &AdvancedDetectionContext{
		TrajectoryFeatures: &TrajectoryFeatures{
			CurvatureVariance: 0.9,
			JitterScore:       0.6,
		},
	}

	result := engine.Evaluate(ctx)
	found := false
	for _, rule := range result.TriggeredRules {
		if rule == "trajectory_curvature_abnormal_jitter" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected trajectory_curvature_abnormal_jitter to be triggered")
	}
}
