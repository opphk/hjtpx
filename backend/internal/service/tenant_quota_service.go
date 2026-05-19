package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/database"
	"gorm.io/gorm"
)

// TenantQuotaService 租户配额服务接口
type TenantQuotaService interface {
	GetTenantQuota(ctx context.Context, tenantID uint, resourceType string) (*model.TenantQuota, error)
	UpdateTenantQuota(ctx context.Context, tenantID uint, resourceType string, limit int64) error
	ConsumeQuota(ctx context.Context, tenantID uint, resourceType string, amount int64) (bool, *model.TenantQuota, error)
	CheckQuota(ctx context.Context, tenantID uint, resourceType string, amount int64) (bool, error)
	ListTenantQuotas(ctx context.Context, tenantID uint) ([]*model.TenantQuota, error)
	ResetQuotas(ctx context.Context, tenantID uint) error
	ResetExpiredQuotas(ctx context.Context) error
}

// tenantQuotaService 租户配额服务实现
type tenantQuotaService struct {
	db *gorm.DB
}

// NewTenantQuotaService 创建租户配额服务实例
func NewTenantQuotaService() TenantQuotaService {
	return &tenantQuotaService{
		db: database.DB,
	}
}

// GetTenantQuota 获取租户配额
func (s *tenantQuotaService) GetTenantQuota(ctx context.Context, tenantID uint, resourceType string) (*model.TenantQuota, error) {
	var quota model.TenantQuota
	if err := s.db.Where("tenant_id = ? AND resource_type = ? AND is_active = ?", tenantID, resourceType, true).First(&quota).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &quota, nil
}

// UpdateTenantQuota 更新租户配额
func (s *tenantQuotaService) UpdateTenantQuota(ctx context.Context, tenantID uint, resourceType string, limit int64) error {
	var quota model.TenantQuota
	if err := s.db.Where("tenant_id = ? AND resource_type = ?", tenantID, resourceType).First(&quota).Error; err != nil {
		return err
	}

	quota.Limit = limit
	quota.Remaining = limit - quota.Used
	quota.UpdatedAt = time.Now()

	return s.db.Save(&quota).Error
}

// ConsumeQuota 消费配额
func (s *tenantQuotaService) ConsumeQuota(ctx context.Context, tenantID uint, resourceType string, amount int64) (bool, *model.TenantQuota, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var quota model.TenantQuota
	if err := tx.Where("tenant_id = ? AND resource_type = ? AND is_active = ?", tenantID, resourceType, true).First(&quota).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return false, nil, fmt.Errorf("quota not found for tenant %d and resource %s", tenantID, resourceType)
		}
		return false, nil, err
	}

	if time.Now().After(quota.ResetAt) {
		quota.Used = 0
		quota.Remaining = quota.Limit
		quota.ResetAt = calculateNextResetTime(quota.PeriodType)
	}

	if quota.HardLimit && quota.Remaining < amount {
		tx.Rollback()
		return false, &quota, nil
	}

	quota.Used += amount
	quota.Remaining -= amount
	quota.LastConsumedAt = func() *time.Time { t := time.Now(); return &t }()
	quota.UpdatedAt = time.Now()

	if err := tx.Save(&quota).Error; err != nil {
		tx.Rollback()
		return false, nil, err
	}

	if err := s.recordUsage(ctx, tx, tenantID, resourceType, amount); err != nil {
		tx.Rollback()
		return false, nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return false, nil, err
	}

	return true, &quota, nil
}

// CheckQuota 检查配额是否足够
func (s *tenantQuotaService) CheckQuota(ctx context.Context, tenantID uint, resourceType string, amount int64) (bool, error) {
	quota, err := s.GetTenantQuota(ctx, tenantID, resourceType)
	if err != nil {
		return false, err
	}

	if quota == nil {
		return false, fmt.Errorf("quota not found")
	}

	if time.Now().After(quota.ResetAt) {
		return quota.Limit >= amount, nil
	}

	return quota.Remaining >= amount, nil
}

// ListTenantQuotas 列出租户配额
func (s *tenantQuotaService) ListTenantQuotas(ctx context.Context, tenantID uint) ([]*model.TenantQuota, error) {
	var quotas []*model.TenantQuota
	if err := s.db.Where("tenant_id = ?", tenantID).Find(&quotas).Error; err != nil {
		return nil, err
	}
	return quotas, nil
}

// ResetQuotas 重置租户所有配额
func (s *tenantQuotaService) ResetQuotas(ctx context.Context, tenantID uint) error {
	return s.db.Model(&model.TenantQuota{}).Where("tenant_id = ?", tenantID).Updates(map[string]interface{}{
		"used":      0,
		"remaining": gorm.Expr("limit"),
		"reset_at":  calculateNextMonthStart(),
	}).Error
}

// ResetExpiredQuotas 重置所有过期的配额
func (s *tenantQuotaService) ResetExpiredQuotas(ctx context.Context) error {
	return s.db.Model(&model.TenantQuota{}).Where("reset_at < ?", time.Now()).Updates(map[string]interface{}{
		"used":      0,
		"remaining": gorm.Expr("limit"),
		"reset_at":  calculateNextMonthStart(),
	}).Error
}

// recordUsage 记录使用量
func (s *tenantQuotaService) recordUsage(ctx context.Context, tx *gorm.DB, tenantID uint, resourceType string, amount int64) error {
	usageDate := time.Now().Format("2006-01-02")

	var usage model.TenantUsage
	if err := tx.Where("tenant_id = ? AND resource_type = ? AND usage_date = ?", tenantID, resourceType, usageDate).First(&usage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			usage = model.TenantUsage{
				TenantID:     tenantID,
				ResourceType: resourceType,
				UsageDate:    usageDate,
				UsageCount:   amount,
				Unit:         "count",
			}
			return tx.Create(&usage).Error
		}
		return err
	}

	usage.UsageCount += amount
	return tx.Save(&usage).Error
}

// calculateNextResetTime 计算下次重置时间
func calculateNextResetTime(periodType string) time.Time {
	now := time.Now()
	switch periodType {
	case "daily":
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	case "weekly":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		daysUntilMonday := 8 - weekday
		return time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 0, 0, 0, 0, now.Location())
	case "monthly":
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	case "yearly":
		return time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	default:
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	}
}
