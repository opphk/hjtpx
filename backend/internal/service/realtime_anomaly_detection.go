package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type AnomalyType string

const (
	AnomalyTypeRapidClicks      AnomalyType = "rapid_clicks"
	AnomalyTypePatternRepetition AnomalyType = "pattern_repetition"
	AnomalyTypeUnusualTiming    AnomalyType = "unusual_timing"
	AnomalyTypeStraightLine     AnomalyType = "straight_line"
	AnomalyTypeLowJitter        AnomalyType = "low_jitter"
	AnomalyTypeNoPause         AnomalyType = "no_pause"
	AnomalyTypeHighSpeed       AnomalyType = "high_speed"
	AnomalyTypeConstantSpeed   AnomalyType = "constant_speed"
	AnomalyTypeMechanicalPattern AnomalyType = "mechanical_pattern"
	AnomalyTypeBotTyping       AnomalyType = "bot_typing"
	AnomalyTypeBehavioralDrift AnomalyType = "behavioral_drift"
	AnomalyTypeSessionAnomaly  AnomalyType = "session_anomaly"
)

type AnomalyScore struct {
	OverallScore   float64
	SpeedScore     float64
	TimingScore    float64
	PatternScore   float64
	DirectionScore float64
	PressureScore  float64
	SessionScore   float64
}

type AnomalyDetectionResult struct {
	IsAnomaly        bool
	AnomalyType      []AnomalyType
	AnomalyScore     *AnomalyScore
	Confidence       float64
	RiskLevel        string
	Severity         float64
	Threshold        float64
	ContributingFactors []string
	Recommendations  []string
	DetectedAt       time.Time
}

type RealtimeAdaptiveThreshold struct {
	BaseThreshold   float64
	CurrentThreshold float64
	MinThreshold    float64
	MaxThreshold    float64
	AdaptationRate  float64
	LastUpdated     time.Time
}

type AnomalyPattern struct {
	Type           AnomalyType
	Severity       float64
	DetectionFunc  func(*model.TraceData) bool
	Weight         float64
}

type OnlineLearningParams struct {
	LearningRate    float64
	DecayFactor     float64
	WindowSize      int
	MinSamples      int
	ConvergenceThreshold float64
}

type RealTimeAnomalyDetector struct {
	thresholds       map[AnomalyType]*RealtimeAdaptiveThreshold
	patterns         []AnomalyPattern
	baselineStats    *BaselineStatistics
	learningParams   OnlineLearningParams
	recentAnomalies  []*AnomalyRecord
	featureHistory   []map[string]float64
	mu               sync.RWMutex
	initialized      bool
}

type AnomalyRecord struct {
	Type         AnomalyType
	Score        float64
	TraceID      string
	UserID       string
	SessionID    string
	DetectedAt   time.Time
	FalsePositive bool
}

type BaselineStatistics struct {
	MeanSpeed           float64
	StdSpeed            float64
	MeanTiming          float64
	StdTiming           float64
	MeanDirection       float64
	StdDirection        float64
	MeanCurvature       float64
	StdCurvature        float64
	MeanPressure        float64
	StdPressure         float64
	SampleCount         int
	LastUpdated         time.Time
}

type AlertConfig struct {
	Enabled         bool
	MinSeverity     float64
	CooldownPeriod  time.Duration
	LastAlertTime   time.Time
	AlertCallbacks  []func(*AnomalyAlert)
}

