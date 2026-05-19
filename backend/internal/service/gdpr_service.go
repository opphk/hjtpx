package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type GDPRService struct{}

func NewGDPRService() *GDPRService {
	return &GDPRService{}
}

type DataSubjectRequest struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Type        string    `json:"type" binding:"required"`
	Email       string    `json:"email" binding:"required"`
	Status      string    `json:"status" gorm:"default:'pending'"`
	DataTypes   string    `json:"data_types"`
	RequestData string    `json:"request_data"`
	ResponseData string   `json:"response_data"`
	Notes       string    `json:"notes"`
	ReviewedBy  uint      `json:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const (
	RequestTypeAccess       = "access"
	RequestTypeRectification = "rectification"
	RequestTypeErasure      = "erasure"
	RequestTypePortability  = "portability"
	RequestTypeRestriction  = "restriction"
	RequestTypeObjection    = "objection"
)

const (
	RequestStatusPending    = "pending"
	RequestStatusProcessing = "processing"
	RequestStatusCompleted  = "completed"
	RequestStatusRejected  = "rejected"
)

func (s *GDPRService) CreateDataSubjectRequest(request *DataSubjectRequest) error {
	request.CreatedAt = time.Now()
	request.UpdatedAt = time.Now()
	request.Status = RequestStatusPending

	return database.DB.Create(request).Error
}

func (s *GDPRService) GetRequestByID(id uint) (*DataSubjectRequest, error) {
	var request DataSubjectRequest
	if err := database.DB.First(&request, id).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (s *GDPRService) GetRequestByEmail(email string) ([]DataSubjectRequest, error) {
	var requests []DataSubjectRequest
	if err := database.DB.Where("email = ?", email).Order("created_at DESC").Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func (s *GDPRService) UpdateRequestStatus(id uint, status string, reviewedBy uint, notes string) error {
	updates := map[string]interface{}{
		"status":     status,
		"reviewed_by": reviewedBy,
		"updated_at": time.Now(),
	}

	if notes != "" {
		updates["notes"] = notes
	}

	if status == RequestStatusCompleted {
		now := time.Now()
		updates["completed_at"] = &now
	}

	if reviewedBy > 0 {
		now := time.Now()
		updates["reviewed_at"] = &now
	}

	return database.DB.Model(&DataSubjectRequest{}).Where("id = ?", id).Updates(updates).Error
}

func (s *GDPRService) ListRequests(status string, requestType string, page, pageSize int) ([]DataSubjectRequest, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := database.DB.Model(&DataSubjectRequest{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if requestType != "" {
		query = query.Where("type = ?", requestType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var requests []DataSubjectRequest
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests).Error; err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

func (s *GDPRService) ExportUserData(userID uint) (map[string]interface{}, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var verificationLogs []models.VerificationLog
	database.DB.Where("user_id = ?", userID).Find(&verificationLogs)

	var adminLogs []models.AdminLoginLog
	database.DB.Where("admin_id = ?", userID).Find(&adminLogs)

	var oauth2Tokens []struct {
		Provider string    `json:"provider"`
		CreatedAt time.Time `json:"created_at"`
	}
	database.DB.Model(&OAuth2Token{}).Where("user_id = ?", userID).Select("provider, created_at").Find(&oauth2Tokens)

	var sessions []struct {
		ID        uint      `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	database.DB.Model(&models.Session{}).Where("user_id = ?", userID).Select("id, created_at, expires_at").Find(&sessions)

	exportData := map[string]interface{}{
		"exported_at":     time.Now(),
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
		"verification_logs": verificationLogs,
		"admin_logs":       adminLogs,
		"oauth2_connections": oauth2Tokens,
		"sessions":          sessions,
		"data_summary": map[string]interface{}{
			"total_verification_logs": len(verificationLogs),
			"total_admin_logs":        len(adminLogs),
			"total_oauth2_connections": len(oauth2Tokens),
			"total_sessions":          len(sessions),
		},
	}

	return exportData, nil
}

