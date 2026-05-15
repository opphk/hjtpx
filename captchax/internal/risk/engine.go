package risk

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"captchax/internal/config"
	"captchax/internal/risk/ai"
)

type MouseTrack struct {
	X           float64
	Y           float64
	Timestamp   int64
	Velocity    float64
	Acceleration float64
	Pressure    float64  // 新增：点击压力
	EventType   string   // 新增：事件类型 (move, press, release)
}

type ClickEvent struct {
	Timestamp   int64
	X           float64
	Y           float64
	Pressure    float64  // 点击压力
	Duration    int64    // 按住持续时间
}

type HesitationPoint struct {
	X           float64
	Y           float64
	Timestamp   int64
	Duration    int64    // 停留时间
}

type BehaviorData struct {
	UserID            string
	SessionID         string
	MouseTracks       []MouseTrack
	ClickEvents       []ClickEvent  // 新增：详细点击事件
	ClickTimes        []int64       // 保留兼容性
	SlideStart        int64
	SlideEnd          int64
	SlidePath         []Point
	Success           bool
	HesitationPoints  []HesitationPoint // 新增：犹豫点
	DeviceOrientation []Point      // 新增：设备方向变化（移动端）
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
	aiTrainer  *ai.Trainer
	aiEnabled  bool
	aiConfig   *config.AIConfig
	mu         sync.RWMutex
}

func NewRiskEngine(cfg *config.RiskConfig, ipLimit *IPLimit, whitelist *Whitelist) *RiskEngine {
	return &RiskEngine{
		config:    cfg,
		ipLimit:   ipLimit,
		whitelist: whitelist,
	}
}

func NewRiskEngineWithAI(cfg *config.RiskConfig, ipLimit *IPLimit, whitelist *Whitelist, aiCfg *config.AIConfig) (*RiskEngine, error) {
	engine := &RiskEngine{
		config:    cfg,
		ipLimit:   ipLimit,
		whitelist: whitelist,
		aiConfig:  aiCfg,
		aiEnabled: aiCfg != nil && aiCfg.Enable,
	}

	if engine.aiEnabled {
		modelConfig := &ai.ModelConfig{
			ModelType:       ai.ModelTypeMLP,
			InputDim:        ai.FeatureDimension,
			HiddenDims:      aiCfg.HiddenDims,
			OutputDim:       1,
			LearningRate:    aiCfg.LearningRate,
			WeightDecay:     aiCfg.WeightDecay,
			BatchSize:       aiCfg.BatchSize,
			Epochs:          aiCfg.TrainingEpochs,
			ValidationSplit: aiCfg.ValidationSplit,
		}

		if len(modelConfig.HiddenDims) == 0 {
			modelConfig.HiddenDims = []int{64, 32, 16}
		}

		engine.aiTrainer = ai.NewTrainer(modelConfig)

		if aiCfg.ModelPath != "" {
			if err := engine.aiTrainer.LoadModel(aiCfg.ModelPath); err != nil {
				return engine, nil
			}
		}
	}

	return engine, nil
}

func (e *RiskEngine) IsAIEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.aiEnabled && e.aiTrainer != nil && e.aiTrainer.IsReady()
}

func (e *RiskEngine) SetAIEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.aiEnabled = enabled
}