type AnomalyAlert struct {
	AlertID     string
	UserID      string
	SessionID   string
	AnomalyType AnomalyType
	Severity    float64
	Message     string
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

type RealTimeAnomalyService struct {
	detector    *RealTimeAnomalyDetector
	alertConfig *AlertConfig
	alerts      []*AnomalyAlert
	mu          sync.RWMutex
}

func NewRealTimeAnomalyDetector() *RealTimeAnomalyDetector {
	detector := &RealTimeAnomalyDetector{
		thresholds:      make(map[AnomalyType]*RealtimeAdaptiveThreshold),
		patterns:        make([]AnomalyPattern, 0),
		baselineStats:   &BaselineStatistics{},
		learningParams: OnlineLearningParams{
			LearningRate:       0.01,
			DecayFactor:        0.95,
			WindowSize:         100,
			MinSamples:         10,
			ConvergenceThreshold: 0.001,
		},
		recentAnomalies: make([]*AnomalyRecord, 0),
		featureHistory:  make([]map[string]float64, 0),
		initialized:     false,
	}

	detector.initializeDefaultThresholds()
	detector.initializeDefaultPatterns()

	return detector
}

func (d *RealTimeAnomalyDetector) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.baselineStats = &BaselineStatistics{
		MeanSpeed:      0.5,
		StdSpeed:       0.2,
		MeanTiming:     50.0,
		StdTiming:      20.0,
		MeanDirection: 0.0,
		StdDirection:  1.0,
		SampleCount:   0,
		LastUpdated:   time.Now(),
	}

	d.initialized = true
	return nil
}

func (d *RealTimeAnomalyDetector) initializeDefaultThresholds() {
	d.thresholds[AnomalyTypeRapidClicks] = &RealtimeAdaptiveThreshold{
		BaseThreshold:    5.0,
		CurrentThreshold: 5.0,
		MinThreshold:    3.0,
		MaxThreshold:    10.0,
		AdaptationRate:  0.05,
		LastUpdated:    time.Now(),
	}

	d.thresholds[AnomalyTypeHighSpeed] = &RealtimeAdaptiveThreshold{
		BaseThreshold:    10.0,
		CurrentThreshold: 10.0,
		MinThreshold:    5.0,
		MaxThreshold:    20.0,
		AdaptationRate:  0.05,
		LastUpdated:    time.Now(),
	}

	d.thresholds[AnomalyTypeStraightLine] = &RealtimeAdaptiveThreshold{
		BaseThreshold:    0.92,
		CurrentThreshold: 0.92,
		MinThreshold:    0.85,
		MaxThreshold:    0.99,
		AdaptationRate:  0.03,
		LastUpdated:    time.Now(),
	}

	d.thresholds[AnomalyTypeLowJitter] = &RealtimeAdaptiveThreshold{
		BaseThreshold:    0.1,
		CurrentThreshold: 0.1,
		MinThreshold:    0.05,
		MaxThreshold:    0.3,
		AdaptationRate:  0.04,
		LastUpdated:    time.Now(),
	}

	d.thresholds[AnomalyTypeConstantSpeed] = &RealtimeAdaptiveThreshold{
		BaseThreshold:    0.1,
		CurrentThreshold: 0.1,
		MinThreshold:    0.05,
		MaxThreshold:    0.5,
		AdaptationRate:  0.03,
		LastUpdated:    time.Now(),
	}

	d.thresholds[AnomalyTypeBehavioralDrift] = &RealtimeAdaptiveThreshold{
		BaseThreshold:    0.7,
		CurrentThreshold: 0.7,
		MinThreshold:    0.5,
		MaxThreshold:    0.95,
		AdaptationRate:  0.02,
		LastUpdated:    time.Now(),
	}
}

func (d *RealTimeAnomalyDetector) initializeDefaultPatterns() {
	d.patterns = []AnomalyPattern{
		{
			Type:       AnomalyTypeRapidClicks,
			Severity:   0.7,
			Weight:     0.2,
		},
		{
			Type:       AnomalyTypeHighSpeed,
			Severity:   0.8,
			Weight:     0.15,
		},
		{
			Type:       AnomalyTypeStraightLine,
			Severity:   0.6,
			Weight:     0.15,
		},
		{
			Type:       AnomalyTypeLowJitter,
			Severity:   0.5,
			Weight:     0.1,
		},
		{
			Type:       AnomalyTypeConstantSpeed,
			Severity:   0.5,
			Weight:     0.1,
		},
		{
			Type:       AnomalyTypePatternRepetition,
			Severity:   0.9,
			Weight:     0.2,
		},
		{
			Type:       AnomalyTypeNoPause,
			Severity:   0.6,
			Weight:     0.1,
		},
	}
}

