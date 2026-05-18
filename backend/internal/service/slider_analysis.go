package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
)

type SliderTrajectory struct {
	Points         []SliderPoint `json:"points"`
	TotalDistance  float64       `json:"total_distance"`
	AverageSpeed   float64       `json:"average_speed"`
	MaxSpeed       float64       `json:"max_speed"`
	MinSpeed       float64       `json:"min_speed"`
	SpeedVariance  float64       `json:"speed_variance"`
	PathEfficiency float64       `json:"path_efficiency"`
	IsBot          bool          `json:"is_bot"`
	Confidence     float64       `json:"confidence"`
}

type SliderPoint struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
}

type SliderAnalysisResult struct {
	Trajectory        *SliderTrajectory `json:"trajectory"`
	Features          *SliderFeatures   `json:"features"`
	AnomalyScore      float64           `json:"anomaly_score"`
	MLScore           float64           `json:"ml_score"`
	IsBot             bool              `json:"is_bot"`
	Confidence        float64           `json:"confidence"`
	RiskIndicators    []string          `json:"risk_indicators"`
	AnomalyDetections []string          `json:"anomaly_detections"`
	TrajectoryPattern string            `json:"trajectory_pattern"`
	SpeedProfile      string            `json:"speed_profile"`
	OverallRiskScore  float64           `json:"overall_risk_score"`
}

type SliderFeatures struct {
	TotalDistance        float64   `json:"total_distance"`
	DirectDistance       float64   `json:"direct_distance"`
	PathEfficiency       float64   `json:"path_efficiency"`
	AverageSpeed         float64   `json:"average_speed"`
	MaxSpeed             float64   `json:"max_speed"`
	MinSpeed             float64   `json:"min_speed"`
	SpeedVariance        float64   `json:"speed_variance"`
	SpeedConsistency     float64   `json:"speed_consistency"`
	AverageAcceleration  float64   `json:"average_acceleration"`
	AccelerationVariance float64   `json:"acceleration_variance"`
	CurvatureAverage     float64   `json:"curvature_average"`
	CurvatureVariance    float64   `json:"curvature_variance"`
	CurvatureMax         float64   `json:"curvature_max"`
	DirectionChanges     int       `json:"direction_changes"`
	MicroCorrections     int       `json:"micro_corrections"`
	BacktrackCount       int       `json:"backtrack_count"`
	BacktrackDistance    float64   `json:"backtrack_distance"`
	PauseCount           int       `json:"pause_count"`
	TotalPauseDuration   float64   `json:"total_pause_duration"`
	HoverCount           int       `json:"hover_count"`
	HoverDurationTotal   float64   `json:"hover_duration_total"`
	StartDelay           int64     `json:"start_delay"`
	ResponseTime         int64     `json:"response_time"`
	TotalDuration        int64     `json:"total_duration"`
	SpeedDistribution    []float64 `json:"speed_distribution"`
	AngleDistribution    []float64 `json:"angle_distribution"`
	JitterScore          float64   `json:"jitter_score"`
	SmoothnessScore      float64   `json:"smoothness_score"`
	TrajectoryEntropy    float64   `json:"trajectory_entropy"`
	VelocityProfile      []float64 `json:"velocity_profile"`
	AccelerationProfile  []float64 `json:"acceleration_profile"`
	FourierFrequency     float64   `json:"fourier_frequency"`
	FourierEnergy        float64   `json:"fourier_energy"`
	FractalDimension     float64   `json:"fractal_dimension"`
	HumanLikenessScore   float64   `json:"human_likeness_score"`
}

type SliderAnalyzer struct {
	model *SliderMLModel
}

type SliderMLModel struct {
	weights          []float64
	bias             float64
	humanTemplates   [][]float64
	botTemplates     [][]float64
	isTrained        bool
	featureExtractor *SliderFeatureExtractor
}

type SliderFeatureExtractor struct{}

func NewSliderAnalyzer() *SliderAnalyzer {
	return &SliderAnalyzer{
		model: NewSliderMLModel(),
	}
}

func NewSliderMLModel() *SliderMLModel {
	return &SliderMLModel{
		weights:          make([]float64, 30),
		bias:             -15.0,
		isTrained:        false,
		featureExtractor: NewSliderFeatureExtractor(),
	}
}

func NewSliderFeatureExtractor() *SliderFeatureExtractor {
	return &SliderFeatureExtractor{}
}

func (sa *SliderAnalyzer) AnalyzeSliderTrajectory(trajectory []SliderPoint, targetPosition int) (*SliderAnalysisResult, error) {
	if len(trajectory) < 3 {
		return &SliderAnalysisResult{
			IsBot:          true,
			Confidence:     0.9,
			RiskIndicators: []string{"轨迹数据点不足"},
		}, nil
	}

	result := &SliderAnalysisResult{
		Trajectory:        sa.analyzeTrajectoryBasic(trajectory, targetPosition),
		RiskIndicators:    make([]string, 0),
		AnomalyDetections: make([]string, 0),
	}

	featureExtractor := NewSliderFeatureExtractor()
	result.Features = featureExtractor.ExtractFeatures(trajectory, targetPosition, result.Trajectory)

	result.AnomalyScore = sa.detectAnomalies(result.Features)

	result.TrajectoryPattern = sa.classifyTrajectoryPattern(result.Features)

	result.SpeedProfile = sa.classifySpeedProfile(result.Features)

	result.MLScore = sa.model.Predict(result.Features)

	result.OverallRiskScore = sa.calculateOverallRiskScore(result)

	result.IsBot = result.OverallRiskScore > 0.5
	result.Confidence = sa.calculateConfidence(result)

	return result, nil
}

func (sa *SliderAnalyzer) analyzeTrajectoryBasic(trajectory []SliderPoint, targetPosition int) *SliderTrajectory {
	if len(trajectory) < 2 {
		return &SliderTrajectory{IsBot: true, Confidence: 0.9}
	}

	sliderTraj := &SliderTrajectory{
		Points: trajectory,
	}

	totalDistance := 0.0
	speeds := make([]float64, 0)
	maxSpeed := 0.0
	minSpeed := math.MaxFloat64
	directionChanges := 0
	var prevAngle float64

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance

		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			speed := distance / dt * 1000
			speeds = append(speeds, speed)
			if speed > maxSpeed {
				maxSpeed = speed
			}
			if speed < minSpeed {
				minSpeed = speed
			}
		}

		if i > 1 {
			angle := math.Atan2(dy, dx)
			if i > 2 {
				angleDiff := math.Abs(angle - prevAngle)
				if angleDiff > math.Pi {
					angleDiff = 2*math.Pi - angleDiff
				}
				if angleDiff > 0.5 {
					directionChanges++
				}
			}
			prevAngle = angle
		}
	}

	sliderTraj.TotalDistance = totalDistance

	if len(speeds) > 0 {
		sum := 0.0
		for _, s := range speeds {
			sum += s
		}
		sliderTraj.AverageSpeed = sum / float64(len(speeds))

		variance := 0.0
		for _, s := range speeds {
			variance += (s - sliderTraj.AverageSpeed) * (s - sliderTraj.AverageSpeed)
		}
		variance /= float64(len(speeds))
		sliderTraj.SpeedVariance = variance
	}

	sliderTraj.MaxSpeed = maxSpeed
	sliderTraj.MinSpeed = minSpeed

	startX := float64(trajectory[0].X)
	endX := float64(trajectory[len(trajectory)-1].X)
	directDistance := math.Abs(endX - startX)

	if totalDistance > 0 {
		sliderTraj.PathEfficiency = directDistance / totalDistance
	}

	if len(trajectory) >= 2 {
		sliderTraj.IsBot = sa.isTrajectoryBotLike(trajectory, sliderTraj)
		sliderTraj.Confidence = sa.calculateTrajectoryConfidence(trajectory, sliderTraj)
	}

	return sliderTraj
}