func (e *RiskEngine) GetAIConfidence() float64 {
	if !e.IsAIEnabled() {
		return 0.0
	}
	return 0.7
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

// ========================================
// 新增的增强分析算法
// ========================================

// AnalyzeMouseAcceleration 分析鼠标加速度
func (e *RiskEngine) AnalyzeMouseAcceleration(tracks []MouseTrack) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(tracks) < 3 {
		return score, factors
	}

	var accelerations []float64
	for i := 2; i < len(tracks); i++ {
		dt1 := float64(tracks[i-1].Timestamp - tracks[i-2].Timestamp)
		dt2 := float64(tracks[i].Timestamp - tracks[i-1].Timestamp)
		
		if dt1 == 0 || dt2 == 0 {
			continue
		}

		dx1 := tracks[i-1].X - tracks[i-2].X
		dy1 := tracks[i-1].Y - tracks[i-2].Y
		v1 := math.Sqrt(dx1*dx1+dy1*dy1) / dt1

		dx2 := tracks[i].X - tracks[i-1].X
		dy2 := tracks[i].Y - tracks[i-1].Y
		v2 := math.Sqrt(dx2*dx2+dy2*dy2) / dt2

		acceleration := math.Abs(v2 - v1) / ((dt1 + dt2) / 2)
		accelerations = append(accelerations, acceleration)
	}

	if len(accelerations) < 2 {
		return score, factors
	}

	meanAccel := 0.0
	for _, a := range accelerations {
		meanAccel += a
	}
	meanAccel /= float64(len(accelerations))

	// 检查加速度是否异常稳定或异常大
	var variance float64
	for _, a := range accelerations {
		diff := a - meanAccel
		variance += diff * diff
	}
	variance /= float64(len(accelerations))
	stdDev := math.Sqrt(variance)

	// 加速度方差过小 - 机器行为
	accelCV := stdDev / (meanAccel + 0.001)
	if accelCV < 0.1 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "abnormal_acceleration_stability",
			Weight: 15,
			Reason: "鼠标加速度过于稳定，疑似机器行为",
		})
	}

	// 检查加速度峰值
	maxAccel := 0.0
	for _, a := range accelerations {
		if a > maxAccel {
			maxAccel = a
		}
	}

	if maxAccel > e.config.AccelerationThreshold {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "extreme_acceleration",
			Weight: 10,
			Reason: "鼠标加速度超出正常范围",
		})
	}

	return score, factors
}

// AnalyzeEnhancedClickPattern 增强点击模式分析
func (e *RiskEngine) AnalyzeEnhancedClickPattern(clicks []ClickEvent) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(clicks) < 2 {
		return score, factors
	}

	// 分析点击压力变化
	var pressures []float64
	for _, click := range clicks {
		pressures = append(pressures, click.Pressure)
	}

	if len(pressures) >= 2 {
		meanPressure := 0.0
		for _, p := range pressures {
			meanPressure += p
		}
		meanPressure /= float64(len(pressures))

		var pressureVariance float64
		for _, p := range pressures {
			diff := p - meanPressure
			pressureVariance += diff * diff
		}
		pressureVariance /= float64(len(pressures))

		if pressureVariance < 0.01 && meanPressure > 0 {
			score += 12
			factors = append(factors, RiskFactor{
				Name:   "uniform_pressure",
				Weight: 12,
				Reason: "点击压力异常均匀，疑似机器行为",
			})
		}
	}

	// 分析双击模式
	doubleClickCount := 0
	for i := 1; i < len(clicks); i++ {
		interval := clicks[i].Timestamp - clicks[i-1].Timestamp
		if interval > 0 && interval < e.config.DoubleClickThreshold {
			doubleClickCount++
		}
	}

	if doubleClickCount == len(clicks)-1 && len(clicks) > 2 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "suspicious_double_click_pattern",
			Weight: 10,
			Reason: "所有点击都呈现双击模式，异常规律",
		})
	}

	// 分析点击持续时间
	totalDuration := int64(0)
	for _, click := range clicks {
		totalDuration += click.Duration
	}
	avgDuration := float64(totalDuration) / float64(len(clicks))

	if avgDuration > 0 {
		var durationVariance float64
		for _, click := range clicks {
			diff := float64(click.Duration) - avgDuration
			durationVariance += diff * diff
		}
		durationVariance /= float64(len(clicks))
		durationCV := math.Sqrt(durationVariance) / avgDuration

		if durationCV < 0.05 {
			score += 8
			factors = append(factors, RiskFactor{
				Name:   "uniform_click_duration",
				Weight: 8,
				Reason: "点击持续时间异常一致",
			})
		}
	}

	// 分析点击间隔
	var intervals []int64
	for i := 1; i < len(clicks); i++ {
		interval := clicks[i].Timestamp - clicks[i-1].Timestamp
		intervals = append(intervals, interval)
	}

	if len(intervals) >= 2 {
		meanInterval := float64(0)
		for _, interval := range intervals {
			meanInterval += float64(interval)
		}
		meanInterval /= float64(len(intervals))

		var intervalVariance float64
		for _, interval := range intervals {
			diff := float64(interval) - meanInterval
			intervalVariance += diff * diff
		}
		intervalVariance /= float64(len(intervals))
		intervalCV := math.Sqrt(intervalVariance) / (meanInterval + 1)

		if intervalCV < 0.05 {
			score += 15
			factors = append(factors, RiskFactor{
				Name:   "mechanical_click_interval",
				Weight: 15,
				Reason: "点击间隔异常规律，疑似机器行为",
			})
		}

		// 检查点击间隔是否过小或过大
		for _, interval := range intervals {
			if interval < e.config.ClickIntervalMin {
				score += 8
				factors = append(factors, RiskFactor{
					Name:   "unusually_short_click_interval",
					Weight: 8,
					Reason: "点击间隔过短，超出人类能力",
				})
				break
			}
			if interval > e.config.ClickIntervalMax {
				score += 5
				factors = append(factors, RiskFactor{
					Name:   "unusually_long_click_interval",
					Weight: 5,
					Reason: "点击间隔过长，行为异常",
				})
				break
			}
		}
	}

	return score, factors
}

