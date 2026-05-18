package service

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOWASPInjectionCheck(t *testing.T) {
	owaspService := NewOWASPService()

	req := httptest.NewRequest("GET", "/api/test", nil)
	safe, msg := owaspService.checkInjection(req)
	assert.True(t, safe)
	assert.Empty(t, msg)
}

func TestOWASPSSRFCheck(t *testing.T) {
	owaspService := NewOWASPService()

	req := httptest.NewRequest("GET", "/api/test", nil)
	safe, msg := owaspService.checkSSRF(req)
	assert.True(t, safe)
	assert.Empty(t, msg)
}

func TestOWASPSanitizeInput(t *testing.T) {
	owaspService := NewOWASPService()

	result := owaspService.SanitizeInput("<script>alert(1)</script>")
	assert.Contains(t, result, "&lt;script&gt;")

	result = owaspService.SanitizeInput("SELECT * FROM users")
	assert.NotContains(t, result, "SELECT")

	result = owaspService.SanitizeInput("Hello World")
	assert.Contains(t, result, "Hello")
}

func TestOWASPCheckCompliance(t *testing.T) {
	owaspService := NewOWASPService()

	req := httptest.NewRequest("GET", "/api/test", nil)
	compliance := owaspService.CheckCompliance(req)

	assert.NotNil(t, compliance)
	assert.Contains(t, compliance, "checks")
	assert.Contains(t, compliance, "score")
	assert.Contains(t, compliance, "compliant")
	assert.Contains(t, compliance, "passed")
	assert.Contains(t, compliance, "total")

	score := compliance["score"].(float64)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestOWASPAllRisks(t *testing.T) {
	owaspService := NewOWASPService()

	risks := owaspService.GetAllRisks()

	assert.NotEmpty(t, risks)
	assert.Len(t, risks, 10)

	riskIDs := make(map[string]bool)
	for _, risk := range risks {
		riskIDs[risk.ID] = true
		assert.NotEmpty(t, risk.ID)
		assert.NotEmpty(t, risk.Name)
		assert.NotEmpty(t, risk.Severity)
		assert.NotEmpty(t, risk.Status)
	}

	for i := 1; i <= 10; i++ {
		assert.True(t, riskIDs[fmt.Sprintf("A%02d", i)], "Missing risk A%02d", i)
	}
}

func TestXSSSecuritySanitizeInput(t *testing.T) {
	xssSecurity := NewXSSSecurity(nil)

	result := xssSecurity.SanitizeInput("<script>alert(1)</script>")
	assert.NotContains(t, result, "script")

	result = xssSecurity.SanitizeInput("<img src=x onerror=alert(1)>")
	assert.NotContains(t, result, "onerror")

	result = xssSecurity.SanitizeInput("<iframe src='evil.com'></iframe>")
	assert.NotContains(t, result, "iframe")

	result = xssSecurity.SanitizeInput("javascript:alert(1)")
	assert.NotContains(t, result, "javascript:")

	result = xssSecurity.SanitizeInput("Hello World")
	assert.Contains(t, result, "Hello")
}

func TestXSSSecurityDetectXSS(t *testing.T) {
	xssSecurity := NewXSSSecurity(nil)

	detected, _ := xssSecurity.DetectXSS("<script>alert(1)</script>")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectXSS("javascript:alert(1)")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectXSS("<img src=x onerror=alert(1)>")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectXSS("Hello World")
	assert.False(t, detected)
}

func TestXSSSecurityDetectSQLInjection(t *testing.T) {
	xssSecurity := NewXSSSecurity(nil)

	detected, _ := xssSecurity.DetectSQLInjection("1 UNION SELECT * FROM users")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectSQLInjection("' OR '1'='1'")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectSQLInjection("'; DROP TABLE users; --")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectSQLInjection("John Doe")
	assert.False(t, detected)
}

func TestXSSSecurityDetectCommandInjection(t *testing.T) {
	xssSecurity := NewXSSSecurity(nil)

	detected, _ := xssSecurity.DetectCommandInjection("file.txt; rm -rf /")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectCommandInjection("`cat /etc/passwd`")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectCommandInjection("$(cat /etc/passwd)")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectCommandInjection("Hello World")
	assert.False(t, detected)
}

func TestIsPrivateOrLocalIP(t *testing.T) {
	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"10.x.x.x", "10.0.0.1", true},
		{"172.16.x.x", "172.16.0.1", true},
		{"172.31.x.x", "172.31.255.255", true},
		{"192.168.x.x", "192.168.1.1", true},
		{"127.x.x.x", "127.0.0.1", true},
		{"169.254.x.x", "169.254.0.1", true},
		{"0.x.x.x", "0.0.0.0", true},
		{"8.8.8.8", "8.8.8.8", false},
		{"Public IP", "1.2.3.4", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isPrivateOrLocalIP(tc.ip)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSecurityHelpersSanitizeHTML(t *testing.T) {
	result := SanitizeHTML("<script>alert(1)</script>")
	assert.NotContains(t, result, "script")

	result = SanitizeHTML("<img src=x onerror=alert(1)>")
	assert.NotContains(t, result, "onerror")

	result = SanitizeHTML("<iframe src='evil.com'></iframe>")
	assert.NotContains(t, result, "iframe")

	result = SanitizeHTML("Hello World")
	assert.Contains(t, result, "Hello")
}

func TestXSSSecurityHelpers(t *testing.T) {
	xssSecurity := NewXSSSecurity(nil)

	detected, _ := xssSecurity.DetectXSS("<script>alert(1)</script>")
	assert.True(t, detected)

	detected, _ = xssSecurity.DetectXSS("Hello World")
	assert.False(t, detected)

	result := xssSecurity.SanitizeInput("<script>alert(1)</script>")
	assert.NotContains(t, result, "script")

	result = xssSecurity.SanitizeInput("Hello World")
	assert.Contains(t, result, "Hello")
}
