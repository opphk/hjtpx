package service

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

type ClickVerification struct {
	Clicks       []ClickData          `json:"clicks"`
	TargetImages []TargetImage        `json:"target_images"`
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
	MultiTargetAnalysis *MultiTargetAnalysis `json:"multi_target_analysis"`
	FaultTolerance     *FaultToleranceResult `json:"fault_tolerance"`
	AnomalyScore       float64              `json:"anomaly_score"`
	MLScore            float64              `json:"ml_score"`
	OverallRiskScore   float64              `json:"overall_risk_score"`
	IsBot              bool                 `json:"is_bot"`
	Confidence         float64              `json:"confidence"`
	RiskIndicators     []string             `json:"risk_indicators"`
	AnomalyDetections  []string             `json:"anomaly_detections"`
}

type MultiTargetAnalysis struct {
	TargetCount           int      `json:"target_count"`
	ClickCount            int      `json:"click_count"`
	CorrectTargetSequence []int    `json:"correct_target_sequence"`
	MissedTargets         []int    `json:"missed_targets"`
	ExtraClicks           int      `json:"extra_clicks"`
	SequenceAccuracy      float64  `json:"sequence_accuracy"`
	TargetHitRate         float64  `json:"target_hit_rate"`
	ClickOrderCorrect     bool     `json:"click_order_correct"`
	ErrorPatterns         []string `json:"error_patterns"`
}

type FaultToleranceResult struct {
	Enabled           bool     `json:"enabled"`
	ToleranceRadius   float64  `json:"tolerance_radius"`
	NearMissClicks    int      `json:"near_miss_clicks"`
	NearMissTolerance float64  `json:"near_miss_tolerance"`
	AcceptedAsValid   bool     `json:"accepted_as_valid"`
	FallThroughCount  int      `json:"fall_through_count"`
	PartialMatchCount int      `json:"partial_match_count"`
	MissDetails       []string `json:"miss_details"`
	EdgeHitCount      int      `json:"edge_hit_count"`
	EdgeTolerance     float64  `json:"edge_tolerance"`
	SmartTolerance    bool     `json:"smart_tolerance"`
	ContextAware      bool     `json:"context_aware"`
}

type ClickPatternAnalysis struct {
	ClickCount           int                   `json:"click_count"`
	ClickIntervals       []float64             `json:"click_intervals"`
	AverageInterval      float64               `json:"average_interval"`
	IntervalVariance     float64               `json:"interval_variance"`
	IntervalStdDev       float64               `json:"interval_std_dev"`
	Regularity           float64               `json:"regularity"`
	PositionDistribution *PositionDistribution `json:"position_distribution"`
	ClickSequence        string                `json:"click_sequence"`
	SequencePattern      string                `json:"sequence_pattern"`
	ClusteringScore      float64               `json:"clustering_score"`
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
	TotalDuration    int64     `json:"total_duration"`
	AverageDuration  float64   `json:"average_duration"`
	DurationVariance float64   `json:"duration_variance"`
	ResponseTimes    []float64 `json:"response_times"`
	FirstClickDelay  int64     `json:"first_click_delay"`
	HesitationTimes  []float64 `json:"hesitation_times"`
	TimingPattern    string    `json:"timing_pattern"`
	IsRhythmic       bool      `json:"is_rhythmic"`
	ClickSpeedVariation float64 `json:"click_speed_variation"`
	IsHumanLike      bool      `json:"is_human_like"`
	IntervalCoefficientOfVariation float64 `json:"interval_coefficient_of_variation"`
	AccelerationPattern string  `json:"acceleration_pattern"`
}

