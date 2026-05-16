package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AlertMessage 告警消息
type AlertMessage struct {
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	EventID   string                 `json:"event_id"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// AlertChannel 告警渠道接口
type AlertChannel interface {
	Name() string
	Send(msg AlertMessage) error
	ValidateConfig() error
}

// ChannelType 渠道类型
type ChannelType string

const (
	// ChannelTypeSlack Slack 渠道类型
	ChannelTypeSlack ChannelType = "slack"
	// ChannelTypeWebhook Webhook 渠道类型
	ChannelTypeWebhook ChannelType = "webhook"
)

// SlackConfig Slack 配置
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
}

// BaseChannel 基础渠道实现
type BaseChannel struct {
	config map[string]interface{}
}

// NewBaseChannel 创建基础渠道
func NewBaseChannel(config map[string]interface{}) BaseChannel {
	return BaseChannel{config: config}
}

// GetConfig 获取配置
func (b *BaseChannel) GetConfig() map[string]interface{} {
	return b.config
}

// ParseSlackConfig 解析 Slack 配置
func ParseSlackConfig(config map[string]interface{}) (*SlackConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var slackConfig SlackConfig
	err = json.Unmarshal(data, &slackConfig)
	if err != nil {
		return nil, err
	}
	if slackConfig.WebhookURL == "" {
		return nil, fmt.Errorf("slack webhook URL is required")
	}
	return &slackConfig, nil
}

// ParseWebhookConfig 解析 Webhook 配置
func ParseWebhookConfig(config map[string]interface{}) (*WebhookConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var webhookConfig WebhookConfig
	err = json.Unmarshal(data, &webhookConfig)
	if err != nil {
		return nil, err
	}
	if webhookConfig.URL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}
	if webhookConfig.Method == "" {
		webhookConfig.Method = "POST"
	}
	return &webhookConfig, nil
}

// SlackChannel Slack 告警渠道
type SlackChannel struct {
	BaseChannel
	slackConfig *SlackConfig
}

// NewSlackChannel 创建 Slack 渠道
func NewSlackChannel(config map[string]interface{}) (*SlackChannel, error) {
	slackConfig, err := ParseSlackConfig(config)
	if err != nil {
		return nil, err
	}
	return &SlackChannel{
		BaseChannel: NewBaseChannel(config),
		slackConfig: slackConfig,
	}, nil
}

// Name 渠道名称
func (s *SlackChannel) Name() string {
	return "slack"
}

// ValidateConfig 验证配置
func (s *SlackChannel) ValidateConfig() error {
	_, err := ParseSlackConfig(s.GetConfig())
	return err
}

// Send 发送告警
func (s *SlackChannel) Send(msg AlertMessage) error {
	payload := s.buildSlackPayload(msg)
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(s.slackConfig.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}
	return nil
}

func (s *SlackChannel) buildSlackPayload(msg AlertMessage) map[string]interface{} {
	color := s.GetSeverityColor(msg.Severity)
	fields := []map[string]interface{}{
		{
			"title": "Severity",
			"value": strings.ToUpper(msg.Severity),
			"short": true,
		},
		{
			"title": "Event ID",
			"value": msg.EventID,
			"short": true,
		},
		{
			"title": "Timestamp",
			"value": msg.Timestamp.Format(time.RFC3339),
			"short": true,
		},
	}

	for k, v := range msg.Context {
		fields = append(fields, map[string]interface{}{
			"title": k,
			"value": fmt.Sprintf("%v", v),
			"short": true,
		})
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  msg.Title,
				"text":   msg.Message,
				"fields": fields,
			},
		},
	}
	if s.slackConfig.Channel != "" {
		payload["channel"] = s.slackConfig.Channel
	}
	if s.slackConfig.Username != "" {
		payload["username"] = s.slackConfig.Username
	}
	if s.slackConfig.IconEmoji != "" {
		payload["icon_emoji"] = s.slackConfig.IconEmoji
	}
	return payload
}

// GetSeverityColor 获取严重程度对应的颜色
func (s *SlackChannel) GetSeverityColor(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "error":
		return "#ff0000"
	case "warning", "warn":
		return "#ffa500"
	case "info":
		return "#36a64f"
	case "debug":
		return "#808080"
	default:
		return "#36a64f"
	}
}

// WebhookChannel Webhook 告警渠道
type WebhookChannel struct {
	BaseChannel
	webhookConfig *WebhookConfig
}

// NewWebhookChannel 创建 Webhook 渠道
func NewWebhookChannel(config map[string]interface{}) (*WebhookChannel, error) {
	webhookConfig, err := ParseWebhookConfig(config)
	if err != nil {
		return nil, err
	}
	return &WebhookChannel{
		BaseChannel:   NewBaseChannel(config),
		webhookConfig: webhookConfig,
	}, nil
}

// Name 渠道名称
func (w *WebhookChannel) Name() string {
	return "webhook"
}

// ValidateConfig 验证配置
func (w *WebhookChannel) ValidateConfig() error {
	_, err := ParseWebhookConfig(w.GetConfig())
	return err
}

// Send 发送告警
func (w *WebhookChannel) Send(msg AlertMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(w.webhookConfig.Method, w.webhookConfig.URL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.webhookConfig.Headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// CreateChannel 创建告警渠道
func CreateChannel(channelType string, config map[string]interface{}) (AlertChannel, error) {
	switch ChannelType(strings.ToLower(channelType)) {
	case ChannelTypeSlack:
		return NewSlackChannel(config)
	case ChannelTypeWebhook:
		return NewWebhookChannel(config)
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}
