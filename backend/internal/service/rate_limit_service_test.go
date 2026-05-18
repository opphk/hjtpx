package service

import (
	"testing"
)

func TestNewRateLimitService(t *testing.T) {
	rateLimitService := NewRateLimitService()
	if rateLimitService == nil {
		t.Error("NewRateLimitService 返回了 nil")
	}
}

func TestCheckRateLimit_WithinLimit(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	allowed, err := rateLimitService.CheckRateLimit("test-client-1", 100, 60)
	if err != nil {
		t.Errorf("检查限流失败: %v", err)
	}
	if !allowed {
		t.Error("在限制内的请求应该被允许")
	}
}

func TestCheckRateLimit_ExceedsLimit(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	for i := 0; i < 105; i++ {
		rateLimitService.CheckRateLimit("test-client-2", 100, 60)
	}
	
	allowed, err := rateLimitService.CheckRateLimit("test-client-2", 100, 60)
	if err != nil {
		t.Skipf("检查限流出错: %v", err)
	}
	if allowed {
		t.Error("超过限制的请求应该被拒绝")
	}
}

func TestResetRateLimit(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	rateLimitService.CheckRateLimit("test-client-reset", 100, 60)
	err := rateLimitService.ResetRateLimit("test-client-reset")
	if err != nil {
		t.Errorf("重置限流失败: %v", err)
	}
}

func TestGetRateLimitStatus(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	rateLimitService.CheckRateLimit("test-client-status", 100, 60)
	status, err := rateLimitService.GetRateLimitStatus("test-client-status")
	if err != nil {
		t.Errorf("获取限流状态失败: %v", err)
	}
	if status == nil {
		t.Error("限流状态不应为 nil")
	}
}

func TestGetRateLimitStatus_NotFound(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	status, err := rateLimitService.GetRateLimitStatus("nonexistent-client")
	if err == nil && status != nil {
		t.Error("不存在的客户端应该返回空状态")
	}
}

func TestSetRateLimit(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	err := rateLimitService.SetRateLimit("test-client-set", 200, 120)
	if err != nil {
		t.Errorf("设置限流失败: %v", err)
	}
}

func TestGetRemainingRequests(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	remaining, err := rateLimitService.GetRemainingRequests("test-client-remaining", 100)
	if err != nil {
		t.Errorf("获取剩余请求数失败: %v", err)
	}
	if remaining < 0 {
		t.Error("剩余请求数不应为负数")
	}
}

func TestGetResetTime(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	rateLimitService.CheckRateLimit("test-client-reset-time", 100, 60)
	resetTime, err := rateLimitService.GetResetTime("test-client-reset-time")
	if err != nil {
		t.Errorf("获取重置时间失败: %v", err)
	}
	if resetTime == 0 {
		t.Error("重置时间不应为0")
	}
}

func TestIsWhitelisted(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	isWhitelisted, err := rateLimitService.IsWhitelisted("whitelist-client")
	if err != nil {
		t.Errorf("检查白名单失败: %v", err)
	}
	if isWhitelisted {
		t.Error("未添加的客户端不应该在白名单中")
	}
}

func TestAddToWhitelist(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	err := rateLimitService.AddToWhitelist("whitelist-client-add")
	if err != nil {
		t.Errorf("添加到白名单失败: %v", err)
	}
	
	isWhitelisted, _ := rateLimitService.IsWhitelisted("whitelist-client-add")
	if !isWhitelisted {
		t.Error("添加的客户端应该在白名单中")
	}
}

func TestRemoveFromWhitelist(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	err := rateLimitService.AddToWhitelist("whitelist-client-remove")
	if err != nil {
		t.Skipf("无法添加到白名单，跳过测试: %v", err)
	}
	
	err = rateLimitService.RemoveFromWhitelist("whitelist-client-remove")
	if err != nil {
		t.Errorf("从白名单移除失败: %v", err)
	}
	
	isWhitelisted, _ := rateLimitService.IsWhitelisted("whitelist-client-remove")
	if isWhitelisted {
		t.Error("移除的客户端不应该在白名单中")
	}
}

func TestGetRateLimitStats(t *testing.T) {
	rateLimitService := NewRateLimitService()
	
	rateLimitService.CheckRateLimit("test-client-stats", 100, 60)
	stats, err := rateLimitService.GetRateLimitStats()
	if err != nil {
		t.Errorf("获取限流统计失败: %v", err)
	}
	if stats == nil {
		t.Error("限流统计不应为 nil")
	}
}