type AccuracyAnalysis struct {
	CorrectClicks       int       `json:"correct_clicks"`
	TotalClicks         int       `json:"total_clicks"`
	Accuracy            float64   `json:"accuracy"`
	MissDistances       []float64 `json:"miss_distances"`
	AverageMissDistance float64   `json:"average_miss_distance"`
	TargetHits          []bool    `json:"target_hits"`
	Precision           float64   `json:"precision"`
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
			IsBot:          true,
			Confidence:     0.9,
			RiskIndicators: []string{"无点击数据"},
		}
	}

	result := &ClickAnalysisResult{
		ClickPattern:       ca.analyzeClickPattern(verification),
		TimingAnalysis:     ca.analyzeTiming(verification),
		AccuracyAnalysis:   ca.analyzeAccuracy(verification),
		MultiTargetAnalysis: ca.analyzeMultiTarget(verification),
		FaultTolerance:     ca.analyzeFaultTolerance(verification),
		RiskIndicators:    make([]string, 0),
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
		
		mean := timing.AverageDuration
		if mean > 0 {
			timing.IntervalCoefficientOfVariation = math.Sqrt(timing.DurationVariance) / mean
		}
		
		timing.ClickSpeedVariation = timing.DurationVariance / (mean * mean + 1)
	}

	if len(verification.Clicks) > 0 {
		timing.FirstClickDelay = verification.Clicks[0].Timestamp
	}

	timing.HesitationTimes = ca.calculateHesitationTimes(verification.Clicks)

	timing.TimingPattern = ca.classifyTimingPattern(timing)
	timing.IsRhythmic = ca.isTimingRhythmic(timing)
	
	timing.IsHumanLike = ca.isHumanLikeTiming(timing)
	timing.AccelerationPattern = ca.analyzeAccelerationPattern(verification.Clicks)

	return timing
}

func (ca *ClickAnalyzer) isHumanLikeTiming(timing *TimingAnalysis) bool {
	if timing == nil {
		return false
	}
	
	if timing.IntervalCoefficientOfVariation > 0.5 && timing.IntervalCoefficientOfVariation < 2.0 {
		return true
	}
	
	if timing.AverageDuration < 100 || timing.AverageDuration > 5000 {
		return false
	}
	
	return timing.ClickSpeedVariation < 3.0
}

