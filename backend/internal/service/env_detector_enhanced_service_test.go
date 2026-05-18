package service_test

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/service"
)

type EnhancedEnvInfo struct {
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
	AudioFingerprint    string
	Fonts               []string
	Plugins             []string
	TouchSupport        bool
	MaxTouchPoints      int
	HardwareConcurrency int
	DeviceMemory        float64
	Fingerprint         string
	WebRTCIPs           []string
	ConnectionType      string
	Headers             map[string]string
}

func newEnhancedEnvDetector() *service.EnhancedEnvDetector {
	return service.NewEnhancedEnvDetector()
}

func toServiceEnhancedEnvInfo(info *EnhancedEnvInfo) *service.EnhancedEnvInfo {
	return &service.EnhancedEnvInfo{
		UserAgent:           info.UserAgent,
		Platform:            info.Platform,
		Language:            info.Language,
		Languages:           info.Languages,
		ScreenWidth:         info.ScreenWidth,
		ScreenHeight:        info.ScreenHeight,
		ColorDepth:          info.ColorDepth,
		PixelRatio:          info.PixelRatio,
		Timezone:            info.Timezone,
		TimezoneOffset:      info.TimezoneOffset,
		CanvasFingerprint:   info.CanvasFingerprint,
		WebGLRenderer:       info.WebGLRenderer,
		WebGLVendor:         info.WebGLVendor,
		AudioFingerprint:    info.AudioFingerprint,
		Fonts:               info.Fonts,
		Plugins:             info.Plugins,
		TouchSupport:        info.TouchSupport,
		MaxTouchPoints:      info.MaxTouchPoints,
		HardwareConcurrency: info.HardwareConcurrency,
		DeviceMemory:        info.DeviceMemory,
		Fingerprint:         info.Fingerprint,
		WebRTCIPs:           info.WebRTCIPs,
		ConnectionType:      info.ConnectionType,
		Headers:             info.Headers,
	}
}

func TestEnhancedEnvDetector_DetectVM(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name            string
		envInfo         *EnhancedEnvInfo
		expectedDetected bool
		expectedVMType  string
	}{
		{
			name: "VMware detected via UserAgent",
			envInfo: &EnhancedEnvInfo{
				UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 VMware",
				WebGLRenderer: "VMware SVGA II Adapter",
			},
			expectedDetected: true,
			expectedVMType:  "vmware",
		},
		{
			name: "VirtualBox detected via UserAgent",
			envInfo: &EnhancedEnvInfo{
				UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) VirtualBox",
				WebGLRenderer: "VirtualBox Graphics Adapter",
			},
			expectedDetected: true,
			expectedVMType:  "virtualbox",
		},
		{
			name: "QEMU/KVM detected via UserAgent",
			envInfo: &EnhancedEnvInfo{
				UserAgent:    "Mozilla/5.0 (X11; Linux x86_64; QEMU KVM)",
				WebGLRenderer: "llvmpipe",
			},
			expectedDetected: true,
			expectedVMType:  "qemu",
		},
		{
			name: "Normal browser - no VM",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
				WebGLRenderer:        "NVIDIA GeForce GTX 1080",
				HardwareConcurrency: 8,
				DeviceMemory:        8.0,
			},
			expectedDetected: false,
			expectedVMType:  "none",
		},
		{
			name: "Low hardware - potential VM",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				HardwareConcurrency: 1,
				DeviceMemory:        0.5,
			},
			expectedDetected: true,
			expectedVMType:  "",
		},
		{
			name: "VMware WebGL renderer detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
				WebGLRenderer: "VMware SVGA 3D",
			},
			expectedDetected: true,
			expectedVMType:  "virtual_renderer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectVM(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.Detected != tt.expectedDetected {
				t.Errorf("DetectVM() detected = %v, expected %v", result.Detected, tt.expectedDetected)
			}
			if tt.expectedVMType != "" && result.VMType != tt.expectedVMType {
				t.Errorf("DetectVM() vmType = %v, expected %v", result.VMType, tt.expectedVMType)
			}
		})
	}
}

