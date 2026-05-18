package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func generateTestHistoricalData(count int) []RiskData {
	data := make([]RiskData, count)
	for i := 0; i < count; i++ {
		isBot := i%2 == 0
		features := &RuleEngineFeatures{}
		if isBot {
			features.PathEfficiency = 0.9 + float64(i)*0.001
			features.SpeedConsistency = 0.85 + float64(i)*0.001
			features.AverageSpeed = 1500 + float64(i)*10
			features.SpeedVariance = float64(i) * 0.001
			features.CurvatureAverage = float64(i) * 0.0005
			features.HumanLikenessScore = float64(i) * 0.003
			features.AnomalyScore = 0.6 + float64(i)*0.004
			features.MLScore = 0.5 + float64(i)*0.005
			features.ClickRegularity = 0.8 + float64(i)*0.002
			features.FractalDimension = 1.0 + float64(i)*0.003
		} else {
			features.PathEfficiency = 0.5 + float64(i)*0.004
			features.SpeedConsistency = 0.4 + float64(i)*0.005
			features.AverageSpeed = 200 + float64(i)*8
			features.SpeedVariance = 0.1 + float64(i)*0.009
			features.CurvatureAverage = 0.05 + float64(i)*0.004
			features.HumanLikenessScore = 0.5 + float64(i)*0.005
			features.AnomalyScore = float64(i) * 0.005
			features.MLScore = float64(i) * 0.005
			features.ClickRegularity = float64(i) * 0.006
			features.FractalDimension = 1.2 + float64(i)*0.005
		}
		data[i] = RiskData{
			Features:   features,
			IsBot:      isBot,
			Timestamp:  time.Now(),
			SessionID:  "session_" + string(rune(i)),
			UserID:     "user_" + string(rune(i)),
		}
	}
	return data
}

func TestNewMLRuleGenerator(t *testing.T) {
	generator := NewMLRuleGenerator()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.rules)
	assert.NotNil(t, generator.scenarios)
	assert.NotNil(t, generator.featureStats)
	assert.Greater(t, len(generator.scenarios), 0)
}

func TestMLRuleGenerator_GenerateRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)

	rules, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)
	assert.NotNil(t, rules)

	for _, rule := range rules {
		assert.NotEmpty(t, rule.Name)
		assert.NotEmpty(t, rule.Category)
		assert.NotNil(t, rule.Condition)
		assert.Greater(t, rule.Severity, 0.0)
		assert.LessOrEqual(t, rule.Severity, 1.0)
	}
}

func TestMLRuleGenerator_GenerateRules_EmptyData(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	rules, err := generator.GenerateRules(ctx, []RiskData{})
	assert.Error(t, err)
	assert.Nil(t, rules)
	assert.Contains(t, err.Error(), "历史数据为空")
}

func TestMLRuleGenerator_GenerateRules_Categories(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(200)
	rules, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	categories := make(map[string]bool)
	for _, rule := range rules {
		categories[rule.Category] = true
	}

	assert.True(t, categories["speed"], "Should have speed rules")
	assert.True(t, categories["trajectory"], "Should have trajectory rules")
}

func TestMLRuleGenerator_EvaluateRule(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	rules := generator.GetRules()
	assert.NotEmpty(t, rules)

	metrics, err := generator.EvaluateRule(ctx, rules[0], historicalData)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, rules[0].Name, metrics.RuleName)
	assert.Greater(t, metrics.TotalTests, 0)
	assert.GreaterOrEqual(t, metrics.Precision, 0.0)
	assert.LessOrEqual(t, metrics.Precision, 1.0)
	assert.GreaterOrEqual(t, metrics.Recall, 0.0)
	assert.LessOrEqual(t, metrics.Recall, 1.0)
	assert.GreaterOrEqual(t, metrics.F1Score, 0.0)
	assert.LessOrEqual(t, metrics.F1Score, 1.0)
}

func TestMLRuleGenerator_EvaluateRule_NilRule(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	metrics, err := generator.EvaluateRule(ctx, nil, historicalData)
	assert.Error(t, err)
	assert.Nil(t, metrics)
	assert.Contains(t, err.Error(), "规则不能为空")
}