func (sa *SliderAnalyzer) isTrajectoryBotLike(trajectory []SliderPoint, sliderTraj *SliderTrajectory) bool {
	if sliderTraj.PathEfficiency > 0.98 {
		return true
	}

	if sliderTraj.AverageSpeed > 1500 {
		return true
	}

	if sliderTraj.SpeedVariance < 0.01 {
		return true
	}

	speeds := sa.extractSpeeds(trajectory)
	if len(speeds) > 3 {
		variance := sa.variance(speeds)
		mean := sa.mean(speeds)
		if mean > 0 && variance/mean < 0.05 {
			return true
		}
	}

	return false
}

func (sa *SliderAnalyzer) calculateTrajectoryConfidence(trajectory []SliderPoint, sliderTraj *SliderTrajectory) float64 {
	confidence := 0.5

	if sliderTraj.PathEfficiency > 0.8 && sliderTraj.PathEfficiency < 0.98 {
		confidence += 0.2
	}

	speeds := sa.extractSpeeds(trajectory)
	if len(speeds) > 5 {
		variance := sa.variance(speeds)
		mean := sa.mean(speeds)
		if mean > 0 {
			cv := variance / mean
			if cv > 0.2 && cv < 2.0 {
				confidence += 0.15
			}
		}
	}

	if len(trajectory) > 20 {
		confidence += 0.1
	}

	totalDuration := float64(trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp)
	if totalDuration > 500 && totalDuration < 10000 {
		confidence += 0.1
	}

	return math.Min(confidence, 0.99)
}

func (sa *SliderAnalyzer) extractSpeeds(trajectory []SliderPoint) []float64 {
	speeds := make([]float64, 0)
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt*1000)
		}
	}
	return speeds
}

func (sfe *SliderFeatureExtractor) ExtractFeatures(trajectory []SliderPoint, targetPosition int, sliderTraj *SliderTrajectory) *SliderFeatures {
	features := &SliderFeatures{}

	if len(trajectory) < 2 {
		return features
	}

	features.TotalDistance = sliderTraj.TotalDistance

	startX := float64(trajectory[0].X)
	endX := float64(trajectory[len(trajectory)-1].X)
	features.DirectDistance = math.Abs(endX - startX)

	if features.TotalDistance > 0 {
		features.PathEfficiency = features.DirectDistance / features.TotalDistance
	}

	speeds := sfe.extractSpeeds(trajectory)
	if len(speeds) > 0 {
		features.AverageSpeed = sfe.mean(speeds)
		features.MaxSpeed = sfe.max(speeds)
		features.MinSpeed = sfe.min(speeds)
		features.SpeedVariance = sfe.variance(speeds)
		features.SpeedConsistency = sfe.calculateSpeedConsistency(speeds)
		features.SpeedDistribution = sfe.calculateSpeedDistribution(speeds, 10)
	}

	accelerations := sfe.extractAccelerations(trajectory)
	if len(accelerations) > 0 {
		features.AverageAcceleration = sfe.mean(accelerations)
		features.AccelerationVariance = sfe.variance(accelerations)
		features.AccelerationProfile = sfe.calculateAccelerationProfile(accelerations, 5)
	}

	curvatures := sfe.extractCurvatures(trajectory)
	if len(curvatures) > 0 {
		features.CurvatureAverage = sfe.mean(curvatures)
		features.CurvatureVariance = sfe.variance(curvatures)
		features.CurvatureMax = sfe.max(curvatures)
	}

	features.DirectionChanges = sfe.countDirectionChanges(trajectory)
	features.MicroCorrections = sfe.countMicroCorrections(trajectory)
	features.BacktrackCount, features.BacktrackDistance = sfe.countBacktrack(trajectory)
	features.PauseCount, features.TotalPauseDuration = sfe.countPauses(trajectory)
	features.HoverCount, features.HoverDurationTotal = sfe.countHovers(trajectory)

	if len(trajectory) > 1 {
		features.StartDelay = trajectory[0].Timestamp
		features.ResponseTime = trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp
		features.TotalDuration = features.ResponseTime
	}

	features.AngleDistribution = sfe.calculateAngleDistribution(trajectory, 8)

	features.JitterScore = sfe.calculateJitterScore(trajectory)
	features.SmoothnessScore = sfe.calculateSmoothnessScore(trajectory)
	features.TrajectoryEntropy = sfe.calculateTrajectoryEntropy(trajectory)
	features.VelocityProfile = sfe.calculateVelocityProfile(trajectory, 10)

	features.FourierFrequency = sfe.calculateFourierFrequency(trajectory)
	features.FourierEnergy = sfe.calculateFourierEnergy(trajectory)
	features.FractalDimension = sfe.calculateFractalDimension(trajectory)

	features.HumanLikenessScore = sfe.calculateHumanLikeness(features)

	return features
}

func (sfe *SliderFeatureExtractor) extractSpeeds(trajectory []SliderPoint) []float64 {
	speeds := make([]float64, 0)
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt*1000)
		}
	}
	return speeds
}

func (sfe *SliderFeatureExtractor) extractAccelerations(trajectory []SliderPoint) []float64 {
	speeds := sfe.extractSpeeds(trajectory)
	accelerations := make([]float64, 0)
	for i := 2; i < len(speeds); i++ {
		dt := float64(trajectory[i+1].Timestamp-trajectory[i-1].Timestamp) / 2
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}
	return accelerations
}

func (sfe *SliderFeatureExtractor) extractCurvatures(trajectory []SliderPoint) []float64 {
	curvatures := make([]float64, 0)
	for i := 1; i < len(trajectory)-1; i++ {
		v1x := float64(trajectory[i].X - trajectory[i-1].X)
		v1y := float64(trajectory[i].Y - trajectory[i-1].Y)
		v2x := float64(trajectory[i+1].X - trajectory[i].X)
		v2y := float64(trajectory[i+1].Y - trajectory[i].Y)

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
			curvatures = append(curvatures, math.Abs(angle))
		}
	}
	return curvatures
}

func (sfe *SliderFeatureExtractor) countDirectionChanges(trajectory []SliderPoint) int {
	if len(trajectory) < 3 {
		return 0
	}

	changes := 0
	var prevAngle float64

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		angle := math.Atan2(dy, dx)

		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 0.5 {
				changes++
			}
		}
		prevAngle = angle
	}

	return changes
}

func (sfe *SliderFeatureExtractor) countMicroCorrections(trajectory []SliderPoint) int {
	if len(trajectory) < 3 {
		return 0
	}

	corrections := 0
	for i := 2; i < len(trajectory); i++ {
		dx1 := float64(trajectory[i-1].X - trajectory[i-2].X)
		dy1 := float64(trajectory[i-1].Y - trajectory[i-2].Y)
		dx2 := float64(trajectory[i].X - trajectory[i-1].X)
		dy2 := float64(trajectory[i].Y - trajectory[i-1].Y)

		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle < 0.9 && cosAngle > -0.9 {
				angle := math.Acos(cosAngle)
				if angle > 0.1 && angle < 0.5 {
					corrections++
				}
			}
		}
	}

	return corrections
}

func (sfe *SliderFeatureExtractor) countBacktrack(trajectory []SliderPoint) (int, float64) {
	if len(trajectory) < 2 {
		return 0, 0
	}

	backtracks := 0
	backtrackDistance := 0.0
	maxX := trajectory[0].X

	for i := 1; i < len(trajectory); i++ {
		if trajectory[i].X > maxX {
			maxX = trajectory[i].X
		} else if maxX-trajectory[i].X > 5 {
			backtracks++
			backtrackDistance += float64(maxX - trajectory[i].X)
		}
	}

	return backtracks, backtrackDistance
}

func (sfe *SliderFeatureExtractor) countPauses(trajectory []SliderPoint) (int, float64) {
	if len(trajectory) < 2 {
		return 0, 0
	}

	pauses := 0
	totalDuration := 0.0

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)

		if distance < 3 && dt > 100 {
			pauses++
			totalDuration += dt
		}
	}

	return pauses, totalDuration
}

func (sfe *SliderFeatureExtractor) countHovers(trajectory []SliderPoint) (int, float64) {
	if len(trajectory) < 2 {
		return 0, 0
	}

	hovers := 0
	totalDuration := 0.0
	hoverStart := -1

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)

		if distance < 5 && dt > 50 {
			if hoverStart == -1 {
				hoverStart = i - 1
			}
		} else {
			if hoverStart != -1 {
				hovers++
				totalDuration += float64(trajectory[i-1].Timestamp - trajectory[hoverStart].Timestamp)
				hoverStart = -1
			}
		}
	}

	if hoverStart != -1 {
		hovers++
		totalDuration += float64(trajectory[len(trajectory)-1].Timestamp - trajectory[hoverStart].Timestamp)
	}

	return hovers, totalDuration
}

