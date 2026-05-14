package risk

import (
	"context"
	"math"
	"time"

	"captchax/internal/config"
)

type MouseTrack struct {
	X           float64
	Y           float64
	Timestamp   int64
	Velocity    float64
	Acceleration float64
}

type BehaviorData struct {
	UserID       string
	SessionID    string
	MouseTracks  []MouseTrack
	ClickTimes   []int64
	SlideStart   int64
	SlideEnd     int64
	SlidePath    []Point
	Success      bool
}

type Point struct {
	X float64
	Y float64
}

type RiskResult struct {
	Score       int
	Level       RiskLevel
	Factors     []RiskFactor
	Recommended Action
	Timestamp   time.Time
}

type RiskFactor struct {
	Name   string
	Weight int
	Reason string
}

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type Action string

const (
	ActionAllow  Action = "allow"
	ActionVerify Action = "verify"
	ActionBlock  Action = "block"
)

type RiskEngine struct {
	config     *config.RiskConfig
	ipLimit    *IPLimit
	whitelist  *Whitelist
}

func NewRiskEngine(cfg *config.RiskConfig, ipLimit *IPLimit, whitelist *Whitelist) *RiskEngine {
	return &RiskEngine{
		config:    cfg,
		ipLimit:   ipLimit,
		whitelist: whitelist,
	}
}

func (e *RiskEngine) TrackBehavior(data *BehaviorData) error {
	return nil
}

func (e *RiskEngine) AnalyzeMouseTrack(tracks []MouseTrack) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(tracks) < 2 {
		factors = append(factors, RiskFactor{
			Name:   "insufficient_track_data",
			Weight: 0,
			Reason: "鼠标轨迹数据不足",
		})
		return score, factors
	}

	smoothness := e.calculateSmoothness(tracks)
	if smoothness > 0.95 {
		score += 20
		factors = append(factors, RiskFactor{
			Name:   "over_smooth_track",
			Weight: 20,
			Reason: "轨迹过于平滑，可能为机器行为",
		})
	}

	jitter := e.calculateJitter(tracks)
	if jitter < 0.1 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "low_jitter",
			Weight: 10,
			Reason: "轨迹抖动过低，缺乏人类特征",
		})
	}

	velocityConsistency := e.calculateVelocityConsistency(tracks)
	if velocityConsistency > 0.9 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "abnormal_velocity",
			Weight: 15,
			Reason: "速度过于均匀，异常模式",
		})
	}

	return score, factors
}

func (e *RiskEngine) AnalyzeClickRhythm(clicks []int64) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(clicks) < 2 {
		return score, factors
	}

	rhythmVariance := e.calculateRhythmVariance(clicks)
	if rhythmVariance < 0.05 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "mechanical_rhythm",
			Weight: 15,
			Reason: "点击节奏过于机械",
		})
	}

	if e.isClickTooFast(clicks) {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "unusually_fast_clicks",
			Weight: 10,
			Reason: "点击速度异常快",
		})
	}

	return score, factors
}

func (e *RiskEngine) CalculateRiskScore(ctx context.Context, behavior *BehaviorData, ip string, domain string) *RiskResult {
	result := &RiskResult{
		Factors:   make([]RiskFactor, 0),
		Timestamp: time.Now(),
	}

	totalScore := 0

	if e.whitelist != nil && e.whitelist.IsWhiteListed(ctx, ip, domain, behavior.UserID) {
		result.Score = 0
		result.Level = RiskLevelLow
		result.Recommended = ActionAllow
		return result
	}

	slideDuration := behavior.SlideEnd - behavior.SlideStart
	if slideDuration > 0 {
		slideSeconds := float64(slideDuration) / 1000.0

		if slideSeconds < 1.0 {
			totalScore += 30
			result.Factors = append(result.Factors, RiskFactor{
				Name:   "slide_too_fast",
				Weight: 30,
				Reason: "滑动完成时间过短(<1秒)，疑似机器行为",
			})
		} else if slideSeconds > 30.0 {
			totalScore += 20
			result.Factors = append(result.Factors, RiskFactor{
				Name:   "slide_too_slow",
				Weight: 20,
				Reason: "滑动完成时间过长(>30秒)，异常行为",
			})
		}
	}

	trackScore, trackFactors := e.AnalyzeMouseTrack(behavior.MouseTracks)
	totalScore += trackScore
	result.Factors = append(result.Factors, trackFactors...)

	clickScore, clickFactors := e.AnalyzeClickRhythm(behavior.ClickTimes)
	totalScore += clickScore
	result.Factors = append(result.Factors, clickFactors...)

	if e.ipLimit != nil {
		ipScore, ipFactors := e.ipLimit.CheckIPRisk(ctx, ip)
		totalScore += ipScore
		result.Factors = append(result.Factors, ipFactors...)
	}

	if totalScore > 100 {
		totalScore = 100
	}

	result.Score = totalScore
	result.Level = e.GetRiskLevel(totalScore)
	result.Recommended = e.getRecommendedAction(result.Level)

	return result
}

