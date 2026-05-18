package trace

import (
	"errors"
	"math"

	"github.com/hjtpx/hjtpx/internal/model"
)

type IntentType string

const (
	IntentNormalUser    IntentType = "normal_user"
	IntentCarefulUser   IntentType = "careful_user"
	IntentImpatientUser IntentType = "impatient_user"
	IntentAutomatedBot  IntentType = "automated_bot"
	IntentHumanLikeBot  IntentType = "human_like_bot"
	IntentScriptedBot   IntentType = "scripted_bot"
	IntentAggressiveBot IntentType = "aggressive_bot"
	IntentUnknown       IntentType = "unknown"
)

type IntentRecognitionResult struct {
	PrimaryIntent     IntentType
	IntentProbabilities map[IntentType]float64
	Confidence        float64
	IntentFeatures    map[string]float64
	BehavioralIndicators []string
	Recommendations   []string
	Reasoning         string
}

type IntentClassifier struct {
	featureExtractor *LSTMFeatureExtractor
	rules           []IntentRule
	confidenceThresholds map[IntentType]float64
}

type IntentRule struct {
	Name        string
	Conditions  []IntentCondition
	IntentType  IntentType
	Weight      float64
	Description string
}

type IntentCondition struct {
	FeatureName string
	Operator    string
	Value       float64
}

type IntentFeatures struct {
	TimeEfficiency      float64
	Precision           float64
	Consistency         float64
	Naturalness         float64
	Purposefulness      float64
	Adaptability        float64
	RhythmRegularity    float64
	PatienceLevel       float64
	ResponseSpeed       float64
	DecisionQuality     float64
}

func NewIntentClassifier() *IntentClassifier {
	classifier := &IntentClassifier{
		featureExtractor: NewLSTMFeatureExtractor(),
		rules:           make([]IntentRule, 0),
		confidenceThresholds: make(map[IntentType]float64),
	}

	classifier.initializeRules()
	classifier.initializeThresholds()

	return classifier
}

func (c *IntentClassifier) initializeRules() {
	c.rules = []IntentRule{
		{
			Name:       "完美直线规则",
			IntentType: IntentAutomatedBot,
			Weight:     0.9,
			Description: "轨迹呈现完美直线特征",
			Conditions: []IntentCondition{
				{FeatureName: "path_ratio", Operator: "<", Value: 1.02},
				{FeatureName: "total_distance", Operator: ">", Value: 100},
			},
		},
		{
			Name:       "恒定速度规则",
			IntentType: IntentAutomatedBot,
			Weight:     0.85,
			Description: "速度保持恒定",
			Conditions: []IntentCondition{
				{FeatureName: "speed_variance", Operator: "<", Value: 5},
				{FeatureName: "avg_speed", Operator: ">", Value: 50},
			},
		},
		{
			Name:       "无停顿规则",
			IntentType: IntentAutomatedBot,
			Weight:     0.7,
			Description: "轨迹无正常停顿",
			Conditions: []IntentCondition{
				{FeatureName: "pause_count", Operator: "==", Value: 0},
				{FeatureName: "total_time", Operator: ">", Value: 2000},
			},
		},
		{
			Name:       "高速移动规则",
			IntentType: IntentImpatientUser,
			Weight:     0.75,
			Description: "异常高的移动速度",
			Conditions: []IntentCondition{
				{FeatureName: "avg_speed", Operator: ">", Value: 500},
				{FeatureName: "speed_variance", Operator: ">", Value: 20},
			},
		},
		{
			Name:       "低速谨慎规则",
			IntentType: IntentCarefulUser,
			Weight:     0.8,
			Description: "移动速度较慢但稳定",
			Conditions: []IntentCondition{
				{FeatureName: "avg_speed", Operator: "<", Value: 100},
				{FeatureName: "pause_count", Operator: ">", Value: 3},
				{FeatureName: "smoothness", Operator: ">", Value: 0.1},
			},
		},
		{
			Name:       "自然流畅规则",
			IntentType: IntentNormalUser,
			Weight:     0.85,
			Description: "轨迹特征符合自然人类行为",
			Conditions: []IntentCondition{
				{FeatureName: "speed_variance", Operator: ">", Value: 10},
				{FeatureName: "pause_count", Operator: ">", Value: 1},
				{FeatureName: "path_ratio", Operator: ">", Value: 1.1},
			},
		},
		{
			Name:       "规律节奏规则",
			IntentType: IntentHumanLikeBot,
			Weight:     0.6,
			Description: "行为过于规律但不完全机械",
			Conditions: []IntentCondition{
				{FeatureName: "speed_variance", Operator: "between", Value: 5},
				{FeatureName: "acceleration_variance", Operator: "<", Value: 0.01},
			},
		},
		{
			Name:       "脚本化规则",
			IntentType: IntentScriptedBot,
			Weight:     0.9,
			Description: "明显的程序化行为特征",
			Conditions: []IntentCondition{
				{FeatureName: "curvature_variance", Operator: "<", Value: 0.001},
				{FeatureName: "direction_entropy", Operator: "<", Value: 1.0},
			},
		},
		{
			Name:       "激进行为规则",
			IntentType: IntentAggressiveBot,
			Weight:     0.85,
			Description: "异常快速和激进的行为",
			Conditions: []IntentCondition{
				{FeatureName: "max_speed", Operator: ">", Value: 1000},
				{FeatureName: "max_acceleration", Operator: ">", Value: 5000},
			},
		},
	}
}

