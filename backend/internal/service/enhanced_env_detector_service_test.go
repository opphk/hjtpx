package service_test

import (
	"testing"
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
	Plugins             []string
	Fonts               []string
	TouchSupport        bool
	MaxTouchPoints      int
	HardwareConcurrency int
	Fingerprint         string
}

type testEnhancedEnvDetector struct {
	*testEnvDetector
}

func newTestEnhancedEnvDetector() *testEnhancedEnvDetector {
	return &testEnhancedEnvDetector{
		testEnvDetector: newTestEnvDetector(),
	}
}

func (d *testEnhancedEnvDetector) CalculateCanvasSimilarityEnhanced(hash1, hash2 string) *CanvasSimilarityResult {
	result := &CanvasSimilarityResult{
		Similarity:   0,
		IsSuspicious: false,
		Confidence:   0,
		Analysis:     "No analysis performed",
	}

	if hash1 == "" || hash2 == "" {
		result.Analysis = "Empty hash provided"
		return result
	}

	if hash1 == hash2 {
		result.Similarity = 100.0
		result.IsSuspicious = true
		result.Confidence = 0.95
		result.Analysis = "Identical fingerprints - potential fingerprinting injection"
		return result
	}

	if len(hash1) != len(hash2) {
		result.Analysis = "Different hash lengths indicate different environments"
		return result
	}

	matchCount := 0
	for i := 0; i < len(hash1); i++ {
		if hash1[i] == hash2[i] {
			matchCount++
		}
	}

	result.Similarity = float64(matchCount) / float64(len(hash1)) * 100.0
	result.Confidence = 0.85

	if result.Similarity > 95.0 {
		result.IsSuspicious = true
		result.Analysis = "High similarity (%.2f%%) - may indicate cloned environment"
	} else if result.Similarity > 85.0 {
		result.Analysis = "Moderate-high similarity (%.2f%%) - possible shared environment"
	}

	return result
}

func (d *testEnhancedEnvDetector) AnalyzeCanvasNoise(info *EnhancedEnvInfo) *CanvasNoiseAnalysis {
	analysis := &CanvasNoiseAnalysis{
		NoiseLevel:      0,
		Entropy:         0,
		PatternDetected: false,
		AnomalyScore:    0,
		Recommendations: []string{},
	}

	if info.CanvasFingerprint == "" {
		analysis.Recommendations = append(analysis.Recommendations, "Canvas fingerprint is empty")
		analysis.AnomalyScore = 50.0
		return analysis
	}

	hash := info.CanvasFingerprint
	if len(hash) < 32 {
		analysis.Recommendations = append(analysis.Recommendations, "Canvas fingerprint length too short")
		analysis.AnomalyScore += 20.0
	}

	if len(hash) > 128 {
		analysis.Recommendations = append(analysis.Recommendations, "Canvas fingerprint length unusually long")
		analysis.AnomalyScore += 10.0
	}

	hexCharCount := 0
	for _, c := range hash {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			hexCharCount++
		}
	}

	hexRatio := float64(hexCharCount) / float64(len(hash))
	if hexRatio < 0.9 {
		analysis.PatternDetected = true
		analysis.Recommendations = append(analysis.Recommendations, "Non-hexadecimal characters detected in fingerprint")
		analysis.AnomalyScore += 30.0
	}

	analysis.NoiseLevel = 1.0 - hexRatio

	return analysis
}

func (d *testEnhancedEnvDetector) AnalyzeWebGLRendererEnhanced(info *EnhancedEnvInfo) map[string]interface{} {
	analysis := make(map[string]interface{})

	if info.WebGLRenderer == "" {
		analysis["status"] = "missing"
		analysis["risk"] = "high"
		analysis["risk_score"] = 80.0
		analysis["reason"] = "WebGL renderer information missing"
		return analysis
	}

	analysis["renderer"] = info.WebGLRenderer
	analysis["vendor"] = info.WebGLVendor
	analysis["status"] = "present"

	rendererLower := toLower(info.WebGLRenderer)

	riskScore := 0.0
	riskFactors := []string{}

	if contains(rendererLower, "swiftshader") {
		riskScore += 60.0
		riskFactors = append(riskFactors, "Software renderer detected: swiftshader")
	}

	if contains(rendererLower, "headless") {
		riskScore += 80.0
		riskFactors = append(riskFactors, "Suspicious pattern: headless")
	}

	analysis["risk_score"] = riskScore
	analysis["risk_factors"] = riskFactors

	if riskScore >= 70 {
		analysis["risk"] = "high"
	} else if riskScore >= 40 {
		analysis["risk"] = "medium"
	} else {
		analysis["risk"] = "low"
	}

	return analysis
}

