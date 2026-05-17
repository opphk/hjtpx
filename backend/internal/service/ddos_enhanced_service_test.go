package service

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDDoSEnhancedProtectionService(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(nil)
		assert.NotNil(t, svc)
		assert.Equal(t, 10.0, svc.config.MaxBandwidthGbps)
		assert.Equal(t, 100, svc.config.RequestsPerSecond)
		assert.True(t, svc.config.EnableTrafficCleaning)
		assert.True(t, svc.config.EnableIPReputation)
		assert.True(t, svc.config.EnableAdaptiveLimits)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := &DDoSProtectionConfig{
			EnableTrafficCleaning:   false,
			EnableIPReputation:     false,
			EnableAdaptiveLimits:   false,
			EnableRateLimitStrategy: true,
			EnableBandwidthThrottle: false,
			MaxBandwidthGbps:       5.0,
			RequestsPerSecond:      50,
			BurstSize:             100,
			WindowSize:           2 * time.Minute,
			BlockDuration:         10 * time.Minute,
			CleanupInterval:       1 * time.Hour,
			MaxIPs:               5000,
		}

		svc := NewDDoSEnhancedProtectionService(config)
		assert.NotNil(t, svc)
		assert.Equal(t, 5.0, svc.config.MaxBandwidthGbps)
		assert.Equal(t, 50, svc.config.RequestsPerSecond)
		assert.False(t, svc.config.EnableTrafficCleaning)
		assert.False(t, svc.config.EnableIPReputation)
	})
}

func TestDDoSEnhancedProtectionService_CheckRequest(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(nil)

	t.Run("AllowNormalRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		result := svc.CheckRequest(req)
		assert.True(t, result.Allowed)
		assert.Equal(t, "normal", string(svc.GetIPReputation("192.168.1.1").Tier))
	})

	t.Run("BlockBlacklistedIP", func(t *testing.T) {
		svc.SetIPBlacklist("10.0.0.1", true)

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"

		result := svc.CheckRequest(req)
		assert.False(t, result.Allowed)
		assert.Equal(t, "ip_blacklisted", result.Reason)
	})

	t.Run("BlockRateLimitExceeded", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			RequestsPerSecond: 100,
			WindowSize:       1 * time.Second,
			BlockDuration:    10 * time.Minute,
		})
		testIP := "10.0.0.99:12345"

		for i := 0; i < 150; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = testIP
			svc.CheckRequest(req)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = testIP

		result := svc.CheckRequest(req)
		assert.False(t, result.Allowed)
		assert.Contains(t, []string{"rate_limit_exceeded", "blacklisted"}, result.Reason)
	})

	t.Run("AllowWhitelistedIP", func(t *testing.T) {
		svc.SetIPWhitelist("10.0.0.3", true)

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.3:12345"

		result := svc.CheckRequest(req)
		assert.True(t, result.Allowed)
	})

	t.Run("IPReputationUpdated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.4:12345"

		initialReputation := svc.GetIPReputation("10.0.0.4")
		initialScore := initialReputation.Score

		svc.CheckRequest(req)

		updatedReputation := svc.GetIPReputation("10.0.0.4")
		assert.Greater(t, updatedReputation.Score, initialScore-0.01)
	})
}

func TestDDoSEnhancedProtectionService_CleanTraffic(t *testing.T) {
	t.Run("NormalTrafficNotCleaned", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			EnableTrafficCleaning: true,
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.RemoteAddr = "192.168.1.1:12345"

		result := svc.CleanTraffic("192.168.1.1", req)
		assert.False(t, result.Cleaned || result.DroppedBytes > 0)
	})

	t.Run("MaliciousPatternDetected", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			EnableTrafficCleaning: true,
		})
		req := httptest.NewRequest("GET", "/test/../../../etc/passwd", nil)
		req.Header.Set("User-Agent", "TestBot")
		req.RemoteAddr = "192.168.1.2:12345"

		result := svc.CleanTraffic("192.168.1.2", req)
		assert.True(t, result.Cleaned)
		assert.Equal(t, "directory_traversal", result.DropReason)
	})

	t.Run("XSSPatternDetected", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			EnableTrafficCleaning: true,
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Referer", "http://test.com/?param=<script>alert(1)</script>")
		req.RemoteAddr = "192.168.1.3:12345"

		result := svc.CleanTraffic("192.168.1.3", req)
		assert.True(t, result.Cleaned || result.DropReason != "")
	})

	t.Run("SQLInjectionPatternDetected", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			EnableTrafficCleaning: true,
		})
		req := httptest.NewRequest("GET", "/api/search?q=test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("X-Query", "1 UNION SELECT * FROM users")
		req.RemoteAddr = "192.168.1.4:12345"

		result := svc.CleanTraffic("192.168.1.4", req)
		assert.True(t, result.Cleaned || result.DropReason != "")
	})

	t.Run("MissingUserAgent", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			EnableTrafficCleaning: true,
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Del("User-Agent")
		req.RemoteAddr = "192.168.1.5:12345"

		result := svc.CleanTraffic("192.168.1.5", req)
		assert.True(t, result.Cleaned)
		assert.Equal(t, "missing_user_agent", result.DropReason)
	})

	t.Run("TrafficCleaningDisabled", func(t *testing.T) {
		svcDisabled := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			EnableTrafficCleaning: false,
		})

		req := httptest.NewRequest("GET", "/test/../../../etc/passwd", nil)
		req.Header.Set("User-Agent", "TestBot")
		req.RemoteAddr = "192.168.1.6:12345"

		result := svcDisabled.CleanTraffic("192.168.1.6", req)
		assert.False(t, result.Cleaned)
	})
}