func (sfe *SliderFeatureExtractor) calculateSpeedDistribution(speeds []float64, bins int) []float64 {
	if len(speeds) == 0 || bins <= 0 {
		return make([]float64, bins)
	}

	distribution := make([]float64, bins)
	minSpeed := sfe.min(speeds)
	maxSpeed := sfe.max(speeds)

	if maxSpeed <= minSpeed {
		return distribution
	}

	binWidth := (maxSpeed - minSpeed) / float64(bins)
	for _, speed := range speeds {
		bin := int((speed - minSpeed) / binWidth)
		if bin >= bins {
			bin = bins - 1
		}
		if bin < 0 {
			bin = 0
		}
		distribution[bin]++
	}

	total := float64(len(speeds))
	for i := range distribution {
		distribution[i] /= total
	}

	return distribution
}

func (sfe *SliderFeatureExtractor) calculateSpeedConsistency(speeds []float64) float64 {
	if len(speeds) < 2 {
		return 0
	}

	mean := sfe.mean(speeds)
	if mean == 0 {
		return 0
	}

	variance := sfe.variance(speeds)
	cv := math.Sqrt(variance) / mean

	return 1.0 - math.Min(cv, 1.0)
}

func (sfe *SliderFeatureExtractor) calculateAccelerationProfile(accelerations []float64, segments int) []float64 {
	if len(accelerations) < segments {
		profile := make([]float64, segments)
		mean := sfe.mean(accelerations)
		for i := range profile {
			profile[i] = mean
		}
		return profile
	}

	profile := make([]float64, segments)
	segmentSize := len(accelerations) / segments

	for i := 0; i < segments; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if i == segments-1 {
			end = len(accelerations)
		}

		segment := accelerations[start:end]
		profile[i] = sfe.mean(segment)
	}

	return profile
}

func (sfe *SliderFeatureExtractor) calculateAngleDistribution(trajectory []SliderPoint, bins int) []float64 {
	if len(trajectory) < 2 || bins <= 0 {
		return make([]float64, bins)
	}

	distribution := make([]float64, bins)
	binSize := 2 * math.Pi / float64(bins)

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		angle := math.Atan2(dy, dx)
		if angle < 0 {
			angle += 2 * math.Pi
		}

		bin := int(angle / binSize)
		if bin >= bins {
			bin = bins - 1
		}
		distribution[bin]++
	}

	total := float64(len(trajectory) - 1)
	for i := range distribution {
		distribution[i] /= total
	}

	return distribution
}

func (sfe *SliderFeatureExtractor) calculateJitterScore(trajectory []SliderPoint) float64 {
	if len(trajectory) < 3 {
		return 0
	}

	smoothed := sfe.smoothTrajectory(trajectory, 3)
	totalJitter := 0.0

	for i := 1; i < len(trajectory); i++ {
		dx1 := float64(trajectory[i].X - trajectory[i-1].X)
		dy1 := float64(trajectory[i].Y - trajectory[i-1].Y)
		dx2 := float64(smoothed[i].X - smoothed[i-1].X)
		dy2 := float64(smoothed[i].Y - smoothed[i-1].Y)

		distance1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		distance2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if distance1 > 0 {
			totalJitter += math.Abs(distance1-distance2) / distance1
		}
	}

	return totalJitter / float64(len(trajectory)-1)
}

func (sfe *SliderFeatureExtractor) smoothTrajectory(trajectory []SliderPoint, windowSize int) []SliderPoint {
	if len(trajectory) < windowSize {
		return trajectory
	}

	if windowSize%2 == 0 {
		windowSize++
	}

	halfWindow := windowSize / 2
	smoothed := make([]SliderPoint, len(trajectory))

	for i := range trajectory {
		start := i - halfWindow
		end := i + halfWindow

		if start < 0 {
			start = 0
		}
		if end >= len(trajectory) {
			end = len(trajectory) - 1
		}

		sumX := 0
		sumY := 0
		count := 0

		for j := start; j <= end; j++ {
			sumX += trajectory[j].X
			sumY += trajectory[j].Y
			count++
		}

		smoothed[i] = trajectory[i]
		smoothed[i].X = sumX / count
		smoothed[i].Y = sumY / count
	}

	return smoothed
}

