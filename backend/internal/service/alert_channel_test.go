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
			_, err := ParseSlackConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
				"url":     "https://example.com/webhook",
				"method":  "POST",
				"headers": map[string]string{"X-API-Key": "test-key"},
			},
			wantErr: false,
		},
		{
			name: "valid with default method",
			config: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name:    "missing url",
			config:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseWebhookConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateChannel(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		config      map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "create slack channel",
			channelType: "slack",
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/test",
			},
			wantErr: false,
		},
		{
			name:        "create webhook channel",
			channelType: "webhook",
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
				assert.Equal(t, tt.channelType, channel.Name())
			}
		})
	}
}

func TestSlackChannel_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/test",
			},
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, _ := NewSlackChannel(tt.config)
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
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, _ := NewWebhookChannel(tt.config)
			err := channel.ValidateConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSlackChannel_getSeverityColor(t *testing.T) {
	config := map[string]interface{}{
		"webhook_url": "https://hooks.slack.com/services/test",
	}
	channel, _ := NewSlackChannel(config)

	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "#ff0000"},
		{"error", "#ff0000"},
		{"warning", "#ffa500"},
		{"warn", "#ffa500"},
		{"info", "#36a64f"},
		{"debug", "#808080"},
		{"unknown", "#36a64f"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			color := channel.GetSeverityColor(tt.severity)
			assert.Equal(t, tt.expected, color)
		})
	}
}
