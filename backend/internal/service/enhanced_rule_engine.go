package service

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type EnhancedRule struct {
	Name          string
	Condition     func(*RuleEngineFeatures) bool
	Weight        float64
	Priority      int
	Description   string
	Category      string
	Severity      float64
	Enabled       bool
	LastTriggered time.Time
	TriggerCount  int
	ruleMapKey    string
}

type EnhancedRuleEngine struct {
	rules              []EnhancedRule
	ruleMap            map[string]*EnhancedRule
	categories         map[string][]string
	weights            map[string]float64
	threshold          float64
	mu                 sync.RWMutex
	performanceTracker *PerformanceTracker
}

type PerformanceTracker struct {
	EvaluationCount   int64
	AverageExecTime   float64
	RuleHitCounts     map[string]int64
	CategoryHitCounts map[string]int64
	mu                sync.Mutex
}

type RuleEngineFeatures struct {
	SliderFeatures   *SliderFeatures       `json:"slider_features,omitempty"`
	ClickFeatures    *ClickPatternAnalysis `json:"click_features,omitempty"`
	TimingFeatures   *TimingAnalysis       `json:"timing_features,omitempty"`
	AccuracyFeatures *AccuracyAnalysis     `json:"accuracy_features,omitempty"`

	PathEfficiency     float64 `json:"path_efficiency"`
	SpeedConsistency   float64 `json:"speed_consistency"`
	AverageSpeed       float64 `json:"average_speed"`
	MaxSpeed           float64 `json:"max_speed"`
	SpeedVariance      float64 `json:"speed_variance"`
	CurvatureAverage   float64 `json:"curvature_average"`
	CurvatureVariance  float64 `json:"curvature_variance"`
	DirectionChanges   int     `json:"direction_changes"`
	MicroCorrections   int     `json:"micro_corrections"`
	BacktrackCount     int     `json:"backtrack_count"`
	PauseCount         int     `json:"pause_count"`
	TotalPauseDuration float64 `json:"total_pause_duration"`
	HesitationTime     float64 `json:"hesitation_time"`
	ResponseTime       float64 `json:"response_time"`
	ClickRegularity    float64 `json:"click_regularity"`
	PositionEntropy    float64 `json:"position_entropy"`
	Accuracy           float64 `json:"accuracy"`
	ClusteringScore    float64 `json:"clustering_score"`
	JitterScore        float64 `json:"jitter_score"`
	SmoothnessScore    float64 `json:"smoothness_score"`
	HumanLikenessScore float64 `json:"human_likeness_score"`
	AnomalyScore       float64 `json:"anomaly_score"`
	MLScore            float64 `json:"ml_score"`
	FractalDimension   float64 `json:"fractal_dimension"`
	FourierFrequency   float64 `json:"fourier_frequency"`
}

type RuleEngineResult struct {
	TotalScore      float64            `json:"total_score"`
	CategoryScores  map[string]float64 `json:"category_scores"`
	TriggeredRules  []string           `json:"triggered_rules"`
	RuleScores      map[string]float64 `json:"rule_scores"`
	IsBot           bool               `json:"is_bot"`
	Confidence      float64            `json:"confidence"`
	RiskLevel       string             `json:"risk_level"`
	Recommendations []string           `json:"recommendations"`
	AnalysisTime    time.Duration      `json:"analysis_time"`
}

func NewEnhancedRuleEngine() *EnhancedRuleEngine {
	engine := &EnhancedRuleEngine{
		rules:      make([]EnhancedRule, 0),
		ruleMap:    make(map[string]*EnhancedRule),
		categories: make(map[string][]string),
		weights:    make(map[string]float64),
		threshold:  0.5,
		performanceTracker: &PerformanceTracker{
			RuleHitCounts:     make(map[string]int64),
			CategoryHitCounts: make(map[string]int64),
		},
	}

	engine.initializeRules()
	engine.initializeWeights()

	return engine
}

