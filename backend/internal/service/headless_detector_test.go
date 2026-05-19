package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewHeadlessDetector(t *testing.T) {
	detector := NewHeadlessDetector()

	if detector == nil {
		t.Fatal("Expected non-nil detector")
	}

	if detector.config == nil {
		t.Error("Expected config to be initialized")
	}

	if detector.stats == nil {
		t.Error("Expected stats to be initialized")
	}

	if detector.detectionCache == nil {
		t.Error("Expected detectionCache to be initialized")
	}

	if detector.sessionData == nil {
		t.Error("Expected sessionData to be initialized")
	}
}

func TestNewHeadlessDetectorWithConfig(t *testing.T) {
	config := &model.HeadlessDetectionConfig{
		EnableNavigatorDetection:   true,
		EnablePluginDetection:     true,
		EnableAutomationDetection: true,
		EnableEnvironmentDetection: true,
		HeadlessThreshold:         60.0,
		StrictMode:               true,
	}

	detector := NewHeadlessDetectorWithConfig(config)

	if detector == nil {
		t.Fatal("Expected non-nil detector")
	}

	if detector.config.HeadlessThreshold != 60.0 {
		t.Errorf("Expected threshold 60.0, got %.2f", detector.config.HeadlessThreshold)
	}

	if !detector.config.StrictMode {
		t.Error("Expected strict mode to be enabled")
	}
}

func TestDetectHeadlessChrome(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.124 Safari/537.36")

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver":            true,
			"languages":            []interface{}{},
			"plugins":              []interface{}{},
			"platform":             "Linux x86_64",
			"deviceMemory":         0.0,
			"hardwareConcurrency":  0.0,
		},
		"webgl_renderer": "SwiftShader SwiftShader 4.0.0 (Build 20.0.0)",
		"screen": map[string]interface{}{
			"width":  0,
			"height": 0,
		},
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsHeadless {
		t.Error("Expected headless to be detected")
	}

	if result.RiskScore < 50 {
		t.Errorf("Expected risk score >= 50, got %.2f", result.RiskScore)
	}

	if len(result.Indicators) == 0 {
		t.Error("Expected at least one indicator")
	}

	found := false
	for _, method := range result.DetectionMethods {
		if method == "automation_tool_check" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected automation_tool_check in detection methods")
	}
}

func TestDetectSelenium(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124 Safari/537.36 Selenium/3.141.59")

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver":           true,
			"languages":          []interface{}{"en-US", "en"},
			"plugins":            []interface{}{},
			"platform":           "Win32",
			"deviceMemory":       8.0,
			"hardwareConcurrency": 8.0,
		},
		"__selenium_evaluate":       true,
		"__webdriver_script_fn":     true,
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsHeadless {
		t.Error("Expected headless/automation to be detected for Selenium")
	}

	if result.DetectedTool != "selenium" {
		t.Errorf("Expected detected tool to be selenium, got %s", result.DetectedTool)
	}
}

func TestDetectPuppeteer(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/100.0.4896.75 Safari/537.36")

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver":    true,
			"languages":    []interface{}{},
			"plugins":      []interface{}{},
			"platform":     "",
			"deviceMemory": 0,
		},
		"$cdc_asdjflasutopfhvcZLmcfl_": true,
		"webgl_renderer":               "ANGLE (Intel, Intel(R) UHD Graphics DirectX FL9_3 0xfff7)",
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsHeadless {
		t.Error("Expected headless to be detected for Puppeteer")
	}

	if result.DetectedTool != "puppeteer" && result.DetectedTool != "headless_browser" {
		t.Errorf("Expected detected tool to be puppeteer or headless_browser, got %s", result.DetectedTool)
	}
}

func TestDetectPlaywright(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/100.0.4896.75 Safari/537.36")

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver":           true,
			"languages":          []interface{}{},
			"plugins":            []interface{}{},
			"platform":           "",
			"deviceMemory":       0,
			"hardwareConcurrency": 0,
		},
		"__playwright__":   true,
		"__pw_api_hooks__": true,
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsHeadless {
		t.Error("Expected headless to be detected for Playwright")
	}

	found := false
	for _, check := range result.AutomationChecks {
		if check.ToolName == "playwright" && check.Detected {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected playwright to be detected in automation checks")
	}
}

