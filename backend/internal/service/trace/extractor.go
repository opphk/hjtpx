package trace

import (
	"errors"
	"math"
	"sort"

	"github.com/hjtpx/hjtpx/internal/model"
)

type TraceExtractor struct{}

func NewTraceExtractor() *TraceExtractor {
	return &TraceExtractor{}
}

func (e *TraceExtractor) ExtractFeatures(traceData *model.TraceData) (*model.TraceFeatures, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据点不足")
	}

	features := &model.TraceFeatures{
		TotalTime:   traceData.TotalTime,
		MoveCount:   e.countMoves(traceData),
		RiskFactors: []string{},
	}

	features.AvgSpeed = e.calculateAvgSpeed(traceData)
	features.MaxSpeed = e.calculateMaxSpeed(traceData)
	features.MinSpeed = e.calculateMinSpeed(traceData)
	features.SpeedVariance = e.calculateSpeedVariance(traceData)
	features.MaxAcceleration = e.calculateMaxAcceleration(traceData)
	features.AvgAcceleration = e.calculateAvgAcceleration(traceData)
	features.AccelVariance = e.calculateAccelVariance(traceData)
	features.Smoothness = e.calculateSmoothness(traceData)
	features.PauseCount = e.calculatePauseCount(traceData)
	features.TotalDistance = e.calculateTotalDistance(traceData)
	features.DirectDistance = e.calculateDirectDistance(traceData)
	features.AvgCurvature = e.calculateAvgCurvature(traceData)
	features.MaxCurvature = e.calculateMaxCurvature(traceData)
	features.JitterFrequency = e.calculateJitterFrequency(traceData)
	features.JitterAmplitude = e.calculateJitterAmplitude(traceData)
	features.SpeedChangeRate = e.calculateSpeedChangeRate(traceData)
	features.DirectionChange = e.calculateDirectionChange(traceData)

	if features.DirectDistance > 0 {
		features.PathRatio = features.TotalDistance / features.DirectDistance
	} else {
		features.PathRatio = 1.0
	}

	features.RiskFactors = e.detectRiskFactors(features)

	return features, nil
}

func (e *TraceExtractor) countMoves(traceData *model.TraceData) int {
	count := 0
	for _, p := range traceData.Points {
		if p.Event == "move" {
			count++
		}
	}
	return count
}

func (e *TraceExtractor) calculateAvgSpeed(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	var totalSpeed float64
	speedCount := 0

	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(curr.Timestamp-prev.Timestamp) / 1000.0

		if time > 0 && time < 1000 {
			speed := distance / time
			totalSpeed += speed
			speedCount++
		}
	}

	if speedCount == 0 {
		return 0
	}

	return totalSpeed / float64(speedCount)
}

func (e *TraceExtractor) calculateMaxSpeed(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	var maxSpeed float64

	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(curr.Timestamp-prev.Timestamp) / 1000.0

		if time > 0 && time < 1000 {
			speed := distance / time
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
	}

	return maxSpeed
}

func (e *TraceExtractor) calculateMinSpeed(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	var minSpeed float64 = math.MaxFloat64

	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(curr.Timestamp-prev.Timestamp) / 1000.0

		if time > 0 && time < 1000 && distance > 0 {
			speed := distance / time
			if speed < minSpeed {
				minSpeed = speed
			}
		}
	}

	if minSpeed == math.MaxFloat64 {
		return 0
	}

	return minSpeed
}

func (e *TraceExtractor) getAllSpeeds(traceData *model.TraceData) []float64 {
	var speeds []float64

	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(curr.Timestamp-prev.Timestamp) / 1000.0

		if time > 0 && time < 1000 {
			speed := distance / time
			speeds = append(speeds, speed)
		}
	}

	return speeds
}

func (e *TraceExtractor) calculateSpeedVariance(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	speeds := e.getAllSpeeds(traceData)
	if len(speeds) < 2 {
		return 0
	}

	avgSpeed := e.calculateAvgSpeed(traceData)

	var variance float64
	for _, speed := range speeds {
		diff := speed - avgSpeed
		variance += diff * diff
	}

	return variance / float64(len(speeds))
}

func (e *TraceExtractor) calculateMaxAcceleration(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	var maxAccel float64

	for i := 2; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-2]
		curr := traceData.Points[i-1]
		next := traceData.Points[i]

		dx1 := curr.X - prev.X
		dy1 := curr.Y - prev.Y
		v1 := math.Sqrt(dx1*dx1 + dy1*dy1)

		dx2 := next.X - curr.X
		dy2 := next.Y - curr.Y
		v2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		time := float64(next.Timestamp-prev.Timestamp) / 1000.0
		if time > 0 && time < 1000 {
			accel := math.Abs(v2-v1) / time
			if accel > maxAccel {
				maxAccel = accel
			}
		}
	}

	return maxAccel
}

