package trace

import (
	"errors"
	"math"

	"github.com/hjtpx/hjtpx/internal/model"
)

type AnomalyPatternType string

const (
	PatternPerfectLine          AnomalyPatternType = "perfect_line"
	PatternConstantSpeed        AnomalyPatternType = "constant_speed"
	PatternZeroVariance         AnomalyPatternType = "zero_variance"
	PatternNoPauses             AnomalyPatternType = "no_pauses"
	PatternInstantJump          AnomalyPatternType = "instant_jump"
	PatternSquareWave           AnomalyPatternType = "square_wave"
	PatternReversedMovement     AnomalyPatternType = "reversed_movement"
	PatternBotPattern           AnomalyPatternType = "bot_pattern"
	PatternHighFrequency       AnomalyPatternType = "high_frequency"
	PatternLowResolution        AnomalyPatternType = "low_resolution"
	PatternAbnormalPressure     AnomalyPatternType = "abnormal_pressure"
	PatternMechanicalClicking   AnomalyPatternType = "mechanical_clicking"
	PatternRoboticScrolling     AnomalyPatternType = "robotic_scrolling"
	PatternUniformCurvature     AnomalyPatternType = "uniform_curvature"
	PatternExcessiveFluidity   AnomalyPatternType = "excessive_fluidity"
	PatternStereotypedMovement  AnomalyPatternType = "stereotyped_movement"
)

type AnomalyPattern struct {
	Type          AnomalyPatternType `json:"type"`
	Description   string            `json:"description"`
	Confidence    float64           `json:"confidence"`
	Severity      string            `json:"severity"`
	FeatureValues map[string]float64 `json:"feature_values"`
}

type AnomalyDetector struct {
	extractor *TraceExtractor
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		extractor: NewTraceExtractor(),
	}
}

