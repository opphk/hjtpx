package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type BehaviorDataPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Event     string  `json:"event"`
}

type MouseTrajectory struct {
	Points           []BehaviorDataPoint `json:"points"`
	TotalDistance    float64             `json:"total_distance"`
	AverageSpeed     float64             `json:"average_speed"`
	MaxSpeed         float64             `json:"max_speed"`
	MinSpeed         float64             `json:"min_speed"`
	SpeedVariance    float64             `json:"speed_variance"`
	PathEfficiency   float64             `json:"path_efficiency"`
	DirectionChanges int                 `json:"direction_changes"`
	Smoothness       float64             `json:"smoothness"`
	TrajectoryLength  float64             `json:"trajectory_length"`
	StayPoints       []StayPoint         `json:"stay_points"`
}

type StayPoint struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Duration int64   `json:"duration"`
}

type ClickPattern struct {
	Clicks          []BehaviorDataPoint `json:"clicks"`
	ClickCount     int                 `json:"click_count"`
	AverageInterval float64             `json:"average_interval"`
	ClickSpeed     float64             `json:"click_speed"`
	Regularity     float64             `json:"regularity"`
	ClickPositions []ClickPosition     `json:"click_positions"`
}

type ClickPosition struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Button int     `json:"button"`
}

type AnalysisResult struct {
	Trajectory      MouseTrajectory  `json:"trajectory"`
	ClickPattern    ClickPattern     `json:"click_pattern"`
	RiskScore       float64          `json:"risk_score"`
	RiskIndicators  []string         `json:"risk_indicators"`
	IsBotLikely     bool             `json:"is_bot_likely"`
	Confidence      float64          `json:"confidence"`
	Features        FeatureVector    `json:"features"`
	MLPrediction    *MLPrediction    `json:"ml_prediction,omitempty"`
}

type FeatureVector struct {
	MouseSpeedAvg      float64 `json:"mouse_speed_avg"`
	MouseSpeedMax      float64 `json:"mouse_speed_max"`
	MouseSpeedMin      float64 `json:"mouse_speed_min"`
	MouseSpeedVar      float64 `json:"mouse_speed_var"`
	AccelerationAvg    float64 `json:"acceleration_avg"`
	AccelerationMax    float64 `json:"acceleration_max"`
	TrajectoryLength   float64 `json:"trajectory_length"`
	PathEfficiency     float64 `json:"path_efficiency"`
	DirectionChanges   int     `json:"direction_changes"`
	Smoothness        float64 `json:"smoothness"`
	ClickCount        int     `json:"click_count"`
	ClickIntervalAvg   float64 `json:"click_interval_avg"`
	ClickRegularity    float64 `json:"click_regularity"`
	ClickSpeed        float64 `json:"click_speed"`
	TotalDuration     int64   `json:"total_duration"`
	DataPointDensity  float64 `json:"data_point_density"`
	IsHeadless       bool    `json:"is_headless"`
	ScreenConsistency bool    `json:"screen_consistency"`
	TimezoneOffset    int     `json:"timezone_offset"`
	FeatureVector     []float64 `json:"vector"`
}

type MLPrediction struct {
	IsBot         bool    `json:"is_bot"`
	Confidence    float64 `json:"confidence"`
	BotScore      float64 `json:"bot_score"`
	HumanScore    float64 `json:"human_score"`
	ModelVersion  string  `json:"model_version"`
	FeaturesUsed  []string `json:"features_used"`
}

type ExtendedBehaviorData struct {
	MouseTrajectory    []MousePoint        `json:"mouse_trajectory"`
	MouseSpeed         []SpeedDataPoint    `json:"mouse_speed"`
	MouseAcceleration  []AccelerationPoint `json:"mouse_acceleration"`
	ClickData          []ClickInfo         `json:"click_data"`
	HoverData          []HoverInfo         `json:"hover_data"`
	ScrollData         []ScrollInfo        `json:"scroll_data"`
	KeyStrokeData      []KeyStrokeInfo     `json:"key_stroke_data"`
	EnvironmentData    EnvironmentInfo     `json:"environment_data"`
}

type MousePoint struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
}

type SpeedDataPoint struct {
	Speed       float64 `json:"speed"`
	Direction   float64 `json:"direction"`
	Timestamp   int64   `json:"timestamp"`
}

