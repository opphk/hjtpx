package service_test

import (
	"testing"
)

func TestCanvasSimilarityEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name     string
		hash1    string
		hash2    string
		expected float64
	}{
		{"identical_hashes", "abc123def456", "abc123def456", 100.0},
		{"empty_hash1", "", "abc123", 0.0},
		{"empty_hash2", "abc123", "", 0.0},
		{"both_empty", "", "", 0.0},
		{"partial_similarity", "abc123", "abc456", 50.0},
		{"no_similarity", "abc123", "xyz789", 0.0},
		{"different_length", "abc123def", "abc123", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.CalculateCanvasSimilarity(tt.hash1, tt.hash2)
			if result != tt.expected {
				t.Errorf("CalculateCanvasSimilarity(%s, %s) = %v, want %v",
					tt.hash1, tt.hash2, result, tt.expected)
			}
		})
	}
}

func TestDetectCanvasAnomaliesEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name              string
		canvasFingerprint string
		expectAnomalies   bool
	}{
		{"normal_fingerprint", "a1b2c3d4e5f678901234567890123456", false},
		{"empty_fingerprint", "", false},
		{"too_short", "abc123", true},
		{"too_long", "a1b2c3d4e5f678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", true},
		{"non_hex_characters", "g1h2i3j4k5l6", true},
		{"all_same_character", "aaaaaaaaaaaaaaaaaaaa", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &EnvInfo{
				CanvasFingerprint: tt.canvasFingerprint,
			}
			anomalies := detector.DetectCanvasAnomalies(info)
			hasAnomalies := len(anomalies) > 0
			if hasAnomalies != tt.expectAnomalies {
				t.Errorf("DetectCanvasAnomalies() for %s = %v, want %v, anomalies: %v",
					tt.name, hasAnomalies, tt.expectAnomalies, anomalies)
			}
		})
	}
}

func TestAnalyzeWebGLDetailsEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name         string
		renderer     string
		vendor       string
		expectedRisk string
	}{
		{"normal_webgl", "NVIDIA GeForce GTX 1080", "NVIDIA", "low"},
		{"empty_webgl", "", "", "high"},
		{"software_renderer_swiftshader", "SwiftShader", "Google", "medium"},
		{"software_renderer_llvmpipe", "llvmpipe", "Mesa", "medium"},
		{"virtual_renderer", "Virtual GPU", "VMware", "medium"},
		{"anonymized_generic", "Generic GPU", "Unknown Vendor", "medium"},
		{"unusual_pattern_headless", "Headless Renderer", "Test", "high"},
		{"unusual_pattern_bot", "Bot Automation", "Test", "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &EnvInfo{
				WebGLRenderer: tt.renderer,
				WebGLVendor:   tt.vendor,
			}
			analysis := detector.AnalyzeWebGLDetails(info)
			risk := analysis["risk"].(string)
			if risk != tt.expectedRisk {
				t.Errorf("AnalyzeWebGLDetails() risk = %v, want %v", risk, tt.expectedRisk)
			}
		})
	}
}

func TestDetectEmulatorIndicatorsEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name                 string
		userAgent            string
		maxTouchPoints       int
		hardwareConcurrency  int
		expectDetected       bool
	}{
		{"normal_browser", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/90.0.0.0 Safari/537.36", 0, 8, false},
		{"android_emulator_genymotion", "Mozilla/5.0 (Linux; Android 11; sdk_phone_x86_64 Build/RP1A.201005.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/90.0.4430.91 Mobile Safari/537.36", 0, 4, true},
		{"android_emulator_bluestacks", "Mozilla/5.0 (Linux; Android 9; BLU G5 Plus Build/P009; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/90.0.4430.91 Mobile Safari/537.36", 0, 8, true},
		{"android_sdk", "Mozilla/5.0 (Linux; Android 10; Android SDK built for x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36", 0, 4, true},
		{"android_nox", "Mozilla/5.0 (Linux; Android 9; Nox) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36", 0, 4, true},
		{"android_normal", "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.91 Mobile Safari/537.36", 5, 8, false},
		{"phantomjs", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/538.1 (KHTML, like Gecko) PhantomJS/2.1.1 Safari/538.1", 0, 4, true},
		{"android_abnormal_touch", "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36", 15, 8, true},
		{"android_abnormal_cpu", "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36", 5, 32, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &EnvInfo{
				UserAgent:           tt.userAgent,
				MaxTouchPoints:      tt.maxTouchPoints,
				HardwareConcurrency: tt.hardwareConcurrency,
			}
			detected, indicators := detector.DetectEmulatorIndicators(info)
			if detected != tt.expectDetected {
				t.Errorf("DetectEmulatorIndicators() for %s = %v, want %v, indicators: %v",
					tt.name, detected, tt.expectDetected, indicators)
			}
		})
	}
}

func TestCalculateProxyRiskScoreEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name     string
		ip       string
		headers  map[string]string
		minScore float64
	}{
		{"no_headers", "192.168.1.1", map[string]string{}, 0.0},
		{"xff_header", "203.0.113.1", map[string]string{"X-Forwarded-For": "192.168.1.1"}, 25.0},
		{"multi_hop_proxy", "203.0.113.1", map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1, 172.16.0.1"}, 40.0},
		{"xri_mismatch", "203.0.113.1", map[string]string{"X-Real-IP": "192.168.1.1"}, 15.0},
		{"via_squid", "203.0.113.1", map[string]string{"Via": "1.1 squid.proxy.com"}, 20.0},
		{"via_nginx", "203.0.113.1", map[string]string{"Via": "nginx/1.18.0"}, 20.0},
		{"proxy_chain", "203.0.113.1", map[string]string{"X-ProxyChain": "http://proxy1.com"}, 30.0},
		{"cdn_original_ip", "203.0.113.1", map[string]string{"X-CDN-Original-IP": "192.168.1.1"}, 20.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := detector.CalculateProxyRiskScore(tt.ip, tt.headers)
			if score < tt.minScore {
				t.Errorf("CalculateProxyRiskScore() for %s = %v, want >= %v",
					tt.name, score, tt.minScore)
			}
		})
	}
}

func TestDetectVPNPatternsEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name          string
		webglVendor   string
		webglRenderer string
		headers       map[string]string
		expectVPN     bool
		minConfidence float64
	}{
		{"no_vpn", "", "", map[string]string{}, false, 0.0},
		{"vpn_header", "", "", map[string]string{"X-VPN-Connection": "true"}, true, 0.95},
		{"vpn_type_header", "", "", map[string]string{"X-VPN-Type": "OpenVPN"}, true, 0.95},
		{"webgl_vmware_vendor", "VMware, Inc.", "", map[string]string{}, true, 0.70},
		{"webgl_virtualbox_vendor", "VirtualBox", "", map[string]string{}, true, 0.70},
		{"webgl_vmware_renderer", "", "VMware SVGA 3D", map[string]string{}, true, 0.75},
		{"webgl_virtualbox_renderer", "", "VirtualBox Graphics Adapter", map[string]string{}, true, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &EnvInfo{
				WebGLVendor:   tt.webglVendor,
				WebGLRenderer: tt.webglRenderer,
			}
			isVPN, confidence, evidence := detector.DetectVPNPatterns(info, tt.headers)
			if isVPN != tt.expectVPN {
				t.Errorf("DetectVPNPatterns() for %s isVPN = %v, want %v, evidence: %v",
					tt.name, isVPN, tt.expectVPN, evidence)
			}
			if isVPN && confidence < tt.minConfidence {
				t.Errorf("DetectVPNPatterns() for %s confidence = %v, want >= %v",
					tt.name, confidence, tt.minConfidence)
			}
		})
	}
}

