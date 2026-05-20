package service

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

var (
	ErrExportRequestNotFound   = errors.New("导出请求未找到")
	ErrDeletionRequestNotFound = errors.New("删除请求未找到")
	ErrExportProcessing        = errors.New("导出正在处理中")
	ErrDeletionProcessing      = errors.New("删除正在处理中")
	ErrInvalidExportFormat     = errors.New("无效的导出格式")
)

// GDPRService GDPR服务
type GDPRService struct{}

// NewGDPRService 创建GDPR服务实例
func NewGDPRService() *GDPRService {
	return &GDPRService{}
}

// GetConsent 获取用户同意设置
func (s *GDPRService) GetConsent(userID uint) (*models.UserConsent, error) {
	var consent models.UserConsent
	err := database.DB.Where("user_id = ?", userID).First(&consent).Error
	if err != nil {
		// 如果不存在，返回默认设置
		return &models.UserConsent{
			UserID:                 userID,
			ConsentMarketing:       false,
			ConsentAnalytics:       true,
			ConsentPersonalization: true,
			ConsentDataSharing:     false,
		}, nil
	}
	return &consent, nil
}

// UpdateConsent 更新用户同意设置
func (s *GDPRService) UpdateConsent(userID uint, consent *models.UserConsent, clientIP, userAgent string) (*models.UserConsent, error) {
	var existing models.UserConsent
	err := database.DB.Where("user_id = ?", userID).First(&existing).Error

	now := time.Now()
	consent.UserID = userID
	consent.ConsentUpdatedAt = now
	consent.ConsentIP = clientIP
	consent.ConsentUserAgent = userAgent

	if err != nil {
		// 创建新记录
		consent.CreatedAt = now
		if err := database.DB.Create(consent).Error; err != nil {
			return nil, err
		}
		return consent, nil
	}

	// 更新现有记录
	updates := map[string]interface{}{
		"consent_marketing":       consent.ConsentMarketing,
		"consent_analytics":       consent.ConsentAnalytics,
		"consent_personalization": consent.ConsentPersonalization,
		"consent_data_sharing":    consent.ConsentDataSharing,
		"consent_updated_at":      now,
		"consent_ip":              clientIP,
		"consent_user_agent":      userAgent,
	}

	if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}

	database.DB.First(&existing, existing.ID)
	return &existing, nil
}

// RequestDataExport 请求数据导出
func (s *GDPRService) RequestDataExport(userID uint, format string) (*models.DataExportRequest, error) {
	if format != "json" && format != "csv" {
		return nil, ErrInvalidExportFormat
	}

	// 检查是否有正在进行的导出
	var existing models.DataExportRequest
	err := database.DB.Where("user_id = ? AND status IN ?", userID, []string{"pending", "processing"}).First(&existing).Error
	if err == nil {
		return nil, ErrExportProcessing
	}

	request := &models.DataExportRequest{
		UserID:       userID,
		ExportFormat: format,
		Status:       "pending",
		RequestedAt:  time.Now(),
	}

	if err := database.DB.Create(request).Error; err != nil {
		return nil, err
	}

	// 异步处理导出（这里简化为同步处理）
	go s.processDataExport(request.ID)

	return request, nil
}

