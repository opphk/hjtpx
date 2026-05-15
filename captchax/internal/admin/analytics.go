package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"captchax/internal/repository"
)

type AnalyticsService struct {
	captchaRepo *repository.CaptchaRepo
	db          *sql.DB
}

func NewAnalyticsService(captchaRepo *repository.CaptchaRepo, db *sql.DB) *AnalyticsService {
	return &AnalyticsService{
		captchaRepo: captchaRepo,
		db:          db,
	}
}

type TimeRange struct {
	Start time.Time
	End   time.Time
	Label string
}

func ParseTimeRange(rangeType string) TimeRange {
	now := time.Now()
	end := now
	var start time.Time

	switch rangeType {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "yesterday":
		start = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "7d":
		start = now.AddDate(0, 0, -7)
	case "30d":
		start = now.AddDate(0, 0, -30)
	case "90d":
		start = now.AddDate(0, 0, -90)
	default:
		start = now.AddDate(0, 0, -7)
	}

	return TimeRange{
		Start: start,
		End:   end,
		Label: rangeType,
	}
}

func ParseCustomTimeRange(startStr, endStr string) (TimeRange, error) {
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return TimeRange{}, fmt.Errorf("invalid start date format: %w", err)
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return TimeRange{}, fmt.Errorf("invalid end date format: %w", err)
	}
	end = end.Add(24*time.Hour - time.Second)
	return TimeRange{
		Start: start,
		End:   end,
		Label: "custom",
	}, nil
}

type OverviewStats struct {
	TotalVerifications  int64            `json:"total_verifications"`
	VerifiedCount       int64            `json:"verified_count"`
	RejectedCount       int64            `json:"rejected_count"`
	SuccessRate         float64          `json:"success_rate"`
	AvgResponseTime     float64          `json:"avg_response_time"`
	AvgRiskScore        float64          `json:"avg_risk_score"`
	UniqueIPs           int64            `json:"unique_ips"`
	UniqueClients       int64            `json:"unique_clients"`
	TimeRange           string           `json:"time_range"`
	VerificationsChange float64          `json:"verifications_change"`
	SuccessRateChange   float64          `json:"success_rate_change"`
	ByCaptchaType       []TypeStats      `json:"by_captcha_type"`
}

type TypeStats struct {
	Type   string `json:"type"`
	Label  string `json:"label"`
	Count  int64  `json:"count"`
	Rate   string `json:"rate"`
}

