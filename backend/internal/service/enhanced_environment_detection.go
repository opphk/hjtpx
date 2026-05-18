package service

import (
	"context"
	"fmt"
	"strings"
)

type EnhancedEnvironmentDetection struct {
	detector *EnhancedEnvDetector
}

type EnhancedEnvDetector struct{}

type EnhancedAutomationResult struct {
	Detected    bool
	ToolName    string
	Confidence  float64
	Evidence    []string
	Severity    string
}

type HeadlessChromeIndicators struct {
	NavigatorWebdriver     bool
	WebGLSoftwareRenderer  bool
	MissingCanvasImageData bool
	ChromeAppPresent       bool
	PermissionsDenied     bool
	AutomationProperties  bool
}

type PlaywrightIndicators struct {
	WindowPlaywright       bool
	WindowPlaywrightEval   bool
	DocumentPlaywright     bool
	NavigatorWebdriver     bool
	ChromeCSIEnabled       bool
	ChromeLoadTimesEnabled bool
}

type SeleniumIndicators struct {
	WebdriverProperty     bool
	MozdriverProperty     bool
	SeleniumProperty      bool
	ChromeOptionsPresent  bool
	FirefoxProfilePresent bool
}

func NewEnhancedEnvironmentDetection() *EnhancedEnvironmentDetection {
	return &EnhancedEnvironmentDetection{
		detector: NewEnhancedEnvDetector(),
	}
}

func NewEnhancedEnvDetector() *EnhancedEnvDetector {
	return &EnhancedEnvDetector{}
}

func (d *EnhancedEnvDetector) DetectHeadlessChrome(ctx context.Context, data map[string]interface{}) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("data is nil")
	}

	indicators := d.analyzeHeadlessChromeIndicators(data)

	score := 0
	detectedIndicators := []string{}

	if indicators.NavigatorWebdriver {
		score += 40
		detectedIndicators = append(detectedIndicators, "navigator.webdriver = true")
	}

	if indicators.WebGLSoftwareRenderer {
		score += 25
		detectedIndicators = append(detectedIndicators, "WebGL software renderer detected")
	}

	if indicators.MissingCanvasImageData {
		score += 20
		detectedIndicators = append(detectedIndicators, "Canvas toDataURL returns empty or missing data")
	}

	if indicators.ChromeAppPresent {
		score += 15
		detectedIndicators = append(detectedIndicators, "Chrome app detection enabled")
	}

	if indicators.PermissionsDenied {
		score += 10
		detectedIndicators = append(detectedIndicators, "Permissions API returns denied")
	}

	if indicators.AutomationProperties {
		score += 30
		detectedIndicators = append(detectedIndicators, "Automation property detected")
	}

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "headless") {
			score += 35
			detectedIndicators = append(detectedIndicators, "UserAgent contains 'headless'")
		}
		if strings.Contains(uaLower, "chrome-lighthouse") {
			score += 40
			detectedIndicators = append(detectedIndicators, "Lighthouse automation detected")
		}
	}

	webglRenderer, _ := data["webgl_renderer"].(string)
	webglRendererLower := strings.ToLower(webglRenderer)
	softwareRenderers := []string{"swiftshader", "llvmpipe", "software", "mesa offscreen", "headless"}
	for _, renderer := range softwareRenderers {
		if strings.Contains(webglRendererLower, renderer) {
			score += 20
			detectedIndicators = append(detectedIndicators, fmt.Sprintf("Software renderer: %s", renderer))
		}
	}

	return score >= 30, nil
}

func (d *EnhancedEnvDetector) analyzeHeadlessChromeIndicators(data map[string]interface{}) HeadlessChromeIndicators {
	indicators := HeadlessChromeIndicators{}

	if val, ok := data["navigator_webdriver"].(bool); ok && val {
		indicators.NavigatorWebdriver = true
	}

	if val, ok := data["webgl_software_renderer"].(bool); ok && val {
		indicators.WebGLSoftwareRenderer = true
	}

	if val, ok := data["missing_canvas_data"].(bool); ok && val {
		indicators.MissingCanvasImageData = true
	}

	if val, ok := data["chrome_app"].(bool); ok && val {
		indicators.ChromeAppPresent = true
	}

	if val, ok := data["permissions_denied"].(bool); ok && val {
		indicators.PermissionsDenied = true
	}

	if val, ok := data["automation_property"].(bool); ok && val {
		indicators.AutomationProperties = true
	}

	return indicators
}