func (ere *EnhancedRuleEngine) initializeRules() {
	ere.rules = []EnhancedRule{
		{
			Name:        "extreme_speed",
			Category:    "speed",
			Description: "检测到极端高速移动",
			Severity:    0.9,
			Weight:      30,
			Priority:    1,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.AverageSpeed > 2000
			},
		},
		{
			Name:        "very_high_speed",
			Category:    "speed",
			Description: "检测到非常高速移动",
			Severity:    0.7,
			Weight:      25,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.AverageSpeed > 1500 && f.AverageSpeed <= 2000
			},
		},
		{
			Name:        "high_speed",
			Category:    "speed",
			Description: "检测到高速移动",
			Severity:    0.5,
			Weight:      20,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.AverageSpeed > 1000 && f.AverageSpeed <= 1500
			},
		},
		{
			Name:        "perfect_path_efficiency",
			Category:    "trajectory",
			Description: "路径效率过高，接近完美直线",
			Severity:    0.95,
			Weight:      35,
			Priority:    1,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.PathEfficiency > 0.98
			},
		},
		{
			Name:        "high_path_efficiency",
			Category:    "trajectory",
			Description: "路径效率过高",
			Severity:    0.7,
			Weight:      25,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.PathEfficiency > 0.95 && f.PathEfficiency <= 0.98
			},
		},
		{
			Name:        "perfect_speed_consistency",
			Category:    "speed",
			Description: "速度过于恒定，缺乏自然变化",
			Severity:    0.85,
			Weight:      30,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.SpeedConsistency > 0.98
			},
		},
		{
			Name:        "very_high_speed_consistency",
			Category:    "speed",
			Description: "速度一致性过高",
			Severity:    0.6,
			Weight:      20,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.SpeedConsistency > 0.95 && f.SpeedConsistency <= 0.98
			},
		},
		{
			Name:        "no_micro_corrections",
			Category:    "trajectory",
			Description: "轨迹无微修正动作",
			Severity:    0.75,
			Weight:      25,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.MicroCorrections == 0 && f.AverageSpeed > 100
			},
		},
		{
			Name:        "no_pauses",
			Category:    "behavior",
			Description: "长时间操作无任何停顿",
			Severity:    0.7,
			Weight:      22,
			Priority:    4,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.PauseCount == 0 && f.ResponseTime > 1000
			},
		},
		{
			Name:        "low_curvature",
			Category:    "trajectory",
			Description: "轨迹曲率过低",
			Severity:    0.65,
			Weight:      20,
			Priority:    4,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.CurvatureAverage < 0.02
			},
		},
		{
			Name:        "high_curvature_variance",
			Category:    "trajectory",
			Description: "曲率变化过大",
			Severity:    0.5,
			Weight:      15,
			Priority:    5,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.CurvatureVariance > 0.5
			},
		},
		{
			Name:        "many_direction_changes",
			Category:    "trajectory",
			Description: "方向变化过于频繁",
			Severity:    0.6,
			Weight:      18,
			Priority:    5,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.DirectionChanges > 30
			},
		},
		{
			Name:        "few_direction_changes",
			Category:    "trajectory",
			Description: "方向变化过少",
			Severity:    0.55,
			Weight:      15,
			Priority:    5,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.DirectionChanges < 3 && f.AverageSpeed > 100
			},
		},
		{
			Name:        "backtrack_detected",
			Category:    "trajectory",
			Description: "检测到回退行为",
			Severity:    0.5,
			Weight:      12,
			Priority:    6,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.BacktrackCount > 3
			},
		},
		{
			Name:        "very_short_hesitation",
			Category:    "behavior",
			Description: "点击前犹豫时间过短",
			Severity:    0.7,
			Weight:      25,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.HesitationTime < 50 && f.ClickRegularity > 0
			},
		},
		{
			Name:        "perfect_click_regularity",
			Category:    "click",
			Description: "点击间隔过于规律",
			Severity:    0.8,
			Weight:      28,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.ClickRegularity > 0.98
			},
		},
		{
			Name:        "high_click_regularity",
			Category:    "click",
			Description: "点击间隔规律性过高",
			Severity:    0.6,
			Weight:      20,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.ClickRegularity > 0.95 && f.ClickRegularity <= 0.98
			},
		},
		{
			Name:        "low_position_entropy",
			Category:    "click",
			Description: "点击位置熵过低",
			Severity:    0.65,
			Weight:      22,
			Priority:    4,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.PositionEntropy < 1.5
			},
		},
		{
			Name:        "high_clustering",
			Category:    "click",
			Description: "点击位置过于集中",
			Severity:    0.55,
			Weight:      18,
			Priority:    4,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.ClusteringScore < 0.2
			},
		},
		{
			Name:        "perfect_accuracy",
			Category:    "accuracy",
			Description: "完美命中所有目标",
			Severity:    0.75,
			Weight:      25,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.Accuracy >= 1.0 && f.ClickRegularity > 0
			},
		},
		{
			Name:        "low_jitter",
			Category:    "trajectory",
			Description: "轨迹抖动过低",
			Severity:    0.7,
			Weight:      23,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.JitterScore < 0.01
			},
		},
		{
			Name:        "low_smoothness",
			Category:    "trajectory",
			Description: "轨迹平滑度过低",
			Severity:    0.45,
			Weight:      15,
			Priority:    5,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.SmoothnessScore < 0.3
			},
		},
		{
			Name:        "high_smoothness",
			Category:    "trajectory",
			Description: "轨迹平滑度过高",
			Severity:    0.7,
			Weight:      22,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.SmoothnessScore > 0.95
			},
		},
		{
			Name:        "low_human_likeness",
			Category:    "general",
			Description: "整体行为不像人类",
			Severity:    0.85,
			Weight:      30,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.HumanLikenessScore < 0.2
			},
		},
		{
			Name:        "very_low_human_likeness",
			Category:    "general",
			Description: "整体行为非常不像人类",
			Severity:    0.95,
			Weight:      35,
			Priority:    1,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.HumanLikenessScore < 0.1
			},
		},
		{
			Name:        "high_anomaly_score",
			Category:    "general",
			Description: "异常检测分数过高",
			Severity:    0.8,
			Weight:      28,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.AnomalyScore > 0.7
			},
		},
		{
			Name:        "very_high_anomaly_score",
			Category:    "general",
			Description: "异常检测分数异常高",
			Severity:    0.9,
			Weight:      32,
			Priority:    1,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.AnomalyScore > 0.85
			},
		},
		{
			Name:        "high_ml_score",
			Category:    "ml",
			Description: "机器学习模型判定为机器人",
			Severity:    0.85,
			Weight:      30,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.MLScore > 0.7
			},
		},
		{
			Name:        "very_high_ml_score",
			Category:    "ml",
			Description: "机器学习模型高度确信为机器人",
			Severity:    0.95,
			Weight:      35,
			Priority:    1,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.MLScore > 0.85
			},
		},
		{
			Name:        "low_fractal_dimension",
			Category:    "trajectory",
			Description: "分形维数过低，轨迹过于简单",
			Severity:    0.7,
			Weight:      24,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.FractalDimension < 1.1
			},
		},
		{
			Name:        "abnormal_fourier_frequency",
			Category:    "trajectory",
			Description: "傅里叶频率异常",
			Severity:    0.6,
			Weight:      18,
			Priority:    4,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.FourierFrequency > 10 || f.FourierFrequency < 0.1
			},
		},
		{
			Name:        "long_pause_duration",
			Category:    "behavior",
			Description: "单次停顿时间过长",
			Severity:    0.4,
			Weight:      12,
			Priority:    6,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.TotalPauseDuration > 5000
			},
		},
		{
			Name:        "many_pauses",
			Category:    "behavior",
			Description: "停顿次数过多",
			Severity:    0.45,
			Weight:      14,
			Priority:    5,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.PauseCount > 20
			},
		},
		{
			Name:        "very_low_accuracy",
			Category:    "accuracy",
			Description: "准确率过低",
			Severity:    0.5,
			Weight:      15,
			Priority:    5,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.Accuracy < 0.3 && f.Accuracy > 0
			},
		},
		{
			Name:        "speed_variance_too_low",
			Category:    "speed",
			Description: "速度方差过低",
			Severity:    0.7,
			Weight:      23,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.SpeedVariance < 0.001
			},
		},
		{
			Name:        "speed_variance_too_high",
			Category:    "speed",
			Description: "速度方差过高",
			Severity:    0.45,
			Weight:      14,
			Priority:    5,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.SpeedVariance > 1.0
			},
		},
		{
			Name:        "max_speed_too_high",
			Category:    "speed",
			Description: "最大速度过高",
			Severity:    0.75,
			Weight:      25,
			Priority:    2,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				return f.MaxSpeed > 3000
			},
		},
		{
			Name:        "combined_high_risk",
			Category:    "combined",
			Description: "多个高风险指标同时出现",
			Severity:    0.9,
			Weight:      40,
			Priority:    1,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				highRiskCount := 0
				if f.PathEfficiency > 0.95 {
					highRiskCount++
				}
				if f.SpeedConsistency > 0.95 {
					highRiskCount++
				}
				if f.ClickRegularity > 0.95 {
					highRiskCount++
				}
				if f.HumanLikenessScore < 0.3 {
					highRiskCount++
				}
				return highRiskCount >= 3
			},
		},
		{
			Name:        "combined_moderate_risk",
			Category:    "combined",
			Description: "多个中等风险指标同时出现",
			Severity:    0.6,
			Weight:      25,
			Priority:    3,
			Enabled:     true,
			Condition: func(f *RuleEngineFeatures) bool {
				moderateRiskCount := 0
				if f.PathEfficiency > 0.9 && f.PathEfficiency <= 0.95 {
					moderateRiskCount++
				}
				if f.SpeedConsistency > 0.9 && f.SpeedConsistency <= 0.95 {
					moderateRiskCount++
				}
				if f.ClickRegularity > 0.9 && f.ClickRegularity <= 0.95 {
					moderateRiskCount++
				}
				if f.HumanLikenessScore >= 0.3 && f.HumanLikenessScore < 0.5 {
					moderateRiskCount++
				}
				return moderateRiskCount >= 3
			},
		},
	}

	for i := range ere.rules {
		ere.rules[i].ruleMapKey = ere.rules[i].Name
		ere.ruleMap[ere.rules[i].Name] = &ere.rules[i]
		ere.categories[ere.rules[i].Category] = append(
			ere.categories[ere.rules[i].Category],
			ere.rules[i].Name,
		)
	}
}

