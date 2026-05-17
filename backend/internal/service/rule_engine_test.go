package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEnhancedRuleEngine(t *testing.T) {
	engine := NewEnhancedRuleEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.rules)
	assert.NotNil(t, engine.ruleMap)
	assert.NotNil(t, engine.categories)
	assert.NotNil(t, engine.weights)
	assert.NotNil(t, engine.performanceTracker)
}

func TestEnhancedRuleEngine_Initialization(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	assert.Greater(t, len(engine.rules), 10)
	assert.Greater(t, len(engine.categories), 0)
	assert.Equal(t, 0.5, engine.threshold)

	for category := range engine.categories {
		assert.NotEmpty(t, engine.categories[category])
	}
}

func TestEnhancedRuleEngine_Evaluate_Basic(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.85,
		SpeedConsistency:   0.6,
		AverageSpeed:       500,
		MaxSpeed:           800,
		SpeedVariance:      0.1,
		CurvatureAverage:   0.1,
		DirectionChanges:   5,
		MicroCorrections:   3,
		BacktrackCount:     1,
		PauseCount:         2,
		HumanLikenessScore: 0.7,
	}

	result := engine.Evaluate(features)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.TotalScore, 0.0)
	assert.LessOrEqual(t, result.TotalScore, 1.0)
	assert.Greater(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)
}

func TestEnhancedRuleEngine_Evaluate_NilFeatures(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	result := engine.Evaluate(nil)
	assert.NotNil(t, result)
	assert.False(t, result.IsBot)
	assert.Equal(t, 0.0, result.Confidence)
	assert.Equal(t, "unknown", result.RiskLevel)
}

func TestEnhancedRuleEngine_Evaluate_HighRisk(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.99,
		SpeedConsistency:   0.99,
		AverageSpeed:       2500,
		MaxSpeed:           3500,
		SpeedVariance:      0.001,
		CurvatureAverage:   0.001,
		DirectionChanges:   0,
		MicroCorrections:   0,
		BacktrackCount:     0,
		PauseCount:         0,
		HumanLikenessScore: 0.05,
		AnomalyScore:       0.9,
		MLScore:            0.85,
		FractalDimension:   1.05,
	}

	result := engine.Evaluate(features)
	assert.NotNil(t, result)
	assert.True(t, result.IsBot)
	assert.Greater(t, result.TotalScore, 0.5)
	assert.GreaterOrEqual(t, len(result.TriggeredRules), 3)
}

func TestEnhancedRuleEngine_Evaluate_TriggeredRules(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.99,
		SpeedConsistency:   0.99,
		AverageSpeed:       2500,
		HumanLikenessScore: 0.05,
	}

	result := engine.Evaluate(features)

	assert.NotEmpty(t, result.TriggeredRules)
	assert.Contains(t, result.TriggeredRules, "perfect_path_efficiency")
	assert.Contains(t, result.TriggeredRules, "extreme_speed")
}

func TestEnhancedRuleEngine_Evaluate_CategoryScores(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.99,
		AverageSpeed:       2500,
		HumanLikenessScore: 0.1,
	}

	result := engine.Evaluate(features)

	assert.NotNil(t, result.CategoryScores)
	assert.Greater(t, len(result.CategoryScores), 0)
}

func TestEnhancedRuleEngine_RuleCategories(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	expectedCategories := []string{"speed", "trajectory", "behavior", "click", "accuracy", "general", "ml", "combined"}

	for _, category := range expectedCategories {
		rules, ok := engine.categories[category]
		assert.True(t, ok, fmt.Sprintf("Category %s should exist", category))
		assert.Greater(t, len(rules), 0, fmt.Sprintf("Category %s should have rules", category))
	}
}

func TestEnhancedRuleEngine_RulePriorities(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	for _, rule := range engine.rules {
		assert.Greater(t, rule.Priority, 0)
		assert.Greater(t, rule.Weight, 0.0)
		assert.Greater(t, rule.Severity, 0.0)
		assert.LessOrEqual(t, rule.Severity, 1.0)
	}
}