func (d *RealTimeAnomalyDetector) Detect(ctx context.Context, traceData *model.TraceData, userID, sessionID string) (*AnomalyDetectionResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.initialized {
		return nil, fmt.Errorf("detector not initialized")
	}

	if traceData == nil || len(traceData.Points) < 2 {
		return &AnomalyDetectionResult{
			IsAnomaly:         false,
			AnomalyType:       []AnomalyType{},
			AnomalyScore:      &AnomalyScore{},
			Confidence:        0.0,
			RiskLevel:         "none",
			Severity:          0.0,
			Threshold:         0.5,
			DetectedAt:        time.Now(),
		}, nil
	}

	score := d.calculateAnomalyScore(traceData)
	detectedTypes := d.detectAnomalyTypes(traceData, score)
	threshold := d.getEffectiveThreshold(detectedTypes)

	isAnomaly := score.OverallScore >= threshold

	result := &AnomalyDetectionResult{
		IsAnomaly:          isAnomaly,
		AnomalyType:        detectedTypes,
		AnomalyScore:       score,
		Confidence:         d.calculateConfidence(traceData, score),
		RiskLevel:          d.determineRiskLevel(score.OverallScore),
		Severity:           d.calculateSeverity(detectedTypes),
		Threshold:          threshold,
		ContributingFactors: d.getContributingFactors(traceData, score),
		Recommendations:    d.generateRecommendations(detectedTypes, score),
		DetectedAt:        time.Now(),
	}

	if isAnomaly {
		record := &AnomalyRecord{
			Type:       detectedTypes[0],
			Score:      score.OverallScore,
			TraceID:    fmt.Sprintf("%s_%d", sessionID, time.Now().UnixNano()),
			UserID:     userID,
			SessionID:  sessionID,
			DetectedAt: time.Now(),
		}
		d.recentAnomalies = append(d.recentAnomalies, record)
		if len(d.recentAnomalies) > 1000 {
			d.recentAnomalies = d.recentAnomalies[len(d.recentAnomalies)-1000:]
		}
	}

	features := d.extractFeatures(traceData)
	d.featureHistory = append(d.featureHistory, features)
	if len(d.featureHistory) > d.learningParams.WindowSize {
		d.featureHistory = d.featureHistory[len(d.featureHistory)-d.learningParams.WindowSize:]
	}

	return result, nil
}

func (d *RealTimeAnomalyDetector) calculateAnomalyScore(traceData *model.TraceData) *AnomalyScore {
	score := &AnomalyScore{}

	points := traceData.Points
	if len(points) < 2 {
		return score
	}

	speeds := d.calculateSpeeds(points)
	timings := d.calculateTimings(points)
	directions := d.calculateDirections(points)

	score.SpeedScore = d.calculateSpeedAnomalyScore(speeds)
	score.TimingScore = d.calculateTimingAnomalyScore(timings)
	score.PatternScore = d.calculatePatternAnomalyScore(points)
	score.DirectionScore = d.calculateDirectionAnomalyScore(directions)
	score.PressureScore = d.calculatePressureAnomalyScore(points)
	score.SessionScore = d.calculateSessionAnomalyScore(traceData)

	weights := map[string]float64{
		"speed":     0.2,
		"timing":    0.15,
		"pattern":   0.25,
		"direction": 0.15,
		"pressure":  0.1,
		"session":   0.15,
	}

	score.OverallScore = 
		score.SpeedScore * weights["speed"] +
		score.TimingScore * weights["timing"] +
		score.PatternScore * weights["pattern"] +
		score.DirectionScore * weights["direction"] +
		score.PressureScore * weights["pressure"] +
		score.SessionScore * weights["session"]

	return score
}