func (d *testEnhancedEnvDetector) EnhancedProxyVPNDetection(info *EnhancedEnvInfo, headers map[string]string) *ProxyVPNEnhancedResult {
	result := &ProxyVPNEnhancedResult{
		IsProxy:           false,
		IsVPN:             false,
		BlacklistMatch:    false,
		DNALeakDetected:   false,
		LatencyAnomaly:    false,
		Confidence:        0,
		RiskScore:         0,
		Evidence:          []string{},
		Recommendations:   []string{},
	}

	if headers["X-Forwarded-For"] != "" {
		result.IsProxy = true
		result.RiskScore += 25.0
		result.Evidence = append(result.Evidence, "Proxy headers detected")
	}

	rendererLower := toLower(info.WebGLRenderer)
	if contains(rendererLower, "virtual") || contains(rendererLower, "vmware") {
		result.IsVPN = true
		result.RiskScore += 40.0
		result.Evidence = append(result.Evidence, "VM indicator detected")
	}

	if result.RiskScore > 70 {
		result.Recommendations = append(result.Recommendations, "Block this connection")
	}

	return result
}

func (d *testEnhancedEnvDetector) EnhancedEmulatorDetection(info *EnhancedEnvInfo) *EmulatorEnhancedResult {
	result := &EmulatorEnhancedResult{
		IsEmulator:         false,
		BatteryAPIStatus:   "unknown",
		AudioContextStatus: "unknown",
		TouchFeatures:      []string{},
		SuspiciousPatterns: []string{},
		RiskScore:         0,
		Confidence:         0,
	}

	uaLower := toLower(info.UserAgent)

	if contains(uaLower, "android") {
		if info.MaxTouchPoints == 0 {
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, "Android without touch support")
			result.RiskScore += 25.0
		}
	}

	if contains(uaLower, "genymotion") || contains(uaLower, "bluestacks") {
		result.IsEmulator = true
		result.RiskScore += 50.0
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "Emulator detected")
	}

	if info.MaxTouchPoints > 0 {
		result.TouchFeatures = append(result.TouchFeatures, "touch_enabled")
	}

	if result.IsEmulator && len(result.SuspiciousPatterns) >= 1 {
		result.Confidence = 0.85
	}

	return result
}

type CanvasSimilarityResult struct {
	Similarity   float64
	IsSuspicious bool
	Confidence   float64
	Analysis     string
}

type CanvasNoiseAnalysis struct {
	NoiseLevel      float64
	Entropy         float64
	PatternDetected bool
	AnomalyScore    float64
	Recommendations []string
}

type ProxyVPNEnhancedResult struct {
	IsProxy          bool
	IsVPN            bool
	BlacklistMatch   bool
	DNALeakDetected  bool
	LatencyAnomaly   bool
	Confidence       float64
	RiskScore        float64
	Evidence         []string
	Recommendations  []string
}

type EmulatorEnhancedResult struct {
	IsEmulator         bool
	BatteryAPIStatus   string
	AudioContextStatus string
	TouchFeatures      []string
	SuspiciousPatterns []string
	RiskScore         float64
	Confidence         float64
}

type LatencyProfile struct {
	Latency         float64
	Jitter          float64
	PacketsLost     float64
	IsVPNLike       bool
	AnomalyScore    float64
	UnusualPatterns []string
}