func (ere *EnhancedRuleEngine) initializeWeights() {
	ere.weights = map[string]float64{
		"speed":      0.20,
		"trajectory": 0.25,
		"behavior":   0.15,
		"click":      0.15,
		"accuracy":   0.10,
		"general":    0.10,
		"ml":         0.05,
		"combined":   0.05,
	}
}

func (ere *EnhancedRuleEngine) Evaluate(features *RuleEngineFeatures) *RuleEngineResult {
	startTime := time.Now()

	if features == nil {
		return &RuleEngineResult{
			IsBot:      false,
			Confidence: 0,
			RiskLevel:  "unknown",
		}
	}

	result := &RuleEngineResult{
		CategoryScores:  make(map[string]float64),
		TriggeredRules:  make([]string, 0),
		RuleScores:      make(map[string]float64),
		Recommendations: make([]string, 0),
	}

	categoryScores := make(map[string]float64)
	categoryWeights := make(map[string]float64)

	ere.mu.RLock()
	defer ere.mu.RUnlock()

	for i := range ere.rules {
		rule := &ere.rules[i]
		if !rule.Enabled {
			continue
		}

		if rule.Condition(features) {
			rule.LastTriggered = time.Now()
			rule.TriggerCount++

			result.TriggeredRules = append(result.TriggeredRules, rule.Name)
			result.RuleScores[rule.Name] = rule.Weight * rule.Severity

			categoryScores[rule.Category] += rule.Weight * rule.Severity
			categoryWeights[rule.Category] += rule.Weight

			ere.performanceTracker.RuleHitCounts[rule.Name]++
			ere.performanceTracker.CategoryHitCounts[rule.Category]++
		}
	}

	totalScore := 0.0
	totalWeight := 0.0

	for category, score := range categoryScores {
		weight := ere.weights[category]
		if categoryWeights[category] > 0 {
			normalizedScore := score / categoryWeights[category]
			result.CategoryScores[category] = normalizedScore
			totalScore += normalizedScore * weight
			totalWeight += weight
		}
	}

	if totalWeight > 0 {
		result.TotalScore = totalScore / totalWeight
	}

	if features.MLScore > 0 {
		result.TotalScore = result.TotalScore*0.7 + features.MLScore*0.3
	}

	if features.AnomalyScore > 0 {
		result.TotalScore = result.TotalScore*0.8 + features.AnomalyScore*0.2
	}

	result.TotalScore = math.Min(math.Max(result.TotalScore, 0), 1)

	result.IsBot = result.TotalScore > ere.threshold
	result.Confidence = ere.calculateConfidence(result)
	result.RiskLevel = ere.classifyRiskLevel(result.TotalScore)

	result.AnalysisTime = time.Since(startTime)

	ere.performanceTracker.mu.Lock()
	ere.performanceTracker.EvaluationCount++
	ere.performanceTracker.AverageExecTime =
		(ere.performanceTracker.AverageExecTime*float64(ere.performanceTracker.EvaluationCount-1) +
			float64(result.AnalysisTime.Microseconds())) /
			float64(ere.performanceTracker.EvaluationCount)
	ere.performanceTracker.mu.Unlock()

	result.Recommendations = ere.generateRecommendations(result)

	return result
}