func (c *IntentClassifier) initializeThresholds() {
	c.confidenceThresholds = map[IntentType]float64{
		IntentNormalUser:    0.7,
		IntentCarefulUser:   0.65,
		IntentImpatientUser: 0.65,
		IntentAutomatedBot:  0.75,
		IntentHumanLikeBot:  0.6,
		IntentScriptedBot:   0.7,
		IntentAggressiveBot: 0.7,
		IntentUnknown:       0.3,
	}
}

func (c *IntentClassifier) RecognizeIntent(traceData *model.TraceData) (*IntentRecognitionResult, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据不足")
	}

	result := &IntentRecognitionResult{
		IntentProbabilities: make(map[IntentType]float64),
		IntentFeatures:      make(map[string]float64),
		BehavioralIndicators: []string{},
		Recommendations:     []string{},
	}

	extractor := NewTraceExtractor()
	basicFeatures, err := extractor.ExtractFeatures(traceData)
	if err != nil {
		return nil, err
	}

	advancedFeatures, err := extractor.ExtractAdvancedFeatures(traceData)
	if err != nil {
		return nil, err
	}

	features := c.extractIntentFeatures(basicFeatures, advancedFeatures, traceData)

	for k, v := range features {
		result.IntentFeatures[k] = v
	}

	intentScores := c.calculateIntentScores(features)

	for intent, score := range intentScores {
		result.IntentProbabilities[intent] = score
	}

	result.PrimaryIntent = c.determinePrimaryIntent(intentScores)

	result.Confidence = c.calculateConfidence(intentScores, result.PrimaryIntent)

	result.BehavioralIndicators = c.extractBehavioralIndicators(features)

	result.Reasoning = c.generateReasoning(result.PrimaryIntent, features)

	result.Recommendations = c.generateRecommendations(result.PrimaryIntent, result.Confidence)

	return result, nil
}

