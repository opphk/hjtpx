package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type AuditLogType string

const (
	AuditLogUserLogin         AuditLogType = "user_login"
	AuditLogUserLogout        AuditLogType = "user_logout"
	AuditLogUserCreate        AuditLogType = "user_create"
	AuditLogUserUpdate        AuditLogType = "user_update"
	AuditLogUserDelete        AuditLogType = "user_delete"
	AuditLogUserPassword      AuditLogType = "user_password_change"
	AuditLogUserMFAEnable     AuditLogType = "user_mfa_enable"
	AuditLogUserMFADisable    AuditLogType = "user_mfa_disable"
	AuditLogUserConsent       AuditLogType = "user_consent"
	AuditLogDataExport        AuditLogType = "data_export"
	AuditLogDataDelete        AuditLogType = "data_delete"
	AuditLogDataAnonymize     AuditLogType = "data_anonymize"
	AuditLogConfigChange      AuditLogType = "config_change"
	AuditLogAPIKeyCreate      AuditLogType = "api_key_create"
	AuditLogAPIKeyRevoke      AuditLogType = "api_key_revoke"
	AuditLogPermissionChange  AuditLogType = "permission_change"
	AuditLogRoleChange        AuditLogType = "role_change"
	AuditLogAccessDenied      AuditLogType = "access_denied"
	AuditLogSensitiveData     AuditLogType = "sensitive_data_access"
	AuditLogAdminAction       AuditLogType = "admin_action"
	AuditLogSystemStartup     AuditLogType = "system_startup"
	AuditLogSystemShutdown    AuditLogType = "system_shutdown"
	AuditLogDatabaseOperation AuditLogType = "database_operation"
	AuditLogCacheOperation    AuditLogType = "cache_operation"
	AuditLogExternalAPI       AuditLogType = "external_api_call"
	AuditLogSecurityScan      AuditLogType = "security_scan"
	AuditLogPerformanceIssue   AuditLogType = "performance_issue"
	AuditLogError             AuditLogType = "error"
	AuditLogWarning           AuditLogType = "warning"
	AuditLogInfo              AuditLogType = "info"
	AuditLogDebug             AuditLogType = "debug"
	AuditLogVerification      AuditLogType = "verification"
	AuditLogCaptchaChallenge  AuditLogType = "captcha_challenge"
	AuditLogRiskDetection     AuditLogType = "risk_detection"
	AuditLogAnomalyDetected   AuditLogType = "anomaly_detected"
)

type AuditLogLevel string

const (
	AuditLogLevelInfo     AuditLogLevel = "info"
	AuditLogLevelWarning  AuditLogLevel = "warning"
	AuditLogLevelError    AuditLogLevel = "error"
	AuditLogLevelCritical AuditLogLevel = "critical"
)