func (e *TraceExtractor) getAllAccelerations(traceData *model.TraceData) []float64 {
	var accelerations []float64

	if len(traceData.Points) < 3 {
		return accelerations
	}

	for i := 2; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-2]
		curr := traceData.Points[i-1]
		next := traceData.Points[i]

		dx1 := curr.X - prev.X
		dy1 := curr.Y - prev.Y
		v1 := math.Sqrt(dx1*dx1 + dy1*dy1)

		dx2 := next.X - curr.X
		dy2 := next.Y - curr.Y
		v2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		time := float64(next.Timestamp-prev.Timestamp) / 1000.0
		if time > 0 && time < 1000 {
			accel := math.Abs(v2-v1) / time
			accelerations = append(accelerations, accel)
		}
	}

	return accelerations
}

func (e *TraceExtractor) calculateAvgAcceleration(traceData *model.TraceData) float64 {
	accelerations := e.getAllAccelerations(traceData)
	if len(accelerations) == 0 {
		return 0
	}

	var sum float64
	for _, accel := range accelerations {
		sum += accel
	}

	return sum / float64(len(accelerations))
}

func (e *TraceExtractor) calculateAccelVariance(traceData *model.TraceData) float64 {
	accelerations := e.getAllAccelerations(traceData)
	if len(accelerations) < 2 {
		return 0
	}

	avgAccel := e.calculateAvgAcceleration(traceData)

	var variance float64
	for _, accel := range accelerations {
		diff := accel - avgAccel
		variance += diff * diff
	}

	return variance / float64(len(accelerations))
}

func (e *TraceExtractor) calculateAvgCurvature(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	var totalCurvature float64
	count := 0

	for i := 1; i < len(traceData.Points)-1; i++ {
		p0 := traceData.Points[i-1]
		p1 := traceData.Points[i]
		p2 := traceData.Points[i+1]

		ax := p1.X - p0.X
		ay := p1.Y - p0.Y
		bx := p2.X - p1.X
		by := p2.Y - p1.Y

		cross := ax*by - ay*bx
		lenA := math.Sqrt(ax*ax + ay*ay)
		lenB := math.Sqrt(bx*bx + by*by)

		if lenA > 0 && lenB > 0 {
			curvature := math.Abs(cross) / (lenA * lenB * (1 + (ax*bx+ay*by)/(lenA*lenB)))
			totalCurvature += curvature
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalCurvature / float64(count)
}

func (e *TraceExtractor) calculateMaxCurvature(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	var maxCurvature float64

	for i := 1; i < len(traceData.Points)-1; i++ {
		p0 := traceData.Points[i-1]
		p1 := traceData.Points[i]
		p2 := traceData.Points[i+1]

		ax := p1.X - p0.X
		ay := p1.Y - p0.Y
		bx := p2.X - p1.X
		by := p2.Y - p1.Y

		cross := ax*by - ay*bx
		lenA := math.Sqrt(ax*ax + ay*ay)
		lenB := math.Sqrt(bx*bx + by*by)

		if lenA > 0 && lenB > 0 {
			curvature := math.Abs(cross) / (lenA * lenB * (1 + (ax*bx+ay*by)/(lenA*lenB)))
			if curvature > maxCurvature {
				maxCurvature = curvature
			}
		}
	}

	return maxCurvature
}

func (e *TraceExtractor) calculateJitterFrequency(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 4 {
		return 0
	}

	jitterCount := 0
	threshold := 0.5

	for i := 2; i < len(traceData.Points)-1; i++ {
		p0 := traceData.Points[i-2]
		p1 := traceData.Points[i-1]
		p2 := traceData.Points[i]
		p3 := traceData.Points[i+1]

		dx1 := p1.X - p0.X
		dy1 := p1.Y - p0.Y
		dx2 := p2.X - p1.X
		dy2 := p2.Y - p1.Y
		dx3 := p3.X - p2.X
		dy3 := p3.Y - p2.Y

		dir1 := math.Atan2(dy1, dx1)
		dir2 := math.Atan2(dy2, dx2)
		dir3 := math.Atan2(dy3, dx3)

		change1 := math.Abs(dir2 - dir1)
		change2 := math.Abs(dir3 - dir2)

		if change1 > threshold && change2 > threshold && change1+change2 > math.Pi {
			jitterCount++
		}
	}

	return float64(jitterCount) / float64(len(traceData.Points)-3)
}

func (e *TraceExtractor) calculateJitterAmplitude(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	var totalJitter float64
	count := 0

	for i := 1; i < len(traceData.Points)-1; i++ {
		p0 := traceData.Points[i-1]
		p1 := traceData.Points[i]
		p2 := traceData.Points[i+1]

		// 计算中点
		midX := (p0.X + p2.X) / 2
		midY := (p0.Y + p2.Y) / 2

		// 计算偏离量
		dx := p1.X - midX
		dy := p1.Y - midY
		jitter := math.Sqrt(dx*dx + dy*dy)

		totalJitter += jitter
		count++
	}

	if count == 0 {
		return 0
	}

	return totalJitter / float64(count)
}

func (e *TraceExtractor) calculateSpeedChangeRate(traceData *model.TraceData) float64 {
	speeds := e.getAllSpeeds(traceData)
	if len(speeds) < 2 {
		return 0
	}

	var totalChange float64
	for i := 1; i < len(speeds); i++ {
		totalChange += math.Abs(speeds[i] - speeds[i-1])
	}

	return totalChange / float64(len(speeds)-1)
}

func (e *TraceExtractor) calculateDirectionChange(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	var totalChange float64
	count := 0

	for i := 2; i < len(traceData.Points); i++ {
		p0 := traceData.Points[i-2]
		p1 := traceData.Points[i-1]
		p2 := traceData.Points[i]

		dir1 := math.Atan2(p1.Y-p0.Y, p1.X-p0.X)
		dir2 := math.Atan2(p2.Y-p1.Y, p2.X-p1.X)

		change := math.Abs(dir2 - dir1)
		if change > math.Pi {
			change = 2*math.Pi - change
		}

		totalChange += change
		count++
	}

	if count == 0 {
		return 0
	}

	return totalChange / float64(count)
}

func (e *TraceExtractor) calculateSmoothness(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	var totalAngleChange float64
	angleChanges := 0

	for i := 2; i < len(traceData.Points); i++ {
		p1 := traceData.Points[i-2]
		p2 := traceData.Points[i-1]
		p3 := traceData.Points[i]

		angle1 := math.Atan2(p2.Y-p1.Y, p2.X-p1.X)
		angle2 := math.Atan2(p3.Y-p2.Y, p3.X-p2.X)

		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}

		totalAngleChange += angleDiff
		angleChanges++
	}

	if angleChanges > 0 {
		return totalAngleChange / float64(angleChanges)
	}

	return 0
}

func (e *TraceExtractor) calculatePauseCount(traceData *model.TraceData) int {
	if len(traceData.Points) < 2 {
		return 0
	}

	pauseCount := 0
	const pauseThresholdMs = 200

	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]

		timeDiff := curr.Timestamp - prev.Timestamp

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)

		if timeDiff > pauseThresholdMs && distance < 2 {
			pauseCount++
		}
	}

	return pauseCount
}

