package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	config := &RateLimiterConfig{
		LimiterType: FixedWindow,
		GlobalLimit: &LimitConfig{
			MaxRequests:    10,
			WindowDuration: time.Minute,
		},
	}

	limiter := NewRateLimiter(config)
	require.NotNil(t, limiter)

	t.Run("添加到白名单", func(t *testing.T) {
		limiter.AddToWhitelist("192.168.1.1")
		assert.True(t, limiter.IsWhitelisted("192.168.1.1"))
	})

	t.Run("添加到黑名单", func(t *testing.T) {
		limiter.AddToBlacklist("10.0.0.1")
		assert.True(t, limiter.IsBlacklisted("10.0.0.1"))
	})

	t.Run("设置IP端点限流", func(t *testing.T) {
		limiter.SetIPEndpointLimit("192.168.1.100", "/api/login", &LimitConfig{
			MaxRequests:    5,
			WindowDuration: time.Minute,
		})
	})
}

func TestXSSProtection(t *testing.T) {
	protector := NewXSSProtector(DefaultXSSConfig)
	require.NotNil(t, protector)

	tests := []struct {
		name  string
		input string
	}{
		{"空字符串", ""},
		{"正常文本", "Hello World"},
		{"移除脚本标签", "<script>alert('xss')</script>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protector.SanitizeHTML(tt.input)
			assert.NotContains(t, result, "<script>")
		})
	}
}

func TestCSRFProtection(t *testing.T) {
	config := &CSRFConfig{
		Secret:      "test-secret-key",
		TokenLength: 32,
		TokenExpiry: 24 * time.Hour,
	}

	csrf := NewCSRFProtection(config)
	require.NotNil(t, csrf)

	t.Run("生成Token", func(t *testing.T) {
		token, err := csrf.GenerateToken("session123")
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("验证Token", func(t *testing.T) {
		token, err := csrf.GenerateToken("session123")
		require.NoError(t, err)

		valid := csrf.ValidateToken("session123", token)
		assert.True(t, valid)
	})

	t.Run("无效Token", func(t *testing.T) {
		valid := csrf.ValidateToken("session123", "invalid-token")
		assert.False(t, valid)
	})

	t.Run("过期Token", func(t *testing.T) {
		config := &CSRFConfig{
			Secret:      "test-secret",
			TokenLength: 32,
			TokenExpiry: -1 * time.Hour,
		}
		csrf := NewCSRFProtection(config)

		token, err := csrf.GenerateToken("session123")
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		valid := csrf.ValidateToken("session123", token)
		assert.False(t, valid)
	})
}

func TestValidation(t *testing.T) {
	validator := NewValidator([]ValidationRule{
		{Field: "email", Type: "email", Required: true},
		{Field: "age", Type: "integer", Min: 0, Max: 150},
		{Field: "name", Type: "string", Min: 2, Max: 50},
	})

	t.Run("有效数据", func(t *testing.T) {
		data := map[string]interface{}{
			"email": "test@example.com",
			"age":   25,
			"name":  "John",
		}
		err := validator.Validate(data)
		assert.NoError(t, err)
	})

	t.Run("无效邮箱", func(t *testing.T) {
		data := map[string]interface{}{
			"email": "invalid-email",
		}
		err := validator.Validate(data)
		assert.Error(t, err)
	})

	t.Run("超出范围", func(t *testing.T) {
		data := map[string]interface{}{
			"email": "test@example.com",
			"age":   200,
		}
		err := validator.Validate(data)
		assert.Error(t, err)
	})

	t.Run("必填字段缺失", func(t *testing.T) {
		data := map[string]interface{}{
			"age": 25,
		}
		err := validator.Validate(data)
		assert.Error(t, err)
	})
}
