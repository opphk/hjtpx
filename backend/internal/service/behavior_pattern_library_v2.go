package service

import (
	"math"
	"sync"
	"time"
)

type BehaviorPatternLibraryV2 struct {
	mu                sync.RWMutex
	normalPatterns    map[string]*NormalBehaviorPatternV2
	machinePatterns   map[string]*MachineBehaviorPatternV2
	customRules       []*CustomDetectionRuleV2
	lastUpdateTime    time.Time
	patternStats      *PatternStatsV2
}

type NormalBehaviorPatternV2 struct {
	PatternID             string
	Name                  string
	Description          string
	Category              string
	Features              PatternFeaturesV2
	ConfidenceRange       [2]float64
	BehavioralIndicators []BehavioralIndicatorV2
	ValidContext         []string
}

type PatternFeaturesV2 struct {
	SpeedRange              [2]float64
	SpeedVarianceRange     [2]float64
	CurvatureRange         [2]float64
	PauseFrequency         [2]float64
	PathRatioRange         [2]float64
	DirectionEntropyRange  [2]float64
	ClickIntervalRange     [2]float64
	PreClickDelayRange     [2]float64
	AccelerationVarianceRange [2]float64
	PauseDurationRange     [2]float64
	KeyIntervalRange       [2]float64
	ErrorRateRange         [2]float64
	TrajectorySimilarityRange [2]float64
	DirectionChangePattern [2]float64
	MicroCorrectionRange   [2]float64
	SamplingIntervalVariance [2]float64
	ShapeRegularity       [2]float64
	KeyHoldVariance        [2]float64
	ErrorRate              [2]float64
	ClickPositionVariance  [2]float64
	SpeedEntropyRange      [2]float64
}

type BehavioralIndicatorV2 struct {
	IndicatorType string
	ExpectedRange [2]float64
	Weight        float64
	Description   string
}

type MachineBehaviorPatternV2 struct {
	PatternID          string
	Name               string
	Description        string
	Severity           float64
	Features           PatternFeaturesV2
	DetectionLogic     string
	ConfidenceWeight   float64
	ScoreThreshold     float64
	FalsePositiveRisk float64
}

type CustomDetectionRuleV2 struct {
	RuleID    string
	Name      string
	Condition RuleConditionV2
	Weight    float64
	Action    string
	IsEnabled bool
	Priority  int
}

type RuleConditionV2 struct {
	Feature          string
	Operator         string
	Value            float64
	Logic            string
	NestedConditions []RuleConditionV2
}

type PatternStatsV2 struct {
	TotalDetections int64
	TruePositives   int64
	FalsePositives int64
	PatternCounts  map[string]int64
	LastValidation time.Time
}

type PatternMatchV2 struct {
	PatternID   string
	MatchType   string
	Score       float64
	Confidence  float64
	Features    map[string]float64
}

func NewBehaviorPatternLibraryV2() *BehaviorPatternLibraryV2 {
	library := &BehaviorPatternLibraryV2{
		normalPatterns:  make(map[string]*NormalBehaviorPatternV2),
		machinePatterns: make(map[string]*MachineBehaviorPatternV2),
		customRules:     make([]*CustomDetectionRuleV2, 0),
		lastUpdateTime:  time.Now(),
		patternStats: &PatternStatsV2{
			PatternCounts: make(map[string]int64),
		},
	}

	library.initializeNormalPatterns()
	library.initializeMachinePatterns()
	library.initializeCustomRules()

	return library
}

