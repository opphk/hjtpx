package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type SSOHandler struct {
	ssoManager *middleware.SSOManager
}

func NewSSOHandler() *SSOHandler {
	return &SSOHandler{
		ssoManager: middleware.GetSSOManager(),
	}
}

func (h *SSOHandler) ListSAMLProviders(c *gin.Context) {
	providers := h.ssoManager.ListSAMLProviders()
	response.Success(c, gin.H{
		"providers": providers,
	})
}

func (h *SSOHandler) ListCASProviders(c *gin.Context) {
	providers := h.ssoManager.ListCASProviders()
	response.Success(c, gin.H{
		"providers": providers,
	})
}

func (h *SSOHandler) RegisterSAMLProvider(c *gin.Context) {
	var req struct {
		Name     string                 `json:"name" binding:"required"`
		EntityID string                 `json:"entity_id" binding:"required"`
		SSOURL   string                 `json:"sso_url" binding:"required"`
		Cert     string                 `json:"certificate" binding:"required"`
		ACSURL   string                 `json:"acs_url"`
		SPID     string                 `json:"sp_entity_id"`
		Extra    map[string]interface{} `json:"extra,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	config := &middleware.SAMLConfig{
		EntityID:                    req.EntityID,
		SSOURL:                      req.SSOURL,
		Certificate:                 req.Cert,
		AssertionConsumerServiceURL: req.ACSURL,
		ServiceProviderEntityID:     req.SPID,
	}

	if err := h.ssoManager.RegisterSAMLProvider(req.Name, config); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":   "SAML provider registered successfully",
		"provider":  req.Name,
	})
}

func (h *SSOHandler) RemoveSAMLProvider(c *gin.Context) {
	name := c.Param("name")
	h.ssoManager.RemoveSAMLProvider(name)
	response.Success(c, "SAML provider removed")
}

func (h *SSOHandler) RegisterCASProvider(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		ServerURL   string `json:"server_url" binding:"required"`
		ServiceURL  string `json:"service_url" binding:"required"`
		ProxyURL    string `json:"proxy_callback,omitempty"`
		Version     string `json:"version,omitempty"`
		AllowProxy  bool   `json:"allow_proxy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	config := &middleware.CASConfig{
		ServerURL:     req.ServerURL,
		ServiceURL:    req.ServiceURL,
		ProxyCallback: req.ProxyURL,
		Version:       req.Version,
		AllowProxy:    req.AllowProxy,
	}

	if err := h.ssoManager.RegisterCASProvider(req.Name, config); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":  "CAS provider registered successfully",
		"provider": req.Name,
	})
}

func (h *SSOHandler) RemoveCASProvider(c *gin.Context) {
	name := c.Param("name")
	h.ssoManager.RemoveCASProvider(name)
	response.Success(c, "CAS provider removed")
}

func (h *SSOHandler) InitiateSAMLSSO(c *gin.Context) {
	providerName := c.Param("provider")

	provider, err := h.ssoManager.GetSAMLProvider(providerName)
	if err != nil {
		response.BadRequest(c, "SAML provider not found")
		return
	}

	requestID, encodedRequest, err := provider.BuildAuthnRequest()
	if err != nil {
		response.InternalServerError(c, "failed to build authentication request")
		return
	}

	response.Success(c, gin.H{
		"request_id":     requestID,
		"encoded_request": encodedRequest,
		"sso_url":        provider.GetSSOURL(),
		"provider":       providerName,
	})
}

func (h *SSOHandler) SAMLCallback(c *gin.Context) {
	providerName := c.Param("provider")
	samlResponse := c.PostForm("SAMLResponse")

	if samlResponse == "" {
		response.BadRequest(c, "SAMLResponse is required")
		return
	}

	provider, err := h.ssoManager.GetSAMLProvider(providerName)
	if err != nil {
		response.BadRequest(c, "SAML provider not found")
		return
	}

	assertion, err := provider.ValidateResponse(samlResponse)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	userInfo := provider.ExtractUserInfo(assertion)

	c.Set("sso_user", userInfo)
	c.Set("sso_provider", "saml")

	response.Success(c, gin.H{
		"user": userInfo,
	})
}

func (h *SSOHandler) InitiateCASLogin(c *gin.Context) {
	providerName := c.Param("provider")

	provider, err := h.ssoManager.GetCASProvider(providerName)
	if err != nil {
		response.BadRequest(c, "CAS provider not found")
		return
	}

	loginURL := provider.BuildLoginURL("")

	response.Success(c, gin.H{
		"login_url": loginURL,
		"provider":  providerName,
	})
}

func (h *SSOHandler) CASCallback(c *gin.Context) {
	providerName := c.Param("provider")
	ticket := c.Query("ticket")

	if ticket == "" {
		response.BadRequest(c, "ticket is required")
		return
	}

	provider, err := h.ssoManager.GetCASProvider(providerName)
	if err != nil {
		response.BadRequest(c, "CAS provider not found")
		return
	}

	validationResp, err := provider.ValidateTicket(ticket)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	userInfo := provider.ExtractUserInfo(validationResp)

	c.Set("sso_user", userInfo)
	c.Set("sso_provider", "cas")

	response.Success(c, gin.H{
		"user": userInfo,
	})
}

func (h *SSOHandler) SAMLMetadata(c *gin.Context) {
	providerName := c.Param("provider")

	provider, err := h.ssoManager.GetSAMLProvider(providerName)
	if err != nil {
		response.BadRequest(c, "SAML provider not found")
		return
	}

	metadata := generateSAMLMetadata(provider)
	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, metadata)
}

func generateSAMLMetadata(provider *middleware.SAMLService) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" 
    entityID="` + provider.GetEntityID() + `">
    <md:SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
        <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
        <md:AssertionConsumerService 
            Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" 
            Location="` + provider.GetAssertionConsumerServiceURL() + `" 
            index="1"/>
    </md:SPSSODescriptor>
</md:EntityDescriptor>`
}

func RegisterSSORoutes(r *gin.RouterGroup) {
	handler := NewSSOHandler()

	sso := r.Group("/sso")
	{
		sso.GET("/saml/providers", handler.ListSAMLProviders)
		sso.POST("/saml/providers", handler.RegisterSAMLProvider)
		sso.DELETE("/saml/providers/:name", handler.RemoveSAMLProvider)
		sso.GET("/saml/:provider/login", handler.InitiateSAMLSSO)
		sso.POST("/saml/:provider/callback", handler.SAMLCallback)
		sso.GET("/saml/:provider/metadata", handler.SAMLMetadata)

		sso.GET("/cas/providers", handler.ListCASProviders)
		sso.POST("/cas/providers", handler.RegisterCASProvider)
		sso.DELETE("/cas/providers/:name", handler.RemoveCASProvider)
		sso.GET("/cas/:provider/login", handler.InitiateCASLogin)
		sso.GET("/cas/:provider/callback", handler.CASCallback)
	}
}