func TestNormalBrowser(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver":           false,
			"languages":          []interface{}{"en-US", "en", "zh-CN"},
			"plugins":            []interface{}{"Chrome PDF Plugin", "Chrome Native Viewer"},
			"platform":           "Win32",
			"deviceMemory":       8.0,
			"hardwareConcurrency": 8.0,
		},
		"webgl_renderer":               "ANGLE (NVIDIA GeForce GTX 1080 DirectX 11.0)",
		"screen": map[string]interface{}{
			"width":  1920,
			"height": 1080,
		},
		"canvas_supported":            true,
		"webgl_supported":             true,
		"session_storage_available":   true,
		"local_storage_available":      true,
		"indexed_db_available":        true,
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.IsHeadless && result.RiskScore > 60 {
		t.Errorf("Expected normal browser not to be flagged as headless, risk: %.2f", result.RiskScore)
	}
}

func TestEmptyNavigatorData(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Indicators) == 0 {
		t.Error("Expected indicators for missing navigator data")
	}
}

func TestNilClientData(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	result := detector.Detect(req, nil)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	found := false
	for _, indicator := range result.Indicators {
		if indicator.Type == "missing_data" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected missing_data indicator when clientData is nil")
	}
}

func TestWebdriverTrue(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver": true,
		},
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.RiskScore < 80 {
		t.Errorf("Expected high risk score for webdriver=true, got %.2f", result.RiskScore)
	}

	found := false
	for _, check := range result.NavigatorChecks {
		if check.Property == "webdriver" && check.IsSuspicious {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected webdriver check to be marked suspicious")
	}
}

func TestEmptyPlugins(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver": false,
			"plugins":   []interface{}{},
		},
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	found := false
	for _, check := range result.PluginChecks {
		if check.Suspicious && check.RiskScore >= 30 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected empty plugins to be flagged")
	}
}

func TestSoftwareRenderer(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver": false,
			"plugins":   []interface{}{"Plugin A"},
		},
		"webgl_renderer": "SwiftShader SwiftShader 4.0.0 (Build 20.0.0)",
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	found := false
	for _, indicator := range result.Indicators {
		if indicator.Type == "environment" && indicator.Severity >= 50 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected software renderer to be detected")
	}
}

func TestZeroScreenSize(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{
		"screen": map[string]interface{}{
			"width":  0,
			"height": 0,
		},
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	found := false
	for _, check := range result.EnvironmentChecks {
		if check.CheckType == "screen_size" && !check.Passed {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected zero screen size to be flagged")
	}
}

func TestCommonHeadlessResolution(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{
		"screen": map[string]interface{}{
			"width":  800,
			"height": 600,
		},
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	found := false
	for _, indicator := range result.Indicators {
		if indicator.Type == "environment" && indicator.Severity >= 30 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected common headless resolution to be flagged")
	}
}

func TestMultipleAutomationIndicators(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver":           true,
			"languages":          []interface{}{},
			"plugins":            []interface{}{},
			"platform":           "",
			"deviceMemory":       0,
			"hardwareConcurrency": 0,
		},
		"__selenium_evaluate":       true,
		"__webdriver_script_fn":     true,
		"$cdc_asdjflasutopfhvcZLmcfl_": true,
		"__playwright__":             true,
		"webgl_renderer":             "SwiftShader",
		"screen": map[string]interface{}{
			"width":  800,
			"height": 600,
		},
	}

	result := detector.Detect(req, clientData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.RiskScore < 70 {
		t.Errorf("Expected high risk score with multiple indicators, got %.2f", result.RiskScore)
	}

	if result.Confidence < 0.7 {
		t.Errorf("Expected high confidence with multiple indicators, got %.2f", result.Confidence)
	}
}

func TestGetStats(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver": true,
		},
	}

	for i := 0; i < 5; i++ {
		detector.Detect(req, clientData)
	}

	stats := detector.GetStats()

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.TotalChecks != 5 {
		t.Errorf("Expected 5 total checks, got %d", stats.TotalChecks)
	}

	if stats.DetectionRate == 0 {
		t.Error("Expected non-zero detection rate")
	}
}