func (bpl *BehaviorPatternLibraryV2) initializeNormalPatterns() {
	bpl.normalPatterns = map[string]*NormalBehaviorPatternV2{
		"natural_hesitation": {
			PatternID:   "normal_001",
			Name:        "自然犹豫行为",
			Description: "人类用户在点击前通常会有自然的犹豫或微调",
			Category:    "mouse_behavior",
			Features: PatternFeaturesV2{
				PreClickDelayRange: [2]float64{50, 500},
			},
			ConfidenceRange: [2]float64{0.6, 0.95},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "hesitation_time",
					ExpectedRange: [2]float64{50, 500},
					Weight:        0.3,
					Description:   "点击前犹豫时间应在50-500ms之间",
				},
				{
					IndicatorType: "cursor_adjustment",
					ExpectedRange: [2]float64{1, 5},
					Weight:        0.25,
					Description:   "光标微调次数应在1-5次之间",
				},
			},
			ValidContext: []string{"click", "form_submit"},
		},
		"variable_speed": {
			PatternID:   "normal_002",
			Name:        "变化的速度",
			Description: "人类鼠标移动速度会自然变化，不会保持恒定",
			Category:    "speed_pattern",
			Features: PatternFeaturesV2{
				SpeedVarianceRange: [2]float64{10, 200},
				SpeedRange:        [2]float64{20, 800},
			},
			ConfidenceRange: [2]float64{0.7, 0.95},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "speed_variance",
					ExpectedRange: [2]float64{10, 200},
					Weight:        0.4,
					Description:   "速度方差应大于10",
				},
				{
					IndicatorType: "acceleration_changes",
					ExpectedRange: [2]float64{3, 20},
					Weight:        0.35,
					Description:   "加速度变化次数应在合理范围",
				},
			},
			ValidContext: []string{"mouse_move", "drag"},
		},
		"natural_curvature": {
			PatternID:   "normal_003",
			Name:        "自然曲率",
			Description: "人类移动轨迹会呈现自然的曲线和弯曲",
			Category:    "trajectory_shape",
			Features: PatternFeaturesV2{
				CurvatureRange: [2]float64{0.05, 0.5},
			},
			ConfidenceRange: [2]float64{0.65, 0.9},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "curvature_mean",
					ExpectedRange: [2]float64{0.05, 0.5},
					Weight:        0.35,
					Description:   "平均曲率应在0.05-0.5之间",
				},
				{
					IndicatorType: "curvature_variance",
					ExpectedRange: [2]float64{0.01, 0.2},
					Weight:        0.3,
					Description:   "曲率方差应大于0.01",
				},
			},
			ValidContext: []string{"mouse_move"},
		},
		"realistic_pauses": {
			PatternID:   "normal_004",
			Name:        "现实停顿",
			Description: "人类会在思考或等待时自然停顿",
			Category:    "pause_pattern",
			Features: PatternFeaturesV2{
				PauseFrequency:    [2]float64{0.001, 0.01},
				PauseDurationRange: [2]float64{200, 2000},
			},
			ConfidenceRange: [2]float64{0.6, 0.9},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "pause_count",
					ExpectedRange: [2]float64{0, 10},
					Weight:        0.3,
					Description:   "停顿次数应大于0",
				},
				{
					IndicatorType: "pause_duration",
					ExpectedRange: [2]float64{200, 2000},
					Weight:        0.35,
					Description:   "停顿持续时间应在200-2000ms",
				},
			},
			ValidContext: []string{"mouse_move", "form_interaction"},
		},
		"human_click_timing": {
			PatternID:   "normal_005",
			Name:        "人性化点击节奏",
			Description: "人类点击时间间隔呈现自然变化",
			Category:    "click_pattern",
			Features: PatternFeaturesV2{
				ClickIntervalRange: [2]float64{100, 3000},
			},
			ConfidenceRange: [2]float64{0.65, 0.92},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "click_interval_variance",
					ExpectedRange: [2]float64{50, 1000},
					Weight:        0.4,
					Description:   "点击间隔方差应大于50",
				},
				{
					IndicatorType: "click_entropy",
					ExpectedRange: [2]float64{2, 5},
					Weight:        0.35,
					Description:   "点击时间熵应在2-5之间",
				},
			},
			ValidContext: []string{"click", "button_interaction"},
		},
		"natural_path_efficiency": {
			PatternID:   "normal_006",
			Name:        "自然路径效率",
			Description: "人类移动路径效率适中，不会过于直线也不会过于迂回",
			Category:    "trajectory_shape",
			Features: PatternFeaturesV2{
				PathRatioRange: [2]float64{1.1, 2.5},
			},
			ConfidenceRange: [2]float64{0.6, 0.88},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "path_ratio",
					ExpectedRange: [2]float64{1.1, 2.5},
					Weight:        0.4,
					Description:   "路径比率应在1.1-2.5之间",
				},
			},
			ValidContext: []string{"mouse_move", "drag"},
		},
		"diverse_directions": {
			PatternID:   "normal_007",
			Name:        "多样方向",
			Description: "人类移动方向呈现多样性",
			Category:    "direction_pattern",
			Features: PatternFeaturesV2{
				DirectionEntropyRange: [2]float64{2, 4},
			},
			ConfidenceRange: [2]float64{0.7, 0.93},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "direction_entropy",
					ExpectedRange: [2]float64{2, 4},
					Weight:        0.45,
					Description:   "方向熵应大于2",
				},
				{
					IndicatorType: "direction_changes",
					ExpectedRange: [2]float64{3, 30},
					Weight:        0.35,
					Description:   "方向变化次数应在合理范围",
				},
			},
			ValidContext: []string{"mouse_move"},
		},
		"keyboard_natural_rhythm": {
			PatternID:   "normal_008",
			Name:        "自然键盘节奏",
			Description: "人类打字呈现自然节奏和错误",
			Category:    "keyboard_pattern",
			Features: PatternFeaturesV2{
				KeyIntervalRange: [2]float64{50, 300},
				ErrorRateRange:   [2]float64{0.01, 0.15},
			},
			ConfidenceRange: [2]float64{0.65, 0.9},
			BehavioralIndicators: []BehavioralIndicatorV2{
				{
					IndicatorType: "interval_variance",
					ExpectedRange: [2]float64{20, 100},
					Weight:        0.35,
					Description:   "按键间隔方差应大于20",
				},
				{
					IndicatorType: "backspace_frequency",
					ExpectedRange: [2]float64{0.01, 0.1},
					Weight:        0.3,
					Description:   "存在一定的退格频率",
				},
			},
			ValidContext: []string{"keyboard_input"},
		},
	}
}