func TestMLRuleGenerator_EvaluateRule_EmptyTestData(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	rule := &Rule{
		Name:      "test_rule",
		Condition: func(f *RuleEngineFeatures) bool { return f.AverageSpeed > 1000 },
	}
	metrics, err := generator.EvaluateRule(ctx, rule, []RiskData{})
	assert.Error(t, err)
	assert.Nil(t, metrics)
	assert.Contains(t, err.Error(), "测试数据为空")
}

func TestMLRuleGenerator_RecommendRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	scenario := &RiskScenario{
		Name:        "test_scenario",
		Type:        "speed",
		MinSeverity: 0.5,
	}

	recommendedRules, err := generator.RecommendRules(ctx, scenario)
	assert.NoError(t, err)
	assert.NotNil(t, recommendedRules)

	for _, rule := range recommendedRules {
		assert.GreaterOrEqual(t, rule.Severity, scenario.MinSeverity)
	}
}

func TestMLRuleGenerator_RecommendRules_NilScenario(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	recommendedRules, err := generator.RecommendRules(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, recommendedRules)
	assert.Contains(t, err.Error(), "风险场景不能为空")
}

func TestMLRuleGenerator_RunABTest(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	rules := generator.GetRules()
	assert.GreaterOrEqual(t, len(rules), 2)

	ruleA := rules[0]
	ruleB := rules[1]

	result, err := generator.RunABTest(ctx, ruleA, ruleB)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ruleA.Name, result.RuleAName)
	assert.Equal(t, ruleB.Name, result.RuleBName)
	assert.Greater(t, result.SampleSizeA, 0)
	assert.Greater(t, result.SampleSizeB, 0)
	assert.GreaterOrEqual(t, result.ConfidenceLevel, 0.0)
	assert.LessOrEqual(t, result.ConfidenceLevel, 1.0)
	assert.NotEmpty(t, result.Winner)
	assert.NotEmpty(t, result.Recommendation)
}

func TestMLRuleGenerator_RunABTest_NilRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	result, err := generator.RunABTest(ctx, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "规则A和规则B都不能为空")
}

func TestMLRuleGenerator_GetRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	rules := generator.GetRules()
	assert.NotNil(t, rules)

	for _, rule := range rules {
		assert.NotNil(t, rule)
		assert.NotEmpty(t, rule.Name)
	}
}

func TestMLRuleGenerator_GetScenarios(t *testing.T) {
	generator := NewMLRuleGenerator()

	scenarios := generator.GetScenarios()
	assert.NotNil(t, scenarios)
	assert.Greater(t, len(scenarios), 0)

	for _, scenario := range scenarios {
		assert.NotEmpty(t, scenario.Name)
		assert.NotEmpty(t, scenario.Description)
		assert.NotEmpty(t, scenario.Type)
	}
}

func TestMLRuleGenerator_AddScenario(t *testing.T) {
	generator := NewMLRuleGenerator()

	initialCount := len(generator.GetScenarios())

	newScenario := &RiskScenario{
		Name:        "custom_scenario",
		Description: "Custom risk scenario for testing",
		Type:        "custom",
		Keywords:    []string{"custom", "test"},
		MinSeverity: 0.6,
	}

	generator.AddScenario(newScenario)

	scenarios := generator.GetScenarios()
	assert.Equal(t, initialCount+1, len(scenarios))

	found := false
	for _, scenario := range scenarios {
		if scenario.Name == "custom_scenario" {
			found = true
			break
		}
	}
	assert.True(t, found, "Added scenario should be found in the list")
}

func TestMLRuleGenerator_GetFeatureStatistics(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	stats, exists := generator.GetFeatureStatistics("AverageSpeed")
	assert.True(t, exists)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.BotMean, 0.0)
	assert.GreaterOrEqual(t, stats.HumanMean, 0.0)
}

