package service

import (
	"net/http/httptest"
	"testing"
)

func TestNewBotDetectionService(t *testing.T) {
	service := NewBotDetectionService()
	if service == nil {
		t.Fatal("Expected non-nil service")
	}
	if len(service.botPatterns) == 0 {
		t.Error("Expected bot user agent patterns")
	}
}

func TestBotDetectLegitimateUser(t *testing.T) {
	service := NewBotDetectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	result := service.DetectBot(req, nil)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.IsBot {
		t.Error("Expected legitimate user not to be detected as bot")
	}
	if result.RiskScore > 0.5 {
		t.Errorf("Expected low risk score, got %.2f", result.RiskScore)
	}
}

func TestBotDetectBotUserAgent(t *testing.T) {
	service := NewBotDetectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Googlebot/2.1 (+http://www.google.com/bot.html)")

	result := service.DetectBot(req, nil)
	if !result.IsBot {
		t.Error("Expected Googlebot to be detected as bot")
	}
	if result.RiskScore < 0.4 {
		t.Errorf("Expected high risk score for bot, got %.2f", result.RiskScore)
	}
}

func TestBotDetectSuspiciousHeaders(t *testing.T) {
	service := NewBotDetectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("X-Scanner", "Nmap")

	result := service.DetectBot(req, nil)
	if result.RiskScore < 0.1 {
		t.Errorf("Expected elevated risk for suspicious headers, got %.2f", result.RiskScore)
	}
}

func TestBotFingerprinting(t *testing.T) {
	service := NewBotDetectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "TestBot/1.0")
	req.RemoteAddr = "192.168.1.50:12345"

	additionalData := map[string]string{
		"X-Canvas-Hash": "abc123",
		"X-WebGL-Hash":  "def456",
	}

	result := service.DetectBot(req, additionalData)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestBotMultipleRequests(t *testing.T) {
	service := NewBotDetectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "SuspiciousBot/1.0")
	req.RemoteAddr = "10.0.0.50:12345"

	for i := 0; i < 50; i++ {
		result := service.DetectBot(req, nil)
		if i > 40 && result.RiskScore < 0.4 {
			t.Logf("Warning: Risk score should increase with multiple requests, got %.2f", result.RiskScore)
		}
	}
}

func TestBotChallengeRecommendation(t *testing.T) {
	service := NewBotDetectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "UnknownScraper/1.0")
	req.RemoteAddr = "172.16.0.50:12345"

	result := service.DetectBot(req, nil)
	if result.RiskScore >= 0.7 && result.ChallengeType == "" {
		t.Error("Expected challenge type for high risk score")
	}
}

func TestBotBlacklist(t *testing.T) {
	service := NewBotDetectionService()
	ip := "192.168.1.100"

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip + ":12345"
	req.Header.Set("User-Agent", "Mozilla/5.0")

	// First request to create fingerprint
	service.DetectBot(req, nil)

	// Now blacklist (we need to add fingerprint to blacklist, not just IP)
	service.mu.Lock()
	for _, fp := range service.fingerprints {
		if fp.IP == ip {
			fp.IsBlacklisted = true
		}
	}
	service.mu.Unlock()

	result := service.DetectBot(req, nil)
	if !result.IsBot {
		t.Error("Expected blacklisted IP to be detected as bot")
	}
	if len(result.Reasons) == 0 {
		t.Log("Warning: Expected reason for blacklist detection")
	}
}
