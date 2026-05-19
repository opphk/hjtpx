package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type NotificationChannel string

const (
	ChannelSlack      NotificationChannel = "slack"
	ChannelWeChatWork NotificationChannel = "wechat_work"
	ChannelEmail      NotificationChannel = "email"
	ChannelWebhook    NotificationChannel = "webhook"
	ChannelSMS        NotificationChannel = "sms"
)

type NotificationSeverity string

const (
	SeverityInfo     NotificationSeverity = "info"
	SeverityWarning  NotificationSeverity = "warning"
	SeverityError    NotificationSeverity = "error"
	SeverityCritical NotificationSeverity = "critical"
)

type NotificationMessage struct {
	ID        string                 `json:"id"`
	Channel   NotificationChannel    `json:"channel"`
	Severity  NotificationSeverity  `json:"severity"`
	Title     string                `json:"title"`
	Content   string                `json:"content"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Recipient string                `json:"recipient,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
}

type SlackChannelConfig struct {
	WebhookURL  string   `json:"webhook_url"`
	Channel     string   `json:"channel,omitempty"`
	Username    string   `json:"username,omitempty"`
	IconEmoji   string   `json:"icon_emoji,omitempty"`
	AtMobiles   []string `json:"at_mobiles,omitempty"`
	IsAtAll     bool     `json:"is_at_all"`
}

type SlackService struct {
	configs map[string]*SlackChannelConfig
	mu      sync.RWMutex
	client  *http.Client
}

func NewSlackService() *SlackService {
	return &SlackService{
		configs: make(map[string]*SlackChannelConfig),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *SlackService) RegisterChannel(name string, config *SlackChannelConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[name] = config
}

func (s *SlackService) GetChannel(name string) (*SlackChannelConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config, ok := s.configs[name]
	return config, ok
}

func (s *SlackService) ListChannels() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.configs))
	for name := range s.configs {
		names = append(names, name)
	}
	return names
}

func (s *SlackService) Send(name string, msg NotificationMessage) error {
	s.mu.RLock()
	config, ok := s.configs[name]
	s.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("slack channel %s not found", name)
	}
	
	payload := s.buildSlackPayload(msg, config)
	
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}
	
	resp, err := s.client.Post(config.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (s *SlackService) buildSlackPayload(msg NotificationMessage, config *SlackChannelConfig) map[string]interface{} {
	attachment := map[string]interface{}{
		"color": s.getSeverityColor(msg.Severity),
		"title": msg.Title,
		"text":  msg.Content,
		"fields": []map[string]interface{}{
			{
				"title": "Severity",
				"value": string(msg.Severity),
				"short": true,
			},
			{
				"title": "Time",
				"value": msg.CreatedAt.Format("2006-01-02 15:04:05"),
				"short": true,
			},
		},
		"footer":     "HJTPX Notification System",
		"footer_icon": "https://platform.slack-edge.com/img/default_application_icon.png",
		"ts":         msg.CreatedAt.Unix(),
	}
	
	if len(msg.Data) > 0 {
		fields := make([]map[string]interface{}, 0, len(msg.Data))
		for key, value := range msg.Data {
			fields = append(fields, map[string]interface{}{
				"title": key,
				"value": fmt.Sprintf("%v", value),
				"short": true,
			})
		}
		if len(fields) > 0 {
			attachment["fields"] = append(attachment["fields"].([]map[string]interface{}), fields...)
		}
	}
	
	slackMsg := map[string]interface{}{
		"attachments": []map[string]interface{}{attachment},
	}
	
	if config.Channel != "" {
		slackMsg["channel"] = config.Channel
	}
	
	if config.Username != "" {
		slackMsg["username"] = config.Username
	}
	
	if config.IconEmoji != "" {
		slackMsg["icon_emoji"] = config.IconEmoji
	}
	
	if msg.Severity == SeverityCritical || config.IsAtAll {
		slackMsg["text"] = "<!channel>"
	}
	
	return slackMsg
}

func (s *SlackService) getSeverityColor(severity NotificationSeverity) string {
	switch severity {
	case SeverityInfo:
		return "#36a64f"
	case SeverityWarning:
		return "#ff9800"
	case SeverityError:
		return "#f44336"
	case SeverityCritical:
		return "#b71c1c"
	default:
		return "#36a64f"
	}
}

func (s *SlackService) SendBlockKit(name string, blocks []map[string]interface{}) error {
	s.mu.RLock()
	config, ok := s.configs[name]
	s.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("slack channel %s not found", name)
	}
	
	payload := map[string]interface{}{
		"blocks": blocks,
	}
	
	if config.Channel != "" {
		payload["channel"] = config.Channel
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal block kit payload: %w", err)
	}
	
	resp, err := s.client.Post(config.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send block kit message: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (s *SlackService) BuildVerificationAlertBlock(msg NotificationMessage) []map[string]interface{} {
	severityEmoji := map[NotificationSeverity]string{
		SeverityInfo:     ":information_source:",
		SeverityWarning:  ":warning:",
		SeverityError:    ":x:",
		SeverityCritical: ":fire:",
	}
	
	emoji, ok := severityEmoji[msg.Severity]
	if !ok {
		emoji = ":bell:"
	}
	
	return []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": fmt.Sprintf("%s %s", emoji, msg.Title),
				"emoji": true,
			},
		},
		{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": msg.Content,
			},
		},
		{
			"type": "divider",
		},
		{
			"type": "context",
			"elements": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Severity:* %s | *Time:* %s | *ID:* %s",
						strings.ToUpper(string(msg.Severity)),
						msg.CreatedAt.Format("2006-01-02 15:04:05"),
						msg.ID),
				},
			},
		},
	}
}

