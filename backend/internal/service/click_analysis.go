package service

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

type ClickVerification struct {
	Clicks       []ClickData   `json:"clicks"`
	TargetImages []TargetImage `json:"target_images"`
	Result       *ClickAnalysisResult `json:"result"`
}

type SliderClickData struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
	Index     int   `json:"index"`
}

type ClickData = SliderClickData

type TargetImage struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ClickAnalysisResult struct {
	ClickPattern       *ClickPatternAnalysis `json:"click_pattern"`
	TimingAnalysis     *TimingAnalysis       `json:"timing_analysis"`
	AccuracyAnalysis   *AccuracyAnalysis     `json:"accuracy_analysis"`
	AnomalyScore       float64              `json:"anomaly_score"`
	MLScore            float64              `json:"ml_score"`
	OverallRiskScore   float64              `json:"overall_risk_score"`
	IsBot              bool                 `json:"is_bot"`
	Confidence         float64              `json:"confidence"`
	RiskIndicators     []string             `json:"risk_indicators"`
	AnomalyDetections  []string             `json:"anomaly_detections"`
}

type ClickPatternAnalysis struct {
	ClickCount          int           `json:"click_count"`
	ClickIntervals      []float64    `json:"click_intervals"`
	AverageInterval     float64      `json:"average_interval"`
	IntervalVariance    float64      `json:"interval_variance"`
	IntervalStdDev      float64      `json:"interval_std_dev"`
	Regularity          float64      `json:"regularity"`
	PositionDistribution *PositionDistribution `json:"position_distribution"`
	ClickSequence       string       `json:"click_sequence"`
	SequencePattern     string       `json:"sequence_pattern"`
	ClusteringScore     float64      `json:"clustering_score"`
}

type PositionDistribution struct {
	XMean     float64 `json:"x_mean"`
	YMean     float64 `json:"y_mean"`
	XVariance float64 `json:"x_variance"`
	YVariance float64 `json:"y_variance"`
	XEntropy  float64 `json:"x_entropy"`
	YEntropy  float64 `json:"y_entropy"`
	SpreadX   float64 `json:"spread_x"`
	SpreadY   float64 `json:"spread_y"`
}

type TimingAnalysis struct {
	TotalDuration    int64        `json:"total_duration"`
	AverageDuration  float64      `json:"average_duration"`
	DurationVariance float64     `json:"duration_variance"`
	ResponseTimes    []float64   `json:"response_times"`
	FirstClickDelay  int64        `json:"first_click_delay"`
	HesitationTimes  []float64   `json:"hesitation_times"`
	TimingPattern    string      `json:"timing_pattern"`
	IsRhythmic       bool        `json:"is_rhythmic"`
}

type AccuracyAnalysis struct {
	CorrectClicks    int           `json:"correct_clicks"`
	TotalClicks      int           `json:"total_clicks"`
	Accuracy         float64       `json:"accuracy"`
	MissDistances    []float64     `json:"miss_distances"`
	AverageMissDistance float64   `json:"average_miss_distance"`
	TargetHits       []bool        `json:"target_hits"`
	Precision        float64       `json:"precision"`
}

type ClickAnalyzer struct {
	model *ClickMLModel
}

type ClickMLModel struct {
	weights   []float64
	bias      float64
	isTrained bool
}

func NewClickAnalyzer() *ClickAnalyzer {
	return &ClickAnalyzer{
		model: NewClickMLModel(),
	}
}

func NewClickMLModel() *ClickMLModel {
	return &ClickMLModel{
		weights:   []float64{0.1, 0.15, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.05},
		bias:      -5.0,
		isTrained: false,
	}
}

