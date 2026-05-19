package captcha

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

type SliderSecurityEnhancer struct {
	minTrajectoryPoints int
	maxSpeed            float64
	maxAcceleration     float64
	enableHightSampling bool
}

type EnhancedTrajectoryAnalysis struct {
	TrajectoryPoints     []TrajectoryPoint
	SpeedProfile         SpeedProfile
	AccelerationProfile  AccelerationProfile
	DirectionChanges     int
	TotalDistance        float64
	AverageSpeed         float64
	MaxSpeed             float64
	MinSpeed             float64
	SpeedVariance        float64
	IsHumanLike          bool
	Confidence           float64
	RiskLevel            string
	AnomalyIndicators    []string
}

type TrajectoryPoint struct {
	X         float64
	Y         float64
	Timestamp int64
	Speed     float64
}

type SpeedProfile struct {
	InitialSpeed   float64
	FinalSpeed     float64
	AverageSpeed   float64
	SpeedFluctuation float64
	SpeedConsistency float64
}

type AccelerationProfile struct {
	AverageAcceleration float64
	MaxAcceleration     float64
	MinAcceleration     float64
	JerkCount           int
	Smoothness          float64
}

func NewSliderSecurityEnhancer() *SliderSecurityEnhancer {
	return &SliderSecurityEnhancer{
		minTrajectoryPoints: 10,
		maxSpeed:            2000,
		maxAcceleration:     500,
		enableHightSampling: true,
	}
}

func (e *SliderSecurityEnhancer) EnhancedTrajectoryAnalysis(points []SliderPoint, targetPosition int) *EnhancedTrajectoryAnalysis {
	if len(points) < e.minTrajectoryPoints {
		return &EnhancedTrajectoryAnalysis{
			IsHumanLike:       false,
			Confidence:        0,
			RiskLevel:         "high",
			AnomalyIndicators: []string{"insufficient_trajectory_points"},
		}
	}

	trajectoryPoints := e.convertToTrajectoryPoints(points)
	
	e.calculateSpeedAndAcceleration(trajectoryPoints)
	
	speedProfile := e.analyzeSpeedProfile(trajectoryPoints)
	
	accelerationProfile := e.analyzeAccelerationProfile(trajectoryPoints)
	
	directionChanges := e.countDirectionChanges(trajectoryPoints)
	
	totalDistance := e.calculateTotalDistance(trajectoryPoints)
	
	speedVariance := e.calculateSpeedVariance(trajectoryPoints)
	
	humanLikeScore := e.calculateHumanLikeScore(trajectoryPoints, speedProfile, accelerationProfile, directionChanges)
	
	riskLevel := e.determineRiskLevel(humanLikeScore, speedProfile, accelerationProfile)
	
	anomalyIndicators := e.detectAnomalyIndicators(trajectoryPoints, speedProfile, accelerationProfile, directionChanges)

	return &EnhancedTrajectoryAnalysis{
		TrajectoryPoints:     trajectoryPoints,
		SpeedProfile:        speedProfile,
		AccelerationProfile: accelerationProfile,
		DirectionChanges:    directionChanges,
		TotalDistance:       totalDistance,
		AverageSpeed:        speedProfile.AverageSpeed,
		MaxSpeed:            speedProfile.SpeedFluctuation,
		MinSpeed:            speedProfile.SpeedConsistency,
		SpeedVariance:       speedVariance,
		IsHumanLike:         humanLikeScore > 0.5,
		Confidence:          humanLikeScore * 100,
		RiskLevel:           riskLevel,
		AnomalyIndicators:   anomalyIndicators,
	}
}

func (e *SliderSecurityEnhancer) convertToTrajectoryPoints(points []SliderPoint) []TrajectoryPoint {
	trajectory := make([]TrajectoryPoint, len(points))
	
	for i, p := range points {
		trajectory[i] = TrajectoryPoint{
			X:         float64(p.X),
			Y:         float64(p.Y),
			Timestamp: p.Timestamp,
		}
	}
	
	return trajectory
}