type StructuredAuditLog struct {
	ID           uint                 `json:"id"`
	Timestamp    time.Time            `json:"timestamp"`
	LogType      AuditLogType         `json:"log_type"`
	Level        AuditLogLevel        `json:"level"`
	UserID       uint                 `json:"user_id,omitempty"`
	Username     string               `json:"username,omitempty"`
	IPAddress    string               `json:"ip_address"`
	UserAgent    string               `json:"user_agent,omitempty"`
	Action       string               `json:"action"`
	ResourceType string               `json:"resource_type,omitempty"`
	ResourceID   string               `json:"resource_id,omitempty"`
	Status       string               `json:"status"`
	ErrorMessage string               `json:"error_message,omitempty"`
	Changes      map[string]struct{}  `json:"changes,omitempty"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
	Duration     int64                `json:"duration_ms,omitempty"`
	SessionID    string               `json:"session_id,omitempty"`
	ServiceName  string               `json:"service_name,omitempty"`
	TraceID      string               `json:"trace_id,omitempty"`
	SpanID       string               `json:"span_id,omitempty"`
	Environment  string               `json:"environment,omitempty"`
	Region       string               `json:"region,omitempty"`
	Version      string               `json:"version,omitempty"`
	RequestID    string               `json:"request_id,omitempty"`
	Method       string               `json:"method,omitempty"`
	Path         string               `json:"path,omitempty"`
	StatusCode   int                  `json:"status_code,omitempty"`
	ResponseTime int64                `json:"response_time_ms,omitempty"`
	ClientID     string               `json:"client_id,omitempty"`
	ApplicationID uint               `json:"application_id,omitempty"`
	RiskScore    float64              `json:"risk_score,omitempty"`
	ThreatLevel  string               `json:"threat_level,omitempty"`
}

type EnhancedLogFormat struct {
	FormatVersion string                 `json:"format_version"`
	Timestamp     time.Time              `json:"timestamp"`
	LogID         string                 `json:"log_id"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Type          string                 `json:"type"`
	Level         string                 `json:"level"`
	Category      string                 `json:"category"`
	SubCategory   string                 `json:"sub_category,omitempty"`
	Message       string                 `json:"message"`
	Description   string                 `json:"description,omitempty"`
	Source        LogSource               `json:"source"`
	Target        *LogTarget             `json:"target,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	StackTrace    string                 `json:"stack_trace,omitempty"`
	Metrics       *LogMetrics            `json:"metrics,omitempty"`
}

type AuditLogQueryParams struct {
	Page          int
	PageSize      int
	LogTypes      []AuditLogType
	Levels        []AuditLogLevel
	UserID        uint
	Username      string
	IPAddress     string
	StartDate     time.Time
	EndDate       time.Time
	Status        string
	ResourceType  string
	ResourceID    string
	SessionID     string
	Action        string
	MinDuration   int64
	MaxDuration   int64
	SearchText    string
	SortBy        string
	SortOrder     string
}

type AuditLogListResult struct {
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
	Logs       []StructuredAuditLog    `json:"logs"`
}

type LogSource struct {
	Service     string `json:"service"`
	Host        string `json:"host"`
	IPAddress   string `json:"ip_address"`
	ContainerID string `json:"container_id,omitempty"`
	PodName     string `json:"pod_name,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ProcessID   int    `json:"process_id,omitempty"`
	ThreadID    int    `json:"thread_id,omitempty"`
}

type LogTarget struct {
	Type         string `json:"type"`
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Owner        string `json:"owner,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`
	Database     string `json:"database,omitempty"`
	Table        string `json:"table,omitempty"`
	Operation    string `json:"operation,omitempty"`
}

type LogMetrics struct {
	Duration     int64                `json:"duration_ms,omitempty"`
	MemoryUsage  int64                `json:"memory_usage_bytes,omitempty"`
	CPUUsage     float64              `json:"cpu_usage_percent,omitempty"`
	NetworkIn    int64                `json:"network_in_bytes,omitempty"`
	NetworkOut   int64                `json:"network_out_bytes,omitempty"`
	DiskIO       int64                `json:"disk_io_bytes,omitempty"`
	CacheHitRate float64              `json:"cache_hit_rate,omitempty"`
	QueryCount   int                  `json:"query_count,omitempty"`
	ErrorCount   int                  `json:"error_count,omitempty"`
	WarningCount int                  `json:"warning_count,omitempty"`
}

type LogSearchParams struct {
	Page              int
	PageSize          int
	Query             string
	FullTextSearch    bool
	Filters           map[string]interface{}
	DateRange         *DateRange
	LogLevels         []string
	LogCategories     []string
	Services          []string
	Hosts             []string
	Tags              []string
	SortBy            string
	SortOrder         string
	GroupBy           string
	AggregateFunction string
	TimeAggregation   string
}

type DateRange struct {
	Start time.Time
	End   time.Time
}

type SearchResult struct {
	Total            int64                `json:"total"`
	Page             int                  `json:"page"`
	PageSize         int                  `json:"page_size"`
	TotalPages       int                  `json:"total_pages"`
	Results          []EnhancedLogFormat  `json:"results"`
	Aggregations     map[string]interface{} `json:"aggregations,omitempty"`
	Highlights       []SearchHighlight   `json:"highlights,omitempty"`
	SearchTimeMs     int64                `json:"search_time_ms"`
	TotalMatched     int64                `json:"total_matched"`
}

