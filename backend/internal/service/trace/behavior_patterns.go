package trace

import (
	"math"

	"github.com/hjtpx/hjtpx/internal/model"
)

type BotBehaviorPattern struct {
	PatternType       string
	Severity          float64
	ConfidenceThreshold float64
	Description       string
}

type SpeedAnomalyPattern struct {
	PatternType       string
	MinSpeed          float64
	MaxSpeed          float64
	Severity          float64
	Description       string
}

type MovementAnomalyPattern struct {
	PatternType       string
	Severity          float64
	Description       string
}

type BehaviorPatternLibrary struct {
	botPatterns          []BotBehaviorPattern
	speedAnomalyPatterns []SpeedAnomalyPattern
	movementAnomalyPatterns []MovementAnomalyPattern
}

func NewBehaviorPatternLibrary() *BehaviorPatternLibrary {
	library := &BehaviorPatternLibrary{}

	library.botPatterns = []BotBehaviorPattern{
		{
			PatternType:         "constant_speed",
			Severity:            0.8,
			ConfidenceThreshold: 0.7,
			Description:         "恒定速度移动，机器人特征明显",
		},
		{
			PatternType:         "perfect_linear",
			Severity:            0.9,
			ConfidenceThreshold: 0.8,
			Description:         "完美直线轨迹，无任何偏差",
		},
		{
			PatternType:         "no_micro_corrections",
			Severity:            0.6,
			ConfidenceThreshold: 0.6,
			Description:         "无微修正行为",
		},
		{
			PatternType:         "no_pauses",
			Severity:            0.5,
			ConfidenceThreshold: 0.5,
			Description:         "无正常停顿行为",
		},
		{
			PatternType:         "mechanical_rhythm",
			Severity:            0.7,
			ConfidenceThreshold: 0.65,
			Description:         "机械节奏的点击或移动",
		},
		{
			PatternType:         "instant_response",
			Severity:            0.85,
			ConfidenceThreshold: 0.75,
			Description:         "即时响应，无犹豫时间",
		},
		{
			PatternType:         "uniform_acceleration",
			Severity:            0.75,
			ConfidenceThreshold: 0.7,
			Description:         "均匀加速度模式",
		},
		{
			PatternType:         "low_entropy",
			Severity:            0.65,
			ConfidenceThreshold: 0.6,
			Description:         "低熵值，行为模式过于规律",
		},
	}

	library.speedAnomalyPatterns = []SpeedAnomalyPattern{
		{
			PatternType:  "extreme_high_speed",
			MinSpeed:     1000,
			MaxSpeed:     math.MaxFloat64,
			Severity:     0.9,
			Description:  "超高速移动",
		},
		{
			PatternType:  "too_slow",
			MinSpeed:     0,
			MaxSpeed:     10,
			Severity:     0.4,
			Description:  "异常慢速移动",
		},
		{
			PatternType:  "sudden_speed_change",
			MinSpeed:     0,
			MaxSpeed:     0,
			Severity:     0.7,
			Description:  "速度突变",
		},
		{
			PatternType:  "zero_variance_speed",
			MinSpeed:     0,
			MaxSpeed:     0,
			Severity:     0.85,
			Description:  "速度方差为零",
		},
		{
			PatternType:  "high_variance_speed",
			MinSpeed:     0,
			MaxSpeed:     0,
			Severity:     0.6,
			Description:  "速度方差异常高",
		},
	}

	library.movementAnomalyPatterns = []MovementAnomalyPattern{
		{
			PatternType: "geometric_perfection",
			Severity:    0.9,
			Description: "几何完美的移动模式",
		},
		{
			PatternType: "repeated_path",
			Severity:    0.8,
			Description: "重复的移动路径",
		},
		{
			PatternType: "no_curvature",
			Severity:    0.85,
			Description: "零曲率移动",
		},
		{
			PatternType: "sharp_angle_turns",
			Severity:    0.5,
			Description: "突然的尖角转弯",
		},
		{
			PatternType: "backtracking",
			Severity:    0.6,
			Description: "回溯移动",
		},
		{
			PatternType: "unnatural_angles",
			Severity:    0.7,
			Description: "不自然的角度变化",
		},
	}

	return library
}

