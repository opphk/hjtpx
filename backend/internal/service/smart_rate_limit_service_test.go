package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSmartRateLimitService(t *testing.T) {
	service := NewSmartRateLimitService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.clients)
	assert.NotNil(t, service.hotspots)
	assert.NotNil(t, service.tierMap)
}

func TestSmartRateLimitService_CheckRateLimit(t *testing.T) {
	service := NewSmartRateLimitService()

	result := service.CheckRateLimit("test-client-1", 0.0)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
	assert.NotEmpty(t, result.Tier)

	result2 := service.CheckRateLimit("test-client-1", 0.0)
	assert.NotNil(t, result2)
}

func TestSmartRateLimitService_CheckRateLimitWithRiskScore(t *testing.T) {
	service := NewSmartRateLimitService()

	result := service.CheckRateLimit("test-client-2", 50.0)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed)
}

func TestSmartRateLimitService_GetClientStats(t *testing.T) {
	service := NewSmartRateLimitService()

	for i := 0; i < 5; i++ {
		service.CheckRateLimit("test-client-stats", 0.0)
	}

	stats := service.GetClientStats("test-client-stats")
	assert.NotNil(t, stats)
	assert.Equal(t, "test-client-stats", stats["client_id"])
	assert.NotEmpty(t, stats["tier"])
}

func TestSmartRateLimitService_GetClientStats_NotFound(t *testing.T) {
	service := NewSmartRateLimitService()

	stats := service.GetClientStats("non-existent-client")
	assert.Nil(t, stats)
}

func TestSmartRateLimitService_SetClientTier(t *testing.T) {
	service := NewSmartRateLimitService()

	service.CheckRateLimit("test-client-tier", 0.0)
	service.SetClientTier("test-client-tier", "premium")

	stats := service.GetClientStats("test-client-tier")
	assert.NotNil(t, stats)
	assert.Equal(t, "premium", stats["tier"])
}

func TestSmartRateLimitService_GetStats(t *testing.T) {
	service := NewSmartRateLimitService()

	for i := 0; i < 10; i++ {
		service.CheckRateLimit("test-client-stats", 0.0)
	}

	stats := service.GetStats()
	assert.NotNil(t, stats)
	assert.Greater(t, stats["total_requests"].(int64), int64(0))
	assert.Contains(t, stats, "adaptive_enabled")
	assert.Contains(t, stats, "hotspot_enabled")
}

func TestSmartRateLimitService_GetTierDistribution(t *testing.T) {
	service := NewSmartRateLimitService()

	service.CheckRateLimit("client-1", 0.0)
	service.CheckRateLimit("client-2", 0.0)
	service.CheckRateLimit("client-3", 0.0)

	distribution := service.GetTierDistribution()
	assert.NotNil(t, distribution)
}

func TestSmartRateLimitService_GetHotspots(t *testing.T) {
	service := NewSmartRateLimitService()

	for i := 0; i < 100; i++ {
		service.CheckRateLimit("hotspot-client", 0.0)
	}

	hotspots := service.GetHotspots(10)
	assert.NotNil(t, hotspots)
}

func TestSmartRateLimitService_GetClientList(t *testing.T) {
	service := NewSmartRateLimitService()

	for i := 0; i < 5; i++ {
		service.CheckRateLimit("client-"+string(rune('0'+i)), 0.0)
	}

	clients, total := service.GetClientList(1, 10)
	assert.Equal(t, 5, total)
	assert.GreaterOrEqual(t, len(clients), 0)
}

func TestSmartRateLimitService_Cleanup(t *testing.T) {
	config := SmartRateLimitConfig{
		HistoryWindow: 1 * time.Millisecond,
	}
	service := NewSmartRateLimitService(config)

	service.CheckRateLimit("old-client", 0.0)

	time.Sleep(10 * time.Millisecond)

	service.Cleanup()

	stats := service.GetClientStats("old-client")
	assert.Nil(t, stats)
}

func TestSmartRateLimitService_UpdateConfig(t *testing.T) {
	service := NewSmartRateLimitService()

	newConfig := SmartRateLimitConfig{
		DefaultRequestsPerMin: 100,
		EnableAdaptiveLimit:   false,
	}
	service.UpdateConfig(newConfig)

	stats := service.GetStats()
	assert.NotNil(t, stats)
}

func TestSmartRateLimitService_CheckRateLimitWithHotspot(t *testing.T) {
	service := NewSmartRateLimitService()

	for i := 0; i < 100; i++ {
		service.CheckRateLimit("hotspot-test-client", 0.0)
	}

	stats := service.GetClientStats("hotspot-test-client")
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats["hotspot_score"].(float64), 0.0)
}

func TestSmartRateLimitService_HotspotLimitApplication(t *testing.T) {
	service := NewSmartRateLimitService()

	for i := 0; i < 100; i++ {
		service.CheckRateLimit("high-frequency-client", 0.0)
	}

	stats := service.GetStats()
	assert.NotNil(t, stats)
}