func (ca *ClickAnalyzer) AnalyzeClickVerification(verification *ClickVerification) *ClickAnalysisResult {
	if verification == nil || len(verification.Clicks) == 0 {
		return &ClickAnalysisResult{
			IsBot:       true,
			Confidence:  0.9,
			RiskIndicators: []string{"无点击数据"},
		}
	}

	result := &ClickAnalysisResult{
		ClickPattern:     ca.analyzeClickPattern(verification),
		TimingAnalysis:   ca.analyzeTiming(verification),
		AccuracyAnalysis:  ca.analyzeAccuracy(verification),
		RiskIndicators:   make([]string, 0),
		AnomalyDetections: make([]string, 0),
	}

	result.AnomalyScore = ca.detectAnomalies(result)
	result.MLScore = ca.model.Predict(result)
	result.OverallRiskScore = ca.calculateOverallRiskScore(result)
	result.IsBot = result.OverallRiskScore > 0.5
	result.Confidence = ca.calculateConfidence(result)

	return result
}

func (ca *ClickAnalyzer) analyzeClickPattern(verification *ClickVerification) *ClickPatternAnalysis {
	pattern := &ClickPatternAnalysis{
		ClickCount: len(verification.Clicks),
	}

	if len(verification.Clicks) < 2 {
		return pattern
	}

	pattern.ClickIntervals = ca.calculateClickIntervals(verification.Clicks)
	if len(pattern.ClickIntervals) > 0 {
		pattern.AverageInterval = ca.mean(pattern.ClickIntervals)
		pattern.IntervalVariance = ca.variance(pattern.ClickIntervals)
		pattern.IntervalStdDev = math.Sqrt(pattern.IntervalVariance)

		if pattern.AverageInterval > 0 {
			pattern.Regularity = 1.0 - math.Min(pattern.IntervalStdDev/pattern.AverageInterval, 1.0)
		}
	}

	pattern.PositionDistribution = ca.analyzePositionDistribution(verification.Clicks)

	pattern.ClickSequence = ca.generateClickSequence(verification.Clicks)
	pattern.SequencePattern = ca.classifySequencePattern(pattern.ClickSequence)

	pattern.ClusteringScore = ca.calculateClusteringScore(verification.Clicks)

	return pattern
}

func (ca *ClickAnalyzer) calculateClickIntervals(clicks []ClickData) []float64 {
	intervals := make([]float64, 0)
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		intervals = append(intervals, interval)
	}
	return intervals
}

func (ca *ClickAnalyzer) analyzePositionDistribution(clicks []ClickData) *PositionDistribution {
	distribution := &PositionDistribution{}

	if len(clicks) == 0 {
		return distribution
	}

	xValues := make([]float64, len(clicks))
	yValues := make([]float64, len(clicks))

	for i, click := range clicks {
		xValues[i] = float64(click.X)
		yValues[i] = float64(click.Y)
	}

	distribution.XMean = ca.mean(xValues)
	distribution.YMean = ca.mean(yValues)
	distribution.XVariance = ca.variance(xValues)
	distribution.YVariance = ca.variance(yValues)

	distribution.XEntropy = ca.calculateEntropy(xValues, 10)
	distribution.YEntropy = ca.calculateEntropy(yValues, 10)

	distribution.SpreadX = ca.max(xValues) - ca.min(xValues)
	distribution.SpreadY = ca.max(yValues) - ca.min(yValues)

	return distribution
}