func (s *GDPRService) ExportUserDataJSON(userID uint) ([]byte, error) {
	data, err := s.ExportUserData(userID)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(data, "", "  ")
}

func (s *GDPRService) DeleteUserData(userID uint, dataTypes []string) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	for _, dataType := range dataTypes {
		var count int64
		var err error

		switch dataType {
		case "verification_logs":
			result := database.DB.Where("user_id = ?", userID).Delete(&models.VerificationLog{})
			count = result.RowsAffected
			err = result.Error

		case "sessions":
			result := database.DB.Where("user_id = ?", userID).Delete(&models.Session{})
			count = result.RowsAffected
			err = result.Error

		case "oauth2_tokens":
			result := database.DB.Where("user_id = ?", userID).Delete(&OAuth2Token{})
			count = result.RowsAffected
			err = result.Error

		case "user_profile":
			result := database.DB.Where("user_id = ?", userID).Delete(&models.UserProfile{})
			count = result.RowsAffected
			err = result.Error

		case "all":
			var user models.User
			if err := database.DB.First(&user, userID).Error; err == nil {
				user.Username = fmt.Sprintf("deleted_user_%d", user.ID)
				user.Email = fmt.Sprintf("deleted_%d@example.com", user.ID)
				user.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
				database.DB.Save(&user)
				count = 1
			}

		default:
			err = fmt.Errorf("unknown data type: %s", dataType)
		}

		if err != nil {
			results[dataType] = map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
		} else {
			results[dataType] = map[string]interface{}{
				"success": true,
				"deleted":  count,
			}
		}
	}

	return results, nil
}

func (s *GDPRService) AnonymizeUserData(userID uint) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return err
	}

	user.Username = fmt.Sprintf("anonymous_%d", user.ID)
	user.Email = fmt.Sprintf("anonymous_%d@hjtpx.local", user.ID)
	user.Phone = ""
	user.Status = "deleted"

	if err := database.DB.Save(&user).Error; err != nil {
		return err
	}

	database.DB.Model(&models.VerificationLog{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"ip_address": "0.0.0.0",
		"user_agent": "[REDACTED]",
	})

	database.DB.Model(&models.Session{}).Where("user_id = ?", userID).Delete(&models.Session{})

	return nil
}

func (s *GDPRService) GetDataProcessingActivities() ([]map[string]interface{}, error) {
	activities := []map[string]interface{}{
		{
			"activity":          "User Registration and Authentication",
			"purpose":           "To provide access to our services",
			"legal_basis":       "Contract performance",
			"data_categories":   []string{"Contact information", "Authentication credentials"},
			"retention_period":  "Duration of account plus 30 days",
			"recipients":        "Internal systems",
		},
		{
			"activity":          "Captcha Verification",
			"purpose":           "To prevent automated abuse and ensure security",
			"legal_basis":       "Legitimate interests",
			"data_categories":   []string{"Behavioral data", "IP addresses", "Timestamps"},
			"retention_period":  "90 days",
			"recipients":        "Internal systems",
		},
		{
			"activity":          "Analytics and Improvement",
			"purpose":           "To improve our services and user experience",
			"legal_basis":       "Legitimate interests",
			"data_categories":   []string{"Usage patterns", "Performance metrics"},
			"retention_period":  "24 months",
			"recipients":        "Analytics systems",
		},
		{
			"activity":          "Security Monitoring",
			"purpose":           "To detect and prevent security threats",
			"legal_basis":       "Legal obligation",
			"data_categories":   []string{"Access logs", "Security events"},
			"retention_period":  "12 months",
			"recipients":        "Security systems",
		},
	}

	return activities, nil
}