func (ca *ClickAnalyzer) analyzeAccelerationPattern(clicks []ClickData) string {
	if len(clicks) < 3 {
		return "unknown"
	}
	
	speeds := make([]float64, len(clicks)-1)
	for i := 1; i < len(clicks); i++ {
		dx := float64(clicks[i].X - clicks[i-1].X)
		dy := float64(clicks[i].Y - clicks[i-1].Y)
		dt := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		if dt > 0 {
			distance := math.Sqrt(dx*dx + dy*dy)
			speeds[i-1] = distance / dt * 1000
		}
	}
	
	if len(speeds) < 2 {
		return "uniform"
	}
	
	accelerations := make([]float64, len(speeds)-1)
	for i := 1; i < len(speeds); i++ {
		accelerations[i-1] = speeds[i] - speeds[i-1]
	}
	
	positiveCount := 0
	negativeCount := 0
	for _, acc := range accelerations {
		if acc > 0 {
			positiveCount++
		} else {
			negativeCount++
		}
	}
	
	if positiveCount > negativeCount*2 {
		return "accelerating"
	} else if negativeCount > positiveCount*2 {
		return "decelerating"
	} else {
		return "variable"
	}
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

func (ca *ClickAnalyzer) analyzeMultiTarget(verification *ClickVerification) *MultiTargetAnalysis {
	analysis := &MultiTargetAnalysis{}

	if len(verification.Clicks) == 0 || len(verification.TargetImages) == 0 {
		return analysis
	}

	analysis.TargetCount = len(verification.TargetImages)
	analysis.ClickCount = len(verification.Clicks)

	costMatrix := ca.buildCostMatrix(verification.Clicks, verification.TargetImages)
	optimalMatching := ca.hungarianAlgorithm(costMatrix)

	hitTargets := make([]bool, len(verification.TargetImages))
	correctSequence := make([]int, 0)
	missedTargets := make([]int, 0)
	errorPatterns := make([]string, 0)

	for clickIdx, targetIdx := range optimalMatching {
		if targetIdx >= 0 && targetIdx < len(verification.TargetImages) {
			hitTargets[targetIdx] = true
			correctSequence = append(correctSequence, targetIdx)
		} else {
			errorPatterns = append(errorPatterns, fmt.Sprintf("点击(%d,%d)未匹配到目标",
				verification.Clicks[clickIdx].X, verification.Clicks[clickIdx].Y))
		}
	}

	for tIdx, hit := range hitTargets {
		if !hit {
			missedTargets = append(missedTargets, tIdx)
		}
	}

	analysis.CorrectTargetSequence = correctSequence
	analysis.MissedTargets = missedTargets
	analysis.ExtraClicks = len(verification.Clicks) - len(correctSequence)

	if analysis.TargetCount > 0 {
		analysis.TargetHitRate = float64(len(correctSequence)) / float64(analysis.TargetCount)
	}

	if len(verification.Clicks) > 0 {
		analysis.SequenceAccuracy = float64(len(correctSequence)) / float64(len(verification.Clicks))
	}

	analysis.ClickOrderCorrect = len(errorPatterns) == 0 && len(missedTargets) == 0
	analysis.ErrorPatterns = errorPatterns

	return analysis
}

func (ca *ClickAnalyzer) buildCostMatrix(clicks []ClickData, targets []TargetImage) [][]float64 {
	n := len(clicks)
	m := len(targets)
	costMatrix := make([][]float64, n)
	
	for i := 0; i < n; i++ {
		costMatrix[i] = make([]float64, m)
		for j := 0; j < m; j++ {
			targetCenterX := float64(targets[j].X) + float64(targets[j].Width)/2
			targetCenterY := float64(targets[j].Y) + float64(targets[j].Height)/2
			dx := float64(clicks[i].X) - targetCenterX
			dy := float64(clicks[i].Y) - targetCenterY
			distance := math.Sqrt(dx*dx + dy*dy)
			
			hitRadius := float64(targets[j].Width+targets[j].Height) / 4
			if distance <= hitRadius {
				costMatrix[i][j] = distance
			} else {
				costMatrix[i][j] = distance * 2
			}
		}
	}
	
	return costMatrix
}

func (ca *ClickAnalyzer) hungarianAlgorithm(costMatrix [][]float64) []int {
	n := len(costMatrix)
	if n == 0 {
		return []int{}
	}
	m := len(costMatrix[0])
	if m == 0 {
		return make([]int, n)
	}

	match := make([]int, n)
	for i := range match {
		match[i] = -1
	}

	usedTargets := make([]bool, m)
	
	for {
		dist := make([]float64, m)
		parent := make([]int, m)
		visited := make([]bool, m)
		
		for i := range dist {
			dist[i] = math.MaxFloat64
			parent[i] = -1
			visited[i] = false
		}
		
		queue := make([]int, 0)
		for i := 0; i < n; i++ {
			if match[i] < 0 {
				queue = append(queue, i)
				dist[i] = 0
			}
		}
		
		for len(queue) > 0 {
			clickIdx := queue[0]
			queue = queue[1:]
			
			for targetIdx := 0; targetIdx < m; targetIdx++ {
				cost := costMatrix[clickIdx][targetIdx]
				newDist := dist[clickIdx] + cost
				
				if newDist < dist[targetIdx] {
					dist[targetIdx] = newDist
					parent[targetIdx] = clickIdx
					if !visited[targetIdx] {
						visited[targetIdx] = true
						queue = append(queue, targetIdx)
					}
				}
			}
		}
		
		target := -1
		for t := 0; t < m; t++ {
			if !usedTargets[t] && parent[t] >= 0 {
				if target < 0 || dist[t] < dist[target] {
					target = t
				}
			}
		}
		
		if target < 0 {
			break
		}
		
		for cur := target; cur >= 0; {
			prev := parent[cur]
			if prev < 0 {
				break
			}
			nextMatch := match[prev]
			match[prev] = cur
			usedTargets[cur] = true
			cur = nextMatch
		}
	}
	
	usedTargets = make([]bool, m)
	for i := 0; i < n; i++ {
		if match[i] >= 0 && !usedTargets[match[i]] {
			usedTargets[match[i]] = true
		} else if match[i] >= 0 {
			for j := 0; j < m; j++ {
				if !usedTargets[j] {
					match[i] = j
					usedTargets[j] = true
					break
				}
			}
		}
	}
	
	return match
}

func (ca *ClickAnalyzer) analyzeFaultTolerance(verification *ClickVerification) *FaultToleranceResult {
	result := &FaultToleranceResult{
		Enabled:           true,
		ToleranceRadius:   25.0,
		NearMissTolerance: 35.0,
		MissDetails:       make([]string, 0),
		EdgeTolerance:     5.0,
		SmartTolerance:    true,
		ContextAware:      true,
	}

	if len(verification.Clicks) == 0 || len(verification.TargetImages) == 0 {
		result.Enabled = false
		return result
	}

	nearMissCount := 0
	partialMatchCount := 0
	edgeHitCount := 0

	for _, click := range verification.Clicks {
		bestMatchIdx := -1
		bestDistance := math.MaxFloat64

		for tIdx, target := range verification.TargetImages {
			targetCenterX := float64(target.X) + float64(target.Width)/2
			targetCenterY := float64(target.Y) + float64(target.Height)/2
			dx := float64(click.X) - targetCenterX
			dy := float64(click.Y) - targetCenterY
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance < bestDistance {
				bestDistance = distance
				bestMatchIdx = tIdx
			}
		}

		if bestMatchIdx >= 0 {
			target := verification.TargetImages[bestMatchIdx]
			isEdgeHit := ca.checkEdgeHit(click, target)
			if isEdgeHit {
				edgeHitCount++
				result.MissDetails = append(result.MissDetails,
					fmt.Sprintf("点击(%d,%d)命中目标边缘，偏移%.1f像素", click.X, click.Y, bestDistance))
			}
		}

		if bestDistance <= result.NearMissTolerance && bestDistance > result.ToleranceRadius {
			nearMissCount++
			result.MissDetails = append(result.MissDetails,
				fmt.Sprintf("点击(%d,%d)偏离目标%.1f像素", click.X, click.Y, bestDistance))
		} else if bestDistance > result.NearMissTolerance {
			partialMatchCount++
			result.MissDetails = append(result.MissDetails,
				fmt.Sprintf("点击(%d,%d)完全偏离所有目标，最近距离%.1f像素", click.X, click.Y, bestDistance))
		}
	}

	result.NearMissClicks = nearMissCount
	result.PartialMatchCount = partialMatchCount
	result.FallThroughCount = partialMatchCount
	result.EdgeHitCount = edgeHitCount

	maxNearMissAllowed := 1
	if len(verification.TargetImages) > 4 {
		maxNearMissAllowed = 2
	}
	result.AcceptedAsValid = nearMissCount <= maxNearMissAllowed && partialMatchCount == 0

	return result
}

func (ca *ClickAnalyzer) checkEdgeHit(click ClickData, target TargetImage) bool {
	targetLeft := float64(target.X)
	targetRight := float64(target.X) + float64(target.Width)
	targetTop := float64(target.Y)
	targetBottom := float64(target.Y) + float64(target.Height)

	clickX := float64(click.X)
	clickY := float64(click.Y)

	distToLeft := math.Abs(clickX - targetLeft)
	distToRight := math.Abs(clickX - targetRight)
	distToTop := math.Abs(clickY - targetTop)
	distToBottom := math.Abs(clickY - targetBottom)

	minDist := math.Min(math.Min(distToLeft, distToRight), math.Min(distToTop, distToBottom))

	return minDist <= ca.calculateEdgeTolerance(target)
}

func (ca *ClickAnalyzer) calculateEdgeTolerance(target TargetImage) float64 {
	avgSize := float64(target.Width+target.Height) / 2
	return avgSize * 0.2
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

type ClickTimingAnalyzer struct{}

func NewClickTimingAnalyzer() *ClickTimingAnalyzer {
	return &ClickTimingAnalyzer{}
}

type TimingFeatures struct {
	Intervals         []float64 `json:"intervals"`
	MeanInterval      float64   `json:"mean_interval"`
	StdDevInterval    float64   `json:"std_dev_interval"`
	CvInterval        float64   `json:"cv_interval"`
	IsRhythmic        bool      `json:"is_rhythmic"`
	RhythmScore       float64   `json:"rhythm_score"`
	TimingPattern     string    `json:"timing_pattern"`
	FirstClickDelay   float64   `json:"first_click_delay"`
	LastClickDelay    float64   `json:"last_click_delay"`
	AccelerationTrend float64   `json:"acceleration_trend"`
	VarianceTrend     float64   `json:"variance_trend"`
	AnomalyIntervals  []int     `json:"anomaly_intervals"`
	ConsistencyScore  float64   `json:"consistency_score"`
}

func (cta *ClickTimingAnalyzer) AnalyzeTiming(clicks []ClickData) *TimingFeatures {
	features := &TimingFeatures{}

	if len(clicks) < 2 {
		return features
	}

	intervals := cta.extractIntervals(clicks)
	features.Intervals = intervals

	if len(intervals) > 0 {
		features.MeanInterval = cta.mean(intervals)
		features.StdDevInterval = cta.stdDev(intervals)

		if features.MeanInterval > 0 {
			features.CvInterval = features.StdDevInterval / features.MeanInterval
		}

		features.IsRhythmic = features.CvInterval < 0.15
		features.RhythmScore = 1.0 - math.Min(features.CvInterval, 1.0)
	}

	if len(clicks) > 0 {
		features.FirstClickDelay = float64(clicks[0].Timestamp)
		features.LastClickDelay = float64(clicks[len(clicks)-1].Timestamp)
	}

	features.AccelerationTrend = cta.calculateAccelerationTrend(intervals)
	features.VarianceTrend = cta.calculateVarianceTrend(intervals)
	features.AnomalyIntervals = cta.detectAnomalyIntervals(intervals)
	features.ConsistencyScore = cta.calculateConsistencyScore(intervals)

	features.TimingPattern = cta.classifyTimingPattern(features)

	return features
}

func (cta *ClickTimingAnalyzer) extractIntervals(clicks []ClickData) []float64 {
	intervals := make([]float64, 0)
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		intervals = append(intervals, interval)
	}
	return intervals
}

func (cta *ClickTimingAnalyzer) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (cta *ClickTimingAnalyzer) stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := cta.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return math.Sqrt(sum / float64(len(values)))
}

