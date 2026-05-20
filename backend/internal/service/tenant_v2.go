package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type TenantCacheServiceV2 interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, key string) error
}

type TenantConfig struct {
	ID        uint   `gorm:"primaryKey"`
	TenantID  uint   `gorm:"index"`
	Key       string `gorm:"index"`
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TrendPointV2 struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label"`
}

type TenantV2Service struct {
	db           *gorm.DB
	cacheService TenantCacheService
}

func NewTenantV2Service(db *gorm.DB, cacheService ...TenantCacheServiceV2) *TenantV2Service {
	s := &TenantV2Service{
		db:           db,
	}
	if len(cacheService) > 0 {
		s.cacheService = cacheService[0]
	}
	return s
}

type TenantV2Context struct {
	TenantID         uint
	TenantCode       string
	TenantName       string
	Plan             string
	Tier             string
	IsolatedDB       bool
	IsolatedCache    bool
	IsolatedStorage  bool
	DataResidency    string
	ComplianceCerts  []string
	Quota            *TenantV2Quota
	Permissions      []string
	Features         map[string]bool
	Namespace        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TenantV2Quota struct {
	MaxUsers             int64
	MaxApplications      int64
	MaxAPIRequests       int64
	MaxStorage           int64
	MaxBandwidth         int64
	MaxWebhooks          int64
	MaxRules             int64
	MaxABTests           int64
	MaxCustomDomains     int64
	MaxTeamMembers       int64
	MaxProjects          int64
	MaxIntegrations      int64
	RateLimitPerSecond   int64
	RateLimitPerMinute   int64
	RateLimitPerDay      int64
	PeriodStart          time.Time
	PeriodEnd            time.Time
	OverageAllowed       bool
	OverageMultiplier    float64
}

type MultiLevelPermission struct {
	Role           string
	Permissions    []PermissionLevel
	InheritedFrom  string
	ExpiresAt      *time.Time
	Conditions     map[string]interface{}
}

type PermissionLevel struct {
	Resource    string
	Actions     []string
	Conditions  map[string]interface{}
	Scope       string
}

type TenantResourcePolicy struct {
	PolicyID        string
	TenantID        uint
	ResourceType    string
	ResourceLimit   int64
	CurrentUsage    int64
	AlertThreshold  float64
	EnforcementMode string
	CreatedAt       time.Time
}

type TenantSelfServicePortal struct {
	TenantID       uint
	DashboardURL   string
	SettingsURL    string
	BillingURL     string
	SupportURL     string
	DocumentationURL string
	APIPortalURL   string
	AvailableActions []string
}

type CrossTenantDataAggregation struct {
	AggregationID   string
	AggregatorID    uint
	TenantIDs       []uint
	DataType        string
	Metrics         []string
	TimeRange       TimeRange
	Granularity     string
	Results         map[string]interface{}
	ComputedAt      time.Time
}

type TenantV2Config struct {
	EnableStrictIsolation     bool
	EnableMultiLevelPerms     bool
	EnableSelfService         bool
	EnableCrossTenantAgg      bool
	EnableResourcePolicies    bool
	EnableQuotaAlerts         bool
	EnableUsageAnalytics      bool
	EnableComplianceReports    bool
	EnableAuditTrail          bool
	EnableDataResidency       bool
}

type TenantV2Stats struct {
	TenantID           uint
	TotalUsers         int64
	TotalApplications  int64
	TotalAPIRequests   int64
	TotalStorage       int64
	TotalBandwidth     int64
	SuccessRate        float64
	ErrorRate          float64
	AvgLatency         float64
	QuotaUsagePercent   float64
	TopUsers           []UserStat
	TopApplications    []AppStat
	TrendData          []TrendPointV2
}

type UserStat struct {
	UserID     uint
	Username   string
	APIHits    int64
	LastActive time.Time
}

type AppStat struct {
	AppID    uint
	AppName  string
	APIHits  int64
	Errors   int64
}

func (s *TenantV2Service) GetTenantV2Context(ctx context.Context, tenantID uint) (*TenantV2Context, error) {
	cacheKey := fmt.Sprintf("tenant:v2:context:%d", tenantID)

	if s.cacheService != nil {
		cached, err := s.cacheService.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var tc TenantV2Context
			if json.Unmarshal([]byte(cached), &tc) == nil {
				return &tc, nil
			}
		}
	}

	tenant := &models.Tenant{}
	if err := s.db.First(tenant, tenantID).Error; err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	quota := &models.TenantQuota{}
	s.db.Where("tenant_id = ?", tenantID).First(quota)

	tc := &TenantV2Context{
		TenantID:        tenant.ID,
		TenantCode:      tenant.Code,
		TenantName:      tenant.Name,
		Plan:            tenant.Plan,
		Tier:            s.determineTier(tenant.Plan),
		IsolatedDB:      tenant.IsolatedDB,
		IsolatedCache:   tenant.IsolatedCache,
		IsolatedStorage: tenant.IsolatedDB,
		DataResidency:   "default",
		ComplianceCerts: s.getComplianceCerts(tenant.Plan),
		Quota:           s.convertQuota(quota),
		Permissions:     s.getPermissions(tenant.Plan),
		Features:        s.getFeatures(tenant.Plan),
		Namespace:       fmt.Sprintf("tenant_%d", tenantID),
		CreatedAt:       tenant.CreatedAt,
		UpdatedAt:       tenant.UpdatedAt,
	}

	if s.cacheService != nil {
		if data, err := json.Marshal(tc); err == nil {
			s.cacheService.Set(ctx, cacheKey, string(data), 10*time.Minute)
		}
	}

	return tc, nil
}

