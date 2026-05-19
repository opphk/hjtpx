package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type DashboardService struct {
	captchaRepo interface {
		CountToday() (int64, error)
		GetPassRate() (float64, error)
	}
	riskRepo interface {
		GetBlockRate() (float64, error)
	}
}

type DashboardData struct {
	Summary              *Summary                  `json:"summary"`
	Trend                []TrendData               `json:"trend"`
	RiskDistribution     *RiskDistributionData     `json:"risk_distribution"`
	CaptchaType          []CaptchaTypeData         `json:"captcha_type"`
	AttackTypeDistribution []AttackTypeData        `json:"attack_type_distribution"`
	RiskScoreDistribution  []RiskScoreBinData      `json:"risk_score_distribution"`
}

type Summary struct {
	TotalRequests   int64   `json:"total_requests"`
	PassRate        float64 `json:"pass_rate"`
	BlockRate       float64 `json:"block_rate"`
	AvgResponseTime int64   `json:"avg_response_time"`
	ActiveSessions  int     `json:"active_sessions"`
}

type TrendData struct {
	Time     string `json:"time"`
	Requests int64  `json:"requests"`
	Success  int64  `json:"success"`
	Failed   int64  `json:"failed"`
}

type RiskDistributionData struct {
	Low      int64 `json:"low"`
	Medium   int64 `json:"medium"`
	High     int64 `json:"high"`
	Critical int64 `json:"critical"`
}

type CaptchaTypeData struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

type AttackTypeData struct {
	AttackType string `json:"attack_type"`
	Count      int64  `json:"count"`
	Percentage float64 `json:"percentage"`
}

type RiskScoreBinData struct {
	BinStart int64   `json:"bin_start"`
	BinEnd   int64   `json:"bin_end"`
	Count    int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type RealTimeVerificationEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	SessionID   string    `json:"session_id"`
	CaptchaType string    `json:"captcha_type"`
	Status      string    `json:"status"`
	RiskScore   float64   `json:"risk_score"`
	IPAddress   string    `json:"ip_address"`
}

var realTimeEvents = make(chan RealTimeVerificationEvent, 1000)

func PublishVerificationEvent(event RealTimeVerificationEvent) {
	select {
	case realTimeEvents <- event:
	default:
	}
}

func SubscribeToVerificationEvents() <-chan RealTimeVerificationEvent {
	return realTimeEvents
}

func NewDashboardService() *DashboardService {
	return &DashboardService{}
}

func (s *DashboardService) GetDashboardData(period string) (*DashboardData, error) {
	data := &DashboardData{}

	summary, err := s.getSummary()
	if err != nil {
		return nil, err
	}
	data.Summary = summary

	trend, err := s.getTrendData(period)
	if err != nil {
		return nil, err
	}
	data.Trend = trend

	distribution, err := s.getRiskDistribution()
	if err != nil {
		return nil, err
	}
	data.RiskDistribution = distribution

	captchaType, err := s.getCaptchaTypeStats()
	if err != nil {
		return nil, err
	}
	data.CaptchaType = captchaType

	attackDistribution, err := s.getAttackTypeDistribution()
	if err != nil {
		return nil, err
	}
	data.AttackTypeDistribution = attackDistribution

	riskScoreDistribution, err := s.getRiskScoreDistribution()
	if err != nil {
		return nil, err
	}
	data.RiskScoreDistribution = riskScoreDistribution

	return data, nil
}

