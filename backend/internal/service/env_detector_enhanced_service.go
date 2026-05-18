package service

import (
	"context"
	"encoding/json"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

type EnhancedEnvDetectorService struct {
	detector           *EnhancedEnvDetector
	blacklistSvc       *BlacklistService
	rateLimitSvc       *RateLimitService
	mu                 sync.RWMutex
	envCache           map[string]*EnhancedEnvInfo
	cacheExpiration    time.Duration
	detectionThreshold float64
}

type EnhancedEnvDetector struct{}

type EnhancedEnvInfo struct {
	UserAgent           string                       `json:"user_agent"`
	Platform            string                       `json:"platform"`
	Language            string                       `json:"language"`
	Languages           []string                     `json:"languages"`
	ScreenWidth         int                          `json:"screen_width"`
	ScreenHeight        int                          `json:"screen_height"`
	ColorDepth          int                          `json:"color_depth"`
	PixelRatio          float64                      `json:"pixel_ratio"`
	Timezone            string                       `json:"timezone"`
	TimezoneOffset      int                          `json:"timezone_offset"`
	CanvasFingerprint   string                       `json:"canvas_fingerprint"`
	WebGLRenderer       string                       `json:"webgl_renderer"`
	WebGLVendor         string                       `json:"webgl_vendor"`
	AudioFingerprint    string                       `json:"audio_fingerprint"`
	Fonts               []string                     `json:"fonts"`
	Plugins             []string                     `json:"plugins"`
	TouchSupport        bool                         `json:"touch_support"`
	MaxTouchPoints      int                          `json:"max_touch_points"`
	HardwareConcurrency int                          `json:"hardware_concurrency"`
	DeviceMemory        float64                      `json:"device_memory"`
	Fingerprint         string                       `json:"fingerprint"`
	WebRTCIPs          []string                     `json:"webrtc_ips"`
	ConnectionType      string                       `json:"connection_type"`
	Headers             map[string]string            `json:"headers"`
	DetectionResults    map[string]DetectionResult   `json:"detection_results"`
}

type DetectionResult struct {
	Detected bool     `json:"detected"`
	Score    float64  `json:"score"`
	Category string   `json:"category"`
	Details  []string `json:"details"`
}

type EnhancedAutomationResult struct {
	Detected         bool     `json:"detected"`
	Risks           []string `json:"risks"`
	AutomationType  string   `json:"automation_type"`
	Confidence      float64  `json:"confidence"`
	DetectionMethod string   `json:"detection_method"`
}

type VMDetectionResult struct {
	Detected     bool     `json:"detected"`
	VMType       string   `json:"vm_type"`
	RiskScore    float64  `json:"risk_score"`
	Indicators   []string `json:"indicators"`
	Confidence   float64  `json:"confidence"`
}

type EmulatorDetectionResult struct {
	Detected      bool     `json:"detected"`
	EmulatorType string   `json:"emulator_type"`
	RiskScore     float64  `json:"risk_score"`
	Indicators    []string `json:"indicators"`
	Confidence    float64  `json:"confidence"`
}

type DebugDetectionResult struct {
	Detected        bool     `json:"detected"`
	DebuggerType    string   `json:"debugger_type"`
	RiskScore       float64  `json:"risk_score"`
	Indicators      []string `json:"indicators"`
	IsOpen          bool     `json:"is_open"`
	Confidence      float64  `json:"confidence"`
}

type EnhancedEnvRiskResult struct {
	RiskLevel          string                    `json:"risk_level"`
	Score              float64                   `json:"score"`
	Risks              []string                  `json:"risks"`
	Action             string                    `json:"action"`
	AutomationResult   *EnhancedAutomationResult `json:"automation_result,omitempty"`
	VMResult           *VMDetectionResult        `json:"vm_result,omitempty"`
	EmulatorResult    *EmulatorDetectionResult   `json:"emulator_result,omitempty"`
	DebugResult       *DebugDetectionResult      `json:"debug_result,omitempty"`
}

type EnhancedRiskCheckResult struct {
	Name      string  `json:"name"`
	Risk     string  `json:"risk"`
	Detected bool    `json:"detected"`
	Score    int     `json:"score"`
	Reason   string  `json:"reason,omitempty"`
	Category string  `json:"category"`
}

type EnhancedEnvDetectionReport struct {
	Timestamp         int64                       `json:"timestamp"`
	EnvScore          float64                     `json:"env_score"`
	IsRisky           bool                        `json:"is_risky"`
	RiskLevel         string                      `json:"risk_level"`
	DetectedTools     []string                    `json:"detected_tools"`
	Checks            []EnhancedRiskCheckResult   `json:"checks"`
	Action            string                      `json:"action"`
	AutomationResult  *EnhancedAutomationResult   `json:"automation_result,omitempty"`
	VMResult          *VMDetectionResult         `json:"vm_result,omitempty"`
	EmulatorResult    *EmulatorDetectionResult    `json:"emulator_result,omitempty"`
	DebugResult       *DebugDetectionResult        `json:"debug_result,omitempty"`
	Confidence        float64                     `json:"confidence"`
	Recommendations   []string                    `json:"recommendations"`
	Accuracy          float64                     `json:"accuracy"`
}

type EnhancedEnvVerifyRequest struct {
	SessionID      string                 `json:"session_id"`
	Type           string                 `json:"type"`
	X              int                    `json:"x"`
	Y              int                    `json:"y"`
	Points         [][2]int               `json:"points"`
	ClickSequence  []int                  `json:"click_sequence"`
	BehaviorData   []BehaviorDataPoint    `json:"behavior_data"`
	SpeedData      json.RawMessage        `json:"speed_data,omitempty"`
	ApplicationID  uint                   `json:"application_id"`
	EnvironmentEnv EnhancedEnvInfo        `json:"environment_env"`
	Fingerprint     string                 `json:"fingerprint"`
	IPAddress      string                 `json:"ip_address"`
	UserAgent      string                 `json:"user_agent"`
}

type EnhancedEnvVerifyResponse struct {
	Success         bool                      `json:"success"`
	Message         string                    `json:"message"`
	RiskLevel       string                    `json:"risk_level"`
	RiskScore       float64                   `json:"risk_score"`
	RiskFactors     []string                  `json:"risk_factors"`
	Action          string                    `json:"action"`
	CaptchaPass     bool                      `json:"captcha_pass"`
	DetectionReport *EnhancedEnvDetectionReport `json:"detection_report,omitempty"`
}

var (
	vmPatterns = map[string][]*regexp.Regexp{
		"vmware": {
			regexp.MustCompile(`(?i)vmware|vmware[_-]?tools`),
			regexp.MustCompile(`(?i)vmware[_-]?virtual[_-]?platform`),
			regexp.MustCompile(`(?i)vmware[_-]?sga`),
		},
		"virtualbox": {
			regexp.MustCompile(`(?i)virtualbox|vbox`),
			regexp.MustCompile(`(?i)vbox[_-]?virtual[_-]?platform`),
			regexp.MustCompile(`(?i)oracle[_-]?virtualbox`),
		},
		"qemu": {
			regexp.MustCompile(`(?i)qemu|kvm|bochs`),
			regexp.MustCompile(`(?i)TCG`),
		},
		"hyperv": {
			regexp.MustCompile(`(?i)hyper[-]?v|microsoft[_-]?virtual`),
			regexp.MustCompile(`(?i)hypervisor`),
		},
		"parallels": {
			regexp.MustCompile(`(?i)parallels`),
			regexp.MustCompile(`(?i)prl[_-]?kernel`),
		},
		"xen": {
			regexp.MustCompile(`(?i)xen`),
			regexp.MustCompile(`(?i)hvm`),
		},
	}

	emulatorPatterns = map[string][]*regexp.Regexp{
		"android_emulator": {
			regexp.MustCompile(`(?i)android[_-]?emulator|genymotion|blue[_-]?stacks`),
			regexp.MustCompile(`(?i)/sdk/gphone`),
			regexp.MustCompile(`(?i)goldfish`),
		},
		"ios_simulator": {
			regexp.MustCompile(`(?i)iphonesimulator|ipadsimulator`),
			regexp.MustCompile(`(?i)corellium`),
		},
		"generic_emulator": {
			regexp.MustCompile(`(?i)emulator|simulator`),
		},
	}

	automationPatterns = map[string][]*regexp.Regexp{
		"selenium": {
			regexp.MustCompile(`(?i)selenium|webdriver`),
			regexp.MustCompile(`(?i)__selenium|__webdriver`),
		},
		"puppeteer": {
			regexp.MustCompile(`(?i)puppeteer`),
			regexp.MustCompile(`(?i)\$cdc_`),
		},
		"playwright": {
			regexp.MustCompile(`(?i)playwright`),
			regexp.MustCompile(`(?i)__playwright|__pw_`),
		},
		"headless_chrome": {
			regexp.MustCompile(`(?i)headless[_-]?chrome`),
		},
		"phantomjs": {
			regexp.MustCompile(`(?i)phantomjs`),
		},
		"appium": {
			regexp.MustCompile(`(?i)appium|uicatalog`),
		},
		"cypress": {
			regexp.MustCompile(`(?i)cypress`),
		},
	}

	debuggerPatterns = map[string][]*regexp.Regexp{
		"chrome_devtools": {
			regexp.MustCompile(`(?i)devtools`),
		},
		"firebug": {
			regexp.MustCompile(`(?i)firebug`),
		},
		"webkit_inspector": {
			regexp.MustCompile(`(?i)webkit[_-]?inspector`),
		},
	}

	vmWebGLPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)vmware|virtualbox|qemu|kvm|hyperv|parallels|xen`),
		regexp.MustCompile(`(?i)swiftshader|llvmpipe|mesa|software`),
		regexp.MustCompile(`(?i)microsoft[_-]?basic[_-]?rendering`),
		regexp.MustCompile(`(?i)google[_-]?inc[_-]?software`),
		regexp.MustCompile(`(?i)unknown|generic|default`),
	}

	vmCPUPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)cpu.*virtual|cpu.*vm`),
	}

	automationProps = []string{
		"webdriver",
		"__webdriver_evaluate",
		"__selenium_evaluate",
		"__webdriver_script_fn",
		"__driver_evaluate",
		"__fxdriver_evaluate",
		"__webdriver_unwrapped",
		"__lastWatirAlert",
		"__$webdriverAsyncExecutor",
		"callSelenium",
		"__selenium",
		"Selenium",
		"$cdc_asdjflasutopfhvcZLmcfl_",
		"$chrome_asyncScriptInfo",
		"__playwright__",
		"__pw_tags",
		"__pw_resume__",
	}
)

