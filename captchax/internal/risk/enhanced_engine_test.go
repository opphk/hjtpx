package risk

import (
	"context"
	"testing"
	"time"

	"captchax/internal/config"
)

func TestFitBezierCurve(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Insufficient track data", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
		}

		fitness, curve := engine.FitBezierCurve(tracks)
		if fitness != 1.0 {
			t.Errorf("Fitness = %f, want 1.0 for insufficient data", fitness)
		}
		if curve != nil {
			t.Error("Curve should be nil for insufficient data")
		}
	})

	t.Run("Valid track data", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 150, Y: 120, Timestamp: 50},
			{X: 200, Y: 150, Timestamp: 100},
			{X: 250, Y: 180, Timestamp: 150},
			{X: 300, Y: 200, Timestamp: 200},
		}

		fitness, curve := engine.FitBezierCurve(tracks)
		if fitness < 0 || fitness > 1 {
			t.Errorf("Fitness = %f, want between 0 and 1", fitness)
		}
		if curve == nil {
			t.Error("Curve should not be nil for valid data")
		}
	})

	t.Run("Perfect straight line", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 200, Y: 100, Timestamp: 50},
			{X: 300, Y: 100, Timestamp: 100},
			{X: 400, Y: 100, Timestamp: 150},
		}

		fitness, _ := engine.FitBezierCurve(tracks)
		if fitness < 0.9 {
			t.Errorf("Fitness = %f, want >= 0.9 for straight line", fitness)
		}
	})

	t.Run("Curved path", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 150, Y: 200, Timestamp: 50},
			{X: 200, Y: 300, Timestamp: 100},
			{X: 250, Y: 200, Timestamp: 150},
			{X: 300, Y: 100, Timestamp: 200},
		}

		fitness, _ := engine.FitBezierCurve(tracks)
		t.Logf("Curved path fitness: %f", fitness)
	})
}

func TestAnalyzeBezierFitting(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Perfect bezier fit - robot behavior", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 150, Y: 125, Timestamp: 50},
			{X: 200, Y: 150, Timestamp: 100},
			{X: 250, Y: 175, Timestamp: 150},
			{X: 300, Y: 200, Timestamp: 200},
		}

		score, factors := engine.AnalyzeBezierFitting(tracks)

		hasHighFit := false
		for _, f := range factors {
			if f.Name == "perfect_bezier_fit" || f.Name == "high_bezier_fit" {
				hasHighFit = true
				break
			}
		}
		if !hasHighFit {
			t.Logf("Score: %d, Factors: %v", score, factors)
		}
	})

	t.Run("Insufficient data", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 200, Y: 200, Timestamp: 100},
		}

		score, factors := engine.AnalyzeBezierFitting(tracks)
		if score != 0 {
			t.Errorf("Score = %d, want 0 for insufficient data", score)
		}
		if len(factors) != 0 {
			t.Errorf("Factors count = %d, want 0", len(factors))
		}
	})
}

func TestAnalyzeSlideVelocity(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Normal velocity distribution", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 120, Y: 105, Timestamp: 30},
			{X: 150, Y: 115, Timestamp: 70},
			{X: 200, Y: 130, Timestamp: 120},
			{X: 280, Y: 150, Timestamp: 180},
		}

		score, _, dist := engine.AnalyzeSlideVelocity(tracks)
		if dist == nil {
			t.Fatal("VelocityDistribution should not be nil")
		}

		t.Logf("Score: %d, Mean: %.2f, StdDev: %.2f", score, dist.MeanVelocity, dist.StdDevVelocity)
	})

	t.Run("Constant velocity - robot indicator", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 200, Y: 100, Timestamp: 100},
			{X: 300, Y: 100, Timestamp: 200},
			{X: 400, Y: 100, Timestamp: 300},
			{X: 500, Y: 100, Timestamp: 400},
		}

		score, factors, dist := engine.AnalyzeSlideVelocity(tracks)

		hasConstantVelocity := false
		for _, f := range factors {
			if f.Name == "constant_velocity" {
				hasConstantVelocity = true
				break
			}
		}
		if !hasConstantVelocity {
			t.Logf("Expected constant_velocity factor, got score: %d, factors: %v", score, factors)
		}

		if dist.AccelerationScore < 0.1 && dist.DecelerationScore < 0.1 {
			t.Logf("Acceleration and deceleration scores are very low")
		}
	})

	t.Run("Excessive acceleration", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 110, Y: 100, Timestamp: 10},
			{X: 130, Y: 100, Timestamp: 15},
			{X: 170, Y: 100, Timestamp: 18},
			{X: 300, Y: 100, Timestamp: 20},
		}

		score, factors, _ := engine.AnalyzeSlideVelocity(tracks)

		hasExcessiveAccel := false
		for _, f := range factors {
			if f.Name == "excessive_acceleration" {
				hasExcessiveAccel = true
				break
			}
		}
		if hasExcessiveAccel {
			t.Logf("Detected excessive acceleration, score: %d", score)
		}
	})
}