func TestEnhancedCanvasSimilarity(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name           string
		hash1          string
		hash2          string
		expectedSim    float64
		isSuspicious   bool
	}{
		{
			name:         "Identical hashes",
			hash1:        "abc123def456",
			hash2:        "abc123def456",
			expectedSim:  100.0,
			isSuspicious: true,
		},
		{
			name:         "Empty first hash",
			hash1:        "",
			hash2:        "abc123",
			expectedSim:  0.0,
			isSuspicious: false,
		},
		{
			name:         "Empty second hash",
			hash1:        "abc123",
			hash2:        "",
			expectedSim:  0.0,
			isSuspicious: false,
		},
		{
			name:         "Different lengths",
			hash1:        "abc123",
			hash2:        "abc123def",
			expectedSim:  0.0,
			isSuspicious: false,
		},
		{
			name:         "Partial match",
			hash1:        "abc123",
			hash2:        "abc456",
			expectedSim:  50.0,
			isSuspicious: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.CalculateCanvasSimilarityEnhanced(tt.hash1, tt.hash2)
			if result.Similarity != tt.expectedSim {
				t.Errorf("CalculateCanvasSimilarityEnhanced() similarity = %v, expected %v", result.Similarity, tt.expectedSim)
			}
			if result.IsSuspicious != tt.isSuspicious {
				t.Errorf("CalculateCanvasSimilarityEnhanced() suspicious = %v, expected %v", result.IsSuspicious, tt.isSuspicious)
			}
		})
	}
}

func TestEnhancedCanvasNoiseAnalysis(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name          string
		envInfo       *EnhancedEnvInfo
		expectAnomaly bool
	}{
		{
			name: "Empty fingerprint",
			envInfo: &EnhancedEnvInfo{
				CanvasFingerprint: "",
			},
			expectAnomaly: true,
		},
		{
			name: "Short fingerprint",
			envInfo: &EnhancedEnvInfo{
				CanvasFingerprint: "abc",
			},
			expectAnomaly: true,
		},
		{
			name: "Normal fingerprint",
			envInfo: &EnhancedEnvInfo{
				CanvasFingerprint: "abcdef1234567890abcdef1234567890",
			},
			expectAnomaly: false,
		},
		{
			name: "Long fingerprint",
			envInfo: &EnhancedEnvInfo{
				CanvasFingerprint: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			expectAnomaly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.AnalyzeCanvasNoise(tt.envInfo)
			if tt.expectAnomaly && result.AnomalyScore == 0 {
				t.Errorf("AnalyzeCanvasNoise() expected anomaly but got score %v", result.AnomalyScore)
			}
			if !tt.expectAnomaly && result.AnomalyScore > 0 {
				t.Errorf("AnalyzeCanvasNoise() expected no anomaly but got score %v", result.AnomalyScore)
			}
		})
	}
}

func TestEnhancedWebGLRendererAnalysis(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name       string
		envInfo    *EnhancedEnvInfo
		expectRisk string
	}{
		{
			name: "Missing renderer",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "",
			},
			expectRisk: "high",
		},
		{
			name: "Software renderer - swiftshader",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "SwiftShader",
				WebGLVendor:   "Google",
			},
			expectRisk: "medium",
		},
		{
			name: "Headless browser",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "HeadlessRenderer",
				WebGLVendor:   "Test",
			},
			expectRisk: "high",
		},
		{
			name: "Normal GPU",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
				WebGLVendor:   "NVIDIA",
			},
			expectRisk: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.AnalyzeWebGLRendererEnhanced(tt.envInfo)
			if result["risk"] != tt.expectRisk {
				t.Errorf("AnalyzeWebGLRendererEnhanced() risk = %v, expected %v", result["risk"], tt.expectRisk)
			}
		})
	}
}

func TestEnhancedProxyVPNDetection(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name        string
		envInfo     *EnhancedEnvInfo
		headers     map[string]string
		expectProxy bool
		expectVPN   bool
	}{
		{
			name: "No proxy indicators",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
			},
			headers:     map[string]string{},
			expectProxy: false,
			expectVPN:   false,
		},
		{
			name: "Proxy header present",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
			},
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
			},
			expectProxy: true,
			expectVPN:   false,
		},
		{
			name: "VM indicator in renderer",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "VMware SVGA",
			},
			headers:     map[string]string{},
			expectProxy: false,
			expectVPN:   true,
		},
		{
			name: "Both proxy and VPN",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "VMware Virtual GPU",
			},
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			expectProxy: true,
			expectVPN:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedProxyVPNDetection(tt.envInfo, tt.headers)
			if result.IsProxy != tt.expectProxy {
				t.Errorf("EnhancedProxyVPNDetection() isProxy = %v, expected %v", result.IsProxy, tt.expectProxy)
			}
			if result.IsVPN != tt.expectVPN {
				t.Errorf("EnhancedProxyVPNDetection() isVPN = %v, expected %v", result.IsVPN, tt.expectVPN)
			}
		})
	}
}

