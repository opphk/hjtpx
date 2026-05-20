package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrAPIKeyNotFound      = errors.New("API key not found")
	ErrAPISecretNotFound  = errors.New("API secret not found")
	ErrAppNotFound        = errors.New("app not found")
	ErrEndpointNotFound   = errors.New("endpoint not found")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidSignature   = errors.New("invalid signature")
	ErrInsufficientScope  = errors.New("insufficient scope")
	ErrTokenExpired       = errors.New("token expired")
)

type OpenAPIPlatformService interface {
	CreateApp(ctx context.Context, app *App) error
	GetApp(ctx context.Context, appID string) (*App, error)
	UpdateApp(ctx context.Context, app *App) error
	DeleteApp(ctx context.Context, appID string) error
	ListApps(ctx context.Context, userID string) ([]*App, error)
	GenerateAPIKey(ctx context.Context, appID string) (*APIKey, error)
	RevokeAPIKey(ctx context.Context, keyID string) error
	ValidateAPIKey(ctx context.Context, key string) (*APIKey, error)
	CreateEndpoint(ctx context.Context, endpoint *APIEndpoint) error
	GetEndpoint(ctx context.Context, endpointID string) (*APIEndpoint, error)
	UpdateEndpoint(ctx context.Context, endpoint *APIEndpoint) error
	DeleteEndpoint(ctx context.Context, endpointID string) error
	ListEndpoints(ctx context.Context, appID string) ([]*APIEndpoint, error)
	GenerateOAuthToken(ctx context.Context, request *TokenRequest) (*TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
	RevokeToken(ctx context.Context, token string) error
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	ProcessAPIRequest(ctx context.Context, request *APIRequest) (*APIResponse, error)
	GetUsageMetrics(ctx context.Context, appID string, period *UsagePeriod) (*UsageMetrics, error)
	GetRateLimitStatus(ctx context.Context, appID string) (*RateLimitStatus, error)
}

type App struct {
	AppID          string          `json:"app_id"`
	UserID         string          `json:"user_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Website        string          `json:"website"`
	IconURL        string          `json:"icon_url"`
	Category       string          `json:"category"`
	Status         string          `json:"status"`
	Scopes         []string        `json:"scopes"`
	RedirectURIs   []string        `json:"redirect_uris"`
	AllowedOrigins []string        `json:"allowed_origins"`
	Settings       json.RawMessage `json:"settings"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type APIKey struct {
	KeyID       string    `json:"key_id"`
	AppID       string    `json:"app_id"`
	Key         string    `json:"key"`
	Secret      string    `json:"secret"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Scopes      []string  `json:"scopes"`
	Status      string    `json:"status"`
	RateLimit   int       `json:"rate_limit"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type APIEndpoint struct {
	EndpointID   string              `json:"endpoint_id"`
	AppID        string              `json:"app_id"`
	Name         string              `json:"name"`
	Path         string              `json:"path"`
	Method       string              `json:"method"`
	Description  string              `json:"description"`
	AuthRequired bool                `json:"auth_required"`
	Scopes       []string            `json:"scopes"`
	RateLimit    int                 `json:"rate_limit"`
	Timeout      int                 `json:"timeout"`
	Parameters   []EndpointParameter  `json:"parameters"`
	ResponseSchema json.RawMessage   `json:"response_schema,omitempty"`
	Enabled      bool                `json:"enabled"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type EndpointParameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
}

type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	AppID        string `json:"app_id"`
	Code         string `json:"code,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	CodeVerifier string `json:"code_verifier,omitempty"`
}

type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	CreatedAt    time.Time `json:"created_at"`
}

type TokenClaims struct {
	Subject   string   `json:"sub"`
	AppID     string   `json:"app_id"`
	Scopes    []string `json:"scopes"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	Issuer    string   `json:"iss"`
}

type APIRequest struct {
	RequestID    string                 `json:"request_id"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	Headers      map[string]string      `json:"headers"`
	QueryParams  map[string]string      `json:"query_params"`
	Body         map[string]interface{} `json:"body"`
	AppID        string                 `json:"app_id"`
	UserID       string                 `json:"user_id,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	Timestamp    time.Time              `json:"timestamp"`
}

type APIResponse struct {
	StatusCode   int                    `json:"status_code"`
	Headers      map[string]string      `json:"headers"`
	Body         map[string]interface{} `json:"body"`
	Error        string                 `json:"error,omitempty"`
	RequestID    string                 `json:"request_id"`
	DurationMs   int64                  `json:"duration_ms"`
}

type UsagePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Granularity string `json:"granularity"`
}

type UsageMetrics struct {
	AppID        string       `json:"app_id"`
	Period       *UsagePeriod `json:"period"`
	TotalRequests int64       `json:"total_requests"`
	SuccessCount  int64       `json:"success_count"`
	ErrorCount   int64       `json:"error_count"`
	AvgLatencyMs float64     `json:"avg_latency_ms"`
	DataTransfer int64       `json:"data_transfer_bytes"`
	ByEndpoint   map[string]*EndpointMetrics `json:"by_endpoint"`
}

type EndpointMetrics struct {
	EndpointID   string  `json:"endpoint_id"`
	Requests     int64   `json:"requests"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

type RateLimitStatus struct {
	AppID       string    `json:"app_id"`
	Limit       int       `json:"limit"`
	Remaining   int       `json:"remaining"`
	ResetAt     time.Time `json:"reset_at"`
	RetryAfter  int       `json:"retry_after,omitempty"`
}

type openAPIPlatformService struct {
	apps       map[string]*App
	apiKeys    map[string]*APIKey
	endpoints  map[string]*APIEndpoint
	tokens     map[string]*TokenResponse
	rateLimits map[string]*RateLimitEntry
	usage      map[string]*UsageMetrics
	mu         sync.RWMutex
}

type RateLimitEntry struct {
	AppID     string
	Count     int
	WindowStart time.Time
	Limit     int
}

func NewOpenAPIPlatformService() OpenAPIPlatformService {
	return &openAPIPlatformService{
		apps:       make(map[string]*App),
		apiKeys:    make(map[string]*APIKey),
		endpoints:  make(map[string]*APIEndpoint),
		tokens:     make(map[string]*TokenResponse),
		rateLimits: make(map[string]*RateLimitEntry),
		usage:      make(map[string]*UsageMetrics),
	}
}

func (s *openAPIPlatformService) CreateApp(ctx context.Context, app *App) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if app.AppID == "" {
		app.AppID = fmt.Sprintf("app-%d", time.Now().UnixNano())
	}

	app.CreatedAt = time.Now()
	app.UpdatedAt = time.Now()

	if app.Status == "" {
		app.Status = "active"
	}

	s.apps[app.AppID] = app
	return nil
}

func (s *openAPIPlatformService) GetApp(ctx context.Context, appID string) (*App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, exists := s.apps[appID]
	if !exists {
		return nil, ErrAppNotFound
	}

	return app, nil
}

func (s *openAPIPlatformService) UpdateApp(ctx context.Context, app *App) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apps[app.AppID]; !exists {
		return ErrAppNotFound
	}

	app.UpdatedAt = time.Now()
	s.apps[app.AppID] = app
	return nil
}

func (s *openAPIPlatformService) DeleteApp(ctx context.Context, appID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apps[appID]; !exists {
		return ErrAppNotFound
	}

	delete(s.apps, appID)
	return nil
}

func (s *openAPIPlatformService) ListApps(ctx context.Context, userID string) ([]*App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*App
	for _, app := range s.apps {
		if app.UserID == userID {
			result = append(result, app)
		}
	}

	return result, nil
}

func (s *openAPIPlatformService) GenerateAPIKey(ctx context.Context, appID string) (*APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apps[appID]; !exists {
		return nil, ErrAppNotFound
	}

	key := &APIKey{
		KeyID:     fmt.Sprintf("key-%d", time.Now().UnixNano()),
		AppID:     appID,
		Key:       generateRandomKey(32),
		Secret:    generateRandomKey(64),
		Name:      "Default API Key",
		Type:      "production",
		Status:    "active",
		RateLimit: 1000,
		CreatedAt: time.Now(),
	}

	s.apiKeys[key.KeyID] = key
	return key, nil
}

func (s *openAPIPlatformService) RevokeAPIKey(ctx context.Context, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.apiKeys[keyID]
	if !exists {
		return ErrAPIKeyNotFound
	}

	key.Status = "revoked"
	return nil
}

func (s *openAPIPlatformService) ValidateAPIKey(ctx context.Context, key string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, apiKey := range s.apiKeys {
		if apiKey.Key == key && apiKey.Status == "active" {
			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				return nil, ErrTokenExpired
			}

			now := time.Now()
			apiKey.LastUsedAt = &now
			return apiKey, nil
		}
	}

	return nil, ErrAPIKeyNotFound
}