func (cta *ClickTimingAnalyzer) calculateAccelerationTrend(intervals []float64) float64 {
	if len(intervals) < 3 {
		return 0
	}

	accelerations := make([]float64, 0)
	for i := 1; i < len(intervals); i++ {
		accel := intervals[i] - intervals[i-1]
		accelerations = append(accelerations, accel)
	}

	meanAccel := cta.mean(accelerations)
	return meanAccel / cta.mean(intervals)
}

func (cta *ClickTimingAnalyzer) calculateVarianceTrend(intervals []float64) float64 {
	if len(intervals) < 6 {
		return 0
	}

	firstHalf := intervals[:len(intervals)/2]
	secondHalf := intervals[len(intervals)/2:]

	firstVariance := cta.stdDev(firstHalf)
	secondVariance := cta.stdDev(secondHalf)

	if firstVariance == 0 {
		return 0
	}

	return (secondVariance - firstVariance) / firstVariance
}

func (cta *ClickTimingAnalyzer) detectAnomalyIntervals(intervals []float64) []int {
	if len(intervals) == 0 {
		return []int{}
	}

	mean := cta.mean(intervals)
	stdDev := cta.stdDev(intervals)

	anomalies := make([]int, 0)
	for i, interval := range intervals {
		if math.Abs(interval-mean) > 2*stdDev {
			anomalies = append(anomalies, i)
		}
	}

	return anomalies
}