func TestEnhancedEnvCheckComprehensive(t *testing.T) {
	detector := newTestEnvDetector()

	info := &EnvInfo{
		UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/90.0.0.0 Safari/537.36",
		Platform:            "Win32",
		Language:            "en-US",
		Languages:           []string{"en-US"},
		ScreenWidth:         1920,
		ScreenHeight:        1080,
		ColorDepth:          24,
		PixelRatio:          1.0,
		Timezone:            "America/New_York",
		TimezoneOffset:      -300,
		CanvasFingerprint:   "a1b2c3d4e5f678901234567890123456",
		WebGLRenderer:       "Intel Iris OpenGL Engine",
		WebGLVendor:         "Intel Inc.",
		Plugins:             []string{"Chrome PDF Plugin", "Chrome PDF Viewer"},
		Fonts:               []string{"Arial", "Helvetica", "Times New Roman", "Verdana"},
		TouchSupport:        false,
		MaxTouchPoints:      0,
		HardwareConcurrency: 8,
		Fingerprint:         "comprehensive_test_fingerprint",
	}

	report := detector.EnhancedEnvCheck(info)

	if report == nil {
		t.Fatal("EnhancedEnvCheck returned nil")
	}

	if report.EnvScore < 70 {
		t.Errorf("Expected high env score for comprehensive valid environment, got %f", report.EnvScore)
	}

	if report.RiskLevel != "low" {
		t.Errorf("Expected low risk level for valid environment, got %s", report.RiskLevel)
	}

	if len(report.Checks) < 10 {
		t.Errorf("Expected at least 10 checks, got %d", len(report.Checks))
	}

	for _, check := range report.Checks {
		if check.Detected && check.Score < 0 {
			t.Errorf("Check %s has negative score: %d", check.Name, check.Score)
		}
	}
}

func TestVPNAndProxyCombinationEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	info := &EnvInfo{
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/90.0",
		WebGLVendor:    "VirtualBox",
		WebGLRenderer: "VirtualBox Graphics Adapter",
	}

	headers := map[string]string{
		"X-Forwarded-For":  "192.168.1.1, 10.0.0.1",
		"X-VPN-Connection": "true",
	}

	proxyRisk := detector.CalculateProxyRiskScore("203.0.113.1", headers)
	isVPN, vpnConfidence, vpnEvidence := detector.DetectVPNPatterns(info, headers)

	if proxyRisk < 40 {
		t.Errorf("Expected high proxy risk score, got %f", proxyRisk)
	}

	if !isVPN {
		t.Error("Expected VPN to be detected")
	}

	if vpnConfidence < 0.9 {
		t.Errorf("Expected high VPN confidence, got %f", vpnConfidence)
	}

	if len(vpnEvidence) == 0 {
		t.Error("Expected VPN evidence to be present")
	}
}

func TestMultipleEmulatorPatternsEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	emulators := []string{
		"genymotion",
		"bluestacks",
		"nox",
		"memu",
		"koplayer",
		"droid4x",
		"mumu",
		"phoenix",
		"smartgaga",
	}

	detectedCount := 0
	for _, emulator := range emulators {
		info := &EnvInfo{
			UserAgent: "Mozilla/5.0 (Linux; Android 11; " + emulator + ") Chrome/90.0",
		}
		detected, _ := detector.DetectEmulatorIndicators(info)
		if detected {
			detectedCount++
		}
	}

	if detectedCount != len(emulators) {
		t.Errorf("Expected all emulators to be detected, got %d/%d", detectedCount, len(emulators))
	}
}

func TestProxyRiskScoreCappingEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	headers := map[string]string{
		"X-Forwarded-For":    "192.168.1.1, 10.0.0.1, 172.16.0.1",
		"X-Real-IP":          "192.168.1.1",
		"Via":                "1.1 squid.proxy.com",
		"X-ProxyChain":       "http://proxy1.com",
		"X-CDN-Original-IP":  "192.168.1.1",
	}

	score := detector.CalculateProxyRiskScore("203.0.113.1", headers)

	if score > 100.0 {
		t.Errorf("Proxy risk score should be capped at 100, got %f", score)
	}

	if score < 50.0 {
		t.Errorf("Expected high proxy risk score with multiple indicators, got %f", score)
	}
}

func TestWebGLAnonymizationEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name     string
		renderer string
		vendor   string
		expected bool
	}{
		{"all_generic", "Generic GPU", "Generic Vendor", true},
		{"all_unknown", "Unknown", "Unknown", true},
		{"mixed_generic_unknown", "Generic GPU", "Unknown", true},
		{"one_generic", "NVIDIA GTX 1080", "Generic Vendor", false},
		{"real_gpu", "NVIDIA GeForce RTX 3080", "NVIDIA", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &EnvInfo{
				WebGLRenderer: tt.renderer,
				WebGLVendor:   tt.vendor,
			}
			analysis := detector.AnalyzeWebGLDetails(info)
			isAnonymized, exists := analysis["anonymized"].(bool)
			if exists && isAnonymized != tt.expected {
				t.Errorf("AnalyzeWebGLDetails() anonymized = %v, want %v", isAnonymized, tt.expected)
			}
		})
	}
}

func TestEmulatorAndroidBuildTagsEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name      string
		userAgent string
		expectEmu bool
	}{
		{"normal_android", "Mozilla/5.0 (Linux; Android 11; SM-G991B Build/RP1A.201005.001) AppleWebKit/537.36", false},
		{"emulator_android", "Mozilla/5.0 (Linux; Android 11; sdk_phone_x86_64 Build/RP1A.201005.001; wv) AppleWebKit/537.36", true},
		{"vbox_android", "Mozilla/5.0 (Linux; Android 9; Android SDK built for x86_64 Build/RP1A.201005.001) AppleWebKit/537.36", true},
		{"emulator_build_tag", "Mozilla/5.0 (Linux; Android 10; Android SDK built for x86 Build/MMB29K) AppleWebKit/537.36", true},
		{"test_build_tag", "Mozilla/5.0 (Linux; Android 11; sdk_phone_x86_64 Build/RP1A.test.001) AppleWebKit/537.36", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &EnvInfo{
				UserAgent: tt.userAgent,
			}
			detected, _ := detector.DetectEmulatorIndicators(info)
			if detected != tt.expectEmu {
				t.Errorf("DetectEmulatorIndicators() for %s = %v, want %v",
					tt.name, detected, tt.expectEmu)
			}
		})
	}
}

func TestCanvasFingerprintValidationEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	tests := []struct {
		name        string
		fingerprint string
		expectValid bool
	}{
		{"valid_sha256", "a1b2c3d4e5f6789012345678901234567890abcd", true},
		{"valid_md5", "a1b2c3d4e5f678901234567890123456", true},
		{"empty", "", true},
		{"too_short_hex", "abc123", false},
		{"with_non_hex", "a1b2c3d4e5f6xyz789012345678901234567890", false},
		{"all_repeating", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &EnvInfo{
				CanvasFingerprint: tt.fingerprint,
			}
			anomalies := detector.DetectCanvasAnomalies(info)
			hasAnomalies := len(anomalies) > 0
			if hasAnomalies == tt.expectValid {
				t.Errorf("DetectCanvasAnomalies() for %s: expected anomalies=%v, got anomalies=%v (anomalies: %v)",
					tt.name, !tt.expectValid, hasAnomalies, anomalies)
			}
		})
	}
}

func TestRiskLevelsEnhanced(t *testing.T) {
	detector := newTestEnvDetector()

	info := &EnvInfo{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/90.0",
	}

	report := detector.EnhancedEnvCheck(info)

	validRiskLevels := map[string]bool{
		"low":      true,
		"medium":   true,
		"high":     true,
		"critical": true,
	}

	if !validRiskLevels[report.RiskLevel] {
		t.Errorf("Invalid risk level: %s", report.RiskLevel)
	}

	if report.EnvScore < 0 || report.EnvScore > 100 {
		t.Errorf("EnvScore out of range: %f", report.EnvScore)
	}
}
