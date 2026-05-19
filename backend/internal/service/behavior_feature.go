package service

import (
	"fmt"
	"math"
	"sort"

	"github.com/hjtpx/hjtpx/internal/model"
)

type MouseBehaviorExtractor struct {
	windowSize int
	threshold  float64
}

func NewMouseBehaviorExtractor() *MouseBehaviorExtractor {
	return &MouseBehaviorExtractor{
		windowSize: 5,
		threshold:  0.15,
	}
}

func (m *MouseBehaviorExtractor) ExtractFeatures(data *model.MouseBehaviorData) *model.MouseBehaviorFeatures {
	features := &model.MouseBehaviorFeatures{
		AnomalyIndicators: make([]string, 0),
	}

	if data == nil || len(data.MousePoints) == 0 {
		features.AddAnomalyIndicator("无鼠标数据")
		features.OverallScore = 100
		features.IsHumanLike = false
		features.Confidence = 0.9
		return features
	}

	features.SpeedFeatures = m.extractSpeedFeatures(data.MousePoints)
	features.AccelerationFeatures = m.extractAccelerationFeatures(data.MousePoints)
	features.DistributionFeatures = m.extractDistributionFeatures(data.ClickPoints)
	features.DoubleClickFeatures = m.extractDoubleClickFeatures(data.ClickPoints)
	features.LatencyFeatures = m.extractLatencyFeatures(data.MousePoints, data.ClickPoints)
	
	features.CalculateOverallScore()

	return features
}

func (m *MouseBehaviorExtractor) extractSpeedFeatures(points []model.MousePoint) model.MouseSpeedFeature {
	feature := model.MouseSpeedFeature{}

	if len(points) < 2 {
		return feature
	}

	speeds := m.calculateSpeeds(points)
	if len(speeds) == 0 {
		return feature
	}

	feature.AverageSpeed = m.mean(speeds)
	feature.MedianSpeed = m.median(speeds)
	feature.MaxSpeed = m.max(speeds)
	feature.MinSpeed = m.min(speeds)
	feature.SpeedVariance = m.variance(speeds)
	feature.SpeedStdDev = math.Sqrt(feature.SpeedVariance)
	feature.SpeedSkewness = m.skewness(speeds)
	feature.SpeedKurtosis = m.kurtosis(speeds)
	feature.SpeedRange = feature.MaxSpeed - feature.MinSpeed

	feature.ZeroSpeedCount = m.countValues(speeds, 0)
	feature.LowSpeedCount = m.countBelow(speeds, 0.1)
	feature.HighSpeedCount = m.countAbove(speeds, 2.0)
	feature.SpeedOutliers = m.findOutliers(speeds)

	return feature
}

func (m *MouseBehaviorExtractor) extractAccelerationFeatures(points []model.MousePoint) model.MouseAccelerationFeature {
	feature := model.MouseAccelerationFeature{}

	if len(points) < 3 {
		return feature
	}

	speeds := m.calculateSpeeds(points)
	if len(speeds) < 2 {
		return feature
	}

	accelerations := m.calculateAccelerations(speeds, points)
	if len(accelerations) == 0 {
		return feature
	}

	feature.AverageAcceleration = m.mean(accelerations)
	feature.MaxAcceleration = m.max(accelerations)
	feature.MinAcceleration = m.min(accelerations)
	feature.AccelerationVariance = m.variance(accelerations)
	feature.AccelerationStdDev = math.Sqrt(feature.AccelerationVariance)

	jerks := m.calculateJerks(accelerations)
	if len(jerks) > 0 {
		feature.JerkAvg = m.mean(jerks)
		feature.JerkMax = m.max(jerks)
		feature.JerkMin = m.min(jerks)
		feature.JerkVariance = m.variance(jerks)
	}

	feature.PositiveAccelRatio = m.calculatePositiveRatio(accelerations)
	feature.NegativeAccelRatio = 1.0 - feature.PositiveAccelRatio

	feature.AccelerationPeaks = m.findPeaks(accelerations)
	feature.AccelerationValleys = m.findValleys(accelerations)
	feature.DirectionChanges = m.countDirectionChanges(points)

	return feature
}