func (ere *EnhancedRuleEngine) calculateConfidence(result *RuleEngineResult) float64 {
	confidence := 0.7

	if len(result.TriggeredRules) >= 5 {
		confidence += 0.1
	}

	if len(result.TriggeredRules) >= 10 {
		confidence += 0.1
	}

	highSeverityCount := 0
	for _, ruleName := range result.TriggeredRules {
		if rule, ok := ere.ruleMap[ruleName]; ok {
			if rule.Severity > 0.8 {
				highSeverityCount++
			}
		}
	}

	if highSeverityCount >= 2 {
		confidence += 0.1
	}

	if len(result.CategoryScores) >= 3 {
		confidence += 0.05
	}

	return math.Min(confidence, 0.99)
}

func (ere *EnhancedRuleEngine) classifyRiskLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "critical"
	case score >= 0.6:
		return "high"
	case score >= 0.4:
		return "medium"
	case score >= 0.2:
		return "low"
	default:
		return "minimal"
	}
}

func (ere *EnhancedRuleEngine) generateRecommendations(result *RuleEngineResult) []string {
	recommendations := make([]string, 0)

	if result.TotalScore > 0.7 {
		recommendations = append(recommendations, "建议增加额外的验证步骤")
	}

	if result.CategoryScores["speed"] > 0.6 {
		recommendations = append(recommendations, "检测到异常速度特征，建议人工审核")
	}

	if result.CategoryScores["trajectory"] > 0.6 {
		recommendations = append(recommendations, "轨迹特征异常，建议进行深度分析")
	}

	if result.CategoryScores["click"] > 0.5 {
		recommendations = append(recommendations, "点击模式异常，建议增加验证难度")
	}

	if len(result.TriggeredRules) > 15 {
		recommendations = append(recommendations, "触发多条规则，建议直接拒绝访问")
	}

	return recommendations
}