func (sfe *SliderFeatureExtractor) calculateSmoothnessScore(trajectory []SliderPoint) float64 {
	if len(trajectory) < 3 {
		return 0
	}

	totalAngleChange := 0.0
	count := 0

	for i := 1; i < len(trajectory)-1; i++ {
		v1x := float64(trajectory[i].X - trajectory[i-1].X)
		v1y := float64(trajectory[i].Y - trajectory[i-1].Y)
		v2x := float64(trajectory[i+1].X - trajectory[i].X)
		v2y := float64(trajectory[i+1].Y - trajectory[i].Y)

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

func (sfe *SliderFeatureExtractor) calculateTrajectoryEntropy(trajectory []SliderPoint) float64 {
	if len(trajectory) < 2 {
		return 0
	}

	buckets := 20
	bucketCounts := make([]int, buckets)
	totalDistance := 0.0

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance
	}

	if totalDistance == 0 {
		return 0
	}

	for i := 1; i < len(trajectory); i++ {
		progress := 0.0
		dist := 0.0

		for j := 1; j <= i; j++ {
			dx := float64(trajectory[j].X - trajectory[j-1].X)
			dy := float64(trajectory[j].Y - trajectory[j-1].Y)
			dist += math.Sqrt(dx*dx + dy*dy)
		}

		progress = dist / totalDistance
		bucket := int(progress * float64(buckets))
		if bucket >= buckets {
			bucket = buckets - 1
		}
		bucketCounts[bucket]++
	}

	entropy := 0.0
	for _, count := range bucketCounts {
		if count > 0 {
			p := float64(count) / float64(len(trajectory))
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (sfe *SliderFeatureExtractor) calculateVelocityProfile(trajectory []SliderPoint, segments int) []float64 {
	if len(trajectory) < segments {
		profile := make([]float64, segments)
		return profile
	}

	profile := make([]float64, segments)
	segmentSize := len(trajectory) / segments

	for i := 0; i < segments; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if i == segments-1 {
			end = len(trajectory)
		}

		speeds := sfe.extractSpeedsFromSegment(trajectory[start:end])
		profile[i] = sfe.mean(speeds)
	}

	return profile
}

func (sfe *SliderFeatureExtractor) extractSpeedsFromSegment(segment []SliderPoint) []float64 {
	speeds := make([]float64, 0)
	for i := 1; i < len(segment); i++ {
		dx := float64(segment[i].X - segment[i-1].X)
		dy := float64(segment[i].Y - segment[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(segment[i].Timestamp - segment[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt*1000)
		}
	}
	return speeds
}

func (sfe *SliderFeatureExtractor) calculateFourierFrequency(trajectory []SliderPoint) float64 {
	if len(trajectory) < 8 {
		return 0
	}

	n := len(trajectory)
	for n&(n-1) != 0 {
		n--
	}
	if n < 8 {
		return 0
	}

	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = float64(trajectory[i].X)
	}

	fft := sfe.fft(x)
	maxMag := 0.0
	dominantIdx := 0

	for i := 1; i < n/2; i++ {
		mag := math.Sqrt(real(fft[i])*real(fft[i]) + imag(fft[i])*imag(fft[i]))
		if mag > maxMag {
			maxMag = mag
			dominantIdx = i
		}
	}

	totalTime := float64(trajectory[n-1].Timestamp - trajectory[0].Timestamp)
	if totalTime > 0 {
		return float64(dominantIdx) / totalTime * 1000
	}

	return 0
}

func (sfe *SliderFeatureExtractor) fft(x []float64) []complex128 {
	n := len(x)
	if n <= 1 {
		result := make([]complex128, n)
		for i, val := range x {
			result[i] = complex(val, 0)
		}
		return result
	}

	even := make([]float64, n/2)
	odd := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	fftEven := sfe.fft(even)
	fftOdd := sfe.fft(odd)

	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		theta := -2 * math.Pi * float64(k) / float64(n)
		t := complex(math.Cos(theta), math.Sin(theta)) * fftOdd[k]
		result[k] = complex(real(fftEven[k])+real(t), imag(fftEven[k])+imag(t))
		result[k+n/2] = complex(real(fftEven[k])-real(t), imag(fftEven[k])-imag(t))
	}

	return result
}

func (sfe *SliderFeatureExtractor) calculateFourierEnergy(trajectory []SliderPoint) float64 {
	if len(trajectory) < 8 {
		return 0
	}

	n := len(trajectory)
	for n&(n-1) != 0 {
		n--
	}
	if n < 8 {
		return 0
	}

	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = float64(trajectory[i].X)
	}

	fft := sfe.fft(x)
	totalEnergy := 0.0

	for i := 1; i < n/2; i++ {
		mag := real(fft[i])*real(fft[i]) + imag(fft[i])*imag(fft[i])
		totalEnergy += mag
	}

	return totalEnergy
}

func (sfe *SliderFeatureExtractor) calculateFractalDimension(trajectory []SliderPoint) float64 {
	if len(trajectory) < 10 {
		return 1.0
	}

	minX, maxX := trajectory[0].X, trajectory[0].X
	minY, maxY := trajectory[0].Y, trajectory[0].Y

	for _, p := range trajectory {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	width := maxX - minX
	height := maxY - minY

	if width == 0 && height == 0 {
		return 1.0
	}

	maxScale := 5
	logScales := make([]float64, maxScale)
	logCounts := make([]float64, maxScale)

	for scale := 0; scale < maxScale; scale++ {
		boxSize := int(math.Pow(2, float64(maxScale-scale)))
		grid := make(map[string]bool)

		for _, p := range trajectory {
			gx := (p.X - minX) / boxSize
			gy := (p.Y - minY) / boxSize
			key := fmt.Sprintf("%d,%d", gx, gy)
			grid[key] = true
		}

		logScales[scale] = math.Log(1.0 / float64(boxSize))
		logCounts[scale] = math.Log(float64(len(grid)))
	}

	return sfe.linearRegression(logScales, logCounts)
}

func (sfe *SliderFeatureExtractor) linearRegression(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return 1.0
	}

	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denominator := float64(n)*sumX2 - sumX*sumX
	if denominator == 0 {
		return 1.0
	}

	return (float64(n)*sumXY - sumX*sumY) / denominator
}

func (sfe *SliderFeatureExtractor) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (sfe *SliderFeatureExtractor) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := sfe.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

func (sfe *SliderFeatureExtractor) max(values []float64) float64 {
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

func (sfe *SliderFeatureExtractor) min(values []float64) float64 {
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

func (sfe *SliderFeatureExtractor) calculateHumanLikeness(features *SliderFeatures) float64 {
	score := 0.5

	if features.PathEfficiency > 0.7 && features.PathEfficiency < 0.98 {
		score += 0.15
	} else if features.PathEfficiency >= 0.98 {
		score -= 0.3
	}

	if features.SpeedConsistency > 0.3 && features.SpeedConsistency < 0.9 {
		score += 0.15
	} else if features.SpeedConsistency <= 0.1 {
		score -= 0.2
	}

	if features.MicroCorrections > 2 && features.MicroCorrections < 20 {
		score += 0.1
	}

	if features.PauseCount > 0 && features.PauseCount < 10 {
		score += 0.1
	}

	if features.CurvatureAverage > 0.01 && features.CurvatureAverage < 0.5 {
		score += 0.1
	}

	if features.JitterScore > 0.01 && features.JitterScore < 0.3 {
		score += 0.1
	}

	if features.SmoothnessScore > 0.3 && features.SmoothnessScore < 0.9 {
		score += 0.1
	}

	if features.BacktrackCount > 0 && features.BacktrackCount < 5 {
		score += 0.05
	}

	return math.Max(0, math.Min(1, score))
}

func (sm *SliderMLModel) Predict(features *SliderFeatures) float64 {
	if features == nil {
		return 0.5
	}

	featureVector := sm.extractFeatureVector(features)

	if !sm.isTrained && len(sm.humanTemplates) == 0 {
		return sm.simplePredict(features)
	}

	score := sm.bias
	for i, val := range featureVector {
		if i < len(sm.weights) {
			score += sm.weights[i] * val
		}
	}

	return 1.0 / (1.0 + math.Exp(-score))
}

func (sm *SliderMLModel) extractFeatureVector(features *SliderFeatures) []float64 {
	vector := make([]float64, 30)

	vector[0] = features.PathEfficiency
	vector[1] = features.SpeedConsistency
	vector[2] = math.Min(features.AverageSpeed/2000, 1.0)
	vector[3] = math.Min(features.MaxSpeed/3000, 1.0)
	vector[4] = math.Min(features.SpeedVariance, 1.0)
	vector[5] = math.Min(features.CurvatureAverage, 1.0)
	vector[6] = math.Min(features.CurvatureVariance, 1.0)
	vector[7] = math.Min(float64(features.DirectionChanges)/20, 1.0)
	vector[8] = math.Min(float64(features.MicroCorrections)/30, 1.0)
	vector[9] = math.Min(float64(features.BacktrackCount)/10, 1.0)
	vector[10] = math.Min(features.BacktrackDistance/100, 1.0)
	vector[11] = math.Min(float64(features.PauseCount)/20, 1.0)
	vector[12] = math.Min(features.TotalPauseDuration/1000, 1.0)
	vector[13] = math.Min(float64(features.HoverCount)/20, 1.0)
	vector[14] = math.Min(features.HoverDurationTotal/1000, 1.0)
	vector[15] = float64(features.StartDelay) / 5000
	vector[16] = math.Min(float64(features.ResponseTime)/10000, 1.0)
	vector[17] = math.Min(features.JitterScore, 1.0)
	vector[18] = features.SmoothnessScore
	vector[19] = features.TrajectoryEntropy / 4.0
	vector[20] = math.Min(features.FourierFrequency/10, 1.0)
	vector[21] = math.Min(features.FourierEnergy/10000, 1.0)
	vector[22] = math.Min(features.FractalDimension/2, 1.0)
	vector[23] = features.HumanLikenessScore
	vector[24] = math.Min(features.AverageAcceleration, 1.0)
	vector[25] = math.Min(features.AccelerationVariance, 1.0)
	vector[26] = math.Min(features.CurvatureMax, 1.0)
	vector[27] = math.Min(features.MinSpeed/1000, 1.0)
	vector[28] = math.Min(features.DirectDistance/500, 1.0)
	vector[29] = math.Min(features.TotalDistance/1000, 1.0)

	return vector
}

func (sm *SliderMLModel) simplePredict(features *SliderFeatures) float64 {
	score := 0.0

	if features.PathEfficiency > 0.98 {
		score += 0.3
	}

	if features.SpeedConsistency > 0.95 {
		score += 0.2
	}

	if features.AverageSpeed > 1500 {
		score += 0.2
	}

	if features.MicroCorrections == 0 && len(sm.getTrajectoryPoints()) > 20 {
		score += 0.15
	}

	if features.PauseCount == 0 && features.TotalDuration > 1000 {
		score += 0.15
	}

	score += (1.0 - features.HumanLikenessScore) * 0.5

	return math.Max(0, math.Min(1, score))
}

func (sm *SliderMLModel) getTrajectoryPoints() []SliderPoint {
	return []SliderPoint{}
}

func (sa *SliderAnalyzer) detectAnomalies(features *SliderFeatures) float64 {
	anomalyScore := 0.0
	anomalyCount := 0

	if features.PathEfficiency > 0.98 {
		anomalyScore += 0.2
		anomalyCount++
	}

	if features.SpeedConsistency > 0.98 {
		anomalyScore += 0.15
		anomalyCount++
	}

	if features.AverageSpeed > 2000 {
		anomalyScore += 0.15
		anomalyCount++
	}

	if features.MicroCorrections == 0 && features.TotalDuration > 500 {
		anomalyScore += 0.1
		anomalyCount++
	}

	if features.PauseCount == 0 && features.TotalDuration > 2000 {
		anomalyScore += 0.1
		anomalyCount++
	}

	if features.BacktrackCount > 5 {
		anomalyScore += 0.05
		anomalyCount++
	}

	if features.CurvatureVariance < 0.001 {
		anomalyScore += 0.1
		anomalyCount++
	}

	if features.SmoothnessScore > 0.95 {
		anomalyScore += 0.1
		anomalyCount++
	}

	if features.HumanLikenessScore < 0.2 {
		anomalyScore += 0.15
		anomalyCount++
	}

	if features.FractalDimension < 1.1 {
		anomalyScore += 0.1
		anomalyCount++
	}

	if anomalyCount > 0 {
		return anomalyScore / float64(anomalyCount)
	}

	return 0.0
}

func (sa *SliderAnalyzer) classifyTrajectoryPattern(features *SliderFeatures) string {
	if features.PathEfficiency > 0.98 {
		return "perfect_straight"
	}

	if features.PathEfficiency > 0.9 {
		return "near_straight"
	}

	if features.BacktrackCount > 3 {
		return "erratic"
	}

	if features.PauseCount > 5 {
		return "hesitant"
	}

	if features.DirectionChanges > 10 {
		return "curved"
	}

	return "normal"
}

func (sa *SliderAnalyzer) classifySpeedProfile(features *SliderFeatures) string {
	if features.AverageSpeed > 2000 {
		return "extremely_fast"
	}

	if features.AverageSpeed > 1500 {
		return "very_fast"
	}

	if features.AverageSpeed > 800 {
		return "fast"
	}

	if features.AverageSpeed > 300 {
		return "normal"
	}

	if features.AverageSpeed > 100 {
		return "slow"
	}

	return "very_slow"
}

func (sa *SliderAnalyzer) calculateOverallRiskScore(result *SliderAnalysisResult) float64 {
	riskScore := 0.0

	riskScore += result.AnomalyScore * 0.3

	riskScore += result.MLScore * 0.4

	if result.Trajectory.PathEfficiency > 0.98 {
		riskScore += 0.15
	}

	if result.Trajectory.AverageSpeed > 1500 {
		riskScore += 0.1
	}

	if result.Features.SpeedConsistency > 0.95 {
		riskScore += 0.1
	}

	if result.Features.MicroCorrections == 0 && result.Trajectory.TotalDistance > 100 {
		riskScore += 0.1
	}

	if result.Features.PauseCount == 0 && result.Features.TotalDuration > 1000 {
		riskScore += 0.05
	}

	if result.Features.HumanLikenessScore < 0.3 {
		riskScore += 0.15
	}

	return math.Min(riskScore, 1.0)
}

func (sa *SliderAnalyzer) calculateConfidence(result *SliderAnalysisResult) float64 {
	confidence := 0.7

	if len(result.Trajectory.Points) > 30 {
		confidence += 0.1
	}

	if result.Features.TotalDuration > 500 && result.Features.TotalDuration < 10000 {
		confidence += 0.1
	}

	variance := sa.variance(sa.extractSpeeds(result.Trajectory.Points))
	mean := sa.mean(sa.extractSpeeds(result.Trajectory.Points))
	if mean > 0 && variance/mean > 0.1 {
		confidence += 0.05
	}

	if result.AnomalyScore > 0.3 || result.MLScore > 0.7 {
		confidence += 0.05
	}

	return math.Min(confidence, 0.99)
}

func (sa *SliderAnalyzer) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (sa *SliderAnalyzer) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := sa.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

func (sa *SliderAnalyzer) max(values []float64) float64 {
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

func (sa *SliderAnalyzer) min(values []float64) float64 {
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

type SliderTrajectoryValidator struct {
	minPoints        int
	maxDuration      int64
	minDistance      float64
	maxSpeed         float64
	allowedYVariance float64
}

func NewSliderTrajectoryValidator() *SliderTrajectoryValidator {
	return &SliderTrajectoryValidator{
		minPoints:        10,
		maxDuration:      30000,
		minDistance:      50,
		maxSpeed:         5000,
		allowedYVariance: 100,
	}
}

func (stv *SliderTrajectoryValidator) Validate(trajectory []SliderPoint) (bool, string) {
	if len(trajectory) < stv.minPoints {
		return false, fmt.Sprintf("轨迹数据点不足: 期望至少 %d 个点，实际 %d 个", stv.minPoints, len(trajectory))
	}

	if len(trajectory) >= 2 {
		duration := trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp
		if duration > stv.maxDuration {
			return false, fmt.Sprintf("轨迹持续时间过长: %d ms", duration)
		}

		if duration < 100 {
			return false, fmt.Sprintf("轨迹持续时间过短: %d ms", duration)
		}
	}

	totalDistance := 0.0
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	if totalDistance < stv.minDistance {
		return false, fmt.Sprintf("轨迹总距离过短: %.2f px", totalDistance)
	}

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)

		if dt > 0 {
			speed := distance / dt * 1000
			if speed > stv.maxSpeed {
				return false, fmt.Sprintf("检测到超高速移动: %.2f px/s", speed)
			}
		}
	}

	return true, "轨迹验证通过"
}

func (sa *SliderAnalyzer) GenerateReport(result *SliderAnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("=== 滑块轨迹分析报告 ===\n\n")

	sb.WriteString("基本轨迹信息:\n")
	sb.WriteString(fmt.Sprintf("  总距离: %.2f px\n", result.Trajectory.TotalDistance))
	sb.WriteString(fmt.Sprintf("  直接距离: %.2f px\n", result.Features.DirectDistance))
	sb.WriteString(fmt.Sprintf("  路径效率: %.4f\n", result.Trajectory.PathEfficiency))
	sb.WriteString(fmt.Sprintf("  总时长: %d ms\n", result.Features.TotalDuration))
	sb.WriteString(fmt.Sprintf("  数据点数: %d\n", len(result.Trajectory.Points)))

	sb.WriteString("\n速度分析:\n")
	sb.WriteString(fmt.Sprintf("  平均速度: %.2f px/s\n", result.Features.AverageSpeed))
	sb.WriteString(fmt.Sprintf("  最大速度: %.2f px/s\n", result.Features.MaxSpeed))
	sb.WriteString(fmt.Sprintf("  最小速度: %.2f px/s\n", result.Features.MinSpeed))
	sb.WriteString(fmt.Sprintf("  速度方差: %.6f\n", result.Features.SpeedVariance))
	sb.WriteString(fmt.Sprintf("  速度一致性: %.4f\n", result.Features.SpeedConsistency))
	sb.WriteString(fmt.Sprintf("  速度配置: %s\n", result.SpeedProfile))

	sb.WriteString("\n轨迹特征:\n")
	sb.WriteString(fmt.Sprintf("  方向变化: %d 次\n", result.Features.DirectionChanges))
	sb.WriteString(fmt.Sprintf("  微修正: %d 次\n", result.Features.MicroCorrections))
	sb.WriteString(fmt.Sprintf("  回退次数: %d 次\n", result.Features.BacktrackCount))
	sb.WriteString(fmt.Sprintf("  回退距离: %.2f px\n", result.Features.BacktrackDistance))
	sb.WriteString(fmt.Sprintf("  停顿次数: %d 次\n", result.Features.PauseCount))
	sb.WriteString(fmt.Sprintf("  停顿总时长: %.2f ms\n", result.Features.TotalPauseDuration))
	sb.WriteString(fmt.Sprintf("  悬停次数: %d 次\n", result.Features.HoverCount))

	sb.WriteString("\n曲率分析:\n")
	sb.WriteString(fmt.Sprintf("  平均曲率: %.6f\n", result.Features.CurvatureAverage))
	sb.WriteString(fmt.Sprintf("  曲率方差: %.6f\n", result.Features.CurvatureVariance))
	sb.WriteString(fmt.Sprintf("  最大曲率: %.6f\n", result.Features.CurvatureMax))

	sb.WriteString("\n高级特征:\n")
	sb.WriteString(fmt.Sprintf("  抖动分数: %.6f\n", result.Features.JitterScore))
	sb.WriteString(fmt.Sprintf("  平滑度分数: %.6f\n", result.Features.SmoothnessScore))
	sb.WriteString(fmt.Sprintf("  轨迹熵: %.4f\n", result.Features.TrajectoryEntropy))
	sb.WriteString(fmt.Sprintf("  傅里叶频率: %.4f\n", result.Features.FourierFrequency))
	sb.WriteString(fmt.Sprintf("  傅里叶能量: %.4f\n", result.Features.FourierEnergy))
	sb.WriteString(fmt.Sprintf("  分形维数: %.4f\n", result.Features.FractalDimension))
	sb.WriteString(fmt.Sprintf("  人类相似度: %.4f\n", result.Features.HumanLikenessScore))

	sb.WriteString("\n风险评估:\n")
	sb.WriteString(fmt.Sprintf("  异常分数: %.4f\n", result.AnomalyScore))
	sb.WriteString(fmt.Sprintf("  机器学习分数: %.4f\n", result.MLScore))
	sb.WriteString(fmt.Sprintf("  综合风险分数: %.4f\n", result.OverallRiskScore))
	sb.WriteString(fmt.Sprintf("  轨迹模式: %s\n", result.TrajectoryPattern))
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

func ParseSliderTrajectory(data []byte) ([]SliderPoint, error) {
	var points []SliderPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, err
	}
	return points, nil
}

func GenerateHumanLikeSliderTrajectory(startX, startY, endX, endY int, duration int64) []SliderPoint {
	trajectory := make([]SliderPoint, 0)

	numPoints := 30 + rand.Intn(20)
	interval := duration / int64(numPoints)

	for i := 0; i <= numPoints; i++ {
		t := float64(i) / float64(numPoints)

		baseX := startX + int(float64(endX-startX)*t)
		baseY := startY

		jitterX := rand.Intn(10) - 5
		jitterY := rand.Intn(10) - 5

		if rand.Float64() < 0.1 {
			jitterX += rand.Intn(20) - 10
			jitterY += rand.Intn(20) - 10
		}

		x := baseX + jitterX
		y := baseY + jitterY

		if i > 0 && rand.Float64() < 0.05 {
			pausePoints := rand.Intn(2) + 1
			for j := 0; j < pausePoints; j++ {
				trajectory = append(trajectory, SliderPoint{
					X:         x,
					Y:         y,
					Timestamp: int64(i)*interval + int64(j*50),
				})
			}
		}

		trajectory = append(trajectory, SliderPoint{
			X:         x,
			Y:         y,
			Timestamp: int64(i) * interval,
		})
	}

	return trajectory
}

func GenerateBotLikeSliderTrajectory(startX, startY, endX, endY int, duration int64) []SliderPoint {
	trajectory := make([]SliderPoint, 0)

	numPoints := 20 + rand.Intn(10)
	interval := duration / int64(numPoints)

	for i := 0; i <= numPoints; i++ {
		t := float64(i) / float64(numPoints)

		x := startX + int(float64(endX-startX)*t)
		y := startY

		trajectory = append(trajectory, SliderPoint{
			X:         x,
			Y:         y,
			Timestamp: int64(i) * interval,
		})
	}

	return trajectory
}

type DTWAnalyzer struct {
	windowSize int
}

func NewDTWAnalyzer() *DTWAnalyzer {
	return &DTWAnalyzer{
		windowSize: 10,
	}
}

func (dtw *DTWAnalyzer) ComputeDistance(traj1, traj2 []SliderPoint) float64 {
	if len(traj1) == 0 || len(traj2) == 0 {
		return math.MaxFloat64
	}

	n, m := len(traj1), len(traj2)
	dtwMatrix := make([][]float64, n+1)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m+1)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}
	dtwMatrix[0][0] = 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			dist := dtw.pointDistance(traj1[i-1], traj2[j-1])
			dtwMatrix[i][j] = dist + math.Min(math.Min(dtwMatrix[i-1][j], dtwMatrix[i][j-1]), dtwMatrix[i-1][j-1])
		}
	}

	return dtwMatrix[n][m]
}