func (m *MouseBehaviorExtractor) extractDistributionFeatures(clicks []model.ClickPoint) model.ClickDistributionFeature {
	feature := model.ClickDistributionFeature{}

	if len(clicks) == 0 {
		return feature
	}

	xValues := m.extractCoordinates(clicks, true)
	yValues := m.extractCoordinates(clicks, false)

	feature.XMean = m.mean(xValues)
	feature.YMean = m.mean(yValues)
	feature.XVariance = m.variance(xValues)
	feature.YVariance = m.variance(yValues)
	feature.XStdDev = math.Sqrt(feature.XVariance)
	feature.YStdDev = math.Sqrt(feature.YVariance)
	feature.XEntropy = m.calculateEntropy(xValues, 10)
	feature.YEntropy = m.calculateEntropy(yValues, 10)
	feature.XSkewness = m.skewness(xValues)
	feature.YSkewness = m.skewness(yValues)
	feature.XKurtosis = m.kurtosis(xValues)
	feature.YKurtosis = m.kurtosis(yValues)
	feature.SpreadX = m.max(xValues) - m.min(xValues)
	feature.SpreadY = m.max(yValues) - m.min(yValues)
	feature.CenterX = feature.XMean
	feature.CenterY = feature.YMean
	feature.Density = m.calculateDensity(xValues, yValues)
	feature.Clusters = m.estimateClusterCount(xValues, yValues)

	return feature
}

func (m *MouseBehaviorExtractor) extractDoubleClickFeatures(clicks []model.ClickPoint) model.DoubleClickFeature {
	feature := model.DoubleClickFeature{}

	if len(clicks) < 2 {
		return feature
	}

	intervals := m.calculateClickIntervals(clicks)
	if len(intervals) == 0 {
		return feature
	}

	for _, interval := range intervals {
		if interval < 100 {
			feature.FastDoubleClickRatio++
		} else if interval < 300 {
			feature.NormalDoubleClickRatio++
		}
	}

	feature.DoubleClickCount = m.countClickType(clicks, "double")
	feature.SingleClickCount = m.countClickType(clicks, "single")
	feature.TripleClickCount = m.countClickType(clicks, "triple")
	feature.AverageInterval = m.mean(intervals)
	feature.MinInterval = m.min(intervals)
	feature.MaxInterval = m.max(intervals)
	feature.IntervalVariance = m.variance(intervals)
	feature.IntervalStdDev = math.Sqrt(feature.IntervalVariance)
	feature.DoubleClickPositions = m.extractDoubleClickPositions(clicks)
	feature.ClickBurstCount = m.detectClickBursts(intervals)

	return feature
}

func (m *MouseBehaviorExtractor) extractLatencyFeatures(points []model.MousePoint, clicks []model.ClickPoint) model.ClickLatencyFeature {
	feature := model.ClickLatencyFeature{}

	if len(clicks) == 0 || len(points) == 0 {
		return feature
	}

	latencies := m.calculateLatencies(points, clicks)
	if len(latencies) == 0 {
		return feature
	}

	feature.AverageLatency = m.mean(latencies)
	feature.MedianLatency = m.median(latencies)
	feature.MinLatency = m.min(latencies)
	feature.MaxLatency = m.max(latencies)
	feature.LatencyVariance = m.variance(latencies)
	feature.LatencyStdDev = math.Sqrt(feature.LatencyVariance)
	feature.FastClickRatio = m.calculateFastClickRatio(latencies)
	feature.SlowClickRatio = m.calculateSlowClickRatio(latencies)
	feature.FirstClickDelay = m.calculateFirstClickDelay(clicks)
	feature.LastClickDelay = m.calculateLastClickDelay(clicks)
	feature.HesitationCount = m.countHesitations(latencies)
	feature.HesitationRatio = m.calculateHesitationRatio(latencies)
	feature.ReactionTimeTrend = m.calculateReactionTimeTrend(latencies)
	feature.LatencyOutliers = m.findLatencyOutliers(latencies)

	return feature
}

func (m *MouseBehaviorExtractor) calculateSpeeds(points []model.MousePoint) []float64 {
	speeds := make([]float64, 0)

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)

		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speed := distance / dt
			speeds = append(speeds, speed)
		}
	}

	return speeds
}

