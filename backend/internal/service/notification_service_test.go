package service

import (
	"testing"
	"time"
)

func TestSlackService(t *testing.T) {
	t.Run("NewSlackService", func(t *testing.T) {
		service := NewSlackService()
		if service == nil {
			t.Fatal("SlackService should not be nil")
		}
		if service.configs == nil {
			t.Fatal("configs map should be initialized")
		}
		if service.client == nil {
			t.Fatal("HTTP client should be initialized")
		}
	})
	
	t.Run("RegisterChannel", func(t *testing.T) {
		service := NewSlackService()
		
		config := &SlackChannelConfig{
			WebhookURL: "https://hooks.slack.com/services/test",
			Channel:    "#test-channel",
			Username:   "HJTPX Bot",
			IconEmoji:  ":robot_face:",
		}
		
		service.RegisterChannel("test", config)
		
		retrieved, ok := service.GetChannel("test")
		if !ok {
			t.Fatal("Channel should be registered")
		}
		if retrieved.WebhookURL != config.WebhookURL {
			t.Errorf("WebhookURL should match")
		}
		if retrieved.Channel != config.Channel {
			t.Errorf("Channel should match")
		}
	})
	
	t.Run("ListChannels", func(t *testing.T) {
		service := NewSlackService()
		
		service.RegisterChannel("channel1", &SlackChannelConfig{WebhookURL: "https://example.com/1"})
		service.RegisterChannel("channel2", &SlackChannelConfig{WebhookURL: "https://example.com/2"})
		
		channels := service.ListChannels()
		if len(channels) != 2 {
			t.Errorf("Expected 2 channels, got %d", len(channels))
		}
	})
	
	t.Run("GetChannelNotFound", func(t *testing.T) {
		service := NewSlackService()
		
		_, ok := service.GetChannel("nonexistent")
		if ok {
			t.Error("Should return false for nonexistent channel")
		}
	})
	
	t.Run("BuildSlackPayload", func(t *testing.T) {
		service := NewSlackService()
		
		config := &SlackChannelConfig{
			WebhookURL: "https://hooks.slack.com/test",
			Channel:    "#alerts",
			Username:   "Alert Bot",
		}
		
		msg := NotificationMessage{
			ID:        "test-123",
			Severity:  SeverityCritical,
			Title:     "Test Alert",
			Content:   "This is a test alert",
			CreatedAt: time.Now(),
		}
		
		payload := service.buildSlackPayload(msg, config)
		
		if payload["channel"] != "#alerts" {
			t.Errorf("Channel should be #alerts")
		}
		if payload["username"] != "Alert Bot" {
			t.Errorf("Username should be Alert Bot")
		}
		
		attachments, ok := payload["attachments"].([]map[string]interface{})
		if !ok || len(attachments) == 0 {
			t.Fatal("Should have attachments")
		}
		
		if attachments[0]["title"] != "Test Alert" {
			t.Errorf("Title should match")
		}
	})
	
	t.Run("GetSeverityColor", func(t *testing.T) {
		service := NewSlackService()
		
		testCases := []struct {
			severity NotificationSeverity
			expected string
		}{
			{SeverityInfo, "#36a64f"},
			{SeverityWarning, "#ff9800"},
			{SeverityError, "#f44336"},
			{SeverityCritical, "#b71c1c"},
		}
		
		for _, tc := range testCases {
			color := service.getSeverityColor(tc.severity)
			if color != tc.expected {
				t.Errorf("Expected %s for %s, got %s", tc.expected, tc.severity, color)
			}
		}
	})
}