func TestResetStats(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver": true,
		},
	}

	for i := 0; i < 5; i++ {
		detector.Detect(req, clientData)
	}

	detector.ResetStats()

	stats := detector.GetStats()

	if stats.TotalChecks != 0 {
		t.Errorf("Expected 0 total checks after reset, got %d", stats.TotalChecks)
	}
}

func TestCacheResult(t *testing.T) {
	detector := NewHeadlessDetector()

	sessionID := "test-session-123"
	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver": true,
		},
	}

	result := detector.Detect(nil, clientData)

	detector.CacheResult(sessionID, result)

	cached := detector.GetCachedResult(sessionID)

	if cached == nil {
		t.Fatal("Expected cached result")
	}

	if cached.RiskScore != result.RiskScore {
		t.Errorf("Expected cached risk score to match, got %.2f vs %.2f", cached.RiskScore, result.RiskScore)
	}
}

func TestUpdateConfig(t *testing.T) {
	detector := NewHeadlessDetector()

	newConfig := &model.HeadlessDetectionConfig{
		EnableNavigatorDetection:   true,
		EnablePluginDetection:      true,
		EnableAutomationDetection:  true,
		EnableEnvironmentDetection: true,
		HeadlessThreshold:          75.0,
		StrictMode:                 true,
	}

	detector.UpdateConfig(newConfig)

	if detector.config.HeadlessThreshold != 75.0 {
		t.Errorf("Expected threshold 75.0, got %.2f", detector.config.HeadlessThreshold)
	}

	if !detector.config.StrictMode {
		t.Error("Expected strict mode to be enabled")
	}
}

func TestGetTopSeverityIndicators(t *testing.T) {
	result := &model.HeadlessDetectionResult{
		Indicators: []model.HeadlessIndicator{
			{Type: "test1", Severity: 30.0},
			{Type: "test2", Severity: 50.0},
			{Type: "test3", Severity: 20.0},
			{Type: "test4", Severity: 70.0},
			{Type: "test5", Severity: 40.0},
		},
	}

	topIndicators := result.GetTopSeverityIndicators(3)

	if len(topIndicators) != 3 {
		t.Errorf("Expected 3 top indicators, got %d", len(topIndicators))
	}

	if topIndicators[0].Severity != 70.0 {
		t.Errorf("Expected highest severity first, got %.2f", topIndicators[0].Severity)
	}

	if topIndicators[2].Severity != 30.0 {
		t.Errorf("Expected third highest severity, got %.2f", topIndicators[2].Severity)
	}
}

func TestCalculateConfidence(t *testing.T) {
	result := &model.HeadlessDetectionResult{
		Indicators: []model.HeadlessIndicator{
			{Type: "test1", Severity: 50.0},
			{Type: "test2", Severity: 60.0},
			{Type: "test3", Severity: 70.0},
		},
		DetectionMethods: []string{"method1", "method2", "method3", "method4"},
	}

	confidence := result.CalculateConfidence()

	if confidence < 0.5 {
		t.Errorf("Expected confidence >= 0.5, got %.2f", confidence)
	}
}

func TestDetermineHeadlessStatus(t *testing.T) {
	config := model.NewHeadlessDetectionConfig()
	config.HeadlessThreshold = 50.0
	config.ConfidenceThreshold = 0.6

	tests := []struct {
		name       string
		riskScore  float64
		confidence float64
		strictMode bool
		expected   bool
	}{
		{"high risk strict", 70.0, 0.7, true, true},
		{"low risk", 30.0, 0.5, false, false},
		{"medium risk lenient", 45.0, 0.8, false, false},
		{"threshold risk", 50.0, 0.6, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config.StrictMode = tc.strictMode
			result := &model.HeadlessDetectionResult{
				RiskScore:  tc.riskScore,
				Confidence: tc.confidence,
			}

			determined := result.DetermineHeadlessStatus(config)

			if determined != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, determined)
			}
		})
	}
}

