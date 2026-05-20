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
	
	// 新增位置分布字段
	KDEPeaks          []KDEPeak      `json:"kde_peaks"`
	ClusterCount      int            `json:"cluster_count"`
	ClusterCenters    []ClusterCenter `json:"cluster_centers"`
	ClusterAssignments []int          `json:"cluster_assignments"`
	SpatialEntropy    float64        `json:"spatial_entropy"`
	GridDistribution  []int          `json:"grid_distribution"`
	OutlierCount      int            `json:"outlier_count"`
	OutlierScore      float64        `json:"outlier_score"`
	ConvexHullArea    float64        `json:"convex_hull_area"`
	DispersionIndex   float64        `json:"dispersion_index"`
}

type KDEPeak struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Density  float64 `json:"density"`
	IsGlobal bool    `json:"is_global"`
}

type ClusterCenter struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	PointCount int    `json:"point_count"`
	Radius    float64 `json:"radius"`
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
	
	// 新增时序分析字段
	IntervalTrend         float64   `json:"interval_trend"`
	JerkPattern           string    `json:"jerk_pattern"`
	TimePressureIndicator float64   `json:"time_pressure_indicator"`
	AttentionDecay        float64   `json:"attention_decay"`
	RhythmConsistency     float64   `json:"rhythm_consistency"`
	ComplexityScore       float64   `json:"complexity_score"`
	PeriodicityScore      float64   `json:"periodicity_score"`
	TransientResponse     []float64 `json:"transient_response"`
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

	// 新增位置分布分析功能
	distribution.KDEPeaks = ca.calculateKDEPeaks(xValues, yValues)
	distribution.SpatialEntropy = ca.calculateSpatialEntropy(xValues, yValues)
	distribution.GridDistribution = ca.calculateGridDistribution(xValues, yValues, 5, 5)
	
	if len(clicks) >= 2 {
		clusters, assignments := ca.performKMeansClustering(xValues, yValues, 3)
		distribution.ClusterCenters = clusters
		distribution.ClusterCount = len(clusters)
		distribution.ClusterAssignments = assignments
		distribution.OutlierCount, distribution.OutlierScore = ca.detectOutliers(xValues, yValues, clusters, assignments)
		distribution.ConvexHullArea = ca.calculateConvexHullArea(xValues, yValues)
		distribution.DispersionIndex = ca.calculateDispersionIndex(xValues, yValues)
	}

	return distribution
}

func (ca *ClickAnalyzer) calculateKDEPeaks(xValues, yValues []float64) []KDEPeak {
	if len(xValues) < 3 {
		return []KDEPeak{}
	}

	// 使用高斯核密度估计
	bandwidth := ca.estimateBandwidth(xValues, yValues)
	peaks := []KDEPeak{}
	
	maxDensity := 0.0

	// 在网格上计算密度
	gridSize := 20
	minX, maxXRange := ca.min(xValues), ca.max(xValues)
	minY, maxYRange := ca.min(yValues), ca.max(yValues)
	
	rangeX := maxXRange - minX
	rangeY := maxYRange - minY
	if rangeX < 1 {
		rangeX = 1
	}
	if rangeY < 1 {
		rangeY = 1
	}

	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			gx := minX + rangeX*float64(i)/float64(gridSize-1)
			gy := minY + rangeY*float64(j)/float64(gridSize-1)
			
			density := 0.0
			for k := 0; k < len(xValues); k++ {
				dx := xValues[k] - gx
				dy := yValues[k] - gy
				density += math.Exp(-(dx*dx + dy*dy) / (2 * bandwidth * bandwidth))
			}
			density /= float64(len(xValues)) * bandwidth * math.Sqrt(2*math.Pi)
			
			if density > maxDensity {
				maxDensity = density
			}
			
			// 检测局部峰值
			isPeak := true
			for di := -1; di <= 1 && isPeak; di++ {
				for dj := -1; dj <= 1 && isPeak; dj++ {
					if di == 0 && dj == 0 {
						continue
					}
					ni, nj := i+di, j+dj
					if ni >= 0 && ni < gridSize && nj >= 0 && nj < gridSize {
						nx := minX + rangeX*float64(ni)/float64(gridSize-1)
						ny := minY + rangeY*float64(nj)/float64(gridSize-1)
						nDensity := 0.0
						for k := 0; k < len(xValues); k++ {
							dx := xValues[k] - nx
							dy := yValues[k] - ny
							nDensity += math.Exp(-(dx*dx + dy*dy) / (2 * bandwidth * bandwidth))
						}
						nDensity /= float64(len(xValues)) * bandwidth * math.Sqrt(2*math.Pi)
						if nDensity > density {
							isPeak = false
						}
					}
				}
			}
			
			if isPeak && density > 0.001 {
				peaks = append(peaks, KDEPeak{X: gx, Y: gy, Density: density, IsGlobal: false})
			}
		}
	}
	
	// 标记全局峰值
	for i := range peaks {
		if peaks[i].Density >= maxDensity*0.9 {
			peaks[i].IsGlobal = true
		}
	}
	
	return peaks
}

func (ca *ClickAnalyzer) estimateBandwidth(xValues, yValues []float64) float64 {
	n := len(xValues)
	if n < 2 {
		return 1.0
	}
	
	// 使用 Silverman 法则
	stdX := ca.stdDev(xValues)
	stdY := ca.stdDev(yValues)
	std := math.Max(stdX, stdY)
	
	return 1.06 * std * math.Pow(float64(n), -0.2)
}

func (ca *ClickAnalyzer) calculateSpatialEntropy(xValues, yValues []float64) float64 {
	if len(xValues) < 2 {
		return 0
	}
	
	// 使用二维直方图计算空间熵
	bins := 5
	grid := make([][]int, bins)
	for i := range grid {
		grid[i] = make([]int, bins)
	}
	
	minX, maxX := ca.min(xValues), ca.max(xValues)
	minY, maxY := ca.min(yValues), ca.max(yValues)
	
	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}
	
	for i := 0; i < len(xValues); i++ {
		binX := int(((xValues[i] - minX) / rangeX) * float64(bins))
		binY := int(((yValues[i] - minY) / rangeY) * float64(bins))
		if binX >= bins {
			binX = bins - 1
		}
		if binY >= bins {
			binY = bins - 1
		}
		if binX < 0 {
			binX = 0
		}
		if binY < 0 {
			binY = 0
		}
		grid[binX][binY]++
	}
	
	entropy := 0.0
	total := float64(len(xValues))
	
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			if grid[i][j] > 0 {
				p := float64(grid[i][j]) / total
				entropy -= p * math.Log2(p)
			}
		}
	}
	
	// 归一化到 [0, 1]
	maxEntropy := math.Log2(float64(bins * bins))
	if maxEntropy > 0 {
		entropy /= maxEntropy
	}
	
	return entropy
}

func (ca *ClickAnalyzer) calculateGridDistribution(xValues, yValues []float64, gridX, gridY int) []int {
	if len(xValues) == 0 {
		return []int{}
	}
	
	distribution := make([]int, gridX*gridY)
	
	minX, maxX := ca.min(xValues), ca.max(xValues)
	minY, maxY := ca.min(yValues), ca.max(yValues)
	
	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}
	
	for i := 0; i < len(xValues); i++ {
		binX := int(((xValues[i] - minX) / rangeX) * float64(gridX))
		binY := int(((yValues[i] - minY) / rangeY) * float64(gridY))
		if binX >= gridX {
			binX = gridX - 1
		}
		if binY >= gridY {
			binY = gridY - 1
		}
		if binX < 0 {
			binX = 0
		}
		if binY < 0 {
			binY = 0
		}
		distribution[binX*gridY+binY]++
	}
	
	return distribution
}