func (e *TraceExtractor) calculateTotalDistance(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	var totalDistance float64

	for i := 1; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-1]
		curr := traceData.Points[i]

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(dx*dx + dy*dy)

		totalDistance += distance
	}

	return totalDistance
}

func (e *TraceExtractor) calculateDirectDistance(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	first := traceData.Points[0]
	last := traceData.Points[len(traceData.Points)-1]

	dx := last.X - first.X
	dy := last.Y - first.Y

	return math.Sqrt(dx*dx + dy*dy)
}

func (e *TraceExtractor) detectRiskFactors(features *model.TraceFeatures) []string {
	riskFactors := []string{}

	if features.SpeedVariance < 10 && features.TotalTime > 1000 {
		riskFactors = append(riskFactors, "速度方差过小")
	}

	if features.AvgSpeed > 1000 {
		riskFactors = append(riskFactors, "平均速度过快")
	}

	if features.AvgSpeed < 10 && features.TotalTime > 5000 {
		riskFactors = append(riskFactors, "平均速度过慢")
	}

	if features.PauseCount == 0 && features.TotalTime > 2000 {
		riskFactors = append(riskFactors, "无正常停顿行为")
	}

	if features.PathRatio < 1.1 {
		riskFactors = append(riskFactors, "轨迹过于直线")
	}

	if features.MaxAcceleration > 5000 {
		riskFactors = append(riskFactors, "加速度异常")
	}

	if features.Smoothness < 0.05 {
		riskFactors = append(riskFactors, "轨迹平滑度异常")
	}

	if features.TotalDistance < 10 && features.TotalTime > 1000 {
		riskFactors = append(riskFactors, "移动距离过短")
	}

	if features.MoveCount == 0 {
		riskFactors = append(riskFactors, "无移动轨迹")
	}

	if features.AccelVariance > 100000 {
		riskFactors = append(riskFactors, "加速度方差异常")
	}

	if features.AvgAcceleration > 2000 {
		riskFactors = append(riskFactors, "平均加速度过高")
	}

	if features.AvgCurvature < 0.01 && features.TotalTime > 2000 {
		riskFactors = append(riskFactors, "曲率过于平滑")
	}

	if features.MaxCurvature > 1.0 {
		riskFactors = append(riskFactors, "最大曲率异常")
	}

	if features.JitterFrequency > 0.5 {
		riskFactors = append(riskFactors, "抖动频率过高")
	}

	if features.JitterAmplitude > 10 {
		riskFactors = append(riskFactors, "抖动幅度异常")
	}

	if features.SpeedChangeRate > 500 {
		riskFactors = append(riskFactors, "速度变化率异常")
	}

	if features.DirectionChange > math.Pi/2 {
		riskFactors = append(riskFactors, "方向变化过于频繁")
	}

	return riskFactors
}

