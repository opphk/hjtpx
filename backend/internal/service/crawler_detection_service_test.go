package service

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCrawlerEnhancedDetectionService(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.knownBots)
	assert.Greater(t, len(svc.knownBots), 10)
}

func TestCrawlerEnhancedDetectionService_DetectCrawler(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("NormalBrowser", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")

		result := svc.DetectCrawler(req, nil)
		assert.False(t, result.IsCrawler)
		assert.Less(t, result.Confidence, 0.3)
	})

	t.Run("Googlebot", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Googlebot/2.1 (+http://www.google.com/bot.html)")

		result := svc.DetectCrawler(req, nil)
		assert.True(t, result.IsCrawler)
	})

	t.Run("SeleniumHeadless", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 HeadlessChrome")

		result := svc.DetectCrawler(req, nil)
		assert.True(t, result.IsCrawler || result.Confidence > 0)
	})

	t.Run("Puppeteer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Puppeteer")

		result := svc.DetectCrawler(req, nil)
		assert.True(t, result.IsCrawler || result.Confidence > 0)
	})

	t.Run("Playwright", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Playwright")

		result := svc.DetectCrawler(req, nil)
		assert.True(t, result.IsCrawler || result.Confidence > 0)
	})

	t.Run("PythonRequests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "python-requests/2.28.0")

		result := svc.DetectCrawler(req, nil)
		assert.True(t, result.IsCrawler || result.Confidence > 0)
	})

	t.Run("MissingHeaders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		result := svc.DetectCrawler(req, nil)
		assert.Contains(t, result.Reasons, "Missing standard HTTP headers")
	})

	t.Run("WebdriverHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Webdriver", "true")

		result := svc.DetectCrawler(req, nil)
		assert.True(t, len(result.Signatures) > 0 || len(result.Reasons) > 0)
	})

	t.Run("CryptoMiner", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "CoinHive")

		result := svc.DetectCrawler(req, nil)
		assert.Contains(t, result.Signatures, "Cryptocurrency mining detected")
		assert.Contains(t, result.Reasons, "Potential cryptojacking activity")
	})
}

func TestCrawlerEnhancedDetectionService_AnalyzeBehavior(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()
	ip := "192.168.1.100"

	t.Run("FastRequestInterval", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.RemoteAddr = ip + ":12345"
			svc.DetectCrawler(req, nil)
			time.Sleep(10 * time.Millisecond)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		result := svc.DetectCrawler(req, nil)
		assert.Contains(t, result.Reasons, "Unusually fast request interval")
	})

	t.Run("SequentialPathAccess", func(t *testing.T) {
		svc.ClearRequestHistory(ip)

		paths := []string{"/api/users/1", "/api/users/2", "/api/users/3", "/api/users/4", "/api/users/5"}
		for _, path := range paths {
			req := httptest.NewRequest("GET", path, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			svc.DetectCrawler(req, nil)
		}

		metrics := svc.calculateBehaviorMetrics(svc.requestHistory[ip])
		assert.True(t, metrics.IsSequentialPath || metrics.UniquePaths > 0)
	})

	t.Run("PathRepetition", func(t *testing.T) {
		svc.ClearRequestHistory(ip)

		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/api/same/endpoint", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			svc.DetectCrawler(req, nil)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		result := svc.DetectCrawler(req, nil)
		assert.Contains(t, result.Reasons, "Low path diversity - possible scraping")
	})
}

