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

func (d *EnvDetector) DetectAutomation(info *EnvInfo) *AutomationResult {
	result := &AutomationResult{
		Detected: false,
		Risks:    []string{},
	}

	uaLower := strings.ToLower(info.UserAgent)

	if strings.Contains(uaLower, "webdriver") {
		result.Risks = append(result.Risks, "Selenium WebDriver detected")
		result.Detected = true
	}

	if strings.Contains(uaLower, "headless") {
		result.Risks = append(result.Risks, "Headless Chrome detected")
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

	if strings.Contains(uaLower, "phantom") {
		result.Risks = append(result.Risks, "PhantomJS detected")
		result.Detected = true
	}

	if strings.Contains(uaLower, "puppeteer") {
		result.Risks = append(result.Risks, "Puppeteer detected")
		result.Detected = true
	}

	if strings.Contains(uaLower, "playwright") {
		result.Risks = append(result.Risks, "Playwright detected")
		result.Detected = true
	}

	if strings.Contains(uaLower, "selenium") {
		result.Risks = append(result.Risks, "Selenium detected")
		result.Detected = true
	}

	if info.Platform == "" {
		result.Risks = append(result.Risks, "Platform information missing")
	}

	return result
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