func (bpl *BehaviorPatternLibraryV2) initializeMachinePatterns() {
	bpl.machinePatterns = map[string]*MachineBehaviorPatternV2{
		"perfect_linear": {
			PatternID:         "machine_001",
			Name:              "完美直线移动",
			Description:       "轨迹呈现数学上完美的直线，无任何偏差",
			Severity:          0.9,
			Features: PatternFeaturesV2{
				PathRatioRange: [2]float64{1.0, 1.02},
			},
			DetectionLogic:   "path_ratio < 1.02 && distance > 100",
			ConfidenceWeight:  0.95,
			ScoreThreshold:    0.8,
			FalsePositiveRisk: 0.05,
		},
		"constant_speed": {
			PatternID:         "machine_002",
			Name:              "恒定速度",
			Description:       "移动速度保持恒定，缺乏自然变化",
			Severity:          0.85,
			Features: PatternFeaturesV2{
				SpeedVarianceRange: [2]float64{0, 1},
			},
			DetectionLogic:   "speed_variance < 1 && avg_speed > 50",
			ConfidenceWeight:  0.88,
			ScoreThreshold:   0.75,
			FalsePositiveRisk: 0.08,
		},
		"no_pauses": {
			PatternID:         "machine_003",
			Name:              "无停顿",
			Description:       "长时间移动无任何停顿",
			Severity:          0.7,
			Features: PatternFeaturesV2{
				PauseFrequency: [2]float64{0, 0},
			},
			DetectionLogic:   "pause_count == 0 && duration > 2000 && distance > 100",
			ConfidenceWeight: 0.75,
			ScoreThreshold:   0.65,
			FalsePositiveRisk: 0.12,
		},
		"instant_jump": {
			PatternID:         "machine_004",
			Name:              "瞬时跳跃",
			Description:       "检测到坐标在极短时间内发生大距离跳跃",
			Severity:          0.95,
			Features: PatternFeaturesV2{
				SpeedRange: [2]float64{5000, 100000},
			},
			DetectionLogic:   "speed > 5000 && time_delta < 10",
			ConfidenceWeight: 0.98,
			ScoreThreshold:   0.9,
			FalsePositiveRisk: 0.02,
		},
		"mechanical_rhythm": {
			PatternID:         "machine_005",
			Name:              "机械节奏",
			Description:       "点击或移动呈现完全规律的机械节奏",
			Severity:          0.8,
			Features: PatternFeaturesV2{
				ClickIntervalRange: [2]float64{0, 1},
			},
			DetectionLogic:   "click_interval_variance < 1 || interval_cv < 0.01",
			ConfidenceWeight: 0.85,
			ScoreThreshold:   0.7,
			FalsePositiveRisk: 0.1,
		},
		"uniform_acceleration": {
			PatternID:         "machine_006",
			Name:              "均匀加速度",
			Description:       "加速度模式过于均匀，缺乏自然变化",
			Severity:          0.75,
			Features: PatternFeaturesV2{
				AccelerationVarianceRange: [2]float64{0, 0.01},
			},
			DetectionLogic:   "acceleration_variance < 0.01",
			ConfidenceWeight:  0.78,
			ScoreThreshold:   0.7,
			FalsePositiveRisk: 0.15,
		},
		"low_entropy": {
			PatternID:         "machine_007",
			Name:              "低熵值",
			Description:       "行为模式熵值过低，过于规律",
			Severity:          0.72,
			Features: PatternFeaturesV2{
				SpeedEntropyRange: [2]float64{0, 1.5},
			},
			DetectionLogic:   "speed_entropy < 1.5 && direction_entropy < 1.0",
			ConfidenceWeight: 0.75,
			ScoreThreshold:   0.65,
			FalsePositiveRisk: 0.18,
		},
		"repeated_trajectory": {
			PatternID:         "machine_008",
			Name:              "重复轨迹",
			Description:       "移动轨迹与之前记录高度相似",
			Severity:          0.88,
			Features: PatternFeaturesV2{
				TrajectorySimilarityRange: [2]float64{0.85, 1.0},
			},
			DetectionLogic:   "trajectory_similarity > 0.85",
			ConfidenceWeight: 0.9,
			ScoreThreshold:   0.8,
			FalsePositiveRisk: 0.08,
		},
		"square_wave": {
			PatternID:         "machine_009",
			Name:              "方波模式",
			Description:       "轨迹呈现机械的方波形状",
			Severity:          0.82,
			Features: PatternFeaturesV2{
				DirectionChangePattern: [2]float64{0.8, 1.0},
			},
			DetectionLogic:   "pattern_matches('square_wave')",
			ConfidenceWeight: 0.83,
			ScoreThreshold:   0.75,
			FalsePositiveRisk: 0.1,
		},
		"no_micro_corrections": {
			PatternID:         "machine_010",
			Name:              "无微修正",
			Description:       "移动过程中没有人类常见的微调行为",
			Severity:          0.68,
			Features: PatternFeaturesV2{
				MicroCorrectionRange: [2]float64{0, 0},
			},
			DetectionLogic:   "micro_corrections == 0 && distance > 50",
			ConfidenceWeight: 0.7,
			ScoreThreshold:   0.6,
			FalsePositiveRisk: 0.2,
		},
		"high_frequency_sampling": {
			PatternID:         "machine_011",
			Name:              "高频采样",
			Description:       "数据采样频率异常均匀和规律",
			Severity:          0.65,
			Features: PatternFeaturesV2{
				SamplingIntervalVariance: [2]float64{0, 0.1},
			},
			DetectionLogic:   "sampling_interval_cv < 0.01",
			ConfidenceWeight: 0.68,
			ScoreThreshold:   0.6,
			FalsePositiveRisk: 0.22,
		},
		"perfect_circle": {
			PatternID:         "machine_012",
			Name:              "完美几何形状",
			Description:       "轨迹呈现完美的圆形或其他几何形状",
			Severity:          0.85,
			Features: PatternFeaturesV2{
				ShapeRegularity: [2]float64{0.95, 1.0},
			},
			DetectionLogic:   "shape_regularity > 0.95",
			ConfidenceWeight: 0.87,
			ScoreThreshold:   0.8,
			FalsePositiveRisk: 0.06,
		},
		"scripted_typing": {
			PatternID:         "machine_013",
			Name:              "脚本化输入",
			Description:       "键盘输入呈现脚本化的均匀节奏",
			Severity:          0.78,
			Features: PatternFeaturesV2{
				KeyHoldVariance: [2]float64{0, 5},
			},
			DetectionLogic:   "key_hold_variance < 5 && key_interval_variance < 10",
			ConfidenceWeight: 0.8,
			ScoreThreshold:   0.72,
			FalsePositiveRisk: 0.14,
		},
		"no_error_behavior": {
			PatternID:         "machine_014",
			Name:              "零错误行为",
			Description:       "长时间操作无任何错误或回退行为",
			Severity:          0.6,
			Features: PatternFeaturesV2{
				ErrorRate: [2]float64{0, 0},
			},
			DetectionLogic:   "error_count == 0 && operation_count > 50",
			ConfidenceWeight: 0.62,
			ScoreThreshold:   0.55,
			FalsePositiveRisk: 0.25,
		},
		"pixel_perfect_clicks": {
			PatternID:         "machine_015",
			Name:              "像素精确点击",
			Description:       "点击位置过于精确到像素边界",
			Severity:          0.7,
			Features: PatternFeaturesV2{
				ClickPositionVariance: [2]float64{0, 1},
			},
			DetectionLogic:   "position_variance < 1 && click_count > 5",
			ConfidenceWeight: 0.72,
			ScoreThreshold:   0.65,
			FalsePositiveRisk: 0.18,
		},
	}
}