func (s *DashboardService) getSummary() (*Summary, error) {
	summary := &Summary{}

	if database.DB == nil {
		summary.TotalRequests = 1000
		summary.PassRate = 95.5
		summary.BlockRate = 2.3
		summary.AvgResponseTime = 120
		summary.ActiveSessions = 0
		return summary, nil
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var totalCount int64
	database.DB.Model(&models.Verification{}).
		Where("created_at >= ?", startOfDay).
		Count(&totalCount)
	summary.TotalRequests = totalCount

	var successCount int64
	database.DB.Model(&models.Verification{}).
		Where("status = ? AND created_at >= ?", "success", startOfDay).
		Count(&successCount)

	if totalCount > 0 {
		summary.PassRate = float64(successCount) / float64(totalCount) * 100
	} else {
		summary.PassRate = 0
	}

	var blockCount int64
	database.DB.Model(&models.Verification{}).
		Where("status = ? AND created_at >= ?", "blocked", startOfDay).
		Count(&blockCount)

	if totalCount > 0 {
		summary.BlockRate = float64(blockCount) / float64(totalCount) * 100
	} else {
		summary.BlockRate = 0
	}

	rows, _ := database.DB.Model(&models.Verification{}).
		Select("COALESCE(AVG(duration), 0) as avg_duration").
		Where("created_at >= ?", startOfDay).
		Rows()
	if rows.Next() {
		var avgDuration float64
		rows.Scan(&avgDuration)
		summary.AvgResponseTime = int64(avgDuration)
	}

	summary.ActiveSessions = GetWebSocketService().GetSessionCount()

	return summary, nil
}

func (s *DashboardService) getTrendData(period string) ([]TrendData, error) {
	now := time.Now()
	var data []TrendData

	if database.DB == nil {
		for i := 5; i >= 0; i-- {
			hour := now.Add(-time.Duration(i) * time.Hour)
			data = append(data, TrendData{
				Time:     fmt.Sprintf("%02d:00", hour.Hour()),
				Requests: int64(50 + i*10),
				Success:  int64(45 + i*8),
				Failed:   int64(5 + i*2),
			})
		}
		return data, nil
	}

	switch period {
	case "hour":
		for i := 23; i >= 0; i-- {
			hour := now.Add(-time.Duration(i) * time.Hour)
			startHour := time.Date(hour.Year(), hour.Month(), hour.Day(), hour.Hour(), 0, 0, 0, hour.Location())
			endHour := startHour.Add(time.Hour)

			var count int64
			database.DB.Model(&models.Verification{}).
				Where("created_at >= ? AND created_at < ?", startHour, endHour).
				Count(&count)

			var successCount int64
			database.DB.Model(&models.Verification{}).
				Where("status = ? AND created_at >= ? AND created_at < ?", "success", startHour, endHour).
				Count(&successCount)

			data = append(data, TrendData{
				Time:     fmt.Sprintf("%02d:00", hour.Hour()),
				Requests: count,
				Success:  successCount,
				Failed:   count - successCount,
			})
		}
	case "day":
		for i := 6; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			startDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
			endDay := startDay.Add(24 * time.Hour)

			var count int64
			database.DB.Model(&models.Verification{}).
				Where("created_at >= ? AND created_at < ?", startDay, endDay).
				Count(&count)

			var successCount int64
			database.DB.Model(&models.Verification{}).
				Where("status = ? AND created_at >= ? AND created_at < ?", "success", startDay, endDay).
				Count(&successCount)

			data = append(data, TrendData{
				Time:     day.Format("01-02"),
				Requests: count,
				Success:  successCount,
				Failed:   count - successCount,
			})
		}
	case "week":
		for i := 6; i >= 0; i-- {
			day := now.AddDate(0, 0, -i*7)
			startDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
			endDay := startDay.AddDate(0, 0, 7)

			var count int64
			database.DB.Model(&models.Verification{}).
				Where("created_at >= ? AND created_at < ?", startDay, endDay).
				Count(&count)

			var successCount int64
			database.DB.Model(&models.Verification{}).
				Where("status = ? AND created_at >= ? AND created_at < ?", "success", startDay, endDay).
				Count(&successCount)

			data = append(data, TrendData{
				Time:     fmt.Sprintf("第%d周", 7-i),
				Requests: count,
				Success:  successCount,
				Failed:   count - successCount,
			})
		}
	case "month":
		for i := 29; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			startDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
			endDay := startDay.Add(24 * time.Hour)

			var count int64
			database.DB.Model(&models.Verification{}).
				Where("created_at >= ? AND created_at < ?", startDay, endDay).
				Count(&count)

			var successCount int64
			database.DB.Model(&models.Verification{}).
				Where("status = ? AND created_at >= ? AND created_at < ?", "success", startDay, endDay).
				Count(&successCount)

			data = append(data, TrendData{
				Time:     day.Format("01-02"),
				Requests: count,
				Success:  successCount,
				Failed:   count - successCount,
			})
		}
	default:
		return nil, errors.New("unsupported period")
	}

	return data, nil
}

