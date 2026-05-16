package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

var (
	ErrApplicationNotFound = errors.New("application not found")
	ErrUserNotFoundApp     = errors.New("user not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrKeyGeneration      = errors.New("failed to generate API key")
)

type ApplicationService struct{}

func NewApplicationService() *ApplicationService {
	return &ApplicationService{}
}

type CreateApplicationInput struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	UserID      uint   `json:"user_id" binding:"required"`
	Description string `json:"description" binding:"max=1000"`
	Domain      string `json:"domain" binding:"max=255"`
	Website     string `json:"website" binding:"max=255"`
}

type UpdateApplicationInput struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	IsActive    *bool   `json:"is_active"`
	Domain      *string `json:"domain" binding:"omitempty,max=255"`
	Website     *string `json:"website" binding:"omitempty,max=255"`
}

type ApplicationConfig struct {
	CaptchaTypes         []string               `json:"captcha_types"`
	MaxVerifyPerMinute   int                    `json:"max_verify_per_minute"`
	MaxVerifyPerDay      int                    `json:"max_verify_per_day"`
	AllowedIPs           []string               `json:"allowed_ips"`
	BlockRefusedRequests bool                   `json:"block_refused_requests"`
	CustomSettings       map[string]interface{} `json:"custom_settings"`
}

type ListApplicationsFilter struct {
	Page      int
	PageSize  int
	Keyword   string
	UserID    uint
	IsActive  *bool
	SortField string
	SortOrder string
}

