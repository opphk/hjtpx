package service

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAutomationDetectionService(t *testing.T) {
	service := NewAutomationDetectionService()
	if service == nil {
		t.Fatal("Expected non-nil service")
	}
	if len(service.seleniumPatterns) == 0 {
		t.Error("Expected selenium patterns")
	}
	if len(service.headlessPatterns) == 0 {
		t.Error("Expected headless patterns")
	}
}

func TestDetectSelenium(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		expectMatch bool
	}{
		{
			name:        "selenium in user agent",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Selenium/3.141.59",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "webdriver selenium",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) webdriver/selenium",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "selenium prototype",
			userAgent:   "Selenium.prototype",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "normal browser",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			frontendData: nil,
			expectMatch:  false,
		},
		{
			name:      "webdriver true in frontend data",
			userAgent: "Mozilla/5.0",
			frontendData: map[string]interface{}{
				"navigator": map[string]interface{}{
					"webdriver": true,
				},
			},
			expectMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && result.ToolType != ToolSelenium {
				t.Errorf("Expected tool type selenium, got %s", result.ToolType)
			}
			if tc.expectMatch && len(result.Indicators) == 0 {
				t.Errorf("Expected indicators to be present")
			}
			if tc.expectMatch && result.Score < 35 {
				t.Errorf("Expected score >= 35, got %f", result.Score)
			}
			if !tc.expectMatch && result.ToolType == ToolSelenium {
				t.Errorf("Expected no selenium detection")
			}
		})
	}
}

func TestDetectPhantomJS(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		expectMatch bool
	}{
		{
			name:        "phantomjs user agent",
			userAgent:   "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) PhantomJS/2.1.1 Safari/537.36",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "callPhantom in user agent",
			userAgent:   "callPhantom-test",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:      "phantom object in frontend",
			userAgent: "Mozilla/5.0",
			frontendData: map[string]interface{}{
				"phantom": true,
			},
			expectMatch: false,
		},
		{
			name:        "normal browser",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			frontendData: nil,
			expectMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && result.ToolType != ToolPhantomJS {
				t.Errorf("Expected tool type phantomjs, got %s", result.ToolType)
			}
			if tc.expectMatch && len(result.Indicators) == 0 {
				t.Errorf("Expected indicators to be present")
			}
			if tc.expectMatch && result.Score < 35 {
				t.Errorf("Expected score >= 35, got %f", result.Score)
			}
			if !tc.expectMatch && result.ToolType == ToolPhantomJS {
				t.Errorf("Expected no phantomjs detection")
			}
		})
	}
}

func TestDetectHeadless(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		expectMatch bool
	}{
		{
			name:        "headless chrome user agent",
			userAgent:   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.124 Safari/537.36",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "headless firefox user agent",
			userAgent:   "Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0 Headless",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "headless in user agent",
			userAgent:   "Mozilla/5.0 headless",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "normal chrome",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124",
			frontendData: nil,
			expectMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && result.ToolType != ToolHeadless {
				t.Errorf("Expected tool type headless, got %s", result.ToolType)
			}
			if tc.expectMatch && len(result.Indicators) == 0 {
				t.Errorf("Expected indicators to be present")
			}
			if tc.expectMatch && result.Score < 25 {
				t.Errorf("Expected score >= 25, got %f", result.Score)
			}
			if !tc.expectMatch && result.ToolType == ToolHeadless {
				t.Errorf("Expected no headless detection")
			}
		})
	}
}

func TestDetectPlaywright(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		expectMatch bool
	}{
		{
			name:        "playwright user agent",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Playwright",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "pw.chromium user agent",
			userAgent:   "Mozilla/5.0 pw.chromium",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "normal browser",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			frontendData: nil,
			expectMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && result.ToolType != ToolPlaywright {
				t.Errorf("Expected tool type playwright, got %s", result.ToolType)
			}
			if !tc.expectMatch && result.ToolType == ToolPlaywright {
				t.Errorf("Expected no playwright detection")
			}
		})
	}
}

func TestDetectPuppeteer(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		expectMatch bool
	}{
		{
			name:        "puppeteer user agent",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 puppeteer",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "puppeteer-extra user agent",
			userAgent:   "Mozilla/5.0 puppeteer-extra",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "normal browser",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			frontendData: nil,
			expectMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && result.ToolType != ToolPuppeteer {
				t.Errorf("Expected tool type puppeteer, got %s", result.ToolType)
			}
			if !tc.expectMatch && result.ToolType == ToolPuppeteer {
				t.Errorf("Expected no puppeteer detection")
			}
		})
	}
}

func TestIsAutomatedRequest(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		expectAuto  bool
	}{
		{
			name:        "automated - selenium",
			userAgent:   "Mozilla/5.0 Selenium",
			frontendData: nil,
			expectAuto:  true,
		},
		{
			name:        "automated - phantomjs",
			userAgent:   "PhantomJS/2.1.1",
			frontendData: nil,
			expectAuto:  true,
		},
		{
			name:        "automated - headless",
			userAgent:   "HeadlessChrome",
			frontendData: nil,
			expectAuto:  true,
		},
		{
			name:        "not automated",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			frontendData: nil,
			expectAuto:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.IsAutomatedRequest(req, tc.frontendData)

			if tc.expectAuto && !result {
				t.Errorf("Expected automated request")
			}
			if !tc.expectAuto && result {
				t.Errorf("Expected not automated request")
			}
		})
	}
}

func TestGetRiskLevel(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name     string
		score    float64
		expected string
	}{
		{name: "high risk", score: 45, expected: "high"},
		{name: "high risk boundary", score: 40, expected: "high"},
		{name: "medium risk", score: 25, expected: "medium"},
		{name: "medium risk boundary", score: 20, expected: "medium"},
		{name: "low risk", score: 15, expected: "low"},
		{name: "low risk boundary", score: 10, expected: "low"},
		{name: "no risk", score: 5, expected: "none"},
		{name: "zero risk", score: 0, expected: "none"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.GetRiskLevel(tc.score)
			if result != tc.expected {
				t.Errorf("Expected risk level %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestDetectionConfidence(t *testing.T) {
	service := NewAutomationDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Selenium/3.141.59")

	result := service.DetectAutomationTool(req, nil)

	if result.Confidence <= 0 {
		t.Error("Expected confidence > 0")
	}
	if result.Confidence > 1 {
		t.Error("Expected confidence <= 1")
	}
}

func TestDetectionOrder(t *testing.T) {
	service := NewAutomationDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Selenium/3.141.59 HeadlessChrome")

	result := service.DetectAutomationTool(req, nil)

	if result.ToolType != ToolSelenium {
		t.Errorf("Expected selenium to be detected first, got %s", result.ToolType)
	}
}

func TestEmptyUserAgent(t *testing.T) {
	service := NewAutomationDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)

	result := service.DetectAutomationTool(req, nil)

	if result.IsAutomated {
		t.Error("Expected no automation detection with empty user agent")
	}
}

func TestDetectionMethodsPopulated(t *testing.T) {
	service := NewAutomationDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Selenium")

	result := service.DetectAutomationTool(req, nil)

	if len(result.DetectionMethods) == 0 {
		t.Error("Expected detection methods to be populated")
	}
}
