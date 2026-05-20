package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type OAuth2Handler struct {
	manager *service.OAuth2ServiceManager
}

func NewOAuth2Handler() *OAuth2Handler {
	return &OAuth2Handler{
		manager: service.GetOAuth2Manager(),
	}
}

func (h *OAuth2Handler) GetProviders(c *gin.Context) {
	providers := h.manager.ListProviders()
	enabledProviders := make(map[string]bool)

	for _, provider := range providers {
		enabledProviders[string(provider)] = h.manager.IsProviderEnabled(provider)
	}

	response.Success(c, gin.H{
		"providers":      providers,
		"enabled":        enabledProviders,
	})
}

func (h *OAuth2Handler) InitiateLogin(c *gin.Context) {
	providerStr := c.Query("provider")
	if providerStr == "" {
		response.BadRequest(c, "provider parameter is required")
		return
	}

	provider := service.OAuth2ProviderType(providerStr)
	providerService, err := h.manager.GetProvider(provider)
	if err != nil {
		response.BadRequest(c, "invalid or unsupported provider: "+providerStr)
		return
	}

	stateStr, state, err := h.manager.GenerateState(provider)
	if err != nil {
		response.InternalServerError(c, "failed to generate state")
		return
	}

	authURL, err := providerService.GetAuthorizationURL(stateStr)
	if err != nil {
		response.InternalServerError(c, "failed to generate authorization URL")
		return
	}

	response.Success(c, gin.H{
		"auth_url":  authURL,
		"state":     stateStr,
		"provider":  string(provider),
		"expires_in": int(state.ExpiresAt.Sub(state.CreatedAt).Seconds()),
	})
}

func (h *OAuth2Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	stateStr := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		errorDesc := c.Query("error_description")
		response.BadRequest(c, "oauth2 error: "+errorParam+" - "+errorDesc)
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

	provider := service.OAuth2ProviderType(state.Provider)
	providerService, err := h.manager.GetProvider(provider)
	if err != nil {
		response.InternalServerError(c, "provider not found")
		return
	}

	tokenResp, err := providerService.ExchangeCode(code)
	if err != nil {
		response.InternalServerError(c, "failed to exchange code: "+err.Error())
		return
	}

	userInfo, err := providerService.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		response.InternalServerError(c, "failed to get user info: "+err.Error())
		return
	}

	if userInfo.Email == "" {
		response.Success(c, gin.H{
			"provider":     userInfo.Provider,
			"provider_id":  userInfo.ProviderID,
			"username":     userInfo.Username,
			"email_missing": true,
			"access_token": tokenResp.AccessToken,
			"token_type":   tokenResp.TokenType,
			"expires_in":   tokenResp.ExpiresIn,
		})
		return
	}

	response.Success(c, gin.H{
		"provider":     userInfo.Provider,
		"provider_id":  userInfo.ProviderID,
		"username":     userInfo.Username,
		"email":        userInfo.Email,
		"name":         userInfo.Name,
		"avatar_url":   userInfo.AvatarURL,
		"access_token": tokenResp.AccessToken,
		"token_type":   tokenResp.TokenType,
		"expires_in":   tokenResp.ExpiresIn,
	})
}

func (h *OAuth2Handler) RefreshToken(c *gin.Context) {
	var req struct {
		Provider     string `json:"provider" binding:"required"`
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	provider := service.OAuth2ProviderType(req.Provider)
	providerService, err := h.manager.GetProvider(provider)
	if err != nil {
		response.BadRequest(c, "invalid or unsupported provider")
		return
	}

	tokenResp, err := providerService.RefreshToken(req.RefreshToken)
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

func (h *OAuth2Handler) RevokeToken(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		Token    string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	provider := service.OAuth2ProviderType(req.Provider)
	providerService, err := h.manager.GetProvider(provider)
	if err != nil {
		response.BadRequest(c, "invalid or unsupported provider")
		return
	}

	if err := providerService.RevokeToken(req.Token); err != nil {
		response.InternalServerError(c, "failed to revoke token: "+err.Error())
		return
	}

	response.Success(c, "token revoked successfully")
}

func (h *OAuth2Handler) GetUserInfo(c *gin.Context) {
	var req struct {
		Provider    string `json:"provider" binding:"required"`
		AccessToken string `json:"access_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	provider := service.OAuth2ProviderType(req.Provider)
	providerService, err := h.manager.GetProvider(provider)
	if err != nil {
		response.BadRequest(c, "invalid or unsupported provider")
		return
	}

	userInfo, err := providerService.GetUserInfo(req.AccessToken)
	if err != nil {
		response.InternalServerError(c, "failed to get user info: "+err.Error())
		return
	}

	response.Success(c, userInfo)
}

func RegisterOAuth2Routes(r *gin.RouterGroup) {
	handler := NewOAuth2Handler()

	oauth := r.Group("/oauth2")
	{
		oauth.GET("/providers", handler.GetProviders)
		oauth.GET("/login/:provider", handler.InitiateLogin)
		oauth.GET("/callback", handler.HandleCallback)
		oauth.POST("/refresh", handler.RefreshToken)
		oauth.POST("/revoke", handler.RevokeToken)
		oauth.POST("/userinfo", handler.GetUserInfo)
	}
}

func OAuth2LoginHandler(c *gin.Context) {
	provider := c.Param("provider")

	manager := service.GetOAuth2Manager()
	providerService, err := manager.GetProvider(service.OAuth2ProviderType(provider))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or unsupported provider"})
		return
	}

	stateStr, state, err := manager.GenerateState(service.OAuth2ProviderType(provider))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	authURL, err := providerService.GetAuthorizationURL(stateStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate authorization URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url":   authURL,
		"state":      stateStr,
		"provider":   provider,
		"expires_in": int(state.ExpiresAt.Sub(state.CreatedAt).Seconds()),
	})
}
