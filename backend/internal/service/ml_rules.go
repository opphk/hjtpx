package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type RiskData struct {
	Features    *RuleEngineFeatures
	IsBot       bool
	Timestamp   time.Time
	SessionID   string
	UserID      string
	RequestCount int
}

type RiskScenario struct {
	Name        string
	Description string
	Type        string
	Keywords    []string
	MinSeverity float64
	HistoryRef  string
}

type RuleMetrics struct {
	RuleName       string
	Precision      float64
	Recall         float64
	F1Score        float64
	Accuracy       float64
	TruePositives  int
	FalsePositives int
	TrueNegatives  int
	FalseNegatives int
	TotalTests     int
	Confidence     float64
	EvaluationTime time.Duration
}

type ABTestResult struct {
	RuleAName       string
	RuleBName       string
	StartTime       time.Time
	EndTime         time.Time
	SampleSizeA     int
	SampleSizeB     int
	MetricsA        *RuleMetrics
	MetricsB        *RuleMetrics
	Winner          string
	ConfidenceLevel float64
	Recommendation  string
	IsConclusive    bool
}

type MLRuleGenerator struct {
	mu          sync.RWMutex
	rules       []*Rule
	scenarios   []*RiskScenario
	featureStats map[string]*FeatureStats
}

type FeatureStats struct {
	BotMean      float64
	BotStd       float64
	HumanMean    float64
	HumanStd     float64
	Discriminative float64
}

type Rule struct {
	Name          string
	Description   string
	Category      string
	Condition     func(*RuleEngineFeatures) bool
	Weight        float64
	Priority      int
	Severity      float64
	Threshold     float64
	Feature       string
	Operator      string
	GeneratedFrom string
	MinSupport    float64
	CreatedAt     time.Time
}

type RuleGeneratorConfig struct {
	MinSupport      float64
	MinConfidence   float64
	MaxRulesPerCategory int
	FeatureThreshold float64
}

var defaultRuleGeneratorConfig = &RuleGeneratorConfig{
	MinSupport:        0.05,
	MinConfidence:      0.7,
	MaxRulesPerCategory: 20,
	FeatureThreshold:   0.1,
}

func NewMLRuleGenerator() *MLRuleGenerator {
	return &MLRuleGenerator{
		rules:         make([]*Rule, 0),
		scenarios:     initializeDefaultScenarios(),
		featureStats:  make(map[string]*FeatureStats),
	}
}

func initializeDefaultScenarios() []*RiskScenario {
	return []*RiskScenario{
		{
			Name:        "high_speed_attack",
			Description: "检测高速自动化攻击行为",
			Type:        "speed",
			Keywords:    []string{"speed", "fast", "rapid"},
			MinSeverity: 0.7,
		},
		{
			Name:        "perfect_trajectory",
			Description: "检测完美路径轨迹（机器人特征）",
			Type:        "trajectory",
			Keywords:    []string{"path", "trajectory", "smooth"},
			MinSeverity: 0.6,
		},
		{
			Name:        "human_simulation",
			Description: "检测模拟人类行为",
			Type:        "behavior",
			Keywords:    []string{"human", "likeness", "behavior"},
			MinSeverity: 0.8,
		},
		{
			Name:        "click_fraud",
			Description: "检测点击欺诈行为",
			Type:        "click",
			Keywords:    []string{"click", "regularity", "position"},
			MinSeverity: 0.6,
		},
		{
			Name:        "combined_attack",
			Description: "检测组合式攻击",
			Type:        "combined",
			Keywords:    []string{"combined", "multiple", "risk"},
			MinSeverity: 0.75,
		},
	}
}

