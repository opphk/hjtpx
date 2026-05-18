package service

import (
	"testing"
)

func TestDetectAutomationIndicators(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name           string
		info           *EnvInfo
		expectedMinConfidence float64
		expectedTool   string
	}{
		{
			name: "Puppeteer UserAgent",
			info: &EnvInfo{
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 HeadlessChrome/91.0.4472.0 Safari/537.36",
				WebGLRenderer: "",
				HardwareConcurrency: 2,
				Plugins: []string{},
				Fonts: []string{},
			},
			expectedMinConfidence: 0.3,
			expectedTool: "selenium",
		},
		{
			name: "Playwright UserAgent",
			info: &EnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 Safari/537.36 Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
				WebGLRenderer: "SwiftShader",
				HardwareConcurrency: 1,
				Plugins: []string{},
				Fonts: []string{"Arial"},
			},
			expectedMinConfidence: 0.3,
			expectedTool: "puppeteer",
		},
		{
			name: "Selenium WebDriver",
			info: &EnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Gecko/20100101 Firefox/89.0 google量为0 selenium webdriver",
				WebGLRenderer: "",
				HardwareConcurrency: 2,
				Plugins: []string{},
				Fonts: []string{},
			},
			expectedMinConfidence: 0.2,
			expectedTool: "selenium",
		},
		{
			name: "Normal Browser",
			info: &EnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 Safari/537.36",
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
				HardwareConcurrency: 8,
				Plugins: []string{"Chrome PDF Plugin", "Chrome PDF Viewer"},
				Fonts: []string{"Arial", "Times New Roman", "Helvetica"},
			},
			expectedMinConfidence: 0.0,
			expectedTool: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			indicators := detector.detectAutomationIndicators(tc.info)

			if tc.expectedMinConfidence > 0 {
				if len(indicators) == 0 {
					t.Errorf("Expected at least one indicator for %s", tc.name)
				} else {
					found := false
					for _, ind := range indicators {
						if ind.Confidence >= tc.expectedMinConfidence {
							found = true
							if tc.expectedTool != "" && ind.Name == tc.expectedTool {
								t.Logf("Found expected tool %s with confidence %.2f", tc.expectedTool, ind.Confidence)
							}
						}
					}
					if !found {
						t.Logf("Indicators found: %v", indicators)
					}
				}
			}
		})
	}
}

func TestDetectAutomationFromHeaders(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name     string
		headers  map[string]string
		expected int
	}{
		{
			name: "Selenium Header",
			headers: map[string]string{
				"X-WD-Agent": "webdriver",
				"User-Agent": "Mozilla/5.0",
			},
			expected: 1,
		},
		{
			name: "Puppeteer Header",
			headers: map[string]string{
				"X-PUPPETEER": "true",
				"User-Agent": "Mozilla/5.0",
			},
			expected: 1,
		},
		{
			name: "Playwright Header",
			headers: map[string]string{
				"X-PLAYWRIGHT": "enabled",
				"User-Agent": "Mozilla/5.0",
			},
			expected: 1,
		},
		{
			name: "Bot Header",
			headers: map[string]string{
				"X-BOT": "true",
				"User-Agent": "Mozilla/5.0",
			},
			expected: 1,
		},
		{
			name:     "No Headers",
			headers:  map[string]string{},
			expected: 0,
		},
		{
			name: "Automation in Generic Header",
			headers: map[string]string{
				"X-Automation-Type": "script",
				"User-Agent":         "Mozilla/5.0",
			},
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			indicators := detector.DetectAutomationFromHeaders(tc.headers)

			if len(indicators) < tc.expected {
				t.Errorf("Expected at least %d indicators, got %d", tc.expected, len(indicators))
			}

			for _, ind := range indicators {
				if ind.Confidence <= 0 {
					t.Errorf("Indicator confidence should be > 0, got %.2f", ind.Confidence)
				}
				if len(ind.Evidence) == 0 {
					t.Errorf("Indicator should have evidence")
				}
			}
		})
	}
}

