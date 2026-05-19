package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
	"golang.org/x/oauth2"
)

type OAuth2Provider string

const (
	ProviderGitHub   OAuth2Provider = "github"
	ProviderGoogle   OAuth2Provider = "google"
	ProviderMicrosoft OAuth2Provider = "microsoft"
	ProviderFacebook OAuth2Provider = "facebook"
)

type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	Scopes      []string
}

type OAuth2State struct {
	State     string
	Provider  OAuth2Provider
	CreatedAt time.Time
	RedirectURL string
}

type OAuth2Client struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	Name         string `json:"name" binding:"required"`
	Provider     string `json:"provider" binding:"required"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret,omitempty"`
	RedirectURL  string `json:"redirect_url"`
	Scopes       string `json:"scopes"`
	IsEnabled    bool   `json:"is_enabled" gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OAuth2Token struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" binding:"required"`
	Provider     string    `json:"provider" binding:"required"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OAuth2ProviderManager struct {
	configs map[OAuth2Provider]OAuth2Config
	states  map[string]*OAuth2State
	mu      sync.RWMutex
}

var oauth2ProviderManager *OAuth2ProviderManager
var oauth2ProviderOnce sync.Once

func GetOAuth2ProviderManager() *OAuth2ProviderManager {
	oauth2ProviderOnce.Do(func() {
		oauth2ProviderManager = &OAuth2ProviderManager{
			configs: make(map[OAuth2Provider]OAuth2Config),
			states:  make(map[string]*OAuth2State),
		}
		oauth2ProviderManager.initializeDefaultProviders()
	})
	return oauth2ProviderManager
}

func (m *OAuth2ProviderManager) initializeDefaultProviders() {
	m.configs[ProviderGitHub] = OAuth2Config{
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
		Scopes:      []string{"user:email", "read:user"},
	}
	m.configs[ProviderGoogle] = OAuth2Config{
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:      []string{"openid", "email", "profile"},
	}
	m.configs[ProviderMicrosoft] = OAuth2Config{
		AuthURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:    "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		UserInfoURL: "https://graph.microsoft.com/v1.0/me",
		Scopes:      []string{"openid", "email", "profile"},
	}
	m.configs[ProviderFacebook] = OAuth2Config{
		AuthURL:     "https://www.facebook.com/v18.0/dialog/oauth",
		TokenURL:    "https://graph.facebook.com/v18.0/oauth/access_token",
		UserInfoURL: "https://graph.facebook.com/me",
		Scopes:      []string{"email", "public_profile"},
	}
}

func (m *OAuth2ProviderManager) RegisterProvider(provider OAuth2Provider, config OAuth2Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[provider] = config
}

func (m *OAuth2ProviderManager) GetConfig(provider OAuth2Provider) (OAuth2Config, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, ok := m.configs[provider]
	return config, ok
}

func (m *OAuth2ProviderManager) GenerateState(provider OAuth2Provider, redirectURL string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	
	m.states[state] = &OAuth2State{
		State:       state,
		Provider:    provider,
		CreatedAt:   time.Now(),
		RedirectURL: redirectURL,
	}
	
	go func() {
		time.Sleep(10 * time.Minute)
		m.mu.Lock()
		delete(m.states, state)
		m.mu.Unlock()
	}()
	
	return state
}

func (m *OAuth2ProviderManager) ValidateState(state string) (*OAuth2State, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stateData, ok := m.states[state]
	if !ok {
		return nil, false
	}
	
	if time.Since(stateData.CreatedAt) > 10*time.Minute {
		delete(m.states, state)
		return nil, false
	}
	
	delete(m.states, state)
	return stateData, true
}

func (m *OAuth2ProviderManager) ExchangeCode(provider OAuth2Provider, code, clientID, clientSecret, redirectURL string) (*oauth2.Token, error) {
	config, ok := m.GetConfig(provider)
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURL)
	data.Set("grant_type", "authorization_code")
	
	resp, err := http.PostForm(config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response failed: %w", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse token response failed: %w", err)
	}
	
	if errorMsg, ok := result["error"].(string); ok {
		return nil, fmt.Errorf("oauth2 error: %s - %v", errorMsg, result["error_description"])
	}
	
	token := &oauth2.Token{}
	token.AccessToken = result["access_token"].(string)
	token.TokenType = result["token_type"].(string)
	
	if refreshToken, ok := result["refresh_token"].(string); ok {
		token.RefreshToken = refreshToken
	}
	
	if expiresIn, ok := result["expires_in"].(float64); ok {
		token.Expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	
	return token, nil
}

func (m *OAuth2ProviderManager) GetUserInfo(provider OAuth2Provider, accessToken string) (map[string]interface{}, error) {
	config, ok := m.GetConfig(provider)
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	req, err := http.NewRequest("GET", config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create user info request failed: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user info failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read user info response failed: %w", err)
	}
	
	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("parse user info failed: %w", err)
	}
	
	return userInfo, nil
}

func GetOAuth2Clients() ([]OAuth2Client, error) {
	var clients []OAuth2Client
	err := database.DB.Where("is_enabled = ?", true).Find(&clients).Error
	return clients, err
}

func CreateOAuth2Client(client *OAuth2Client) error {
	if client.ClientSecret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		client.ClientSecret = hex.EncodeToString(b)
	}
	return database.DB.Create(client).Error
}

func UpdateOAuth2Client(client *OAuth2Client) error {
	return database.DB.Save(client).Error
}

func DeleteOAuth2Client(id uint) error {
	return database.DB.Delete(&OAuth2Client{}, id).Error
}

func GetOAuth2ClientByID(id uint) (*OAuth2Client, error) {
	var client OAuth2Client
	err := database.DB.First(&client, id).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func GetOAuth2ClientByProvider(provider string) (*OAuth2Client, error) {
	var client OAuth2Client
	err := database.DB.Where("provider = ? AND is_enabled = ?", provider, true).First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func SaveOAuth2Token(userID uint, provider string, token *oauth2.Token, scope string) error {
	oauth2Token := OAuth2Token{
		UserID:       userID,
		Provider:     provider,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresAt:    token.Expiry,
		Scope:        scope,
	}
	
	var existing OAuth2Token
	err := database.DB.Where("user_id = ? AND provider = ?", userID, provider).First(&existing).Error
	if err == nil {
		oauth2Token.ID = existing.ID
		oauth2Token.CreatedAt = existing.CreatedAt
		return database.DB.Save(&oauth2Token).Error
	}
	
	return database.DB.Create(&oauth2Token).Error
}

func GetOAuth2Token(userID uint, provider string) (*OAuth2Token, error) {
	var token OAuth2Token
	err := database.DB.Where("user_id = ? AND provider = ?", userID, provider).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func DeleteOAuth2Token(userID uint, provider string) error {
	return database.DB.Where("user_id = ? AND provider = ?", userID, provider).Delete(&OAuth2Token{}).Error
}

type OAuth2AuthorizeRequest struct {
	Provider string `form:"provider" binding:"required"`
	RedirectURI string `form:"redirect_uri"`
	Scope    string `form:"scope"`
	State    string `form:"state"`
}

type OAuth2CallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

type OAuth2UserInfo struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
}

func OAuth2Authorize(c *gin.Context) {
	var req OAuth2AuthorizeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	provider := OAuth2Provider(req.Provider)
	manager := GetOAuth2ProviderManager()
	
	client, err := GetOAuth2ClientByProvider(string(provider))
	if err != nil {
		response.Fail(c, response.CodeUnauthorized, "OAuth2 provider not configured")
		return
	}

	config, ok := manager.GetConfig(provider)
	if !ok {
		response.Fail(c, response.CodeInvalidParams, "unsupported OAuth2 provider")
		return
	}
	
	config.ClientID = client.ClientID
	config.RedirectURL = client.RedirectURL
	
	state := manager.GenerateState(provider, req.RedirectURI)
	
	authURL := buildAuthURL(config, state)
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "redirect to OAuth2 provider",
		"data": gin.H{
			"auth_url": authURL,
			"state":    state,
		},
	})
}

func OAuth2Callback(c *gin.Context) {
	var req OAuth2CallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	manager := GetOAuth2ProviderManager()
	stateData, valid := manager.ValidateState(req.State)
	if !valid {
		response.Fail(c, response.CodeUnauthorized, "invalid or expired state parameter")
		return
	}
	
	provider := stateData.Provider
	
	client, err := GetOAuth2ClientByProvider(string(provider))
	if err != nil {
		response.Fail(c, response.CodeUnauthorized, "OAuth2 provider not configured")
		return
	}
	
	token, err := manager.ExchangeCode(provider, req.Code, client.ClientID, client.ClientSecret, client.RedirectURL)
	if err != nil {
		response.Fail(c, response.CodeServerError, fmt.Sprintf("token exchange failed: %v", err))
		return
	}

	userInfo, err := manager.GetUserInfo(provider, token.AccessToken)
	if err != nil {
		response.Fail(c, response.CodeServerError, fmt.Sprintf("get user info failed: %v", err))
		return
	}
	
	oauth2UserInfo := extractUserInfo(provider, userInfo)
	
	user, err := findOrCreateOAuth2User(oauth2UserInfo)
	if err != nil {
		response.Fail(c, response.CodeServerError, fmt.Sprintf("user processing failed: %v", err))
		return
	}
	
	scopes := strings.Split(client.Scopes, ",")
	if err := SaveOAuth2Token(user.ID, string(provider), token, strings.Join(scopes, " ")); err != nil {
		fmt.Printf("Failed to save OAuth2 token: %v\n", err)
	}
	
	jwtToken, refreshToken, err := generateJWTTokens(user)
	if err != nil {
		response.Fail(c, response.CodeServerError, "failed to generate token")
		return
	}
	
	if stateData.RedirectURL != "" {
		redirectURL, _ := url.Parse(stateData.RedirectURL)
		q := redirectURL.Query()
		q.Set("token", jwtToken)
		if refreshToken != "" {
			q.Set("refresh_token", refreshToken)
		}
		q.Set("user_id", fmt.Sprintf("%d", user.ID))
		redirectURL.RawQuery = q.Encode()
		
		c.Redirect(http.StatusFound, redirectURL.String())
		return
	}
	
	c.JSON(200, gin.H{
		"code": 0,
		"message": "OAuth2 authentication successful",
		"data": gin.H{
			"token":         jwtToken,
			"refresh_token": refreshToken,
			"expires_in":    7200,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
			},
			"oauth2_user": oauth2UserInfo,
		},
	})
}

func OAuth2Revoke(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	var req struct {
		Provider string `json:"provider" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	if err := DeleteOAuth2Token(userID.(uint), req.Provider); err != nil {
		response.Fail(c, response.CodeServerError, "failed to revoke token")
		return
	}
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "OAuth2 token revoked successfully",
	})
}