func (lib *BehaviorPatternLibrary) DetectBotPatterns(features *AdvancedFeatures, basicFeatures *model.TraceFeatures) []BotPatternMatch {
	matches := []BotPatternMatch{}

	if features != nil {
		if features.SpeedVariance < 5 && basicFeatures.AvgSpeed > 50 {
			matches = append(matches, BotPatternMatch{
				Pattern:     lib.botPatterns[0],
				Confidence:  0.9,
				DetectedValue: features.SpeedVariance,
			})
		}

		if features.Sinuosity < 1.05 && basicFeatures.TotalDistance > 100 {
			matches = append(matches, BotPatternMatch{
				Pattern:     lib.botPatterns[1],
				Confidence:  0.95,
				DetectedValue: features.Sinuosity,
			})
		}

		if features.CurvatureVariance < 0.001 && basicFeatures.MoveCount > 20 {
			matches = append(matches, BotPatternMatch{
				Pattern:     lib.botPatterns[2],
				Confidence:  0.75,
				DetectedValue: features.CurvatureVariance,
			})
		}

		if basicFeatures.PauseCount == 0 && basicFeatures.TotalTime > 2000 {
			matches = append(matches, BotPatternMatch{
				Pattern:     lib.botPatterns[3],
				Confidence:  0.6,
				DetectedValue: 0,
			})
		}

		if features.AccelerationVariance < 0.01 {
			matches = append(matches, BotPatternMatch{
				Pattern:     lib.botPatterns[6],
				Confidence:  0.8,
				DetectedValue: features.AccelerationVariance,
			})
		}

		if features.SpeedEntropy < 1.5 {
			matches = append(matches, BotPatternMatch{
				Pattern:     lib.botPatterns[7],
				Confidence:  0.7,
				DetectedValue: features.SpeedEntropy,
			})
		}
	}

	return matches
}

func (lib *BehaviorPatternLibrary) DetectSpeedAnomalies(features *AdvancedFeatures, basicFeatures *model.TraceFeatures) []SpeedAnomalyMatch {
	matches := []SpeedAnomalyMatch{}

	if basicFeatures.AvgSpeed > 1000 {
		matches = append(matches, SpeedAnomalyMatch{
			Pattern:        lib.speedAnomalyPatterns[0],
			Confidence:     0.95,
			DetectedSpeed:  basicFeatures.AvgSpeed,
		})
	}

	if basicFeatures.AvgSpeed < 10 && basicFeatures.TotalTime > 5000 {
		matches = append(matches, SpeedAnomalyMatch{
			Pattern:        lib.speedAnomalyPatterns[1],
			Confidence:     0.5,
			DetectedSpeed:  basicFeatures.AvgSpeed,
		})
	}

	if features != nil {
		if features.SpeedVariance < 1 {
			matches = append(matches, SpeedAnomalyMatch{
				Pattern:        lib.speedAnomalyPatterns[3],
				Confidence:     0.9,
				DetectedSpeed:  basicFeatures.AvgSpeed,
			})
		}

		if features.SpeedVariance > 1000 && basicFeatures.AvgSpeed > 100 {
			matches = append(matches, SpeedAnomalyMatch{
				Pattern:        lib.speedAnomalyPatterns[4],
				Confidence:     0.7,
				DetectedSpeed:  basicFeatures.AvgSpeed,
			})
		}
	}

	return matches
}

func (lib *BehaviorPatternLibrary) DetectMovementAnomalies(features *AdvancedFeatures, basicFeatures *model.TraceFeatures) []MovementAnomalyMatch {
	matches := []MovementAnomalyMatch{}

	if features != nil {
		if features.Sinuosity < 1.02 && basicFeatures.TotalDistance > 200 {
			matches = append(matches, MovementAnomalyMatch{
				Pattern:    lib.movementAnomalyPatterns[0],
				Confidence: 0.85,
			})
		}

		if features.CurvatureMedian < 0.01 && basicFeatures.MoveCount > 30 {
			matches = append(matches, MovementAnomalyMatch{
				Pattern:    lib.movementAnomalyPatterns[2],
				Confidence: 0.9,
			})
		}

		if features.DirectionEntropy < 1.0 && basicFeatures.MoveCount > 20 {
			matches = append(matches, MovementAnomalyMatch{
				Pattern:    lib.movementAnomalyPatterns[5],
				Confidence: 0.75,
			})
		}
	}

	return matches
}

func (lib *BehaviorPatternLibrary) GetPatternRiskScore(
	botMatches []BotPatternMatch,
	speedMatches []SpeedAnomalyMatch,
	movementMatches []MovementAnomalyMatch,
) float64 {
	var totalScore float64
	var totalWeight float64

	for _, match := range botMatches {
		if match.Confidence >= match.Pattern.ConfidenceThreshold {
			totalScore += match.Pattern.Severity * match.Confidence
			totalWeight += match.Confidence
		}
	}

	for _, match := range speedMatches {
		totalScore += match.Pattern.Severity * match.Confidence
		totalWeight += match.Confidence
	}

	for _, match := range movementMatches {
		totalScore += match.Pattern.Severity * match.Confidence
		totalWeight += match.Confidence
	}

	if totalWeight > 0 {
		return math.Min(totalScore/totalWeight, 1.0)
	}

	return 0
}