func TestEnhancedAutomationDetection(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name                string
		info                *EnvInfo
		frontendDetections  []string
		expectDetected      bool
	}{
		{
			name: "Automation Behavior",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
				WebGLRenderer:       "SwiftShader",
				HardwareConcurrency: 2,
				Plugins:             []string{},
				Fonts:               []string{},
			},
			frontendDetections: []string{"timing_uniform", "click_pattern_robotic"},
			expectDetected:      true,
		},
		{
			name: "Resource Patterns",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 Chrome/91.0",
				WebGLRenderer:       "",
				HardwareConcurrency: 1,
				Plugins:             []string{},
				Fonts:               []string{},
			},
			frontendDetections: []string{"no_images", "minimal_requests"},
			expectDetected:      true,
		},
		{
			name: "Normal Behavior",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0 Safari/537.36",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				HardwareConcurrency: 8,
				Plugins:             []string{"Chrome PDF Plugin"},
				Fonts:               []string{"Arial", "Helvetica"},
			},
			frontendDetections: []string{"normal_human_behavior"},
			expectDetected:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detected, confidence, evidence := detector.EnhancedAutomationDetection(tc.info, tc.frontendDetections)

			if detected != tc.expectDetected {
				t.Errorf("Expected detected=%v, got detected=%v (confidence: %.2f)", tc.expectDetected, detected, confidence)
			}

			if detected && confidence <= 0 {
				t.Errorf("Detected should have confidence > 0, got %.2f", confidence)
			}

			if detected && len(evidence) == 0 {
				t.Errorf("Detected should have evidence")
			}

			t.Logf("Test case '%s': detected=%v, confidence=%.2f, evidence count=%d", tc.name, detected, confidence, len(evidence))
		})
	}
}

func TestAnalyzeBrowserFingerprint(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name             string
		info             *EnvInfo
		expectSuspicious bool
		expectFeatures   int
	}{
		{
			name: "Normal Browser",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 Safari/537.36",
				CanvasFingerprint:   "abc123def456",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Plugins:             []string{"Chrome PDF Plugin", "Chrome PDF Viewer"},
				Fonts:               []string{"Arial", "Helvetica", "Times New Roman"},
				Languages:           []string{"en-US", "en"},
				Language:            "en-US",
			},
			expectSuspicious: false,
			expectFeatures:   0,
		},
		{
			name: "Missing Canvas",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
				CanvasFingerprint:   "",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Plugins:             []string{"Chrome PDF Plugin"},
				Fonts:               []string{"Arial", "Helvetica"},
				Languages:           []string{"en-US"},
				Language:            "en-US",
			},
			expectSuspicious: true,
			expectFeatures:   1,
		},
		{
			name: "Automation UA",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.0 Safari/537.36",
				CanvasFingerprint:   "abc123",
				WebGLRenderer:       "SwiftShader",
				WebGLVendor:         "Google Inc.",
				Plugins:             []string{},
				Fonts:               []string{},
				Languages:           []string{},
				Language:            "",
			},
			expectSuspicious: true,
			expectFeatures:   6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := detector.AnalyzeBrowserFingerprint(tc.info)

			if analysis.IsSuspicious != tc.expectSuspicious {
				t.Errorf("Expected suspicious=%v, got suspicious=%v", tc.expectSuspicious, analysis.IsSuspicious)
			}

			if len(analysis.SuspiciousFeatures) < tc.expectFeatures {
				t.Errorf("Expected at least %d suspicious features, got %d: %v", tc.expectFeatures, len(analysis.SuspiciousFeatures), analysis.SuspiciousFeatures)
			}

			if analysis.Browser == "" {
				t.Errorf("Browser should be parsed")
			}

			t.Logf("Test case '%s': browser=%s, version=%s, suspicious=%v, features=%v", 
				tc.name, analysis.Browser, analysis.Version, analysis.IsSuspicious, analysis.SuspiciousFeatures)
		})
	}
}

