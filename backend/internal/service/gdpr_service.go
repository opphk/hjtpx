package service

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

var (
	ErrExportRequestNotFound = errors.New("导出请求未找到")
	ErrDeletionRequestNotFound = errors.New("删除请求未找到")
	ErrExportProcessing = errors.New("导出正在处理中")
	ErrDeletionProcessing = errors.New("删除正在处理中")
	ErrInvalidExportFormat = errors.New("无效的导出格式")
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
			UserID:                userID,
			ConsentMarketing:      false,
			ConsentAnalytics:      true,
			ConsentPersonalization: true,
			ConsentDataSharing:    false,
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
		"consent_marketing":      consent.ConsentMarketing,
		"consent_analytics":      consent.ConsentAnalytics,
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
	exportDir := "./exports"
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
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"nickname":   user.Nickname,
			"avatar":     user.Avatar,
			"phone":      user.Phone,
			"bio":        user.Bio,
			"is_verified": user.IsVerified,
			"status":     user.Status,
			"created_at": user.CreatedAt,
		},
		"applications": applications,
		"verifications": verifications,
		"consent": consent,
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
		"status":      "completed",
		"processed_at": now,
		"audit_log":   auditLog,
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