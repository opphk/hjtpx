package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOAuth2ProviderManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("InitializeDefaultProviders", func(t *testing.T) {
		manager := GetOAuth2ProviderManager()
		if manager == nil {
			t.Fatal("OAuth2ProviderManager should not be nil")
		}
		
		providers := []OAuth2Provider{ProviderGitHub, ProviderGoogle, ProviderMicrosoft, ProviderFacebook}
		for _, provider := range providers {
			config, ok := manager.GetConfig(provider)
			if !ok {
				t.Errorf("Provider %s should be registered", provider)
			}
			if config.AuthURL == "" {
				t.Errorf("Provider %s should have AuthURL", provider)
			}
			if config.TokenURL == "" {
				t.Errorf("Provider %s should have TokenURL", provider)
			}
		}
	})
	
	t.Run("GenerateAndValidateState", func(t *testing.T) {
		manager := GetOAuth2ProviderManager()
		
		state := manager.GenerateState(ProviderGitHub, "http://localhost/callback")
		if state == "" {
			t.Fatal("Generated state should not be empty")
		}
		
		stateData, valid := manager.ValidateState(state)
		if !valid {
			t.Fatal("State should be valid")
		}
		if stateData.Provider != ProviderGitHub {
			t.Errorf("State provider should be %s, got %s", ProviderGitHub, stateData.Provider)
		}
		if stateData.RedirectURL != "http://localhost/callback" {
			t.Errorf("State redirect URL should match")
		}
		
		_, valid = manager.ValidateState(state)
		if valid {
			t.Fatal("State should be invalidated after validation")
		}
	})
	
	t.Run("RegisterProvider", func(t *testing.T) {
		manager := GetOAuth2ProviderManager()
		
		newConfig := OAuth2Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			AuthURL:     "https://custom.auth.url/oauth/authorize",
			TokenURL:    "https://custom.auth.url/oauth/token",
			UserInfoURL: "https://custom.auth.url/userinfo",
			Scopes:      []string{"openid", "profile"},
		}
		
		customProvider := OAuth2Provider("custom")
		manager.RegisterProvider(customProvider, newConfig)
		
		config, ok := manager.GetConfig(customProvider)
		if !ok {
			t.Fatal("Custom provider should be registered")
		}
		if config.ClientID != "test-client-id" {
			t.Errorf("ClientID should match")
		}
	})
	
	t.Run("ValidateInvalidState", func(t *testing.T) {
		manager := GetOAuth2ProviderManager()
		
		_, valid := manager.ValidateState("invalid-state")
		if valid {
			t.Fatal("Invalid state should not be valid")
		}
	})
}