func TestParseUserAgent(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		ua           string
		expectedBrowser string
		expectVersion   bool
	}{
		{
			ua:             "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 Safari/537.36",
			expectedBrowser: "Chrome",
			expectVersion:   true,
		},
		{
			ua:             "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
			expectedBrowser: "Firefox",
			expectVersion:   true,
		},
		{
			ua:             "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 Safari/537.36 Edg/91.0.100.0",
			expectedBrowser: "Edge",
			expectVersion:   true,
		},
		{
			ua:             "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Safari/605.1.15",
			expectedBrowser: "Safari",
			expectVersion:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedBrowser, func(t *testing.T) {
			browser, version := detector.parseUserAgent(tc.ua)

			if browser != tc.expectedBrowser {
				t.Errorf("Expected browser '%s', got '%s'", tc.expectedBrowser, browser)
			}

			if tc.expectVersion && version == "0.0" {
				t.Errorf("Expected version to be parsed, got '%s'", version)
			}

			t.Logf("UA: %s -> Browser: %s, Version: %s", tc.ua[:50], browser, version)
		})
	}
}

func TestDetectOS(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		ua        string
		platform  string
		expectedOS string
	}{
		{
			ua:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			platform: "Win32",
			expectedOS: "Windows",
		},
		{
			ua:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
			platform: "MacIntel",
			expectedOS: "macOS",
		},
		{
			ua:        "Mozilla/5.0 (X11; Linux x86_64) Firefox/89.0",
			platform: "Linux x86_64",
			expectedOS: "Linux",
		},
		{
			ua:        "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 Chrome/91.0.4472.0 Mobile Safari/537.36",
			platform: "Linux armv8l",
			expectedOS: "Android",
		},
		{
			ua:        "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 Mobile Safari/604.1",
			platform: "iPhone",
			expectedOS: "iOS",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedOS, func(t *testing.T) {
			os := detector.detectOS(tc.ua, tc.platform)

			if os != tc.expectedOS {
				t.Errorf("Expected OS '%s', got '%s'", tc.expectedOS, os)
			}
		})
	}
}

func TestCalculateFingerprintEntropy(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name           string
		info           *EnvInfo
		minEntropy     float64
	}{
		{
			name: "Full Fingerprint",
			info: &EnvInfo{
				CanvasFingerprint: "abcdefghijklmnopqrstuvwxyz123456",
				WebGLRenderer:     "NVIDIA GeForce GTX 1080",
				Fonts:             []string{"Arial", "Helvetica", "Times New Roman", "Verdana"},
				Languages:         []string{"en-US", "en", "zh-CN"},
				ScreenWidth:       1920,
				ScreenHeight:      1080,
				Timezone:          "America/New_York",
			},
			minEntropy: 40,
		},
		{
			name: "Minimal Fingerprint",
			info: &EnvInfo{
				CanvasFingerprint: "abc",
				WebGLRenderer:     "",
				Fonts:             []string{},
				Languages:         []string{},
				ScreenWidth:       0,
				ScreenHeight:      0,
				Timezone:          "",
			},
			minEntropy: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entropy := detector.CalculateFingerprintEntropy(tc.info)

			if entropy < tc.minEntropy {
				t.Errorf("Expected entropy >= %.2f, got %.2f", tc.minEntropy, entropy)
			}

			if entropy > 100 {
				t.Errorf("Entropy should be capped at 100, got %.2f", entropy)
			}

			t.Logf("Test case '%s': entropy=%.2f", tc.name, entropy)
		})
	}
}