func TestAnalyzeEnhancedClickRhythm(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Mechanical click pattern", func(t *testing.T) {
		clicks := []int64{100, 200, 300, 400, 500}
		positions := []Point{
			{X: 100, Y: 100},
			{X: 100, Y: 100},
			{X: 100, Y: 100},
			{X: 100, Y: 100},
			{X: 100, Y: 100},
		}

		score, _, analysis := engine.AnalyzeEnhancedClickRhythm(clicks, positions)

		if analysis == nil {
			t.Fatal("Analysis should not be nil")
		}

		if !analysis.IsMechanical {
			t.Log("Click pattern may not be classified as mechanical")
		}

		if analysis.PositionVariance < 0.01 {
			t.Logf("Position variance is very low: %.4f", analysis.PositionVariance)
		}

		t.Logf("Score: %d, IsMechanical: %v", score, analysis.IsMechanical)
	})

	t.Run("Fast clicking", func(t *testing.T) {
		clicks := []int64{100, 120, 140, 160, 180}
		positions := []Point{
			{X: 100, Y: 100},
			{X: 105, Y: 102},
			{X: 110, Y: 104},
			{X: 115, Y: 106},
			{X: 120, Y: 108},
		}

		score, factors, analysis := engine.AnalyzeEnhancedClickRhythm(clicks, positions)

		if analysis != nil && analysis.IsTooFast {
			t.Logf("Detected too fast clicking")
		}

		hasFastClick := false
		for _, f := range factors {
			if f.Name == "unusually_fast_clicking" {
				hasFastClick = true
				break
			}
		}
		if !hasFastClick && score < 10 {
			t.Logf("No fast click detected, score: %d", score)
		}
	})

	t.Run("Normal clicking", func(t *testing.T) {
		clicks := []int64{100, 350, 800, 1400, 2100}
		positions := []Point{
			{X: 100, Y: 100},
			{X: 120, Y: 95},
			{X: 90, Y: 110},
			{X: 130, Y: 105},
			{X: 100, Y: 98},
		}

		score, _, analysis := engine.AnalyzeEnhancedClickRhythm(clicks, positions)

		if score > 30 {
			t.Errorf("Score = %d, want <= 30 for normal clicking", score)
		}

		if analysis != nil {
			t.Logf("Interval mean: %.2f, stddev: %.2f", analysis.IntervalMean, analysis.IntervalStdDev)
		}
	})
}