func TestEnhancedEnvDetector_DetectEmulator(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name              string
		envInfo           *EnhancedEnvInfo
		expectedDetected  bool
		expectedEmuType   string
	}{
		{
			name: "Android Emulator detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Linux; Android 11; Android SDK built for x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
				Platform:     "Android",
				MaxTouchPoints: 1,
			},
			expectedDetected: true,
			expectedEmuType: "android_emulator",
		},
		{
			name: "Genymotion detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Linux; Android 10; HUAWEI LYO-L21 Build/HUAWEILYO-L21; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/87.0.4280.141 Mobile Safari/537.36 genymotion",
			},
			expectedDetected: true,
			expectedEmuType: "android_emulator",
		},
		{
			name: "iOS Simulator detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 iphonesimulator",
			},
			expectedDetected: true,
			expectedEmuType: "ios_simulator",
		},
		{
			name: "Normal mobile device",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15",
				Platform:     "iPhone",
				MaxTouchPoints: 5,
			},
			expectedDetected: false,
			expectedEmuType: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectEmulator(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.Detected != tt.expectedDetected {
				t.Errorf("DetectEmulator() detected = %v, expected %v", result.Detected, tt.expectedDetected)
			}
		})
	}
}

func TestEnhancedEnvDetector_DetectDebugState(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name            string
		envInfo         *EnhancedEnvInfo
		expectedDetected bool
	}{
		{
			name: "DevTools in UserAgent",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36 devtools",
			},
			expectedDetected: true,
		},
		{
			name: "Software WebGL renderer - debug mode",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
				WebGLVendor:   "Google Inc.",
				WebGLRenderer: "SwiftShader for Chrome",
			},
			expectedDetected: true,
		},
		{
			name: "Normal browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
				WebGLVendor:   "NVIDIA Corporation",
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
			},
			expectedDetected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectDebugState(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.Detected != tt.expectedDetected {
				t.Errorf("DetectDebugState() detected = %v, expected %v", result.Detected, tt.expectedDetected)
			}
		})
	}
}

func TestEnhancedEnvDetector_DetectAutomationEnhanced(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name           string
		envInfo        *EnhancedEnvInfo
		expectedDetected bool
	}{
		{
			name: "Selenium WebDriver detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 webdriver selenium",
			},
			expectedDetected: true,
		},
		{
			name: "Puppeteer detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.0.0 Safari/537.36 puppeteer",
			},
			expectedDetected: true,
		},
		{
			name: "Playwright detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.0.0 Safari/537.36 playwright",
			},
			expectedDetected: true,
		},
		{
			name: "Headless Chrome detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/88.0.4324.96 Safari/537.36",
			},
			expectedDetected: true,
		},
		{
			name: "Empty UserAgent - suspicious",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "",
			},
			expectedDetected: true,
		},
		{
			name: "Missing Canvas fingerprint",
			envInfo: &EnhancedEnvInfo{
				UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				CanvasFingerprint: "",
			},
			expectedDetected: true,
		},
		{
			name: "Normal browser environment",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
				Platform:           "Win32",
				Languages:           []string{"en-US", "zh-CN"},
				Language:            "en-US",
				CanvasFingerprint:   "validhash123",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Fonts:               []string{"Arial", "Helvetica", "Times"},
				AudioFingerprint:    "audiohash123",
				HardwareConcurrency: 8,
			},
			expectedDetected: false,
		},
		{
			name: "Missing multiple fingerprints",
			envInfo: &EnhancedEnvInfo{
				UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				CanvasFingerprint: "",
				WebGLRenderer:    "",
				AudioFingerprint: "",
			},
			expectedDetected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectAutomationEnhanced(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.Detected != tt.expectedDetected {
				t.Errorf("DetectAutomationEnhanced() detected = %v, expected %v, risks = %v",
					result.Detected, tt.expectedDetected, result.Risks)
			}
		})
	}
}