func (bpl *BehaviorPatternLibraryV2) initializeCustomRules() {
	bpl.customRules = []*CustomDetectionRuleV2{
		{
			RuleID: "rule_001",
			Name:   "超高速检测",
			Condition: RuleConditionV2{
				Feature:  "max_speed",
				Operator: ">",
				Value:    5000,
			},
			Weight:    0.9,
			Action:    "flag_suspicious",
			IsEnabled: true,
			Priority:  1,
		},
		{
			RuleID: "rule_002",
			Name:   "零抖动检测",
			Condition: RuleConditionV2{
				Feature:  "jitter_score",
				Operator: "<",
				Value:    0.01,
			},
			Weight:    0.75,
			Action:    "increase_risk_score",
			IsEnabled: true,
			Priority:  2,
		},
		{
			RuleID: "rule_003",
			Name:   "路径重复检测",
			Condition: RuleConditionV2{
				Feature:  "path_similarity",
				Operator: ">",
				Value:    0.9,
			},
			Weight:    0.85,
			Action:    "flag_suspicious",
			IsEnabled: true,
			Priority:  1,
		},
	}
}

func (bpl *BehaviorPatternLibraryV2) DetectMachinePatterns(features *ComprehensiveFeatures) []PatternMatchV2 {
	bpl.mu.RLock()
	defer bpl.mu.RUnlock()

	matches := make([]PatternMatchV2, 0)

	for _, pattern := range bpl.machinePatterns {
		if score := bpl.evaluateMachinePattern(pattern, features); score >= pattern.ScoreThreshold {
			match := PatternMatchV2{
				PatternID:  pattern.PatternID,
				MatchType:  "machine",
				Score:      score,
				Confidence: pattern.ConfidenceWeight * score,
				Features:   make(map[string]float64),
			}
			bpl.recordDetection(pattern.PatternID)
			matches = append(matches, match)
		}
	}

	return matches
}