func TestOAuth2Handlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("OAuth2Authorize", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/api/v1/oauth2/authorize?provider=github", nil)
		c.Request = req
		
		OAuth2Authorize(c)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	t.Run("OAuth2AuthorizeMissingProvider", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/api/v1/oauth2/authorize", nil)
		c.Request = req
		
		OAuth2Authorize(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("OAuth2CallbackMissingParams", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/api/v1/oauth2/callback", nil)
		c.Request = req
		
		OAuth2Callback(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("ListOAuth2Clients", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/admin/api/oauth2/clients", nil)
		c.Request = req
		
		ListOAuth2Clients(c)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	t.Run("CreateOAuth2ClientMissingFields", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("POST", "/admin/api/oauth2/clients", strings.NewReader(`{"name": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		CreateOAuth2ClientHandler(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("OAuth2RevokeMissingProvider", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		c.Set("user_id", uint(1))
		
		req, _ := http.NewRequest("POST", "/api/v1/oauth2/revoke", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		OAuth2Revoke(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("OAuth2UserInfoNotAuthenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/api/v1/oauth2/userinfo", nil)
		c.Request = req
		
		OAuth2UserInfo(c)
		
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestExtractUserInfo(t *testing.T) {
	t.Run("GitHub", func(t *testing.T) {
		userInfo := map[string]interface{}{
			"id":         float64(12345),
			"login":      "testuser",
			"name":       "Test User",
			"email":      "test@example.com",
			"avatar_url": "https://example.com/avatar.png",
		}
		
		result := extractUserInfo(ProviderGitHub, userInfo)
		
		if result.Provider != "github" {
			t.Errorf("Provider should be github")
		}
		if result.ID != "12345" {
			t.Errorf("ID should be 12345, got %s", result.ID)
		}
		if result.Name != "Test User" {
			t.Errorf("Name should be Test User")
		}
		if result.Email != "test@example.com" {
			t.Errorf("Email should be test@example.com")
		}
		if result.Avatar != "https://example.com/avatar.png" {
			t.Errorf("Avatar should match")
		}
	})
	
	t.Run("Google", func(t *testing.T) {
		userInfo := map[string]interface{}{
			"id":      "google-12345",
			"name":   "Google User",
			"email":   "google@example.com",
			"picture": "https://example.com/google-avatar.png",
		}
		
		result := extractUserInfo(ProviderGoogle, userInfo)
		
		if result.Provider != "google" {
			t.Errorf("Provider should be google")
		}
		if result.ID != "google-12345" {
			t.Errorf("ID should be google-12345")
		}
	})
	
	t.Run("Microsoft", func(t *testing.T) {
		userInfo := map[string]interface{}{
			"id":                "microsoft-67890",
			"displayName":       "Microsoft User",
			"userPrincipalName": "user@microsoft.com",
		}
		
		result := extractUserInfo(ProviderMicrosoft, userInfo)
		
		if result.Provider != "microsoft" {
			t.Errorf("Provider should be microsoft")
		}
		if result.ID != "microsoft-67890" {
			t.Errorf("ID should be microsoft-67890")
		}
		if result.Email != "user@microsoft.com" {
			t.Errorf("Email should be user@microsoft.com")
		}
	})
	
	t.Run("Facebook", func(t *testing.T) {
		userInfo := map[string]interface{}{
			"id":   "fb-11111",
			"name": "Facebook User",
			"email": "fb@example.com",
			"picture": map[string]interface{}{
				"data": map[string]interface{}{
					"url": "https://example.com/fb-avatar.png",
				},
			},
		}
		
		result := extractUserInfo(ProviderFacebook, userInfo)
		
		if result.Provider != "facebook" {
			t.Errorf("Provider should be facebook")
		}
		if result.ID != "fb-11111" {
			t.Errorf("ID should be fb-11111")
		}
		if result.Avatar != "https://example.com/fb-avatar.png" {
			t.Errorf("Avatar should match")
		}
	})
	
	t.Run("GitHubWithoutName", func(t *testing.T) {
		userInfo := map[string]interface{}{
			"id":         float64(12345),
			"login":      "testuser",
			"avatar_url": "https://example.com/avatar.png",
		}
		
		result := extractUserInfo(ProviderGitHub, userInfo)
		
		if result.Name != "testuser" {
			t.Errorf("Name should fall back to login")
		}
	})
	
	t.Run("GitHubWithoutEmail", func(t *testing.T) {
		userInfo := map[string]interface{}{
			"id":         float64(12345),
			"login":      "testuser",
			"avatar_url": "https://example.com/avatar.png",
		}
		
		result := extractUserInfo(ProviderGitHub, userInfo)
		
		if result.Email != "" {
			t.Errorf("Email should be empty when not provided")
		}
	})
}

func TestBuildAuthURL(t *testing.T) {
	config := OAuth2Config{
		ClientID:    "test-client-id",
		RedirectURL: "http://localhost/callback",
		AuthURL:     "https://github.com/login/oauth/authorize",
		Scopes:      []string{"user:email", "read:user"},
	}
	
	state := "test-state"
	
	authURL := buildAuthURL(config, state)
	
	if !strings.Contains(authURL, "client_id=test-client-id") {
		t.Errorf("AuthURL should contain client_id")
	}
	if !strings.Contains(authURL, "redirect_uri=") {
		t.Errorf("AuthURL should contain redirect_uri")
	}
	if !strings.Contains(authURL, "state=test-state") {
		t.Errorf("AuthURL should contain state")
	}
	if !strings.Contains(authURL, "scope=") {
		t.Errorf("AuthURL should contain scope")
	}
}
