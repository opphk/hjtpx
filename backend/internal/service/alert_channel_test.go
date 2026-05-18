package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertMessage(t *testing.T) {
	msg := AlertMessage{
		Title:     "Test Alert",
		Message:   "Test message",
		Severity:  "warning",
		EventID:   "evt-123",
		Timestamp: time.Now(),
		Context:   map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "Test Alert", msg.Title)
	assert.Equal(t, "Test message", msg.Message)
	assert.Equal(t, "warning", msg.Severity)
	assert.Equal(t, "evt-123", msg.EventID)
	assert.NotNil(t, msg.Context)
}

func TestBaseChannel_GetConfig(t *testing.T) {
	config := map[string]interface{}{"key": "value"}
	channel := NewBaseChannel(config)
	assert.Equal(t, config, channel.GetConfig())
}

func TestBaseChannel_GetSeverityColor(t *testing.T) {
	channel := NewBaseChannel(map[string]interface{}{})

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

func TestParseSlackConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"webhook_url": "https://hooks.slack.com/services/xxx",
			"channel":    "#alerts",
			"username":   "Bot",
		}
		slackConfig, err := ParseSlackConfig(config)
		assert.NoError(t, err)
		assert.NotNil(t, slackConfig)
		assert.Equal(t, "https://hooks.slack.com/services/xxx", slackConfig.WebhookURL)
		assert.Equal(t, "#alerts", slackConfig.Channel)
	})

	t.Run("missing webhook_url", func(t *testing.T) {
		config := map[string]interface{}{
			"channel": "#alerts",
		}
		slackConfig, err := ParseSlackConfig(config)
		assert.Error(t, err)
		assert.Nil(t, slackConfig)
	})
}

func TestParseWebhookConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"url":    "https://example.com/webhook",
			"method": "POST",
		}
		webhookConfig, err := ParseWebhookConfig(config)
		assert.NoError(t, err)
		assert.NotNil(t, webhookConfig)
		assert.Equal(t, "https://example.com/webhook", webhookConfig.URL)
		assert.Equal(t, "POST", webhookConfig.Method)
	})

	t.Run("missing url", func(t *testing.T) {
		config := map[string]interface{}{
			"method": "POST",
		}
		webhookConfig, err := ParseWebhookConfig(config)
		assert.Error(t, err)
		assert.Nil(t, webhookConfig)
	})

	t.Run("default method", func(t *testing.T) {
		config := map[string]interface{}{
			"url": "https://example.com/webhook",
		}
		webhookConfig, err := ParseWebhookConfig(config)
		assert.NoError(t, err)
		assert.NotNil(t, webhookConfig)
		assert.Equal(t, "POST", webhookConfig.Method)
	})
}

func TestParseEmailConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
			"username":     "user@example.com",
			"password":     "password",
			"from_address": "from@example.com",
			"to_addresses": []string{"to@example.com"},
		}
		emailConfig, err := ParseEmailConfig(config)
		assert.NoError(t, err)
		assert.NotNil(t, emailConfig)
		assert.Equal(t, "smtp.example.com", emailConfig.SMTPHost)
		assert.Equal(t, 587, emailConfig.SMTPPort)
	})

	t.Run("missing smtp_host", func(t *testing.T) {
		config := map[string]interface{}{
			"smtp_port":    587,
			"username":     "user@example.com",
			"from_address": "from@example.com",
			"to_addresses": []string{"to@example.com"},
		}
		emailConfig, err := ParseEmailConfig(config)
		assert.Error(t, err)
		assert.Nil(t, emailConfig)
	})

	t.Run("default smtp_port", func(t *testing.T) {
		config := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"username":     "user@example.com",
			"from_address": "from@example.com",
			"to_addresses": []string{"to@example.com"},
		}
		emailConfig, err := ParseEmailConfig(config)
		assert.NoError(t, err)
		assert.NotNil(t, emailConfig)
		assert.Equal(t, 587, emailConfig.SMTPPort)
	})

	t.Run("missing to_addresses", func(t *testing.T) {
		config := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
			"username":     "user@example.com",
			"from_address": "from@example.com",
		}
		emailConfig, err := ParseEmailConfig(config)
		assert.Error(t, err)
		assert.Nil(t, emailConfig)
	})
}

func TestParseDingTalkConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := map[string]interface{}{
			"webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
			"secret":      "secret",
			"at_mobiles":  []string{"13800138000"},
			"is_at_all":   false,
		}
		dingtalkConfig, err := ParseDingTalkConfig(config)
		assert.NoError(t, err)
		assert.NotNil(t, dingtalkConfig)
		assert.Equal(t, "https://oapi.dingtalk.com/robot/send?access_token=xxx", dingtalkConfig.WebhookURL)
		assert.Equal(t, "secret", dingtalkConfig.Secret)
	})

	t.Run("missing webhook_url", func(t *testing.T) {
		config := map[string]interface{}{
			"secret": "secret",
		}
		dingtalkConfig, err := ParseDingTalkConfig(config)
		assert.Error(t, err)
		assert.Nil(t, dingtalkConfig)
	})
}

func TestSlackChannel(t *testing.T) {
	t.Run("create slack channel", func(t *testing.T) {
		config := map[string]interface{}{
			"webhook_url": "https://hooks.slack.com/services/xxx",
			"channel":     "#alerts",
		}
		channel, err := NewSlackChannel(config)
		assert.NoError(t, err)
		assert.NotNil(t, channel)
		assert.Equal(t, "slack", channel.Name())
	})

	t.Run("validate config", func(t *testing.T) {
		config := map[string]interface{}{
			"webhook_url": "https://hooks.slack.com/services/xxx",
		}
		channel, _ := NewSlackChannel(config)
		err := channel.ValidateConfig()
		assert.NoError(t, err)
	})
}