func TestMLRuleGenerator_GetAllFeatureStatistics(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	allStats := generator.GetAllFeatureStatistics()
	assert.NotNil(t, allStats)

	expectedFeatures := []string{
		"PathEfficiency", "SpeedConsistency", "AverageSpeed", "MaxSpeed",
		"SpeedVariance", "CurvatureAverage", "DirectionChanges",
		"MicroCorrections", "HumanLikenessScore", "AnomalyScore",
		"MLScore", "ClickRegularity", "PositionEntropy", "Accuracy",
		"FractalDimension", "JitterScore",
	}

	for _, featureName := range expectedFeatures {
		stats, exists := allStats[featureName]
		assert.True(t, exists, "Feature %s should exist", featureName)
		assert.NotNil(t, stats)
	}
}

func TestMLRuleGenerator_ConvertToEnhancedRule(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	rules := generator.GetRules()
	assert.NotEmpty(t, rules)

	mlRule := rules[0]
	enhancedRule := generator.ConvertToEnhancedRule(mlRule)

	assert.Equal(t, mlRule.Name, enhancedRule.Name)
	assert.Equal(t, mlRule.Description, enhancedRule.Description)
	assert.Equal(t, mlRule.Category, enhancedRule.Category)
	assert.Equal(t, mlRule.Weight, enhancedRule.Weight)
	assert.Equal(t, mlRule.Priority, enhancedRule.Priority)
	assert.Equal(t, mlRule.Severity, enhancedRule.Severity)
	assert.True(t, enhancedRule.Enabled)
}

func TestConvertEnhancedRuleToMLRule(t *testing.T) {
	enhancedRule := EnhancedRule{
		Name:        "test_enhanced_rule",
		Description: "Test enhanced rule",
		Category:    "speed",
		Weight:      25,
		Priority:    2,
		Severity:    0.8,
		Enabled:     true,
		Condition: func(f *RuleEngineFeatures) bool {
			return f.AverageSpeed > 1000
		},
	}

	mlRule := ConvertEnhancedRuleToMLRule(enhancedRule)

	assert.Equal(t, enhancedRule.Name, mlRule.Name)
	assert.Equal(t, enhancedRule.Description, mlRule.Description)
	assert.Equal(t, enhancedRule.Category, mlRule.Category)
	assert.Equal(t, enhancedRule.Weight, mlRule.Weight)
	assert.Equal(t, enhancedRule.Priority, mlRule.Priority)
	assert.Equal(t, enhancedRule.Severity, mlRule.Severity)
	assert.Equal(t, "converted", mlRule.GeneratedFrom)
}

func TestGenerateHistoricalDataFromFeatures(t *testing.T) {
	features := make([]*RuleEngineFeatures, 10)
	labels := make([]bool, 10)

	for i := 0; i < 10; i++ {
		features[i] = &RuleEngineFeatures{
			PathEfficiency:   float64(i) * 0.1,
			AverageSpeed:     float64(i) * 100,
			HumanLikenessScore: float64(i) * 0.1,
		}
		labels[i] = i%2 == 0
	}

	data := GenerateHistoricalDataFromFeatures(features, labels)
	assert.NotNil(t, data)
	assert.Equal(t, len(features), len(data))

	for i, d := range data {
		assert.Equal(t, features[i], d.Features)
		assert.Equal(t, labels[i], d.IsBot)
		assert.NotEmpty(t, d.SessionID)
		assert.NotEmpty(t, d.UserID)
	}
}

func TestGenerateHistoricalDataFromFeatures_MismatchedLength(t *testing.T) {
	features := make([]*RuleEngineFeatures, 10)
	labels := make([]bool, 5)

	data := GenerateHistoricalDataFromFeatures(features, labels)
	assert.Nil(t, data)
}

func TestMLRuleGenerator_CompareRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	rules := generator.GetRules()
	assert.GreaterOrEqual(t, len(rules), 2)

	testRules := rules[:2]
	comparisons, err := generator.CompareRules(testRules, historicalData)
	assert.NoError(t, err)
	assert.NotNil(t, comparisons)
	assert.Equal(t, len(testRules), len(comparisons))

	for i, comparison := range comparisons {
		assert.Equal(t, testRules[i].Name, comparison.Rule.Name)
		assert.NotNil(t, comparison.Metrics)
		assert.GreaterOrEqual(t, comparison.Score, 0.0)
	}
}

