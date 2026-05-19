package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type TenantService struct {
	db           *gorm.DB
	cacheService TenantCacheService
}

type TenantCacheService interface {
	Get(ctx context.Context, key string) TenantCacheResult
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, key string) error
}

type TenantCacheResult interface {
	Result() (string, error)
}

func NewTenantService(db *gorm.DB, cacheService interface{}) *TenantService {
	var cs TenantCacheService
	if cacheService != nil {
		if typedCs, ok := cacheService.(TenantCacheService); ok {
			cs = typedCs
		}
	}
	return &TenantService{
		db:           db,
		cacheService: cs,
	}
}

type TenantContext struct {
	TenantID   uint
	TenantCode string
	TenantName string
	IsolatedDB bool
	IsolatedCache bool
	Plan       string
	Quota      *models.TenantQuota
}

func (s *TenantService) GetTenantContext(tenantID uint) (*TenantContext, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("tenant:context:%d", tenantID)

	if s.cacheService != nil {
		cacheResult := s.cacheService.Get(ctx, cacheKey)
		cached, err := cacheResult.Result()
		if err == nil && cached != "" {
			var tc TenantContext
			if json.Unmarshal([]byte(cached), &tc) == nil {
				return &tc, nil
			}
		}
	}

	var tenant models.Tenant
	if err := s.db.Preload("Quota").First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}

	tc := &TenantContext{
		TenantID:       tenant.ID,
		TenantCode:     tenant.Code,
		TenantName:     tenant.Name,
		IsolatedDB:     tenant.IsolatedDB,
		IsolatedCache:  tenant.IsolatedCache,
		Plan:           tenant.Plan,
	}

	if s.cacheService != nil {
		if data, err := json.Marshal(tc); err == nil {
			s.cacheService.Set(ctx, cacheKey, string(data), 10*time.Minute)
		}
	}

	return tc, nil
}

func (s *TenantService) CreateTenant(tenant *models.Tenant, creatorID uint) (*models.Tenant, error) {
	if err := s.db.Create(tenant).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	monthLater := now.AddDate(0, 1, 0)
	
	quota := &models.TenantQuota{
		TenantID:    tenant.ID,
		MaxUsers:    10,
		MaxApplications: 5,
		MaxAPIRequests: 100000,
		MaxStorage:  10737418240,
		MaxBandwidth: 10737418240,
		MaxWebhooks: 10,
		MaxRules:    50,
		MaxABTests:  5,
		PeriodStart: &now,
		PeriodEnd:   &monthLater,
	}
	if err := s.db.Create(quota).Error; err != nil {
		return nil, err
	}

	billing := &models.TenantBilling{
		TenantID:     tenant.ID,
		Plan:         tenant.Plan,
		BillingCycle: "monthly",
		Price:        0,
	}
	if err := s.db.Create(billing).Error; err != nil {
		return nil, err
	}

	s.logTenantAction(tenant.ID, creatorID, "create", "tenant", fmt.Sprintf("%d", tenant.ID), "", "")

	return tenant, nil
}