func TestBrowserFingerprintGeneration(t *testing.T) {
	data := map[string]interface{}{
		"user_agent":     "Mozilla/5.0 Chrome/120.0.0.0",
		"platform":       "Win32",
		"languages":      []interface{}{"en-US", "en"},
		"webgl_renderer": "NVIDIA GTX 1080",
	}

	fingerprint := GenerateFingerprint(data)

	if fingerprint == "" {
		t.Error("Expected non-empty fingerprint")
	}

	if len(fingerprint) != 64 {
		t.Errorf("Expected 64 character SHA256 hash, got %d characters", len(fingerprint))
	}

	fingerprint2 := GenerateFingerprint(data)

	if fingerprint != fingerprint2 {
		t.Error("Expected deterministic fingerprint generation")
	}

	data["platform"] = "Linux x86_64"
	fingerprint3 := GenerateFingerprint(data)

	if fingerprint == fingerprint3 {
		t.Error("Expected different fingerprint for different data")
	}
}

func TestHeadlessDetectionStats(t *testing.T) {
	stats := &model.HeadlessDetectionStats{
		StartTime: time.Now(),
	}

	result := &model.HeadlessDetectionResult{
		IsHeadless:        true,
		RiskScore:         75.0,
		Confidence:        0.8,
		DetectionMethods:   []string{"method1"},
	}

	stats.UpdateStats(result)

	if stats.TotalChecks != 1 {
		t.Errorf("Expected 1 total check, got %d", stats.TotalChecks)
	}

	if stats.HeadlessDetected != 1 {
		t.Errorf("Expected 1 headless detected, got %d", stats.HeadlessDetected)
	}

	if stats.AvgRiskScore != 75.0 {
		t.Errorf("Expected avg risk score 75.0, got %.2f", stats.AvgRiskScore)
	}

	if stats.DetectionRate != 100.0 {
		t.Errorf("Expected 100%% detection rate, got %.2f%%", stats.DetectionRate)
	}
}

func TestNewHeadlessDetectionConfig(t *testing.T) {
	config := model.NewHeadlessDetectionConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if !config.EnableNavigatorDetection {
		t.Error("Expected navigator detection to be enabled")
	}

	if !config.EnablePluginDetection {
		t.Error("Expected plugin detection to be enabled")
	}

	if !config.EnableAutomationDetection {
		t.Error("Expected automation detection to be enabled")
	}

	if !config.EnableEnvironmentDetection {
		t.Error("Expected environment detection to be enabled")
	}

	if config.HeadlessThreshold != 50.0 {
		t.Errorf("Expected default threshold 50.0, got %.2f", config.HeadlessThreshold)
	}

	if config.ConfidenceThreshold != 0.6 {
		t.Errorf("Expected default confidence threshold 0.6, got %.2f", config.ConfidenceThreshold)
	}

	if config.StrictMode {
		t.Error("Expected strict mode to be disabled by default")
	}
}

func TestHeadlessIndicator(t *testing.T) {
	indicator := model.HeadlessIndicator{
		Type:        "automation",
		Name:        "WebDriver Detected",
		Description: "navigator.webdriver is true",
		Severity:    85.0,
		Evidence:    "navigator.webdriver = true",
	}

	if indicator.Type != "automation" {
		t.Errorf("Expected type 'automation', got '%s'", indicator.Type)
	}

	if indicator.Severity != 85.0 {
		t.Errorf("Expected severity 85.0, got %.2f", indicator.Severity)
	}
}

func TestNavigatorCheck(t *testing.T) {
	check := model.NavigatorCheck{
		Property:        "webdriver",
		Expected:        "false",
		Actual:          "true",
		Present:         true,
		IsSuspicious:    true,
		RiskScore:       85.0,
		DetectionMethod: "navigator_property_check",
	}

	if check.Property != "webdriver" {
		t.Errorf("Expected property 'webdriver', got '%s'", check.Property)
	}

	if !check.IsSuspicious {
		t.Error("Expected check to be suspicious")
	}

	if check.RiskScore != 85.0 {
		t.Errorf("Expected risk score 85.0, got %.2f", check.RiskScore)
	}
}