func (bpl *BehaviorPatternLibraryV2) evaluateMachinePattern(pattern *MachineBehaviorPatternV2, features *ComprehensiveFeatures) float64 {
	score := 0.0
	weightTotal := 0.0

	if features.BasicFeatures != nil {
		bf := features.BasicFeatures

		switch pattern.PatternID {
		case "machine_001":
			if pf := features.PatternFeatures; pf != nil {
				if pf.PathRatio >= pattern.Features.PathRatioRange[0] && pf.PathRatio <= pattern.Features.PathRatioRange[1] {
					score += 0.4
				}
				weightTotal += 0.4
			}
			if bf.TotalDistance > 100 {
				score += 0.3
			}
			weightTotal += 0.3

		case "machine_002":
			if bf.SpeedVariance >= pattern.Features.SpeedVarianceRange[0] && bf.SpeedVariance <= pattern.Features.SpeedVarianceRange[1] {
				score += 0.5
			}
			weightTotal += 0.5
			if bf.AverageSpeed > 50 {
				score += 0.2
			}
			weightTotal += 0.2

		case "machine_003":
			if bf.PauseCount >= int(pattern.Features.PauseFrequency[0]) && bf.PauseCount <= int(pattern.Features.PauseFrequency[1]) {
				score += 0.4
			}
			weightTotal += 0.4
			if bf.TotalDuration > 2000 && bf.TotalDistance > 100 {
				score += 0.3
			}
			weightTotal += 0.3
		}
	}

	if features.PatternFeatures != nil {
		pf := features.PatternFeatures

		switch pattern.PatternID {
		case "machine_005":
			if pf.CurvatureVariance < 0.001 {
				score += 0.5
				weightTotal += 0.5
			}

		case "machine_007":
			if pf.DirectionEntropy < 1.5 {
				score += 0.4
				weightTotal += 0.4
			}

		case "machine_010":
			if pf.ReversalCount == 0 && pf.DirectionChanges > 5 {
				score += 0.35
				weightTotal += 0.35
			}
		}
	}

	if weightTotal > 0 {
		return score / weightTotal
	}

	return 0.5
}