func (c *IntentClassifier) extractIntentFeatures(
	basic *model.TraceFeatures,
	advanced *AdvancedFeatures,
	traceData *model.TraceData,
) map[string]float64 {
	features := make(map[string]float64)

	if basic != nil {
		features["avg_speed"] = basic.AvgSpeed
		features["max_speed"] = basic.MaxSpeed
		features["min_speed"] = basic.MinSpeed
		features["speed_variance"] = basic.SpeedVariance
		features["max_acceleration"] = basic.MaxAcceleration
		features["avg_acceleration"] = basic.AvgAcceleration
		features["accel_variance"] = basic.AccelVariance
		features["smoothness"] = basic.Smoothness
		features["pause_count"] = float64(basic.PauseCount)
		features["total_distance"] = basic.TotalDistance
		features["direct_distance"] = basic.DirectDistance
		features["path_ratio"] = basic.PathRatio
		features["avg_curvature"] = basic.AvgCurvature
		features["max_curvature"] = basic.MaxCurvature
		features["total_time"] = float64(basic.TotalTime)
		features["move_count"] = float64(basic.MoveCount)
		features["jitter_frequency"] = basic.JitterFrequency
		features["jitter_amplitude"] = basic.JitterAmplitude
		features["speed_change_rate"] = basic.SpeedChangeRate
		features["direction_change"] = basic.DirectionChange
	}

	if advanced != nil {
		features["median_speed"] = advanced.MedianSpeed
		features["speed_skewness"] = advanced.SpeedSkewness
		features["speed_kurtosis"] = advanced.SpeedKurtosis
		features["speed_entropy"] = advanced.SpeedEntropy
		features["acceleration_variance"] = advanced.AccelerationVariance
		features["acceleration_skewness"] = advanced.AccelerationSkewness
		features["jerk_mean"] = advanced.JerkMean
		features["jerk_max"] = advanced.JerkMax
		features["curvature_median"] = advanced.CurvatureMedian
		features["curvature_variance"] = advanced.CurvatureVariance
		features["curvature_max"] = advanced.CurvatureMax
		features["direction_change_rate"] = advanced.DirectionChangeRate
		features["direction_entropy"] = advanced.DirectionEntropy
		features["sinuosity"] = advanced.Sinuosity
		features["velocity_profile_entropy"] = advanced.VelocityProfileEntropy
		features["acceleration_profile_entropy"] = advanced.AccelerationProfileEntropy
	}

	features["time_efficiency"] = c.calculateTimeEfficiency(basic, traceData)
	features["precision"] = c.calculatePrecision(basic)
	features["consistency"] = c.calculateConsistency(basic, advanced)
	features["naturalness"] = c.calculateNaturalness(basic, advanced)
	features["purposefulness"] = c.calculatePurposefulness(basic)
	features["adaptability"] = c.calculateAdaptability(basic, advanced)
	features["rhythm_regularity"] = c.calculateRhythmRegularity(basic)
	features["patience_level"] = c.calculatePatienceLevel(basic)
	features["response_speed"] = c.calculateResponseSpeed(basic)
	features["decision_quality"] = c.calculateDecisionQuality(basic, advanced)

	return features
}

func (c *IntentClassifier) calculateTimeEfficiency(basic *model.TraceFeatures, traceData *model.TraceData) float64 {
	if basic == nil || basic.TotalDistance == 0 {
		return 0.5
	}

	efficiency := basic.DirectDistance / basic.TotalDistance

	if basic.TotalTime > 0 {
		timeRatio := float64(basic.TotalTime) / basic.TotalDistance
		if timeRatio > 1 {
			efficiency *= 0.8
		}
	}

	return math.Max(0, math.Min(1, efficiency))
}

func (c *IntentClassifier) calculatePrecision(basic *model.TraceFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	if basic.DirectDistance == 0 {
		return 0.5
	}

	precision := math.Min(1.0, basic.DirectDistance/basic.TotalDistance)

	return precision
}

func (c *IntentClassifier) calculateConsistency(basic *model.TraceFeatures, advanced *AdvancedFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	consistency := 0.5

	if advanced != nil {
		if advanced.SpeedVariance > 0 && advanced.SpeedVariance < 100 {
			consistency = 0.7
		} else if advanced.SpeedVariance >= 100 {
			consistency = 0.3
		}

		if advanced.AccelerationVariance > 0.001 {
			consistency += 0.2
		}
	}

	if basic.SpeedChangeRate < 100 {
		consistency += 0.1
	} else if basic.SpeedChangeRate > 500 {
		consistency -= 0.2
	}

	return math.Max(0, math.Min(1, consistency))
}

func (c *IntentClassifier) calculateNaturalness(basic *model.TraceFeatures, advanced *AdvancedFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	naturalness := 0.5

	if basic.PauseCount > 0 && basic.PauseCount < 10 {
		naturalness += 0.15
	}

	if basic.Smoothness > 0.1 {
		naturalness += 0.15
	}

	if basic.PathRatio > 1.1 && basic.PathRatio < 3.0 {
		naturalness += 0.1
	}

	if advanced != nil {
		if advanced.SpeedEntropy > 1.5 && advanced.SpeedEntropy < 3.0 {
			naturalness += 0.1
		}

		if advanced.DirectionEntropy > 1.5 {
			naturalness += 0.1
		}

		if advanced.Sinuosity > 1.05 && advanced.Sinuosity < 2.5 {
			naturalness += 0.1
		}
	}

	if basic.MaxAcceleration < 3000 {
		naturalness += 0.1
	}

	return math.Max(0, math.Min(1, naturalness))
}

