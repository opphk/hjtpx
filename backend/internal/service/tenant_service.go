package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantService 租户服务接口
type TenantService interface {
	CreateTenant(ctx context.Context, tenant *model.Tenant) error
	GetTenant(ctx context.Context, id uint) (*model.Tenant, error)
	GetTenantByDomain(ctx context.Context, domain string) (*model.Tenant, error)
	GetTenantBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error)
	UpdateTenant(ctx context.Context, tenant *model.Tenant) error
	DeleteTenant(ctx context.Context, id uint) error
	ListTenants(ctx context.Context, status string, page, pageSize int) ([]*model.Tenant, int64, error)
	AddTenantUser(ctx context.Context, tenantID, userID uint, role string) error
	RemoveTenantUser(ctx context.Context, tenantID, userID uint) error
	UpdateTenantUserRole(ctx context.Context, tenantID, userID uint, role string) error
	ListTenantUsers(ctx context.Context, tenantID uint) ([]*model.TenantUser, error)
	CreateInvitation(ctx context.Context, tenantID uint, email, role string, invitedBy uint) (*model.TenantInvitation, error)
	AcceptInvitation(ctx context.Context, token string, userID uint) error
	RevokeInvitation(ctx context.Context, invitationID uint) error
}

// tenantService 租户服务实现
type tenantService struct {
	db *gorm.DB
}

// NewTenantService 创建租户服务实例
func NewTenantService() TenantService {
	return &tenantService{
		db: database.DB,
	}
}

