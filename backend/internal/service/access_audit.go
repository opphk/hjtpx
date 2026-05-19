package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type AccessEventType string

const (
	AccessLogin           AccessEventType = "login"
	AccessLogout          AccessEventType = "logout"
	AccessView           AccessEventType = "view"
	AccessCreate         AccessEventType = "create"
	AccessUpdate         AccessEventType = "update"
	AccessDelete         AccessEventType = "delete"
	AccessExport         AccessEventType = "export"
	AccessImport         AccessEventType = "import"
	AccessAdminAction    AccessEventType = "admin_action"
	AccessSensitiveData  AccessEventType = "sensitive_data_access"
	AccessConfigChange   AccessEventType = "config_change"
	AccessPermissionGrant AccessEventType = "permission_grant"
	AccessPermissionRevoke AccessEventType = "permission_revoke"
	AccessRoleAssign     AccessEventType = "role_assign"
	AccessRoleRemove     AccessEventType = "role_remove"
)

type AccessSeverity string

const (
	SeverityInfo     AccessSeverity = "info"
	SeverityLow      AccessSeverity = "low"
	SeverityMedium   AccessSeverity = "medium"
	SeverityHigh     AccessSeverity = "high"
	SeverityCritical AccessSeverity = "critical"
)

type AccessAuditLog struct {
	ID            uint                 `json:"id"`
	Timestamp     time.Time            `json:"timestamp"`
	EventType     AccessEventType      `json:"event_type"`
	Severity      AccessSeverity       `json:"severity"`
	UserID        uint                 `json:"user_id"`
	Username      string               `json:"username"`
	IPAddress     string               `json:"ip_address"`
	UserAgent     string               `json:"user_agent"`
	ResourceType  string               `json:"resource_type"`
	ResourceID    string               `json:"resource_id"`
	Action        string               `json:"action"`
	Status        string               `json:"status"`
	ErrorMessage  string               `json:"error_message,omitempty"`
	SessionID     string               `json:"session_id,omitempty"`
	RequestPath   string               `json:"request_path"`
	RequestMethod string               `json:"request_method"`
	ResponseTime  int64               `json:"response_time_ms"`
	Metadata      map[string]string    `json:"metadata,omitempty"`
	GeoLocation   string               `json:"geo_location,omitempty"`
	RiskScore     float64             `json:"risk_score"`
	Tags          []string             `json:"tags,omitempty"`
}