func (c *IntentClassifier) calculatePurposefulness(basic *model.TraceFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	purposefulness := 0.5

	if basic.TotalDistance > 100 {
		purposefulness += 0.2
	}

	if basic.PathRatio < 2.0 {
		purposefulness += 0.15
	}

	if basic.DirectionChange < math.Pi {
		purposefulness += 0.1
	}

	return math.Max(0, math.Min(1, purposefulness))
}

func (c *IntentClassifier) calculateAdaptability(basic *model.TraceFeatures, advanced *AdvancedFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	adaptability := 0.5

	if advanced != nil {
		if advanced.SpeedVariance > 10 {
			adaptability += 0.2
		}

		if advanced.JerkMean > 0.001 {
			adaptability += 0.1
		}

		if advanced.CurvatureVariance > 0.001 {
			adaptability += 0.1
		}
	}

	if basic.JitterFrequency > 0.05 && basic.JitterFrequency < 0.3 {
		adaptability += 0.1
	}

	return math.Max(0, math.Min(1, adaptability))
}

func (c *IntentClassifier) calculateRhythmRegularity(basic *model.TraceFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	regularity := 0.5

	if basic.SpeedVariance < 5 {
		regularity += 0.3
	} else if basic.SpeedVariance > 50 {
		regularity -= 0.2
	}

	if basic.SpeedChangeRate < 50 {
		regularity += 0.2
	}

	return math.Max(0, math.Min(1, regularity))
}

func (c *IntentClassifier) calculatePatienceLevel(basic *model.TraceFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	patience := 0.5

	if basic.TotalTime > 3000 {
		patience += 0.2
	}

	if basic.PauseCount > 2 {
		patience += 0.2
	}

	if basic.AvgSpeed < 100 {
		patience += 0.1
	}

	return math.Max(0, math.Min(1, patience))
}

func (c *IntentClassifier) calculateResponseSpeed(basic *model.TraceFeatures) float64 {
	if basic == nil || basic.TotalTime == 0 {
		return 0.5
	}

	speed := 0.5

	if basic.TotalTime < 2000 {
		speed += 0.3
	} else if basic.TotalTime > 5000 {
		speed -= 0.2
	}

	if basic.AvgSpeed > 500 {
		speed += 0.2
	}

	return math.Max(0, math.Min(1, speed))
}

func (c *IntentClassifier) calculateDecisionQuality(basic *model.TraceFeatures, advanced *AdvancedFeatures) float64 {
	if basic == nil {
		return 0.5
	}

	quality := 0.5

	if advanced != nil {
		if advanced.SpeedEntropy > 2.0 {
			quality += 0.15
		}

		if advanced.DirectionEntropy > 1.5 {
			quality += 0.15
		}

		if advanced.Sinuosity > 1.1 && advanced.Sinuosity < 2.5 {
			quality += 0.1
		}
	}

	if basic.DirectionChange > 0.1 && basic.DirectionChange < math.Pi/2 {
		quality += 0.1
	}

	return math.Max(0, math.Min(1, quality))
}

func (c *IntentClassifier) calculateIntentScores(features map[string]float64) map[IntentType]float64 {
	scores := make(map[IntentType]float64)

	scores[IntentNormalUser] = c.calculateNormalUserScore(features)
	scores[IntentCarefulUser] = c.calculateCarefulUserScore(features)
	scores[IntentImpatientUser] = c.calculateImpatientUserScore(features)
	scores[IntentAutomatedBot] = c.calculateAutomatedBotScore(features)
	scores[IntentHumanLikeBot] = c.calculateHumanLikeBotScore(features)
	scores[IntentScriptedBot] = c.calculateScriptedBotScore(features)
	scores[IntentAggressiveBot] = c.calculateAggressiveBotScore(features)
	scores[IntentUnknown] = c.calculateUnknownScore(features)

	totalScore := 0.0
	for _, score := range scores {
		totalScore += score
	}

	if totalScore > 0 {
		for intent := range scores {
			scores[intent] /= totalScore
		}
	}

	return scores
}

