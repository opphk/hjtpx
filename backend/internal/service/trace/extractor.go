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
	CurvatureMedian         float64
	CurvatureVariance       float64
	CurvatureMax            float64
	CurvatureEntropy        float64
	DirectionChangeRate     float64
	DirectionEntropy        float64
	Sinuosity               float64
	StartEndAngle           float64
	AreaUnderCurve          float64
	TimeNormalizedDistance  float64
	VelocityProfileEntropy  float64
	AccelerationProfileEntropy float64
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
	}

	accelerations := e.computeAccelerations(traceData)
	if len(accelerations) > 0 {
		features.MedianAcceleration = e.medianFloat(accelerations)
		features.AccelerationVariance = e.varianceFloat(accelerations)
		features.AccelerationSkewness = e.calculateSkewness(accelerations)
	}

	jerks := e.computeJerks(traceData)
	if len(jerks) > 0 {
		features.JerkMean = e.meanFloat(jerks)
		features.JerkMax = e.maxAbsFloat(jerks)
	}

	curvatures := e.computeCurvatures(traceData)
	if len(curvatures) > 0 {
		features.CurvatureMedian = e.medianFloat(curvatures)
		features.CurvatureVariance = e.varianceFloat(curvatures)
		features.CurvatureMax = e.maxAbsFloat(curvatures)
		features.CurvatureEntropy = e.calculateEntropy(curvatures, 5)
	}

	directions := e.computeDirections(traceData)
	if len(directions) > 1 {
		features.DirectionChangeRate = e.calculateDirectionChangeRate(directions)
		features.DirectionEntropy = e.calculateDirectionEntropy(directions)
	}

	if len(traceData.Points) > 1 {
		features.Sinuosity = e.calculateSinuosity(traceData)
		features.StartEndAngle = e.calculateStartEndAngle(traceData)
		features.AreaUnderCurve = e.calculateAreaUnderCurve(traceData)
		features.TimeNormalizedDistance = e.calculateTimeNormalizedDistance(traceData)
	}

	features.VelocityProfileEntropy = e.calculateProfileEntropy(speeds, 5)
	features.AccelerationProfileEntropy = e.calculateProfileEntropy(accelerations, 5)

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

func (e *TraceExtractor) ExtractEnhancedFeatures(traceData *model.TraceData) (*EnhancedFeatures, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据点不足")
	}

	features := &EnhancedFeatures{}

	_, err := e.ExtractFeatures(traceData)
	if err != nil {
		return nil, err
	}

	advanced, err := e.ExtractAdvancedFeatures(traceData)
	if err == nil && advanced != nil {
		features.CurvatureVariance = advanced.CurvatureVariance
		features.CurvatureEntropy = advanced.CurvatureEntropy
	}

	pressures := e.extractPressureSequence(traceData)
	if len(pressures) > 0 {
		features.AvgPressure = e.meanFloat(pressures)
		features.PressureVariance = e.varianceFloat(pressures)
		features.MaxPressure = e.maxFloat(pressures)
		features.MinPressure = e.minFloat(pressures)
		features.PressureSkewness = e.calculateSkewness(pressures)
		features.PressureKurtosis = e.calculateKurtosis(pressures)
	}

	if len(traceData.ClickData) > 0 {
		features.ClickCount = len(traceData.ClickData)
		features.AvgClickInterval = e.calculateClickIntervalFromClickData(traceData.ClickData)
		features.ClickRegularity = e.calculateClickRegularity(traceData.ClickData)
		features.ClickAreaSize = e.calculateClickAreaSize(traceData.ClickData)
		features.TargetedClickRate = e.calculateTargetedClickRate(traceData.ClickData)
		features.AvgClickPressure = e.calculateAvgClickPressure(traceData.ClickData)
		features.ClickPressureVariance = e.calculateClickPressureVariance(traceData.ClickData)
	}

	if len(traceData.ScrollData) > 0 {
		features.ScrollCount = len(traceData.ScrollData)
		features.AvgScrollVelocity = e.calculateAvgScrollVelocity(traceData.ScrollData)
		features.ScrollRegularity = e.calculateScrollRegularity(traceData.ScrollData)
		features.ScrollDirectionEntropy = e.calculateScrollDirectionEntropy(traceData.ScrollData)
		features.ScrollVelocityVariance = e.calculateScrollVelocityVariance(traceData.ScrollData)
	}

	features.MovementFluidity = e.calculateMovementFluidity(traceData)

	features.CurvatureSkewness = e.calculateCurvatureSkewness(traceData)

	return features, nil
}

