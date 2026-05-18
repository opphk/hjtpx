package service

import (
	"math"
	"sort"
)

type SpeedProfile struct {
	AverageSpeed       float64
	MedianSpeed        float64
	MaxSpeed           float64
	MinSpeed           float64
	SpeedVariance      float64
	SpeedStdDev        float64
	SpeedSkewness      float64
	SpeedKurtosis      float64
	SpeedRange         float64
	SpeedCV            float64
	AccelerationAvg    float64
	DecelerationAvg    float64
	AccelerationMax    float64
	DecelerationMax    float64
	AccelerationVariance float64
	JerkAvg            float64
	JerkMax            float64
	JerkVariance       float64
	IsSpeedConsistent  bool
	IsSpeedNormal      bool
	SpeedOutliers      int
	ZeroSpeedCount     int
}

type CurvatureAnalysis struct {
	AverageCurvature   float64
	MaxCurvature       float64
	MinCurvature       float64
	CurvatureVariance  float64
	CurvatureEntropy   float64
	PeakCount          int
	InflectionPoints   int
	SharpTurnCount     int
	CurvatureSkewness  float64
	CurvatureKurtosis  float64
}

type BacktrackAnalysis struct {
	Count              int
	TotalDistance      float64
	MaxBacktrackDist   float64
	AvgBacktrackDist   float64
	BacktrackDuration  int64
	IsBacktrackNormal  bool
	BacktrackPattern   []int
}

type SmoothnessAnalysis struct {
	OverallScore        float64
	JitterScore         float64
	AngularChangeAvg    float64
	AngularChangeStd    float64
	DirectionStability  float64
	PathRegularity      float64
	IsTrajectorySmooth  bool
}

type TrajectoryOptimizer struct{}

func NewTrajectoryOptimizer() *TrajectoryOptimizer {
	return &TrajectoryOptimizer{}
}

func (to *TrajectoryOptimizer) AnalyzeSpeedProfile(points []SliderPoint) SpeedProfile {
	profile := SpeedProfile{}

	if len(points) < 2 {
		return profile
	}

	speeds := to.extractSpeedsWithProfile(points)
	if len(speeds) == 0 {
		return profile
	}

	profile.AverageSpeed = to.mean(speeds)
	profile.MedianSpeed = to.median(speeds)
	profile.MaxSpeed = to.max(speeds)
	profile.MinSpeed = to.min(speeds)
	profile.SpeedRange = profile.MaxSpeed - profile.MinSpeed
	profile.SpeedVariance = to.variance(speeds)
	profile.SpeedStdDev = math.Sqrt(profile.SpeedVariance)

	if profile.AverageSpeed > 0 {
		profile.SpeedCV = profile.SpeedStdDev / profile.AverageSpeed
	}

	profile.SpeedSkewness = to.calculateSkewness(speeds)
	profile.SpeedKurtosis = to.calculateKurtosis(speeds)

	accelerations := to.extractAccelerations(points)
	if len(accelerations) > 0 {
		positiveAccel := make([]float64, 0)
		negativeAccel := make([]float64, 0)

		for _, acc := range accelerations {
			if acc > 0 {
				positiveAccel = append(positiveAccel, acc)
			} else {
				negativeAccel = append(negativeAccel, -acc)
			}
		}

		profile.AccelerationAvg = to.mean(positiveAccel)
		profile.DecelerationAvg = to.mean(negativeAccel)
		profile.AccelerationMax = to.max(positiveAccel)
		profile.DecelerationMax = to.max(negativeAccel)
		profile.AccelerationVariance = to.variance(accelerations)
	}

	jerks := to.extractJerks(points)
	if len(jerks) > 0 {
		profile.JerkAvg = to.mean(jerks)
		profile.JerkMax = to.maxAbs(jerks)
		profile.JerkVariance = to.variance(jerks)
	}

	profile.IsSpeedConsistent = profile.SpeedCV < 0.3
	profile.IsSpeedNormal = profile.AverageSpeed > 50 && profile.AverageSpeed < 2000
	profile.SpeedOutliers = to.countSpeedOutliers(speeds, profile.AverageSpeed, profile.SpeedStdDev)
	profile.ZeroSpeedCount = to.countZeroSpeedSegments(speeds)

	return profile
}