func (ca *ClickAnalyzer) performKMeansClustering(xValues, yValues []float64, k int) ([]ClusterCenter, []int) {
	n := len(xValues)
	if n < k || k < 1 {
		return []ClusterCenter{}, []int{}
	}
	
	// 随机初始化聚类中心
	centers := make([]ClusterCenter, k)
	for i := 0; i < k; i++ {
		idx := rand.Intn(n)
		centers[i] = ClusterCenter{X: xValues[idx], Y: yValues[idx]}
	}
	
	assignments := make([]int, n)
	changed := true
	iterations := 0
	
	for changed && iterations < 100 {
		changed = false
		iterations++
		
		// 分配点到最近的聚类中心
		for i := 0; i < n; i++ {
			minDist := math.MaxFloat64
			minIdx := 0
			for j := 0; j < k; j++ {
				dx := xValues[i] - centers[j].X
				dy := yValues[i] - centers[j].Y
				dist := dx*dx + dy*dy
				if dist < minDist {
					minDist = dist
					minIdx = j
				}
			}
			if assignments[i] != minIdx {
				assignments[i] = minIdx
				changed = true
			}
		}
		
		// 更新聚类中心
		for j := 0; j < k; j++ {
			sumX, sumY, count := 0.0, 0.0, 0
			for i := 0; i < n; i++ {
				if assignments[i] == j {
					sumX += xValues[i]
					sumY += yValues[i]
					count++
				}
			}
			if count > 0 {
				centers[j].X = sumX / float64(count)
				centers[j].Y = sumY / float64(count)
				centers[j].PointCount = count
			}
		}
	}
	
	// 计算每个聚类的半径
	for j := 0; j < k; j++ {
		maxDist := 0.0
		for i := 0; i < n; i++ {
			if assignments[i] == j {
				dx := xValues[i] - centers[j].X
				dy := yValues[i] - centers[j].Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > maxDist {
					maxDist = dist
				}
			}
		}
		centers[j].Radius = maxDist
	}
	
	// 移除空聚类
	validCenters := []ClusterCenter{}
	for _, c := range centers {
		if c.PointCount > 0 {
			validCenters = append(validCenters, c)
		}
	}
	
	return validCenters, assignments
}

func (ca *ClickAnalyzer) detectOutliers(xValues, yValues []float64, clusters []ClusterCenter, assignments []int) (int, float64) {
	if len(clusters) == 0 || len(assignments) != len(xValues) {
		return 0, 0
	}
	
	outlierCount := 0
	totalDistance := 0.0
	
	for i := 0; i < len(xValues); i++ {
		clusterIdx := assignments[i]
		if clusterIdx >= len(clusters) {
			outlierCount++
			continue
		}
		
		dx := xValues[i] - clusters[clusterIdx].X
		dy := yValues[i] - clusters[clusterIdx].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		
		totalDistance += distance
		
		// 如果距离超过聚类半径的2倍，则视为离群点
		if distance > clusters[clusterIdx].Radius*2 {
			outlierCount++
		}
	}
	
	avgDistance := totalDistance / float64(len(xValues))
	outlierScore := float64(outlierCount) / float64(len(xValues)) * (1 + avgDistance/100)
	
	return outlierCount, math.Min(outlierScore, 1.0)
}

func (ca *ClickAnalyzer) calculateConvexHullArea(xValues, yValues []float64) float64 {
	if len(xValues) < 3 {
		return 0
	}
	
	// 使用 Andrew's monotone chain algorithm 计算凸包
	points := make([][2]float64, len(xValues))
	for i := 0; i < len(xValues); i++ {
		points[i] = [2]float64{xValues[i], yValues[i]}
	}
	
	// 按 x 坐标排序，然后按 y 坐标排序
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if points[j][0] < points[i][0] || (points[j][0] == points[i][0] && points[j][1] < points[i][1]) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
	
	// 构建下凸包和上凸包
	lower := make([][2]float64, 0)
	for _, p := range points {
		for len(lower) >= 2 {
			l := len(lower)
			if (lower[l-1][0]-lower[l-2][0])*(p[1]-lower[l-2][1]) <= (lower[l-1][1]-lower[l-2][1])*(p[0]-lower[l-2][0]) {
				lower = lower[:l-1]
			} else {
				break
			}
		}
		lower = append(lower, p)
	}
	
	upper := make([][2]float64, 0)
	for i := len(points) - 1; i >= 0; i-- {
		p := points[i]
		for len(upper) >= 2 {
			l := len(upper)
			if (upper[l-1][0]-upper[l-2][0])*(p[1]-upper[l-2][1]) <= (upper[l-1][1]-upper[l-2][1])*(p[0]-upper[l-2][0]) {
				upper = upper[:l-1]
			} else {
				break
			}
		}
		upper = append(upper, p)
	}
	
	// 合并凸包（去掉重复点）
	hull := append(lower[:len(lower)-1], upper[:len(upper)-1]...)
	
	// 计算面积
	if len(hull) < 3 {
		return 0
	}
	
	area := 0.0
	for i := 0; i < len(hull); i++ {
		j := (i + 1) % len(hull)
		area += hull[i][0] * hull[j][1]
		area -= hull[j][0] * hull[i][1]
	}
	
	return math.Abs(area) / 2.0
}

func (ca *ClickAnalyzer) calculateDispersionIndex(xValues, yValues []float64) float64 {
	if len(xValues) < 2 {
		return 0
	}
	
	// 计算平均距离和标准差
	meanX := ca.mean(xValues)
	meanY := ca.mean(yValues)
	
	avgDistance := 0.0
	for i := 0; i < len(xValues); i++ {
		dx := xValues[i] - meanX
		dy := yValues[i] - meanY
		avgDistance += math.Sqrt(dx*dx + dy*dy)
	}
	avgDistance /= float64(len(xValues))
	
	if avgDistance == 0 {
		return 0
	}
	
	// 计算距离的标准差
	stdDistance := 0.0
	for i := 0; i < len(xValues); i++ {
		dx := xValues[i] - meanX
		dy := yValues[i] - meanY
		distance := math.Sqrt(dx*dx + dy*dy)
		stdDistance += (distance - avgDistance) * (distance - avgDistance)
	}
	stdDistance = math.Sqrt(stdDistance / float64(len(xValues)))
	
	return stdDistance / avgDistance
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

	// 新增时序分析增强功能
	timing.IntervalTrend = ca.calculateIntervalTrend(timing.ResponseTimes)
	timing.JerkPattern = ca.classifyJerkPattern(timing.ResponseTimes)
	timing.TimePressureIndicator = ca.calculateTimePressureIndicator(timing)
	timing.AttentionDecay = ca.calculateAttentionDecay(timing.ResponseTimes)
	timing.RhythmConsistency = ca.calculateRhythmConsistency(timing.ResponseTimes)
	timing.ComplexityScore = ca.calculateComplexityScore(timing.ResponseTimes)
	timing.PeriodicityScore = ca.detectPeriodicity(timing.ResponseTimes)
	timing.TransientResponse = ca.extractTransientResponse(timing.ResponseTimes)

	return timing
}