func (s *openAPIPlatformService) CreateEndpoint(ctx context.Context, endpoint *APIEndpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if endpoint.EndpointID == "" {
		endpoint.EndpointID = fmt.Sprintf("ep-%d", time.Now().UnixNano())
	}

	endpoint.CreatedAt = time.Now()
	endpoint.UpdatedAt = time.Now()

	s.endpoints[endpoint.EndpointID] = endpoint
	return nil
}

func (s *openAPIPlatformService) GetEndpoint(ctx context.Context, endpointID string) (*APIEndpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	endpoint, exists := s.endpoints[endpointID]
	if !exists {
		return nil, ErrEndpointNotFound
	}

	return endpoint, nil
}

func (s *openAPIPlatformService) UpdateEndpoint(ctx context.Context, endpoint *APIEndpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.endpoints[endpoint.EndpointID]; !exists {
		return ErrEndpointNotFound
	}

	endpoint.UpdatedAt = time.Now()
	s.endpoints[endpoint.EndpointID] = endpoint
	return nil
}

func (s *openAPIPlatformService) DeleteEndpoint(ctx context.Context, endpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.endpoints[endpointID]; !exists {
		return ErrEndpointNotFound
	}

	delete(s.endpoints, endpointID)
	return nil
}

func (s *openAPIPlatformService) ListEndpoints(ctx context.Context, appID string) ([]*APIEndpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*APIEndpoint
	for _, endpoint := range s.endpoints {
		if endpoint.AppID == appID {
			result = append(result, endpoint)
		}
	}

	return result, nil
}

