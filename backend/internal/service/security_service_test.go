package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFingerprintService(t *testing.T) {
	service := NewFingerprintService()

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-browser")

	fp := service.ExtractFingerprintData(req, map[string]string{})

	assert.NotEmpty(t, fp.FingerprintID)
	assert.Equal(t, "test-browser", fp.UserAgent)
	assert.Equal(t, 1, fp.RequestCount)

	service.AddToBlacklist(fp.FingerprintID, "test reason")
	isBlacklisted, reason := service.IsBlacklisted(fp.FingerprintID)
	assert.True(t, isBlacklisted)
	assert.Equal(t, "test reason", reason)
}

func TestReplayProtectionService(t *testing.T) {
	service := NewReplayProtectionService()

	nonce, err := service.GenerateNonce()
	assert.NoError(t, err)
	assert.NotEmpty(t, nonce)

	sig, err := service.CreateSignedRequest("GET", "/test", "", nil, "secret")
	assert.NoError(t, err)
	assert.NotNil(t, sig)
	assert.NotEmpty(t, sig.Signature)
	assert.NotEmpty(t, sig.Nonce)
}

func TestSmartRateLimitService(t *testing.T) {
	service := NewSmartRateLimitService()

	for i := 0; i < 5; i++ {
		result := service.CheckRateLimit("test-client", 0)
		assert.True(t, result.Allowed)
	}

	result := service.CheckRateLimit("test-client", 0)
	assert.False(t, result.Allowed)

	stats := service.GetClientStats("test-client")
	assert.NotNil(t, stats)
	assert.Equal(t, "test-client", stats["client_id"])
}

func TestAnomalyDetectionService(t *testing.T) {
	service := NewAnomalyDetectionService()

	for i := 0; i < 15; i++ {
		service.RecordTraffic("test-client", 100, "GET", "/test", "test-agent")
	}

	result := service.DetectAnomaly("test-client")
	assert.NotNil(t, result)
}

func TestInputValidator(t *testing.T) {
	validator := NewInputValidator()

	result := validator.ValidateInput("normal input")
	assert.True(t, result.IsValid)
	assert.Empty(t, result.Errors)

	result = validator.ValidateInput("<script>alert(1)</script>")
	assert.False(t, result.IsValid)
	assert.NotEmpty(t, result.Errors)

	result = validator.ValidateInput("SELECT * FROM users")
	assert.False(t, result.IsValid)
	assert.NotEmpty(t, result.Errors)

	sanitized := validator.SanitizeInput("<script>xss</script>")
	assert.NotContains(t, sanitized, "<script")
}

func TestSecurityHeadersConfig(t *testing.T) {
	assert.NotEmpty(t, DefaultBasicSecurityHeaders.CSP)
	assert.NotEmpty(t, DefaultBasicSecurityHeaders.HSTS)
	assert.NotEmpty(t, DefaultBasicSecurityHeaders.XFrameOptions)
	assert.NotEmpty(t, DefaultBasicSecurityHeaders.XContentTypeOptions)
	assert.NotEmpty(t, DefaultBasicSecurityHeaders.XXSSProtection)
	assert.NotEmpty(t, DefaultBasicSecurityHeaders.ReferrerPolicy)
}

func TestCalculateSignature(t *testing.T) {
	service := NewReplayProtectionService()
	sig := service.CalculateSignature("test-secret", "GET", "/test", "", 123456, "test-nonce", []byte("body"))

	assert.NotEmpty(t, sig)
}
