package middleware

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/pkg/logger"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	ErrSAMLAssertionInvalid    = errors.New("saml assertion is invalid")
	ErrSAMLAssertionExpired    = errors.New("saml assertion has expired")
	ErrSAMLConfigurationInvalid = errors.New("saml configuration is invalid")
	ErrCASSessionInvalid       = errors.New("cas session is invalid")
	ErrCASSessionExpired       = errors.New("cas session has expired")
	ErrSSOProviderNotFound    = errors.New("sso provider not found")
)

type SSOProviderType string

const (
	SSOProviderSAML SSOProviderType = "saml"
	SSOProviderCAS SSOProviderType = "cas"
)

type SAMLConfig struct {
	EntityID          string            `json:"entity_id"`
	SSOURL            string            `json:"sso_url"`
	SLOURL            string            `json:"slo_url,omitempty"`
	Certificate       string            `json:"certificate"`
	PrivateKey        string            `json:"private_key,omitempty"`
	AssertionConsumerServiceURL string `json:"acs_url"`
	ServiceProviderEntityID string      `json:"sp_entity_id"`
	NameIDFormat     string            `json:"name_id_format,omitempty"`
	AuthnContextClass string           `json:"authn_context_class,omitempty"`
	WantAssertionsSigned bool          `json:"want_assertions_signed"`
	WantAssertionsEncrypted bool       `json:"want_assertions_encrypted"`
	AllowedAudiences []string          `json:"allowed_audiences,omitempty"`
}

type SAMLAssertion struct {
	XMLName xml.Name `xml:"Assertion"`
	ID      string   `xml:"ID,attr"`
	Issuer  string   `xml:"Issuer"`
	Subject struct {
		NameID string `xml:"NameID"`
	} `xml:"Subject"`
	Conditions struct {
		NotBefore    time.Time `xml:"NotBefore,attr"`
		NotOnOrAfter time.Time `xml:"NotOnOrAfter,attr"`
		Audience     string    `xml:"Audience"`
	} `xml:"Conditions"`
	AuthnStatement struct {
		AuthnInstant time.Time `xml:"AuthnInstant,attr"`
		SessionIndex string    `xml:"SessionIndex,attr"`
	} `xml:"AuthnStatement"`
	Attributes struct {
		Attribute []struct {
			Name   string   `xml:"Name,attr"`
			Values []string `xml:"AttributeValue"`
		} `xml:"Attribute"`
	} `xml:"AttributeStatement>Attribute"`
}

type SAMLService struct {
	config      *SAMLConfig
	certificate *x509.Certificate
	logger      *logger.Logger
}

type CASConfig struct {
	ServerURL     string   `json:"server_url"`
	ServiceURL    string   `json:"service_url"`
	ProxyCallback string   `json:"proxy_callback,omitempty"`
	Version       string   `json:"version,omitempty"`
	AllowProxy    bool     `json:"allow_proxy"`
	PGTURL        string   `json:"pgt_url,omitempty"`
	Attributes    []string `json:"attributes,omitempty"`
}

type CASValidationResponse struct {
	ServiceTicket string   `json:"service_ticket"`
	User          string   `json:"user"`
	Attributes    map[string][]string `json:"attributes,omitempty"`
	Proxies       []string `json:"proxies,omitempty"`
}

type CASService struct {
	config *CASConfig
	client *http.Client
	logger *logger.Logger
}

type SSOManager struct {
	samlProviders map[string]*SAMLService
	casProviders   map[string]*CASService
	mu             sync.RWMutex
	logger         *logger.Logger
}