type SearchHighlight struct {
	LogID    string            `json:"log_id"`
	Field    string            `json:"field"`
	Fragment string            `json:"fragment"`
	Matches  []HighlightMatch `json:"matches,omitempty"`
}

type HighlightMatch struct {
	Start  int
	End    int
	Text   string
}

type LogService struct{}

func NewLogService() *LogService {
	return &LogService{}
}

func (s *LogService) CreateAuditLog(log *models.AuditLog) error {
	return database.DB.Create(log).Error
}

func (s *LogService) CreateStructuredAuditLog(log *StructuredAuditLog) (*models.AuditLog, error) {
	auditLog := &models.AuditLog{
		LogType:      string(log.LogType),
		Level:        string(log.Level),
		UserID:       log.UserID,
		Username:     log.Username,
		IPAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Status:       log.Status,
		ErrorMessage: log.ErrorMessage,
		Duration:     log.Duration,
		SessionID:    log.SessionID,
	}

	if log.Changes != nil {
		changesJSON, _ := json.Marshal(log.Changes)
		auditLog.Changes = string(changesJSON)
	}

	if log.Metadata != nil {
		metadataJSON, _ := json.Marshal(log.Metadata)
		auditLog.Metadata = string(metadataJSON)
	}

	if err := database.DB.Create(auditLog).Error; err != nil {
		return nil, err
	}

	return auditLog, nil
}

func (s *LogService) LogUserAction(userID uint, username, action string, ipAddress, userAgent string, level AuditLogLevel, status string) error {
	log := &StructuredAuditLog{
		Timestamp: time.Now(),
		LogType:   AuditLogType(action),
		Level:     level,
		UserID:    userID,
		Username:  username,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Action:    action,
		Status:    status,
	}

	_, err := s.CreateStructuredAuditLog(log)
	return err
}

func (s *LogService) LogDataAccess(userID uint, username string, resourceType, resourceID string, ipAddress, userAgent string, duration int64) error {
	log := &StructuredAuditLog{
		Timestamp:    time.Now(),
		LogType:      AuditLogSensitiveData,
		Level:        AuditLogLevelInfo,
		UserID:       userID,
		Username:     username,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Action:       "access",
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       "success",
		Duration:     duration,
	}

	_, err := s.CreateStructuredAuditLog(log)
	return err
}

func (s *LogService) LogSecurityEvent(logType AuditLogType, level AuditLogLevel, userID uint, username string, action, status, errorMsg, ipAddress string) error {
	log := &StructuredAuditLog{
		Timestamp:   time.Now(),
		LogType:     logType,
		Level:       level,
		UserID:      userID,
		Username:    username,
		IPAddress:   ipAddress,
		Action:      action,
		Status:      status,
		ErrorMessage: errorMsg,
	}

	_, err := s.CreateStructuredAuditLog(log)
	return err
}