type EnhancedFeatures struct {
	CurvatureVariance      float64
	CurvatureSkewness      float64
	CurvatureEntropy       float64
	AvgPressure            float64
	PressureVariance       float64
	MaxPressure            float64
	MinPressure            float64
	PressureSkewness       float64
	PressureKurtosis       float64
	PressureConsistency    float64
	ClickCount             int
	AvgClickInterval       float64
	ClickRegularity        float64
	ClickAreaSize          float64
	TargetedClickRate      float64
	AvgClickPressure      float64
	ClickPressureVariance float64
	ScrollCount            int
	AvgScrollVelocity      float64
	ScrollRegularity       float64
	ScrollDirectionEntropy float64
	ScrollVelocityVariance float64
	MovementFluidity       float64
	SpatialSpreadX         float64
	SpatialSpreadY         float64
	TemporalIntervalStdDev float64
	PauseRatio             float64
}

func (e *TraceExtractor) extractPressureSequence(traceData *model.TraceData) []float64 {
	pressures := make([]float64, 0, len(traceData.Points))
	for _, p := range traceData.Points {
		if p.Pressure > 0 {
			pressures = append(pressures, p.Pressure)
		} else if p.Event == "click" || p.Event == "touchstart" {
			pressures = append(pressures, 0.5)
		} else {
			pressures = append(pressures, 0.1)
		}
	}
	return pressures
}

func (e *TraceExtractor) calculateClickIntervalFromClickData(clicks []model.ClickInfo) float64 {
	if len(clicks) < 2 {
		return 0
	}

	var totalInterval float64
	count := 0

	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		if interval > 0 {
			totalInterval += interval
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalInterval / float64(count)
}

func (e *TraceExtractor) calculateClickRegularity(clicks []model.ClickInfo) float64 {
	if len(clicks) < 3 {
		return 1.0
	}

	intervals := make([]float64, 0)
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		if interval > 0 {
			intervals = append(intervals, interval)
		}
	}

	if len(intervals) < 2 {
		return 1.0
	}

	mean := e.meanFloat(intervals)
	if mean == 0 {
		return 0
	}

	var variance float64
	for _, interval := range intervals {
		diff := interval - mean
		variance += diff * diff
	}
	variance /= float64(len(intervals))

	stdDev := math.Sqrt(variance)

	regularity := 1.0 - math.Min(stdDev/mean, 1.0)

	return regularity
}

func (e *TraceExtractor) calculateClickAreaSize(clicks []model.ClickInfo) float64 {
	if len(clicks) < 2 {
		return 0
	}

	minX, maxX := clicks[0].X, clicks[0].X
	minY, maxY := clicks[0].Y, clicks[0].Y

	for _, c := range clicks {
		if c.X < minX {
			minX = c.X
		}
		if c.X > maxX {
			maxX = c.X
		}
		if c.Y < minY {
			minY = c.Y
		}
		if c.Y > maxY {
			maxY = c.Y
		}
	}

	return (maxX - minX) * (maxY - minY) / 10000.0
}

func (e *TraceExtractor) calculateTargetedClickRate(clicks []model.ClickInfo) float64 {
	if len(clicks) == 0 {
		return 0
	}

	targetedCount := 0
	for _, c := range clicks {
		if c.IsTargeted {
			targetedCount++
		}
	}

	return float64(targetedCount) / float64(len(clicks))
}

func (e *TraceExtractor) calculateAvgClickPressure(clicks []model.ClickInfo) float64 {
	if len(clicks) == 0 {
		return 0
	}

	var totalPressure float64
	for _, c := range clicks {
		if c.Pressure > 0 {
			totalPressure += c.Pressure
		} else {
			totalPressure += 0.5
		}
	}

	return totalPressure / float64(len(clicks))
}