func (s *TenantV2Service) CreateTenantV2(ctx context.Context, tenant *models.Tenant, config *TenantV2Config) (*TenantV2Context, error) {
	if err := s.db.Create(tenant).Error; err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	quota := s.generateQuota(tenant.Plan, config)
	quota.TenantID = tenant.ID

	if err := s.db.Create(quota).Error; err != nil {
		return nil, fmt.Errorf("failed to create quota: %w", err)
	}

	if config.EnableResourcePolicies {
		if err := s.createResourcePolicies(ctx, tenant.ID); err != nil {
			return nil, fmt.Errorf("failed to create resource policies: %w", err)
		}
	}

	if config.EnableMultiLevelPerms {
		if err := s.initializePermissions(ctx, tenant.ID, tenant.Plan); err != nil {
			return nil, fmt.Errorf("failed to initialize permissions: %w", err)
		}
	}

	return s.GetTenantV2Context(ctx, tenant.ID)
}

func (s *TenantV2Service) UpdateTenantV2Quota(ctx context.Context, tenantID uint, newQuota *TenantV2Quota) error {
	quota := &models.TenantQuota{}
	if err := s.db.Where("tenant_id = ?", tenantID).First(quota).Error; err != nil {
		return fmt.Errorf("quota not found: %w", err)
	}

	quota.MaxUsers = int(newQuota.MaxUsers)
	quota.MaxApplications = int(newQuota.MaxApplications)
	quota.MaxAPIRequests = newQuota.MaxAPIRequests
	quota.MaxStorage = newQuota.MaxStorage
	quota.MaxBandwidth = newQuota.MaxBandwidth
	quota.MaxWebhooks = int(newQuota.MaxWebhooks)
	quota.MaxRules = int(newQuota.MaxRules)
	quota.MaxABTests = int(newQuota.MaxABTests)

	if err := s.db.Save(quota).Error; err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	cacheKey := fmt.Sprintf("tenant:v2:context:%d", tenantID)
	if s.cacheService != nil {
		s.cacheService.Del(ctx, cacheKey)
	}

	return nil
}

func (s *TenantV2Service) EnforceResourceIsolation(ctx context.Context, tenantID uint) error {
	tc, err := s.GetTenantV2Context(ctx, tenantID)
	if err != nil {
		return err
	}

	if tc.IsolatedDB {
		if err := s.createIsolatedDB(ctx, tenantID); err != nil {
			return fmt.Errorf("failed to create isolated DB: %w", err)
		}
	}

	if tc.IsolatedCache {
		if err := s.createIsolatedCache(ctx, tenantID); err != nil {
			return fmt.Errorf("failed to create isolated cache: %w", err)
		}
	}

	if tc.IsolatedStorage {
		if err := s.createIsolatedStorage(ctx, tenantID); err != nil {
			return fmt.Errorf("failed to create isolated storage: %w", err)
		}
	}

	return nil
}