type SSOUserInfo struct {
	Provider   string                 `json:"provider"`
	ProviderID string                 `json:"provider_id"`
	Username   string                 `json:"username"`
	Email      string                 `json:"email,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
}

var ssoManager *SSOManager
var ssoOnce sync.Once

func GetSSOManager() *SSOManager {
	ssoOnce.Do(func() {
		ssoManager = &SSOManager{
			samlProviders: make(map[string]*SAMLService),
			casProviders:   make(map[string]*CASService),
			logger:         logger.Default(),
		}
	})
	return ssoManager
}

func (m *SSOManager) RegisterSAMLProvider(name string, config *SAMLConfig) error {
	if config.EntityID == "" || config.SSOURL == "" || config.Certificate == "" {
		return ErrSAMLConfigurationInvalid
	}

	certBytes, err := base64.StdEncoding.DecodeString(config.Certificate)
	if err != nil {
		return fmt.Errorf("invalid certificate format: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.samlProviders[name] = &SAMLService{
		config:      config,
		certificate: cert,
		logger:      m.logger,
	}

	m.logger.Log(logger.INFO, "SAML provider registered", logger.Fields{"name": name})
	return nil
}

func (m *SSOManager) RegisterCASProvider(name string, config *CASConfig) error {
	if config.ServerURL == "" || config.ServiceURL == "" {
		return fmt.Errorf("server_url and service_url are required for CAS")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.casProviders[name] = &CASService{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: m.logger,
	}

	m.logger.Log(logger.INFO, "CAS provider registered", logger.Fields{"name": name})
	return nil
}

func (m *SSOManager) GetSAMLProvider(name string) (*SAMLService, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.samlProviders[name]
	if !exists {
		return nil, ErrSSOProviderNotFound
	}
	return provider, nil
}

func (m *SSOManager) GetCASProvider(name string) (*CASService, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.casProviders[name]
	if !exists {
		return nil, ErrSSOProviderNotFound
	}
	return provider, nil
}

func (s *SAMLService) GetAuthnRequestID() string {
	return fmt.Sprintf("_%d", time.Now().UnixNano())
}

func (s *SAMLService) BuildAuthnRequest() (string, string, error) {
	requestID := s.GetAuthnRequestID()
	issueInstant := time.Now().UTC()

	nameIDFormat := s.config.NameIDFormat
	if nameIDFormat == "" {
		nameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	}

	authnRequest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="%s"
    Version="2.0"
    IssueInstant="%s"
    Destination="%s"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
    AssertionConsumerServiceURL="%s">
    <saml:Issuer>%s</saml:Issuer>
    <samlp:NameIDPolicy
        Format="%s"
        AllowCreate="true"/>
    <samlp:RequestedAuthnContext>
        <saml:AuthnContextClassRef>%s</saml:AuthnContextClassRef>
    </samlp:RequestedAuthnContext>
</samlp:AuthnRequest>`,
		requestID,
		issueInstant.Format(time.RFC3339Nano),
		s.config.SSOURL,
		s.config.AssertionConsumerServiceURL,
		s.config.EntityID,
		nameIDFormat,
		s.config.AuthnContextClass,
	)

	encoded := base64.StdEncoding.EncodeToString([]byte(authnRequest))
	return requestID, encoded, nil
}

func (s *SAMLService) ValidateResponse(responseB64 string) (*SAMLAssertion, error) {
	responseXML, err := base64.StdEncoding.DecodeString(responseB64)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode base64", ErrSAMLAssertionInvalid)
	}

	var assertion SAMLAssertion
	if err := xml.Unmarshal(responseXML, &assertion); err != nil {
		return nil, fmt.Errorf("%w: failed to parse XML", ErrSAMLAssertionInvalid)
	}

	if time.Now().After(assertion.Conditions.NotOnOrAfter) {
		return nil, ErrSAMLAssertionExpired
	}

	if time.Now().Before(assertion.Conditions.NotBefore) {
		return nil, ErrSAMLAssertionInvalid
	}

	if len(s.config.AllowedAudiences) > 0 {
		allowed := false
		for _, aud := range s.config.AllowedAudiences {
			if aud == assertion.Conditions.Audience {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("%w: audience not allowed", ErrSAMLAssertionInvalid)
		}
	}

	s.logger.Log(logger.INFO, "SAML assertion validated", logger.Fields{"assertion_id": assertion.ID, "name_id": assertion.Subject.NameID})
	return &assertion, nil
}