func (ca *ClickAnalyzer) calculateEntropy(values []float64, bins int) float64 {
	if len(values) == 0 {
		return 0
	}

	minVal := ca.min(values)
	maxVal := ca.max(values)

	if maxVal <= minVal {
		return 0
	}

	bucketCounts := make([]int, bins)
	binWidth := (maxVal - minVal) / float64(bins)

	for _, v := range values {
		bin := int((v - minVal) / binWidth)
		if bin >= bins {
			bin = bins - 1
		}
		if bin < 0 {
			bin = 0
		}
		bucketCounts[bin]++
	}

	entropy := 0.0
	total := float64(len(values))

	for _, count := range bucketCounts {
		if count > 0 {
			p := float64(count) / total
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (ca *ClickAnalyzer) generateClickSequence(clicks []ClickData) string {
	if len(clicks) == 0 {
		return ""
	}

	positions := make([]string, len(clicks))
	for i, click := range clicks {
		if i == 0 {
			positions[i] = "start"
		} else {
			prevX := float64(clicks[i-1].X)
			prevY := float64(clicks[i-1].Y)
			currX := float64(click.X)
			currY := float64(click.Y)

			dx := currX - prevX
			dy := currY - prevY

			if math.Abs(dx) > math.Abs(dy) {
				if dx > 0 {
					positions[i] = "right"
				} else {
					positions[i] = "left"
				}
			} else {
				if dy > 0 {
					positions[i] = "down"
				} else {
					positions[i] = "up"
				}
			}
		}
	}

	return strings.Join(positions, "->")
}

func (ca *ClickAnalyzer) classifySequencePattern(sequence string) string {
	if sequence == "" {
		return "unknown"
	}

	parts := strings.Split(sequence, "->")
	if len(parts) < 2 {
		return "single"
	}

	uniqueDirections := make(map[string]bool)
	for _, part := range parts {
		if part != "start" {
			uniqueDirections[part] = true
		}
	}

	if len(uniqueDirections) == 1 {
		return "linear"
	}

	repeats := 0
	for i := 1; i < len(parts); i++ {
		if parts[i] == parts[i-1] && parts[i] != "start" {
			repeats++
		}
	}

	if repeats > len(parts)/3 {
		return "repeated"
	}

	return "varied"
}

func (ca *ClickAnalyzer) calculateClusteringScore(clicks []ClickData) float64 {
	if len(clicks) < 2 {
		return 0
	}

	centroidX := 0.0
	centroidY := 0.0
	for _, click := range clicks {
		centroidX += float64(click.X)
		centroidY += float64(click.Y)
	}
	centroidX /= float64(len(clicks))
	centroidY /= float64(len(clicks))

	variance := 0.0
	for _, click := range clicks {
		dx := float64(click.X) - centroidX
		dy := float64(click.Y) - centroidY
		variance += dx*dx + dy*dy
	}
	variance /= float64(len(clicks))

	stdDev := math.Sqrt(variance)

	normalizedSpread := stdDev / 500.0
	if normalizedSpread > 1 {
		normalizedSpread = 1
	}

	return normalizedSpread
}

func (ca *ClickAnalyzer) analyzeTiming(verification *ClickVerification) *TimingAnalysis {
	timing := &TimingAnalysis{}

	if len(verification.Clicks) == 0 {
		return timing
	}

	if len(verification.Clicks) >= 2 {
		timing.TotalDuration = verification.Clicks[len(verification.Clicks)-1].Timestamp - verification.Clicks[0].Timestamp
	}

	timing.ResponseTimes = ca.calculateResponseTimes(verification.Clicks)
	if len(timing.ResponseTimes) > 0 {
		timing.AverageDuration = ca.mean(timing.ResponseTimes)
		timing.DurationVariance = ca.variance(timing.ResponseTimes)
	}

	if len(verification.Clicks) > 0 {
		timing.FirstClickDelay = verification.Clicks[0].Timestamp
	}

	timing.HesitationTimes = ca.calculateHesitationTimes(verification.Clicks)

	timing.TimingPattern = ca.classifyTimingPattern(timing)
	timing.IsRhythmic = ca.isTimingRhythmic(timing)

	return timing
}

func (ca *ClickAnalyzer) calculateResponseTimes(clicks []ClickData) []float64 {
	times := make([]float64, 0)
	for i := 1; i < len(clicks); i++ {
		times = append(times, float64(clicks[i].Timestamp-clicks[i-1].Timestamp))
	}
	return times
}

func (ca *ClickAnalyzer) calculateHesitationTimes(clicks []ClickData) []float64 {
	hesitations := make([]float64, 0)
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		if interval > 300 {
			hesitations = append(hesitations, interval)
		}
	}
	return hesitations
}

func (ca *ClickAnalyzer) classifyTimingPattern(timing *TimingAnalysis) string {
	if timing.TotalDuration == 0 {
		return "unknown"
	}

	if timing.TotalDuration < 1000 {
		return "very_fast"
	}

	if timing.TotalDuration < 3000 {
		return "fast"
	}

	if timing.TotalDuration < 8000 {
		return "normal"
	}

	return "slow"
}

func (ca *ClickAnalyzer) isTimingRhythmic(timing *TimingAnalysis) bool {
	if len(timing.ResponseTimes) < 3 {
		return false
	}

	mean := timing.AverageDuration
	if mean == 0 {
		return false
	}

	cv := math.Sqrt(timing.DurationVariance) / mean

	return cv < 0.15
}

func (ca *ClickAnalyzer) analyzeAccuracy(verification *ClickVerification) *AccuracyAnalysis {
	accuracy := &AccuracyAnalysis{}

	if len(verification.Clicks) == 0 {
		return accuracy
	}

	accuracy.TotalClicks = len(verification.Clicks)
	accuracy.TargetHits = make([]bool, len(verification.Clicks))
	accuracy.MissDistances = make([]float64, 0)

	for i, click := range verification.Clicks {
		isHit := false

		for _, target := range verification.TargetImages {
			dx := float64(click.X) - (float64(target.X) + float64(target.Width)/2)
			dy := float64(click.Y) - (float64(target.Y) + float64(target.Height)/2)
			distance := math.Sqrt(dx*dx + dy*dy)

			hitRadius := float64(target.Width+target.Height) / 4

			if distance < hitRadius {
				isHit = true
				break
			}

			accuracy.MissDistances = append(accuracy.MissDistances, distance)
		}

		accuracy.TargetHits[i] = isHit
		if isHit {
			accuracy.CorrectClicks++
		}
	}

	if accuracy.TotalClicks > 0 {
		accuracy.Accuracy = float64(accuracy.CorrectClicks) / float64(accuracy.TotalClicks)
	}

	if len(accuracy.MissDistances) > 0 {
		accuracy.AverageMissDistance = ca.mean(accuracy.MissDistances)
	}

	precision := 0.0
	if len(verification.TargetImages) > 0 {
		precision = float64(accuracy.CorrectClicks) / float64(len(verification.TargetImages))
	}
	accuracy.Precision = math.Min(precision, 1.0)

	return accuracy
}

func (cm *ClickMLModel) Predict(result *ClickAnalysisResult) float64 {
	if result == nil {
		return 0.5
	}

	score := cm.bias

	if result.ClickPattern != nil {
		if result.ClickPattern.Regularity > 0.95 {
			score += 0.2
		}

		if result.ClickPattern.ClusteringScore < 0.2 {
			score += 0.15
		}
	}

	if result.TimingAnalysis != nil {
		if result.TimingAnalysis.IsRhythmic {
			score += 0.15
		}

		if result.TimingAnalysis.TimingPattern == "very_fast" {
			score += 0.1
		}

		if len(result.TimingAnalysis.HesitationTimes) == 0 && result.TimingAnalysis.TotalDuration > 1000 {
			score += 0.1
		}
	}

	if result.AccuracyAnalysis != nil {
		if result.AccuracyAnalysis.Accuracy > 0.9 {
			score += 0.1
		}

		if result.AccuracyAnalysis.AverageMissDistance < 10 {
			score += 0.1
		}
	}

	return 1.0 / (1.0 + math.Exp(-score))
}

func (ca *ClickAnalyzer) detectAnomalies(result *ClickAnalysisResult) float64 {
	anomalyScore := 0.0
	anomalyCount := 0

	if result.ClickPattern != nil {
		if result.ClickPattern.Regularity > 0.98 {
			anomalyScore += 0.25
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "点击间隔过于规律")
		}

		if result.ClickPattern.SequencePattern == "linear" && len(result.ClickPattern.ClickIntervals) > 3 {
			anomalyScore += 0.15
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "点击方向过于线性")
		}

		if result.ClickPattern.ClusteringScore < 0.1 {
			anomalyScore += 0.1
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "点击位置过于集中")
		}

		if result.ClickPattern.PositionDistribution != nil {
			if result.ClickPattern.PositionDistribution.XEntropy < 1.0 {
				anomalyScore += 0.1
				anomalyCount++
				result.AnomalyDetections = append(result.AnomalyDetections, "X轴位置熵过低")
			}

			if result.ClickPattern.PositionDistribution.YEntropy < 1.0 {
				anomalyScore += 0.1
				anomalyCount++
				result.AnomalyDetections = append(result.AnomalyDetections, "Y轴位置熵过低")
			}
		}
	}

	if result.TimingAnalysis != nil {
		if result.TimingAnalysis.IsRhythmic {
			anomalyScore += 0.2
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "点击节奏过于规律")
		}

		if result.TimingAnalysis.TimingPattern == "very_fast" {
			anomalyScore += 0.15
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "点击速度异常快")
		}

		if len(result.TimingAnalysis.HesitationTimes) == 0 && result.TimingAnalysis.TotalDuration > 2000 {
			anomalyScore += 0.1
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "无犹豫时间")
		}
	}

	if result.AccuracyAnalysis != nil {
		if result.AccuracyAnalysis.Accuracy == 1.0 && result.AccuracyAnalysis.TotalClicks > 3 {
			anomalyScore += 0.15
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "完美命中")
		}

		if result.AccuracyAnalysis.AverageMissDistance < 5 && result.AccuracyAnalysis.TotalClicks > 2 {
			anomalyScore += 0.1
			anomalyCount++
			result.AnomalyDetections = append(result.AnomalyDetections, "点击精度异常高")
		}
	}

	if anomalyCount > 0 {
		return anomalyScore / float64(anomalyCount)
	}

	return 0.0
}