func TestWebhookChannel(t *testing.T) {
	t.Run("create webhook channel", func(t *testing.T) {
		config := map[string]interface{}{
			"url":    "https://example.com/webhook",
			"method": "POST",
		}
		channel, err := NewWebhookChannel(config)
		assert.NoError(t, err)
		assert.NotNil(t, channel)
		assert.Equal(t, "webhook", channel.Name())
	})

	t.Run("validate config", func(t *testing.T) {
		config := map[string]interface{}{
			"url": "https://example.com/webhook",
		}
		channel, _ := NewWebhookChannel(config)
		err := channel.ValidateConfig()
		assert.NoError(t, err)
	})
}

func TestEmailChannel(t *testing.T) {
	t.Run("create email channel", func(t *testing.T) {
		config := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
			"username":     "user@example.com",
			"password":     "password",
			"from_address": "from@example.com",
			"to_addresses": []string{"to@example.com"},
		}
		channel, err := NewEmailChannel(config)
		assert.NoError(t, err)
		assert.NotNil(t, channel)
		assert.Equal(t, "email", channel.Name())
	})

	t.Run("format context", func(t *testing.T) {
		config := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
			"username":     "user@example.com",
			"from_address": "from@example.com",
			"to_addresses": []string{"to@example.com"},
		}
		channel, _ := NewEmailChannel(config)
		ctx := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}
		formatted := channel.formatContext(ctx)
		assert.Contains(t, formatted, "key1")
		assert.Contains(t, formatted, "value1")
	})

	t.Run("format nil context", func(t *testing.T) {
		config := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
			"username":     "user@example.com",
			"from_address": "from@example.com",
			"to_addresses": []string{"to@example.com"},
		}
		channel, _ := NewEmailChannel(config)
		formatted := channel.formatContext(nil)
		assert.Equal(t, "无", formatted)
	})
}

func TestDingTalkChannel(t *testing.T) {
	t.Run("create dingtalk channel", func(t *testing.T) {
		config := map[string]interface{}{
			"webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
			"secret":      "secret",
		}
		channel, err := NewDingTalkChannel(config)
		assert.NoError(t, err)
		assert.NotNil(t, channel)
		assert.Equal(t, "dingtalk", channel.Name())
	})

	t.Run("validate config", func(t *testing.T) {
		config := map[string]interface{}{
			"webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
		}
		channel, _ := NewDingTalkChannel(config)
		err := channel.ValidateConfig()
		assert.NoError(t, err)
	})
}

func TestCreateChannel(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		config      map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "slack channel",
			channelType: "slack",
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/xxx",
			},
			wantErr: false,
		},
		{
			name:        "webhook channel",
			channelType: "webhook",
			config: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name:        "email channel",
			channelType: "email",
			config: map[string]interface{}{
				"smtp_host":    "smtp.example.com",
				"smtp_port":    587,
				"username":     "user@example.com",
				"from_address": "from@example.com",
				"to_addresses": []string{"to@example.com"},
			},
			wantErr: false,
		},
		{
			name:        "dingtalk channel",
			channelType: "dingtalk",
			config: map[string]interface{}{
				"webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
			},
			wantErr: false,
		},
		{
			name:        "unsupported channel",
			channelType: "unsupported",
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

func TestSlackChannel_BuildPayload(t *testing.T) {
	config := map[string]interface{}{
		"webhook_url": "https://hooks.slack.com/services/xxx",
		"channel":     "#alerts",
		"username":    "Bot",
		"icon_emoji":  ":robot_face:",
	}
	channel, _ := NewSlackChannel(config)

	msg := AlertMessage{
		Title:     "Test Alert",
		Message:   "Test message content",
		Severity:  "warning",
		EventID:   "evt-123",
		Timestamp: time.Now(),
		Context:   map[string]interface{}{"key": "value"},
	}

	payload := channel.buildSlackPayload(msg)
	assert.NotNil(t, payload)
	assert.Contains(t, payload, "attachments")
	attachments, ok := payload["attachments"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, attachments, 1)
	assert.Equal(t, "#ffa500", attachments[0]["color"])
}

func TestDingTalkChannel_BuildPayload(t *testing.T) {
	config := map[string]interface{}{
		"webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
		"at_mobiles":  []string{"13800138000"},
		"is_at_all":   false,
	}
	channel, _ := NewDingTalkChannel(config)

	msg := AlertMessage{
		Title:     "Test Alert",
		Message:   "Test message content",
		Severity:  "critical",
		EventID:   "evt-123",
		Timestamp: time.Now(),
		Context:   map[string]interface{}{"key": "value"},
	}

	payload := channel.buildDingTalkPayload(msg)
	assert.NotNil(t, payload)
	assert.Contains(t, payload, "msgtype")
	assert.Contains(t, payload, "markdown")
	assert.Contains(t, payload, "at")
}

func TestChannelType_Constants(t *testing.T) {
	assert.Equal(t, ChannelType("slack"), ChannelTypeSlack)
	assert.Equal(t, ChannelType("webhook"), ChannelTypeWebhook)
	assert.Equal(t, ChannelType("email"), ChannelTypeEmail)
	assert.Equal(t, ChannelType("dingtalk"), ChannelTypeDingTalk)
}