// AnalyzePathComplexity 分析路径复杂度
func (e *RiskEngine) AnalyzePathComplexity(tracks []MouseTrack) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(tracks) < 5 {
		return score, factors
	}

	// 计算方向变化次数
	directionChanges := 0
	for i := 2; i < len(tracks); i++ {
		angle1 := math.Atan2(tracks[i-1].Y-tracks[i-2].Y, tracks[i-1].X-tracks[i-2].X)
		angle2 := math.Atan2(tracks[i].Y-tracks[i-1].Y, tracks[i].X-tracks[i-1].X)
		
		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}
		
		if angleDiff > 0.5 { // 超过一定角度变化
			directionChanges++
		}
	}

	// 方向变化过少 - 机器行为
	if directionChanges < e.config.DirectionChangeThreshold/2 && len(tracks) > 20 {
		score += 12
		factors = append(factors, RiskFactor{
			Name:   "insufficient_direction_changes",
			Weight: 12,
			Reason: "路径方向变化过少，过于线性",
		})
	}

	// 计算路径弯曲度
	curvatureSum := 0.0
	curvatureCount := 0
	for i := 2; i < len(tracks); i++ {
		// 使用三点计算曲率
		p1 := Point{tracks[i-2].X, tracks[i-2].Y}
		p2 := Point{tracks[i-1].X, tracks[i-1].Y}
		p3 := Point{tracks[i].X, tracks[i].Y}

		curvature := e.calculateThreePointCurvature(p1, p2, p3)
		curvatureSum += curvature
		curvatureCount++
	}

	if curvatureCount > 0 {
		avgCurvature := curvatureSum / float64(curvatureCount)
		if avgCurvature > e.config.PathCurvatureThreshold {
			score += 10
			factors = append(factors, RiskFactor{
				Name:   "abnormal_curvature",
				Weight: 10,
				Reason: "路径曲率异常，疑似完美曲线",
			})
		}
	}

	// 计算路径复杂度
	complexity := e.calculatePathComplexity(tracks)
	if complexity < e.config.PathComplexityThreshold {
		score += 8
		factors = append(factors, RiskFactor{
			Name:   "low_path_complexity",
			Weight: 8,
			Reason: "路径复杂度过低，过于简单规整",
		})
	}

	return score, factors
}

func (e *RiskEngine) calculateThreePointCurvature(p1, p2, p3 Point) float64 {
	// 计算三角形面积
	area := math.Abs((p2.X-p1.X)*(p3.Y-p1.Y) - (p2.Y-p1.Y)*(p3.X-p1.X)) / 2
	
	// 计算边长
	a := math.Sqrt((p2.X-p1.X)*(p2.X-p1.X) + (p2.Y-p1.Y)*(p2.Y-p1.Y))
	b := math.Sqrt((p3.X-p2.X)*(p3.X-p2.X) + (p3.Y-p2.Y)*(p3.Y-p2.Y))
	c := math.Sqrt((p3.X-p1.X)*(p3.X-p1.X) + (p3.Y-p1.Y)*(p3.Y-p1.Y))
	
	if a == 0 || b == 0 || c == 0 {
		return 0
	}
	
	// 计算曲率半径 R = (a*b*c)/(4*area)
	if area == 0 {
		return 1.0 // 完全直线
	}
	
	R := (a * b * c) / (4 * area)
	if R == 0 {
		return 1.0
	}
	
	// 曲率 = 1/R，归一化
	curvature := 1.0 / (1.0 + R/100.0)
	return curvature
}

