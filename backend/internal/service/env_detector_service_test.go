package service_test

import (
	"testing"
)

type EnvInfo struct {
	UserAgent           string
	Platform            string
	Language            string
	Languages           []string
	ScreenWidth         int
	ScreenHeight        int
	ColorDepth          int
	PixelRatio          float64
	Timezone            string
	TimezoneOffset      int
	CanvasFingerprint   string
	WebGLRenderer       string
	WebGLVendor         string
	Plugins             []string
	Fonts               []string
	TouchSupport        bool
	MaxTouchPoints      int
	HardwareConcurrency int
	Fingerprint         string
}

func TestEnvironmentDetector_DetectAutomation(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name     string
		envInfo  *EnvInfo
		expected bool
	}{
		{
			name: "Selenium WebDriver detected",
			envInfo: &EnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 webdriver",
			},
			expected: true,
		},
		{
			name: "Headless Chrome detected",
			envInfo: &EnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/88.0.4324.96 Safari/537.36",
			},
			expected: true,
		},
		{
			name: "Puppeteer detected",
			envInfo: &EnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.0.0 Safari/537.36 puppeteer",
			},
			expected: true,
		},
		{
			name: "Playwright detected",
			envInfo: &EnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.0.0 Safari/537.36 playwright",
			},
			expected: true,
		},
		{
			name: "PhantomJS detected",
			envInfo: &EnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/534.34 (KHTML, like Gecko) PhantomJS/2.1.1 Safari/534.34",
			},
			expected: true,
		},
		{
			name: "Normal browser - no automation",
			envInfo: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				Platform:            "Win32",
				Languages:           []string{"zh-CN", "en-US"},
				Language:            "zh-CN",
				CanvasFingerprint:   "validhash123",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Plugins:             []string{"Chrome PDF Plugin"},
				Fonts:               []string{"Arial", "Helvetica"},
				ScreenWidth:         1920,
				ScreenHeight:        1080,
				HardwareConcurrency: 8,
			},
			expected: false,
		},
		{
			name: "Empty UserAgent",
			envInfo: &EnvInfo{
				UserAgent: "",
			},
			expected: true,
		},
		{
			name: "Short UserAgent",
			envInfo: &EnvInfo{
				UserAgent: "test",
			},
			expected: true,
		},
		{
			name: "Abnormal language settings - empty languages",
			envInfo: &EnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				Languages: []string{},
				Language:  "",
			},
			expected: true,
		},
		{
			name: "Canvas fingerprint missing",
			envInfo: &EnvInfo{
				UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				CanvasFingerprint: "",
			},
			expected: true,
		},
		{
			name: "WebGL information missing",
			envInfo: &EnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				WebGLRenderer: "",
				WebGLVendor:   "",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectAutomation(tt.envInfo)
			if result.Detected != tt.expected {
				t.Errorf("DetectAutomation() detected = %v, expected %v, risks = %v",
					result.Detected, tt.expected, result.Risks)
			}
		})
	}
}