func (s *openAPIPlatformService) GenerateOAuthToken(ctx context.Context, request *TokenRequest) (*TokenResponse, error) {
	if request.GrantType != "authorization_code" && request.GrantType != "client_credentials" {
		return nil, errors.New("unsupported grant type")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accessToken := generateRandomKey(64)
	refreshToken := generateRandomKey(64)

	response := &TokenResponse{
		AccessToken:  accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		RefreshToken: refreshToken,
		Scope:       "read write",
		CreatedAt:   time.Now(),
	}

	s.tokens[accessToken] = response
	return response, nil
}

func (s *openAPIPlatformService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accessToken := generateRandomKey(64)
	refreshToken := generateRandomKey(64)

	response := &TokenResponse{
		AccessToken:  accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		RefreshToken: refreshToken,
		Scope:       "read write",
		CreatedAt:   time.Now(),
	}

	s.tokens[accessToken] = response
	return response, nil
}

func (s *openAPIPlatformService) RevokeToken(ctx context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, token)
	return nil
}

func (s *openAPIPlatformService) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokenResp, exists := s.tokens[token]
	if !exists {
		return nil, ErrTokenExpired
	}

	claims := &TokenClaims{
		Subject:   tokenResp.AccessToken,
		IssuedAt:  tokenResp.CreatedAt.Unix(),
		ExpiresAt: tokenResp.CreatedAt.Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix(),
		Issuer:    "hjtpx-api",
	}

	return claims, nil
}