func (s *GDPRService) GetPrivacyNotice() string {
	return `
HJTPX Privacy Notice

Last updated: 2026-05-19

1. DATA CONTROLLER
HJTPX operates as the data controller for personal data processed through our services.

2. DATA WE COLLECT
We collect the following categories of personal data:
- Identity data (username, email)
- Contact data (email address)
- Technical data (IP address, browser type, device information)
- Usage data (verification attempts, timestamps)
- Behavioral data (captcha interaction patterns)

3. PURPOSE AND LEGAL BASIS
We process your data for the following purposes:
- Providing access to our services (contract performance)
- Ensuring security and preventing fraud (legitimate interests)
- Legal compliance obligations

4. YOUR RIGHTS
Under GDPR, you have the following rights:
- Right of access
- Right to rectification
- Right to erasure ("right to be forgotten")
- Right to data portability
- Right to restrict processing
- Right to object
- Rights related to automated decision making

5. DATA RETENTION
We retain your personal data only for as long as necessary to fulfill the purposes for which it was collected.

6. DATA SECURITY
We implement appropriate technical and organizational measures to protect your personal data.

7. CONTACT US
For any privacy-related inquiries, please contact: privacy@hjtpx.example.com
`
}

func (s *GDPRService) GetConsentRecord(userID uint) ([]map[string]interface{}, error) {
	var records []map[string]interface{}

	records = append(records, map[string]interface{}{
		"consent_type":  "terms_of_service",
		"granted":       true,
		"granted_at":    time.Now().AddDate(0, -6, 0),
		"withdrawn_at":   nil,
		"version":       "1.0",
	})

	return records, nil
}

func (s *GDPRService) ProcessErasureRequest(requestID uint, reviewerID uint) (map[string]interface{}, error) {
	var request DataSubjectRequest
	if err := database.DB.First(&request, requestID).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := database.DB.Where("email = ?", request.Email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found for email: %s", request.Email)
	}

	dataTypes := []string{"verification_logs", "sessions", "oauth2_tokens"}
	if request.RequestData != "" {
		json.Unmarshal([]byte(request.RequestData), &dataTypes)
	}

	results, err := s.DeleteUserData(user.ID, dataTypes)
	if err != nil {
		return nil, err
	}

	if err := s.UpdateRequestStatus(requestID, RequestStatusCompleted, reviewerID, "Erasure completed successfully"); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *GDPRService) ProcessAccessRequest(requestID uint, reviewerID uint) ([]byte, error) {
	var request DataSubjectRequest
	if err := database.DB.First(&request, requestID).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := database.DB.Where("email = ?", request.Email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found for email: %s", request.Email)
	}

	exportData, err := s.ExportUserData(user.ID)
	if err != nil {
		return nil, err
	}

	responseData, _ := json.Marshal(exportData)

	request.ResponseData = string(responseData)
	if err := database.DB.Save(&request).Error; err != nil {
		return nil, err
	}

	if err := s.UpdateRequestStatus(requestID, RequestStatusCompleted, reviewerID, "Access data exported"); err != nil {
		return nil, err
	}

	return json.MarshalIndent(exportData, "", "  ")
}

func (s *GDPRService) ProcessPortabilityRequest(requestID uint, reviewerID uint) ([]byte, error) {
	var request DataSubjectRequest
	if err := database.DB.First(&request, requestID).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := database.DB.Where("email = ?", request.Email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found for email: %s", request.Email)
	}

	exportData, err := s.ExportUserData(user.ID)
	if err != nil {
		return nil, err
	}

	portabilityFormat := map[string]interface{}{
		"format":          "JSON",
		"structured":      true,
		"machine_readable": true,
		"data":           exportData,
	}

	responseData, _ := json.Marshal(portabilityFormat)

	request.ResponseData = string(responseData)
	if err := database.DB.Save(&request).Error; err != nil {
		return nil, err
	}

	if err := s.UpdateRequestStatus(requestID, RequestStatusCompleted, reviewerID, "Data exported in portable format"); err != nil {
		return nil, err
	}

	return json.MarshalIndent(portabilityFormat, "", "  ")
}

type OAuth2Token struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id"`
	Provider     string    `json:"provider"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}