func (e *RiskEngine) calculatePathComplexity(tracks []MouseTrack) float64 {
	if len(tracks) < 3 {
		return 0
	}

	totalLength := 0.0
	straightLineLength := math.Sqrt(
		math.Pow(tracks[len(tracks)-1].X-tracks[0].X, 2) +
		math.Pow(tracks[len(tracks)-1].Y-tracks[0].Y, 2),
	)

	for i := 1; i < len(tracks); i++ {
		dx := tracks[i].X - tracks[i-1].X
		dy := tracks[i].Y - tracks[i-1].Y
		totalLength += math.Sqrt(dx*dx + dy*dy)
	}

	if totalLength == 0 {
		return 0
	}

	// 复杂度 = 实际路径长度 / 直线距离
	complexity := totalLength / (straightLineLength + 1.0)
	// 归一化到 0-1
	normalizedComplexity := 1.0 - 1.0/(1.0+complexity)
	return normalizedComplexity
}

// AnalyzeHesitation 分析犹豫行为
func (e *RiskEngine) AnalyzeHesitation(hesitations []HesitationPoint) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(hesitations) == 0 {
		return score, factors
	}

	longHesitationCount := 0
	for _, h := range hesitations {
		if h.Duration > e.config.HesitationThreshold {
			longHesitationCount++
		}
	}

	// 犹豫过多或过少都可能异常
	if longHesitationCount == len(hesitations) && len(hesitations) > 2 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "excessive_hesitation",
			Weight: 10,
			Reason: "异常多的长时间犹豫点",
		})
	}

	return score, factors
}

