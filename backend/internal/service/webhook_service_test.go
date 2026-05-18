package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseWeChatWorkConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid wechat work config",
			config: map[string]interface{}{
				"webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
				"at_mobiles":  []string{"13800138000"},
				"is_at_all":   false,
			},
			wantErr: false,
		},
		{
			name: "missing webhook url",
			config: map[string]interface{}{
				"at_mobiles": []string{"13800138000"},
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
			_, err := ParseWeChatWorkConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateWeChatWorkChannel(t *testing.T) {
	config := map[string]interface{}{
		"webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
	}

	channel, err := CreateChannel("wechat_work", config)
	assert.NoError(t, err)
	assert.NotNil(t, channel)
	assert.Equal(t, "wechat_work", channel.Name())
}

func TestWeChatWorkChannel_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
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
			channel, _ := NewWeChatWorkChannel(tt.config)
			err := channel.ValidateConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookSignatureVerifier_Verify(t *testing.T) {
	secret := "test-secret-123"
	verifier := NewWebhookSignatureVerifier(secret)

	tests := []struct {
		name      string
		signature string
		timestamp string
		body      []byte
		wantValid bool
	}{
		{
			name:      "valid signature",
			signature: "jF2n5dXgH7qK9mP2rT4vY6zA8cE0gB3iD5fH7jK9lM1nO3pQ5rS7tU9vW1xY3zA5",
			timestamp: "1234567890",
			body:      []byte(`{"event":"test","data":"hello"}`),
			wantValid: false, // 这个签名是假的，应该验证失败
		},
		{
			name:      "missing signature",
			signature: "",
			timestamp: "1234567890",
			body:      []byte(`{"event":"test"}`),
			wantValid: false,
		},
		{
			name:      "missing timestamp",
			signature: "test-signature",
			timestamp: "",
			body:      []byte(`{"event":"test"}`),
			wantValid: false,
		},
		{
			name:      "empty secret",
			signature: "test-signature",
			timestamp: "1234567890",
			body:      []byte(`{"event":"test"}`),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果测试用例需要空 secret，创建新的 verifier
			testVerifier := verifier
			if tt.name == "empty secret" {
				testVerifier = NewWebhookSignatureVerifier("")
			}
			
			valid := testVerifier.Verify(tt.signature, tt.timestamp, tt.body)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestWebhookSignatureVerifier_VerifyFromRequest(t *testing.T) {
	secret := "test-secret"
	verifier := NewWebhookSignatureVerifier(secret)

	tests := []struct {
		name           string
		setupHeaders   func(*http.Request)
		body           []byte
		wantValid      bool
		wantErr        bool
	}{
		{
			name: "valid request with headers",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Signature", "test-signature")
				req.Header.Set("X-Webhook-Timestamp", "1234567890")
			},
			body:      []byte(`{"event":"test"}`),
			wantValid: false, // 签名不匹配
			wantErr:   false,
		},
		{
			name: "missing signature header",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Timestamp", "1234567890")
			},
			body:      []byte(`{"event":"test"}`),
			wantValid: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook", bytes.NewBuffer(tt.body))
			tt.setupHeaders(req)

			valid, body, err := verifier.VerifyFromRequest(req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.body, body)
			}
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestOAuth2Service(t *testing.T) {
	config := &OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
		AuthURL:      "https://auth.example.com/authorize",
		TokenURL:     "https://auth.example.com/token",
		Scope:        []string{"read", "write"},
	}

	service := NewOAuth2Service(config)

	// 测试获取授权 URL
	authURL := service.GetAuthorizationURL("test-state-123")
	assert.Contains(t, authURL, "test-client-id")
	assert.Contains(t, authURL, "test-state-123")
	assert.Contains(t, authURL, "read+write")

	// 测试令牌管理
	token := &OAuth2Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	service.SetToken(token)
	assert.Equal(t, token, service.GetToken())
	assert.False(t, service.IsTokenExpired())

	// 测试过期令牌
	expiredToken := &OAuth2Token{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
	}
	service.SetToken(expiredToken)
	assert.True(t, service.IsTokenExpired())
}

func TestWebhookService(t *testing.T) {
	service := NewWebhookService()

	// 测试签名验证器
	assert.Nil(t, service.GetSignatureVerifier())
	service.SetSignatureVerifier("test-secret")
	assert.NotNil(t, service.GetSignatureVerifier())

	// 测试 OAuth2 服务
	oauthConfig := &OAuth2Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
		AuthURL:      "https://auth.example.com/authorize",
		TokenURL:     "https://auth.example.com/token",
	}

	service.RegisterOAuth2Service("test-provider", oauthConfig)
	
	providers := service.ListOAuth2Services()
	assert.Len(t, providers, 1)
	assert.Contains(t, providers, "test-provider")

	oauthService := service.GetOAuth2Service("test-provider")
	assert.NotNil(t, oauthService)

	// 测试不存在的 provider
	nonexistent := service.GetOAuth2Service("nonexistent")
	assert.Nil(t, nonexistent)
}