func (ca *ClickAnalyzer) calculateOverallRiskScore(result *ClickAnalysisResult) float64 {
	riskScore := 0.0

	riskScore += result.AnomalyScore * 0.35

	riskScore += result.MLScore * 0.35

	if result.ClickPattern != nil {
		if result.ClickPattern.Regularity > 0.95 {
			riskScore += 0.15
		}

		if result.ClickPattern.ClusteringScore < 0.2 {
			riskScore += 0.1
		}
	}

	if result.TimingAnalysis != nil {
		if result.TimingAnalysis.IsRhythmic {
			riskScore += 0.15
		}

		if result.TimingAnalysis.TimingPattern == "very_fast" {
			riskScore += 0.1
		}
	}

	if result.AccuracyAnalysis != nil {
		if result.AccuracyAnalysis.Accuracy == 1.0 && result.AccuracyAnalysis.TotalClicks > 2 {
			riskScore += 0.1
		}
	}

	return math.Min(riskScore, 1.0)
}

func (ca *ClickAnalyzer) calculateConfidence(result *ClickAnalysisResult) float64 {
	confidence := 0.7

	if result.ClickPattern != nil && result.ClickPattern.ClickCount > 3 {
		confidence += 0.1
	}

	if result.TimingAnalysis != nil && result.TimingAnalysis.TotalDuration > 500 {
		confidence += 0.1
	}

	if result.AccuracyAnalysis != nil && result.AccuracyAnalysis.TotalClicks > 2 {
		confidence += 0.1
	}

	if result.AnomalyScore > 0.3 || result.MLScore > 0.6 {
		confidence += 0.05
	}

	return math.Min(confidence, 0.99)
}

