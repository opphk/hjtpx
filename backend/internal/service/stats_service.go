package service

import (
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type StatsService struct{}

func NewStatsService() *StatsService {
	return &StatsService{}
}

type OverviewStats struct {
	TotalVerifications int64   `json:"total_verifications"`
	SuccessCount       int64   `json:"success_count"`
	FailedCount        int64   `json:"failed_count"`
	PendingCount       int64   `json:"pending_count"`
	SuccessRate        float64 `json:"success_rate"`
	AvgRiskScore       float64 `json:"avg_risk_score"`
	TotalApplications  int64   `json:"total_applications"`
	TotalUsers         int64   `json:"total_users"`
}

func (s *StatsService) GetOverviewStats() (*OverviewStats, error) {
	var stats OverviewStats

	database.DB.Model(&models.Verification{}).Count(&stats.TotalVerifications)
	database.DB.Model(&models.Verification{}).Where("status = ?", "success").Count(&stats.SuccessCount)
	database.DB.Model(&models.Verification{}).Where("status = ?", "failed").Count(&stats.FailedCount)
	database.DB.Model(&models.Verification{}).Where("status = ?", "pending").Count(&stats.PendingCount)
	database.DB.Model(&models.Application{}).Count(&stats.TotalApplications)
	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	if stats.TotalVerifications > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalVerifications) * 100
	}

	rows, _ := database.DB.Model(&models.Verification{}).Select("COALESCE(AVG(risk_score), 0) as avg_risk").Rows()
	if rows.Next() {
		rows.Scan(&stats.AvgRiskScore)
	}

	return &stats, nil
}

type CaptchaTypeStats struct {
	CaptchaType  string  `json:"captcha_type"`
	TotalCount   int64   `json:"total_count"`
	SuccessCount int64   `json:"success_count"`
	FailedCount  int64   `json:"failed_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgRiskScore float64 `json:"avg_risk_score"`
	AvgDuration  int64   `json:"avg_duration"`
}

func (s *StatsService) GetCaptchaTypeStats() ([]CaptchaTypeStats, error) {
	var results []CaptchaTypeStats

	rows, err := database.DB.Model(&models.Verification{}).
		Select("captcha_type, COUNT(*) as total_count, SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count, SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count, COALESCE(AVG(risk_score), 0) as avg_risk_score, COALESCE(AVG(duration), 0) as avg_duration").
		Group("captcha_type").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat CaptchaTypeStats
		if err := rows.Scan(&stat.CaptchaType, &stat.TotalCount, &stat.SuccessCount, &stat.FailedCount, &stat.AvgRiskScore, &stat.AvgDuration); err != nil {
			continue
		}
		if stat.TotalCount > 0 {
			stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.TotalCount) * 100
		}
		results = append(results, stat)
	}

	return results, nil
}

type ApplicationStats struct {
	ApplicationID   uint    `json:"application_id"`
	ApplicationName string  `json:"application_name"`
	TotalCount      int64   `json:"total_count"`
	SuccessCount    int64   `json:"success_count"`
	FailedCount     int64   `json:"failed_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgRiskScore    float64 `json:"avg_risk_score"`
}

func (s *StatsService) GetApplicationStats(limit int) ([]ApplicationStats, error) {
	if limit <= 0 {
		limit = 10
	}

	var results []ApplicationStats

	rows, err := database.DB.Model(&models.Verification{}).
		Select("verifications.application_id, applications.name as application_name, COUNT(*) as total_count, SUM(CASE WHEN verifications.status = 'success' THEN 1 ELSE 0 END) as success_count, SUM(CASE WHEN verifications.status = 'failed' THEN 1 ELSE 0 END) as failed_count, COALESCE(AVG(verifications.risk_score), 0) as avg_risk_score").
		Joins("LEFT JOIN applications ON verifications.application_id = applications.id").
		Group("verifications.application_id, applications.name").
		Order("total_count DESC").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat ApplicationStats
		if err := rows.Scan(&stat.ApplicationID, &stat.ApplicationName, &stat.TotalCount, &stat.SuccessCount, &stat.FailedCount, &stat.AvgRiskScore); err != nil {
			continue
		}
		if stat.TotalCount > 0 {
			stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.TotalCount) * 100
		}
		results = append(results, stat)
	}

	return results, nil
}