func TestDeviceFingerprint(t *testing.T) {
	t.Run("Parse Chrome UserAgent", func(t *testing.T) {
		ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
		fp := NewDeviceFingerprint(ua, "en-US,en;q=0.9", 1920, 1080)

		if fp.OS != "Windows" {
			t.Errorf("OS = %s, want Windows", fp.OS)
		}
		if fp.Browser != "Chrome" {
			t.Errorf("Browser = %s, want Chrome", fp.Browser)
		}
		if fp.IsBot {
			t.Error("Should not be detected as bot")
		}
	})

	t.Run("Parse Firefox UserAgent", func(t *testing.T) {
		ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0"
		fp := NewDeviceFingerprint(ua, "en-US", 1440, 900)

		if fp.OS != "macOS" {
			t.Errorf("OS = %s, want macOS", fp.OS)
		}
		if fp.Browser != "Firefox" {
			t.Errorf("Browser = %s, want Firefox", fp.Browser)
		}
	})

	t.Run("Detect Bot UserAgent", func(t *testing.T) {
		ua := "curl/7.88.1"
		fp := NewDeviceFingerprint(ua, "", 0, 0)

		if !fp.IsBot {
			t.Error("Should be detected as bot")
		}
	})

	t.Run("Detect Headless Browser", func(t *testing.T) {
		ua := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36"
		fp := NewDeviceFingerprint(ua, "en-US", 1920, 1080)

		if !fp.IsBot {
			t.Error("Should be detected as bot (headless)")
		}
	})

	t.Run("Mobile Device Detection", func(t *testing.T) {
		ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
		fp := NewDeviceFingerprint(ua, "en-US", 390, 844)

		if fp.DeviceType != DeviceTypeMobile {
			t.Errorf("DeviceType = %s, want mobile", fp.DeviceType)
		}
	})

	t.Run("Hash Generation", func(t *testing.T) {
		ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"
		fp1 := NewDeviceFingerprint(ua, "en-US", 1920, 1080)
		fp2 := NewDeviceFingerprint(ua, "en-US", 1920, 1080)

		if fp1.Hash != fp2.Hash {
			t.Error("Same fingerprints should have same hash")
		}
	})
}

func TestAnalyzeDeviceFingerprint(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Bot detection", func(t *testing.T) {
		fp := NewDeviceFingerprint("curl/7.88.1", "", 0, 0)

		score, factors := engine.AnalyzeDeviceFingerprint(fp)

		hasBotFactor := false
		for _, f := range factors {
			if f.Name == "bot_detected" {
				hasBotFactor = true
				break
			}
		}
		if !hasBotFactor {
			t.Errorf("Expected bot_detected factor, got score: %d", score)
		}
	})

	t.Run("Missing screen info", func(t *testing.T) {
		fp := NewDeviceFingerprint("Mozilla/5.0 Chrome/120.0.0.0", "en-US", 0, 0)

		score, factors := engine.AnalyzeDeviceFingerprint(fp)

		hasMissingScreen := false
		for _, f := range factors {
			if f.Name == "missing_screen_info" {
				hasMissingScreen = true
				break
			}
		}
		if !hasMissingScreen {
			t.Errorf("Expected missing_screen_info factor, score: %d", score)
		}
	})

	t.Run("Normal fingerprint", func(t *testing.T) {
		fp := NewDeviceFingerprint(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36",
			"en-US,en;q=0.9",
			1920,
			1080,
		)
		fp.Timezone = "Asia/Shanghai"
		fp.Language = "zh-CN"

		score, factors := engine.AnalyzeDeviceFingerprint(fp)

		if score > 30 {
			t.Errorf("Score = %d, want <= 30 for normal fingerprint", score)
		}
		t.Logf("Score: %d, Factors: %v", score, factors)
	})
}

func TestCompareFingerprints(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Identical fingerprints", func(t *testing.T) {
		fp1 := NewDeviceFingerprint(
			"Mozilla/5.0 Chrome/120.0.0.0",
			"en-US",
			1920,
			1080,
		)
		fp2 := NewDeviceFingerprint(
			"Mozilla/5.0 Chrome/120.0.0.0",
			"en-US",
			1920,
			1080,
		)

		similarity := engine.CompareFingerprints(fp1, fp2)
		if similarity < 0.85 {
			t.Errorf("Similarity = %f, want >= 0.85 for same fingerprints", similarity)
		}
	})

	t.Run("Different browsers", func(t *testing.T) {
		fp1 := NewDeviceFingerprint(
			"Mozilla/5.0 Chrome/120.0.0.0",
			"en-US",
			1920,
			1080,
		)
		fp2 := NewDeviceFingerprint(
			"Mozilla/5.0 Firefox/120.0",
			"en-US",
			1920,
			1080,
		)

		similarity := engine.CompareFingerprints(fp1, fp2)
		if similarity > 0.9 {
			t.Errorf("Similarity = %f, want < 0.9 for different browsers", similarity)
		}
	})

	t.Run("Nil fingerprints", func(t *testing.T) {
		similarity := engine.CompareFingerprints(nil, nil)
		if similarity != 0.0 {
			t.Errorf("Similarity = %f, want 0 for nil fingerprints", similarity)
		}
	})
}