func (g *MLRuleGenerator) GenerateRules(ctx context.Context, historicalData []RiskData) ([]*Rule, error) {
	if len(historicalData) == 0 {
		return nil, fmt.Errorf("历史数据为空")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.calculateFeatureStatistics(historicalData)

	rules := make([]*Rule, 0)

	speedRules := g.generateSpeedRules(historicalData)
	rules = append(rules, speedRules...)

	trajectoryRules := g.generateTrajectoryRules(historicalData)
	rules = append(rules, trajectoryRules...)

	behaviorRules := g.generateBehaviorRules(historicalData)
	rules = append(rules, behaviorRules...)

	clickRules := g.generateClickRules(historicalData)
	rules = append(rules, clickRules...)

	combinedRules := g.generateCombinedRules(historicalData)
	rules = append(rules, combinedRules...)

	g.rules = rules

	return rules, nil
}

func (g *MLRuleGenerator) calculateFeatureStatistics(data []RiskData) {
	g.featureStats = make(map[string]*FeatureStats)

	featureNames := []string{
		"PathEfficiency", "SpeedConsistency", "AverageSpeed", "MaxSpeed",
		"SpeedVariance", "CurvatureAverage", "DirectionChanges",
		"MicroCorrections", "HumanLikenessScore", "AnomalyScore",
		"MLScore", "ClickRegularity", "PositionEntropy", "Accuracy",
		"FractalDimension", "JitterScore",
	}

	for _, featureName := range featureNames {
		stats := &FeatureStats{}
		botValues := make([]float64, 0)
		humanValues := make([]float64, 0)

		for _, d := range data {
			if d.Features == nil {
				continue
			}
			value := g.getFeatureValue(d.Features, featureName)
			if value == 0 && featureName != "DirectionChanges" && featureName != "MicroCorrections" {
				continue
			}
			if d.IsBot {
				botValues = append(botValues, value)
			} else {
				humanValues = append(humanValues, value)
			}
		}

		if len(botValues) > 0 {
			stats.BotMean = calculateMean(botValues)
			stats.BotStd = calculateStd(botValues, stats.BotMean)
		}
		if len(humanValues) > 0 {
			stats.HumanMean = calculateMean(humanValues)
			stats.HumanStd = calculateStd(humanValues, stats.HumanMean)
		}

		stats.Discriminative = math.Abs(stats.BotMean-stats.HumanMean) /
			((stats.BotStd + stats.HumanStd) / 2 + 0.001)

		g.featureStats[featureName] = stats
	}
}

func (g *MLRuleGenerator) getFeatureValue(features *RuleEngineFeatures, name string) float64 {
	switch name {
	case "PathEfficiency":
		return features.PathEfficiency
	case "SpeedConsistency":
		return features.SpeedConsistency
	case "AverageSpeed":
		return features.AverageSpeed
	case "MaxSpeed":
		return features.MaxSpeed
	case "SpeedVariance":
		return features.SpeedVariance
	case "CurvatureAverage":
		return features.CurvatureAverage
	case "DirectionChanges":
		return float64(features.DirectionChanges)
	case "MicroCorrections":
		return float64(features.MicroCorrections)
	case "HumanLikenessScore":
		return features.HumanLikenessScore
	case "AnomalyScore":
		return features.AnomalyScore
	case "MLScore":
		return features.MLScore
	case "ClickRegularity":
		return features.ClickRegularity
	case "PositionEntropy":
		return features.PositionEntropy
	case "Accuracy":
		return features.Accuracy
	case "FractalDimension":
		return features.FractalDimension
	case "JitterScore":
		return features.JitterScore
	default:
		return 0
	}
}

func (g *MLRuleGenerator) generateSpeedRules(data []RiskData) []*Rule {
	rules := make([]*Rule, 0)

	speedStats := g.featureStats["AverageSpeed"]
	if speedStats != nil && speedStats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
		threshold := (speedStats.BotMean + speedStats.HumanMean) / 2

		rule := &Rule{
			Name:          "ml_generated_speed_threshold",
			Description:   fmt.Sprintf("基于ML生成的速度阈值规则 (阈值: %.2f)", threshold),
			Category:      "speed",
			Feature:       "AverageSpeed",
			Operator:      ">",
			Threshold:     threshold,
			Severity:      0.8,
			Weight:        25,
			Priority:      2,
			GeneratedFrom: "ml_speed_analysis",
			MinSupport:    0.1,
			CreatedAt:     time.Now(),
		}
		rule.Condition = func(f *RuleEngineFeatures) bool {
			return f.AverageSpeed > rule.Threshold
		}
		rules = append(rules, rule)

		highSpeedRule := &Rule{
			Name:          "ml_generated_high_speed",
			Description:   fmt.Sprintf("ML检测到异常高速行为 (阈值: %.2f)", threshold*1.2),
			Category:      "speed",
			Feature:       "AverageSpeed",
			Operator:      ">",
			Threshold:     threshold * 1.2,
			Severity:      0.9,
			Weight:        30,
			Priority:      1,
			GeneratedFrom: "ml_speed_analysis",
			MinSupport:    0.05,
			CreatedAt:     time.Now(),
		}
		highSpeedRule.Condition = func(f *RuleEngineFeatures) bool {
			return f.AverageSpeed > highSpeedRule.Threshold
		}
		rules = append(rules, highSpeedRule)
	}

	speedVarianceStats := g.featureStats["SpeedVariance"]
	if speedVarianceStats != nil && speedVarianceStats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
		if speedVarianceStats.BotMean < speedVarianceStats.HumanMean {
			rule := &Rule{
				Name:          "ml_generated_low_speed_variance",
				Description:   "ML检测到异常稳定的速度（机器人特征）",
				Category:      "speed",
				Feature:       "SpeedVariance",
				Operator:      "<",
				Threshold:     (speedVarianceStats.BotMean + speedVarianceStats.HumanMean) / 2,
				Severity:      0.75,
				Weight:        22,
				Priority:      3,
				GeneratedFrom: "ml_speed_analysis",
				MinSupport:    0.1,
				CreatedAt:     time.Now(),
			}
			rule.Condition = func(f *RuleEngineFeatures) bool {
				return f.SpeedVariance < rule.Threshold && f.SpeedVariance >= 0
			}
			rules = append(rules, rule)
		}
	}

	return rules
}