func (e *RiskEngine) GetRiskLevel(score int) RiskLevel {
	switch {
	case score >= 80:
		return RiskLevelCritical
	case score >= 50:
		return RiskLevelHigh
	case score >= 25:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

func (e *RiskEngine) getRecommendedAction(level RiskLevel) Action {
	switch level {
	case RiskLevelLow:
		return ActionAllow
	case RiskLevelMedium:
		return ActionVerify
	case RiskLevelHigh:
		return ActionVerify
	case RiskLevelCritical:
		return ActionBlock
	default:
		return ActionVerify
	}
}

func (e *RiskEngine) calculateSmoothness(tracks []MouseTrack) float64 {
	if len(tracks) < 3 {
		return 1.0
	}

	var totalAngleChange float64
	angles := e.calculateAngles(tracks)

	for i := 1; i < len(angles); i++ {
		angleDiff := math.Abs(angles[i] - angles[i-1])
		totalAngleChange += angleDiff
	}

	maxPossibleChange := float64(len(tracks)-1) * math.Pi
	if maxPossibleChange == 0 {
		return 1.0
	}

	smoothness := 1.0 - (totalAngleChange / maxPossibleChange)
	return smoothness
}

func (e *RiskEngine) calculateAngles(tracks []MouseTrack) []float64 {
	angles := make([]float64, 0, len(tracks)-2)

	for i := 1; i < len(tracks)-1; i++ {
		v1x := tracks[i].X - tracks[i-1].X
		v1y := tracks[i].Y - tracks[i-1].Y
		v2x := tracks[i+1].X - tracks[i].X
		v2y := tracks[i+1].Y - tracks[i].Y

		dot := v1x*v2x + v1y*v2y
		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 == 0 || mag2 == 0 {
			angles = append(angles, 0)
			continue
		}

		cosAngle := dot / (mag1 * mag2)
		if cosAngle > 1 {
			cosAngle = 1
		}
		if cosAngle < -1 {
			cosAngle = -1
		}

		angle := math.Acos(cosAngle)
		angles = append(angles, angle)
	}

	return angles
}

func (e *RiskEngine) calculateJitter(tracks []MouseTrack) float64 {
	if len(tracks) < 3 {
		return 0.0
	}

	var jitterSum float64
	count := 0

	for i := 2; i < len(tracks); i++ {
		v1x := tracks[i-1].X - tracks[i-2].X
		v1y := tracks[i-1].Y - tracks[i-2].Y
		v2x := tracks[i].X - tracks[i-1].X
		v2y := tracks[i].Y - tracks[i-1].Y

		dx := v2x - v1x
		dy := v2y - v1y
		jitter := math.Sqrt(dx*dx + dy*dy)

		jitterSum += jitter
		count++
	}

	if count == 0 {
		return 0.0
	}

	return jitterSum / float64(count)
}

func (e *RiskEngine) calculateVelocityConsistency(tracks []MouseTrack) float64 {
	if len(tracks) < 2 {
		return 1.0
	}

	var velocities []float64
	for i := 1; i < len(tracks); i++ {
		dx := tracks[i].X - tracks[i-1].X
		dy := tracks[i].Y - tracks[i-1].Y
		dt := float64(tracks[i].Timestamp - tracks[i-1].Timestamp)
		if dt == 0 {
			continue
		}
		distance := math.Sqrt(dx*dx + dy*dy)
		velocity := distance / dt
		velocities = append(velocities, velocity)
	}

	if len(velocities) < 2 {
		return 1.0
	}

	mean := 0.0
	for _, v := range velocities {
		mean += v
	}
	mean /= float64(len(velocities))

	if mean == 0 {
		return 1.0
	}

	var variance float64
	for _, v := range velocities {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(velocities))

	coefficientOfVariation := math.Sqrt(variance) / mean

	return coefficientOfVariation
}

func (e *RiskEngine) calculateRhythmVariance(clicks []int64) float64 {
	if len(clicks) < 3 {
		return 1.0
	}

	var intervals []float64
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i] - clicks[i-1])
		intervals = append(intervals, interval)
	}

	mean := 0.0
	for _, interval := range intervals {
		mean += interval
	}
	mean /= float64(len(intervals))

	if mean == 0 {
		return 0.0
	}

	var variance float64
	for _, interval := range intervals {
		diff := interval - mean
		variance += diff * diff
	}
	variance /= float64(len(intervals))

	return variance / (mean * mean)
}

func (e *RiskEngine) isClickTooFast(clicks []int64) bool {
	if len(clicks) < 2 {
		return false
	}

	for i := 1; i < len(clicks); i++ {
		interval := clicks[i] - clicks[i-1]
		if interval < 50 {
			return true
		}
	}
	return false
}