func TestEnhancedEnvDetector_CalculateEnhancedEnvScore(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name        string
		envInfo     *EnhancedEnvInfo
		minExpected float64
		maxExpected float64
	}{
		{
			name: "Normal browser environment - high score",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				Platform:           "Win32",
				Languages:           []string{"en-US"},
				Language:            "en-US",
				CanvasFingerprint:   "validhash123",
				WebGLRenderer:      "NVIDIA GeForce GTX 1080",
				WebGLVendor:        "NVIDIA",
				Fonts:               []string{"Arial", "Helvetica"},
				AudioFingerprint:    "audiohash123",
				HardwareConcurrency: 8,
			},
			minExpected: 75,
			maxExpected: 100,
		},
		{
			name: "Automation detected - low score",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/88.0.4324.96 webdriver",
				Platform:           "",
				Languages:           []string{},
				Language:            "",
				CanvasFingerprint:   "",
				WebGLRenderer:      "",
				WebGLVendor:        "",
				Fonts:               []string{},
				AudioFingerprint:    "",
				HardwareConcurrency: 0,
			},
			minExpected: 0,
			maxExpected: 30,
		},
		{
			name: "VM detected - medium-low score",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 VMware",
				Platform:           "Win32",
				Languages:           []string{"en-US"},
				Language:            "en-US",
				CanvasFingerprint:   "hash123",
				WebGLRenderer:      "VMware SVGA",
				WebGLVendor:        "VMware",
				Fonts:               []string{"Arial"},
				AudioFingerprint:    "audiohash123",
				HardwareConcurrency: 2,
				DeviceMemory:       2.0,
			},
			minExpected: 30,
			maxExpected: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := detector.CalculateEnhancedEnvScore(toServiceEnhancedEnvInfo(tt.envInfo))
			if score < tt.minExpected || score > tt.maxExpected {
				t.Errorf("CalculateEnhancedEnvScore() = %v, expected between %v and %v",
					score, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestEnhancedEnvDetector_EvaluateEnhancedRisk(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name       string
		envInfo    *EnhancedEnvInfo
		riskLevel  string
		hasAction  bool
	}{
		{
			name: "Critical risk - automation with high confidence",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/88.0.4324.96 webdriver selenium",
			},
			riskLevel: "critical",
			hasAction: true,
		},
		{
			name: "High risk - VM detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 VMware VirtualBox",
				WebGLRenderer: "VMware SVGA",
			},
			riskLevel: "critical",
			hasAction: true,
		},
		{
			name: "Medium risk - some issues",
			envInfo: &EnhancedEnvInfo{
				UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				CanvasFingerprint: "",
				WebGLRenderer:    "",
				WebGLVendor:      "",
				Fonts:             []string{"Arial"},
			},
			riskLevel: "medium",
			hasAction: true,
		},
		{
			name: "Low risk - normal browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
				Platform:           "Win32",
				Languages:           []string{"en-US"},
				Language:            "en-US",
				CanvasFingerprint:   "validhash123",
				WebGLRenderer:      "NVIDIA GeForce GTX 1080",
				WebGLVendor:        "NVIDIA",
				Fonts:               []string{"Arial", "Helvetica"},
				AudioFingerprint:    "audiohash123",
				HardwareConcurrency: 8,
			},
			riskLevel: "low",
			hasAction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EvaluateEnhancedRisk(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.RiskLevel != tt.riskLevel {
				t.Errorf("EvaluateEnhancedRisk() riskLevel = %v, expected %v", result.RiskLevel, tt.riskLevel)
			}
			if tt.hasAction && result.Action == "" {
				t.Error("EvaluateEnhancedRisk() action is empty")
			}
		})
	}
}

func TestEnhancedEnvDetector_RunAllEnhancedChecks(t *testing.T) {
	detector := newEnhancedEnvDetector()

	envInfo := &EnhancedEnvInfo{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
	}

	report := detector.RunAllEnhancedChecks(toServiceEnhancedEnvInfo(envInfo))

	if report == nil {
		t.Fatal("RunAllEnhancedChecks() returned nil")
	}

	if report.EnvScore < 0 || report.EnvScore > 100 {
		t.Errorf("RunAllEnhancedChecks() EnvScore = %v, expected between 0 and 100", report.EnvScore)
	}

	validRiskLevels := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if !validRiskLevels[report.RiskLevel] {
		t.Errorf("RunAllEnhancedChecks() RiskLevel = %v, expected low/medium/high/critical", report.RiskLevel)
	}

	validActions := map[string]bool{"pass": true, "review": true, "block": true, "monitor": true}
	if !validActions[report.Action] {
		t.Errorf("RunAllEnhancedChecks() Action = %v, expected pass/review/block/monitor", report.Action)
	}

	if len(report.Checks) == 0 {
		t.Error("RunAllEnhancedChecks() returned no checks")
	}

	if report.Accuracy <= 0 || report.Accuracy > 100 {
		t.Errorf("RunAllEnhancedChecks() Accuracy = %v, expected between 0 and 100", report.Accuracy)
	}
}