func (e *TraceExtractor) calculateClickPressureVariance(clicks []model.ClickInfo) float64 {
	if len(clicks) < 2 {
		return 0
	}

	pressures := make([]float64, len(clicks))
	for i, c := range clicks {
		if c.Pressure > 0 {
			pressures[i] = c.Pressure
		} else {
			pressures[i] = 0.5
		}
	}

	return e.varianceFloat(pressures)
}

func (e *TraceExtractor) calculateAvgScrollVelocity(scrolls []model.ScrollInfo) float64 {
	if len(scrolls) == 0 {
		return 0
	}

	var totalVelocity float64
	for _, s := range scrolls {
		if s.Velocity > 0 {
			totalVelocity += s.Velocity
		} else {
			dx := math.Abs(s.DeltaX)
			dy := math.Abs(s.DeltaY)
			totalVelocity += math.Sqrt(dx*dx + dy*dy)
		}
	}

	return totalVelocity / float64(len(scrolls))
}

func (e *TraceExtractor) calculateScrollRegularity(scrolls []model.ScrollInfo) float64 {
	if len(scrolls) < 3 {
		return 1.0
	}

	velocities := make([]float64, len(scrolls))
	for i, s := range scrolls {
		if s.Velocity > 0 {
			velocities[i] = s.Velocity
		} else {
			dx := math.Abs(s.DeltaX)
			dy := math.Abs(s.DeltaY)
			velocities[i] = math.Sqrt(dx*dx + dy*dy)
		}
	}

	mean := e.meanFloat(velocities)
	if mean == 0 {
		return 0
	}

	var variance float64
	for _, v := range velocities {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(velocities))

	stdDev := math.Sqrt(variance)

	return 1.0 - math.Min(stdDev/mean, 1.0)
}

func (e *TraceExtractor) calculateScrollDirectionEntropy(scrolls []model.ScrollInfo) float64 {
	if len(scrolls) == 0 {
		return 0
	}

	directions := make([]string, 0)
	for _, s := range scrolls {
		directions = append(directions, s.Direction)
	}

	directionCounts := make(map[string]int)
	for _, d := range directions {
		directionCounts[d]++
	}

	total := len(directions)
	entropy := 0.0

	for _, count := range directionCounts {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *TraceExtractor) calculateScrollVelocityVariance(scrolls []model.ScrollInfo) float64 {
	if len(scrolls) < 2 {
		return 0
	}

	velocities := make([]float64, len(scrolls))
	for i, s := range scrolls {
		if s.Velocity > 0 {
			velocities[i] = s.Velocity
		} else {
			dx := math.Abs(s.DeltaX)
			dy := math.Abs(s.DeltaY)
			velocities[i] = math.Sqrt(dx*dx + dy*dy)
		}
	}

	return e.varianceFloat(velocities)
}

func (e *TraceExtractor) calculateMovementFluidity(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 1.0
	}

	smoothness := e.calculateSmoothness(traceData)

	pauseRatio := 0.0
	if traceData.TotalTime > 0 {
		pauseCount := e.calculatePauseCount(traceData)
		pauseRatio = float64(pauseCount) / float64(len(traceData.Points))
	}

	speedVariance := e.calculateSpeedVariance(traceData)
	avgSpeed := e.calculateAvgSpeed(traceData)
	normalizedVariance := 0.0
	if avgSpeed > 0 {
		normalizedVariance = speedVariance / (avgSpeed * avgSpeed)
	}

	smoothnessScore := math.Max(0, 1.0-smoothness/0.3)

	fluidity := smoothnessScore * (1.0 - pauseRatio) * (1.0 - math.Min(normalizedVariance, 1.0))

	return math.Max(0, math.Min(1, fluidity))
}

func (e *TraceExtractor) calculateCurvatureSkewness(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	curvatures := e.computeCurvatures(traceData)
	if len(curvatures) < 2 {
		return 0
	}

	return e.calculateSkewness(curvatures)
}

func (e *TraceExtractor) maxFloat(values []float64) float64 {
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

func (e *TraceExtractor) minFloat(values []float64) float64 {
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
