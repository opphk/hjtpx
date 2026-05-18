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

			if tc.expectMatch && result.ToolType != "selenium" {
				t.Errorf("Expected tool type selenium, got %s", result.ToolType)
			}
			if tc.expectMatch && len(result.Evidence) == 0 {
				t.Errorf("Expected evidence to be present")
			}
			if tc.expectMatch && result.RiskScore < 35 {
				t.Errorf("Expected risk score >= 35, got %f", result.RiskScore)
			}
			if !tc.expectMatch && result.ToolType == "selenium" {
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
				"phantom": map[string]interface{}{},
			},
			expectMatch: true,
		},
		{
			name:        "normal browser",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			frontendData: nil,
			expectMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && result.ToolType != "phantomjs" {
				t.Errorf("Expected tool type phantomjs, got %s", result.ToolType)
			}
			if tc.expectMatch && len(result.Evidence) == 0 {
				t.Errorf("Expected evidence to be present")
			}
			if tc.expectMatch && result.RiskScore < 40 {
				t.Errorf("Expected risk score >= 40, got %f", result.RiskScore)
			}
			if !tc.expectMatch && result.ToolType == "phantomjs" {
				t.Errorf("Expected no phantomjs detection")
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
			userAgent:   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.124 Safari/537.36",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "chrome-headless",
			userAgent:   "Mozilla/5.0 chrome-headless",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:      "cdc object in frontend",
			userAgent: "Mozilla/5.0",
			frontendData: map[string]interface{}{
				"cdc_object": true,
			},
			expectMatch: true,
		},
		{
			name:        "normal chrome",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124",
			frontendData: nil,
			expectMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && result.ToolType != "puppeteer" {
				t.Errorf("Expected tool type puppeteer, got %s", result.ToolType)
			}
			if tc.expectMatch && len(result.Evidence) == 0 {
				t.Errorf("Expected evidence to be present")
			}
			if tc.expectMatch && result.RiskScore < 38 {
				t.Errorf("Expected risk score >= 38, got %f", result.RiskScore)
			}
			if !tc.expectMatch && result.ToolType == "puppeteer" {
				t.Errorf("Expected no puppeteer detection")
			}
		})
	}
}

func TestDetectHeadlessBrowser(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		expectMatch bool
	}{
		{
			name:        "headless chrome",
			userAgent:   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.124 Safari/537.36",
			frontendData: nil,
			expectMatch:  true,
		},
		{
			name:        "headless firefox",
			userAgent:   "Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0",
			frontendData: map[string]interface{}{
				"webgl_renderer": "SwiftShader",
			},
			expectMatch: true,
		},
		{
			name:        "no plugins",
			userAgent:   "Mozilla/5.0",
			frontendData: map[string]interface{}{
				"plugins": []interface{}{},
			},
			expectMatch: true,
		},
		{
			name:        "zero screen size",
			userAgent:   "Mozilla/5.0",
			frontendData: map[string]interface{}{
				"screen_size": "0x0",
			},
			expectMatch: true,
		},
		{
			name:        "normal browser",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			frontendData: map[string]interface{}{
				"plugins": []interface{}{"Chrome PDF Plugin"},
			},
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && !result.HeadlessDetected {
				t.Errorf("Expected headless detection, got HeadlessDetected=false")
			}
			if !tc.expectMatch && result.HeadlessDetected {
				t.Errorf("Expected no headless detection, got HeadlessDetected=true")
			}
		})
	}
}

func TestDetectDebugger(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		frontendData map[string]interface{}
		expectMatch bool
	}{
		{
			name: "debugger detected",
			frontendData: map[string]interface{}{
				"debugger_detected": true,
			},
			expectMatch: true,
		},
		{
			name: "devtools open",
			frontendData: map[string]interface{}{
				"devtools_open": true,
			},
			expectMatch: true,
		},
		{
			name: "debugger statement",
			frontendData: map[string]interface{}{
				"debugger_statement": true,
			},
			expectMatch: true,
		},
		{
			name: "timing anomaly",
			frontendData: map[string]interface{}{
				"timing_anomaly": 100.0,
			},
			expectMatch: true,
		},
		{
			name:        "normal",
			frontendData: nil,
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.expectMatch && !result.DebuggerDetected {
				t.Errorf("Expected debugger detection, got DebuggerDetected=false")
			}
			if !tc.expectMatch && result.DebuggerDetected {
				t.Errorf("Expected no debugger detection, got DebuggerDetected=true")
			}
		})
	}
}

