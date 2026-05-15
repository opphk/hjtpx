package service

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/hjtpx/hjtpx/pkg/models"
)

// BehaviorDataPoint 行为数据点
type BehaviorDataPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Event     string  `json:"event"`
}

// MouseTrajectory 鼠标轨迹分析
type MouseTrajectory struct {
	Points         []BehaviorDataPoint `json:"points"`
	TotalDistance  float64             `json:"total_distance"`
	AverageSpeed   float64             `json:"average_speed"`
	MaxSpeed       float64             `json:"max_speed"`
	PathEfficiency float64             `json:"path_efficiency"`
	DirectionChanges int              `json:"direction_changes"`
}

// ClickPattern 点击模式分析
type ClickPattern struct {
	Clicks         []BehaviorDataPoint `json:"clicks"`
	ClickCount     int                 `json:"click_count"`
	AverageInterval float64            `json:"average_interval"`
	ClickSpeed     float64             `json:"click_speed"`
	Regularity     float64             `json:"regularity"`
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Trajectory     MouseTrajectory `json:"trajectory"`
	ClickPattern   ClickPattern    `json:"click_pattern"`
	RiskScore      float64         `json:"risk_score"`
	RiskIndicators []string        `json:"risk_indicators"`
	IsBotLikely    bool            `json:"is_bot_likely"`
	Confidence     float64         `json:"confidence"`
}

// BehaviorAnalysisService 行为分析服务
type BehaviorAnalysisService struct{}

func NewBehaviorAnalysisService() *BehaviorAnalysisService {
	return &BehaviorAnalysisService{}
}

// AnalyzeBehavior 分析行为数据
func (s *BehaviorAnalysisService) AnalyzeBehavior(behaviorData []models.BehaviorData) (*AnalysisResult, error) {
	result := &AnalysisResult{
		RiskIndicators: []string{},
	}

	var points []BehaviorDataPoint
	var clicks []BehaviorDataPoint

	for _, bd := range behaviorData {
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

	s.calculateRiskScore(result)

	return result, nil
}

// analyzeMouseTrajectory 分析鼠标轨迹
func (s *BehaviorAnalysisService) analyzeMouseTrajectory(points []BehaviorDataPoint) MouseTrajectory {
	traj := MouseTrajectory{
		Points: points,
	}

	if len(points) < 2 {
		return traj
	}

	totalDistance := 0.0
	maxSpeed := 0.0
	speeds := []float64{}
	directionChanges := 0
	prevAngle := 0.0

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance

		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speed := distance / dt
			speeds = append(speeds, speed)
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}

		if i > 1 {
			angle := math.Atan2(dy, dx)
			if math.Abs(angle-prevAngle) > 0.5 {
				directionChanges++
			}
			prevAngle = angle
		}
	}

	traj.TotalDistance = totalDistance
	traj.MaxSpeed = maxSpeed
	traj.DirectionChanges = directionChanges

	if len(speeds) > 0 {
		avgSpeed := 0.0
		for _, speed := range speeds {
			avgSpeed += speed
		}
		traj.AverageSpeed = avgSpeed / float64(len(speeds))
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

	return traj
}

// analyzeClickPattern 分析点击模式
func (s *BehaviorAnalysisService) analyzeClickPattern(clicks []BehaviorDataPoint) ClickPattern {
	pattern := ClickPattern{
		Clicks:     clicks,
		ClickCount: len(clicks),
	}

	if len(clicks) < 2 {
		return pattern
	}

	intervals := []float64{}
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	if len(intervals) > 0 {
		avgInterval := 0.0
		for _, interval := range intervals {
			avgInterval += interval
		}
		avgInterval = avgInterval / float64(len(intervals))
		pattern.AverageInterval = avgInterval

		variance := 0.0
		for _, interval := range intervals {
			variance += math.Pow(interval-avgInterval, 2)
		}
		variance = variance / float64(len(intervals))
		stdDev := math.Sqrt(variance)

		if avgInterval > 0 {
			pattern.Regularity = 1 - (stdDev / avgInterval)
		}
	}

	totalTime := float64(clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp)
	if totalTime > 0 {
		pattern.ClickSpeed = float64(len(clicks)) / (totalTime / 1000)
	}

	return pattern
}

// calculateRiskScore 计算风险评分
func (s *BehaviorAnalysisService) calculateRiskScore(result *AnalysisResult) {
	riskScore := 0.0
	indicators := []string{}

	if result.Trajectory.AverageSpeed > 0 {
		if result.Trajectory.AverageSpeed > 5 {
			riskScore += 20
			indicators = append(indicators, "异常高速移动")
		}
		if result.Trajectory.AverageSpeed < 0.1 {
			riskScore += 15
			indicators = append(indicators, "异常低速移动")
		}
	}

	if result.Trajectory.PathEfficiency > 0.95 && result.Trajectory.TotalDistance > 100 {
		riskScore += 25
		indicators = append(indicators, "路径过于笔直")
	}

	if result.ClickPattern.Regularity > 0.9 && result.ClickPattern.ClickCount > 2 {
		riskScore += 20
		indicators = append(indicators, "点击间隔过于规律")
	}

	if result.ClickPattern.ClickSpeed > 10 {
		riskScore += 25
		indicators = append(indicators, "点击速度异常快")
	}

	if len(result.Trajectory.Points) < 10 {
		riskScore += 15
		indicators = append(indicators, "行为数据点过少")
	}

	result.RiskScore = math.Min(riskScore, 100)
	result.RiskIndicators = indicators
	result.IsBotLikely = riskScore >= 50
	result.Confidence = math.Min(riskScore/100+0.3, 0.95)
}

// CalculateRiskScore 计算整体风险评分
func (s *BehaviorAnalysisService) CalculateRiskScore(verification *models.Verification, behaviorData []models.BehaviorData) float64 {
	result, err := s.AnalyzeBehavior(behaviorData)
	if err != nil {
		return 50.0
	}
	return result.RiskScore
}

// GenerateAnalysisReport 生成分析报告
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
	report += fmt.Sprintf("  * 方向变化: %d\n", result.Trajectory.DirectionChanges)
	report += fmt.Sprintf("- 点击模式:\n")
	report += fmt.Sprintf("  * 点击次数: %d\n", result.ClickPattern.ClickCount)
	report += fmt.Sprintf("  * 平均间隔: %.2fms\n", result.ClickPattern.AverageInterval)
	report += fmt.Sprintf("  * 点击速度: %.2f点击/秒\n", result.ClickPattern.ClickSpeed)
	report += fmt.Sprintf("  * 规律性: %.2f\n", result.ClickPattern.Regularity)

	return report
}

// VerifyWithBehaviorAnalysis 结合行为分析进行验证
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