func (s *LogService) QueryAuditLogs(params AuditLogQueryParams) (*AuditLogListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	query := database.DB.Model(&models.AuditLog{})

	if len(params.LogTypes) > 0 {
		logTypeStrings := make([]string, len(params.LogTypes))
		for i, lt := range params.LogTypes {
			logTypeStrings[i] = string(lt)
		}
		query = query.Where("log_type IN ?", logTypeStrings)
	}

	if len(params.Levels) > 0 {
		levelStrings := make([]string, len(params.Levels))
		for i, l := range params.Levels {
			levelStrings[i] = string(l)
		}
		query = query.Where("level IN ?", levelStrings)
	}

	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}

	if params.Username != "" {
		query = query.Where("username LIKE ?", "%"+params.Username+"%")
	}

	if params.IPAddress != "" {
		query = query.Where("ip_address LIKE ?", "%"+params.IPAddress+"%")
	}

	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}

	if !params.EndDate.IsZero() {
		query = query.Where("created_at < ?", params.EndDate)
	}

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if params.ResourceType != "" {
		query = query.Where("resource_type = ?", params.ResourceType)
	}

	if params.ResourceID != "" {
		query = query.Where("resource_id = ?", params.ResourceID)
	}

	if params.SessionID != "" {
		query = query.Where("session_id = ?", params.SessionID)
	}

	if params.Action != "" {
		query = query.Where("action LIKE ?", "%"+params.Action+"%")
	}

	if params.MinDuration > 0 {
		query = query.Where("duration >= ?", params.MinDuration)
	}

	if params.MaxDuration > 0 {
		query = query.Where("duration <= ?", params.MaxDuration)
	}

	if params.SearchText != "" {
		searchPattern := "%" + params.SearchText + "%"
		query = query.Where("action LIKE ? OR username LIKE ? OR error_message LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	sortBy := "created_at"
	sortOrder := "DESC"
	if params.SortBy != "" {
		sortBy = params.SortBy
	}
	if params.SortOrder != "" {
		sortOrder = params.SortOrder
	}

	var logs []models.AuditLog
	offset := (params.Page - 1) * params.PageSize
	if err := query.
		Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Offset(offset).
		Limit(params.PageSize).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	structuredLogs := make([]StructuredAuditLog, len(logs))
	for i, log := range logs {
		structuredLogs[i] = s.convertToStructuredLog(log)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &AuditLogListResult{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Logs:       structuredLogs,
	}, nil
}

func (s *LogService) convertToStructuredLog(log models.AuditLog) StructuredAuditLog {
	structuredLog := StructuredAuditLog{
		ID:           log.ID,
		Timestamp:    log.CreatedAt,
		LogType:      AuditLogType(log.LogType),
		Level:        AuditLogLevel(log.Level),
		UserID:       log.UserID,
		Username:     log.Username,
		IPAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Status:       log.Status,
		ErrorMessage: log.ErrorMessage,
		Duration:     log.Duration,
		SessionID:    log.SessionID,
	}

	if log.Changes != "" {
		var changes map[string]struct{}
		if err := json.Unmarshal([]byte(log.Changes), &changes); err == nil {
			structuredLog.Changes = changes
		}
	}

	if log.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(log.Metadata), &metadata); err == nil {
			structuredLog.Metadata = metadata
		}
	}

	return structuredLog
}

func (s *LogService) ExportAuditLogsJSON(params AuditLogQueryParams) ([]byte, error) {
	result, err := s.QueryAuditLogs(params)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(result, "", "  ")
}

func (s *LogService) GetAuditLogByID(id uint) (*StructuredAuditLog, error) {
	var log models.AuditLog
	if err := database.DB.First(&log, id).Error; err != nil {
		return nil, err
	}

	structuredLog := s.convertToStructuredLog(log)
	return &structuredLog, nil
}

func (s *LogService) GetAuditStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	query := database.DB.Model(&models.AuditLog{})
	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at < ?", endDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var byType map[string]int64
	if err := query.Select("log_type, count(*) as count").Group("log_type").Scan(&byType).Error; err != nil {
		return nil, err
	}

	var byLevel map[string]int64
	if err := query.Select("level, count(*) as count").Group("level").Scan(&byLevel).Error; err != nil {
		return nil, err
	}

	var byStatus map[string]int64
	if err := query.Select("status, count(*) as count").Group("status").Scan(&byStatus).Error; err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_logs":     total,
		"by_type":        byType,
		"by_level":       byLevel,
		"by_status":      byStatus,
		"start_date":     startDate,
		"end_date":       endDate,
		"generated_at":   time.Now(),
	}

	return stats, nil
}