func NewEnhancedEnvDetectorService() *EnhancedEnvDetectorService {
	return &EnhancedEnvDetectorService{
		detector:           NewEnhancedEnvDetector(),
		blacklistSvc:       NewBlacklistService(),
		rateLimitSvc:       NewRateLimitService(),
		envCache:           make(map[string]*EnhancedEnvInfo),
		cacheExpiration:    5 * time.Minute,
		detectionThreshold: 0.95,
	}
}

func NewEnhancedEnvDetector() *EnhancedEnvDetector {
	return &EnhancedEnvDetector{}
}

func (d *EnhancedEnvDetector) DetectVM(info *EnhancedEnvInfo) *VMDetectionResult {
	result := &VMDetectionResult{
		Detected:   false,
		VMType:     "none",
		RiskScore:  0,
		Indicators: []string{},
		Confidence: 0,
	}

	uaLower := strings.ToLower(info.UserAgent)
	screenLower := strings.ToLower(info.Platform)

	detectedTypes := make(map[string]int)

	for vmType, patterns := range vmPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(uaLower) || pattern.MatchString(screenLower) {
				detectedTypes[vmType]++
				result.Indicators = append(result.Indicators, "UA: "+vmType)
			}
		}
	}

	if info.WebGLRenderer != "" {
		rendererLower := strings.ToLower(info.WebGLRenderer)
		for _, pattern := range vmWebGLPatterns {
			if pattern.MatchString(rendererLower) {
				result.Indicators = append(result.Indicators, "WebGL: software/virtual renderer")
				detectedTypes["virtual_renderer"]++
				break
			}
		}
	}

	if info.HardwareConcurrency > 0 && info.HardwareConcurrency < 2 {
		result.Indicators = append(result.Indicators, "Low core count (possible VM)")
		detectedTypes["low_cores"]++
	}

	if info.DeviceMemory > 0 && info.DeviceMemory < 1 {
		result.Indicators = append(result.Indicators, "Low device memory (possible VM)")
		detectedTypes["low_memory"]++
	}

	screenScore := info.ScreenWidth*info.ScreenHeight + info.ColorDepth
	if screenScore == 0 || screenScore > 0 && info.PixelRatio == 0 {
		result.Indicators = append(result.Indicators, "Suspicious screen parameters")
		detectedTypes["screen_anomaly"]++
	}

	if len(detectedTypes) > 0 {
		result.Detected = true
		maxCount := 0
		for vmType, count := range detectedTypes {
			if vmType != "virtual_renderer" && vmType != "low_cores" && vmType != "low_memory" && vmType != "screen_anomaly" {
				if count > maxCount {
					maxCount = count
					result.VMType = vmType
				}
			}
		}
		if result.VMType == "" {
			result.VMType = "virtual_renderer"
		}

		result.RiskScore = math.Min(float64(len(result.Indicators))*20+float64(maxCount)*10, 100)
		result.Confidence = math.Min(float64(len(detectedTypes))/float64(len(vmPatterns)), 1.0)
	}

	return result
}