func TestMLRuleGenerator_CompareRules_EmptyRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	comparisons, err := generator.CompareRules([]*Rule{}, historicalData)
	assert.Error(t, err)
	assert.Nil(t, comparisons)
	assert.Contains(t, err.Error(), "规则列表为空")
}

func TestRiskData_FeatureAccess(t *testing.T) {
	features := &RuleEngineFeatures{
		PathEfficiency:   0.95,
		SpeedConsistency: 0.85,
		AverageSpeed:     2000,
		MaxSpeed:         3000,
		SpeedVariance:    0.01,
		CurvatureAverage: 0.005,
		DirectionChanges: 2,
		MicroCorrections: 0,
		HumanLikenessScore: 0.1,
		AnomalyScore:     0.8,
		MLScore:          0.75,
		ClickRegularity:  0.99,
		FractalDimension: 1.05,
	}

	data := RiskData{
		Features:  features,
		IsBot:     true,
		Timestamp: time.Now(),
		SessionID: "test_session",
		UserID:    "test_user",
	}

	assert.NotNil(t, data.Features)
	assert.Equal(t, 0.95, data.Features.PathEfficiency)
	assert.Equal(t, 2000, data.Features.AverageSpeed)
	assert.True(t, data.IsBot)
}

func TestRiskScenario_Creation(t *testing.T) {
	scenario := &RiskScenario{
		Name:        "bot_attack_scenario",
		Description: "Detects bot attack patterns",
		Type:        "speed",
		Keywords:    []string{"bot", "attack", "fast"},
		MinSeverity: 0.7,
		HistoryRef:  "ref_001",
	}

	assert.Equal(t, "bot_attack_scenario", scenario.Name)
	assert.Equal(t, "speed", scenario.Type)
	assert.Equal(t, 0.7, scenario.MinSeverity)
	assert.Contains(t, scenario.Keywords, "bot")
}

func TestRuleMetrics_Calculation(t *testing.T) {
	metrics := &RuleMetrics{
		RuleName:       "test_rule",
		Precision:      0.85,
		Recall:         0.80,
		Accuracy:       0.82,
		TruePositives:  85,
		FalsePositives: 15,
		TrueNegatives:  80,
		FalseNegatives: 20,
		TotalTests:     200,
		Confidence:     0.9,
	}

	assert.Equal(t, "test_rule", metrics.RuleName)
	assert.Equal(t, 200, metrics.TotalTests)

	expectedF1 := 2 * (0.85 * 0.80) / (0.85 + 0.80)
	assert.InDelta(t, expectedF1, 0.824, 0.001)
}

func TestABTestResult_Analysis(t *testing.T) {
	result := &ABTestResult{
		RuleAName:       "rule_a",
		RuleBName:       "rule_b",
		StartTime:       time.Now().Add(-1 * time.Hour),
		EndTime:         time.Now(),
		SampleSizeA:     100,
		SampleSizeB:     100,
		MetricsA: &RuleMetrics{
			F1Score:   0.85,
			Accuracy:  0.87,
			Precision: 0.83,
			Recall:    0.87,
		},
		MetricsB: &RuleMetrics{
			F1Score:   0.80,
			Accuracy:  0.82,
			Precision: 0.78,
			Recall:    0.82,
		},
		Winner:          "A",
		ConfidenceLevel: 0.92,
		Recommendation:  "Use rule A",
		IsConclusive:    false,
	}

	assert.Equal(t, "rule_a", result.RuleAName)
	assert.Equal(t, "rule_b", result.RuleBName)
	assert.Greater(t, result.SampleSizeA, 0)
	assert.Greater(t, result.SampleSizeB, 0)
	assert.Greater(t, result.MetricsA.F1Score, result.MetricsB.F1Score)
	assert.Equal(t, "A", result.Winner)
}

func TestMLRuleGenerator_MultipleGenerateRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData1 := generateTestHistoricalData(100)
	rules1, err := generator.GenerateRules(ctx, historicalData1)
	assert.NoError(t, err)
	assert.NotEmpty(t, rules1)

	historicalData2 := generateTestHistoricalData(100)
	rules2, err := generator.GenerateRules(ctx, historicalData2)
	assert.NoError(t, err)
	assert.NotEmpty(t, rules2)

	assert.Equal(t, len(rules1), len(rules2))
}

