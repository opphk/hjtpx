package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	github.com/hjtpx/hjtpx/internal/model"
)

var (
	headlessChromePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)HeadlessChrome`),
		regexp.MustCompile(`(?i)chrome-headless`),
		regexp.MustCompile(`(?i)Headless\s+Chrome`),
		regexp.MustCompile(`(?i)\(X11\).*AppleWebKit.*Chrome/.*Safari/.*Headless`),
	}

	headlessFirefoxPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)Firefox/.*Headless`),
		regexp.MustCompile(`(?i)firefox-headless`),
		regexp.MustCompile(`(?i)HeadlessFirefox`),
	}

	automationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)webdriver`),
		regexp.MustCompile(`(?i)selenium`),
		regexp.MustCompile(`(?i)phantom`),
		regexp.MustCompile(`(?i)puppeteer`),
		regexp.MustCompile(`(?i)playwright`),
		regexp.MustCompile(`(?i)automation`),
		regexp.MustCompile(`(?i)chrome-automation`),
		regexp.MustCompile(`(?i)__selenium_evaluate`),
		regexp.MustCompile(`(?i)__webdriver_script_fn`),
		regexp.MustCompile(`(?i)\$cdc_`),
		regexp.MustCompile(`(?i)\$chrome_asyncScriptInfo`),
		regexp.MustCompile(`(?i)__puppeteer_evaluation_script`),
		regexp.MustCompile(`(?i)__playwright__`),
		regexp.MustCompile(`(?i)__pw_api_hooks__`),
	}

	softwareRendererPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)swiftshader`),
		regexp.MustCompile(`(?i)llvmpipe`),
		regexp.MustCompile(`(?i)software`),
		regexp.MustCompile(`(?i)virtualbox`),
		regexp.MustCompile(`(?i)vmware`),
		regexp.MustCompile(`(?i)parallels`),
	}

	commonBotPlugins = []string{
		"chrome pdf plugin",
		"internal pdf viewer",
		"native client",
		"googletalk",
		"facebook plugin",
		"sharethis",
	}

	headlessNavigatorIndicators = map[string]struct {
		SuspiciousValue string
		RiskScore       float64
		Description     string
	}{
		"webdriver": {
			SuspiciousValue: "true",
			RiskScore:       85.0,
			Description:     "navigator.webdriver is true, indicating automation tool",
		},
		"languages": {
			SuspiciousValue: "",
			RiskScore:       30.0,
			Description:     "navigator.languages is empty or missing",
		},
		"plugins": {
			SuspiciousValue: "",
			RiskScore:       25.0,
			Description:     "navigator.plugins is empty",
		},
		"permissions": {
			SuspiciousValue: "",
			RiskScore:       20.0,
			Description:     "navigator.permissions query returns denied",
		},
		"deviceMemory": {
			SuspiciousValue: "",
			RiskScore:       15.0,
			Description:     "navigator.deviceMemory is missing or abnormal",
		},
		"hardwareConcurrency": {
			SuspiciousValue: "",
			RiskScore:       10.0,
			Description:     "navigator.hardwareConcurrency is missing or abnormal",
		},
		"platform": {
			SuspiciousValue: "",
			RiskScore:       15.0,
			Description:     "navigator.platform is empty or suspicious",
		},
		"vendor": {
			SuspiciousValue: "",
			RiskScore:       10.0,
			Description:     "navigator.vendor is empty",
		},
	}

	automationToolSignatures = map[string]struct {
		Name       string
		Patterns   []*regexp.Regexp
		Indicators []string
		Weight     float64
	}{
		"selenium": {
			Name: "Selenium",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)selenium`),
				regexp.MustCompile(`(?i)webdriver.*selenium`),
			},
			Indicators: []string{
				"__selenium_evaluate",
				"__webdriver_script_fn",
				"Selenium.prototype",
				"selenium_webdriver",
				"__driver_evaluate",
				"__fxdriver_evaluate",
			},
			Weight: 80.0,
		},
		"puppeteer": {
			Name: "Puppeteer",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)puppeteer`),
				regexp.MustCompile(`(?i)headless.*chrome`),
				regexp.MustCompile(`(?i)chrome.*headless`),
			},
			Indicators: []string{
				"$cdc_asdjflasutopfhvcZLmcfl_",
				"$chrome_asyncScriptInfo",
				"__puppeteer_evaluation_script",
				"__puppeteer_global__",
				"__puppeteer_script_url",
			},
			Weight: 85.0,
		},
		"playwright": {
			Name: "Playwright",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)playwright`),
			},
			Indicators: []string{
				"__playwright__",
				"__pw_api_hooks__",
				"__pw_resume__",
				"__pw_timeout__",
				"__pw_script_data__",
			},
			Weight: 82.0,
		},
		"phantomjs": {
			Name: "PhantomJS",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)phantomjs`),
				regexp.MustCompile(`(?i)phantom\.js`),
			},
			Indicators: []string{
				"phantom",
				"callPhantom",
				"_phantom",
				"page.onCallback",
			},
			Weight: 88.0,
		},
		"webdriver": {
			Name: "WebDriver",
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)webdriver`),
				regexp.MustCompile(`(?i)chrome-automation`),
			},
			Indicators: []string{
				"webdriver",
				"__webdriver_evaluate",
				"__driver_evaluate",
				"_WEBDRIVER_EVALUATOR_",
			},
			Weight: 75.0,
		},
	}

	environmentAnomalies = map[string]struct {
		Name        string
		CheckFunc   func(interface{}) bool
		RiskScore   float64
		Description string
	}{
		"zeroScreenSize": {
			Name: "Zero Screen Size",
			CheckFunc: func(v interface{}) bool {
				if m, ok := v.(map[string]interface{}); ok {
					w, h := getScreenSize(m)
					return w == 0 || h == 0
				}
				return false
			},
			RiskScore:   45.0,
			Description: "Screen size is zero or invalid",
		},
		"missingCanvas": {
			Name: "Missing Canvas Support",
			CheckFunc: func(v interface{}) bool {
				if b, ok := v.(bool); ok {
					return !b
				}
				return true
			},
			RiskScore:   35.0,
			Description: "Canvas element not supported",
		},
		"missingWebGL": {
			Name: "Missing WebGL Support",
			CheckFunc: func(v interface{}) bool {
				if b, ok := v.(bool); ok {
					return !b
				}
				return true
			},
			RiskScore:   30.0,
			Description: "WebGL not supported",
		},
		"noStorage": {
			Name: "No Storage Available",
			CheckFunc: func(v interface{}) bool {
				if b, ok := v.(bool); ok {
					return !b
				}
				return false
			},
			RiskScore:   25.0,
			Description: "localStorage/sessionStorage not available",
		},
	}
)

type HeadlessDetector struct {
	config         *model.HeadlessDetectionConfig
	stats          *model.HeadlessDetectionStats
	detectionCache map[string]*model.HeadlessDetectionResult
	mu             sync.RWMutex
	sessionData    map[string]*SessionInfo
}

type SessionInfo struct {
	SessionID      string
	RequestCount   int
	LastCheckTime  time.Time
	DetectionCount int
}

func NewHeadlessDetector() *HeadlessDetector {
	return &HeadlessDetector{
		config:         model.NewHeadlessDetectionConfig(),
		stats:          &model.HeadlessDetectionStats{StartTime: time.Now()},
		detectionCache: make(map[string]*model.HeadlessDetectionResult),
		sessionData:    make(map[string]*SessionInfo),
	}
}

func NewHeadlessDetectorWithConfig(config *model.HeadlessDetectionConfig) *HeadlessDetector {
	detector := NewHeadlessDetector()
	if config != nil {
		detector.config = config
	}
	return detector
}

func (d *HeadlessDetector) Detect(r *http.Request, clientData map[string]interface{}) *model.HeadlessDetectionResult {
	result := &model.HeadlessDetectionResult{
		Timestamp:         time.Now(),
		DetectionMethods:  make([]string, 0),
		Indicators:        make([]model.HeadlessIndicator, 0),
		NavigatorChecks:   make([]model.NavigatorCheck, 0),
		PluginChecks:      make([]model.PluginCheck, 0),
		AutomationChecks: make([]model.AutomationCheck, 0),
		EnvironmentChecks: make([]model.EnvironmentCheck, 0),
		Recommendations:  make([]string, 0),
	}

	sessionID := d.getOrCreateSession(r)
	result.SessionID = sessionID

	if d.config.EnableNavigatorDetection {
		d.detectNavigatorProperties(clientData, result)
	}

	if d.config.EnablePluginDetection {
		d.detectPlugins(clientData, result)
	}

	if d.config.EnableAutomationDetection {
		d.detectAutomationTools(r, clientData, result)
	}

	if d.config.EnableEnvironmentDetection {
		d.detectEnvironmentAnomalies(clientData, result)
	}

	d.detectBrowserEnvironment(clientData, result)

	result.Confidence = result.CalculateConfidence()
	result.IsHeadless = result.DetermineHeadlessStatus(d.config)

	d.updateStats(result)

	if result.IsHeadless {
		result.DetectedTool = d.identifyPrimaryTool(result)
		d.generateRecommendations(result)
	}

	return result
}

func (d *HeadlessDetector) detectNavigatorProperties(data map[string]interface{}, result *model.HeadlessDetectionResult) {
	result.DetectionMethods = append(result.DetectionMethods, "navigator_property_check")

	if data == nil {
		result.AddIndicator(model.HeadlessIndicator{
			Type:        "missing_data",
			Name:        "No Client Data",
			Description: "No client-side detection data provided",
			Severity:    10.0,
			Evidence:    "clientData is nil",
		})
		return
	}

	navigator, ok := data["navigator"].(map[string]interface{})
	if !ok {
		result.AddIndicator(model.HeadlessIndicator{
			Type:        "missing_data",
			Name:        "No Navigator Data",
			Description: "Navigator data not available",
			Severity:    15.0,
			Evidence:    "navigator object missing",
		})
		return
	}

	for prop, info := range headlessNavigatorIndicators {
		check := model.NavigatorCheck{
			Property:        prop,
			Expected:        "varies by property",
			Actual:          "",
			Present:         false,
			IsSuspicious:    false,
			RiskScore:       0,
			DetectionMethod: "navigator_property_check",
		}

		value, exists := navigator[prop]

		switch prop {
		case "webdriver":
			check.Expected = "false or undefined"
			if exists {
				check.Present = true
				check.Actual = fmt.Sprintf("%v", value)
				if v, ok := value.(bool); ok && v {
					check.IsSuspicious = true
					check.RiskScore = info.RiskScore
					result.AddIndicator(model.HeadlessIndicator{
						Type:        "automation",
						Name:        "WebDriver Detected",
						Description: info.Description,
						Severity:    info.RiskScore,
						Evidence:    "navigator.webdriver = true",
					})
				}
			}

		case "languages":
			check.Expected = "valid language array"
			if exists {
				check.Present = true
				if langs, ok := value.([]interface{}); ok && len(langs) > 0 {
					check.Actual = fmt.Sprintf("%v", langs)
				} else {
					check.IsSuspicious = true
					check.RiskScore = info.RiskScore
					result.AddIndicator(model.HeadlessIndicator{
						Type:        "configuration",
						Name:        "Empty Languages",
						Description: info.Description,
						Severity:    info.RiskScore,
						Evidence:    "navigator.languages is empty",
					})
				}
			} else {
				check.IsSuspicious = true
				check.RiskScore = info.RiskScore
			}

		case "plugins":
			check.Expected = "array with plugins"
			if exists {
				check.Present = true
				if plugins, ok := value.([]interface{}); ok {
					check.Actual = fmt.Sprintf("count: %d", len(plugins))
					if len(plugins) == 0 {
						check.IsSuspicious = true
						check.RiskScore = info.RiskScore
						result.AddIndicator(model.HeadlessIndicator{
							Type:        "configuration",
							Name:        "No Plugins",
							Description: info.Description,
							Severity:    info.RiskScore,
							Evidence:    "navigator.plugins is empty",
						})
					}
				}
			}

		case "permissions":
			check.Expected = "granted or prompt"
			if exists {
				check.Present = true
				if perms, ok := value.(map[string]interface{}); ok {
					if status, ok := perms["notifications"].(string); ok && status == "denied" {
						check.IsSuspicious = true
						check.RiskScore = info.RiskScore
					}
				}
			}

		case "deviceMemory":
			check.Expected = "> 0"
			if exists {
				check.Present = true
				if mem, ok := value.(float64); ok && mem > 0 {
					check.Actual = fmt.Sprintf("%.1f GB", mem)
				} else {
					check.IsSuspicious = true
					check.RiskScore = info.RiskScore
					result.AddIndicator(model.HeadlessIndicator{
						Type:        "configuration",
						Name:        "Missing Device Memory",
						Description: info.Description,
						Severity:    info.RiskScore,
						Evidence:    "navigator.deviceMemory missing or invalid",
					})
				}
			}

		case "hardwareConcurrency":
			check.Expected = "> 0"
			if exists {
				check.Present = true
				if cores, ok := value.(float64); ok && cores > 0 {
					check.Actual = fmt.Sprintf("%.0f cores", cores)
				} else {
					check.IsSuspicious = true
					check.RiskScore = info.RiskScore
				}
			}

		case "platform":
			check.Expected = "valid platform string"
			if exists {
				check.Present = true
				if plat, ok := value.(string); ok && plat != "" {
					check.Actual = plat
				} else {
					check.IsSuspicious = true
					check.RiskScore = info.RiskScore
					result.AddIndicator(model.HeadlessIndicator{
						Type:        "configuration",
						Name:        "Missing Platform",
						Description: info.Description,
						Severity:    info.RiskScore,
						Evidence:    "navigator.platform is empty",
					})
				}
			}

		case "vendor":
			check.Expected = "valid vendor string"
			if exists {
				check.Present = true
				if vendor, ok := value.(string); ok && vendor != "" {
					check.Actual = vendor
				} else {
					check.IsSuspicious = true
					check.RiskScore = info.RiskScore
				}
			}
		}

		result.NavigatorChecks = append(result.NavigatorChecks, check)
	}
}

func (d *HeadlessDetector) detectPlugins(data map[string]interface{}, result *model.HeadlessDetectionResult) {
	result.DetectionMethods = append(result.DetectionMethods, "plugin_check")

	if data == nil {
		return
	}

	plugins, ok := data["plugins"].([]interface{})
	if !ok {
		result.AddIndicator(model.HeadlessIndicator{
			Type:        "configuration",
			Name:        "Cannot Check Plugins",
			Description: "Plugin list not available",
			Severity:    10.0,
			Evidence:    "plugins data missing",
		})
		return
	}

	if len(plugins) == 0 {
		result.AddIndicator(model.HeadlessIndicator{
			Type:        "configuration",
			Name:        "No Plugins Installed",
			Description: "No browser plugins detected, unusual for normal browser",
			Severity:    30.0,
			Evidence:    "plugins array is empty",
		})
		result.PluginChecks = append(result.PluginChecks, model.PluginCheck{
			PluginName:      "none",
			Present:         false,
			IsCommon:        false,
			Suspicious:     true,
			RiskScore:      30.0,
			DetectionMethod: "plugin_check",
		})
		return
	}

	pluginNames := make([]string, 0)
	suspiciousCount := 0

	for _, p := range plugins {
		if p == nil {
			continue
		}

		var name string
		switch v := p.(type) {
		case string:
			name = v
		case map[string]interface{}:
			if n, ok := v["name"].(string); ok {
				name = n
			}
		default:
			name = fmt.Sprintf("%v", p)
		}

		if name == "" {
			continue
		}

		pluginNames = append(pluginNames, name)

		isCommon := false
		suspicious := false

		lowerName := strings.ToLower(name)
		for _, common := range commonBotPlugins {
			if strings.Contains(lowerName, common) {
				suspicious = true
				suspiciousCount++
				break
			}
		}

		check := model.PluginCheck{
			PluginName:      name,
			Present:         true,
			IsCommon:        isCommon,
			Suspicious:     suspicious,
			RiskScore:      0,
			DetectionMethod: "plugin_check",
		}

		if suspicious {
			check.RiskScore = 25.0
		}

		result.PluginChecks = append(result.PluginChecks, check)
	}

	if suspiciousCount > 0 {
		result.AddIndicator(model.HeadlessIndicator{
			Type:        "configuration",
			Name:        "Suspicious Plugin Pattern",
			Description: fmt.Sprintf("Found %d suspicious plugins in plugin list", suspiciousCount),
			Severity:    25.0 * float64(suspiciousCount),
			Evidence:    fmt.Sprintf("suspicious plugins: %d", suspiciousCount),
		})
	}

	if len(pluginNames) < 3 && len(plugins) > 0 {
		result.AddIndicator(model.HeadlessIndicator{
			Type:        "configuration",
			Name:        "Few Plugins",
			Description: "Very few browser plugins detected, may indicate headless environment",
			Severity:    20.0,
			Evidence:    fmt.Sprintf("only %d plugins detected", len(pluginNames)),
		})
	}
}

func (d *HeadlessDetector) detectAutomationTools(r *http.Request, data map[string]interface{}, result *model.HeadlessDetectionResult) {
	result.DetectionMethods = append(result.DetectionMethods, "automation_tool_check")

	userAgent := ""
	if r != nil {
		userAgent = r.UserAgent()
	}

	uaIndicators := make([]string, 0)
	uaRiskScore := 0.0

	for toolName, signature := range automationToolSignatures {
		check := model.AutomationCheck{
			ToolName:       toolName,
			Detected:       false,
			Confidence:     0,
			RiskScore:      0,
			Indicators:     make([]string, 0),
			DetectionMethod: "automation_tool_check",
		}

		detected := false
		confidence := 0.0

		for _, pattern := range signature.Patterns {
			if pattern.MatchString(userAgent) {
				detected = true
				confidence += 0.4
				check.Indicators = append(check.Indicators, fmt.Sprintf("UA pattern: %s", pattern.String()))
			}
		}

		if data != nil {
			for _, indicator := range signature.Indicators {
				if d.checkIndicatorInData(data, indicator) {
					detected = true
					confidence += 0.5
					check.Indicators = append(check.Indicators, fmt.Sprintf("JS indicator: %s", indicator))
				}
			}
		}

		if detected {
			check.Detected = true
			check.Confidence = math.Min(confidence, 1.0)
			check.RiskScore = signature.Weight * check.Confidence

			result.AddIndicator(model.HeadlessIndicator{
				Type:        "automation",
				Name:        fmt.Sprintf("%s Detected", signature.Name),
				Description: fmt.Sprintf("%s automation tool detected", signature.Name),
				Severity:    check.RiskScore,
				Evidence:    strings.Join(check.Indicators, "; "),
			})
		}

		result.AutomationChecks = append(result.AutomationChecks, check)
	}

	for _, pattern := range headlessChromePatterns {
		if pattern.MatchString(userAgent) {
			uaIndicators = append(uaIndicators, "headless_chrome_ua")
			uaRiskScore += 45.0
		}
	}

	for _, pattern := range headlessFirefoxPatterns {
		if pattern.MatchString(userAgent) {
			uaIndicators = append(uaIndicators, "headless_firefox_ua")
			uaRiskScore += 45.0
		}
	}

	if len(uaIndicators) > 0 {
		result.AddIndicator(model.HeadlessIndicator{
			Type:        "user_agent",
			Name:        "Headless Browser in User Agent",
			Description: "User agent string contains headless browser signature",
			Severity:    uaRiskScore,
			Evidence:    strings.Join(uaIndicators, ", "),
		})
	}
}

func (d *HeadlessDetector) detectEnvironmentAnomalies(data map[string]interface{}, result *model.HeadlessDetectionResult) {
	result.DetectionMethods = append(result.DetectionMethods, "environment_anomaly_check")

	if data == nil {
		return
	}

	if screenData, ok := data["screen"]; ok {
		check := model.EnvironmentCheck{
			CheckType:   "screen_size",
			Name:        "Screen Size Validation",
			Passed:      true,
			RiskScore:   0,
			Description: "Validates screen dimensions",
		}

		if m, ok := screenData.(map[string]interface{}); ok {
			width, height := getScreenSize(m)

			if width == 0 || height == 0 {
				check.Passed = false
				check.RiskScore = 45.0
				check.Evidence = fmt.Sprintf("Invalid screen size: %dx%d", width, height)
				result.AddIndicator(model.HeadlessIndicator{
					Type:        "environment",
					Name:        "Zero Screen Size",
					Description: "Screen dimensions are zero or invalid",
					Severity:    45.0,
					Evidence:    check.Evidence,
				})
			} else if width == 800 && height == 600 {
				check.Passed = false
				check.RiskScore = 30.0
				check.Evidence = fmt.Sprintf("Common headless resolution: %dx%d", width, height)
				result.AddIndicator(model.HeadlessIndicator{
					Type:        "environment",
					Name:        "Common Headless Resolution",
					Description: "Detected common headless browser resolution",
					Severity:    30.0,
					Evidence:    check.Evidence,
				})
			}
		}

		result.EnvironmentChecks = append(result.EnvironmentChecks, check)
	}

	if webglRenderer, ok := data["webgl_renderer"].(string); ok {
		check := model.EnvironmentCheck{
			CheckType:   "webgl_renderer",
			Name:        "WebGL Renderer Check",
			Passed:      true,
			RiskScore:   0,
			Description: "Validates WebGL renderer for software/virtual indicators",
		}

		lowerRenderer := strings.ToLower(webglRenderer)
		for _, pattern := range softwareRendererPatterns {
			if pattern.MatchString(lowerRenderer) {
				check.Passed = false
				check.RiskScore = 50.0
				check.Evidence = fmt.Sprintf("Software renderer detected: %s", webglRenderer)
				result.AddIndicator(model.HeadlessIndicator{
					Type:        "environment",
					Name:        "Software WebGL Renderer",
					Description: "WebGL is using software rendering, common in headless browsers",
					Severity:    50.0,
					Evidence:    webglRenderer,
				})
				break
			}
		}

		result.EnvironmentChecks = append(result.EnvironmentChecks, check)
	}

	if canvasData, ok := data["canvas_hash"]; ok {
		check := model.EnvironmentCheck{
			CheckType:   "canvas_fingerprint",
			Name:        "Canvas Fingerprint Check",
			Passed:      true,
			RiskScore:   0,
			Description: "Validates canvas fingerprint",
		}

		if hash, ok := canvasData.(string); ok && d.isCommonCanvasHash(hash) {
			check.Passed = false
			check.RiskScore = 35.0
			check.Evidence = fmt.Sprintf("Common canvas hash: %s", hash)
			result.AddIndicator(model.HeadlessIndicator{
				Type:        "environment",
				Name:        "Common Canvas Hash",
				Description: "Canvas fingerprint matches common headless patterns",
				Severity:    35.0,
				Evidence:    hash,
			})
		}

		result.EnvironmentChecks = append(result.EnvironmentChecks, check)
	}

	if storage, ok := data["storage_available"].(bool); ok {
		check := model.EnvironmentCheck{
			CheckType:   "storage",
			Name:        "Storage Availability Check",
			Passed:      storage,
			RiskScore:   0,
			Description: "Validates localStorage/sessionStorage availability",
		}

		if !storage {
			check.RiskScore = 25.0
			result.AddIndicator(model.HeadlessIndicator{
				Type:        "environment",
				Name:        "No Storage Available",
				Description: "Browser storage (localStorage/sessionStorage) not available",
				Severity:    25.0,
				Evidence:    "storage_available = false",
			})
		}

		result.EnvironmentChecks = append(result.EnvironmentChecks, check)
	}

	if touchPoints, ok := data["max_touch_points"].(float64); ok {
		check := model.EnvironmentCheck{
			CheckType:   "touch_support",
			Name:        "Touch Support Validation",
			Passed:      true,
			RiskScore:   0,
			Description: "Validates touch support configuration",
		}

		if touchPoints == 0 {
			userAgent := ""
			if r := getRequestFromContext(); r != nil {
				userAgent = r.UserAgent()
			}

			if strings.Contains(strings.ToLower(userAgent), "mobile") {
				check.Passed = false
				check.RiskScore = 20.0
				check.Evidence = "Mobile UA but no touch support"
				result.AddIndicator(model.HeadlessIndicator{
					Type:        "environment",
					Name:        "Touch Inconsistency",
					Description: "Mobile user agent but touch support disabled",
					Severity:    20.0,
					Evidence:    check.Evidence,
				})
			}
		}

		result.EnvironmentChecks = append(result.EnvironmentChecks, check)
	}
}

func (d *HeadlessDetector) detectBrowserEnvironment(data map[string]interface{}, result *model.HeadlessDetectionResult) {
	result.DetectionMethods = append(result.DetectionMethods, "browser_environment_check")

	if data == nil {
		return
	}

	if userAgent, ok := data["user_agent"].(string); ok {
		browser, version := parseBrowserInfo(userAgent)
		os := parseOS(userAgent)

		if browser != "" {
			result.EnvironmentChecks = append(result.EnvironmentChecks, model.EnvironmentCheck{
				CheckType:   "browser_info",
				Name:        "Browser Information",
				Passed:      true,
				RiskScore:   0,
				Description: fmt.Sprintf("Browser: %s %s, OS: %s", browser, version, os),
			})
		}
	}

	checks := []struct {
		name          string
		checkType     string
		dataKey       string
		missingRisk   float64
	}{
		{"Canvas Support", "canvas_support", "canvas_supported", 20.0},
		{"WebGL Support", "webgl_support", "webgl_supported", 25.0},
		{"AudioContext", "audio_context", "audio_context_supported", 15.0},
		{"Session Storage", "session_storage", "session_storage_available", 10.0},
		{"Local Storage", "local_storage", "local_storage_available", 10.0},
		{"IndexedDB", "indexed_db", "indexed_db_available", 15.0},
	}

	for _, check := range checks {
		checkResult := model.EnvironmentCheck{
			CheckType:   check.checkType,
			Name:        check.name,
			Passed:      true,
			RiskScore:   0,
			Description: fmt.Sprintf("Checks %s availability", check.name),
		}

		if val, ok := data[check.dataKey]; ok {
			if b, ok := val.(bool); ok && !b {
				checkResult.Passed = false
				checkResult.RiskScore = check.missingRisk
				checkResult.Evidence = fmt.Sprintf("%s not available", check.name)
				result.AddIndicator(model.HeadlessIndicator{
					Type:        "environment",
					Name:        fmt.Sprintf("%s Missing", check.name),
					Description: fmt.Sprintf("%s is not available in this environment", check.name),
					Severity:    check.missingRisk,
					Evidence:    checkResult.Evidence,
				})
			}
		}

		result.EnvironmentChecks = append(result.EnvironmentChecks, checkResult)
	}
}

func (d *HeadlessDetector) checkIndicatorInData(data map[string]interface{}, indicator string) bool {
	dataStr := fmt.Sprintf("%v", data)
	lowerData := strings.ToLower(dataStr)
	lowerIndicator := strings.ToLower(indicator)

	return strings.Contains(lowerData, lowerIndicator)
}

func (d *HeadlessDetector) isCommonCanvasHash(hash string) bool {
	if hash == "" {
		return false
	}

	commonHashes := map[string]bool{
		"a1b2c3d4e5f6": true,
		"1234567890ab": true,
		"ffffffffffff": true,
		"000000000000": true,
	}

	if commonHashes[hash] {
		return true
	}

	count := 0
	for _, c := range hash {
		if c == '0' || c == 'f' || c == 'F' {
			count++
		}
	}

	return count > len(hash)/2
}

func (d *HeadlessDetector) identifyPrimaryTool(result *model.HeadlessDetectionResult) string {
	maxScore := 0.0
	primaryTool := ""

	for _, check := range result.AutomationChecks {
		if check.Detected && check.RiskScore > maxScore {
			maxScore = check.RiskScore
			primaryTool = check.ToolName
		}
	}

	if primaryTool == "" && result.IsHeadless {
		hasHeadlessUA := false
		for _, method := range result.DetectionMethods {
			if strings.Contains(strings.ToLower(method), "headless") {
				hasHeadlessUA = true
				break
			}
		}
		if hasHeadlessUA {
			primaryTool = "headless_browser"
		}
	}

	return primaryTool
}

func (d *HeadlessDetector) generateRecommendations(result *model.HeadlessDetectionResult) {
	if result.IsHeadless {
		result.Recommendations = append(result.Recommendations, "Consider requiring additional verification for headless browser access")
	}

	if len(result.NavigatorChecks) > 0 {
		suspiciousCount := 0
		for _, check := range result.NavigatorChecks {
			if check.IsSuspicious {
				suspiciousCount++
			}
		}
		if suspiciousCount > 3 {
			result.Recommendations = append(result.Recommendations, "Multiple navigator property anomalies detected - consider enhanced verification")
		}
	}

	if result.RiskScore > 70 {
		result.Recommendations = append(result.Recommendations, "High risk score detected - recommend blocking or challenging")
	}

	if len(result.Recommendations) == 0 {
		result.Recommendations = append(result.Recommendations, "No immediate action required, monitor for pattern changes")
	}
}

func (d *HeadlessDetector) updateStats(result *model.HeadlessDetectionResult) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stats.UpdateStats(result)
}

func (d *HeadlessDetector) getOrCreateSession(r *http.Request) string {
	if r == nil {
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	if _, exists := d.sessionData[sessionID]; !exists {
		d.sessionData[sessionID] = &SessionInfo{
			SessionID:      sessionID,
			RequestCount:   0,
			LastCheckTime:  time.Now(),
			DetectionCount: 0,
		}
	}

	session := d.sessionData[sessionID]
	session.RequestCount++
	session.LastCheckTime = time.Now()
	session.DetectionCount++

	return sessionID
}

func (d *HeadlessDetector) GetStats() *model.HeadlessDetectionStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := *d.stats
	return &stats
}

func (d *HeadlessDetector) ResetStats() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stats = &model.HeadlessDetectionStats{
		StartTime: time.Now(),
	}
}

func (d *HeadlessDetector) GetConfig() *model.HeadlessDetectionConfig {
	return d.config
}

func (d *HeadlessDetector) UpdateConfig(config *model.HeadlessDetectionConfig) {
	if config != nil {
		d.config = config
	}
}

func (d *HeadlessDetector) CacheResult(sessionID string, result *model.HeadlessDetectionResult) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.detectionCache[sessionID] = result
}

func (d *HeadlessDetector) GetCachedResult(sessionID string) *model.HeadlessDetectionResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if result, exists := d.detectionCache[sessionID]; exists {
		return result
	}
	return nil
}

func getScreenSize(data map[string]interface{}) (int, int) {
	width := 0
	height := 0

	if w, ok := data["width"].(float64); ok {
		width = int(w)
	} else if w, ok := data["width"].(int); ok {
		width = w
	}

	if h, ok := data["height"].(float64); ok {
		height = int(h)
	} else if h, ok := data["height"].(int); ok {
		height = h
	}

	return width, height
}

func parseBrowserInfo(userAgent string) (string, string) {
	browserPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"Chrome", regexp.MustCompile(`Chrome/(\d+)`)},
		{"Firefox", regexp.MustCompile(`Firefox/(\d+)`)},
		{"Safari", regexp.MustCompile(`Safari/(\d+)`)},
		{"Edge", regexp.MustCompile(`Edg/(\d+)`)},
		{"Opera", regexp.MustCompile(`OPR/(\d+)`)},
		{"IE", regexp.MustCompile(`MSIE\s?(\d+)`)},
	}

	for _, bp := range browserPatterns {
		if matches := bp.pattern.FindStringSubmatch(userAgent); len(matches) > 1 {
			return bp.name, matches[1]
		}
	}

	return "", ""
}

func parseOS(userAgent string) string {
	osPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"Windows", regexp.MustCompile(`Windows\s+NT\s?([\d.]+)`)},
		{"macOS", regexp.MustCompile(`Macintosh.*Mac\s+OS\s+X\s?([\d_]+)`)},
		{"Linux", regexp.MustCompile(`Linux`)},
		{"Android", regexp.MustCompile(`Android\s([\d.]+)`)},
		{"iOS", regexp.MustCompile(`iPhone\s+OS\s([\d_]+)`)},
	}

	for _, op := range osPatterns {
		if op.pattern.MatchString(userAgent) {
			return op.name
		}
	}

	return "Unknown"
}

func GenerateFingerprint(data map[string]interface{}) string {
	hasher := sha256.New()

	keys := []string{
		"user_agent",
		"platform",
		"languages",
		"plugins",
		"canvas_hash",
		"webgl_renderer",
	}

	for _, key := range keys {
		if val, ok := data[key]; ok {
			hasher.Write([]byte(key))
			hasher.Write([]byte(":"))
			hasher.Write([]byte(fmt.Sprintf("%v", val)))
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func getRequestFromContext() *http.Request {
	return nil
}
