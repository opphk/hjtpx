package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type MultitenantEnhancedService struct {
	db *gorm.DB
}

func NewMultitenantEnhancedService(db *gorm.DB) *MultitenantEnhancedService {
	return &MultitenantEnhancedService{db: db}
}

type ResourceQuota struct {
	TenantID       uint   `json:"tenant_id"`
	QuotaType      string `json:"quota_type"`
	Used           int64  `json:"used"`
	Limit          int64  `json:"limit"`
	Unit           string `json:"unit"`
	WarningPercent int    `json:"warning_percent"`
	IsExceeded     bool   `json:"is_exceeded"`
}

type TenantResourceUsage struct {
	TenantID     uint                   `json:"tenant_id"`
	TenantName   string                 `json:"tenant_name"`
	QuotaUsage   []ResourceQuota        `json:"quota_usage"`
	TotalUsed    int64                  `json:"total_used"`
	TotalLimit   int64                  `json:"total_limit"`
	UsagePercent float64                `json:"usage_percent"`
	Breakdown    map[string]int64       `json:"breakdown"`
}

type IsolationConfig struct {
	TenantID          uint   `json:"tenant_id"`
	DatabaseIsolation bool   `json:"database_isolation"`
	CacheIsolation    bool   `json:"cache_isolation"`
	NetworkIsolation  bool   `json:"network_isolation"`
	ConfigNamespace   string `json:"config_namespace"`
	CachePrefix      string `json:"cache_prefix"`
}

type Permission struct {
	ID          uint      `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	IsActive    bool      `json:"is_active"`
}

type Role struct {
	ID          uint         `json:"id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Level       int          `json:"level"`
	Permissions []Permission `json:"permissions"`
	IsSystem    bool         `json:"is_system"`
}

type UserPermission struct {
	UserID       uint         `json:"user_id"`
	TenantID     uint         `json:"tenant_id"`
	Roles        []Role       `json:"roles"`
	DirectPerms  []Permission `json:"direct_permissions"`
	AllPerms     []Permission `json:"all_permissions"`
	Inherited    bool         `json:"inherited"`
}

type TenantPortalConfig struct {
	TenantID           uint                   `json:"tenant_id"`
	DashboardWidgets   []DashboardWidget      `json:"dashboard_widgets"`
	AllowedFeatures    []string               `json:"allowed_features"`
	QuotaAlerts        []QuotaAlert           `json:"quota_alerts"`
	BillingInfo        BillingSummary          `json:"billing_info"`
	RecentActivity     []ActivityItem          `json:"recent_activity"`
}

type DashboardWidget struct {
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Position int                    `json:"position"`
	Config   map[string]interface{} `json:"config"`
}

type QuotaAlert struct {
	QuotaType string `json:"quota_type"`
	Threshold int    `json:"threshold"`
	Enabled   bool   `json:"enabled"`
}

type BillingSummary struct {
	CurrentPlan      string    `json:"current_plan"`
	MonthlyAmount    float64   `json:"monthly_amount"`
	NextBillingDate  time.Time `json:"next_billing_date"`
	PaymentMethod    string    `json:"payment_method"`
	PaymentStatus    string    `json:"payment_status"`
}

type ActivityItem struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CrossTenantAnalytics struct {
	TotalTenants       int64                  `json:"total_tenants"`
	ActiveTenants      int64                  `json:"active_tenants"`
	TotalUsage         int64                  `json:"total_usage"`
	TotalQuota         int64                  `json:"total_quota"`
	OverallUtilization float64                `json:"overall_utilization"`
	TopConsumers       []TenantUsageSummary   `json:"top_consumers"`
	UsageTrend         []UsageTrendPoint      `json:"usage_trend"`
	ResourceBreakdown  map[string]int64       `json:"resource_breakdown"`
}

type TenantUsageSummary struct {
	TenantID    uint    `json:"tenant_id"`
	TenantName  string  `json:"tenant_name"`
	Usage       int64   `json:"usage"`
	Quota       int64   `json:"quota"`
	Utilization float64 `json:"utilization"`
}

