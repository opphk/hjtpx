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
	Summary          *Summary                `json:"summary"`
	Trend            []TrendData             `json:"trend"`
	RiskDistribution *RiskDistributionData   `json:"risk_distribution"`
	CaptchaType      []CaptchaTypeData       `json:"captcha_type"`
}

type Summary struct {
	TotalRequests    int64   `json:"total_requests"`
	PassRate         float64 `json:"pass_rate"`
	BlockRate        float64 `json:"block_rate"`
	AvgResponseTime  int64   `json:"avg_response_time"`
}

type TrendData struct {
	Time     string `json:"time"`
	Requests int64  `json:"requests"`
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

	return data, nil
}

func (s *DashboardService) getSummary() (*Summary, error) {
	summary := &Summary{}

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

	return summary, nil
}

func (s *DashboardService) getTrendData(period string) ([]TrendData, error) {
	now := time.Now()
	var data []TrendData

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

			data = append(data, TrendData{
				Time:     fmt.Sprintf("%02d:00", hour.Hour()),
				Requests: count,
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

			data = append(data, TrendData{
				Time:     day.Format("01-02"),
				Requests: count,
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

			data = append(data, TrendData{
				Time:     fmt.Sprintf("第%d周", 7-i),
				Requests: count,
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

			data = append(data, TrendData{
				Time:     day.Format("01-02"),
				Requests: count,
			})
		}
	default:
		return nil, errors.New("unsupported period")
	}

	return data, nil
}

func (s *DashboardService) getRiskDistribution() (*RiskDistributionData, error) {
	distribution := &RiskDistributionData{}

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
		"slider":   "滑动验证",
		"click":    "点选验证",
		"image":    "图片验证",
		"text":     "文字验证",
		"gesture":  "手势验证",
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
	}

	csv += "\n时间,请求数\n"
	if data.Trend != nil {
		for _, t := range data.Trend {
			csv += fmt.Sprintf("%s,%d\n", t.Time, t.Requests)
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