func (s *DashboardService) getRiskDistribution() (*RiskDistributionData, error) {
	distribution := &RiskDistributionData{}

	if database.DB == nil {
		distribution.Low = 750
		distribution.Medium = 150
		distribution.High = 70
		distribution.Critical = 30
		return distribution, nil
	}

	var totalCount int64
	database.DB.Model(&models.Verification{}).Count(&totalCount)

	if totalCount == 0 {
		return distribution, nil
	}

	var lowCount, mediumCount, highCount, criticalCount int64

	database.DB.Model(&models.Verification{}).
		Where("risk_score >= 0 AND risk_score < 30").
		Count(&lowCount)

	database.DB.Model(&models.Verification{}).
		Where("risk_score >= 30 AND risk_score < 60").
		Count(&mediumCount)

	database.DB.Model(&models.Verification{}).
		Where("risk_score >= 60 AND risk_score < 80").
		Count(&highCount)

	database.DB.Model(&models.Verification{}).
		Where("risk_score >= 80 AND risk_score <= 100").
		Count(&criticalCount)

	distribution.Low = lowCount
	distribution.Medium = mediumCount
	distribution.High = highCount
	distribution.Critical = criticalCount

	return distribution, nil
}

func (s *DashboardService) getCaptchaTypeStats() ([]CaptchaTypeData, error) {
	var results []CaptchaTypeData

	if database.DB == nil {
		return []CaptchaTypeData{
			{"滑动验证", 500},
			{"点选验证", 300},
			{"图片验证", 150},
			{"文字验证", 40},
			{"手势验证", 10},
		}, nil
	}

	rows, err := database.DB.Model(&models.Verification{}).
		Select("captcha_type, COUNT(*) as count").
		Group("captcha_type").
		Order("count DESC").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	typeMap := map[string]string{
		"slider":  "滑动验证",
		"click":   "点选验证",
		"image":   "图片验证",
		"text":    "文字验证",
		"gesture": "手势验证",
	}

	for rows.Next() {
		var captchaType, typeName string
		var count int64

		if err := rows.Scan(&captchaType, &count); err != nil {
			continue
		}

		if mapped, ok := typeMap[captchaType]; ok {
			typeName = mapped
		} else {
			typeName = captchaType
		}

		results = append(results, CaptchaTypeData{
			Type:  typeName,
			Count: count,
		})
	}

	return results, nil
}

func (s *DashboardService) getAttackTypeDistribution() ([]AttackTypeData, error) {
	var results []AttackTypeData

	if database.DB == nil {
		return []AttackTypeData{
			{"暴力破解", 50, 5.0},
			{"自动化攻击", 100, 10.0},
			{"异常行为", 150, 15.0},
			{"正常请求", 650, 65.0},
			{"代理攻击", 50, 5.0},
		}, nil
	}

	attackTypes := []struct {
		Name string
		Expr string
	}{
		{"暴力破解", "risk_score >= 80 AND status = 'failed'"},
		{"自动化攻击", "risk_score >= 60 AND risk_score < 80"},
		{"异常行为", "risk_score >= 30 AND risk_score < 60"},
		{"正常请求", "risk_score < 30 AND status = 'success'"},
		{"代理攻击", "risk_score >= 70"},
	}

	var totalCount int64
	database.DB.Model(&models.Verification{}).Count(&totalCount)

	if totalCount == 0 {
		totalCount = 1
	}

	for _, at := range attackTypes {
		var count int64
		database.DB.Model(&models.Verification{}).
			Where(at.Expr).
			Count(&count)

		results = append(results, AttackTypeData{
			AttackType: at.Name,
			Count:      count,
			Percentage: float64(count) / float64(totalCount) * 100,
		})
	}

	return results, nil
}