func TestEnhancedEmulatorDetection(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name          string
		envInfo       *EnhancedEnvInfo
		expectEmulator bool
	}{
		{
			name: "Normal desktop browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
				MaxTouchPoints: 0,
			},
			expectEmulator: false,
		},
		{
			name: "Android emulator - Genymotion",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Linux; Android 11; Genymotion Build/RKQ1.200826.002) Chrome/120.0.0.0",
				MaxTouchPoints: 0,
			},
			expectEmulator: true,
		},
		{
			name: "Android emulator - BlueStacks",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Linux; Android 9; BlueStacks Build/PIKQ79) Chrome/120.0.0.0",
				MaxTouchPoints: 5,
			},
			expectEmulator: true,
		},
		{
			name: "Real Android device",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Linux; Android 13; SM-G998B) Chrome/120.0.0.0",
				MaxTouchPoints: 5,
			},
			expectEmulator: false,
		},
		{
			name: "Android without touch - suspicious",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Linux; Android 11) Chrome/120.0.0.0",
				MaxTouchPoints: 0,
			},
			expectEmulator: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedEmulatorDetection(tt.envInfo)
			if result.IsEmulator != tt.expectEmulator {
				t.Errorf("EnhancedEmulatorDetection() isEmulator = %v, expected %v", result.IsEmulator, tt.expectEmulator)
			}
		})
	}
}

func TestBatteryAPIEmulatorDetection(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name           string
		envInfo        *EnhancedEnvInfo
		expectedStatus string
	}{
		{
			name: "Android with touch - normal",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Linux; Android 11) Chrome/120.0.0.0",
				MaxTouchPoints: 5,
			},
			expectedStatus: "unknown",
		},
		{
			name: "Android without touch - suspicious",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Linux; Android 11) Chrome/120.0.0.0",
				MaxTouchPoints: 0,
			},
			expectedStatus: "unknown",
		},
		{
			name: "Desktop browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0",
				MaxTouchPoints: 0,
			},
			expectedStatus: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedEmulatorDetection(tt.envInfo)
			if result.BatteryAPIStatus != tt.expectedStatus {
				t.Errorf("EnhancedEmulatorDetection() BatteryAPIStatus = %v, expected %v", result.BatteryAPIStatus, tt.expectedStatus)
			}
		})
	}
}

func TestAudioContextEmulatorDetection(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name           string
		envInfo        *EnhancedEnvInfo
		expectedStatus string
	}{
		{
			name: "Normal browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0",
			},
			expectedStatus: "unknown",
		},
		{
			name: "Mobile browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0) Safari/604.1",
			},
			expectedStatus: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedEmulatorDetection(tt.envInfo)
			if result.AudioContextStatus != tt.expectedStatus {
				t.Errorf("EnhancedEmulatorDetection() AudioContextStatus = %v, expected %v", result.AudioContextStatus, tt.expectedStatus)
			}
		})
	}
}

func TestTouchFeaturesDetection(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name          string
		envInfo       *EnhancedEnvInfo
		expectTouch   bool
	}{
		{
			name: "Touch device",
			envInfo: &EnhancedEnvInfo{
				MaxTouchPoints: 5,
				TouchSupport:   true,
			},
			expectTouch: true,
		},
		{
			name: "Desktop without touch",
			envInfo: &EnhancedEnvInfo{
				MaxTouchPoints: 0,
				TouchSupport:   false,
			},
			expectTouch: false,
		},
		{
			name: "Touch device with 10 points",
			envInfo: &EnhancedEnvInfo{
				MaxTouchPoints: 10,
				TouchSupport:   true,
			},
			expectTouch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedEmulatorDetection(tt.envInfo)
			hasTouchFeature := len(result.TouchFeatures) > 0
			if hasTouchFeature != tt.expectTouch {
				t.Errorf("EnhancedEmulatorDetection() touchFeature = %v, expected %v", hasTouchFeature, tt.expectTouch)
			}
		})
	}
}