func (ca *ClickAnalyzer) calculateIntervalTrend(intervals []float64) float64 {
	if len(intervals) < 2 {
		return 0
	}
	
	// 使用线性回归计算趋势
	n := float64(len(intervals))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	
	for i, interval := range intervals {
		x := float64(i)
		y := interval
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}
	
	return (n*sumXY - sumX*sumY) / denominator
}

func (ca *ClickAnalyzer) classifyJerkPattern(intervals []float64) string {
	if len(intervals) < 3 {
		return "unknown"
	}
	
	// 计算加速度变化率（加加速度/Jerk）
	jerkValues := make([]float64, len(intervals)-2)
	for i := 1; i < len(intervals)-1; i++ {
		prevAccel := intervals[i] - intervals[i-1]
		currAccel := intervals[i+1] - intervals[i]
		jerkValues[i-1] = currAccel - prevAccel
	}
	
	meanJerk := ca.mean(jerkValues)
	stdJerk := math.Sqrt(ca.variance(jerkValues))
	
	// 根据加加速度模式分类
	if stdJerk < 20 && math.Abs(meanJerk) < 10 {
		return "smooth"
	} else if stdJerk > 100 {
		return "jerky"
	} else if meanJerk > 20 {
		return "increasing"
	} else if meanJerk < -20 {
		return "decreasing"
	}
	return "variable"
}

func (ca *ClickAnalyzer) calculateTimePressureIndicator(timing *TimingAnalysis) float64 {
	if timing.TotalDuration == 0 || timing.AverageDuration == 0 {
		return 0
	}
	
	basePressure := 1.0 - float64(timing.TotalDuration)/5000.0
	rhythmPressure := 0.0
	if timing.IsRhythmic {
		rhythmPressure = 0.3
	}
	
	variabilityPressure := 0.0
	if timing.IntervalCoefficientOfVariation < 0.3 {
		variabilityPressure = 0.2
	}
	
	return math.Min(basePressure+rhythmPressure+variabilityPressure, 1.0)
}

func (ca *ClickAnalyzer) calculateAttentionDecay(intervals []float64) float64 {
	if len(intervals) < 3 {
		return 0
	}
	
	firstThird := intervals[:len(intervals)/3]
	lastThird := intervals[len(intervals)*2/3:]
	
	if len(firstThird) == 0 || len(lastThird) == 0 {
		return 0
	}
	
	firstMean := ca.mean(firstThird)
	lastMean := ca.mean(lastThird)
	
	if firstMean == 0 {
		return 0
	}
	
	decay := (lastMean - firstMean) / firstMean
	return math.Max(-1.0, math.Min(1.0, decay))
}

func (ca *ClickAnalyzer) calculateRhythmConsistency(intervals []float64) float64 {
	if len(intervals) < 2 {
		return 0
	}
	
	mean := ca.mean(intervals)
	if mean == 0 {
		return 0
	}
	
	totalDeviation := 0.0
	for _, interval := range intervals {
		totalDeviation += math.Abs(interval - mean)
	}
	
	avgDeviation := totalDeviation / float64(len(intervals))
	return 1.0 - math.Min(avgDeviation/mean, 1.0)
}

func (ca *ClickAnalyzer) calculateComplexityScore(intervals []float64) float64 {
	if len(intervals) < 2 {
		return 0
	}
	
	// 使用样本熵来衡量复杂度
	return ca.calculateSampleEntropy(intervals)
}

func (ca *ClickAnalyzer) calculateSampleEntropy(intervals []float64) float64 {
	if len(intervals) < 4 {
		return 0
	}
	
	n := len(intervals)
	m := 2
	r := 0.2 * ca.stdDev(intervals)
	
	if r == 0 {
		return 0
	}
	
	// 计算模板匹配数
	counts := make([]int, 2)
	for i := 0; i <= n-m; i++ {
		for j := 0; j <= n-m; j++ {
			if i != j {
				dist := 0.0
				for k := 0; k < m; k++ {
					dist = math.Max(dist, math.Abs(intervals[i+k]-intervals[j+k]))
				}
				if dist <= r {
					counts[0]++
				}
			}
		}
	}
	
	for i := 0; i <= n-m-1; i++ {
		for j := 0; j <= n-m-1; j++ {
			if i != j {
				dist := 0.0
				for k := 0; k < m+1; k++ {
					dist = math.Max(dist, math.Abs(intervals[i+k]-intervals[j+k]))
				}
				if dist <= r {
					counts[1]++
				}
			}
		}
	}
	
	if counts[0] == 0 || counts[1] == 0 {
		return 0
	}
	
	return -math.Log(float64(counts[1]) / float64(counts[0]))
}

func (ca *ClickAnalyzer) detectPeriodicity(intervals []float64) float64 {
	if len(intervals) < 4 {
		return 0
	}
	
	// 使用自相关检测周期性
	n := len(intervals)
	maxLag := n / 2
	if maxLag < 2 {
		return 0
	}
	
	maxCorrelation := 0.0
	for lag := 1; lag <= maxLag; lag++ {
		sumXY, sumX, sumY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0
		count := 0
		for i := 0; i < n-lag; i++ {
			x := intervals[i]
			y := intervals[i+lag]
			sumXY += x * y
			sumX += x
			sumY += y
			sumX2 += x * x
			sumY2 += y * y
			count++
		}
		
		if count == 0 {
			continue
		}
		
		fCount := float64(count)
		denominator := math.Sqrt((fCount*sumX2-sumX*sumX) * (fCount*sumY2-sumY*sumY))
		if denominator == 0 {
			continue
		}
		
		correlation := (fCount*sumXY - sumX*sumY) / denominator
		if correlation > maxCorrelation {
			maxCorrelation = correlation
		}
	}
	
	return maxCorrelation
}

func (ca *ClickAnalyzer) extractTransientResponse(intervals []float64) []float64 {
	if len(intervals) < 2 {
		return []float64{}
	}
	
	// 提取瞬态响应特征：第一个间隔、最后一个间隔、最大间隔、最小间隔
	transient := make([]float64, 4)
	transient[0] = intervals[0]
	transient[1] = intervals[len(intervals)-1]
	transient[2] = ca.max(intervals)
	transient[3] = ca.min(intervals)
	
	return transient
}

