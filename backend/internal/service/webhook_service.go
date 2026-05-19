package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
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

type WebhookEvent string

const (
	EventVerificationSuccess  WebhookEvent = "verification.success"
	EventVerificationFailed   WebhookEvent = "verification.failed"
	EventVerificationBlocked  WebhookEvent = "verification.blocked"
	EventRiskDetected         WebhookEvent = "risk.detected"
	EventAttackDetected       WebhookEvent = "attack.detected"
	EventUserCreated          WebhookEvent = "user.created"
	EventUserUpdated          WebhookEvent = "user.updated"
	EventUserDeleted          WebhookEvent = "user.deleted"
	EventApplicationCreated   WebhookEvent = "application.created"
	EventApplicationUpdated   WebhookEvent = "application.updated"
	EventConfigChanged        WebhookEvent = "config.changed"
	EventSystemAlert          WebhookEvent = "system.alert"
	EventAll                  WebhookEvent = "*"
)

type WebhookDeliveryStatus string

const (
	DeliveryStatusPending   WebhookDeliveryStatus = "pending"
	DeliveryStatusSuccess   WebhookDeliveryStatus = "success"
	DeliveryStatusFailed    WebhookDeliveryStatus = "failed"
	DeliveryStatusRetrying  WebhookDeliveryStatus = "retrying"
	DeliveryStatusCancelled WebhookDeliveryStatus = "cancelled"
)