type TrendDataPoint struct {
	Date         string  `json:"date"`
	TotalCount   int64   `json:"total_count"`
	SuccessCount int64   `json:"success_count"`
	FailedCount  int64   `json:"failed_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgRiskScore float64 `json:"avg_risk_score"`
}

func (s *StatsService) GetTrendData(days int) ([]TrendDataPoint, error) {
	if days <= 0 {
		days = 7
	}

	now := time.Now()
	var results []TrendDataPoint

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var totalCount, successCount, failedCount int64
		var avgRiskScore float64

		database.DB.Model(&models.Verification{}).
			Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
			Count(&totalCount)

		database.DB.Model(&models.Verification{}).
			Where("status = ? AND created_at >= ? AND created_at < ?", "success", startOfDay, endOfDay).
			Count(&successCount)

		database.DB.Model(&models.Verification{}).
			Where("status = ? AND created_at >= ? AND created_at < ?", "failed", startOfDay, endOfDay).
			Count(&failedCount)

		rows, _ := database.DB.Model(&models.Verification{}).
			Select("COALESCE(AVG(risk_score), 0) as avg_risk").
			Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
			Rows()
		if rows.Next() {
			rows.Scan(&avgRiskScore)
		}

		successRate := 0.0
		if totalCount > 0 {
			successRate = float64(successCount) / float64(totalCount) * 100
		}

		results = append(results, TrendDataPoint{
			Date:         startOfDay.Format("2006-01-02"),
			TotalCount:   totalCount,
			SuccessCount: successCount,
			FailedCount:  failedCount,
			SuccessRate:  successRate,
			AvgRiskScore: avgRiskScore,
		})
	}

	return results, nil
}

type HourlyStats struct {
	Hour         int   `json:"hour"`
	TotalCount   int64 `json:"total_count"`
	SuccessCount int64 `json:"success_count"`
	FailedCount  int64 `json:"failed_count"`
}

func (s *StatsService) GetHourlyStats(date string) ([]HourlyStats, error) {
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		targetDate = time.Now()
	}

	var results []HourlyStats

	for hour := 0; hour < 24; hour++ {
		startHour := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), hour, 0, 0, 0, targetDate.Location())
		endHour := startHour.Add(time.Hour)

		var totalCount, successCount, failedCount int64

		database.DB.Model(&models.Verification{}).
			Where("created_at >= ? AND created_at < ?", startHour, endHour).
			Count(&totalCount)

		database.DB.Model(&models.Verification{}).
			Where("status = ? AND created_at >= ? AND created_at < ?", "success", startHour, endHour).
			Count(&successCount)

		database.DB.Model(&models.Verification{}).
			Where("status = ? AND created_at >= ? AND created_at < ?", "failed", startHour, endHour).
			Count(&failedCount)

		results = append(results, HourlyStats{
			Hour:         hour,
			TotalCount:   totalCount,
			SuccessCount: successCount,
			FailedCount:  failedCount,
		})
	}

	return results, nil
}

type RealtimeStats struct {
	CurrentMinute   int64   `json:"current_minute"`
	LastMinute      int64   `json:"last_minute"`
	CurrentHour     int64   `json:"current_hour"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime float64 `json:"avg_response_time"`
	ActiveSessions  int64   `json:"active_sessions"`
}

func (s *StatsService) GetRealtimeStats() (*RealtimeStats, error) {
	now := time.Now()
	startOfMinute := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	endOfMinute := startOfMinute.Add(time.Minute)
	startOfLastMinute := startOfMinute.Add(-time.Minute)
	startOfHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	var stats RealtimeStats

	database.DB.Model(&models.Verification{}).
		Where("created_at >= ? AND created_at < ?", startOfMinute, endOfMinute).
		Count(&stats.CurrentMinute)

	database.DB.Model(&models.Verification{}).
		Where("created_at >= ? AND created_at < ?", startOfLastMinute, startOfMinute).
		Count(&stats.LastMinute)

	database.DB.Model(&models.Verification{}).
		Where("created_at >= ?", startOfHour).
		Count(&stats.CurrentHour)

	var totalCount, successCount int64
	database.DB.Model(&models.Verification{}).
		Where("created_at >= ?", startOfHour).
		Count(&totalCount)
	database.DB.Model(&models.Verification{}).
		Where("status = ? AND created_at >= ?", "success", startOfHour).
		Count(&successCount)

	if totalCount > 0 {
		stats.SuccessRate = float64(successCount) / float64(totalCount) * 100
	}

	rows, _ := database.DB.Model(&models.Verification{}).
		Select("COALESCE(AVG(duration), 0) as avg_duration").
		Where("created_at >= ?", startOfHour).
		Rows()
	if rows.Next() {
		rows.Scan(&stats.AvgResponseTime)
	}

	database.DB.Model(&models.Verification{}).
		Where("created_at >= ?", now.Add(-30*time.Minute)).
		Distinct("session_id").
		Count(&stats.ActiveSessions)

	return &stats, nil
}