func (d *EnhancedEnvDetector) DetectPlaywright(ctx context.Context, data map[string]interface{}) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("data is nil")
	}

	indicators := d.analyzePlaywrightIndicators(data)

	score := 0
	detectedIndicators := []string{}

	if indicators.WindowPlaywright {
		score += 45
		detectedIndicators = append(detectedIndicators, "window.playwright detected")
	}

	if indicators.WindowPlaywrightEval {
		score += 40
		detectedIndicators = append(detectedIndicators, "window.playwright.eval detected")
	}

	if indicators.DocumentPlaywright {
		score += 35
		detectedIndicators = append(detectedIndicators, "document.playwright detected")
	}

	if indicators.NavigatorWebdriver {
		score += 30
		detectedIndicators = append(detectedIndicators, "navigator.webdriver = true (Playwright)")
	}

	if indicators.ChromeCSIEnabled {
		score += 20
		detectedIndicators = append(detectedIndicators, "Chrome CSI timing enabled")
	}

	if indicators.ChromeLoadTimesEnabled {
		score += 20
		detectedIndicators = append(detectedIndicators, "Chrome load times enabled")
	}

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "playwright") {
			score += 40
			detectedIndicators = append(detectedIndicators, "UserAgent contains 'playwright'")
		}
	}

	if val, ok := data["playwright_channel"].(string); ok && val != "" {
		score += 25
		detectedIndicators = append(detectedIndicators, fmt.Sprintf("Playwright channel: %s", val))
	}

	if val, ok := data["browser_version"].(string); ok {
		if strings.Contains(strings.ToLower(val), "playwright") {
			score += 30
			detectedIndicators = append(detectedIndicators, "Browser version contains 'playwright'")
		}
	}

	return score >= 35, nil
}

func (d *EnhancedEnvDetector) analyzePlaywrightIndicators(data map[string]interface{}) PlaywrightIndicators {
	indicators := PlaywrightIndicators{}

	if val, ok := data["window_playwright"].(bool); ok && val {
		indicators.WindowPlaywright = true
	}

	if val, ok := data["window_playwright_eval"].(bool); ok && val {
		indicators.WindowPlaywrightEval = true
	}

	if val, ok := data["document_playwright"].(bool); ok && val {
		indicators.DocumentPlaywright = true
	}

	if val, ok := data["navigator_webdriver"].(bool); ok && val {
		indicators.NavigatorWebdriver = true
	}

	if val, ok := data["chrome_csi_enabled"].(bool); ok && val {
		indicators.ChromeCSIEnabled = true
	}

	if val, ok := data["chrome_load_times_enabled"].(bool); ok && val {
		indicators.ChromeLoadTimesEnabled = true
	}

	return indicators
}

func (d *EnhancedEnvDetector) DetectSelenium(ctx context.Context, data map[string]interface{}) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("data is nil")
	}

	indicators := d.analyzeSeleniumIndicators(data)

	score := 0
	detectedIndicators := []string{}

	if indicators.WebdriverProperty {
		score += 50
		detectedIndicators = append(detectedIndicators, "webdriver property detected")
	}

	if indicators.MozdriverProperty {
		score += 40
		detectedIndicators = append(detectedIndicators, "mozdriver property detected")
	}

	if indicators.SeleniumProperty {
		score += 45
		detectedIndicators = append(detectedIndicators, "selenium property detected")
	}

	if indicators.ChromeOptionsPresent {
		score += 25
		detectedIndicators = append(detectedIndicators, "Chrome options detected")
	}

	if indicators.FirefoxProfilePresent {
		score += 25
		detectedIndicators = append(detectedIndicators, "Firefox profile detected")
	}

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "webdriver") {
			score += 40
			detectedIndicators = append(detectedIndicators, "UserAgent contains 'webdriver'")
		}
		if strings.Contains(uaLower, "selenium") {
			score += 35
			detectedIndicators = append(detectedIndicators, "UserAgent contains 'selenium'")
		}
	}

	if val, ok := data["driver_name"].(string); ok && val != "" {
		score += 30
		detectedIndicators = append(detectedIndicators, fmt.Sprintf("Driver name: %s", val))
	}

	if val, ok := data["selenium_version"].(string); ok && val != "" {
		score += 25
		detectedIndicators = append(detectedIndicators, fmt.Sprintf("Selenium version detected: %s", val))
	}

	windowProperties := []string{"__webdriver", "__selenium", "__fxdriver", "__driver_evaluate", "__webdriver_script_function"}
	if windowProps, ok := data["window_properties"].(map[string]bool); ok {
		for _, prop := range windowProperties {
			if val, exists := windowProps[prop]; exists && val {
				score += 35
				detectedIndicators = append(detectedIndicators, fmt.Sprintf("Window property: %s", prop))
			}
		}
	}

	return score >= 35, nil
}

