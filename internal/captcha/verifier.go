package captcha

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type TrackPoint struct {
	X      float64     `json:"x"`
	Y      float64     `json:"y"`
	T      int64       `json:"t"`
	Action TrackAction `json:"action"`
}

type TrackAction string

const (
	ActionMove    TrackAction = "move"
	ActionDown    TrackAction = "down"
	ActionUp      TrackAction = "up"
	ActionClick   TrackAction = "click"
)

type TrackData struct {
	Token    string       `json:"token"`
	Points   []TrackPoint `json:"points"`
	Duration int64        `json:"duration"`
}

type VerificationResult struct {
	Valid          bool    `json:"valid"`
	Score          float64 `json:"score"`
	Confidence     float64 `json:"confidence"`
	TargetX        int     `json:"target_x"`
	ActualX        float64 `json:"actual_x"`
	Offset         float64 `json:"offset"`
	Analysis       *TrackAnalysis `json:"analysis"`
	RiskLevel      RiskLevel `json:"risk_level"`
	Recommendations []string `json:"recommendations,omitempty"`
}

type TrackAnalysis struct {
	TotalPoints     int           `json:"total_points"`
	TotalDistance   float64       `json:"total_distance"`
	AvgSpeed        float64       `json:"avg_speed"`
	MaxSpeed        float64       `json:"max_speed"`
	MinSpeed        float64       `json:"min_speed"`
	SpeedVariance   float64       `json:"speed_variance"`
	AvgAcceleration float64      `json:"avg_acceleration"`
	MaxAcceleration float64       `json:"max_acceleration"`
	DirectionChanges int         `json:"direction_changes"`
	HasHumanPattern bool         `json:"has_human_pattern"`
	SuspiciousMarks []Suspicion  `json:"suspicious_marks,omitempty"`
}

type Suspicion struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    float64 `json:"severity"`
}

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type TrajectoryVerifier struct {
	config *VerifierConfig
}

type VerifierConfig struct {
	MaxOffsetTolerance float64
	MinHumanTime       int64
	MaxSpeedThreshold  float64
	MinTrackPoints     int
	Sensitivity        float64
}

var DefaultVerifierConfig = &VerifierConfig{
	MaxOffsetTolerance: 10.0,
	MinHumanTime:       500,
	MaxSpeedThreshold:  2000.0,
	MinTrackPoints:     10,
	Sensitivity:        0.7,
}

func NewTrajectoryVerifier(config *VerifierConfig) *TrajectoryVerifier {
	if config == nil {
		config = DefaultVerifierConfig
	}
	return &TrajectoryVerifier{config: config}
}

func (v *TrajectoryVerifier) ParseTrackData(data string) (*TrackData, error) {
	var track TrackData
	if err := json.Unmarshal([]byte(data), &track); err != nil {
		return nil, fmt.Errorf("failed to parse track data: %w", err)
	}
	return &track, nil
}

func (v *TrajectoryVerifier) Verify(trackData *TrackData, targetX int) *VerificationResult {
	result := &VerificationResult{
		Valid:      false,
		TargetX:   targetX,
		RiskLevel: RiskLow,
	}

	if len(trackData.Points) < v.config.MinTrackPoints {
		result.Analysis = &TrackAnalysis{
			TotalPoints: len(trackData.Points),
		}
		result.Confidence = 0.0
		result.Analysis.SuspiciousMarks = append(result.Analysis.SuspiciousMarks,
			Suspicion{
				Type:        "insufficient_data",
				Description: "Not enough track points for analysis",
				Severity:    0.8,
			})
		result.RiskLevel = RiskHigh
		return result
	}

	analysis := v.analyzeTrajectory(trackData)
	result.Analysis = analysis

	offset := math.Abs(float64(targetX) - analysis.TotalDistance)
	result.ActualX = analysis.TotalDistance
	result.Offset = offset

	positionValid := offset <= v.config.MaxOffsetTolerance
	timeValid := trackData.Duration >= v.config.MinHumanTime
	patternValid := analysis.HasHumanPattern

	result.Score = v.calculateScore(analysis, positionValid, timeValid, patternValid)
	result.Confidence = result.Score

	result.Valid = positionValid && timeValid && result.Score >= v.config.Sensitivity

	result.RiskLevel = v.determineRiskLevel(analysis)

	if result.RiskLevel == RiskHigh {
		result.Valid = false
	}

	result.Recommendations = v.generateRecommendations(result)

	return result
}