func (a *AnomalyDetector) DetectAnomalies(traceData *model.TraceData) ([]AnomalyPattern, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据不足")
	}

	features, err := a.extractor.ExtractFeatures(traceData)
	if err != nil {
		return nil, err
	}

	var anomalies []AnomalyPattern

	if a.isPerfectLine(features, traceData) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternPerfectLine,
			Description: "轨迹呈现完美直线，疑似自动化工具生成",
			Confidence:  a.calculatePerfectLineConfidence(features, traceData),
			Severity:    "high",
			FeatureValues: map[string]float64{
				"path_ratio": features.PathRatio,
			},
		})
	}

	if a.isConstantSpeed(features) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternConstantSpeed,
			Description: "速度保持恒定，缺乏自然人类行为变化",
			Confidence:  a.calculateConstantSpeedConfidence(features),
			Severity:    "high",
			FeatureValues: map[string]float64{
				"speed_variance": features.SpeedVariance,
				"avg_speed":      features.AvgSpeed,
			},
		})
	}

	if a.isZeroVariance(features) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternZeroVariance,
			Description: "特征方差接近零，行为过于规律",
			Confidence:  a.calculateZeroVarianceConfidence(features),
			Severity:    "medium",
			FeatureValues: map[string]float64{
				"speed_variance": features.SpeedVariance,
				"accel_variance": features.AccelVariance,
			},
		})
	}

	if a.isNoPauses(features) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternNoPauses,
			Description: "轨迹中无停顿，不符合人类操作习惯",
			Confidence:  a.calculateNoPausesConfidence(features),
			Severity:    "medium",
			FeatureValues: map[string]float64{
				"pause_count": float64(features.PauseCount),
				"total_time":  float64(features.TotalTime),
			},
		})
	}

	if a.hasInstantJump(traceData) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternInstantJump,
			Description: "检测到瞬时跳跃，疑似坐标直接设置",
			Confidence:  a.calculateInstantJumpConfidence(traceData),
			Severity:    "high",
			FeatureValues: map[string]float64{
				"max_speed": features.MaxSpeed,
			},
		})
	}

	if a.isSquareWave(traceData) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternSquareWave,
			Description: "轨迹呈现方波模式，疑似程序化移动",
			Confidence:  a.calculateSquareWaveConfidence(traceData),
			Severity:    "high",
			FeatureValues: map[string]float64{
				"jitter_frequency": features.JitterFrequency,
			},
		})
	}

	if a.hasReversedMovement(traceData) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternReversedMovement,
			Description: "检测到反向移动模式，疑似自动化脚本",
			Confidence:  a.calculateReversedMovementConfidence(traceData),
			Severity:    "medium",
			FeatureValues: map[string]float64{
				"direction_change": features.DirectionChange,
			},
		})
	}

	if a.isBotPattern(features) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternBotPattern,
			Description: "综合特征匹配已知机器人行为模式",
			Confidence:  a.calculateBotPatternConfidence(features),
			Severity:    "critical",
			FeatureValues: map[string]float64{
				"total_score": float64(len(features.RiskFactors)),
			},
		})
	}

	if a.isHighFrequency(traceData) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternHighFrequency,
			Description: "采样频率异常高，疑似自动化工具",
			Confidence:  a.calculateHighFrequencyConfidence(traceData),
			Severity:    "medium",
			FeatureValues: map[string]float64{
				"point_count": float64(len(traceData.Points)),
				"total_time":  float64(traceData.TotalTime),
			},
		})
	}

	if a.isLowResolution(traceData) {
		anomalies = append(anomalies, AnomalyPattern{
			Type:        PatternLowResolution,
			Description: "坐标精度过低，疑似模拟或低精度自动化",
			Confidence:  a.calculateLowResolutionConfidence(traceData),
			Severity:    "low",
			FeatureValues: map[string]float64{
				"point_count": float64(len(traceData.Points)),
			},
		})
	}

	enhancedFeatures, err := a.extractor.ExtractEnhancedFeatures(traceData)
	if err == nil && enhancedFeatures != nil {
		if a.isAbnormalPressure(enhancedFeatures) {
			anomalies = append(anomalies, AnomalyPattern{
				Type:        PatternAbnormalPressure,
				Description: "点击压力异常,疑似自动化工具",
				Confidence:  a.calculateAbnormalPressureConfidence(enhancedFeatures),
				Severity:    "medium",
				FeatureValues: map[string]float64{
					"pressure_variance": enhancedFeatures.PressureVariance,
					"pressure_consistency": enhancedFeatures.PressureConsistency,
				},
			})
		}

		if a.isMechanicalClicking(enhancedFeatures) {
			anomalies = append(anomalies, AnomalyPattern{
				Type:        PatternMechanicalClicking,
				Description: "点击模式过于机械,缺乏人类自然变化",
				Confidence:  a.calculateMechanicalClickingConfidence(enhancedFeatures),
				Severity:    "high",
				FeatureValues: map[string]float64{
					"click_regularity": enhancedFeatures.ClickRegularity,
				},
			})
		}

		if a.isRoboticScrolling(enhancedFeatures) {
			anomalies = append(anomalies, AnomalyPattern{
				Type:        PatternRoboticScrolling,
				Description: "滚动行为过于规律,疑似自动化脚本",
				Confidence:  a.calculateRoboticScrollingConfidence(enhancedFeatures),
				Severity:    "high",
				FeatureValues: map[string]float64{
					"scroll_regularity": enhancedFeatures.ScrollRegularity,
				},
			})
		}

		if a.isUniformCurvature(enhancedFeatures) {
			anomalies = append(anomalies, AnomalyPattern{
				Type:        PatternUniformCurvature,
				Description: "轨迹曲率过于均匀,缺乏自然波动",
				Confidence:  a.calculateUniformCurvatureConfidence(enhancedFeatures),
				Severity:    "high",
				FeatureValues: map[string]float64{
					"curvature_variance": enhancedFeatures.CurvatureVariance,
				},
			})
		}

		if a.isExcessiveFluidity(enhancedFeatures) {
			anomalies = append(anomalies, AnomalyPattern{
				Type:        PatternExcessiveFluidity,
				Description: "移动过于流畅,不符合人类操作特征",
				Confidence:  a.calculateExcessiveFluidityConfidence(enhancedFeatures),
				Severity:    "medium",
				FeatureValues: map[string]float64{
					"movement_fluidity": enhancedFeatures.MovementFluidity,
				},
			})
		}

		if a.isStereotypedMovement(enhancedFeatures) {
			anomalies = append(anomalies, AnomalyPattern{
				Type:        PatternStereotypedMovement,
				Description: "行为模式单一重复,疑似自动化执行",
				Confidence:  a.calculateStereotypedMovementConfidence(enhancedFeatures),
				Severity:    "critical",
				FeatureValues: map[string]float64{
					"spatial_entropy": enhancedFeatures.SpatialSpreadX,
				},
			})
		}
	}

	return anomalies, nil
}