func (dtw *DTWAnalyzer) pointDistance(p1, p2 SliderPoint) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func (dtw *DTWAnalyzer) ComputeSimilarity(traj1, traj2 []SliderPoint) float64 {
	distance := dtw.ComputeDistance(traj1, traj2)
	maxPossibleDistance := 1000.0
	similarity := 1.0 - math.Min(distance/maxPossibleDistance, 1.0)
	return math.Max(0, similarity)
}

type BotTrajectoryPattern struct {
	Name        string
	Description string
	Detector    func([]SliderPoint) bool
	Weight      float64
}

type BotPatternLibrary struct {
	patterns []BotTrajectoryPattern
}

func NewBotPatternLibrary() *BotPatternLibrary {
	return &BotPatternLibrary{
		patterns: []BotTrajectoryPattern{
			{
				Name:        "perfect_linear",
				Description: "完全直线轨迹，无任何偏差",
				Detector: func(traj []SliderPoint) bool {
					if len(traj) < 3 {
						return false
					}
					startX := float64(traj[0].X)
					endX := float64(traj[len(traj)-1].X)
					totalDist := 0.0
					for i := 1; i < len(traj); i++ {
						dx := float64(traj[i].X - traj[i-1].X)
						dy := float64(traj[i].Y - traj[i-1].Y)
						totalDist += math.Sqrt(dx*dx + dy*dy)
					}
					directDist := math.Abs(endX - startX)
					efficiency := directDist / totalDist
					return efficiency > 0.999
				},
				Weight: 0.4,
			},
			{
				Name:        "constant_speed",
				Description: "恒定速度移动",
				Detector: func(traj []SliderPoint) bool {
					if len(traj) < 5 {
						return false
					}
					speeds := make([]float64, 0)
					for i := 1; i < len(traj); i++ {
						dx := float64(traj[i].X - traj[i-1].X)
						dy := float64(traj[i].Y - traj[i-1].Y)
						dist := math.Sqrt(dx*dx + dy*dy)
						dt := float64(traj[i].Timestamp - traj[i-1].Timestamp)
						if dt > 0 {
							speeds = append(speeds, dist/dt*1000)
						}
					}
					if len(speeds) < 3 {
						return false
					}
					mean := 0.0
					for _, s := range speeds {
						mean += s
					}
					mean /= float64(len(speeds))
					variance := 0.0
					for _, s := range speeds {
						variance += (s - mean) * (s - mean)
					}
					variance /= float64(len(speeds))
					cv := math.Sqrt(variance) / mean
					return cv < 0.02 && mean > 100
				},
				Weight: 0.35,
			},
			{
				Name:        "instant_completion",
				Description: "瞬间完成轨迹，无延迟",
				Detector: func(traj []SliderPoint) bool {
					if len(traj) < 2 {
						return false
					}
					duration := traj[len(traj)-1].Timestamp - traj[0].Timestamp
					totalDist := 0.0
					for i := 1; i < len(traj); i++ {
						dx := float64(traj[i].X - traj[i-1].X)
						dy := float64(traj[i].Y - traj[i-1].Y)
						totalDist += math.Sqrt(dx*dx + dy*dy)
					}
					return duration < 500 && totalDist > 100
				},
				Weight: 0.3,
			},
			{
				Name:        "no_human_features",
				Description: "缺少人类行为特征",
				Detector: func(traj []SliderPoint) bool {
					if len(traj) < 10 {
						return false
					}
					hasPause := false
					hasCorrection := false
					hasYVariation := false

					for i := 1; i < len(traj); i++ {
						dx := float64(traj[i].X - traj[i-1].X)
						dy := float64(traj[i].Y - traj[i-1].Y)
						dist := math.Sqrt(dx*dx + dy*dy)
						dt := float64(traj[i].Timestamp - traj[i-1].Timestamp)

						if dist < 3 && dt > 100 {
							hasPause = true
						}

						if i > 1 {
							dx2 := float64(traj[i-1].X - traj[i-2].X)
							dy2 := float64(traj[i-1].Y - traj[i-2].Y)
							dot := dx*dx2 + dy*dy2
							mag1 := math.Sqrt(dx*dx + dy*dy)
							mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)
							if mag1 > 0 && mag2 > 0 {
								cosAngle := dot / (mag1 * mag2)
								if cosAngle < 0.95 && cosAngle > -0.95 {
									hasCorrection = true
								}
							}
						}

						if math.Abs(float64(traj[i].Y)-float64(traj[0].Y)) > 10 {
							hasYVariation = true
						}
					}

					return !hasPause && !hasCorrection && !hasYVariation
				},
				Weight: 0.25,
			},
			{
				Name:        "mechanical_movement",
				Description: "机械式移动，均匀采样",
				Detector: func(traj []SliderPoint) bool {
					if len(traj) < 5 {
						return false
					}
					intervals := make([]float64, 0)
					for i := 1; i < len(traj); i++ {
						dt := float64(traj[i].Timestamp - traj[i-1].Timestamp)
						if dt > 0 {
							intervals = append(intervals, dt)
						}
					}
					if len(intervals) < 3 {
						return false
					}
					mean := 0.0
					for _, t := range intervals {
						mean += t
					}
					mean /= float64(len(intervals))
					variance := 0.0
					for _, t := range intervals {
						variance += (t - mean) * (t - mean)
					}
					variance /= float64(len(intervals))
					cv := math.Sqrt(variance) / mean
					return cv < 0.05
				},
				Weight: 0.2,
			},
		},
	}
}