func TestPluginCheck(t *testing.T) {
	check := model.PluginCheck{
		PluginName:      "Chrome PDF Plugin",
		Present:        true,
		IsCommon:       true,
		Suspicious:     false,
		RiskScore:      0,
		DetectionMethod: "plugin_check",
	}

	if !check.Present {
		t.Error("Expected plugin to be present")
	}

	if !check.IsCommon {
		t.Error("Expected plugin to be marked as common")
	}
}

func TestAutomationCheck(t *testing.T) {
	check := model.AutomationCheck{
		ToolName:        "selenium",
		Detected:        true,
		Confidence:      0.85,
		RiskScore:       70.0,
		Indicators:      []string{"UA pattern", "JS indicator"},
		DetectionMethod: "automation_tool_check",
	}

	if !check.Detected {
		t.Error("Expected tool to be detected")
	}

	if check.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %.2f", check.Confidence)
	}

	if len(check.Indicators) != 2 {
		t.Errorf("Expected 2 indicators, got %d", len(check.Indicators))
	}
}

func TestEnvironmentCheck(t *testing.T) {
	check := model.EnvironmentCheck{
		CheckType:   "screen_size",
		Name:        "Screen Size Validation",
		Passed:      false,
		RiskScore:   45.0,
		Description: "Validates screen dimensions",
		Evidence:    "0x0 resolution",
	}

	if check.Passed {
		t.Error("Expected check to fail")
	}

	if check.RiskScore != 45.0 {
		t.Errorf("Expected risk score 45.0, got %.2f", check.RiskScore)
	}
}

func TestSessionManagement(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Session-ID", "session-abc-123")

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver": true,
		},
	}

	result := detector.Detect(req, clientData)

	if result.SessionID != "session-abc-123" {
		t.Errorf("Expected session ID 'session-abc-123', got '%s'", result.SessionID)
	}

	if len(detector.sessionData) != 1 {
		t.Errorf("Expected 1 session, got %d", len(detector.sessionData))
	}

	result2 := detector.Detect(req, clientData)

	if result2.SessionID != "session-abc-123" {
		t.Errorf("Expected session ID 'session-abc-123' for second request, got '%s'", result2.SessionID)
	}

	session := detector.sessionData["session-abc-123"]
	if session.RequestCount != 2 {
		t.Errorf("Expected 2 requests in session, got %d", session.RequestCount)
	}
}

func TestRecommendations(t *testing.T) {
	detector := NewHeadlessDetector()

	req := httptest.NewRequest("GET", "/test", nil)

	clientData := map[string]interface{}{
		"navigator": map[string]interface{}{
			"webdriver":           true,
			"languages":          []interface{}{},
			"plugins":            []interface{}{},
			"platform":           "",
			"deviceMemory":       0,
			"hardwareConcurrency": 0,
		},
		"webgl_renderer": "SwiftShader",
	}

	result := detector.Detect(req, clientData)

	if len(result.Recommendations) == 0 {
		t.Error("Expected recommendations for headless detection")
	}
}

func TestGetScreenSize(t *testing.T) {
	tests := []struct {
		name           string
		input          map[string]interface{}
		expectedWidth  int
		expectedHeight int
	}{
		{"float values", map[string]interface{}{"width": 1920.0, "height": 1080.0}, 1920, 1080},
		{"int values", map[string]interface{}{"width": 1366, "height": 768}, 1366, 768},
		{"zero values", map[string]interface{}{"width": 0, "height": 0}, 0, 0},
		{"mixed types", map[string]interface{}{"width": 1920.0, "height": 720}, 1920, 720},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			width, height := getScreenSize(tc.input)

			if width != tc.expectedWidth {
				t.Errorf("Expected width %d, got %d", tc.expectedWidth, width)
			}

			if height != tc.expectedHeight {
				t.Errorf("Expected height %d, got %d", tc.expectedHeight, height)
			}
		})
	}
}