func (ca *ClickAnalyzer) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (ca *ClickAnalyzer) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := ca.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

func (ca *ClickAnalyzer) max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (ca *ClickAnalyzer) min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

type ClickPatternDetector struct {
	patterns []ClickVerifyPattern
}

type ClickVerifyPattern struct {
	Name      string
	Condition func(*ClickAnalysisResult) bool
	Weight    float64
}

func NewClickPatternDetector() *ClickPatternDetector {
	return &ClickPatternDetector{
		patterns: []ClickVerifyPattern{
			{
				Name: "perfect_regular",
				Condition: func(r *ClickAnalysisResult) bool {
					return r.ClickPattern != nil && r.ClickPattern.Regularity > 0.98
				},
				Weight: 0.3,
			},
			{
				Name: "machine_precision",
				Condition: func(r *ClickAnalysisResult) bool {
					return r.AccuracyAnalysis != nil && r.AccuracyAnalysis.Accuracy == 1.0 && r.AccuracyAnalysis.TotalClicks > 2
				},
				Weight: 0.25,
			},
			{
				Name: "rhythmic_timing",
				Condition: func(r *ClickAnalysisResult) bool {
					return r.TimingAnalysis != nil && r.TimingAnalysis.IsRhythmic
				},
				Weight: 0.2,
			},
			{
				Name: "too_fast",
				Condition: func(r *ClickAnalysisResult) bool {
					return r.TimingAnalysis != nil && r.TimingAnalysis.TimingPattern == "very_fast"
				},
				Weight: 0.15,
			},
			{
				Name: "no_hesitation",
				Condition: func(r *ClickAnalysisResult) bool {
					return r.TimingAnalysis != nil && len(r.TimingAnalysis.HesitationTimes) == 0 && r.TimingAnalysis.TotalDuration > 2000
				},
				Weight: 0.1,
			},
		},
	}
}

