package behavior

import (
	"math"
	"time"
)

type Analyzer struct {
	humanThreshold      float64
	suspiciousThreshold float64
}

type TrajectoryPoint struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	T int64 `json:"t"`
}

type BehaviorData struct {
	Trajectory     []TrajectoryPoint `json:"trajectory"`
	TotalTime      int64             `json:"total_time_ms"`
	ClickCount     int               `json:"click_count"`
	ScrollCount    int               `json:"scroll_count"`
	KeyPressCount  int               `json:"key_press_count"`
	MouseMoveCount int               `json:"mouse_move_count"`
}

type AnalysisResult struct {
	RiskScore       float64     `json:"risk_score"`
	IsHuman         bool        `json:"is_human"`
	Confidence      float64     `json:"confidence"`
	Factors         []Factor   `json:"factors"`
	Recommendations []string    `json:"recommendations"`
}

type Factor struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Reason string  `json:"reason"`
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		humanThreshold:      0.6,
		suspiciousThreshold: 0.7,
	}
}

func (a *Analyzer) Analyze(data *BehaviorData) *AnalysisResult {
	result := &AnalysisResult{
		Factors:         make([]Factor, 0),
		Recommendations: make([]string, 0),
	}

	if len(data.Trajectory) > 1 {
		speedScore := a.analyzeSpeed(data.Trajectory)
		result.Factors = append(result.Factors, Factor{
			Name:   "speed",
			Score:  speedScore,
			Weight: 0.25,
			Reason: a.getSpeedReason(speedScore),
		})

		accelerationScore := a.analyzeAcceleration(data.Trajectory)
		result.Factors = append(result.Factors, Factor{
			Name:   "acceleration",
			Score:  accelerationScore,
			Weight: 0.2,
			Reason: a.getAccelerationReason(accelerationScore),
		})

		directionScore := a.analyzeDirection(data.Trajectory)
		result.Factors = append(result.Factors, Factor{
			Name:   "direction",
			Score:  directionScore,
			Weight: 0.15,
			Reason: a.getDirectionReason(directionScore),
		})

		pauseScore := a.analyzePauses(data.Trajectory)
		result.Factors = append(result.Factors, Factor{
			Name:   "pause",
			Score:  pauseScore,
			Weight: 0.2,
			Reason: a.getPauseReason(pauseScore),
		})
	}

	timeScore := a.analyzeTime(data.TotalTime)
	result.Factors = append(result.Factors, Factor{
		Name:   "time",
		Score:  timeScore,
		Weight: 0.1,
		Reason: a.getTimeReason(timeScore),
	})

	clickScore := a.analyzeClicks(data.ClickCount)
	result.Factors = append(result.Factors, Factor{
		Name:   "click_pattern",
		Score:  clickScore,
		Weight: 0.1,
		Reason: a.getClickReason(clickScore),
	})

	result.RiskScore = a.calculateWeightedScore(result.Factors)
	result.IsHuman = result.RiskScore >= a.humanThreshold
	result.Confidence = a.calculateConfidence(result.Factors)

	if result.RiskScore < a.humanThreshold {
		result.Recommendations = append(result.Recommendations, "Require additional verification")
	}
	if result.RiskScore > a.suspiciousThreshold {
		result.Recommendations = append(result.Recommendations, "Block or require extra captcha")
	}

	return result
}

func (a *Analyzer) analyzeSpeed(trajectory []TrajectoryPoint) float64 {
	if len(trajectory) < 2 {
		return 0.5
	}

	speeds := make([]float64, 0)
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		dt := float64(trajectory[i].T - trajectory[i-1].T)
		if dt > 0 {
			distance := math.Sqrt(dx*dx + dy*dy)
			speed := distance / dt
			speeds = append(speeds, speed)
		}
	}

	if len(speeds) == 0 {
		return 0.5
	}

	avgSpeed := 0.0
	for _, s := range speeds {
		avgSpeed += s
	}
	avgSpeed /= float64(len(speeds))

	if avgSpeed < 0.1 || avgSpeed > 5.0 {
		return 0.3
	}
	if avgSpeed >= 0.3 && avgSpeed <= 2.0 {
		return 1.0
	}
	return 0.7
}

func (a *Analyzer) analyzeAcceleration(trajectory []TrajectoryPoint) float64 {
	if len(trajectory) < 3 {
		return 0.5
	}

	accelerations := make([]float64, 0)
	var prevSpeed float64 = 0

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		dt := float64(trajectory[i].T - trajectory[i-1].T)
		if dt > 0 {
			distance := math.Sqrt(dx*dx + dy*dy)
			speed := distance / dt
			if prevSpeed > 0 {
				accel := math.Abs(speed - prevSpeed)
				accelerations = append(accelerations, accel)
			}
			prevSpeed = speed
		}
	}

	if len(accelerations) == 0 {
		return 0.5
	}

	variance := 0.0
	avgAccel := 0.0
	for _, acc := range accelerations {
		avgAccel += acc
	}
	avgAccel /= float64(len(accelerations))

	for _, acc := range accelerations {
		diff := acc - avgAccel
		variance += diff * diff
	}
	variance /= float64(len(accelerations))

	humanVariance := variance < 10.0
	if humanVariance {
		return 1.0
	}
	return 0.5
}