type AdvancedFeatures struct {
	MedianSpeed              float64
	SpeedSkewness           float64
	SpeedKurtosis           float64
	SpeedEntropy            float64
	SpeedVariance           float64
	MedianAcceleration      float64
	AccelerationVariance    float64
	AccelerationSkewness    float64
	JerkMean                float64
	JerkMax                 float64
	JerkVariance            float64
	CurvatureMedian         float64
	CurvatureVariance       float64
	CurvatureMax            float64
	DirectionChangeRate     float64
	DirectionEntropy        float64
	Sinuosity               float64
	StartEndAngle           float64
	AreaUnderCurve          float64
	TimeNormalizedDistance  float64
	VelocityProfileEntropy  float64
	AccelerationProfileEntropy float64
	HurstExponent           float64
	FractalDimension        float64
	SpectralEntropy         float64
	PermutationEntropy      float64
	ApproximateEntropy      float64
	SampleEntropy           float64
	CircularVariance        float64
	LyapunovExponent        float64
	CorrelationDimension    float64
	KurtosisExcess          float64
	VelocityPeakCount       int
	AccelerationPeakCount   int
	AutocorrelationLag1     float64
	AutocorrelationLag2     float64
	ZeroCrossingRate        float64
	TemporalIrregularity    float64
	SpatialDispersion       float64
	DirectionalConsistency  float64
}

func (e *TraceExtractor) ExtractAdvancedFeatures(traceData *model.TraceData) (*AdvancedFeatures, error) {
	if traceData == nil || len(traceData.Points) < 3 {
		return nil, errors.New("轨迹数据点不足")
	}

	features := &AdvancedFeatures{}

	speeds := e.getAllSpeeds(traceData)
	if len(speeds) > 0 {
		features.MedianSpeed = e.medianFloat(speeds)
		features.SpeedVariance = e.varianceFloat(speeds)
		features.SpeedSkewness = e.calculateSkewness(speeds)
		features.SpeedKurtosis = e.calculateKurtosis(speeds)
		features.SpeedEntropy = e.calculateEntropy(speeds, 10)
		features.HurstExponent = e.calculateHurstExponent(speeds)
		features.SpectralEntropy = e.calculateSpectralEntropy(speeds)
		features.PermutationEntropy = e.calculatePermutationEntropy(speeds)
		features.ApproximateEntropy = e.calculateApproximateEntropy(speeds)
		features.SampleEntropy = e.calculateSampleEntropy(speeds)
		features.VelocityPeakCount = e.countPeaks(speeds)
		features.AutocorrelationLag1 = e.calculateAutocorrelation(speeds, 1)
		features.AutocorrelationLag2 = e.calculateAutocorrelation(speeds, 2)
		features.ZeroCrossingRate = e.calculateZeroCrossingRate(speeds)
		features.KurtosisExcess = features.SpeedKurtosis - 3.0
	}

	accelerations := e.computeAccelerations(traceData)
	if len(accelerations) > 0 {
		features.MedianAcceleration = e.medianFloat(accelerations)
		features.AccelerationVariance = e.varianceFloat(accelerations)
		features.AccelerationSkewness = e.calculateSkewness(accelerations)
		features.AccelerationPeakCount = e.countPeaks(accelerations)
	}

	jerks := e.computeJerks(traceData)
	if len(jerks) > 0 {
		features.JerkMean = e.meanFloat(jerks)
		features.JerkMax = e.maxAbsFloat(jerks)
		features.JerkVariance = e.varianceFloat(jerks)
	}

	curvatures := e.computeCurvatures(traceData)
	if len(curvatures) > 0 {
		features.CurvatureMedian = e.medianFloat(curvatures)
		features.CurvatureVariance = e.varianceFloat(curvatures)
		features.CurvatureMax = e.maxAbsFloat(curvatures)
	}

	directions := e.computeDirections(traceData)
	if len(directions) > 1 {
		features.DirectionChangeRate = e.calculateDirectionChangeRate(directions)
		features.DirectionEntropy = e.calculateDirectionEntropy(directions)
		features.CircularVariance = e.calculateCircularVariance(directions)
		features.DirectionalConsistency = e.calculateDirectionalConsistency(directions)
	}

	if len(traceData.Points) > 1 {
		features.Sinuosity = e.calculateSinuosity(traceData)
		features.StartEndAngle = e.calculateStartEndAngle(traceData)
		features.AreaUnderCurve = e.calculateAreaUnderCurve(traceData)
		features.TimeNormalizedDistance = e.calculateTimeNormalizedDistance(traceData)
		features.FractalDimension = e.calculateFractalDimension(traceData)
		features.CorrelationDimension = e.calculateCorrelationDimension(traceData)
		features.SpatialDispersion = e.calculateSpatialDispersion(traceData)
		features.TemporalIrregularity = e.calculateTemporalIrregularity(traceData)
	}

	features.VelocityProfileEntropy = e.calculateProfileEntropy(speeds, 5)
	features.AccelerationProfileEntropy = e.calculateProfileEntropy(accelerations, 5)

	features.LyapunovExponent = e.calculateLyapunovExponent(speeds)

	return features, nil
}

func (e *TraceExtractor) computeAccelerations(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 3 {
		return nil
	}

	accelerations := make([]float64, 0, len(traceData.Points)-2)

	for i := 2; i < len(traceData.Points); i++ {
		prev := traceData.Points[i-2]
		curr := traceData.Points[i-1]
		next := traceData.Points[i]

		dx1 := float64(curr.X - prev.X)
		dy1 := float64(curr.Y - prev.Y)
		v1 := math.Sqrt(dx1*dx1 + dy1*dy1)

		dx2 := float64(next.X - curr.X)
		dy2 := float64(next.Y - curr.Y)
		v2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		time := float64(next.Timestamp-prev.Timestamp) / 1000.0
		if time > 0 && time < 1000 {
			accel := (v2 - v1) / time
			accelerations = append(accelerations, accel)
		}
	}

	return accelerations
}

