package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertMessage_Creation(t *testing.T) {
	msg := AlertMessage{
		Title:     "Test Alert",
		Message:   "This is a test alert message",
		Severity:  "warning",
		EventID:   "evt-123",
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"key": "value",
			"num": 123,
		},
	}

	assert.Equal(t, "Test Alert", msg.Title)
	assert.Equal(t, "warning", msg.Severity)
	assert.Equal(t, "evt-123", msg.EventID)
	assert.NotEmpty(t, msg.Context)
}

func TestParseSlackConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid slack config",
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/test",
				"channel":     "#alerts",
				"username":    "AlertBot",
				"icon_emoji":  ":warning:",
			},
			wantErr: false,
		},
		{
			name: "missing webhook url",
			config: map[string]interface{}{
				"channel": "#alerts",
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseSlackConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, "https://hooks.slack.com/services/test", config.WebhookURL)
			}
		})
	}
}

func TestParseWebhookConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid webhook config",
			config: map[string]interface{}{
				"url":    "https://example.com/webhook",
				"method": "POST",
			},
			wantErr: false,
		},
		{
			name: "missing url",
			config: map[string]interface{}{
				"method": "POST",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseWebhookConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, "https://example.com/webhook", config.URL)
			}
		})
	}
}

func TestCreateChannel(t *testing.T) {
	tests := []struct {
		name        string
		channelType ChannelType
		config      map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "create slack channel",
			channelType: ChannelTypeSlack,
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/test",
			},
			wantErr: false,
		},
		{
			name:        "create webhook channel",
			channelType: ChannelTypeWebhook,
			config: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name:        "invalid channel type",
			channelType: "invalid",
			config:      map[string]interface{}{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := CreateChannel(tt.channelType, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, channel)
			}
		})
	}
}

func TestSlackChannel_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *SlackConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &SlackConfig{
				WebhookURL: "https://hooks.slack.com/services/test",
			},
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  &SlackConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewSlackChannel(tt.config)
			err := channel.ValidateConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookChannel_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *WebhookConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &WebhookConfig{
				URL: "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  &WebhookConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewWebhookChannel(tt.config)
			err := channel.ValidateConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSlackChannel_Name(t *testing.T) {
	config := &SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
	}
	channel := NewSlackChannel(config)
	assert.Equal(t, "slack", channel.Name())
}

func TestWebhookChannel_Name(t *testing.T) {
	config := &WebhookConfig{
		URL: "https://example.com/webhook",
	}
	channel := NewWebhookChannel(config)
	assert.Equal(t, "webhook", channel.Name())
}