func TestEnhancedEnvDetector_ReportGeneration(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name          string
		envInfo       *EnhancedEnvInfo
		expectVM      bool
		expectEmu     bool
		expectDebug   bool
		expectAuto    bool
	}{
		{
			name: "Full automation environment",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/88.0.4324.96 webdriver selenium puppeteer",
				CanvasFingerprint:   "",
				WebGLRenderer:      "",
				AudioFingerprint:    "",
				Fonts:               []string{},
				HardwareConcurrency: 0,
			},
			expectVM:    false,
			expectEmu:   false,
			expectDebug: false,
			expectAuto:  true,
		},
		{
			name: "VM with automation",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/88.0.0.0 VMware VirtualBox webdriver",
				WebGLRenderer:      "VMware SVGA 3D",
				CanvasFingerprint:   "",
				AudioFingerprint:    "",
				HardwareConcurrency: 1,
			},
			expectVM:    true,
			expectEmu:   false,
			expectDebug: false,
			expectAuto:  true,
		},
		{
			name: "Clean environment",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
				Platform:           "Win32",
				Languages:           []string{"en-US"},
				Language:            "en-US",
				CanvasFingerprint:   "validcanvashash123",
				WebGLRenderer:      "NVIDIA GeForce GTX 1080",
				WebGLVendor:        "NVIDIA",
				AudioFingerprint:    "validaudiohash123",
				Fonts:               []string{"Arial", "Helvetica", "Times"},
				HardwareConcurrency: 8,
				DeviceMemory:       16.0,
			},
			expectVM:    false,
			expectEmu:   false,
			expectDebug: false,
			expectAuto:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := detector.RunAllEnhancedChecks(toServiceEnhancedEnvInfo(tt.envInfo))

			if tt.expectVM && (report.VMResult == nil || !report.VMResult.Detected) {
				t.Errorf("RunAllEnhancedChecks() expected VM detection, got nil or not detected")
			}
			if tt.expectEmu && (report.EmulatorResult == nil || !report.EmulatorResult.Detected) {
				t.Errorf("RunAllEnhancedChecks() expected Emulator detection, got nil or not detected")
			}
			if tt.expectDebug && (report.DebugResult == nil || !report.DebugResult.Detected) {
				t.Errorf("RunAllEnhancedChecks() expected Debug detection, got nil or not detected")
			}
			if tt.expectAuto && (report.AutomationResult == nil || !report.AutomationResult.Detected) {
				t.Errorf("RunAllEnhancedChecks() expected Automation detection, got nil or not detected")
			}
		})
	}
}

func TestEnhancedEnvInfo_Model(t *testing.T) {
	envInfo := &EnhancedEnvInfo{
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
		AudioFingerprint:   "audiohash123",
		Fonts:               []string{"Arial", "Helvetica"},
		Plugins:             []string{"Chrome PDF Plugin"},
		TouchSupport:        true,
		MaxTouchPoints:      10,
		HardwareConcurrency: 8,
		DeviceMemory:        8.0,
		Fingerprint:         "devicefingerprint123",
		WebRTCIPs:           []string{"192.168.1.100"},
		ConnectionType:      "wifi",
	}

	if envInfo.UserAgent == "" {
		t.Error("EnhancedEnvInfo.UserAgent should not be empty")
	}
	if envInfo.Platform == "" {
		t.Error("EnhancedEnvInfo.Platform should not be empty")
	}
	if envInfo.ScreenWidth == 0 {
		t.Error("EnhancedEnvInfo.ScreenWidth should not be zero")
	}
}