func (ere *EnhancedRuleEngine) AddRule(rule EnhancedRule) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	for i, existingRule := range ere.rules {
		if existingRule.Name == rule.Name {
			ere.rules[i] = rule
			ere.ruleMap[rule.Name] = &ere.rules[i]
			return
		}
	}

	rule.ruleMapKey = rule.Name
	ere.rules = append(ere.rules, rule)
	newRule := &ere.rules[len(ere.rules)-1]
	ere.ruleMap[rule.Name] = newRule
	ere.categories[rule.Category] = append(ere.categories[rule.Category], rule.Name)
}

func (ere *EnhancedRuleEngine) RemoveRule(name string) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	for i, rule := range ere.rules {
		if rule.Name == name {
			ere.rules = append(ere.rules[:i], ere.rules[i+1:]...)
			delete(ere.ruleMap, name)

			categoryRules := ere.categories[rule.Category]
			for j, ruleName := range categoryRules {
				if ruleName == name {
					ere.categories[rule.Category] = append(
						categoryRules[:j],
						categoryRules[j+1:]...,
					)
					break
				}
			}
			break
		}
	}
}

func (ere *EnhancedRuleEngine) EnableRule(name string) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	if rule, ok := ere.ruleMap[name]; ok {
		rule.Enabled = true
	}
}

func (ere *EnhancedRuleEngine) DisableRule(name string) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	if rule, ok := ere.ruleMap[name]; ok {
		rule.Enabled = false
	}
}

func (ere *EnhancedRuleEngine) SetThreshold(threshold float64) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	ere.threshold = math.Max(0, math.Min(1, threshold))
}

func (ere *EnhancedRuleEngine) SetCategoryWeight(category string, weight float64) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	ere.weights[category] = math.Max(0, math.Min(1, weight))
}

func (ere *EnhancedRuleEngine) GetTriggeredRules(features *RuleEngineFeatures) []string {
	triggered := make([]string, 0)

	ere.mu.RLock()
	defer ere.mu.RUnlock()

	for _, rule := range ere.rules {
		if rule.Enabled && rule.Condition(features) {
			triggered = append(triggered, rule.Name)
		}
	}

	return triggered
}

func (ere *EnhancedRuleEngine) GetRulesByCategory(category string) []EnhancedRule {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	rules := make([]EnhancedRule, 0)
	for _, rule := range ere.rules {
		if rule.Category == category {
			rules = append(rules, rule)
		}
	}
	return rules
}

func (ere *EnhancedRuleEngine) GetAllRules() []EnhancedRule {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	return append([]EnhancedRule{}, ere.rules...)
}

