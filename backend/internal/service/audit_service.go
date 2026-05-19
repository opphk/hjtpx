package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
)

type AuditService struct{}

func NewAuditService() *AuditService {
	return &AuditService{}
}

type PermissionChange struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id"`
	TargetUserID uint      `json:"target_user_id"`
	Action       string    `json:"action" binding:"required"`
	Resource     string    `json:"resource" binding:"required"`
	ResourceID   string    `json:"resource_id"`
	OldValue    string    `json:"old_value"`
	NewValue    string    `json:"new_value"`
	Reason      string    `json:"reason"`
	ApprovedBy  uint      `json:"approved_by"`
	ApprovedAt  *time.Time `json:"approved_at"`
	Status      string    `json:"status"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}

const (
	PermissionActionGrant    = "grant"
	PermissionActionRevoke   = "revoke"
	PermissionActionModify   = "modify"
	PermissionActionRoleChange = "role_change"
)

const (
	PermissionStatusPending  = "pending"
	PermissionStatusApproved = "approved"
	PermissionStatusRejected = "rejected"
	PermissionStatusApplied  = "applied"
)

type AccessControlAudit struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resource_id"`
	Result       string    `json:"result"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Details      string    `json:"details"`
	IsAnomaly    bool      `json:"is_anomaly" gorm:"default:false"`
	AnomalyReason string   `json:"anomaly_reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type RoleChangeAudit struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id"`
	PerformedBy  uint      `json:"performed_by"`
	OldRole      string    `json:"old_role"`
	NewRole      string    `json:"new_role"`
	Reason       string    `json:"reason"`
	IPAddress    string    `json:"ip_address"`
	ApprovedBy   uint      `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *AuditService) CreatePermissionChange(change *PermissionChange) error {
	change.CreatedAt = time.Now()
	change.Status = PermissionStatusPending
	return database.DB.Create(change).Error
}

func (s *AuditService) GetPermissionChangeByID(id uint) (*PermissionChange, error) {
	var change PermissionChange
	if err := database.DB.First(&change, id).Error; err != nil {
		return nil, err
	}
	return &change, nil
}

func (s *AuditService) ApprovePermissionChange(id uint, approvedBy uint) error {
	now := time.Now()
	return database.DB.Model(&PermissionChange{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      PermissionStatusApproved,
		"approved_by": approvedBy,
		"approved_at": &now,
	}).Error
}

func (s *AuditService) RejectPermissionChange(id uint, rejectedBy uint, reason string) error {
	now := time.Now()
	return database.DB.Model(&PermissionChange{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      PermissionStatusRejected,
		"approved_by": rejectedBy,
		"approved_at": &now,
		"reason":      reason,
	}).Error
}

func (s *AuditService) ListPermissionChanges(userID uint, status string, page, pageSize int) ([]PermissionChange, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := database.DB.Model(&PermissionChange{})

	if userID > 0 {
		query = query.Where("user_id = ? OR target_user_id = ?", userID, userID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var changes []PermissionChange
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&changes).Error; err != nil {
		return nil, 0, err
	}

	return changes, total, nil
}

func (s *AuditService) RecordAccessControlAudit(audit *AccessControlAudit) error {
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = time.Now()
	}

	isAnomaly := s.detectAnomaly(audit)
	audit.IsAnomaly = isAnomaly
	if isAnomaly {
		audit.AnomalyReason = s.getAnomalyReason(audit)
	}

	return database.DB.Create(audit).Error
}

func (s *AuditService) detectAnomaly(audit *AccessControlAudit) bool {
	var recentAccessCount int64
	database.DB.Model(&AccessControlAudit{}).
		Where("user_id = ? AND created_at > ?", audit.UserID, time.Now().Add(-1*time.Hour)).
		Count(&recentAccessCount)

	if recentAccessCount > 100 {
		return true
	}

	var sameActionCount int64
	database.DB.Model(&AccessControlAudit{}).
		Where("user_id = ? AND action = ? AND resource = ? AND created_at > ?",
			audit.UserID, audit.Action, audit.Resource, time.Now().Add(-5*time.Minute)).
		Count(&sameActionCount)

	if sameActionCount > 10 {
		return true
	}

	return false
}

func (s *AuditService) getAnomalyReason(audit *AccessControlAudit) string {
	var recentAccessCount int64
	database.DB.Model(&AccessControlAudit{}).
		Where("user_id = ? AND created_at > ?", audit.UserID, time.Now().Add(-1*time.Hour)).
		Count(&recentAccessCount)

	if recentAccessCount > 100 {
		return "Unusual high frequency of access control operations"
	}

	return "Unusual pattern detected"
}