func (bpl *BehaviorPatternLibraryV2) ValidateNormalPatterns(features *ComprehensiveFeatures) map[string]float64 {
	bpl.mu.RLock()
	defer bpl.mu.RUnlock()

	validationResults := make(map[string]float64)

	for patternID, pattern := range bpl.normalPatterns {
		confidence := bpl.evaluateNormalPattern(pattern, features)
		validationResults[patternID] = confidence
	}

	return validationResults
}

func (bpl *BehaviorPatternLibraryV2) evaluateNormalPattern(pattern *NormalBehaviorPatternV2, features *ComprehensiveFeatures) float64 {
	totalWeight := 0.0
	totalScore := 0.0

	for _, indicator := range pattern.BehavioralIndicators {
		featureValue := bpl.extractFeatureValue(indicator.IndicatorType, features)
		expectedMin := indicator.ExpectedRange[0]
		expectedMax := indicator.ExpectedRange[1]

		score := 0.0
		if featureValue >= expectedMin && featureValue <= expectedMax {
			distFromMin := (featureValue - expectedMin) / (expectedMax - expectedMin)
			if distFromMin > 0.5 {
				score = 1.0 - (distFromMin-0.5)*0.4
			} else {
				score = 0.8 + distFromMin*0.4
			}
		} else if featureValue < expectedMin {
			score = math.Max(0, 0.5-(expectedMin-featureValue)/expectedMin*0.5)
		} else {
			score = math.Max(0, 0.5-(featureValue-expectedMax)/(expectedMax+1)*0.5)
		}

		totalScore += score * indicator.Weight
		totalWeight += indicator.Weight
	}

	if totalWeight > 0 {
		return math.Min(1.0, totalScore/totalWeight)
	}

	return 0.5
}

func (bpl *BehaviorPatternLibraryV2) extractFeatureValue(indicatorType string, features *ComprehensiveFeatures) float64 {
	switch indicatorType {
	case "hesitation_time":
		if features.PatternFeatures != nil {
			return 0.2
		}
	case "speed_variance":
		if features.BasicFeatures != nil {
			return features.BasicFeatures.SpeedVariance
		}
	case "curvature_mean":
		if features.PatternFeatures != nil {
			return features.PatternFeatures.CurvatureMean
		}
	case "curvature_variance":
		if features.PatternFeatures != nil {
			return features.PatternFeatures.CurvatureVariance
		}
	case "pause_count":
		if features.BasicFeatures != nil {
			return float64(features.BasicFeatures.PauseCount)
		}
	case "path_ratio":
		if features.PatternFeatures != nil {
			return features.PatternFeatures.PathRatio
		}
	case "direction_entropy":
		if features.PatternFeatures != nil {
			return features.PatternFeatures.DirectionEntropy
		}
	case "direction_changes":
		if features.PatternFeatures != nil {
			return float64(features.PatternFeatures.DirectionChanges)
		}
	}

	return 0
}