func (s *AnalyticsService) GetOverview(ctx context.Context, timeRange TimeRange, prevRange TimeRange) (*OverviewStats, error) {
	stats := &OverviewStats{
		TimeRange: timeRange.Label,
	}

	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE result = true) as verified,
			COUNT(*) FILTER (WHERE result = false) as rejected,
			COALESCE(AVG(duration), 0) as avg_duration,
			COALESCE(AVG(risk_score), 0) as avg_risk_score,
			COUNT(DISTINCT ip) as unique_ips,
			COUNT(DISTINCT client_id) as unique_clients
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
	`

	err := s.db.QueryRowContext(ctx, query, timeRange.Start, timeRange.End).Scan(
		&stats.TotalVerifications,
		&stats.VerifiedCount,
		&stats.RejectedCount,
		&stats.AvgResponseTime,
		&stats.AvgRiskScore,
		&stats.UniqueIPs,
		&stats.UniqueClients,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get overview stats: %w", err)
	}

	if stats.TotalVerifications > 0 {
		stats.SuccessRate = float64(stats.VerifiedCount) / float64(stats.TotalVerifications) * 100
	}

	stats.ByCaptchaType, err = s.getTypeStats(ctx, timeRange.Start, timeRange.End)
	if err != nil {
		stats.ByCaptchaType = []TypeStats{}
	}

	currentTotal, prevTotal := stats.TotalVerifications, int64(0)
	if prevRange.Start.Before(timeRange.Start) {
		prevQuery := `SELECT COUNT(*) FROM captcha_logs WHERE created_at >= $1 AND created_at <= $2`
		s.db.QueryRowContext(ctx, prevQuery, prevRange.Start, prevRange.End).Scan(&prevTotal)
	}

	if prevTotal > 0 {
		stats.VerificationsChange = (float64(currentTotal) - float64(prevTotal)) / float64(prevTotal) * 100
	}

	currentSuccessRate := stats.SuccessRate
	prevSuccessRate := 0.0
	if prevTotal > 0 {
		var prevVerified int64
		prevQuery := `SELECT COUNT(*) FILTER (WHERE result = true) FROM captcha_logs WHERE created_at >= $1 AND created_at <= $2`
		s.db.QueryRowContext(ctx, prevQuery, prevRange.Start, prevRange.End).Scan(&prevVerified)
		prevSuccessRate = float64(prevVerified) / float64(prevTotal) * 100
	}
	stats.SuccessRateChange = currentSuccessRate - prevSuccessRate

	return stats, nil
}

func (s *AnalyticsService) getTypeStats(ctx context.Context, start, end time.Time) ([]TypeStats, error) {
	query := `
		SELECT 
			captcha_type,
			COUNT(*) as count
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY captcha_type
		ORDER BY count DESC
	`

	rows, err := s.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []TypeStats
	var totalCount int64
	typeCounts := make(map[string]int64)

	for rows.Next() {
		var t string
		var count int64
		if err := rows.Scan(&t, &count); err != nil {
			return nil, err
		}
		typeCounts[t] = count
		totalCount += count
	}

	typeLabels := map[string]string{
		"slider":  "滑动验证",
		"click":   "点选验证",
		"rotate":  "旋转验证",
		"puzzle":  "拼图验证",
		"icon":    "图标验证",
		"text":    "文字验证",
	}

	for t, count := range typeCounts {
		rate := "0%"
		if totalCount > 0 {
			rate = fmt.Sprintf("%.1f%%", float64(count)/float64(totalCount)*100)
		}
		stats = append(stats, TypeStats{
			Type:  t,
			Label: typeLabels[t],
			Count: count,
			Rate:  rate,
		})
	}

	return stats, nil
}

type TrendData struct {
	Time           string `json:"time"`
	Verified       int64  `json:"verified"`
	Rejected       int64  `json:"rejected"`
	Total          int64  `json:"total"`
	SuccessRate    string `json:"success_rate"`
	AvgResponseTime float64 `json:"avg_response_time"`
}

func (s *AnalyticsService) GetTrends(ctx context.Context, timeRange TimeRange, interval string) ([]TrendData, error) {
	var truncUnit string
	switch interval {
	case "hour":
		truncUnit = "hour"
	case "day":
		truncUnit = "day"
	case "week":
		truncUnit = "week"
	default:
		if timeRange.End.Sub(timeRange.Start) > 48*time.Hour {
			truncUnit = "day"
		} else {
			truncUnit = "hour"
		}
	}

	query := fmt.Sprintf(`
		SELECT 
			DATE_TRUNC('%s', created_at) as time_bucket,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE result = true) as verified,
			COUNT(*) FILTER (WHERE result = false) as rejected,
			COALESCE(AVG(duration), 0) as avg_duration
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY time_bucket
		ORDER BY time_bucket
	`, truncUnit)

	rows, err := s.db.QueryContext(ctx, query, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}
	defer rows.Close()

	var trends []TrendData
	for rows.Next() {
		var t TrendData
		var timeBucket time.Time
		if err := rows.Scan(&timeBucket, &t.Total, &t.Verified, &t.Rejected, &t.AvgResponseTime); err != nil {
			return nil, err
		}

		switch interval {
		case "hour":
			t.Time = timeBucket.Format("15:04")
		case "day":
			t.Time = timeBucket.Format("01-02")
		case "week":
			t.Time = timeBucket.Format("01-02")
		default:
			if truncUnit == "hour" {
				t.Time = timeBucket.Format("15:04")
			} else {
				t.Time = timeBucket.Format("01-02")
			}
		}

		if t.Total > 0 {
			t.SuccessRate = fmt.Sprintf("%.1f%%", float64(t.Verified)/float64(t.Total)*100)
		} else {
			t.SuccessRate = "0%"
		}

		trends = append(trends, t)
	}

	return trends, nil
}

type DistributionData struct {
	Type   string  `json:"type"`
	Label  string  `json:"label"`
	Count  int64   `json:"count"`
	Rate   float64 `json:"rate"`
	Color  string  `json:"color"`
}

func (s *AnalyticsService) GetDistribution(ctx context.Context, timeRange TimeRange, groupBy string) ([]DistributionData, error) {
	var groupExpr string
	typeLabels := map[string]string{
		"slider":  "滑动验证",
		"click":   "点选验证",
		"rotate":  "旋转验证",
		"puzzle":  "拼图验证",
		"icon":    "图标验证",
		"text":    "文字验证",
	}

	typeColors := map[string]string{
		"slider":  "#3b82f6",
		"click":   "#8b5cf6",
		"rotate":  "#ec4899",
		"puzzle":  "#f59e0b",
		"icon":    "#10b981",
		"text":    "#06b6d4",
	}

	switch groupBy {
	case "type":
		groupExpr = "captcha_type"
	case "result":
		groupExpr = "CASE WHEN result = true THEN 'success' ELSE 'fail' END"
	case "risk_level":
		groupExpr = `CASE 
			WHEN risk_score < 30 THEN 'low' 
			WHEN risk_score < 70 THEN 'medium' 
			ELSE 'high' END`
	default:
		groupExpr = "captcha_type"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as category,
			COUNT(*) as count
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY category
		ORDER BY count DESC
	`, groupExpr)

	rows, err := s.db.QueryContext(ctx, query, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution: %w", err)
	}
	defer rows.Close()

	var distributions []DistributionData
	var totalCount int64

	for rows.Next() {
		var d DistributionData
		if err := rows.Scan(&d.Type, &d.Count); err != nil {
			return nil, err
		}
		totalCount += d.Count
		distributions = append(distributions, d)
	}

	for i := range distributions {
		if totalCount > 0 {
			distributions[i].Rate = float64(distributions[i].Count) / float64(totalCount) * 100
		}
		if groupBy == "type" {
			distributions[i].Label = typeLabels[distributions[i].Type]
			distributions[i].Color = typeColors[distributions[i].Type]
			if distributions[i].Label == "" {
				distributions[i].Label = distributions[i].Type
			}
			if distributions[i].Color == "" {
				distributions[i].Color = "#94a3b8"
			}
		} else if groupBy == "result" {
			if distributions[i].Type == "success" {
				distributions[i].Label = "验证成功"
				distributions[i].Color = "#10b981"
			} else {
				distributions[i].Label = "验证失败"
				distributions[i].Color = "#ef4444"
			}
		} else if groupBy == "risk_level" {
			if distributions[i].Type == "low" {
				distributions[i].Label = "低风险"
				distributions[i].Color = "#22c55e"
			} else if distributions[i].Type == "medium" {
				distributions[i].Label = "中风险"
				distributions[i].Color = "#eab308"
			} else {
				distributions[i].Label = "高风险"
				distributions[i].Color = "#ef4444"
			}
		}
	}

	return distributions, nil
}

