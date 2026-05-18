package service

import (
	"fmt"
	"math"
)

type BotDetectionResult = EnhancedBotDetectionResult

type EnhancedSliderAnalyzer struct {
	optimizer      *TrajectoryOptimizer
	dtwOptimizer   *OptimizedDTW
	multiScaleDTW  *MultiScaleDTW
	dtwClassifier  *DTWClassifier
}

type EnhancedSliderFeatures struct {
	SpeedProfile      SpeedProfile
	Curvature        CurvatureAnalysis
	Backtrack        BacktrackAnalysis
	Smoothness       SmoothnessAnalysis
	BotScore         float64
	HumanScore       float64
	Confidence       float64
	RiskLevel        string
	Indicators       []string
}

type EnhancedBotDetectionResult struct {
	IsBot             bool
	BotScore          float64
	Confidence        float64
	RiskLevel         string
	Indicators        []string
	SpeedAnomaly      bool
	CurvatureAnomaly  bool
	BacktrackAnomaly  bool
	SmoothnessAnomaly bool
	PatternMatch      string
	SimilarityScore   float64
}

func NewEnhancedSliderAnalyzer() *EnhancedSliderAnalyzer {
	return &EnhancedSliderAnalyzer{
		optimizer:     NewTrajectoryOptimizer(),
		dtwOptimizer:  NewOptimizedDTW(),
		multiScaleDTW: NewMultiScaleDTW(),
		dtwClassifier: NewDTWClassifier(),
	}
}

func (esa *EnhancedSliderAnalyzer) AnalyzeTrajectory(trajectory []SliderPoint, targetPosition int) *EnhancedSliderFeatures {
	features := &EnhancedSliderFeatures{
		Indicators: make([]string, 0),
	}

	if len(trajectory) < 3 {
		features.BotScore = 1.0
		features.HumanScore = 0.0
		features.Confidence = 0.9
		features.RiskLevel = "high"
		features.Indicators = append(features.Indicators, "轨迹数据点不足")
		return features
	}

	features.SpeedProfile = esa.optimizer.AnalyzeSpeedProfile(trajectory)
	features.Curvature = esa.optimizer.AnalyzeCurvature(trajectory)
	features.Backtrack = esa.optimizer.AnalyzeBacktrack(trajectory)
	features.Smoothness = esa.optimizer.AnalyzeSmoothness(trajectory)

	features.BotScore = esa.calculateBotScore(features)
	features.HumanScore = 1.0 - features.BotScore
	features.Confidence = esa.calculateConfidence(features)
	features.RiskLevel = esa.classifyRiskLevel(features.BotScore)

	esa.detectAnomalies(features)

	return features
}

func (esa *EnhancedSliderAnalyzer) DetectBot(trajectory []SliderPoint, targetPosition int) *EnhancedBotDetectionResult {
	result := &EnhancedBotDetectionResult{
		Indicators: make([]string, 0),
	}

	if len(trajectory) < 3 {
		result.IsBot = true
		result.BotScore = 1.0
		result.Confidence = 0.9
		result.RiskLevel = "high"
		result.Indicators = append(result.Indicators, "轨迹数据点不足")
		return result
	}

	speedProfile := esa.optimizer.AnalyzeSpeedProfile(trajectory)
	curvature := esa.optimizer.AnalyzeCurvature(trajectory)
	backtrack := esa.optimizer.AnalyzeBacktrack(trajectory)
	smoothness := esa.optimizer.AnalyzeSmoothness(trajectory)

	result.BotScore = esa.calculateBotScoreFromProfiles(speedProfile, curvature, backtrack, smoothness)
	result.Confidence = esa.calculateConfidenceFromProfiles(speedProfile, curvature, backtrack, smoothness)
	result.RiskLevel = esa.classifyRiskLevel(result.BotScore)
	result.IsBot = result.BotScore > 0.5

	result.SpeedAnomaly = esa.detectSpeedAnomaly(speedProfile, &result.Indicators)
	result.CurvatureAnomaly = esa.detectCurvatureAnomaly(curvature, &result.Indicators)
	result.BacktrackAnomaly = esa.detectBacktrackAnomaly(backtrack, &result.Indicators)
	result.SmoothnessAnomaly = esa.detectSmoothnessAnomaly(smoothness, &result.Indicators)

	esa.detectPatternMatch(trajectory, result)

	return result
}

