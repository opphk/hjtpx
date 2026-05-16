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