func (cpd *ClickPatternDetector) DetectPatterns(result *ClickAnalysisResult) []string {
	detected := make([]string, 0)
	for _, pattern := range cpd.patterns {
		if pattern.Condition(result) {
			detected = append(detected, pattern.Name)
		}
	}
	return detected
}

func (ca *ClickAnalyzer) GenerateReport(result *ClickAnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("=== 点选验证分析报告 ===\n\n")

	if result.ClickPattern != nil {
		sb.WriteString("点击模式分析:\n")
		sb.WriteString(fmt.Sprintf("  点击次数: %d\n", result.ClickPattern.ClickCount))
		sb.WriteString(fmt.Sprintf("  平均间隔: %.2f ms\n", result.ClickPattern.AverageInterval))
		sb.WriteString(fmt.Sprintf("  间隔方差: %.2f\n", result.ClickPattern.IntervalVariance))
		sb.WriteString(fmt.Sprintf("  间隔标准差: %.2f ms\n", result.ClickPattern.IntervalStdDev))
		sb.WriteString(fmt.Sprintf("  规律性: %.4f\n", result.ClickPattern.Regularity))
		sb.WriteString(fmt.Sprintf("  聚集分数: %.4f\n", result.ClickPattern.ClusteringScore))
		sb.WriteString(fmt.Sprintf("  点击序列: %s\n", result.ClickPattern.ClickSequence))
		sb.WriteString(fmt.Sprintf("  序列模式: %s\n", result.ClickPattern.SequencePattern))

		if result.ClickPattern.PositionDistribution != nil {
			sb.WriteString("  位置分布:\n")
			sb.WriteString(fmt.Sprintf("    X均值: %.2f, Y均值: %.2f\n", result.ClickPattern.PositionDistribution.XMean, result.ClickPattern.PositionDistribution.YMean))
			sb.WriteString(fmt.Sprintf("    X方差: %.2f, Y方差: %.2f\n", result.ClickPattern.PositionDistribution.XVariance, result.ClickPattern.PositionDistribution.YVariance))
			sb.WriteString(fmt.Sprintf("    X熵: %.4f, Y熵: %.4f\n", result.ClickPattern.PositionDistribution.XEntropy, result.ClickPattern.PositionDistribution.YEntropy))
			sb.WriteString(fmt.Sprintf("    X范围: %.2f, Y范围: %.2f\n", result.ClickPattern.PositionDistribution.SpreadX, result.ClickPattern.PositionDistribution.SpreadY))
		}
	}

	if result.TimingAnalysis != nil {
		sb.WriteString("\n时序分析:\n")
		sb.WriteString(fmt.Sprintf("  总时长: %d ms\n", result.TimingAnalysis.TotalDuration))
		sb.WriteString(fmt.Sprintf("  平均响应时间: %.2f ms\n", result.TimingAnalysis.AverageDuration))
		sb.WriteString(fmt.Sprintf("  时序方差: %.2f\n", result.TimingAnalysis.DurationVariance))
		sb.WriteString(fmt.Sprintf("  首次点击延迟: %d ms\n", result.TimingAnalysis.FirstClickDelay))
		sb.WriteString(fmt.Sprintf("  时序模式: %s\n", result.TimingAnalysis.TimingPattern))
		sb.WriteString(fmt.Sprintf("  节奏性: %v\n", result.TimingAnalysis.IsRhythmic))
		sb.WriteString(fmt.Sprintf("  犹豫次数: %d\n", len(result.TimingAnalysis.HesitationTimes)))
	}

	if result.AccuracyAnalysis != nil {
		sb.WriteString("\n准确性分析:\n")
		sb.WriteString(fmt.Sprintf("  正确点击: %d / %d\n", result.AccuracyAnalysis.CorrectClicks, result.AccuracyAnalysis.TotalClicks))
		sb.WriteString(fmt.Sprintf("  准确率: %.4f\n", result.AccuracyAnalysis.Accuracy))
		sb.WriteString(fmt.Sprintf("  平均偏移距离: %.2f px\n", result.AccuracyAnalysis.AverageMissDistance))
		sb.WriteString(fmt.Sprintf("  精确度: %.4f\n", result.AccuracyAnalysis.Precision))
	}

	sb.WriteString("\n风险评估:\n")
	sb.WriteString(fmt.Sprintf("  异常分数: %.4f\n", result.AnomalyScore))
	sb.WriteString(fmt.Sprintf("  机器学习分数: %.4f\n", result.MLScore))
	sb.WriteString(fmt.Sprintf("  综合风险分数: %.4f\n", result.OverallRiskScore))
	sb.WriteString(fmt.Sprintf("  判定为机器人: %v\n", result.IsBot))
	sb.WriteString(fmt.Sprintf("  置信度: %.4f\n", result.Confidence))

	if len(result.RiskIndicators) > 0 {
		sb.WriteString("\n风险指标:\n")
		for _, indicator := range result.RiskIndicators {
			sb.WriteString(fmt.Sprintf("  - %s\n", indicator))
		}
	}

	if len(result.AnomalyDetections) > 0 {
		sb.WriteString("\n异常检测:\n")
		for _, detection := range result.AnomalyDetections {
			sb.WriteString(fmt.Sprintf("  - %s\n", detection))
		}
	}

	return sb.String()
}