type Webhook struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Name           string    `json:"name" binding:"required"`
	URL            string    `json:"url" binding:"required"`
	Secret         string    `json:"secret,omitempty"`
	Events         string    `json:"events"`
	Headers        string    `json:"headers,omitempty"`
	IsEnabled      bool      `json:"is_enabled" gorm:"default:true"`
	RetryEnabled   bool      `json:"retry_enabled" gorm:"default:true"`
	RetryCount     int       `json:"retry_count" gorm:"default:3"`
	RetryDelay     int       `json:"retry_delay" gorm:"default:60"`
	Timeout        int       `json:"timeout" gorm:"default:30"`
	ContentType    string    `json:"content_type" gorm:"default:'application/json'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WebhookDelivery struct {
	ID           uint                `json:"id" gorm:"primaryKey"`
	WebhookID    uint                `json:"webhook_id"`
	Event        string              `json:"event"`
	Payload      string              `json:"payload"`
	Status       WebhookDeliveryStatus `json:"status"`
	Attempts     int                 `json:"attempts"`
	LastAttempt  *time.Time          `json:"last_attempt,omitempty"`
	NextRetry    *time.Time          `json:"next_retry,omitempty"`
	ResponseCode int                 `json:"response_code,omitempty"`
	ResponseBody string              `json:"response_body,omitempty"`
	Error        string              `json:"error,omitempty"`
	CreatedAt    time.Time
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
}

type WebhookPayload struct {
	Event       WebhookEvent `json:"event"`
	Timestamp   time.Time   `json:"timestamp"`
	WebhookID   string      `json:"webhook_id,omitempty"`
	Sequence    uint64      `json:"sequence"`
	Data        interface{} `json:"data"`
}

var (
	webhookSequence uint64
	webhookQueues   = make(map[uint]*WebhookEventQueue)
	webhookMu       sync.RWMutex
)

type WebhookEventQueue struct {
	webhookID uint
	events    []WebhookDelivery
	mu        sync.Mutex
	cond      *sync.Cond
	stopCh    chan struct{}
}

func NewWebhookEventQueue(webhookID uint) *WebhookEventQueue {
	q := &WebhookEventQueue{
		webhookID: webhookID,
		events:    make([]WebhookDelivery, 0),
		stopCh:    make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *WebhookEventQueue) Push(event WebhookDelivery) {
	q.mu.Lock()
	q.events = append(q.events, event)
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *WebhookEventQueue) Pop() (WebhookDelivery, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	for len(q.events) == 0 {
		select {
		case <-q.stopCh:
			return WebhookDelivery{}, false
		default:
			q.cond.Wait()
		}
	}
	
	event := q.events[0]
	q.events = q.events[1:]
	return event, true
}

func (q *WebhookEventQueue) Stop() {
	close(q.stopCh)
}

type WebhookManager struct {
	webhooks     map[uint]*Webhook
	queues       map[uint]*WebhookEventQueue
	httpClient   *http.Client
	deliveryChan chan WebhookDelivery
	stopCh       chan struct{}
	mu           sync.RWMutex
}

func NewWebhookManager() *WebhookManager {
	manager := &WebhookManager{
		webhooks:     make(map[uint]*Webhook),
		queues:       make(map[uint]*WebhookEventQueue),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		deliveryChan: make(chan WebhookDelivery, 1000),
		stopCh:       make(chan struct{}),
	}
	
	go manager.processDeliveries()
	
	return manager
}

func (m *WebhookManager) RegisterWebhook(webhook *Webhook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.webhooks[webhook.ID] = webhook
	
	if _, exists := m.queues[webhook.ID]; !exists {
		m.queues[webhook.ID] = NewWebhookEventQueue(webhook.ID)
		go m.processWebhookQueue(webhook.ID)
	}
}

func (m *WebhookManager) UnregisterWebhook(webhookID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if queue, exists := m.queues[webhookID]; exists {
		queue.Stop()
		delete(m.queues, webhookID)
	}
	delete(m.webhooks, webhookID)
}

func (m *WebhookManager) GetWebhook(webhookID uint) (*Webhook, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	webhook, ok := m.webhooks[webhookID]
	return webhook, ok
}

func (m *WebhookManager) ListWebhooks() []*Webhook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	webhooks := make([]*Webhook, 0, len(m.webhooks))
	for _, webhook := range m.webhooks {
		webhooks = append(webhooks, webhook)
	}
	return webhooks
}

func (m *WebhookManager) TriggerEvent(event WebhookEvent, data interface{}) {
	sequence := atomic.AddUint64(&webhookSequence, 1)
	
	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now(),
		Sequence:  sequence,
		Data:      data,
	}
	
	payloadJSON, _ := json.Marshal(payload)
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, webhook := range m.webhooks {
		if !webhook.IsEnabled {
			continue
		}
		
		if !m.shouldDeliverEvent(webhook, event) {
			continue
		}
		
		delivery := WebhookDelivery{
			WebhookID: webhook.ID,
			Event:     string(event),
			Payload:   string(payloadJSON),
			Status:    DeliveryStatusPending,
		}
		
		if queue, exists := m.queues[webhook.ID]; exists {
			queue.Push(delivery)
		}
	}
}

func (m *WebhookManager) shouldDeliverEvent(webhook *Webhook, event WebhookEvent) bool {
	if webhook.Events == "" || webhook.Events == "*" {
		return true
	}
	
	events := parseWebhookEvents(webhook.Events)
	for _, e := range events {
		if e == string(event) || e == "*" {
			return true
		}
	}
	return false
}

func parseWebhookEvents(eventsStr string) []string {
	if eventsStr == "" {
		return []string{}
	}
	
	events := make([]string, 0)
	parts := splitAndTrim(eventsStr, ",")
	for _, part := range parts {
		events = append(events, strings.TrimSpace(part))
	}
	return events
}

func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	
	result := make([]string, 0)
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			if start < i {
				result = append(result, s[start:i])
			}
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func (m *WebhookManager) processWebhookQueue(webhookID uint) {
	queue, ok := m.queues[webhookID]
	if !ok {
		return
	}
	
	for {
		delivery, ok := queue.Pop()
		if !ok {
			return
		}
		
		m.deliveryChan <- delivery
	}
}

func (m *WebhookManager) processDeliveries() {
	for {
		select {
		case <-m.stopCh:
			return
		case delivery := <-m.deliveryChan:
			go m.deliverWebhook(delivery)
		}
	}
}

func (m *WebhookManager) deliverWebhook(delivery WebhookDelivery) {
	m.mu.RLock()
	webhook, ok := m.webhooks[delivery.WebhookID]
	m.mu.RUnlock()
	
	if !ok || !webhook.IsEnabled {
		return
	}
	
	var lastErr error
	maxAttempts := 1
	if webhook.RetryEnabled {
		maxAttempts = webhook.RetryCount + 1
	}
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		delivery.Attempts = attempt
		now := time.Now()
		delivery.LastAttempt = &now
		
		if attempt > 1 {
			delivery.Status = DeliveryStatusRetrying
		}
		
		if err := m.sendWebhook(webhook, delivery); err != nil {
			lastErr = err
			delivery.Error = err.Error()
			
			if attempt < maxAttempts {
				delay := time.Duration(webhook.RetryDelay) * time.Second * time.Duration(math.Pow(2, float64(attempt-1)))
				nextRetry := time.Now().Add(delay)
				delivery.NextRetry = &nextRetry
				
				time.Sleep(delay)
				continue
			}
		} else {
			delivery.Status = DeliveryStatusSuccess
			completed := time.Now()
			delivery.CompletedAt = &completed
			break
		}
	}
	
	if delivery.Status != DeliveryStatusSuccess {
		delivery.Status = DeliveryStatusFailed
	}
	
	m.saveDelivery(&delivery)
	
	m.recordWebhookMetrics(webhook.ID, delivery)
}

func (m *WebhookManager) sendWebhook(webhook *Webhook, delivery WebhookDelivery) error {
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBufferString(delivery.Payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", webhook.ContentType)
	req.Header.Set("User-Agent", "HJTPX-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", delivery.Event)
	req.Header.Set("X-Webhook-Delivery", fmt.Sprintf("%d", delivery.ID))
	req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	
	if webhook.Headers != "" {
		headers := parseWebhookHeaders(webhook.Headers)
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
	
	if webhook.Secret != "" {
		signature := m.signPayload(webhook.Secret, delivery.Payload, delivery.Event)
		req.Header.Set("X-Webhook-Signature", signature)
	}
	
	client := &http.Client{
		Timeout: time.Duration(webhook.Timeout) * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	delivery.ResponseCode = resp.StatusCode
	
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	delivery.ResponseBody = string(body)
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

func (m *WebhookManager) signPayload(secret, payload, event string) string {
	timestamp := time.Now().Unix()
	signData := fmt.Sprintf("%d.%s.%s", timestamp, event, payload)
	
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signData))
	
	return fmt.Sprintf("v1=%s", hex.EncodeToString(h.Sum(nil)))
}

func parseWebhookHeaders(headersStr string) map[string]string {
	headers := make(map[string]string)
	if headersStr == "" {
		return headers
	}
	
	parts := splitAndTrim(headersStr, ";")
	for _, part := range parts {
		kv := splitAndTrim(part, ":")
		if len(kv) == 2 {
			headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return headers
}

func (m *WebhookManager) saveDelivery(delivery *WebhookDelivery) {
	deliveryRecord := WebhookDelivery{
		ID:           delivery.ID,
		WebhookID:    delivery.WebhookID,
		Event:        delivery.Event,
		Payload:      delivery.Payload,
		Status:       delivery.Status,
		Attempts:     delivery.Attempts,
		LastAttempt:  delivery.LastAttempt,
		NextRetry:    delivery.NextRetry,
		ResponseCode: delivery.ResponseCode,
		ResponseBody: delivery.ResponseBody,
		Error:        delivery.Error,
		CompletedAt:  delivery.CompletedAt,
	}
	
	database.DB.Save(&deliveryRecord)
}

func (m *WebhookManager) recordWebhookMetrics(webhookID uint, delivery WebhookDelivery) {
	if redisClient := redis.GetClient(); redisClient != nil {
		prefix := fmt.Sprintf("webhook:metrics:%d", webhookID)
		
		if delivery.Status == DeliveryStatusSuccess {
			redisClient.Incr(prefix + ":success")
		} else {
			redisClient.Incr(prefix + ":failed")
		}
		redisClient.Incr(prefix + ":total")
		
		redisClient.Expire(prefix+":success", 24*time.Hour)
		redisClient.Expire(prefix+":failed", 24*time.Hour)
		redisClient.Expire(prefix+":total", 24*time.Hour)
	}
}

func (m *WebhookManager) Stop() {
	close(m.stopCh)
}

var defaultWebhookManager *WebhookManager
var webhookManagerOnce sync.Once

func GetWebhookManager() *WebhookManager {
	webhookManagerOnce.Do(func() {
		defaultWebhookManager = NewWebhookManager()
	})
	return defaultWebhookManager
}

func CreateWebhook(webhook *Webhook) error {
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()
	
	if err := database.DB.Create(webhook).Error; err != nil {
		return err
	}
	
	manager := GetWebhookManager()
	manager.RegisterWebhook(webhook)
	
	return nil
}

func UpdateWebhook(webhook *Webhook) error {
	webhook.UpdatedAt = time.Now()
	
	if err := database.DB.Save(webhook).Error; err != nil {
		return err
	}
	
	manager := GetWebhookManager()
	manager.UnregisterWebhook(webhook.ID)
	if webhook.IsEnabled {
		manager.RegisterWebhook(webhook)
	}
	
	return nil
}

func DeleteWebhook(id uint) error {
	manager := GetWebhookManager()
	manager.UnregisterWebhook(id)
	
	return database.DB.Delete(&Webhook{}, id).Error
}

func GetWebhookByID(id uint) (*Webhook, error) {
	var webhook Webhook
	if err := database.DB.First(&webhook, id).Error; err != nil {
		return nil, err
	}
	return &webhook, nil
}

func ListWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	if err := database.DB.Order("created_at DESC").Find(&webhooks).Error; err != nil {
		return nil, err
	}
	return webhooks, nil
}

func GetWebhookDeliveries(webhookID uint, limit int) ([]WebhookDelivery, error) {
	var deliveries []WebhookDelivery
	query := database.DB.Where("webhook_id = ?", webhookID)
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	if err := query.Order("created_at DESC").Find(&deliveries).Error; err != nil {
		return nil, err
	}
	return deliveries, nil
}

func GetWebhookDeliveryStats(webhookID uint) (map[string]interface{}, error) {
	var total, success, failed, pending, retrying int64
	
	database.DB.Model(&WebhookDelivery{}).Where("webhook_id = ?", webhookID).Count(&total)
	database.DB.Model(&WebhookDelivery{}).Where("webhook_id = ? AND status = ?", webhookID, DeliveryStatusSuccess).Count(&success)
	database.DB.Model(&WebhookDelivery{}).Where("webhook_id = ? AND status = ?", webhookID, DeliveryStatusFailed).Count(&failed)
	database.DB.Model(&WebhookDelivery{}).Where("webhook_id = ? AND status = ?", webhookID, DeliveryStatusPending).Count(&pending)
	database.DB.Model(&WebhookDelivery{}).Where("webhook_id = ? AND status = ?", webhookID, DeliveryStatusRetrying).Count(&retrying)
	
	return map[string]interface{}{
		"total":     total,
		"success":   success,
		"failed":    failed,
		"pending":   pending,
		"retrying":  retrying,
		"success_rate": func() float64 {
			if total == 0 {
				return 0
			}
			return float64(success) / float64(total) * 100
		}(),
	}, nil
}

func RetryWebhookDelivery(deliveryID uint) error {
	var delivery WebhookDelivery
	if err := database.DB.First(&delivery, deliveryID).Error; err != nil {
		return err
	}
	
	if delivery.Status != DeliveryStatusFailed && delivery.Status != DeliveryStatusRetrying {
		return fmt.Errorf("cannot retry delivery with status: %s", delivery.Status)
	}
	
	delivery.Status = DeliveryStatusPending
	delivery.Attempts = 0
	delivery.NextRetry = nil
	delivery.Error = ""
	
	database.DB.Save(&delivery)
	
	manager := GetWebhookManager()
	if webhook, ok := manager.GetWebhook(delivery.WebhookID); ok {
		manager.deliveryChan <- delivery
		_ = webhook
	}
	
	return nil
}

func TriggerWebhookEvent(event WebhookEvent, data interface{}) {
	manager := GetWebhookManager()
	manager.TriggerEvent(event, data)
}

func TestWebhook(webhookID uint) (map[string]interface{}, error) {
	webhook, err := GetWebhookByID(webhookID)
	if err != nil {
		return nil, err
	}
	
	testPayload := WebhookPayload{
		Event:     EventSystemAlert,
		Timestamp: time.Now(),
		WebhookID: fmt.Sprintf("%d", webhookID),
		Sequence:  atomic.AddUint64(&webhookSequence, 1),
		Data: map[string]interface{}{
			"type":      "test",
			"message":   "This is a test webhook delivery",
			"webhook_id": webhookID,
		},
	}
	
	payloadJSON, _ := json.Marshal(testPayload)
	
	testDelivery := WebhookDelivery{
		WebhookID: webhookID,
		Event:     string(EventSystemAlert),
		Payload:   string(payloadJSON),
		Status:    DeliveryStatusPending,
	}
	
	manager := GetWebhookManager()
	
	if err := manager.sendWebhook(webhook, testDelivery); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"payload": testPayload,
		}, err
	}
	
	return map[string]interface{}{
		"success":      true,
		"response_code": testDelivery.ResponseCode,
		"response_body": testDelivery.ResponseBody,
		"payload":      testPayload,
	}, nil
}

func InitWebhookManager() {
	manager := GetWebhookManager()
	
	webhooks, err := ListWebhooks()
	if err != nil {
		return
	}
	
	for i := range webhooks {
		if webhooks[i].IsEnabled {
			manager.RegisterWebhook(&webhooks[i])
		}
	}
}

func (m *WebhookManager) DeliveriesForWebhooks(webhookIDs []uint) ([]WebhookDelivery, error) {
	var deliveries []WebhookDelivery
	if len(webhookIDs) == 0 {
		return deliveries, nil
	}
	
	if err := database.DB.Where("webhook_id IN ?", webhookIDs).
		Order("created_at DESC").
		Limit(100).
		Find(&deliveries).Error; err != nil {
		return nil, err
	}
	
	return deliveries, nil
}

func CancelPendingDeliveries(webhookID uint) error {
	return database.DB.Model(&WebhookDelivery{}).
		Where("webhook_id = ? AND status IN ?", webhookID, []WebhookDeliveryStatus{DeliveryStatusPending, DeliveryStatusRetrying}).
		Updates(map[string]interface{}{
			"status": DeliveryStatusCancelled,
		}).Error
}