func (s *DashboardService) GetAttackTypeDistribution() ([]AttackTypeData, error) {
	return s.getAttackTypeDistribution()
}

func (s *DashboardService) GetRiskScoreDistribution() ([]RiskScoreBinData, error) {
	return s.getRiskScoreDistribution()
}

func (s *DashboardService) getRiskScoreDistribution() ([]RiskScoreBinData, error) {
	var results []RiskScoreBinData

	if database.DB == nil {
		return []RiskScoreBinData{
			{0, 9, 200, 20.0},
			{10, 19, 150, 15.0},
			{20, 29, 150, 15.0},
			{30, 39, 100, 10.0},
			{40, 49, 100, 10.0},
			{50, 59, 80, 8.0},
			{60, 69, 70, 7.0},
			{70, 79, 50, 5.0},
			{80, 89, 40, 4.0},
			{90, 100, 60, 6.0},
		}, nil
	}

	bins := []struct {
		Start int64
		End   int64
	}{
		{0, 10},
		{10, 20},
		{20, 30},
		{30, 40},
		{40, 50},
		{50, 60},
		{60, 70},
		{70, 80},
		{80, 90},
		{90, 101},
	}

	var totalCount int64
	database.DB.Model(&models.Verification{}).Count(&totalCount)

	if totalCount == 0 {
		totalCount = 1
	}

	for _, bin := range bins {
		var count int64
		database.DB.Model(&models.Verification{}).
			Where("risk_score >= ? AND risk_score < ?", bin.Start, bin.End).
			Count(&count)

		results = append(results, RiskScoreBinData{
			BinStart:   bin.Start,
			BinEnd:     bin.End - 1,
			Count:      count,
			Percentage: float64(count) / float64(totalCount) * 100,
		})
	}

	return results, nil
}

func (s *DashboardService) ExportData(format, period string) ([]byte, error) {
	data, err := s.GetDashboardData(period)
	if err != nil {
		return nil, err
	}

	switch format {
	case "csv":
		return s.exportToCSV(data)
	case "json":
		return json.Marshal(data)
	case "excel":
		return s.exportToExcel(data)
	default:
		return nil, errors.New("unsupported export format")
	}
}

func (s *DashboardService) exportToCSV(data *DashboardData) ([]byte, error) {
	var csv string

	csv += "指标,数值\n"
	if data.Summary != nil {
		csv += fmt.Sprintf("今日验证,%d\n", data.Summary.TotalRequests)
		csv += fmt.Sprintf("通过率,%.2f%%\n", data.Summary.PassRate)
		csv += fmt.Sprintf("拦截率,%.2f%%\n", data.Summary.BlockRate)
		csv += fmt.Sprintf("平均响应,%dms\n", data.Summary.AvgResponseTime)
		csv += fmt.Sprintf("活跃会话,%d\n", data.Summary.ActiveSessions)
	}

	csv += "\n时间,请求数,成功数,失败数\n"
	if data.Trend != nil {
		for _, t := range data.Trend {
			csv += fmt.Sprintf("%s,%d,%d,%d\n", t.Time, t.Requests, t.Success, t.Failed)
		}
	}

	csv += "\n风险等级,数量\n"
	if data.RiskDistribution != nil {
		csv += fmt.Sprintf("低风险,%d\n", data.RiskDistribution.Low)
		csv += fmt.Sprintf("中风险,%d\n", data.RiskDistribution.Medium)
		csv += fmt.Sprintf("高风险,%d\n", data.RiskDistribution.High)
		csv += fmt.Sprintf("极高风险,%d\n", data.RiskDistribution.Critical)
	}

	csv += "\n验证码类型,使用次数\n"
	if data.CaptchaType != nil {
		for _, ct := range data.CaptchaType {
			csv += fmt.Sprintf("%s,%d\n", ct.Type, ct.Count)
		}
	}

	csv += "\n攻击类型,数量,百分比\n"
	if data.AttackTypeDistribution != nil {
		for _, at := range data.AttackTypeDistribution {
			csv += fmt.Sprintf("%s,%d,%.2f%%\n", at.AttackType, at.Count, at.Percentage)
		}
	}

	csv += "\n风险评分区间,数量,百分比\n"
	if data.RiskScoreDistribution != nil {
		for _, rs := range data.RiskScoreDistribution {
			csv += fmt.Sprintf("%d-%d,%d,%.2f%%\n", rs.BinStart, rs.BinEnd, rs.Count, rs.Percentage)
		}
	}

	return []byte(csv), nil
}