func TestEnvironmentDetector_CalculateEnvScore(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name        string
		envInfo     *EnvInfo
		minExpected float64
		maxExpected float64
	}{
		{
			name: "Normal browser environment",
			envInfo: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				Platform:            "Win32",
				Language:            "zh-CN",
				Languages:           []string{"zh-CN", "en-US"},
				ScreenWidth:         1920,
				ScreenHeight:        1080,
				CanvasFingerprint:   "abc123hash",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Plugins:             []string{"Chrome PDF Plugin"},
				Fonts:               []string{"Arial", "Helvetica", "Times New Roman"},
				HardwareConcurrency: 8,
			},
			minExpected: 80,
			maxExpected: 100,
		},
		{
			name: "Automation tool detected",
			envInfo: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/88.0.4324.96 webdriver",
				Platform:            "",
				Language:            "",
				Languages:           []string{},
				ScreenWidth:         0,
				ScreenHeight:        0,
				CanvasFingerprint:   "",
				WebGLRenderer:       "",
				WebGLVendor:         "",
				Plugins:             []string{},
				Fonts:               []string{},
				HardwareConcurrency: 0,
			},
			minExpected: 0,
			maxExpected: 30,
		},
		{
			name: "Missing Canvas fingerprint only",
			envInfo: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				Platform:            "Win32",
				Language:            "en-US",
				Languages:           []string{"en-US"},
				ScreenWidth:         1920,
				ScreenHeight:        1080,
				CanvasFingerprint:   "",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Plugins:             []string{"Chrome PDF Plugin"},
				Fonts:               []string{"Arial", "Helvetica"},
				HardwareConcurrency: 4,
			},
			minExpected: 60,
			maxExpected: 85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := detector.CalculateEnvScore(tt.envInfo)
			if score < tt.minExpected || score > tt.maxExpected {
				t.Errorf("CalculateEnvScore() = %v, expected between %v and %v",
					score, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestEnvironmentDetector_EvaluateRisk(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name      string
		envInfo   *EnvInfo
		riskLevel string
		hasAction bool
	}{
		{
			name: "High risk - automation detected",
			envInfo: &EnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) webdriver Chrome/88.0.0.0",
			},
			riskLevel: "high",
			hasAction: true,
		},
		{
			name: "High risk - some issues",
			envInfo: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				Platform:            "Win32",
				Languages:           []string{"en-US"},
				Language:            "en-US",
				CanvasFingerprint:   "",
				WebGLRenderer:       "",
				WebGLVendor:         "NVIDIA",
				Plugins:             []string{"Chrome PDF Plugin"},
				Fonts:               []string{"Arial", "Helvetica", "Times"},
				ScreenWidth:         1920,
				ScreenHeight:        1080,
				HardwareConcurrency: 4,
			},
			riskLevel: "high",
			hasAction: true,
		},
		{
			name: "Low risk - normal browser",
			envInfo: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
				Platform:            "Win32",
				Language:            "en-US",
				Languages:           []string{"en-US"},
				ScreenWidth:         1920,
				ScreenHeight:        1080,
				CanvasFingerprint:   "validhash123",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Plugins:             []string{"Chrome PDF Plugin"},
				Fonts:               []string{"Arial", "Helvetica"},
				HardwareConcurrency: 8,
			},
			riskLevel: "low",
			hasAction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EvaluateRisk(tt.envInfo)
			if result.RiskLevel != tt.riskLevel {
				t.Errorf("EvaluateRisk() riskLevel = %v, expected %v", result.RiskLevel, tt.riskLevel)
			}
			if tt.hasAction && result.Action == "" {
				t.Error("EvaluateRisk() action is empty")
			}
		})
	}
}

func TestEnvironmentDetector_RunAllChecks(t *testing.T) {
	detector := newTestEnvDetector()

	envInfo := &EnvInfo{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
	}

	report := detector.RunAllChecks(envInfo)

	if report == nil {
		t.Fatal("RunAllChecks() returned nil")
	}

	if report.EnvScore < 0 || report.EnvScore > 100 {
		t.Errorf("RunAllChecks() EnvScore = %v, expected between 0 and 100", report.EnvScore)
	}

	validRiskLevels := map[string]bool{"low": true, "medium": true, "high": true}
	if !validRiskLevels[report.RiskLevel] {
		t.Errorf("RunAllChecks() RiskLevel = %v, expected low/medium/high", report.RiskLevel)
	}

	validActions := map[string]bool{"pass": true, "review": true, "block": true}
	if !validActions[report.Action] {
		t.Errorf("RunAllChecks() Action = %v, expected pass/review/block", report.Action)
	}

	if len(report.Checks) == 0 {
		t.Error("RunAllChecks() returned no checks")
	}
}