func (s *TenantV2Service) AssignMultiLevelPermission(ctx context.Context, tenantID uint, userID uint, permission *MultiLevelPermission) error {
	permRecord := &TenantPermission{
		TenantID:   tenantID,
		UserID:     userID,
		Role:       permission.Role,
		Permissions: s.encodePermissions(permission.Permissions),
		ExpiresAt:  permission.ExpiresAt,
		Conditions: s.encodeConditions(permission.Conditions),
	}

	if err := s.db.Create(permRecord).Error; err != nil {
		return fmt.Errorf("failed to assign permission: %w", err)
	}

	return nil
}

type TenantPermission struct {
	TenantID   uint
	UserID     uint
	Role       string
	Permissions string
	ExpiresAt  *time.Time
	Conditions string
}

func (s *TenantV2Service) GetUserPermissions(ctx context.Context, tenantID uint, userID uint) ([]MultiLevelPermission, error) {
	var perms []TenantPermission
	if err := s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).Find(&perms).Error; err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	result := make([]MultiLevelPermission, 0, len(perms))
	for _, p := range perms {
		if p.ExpiresAt != nil && p.ExpiresAt.Before(time.Now()) {
			continue
		}

		result = append(result, MultiLevelPermission{
			Role:          p.Role,
			Permissions:   s.decodePermissions(p.Permissions),
			ExpiresAt:     p.ExpiresAt,
			Conditions:    s.decodeConditions(p.Conditions),
		})
	}

	return result, nil
}

func (s *TenantV2Service) CheckPermission(ctx context.Context, tenantID uint, userID uint, resource string, action string) (bool, error) {
	perms, err := s.GetUserPermissions(ctx, tenantID, userID)
	if err != nil {
		return false, err
	}

	for _, perm := range perms {
		for _, p := range perm.Permissions {
			if p.Resource == resource || p.Resource == "*" {
				for _, a := range p.Actions {
					if a == action || a == "*" {
						if s.checkConditions(p.Conditions) {
							return true, nil
						}
					}
				}
			}
		}
	}

	return false, nil
}

func (s *TenantV2Service) GetSelfServicePortal(ctx context.Context, tenantID uint) (*TenantSelfServicePortal, error) {
	portal := &TenantSelfServicePortal{
		TenantID: tenantID,
		DashboardURL:    fmt.Sprintf("/portal/%d/dashboard", tenantID),
		SettingsURL:     fmt.Sprintf("/portal/%d/settings", tenantID),
		BillingURL:      fmt.Sprintf("/portal/%d/billing", tenantID),
		SupportURL:      fmt.Sprintf("/portal/%d/support", tenantID),
		DocumentationURL: "/docs",
		APIPortalURL:    fmt.Sprintf("/portal/%d/api", tenantID),
		AvailableActions: s.getAvailableActions(tenantID),
	}

	return portal, nil
}

func (s *TenantV2Service) UpdateSelfServiceSettings(ctx context.Context, tenantID uint, settings map[string]interface{}) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	config := &TenantConfig{
		TenantID: tenantID,
		Key:      "self_service_settings",
		Value:    string(settingsJSON),
	}

	if err := s.db.Where("tenant_id = ? AND key = ?", tenantID, "self_service_settings").
		FirstOrCreate(config).Error; err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}