func TestAnalyzeNetworkEnvironment(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name           string
		ip             string
		headers        map[string]string
		info           *EnvInfo
		expectVPN      bool
		expectProxy    bool
		expectTor      bool
		minRiskScore   float64
	}{
		{
			name:        "Direct Connection",
			ip:          "203.0.113.1",
			headers:     map[string]string{},
			info:        &EnvInfo{WebGLRenderer: "NVIDIA GeForce GTX 1080", WebGLVendor: "NVIDIA"},
			expectVPN:   false,
			expectProxy: false,
			expectTor:   false,
			minRiskScore: 0,
		},
		{
			name: "Proxy Connection",
			ip:   "203.0.113.1",
			headers: map[string]string{
				"X-Forwarded-For": "192.0.2.1, 192.168.1.1, 10.0.0.1",
				"X-Real-IP":       "192.0.2.1",
			},
			info:          &EnvInfo{WebGLRenderer: "NVIDIA GeForce GTX 1080", WebGLVendor: "NVIDIA"},
			expectVPN:     false,
			expectProxy:  true,
			expectTor:     false,
			minRiskScore:  10,
		},
		{
			name: "VPN via WebGL",
			ip:   "203.0.113.1",
			headers: map[string]string{},
			info: &EnvInfo{
				WebGLRenderer: "VMware, Inc. VMware virtual platform",
				WebGLVendor:   "VMware",
			},
			expectVPN:    true,
			expectProxy:  false,
			expectTor:    false,
			minRiskScore: 20,
		},
		{
			name: "Tor Exit Node",
			ip:   "128.31.0.34",
			headers: map[string]string{},
			info:        &EnvInfo{WebGLRenderer: "Intel HD Graphics", WebGLVendor: "Intel"},
			expectVPN:   false,
			expectProxy: false,
			expectTor:   true,
			minRiskScore: 50,
		},
		{
			name: "Datacenter IP",
			ip:   "45.33.32.156",
			headers: map[string]string{},
			info:        &EnvInfo{WebGLRenderer: "Intel HD Graphics", WebGLVendor: "Intel"},
			expectVPN:    false,
			expectProxy:  false,
			expectTor:    false,
			minRiskScore: 20,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := detector.AnalyzeNetworkEnvironment(tc.ip, tc.headers, tc.info)

			if tc.expectVPN && !analysis.IsVPN {
				t.Errorf("Expected VPN detected")
			}
			if tc.expectProxy && !analysis.IsProxy {
				t.Errorf("Expected Proxy detected")
			}
			if tc.expectTor && !analysis.IsTor {
				t.Errorf("Expected Tor detected")
			}

			if analysis.RiskScore < tc.minRiskScore {
				t.Errorf("Expected risk score >= %.2f, got %.2f", tc.minRiskScore, analysis.RiskScore)
			}

			if analysis.RiskScore > 70 {
				t.Errorf("Risk score should be capped at 70, got %.2f", analysis.RiskScore)
			}

			t.Logf("Test case '%s': VPN=%v, Proxy=%v, Tor=%v, Datacenter=%v, RiskScore=%.2f, Evidence=%v",
				tc.name, analysis.IsVPN, analysis.IsProxy, analysis.IsTor, analysis.IsDatacenter, analysis.RiskScore, analysis.Evidence)
		})
	}
}

func TestDetectProxyViaHeaders(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name      string
		headers   map[string]string
		detected  bool
		minConf   float64
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "192.0.2.1, 192.168.1.1"},
			detected: true,
			minConf:  0.2,
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "192.0.2.1"},
			detected: true,
			minConf:  0.2,
		},
		{
			name:     "Via Header",
			headers:  map[string]string{"Via": "1.1 proxy.example.com"},
			detected: true,
			minConf:  0.2,
		},
		{
			name:     "Multiple Proxy Headers",
			headers:  map[string]string{"X-Forwarded-For": "192.0.2.1", "X-Real-IP": "10.0.0.1", "Via": "1.1 proxy"},
			detected: true,
			minConf:  0.5,
		},
		{
			name:     "No Headers",
			headers:  map[string]string{},
			detected: false,
			minConf:  0,
		},
		{
			name:     "Tor Indicator",
			headers:  map[string]string{"X-Forwarded-For": "tor.onion"},
			detected: true,
			minConf:  0.5,
		},
		{
			name:     "VPN Indicator",
			headers:  map[string]string{"X-Forwarded-For": "vpn.connection"},
			detected: true,
			minConf:  0.4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detected, confidence, evidence := detector.DetectProxyViaHeaders(tc.headers)

			if detected != tc.detected {
				t.Errorf("Expected detected=%v, got detected=%v", tc.detected, detected)
			}

			if detected && confidence < tc.minConf {
				t.Errorf("Expected confidence >= %.2f, got %.2f", tc.minConf, confidence)
			}

			t.Logf("Test case '%s': detected=%v, confidence=%.2f, evidence count=%d",
				tc.name, detected, confidence, len(evidence))
		})
	}
}