func (esa *EnhancedSliderAnalyzer) calculateBotScore(features *EnhancedSliderFeatures) float64 {
	score := 0.0
	sp := features.SpeedProfile
	cu := features.Curvature
	bt := features.Backtrack
	sm := features.Smoothness

	if sp.IsSpeedConsistent && sp.AverageSpeed > 100 && sp.AverageSpeed < 3000 {
		score += 0.15
	}

	if sp.SpeedCV < 0.1 && sp.AverageSpeed > 50 {
		score += 0.2
		features.Indicators = append(features.Indicators, "速度变化异常恒定")
	}

	if sp.MaxSpeed > 2000 {
		score += 0.15
		features.Indicators = append(features.Indicators, "检测到超高速移动")
	}

	if cu.AverageCurvature < 0.02 && cu.PeakCount == 0 {
		score += 0.2
		features.Indicators = append(features.Indicators, "轨迹曲率异常低")
	}

	if bt.Count == 0 && bt.TotalDistance == 0 {
		score += 0.1
		features.Indicators = append(features.Indicators, "无回退行为")
	}

	if sm.IsTrajectorySmooth && sm.JitterScore < 0.05 {
		score += 0.2
		features.Indicators = append(features.Indicators, "轨迹异常平滑")
	}

	return math.Min(score, 1.0)
}