func (d *EnhancedEnvDetector) analyzeSeleniumIndicators(data map[string]interface{}) SeleniumIndicators {
	indicators := SeleniumIndicators{}

	if val, ok := data["webdriver_property"].(bool); ok && val {
		indicators.WebdriverProperty = true
	}

	if val, ok := data["mozdriver_property"].(bool); ok && val {
		indicators.MozdriverProperty = true
	}

	if val, ok := data["selenium_property"].(bool); ok && val {
		indicators.SeleniumProperty = true
	}

	if val, ok := data["chrome_options"].(bool); ok && val {
		indicators.ChromeOptionsPresent = true
	}

	if val, ok := data["firefox_profile"].(bool); ok && val {
		indicators.FirefoxProfilePresent = true
	}

	return indicators
}

func (d *EnhancedEnvDetector) DetectAutomationTools(ctx context.Context, data map[string]interface{}) (map[string]bool, error) {
	if data == nil {
		return nil, fmt.Errorf("data is nil")
	}

	results := make(map[string]bool)

	headlessDetected, _ := d.DetectHeadlessChrome(ctx, data)
	results["headless_chrome"] = headlessDetected

	playwrightDetected, _ := d.DetectPlaywright(ctx, data)
	results["playwright"] = playwrightDetected

	seleniumDetected, _ := d.DetectSelenium(ctx, data)
	results["selenium"] = seleniumDetected

	results["puppeteer"] = d.detectPuppeteer(ctx, data)
	results["phantomjs"] = d.detectPhantomJS(ctx, data)
	results["splash"] = d.detectSplash(ctx, data)
	results["selenoid"] = d.detectSelenoid(ctx, data)
	results["zalenium"] = d.detectZalenium(ctx, data)
	results["cyber_villains"] = d.detectCyberVillains(ctx, data)

	return results, nil
}

func (d *EnhancedEnvDetector) detectPuppeteer(ctx context.Context, data map[string]interface{}) bool {
	score := 0

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "puppeteer") {
			score += 50
		}
		if strings.Contains(uaLower, "chrome/") && strings.Contains(uaLower, "headless") {
			score += 30
		}
	}

	if val, ok := data["puppeteer_protocol"].(bool); ok && val {
		score += 40
	}

	if val, ok := data["puppeteer_endpoint"].(bool); ok && val {
		score += 35
	}

	if val, ok := data["navigator_webdriver"].(bool); ok && val {
		score += 25
	}

	if val, ok := data["webgl_vendor"].(string); ok {
		if strings.Contains(strings.ToLower(val), "google") && strings.Contains(strings.ToLower(val), "swiftshader") {
			score += 20
		}
	}

	return score >= 30
}

func (d *EnhancedEnvDetector) detectPhantomJS(ctx context.Context, data map[string]interface{}) bool {
	score := 0

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "phantomjs") {
			score += 50
		}
	}

	if val, ok := data["phantomjs_page_callbacks"].(bool); ok && val {
		score += 40
	}

	if val, ok := data["phantomjs_version"].(string); ok && val != "" {
		score += 35
	}

	if val, ok := data["page_settings"].(map[string]interface{}); ok {
		if val["loadImages"] == false && val["javascriptEnabled"] == true {
			score += 25
		}
	}

	webglVendor, _ := data["webgl_vendor"].(string)
	if strings.Contains(strings.ToLower(webglVendor), "phantomjs") {
		score += 30
	}

	return score >= 30
}