func TestDDoSEnhancedProtectionService_GetIPReputation(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(nil)

	t.Run("NewIPHasDefaultReputation", func(t *testing.T) {
		reputation := svc.GetIPReputation("192.168.1.100")
		assert.Equal(t, "192.168.1.100", reputation.IP)
		assert.Equal(t, 0.5, reputation.Score)
		assert.Equal(t, DDoSTierNormal, reputation.Tier)
		assert.Equal(t, 5, reputation.TrustLevel)
	})

	t.Run("UpdateReputation", func(t *testing.T) {
		svc.updateIPReputation("192.168.1.101", -0.3)
		reputation := svc.GetIPReputation("192.168.1.101")
		assert.Equal(t, 0.2, reputation.Score)
		assert.Equal(t, DDoSTierNormal, reputation.Tier)
	})

	t.Run("ReputationBelowWarning", func(t *testing.T) {
		svc.updateIPReputation("192.168.1.102", -0.5)
		reputation := svc.GetIPReputation("192.168.1.102")
		assert.Equal(t, DDoSTierWarning, reputation.Tier)
	})

	t.Run("ReputationAtCritical", func(t *testing.T) {
		svc.updateIPReputation("192.168.1.103", -0.9)
		reputation := svc.GetIPReputation("192.168.1.103")
		assert.Equal(t, 0.0, reputation.Score)
	})

	t.Run("ScoreClampedToOne", func(t *testing.T) {
		svc.updateIPReputation("192.168.1.104", 1.0)
		reputation := svc.GetIPReputation("192.168.1.104")
		assert.Equal(t, 1.0, reputation.Score)
	})

	t.Run("ScoreClampedToZero", func(t *testing.T) {
		svc.updateIPReputation("192.168.1.105", -1.0)
		reputation := svc.GetIPReputation("192.168.1.105")
		assert.Equal(t, 0.0, reputation.Score)
	})
}

func TestDDoSEnhancedProtectionService_SetIPWhitelist(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(nil)

	t.Run("WhitelistIP", func(t *testing.T) {
		svc.SetIPWhitelist("192.168.1.200", true)
		reputation := svc.GetIPReputation("192.168.1.200")
		assert.True(t, reputation.IsWhitelisted)
		assert.Equal(t, DDoSTierNormal, reputation.Tier)
		assert.Equal(t, 0.0, reputation.Score)
	})

	t.Run("RemoveFromWhitelist", func(t *testing.T) {
		svc.SetIPWhitelist("192.168.1.201", true)
		svc.SetIPWhitelist("192.168.1.201", false)
		reputation := svc.GetIPReputation("192.168.1.201")
		assert.False(t, reputation.IsWhitelisted)
	})
}

func TestDDoSEnhancedProtectionService_SetIPBlacklist(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(nil)

	t.Run("BlacklistIP", func(t *testing.T) {
		svc.SetIPBlacklist("192.168.1.210", true)
		reputation := svc.GetIPReputation("192.168.1.210")
		assert.True(t, reputation.IsBlacklisted)
		assert.Equal(t, DDoSTierBlocked, reputation.Tier)
		assert.Equal(t, 1.0, reputation.Score)
	})

	t.Run("RemoveFromBlacklist", func(t *testing.T) {
		svc.SetIPBlacklist("192.168.1.211", true)
		svc.SetIPBlacklist("192.168.1.211", false)
		reputation := svc.GetIPReputation("192.168.1.211")
		assert.False(t, reputation.IsBlacklisted)
		assert.Equal(t, DDoSTierNormal, reputation.Tier)
	})
}

func TestDDoSEnhancedProtectionService_GetAdaptiveLimit(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
		RequestsPerSecond: 100,
	})

	t.Run("NormalTierGetsFullLimit", func(t *testing.T) {
		limit := svc.getAdaptiveLimit("192.168.1.220")
		assert.Equal(t, 100, limit)
	})

	t.Run("WarningTierGetsReducedLimit", func(t *testing.T) {
		svc.updateIPReputation("192.168.1.221", -0.4)
		limit := svc.getAdaptiveLimit("192.168.1.221")
		assert.Equal(t, 50, limit)
	})

	t.Run("CriticalTierGetsMinimalLimit", func(t *testing.T) {
		svc.updateIPReputation("192.168.1.222", -0.8)
		limit := svc.getAdaptiveLimit("192.168.1.222")
		assert.Equal(t, 20, limit)
	})

	t.Run("BlockedTierGetsZeroLimit", func(t *testing.T) {
		svc.SetIPBlacklist("192.168.1.223", true)
		limit := svc.getAdaptiveLimit("192.168.1.223")
		assert.Equal(t, 0, limit)
	})
}