type BotPatternMatch struct {
	Pattern         BotBehaviorPattern
	Confidence      float64
	DetectedValue   float64
}

type SpeedAnomalyMatch struct {
	Pattern         SpeedAnomalyPattern
	Confidence      float64
	DetectedSpeed   float64
}

type MovementAnomalyMatch struct {
	Pattern         MovementAnomalyPattern
	Confidence      float64
}

type PatternAnalysisResult struct {
	TraceID              string
	BotPatternScore      float64
	SpeedAnomalyScore    float64
	MovementAnomalyScore float64
	CombinedRiskScore    float64
	DetectedPatterns     []string
	RiskLevel            string
	IsBot                bool
}

func (lib *BehaviorPatternLibrary) AnalyzeComprehensiveRisk(
	traceData *model.TraceData,
	basicFeatures *model.TraceFeatures,
	advancedFeatures *AdvancedFeatures,
) *PatternAnalysisResult {
	result := &PatternAnalysisResult{
		DetectedPatterns: []string{},
	}

	botMatches := lib.DetectBotPatterns(advancedFeatures, basicFeatures)
	speedMatches := lib.DetectSpeedAnomalies(advancedFeatures, basicFeatures)
	movementMatches := lib.DetectMovementAnomalies(advancedFeatures, basicFeatures)

	for _, match := range botMatches {
		result.BotPatternScore += match.Pattern.Severity * match.Confidence
		if match.Confidence >= match.Pattern.ConfidenceThreshold {
			result.DetectedPatterns = append(result.DetectedPatterns, match.Pattern.Description)
		}
	}

	for _, match := range speedMatches {
		result.SpeedAnomalyScore += match.Pattern.Severity * match.Confidence
		result.DetectedPatterns = append(result.DetectedPatterns, match.Pattern.Description)
	}

	for _, match := range movementMatches {
		result.MovementAnomalyScore += match.Pattern.Severity * match.Confidence
		result.DetectedPatterns = append(result.DetectedPatterns, match.Pattern.Description)
	}

	totalMatches := len(botMatches) + len(speedMatches) + len(movementMatches)
	if totalMatches > 0 {
		result.BotPatternScore /= float64(len(botMatches))
		if len(speedMatches) > 0 {
			result.SpeedAnomalyScore /= float64(len(speedMatches))
		}
		if len(movementMatches) > 0 {
			result.MovementAnomalyScore /= float64(len(movementMatches))
		}
	}

	result.CombinedRiskScore = lib.GetPatternRiskScore(botMatches, speedMatches, movementMatches)

	botWeight := 0.4
	speedWeight := 0.35
	movementWeight := 0.25

	result.CombinedRiskScore = 
		result.BotPatternScore*botWeight + 
		result.SpeedAnomalyScore*speedWeight + 
		result.MovementAnomalyScore*movementWeight

	if result.CombinedRiskScore >= 0.8 {
		result.RiskLevel = "critical"
		result.IsBot = true
	} else if result.CombinedRiskScore >= 0.6 {
		result.RiskLevel = "high"
		result.IsBot = true
	} else if result.CombinedRiskScore >= 0.4 {
		result.RiskLevel = "medium"
		result.IsBot = false
	} else if result.CombinedRiskScore >= 0.2 {
		result.RiskLevel = "low"
		result.IsBot = false
	} else {
		result.RiskLevel = "minimal"
		result.IsBot = false
	}

	return result
}

func (lib *BehaviorPatternLibrary) GetAllPatterns() map[string][]interface{} {
	patterns := make(map[string][]interface{})

	botPatternsInterface := make([]interface{}, len(lib.botPatterns))
	for i, p := range lib.botPatterns {
		botPatternsInterface[i] = p
	}
	patterns["bot_patterns"] = botPatternsInterface

	speedPatternsInterface := make([]interface{}, len(lib.speedAnomalyPatterns))
	for i, p := range lib.speedAnomalyPatterns {
		speedPatternsInterface[i] = p
	}
	patterns["speed_anomaly_patterns"] = speedPatternsInterface

	movementPatternsInterface := make([]interface{}, len(lib.movementAnomalyPatterns))
	for i, p := range lib.movementAnomalyPatterns {
		movementPatternsInterface[i] = p
	}
	patterns["movement_anomaly_patterns"] = movementPatternsInterface

	return patterns
}