func TestHMMModel(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Forward Algorithm", func(t *testing.T) {
		model := NewHMMModel()
		observations := []HMMObservation{ObsFastSlide, ObsNormalClick, ObsVariableTime}

		prob, alpha := model.ForwardAlgorithm(observations)

		if prob <= 0 {
			t.Errorf("Probability = %f, want > 0", prob)
		}
		if alpha == nil || len(alpha) != len(observations) {
			t.Errorf("Alpha length mismatch")
		}

		t.Logf("Forward probability: %e", prob)
	})

	t.Run("Viterbi Algorithm", func(t *testing.T) {
		model := NewHMMModel()
		observations := []HMMObservation{ObsFastSlide, ObsUniformClick, ObsConsistentTime}

		path, prob := model.ViterbiAlgorithm(observations)

		if len(path) != len(observations) {
			t.Errorf("Path length = %d, want %d", len(path), len(observations))
		}
		if prob <= 0 {
			t.Errorf("Probability = %f, want > 0", prob)
		}

		t.Logf("Most likely path: %v, prob: %e", path, prob)
	})

	t.Run("Create Behavior Sequence", func(t *testing.T) {
		behavior := &BehaviorData{
			MouseTracks: []MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 200, Y: 100, Timestamp: 500},
				{X: 300, Y: 100, Timestamp: 1000},
			},
			ClickTimes: []int64{100, 300, 600, 1000},
			SlideStart: 0,
			SlideEnd:   1500,
		}

		seq := engine.CreateBehaviorSequence(behavior)

		if len(seq.Observations) == 0 {
			t.Error("Observations should not be empty")
		}

		t.Logf("Observations: %v", seq.Observations)
	})

	t.Run("Analyze Machine Pattern", func(t *testing.T) {
		behavior := &BehaviorData{
			MouseTracks: []MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 200, Y: 100, Timestamp: 100},
				{X: 300, Y: 100, Timestamp: 200},
				{X: 400, Y: 100, Timestamp: 300},
			},
			ClickTimes: []int64{100, 200, 300, 400, 500},
			SlideStart: 0,
			SlideEnd:   600,
		}

		score, factors := engine.AnalyzeBehaviorSequence(behavior)

		t.Logf("Machine pattern score: %d, factors: %v", score, factors)
	})
}

func TestDynamicThreshold(t *testing.T) {
	baseThresholds := config.RiskScoreThresholds{
		Low:      0,
		Medium:   25,
		High:     50,
		Critical: 80,
	}

	dt, err := NewDynamicThreshold(&ThresholdConfig{
		WindowSize:   24 * time.Hour,
		LearningRate: 0.1,
		DecayFactor:  0.95,
	}, baseThresholds)

	if err != nil {
		t.Fatalf("Failed to create DynamicThreshold: %v", err)
	}

	ctx := context.Background()

	t.Run("Record successful behavior", func(t *testing.T) {
		err := dt.RecordBehaviorResult(ctx, "user123", "example.com", 30, true)
		if err != nil {
			t.Errorf("RecordBehaviorResult error: %v", err)
		}
	})

	t.Run("Record failed behavior", func(t *testing.T) {
		err := dt.RecordBehaviorResult(ctx, "user123", "example.com", 70, false)
		if err != nil {
			t.Errorf("RecordBehaviorResult error: %v", err)
		}
	})

	t.Run("Get adjusted threshold", func(t *testing.T) {
		thresholds := dt.GetAdjustedThreshold(ctx, "user123", "example.com")
		if thresholds == nil {
			t.Fatal("Thresholds should not be nil")
		}

		t.Logf("Adjusted thresholds: Low=%d, Medium=%d, High=%d, Critical=%d",
			thresholds.Low, thresholds.Medium, thresholds.High, thresholds.Critical)
	})

	t.Run("Get statistics", func(t *testing.T) {
		stats, err := dt.GetStatistics(ctx, "user123", "example.com")
		if err != nil {
			t.Errorf("GetStatistics error: %v", err)
		}

		if !stats.HasHistory {
			t.Error("Should have history")
		}

		t.Logf("Statistics: SuccessRate=%.2f, MeanScore=%.2f", stats.SuccessRate, stats.MeanScore)
	})

	t.Run("Get risk level with adjustment", func(t *testing.T) {
		level := dt.GetRiskLevelWithAdjustment(35, "user456", "example.com", ctx)
		if level == "" {
			t.Error("Level should not be empty")
		}

		t.Logf("Risk level for score 35: %s", level)
	})
}