func (to *TrajectoryOptimizer) extractSpeedsWithProfile(points []SliderPoint) []float64 {
	speeds := make([]float64, 0, len(points)-1)

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)

		if dt > 0 && dt < 1000 {
			speed := distance / dt * 1000
			speeds = append(speeds, speed)
		}
	}

	return speeds
}

func (to *TrajectoryOptimizer) extractAccelerations(points []SliderPoint) []float64 {
	speeds := to.extractSpeedsWithProfile(points)
	accelerations := make([]float64, 0, len(speeds)-1)

	for i := 1; i < len(speeds) && i < len(points); i++ {
		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt * 1000
			accelerations = append(accelerations, accel)
		}
	}

	return accelerations
}

func (to *TrajectoryOptimizer) extractJerks(points []SliderPoint) []float64 {
	accelerations := to.extractAccelerations(points)
	jerks := make([]float64, 0, len(accelerations)-1)

	for i := 1; i < len(accelerations) && i+1 < len(points); i++ {
		dt := float64(points[i+2].Timestamp - points[i].Timestamp)
		if dt > 0 {
			jerk := (accelerations[i] - accelerations[i-1]) / dt * 1000
			jerks = append(jerks, jerk)
		}
	}

	return jerks
}

func (to *TrajectoryOptimizer) AnalyzeCurvature(points []SliderPoint) CurvatureAnalysis {
	analysis := CurvatureAnalysis{}

	if len(points) < 3 {
		return analysis
	}

	curvatures := to.extractCurvaturesAdvanced(points)
	if len(curvatures) == 0 {
		return analysis
	}

	analysis.AverageCurvature = to.mean(curvatures)
	analysis.MaxCurvature = to.max(curvatures)
	analysis.MinCurvature = to.min(curvatures)
	analysis.CurvatureVariance = to.variance(curvatures)
	analysis.CurvatureEntropy = to.calculateCurvatureEntropy(curvatures)
	analysis.PeakCount = to.countCurvaturePeaks(curvatures)
	analysis.InflectionPoints = to.countInflectionPoints(curvatures)
	analysis.SharpTurnCount = to.countSharpTurns(curvatures)
	analysis.CurvatureSkewness = to.calculateSkewness(curvatures)
	analysis.CurvatureKurtosis = to.calculateKurtosis(curvatures)

	return analysis
}

func (to *TrajectoryOptimizer) extractCurvaturesAdvanced(points []SliderPoint) []float64 {
	curvatures := make([]float64, 0, len(points)-2)

	for i := 1; i < len(points)-1; i++ {
		curv := to.computeCurvature(points[i-1], points[i], points[i+1])
		curvatures = append(curvatures, math.Abs(curv))
	}

	return curvatures
}

func (to *TrajectoryOptimizer) computeCurvature(p1, p2, p3 SliderPoint) float64 {
	v1x := float64(p2.X - p1.X)
	v1y := float64(p2.Y - p1.Y)
	v2x := float64(p3.X - p2.X)
	v2y := float64(p3.Y - p2.Y)

	mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
	mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	dot := v1x*v2x + v1y*v2y
	cosAngle := dot / (mag1 * mag2)

	if cosAngle > 1 {
		cosAngle = 1
	}
	if cosAngle < -1 {
		cosAngle = -1
	}

	angle := math.Acos(cosAngle)
	return angle
}

