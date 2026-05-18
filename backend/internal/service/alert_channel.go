package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
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
	ChannelTypeSlack      ChannelType = "slack"
	ChannelTypeWebhook    ChannelType = "webhook"
	ChannelTypeEmail      ChannelType = "email"
	ChannelTypeDingTalk   ChannelType = "dingtalk"
	ChannelTypeWeChatWork ChannelType = "wechat_work"
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

// EmailConfig Email 配置
type EmailConfig struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	FromAddress  string   `json:"from_address"`
	ToAddresses  []string `json:"to_addresses"`
	UseTLS       bool     `json:"use_tls"`
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret,omitempty"`
	AtMobiles  []string `json:"at_mobiles,omitempty"`
	IsAtAll    bool   `json:"is_at_all"`
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

// GetSeverityColor 获取严重程度对应的颜色
func (b *BaseChannel) GetSeverityColor(severity string) string {
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
	case ChannelTypeEmail:
		return NewEmailChannel(config)
	case ChannelTypeDingTalk:
		return NewDingTalkChannel(config)
	case ChannelTypeWeChatWork:
		return NewWeChatWorkChannel(config)
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// EmailChannel Email 告警渠道
type EmailChannel struct {
	BaseChannel
	emailConfig *EmailConfig
}

// NewEmailChannel 创建 Email 渠道
func NewEmailChannel(config map[string]interface{}) (*EmailChannel, error) {
	emailConfig, err := ParseEmailConfig(config)
	if err != nil {
		return nil, err
	}
	return &EmailChannel{
		BaseChannel: NewBaseChannel(config),
		emailConfig: emailConfig,
	}, nil
}

// Name 渠道名称
func (e *EmailChannel) Name() string {
	return "email"
}

// ValidateConfig 验证配置
func (e *EmailChannel) ValidateConfig() error {
	_, err := ParseEmailConfig(e.GetConfig())
	return err
}

// Send 发送告警
func (e *EmailChannel) Send(msg AlertMessage) error {
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(msg.Severity), msg.Title)
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
<h2 style="color: %s;">%s</h2>
<p><strong>严重级别:</strong> %s</p>
<p><strong>事件ID:</strong> %s</p>
<p><strong>时间:</strong> %s</p>
<hr/>
<h3>详细信息:</h3>
<p>%s</p>
<hr/>
<h3>上下文:</h3>
<pre>%s</pre>
</body>
</html>
`,
		e.GetSeverityColor(msg.Severity),
		msg.Title,
		strings.ToUpper(msg.Severity),
		msg.EventID,
		msg.Timestamp.Format("2006-01-02 15:04:05"),
		msg.Message,
		e.formatContext(msg.Context),
	)

	return e.sendEmail(subject, body)
}

func (e *EmailChannel) formatContext(ctx map[string]interface{}) string {
	if ctx == nil {
		return "无"
	}
	var result strings.Builder
	for k, v := range ctx {
		result.WriteString(fmt.Sprintf("%s: %v\n", k, v))
	}
	return result.String()
}

func (e *EmailChannel) sendEmail(subject, body string) error {
	auth := smtp.PlainAuth("", e.emailConfig.Username, e.emailConfig.Password, e.emailConfig.SMTPHost)
	to := e.emailConfig.ToAddresses

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", e.emailConfig.FromAddress, strings.Join(to, ","), subject, body)

	var addr string
	if e.emailConfig.UseTLS {
		addr = fmt.Sprintf("%s:%d", e.emailConfig.SMTPHost, e.emailConfig.SMTPPort)
		err := smtp.SendMail(addr, auth, e.emailConfig.FromAddress, to, []byte(msg))
		if err != nil {
			return err
		}
	} else {
		addr = fmt.Sprintf("%s:%d", e.emailConfig.SMTPHost, e.emailConfig.SMTPPort)
		err := smtp.SendMail(addr, auth, e.emailConfig.FromAddress, to, []byte(msg))
		if err != nil {
			return err
		}
	}
	return nil
}

// ParseEmailConfig 解析 Email 配置
func ParseEmailConfig(config map[string]interface{}) (*EmailConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var emailConfig EmailConfig
	err = json.Unmarshal(data, &emailConfig)
	if err != nil {
		return nil, err
	}
	if emailConfig.SMTPHost == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if emailConfig.SMTPPort == 0 {
		emailConfig.SMTPPort = 587
	}
	if emailConfig.Username == "" {
		return nil, fmt.Errorf("SMTP username is required")
	}
	if emailConfig.FromAddress == "" {
		return nil, fmt.Errorf("from address is required")
	}
	if len(emailConfig.ToAddresses) == 0 {
		return nil, fmt.Errorf("at least one to address is required")
	}
	return &emailConfig, nil
}

// DingTalkChannel 钉钉告警渠道
type DingTalkChannel struct {
	BaseChannel
	dingTalkConfig *DingTalkConfig
}

// NewDingTalkChannel 创建钉钉渠道
func NewDingTalkChannel(config map[string]interface{}) (*DingTalkChannel, error) {
	dingTalkConfig, err := ParseDingTalkConfig(config)
	if err != nil {
		return nil, err
	}
	return &DingTalkChannel{
		BaseChannel:    NewBaseChannel(config),
		dingTalkConfig: dingTalkConfig,
	}, nil
}

// Name 渠道名称
func (d *DingTalkChannel) Name() string {
	return "dingtalk"
}

// ValidateConfig 验证配置
func (d *DingTalkChannel) ValidateConfig() error {
	_, err := ParseDingTalkConfig(d.GetConfig())
	return err
}

// Send 发送告警
func (d *DingTalkChannel) Send(msg AlertMessage) error {
	payload := d.buildDingTalkPayload(msg)
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(d.dingTalkConfig.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dingtalk API returned status %d", resp.StatusCode)
	}
	return nil
}

func (d *DingTalkChannel) buildDingTalkPayload(msg AlertMessage) map[string]interface{} {
	content := fmt.Sprintf("### [%s] %s\n\n", strings.ToUpper(msg.Severity), msg.Title)
	content += fmt.Sprintf("**严重级别:** %s\n\n", strings.ToUpper(msg.Severity))
	content += fmt.Sprintf("**事件ID:** %s\n\n", msg.EventID)
	content += fmt.Sprintf("**时间:** %s\n\n", msg.Timestamp.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("**详细信息:**\n%s\n\n", msg.Message)
	if len(msg.Context) > 0 {
		content += "**上下文:**\n"
		for k, v := range msg.Context {
			content += fmt.Sprintf("- %s: %v\n", k, v)
		}
	}

	return map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": msg.Title,
			"text":  content,
		},
		"at": map[string]interface{}{
			"atMobiles": d.dingTalkConfig.AtMobiles,
			"isAtAll":    d.dingTalkConfig.IsAtAll,
		},
	}
}

// ParseDingTalkConfig 解析钉钉配置
func ParseDingTalkConfig(config map[string]interface{}) (*DingTalkConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var dingTalkConfig DingTalkConfig
	err = json.Unmarshal(data, &dingTalkConfig)
	if err != nil {
		return nil, err
	}
	if dingTalkConfig.WebhookURL == "" {
		return nil, fmt.Errorf("dingtalk webhook URL is required")
	}
	return &dingTalkConfig, nil
}