func (g *MLRuleGenerator) generateTrajectoryRules(data []RiskData) []*Rule {
	rules := make([]*Rule, 0)

	pathEffStats := g.featureStats["PathEfficiency"]
	if pathEffStats != nil && pathEffStats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
		if pathEffStats.BotMean > pathEffStats.HumanMean {
			threshold := (pathEffStats.BotMean + pathEffStats.HumanMean) / 2

			rule := &Rule{
				Name:          "ml_generated_high_path_efficiency",
				Description:   fmt.Sprintf("ML检测到异常高路径效率（机器人特征，阈值: %.3f）", threshold),
				Category:      "trajectory",
				Feature:       "PathEfficiency",
				Operator:      ">",
				Threshold:     threshold,
				Severity:      0.85,
				Weight:        28,
				Priority:      2,
				GeneratedFrom: "ml_trajectory_analysis",
				MinSupport:    0.1,
				CreatedAt:     time.Now(),
			}
			rule.Condition = func(f *RuleEngineFeatures) bool {
				return f.PathEfficiency > rule.Threshold
			}
			rules = append(rules, rule)

			perfectRule := &Rule{
				Name:          "ml_generated_perfect_trajectory",
				Description:   "ML检测到近乎完美的轨迹（高度疑似机器人）",
				Category:      "trajectory",
				Feature:       "PathEfficiency",
				Operator:      ">",
				Threshold:     0.98,
				Severity:      0.95,
				Weight:        35,
				Priority:      1,
				GeneratedFrom: "ml_trajectory_analysis",
				MinSupport:    0.05,
				CreatedAt:     time.Now(),
			}
			perfectRule.Condition = func(f *RuleEngineFeatures) bool {
				return f.PathEfficiency > perfectRule.Threshold
			}
			rules = append(rules, perfectRule)
		}
	}

	curvatureStats := g.featureStats["CurvatureAverage"]
	if curvatureStats != nil && curvatureStats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
		if curvatureStats.BotMean < curvatureStats.HumanMean {
			rule := &Rule{
				Name:          "ml_generated_low_curvature",
				Description:   "ML检测到异常低曲率轨迹",
				Category:      "trajectory",
				Feature:       "CurvatureAverage",
				Operator:      "<",
				Threshold:     (curvatureStats.BotMean + curvatureStats.HumanMean) / 2,
				Severity:      0.7,
				Weight:        23,
				Priority:      3,
				GeneratedFrom: "ml_trajectory_analysis",
				MinSupport:    0.1,
				CreatedAt:     time.Now(),
			}
			rule.Condition = func(f *RuleEngineFeatures) bool {
				return f.CurvatureAverage < rule.Threshold && f.CurvatureAverage >= 0
			}
			rules = append(rules, rule)
		}
	}

	return rules
}