type PermissionChange struct {
	ChangeType     string            `json:"change_type"`
	UserID         uint              `json:"user_id"`
	TargetUserID   uint              `json:"target_user_id,omitempty"`
	Permission     string            `json:"permission"`
	OldValue       string            `json:"old_value,omitempty"`
	NewValue       string            `json:"new_value,omitempty"`
	ChangedBy      uint              `json:"changed_by"`
	Reason         string            `json:"reason,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
	ApprovedBy     uint              `json:"approved_by,omitempty"`
	Status         string            `json:"status"`
}

type SensitiveOperation struct {
	OperationType string            `json:"operation_type"`
	Description   string            `json:"description"`
	DataType      string            `json:"data_type"`
	RiskLevel     AccessSeverity   `json:"risk_level"`
	RequiresApproval bool           `json:"requires_approval"`
	ApprovalRoles []string          `json:"approval_roles,omitempty"`
	AuditRequired bool             `json:"audit_required"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type AbnormalAccessPattern struct {
	PatternType     string    `json:"pattern_type"`
	Description     string    `json:"description"`
	UserID          uint     `json:"user_id"`
	IPAddress       string    `json:"ip_address"`
	FirstOccurrence time.Time `json:"first_occurrence"`
	LastOccurrence  time.Time `json:"last_occurrence"`
	OccurrenceCount int       `json:"occurrence_count"`
	Severity        AccessSeverity `json:"severity"`
	Recommendation  string    `json:"recommendation"`
	RiskScore       float64   `json:"risk_score"`
}

type AccessAuditService struct {
	accessLogs             []*AccessAuditLog
	permissionChanges      []*PermissionChange
	sensitiveOperations    []*SensitiveOperation
	abnormalPatterns       []*AbnormalAccessPattern
	mu                     sync.RWMutex
	maxLogs                int
	retentionDays          int
	alertThreshold         map[string]int
	ipAccessCounts         map[string]int
	userAccessCounts       map[uint]int
	timeWindow             time.Duration
	enableGeoLocation      bool
	enableRiskScoring      bool
	sensitivePatterns      []*regexp.Regexp
}

func NewAccessAuditService() *AccessAuditService {
	service := &AccessAuditService{
		accessLogs:          make([]*AccessAuditLog, 0),
		permissionChanges:   make([]*PermissionChange, 0),
		sensitiveOperations: make([]*SensitiveOperation, 0),
		abnormalPatterns:    make([]*AbnormalAccessPattern, 0),
		maxLogs:             10000,
		retentionDays:       90,
		alertThreshold: map[string]int{
			"failed_login":        5,
			"admin_action":        10,
			"sensitive_data":      3,
			"permission_change":   2,
			"export":             5,
		},
		ipAccessCounts:   make(map[string]int),
		userAccessCounts: make(map[uint]int),
		timeWindow:       1 * time.Hour,
		enableGeoLocation: true,
		enableRiskScoring: true,
		sensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|secret|token|api[_-]key|credentials)`),
			regexp.MustCompile(`(?i)(ssn|credit[_-]card|cvv|bank[_-]account)`),
			regexp.MustCompile(`(?i)(private[_-]key|ssh[_-]key)`),
			regexp.MustCompile(`(?i)(email|phone|address|social[_-]security)`),
		},
	}

	go service.cleanupOldLogs()
	go service.detectAbnormalPatterns()

	return service
}

func (s *AccessAuditService) LogAccess(eventType AccessEventType, userID uint, username, ipAddress, userAgent string, resourceType, resourceID, action, status string, responseTime int64, metadata map[string]string) *AccessAuditLog {
	severity := s.determineSeverity(eventType, status)
	riskScore := s.calculateRiskScore(eventType, ipAddress, userID, resourceType)

	geoLocation := ""
	if s.enableGeoLocation {
		geoLocation = s.getGeoLocation(ipAddress)
	}

	accessLog := &AccessAuditLog{
		Timestamp:     time.Now(),
		EventType:    eventType,
		Severity:      severity,
		UserID:       userID,
		Username:     username,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Status:       status,
		SessionID:    metadata["session_id"],
		RequestPath:  metadata["request_path"],
		ResponseTime: responseTime,
		Metadata:     metadata,
		GeoLocation:  geoLocation,
		RiskScore:    riskScore,
		Tags:        s.generateTags(eventType, resourceType, status),
	}

	s.mu.Lock()
	s.accessLogs = append(s.accessLogs, accessLog)
	if len(s.accessLogs) > s.maxLogs {
		s.accessLogs = s.accessLogs[len(s.accessLogs)-s.maxLogs:]
	}

	s.updateAccessCounts(ipAddress, userID)
	s.mu.Unlock()

	s.checkAlertThreshold(eventType, ipAddress, userID)

	if err := s.persistAccessLog(accessLog); err != nil {
		fmt.Printf("Failed to persist access log: %v\n", err)
	}

	return accessLog
}

func (s *AccessAuditService) LogAccessFromRequest(eventType AccessEventType, userID uint, username string, r *http.Request, resourceType, resourceID, action, status string, responseTime int64) *AccessAuditLog {
	metadata := map[string]string{
		"request_path": r.URL.Path,
	}

	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		metadata["session_id"] = sessionID
	}

	return s.LogAccess(
		eventType,
		userID,
		username,
		getClientIP(r),
		r.UserAgent(),
		resourceType,
		resourceID,
		action,
		status,
		responseTime,
		metadata,
	)
}

func (s *AccessAuditService) LogPermissionChange(change *PermissionChange) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	change.Timestamp = time.Now()
	s.permissionChanges = append(s.permissionChanges, change)

	if err := s.persistPermissionChange(change); err != nil {
		return err
	}

	auditLog := &AccessAuditLog{
		Timestamp:    time.Now(),
		EventType:   AccessPermissionGrant,
		Severity:    SeverityHigh,
		UserID:      change.ChangedBy,
		Action:      change.ChangeType,
		ResourceType: "permission",
		ResourceID:  fmt.Sprintf("%d", change.UserID),
		Status:      "completed",
		Metadata: map[string]string{
			"permission": change.Permission,
			"old_value":  change.OldValue,
			"new_value":  change.NewValue,
			"reason":     change.Reason,
		},
		RiskScore: 75,
		Tags:      []string{"permission_change", "security"},
	}

	s.accessLogs = append(s.accessLogs, auditLog)

	return nil
}

func (s *AccessAuditService) LogSensitiveOperation(operationType, description, dataType string, userID uint, username, ipAddress string, riskLevel AccessSeverity) error {
	sensitiveOp := &SensitiveOperation{
		OperationType:  operationType,
		Description:    description,
		DataType:       dataType,
		RiskLevel:      riskLevel,
		RequiresApproval: riskLevel == SeverityHigh || riskLevel == SeverityCritical,
		AuditRequired: true,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sensitiveOperations = append(s.sensitiveOperations, sensitiveOp)

	accessLog := &AccessAuditLog{
		Timestamp:    time.Now(),
		EventType:   AccessSensitiveData,
		Severity:    riskLevel,
		UserID:      userID,
		Username:    username,
		IPAddress:   ipAddress,
		ResourceType: dataType,
		Action:      operationType,
		Status:      "completed",
		Metadata: map[string]string{
			"description": description,
		},
		RiskScore: s.calculateRiskScore(AccessSensitiveData, ipAddress, userID, dataType),
		Tags:      []string{"sensitive_data", "gdpr"},
	}

	s.accessLogs = append(s.accessLogs, accessLog)

	return s.persistAccessLog(accessLog)
}

func (s *AccessAuditService) DetectAbnormalAccess(userID uint, ipAddress string) ([]AbnormalAccessPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	patterns := make([]AbnormalAccessPattern, 0)

	ipCount := s.ipAccessCounts[ipAddress]
	if ipCount > 50 {
		patterns = append(patterns, AbnormalAccessPattern{
			PatternType:    "high_ip_frequency",
			Description:   fmt.Sprintf("IP %s has accessed %d times in the last hour", ipAddress, ipCount),
			UserID:        userID,
			IPAddress:     ipAddress,
			OccurrenceCount: ipCount,
			Severity:      SeverityMedium,
			Recommendation: "Review IP access patterns",
			RiskScore:     60,
		})
	}

	userCount := s.userAccessCounts[userID]
	if userCount > 100 {
		patterns = append(patterns, AbnormalAccessPattern{
			PatternType:    "high_user_frequency",
			Description:   fmt.Sprintf("User %d has performed %d actions in the last hour", userID, userCount),
			UserID:        userID,
			IPAddress:     ipAddress,
			OccurrenceCount: userCount,
			Severity:      SeverityMedium,
			Recommendation: "Review user activity",
			RiskScore:     65,
		})
	}

	for _, pattern := range s.abnormalPatterns {
		if pattern.UserID == userID || pattern.IPAddress == ipAddress {
			patterns = append(patterns, *pattern)
		}
	}

	return patterns, nil
}

func (s *AccessAuditService) GetAccessLogs(filter AccessLogFilter) ([]*AccessAuditLog, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*AccessAuditLog, 0)

	for _, log := range s.accessLogs {
		if filter.UserID > 0 && log.UserID != filter.UserID {
			continue
		}
		if filter.IPAddress != "" && log.IPAddress != filter.IPAddress {
			continue
		}
		if filter.EventType != "" && log.EventType != filter.EventType {
			continue
		}
		if filter.ResourceType != "" && log.ResourceType != filter.ResourceType {
			continue
		}
		if !filter.StartDate.IsZero() && log.Timestamp.Before(filter.StartDate) {
			continue
		}
		if !filter.EndDate.IsZero() && log.Timestamp.After(filter.EndDate) {
			continue
		}
		if filter.Status != "" && log.Status != filter.Status {
			continue
		}
		if filter.Severity != "" && log.Severity != filter.Severity {
			continue
		}

		filtered = append(filtered, log)
	}

	total := int64(len(filtered))

	start := 0
	end := len(filtered)
	if filter.Offset > 0 {
		start = filter.Offset
	}
	if filter.Limit > 0 && start+filter.Limit < len(filtered) {
		end = start + filter.Limit
	}

	return filtered[start:end], total, nil
}

type AccessLogFilter struct {
	UserID       uint
	IPAddress    string
	EventType    AccessEventType
	ResourceType string
	StartDate    time.Time
	EndDate      time.Time
	Status       string
	Severity     AccessSeverity
	Limit        int
	Offset       int
}

func (s *AccessAuditService) GetAccessStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_access":     0,
		"by_event_type":    make(map[string]int),
		"by_severity":      make(map[string]int),
		"by_status":        make(map[string]int),
		"unique_users":     make(map[uint]bool),
		"unique_ips":       make(map[string]bool),
		"avg_response_time": 0.0,
	}

	var totalResponseTime int64
	var count int

	for _, log := range s.accessLogs {
		if !startDate.IsZero() && log.Timestamp.Before(startDate) {
			continue
		}
		if !endDate.IsZero() && log.Timestamp.After(endDate) {
			continue
		}

		stats["total_access"] = stats["total_access"].(int) + 1
		stats["by_event_type"].(map[string]int)[string(log.EventType)]++
		stats["by_severity"].(map[string]int)[string(log.Severity)]++
		stats["by_status"].(map[string]int)[log.Status]++
		stats["unique_users"].(map[uint]bool)[log.UserID] = true
		stats["unique_ips"].(map[string]bool)[log.IPAddress] = true

		totalResponseTime += log.ResponseTime
		count++
	}

	if count > 0 {
		stats["avg_response_time"] = float64(totalResponseTime) / float64(count)
	}

	stats["unique_users_count"] = len(stats["unique_users"].(map[uint]bool))
	stats["unique_ips_count"] = len(stats["unique_ips"].(map[string]bool))

	delete(stats, "unique_users")
	delete(stats, "unique_ips")

	return stats, nil
}

func (s *AccessAuditService) GetPermissionChangeHistory(userID uint) ([]*PermissionChange, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	changes := make([]*PermissionChange, 0)
	for _, change := range s.permissionChanges {
		if change.UserID == userID || change.TargetUserID == userID {
			changes = append(changes, change)
		}
	}

	return changes, nil
}

func (s *AccessAuditService) ExportAccessLogs(format string, filter AccessLogFilter) ([]byte, error) {
	logs, _, err := s.GetAccessLogs(filter)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(logs, "", "  ")
	case "csv":
		return s.exportToCSV(logs)
	default:
		return json.MarshalIndent(logs, "", "  ")
	}
}

func (s *AccessAuditService) exportToCSV(logs []*AccessAuditLog) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("ID,Timestamp,EventType,Severity,UserID,Username,IPAddress,ResourceType,ResourceID,Action,Status,ResponseTime\n")

	for _, log := range logs {
		builder.WriteString(fmt.Sprintf("%d,%s,%s,%s,%d,%s,%s,%s,%s,%s,%s,%d\n",
			log.ID,
			log.Timestamp.Format(time.RFC3339),
			log.EventType,
			log.Severity,
			log.UserID,
			log.Username,
			log.IPAddress,
			log.ResourceType,
			log.ResourceID,
			log.Action,
			log.Status,
			log.ResponseTime,
		))
	}

	return []byte(builder.String()), nil
}

func (s *AccessAuditService) determineSeverity(eventType AccessEventType, status string) AccessSeverity {
	if status == "failed" {
		return SeverityMedium
	}

	switch eventType {
	case AccessAdminAction, AccessSensitiveData, AccessPermissionGrant, AccessPermissionRevoke:
		return SeverityHigh
	case AccessDelete, AccessConfigChange, AccessExport:
		return SeverityMedium
	case AccessLogin, AccessLogout:
		return SeverityLow
	default:
		return SeverityInfo
	}
}

func (s *AccessAuditService) calculateRiskScore(eventType AccessEventType, ipAddress string, userID uint, resourceType string) float64 {
	if !s.enableRiskScoring {
		return 0
	}

	score := 0.0

	switch eventType {
	case AccessSensitiveData:
		score += 30
	case AccessAdminAction:
		score += 25
	case AccessPermissionGrant, AccessPermissionRevoke:
		score += 35
	case AccessDelete:
		score += 20
	}

	ipCount := s.ipAccessCounts[ipAddress]
	if ipCount > 30 {
		score += 15
	}

	userCount := s.userAccessCounts[userID]
	if userCount > 50 {
		score += 10
	}

	if score > 100 {
		score = 100
	}

	return score
}

func (s *AccessAuditService) generateTags(eventType AccessEventType, resourceType, status string) []string {
	tags := []string{string(eventType)}

	if strings.HasPrefix(resourceType, "admin") {
		tags = append(tags, "admin")
	}
	if eventType == AccessSensitiveData {
		tags = append(tags, "sensitive", "gdpr")
	}
	if status == "failed" {
		tags = append(tags, "failed")
	}
	if s.isSensitiveResourceType(resourceType) {
		tags = append(tags, "pii")
	}

	return tags
}

func (s *AccessAuditService) isSensitiveResourceType(resourceType string) bool {
	sensitiveTypes := []string{"user", "password", "email", "phone", "address", "payment", "financial"}
	for _, t := range sensitiveTypes {
		if strings.Contains(strings.ToLower(resourceType), t) {
			return true
		}
	}
	return false
}

func (s *AccessAuditService) getGeoLocation(ipAddress string) string {
	if strings.HasPrefix(ipAddress, "10.") || strings.HasPrefix(ipAddress, "192.168.") || strings.HasPrefix(ipAddress, "172.") {
		return "Private Network"
	}
	return "Unknown"
}

func (s *AccessAuditService) updateAccessCounts(ipAddress string, userID uint) {
	s.ipAccessCounts[ipAddress]++
	s.userAccessCounts[userID]++

	if len(s.ipAccessCounts) > 10000 {
		s.cleanupOldCounts()
	}
}

func (s *AccessAuditService) cleanupOldCounts() {
	for ip := range s.ipAccessCounts {
		if s.ipAccessCounts[ip] < 5 {
			delete(s.ipAccessCounts, ip)
		}
	}

	for userID := range s.userAccessCounts {
		if s.userAccessCounts[userID] < 5 {
			delete(s.userAccessCounts, userID)
		}
	}
}

func (s *AccessAuditService) checkAlertThreshold(eventType AccessEventType, ipAddress string, userID uint) {
	key := string(eventType)
	if threshold, exists := s.alertThreshold[key]; exists {
		count := 0
		switch eventType {
		case AccessLogin:
			count = s.ipAccessCounts[ipAddress]
		default:
			count = s.userAccessCounts[userID]
		}

		if count >= threshold {
			s.triggerAlert(eventType, ipAddress, userID, count)
		}
	}
}

func (s *AccessAuditService) triggerAlert(eventType AccessEventType, ipAddress string, userID uint, count int) {
	alert := map[string]interface{}{
		"timestamp":      time.Now(),
		"event_type":    eventType,
		"ip_address":    ipAddress,
		"user_id":       userID,
		"count":         count,
		"alert_type":    "threshold_exceeded",
	}

	alertJSON, _ := json.Marshal(alert)
	fmt.Printf("[AccessAuditAlert] %s\n", string(alertJSON))
}

func (s *AccessAuditService) cleanupOldLogs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-time.Duration(s.retentionDays) * 24 * time.Hour)
		newLogs := make([]*AccessAuditLog, 0)
		for _, log := range s.accessLogs {
			if log.Timestamp.After(cutoff) {
				newLogs = append(newLogs, log)
			}
		}
		s.accessLogs = newLogs
		s.mu.Unlock()
	}
}

func (s *AccessAuditService) detectAbnormalPatterns() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()

		for ip, count := range s.ipAccessCounts {
			if count > 100 {
				pattern := &AbnormalAccessPattern{
					PatternType:    "abnormal_frequency",
					Description:   fmt.Sprintf("Abnormal access frequency from IP %s: %d requests/hour", ip, count),
					IPAddress:     ip,
					LastOccurrence: time.Now(),
					OccurrenceCount: count,
					Severity:      SeverityHigh,
					Recommendation: "Investigate potential abuse",
					RiskScore:     80,
				}
				s.abnormalPatterns = append(s.abnormalPatterns, pattern)
			}
		}

		if len(s.abnormalPatterns) > 1000 {
			s.abnormalPatterns = s.abnormalPatterns[len(s.abnormalPatterns)-1000:]
		}

		s.mu.Unlock()
	}
}

func (s *AccessAuditService) persistAccessLog(log *AccessAuditLog) error {
	auditLog := &models.AuditLog{
		LogType:      string(log.EventType),
		Level:        string(log.Severity),
		UserID:       log.UserID,
		Username:     log.Username,
		IPAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Status:       log.Status,
		ErrorMessage: log.ErrorMessage,
		Duration:     log.ResponseTime,
		SessionID:    log.SessionID,
	}

	if log.Metadata != nil {
		metadataJSON, _ := json.Marshal(log.Metadata)
		auditLog.Metadata = string(metadataJSON)
	}

	return database.DB.Create(auditLog).Error
}

func (s *AccessAuditService) persistPermissionChange(change *PermissionChange) error {
	changeJSON, _ := json.Marshal(change)

	auditLog := &models.AuditLog{
		LogType:      string(AccessPermissionGrant),
		Level:        string(SeverityHigh),
		UserID:       change.ChangedBy,
		Action:       change.ChangeType,
		ResourceType: "permission",
		ResourceID:   fmt.Sprintf("%d", change.UserID),
		Status:       change.Status,
		Changes:      string(changeJSON),
	}

	return database.DB.Create(auditLog).Error
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}

func (s *AccessAuditService) GetRecentAbnormalPatterns(limit int) []*AbnormalAccessPattern {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.abnormalPatterns) {
		limit = len(s.abnormalPatterns)
	}

	start := len(s.abnormalPatterns) - limit
	return s.abnormalPatterns[start:]
}

func (s *AccessAuditService) GetUserAccessSummary(userID uint, days int) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	startDate := time.Now().AddDate(0, 0, -days)

	var totalAccess int
	var byEventType = make(map[string]int)
	var bySeverity = make(map[string]int)
	var uniqueIPs = make(map[string]bool)
	var totalResponseTime int64

	for _, log := range s.accessLogs {
		if log.UserID != userID {
			continue
		}
		if log.Timestamp.Before(startDate) {
			continue
		}

		totalAccess++
		byEventType[string(log.EventType)]++
		bySeverity[string(log.Severity)]++
		uniqueIPs[log.IPAddress] = true
		totalResponseTime += log.ResponseTime
	}

	avgResponseTime := 0.0
	if totalAccess > 0 {
		avgResponseTime = float64(totalResponseTime) / float64(totalAccess)
	}

	return map[string]interface{}{
		"user_id":              userID,
		"period_days":         days,
		"total_access":        totalAccess,
		"by_event_type":       byEventType,
		"by_severity":         bySeverity,
		"unique_ips_count":    len(uniqueIPs),
		"avg_response_time_ms": avgResponseTime,
	}, nil
}

func (s *AccessAuditService) SetAlertThreshold(eventType string, threshold int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertThreshold[eventType] = threshold
}

func (s *AccessAuditService) GetAlertThresholds() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thresholds := make(map[string]int)
	for k, v := range s.alertThreshold {
		thresholds[k] = v
	}
	return thresholds
}