func (a *AnomalyDetector) isPerfectLine(features *model.TraceFeatures, traceData *model.TraceData) bool {
	return features.PathRatio < 1.02 && features.TotalDistance > 50
}

func (a *AnomalyDetector) calculatePerfectLineConfidence(features *model.TraceFeatures, traceData *model.TraceData) float64 {
	if features.PathRatio >= 1.02 {
		return 0
	}
	return math.Min(1.0, (1.02-features.PathRatio)*50)
}

func (a *AnomalyDetector) isConstantSpeed(features *model.TraceFeatures) bool {
	return features.SpeedVariance < 5 && features.AvgSpeed > 50 && features.TotalTime > 1000
}

func (a *AnomalyDetector) calculateConstantSpeedConfidence(features *model.TraceFeatures) float64 {
	if features.SpeedVariance >= 5 {
		return 0
	}
	if features.AvgSpeed <= 50 {
		return 0
	}
	return math.Min(1.0, (5-features.SpeedVariance)/5*0.8+0.2)
}

func (a *AnomalyDetector) isZeroVariance(features *model.TraceFeatures) bool {
	return features.SpeedVariance < 1 && features.AccelVariance < 100
}

func (a *AnomalyDetector) calculateZeroVarianceConfidence(features *model.TraceFeatures) float64 {
	speedScore := math.Max(0, 1-features.SpeedVariance)
	accelScore := math.Max(0, 1-features.AccelVariance/100)
	return (speedScore + accelScore) / 2
}

func (a *AnomalyDetector) isNoPauses(features *model.TraceFeatures) bool {
	return features.PauseCount == 0 && features.TotalTime > 2000 && features.TotalDistance > 100
}

func (a *AnomalyDetector) calculateNoPausesConfidence(features *model.TraceFeatures) float64 {
	if features.PauseCount > 0 {
		return 0
	}
	if features.TotalTime <= 2000 {
		return 0
	}
	return math.Min(1.0, float64(features.TotalTime)/5000)
}

func (a *AnomalyDetector) hasInstantJump(traceData *model.TraceData) bool {
	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]
		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(curr.Timestamp-prev.Timestamp) / 1000.0
		if time > 0 && time < 0.01 && distance > 50 {
			return true
		}
	}
	return false
}

func (a *AnomalyDetector) calculateInstantJumpConfidence(traceData *model.TraceData) float64 {
	maxRatio := 0.0
	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]
		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(curr.Timestamp-prev.Timestamp) / 1000.0
		if time > 0 && time < 0.1 {
			ratio := distance / time
			if ratio > maxRatio {
				maxRatio = ratio
			}
		}
	}
	return math.Min(1.0, maxRatio/5000)
}

func (a *AnomalyDetector) isSquareWave(traceData *model.TraceData) bool {
	if len(traceData.Points) < 10 {
		return false
	}
	verticalCount := 0
	horizontalCount := 0
	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]
		dx := math.Abs(curr.X - prev.X)
		dy := math.Abs(curr.Y - prev.Y)
		if dx < 2 && dy > 10 {
			verticalCount++
		} else if dy < 2 && dx > 10 {
			horizontalCount++
		}
	}
	return verticalCount > 3 && horizontalCount > 3
}

func (a *AnomalyDetector) calculateSquareWaveConfidence(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 10 {
		return 0
	}
	verticalCount := 0
	horizontalCount := 0
	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]
		dx := math.Abs(curr.X - prev.X)
		dy := math.Abs(curr.Y - prev.Y)
		if dx < 2 && dy > 10 {
			verticalCount++
		} else if dy < 2 && dx > 10 {
			horizontalCount++
		}
	}
	minCount := math.Min(float64(verticalCount), float64(horizontalCount))
	return math.Min(1.0, minCount/10)
}

func (a *AnomalyDetector) hasReversedMovement(traceData *model.TraceData) bool {
	if len(traceData.Points) < 5 {
		return false
	}
	reversals := 0
	for i := 2; i < len(traceData.Points); i++ {
		p0 := traceData.Points[i-2]
		p1 := traceData.Points[i-1]
		p2 := traceData.Points[i]
		dx1 := p1.X - p0.X
		dy1 := p1.Y - p0.Y
		dx2 := p2.X - p1.X
		dy2 := p2.Y - p1.Y
		if dx1*dx2 < -50 || dy1*dy2 < -50 {
			reversals++
		}
	}
	return reversals > 2
}