func OAuth2UserInfoHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Fail(c, response.CodeUnauthorized, "user not authenticated")
		return
	}
	
	var req struct {
		Provider string `json:"provider" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	token, err := GetOAuth2Token(userID.(uint), req.Provider)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "OAuth2 token not found")
		return
	}
	
	manager := GetOAuth2ProviderManager()
	userInfo, err := manager.GetUserInfo(OAuth2Provider(req.Provider), token.AccessToken)
	if err != nil {
		response.Fail(c, response.CodeServerError, "failed to get user info")
		return
	}
	
	oauth2UserInfo := extractUserInfo(OAuth2Provider(req.Provider), userInfo)
	
	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"token": gin.H{
				"expires_at": token.ExpiresAt,
				"scope":      token.Scope,
				"token_type": token.TokenType,
			},
			"oauth2_user": oauth2UserInfo,
		},
	})
}

func ListOAuth2Clients(c *gin.Context) {
	clients, err := GetOAuth2Clients()
	if err != nil {
		response.Fail(c, response.CodeServerError, "failed to get OAuth2 clients")
		return
	}
	
	for i := range clients {
		clients[i].ClientSecret = ""
	}
	
	c.JSON(200, gin.H{
		"code": 0,
		"data": clients,
	})
}

func CreateOAuth2ClientHandler(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Provider    string   `json:"provider" binding:"required"`
		ClientID    string   `json:"client_id" binding:"required"`
		ClientSecret string  `json:"client_secret"`
		RedirectURL string   `json:"redirect_url" binding:"required"`
		Scopes      []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	client := &OAuth2Client{
		Name:         req.Name,
		Provider:     req.Provider,
		ClientID:    req.ClientID,
		ClientSecret: req.ClientSecret,
		RedirectURL:  req.RedirectURL,
		Scopes:       strings.Join(req.Scopes, ","),
		IsEnabled:    true,
	}
	
	if err := CreateOAuth2Client(client); err != nil {
		response.Fail(c, response.CodeServerError, "failed to create OAuth2 client")
		return
	}
	
	manager := GetOAuth2ProviderManager()
	config, ok := manager.GetConfig(OAuth2Provider(req.Provider))
	if ok {
		config.ClientID = req.ClientID
		if req.ClientSecret != "" {
			config.ClientSecret = req.ClientSecret
		}
		config.RedirectURL = req.RedirectURL
		if len(req.Scopes) > 0 {
			config.Scopes = req.Scopes
		}
		manager.RegisterProvider(OAuth2Provider(req.Provider), config)
	}
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "OAuth2 client created successfully",
		"data": gin.H{
			"id":            client.ID,
			"client_secret": client.ClientSecret,
		},
	})
}

func UpdateOAuth2ClientHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)
	
	client, err := GetOAuth2ClientByID(idNum)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "OAuth2 client not found")
		return
	}
	
	var req struct {
		Name        string   `json:"name"`
		ClientID    string   `json:"client_id"`
		ClientSecret string  `json:"client_secret"`
		RedirectURL string   `json:"redirect_url"`
		Scopes      []string `json:"scopes"`
		IsEnabled   *bool    `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	if req.Name != "" {
		client.Name = req.Name
	}
	if req.ClientID != "" {
		client.ClientID = req.ClientID
	}
	if req.ClientSecret != "" {
		client.ClientSecret = req.ClientSecret
	}
	if req.RedirectURL != "" {
		client.RedirectURL = req.RedirectURL
	}
	if len(req.Scopes) > 0 {
		client.Scopes = strings.Join(req.Scopes, ",")
	}
	if req.IsEnabled != nil {
		client.IsEnabled = *req.IsEnabled
	}
	
	if err := UpdateOAuth2Client(client); err != nil {
		response.Fail(c, response.CodeServerError, "failed to update OAuth2 client")
		return
	}
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "OAuth2 client updated successfully",
	})
}