// EnhancedCalculateRiskScore 增强版风险评分计算
func (e *RiskEngine) EnhancedCalculateRiskScore(ctx context.Context, behavior *BehaviorData, ip string, domain string) *RiskResult {
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

	// 原有的分析
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

	// 原有的鼠标轨迹分析
	trackScore, trackFactors := e.AnalyzeMouseTrack(behavior.MouseTracks)
	totalScore += trackScore
	result.Factors = append(result.Factors, trackFactors...)

	// 原有的点击节奏分析
	if len(behavior.ClickTimes) > 0 {
		clickScore, clickFactors := e.AnalyzeClickRhythm(behavior.ClickTimes)
		totalScore += clickScore
		result.Factors = append(result.Factors, clickFactors...)
	}

	// 新增：加速度分析
	accelScore, accelFactors := e.AnalyzeMouseAcceleration(behavior.MouseTracks)
	totalScore += accelScore
	result.Factors = append(result.Factors, accelFactors...)

	// 新增：路径复杂度分析
	pathScore, pathFactors := e.AnalyzePathComplexity(behavior.MouseTracks)
	totalScore += pathScore
	result.Factors = append(result.Factors, pathFactors...)

	// 新增：增强点击模式分析
	if len(behavior.ClickEvents) > 0 {
		clickPatternScore, clickPatternFactors := e.AnalyzeEnhancedClickPattern(behavior.ClickEvents)
		totalScore += clickPatternScore
		result.Factors = append(result.Factors, clickPatternFactors...)
	}

	// 新增：犹豫行为分析
	if len(behavior.HesitationPoints) > 0 {
		hesitationScore, hesitationFactors := e.AnalyzeHesitation(behavior.HesitationPoints)
		totalScore += hesitationScore
		result.Factors = append(result.Factors, hesitationFactors...)
	}

	// IP 限制检查
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

// ========================================
// 自适应权重风险评分系统
// ========================================

// RiskFactorWeight 存储风险因子的权重
type RiskFactorWeight struct {
	Name           string
	BaseWeight     float64
	CurrentWeight  float64
	AdjustmentRate float64
}

// AdaptiveRiskScorer 自适应风险评分器
type AdaptiveRiskScorer struct {
	factorWeights  map[string]*RiskFactorWeight
	historyScores  []int
	historyResults []bool // true表示正确识别，false表示误报
}

// NewAdaptiveRiskScorer 创建自适应评分器
func NewAdaptiveRiskScorer() *AdaptiveRiskScorer {
	scorer := &AdaptiveRiskScorer{
		factorWeights:  make(map[string]*RiskFactorWeight),
		historyScores:  make([]int, 0),
		historyResults: make([]bool, 0),
	}

	// 初始化默认权重
	defaultWeights := map[string]float64{
		"slide_too_fast":                  30,
		"slide_too_slow":                  20,
		"over_smooth_track":               20,
		"low_jitter":                      10,
		"abnormal_velocity":               15,
		"mechanical_rhythm":               15,
		"unusually_fast_clicks":           10,
		"abnormal_acceleration_stability": 15,
		"extreme_acceleration":            10,
		"uniform_pressure":                12,
		"suspicious_double_click_pattern": 10,
		"uniform_click_duration":          8,
		"mechanical_click_interval":       15,
		"unusually_short_click_interval":  8,
		"unusually_long_click_interval":   5,
		"insufficient_direction_changes":  12,
		"abnormal_curvature":              10,
		"low_path_complexity":             8,
		"excessive_hesitation":            10,
	}

	for name, weight := range defaultWeights {
		scorer.factorWeights[name] = &RiskFactorWeight{
			Name:           name,
			BaseWeight:     weight,
			CurrentWeight:  weight,
			AdjustmentRate: 0.05,
		}
	}

	return scorer
}

// CalculateWeightedScore 使用自适应权重计算分数
func (ars *AdaptiveRiskScorer) CalculateWeightedScore(factors []RiskFactor) int {
	totalScore := 0.0

	for _, factor := range factors {
		if weight, exists := ars.factorWeights[factor.Name]; exists {
			totalScore += float64(factor.Weight) * (weight.CurrentWeight / weight.BaseWeight)
		} else {
			totalScore += float64(factor.Weight)
		}
	}

	if totalScore > 100 {
		totalScore = 100
	}

	return int(totalScore)
}

// UpdateWeights 根据反馈更新权重
func (ars *AdaptiveRiskScorer) UpdateWeights(factors []RiskFactor, isCorrect bool) {
	// 记录历史
	if len(ars.historyScores) >= 1000 {
		ars.historyScores = ars.historyScores[1:]
		ars.historyResults = ars.historyResults[1:]
	}

	ars.historyResults = append(ars.historyResults, isCorrect)

	// 更新权重
	for _, factor := range factors {
		if weight, exists := ars.factorWeights[factor.Name]; exists {
			if isCorrect {
				// 正确识别，增加权重
				weight.CurrentWeight *= (1 + weight.AdjustmentRate)
			} else {
				// 误报，减少权重
				weight.CurrentWeight *= (1 - weight.AdjustmentRate)
			}

			// 限制权重范围
			minWeight := weight.BaseWeight * 0.5
			maxWeight := weight.BaseWeight * 2.0
			if weight.CurrentWeight < minWeight {
				weight.CurrentWeight = minWeight
			} else if weight.CurrentWeight > maxWeight {
				weight.CurrentWeight = maxWeight
			}
		}
	}
}

// CalculateRiskWithAdaptiveWeights 使用自适应权重计算风险
func (e *RiskEngine) CalculateRiskWithAdaptiveWeights(
	ctx context.Context,
	behavior *BehaviorData,
	ip string,
	domain string,
	scorer *AdaptiveRiskScorer,
) *RiskResult {
	// 首先获取所有风险因子
	tempResult := e.EnhancedCalculateRiskScore(ctx, behavior, ip, domain)

	// 使用自适应权重重新计算分数
	if scorer != nil {
		weightedScore := scorer.CalculateWeightedScore(tempResult.Factors)
		tempResult.Score = weightedScore
		tempResult.Level = e.GetRiskLevel(weightedScore)
		tempResult.Recommended = e.getRecommendedAction(tempResult.Level)
	}

	return tempResult
}

// ========================================
// AI 深度学习模型集成
// ========================================

func (e *RiskEngine) PredictWithAI(ctx context.Context, behavior *BehaviorData) (*ai.PredictionResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.aiTrainer == nil || !e.aiEnabled {
		return nil, fmt.Errorf("AI model is not enabled")
	}

	aiBehavior := convertToAIBehavior(behavior)
	return e.aiTrainer.Predict(ctx, aiBehavior)
}

func (e *RiskEngine) CalculateRiskScoreWithAI(ctx context.Context, behavior *BehaviorData, ip string, domain string) *RiskResult {
	result := &RiskResult{
		Factors:   make([]RiskFactor, 0),
		Timestamp: time.Now(),
	}

	e.mu.RLock()
	aiEnabled := e.aiEnabled && e.aiTrainer != nil && e.aiTrainer.IsReady()
	e.mu.RUnlock()

	ruleResult := e.EnhancedCalculateRiskScore(ctx, behavior, ip, domain)

	if aiEnabled {
		aiBehavior := convertToAIBehavior(behavior)
		aiResult, err := e.aiTrainer.Predict(ctx, aiBehavior)
		if err == nil && aiResult != nil {
			ruleScore := float64(ruleResult.Score)
			aiScore := aiResult.Score * 100.0

			aiWeight := 0.7
			ruleWeight := 0.3

			combinedScore := ruleScore*ruleWeight + aiScore*aiWeight

			combinedScoreInt := int(combinedScore)
			if combinedScoreInt > 100 {
				combinedScoreInt = 100
			}

			result.Score = combinedScoreInt
			result.Factors = append(result.Factors, RiskFactor{
				Name:   "ai_model_prediction",
				Weight: int(aiScore),
				Reason: fmt.Sprintf("AI模型预测: %.2f, 置信度: %.2f", aiScore, aiResult.Confidence),
			})

			result.Level = e.GetRiskLevel(combinedScoreInt)

			if aiResult.ShouldBlock() || ruleResult.Level == RiskLevelCritical {
				result.Recommended = ActionBlock
			} else if aiResult.IsHighRisk() || ruleResult.Level == RiskLevelHigh {
				result.Recommended = ActionVerify
			} else {
				result.Recommended = ActionAllow
			}

			return result
		}
	}

	return ruleResult
}

func (e *RiskEngine) TrainAIModel(ctx context.Context, behaviors []*BehaviorData, labels []float64) (*ai.TrainingResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.aiTrainer == nil {
		return nil, fmt.Errorf("AI trainer not initialized")
	}

	if !e.aiEnabled {
		return nil, fmt.Errorf("AI is not enabled in config")
	}

	return e.aiTrainer.Train(ctx, behaviors, labels)
}

func (e *RiskEngine) UpdateAIModelWithFeedback(ctx context.Context, behavior *BehaviorData, isBot bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.aiTrainer == nil || !e.aiEnabled {
		return fmt.Errorf("AI model is not enabled")
	}

	return e.aiTrainer.UpdateWithFeedback(ctx, behavior, isBot)
}

func (e *RiskEngine) SaveAIModel(path string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.aiTrainer == nil {
		return fmt.Errorf("AI trainer not initialized")
	}

	return e.aiTrainer.SaveModel(path)
}

func (e *RiskEngine) LoadAIModel(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.aiTrainer == nil {
		return fmt.Errorf("AI trainer not initialized")
	}

	return e.aiTrainer.LoadModel(path)
}

func (e *RiskEngine) GetAIStats() *AIStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &AIStats{
		Enabled: e.aiEnabled,
	}

	if e.aiTrainer != nil {
		stats.Ready = e.aiTrainer.IsReady()
		stats.Training = e.aiTrainer.IsTraining()
	}

	if e.aiConfig != nil {
		stats.ModelPath = e.aiConfig.ModelPath
		stats.BatchSize = e.aiConfig.BatchSize
		stats.LearningRate = e.aiConfig.LearningRate
	}

	return stats
}

type AIStats struct {
	Enabled     bool
	Ready       bool
	Training    bool
	ModelPath   string
	BatchSize   int
	LearningRate float64
}