func (v *TrajectoryVerifier) analyzeTrajectory(track *TrackData) *TrackAnalysis {
	analysis := &TrackAnalysis{
		TotalPoints: len(track.Points),
	}

	if len(track.Points) < 2 {
		return analysis
	}

	var totalDistance float64
	var speeds []float64
	var accelerations []float64
	var prevSpeed float64 = 0
	directionChanges := 0
	var prevAngle float64 = 0

	for i := 1; i < len(track.Points); i++ {
		prev := track.Points[i-1]
		curr := track.Points[i]

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		dt := float64(curr.T - prev.T)

		if dt <= 0 {
			dt = 1
		}

		segmentDist := math.Sqrt(dx*dx + dy*dy)
		totalDistance += segmentDist

		speed := segmentDist / dt * 1000
		speeds = append(speeds, speed)

		if i > 1 && prevSpeed > 0 {
			accel := (speed - prevSpeed) / dt * 1000
			accelerations = append(accelerations, math.Abs(accel))
		}
		prevSpeed = speed

		angle := math.Atan2(dy, dx) * 180 / math.Pi
		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > 30 {
				directionChanges++
			}
		}
		prevAngle = angle
	}

	analysis.TotalDistance = totalDistance

	if len(speeds) > 0 {
		var sumSpeed float64
		var maxSpeed, minSpeed float64 = speeds[0], speeds[0]
		for _, s := range speeds {
			sumSpeed += s
			if s > maxSpeed {
				maxSpeed = s
			}
			if s < minSpeed {
				minSpeed = s
			}
		}
		analysis.AvgSpeed = sumSpeed / float64(len(speeds))
		analysis.MaxSpeed = maxSpeed
		analysis.MinSpeed = minSpeed

		if len(speeds) > 1 {
			var sumSqDiff float64
			for _, s := range speeds {
				diff := s - analysis.AvgSpeed
				sumSqDiff += diff * diff
			}
			analysis.SpeedVariance = sumSqDiff / float64(len(speeds))
		}
	}

	if len(accelerations) > 0 {
		var sumAccel float64
		var maxAccel float64 = accelerations[0]
		for _, a := range accelerations {
			sumAccel += a
			if a > maxAccel {
				maxAccel = a
			}
		}
		analysis.AvgAcceleration = sumAccel / float64(len(accelerations))
		analysis.MaxAcceleration = maxAccel
	}

	analysis.DirectionChanges = directionChanges
	analysis.HasHumanPattern = v.detectHumanPattern(analysis, track)

	if !analysis.HasHumanPattern {
		analysis.SuspiciousMarks = append(analysis.SuspiciousMarks,
			Suspicion{
				Type:        "mechanical_pattern",
				Description: "Track pattern appears mechanical, not human-like",
				Severity:    0.7,
			})
	}

	if analysis.MaxSpeed > v.config.MaxSpeedThreshold {
		analysis.SuspiciousMarks = append(analysis.SuspiciousMarks,
			Suspicion{
				Type:        "unrealistic_speed",
				Description: fmt.Sprintf("Max speed %.2f exceeds human capability", analysis.MaxSpeed),
				Severity:    0.6,
			})
	}

	if float64(track.Duration) < float64(v.config.MinHumanTime)*0.5 {
		analysis.SuspiciousMarks = append(analysis.SuspiciousMarks,
			Suspicion{
				Type:        "too_fast",
				Description: "Completion time is suspiciously fast",
				Severity:    0.8,
			})
	}

	if analysis.SpeedVariance < 10 {
		analysis.SuspiciousMarks = append(analysis.SuspiciousMarks,
			Suspicion{
				Type:        "uniform_speed",
				Description: "Speed is too uniform, likely automated",
				Severity:    0.6,
			})
	}

	return analysis
}