func (s *SAMLService) ExtractUserInfo(assertion *SAMLAssertion) *SSOUserInfo {
	userInfo := &SSOUserInfo{
		Provider:   "saml",
		ProviderID: assertion.ID,
		Username:   assertion.Subject.NameID,
		Attributes: make(map[string]interface{}),
	}

	for _, attr := range assertion.Attributes.Attribute {
		if len(attr.Values) == 1 {
			userInfo.Attributes[attr.Name] = attr.Values[0]
		} else {
			userInfo.Attributes[attr.Name] = attr.Values
		}

		switch attr.Name {
		case "email", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
			userInfo.Email = attr.Values[0]
		case "username", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name":
			userInfo.Username = attr.Values[0]
		}
	}

	return userInfo
}

func (c *CASService) BuildLoginURL(ticket string) string {
	u, _ := url.Parse(c.config.ServerURL + "/login")
	q := u.Query()
	q.Set("service", c.config.ServiceURL)
	if ticket != "" {
		q.Set("ticket", ticket)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *CASService) BuildLogoutURL(redirectURL string) string {
	u, _ := url.Parse(c.config.ServerURL + "/logout")
	q := u.Query()
	q.Set("service", redirectURL)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *CASService) ValidateTicket(ticket string) (*CASValidationResponse, error) {
	validateURL := c.config.ServerURL + "/proxyValidate"
	if c.config.Version == "1.0" {
		validateURL = c.config.ServerURL + "/validate"
	}

	u, _ := url.Parse(validateURL)
	q := u.Query()
	q.Set("ticket", ticket)
	q.Set("service", c.config.ServiceURL)
	if c.config.PGTURL != "" {
		q.Set("pgtUrl", c.config.PGTURL)
	}
	u.RawQuery = q.Encode()

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCASSessionInvalid, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response", ErrCASSessionInvalid)
	}

	return c.parseValidationResponse(string(body))
}

