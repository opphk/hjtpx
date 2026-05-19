package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

type EnvDetectorService struct {
	envDetector     *EnvDetector
	blacklistSvc    *BlacklistService
	rateLimitSvc    *RateLimitService
	mu              sync.RWMutex
	envCache        map[string]*EnvInfo
	cacheExpiration time.Duration
}

type EnvDetector struct{}

type EnvInfo struct {
	UserAgent           string   `json:"user_agent"`
	Platform            string   `json:"platform"`
	Language            string   `json:"language"`
	Languages           []string `json:"languages"`
	ScreenWidth         int      `json:"screen_width"`
	ScreenHeight        int      `json:"screen_height"`
	ColorDepth          int      `json:"color_depth"`
	PixelRatio          float64  `json:"pixel_ratio"`
	Timezone            string   `json:"timezone"`
	TimezoneOffset      int      `json:"timezone_offset"`
	CanvasFingerprint   string   `json:"canvas_fingerprint"`
	WebGLRenderer       string   `json:"webgl_renderer"`
	WebGLVendor         string   `json:"webgl_vendor"`
	Plugins             []string `json:"plugins"`
	Fonts               []string `json:"fonts"`
	TouchSupport        bool     `json:"touch_support"`
	MaxTouchPoints      int      `json:"max_touch_points"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	Fingerprint         string   `json:"fingerprint"`
}

type AutomationResult struct {
	Detected bool
	Risks    []string
}

type EnvRiskResult struct {
	RiskLevel string   `json:"risk_level"`
	Score     float64  `json:"score"`
	Risks     []string `json:"risks"`
	Action    string   `json:"action"`
}

type RiskCheckResult struct {
	Name     string `json:"name"`
	Risk     string `json:"risk"`
	Detected bool   `json:"detected"`
	Score    int    `json:"score"`
	Reason   string `json:"reason,omitempty"`
}

type EnvDetectionReport struct {
	Timestamp     int64             `json:"timestamp"`
	EnvScore      float64           `json:"env_score"`
	IsRisky       bool              `json:"is_risky"`
	RiskLevel     string            `json:"risk_level"`
	DetectedTools []string          `json:"detected_tools"`
	Checks        []RiskCheckResult `json:"checks"`
	Action        string            `json:"action"`
}

type EnvVerifyRequest struct {
	SessionID      string              `json:"session_id"`
	Type           string              `json:"type"`
	X              int                 `json:"x"`
	Y              int                 `json:"y"`
	Points         [][2]int            `json:"points"`
	ClickSequence  []int               `json:"click_sequence"`
	BehaviorData   []BehaviorDataPoint `json:"behavior_data"`
	SpeedData      json.RawMessage     `json:"speed_data,omitempty"`
	ApplicationID  uint                `json:"application_id"`
	EnvironmentEnv EnvInfo             `json:"environment_env"`
	Fingerprint    string              `json:"fingerprint"`
	IPAddress      string              `json:"ip_address"`
	UserAgent      string              `json:"user_agent"`
}

type EnvVerifyResponse struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	RiskLevel   string   `json:"risk_level"`
	RiskScore   float64  `json:"risk_score"`
	RiskFactors []string `json:"risk_factors"`
	Action      string   `json:"action"`
	CaptchaPass bool     `json:"captcha_pass"`
}

func NewEnvDetectorService() *EnvDetectorService {
	return &EnvDetectorService{
		envDetector:     NewEnvDetectorBackend(),
		blacklistSvc:    NewBlacklistService(),
		rateLimitSvc:    NewRateLimitService(),
		envCache:        make(map[string]*EnvInfo),
		cacheExpiration: 5 * time.Minute,
	}
}

func NewEnvDetectorBackend() *EnvDetector {
	return &EnvDetector{}
}

type AutomationIndicators struct {
	Name          string
	Confidence    float64
	DetectionType string
	Evidence      []string
}

var automationSignatures = map[string][]string{
	"puppeteer": {
		"$cdc_asdjflasutopfhvcZLmcfl_",
		"$chrome_asyncScriptInfo",
		"__webdriver_evaluate",
		"__puppeteer_evaluation_script",
		"Puppeteer",
		"HeadlessChrome",
	},
	"playwright": {
		"__playwright__",
		"__pw_tags",
		"__pw_resume__",
		"__pw_connect__",
		"__playwright_unstripped__",
		"playwright",
	},
	"selenium": {
		"__selenium_evaluate",
		"__webdriver_script_fn",
		"__driver_evaluate",
		"__fxdriver_evaluate",
		"__webdriver_unwrapped",
		"__lastWatirAlert",
		"__$webdriverAsyncExecutor",
		"callSelenium",
		"Selenium",
		"selenium",
		"webdriver",
	},
	"cypress": {
		"__cypress_",
		"Cypress",
		"cypress",
	},
	"nightmare": {
		"Nightmare",
		"nightmare",
	},
	"testcafe": {
		"__TESTCAFE",
		"testcafe",
	},
	"webdriverio": {
		"WebDriver",
		"webdriverio",
		"wdio",
	},
}

var automationHeaders = map[string]string{
	"X-WD-Agent":      "webdriver",
	"X-SELENIUM":      "selenium",
	"X-PUPPETEER":     "puppeteer",
	"X-PLAYWRIGHT":    "playwright",
	"X-AUTOMATION":    "automation",
	"X-BOT":           "bot",
	"X-CRAWLER":       "crawler",
}

func (d *EnvDetector) DetectAutomation(info *EnvInfo) *AutomationResult {
	result := &AutomationResult{
		Detected: false,
		Risks:    []string{},
	}

	indicators := d.detectAutomationIndicators(info)
	
	for _, indicator := range indicators {
		if indicator.Confidence > 0.5 {
			result.Risks = append(result.Risks, fmt.Sprintf("%s detected (confidence: %.0f%%)", indicator.Name, indicator.Confidence*100))
			result.Detected = true
		}
	}

	if strings.Contains(strings.ToLower(info.UserAgent), "webdriver") {
		result.Risks = append(result.Risks, "Selenium WebDriver detected in UserAgent")
		result.Detected = true
	}

	if strings.Contains(strings.ToLower(info.UserAgent), "headless") {
		result.Risks = append(result.Risks, "Headless Chrome detected in UserAgent")
		result.Detected = true
	}

	if info.UserAgent == "" || len(info.UserAgent) < 20 {
		result.Risks = append(result.Risks, "Empty or short UserAgent")
		result.Detected = true
	}

	if len(info.Languages) == 0 || (len(info.Languages) == 1 && info.Language == "") {
		result.Risks = append(result.Risks, "Abnormal language settings")
		result.Detected = true
	}

	if info.CanvasFingerprint == "" {
		result.Risks = append(result.Risks, "Canvas fingerprint missing")
		result.Detected = true
	}

	if info.WebGLRenderer == "" || info.WebGLVendor == "" {
		result.Risks = append(result.Risks, "WebGL information missing")
	}

	if strings.Contains(strings.ToLower(info.UserAgent), "phantom") {
		result.Risks = append(result.Risks, "PhantomJS detected")
		result.Detected = true
	}

	return result
}

func (d *EnvDetector) detectAutomationIndicators(info *EnvInfo) []AutomationIndicators {
	indicators := []AutomationIndicators{}
	uaLower := strings.ToLower(info.UserAgent)

	for automationType, signatures := range automationSignatures {
		confidence := 0.0
		evidence := []string{}

		for _, sig := range signatures {
			if strings.Contains(uaLower, strings.ToLower(sig)) {
				confidence += 0.3
				evidence = append(evidence, fmt.Sprintf("UA match: %s", sig))
			}
		}

		if strings.Contains(info.WebGLRenderer, "SwiftShader") || 
		   strings.Contains(info.WebGLRenderer, "llvmpipe") ||
		   strings.Contains(strings.ToLower(info.WebGLRenderer), "software") {
			confidence += 0.25
			evidence = append(evidence, "Software WebGL renderer detected")
		}

		if info.HardwareConcurrency > 0 && info.HardwareConcurrency <= 2 {
			confidence += 0.15
			evidence = append(evidence, fmt.Sprintf("Low CPU cores: %d", info.HardwareConcurrency))
		}

		if len(info.Plugins) == 0 {
			confidence += 0.15
			evidence = append(evidence, "No plugins detected")
		}

		if len(info.Fonts) < 3 {
			confidence += 0.1
			evidence = append(evidence, "Limited fonts detected")
		}

		if info.TouchSupport && info.MaxTouchPoints == 0 {
			confidence += 0.1
			evidence = append(evidence, "Touch support inconsistency")
		}

		if confidence > 0 {
			indicators = append(indicators, AutomationIndicators{
				Name:          automationType,
				Confidence:    math.Min(confidence, 1.0),
				DetectionType: "signature_match",
				Evidence:      evidence,
			})
		}
	}

	nightmarePatterns := []string{"nightmare", "electron", "nw.js"}
	for _, pattern := range nightmarePatterns {
		if strings.Contains(uaLower, pattern) {
			confidence := 0.4
			if strings.Contains(uaLower, "node") || strings.Contains(info.Platform, "node") {
				confidence += 0.2
			}
			indicators = append(indicators, AutomationIndicators{
				Name:          "nightmare_electron",
				Confidence:    confidence,
				DetectionType: "framework_pattern",
				Evidence:      []string{fmt.Sprintf("UA contains: %s", pattern)},
			})
		}
	}

	cypressPatterns := []string{"cypress", "cypress_runner"}
	for _, pattern := range cypressPatterns {
		if strings.Contains(uaLower, pattern) {
			indicators = append(indicators, AutomationIndicators{
				Name:          "cypress",
				Confidence:    0.75,
				DetectionType: "framework_pattern",
				Evidence:      []string{fmt.Sprintf("UA contains: %s", pattern)},
			})
		}
	}

	return indicators
}

func (d *EnvDetector) DetectAutomationFromHeaders(headers map[string]string) []AutomationIndicators {
	indicators := []AutomationIndicators{}

	for headerName, automationType := range automationHeaders {
		if value, exists := headers[headerName]; exists && value != "" {
			indicators = append(indicators, AutomationIndicators{
				Name:          automationType,
				Confidence:    0.9,
				DetectionType: "header",
				Evidence:      []string{fmt.Sprintf("Header %s: %s", headerName, value)},
			})
		}
	}

	for headerName, value := range headers {
		headerLower := strings.ToLower(headerName)
		valueLower := strings.ToLower(value)
		
		if strings.Contains(headerLower, "automation") || strings.Contains(valueLower, "automation") {
			indicators = append(indicators, AutomationIndicators{
				Name:          "generic_automation",
				Confidence:    0.7,
				DetectionType: "header_content",
				Evidence:      []string{fmt.Sprintf("Automation indicator in header: %s", headerName)},
			})
		}

		if strings.Contains(headerLower, "bot") || strings.Contains(valueLower, "bot") {
			indicators = append(indicators, AutomationIndicators{
				Name:          "bot",
				Confidence:    0.75,
				DetectionType: "header_content",
				Evidence:      []string{fmt.Sprintf("Bot indicator in header: %s", headerName)},
			})
		}
	}

	return indicators
}

func (d *EnvDetector) EnhancedAutomationDetection(info *EnvInfo, frontendDetections []string) (bool, float64, []string) {
	detected := false
	totalConfidence := 0.0
	allEvidence := []string{}

	indicators := d.detectAutomationIndicators(info)
	for _, indicator := range indicators {
		if indicator.Confidence > 0.4 {
			detected = true
			totalConfidence += indicator.Confidence
			allEvidence = append(allEvidence, fmt.Sprintf("%s: %.0f%%", indicator.Name, indicator.Confidence*100))
			allEvidence = append(allEvidence, indicator.Evidence...)
		}
	}

	behaviorPatterns := []string{
		"timing_uniform", "click_pattern_robotic", "mouse_movement_linear",
		"keystroke_regular", "no_human_delay", "suspicious_session",
	}
	for _, pattern := range behaviorPatterns {
		for _, detection := range frontendDetections {
			if strings.Contains(strings.ToLower(detection), pattern) {
				detected = true
				totalConfidence += 0.3
				allEvidence = append(allEvidence, fmt.Sprintf("Behavior: %s", pattern))
				break
			}
		}
	}

	resourcePatterns := []string{
		"no_images", "blocking_scripts", "no_css", "minimal_requests",
		"rapid_fire", "concurrent_sessions",
	}
	for _, pattern := range resourcePatterns {
		for _, detection := range frontendDetections {
			if strings.Contains(strings.ToLower(detection), pattern) {
				detected = true
				totalConfidence += 0.2
				allEvidence = append(allEvidence, fmt.Sprintf("Resource: %s", pattern))
				break
			}
		}
	}

	return detected, math.Min(totalConfidence, 100.0), allEvidence
}

func (d *EnvDetector) CalculateEnvScore(info *EnvInfo) float64 {
	score := 100.0

	automation := d.DetectAutomation(info)
	if automation.Detected {
		score -= float64(len(automation.Risks)) * 20
	}

	if info.CanvasFingerprint == "" {
		score -= 10
	}

	if info.WebGLRenderer == "" || info.WebGLVendor == "" {
		score -= 5
	}

	if len(info.Fonts) < 3 {
		score -= 10
	}

	if info.Platform == "" {
		score -= 5
	}

	if info.ScreenWidth == 0 || info.ScreenHeight == 0 {
		score -= 5
	}

	if info.HardwareConcurrency <= 0 {
		score -= 5
	}

	if score < 0 {
		score = 0
	}
	return score
}

func (d *EnvDetector) EvaluateRisk(info *EnvInfo) *EnvRiskResult {
	automation := d.DetectAutomation(info)
	score := d.CalculateEnvScore(info)

	riskLevel := "low"
	if score < 60 {
		riskLevel = "high"
	} else if score < 80 {
		riskLevel = "medium"
	}

	return &EnvRiskResult{
		RiskLevel: riskLevel,
		Score:     score,
		Risks:     automation.Risks,
		Action:    d.determineAction(automation, score),
	}
}

func (d *EnvDetector) determineAction(automation *AutomationResult, score float64) string {
	if automation.Detected && len(automation.Risks) >= 2 {
		return "block"
	} else if automation.Detected || score < 70 {
		return "review"
	}
	return "pass"
}

func (d *EnvDetector) RunAllChecks(info *EnvInfo) *EnvDetectionReport {
	checks := []RiskCheckResult{
		{Name: "selenium", Risk: "high", Detected: false},
		{Name: "headless", Risk: "high", Detected: false},
		{Name: "phantomjs", Risk: "high", Detected: false},
		{Name: "puppeteer", Risk: "high", Detected: false},
		{Name: "playwright", Risk: "high", Detected: false},
		{Name: "devtools", Risk: "medium", Detected: false},
		{Name: "webgl_missing", Risk: "medium", Detected: false},
		{Name: "canvas_missing", Risk: "medium", Detected: false},
		{Name: "abnormal_language", Risk: "low", Detected: false},
		{Name: "no_plugins", Risk: "low", Detected: false},
	}

	uaLower := strings.ToLower(info.UserAgent)

	if strings.Contains(uaLower, "webdriver") || strings.Contains(uaLower, "selenium") {
		checks[0].Detected = true
		checks[0].Score = 40
		checks[0].Reason = "Selenium WebDriver特征检测"
	}

	if strings.Contains(uaLower, "headless") {
		checks[1].Detected = true
		checks[1].Score = 40
		checks[1].Reason = "Headless Chrome特征检测"
	}

	if strings.Contains(uaLower, "phantom") {
		checks[2].Detected = true
		checks[2].Score = 40
		checks[2].Reason = "PhantomJS特征检测"
	}

	if strings.Contains(uaLower, "puppeteer") {
		checks[3].Detected = true
		checks[3].Score = 40
		checks[3].Reason = "Puppeteer特征检测"
	}

	if strings.Contains(uaLower, "playwright") {
		checks[4].Detected = true
		checks[4].Score = 40
		checks[4].Reason = "Playwright特征检测"
	}

	if info.WebGLRenderer == "" || info.WebGLVendor == "" {
		checks[6].Detected = true
		checks[6].Score = 20
		checks[6].Reason = "WebGL信息缺失"
	}

	if info.CanvasFingerprint == "" {
		checks[7].Detected = true
		checks[7].Score = 20
		checks[7].Reason = "Canvas指纹缺失"
	}

	if len(info.Languages) == 0 || (len(info.Languages) == 1 && info.Language == "") {
		checks[8].Detected = true
		checks[8].Score = 10
		checks[8].Reason = "语言设置异常"
	}

	if len(info.Plugins) == 0 {
		checks[9].Detected = true
		checks[9].Score = 10
		checks[9].Reason = "无可用插件"
	}

	envScore := d.CalculateEnvScore(info)
	riskLevel := "low"
	if envScore < 60 {
		riskLevel = "high"
	} else if envScore < 80 {
		riskLevel = "medium"
	}

	detectedTools := []string{}
	for _, check := range checks {
		if check.Detected {
			detectedTools = append(detectedTools, check.Name)
		}
	}

	return &EnvDetectionReport{
		EnvScore:      envScore,
		IsRisky:       envScore < 80,
		RiskLevel:     riskLevel,
		DetectedTools: detectedTools,
		Checks:        checks,
		Action:        d.determineAction(&AutomationResult{Detected: envScore < 80, Risks: detectedTools}, envScore),
	}
}

func (s *EnvDetectorService) VerifyWithEnv(sessionID string, req *EnvVerifyRequest) (*EnvVerifyResponse, error) {
	envInfo := &EnvInfo{
		UserAgent:           req.UserAgent,
		Platform:            req.EnvironmentEnv.Platform,
		Language:            req.EnvironmentEnv.Language,
		Languages:           req.EnvironmentEnv.Languages,
		ScreenWidth:         req.EnvironmentEnv.ScreenWidth,
		ScreenHeight:        req.EnvironmentEnv.ScreenHeight,
		ColorDepth:          req.EnvironmentEnv.ColorDepth,
		PixelRatio:          req.EnvironmentEnv.PixelRatio,
		Timezone:            req.EnvironmentEnv.Timezone,
		TimezoneOffset:      req.EnvironmentEnv.TimezoneOffset,
		CanvasFingerprint:   req.EnvironmentEnv.CanvasFingerprint,
		WebGLRenderer:       req.EnvironmentEnv.WebGLRenderer,
		WebGLVendor:         req.EnvironmentEnv.WebGLVendor,
		Plugins:             req.EnvironmentEnv.Plugins,
		Fonts:               req.EnvironmentEnv.Fonts,
		TouchSupport:        req.EnvironmentEnv.TouchSupport,
		MaxTouchPoints:      req.EnvironmentEnv.MaxTouchPoints,
		HardwareConcurrency: req.EnvironmentEnv.HardwareConcurrency,
		Fingerprint:         req.Fingerprint,
	}

	if envInfo.UserAgent == "" {
		envInfo.UserAgent = req.UserAgent
	}

	blacklisted, reason := s.blacklistSvc.CheckBlacklist(req.IPAddress, "ip")
	if blacklisted {
		return &EnvVerifyResponse{
			Success:     false,
			RiskLevel:   "high",
			RiskScore:   100.0,
			RiskFactors: []string{"IP黑名单: " + reason.Error()},
			Action:      "block",
			Message:     "IP已被列入黑名单",
		}, nil
	}

	if req.Fingerprint != "" {
		blacklisted, reason = s.blacklistSvc.CheckBlacklist(req.Fingerprint, "device_id")
		if blacklisted {
			return &EnvVerifyResponse{
				Success:     false,
				RiskLevel:   "high",
				RiskScore:   100.0,
				RiskFactors: []string{"设备黑名单: " + reason.Error()},
				Action:      "block",
				Message:     "设备已被列入黑名单",
			}, nil
		}
	}

	ipRateLimitConfig := &RateLimitConfig{
		MaxRequests: 100,
		WindowSecs:  60,
	}
	rateLimitResult, err := s.rateLimitSvc.CheckIPRateLimit(context.Background(), req.IPAddress, ipRateLimitConfig)
	if err == nil && !rateLimitResult.Allowed {
		return &EnvVerifyResponse{
			Success:     false,
			RiskLevel:   "medium",
			RiskScore:   70.0,
			RiskFactors: []string{"IP请求频率超限"},
			Action:      "review",
			Message:     "请求过于频繁，请稍后再试",
		}, nil
	}

	if req.Fingerprint != "" {
		fpRateLimitConfig := &RateLimitConfig{
			MaxRequests: 50,
			WindowSecs:  60,
		}
		rateLimitResult, err = s.rateLimitSvc.CheckIPRateLimit(context.Background(), req.Fingerprint, fpRateLimitConfig)
		if err == nil && !rateLimitResult.Allowed {
			return &EnvVerifyResponse{
				Success:     false,
				RiskLevel:   "medium",
				RiskScore:   65.0,
				RiskFactors: []string{"设备请求频率超限"},
				Action:      "review",
				Message:     "请求过于频繁，请稍后再试",
			}, nil
		}
	}

	envRisk := s.envDetector.EvaluateRisk(envInfo)

	if envRisk.Action == "block" {
		return &EnvVerifyResponse{
			Success:     false,
			RiskLevel:   envRisk.RiskLevel,
			RiskScore:   envRisk.Score,
			RiskFactors: envRisk.Risks,
			Action:      "block",
			Message:     "环境检测异常",
			CaptchaPass: false,
		}, nil
	}

	captchaPass := true
	if envRisk.Action == "review" || envRisk.Score < 70 {
		captchaPass = false
	}

	return &EnvVerifyResponse{
		Success:     true,
		RiskLevel:   envRisk.RiskLevel,
		RiskScore:   envRisk.Score,
		RiskFactors: envRisk.Risks,
		Action:      envRisk.Action,
		Message:     "环境检测通过",
		CaptchaPass: captchaPass,
	}, nil
}

func (s *EnvDetectorService) CheckBlacklist(ip, fingerprint string) (bool, string) {
	blocked, _ := s.blacklistSvc.CheckBlacklist(ip, "ip")
	if blocked {
		return true, "IP已被列入黑名单"
	}

	if fingerprint != "" {
		blocked, _ = s.blacklistSvc.CheckBlacklist(fingerprint, "device_id")
		if blocked {
			return true, "设备已被列入黑名单"
		}
	}

	return false, ""
}

func (s *EnvDetectorService) GetEnvDetectionReport(envInfo *EnvInfo) *EnvDetectionReport {
	return s.envDetector.RunAllChecks(envInfo)
}

func (s *EnvDetectorService) CacheEnvInfo(sessionID string, envInfo *EnvInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.envCache[sessionID] = envInfo
}

func (s *EnvDetectorService) GetCachedEnvInfo(sessionID string) *EnvInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.envCache[sessionID]; ok {
		return info
	}
	return nil
}

func (s *EnvDetectorService) CleanupExpiredCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for sessionID := range s.envCache {
		if now.Sub(time.Now()) > s.cacheExpiration {
			delete(s.envCache, sessionID)
		}
	}
}

func (d *EnvDetector) CalculateCanvasSimilarity(hash1, hash2 string) float64 {
	if hash1 == "" || hash2 == "" {
		return 0.0
	}

	if hash1 == hash2 {
		return 100.0
	}

	if len(hash1) != len(hash2) {
		return 0.0
	}

	matchCount := 0
	totalLength := len(hash1)
	for i := 0; i < totalLength; i++ {
		if hash1[i] == hash2[i] {
			matchCount++
		}
	}

	return float64(matchCount) / float64(totalLength) * 100.0
}

func (d *EnvDetector) DetectCanvasAnomalies(info *EnvInfo) []string {
	anomalies := []string{}

	if info.CanvasFingerprint == "" {
		return anomalies
	}

	if len(info.CanvasFingerprint) < 32 {
		anomalies = append(anomalies, "Canvas指纹长度异常短")
	}

	if len(info.CanvasFingerprint) > 128 {
		anomalies = append(anomalies, "Canvas指纹长度异常长")
	}

	hasHexOnly := true
	for _, c := range info.CanvasFingerprint {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			hasHexOnly = false
			break
		}
	}
	if !hasHexOnly && len(info.CanvasFingerprint) > 0 {
		anomalies = append(anomalies, "Canvas指纹包含非十六进制字符")
	}

	repeatCount := 0
	maxRepeat := 0
	var lastChar rune
	for _, c := range info.CanvasFingerprint {
		if c == lastChar {
			repeatCount++
			if repeatCount > maxRepeat {
				maxRepeat = repeatCount
			}
		} else {
			repeatCount = 0
		}
		lastChar = c
	}
	if maxRepeat > len(info.CanvasFingerprint)/2 {
		anomalies = append(anomalies, "Canvas指纹存在异常重复模式")
	}

	return anomalies
}

func (d *EnvDetector) AnalyzeWebGLDetails(info *EnvInfo) map[string]interface{} {
	analysis := make(map[string]interface{})

	if info.WebGLRenderer == "" {
		analysis["status"] = "missing"
		analysis["risk"] = "high"
		return analysis
	}

	analysis["status"] = "present"
	analysis["renderer"] = info.WebGLRenderer
	analysis["vendor"] = info.WebGLVendor

	rendererLower := strings.ToLower(info.WebGLRenderer)
	vendorLower := strings.ToLower(info.WebGLVendor)

	softwareIndicators := []string{"swiftshader", "llvmpipe", "software", "emulated", "virtual"}
	for _, indicator := range softwareIndicators {
		if strings.Contains(rendererLower, indicator) || strings.Contains(vendorLower, indicator) {
			analysis["software_detected"] = true
			analysis["risk"] = "medium"
			analysis["reason"] = fmt.Sprintf("检测到软件渲染器: %s", indicator)
			return analysis
		}
	}

	anonymizedIndicators := []string{"generic", "unknown", "default", "standard"}
	matchCount := 0
	for _, indicator := range anonymizedIndicators {
		if strings.Contains(rendererLower, indicator) || strings.Contains(vendorLower, indicator) {
			matchCount++
		}
	}
	if matchCount >= 2 {
		analysis["anonymized"] = true
		analysis["risk"] = "medium"
		analysis["reason"] = "WebGL信息可能被故意匿名化"
		return analysis
	}

	unusualPatterns := []string{"headless", "bot", "automation", "test"}
	for _, pattern := range unusualPatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			analysis["unusual_pattern"] = true
			analysis["risk"] = "high"
			analysis["reason"] = fmt.Sprintf("WebGL信息包含异常标识: %s", pattern)
			return analysis
		}
	}

	analysis["risk"] = "low"
	return analysis
}

func (d *EnvDetector) DetectEmulatorIndicators(info *EnvInfo) (bool, []string) {
	indicators := []string{}

	uaLower := strings.ToLower(info.UserAgent)

	emulatorPatterns := []string{
		"android sdk",
		"sdk_phone",
		"genymotion",
		"bluestacks",
		"nox",
		"memu",
		"LDPlayer",
		"koplayer",
		"droid4x",
		"left",
		"mumu",
		"xyson",
		"youwave",
		"andy",
		"remix os",
		"phoenix",
		"tencent",
		"smartgaga",
	}

	for _, pattern := range emulatorPatterns {
		if strings.Contains(uaLower, strings.ToLower(pattern)) {
			indicators = append(indicators, fmt.Sprintf("检测到模拟器标识: %s", pattern))
		}
	}

	if strings.Contains(uaLower, "android") && strings.Contains(uaLower, "build/") {
		buildIndex := strings.Index(uaLower, "build/")
		if buildIndex > 0 {
			buildPart := uaLower[buildIndex:]
			if strings.Contains(buildPart, "emulator") || strings.Contains(buildPart, "test") || strings.Contains(buildPart, "vbox") || strings.Contains(buildPart, "x86") {
				indicators = append(indicators, "Android Build标签包含模拟器特征")
			}
		}
	}

	if strings.Contains(uaLower, "android") {
		if info.MaxTouchPoints == 0 || info.MaxTouchPoints > 10 {
			indicators = append(indicators, "Android设备触摸点数异常")
		}

		if info.HardwareConcurrency > 16 {
			indicators = append(indicators, fmt.Sprintf("Android设备CPU核心数异常: %d", info.HardwareConcurrency))
		}

		if strings.Contains(uaLower, "x86") || strings.Contains(uaLower, "x64") {
			if !strings.Contains(uaLower, "chrome") {
				indicators = append(indicators, "非Chrome浏览器的x86/x64架构")
			}
		}
	}

	browserPatterns := []string{
		"chromium",
		"phantomjs",
		"slimerjs",
		"webkit2png",
	}

	for _, pattern := range browserPatterns {
		if strings.Contains(uaLower, pattern) && !strings.Contains(uaLower, "chrome") && !strings.Contains(uaLower, "safari") {
			indicators = append(indicators, fmt.Sprintf("异常浏览器引擎: %s", pattern))
		}
	}

	return len(indicators) > 0, indicators
}

func (d *EnvDetector) CalculateProxyRiskScore(ip string, headers map[string]string) float64 {
	score := 0.0

	xff := headers["X-Forwarded-For"]
	xri := headers["X-Real-IP"]
	via := headers["Via"]

	if xff != "" {
		score += 25.0
		parts := strings.Split(xff, ",")
		if len(parts) > 2 {
			score += 15.0
		}
	}

	if xri != "" && xri != ip {
		score += 15.0
	}

	if via != "" {
		viaLower := strings.ToLower(via)
		proxyKeywords := []string{"proxy", "squid", "nginx", "apache", "varnish", "traefik", "haproxy"}
		for _, keyword := range proxyKeywords {
			if strings.Contains(viaLower, keyword) {
				score += 20.0
				break
			}
		}
	}

	proxyChain := headers["X-ProxyChain"]
	if proxyChain != "" {
		score += 30.0
	}

	cdnHeaders := headers["X-CDN-Original-IP"]
	if cdnHeaders != "" {
		score += 20.0
	}

	return math.Min(score, 100.0)
}

func (d *EnvDetector) DetectVPNPatterns(info *EnvInfo, headers map[string]string) (bool, float64, []string) {
	isVPN := false
	confidence := 0.0
	evidence := []string{}

	vpnHeaderIndicators := []string{
		"X-VPN-Connection",
		"X-VPN-Type",
		"X-ProxyVPN",
		"X-Anonymizer",
	}

	for _, header := range vpnHeaderIndicators {
		if _, exists := headers[header]; exists {
			isVPN = true
			confidence = math.Max(confidence, 0.95)
			evidence = append(evidence, fmt.Sprintf("检测到VPN头部标识: %s", header))
		}
	}

	if info.WebGLVendor != "" {
		vendorLower := strings.ToLower(info.WebGLVendor)
		vpnKeywords := []string{"virtual", "vpn", "virtualbox", "vmware"}
		for _, keyword := range vpnKeywords {
			if strings.Contains(vendorLower, keyword) {
				isVPN = true
				confidence = math.Max(confidence, 0.70)
				evidence = append(evidence, fmt.Sprintf("WebGL厂商包含VPN标识: %s", keyword))
			}
		}
	}

	if info.WebGLRenderer != "" {
		rendererLower := strings.ToLower(info.WebGLRenderer)
		vmPatterns := []string{"vmware", "virtualbox", "virtual", "parallels"}
		for _, pattern := range vmPatterns {
			if strings.Contains(rendererLower, pattern) {
				isVPN = true
				confidence = math.Max(confidence, 0.75)
				evidence = append(evidence, fmt.Sprintf("WebGL渲染器检测到虚拟机: %s", pattern))
			}
		}
	}

	return isVPN, confidence, evidence
}

func (d *EnvDetector) EnhancedEnvCheck(info *EnvInfo) *EnvDetectionReport {
	report := d.RunAllChecks(info)

	canvasAnomalies := d.DetectCanvasAnomalies(info)
	if len(canvasAnomalies) > 0 {
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "canvas_anomaly",
			Risk:     "medium",
			Detected: true,
			Score:    20,
			Reason:   strings.Join(canvasAnomalies, "; "),
		})
	}

	webglAnalysis := d.AnalyzeWebGLDetails(info)
	if risk, ok := webglAnalysis["risk"].(string); ok && risk != "low" {
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "webgl_anomaly",
			Risk:     risk,
			Detected: true,
			Score:    15,
			Reason:   webglAnalysis["reason"].(string),
		})
	}

	emulatorDetected, emulatorIndicators := d.DetectEmulatorIndicators(info)
	if emulatorDetected {
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "emulator_detected",
			Risk:     "medium",
			Detected: true,
			Score:    30,
			Reason:   strings.Join(emulatorIndicators, "; "),
		})
	}

	return report
}

func (s *EnvDetectorService) EnhancedVerifyWithEnv(sessionID string, req *EnvVerifyRequest) (*EnvVerifyResponse, error) {
	envInfo := &EnvInfo{
		UserAgent:           req.UserAgent,
		Platform:            req.EnvironmentEnv.Platform,
		Language:            req.EnvironmentEnv.Language,
		Languages:           req.EnvironmentEnv.Languages,
		ScreenWidth:         req.EnvironmentEnv.ScreenWidth,
		ScreenHeight:        req.EnvironmentEnv.ScreenHeight,
		ColorDepth:          req.EnvironmentEnv.ColorDepth,
		PixelRatio:          req.EnvironmentEnv.PixelRatio,
		Timezone:            req.EnvironmentEnv.Timezone,
		TimezoneOffset:      req.EnvironmentEnv.TimezoneOffset,
		CanvasFingerprint:   req.EnvironmentEnv.CanvasFingerprint,
		WebGLRenderer:       req.EnvironmentEnv.WebGLRenderer,
		WebGLVendor:         req.EnvironmentEnv.WebGLVendor,
		Plugins:             req.EnvironmentEnv.Plugins,
		Fonts:               req.EnvironmentEnv.Fonts,
		TouchSupport:        req.EnvironmentEnv.TouchSupport,
		MaxTouchPoints:      req.EnvironmentEnv.MaxTouchPoints,
		HardwareConcurrency: req.EnvironmentEnv.HardwareConcurrency,
		Fingerprint:         req.Fingerprint,
	}

	if envInfo.UserAgent == "" {
		envInfo.UserAgent = req.UserAgent
	}

	blacklisted, reason := s.blacklistSvc.CheckBlacklist(req.IPAddress, "ip")
	if blacklisted {
		return &EnvVerifyResponse{
			Success:     false,
			RiskLevel:   "high",
			RiskScore:   100.0,
			RiskFactors: []string{"IP黑名单: " + reason.Error()},
			Action:      "block",
			Message:     "IP已被列入黑名单",
		}, nil
	}

	if req.Fingerprint != "" {
		blacklisted, reason = s.blacklistSvc.CheckBlacklist(req.Fingerprint, "device_id")
		if blacklisted {
			return &EnvVerifyResponse{
				Success:     false,
				RiskLevel:   "high",
				RiskScore:   100.0,
				RiskFactors: []string{"设备黑名单: " + reason.Error()},
				Action:      "block",
				Message:     "设备已被列入黑名单",
			}, nil
		}
	}

	ipRateLimitConfig := &RateLimitConfig{
		MaxRequests: 100,
		WindowSecs:  60,
	}
	rateLimitResult, err := s.rateLimitSvc.CheckIPRateLimit(context.Background(), req.IPAddress, ipRateLimitConfig)
	if err == nil && !rateLimitResult.Allowed {
		return &EnvVerifyResponse{
			Success:     false,
			RiskLevel:   "medium",
			RiskScore:   70.0,
			RiskFactors: []string{"IP请求频率超限"},
			Action:      "review",
			Message:     "请求过于频繁，请稍后再试",
		}, nil
	}

	if req.Fingerprint != "" {
		fpRateLimitConfig := &RateLimitConfig{
			MaxRequests: 50,
			WindowSecs:  60,
		}
		rateLimitResult, err = s.rateLimitSvc.CheckIPRateLimit(context.Background(), req.Fingerprint, fpRateLimitConfig)
		if err == nil && !rateLimitResult.Allowed {
			return &EnvVerifyResponse{
				Success:     false,
				RiskLevel:   "medium",
				RiskScore:   65.0,
				RiskFactors: []string{"设备请求频率超限"},
				Action:      "review",
				Message:     "请求过于频繁，请稍后再试",
			}, nil
		}
	}

	enhancedReport := s.envDetector.EnhancedEnvCheck(envInfo)

	headers := make(map[string]string)
	proxyRisk := s.envDetector.CalculateProxyRiskScore(req.IPAddress, headers)
	if proxyRisk > 30 {
		enhancedReport.EnvScore -= proxyRisk * 0.2
		enhancedReport.Checks = append(enhancedReport.Checks, RiskCheckResult{
			Name:     "proxy_risk",
			Risk:     "medium",
			Detected: true,
			Score:    int(proxyRisk),
			Reason:   fmt.Sprintf("代理风险评分: %.2f", proxyRisk),
		})
	}

	isVPN, vpnConfidence, vpnEvidence := s.envDetector.DetectVPNPatterns(envInfo, headers)
	if isVPN {
		enhancedReport.EnvScore -= vpnConfidence * 20
		enhancedReport.Checks = append(enhancedReport.Checks, RiskCheckResult{
			Name:     "vpn_detected",
			Risk:     "medium",
			Detected: true,
			Score:    int(vpnConfidence * 100),
			Reason:   strings.Join(vpnEvidence, "; "),
		})
	}

	emulatorDetected, emulatorIndicators := s.envDetector.DetectEmulatorIndicators(envInfo)
	if emulatorDetected {
		enhancedReport.EnvScore -= 25
		enhancedReport.Checks = append(enhancedReport.Checks, RiskCheckResult{
			Name:     "emulator_detected",
			Risk:     "medium",
			Detected: true,
			Score:    30,
			Reason:   strings.Join(emulatorIndicators, "; "),
		})
	}

	if enhancedReport.EnvScore < 0 {
		enhancedReport.EnvScore = 0
	}

	riskLevel := "low"
	if enhancedReport.EnvScore < 60 {
		riskLevel = "high"
	} else if enhancedReport.EnvScore < 80 {
		riskLevel = "medium"
	}

	action := "pass"
	if enhancedReport.EnvScore < 50 || emulatorDetected {
		action = "block"
	} else if enhancedReport.EnvScore < 70 {
		action = "review"
	}

	captchaPass := true
	if action == "block" || action == "review" {
		captchaPass = false
	}

	return &EnvVerifyResponse{
		Success:     true,
		RiskLevel:   riskLevel,
		RiskScore:   enhancedReport.EnvScore,
		RiskFactors: enhancedReport.DetectedTools,
		Action:      action,
		Message:     "环境检测通过",
		CaptchaPass: captchaPass,
	}, nil
}

func (d *EnvDetector) DetectVMFeatures(info *EnvInfo, frontendDetections []string) (bool, float64, []string) {
	detected := false
	score := 0.0
	evidence := []string{}

	vmPatterns := []string{
		"vmware", "virtualbox", "parallels", "hyperv", "qemu", "kvm", "xen",
		"virtual", "vbox", "vboxservice", "vboxguest",
	}

	uaLower := strings.ToLower(info.UserAgent)
	platformLower := strings.ToLower(info.Platform)
	rendererLower := strings.ToLower(info.WebGLRenderer)
	vendorLower := strings.ToLower(info.WebGLVendor)

	for _, pattern := range vmPatterns {
		if strings.Contains(uaLower, pattern) || strings.Contains(platformLower, pattern) {
			detected = true
			score += 35.0
			evidence = append(evidence, fmt.Sprintf("VM特征-UserAgent/Platform: %s", pattern))
		}
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			detected = true
			score += 40.0
			evidence = append(evidence, fmt.Sprintf("VM特征-WebGL渲染器: %s", pattern))
		}
	}

	if info.HardwareConcurrency == 1 {
		detected = true
		score += 20.0
		evidence = append(evidence, "VM特征-单核CPU")
	} else if info.HardwareConcurrency == 2 {
		detected = true
		score += 10.0
		evidence = append(evidence, "VM特征-双核CPU")
	}

	if info.HardwareConcurrency > 0 && info.HardwareConcurrency < 2 {
		detected = true
		score += 15.0
		evidence = append(evidence, fmt.Sprintf("VM特征-低CPU核心数: %d", info.HardwareConcurrency))
	}

	softwareRenderers := []string{"swiftshader", "llvmpipe", "mesa", "software", "emulated"}
	for _, renderer := range softwareRenderers {
		if strings.Contains(rendererLower, renderer) {
			detected = true
			score += 35.0
			evidence = append(evidence, fmt.Sprintf("VM特征-软件渲染器: %s", renderer))
		}
	}

	if len(frontendDetections) > 0 {
		for _, detection := range frontendDetections {
			if strings.Contains(detection, "vm_") || strings.Contains(detection, "cpu_cores") || strings.Contains(detection, "device_memory") {
				detected = true
				score += 25.0
				evidence = append(evidence, fmt.Sprintf("前端VM检测: %s", detection))
			}
		}
	}

	return detected, math.Min(score, 100.0), evidence
}

func (d *EnvDetector) DetectSandboxEscape(info *EnvInfo, frontendDetections []string) (bool, float64, []string) {
	detected := false
	score := 0.0
	evidence := []string{}

	nodeIndicators := []string{
		"node_env", "require_available", "require_defined", "process_object",
		"module_exports", "node_path_vars", "global_object",
	}

	for _, indicator := range nodeIndicators {
		found := false
		for _, detection := range frontendDetections {
			if strings.Contains(strings.ToLower(detection), indicator) {
				found = true
				break
			}
		}
		if found {
			detected = true
			score += 30.0
			evidence = append(evidence, fmt.Sprintf("沙箱逃逸-Node环境: %s", indicator))
		}
	}

	sandboxFilePatterns := []string{
		"vboxservice", "vboxguest", "vmware-toolbox", "vboxcontrol",
		"vmware fusion", "virtualbox", "vboxmouse", "vboxservice",
	}

	for _, pattern := range sandboxFilePatterns {
		uaLower := strings.ToLower(info.UserAgent)
		if strings.Contains(uaLower, pattern) {
			detected = true
			score += 35.0
			evidence = append(evidence, fmt.Sprintf("沙箱逃逸-文件路径: %s", pattern))
		}
	}

	sandboxPlugins := []string{"vbox", "vmware", "virtual", "sandbox"}
	if len(info.Plugins) > 0 {
		for _, plugin := range info.Plugins {
			pluginLower := strings.ToLower(plugin)
			for _, sandbox := range sandboxPlugins {
				if strings.Contains(pluginLower, sandbox) {
					detected = true
					score += 30.0
					evidence = append(evidence, fmt.Sprintf("沙箱逃逸-插件: %s", plugin))
					break
				}
			}
		}
	}

	for _, detection := range frontendDetections {
		detectionLower := strings.ToLower(detection)
		if strings.Contains(detectionLower, "sandbox_") || strings.Contains(detectionLower, "node_") {
			if !strings.Contains(detectionLower, "virtualbox") && !strings.Contains(detectionLower, "vmware") {
				detected = true
				score += 25.0
				evidence = append(evidence, fmt.Sprintf("前端沙箱检测: %s", detection))
			}
		}
	}

	return detected, math.Min(score, 100.0), evidence
}

func (d *EnvDetector) DetectDebuggerEnhanced(info *EnvInfo, frontendDetections []string) (bool, float64, []string) {
	detected := false
	score := 0.0
	evidence := []string{}

	devtoolsPatterns := []string{
		"devtools_", "debugger_", "stack_debugger", "console_",
		"define_property_debugger", "execution_paused", "debugger_prop:",
		"timing_high_variance", "timing_too_slow", "date_loop_inhibited",
	}

	for _, pattern := range devtoolsPatterns {
		for _, detection := range frontendDetections {
			if strings.Contains(strings.ToLower(detection), pattern) {
				detected = true
				score += 25.0
				evidence = append(evidence, fmt.Sprintf("调试器检测: %s", detection))
				break
			}
		}
	}

	uaLower := strings.ToLower(info.UserAgent)
	debuggerIndicators := []string{
		"debugger", "__webdriver", "__selenium", "__fxdriver",
		"__driver", "__webdriver_script",
	}

	for _, indicator := range debuggerIndicators {
		if strings.Contains(uaLower, indicator) {
			detected = true
			score += 30.0
			evidence = append(evidence, fmt.Sprintf("调试器特征-UserAgent: %s", indicator))
		}
	}

	if info.WebGLRenderer == "" && info.WebGLVendor == "" {
		detected = true
		score += 20.0
		evidence = append(evidence, "调试器检测-WebGL信息缺失")
	}

	if info.CanvasFingerprint == "" {
		detected = true
		score += 15.0
		evidence = append(evidence, "调试器检测-Canvas指纹缺失")
	}

	if len(info.Languages) == 0 || len(info.Languages) == 1 && info.Language == "" {
		detected = true
		score += 10.0
		evidence = append(evidence, "调试器检测-语言信息异常")
	}

	if info.ScreenWidth == 0 || info.ScreenHeight == 0 {
		detected = true
		score += 15.0
		evidence = append(evidence, "调试器检测-屏幕尺寸异常")
	}

	return detected, math.Min(score, 100.0), evidence
}

func (d *EnvDetector) EnhancedVMCheck(info *EnvInfo, frontendDetections []string) *EnvDetectionReport {
	report := d.RunAllChecks(info)

	vmDetected, vmScore, vmEvidence := d.DetectVMFeatures(info, frontendDetections)
	if vmDetected {
		report.EnvScore -= vmScore * 0.3
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "vm_features_detected",
			Risk:     "high",
			Detected: true,
			Score:    int(vmScore),
			Reason:   strings.Join(vmEvidence, "; "),
		})
	}

	sandboxDetected, sandboxScore, sandboxEvidence := d.DetectSandboxEscape(info, frontendDetections)
	if sandboxDetected {
		report.EnvScore -= sandboxScore * 0.3
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "sandbox_escape_detected",
			Risk:     "high",
			Detected: true,
			Score:    int(sandboxScore),
			Reason:   strings.Join(sandboxEvidence, "; "),
		})
	}

	debuggerDetected, debuggerScore, debuggerEvidence := d.DetectDebuggerEnhanced(info, frontendDetections)
	if debuggerDetected {
		report.EnvScore -= debuggerScore * 0.2
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "debugger_detected",
			Risk:     "medium",
			Detected: true,
			Score:    int(debuggerScore),
			Reason:   strings.Join(debuggerEvidence, "; "),
		})
	}

	if report.EnvScore < 0 {
		report.EnvScore = 0
	}

	return report
}

type BrowserFingerprint struct {
	Hash           string
	Components     map[string]string
	AnomalyScore   float64
	FingerprintID  string
}

type EnvFingerprintAnalysis struct {
	Browser       string
	Version       string
	OS            string
	IsSuspicious  bool
	SuspiciousFeatures []string
}

func (d *EnvDetector) AnalyzeBrowserFingerprint(info *EnvInfo) *EnvFingerprintAnalysis {
	analysis := &EnvFingerprintAnalysis{
		IsSuspicious:      false,
		SuspiciousFeatures: []string{},
	}

	browser, version := d.parseUserAgent(info.UserAgent)
	analysis.Browser = browser
	analysis.Version = version
	analysis.OS = d.detectOS(info.UserAgent, info.Platform)

	if info.CanvasFingerprint == "" {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "missing_canvas")
	}

	if len(info.CanvasFingerprint) < 32 {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "short_canvas_fingerprint")
	}

	webglRisk := d.AnalyzeWebGLDetails(info)
	if risk, ok := webglRisk["risk"].(string); ok && risk == "high" {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "suspicious_webgl")
	}

	if info.WebGLRenderer == "" || info.WebGLVendor == "" {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "missing_webgl_info")
	}

	uaLower := strings.ToLower(info.UserAgent)
	if strings.Contains(uaLower, "headless") ||
	   strings.Contains(uaLower, "phantom") ||
	   strings.Contains(uaLower, "puppeteer") ||
	   strings.Contains(uaLower, "playwright") ||
	   strings.Contains(uaLower, "selenium") {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "automation_framework_ua")
	}

	if len(info.Fonts) < 3 {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "limited_fonts")
	}

	if len(info.Plugins) == 0 {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "no_plugins")
	}

	if len(info.Languages) == 0 || (len(info.Languages) == 1 && info.Language == "") {
		analysis.IsSuspicious = true
		analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "abnormal_languages")
	}

	anomalies := d.DetectCanvasAnomalies(info)
	if len(anomalies) > 0 {
		analysis.IsSuspicious = true
		for _, anomaly := range anomalies {
			analysis.SuspiciousFeatures = append(analysis.SuspiciousFeatures, "canvas_"+anomaly)
		}
	}

	return analysis
}

func (d *EnvDetector) parseUserAgent(ua string) (browser string, version string) {
	uaLower := strings.ToLower(ua)

	if strings.Contains(uaLower, "edg/") {
		browser = "Edge"
		if idx := strings.Index(uaLower, "edg/"); idx != -1 {
			version = d.extractVersion(ua[idx+4:])
		}
		return
	}

	browserPatterns := []struct {
		Name    string
		Pattern string
	}{
		{"Chrome", "chrome/"},
		{"Firefox", "firefox/"},
		{"Safari", "safari/"},
		{"Opera", "opera/"},
		{"IE", "msie "},
		{"IE", "trident/"},
	}

	for _, bp := range browserPatterns {
		if strings.Contains(uaLower, bp.Pattern) {
			browser = bp.Name
			if idx := strings.Index(uaLower, bp.Pattern); idx != -1 {
				version = d.extractVersion(ua[idx+len(bp.Pattern):])
			}
			return
		}
	}

	return "Unknown", "0.0"
}

func (d *EnvDetector) extractVersion(versionStr string) string {
	parts := strings.Split(versionStr, ".")
	if len(parts) == 0 {
		return "0.0"
	}
	
	end := 0
	for i, c := range versionStr {
		if c == ' ' || c == ')' || c == '/' || c == ';' {
			end = i
			break
		}
		end = i + 1
	}
	
	return strings.TrimSpace(versionStr[:end])
}

func (d *EnvDetector) detectOS(ua string, platform string) string {
	uaLower := strings.ToLower(ua)
	platformLower := strings.ToLower(platform)

	if strings.Contains(uaLower, "windows") || strings.Contains(platformLower, "win") {
		return "Windows"
	}
	if strings.Contains(uaLower, "mac os") || strings.Contains(uaLower, "macos") || strings.Contains(platformLower, "mac") {
		return "macOS"
	}
	if strings.Contains(uaLower, "linux") && !strings.Contains(uaLower, "android") {
		return "Linux"
	}
	if strings.Contains(uaLower, "android") {
		return "Android"
	}
	if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "ios") {
		return "iOS"
	}

	return "Unknown"
}

func (d *EnvDetector) CalculateFingerprintEntropy(info *EnvInfo) float64 {
	entropy := 0.0

	if info.CanvasFingerprint != "" {
		entropy += float64(len(info.CanvasFingerprint)) * 0.5
	}

	if info.WebGLRenderer != "" {
		entropy += float64(len(info.WebGLRenderer)) * 0.3
	}

	if len(info.Fonts) > 0 {
		entropy += float64(len(info.Fonts)) * 1.5
	}

	if len(info.Languages) > 0 {
		entropy += float64(len(info.Languages)) * 2.0
	}

	if info.ScreenWidth > 0 && info.ScreenHeight > 0 {
		entropy += 4.0
	}

	if info.Timezone != "" {
		entropy += 3.0
	}

	return math.Min(entropy, 100.0)
}

type NetworkEnvironmentAnalysis struct {
	IsVPN        bool
	IsTor        bool
	IsProxy      bool
	IsDatacenter bool
	ASN          int
	Country      string
	RiskScore    float64
	Evidence     []string
}

func (d *EnvDetector) AnalyzeNetworkEnvironment(ip string, headers map[string]string, info *EnvInfo) *NetworkEnvironmentAnalysis {
	analysis := &NetworkEnvironmentAnalysis{
		RiskScore: 0,
		Evidence:  []string{},
	}

	proxyRisk := d.CalculateProxyRiskScore(ip, headers)
	analysis.RiskScore += proxyRisk * 0.3

	if proxyRisk > 30 {
		analysis.IsProxy = true
		analysis.Evidence = append(analysis.Evidence, fmt.Sprintf("Proxy risk score: %.2f", proxyRisk))
	}

	isVPN, vpnConfidence, vpnEvidence := d.DetectVPNPatterns(info, headers)
	if isVPN {
		analysis.IsVPN = true
		analysis.RiskScore += vpnConfidence * 30
		analysis.Evidence = append(analysis.Evidence, vpnEvidence...)
	}

	xff := headers["X-Forwarded-For"]
	xri := headers["X-Real-IP"]
	via := headers["Via"]

	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 2 {
			analysis.IsProxy = true
			analysis.RiskScore += 15
			analysis.Evidence = append(analysis.Evidence, "Multiple proxy hops detected via X-Forwarded-For")
		}
	}

	if xri != "" && xri != ip {
		analysis.IsProxy = true
		analysis.RiskScore += 10
		analysis.Evidence = append(analysis.Evidence, "X-Real-IP differs from connecting IP")
	}

	if via != "" {
		viaLower := strings.ToLower(via)
		if strings.Contains(viaLower, "proxy") || strings.Contains(viaLower, "squid") {
			analysis.IsProxy = true
			analysis.RiskScore += 15
			analysis.Evidence = append(analysis.Evidence, "Proxy detected via Via header")
		}
	}

	datacenterRanges := []string{
		"52.94.", "54.240.", "35.180.", "18.228.",
		"45.33.", "104.238.", "107.170.", "159.89.",
		"128.31.", "199.87.", "199.58.", "171.25.",
	}

	for _, range_ := range datacenterRanges {
		if strings.HasPrefix(ip, range_) {
			analysis.IsDatacenter = true
			analysis.RiskScore += 20
			analysis.Evidence = append(analysis.Evidence, fmt.Sprintf("IP in datacenter range: %s", range_))
			break
		}
	}

	torExitNodes := []string{
		"128.31.0.", "199.87.154.", "199.58.186.",
		"171.25.193.", "162.247.72.", "45.33.32.",
		"104.244.76.", "77.247.181.", "93.95.227.",
	}

	for _, torIP := range torExitNodes {
		if strings.HasPrefix(ip, torIP) {
			analysis.IsTor = true
			analysis.RiskScore += 50
			analysis.Evidence = append(analysis.Evidence, "Known Tor exit node IP range")
			break
		}
	}

	if analysis.RiskScore > 70 {
		analysis.RiskScore = 70
	}

	return analysis
}

func (d *EnvDetector) DetectProxyViaHeaders(headers map[string]string) (bool, float64, []string) {
	detected := false
	confidence := 0.0
	evidence := []string{}

	proxyHeaders := map[string]string{
		"X-Forwarded-For":   "X-Forwarded-For header present",
		"X-Real-IP":        "X-Real-IP header present",
		"Via":              "Via header present",
		"X-Proxy-ID":       "Proxy ID header present",
		"X-ProxyChain":     "Proxy chain header present",
		"Forwarded":        "Forwarded header present",
		"X-CLIENT-IP":      "Client IP header present",
		"True-Client-IP":    "True Client IP header present",
		"CF-Connecting-IP": "Cloudflare IP header present",
		"X-Sucuri-ID":      "Sucuri proxy header present",
		"X-CD-Real-IP":     "CDN real IP header present",
	}

	for header, desc := range proxyHeaders {
		if value, exists := headers[header]; exists && value != "" {
			detected = true
			confidence += 0.2
			evidence = append(evidence, fmt.Sprintf("%s: %s", desc, value))
		}
	}

	for header, value := range headers {
		headerLower := strings.ToLower(header)
		valueLower := strings.ToLower(value)
		
		if strings.Contains(headerLower, "forwarded") ||
		   strings.Contains(headerLower, "via") ||
		   strings.Contains(headerLower, "proxy") {
			detected = true
			confidence += 0.3
			evidence = append(evidence, fmt.Sprintf("Proxy indicator in header %s", header))
		}

		if strings.Contains(valueLower, "tor") || strings.Contains(valueLower, "onion") {
			detected = true
			confidence += 0.5
			evidence = append(evidence, "Tor-related header value detected")
		}

		if strings.Contains(valueLower, "vpn") || strings.Contains(valueLower, "virtual private") {
			detected = true
			confidence += 0.4
			evidence = append(evidence, "VPN-related header value detected")
		}
	}

	return detected, math.Min(confidence, 1.0), evidence
}

func (d *EnvDetector) DetectVPNViaASN(asn int) (bool, string) {
	vpnASN := map[int]string{
		201229: "Private Internet Access",
		212502: "CyberGhost",
		202132: "NordVPN",
		203378: "ExpressVPN",
		19679:  "Hide My Ass",
		49028:  "HotSpot Shield",
		393552: "Surfshark",
		206728: "IPVanish",
		35488:  "Astrill",
		9009:   "Mullvad",
		397185: "ProtonVPN",
		43260:  "TunnelBear",
	}

	if provider, exists := vpnASN[asn]; exists {
		return true, provider
	}

	asnRanges := []struct {
		Start  int
		End    int
		Provider string
	}{
		{201229, 209710, "NordVPN Group"},
		{203378, 203386, "ExpressVPN"},
		{202132, 202141, "NordVPN"},
	}

	for _, asnRange := range asnRanges {
		if asn >= asnRange.Start && asn <= asnRange.End {
			return true, asnRange.Provider
		}
	}

	return false, ""
}

func (d *EnvDetector) DetectTorExitNode(ip string) bool {
	torExitNodes := []string{
		"128.31.0.34", "128.31.0.39", "128.31.0.42",
		"199.87.154.10", "199.87.154.11", "199.87.154.22",
		"199.58.186.10", "199.58.186.11", "199.58.186.12",
		"171.25.193.9", "171.25.193.10", "171.25.193.11",
		"162.247.72.27", "162.247.72.28", "162.247.72.29",
		"45.33.32.156", "45.33.32.157", "45.33.32.158",
		"104.244.76.13", "104.244.76.14", "104.244.76.15",
		"77.247.181.218", "77.247.181.219", "77.247.181.220",
		"93.95.227.22", "93.95.227.23", "93.95.227.24",
	}

	for _, torIP := range torExitNodes {
		if ip == torIP || strings.HasPrefix(ip, torIP[:strings.LastIndex(torIP, ".")]) {
			return true
		}
	}

	return false
}

type EnhancedEnvDetectorV3 struct {
	*EnvDetector
	canvasAnalyzer  *CanvasFingerprintAnalyzer
	webglAnalyzer   *WebGLFingerprintAnalyzer
	headlessDetector *HeadlessBrowserDetector
}

type CanvasFingerprintAnalyzer struct {
	commonHashes map[string]float64
	entropyMap   map[string]float64
}

type WebGLFingerprintAnalyzer struct {
	softwareRenders map[string]float64
	rendererDB      map[string]*RendererInfo
}

type HeadlessBrowserDetector struct {
	signatures map[string]*HeadlessSignature
	evasionDB  map[string]bool
}

type HeadlessSignature struct {
	Name        string
	Patterns    []string
	Weight      float64
	Description string
}

type RendererInfo struct {
	Vendor       string
	Renderer     string
	IsSoftware   bool
	IsVirtual    bool
	RiskLevel    string
}

type CanvasAnalysisResult struct {
	Hash               string
	Entropy            float64
	AnomalyScore       float64
	AnomalyReasons     []string
	IsSuspicious       bool
	CommonHashMatch    bool
	MatchPercentage    float64
}

type WebGLAnalysisResultV3 struct {
	Vendor             string
	Renderer           string
	IsSoftware         bool
	IsVirtualMachine   bool
	RiskScore          float64
	DetectedIndicators []string
	Recommendations    []string
}

type HeadlessDetectionResult struct {
	IsHeadless         bool
	BrowserType        string
	Confidence         float64
	DetectedSignatures []string
	EvasionAttempts    int
	RiskLevel          string
}

func NewEnhancedEnvDetectorV3() *EnhancedEnvDetectorV3 {
	detector := &EnhancedEnvDetectorV3{
		EnvDetector:        NewEnvDetectorBackend(),
		canvasAnalyzer:     NewCanvasFingerprintAnalyzer(),
		webglAnalyzer:      NewWebGLFingerprintAnalyzer(),
		headlessDetector:  NewHeadlessBrowserDetector(),
	}
	detector.initializeAnalyzers()
	return detector
}

func NewCanvasFingerprintAnalyzer() *CanvasFingerprintAnalyzer {
	return &CanvasFingerprintAnalyzer{
		commonHashes: make(map[string]float64),
		entropyMap:   make(map[string]float64),
	}
}

func NewWebGLFingerprintAnalyzer() *WebGLFingerprintAnalyzer {
	return &WebGLFingerprintAnalyzer{
		softwareRenders: make(map[string]float64),
		rendererDB:      make(map[string]*RendererInfo),
	}
}

func NewHeadlessBrowserDetector() *HeadlessBrowserDetector {
	return &HeadlessBrowserDetector{
		signatures: make(map[string]*HeadlessSignature),
		evasionDB:  make(map[string]bool),
	}
}

func (d *EnhancedEnvDetectorV3) initializeAnalyzers() {
	d.canvasAnalyzer.initializeCommonHashes()
	d.webglAnalyzer.initializeRendererDB()
	d.headlessDetector.initializeSignatures()
}

func (c *CanvasFingerprintAnalyzer) initializeCommonHashes() {
	c.commonHashes = map[string]float64{
		"a1b2c3d4e5f6": 0.8,
		"1234567890ab": 0.7,
		"ffffffffffff": 0.6,
		"000000000000": 0.5,
		"deadbeef1234": 0.6,
		"abcd1234efgh": 0.5,
	}
	
	c.entropyMap = map[string]float64{
		"low_entropy":    0.3,
		"medium_entropy":  0.6,
		"high_entropy":    0.9,
	}
}

func (w *WebGLFingerprintAnalyzer) initializeRendererDB() {
	w.softwareRenders = map[string]float64{
		"swiftshader": 0.9,
		"llvmpipe":    0.85,
		"mesa":        0.7,
		"software":    0.8,
		"emulated":    0.75,
	}
	
	w.rendererDB = map[string]*RendererInfo{
		"swiftshader": {
			Vendor:      "Google",
			Renderer:    "SwiftShader",
			IsSoftware:  true,
			IsVirtual:   false,
			RiskLevel:   "high",
		},
		"llvmpipe": {
			Vendor:      "LLVM",
			Renderer:    "llvmpipe",
			IsSoftware:  true,
			IsVirtual:   false,
			RiskLevel:   "high",
		},
		"vmware": {
			Vendor:      "VMware",
			Renderer:    "VMware",
			IsSoftware:  false,
			IsVirtual:   true,
			RiskLevel:   "medium",
		},
		"virtualbox": {
			Vendor:      "Oracle",
			Renderer:    "VirtualBox",
			IsSoftware:  false,
			IsVirtual:   true,
			RiskLevel:   "medium",
		},
	}
}

func (h *HeadlessBrowserDetector) initializeSignatures() {
	h.signatures = map[string]*HeadlessSignature{
		"navigator_webdriver": {
			Name:        "Navigator WebDriver",
			Patterns:    []string{"navigator.webdriver"},
			Weight:      0.9,
			Description: "navigator.webdriver is true",
		},
		"chrome_runtime": {
			Name:        "Chrome Runtime Missing",
			Patterns:    []string{"chrome.runtime"},
			Weight:      0.7,
			Description: "Chrome runtime object missing",
		},
		"permissions_api": {
			Name:        "Permissions API Anomaly",
			Patterns:    []string{"permissions.query"},
			Weight:      0.6,
			Description: "Permissions API behaves abnormally",
		},
		"user_agent": {
			Name:        "Headless User Agent",
			Patterns:    []string{"HeadlessChrome", "Headless"},
			Weight:      0.8,
			Description: "User agent contains Headless",
		},
		"window_outer": {
			Name:        "Zero Window Size",
			Patterns:    []string{"window.outer"},
			Weight:      0.75,
			Description: "Window outer dimensions are zero",
		},
		"plugins_missing": {
			Name:        "Plugins Missing",
			Patterns:    []string{"navigator.plugins"},
			Weight:      0.5,
			Description: "No plugins detected",
		},
		"languages_empty": {
			Name:        "Empty Languages",
			Patterns:    []string{"navigator.languages"},
			Weight:      0.5,
			Description: "Navigator languages array is empty",
		},
		"webgl_debug": {
			Name:        "WebGL Debug Blocked",
			Patterns:    []string{"WEBGL_debug_renderer_info"},
			Weight:      0.65,
			Description: "WebGL debug renderer info blocked",
		},
	}
	
	h.evasionDB = map[string]bool{
		"navigator.webdriver = undefined": true,
		"Object.defineProperty(navigator, 'webdriver', {get: () => false})": true,
		"chrome.runtime undefined": true,
	}
}

func (d *EnhancedEnvDetectorV3) AnalyzeCanvasFingerprintV3(canvasHash string) *CanvasAnalysisResult {
	result := &CanvasAnalysisResult{
		Hash:               canvasHash,
		AnomalyReasons:     []string{},
	}
	
	if canvasHash == "" {
		result.AnomalyScore = 0.5
		result.AnomalyReasons = append(result.AnomalyReasons, "Empty canvas fingerprint")
		result.IsSuspicious = true
		return result
	}
	
	result.Entropy = d.calculateCanvasEntropy(canvasHash)
	
	if result.Entropy < 2.0 {
		result.AnomalyScore += 0.3
		result.AnomalyReasons = append(result.AnomalyReasons, "Low entropy fingerprint")
	}
	
	for commonHash, confidence := range d.canvasAnalyzer.commonHashes {
		similarity := d.calculateHashSimilarity(canvasHash, commonHash)
		if similarity > 0.8 {
			result.CommonHashMatch = true
			result.MatchPercentage = similarity
			result.AnomalyScore += confidence * 0.5
			result.AnomalyReasons = append(result.AnomalyReasons, fmt.Sprintf("Matches common hash (%.0f%%)", similarity*100))
		}
	}
	
	if len(canvasHash) < 32 {
		result.AnomalyScore += 0.2
		result.AnomalyReasons = append(result.AnomalyReasons, "Suspiciously short hash")
	}
	
	if d.hasRepeatingPattern(canvasHash) {
		result.AnomalyScore += 0.25
		result.AnomalyReasons = append(result.AnomalyReasons, "Repeating pattern detected")
	}
	
	result.IsSuspicious = result.AnomalyScore > 0.5
	result.AnomalyScore = math.Min(result.AnomalyScore, 1.0)
	
	return result
}

func (d *EnhancedEnvDetectorV3) calculateCanvasEntropy(hash string) float64 {
	if len(hash) == 0 {
		return 0.0
	}
	
	freq := make(map[rune]int)
	for _, char := range hash {
		freq[char]++
	}
	
	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / float64(len(hash))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	
	maxEntropy := math.Log2(float64(len(hash)))
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}
	return 0.0
}

func (d *EnhancedEnvDetectorV3) calculateHashSimilarity(hash1, hash2 string) float64 {
	if len(hash1) != len(hash2) {
		return 0.0
	}
	
	matches := 0
	for i := 0; i < len(hash1); i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}
	
	return float64(matches) / float64(len(hash1))
}

func (d *EnhancedEnvDetectorV3) hasRepeatingPattern(hash string) bool {
	if len(hash) < 4 {
		return false
	}
	
	for patternLen := 1; patternLen <= len(hash)/2; patternLen++ {
		pattern := hash[:patternLen]
		allSame := true
		for i := patternLen; i < len(hash); i += patternLen {
			end := i + patternLen
			if end > len(hash) {
				end = len(hash)
			}
			if hash[i:end] != pattern {
				allSame = false
				break
			}
		}
		if allSame && len(hash)/patternLen > 2 {
			return true
		}
	}
	
	return false
}

func (d *EnhancedEnvDetectorV3) AnalyzeWebGLFingerprintV3(renderer, vendor string) *WebGLAnalysisResultV3 {
	result := &WebGLAnalysisResultV3{
		Vendor:             vendor,
		Renderer:           renderer,
		DetectedIndicators: []string{},
		Recommendations:    []string{},
	}
	
	if renderer == "" || vendor == "" {
		result.RiskScore = 0.5
		result.DetectedIndicators = append(result.DetectedIndicators, "Missing WebGL information")
		return result
	}
	
	rendererLower := strings.ToLower(renderer)
	vendorLower := strings.ToLower(vendor)
	
	for softwareRender, risk := range d.webglAnalyzer.softwareRenders {
		if strings.Contains(rendererLower, softwareRender) {
			result.IsSoftware = true
			result.RiskScore = math.Max(result.RiskScore, risk)
			result.DetectedIndicators = append(result.DetectedIndicators, fmt.Sprintf("Software renderer: %s", softwareRender))
			result.Recommendations = append(result.Recommendations, "Consider blocking or requiring additional verification")
		}
	}
	
	vmPatterns := []string{"vmware", "virtualbox", "parallels", "hyperv", "qemu", "kvm"}
	for _, pattern := range vmPatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			result.IsVirtualMachine = true
			result.RiskScore = math.Max(result.RiskScore, 0.7)
			result.DetectedIndicators = append(result.DetectedIndicators, fmt.Sprintf("Virtual machine: %s", pattern))
		}
	}
	
	anonymizedPatterns := []string{"generic", "unknown", "default", "standard"}
	anonymizedCount := 0
	for _, pattern := range anonymizedPatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			anonymizedCount++
		}
	}
	if anonymizedCount >= 2 {
		result.RiskScore = math.Max(result.RiskScore, 0.6)
		result.DetectedIndicators = append(result.DetectedIndicators, "Anonymized WebGL information")
	}
	
	headlessPatterns := []string{"headless", "bot", "automation", "test"}
	for _, pattern := range headlessPatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			result.RiskScore = math.Max(result.RiskScore, 0.85)
			result.DetectedIndicators = append(result.DetectedIndicators, fmt.Sprintf("Headless indicator: %s", pattern))
		}
	}
	
	if result.RiskScore < 0.3 {
		result.Recommendations = append(result.Recommendations, "Normal WebGL fingerprint")
	}
	
	return result
}

func (d *EnhancedEnvDetectorV3) DetectHeadlessBrowserV3(info *EnvInfo, frontendData map[string]interface{}) *HeadlessDetectionResult {
	result := &HeadlessDetectionResult{
		DetectedSignatures: []string{},
	}
	
	totalWeight := 0.0
	detectedWeight := 0.0
	
	uaLower := strings.ToLower(info.UserAgent)
	for sigName, sig := range d.headlessDetector.signatures {
		weight := sig.Weight
		totalWeight += weight
		
		switch sigName {
		case "user_agent":
			for _, pattern := range sig.Patterns {
				if strings.Contains(uaLower, strings.ToLower(pattern)) {
					detectedWeight += weight
					result.DetectedSignatures = append(result.DetectedSignatures, sig.Description)
					break
				}
			}
		
		case "navigator_webdriver":
			if frontendData != nil {
				if webdriver, ok := frontendData["navigator_webdriver"].(bool); ok && webdriver {
					detectedWeight += weight
					result.DetectedSignatures = append(result.DetectedSignatures, sig.Description)
				}
			}
		
		case "plugins_missing":
			if len(info.Plugins) == 0 {
				detectedWeight += weight * 0.5
				result.DetectedSignatures = append(result.DetectedSignatures, sig.Description)
			}
		
		case "languages_empty":
			if len(info.Languages) == 0 {
				detectedWeight += weight * 0.5
				result.DetectedSignatures = append(result.DetectedSignatures, sig.Description)
			}
		
		case "window_outer":
			if frontendData != nil {
				if outerWidth, ok := frontendData["outer_width"].(float64); ok && outerWidth == 0 {
					detectedWeight += weight
					result.DetectedSignatures = append(result.DetectedSignatures, sig.Description)
				}
				if outerHeight, ok := frontendData["outer_height"].(float64); ok && outerHeight == 0 {
					detectedWeight += weight
					result.DetectedSignatures = append(result.DetectedSignatures, "Zero window height")
				}
			}
		
		case "webgl_debug":
			if info.WebGLRenderer == "" || info.WebGLVendor == "" {
				detectedWeight += weight * 0.5
				result.DetectedSignatures = append(result.DetectedSignatures, sig.Description)
			}
		}
	}
	
	if totalWeight > 0 {
		result.Confidence = detectedWeight / totalWeight
	}
	
	result.IsHeadless = result.Confidence > 0.6
	
	if result.Confidence > 0.8 {
		result.RiskLevel = "high"
		result.BrowserType = "definite_headless"
	} else if result.Confidence > 0.6 {
		result.RiskLevel = "medium"
		result.BrowserType = "probable_headless"
	} else if result.Confidence > 0.3 {
		result.RiskLevel = "low"
		result.BrowserType = "possible_headless"
	} else {
		result.RiskLevel = "none"
		result.BrowserType = "normal_browser"
	}
	
	if result.IsHeadless {
		if strings.Contains(uaLower, "headlesschrome") || strings.Contains(uaLower, "chrome-headless") {
			result.BrowserType = "chrome_headless"
		} else if strings.Contains(uaLower, "phantom") {
			result.BrowserType = "phantomjs"
		} else if strings.Contains(uaLower, "firefox-headless") {
			result.BrowserType = "firefox_headless"
		}
	}
	
	if frontendData != nil {
		result.EvasionAttempts = d.detectEvasionAttempts(frontendData)
		if result.EvasionAttempts > 0 {
			result.Confidence = math.Min(result.Confidence+0.2, 1.0)
		}
	}
	
	return result
}

func (d *EnhancedEnvDetectorV3) detectEvasionAttempts(frontendData map[string]interface{}) int {
	count := 0
	
	for key := range frontendData {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "evasion") || strings.Contains(keyLower, "stealth") {
			count++
		}
	}
	
	if overrideCount, ok := frontendData["property_overrides"].(int); ok {
		count += overrideCount
	}
	
	return count
}

func (d *EnhancedEnvDetectorV3) RunEnhancedEnvCheckV3(info *EnvInfo, frontendData map[string]interface{}) *EnvDetectionReport {
	report := d.EnvDetector.EnhancedEnvCheck(info)
	
	if info.CanvasFingerprint != "" {
		canvasResult := d.AnalyzeCanvasFingerprintV3(info.CanvasFingerprint)
		if canvasResult.IsSuspicious {
			report.EnvScore -= canvasResult.AnomalyScore * 20
			report.Checks = append(report.Checks, RiskCheckResult{
				Name:     "canvas_v3_anomaly",
				Risk:     "medium",
				Detected: true,
				Score:    int(canvasResult.AnomalyScore * 100),
				Reason:   strings.Join(canvasResult.AnomalyReasons, "; "),
			})
		}
	}
	
	webglResult := d.AnalyzeWebGLFingerprintV3(info.WebGLRenderer, info.WebGLVendor)
	if webglResult.RiskScore > 0.5 {
		report.EnvScore -= webglResult.RiskScore * 15
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "webgl_v3_anomaly",
			Risk:     "high",
			Detected: true,
			Score:    int(webglResult.RiskScore * 100),
			Reason:   strings.Join(webglResult.DetectedIndicators, "; "),
		})
	}
	
	headlessResult := d.DetectHeadlessBrowserV3(info, frontendData)
	if headlessResult.IsHeadless {
		report.EnvScore -= headlessResult.Confidence * 25
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "headless_browser_v3",
			Risk:     "high",
			Detected: true,
			Score:    int(headlessResult.Confidence * 100),
			Reason:   fmt.Sprintf("%s (%.0f%% confidence)", headlessResult.BrowserType, headlessResult.Confidence*100),
		})
	}
	
	if report.EnvScore < 0 {
		report.EnvScore = 0
	}
	
	if report.EnvScore < 60 {
		report.RiskLevel = "high"
		report.IsRisky = true
		report.Action = "block"
	} else if report.EnvScore < 80 {
		report.RiskLevel = "medium"
		if report.IsRisky {
			report.Action = "review"
		}
	}
	
	return report
}

func (d *EnhancedEnvDetectorV3) AnalyzeCanvasFingerprintAdvanced(canvasData string, additionalInfo map[string]interface{}) *CanvasAnalysisResult {
	result := &CanvasAnalysisResult{
		Hash:               canvasData,
		AnomalyReasons:     []string{},
	}
	
	if canvasData == "" {
		result.AnomalyScore = 0.7
		result.AnomalyReasons = append(result.AnomalyReasons, "Empty canvas fingerprint")
		result.IsSuspicious = true
		return result
	}
	
	result.Entropy = d.calculateCanvasEntropy(canvasData)
	
	if result.Entropy < 0.5 {
		result.AnomalyScore += 0.2
		result.AnomalyReasons = append(result.AnomalyReasons, "Very low entropy")
	}
	
	if len(canvasData) < 64 {
		result.AnomalyScore += 0.15
		result.AnomalyReasons = append(result.AnomalyReasons, "Unusually short fingerprint")
	}
	
	if additionalInfo != nil {
		if pixelData, ok := additionalInfo["pixel_variance"].(float64); ok && pixelData < 0.1 {
			result.AnomalyScore += 0.25
			result.AnomalyReasons = append(result.AnomalyReasons, "Low pixel variance")
		}
		
		if consistency, ok := additionalInfo["cross_session_consistency"].(float64); ok && consistency > 0.95 {
			result.AnomalyScore += 0.2
			result.AnomalyReasons = append(result.AnomalyReasons, "Suspiciously consistent across sessions")
		}
		
		if colorDepth, ok := additionalInfo["color_depth"].(int); ok && colorDepth < 24 {
			result.AnomalyScore += 0.15
			result.AnomalyReasons = append(result.AnomalyReasons, "Low color depth")
		}
	}
	
	result.IsSuspicious = result.AnomalyScore > 0.5
	result.AnomalyScore = math.Min(result.AnomalyScore, 1.0)
	
	return result
}

func (d *EnhancedEnvDetectorV3) AnalyzeWebGLFingerprintAdvanced(glInfo map[string]interface{}) *WebGLAnalysisResultV3 {
	result := &WebGLAnalysisResultV3{
		DetectedIndicators: []string{},
		Recommendations:    []string{},
	}
	
	if glInfo == nil {
		result.RiskScore = 0.6
		result.DetectedIndicators = append(result.DetectedIndicators, "Missing WebGL info")
		return result
	}
	
	if renderer, ok := glInfo["renderer"].(string); ok {
		result.Renderer = renderer
		rendererLower := strings.ToLower(renderer)
		
		for softwareRender, risk := range d.webglAnalyzer.softwareRenders {
			if strings.Contains(rendererLower, softwareRender) {
				result.IsSoftware = true
				result.RiskScore = math.Max(result.RiskScore, risk)
				result.DetectedIndicators = append(result.DetectedIndicators, "Software renderer: "+softwareRender)
			}
		}
	}
	
	if vendor, ok := glInfo["vendor"].(string); ok {
		result.Vendor = vendor
	}
	
	if extensions, ok := glInfo["extensions"].([]string); ok {
		if len(extensions) < 5 {
			result.RiskScore = math.Max(result.RiskScore, 0.4)
			result.DetectedIndicators = append(result.DetectedIndicators, fmt.Sprintf("Low extension count: %d", len(extensions)))
		}
	}
	
	if maxTexSize, ok := glInfo["max_texture_size"].(int); ok {
		if maxTexSize < 2048 {
			result.RiskScore = math.Max(result.RiskScore, 0.5)
			result.DetectedIndicators = append(result.DetectedIndicators, fmt.Sprintf("Low max texture size: %d", maxTexSize))
		}
	}
	
	if debugInfoAvailable, ok := glInfo["debug_info_available"].(bool); ok && !debugInfoAvailable {
		result.RiskScore = math.Max(result.RiskScore, 0.35)
		result.DetectedIndicators = append(result.DetectedIndicators, "WebGL debug info blocked")
	}
	
	return result
}