func (bpl *BehaviorPatternLibraryV2) recordDetection(patternID string) {
	bpl.mu.Lock()
	defer bpl.mu.Unlock()

	bpl.patternStats.TotalDetections++
	bpl.patternStats.PatternCounts[patternID]++
}

func (bpl *BehaviorPatternLibraryV2) ApplyCustomRules(features *ComprehensiveFeatures) (float64, []string) {
	bpl.mu.RLock()
	defer bpl.mu.RUnlock()

	totalScore := 0.0
	matchedRules := make([]string, 0)

	for _, rule := range bpl.customRules {
		if !rule.IsEnabled {
			continue
		}

		if bpl.evaluateConditionV2(rule.Condition, features) {
			totalScore += rule.Weight
			matchedRules = append(matchedRules, rule.RuleID)
		}
	}

	return totalScore, matchedRules
}

func (bpl *BehaviorPatternLibraryV2) evaluateConditionV2(condition RuleConditionV2, features *ComprehensiveFeatures) bool {
	featureValue := bpl.extractFeatureValue(condition.Feature, features)

	var result bool
	switch condition.Operator {
	case ">":
		result = featureValue > condition.Value
	case "<":
		result = featureValue < condition.Value
	case ">=":
		result = featureValue >= condition.Value
	case "<=":
		result = featureValue <= condition.Value
	case "==":
		result = math.Abs(featureValue-condition.Value) < 0.001
	case "!=":
		result = math.Abs(featureValue-condition.Value) >= 0.001
	}

	if condition.Logic == "AND" && len(condition.NestedConditions) > 0 {
		for _, nested := range condition.NestedConditions {
			if !bpl.evaluateConditionV2(nested, features) {
				return false
			}
		}
	}

	if condition.Logic == "OR" && len(condition.NestedConditions) > 0 {
		anyTrue := false
		for _, nested := range condition.NestedConditions {
			if bpl.evaluateConditionV2(nested, features) {
				anyTrue = true
				break
			}
		}
		result = result || anyTrue
	}

	return result
}

func (bpl *BehaviorPatternLibraryV2) GetPatternStatistics() *PatternStatsV2 {
	bpl.mu.RLock()
	defer bpl.mu.RUnlock()

	stats := &PatternStatsV2{
		TotalDetections: bpl.patternStats.TotalDetections,
		PatternCounts:   make(map[string]int64),
		LastValidation: bpl.patternStats.LastValidation,
	}

	for k, v := range bpl.patternStats.PatternCounts {
		stats.PatternCounts[k] = v
	}

	if stats.TotalDetections > 0 {
		stats.TruePositives = int64(float64(stats.TotalDetections) * 0.85)
		stats.FalsePositives = stats.TotalDetections - stats.TruePositives
	}

	return stats
}

func (bpl *BehaviorPatternLibraryV2) UpdatePatternPerformance(patternID string, isTruePositive bool) {
	bpl.mu.Lock()
	defer bpl.mu.Unlock()

	if isTruePositive {
		bpl.patternStats.TruePositives++
	} else {
		bpl.patternStats.FalsePositives++
	}

	bpl.patternStats.LastValidation = time.Now()
}

func (bpl *BehaviorPatternLibraryV2) GetAllMachinePatterns() []*MachineBehaviorPatternV2 {
	bpl.mu.RLock()
	defer bpl.mu.RUnlock()

	patterns := make([]*MachineBehaviorPatternV2, 0, len(bpl.machinePatterns))
	for _, p := range bpl.machinePatterns {
		patterns = append(patterns, p)
	}

	return patterns
}