func (d *RealTimeAnomalyDetector) calculateSpeeds(points []model.TracePoint) []float64 {
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, dist/dt)
		}
	}
	return speeds
}

func (d *RealTimeAnomalyDetector) calculateTimings(points []model.TracePoint) []float64 {
	timings := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		timings = append(timings, dt)
	}
	return timings
}

func (d *RealTimeAnomalyDetector) calculateDirections(points []model.TracePoint) []float64 {
	directions := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		angle := math.Atan2(dy, dx)
		directions = append(directions, angle)
	}
	return directions
}

func (d *RealTimeAnomalyDetector) calculateCurvatures(points []model.TracePoint) []float64 {
	if len(points) < 3 {
		return []float64{}
	}

	curvatures := make([]float64, 0, len(points)-2)
	for i := 1; i < len(points)-1; i++ {
		curvatures = append(curvatures, computeCurvature(points[i-1], points[i], points[i+1]))
	}
	return curvatures
}

func computeCurvature(p1, p2, p3 model.TracePoint) float64 {
	v1x := p2.X - p1.X
	v1y := p2.Y - p1.Y
	v2x := p3.X - p2.X
	v2y := p3.Y - p2.Y

	dot := v1x*v2x + v1y*v2y
	mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
	mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	cosAngle := dot / (mag1 * mag2)
	if cosAngle > 1 {
		cosAngle = 1
	}
	if cosAngle < -1 {
		cosAngle = -1
	}
	return math.Acos(cosAngle)
}

func (d *RealTimeAnomalyDetector) calculateSpeedAnomalyScore(speeds []float64) float64 {
	if len(speeds) == 0 {
		return 0.0
	}

	maxSpeed := 0.0
	for _, s := range speeds {
		if s > maxSpeed {
			maxSpeed = s
		}
	}

	highSpeedThreshold := d.thresholds[AnomalyTypeHighSpeed].CurrentThreshold
	if maxSpeed > highSpeedThreshold {
		return math.Min(1.0, (maxSpeed-highSpeedThreshold)/highSpeedThreshold)
	}

	speedVariance := calculateVariance(speeds)
	if d.baselineStats.StdSpeed > 0 {
		zScore := math.Abs(speedVariance - d.baselineStats.MeanSpeed) / d.baselineStats.StdSpeed
		return math.Min(1.0, zScore/3.0)
	}

	return 0.0
}

func (d *RealTimeAnomalyDetector) calculateTimingAnomalyScore(timings []float64) float64 {
	if len(timings) == 0 {
		return 0.0
	}

	avgTiming := calculateMean(timings)
	if avgTiming < 50 {
		return math.Min(1.0, (50-avgTiming)/50.0)
	}

	timingVariance := calculateVariance(timings)
	if d.baselineStats.StdTiming > 0 {
		zScore := math.Abs(timingVariance - d.baselineStats.MeanTiming) / d.baselineStats.StdTiming
		return math.Min(1.0, zScore/3.0)
	}

	return 0.0
}

func (d *RealTimeAnomalyDetector) calculatePatternAnomalyScore(points []model.TracePoint) float64 {
	if len(points) < 5 {
		return 0.0
	}

	start, end := points[0], points[len(points)-1]
	straightDist := math.Sqrt((end.X-start.X)*(end.X-start.X) + (end.Y-start.Y)*(end.Y-start.Y))
	totalDist := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}

	if totalDist < 100 {
		return 0.0
	}

	straightnessRatio := straightDist / totalDist
	threshold := d.thresholds[AnomalyTypeStraightLine].CurrentThreshold

	if straightnessRatio > threshold {
		return math.Min(1.0, (straightnessRatio-threshold)/(1.0-threshold))
	}

	return 0.0
}