func (c *CASService) parseValidationResponse(body string) (*CASValidationResponse, error) {
	var validationResp CASValidationResponse
	validationResp.Attributes = make(map[string][]string)

	if c.config.Version == "1.0" {
		if len(body) < 4 || body[:4] != "yes\n" {
			return nil, ErrCASSessionInvalid
		}
		lines := splitLines(body)
		if len(lines) >= 2 {
			validationResp.User = lines[1]
		}
	} else {
		startIdx := 0
		if len(body) >= 9 && body[:9] == "<cas:succ" {
			startIdx = 9
		}

		if startIdx == 0 || body[startIdx:startIdx+6] != "ess>" {
			return nil, ErrCASSessionInvalid
		}

		endIdx := findString(body, "</cas:success>", startIdx)
		if endIdx == -1 {
			return nil, ErrCASSessionInvalid
		}

		userEnd := findString(body, "</cas:user>", startIdx)
		if userEnd != -1 && userEnd < endIdx {
			validationResp.User = body[startIdx+6 : userEnd]
		}

		attrStart := findString(body, "<cas:attributes>", startIdx)
		if attrStart != -1 && attrStart < endIdx {
			attrEnd := findString(body, "</cas:attributes>", attrStart)
			if attrEnd != -1 && attrEnd < endIdx {
				c.parseAttributes(body[attrStart:attrEnd], validationResp.Attributes)
			}
		}
	}

	c.logger.Log(logger.INFO, "CAS ticket validated", logger.Fields{"user": validationResp.User})
	return &validationResp, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if start < i {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func findString(s, substr string, start int) int {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (c *CASService) parseAttributes(attrsXML string, attrs map[string][]string) {
	start := 0
	for {
		tagStart := findString(attrsXML, "<cas:", start)
		if tagStart == -1 {
			break
		}

		tagEnd := findString(attrsXML, ">", tagStart)
		if tagEnd == -1 {
			break
		}

		tagClose := findString(attrsXML, "</cas:", tagEnd)
		if tagClose == -1 {
			break
		}

		tag := attrsXML[tagStart+5 : tagEnd]
		value := attrsXML[tagEnd+1 : tagClose]

		if tag == "attribute" {
			nameStart := findString(attrsXML, "name=\"", tagStart)
			if nameStart != -1 && nameStart < tagEnd {
				nameEnd := findString(attrsXML, "\"", nameStart+6)
				if nameEnd != -1 && nameEnd < tagEnd {
					name := attrsXML[nameStart+6 : nameEnd]
					attrs[name] = append(attrs[name], value)
				}
			}
		} else {
			attrs[tag] = append(attrs[tag], value)
		}

		start = tagClose + len("</cas:>")
	}
}

func (c *CASService) ExtractUserInfo(resp *CASValidationResponse) *SSOUserInfo {
	userInfo := &SSOUserInfo{
		Provider:    "cas",
		ProviderID:  resp.ServiceTicket,
		Username:    resp.User,
		Attributes:  make(map[string]interface{}),
	}

	for key, values := range resp.Attributes {
		if len(values) == 1 {
			userInfo.Attributes[key] = values[0]
		} else {
			userInfo.Attributes[key] = values
		}

		switch key {
		case "email", "mail":
			userInfo.Email = values[0]
		}
	}

	return userInfo
}

func SSOMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ssoSession := c.GetHeader("X-SSO-Session")
		if ssoSession == "" {
			ssoSession, _ = c.Cookie("sso_session")
		}

		if ssoSession != "" {
			manager := GetSSOManager()
			userInfo, err := manager.ValidateSSOSession(ssoSession)
			if err == nil && userInfo != nil {
				c.Set("sso_user", userInfo)
				c.Set("sso_provider", userInfo.Provider)
				c.Next()
				return
			}
		}

		c.Next()
	}
}

func (m *SSOManager) ValidateSSOSession(sessionID string) (*SSOUserInfo, error) {
	return nil, ErrCASSessionInvalid
}

func SAMLSPMiddleware(providerName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := GetSSOManager()
		provider, err := manager.GetSAMLProvider(providerName)
		if err != nil {
			response.BadRequest(c, "SAML provider not found")
			c.Abort()
			return
		}

		samlResponse := c.PostForm("SAMLResponse")
		if samlResponse == "" {
			requestID, encoded, err := provider.BuildAuthnRequest()
			if err != nil {
				response.InternalServerError(c, "failed to build authn request")
				c.Abort()
				return
			}

			c.Set("saml_request_id", requestID)
			c.Set("saml_encoded_request", encoded)
			c.Set("saml_provider", providerName)
			c.Set("saml_auth_url", provider.config.SSOURL)

			c.Abort()
			return
		}

		assertion, err := provider.ValidateResponse(samlResponse)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		userInfo := provider.ExtractUserInfo(assertion)
		c.Set("sso_user", userInfo)
		c.Set("sso_provider", "saml")
		c.Next()
	}
}

func CASMiddleware(providerName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := GetSSOManager()
		provider, err := manager.GetCASProvider(providerName)
		if err != nil {
			response.BadRequest(c, "CAS provider not found")
			c.Abort()
			return
		}

		ticket := c.Query("ticket")
		if ticket == "" {
			loginURL := provider.BuildLoginURL("")
			c.Redirect(http.StatusFound, loginURL)
			c.Abort()
			return
		}

		validationResp, err := provider.ValidateTicket(ticket)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		userInfo := provider.ExtractUserInfo(validationResp)
		userInfo.SessionID = fmt.Sprintf("cas_%s_%d", userInfo.Username, time.Now().UnixNano())

		c.Set("sso_user", userInfo)
		c.Set("sso_provider", "cas")
		c.Set("sso_session", userInfo.SessionID)

		c.SetCookie("sso_session", userInfo.SessionID, 3600*8, "/", "", false, true)
		c.Next()
	}
}

func GetSSOUser(c *gin.Context) *SSOUserInfo {
	if user, exists := c.Get("sso_user"); exists {
		return user.(*SSOUserInfo)
	}
	return nil
}

func (m *SSOManager) ListSAMLProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.samlProviders))
	for name := range m.samlProviders {
		names = append(names, name)
	}
	return names
}

func (m *SSOManager) ListCASProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.casProviders))
	for name := range m.casProviders {
		names = append(names, name)
	}
	return names
}

func (m *SSOManager) RemoveSAMLProvider(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.samlProviders, name)
}

func (m *SSOManager) RemoveCASProvider(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.casProviders, name)
}