func (s *AuditService) QueryAccessControlAudits(params map[string]interface{}) ([]AccessControlAudit, int64, error) {
	page := 1
	pageSize := 20

	if p, ok := params["page"].(int); ok {
		page = p
	}
	if ps, ok := params["page_size"].(int); ok && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	query := database.DB.Model(&AccessControlAudit{})

	if userID, ok := params["user_id"].(uint); ok && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if username, ok := params["username"].(string); ok && username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if action, ok := params["action"].(string); ok && action != "" {
		query = query.Where("action = ?", action)
	}
	if resource, ok := params["resource"].(string); ok && resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if result, ok := params["result"].(string); ok && result != "" {
		query = query.Where("result = ?", result)
	}
	if isAnomaly, ok := params["is_anomaly"].(bool); ok {
		query = query.Where("is_anomaly = ?", isAnomaly)
	}
	if startDate, ok := params["start_date"].(time.Time); ok {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := params["end_date"].(time.Time); ok {
		query = query.Where("created_at < ?", endDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var audits []AccessControlAudit
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&audits).Error; err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

func (s *AuditService) GetAnomalies(startDate, endDate time.Time) ([]AccessControlAudit, error) {
	var anomalies []AccessControlAudit
	err := database.DB.Where("is_anomaly = ? AND created_at >= ? AND created_at < ?",
		true, startDate, endDate).
		Order("created_at DESC").
		Find(&anomalies).Error

	return anomalies, err
}

func (s *AuditService) GetAccessControlStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalAccess int64
	database.DB.Model(&AccessControlAudit{}).
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Count(&totalAccess)
	stats["total_access"] = totalAccess

	var anomalyCount int64
	database.DB.Model(&AccessControlAudit{}).
		Where("is_anomaly = ? AND created_at >= ? AND created_at < ?", true, startDate, endDate).
		Count(&anomalyCount)
	stats["anomaly_count"] = anomalyCount

	var actionCounts []struct {
		Action string
		Count  int64
	}
	database.DB.Model(&AccessControlAudit{}).
		Select("action, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Group("action").
		Order("count DESC").
		Scan(&actionCounts)

	actionCountMap := make(map[string]int64)
	for _, ac := range actionCounts {
		actionCountMap[ac.Action] = ac.Count
	}
	stats["action_counts"] = actionCountMap

	var resultCounts []struct {
		Result string
		Count  int64
	}
	database.DB.Model(&AccessControlAudit{}).
		Select("result, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Group("result").
		Scan(&resultCounts)

	resultCountMap := make(map[string]int64)
	for _, rc := range resultCounts {
		resultCountMap[rc.Result] = rc.Count
	}
	stats["result_counts"] = resultCountMap

	var topUsers []struct {
		Username string
		Count    int64
	}
	database.DB.Model(&AccessControlAudit{}).
		Select("username, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Group("username").
		Order("count DESC").
		Limit(10).
		Scan(&topUsers)
	stats["top_users"] = topUsers

	var permissionChanges []struct {
		Action string
		Count  int64
	}
	database.DB.Model(&PermissionChange{}).
		Select("action, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Group("action").
		Scan(&permissionChanges)

	permissionChangeMap := make(map[string]int64)
	for _, pc := range permissionChanges {
		permissionChangeMap[pc.Action] = pc.Count
	}
	stats["permission_changes"] = permissionChangeMap

	return stats, nil
}

func (s *AuditService) CreateRoleChangeAudit(audit *RoleChangeAudit) error {
	audit.CreatedAt = time.Now()
	audit.Status = PermissionStatusPending
	return database.DB.Create(audit).Error
}

func (s *AuditService) ListRoleChanges(userID uint, page, pageSize int) ([]RoleChangeAudit, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := database.DB.Model(&RoleChangeAudit{})

	if userID > 0 {
		query = query.Where("user_id = ? OR performed_by = ?", userID, userID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var audits []RoleChangeAudit
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&audits).Error; err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

func (s *AuditService) GetSuspiciousActivityReport(startDate, endDate time.Time) (map[string]interface{}, error) {
	report := make(map[string]interface{})

	anomalies, err := s.GetAnomalies(startDate, endDate)
	if err != nil {
		return nil, err
	}
	report["anomalies"] = anomalies
	report["anomaly_count"] = len(anomalies)

	var suspiciousUsers []struct {
		UserID   uint
		Username string
		Count    int64
	}
	database.DB.Model(&AccessControlAudit{}).
		Select("user_id, username, COUNT(*) as count").
		Where("is_anomaly = ? AND created_at >= ? AND created_at < ?", true, startDate, endDate).
		Group("user_id, username").
		Order("count DESC").
		Scan(&suspiciousUsers)
	report["suspicious_users"] = suspiciousUsers

	var repeatedFailures []struct {
		IPAddress string
		Count     int64
	}
	database.DB.Model(&AccessControlAudit{}).
		Select("ip_address, COUNT(*) as count").
		Where("result = ? AND created_at >= ? AND created_at < ?", "denied", startDate, endDate).
		Group("ip_address").
		Having("count > ?", 10).
		Order("count DESC").
		Scan(&repeatedFailures)
	report["repeated_failures_by_ip"] = repeatedFailures

	var privilegeEscalationAttempts []PermissionChange
	database.DB.Model(&PermissionChange{}).
		Where("action = ? AND created_at >= ? AND created_at < ?", PermissionActionRoleChange, startDate, endDate).
		Find(&privilegeEscalationAttempts)
	report["privilege_escalation_attempts"] = privilegeEscalationAttempts

	report["generated_at"] = time.Now()
	report["period"] = map[string]interface{}{
		"start": startDate,
		"end":   endDate,
	}

	return report, nil
}

func (s *AuditService) ExportAccessControlAudits(params map[string]interface{}, format string) ([]byte, error) {
	audits, _, err := s.QueryAccessControlAudits(params)
	if err != nil {
		return nil, err
	}

	switch format {
	case "csv":
		return s.exportAuditsToCSV(audits)
	case "json":
		return json.MarshalIndent(audits, "", "  ")
	default:
		return s.exportAuditsToCSV(audits)
	}
}

func (s *AuditService) exportAuditsToCSV(audits []AccessControlAudit) ([]byte, error) {
	var result string
	result = "ID,UserID,Username,Action,Resource,ResourceID,Result,IPAddress,IsAnomaly,AnomalyReason,CreatedAt\n"

	for _, audit := range audits {
		result += fmt.Sprintf("%d,%d,%s,%s,%s,%s,%s,%s,%t,%s,%s\n",
			audit.ID,
			audit.UserID,
			audit.Username,
			audit.Action,
			audit.Resource,
			audit.ResourceID,
			audit.Result,
			audit.IPAddress,
			audit.IsAnomaly,
			audit.AnomalyReason,
			audit.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	return []byte(result), nil
}

func (s *AuditService) AlertOnPermissionAnomaly(audit *AccessControlAudit) error {
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"user_id":     audit.UserID,
		"username":    audit.Username,
		"action":      audit.Action,
		"resource":    audit.Resource,
		"ip_address":  audit.IPAddress,
		"anomaly":    audit.IsAnomaly,
		"reason":     audit.AnomalyReason,
		"timestamp":  audit.CreatedAt,
	})

	log := &AuditLog{
		UserID:     audit.UserID,
		Username:   audit.Username,
		Action:     "permission_anomaly_alert",
		Resource:   "access_control",
		ResourceID: fmt.Sprintf("%d", audit.ID),
		Details:    string(detailsJSON),
		IPAddress:  audit.IPAddress,
		Status:     "alert",
	}

	logService := NewLogService()
	return logService.CreateAuditLog(log)
}

func (s *AuditService) GetUserAccessSummary(userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	var totalAccess int64
	database.DB.Model(&AccessControlAudit{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startDate, endDate).
		Count(&totalAccess)
	summary["total_access"] = totalAccess

	var anomalyCount int64
	database.DB.Model(&AccessControlAudit{}).
		Where("user_id = ? AND is_anomaly = ? AND created_at >= ? AND created_at < ?", userID, true, startDate, endDate).
		Count(&anomalyCount)
	summary["anomaly_count"] = anomalyCount

	var roleChanges []RoleChangeAudit
	database.DB.Model(&RoleChangeAudit{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startDate, endDate).
		Find(&roleChanges)
	summary["role_changes"] = roleChanges

	var permissionChanges []PermissionChange
	database.DB.Model(&PermissionChange{}).
		Where("target_user_id = ? AND created_at >= ? AND created_at < ?", userID, startDate, endDate).
		Find(&permissionChanges)
	summary["permission_changes_received"] = permissionChanges

	summary["user_id"] = userID
	summary["period"] = map[string]interface{}{
		"start": startDate,
		"end":   endDate,
	}

	return summary, nil
}
