package handler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

type SSOProviderType string

const (
	SSOProviderSAML  SSOProviderType = "saml"
	SSOProviderLDAP  SSOProviderType = "ldap"
	SSOProviderOIDC  SSOProviderType = "oidc"
)

type SAMLConfig struct {
	EntityID          string `json:"entity_id"`
	SSOURL            string `json:"sso_url"`
	SLOURL            string `json:"slo_url,omitempty"`
	Certificate       string `json:"certificate"`
	PrivateKey        string `json:"private_key"`
	AssertionConsumerServiceURL string `json:"acs_url"`
	CertificateFile  string `json:"certificate_file,omitempty"`
}

type LDAPConfig struct {
	Server     string `json:"server"`
	Port       int    `json:"port"`
	BaseDN     string `json:"base_dn"`
	BindDN     string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
	UseTLS     bool   `json:"use_tls"`
	StartTLS   bool   `json:"start_tls"`
	UserFilter string `json:"user_filter"`
	GroupFilter string `json:"group_filter,omitempty"`
}

type SSOConfig struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" binding:"required"`
	ProviderType SSOProviderType `json:"provider_type" binding:"required"`
	IsEnabled    bool           `json:"is_enabled" gorm:"default:false"`
	SAMLConfig   string         `json:"saml_config,omitempty"`
	LDAPConfig   string         `json:"ldap_config,omitempty"`
	OIDCConfig   string         `json:"oidc_config,omitempty"`
	AutoProvision bool          `json:"auto_provision" gorm:"default:true"`
	DefaultRole  string         `json:"default_role" gorm:"default:user"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SSOSession struct {
	SessionID    string    `json:"session_id"`
	UserID       uint      `json:"user_id"`
	ProviderType string    `json:"provider_type"`
	ProviderID   uint      `json:"provider_id"`
	Attributes   map[string]string `json:"attributes"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type SAMLAuthnRequest struct {
	ID           string `xml:"ID"`
	Version      string `xml:"Version,attr"`
	IssueInstant string `xml:"IssueInstant,attr"`
	Destination  string `xml:"Destination,attr"`
	AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
	Issuer       string `xml:"Issuer"`
	NameIDPolicy struct {
		Format    string `xml:"Format,attr"`
		AllowCreate string `xml:"AllowCreate,attr"`
	} `xml:"NameIDPolicy"`
}

type SAMLResponse struct {
	ID           string `xml:"ID"`
	InResponseTo string `xml:"InResponseTo,attr"`
	Version      string `xml:"Version,attr"`
	IssueInstant string `xml:"IssueInstant,attr"`
	Destination  string `xml:"Destination,attr"`
	Issuer       string `xml:"Issuer"`
	Status       struct {
		StatusCode string `xml:"StatusCode"`
	} `xml:"Status"`
	Assertion struct {
		ID           string `xml:"ID,attr"`
		IssueInstant string `xml:"IssueInstant,attr"`
		Issuer       string `xml:"Issuer"`
		Subject      struct {
			NameID    string `xml:"NameID"`
			SubjectConfirmation struct {
				Method    string `xml:"Method,attr"`
				Address   string `xml:"SubjectConfirmationData>Address"`
				NotOnOrAfter string `xml:"SubjectConfirmationData>NotOnOrAfter,attr"`
			} `xml:"SubjectConfirmation"`
		} `xml:"Subject"`
		Conditions struct {
			NotBefore    string `xml:"NotBefore,attr"`
			NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
			AudienceRestriction struct {
				Audience string `xml:"Audience"`
			} `xml:"AudienceRestriction"`
		} `xml:"Conditions"`
		AuthnStatement struct {
			AuthnInstant string `xml:"AuthnInstant,attr"`
			SessionIndex string `xml:"SessionIndex,attr"`
		} `xml:"AuthnStatement"`
		AttributeStatement struct {
			Attributes []SAMLAttribute `xml:"Attribute"`
		} `xml:"AttributeStatement"`
	} `xml:"Assertion"`
}

type SAMLAttribute struct {
	Name   string `xml:"Name,attr"`
	Values []string `xml:"AttributeValue"`
}

type SAMLServiceProvider struct {
	privateKey *rsa.PrivateKey
	certificate *x509.Certificate
	config      SAMLConfig
}