func (bpl *BotPatternLibrary) DetectPatterns(traj []SliderPoint) (float64, []string) {
	totalScore := 0.0
	detectedPatterns := make([]string, 0)

	for _, pattern := range bpl.patterns {
		if pattern.Detector(traj) {
			totalScore += pattern.Weight
			detectedPatterns = append(detectedPatterns, pattern.Name+": "+pattern.Description)
		}
	}

	return totalScore, detectedPatterns
}

func (sa *SliderAnalyzer) AnalyzeAdvancedFeatures(trajectory []SliderPoint, targetPosition int) map[string]float64 {
	features := make(map[string]float64)

	if len(trajectory) < 3 {
		return features
	}

	accelerations := sa.extractAccelerations(trajectory)
	if len(accelerations) > 0 {
		features["acceleration_mean"] = sa.mean(accelerations)
		features["acceleration_std"] = math.Sqrt(sa.variance(accelerations))
		features["acceleration_max"] = sa.max(accelerations)
		features["acceleration_min"] = sa.min(accelerations)

		posCount := 0
		negCount := 0
		for _, acc := range accelerations {
			if acc > 0 {
				posCount++
			} else {
				negCount++
			}
		}
		features["acceleration_pos_ratio"] = float64(posCount) / float64(len(accelerations))
		features["acceleration_balance"] = math.Abs(float64(posCount)-float64(negCount)) / float64(len(accelerations))
	}

	curvatures := sa.extractCurvatures(trajectory)
	if len(curvatures) > 0 {
		features["curvature_mean"] = sa.mean(curvatures)
		features["curvature_std"] = math.Sqrt(sa.variance(curvatures))
		features["curvature_max"] = sa.max(curvatures)

		significantCurvatures := 0
		for _, c := range curvatures {
			if c > 0.1 {
				significantCurvatures++
			}
		}
		features["curvature_significant_ratio"] = float64(significantCurvatures) / float64(len(curvatures))
	}

	jitter := sa.calculateJitterAdvanced(trajectory)
	features["jitter_score"] = jitter
	features["jitter_normalized"] = math.Min(jitter*10, 1.0)

	smoothness := sa.calculateSmoothnessAdvanced(trajectory)
	features["smoothness_score"] = smoothness

	fourier := sa.calculateFourierFeatures(trajectory)
	features["fourier_dominant_freq"] = fourier["dominant_freq"]
	features["fourier_energy"] = fourier["energy"]
	features["fourier_entropy"] = fourier["entropy"]

	fractal := sa.calculateFractalDimensionSimple(trajectory)
	features["fractal_dimension"] = fractal

	wavelet := sa.calculateWaveletFeatures(trajectory)
	features["wavelet_energy"] = wavelet["energy"]
	features["wavelet_variance"] = wavelet["variance"]

	speeds := sa.extractSpeeds(trajectory)
	if len(speeds) > 0 {
		features["speed_skewness"] = sa.calculateSkewness(speeds)
		features["speed_kurtosis"] = sa.calculateKurtosis(speeds)
		features["speed_range"] = sa.max(speeds) - sa.min(speeds)
	}

	features["start_delay_normalized"] = math.Min(float64(trajectory[0].Timestamp)/5000, 1.0)
	features["end_behavior"] = sa.analyzeEndBehavior(trajectory)

	return features
}