func (e *TraceExtractor) computeJerks(traceData *model.TraceData) []float64 {
	accelerations := e.computeAccelerations(traceData)
	if len(accelerations) < 2 || len(traceData.Points) < 4 {
		return nil
	}

	jerks := make([]float64, 0, len(accelerations)-1)

	for i := 2; i < len(traceData.Points); i++ {
		if i >= len(accelerations) {
			break
		}
		time := float64(traceData.Points[i].Timestamp-traceData.Points[i-2].Timestamp) / 1000.0
		if time > 0 && i-1 >= 0 && i-1 < len(accelerations) {
			jerk := (accelerations[i] - accelerations[i-1]) / time
			jerks = append(jerks, jerk)
		}
	}

	return jerks
}

func (e *TraceExtractor) computeCurvatures(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 3 {
		return nil
	}

	curvatures := make([]float64, 0, len(traceData.Points)-2)

	for i := 1; i < len(traceData.Points)-1; i++ {
		p1 := traceData.Points[i-1]
		p2 := traceData.Points[i]
		p3 := traceData.Points[i+1]

		v1x := float64(p2.X - p1.X)
		v1y := float64(p2.Y - p1.Y)
		v2x := float64(p3.X - p2.X)
		v2y := float64(p3.Y - p2.Y)

		dot := v1x*v2x + v1y*v2y
		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			curvatures = append(curvatures, math.Abs(angle))
		}
	}

	return curvatures
}

func (e *TraceExtractor) computeDirections(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 2 {
		return nil
	}

	directions := make([]float64, 0, len(traceData.Points)-1)

	for i := 1; i < len(traceData.Points); i++ {
		dx := float64(traceData.Points[i].X - traceData.Points[i-1].X)
		dy := float64(traceData.Points[i].Y - traceData.Points[i-1].Y)
		angle := math.Atan2(dy, dx)
		directions = append(directions, angle)
	}

	return directions
}

func (e *TraceExtractor) calculateDirectionChangeRate(directions []float64) float64 {
	if len(directions) < 2 {
		return 0
	}

	changes := 0
	for i := 1; i < len(directions); i++ {
		diff := math.Abs(directions[i] - directions[i-1])
		if diff > math.Pi {
			diff = 2*math.Pi - diff
		}
		if diff > 0.5 {
			changes++
		}
	}

	return float64(changes) / float64(len(directions))
}

func (e *TraceExtractor) calculateDirectionEntropy(directions []float64) float64 {
	if len(directions) < 4 {
		return 0
	}

	buckets := 8
	bucketSize := 2 * math.Pi / float64(buckets)

	counts := make([]int, buckets)
	for _, dir := range directions {
		normalized := dir
		if normalized < 0 {
			normalized += 2 * math.Pi
		}
		bucket := int(normalized / bucketSize)
		if bucket >= buckets {
			bucket = buckets - 1
		}
		counts[bucket]++
	}

	total := len(directions)
	entropy := 0.0
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *TraceExtractor) calculateSinuosity(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 1.0
	}

	totalDistance := e.calculateTotalDistance(traceData)
	directDistance := e.calculateDirectDistance(traceData)

	if directDistance == 0 {
		return 1.0
	}

	return totalDistance / directDistance
}

func (e *TraceExtractor) calculateStartEndAngle(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	first := traceData.Points[0]
	last := traceData.Points[len(traceData.Points)-1]

	dx := float64(last.X - first.X)
	dy := float64(last.Y - first.Y)

	return math.Atan2(dy, dx)
}

func (e *TraceExtractor) calculateAreaUnderCurve(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	area := 0.0
	for i := 1; i < len(traceData.Points); i++ {
		y1 := float64(traceData.Points[i-1].Y)
		y2 := float64(traceData.Points[i].Y)
		avgY := (y1 + y2) / 2

		dx := float64(traceData.Points[i].X - traceData.Points[i-1].X)

		area += avgY * dx
	}

	return math.Abs(area)
}

func (e *TraceExtractor) calculateTimeNormalizedDistance(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	totalDistance := e.calculateTotalDistance(traceData)
	totalTime := float64(traceData.Points[len(traceData.Points)-1].Timestamp-traceData.Points[0].Timestamp) / 1000.0

	if totalTime == 0 {
		return 0
	}

	return totalDistance / totalTime
}