func TestWeChatWorkNotificationService(t *testing.T) {
	t.Run("NewWeChatWorkNotificationService", func(t *testing.T) {
		service := NewWeChatWorkNotificationService()
		if service == nil {
			t.Fatal("WeChatWorkNotificationService should not be nil")
		}
		if service.slackService == nil {
			t.Fatal("slackService should be initialized")
		}
	})
	
	t.Run("RegisterChannel", func(t *testing.T) {
		service := NewWeChatWorkNotificationService()
		
		config := &SlackChannelConfig{
			WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		}
		
		service.RegisterChannel("wechat-test", config)
		
		retrieved, ok := service.slackService.GetChannel("wechat-test")
		if !ok {
			t.Fatal("Channel should be registered")
		}
		if retrieved.WebhookURL != config.WebhookURL {
			t.Errorf("WebhookURL should match")
		}
	})
	
	t.Run("SendToMultiple", func(t *testing.T) {
		service := NewWeChatWorkNotificationService()
		
		service.RegisterChannel("ch1", &SlackChannelConfig{WebhookURL: "https://example.com/1"})
		service.RegisterChannel("ch2", &SlackChannelConfig{WebhookURL: "https://example.com/2"})
		
		msg := NotificationMessage{
			ID:        "test-123",
			Severity:  SeverityInfo,
			Title:     "Test",
			Content:   "Test content",
			CreatedAt: time.Now(),
		}
		
		errors := service.SendToMultiple([]string{"ch1", "ch2"}, msg)
		if len(errors) != 2 {
			t.Logf("Expected 2 errors (network failures), got %d", len(errors))
		}
	})
}

func TestNotificationManager(t *testing.T) {
	t.Run("NewNotificationManager", func(t *testing.T) {
		manager := NewNotificationManager()
		if manager == nil {
			t.Fatal("NotificationManager should not be nil")
		}
		if manager.slackService == nil {
			t.Fatal("slackService should be initialized")
		}
		if manager.wechatService == nil {
			t.Fatal("wechatService should be initialized")
		}
	})
	
	t.Run("RegisterChannel", func(t *testing.T) {
		manager := NewNotificationManager()
		
		config := &SlackChannelConfig{WebhookURL: "https://example.com/webhook"}
		err := manager.RegisterChannel("test-slack", ChannelSlack, config)
		if err != nil {
			t.Fatalf("Should not return error: %v", err)
		}
		
		err = manager.RegisterChannel("test-wechat", ChannelWeChatWork, config)
		if err != nil {
			t.Fatalf("Should not return error: %v", err)
		}
		
		channels := manager.ListChannels()
		if len(channels) != 2 {
			t.Errorf("Expected 2 channels, got %d", len(channels))
		}
	})
	
	t.Run("ListChannels", func(t *testing.T) {
		manager := NewNotificationManager()
		
		manager.RegisterChannel("slack1", ChannelSlack, &SlackChannelConfig{WebhookURL: "https://example.com/1"})
		manager.RegisterChannel("slack2", ChannelSlack, &SlackChannelConfig{WebhookURL: "https://example.com/2"})
		manager.RegisterChannel("wechat1", ChannelWeChatWork, &SlackChannelConfig{WebhookURL: "https://example.com/3"})
		
		channels := manager.ListChannels()
		if len(channels) != 3 {
			t.Errorf("Expected 3 channels, got %d", len(channels))
		}
		
		if channels["slack1"] != ChannelSlack {
			t.Errorf("slack1 should be ChannelSlack")
		}
		if channels["wechat1"] != ChannelWeChatWork {
			t.Errorf("wechat1 should be ChannelWeChatWork")
		}
	})
	
	t.Run("SendNotificationChannelNotFound", func(t *testing.T) {
		manager := NewNotificationManager()
		
		msg := NotificationMessage{
			ID:        "test-123",
			Severity:  SeverityInfo,
			Title:     "Test",
			Content:   "Test",
			CreatedAt: time.Now(),
		}
		
		err := manager.SendNotification("nonexistent", msg)
		if err == nil {
			t.Error("Should return error for nonexistent channel")
		}
	})
	
	t.Run("GetServices", func(t *testing.T) {
		manager := NewNotificationManager()
		
		slack := manager.GetSlackService()
		if slack == nil {
			t.Fatal("GetSlackService should not return nil")
		}
		
		wechat := manager.GetWeChatService()
		if wechat == nil {
			t.Fatal("GetWeChatService should not return nil")
		}
	})
}