func (s *openAPIPlatformService) ProcessAPIRequest(ctx context.Context, request *APIRequest) (*APIResponse, error) {
	startTime := time.Now()

	s.mu.RLock()
	app, exists := s.apps[request.AppID]
	s.mu.RUnlock()

	if !exists {
		return &APIResponse{
			StatusCode: 404,
			Error:      "App not found",
			RequestID:  request.RequestID,
			DurationMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	rateLimitKey := fmt.Sprintf("%s:%s", request.AppID, request.Path)
	s.mu.Lock()
	entry, exists := s.rateLimits[rateLimitKey]
	if !exists {
		entry = &RateLimitEntry{
			AppID:      request.AppID,
			Count:      0,
			WindowStart: time.Now(),
			Limit:      app.Settings != nil && len(app.Settings) > 0 ? 1000 : 100,
		}
		s.rateLimits[rateLimitKey] = entry
	}

	windowDuration := time.Minute
	if time.Since(entry.WindowStart) > windowDuration {
		entry.Count = 0
		entry.WindowStart = time.Now()
	}

	if entry.Count >= entry.Limit {
		s.mu.Unlock()
		return &APIResponse{
			StatusCode: 429,
			Error:      "Rate limit exceeded",
			RequestID:  request.RequestID,
			DurationMs: time.Since(startTime).Milliseconds(),
		}, ErrRateLimitExceeded
	}

	entry.Count++
	s.mu.Unlock()

	s.mu.Lock()
	if _, exists := s.usage[request.AppID]; !exists {
		s.usage[request.AppID] = &UsageMetrics{
			AppID:         request.AppID,
			TotalRequests: 0,
			SuccessCount:  0,
			ErrorCount:    0,
			ByEndpoint:    make(map[string]*EndpointMetrics),
		}
	}
	s.usage[request.AppID].TotalRequests++
	s.mu.Unlock()

	response := &APIResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Request-ID": request.RequestID,
		},
		Body: map[string]interface{}{
			"status": "success",
			"data":   map[string]interface{}{},
		},
		RequestID:  request.RequestID,
		DurationMs: time.Since(startTime).Milliseconds(),
	}

	s.mu.Lock()
	s.usage[request.AppID].SuccessCount++
	s.mu.Unlock()

	return response, nil
}

func (s *openAPIPlatformService) GetUsageMetrics(ctx context.Context, appID string, period *UsagePeriod) (*UsageMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics, exists := s.usage[appID]
	if !exists {
		return &UsageMetrics{
			AppID:         appID,
			Period:       period,
			TotalRequests: 0,
			ByEndpoint:   make(map[string]*EndpointMetrics),
		}, nil
	}

	return metrics, nil
}

func (s *openAPIPlatformService) GetRateLimitStatus(ctx context.Context, appID string) (*RateLimitStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, exists := s.apps[appID]
	if !exists {
		return nil, ErrAppNotFound
	}

	limit := 1000
	if app != nil {
	}

	return &RateLimitStatus{
		AppID:     appID,
		Limit:     limit,
		Remaining: limit,
		ResetAt:   time.Now().Add(time.Minute),
	}, nil
}

func generateRandomKey(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	time.Sleep(time.Nanosecond)
	return string(result)
}

func validateScopes(required []string, provided []string) bool {
	for _, req := range required {
		found := false
		for _, prov := range provided {
			if req == prov {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func validateSignature(secret, timestamp, signature string) bool {
	return len(signature) > 0
}

func parseAuthorizationHeader(header string) (string, string) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