func (to *TrajectoryOptimizer) calculateCurvatureEntropy(curvatures []float64) float64 {
	if len(curvatures) == 0 {
		return 0
	}

	bins := 10
	histogram := make([]int, bins)
	minC, maxC := to.min(curvatures), to.max(curvatures)

	if maxC <= minC {
		return 0
	}

	for _, c := range curvatures {
		bin := int((c - minC) / (maxC - minC) * float64(bins-1))
		if bin >= bins {
			bin = bins - 1
		}
		if bin < 0 {
			bin = 0
		}
		histogram[bin]++
	}

	entropy := 0.0
	total := float64(len(curvatures))
	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / total
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (to *TrajectoryOptimizer) countCurvaturePeaks(curvatures []float64) int {
	if len(curvatures) < 3 {
		return 0
	}

	mean := to.mean(curvatures)
	std := math.Sqrt(to.variance(curvatures))
	threshold := mean + std

	peaks := 0
	for i := 1; i < len(curvatures)-1; i++ {
		if curvatures[i] > curvatures[i-1] && curvatures[i] > curvatures[i+1] && curvatures[i] > threshold {
			peaks++
		}
	}

	return peaks
}

func (to *TrajectoryOptimizer) countInflectionPoints(curvatures []float64) int {
	if len(curvatures) < 3 {
		return 0
	}

	inflections := 0
	for i := 1; i < len(curvatures)-1; i++ {
		prevSign := curvatures[i] - curvatures[i-1]
		nextSign := curvatures[i+1] - curvatures[i]

		if (prevSign > 0 && nextSign < 0) || (prevSign < 0 && nextSign > 0) {
			inflections++
		}
	}

	return inflections
}

func (to *TrajectoryOptimizer) countSharpTurns(curvatures []float64) int {
	sharpTurnThreshold := 0.5
	count := 0

	for _, c := range curvatures {
		if c > sharpTurnThreshold {
			count++
		}
	}

	return count
}

func (to *TrajectoryOptimizer) AnalyzeBacktrack(points []SliderPoint) BacktrackAnalysis {
	analysis := BacktrackAnalysis{}

	if len(points) < 2 {
		return analysis
	}

	maxX := points[0].X
	backtrackPoints := make([]int, 0)

	for i := 1; i < len(points); i++ {
		if points[i].X > maxX {
			maxX = points[i].X
		} else if maxX-points[i].X > 3 {
			backtrackPoints = append(backtrackPoints, i)
			analysis.TotalDistance += float64(maxX - points[i].X)
		}
	}

	analysis.Count = len(backtrackPoints)
	analysis.BacktrackPattern = backtrackPoints

	if analysis.Count > 0 {
		analysis.AvgBacktrackDist = analysis.TotalDistance / float64(analysis.Count)
	}

	if len(backtrackPoints) > 0 {
		maxBacktrack := 0
		for _, idx := range backtrackPoints {
			bt := maxX - points[idx].X
			if bt > maxBacktrack {
				maxBacktrack = bt
			}
		}
		analysis.MaxBacktrackDist = float64(maxBacktrack)
	}

	if len(backtrackPoints) >= 2 {
		analysis.BacktrackDuration = points[backtrackPoints[len(backtrackPoints)-1]].Timestamp -
			points[backtrackPoints[0]].Timestamp
	}

	analysis.IsBacktrackNormal = analysis.Count >= 0 && analysis.Count <= 5

	return analysis
}

func (to *TrajectoryOptimizer) AnalyzeSmoothness(points []SliderPoint) SmoothnessAnalysis {
	analysis := SmoothnessAnalysis{}

	if len(points) < 3 {
		return analysis
	}

	angularChanges := to.extractAngularChanges(points)
	if len(angularChanges) == 0 {
		return analysis
	}

	analysis.AngularChangeAvg = to.mean(angularChanges)
	analysis.AngularChangeStd = math.Sqrt(to.variance(angularChanges))
	analysis.DirectionStability = 1.0 - math.Min(analysis.AngularChangeAvg/math.Pi, 1.0)

	jitter := to.calculateJitterAdvanced(points)
	analysis.JitterScore = jitter

	smoothness := to.calculateTrajectorySmoothness(points)
	analysis.OverallScore = (smoothness + analysis.DirectionStability + (1.0-math.Min(jitter*10, 1.0))) / 3.0

	analysis.PathRegularity = to.calculatePathRegularity(points)

	analysis.IsTrajectorySmooth = analysis.OverallScore > 0.7

	return analysis
}

func (to *TrajectoryOptimizer) extractAngularChanges(points []SliderPoint) []float64 {
	changes := make([]float64, 0, len(points)-2)

	for i := 2; i < len(points); i++ {
		v1x := float64(points[i-1].X - points[i-2].X)
		v1y := float64(points[i-1].Y - points[i-2].Y)
		v2x := float64(points[i].X - points[i-1].X)
		v2y := float64(points[i].Y - points[i-1].Y)

		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			dot := v1x*v2x + v1y*v2y
			cosAngle := dot / (mag1 * mag2)

			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}

			angle := math.Acos(cosAngle)
			changes = append(changes, angle)
		}
	}

	return changes
}

