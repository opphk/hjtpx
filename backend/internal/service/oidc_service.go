package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/pkg/logger"
)

var (
	ErrOIDCProviderNotFound = errors.New("oidc provider not found")
	ErrOIDCExchangeFailed   = errors.New("oidc code exchange failed")
	ErrOIDCInvalidState     = errors.New("oidc invalid state parameter")
	ErrOIDCUserInfoFailed   = errors.New("oidc user info request failed")
)

type OIDCProviderType string

const (
	OIDCProviderGeneric OIDCProviderType = "generic"
)

type OIDCProviderConfig struct {
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	RedirectURI    string   `json:"redirect_uri"`
	IssuerURL      string   `json:"issuer_url"`
	AuthorizationURL string `json:"authorization_url,omitempty"`
	TokenURL       string   `json:"token_url,omitempty"`
	UserInfoURL    string   `json:"user_info_url,omitempty"`
	EndSessionURL  string   `json:"end_session_url,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	ExtraParams    map[string]string `json:"extra_params,omitempty"`
}

type OIDCUserInfo struct {
	Provider    string                 `json:"provider"`
	ProviderID  string                 `json:"provider_id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	Name        string                 `json:"name,omitempty"`
	AvatarURL   string                 `json:"avatar_url,omitempty"`
	EmailVerified bool                 `json:"email_verified"`
	RawData     map[string]interface{} `json:"raw_data,omitempty"`
}