func (sa *SliderAnalyzer) extractAccelerations(trajectory []SliderPoint) []float64 {
	speeds := sa.extractSpeeds(trajectory)
	accelerations := make([]float64, 0)
	for i := 2; i < len(speeds); i++ {
		dt := float64(trajectory[i+1].Timestamp-trajectory[i-1].Timestamp) / 2
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}
	return accelerations
}

func (sa *SliderAnalyzer) extractCurvatures(trajectory []SliderPoint) []float64 {
	curvatures := make([]float64, 0)
	for i := 1; i < len(trajectory)-1; i++ {
		v1x := float64(trajectory[i].X - trajectory[i-1].X)
		v1y := float64(trajectory[i].Y - trajectory[i-1].Y)
		v2x := float64(trajectory[i+1].X - trajectory[i].X)
		v2y := float64(trajectory[i+1].Y - trajectory[i].Y)

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
			curvatures = append(curvatures, math.Abs(angle))
		}
	}
	return curvatures
}

func (sa *SliderAnalyzer) calculateJitterAdvanced(trajectory []SliderPoint) float64 {
	if len(trajectory) < 3 {
		return 0
	}

	smoothed := sa.smoothTrajectoryAdvanced(trajectory, 3)
	totalJitter := 0.0

	for i := 1; i < len(trajectory); i++ {
		dx1 := float64(trajectory[i].X - trajectory[i-1].X)
		dy1 := float64(trajectory[i].Y - trajectory[i-1].Y)
		dx2 := float64(smoothed[i].X - smoothed[i-1].X)
		dy2 := float64(smoothed[i].Y - smoothed[i-1].Y)

		distance1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		distance2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if distance1 > 0 {
			totalJitter += math.Abs(distance1-distance2) / distance1
		}
	}

	return totalJitter / float64(len(trajectory)-1)
}

func (sa *SliderAnalyzer) smoothTrajectoryAdvanced(trajectory []SliderPoint, windowSize int) []SliderPoint {
	if len(trajectory) < windowSize {
		return trajectory
	}

	if windowSize%2 == 0 {
		windowSize++
	}

	halfWindow := windowSize / 2
	smoothed := make([]SliderPoint, len(trajectory))

	for i := range trajectory {
		start := i - halfWindow
		end := i + halfWindow

		if start < 0 {
			start = 0
		}
		if end >= len(trajectory) {
			end = len(trajectory) - 1
		}

		sumX := 0
		sumY := 0
		count := 0

		for j := start; j <= end; j++ {
			sumX += trajectory[j].X
			sumY += trajectory[j].Y
			count++
		}

		smoothed[i] = trajectory[i]
		smoothed[i].X = sumX / count
		smoothed[i].Y = sumY / count
	}

	return smoothed
}