func (esa *EnhancedSliderAnalyzer) calculateBotScoreFromProfiles(
	speedProfile SpeedProfile,
	curvature CurvatureAnalysis,
	backtrack BacktrackAnalysis,
	smoothness SmoothnessAnalysis,
) float64 {
	score := 0.0

	if speedProfile.IsSpeedConsistent && speedProfile.AverageSpeed > 100 {
		score += 0.15
	}

	if speedProfile.SpeedCV < 0.1 {
		score += 0.2
	}

	if speedProfile.MaxSpeed > 2000 {
		score += 0.15
	}

	if curvature.AverageCurvature < 0.02 {
		score += 0.2
	}

	if curvature.PeakCount == 0 {
		score += 0.1
	}

	if backtrack.Count == 0 {
		score += 0.1
	}

	if smoothness.IsTrajectorySmooth && smoothness.JitterScore < 0.05 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

func (esa *EnhancedSliderAnalyzer) calculateConfidence(features *EnhancedSliderFeatures) float64 {
	confidence := 0.7

	sp := features.SpeedProfile
	if sp.AverageSpeed > 50 && sp.AverageSpeed < 2000 {
		confidence += 0.1
	}

	if sp.SpeedVariance > 0 {
		confidence += 0.1
	}

	if features.Curvature.CurvatureVariance > 0 {
		confidence += 0.1
	}

	return math.Min(confidence, 0.99)
}

func (esa *EnhancedSliderAnalyzer) calculateConfidenceFromProfiles(
	speedProfile SpeedProfile,
	curvature CurvatureAnalysis,
	backtrack BacktrackAnalysis,
	smoothness SmoothnessAnalysis,
) float64 {
	confidence := 0.7

	if speedProfile.AverageSpeed > 50 && speedProfile.AverageSpeed < 2000 {
		confidence += 0.1
	}

	if curvature.CurvatureVariance > 0 {
		confidence += 0.1
	}

	if smoothness.OverallScore > 0 {
		confidence += 0.1
	}

	return math.Min(confidence, 0.99)
}

func (esa *EnhancedSliderAnalyzer) classifyRiskLevel(score float64) string {
	if score > 0.8 {
		return "critical"
	}
	if score > 0.6 {
		return "high"
	}
	if score > 0.4 {
		return "medium"
	}
	if score > 0.2 {
		return "low"
	}
	return "minimal"
}

func (esa *EnhancedSliderAnalyzer) detectAnomalies(features *EnhancedSliderFeatures) {
	sp := features.SpeedProfile

	if sp.MaxSpeed > 3000 {
		features.Indicators = append(features.Indicators, "极端超速")
	}

	if sp.SpeedOutliers > 10 && sp.SpeedVariance < 100 {
		features.Indicators = append(features.Indicators, "速度异常波动")
	}

	if sp.ZeroSpeedCount > 5 && sp.SpeedCV < 0.1 {
		features.Indicators = append(features.Indicators, "大量停顿")
	}

	cu := features.Curvature

	if cu.SharpTurnCount == 0 && cu.PeakCount == 0 {
		features.Indicators = append(features.Indicators, "无急转弯")
	}

	if cu.CurvatureEntropy < 1.0 {
		features.Indicators = append(features.Indicators, "曲率分布异常")
	}

	sm := features.Smoothness

	if sm.DirectionStability > 0.95 {
		features.Indicators = append(features.Indicators, "方向异常稳定")
	}
}

func (esa *EnhancedSliderAnalyzer) detectSpeedAnomaly(speedProfile SpeedProfile, indicators *[]string) bool {
	isAnomaly := false

	if speedProfile.SpeedCV < 0.05 && speedProfile.AverageSpeed > 100 {
		isAnomaly = true
		*indicators = append(*indicators, "速度变异系数异常低")
	}

	if speedProfile.MaxSpeed > 2500 {
		isAnomaly = true
		*indicators = append(*indicators, "最大速度异常高")
	}

	if speedProfile.AccelerationMax > 5000 || speedProfile.DecelerationMax > 5000 {
		isAnomaly = true
		*indicators = append(*indicators, "加速度异常")
	}

	if speedProfile.JerkAvg < 0.1 && speedProfile.JerkMax < 1.0 {
		isAnomaly = true
		*indicators = append(*indicators, "加加速度异常平稳")
	}

	return isAnomaly
}

func (esa *EnhancedSliderAnalyzer) detectCurvatureAnomaly(curvature CurvatureAnalysis, indicators *[]string) bool {
	isAnomaly := false

	if curvature.AverageCurvature < 0.01 {
		isAnomaly = true
		*indicators = append(*indicators, "平均曲率异常低")
	}

	if curvature.PeakCount == 0 && curvature.SharpTurnCount == 0 {
		isAnomaly = true
		*indicators = append(*indicators, "无曲率峰值")
	}

	if curvature.CurvatureVariance < 0.0001 {
		isAnomaly = true
		*indicators = append(*indicators, "曲率方差异常低")
	}

	if curvature.CurvatureEntropy < 1.5 {
		isAnomaly = true
		*indicators = append(*indicators, "曲率熵异常低")
	}

	return isAnomaly
}

func (esa *EnhancedSliderAnalyzer) detectBacktrackAnomaly(backtrack BacktrackAnalysis, indicators *[]string) bool {
	isAnomaly := false

	if backtrack.Count == 0 {
		isAnomaly = true
		*indicators = append(*indicators, "无回退行为")
	}

	if backtrack.Count > 10 {
		isAnomaly = true
		*indicators = append(*indicators, "回退次数过多")
	}

	if backtrack.MaxBacktrackDist > 100 {
		isAnomaly = true
		*indicators = append(*indicators, "单次回退距离过大")
	}

	return isAnomaly
}

func (esa *EnhancedSliderAnalyzer) detectSmoothnessAnomaly(smoothness SmoothnessAnalysis, indicators *[]string) bool {
	isAnomaly := false

	if smoothness.OverallScore > 0.95 {
		isAnomaly = true
		*indicators = append(*indicators, "轨迹平滑度异常高")
	}

	if smoothness.JitterScore < 0.01 {
		isAnomaly = true
		*indicators = append(*indicators, "抖动分数异常低")
	}

	if smoothness.DirectionStability > 0.98 {
		isAnomaly = true
		*indicators = append(*indicators, "方向稳定性异常高")
	}

	if smoothness.PathRegularity > 0.95 {
		isAnomaly = true
		*indicators = append(*indicators, "路径规律性异常高")
	}

	return isAnomaly
}

func (esa *EnhancedSliderAnalyzer) detectPatternMatch(trajectory []SliderPoint, result *EnhancedBotDetectionResult) {
	pattern, similarity := esa.dtwClassifier.Classify(trajectory)
	result.PatternMatch = pattern
	result.SimilarityScore = similarity

	if similarity > 0.85 {
		result.Indicators = append(result.Indicators, "轨迹与已知机器人模式高度相似")
	}
}

func (esa *EnhancedSliderAnalyzer) CompareWithTemplate(template []SliderPoint, candidate []SliderPoint) float64 {
	return esa.dtwOptimizer.ComputeSimilarity(template, candidate)
}

func (esa *EnhancedSliderAnalyzer) ComputeMultiScaleSimilarity(traj1, traj2 []SliderPoint) float64 {
	distance := esa.multiScaleDTW.ComputeDistance(traj1, traj2)
	maxDist := 1000.0
	similarity := 1.0 - math.Min(distance/maxDist, 1.0)
	return math.Max(0, similarity)
}

func (esa *EnhancedSliderAnalyzer) AddHumanTemplate(name string, trajectory []SliderPoint) {
	esa.dtwClassifier.AddTemplate(name, trajectory)
}

func (esa *EnhancedSliderAnalyzer) ClassifyTrajectory(trajectory []SliderPoint) (string, float64) {
	return esa.dtwClassifier.Classify(trajectory)
}

func (esa *EnhancedSliderAnalyzer) GenerateComprehensiveReport(trajectory []SliderPoint, targetPosition int) string {
	result := esa.DetectBot(trajectory, targetPosition)
	features := esa.AnalyzeTrajectory(trajectory, targetPosition)

	report := "=== 增强版滑块轨迹分析报告 ===\n\n"
	report += "=== 基本信息 ===\n"
	report += fmt.Sprintf("轨迹点数: %d\n", len(trajectory))
	if len(trajectory) > 1 {
		report += fmt.Sprintf("总时长: %d ms\n", trajectory[len(trajectory)-1].Timestamp-trajectory[0].Timestamp)
	}

	report += "\n=== 速度分析 ===\n"
	sp := features.SpeedProfile
	report += fmt.Sprintf("平均速度: %.2f px/s\n", sp.AverageSpeed)
	report += fmt.Sprintf("最大速度: %.2f px/s\n", sp.MaxSpeed)
	report += fmt.Sprintf("速度标准差: %.4f\n", sp.SpeedStdDev)
	report += fmt.Sprintf("速度变异系数: %.4f\n", sp.SpeedCV)
	report += fmt.Sprintf("平均加速度: %.4f\n", sp.AccelerationAvg)
	report += fmt.Sprintf("平均减速度: %.4f\n", sp.DecelerationAvg)
	report += fmt.Sprintf("平均加加速度: %.4f\n", sp.JerkAvg)

	report += "\n=== 曲率分析 ===\n"
	cu := features.Curvature
	report += fmt.Sprintf("平均曲率: %.6f\n", cu.AverageCurvature)
	report += fmt.Sprintf("曲率方差: %.8f\n", cu.CurvatureVariance)
	report += fmt.Sprintf("曲率峰值数: %d\n", cu.PeakCount)
	report += fmt.Sprintf("急转弯次数: %d\n", cu.SharpTurnCount)
	report += fmt.Sprintf("曲率熵: %.4f\n", cu.CurvatureEntropy)

	report += "\n=== 回退分析 ===\n"
	bt := features.Backtrack
	report += fmt.Sprintf("回退次数: %d\n", bt.Count)
	report += fmt.Sprintf("总回退距离: %.2f px\n", bt.TotalDistance)
	report += fmt.Sprintf("最大回退距离: %.2f px\n", bt.MaxBacktrackDist)

	report += "\n=== 平滑度分析 ===\n"
	sm := features.Smoothness
	report += fmt.Sprintf("综合平滑度: %.4f\n", sm.OverallScore)
	report += fmt.Sprintf("抖动分数: %.6f\n", sm.JitterScore)
	report += fmt.Sprintf("方向稳定性: %.4f\n", sm.DirectionStability)
	report += fmt.Sprintf("路径规律性: %.4f\n", sm.PathRegularity)

	report += "\n=== 机器人检测结果 ===\n"
	report += fmt.Sprintf("机器人概率: %.4f\n", result.BotScore)
	report += fmt.Sprintf("人类概率: %.4f\n", features.HumanScore)
	report += fmt.Sprintf("置信度: %.4f\n", result.Confidence)
	report += fmt.Sprintf("风险等级: %s\n", result.RiskLevel)
	report += fmt.Sprintf("判定为机器人: %v\n", result.IsBot)

	if len(result.Indicators) > 0 {
		report += "\n=== 异常指标 ===\n"
		for _, indicator := range result.Indicators {
			report += "- " + indicator + "\n"
		}
	}

	if result.PatternMatch != "" {
		report += "\n=== 模式匹配 ===\n"
		report += fmt.Sprintf("匹配模式: %s\n", result.PatternMatch)
		report += fmt.Sprintf("相似度: %.4f\n", result.SimilarityScore)
	}

	return report
}

func (esa *EnhancedSliderAnalyzer) ValidateTrajectory(trajectory []SliderPoint) (bool, string) {
	if len(trajectory) < 10 {
		return false, "轨迹数据点不足"
	}

	if len(trajectory) > 1 {
		duration := trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp
		if duration < 100 {
			return false, "轨迹持续时间过短"
		}
		if duration > 60000 {
			return false, "轨迹持续时间过长"
		}
	}

	totalDistance := 0.0
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	if totalDistance < 50 {
		return false, "移动总距离过短"
	}

	maxSpeed := 0.0
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			speed := dist / dt * 1000
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
	}

	if maxSpeed > 10000 {
		return false, "检测到超高速移动"
	}

	return true, "轨迹验证通过"
}
