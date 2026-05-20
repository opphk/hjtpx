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

	github.com/hjtpx/hjtpx/internal/pkg/logger"
)

var (
	ErrOAuth2ProviderNotFound = errors.New("oauth2 provider not found")
	ErrOAuth2ExchangeFailed   = errors.New("oauth2 code exchange failed")
	ErrOAuth2RefreshFailed    = errors.New("oauth2 token refresh failed")
	ErrOAuth2InvalidState     = errors.New("oauth2 invalid state parameter")
	ErrOAuth2UserInfoFailed   = errors.New("oauth2 user info request failed")
)

type OAuth2ProviderType string

const (
	ProviderGitHub OAuth2ProviderType = "github"
	ProviderGoogle OAuth2ProviderType = "google"
)

type OAuth2ProviderConfig struct {
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	RedirectURI    string   `json:"redirect_uri"`
	Scopes         []string `json:"scopes,omitempty"`
	AuthURL        string   `json:"auth_url,omitempty"`
	TokenURL       string   `json:"token_url,omitempty"`
	UserInfoURL    string   `json:"user_info_url,omitempty"`
	RevokeURL      string   `json:"revoke_url,omitempty"`
	ExtraParams    map[string]string `json:"extra_params,omitempty"`
}

type OAuth2UserInfo struct {
	Provider    string            `json:"provider"`
	ProviderID  string            `json:"provider_id"`
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	Name        string            `json:"name,omitempty"`
	AvatarURL   string            `json:"avatar_url,omitempty"`
	RawData     map[string]interface{} `json:"raw_data,omitempty"`
}

