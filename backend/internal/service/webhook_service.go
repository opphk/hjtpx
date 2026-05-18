package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// WeChatWorkConfig 企业微信配置
type WeChatWorkConfig struct {
	WebhookURL string   `json:"webhook_url"`
	AtMobiles  []string `json:"at_mobiles,omitempty"`
	IsAtAll    bool     `json:"is_at_all"`
}

// WeChatWorkChannel 企业微信告警渠道
type WeChatWorkChannel struct {
	BaseChannel
	weChatWorkConfig *WeChatWorkConfig
}

// NewWeChatWorkChannel 创建企业微信渠道
func NewWeChatWorkChannel(config map[string]interface{}) (*WeChatWorkChannel, error) {
	weChatWorkConfig, err := ParseWeChatWorkConfig(config)
	if err != nil {
		return nil, err
	}
	return &WeChatWorkChannel{
		BaseChannel:       NewBaseChannel(config),
		weChatWorkConfig: weChatWorkConfig,
	}, nil
}

// Name 渠道名称
func (w *WeChatWorkChannel) Name() string {
	return "wechat_work"
}

// ValidateConfig 验证配置
func (w *WeChatWorkChannel) ValidateConfig() error {
	_, err := ParseWeChatWorkConfig(w.GetConfig())
	return err
}

// Send 发送告警
func (w *WeChatWorkChannel) Send(msg AlertMessage) error {
	payload := w.buildWeChatWorkPayload(msg)
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(w.weChatWorkConfig.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("wechat work API returned status %d", resp.StatusCode)
	}
	return nil
}

func (w *WeChatWorkChannel) buildWeChatWorkPayload(msg AlertMessage) map[string]interface{} {
	content := fmt.Sprintf("### [%s] %s\n\n", msg.Severity, msg.Title)
	content += fmt.Sprintf("> **严重级别:** %s\n", msg.Severity)
	content += fmt.Sprintf("> **事件ID:** %s\n", msg.EventID)
	content += fmt.Sprintf("> **时间:** %s\n", msg.Timestamp.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("> **详细信息:**\n> %s\n", msg.Message)
	if len(msg.Context) > 0 {
		content += "> **上下文:**\n"
		for k, v := range msg.Context {
			content += fmt.Sprintf("> - %s: %v\n", k, v)
		}
	}

	return map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": content,
		},
	}
}

// ParseWeChatWorkConfig 解析企业微信配置
func ParseWeChatWorkConfig(config map[string]interface{}) (*WeChatWorkConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var weChatWorkConfig WeChatWorkConfig
	err = json.Unmarshal(data, &weChatWorkConfig)
	if err != nil {
		return nil, err
	}
	if weChatWorkConfig.WebhookURL == "" {
		return nil, fmt.Errorf("wechat work webhook URL is required")
	}
	return &weChatWorkConfig, nil
}

// WebhookSignatureVerifier Webhook 签名验证器
type WebhookSignatureVerifier struct {
	secret string
}

// NewWebhookSignatureVerifier 创建 Webhook 签名验证器
func NewWebhookSignatureVerifier(secret string) *WebhookSignatureVerifier {
	return &WebhookSignatureVerifier{
		secret: secret,
	}
}

// Verify 验证 Webhook 签名
func (v *WebhookSignatureVerifier) Verify(signature string, timestamp string, body []byte) bool {
	if signature == "" || timestamp == "" || v.secret == "" {
		return false
	}

	// 构造签名数据
	signData := timestamp + string(body)
	
	// 使用 HMAC-SHA256 签名
	h := hmac.New(sha256.New, []byte(v.secret))
	h.Write([]byte(signData))
	expectedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	
	return expectedSignature == signature
}