func (sa *SliderAnalyzer) calculateSmoothnessAdvanced(trajectory []SliderPoint) float64 {
	if len(trajectory) < 3 {
		return 0
	}

	totalAngleChange := 0.0
	count := 0

	for i := 1; i < len(trajectory)-1; i++ {
		v1x := float64(trajectory[i].X - trajectory[i-1].X)
		v1y := float64(trajectory[i].Y - trajectory[i-1].Y)
		v2x := float64(trajectory[i+1].X - trajectory[i].X)
		v2y := float64(trajectory[i+1].Y - trajectory[i].Y)

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

func (sa *SliderAnalyzer) calculateFourierFeatures(trajectory []SliderPoint) map[string]float64 {
	features := make(map[string]float64)

	if len(trajectory) < 8 {
		return features
	}

	n := len(trajectory)
	for n&(n-1) != 0 {
		n--
	}
	if n < 8 {
		return features
	}

	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = float64(trajectory[i].X)
	}

	fft := sa.fft(x)
	maxMag := 0.0
	dominantIdx := 0
	totalEnergy := 0.0

	for i := 1; i < n/2; i++ {
		mag := math.Sqrt(real(fft[i])*real(fft[i]) + imag(fft[i])*imag(fft[i]))
		totalEnergy += mag * mag
		if mag > maxMag {
			maxMag = mag
			dominantIdx = i
		}
	}

	totalTime := float64(trajectory[n-1].Timestamp - trajectory[0].Timestamp)
	if totalTime > 0 {
		features["dominant_freq"] = float64(dominantIdx) / totalTime * 1000
	}
	features["energy"] = totalEnergy

	entropy := 0.0
	for i := 1; i < n/2; i++ {
		mag := math.Sqrt(real(fft[i])*real(fft[i]) + imag(fft[i])*imag(fft[i]))
		if mag > 0 {
			p := (mag * mag) / totalEnergy
			entropy -= p * math.Log2(p)
		}
	}
	features["entropy"] = entropy

	return features
}

func (sa *SliderAnalyzer) fft(x []float64) []complex128 {
	n := len(x)
	if n <= 1 {
		result := make([]complex128, n)
		for i, val := range x {
			result[i] = complex(val, 0)
		}
		return result
	}

	even := make([]float64, n/2)
	odd := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	fftEven := sa.fft(even)
	fftOdd := sa.fft(odd)

	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		theta := -2 * math.Pi * float64(k) / float64(n)
		t := complex(math.Cos(theta), math.Sin(theta)) * fftOdd[k]
		result[k] = complex(real(fftEven[k])+real(t), imag(fftEven[k])+imag(t))
		result[k+n/2] = complex(real(fftEven[k])-real(t), imag(fftEven[k])-imag(t))
	}

	return result
}

func (sa *SliderAnalyzer) calculateWaveletFeatures(trajectory []SliderPoint) map[string]float64 {
	features := make(map[string]float64)

	if len(trajectory) < 4 {
		return features
	}

	levels := 3
	coefficients := make([][]float64, levels)

	for level := 0; level < levels && len(trajectory) > 1; level++ {
		coeffs := make([]float64, 0)
		for i := 0; i < len(trajectory)-1; i += 2 {
			avg := float64(trajectory[i].X+trajectory[i+1].X) / 2
			detail := float64(trajectory[i].X-trajectory[i+1].X) / 2
			coeffs = append(coeffs, detail)
			_ = avg
		}
		coefficients[level] = coeffs

		newTraj := make([]SliderPoint, len(coeffs))
		for i := 0; i < len(coeffs); i++ {
			newTraj[i] = SliderPoint{
				X:         int(float64(trajectory[i].X+trajectory[i+1].X) / 2),
				Y:         trajectory[i].Y,
				Timestamp: trajectory[i].Timestamp,
			}
		}
		trajectory = newTraj
	}

	totalEnergy := 0.0
	for _, level := range coefficients {
		for _, c := range level {
			totalEnergy += c * c
		}
	}
	features["energy"] = totalEnergy

	var variance float64
	for _, level := range coefficients {
		if len(level) > 1 {
			mean := 0.0
			for _, c := range level {
				mean += c
			}
			mean /= float64(len(level))
			for _, c := range level {
				variance += (c - mean) * (c - mean)
			}
			variance /= float64(len(level))
		}
	}
	features["variance"] = variance

	return features
}

func (sa *SliderAnalyzer) calculateSkewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	mean := sa.mean(values)
	stdDev := math.Sqrt(sa.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func (sa *SliderAnalyzer) calculateKurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}
	mean := sa.mean(values)
	stdDev := math.Sqrt(sa.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 4)
	}
	return (sum / float64(len(values))) - 3
}

func (sa *SliderAnalyzer) analyzeEndBehavior(trajectory []SliderPoint) float64 {
	if len(trajectory) < 5 {
		return 0.5
	}

	lastPoints := trajectory[len(trajectory)-5:]
	totalDist := 0.0
	netDist := 0.0

	for i := 1; i < len(lastPoints); i++ {
		dx := float64(lastPoints[i].X - lastPoints[i-1].X)
		dy := float64(lastPoints[i].Y - lastPoints[i-1].Y)
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}

	startX := float64(lastPoints[0].X)
	startY := float64(lastPoints[0].Y)
	endX := float64(lastPoints[len(lastPoints)-1].X)
	endY := float64(lastPoints[len(lastPoints)-1].Y)
	netDist = math.Sqrt((endX-startX)*(endX-startX) + (endY-startY)*(endY-startY))

	if totalDist == 0 {
		return 0.5
	}

	return netDist / totalDist
}

func (sa *SliderAnalyzer) calculateFractalDimensionSimple(trajectory []SliderPoint) float64 {
	if len(trajectory) < 10 {
		return 1.0
	}

	minX, maxX := trajectory[0].X, trajectory[0].X
	minY, maxY := trajectory[0].Y, trajectory[0].Y

	for _, p := range trajectory {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	width := maxX - minX
	height := maxY - minY

	if width == 0 && height == 0 {
		return 1.0
	}

	maxScale := 5
	logScales := make([]float64, maxScale)
	logCounts := make([]float64, maxScale)

	for scale := 0; scale < maxScale; scale++ {
		boxSize := int(math.Pow(2, float64(maxScale-scale)))
		grid := make(map[string]bool)

		for _, p := range trajectory {
			gx := (p.X - minX) / boxSize
			gy := (p.Y - minY) / boxSize
			key := fmt.Sprintf("%d,%d", gx, gy)
			grid[key] = true
		}

		logScales[scale] = math.Log(1.0 / float64(boxSize))
		logCounts[scale] = math.Log(float64(len(grid)))
	}

	return math.Max(1.0, math.Min(sa.linearRegressionSimple(logScales, logCounts), 2.0))
}

func (sa *SliderAnalyzer) linearRegressionSimple(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return 1.0
	}

	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denominator := float64(n)*sumX2 - sumX*sumX
	if denominator == 0 {
		return 1.0
	}

	return (float64(n)*sumXY - sumX*sumY) / denominator
}

func (sa *SliderAnalyzer) CalculateAdvancedBotScore(trajectory []SliderPoint, targetPosition int) (float64, []string) {
	if len(trajectory) < 3 {
		return 1.0, []string{"轨迹数据点不足"}
	}

	dtwAnalyzer := NewDTWAnalyzer()
	botPatternLibrary := NewBotPatternLibrary()
	_ = dtwAnalyzer

	botScore := 0.0
	indicators := make([]string, 0)

	advancedFeatures := sa.AnalyzeAdvancedFeatures(trajectory, targetPosition)

	if advancedFeatures["acceleration_std"] < 0.01 && advancedFeatures["acceleration_mean"] < 0.1 {
		botScore += 0.15
		indicators = append(indicators, "加速度变化异常平稳")
	}

	if advancedFeatures["curvature_mean"] < 0.01 && advancedFeatures["curvature_significant_ratio"] < 0.05 {
		botScore += 0.2
		indicators = append(indicators, "曲率异常低")
	}

	if advancedFeatures["jitter_normalized"] < 0.05 {
		botScore += 0.15
		indicators = append(indicators, "轨迹抖动异常低")
	}

	if advancedFeatures["fourier_entropy"] < 2.0 {
		botScore += 0.1
		indicators = append(indicators, "频谱熵异常低")
	}

	if advancedFeatures["fractal_dimension"] < 1.2 {
		botScore += 0.15
		indicators = append(indicators, "分形维数过低")
	}

	if advancedFeatures["speed_skewness"] < 0.1 && advancedFeatures["speed_kurtosis"] < 0.5 {
		botScore += 0.1
		indicators = append(indicators, "速度分布异常规则")
	}

	if advancedFeatures["end_behavior"] > 0.99 {
		botScore += 0.1
		indicators = append(indicators, "末端行为异常")
	}

	patternScore, patternIndicators := botPatternLibrary.DetectPatterns(trajectory)
	botScore += patternScore * 0.3
	indicators = append(indicators, patternIndicators...)

	return math.Min(botScore, 1.0), indicators
}