func (m *MouseBehaviorExtractor) calculateAccelerations(speeds []float64, points []model.MousePoint) []float64 {
	accelerations := make([]float64, 0)

	for i := 2; i < len(speeds); i++ {
		dt := float64(points[i].Timestamp - points[i-2].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	return accelerations
}

func (m *MouseBehaviorExtractor) calculateJerks(accelerations []float64) []float64 {
	jerks := make([]float64, 0)

	for i := 1; i < len(accelerations); i++ {
		jerk := accelerations[i] - accelerations[i-1]
		jerks = append(jerks, jerk)
	}

	return jerks
}

func (m *MouseBehaviorExtractor) calculateClickIntervals(clicks []model.ClickPoint) []float64 {
	intervals := make([]float64, 0)

	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	return intervals
}

func (m *MouseBehaviorExtractor) calculateLatencies(points []model.MousePoint, clicks []model.ClickPoint) []float64 {
	latencies := make([]float64, 0)

	for _, click := range clicks {
		latency := m.findNearestMovementTime(points, click.Timestamp)
		if latency > 0 {
			latencies = append(latencies, latency)
		}
	}

	return latencies
}

func (m *MouseBehaviorExtractor) findNearestMovementTime(points []model.MousePoint, clickTime int64) float64 {
	var nearestTime int64 = 0
	minDiff := int64(math.MaxInt64)

	for _, point := range points {
		if point.Timestamp < clickTime {
			diff := clickTime - point.Timestamp
			if diff < minDiff {
				minDiff = diff
				nearestTime = point.Timestamp
			}
		}
	}

	if nearestTime > 0 {
		return float64(clickTime - nearestTime)
	}
	return 0
}

func (m *MouseBehaviorExtractor) extractCoordinates(clicks []model.ClickPoint, isX bool) []float64 {
	coords := make([]float64, len(clicks))

	for i, click := range clicks {
		if isX {
			coords[i] = float64(click.X)
		} else {
			coords[i] = float64(click.Y)
		}
	}

	return coords
}

func (m *MouseBehaviorExtractor) calculateEntropy(values []float64, bins int) float64 {
	if len(values) == 0 {
		return 0
	}

	minVal := m.min(values)
	maxVal := m.max(values)

	if maxVal <= minVal {
		return 0
	}

	bucketCounts := make([]int, bins)
	binWidth := (maxVal - minVal) / float64(bins)
	if binWidth == 0 {
		return 0
	}

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

func (m *MouseBehaviorExtractor) calculateDensity(xValues, yValues []float64) float64 {
	if len(xValues) < 2 {
		return 0
	}

	spreadX := m.max(xValues) - m.min(xValues)
	spreadY := m.max(yValues) - m.min(yValues)

	if spreadX <= 0 || spreadY <= 0 {
		return 0
	}

	area := spreadX * spreadY
	return float64(len(xValues)) / area
}

func (m *MouseBehaviorExtractor) estimateClusterCount(xValues, yValues []float64) int {
	if len(xValues) < 3 {
		return 1
	}

	stdX := math.Sqrt(m.variance(xValues))
	stdY := math.Sqrt(m.variance(yValues))

	threshold := math.Max(stdX, stdY) * 1.5

	count := 0
	for i := 0; i < len(xValues); i++ {
		isNewCluster := true
		for j := 0; j < i; j++ {
			dx := xValues[i] - xValues[j]
			dy := yValues[i] - yValues[j]
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance < threshold {
				isNewCluster = false
				break
			}
		}
		if isNewCluster {
			count++
		}
	}

	return count
}

func (m *MouseBehaviorExtractor) countClickType(clicks []model.ClickPoint, clickType string) int {
	count := 0
	for _, click := range clicks {
		if click.ClickType == clickType {
			count++
		}
	}
	return count
}

func (m *MouseBehaviorExtractor) extractDoubleClickPositions(clicks []model.ClickPoint) [][]int {
	positions := make([][]int, 0)

	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		if interval < 300 {
			positions = append(positions, []int{clicks[i-1].X, clicks[i-1].Y, clicks[i].X, clicks[i].Y})
		}
	}

	return positions
}

func (m *MouseBehaviorExtractor) detectClickBursts(intervals []float64) int {
	burstCount := 0
	consecutiveFast := 0

	for _, interval := range intervals {
		if interval < 150 {
			consecutiveFast++
			if consecutiveFast >= 2 {
				burstCount++
			}
		} else {
			consecutiveFast = 0
		}
	}

	return burstCount
}

func (m *MouseBehaviorExtractor) calculateFastClickRatio(latencies []float64) float64 {
	if len(latencies) == 0 {
		return 0
	}

	fastCount := 0
	for _, latency := range latencies {
		if latency < 150 {
			fastCount++
		}
	}

	return float64(fastCount) / float64(len(latencies))
}

func (m *MouseBehaviorExtractor) calculateSlowClickRatio(latencies []float64) float64 {
	if len(latencies) == 0 {
		return 0
	}

	slowCount := 0
	for _, latency := range latencies {
		if latency > 500 {
			slowCount++
		}
	}

	return float64(slowCount) / float64(len(latencies))
}

func (m *MouseBehaviorExtractor) calculateFirstClickDelay(clicks []model.ClickPoint) float64 {
	if len(clicks) == 0 {
		return 0
	}
	return float64(clicks[0].Timestamp)
}

func (m *MouseBehaviorExtractor) calculateLastClickDelay(clicks []model.ClickPoint) float64 {
	if len(clicks) == 0 {
		return 0
	}
	return float64(clicks[len(clicks)-1].Timestamp)
}

func (m *MouseBehaviorExtractor) countHesitations(latencies []float64) int {
	count := 0
	for _, latency := range latencies {
		if latency > 300 {
			count++
		}
	}
	return count
}

func (m *MouseBehaviorExtractor) calculateHesitationRatio(latencies []float64) float64 {
	if len(latencies) == 0 {
		return 0
	}
	return float64(m.countHesitations(latencies)) / float64(len(latencies))
}

func (m *MouseBehaviorExtractor) calculateReactionTimeTrend(latencies []float64) float64 {
	if len(latencies) < 3 {
		return 0
	}

	firstThird := latencies[:len(latencies)/3]
	lastThird := latencies[len(latencies)*2/3:]

	firstMean := m.mean(firstThird)
	lastMean := m.mean(lastThird)

	if firstMean == 0 {
		return 0
	}

	return (lastMean - firstMean) / firstMean
}

func (m *MouseBehaviorExtractor) findLatencyOutliers(latencies []float64) []int {
	outliers := make([]int, 0)

	if len(latencies) < 3 {
		return outliers
	}

	mean := m.mean(latencies)
	stdDev := math.Sqrt(m.variance(latencies))

	for i, latency := range latencies {
		if math.Abs(latency-mean) > 2*stdDev {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

func (m *MouseBehaviorExtractor) countDirectionChanges(points []model.MousePoint) int {
	if len(points) < 3 {
		return 0
	}

	count := 0
	prevAngle := 0.0

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		angle := math.Atan2(dy, dx)

		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 0.5 {
				count++
			}
		}

		prevAngle = angle
	}

	return count
}

func (m *MouseBehaviorExtractor) findPeaks(values []float64) []float64 {
	peaks := make([]float64, 0)

	for i := 1; i < len(values)-1; i++ {
		if values[i] > values[i-1] && values[i] > values[i+1] {
			peaks = append(peaks, values[i])
		}
	}

	return peaks
}

func (m *MouseBehaviorExtractor) findValleys(values []float64) []float64 {
	valleys := make([]float64, 0)

	for i := 1; i < len(values)-1; i++ {
		if values[i] < values[i-1] && values[i] < values[i+1] {
			valleys = append(valleys, values[i])
		}
	}

	return valleys
}

func (m *MouseBehaviorExtractor) calculatePositiveRatio(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	positiveCount := 0
	for _, v := range values {
		if v > 0 {
			positiveCount++
		}
	}

	return float64(positiveCount) / float64(len(values))
}

func (m *MouseBehaviorExtractor) countValues(values []float64, target float64) int {
	count := 0
	for _, v := range values {
		if math.Abs(v-target) < 0.001 {
			count++
		}
	}
	return count
}

func (m *MouseBehaviorExtractor) countBelow(values []float64, threshold float64) int {
	count := 0
	for _, v := range values {
		if v < threshold {
			count++
		}
	}
	return count
}

func (m *MouseBehaviorExtractor) countAbove(values []float64, threshold float64) int {
	count := 0
	for _, v := range values {
		if v > threshold {
			count++
		}
	}
	return count
}

func (m *MouseBehaviorExtractor) findOutliers(values []float64) []int {
	outliers := make([]int, 0)

	if len(values) < 3 {
		return outliers
	}

	mean := m.mean(values)
	stdDev := math.Sqrt(m.variance(values))

	for i, v := range values {
		if math.Abs(v-mean) > 2*stdDev {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

func (m *MouseBehaviorExtractor) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (m *MouseBehaviorExtractor) median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func (m *MouseBehaviorExtractor) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := m.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

func (m *MouseBehaviorExtractor) max(values []float64) float64 {
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

func (m *MouseBehaviorExtractor) min(values []float64) float64 {
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

func (m *MouseBehaviorExtractor) skewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	mean := m.mean(values)
	stdDev := math.Sqrt(m.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func (m *MouseBehaviorExtractor) kurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}
	mean := m.mean(values)
	stdDev := math.Sqrt(m.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 4)
	}
	return (sum / float64(len(values))) - 3
}

func (m *MouseBehaviorExtractor) AnalyzeHumanLikelihood(features *model.MouseBehaviorFeatures) (bool, float64) {
	if features == nil {
		return false, 0.0
	}

	likelihood := 100.0

	if features.SpeedFeatures.SpeedStdDev/features.SpeedFeatures.AverageSpeed < 0.1 {
		likelihood -= 20
	}

	if features.AccelerationFeatures.AccelerationStdDev < 0.01 {
		likelihood -= 15
	}

	if features.DoubleClickFeatures.FastDoubleClickRatio > 0.8 {
		likelihood -= 15
	}

	if features.LatencyFeatures.FastClickRatio > 0.9 {
		likelihood -= 20
	}

	if features.LatencyFeatures.HesitationCount == 0 {
		likelihood -= 10
	}

	likelihood = math.Max(0, likelihood)
	isHuman := likelihood >= 50

	return isHuman, likelihood
}

func (m *MouseBehaviorExtractor) GenerateReport(features *model.MouseBehaviorFeatures) string {
	if features == nil {
		return "无特征数据"
	}

	report := "=== 鼠标行为特征分析报告 ===\n\n"

	report += "速度特征:\n"
	report += fmt.Sprintf("  平均速度: %.6f\n", features.SpeedFeatures.AverageSpeed)
	report += fmt.Sprintf("  最大速度: %.6f\n", features.SpeedFeatures.MaxSpeed)
	report += fmt.Sprintf("  速度标准差: %.6f\n", features.SpeedFeatures.SpeedStdDev)
	report += fmt.Sprintf("  速度方差: %.6f\n", features.SpeedFeatures.SpeedVariance)
	report += fmt.Sprintf("  零速度计数: %d\n", features.SpeedFeatures.ZeroSpeedCount)

	report += "\n加速度特征:\n"
	report += fmt.Sprintf("  平均加速度: %.6f\n", features.AccelerationFeatures.AverageAcceleration)
	report += fmt.Sprintf("  最大加速度: %.6f\n", features.AccelerationFeatures.MaxAcceleration)
	report += fmt.Sprintf("  加加速度均值: %.6f\n", features.AccelerationFeatures.JerkAvg)
	report += fmt.Sprintf("  方向变化次数: %d\n", features.AccelerationFeatures.DirectionChanges)

	report += "\n点击分布特征:\n"
	report += fmt.Sprintf("  X均值: %.2f, Y均值: %.2f\n", features.DistributionFeatures.XMean, features.DistributionFeatures.YMean)
	report += fmt.Sprintf("  X标准差: %.2f, Y标准差: %.2f\n", features.DistributionFeatures.XStdDev, features.DistributionFeatures.YStdDev)
	report += fmt.Sprintf("  分布熵: X=%.4f, Y=%.4f\n", features.DistributionFeatures.XEntropy, features.DistributionFeatures.YEntropy)
	report += fmt.Sprintf("  聚类数: %d\n", features.DistributionFeatures.Clusters)

	report += "\n双击特征:\n"
	report += fmt.Sprintf("  双击次数: %d\n", features.DoubleClickFeatures.DoubleClickCount)
	report += fmt.Sprintf("  平均间隔: %.2fms\n", features.DoubleClickFeatures.AverageInterval)
	report += fmt.Sprintf("  快速双击比例: %.2f%%\n", features.DoubleClickFeatures.FastDoubleClickRatio*100)

	report += "\n点击延迟特征:\n"
	report += fmt.Sprintf("  平均延迟: %.2fms\n", features.LatencyFeatures.AverageLatency)
	report += fmt.Sprintf("  中位延迟: %.2fms\n", features.LatencyFeatures.MedianLatency)
	report += fmt.Sprintf("  犹豫次数: %d\n", features.LatencyFeatures.HesitationCount)
	report += fmt.Sprintf("  快速点击比例: %.2f%%\n", features.LatencyFeatures.FastClickRatio*100)

	report += "\n综合评估:\n"
	report += fmt.Sprintf("  风险评分: %.2f\n", features.OverallScore)
	report += fmt.Sprintf("  是否为人类: %v\n", features.IsHumanLike)
	report += fmt.Sprintf("  置信度: %.2f%%\n", features.Confidence*100)

	if len(features.AnomalyIndicators) > 0 {
		report += "\n异常指标:\n"
		for _, indicator := range features.AnomalyIndicators {
			report += fmt.Sprintf("  - %s\n", indicator)
		}
	}

	return report
}