// processDataExport 处理数据导出
func (s *GDPRService) processDataExport(requestID uint) {
	var request models.DataExportRequest
	if err := database.DB.First(&request, requestID).Error; err != nil {
		return
	}

	// 更新状态为处理中
	database.DB.Model(&request).Update("status", "processing")

	// 获取用户数据
	userData, err := s.collectUserData(request.UserID)
	if err != nil {
		database.DB.Model(&request).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// 创建导出目录
	exportDir := github.com/hjtpx/hjtpx/exports"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		database.DB.Model(&request).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// 生成文件
	fileName := fmt.Sprintf("user_data_%d_%s.%s", request.UserID, time.Now().Format("20060102150405"), request.ExportFormat)
	filePath := filepath.Join(exportDir, fileName)

	if request.ExportFormat == "json" {
		err = s.exportToJSON(userData, filePath)
	} else {
		err = s.exportToCSV(userData, filePath)
	}

	if err != nil {
		database.DB.Model(&request).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// 更新状态为完成
	now := time.Now()
	database.DB.Model(&request).Updates(map[string]interface{}{
		"status":       "completed",
		"file_path":    filePath,
		"completed_at": now,
	})
}

// collectUserData 收集用户数据
func (s *GDPRService) collectUserData(userID uint) (map[string]interface{}, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("用户未找到")
	}

	// 获取用户应用
	var applications []models.Application
	database.DB.Where("user_id = ?", userID).Find(&applications)

	// 获取用户验证记录
	var verifications []models.Verification
	database.DB.Where("user_id = ?", userID).Find(&verifications)

	// 获取用户同意设置
	var consent models.UserConsent
	database.DB.Where("user_id = ?", userID).First(&consent)

	return map[string]interface{}{
		"user": map[string]interface{}{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"nickname":    user.Nickname,
			"avatar":      user.Avatar,
			"phone":       user.Phone,
			"bio":         user.Bio,
			"is_verified": user.IsVerified,
			"status":      user.Status,
			"created_at":  user.CreatedAt,
		},
		"applications":  applications,
		"verifications": verifications,
		"consent":       consent,
	}, nil
}

// exportToJSON 导出为JSON格式
func (s *GDPRService) exportToJSON(data map[string]interface{}, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// exportToCSV 导出为CSV格式
func (s *GDPRService) exportToCSV(data map[string]interface{}, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入用户信息头部
	userData := data["user"].(map[string]interface{})
	writer.Write([]string{"Section", "Field", "Value"})

	// 写入用户信息
	for k, v := range userData {
		writer.Write([]string{"User", k, fmt.Sprintf("%v", v)})
	}

	// 写入应用信息
	if apps, ok := data["applications"].([]models.Application); ok {
		for _, app := range apps {
			writer.Write([]string{"Application", "ID", fmt.Sprintf("%d", app.ID)})
			writer.Write([]string{"Application", "Name", app.Name})
			writer.Write([]string{"Application", "Description", app.Description})
		}
	}

	return nil
}

// GetExportRequest 获取导出请求状态
func (s *GDPRService) GetExportRequest(requestID uint) (*models.DataExportRequest, error) {
	var request models.DataExportRequest
	if err := database.DB.First(&request, requestID).Error; err != nil {
		return nil, ErrExportRequestNotFound
	}
	return &request, nil
}

// RequestDataDeletion 请求数据删除
func (s *GDPRService) RequestDataDeletion(userID uint, reason string) (*models.DataDeletionRequest, error) {
	// 检查是否有正在进行的删除请求
	var existing models.DataDeletionRequest
	err := database.DB.Where("user_id = ? AND status IN ?", userID, []string{"pending", "processing"}).First(&existing).Error
	if err == nil {
		return nil, ErrDeletionProcessing
	}

	request := &models.DataDeletionRequest{
		UserID:      userID,
		Status:      "pending",
		RequestedAt: time.Now(),
		Reason:      reason,
	}

	if err := database.DB.Create(request).Error; err != nil {
		return nil, err
	}

	// 异步处理删除请求
	go s.processDataDeletion(request.ID)

	return request, nil
}

// processDataDeletion 处理数据删除
func (s *GDPRService) processDataDeletion(requestID uint) {
	var request models.DataDeletionRequest
	if err := database.DB.First(&request, requestID).Error; err != nil {
		return
	}

	// 更新状态为处理中
	database.DB.Model(&request).Update("status", "processing")

	// 获取用户数据并创建快照
	var user models.User
	if err := database.DB.First(&user, request.UserID).Error; err != nil {
		database.DB.Model(&request).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// 创建用户数据快照
	userData, _ := json.Marshal(map[string]interface{}{
		"user": user,
	})

	snapshot := &models.UserDataSnapshot{
		UserID:       request.UserID,
		UserData:     string(userData),
		DeletedAt:    time.Now(),
		RetentionEnd: time.Now().AddDate(0, 6, 0), // 保留6个月
	}

	if err := database.DB.Create(snapshot).Error; err != nil {
		// 继续处理，即使快照创建失败
	}

	// 软删除用户数据（更新状态）
	auditLog := fmt.Sprintf("User data deletion requested at %s by user %d", time.Now().Format(time.RFC3339), request.UserID)

	// 更新用户状态
	database.DB.Model(&user).Update("status", "deleted")

	// 更新验证记录（软删除）
	database.DB.Model(&models.Verification{}).Where("user_id = ?", request.UserID).Update("deleted_at", time.Now())

	// 完成删除请求
	now := time.Now()
	database.DB.Model(&request).Updates(map[string]interface{}{
		"status":       "completed",
		"processed_at": now,
		"audit_log":    auditLog,
	})
}

// GetDeletionRequest 获取删除请求状态
func (s *GDPRService) GetDeletionRequest(requestID uint) (*models.DataDeletionRequest, error) {
	var request models.DataDeletionRequest
	if err := database.DB.First(&request, requestID).Error; err != nil {
		return nil, ErrDeletionRequestNotFound
	}
	return &request, nil
}

type DataProcessingConsent struct {
	ProcessingType     string `json:"processing_type"`
	Purpose            string `json:"purpose"`
	LegalBasis         string `json:"legal_basis"`
	ThirdPartySharing  bool   `json:"third_party_sharing"`
	InternationalTransfer bool `json:"international_transfer"`
	RetentionPeriod    int    `json:"retention_period_days"`
	ConsentGiven       bool   `json:"consent_given"`
}

type ProcessingActivity struct {
	ActivityType string                 `json:"activity_type"`
	Description  string                 `json:"description"`
	Purpose      string                 `json:"purpose"`
	DataCategories []string             `json:"data_categories"`
	LegalBasis   string                 `json:"legal_basis"`
	Recipients   []string               `json:"recipients"`
	Transfers    []string               `json:"transfers"`
	RetentionDays int                   `json:"retention_days"`
	SecurityMeasures string             `json:"security_measures"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func (s *GDPRService) AnonymizeUserData(userID uint) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("用户未找到")
	}

	anonymizedUsername := generateAnonymizedID("user")
	anonymizedEmail := anonymizedUsername + "@anonymized.local"
	anonymizedPhone := "0000000000"
	anonymizedNickname := "Anonymous User"

	updates := map[string]interface{}{
		"username":    anonymizedUsername,
		"email":       anonymizedEmail,
		"phone":       anonymizedPhone,
		"nickname":    anonymizedNickname,
		"bio":         "",
		"avatar":      "",
		"status":      "anonymized",
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return fmt.Errorf("匿名化失败: %w", err)
	}

	s.anonymizeUserVerifications(userID, anonymizedUsername)
	s.anonymizeUserApplications(userID, anonymizedUsername)
	s.anonymizeUserSessions(userID)

	if err := s.logAnonymization(userID, updates); err != nil {
		fmt.Printf("警告: 匿名化日志记录失败: %v\n", err)
	}

	return nil
}

func (s *GDPRService) anonymizeUserVerifications(userID uint, anonymizedUsername string) {
	database.DB.Model(&models.Verification{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"ip_address":    "0.0.0.0",
			"user_agent":    "Anonymized",
		})
}

func (s *GDPRService) anonymizeUserApplications(userID uint, anonymizedUsername string) {
	database.DB.Model(&models.Application{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"name":        "Anonymized Application " + generateAnonymizedID("app"),
			"description": "",
			"website":     "",
			"domain":      "",
		})
}

func (s *GDPRService) anonymizeUserSessions(userID uint) {
	database.DB.Model(&models.UserMFA{}).
		Where("user_id = ?", userID).
		Update("secret", "")
}

func (s *GDPRService) logAnonymization(userID uint, changes map[string]interface{}) error {
	changesJSON, _ := json.Marshal(changes)

	auditLog := &models.AuditLog{
		LogType:      "data_anonymize",
		Level:        "warning",
		UserID:       userID,
		Username:     "system",
		IPAddress:    "0.0.0.0",
		Action:       "anonymize_user_data",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", userID),
		Status:       "completed",
		Changes:      string(changesJSON),
	}

	return database.DB.Create(auditLog).Error
}

func (s *GDPRService) RequestDataAnonymization(userID uint, reason string) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("用户未找到")
	}

	if user.Status == "anonymized" {
		return errors.New("用户数据已经匿名化")
	}

	if err := s.AnonymizeUserData(userID); err != nil {
		return err
	}

	return nil
}

func (s *GDPRService) GetDataProcessingRecords(userID uint) ([]ProcessingActivity, error) {
	var verifications []models.Verification
	if err := database.DB.Where("user_id = ?", userID).Find(&verifications).Error; err != nil {
		return nil, err
	}

	activities := make([]ProcessingActivity, 0)

	activities = append(activities, ProcessingActivity{
		ActivityType:     "verification",
		Description:      "Captcha verification processing",
		Purpose:          "Security and fraud prevention",
		DataCategories:   []string{"IP address", "User agent", "Behavior data"},
		LegalBasis:       "Legitimate interest",
		Recipients:       []string{"Application owners"},
		RetentionDays:    90,
		SecurityMeasures: "Encryption, Access controls",
	})

	activities = append(activities, ProcessingActivity{
		ActivityType:     "analytics",
		Description:      "Usage analytics and performance monitoring",
		Purpose:          "Service improvement",
		DataCategories:   []string{"Usage patterns", "Performance metrics"},
		LegalBasis:       "Consent",
		Recipients:       []string{"Internal analytics systems"},
		RetentionDays:    365,
		SecurityMeasures: "Aggregation, Anonymization",
	})

	return activities, nil
}

func (s *GDPRService) RecordDataProcessingConsent(userID uint, consent DataProcessingConsent, clientIP, userAgent string) error {
	consentJSON, _ := json.Marshal(consent)

	consentRecord := &models.AuditLog{
		LogType:      "data_processing_consent",
		Level:        "info",
		UserID:       userID,
		IPAddress:    clientIP,
		UserAgent:    userAgent,
		Action:       "record_consent",
		ResourceType: "consent",
		Status:       "recorded",
		Metadata:     string(consentJSON),
	}

	return database.DB.Create(consentRecord).Error
}

func (s *GDPRService) GetDataProcessingConsent(userID uint) ([]DataProcessingConsent, error) {
	var logs []models.AuditLog
	if err := database.DB.Where("user_id = ? AND log_type = ?", userID, "data_processing_consent").
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	consents := make([]DataProcessingConsent, 0)
	for _, log := range logs {
		if log.Metadata != "" {
			var consent DataProcessingConsent
			if err := json.Unmarshal([]byte(log.Metadata), &consent); err == nil {
				consents = append(consents, consent)
			}
		}
	}

	return consents, nil
}

func (s *GDPRService) RevokeDataProcessingConsent(userID uint, processingType string) error {
	consent := &models.UserConsent{}
	if err := database.DB.Where("user_id = ?", userID).First(consent).Error; err != nil {
		return errors.New("用户同意记录未找到")
	}

	updates := map[string]interface{}{}

	switch processingType {
	case "marketing":
		updates["consent_marketing"] = false
	case "analytics":
		updates["consent_analytics"] = false
	case "personalization":
		updates["consent_personalization"] = false
	case "data_sharing":
		updates["consent_data_sharing"] = false
	default:
		return errors.New("无效的处理类型")
	}

	if err := database.DB.Model(consent).Updates(updates).Error; err != nil {
		return err
	}

	auditLog := &models.AuditLog{
		LogType:      "consent_revocation",
		Level:        "warning",
		UserID:       userID,
		Action:       "revoke_consent",
		ResourceType: "consent",
		ResourceID:   processingType,
		Status:       "completed",
		Changes:      fmt.Sprintf(`{"processing_type": "%s", "action": "revoked"}`, processingType),
	}

	return database.DB.Create(auditLog).Error
}

func (s *GDPRService) GeneratePrivacyReport(userID uint) (map[string]interface{}, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("用户未找到")
	}

	var verifications []models.Verification
	if err := database.DB.Where("user_id = ?", userID).Find(&verifications).Error; err != nil {
		return nil, err
	}

	var applications []models.Application
	if err := database.DB.Where("user_id = ?", userID).Find(&applications).Error; err != nil {
		return nil, err
	}

	var consent models.UserConsent
	if err := database.DB.Where("user_id = ?", userID).First(&consent).Error; err != nil {
		consent = models.UserConsent{}
	}

	report := map[string]interface{}{
		"user_id":              userID,
		"report_generated_at":  time.Now(),
		"account_status":       user.Status,
		"account_created_at":   user.CreatedAt,
		"data_summary": map[string]interface{}{
			"total_verifications": len(verifications),
			"total_applications":  len(applications),
		},
		"consent_status": map[string]interface{}{
			"marketing":       consent.ConsentMarketing,
			"analytics":       consent.ConsentAnalytics,
			"personalization": consent.ConsentPersonalization,
			"data_sharing":    consent.ConsentDataSharing,
			"last_updated":    consent.ConsentUpdatedAt,
		},
		"data_categories": []string{
			"Account information",
			"Verification history",
			"Application data",
			"Usage analytics",
		},
		"rights_available": []string{
			"Right to access",
			"Right to rectification",
			"Right to erasure",
			"Right to data portability",
			"Right to object",
			"Right to restriction",
		},
	}

	return report, nil
}

func generateAnonymizedID(prefix string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return prefix + "_" + fmt.Sprintf("%x", bytes)
}

func (s *GDPRService) CheckDataBreach(userID uint) (bool, []string, error) {
	var breaches []string

	var verifications []models.Verification
	if err := database.DB.Where("user_id = ?", userID).Find(&verifications).Error; err != nil {
		return false, nil, err
	}

	ipCounts := make(map[string]int)
	for _, v := range verifications {
		ipCounts[v.IPAddress]++
	}

	for ip, count := range ipCounts {
		if count > 100 {
			breaches = append(breaches, fmt.Sprintf("异常IP访问: %s 出现 %d 次", ip, count))
		}
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err == nil {
		if user.PasswordResetToken != "" {
			breaches = append(breaches, "未过期的密码重置令牌")
		}
	}

	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailPattern.MatchString(user.Email) {
		breaches = append(breaches, "邮箱格式异常")
	}

	return len(breaches) > 0, breaches, nil
}

func (s *GDPRService) GetDataRetentionPolicy(userID uint) (map[string]interface{}, error) {
	var verifications []models.Verification
	if err := database.DB.Where("user_id = ?", userID).Find(&verifications).Error; err != nil {
		return nil, err
	}

	oldestRecord := time.Now()
	if len(verifications) > 0 {
		oldestRecord = verifications[0].CreatedAt
	}

	retentionPolicies := []map[string]interface{}{
		{
			"data_type":      "Verification records",
			"retention_days": 90,
			"legal_basis":    "Legitimate interest",
		},
		{
			"data_type":      "Account information",
			"retention_days": 2555,
			"legal_basis":    "Contract performance",
		},
		{
			"data_type":      "Analytics data",
			"retention_days": 365,
			"legal_basis":    "Consent",
		},
		{
			"data_type":      "Consent records",
			"retention_days": 2555,
			"legal_basis":    "Legal obligation",
		},
	}

	policy := map[string]interface{}{
		"user_id":              userID,
		"oldest_record_date":  oldestRecord,
		"policy_generated_at": time.Now(),
		"retention_policies":  retentionPolicies,
		"deletion_request":    "/api/gdpr/request-deletion",
		"export_request":      "/api/gdpr/request-export",
	}

	return policy, nil
}

func (s *GDPRService) VerifyDataSubjectIdentity(userID uint, verificationData map[string]string) (bool, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return false, errors.New("用户未找到")
	}

	if verificationData["email"] != "" {
		if !strings.EqualFold(verificationData["email"], user.Email) {
			return false, errors.New("邮箱验证失败")
		}
	}

	if verificationData["username"] != "" {
		if !strings.EqualFold(verificationData["username"], user.Username) {
			return false, errors.New("用户名验证失败")
		}
	}

	if verificationData["phone"] != "" {
		if verificationData["phone"] != user.Phone {
			return false, errors.New("手机号验证失败")
		}
	}

	auditLog := &models.AuditLog{
		LogType:      "identity_verification",
		Level:        "info",
		UserID:       userID,
		Username:     user.Username,
		IPAddress:    verificationData["ip_address"],
		Action:       "verify_identity",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", userID),
		Status:       "success",
	}

	database.DB.Create(auditLog)

	return true, nil
}

func (s *GDPRService) GetCrossBorderDataTransfers(userID uint) ([]map[string]interface{}, error) {
	var verifications []models.Verification
	if err := database.DB.Where("user_id = ?", userID).Find(&verifications).Error; err != nil {
		return nil, err
	}

	transfers := []map[string]interface{}{
		{
			"transfer_type":    "Cloud storage",
			"destination":      "EU data centers",
			"data_categories":  []string{"Verification data", "Analytics"},
			"safeguards":       "Standard Contractual Clauses",
			"adequacy_decision": true,
		},
		{
			"transfer_type":    "Analytics services",
			"destination":      "Analytics platform (EU)",
			"data_categories":  []string{"Aggregated statistics"},
			"safeguards":       "Anonymization",
			"adequacy_decision": false,
		},
	}

	return transfers, nil
}