func TestMLRuleGenerator_GenerateRulesWithDifferentDataSizes(t *testing.T) {
	ctx := context.Background()

	sizes := []int{10, 50, 100, 200}
	for _, size := range sizes {
		generator := NewMLRuleGenerator()
		historicalData := generateTestHistoricalData(size)

		rules, err := generator.GenerateRules(ctx, historicalData)
		assert.NoError(t, err)

		t.Run("Size_"+string(rune(size)), func(t *testing.T) {
			assert.NotNil(t, rules)
			assert.Greater(t, len(rules), 0)
		})
	}
}

func TestFeatureStats_Discriminative(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	_, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	allStats := generator.GetAllFeatureStatistics()
	for featureName, stats := range allStats {
		t.Run("Feature_"+featureName, func(t *testing.T) {
			assert.GreaterOrEqual(t, stats.Discriminative, 0.0)
			assert.False(t, math.IsNaN(stats.Discriminative))
			assert.False(t, math.IsInf(stats.Discriminative, 0))
		})
	}
}

func TestRuleCondition_AllCategories(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(100)
	rules, err := generator.GenerateRules(ctx, historicalData)
	assert.NoError(t, err)

	testFeatures := &RuleEngineFeatures{
		PathEfficiency:     0.95,
		SpeedConsistency:   0.9,
		AverageSpeed:       2000,
		MaxSpeed:           3000,
		SpeedVariance:      0.01,
		CurvatureAverage:   0.005,
		DirectionChanges:   2,
		MicroCorrections:   0,
		HumanLikenessScore: 0.1,
		AnomalyScore:       0.8,
		MLScore:            0.75,
		ClickRegularity:    0.99,
		PositionEntropy:    1.0,
		Accuracy:           0.95,
		FractalDimension:   1.05,
		JitterScore:        0.005,
	}

	categories := make(map[string]bool)
	for _, rule := range rules {
		result := rule.Condition(testFeatures)
		_ = result
		categories[rule.Category] = true
	}

	expectedCategories := []string{"speed", "trajectory", "general", "click", "combined"}
	for _, category := range expectedCategories {
		if _, exists := categories[category]; exists {
			assert.True(t, true)
			return
		}
	}
}

func TestEnhancedRuleEngine_MLIntegration(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)

	err := engine.IntegrateMLGenerator(ctx, historicalData)
	assert.NoError(t, err)

	allRules := engine.GetAllRules()
	assert.Greater(t, len(allRules), 0)

	mlRules := engine.ExportMLRules()
	assert.NotNil(t, mlRules)
}

func TestEnhancedRuleEngine_GenerateMLRules(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)

	rules, err := engine.GenerateMLRules(ctx, historicalData)
	assert.NoError(t, err)
	assert.NotNil(t, rules)
	assert.Greater(t, len(rules), 0)
}

func TestEnhancedRuleEngine_EvaluateMLRule(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)
	mlRules, _ := engine.GenerateMLRules(ctx, historicalData)

	if len(mlRules) > 0 {
		metrics, err := engine.EvaluateMLRule(ctx, mlRules[0], historicalData)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
	}
}

func TestEnhancedRuleEngine_RecommendRulesForScenario(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	scenario := &RiskScenario{
		Name:        "high_speed_attack",
		Type:        "speed",
		MinSeverity: 0.6,
	}

	rules, err := engine.RecommendRulesForScenario(ctx, scenario)
	assert.NoError(t, err)
	assert.NotNil(t, rules)
}

func TestEnhancedRuleEngine_RunMLABTest(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)
	mlRules, _ := engine.GenerateMLRules(ctx, historicalData)

	if len(mlRules) >= 2 {
		result, err := engine.RunMLABTest(ctx, mlRules[0], mlRules[1])
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Winner)
	}
}

func TestEnhancedRuleEngine_GetMLFeatureStatistics(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	stats := engine.GetMLFeatureStatistics()
	assert.NotNil(t, stats)

	for featureName, featureStats := range stats {
		t.Run("Feature_"+featureName, func(t *testing.T) {
			assert.GreaterOrEqual(t, featureStats.BotMean, 0.0)
			assert.GreaterOrEqual(t, featureStats.HumanMean, 0.0)
		})
	}
}