func (e *SliderSecurityEnhancer) calculateSpeedAndAcceleration(points []TrajectoryPoint) {
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt <= 0 {
			dt = 1
		}
		
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		
		speed := distance / dt * 1000
		points[i].Speed = speed
		
		if i > 1 {
			prevSpeed := points[i-1].Speed
			acceleration := (speed - prevSpeed) / dt * 1000
		}
	}
	
	if len(points) > 0 {
		points[0].Speed = 0
	}
}

func (e *SliderSecurityEnhancer) analyzeSpeedProfile(points []TrajectoryPoint) SpeedProfile {
	if len(points) < 2 {
		return SpeedProfile{}
	}
	
	initialSpeed := points[0].Speed
	finalSpeed := points[len(points)-1].Speed
	
	var totalSpeed float64
	var maxSpeed float64
	var minSpeed float64 = 10000
	
	for _, p := range points {
		totalSpeed += p.Speed
		if p.Speed > maxSpeed {
			maxSpeed = p.Speed
		}
		if p.Speed < minSpeed {
			minSpeed = p.Speed
		}
	}
	
	averageSpeed := totalSpeed / float64(len(points))
	
	speedFluctuation := maxSpeed - averageSpeed
	speedConsistency := averageSpeed - minSpeed
	
	return SpeedProfile{
		InitialSpeed:     initialSpeed,
		FinalSpeed:       finalSpeed,
		AverageSpeed:     averageSpeed,
		SpeedFluctuation: speedFluctuation,
		SpeedConsistency: speedConsistency,
	}
}

func (e *SliderSecurityEnhancer) analyzeAccelerationProfile(points []TrajectoryPoint) AccelerationProfile {
	if len(points) < 3 {
		return AccelerationProfile{}
	}
	
	accelerations := make([]float64, 0, len(points)-2)
	
	for i := 2; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-2].Timestamp)
		if dt <= 0 {
			dt = 1
		}
		
		speedDiff := points[i].Speed - points[i-2].Speed
		acceleration := math.Abs(speedDiff / dt * 1000)
		accelerations = append(accelerations, acceleration)
	}
	
	if len(accelerations) == 0 {
		return AccelerationProfile{}
	}
	
	var totalAccel float64
	var maxAccel float64
	var minAccel float64 = 10000
	jerkCount := 0
	
	for i, accel := range accelerations {
		totalAccel += accel
		
		if accel > maxAccel {
			maxAccel = accel
		}
		if accel < minAccel {
			minAccel = accel
		}
		
		if i > 0 {
			jerk := math.Abs(accel - accelerations[i-1])
			if jerk > 100 {
				jerkCount++
			}
		}
	}
	
	averageAccel := totalAccel / float64(len(accelerations))
	
	smoothness := 1.0 - math.Min(float64(jerkCount)/float64(len(accelerations)), 1.0)
	
	return AccelerationProfile{
		AverageAcceleration: averageAccel,
		MaxAcceleration:     maxAccel,
		MinAcceleration:     minAccel,
		JerkCount:           jerkCount,
		Smoothness:          smoothness,
	}
}

func (e *SliderSecurityEnhancer) countDirectionChanges(points []TrajectoryPoint) int {
	if len(points) < 3 {
		return 0
	}
	
	changes := 0
	var prevAngle float64
	
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		
		if math.Abs(dx) < 0.1 && math.Abs(dy) < 0.1 {
			continue
		}
		
		angle := math.Atan2(dy, dx)
		
		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi/4 {
				changes++
			}
		}
		
		prevAngle = angle
	}
	
	return changes
}

func (e *SliderSecurityEnhancer) calculateTotalDistance(points []TrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	
	var totalDistance float64
	
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance
	}
	
	return totalDistance
}

func (e *SliderSecurityEnhancer) calculateSpeedVariance(points []TrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	
	var totalSpeed float64
	for _, p := range points {
		totalSpeed += p.Speed
	}
	mean := totalSpeed / float64(len(points))
	
	var varianceSum float64
	for _, p := range points {
		diff := p.Speed - mean
		varianceSum += diff * diff
	}
	
	return varianceSum / float64(len(points))
}