func (a *AnomalyDetector) calculateReversedMovementConfidence(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 5 {
		return 0
	}
	reversals := 0
	for i := 2; i < len(traceData.Points); i++ {
		p0 := traceData.Points[i-2]
		p1 := traceData.Points[i-1]
		p2 := traceData.Points[i]
		dx1 := p1.X - p0.X
		dy1 := p1.Y - p0.Y
		dx2 := p2.X - p1.X
		dy2 := p2.Y - p1.Y
		if dx1*dx2 < -50 || dy1*dy2 < -50 {
			reversals++
		}
	}
	return math.Min(1.0, float64(reversals)/5)
}

func (a *AnomalyDetector) isBotPattern(features *model.TraceFeatures) bool {
	riskCount := len(features.RiskFactors)
	return riskCount >= 4 || (riskCount >= 3 && features.AvgSpeed > 500)
}

func (a *AnomalyDetector) calculateBotPatternConfidence(features *model.TraceFeatures) float64 {
	riskCount := len(features.RiskFactors)
	baseScore := float64(riskCount) / 6
	if features.AvgSpeed > 500 {
		baseScore += 0.2
	}
	return math.Min(1.0, baseScore)
}

func (a *AnomalyDetector) isHighFrequency(traceData *model.TraceData) bool {
	if traceData.TotalTime == 0 {
		return false
	}
	frequency := float64(len(traceData.Points)) / float64(traceData.TotalTime) * 1000
	return frequency > 200
}

func (a *AnomalyDetector) calculateHighFrequencyConfidence(traceData *model.TraceData) float64 {
	if traceData.TotalTime == 0 {
		return 0
	}
	frequency := float64(len(traceData.Points)) / float64(traceData.TotalTime) * 1000
	if frequency <= 200 {
		return 0
	}
	return math.Min(1.0, (frequency-200)/300)
}

func (a *AnomalyDetector) isLowResolution(traceData *model.TraceData) bool {
	uniqueX := make(map[int]bool)
	uniqueY := make(map[int]bool)
	for _, p := range traceData.Points {
		uniqueX[int(p.X)] = true
		uniqueY[int(p.Y)] = true
	}
	return len(uniqueX) < 10 || len(uniqueY) < 10
}

func (a *AnomalyDetector) calculateLowResolutionConfidence(traceData *model.TraceData) float64 {
	uniqueX := make(map[int]bool)
	uniqueY := make(map[int]bool)
	for _, p := range traceData.Points {
		uniqueX[int(p.X)] = true
		uniqueY[int(p.Y)] = true
	}
	minUnique := math.Min(float64(len(uniqueX)), float64(len(uniqueY)))
	if minUnique >= 10 {
		return 0
	}
	return math.Min(1.0, (10-minUnique)/10)
}

func (a *AnomalyDetector) GetAnomalySummary(anomalies []AnomalyPattern) map[string]interface{} {
	summary := make(map[string]interface{})
	summary["total_anomalies"] = len(anomalies)

	severityCount := make(map[string]int)
	patternTypes := make([]AnomalyPatternType, 0)
	maxConfidence := 0.0

	for _, anomaly := range anomalies {
		severityCount[anomaly.Severity]++
		patternTypes = append(patternTypes, anomaly.Type)
		if anomaly.Confidence > maxConfidence {
			maxConfidence = anomaly.Confidence
		}
	}

	summary["severity_count"] = severityCount
	summary["pattern_types"] = patternTypes
	summary["max_confidence"] = maxConfidence
	summary["is_suspicious"] = len(anomalies) > 0 && maxConfidence > 0.7

	return summary
}

func (a *AnomalyDetector) isAbnormalPressure(enhanced *EnhancedFeatures) bool {
	if enhanced.PressureVariance < 0.01 && enhanced.PressureConsistency > 0.95 {
		return true
	}
	if enhanced.PressureVariance > 0.5 {
		return true
	}
	return false
}

func (a *AnomalyDetector) calculateAbnormalPressureConfidence(enhanced *EnhancedFeatures) float64 {
	if enhanced.PressureVariance < 0.01 && enhanced.PressureConsistency > 0.95 {
		return math.Min(1.0, (enhanced.PressureConsistency-0.95)*20+0.5)
	}
	if enhanced.PressureVariance > 0.5 {
		return math.Min(1.0, enhanced.PressureVariance)
	}
	return 0.0
}

