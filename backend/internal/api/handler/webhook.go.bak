package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var webhookServiceInstance *service.WebhookService

// InitWebhookService 初始化 Webhook 服务
func InitWebhookService() {
	webhookServiceInstance = service.NewWebhookService()
}

// GetWebhookService 获取 Webhook 服务实例
func GetWebhookService() *service.WebhookService {
	return webhookServiceInstance
}

// WebhookRequest Webhook 请求
type WebhookRequest struct {
	Event string      `json:"event" binding:"required"`
	Data  interface{} `json:"data"`
}

// HandleWebhook 处理 Webhook 请求
// @Summary 处理 Webhook 请求
// @Description 接收并处理第三方服务发送的 Webhook 事件
// @Tags Webhook
// @Accept json
// @Produce json
// @Param body body WebhookRequest true "Webhook 请求"
// @Success 200 {object} response.Response "处理成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "签名验证失败"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/webhook [post]
func HandleWebhook(c *gin.Context) {
	// 验证签名
	if webhookServiceInstance.GetSignatureVerifier() != nil {
		valid, _, err := webhookServiceInstance.GetSignatureVerifier().VerifyFromRequest(c.Request)
		if err != nil {
			response.InternalServerError(c, "failed to verify signature")
			return
		}
		if !valid {
			response.Unauthorized(c, "invalid signature")
			return
		}
	}

	// 解析请求体
	var req WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	// 处理 Webhook 事件
	result, err := processWebhookEvent(req)
	if err != nil {
		response.InternalServerError(c, "failed to process webhook: "+err.Error())
		return
	}

	response.Success(c, result)
}

// processWebhookEvent 处理 Webhook 事件
func processWebhookEvent(req WebhookRequest) (interface{}, error) {
	// 这里可以根据事件类型进行不同的处理
	// 目前只是简单地返回事件信息
	return map[string]interface{}{
		"event":   req.Event,
		"status":  "processed",
		"message": "Webhook event received successfully",
	}, nil
}

// OAuth2InitiateRequest OAuth2 发起授权请求
type OAuth2InitiateRequest struct {
	Provider string `json:"provider" binding:"required"`
	State    string `json:"state" binding:"required"`
}

// OAuth2InitiateResponse OAuth2 发起授权响应
type OAuth2InitiateResponse struct {
	AuthorizationURL string `json:"authorization_url"`
}

// InitiateOAuth2 发起 OAuth2 授权流程
// @Summary 发起 OAuth2 授权
// @Description 生成 OAuth2 授权 URL，引导用户到第三方服务进行授权
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param body body OAuth2InitiateRequest true "OAuth2 授权请求"
// @Success 200 {object} response.Response{data=OAuth2InitiateResponse} "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "Provider 不存在"
// @Router /api/v1/oauth2/initiate [post]
func InitiateOAuth2(c *gin.Context) {
	var req OAuth2InitiateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	oauth2Service := webhookServiceInstance.GetOAuth2Service(req.Provider)
	if oauth2Service == nil {
		response.NotFound(c, "provider not found")
		return
	}

	authURL := oauth2Service.GetAuthorizationURL(req.State)
	response.Success(c, OAuth2InitiateResponse{
		AuthorizationURL: authURL,
	})
}

// OAuth2CallbackRequest OAuth2 回调请求
type OAuth2CallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

// OAuth2CallbackResponse OAuth2 回调响应
type OAuth2CallbackResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// HandleOAuth2Callback 处理 OAuth2 回调
// @Summary 处理 OAuth2 回调
// @Description 接收第三方服务的 OAuth2 回调，交换授权码获取访问令牌
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param provider path string true "OAuth2 提供商名称"
// @Param code query string true "授权码"
// @Param state query string true "状态值"
// @Success 200 {object} response.Response{data=OAuth2CallbackResponse} "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "Provider 不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/oauth2/callback/{provider} [get]
func HandleOAuth2Callback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		response.BadRequest(c, "code and state are required")
		return
	}

	oauth2Service := webhookServiceInstance.GetOAuth2Service(provider)
	if oauth2Service == nil {
		response.NotFound(c, "provider not found")
		return
	}

	token, err := oauth2Service.ExchangeCode(code)
	if err != nil {
		response.InternalServerError(c, "failed to exchange code: "+err.Error())
		return
	}

	response.Success(c, OAuth2CallbackResponse{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.ExpiresIn,
		RefreshToken: token.RefreshToken,
	})
}