func (ere *EnhancedRuleEngine) GetCategories() []string {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	categories := make([]string, 0, len(ere.categories))
	for category := range ere.categories {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}

func (ere *EnhancedRuleEngine) GetPerformanceStats() map[string]interface{} {
	ere.performanceTracker.mu.Lock()
	defer ere.performanceTracker.mu.Unlock()

	stats := map[string]interface{}{
		"evaluation_count":    ere.performanceTracker.EvaluationCount,
		"average_exec_time":   ere.performanceTracker.AverageExecTime,
		"rule_hit_counts":     ere.performanceTracker.RuleHitCounts,
		"category_hit_counts": ere.performanceTracker.CategoryHitCounts,
	}

	return stats
}

func (ere *EnhancedRuleEngine) ResetPerformanceStats() {
	ere.performanceTracker.mu.Lock()
	defer ere.performanceTracker.mu.Unlock()

	ere.performanceTracker.EvaluationCount = 0
	ere.performanceTracker.AverageExecTime = 0

	for key := range ere.performanceTracker.RuleHitCounts {
		ere.performanceTracker.RuleHitCounts[key] = 0
	}
	for key := range ere.performanceTracker.CategoryHitCounts {
		ere.performanceTracker.CategoryHitCounts[key] = 0
	}
}

func (ere *EnhancedRuleEngine) ExportRules() string {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("=== 增强版规则引擎导出 ===\n\n")

	sb.WriteString(fmt.Sprintf("阈值: %.2f\n", ere.threshold))
	sb.WriteString(fmt.Sprintf("规则总数: %d\n\n", len(ere.rules)))

	for category, rules := range ere.categories {
		sb.WriteString(fmt.Sprintf("\n[%s] (%d 条规则)\n", category, len(rules)))
		for _, ruleName := range rules {
			if rule, ok := ere.ruleMap[ruleName]; ok {
				sb.WriteString(fmt.Sprintf("  - %s\n", rule.Name))
				sb.WriteString(fmt.Sprintf("    描述: %s\n", rule.Description))
				sb.WriteString(fmt.Sprintf("    权重: %.2f, 严重度: %.2f\n", rule.Weight, rule.Severity))
				sb.WriteString(fmt.Sprintf("    优先级: %d, 启用: %v\n", rule.Priority, rule.Enabled))
				sb.WriteString(fmt.Sprintf("    触发次数: %d\n", rule.TriggerCount))
				if !rule.LastTriggered.IsZero() {
					sb.WriteString(fmt.Sprintf("    上次触发: %s\n", rule.LastTriggered.Format(time.RFC3339)))
				}
			}
		}
	}

	return sb.String()
}

func CreateEnhancedFeaturesFromSliderResult(result *SliderAnalysisResult) *RuleEngineFeatures {
	if result == nil {
		return &RuleEngineFeatures{}
	}

	features := &RuleEngineFeatures{
		SliderFeatures: result.Features,
		PathEfficiency: result.Trajectory.PathEfficiency,
		AverageSpeed:   result.Trajectory.AverageSpeed,
		MaxSpeed:       result.Trajectory.MaxSpeed,
		SpeedVariance:  result.Trajectory.SpeedVariance,
		AnomalyScore:   result.AnomalyScore,
		MLScore:        result.MLScore,
	}

	if result.Features != nil {
		features.SpeedConsistency = result.Features.SpeedConsistency
		features.CurvatureAverage = result.Features.CurvatureAverage
		features.CurvatureVariance = result.Features.CurvatureVariance
		features.DirectionChanges = result.Features.DirectionChanges
		features.MicroCorrections = result.Features.MicroCorrections
		features.BacktrackCount = result.Features.BacktrackCount
		features.PauseCount = result.Features.PauseCount
		features.TotalPauseDuration = result.Features.TotalPauseDuration
		features.ResponseTime = float64(result.Features.ResponseTime)
		features.JitterScore = result.Features.JitterScore
		features.SmoothnessScore = result.Features.SmoothnessScore
		features.HumanLikenessScore = result.Features.HumanLikenessScore
		features.FractalDimension = result.Features.FractalDimension
		features.FourierFrequency = result.Features.FourierFrequency
	}

	return features
}

func CreateEnhancedFeaturesFromClickResult(result *ClickAnalysisResult) *RuleEngineFeatures {
	if result == nil {
		return &RuleEngineFeatures{}
	}

	features := &RuleEngineFeatures{
		ClickFeatures:    result.ClickPattern,
		TimingFeatures:   result.TimingAnalysis,
		AccuracyFeatures: result.AccuracyAnalysis,
		AnomalyScore:     result.AnomalyScore,
		MLScore:          result.MLScore,
	}

	if result.ClickPattern != nil {
		features.ClickRegularity = result.ClickPattern.Regularity
		features.PositionEntropy = (result.ClickPattern.PositionDistribution.XEntropy +
			result.ClickPattern.PositionDistribution.YEntropy) / 2
		features.ClusteringScore = result.ClickPattern.ClusteringScore
	}

	if result.TimingAnalysis != nil {
		features.ResponseTime = float64(result.TimingAnalysis.TotalDuration)
		features.HesitationTime = result.TimingAnalysis.AverageDuration
	}

	if result.AccuracyAnalysis != nil {
		features.Accuracy = result.AccuracyAnalysis.Accuracy
	}

	return features
}

func CombineEnhancedFeatures(features ...*RuleEngineFeatures) *RuleEngineFeatures {
	combined := &RuleEngineFeatures{}

	for _, f := range features {
		if f == nil {
			continue
		}

		if f.PathEfficiency > 0 {
			combined.PathEfficiency = f.PathEfficiency
		}
		if f.AverageSpeed > 0 {
			combined.AverageSpeed = f.AverageSpeed
		}
		if f.MaxSpeed > 0 {
			combined.MaxSpeed = f.MaxSpeed
		}
		if f.SpeedVariance >= 0 {
			combined.SpeedVariance = f.SpeedVariance
		}
		if f.SpeedConsistency >= 0 {
			combined.SpeedConsistency = f.SpeedConsistency
		}
		if f.CurvatureAverage >= 0 {
			combined.CurvatureAverage = f.CurvatureAverage
		}
		if f.CurvatureVariance >= 0 {
			combined.CurvatureVariance = f.CurvatureVariance
		}
		if f.DirectionChanges >= 0 {
			combined.DirectionChanges = f.DirectionChanges
		}
		if f.MicroCorrections >= 0 {
			combined.MicroCorrections = f.MicroCorrections
		}
		if f.BacktrackCount >= 0 {
			combined.BacktrackCount = f.BacktrackCount
		}
		if f.PauseCount >= 0 {
			combined.PauseCount = f.PauseCount
		}
		if f.TotalPauseDuration >= 0 {
			combined.TotalPauseDuration = f.TotalPauseDuration
		}
		if f.HesitationTime >= 0 {
			combined.HesitationTime = f.HesitationTime
		}
		if f.ResponseTime >= 0 {
			combined.ResponseTime = f.ResponseTime
		}
		if f.ClickRegularity >= 0 {
			combined.ClickRegularity = f.ClickRegularity
		}
		if f.PositionEntropy >= 0 {
			combined.PositionEntropy = f.PositionEntropy
		}
		if f.Accuracy >= 0 {
			combined.Accuracy = f.Accuracy
		}
		if f.ClusteringScore >= 0 {
			combined.ClusteringScore = f.ClusteringScore
		}
		if f.JitterScore >= 0 {
			combined.JitterScore = f.JitterScore
		}
		if f.SmoothnessScore >= 0 {
			combined.SmoothnessScore = f.SmoothnessScore
		}
		if f.HumanLikenessScore >= 0 {
			combined.HumanLikenessScore = f.HumanLikenessScore
		}
		if f.AnomalyScore >= 0 {
			combined.AnomalyScore = math.Max(combined.AnomalyScore, f.AnomalyScore)
		}
		if f.MLScore >= 0 {
			combined.MLScore = math.Max(combined.MLScore, f.MLScore)
		}
		if f.FractalDimension > 0 {
			combined.FractalDimension = f.FractalDimension
		}
		if f.FourierFrequency >= 0 {
			combined.FourierFrequency = f.FourierFrequency
		}
	}

	return combined
}

type EngineRuleVersion struct {
	Version     string         `json:"version"`
	ChangeType  string         `json:"change_type"`
	Description string         `json:"description"`
	Operator    string         `json:"operator"`
	CreatedAt   time.Time      `json:"created_at"`
	Rules       []EnhancedRule `json:"rules"`
	IsCurrent   bool           `json:"is_current"`
}

type RuleEngineConfig struct {
	Version           string                  `json:"version"`
	Threshold         float64                 `json:"threshold"`
	CategoryWeights   map[string]float64      `json:"category_weights"`
	Rules             []EnhancedRule         `json:"rules"`
}

func (ere *EnhancedRuleEngine) ExportToConfig() *RuleEngineConfig {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	return &RuleEngineConfig{
		Version:         fmt.Sprintf("v%d.%d.%d", time.Now().Year(), int(time.Now().Month()), time.Now().Day()),
		Threshold:       ere.threshold,
		CategoryWeights: ere.weights,
		Rules:           append([]EnhancedRule{}, ere.rules...),
	}
}

func (ere *EnhancedRuleEngine) LoadFromConfig(config *RuleEngineConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	ere.mu.Lock()
	defer ere.mu.Unlock()

	if config.Threshold > 0 {
		ere.threshold = config.Threshold
	}

	if config.CategoryWeights != nil {
		for category, weight := range config.CategoryWeights {
			ere.weights[category] = weight
		}
	}

	if config.Rules != nil {
		ere.rules = make([]EnhancedRule, 0, len(config.Rules))
		ere.ruleMap = make(map[string]*EnhancedRule)
		ere.categories = make(map[string][]string)

		for _, rule := range config.Rules {
			rule.ruleMapKey = rule.Name
			ere.rules = append(ere.rules, rule)
			newRule := &ere.rules[len(ere.rules)-1]
			ere.ruleMap[rule.Name] = newRule
			ere.categories[rule.Category] = append(ere.categories[rule.Category], rule.Name)
		}
	}

	return nil
}

func (ere *EnhancedRuleEngine) ReloadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config RuleEngineConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return ere.LoadFromConfig(&config)
}