func TestEnhancedRuleEngine_Weights(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	totalWeight := 0.0
	for _, weight := range engine.weights {
		totalWeight += weight
	}

	assert.GreaterOrEqual(t, totalWeight, 0.9)
	assert.LessOrEqual(t, totalWeight, 1.1)
}

func TestEnhancedRuleEngine_AddRule(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	initialCount := len(engine.rules)

	newRule := EnhancedRule{
		Name:        "custom_rule",
		Condition:   func(f *RuleEngineFeatures) bool { return f.AverageSpeed > 1000 },
		Weight:      20,
		Priority:    1,
		Description: "Custom rule",
		Category:    "speed",
		Severity:    0.8,
		Enabled:     true,
	}

	engine.AddRule(newRule)

	assert.Equal(t, initialCount+1, len(engine.rules))
	assert.Contains(t, engine.ruleMap, "custom_rule")
}

func TestEnhancedRuleEngine_AddRule_Update(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	initialCount := len(engine.rules)

	rule := EnhancedRule{
		Name:        "custom_test_rule",
		Condition:   func(f *RuleEngineFeatures) bool { return false },
		Weight:      10,
		Priority:    1,
		Description: "Test",
		Category:    "speed",
		Severity:    0.5,
		Enabled:     true,
	}

	engine.AddRule(rule)

	updatedRule := rule
	updatedRule.Weight = 50
	engine.AddRule(updatedRule)

	assert.Equal(t, 50.0, engine.ruleMap["custom_test_rule"].Weight)
	assert.Equal(t, initialCount+1, len(engine.rules))
}

func TestEnhancedRuleEngine_RemoveRule(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	initialCount := len(engine.rules)

	rule := EnhancedRule{
		Name:        "to_remove",
		Condition:   func(f *RuleEngineFeatures) bool { return false },
		Weight:      10,
		Priority:    1,
		Description: "To remove",
		Category:    "speed",
		Severity:    0.5,
		Enabled:     true,
	}

	engine.AddRule(rule)
	engine.RemoveRule("to_remove")

	assert.Equal(t, initialCount, len(engine.rules))
	assert.NotContains(t, engine.ruleMap, "to_remove")
}

func TestEnhancedRuleEngine_EnableRule(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	engine.DisableRule("extreme_speed")
	assert.False(t, engine.ruleMap["extreme_speed"].Enabled)

	engine.EnableRule("extreme_speed")
	assert.True(t, engine.ruleMap["extreme_speed"].Enabled)
}

func TestEnhancedRuleEngine_DisableRule(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	engine.DisableRule("extreme_speed")
	assert.False(t, engine.ruleMap["extreme_speed"].Enabled)
}

func TestEnhancedRuleEngine_SetThreshold(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	engine.SetThreshold(0.7)
	assert.Equal(t, 0.7, engine.threshold)

	engine.SetThreshold(1.5)
	assert.Equal(t, 1.0, engine.threshold)

	engine.SetThreshold(-0.5)
	assert.Equal(t, 0.0, engine.threshold)
}

func TestEnhancedRuleEngine_SetCategoryWeight(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	engine.SetCategoryWeight("speed", 0.5)
	assert.Equal(t, 0.5, engine.weights["speed"])

	engine.SetCategoryWeight("speed", 1.5)
	assert.Equal(t, 1.0, engine.weights["speed"])

	engine.SetCategoryWeight("speed", -0.1)
	assert.Equal(t, 0.0, engine.weights["speed"])
}

func TestEnhancedRuleEngine_GetTriggeredRules(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.99,
		AverageSpeed:   2500,
	}

	triggered := engine.GetTriggeredRules(features)
	assert.NotEmpty(t, triggered)
}

func TestEnhancedRuleEngine_GetRulesByCategory(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	speedRules := engine.GetRulesByCategory("speed")
	assert.Greater(t, len(speedRules), 0)

	for _, rule := range speedRules {
		assert.Equal(t, "speed", rule.Category)
	}
}