type OIDCState struct {
	State     string    `json:"state"`
	Provider  string    `json:"provider"`
	Nonce     string    `json:"nonce"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type OIDCService struct {
	provider   OIDCProviderType
	config     *OIDCProviderConfig
	token      *OIDCTokenResponse
	tokenMu    sync.RWMutex
	logger     *logger.Logger
	httpClient *http.Client
}

type OIDCServiceManager struct {
	providers    map[string]*OIDCService
	states       map[string]*OIDCState
	stateTTL     time.Duration
	mu           sync.RWMutex
	logger       *logger.Logger
}

var oidcManager *OIDCServiceManager
var oidcOnce sync.Once

func GetOIDCManager() *OIDCServiceManager {
	oidcOnce.Do(func() {
		oidcManager = &OIDCServiceManager{
			providers: make(map[string]*OIDCService),
			states:    make(map[string]*OIDCState),
			stateTTL:  10 * time.Minute,
			logger:    logger.Default(),
		}
	})
	return oidcManager
}

func (m *OIDCServiceManager) RegisterProvider(name string, config *OIDCProviderConfig) error {
	if config.ClientID == "" || config.ClientSecret == "" || config.IssuerURL == "" {
		return fmt.Errorf("client_id, client_secret and issuer_url are required")
	}

	service := &OIDCService{
		provider: OIDCProviderGeneric,
		config:   config,
		logger:   logger.Default(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if config.AuthorizationURL == "" {
		config.AuthorizationURL = config.IssuerURL + "/authorize"
	}
	if config.TokenURL == "" {
		config.TokenURL = config.IssuerURL + "/token"
	}
	if config.UserInfoURL == "" {
		config.UserInfoURL = config.IssuerURL + "/userinfo"
	}
	if config.EndSessionURL == "" {
		config.EndSessionURL = config.IssuerURL + "/logout"
	}
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "email", "profile"}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers[name] = service
	m.logger.Log(logger.INFO, "OIDC provider registered", logger.Fields{"provider": name})
	return nil
}

func (m *OIDCServiceManager) UnregisterProvider(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.providers, name)
	m.logger.Log(logger.INFO, "OIDC provider unregistered", logger.Fields{"provider": name})
}

func (m *OIDCServiceManager) GetProvider(name string) (*OIDCService, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		return nil, ErrOIDCProviderNotFound
	}
	return provider, nil
}

func (m *OIDCServiceManager) GenerateState(providerName string) (string, string, *OIDCState, error) {
	stateStr, err := generateRandomState()
	if err != nil {
		return "", "", nil, err
	}

	nonce, err := generateRandomState()
	if err != nil {
		return "", "", nil, err
	}

	state := &OIDCState{
		State:     stateStr,
		Provider:  providerName,
		Nonce:     nonce,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(m.stateTTL),
	}

	m.mu.Lock()
	m.states[stateStr] = state
	m.mu.Unlock()

	return stateStr, nonce, state, nil
}

func (m *OIDCServiceManager) ValidateState(stateStr string) (*OIDCState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.states[stateStr]
	if !exists {
		return nil, ErrOIDCInvalidState
	}

	if time.Now().After(state.ExpiresAt) {
		delete(m.states, stateStr)
		return nil, ErrOIDCInvalidState
	}

	delete(m.states, stateStr)
	return state, nil
}

func (m *OIDCServiceManager) CleanupExpiredStates() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for stateStr, state := range m.states {
		if now.After(state.ExpiresAt) {
			delete(m.states, stateStr)
		}
	}
}

func (s *OIDCService) GetAuthorizationURL(state, nonce string) (string, error) {
	authURL, err := url.Parse(s.config.AuthorizationURL)
	if err != nil {
		return "", err
	}

	params := authURL.Query()
	params.Set("client_id", s.config.ClientID)
	params.Set("redirect_uri", s.config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("nonce", nonce)
	params.Set("scope", strings.Join(s.config.Scopes, " "))

	for key, value := range s.config.ExtraParams {
		params.Set(key, value)
	}

	authURL.RawQuery = params.Encode()
	return authURL.String(), nil
}

func (s *OIDCService) ExchangeCode(code string) (*OIDCTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", s.config.RedirectURI)
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)

	resp, err := s.httpClient.PostForm(s.config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCExchangeFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOIDCExchangeFailed, resp.StatusCode, string(body))
	}

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response", ErrOIDCExchangeFailed)
	}

	s.tokenMu.Lock()
	s.token = &tokenResp
	s.tokenMu.Unlock()

	s.logger.Log(logger.INFO, "OIDC token exchanged successfully", logger.Fields{"provider": string(s.provider)})
	return &tokenResp, nil
}

func (s *OIDCService) RefreshToken(refreshToken string) (*OIDCTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)

	resp, err := s.httpClient.PostForm(s.config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCExchangeFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOIDCExchangeFailed, resp.StatusCode, string(body))
	}

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response", ErrOIDCExchangeFailed)
	}

	s.tokenMu.Lock()
	s.token = &tokenResp
	s.tokenMu.Unlock()

	s.logger.Log(logger.INFO, "OIDC token refreshed successfully", logger.Fields{"provider": string(s.provider)})
	return &tokenResp, nil
}

func (s *OIDCService) GetUserInfo(accessToken string) (*OIDCUserInfo, error) {
	req, err := http.NewRequest("GET", s.config.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCUserInfoFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOIDCUserInfoFailed, resp.StatusCode, string(body))
	}

	var rawData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("%w: failed to decode user info", ErrOIDCUserInfoFailed)
	}

	userInfo := &OIDCUserInfo{
		Provider:    string(s.provider),
		ProviderID:  getStringValue(rawData, "sub"),
		Username:    getStringValue(rawData, "preferred_username", "name", "email"),
		Email:       getStringValue(rawData, "email"),
		Name:        getStringValue(rawData, "name"),
		AvatarURL:   getStringValue(rawData, "picture", "avatar"),
		EmailVerified: getBoolValue(rawData, "email_verified"),
		RawData:     rawData,
	}

	s.logger.Log(logger.INFO, "OIDC user info retrieved", logger.Fields{"provider": string(s.provider), "user": userInfo.Username})
	return userInfo, nil
}

func (s *OIDCService) RevokeToken(token string) error {
	if s.config.EndSessionURL == "" {
		return nil
	}

	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)

	resp, err := s.httpClient.PostForm(s.config.EndSessionURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("revoke failed with status %d", resp.StatusCode)
	}

	s.tokenMu.Lock()
	s.token = nil
	s.tokenMu.Unlock()

	s.logger.Log(logger.INFO, "OIDC token revoked", logger.Fields{"provider": string(s.provider)})
	return nil
}

func (s *OIDCService) GetEndSessionURL(postLogoutRedirectURI string) (string, error) {
	if s.config.EndSessionURL == "" {
		return "", fmt.Errorf("end session URL not configured")
	}

	endSessionURL, err := url.Parse(s.config.EndSessionURL)
	if err != nil {
		return "", err
	}

	params := endSessionURL.Query()
	params.Set("client_id", s.config.ClientID)
	if postLogoutRedirectURI != "" {
		params.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}

	endSessionURL.RawQuery = params.Encode()
	return endSessionURL.String(), nil
}

func (s *OIDCService) GetToken() *OIDCTokenResponse {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()
	return s.token
}

func (s *OIDCService) IsTokenExpired() bool {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()

	if s.token == nil || s.token.ExpiresIn == 0 {
		return true
	}

	return false
}

func getStringValue(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, exists := data[key]; exists {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

func getBoolValue(data map[string]interface{}, key string) bool {
	if val, exists := data[key]; exists {
		return val.(bool)
	}
	return false
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (m *OIDCServiceManager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, 0, len(m.providers))
	for provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers
}

func (m *OIDCServiceManager) IsProviderEnabled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.providers[name]
	return exists
}