func (ere *EnhancedRuleEngine) SaveToFile(filePath string) error {
	config := ere.ExportToConfig()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

func (ere *EnhancedRuleEngine) CreateVersionSnapshot() *EngineRuleVersion {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	version := fmt.Sprintf("v%d.%d.%d.%d",
		time.Now().Year(),
		int(time.Now().Month()),
		time.Now().Day(),
		time.Now().Hour()*100+time.Now().Minute(),
	)

	return &EngineRuleVersion{
		Version:     version,
		ChangeType:  "snapshot",
		Description: "自动快照",
		Operator:    "system",
		CreatedAt:   time.Now(),
		Rules:       append([]EnhancedRule{}, ere.rules...),
		IsCurrent:   true,
	}
}

func (ere *EnhancedRuleEngine) LoadVersionSnapshot(version *EngineRuleVersion) error {
	if version == nil || version.Rules == nil {
		return fmt.Errorf("版本快照无效")
	}

	ere.mu.Lock()
	defer ere.mu.Unlock()

	ere.rules = append([]EnhancedRule{}, version.Rules...)
	ere.ruleMap = make(map[string]*EnhancedRule)
	ere.categories = make(map[string][]string)

	for i := range ere.rules {
		ere.rules[i].ruleMapKey = ere.rules[i].Name
		ere.ruleMap[ere.rules[i].Name] = &ere.rules[i]
		ere.categories[ere.rules[i].Category] = append(
			ere.categories[ere.rules[i].Category],
			ere.rules[i].Name,
		)
	}

	return nil
}

func (ere *EnhancedRuleEngine) AddRuleWithValidation(rule EnhancedRule) error {
	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}

	if _, exists := ere.ruleMap[rule.Name]; exists {
		return fmt.Errorf("规则 '%s' 已存在", rule.Name)
	}

	if rule.Weight < 0 {
		return fmt.Errorf("规则权重不能为负数")
	}

	if rule.Priority < 0 {
		return fmt.Errorf("规则优先级不能为负数")
	}

	ere.AddRule(rule)
	return nil
}