func (d *EnhancedEnvDetector) DetectEmulator(info *EnhancedEnvInfo) *EmulatorDetectionResult {
	result := &EmulatorDetectionResult{
		Detected:      false,
		EmulatorType:  "none",
		RiskScore:     0,
		Indicators:    []string{},
		Confidence:    0,
	}

	uaLower := strings.ToLower(info.UserAgent)

	detectedTypes := make(map[string]int)

	for emuType, patterns := range emulatorPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(uaLower) {
				detectedTypes[emuType]++
				result.Indicators = append(result.Indicators, "UA: "+emuType)
			}
		}
	}

	if info.Platform != "" {
		platformLower := strings.ToLower(info.Platform)
		if strings.Contains(platformLower, "android") && strings.Contains(uaLower, "mobile") {
			result.Indicators = append(result.Indicators, "Android platform detected")
			detectedTypes["mobile_platform"]++
		}
	}

	if info.TouchSupport && info.MaxTouchPoints > 0 {
		if info.MaxTouchPoints == 1 {
			result.Indicators = append(result.Indicators, "Single touch point (possible emulator)")
			detectedTypes["single_touch"]++
		}
	}

	if len(result.Indicators) > 0 && len(detectedTypes) >= 2 {
		result.Detected = true
		for emuType, count := range detectedTypes {
			if count > 0 && emuType != "mobile_platform" && emuType != "single_touch" {
				result.EmulatorType = emuType
				break
			}
		}
		result.RiskScore = math.Min(float64(len(result.Indicators))*15+30, 100)
		result.Confidence = math.Min(float64(len(detectedTypes))/3.0, 1.0)
	}

	return result
}

