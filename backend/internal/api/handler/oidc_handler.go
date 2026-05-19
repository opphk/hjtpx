package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type OIDCHandler struct {
	manager *service.OIDCServiceManager
}

func NewOIDCHandler() *OIDCHandler {
	return &OIDCHandler{
		manager: service.GetOIDCManager(),
	}
}

func (h *OIDCHandler) GetProviders(c *gin.Context) {
	providers := h.manager.ListProviders()
	enabledProviders := make(map[string]bool)

	for _, provider := range providers {
		enabledProviders[provider] = h.manager.IsProviderEnabled(provider)
	}

	response.Success(c, gin.H{
		"providers": providers,
		"enabled":   enabledProviders,
	})
}

func (h *OIDCHandler) RegisterProvider(c *gin.Context) {
	var req struct {
		Name          string   `json:"name" binding:"required"`
		ClientID      string   `json:"client_id" binding:"required"`
		ClientSecret  string   `json:"client_secret" binding:"required"`
		IssuerURL     string   `json:"issuer_url" binding:"required"`
		RedirectURI   string   `json:"redirect_uri"`
		AuthURL       string   `json:"authorization_url,omitempty"`
		TokenURL      string   `json:"token_url,omitempty"`
		UserInfoURL   string   `json:"user_info_url,omitempty"`
		EndSessionURL string   `json:"end_session_url,omitempty"`
		Scopes        []string `json:"scopes,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	config := &service.OIDCProviderConfig{
		ClientID:        req.ClientID,
		ClientSecret:    req.ClientSecret,
		RedirectURI:     req.RedirectURI,
		IssuerURL:       req.IssuerURL,
		AuthorizationURL: req.AuthURL,
		TokenURL:       req.TokenURL,
		UserInfoURL:     req.UserInfoURL,
		EndSessionURL:   req.EndSessionURL,
		Scopes:          req.Scopes,
	}

	if err := h.manager.RegisterProvider(req.Name, config); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":  "OIDC provider registered successfully",
		"provider": req.Name,
	})
}

func (h *OIDCHandler) RemoveProvider(c *gin.Context) {
	name := c.Param("name")
	h.manager.UnregisterProvider(name)
	response.Success(c, "OIDC provider removed")
}

func (h *OIDCHandler) InitiateLogin(c *gin.Context) {
	providerName := c.Param("provider")

	provider, err := h.manager.GetProvider(providerName)
	if err != nil {
		response.BadRequest(c, "OIDC provider not found: "+providerName)
		return
	}

	stateStr, nonce, state, err := h.manager.GenerateState(providerName)
	if err != nil {
		response.InternalServerError(c, "failed to generate state")
		return
	}

	authURL, err := provider.GetAuthorizationURL(stateStr, nonce)
	if err != nil {
		response.InternalServerError(c, "failed to generate authorization URL")
		return
	}

	response.Success(c, gin.H{
		"auth_url":    authURL,
		"state":       stateStr,
		"nonce":       nonce,
		"provider":    providerName,
		"expires_in":  int(state.ExpiresAt.Sub(state.CreatedAt).Seconds()),
	})
}

func (h *OIDCHandler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	stateStr := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		errorDesc := c.Query("error_description")
		response.BadRequest(c, "OIDC error: "+errorParam+" - "+errorDesc)
		return
	}

	if code == "" || stateStr == "" {
		response.BadRequest(c, "code and state parameters are required")
		return
	}

	state, err := h.manager.ValidateState(stateStr)
	if err != nil {
		response.BadRequest(c, "invalid or expired state")
		return
	}

	provider, err := h.manager.GetProvider(state.Provider)
	if err != nil {
		response.InternalServerError(c, "provider not found")
		return
	}

	tokenResp, err := provider.ExchangeCode(code)
	if err != nil {
		response.InternalServerError(c, "failed to exchange code: "+err.Error())
		return
	}

	userInfo, err := provider.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		response.InternalServerError(c, "failed to get user info: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"provider":      userInfo.Provider,
		"provider_id":   userInfo.ProviderID,
		"username":      userInfo.Username,
		"email":         userInfo.Email,
		"name":          userInfo.Name,
		"avatar_url":    userInfo.AvatarURL,
		"email_verified": userInfo.EmailVerified,
		"access_token":  tokenResp.AccessToken,
		"token_type":    tokenResp.TokenType,
		"expires_in":    tokenResp.ExpiresIn,
		"refresh_token": tokenResp.RefreshToken,
	})
}

func (h *OIDCHandler) RefreshToken(c *gin.Context) {
	var req struct {
		Provider     string `json:"provider" binding:"required"`
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	provider, err := h.manager.GetProvider(req.Provider)
	if err != nil {
		response.BadRequest(c, "invalid or unsupported provider")
		return
	}

	tokenResp, err := provider.RefreshToken(req.RefreshToken)
	if err != nil {
		response.InternalServerError(c, "failed to refresh token: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"access_token":  tokenResp.AccessToken,
		"token_type":    tokenResp.TokenType,
		"expires_in":    tokenResp.ExpiresIn,
		"refresh_token": tokenResp.RefreshToken,
	})
}

func (h *OIDCHandler) RevokeToken(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		Token    string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	provider, err := h.manager.GetProvider(req.Provider)
	if err != nil {
		response.BadRequest(c, "invalid or unsupported provider")
		return
	}

	if err := provider.RevokeToken(req.Token); err != nil {
		response.InternalServerError(c, "failed to revoke token: "+err.Error())
		return
	}

	response.Success(c, "token revoked successfully")
}

func (h *OIDCHandler) GetUserInfo(c *gin.Context) {
	var req struct {
		Provider    string `json:"provider" binding:"required"`
		AccessToken string `json:"access_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	provider, err := h.manager.GetProvider(req.Provider)
	if err != nil {
		response.BadRequest(c, "invalid or unsupported provider")
		return
	}

	userInfo, err := provider.GetUserInfo(req.AccessToken)
	if err != nil {
		response.InternalServerError(c, "failed to get user info: "+err.Error())
		return
	}

	response.Success(c, userInfo)
}

func (h *OIDCHandler) GetEndSessionURL(c *gin.Context) {
	providerName := c.Param("provider")
	postLogoutRedirectURI := c.Query("redirect_uri")

	provider, err := h.manager.GetProvider(providerName)
	if err != nil {
		response.BadRequest(c, "OIDC provider not found")
		return
	}

	endSessionURL, err := provider.GetEndSessionURL(postLogoutRedirectURI)
	if err != nil {
		response.InternalServerError(c, "failed to get end session URL")
		return
	}

	response.Success(c, gin.H{
		"end_session_url": endSessionURL,
		"provider":        providerName,
	})
}

func RegisterOIDCRoutes(r *gin.RouterGroup) {
	handler := NewOIDCHandler()

	oidc := r.Group("/oidc")
	{
		oidc.GET("/providers", handler.GetProviders)
		oidc.POST("/providers", handler.RegisterProvider)
		oidc.DELETE("/providers/:name", handler.RemoveProvider)
		oidc.GET("/:provider/login", handler.InitiateLogin)
		oidc.GET("/:provider/callback", handler.HandleCallback)
		oidc.POST("/refresh", handler.RefreshToken)
		oidc.POST("/revoke", handler.RevokeToken)
		oidc.POST("/userinfo", handler.GetUserInfo)
		oidc.GET("/:provider/logout", handler.GetEndSessionURL)
	}
}