type GeoStats struct {
	Country    string `json:"country"`
	CountryCode string `json:"country_code"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Count      int64  `json:"count"`
	SuccessRate float64 `json:"success_rate"`
	RiskScore  float64 `json:"risk_score"`
}

func (s *AnalyticsService) GetGeoDistribution(ctx context.Context, timeRange TimeRange, limit int) ([]GeoStats, error) {
	query := `
		SELECT 
			COALESCE(SPLIT_PART(ip, '.', 1) || '.' || SPLIT_PART(ip, '.', 2), 'unknown') as region_prefix,
			COUNT(*) as count,
			COUNT(*) FILTER (WHERE result = true) as success_count,
			AVG(risk_score) as avg_risk
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY region_prefix
		ORDER BY count DESC
		LIMIT $3
	`

	rows, err := s.db.QueryContext(ctx, query, timeRange.Start, timeRange.End, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get geo distribution: %w", err)
	}
	defer rows.Close()

	var geoStats []GeoStats
	for rows.Next() {
		var g GeoStats
		if err := rows.Scan(&g.Province, &g.Count, &g.SuccessRate, &g.RiskScore); err != nil {
			return nil, err
		}
		if g.Count > 0 {
			g.SuccessRate = float64(g.SuccessRate) / float64(g.Count) * 100
		}
		g.Country = "中国"
		g.CountryCode = "CN"
		geoStats = append(geoStats, g)
	}

	return geoStats, nil
}

type DeviceStats struct {
	DeviceType   string  `json:"device_type"`
	Browser      string  `json:"browser"`
	OS           string  `json:"os"`
	Count        int64   `json:"count"`
	Rate         float64 `json:"rate"`
	SuccessRate  float64 `json:"success_rate"`
	AvgRiskScore float64 `json:"avg_risk_score"`
}

func (s *AnalyticsService) GetDeviceDistribution(ctx context.Context, timeRange TimeRange) ([]DeviceStats, error) {
	query := `
		SELECT 
			user_agent,
			COUNT(*) as count,
			COUNT(*) FILTER (WHERE result = true) as success_count,
			AVG(risk_score) as avg_risk
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY user_agent
		ORDER BY count DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get device distribution: %w", err)
	}
	defer rows.Close()

	var deviceStats []DeviceStats
	var totalCount int64

	typeDeviceMap := make(map[string]struct {
		stats       DeviceStats
		successCount int64
	})

	for rows.Next() {
		var ua sql.NullString
		var count, successCount int64
		var avgRisk float64

		if err := rows.Scan(&ua, &count, &successCount, &avgRisk); err != nil {
			continue
		}

		totalCount += count

		deviceType := "Unknown"
		browser := "Unknown"
		os := "Unknown"

		if ua.Valid && ua.String != "" {
			uaStr := strings.ToLower(ua.String)

			if strings.Contains(uaStr, "mobile") || strings.Contains(uaStr, "android") || strings.Contains(uaStr, "iphone") {
				deviceType = "Mobile"
			} else if strings.Contains(uaStr, "tablet") || strings.Contains(uaStr, "ipad") {
				deviceType = "Tablet"
			} else {
				deviceType = "Desktop"
			}

			switch {
			case strings.Contains(uaStr, "chrome") && !strings.Contains(uaStr, "edge"):
				browser = "Chrome"
			case strings.Contains(uaStr, "firefox"):
				browser = "Firefox"
			case strings.Contains(uaStr, "safari") && !strings.Contains(uaStr, "chrome"):
				browser = "Safari"
			case strings.Contains(uaStr, "edge"):
				browser = "Edge"
			case strings.Contains(uaStr, "msie") || strings.Contains(uaStr, "trident"):
				browser = "IE"
			}

			switch {
			case strings.Contains(uaStr, "windows"):
				os = "Windows"
			case strings.Contains(uaStr, "mac os") || strings.Contains(uaStr, "macos"):
				os = "macOS"
			case strings.Contains(uaStr, "linux") && !strings.Contains(uaStr, "android"):
				os = "Linux"
			case strings.Contains(uaStr, "android"):
				os = "Android"
			case strings.Contains(uaStr, "ios") || strings.Contains(uaStr, "iphone") || strings.Contains(uaStr, "ipad"):
				os = "iOS"
			}
		}

		key := deviceType + "|" + browser + "|" + os
		if existing, ok := typeDeviceMap[key]; ok {
			existing.stats.Count += count
			existing.successCount += successCount
			if existing.stats.Count > 0 {
				existing.stats.SuccessRate = float64(existing.successCount) / float64(existing.stats.Count) * 100
			}
			typeDeviceMap[key] = existing
		} else {
			successRate := float64(0)
			if count > 0 {
				successRate = float64(successCount) / float64(count) * 100
			}
			typeDeviceMap[key] = struct {
				stats       DeviceStats
				successCount int64
			}{
				stats: DeviceStats{
					DeviceType:   deviceType,
					Browser:      browser,
					OS:           os,
					Count:        count,
					SuccessRate:  successRate,
					AvgRiskScore: avgRisk,
				},
				successCount: successCount,
			}
		}
	}

	for _, ds := range typeDeviceMap {
		if totalCount > 0 {
			ds.stats.Rate = float64(ds.stats.Count) / float64(totalCount) * 100
		}
		deviceStats = append(deviceStats, ds.stats)
	}

	return deviceStats, nil
}