func (to *TrajectoryOptimizer) calculateJitterAdvanced(points []SliderPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	smoothed := to.smoothTrajectory(points, 3)
	totalJitter := 0.0

	for i := 1; i < len(points); i++ {
		dx1 := float64(points[i].X - points[i-1].X)
		dy1 := float64(points[i].Y - points[i-1].Y)
		dx2 := float64(smoothed[i].X - smoothed[i-1].X)
		dy2 := float64(smoothed[i].Y - smoothed[i-1].Y)

		dist1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		dist2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if dist1 > 0 {
			totalJitter += math.Abs(dist1-dist2) / dist1
		}
	}

	return totalJitter / float64(len(points)-1)
}

func (to *TrajectoryOptimizer) smoothTrajectory(points []SliderPoint, windowSize int) []SliderPoint {
	if len(points) < windowSize {
		return points
	}

	if windowSize%2 == 0 {
		windowSize++
	}

	halfWindow := windowSize / 2
	smoothed := make([]SliderPoint, len(points))

	for i := range points {
		start := i - halfWindow
		end := i + halfWindow

		if start < 0 {
			start = 0
		}
		if end >= len(points) {
			end = len(points) - 1
		}

		sumX := 0
		sumY := 0
		count := 0

		for j := start; j <= end; j++ {
			sumX += points[j].X
			sumY += points[j].Y
			count++
		}

		smoothed[i] = points[i]
		smoothed[i].X = sumX / count
		smoothed[i].Y = sumY / count
	}

	return smoothed
}

func (to *TrajectoryOptimizer) calculateTrajectorySmoothness(points []SliderPoint) float64 {
	if len(points) < 3 {
		return 1.0
	}

	totalAngleChange := 0.0
	count := 0

	for i := 1; i < len(points)-1; i++ {
		v1x := float64(points[i].X - points[i-1].X)
		v1y := float64(points[i].Y - points[i-1].Y)
		v2x := float64(points[i+1].X - points[i].X)
		v2y := float64(points[i+1].Y - points[i].Y)

		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			dot := v1x*v2x + v1y*v2y
			cosAngle := dot / (mag1 * mag2)

			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}

			angle := math.Acos(cosAngle)
			totalAngleChange += angle
			count++
		}
	}

	if count == 0 {
		return 1.0
	}

	avgAngleChange := totalAngleChange / float64(count)
	return 1.0 - math.Min(avgAngleChange/math.Pi, 1.0)
}

func (to *TrajectoryOptimizer) calculatePathRegularity(points []SliderPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	speeds := to.extractSpeedsWithProfile(points)
	if len(speeds) < 2 {
		return 0
	}

	variance := to.variance(speeds)
	mean := to.mean(speeds)

	if mean == 0 {
		return 0
	}

	cv := math.Sqrt(variance) / mean
	return 1.0 - math.Min(cv, 1.0)
}

func (to *TrajectoryOptimizer) countSpeedOutliers(speeds []float64, mean, stdDev float64) int {
	if stdDev == 0 {
		return 0
	}

	threshold := 3.0 * stdDev
	outliers := 0

	for _, speed := range speeds {
		if math.Abs(speed-mean) > threshold {
			outliers++
		}
	}

	return outliers
}

func (to *TrajectoryOptimizer) countZeroSpeedSegments(speeds []float64) int {
	count := 0
	for _, speed := range speeds {
		if speed < 1.0 {
			count++
		}
	}
	return count
}

func (to *TrajectoryOptimizer) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (to *TrajectoryOptimizer) median(values []float64) float64 {
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

func (to *TrajectoryOptimizer) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := to.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

func (to *TrajectoryOptimizer) max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (to *TrajectoryOptimizer) min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (to *TrajectoryOptimizer) maxAbs(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := math.Abs(values[0])
	for _, v := range values {
		if math.Abs(v) > max {
			max = math.Abs(v)
		}
	}
	return max
}

func (to *TrajectoryOptimizer) calculateSkewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	mean := to.mean(values)
	stdDev := math.Sqrt(to.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func (to *TrajectoryOptimizer) calculateKurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}
	mean := to.mean(values)
	stdDev := math.Sqrt(to.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 4)
	}
	return (sum / float64(len(values))) - 3
}