func (s *TenantService) GetTenant(tenantID uint) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := s.db.Preload("Quota").Preload("Billing").First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *TenantService) GetTenantByCode(code string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := s.db.Preload("Quota").Preload("Billing").Where("code = ?", code).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *TenantService) ListTenants(page, pageSize int, status, plan, search string) ([]models.Tenant, int64, error) {
	var tenants []models.Tenant
	var total int64

	query := s.db.Model(&models.Tenant{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if plan != "" {
		query = query.Where("plan = ?", plan)
	}
	if search != "" {
		query = query.Where("name LIKE ? OR code LIKE ? OR contact_email LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("Quota").Preload("Billing").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

func (s *TenantService) UpdateTenant(tenantID uint, updates map[string]interface{}, adminID uint) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}

	oldData, _ := json.Marshal(tenant)

	allowedFields := []string{"name", "logo", "website", "contact_email", "contact_phone",
		"address", "description", "domain", "settings", "status"}

	updateData := make(map[string]interface{})
	for key, value := range updates {
		for _, allowed := range allowedFields {
			if key == allowed {
				updateData[key] = value
				break
			}
		}
	}

	if status, ok := updates["status"].(string); ok {
		if status == "suspended" && tenant.Status != "suspended" {
			now := time.Now()
			updateData["suspended_at"] = &now
		} else if status == "active" && tenant.Status == "suspended" {
			updateData["suspended_at"] = nil
		}
	}

	if err := s.db.Model(&tenant).Updates(updateData).Error; err != nil {
		return nil, err
	}

	newData, _ := json.Marshal(tenant)
	s.logTenantAction(tenantID, adminID, "update", "tenant", fmt.Sprintf("%d", tenantID),
		string(oldData), string(newData))

	if s.cacheService != nil {
		ctx := context.Background()
		s.cacheService.Del(ctx, fmt.Sprintf("tenant:context:%d", tenantID))
	}

	return &tenant, nil
}

func (s *TenantService) DeleteTenant(tenantID uint, adminID uint) error {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		return err
	}

	oldData, _ := json.Marshal(tenant)

	tx := s.db.Begin()

	if err := tx.Where("tenant_id = ?", tenantID).Delete(&models.TenantUser{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("tenant_id = ?", tenantID).Delete(&models.TenantQuota{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("tenant_id = ?", tenantID).Delete(&models.TenantBilling{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("tenant_id = ?", tenantID).Delete(&models.TenantInvitation{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&tenant).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	s.logTenantAction(tenantID, adminID, "delete", "tenant", fmt.Sprintf("%d", tenantID),
		string(oldData), "")

	if s.cacheService != nil {
		ctx := context.Background()
		s.cacheService.Del(ctx, fmt.Sprintf("tenant:context:%d", tenantID))
	}

	return nil
}

func (s *TenantService) AddTenantUser(tenantID, userID uint, role string, invitedBy uint) (*models.TenantUser, error) {
	var existing models.TenantUser
	err := s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("user already exists in tenant")
	}

	tc, err := s.GetTenantContext(tenantID)
	if err != nil {
		return nil, err
	}

	if tc.Quota != nil && tc.Quota.CurrentUsers >= tc.Quota.MaxUsers {
		return nil, fmt.Errorf("tenant user quota exceeded")
	}

	user := &models.TenantUser{
		TenantID:  tenantID,
		UserID:   userID,
		Role:     role,
		Status:   "active",
		InvitedBy: invitedBy,
		InvitedAt: func() *time.Time { t := time.Now(); return &t }(),
		JoinedAt:  func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	s.db.Model(&models.TenantQuota{}).Where("tenant_id = ?", tenantID).
		Update("current_users", gorm.Expr("current_users + 1"))

	return user, nil
}

func (s *TenantService) RemoveTenantUser(tenantID, userID uint) error {
	result := s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).Delete(&models.TenantUser{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found in tenant")
	}

	s.db.Model(&models.TenantQuota{}).Where("tenant_id = ?", tenantID).
		Update("current_users", gorm.Expr("GREATEST(current_users - 1, 0)"))

	return nil
}

func (s *TenantService) UpdateTenantUserRole(tenantID, userID uint, newRole string, adminID uint) error {
	var user models.TenantUser
	if err := s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&user).Error; err != nil {
		return err
	}

	oldRole := user.Role
	if err := s.db.Model(&user).Update("role", newRole).Error; err != nil {
		return err
	}

	s.logTenantAction(tenantID, adminID, "update_role", "tenant_user",
		fmt.Sprintf("%d-%d", tenantID, userID), oldRole, newRole)

	return nil
}

func (s *TenantService) ListTenantUsers(tenantID uint, page, pageSize int) ([]models.TenantUser, int64, error) {
	var users []models.TenantUser
	var total int64

	query := s.db.Model(&models.TenantUser{}).Where("tenant_id = ?", tenantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *TenantService) CreateInvitation(tenantID uint, email, role string, invitedBy uint) (*models.TenantInvitation, error) {
	var existing models.TenantInvitation
	err := s.db.Where("tenant_id = ? AND email = ? AND status = ?", tenantID, email, "pending").First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("pending invitation already exists for this email")
	}

	token := generateSecureTokenForTenant(32)
	invitation := &models.TenantInvitation{
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		Token:     token,
		Status:    "pending",
		InvitedBy: invitedBy,
		InvitedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.db.Create(invitation).Error; err != nil {
		return nil, err
	}

	return invitation, nil
}

func (s *TenantService) AcceptInvitation(token string, userID uint) (*models.TenantUser, error) {
	var invitation models.TenantInvitation
	if err := s.db.Where("token = ? AND status = ?", token, "pending").First(&invitation).Error; err != nil {
		return nil, fmt.Errorf("invalid or expired invitation")
	}

	if time.Now().After(invitation.ExpiresAt) {
		s.db.Model(&invitation).Update("status", "expired")
		return nil, fmt.Errorf("invitation has expired")
	}

	user, err := s.AddTenantUser(invitation.TenantID, userID, invitation.Role, invitation.InvitedBy)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	s.db.Model(&invitation).Updates(map[string]interface{}{
		"status":     "accepted",
		"accepted_at": &now,
	})

	return user, nil
}

func (s *TenantService) CheckQuota(tenantID uint, resourceType string) (bool, string) {
	tc, err := s.GetTenantContext(tenantID)
	if err != nil {
		return false, fmt.Sprintf("failed to get tenant context: %v", err)
	}

	if tc.Quota == nil {
		return false, "quota not configured"
	}

	switch resourceType {
	case "user":
		if tc.Quota.CurrentUsers >= tc.Quota.MaxUsers {
			return false, fmt.Sprintf("user quota exceeded: %d/%d", tc.Quota.CurrentUsers, tc.Quota.MaxUsers)
		}
	case "application":
		if tc.Quota.CurrentApps >= tc.Quota.MaxApplications {
			return false, fmt.Sprintf("application quota exceeded: %d/%d", tc.Quota.CurrentApps, tc.Quota.MaxApplications)
		}
	case "webhook":
		if tc.Quota.CurrentWebhooks >= tc.Quota.MaxWebhooks {
			return false, fmt.Sprintf("webhook quota exceeded: %d/%d", tc.Quota.CurrentWebhooks, tc.Quota.MaxWebhooks)
		}
	case "rule":
		if tc.Quota.CurrentRules >= tc.Quota.MaxRules {
			return false, fmt.Sprintf("rule quota exceeded: %d/%d", tc.Quota.CurrentRules, tc.Quota.MaxRules)
		}
	case "ab_test":
		if tc.Quota.CurrentABTests >= tc.Quota.MaxABTests {
			return false, fmt.Sprintf("A/B test quota exceeded: %d/%d", tc.Quota.CurrentABTests, tc.Quota.MaxABTests)
		}
	}

	return true, ""
}

func (s *TenantService) UpdateQuotaUsage(tenantID uint, resourceType string, increment int) error {
	field := ""
	switch resourceType {
	case "user":
		field = "current_users"
	case "application":
		field = "current_apps"
	case "webhook":
		field = "current_webhooks"
	case "rule":
		field = "current_rules"
	case "ab_test":
		field = "current_ab_tests"
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if increment > 0 {
		return s.db.Model(&models.TenantQuota{}).Where("tenant_id = ?", tenantID).
			Update(field, gorm.Expr(field+" + ?", increment)).Error
	}
	return s.db.Model(&models.TenantQuota{}).Where("tenant_id = ?", tenantID).
		Update(field, gorm.Expr("GREATEST("+field+" + ?, 0)", increment)).Error
}

func (s *TenantService) UpdateBillingPlan(tenantID uint, plan string, price float64, adminID uint) error {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		return err
	}

	oldPlan := tenant.Plan

	tx := s.db.Begin()

	if err := tx.Model(&tenant).Update("plan", plan).Error; err != nil {
		tx.Rollback()
		return err
	}

	quotaUpdates := getQuotaForPlan(plan)
	quotaUpdates["period_start"] = time.Now()
	quotaUpdates["period_end"] = time.Now().AddDate(0, 1, 0)

	if err := tx.Model(&models.TenantQuota{}).Where("tenant_id = ?", tenantID).Updates(quotaUpdates).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&models.TenantBilling{}).Where("tenant_id = ?", tenantID).Updates(map[string]interface{}{
		"plan":    plan,
		"price":   price,
		"status":  "active",
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	s.logTenantAction(tenantID, adminID, "update_plan", "tenant",
		fmt.Sprintf("%d", tenantID), oldPlan, plan)

	return nil
}

func getQuotaForPlan(plan string) map[string]interface{} {
	quotas := map[string]map[string]interface{}{
		"free": {
			"max_users":        10,
			"max_applications": 5,
			"max_api_requests": 100000,
			"max_storage":      int64(10737418240),
			"max_webhooks":     10,
			"max_rules":        50,
			"max_ab_tests":     5,
			"custom_branding":  false,
			"advanced_analytics": false,
			"sso_enabled":      false,
		},
		"starter": {
			"max_users":        50,
			"max_applications": 20,
			"max_api_requests": 1000000,
			"max_storage":      int64(53687091200),
			"max_webhooks":     50,
			"max_rules":        200,
			"max_ab_tests":     20,
			"custom_branding":  true,
			"advanced_analytics": true,
			"sso_enabled":      false,
		},
		"professional": {
			"max_users":        200,
			"max_applications": 100,
			"max_api_requests": 10000000,
			"max_storage":      int64(107374182400),
			"max_webhooks":     200,
			"max_rules":        500,
			"max_ab_tests":     50,
			"custom_branding":  true,
			"advanced_analytics": true,
			"sso_enabled":      true,
		},
		"enterprise": {
			"max_users":        -1,
			"max_applications": -1,
			"max_api_requests": -1,
			"max_storage":      int64(-1),
			"max_webhooks":     -1,
			"max_rules":        -1,
			"max_ab_tests":     -1,
			"custom_branding":  true,
			"advanced_analytics": true,
			"sso_enabled":      true,
		},
	}

	if q, ok := quotas[plan]; ok {
		return q
	}
	return quotas["free"]
}

func (s *TenantService) GetTenantUsageStats(tenantID uint) (map[string]interface{}, error) {
	var quota models.TenantQuota
	if err := s.db.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		return nil, err
	}

	var appCount int64
	s.db.Model(&models.Application{}).Where("tenant_id = ?", tenantID).Count(&appCount)

	var userCount int64
	s.db.Model(&models.TenantUser{}).Where("tenant_id = ?", tenantID).Count(&userCount)

	var webhookCount int64
	s.db.Model(&models.WebhookConfig{}).Where("tenant_id = ?", tenantID).Count(&webhookCount)

	var ruleCount int64
	s.db.Model(&models.RiskRule{}).Where("application_id IN (?)",
		s.db.Model(&models.Application{}).Select("id").Where("tenant_id = ?", tenantID),
	).Count(&ruleCount)

	var abTestCount int64
	s.db.Model(&models.ABTest{}).Where("application_id IN (?)",
		s.db.Model(&models.Application{}).Select("id").Where("tenant_id = ?", tenantID),
	).Count(&abTestCount)

	stats := map[string]interface{}{
		"users": map[string]interface{}{
			"used": quota.CurrentUsers,
			"limit": quota.MaxUsers,
			"percentage": calculatePercentage(quota.CurrentUsers, quota.MaxUsers),
		},
		"applications": map[string]interface{}{
			"used": appCount,
			"limit": quota.MaxApplications,
			"percentage": calculatePercentage(int(appCount), quota.MaxApplications),
		},
		"api_requests": map[string]interface{}{
			"used": quota.CurrentAPIRequests,
			"limit": quota.MaxAPIRequests,
			"percentage": calculatePercentage64(quota.CurrentAPIRequests, quota.MaxAPIRequests),
		},
		"storage": map[string]interface{}{
			"used": quota.CurrentStorage,
			"limit": quota.MaxStorage,
			"percentage": calculatePercentage64(quota.CurrentStorage, quota.MaxStorage),
		},
		"webhooks": map[string]interface{}{
			"used": webhookCount,
			"limit": quota.MaxWebhooks,
			"percentage": calculatePercentage(int(webhookCount), quota.MaxWebhooks),
		},
		"rules": map[string]interface{}{
			"used": ruleCount,
			"limit": quota.MaxRules,
			"percentage": calculatePercentage(int(ruleCount), quota.MaxRules),
		},
		"ab_tests": map[string]interface{}{
			"used": abTestCount,
			"limit": quota.MaxABTests,
			"percentage": calculatePercentage(int(abTestCount), quota.MaxABTests),
		},
		"period_start": quota.PeriodStart,
		"period_end":   quota.PeriodEnd,
	}

	return stats, nil
}

func (s *TenantService) GetTenantAuditLogs(tenantID uint, page, pageSize int, action, resourceType string) ([]models.TenantAuditLog, int64, error) {
	var logs []models.TenantAuditLog
	var total int64

	query := s.db.Model(&models.TenantAuditLog{}).Where("tenant_id = ?", tenantID)

	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *TenantService) logTenantAction(tenantID, userID uint, action, resourceType, resourceID, oldValue, newValue string) {
	log := models.TenantAuditLog{
		TenantID:     tenantID,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      fmt.Sprintf(`{"old": %s, "new": %s}`, oldValue, newValue),
	}
	s.db.Create(&log)
}

func (s *TenantService) ApplyTenantScope(query *gorm.DB, tenantID uint) *gorm.DB {
	return query.Where("tenant_id = ?", tenantID)
}

func (s *TenantService) GetTenantCacheKey(tenantID uint, resourceType, resourceID string) string {
	return fmt.Sprintf("tenant:%d:%s:%s", tenantID, resourceType, resourceID)
}

func (s *TenantService) InvalidateTenantCache(tenantID uint) {
	if s.cacheService != nil {
		ctx := context.Background()
		pattern := fmt.Sprintf("tenant:%d:*", tenantID)
		s.cacheService.Del(ctx, fmt.Sprintf("tenant:context:%d", tenantID))
		s.cacheService.Del(ctx, pattern)
	}
}

func generateSecureTokenForTenant(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(result)
}

func calculatePercentage(used, limit int) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}

func calculatePercentage64(used, limit int64) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}

type TenantScopeMiddleware struct {
	tenantService *TenantService
}

func NewTenantScopeMiddleware(tenantService *TenantService) *TenantScopeMiddleware {
	return &TenantScopeMiddleware{tenantService: tenantService}
}

func (m *TenantScopeMiddleware) GetTenantIDFromContext(c context.Context) (uint, bool) {
	if tenantID, ok := c.Value("tenant_id").(uint); ok {
		return tenantID, true
	}
	return 0, false
}

func (m *TenantScopeMiddleware) SetTenantContext(c context.Context, tenantID uint) context.Context {
	return context.WithValue(c, "tenant_id", tenantID)
}

func (m *TenantScopeMiddleware) FilterByTenant(query *gorm.DB, c context.Context) *gorm.DB {
	if tenantID, ok := m.GetTenantIDFromContext(c); ok {
		return m.tenantService.ApplyTenantScope(query, tenantID)
	}
	return query
}

func (m *TenantScopeMiddleware) GetTenantCache(c context.Context, tenantID uint, resourceType, resourceID string) (string, error) {
	if m.tenantService.cacheService == nil {
		return "", fmt.Errorf("cache service not available")
	}

	ctx := context.Background()
	key := m.tenantService.GetTenantCacheKey(tenantID, resourceType, resourceID)
	result, err := m.tenantService.cacheService.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

func (m *TenantScopeMiddleware) SetTenantCache(c context.Context, tenantID uint, resourceType, resourceID, value string, ttl time.Duration) error {
	if m.tenantService.cacheService == nil {
		return fmt.Errorf("cache service not available")
	}

	ctx := context.Background()
	key := m.tenantService.GetTenantCacheKey(tenantID, resourceType, resourceID)
	return m.tenantService.cacheService.Set(ctx, key, value, ttl)
}