type AccelerationPoint struct {
	Acceleration float64 `json:"acceleration"`
	Timestamp    int64   `json:"timestamp"`
}

type ClickInfo struct {
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Button       int     `json:"button"`
	Timestamp    int64   `json:"timestamp"`
	HoldDuration int64   `json:"hold_duration"`
}

type HoverInfo struct {
	ElementID    string  `json:"element_id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	EnterTime    int64   `json:"enter_time"`
	LeaveTime    int64   `json:"leave_time"`
	Duration     int64   `json:"duration"`
}

type ScrollInfo struct {
	ScrollX      int64   `json:"scroll_x"`
	ScrollY      int64   `json:"scroll_y"`
	DeltaX       int64   `json:"delta_x"`
	DeltaY       int64   `json:"delta_y"`
	Timestamp    int64   `json:"timestamp"`
	Velocity     float64 `json:"velocity"`
}

type KeyStrokeInfo struct {
	Key         string `json:"key"`
	KeyCode     int    `json:"key_code"`
	Timestamp   int64  `json:"timestamp"`
	HoldTime    int64  `json:"hold_time"`
	IsModifier  bool   `json:"is_modifier"`
}

type EnvironmentInfo struct {
	ScreenWidth       int      `json:"screen_width"`
	ScreenHeight      int      `json:"screen_height"`
	WindowWidth       int      `json:"window_width"`
	WindowHeight      int      `json:"window_height"`
	ColorDepth        int      `json:"color_depth"`
	Timezone          string   `json:"timezone"`
	Language          string   `json:"language"`
	Platform          string   `json:"platform"`
	UserAgent         string   `json:"user_agent"`
	Plugins           []string `json:"plugins"`
	WebGLRenderer     string   `json:"webgl_renderer"`
	CanvasFingerprint string   `json:"canvas_fingerprint"`
	IsHeadless        bool     `json:"is_headless"`
	HasTouchSupport   bool     `json:"has_touch_support"`
}

type KeyStrokeAnalysis struct {
	TotalKeystrokes   int           `json:"total_keystrokes"`
	AverageInterval   float64       `json:"average_interval"`
	IntervalVariance  float64       `json:"interval_variance"`
	AverageHoldTime   float64       `json:"average_hold_time"`
	ModifierUsage     float64       `json:"modifier_usage"`
	IsTypingPattern   bool          `json:"is_typing_pattern"`
	TypingRhythm      float64       `json:"typing_rhythm"`
}

type ScrollAnalysis struct {
	TotalScrolls    int           `json:"total_scrolls"`
	AverageVelocity float64       `json:"average_velocity"`
	MaxVelocity     float64       `json:"max_velocity"`
	ScrollPattern   float64       `json:"scroll_pattern"`
	DirectionCount  int           `json:"direction_count"`
}

type RiskWeights struct {
	SpeedWeight           float64
	AccelerationWeight    float64
	TrajectoryWeight      float64
	ClickWeight           float64
	KeyboardWeight        float64
	EnvironmentWeight     float64
}

type BehaviorAnalysisService struct {
	modelService   *ModelService
	weights        RiskWeights
	cache          *AnalysisCache
	cacheMutex     sync.RWMutex
	streamProcessor *StreamProcessor
}

type AnalysisCache struct {
	entries map[string]*CachedResult
	maxSize int
}

type CachedResult struct {
	Result     *AnalysisResult
	ExpiresAt  time.Time
}

type StreamProcessor struct {
	buffer      []BehaviorDataPoint
	maxBufferSize int
	analyzeCallback func(*AnalysisResult)
}

func NewBehaviorAnalysisService() *BehaviorAnalysisService {
	service := &BehaviorAnalysisService{
		weights: RiskWeights{
			SpeedWeight:        0.2,
			AccelerationWeight: 0.15,
			TrajectoryWeight:   0.25,
			ClickWeight:        0.2,
			KeyboardWeight:     0.1,
			EnvironmentWeight:  0.1,
		},
		cache: &AnalysisCache{
			entries: make(map[string]*CachedResult),
			maxSize: 1000,
		},
		streamProcessor: &StreamProcessor{
			buffer: make([]BehaviorDataPoint, 0),
			maxBufferSize: 1000,
		},
	}
	
	modelService := NewModelService()
	service.modelService = modelService
	
	return service
}

func (s *BehaviorAnalysisService) AnalyzeBehavior(behaviorData []models.BehaviorData) (*AnalysisResult, error) {
	result := &AnalysisResult{
		RiskIndicators: []string{},
	}

	var points []BehaviorDataPoint
	var clicks []BehaviorDataPoint
	var extendedData *ExtendedBehaviorData

	for _, bd := range behaviorData {
		if bd.DataType == "extended" || bd.DataType == "extended_collection" {
			var ext ExtendedBehaviorData
			if err := json.Unmarshal([]byte(bd.Data), &ext); err == nil {
				extendedData = &ext
			}
			continue
		}

		var dp BehaviorDataPoint
		if err := json.Unmarshal([]byte(bd.Data), &dp); err == nil {
			points = append(points, dp)
			if dp.Event == "click" {
				clicks = append(clicks, dp)
			}
		}
	}

	if len(points) > 0 {
		result.Trajectory = s.analyzeMouseTrajectory(points)
	}

	if len(clicks) > 0 {
		result.ClickPattern = s.analyzeClickPattern(clicks)
	}

	s.extractFeatures(result, points, extendedData)

	if s.modelService != nil {
		prediction := s.modelService.Predict(&result.Features)
		result.MLPrediction = prediction
	}

	s.calculateRiskScore(result, extendedData)

	return result, nil
}

func (s *BehaviorAnalysisService) analyzeMouseTrajectory(points []BehaviorDataPoint) MouseTrajectory {
	traj := MouseTrajectory{
		Points: points,
	}

	if len(points) < 2 {
		return traj
	}

	var totalDistance, totalSpeed float64
	var maxSpeed, minSpeed float64 = 0, math.MaxFloat64
	speeds := []float64{}
	accelerations := []float64{}
	directionChanges := 0
	prevAngle := 0.0
	prevSpeed := 0.0

	traj.StayPoints = s.detectStayPoints(points)

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance

		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speed := distance / dt
			speeds = append(speeds, speed)
			totalSpeed += speed

			if speed > maxSpeed {
				maxSpeed = speed
			}
			if speed < minSpeed {
				minSpeed = speed
			}

			acceleration := (speed - prevSpeed) / dt
			accelerations = append(accelerations, math.Abs(acceleration))

			prevSpeed = speed
		}

		if i > 1 {
			angle := math.Atan2(dy, dx)
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > 0.5 && angleDiff < math.Pi-0.5 {
				directionChanges++
			}
			prevAngle = angle
		}
	}

	traj.TotalDistance = totalDistance
	traj.MaxSpeed = maxSpeed
	traj.MinSpeed = minSpeed
	traj.DirectionChanges = directionChanges

	if len(speeds) > 0 {
		traj.AverageSpeed = totalSpeed / float64(len(speeds))
		variance := s.calculateVariance(speeds, traj.AverageSpeed)
		traj.SpeedVariance = variance
	}

	firstPoint := points[0]
	lastPoint := points[len(points)-1]
	straightDistance := math.Sqrt(
		math.Pow(float64(lastPoint.X-firstPoint.X), 2) +
			math.Pow(float64(lastPoint.Y-firstPoint.Y), 2),
	)

	if totalDistance > 0 {
		traj.PathEfficiency = straightDistance / totalDistance
	}

	traj.TrajectoryLength = s.calculateTrajectoryLength(points)
	traj.Smoothness = s.calculateSmoothness(points)

	return traj
}

func (s *BehaviorAnalysisService) detectStayPoints(points []BehaviorDataPoint) []StayPoint {
	if len(points) < 3 {
		return nil
	}

	stayPoints := []StayPoint{}
	threshold := 5.0
	minDuration := int64(100)

	var currentStay *StayPoint
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance < threshold {
			if currentStay == nil {
				currentStay = &StayPoint{
					X: float64(points[i].X),
					Y: float64(points[i].Y),
					Duration: points[i].Timestamp - points[0].Timestamp,
				}
			} else {
				currentStay.Duration = points[i].Timestamp - points[0].Timestamp
			}
		} else {
			if currentStay != nil && currentStay.Duration >= minDuration {
				stayPoints = append(stayPoints, *currentStay)
			}
			currentStay = nil
		}
	}

	if currentStay != nil && currentStay.Duration >= minDuration {
		stayPoints = append(stayPoints, *currentStay)
	}

	return stayPoints
}

func (s *BehaviorAnalysisService) calculateTrajectoryLength(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	totalLength := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalLength += math.Sqrt(dx*dx + dy*dy)
	}

	return totalLength
}

func (s *BehaviorAnalysisService) calculateSmoothness(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 1.0
	}

	var totalCurvature float64
	count := 0

	for i := 1; i < len(points)-1; i++ {
		v1x := float64(points[i].X - points[i-1].X)
		v1y := float64(points[i].Y - points[i-1].Y)
		v2x := float64(points[i+1].X - points[i].X)
		v2y := float64(points[i+1].Y - points[i].Y)

		dot := v1x*v2x + v1y*v2y
		len1 := math.Sqrt(v1x*v1x + v1y*v1y)
		len2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if len1 > 0 && len2 > 0 {
			cosAngle := dot / (len1 * len2)
			cosAngle = math.Max(-1, math.Min(1, cosAngle))
			angle := math.Acos(cosAngle)
			totalCurvature += angle
			count++
		}
	}

	if count > 0 {
		avgCurvature := totalCurvature / float64(count)
		smoothness := 1.0 - (avgCurvature / math.Pi)
		return math.Max(0, smoothness)
	}

	return 1.0
}

func (s *BehaviorAnalysisService) analyzeClickPattern(clicks []BehaviorDataPoint) ClickPattern {
	pattern := ClickPattern{
		Clicks:      clicks,
		ClickCount: len(clicks),
	}

	if len(clicks) < 2 {
		return pattern
	}

	intervals := []float64{}
	positions := []ClickPosition{}

	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	for _, click := range clicks {
		positions = append(positions, ClickPosition{
			X:      float64(click.X),
			Y:      float64(click.Y),
			Button: 0,
		})
	}
	pattern.ClickPositions = positions

	if len(intervals) > 0 {
		avgInterval := 0.0
		for _, interval := range intervals {
			avgInterval += interval
		}
		avgInterval = avgInterval / float64(len(intervals))
		pattern.AverageInterval = avgInterval

		variance := s.calculateVariance(intervals, avgInterval)
		if avgInterval > 0 {
			pattern.Regularity = 1 - (math.Sqrt(variance) / avgInterval)
			pattern.Regularity = math.Max(0, pattern.Regularity)
		}
	}

	totalTime := float64(clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp)
	if totalTime > 0 {
		pattern.ClickSpeed = float64(len(clicks)) / (totalTime / 1000)
	}

	return pattern
}

func (s *BehaviorAnalysisService) calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(values))
}

func (s *BehaviorAnalysisService) extractFeatures(result *AnalysisResult, points []BehaviorDataPoint, extendedData *ExtendedBehaviorData) {
	features := FeatureVector{}

	features.MouseSpeedAvg = result.Trajectory.AverageSpeed
	features.MouseSpeedMax = result.Trajectory.MaxSpeed
	features.MouseSpeedMin = result.Trajectory.MinSpeed
	features.MouseSpeedVar = result.Trajectory.SpeedVariance
	features.TrajectoryLength = result.Trajectory.TotalDistance
	features.PathEfficiency = result.Trajectory.PathEfficiency
	features.DirectionChanges = result.Trajectory.DirectionChanges
	features.Smoothness = result.Trajectory.Smoothness

	features.ClickCount = result.ClickPattern.ClickCount
	features.ClickIntervalAvg = result.ClickPattern.AverageInterval
	features.ClickRegularity = result.ClickPattern.Regularity
	features.ClickSpeed = result.ClickPattern.ClickSpeed

	if len(points) >= 2 {
		features.TotalDuration = points[len(points)-1].Timestamp - points[0].Timestamp
		if features.TotalDuration > 0 {
			features.DataPointDensity = float64(len(points)) / (float64(features.TotalDuration) / 1000)
		}
	}

	if extendedData != nil {
		features.IsHeadless = extendedData.EnvironmentData.IsHeadless
		features.ScreenConsistency = s.checkScreenConsistency(extendedData)
		features.TimezoneOffset = s.parseTimezoneOffset(extendedData.EnvironmentData.Timezone)
	}

	featureVector := []float64{
		features.MouseSpeedAvg,
		features.MouseSpeedMax,
		features.MouseSpeedMin,
		features.MouseSpeedVar,
		features.TrajectoryLength / 1000,
		features.PathEfficiency,
		float64(features.DirectionChanges) / 10,
		features.Smoothness,
		float64(features.ClickCount) / 10,
		features.ClickRegularity,
		features.ClickSpeed / 10,
		features.DataPointDensity,
	}
	features.FeatureVector = featureVector

	result.Features = features
}

func (s *BehaviorAnalysisService) checkScreenConsistency(data *ExtendedBehaviorData) bool {
	if data == nil || data.EnvironmentData.ScreenWidth == 0 {
		return true
	}

	aspectRatio := float64(data.EnvironmentData.ScreenWidth) / float64(data.EnvironmentData.ScreenHeight)
	validAspect := aspectRatio >= 0.5 && aspectRatio <= 3.0

	commonResolutions := map[string]bool{
		"1920x1080": true,
		"1366x768":  true,
		"1536x864":  true,
		"1440x900":  true,
		"1280x720":  true,
		"2560x1440": true,
	}

	resKey := fmt.Sprintf("%dx%d", data.EnvironmentData.ScreenWidth, data.EnvironmentData.ScreenHeight)
	isCommon := commonResolutions[resKey]

	return validAspect || isCommon
}

func (s *BehaviorAnalysisService) parseTimezoneOffset(timezone string) int {
	var offset int
	fmt.Sscanf(timezone, "UTC%d", &offset)
	return offset
}

func (s *BehaviorAnalysisService) calculateRiskScore(result *AnalysisResult, extendedData *ExtendedBehaviorData) {
	riskScore := 0.0
	indicators := []string{}

	if result.Trajectory.AverageSpeed > 0 {
		if result.Trajectory.AverageSpeed > 5 {
			riskScore += 20 * s.weights.SpeedWeight * 100
			indicators = append(indicators, "异常高速移动")
		}
		if result.Trajectory.AverageSpeed < 0.1 {
			riskScore += 15 * s.weights.SpeedWeight * 100
			indicators = append(indicators, "异常低速移动")
		}
	}

	if result.Trajectory.SpeedVariance > 100 {
		riskScore += 10 * s.weights.SpeedWeight * 100
		indicators = append(indicators, "速度变化过大")
	}

	if result.Trajectory.PathEfficiency > 0.95 && result.Trajectory.TotalDistance > 100 {
		riskScore += 25 * s.weights.TrajectoryWeight * 100
		indicators = append(indicators, "路径过于笔直")
	}

	if result.Trajectory.Smoothness < 0.3 {
		riskScore += 15 * s.weights.TrajectoryWeight * 100
		indicators = append(indicators, "轨迹不平滑")
	}

	if result.Trajectory.DirectionChanges > 50 {
		riskScore += 10 * s.weights.TrajectoryWeight * 100
		indicators = append(indicators, "方向变化过多")
	}

	if result.ClickPattern.Regularity > 0.9 && result.ClickPattern.ClickCount > 2 {
		riskScore += 20 * s.weights.ClickWeight * 100
		indicators = append(indicators, "点击间隔过于规律")
	}

	if result.ClickPattern.ClickSpeed > 10 {
		riskScore += 25 * s.weights.ClickWeight * 100
		indicators = append(indicators, "点击速度异常快")
	}

	if len(result.Trajectory.Points) < 10 {
		riskScore += 15 * s.weights.TrajectoryWeight * 100
		indicators = append(indicators, "行为数据点过少")
	}

	if result.Trajectory.MinSpeed == math.MaxFloat64 {
		result.Trajectory.MinSpeed = 0
	}

	if result.Trajectory.MaxSpeed > 0 && result.Trajectory.MinSpeed > 0 {
		speedRatio := result.Trajectory.MaxSpeed / result.Trajectory.MinSpeed
		if speedRatio > 100 {
			riskScore += 10 * s.weights.SpeedWeight * 100
			indicators = append(indicators, "速度范围异常")
		}
	}

	if extendedData != nil {
		if extendedData.EnvironmentData.IsHeadless {
			riskScore += 30 * s.weights.EnvironmentWeight * 100
			indicators = append(indicators, "检测到无头浏览器")
		}

		if len(extendedData.EnvironmentData.Plugins) == 0 {
			riskScore += 5 * s.weights.EnvironmentWeight * 100
			indicators = append(indicators, "无浏览器插件")
		}
	}

	if result.MLPrediction != nil {
		mlWeight := 0.3
		riskScore = riskScore*(1-mlWeight) + result.MLPrediction.BotScore*mlWeight*100
		
		if result.MLPrediction.BotScore > 0.7 {
			indicators = append(indicators, fmt.Sprintf("ML模型判定为机器人 (置信度: %.2f)", result.MLPrediction.Confidence))
		}
	}

	result.RiskScore = math.Min(riskScore, 100)
	result.RiskIndicators = indicators
	result.IsBotLikely = riskScore >= 50
	result.Confidence = math.Min(riskScore/100+0.3, 0.95)
}

func (s *BehaviorAnalysisService) CalculateRiskScore(verification *models.Verification, behaviorData []models.BehaviorData) float64 {
	result, err := s.AnalyzeBehavior(behaviorData)
	if err != nil {
		return 50.0
	}
	return result.RiskScore
}

func (s *BehaviorAnalysisService) GenerateAnalysisReport(result *AnalysisResult) string {
	report := fmt.Sprintf("行为分析报告:\n")
	report += fmt.Sprintf("- 风险评分: %.2f\n", result.RiskScore)
	report += fmt.Sprintf("- 疑似机器人: %v\n", result.IsBotLikely)
	report += fmt.Sprintf("- 置信度: %.2f\n", result.Confidence)
	report += fmt.Sprintf("- 风险指标:\n")
	for _, indicator := range result.RiskIndicators {
		report += fmt.Sprintf("  * %s\n", indicator)
	}
	report += fmt.Sprintf("- 轨迹分析:\n")
	report += fmt.Sprintf("  * 总距离: %.2f\n", result.Trajectory.TotalDistance)
	report += fmt.Sprintf("  * 平均速度: %.2f\n", result.Trajectory.AverageSpeed)
	report += fmt.Sprintf("  * 最大速度: %.2f\n", result.Trajectory.MaxSpeed)
	report += fmt.Sprintf("  * 路径效率: %.2f\n", result.Trajectory.PathEfficiency)
	report += fmt.Sprintf("  * 平滑度: %.2f\n", result.Trajectory.Smoothness)
	report += fmt.Sprintf("  * 方向变化: %d\n", result.Trajectory.DirectionChanges)
	report += fmt.Sprintf("- 点击模式:\n")
	report += fmt.Sprintf("  * 点击次数: %d\n", result.ClickPattern.ClickCount)
	report += fmt.Sprintf("  * 平均间隔: %.2fms\n", result.ClickPattern.AverageInterval)
	report += fmt.Sprintf("  * 点击速度: %.2f点击/秒\n", result.ClickPattern.ClickSpeed)
	report += fmt.Sprintf("  * 规律性: %.2f\n", result.ClickPattern.Regularity)
	report += fmt.Sprintf("- 特征向量:\n")
	report += fmt.Sprintf("  * 速度变化: %.2f\n", result.Trajectory.SpeedVariance)
	report += fmt.Sprintf("  * 数据点密度: %.2f\n", result.Features.DataPointDensity)
	
	if result.MLPrediction != nil {
		report += fmt.Sprintf("- ML预测:\n")
		report += fmt.Sprintf("  * 机器人分数: %.2f\n", result.MLPrediction.BotScore)
		report += fmt.Sprintf("  * 人类分数: %.2f\n", result.MLPrediction.HumanScore)
		report += fmt.Sprintf("  * 模型版本: %s\n", result.MLPrediction.ModelVersion)
	}

	return report
}

func (s *BehaviorAnalysisService) VerifyWithBehaviorAnalysis(
	captchaSuccess bool,
	behaviorData []models.BehaviorData,
) (bool, float64, string) {
	result, _ := s.AnalyzeBehavior(behaviorData)

	analysisReport := s.GenerateAnalysisReport(result)

	var finalResult bool
	if result.RiskScore < 30 {
		finalResult = captchaSuccess
	} else if result.RiskScore < 70 {
		finalResult = captchaSuccess && result.RiskScore < 50
	} else {
		finalResult = false
	}

	return finalResult, result.RiskScore, analysisReport
}

func (s *BehaviorAnalysisService) AnalyzeKeyStroke(keyStrokes []KeyStrokeInfo) KeyStrokeAnalysis {
	analysis := KeyStrokeAnalysis{}

	if len(keyStrokes) == 0 {
		return analysis
	}

	analysis.TotalKeystrokes = len(keyStrokes)

	intervals := []float64{}
	totalHoldTime := int64(0)
	modifierCount := 0

	for i := 1; i < len(keyStrokes); i++ {
		interval := float64(keyStrokes[i].Timestamp - keyStrokes[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	for _, ks := range keyStrokes {
		totalHoldTime += ks.HoldTime
		if ks.IsModifier {
			modifierCount++
		}
	}

	if len(intervals) > 0 {
		avgInterval := 0.0
		for _, interval := range intervals {
			avgInterval += interval
		}
		analysis.AverageInterval = avgInterval / float64(len(intervals))
		analysis.IntervalVariance = s.calculateVariance(intervals, analysis.AverageInterval)
	}

	if len(keyStrokes) > 0 {
		analysis.AverageHoldTime = float64(totalHoldTime) / float64(len(keyStrokes))
	}

	if len(keyStrokes) > 0 {
		analysis.ModifierUsage = float64(modifierCount) / float64(len(keyStrokes))
	}

	analysis.TypingRhythm = 1.0 - math.Min(1.0, math.Sqrt(analysis.IntervalVariance)/math.Max(1.0, analysis.AverageInterval))
	analysis.IsTypingPattern = analysis.TypingRhythm > 0.3 && analysis.AverageHoldTime > 30

	return analysis
}

func (s *BehaviorAnalysisService) AnalyzeScroll(scrolls []ScrollInfo) ScrollAnalysis {
	analysis := ScrollAnalysis{}

	if len(scrolls) == 0 {
		return analysis
	}

	analysis.TotalScrolls = len(scrolls)

	velocities := []float64{}
	directions := make(map[string]bool)

	for _, scroll := range scrolls {
		velocities = append(velocities, scroll.Velocity)

		if scroll.DeltaY > 0 {
			directions["down"] = true
		} else if scroll.DeltaY < 0 {
			directions["up"] = true
		}
		if scroll.DeltaX > 0 {
			directions["right"] = true
		} else if scroll.DeltaX < 0 {
			directions["left"] = true
		}
	}

	if len(velocities) > 0 {
		totalVelocity := 0.0
		maxVelocity := 0.0
		for _, v := range velocities {
			totalVelocity += v
			if v > maxVelocity {
				maxVelocity = v
			}
		}
		analysis.AverageVelocity = totalVelocity / float64(len(velocities))
		analysis.MaxVelocity = maxVelocity
	}

	analysis.DirectionCount = len(directions)
	
	variance := s.calculateVariance(velocities, analysis.AverageVelocity)
	if analysis.AverageVelocity > 0 {
		analysis.ScrollPattern = 1.0 - (math.Sqrt(variance) / analysis.AverageVelocity)
	}

	return analysis
}

func (s *BehaviorAnalysisService) StreamAddPoint(point BehaviorDataPoint) {
	s.streamProcessor.buffer = append(s.streamProcessor.buffer, point)
	
	if len(s.streamProcessor.buffer) > s.streamProcessor.maxBufferSize {
		s.streamProcessor.buffer = s.streamProcessor.buffer[1:]
	}
}

func (s *BehaviorAnalysisService) StreamAnalyze() *AnalysisResult {
	if len(s.streamProcessor.buffer) < 10 {
		return nil
	}

	var points []BehaviorDataPoint
	copy(points, s.streamProcessor.buffer)

	result := &AnalysisResult{}
	result.Trajectory = s.analyzeMouseTrajectory(points)
	
	return result
}

func (s *BehaviorAnalysisService) StreamClear() {
	s.streamProcessor.buffer = make([]BehaviorDataPoint, 0)
}

func (s *BehaviorAnalysisService) GetCachedResult(key string) *AnalysisResult {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if entry, exists := s.cache.entries[key]; exists {
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Result
		}
		delete(s.cache.entries, key)
	}
	return nil
}

func (s *BehaviorAnalysisService) SetCachedResult(key string, result *AnalysisResult, ttl time.Duration) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if len(s.cache.entries) >= s.cache.maxSize {
		s.cleanupCache()
	}

	s.cache.entries[key] = &CachedResult{
		Result:    result,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (s *BehaviorAnalysisService) cleanupCache() {
	now := time.Now()
	keysToDelete := []string{}

	for key, entry := range s.cache.entries {
		if now.After(entry.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	if len(keysToDelete) == 0 && len(s.cache.entries) >= s.cache.maxSize {
		sortedKeys := make([]string, 0, len(s.cache.entries))
		for key := range s.cache.entries {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)
		if len(sortedKeys) > 0 {
			keysToDelete = append(keysToDelete, sortedKeys[0])
		}
	}

	for _, key := range keysToDelete {
		delete(s.cache.entries, key)
	}
}

func (s *BehaviorAnalysisService) UpdateWeights(weights RiskWeights) {
	s.weights = weights
}

func (s *BehaviorAnalysisService) SetModelService(modelService *ModelService) {
	s.modelService = modelService
}

func (s *BehaviorAnalysisService) AnalyzeExtendedData(data *ExtendedBehaviorData) (*AnalysisResult, error) {
	result := &AnalysisResult{
		RiskIndicators: []string{},
	}

	if data.MouseTrajectory != nil && len(data.MouseTrajectory) > 0 {
		points := make([]BehaviorDataPoint, len(data.MouseTrajectory))
		for i, mp := range data.MouseTrajectory {
			points[i] = BehaviorDataPoint{
				X:         int(mp.X),
				Y:         int(mp.Y),
				Timestamp: mp.Timestamp,
				Event:     "mousemove",
			}
		}
		result.Trajectory = s.analyzeMouseTrajectory(points)
	}

	if data.MouseSpeed != nil && len(data.MouseSpeed) > 0 {
		speeds := make([]float64, len(data.MouseSpeed))
		for i, sp := range data.MouseSpeed {
			speeds[i] = sp.Speed
		}
		if len(speeds) > 0 {
			sum := 0.0
			for _, s := range speeds {
				sum += s
			}
			result.Trajectory.AverageSpeed = sum / float64(len(speeds))
			
			maxSpeed := 0.0
			for _, s := range speeds {
				if s > maxSpeed {
					maxSpeed = s
				}
			}
			result.Trajectory.MaxSpeed = maxSpeed
		}
	}

	if data.ClickData != nil && len(data.ClickData) > 0 {
		result.ClickPattern.ClickCount = len(data.ClickData)
		clickPositions := make([]ClickPosition, len(data.ClickData))
		for i, cd := range data.ClickData {
			clickPositions[i] = ClickPosition{
				X:      cd.X,
				Y:      cd.Y,
				Button: cd.Button,
			}
		}
		result.ClickPattern.ClickPositions = clickPositions

		if len(data.ClickData) >= 2 {
			intervals := []float64{}
			for i := 1; i < len(data.ClickData); i++ {
				interval := float64(data.ClickData[i].Timestamp - data.ClickData[i-1].Timestamp)
				intervals = append(intervals, interval)
			}
			if len(intervals) > 0 {
				sum := 0.0
				for _, iv := range intervals {
					sum += iv
				}
				result.ClickPattern.AverageInterval = sum / float64(len(intervals))
			}
		}
	}

	s.extractFeatures(result, nil, data)

	keyStrokeAnalysis := s.AnalyzeKeyStroke(data.KeyStrokeData)
	scrollAnalysis := s.AnalyzeScroll(data.ScrollData)

	if keyStrokeAnalysis.TotalKeystrokes > 0 {
		if keyStrokeAnalysis.IsTypingPattern {
			result.RiskIndicators = append(result.RiskIndicators, "检测到键盘输入模式")
		}
	}

	if scrollAnalysis.TotalScrolls > 0 {
		if scrollAnalysis.ScrollPattern > 0.9 {
			result.RiskIndicators = append(result.RiskIndicators, "滚动行为过于规律")
		}
	}

	s.calculateRiskScore(result, data)

	return result, nil
}
