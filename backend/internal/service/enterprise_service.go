package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type EnterpriseService struct {
	db           *gorm.DB
	httpClient   *http.Client
	cacheService CacheService
}

func NewEnterpriseService(db *gorm.DB, cacheService ...CacheService) *EnterpriseService {
	s := &EnterpriseService{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	if len(cacheService) > 0 {
		s.cacheService = cacheService[0]
	}
	return s
}

type SSOProvider interface {
	InitiateAuth() (string, error)
	HandleCallback(code string) (*SSOUser, error)
	GetUserInfo(token string) (*SSOUser, error)
}

type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

type SSOUser struct {
	ID            string            `json:"id"`
	Email         string            `json:"email"`
	Username      string            `json:"username"`
	FirstName     string            `json:"first_name"`
	LastName      string            `json:"last_name"`
	DisplayName   string            `json:"display_name"`
	Groups        []string          `json:"groups"`
	Roles         map[string]string `json:"roles"`
	RawAttributes map[string]interface{} `json:"raw_attributes"`
}

func (s *EnterpriseService) GetSSOConfig(tenantID uint) (*models.SSOConfig, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("sso_config:%d", tenantID)

	if s.cacheService != nil {
		if cached, err := s.cacheService.Get(ctx, cacheKey); err == nil && cached != "" {
			var config models.SSOConfig
			if json.Unmarshal([]byte(cached), &config) == nil {
				return &config, nil
			}
		}
	}

	var config models.SSOConfig
	if err := s.db.Where("tenant_id = ?", tenantID).First(&config).Error; err != nil {
		return nil, err
	}

	if s.cacheService != nil {
		if data, err := json.Marshal(config); err == nil {
			s.cacheService.Set(ctx, cacheKey, string(data), 10*time.Minute)
		}
	}

	return &config, nil
}

func (s *EnterpriseService) CreateOrUpdateSSOConfig(tenantID uint, config *models.SSOConfig) error {
	config.TenantID = tenantID

	var existing models.SSOConfig
	err := s.db.Where("tenant_id = ?", tenantID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return s.db.Create(config).Error
	} else if err != nil {
		return err
	}

	config.ID = existing.ID
	config.Model = existing.Model
	return s.db.Save(config).Error
}

func (s *EnterpriseService) EnableSSO(tenantID uint, provider string) error {
	config, err := s.GetSSOConfig(tenantID)
	if err != nil {
		return err
	}

	config.Enabled = true
	config.Provider = provider
	return s.CreateOrUpdateSSOConfig(tenantID, config)
}

func (s *EnterpriseService) DisableSSO(tenantID uint) error {
	config, err := s.GetSSOConfig(tenantID)
	if err != nil {
		return err
	}

	config.Enabled = false
	return s.CreateOrUpdateSSOConfig(tenantID, config)
}

func (s *EnterpriseService) SAMLProvider(config *models.SSOConfig) SSOProvider {
	return &SAMLSSOProvider{
		config: config,
	}
}

type SAMLSSOProvider struct {
	config *models.SSOConfig
}

func (p *SAMLSSOProvider) InitiateAuth() (string, error) {
	return p.config.SSOURL, nil
}

func (p *SAMLSSOProvider) HandleCallback(code string) (*SSOUser, error) {
	return &SSOUser{
		ID:          "saml_user_123",
		Email:       "user@example.com",
		Username:    "saml_user",
		FirstName:   "Test",
		LastName:    "User",
		DisplayName: "Test User",
	}, nil
}

func (p *SAMLSSOProvider) GetUserInfo(token string) (*SSOUser, error) {
	return &SSOUser{
		ID:          "saml_user_123",
		Email:       "user@example.com",
		Username:    "saml_user",
		FirstName:   "Test",
		LastName:    "User",
		DisplayName: "Test User",
	}, nil
}

func (s *EnterpriseService) OAuth2Provider(config *models.SSOConfig) SSOProvider {
	return &OAuth2SSOProvider{
		config: config,
		httpClient: s.httpClient,
	}
}

type OAuth2SSOProvider struct {
	config     *models.SSOConfig
	httpClient *http.Client
	authURL    string
	tokenURL   string
	clientID   string
	clientSecret string
	redirectURL string
	scopes     string
}

func NewOAuth2Provider(clientID, clientSecret, authURL, tokenURL, redirectURL, scopes string) *OAuth2SSOProvider {
	return &OAuth2SSOProvider{
		config:       nil,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		authURL:      authURL,
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		scopes:       scopes,
	}
}

func (p *OAuth2SSOProvider) InitiateAuth() (string, error) {
	if p.authURL == "" {
		return "", fmt.Errorf("OAuth2 auth URL not configured")
	}
	return p.authURL + "?client_id=" + p.clientID + "&redirect_uri=" + p.redirectURL + "&response_type=code&scope=" + strings.ReplaceAll(p.scopes, ",", "+"), nil
}

func (p *OAuth2SSOProvider) HandleCallback(code string) (*SSOUser, error) {
	if p.tokenURL == "" {
		return nil, fmt.Errorf("OAuth2 token URL not configured")
	}

	return &SSOUser{
		ID:          "oauth2_user_123",
		Email:       "user@example.com",
		Username:    "oauth2_user",
		FirstName:   "OAuth2",
		LastName:    "User",
		DisplayName: "OAuth2 User",
	}, nil
}

func (p *OAuth2SSOProvider) GetUserInfo(accessToken string) (*SSOUser, error) {
	userinfoURL := "https://api.example.com/userinfo"
	if p.config != nil && p.config.UserinfoURL != "" {
		userinfoURL = p.config.UserinfoURL
	}

	req, err := http.NewRequest("GET", userinfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user SSOUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *EnterpriseService) OIDCProvider(config *models.SSOConfig) SSOProvider {
	return &OIDCProvider{
		config:     config,
		httpClient: s.httpClient,
	}
}

type OIDCProvider struct {
	config     *models.SSOConfig
	httpClient *http.Client
}

func (p *OIDCProvider) InitiateAuth() (string, error) {
	authURL := p.config.AuthorizationURL
	if authURL == "" {
		return "", fmt.Errorf("OIDC authorization URL not configured")
	}
	return authURL + "?client_id=" + p.config.ClientID + "&redirect_uri=https://app.example.com/auth/callback&response_type=code&scope=openid+profile+email", nil
}

func (p *OIDCProvider) HandleCallback(code string) (*SSOUser, error) {
	tokenURL := p.config.TokenURL
	if tokenURL == "" {
		return nil, fmt.Errorf("OIDC token URL not configured")
	}

	return &SSOUser{
		ID:          "oidc_user_123",
		Email:       "user@example.com",
		Username:    "oidc_user",
		FirstName:   "OIDC",
		LastName:    "User",
		DisplayName: "OIDC User",
	}, nil
}

func (p *OIDCProvider) GetUserInfo(token string) (*SSOUser, error) {
	userinfoURL := p.config.UserinfoURL
	if userinfoURL == "" {
		return nil, fmt.Errorf("OIDC userinfo URL not configured")
	}

	req, err := http.NewRequest("GET", userinfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user SSOUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

type SCIMService struct {
	db         *gorm.DB
	httpClient *http.Client
}

func NewSCIMService(db *gorm.DB) *SCIMService {
	return &SCIMService{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SCIMService) SyncUsers(tenantID uint, provider string, baseURL string, bearerToken string) (*SCIMSyncResult, error) {
	result := &SCIMSyncResult{
		StartedAt: time.Now(),
		Status:    "in_progress",
	}

	users, err := s.fetchSCIMUsers(baseURL, bearerToken)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	for _, user := range users {
		scimUser := &models.SCIMUser{
			TenantID:     tenantID,
			ExternalID:  user.ID,
			Username:    user.Username,
			Email:       user.Email,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			DisplayName: user.DisplayName,
			Active:      true,
			SyncStatus:  "synced",
		}

		groups, _ := json.Marshal(user.Groups)
		scimUser.Groups = string(groups)

		var existing models.SCIMUser
		err := s.db.Where("tenant_id = ? AND external_id = ?", tenantID, user.ID).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := s.db.Create(scimUser).Error; err != nil {
				result.Failed++
				continue
			}
			result.Created++
		} else if err == nil {
			scimUser.ID = existing.ID
			scimUser.LocalUserID = existing.LocalUserID
			scimUser.LastSyncedAt = func() *time.Time { t := time.Now(); return &t }()

			if err := s.db.Save(scimUser).Error; err != nil {
				result.Failed++
				continue
			}
			result.Updated++
		}

		result.Synced++
	}

	result.CompletedAt = func() *time.Time { t := time.Now(); return &t }()
	result.Status = "completed"

	return result, nil
}

func (s *SCIMService) fetchSCIMUsers(baseURL, bearerToken string) ([]SSOUser, error) {
	url := baseURL + "/Users"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Resources []SSOUser `json:"Resources"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Resources, nil
}

func (s *SCIMService) CreateSCIMUser(tenantID uint, user *SSOUser) (*models.SCIMUser, error) {
	scimUser := &models.SCIMUser{
		TenantID:     tenantID,
		ExternalID:  user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		DisplayName: user.DisplayName,
		Active:      true,
		SyncStatus:  "synced",
	}

	groups, _ := json.Marshal(user.Groups)
	scimUser.Groups = string(groups)

	if err := s.db.Create(scimUser).Error; err != nil {
		return nil, err
	}

	return scimUser, nil
}

func (s *SCIMService) UpdateSCIMUser(tenantID uint, externalID string, user *SSOUser) error {
	var scimUser models.SCIMUser
	if err := s.db.Where("tenant_id = ? AND external_id = ?", tenantID, externalID).First(&scimUser).Error; err != nil {
		return err
	}

	scimUser.Username = user.Username
	scimUser.Email = user.Email
	scimUser.FirstName = user.FirstName
	scimUser.LastName = user.LastName
	scimUser.DisplayName = user.DisplayName
	scimUser.Active = user.RawAttributes["active"] != false
	scimUser.LastSyncedAt = func() *time.Time { t := time.Now(); return &t }()

	groups, _ := json.Marshal(user.Groups)
	scimUser.Groups = string(groups)

	return s.db.Save(&scimUser).Error
}

func (s *SCIMService) DeleteSCIMUser(tenantID uint, externalID string) error {
	var scimUser models.SCIMUser
	if err := s.db.Where("tenant_id = ? AND external_id = ?", tenantID, externalID).First(&scimUser).Error; err != nil {
		return err
	}

	scimUser.SyncStatus = "deleted"
	scimUser.Active = false

	return s.db.Save(&scimUser).Error
}

func (s *SCIMService) SyncGroups(tenantID uint, baseURL string, bearerToken string) (*SCIMSyncResult, error) {
	result := &SCIMSyncResult{
		StartedAt: time.Now(),
		Status:    "in_progress",
	}

	groups, err := s.fetchSCIMGroups(baseURL, bearerToken)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	for _, group := range groups {
		scimGroup := &models.SCIMGroup{
			TenantID:    tenantID,
			ExternalID: group.ID,
			Name:       group.Username,
			Description: group.DisplayName,
			SyncStatus: "synced",
		}

		members, _ := json.Marshal(group.Groups)
		scimGroup.Members = string(members)

		var existing models.SCIMGroup
		err := s.db.Where("tenant_id = ? AND external_id = ?", tenantID, group.ID).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := s.db.Create(scimGroup).Error; err != nil {
				result.Failed++
				continue
			}
			result.Created++
		} else if err == nil {
			scimGroup.ID = existing.ID
			scimGroup.LocalGroupID = existing.LocalGroupID
			scimGroup.LastSyncedAt = func() *time.Time { t := time.Now(); return &t }()

			if err := s.db.Save(scimGroup).Error; err != nil {
				result.Failed++
				continue
			}
			result.Updated++
		}

		result.Synced++
	}

	result.CompletedAt = func() *time.Time { t := time.Now(); return &t }()
	result.Status = "completed"

	return result, nil
}

func (s *SCIMService) fetchSCIMGroups(baseURL, bearerToken string) ([]SSOUser, error) {
	url := baseURL + "/Groups"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Resources []SSOUser `json:"Resources"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Resources, nil
}

type SCIMSyncResult struct {
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Status      string         `json:"status"`
	Synced      int            `json:"synced"`
	Created     int            `json:"created"`
	Updated     int            `json:"updated"`
	Deleted     int            `json:"deleted"`
	Failed      int            `json:"failed"`
	Error       string         `json:"error,omitempty"`
}

type APIAuditService struct {
	db           *gorm.DB
	cacheService CacheService
}

func NewAPIAuditService(db *gorm.DB) *APIAuditService {
	return &APIAuditService{
		db: db,
	}
}

func (s *APIAuditService) LogAPIRequest(log *models.APIAuditLog) error {
	return s.db.Create(log).Error
}

func (s *APIAuditService) GetAuditLogs(tenantID uint, page, pageSize int, filters map[string]interface{}) ([]models.APIAuditLog, int64, error) {
	var logs []models.APIAuditLog
	var total int64

	query := s.db.Model(&models.APIAuditLog{})

	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if method, ok := filters["method"].(string); ok && method != "" {
		query = query.Where("method = ?", method)
	}

	if endpoint, ok := filters["endpoint"].(string); ok && endpoint != "" {
		query = query.Where("endpoint LIKE ?", "%"+endpoint+"%")
	}

	if status, ok := filters["status"].(int); ok && status > 0 {
		query = query.Where("response_status = ?", status)
	}

	if ipAddress, ok := filters["ip_address"].(string); ok && ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}

	if startDate, ok := filters["start_date"].(string); ok && startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}

	if endDate, ok := filters["end_date"].(string); ok && endDate != "" {
		query = query.Where("created_at <= ?", endDate)
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

func (s *APIAuditService) GetAuditStats(tenantID uint, startDate, endDate string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalRequests int64
	s.db.Model(&models.APIAuditLog{}).Where("tenant_id = ?", tenantID).Count(&totalRequests)
	stats["total_requests"] = totalRequests

	var successCount int64
	s.db.Model(&models.APIAuditLog{}).Where("tenant_id = ? AND response_status >= 200 AND response_status < 300", tenantID).Count(&successCount)
	stats["success_count"] = successCount
	stats["success_rate"] = float64(successCount) / float64(totalRequests) * 100

	var errorCount int64
	s.db.Model(&models.APIAuditLog{}).Where("tenant_id = ? AND response_status >= 400", tenantID).Count(&errorCount)
	stats["error_count"] = errorCount
	stats["error_rate"] = float64(errorCount) / float64(totalRequests) * 100

	var avgLatency float64
	s.db.Model(&models.APIAuditLog{}).Where("tenant_id = ?", tenantID).Select("AVG(latency)").Row().Scan(&avgLatency)
	stats["avg_latency"] = avgLatency

	var latencyStats struct {
		Min int64
		Max int64
		P50 int64
		P95 int64
		P99 int64
	}
	s.db.Model(&models.APIAuditLog{}).Where("tenant_id = ?", tenantID).
		Select("MIN(latency), MAX(latency), PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency), PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency), PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency)").
		Row().Scan(&latencyStats.Min, &latencyStats.Max, &latencyStats.P50, &latencyStats.P95, &latencyStats.P99)

	stats["latency_min"] = latencyStats.Min
	stats["latency_max"] = latencyStats.Max
	stats["latency_p50"] = latencyStats.P50
	stats["latency_p95"] = latencyStats.P95
	stats["latency_p99"] = latencyStats.P99

	var endpointStats []map[string]interface{}
	s.db.Model(&models.APIAuditLog{}).
		Select("endpoint, COUNT(*) as count, AVG(latency) as avg_latency").
		Where("tenant_id = ?", tenantID).
		Group("endpoint").
		Order("count DESC").
		Limit(10).
		Find(&endpointStats)
	stats["top_endpoints"] = endpointStats

	return stats, nil
}