func TestDetectVPNViaASN(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		asn             int
		expectVPN       bool
		expectProvider  string
	}{
		{201229, true, "Private Internet Access"},
		{212502, true, "CyberGhost"},
		{202132, true, "NordVPN"},
		{203378, true, "ExpressVPN"},
		{19679, true, "Hide My Ass"},
		{49028, true, "HotSpot Shield"},
		{9009, true, "Mullvad"},
		{12345, false, ""},
		{54321, false, ""},
		{0, false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.expectProvider, func(t *testing.T) {
			isVPN, provider := detector.DetectVPNViaASN(tc.asn)

			if isVPN != tc.expectVPN {
				t.Errorf("For ASN %d, expected VPN=%v, got VPN=%v", tc.asn, tc.expectVPN, isVPN)
			}

			if tc.expectVPN && provider != tc.expectProvider {
				t.Errorf("For ASN %d, expected provider '%s', got '%s'", tc.asn, tc.expectProvider, provider)
			}
		})
	}
}

func TestDetectTorExitNode(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		ip        string
		expectTor bool
	}{
		{"128.31.0.34", true},
		{"199.87.154.10", true},
		{"199.58.186.11", true},
		{"171.25.193.9", true},
		{"162.247.72.27", true},
		{"45.33.32.156", true},
		{"104.244.76.14", true},
		{"77.247.181.219", true},
		{"93.95.227.23", true},
		{"8.8.8.8", false},
		{"203.0.113.1", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			isTor := detector.DetectTorExitNode(tc.ip)

			if isTor != tc.expectTor {
				t.Errorf("For IP %s, expected Tor=%v, got Tor=%v", tc.ip, tc.expectTor, isTor)
			}
		})
	}
}

func TestCalculateEnvScore(t *testing.T) {
	detector := NewEnvDetectorBackend()

	testCases := []struct {
		name           string
		info           *EnvInfo
		expectedMinScore float64
	}{
		{
			name: "Normal Environment",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
				CanvasFingerprint:   "abc123def456",
				WebGLRenderer:       "NVIDIA GeForce GTX 1080",
				WebGLVendor:         "NVIDIA",
				Fingerprint:         "fp123",
				Platform:           "Win32",
				ScreenWidth:         1920,
				ScreenHeight:        1080,
				HardwareConcurrency: 8,
			},
			expectedMinScore: 70,
		},
		{
			name: "Automated Environment",
			info: &EnvInfo{
				UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) HeadlessChrome/91.0",
				CanvasFingerprint:   "",
				WebGLRenderer:       "SwiftShader",
				WebGLVendor:         "Google Inc.",
				Fingerprint:         "",
				Platform:           "",
				ScreenWidth:         0,
				ScreenHeight:        0,
				HardwareConcurrency: 1,
				Fonts:               []string{},
			},
			expectedMinScore: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := detector.CalculateEnvScore(tc.info)

			if score < tc.expectedMinScore {
				t.Errorf("Expected score >= %.2f, got %.2f", tc.expectedMinScore, score)
			}

			if score > 100 {
				t.Errorf("Score should be capped at 100, got %.2f", score)
			}

			t.Logf("Test case '%s': score=%.2f", tc.name, score)
		})
	}
}