func TestEnhancedRuleEngine_GetAllRules(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	rules := engine.GetAllRules()
	assert.Equal(t, len(engine.rules), len(rules))
}

func TestEnhancedRuleEngine_GetCategories(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	categories := engine.GetCategories()
	assert.Greater(t, len(categories), 0)
}

func TestEnhancedRuleEngine_GetPerformanceStats(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.5,
		AverageSpeed:   500,
	}

	engine.Evaluate(features)

	stats := engine.GetPerformanceStats()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1), stats["evaluation_count"])
}

func TestEnhancedRuleEngine_ResetPerformanceStats(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.5,
		AverageSpeed:   500,
	}

	engine.Evaluate(features)
	engine.ResetPerformanceStats()

	stats := engine.GetPerformanceStats()
	assert.Equal(t, int64(0), stats["evaluation_count"])
}

func TestEnhancedRuleEngine_ExportRules(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	export := engine.ExportRules()
	assert.NotEmpty(t, export)
	assert.Contains(t, export, "=== 增强版规则引擎导出 ===")
	assert.Contains(t, export, "阈值")
	assert.Contains(t, export, "规则总数")
}

func TestEnhancedRuleEngine_AnalyzeTime(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.5,
		AverageSpeed:   500,
	}

	result := engine.Evaluate(features)
	assert.Greater(t, result.AnalysisTime, 0*time.Millisecond)
}

func TestRuleEngineConfidence(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	tests := []struct {
		name          string
		result        *RuleEngineResult
		minConfidence float64
	}{
		{
			name: "high confidence",
			result: &RuleEngineResult{
				TriggeredRules: make([]string, 15),
				CategoryScores: map[string]float64{
					"speed":      0.6,
					"trajectory": 0.7,
					"behavior":   0.5,
				},
			},
			minConfidence: 0.85,
		},
		{
			name: "low confidence",
			result: &RuleEngineResult{
				TriggeredRules: make([]string, 2),
				CategoryScores: map[string]float64{
					"speed": 0.6,
				},
			},
			minConfidence: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := engine.calculateConfidence(tt.result)
			assert.GreaterOrEqual(t, confidence, tt.minConfidence)
			assert.LessOrEqual(t, confidence, 0.99)
		})
	}
}

func TestClassifyRiskLevel(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	tests := []struct {
		score float64
		level string
	}{
		{0.9, "critical"},
		{0.8, "critical"},
		{0.7, "high"},
		{0.6, "high"},
		{0.5, "medium"},
		{0.4, "medium"},
		{0.3, "low"},
		{0.2, "low"},
		{0.1, "minimal"},
		{0.0, "minimal"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("score_%f", tt.score), func(t *testing.T) {
			level := engine.classifyRiskLevel(tt.score)
			assert.Equal(t, tt.level, level)
		})
	}
}

func TestRuleEngineRecommendations(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	tests := []struct {
		name               string
		result             *RuleEngineResult
		minRecommendations int
	}{
		{
			name: "high risk",
			result: &RuleEngineResult{
				TotalScore: 0.8,
				CategoryScores: map[string]float64{
					"speed":      0.7,
					"trajectory": 0.7,
					"click":      0.6,
				},
				TriggeredRules: make([]string, 20),
			},
			minRecommendations: 3,
		},
		{
			name: "low risk",
			result: &RuleEngineResult{
				TotalScore: 0.2,
				CategoryScores: map[string]float64{
					"speed": 0.2,
				},
				TriggeredRules: make([]string, 2),
			},
			minRecommendations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := engine.generateRecommendations(tt.result)
			assert.GreaterOrEqual(t, len(recommendations), tt.minRecommendations)
		})
	}
}