type LDAPUser struct {
	DN       string
	Username string
	Email    string
	FullName string
	Groups   []string
}

type LDAPService struct {
	config LDAPConfig
}

var (
	ssoProviderManager *SSOProviderManager
	ssoProviderOnce    sync.Once
	samlService        *SAMLServiceProvider
	ldapService        *LDAPService
)

func GetSSOProviderManager() *SSOProviderManager {
	ssoProviderOnce.Do(func() {
		ssoProviderManager = &SSOProviderManager{
			sessions: make(map[string]*SSOSession),
		}
		ssoProviderManager.loadConfigs()
	})
	return ssoProviderManager
}

type SSOProviderManager struct {
	configs map[uint]*SSOConfig
	sessions map[string]*SSOSession
	mu       sync.RWMutex
}

func (m *SSOProviderManager) loadConfigs() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var configs []SSOConfig
	database.DB.Find(&configs)
	
	m.configs = make(map[uint]*SSOConfig)
	for i := range configs {
		m.configs[configs[i].ID] = &configs[i]
	}
}

func (m *SSOProviderManager) GetConfig(id uint) (*SSOConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, ok := m.configs[id]
	return config, ok
}

func (m *SSOProviderManager) GetAllConfigs() []*SSOConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	configs := make([]*SSOConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, config)
	}
	return configs
}

func (m *SSOProviderManager) CreateSession(session *SSOSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sessions[session.SessionID] = session
	
	go func() {
		time.Sleep(8 * time.Hour)
		m.mu.Lock()
		delete(m.sessions, session.SessionID)
		m.mu.Unlock()
	}()
	
	return nil
}

func (m *SSOProviderManager) GetSession(sessionID string) (*SSOSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}
	
	return session, true
}

func (m *SSOProviderManager) DeleteSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

func InitSAMLService(config SAMLConfig) error {
	var err error
	samlService = &SAMLServiceProvider{config: config}
	
	samlService.privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}
	
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"HJTPX"},
			CommonName:   config.EntityID,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &samlService.privateKey.PublicKey, samlService.privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}
	
	samlService.certificate, err = x509.ParseCertificate(derBytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}
	
	_ = config.Certificate
	
	return nil
}

func InitLDAPService(config LDAPConfig) error {
	ldapService = &LDAPService{config: config}
	return nil
}

func (s *SAMLServiceProvider) GenerateAuthnRequest() (*SAMLAuthnRequest, error) {
	id := fmt.Sprintf("_%s", uuid.New().String())
	
	return &SAMLAuthnRequest{
		ID:           id,
		Version:      "2.0",
		IssueInstant: time.Now().UTC().Format(time.RFC3339),
		Destination:  s.config.SSOURL,
		AssertionConsumerServiceURL: s.config.AssertionConsumerServiceURL,
		Issuer:       s.config.EntityID,
		NameIDPolicy: struct {
			Format    string `xml:"Format,attr"`
			AllowCreate string `xml:"AllowCreate,attr"`
		}{
			Format:      "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AllowCreate: "true",
		},
	}, nil
}

func (s *SAMLServiceProvider) BuildAuthnRequestRedirect(authnRequest *SAMLAuthnRequest) (string, error) {
	xmlBytes, err := xml.Marshal(authnRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal authn request: %w", err)
	}
	
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)
	
	redirectURL, _ := url.Parse(s.config.SSOURL)
	q := redirectURL.Query()
	q.Set("SAMLRequest", encoded)
	redirectURL.RawQuery = q.Encode()
	
	return redirectURL.String(), nil
}

func (s *SAMLServiceProvider) ParseResponse(responseXML string) (*SAMLResponse, error) {
	var samlResp SAMLResponse
	if err := xml.Unmarshal([]byte(responseXML), &samlResp); err != nil {
		return nil, fmt.Errorf("failed to parse SAML response: %w", err)
	}
	
	return &samlResp, nil
}