func (s *LogService) GetUserActivitySummary(userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	query := database.DB.Model(&models.AuditLog{}).Where("user_id = ?", userID)

	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at < ?", endDate)
	}

	var totalActions int64
	if err := query.Count(&totalActions).Error; err != nil {
		return nil, err
	}

	var byType map[string]int64
	if err := query.Select("log_type, count(*) as count").Group("log_type").Scan(&byType).Error; err != nil {
		return nil, err
	}

	var uniqueIPs int64
	if err := query.Select("COUNT(DISTINCT ip_address)").Scan(&uniqueIPs).Error; err != nil {
		return nil, err
	}

	avgDurationQuery := database.DB.Model(&models.AuditLog{}).
		Where("user_id = ? AND duration > 0", userID)
	if !startDate.IsZero() {
		avgDurationQuery = avgDurationQuery.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		avgDurationQuery = avgDurationQuery.Where("created_at < ?", endDate)
	}

	var avgDuration float64
	avgDurationQuery.Select("AVG(duration)").Scan(&avgDuration)

	summary := map[string]interface{}{
		"user_id":          userID,
		"total_actions":    totalActions,
		"action_by_type":   byType,
		"unique_ips":       uniqueIPs,
		"avg_action_duration_ms": avgDuration,
		"start_date":       startDate,
		"end_date":         endDate,
	}

	return summary, nil
}