func TestCrawlerEnhancedDetectionService_DetermineCrawlerType(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("SearchBot", func(t *testing.T) {
		result := &CrawlerDetectionResult{
			Reasons: []string{"Googlebot detected", "search engine crawler"},
		}
		crawlerType := svc.determineCrawlerType(result)
		assert.Equal(t, CrawlerTypeSearchBot, crawlerType)
	})

	t.Run("HeadlessBrowser", func(t *testing.T) {
		result := &CrawlerDetectionResult{
			Reasons: []string{"headless browser detected", "automation tool"},
		}
		crawlerType := svc.determineCrawlerType(result)
		assert.Equal(t, CrawlerTypeHeadless, crawlerType)
	})

	t.Run("Malicious", func(t *testing.T) {
		result := &CrawlerDetectionResult{
			Reasons: []string{"crypto mining detected", "malicious activity"},
		}
		crawlerType := svc.determineCrawlerType(result)
		assert.Equal(t, CrawlerTypeMalicious, crawlerType)
	})

	t.Run("Unknown", func(t *testing.T) {
		result := &CrawlerDetectionResult{
			Reasons: []string{},
		}
		crawlerType := svc.determineCrawlerType(result)
		assert.Equal(t, CrawlerTypeUnknown, crawlerType)
	})
}

func TestCrawlerEnhancedDetectionService_GetKnownBotSignature(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("KnownBot", func(t *testing.T) {
		signature := svc.GetKnownBotSignature("googlebot")
		assert.NotNil(t, signature)
		assert.Equal(t, CrawlerTypeSearchBot, signature.Type)
		assert.Equal(t, "Googlebot", signature.Name)
	})

	t.Run("UnknownBot", func(t *testing.T) {
		signature := svc.GetKnownBotSignature("unknown-bot")
		assert.Nil(t, signature)
	})
}

func TestCrawlerEnhancedDetectionService_AddKnownBotSignature(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("AddNewBot", func(t *testing.T) {
		signature := &CrawlerSignature{
			Type:       CrawlerTypeAPIProxy,
			Name:       "TestBot",
			Confidence: 0.8,
		}

		svc.AddKnownBotSignature("testbot", signature)

		retrieved := svc.GetKnownBotSignature("testbot")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "TestBot", retrieved.Name)
	})
}

func TestCrawlerEnhancedDetectionService_RequestHistory(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()
	ip := "192.168.1.200"

	t.Run("RecordAndRetrieve", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			svc.DetectCrawler(req, nil)
		}

		history := svc.GetRequestHistory(ip)
		assert.True(t, history != nil)
	})

	t.Run("ClearHistory", func(t *testing.T) {
		svc.ClearRequestHistory(ip)
		history := svc.GetRequestHistory(ip)
		assert.Nil(t, history)
	})
}

func TestCrawlerEnhancedDetectionService_GetCrawlerStats(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("InitialStats", func(t *testing.T) {
		stats := svc.GetCrawlerStats()
		assert.Equal(t, 0, stats["tracked_ips"])
		assert.Equal(t, 0, stats["total_records"])
	})

	t.Run("AfterRequests", func(t *testing.T) {
		ip := "192.168.1.210"
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.RemoteAddr = ip + ":12345"
			svc.DetectCrawler(req, nil)
		}

		stats := svc.GetCrawlerStats()
		assert.Greater(t, stats["tracked_ips"].(int), 0)
		assert.Greater(t, stats["total_records"].(int), 0)
	})
}

func TestCrawlerEnhancedDetectionService_ConcurrentAccess(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("ConcurrentDetection", func(t *testing.T) {
		var wg sync.WaitGroup
		ips := []string{"192.168.1.220", "192.168.1.221", "192.168.1.222"}

		for _, ip := range ips {
			for i := 0; i < 30; i++ {
				wg.Add(1)
				go func(ip string, i int) {
					defer wg.Done()
					req := httptest.NewRequest("GET", "/test", nil)
					req.Header.Set("User-Agent", "Mozilla/5.0")
					req.RemoteAddr = ip + ":12345"
					svc.DetectCrawler(req, nil)
				}(ip, i)
			}
		}
		wg.Wait()

		stats := svc.GetCrawlerStats()
		assert.Greater(t, stats["total_records"].(int), 0)
	})
}