func (a *AnomalyDetector) isMechanicalClicking(enhanced *EnhancedFeatures) bool {
	if enhanced.ClickCount == 0 {
		return false
	}
	if enhanced.ClickRegularity > 0.95 && enhanced.ClickCount >= 3 {
		return true
	}
	if enhanced.ClickRegularity > 0.90 && enhanced.ClickAreaSize < 1.0 {
		return true
	}
	return false
}

func (a *AnomalyDetector) calculateMechanicalClickingConfidence(enhanced *EnhancedFeatures) float64 {
	if enhanced.ClickCount == 0 {
		return 0.0
	}

	baseScore := 0.0
	if enhanced.ClickRegularity > 0.95 {
		baseScore += 0.6
	} else if enhanced.ClickRegularity > 0.90 {
		baseScore += 0.3
	}

	if enhanced.ClickAreaSize < 1.0 && enhanced.ClickCount >= 3 {
		baseScore += 0.2
	}

	if enhanced.ClickRegularity > 0.98 {
		baseScore += 0.2
	}

	return math.Min(1.0, baseScore)
}

func (a *AnomalyDetector) isRoboticScrolling(enhanced *EnhancedFeatures) bool {
	if enhanced.ScrollCount == 0 {
		return false
	}
	if enhanced.ScrollRegularity > 0.95 && enhanced.ScrollCount >= 3 {
		return true
	}
	if enhanced.ScrollVelocityVariance < 0.1 {
		return true
	}
	return false
}

func (a *AnomalyDetector) calculateRoboticScrollingConfidence(enhanced *EnhancedFeatures) float64 {
	if enhanced.ScrollCount == 0 {
		return 0.0
	}

	baseScore := 0.0
	if enhanced.ScrollRegularity > 0.95 {
		baseScore += 0.6
	} else if enhanced.ScrollRegularity > 0.90 {
		baseScore += 0.3
	}

	if enhanced.ScrollVelocityVariance < 0.1 {
		baseScore += 0.3
	}

	return math.Min(1.0, baseScore)
}

func (a *AnomalyDetector) isUniformCurvature(enhanced *EnhancedFeatures) bool {
	if enhanced.CurvatureVariance < 0.001 && enhanced.CurvatureSkewness < 0.1 {
		return true
	}
	return false
}

func (a *AnomalyDetector) calculateUniformCurvatureConfidence(enhanced *EnhancedFeatures) float64 {
	if enhanced.CurvatureVariance >= 0.001 {
		return 0.0
	}

	confidence := 0.5
	if enhanced.CurvatureSkewness < 0.05 {
		confidence += 0.3
	}
	if enhanced.CurvatureSkewness < 0.01 {
		confidence += 0.2
	}

	return math.Min(1.0, confidence)
}

func (a *AnomalyDetector) isExcessiveFluidity(enhanced *EnhancedFeatures) bool {
	if enhanced.MovementFluidity > 0.95 {
		return true
	}
	return false
}

func (a *AnomalyDetector) calculateExcessiveFluidityConfidence(enhanced *EnhancedFeatures) float64 {
	if enhanced.MovementFluidity <= 0.95 {
		return 0.0
	}

	return math.Min(1.0, (enhanced.MovementFluidity-0.95)*20)
}

func (a *AnomalyDetector) isStereotypedMovement(enhanced *EnhancedFeatures) bool {
	if enhanced.SpatialSpreadX < 0.05 && enhanced.SpatialSpreadY < 0.05 {
		return true
	}
	if enhanced.TemporalIntervalStdDev < 0.1 && enhanced.PauseRatio < 0.05 {
		return true
	}
	return false
}

func (a *AnomalyDetector) calculateStereotypedMovementConfidence(enhanced *EnhancedFeatures) float64 {
	baseScore := 0.0

	if enhanced.SpatialSpreadX < 0.05 && enhanced.SpatialSpreadY < 0.05 {
		baseScore += 0.4
	}

	if enhanced.TemporalIntervalStdDev < 0.1 && enhanced.PauseRatio < 0.05 {
		baseScore += 0.4
	}

	if enhanced.MovementFluidity > 0.9 {
		baseScore += 0.2
	}

	return math.Min(1.0, baseScore)
}