func (c *IntentClassifier) calculateNormalUserScore(features map[string]float64) float64 {
	score := 0.0

	if v, ok := features["naturalness"]; ok {
		score += v * 0.4
	}

	if v, ok := features["pause_count"]; ok && v > 0 && v < 10 {
		score += 0.2
	}

	if v, ok := features["speed_variance"]; ok && v > 10 && v < 100 {
		score += 0.2
	}

	if v, ok := features["path_ratio"]; ok && v > 1.1 && v < 3.0 {
		score += 0.1
	}

	if v, ok := features["smoothness"]; ok && v > 0.1 {
		score += 0.1
	}

	return score
}

func (c *IntentClassifier) calculateCarefulUserScore(features map[string]float64) float64 {
	score := 0.0

	if v, ok := features["patience_level"]; ok {
		score += v * 0.3
	}

	if v, ok := features["pause_count"]; ok && v > 3 {
		score += 0.25
	}

	if v, ok := features["avg_speed"]; ok && v < 100 {
		score += 0.2
	}

	if v, ok := features["smoothness"]; ok && v > 0.15 {
		score += 0.15
	}

	if v, ok := features["decision_quality"]; ok {
		score += v * 0.1
	}

	return score
}

func (c *IntentClassifier) calculateImpatientUserScore(features map[string]float64) float64 {
	score := 0.0

	if v, ok := features["response_speed"]; ok {
		score += v * 0.3
	}

	if v, ok := features["avg_speed"]; ok && v > 300 {
		score += 0.25
	}

	if v, ok := features["pause_count"]; ok && v < 2 {
		score += 0.2
	}

	if v, ok := features["total_time"]; ok && v < 3000 {
		score += 0.15
	}

	if v, ok := features["patience_level"]; ok {
		score += (1 - v) * 0.1
	}

	return score
}

func (c *IntentClassifier) calculateAutomatedBotScore(features map[string]float64) float64 {
	score := 0.0

	if v, ok := features["path_ratio"]; ok && v < 1.02 {
		score += 0.3
	}

	if v, ok := features["speed_variance"]; ok && v < 5 {
		score += 0.25
	}

	if v, ok := features["pause_count"]; ok && v == 0 {
		score += 0.2
	}

	if v, ok := features["rhythm_regularity"]; ok && v > 0.7 {
		score += 0.15
	}

	if v, ok := features["naturalness"]; ok {
		score += (1 - v) * 0.1
	}

	return score
}

func (c *IntentClassifier) calculateHumanLikeBotScore(features map[string]float64) float64 {
	score := 0.0

	if v, ok := features["rhythm_regularity"]; ok && v > 0.5 && v < 0.7 {
		score += 0.3
	}

	if v, ok := features["naturalness"]; ok && v > 0.4 && v < 0.7 {
		score += 0.25
	}

	if v, ok := features["pause_count"]; ok && v > 0 && v < 3 {
		score += 0.2
	}

	if v, ok := features["adaptability"]; ok && v > 0.4 && v < 0.7 {
		score += 0.15
	}

	if v, ok := features["consistency"]; ok {
		score += v * 0.1
	}

	return score
}

func (c *IntentClassifier) calculateScriptedBotScore(features map[string]float64) float64 {
	score := 0.0

	if v, ok := features["curvature_variance"]; ok && v < 0.001 {
		score += 0.35
	}

	if v, ok := features["direction_entropy"]; ok && v < 1.0 {
		score += 0.25
	}

	if v, ok := features["speed_entropy"]; ok && v < 1.5 {
		score += 0.2
	}

	if v, ok := features["rhythm_regularity"]; ok && v > 0.8 {
		score += 0.1
	}

	if v, ok := features["naturalness"]; ok {
		score += (1 - v) * 0.1
	}

	return score
}