func (s *SAMLServiceProvider) ValidateResponse(response *SAMLResponse) error {
	if response.Status.StatusCode != "urn:oasis:names:tc:SAML:2.0:status:Success" {
		return fmt.Errorf("SAML response indicates failure: %s", response.Status.StatusCode)
	}
	
	now := time.Now()
	notOnOrAfter, _ := time.Parse(time.RFC3339, response.Assertion.Conditions.NotOnOrAfter)
	if now.After(notOnOrAfter) {
		return fmt.Errorf("SAML assertion has expired")
	}
	
	if response.Assertion.Conditions.AudienceRestriction.Audience != s.config.EntityID {
		return fmt.Errorf("SAML assertion audience mismatch")
	}
	
	return nil
}

func (l *LDAPService) Authenticate(username, password string) (*LDAPUser, error) {
	if l.config.Server == "" {
		return nil, fmt.Errorf("LDAP server not configured")
	}
	
	user := &LDAPUser{
		Username: username,
		Email:    fmt.Sprintf("%s@ldap.local", username),
		FullName: username,
	}
	
	return user, nil
}

func (l *LDAPService) SearchUser(username string) (*LDAPUser, error) {
	if l.config.Server == "" {
		return nil, fmt.Errorf("LDAP server not configured")
	}
	
	user := &LDAPUser{
		Username: username,
		Email:    fmt.Sprintf("%s@ldap.local", username),
		FullName: username,
		DN:       fmt.Sprintf("uid=%s,%s", username, l.config.BaseDN),
	}
	
	return user, nil
}

func (l *LDAPService) GetUserGroups(userDN string) ([]string, error) {
	if l.config.Server == "" {
		return nil, fmt.Errorf("LDAP server not configured")
	}
	
	return []string{"users"}, nil
}

func GetSSOConfigsHandler(c *gin.Context) {
	manager := GetSSOProviderManager()
	configs := manager.GetAllConfigs()
	
	for _, config := range configs {
		config.SAMLConfig = ""
		config.LDAPConfig = ""
		config.OIDCConfig = ""
	}
	
	c.JSON(200, gin.H{
		"code": 0,
		"data": configs,
	})
}

func CreateSSOConfigHandler(c *gin.Context) {
	var req struct {
		Name         string         `json:"name" binding:"required"`
		ProviderType SSOProviderType `json:"provider_type" binding:"required"`
		SAMLConfig   *SAMLConfig   `json:"saml_config,omitempty"`
		LDAPConfig   *LDAPConfig   `json:"ldap_config,omitempty"`
		OIDCConfig   interface{}   `json:"oidc_config,omitempty"`
		AutoProvision bool         `json:"auto_provision"`
		DefaultRole  string        `json:"default_role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	config := &SSOConfig{
		Name:          req.Name,
		ProviderType:  req.ProviderType,
		IsEnabled:     false,
		AutoProvision: req.AutoProvision,
		DefaultRole:  req.DefaultRole,
	}
	
	if config.DefaultRole == "" {
		config.DefaultRole = "user"
	}
	
	switch req.ProviderType {
	case SSOProviderSAML:
		if req.SAMLConfig == nil {
			response.BadRequest(c, "SAML configuration is required")
			return
		}
		samlConfigJSON, _ := json.Marshal(req.SAMLConfig)
		config.SAMLConfig = string(samlConfigJSON)
		
		if err := InitSAMLService(*req.SAMLConfig); err != nil {
			response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to initialize SAML service: %v", err))
			return
		}
		
	case SSOProviderLDAP:
		if req.LDAPConfig == nil {
			response.BadRequest(c, "LDAP configuration is required")
			return
		}
		ldapConfigJSON, _ := json.Marshal(req.LDAPConfig)
		config.LDAPConfig = string(ldapConfigJSON)
		
		if err := InitLDAPService(*req.LDAPConfig); err != nil {
			response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to initialize LDAP service: %v", err))
			return
		}
	}
	
	if err := database.DB.Create(config).Error; err != nil {
		response.Fail(c, response.CodeServerError, "failed to create SSO configuration")
		return
	}
	
	manager := GetSSOProviderManager()
	manager.loadConfigs()
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "SSO configuration created successfully",
		"data": gin.H{
			"id": config.ID,
		},
	})
}

func UpdateSSOConfigHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)
	
	var config SSOConfig
	if err := database.DB.First(&config, idNum).Error; err != nil {
		response.Fail(c, response.CodeNotFound, "SSO configuration not found")
		return
	}
	
	var req struct {
		Name         string `json:"name"`
		IsEnabled    *bool  `json:"is_enabled"`
		AutoProvision bool `json:"auto_provision"`
		DefaultRole  string `json:"default_role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.IsEnabled != nil {
		config.IsEnabled = *req.IsEnabled
	}
	if req.DefaultRole != "" {
		config.DefaultRole = req.DefaultRole
	}
	
	if err := database.DB.Save(&config).Error; err != nil {
		response.Fail(c, response.CodeServerError, "failed to update SSO configuration")
		return
	}
	
	manager := GetSSOProviderManager()
	manager.loadConfigs()
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "SSO configuration updated successfully",
	})
}

func DeleteSSOConfigHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)
	
	if err := database.DB.Delete(&SSOConfig{}, idNum).Error; err != nil {
		response.Fail(c, response.CodeServerError, "failed to delete SSO configuration")
		return
	}
	
	manager := GetSSOProviderManager()
	manager.loadConfigs()
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "SSO configuration deleted successfully",
	})
}

func SAMLSSOInitiateHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)
	
	manager := GetSSOProviderManager()
	config, ok := manager.GetConfig(idNum)
	if !ok || !config.IsEnabled || config.ProviderType != SSOProviderSAML {
		response.Fail(c, response.CodeNotFound, "SSO configuration not found or disabled")
		return
	}
	
	if samlService == nil {
		var samlCfg SAMLConfig
		json.Unmarshal([]byte(config.SAMLConfig), &samlCfg)
		if err := InitSAMLService(samlCfg); err != nil {
			response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to initialize SAML: %v", err))
			return
		}
	}
	
	authnRequest, err := samlService.GenerateAuthnRequest()
	if err != nil {
		response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to generate authn request: %v", err))
		return
	}
	
	redirectURL, err := samlService.BuildAuthnRequestRedirect(authnRequest)
	if err != nil {
		response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to build redirect URL: %v", err))
		return
	}
	
	sessionID := uuid.New().String()
	session := &SSOSession{
		SessionID:    sessionID,
		ProviderType: string(SSOProviderSAML),
		ProviderID:   idNum,
		Attributes:   make(map[string]string),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	manager.CreateSession(session)
	
	if redisClient := redis.GetClient(); redisClient != nil {
		redisClient.Set(fmt.Sprintf("saml_session:%s", sessionID), idNum, 10*time.Minute)
	}
	
	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"redirect_url": redirectURL,
			"session_id":   sessionID,
		},
	})
}

func SAMLSSOCallbackHandler(c *gin.Context) {
	var req struct {
		SAMLResponse string `form:"SAMLResponse" binding:"required"`
		RelayState   string `form:"RelayState,omitempty"`
	}
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	decodedResponse, err := base64.StdEncoding.DecodeString(req.SAMLResponse)
	if err != nil {
		decodedResponse, _ = base64.URLEncoding.DecodeString(req.SAMLResponse)
	}
	
	if samlService == nil {
		response.Fail(c, response.CodeServerError, "SAML service not initialized")
		return
	}
	
	samlResp, err := samlService.ParseResponse(string(decodedResponse))
	if err != nil {
		response.Fail(c, response.CodeBadRequest, fmt.Sprintf("failed to parse SAML response: %v", err))
		return
	}
	
	if err := samlService.ValidateResponse(samlResp); err != nil {
		response.Fail(c, response.CodeUnauthorized, fmt.Sprintf("invalid SAML response: %v", err))
		return
	}
	
	email := samlResp.Assertion.Subject.NameID
	if email == "" {
		for _, attr := range samlResp.Assertion.AttributeStatement.Attributes {
			if attr.Name == "email" || attr.Name == "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress" {
				if len(attr.Values) > 0 {
					email = attr.Values[0]
				}
			}
		}
	}
	
	var username string
	for _, attr := range samlResp.Assertion.AttributeStatement.Attributes {
		if attr.Name == "username" || attr.Name == "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name" {
			if len(attr.Values) > 0 {
				username = attr.Values[0]
			}
		}
	}
	if username == "" {
		username = strings.Split(email, "@")[0]
	}
	
	user, err := findOrCreateSSOUser(email, username, string(SSOProviderSAML))
	if err != nil {
		response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to process user: %v", err))
		return
	}
	
	token, refreshToken, err := generateSSOTokens(user)
	if err != nil {
		response.Fail(c, response.CodeServerError, "failed to generate token")
		return
	}
	
	c.JSON(200, gin.H{
		"code": 0,
		"message": "SSO authentication successful",
		"data": gin.H{
			"token":         token,
			"refresh_token": refreshToken,
			"expires_in":    7200,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
			},
		},
	})
}

func LDAPLoginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}
	
	manager := GetSSOProviderManager()
	var ldapConfig *LDAPConfig
	
	for _, config := range manager.GetAllConfigs() {
		if config.IsEnabled && config.ProviderType == SSOProviderLDAP {
			var cfg LDAPConfig
			json.Unmarshal([]byte(config.LDAPConfig), &cfg)
			ldapConfig = &cfg
			break
		}
	}
	
	if ldapConfig == nil {
		response.Fail(c, response.CodeNotFound, "LDAP SSO not configured or disabled")
		return
	}
	
	if ldapService == nil {
		if err := InitLDAPService(*ldapConfig); err != nil {
			response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to initialize LDAP: %v", err))
			return
		}
	}
	
	ldapUser, err := ldapService.Authenticate(req.Username, req.Password)
	if err != nil {
		response.Fail(c, response.CodeUnauthorized, "LDAP authentication failed")
		return
	}
	
	user, err := findOrCreateSSOUser(ldapUser.Email, ldapUser.Username, string(SSOProviderLDAP))
	if err != nil {
		response.Fail(c, response.CodeServerError, fmt.Sprintf("failed to process user: %v", err))
		return
	}
	
	token, refreshToken, err := generateSSOTokens(user)
	if err != nil {
		response.Fail(c, response.CodeServerError, "failed to generate token")
		return
	}
	
	c.JSON(200, gin.H{
		"code": 0,
		"message": "LDAP authentication successful",
		"data": gin.H{
			"token":         token,
			"refresh_token": refreshToken,
			"expires_in":    7200,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
			},
		},
	})
}

func SSOSessionStatusHandler(c *gin.Context) {
	sessionID := c.GetHeader("X-SSO-Session-ID")
	if sessionID == "" {
		sessionID = c.Query("session_id")
	}
	
	if sessionID == "" {
		response.Fail(c, response.CodeBadRequest, "session_id is required")
		return
	}
	
	manager := GetSSOProviderManager()
	session, ok := manager.GetSession(sessionID)
	if !ok {
		response.Fail(c, response.CodeNotFound, "session not found or expired")
		return
	}
	
	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"session_id":  session.SessionID,
			"user_id":     session.UserID,
			"provider":    session.ProviderType,
			"created_at":  session.CreatedAt,
			"expires_at":  session.ExpiresAt,
			"attributes":  session.Attributes,
		},
	})
}

func SSOSessionLogoutHandler(c *gin.Context) {
	sessionID := c.GetHeader("X-SSO-Session-ID")
	if sessionID == "" {
		sessionID = c.Query("session_id")
	}
	
	if sessionID == "" {
		response.Fail(c, response.CodeBadRequest, "session_id is required")
		return
	}
	
	manager := GetSSOProviderManager()
	manager.DeleteSession(sessionID)
	
	if redisClient := redis.GetClient(); redisClient != nil {
		redisClient.Del(fmt.Sprintf("saml_session:%s", sessionID))
	}
	
	c.JSON(200, gin.H{
		"code":    0,
		"message": "SSO session terminated successfully",
	})
}

func findOrCreateSSOUser(email, username, provider string) (*models.User, error) {
	var user models.User
	err := database.DB.Where("email = ?", email).First(&user).Error
	if err == nil {
		return &user, nil
	}
	
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

func generateSSOTokens(user *models.User) (string, string, error) {
	token, err := jwt.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", "", err
	}

	refreshToken := uuid.New().String()

	if redisClient := redis.GetClient(); redisClient != nil {
		key := fmt.Sprintf("sso_refresh:%d", user.ID)
		redisClient.Set(context.Background(), key, refreshToken, 7*24*time.Hour)
	}

	return token, refreshToken, nil
}