func TestProxyVPNRiskScoreCalculation(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name           string
		envInfo        *EnhancedEnvInfo
		headers        map[string]string
		minRiskScore   float64
		maxRiskScore   float64
	}{
		{
			name: "No risk factors",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
			},
			headers:      map[string]string{},
			minRiskScore: 0,
			maxRiskScore: 20,
		},
		{
			name: "Proxy detected",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
			},
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			minRiskScore: 20,
			maxRiskScore: 50,
		},
		{
			name: "VM detected",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "VMware Virtual GPU",
			},
			headers:      map[string]string{},
			minRiskScore: 30,
			maxRiskScore: 60,
		},
		{
			name: "Both proxy and VM",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "VMware Virtual GPU",
			},
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			minRiskScore: 50,
			maxRiskScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedProxyVPNDetection(tt.envInfo, tt.headers)
			if result.RiskScore < tt.minRiskScore || result.RiskScore > tt.maxRiskScore {
				t.Errorf("EnhancedProxyVPNDetection() riskScore = %v, expected between %v and %v", result.RiskScore, tt.minRiskScore, tt.maxRiskScore)
			}
		})
	}
}

func TestEmulatorConfidenceCalculation(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name              string
		envInfo           *EnhancedEnvInfo
		expectHighConfidence bool
	}{
		{
			name: "Emulator with patterns",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Linux; Android 11; Genymotion) Chrome/120.0.0.0",
				MaxTouchPoints: 0,
			},
			expectHighConfidence: true,
		},
		{
			name: "Normal browser",
			envInfo: &EnhancedEnvInfo{
				UserAgent:      "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0",
				MaxTouchPoints: 0,
			},
			expectHighConfidence: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedEmulatorDetection(tt.envInfo)
			if tt.expectHighConfidence && result.Confidence < 0.8 {
				t.Errorf("EnhancedEmulatorDetection() confidence = %v, expected high confidence", result.Confidence)
			}
			if !tt.expectHighConfidence && result.Confidence > 0.5 {
				t.Errorf("EnhancedEmulatorDetection() confidence = %v, expected low confidence", result.Confidence)
			}
		})
	}
}

func TestCanvasNoiseRecommendations(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name                string
		envInfo             *EnhancedEnvInfo
		expectRecommendations bool
	}{
		{
			name: "Empty fingerprint",
			envInfo: &EnhancedEnvInfo{
				CanvasFingerprint: "",
			},
			expectRecommendations: true,
		},
		{
			name: "Short fingerprint",
			envInfo: &EnhancedEnvInfo{
				CanvasFingerprint: "abc",
			},
			expectRecommendations: true,
		},
		{
			name: "Normal fingerprint",
			envInfo: &EnhancedEnvInfo{
				CanvasFingerprint: "abcdef1234567890abcdef1234567890",
			},
			expectRecommendations: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.AnalyzeCanvasNoise(tt.envInfo)
			hasRecommendations := len(result.Recommendations) > 0
			if hasRecommendations != tt.expectRecommendations {
				t.Errorf("AnalyzeCanvasNoise() recommendations = %v, expected %v", hasRecommendations, tt.expectRecommendations)
			}
		})
	}
}

func TestProxyVPNRecommendations(t *testing.T) {
	detector := newTestEnhancedEnvDetector()

	tests := []struct {
		name          string
		envInfo       *EnhancedEnvInfo
		headers       map[string]string
		expectBlock   bool
	}{
		{
			name: "High risk VPN",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "VMware Virtual GPU",
			},
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
			},
			expectBlock: true,
		},
		{
			name: "Low risk",
			envInfo: &EnhancedEnvInfo{
				WebGLRenderer: "NVIDIA GeForce GTX 1080",
			},
			headers:     map[string]string{},
			expectBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.EnhancedProxyVPNDetection(tt.envInfo, tt.headers)
			hasBlockRecommendation := false
			for _, rec := range result.Recommendations {
				if rec == "Block this connection" {
					hasBlockRecommendation = true
					break
				}
			}
			if hasBlockRecommendation != tt.expectBlock {
				t.Errorf("EnhancedProxyVPNDetection() blockRecommendation = %v, expected %v", hasBlockRecommendation, tt.expectBlock)
			}
		})
	}
}