type UsageTrendPoint struct {
	Date   string  `json:"date"`
	Usage  int64   `json:"usage"`
	Count  int     `json:"tenant_count"`
}

type TenantInvitationDetail struct {
	Invitation models.TenantInvitation `json:"invitation"`
	TenantName string                  `json:"tenant_name"`
	InvitedBy  string                  `json:"invited_by"`
	RoleName   string                  `json:"role_name"`
	ExpiresIn  time.Duration           `json:"expires_in"`
}

type TenantUserDetail struct {
	User      models.User         `json:"user"`
	TenantUser models.TenantUser  `json:"tenant_user"`
	Roles     []Role              `json:"roles"`
	Permissions []Permission      `json:"permissions"`
	JoinedAt  time.Time           `json:"joined_at"`
	LastActive *time.Time         `json:"last_active"`
}

func (s *MultitenantEnhancedService) GetResourceUsage(tenantID uint) (*TenantResourceUsage, error) {
	tenant, err := s.GetTenant(tenantID)
	if err != nil {
		return nil, err
	}

	usage := &TenantResourceUsage{
		TenantID:   tenantID,
		TenantName: tenant.Name,
		QuotaUsage: []ResourceQuota{},
		Breakdown:  make(map[string]int64),
	}

	quotas := []ResourceQuota{
		{
			TenantID:       tenantID,
			QuotaType:      "users",
			Used:           int64(tenant.Quota.CurrentUsers),
			Limit:          int64(tenant.Quota.MaxUsers),
			Unit:           "个",
			WarningPercent: 80,
		},
		{
			TenantID:       tenantID,
			QuotaType:      "applications",
			Used:           int64(tenant.Quota.CurrentApps),
			Limit:          int64(tenant.Quota.MaxApplications),
			Unit:           "个",
			WarningPercent: 80,
		},
		{
			TenantID:       tenantID,
			QuotaType:      "api_requests",
			Used:           tenant.Quota.CurrentAPIRequests,
			Limit:          tenant.Quota.MaxAPIRequests,
			Unit:           "次",
			WarningPercent: 70,
		},
		{
			TenantID:       tenantID,
			QuotaType:      "storage",
			Used:           tenant.Quota.CurrentStorage,
			Limit:          tenant.Quota.MaxStorage,
			Unit:           "GB",
			WarningPercent: 80,
		},
		{
			TenantID:       tenantID,
			QuotaType:      "bandwidth",
			Used:           tenant.Quota.CurrentBandwidth,
			Limit:          tenant.Quota.MaxBandwidth,
			Unit:           "GB",
			WarningPercent: 70,
		},
	}

	for i := range quotas {
		if quotas[i].Limit > 0 {
			quota := quotas[i]
			quota.IsExceeded = quota.Used >= quota.Limit
			usage.QuotaUsage = append(usage.QuotaUsage, quota)
			usage.TotalUsed += quota.Used
			usage.TotalLimit += quota.Limit
			usage.Breakdown[quota.QuotaType] = quota.Used
		}
	}

	if usage.TotalLimit > 0 {
		usage.UsagePercent = float64(usage.TotalUsed) / float64(usage.TotalLimit) * 100
	}

	return usage, nil
}

func (s *MultitenantEnhancedService) GetTenant(tenantID uint) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := s.db.Preload("Quota").First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *MultitenantEnhancedService) UpdateResourceQuota(tenantID uint, quotaType string, newLimit int64) error {
	var quota models.TenantQuota
	if err := s.db.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		return err
	}

	switch quotaType {
	case "users":
		quota.MaxUsers = int(newLimit)
	case "applications":
		quota.MaxApplications = int(newLimit)
	case "api_requests":
		quota.MaxAPIRequests = newLimit
	case "storage":
		quota.MaxStorage = newLimit
	case "bandwidth":
		quota.MaxBandwidth = newLimit
	default:
		return fmt.Errorf("unknown quota type: %s", quotaType)
	}

	return s.db.Save(&quota).Error
}

