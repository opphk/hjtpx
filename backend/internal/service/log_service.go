package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type LogService struct {
	redisClient *goredis.Client
}

func NewLogService() *LogService {
	return &LogService{
		redisClient: redis.GetClient(),
	}
}

type LogCategory string

const (
	LogCategoryVerification LogCategory = "verification"
	LogCategorySecurity    LogCategory = "security"
	LogCategoryAudit       LogCategory = "audit"
	LogCategorySystem      LogCategory = "system"
	LogCategoryPerformance LogCategory = "performance"
)

type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelCritical LogLevel = "critical"
)

type LogEntry struct {
	ID              uint                 `json:"id"`
	Timestamp       time.Time            `json:"timestamp"`
	Level           LogLevel             `json:"level"`
	Category        LogCategory          `json:"category"`
	Message         string               `json:"message"`
	SessionID       string               `json:"session_id,omitempty"`
	ApplicationID   uint                 `json:"application_id,omitempty"`
	UserID          uint                 `json:"user_id,omitempty"`
	IPAddress       string               `json:"ip_address,omitempty"`
	UserAgent       string               `json:"user_agent,omitempty"`
	CaptchaType     string               `json:"captcha_type,omitempty"`
	Status          string               `json:"status,omitempty"`
	RiskScore       float64              `json:"risk_score,omitempty"`
	Duration        int64                `json:"duration,omitempty"`
	RequestID       string               `json:"request_id,omitempty"`
	TraceID         string               `json:"trace_id,omitempty"`
	SpanID          string               `json:"span_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ErrorDetail     string               `json:"error_detail,omitempty"`
	DeviceFingerprint string             `json:"device_fingerprint,omitempty"`
	GeoLocation     string               `json:"geo_location,omitempty"`
	ResponseTime    float64              `json:"response_time,omitempty"`
	CacheHit        bool                 `json:"cache_hit,omitempty"`
}

func (s *LogService) CreateVerificationLog(log *models.VerificationLog) error {
	ctx := context.Background()
	
	if err := database.DB.Create(log).Error; err != nil {
		return err
	}
	
	s.invalidateLogCache(ctx, log.ApplicationID)
	
	s.recordLogMetrics(ctx, log)
	
	return nil
}

func (s *LogService) recordLogMetrics(ctx context.Context, log *models.VerificationLog) {
	if s.redisClient == nil {
		return
	}
	
	now := time.Now()
	dateKey := now.Format("2006-01-02")
	hourKey := now.Format("2006-01-02:15")
	
	s.redisClient.HIncrBy(ctx, fmt.Sprintf("log:stats:date:%s", dateKey), "total", 1)
	s.redisClient.HIncrBy(ctx, fmt.Sprintf("log:stats:date:%s", dateKey), fmt.Sprintf("status:%s", log.Status), 1)
	s.redisClient.HIncrBy(ctx, fmt.Sprintf("log:stats:hour:%s", hourKey), "total", 1)
	
	if log.RiskScore > 50 {
		s.redisClient.HIncrBy(ctx, fmt.Sprintf("log:stats:date:%s", dateKey), "high_risk", 1)
	}
	
	s.redisClient.Expire(ctx, fmt.Sprintf("log:stats:date:%s", dateKey), 90*24*time.Hour)
	s.redisClient.Expire(ctx, fmt.Sprintf("log:stats:hour:%s", hourKey), 7*24*time.Hour)
}

func (s *LogService) invalidateLogCache(ctx context.Context, appID uint) {
	if s.redisClient == nil {
		return
	}
	
	patterns := []string{
		"log:query:*",
		"log:stats:*",
		"log:analytics:*",
	}
	
	for _, pattern := range patterns {
		iter := s.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			s.redisClient.Del(ctx, iter.Val())
		}
	}
}

func (s *LogService) GetLogByID(id uint) (*models.VerificationLog, error) {
	var log models.VerificationLog
	err := database.DB.Preload("Verification").
		Preload("Verification.BehaviorData").
		Preload("Application").
		First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

type LogQueryParams struct {
	Page          int
	PageSize      int
	ApplicationID uint
	Status        string
	CaptchaType   string
	SessionID     string
	StartDate     time.Time
	EndDate       time.Time
	MinRiskScore  float64
	MaxRiskScore  float64
	IPAddress     string
	UserAgent     string
	Level         LogLevel
	Category      LogCategory
	RequestID     string
	SortBy        string
	SortOrder     string
}

type LogListResult struct {
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
	Logs       []models.VerificationLog `json:"logs"`
	Stats      *LogQueryStats           `json:"stats,omitempty"`
}

type LogQueryStats struct {
	AvgRiskScore   float64            `json:"avg_risk_score"`
	MaxRiskScore   float64            `json:"max_risk_score"`
	MinRiskScore   float64            `json:"min_risk_score"`
	AvgDuration    float64            `json:"avg_duration"`
	TotalDuration  int64              `json:"total_duration"`
	StatusCounts   map[string]int64   `json:"status_counts"`
	TypeCounts     map[string]int64   `json:"type_counts"`
	TopIPs         map[string]int64   `json:"top_ips"`
	CacheHit       bool               `json:"cache_hit"`
	QueryTimeMs    float64             `json:"query_time_ms"`
}

func (s *LogService) QueryLogs(params LogQueryParams) (*LogListResult, error) {
	startTime := time.Now()
	ctx := context.Background()
	
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.SortBy == "" {
		params.SortBy = "created_at"
	}
	if params.SortOrder == "" {
		params.SortOrder = "desc"
	}
	
	cacheKey := s.generateCacheKey(params)
	if cached, err := s.getFromCache(ctx, cacheKey); err == nil && cached != nil {
		result := cached.(*LogListResult)
		result.Stats.CacheHit = true
		result.Stats.QueryTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return result, nil
	}
	
	query := database.DB.Model(&models.VerificationLog{})

	if params.ApplicationID > 0 {
		query = query.Where("application_id = ?", params.ApplicationID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.CaptchaType != "" {
		query = query.Where("captcha_type = ?", params.CaptchaType)
	}
	if params.SessionID != "" {
		query = query.Where("session_id LIKE ?", "%"+params.SessionID+"%")
	}
	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("created_at < ?", params.EndDate)
	}
	if params.MinRiskScore > 0 {
		query = query.Where("risk_score >= ?", params.MinRiskScore)
	}
	if params.MaxRiskScore > 0 {
		query = query.Where("risk_score <= ?", params.MaxRiskScore)
	}
	if params.IPAddress != "" {
		query = query.Where("ip_address LIKE ?", "%"+params.IPAddress+"%")
	}
	if params.RequestID != "" {
		query = query.Where("session_id LIKE ?", "%"+params.RequestID+"%")
	}

	validSortFields := map[string]bool{
		"created_at": true, "id": true, "risk_score": true, 
		"duration": true, "status": true, "captcha_type": true,
	}
	sortField := "created_at"
	if validSortFields[params.SortBy] {
		sortField = params.SortBy
	}
	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	orderClause := fmt.Sprintf("%s %s", sortField, sortOrder)

	var total int64
	countQuery := query.Session(nil)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	var logs []models.VerificationLog
	offset := (params.Page - 1) * params.PageSize
	if err := query.Preload("Application").
		Order(orderClause).
		Offset(offset).
		Limit(params.PageSize).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	stats, _ := s.computeQueryStats(query)

	result := &LogListResult{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Logs:       logs,
		Stats:      stats,
	}
	
	result.Stats.QueryTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
	
	s.saveToCache(ctx, cacheKey, result, 5*time.Minute)

	return result, nil
}

func (s *LogService) computeQueryStats(baseQuery *gorm.DB) (*LogQueryStats, error) {
	stats := &LogQueryStats{
		StatusCounts: make(map[string]int64),
		TypeCounts: make(map[string]int64),
		TopIPs: make(map[string]int64),
	}

	var riskStats struct {
		Avg float64
		Max float64
		Min float64
	}
	err := baseQuery.Session(nil).
		Select("COALESCE(AVG(risk_score), 0) as avg, COALESCE(MAX(risk_score), 0) as max, COALESCE(MIN(risk_score), 0) as min").
		Row().Scan(&riskStats.Avg, &riskStats.Max, &riskStats.Min)
	if err == nil {
		stats.AvgRiskScore = riskStats.Avg
		stats.MaxRiskScore = riskStats.Max
		stats.MinRiskScore = riskStats.Min
	}

	var durationStats struct {
		Avg float64
		Sum int64
	}
	err = baseQuery.Session(nil).
		Select("COALESCE(AVG(duration), 0) as avg, COALESCE(SUM(duration), 0) as sum").
		Row().Scan(&durationStats.Avg, &durationStats.Sum)
	if err == nil {
		stats.AvgDuration = durationStats.Avg
		stats.TotalDuration = durationStats.Sum
	}

	rows, err := baseQuery.Session(nil).
		Select("status, COUNT(*) as count").
		Group("status").
		Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status string
			var count int64
			if rows.Scan(&status, &count) == nil {
				stats.StatusCounts[status] = count
			}
		}
	}

	rows, err = baseQuery.Session(nil).
		Select("captcha_type, COUNT(*) as count").
		Group("captcha_type").
		Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var captchaType string
			var count int64
			if rows.Scan(&captchaType, &count) == nil {
				stats.TypeCounts[captchaType] = count
			}
		}
	}

	rows, err = baseQuery.Session(nil).
		Select("ip_address, COUNT(*) as count").
		Where("ip_address != ''").
		Group("ip_address").
		Order("count DESC").
		Limit(10).
		Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ip string
			var count int64
			if rows.Scan(&ip, &count) == nil {
				stats.TopIPs[ip] = count
			}
		}
	}

	return stats, nil
}

func (s *LogService) generateCacheKey(params LogQueryParams) string {
	data, _ := json.Marshal(params)
	return fmt.Sprintf("log:query:%s", string(data))
}

func (s *LogService) getFromCache(ctx context.Context, key string) (interface{}, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}
	
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	
	var result LogListResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

func (s *LogService) saveToCache(ctx context.Context, key string, value *LogListResult, ttl time.Duration) {
	if s.redisClient == nil {
		return
	}
	
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	
	s.redisClient.Set(ctx, key, data, ttl)
}

type LogExportParams struct {
	ApplicationID uint
	Status        string
	CaptchaType   string
	StartDate     time.Time
	EndDate       time.Time
	Format        string
	IncludeStats  bool
	Stream        bool
}

type LogExportResult struct {
	Data        []byte
	RecordCount int
	FileSize    int64
	Format      string
	GeneratedAt time.Time
	Stats       *LogQueryStats `json:"stats,omitempty"`
}

func (s *LogService) ExportLogs(params LogExportParams) (*LogExportResult, error) {
	query := database.DB.Model(&models.VerificationLog{})

	if params.ApplicationID > 0 {
		query = query.Where("application_id = ?", params.ApplicationID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.CaptchaType != "" {
		query = query.Where("captcha_type = ?", params.CaptchaType)
	}
	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("created_at < ?", params.EndDate)
	}

	var logs []models.VerificationLog
	if err := query.Preload("Application").
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	var data []byte
	var err error

	switch params.Format {
	case "json":
		data, err = s.exportToJSON(logs)
	case "csv":
		data, err = s.exportToCSV(logs)
	default:
		data, err = s.exportToCSV(logs)
	}

	if err != nil {
		return nil, err
	}

	result := &LogExportResult{
		Data:        data,
		RecordCount: len(logs),
		FileSize:    int64(len(data)),
		Format:      params.Format,
		GeneratedAt: time.Now(),
	}

	if params.IncludeStats {
		stats, _ := s.computeQueryStats(query)
		result.Stats = stats
	}

	return result, nil
}

func (s *LogService) exportToJSON(logs []models.VerificationLog) ([]byte, error) {
	type LogExport struct {
		ID              uint      `json:"id"`
		SessionID       string    `json:"session_id"`
		ApplicationID   uint      `json:"application_id"`
		ApplicationName string    `json:"application_name"`
		CaptchaType     string    `json:"captcha_type"`
		Status          string    `json:"status"`
		IPAddress       string    `json:"ip_address"`
		UserAgent       string    `json:"user_agent"`
		RiskScore       float64   `json:"risk_score"`
		RiskLevel       string    `json:"risk_level"`
		Duration        int64     `json:"duration"`
		AnalysisResult  string    `json:"analysis_result"`
		CreatedAt       time.Time `json:"created_at"`
	}

	exports := make([]LogExport, len(logs))
	for i, log := range logs {
		exports[i] = LogExport{
			ID:              log.ID,
			SessionID:       log.SessionID,
			ApplicationID:   log.ApplicationID,
			ApplicationName: log.Application.Name,
			CaptchaType:     log.CaptchaType,
			Status:          log.Status,
			IPAddress:       log.IPAddress,
			UserAgent:       log.UserAgent,
			RiskScore:       log.RiskScore,
			RiskLevel:       calculateRiskLevel(log.RiskScore),
			Duration:        log.Duration,
			AnalysisResult:  log.AnalysisResult,
			CreatedAt:       log.CreatedAt,
		}
	}

	exportData := map[string]interface{}{
		"export_version": "2.0",
		"generated_at":   time.Now().Format(time.RFC3339),
		"record_count":   len(logs),
		"logs":           exports,
	}

	return json.MarshalIndent(exportData, "", "  ")
}

func (s *LogService) exportToCSV(logs []models.VerificationLog) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"ID",
		"Session ID",
		"Application Name",
		"Application ID",
		"Captcha Type",
		"Status",
		"Risk Level",
		"IP Address",
		"Risk Score",
		"Duration (ms)",
		"User Agent",
		"Analysis Result",
		"Created At",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, log := range logs {
		applicationName := ""
		if log.Application.Name != "" {
			applicationName = log.Application.Name
		}

		row := []string{
			fmt.Sprintf("%d", log.ID),
			log.SessionID,
			applicationName,
			fmt.Sprintf("%d", log.ApplicationID),
			log.CaptchaType,
			log.Status,
			calculateRiskLevel(log.RiskScore),
			log.IPAddress,
			fmt.Sprintf("%.2f", log.RiskScore),
			fmt.Sprintf("%d", log.Duration),
			escapeCSVField(log.UserAgent),
			escapeCSVField(log.AnalysisResult),
			log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func escapeCSVField(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func (s *LogService) StreamExportLogs(params LogExportParams, writer io.Writer) error {
	query := database.DB.Model(&models.VerificationLog{})

	if params.ApplicationID > 0 {
		query = query.Where("application_id = ?", params.ApplicationID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.CaptchaType != "" {
		query = query.Where("captcha_type = ?", params.CaptchaType)
	}
	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("created_at < ?", params.EndDate)
	}

	if params.Format == "json" {
		writer.Write([]byte(`{"export_version":"2.0","generated_at":"` + time.Now().Format(time.RFC3339) + `","logs":[`))
	} else {
		csvWriter := csv.NewWriter(writer)
		headers := []string{
			"ID", "Session ID", "Application Name", "Application ID", 
			"Captcha Type", "Status", "Risk Level", "IP Address", 
			"Risk Score", "Duration (ms)", "User Agent", "Analysis Result", "Created At",
		}
		csvWriter.Write(headers)
		csvWriter.Flush()
	}

	offset := 0
	batchSize := 1000
	isFirst := true
	
	for {
		var logs []models.VerificationLog
		if err := query.Preload("Application").
			Order("created_at DESC").
			Offset(offset).
			Limit(batchSize).
			Find(&logs).Error; err != nil {
			return err
		}

		if len(logs) == 0 {
			break
		}

		for _, log := range logs {
			if params.Format == "json" {
				logJSON, err := s.logToJSON(log)
				if err != nil {
					continue
				}
				if !isFirst {
					writer.Write([]byte(","))
				}
				writer.Write(logJSON)
				isFirst = false
			} else {
				csvWriter := csv.NewWriter(writer)
				row := []string{
					fmt.Sprintf("%d", log.ID),
					log.SessionID,
					log.Application.Name,
					fmt.Sprintf("%d", log.ApplicationID),
					log.CaptchaType,
					log.Status,
					calculateRiskLevel(log.RiskScore),
					log.IPAddress,
					fmt.Sprintf("%.2f", log.RiskScore),
					fmt.Sprintf("%d", log.Duration),
					escapeCSVField(log.UserAgent),
					escapeCSVField(log.AnalysisResult),
					log.CreatedAt.Format("2006-01-02 15:04:05"),
				}
				csvWriter.Write(row)
				csvWriter.Flush()
			}
		}

		offset += batchSize
	}

	if params.Format == "json" {
		writer.Write([]byte(`],"record_count":` + strconv.Itoa(offset) + `}`))
	}

	return nil
}

func (s *LogService) logToJSON(log models.VerificationLog) ([]byte, error) {
	type LogExport struct {
		ID              uint      `json:"id"`
		SessionID       string    `json:"session_id"`
		ApplicationID   uint      `json:"application_id"`
		ApplicationName string    `json:"application_name"`
		CaptchaType     string    `json:"captcha_type"`
		Status          string    `json:"status"`
		RiskLevel       string    `json:"risk_level"`
		IPAddress       string    `json:"ip_address"`
		UserAgent       string    `json:"user_agent"`
		RiskScore       float64   `json:"risk_score"`
		Duration        int64     `json:"duration"`
		AnalysisResult  string    `json:"analysis_result"`
		CreatedAt       time.Time `json:"created_at"`
	}

	exp := LogExport{
		ID:              log.ID,
		SessionID:       log.SessionID,
		ApplicationID:   log.ApplicationID,
		ApplicationName: log.Application.Name,
		CaptchaType:     log.CaptchaType,
		Status:          log.Status,
		RiskLevel:       calculateRiskLevel(log.RiskScore),
		IPAddress:       log.IPAddress,
		UserAgent:       log.UserAgent,
		RiskScore:       log.RiskScore,
		Duration:        log.Duration,
		AnalysisResult:  log.AnalysisResult,
		CreatedAt:       log.CreatedAt,
	}

	return json.Marshal(exp)
}

func calculateRiskLevel(riskScore float64) string {
	if riskScore >= 80 {
		return "critical"
	} else if riskScore >= 60 {
		return "high"
	} else if riskScore >= 30 {
		return "medium"
	}
	return "low"
}

func (s *LogService) DeleteOldLogs(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	result := database.DB.Where("created_at < ?", cutoffDate).Delete(&models.VerificationLog{})
	return result.RowsAffected, result.Error
}

func (s *LogService) GetLogsBySessionID(sessionID string) ([]models.VerificationLog, error) {
	var logs []models.VerificationLog
	err := database.DB.Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *LogService) GetLogCountByStatus(status string) (int64, error) {
	var count int64
	err := database.DB.Model(&models.VerificationLog{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

func (s *LogService) GetLogCountByDateRange(start, end time.Time) (int64, error) {
	var count int64
	err := database.DB.Model(&models.VerificationLog{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&count).Error
	return count, err
}

type LogAnalytics struct {
	TotalCount     int64             `json:"total_count"`
	SuccessCount   int64             `json:"success_count"`
	FailedCount    int64             `json:"failed_count"`
	PendingCount   int64             `json:"pending_count"`
	AvgRiskScore   float64           `json:"avg_risk_score"`
	AvgDuration    float64           `json:"avg_duration"`
	SuccessRate    float64           `json:"success_rate"`
	TypeBreakdown  map[string]int64  `json:"type_breakdown"`
	HourlyTrend    []HourlyStat      `json:"hourly_trend"`
	TopApplications []AppStat        `json:"top_applications"`
	TopIPs         []IPStat           `json:"top_ips"`
	RiskDistribution map[string]int64 `json:"risk_distribution"`
}

type HourlyStat struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

type AppStat struct {
	ApplicationID   uint   `json:"application_id"`
	ApplicationName string `json:"application_name"`
	Count           int64  `json:"count"`
}

type IPStat struct {
	IPAddress string `json:"ip_address"`
	Count     int64  `json:"count"`
}

func (s *LogService) GetLogAnalytics(startDate, endDate time.Time) (*LogAnalytics, error) {
	analytics := &LogAnalytics{
		TypeBreakdown:   make(map[string]int64),
		RiskDistribution: make(map[string]int64),
	}

	query := database.DB.Model(&models.VerificationLog{}).Where("created_at >= ? AND created_at < ?", startDate, endDate)

	query.Model(&models.VerificationLog{}).Count(&analytics.TotalCount)
	query.Where("status = ?", "success").Count(&analytics.SuccessCount)
	query.Where("status = ?", "failed").Count(&analytics.FailedCount)
	query.Where("status = ?", "pending").Count(&analytics.PendingCount)

	if analytics.TotalCount > 0 {
		analytics.SuccessRate = float64(analytics.SuccessCount) / float64(analytics.TotalCount) * 100
	}

	var avgStats struct {
		AvgRisk    float64
		AvgDur     float64
	}
	err := query.Select("COALESCE(AVG(risk_score), 0) as avg_risk, COALESCE(AVG(duration), 0) as avg_dur").Row().Scan(&avgStats.AvgRisk, &avgStats.AvgDur)
	if err == nil {
		analytics.AvgRiskScore = avgStats.AvgRisk
		analytics.AvgDuration = avgStats.AvgDur
	}

	rows, err := query.Select("captcha_type, COUNT(*) as count").Group("captcha_type").Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ct string
			var cnt int64
			if rows.Scan(&ct, &cnt) == nil {
				analytics.TypeBreakdown[ct] = cnt
			}
		}
	}

	hourlyRows, err := query.Select("DATE_FORMAT(created_at, '%Y-%m-%d %H:00') as hour, COUNT(*) as count").
		Group("hour").
		Order("hour ASC").
		Limit(168).
		Rows()
	if err == nil {
		defer hourlyRows.Close()
		for hourlyRows.Next() {
			var stat HourlyStat
			if hourlyRows.Scan(&stat.Hour, &stat.Count) == nil {
				analytics.HourlyTrend = append(analytics.HourlyTrend, stat)
			}
		}
	}

	appRows, err := query.Select("verification_logs.application_id, applications.name, COUNT(*) as count").
		Joins("LEFT JOIN applications ON verification_logs.application_id = applications.id").
		Group("verification_logs.application_id").
		Order("count DESC").
		Limit(10).
		Rows()
	if err == nil {
		defer appRows.Close()
		for appRows.Next() {
			var stat AppStat
			if appRows.Scan(&stat.ApplicationID, &stat.ApplicationName, &stat.Count) == nil {
				analytics.TopApplications = append(analytics.TopApplications, stat)
			}
		}
	}

	ipRows, err := query.Select("ip_address, COUNT(*) as count").
		Where("ip_address != ''").
		Group("ip_address").
		Order("count DESC").
		Limit(20).
		Rows()
	if err == nil {
		defer ipRows.Close()
		for ipRows.Next() {
			var stat IPStat
			if ipRows.Scan(&stat.IPAddress, &stat.Count) == nil {
				analytics.TopIPs = append(analytics.TopIPs, stat)
			}
		}
	}

	riskRows, err := query.Select(
		`CASE 
			WHEN risk_score >= 80 THEN 'critical'
			WHEN risk_score >= 60 THEN 'high'
			WHEN risk_score >= 30 THEN 'medium'
			ELSE 'low'
		END as risk_level, COUNT(*) as count`,
	).Group("risk_level").Rows()
	if err == nil {
		defer riskRows.Close()
		for riskRows.Next() {
			var level string
			var count int64
			if riskRows.Scan(&level, &count) == nil {
				analytics.RiskDistribution[level] = count
			}
		}
	}

	return analytics, nil
}

func (s *LogService) GetLogStatistics() (map[string]interface{}, error) {
	ctx := context.Background()
	
	if s.redisClient != nil {
		if cached, err := s.redisClient.Get(ctx, "log:statistics:overview").Result(); err == nil {
			var stats map[string]interface{}
			if json.Unmarshal([]byte(cached), &stats) == nil {
				return stats, nil
			}
		}
	}

	stats := make(map[string]interface{})
	
	var totalCount, successCount, failedCount, pendingCount int64
	database.DB.Model(&models.VerificationLog{}).Count(&totalCount)
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "success").Count(&successCount)
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "failed").Count(&failedCount)
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "pending").Count(&pendingCount)

	stats["total_count"] = totalCount
	stats["success_count"] = successCount
	stats["failed_count"] = failedCount
	stats["pending_count"] = pendingCount
	
	if totalCount > 0 {
		stats["success_rate"] = float64(successCount) / float64(totalCount) * 100
	}

	var avgRiskScore float64
	database.DB.Model(&models.VerificationLog{}).Select("COALESCE(AVG(risk_score), 0)").Row().Scan(&avgRiskScore)
	stats["avg_risk_score"] = avgRiskScore

	typeCounts := make(map[string]int64)
	rows, _ := database.DB.Model(&models.VerificationLog{}).Select("captcha_type, COUNT(*)").Group("captcha_type").Rows()
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ct string
			var cnt int64
			if rows.Scan(&ct, &cnt) == nil {
				typeCounts[ct] = cnt
			}
		}
	}
	stats["type_counts"] = typeCounts

	if s.redisClient != nil {
		if data, err := json.Marshal(stats); err == nil {
			s.redisClient.Set(ctx, "log:statistics:overview", data, 5*time.Minute)
		}
	}

	return stats, nil
}