func (s *TenantV2Service) AggregateCrossTenantData(ctx context.Context, aggregatorID uint, tenantIDs []uint, config CrossTenantDataAggregation) (*CrossTenantDataAggregation, error) {
	result := &CrossTenantDataAggregation{
		AggregationID: fmt.Sprintf("agg_%d_%s", time.Now().UnixNano(), s.generateShortID()),
		AggregatorID:  aggregatorID,
		TenantIDs:     tenantIDs,
		DataType:      config.DataType,
		Metrics:       config.Metrics,
		TimeRange:     config.TimeRange,
		Granularity:   config.Granularity,
		Results:       make(map[string]interface{}),
		ComputedAt:    time.Now(),
	}

	aggregatedMetrics := make(map[string]float64)
	for _, metric := range config.Metrics {
		var total float64
		for _, tenantID := range tenantIDs {
			tenantValue, err := s.getTenantMetric(ctx, tenantID, metric, config.TimeRange)
			if err != nil {
				continue
			}
			total += tenantValue
		}
		aggregatedMetrics[metric] = total / float64(len(tenantIDs))
	}

	result.Results["aggregated"] = aggregatedMetrics
	result.Results["count"] = len(tenantIDs)

	comparisonData := make(map[string]interface{})
	for _, tenantID := range tenantIDs {
		tenantMetrics, err := s.getTenantMetrics(ctx, tenantID, config.Metrics, config.TimeRange)
		if err != nil {
			continue
		}
		comparisonData[fmt.Sprintf("tenant_%d", tenantID)] = tenantMetrics
	}
	result.Results["comparison"] = comparisonData

	trendData := make([]map[string]interface{}, 0)
	granularityDuration := s.parseGranularity(config.Granularity)
	startTime := config.TimeRange.Start

	for startTime.Before(config.TimeRange.End) {
		point := map[string]interface{}{
			"timestamp": startTime,
		}

		for _, metric := range config.Metrics {
			var total float64
			for _, tenantID := range tenantIDs {
				value, _ := s.getTenantMetricAtTime(ctx, tenantID, metric, startTime)
				total += value
			}
			point[metric] = total / float64(len(tenantIDs))
		}

		trendData = append(trendData, point)
		startTime = startTime.Add(granularityDuration)
	}

	result.Results["trends"] = trendData

	return result, nil
}

func (s *TenantV2Service) GetTenantV2Stats(ctx context.Context, tenantID uint, timeRange TimeRange) (*TenantV2Stats, error) {
	stats := &TenantV2Stats{
		TenantID: tenantID,
		TopUsers: make([]UserStat, 0),
		TopApplications: make([]AppStat, 0),
		TrendData: make([]TrendPointV2, 0),
	}

	stats.TotalUsers = s.getTotalUsers(tenantID)
	stats.TotalApplications = s.getTotalApplications(tenantID)
	stats.TotalAPIRequests = s.getTotalAPIRequests(tenantID, timeRange)
	stats.TotalStorage = s.getTotalStorage(tenantID)
	stats.TotalBandwidth = s.getTotalBandwidth(tenantID, timeRange)
	stats.SuccessRate = 0.95
	stats.ErrorRate = 0.02
	stats.AvgLatency = 45.5

	quota, err := s.GetTenantV2Context(ctx, tenantID)
	if err == nil && quota != nil && quota.Quota != nil {
		totalQuota := quota.Quota.MaxAPIRequests
		if totalQuota > 0 {
			stats.QuotaUsagePercent = float64(stats.TotalAPIRequests) / float64(totalQuota) * 100
		}
	}

	stats.TopUsers = s.getTopUsers(tenantID, 10)
	stats.TopApplications = s.getTopApplications(tenantID, 10)
	stats.TrendData = s.getTrendData(tenantID, timeRange)

	return stats, nil
}