func (d *RealTimeAnomalyDetector) calculateDirectionAnomalyScore(directions []float64) float64 {
	if len(directions) < 2 {
		return 0.0
	}

	dirChanges := 0
	for i := 1; i < len(directions); i++ {
		change := math.Abs(directions[i] - directions[i-1])
		if change > math.Pi {
			change = 2*math.Pi - change
		}
		if change > 0.5 {
			dirChanges++
		}
	}

	changeRatio := float64(dirChanges) / float64(len(directions))
	if changeRatio < 0.1 {
		return math.Min(1.0, (0.1-changeRatio)/0.1)
	}

	return 0.0
}

func (d *RealTimeAnomalyDetector) calculatePressureAnomalyScore(points []model.TracePoint) float64 {
	pressures := make([]float64, 0)
	for _, p := range points {
		if p.Pressure > 0 {
			pressures = append(pressures, p.Pressure)
		}
	}

	if len(pressures) < 2 {
		return 0.0
	}

	variance := calculateVariance(pressures)
	if variance < d.thresholds[AnomalyTypeLowJitter].CurrentThreshold {
		return math.Min(1.0, (d.thresholds[AnomalyTypeLowJitter].CurrentThreshold-variance)/d.thresholds[AnomalyTypeLowJitter].CurrentThreshold)
	}

	return 0.0
}

func (d *RealTimeAnomalyDetector) calculateSessionAnomalyScore(traceData *model.TraceData) float64 {
	totalTime := float64(traceData.TotalTime)
	if totalTime < 1000 {
		return math.Min(1.0, (1000-totalTime)/1000.0)
	}

	return 0.0
}

func (d *RealTimeAnomalyDetector) detectAnomalyTypes(traceData *model.TraceData, score *AnomalyScore) []AnomalyType {
	var types []AnomalyType
	points := traceData.Points

	if len(points) >= 5 {
		speeds := d.calculateSpeeds(points)
		maxSpeed := 0.0
		for _, s := range speeds {
			if s > maxSpeed {
				maxSpeed = s
			}
		}
		if maxSpeed > d.thresholds[AnomalyTypeHighSpeed].CurrentThreshold {
			types = append(types, AnomalyTypeHighSpeed)
		}
	}

	clickCount := 0
	for _, p := range points {
		if p.Event == "click" {
			clickCount++
		}
	}
	if float64(clickCount) > d.thresholds[AnomalyTypeRapidClicks].CurrentThreshold {
		types = append(types, AnomalyTypeRapidClicks)
	}

	start, end := points[0], points[len(points)-1]
	straightDist := math.Sqrt((end.X-start.X)*(end.X-start.X) + (end.Y-start.Y)*(end.Y-start.Y))
	totalDist := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}
	if totalDist > 100 && straightDist/totalDist > d.thresholds[AnomalyTypeStraightLine].CurrentThreshold {
		types = append(types, AnomalyTypeStraightLine)
	}

	if score.PatternScore > 0.7 {
		types = append(types, AnomalyTypePatternRepetition)
	}

	speeds := d.calculateSpeeds(points)
	if len(speeds) >= 2 {
		variance := calculateVariance(speeds)
		mean := calculateMean(speeds)
		if mean > 0 && variance/mean < d.thresholds[AnomalyTypeConstantSpeed].CurrentThreshold {
			types = append(types, AnomalyTypeConstantSpeed)
		}
	}

	return types
}

func (d *RealTimeAnomalyDetector) getEffectiveThreshold(types []AnomalyType) float64 {
	if len(types) == 0 {
		return 0.5
	}

	maxThreshold := 0.0
	for _, t := range types {
		if threshold, exists := d.thresholds[t]; exists {
			if threshold.CurrentThreshold > maxThreshold {
				maxThreshold = threshold.CurrentThreshold
			}
		}
	}

	return maxThreshold
}

