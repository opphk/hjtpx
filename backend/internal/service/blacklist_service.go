package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type BlacklistService struct{}

func NewBlacklistService() *BlacklistService {
	return &BlacklistService{}
}

var (
	ErrBlacklistNotFound = errors.New("blacklist item not found")
)

type BlacklistType string

const (
	BlacklistTypeIP       BlacklistType = "ip"
	BlacklistTypeUserID   BlacklistType = "user_id"
	BlacklistTypeDeviceID BlacklistType = "device_id"
	BlacklistTypePhone    BlacklistType = "phone"
	BlacklistTypeEmail    BlacklistType = "email"
)

type BlacklistSource string

const (
	BlacklistSourceManual BlacklistSource = "manual"
	BlacklistSourceAuto   BlacklistSource = "auto"
	BlacklistSourceImport BlacklistSource = "import"
)

type BlacklistAction string

const (
	BlacklistActionBlock   BlacklistAction = "block"
	BlacklistActionCaptcha BlacklistAction = "captcha"
	BlacklistActionReview  BlacklistAction = "review"
)

type BlacklistStatus string

const (
	BlacklistStatusActive    BlacklistStatus = "active"
	BlacklistStatusExpired   BlacklistStatus = "expired"
	BlacklistStatusUnblocked BlacklistStatus = "unblocked"
)

type ListBlacklistFilter struct {
	Page          int
	PageSize      int
	Type          string
	Source        string
	Status        string
	Keyword       string
	StartDate     time.Time
	EndDate       time.Time
	ApplicationID uint
}

type BlacklistSummary struct {
	Total         int64 `json:"total"`
	TodayAdded    int64 `json:"today_added"`
	AutoUnblocked int64 `json:"auto_unblocked"`
	TotalBlocked  int64 `json:"total_blocked"`
}