func (a *Analyzer) analyzeDirection(trajectory []TrajectoryPoint) float64 {
	if len(trajectory) < 3 {
		return 0.5
	}

	directionChanges := 0
	for i := 2; i < len(trajectory); i++ {
		dx1 := trajectory[i-1].X - trajectory[i-2].X
		dy1 := trajectory[i-1].Y - trajectory[i-2].Y
		dx2 := trajectory[i].X - trajectory[i-1].X
		dy2 := trajectory[i].Y - trajectory[i-1].Y

		dot := float64(dx1*dx2 + dy1*dy2)
		if dot < 0 {
			directionChanges++
		}
	}

	changeRatio := float64(directionChanges) / float64(len(trajectory)-2)

	if changeRatio > 0.5 {
		return 1.0
	}
	if changeRatio > 0.2 {
		return 0.8
	}
	return 0.6
}

func (a *Analyzer) analyzePauses(trajectory []TrajectoryPoint) float64 {
	if len(trajectory) < 2 {
		return 0.5
	}

	longPauses := 0
	for i := 1; i < len(trajectory); i++ {
		dt := trajectory[i].T - trajectory[i-1].T
		if dt > 100 {
			longPauses++
		}
	}

	pauseRatio := float64(longPauses) / float64(len(trajectory)-1)

	if pauseRatio > 0.1 {
		return 1.0
	}
	if pauseRatio > 0.01 {
		return 0.7
	}
	return 0.4
}

func (a *Analyzer) analyzeTime(totalTimeMs int64) float64 {
	if totalTimeMs < 300 {
		return 0.1
	}
	if totalTimeMs < 1000 {
		return 0.3
	}
	if totalTimeMs >= 3000 && totalTimeMs <= 15000 {
		return 1.0
	}
	if totalTimeMs <= 30000 {
		return 0.8
	}
	return 0.6
}

func (a *Analyzer) analyzeClicks(clickCount int) float64 {
	if clickCount == 0 {
		return 0.3
	}
	if clickCount >= 1 && clickCount <= 5 {
		return 1.0
	}
	if clickCount <= 10 {
		return 0.8
	}
	return 0.6
}

func (a *Analyzer) calculateWeightedScore(factors []Factor) float64 {
	weightedSum := 0.0
	totalWeight := 0.0

	for _, f := range factors {
		weightedSum += f.Score * f.Weight
		totalWeight += f.Weight
	}

	if totalWeight > 0 {
		return weightedSum / totalWeight
	}
	return 0.5
}

func (a *Analyzer) calculateConfidence(factors []Factor) float64 {
	if len(factors) == 0 {
		return 0.5
	}

	variance := 0.0
	avgScore := 0.0
	for _, f := range factors {
		avgScore += f.Score
	}
	avgScore /= float64(len(factors))

	for _, f := range factors {
		diff := f.Score - avgScore
		variance += diff * diff
	}
	variance /= float64(len(factors))

	return 1.0 - math.Min(variance, 1.0)
}

func (a *Analyzer) getSpeedReason(score float64) string {
	if score >= 0.8 {
		return "Speed pattern is human-like"
	}
	return "Speed pattern is suspicious"
}

func (a *Analyzer) getAccelerationReason(score float64) string {
	if score >= 0.8 {
		return "Acceleration pattern is natural"
	}
	return "Acceleration pattern is unusual"
}

func (a *Analyzer) getDirectionReason(score float64) string {
	if score >= 0.8 {
		return "Direction changes are natural"
	}
	return "Too few or too many direction changes"
}

func (a *Analyzer) getPauseReason(score float64) string {
	if score >= 0.8 {
		return "Human-like pauses detected"
	}
	return "Lack of natural pauses"
}

func (a *Analyzer) getTimeReason(score float64) string {
	if score >= 0.8 {
		return "Response time is appropriate"
	}
	return "Response time is too fast or too slow"
}

func (a *Analyzer) getClickReason(score float64) string {
	if score >= 0.8 {
		return "Click pattern is normal"
	}
	return "Click pattern is unusual"
}

type BehaviorCollector struct {
	StartTime time.Time
	Points    []TrajectoryPoint
}

func NewBehaviorCollector() *BehaviorCollector {
	return &BehaviorCollector{
		StartTime: time.Now(),
		Points:    make([]TrajectoryPoint, 0),
	}
}

func (bc *BehaviorCollector) AddPoint(x, y int) {
	bc.Points = append(bc.Points, TrajectoryPoint{
		X: x,
		Y: y,
		T: time.Since(bc.StartTime).Milliseconds(),
	})
}

func (bc *BehaviorCollector) GetData(clickCount, scrollCount, keyPressCount, mouseMoveCount int) *BehaviorData {
	return &BehaviorData{
		Trajectory:     bc.Points,
		TotalTime:      time.Since(bc.StartTime).Milliseconds(),
		ClickCount:     clickCount,
		ScrollCount:    scrollCount,
		KeyPressCount:  keyPressCount,
		MouseMoveCount: mouseMoveCount,
	}
}