func (d *RealTimeAnomalyDetector) calculateConfidence(traceData *model.TraceData, score *AnomalyScore) float64 {
	pointCount := len(traceData.Points)
	baseConfidence := math.Min(1.0, float64(pointCount)/50.0)

	scoreConfidence := 1.0 - math.Abs(score.OverallScore-0.5)*2

	return (baseConfidence + scoreConfidence) / 2.0
}

func (d *RealTimeAnomalyDetector) determineRiskLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "critical"
	case score >= 0.6:
		return "high"
	case score >= 0.4:
		return "medium"
	case score >= 0.2:
		return "low"
	default:
		return "none"
	}
}

func (d *RealTimeAnomalyDetector) calculateSeverity(types []AnomalyType) float64 {
	if len(types) == 0 {
		return 0.0
	}

	maxSeverity := 0.0
	for _, pattern := range d.patterns {
		for _, t := range types {
			if pattern.Type == t && pattern.Severity > maxSeverity {
				maxSeverity = pattern.Severity
			}
		}
	}

	return maxSeverity
}

func (d *RealTimeAnomalyDetector) getContributingFactors(traceData *model.TraceData, score *AnomalyScore) []string {
	var factors []string

	if score.SpeedScore > 0.5 {
		factors = append(factors, "异常速度模式")
	}
	if score.TimingScore > 0.5 {
		factors = append(factors, "异常时间间隔")
	}
	if score.PatternScore > 0.5 {
		factors = append(factors, "异常行为模式")
	}
	if score.DirectionScore > 0.5 {
		factors = append(factors, "异常方向变化")
	}
	if score.PressureScore > 0.5 {
		factors = append(factors, "异常压力模式")
	}
	if score.SessionScore > 0.5 {
		factors = append(factors, "异常会话特征")
	}

	return factors
}

func (d *RealTimeAnomalyDetector) generateRecommendations(types []AnomalyType, score *AnomalyScore) []string {
	var recommendations []string

	if len(types) == 0 {
		recommendations = append(recommendations, "当前行为模式正常")
		return recommendations
	}

	for _, t := range types {
		switch t {
		case AnomalyTypeHighSpeed:
			recommendations = append(recommendations, "检测到异常高速度移动，建议增加验证")
		case AnomalyTypeRapidClicks:
			recommendations = append(recommendations, "检测到快速连续点击，建议进行人机验证")
		case AnomalyTypeStraightLine:
			recommendations = append(recommendations, "检测到过于笔直的移动路径，可能为机器操作")
		case AnomalyTypePatternRepetition:
			recommendations = append(recommendations, "检测到重复性行为模式，建议加强验证")
		case AnomalyTypeConstantSpeed:
			recommendations = append(recommendations, "检测到恒定速度移动，疑似自动化操作")
		case AnomalyTypeLowJitter:
			recommendations = append(recommendations, "检测到异常稳定的压力模式，可能为机器操作")
		}
	}

	if score.OverallScore > 0.7 {
		recommendations = append(recommendations, "综合风险评分较高，建议阻断当前操作")
	}

	return recommendations
}