func (s *SlackService) BuildDataBlock(fields map[string]interface{}) []map[string]interface{} {
	blocks := make([]map[string]interface{}, 0)
	
	for key, value := range fields {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"fields": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s*", key),
				},
				{
					"type": "plain_text",
					"text": fmt.Sprintf("%v", value),
				},
			},
		})
	}
	
	return blocks
}

type WeChatWorkNotificationService struct {
	slackService *SlackService
}

func NewWeChatWorkNotificationService() *WeChatWorkNotificationService {
	return &WeChatWorkNotificationService{
		slackService: NewSlackService(),
	}
}

func (s *WeChatWorkNotificationService) RegisterChannel(name string, config *SlackChannelConfig) {
	s.slackService.RegisterChannel(name, config)
}

func (s *WeChatWorkNotificationService) Send(name string, msg NotificationMessage) error {
	return s.slackService.Send(name, msg)
}

func (s *WeChatWorkNotificationService) SendToMultiple(channels []string, msg NotificationMessage) []error {
	errors := make([]error, 0)
	
	for _, channel := range channels {
		if err := s.Send(channel, msg); err != nil {
			errors = append(errors, err)
		}
	}
	
	return errors
}

type NotificationManager struct {
	slackService *SlackService
	wechatService *WeChatWorkNotificationService
	channels     map[string]NotificationChannel
	mu           sync.RWMutex
}

func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		slackService:  NewSlackService(),
		wechatService: NewWeChatWorkNotificationService(),
		channels:     make(map[string]NotificationChannel),
	}
}

func (m *NotificationManager) RegisterChannel(name string, channel NotificationChannel, config interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.channels[name] = channel
	
	switch channel {
	case ChannelSlack:
		if cfg, ok := config.(*SlackChannelConfig); ok {
			m.slackService.RegisterChannel(name, cfg)
		}
	case ChannelWeChatWork:
		if cfg, ok := config.(*SlackChannelConfig); ok {
			m.wechatService.RegisterChannel(name, cfg)
		}
	}
	
	return nil
}

func (m *NotificationManager) SendNotification(channelName string, msg NotificationMessage) error {
	m.mu.RLock()
	channel, ok := m.channels[channelName]
	m.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("channel %s not registered", channelName)
	}
	
	switch channel {
	case ChannelSlack:
		return m.slackService.Send(channelName, msg)
	case ChannelWeChatWork:
		return m.wechatService.Send(channelName, msg)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel)
	}
}

func (m *NotificationManager) BroadcastNotification(msg NotificationMessage) []error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	errors := make([]error, 0)
	
	for channelName, channel := range m.channels {
		switch channel {
		case ChannelSlack:
			if err := m.slackService.Send(channelName, msg); err != nil {
				errors = append(errors, fmt.Errorf("slack/%s: %w", channelName, err))
			}
		case ChannelWeChatWork:
			if err := m.wechatService.Send(channelName, msg); err != nil {
				errors = append(errors, fmt.Errorf("wechat/%s: %w", channelName, err))
			}
		}
	}
	
	return errors
}

func (m *NotificationManager) ListChannels() map[string]NotificationChannel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]NotificationChannel)
	for k, v := range m.channels {
		result[k] = v
	}
	return result
}

func (m *NotificationManager) GetSlackService() *SlackService {
	return m.slackService
}

func (m *NotificationManager) GetWeChatService() *WeChatWorkNotificationService {
	return m.wechatService
}

var (
	defaultNotificationManager *NotificationManager
	notificationManagerOnce    sync.Once
)

func GetNotificationManager() *NotificationManager {
	notificationManagerOnce.Do(func() {
		defaultNotificationManager = NewNotificationManager()
	})
	return defaultNotificationManager
}

func SendVerificationAlert(severity NotificationSeverity, title, content string, data map[string]interface{}) {
	msg := NotificationMessage{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Channel:   ChannelSlack,
		Severity:  severity,
		Title:     title,
		Content:   content,
		Data:      data,
		CreatedAt: time.Now(),
	}
	
	manager := GetNotificationManager()
	channels := manager.ListChannels()
	
	for channelName := range channels {
		go func(ch string) {
			manager.SendNotification(ch, msg)
		}(channelName)
	}
}

func SendSecurityAlert(severity NotificationSeverity, alertType string, details map[string]interface{}) {
	content := fmt.Sprintf("*Security Alert: %s*\n\n", alertType)
	
	for key, value := range details {
		content += fmt.Sprintf("• *%s*: `%v`\n", key, value)
	}
	
	SendVerificationAlert(severity, "Security Alert", content, details)
}

func SendSystemNotification(severity NotificationSeverity, title, content string) {
	SendVerificationAlert(severity, title, content, nil)
}

func SendVerificationStats(stats map[string]interface{}) {
	manager := GetNotificationManager()
	
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": ":bar_chart: Verification Statistics Update",
				"emoji": true,
			},
		},
	}
	
	for key, value := range stats {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"fields": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s*", key),
				},
				{
					"type": "plain_text",
					"text": fmt.Sprintf("%v", value),
				},
			},
		})
	}
	
	msg := NotificationMessage{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Severity:  SeverityInfo,
		Title:     "Verification Statistics",
		Content:   "Statistics update",
		Data:      stats,
		CreatedAt: time.Now(),
	}
	
	slackService := manager.GetSlackService()
	channels := slackService.ListChannels()
	
	for _, channel := range channels {
		go func(ch string) {
			slackService.SendBlockKit(ch, blocks)
			_ = msg
		}(channel)
	}
}