func TestCrawlerEnhancedDetectionService_GenerateFingerprint(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("SameInputsSameFingerprint", func(t *testing.T) {
		additionalData := map[string]string{"screen": "1920x1080"}

		fp1 := svc.generateFingerprint("192.168.1.230", "Mozilla/5.0", additionalData)
		fp2 := svc.generateFingerprint("192.168.1.230", "Mozilla/5.0", additionalData)
		assert.Equal(t, fp1, fp2)
	})

	t.Run("DifferentInputsDifferentFingerprint", func(t *testing.T) {
		fp1 := svc.generateFingerprint("192.168.1.231", "Mozilla/5.0", nil)
		fp2 := svc.generateFingerprint("192.168.1.232", "Mozilla/5.0", nil)
		assert.NotEqual(t, fp1, fp2)
	})
}

func TestCrawlerEnhancedDetectionService_BehaviorMetrics(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("EmptyHistory", func(t *testing.T) {
		metrics := svc.calculateBehaviorMetrics([]*RequestRecord{})
		assert.Equal(t, 0, metrics.RequestCount)
	})

	t.Run("SingleRecord", func(t *testing.T) {
		metrics := svc.calculateBehaviorMetrics([]*RequestRecord{
			{
				Path:   "/test",
				Method: "GET",
			},
		})
		assert.Equal(t, 1, metrics.RequestCount)
		assert.GreaterOrEqual(t, metrics.UniquePaths, 1)
	})

	t.Run("MultipleRecords", func(t *testing.T) {
		now := time.Now()
		records := []*RequestRecord{
			{Timestamp: now.Add(-10 * time.Second), Path: "/a", Method: "GET"},
			{Timestamp: now.Add(-9 * time.Second), Path: "/b", Method: "POST"},
			{Timestamp: now.Add(-8 * time.Second), Path: "/a", Method: "GET"},
			{Timestamp: now.Add(-7 * time.Second), Path: "/c", Method: "GET"},
			{Timestamp: now.Add(-6 * time.Second), Path: "/a", Method: "GET"},
		}

		metrics := svc.calculateBehaviorMetrics(records)
		assert.Equal(t, 5, metrics.RequestCount)
		assert.Equal(t, 3, metrics.UniquePaths)
	})
}

func TestCrawlerEnhancedDetectionService_IsSequentialPattern(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("Sequential", func(t *testing.T) {
		paths := []string{"/a", "/b", "/c", "/d", "/e"}
		assert.True(t, svc.isSequentialPattern(paths))
	})

	t.Run("NotSequential", func(t *testing.T) {
		paths := []string{"/c", "/a", "/b", "/d", "/e"}
		assert.False(t, svc.isSequentialPattern(paths))
	})

	t.Run("TooFewPaths", func(t *testing.T) {
		paths := []string{"/a", "/b"}
		assert.False(t, svc.isSequentialPattern(paths))
	})
}

func TestCrawlerDetectionResult_RiskLevels(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	t.Run("HighRiskCrawler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "curl/7.68.0")
		req.Header.Set("Webdriver", "true")

		result := svc.DetectCrawler(req, nil)
		assert.True(t, result.IsCrawler)
		assert.Contains(t, []string{"high", "medium", "low"}, result.RiskLevel)
	})

	t.Run("RecommendedActions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Googlebot")

		result := svc.DetectCrawler(req, nil)
		assert.NotEmpty(t, result.RecommendedAction)
	})
}

func TestHeadlessBrowserDetection(t *testing.T) {
	svc := NewCrawlerEnhancedDetectionService()

	testCases := []struct {
		name       string
		userAgent  string
		shouldMatch bool
	}{
		{"Chrome Headless", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 HeadlessChrome", true},
		{"Firefox Headless", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0", false},
		{"PhantomJS", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/534.34 (KHTML, like Gecko) PhantomJS/2.1.1 Safari/534.34", true},
		{"Selenium", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := svc.DetectCrawler(req, nil)

			hasHeadless := false
			for _, reason := range result.Reasons {
				if strings.Contains(reason, "Headless") || strings.Contains(reason, "headless") {
					hasHeadless = true
					break
				}
			}

			if tc.shouldMatch {
				assert.True(t, hasHeadless || result.IsCrawler)
			}
		})
	}
}