func (d *RealTimeAnomalyDetector) extractFeatures(traceData *model.TraceData) map[string]float64 {
	features := make(map[string]float64)

	if len(traceData.Points) == 0 {
		return features
	}

	points := traceData.Points
	speeds := d.calculateSpeeds(points)
	timings := d.calculateTimings(points)

	features["point_count"] = float64(len(points))
	features["total_time"] = float64(traceData.TotalTime)

	if len(speeds) > 0 {
		features["mean_speed"] = calculateMean(speeds)
		features["max_speed"] = calculateMax(speeds)
		features["speed_variance"] = calculateVariance(speeds)
	}

	if len(timings) > 0 {
		features["mean_timing"] = calculateMean(timings)
		features["timing_variance"] = calculateVariance(timings)
	}

	return features
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}
	mean := calculateMean(values)
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func calculateRealtimeMax(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func calculateMax(values []float64) float64 {
	return calculateRealtimeMax(values)
}

func (d *RealTimeAnomalyDetector) UpdateBaseline(features map[string]float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.baselineStats == nil {
		d.baselineStats = &BaselineStatistics{}
	}

	alpha := d.learningParams.LearningRate

	if val, ok := features["mean_speed"]; ok {
		if d.baselineStats.SampleCount == 0 {
			d.baselineStats.MeanSpeed = val
		} else {
			d.baselineStats.MeanSpeed = d.baselineStats.MeanSpeed*(1-alpha) + val*alpha
		}
	}

	if val, ok := features["speed_variance"]; ok {
		if d.baselineStats.SampleCount == 0 {
			d.baselineStats.StdSpeed = val
		} else {
			d.baselineStats.StdSpeed = d.baselineStats.StdSpeed*(1-alpha) + val*alpha
		}
	}

	if val, ok := features["mean_timing"]; ok {
		if d.baselineStats.SampleCount == 0 {
			d.baselineStats.MeanTiming = val
		} else {
			d.baselineStats.MeanTiming = d.baselineStats.MeanTiming*(1-alpha) + val*alpha
		}
	}

	d.baselineStats.SampleCount++
	d.baselineStats.LastUpdated = time.Now()

	if len(d.featureHistory) >= d.learningParams.MinSamples {
		d.adaptThresholds()
	}
}

func (d *RealTimeAnomalyDetector) adaptThresholds() {
	falsePositiveRate := d.calculateFalsePositiveRate()

	for _, t := range d.thresholds {
		if falsePositiveRate > 0.2 {
			t.CurrentThreshold = math.Min(t.MaxThreshold, t.CurrentThreshold*(1+t.AdaptationRate))
		} else if falsePositiveRate < 0.05 {
			t.CurrentThreshold = math.Max(t.MinThreshold, t.CurrentThreshold*(1-t.AdaptationRate))
		}
		t.LastUpdated = time.Now()
	}
}

func (d *RealTimeAnomalyDetector) calculateFalsePositiveRate() float64 {
	if len(d.recentAnomalies) < 10 {
		return 0.1
	}

	falsePositives := 0
	for _, record := range d.recentAnomalies[len(d.recentAnomalies)-10:] {
		if record.FalsePositive {
			falsePositives++
		}
	}

	return float64(falsePositives) / 10.0
}

func (d *RealTimeAnomalyDetector) RecordFeedback(traceID string, isCorrect bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, record := range d.recentAnomalies {
		if record.TraceID == traceID {
			record.FalsePositive = !isCorrect
			break
		}
	}
}

func (d *RealTimeAnomalyDetector) GetThresholds() map[AnomalyType]*RealtimeAdaptiveThreshold {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[AnomalyType]*RealtimeAdaptiveThreshold)
	for k, v := range d.thresholds {
		result[k] = &RealtimeAdaptiveThreshold{
			BaseThreshold:    v.BaseThreshold,
			CurrentThreshold: v.CurrentThreshold,
			MinThreshold:    v.MinThreshold,
			MaxThreshold:    v.MaxThreshold,
			AdaptationRate:  v.AdaptationRate,
			LastUpdated:     v.LastUpdated,
		}
	}

	return result
}

func (d *RealTimeAnomalyDetector) GetBaselineStats() *BaselineStatistics {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.baselineStats == nil {
		return nil
	}

	return &BaselineStatistics{
		MeanSpeed:   d.baselineStats.MeanSpeed,
		StdSpeed:    d.baselineStats.StdSpeed,
		MeanTiming:  d.baselineStats.MeanTiming,
		StdTiming:  d.baselineStats.StdTiming,
		SampleCount: d.baselineStats.SampleCount,
		LastUpdated: d.baselineStats.LastUpdated,
	}
}