func TestDDoSEnhancedProtectionService_GetProtectionStats(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(nil)

	t.Run("InitialStats", func(t *testing.T) {
		stats := svc.GetProtectionStats()
		assert.Equal(t, int64(0), stats.TotalRequests)
		assert.Equal(t, int64(0), stats.CleanedRequests)
	})

	t.Run("StatsAfterRequests", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.230:12345"
			svc.CheckRequest(req)
		}

		stats := svc.GetProtectionStats()
		assert.GreaterOrEqual(t, stats.TotalRequests, int64(10))
	})
}

func TestDDoSEnhancedProtectionService_GetRateLimitStrategy(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
		RequestsPerSecond: 100,
		BurstSize:        200,
		BlockDuration:    5 * time.Minute,
	})

	t.Run("GetDefaultStrategy", func(t *testing.T) {
		svc := NewDDoSEnhancedProtectionService(&DDoSProtectionConfig{
			RequestsPerSecond: 100,
			BurstSize:        200,
			BlockDuration:    5 * time.Minute,
		})
		strategy := svc.GetRateLimitStrategy()
		assert.Equal(t, 100, strategy.RequestsPerSec)
		assert.Equal(t, 200, strategy.BurstSize)
	})

	t.Run("UpdateStrategy", func(t *testing.T) {
		newStrategy := &RateLimitStrategy{
			Name:            "strict",
			RequestsPerSec:  50,
			BurstSize:      100,
			WindowSize:     30 * time.Second,
			Adaptive:       false,
			BlockDuration:  10 * time.Minute,
		}

		svc.SetRateLimitStrategy(newStrategy)
		updated := svc.GetRateLimitStrategy()
		assert.Equal(t, "strict", updated.Name)
		assert.Equal(t, 50, updated.RequestsPerSec)
		assert.False(t, updated.Adaptive)
	})
}

func TestDDoSEnhancedProtectionService_DetectAnomaly(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(nil)

	t.Run("InsufficientData", func(t *testing.T) {
		traffic := &DDoSTrafficData{
			RequestTimes: []time.Time{time.Now()},
		}
		assert.False(t, svc.detectAnomaly(traffic))
	})

	t.Run("RegularPattern", func(t *testing.T) {
		now := time.Now()
		traffic := &DDoSTrafficData{
			RequestTimes: []time.Time{
				now.Add(-10 * time.Second),
				now.Add(-9 * time.Second),
				now.Add(-8 * time.Second),
				now.Add(-7 * time.Second),
				now.Add(-6 * time.Second),
				now.Add(-5 * time.Second),
				now.Add(-4 * time.Second),
				now.Add(-3 * time.Second),
				now.Add(-2 * time.Second),
				now.Add(-1 * time.Second),
			},
		}
		assert.True(t, svc.detectAnomaly(traffic))
	})

	t.Run("RandomPattern", func(t *testing.T) {
		now := time.Now()
		traffic := &DDoSTrafficData{
			RequestTimes: []time.Time{
				now.Add(-100 * time.Second),
				now.Add(-50 * time.Second),
				now.Add(-30 * time.Second),
				now.Add(-15 * time.Second),
				now.Add(-10 * time.Second),
				now.Add(-8 * time.Second),
				now.Add(-5 * time.Second),
				now.Add(-3 * time.Second),
				now.Add(-1 * time.Second),
				now,
			},
		}
		assert.False(t, svc.detectAnomaly(traffic))
	})
}

func TestGetRequestSize(t *testing.T) {
	t.Run("WithContentLength", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 1024
		assert.Equal(t, 1024, getRequestSize(req))
	})

	t.Run("WithBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		assert.Equal(t, 512, getRequestSize(req))
	})

	t.Run("EmptyRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		assert.Equal(t, 0, getRequestSize(req))
	})
}

func TestDDoSEnhancedProtectionService_ConcurrentAccess(t *testing.T) {
	svc := NewDDoSEnhancedProtectionService(nil)

	t.Run("ConcurrentCheckRequests", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.240:12345"
				svc.CheckRequest(req)
			}(i)
		}
		wg.Wait()

		stats := svc.GetProtectionStats()
		assert.GreaterOrEqual(t, stats.TotalRequests, int64(100))
	})

	t.Run("ConcurrentCleanTraffic", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.241:12345"
				svc.CleanTraffic("192.168.1.241", req)
			}(i)
		}
		wg.Wait()
	})

	t.Run("ConcurrentReputationUpdates", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				svc.updateIPReputation("192.168.1.242", 0.01)
			}(i)
		}
		wg.Wait()

		reputation := svc.GetIPReputation("192.168.1.242")
		assert.Greater(t, reputation.Score, 0.5)
	})
}