func DeleteOAuth2ClientHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)
	
	if err := DeleteOAuth2Client(idNum); err != nil {
		response.Fail(c, response.CodeServerError, "failed to delete OAuth2 client")
		return
	}
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "OAuth2 client deleted successfully",
	})
}

func buildAuthURL(config OAuth2Config, state string) string {
	authURL, _ := url.Parse(config.AuthURL)
	q := authURL.Query()
	q.Set("client_id", config.ClientID)
	q.Set("redirect_uri", config.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(config.Scopes, " "))
	q.Set("state", state)
	authURL.RawQuery = q.Encode()
	return authURL.String()
}

func extractUserInfo(provider OAuth2Provider, userInfo map[string]interface{}) OAuth2UserInfo {
	info := OAuth2UserInfo{Provider: string(provider)}
	
	switch provider {
	case ProviderGitHub:
		if id, ok := userInfo["id"].(float64); ok {
			info.ID = fmt.Sprintf("%.0f", id)
		}
		if login, ok := userInfo["login"].(string); ok {
			info.Name = login
		}
		if name, ok := userInfo["name"].(string); ok && name != "" {
			info.Name = name
		}
		if email, ok := userInfo["email"].(string); ok {
			info.Email = email
		}
		if avatar, ok := userInfo["avatar_url"].(string); ok {
			info.Avatar = avatar
		}
		
	case ProviderGoogle:
		if id, ok := userInfo["id"].(string); ok {
			info.ID = id
		}
		if name, ok := userInfo["name"].(string); ok {
			info.Name = name
		}
		if email, ok := userInfo["email"].(string); ok {
			info.Email = email
		}
		if picture, ok := userInfo["picture"].(string); ok {
			info.Avatar = picture
		}
		
	case ProviderMicrosoft:
		if id, ok := userInfo["id"].(string); ok {
			info.ID = id
		}
		if name, ok := userInfo["displayName"].(string); ok {
			info.Name = name
		}
		if emails, ok := userInfo["userPrincipalName"].(string); ok {
			info.Email = emails
		}
		
	case ProviderFacebook:
		if id, ok := userInfo["id"].(string); ok {
			info.ID = id
		}
		if name, ok := userInfo["name"].(string); ok {
			info.Name = name
		}
		if email, ok := userInfo["email"].(string); ok {
			info.Email = email
		}
		if picture, ok := userInfo["picture"].(map[string]interface{}); ok {
			if data, ok := picture["data"].(map[string]interface{}); ok {
				if url, ok := data["url"].(string); ok {
					info.Avatar = url
				}
			}
		}
	}
	
	return info
}

