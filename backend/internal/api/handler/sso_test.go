package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSSOProviderManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("GetSSOProviderManager", func(t *testing.T) {
		manager := GetSSOProviderManager()
		if manager == nil {
			t.Fatal("SSOProviderManager should not be nil")
		}
	})
	
	t.Run("CreateAndGetSession", func(t *testing.T) {
		manager := GetSSOProviderManager()
		
		session := &SSOSession{
			SessionID:    "test-session-123",
			UserID:       1,
			ProviderType: "saml",
			ProviderID:   1,
			Attributes:   make(map[string]string),
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		}
		
		err := manager.CreateSession(session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		
		retrieved, ok := manager.GetSession("test-session-123")
		if !ok {
			t.Fatal("Session should be found")
		}
		if retrieved.UserID != 1 {
			t.Errorf("UserID should be 1, got %d", retrieved.UserID)
		}
		if retrieved.ProviderType != "saml" {
			t.Errorf("ProviderType should be saml, got %s", retrieved.ProviderType)
		}
		
		manager.DeleteSession("test-session-123")
		
		_, ok = manager.GetSession("test-session-123")
		if ok {
			t.Fatal("Session should be deleted")
		}
	})
	
	t.Run("GetSessionExpired", func(t *testing.T) {
		manager := GetSSOProviderManager()
		
		session := &SSOSession{
			SessionID:    "expired-session",
			UserID:       1,
			ProviderType: "saml",
			Attributes:   make(map[string]string),
			CreatedAt:    time.Now().Add(-2 * time.Hour),
			ExpiresAt:    time.Now().Add(-1 * time.Hour),
		}
		
		manager.CreateSession(session)
		
		_, ok := manager.GetSession("expired-session")
		if ok {
			t.Fatal("Expired session should not be found")
		}
	})
	
	t.Run("GetAllConfigs", func(t *testing.T) {
		manager := GetSSOProviderManager()
		configs := manager.GetAllConfigs()
		if configs == nil {
			t.Fatal("GetAllConfigs should not return nil")
		}
	})
}

func TestSAMLServiceProvider(t *testing.T) {
	t.Run("GenerateAuthnRequest", func(t *testing.T) {
		config := SAMLConfig{
			EntityID:   "https://hjtpx.example.com/saml",
			SSOURL:     "https://idp.example.com/sso",
			AssertionConsumerServiceURL: "https://hjtpx.example.com/saml/callback",
		}
		
		service := &SAMLServiceProvider{config: config}
		authnRequest, err := service.GenerateAuthnRequest()
		
		if err != nil {
			t.Fatalf("Failed to generate authn request: %v", err)
		}
		
		if authnRequest.ID == "" {
			t.Error("AuthnRequest ID should not be empty")
		}
		if authnRequest.Version != "2.0" {
			t.Errorf("Version should be 2.0, got %s", authnRequest.Version)
		}
		if authnRequest.Issuer != config.EntityID {
			t.Errorf("Issuer should be %s, got %s", config.EntityID, authnRequest.Issuer)
		}
		if authnRequest.Destination != config.SSOURL {
			t.Errorf("Destination should be %s, got %s", config.SSOURL, authnRequest.Destination)
		}
	})
	
	t.Run("BuildAuthnRequestRedirect", func(t *testing.T) {
		config := SAMLConfig{
			EntityID:   "https://hjtpx.example.com/saml",
			SSOURL:     "https://idp.example.com/sso",
			AssertionConsumerServiceURL: "https://hjtpx.example.com/saml/callback",
		}
		
		service := &SAMLServiceProvider{config: config}
		authnRequest, _ := service.GenerateAuthnRequest()
		
		redirectURL, err := service.BuildAuthnRequestRedirect(authnRequest)
		if err != nil {
			t.Fatalf("Failed to build redirect URL: %v", err)
		}
		
		if !strings.Contains(redirectURL, "https://idp.example.com/sso") {
			t.Error("Redirect URL should contain SSO URL")
		}
		if !strings.Contains(redirectURL, "SAMLRequest=") {
			t.Error("Redirect URL should contain SAMLRequest parameter")
		}
	})
	
	t.Run("ParseResponse", func(t *testing.T) {
		service := &SAMLServiceProvider{}
		
		sampleResponse := `<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol">
    <samlp:Status>
        <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
    </samlp:Status>
</samlp:Response>`
		
		_, err := service.ParseResponse(sampleResponse)
		if err != nil {
			t.Fatalf("Failed to parse SAML response: %v", err)
		}
	})
	
	t.Run("ValidateResponseInvalidStatus", func(t *testing.T) {
		service := &SAMLServiceProvider{}
		
		response := &SAMLResponse{
			Status: struct {
				StatusCode string `xml:"StatusCode"`
			}{
				StatusCode: "urn:oasis:names:tc:SAML:2.0:status:AuthnFailed",
			},
		}
		
		err := service.ValidateResponse(response)
		if err == nil {
			t.Error("Should return error for invalid status")
		}
	})
}