func (d *EnhancedEnvDetector) DetectDebugState(info *EnhancedEnvInfo) *DebugDetectionResult {
	result := &DebugDetectionResult{
		Detected:      false,
		DebuggerType:  "none",
		RiskScore:     0,
		Indicators:    []string{},
		IsOpen:        false,
		Confidence:    0,
	}

	uaLower := strings.ToLower(info.UserAgent)

	for debuggerType, patterns := range debuggerPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(uaLower) {
				result.Indicators = append(result.Indicators, "UA: "+debuggerType)
				result.DebuggerType = debuggerType
				result.Detected = true
			}
		}
	}

	if strings.Contains(uaLower, "devtools") || strings.Contains(uaLower, "inspect") {
		result.Indicators = append(result.Indicators, "DevTools keyword in UA")
		result.IsOpen = true
		result.DebuggerType = "chrome_devtools"
		result.Detected = true
	}

	if info.WebGLVendor == "Google Inc." && strings.Contains(strings.ToLower(info.WebGLRenderer), "swiftshader") {
		result.Indicators = append(result.Indicators, "Software WebGL renderer (debug mode)")
		result.RiskScore += 30
	}

	if result.Detected {
		result.RiskScore = math.Min(result.RiskScore+float64(len(result.Indicators))*15, 100)
		result.Confidence = math.Min(float64(len(result.Indicators))/3.0, 1.0)
	}

	return result
}

func (d *EnhancedEnvDetector) DetectAutomationEnhanced(info *EnhancedEnvInfo) *EnhancedAutomationResult {
	result := &EnhancedAutomationResult{
		Detected:        false,
		Risks:           []string{},
		AutomationType:  "none",
		Confidence:      0,
		DetectionMethod: "multi_factor",
	}

	uaLower := strings.ToLower(info.UserAgent)

	detectedTypes := make(map[string]int)

	for autoType, patterns := range automationPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(uaLower) {
				detectedTypes[autoType]++
				result.Risks = append(result.Risks, autoType+" detected via UA")
			}
		}
	}

	if info.UserAgent == "" || len(info.UserAgent) < 20 {
		result.Risks = append(result.Risks, "Empty or short UserAgent")
		detectedTypes["no_ua"]++
	}

	if len(info.Languages) == 0 || (len(info.Languages) == 1 && info.Language == "") {
		result.Risks = append(result.Risks, "Abnormal language settings")
		detectedTypes["abnormal_lang"]++
	}

	if info.CanvasFingerprint == "" {
		result.Risks = append(result.Risks, "Canvas fingerprint missing")
		detectedTypes["no_canvas"]++
	}

	if info.WebGLRenderer == "" || info.WebGLVendor == "" {
		result.Risks = append(result.Risks, "WebGL information missing")
		detectedTypes["no_webgl"]++
	}

	if len(info.Fonts) < 3 {
		result.Risks = append(result.Risks, "Few fonts detected")
		detectedTypes["few_fonts"]++
	}

	if info.AudioFingerprint == "" {
		result.Risks = append(result.Risks, "Audio fingerprint missing")
		detectedTypes["no_audio"]++
	}

	if info.Platform == "" {
		result.Risks = append(result.Risks, "Platform information missing")
		detectedTypes["no_platform"]++
	}

	if info.HardwareConcurrency <= 0 {
		result.Risks = append(result.Risks, "Hardware concurrency missing")
		detectedTypes["no_concurrency"]++
	}

	if len(detectedTypes) > 0 {
		result.Detected = true
		result.AutomationType = "unknown"
		for autoType, count := range detectedTypes {
			if autoType != "no_ua" && autoType != "abnormal_lang" && autoType != "no_canvas" &&
				autoType != "no_webgl" && autoType != "few_fonts" && autoType != "no_audio" &&
				autoType != "no_platform" && autoType != "no_concurrency" {
				if count > 0 {
					result.AutomationType = autoType
					break
				}
			}
		}

		suspiciousCount := 0
		for _, key := range []string{"no_ua", "abnormal_lang", "no_canvas", "no_webgl", "few_fonts", "no_audio", "no_platform", "no_concurrency"} {
			if _, ok := detectedTypes[key]; ok {
				suspiciousCount++
			}
		}

		result.Confidence = math.Min(float64(len(detectedTypes))/float64(len(automationPatterns)+8), 1.0)
	}

	return result
}