func (cta *ClickTimingAnalyzer) calculateConsistencyScore(intervals []float64) float64 {
	if len(intervals) < 2 {
		return 0
	}

	mean := cta.mean(intervals)
	if mean == 0 {
		return 0
	}

	consistentCount := 0
	for _, interval := range intervals {
		if math.Abs(interval-mean)/mean < 0.3 {
			consistentCount++
		}
	}

	return float64(consistentCount) / float64(len(intervals))
}

func (cta *ClickTimingAnalyzer) classifyTimingPattern(features *TimingFeatures) string {
	if features.MeanInterval < 100 {
		return "extremely_fast"
	}
	if features.MeanInterval < 300 {
		return "very_fast"
	}
	if features.MeanInterval < 600 {
		return "fast"
	}
	if features.MeanInterval < 1500 {
		return "normal"
	}
	if features.MeanInterval < 3000 {
		return "slow"
	}
	return "very_slow"
}

type ClickPressureAnalyzer struct{}

func NewClickPressureAnalyzer() *ClickPressureAnalyzer {
	return &ClickPressureAnalyzer{}
}

type PressureFeatures struct {
	HasPressureData     bool      `json:"has_pressure_data"`
	Pressures           []float64 `json:"pressures"`
	MeanPressure        float64   `json:"mean_pressure"`
	PressureVariance    float64   `json:"pressure_variance"`
	PressureConsistency float64   `json:"pressure_consistency"`
	IsBotLike           bool      `json:"is_bot_like"`
}

type ClickDataWithPressure struct {
	X         int
	Y         int
	Timestamp int64
	Pressure  float64
}