func (g *MLRuleGenerator) generateBehaviorRules(data []RiskData) []*Rule {
	rules := make([]*Rule, 0)

	humanLikenessStats := g.featureStats["HumanLikenessScore"]
	if humanLikenessStats != nil && humanLikenessStats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
		if humanLikenessStats.BotMean < humanLikenessStats.HumanMean {
			threshold := (humanLikenessStats.BotMean + humanLikenessStats.HumanMean) / 2

			rule := &Rule{
				Name:          "ml_generated_low_human_likeness",
				Description:   fmt.Sprintf("ML检测到低人类相似度（阈值: %.3f）", threshold),
				Category:      "general",
				Feature:       "HumanLikenessScore",
				Operator:      "<",
				Threshold:     threshold,
				Severity:      0.8,
				Weight:        28,
				Priority:      2,
				GeneratedFrom: "ml_behavior_analysis",
				MinSupport:    0.1,
				CreatedAt:     time.Now(),
			}
			rule.Condition = func(f *RuleEngineFeatures) bool {
				return f.HumanLikenessScore < rule.Threshold && f.HumanLikenessScore >= 0
			}
			rules = append(rules, rule)

			veryLowRule := &Rule{
				Name:          "ml_generated_very_low_human_likeness",
				Description:   "ML检测到极低人类相似度（高度疑似机器人）",
				Category:      "general",
				Feature:       "HumanLikenessScore",
				Operator:      "<",
				Threshold:     humanLikenessStats.BotMean + humanLikenessStats.BotStd,
				Severity:      0.95,
				Weight:        35,
				Priority:      1,
				GeneratedFrom: "ml_behavior_analysis",
				MinSupport:    0.05,
				CreatedAt:     time.Now(),
			}
			veryLowRule.Condition = func(f *RuleEngineFeatures) bool {
				return f.HumanLikenessScore < veryLowRule.Threshold && f.HumanLikenessScore >= 0
			}
			rules = append(rules, veryLowRule)
		}
	}

	anomalyStats := g.featureStats["AnomalyScore"]
	if anomalyStats != nil && anomalyStats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
		if anomalyStats.BotMean > anomalyStats.HumanMean {
			threshold := (anomalyStats.BotMean + anomalyStats.HumanMean) / 2

			rule := &Rule{
				Name:          "ml_generated_high_anomaly",
				Description:   fmt.Sprintf("ML检测到高异常分数（阈值: %.3f）", threshold),
				Category:      "general",
				Feature:       "AnomalyScore",
				Operator:      ">",
				Threshold:     threshold,
				Severity:      0.85,
				Weight:        30,
				Priority:      2,
				GeneratedFrom: "ml_behavior_analysis",
				MinSupport:    0.1,
				CreatedAt:     time.Now(),
			}
			rule.Condition = func(f *RuleEngineFeatures) bool {
				return f.AnomalyScore > rule.Threshold
			}
			rules = append(rules, rule)
		}
	}

	return rules
}