func TestBehavioralPatterns(t *testing.T) {
	service := NewAutomationDetectionService()
	ip := "192.168.1.200"

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = ip + ":12345"

	for i := 0; i < 10; i++ {
		result := service.DetectAutomationTool(req, nil)
		time.Sleep(50 * time.Millisecond)

		if i >= 5 {
			if result.AutoBehavioralIndicators.IsHumanLike {
				t.Logf("Warning: Expected non-human behavior for rapid requests")
			}
		}
	}
}

func TestSessionBehaviorAnalysis(t *testing.T) {
	service := NewAutomationDetectionService()
	sessionID := "test-session-123"

	now := time.Now()
	for i := 0; i < 20; i++ {
		service.RecordSessionBehavior(sessionID, "/api/test", now.Add(time.Duration(i)*100*time.Millisecond))
	}

	analysis := service.GetSessionAnalysis(sessionID)
	if analysis == nil {
		t.Fatal("Expected session analysis")
	}
	if analysis.TotalRequests != 20 {
		t.Errorf("Expected 20 requests, got %d", analysis.TotalRequests)
	}
}

func TestRiskScoreCalculation(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name        string
		userAgent   string
		frontendData map[string]interface{}
		minScore    float64
	}{
		{
			name:        "multiple indicators",
			userAgent:   "Selenium HeadlessChrome",
			frontendData: map[string]interface{}{
				"webdriver": true,
				"plugins":   []interface{}{},
			},
			minScore: 80,
		},
		{
			name:        "single indicator",
			userAgent:   "Mozilla/5.0",
			frontendData: map[string]interface{}{
				"webdriver": true,
			},
			minScore: 30,
		},
		{
			name:        "no indicators",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			frontendData: nil,
			minScore:    -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, tc.frontendData)

			if tc.minScore >= 0 && result.RiskScore < tc.minScore {
				t.Errorf("Expected risk score >= %.2f, got %.2f", tc.minScore, result.RiskScore)
			}
		})
	}
}

func TestCleanupOldRecords(t *testing.T) {
	service := NewAutomationDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.300:12345"
	req.Header.Set("User-Agent", "Mozilla/5.0")

	service.DetectAutomationTool(req, nil)

	service.CleanupOldRecords()

	service.mu.RLock()
	if len(service.behaviorPatterns) != 0 {
		t.Error("Expected behavior patterns to be cleaned")
	}
	service.mu.RUnlock()
}

func TestConfidenceCalculation(t *testing.T) {
	service := NewAutomationDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Selenium HeadlessChrome")
	req.RemoteAddr = "192.168.1.400:12345"

	frontendData := map[string]interface{}{
		"webdriver":       true,
		"debugger_detected": true,
	}

	result := service.DetectAutomationTool(req, frontendData)

	if result.Confidence < 0.5 {
		t.Errorf("Expected confidence >= 0.5 for multiple indicators, got %.2f", result.Confidence)
	}
}

func TestToolTypeDetection(t *testing.T) {
	service := NewAutomationDetectionService()

	testCases := []struct {
		name      string
		userAgent string
		toolType  string
	}{
		{"selenium", "Selenium/3.141.59", "selenium"},
		{"phantomjs", "PhantomJS/2.1.1", "phantomjs"},
		{"puppeteer", "HeadlessChrome/91.0.4472.124", "puppeteer"},
		{"playwright", "playwright-test", "playwright"},
		{"webdriver", "webdriver-test", "webdriver"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectAutomationTool(req, nil)

			if result.ToolType != tc.toolType {
				t.Errorf("Expected tool type %s, got %s", tc.toolType, result.ToolType)
			}
		})
	}
}