func (cpa *ClickPressureAnalyzer) AnalyzePressure(clickEvents []map[string]interface{}) *PressureFeatures {
	features := &PressureFeatures{}

	if len(clickEvents) == 0 {
		return features
	}

	pressures := make([]float64, 0)
	for _, event := range clickEvents {
		if pressure, ok := event["pressure"].(float64); ok {
			pressures = append(pressures, pressure)
		} else if force, ok := event["force"].(float64); ok {
			pressures = append(pressures, force)
		}
	}

	if len(pressures) == 0 {
		features.HasPressureData = false
		return features
	}

	features.HasPressureData = true
	features.Pressures = pressures
	features.MeanPressure = cpa.mean(pressures)
	features.PressureVariance = cpa.variance(pressures)
	features.PressureConsistency = 1.0 - math.Min(math.Sqrt(features.PressureVariance)/features.MeanPressure, 1.0)

	features.IsBotLike = features.PressureConsistency > 0.95 && features.MeanPressure > 0.8

	return features
}

func (cpa *ClickPressureAnalyzer) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (cpa *ClickPressureAnalyzer) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := cpa.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

type AnomalyClickPattern struct {
	Name        string
	Description string
	Detector    func(*ClickAnalysisResult) bool
	Weight      float64
}

type AnomalyClickDetector struct {
	patterns []AnomalyClickPattern
}

func NewAnomalyClickDetector() *AnomalyClickDetector {
	return &AnomalyClickDetector{
		patterns: []AnomalyClickPattern{
			{
				Name:        "perfect_precision",
				Description: "异常精准的点击，无任何偏差",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.AccuracyAnalysis == nil {
						return false
					}
					return result.AccuracyAnalysis.Accuracy == 1.0 &&
						result.AccuracyAnalysis.AverageMissDistance < 5 &&
						result.AccuracyAnalysis.TotalClicks > 2
				},
				Weight: 0.35,
			},
			{
				Name:        "mechanical_timing",
				Description: "机械般规律的点击间隔",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.IsRhythmic &&
						result.TimingAnalysis.DurationVariance < 100
				},
				Weight: 0.3,
			},
			{
				Name:        "instant_response",
				Description: "无犹豫的即时响应",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.FirstClickDelay < 100 &&
						len(result.TimingAnalysis.HesitationTimes) == 0 &&
						result.TimingAnalysis.TotalDuration > 1000
				},
				Weight: 0.25,
			},
			{
				Name:        "uniform_position",
				Description: "均匀分布的点击位置",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.ClickPattern == nil ||
						result.ClickPattern.PositionDistribution == nil {
						return false
					}
					dist := result.ClickPattern.PositionDistribution
					return dist.XVariance < 10 && dist.YVariance < 10
				},
				Weight: 0.2,
			},
			{
				Name:        "linear_trajectory",
				Description: "线性点击轨迹",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.ClickPattern == nil {
						return false
					}
					return result.ClickPattern.SequencePattern == "linear" &&
						len(result.ClickPattern.ClickIntervals) > 3
				},
				Weight: 0.2,
			},
			{
				Name:        "no_hesitation",
				Description: "整个过程中无犹豫",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return len(result.TimingAnalysis.HesitationTimes) == 0 &&
						result.TimingAnalysis.TotalDuration > 2000
				},
				Weight: 0.15,
			},
			{
				Name:        "constant_interval",
				Description: "恒定的点击间隔",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.ClickPattern == nil || len(result.ClickPattern.ClickIntervals) < 3 {
						return false
					}
					intervals := result.ClickPattern.ClickIntervals
					mean := 0.0
					for _, i := range intervals {
						mean += i
					}
					mean /= float64(len(intervals))

					if mean == 0 {
						return false
					}

					for _, interval := range intervals {
						if math.Abs(interval-mean)/mean > 0.05 {
							return false
						}
					}
					return true
				},
				Weight: 0.25,
			},
			{
				Name:        "high_accuracy",
				Description: "异常高的准确率",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.AccuracyAnalysis == nil {
						return false
					}
					return result.AccuracyAnalysis.Accuracy > 0.95 &&
						result.AccuracyAnalysis.TotalClicks > 4
				},
				Weight: 0.3,
			},
		},
	}
}

func (acd *AnomalyClickDetector) DetectAnomalies(result *ClickAnalysisResult) (float64, []string) {
	totalScore := 0.0
	detectedPatterns := make([]string, 0)

	for _, pattern := range acd.patterns {
		if pattern.Detector(result) {
			totalScore += pattern.Weight
			detectedPatterns = append(detectedPatterns,
				fmt.Sprintf("%s: %s", pattern.Name, pattern.Description))
		}
	}

	return totalScore, detectedPatterns
}