func findOrCreateOAuth2User(oauth2Info OAuth2UserInfo) (*models.User, error) {
	email := oauth2Info.Email
	if email == "" {
		email = fmt.Sprintf("%s-%s@oauth.local", oauth2Info.Provider, oauth2Info.ID)
	}
	
	var user models.User
	err := database.DB.Where("email = ?", email).First(&user).Error
	if err == nil {
		return &user, nil
	}
	
	username := oauth2Info.Name
	if username == "" {
		username = fmt.Sprintf("%s_%s", oauth2Info.Provider, oauth2Info.ID)
	}
	
	uid := uuid.New().String()[:8]
	username = fmt.Sprintf("%s_%s", username, uid)
	
	hashedEmail := sha256.Sum256([]byte(email))
	usernameHash := fmt.Sprintf("%x", hashedEmail)[:12]
	username = fmt.Sprintf("oauth_%s", usernameHash)
	
	user = models.User{
		Username: username,
		Email:    email,
		Status:   "active",
	}
	
	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	
	return &user, nil
}

func generateJWTTokens(user *models.User) (string, string, error) {
	token, err := jwt.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", "", err
	}

	refreshToken := uuid.New().String()

	if redisClient := redis.GetClient(); redisClient != nil {
		key := fmt.Sprintf("refresh_token:%d", user.ID)
		redisClient.Set(context.Background(), key, refreshToken, 7*24*time.Hour)
	}

	return token, refreshToken, nil
}