func (e *SliderSecurityEnhancer) calculateHumanLikeScore(points []TrajectoryPoint, speedProfile SpeedProfile, accelProfile AccelerationProfile, directionChanges int) float64 {
	score := 1.0
	
	if speedProfile.AverageSpeed > e.maxSpeed {
		score -= 0.3
	}
	
	if accelProfile.MaxAcceleration > e.maxAcceleration {
		score -= 0.2
	}
	
	smoothness := accelProfile.Smoothness
	score *= (0.5 + 0.5*smoothness)
	
	normalChanges := float64(directionChanges) / float64(len(points))
	if normalChanges > 0.5 {
		score -= 0.2
	}
	
	if len(points) < 20 {
		score -= 0.1
	}
	
	timeSpan := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	if timeSpan < 500 {
		score -= 0.3
	} else if timeSpan > 30000 {
		score -= 0.1
	}
	
	speedVariance := e.calculateSpeedVariance(points)
	if speedVariance < 10 {
		score -= 0.3
	}
	
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (e *SliderSecurityEnhancer) determineRiskLevel(humanLikeScore float64, speedProfile SpeedProfile, accelProfile AccelerationProfile) string {
	if humanLikeScore > 0.8 && accelProfile.Smoothness > 0.7 {
		return "low"
	} else if humanLikeScore > 0.5 {
		return "medium"
	}
	return "high"
}

func (e *SliderSecurityEnhancer) detectAnomalyIndicators(points []TrajectoryPoint, speedProfile SpeedProfile, accelProfile AccelerationProfile, directionChanges int) []string {
	indicators := []string{}
	
	if len(points) < e.minTrajectoryPoints {
		indicators = append(indicators, "trajectory_too_short")
	}
	
	if speedProfile.AverageSpeed > e.maxSpeed {
		indicators = append(indicators, "abnormally_fast")
	}
	
	if accelProfile.MaxAcceleration > e.maxAcceleration {
		indicators = append(indicators, "sudden_acceleration")
	}
	
	if accelProfile.JerkCount > len(points)/5 {
		indicators = append(indicators, "jerky_movement")
	}
	
	if directionChanges > len(points)/3 {
		indicators = append(indicators, "too_many_direction_changes")
	}
	
	timeSpan := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	if timeSpan < 500 {
		indicators = append(indicators, "too_fast_completion")
	}
	
	speedVariance := e.calculateSpeedVariance(points)
	if speedVariance < 10 {
		indicators = append(indicators, "suspiciously_consistent_speed")
	}
	
	if len(indicators) == 0 {
		indicators = append(indicators, "normal_trajectory")
	}
	
	return indicators
}

type ClickCaptchaSecurityEnhancer struct {
	minClickPoints int
	maxClickSpeed int64
	enableZoneAnalysis bool
}

type EnhancedClickAnalysis struct {
	ClickPoints       []ClickPoint
	TotalClicks       int
	ClickTimeSpan     int64
	AverageInterval   int64
	IntervalVariance  float64
	ZoneDistribution  map[string]int
	ClickPattern      string
	IsHumanLike       bool
	Confidence        float64
	RiskLevel         string
	AnomalyIndicators []string
}

type ClickPoint struct {
	X         int
	Y         int
	Timestamp int64
	Zone      string
}

func NewClickCaptchaSecurityEnhancer() *ClickCaptchaSecurityEnhancer {
	return &ClickCaptchaSecurityEnhancer{
		minClickPoints:     3,
		maxClickSpeed:      100,
		enableZoneAnalysis: true,
	}
}

func (e *ClickCaptchaSecurityEnhancer) AnalyzeClickPattern(clicks []ClickPoint) *EnhancedClickAnalysis {
	if len(clicks) < e.minClickPoints {
		return &EnhancedClickAnalysis{
			IsHumanLike:       false,
			Confidence:        0,
			RiskLevel:         "high",
			AnomalyIndicators: []string{"insufficient_clicks"},
		}
	}
	
	timeSpan := clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp
	
	intervals := e.calculateIntervals(clicks)
	avgInterval := e.calculateAverage(intervals)
	intervalVariance := e.calculateVariance(intervals)
	
	zoneDistribution := e.analyzeZoneDistribution(clicks)
	
	pattern := e.identifyClickPattern(clicks, avgInterval)
	
	humanLikeScore := e.calculateHumanLikeScore(clicks, avgInterval, intervalVariance)
	
	riskLevel := e.determineRiskLevel(humanLikeScore)
	
	anomalyIndicators := e.detectAnomalyIndicators(clicks, avgInterval, intervalVariance)
	
	return &EnhancedClickAnalysis{
		ClickPoints:       clicks,
		TotalClicks:       len(clicks),
		ClickTimeSpan:     timeSpan,
		AverageInterval:   avgInterval,
		IntervalVariance:  intervalVariance,
		ZoneDistribution: zoneDistribution,
		ClickPattern:     pattern,
		IsHumanLike:      humanLikeScore > 0.5,
		Confidence:       humanLikeScore * 100,
		RiskLevel:        riskLevel,
		AnomalyIndicators: anomalyIndicators,
	}
}

func (e *ClickCaptchaSecurityEnhancer) calculateIntervals(clicks []ClickPoint) []int64 {
	intervals := make([]int64, 0, len(clicks)-1)
	
	for i := 1; i < len(clicks); i++ {
		interval := clicks[i].Timestamp - clicks[i-1].Timestamp
		intervals = append(intervals, interval)
	}
	
	return intervals
}

func (e *ClickCaptchaSecurityEnhancer) calculateAverage(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	
	var total int64
	for _, v := range values {
		total += v
	}
	
	return total / int64(len(values))
}

func (e *ClickCaptchaSecurityEnhancer) calculateVariance(values []int64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	mean := float64(e.calculateAverage(values))
	
	var varianceSum float64
	for _, v := range values {
		diff := float64(v) - mean
		varianceSum += diff * diff
	}
	
	return varianceSum / float64(len(values))
}

func (e *ClickCaptchaSecurityEnhancer) analyzeZoneDistribution(clicks []ClickPoint) map[string]int {
	zones := make(map[string]int)
	
	for _, click := range clicks {
		zone := e.getZone(click.X, click.Y)
		zones[zone]++
	}
	
	return zones
}

func (e *ClickCaptchaSecurityEnhancer) getZone(x, y int) string {
	zoneX := x / 100
	zoneY := y / 100
	return fmt.Sprintf("zone_%d_%d", zoneX, zoneY)
}

func (e *ClickCaptchaSecurityEnhancer) identifyClickPattern(clicks []ClickPoint, avgInterval int64) string {
	if len(clicks) < 2 {
		return "insufficient_data"
	}
	
	if avgInterval < 200 {
		return "rapid"
	} else if avgInterval < 1000 {
		return "normal"
	}
	return "slow"
}

func (e *ClickCaptchaSecurityEnhancer) calculateHumanLikeScore(clicks []ClickPoint, avgInterval int64, variance float64) float64 {
	score := 1.0
	
	if avgInterval < e.maxClickSpeed {
		score -= 0.4
	}
	
	if variance < 100 {
		score -= 0.3
	}
	
	if len(clicks) < 3 {
		score -= 0.2
	}
	
	timeSpan := float64(clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp)
	if timeSpan < 300 {
		score -= 0.3
	}
	
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (e *ClickCaptchaSecurityEnhancer) determineRiskLevel(humanLikeScore float64) string {
	if humanLikeScore > 0.7 {
		return "low"
	} else if humanLikeScore > 0.4 {
		return "medium"
	}
	return "high"
}

func (e *ClickCaptchaSecurityEnhancer) detectAnomalyIndicators(clicks []ClickPoint, avgInterval int64, variance float64) []string {
	indicators := []string{}
	
	if len(clicks) < e.minClickPoints {
		indicators = append(indicators, "too_few_clicks")
	}
	
	if avgInterval < e.maxClickSpeed {
		indicators = append(indicators, "abnormally_fast_clicks")
	}
	
	if variance < 100 {
		indicators = append(indicators, "suspiciously_regular_intervals")
	}
	
	timeSpan := float64(clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp)
	if timeSpan < 300 {
		indicators = append(indicators, "completed_too_quickly")
	}
	
	if len(indicators) == 0 {
		indicators = append(indicators, "normal_pattern")
	}
	
	return indicators
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