func (s *BlacklistService) ListBlacklist(filter *ListBlacklistFilter) (*PaginatedResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	query := database.DB.Model(&models.Blacklist{})

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else {
		query = query.Where("status = ?", string(BlacklistStatusActive))
	}
	if filter.Keyword != "" {
		query = query.Where("target LIKE ?", "%"+filter.Keyword+"%")
	}
	if !filter.StartDate.IsZero() {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	var items []models.Blacklist
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       items,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *BlacklistService) GetBlacklistByID(id uint) (*models.Blacklist, error) {
	var item models.Blacklist
	if err := database.DB.First(&item, id).Error; err != nil {
		return nil, ErrBlacklistNotFound
	}
	return &item, nil
}

type CreateBlacklistInput struct {
	Target         string   `json:"target" binding:"required"`
	Type           string   `json:"type" binding:"required"`
	Source         string   `json:"source"`
	Reason         string   `json:"reason"`
	Action         string   `json:"action"`
	ApplicationIDs []string `json:"application_ids"`
	Expiration     string   `json:"expiration"`
	Note           string   `json:"note"`
	CreatedBy      uint     `json:"created_by"`
}

func (s *BlacklistService) CreateBlacklist(input *CreateBlacklistInput) (*models.Blacklist, error) {
	if input.Target == "" || input.Type == "" {
		return nil, errors.New("invalid input")
	}

	item := &models.Blacklist{
		Target:     input.Target,
		Type:       input.Type,
		Source:     input.Source,
		Reason:     input.Reason,
		Action:     input.Action,
		Expiration: input.Expiration,
		Status:     string(BlacklistStatusActive),
		Note:       input.Note,
		CreatedBy:  input.CreatedBy,
		HitCount:   0,
	}

	if item.Source == "" {
		item.Source = string(BlacklistSourceManual)
	}
	if item.Action == "" {
		item.Action = string(BlacklistActionBlock)
	}

	if len(input.ApplicationIDs) > 0 {
		appsJSON, err := json.Marshal(input.ApplicationIDs)
		if err == nil {
			item.ApplicationIDs = string(appsJSON)
		}
	}

	if err := database.DB.Create(item).Error; err != nil {
		return nil, err
	}

	if redis.Client != nil {
		ctx := context.Background()
		blacklistKey := "blacklist:" + item.Type + ":" + item.Target
		if item.Expiration == "permanent" || item.Expiration == "" {
			redis.Client.Set(ctx, blacklistKey, "1", 0)
		} else {
			if expTime, err := time.Parse("2006-01-02", item.Expiration); err == nil {
				duration := time.Until(expTime)
				if duration > 0 {
					redis.Client.Set(ctx, blacklistKey, "1", duration)
				}
			}
		}
	}

	return item, nil
}

type UpdateBlacklistInput struct {
	Type           *string  `json:"type"`
	Reason         *string  `json:"reason"`
	Action         *string  `json:"action"`
	ApplicationIDs []string `json:"application_ids"`
	Expiration     *string  `json:"expiration"`
	Note           *string  `json:"note"`
}

func (s *BlacklistService) UpdateBlacklist(id uint, input *UpdateBlacklistInput) (*models.Blacklist, error) {
	item, err := s.GetBlacklistByID(id)
	if err != nil {
		return nil, err
	}

	if input.Type != nil {
		item.Type = *input.Type
	}
	if input.Reason != nil {
		item.Reason = *input.Reason
	}
	if input.Action != nil {
		item.Action = *input.Action
	}
	if input.Expiration != nil {
		item.Expiration = *input.Expiration
	}
	if input.Note != nil {
		item.Note = *input.Note
	}
	if len(input.ApplicationIDs) > 0 {
		appsJSON, err := json.Marshal(input.ApplicationIDs)
		if err == nil {
			item.ApplicationIDs = string(appsJSON)
		}
	}

	if err := database.DB.Save(item).Error; err != nil {
		return nil, err
	}

	return item, nil
}

func (s *BlacklistService) DeleteBlacklist(id uint) error {
	item, err := s.GetBlacklistByID(id)
	if err != nil {
		return err
	}

	if redis.Client != nil {
		ctx := context.Background()
		blacklistKey := "blacklist:" + item.Type + ":" + item.Target
		redis.Client.Del(ctx, blacklistKey)
	}

	return database.DB.Delete(item).Error
}

func (s *BlacklistService) UnblockBlacklist(id uint) (*models.Blacklist, error) {
	item, err := s.GetBlacklistByID(id)
	if err != nil {
		return nil, err
	}

	item.Status = string(BlacklistStatusUnblocked)

	if err := database.DB.Save(item).Error; err != nil {
		return nil, err
	}

	if redis.Client != nil {
		ctx := context.Background()
		blacklistKey := "blacklist:" + item.Type + ":" + item.Target
		redis.Client.Del(ctx, blacklistKey)
	}

	return item, nil
}

func (s *BlacklistService) IncrementHitCount(id uint) error {
	return database.DB.Model(&models.Blacklist{}).Where("id = ?", id).UpdateColumn("hit_count", database.DB.Raw("hit_count + 1")).Error
}

func (s *BlacklistService) GetBlacklistSummary() (*BlacklistSummary, error) {
	summary := &BlacklistSummary{}

	if err := database.DB.Model(&models.Blacklist{}).Count(&summary.Total).Error; err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	if err := database.DB.Model(&models.Blacklist{}).Where("created_at >= ?", today).Count(&summary.TodayAdded).Error; err != nil {
		return nil, err
	}

	if err := database.DB.Model(&models.Blacklist{}).Where("status = ?", string(BlacklistStatusUnblocked)).Count(&summary.AutoUnblocked).Error; err != nil {
		return nil, err
	}

	summary.TotalBlocked = summary.Total

	return summary, nil
}

func (s *BlacklistService) BatchCreateBlacklist(items []CreateBlacklistInput) (int, error) {
	count := 0
	for _, item := range items {
		_, err := s.CreateBlacklist(&item)
		if err == nil {
			count++
		}
	}
	return count, nil
}

func (s *BlacklistService) CheckBlacklist(target, blType string) (bool, error) {
	var item models.Blacklist
	err := database.DB.Where("target = ? AND type = ? AND status = ?", target, blType, string(BlacklistStatusActive)).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if item.Status != string(BlacklistStatusActive) {
		return false, nil
	}

	if item.Expiration != "" && item.Expiration != "permanent" {
		if expTime, err := time.Parse("2006-01-02", item.Expiration); err == nil {
			if time.Now().After(expTime) {
				return false, nil
			}
		}
	}

	return true, nil
}