func (g *MLRuleGenerator) generateClickRules(data []RiskData) []*Rule {
	rules := make([]*Rule, 0)

	clickRegStats := g.featureStats["ClickRegularity"]
	if clickRegStats != nil && clickRegStats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
		if clickRegStats.BotMean > clickRegStats.HumanMean {
			threshold := (clickRegStats.BotMean + clickRegStats.HumanMean) / 2

			rule := &Rule{
				Name:          "ml_generated_high_click_regularity",
				Description:   fmt.Sprintf("ML检测到异常规律的点击模式（阈值: %.3f）", threshold),
				Category:      "click",
				Feature:       "ClickRegularity",
				Operator:      ">",
				Threshold:     threshold,
				Severity:      0.75,
				Weight:        25,
				Priority:      2,
				GeneratedFrom: "ml_click_analysis",
				MinSupport:    0.1,
				CreatedAt:     time.Now(),
			}
			rule.Condition = func(f *RuleEngineFeatures) bool {
				return f.ClickRegularity > rule.Threshold
			}
			rules = append(rules, rule)

			perfectRule := &Rule{
				Name:          "ml_generated_perfect_click_regularity",
				Description:   "ML检测到近乎完美的点击规律性",
				Category:      "click",
				Feature:       "ClickRegularity",
				Operator:      ">",
				Threshold:     0.98,
				Severity:      0.9,
				Weight:        32,
				Priority:      1,
				GeneratedFrom: "ml_click_analysis",
				MinSupport:    0.05,
				CreatedAt:     time.Now(),
			}
			perfectRule.Condition = func(f *RuleEngineFeatures) bool {
				return f.ClickRegularity > perfectRule.Threshold
			}
			rules = append(rules, perfectRule)
		}
	}

	return rules
}

func (g *MLRuleGenerator) generateCombinedRules(data []RiskData) []*Rule {
	rules := make([]*Rule, 0)

	highRiskFeatures := []string{"PathEfficiency", "SpeedConsistency", "HumanLikenessScore", "AnomalyScore"}
	riskCount := 0
	for _, feature := range highRiskFeatures {
		if stats, ok := g.featureStats[feature]; ok && stats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
			riskCount++
		}
	}

	if riskCount >= 2 {
		rule := &Rule{
			Name:          "ml_generated_multi_feature_risk",
			Description:   "ML检测到多特征同时异常（组合攻击检测）",
			Category:      "combined",
			Severity:      0.9,
			Weight:        35,
			Priority:      1,
			GeneratedFrom: "ml_combined_analysis",
			MinSupport:    0.1,
			CreatedAt:     time.Now(),
		}
		rule.Condition = func(f *RuleEngineFeatures) bool {
			highRiskCount := 0
			if stats := g.featureStats["PathEfficiency"]; stats != nil && stats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
				if f.PathEfficiency > (stats.BotMean+stats.HumanMean)/2 {
					highRiskCount++
				}
			}
			if stats := g.featureStats["SpeedConsistency"]; stats != nil && stats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
				if f.SpeedConsistency > (stats.BotMean+stats.HumanMean)/2 {
					highRiskCount++
				}
			}
			if stats := g.featureStats["HumanLikenessScore"]; stats != nil && stats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
				if f.HumanLikenessScore < (stats.BotMean+stats.HumanMean)/2 {
					highRiskCount++
				}
			}
			if stats := g.featureStats["AnomalyScore"]; stats != nil && stats.Discriminative > defaultRuleGeneratorConfig.FeatureThreshold {
				if f.AnomalyScore > (stats.BotMean+stats.HumanMean)/2 {
					highRiskCount++
				}
			}
			return highRiskCount >= 2
		}
		rules = append(rules, rule)
	}

	return rules
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStd(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)))
}