func (s *LogService) DetectAnomalousActivity(userID uint, threshold float64) ([]StructuredAuditLog, error) {
	var logs []models.AuditLog
	if err := database.DB.Where("user_id = ? AND created_at >= ?",
		userID, time.Now().Add(-24*time.Hour)).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	anomalies := make([]StructuredAuditLog, 0)

	ipCounts := make(map[string]int)
	for _, log := range logs {
		ipCounts[log.IPAddress]++
	}

	for _, log := range logs {
		if float64(ipCounts[log.IPAddress]) > threshold*float64(len(logs)) && ipCounts[log.IPAddress] > 10 {
			structuredLog := s.convertToStructuredLog(log)
			structuredLog.Metadata = map[string]string{
				"anomaly_type": "high_ip_concentration",
				"ip_count":     fmt.Sprintf("%d", ipCounts[log.IPAddress]),
				"threshold":   fmt.Sprintf("%.2f", threshold),
			}
			anomalies = append(anomalies, structuredLog)
		}

		if log.Level == string(AuditLogLevelError) || log.Level == string(AuditLogLevelCritical) {
			structuredLog := s.convertToStructuredLog(log)
			anomalies = append(anomalies, structuredLog)
		}
	}

	return anomalies, nil
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

func (s *LogService) CreateEnhancedLog(log *EnhancedLogFormat) error {
	if log.LogID == "" {
		log.LogID = fmt.Sprintf("log-%d-%s", time.Now().UnixNano(), generateRandomString(8))
	}
	if log.FormatVersion == "" {
		log.FormatVersion = "1.0"
	}

	auditLog := &models.AuditLog{
		LogType:      AuditLogType(log.Type),
		Level:        AuditLogLevel(log.Level),
		Action:       log.Message,
		Status:       "success",
		Metadata:     fmt.Sprintf(`{"category":"%s","sub_category":"%s","tags":"%v"}`, log.Category, log.SubCategory, log.Tags),
	}

	if log.Source.Service != "" {
		auditLog.Username = log.Source.Service
	}
	if log.Source.IPAddress != "" {
		auditLog.IPAddress = log.Source.IPAddress
	}

	return database.DB.Create(auditLog).Error
}

func (s *LogService) SearchLogs(params LogSearchParams) (*SearchResult, error) {
	startTime := time.Now()
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	query := database.DB.Model(&models.AuditLog{})

	if params.Query != "" {
		if params.FullTextSearch {
			searchPattern := "%" + params.Query + "%"
			query = query.Where(
				"action LIKE ? OR username LIKE ? OR error_message LIKE ? OR metadata LIKE ?",
				searchPattern, searchPattern, searchPattern, searchPattern,
			)
		} else {
			query = query.Where(
				"action LIKE ? OR username LIKE ?",
				"%"+params.Query+"%", "%"+params.Query+"%",
			)
		}
	}

	if len(params.LogLevels) > 0 {
		query = query.Where("level IN ?", params.LogLevels)
	}

	if len(params.LogCategories) > 0 {
		logTypeStrings := make([]string, len(params.LogCategories))
		for i, cat := range params.LogCategories {
			logTypeStrings[i] = cat
		}
		query = query.Where("log_type IN ?", logTypeStrings)
	}

	if params.DateRange != nil {
		if !params.DateRange.Start.IsZero() {
			query = query.Where("created_at >= ?", params.DateRange.Start)
		}
		if !params.DateRange.End.IsZero() {
			query = query.Where("created_at < ?", params.DateRange.End)
		}
	}

	if params.Filters != nil {
		if userID, ok := params.Filters["user_id"].(uint); ok && userID > 0 {
			query = query.Where("user_id = ?", userID)
		}
		if username, ok := params.Filters["username"].(string); ok && username != "" {
			query = query.Where("username LIKE ?", "%"+username+"%")
		}
		if ipAddress, ok := params.Filters["ip_address"].(string); ok && ipAddress != "" {
			query = query.Where("ip_address LIKE ?", "%"+ipAddress+"%")
		}
		if status, ok := params.Filters["status"].(string); ok && status != "" {
			query = query.Where("status = ?", status)
		}
		if resourceType, ok := params.Filters["resource_type"].(string); ok && resourceType != "" {
			query = query.Where("resource_type = ?", resourceType)
		}
		if resourceID, ok := params.Filters["resource_id"].(string); ok && resourceID != "" {
			query = query.Where("resource_id = ?", resourceID)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	sortBy := "created_at"
	sortOrder := "DESC"
	if params.SortBy != "" {
		sortBy = params.SortBy
	}
	if params.SortOrder != "" {
		sortOrder = params.SortOrder
	}

	var logs []models.AuditLog
	offset := (params.Page - 1) * params.PageSize
	if err := query.
		Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Offset(offset).
		Limit(params.PageSize).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	results := make([]EnhancedLogFormat, len(logs))
	for i, log := range logs {
		results[i] = s.convertToEnhancedFormat(log)
	}

	searchTime := time.Since(startTime).Milliseconds()
	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &SearchResult{
		Total:        total,
		Page:         params.Page,
		PageSize:     params.PageSize,
		TotalPages:   totalPages,
		Results:      results,
		SearchTimeMs: searchTime,
		TotalMatched: total,
	}, nil
}

func (s *LogService) convertToEnhancedFormat(log models.AuditLog) EnhancedLogFormat {
	return EnhancedLogFormat{
		FormatVersion: "1.0",
		Timestamp:    log.CreatedAt,
		LogID:        fmt.Sprintf("log-%d", log.ID),
		Type:         log.LogType,
		Level:        log.Level,
		Category:     log.LogType,
		Message:      log.Action,
		Description: log.ErrorMessage,
		Source: LogSource{
			Service:   log.Username,
			IPAddress: log.IPAddress,
		},
		Context: map[string]interface{}{
			"user_id":       log.UserID,
			"user_agent":    log.UserAgent,
			"resource_type": log.ResourceType,
			"resource_id":   log.ResourceID,
			"session_id":    log.SessionID,
		},
	}
}

func (s *LogService) GetLogAggregations(groupBy string, startDate, endDate time.Time) (map[string]interface{}, error) {
	query := database.DB.Model(&models.AuditLog{})

	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at < ?", endDate)
	}

	aggregations := make(map[string]interface{})

	switch groupBy {
	case "level":
		var results []struct {
			Level string
			Count int64
		}
		if err := query.Select("level, COUNT(*) as count").Group("level").Scan(&results).Error; err != nil {
			return nil, err
		}
		levelCounts := make(map[string]int64)
		for _, r := range results {
			levelCounts[r.Level] = r.Count
		}
		aggregations["by_level"] = levelCounts

	case "log_type":
		var results []struct {
			LogType string
			Count   int64
		}
		if err := query.Select("log_type, COUNT(*) as count").Group("log_type").Scan(&results).Error; err != nil {
			return nil, err
		}
		typeCounts := make(map[string]int64)
		for _, r := range results {
			typeCounts[r.LogType] = r.Count
		}
		aggregations["by_type"] = typeCounts

	case "hour":
		var results []struct {
			Hour  int
			Count int64
		}
		if err := query.Select("EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count").
			Group("EXTRACT(HOUR FROM created_at)").
			Order("hour").
			Scan(&results).Error; err != nil {
			return nil, err
		}
		hourlyCounts := make(map[int]int64)
		for _, r := range results {
			hourlyCounts[r.Hour] = r.Count
		}
		aggregations["by_hour"] = hourlyCounts

	case "date":
		var results []struct {
			Date  time.Time
			Count int64
		}
		if err := query.Select("DATE(created_at) as date, COUNT(*) as count").
			Group("DATE(created_at)").
			Order("date DESC").
			Scan(&results).Error; err != nil {
			return nil, err
		}
		dailyCounts := make(map[string]int64)
		for _, r := range results {
			dailyCounts[r.Date.Format("2006-01-02")] = r.Count
		}
		aggregations["by_date"] = dailyCounts

	case "status":
		var results []struct {
			Status string
			Count  int64
		}
		if err := query.Select("status, COUNT(*) as count").Group("status").Scan(&results).Error; err != nil {
			return nil, err
		}
		statusCounts := make(map[string]int64)
		for _, r := range results {
			statusCounts[r.Status] = r.Count
		}
		aggregations["by_status"] = statusCounts

	case "user":
		var results []struct {
			Username string
			Count    int64
		}
		if err := query.Select("username, COUNT(*) as count").
			Where("username != ''").
			Group("username").
			Order("count DESC").
			Limit(10).
			Scan(&results).Error; err != nil {
			return nil, err
		}
		userCounts := make(map[string]int64)
		for _, r := range results {
			userCounts[r.Username] = r.Count
		}
		aggregations["by_user"] = userCounts

	default:
		var total int64
		if err := query.Count(&total).Error; err != nil {
			return nil, err
		}
		aggregations["total"] = total
	}

	return aggregations, nil
}

func (s *LogService) CreateStructuredLogWithMetrics(log *StructuredAuditLog, metrics *LogMetrics) error {
	auditLog := &models.AuditLog{
		LogType:      string(log.LogType),
		Level:        string(log.Level),
		UserID:       log.UserID,
		Username:     log.Username,
		IPAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Status:       log.Status,
		ErrorMessage: log.ErrorMessage,
		Duration:     log.Duration,
		SessionID:    log.SessionID,
	}

	if metrics != nil {
		metadata := make(map[string]interface{})
		if metrics.Duration > 0 {
			metadata["duration_ms"] = metrics.Duration
		}
		if metrics.MemoryUsage > 0 {
			metadata["memory_usage_bytes"] = metrics.MemoryUsage
		}
		if metrics.CPUUsage > 0 {
			metadata["cpu_usage_percent"] = metrics.CPUUsage
		}
		if metrics.ErrorCount > 0 {
			metadata["error_count"] = metrics.ErrorCount
		}
		if metrics.WarningCount > 0 {
			metadata["warning_count"] = metrics.WarningCount
		}
		metadataJSON, _ := json.Marshal(metadata)
		auditLog.Metadata = string(metadataJSON)
	}

	return database.DB.Create(auditLog).Error
}

func (s *LogService) GetLogsByTraceID(traceID string) ([]StructuredAuditLog, error) {
	var logs []models.AuditLog
	err := database.DB.Where("session_id = ?", traceID).
		Order("created_at ASC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	structuredLogs := make([]StructuredAuditLog, len(logs))
	for i, log := range logs {
		structuredLogs[i] = s.convertToStructuredLog(log)
	}
	return structuredLogs, nil
}

func (s *LogService) GetRecentLogs(limit int) ([]StructuredAuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var logs []models.AuditLog
	err := database.DB.Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	structuredLogs := make([]StructuredAuditLog, len(logs))
	for i, log := range logs {
		structuredLogs[i] = s.convertToStructuredLog(log)
	}
	return structuredLogs, nil
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