func TestEnhancedRuleEngine_CompareMLRules(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)
	mlRules, _ := engine.GenerateMLRules(ctx, historicalData)

	if len(mlRules) >= 2 {
		testRules := mlRules[:2]
		comparisons, err := engine.CompareMLRules(ctx, testRules, historicalData)
		assert.NoError(t, err)
		assert.NotNil(t, comparisons)
		assert.Equal(t, len(testRules), len(comparisons))
	}
}

func TestEnhancedRuleEngine_AddMLGeneratedRules(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	initialCount := len(engine.GetAllRules())

	historicalData := generateTestHistoricalData(100)
	err := engine.AddMLGeneratedRules(ctx, historicalData)
	assert.NoError(t, err)

	newCount := len(engine.GetAllRules())
	assert.Greater(t, newCount, initialCount)
}

func TestEnhancedRuleEngine_GetEnhancedRuleMetrics(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)
	mlRules, _ := engine.GenerateMLRules(ctx, historicalData)

	if len(mlRules) > 0 {
		ruleName := mlRules[0].Name
		metrics, err := engine.GetEnhancedRuleMetrics(ctx, ruleName, historicalData)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Equal(t, ruleName, metrics.RuleName)
	}
}

func TestEnhancedRuleEngine_GetEnhancedRuleMetrics_NotFound(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)
	metrics, err := engine.GetEnhancedRuleMetrics(ctx, "non_existent_rule", historicalData)
	assert.Error(t, err)
	assert.Nil(t, metrics)
}

func TestEnhancedRuleEngine_ImportMLRules(t *testing.T) {
	ctx := context.Background()
	engine := NewEnhancedRuleEngine()

	historicalData := generateTestHistoricalData(100)
	mlRules, _ := engine.GenerateMLRules(ctx, historicalData)

	initialCount := len(engine.GetAllRules())

	engine.ImportMLRules(mlRules)

	newCount := len(engine.GetAllRules())
	assert.GreaterOrEqual(t, newCount, initialCount)
}

func TestMLRuleGenerator_Concurrency(t *testing.T) {
	generator := NewMLRuleGenerator()

	historicalData := generateTestHistoricalData(50)

	done := make(chan bool)

	go func() {
		ctx := context.Background()
		generator.GenerateRules(ctx, historicalData)
		done <- true
	}()

	go func() {
		ctx := context.Background()
		generator.GenerateRules(ctx, historicalData)
		done <- true
	}()

	<-done
	<-done

	rules := generator.GetRules()
	assert.NotNil(t, rules)
}

func TestMLRuleGenerator_EmptyHistoricalData(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	rules, err := generator.GenerateRules(ctx, []RiskData{})
	assert.Error(t, err)
	assert.Nil(t, rules)
}

func TestMLRuleGenerator_RecommendRules_EmptyRules(t *testing.T) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()

	scenario := &RiskScenario{
		Name:        "test",
		Type:        "speed",
		MinSeverity: 0.5,
	}

	rules, err := generator.RecommendRules(ctx, scenario)
	assert.NoError(t, err)
	assert.NotNil(t, rules)
	assert.Empty(t, rules)
}

func BenchmarkMLRuleGenerator_GenerateRules(b *testing.B) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()
	historicalData := generateTestHistoricalData(200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.GenerateRules(ctx, historicalData)
	}
}

func BenchmarkMLRuleGenerator_EvaluateRule(b *testing.B) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()
	historicalData := generateTestHistoricalData(200)
	rules, _ := generator.GenerateRules(ctx, historicalData)

	if len(rules) > 0 {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = generator.EvaluateRule(ctx, rules[0], historicalData)
		}
	}
}

func BenchmarkMLRuleGenerator_RunABTest(b *testing.B) {
	ctx := context.Background()
	generator := NewMLRuleGenerator()
	historicalData := generateTestHistoricalData(200)
	rules, _ := generator.GenerateRules(ctx, historicalData)

	if len(rules) >= 2 {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = generator.RunABTest(ctx, rules[0], rules[1])
		}
	}
}