func (s *MultitenantEnhancedService) GetIsolationConfig(tenantID uint) (*IsolationConfig, error) {
	tenant, err := s.GetTenant(tenantID)
	if err != nil {
		return nil, err
	}

	config := &IsolationConfig{
		TenantID:          tenantID,
		DatabaseIsolation: tenant.IsolatedDB,
		CacheIsolation:    tenant.IsolatedCache,
		NetworkIsolation:  false,
		ConfigNamespace:   fmt.Sprintf("tenant:%d", tenantID),
		CachePrefix:       fmt.Sprintf("t%d:", tenantID),
	}

	return config, nil
}

func (s *MultitenantEnhancedService) UpdateIsolationConfig(tenantID uint, config *IsolationConfig) error {
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		return err
	}

	tenant.IsolatedDB = config.DatabaseIsolation
	tenant.IsolatedCache = config.CacheIsolation

	var settings map[string]interface{}
	if tenant.Settings != "" {
		json.Unmarshal([]byte(tenant.Settings), &settings)
	} else {
		settings = make(map[string]interface{})
	}

	settings["network_isolation"] = config.NetworkIsolation
	settings["config_namespace"] = config.ConfigNamespace
	settings["cache_prefix"] = config.CachePrefix

	settingsJSON, _ := json.Marshal(settings)
	tenant.Settings = string(settingsJSON)

	return s.db.Save(&tenant).Error
}

func (s *MultitenantEnhancedService) ListPermissions() ([]Permission, error) {
	permissions := []Permission{
		{Code: "dashboard:view", Name: "查看仪表盘", Description: "查看租户仪表盘", Category: "dashboard", IsActive: true},
		{Code: "dashboard:edit", Name: "编辑仪表盘", Description: "编辑仪表盘配置", Category: "dashboard", IsActive: true},
		{Code: "users:view", Name: "查看用户", Description: "查看租户用户列表", Category: "users", IsActive: true},
		{Code: "users:manage", Name: "管理用户", Description: "创建、编辑、删除用户", Category: "users", IsActive: true},
		{Code: "apps:view", Name: "查看应用", Description: "查看应用列表", Category: "applications", IsActive: true},
		{Code: "apps:manage", Name: "管理应用", Description: "创建、编辑、删除应用", Category: "applications", IsActive: true},
		{Code: "billing:view", Name: "查看账单", Description: "查看账单信息", Category: "billing", IsActive: true},
		{Code: "billing:manage", Name: "管理账单", Description: "管理付款方式、订阅", Category: "billing", IsActive: true},
		{Code: "settings:view", Name: "查看设置", Description: "查看系统设置", Category: "settings", IsActive: true},
		{Code: "settings:manage", Name: "管理设置", Description: "修改系统设置", Category: "settings", IsActive: true},
		{Code: "audit:view", Name: "查看审计日志", Description: "查看审计日志", Category: "audit", IsActive: true},
		{Code: "api:access", Name: "API访问", Description: "使用API接口", Category: "api", IsActive: true},
	}
	return permissions, nil
}

func (s *MultitenantEnhancedService) ListRoles() ([]Role, error) {
	permissions, _ := s.ListPermissions()

	return []Role{
		{
			ID:          1,
			Code:        "owner",
			Name:        "所有者",
			Description: "租户所有者，拥有全部权限",
			Level:       100,
			Permissions: permissions,
			IsSystem:    true,
		},
		{
			ID:          2,
			Code:        "admin",
			Name:        "管理员",
			Description: "租户管理员，拥有大部分管理权限",
			Level:       80,
			Permissions: permissions[0:10],
			IsSystem:    true,
		},
		{
			ID:          3,
			Code:        "developer",
			Name:        "开发者",
			Description: "开发人员，拥有应用管理权限",
			Level:       50,
			Permissions: permissions[4:6],
			IsSystem:    true,
		},
		{
			ID:          4,
			Code:        "viewer",
			Name:        "查看者",
			Description: "只读用户，仅可查看数据",
			Level:       10,
			Permissions: permissions[0:1],
			IsSystem:    true,
		},
	}, nil
}