func TestParseBrowserInfo(t *testing.T) {
	tests := []struct {
		name            string
		userAgent       string
		expectedBrowser string
		expectedVersion string
	}{
		{"Chrome", "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36", "Chrome", "120"},
		{"Firefox", "Mozilla/5.0 Firefox/119.0", "Firefox", "119"},
		{"Edge", "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0", "Edge", "120"},
		{"Safari", "Mozilla/5.0 Safari/605.1.15", "Safari", "605"},
		{"Unknown", "Some Unknown Browser", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			browser, version := parseBrowserInfo(tc.userAgent)

			if browser != tc.expectedBrowser {
				t.Errorf("Expected browser '%s', got '%s'", tc.expectedBrowser, browser)
			}

			if version != tc.expectedVersion {
				t.Errorf("Expected version '%s', got '%s'", tc.expectedVersion, version)
			}
		})
	}
}

func TestParseOS(t *testing.T) {
	tests := []struct {
		name         string
		userAgent    string
		expectedOS   string
	}{
		{"Windows", "Mozilla/5.0 Windows NT 10.0; Win64; x64", "Windows"},
		{"macOS", "Mozilla/5.0 Macintosh; Intel Mac OS X 10_15_7", "macOS"},
		{"Linux", "Mozilla/5.0 X11; Linux x86_64", "Linux"},
		{"Android", "Mozilla/5.0 Linux; Android 12; SM-G998B", "Android"},
		{"iOS", "Mozilla/5.0 iPhone; CPU iPhone OS 16_0 like Mac OS X", "iOS"},
		{"Unknown", "Mozilla/5.0 Unknown OS", "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			os := parseOS(tc.userAgent)

			if os != tc.expectedOS {
				t.Errorf("Expected OS '%s', got '%s'", tc.expectedOS, os)
			}
		})
	}
}

func TestAddIndicator(t *testing.T) {
	result := &model.HeadlessDetectionResult{
		RiskScore:  0,
		Indicators: make([]model.HeadlessIndicator, 0),
	}

	result.AddIndicator(model.HeadlessIndicator{
		Type:      "test",
		Name:      "Test",
		Severity:  50.0,
	})

	if len(result.Indicators) != 1 {
		t.Errorf("Expected 1 indicator, got %d", len(result.Indicators))
	}

	if result.RiskScore != 50.0 {
		t.Errorf("Expected risk score 50.0, got %.2f", result.RiskScore)
	}

	result.AddIndicator(model.HeadlessIndicator{
		Type:     "test2",
		Name:     "Test 2",
		Severity: 30.0,
	})

	if len(result.Indicators) != 2 {
		t.Errorf("Expected 2 indicators, got %d", len(result.Indicators))
	}

	if result.RiskScore != 80.0 {
		t.Errorf("Expected risk score 80.0, got %.2f", result.RiskScore)
	}
}

func TestConcurrentAccess(t *testing.T) {
	detector := NewHeadlessDetector()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Session-ID", fmt.Sprintf("session-%d", id))

			clientData := map[string]interface{}{
				"navigator": map[string]interface{}{
					"webdriver": true,
				},
			}

			detector.Detect(req, clientData)
			detector.GetStats()
			detector.GetConfig()

			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if len(detector.sessionData) != 10 {
		t.Errorf("Expected 10 sessions, got %d", len(detector.sessionData))
	}
}

func TestHTTPRequestHandling(t *testing.T) {
	detector := NewHeadlessDetector()

	tests := []struct {
		name      string
		method    string
		path      string
		headers   map[string]string
		expectNil bool
	}{
		{"GET request", "GET", "/api/test", nil, false},
		{"POST request", "POST", "/api/submit", nil, false},
		{"With session", "GET", "/test", map[string]string{"X-Session-ID": "test123"}, false},
		{"Nil request", "", "", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.method != "" {
				req = httptest.NewRequest(tc.method, tc.path, nil)
				for k, v := range tc.headers {
					req.Header.Set(k, v)
				}
			}

			clientData := map[string]interface{}{
				"navigator": map[string]interface{}{
					"webdriver": true,
				},
			}

			result := detector.Detect(req, clientData)

			if tc.expectNil && result != nil {
				t.Error("Expected nil result")
			}

			if !tc.expectNil && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}