func (d *EnhancedEnvDetector) detectSplash(ctx context.Context, data map[string]interface{}) bool {
	score := 0

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "splash") {
			score += 50
		}
	}

	if val, ok := data["splash_headers"].(bool); ok && val {
		score += 40
	}

	if val, ok := data["splash_request_args"].(map[string]interface{}); ok && len(val) > 0 {
		score += 35
	}

	if val, ok := data["rendered_html_length"]; ok {
		if length, ok := val.(float64); ok && length == 0 {
			score += 20
		}
	}

	return score >= 35
}

func (d *EnhancedEnvDetector) detectSelenoid(ctx context.Context, data map[string]interface{}) bool {
	score := 0

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "selenoid") {
			score += 50
		}
	}

	if val, ok := data["selenoid_hostname"].(string); ok {
		if strings.Contains(strings.ToLower(val), "selenoid") {
			score += 40
		}
	}

	if val, ok := data["docker_container"].(bool); ok && val {
		score += 30
	}

	if val, ok := data["screen_resolution"].(string); ok {
		if val == "1920x1080" || val == "1366x768" {
			score += 15
		}
	}

	return score >= 35
}

func (d *EnhancedEnvDetector) detectZalenium(ctx context.Context, data map[string]interface{}) bool {
	score := 0

	if ua, ok := data["user_agent"].(string); ok {
		uaLower := strings.ToLower(ua)
		if strings.Contains(uaLower, "zalenium") {
			score += 50
		}
	}

	if val, ok := data["docker_host_ip"].(string); ok && val != "" {
		score += 30
	}

	if val, ok := data["video_recording"].(bool); ok && val {
		score += 25
	}

	if val, ok := data["selenium_grid"].(bool); ok && val {
		score += 30
	}

	return score >= 35
}

func (d *EnhancedEnvDetector) detectCyberVillains(ctx context.Context, data map[string]interface{}) bool {
	score := 0

	if val, ok := data["cyber_villains_agent"].(bool); ok && val {
		score += 60
	}

	if val, ok := data["stealth_mode"].(bool); ok && val {
		score += 35
	}

	if val, ok := data["randomized_canvas"].(bool); ok && !val {
		score += 25
	}

	if val, ok := data["undetected_chromedriver"].(bool); ok && val {
		score += 50
	}

	return score >= 40
}

func (d *EnhancedEnvDetector) GetAutomationDetectionResult(ctx context.Context, data map[string]interface{}) *EnhancedAutomationResult {
	result := &EnhancedAutomationResult{
		Detected:   false,
		Confidence: 0,
		Evidence:   []string{},
		Severity:   "low",
	}

	tools := map[string]bool{}

	headlessDetected, _ := d.DetectHeadlessChrome(ctx, data)
	tools["headless_chrome"] = headlessDetected

	playwrightDetected, _ := d.DetectPlaywright(ctx, data)
	tools["playwright"] = playwrightDetected

	seleniumDetected, _ := d.DetectSelenium(ctx, data)
	tools["selenium"] = seleniumDetected

	puppeteerDetected := d.detectPuppeteer(ctx, data)
	tools["puppeteer"] = puppeteerDetected

	phantomjsDetected := d.detectPhantomJS(ctx, data)
	tools["phantomjs"] = phantomjsDetected

	splashDetected := d.detectSplash(ctx, data)
	tools["splash"] = splashDetected

	selenoidDetected := d.detectSelenoid(ctx, data)
	tools["selenoid"] = selenoidDetected

	zaleniumDetected := d.detectZalenium(ctx, data)
	tools["zalenium"] = zaleniumDetected

	cyberVillainsDetected := d.detectCyberVillains(ctx, data)
	tools["cyber_villains"] = cyberVillainsDetected

	detectedCount := 0
	for tool, detected := range tools {
		if detected {
			detectedCount++
			result.Evidence = append(result.Evidence, fmt.Sprintf("Detected: %s", tool))
		}
	}

	if detectedCount > 0 {
		result.Detected = true
		result.ToolName = d.getPrimaryTool(tools)

		if detectedCount >= 3 {
			result.Confidence = 95.0
			result.Severity = "critical"
		} else if detectedCount == 2 {
			result.Confidence = 80.0
			result.Severity = "high"
		} else {
			result.Confidence = 60.0
			result.Severity = "medium"
		}
	}

	return result
}

