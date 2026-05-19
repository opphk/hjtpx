package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type AlertMessage struct {
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	EventID   string                 `json:"event_id"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type AlertChannel interface {
	Name() string
	Send(msg AlertMessage) error
	ValidateConfig() error
}

type ChannelType string

const (
	ChannelTypeSlack    ChannelType = "slack"
	ChannelTypeWebhook  ChannelType = "webhook"
	ChannelTypeEmail    ChannelType = "email"
	ChannelTypeDingTalk ChannelType = "dingtalk"
)

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
}

type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
}

func ParseSlackConfig(config map[string]interface{}) (*SlackConfig, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required")
	}

	return &SlackConfig{
		WebhookURL: webhookURL,
		Channel:    getConfigString(config, "channel"),
		Username:   getConfigString(config, "username"),
		IconEmoji:  getConfigString(config, "icon_emoji"),
	}, nil
}

func ParseWebhookConfig(config map[string]interface{}) (*WebhookConfig, error) {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	method := getConfigString(config, "method")
	if method == "" {
		method = "POST"
	}

	return &WebhookConfig{
		URL:    url,
		Method: strings.ToUpper(method),
	}, nil
}

func getConfigString(config map[string]interface{}, key string) string {
	if v, ok := config[key].(string); ok {
		return v
	}
	return ""
}

type SlackChannel struct {
	config *SlackConfig
}

func NewSlackChannel(config *SlackConfig) *SlackChannel {
	return &SlackChannel{config: config}
}

func (c *SlackChannel) Name() string {
	return "slack"
}

func (c *SlackChannel) Send(msg AlertMessage) error {
	payload := map[string]interface{}{
		"text": fmt.Sprintf("*[%s]* %s\n%s", msg.Severity, msg.Title, msg.Message),
	}

	if c.config.Channel != "" {
		payload["channel"] = c.config.Channel
	}
	if c.config.Username != "" {
		payload["username"] = c.config.Username
	}
	if c.config.IconEmoji != "" {
		payload["icon_emoji"] = c.config.IconEmoji
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(c.config.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *SlackChannel) ValidateConfig() error {
	if c.config.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}
	return nil
}

type WebhookChannel struct {
	config *WebhookConfig
}

func NewWebhookChannel(config *WebhookConfig) *WebhookChannel {
	return &WebhookChannel{config: config}
}

func (c *WebhookChannel) Name() string {
	return "webhook"
}

func (c *WebhookChannel) Send(msg AlertMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(c.config.Method, c.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *WebhookChannel) ValidateConfig() error {
	if c.config.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

func CreateChannel(channelType ChannelType, config map[string]interface{}) (AlertChannel, error) {
	switch channelType {
	case ChannelTypeSlack:
		slackConfig, err := ParseSlackConfig(config)
		if err != nil {
			return nil, err
		}
		return NewSlackChannel(slackConfig), nil
	case ChannelTypeWebhook:
		webhookConfig, err := ParseWebhookConfig(config)
		if err != nil {
			return nil, err
		}
		return NewWebhookChannel(webhookConfig), nil
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}