// VerifyFromRequest 从 HTTP 请求中验证签名
func (v *WebhookSignatureVerifier) VerifyFromRequest(r *http.Request) (bool, []byte, error) {
	// 获取签名和时间戳
	signature := r.Header.Get("X-Webhook-Signature")
	timestamp := r.Header.Get("X-Webhook-Timestamp")
	
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, nil, err
	}
	defer r.Body.Close()
	
	// 恢复请求体，以便后续处理
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	
	// 验证签名
	valid := v.Verify(signature, timestamp, body)
	return valid, body, nil
}

// OAuth2Config OAuth2 配置
type OAuth2Config struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	Scope        []string `json:"scope,omitempty"`
}

// OAuth2Token OAuth2 令牌
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// OAuth2Service OAuth2 服务
type OAuth2Service struct {
	config *OAuth2Config
	token  *OAuth2Token
	mu     sync.RWMutex
}

// NewOAuth2Service 创建 OAuth2 服务
func NewOAuth2Service(config *OAuth2Config) *OAuth2Service {
	return &OAuth2Service{
		config: config,
	}
}

// GetAuthorizationURL 获取授权 URL
func (s *OAuth2Service) GetAuthorizationURL(state string) string {
	u, _ := url.Parse(s.config.AuthURL)
	q := u.Query()
	q.Set("client_id", s.config.ClientID)
	q.Set("redirect_uri", s.config.RedirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	if len(s.config.Scope) > 0 {
		scopeStr := ""
		for i, scope := range s.config.Scope {
			if i > 0 {
				scopeStr += " "
			}
			scopeStr += scope
		}
		q.Set("scope", scopeStr)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// ExchangeCode 交换授权码获取令牌
func (s *OAuth2Service) ExchangeCode(code string) (*OAuth2Token, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.config.RedirectURI)
	
	resp, err := http.PostForm(s.config.TokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var token OAuth2Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	
	// 设置过期时间
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	
	s.mu.Lock()
	s.token = &token
	s.mu.Unlock()
	
	return &token, nil
}

// RefreshToken 刷新令牌
func (s *OAuth2Service) RefreshToken(refreshToken string) (*OAuth2Token, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("refresh_token", refreshToken)
	
	resp, err := http.PostForm(s.config.TokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var token OAuth2Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	
	// 设置过期时间
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	
	s.mu.Lock()
	s.token = &token
	s.mu.Unlock()
	
	return &token, nil
}

// GetToken 获取当前令牌
func (s *OAuth2Service) GetToken() *OAuth2Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// IsTokenExpired 检查令牌是否过期
func (s *OAuth2Service) IsTokenExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.token == nil {
		return true
	}
	if s.token.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(s.token.ExpiresAt)
}

// SetToken 设置令牌
func (s *OAuth2Service) SetToken(token *OAuth2Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
}

// WebhookService Webhook 服务
type WebhookService struct {
	signatureVerifier *WebhookSignatureVerifier
	oauth2Services    map[string]*OAuth2Service
	mu                sync.RWMutex
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService() *WebhookService {
	return &WebhookService{
		oauth2Services: make(map[string]*OAuth2Service),
	}
}

// SetSignatureVerifier 设置签名验证器
func (s *WebhookService) SetSignatureVerifier(secret string) {
	s.signatureVerifier = NewWebhookSignatureVerifier(secret)
}

// GetSignatureVerifier 获取签名验证器
func (s *WebhookService) GetSignatureVerifier() *WebhookSignatureVerifier {
	return s.signatureVerifier
}

// RegisterOAuth2Service 注册 OAuth2 服务
func (s *WebhookService) RegisterOAuth2Service(name string, config *OAuth2Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.oauth2Services[name] = NewOAuth2Service(config)
}

// GetOAuth2Service 获取 OAuth2 服务
func (s *WebhookService) GetOAuth2Service(name string) *OAuth2Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.oauth2Services[name]
}

// ListOAuth2Services 列出所有 OAuth2 服务
func (s *WebhookService) ListOAuth2Services() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.oauth2Services))
	for name := range s.oauth2Services {
		names = append(names, name)
	}
	return names
}