func (ca *ClickAnalyzer) stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	return math.Sqrt(ca.variance(values))
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
		// 使用两个数组分别存储 clicks 和 targets 的距离
		distClick := make([]float64, n)
		distTarget := make([]float64, m)
		parent := make([]int, m)
		visited := make([]bool, m)
		
		for i := range distClick {
			distClick[i] = math.MaxFloat64
		}
		for i := range distTarget {
			distTarget[i] = math.MaxFloat64
			parent[i] = -1
			visited[i] = false
		}
		
		queue := make([]int, 0)
		for i := 0; i < n; i++ {
			if match[i] < 0 {
				queue = append(queue, i)
				distClick[i] = 0
			}
		}
		
		for len(queue) > 0 {
			clickIdx := queue[0]
			queue = queue[1:]
			
			for targetIdx := 0; targetIdx < m; targetIdx++ {
				cost := costMatrix[clickIdx][targetIdx]
				newDist := distClick[clickIdx] + cost
				
				if newDist < distTarget[targetIdx] {
					distTarget[targetIdx] = newDist
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
				if target < 0 || distTarget[t] < distTarget[target] {
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

type EnhancedPositionDetector struct {
	toleranceRadius    float64
	smartTolerance     bool
	adaptiveThreshold  bool
	edgeTolerance      float64
	positionWeight     float64
	confidenceThreshold float64
}

func NewEnhancedPositionDetector() *EnhancedPositionDetector {
	return &EnhancedPositionDetector{
		toleranceRadius:    25.0,
		smartTolerance:     true,
		adaptiveThreshold:  true,
		edgeTolerance:      5.0,
		positionWeight:     0.6,
		confidenceThreshold: 0.75,
	}
}

type PositionDetectionResult struct {
	IsHit          bool
	TargetIndex    int
	Distance       float64
	OffsetX        float64
	OffsetY        float64
	Confidence     float64
	HitType        string
	AdjustedRadius float64
	NearMiss       bool
	EdgeHit        bool
	ErrorMargin    float64
}

func (detector *EnhancedPositionDetector) DetectPosition(clickX, clickY int, targets []TargetImage) *PositionDetectionResult {
	result := &PositionDetectionResult{}

	if len(targets) == 0 {
		return result
	}

	bestTargetIdx := -1
	bestDistance := math.MaxFloat64
	adjustedRadius := detector.toleranceRadius

	for idx, target := range targets {
		if detector.adaptiveThreshold {
			adjustedRadius = detector.calculateAdaptiveRadius(target)
		}

		targetCenterX := float64(target.X) + float64(target.Width)/2
		targetCenterY := float64(target.Y) + float64(target.Height)/2

		dx := float64(clickX) - targetCenterX
		dy := float64(clickY) - targetCenterY
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance < bestDistance {
			bestDistance = distance
			bestTargetIdx = idx
		}
	}

	if bestTargetIdx >= 0 {
		target := targets[bestTargetIdx]
		targetCenterX := float64(target.X) + float64(target.Width)/2
		targetCenterY := float64(target.Y) + float64(target.Height)/2

		result.TargetIndex = bestTargetIdx
		result.Distance = bestDistance
		result.OffsetX = float64(clickX) - targetCenterX
		result.OffsetY = float64(clickY) - targetCenterY
		result.AdjustedRadius = adjustedRadius

		result.IsHit = bestDistance <= adjustedRadius
		result.NearMiss = bestDistance > adjustedRadius && bestDistance <= adjustedRadius*1.4
		result.EdgeHit = detector.isEdgeHit(clickX, clickY, target)

		result.Confidence = detector.calculateConfidence(result, target)
		result.HitType = detector.classifyHitType(result)

		result.ErrorMargin = detector.calculateErrorMargin(result)
	}

	return result
}

func (detector *EnhancedPositionDetector) calculateAdaptiveRadius(target TargetImage) float64 {
	baseRadius := float64(target.Width+target.Height) / 4

	diagonal := math.Sqrt(float64(target.Width*target.Width + target.Height*target.Height))
	circularRadius := diagonal / 2

	adaptiveRadius := (baseRadius + circularRadius) / 2

	if detector.smartTolerance {
		sizeVariation := float64(target.Width-target.Height) / float64(target.Width+target.Height+1)
		adaptiveRadius *= (1.0 + sizeVariation*0.2)
	}

	return adaptiveRadius
}

func (detector *EnhancedPositionDetector) isEdgeHit(clickX, clickY int, target TargetImage) bool {
	clickXFloat := float64(clickX)
	clickYFloat := float64(clickY)

	distToLeft := math.Abs(clickXFloat - float64(target.X))
	distToRight := math.Abs(clickXFloat - float64(target.X+target.Width))
	distToTop := math.Abs(clickYFloat - float64(target.Y))
	distToBottom := math.Abs(clickYFloat - float64(target.Y+target.Height))

	minDist := math.Min(math.Min(distToLeft, distToRight), math.Min(distToTop, distToBottom))

	effectiveEdgeTolerance := detector.edgeTolerance * (float64(target.Width+target.Height) / 80)

	return minDist <= effectiveEdgeTolerance
}

func (detector *EnhancedPositionDetector) calculateConfidence(result *PositionDetectionResult, target TargetImage) float64 {
	confidence := 1.0

	maxDistance := result.AdjustedRadius
	if maxDistance > 0 {
		distanceRatio := 1.0 - (result.Distance / maxDistance)
		confidence *= (0.5 + distanceRatio*0.5)
	}

	if result.NearMiss {
		confidence *= 0.6
	}

	if result.EdgeHit {
		confidence *= 0.85
	}

	targetSize := float64(target.Width + target.Height)
	if targetSize > 0 {
		precisionBonus := 1.0 - (result.Distance / targetSize)
		if precisionBonus > 0 {
			confidence += precisionBonus * 0.1
		}
	}

	return math.Min(confidence, 1.0)
}

func (detector *EnhancedPositionDetector) classifyHitType(result *PositionDetectionResult) string {
	if result.Distance <= result.AdjustedRadius*0.3 {
		return "perfect"
	} else if result.Distance <= result.AdjustedRadius*0.7 {
		return "good"
	} else if result.Distance <= result.AdjustedRadius {
		return "acceptable"
	} else if result.NearMiss {
		return "near_miss"
	}
	return "miss"
}

func (detector *EnhancedPositionDetector) calculateErrorMargin(result *PositionDetectionResult) float64 {
	if !result.IsHit {
		return result.Distance - result.AdjustedRadius
	}

	baseError := result.Distance / result.AdjustedRadius * 5

	if result.EdgeHit {
		baseError *= 1.5
	}

	return baseError
}

type MultiTargetPositionAnalyzer struct {
	detector *EnhancedPositionDetector
}

func NewMultiTargetPositionAnalyzer() *MultiTargetPositionAnalyzer {
	return &MultiTargetPositionAnalyzer{
		detector: NewEnhancedPositionDetector(),
	}
}

type MultiTargetAnalysisResult struct {
	TotalTargets    int
	HitTargets      int
	MissedTargets   []int
	NearMissTargets []int
	ExtraClicks     int
	HitRate         float64
	Accuracy        float64
	Confidence      float64
	Results         []*PositionDetectionResult
	ErrorPatterns  []string
}

func (analyzer *MultiTargetPositionAnalyzer) Analyze(clickX, clickY int, targets []TargetImage, expectedCount int) *MultiTargetAnalysisResult {
	result := &MultiTargetAnalysisResult{
		TotalTargets: len(targets),
		Results:      make([]*PositionDetectionResult, 0),
		MissedTargets: make([]int, 0),
		NearMissTargets: make([]int, 0),
		ErrorPatterns: make([]string, 0),
	}

	detectionResult := analyzer.detector.DetectPosition(clickX, clickY, targets)
	result.Results = append(result.Results, detectionResult)

	if detectionResult.IsHit {
		result.HitTargets++
	} else if detectionResult.NearMiss {
		result.NearMissTargets = append(result.NearMissTargets, detectionResult.TargetIndex)
	} else {
		result.MissedTargets = append(result.MissedTargets, detectionResult.TargetIndex)
		result.ErrorPatterns = append(result.ErrorPatterns,
			fmt.Sprintf("点击位置(x:%d, y:%d)偏离目标%d，偏移量:%.1f",
				clickX, clickY, detectionResult.TargetIndex, detectionResult.Distance))
	}

	if len(targets) > 0 {
		result.HitRate = float64(result.HitTargets) / float64(len(targets))
	}

	if expectedCount > 0 {
		result.Accuracy = float64(result.HitTargets) / float64(expectedCount)
	}

	avgConfidence := 0.0
	for _, r := range result.Results {
		avgConfidence += r.Confidence
	}
	if len(result.Results) > 0 {
		result.Confidence = avgConfidence / float64(len(result.Results))
	}

	return result
}

type PositionClusterAnalyzer struct {
	clusterRadius float64
}

func NewPositionClusterAnalyzer() *PositionClusterAnalyzer {
	return &PositionClusterAnalyzer{
		clusterRadius: 30.0,
	}
}

type Cluster struct {
	CentroidX float64
	CentroidY float64
	Count     int
	Points    []struct{ X, Y int }
	SpreadX   float64
	SpreadY   float64
	Density   float64
}

func (analyzer *PositionClusterAnalyzer) AnalyzeClusters(clicks []ClickData) []*Cluster {
	clusters := make([]*Cluster, 0)

	if len(clicks) == 0 {
		return clusters
	}

	visited := make([]bool, len(clicks))

	for i := 0; i < len(clicks); i++ {
		if visited[i] {
			continue
		}

		cluster := &Cluster{
			Points: make([]struct{ X, Y int }, 0),
		}

		analyzer.expandCluster(clicks, i, visited, cluster)

		if len(cluster.Points) > 0 {
			analyzer.calculateClusterMetrics(cluster)
			clusters = append(clusters, cluster)
		}
	}

	return clusters
}

func (analyzer *PositionClusterAnalyzer) expandCluster(clicks []ClickData, index int, visited []bool, cluster *Cluster) {
	visited[index] = true
	cluster.Points = append(cluster.Points, struct{ X, Y int }{clicks[index].X, clicks[index].Y})

	for i := 0; i < len(clicks); i++ {
		if !visited[i] {
			dx := float64(clicks[i].X - clicks[index].X)
			dy := float64(clicks[i].Y - clicks[index].Y)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance <= analyzer.clusterRadius {
				analyzer.expandCluster(clicks, i, visited, cluster)
			}
		}
	}
}

func (analyzer *PositionClusterAnalyzer) calculateClusterMetrics(cluster *Cluster) {
	if len(cluster.Points) == 0 {
		return
	}

	cluster.Count = len(cluster.Points)

	var sumX, sumY float64
	for _, p := range cluster.Points {
		sumX += float64(p.X)
		sumY += float64(p.Y)
	}
	cluster.CentroidX = sumX / float64(cluster.Count)
	cluster.CentroidY = sumY / float64(cluster.Count)

	var varianceX, varianceY float64
	for _, p := range cluster.Points {
		varianceX += (float64(p.X) - cluster.CentroidX) * (float64(p.X) - cluster.CentroidX)
		varianceY += (float64(p.Y) - cluster.CentroidY) * (float64(p.Y) - cluster.CentroidY)
	}
	cluster.SpreadX = math.Sqrt(varianceX / float64(cluster.Count))
	cluster.SpreadY = math.Sqrt(varianceY / float64(cluster.Count))

	area := 4 * cluster.SpreadX * cluster.SpreadY
	if area > 0 {
		cluster.Density = float64(cluster.Count) / area
	}
}

type TrajectoryPositionAnalyzer struct {
	windowSize   int
	smoothingFactor float64
}

func NewTrajectoryPositionAnalyzer() *TrajectoryPositionAnalyzer {
	return &TrajectoryPositionAnalyzer{
		windowSize:   5,
		smoothingFactor: 0.3,
	}
}

type TrajectoryAnalysis struct {
	SmoothedPositions []struct{ X, Y float64 }
	Velocities        []float64
	Accelerations     []float64
	DirectionChanges  int
	PathLength        float64
	Directness        float64
}

func (analyzer *TrajectoryPositionAnalyzer) AnalyzeTrajectory(clicks []ClickData) *TrajectoryAnalysis {
	analysis := &TrajectoryAnalysis{}

	if len(clicks) < 2 {
		return analysis
	}

	analysis.SmoothedPositions = analyzer.smoothPositions(clicks)
	analysis.Velocities = analyzer.calculateVelocities(clicks)
	analysis.Accelerations = analyzer.calculateAccelerations(clicks)
	analysis.DirectionChanges = analyzer.countDirectionChanges(clicks)
	analysis.PathLength = analyzer.calculatePathLength(clicks)
	analysis.Directness = analyzer.calculateDirectness(clicks)

	return analysis
}

func (analyzer *TrajectoryPositionAnalyzer) smoothPositions(clicks []ClickData) []struct{ X, Y float64 } {
	smoothed := make([]struct{ X, Y float64 }, len(clicks))

	window := analyzer.windowSize
	if window < 3 {
		window = 3
	}

	for i := 0; i < len(clicks); i++ {
		start := i - window/2
		if start < 0 {
			start = 0
		}
		end := i + window/2
		if end >= len(clicks) {
			end = len(clicks) - 1
		}

		var sumX, sumY float64
		count := 0
		for j := start; j <= end; j++ {
			sumX += float64(clicks[j].X)
			sumY += float64(clicks[j].Y)
			count++
		}

		smoothed[i].X = sumX / float64(count)
		smoothed[i].Y = sumY / float64(count)
	}

	return smoothed
}

func (analyzer *TrajectoryPositionAnalyzer) calculateVelocities(clicks []ClickData) []float64 {
	velocities := make([]float64, len(clicks))

	for i := 1; i < len(clicks); i++ {
		dx := float64(clicks[i].X - clicks[i-1].X)
		dy := float64(clicks[i].Y - clicks[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)

		dt := float64(clicks[i].Timestamp-clicks[i-1].Timestamp) / 1000.0
		if dt > 0 {
			velocities[i] = distance / dt
		}
	}

	return velocities
}

func (analyzer *TrajectoryPositionAnalyzer) calculateAccelerations(clicks []ClickData) []float64 {
	accelerations := make([]float64, len(clicks))
	velocities := analyzer.calculateVelocities(clicks)

	for i := 2; i < len(clicks); i++ {
		dv := velocities[i] - velocities[i-1]
		dt := float64(clicks[i].Timestamp-clicks[i-1].Timestamp) / 1000.0
		if dt > 0 {
			accelerations[i] = dv / dt
		}
	}

	return accelerations
}

func (analyzer *TrajectoryPositionAnalyzer) countDirectionChanges(clicks []ClickData) int {
	if len(clicks) < 3 {
		return 0
	}

	changes := 0
	for i := 2; i < len(clicks); i++ {
		dx1 := float64(clicks[i-1].X - clicks[i-2].X)
		dy1 := float64(clicks[i-1].Y - clicks[i-2].Y)
		dx2 := float64(clicks[i].X - clicks[i-1].X)
		dy2 := float64(clicks[i].Y - clicks[i-1].Y)

		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle < 0.5 {
				changes++
			}
		}
	}

	return changes
}

func (analyzer *TrajectoryPositionAnalyzer) calculatePathLength(clicks []ClickData) float64 {
	var length float64

	for i := 1; i < len(clicks); i++ {
		dx := float64(clicks[i].X - clicks[i-1].X)
		dy := float64(clicks[i].Y - clicks[i-1].Y)
		length += math.Sqrt(dx*dx + dy*dy)
	}

	return length
}

func (analyzer *TrajectoryPositionAnalyzer) calculateDirectness(clicks []ClickData) float64 {
	if len(clicks) < 2 {
		return 0
	}

	startX := float64(clicks[0].X)
	startY := float64(clicks[0].Y)
	endX := float64(clicks[len(clicks)-1].X)
	endY := float64(clicks[len(clicks)-1].Y)

	directDistance := math.Sqrt((endX-startX)*(endX-startX) + (endY-startY)*(endY-startY))
	pathLength := analyzer.calculatePathLength(clicks)

	if pathLength > 0 {
		return directDistance / pathLength
	}

	return 0
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
	
	// 新增压力检测字段
	PressureTrend         float64   `json:"pressure_trend"`
	PressurePeaks         []float64 `json:"pressure_peaks"`
	PressureValleys       []float64 `json:"pressure_valleys"`
	PressureDerivative    float64   `json:"pressure_derivative"`
	PressureJerk          float64   `json:"pressure_jerk"`
	AbnormalPressureRatio float64   `json:"abnormal_pressure_ratio"`
	PressurePattern       string    `json:"pressure_pattern"`
	PressureAnomalyScore  float64   `json:"pressure_anomaly_score"`
	StabilizationTime     float64   `json:"stabilization_time"`
	ReleaseVelocity       float64   `json:"release_velocity"`
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
	timestamps := make([]int64, 0)
	
	for _, event := range clickEvents {
		if pressure, ok := event["pressure"].(float64); ok {
			pressures = append(pressures, pressure)
		} else if force, ok := event["force"].(float64); ok {
			pressures = append(pressures, force)
		}
		
		if ts, ok := event["timestamp"].(int64); ok {
			timestamps = append(timestamps, ts)
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
	
	if features.MeanPressure > 0 {
		features.PressureConsistency = 1.0 - math.Min(math.Sqrt(features.PressureVariance)/features.MeanPressure, 1.0)
	}

	features.IsBotLike = features.PressureConsistency > 0.95 && features.MeanPressure > 0.8

	// 新增压力检测增强功能
	features.PressureTrend = cpa.calculatePressureTrend(pressures)
	features.PressurePeaks, features.PressureValleys = cpa.detectPressureExtrema(pressures)
	features.PressureDerivative = cpa.calculatePressureDerivative(pressures, timestamps)
	features.PressureJerk = cpa.calculatePressureJerk(pressures, timestamps)
	features.AbnormalPressureRatio = cpa.calculateAbnormalPressureRatio(pressures)
	features.PressurePattern = cpa.classifyPressurePattern(pressures)
	features.PressureAnomalyScore = cpa.calculatePressureAnomalyScore(features)
	features.StabilizationTime = cpa.calculateStabilizationTime(pressures)
	features.ReleaseVelocity = cpa.calculateReleaseVelocity(pressures, timestamps)

	return features
}

func (cpa *ClickPressureAnalyzer) calculatePressureTrend(pressures []float64) float64 {
	if len(pressures) < 2 {
		return 0
	}
	
	n := float64(len(pressures))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	
	for i, p := range pressures {
		x := float64(i)
		y := p
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}
	
	return (n*sumXY - sumX*sumY) / denominator
}

func (cpa *ClickPressureAnalyzer) detectPressureExtrema(pressures []float64) ([]float64, []float64) {
	if len(pressures) < 3 {
		return []float64{}, []float64{}
	}
	
	peaks := []float64{}
	valleys := []float64{}
	
	for i := 1; i < len(pressures)-1; i++ {
		if pressures[i] > pressures[i-1] && pressures[i] > pressures[i+1] {
			peaks = append(peaks, pressures[i])
		} else if pressures[i] < pressures[i-1] && pressures[i] < pressures[i+1] {
			valleys = append(valleys, pressures[i])
		}
	}
	
	return peaks, valleys
}

func (cpa *ClickPressureAnalyzer) calculatePressureDerivative(pressures []float64, timestamps []int64) float64 {
	if len(pressures) < 2 {
		return 0
	}
	
	derivatives := make([]float64, 0)
	for i := 1; i < len(pressures); i++ {
		dt := 1.0
		if len(timestamps) > i {
			dt = float64(timestamps[i] - timestamps[i-1])
			if dt <= 0 {
				dt = 1.0
			}
		}
		derivatives = append(derivatives, (pressures[i]-pressures[i-1])/dt)
	}
	
	return cpa.mean(derivatives)
}

func (cpa *ClickPressureAnalyzer) calculatePressureJerk(pressures []float64, timestamps []int64) float64 {
	if len(pressures) < 3 {
		return 0
	}
	
	derivatives := make([]float64, 0)
	for i := 1; i < len(pressures); i++ {
		dt := 1.0
		if len(timestamps) > i {
			dt = float64(timestamps[i] - timestamps[i-1])
			if dt <= 0 {
				dt = 1.0
			}
		}
		derivatives = append(derivatives, (pressures[i]-pressures[i-1])/dt)
	}
	
	if len(derivatives) < 2 {
		return 0
	}
	
	jerks := make([]float64, 0)
	for i := 1; i < len(derivatives); i++ {
		dt := 1.0
		if len(timestamps) > i+1 {
			dt = float64(timestamps[i+1] - timestamps[i])
			if dt <= 0 {
				dt = 1.0
			}
		}
		jerks = append(jerks, (derivatives[i]-derivatives[i-1])/dt)
	}
	
	return cpa.mean(jerks)
}

func (cpa *ClickPressureAnalyzer) calculateAbnormalPressureRatio(pressures []float64) float64 {
	if len(pressures) == 0 {
		return 0
	}
	
	mean := cpa.mean(pressures)
	std := math.Sqrt(cpa.variance(pressures))
	
	abnormalCount := 0
	for _, p := range pressures {
		if math.Abs(p-mean) > 2*std {
			abnormalCount++
		}
	}
	
	return float64(abnormalCount) / float64(len(pressures))
}

func (cpa *ClickPressureAnalyzer) classifyPressurePattern(pressures []float64) string {
	if len(pressures) < 3 {
		return "unknown"
	}
	
	mean := cpa.mean(pressures)
	std := math.Sqrt(cpa.variance(pressures))
	coeffVar := std / mean
	
	// 检测各种模式
	if coeffVar < 0.1 {
		return "stable"
	} else if coeffVar > 0.5 {
		return "volatile"
	}
	
	// 检测递增/递减趋势
	trend := cpa.calculatePressureTrend(pressures)
	if trend > 0.01 {
		return "increasing"
	} else if trend < -0.01 {
		return "decreasing"
	}
	
	// 检测周期性模式
	periodicity := cpa.detectPressurePeriodicity(pressures)
	if periodicity > 0.7 {
		return "periodic"
	}
	
	return "variable"
}

func (cpa *ClickPressureAnalyzer) detectPressurePeriodicity(pressures []float64) float64 {
	if len(pressures) < 4 {
		return 0
	}
	
	n := len(pressures)
	maxLag := n / 2
	if maxLag < 2 {
		return 0
	}
	
	maxCorrelation := 0.0
	for lag := 1; lag <= maxLag; lag++ {
		sumXY, sumX, sumY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0
		count := 0
		for i := 0; i < n-lag; i++ {
			x := pressures[i]
			y := pressures[i+lag]
			sumXY += x * y
			sumX += x
			sumY += y
			sumX2 += x * x
			sumY2 += y * y
			count++
		}
		
		if count == 0 {
			continue
		}
		
		fCount := float64(count)
		denominator := math.Sqrt((fCount*sumX2-sumX*sumX) * (fCount*sumY2-sumY*sumY))
		if denominator == 0 {
			continue
		}
		
		correlation := (fCount*sumXY - sumX*sumY) / denominator
		if correlation > maxCorrelation {
			maxCorrelation = correlation
		}
	}
	
	return maxCorrelation
}

func (cpa *ClickPressureAnalyzer) calculatePressureAnomalyScore(features *PressureFeatures) float64 {
	if !features.HasPressureData {
		return 0
	}
	
	score := 0.0
	
	// 压力一致性过高（机器人特征）
	if features.PressureConsistency > 0.95 {
		score += 0.3
	}
	
	// 压力过高或过低
	if features.MeanPressure > 0.9 || features.MeanPressure < 0.1 {
		score += 0.2
	}
	
	// 异常压力比例高
	if features.AbnormalPressureRatio > 0.3 {
		score += 0.25
	}
	
	// 压力变化过于剧烈
	if math.Abs(features.PressureJerk) > 0.1 {
		score += 0.15
	}
	
	// 压力模式异常
	if features.PressurePattern == "stable" && len(features.Pressures) > 5 {
		score += 0.1
	}
	
	return math.Min(score, 1.0)
}

func (cpa *ClickPressureAnalyzer) calculateStabilizationTime(pressures []float64) float64 {
	if len(pressures) < 3 {
		return 0
	}
	
	mean := cpa.mean(pressures)
	std := math.Sqrt(cpa.variance(pressures))
	threshold := std * 0.1
	
	stabilizedIdx := -1
	for i := len(pressures) - 1; i >= 0; i-- {
		if math.Abs(pressures[i]-mean) > threshold {
			stabilizedIdx = i
			break
		}
	}
	
	if stabilizedIdx == -1 {
		return 0
	}
	
	return float64(len(pressures) - stabilizedIdx - 1)
}

func (cpa *ClickPressureAnalyzer) calculateReleaseVelocity(pressures []float64, timestamps []int64) float64 {
	if len(pressures) < 2 {
		return 0
	}
	
	// 找到最大压力点
	maxIdx := 0
	maxPressure := pressures[0]
	for i, p := range pressures {
		if p > maxPressure {
			maxPressure = p
			maxIdx = i
		}
	}
	
	// 计算从最大压力到释放的速度
	if maxIdx == len(pressures)-1 {
		return 0
	}
	
	releaseDuration := 1.0
	if len(timestamps) > maxIdx+1 {
		releaseDuration = float64(timestamps[len(timestamps)-1] - timestamps[maxIdx])
		if releaseDuration <= 0 {
			releaseDuration = 1.0
		}
	}
	
	releaseAmount := maxPressure - pressures[len(pressures)-1]
	return releaseAmount / releaseDuration
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
			// 新增异常模式
			{
				Name:        "abnormally_fast",
				Description: "点击速度异常快",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.TimingPattern == "very_fast" &&
						result.TimingAnalysis.TotalDuration < 500 &&
						result.ClickPattern != nil &&
						result.ClickPattern.ClickCount > 2
				},
				Weight: 0.25,
			},
			{
				Name:        "zero_variance",
				Description: "点击间隔方差为零",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.DurationVariance == 0 &&
						len(result.TimingAnalysis.ResponseTimes) > 2
				},
				Weight: 0.35,
			},
			{
				Name:        "extreme_clustering",
				Description: "点击位置极度聚集",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.ClickPattern == nil ||
						result.ClickPattern.PositionDistribution == nil {
						return false
					}
					dist := result.ClickPattern.PositionDistribution
					return dist.SpatialEntropy < 0.2 &&
						dist.ClusterCount == 1
				},
				Weight: 0.2,
			},
			{
				Name:        "periodic_pattern",
				Description: "周期性点击模式",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.PeriodicityScore > 0.8 &&
						result.TimingAnalysis.RhythmConsistency > 0.95
				},
				Weight: 0.25,
			},
			{
				Name:        "mechanical_jerk",
				Description: "机械性加加速度模式",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.JerkPattern == "smooth" &&
						result.TimingAnalysis.IsRhythmic
				},
				Weight: 0.2,
			},
			{
				Name:        "attention_collapse",
				Description: "注意力急剧下降",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.AttentionDecay > 0.8
				},
				Weight: 0.15,
			},
			{
				Name:        "outlier_click",
				Description: "存在大量离群点击",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.ClickPattern == nil ||
						result.ClickPattern.PositionDistribution == nil {
						return false
					}
					dist := result.ClickPattern.PositionDistribution
					return dist.OutlierScore > 0.5
				},
				Weight: 0.2,
			},
			{
				Name:        "predictable_pattern",
				Description: "高度可预测的点击模式",
				Detector: func(result *ClickAnalysisResult) bool {
					if result.TimingAnalysis == nil {
						return false
					}
					return result.TimingAnalysis.ComplexityScore < 0.5 &&
						result.TimingAnalysis.RhythmConsistency > 0.9
				},
				Weight: 0.25,
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

// MLEnhancedAnomalyDetector 使用机器学习增强的异常检测
type MLEnhancedAnomalyDetector struct {
	ensembleModels []AnomalyModel
	weights        []float64
}

type AnomalyModel interface {
	Predict(result *ClickAnalysisResult) float64
}

func NewMLEnhancedAnomalyDetector() *MLEnhancedAnomalyDetector {
	return &MLEnhancedAnomalyDetector{
		ensembleModels: []AnomalyModel{
			NewIsolationForestModel(),
			NewAutoEncoderModel(),
			NewOneClassSVMModel(),
			NewLOFModel(),
		},
		weights: []float64{0.25, 0.25, 0.25, 0.25},
	}
}

func (med *MLEnhancedAnomalyDetector) Detect(result *ClickAnalysisResult) (float64, []string) {
	scores := make([]float64, len(med.ensembleModels))
	detectedPatterns := make([]string, 0)

	for i, model := range med.ensembleModels {
		scores[i] = model.Predict(result)
	}

	// 集成预测：加权平均
	ensembleScore := 0.0
	for i, score := range scores {
		ensembleScore += score * med.weights[i]
	}

	// 检测具体的异常模式
	if ensembleScore > 0.5 {
		detectedPatterns = append(detectedPatterns, med.detectSpecificAnomalies(result, scores)...)
	}

	return ensembleScore, detectedPatterns
}

func (med *MLEnhancedAnomalyDetector) detectSpecificAnomalies(result *ClickAnalysisResult, scores []float64) []string {
	patterns := make([]string, 0)

	// 根据各模型的得分判断具体异常类型
	if scores[0] > 0.7 { // Isolation Forest 检测到的异常
		patterns = append(patterns, "isolation_forest_outlier: 孤立森林检测到离群点")
	}

	if scores[1] > 0.7 { // AutoEncoder 检测到的异常
		patterns = append(patterns, "autoencoder_reconstruction: 自编码器重构误差异常")
	}

	if scores[2] > 0.7 { // One-Class SVM 检测到的异常
		patterns = append(patterns, "oneclass_svm_boundary: SVM边界外样本")
	}

	if scores[3] > 0.7 { // LOF 检测到的异常
		patterns = append(patterns, "lof_outlier: 局部离群因子异常")
	}

	return patterns
}

// IsolationForestModel 孤立森林模型
type IsolationForestModel struct {
	trees []isolationTree
}

type isolationTree struct {
	splitFeature int
	splitValue   float64
	left         *isolationTree
	right        *isolationTree
	isLeaf       bool
	depth        int
}

func NewIsolationForestModel() *IsolationForestModel {
	return &IsolationForestModel{
		trees: buildIsolationForest(100),
	}
}

func buildIsolationForest(numTrees int) []isolationTree {
	trees := make([]isolationTree, numTrees)
	for i := 0; i < numTrees; i++ {
		trees[i] = buildIsolationTree(0, 10)
	}
	return trees
}

func buildIsolationTree(depth, maxDepth int) isolationTree {
	if depth >= maxDepth {
		return isolationTree{isLeaf: true, depth: depth}
	}
	return isolationTree{
		splitFeature: rand.Intn(8),
		splitValue:   rand.Float64(),
		left:         &isolationTree{isLeaf: true, depth: depth + 1},
		right:        &isolationTree{isLeaf: true, depth: depth + 1},
		isLeaf:       false,
		depth:        depth,
	}
}

func (ifm *IsolationForestModel) Predict(result *ClickAnalysisResult) float64 {
	if result == nil {
		return 0.5
	}

	features := ifm.extractFeatures(result)
	if len(features) == 0 {
		return 0.5
	}

	avgDepth := 0.0
	for _, tree := range ifm.trees {
		avgDepth += float64(ifm.traverseTree(tree, features, 0))
	}
	avgDepth /= float64(len(ifm.trees))

	// 计算异常分数
	n := float64(len(ifm.trees))
	score := math.Pow(2, -avgDepth/math.Log(float64(n)))

	return score
}

func (ifm *IsolationForestModel) extractFeatures(result *ClickAnalysisResult) []float64 {
	features := make([]float64, 0)

	if result.ClickPattern != nil {
		features = append(features, result.ClickPattern.Regularity)
		features = append(features, result.ClickPattern.ClusteringScore)
	}

	if result.TimingAnalysis != nil {
		features = append(features, float64(result.TimingAnalysis.TotalDuration)/5000.0)
		features = append(features, result.TimingAnalysis.IntervalCoefficientOfVariation)
		features = append(features, result.TimingAnalysis.RhythmConsistency)
	}

	if result.AccuracyAnalysis != nil {
		features = append(features, result.AccuracyAnalysis.Accuracy)
		features = append(features, result.AccuracyAnalysis.AverageMissDistance/100.0)
	}

	return features
}

func (ifm *IsolationForestModel) traverseTree(tree isolationTree, features []float64, depth int) int {
	if tree.isLeaf {
		return depth
	}

	if tree.splitFeature < len(features) && features[tree.splitFeature] < tree.splitValue {
		if tree.left != nil {
			return ifm.traverseTree(*tree.left, features, depth+1)
		}
	} else {
		if tree.right != nil {
			return ifm.traverseTree(*tree.right, features, depth+1)
		}
	}

	return depth
}

// AutoEncoderModel 自编码器模型
type AutoEncoderModel struct {
	weights1 [][]float64
	weights2 [][]float64
	bias1    []float64
	bias2    []float64
}

func NewAutoEncoderModel() *AutoEncoderModel {
	return &AutoEncoderModel{
		weights1: [][]float64{
			{0.5, 0.3, 0.2},
			{0.3, 0.5, 0.2},
			{0.2, 0.3, 0.5},
		},
		weights2: [][]float64{
			{0.5, 0.3, 0.2},
			{0.3, 0.5, 0.2},
			{0.2, 0.3, 0.5},
		},
		bias1: []float64{0.1, 0.1, 0.1},
		bias2: []float64{0.1, 0.1, 0.1},
	}
}

func (aem *AutoEncoderModel) Predict(result *ClickAnalysisResult) float64 {
	if result == nil {
		return 0.5
	}

	features := aem.extractFeatures(result)
	if len(features) == 0 {
		return 0.5
	}

	// 前向传播
	hidden := make([]float64, 3)
	for i := range hidden {
		sum := aem.bias1[i]
		for j, f := range features {
			if j < len(aem.weights1[i]) {
				sum += f * aem.weights1[i][j]
			}
		}
		hidden[i] = math.Tanh(sum)
	}

	reconstructed := make([]float64, len(features))
	for i := range reconstructed {
		sum := aem.bias2[i%3]
		for j, h := range hidden {
			if i < len(aem.weights2[j]) {
				sum += h * aem.weights2[j][i%3]
			}
		}
		reconstructed[i] = math.Tanh(sum)
	}

	// 计算重构误差
	error := 0.0
	for i, f := range features {
		if i < len(reconstructed) {
			error += math.Abs(f - reconstructed[i])
		}
	}
	error /= float64(len(features))

	// 将误差转换为异常分数
	return math.Min(error*5, 1.0)
}

func (aem *AutoEncoderModel) extractFeatures(result *ClickAnalysisResult) []float64 {
	features := make([]float64, 0)

	if result.ClickPattern != nil {
		features = append(features, result.ClickPattern.Regularity)
	}
	if result.TimingAnalysis != nil {
		features = append(features, result.TimingAnalysis.IntervalCoefficientOfVariation)
	}
	if result.AccuracyAnalysis != nil {
		features = append(features, result.AccuracyAnalysis.Accuracy)
	}

	return features
}

// OneClassSVMModel 一类支持向量机模型
type OneClassSVMModel struct {
	supportVectors [][]float64
	weights        []float64
	bias           float64
	kernel         string
}

func NewOneClassSVMModel() *OneClassSVMModel {
	return &OneClassSVMModel{
		supportVectors: [][]float64{
			{0.5, 0.5, 0.5},
			{0.6, 0.4, 0.5},
			{0.4, 0.6, 0.5},
		},
		weights: []float64{0.33, 0.33, 0.34},
		bias:    -0.5,
		kernel:  "rbf",
	}
}

func (ocsvm *OneClassSVMModel) Predict(result *ClickAnalysisResult) float64 {
	if result == nil {
		return 0.5
	}

	features := ocsvm.extractFeatures(result)
	if len(features) == 0 {
		return 0.5
	}

	// 计算决策函数值
	score := ocsvm.bias
	for i, sv := range ocsvm.supportVectors {
		score += ocsvm.weights[i] * ocsvm.rbfKernel(features, sv)
	}

	// 将决策函数值转换为异常分数
	return 1.0 / (1.0 + math.Exp(score))
}

func (ocsvm *OneClassSVMModel) rbfKernel(x, y []float64) float64 {
	sum := 0.0
	for i := range x {
		if i < len(y) {
			sum += (x[i] - y[i]) * (x[i] - y[i])
		}
	}
	return math.Exp(-sum)
}

func (ocsvm *OneClassSVMModel) extractFeatures(result *ClickAnalysisResult) []float64 {
	features := make([]float64, 3)

	if result.ClickPattern != nil {
		features[0] = result.ClickPattern.Regularity
	}
	if result.TimingAnalysis != nil {
		features[1] = result.TimingAnalysis.IntervalCoefficientOfVariation
	}
	if result.AccuracyAnalysis != nil {
		features[2] = result.AccuracyAnalysis.Accuracy
	}

	return features
}

// LOFModel 局部离群因子模型
type LOFModel struct {
	k int
}

func NewLOFModel() *LOFModel {
	return &LOFModel{k: 5}
}

func (lof *LOFModel) Predict(result *ClickAnalysisResult) float64 {
	if result == nil {
		return 0.5
	}

	// 使用模拟的LOF分数计算
	score := 0.0
	count := 0

	if result.ClickPattern != nil {
		// 高规律性和低聚集性可能表示异常
		if result.ClickPattern.Regularity > 0.9 {
			score += 0.3
			count++
		}
		if result.ClickPattern.ClusteringScore < 0.2 {
			score += 0.25
			count++
		}
	}

	if result.TimingAnalysis != nil {
		if result.TimingAnalysis.IsRhythmic {
			score += 0.25
			count++
		}
		if result.TimingAnalysis.IntervalCoefficientOfVariation < 0.2 {
			score += 0.2
			count++
		}
	}

	if count > 0 {
		return math.Min(score/float64(count), 1.0)
	}

	return 0.3
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