func (g *MLRuleGenerator) EvaluateRule(ctx context.Context, rule *Rule, testData []RiskData) (*RuleMetrics, error) {
	if rule == nil {
		return nil, fmt.Errorf("规则不能为空")
	}
	if len(testData) == 0 {
		return nil, fmt.Errorf("测试数据为空")
	}

	startTime := time.Now()

	metrics := &RuleMetrics{
		RuleName:   rule.Name,
		TotalTests: len(testData),
	}

	for _, data := range testData {
		predictedBot := false
		if data.Features != nil {
			predictedBot = rule.Condition(data.Features)
		}

		if predictedBot && data.IsBot {
			metrics.TruePositives++
		} else if predictedBot && !data.IsBot {
			metrics.FalsePositives++
		} else if !predictedBot && !data.IsBot {
			metrics.TrueNegatives++
		} else if !predictedBot && data.IsBot {
			metrics.FalseNegatives++
		}
	}

	totalPredictedBot := metrics.TruePositives + metrics.FalsePositives
	if totalPredictedBot > 0 {
		metrics.Precision = float64(metrics.TruePositives) / float64(totalPredictedBot)
	}

	totalActualBot := metrics.TruePositives + metrics.FalseNegatives
	if totalActualBot > 0 {
		metrics.Recall = float64(metrics.TruePositives) / float64(totalActualBot)
	}

	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1Score = 2 * (metrics.Precision * metrics.Recall) / (metrics.Precision + metrics.Recall)
	}

	correctPredictions := metrics.TruePositives + metrics.TrueNegatives
	metrics.Accuracy = float64(correctPredictions) / float64(metrics.TotalTests)

	metrics.Confidence = calculateConfidence(metrics)
	metrics.EvaluationTime = time.Since(startTime)

	return metrics, nil
}

func calculateConfidence(metrics *RuleMetrics) float64 {
	baseConfidence := 0.5

	if metrics.TotalTests >= 100 {
		baseConfidence += 0.2
	} else if metrics.TotalTests >= 50 {
		baseConfidence += 0.1
	}

	baseConfidence += metrics.Accuracy * 0.3

	if metrics.Precision > 0.9 && metrics.Recall > 0.7 {
		baseConfidence += 0.1
	} else if metrics.Precision > 0.8 && metrics.Recall > 0.6 {
		baseConfidence += 0.05
	}

	return math.Min(baseConfidence, 0.99)
}