func (v *TrajectoryVerifier) detectHumanPattern(analysis *TrackAnalysis, track *TrackData) bool {
	if analysis.DirectionChanges < 2 && analysis.TotalDistance > 100 {
		return false
	}

	if analysis.SpeedVariance < 5 && analysis.AvgSpeed > 50 {
		return false
	}

	if analysis.AvgAcceleration < 50 && analysis.TotalDistance > 150 {
		return false
	}

	if analysis.MaxSpeed > v.config.MaxSpeedThreshold*0.8 && analysis.TotalDistance > 200 {
		return false
	}

	humanScore := 0.0

	if analysis.DirectionChanges >= 2 {
		humanScore += 0.2
	}
	if analysis.DirectionChanges >= 5 {
		humanScore += 0.1
	}

	if analysis.SpeedVariance >= 20 {
		humanScore += 0.2
	} else if analysis.SpeedVariance >= 10 {
		humanScore += 0.1
	}

	if analysis.AvgAcceleration >= 100 {
		humanScore += 0.2
	} else if analysis.AvgAcceleration >= 50 {
		humanScore += 0.1
	}

	if float64(track.Duration) >= float64(v.config.MinHumanTime) {
		humanScore += 0.2
	}

	if analysis.TotalPoints >= 30 {
		humanScore += 0.1
	}

	return humanScore >= 0.6
}

func (v *TrajectoryVerifier) calculateScore(analysis *TrackAnalysis, positionValid, timeValid, patternValid bool) float64 {
	score := 0.0

	if positionValid {
		score += 0.4
		offsetFactor := 1.0 - (analysis.TotalDistance / 500.0)
		if offsetFactor < 0 {
			offsetFactor = 0
		}
		score += offsetFactor * 0.2
	}

	if timeValid {
		score += 0.2
	}

	if patternValid {
		score += 0.2
	}

	if analysis.SpeedVariance > 30 {
		score += 0.1
	}

	if analysis.DirectionChanges >= 3 {
		score += 0.1
	}

	suspiciousPenalty := 0.0
	for _, s := range analysis.SuspiciousMarks {
		suspiciousPenalty += s.Severity * 0.15
	}
	score -= suspiciousPenalty

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

func (v *TrajectoryVerifier) determineRiskLevel(analysis *TrackAnalysis) RiskLevel {
	if len(analysis.SuspiciousMarks) == 0 {
		return RiskLow
	}

	highSeverityCount := 0
	mediumSeverityCount := 0

	for _, s := range analysis.SuspiciousMarks {
		if s.Severity >= 0.7 {
			highSeverityCount++
		} else if s.Severity >= 0.5 {
			mediumSeverityCount++
		}
	}

	if highSeverityCount >= 2 || (highSeverityCount >= 1 && mediumSeverityCount >= 2) {
		return RiskHigh
	}

	if highSeverityCount >= 1 || mediumSeverityCount >= 2 {
		return RiskMedium
	}

	return RiskLow
}

func (v *TrajectoryVerifier) generateRecommendations(result *VerificationResult) []string {
	var recs []string

	if result.RiskLevel == RiskHigh {
		recs = append(recs, "Request additional verification")
		recs = append(recs, "Consider blocking the user temporarily")
	}

	if result.Analysis != nil {
		if !result.Analysis.HasHumanPattern {
			recs = append(recs, "Track pattern does not appear human-like")
		}

		if result.Analysis.MaxSpeed > v.config.MaxSpeedThreshold {
			recs = append(recs, "Speed exceeds human capability thresholds")
		}

		if result.Analysis.SpeedVariance < 10 {
			recs = append(recs, "Movement pattern is too uniform")
		}
	}

	if result.Valid {
		recs = append(recs, "Verification passed")
	} else {
		recs = append(recs, fmt.Sprintf("Offset %.2f exceeds tolerance %.2f", result.Offset, v.config.MaxOffsetTolerance))
	}

	return recs
}

func (v *TrajectoryVerifier) ValidateTrackFormat(track *TrackData) error {
	if track == nil {
		return fmt.Errorf("track data is nil")
	}

	if len(track.Points) == 0 {
		return fmt.Errorf("no track points provided")
	}

	for i, point := range track.Points {
		if point.X < 0 || point.Y < 0 {
			return fmt.Errorf("invalid coordinates at point %d", i)
		}
		if point.T < 0 {
			return fmt.Errorf("invalid timestamp at point %d", i)
		}
	}

	if track.Duration < 0 {
		return fmt.Errorf("invalid duration")
	}

	return nil
}

func ParseTimestamp(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