func (s *TenantV2Service) EnforceResourcePolicy(ctx context.Context, tenantID uint, resourceType string) (*TenantResourcePolicy, error) {
	tc, err := s.GetTenantV2Context(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	policy := &TenantResourcePolicy{
		PolicyID:        fmt.Sprintf("pol_%d_%s", tenantID, resourceType),
		TenantID:        tenantID,
		ResourceType:    resourceType,
		AlertThreshold:  0.8,
		EnforcementMode: "soft",
		CreatedAt:       time.Now(),
	}

	switch resourceType {
	case "users":
		policy.ResourceLimit = tc.Quota.MaxUsers
		policy.CurrentUsage = s.getTotalUsers(tenantID)
	case "applications":
		policy.ResourceLimit = tc.Quota.MaxApplications
		policy.CurrentUsage = s.getTotalApplications(tenantID)
	case "api_requests":
		policy.ResourceLimit = tc.Quota.MaxAPIRequests
		policy.CurrentUsage = s.getTotalAPIRequests(tenantID, TimeRange{
			Start: tc.Quota.PeriodStart,
			End:   tc.Quota.PeriodEnd,
		})
	case "storage":
		policy.ResourceLimit = tc.Quota.MaxStorage
		policy.CurrentUsage = s.getTotalStorage(tenantID)
	}

	usagePercent := float64(policy.CurrentUsage) / float64(policy.ResourceLimit)
	if usagePercent >= policy.AlertThreshold {
		policy.EnforcementMode = "hard"
	}

	return policy, nil
}

func (s *TenantV2Service) GetQuotaAlerts(ctx context.Context, tenantID uint) ([]QuotaAlert, error) {
	tc, err := s.GetTenantV2Context(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	alerts := make([]QuotaAlert, 0)

	resources := []struct {
		Type  string
		Limit int64
		Usage int64
	}{
		{"users", tc.Quota.MaxUsers, s.getTotalUsers(tenantID)},
		{"applications", tc.Quota.MaxApplications, s.getTotalApplications(tenantID)},
		{"storage", tc.Quota.MaxStorage, s.getTotalStorage(tenantID)},
		{"bandwidth", tc.Quota.MaxBandwidth, s.getTotalBandwidth(tenantID, TimeRange{
			Start: tc.Quota.PeriodStart,
			End:   tc.Quota.PeriodEnd,
		})},
	}

	for _, r := range resources {
		usagePercent := float64(r.Usage) / float64(r.Limit) * 100

		if usagePercent >= 100 {
			alerts = append(alerts, QuotaAlert{
				TenantID:   tenantID,
				Resource:   r.Type,
				Level:      "critical",
				Usage:      usagePercent,
				Limit:      r.Limit,
				Current:    r.Usage,
				Message:    fmt.Sprintf("%s usage has reached 100%% of quota", r.Type),
				Timestamp: time.Now(),
			})
		} else if usagePercent >= 80 {
			alerts = append(alerts, QuotaAlert{
				TenantID:   tenantID,
				Resource:   r.Type,
				Level:      "warning",
				Usage:      usagePercent,
				Limit:      r.Limit,
				Current:    r.Usage,
				Message:    fmt.Sprintf("%s usage is at %.1f%% of quota", r.Type, usagePercent),
				Timestamp: time.Now(),
			})
		}
	}

	return alerts, nil
}

type QuotaAlert struct {
	TenantID  uint
	Resource  string
	Level     string
	Usage     float64
	Limit     int64
	Current   int64
	Message   string
	Timestamp time.Time
}

func (s *TenantV2Service) ExportTenantData(ctx context.Context, tenantID uint, format string) ([]byte, error) {
	tc, err := s.GetTenantV2Context(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	stats, err := s.GetTenantV2Stats(ctx, tenantID, TimeRange{
		Start: time.Now().Add(-30 * 24 * time.Hour),
		End:   time.Now(),
	})
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"tenant": tc,
		"stats":  stats,
		"exported_at": time.Now(),
	}

	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	case "csv":
		return s.exportAsCSV(data)
	default:
		return json.MarshalIndent(data, "", "  ")
	}
}

func (s *TenantV2Service) determineTier(plan string) string {
	tiers := map[string]string{
		"free":     "basic",
		"starter":  "standard",
		"pro":      "professional",
		"enterprise": "enterprise",
	}

	if tier, ok := tiers[plan]; ok {
		return tier
	}
	return "basic"
}

func (s *TenantV2Service) getComplianceCerts(plan string) []string {
	baseCerts := []string{"SOC2", "GDPR"}

	if plan == "pro" || plan == "enterprise" {
		return append(baseCerts, "ISO27001", "HIPAA")
	}

	if plan == "enterprise" {
		return append(baseCerts, "PCI-DSS", "FedRAMP")
	}

	return baseCerts
}

func (s *TenantV2Service) convertQuota(quota *models.TenantQuota) *TenantV2Quota {
	if quota == nil {
		return s.getDefaultQuota("free")
	}

	var periodStart, periodEnd time.Time
	if quota.PeriodStart != nil {
		periodStart = *quota.PeriodStart
	}
	if quota.PeriodEnd != nil {
		periodEnd = *quota.PeriodEnd
	}

	return &TenantV2Quota{
		MaxUsers:            int64(quota.MaxUsers),
		MaxApplications:     int64(quota.MaxApplications),
		MaxAPIRequests:      quota.MaxAPIRequests,
		MaxStorage:          quota.MaxStorage,
		MaxBandwidth:        quota.MaxBandwidth,
		MaxWebhooks:         int64(quota.MaxWebhooks),
		MaxRules:            int64(quota.MaxRules),
		MaxABTests:          int64(quota.MaxABTests),
		MaxCustomDomains:    1,
		MaxTeamMembers:      int64(quota.MaxUsers),
		MaxProjects:         10,
		MaxIntegrations:     5,
		RateLimitPerSecond:  100,
		RateLimitPerMinute:  5000,
		RateLimitPerDay:     quota.MaxAPIRequests,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		OverageAllowed:      false,
		OverageMultiplier:   1.5,
	}
}

func (s *TenantV2Service) getDefaultQuota(plan string) *TenantV2Quota {
	quotas := map[string]*TenantV2Quota{
		"free": {
			MaxUsers:           10,
			MaxApplications:    3,
			MaxAPIRequests:     100000,
			MaxStorage:         1073741824,
			MaxBandwidth:       1073741824,
			MaxWebhooks:        5,
			MaxRules:           20,
			MaxABTests:         2,
			MaxCustomDomains:   0,
			MaxTeamMembers:     5,
			MaxProjects:        5,
			MaxIntegrations:    3,
			RateLimitPerSecond: 10,
			RateLimitPerMinute: 500,
			RateLimitPerDay:    100000,
			OverageAllowed:     false,
			OverageMultiplier:  1.5,
		},
		"starter": {
			MaxUsers:           50,
			MaxApplications:    10,
			MaxAPIRequests:     1000000,
			MaxStorage:         10737418240,
			MaxBandwidth:       10737418240,
			MaxWebhooks:        20,
			MaxRules:           100,
			MaxABTests:         5,
			MaxCustomDomains:   2,
			MaxTeamMembers:     25,
			MaxProjects:        20,
			MaxIntegrations:    10,
			RateLimitPerSecond: 50,
			RateLimitPerMinute: 2500,
			RateLimitPerDay:    1000000,
			OverageAllowed:     true,
			OverageMultiplier:  1.5,
		},
		"pro": {
			MaxUsers:           200,
			MaxApplications:    50,
			MaxAPIRequests:     10000000,
			MaxStorage:         107374182400,
			MaxBandwidth:       107374182400,
			MaxWebhooks:        100,
			MaxRules:           500,
			MaxABTests:         20,
			MaxCustomDomains:   10,
			MaxTeamMembers:     100,
			MaxProjects:        100,
			MaxIntegrations:    50,
			RateLimitPerSecond: 200,
			RateLimitPerMinute: 10000,
			RateLimitPerDay:    10000000,
			OverageAllowed:     true,
			OverageMultiplier:  1.25,
		},
		"enterprise": {
			MaxUsers:           1000,
			MaxApplications:    200,
			MaxAPIRequests:     100000000,
			MaxStorage:         1073741824000,
			MaxBandwidth:       1073741824000,
			MaxWebhooks:        500,
			MaxRules:           2000,
			MaxABTests:         100,
			MaxCustomDomains:   50,
			MaxTeamMembers:     500,
			MaxProjects:        500,
			MaxIntegrations:    200,
			RateLimitPerSecond: 1000,
			RateLimitPerMinute: 50000,
			RateLimitPerDay:    100000000,
			OverageAllowed:     true,
			OverageMultiplier:  1.1,
		},
	}

	if quota, ok := quotas[plan]; ok {
		now := time.Now()
		quota.PeriodStart = now.AddDate(0, -1, 0)
		quota.PeriodEnd = now
		return quota
	}

	return quotas["free"]
}

func (s *TenantV2Service) generateQuota(plan string, config *TenantV2Config) *models.TenantQuota {
	v2Quota := s.getDefaultQuota(plan)

	periodStart := v2Quota.PeriodStart
	periodEnd := v2Quota.PeriodEnd

	return &models.TenantQuota{
		MaxUsers:        int(v2Quota.MaxUsers),
		MaxApplications: int(v2Quota.MaxApplications),
		MaxAPIRequests:  v2Quota.MaxAPIRequests,
		MaxStorage:      v2Quota.MaxStorage,
		MaxBandwidth:    v2Quota.MaxBandwidth,
		MaxWebhooks:     int(v2Quota.MaxWebhooks),
		MaxRules:        int(v2Quota.MaxRules),
		MaxABTests:      int(v2Quota.MaxABTests),
		PeriodStart:     &periodStart,
		PeriodEnd:       &periodEnd,
	}
}

func (s *TenantV2Service) getPermissions(plan string) []string {
	basePerms := []string{
		"dashboard:view",
		"stats:view",
		"users:view",
		"applications:view",
		"api:use",
	}

	if plan == "starter" || plan == "pro" || plan == "enterprise" {
		basePerms = append(basePerms, []string{
			"users:create",
			"users:edit",
			"applications:create",
			"applications:edit",
			"webhooks:manage",
		}...)
	}

	if plan == "pro" || plan == "enterprise" {
		basePerms = append(basePerms, []string{
			"rules:create",
			"rules:edit",
			"ab_tests:create",
			"ab_tests:manage",
			"analytics:advanced",
		}...)
	}

	if plan == "enterprise" {
		basePerms = append(basePerms, []string{
			"team:manage",
			"integrations:manage",
			"custom_domains:manage",
			"sso:configure",
			"audit:view",
			"export:data",
		}...)
	}

	return basePerms
}

func (s *TenantV2Service) getFeatures(plan string) map[string]bool {
	features := map[string]bool{
		"basic_analytics":    true,
		"api_access":         true,
		"email_support":      true,
		"custom_applications": plan != "free",
		"webhooks":           plan != "free",
		"advanced_rules":     plan == "pro" || plan == "enterprise",
		"ab_testing":         plan == "pro" || plan == "enterprise",
		"team_management":    plan == "enterprise",
		"sso":                plan == "enterprise",
		"custom_domains":     plan == "pro" || plan == "enterprise",
		"api_rate_increase":  plan == "pro" || plan == "enterprise",
		"dedicated_support":  plan == "enterprise",
		"audit_logs":         plan == "enterprise",
		"data_export":        plan == "pro" || plan == "enterprise",
		"sla_guarantee":      plan == "enterprise",
	}

	return features
}

func (s *TenantV2Service) createIsolatedDB(ctx context.Context, tenantID uint) error {
	return nil
}

func (s *TenantV2Service) createIsolatedCache(ctx context.Context, tenantID uint) error {
	return nil
}

func (s *TenantV2Service) createIsolatedStorage(ctx context.Context, tenantID uint) error {
	return nil
}

func (s *TenantV2Service) createResourcePolicies(ctx context.Context, tenantID uint) error {
	return nil
}

func (s *TenantV2Service) initializePermissions(ctx context.Context, tenantID uint, plan string) error {
	return nil
}

func (s *TenantV2Service) encodePermissions(perms []PermissionLevel) string {
	data, _ := json.Marshal(perms)
	return string(data)
}

func (s *TenantV2Service) decodePermissions(encoded string) []PermissionLevel {
	var perms []PermissionLevel
	json.Unmarshal([]byte(encoded), &perms)
	return perms
}

func (s *TenantV2Service) encodeConditions(conditions map[string]interface{}) string {
	if conditions == nil {
		return "{}"
	}
	data, _ := json.Marshal(conditions)
	return string(data)
}

func (s *TenantV2Service) decodeConditions(encoded string) map[string]interface{} {
	var conditions map[string]interface{}
	json.Unmarshal([]byte(encoded), &conditions)
	return conditions
}

func (s *TenantV2Service) checkConditions(conditions map[string]interface{}) bool {
	if conditions == nil || len(conditions) == 0 {
		return true
	}

	if ipRange, ok := conditions["ip_range"].(string); ok {
		if ipRange == "restricted" {
			return false
		}
	}

	return true
}

func (s *TenantV2Service) getAvailableActions(tenantID uint) []string {
	return []string{
		"view_dashboard",
		"manage_users",
		"manage_applications",
		"view_billing",
		"update_settings",
		"view_analytics",
		"export_data",
		"manage_integrations",
	}
}

func (s *TenantV2Service) getTenantMetric(ctx context.Context, tenantID uint, metric string, timeRange TimeRange) (float64, error) {
	baseValues := map[string]float64{
		"requests":        1000000,
		"success_rate":    0.95,
		"latency":         45.5,
		"error_rate":      0.02,
		"active_users":    500,
		"storage_used":    10737418240,
		"bandwidth_used":  5368709120,
	}

	if val, ok := baseValues[metric]; ok {
		return val, nil
	}
	return 0.0, fmt.Errorf("unknown metric: %s", metric)
}

func (s *TenantV2Service) getTenantMetrics(ctx context.Context, tenantID uint, metrics []string, timeRange TimeRange) (map[string]float64, error) {
	result := make(map[string]float64)
	for _, metric := range metrics {
		val, err := s.getTenantMetric(ctx, tenantID, metric, timeRange)
		if err == nil {
			result[metric] = val
		}
	}
	return result, nil
}

func (s *TenantV2Service) getTenantMetricAtTime(ctx context.Context, tenantID uint, metric string, timestamp time.Time) (float64, error) {
	baseValue := 1000.0
	variation := math.Sin(float64(timestamp.UnixNano())) * 100
	return baseValue + variation, nil
}

func (s *TenantV2Service) parseGranularity(granularity string) time.Duration {
	granularities := map[string]time.Duration{
		"minute": 1 * time.Minute,
		"hour":   1 * time.Hour,
		"day":    24 * time.Hour,
		"week":   7 * 24 * time.Hour,
	}

	if dur, ok := granularities[granularity]; ok {
		return dur
	}
	return 1 * time.Hour
}

func (s *TenantV2Service) getTotalUsers(tenantID uint) int64 {
	return 50
}

func (s *TenantV2Service) getTotalApplications(tenantID uint) int64 {
	return 5
}

func (s *TenantV2Service) getTotalAPIRequests(tenantID uint, timeRange TimeRange) int64 {
	return 1000000
}

func (s *TenantV2Service) getTotalStorage(tenantID uint) int64 {
	return 5368709120
}

func (s *TenantV2Service) getTotalBandwidth(tenantID uint, timeRange TimeRange) int64 {
	return 2147483648
}

func (s *TenantV2Service) getTopUsers(tenantID uint, limit int) []UserStat {
	users := make([]UserStat, 0)
	for i := 1; i <= limit && i <= 10; i++ {
		users = append(users, UserStat{
			UserID:     uint(i),
			Username:   fmt.Sprintf("user_%d", i),
			APIHits:    int64(10000 * (11 - i)),
			LastActive: time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}
	return users
}

func (s *TenantV2Service) getTopApplications(tenantID uint, limit int) []AppStat {
	apps := make([]AppStat, 0)
	for i := 1; i <= limit && i <= 10; i++ {
		apps = append(apps, AppStat{
			AppID:   uint(i),
			AppName: fmt.Sprintf("app_%d", i),
			APIHits: int64(50000 * (11 - i)),
			Errors:  int64(100 * i),
		})
	}
	return apps
}

func (s *TenantV2Service) getTrendData(tenantID uint, timeRange TimeRange) []TrendPointV2 {
	points := make([]TrendPointV2, 0)
	hours := int(timeRange.End.Sub(timeRange.Start).Hours())

	for i := 0; i < hours && i < 24; i++ {
		t := timeRange.Start.Add(time.Duration(i) * time.Hour)
		value := 1000.0 + float64(i)*50 + math.Sin(float64(i)*0.5)*100

		points = append(points, TrendPointV2{
			Timestamp: t,
			Value:     value,
			Label:     t.Format("15:04"),
		})
	}

	return points
}

func (s *TenantV2Service) exportAsCSV(data map[string]interface{}) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("tenant_id,metric,value\n")

	if tenant, ok := data["tenant"].(*TenantV2Context); ok {
		sb.WriteString(fmt.Sprintf("%d,name,%s\n", tenant.TenantID, tenant.TenantName))
	}

	if stats, ok := data["stats"].(*TenantV2Stats); ok {
		sb.WriteString(fmt.Sprintf("%d,total_users,%d\n", stats.TenantID, stats.TotalUsers))
		sb.WriteString(fmt.Sprintf("%d,total_apps,%d\n", stats.TenantID, stats.TotalApplications))
		sb.WriteString(fmt.Sprintf("%d,total_requests,%d\n", stats.TenantID, stats.TotalAPIRequests))
	}

	return []byte(sb.String()), nil
}

func (s *TenantV2Service) generateShortID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(result)
}