// OAuth2RefreshRequest OAuth2 刷新令牌请求
type OAuth2RefreshRequest struct {
	Provider     string `json:"provider" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshOAuth2Token 刷新 OAuth2 令牌
// @Summary 刷新 OAuth2 令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param body body OAuth2RefreshRequest true "刷新令牌请求"
// @Success 200 {object} response.Response{data=OAuth2CallbackResponse} "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "Provider 不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/oauth2/refresh [post]
func RefreshOAuth2Token(c *gin.Context) {
	var req OAuth2RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	oauth2Service := webhookServiceInstance.GetOAuth2Service(req.Provider)
	if oauth2Service == nil {
		response.NotFound(c, "provider not found")
		return
	}

	token, err := oauth2Service.RefreshToken(req.RefreshToken)
	if err != nil {
		response.InternalServerError(c, "failed to refresh token: "+err.Error())
		return
	}

	response.Success(c, OAuth2CallbackResponse{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.ExpiresIn,
		RefreshToken: token.RefreshToken,
	})
}

// RegisterOAuth2ProviderRequest 注册 OAuth2 提供商请求
type RegisterOAuth2ProviderRequest struct {
	Name         string                `json:"name" binding:"required"`
	ClientID     string                `json:"client_id" binding:"required"`
	ClientSecret string                `json:"client_secret" binding:"required"`
	RedirectURI  string                `json:"redirect_uri" binding:"required"`
	AuthURL      string                `json:"auth_url" binding:"required"`
	TokenURL     string                `json:"token_url" binding:"required"`
	Scope        []string              `json:"scope,omitempty"`
}

// RegisterOAuth2Provider 注册 OAuth2 提供商
// @Summary 注册 OAuth2 提供商
// @Description 注册新的 OAuth2 提供商配置
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param body body RegisterOAuth2ProviderRequest true "注册请求"
// @Success 200 {object} response.Response "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/oauth2/providers [post]
func RegisterOAuth2Provider(c *gin.Context) {
	var req RegisterOAuth2ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	config := &service.OAuth2Config{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		RedirectURI:  req.RedirectURI,
		AuthURL:      req.AuthURL,
		TokenURL:     req.TokenURL,
		Scope:        req.Scope,
	}

	webhookServiceInstance.RegisterOAuth2Service(req.Name, config)
	response.Success(c, gin.H{"message": "provider registered successfully"})
}

// ListOAuth2Providers 列出所有 OAuth2 提供商
// @Summary 列出 OAuth2 提供商
// @Description 获取所有已注册的 OAuth2 提供商列表
// @Tags OAuth2
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]string} "成功"
// @Router /api/v1/oauth2/providers [get]
func ListOAuth2Providers(c *gin.Context) {
	providers := webhookServiceInstance.ListOAuth2Services()
	response.Success(c, providers)
}

// SetWebhookSecretRequest 设置 Webhook 密钥请求
type SetWebhookSecretRequest struct {
	Secret string `json:"secret" binding:"required"`
}

// SetWebhookSecret 设置 Webhook 签名密钥
// @Summary 设置 Webhook 签名密钥
// @Description 配置 Webhook 请求的签名验证密钥
// @Tags Webhook
// @Accept json
// @Produce json
// @Param body body SetWebhookSecretRequest true "设置密钥请求"
// @Success 200 {object} response.Response "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/webhook/secret [post]
func SetWebhookSecret(c *gin.Context) {
	var req SetWebhookSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	webhookServiceInstance.SetSignatureVerifier(req.Secret)
	response.Success(c, gin.H{"message": "webhook secret set successfully"})
}

// TestWebhookSignature 测试 Webhook 签名
// @Summary 测试 Webhook 签名
// @Description 生成测试签名，用于调试
// @Tags Webhook
// @Accept json
// @Produce json
// @Router /api/v1/webhook/test-signature [post]
func TestWebhookSignature(c *gin.Context) {
	// 这个端点可以用于测试签名生成
	response.Success(c, gin.H{"message": "signature test endpoint"})
}