func TestCreateEnhancedFeaturesFromSliderResult(t *testing.T) {
	sliderResult := &SliderAnalysisResult{
		Trajectory: &SliderTrajectory{
			PathEfficiency: 0.85,
			AverageSpeed:   500,
			MaxSpeed:       800,
		},
		Features: &SliderFeatures{
			SpeedConsistency:   0.6,
			CurvatureAverage:   0.1,
			DirectionChanges:   5,
			MicroCorrections:   3,
			BacktrackCount:     1,
			PauseCount:         2,
			TotalPauseDuration: 200,
			ResponseTime:       3000,
			JitterScore:        0.1,
			SmoothnessScore:    0.7,
			HumanLikenessScore: 0.65,
			FractalDimension:   1.3,
			FourierFrequency:   1.5,
		},
		AnomalyScore: 0.3,
		MLScore:      0.4,
		IsBot:        false,
	}

	features := CreateEnhancedFeaturesFromSliderResult(sliderResult)
	assert.NotNil(t, features)
	assert.Equal(t, sliderResult.Trajectory.PathEfficiency, features.PathEfficiency)
	assert.Equal(t, sliderResult.Trajectory.AverageSpeed, features.AverageSpeed)
	assert.Equal(t, sliderResult.Features.HumanLikenessScore, features.HumanLikenessScore)
}

func TestCreateEnhancedFeaturesFromSliderResult_Nil(t *testing.T) {
	features := CreateEnhancedFeaturesFromSliderResult(nil)
	assert.NotNil(t, features)
}

func TestCreateEnhancedFeaturesFromClickResult(t *testing.T) {
	clickResult := &ClickAnalysisResult{
		ClickPattern: &ClickPatternAnalysis{
			Regularity: 0.85,
			PositionDistribution: &PositionDistribution{
				XEntropy: 3.0,
				YEntropy: 2.8,
			},
			ClusteringScore: 0.5,
		},
		TimingAnalysis: &TimingAnalysis{
			TotalDuration:   2500,
			AverageDuration: 625,
		},
		AccuracyAnalysis: &AccuracyAnalysis{
			Accuracy: 0.8,
		},
		AnomalyScore: 0.3,
		MLScore:      0.4,
		IsBot:        false,
	}

	features := CreateEnhancedFeaturesFromClickResult(clickResult)
	assert.NotNil(t, features)
	assert.Equal(t, clickResult.ClickPattern.Regularity, features.ClickRegularity)
	assert.Equal(t, float64(clickResult.TimingAnalysis.TotalDuration), features.ResponseTime)
}

func TestCreateEnhancedFeaturesFromClickResult_Nil(t *testing.T) {
	features := CreateEnhancedFeaturesFromClickResult(nil)
	assert.NotNil(t, features)
}

func TestCombineEnhancedFeatures(t *testing.T) {
	features1 := &RuleEngineFeatures{
		PathEfficiency:     0.85,
		AverageSpeed:       500,
		HumanLikenessScore: 0.6,
	}

	features2 := &RuleEngineFeatures{
		ClickRegularity: 0.85,
		Accuracy:        0.9,
		AnomalyScore:    0.4,
	}

	combined := CombineEnhancedFeatures(features1, features2)
	assert.NotNil(t, combined)
	assert.Equal(t, features1.PathEfficiency, combined.PathEfficiency)
	assert.Equal(t, features1.AverageSpeed, combined.AverageSpeed)
	assert.Equal(t, features2.ClickRegularity, combined.ClickRegularity)
	assert.Equal(t, features2.Accuracy, combined.Accuracy)
}

func TestCombineEnhancedFeatures_Multiple(t *testing.T) {
	features := []*RuleEngineFeatures{
		{PathEfficiency: 0.85, AverageSpeed: 500},
		{ClickRegularity: 0.9, Accuracy: 0.8},
		{AnomalyScore: 0.3, MLScore: 0.4},
	}

	combined := CombineEnhancedFeatures(features...)
	assert.NotNil(t, combined)
	assert.Equal(t, 0.85, combined.PathEfficiency)
	assert.Equal(t, 500.0, combined.AverageSpeed)
	assert.Equal(t, 0.9, combined.ClickRegularity)
	assert.Equal(t, 0.8, combined.Accuracy)
	assert.Equal(t, 0.3, combined.AnomalyScore)
	assert.Equal(t, 0.4, combined.MLScore)
}

func TestCombineEnhancedFeatures_Empty(t *testing.T) {
	combined := CombineEnhancedFeatures()
	assert.NotNil(t, combined)
}