func TestEnhancedEnvDetector_Recommendations(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name                  string
		envInfo               *EnhancedEnvInfo
		expectRecommendation  bool
	}{
		{
			name: "High risk - should have block recommendation",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/88.0.0.0 webdriver",
			},
			expectRecommendation: true,
		},
		{
			name: "Clean environment",
			envInfo: &EnhancedEnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
				Platform:           "Win32",
				Languages:           []string{"en-US"},
				Language:            "en-US",
				CanvasFingerprint:   "validhash123",
				WebGLRenderer:      "NVIDIA GeForce GTX 1080",
				WebGLVendor:        "NVIDIA",
				AudioFingerprint:    "audiohash123",
				Fonts:               []string{"Arial", "Helvetica"},
				HardwareConcurrency: 8,
			},
			expectRecommendation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := detector.RunAllEnhancedChecks(toServiceEnhancedEnvInfo(tt.envInfo))
			if tt.expectRecommendation && len(report.Recommendations) == 0 {
				t.Error("RunAllEnhancedChecks() should have recommendations")
			}
		})
	}
}

func TestEnhancedEnvDetector_WebGLPatterns(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name            string
		envInfo         *EnhancedEnvInfo
		expectVM        bool
	}{
		{
			name: "SwiftShader renderer - software rendering",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
				WebGLRenderer: "SwiftShader 4.0.0",
			},
			expectVM: true,
		},
		{
			name: "LLVMpipe renderer - software rendering",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
				WebGLRenderer: "llvmpipe",
			},
			expectVM: true,
		},
		{
			name: "Mesa software renderer",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
				WebGLRenderer: "Mesa DRI Intel",
			},
			expectVM: true,
		},
		{
			name: "Real GPU renderer",
			envInfo: &EnhancedEnvInfo{
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
			},
			expectVM: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectVM(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.Detected != tt.expectVM {
				t.Errorf("DetectVM() detected = %v, expected %v", result.Detected, tt.expectVM)
			}
		})
	}
}

func TestEnhancedEnvDetector_EmulatorPatterns(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name            string
		envInfo         *EnhancedEnvInfo
		expectEmu       bool
	}{
		{
			name: "BlueStacks emulator",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Linux; Android 9; Redmi 6 Pro Build/PKQ1.190101.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/75.0.3770.143 Mobile Safari/537.36 blue stacks",
			},
			expectEmu: true,
		},
		{
			name: "Nox emulator",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Linux; Android 7.1.2; Smart 5.5 Build/NRD90M; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/79.0.3945.136 Mobile Safari/537.36 nox",
			},
			expectEmu: true,
		},
		{
			name: "Real Android device",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Linux; Android 10; SM-G975F Build/QP1A.190711.020; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/87.0.4280.141 Mobile Safari/537.36",
			},
			expectEmu: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectEmulator(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.Detected != tt.expectEmu {
				t.Errorf("DetectEmulator() detected = %v, expected %v", result.Detected, tt.expectEmu)
			}
		})
	}
}

func TestEnhancedEnvDetector_DebuggerPatterns(t *testing.T) {
	detector := newEnhancedEnvDetector()

	tests := []struct {
		name            string
		envInfo         *EnhancedEnvInfo
		expectDebug     bool
		expectIsOpen    bool
	}{
		{
			name: "DevTools keyword in UA",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 devtools",
			},
			expectDebug:  true,
			expectIsOpen: true,
		},
		{
			name: "Firebug detected",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 firebug",
			},
			expectDebug:  true,
			expectIsOpen: false,
		},
		{
			name: "Clean browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
			},
			expectDebug:  false,
			expectIsOpen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectDebugState(toServiceEnhancedEnvInfo(tt.envInfo))
			if result.Detected != tt.expectDebug {
				t.Errorf("DetectDebugState() detected = %v, expected %v", result.Detected, tt.expectDebug)
			}
			if result.IsOpen != tt.expectIsOpen {
				t.Errorf("DetectDebugState() isOpen = %v, expected %v", result.IsOpen, tt.expectIsOpen)
			}
		})
	}
}
