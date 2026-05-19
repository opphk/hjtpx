package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type BehaviorAnalyticsService struct{}

func NewBehaviorAnalyticsService() *BehaviorAnalyticsService {
	return &BehaviorAnalyticsService{}
}

type HeatmapPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Count     int     `json:"count"`
	Intensity float64 `json:"intensity"`
}

type Trajectory struct {
	UserID    string              `json:"userId"`
	SessionID string              `json:"sessionId"`
	Points    []BehaviorDataPoint `json:"points"`
	StartTime string              `json:"startTime"`
	EndTime   string              `json:"endTime"`
	Duration  int64               `json:"duration"`
}

type BehaviorRiskDistribution struct {
	Range      string  `json:"range"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type BehaviorSummary struct {
	TotalSessions      int64   `json:"totalSessions"`
	TotalInteractions  int64   `json:"totalInteractions"`
	AvgSessionDuration float64 `json:"avgSessionDuration"`
	AvgMouseSpeed      float64 `json:"avgMouseSpeed"`
	ClickCount         int64   `json:"clickCount"`
	KeyboardEventCount int64   `json:"keyboardEventCount"`
	AnomalyCount       int64   `json:"anomalyCount"`
	HighRiskUsers      int64   `json:"highRiskUsers"`
}

func (s *BehaviorAnalyticsService) GetBehaviorSummary(period string) (*BehaviorSummary, error) {
	summary := &BehaviorSummary{
		TotalSessions:      1247,
		TotalInteractions:  89432,
		AvgSessionDuration: 187.5,
		AvgMouseSpeed:      4.2,
		ClickCount:         34218,
		KeyboardEventCount: 55214,
		AnomalyCount:       89,
		HighRiskUsers:      23,
	}

	return summary, nil
}

func (s *BehaviorAnalyticsService) GetHeatmapData(period string) ([]HeatmapPoint, error) {
	heatmap := make([]HeatmapPoint, 0)

	gridSize := 20
	for x := 0; x < 1920; x += gridSize {
		for y := 0; y < 1080; y += gridSize {
			if rand.Float64() > 0.85 {
				count := rand.Intn(50) + 1
				intensity := float64(count) / 50.0
				heatmap = append(heatmap, HeatmapPoint{
					X:         x,
					Y:         y,
					Count:     count,
					Intensity: intensity,
				})
			}
		}
	}

	return heatmap, nil
}

func (s *BehaviorAnalyticsService) GetRecentTrajectories(limit int) ([]Trajectory, error) {
	trajectories := make([]Trajectory, 0, limit)

	userIDs := []string{"user_1", "user_2", "user_3", "user_4", "user_5", "user_6", "user_7", "user_8"}
	sessionIDs := []string{"session_001", "session_002", "session_003", "session_004", "session_005"}

	for i := 0; i < limit; i++ {
		pointCount := rand.Intn(50) + 10
		points := make([]BehaviorDataPoint, 0, pointCount)

		startX, startY := rand.Intn(1600)+100, rand.Intn(800)+100
		currentX, currentY := startX, startY
		startTime := time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second).UnixMilli()

		for j := 0; j < pointCount; j++ {
			event := "move"
			if rand.Float64() > 0.9 {
				event = "click"
			}

			currentX += rand.Intn(100) - 50
			currentY += rand.Intn(100) - 50
			currentX = maxInt(0, minInt(currentX, 1920))
			currentY = maxInt(0, minInt(currentY, 1080))

			points = append(points, BehaviorDataPoint{
				X:         currentX,
				Y:         currentY,
				Timestamp: startTime + int64(j*100),
				Event:     event,
			})
		}

		duration := int64(pointCount * 100)
		trajectories = append(trajectories, Trajectory{
			UserID:    userIDs[i%len(userIDs)],
			SessionID: sessionIDs[i%len(sessionIDs)],
			Points:    points,
			StartTime: time.UnixMilli(startTime).Format(time.RFC3339),
			EndTime:   time.UnixMilli(startTime + duration).Format(time.RFC3339),
			Duration:  duration,
		})
	}

	return trajectories, nil
}

type BehaviorAnomaly struct {
	PatternID     string
	Type          string
	Description   string
	Severity      string
	Count         int
	AffectedUsers int
	FirstSeen     string
	LastSeen      string
}

func (s *BehaviorAnalyticsService) GetAnomalyPatterns(period string) ([]BehaviorAnomaly, error) {
	anomalies := []BehaviorAnomaly{
		{
			PatternID:     "ANOM_001",
			Type:          "linear_mouse_movement",
			Description:   "检测到高度线性的鼠标移动路径，可能是自动化脚本",
			Severity:      "high",
			Count:         42,
			AffectedUsers: 18,
			FirstSeen:     time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
			LastSeen:      time.Now().Format(time.RFC3339),
		},
		{
			PatternID:     "ANOM_002",
			Type:          "rapid_clicks",
			Description:   "异常快速的点击模式，超出人类正常范围",
			Severity:      "critical",
			Count:         17,
			AffectedUsers: 8,
			FirstSeen:     time.Now().Add(-14 * 24 * time.Hour).Format(time.RFC3339),
			LastSeen:      time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			PatternID:     "ANOM_003",
			Type:          "repeated_trajectory",
			Description:   "高度相似的重复轨迹模式",
			Severity:      "medium",
			Count:         78,
			AffectedUsers: 32,
			FirstSeen:     time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			LastSeen:      time.Now().Format(time.RFC3339),
		},
		{
			PatternID:     "ANOM_004",
			Type:          "unusual_timing",
			Description:   "异常的操作时序模式，表现出机器人特征",
			Severity:      "high",
			Count:         23,
			AffectedUsers: 11,
			FirstSeen:     time.Now().Add(-5 * 24 * time.Hour).Format(time.RFC3339),
			LastSeen:      time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			PatternID:     "ANOM_005",
			Type:          "no_hesitation",
			Description:   "点击前无犹豫，直接定位到目标位置",
			Severity:      "medium",
			Count:         56,
			AffectedUsers: 25,
			FirstSeen:     time.Now().Add(-20 * 24 * time.Hour).Format(time.RFC3339),
			LastSeen:      time.Now().Format(time.RFC3339),
		},
	}

	return anomalies, nil
}

func (s *BehaviorAnalyticsService) GetRiskDistribution(period string) ([]BehaviorRiskDistribution, error) {
	distribution := []BehaviorRiskDistribution{
		{
			Range:      "0-20",
			Count:      892,
			Percentage: 71.5,
		},
		{
			Range:      "21-40",
			Count:      234,
			Percentage: 18.8,
		},
		{
			Range:      "41-60",
			Count:      78,
			Percentage: 6.3,
		},
		{
			Range:      "61-80",
			Count:      32,
			Percentage: 2.6,
		},
		{
			Range:      "81-100",
			Count:      11,
			Percentage: 0.9,
		},
	}

	return distribution, nil
}

func (s *BehaviorAnalyticsService) ExportBehaviorData(format, period string) ([]byte, error) {
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = s.exportJSON(period)
	case "csv":
		data, err = s.exportCSV(period)
	default:
		data, err = s.exportCSV(period)
	}

	return data, err
}

func (s *BehaviorAnalyticsService) exportJSON(period string) ([]byte, error) {
	summary, _ := s.GetBehaviorSummary(period)
	heatmap, _ := s.GetHeatmapData(period)
	trajectories, _ := s.GetRecentTrajectories(10)
	anomalies, _ := s.GetAnomalyPatterns(period)
	riskDistribution, _ := s.GetRiskDistribution(period)

	exportData := map[string]interface{}{
		"exportTime":       time.Now().Format(time.RFC3339),
		"period":           period,
		"summary":          summary,
		"heatmap":          heatmap,
		"trajectories":     trajectories,
		"anomalies":        anomalies,
		"riskDistribution": riskDistribution,
	}

	return json.MarshalIndent(exportData, "", "  ")
}

func (s *BehaviorAnalyticsService) exportCSV(period string) ([]byte, error) {
	summary, _ := s.GetBehaviorSummary(period)
	csvContent := "用户行为分析报表\n"
	csvContent += "导出时间," + time.Now().Format("2006-01-02 15:04:05") + "\n"
	csvContent += "时间范围," + period + "\n\n"

	csvContent += "指标,数值\n"
	csvContent += fmt.Sprintf("总会话数,%d\n", summary.TotalSessions)
	csvContent += fmt.Sprintf("总交互数,%d\n", summary.TotalInteractions)
	csvContent += fmt.Sprintf("平均会话时长(秒),%.2f\n", summary.AvgSessionDuration)
	csvContent += fmt.Sprintf("平均鼠标速度,%.2f\n", summary.AvgMouseSpeed)
	csvContent += fmt.Sprintf("点击数,%d\n", summary.ClickCount)
	csvContent += fmt.Sprintf("键盘事件数,%d\n", summary.KeyboardEventCount)
	csvContent += fmt.Sprintf("异常数,%d\n", summary.AnomalyCount)
	csvContent += fmt.Sprintf("高风险用户数,%d\n", summary.HighRiskUsers)

	return []byte(csvContent), nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type SankeyNode struct {
	Name string `json:"name"`
}

type SankeyLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Value  int    `json:"value"`
}

type SankeyData struct {
	Nodes []SankeyNode `json:"nodes"`
	Links []SankeyLink `json:"links"`
}

type RadarIndicator struct {
	Name string `json:"name"`
	Max  int    `json:"max"`
}

type RadarSeriesData struct {
	Value     []int              `json:"value"`
	Name      string             `json:"name"`
	ItemStyle map[string]string  `json:"itemStyle,omitempty"`
	AreaStyle map[string]float64 `json:"areaStyle,omitempty"`
}

type RadarData struct {
	Indicator []RadarIndicator  `json:"indicator"`
	Data      []RadarSeriesData `json:"data"`
}

func (s *BehaviorAnalyticsService) GetSankeyData(period string) (*SankeyData, error) {
	data := &SankeyData{
		Nodes: []SankeyNode{
			{Name: "进入页面"},
			{Name: "浏览内容"},
			{Name: "点击验证码"},
			{Name: "验证成功"},
			{Name: "验证失败"},
			{Name: "重新验证"},
			{Name: "离开页面"},
		},
		Links: []SankeyLink{
			{Source: "进入页面", Target: "浏览内容", Value: 1000},
			{Source: "浏览内容", Target: "点击验证码", Value: 800},
			{Source: "点击验证码", Target: "验证成功", Value: 600},
			{Source: "点击验证码", Target: "验证失败", Value: 200},
			{Source: "验证失败", Target: "重新验证", Value: 150},
			{Source: "验证失败", Target: "离开页面", Value: 50},
			{Source: "重新验证", Target: "验证成功", Value: 120},
			{Source: "重新验证", Target: "离开页面", Value: 30},
			{Source: "验证成功", Target: "离开页面", Value: 720},
			{Source: "浏览内容", Target: "离开页面", Value: 200},
		},
	}
	return data, nil
}

func (s *BehaviorAnalyticsService) GetRadarData(period string) (*RadarData, error) {
	data := &RadarData{
		Indicator: []RadarIndicator{
			{Name: "鼠标移动速度", Max: 100},
			{Name: "点击间隔", Max: 100},
			{Name: "轨迹直线度", Max: 100},
			{Name: "操作频率", Max: 100},
			{Name: "响应时间", Max: 100},
			{Name: "一致性", Max: 100},
		},
		Data: []RadarSeriesData{
			{
				Value:     []int{20, 15, 25, 30, 20, 25},
				Name:      "正常用户",
				ItemStyle: map[string]string{"color": "#28a745"},
				AreaStyle: map[string]float64{"opacity": 0.3},
			},
			{
				Value:     []int{75, 80, 85, 70, 80, 90},
				Name:      "高风险用户",
				ItemStyle: map[string]string{"color": "#dc3545"},
				AreaStyle: map[string]float64{"opacity": 0.3},
			},
		},
	}
	return data, nil
}
