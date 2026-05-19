package service

import (
	"math"
)

func sliderMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sliderMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type SliderAnalysisV2 struct{}

func (s *SliderAnalysisV2) AnalyzeTrajectory(points []SliderTrajectoryPoint) *TrajectoryAnalysisResult {
	smoothedPoints := s.smoothTrajectory(points)
	speedFeatures := s.extractSpeedFeatures(smoothedPoints)
	jitterScore := s.detectJitter(smoothedPoints)
	accelerationPattern := s.detectAccelerationPattern(smoothedPoints)

	return &TrajectoryAnalysisResult{
		SmoothedPoints:      smoothedPoints,
		SpeedFeatures:        speedFeatures,
		JitterScore:          jitterScore,
		AccelerationPattern: accelerationPattern,
	}
}

func (s *SliderAnalysisV2) smoothTrajectory(points []SliderTrajectoryPoint) []SliderTrajectoryPoint {
	if len(points) < 3 {
		return points
	}
	smoothed := make([]SliderTrajectoryPoint, len(points))
	windowSize := 5

	for i := range points {
		sumX, sumY := 0.0, 0.0
		count := 0
		for j := sliderMax(0, i-windowSize/2); j <= sliderMin(len(points)-1, i+windowSize/2); j++ {
			sumX += points[j].X
			sumY += points[j].Y
			count++
		}
		smoothed[i].X = sumX / float64(count)
		smoothed[i].Y = sumY / float64(count)
		smoothed[i].Timestamp = points[i].Timestamp
	}
	return smoothed
}

func (s *SliderAnalysisV2) extractSpeedFeatures(points []SliderTrajectoryPoint) SpeedFeatures {
	var speeds []float64
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := points[i].Timestamp - points[i-1].Timestamp
		if dt > 0 {
			speed := math.Sqrt(dx*dx+dy*dy) / float64(dt)
			speeds = append(speeds, speed)
		}
	}

	avgSpeed := sliderAnalysisCalculateAverage(speeds)
	maxSpeed := sliderAnalysisCalculateMax(speeds)
	minSpeed := sliderAnalysisCalculateMin(speeds)
	variance := sliderAnalysisCalculateVariance(speeds, avgSpeed)

	return SpeedFeatures{
		Average:  avgSpeed,
		Max:      maxSpeed,
		Min:      minSpeed,
		Variance: variance,
	}
}

func (s *SliderAnalysisV2) detectJitter(points []SliderTrajectoryPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	jitterCount := 0
	for i := 2; i < len(points); i++ {
		dir1 := math.Atan2(points[i-1].Y-points[i-2].Y, points[i-1].X-points[i-2].X)
		dir2 := math.Atan2(points[i].Y-points[i-1].Y, points[i].X-points[i-1].X)
		diff := math.Abs(dir2 - dir1)
		if diff > math.Pi/4 {
			jitterCount++
		}
	}

	return float64(jitterCount) / float64(len(points))
}

func (s *SliderAnalysisV2) detectAccelerationPattern(points []SliderTrajectoryPoint) AccelerationPattern {
	var accelerations []float64
	for i := 2; i < len(points); i++ {
		speed1 := s.calculateSpeed(points[i-2], points[i-1])
		speed2 := s.calculateSpeed(points[i-1], points[i])
		acc := speed2 - speed1
		accelerations = append(accelerations, acc)
	}

	if len(accelerations) == 0 {
		return AccelerationPattern{Type: "unknown"}
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

	pattern := "uniform"
	if positiveCount > negativeCount*2 {
		pattern = "accelerating"
	} else if negativeCount > positiveCount*2 {
		pattern = "decelerating"
	}

	return AccelerationPattern{
		Type:            pattern,
		Average:         sliderAnalysisCalculateAverage(accelerations),
		MaxAcceleration: sliderAnalysisCalculateMax(accelerations),
	}
}

func (s *SliderAnalysisV2) calculateSpeed(p1, p2 SliderTrajectoryPoint) float64 {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	dt := p2.Timestamp - p1.Timestamp
	if dt == 0 {
		return 0
	}
	return math.Sqrt(dx*dx+dy*dy) / float64(dt)
}

func sliderAnalysisCalculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func sliderAnalysisCalculateMax(values []float64) float64 {
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

func sliderAnalysisCalculateMin(values []float64) float64 {
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

func sliderAnalysisCalculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

type SliderTrajectoryPoint struct {
	X         float64
	Y         float64
	Timestamp int64
}

type TrajectoryAnalysisResult struct {
	SmoothedPoints      []SliderTrajectoryPoint
	SpeedFeatures       SpeedFeatures
	JitterScore         float64
	AccelerationPattern AccelerationPattern
}

type SpeedFeatures struct {
	Average  float64
	Max      float64
	Min      float64
	Variance float64
}

type AccelerationPattern struct {
	Type            string
	Average         float64
	MaxAcceleration float64
}