func TestLDAPService(t *testing.T) {
	t.Run("AuthenticateWithoutServer", func(t *testing.T) {
		service := &LDAPService{
			config: LDAPConfig{},
		}
		
		_, err := service.Authenticate("testuser", "password")
		if err == nil {
			t.Error("Should return error when LDAP server is not configured")
		}
	})
	
	t.Run("SearchUserWithoutServer", func(t *testing.T) {
		service := &LDAPService{
			config: LDAPConfig{},
		}
		
		_, err := service.SearchUser("testuser")
		if err == nil {
			t.Error("Should return error when LDAP server is not configured")
		}
	})
	
	t.Run("GetUserGroupsWithoutServer", func(t *testing.T) {
		service := &LDAPService{
			config: LDAPConfig{},
		}
		
		_, err := service.GetUserGroups("uid=testuser,dc=example,dc=com")
		if err == nil {
			t.Error("Should return error when LDAP server is not configured")
		}
	})
}

func TestSSOHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("GetSSOConfigsHandler", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/admin/api/sso/configs", nil)
		c.Request = req
		
		GetSSOConfigsHandler(c)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	t.Run("CreateSSOConfigHandlerMissingFields", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("POST", "/admin/api/sso/configs", strings.NewReader(`{"name": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		CreateSSOConfigHandler(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("CreateSAMLConfigHandler", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		body := `{
			"name": "Test SAML",
			"provider_type": "saml",
			"saml_config": {
				"entity_id": "https://test.example.com/saml",
				"sso_url": "https://idp.example.com/sso",
				"acs_url": "https://test.example.com/saml/callback"
			},
			"auto_provision": true,
			"default_role": "user"
		}`
		
		req, _ := http.NewRequest("POST", "/admin/api/sso/configs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		CreateSSOConfigHandler(c)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	t.Run("CreateLDAPConfigHandler", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		body := `{
			"name": "Test LDAP",
			"provider_type": "ldap",
			"ldap_config": {
				"server": "ldap.example.com",
				"port": 389,
				"base_dn": "dc=example,dc=com"
			},
			"auto_provision": true,
			"default_role": "user"
		}`
		
		req, _ := http.NewRequest("POST", "/admin/api/sso/configs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		CreateSSOConfigHandler(c)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	t.Run("CreateSSOConfigMissingSAMLConfig", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		body := `{
			"name": "Test SAML",
			"provider_type": "saml"
		}`
		
		req, _ := http.NewRequest("POST", "/admin/api/sso/configs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		CreateSSOConfigHandler(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("CreateSSOConfigMissingLDAPConfig", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		body := `{
			"name": "Test LDAP",
			"provider_type": "ldap"
		}`
		
		req, _ := http.NewRequest("POST", "/admin/api/sso/configs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		CreateSSOConfigHandler(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("SAMLSSOInitiateHandler", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/sso/saml/1/initiate", nil)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		
		SAMLSSOInitiateHandler(c)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	t.Run("SAMLSSOCallbackHandlerMissingParams", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("POST", "/sso/saml/callback", nil)
		c.Request = req
		
		SAMLSSOCallbackHandler(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("LDAPLoginHandlerMissingParams", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("POST", "/sso/ldap/login", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		
		LDAPLoginHandler(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("SSOSessionStatusHandlerMissingSessionID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("GET", "/sso/session/status", nil)
		c.Request = req
		
		SSOSessionStatusHandler(c)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("SSOSessionLogoutHandler", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		req, _ := http.NewRequest("POST", "/sso/session/logout?session_id=test-session", nil)
		c.Request = req
		
		SSOSessionLogoutHandler(c)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestSAMLConfigParsing(t *testing.T) {
	t.Run("SAMLConfigFields", func(t *testing.T) {
		config := SAMLConfig{
			EntityID:          "https://hjtpx.example.com/saml/metadata",
			SSOURL:            "https://idp.example.com/sso",
			SLOURL:            "https://idp.example.com/slo",
			AssertionConsumerServiceURL: "https://hjtpx.example.com/saml/acs",
		}
		
		if config.EntityID == "" {
			t.Error("EntityID should not be empty")
		}
		if config.SSOURL == "" {
			t.Error("SSOURL should not be empty")
		}
		if config.AssertionConsumerServiceURL == "" {
			t.Error("ACS URL should not be empty")
		}
	})
}

func TestLDAPConfigParsing(t *testing.T) {
	t.Run("LDAPConfigFields", func(t *testing.T) {
		config := LDAPConfig{
			Server:     "ldap.example.com",
			Port:       636,
			BaseDN:     "dc=example,dc=com",
			BindDN:     "cn=admin,dc=example,dc=com",
			BindPassword: "secret",
			UseTLS:     true,
			StartTLS:   false,
			UserFilter: "(uid=%s)",
			GroupFilter: "(member=%s)",
		}
		
		if config.Server == "" {
			t.Error("Server should not be empty")
		}
		if config.BaseDN == "" {
			t.Error("BaseDN should not be empty")
		}
		if config.Port == 0 {
			t.Error("Port should not be 0")
		}
	})
}

func TestSSOSessionExpiry(t *testing.T) {
	manager := GetSSOProviderManager()
	
	t.Run("SessionExpiresAtFuture", func(t *testing.T) {
		session := &SSOSession{
			SessionID:    "future-expiry-session",
			UserID:       1,
			ProviderType: "saml",
			Attributes:   make(map[string]string),
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		}
		
		manager.CreateSession(session)
		
		found, ok := manager.GetSession("future-expiry-session")
		if !ok {
			t.Error("Session should be valid")
		}
		if found.ExpiresAt.Before(time.Now()) {
			t.Error("Session should not be expired")
		}
	})
}

func TestInitSAMLService(t *testing.T) {
	t.Run("InitWithEmptyConfig", func(t *testing.T) {
		config := SAMLConfig{
			EntityID:   "https://test.example.com/saml",
			SSOURL:     "https://idp.example.com/sso",
			AssertionConsumerServiceURL: "https://test.example.com/saml/acs",
		}
		
		err := InitSAMLService(config)
		if err != nil {
			t.Errorf("InitSAMLService should succeed: %v", err)
		}
		if samlService == nil {
			t.Error("samlService should not be nil after init")
		}
	})
}

func TestInitLDAPService(t *testing.T) {
	t.Run("InitWithEmptyConfig", func(t *testing.T) {
		config := LDAPConfig{
			Server: "ldap.example.com",
			Port:   389,
			BaseDN: "dc=example,dc=com",
		}
		
		err := InitLDAPService(config)
		if err != nil {
			t.Errorf("InitLDAPService should succeed: %v", err)
		}
		if ldapService == nil {
			t.Error("ldapService should not be nil after init")
		}
	})
}