func (c *IntentClassifier) calculateAggressiveBotScore(features map[string]float64) float64 {
	score := 0.0

	if v, ok := features["max_speed"]; ok && v > 1000 {
		score += 0.35
	}

	if v, ok := features["max_acceleration"]; ok && v > 5000 {
		score += 0.3
	}

	if v, ok := features["response_speed"]; ok && v > 0.8 {
		score += 0.2
	}

	if v, ok := features["patience_level"]; ok {
		score += (1 - v) * 0.1
	}

	if v, ok := features["naturalness"]; ok {
		score += (1 - v) * 0.05
	}

	return score
}

func (c *IntentClassifier) calculateUnknownScore(features map[string]float64) float64 {
	score := 0.5

	confidence := 0.0
	for _, v := range features {
		if v > 0.3 && v < 0.7 {
			confidence += 0.1
		}
	}

	score += confidence

	return math.Min(1.0, score)
}

func (c *IntentClassifier) determinePrimaryIntent(scores map[IntentType]float64) IntentType {
	var maxIntent IntentType
	var maxScore float64 = -1

	for intent, score := range scores {
		if score > maxScore {
			maxScore = score
			maxIntent = intent
		}
	}

	if maxScore < c.confidenceThresholds[maxIntent] {
		return IntentUnknown
	}

	return maxIntent
}

func (c *IntentClassifier) calculateConfidence(scores map[IntentType]float64, primary IntentType) float64 {
	primaryScore := scores[primary]

	secondaryScore := 0.0
	for intent, score := range scores {
		if intent != primary && score > secondaryScore {
			secondaryScore = score
		}
	}

	confidence := primaryScore - secondaryScore

	confidence = (confidence + 1.0) / 2.0

	return math.Max(0, math.Min(1, confidence))
}

func (c *IntentClassifier) extractBehavioralIndicators(features map[string]float64) []string {
	indicators := []string{}

	if v, ok := features["pause_count"]; ok {
		if v == 0 {
			indicators = append(indicators, "无停顿行为")
		} else if v > 5 {
			indicators = append(indicators, "频繁停顿")
		}
	}

	if v, ok := features["speed_variance"]; ok {
		if v < 5 {
			indicators = append(indicators, "速度异常稳定")
		} else if v > 100 {
			indicators = append(indicators, "速度波动剧烈")
		}
	}

	if v, ok := features["path_ratio"]; ok {
		if v < 1.05 {
			indicators = append(indicators, "轨迹过于直线")
		} else if v > 3.0 {
			indicators = append(indicators, "轨迹极度曲折")
		}
	}

	if v, ok := features["naturalness"]; ok {
		if v < 0.3 {
			indicators = append(indicators, "行为极不自然")
		} else if v > 0.8 {
			indicators = append(indicators, "行为非常自然")
		}
	}

	if v, ok := features["rhythm_regularity"]; ok {
		if v > 0.8 {
			indicators = append(indicators, "节奏高度规律")
		} else if v < 0.3 {
			indicators = append(indicators, "节奏混乱")
		}
	}

	if v, ok := features["avg_speed"]; ok {
		if v > 500 {
			indicators = append(indicators, "移动速度极快")
		} else if v < 50 {
			indicators = append(indicators, "移动速度极慢")
		}
	}

	if v, ok := features["max_acceleration"]; ok {
		if v > 5000 {
			indicators = append(indicators, "加速度异常")
		}
	}

	return indicators
}

func (c *IntentClassifier) generateReasoning(primary IntentType, features map[string]float64) string {
	switch primary {
	case IntentNormalUser:
		return "轨迹特征显示正常的人类操作行为，具有合理的速度变化、自然的停顿和适度的轨迹曲折度。"
	case IntentCarefulUser:
		return "行为特征表明用户较为谨慎，移动速度较慢，有较多的停顿和微调动作。"
	case IntentImpatientUser:
		return "检测到快速且缺乏停顿的行为模式，可能表明用户较为急躁。"
	case IntentAutomatedBot:
		return "轨迹呈现高度规律性，缺乏自然变化，符合自动化程序的典型特征。"
	case IntentHumanLikeBot:
		return "行为模式部分模仿人类但存在明显规律性，可能为高级自动化工具。"
	case IntentScriptedBot:
		return "检测到程序化的脚本执行特征，轨迹过于规则且缺乏变化。"
	case IntentAggressiveBot:
		return "行为异常激进，移动速度快且加速度大，疑似攻击性自动化工具。"
	default:
		return "无法明确确定行为意图，建议进一步分析。"
	}
}