type RiskLevelStats struct {
	Level    string `json:"level"`
	Label    string `json:"label"`
	Count    int64  `json:"count"`
	Rate     float64 `json:"rate"`
	Color    string `json:"color"`
}

func (s *AnalyticsService) GetRiskDistribution(ctx context.Context, timeRange TimeRange) ([]RiskLevelStats, error) {
	query := `
		SELECT 
			CASE 
				WHEN risk_score < 30 THEN 'low'
				WHEN risk_score < 70 THEN 'medium'
				ELSE 'high'
			END as risk_level,
			COUNT(*) as count
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY risk_level
		ORDER BY 
			CASE risk_level 
				WHEN 'low' THEN 1 
				WHEN 'medium' THEN 2 
				WHEN 'high' THEN 3 
			END
	`

	rows, err := s.db.QueryContext(ctx, query, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk distribution: %w", err)
	}
	defer rows.Close()

	var riskStats []RiskLevelStats
	var totalCount int64

	levelInfo := map[string]struct {
		Label string
		Color string
	}{
		"low":    {"低风险", "#22c55e"},
		"medium": {"中风险", "#eab308"},
		"high":   {"高风险", "#ef4444"},
	}

	for rows.Next() {
		var r RiskLevelStats
		if err := rows.Scan(&r.Level, &r.Count); err != nil {
			return nil, err
		}
		totalCount += r.Count
		if info, ok := levelInfo[r.Level]; ok {
			r.Label = info.Label
			r.Color = info.Color
		}
		riskStats = append(riskStats, r)
	}

	for i := range riskStats {
		if totalCount > 0 {
			riskStats[i].Rate = float64(riskStats[i].Count) / float64(totalCount) * 100
		}
	}

	return riskStats, nil
}