func (d *EnhancedEnvDetector) getPrimaryTool(tools map[string]bool) string {
	priorityOrder := []string{"cyber_villains", "selenium", "playwright", "puppeteer", "headless_chrome", "phantomjs", "splash", "selenoid", "zalenium"}

	for _, tool := range priorityOrder {
		if tools[tool] {
			return tool
		}
	}

	return "unknown"
}

func (d *EnhancedEnvDetector) AnalyzeAllAutomationIndicators(ctx context.Context, data map[string]interface{}) map[string]interface{} {
	analysis := make(map[string]interface{})

	headlessDetected, headlessErr := d.DetectHeadlessChrome(ctx, data)
	analysis["headless_chrome_detected"] = headlessDetected
	if headlessErr != nil {
		analysis["headless_chrome_error"] = headlessErr.Error()
	}

	playwrightDetected, playwrightErr := d.DetectPlaywright(ctx, data)
	analysis["playwright_detected"] = playwrightDetected
	if playwrightErr != nil {
		analysis["playwright_error"] = playwrightErr.Error()
	}

	seleniumDetected, seleniumErr := d.DetectSelenium(ctx, data)
	analysis["selenium_detected"] = seleniumDetected
	if seleniumErr != nil {
		analysis["selenium_error"] = seleniumErr.Error()
	}

	allTools, toolsErr := d.DetectAutomationTools(ctx, data)
	if toolsErr == nil {
		analysis["all_tools"] = allTools
	}

	detectionResult := d.GetAutomationDetectionResult(ctx, data)
	analysis["detection_result"] = detectionResult

	overallScore := 0.0
	toolCount := 0

	if headlessDetected {
		overallScore += 30
		toolCount++
	}
	if playwrightDetected {
		overallScore += 40
		toolCount++
	}
	if seleniumDetected {
		overallScore += 35
		toolCount++
	}

	if allTools != nil {
		for _, detected := range allTools {
			if detected {
				overallScore += 25
				toolCount++
			}
		}
	}

	if toolCount > 0 {
		analysis["overall_automation_score"] = overallScore / float64(toolCount)
	} else {
		analysis["overall_automation_score"] = 0.0
	}

	analysis["total_tools_detected"] = toolCount

	return analysis
}

func (s *EnhancedEnvironmentDetection) DetectHeadlessChrome(ctx context.Context, data map[string]interface{}) (bool, error) {
	return s.detector.DetectHeadlessChrome(ctx, data)
}

func (s *EnhancedEnvironmentDetection) DetectPlaywright(ctx context.Context, data map[string]interface{}) (bool, error) {
	return s.detector.DetectPlaywright(ctx, data)
}

func (s *EnhancedEnvironmentDetection) DetectSelenium(ctx context.Context, data map[string]interface{}) (bool, error) {
	return s.detector.DetectSelenium(ctx, data)
}

func (s *EnhancedEnvironmentDetection) DetectAutomationTools(ctx context.Context, data map[string]interface{}) (map[string]bool, error) {
	return s.detector.DetectAutomationTools(ctx, data)
}

func (s *EnhancedEnvironmentDetection) GetDetectionResult(ctx context.Context, data map[string]interface{}) *EnhancedAutomationResult {
	return s.detector.GetAutomationDetectionResult(ctx, data)
}

func (s *EnhancedEnvironmentDetection) AnalyzeAll(ctx context.Context, data map[string]interface{}) map[string]interface{} {
	return s.detector.AnalyzeAllAutomationIndicators(ctx, data)
}
