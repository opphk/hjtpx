package service

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewDDoSProtectionService(t *testing.T) {
	service := NewDDoSProtectionService()
	if service == nil {
		t.Fatal("Expected non-nil service")
	}
	if service.maxIPs != 10000 {
		t.Errorf("Expected maxIPs=10000, got %d", service.maxIPs)
	}
}

func TestDDoSCheckRequest(t *testing.T) {
	service := NewDDoSProtectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	result := service.CheckRequest(req)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !result.Allowed {
		t.Error("Expected request to be allowed")
	}
}

func TestDDoSRateLimit(t *testing.T) {
	service := NewDDoSProtectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	for i := 0; i < 100; i++ {
		result := service.CheckRequest(req)
		if i < 100 && !result.Allowed {
			t.Errorf("Expected request %d to be allowed", i)
		}
	}

	result := service.CheckRequest(req)
	if result.Allowed {
		t.Error("Expected request to be rate limited")
	}
	if result.Reason != "rate_limit" {
		t.Errorf("Expected reason 'rate_limit', got '%s'", result.Reason)
	}
}

func TestDDoSBlacklist(t *testing.T) {
	service := NewDDoSProtectionService()
	ip := "192.168.1.100"
	service.AddToBlacklist(ip, "test", time.Hour)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip + ":12345"

	result := service.CheckRequest(req)
	if result.Allowed {
		t.Error("Expected blacklisted IP to be blocked")
	}
	if result.Reason != "blacklisted" {
		t.Errorf("Expected reason 'blacklisted', got '%s'", result.Reason)
	}

	service.RemoveFromBlacklist(ip)
	result = service.CheckRequest(req)
	if !result.Allowed {
		t.Error("Expected request to be allowed after removing from blacklist")
	}
}

func TestDDoSGetIPStats(t *testing.T) {
	service := NewDDoSProtectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "172.16.0.1:12345"

	for i := 0; i < 10; i++ {
		service.CheckRequest(req)
	}

	stats := service.GetIPStats("172.16.0.1")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
	if stats.RequestCount < 10 {
		t.Errorf("Expected at least 10 requests, got %d", stats.RequestCount)
	}
}

func TestDDoSAnomalyDetection(t *testing.T) {
	service := NewDDoSProtectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.2.1:12345"

	for i := 0; i < 100; i++ {
		service.CheckRequest(req)
		time.Sleep(1 * time.Millisecond)
	}

	stats := service.GetIPStats("192.168.2.1")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
}

func TestDDoSWhitelist(t *testing.T) {
	service := NewDDoSProtectionService()
	ip := "10.0.0.100"
	service.AddToWhitelist(ip)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip + ":12345"

	result := service.CheckRequest(req)
	if !result.Allowed {
		t.Error("Expected whitelisted IP to be allowed")
	}
	if result.Reason != "whitelisted" {
		t.Errorf("Expected reason 'whitelisted', got '%s'", result.Reason)
	}

	service.RemoveFromWhitelist(ip)
	result = service.CheckRequest(req)
	if !result.Allowed {
		t.Error("Expected request to be allowed after removing from whitelist")
	}
}

func TestDDoSBotDetection(t *testing.T) {
	service := NewDDoSProtectionService()

	botReq := httptest.NewRequest("GET", "/test", nil)
	botReq.RemoteAddr = "192.168.3.1:12345"
	botReq.Header.Set("User-Agent", "curl/7.68.0")

	result := service.CheckRequest(botReq)
	if result.Allowed {
		t.Error("Expected bot user agent to be blocked")
	}
	if result.Reason != "bot_detected" {
		t.Errorf("Expected reason 'bot_detected', got '%s'", result.Reason)
	}

	normalReq := httptest.NewRequest("GET", "/test", nil)
	normalReq.RemoteAddr = "192.168.3.2:12345"
	normalReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0)")

	result = service.CheckRequest(normalReq)
	if !result.Allowed {
		t.Error("Expected normal user agent to be allowed")
	}
}

func TestDDoSIPReputation(t *testing.T) {
	service := NewDDoSProtectionService()

	privateReq := httptest.NewRequest("GET", "/test", nil)
	privateReq.RemoteAddr = "192.168.1.1:12345"

	result := service.CheckRequest(privateReq)
	if !result.Allowed {
		t.Error("Expected private IP to be allowed")
	}

	stats := service.GetIPStats("192.168.1.1")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
	if stats.Reputation < 70 {
		t.Errorf("Expected high reputation score for private IP, got %d", stats.Reputation)
	}
}

func TestDDoSAdvancedAnomalyDetection(t *testing.T) {
	service := NewDDoSProtectionService()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.4.1:12345"
	req.Header.Set("User-Agent", "TestAgent/1.0")

	for i := 0; i < 50; i++ {
		service.CheckRequest(req)
		time.Sleep(10 * time.Millisecond)
	}

	stats := service.GetIPStats("192.168.4.1")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
}

func TestDDoSDynamicRateLimit(t *testing.T) {
	service := NewDDoSProtectionService()

	result := service.getDynamicRateLimit(80, 0.0)
	if result != 60 {
		t.Errorf("Expected rate limit 60 for high reputation, got %d", result)
	}

	result = service.getDynamicRateLimit(25, 0.0)
	if result != 18 {
		t.Errorf("Expected rate limit 18 for low reputation, got %d", result)
	}

	result = service.getDynamicRateLimit(80, 0.6)
	if result != 30 {
		t.Errorf("Expected rate limit 30 for anomaly score > 0.5, got %d", result)
	}
}

func TestDDoSGetGlobalStats(t *testing.T) {
	service := NewDDoSProtectionService()

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.2:12345"

	service.CheckRequest(req1)
	service.CheckRequest(req2)

	stats := service.GetGlobalStats()
	if stats["total_ips"] != 2 {
		t.Errorf("Expected total_ips 2, got %v", stats["total_ips"])
	}
	if stats["blacklist_count"] != 0 {
		t.Errorf("Expected blacklist_count 0, got %v", stats["blacklist_count"])
	}
}

func TestDDoSCountUnique(t *testing.T) {
	service := NewDDoSProtectionService()

	items := []string{"a", "b", "a", "c", "b"}
	count := service.countUnique(items)
	if count != 3 {
		t.Errorf("Expected 3 unique items, got %d", count)
	}
}