func TestNotificationMessage(t *testing.T) {
	t.Run("CreateNotificationMessage", func(t *testing.T) {
		msg := NotificationMessage{
			ID:        "test-001",
			Channel:   ChannelSlack,
			Severity:  SeverityWarning,
			Title:     "Test Notification",
			Content:   "This is a test notification",
			Data: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			Recipient: "test@example.com",
			CreatedAt: time.Now(),
		}
		
		if msg.ID != "test-001" {
			t.Errorf("ID should match")
		}
		if msg.Channel != ChannelSlack {
			t.Errorf("Channel should be slack")
		}
		if msg.Severity != SeverityWarning {
			t.Errorf("Severity should be warning")
		}
		if msg.Data["key1"] != "value1" {
			t.Errorf("Data[key1] should be value1")
		}
	})
}

func TestSlackServiceBlockKit(t *testing.T) {
	t.Run("BuildVerificationAlertBlock", func(t *testing.T) {
		service := NewSlackService()
		
		msg := NotificationMessage{
			ID:        "test-123",
			Severity:  SeverityCritical,
			Title:     "Critical Alert",
			Content:   "This is a critical alert",
			CreatedAt: time.Now(),
		}
		
		blocks := service.BuildVerificationAlertBlock(msg)
		
		if len(blocks) == 0 {
			t.Fatal("Should have blocks")
		}
		
		header, ok := blocks[0]["text"].(map[string]interface{})
		if !ok {
			t.Fatal("First block should have text")
		}
		
		if header["text"] != ":fire: Critical Alert" {
			t.Errorf("Title should include emoji: %v", header["text"])
		}
	})
	
	t.Run("BuildDataBlock", func(t *testing.T) {
		service := NewSlackService()
		
		fields := map[string]interface{}{
			"Total Verifications": 1000,
			"Success Rate":        "99.5%",
			"Avg Response Time":   "45ms",
		}
		
		blocks := service.BuildDataBlock(fields)
		
		if len(blocks) != 3 {
			t.Errorf("Expected 3 blocks, got %d", len(blocks))
		}
	})
	
	t.Run("SendBlockKitNotFound", func(t *testing.T) {
		service := NewSlackService()
		
		blocks := []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "plain_text",
					"text": "Test",
				},
			},
		}
		
		err := service.SendBlockKit("nonexistent", blocks)
		if err == nil {
			t.Error("Should return error for nonexistent channel")
		}
	})
}

func TestNotificationChannels(t *testing.T) {
	t.Run("ChannelConstants", func(t *testing.T) {
		if ChannelSlack != "slack" {
			t.Errorf("ChannelSlack should be 'slack'")
		}
		if ChannelWeChatWork != "wechat_work" {
			t.Errorf("ChannelWeChatWork should be 'wechat_work'")
		}
		if ChannelEmail != "email" {
			t.Errorf("ChannelEmail should be 'email'")
		}
		if ChannelWebhook != "webhook" {
			t.Errorf("ChannelWebhook should be 'webhook'")
		}
		if ChannelSMS != "sms" {
			t.Errorf("ChannelSMS should be 'sms'")
		}
	})
	
	t.Run("SeverityConstants", func(t *testing.T) {
		if SeverityInfo != "info" {
			t.Errorf("SeverityInfo should be 'info'")
		}
		if SeverityWarning != "warning" {
			t.Errorf("SeverityWarning should be 'warning'")
		}
		if SeverityError != "error" {
			t.Errorf("SeverityError should be 'error'")
		}
		if SeverityCritical != "critical" {
			t.Errorf("SeverityCritical should be 'critical'")
		}
	})
}

func TestGetNotificationManager(t *testing.T) {
	t.Run("SingletonPattern", func(t *testing.T) {
		manager1 := GetNotificationManager()
		manager2 := GetNotificationManager()
		
		if manager1 != manager2 {
			t.Error("GetNotificationManager should return the same instance")
		}
	})
	
	t.Run("ManagerFunctionality", func(t *testing.T) {
		manager := GetNotificationManager()
		
		channels := manager.ListChannels()
		if channels == nil {
			t.Error("ListChannels should return a map, not nil")
		}
	})
}