type RiskDistribution struct {
	RiskLevel  string  `json:"risk_level"`
	MinScore   float64 `json:"min_score"`
	MaxScore   float64 `json:"max_score"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

func (s *StatsService) GetRiskDistribution() ([]RiskDistribution, error) {
	var totalCount int64
	database.DB.Model(&models.Verification{}).Count(&totalCount)

	if totalCount == 0 {
		return []RiskDistribution{
			{RiskLevel: "Low (0-30)", MinScore: 0, MaxScore: 30, Count: 0, Percentage: 0},
			{RiskLevel: "Medium (30-60)", MinScore: 30, MaxScore: 60, Count: 0, Percentage: 0},
			{RiskLevel: "High (60-80)", MinScore: 60, MaxScore: 80, Count: 0, Percentage: 0},
			{RiskLevel: "Critical (80-100)", MinScore: 80, MaxScore: 100, Count: 0, Percentage: 0},
		}, nil
	}

	levels := []struct {
		level    string
		min, max float64
	}{
		{"Low (0-30)", 0, 30},
		{"Medium (30-60)", 30, 60},
		{"High (60-80)", 60, 80},
		{"Critical (80-100)", 80, 100},
	}

	var distributions []RiskDistribution

	for _, l := range levels {
		var count int64
		query := database.DB.Model(&models.Verification{})
		if l.min == 0 {
			query = query.Where("risk_score >= ? AND risk_score < ?", l.min, l.max)
		} else {
			query = query.Where("risk_score >= ? AND risk_score < ?", l.min, l.max)
		}
		query.Count(&count)

		distributions = append(distributions, RiskDistribution{
			RiskLevel:  l.level,
			MinScore:   l.min,
			MaxScore:   l.max,
			Count:      count,
			Percentage: float64(count) / float64(totalCount) * 100,
		})
	}

	return distributions, nil
}

type TopIPs struct {
	IPAddress    string  `json:"ip_address"`
	RequestCount int64   `json:"request_count"`
	SuccessRate  float64 `json:"success_rate"`
}

func (s *StatsService) GetTopIPs(limit int) ([]TopIPs, error) {
	if limit <= 0 {
		limit = 10
	}

	var results []TopIPs

	rows, err := database.DB.Model(&models.Verification{}).
		Select("ip_address, COUNT(*) as request_count, CASE WHEN COUNT(*) > 0 THEN CAST(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS FLOAT) / CAST(COUNT(*) AS FLOAT) * 100 ELSE 0 END as success_rate").
		Where("ip_address != ''").
		Group("ip_address").
		Order("request_count DESC").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ip TopIPs
		if err := rows.Scan(&ip.IPAddress, &ip.RequestCount, &ip.SuccessRate); err != nil {
			continue
		}
		results = append(results, ip)
	}

	return results, nil
}

func (s *StatsService) GetLogStatistics() (map[string]interface{}, error) {
	var totalCount, successCount, failedCount int64
	var avgRiskScore float64

	database.DB.Model(&models.VerificationLog{}).Count(&totalCount)
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "success").Count(&successCount)
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "failed").Count(&failedCount)

	rows, _ := database.DB.Model(&models.VerificationLog{}).Select("COALESCE(AVG(risk_score), 0) as avg_risk").Rows()
	if rows.Next() {
		rows.Scan(&avgRiskScore)
	}

	type CaptchaStats struct {
		CaptchaType string  `json:"captcha_type"`
		Count       int64   `json:"count"`
		SuccessRate float64 `json:"success_rate"`
	}

	var captchaStats []CaptchaStats
	database.DB.Model(&models.VerificationLog{}).
		Select("captcha_type, COUNT(*) as count").
		Group("captcha_type").
		Scan(&captchaStats)

	for i := range captchaStats {
		var success int64
		database.DB.Model(&models.VerificationLog{}).
			Where("captcha_type = ? AND status = ?", captchaStats[i].CaptchaType, "success").
			Count(&success)
		if captchaStats[i].Count > 0 {
			captchaStats[i].SuccessRate = float64(success) / float64(captchaStats[i].Count)
		}
	}

	successRate := 0.0
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount)
	}

	return map[string]interface{}{
		"total_count":    totalCount,
		"success_count":  successCount,
		"failed_count":   failedCount,
		"success_rate":   successRate,
		"avg_risk_score": avgRiskScore,
		"captcha_stats":  captchaStats,
	}, nil
}

func (s *StatsService) GenerateReport(reportType string, startDate, endDate time.Time) (string, error) {
	switch reportType {
	case "daily":
		return s.generateDailyReport(startDate)
	case "weekly":
		return s.generateWeeklyReport(startDate)
	case "monthly":
		return s.generateMonthlyReport(startDate)
	default:
		return "", fmt.Errorf("unsupported report type: %s", reportType)
	}
}

func (s *StatsService) generateDailyReport(date time.Time) (string, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var totalCount, successCount, failedCount int64
	database.DB.Model(&models.Verification{}).Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).Count(&totalCount)
	database.DB.Model(&models.Verification{}).Where("status = ? AND created_at >= ? AND created_at < ?", "success", startOfDay, endOfDay).Count(&successCount)
	database.DB.Model(&models.Verification{}).Where("status = ? AND created_at >= ? AND created_at < ?", "failed", startOfDay, endOfDay).Count(&failedCount)

	report := fmt.Sprintf("=== Daily Report: %s ===\n", date.Format("2006-01-02"))
	report += fmt.Sprintf("Total Verifications: %d\n", totalCount)
	report += fmt.Sprintf("Success: %d (%.2f%%)\n", successCount, float64(successCount)/float64(totalCount)*100)
	report += fmt.Sprintf("Failed: %d (%.2f%%)\n", failedCount, float64(failedCount)/float64(totalCount)*100)

	return report, nil
}

func (s *StatsService) generateWeeklyReport(date time.Time) (string, error) {
	startOfWeek := date.AddDate(0, 0, -int(date.Weekday()))
	endOfWeek := startOfWeek.AddDate(0, 0, 7)

	var totalCount, successCount, failedCount int64
	database.DB.Model(&models.Verification{}).Where("created_at >= ? AND created_at < ?", startOfWeek, endOfWeek).Count(&totalCount)
	database.DB.Model(&models.Verification{}).Where("status = ? AND created_at >= ? AND created_at < ?", "success", startOfWeek, endOfWeek).Count(&successCount)
	database.DB.Model(&models.Verification{}).Where("status = ? AND created_at >= ? AND created_at < ?", "failed", startOfWeek, endOfWeek).Count(&failedCount)

	report := fmt.Sprintf("=== Weekly Report: %s - %s ===\n", startOfWeek.Format("2006-01-02"), endOfWeek.AddDate(0, 0, -1).Format("2006-01-02"))
	report += fmt.Sprintf("Total Verifications: %d\n", totalCount)
	report += fmt.Sprintf("Success: %d (%.2f%%)\n", successCount, float64(successCount)/float64(totalCount)*100)
	report += fmt.Sprintf("Failed: %d (%.2f%%)\n", failedCount, float64(failedCount)/float64(totalCount)*100)

	return report, nil
}

func (s *StatsService) generateMonthlyReport(date time.Time) (string, error) {
	startOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	var totalCount, successCount, failedCount int64
	database.DB.Model(&models.Verification{}).Where("created_at >= ? AND created_at < ?", startOfMonth, endOfMonth).Count(&totalCount)
	database.DB.Model(&models.Verification{}).Where("status = ? AND created_at >= ? AND created_at < ?", "success", startOfMonth, endOfMonth).Count(&successCount)
	database.DB.Model(&models.Verification{}).Where("status = ? AND created_at >= ? AND created_at < ?", "failed", startOfMonth, endOfMonth).Count(&failedCount)

	report := fmt.Sprintf("=== Monthly Report: %s ===\n", date.Format("2006-01"))
	report += fmt.Sprintf("Total Verifications: %d\n", totalCount)
	report += fmt.Sprintf("Success: %d (%.2f%%)\n", successCount, float64(successCount)/float64(totalCount)*100)
	report += fmt.Sprintf("Failed: %d (%.2f%%)\n", failedCount, float64(failedCount)/float64(totalCount)*100)

	return report, nil
}
