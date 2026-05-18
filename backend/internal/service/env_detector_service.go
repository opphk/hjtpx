package service

import (
	"context"
	"encoding/json"
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