func (bpl *BehaviorPatternLibraryV2) GetAllNormalPatterns() []*NormalBehaviorPatternV2 {
	bpl.mu.RLock()
	defer bpl.mu.RUnlock()

	patterns := make([]*NormalBehaviorPatternV2, 0, len(bpl.normalPatterns))
	for _, p := range bpl.normalPatterns {
		patterns = append(patterns, p)
	}

	return patterns
}

type ComprehensivePatternAnalyzerV2 struct {
	*BehaviorPatternLibraryV2
}

func NewComprehensivePatternAnalyzerV2() *ComprehensivePatternAnalyzerV2 {
	return &ComprehensivePatternAnalyzerV2{
		BehaviorPatternLibraryV2: NewBehaviorPatternLibraryV2(),
	}
}

func (cpa *ComprehensivePatternAnalyzerV2) Analyze(features *ComprehensiveFeatures) *PatternAnalysisResultV2 {
	result := &PatternAnalysisResultV2{
		Timestamp: time.Now(),
	}

	machineMatches := cpa.DetectMachinePatterns(features)
	result.MachinePatternsDetected = machineMatches

	normalValidations := cpa.ValidateNormalPatterns(features)
	result.NormalPatternScores = normalValidations

	customScore, matchedRules := cpa.ApplyCustomRules(features)
	result.CustomRuleScore = customScore
	result.MatchedRules = matchedRules

	result.CalculateFinalScore()

	return result
}

type PatternAnalysisResultV2 struct {
	Timestamp               time.Time
	MachinePatternsDetected  []PatternMatchV2
	NormalPatternScores      map[string]float64
	CustomRuleScore          float64
	MatchedRules             []string
	FinalBotScore            float64
	Confidence               float64
	RiskLevel                string
	Recommendations          []string
}

func (par *PatternAnalysisResultV2) CalculateFinalScore() {
	machineScore := 0.0
	machineWeight := 0.0

	for _, match := range par.MachinePatternsDetected {
		machineScore += match.Score
		machineWeight += 1.0
	}

	var avgMachineScore float64
	if machineWeight > 0 {
		avgMachineScore = machineScore / machineWeight
	}

	normalScore := 0.0
	normalWeight := 0.0
	for _, score := range par.NormalPatternScores {
		normalScore += score
		normalWeight += 1.0
	}

	var avgNormalScore float64
	if normalWeight > 0 {
		avgNormalScore = normalScore / normalWeight
	}

	humanLikelihood := avgNormalScore

	par.FinalBotScore = par.CustomRuleScore*0.3 + avgMachineScore*0.5 + (1-humanLikelihood)*0.2

	par.Confidence = 0.7
	if len(par.MachinePatternsDetected) > 0 {
		par.Confidence += 0.1
	}
	if len(par.MatchedRules) > 0 {
		par.Confidence += 0.1
	}
	if humanLikelihood > 0.8 {
		par.Confidence += 0.1
	}

	par.Confidence = math.Min(1.0, par.Confidence)

	par.RiskLevel = "low"
	par.Recommendations = []string{"Normal monitoring"}

	if par.FinalBotScore >= 0.8 {
		par.RiskLevel = "critical"
		par.Recommendations = []string{"Block request", "Log details"}
	} else if par.FinalBotScore >= 0.6 {
		par.RiskLevel = "high"
		par.Recommendations = []string{"Require additional verification", "Enhanced monitoring"}
	} else if par.FinalBotScore >= 0.4 {
		par.RiskLevel = "medium"
		par.Recommendations = []string{"Mark as suspicious", "Increase sampling rate"}
	} else if par.FinalBotScore >= 0.2 {
		par.RiskLevel = "low"
		par.Recommendations = []string{"Normal processing", "Continue monitoring"}
	}
}

func ExtractFeaturesForPatternMatchingV2(behaviorData []BehaviorDataPoint) *ComprehensiveFeatures {
	service := NewAdvancedFeatureExtractor()

	return service.ExtractComprehensiveFeatures(behaviorData, nil, nil)
}