func (g *MLRuleGenerator) RecommendRules(ctx context.Context, scenario *RiskScenario) ([]*Rule, error) {
	if scenario == nil {
		return nil, fmt.Errorf("风险场景不能为空")
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	recommendedRules := make([]*Rule, 0)

	for _, rule := range g.rules {
		if rule.Severity < scenario.MinSeverity {
			continue
		}

		if g.ruleMatchesScenario(rule, scenario) {
			recommendedRules = append(recommendedRules, rule)
		}
	}

	sort.Slice(recommendedRules, func(i, j int) bool {
		if recommendedRules[i].Severity != recommendedRules[j].Severity {
			return recommendedRules[i].Severity > recommendedRules[j].Severity
		}
		return recommendedRules[i].Priority < recommendedRules[j].Priority
	})

	return recommendedRules, nil
}

func (g *MLRuleGenerator) ruleMatchesScenario(rule *Rule, scenario *RiskScenario) bool {
	if scenario.Type != "" && rule.Category == scenario.Type {
		return true
	}

	ruleDescLower := toLower(rule.Description)
	for _, keyword := range scenario.Keywords {
		if stringContains(ruleDescLower, toLower(keyword)) {
			return true
		}
	}

	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (g *MLRuleGenerator) RunABTest(ctx context.Context, ruleA, ruleB *Rule) (*ABTestResult, error) {
	if ruleA == nil || ruleB == nil {
		return nil, fmt.Errorf("规则A和规则B都不能为空")
	}

	result := &ABTestResult{
		RuleAName: ruleA.Name,
		RuleBName: ruleB.Name,
		StartTime: time.Now(),
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	syntheticData := g.generateSyntheticTestData(200)
	syntheticData = append(syntheticData, g.generateSyntheticTestData(200)...)

	shuffleData(syntheticData)

	midpoint := len(syntheticData) / 2
	dataA := syntheticData[:midpoint]
	dataB := syntheticData[midpoint:]

	var err error
	result.MetricsA, err = g.EvaluateRule(ctx, ruleA, dataA)
	if err != nil {
		return nil, fmt.Errorf("评估规则A失败: %w", err)
	}

	result.MetricsB, err = g.EvaluateRule(ctx, ruleB, dataB)
	if err != nil {
		return nil, fmt.Errorf("评估规则B失败: %w", err)
	}

	result.SampleSizeA = len(dataA)
	result.SampleSizeB = len(dataB)
	result.EndTime = time.Now()

	result.ConfidenceLevel = calculateABTestConfidence(result.MetricsA, result.MetricsB)
	result.Winner = determineWinner(result.MetricsA, result.MetricsB)
	result.Recommendation = generateRecommendation(result)
	result.IsConclusive = result.ConfidenceLevel > 0.95

	return result, nil
}

func (g *MLRuleGenerator) generateSyntheticTestData(count int) []RiskData {
	data := make([]RiskData, count)

	for i := 0; i < count; i++ {
		isBot := rand.Float64() < 0.5

		features := &RuleEngineFeatures{}

		if isBot {
			features.PathEfficiency = 0.9 + rand.Float64()*0.1
			features.SpeedConsistency = 0.85 + rand.Float64()*0.15
			features.AverageSpeed = 1500 + rand.Float64()*1500
			features.SpeedVariance = rand.Float64() * 0.1
			features.CurvatureAverage = rand.Float64() * 0.05
			features.HumanLikenessScore = rand.Float64() * 0.3
			features.AnomalyScore = 0.6 + rand.Float64()*0.4
			features.MLScore = 0.5 + rand.Float64()*0.5
			features.ClickRegularity = 0.8 + rand.Float64()*0.2
			features.FractalDimension = 1.0 + rand.Float64()*0.1
		} else {
			features.PathEfficiency = 0.5 + rand.Float64()*0.4
			features.SpeedConsistency = 0.4 + rand.Float64()*0.5
			features.AverageSpeed = 200 + rand.Float64()*800
			features.SpeedVariance = 0.1 + rand.Float64()*0.9
			features.CurvatureAverage = 0.05 + rand.Float64()*0.45
			features.HumanLikenessScore = 0.5 + rand.Float64()*0.5
			features.AnomalyScore = rand.Float64() * 0.5
			features.MLScore = rand.Float64() * 0.5
			features.ClickRegularity = rand.Float64() * 0.6
			features.FractalDimension = 1.2 + rand.Float64()*0.5
		}

		data[i] = RiskData{
			Features:  features,
			IsBot:     isBot,
			Timestamp: time.Now(),
			SessionID: fmt.Sprintf("session_%d", i),
			UserID:    fmt.Sprintf("user_%d", rand.Intn(1000)),
		}
	}

	return data
}

func shuffleData(data []RiskData) {
	for i := len(data) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		data[i], data[j] = data[j], data[i]
	}
}

func calculateABTestConfidence(metricsA, metricsB *RuleMetrics) float64 {
	deltaF1 := math.Abs(metricsA.F1Score - metricsB.F1Score)

	nA := float64(metricsA.TotalTests)
	nB := float64(metricsB.TotalTests)

	seA := math.Sqrt(metricsA.F1Score*(1-metricsA.F1Score) / nA)
	seB := math.Sqrt(metricsB.F1Score*(1-metricsB.F1Score) / nB)

	if seA == 0 && seB == 0 {
		return 1.0
	}

	seDiff := math.Sqrt(seA*seA + seB*seB)
	if seDiff == 0 {
		return 1.0
	}

	zScore := deltaF1 / seDiff

	confidence := 0.5 * (1 + math.Erf(zScore/math.Sqrt2))

	return math.Min(math.Max(confidence, 0), 1)
}

func determineWinner(metricsA, metricsB *RuleMetrics) string {
	if metricsA.F1Score > metricsB.F1Score {
		return "A"
	} else if metricsB.F1Score > metricsA.F1Score {
		return "B"
	}

	if metricsA.Accuracy > metricsB.Accuracy {
		return "A"
	} else if metricsB.Accuracy > metricsA.Accuracy {
		return "B"
	}

	return "tie"
}

func generateRecommendation(result *ABTestResult) string {
	if result.Winner == "tie" {
		return "规则A和规则B性能相近，建议根据业务需求选择或继续收集数据"
	}

	winnerName := result.RuleAName
	_ = result.RuleBName
	winnerMetrics := result.MetricsA

	if result.Winner == "B" {
		winnerName = result.RuleBName
		_ = result.RuleAName
		winnerMetrics = result.MetricsB
	}

	if result.IsConclusive {
		return fmt.Sprintf("推荐使用规则 %s，F1分数提升 %.2f%%，置信度 %.1f%%",
			winnerName,
			(math.Abs(result.MetricsA.F1Score-result.MetricsB.F1Score)/math.Max(result.MetricsA.F1Score, result.MetricsB.F1Score))*100,
			result.ConfidenceLevel*100)
	}

	return fmt.Sprintf("规则 %s 表现更优，但需要更多数据验证（F1: %.3f），当前置信度 %.1f%%",
		winnerName, winnerMetrics.F1Score, result.ConfidenceLevel*100)
}

func (g *MLRuleGenerator) GetRules() []*Rule {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return append([]*Rule{}, g.rules...)
}

func (g *MLRuleGenerator) GetScenarios() []*RiskScenario {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return append([]*RiskScenario{}, g.scenarios...)
}

func (g *MLRuleGenerator) AddScenario(scenario *RiskScenario) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.scenarios = append(g.scenarios, scenario)
}

func (g *MLRuleGenerator) GetFeatureStatistics(featureName string) (*FeatureStats, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats, ok := g.featureStats[featureName]
	return stats, ok
}

func (g *MLRuleGenerator) GetAllFeatureStatistics() map[string]*FeatureStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]*FeatureStats)
	for k, v := range g.featureStats {
		result[k] = v
	}
	return result
}