func TestCombineEnhancedFeatures_NilValues(t *testing.T) {
	features := []*RuleEngineFeatures{
		{PathEfficiency: 0.85, AverageSpeed: 500},
		nil,
		{Accuracy: 0.8},
	}

	combined := CombineEnhancedFeatures(features...)
	assert.NotNil(t, combined)
	assert.Equal(t, 0.85, combined.PathEfficiency)
	assert.Equal(t, 0.8, combined.Accuracy)
}

func TestEnhancedRuleEngine_Performance(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.85,
		SpeedConsistency:   0.6,
		AverageSpeed:       500,
		MaxSpeed:           800,
		HumanLikenessScore: 0.7,
	}

	start := time.Now()
	for i := 0; i < 1000; i++ {
		_ = engine.Evaluate(features)
	}
	duration := time.Since(start)

	t.Logf("1000次评估耗时: %v", duration)
	assert.Less(t, duration, 5*time.Second, "性能测试: 1000次评估应在5秒内完成")
}

func TestEnhancedRuleEngine_RuleConsistency(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.99,
		AverageSpeed:   2500,
	}

	rules1 := engine.GetTriggeredRules(features)
	rules2 := engine.GetTriggeredRules(features)

	assert.Equal(t, len(rules1), len(rules2))
	for i := range rules1 {
		assert.Equal(t, rules1[i], rules2[i])
	}
}

func TestEnhancedRuleEngine_ThresholdEffect(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	highRiskFeatures := &RuleEngineFeatures{
		PathEfficiency:     0.99,
		SpeedConsistency:    0.99,
		AverageSpeed:        2500,
		HumanLikenessScore:  0.05,
	}

	engine.SetThreshold(0.9)
	result1 := engine.Evaluate(highRiskFeatures)

	engine.SetThreshold(0.3)
	result2 := engine.Evaluate(highRiskFeatures)

	assert.InDelta(t, result1.TotalScore, result2.TotalScore, 0.0001)
	assert.NotEqual(t, result1.IsBot, result2.IsBot)
}

func TestEnhancedRuleEngine_CategoryWeightsEffect(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.99,
		AverageSpeed:   2500,
	}

	engine.SetCategoryWeight("speed", 0.8)
	engine.SetCategoryWeight("trajectory", 0.2)
	result1 := engine.Evaluate(features)
	t.Logf("result1 speed score: %f", result1.CategoryScores["speed"])

	engine.SetCategoryWeight("speed", 0.2)
	engine.SetCategoryWeight("trajectory", 0.8)
	result2 := engine.Evaluate(features)
	t.Logf("result2 speed score: %f", result2.CategoryScores["speed"])

	assert.NotNil(t, result1.CategoryScores["speed"])
	assert.NotNil(t, result2.CategoryScores["speed"])
}

func TestEnhancedRuleEngine_DisabledRulesNotTriggered(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	engine.DisableRule("extreme_speed")

	features := &RuleEngineFeatures{
		PathEfficiency: 0.5,
		AverageSpeed:   2500,
	}

	result := engine.Evaluate(features)
	for _, rule := range result.TriggeredRules {
		assert.NotEqual(t, "extreme_speed", rule)
	}
}

func TestEnhancedRuleEngine_EnabledRulesTriggered(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	engine.EnableRule("extreme_speed")

	features := &RuleEngineFeatures{
		PathEfficiency: 0.5,
		AverageSpeed:   2500,
	}

	result := engine.Evaluate(features)
	triggered := false
	for _, rule := range result.TriggeredRules {
		if rule == "extreme_speed" {
			triggered = true
			break
		}
	}
	assert.True(t, triggered)
}

func TestEnhancedRuleEngine_RuleHitCounting(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.99,
		AverageSpeed:   2500,
	}

	engine.Evaluate(features)
	engine.Evaluate(features)
	engine.Evaluate(features)

	stats := engine.GetPerformanceStats()
	hitCounts := stats["rule_hit_counts"].(map[string]int64)

	found := false
	for ruleName, count := range hitCounts {
		if ruleName == "extreme_speed" && count >= 3 {
			found = true
			break
		}
	}
	assert.True(t, found, "extreme_speed rule should be hit at least 3 times")
}