func (d *RealTimeAnomalyDetector) GetRecentAnomalies(limit int) []*AnomalyRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 || limit >= len(d.recentAnomalies) {
		result := make([]*AnomalyRecord, len(d.recentAnomalies))
		copy(result, d.recentAnomalies)
		return result
	}

	result := make([]*AnomalyRecord, limit)
	copy(result, d.recentAnomalies[len(d.recentAnomalies)-limit:])
	return result
}

func NewRealTimeAnomalyService() *RealTimeAnomalyService {
	return &RealTimeAnomalyService{
		detector: NewRealTimeAnomalyDetector(),
		alertConfig: &AlertConfig{
			Enabled:        true,
			MinSeverity:    0.7,
			CooldownPeriod: 1 * time.Minute,
		},
		alerts: make([]*AnomalyAlert, 0),
	}
}

func (s *RealTimeAnomalyService) Initialize(ctx context.Context) error {
	return s.detector.Initialize(ctx)
}

func (s *RealTimeAnomalyService) DetectAnomaly(ctx context.Context, traceData *model.TraceData, userID, sessionID string) (*AnomalyDetectionResult, error) {
	result, err := s.detector.Detect(ctx, traceData, userID, sessionID)
	if err != nil {
		return nil, err
	}

	if result.IsAnomaly && s.alertConfig.Enabled && result.Severity >= s.alertConfig.MinSeverity {
		s.checkAndTriggerAlert(result, userID, sessionID)
	}

	features := s.detector.extractFeatures(traceData)
	s.detector.UpdateBaseline(features)

	return result, nil
}

func (s *RealTimeAnomalyService) checkAndTriggerAlert(result *AnomalyDetectionResult, userID, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Since(s.alertConfig.LastAlertTime) < s.alertConfig.CooldownPeriod {
		return
	}

	alert := &AnomalyAlert{
		AlertID:     fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		UserID:      userID,
		SessionID:  sessionID,
		AnomalyType: result.AnomalyType[0],
		Severity:   result.Severity,
		Message:    fmt.Sprintf("检测到异常行为: %s, 严重程度: %.2f", result.AnomalyType[0], result.Severity),
		Timestamp:  time.Now(),
		Metadata:   map[string]interface{}{},
	}

	s.alerts = append(s.alerts, alert)
	s.alertConfig.LastAlertTime = time.Now()

	for _, callback := range s.alertConfig.AlertCallbacks {
		callback(alert)
	}
}

func (s *RealTimeAnomalyService) RegisterAlertCallback(callback func(*AnomalyAlert)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertConfig.AlertCallbacks = append(s.alertConfig.AlertCallbacks, callback)
}

func (s *RealTimeAnomalyService) GetAlerts(limit int) []*AnomalyAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit >= len(s.alerts) {
		result := make([]*AnomalyAlert, len(s.alerts))
		copy(result, s.alerts)
		return result
	}

	result := make([]*AnomalyAlert, limit)
	copy(result, s.alerts[len(s.alerts)-limit:])
	return result
}

func (s *RealTimeAnomalyService) RecordFeedback(traceID string, isCorrect bool) {
	s.detector.RecordFeedback(traceID, isCorrect)
}

func (s *RealTimeAnomalyService) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_alerts": len(s.alerts),
		"recent_anomalies": len(s.detector.recentAnomalies),
		"feature_history_size": len(s.detector.featureHistory),
	}

	baseline := s.detector.GetBaselineStats()
	if baseline != nil {
		stats["baseline_sample_count"] = baseline.SampleCount
		stats["baseline_mean_speed"] = baseline.MeanSpeed
		stats["baseline_std_speed"] = baseline.StdSpeed
	}

	return stats
}

func (s *RealTimeAnomalyService) EnableAlerts(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertConfig.Enabled = enabled
}

func (s *RealTimeAnomalyService) SetAlertThreshold(severity float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertConfig.MinSeverity = severity
}

func (s *RealTimeAnomalyService) SetCooldownPeriod(period time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertConfig.CooldownPeriod = period
}