func (ere *EnhancedRuleEngine) UpdateRule(name string, updateFunc func(*EnhancedRule) error) error {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	rule, exists := ere.ruleMap[name]
	if !exists {
		return fmt.Errorf("规则 '%s' 不存在", name)
	}

	if err := updateFunc(rule); err != nil {
		return err
	}

	rule.ruleMapKey = rule.Name

	for i, r := range ere.rules {
		if r.Name == name {
			ere.rules[i] = *rule
			break
		}
	}

	return nil
}

func (ere *EnhancedRuleEngine) ValidateRule(rule EnhancedRule) []string {
	var errors []string

	if rule.Name == "" {
		errors = append(errors, "规则名称不能为空")
	}

	if rule.Category == "" {
		errors = append(errors, "规则分类不能为空")
	}

	if rule.Weight < 0 {
		errors = append(errors, "规则权重不能为负数")
	}

	if rule.Severity < 0 || rule.Severity > 1 {
		errors = append(errors, "规则严重度必须在0-1之间")
	}

	if rule.Condition == nil {
		errors = append(errors, "规则条件不能为空")
	}

	return errors
}

func (ere *EnhancedRuleEngine) GetRuleByName(name string) (*EnhancedRule, bool) {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	rule, exists := ere.ruleMap[name]
	if !exists {
		return nil, false
	}

	return &EnhancedRule{
		Name:          rule.Name,
		Category:      rule.Category,
		Description:   rule.Description,
		Weight:        rule.Weight,
		Priority:      rule.Priority,
		Severity:      rule.Severity,
		Enabled:       rule.Enabled,
		LastTriggered: rule.LastTriggered,
		TriggerCount:  rule.TriggerCount,
	}, true
}

func (ere *EnhancedRuleEngine) EnableCategory(category string) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	if rules, ok := ere.categories[category]; ok {
		for _, ruleName := range rules {
			if rule, exists := ere.ruleMap[ruleName]; exists {
				rule.Enabled = true
			}
		}
	}
}

func (ere *EnhancedRuleEngine) DisableCategory(category string) {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	if rules, ok := ere.categories[category]; ok {
		for _, ruleName := range rules {
			if rule, exists := ere.ruleMap[ruleName]; exists {
				rule.Enabled = false
			}
		}
	}
}

func (ere *EnhancedRuleEngine) GetEnabledRules() []EnhancedRule {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	var enabledRules []EnhancedRule
	for _, rule := range ere.rules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	return enabledRules
}

func (ere *EnhancedRuleEngine) GetRuleStats() map[string]interface{} {
	ere.mu.RLock()
	defer ere.mu.RUnlock()

	stats := map[string]interface{}{
		"total_rules":    len(ere.rules),
		"enabled_rules":  0,
		"disabled_rules": 0,
		"categories":     make(map[string]int),
		"top_triggered":  []map[string]interface{}{},
	}

	for _, rule := range ere.rules {
		if rule.Enabled {
			stats["enabled_rules"] = stats["enabled_rules"].(int) + 1
		} else {
			stats["disabled_rules"] = stats["disabled_rules"].(int) + 1
		}
		stats["categories"].(map[string]int)[rule.Category]++
	}

	type triggeredRule struct {
		Name         string
		TriggerCount int
	}
	var triggeredRules []triggeredRule
	for _, rule := range ere.rules {
		if rule.TriggerCount > 0 {
			triggeredRules = append(triggeredRules, triggeredRule{
				Name:         rule.Name,
				TriggerCount: rule.TriggerCount,
			})
		}
	}

	sort.Slice(triggeredRules, func(i, j int) bool {
		return triggeredRules[i].TriggerCount > triggeredRules[j].TriggerCount
	})

	var topTriggered []map[string]interface{}
	minLen := 10
	if len(triggeredRules) < minLen {
		minLen = len(triggeredRules)
	}
	for i := 0; i < minLen; i++ {
		topTriggered = append(topTriggered, map[string]interface{}{
			"name":          triggeredRules[i].Name,
			"trigger_count": triggeredRules[i].TriggerCount,
		})
	}
	stats["top_triggered"] = topTriggered

	return stats
}

func (ere *EnhancedRuleEngine) ResetTriggerCounts() {
	ere.mu.Lock()
	defer ere.mu.Unlock()

	for i := range ere.rules {
		ere.rules[i].TriggerCount = 0
		ere.rules[i].LastTriggered = time.Time{}
	}
}