type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type ApplicationResponse struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	UserID      uint               `json:"user_id"`
	Description string             `json:"description"`
	APIKey      string             `json:"api_key"`
	Domain      string             `json:"domain"`
	Website     string             `json:"website"`
	IsActive    bool               `json:"is_active"`
	Config      *ApplicationConfig `json:"config,omitempty"`
	User        *UserResponse      `json:"user,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", ErrKeyGeneration
	}
	return hex.EncodeToString(bytes), nil
}

func (s *ApplicationService) CreateApplication(input *CreateApplicationInput) (*models.Application, error) {
	if input.Name == "" {
		return nil, ErrInvalidInput
	}

	var user models.User
	if err := database.DB.First(&user, input.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFoundApp
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	defaultConfig := ApplicationConfig{
		CaptchaTypes:         []string{"slider", "click"},
		MaxVerifyPerMinute:   60,
		MaxVerifyPerDay:      5000,
		AllowedIPs:           []string{},
		BlockRefusedRequests: false,
		CustomSettings:       map[string]interface{}{},
	}

	defaultConfigJSON, err := json.Marshal(defaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	application := &models.Application{
		Name:        input.Name,
		UserID:      input.UserID,
		Description: input.Description,
		APIKey:      apiKey,
		Domain:      input.Domain,
		Website:     input.Website,
		IsActive:    true,
		Config:      string(defaultConfigJSON),
	}

	if err := database.DB.Create(application).Error; err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	database.DB.Preload("User").First(application, application.ID)
	return application, nil
}

func (s *ApplicationService) GetApplicationByID(id uint) (*models.Application, error) {
	var application models.Application
	if err := database.DB.Preload("User").First(&application, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}
	return &application, nil
}

func (s *ApplicationService) GetApplicationByAPIKey(apiKey string) (*models.Application, error) {
	var application models.Application
	if err := database.DB.Where("api_key = ?", apiKey).First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}
	return &application, nil
}

func (s *ApplicationService) ListApplications(filter *ListApplicationsFilter) (*PaginatedResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	db := database.DB.Model(&models.Application{})

	if filter.Keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ? OR domain LIKE ?",
			"%"+filter.Keyword+"%", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	if filter.UserID > 0 {
		db = db.Where("user_id = ?", filter.UserID)
	}

	if filter.IsActive != nil {
		db = db.Where("is_active = ?", *filter.IsActive)
	}

	sortField := "created_at"
	sortOrder := "DESC"
	if filter.SortField != "" {
		sortField = filter.SortField
	}
	if filter.SortOrder == "ASC" {
		sortOrder = "ASC"
	}
	db = db.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count applications: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	var applications []models.Application
	if err := db.Preload("User").Offset(offset).Limit(filter.PageSize).Find(&applications).Error; err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       applications,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ApplicationService) UpdateApplication(id uint, input *UpdateApplicationInput) (*models.Application, error) {
	application, err := s.GetApplicationByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if input.Name != nil && *input.Name != "" {
		updates["name"] = *input.Name
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}
	if input.Domain != nil {
		updates["domain"] = *input.Domain
	}
	if input.Website != nil {
		updates["website"] = *input.Website
	}

	if len(updates) > 0 {
		if err := database.DB.Model(application).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update application: %w", err)
		}
	}

	database.DB.Preload("User").First(application, id)
	return application, nil
}

func (s *ApplicationService) DeleteApplication(id uint) error {
	application, err := s.GetApplicationByID(id)
	if err != nil {
		return err
	}

	if err := database.DB.Delete(application).Error; err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	return nil
}

func (s *ApplicationService) RegenerateAPIKey(id uint) (*models.Application, string, error) {
	application, err := s.GetApplicationByID(id)
	if err != nil {
		return nil, "", err
	}

	oldKey := application.APIKey

	newKey, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}

	if err := database.DB.Model(application).Update("api_key", newKey).Error; err != nil {
		return nil, "", fmt.Errorf("failed to regenerate API key: %w", err)
	}

	application.APIKey = newKey

	historyRecord := models.APIKeyHistory{
		ApplicationID: application.ID,
		OldAPIKey:     oldKey,
		NewAPIKey:     newKey,
		ChangedAt:     time.Now(),
	}
	database.DB.Create(&historyRecord)

	return application, oldKey, nil
}

func (s *ApplicationService) GetApplicationConfig(id uint) (*ApplicationConfig, error) {
	application, err := s.GetApplicationByID(id)
	if err != nil {
		return nil, err
	}

	var config ApplicationConfig
	if application.Config == "" || application.Config == "{}" {
		config = ApplicationConfig{
			CaptchaTypes:         []string{"slider", "click"},
			MaxVerifyPerMinute:   60,
			MaxVerifyPerDay:      5000,
			AllowedIPs:           []string{},
			BlockRefusedRequests: false,
			CustomSettings:       map[string]interface{}{},
		}
		return &config, nil
	}

	if err := json.Unmarshal([]byte(application.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func (s *ApplicationService) UpdateApplicationConfig(id uint, config *ApplicationConfig) (*models.Application, error) {
	application, err := s.GetApplicationByID(id)
	if err != nil {
		return nil, err
	}

	if config.CaptchaTypes == nil {
		config.CaptchaTypes = []string{"slider", "click"}
	}
	if config.MaxVerifyPerMinute <= 0 {
		config.MaxVerifyPerMinute = 60
	}
	if config.MaxVerifyPerDay <= 0 {
		config.MaxVerifyPerDay = 5000
	}
	if config.AllowedIPs == nil {
		config.AllowedIPs = []string{}
	}
	if config.CustomSettings == nil {
		config.CustomSettings = map[string]interface{}{}
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := database.DB.Model(application).Update("config", string(configJSON)).Error; err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	application.Config = string(configJSON)
	return application, nil
}

func (s *ApplicationService) GetApplicationsByUserID(userID uint) ([]models.Application, error) {
	var applications []models.Application
	if err := database.DB.Where("user_id = ?", userID).Find(&applications).Error; err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}
	return applications, nil
}

func (s *ApplicationService) GetApplicationStatistics(id uint) (*ApplicationStatistics, error) {
	application, err := s.GetApplicationByID(id)
	if err != nil {
		return nil, err
	}

	var totalVerifications int64
	var passedVerifications int64
	var failedVerifications int64

	database.DB.Model(&models.Verification{}).
		Where("application_id = ?", id).
		Count(&totalVerifications)

	database.DB.Model(&models.Verification{}).
		Where("application_id = ? AND status = ?", id, "passed").
		Count(&passedVerifications)

	database.DB.Model(&models.Verification{}).
		Where("application_id = ? AND status = ?", id, "failed").
		Count(&failedVerifications)

	today := time.Now().Truncate(24 * time.Hour)
	var todayVerifications int64
	database.DB.Model(&models.Verification{}).
		Where("application_id = ? AND created_at >= ?", id, today).
		Count(&todayVerifications)

	var recentVerifications []models.Verification
	database.DB.Where("application_id = ?", id).
		Order("created_at DESC").
		Limit(10).
		Find(&recentVerifications)

	passRate := 0.0
	if totalVerifications > 0 {
		passRate = float64(passedVerifications) / float64(totalVerifications) * 100
	}

	return &ApplicationStatistics{
		TotalVerifications:   totalVerifications,
		PassedVerifications:   passedVerifications,
		FailedVerifications:  failedVerifications,
		PassRate:             passRate,
		TodayVerifications:   todayVerifications,
		Application:          application,
		RecentVerifications:  recentVerifications,
	}, nil
}

type ApplicationStatistics struct {
	TotalVerifications   int64                `json:"total_verifications"`
	PassedVerifications   int64                `json:"passed_verifications"`
	FailedVerifications  int64                `json:"failed_verifications"`
	PassRate             float64              `json:"pass_rate"`
	TodayVerifications   int64                `json:"today_verifications"`
	Application          *models.Application  `json:"application"`
	RecentVerifications  []models.Verification `json:"recent_verifications"`
}

func ToApplicationResponse(app *models.Application) *ApplicationResponse {
	resp := &ApplicationResponse{
		ID:          app.ID,
		Name:        app.Name,
		UserID:      app.UserID,
		Description: app.Description,
		APIKey:      app.APIKey,
		Domain:      app.Domain,
		Website:     app.Website,
		IsActive:    app.IsActive,
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
	}

	if app.User.ID != 0 {
		resp.User = &UserResponse{
			ID:       app.User.ID,
			Username: app.User.Username,
			Email:    app.User.Email,
		}
	}

	if app.Config != "" && app.Config != "{}" {
		var config ApplicationConfig
		if err := json.Unmarshal([]byte(app.Config), &config); err == nil {
			resp.Config = &config
		}
	}

	return resp
}