func (d *EnhancedEnvDetector) CalculateEnhancedEnvScore(info *EnhancedEnvInfo) float64 {
	score := 100.0

	automation := d.DetectAutomationEnhanced(info)
	vm := d.DetectVM(info)
	emulator := d.DetectEmulator(info)
	debug := d.DetectDebugState(info)

	if automation.Detected {
		score -= float64(len(automation.Risks)) * 10
		score -= (1 - automation.Confidence) * 20
	}

	if vm.Detected {
		score -= vm.RiskScore * 0.3
	}

	if emulator.Detected {
		score -= emulator.RiskScore * 0.25
	}

	if debug.Detected {
		score -= debug.RiskScore * 0.2
	}

	if info.CanvasFingerprint == "" {
		score -= 8
	}

	if info.WebGLRenderer == "" || info.WebGLVendor == "" {
		score -= 5
	}

	if len(info.Fonts) < 3 {
		score -= 8
	}

	if info.AudioFingerprint == "" {
		score -= 5
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

func (d *EnhancedEnvDetector) EvaluateEnhancedRisk(info *EnhancedEnvInfo) *EnhancedEnvRiskResult {
	automation := d.DetectAutomationEnhanced(info)
	vm := d.DetectVM(info)
	emulator := d.DetectEmulator(info)
	debug := d.DetectDebugState(info)
	score := d.CalculateEnhancedEnvScore(info)

	riskLevel := "low"
	if score < 50 {
		riskLevel = "critical"
	} else if score < 60 {
		riskLevel = "high"
	} else if score < 75 {
		riskLevel = "medium"
	}

	allRisks := append([]string{}, automation.Risks...)
	if vm.Detected {
		allRisks = append(allRisks, "VM detected: "+vm.VMType)
	}
	if emulator.Detected {
		allRisks = append(allRisks, "Emulator detected: "+emulator.EmulatorType)
	}
	if debug.Detected {
		allRisks = append(allRisks, "Debugger detected: "+debug.DebuggerType)
	}

	return &EnhancedEnvRiskResult{
		RiskLevel:        riskLevel,
		Score:            score,
		Risks:            allRisks,
		Action:           d.determineEnhancedAction(automation, vm, emulator, debug, score),
		AutomationResult:  automation,
		VMResult:          vm,
		EmulatorResult:   emulator,
		DebugResult:       debug,
	}
}

func (d *EnhancedEnvDetector) determineEnhancedAction(automation *EnhancedAutomationResult, vm *VMDetectionResult, emulator *EmulatorDetectionResult, debug *DebugDetectionResult, score float64) string {
	if automation.Detected && automation.Confidence > 0.7 {
		return "block"
	}
	if vm.Detected && vm.Confidence > 0.6 {
		return "block"
	}
	if emulator.Detected && emulator.Confidence > 0.6 {
		return "review"
	}
	if debug.Detected && debug.IsOpen {
		return "review"
	}
	if automation.Detected || vm.Detected || score < 60 {
		return "review"
	}
	if score < 75 {
		return "monitor"
	}
	return "pass"
}

func (d *EnhancedEnvDetector) RunAllEnhancedChecks(info *EnhancedEnvInfo) *EnhancedEnvDetectionReport {
	checks := []EnhancedRiskCheckResult{
		{Name: "selenium", Risk: "high", Category: "automation"},
		{Name: "puppeteer", Risk: "high", Category: "automation"},
		{Name: "playwright", Risk: "high", Category: "automation"},
		{Name: "headless_chrome", Risk: "high", Category: "automation"},
		{Name: "phantomjs", Risk: "high", Category: "automation"},
		{Name: "appium", Risk: "high", Category: "automation"},
		{Name: "cypress", Risk: "high", Category: "automation"},
		{Name: "chrome_devtools", Risk: "medium", Category: "debug"},
		{Name: "firebug", Risk: "medium", Category: "debug"},
		{Name: "vmware", Risk: "high", Category: "vm"},
		{Name: "virtualbox", Risk: "high", Category: "vm"},
		{Name: "qemu", Risk: "high", Category: "vm"},
		{Name: "hyperv", Risk: "high", Category: "vm"},
		{Name: "parallels", Risk: "high", Category: "vm"},
		{Name: "android_emulator", Risk: "medium", Category: "emulator"},
		{Name: "ios_simulator", Risk: "medium", Category: "emulator"},
		{Name: "webgl_software_renderer", Risk: "medium", Category: "vm"},
		{Name: "low_hardware", Risk: "medium", Category: "environment"},
		{Name: "missing_fingerprints", Risk: "medium", Category: "environment"},
		{Name: "abnormal_language", Risk: "low", Category: "environment"},
	}

	uaLower := strings.ToLower(info.UserAgent)

	for i, check := range checks {
		detected := false

		switch check.Name {
		case "selenium":
			detected = regexp.MustCompile(`(?i)selenium|webdriver|__selenium|__webdriver`).MatchString(uaLower)
		case "puppeteer":
			detected = regexp.MustCompile(`(?i)puppeteer|\$cdc_`).MatchString(uaLower)
		case "playwright":
			detected = regexp.MustCompile(`(?i)playwright|__playwright|__pw_`).MatchString(uaLower)
		case "headless_chrome":
			detected = regexp.MustCompile(`(?i)headless`).MatchString(uaLower)
		case "phantomjs":
			detected = regexp.MustCompile(`(?i)phantomjs`).MatchString(uaLower)
		case "appium":
			detected = regexp.MustCompile(`(?i)appium`).MatchString(uaLower)
		case "cypress":
			detected = regexp.MustCompile(`(?i)cypress`).MatchString(uaLower)
		case "chrome_devtools":
			detected = regexp.MustCompile(`(?i)devtools`).MatchString(uaLower)
		case "firebug":
			detected = regexp.MustCompile(`(?i)firebug`).MatchString(uaLower)
		case "vmware":
			detected = regexp.MustCompile(`(?i)vmware`).MatchString(uaLower)
		case "virtualbox":
			detected = regexp.MustCompile(`(?i)virtualbox|vbox`).MatchString(uaLower)
		case "qemu":
			detected = regexp.MustCompile(`(?i)qemu|kvm`).MatchString(uaLower)
		case "hyperv":
			detected = regexp.MustCompile(`(?i)hyper|v`).MatchString(uaLower)
		case "parallels":
			detected = regexp.MustCompile(`(?i)parallels`).MatchString(uaLower)
		case "android_emulator":
			detected = regexp.MustCompile(`(?i)android.*emulator|genymotion|goldfish`).MatchString(uaLower)
		case "ios_simulator":
			detected = regexp.MustCompile(`(?i)iphonesimulator|ipadsimulator`).MatchString(uaLower)
		case "webgl_software_renderer":
			if info.WebGLRenderer != "" {
				detected = regexp.MustCompile(`(?i)swiftshader|llvmpipe|mesa|software|unknown|generic`).MatchString(info.WebGLRenderer)
			}
		case "low_hardware":
			detected = (info.HardwareConcurrency > 0 && info.HardwareConcurrency < 2) ||
				(info.DeviceMemory > 0 && info.DeviceMemory < 1)
		case "missing_fingerprints":
			missing := 0
			if info.CanvasFingerprint == "" {
				missing++
			}
			if info.WebGLRenderer == "" {
				missing++
			}
			if info.AudioFingerprint == "" {
				missing++
			}
			detected = missing >= 2
		case "abnormal_language":
			detected = len(info.Languages) == 0 || (len(info.Languages) == 1 && info.Language == "")
		}

		if detected {
			checks[i].Detected = true
			checks[i].Score = d.getScoreForRisk(check.Risk)
			checks[i].Reason = d.getReasonForCheck(check.Name)
		}
	}

	envScore := d.CalculateEnhancedEnvScore(info)
	riskLevel := "low"
	if envScore < 50 {
		riskLevel = "critical"
	} else if envScore < 60 {
		riskLevel = "high"
	} else if envScore < 75 {
		riskLevel = "medium"
	}

	detectedTools := []string{}
	for _, check := range checks {
		if check.Detected {
			detectedTools = append(detectedTools, check.Name)
		}
	}

	automation := d.DetectAutomationEnhanced(info)
	vm := d.DetectVM(info)
	emulator := d.DetectEmulator(info)
	debug := d.DetectDebugState(info)

	confidence := d.calculateAccuracy(info, automation, vm, emulator, debug)

	return &EnhancedEnvDetectionReport{
		Timestamp:        time.Now().Unix(),
		EnvScore:         envScore,
		IsRisky:          envScore < 75,
		RiskLevel:        riskLevel,
		DetectedTools:    detectedTools,
		Checks:           checks,
		Action:           d.determineEnhancedAction(automation, vm, emulator, debug, envScore),
		AutomationResult: automation,
		VMResult:         vm,
		EmulatorResult:   emulator,
		DebugResult:      debug,
		Confidence:       confidence,
		Accuracy:         confidence * 100,
		Recommendations:  d.generateRecommendations(riskLevel, automation, vm, emulator, debug),
	}
}

func (d *EnhancedEnvDetector) getScoreForRisk(risk string) int {
	switch risk {
	case "high":
		return 40
	case "medium":
		return 20
	case "low":
		return 10
	default:
		return 15
	}
}

func (d *EnhancedEnvDetector) getReasonForCheck(name string) string {
	reasons := map[string]string{
		"selenium":              "Selenium WebDriver特征检测",
		"puppeteer":             "Puppeteer特征检测",
		"playwright":            "Playwright特征检测",
		"headless_chrome":      "无头浏览器特征检测",
		"phantomjs":             "PhantomJS特征检测",
		"appium":                "Appium特征检测",
		"cypress":               "Cypress特征检测",
		"chrome_devtools":       "Chrome DevTools检测",
		"firebug":               "Firebug检测",
		"vmware":                "VMware虚拟机检测",
		"virtualbox":           "VirtualBox虚拟机检测",
		"qemu":                  "QEMU/KVM虚拟机检测",
		"hyperv":                "Hyper-V虚拟机检测",
		"parallels":             "Parallels虚拟机检测",
		"android_emulator":      "Android模拟器检测",
		"ios_simulator":         "iOS模拟器检测",
		"webgl_software_renderer": "WebGL软件渲染器检测",
		"low_hardware":          "低硬件配置检测",
		"missing_fingerprints":  "缺失多个指纹特征",
		"abnormal_language":     "语言设置异常",
	}
	if reason, ok := reasons[name]; ok {
		return reason
	}
	return name + "特征检测"
}

func (d *EnhancedEnvDetector) calculateAccuracy(info *EnhancedEnvInfo, automation *EnhancedAutomationResult, vm *VMDetectionResult, emulator *EmulatorDetectionResult, debug *DebugDetectionResult) float64 {
	factors := 1.0

	if automation.Detected {
		factors *= 0.9 + (automation.Confidence * 0.1)
	}
	if vm.Detected {
		factors *= 0.85 + (vm.Confidence * 0.15)
	}
	if emulator.Detected {
		factors *= 0.9 + (emulator.Confidence * 0.1)
	}
	if debug.Detected {
		factors *= 0.95 + (debug.Confidence * 0.05)
	}

	if info.CanvasFingerprint != "" && info.WebGLRenderer != "" && info.AudioFingerprint != "" {
		factors *= 1.05
	}

	if factors > 1.0 {
		factors = 1.0
	}

	return factors
}

func (d *EnhancedEnvDetector) generateRecommendations(riskLevel string, automation *EnhancedAutomationResult, vm *VMDetectionResult, emulator *EmulatorDetectionResult, debug *DebugDetectionResult) []string {
	recommendations := []string{}

	switch riskLevel {
	case "critical":
		recommendations = append(recommendations, "严重风险，建议立即阻止访问")
	case "high":
		recommendations = append(recommendations, "高风险，建议阻止或要求额外验证")
	case "medium":
		recommendations = append(recommendations, "中风险，建议启用验证码或人工审核")
	case "low":
		recommendations = append(recommendations, "低风险，允许正常访问")
	}

	if automation.Detected && automation.Confidence > 0.5 {
		recommendations = append(recommendations, "检测到自动化工具("+automation.AutomationType+")，建议阻止")
	}

	if vm.Detected {
		recommendations = append(recommendations, "检测到虚拟机("+vm.VMType+")，请确认是否为合法用途")
	}

	if emulator.Detected {
		recommendations = append(recommendations, "检测到模拟器("+emulator.EmulatorType+")，请确认是否为合法用途")
	}

	if debug.Detected && debug.IsOpen {
		recommendations = append(recommendations, "检测到调试工具处于打开状态")
	}

	return recommendations
}

func (s *EnhancedEnvDetectorService) VerifyWithEnhancedEnv(sessionID string, req *EnhancedEnvVerifyRequest) (*EnhancedEnvVerifyResponse, error) {
	envInfo := &EnhancedEnvInfo{
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
		AudioFingerprint:    req.EnvironmentEnv.AudioFingerprint,
		Plugins:             req.EnvironmentEnv.Plugins,
		Fonts:               req.EnvironmentEnv.Fonts,
		TouchSupport:        req.EnvironmentEnv.TouchSupport,
		MaxTouchPoints:      req.EnvironmentEnv.MaxTouchPoints,
		HardwareConcurrency: req.EnvironmentEnv.HardwareConcurrency,
		DeviceMemory:        req.EnvironmentEnv.DeviceMemory,
		Fingerprint:         req.Fingerprint,
		WebRTCIPs:          req.EnvironmentEnv.WebRTCIPs,
		ConnectionType:      req.EnvironmentEnv.ConnectionType,
		Headers:             req.EnvironmentEnv.Headers,
	}

	if envInfo.UserAgent == "" {
		envInfo.UserAgent = req.UserAgent
	}

	blacklisted, reason := s.blacklistSvc.CheckBlacklist(req.IPAddress, "ip")
	if blacklisted {
		return &EnhancedEnvVerifyResponse{
			Success:     false,
			RiskLevel:   "critical",
			RiskScore:   100.0,
			RiskFactors: []string{"IP黑名单: " + reason.Error()},
			Action:      "block",
			Message:     "IP已被列入黑名单",
		}, nil
	}

	if req.Fingerprint != "" {
		blacklisted, reason = s.blacklistSvc.CheckBlacklist(req.Fingerprint, "device_id")
		if blacklisted {
			return &EnhancedEnvVerifyResponse{
				Success:     false,
				RiskLevel:   "critical",
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
		return &EnhancedEnvVerifyResponse{
			Success:     false,
			RiskLevel:   "medium",
			RiskScore:   70.0,
			RiskFactors: []string{"IP请求频率超限"},
			Action:      "review",
			Message:     "请求过于频繁，请稍后再试",
		}, nil
	}

	envRisk := s.detector.EvaluateEnhancedRisk(envInfo)
	detectionReport := s.detector.RunAllEnhancedChecks(envInfo)

	if envRisk.Action == "block" {
		return &EnhancedEnvVerifyResponse{
			Success:         false,
			RiskLevel:       envRisk.RiskLevel,
			RiskScore:       envRisk.Score,
			RiskFactors:     envRisk.Risks,
			Action:          "block",
			Message:         "环境检测异常",
			CaptchaPass:     false,
			DetectionReport: detectionReport,
		}, nil
	}

	captchaPass := true
	if envRisk.Action == "review" || envRisk.Action == "monitor" || envRisk.Score < 70 {
		captchaPass = false
	}

	return &EnhancedEnvVerifyResponse{
		Success:         true,
		RiskLevel:       envRisk.RiskLevel,
		RiskScore:       envRisk.Score,
		RiskFactors:     envRisk.Risks,
		Action:          envRisk.Action,
		Message:         "环境检测完成",
		CaptchaPass:     captchaPass,
		DetectionReport: detectionReport,
	}, nil
}

func (s *EnhancedEnvDetectorService) GetEnhancedDetectionReport(envInfo *EnhancedEnvInfo) *EnhancedEnvDetectionReport {
	return s.detector.RunAllEnhancedChecks(envInfo)
}

func (s *EnhancedEnvDetectorService) CacheEnhancedEnvInfo(sessionID string, envInfo *EnhancedEnvInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.envCache[sessionID] = envInfo
}

func (s *EnhancedEnvDetectorService) GetCachedEnhancedEnvInfo(sessionID string) *EnhancedEnvInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.envCache[sessionID]; ok {
		return info
	}
	return nil
}

func (s *EnhancedEnvDetectorService) CleanupExpiredEnhancedCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for sessionID := range s.envCache {
		if now.Sub(now) > s.cacheExpiration {
			delete(s.envCache, sessionID)
		}
	}
}

func (s *EnhancedEnvDetectorService) DetectVM(info *EnhancedEnvInfo) *VMDetectionResult {
	return s.detector.DetectVM(info)
}

func (s *EnhancedEnvDetectorService) DetectEmulator(info *EnhancedEnvInfo) *EmulatorDetectionResult {
	return s.detector.DetectEmulator(info)
}

func (s *EnhancedEnvDetectorService) DetectDebugState(info *EnhancedEnvInfo) *DebugDetectionResult {
	return s.detector.DetectDebugState(info)
}

func (s *EnhancedEnvDetectorService) DetectAutomationEnhanced(info *EnhancedEnvInfo) *EnhancedAutomationResult {
	return s.detector.DetectAutomationEnhanced(info)
}