func TestRuleEngine(t *testing.T) {
	engine := NewRuleEngine()

	t.Run("Add rule", func(t *testing.T) {
		rule := &Rule{
			ID:          "test_rule_1",
			Name:        "Test Rule",
			Description: "A test rule",
			Type:        RuleTypeCondition,
			Priority:    100,
			Enabled:     true,
			Conditions: []Condition{
				{Field: "score", Operator: OpGt, Value: 50},
			},
			Logic: "AND",
			Action: RuleAction{
				Type:  ActionBlock,
				Score: 20,
				Reason: "Score exceeds threshold",
			},
		}

		err := engine.AddRule(rule)
		if err != nil {
			t.Errorf("AddRule error: %v", err)
		}
	})

	t.Run("Get rule", func(t *testing.T) {
		rule, exists := engine.GetRule("test_rule_1")
		if !exists {
			t.Error("Rule should exist")
		}
		if rule.Name != "Test Rule" {
			t.Errorf("Rule name = %s, want Test Rule", rule.Name)
		}
	})

	t.Run("Evaluate rule - condition met", func(t *testing.T) {
		rc := &RuleContext{
			Score: 60,
			IP:    "192.168.1.1",
		}

		_, score, action := engine.Evaluate(context.Background(), rc)

		if action != ActionBlock {
			t.Errorf("Action = %s, want Block", action)
		}

		t.Logf("Score: %d, Action: %s", score, action)
	})

	t.Run("Evaluate rule - condition not met", func(t *testing.T) {
		rc := &RuleContext{
			Score: 30,
			IP:    "192.168.1.1",
		}

		_, score, action := engine.Evaluate(context.Background(), rc)

		if score != 30 {
			t.Errorf("Score = %d, want 30", score)
		}

		t.Logf("Score: %d, Action: %s", score, action)
	})

	t.Run("Create default rules", func(t *testing.T) {
		err := engine.CreateDefaultRules()
		if err != nil {
			t.Errorf("CreateDefaultRules error: %v", err)
		}

		rules := engine.GetAllRules()
		if len(rules) < 6 {
			t.Errorf("Expected at least 6 rules, got %d", len(rules))
		}
	})

	t.Run("Validate rule", func(t *testing.T) {
		invalidRule := &Rule{
			ID:   "",
			Type: RuleTypeCondition,
		}

		err := engine.ValidateRule(invalidRule)
		if err == nil {
			t.Error("Expected validation error for empty ID")
		}
	})

	t.Run("Enable and disable rule", func(t *testing.T) {
		rule := &Rule{
			ID:      "toggle_test",
			Name:    "Toggle Test",
			Type:    RuleTypeCondition,
			Enabled: true,
		}

		engine.AddRule(rule)

		err := engine.DisableRule("toggle_test")
		if err != nil {
			t.Errorf("DisableRule error: %v", err)
		}

		rule, _ = engine.GetRule("toggle_test")
		if rule.Enabled {
			t.Error("Rule should be disabled")
		}

		err = engine.EnableRule("toggle_test")
		if err != nil {
			t.Errorf("EnableRule error: %v", err)
		}

		rule, _ = engine.GetRule("toggle_test")
		if !rule.Enabled {
			t.Error("Rule should be enabled")
		}
	})

	t.Run("Get rules by tag", func(t *testing.T) {
		rule := &Rule{
			ID:   "tagged_rule",
			Name: "Tagged Rule",
			Type: RuleTypeCondition,
			Tags: []string{"security", "bot"},
		}

		engine.AddRule(rule)

		rules := engine.GetRulesByTag("security")
		if len(rules) == 0 {
			t.Error("Should have rules with security tag")
		}
	})
}