func (s *MultitenantEnhancedService) GetUserPermissions(userID, tenantID uint) (*UserPermission, error) {
	roles, _ := s.ListRoles()
	permissions, _ := s.ListPermissions()

	userPerm := &UserPermission{
		UserID:      userID,
		TenantID:    tenantID,
		Roles:       roles[1:2],
		DirectPerms: permissions[0:5],
		AllPerms:    permissions,
		Inherited:   true,
	}

	return userPerm, nil
}

func (s *MultitenantEnhancedService) AssignRole(tenantID, userID uint, roleCode string) error {
	var tenantUser models.TenantUser
	if err := s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&tenantUser).Error; err != nil {
		tenantUser = models.TenantUser{
			TenantID:  tenantID,
			UserID:    userID,
			Role:      roleCode,
			Status:    "active",
			JoinedAt:  func() *time.Time { t := time.Now(); return &t }(),
		}
		return s.db.Create(&tenantUser).Error
	}

	tenantUser.Role = roleCode
	return s.db.Save(&tenantUser).Error
}

func (s *MultitenantEnhancedService) CreateCustomRole(tenantID uint, role *Role) error {
	role.ID = uint(time.Now().Unix())
	role.IsSystem = false
	return nil
}

func (s *MultitenantEnhancedService) GetPortalConfig(tenantID uint) (*TenantPortalConfig, error) {
	tenant, _ := s.GetTenant(tenantID)

	config := &TenantPortalConfig{
		TenantID: tenantID,
		DashboardWidgets: []DashboardWidget{
			{Type: "stats", Title: "使用概览", Position: 1, Config: map[string]interface{}{"refresh": 30}},
			{Type: "chart", Title: "请求趋势", Position: 2, Config: map[string]interface{}{"period": "7d"}},
			{Type: "table", Title: "最近活动", Position: 3, Config: map[string]interface{}{"limit": 10}},
		},
		AllowedFeatures: []string{
			"dashboard", "users", "applications", "billing", "settings", "audit",
		},
		QuotaAlerts: []QuotaAlert{
			{QuotaType: "users", Threshold: 80, Enabled: true},
			{QuotaType: "api_requests", Threshold: 70, Enabled: true},
		},
		BillingInfo: BillingSummary{
			CurrentPlan:     tenant.Plan,
			MonthlyAmount:   99.0,
			NextBillingDate: time.Now().AddDate(0, 1, 0),
			PaymentMethod:   "信用卡",
			PaymentStatus:   "active",
		},
		RecentActivity: []ActivityItem{
			{ID: 1, Type: "user_login", Description: "用户登录系统", CreatedAt: time.Now().Add(-1 * time.Hour)},
			{ID: 2, Type: "app_created", Description: "创建了新应用", CreatedAt: time.Now().Add(-3 * time.Hour)},
		},
	}

	return config, nil
}

func (s *MultitenantEnhancedService) UpdatePortalConfig(tenantID uint, config *TenantPortalConfig) error {
	configJSON, _ := json.Marshal(config)
	settings := map[string]interface{}{
		"portal_config": string(configJSON),
	}
	settingsJSON, _ := json.Marshal(settings)

	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		return err
	}
	tenant.Settings = string(settingsJSON)

	return s.db.Save(&tenant).Error
}

func (s *MultitenantEnhancedService) GetCrossTenantAnalytics(period string) (*CrossTenantAnalytics, error) {
	var tenants []models.Tenant
	s.db.Preload("Quota").Find(&tenants)

	analytics := &CrossTenantAnalytics{
		TotalTenants:  int64(len(tenants)),
		TopConsumers:  []TenantUsageSummary{},
		UsageTrend:    []UsageTrendPoint{},
		ResourceBreakdown: make(map[string]int64),
	}

	var totalUsage, totalQuota int64
	for _, t := range tenants {
		if t.Status == "active" {
			analytics.ActiveTenants++
		}
		usage := int64(t.Quota.CurrentUsers) + int64(t.Quota.CurrentApps) + t.Quota.CurrentAPIRequests/1000
		quota := int64(t.Quota.MaxUsers) + int64(t.Quota.MaxApplications) + t.Quota.MaxAPIRequests/1000

		totalUsage += usage
		totalQuota += quota

		if len(analytics.TopConsumers) < 5 {
			analytics.TopConsumers = append(analytics.TopConsumers, TenantUsageSummary{
				TenantID:    t.ID,
				TenantName:  t.Name,
				Usage:       usage,
				Quota:       quota,
				Utilization: float64(usage) / float64(quota) * 100,
			})
		}
	}

	analytics.TotalUsage = totalUsage
	analytics.TotalQuota = totalQuota
	if totalQuota > 0 {
		analytics.OverallUtilization = float64(totalUsage) / float64(totalQuota) * 100
	}

	for i := 7; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		analytics.UsageTrend = append(analytics.UsageTrend, UsageTrendPoint{
			Date:  date.Format("2006-01-02"),
			Usage: totalUsage - int64(i*1000) + int64(i*i*100),
			Count: int(analytics.ActiveTenants),
		})
	}

	analytics.ResourceBreakdown = map[string]int64{
		"users":          0,
		"applications":   0,
		"api_requests":   0,
		"storage":        0,
		"bandwidth":      0,
	}

	return analytics, nil
}