type AdvancedClickAnalyzer struct {
	timingAnalyzer   *ClickTimingAnalyzer
	pressureAnalyzer *ClickPressureAnalyzer
	anomalyDetector  *AnomalyClickDetector
}

func NewAdvancedClickAnalyzer() *AdvancedClickAnalyzer {
	return &AdvancedClickAnalyzer{
		timingAnalyzer:   NewClickTimingAnalyzer(),
		pressureAnalyzer: NewClickPressureAnalyzer(),
		anomalyDetector:  NewAnomalyClickDetector(),
	}
}

type AdvancedClickResult struct {
	BasicResult      *ClickAnalysisResult
	TimingFeatures   *TimingFeatures
	PressureFeatures *PressureFeatures
	AnomalyPatterns  []string
	AnomalyScore     float64
	BotScore         float64
}

func (aca *AdvancedClickAnalyzer) AnalyzeAdvanced(verification *ClickVerification) *AdvancedClickResult {
	result := &AdvancedClickResult{
		BasicResult: NewClickAnalyzer().AnalyzeClickVerification(verification),
	}

	if len(verification.Clicks) >= 2 {
		result.TimingFeatures = aca.timingAnalyzer.AnalyzeTiming(verification.Clicks)
	}

	if verification.Clicks != nil && len(verification.Clicks) > 0 {
		clickEvents := make([]map[string]interface{}, len(verification.Clicks))
		for i, click := range verification.Clicks {
			clickEvents[i] = map[string]interface{}{
				"x":         click.X,
				"y":         click.Y,
				"timestamp": click.Timestamp,
			}
		}
		result.PressureFeatures = aca.pressureAnalyzer.AnalyzePressure(clickEvents)
	}

	anomalyScore, anomalyPatterns := aca.anomalyDetector.DetectAnomalies(result.BasicResult)
	result.AnomalyPatterns = anomalyPatterns
	result.AnomalyScore = anomalyScore

	result.BotScore = aca.calculateBotScore(result)

	return result
}

func (aca *AdvancedClickAnalyzer) calculateBotScore(result *AdvancedClickResult) float64 {
	botScore := 0.0

	if result.BasicResult != nil {
		botScore += result.BasicResult.MLScore * 0.25
		botScore += result.BasicResult.AnomalyScore * 0.25
	}

	if result.TimingFeatures != nil {
		if result.TimingFeatures.IsRhythmic {
			botScore += 0.15
		}
		if result.TimingFeatures.RhythmScore > 0.9 {
			botScore += 0.1
		}
		if result.TimingFeatures.ConsistencyScore > 0.9 {
			botScore += 0.1
		}
	}

	if result.PressureFeatures != nil && result.PressureFeatures.HasPressureData {
		if result.PressureFeatures.IsBotLike {
			botScore += 0.15
		}
	}

	botScore += result.AnomalyScore * 0.25

	return math.Min(botScore, 1.0)
}

func (ca *ClickAnalyzer) AnalyzeWithAdvancedFeatures(verification *ClickVerification) *ClickAnalysisResult {
	result := ca.AnalyzeClickVerification(verification)

	advancedAnalyzer := NewAdvancedClickAnalyzer()
	advancedResult := advancedAnalyzer.AnalyzeAdvanced(verification)

	if advancedResult.TimingFeatures != nil {
		result.RiskIndicators = append(result.RiskIndicators,
			fmt.Sprintf("时序模式: %s", advancedResult.TimingFeatures.TimingPattern))
		result.RiskIndicators = append(result.RiskIndicators,
			fmt.Sprintf("节奏性分数: %.2f", advancedResult.TimingFeatures.RhythmScore))

		if advancedResult.TimingFeatures.IsRhythmic {
			result.AnomalyDetections = append(result.AnomalyDetections,
				"检测到机械节奏模式")
		}
	}

	if advancedResult.PressureFeatures != nil && advancedResult.PressureFeatures.HasPressureData {
		result.RiskIndicators = append(result.RiskIndicators,
			fmt.Sprintf("压力一致性: %.2f", advancedResult.PressureFeatures.PressureConsistency))
	}

	result.RiskIndicators = append(result.RiskIndicators, advancedResult.AnomalyPatterns...)

	return result
}