type OAuth2State struct {
	State     string    `json:"state"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

type OAuth2ServiceManager struct {
	providers    map[OAuth2ProviderType]*OAuth2Service
	states       map[string]*OAuth2State
	stateTTL     time.Duration
	mu           sync.RWMutex
	logger       *logger.Logger
}

type OAuth2Service struct {
	provider   OAuth2ProviderType
	config     *OAuth2ProviderConfig
	token      *OAuth2TokenResponse
	tokenMu    sync.RWMutex
	logger     *logger.Logger
}

var oauth2Manager *OAuth2ServiceManager
var oauth2Once sync.Once

func GetOAuth2Manager() *OAuth2ServiceManager {
	oauth2Once.Do(func() {
		oauth2Manager = &OAuth2ServiceManager{
		providers: make(map[OAuth2ProviderType]*OAuth2Service),
		states:    make(map[string]*OAuth2State),
		stateTTL:  10 * time.Minute,
		logger:    logger.Default(),
	}
	})
	return oauth2Manager
}

func (m *OAuth2ServiceManager) RegisterProvider(providerType OAuth2ProviderType, config *OAuth2ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.ClientID == "" || config.ClientSecret == "" {
		return fmt.Errorf("client_id and client_secret are required")
	}

	service := &OAuth2Service{
		provider: providerType,
		config:   config,
		logger:   logger.Default(),
	}

	switch providerType {
	case ProviderGitHub:
		if config.AuthURL == "" {
			config.AuthURL = "https://github.com/login/oauth/authorize"
		}
		if config.TokenURL == "" {
			config.TokenURL = "https://github.com/login/oauth/access_token"
		}
		if config.UserInfoURL == "" {
			config.UserInfoURL = "https://api.github.com/user"
		}
		if len(config.Scopes) == 0 {
			config.Scopes = []string{"read:user", "user:email"}
		}

	case ProviderGoogle:
		if config.AuthURL == "" {
			config.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
		}
		if config.TokenURL == "" {
			config.TokenURL = "https://oauth2.googleapis.com/token"
		}
		if config.UserInfoURL == "" {
			config.UserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
		}
		if config.RevokeURL == "" {
			config.RevokeURL = "https://oauth2.googleapis.com/revoke"
		}
		if len(config.Scopes) == 0 {
			config.Scopes = []string{"openid", "email", "profile"}
		}
	}

	m.providers[providerType] = service
	m.logger.Log(logger.INFO, "OAuth2 provider registered", logger.Fields{"provider": string(providerType)})
	return nil
}

func (m *OAuth2ServiceManager) UnregisterProvider(providerType OAuth2ProviderType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.providers, providerType)
	m.logger.Log(logger.INFO, "OAuth2 provider unregistered", logger.Fields{"provider": string(providerType)})
}

func (m *OAuth2ServiceManager) GetProvider(providerType OAuth2ProviderType) (*OAuth2Service, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[providerType]
	if !exists {
		return nil, ErrOAuth2ProviderNotFound
	}
	return provider, nil
}

func (m *OAuth2ServiceManager) GenerateState(providerType OAuth2ProviderType) (string, *OAuth2State, error) {
	stateStr, err := generateRandomState()
	if err != nil {
		return "", nil, err
	}

	state := &OAuth2State{
		State:     stateStr,
		Provider:  string(providerType),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(m.stateTTL),
	}

	m.mu.Lock()
	m.states[stateStr] = state
	m.mu.Unlock()

	return stateStr, state, nil
}

func (m *OAuth2ServiceManager) ValidateState(stateStr string) (*OAuth2State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.states[stateStr]
	if !exists {
		return nil, ErrOAuth2InvalidState
	}

	if time.Now().After(state.ExpiresAt) {
		delete(m.states, stateStr)
		return nil, ErrOAuth2InvalidState
	}

	delete(m.states, stateStr)
	return state, nil
}

func (m *OAuth2ServiceManager) CleanupExpiredStates() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for stateStr, state := range m.states {
		if now.After(state.ExpiresAt) {
			delete(m.states, stateStr)
		}
	}
}

func (s *OAuth2Service) GetAuthorizationURL(state string) (string, error) {
	authURL, err := url.Parse(s.config.AuthURL)
	if err != nil {
		return "", err
	}

	params := authURL.Query()
	params.Set("client_id", s.config.ClientID)
	params.Set("redirect_uri", s.config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("scope", strings.Join(s.config.Scopes, " "))

	for key, value := range s.config.ExtraParams {
		params.Set(key, value)
	}

	if s.provider == ProviderGoogle {
		params.Set("access_type", "offline")
		params.Set("prompt", "consent")
	}

	authURL.RawQuery = params.Encode()
	return authURL.String(), nil
}

func (s *OAuth2Service) ExchangeCode(code string) (*OAuth2TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", s.config.RedirectURI)
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)

	resp, err := http.PostForm(s.config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuth2ExchangeFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOAuth2ExchangeFailed, resp.StatusCode, string(body))
	}

	var tokenResp OAuth2TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Log(logger.ERROR, "Failed to decode token response", logger.Fields{"error": err.Error(), "body": string(body)})
		return nil, fmt.Errorf("%w: failed to decode response", ErrOAuth2ExchangeFailed)
	}

	s.tokenMu.Lock()
	s.token = &tokenResp
	s.tokenMu.Unlock()

	s.logger.Log(logger.INFO, "OAuth2 token exchanged successfully", logger.Fields{"provider": string(s.provider)})
	return &tokenResp, nil
}

func (s *OAuth2Service) RefreshToken(refreshToken string) (*OAuth2TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)

	resp, err := http.PostForm(s.config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuth2RefreshFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOAuth2RefreshFailed, resp.StatusCode, string(body))
	}

	var tokenResp OAuth2TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response", ErrOAuth2RefreshFailed)
	}

	s.tokenMu.Lock()
	s.token = &tokenResp
	s.tokenMu.Unlock()

	s.logger.Log(logger.INFO, "OAuth2 token refreshed successfully", logger.Fields{"provider": string(s.provider)})
	return &tokenResp, nil
}

func (s *OAuth2Service) GetUserInfo(accessToken string) (*OAuth2UserInfo, error) {
	req, err := http.NewRequest("GET", s.config.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	if s.provider == ProviderGitHub {
		req.Header.Set("Accept", "application/vnd.github.v3+json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuth2UserInfoFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOAuth2UserInfoFailed, resp.StatusCode, string(body))
	}

	var userInfo *OAuth2UserInfo

	switch s.provider {
	case ProviderGitHub:
		var githubUser GitHubUser
		if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
			return nil, fmt.Errorf("%w: failed to decode GitHub user", ErrOAuth2UserInfoFailed)
		}

		emails, _ := s.getGitHubEmails(accessToken)

		userInfo = &OAuth2UserInfo{
			Provider:   string(ProviderGitHub),
			ProviderID: fmt.Sprintf("%d", githubUser.ID),
			Username:   githubUser.Login,
			Email:      githubUser.Email,
			Name:       githubUser.Name,
			AvatarURL:  githubUser.AvatarURL,
			RawData: map[string]interface{}{
				"github_id":  githubUser.ID,
				"login":      githubUser.Login,
				"emails":     emails,
			},
		}

	case ProviderGoogle:
		var googleUser GoogleUserInfo
		if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
			return nil, fmt.Errorf("%w: failed to decode Google user", ErrOAuth2UserInfoFailed)
		}

		userInfo = &OAuth2UserInfo{
			Provider:   string(ProviderGoogle),
			ProviderID: googleUser.ID,
			Username:   googleUser.Email,
			Email:      googleUser.Email,
			Name:       googleUser.Name,
			AvatarURL:  googleUser.Picture,
			RawData: map[string]interface{}{
				"google_id":        googleUser.ID,
				"verified_email":   googleUser.VerifiedEmail,
			},
		}
	}

	s.logger.Log(logger.INFO, "OAuth2 user info retrieved", logger.Fields{"provider": string(s.provider), "user": userInfo.Username})
	return userInfo, nil
}

func (s *OAuth2Service) getGitHubEmails(accessToken string) ([]string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to get emails: status %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	var result []string
	for _, email := range emails {
		if email.Primary && email.Verified {
			result = append(result, email.Email)
			break
		}
	}

	return result, nil
}

func (s *OAuth2Service) RevokeToken(token string) error {
	if s.config.RevokeURL == "" {
		return nil
	}

	data := url.Values{}
	data.Set("token", token)

	resp, err := http.PostForm(s.config.RevokeURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("revoke failed with status %d", resp.StatusCode)
	}

	s.tokenMu.Lock()
	s.token = nil
	s.tokenMu.Unlock()

	s.logger.Log(logger.INFO, "OAuth2 token revoked", logger.Fields{"provider": string(s.provider)})
	return nil
}

func (s *OAuth2Service) GetToken() *OAuth2TokenResponse {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()
	return s.token
}

func (s *OAuth2Service) IsTokenExpired() bool {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()

	if s.token == nil || s.token.ExpiresIn == 0 {
		return true
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

func (m *OAuth2ServiceManager) ListProviders() []OAuth2ProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]OAuth2ProviderType, 0, len(m.providers))
	for provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers
}

func (m *OAuth2ServiceManager) IsProviderEnabled(providerType OAuth2ProviderType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.providers[providerType]
	return exists
}
