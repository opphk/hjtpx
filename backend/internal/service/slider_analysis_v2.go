package service

import (
	"math"
	"sync"
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

type SliderAnalysisV2 struct {
	enableParallel bool
	windowSize      int
}

func NewSliderAnalysisV2() *SliderAnalysisV2 {
	return &SliderAnalysisV2{
		enableParallel: true,
		windowSize:     5,
	}
}

func (s *SliderAnalysisV2) SetParallelEnabled(enabled bool) {
	s.enableParallel = enabled
}

func (s *SliderAnalysisV2) SetWindowSize(size int) {
	if size > 0 && size <= 20 {
		s.windowSize = size
	}
}

func (s *SliderAnalysisV2) AnalyzeTrajectory(points []SliderTrajectoryPoint) *TrajectoryAnalysisResult {
	if len(points) < 2 {
		return &TrajectoryAnalysisResult{
			SmoothedPoints:      points,
			SpeedFeatures:       SpeedFeatures{},
			JitterScore:         0,
			AccelerationPattern: AccelerationPattern{Type: "insufficient_data"},
		}
	}

	smoothedPoints := s.smoothTrajectory(points)
	
	var speedFeatures SpeedFeatures
	var jitterScore float64
	var accelerationPattern AccelerationPattern
	var waitGroup sync.WaitGroup
	
	if s.enableParallel && len(points) > 10 {
		var mu sync.Mutex
		
		waitGroup.Add(3)
		
		go func() {
			defer waitGroup.Done()
			speedFeatures = s.extractSpeedFeaturesOptimized(smoothedPoints)
		}()
		
		go func() {
			defer waitGroup.Done()
			jitterScore = s.detectJitterOptimized(smoothedPoints)
		}()
		
		go func() {
			defer waitGroup.Done()
			accelerationPattern = s.detectAccelerationPatternOptimized(smoothedPoints)
		}()
		
		waitGroup.Wait()
	} else {
		speedFeatures = s.extractSpeedFeaturesOptimized(smoothedPoints)
		jitterScore = s.detectJitterOptimized(smoothedPoints)
		accelerationPattern = s.detectAccelerationPatternOptimized(smoothedPoints)
	}

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
	windowSize := s.windowSize

	for i := range points {
		sumX, sumY := 0.0, 0.0
		count := 0
		start := sliderMax(0, i-windowSize/2)
		end := sliderMin(len(points)-1, i+windowSize/2)
		for j := start; j <= end; j++ {
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

func (s *SliderAnalysisV2) extractSpeedFeaturesOptimized(points []SliderTrajectoryPoint) SpeedFeatures {
	if len(points) < 2 {
		return SpeedFeatures{}
	}
	
	speeds := make([]float64, 0, len(points)-1)
	var sumSpeed, maxSpeed, minSpeed float64 = 0, 0, math.MaxFloat64
	
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := points[i].Timestamp - points[i-1].Timestamp
		if dt > 0 {
			speed := math.Sqrt(dx*dx+dy*dy) / float64(dt)
			speeds = append(speeds, speed)
			sumSpeed += speed
			if speed > maxSpeed {
				maxSpeed = speed
			}
			if speed < minSpeed {
				minSpeed = speed
			}
		}
	}

	if len(speeds) == 0 {
		return SpeedFeatures{}
	}
	
	avgSpeed := sumSpeed / float64(len(speeds))
	
	var varianceSum float64
	for _, speed := range speeds {
		diff := speed - avgSpeed
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(speeds))

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

func (s *SliderAnalysisV2) detectJitterOptimized(points []SliderTrajectoryPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	jitterCount := 0
	threshold := math.Pi / 4
	
	for i := 2; i < len(points); i++ {
		dx1 := points[i-1].X - points[i-2].X
		dy1 := points[i-1].Y - points[i-2].Y
		dx2 := points[i].X - points[i-1].X
		dy2 := points[i].Y - points[i-1].Y
		
		if dx1 == 0 && dy1 == 0 || dx2 == 0 && dy2 == 0 {
			continue
		}
		
		dir1 := math.Atan2(dy1, dx1)
		dir2 := math.Atan2(dy2, dx2)
		diff := math.Abs(dir2 - dir1)
		
		if diff > threshold && diff < 2*math.Pi-threshold {
			jitterCount++
		}
	}

	if len(points) <= 2 {
		return 0
	}
	return float64(jitterCount) / float64(len(points)-2)
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

func (s *SliderAnalysisV2) detectAccelerationPatternOptimized(points []SliderTrajectoryPoint) AccelerationPattern {
	if len(points) < 3 {
		return AccelerationPattern{Type: "insufficient_data"}
	}

	var accelerations []float64
	accelerations = make([]float64, 0, len(points)-2)
	
	var sumAcc, maxAcc, minAcc float64 = 0, 0, math.MaxFloat64
	positiveCount := 0
	negativeCount := 0

	for i := 2; i < len(points); i++ {
		speed1 := s.calculateSpeed(points[i-2], points[i-1])
		speed2 := s.calculateSpeed(points[i-1], points[i])
		acc := speed2 - speed1
		accelerations = append(accelerations, acc)
		
		sumAcc += acc
		if acc > maxAcc {
			maxAcc = acc
		}
		if acc < minAcc {
			minAcc = acc
		}
		if acc > 0 {
			positiveCount++
		} else {
			negativeCount++
		}
	}

	if len(accelerations) == 0 {
		return AccelerationPattern{Type: "unknown"}
	}

	pattern := "uniform"
	if positiveCount > negativeCount*2 {
		pattern = "accelerating"
	} else if negativeCount > positiveCount*2 {
		pattern = "decelerating"
	}

	avgAcc := sumAcc / float64(len(accelerations))

	return AccelerationPattern{
		Type:            pattern,
		Average:         avgAcc,
		MaxAcceleration: maxAcc,
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

func (s *SliderAnalysisV2) CalculatePathEfficiency(points []SliderTrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	directDistance := math.Sqrt(
		math.Pow(points[len(points)-1].X-points[0].X, 2) +
		math.Pow(points[len(points)-1].Y-points[0].Y, 2),
	)

	var pathLength float64
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		pathLength += math.Sqrt(dx*dx + dy*dy)
	}

	if pathLength == 0 {
		return 0
	}

	return directDistance / pathLength
}

func (s *SliderAnalysisV2) DetectPauses(points []SliderTrajectoryPoint, threshold float64) (pauseCount int, totalPauseDuration int64) {
	if len(points) < 2 {
		return 0, 0
	}

	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := points[i].Timestamp - points[i-1].Timestamp

		if dt > 0 {
			speed := math.Sqrt(dx*dx+dy*dy) / float64(dt)
			if speed < threshold {
				pauseCount++
				totalPauseDuration += dt
			}
		}
	}

	return
}

func (s *SliderAnalysisV2) AnalyzeTrajectoryExtended(points []SliderTrajectoryPoint) *ExtendedAnalysisResult {
	if len(points) < 2 {
		return &ExtendedAnalysisResult{
			TrajectoryAnalysisResult: &TrajectoryAnalysisResult{
				SmoothedPoints:      points,
				SpeedFeatures:       SpeedFeatures{},
				JitterScore:         0,
				AccelerationPattern: AccelerationPattern{Type: "insufficient_data"},
			},
			PathEfficiency:     0,
			PauseCount:        0,
			PauseDuration:     0,
			IsHumanLikely:     false,
			Confidence:        0,
		}
	}

	basicAnalysis := s.AnalyzeTrajectory(points)
	pathEfficiency := s.CalculatePathEfficiency(points)
	pauseCount, pauseDuration := s.DetectPauses(points, 0.5)

	isHumanLikely := s.evaluateHumanLikeness(basicAnalysis, pathEfficiency, pauseCount)
	confidence := s.calculateConfidence(basicAnalysis, pathEfficiency, pauseCount)

	return &ExtendedAnalysisResult{
		TrajectoryAnalysisResult: basicAnalysis,
		PathEfficiency:           pathEfficiency,
		PauseCount:              pauseCount,
		PauseDuration:            pauseDuration,
		IsHumanLikely:           isHumanLikely,
		Confidence:              confidence,
	}
}

func (s *SliderAnalysisV2) evaluateHumanLikely(analysis *TrajectoryAnalysisResult, pathEfficiency float64, pauseCount int) bool {
	if analysis.JitterScore > 0.5 {
		return false
	}

	if pathEfficiency < 0.3 {
		return false
	}

	if analysis.AccelerationPattern.Type == "uniform" && analysis.SpeedFeatures.Variance < 0.1 {
		return false
	}

	if pauseCount > len(analysis.SmoothedPoints)/3 {
		return false
	}

	return true
}

func (s *SliderAnalysisV2) calculateConfidence(analysis *TrajectoryAnalysisResult, pathEfficiency float64, pauseCount int) float64 {
	if len(analysis.SmoothedPoints) < 3 {
		return 0.1
	}

	confidence := 0.5

	if analysis.JitterScore < 0.3 {
		confidence += 0.2
	} else if analysis.JitterScore > 0.6 {
		confidence -= 0.3
	}

	if pathEfficiency > 0.7 {
		confidence += 0.15
	} else if pathEfficiency < 0.4 {
		confidence -= 0.2
	}

	if analysis.SpeedFeatures.Variance > 0.1 {
		confidence += 0.1
	}

	if pauseCount < len(analysis.SmoothedPoints)/5 {
		confidence += 0.1
	}

	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

type ExtendedAnalysisResult struct {
	*TrajectoryAnalysisResult
	PathEfficiency float64
	PauseCount     int
	PauseDuration  int64
	IsHumanLikely  bool
	Confidence     float64
}