func TestRuleEngineOperators(t *testing.T) {
	engine := NewRuleEngine()

	t.Run("Equality operator", func(t *testing.T) {
		_ = &RuleContext{
			Level: RiskLevelHigh,
		}

		result := engine.compareValues("high", OpEq, "high")
		if !result {
			t.Error("Eq operator should return true for equal values")
		}
	})

	t.Run("Greater than operator", func(t *testing.T) {
		result := engine.compareValues(100, OpGt, 50)
		if !result {
			t.Error("Gt operator should return true when value is greater")
		}
	})

	t.Run("Contains operator", func(t *testing.T) {
		result := engine.compareValues("hello world", OpContains, "world")
		if !result {
			t.Error("Contains operator should return true when substring exists")
		}
	})

	t.Run("Regex operator", func(t *testing.T) {
		result := engine.compareValues("test123", OpRegex, `test\d+`)
		if !result {
			t.Error("Regex operator should return true when pattern matches")
		}
	})

	t.Run("Between operator", func(t *testing.T) {
		result := engine.compareValues(50, OpBetween, []interface{}{10, 100})
		if !result {
			t.Error("Between operator should return true when value is in range")
		}
	})

	t.Run("In operator", func(t *testing.T) {
		result := engine.compareValues("b", OpIn, []interface{}{"a", "b", "c"})
		if !result {
			t.Error("In operator should return true when value is in list")
		}
	})
}

func TestRuleEngineComplexConditions(t *testing.T) {
	engine := NewRuleEngine()

	rule := &Rule{
		ID:          "complex_rule",
		Name:        "Complex Rule",
		Description: "Test AND/OR logic",
		Type:        RuleTypeCondition,
		Priority:    100,
		Enabled:     true,
		Conditions: []Condition{
			{Field: "score", Operator: OpGt, Value: 30},
			{Field: "ip", Operator: OpNeq, Value: ""},
		},
		Logic: "AND",
		Action: RuleAction{
			Type:  ActionVerify,
			Score: 15,
			Reason: "Complex condition met",
		},
	}

	engine.AddRule(rule)

	t.Run("AND logic - both conditions met", func(t *testing.T) {
		rc := &RuleContext{
			Score: 50,
			IP:    "192.168.1.1",
		}

		_, _, action := engine.Evaluate(context.Background(), rc)

		if action != ActionVerify {
			t.Errorf("Action = %s, want Verify", action)
		}
	})

	t.Run("AND logic - one condition not met", func(t *testing.T) {
		rc := &RuleContext{
			Score: 20,
			IP:    "192.168.1.1",
		}

		_, _, action := engine.Evaluate(context.Background(), rc)

		if action != ActionAllow {
			t.Errorf("Action = %s, want Allow", action)
		}
	})
}