func (s *DashboardService) exportToExcel(data *DashboardData) ([]byte, error) {
	return json.Marshal(data)
}

type DashboardAlert struct {
	Type      string `json:"type"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func (s *DashboardService) CheckAlerts() []DashboardAlert {
	var alerts []DashboardAlert

	summary, err := s.getSummary()
	if err == nil && summary != nil {
		if summary.BlockRate > 20 {
			alerts = append(alerts, DashboardAlert{
				Type:      "high_block_rate",
				Level:     "warning",
				Message:   fmt.Sprintf("拦截率异常: %.2f%%", summary.BlockRate),
				Timestamp: time.Now().Unix(),
			})
		}

		if summary.AvgResponseTime > 500 {
			alerts = append(alerts, DashboardAlert{
				Type:      "slow_response",
				Level:     "info",
				Message:   fmt.Sprintf("响应时间过长: %dms", summary.AvgResponseTime),
				Timestamp: time.Now().Unix(),
			})
		}
	}

	return alerts
}

type ExtendedStats struct {
	TotalUsers   int64   `json:"total_users"`
	TotalApps    int64   `json:"total_apps"`
	CurrentQPS   float64 `json:"current_qps"`
	ErrorRate    float64 `json:"error_rate"`
	UserGrowth   float64 `json:"user_growth"`
	AppGrowth    float64 `json:"app_growth"`
	ErrorGrowth  float64 `json:"error_growth"`
}

func (s *DashboardService) GetExtendedStats() (*ExtendedStats, error) {
	stats := &ExtendedStats{}

	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	database.DB.Model(&models.Application{}).Count(&stats.TotalApps)

	now := time.Now()
	startOfMinute := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	var countInMinute int64
	database.DB.Model(&models.Verification{}).
		Where("created_at >= ?", startOfMinute).
		Count(&countInMinute)
	stats.CurrentQPS = float64(countInMinute)

	var totalCount, errorCount int64
	database.DB.Model(&models.Verification{}).Count(&totalCount)
	database.DB.Model(&models.Verification{}).
		Where("status = ?", "failed").
		Count(&errorCount)
	if totalCount > 0 {
		stats.ErrorRate = float64(errorCount) / float64(totalCount) * 100
	}

	startOfWeek := now.AddDate(0, 0, -7)
	var weekStartCount, weekEndCount int64
	database.DB.Model(&models.User{}).
		Where("created_at >= ?", startOfWeek).
		Count(&weekStartCount)
	weekStart := startOfWeek.AddDate(0, 0, -7)
	database.DB.Model(&models.User{}).
		Where("created_at >= ? AND created_at < ?", weekStart, startOfWeek).
		Count(&weekEndCount)
	if weekEndCount > 0 {
		stats.UserGrowth = float64(weekStartCount-weekEndCount) / float64(weekEndCount) * 100
	}

	startOfWeekApps := now.AddDate(0, 0, -7)
	var weekStartApps, weekEndApps int64
	database.DB.Model(&models.Application{}).
		Where("created_at >= ?", startOfWeekApps).
		Count(&weekStartApps)
	weekStartAppsTime := startOfWeekApps.AddDate(0, 0, -7)
	database.DB.Model(&models.Application{}).
		Where("created_at >= ? AND created_at < ?", weekStartAppsTime, startOfWeekApps).
		Count(&weekEndApps)
	if weekEndApps > 0 {
		stats.AppGrowth = float64(weekStartApps-weekEndApps) / float64(weekEndApps) * 100
	}

	return stats, nil
}