func (e *TraceExtractor) calculateEntropy(values []float64, bucketCount int) float64 {
	if len(values) < 2 || bucketCount < 2 {
		return 0
	}

	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	rangeVal := maxVal - minVal
	if rangeVal < 0.001 {
		return 0
	}

	bucketSize := rangeVal / float64(bucketCount)
	if bucketSize == 0 {
		return 0
	}

	counts := make([]int, bucketCount)
	for _, v := range values {
		bucket := int((v - minVal) / bucketSize)
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		if bucket < 0 {
			bucket = 0
		}
		counts[bucket]++
	}

	total := len(values)
	entropy := 0.0
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *TraceExtractor) calculateProfileEntropy(values []float64, windowSize int) float64 {
	if len(values) < windowSize || windowSize < 2 {
		return 0
	}

	profiles := make([]int, windowSize)
	for i := 0; i < len(values); i++ {
		bucket := (i * windowSize) / len(values)
		if bucket >= windowSize {
			bucket = windowSize - 1
		}
		profiles[bucket]++
	}

	total := len(values)
	entropy := 0.0
	for _, count := range profiles {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *TraceExtractor) medianFloat(values []float64) float64 {
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

func (e *TraceExtractor) meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (e *TraceExtractor) varianceFloat(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := e.meanFloat(values)
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func (e *TraceExtractor) maxAbsFloat(values []float64) float64 {
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

func (e *TraceExtractor) calculateSkewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	mean := e.meanFloat(values)
	stdDev := math.Sqrt(e.varianceFloat(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func (e *TraceExtractor) calculateKurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}
	mean := e.meanFloat(values)
	stdDev := math.Sqrt(e.varianceFloat(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 4)
	}
	n := float64(len(values))
	return (sum / n) - 3.0
}

func (e *TraceExtractor) calculateHurstExponent(values []float64) float64 {
	if len(values) < 10 {
		return 0.5
	}

	n := len(values)
	stdDev := math.Sqrt(e.varianceFloat(values))

	if stdDev == 0 {
		return 0.5
	}

	ranges := []int{4, 8, 16, 32}
	rsValues := make([]float64, 0)

	for _, lag := range ranges {
		if n < lag*2 {
			continue
		}

		subseries := n / lag
		rSum := 0.0

		for i := 0; i < subseries; i++ {
			endIdx := (i + 1) * lag
			if endIdx > n {
				endIdx = n
			}
			subseq := values[i*lag : endIdx]
			subMean := e.meanFloat(subseq)

			maxDev := -1e10
			minDev := 1e10
			cumsum := 0.0
			for _, v := range subseq {
				cumsum += v - subMean
				if cumsum > maxDev {
					maxDev = cumsum
				}
				if cumsum < minDev {
					minDev = cumsum
				}
			}

			r := maxDev - minDev
			rSum += r
		}

		avgR := rSum / float64(subseries)
		rsValues = append(rsValues, avgR/stdDev)
	}

	if len(rsValues) < 2 {
		return 0.5
	}

	sumLogN := 0.0
	sumLogRS := 0.0
	sumLogN2 := 0.0
	sumLogNLogRS := 0.0

	for i, lag := range ranges[:len(rsValues)] {
		logN := math.Log(float64(lag))
		logRS := math.Log(rsValues[i])

		sumLogN += logN
		sumLogRS += logRS
		sumLogN2 += logN * logN
		sumLogNLogRS += logN * logRS
	}

	m := float64(len(rsValues))
	denom := m*sumLogN2 - sumLogN*sumLogN
	if denom == 0 {
		return 0.5
	}

	hurst := (m*sumLogNLogRS - sumLogN*sumLogRS) / denom

	if hurst < 0 {
		hurst = 0
	}
	if hurst > 1 {
		hurst = 1
	}

	return hurst
}

func (e *TraceExtractor) calculateSpectralEntropy(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}

	fft := e.computeFFT(values)
	if len(fft) == 0 {
		return 0
	}

	var sum float64
	for _, v := range fft {
		sum += v
	}

	if sum == 0 {
		return 0
	}

	entropy := 0.0
	for _, v := range fft {
		p := v / sum
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *TraceExtractor) computeFFT(values []float64) []float64 {
	n := len(values)
	if n < 2 {
		return values
	}

	result := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		var real, imag float64
		for j := 0; j < n; j++ {
			angle := -2 * math.Pi * float64(i*j) / float64(n)
			real += values[j] * math.Cos(angle)
			imag += values[j] * math.Sin(angle)
		}
		result[i] = math.Sqrt(real*real + imag*imag)
	}

	return result
}

func (e *TraceExtractor) calculatePermutationEntropy(values []float64) float64 {
	if len(values) < 5 {
		return 0
	}

	order := 3
	n := len(values) - order + 1
	if n < 1 {
		return 0
	}

	patterns := make(map[int]int)
	for i := 0; i < n; i++ {
		indices := make([]int, order)
		for j := 0; j < order; j++ {
			indices[j] = j
		}

		for j := 0; j < order; j++ {
			for k := j + 1; k < order; k++ {
				if values[i+k] < values[i+j] {
					indices[j], indices[k] = indices[k], indices[j]
				}
			}
		}

		pattern := 0
		for j := 0; j < order; j++ {
			pattern += indices[j] * int(math.Pow(float64(order), float64(j)))
		}
		patterns[pattern]++
	}

	entropy := 0.0
	total := float64(n)
	for _, count := range patterns {
		p := float64(count) / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	maxEntropy := math.Log2(math.Pow(float64(order), float64(order)))
	if maxEntropy > 0 {
		entropy /= maxEntropy
	}

	return entropy
}

func (e *TraceExtractor) calculateApproximateEntropy(values []float64) float64 {
	if len(values) < 10 {
		return 0
	}

	m := 2
	r := 0.2 * e.stdDev(values)

	phi := e.computePhi(values, m, r)
	phi1 := e.computePhi(values, m+1, r)

	if phi == 0 || phi1 == 0 {
		return 0
	}

	return phi - phi1
}

func (e *TraceExtractor) calculateSampleEntropy(values []float64) float64 {
	if len(values) < 10 {
		return 0
	}

	m := 2
	r := 0.2 * e.stdDev(values)

	b := e.countTemplateMatches(values, m, r)
	a := e.countTemplateMatches(values, m+1, r)

	if b == 0 || a == 0 {
		return 0
	}

	return -math.Log(a / b)
}

func (e *TraceExtractor) computePhi(data []float64, m int, r float64) float64 {
	n := len(data)
	if n < m {
		return 0
	}

	count := 0
	for i := 0; i < n-m+1; i++ {
		for j := 0; j < n-m+1; j++ {
			if i == j {
				continue
			}
			match := true
			for k := 0; k < m; k++ {
				if math.Abs(data[i+k]-data[j+k]) > r {
					match = false
					break
				}
			}
			if match {
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	return math.Log(float64(count) / float64((n-m+1)*(n-m)))
}

func (e *TraceExtractor) countTemplateMatches(data []float64, m int, r float64) float64 {
	n := len(data)
	if n < m {
		return 0
	}

	count := 0
	for i := 0; i < n-m+1; i++ {
		for j := i + 1; j < n-m+1; j++ {
			match := true
			for k := 0; k < m; k++ {
				if math.Abs(data[i+k]-data[j+k]) > r {
					match = false
					break
				}
			}
			if match {
				count++
			}
		}
	}

	return float64(count)
}

func (e *TraceExtractor) stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	return math.Sqrt(e.varianceFloat(values))
}

func (e *TraceExtractor) countPeaks(values []float64) int {
	if len(values) < 3 {
		return 0
	}

	peaks := 0
	threshold := e.stdDev(values) * 0.3

	for i := 1; i < len(values)-1; i++ {
		if values[i] > values[i-1]+threshold && values[i] > values[i+1]+threshold {
			peaks++
		}
	}

	return peaks
}

func (e *TraceExtractor) calculateAutocorrelation(values []float64, lag int) float64 {
	if len(values) < lag+1 {
		return 0
	}

	mean := e.meanFloat(values)
	n := len(values) - lag

	numerator := 0.0
	denominator := 0.0

	for i := 0; i < n; i++ {
		numerator += (values[i] - mean) * (values[i+lag] - mean)
		denominator += (values[i] - mean) * (values[i] - mean)
	}

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func (e *TraceExtractor) calculateZeroCrossingRate(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	crossings := 0
	for i := 1; i < len(values); i++ {
		if values[i-1]*values[i] < 0 {
			crossings++
		}
	}

	return float64(crossings) / float64(len(values)-1)
}

func (e *TraceExtractor) calculateCircularVariance(directions []float64) float64 {
	if len(directions) < 2 {
		return 0
	}

	var sinSum, cosSum float64
	for _, dir := range directions {
		sinSum += math.Sin(dir)
		cosSum += math.Cos(dir)
	}

	sinMean := sinSum / float64(len(directions))
	cosMean := cosSum / float64(len(directions))

	r := math.Sqrt(sinMean*sinMean + cosMean*cosMean)

	return 1.0 - r
}

func (e *TraceExtractor) calculateDirectionalConsistency(directions []float64) float64 {
	if len(directions) < 2 {
		return 0
	}

	meanDir := 0.0
	for _, dir := range directions {
		meanDir += dir
	}
	meanDir /= float64(len(directions))

	var sumDiff float64
	for _, dir := range directions {
		diff := math.Abs(dir - meanDir)
		if diff > math.Pi {
			diff = 2*math.Pi - diff
		}
		sumDiff += diff
	}

	avgDiff := sumDiff / float64(len(directions))

	return 1.0 - avgDiff/math.Pi
}

func (e *TraceExtractor) calculateFractalDimension(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 10 {
		return 1.0
	}

	points := make([][]float64, len(traceData.Points))
	for i, p := range traceData.Points {
		points[i] = []float64{p.X, p.Y}
	}

	return e.boxCountFractalDimension(points)
}

func (e *TraceExtractor) boxCountFractalDimension(points [][]float64) float64 {
	if len(points) < 5 {
		return 1.0
	}

	minX, maxX := points[0][0], points[0][0]
	minY, maxY := points[0][1], points[0][1]

	for _, p := range points {
		if p[0] < minX {
			minX = p[0]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}

	boxSizes := []float64{0.05, 0.1, 0.2, 0.3, 0.4}
	counts := make([]int, 0)

	for _, boxSize := range boxSizes {
		count := e.countBoxes(points, boxSize, minX, maxX, minY, maxY)
		if count > 0 && count < len(points) {
			counts = append(counts, count)
		}
	}

	if len(counts) < 2 {
		return 1.5
	}

	sumLogS := 0.0
	sumLogN := 0.0
	sumLogS2 := 0.0
	sumLogSLogN := 0.0

	for i, count := range counts {
		logS := math.Log(1.0 / boxSizes[i])
		logN := math.Log(float64(count))

		sumLogS += logS
		sumLogN += logN
		sumLogS2 += logS * logS
		sumLogSLogN += logS * logN
	}

	n := float64(len(counts))
	denom := n*sumLogS2 - sumLogS*sumLogS
	if denom == 0 {
		return 1.5
	}

	dimension := (n*sumLogSLogN - sumLogS*sumLogN) / denom

	if dimension < 1 {
		dimension = 1
	}
	if dimension > 2 {
		dimension = 2
	}

	return dimension
}

func (e *TraceExtractor) countBoxes(points [][]float64, boxSize, minX, maxX, minY, maxY float64) int {
	boxesX := int((maxX-minX)/boxSize) + 1
	boxesY := int((maxY-minY)/boxSize) + 1

	boxSet := make(map[int]bool)
	for _, p := range points {
		boxX := int((p[0] - minX) / boxSize)
		boxY := int((p[1] - minY) / boxSize)
		if boxX >= boxesX {
			boxX = boxesX - 1
		}
		if boxY >= boxesY {
			boxY = boxesY - 1
		}
		if boxX < 0 {
			boxX = 0
		}
		if boxY < 0 {
			boxY = 0
		}
		boxIndex := boxY*boxesX + boxX
		boxSet[boxIndex] = true
	}

	return len(boxSet)
}

func (e *TraceExtractor) calculateCorrelationDimension(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 20 {
		return 1.0
	}

	points := make([][]float64, len(traceData.Points))
	for i, p := range traceData.Points {
		points[i] = []float64{p.X, p.Y}
	}

	return e.computeCorrelationDimension(points)
}

func (e *TraceExtractor) computeCorrelationDimension(points [][]float64) float64 {
	n := len(points)
	if n < 10 {
		return 1.0
	}

	radii := []float64{2, 5, 10, 20, 30}
	counts := make([]int, 0)

	for _, r := range radii {
		count := 0
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				dx := points[i][0] - points[j][0]
				dy := points[i][1] - points[j][1]
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < r {
					count++
				}
			}
		}
		if count > 0 {
			counts = append(counts, count)
		}
	}

	if len(counts) < 2 {
		return 1.5
	}

	sumLogR := 0.0
	sumLogC := 0.0
	sumLogR2 := 0.0
	sumLogRLogC := 0.0

	for i, count := range counts {
		logR := math.Log(radii[i])
		logC := math.Log(float64(count))

		sumLogR += logR
		sumLogC += logC
		sumLogR2 += logR * logR
		sumLogRLogC += logR * logC
	}

	m := float64(len(counts))
	denom := m*sumLogR2 - sumLogR*sumLogR
	if denom == 0 {
		return 1.5
	}

	dimension := (m*sumLogRLogC - sumLogR*sumLogC) / denom

	if dimension < 1 {
		dimension = 1
	}
	if dimension > 2 {
		dimension = 2
	}

	return dimension
}

func (e *TraceExtractor) calculateSpatialDispersion(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	var sumX, sumY float64
	for _, p := range traceData.Points {
		sumX += p.X
		sumY += p.Y
	}
	meanX := sumX / float64(len(traceData.Points))
	meanY := sumY / float64(len(traceData.Points))

	var variance float64
	for _, p := range traceData.Points {
		dx := p.X - meanX
		dy := p.Y - meanY
		variance += dx*dx + dy*dy
	}
	variance /= float64(len(traceData.Points))

	return math.Sqrt(variance)
}

func (e *TraceExtractor) calculateTemporalIrregularity(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	intervals := make([]float64, len(traceData.Points)-1)
	for i := 1; i < len(traceData.Points); i++ {
		intervals[i-1] = float64(traceData.Points[i].Timestamp - traceData.Points[i-1].Timestamp)
	}

	mean := e.meanFloat(intervals)
	if mean == 0 {
		return 0
	}

	var sumRatio float64
	for _, interval := range intervals {
		sumRatio += math.Abs(interval - mean) / mean
	}

	return sumRatio / float64(len(intervals))
}

func (e *TraceExtractor) calculateLyapunovExponent(values []float64) float64 {
	if len(values) < 50 {
		return 0
	}

	n := len(values)
	exponents := make([]float64, 0)

	for i := 0; i < n-20; i += 10 {
		d0 := math.Abs(values[i+1] - values[i])
		if d0 == 0 {
			continue
		}

		for j := i + 1; j < n-10; j++ {
			d1 := math.Abs(values[j+1] - values[j])
			if d1 == 0 {
				continue
			}

			if math.Abs(d0-d1)/d0 < 0.1 {
				dist := 0.0
				for k := 0; k < 5 && i+k < n && j+k < n; k++ {
					dist += math.Abs(values[i+k] - values[j+k])
				}
				if dist > 0 {
					exponents = append(exponents, math.Log(dist/d0))
				}
				break
			}
		}
	}

	if len(exponents) == 0 {
		return 0
	}

	return e.meanFloat(exponents)
}
