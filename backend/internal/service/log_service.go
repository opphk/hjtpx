package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type LogService struct{}

func NewLogService() *LogService {
	return &LogService{}
}

func (s *LogService) CreateVerificationLog(log *models.VerificationLog) error {
	return database.DB.Create(log).Error
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
}

type LogListResult struct {
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
	Logs       []models.VerificationLog
}

func (s *LogService) QueryLogs(params LogQueryParams) (*LogListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
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

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var logs []models.VerificationLog
	offset := (params.Page - 1) * params.PageSize
	if err := query.Preload("Application").
		Order("created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &LogListResult{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Logs:       logs,
	}, nil
}

type LogExportParams struct {
	ApplicationID uint
	Status        string
	CaptchaType   string
	StartDate     time.Time
	EndDate       time.Time
	Format        string
}

func (s *LogService) ExportLogs(params LogExportParams) ([]byte, error) {
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

	switch params.Format {
	case "csv":
		return s.exportToCSV(logs)
	default:
		return s.exportToCSV(logs)
	}
}

func (s *LogService) exportToCSV(logs []models.VerificationLog) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"ID",
		"Session ID",
		"Application Name",
		"Captcha Type",
		"Status",
		"IP Address",
		"Risk Score",
		"Duration (ms)",
		"Created At",
		"User Agent",
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
			log.CaptchaType,
			log.Status,
			log.IPAddress,
			fmt.Sprintf("%.2f", log.RiskScore),
			fmt.Sprintf("%d", log.Duration),
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.UserAgent,
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

type AuditLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id"`
	Username     string    `json:"username"`
	Action       string    `json:"action" binding:"required"`
	Resource     string    `json:"resource" binding:"required"`
	ResourceID   string    `json:"resource_id"`
	Details      string    `json:"details"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *LogService) CreateAuditLog(log *AuditLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return database.DB.Create(log).Error
}

type AuditLogQueryParams struct {
	Page        int
	PageSize    int
	UserID      uint
	Username    string
	Action      string
	Resource    string
	ResourceID  string
	Status      string
	StartDate   time.Time
	EndDate     time.Time
	IPAddress   string
	Keyword     string
}

type AuditLogListResult struct {
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
	Logs       []AuditLog
}

func (s *LogService) QueryAuditLogs(params AuditLogQueryParams) (*AuditLogListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	query := database.DB.Model(&AuditLog{})

	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Username != "" {
		query = query.Where("username LIKE ?", "%"+params.Username+"%")
	}
	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}
	if params.Resource != "" {
		query = query.Where("resource = ?", params.Resource)
	}
	if params.ResourceID != "" {
		query = query.Where("resource_id = ?", params.ResourceID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("created_at < ?", params.EndDate)
	}
	if params.IPAddress != "" {
		query = query.Where("ip_address LIKE ?", "%"+params.IPAddress+"%")
	}
	if params.Keyword != "" {
		query = query.Where("details LIKE ? OR error_message LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var logs []AuditLog
	offset := (params.Page - 1) * params.PageSize
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &AuditLogListResult{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Logs:       logs,
	}, nil
}

type AuditLogStats struct {
	TotalLogs        int64            `json:"total_logs"`
	ActionCounts     map[string]int64 `json:"action_counts"`
	ResourceCounts   map[string]int64 `json:"resource_counts"`
	StatusCounts     map[string]int64 `json:"status_counts"`
	TopUsers         []UserActionCount `json:"top_users"`
	TopResources     []ResourceAccessCount `json:"top_resources"`
	FailuresCount    int64            `json:"failures_count"`
	SuccessRate      float64          `json:"success_rate"`
	DailyTrend       []DailyCount     `json:"daily_trend"`
}

type UserActionCount struct {
	Username string `json:"username"`
	Count    int64  `json:"count"`
}

type ResourceAccessCount struct {
	Resource string `json:"resource"`
	Count    int64  `json:"count"`
}

type DailyCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

func (s *LogService) GetAuditLogStats(startDate, endDate time.Time) (*AuditLogStats, error) {
	stats := &AuditLogStats{
		ActionCounts:   make(map[string]int64),
		ResourceCounts: make(map[string]int64),
		StatusCounts:   make(map[string]int64),
	}

	query := database.DB.Model(&AuditLog{})
	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at < ?", endDate)
	}

	if err := query.Count(&stats.TotalLogs).Error; err != nil {
		return nil, err
	}

	actionRows, _ := s.getCountsByField("action", startDate, endDate)
	for _, row := range actionRows {
		stats.ActionCounts[row.Key] = row.Count
	}

	resourceRows, _ := s.getCountsByField("resource", startDate, endDate)
	for _, row := range resourceRows {
		stats.ResourceCounts[row.Key] = row.Count
	}

	statusRows, _ := s.getCountsByField("status", startDate, endDate)
	for _, row := range statusRows {
		stats.StatusCounts[row.Key] = row.Count
	}

	var topUsers []UserActionCount
	database.DB.Model(&AuditLog{}).
		Select("username, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Group("username").
		Order("count DESC").
		Limit(10).
		Scan(&topUsers)
	stats.TopUsers = topUsers

	var topResources []ResourceAccessCount
	database.DB.Model(&AuditLog{}).
		Select("resource, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Group("resource").
		Order("count DESC").
		Limit(10).
		Scan(&topResources)
	stats.TopResources = topResources

	database.DB.Model(&AuditLog{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "failure", startDate, endDate).
		Count(&stats.FailuresCount)

	if stats.TotalLogs > 0 {
		successCount := stats.TotalLogs - stats.FailuresCount
		stats.SuccessRate = float64(successCount) / float64(stats.TotalLogs) * 100
	}

	stats.DailyTrend = s.getDailyTrend(startDate, endDate)

	return stats, nil
}

type CountResult struct {
	Key   string
	Count int64
}

func (s *LogService) getCountsByField(field string, startDate, endDate time.Time) ([]CountResult, error) {
	var results []CountResult

	query := database.DB.Model(&AuditLog{}).
		Select(field+" as key, COUNT(*) as count").
		Group(field)

	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at < ?", endDate)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (s *LogService) getDailyTrend(startDate, endDate time.Time) []DailyCount {
	var results []DailyCount

	query := database.DB.Model(&AuditLog{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Group("DATE(created_at)").
		Order("date ASC")

	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at < ?", endDate)
	}

	query.Scan(&results)

	return results
}

func (s *LogService) ExportAuditLogs(params AuditLogQueryParams, format string) ([]byte, error) {
	result, err := s.QueryAuditLogs(params)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "csv":
		return s.exportAuditLogsToCSV(result.Logs)
	case "json":
		return s.exportAuditLogsToJSON(result.Logs)
	default:
		return s.exportAuditLogsToCSV(result.Logs)
	}
}

func (s *LogService) exportAuditLogsToCSV(logs []AuditLog) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"ID",
		"User ID",
		"Username",
		"Action",
		"Resource",
		"Resource ID",
		"Status",
		"IP Address",
		"Created At",
		"Details",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, log := range logs {
		row := []string{
			fmt.Sprintf("%d", log.ID),
			fmt.Sprintf("%d", log.UserID),
			log.Username,
			log.Action,
			log.Resource,
			log.ResourceID,
			log.Status,
			log.IPAddress,
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.Details,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), nil
}

func (s *LogService) exportAuditLogsToJSON(logs []AuditLog) ([]byte, error) {
	data := map[string]interface{}{
		"exported_at": time.Now(),
		"total":      len(logs),
		"logs":       logs,
	}
	return json.Marshal(data)
}

func (s *LogService) DeleteAuditLogs(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	result := database.DB.Where("created_at < ?", cutoffDate).Delete(&AuditLog{})
	return result.RowsAffected, result.Error
}

func (s *LogService) GetAuditLogByID(id uint) (*AuditLog, error) {
	var log AuditLog
	if err := database.DB.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func (s *LogService) GetUserActivitySummary(userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	var totalActions int64
	var failedActions int64

	database.DB.Model(&AuditLog{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startDate, endDate).
		Count(&totalActions)

	database.DB.Model(&AuditLog{}).
		Where("user_id = ? AND status = ? AND created_at >= ? AND created_at < ?", userID, "failure", startDate, endDate).
		Count(&failedActions)

	var actionsByType []CountResult
	database.DB.Model(&AuditLog{}).
		Select("action as key, COUNT(*) as count").
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startDate, endDate).
		Group("action").
		Order("count DESC").
		Scan(&actionsByType)

	var actionsByResource []CountResult
	database.DB.Model(&AuditLog{}).
		Select("resource as key, COUNT(*) as count").
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startDate, endDate).
		Group("resource").
		Order("count DESC").
		Scan(&actionsByResource)

	return map[string]interface{}{
		"user_id":           userID,
		"total_actions":     totalActions,
		"failed_actions":    failedActions,
		"success_rate":      func() float64 {
			if totalActions == 0 {
				return 0
			}
			return float64(totalActions-failedActions) / float64(totalActions) * 100
		}(),
		"actions_by_type":    actionsByType,
		"actions_by_resource": actionsByResource,
	}, nil
}

func (s *LogService) GetSecurityAuditReport(startDate, endDate time.Time) (map[string]interface{}, error) {
	var suspiciousActivities []AuditLog

	database.DB.Model(&AuditLog{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "failure", startDate, endDate).
		Order("created_at DESC").
		Limit(100).
		Find(&suspiciousActivities)

	failurePatterns := s.analyzeFailurePatterns(suspiciousActivities)

	var privilegeEscalationAttempts int64
	database.DB.Model(&AuditLog{}).
		Where("action IN ? AND created_at >= ? AND created_at < ?",
			[]string{"role_change", "permission_grant", "permission_revoke", "admin_access"},
			startDate, endDate).
		Count(&privilegeEscalationAttempts)

	var unauthorizedAccessAttempts int64
	database.DB.Model(&AuditLog{}).
		Where("status = ? AND action LIKE ? AND created_at >= ? AND created_at < ?",
			"failure", "%access%", startDate, endDate).
		Count(&unauthorizedAccessAttempts)

	return map[string]interface{}{
		"report_period": map[string]interface{}{
			"start": startDate,
			"end":   endDate,
		},
		"suspicious_activities":      suspiciousActivities,
		"failure_patterns":           failurePatterns,
		"privilege_escalation_attempts": privilegeEscalationAttempts,
		"unauthorized_access_attempts": unauthorizedAccessAttempts,
		"generated_at":               time.Now(),
	}, nil
}

func (s *LogService) analyzeFailurePatterns(activities []AuditLog) map[string]interface{} {
	patterns := make(map[string]interface{})

	ipCounts := make(map[string]int)
	userCounts := make(map[string]int)

	for _, activity := range activities {
		ipCounts[activity.IPAddress]++
		userCounts[activity.Username]++
	}

	var repeatedFailures []map[string]interface{}
	for ip, count := range ipCounts {
		if count > 5 {
			repeatedFailures = append(repeatedFailures, map[string]interface{}{
				"type":  "ip",
				"value": ip,
				"count": count,
			})
		}
	}

	for username, count := range userCounts {
		if count > 10 {
			repeatedFailures = append(repeatedFailures, map[string]interface{}{
				"type":  "user",
				"value": username,
				"count": count,
			})
		}
	}

	patterns["repeated_failures"] = repeatedFailures
	patterns["total_failed_attempts"] = len(activities)

	return patterns
}

func RecordUserAction(userID uint, username, action, resource, resourceID, details, ipAddress, userAgent string) {
	log := &AuditLog{
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Status:     "success",
	}

	service := NewLogService()
	service.CreateAuditLog(log)
}

func RecordUserActionWithError(userID uint, username, action, resource, resourceID, details, ipAddress, userAgent, errorMsg string) {
	log := &AuditLog{
		UserID:       userID,
		Username:     username,
		Action:       action,
		Resource:     resource,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       "failure",
		ErrorMessage: errorMsg,
	}

	service := NewLogService()
	service.CreateAuditLog(log)
}