func (g *MLRuleGenerator) ConvertToEnhancedRule(rule *Rule) EnhancedRule {
	return EnhancedRule{
		Name:        rule.Name,
		Description: rule.Description,
		Category:    rule.Category,
		Weight:      rule.Weight,
		Priority:    rule.Priority,
		Severity:    rule.Severity,
		Enabled:     true,
		Condition: func(f *RuleEngineFeatures) bool {
			return rule.Condition(f)
		},
	}
}

func ConvertEnhancedRuleToMLRule(rule EnhancedRule) *Rule {
	return &Rule{
		Name:          rule.Name,
		Description:   rule.Description,
		Category:      rule.Category,
		Weight:        rule.Weight,
		Priority:      rule.Priority,
		Severity:      rule.Severity,
		Condition:     rule.Condition,
		GeneratedFrom: "converted",
		CreatedAt:     time.Now(),
	}
}

func GenerateHistoricalDataFromFeatures(features []*RuleEngineFeatures, labels []bool) []RiskData {
	if len(features) != len(labels) {
		return nil
	}

	data := make([]RiskData, len(features))
	for i, f := range features {
		data[i] = RiskData{
			Features:  f,
			IsBot:     labels[i],
			Timestamp: time.Now(),
			SessionID: fmt.Sprintf("session_%d", i),
			UserID:    fmt.Sprintf("user_%d", i),
		}
	}

	return data
}

type RuleComparison struct {
	Rule         *Rule
	Metrics      *RuleMetrics
	Rank         int
	Score        float64
}

func (g *MLRuleGenerator) CompareRules(rules []*Rule, testData []RiskData) ([]RuleComparison, error) {
	comparisons := make([]RuleComparison, 0)

	for i, rule := range rules {
		metrics, err := g.EvaluateRule(context.Background(), rule, testData)
		if err != nil {
			continue
		}

		score := metrics.F1Score*0.4 + metrics.Accuracy*0.3 + metrics.Precision*0.15 + metrics.Recall*0.15

		comparisons = append(comparisons, RuleComparison{
			Rule:    rule,
			Metrics: metrics,
			Rank:    i + 1,
			Score:   score,
		})
	}

	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].Score > comparisons[j].Score
	})

	for i := range comparisons {
		comparisons[i].Rank = i + 1
	}

	return comparisons, nil
}
