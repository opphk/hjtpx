package trace

import (
	"errors"
	"math"

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
	features.Smoothness = e.calculateSmoothness(traceData)
	features.PauseCount = e.calculatePauseCount(traceData)
	features.TotalDistance = e.calculateTotalDistance(traceData)
	features.DirectDistance = e.calculateDirectDistance(traceData)

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

	return riskFactors
}