type HourlyPattern struct {
	Hour        int   `json:"hour"`
	AvgCount    int64 `json:"avg_count"`
	AvgSuccess  float64 `json:"avg_success_rate"`
}

func (s *AnalyticsService) GetHourlyPattern(ctx context.Context, days int) ([]HourlyPattern, error) {
	query := `
		SELECT 
			EXTRACT(HOUR FROM created_at) as hour,
			AVG(hourly_count) as avg_count,
			AVG(hourly_success_rate) as avg_success
		FROM (
			SELECT 
				DATE_TRUNC('hour', created_at) as hour,
				COUNT(*) as hourly_count,
				COUNT(*) FILTER (WHERE result = true)::float / NULLIF(COUNT(*), 0) * 100 as hourly_success_rate
			FROM captcha_logs
			WHERE created_at >= NOW() - INTERVAL '1 day' * $1
			GROUP BY DATE_TRUNC('hour', created_at)
		) hourly_data
		GROUP BY EXTRACT(HOUR FROM hour)
		ORDER BY hour
	`

	rows, err := s.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly pattern: %w", err)
	}
	defer rows.Close()

	var patterns []HourlyPattern
	for rows.Next() {
		var p HourlyPattern
		if err := rows.Scan(&p.Hour, &p.AvgCount, &p.AvgSuccess); err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}

	return patterns, nil
}