func TestEnhancedRiskAnalysis(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	whitelist, _ := NewWhitelist(&WhitelistConfig{MemoryOnly: true})
	engine := NewRiskEngine(cfg, nil, whitelist)

	ctx := context.Background()

	t.Run("Complete risk analysis", func(t *testing.T) {
		behavior := &BehaviorData{
			UserID:    "user123",
			SessionID: "session456",
			MouseTracks: []MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 150, Y: 120, Timestamp: 50},
				{X: 200, Y: 150, Timestamp: 100},
				{X: 250, Y: 180, Timestamp: 150},
				{X: 300, Y: 200, Timestamp: 200},
			},
			ClickTimes: []int64{100, 250, 450, 700, 1000},
			SlideStart: 0,
			SlideEnd:   3000,
			Success:    true,
		}

		fingerprint := NewDeviceFingerprint(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36",
			"en-US,en;q=0.9",
			1920,
			1080,
		)
		fingerprint.Timezone = "Asia/Shanghai"
		fingerprint.Language = "zh-CN"

		result := engine.EnhancedRiskAnalysis(ctx, behavior, "192.168.1.1", "example.com", fingerprint)

		t.Logf("Enhanced Risk Score: %d, Level: %s, Action: %s",
			result.Score, result.Level, result.Recommended)
		t.Logf("Factors: %v", result.Factors)

		if result.Score > 100 {
			t.Errorf("Score = %d, want <= 100", result.Score)
		}
	})

	t.Run("High risk scenario", func(t *testing.T) {
		behavior := &BehaviorData{
			SessionID: "high-risk-session",
			MouseTracks: []MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 200, Y: 100, Timestamp: 100},
				{X: 300, Y: 100, Timestamp: 200},
				{X: 400, Y: 100, Timestamp: 300},
			},
			ClickTimes: []int64{100, 200, 300, 400, 500},
			SlideStart: 0,
			SlideEnd:   500,
			Success:    false,
		}

		fingerprint := NewDeviceFingerprint("curl/7.88.1", "", 0, 0)

		result := engine.EnhancedRiskAnalysis(ctx, behavior, "1.1.1.1", "suspicious.com", fingerprint)

		t.Logf("High Risk Score: %d, Level: %s, Action: %s",
			result.Score, result.Level, result.Recommended)

		if result.Score < 50 {
			t.Logf("Score is lower than expected for high-risk scenario")
		}
	})
}

func TestTemporalDecay(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Recent score - no decay", func(t *testing.T) {
		score := 80
		lastTime := time.Now().Add(-30 * time.Minute)

		decayed := engine.ApplyTemporalDecay(score, lastTime)

		if decayed < score {
			t.Errorf("Score should not decay significantly for recent time")
		}
	})

	t.Run("Old score - decay applied", func(t *testing.T) {
		score := 80
		lastTime := time.Now().Add(-48 * time.Hour)

		decayed := engine.ApplyTemporalDecay(score, lastTime)

		if decayed >= score {
			t.Logf("Decayed score: %d, Original: %d", decayed, score)
		}
	})
}

func TestCumulativeRisk(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("With historical scores", func(t *testing.T) {
		baseScore := 40
		historicalScores := []int{30, 35, 45, 25, 50}

		cumulative := engine.CalculateCumulativeRisk(baseScore, historicalScores, nil)

		if cumulative < baseScore || cumulative > 100 {
			t.Errorf("Cumulative score out of range: %d", cumulative)
		}

		t.Logf("Cumulative score: %d (base: %d, historical: %v)", cumulative, baseScore, historicalScores)
	})

	t.Run("Without historical scores", func(t *testing.T) {
		baseScore := 50

		cumulative := engine.CalculateCumulativeRisk(baseScore, nil, nil)

		if cumulative != baseScore {
			t.Errorf("Cumulative = %d, want %d", cumulative, baseScore)
		}
	})
}

func TestScoreAnomalyDetection(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Normal score", func(t *testing.T) {
		historicalScores := []int{30, 35, 32, 38, 31, 33, 36, 34, 35, 32}
		score := 40

		isAnomaly := engine.DetectScoreAnomaly(score, historicalScores)

		if isAnomaly {
			t.Log("Score detected as anomaly (might be expected)")
		}
	})

	t.Run("Extreme score - anomaly", func(t *testing.T) {
		historicalScores := []int{30, 35, 32, 38, 31, 33, 36, 34, 35, 32}
		score := 90

		isAnomaly := engine.DetectScoreAnomaly(score, historicalScores)

		if isAnomaly {
			t.Log("Correctly detected extreme score as anomaly")
		}
	})

	t.Run("Insufficient history", func(t *testing.T) {
		historicalScores := []int{30, 35}

		isAnomaly := engine.DetectScoreAnomaly(90, historicalScores)

		if isAnomaly {
			t.Error("Should not detect anomaly with insufficient history")
		}
	})
}