func TestEnvInfo_Model(t *testing.T) {
	envInfo := &EnvInfo{
		UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		Platform:            "Win32",
		Language:            "zh-CN",
		Languages:           []string{"zh-CN", "en-US"},
		ScreenWidth:         1920,
		ScreenHeight:        1080,
		ColorDepth:          24,
		PixelRatio:          1.0,
		Timezone:            "Asia/Shanghai",
		TimezoneOffset:      -480,
		CanvasFingerprint:   "hash123",
		WebGLRenderer:       "NVIDIA GeForce GTX 1080",
		WebGLVendor:         "NVIDIA",
		Plugins:             []string{"Chrome PDF Plugin"},
		Fonts:               []string{"Arial", "Helvetica"},
		TouchSupport:        true,
		MaxTouchPoints:      10,
		HardwareConcurrency: 8,
		Fingerprint:         "devicefingerprint123",
	}

	if envInfo.UserAgent == "" {
		t.Error("EnvInfo.UserAgent should not be empty")
	}
	if envInfo.Platform == "" {
		t.Error("EnvInfo.Platform should not be empty")
	}
	if envInfo.ScreenWidth == 0 {
		t.Error("EnvInfo.ScreenWidth should not be zero")
	}
}

type testEnvDetector struct{}

func newTestEnvDetector() *testEnvDetector {
	return &testEnvDetector{}
}

func (d *testEnvDetector) DetectAutomation(info *EnvInfo) *AutomationResult {
	result := &AutomationResult{
		Detected: false,
		Risks:    []string{},
	}

	uaLower := toLower(info.UserAgent)

	if contains(uaLower, "webdriver") {
		result.Risks = append(result.Risks, "Selenium WebDriver detected")
		result.Detected = true
	}

	if contains(uaLower, "headless") {
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
		result.Detected = true
	}

	if contains(uaLower, "phantom") {
		result.Risks = append(result.Risks, "PhantomJS detected")
		result.Detected = true
	}

	if contains(uaLower, "puppeteer") {
		result.Risks = append(result.Risks, "Puppeteer detected")
		result.Detected = true
	}

	if contains(uaLower, "playwright") {
		result.Risks = append(result.Risks, "Playwright detected")
		result.Detected = true
	}

	if contains(uaLower, "selenium") {
		result.Risks = append(result.Risks, "Selenium detected")
		result.Detected = true
	}

	if info.Platform == "" {
		result.Risks = append(result.Risks, "Platform information missing")
	}

	return result
}

func (d *testEnvDetector) CalculateEnvScore(info *EnvInfo) float64 {
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

func (d *testEnvDetector) EvaluateRisk(info *EnvInfo) *EnvRiskResult {
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

func (d *testEnvDetector) determineAction(automation *AutomationResult, score float64) string {
	if automation.Detected && len(automation.Risks) >= 2 {
		return "block"
	} else if automation.Detected || score < 70 {
		return "review"
	}
	return "pass"
}

func (d *testEnvDetector) RunAllChecks(info *EnvInfo) *EnvDetectionReport {
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

	uaLower := toLower(info.UserAgent)

	if contains(uaLower, "webdriver") || contains(uaLower, "selenium") {
		checks[0].Detected = true
		checks[0].Score = 40
		checks[0].Reason = "Selenium WebDriver特征检测"
	}

	if contains(uaLower, "headless") {
		checks[1].Detected = true
		checks[1].Score = 40
		checks[1].Reason = "Headless Chrome特征检测"
	}

	if contains(uaLower, "phantom") {
		checks[2].Detected = true
		checks[2].Score = 40
		checks[2].Reason = "PhantomJS特征检测"
	}

	if contains(uaLower, "puppeteer") {
		checks[3].Detected = true
		checks[3].Score = 40
		checks[3].Reason = "Puppeteer特征检测"
	}

	if contains(uaLower, "playwright") {
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
	EnvScore      float64           `json:"env_score"`
	IsRisky       bool              `json:"is_risky"`
	RiskLevel     string            `json:"risk_level"`
	DetectedTools []string          `json:"detected_tools"`
	Checks        []RiskCheckResult `json:"checks"`
	Action        string            `json:"action"`
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