// CreateTenant 创建租户
func (s *tenantService) CreateTenant(ctx context.Context, tenant *model.Tenant) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()
	tenant.Status = "active"

	if tenant.Subdomain == "" {
		tenant.Subdomain = generateSubdomain(tenant.Name)
	}

	if err := tx.Create(tenant).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := s.createDefaultQuotas(ctx, tx, tenant.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.createDefaultFeatures(ctx, tx, tenant.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// createDefaultQuotas 创建默认配额
func (s *tenantService) createDefaultQuotas(ctx context.Context, tx *gorm.DB, tenantID uint) error {
	defaultQuotas := []model.TenantQuota{
		{
			TenantID:         tenantID,
			ResourceType:     "verification",
			Limit:            10000,
			Used:             0,
			Remaining:        10000,
			WarningThreshold: 80,
			HardLimit:        true,
			PeriodType:       "monthly",
			ResetAt:          calculateNextMonthStart(),
			IsActive:         true,
		},
		{
			TenantID:         tenantID,
			ResourceType:     "api_call",
			Limit:            100000,
			Used:             0,
			Remaining:        100000,
			WarningThreshold: 80,
			HardLimit:        true,
			PeriodType:       "monthly",
			ResetAt:          calculateNextMonthStart(),
			IsActive:         true,
		},
		{
			TenantID:         tenantID,
			ResourceType:     "storage",
			Limit:            1024 * 1024 * 1024, // 1GB
			Used:             0,
			Remaining:        1024 * 1024 * 1024,
			WarningThreshold: 80,
			HardLimit:        true,
			PeriodType:       "monthly",
			ResetAt:          calculateNextMonthStart(),
			IsActive:         true,
		},
	}

	return tx.Create(&defaultQuotas).Error
}

// createDefaultFeatures 创建默认功能开关
func (s *tenantService) createDefaultFeatures(ctx context.Context, tx *gorm.DB, tenantID uint) error {
	defaultFeatures := []model.TenantFeature{
		{TenantID: tenantID, FeatureKey: "slider_captcha", IsEnabled: true, Description: "滑块验证码"},
		{TenantID: tenantID, FeatureKey: "click_captcha", IsEnabled: true, Description: "点选验证码"},
		{TenantID: tenantID, FeatureKey: "gesture_captcha", IsEnabled: true, Description: "手势验证码"},
		{TenantID: tenantID, FeatureKey: "smart_captcha", IsEnabled: true, Description: "智能验证码"},
		{TenantID: tenantID, FeatureKey: "risk_analysis", IsEnabled: true, Description: "风险分析"},
		{TenantID: tenantID, FeatureKey: "api_access", IsEnabled: true, Description: "API访问"},
		{TenantID: tenantID, FeatureKey: "ab_testing", IsEnabled: false, Description: "A/B测试"},
		{TenantID: tenantID, FeatureKey: "advanced_analytics", IsEnabled: false, Description: "高级分析"},
	}

	return tx.Create(&defaultFeatures).Error
}

// GetTenant 根据ID获取租户
func (s *tenantService) GetTenant(ctx context.Context, id uint) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := s.db.First(&tenant, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

// GetTenantByDomain 根据域名获取租户
func (s *tenantService) GetTenantByDomain(ctx context.Context, domain string) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := s.db.Where("domain = ? AND status = ?", domain, "active").First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

// GetTenantBySubdomain 根据子域名获取租户
func (s *tenantService) GetTenantBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error) {
	var tenant model.Tenant
	if err := s.db.Where("subdomain = ? AND status = ?", subdomain, "active").First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

// UpdateTenant 更新租户
func (s *tenantService) UpdateTenant(ctx context.Context, tenant *model.Tenant) error {
	tenant.UpdatedAt = time.Now()
	return s.db.Save(tenant).Error
}

// DeleteTenant 删除租户（软删除）
func (s *tenantService) DeleteTenant(ctx context.Context, id uint) error {
	return s.db.Model(&model.Tenant{}).Where("id = ?", id).Update("status", "deleted").Error
}

// ListTenants 列出租户
func (s *tenantService) ListTenants(ctx context.Context, status string, page, pageSize int) ([]*model.Tenant, int64, error) {
	var tenants []*model.Tenant
	var total int64

	query := s.db.Model(&model.Tenant{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

// AddTenantUser 添加租户用户
func (s *tenantService) AddTenantUser(ctx context.Context, tenantID, userID uint, role string) error {
	tenantUser := model.TenantUser{
		TenantID:  tenantID,
		UserID:    userID,
		Role:      role,
		Status:    "active",
		JoinedAt:  time.Now(),
	}
	return s.db.Create(&tenantUser).Error
}

// RemoveTenantUser 移除租户用户
func (s *tenantService) RemoveTenantUser(ctx context.Context, tenantID, userID uint) error {
	return s.db.Delete(&model.TenantUser{}, "tenant_id = ? AND user_id = ?", tenantID, userID).Error
}

// UpdateTenantUserRole 更新租户用户角色
func (s *tenantService) UpdateTenantUserRole(ctx context.Context, tenantID, userID uint, role string) error {
	return s.db.Model(&model.TenantUser{}).Where("tenant_id = ? AND user_id = ?", tenantID, userID).Update("role", role).Error
}

// ListTenantUsers 列出租户用户
func (s *tenantService) ListTenantUsers(ctx context.Context, tenantID uint) ([]*model.TenantUser, error) {
	var users []*model.TenantUser
	if err := s.db.Where("tenant_id = ?", tenantID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// CreateInvitation 创建邀请
func (s *tenantService) CreateInvitation(ctx context.Context, tenantID uint, email, role string, invitedBy uint) (*model.TenantInvitation, error) {
	invitation := model.TenantInvitation{
		TenantID:  tenantID,
		Email:     email,
		Token:     uuid.New().String(),
		Role:      role,
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		InvitedBy: invitedBy,
	}

	if err := s.db.Create(&invitation).Error; err != nil {
		return nil, err
	}

	return &invitation, nil
}

// AcceptInvitation 接受邀请
func (s *tenantService) AcceptInvitation(ctx context.Context, token string, userID uint) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var invitation model.TenantInvitation
	if err := tx.Where("token = ? AND status = ?", token, "pending").First(&invitation).Error; err != nil {
		tx.Rollback()
		return err
	}

	if time.Now().After(invitation.ExpiresAt) {
		tx.Rollback()
		return fmt.Errorf("invitation expired")
	}

	if err := tx.Create(&model.TenantUser{
		TenantID:  invitation.TenantID,
		UserID:    userID,
		Role:      invitation.Role,
		Status:    "active",
		JoinedAt:  time.Now(),
		InvitedBy: invitation.InvitedBy,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&invitation).Update("status", "accepted").Update("accepted_at", time.Now()).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RevokeInvitation 撤销邀请
func (s *tenantService) RevokeInvitation(ctx context.Context, invitationID uint) error {
	return s.db.Model(&model.TenantInvitation{}).Where("id = ?", invitationID).Update("status", "revoked").Error
}

// generateSubdomain 生成子域名
func generateSubdomain(name string) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s-%d", name, rand.Intn(10000))
}

// calculateNextMonthStart 计算下月初时间
func calculateNextMonthStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
}

// TenantContextKey 租户上下文键
type TenantContextKey struct{}

// GetTenantFromContext 从上下文获取租户
func GetTenantFromContext(ctx context.Context) (*model.Tenant, bool) {
	tenant, ok := ctx.Value(TenantContextKey{}).(*model.Tenant)
	return tenant, ok
}