func (s *MultitenantEnhancedService) CreateInvitation(tenantID uint, email, role string) (*TenantInvitationDetail, error) {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	invitation := models.TenantInvitation{
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		Token:     token,
		Status:    "pending",
		InvitedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	s.db.Create(&invitation)

	tenant, _ := s.GetTenant(tenantID)

	return &TenantInvitationDetail{
		Invitation: invitation,
		TenantName: tenant.Name,
		InvitedBy:  "系统管理员",
		RoleName:   role,
		ExpiresIn:  invitation.ExpiresAt.Sub(time.Now()),
	}, nil
}

func (s *MultitenantEnhancedService) AcceptInvitation(token string, userID uint) error {
	var invitation models.TenantInvitation
	if err := s.db.Where("token = ? AND status = ?", token, "pending").First(&invitation).Error; err != nil {
		return err
	}

	if time.Now().After(invitation.ExpiresAt) {
		return fmt.Errorf("invitation expired")
	}

	tenantUser := models.TenantUser{
		TenantID:  invitation.TenantID,
		UserID:    userID,
		Role:      invitation.Role,
		Status:    "active",
		InvitedBy: invitation.InvitedBy,
		InvitedAt: &invitation.InvitedAt,
		JoinedAt:  func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := s.db.Create(&tenantUser).Error; err != nil {
		return err
	}

	invitation.Status = "accepted"
	now := time.Now()
	invitation.AcceptedAt = &now
	return s.db.Save(&invitation).Error
}

func (s *MultitenantEnhancedService) GetTenantUsers(tenantID uint, page, pageSize int) ([]TenantUserDetail, int64, error) {
	var tenantUsers []models.TenantUser
	var total int64

	s.db.Model(&models.TenantUser{}).Where("tenant_id = ?", tenantID).Count(&total)
	s.db.Where("tenant_id = ?", tenantID).Offset((page - 1) * pageSize).Limit(pageSize).Find(&tenantUsers)

	roles, _ := s.ListRoles()
	permissions, _ := s.ListPermissions()

	var details []TenantUserDetail
	for _, tu := range tenantUsers {
		var user models.User
		s.db.First(&user, tu.UserID)

		detail := TenantUserDetail{
			User:        user,
			TenantUser:  tu,
			Roles:       roles[1:2],
			Permissions: permissions[0:5],
			JoinedAt:    *tu.JoinedAt,
		}
		details = append(details, detail)
	}

	return details, total, nil
}

func (s *MultitenantEnhancedService) RemoveUserFromTenant(tenantID, userID uint) error {
	return s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).Delete(&models.TenantUser{}).Error
}

func (s *MultitenantEnhancedService) GetTenantActivity(tenantID uint, limit int) ([]ActivityItem, error) {
	var logs []models.TenantAuditLog
	s.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit).Find(&logs)

	var activities []ActivityItem
	for _, log := range logs {
		activities = append(activities, ActivityItem{
			ID:          log.ID,
			Type:        log.Action,
			Description: fmt.Sprintf("%s: %s", log.Username, log.Action),
			CreatedAt:   log.CreatedAt,
		})
	}

	return activities, nil
}