func TestEnhancedRuleEngine_CategoryHitCounting(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency: 0.99,
		AverageSpeed:   2500,
	}

	engine.Evaluate(features)

	stats := engine.GetPerformanceStats()
	hitCounts := stats["category_hit_counts"].(map[string]int64)

	assert.Greater(t, hitCounts["speed"], int64(0))
}

func TestEnhancedRuleEngine_AllRuleConditions(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	for _, rule := range engine.rules {
		features := &RuleEngineFeatures{
			PathEfficiency:     0.85,
			SpeedConsistency:   0.6,
			AverageSpeed:       500,
			MaxSpeed:           800,
			SpeedVariance:      0.1,
			CurvatureAverage:   0.1,
			CurvatureVariance:  0.05,
			DirectionChanges:   5,
			MicroCorrections:   3,
			BacktrackCount:     1,
			PauseCount:         2,
			TotalPauseDuration: 200,
			HesitationTime:     100,
			ResponseTime:       3000,
			ClickRegularity:    0.5,
			PositionEntropy:    3.0,
			Accuracy:           0.7,
			ClusteringScore:    0.5,
			JitterScore:        0.1,
			SmoothnessScore:    0.7,
			HumanLikenessScore: 0.6,
			AnomalyScore:       0.3,
			MLScore:            0.4,
			FractalDimension:   1.3,
			FourierFrequency:   1.5,
		}

		assert.NotPanics(t, func() {
			result := rule.Condition(features)
			_ = result
		}, fmt.Sprintf("Rule %s should not panic", rule.Name))
	}
}

func TestEnhancedRuleEngine_HighAccuracy(t *testing.T) {
func TestEnhancedRuleEngine_Integration(t *testing.T) {
	engine := NewEnhancedRuleEngine()

	sliderAnalyzer := NewSliderAnalyzer()
	humanTrajectory := GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000)
	sliderResult, _ := sliderAnalyzer.AnalyzeSliderTrajectory(humanTrajectory, 500)

	clickAnalyzer := NewClickAnalyzer()
	targets := []TargetImage{
		{X: 100, Y: 200, Width: 50, Height: 50},
		{X: 300, Y: 400, Width: 50, Height: 50},
	}
	clickVerification := &ClickVerification{
		Clicks:       GenerateHumanLikeClickData(targets, 3000),
		TargetImages: targets,
	}
	clickResult := clickAnalyzer.AnalyzeClickVerification(clickVerification)

	sliderFeatures := CreateEnhancedFeaturesFromSliderResult(sliderResult)
	clickFeatures := CreateEnhancedFeaturesFromClickResult(clickResult)

	combinedFeatures := CombineEnhancedFeatures(sliderFeatures, clickFeatures)

	result := engine.Evaluate(combinedFeatures)

	assert.NotNil(t, result)
	assert.Greater(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)

	t.Logf("综合分析风险分数: %.4f", result.TotalScore)
	t.Logf("判定为机器人: %v", result.IsBot)
	t.Logf("触发规则数量: %d", len(result.TriggeredRules))
}

func BenchmarkEnhancedRuleEngine_Evaluate(b *testing.B) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.85,
		SpeedConsistency:   0.6,
		AverageSpeed:       500,
		MaxSpeed:           800,
		HumanLikenessScore: 0.7,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Evaluate(features)
	}
}

func BenchmarkEnhancedRuleEngine_EvaluateHighRisk(b *testing.B) {
	engine := NewEnhancedRuleEngine()

	features := &RuleEngineFeatures{
		PathEfficiency:     0.99,
		SpeedConsistency:   0.99,
		AverageSpeed:       2500,
		MaxSpeed:           3500,
		HumanLikenessScore: 0.05,
		AnomalyScore:       0.85,
		MLScore:            0.8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Evaluate(features)
	}
}