func GenerateHumanLikeClickData(targets []TargetImage, duration int64) []SliderClickData {
	clicks := make([]SliderClickData, 0)

	numClicks := len(targets)
	if numClicks == 0 {
		numClicks = 3 + rand.Intn(3)
	}

	interval := duration / int64(numClicks)

	for i := 0; i < numClicks; i++ {
		var x, y int

		if i < len(targets) {
			target := targets[i]
			offsetX := rand.Intn(target.Width) - target.Width/2
			offsetY := rand.Intn(target.Height) - target.Height/2
			x = target.X + target.Width/2 + offsetX
			y = target.Y + target.Height/2 + offsetY
		} else {
			x = 100 + rand.Intn(600)
			y = 100 + rand.Intn(400)
		}

		jitterX := rand.Intn(20) - 10
		jitterY := rand.Intn(20) - 10
		x += jitterX
		y += jitterY

		hesitation := int64(0)
		if rand.Float64() < 0.3 {
			hesitation = int64(100 + rand.Intn(300))
		}

		clicks = append(clicks, SliderClickData{
			X:         x,
			Y:         y,
			Timestamp: int64(i)*interval + hesitation,
			Index:     i,
		})
	}

	return clicks
}

func GenerateBotLikeClickData(targets []TargetImage, duration int64) []SliderClickData {
	clicks := make([]SliderClickData, 0)

	numClicks := len(targets)
	if numClicks == 0 {
		numClicks = 3
	}

	interval := duration / int64(numClicks)

	for i := 0; i < numClicks; i++ {
		var x, y int

		if i < len(targets) {
			target := targets[i]
			x = target.X + target.Width/2
			y = target.Y + target.Height/2
		} else {
			x = 400
			y = 300
		}

		clicks = append(clicks, SliderClickData{
			X:         x,
			Y:         y,
			Timestamp: int64(i) * interval,
			Index:     i,
		})
	}

	return clicks
}