func (c *IntentClassifier) generateRecommendations(primary IntentType, confidence float64) []string {
	recommendations := []string{}

	switch primary {
	case IntentNormalUser:
		recommendations = append(recommendations, "正常通过验证")
	case IntentCarefulUser:
		recommendations = append(recommendations, "正常通过验证", "可以考虑简化验证流程")
	case IntentImpatientUser:
		recommendations = append(recommendations, "建议启用额外验证", "考虑添加延时机制")
	case IntentAutomatedBot:
		recommendations = append(recommendations, "强烈建议阻止", "记录日志并考虑加入黑名单")
		if confidence < 0.8 {
			recommendations = append(recommendations, "置信度不足，建议人工审核")
		}
	case IntentHumanLikeBot:
		recommendations = append(recommendations, "建议增加验证难度", "持续监控后续行为")
	case IntentScriptedBot:
		recommendations = append(recommendations, "强烈建议阻止", "检测到程序化攻击特征")
	case IntentAggressiveBot:
		recommendations = append(recommendations, "立即阻止", "可能正在发起攻击")
	default:
		recommendations = append(recommendations, "建议人工审核确认")
	}

	return recommendations
}

func (c *IntentClassifier) BatchRecognize(traceDataList []*model.TraceData) ([]*IntentRecognitionResult, error) {
	results := make([]*IntentRecognitionResult, 0, len(traceDataList))

	for _, traceData := range traceDataList {
		result, err := c.RecognizeIntent(traceData)
		if err != nil {
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

func (c *IntentClassifier) GetIntentStatistics(results []*IntentRecognitionResult) map[string]interface{} {
	stats := make(map[string]interface{})

	intentCounts := make(map[IntentType]int)
	totalConfidence := 0.0

	for _, result := range results {
		intentCounts[result.PrimaryIntent]++
		totalConfidence += result.Confidence
	}

	avgConfidence := 0.0
	if len(results) > 0 {
		avgConfidence = totalConfidence / float64(len(results))
	}

	stats["total_samples"] = len(results)
	stats["intent_distribution"] = intentCounts
	stats["average_confidence"] = avgConfidence

	mostCommonIntent := IntentUnknown
	maxCount := 0
	for intent, count := range intentCounts {
		if count > maxCount {
			maxCount = count
			mostCommonIntent = intent
		}
	}
	stats["most_common_intent"] = mostCommonIntent
	stats["most_common_intent_count"] = maxCount

	return stats
}

type MultiLevelIntentAnalyzer struct {
	classifiers []IntentClassifier
}

func NewMultiLevelIntentAnalyzer() *MultiLevelIntentAnalyzer {
	analyzer := &MultiLevelIntentAnalyzer{
		classifiers: make([]IntentClassifier, 0),
	}

	for i := 0; i < 3; i++ {
		analyzer.classifiers = append(analyzer.classifiers, *NewIntentClassifier())
	}

	return analyzer
}

func (a *MultiLevelIntentAnalyzer) AnalyzeMultiLevel(traceData *model.TraceData) (*IntentRecognitionResult, error) {
	if len(a.classifiers) == 0 {
		classifier := NewIntentClassifier()
		return classifier.RecognizeIntent(traceData)
	}

	baseResult, err := a.classifiers[0].RecognizeIntent(traceData)
	if err != nil {
		return nil, err
	}

	if baseResult.Confidence > 0.8 {
		return baseResult, nil
	}

	votes := make(map[IntentType]int)
	votes[baseResult.PrimaryIntent]++

	for i := 1; i < len(a.classifiers); i++ {
		result, err := a.classifiers[i].RecognizeIntent(traceData)
		if err != nil {
			continue
		}
		votes[result.PrimaryIntent]++
	}

	maxVotes := 0
	consensusIntent := baseResult.PrimaryIntent
	for intent, count := range votes {
		if count > maxVotes {
			maxVotes = count
			consensusIntent = intent
		}
	}

	if maxVotes > 1 {
		baseResult.PrimaryIntent = consensusIntent
		baseResult.Confidence = float64(maxVotes) / float64(len(a.classifiers))
	}

	return baseResult, nil
}
