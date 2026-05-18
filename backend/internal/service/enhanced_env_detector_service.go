package service

import (
	"context"
	"fmt"
	"math"
	"strings"
)

type EnhancedEnvDetectorService struct {
	*EnvDetectorService
	enhancedDetector *EnhancedEnvDetector
}

type EnhancedEnvDetector struct {
	*EnvDetector
}

type CanvasNoiseAnalysis struct {
	NoiseLevel        float64 `json:"noise_level"`
	Entropy           float64 `json:"entropy"`
	PatternDetected   bool    `json:"pattern_detected"`
	AnomalyScore      float64 `json:"anomaly_score"`
	Recommendations   []string `json:"recommendations"`
}

type CanvasSimilarityResult struct {
	Similarity        float64 `json:"similarity"`
	IsSuspicious      bool    `json:"is_suspicious"`
	Confidence        float64 `json:"confidence"`
	Analysis          string  `json:"analysis"`
}

type WebGLPerformanceProfile struct {
	RenderTime        float64 `json:"render_time_ms"`
	TextureLoadTime   float64 `json:"texture_load_time_ms"`
	ShaderCompileTime float64 `json:"shader_compile_time_ms"`
	DrawCallCount     int     `json:"draw_call_count"`
	TriangleCount     int     `json:"triangle_count"`
	IsConsistent      bool    `json:"is_consistent"`
	AnomalyScore      float64 `json:"anomaly_score"`
}

type WebGLExtensionAnalysis struct {
	Extensions        []string `json:"extensions"`
	ExtensionCount    int      `json:"extension_count"`
	CriticalMissing   []string `json:"critical_missing"`
	SuspiciousExts    []string `json:"suspicious_extensions"`
	RiskScore         float64  `json:"risk_score"`
}

type DNSLeakResult struct {
	Detected          bool     `json:"detected"`
	DNSServers        []string `json:"dns_servers"`
	UnusualPatterns   []string `json:"unusual_patterns"`
	RiskScore         float64  `json:"risk_score"`
}

type LatencyProfile struct {
	Latency           float64   `json:"latency_ms"`
	Jitter            float64   `json:"jitter_ms"`
	PacketsLost       float64   `json:"packets_lost_percent"`
	IsVPNLike         bool      `json:"is_vpn_like"`
	AnomalyScore      float64   `json:"anomaly_score"`
	UnusualPatterns   []string  `json:"unusual_patterns"`
}

type ProxyVPNEnhancedResult struct {
	IsProxy            bool     `json:"is_proxy"`
	IsVPN              bool     `json:"is_vpn"`
	BlacklistMatch     bool     `json:"blacklist_match"`
	DNALeakDetected    bool     `json:"dns_leak_detected"`
	LatencyAnomaly     bool     `json:"latency_anomaly"`
	Confidence          float64  `json:"confidence"`
	RiskScore           float64  `json:"risk_score"`
	Evidence            []string `json:"evidence"`
	Recommendations     []string `json:"recommendations"`
}

type EmulatorEnhancedResult struct {
	IsEmulator         bool     `json:"is_emulator"`
	BatteryAPIStatus   string   `json:"battery_api_status"`
	AudioContextStatus string   `json:"audio_context_status"`
	TouchFeatures       []string `json:"touch_features"`
	SuspiciousPatterns []string `json:"suspicious_patterns"`
	RiskScore          float64  `json:"risk_score"`
	Confidence         float64  `json:"confidence"`
}

func NewEnhancedEnvDetectorService() *EnhancedEnvDetectorService {
	return &EnhancedEnvDetectorService{
		EnvDetectorService: NewEnvDetectorService(),
		enhancedDetector:   NewEnhancedEnvDetector(),
	}
}

func NewEnhancedEnvDetector() *EnhancedEnvDetector {
	return &EnhancedEnvDetector{
		EnvDetector: NewEnvDetectorBackend(),
	}
}

func (d *EnhancedEnvDetector) CalculateCanvasSimilarityEnhanced(hash1, hash2 string) *CanvasSimilarityResult {
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
		result.Analysis = fmt.Sprintf("High similarity (%.2f%%) - may indicate cloned environment", result.Similarity)
	} else if result.Similarity > 85.0 {
		result.Analysis = fmt.Sprintf("Moderate-high similarity (%.2f%%) - possible shared environment", result.Similarity)
	} else if result.Similarity > 70.0 {
		result.Analysis = fmt.Sprintf("Moderate similarity (%.2f%%) - different but related environments", result.Similarity)
	} else {
		result.Analysis = fmt.Sprintf("Low similarity (%.2f%%) - different environments", result.Similarity)
	}

	return result
}

func (d *EnhancedEnvDetector) AnalyzeCanvasNoise(info *EnvInfo) *CanvasNoiseAnalysis {
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

	charFreq := make(map[rune]int)
	for _, c := range hash {
		charFreq[c]++
	}

	maxFreq := 0
	totalChars := len(hash)
	for _, freq := range charFreq {
		if freq > maxFreq {
			maxFreq = freq
		}
	}

	dominantRatio := float64(maxFreq) / float64(totalChars)
	analysis.NoiseLevel = 1.0 - dominantRatio

	if dominantRatio > 0.5 {
		analysis.PatternDetected = true
		analysis.Recommendations = append(analysis.Recommendations, fmt.Sprintf("Unusual character dominance (%.1f%%)", dominantRatio*100))
		analysis.AnomalyScore += 25.0
	}

	analysis.Entropy = d.calculateEntropy(hash)

	if analysis.Entropy < 3.0 {
		analysis.Recommendations = append(analysis.Recommendations, "Low entropy indicates predictable fingerprint")
		analysis.AnomalyScore += 15.0
	}

	analysis.NoiseLevel = math.Min(analysis.NoiseLevel*100, 100)

	if analysis.AnomalyScore > 50 {
		analysis.Recommendations = append(analysis.Recommendations, "HIGH: Manual verification recommended")
	}

	return analysis
}

func (d *EnhancedEnvDetector) calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]float64)
	for _, c := range s {
		freq[c]++
	}

	entropy := 0.0
	for _, count := range freq {
		p := count / float64(len(s))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (d *EnhancedEnvDetector) AnalyzeWebGLRendererEnhanced(info *EnvInfo) map[string]interface{} {
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

	rendererLower := strings.ToLower(info.WebGLRenderer)
	vendorLower := strings.ToLower(info.WebGLVendor)

	riskScore := 0.0
	riskFactors := []string{}

	softwarePatterns := map[string]float64{
		"swiftshader": 60.0,
		"llvmpipe":    70.0,
		"mesa":        50.0,
		"software":    65.0,
		"virtual":     45.0,
		"emulated":    55.0,
		"google inc":  40.0,
	}

	for pattern, risk := range softwarePatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			riskScore += risk
			riskFactors = append(riskFactors, fmt.Sprintf("Software renderer detected: %s", pattern))
			break
		}
	}

	anonymizedPatterns := []string{"unknown", "generic", "default", "standard", "microsoft basic"}
	anonymizedCount := 0
	for _, pattern := range anonymizedPatterns {
		if strings.Contains(rendererLower, pattern) {
			anonymizedCount++
		}
	}
	if anonymizedCount >= 2 {
		riskScore += 30.0
		riskFactors = append(riskFactors, "Renderer appears anonymized")
	}

	suspiciousPatterns := map[string]float64{
		"headless":  80.0,
		"bot":       85.0,
		"automation": 75.0,
		"test":      60.0,
		"phantom":   90.0,
	}

	for pattern, risk := range suspiciousPatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			riskScore += risk
			riskFactors = append(riskFactors, fmt.Sprintf("Suspicious pattern: %s", pattern))
			break
		}
	}

	vmPatterns := map[string]float64{
		"vmware":     65.0,
		"virtualbox": 60.0,
		"parallels":  55.0,
		"qemu":       50.0,
		"kvm":        45.0,
		"hyperv":     50.0,
	}

	for pattern, risk := range vmPatterns {
		if strings.Contains(rendererLower, pattern) {
			riskScore += risk
			riskFactors = append(riskFactors, fmt.Sprintf("Virtual machine detected: %s", pattern))
			break
		}
	}

	riskScore = math.Min(riskScore, 100.0)

	analysis["risk_score"] = riskScore
	analysis["risk_factors"] = riskFactors

	if riskScore >= 70 {
		analysis["risk"] = "high"
		analysis["recommendation"] = "Block or review this request"
	} else if riskScore >= 40 {
		analysis["risk"] = "medium"
		analysis["recommendation"] = "Additional verification recommended"
	} else {
		analysis["risk"] = "low"
		analysis["recommendation"] = "Proceed normally"
	}

	return analysis
}

func (d *EnhancedEnvDetector) AnalyzeWebGLExtensions(info *EnvInfo) *WebGLExtensionAnalysis {
	analysis := &WebGLExtensionAnalysis{
		Extensions:      []string{},
		ExtensionCount:  0,
		CriticalMissing: []string{},
		SuspiciousExts:  []string{},
		RiskScore:       0,
	}

	if info.WebGLRenderer == "" {
		analysis.RiskScore = 50.0
		return analysis
	}

	commonExtensions := []string{
		"OES_texture_float",
		"WEBGL_debug_renderer_info",
		"EXT_texture_filter_anisotropic",
		"WEBGL_lose_context",
		"OES_standard_derivatives",
	}

	for _, ext := range commonExtensions {
		if strings.Contains(info.WebGLRenderer, ext) {
			analysis.Extensions = append(analysis.Extensions, ext)
		}
	}

	analysis.ExtensionCount = len(analysis.Extensions)

	if analysis.ExtensionCount < 3 {
		analysis.CriticalMissing = append(analysis.CriticalMissing, "Very few extensions available")
		analysis.RiskScore += 20.0
	}

	suspiciousExtPatterns := []string{
		"debug",
		"test",
		"mock",
		"fake",
	}

	for _, ext := range analysis.Extensions {
		extLower := strings.ToLower(ext)
		for _, pattern := range suspiciousExtPatterns {
			if strings.Contains(extLower, pattern) {
				analysis.SuspiciousExts = append(analysis.SuspiciousExts, ext)
				analysis.RiskScore += 15.0
				break
			}
		}
	}

	analysis.RiskScore = math.Min(analysis.RiskScore, 100.0)

	return analysis
}

func (d *EnhancedEnvDetector) DetectDNSLeak(info *EnvInfo) *DNSLeakResult {
	result := &DNSLeakResult{
		Detected:        false,
		DNSServers:      []string{},
		UnusualPatterns: []string{},
		RiskScore:       0,
	}

	if info.WebGLVendor != "" {
		vendorLower := strings.ToLower(info.WebGLVendor)
		if strings.Contains(vendorLower, "openvpn") || strings.Contains(vendorLower, "wireguard") {
			result.Detected = true
			result.RiskScore += 40.0
			result.UnusualPatterns = append(result.UnusualPatterns, "VPN DNS pattern in WebGL vendor")
		}
	}

	if info.WebGLRenderer != "" {
		rendererLower := strings.ToLower(info.WebGLRenderer)
		if strings.Contains(rendererLower, "dns") || strings.Contains(rendererLower, "resolver") {
			result.Detected = true
			result.RiskScore += 30.0
			result.UnusualPatterns = append(result.UnusualPatterns, "DNS-related string in renderer")
		}
	}

	result.RiskScore = math.Min(result.RiskScore, 100.0)

	return result
}

func (d *EnhancedEnvDetector) AnalyzeLatencyProfile(latencyMs, jitterMs, packetLossPercent float64) *LatencyProfile {
	profile := &LatencyProfile{
		Latency:         latencyMs,
		Jitter:          jitterMs,
		PacketsLost:     packetLossPercent,
		IsVPNLike:       false,
		AnomalyScore:    0,
		UnusualPatterns: []string{},
	}

	if latencyMs > 300 {
		profile.AnomalyScore += 25.0
		profile.UnusualPatterns = append(profile.UnusualPatterns, "High latency detected")
	}

	if jitterMs > 50 {
		profile.AnomalyScore += 20.0
		profile.UnusualPatterns = append(profile.UnusualPatterns, "High jitter detected")
	}

	if packetLossPercent > 5 {
		profile.AnomalyScore += 30.0
		profile.UnusualPatterns = append(profile.UnusualPatterns, "Significant packet loss")
	}

	if latencyMs > 200 && jitterMs > 30 {
		profile.IsVPNLike = true
		profile.AnomalyScore += 25.0
		profile.UnusualPatterns = append(profile.UnusualPatterns, "VPN-like connection characteristics")
	}

	profile.AnomalyScore = math.Min(profile.AnomalyScore, 100.0)

	return profile
}

func (d *EnhancedEnvDetector) EnhancedProxyVPNDetection(info *EnvInfo, headers map[string]string) *ProxyVPNEnhancedResult {
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

	proxyRisk := d.CalculateProxyRiskScore("", headers)
	if proxyRisk > 30 {
		result.IsProxy = true
		result.RiskScore += proxyRisk * 0.4
		result.Evidence = append(result.Evidence, fmt.Sprintf("Proxy headers detected (risk: %.1f)", proxyRisk))
	}

	isVPN, vpnConfidence, vpnEvidence := d.DetectVPNPatterns(info, headers)
	if isVPN {
		result.IsVPN = true
		result.RiskScore += vpnConfidence * 40
		result.Evidence = append(result.Evidence, vpnEvidence...)
	}

	dnsLeakResult := d.DetectDNSLeak(info)
	if dnsLeakResult.Detected {
		result.DNALeakDetected = true
		result.RiskScore += dnsLeakResult.RiskScore * 0.3
		result.Evidence = append(result.Evidence, "DNS leak pattern detected")
	}

	if info.WebGLRenderer != "" {
		rendererLower := strings.ToLower(info.WebGLRenderer)
		vmIndicators := []string{"vmware", "virtualbox", "parallels", "hyperv", "qemu"}
		for _, indicator := range vmIndicators {
			if strings.Contains(rendererLower, indicator) {
				result.RiskScore += 15.0
				result.Evidence = append(result.Evidence, fmt.Sprintf("VM indicator: %s", indicator))
				break
			}
		}
	}

	result.RiskScore = math.Min(result.RiskScore, 100.0)

	if result.IsVPN && result.DNALeakDetected {
		result.Confidence = 0.90
	} else if result.IsVPN || result.IsProxy {
		result.Confidence = 0.75
	} else if result.RiskScore > 30 {
		result.Confidence = 0.50
	} else {
		result.Confidence = 0.20
	}

	if result.RiskScore > 70 {
		result.Recommendations = append(result.Recommendations, "Block this connection")
	} else if result.RiskScore > 40 {
		result.Recommendations = append(result.Recommendations, "Require additional verification")
	}

	return result
}

func (d *EnhancedEnvDetector) EnhancedEmulatorDetection(info *EnvInfo) *EmulatorEnhancedResult {
	result := &EmulatorEnhancedResult{
		IsEmulator:         false,
		BatteryAPIStatus:   "unknown",
		AudioContextStatus: "unknown",
		TouchFeatures:      []string{},
		SuspiciousPatterns: []string{},
		RiskScore:         0,
		Confidence:         0,
	}

	emulatorDetected, indicators := d.DetectEmulatorIndicators(info)
	if emulatorDetected {
		result.IsEmulator = true
		result.RiskScore += 50.0
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, indicators...)
	}

	uaLower := strings.ToLower(info.UserAgent)

	batteryIndicators := []string{"android", "ios", "mobile"}
	batteryDetected := false
	for _, indicator := range batteryIndicators {
		if strings.Contains(uaLower, indicator) {
			batteryDetected = true
			break
		}
	}

	if batteryDetected {
		if info.MaxTouchPoints == 0 {
			result.BatteryAPIStatus = "suspicious"
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, "Mobile device without touch support")
			result.RiskScore += 20.0
		} else if info.MaxTouchPoints > 10 {
			result.BatteryAPIStatus = "suspicious"
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, "Unusually high touch point count")
			result.RiskScore += 15.0
		} else {
			result.BatteryAPIStatus = "normal"
		}
	}

	if info.HardwareConcurrency > 16 {
		result.SuspiciousPatterns = append(result.SuspiciousPatterns,
			fmt.Sprintf("Unusually high CPU core count: %d", info.HardwareConcurrency))
		result.RiskScore += 15.0
	}

	if info.HardwareConcurrency == 1 && !strings.Contains(uaLower, "mobile") {
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "Single core CPU on non-mobile device")
		result.RiskScore += 10.0
	}

	touchFeatures := []string{}
	if info.MaxTouchPoints > 0 {
		touchFeatures = append(touchFeatures, fmt.Sprintf("touch_points:%d", info.MaxTouchPoints))
	}
	if info.TouchSupport {
		touchFeatures = append(touchFeatures, "touch_enabled")
	}
	result.TouchFeatures = touchFeatures

	if strings.Contains(uaLower, "android") {
		if info.MaxTouchPoints == 0 {
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, "Android without touch support")
			result.RiskScore += 25.0
		}

		if info.HardwareConcurrency > 8 {
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, "Android with unusually high core count")
			result.RiskScore += 15.0
		}
	}

	result.RiskScore = math.Min(result.RiskScore, 100.0)

	if result.IsEmulator && len(result.SuspiciousPatterns) >= 2 {
		result.Confidence = 0.85
	} else if result.IsEmulator {
		result.Confidence = 0.65
	} else if result.RiskScore > 30 {
		result.Confidence = 0.45
	} else {
		result.Confidence = 0.15
	}

	return result
}

func (s *EnhancedEnvDetectorService) RunEnhancedDetection(ctx context.Context, info *EnvInfo, headers map[string]string) *EnvDetectionReport {
	report := s.enhancedDetector.RunAllChecks(info)

	canvasAnalysis := s.enhancedDetector.AnalyzeCanvasNoise(info)
	if canvasAnalysis.AnomalyScore > 30 {
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "canvas_noise_analysis",
			Risk:     "medium",
			Detected: true,
			Score:    int(canvasAnalysis.AnomalyScore),
			Reason:   fmt.Sprintf("Canvas anomaly score: %.1f, recommendations: %v", canvasAnalysis.AnomalyScore, canvasAnalysis.Recommendations),
		})
	}

	webglAnalysis := s.enhancedDetector.AnalyzeWebGLRendererEnhanced(info)
	if riskScore, ok := webglAnalysis["risk_score"].(float64); ok && riskScore > 30 {
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "webgl_renderer_analysis",
			Risk:     webglAnalysis["risk"].(string),
			Detected: true,
			Score:    int(riskScore),
			Reason:   fmt.Sprintf("WebGL risk: %.1f, factors: %v", riskScore, webglAnalysis["risk_factors"]),
		})
	}

	proxyVPNResult := s.enhancedDetector.EnhancedProxyVPNDetection(info, headers)
	if proxyVPNResult.RiskScore > 30 {
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "proxy_vpn_enhanced",
			Risk:     "high",
			Detected: true,
			Score:    int(proxyVPNResult.RiskScore),
			Reason:   fmt.Sprintf("Proxy/VPN risk: %.1f, evidence: %v", proxyVPNResult.RiskScore, proxyVPNResult.Evidence),
		})
	}

	emulatorResult := s.enhancedDetector.EnhancedEmulatorDetection(info)
	if emulatorResult.RiskScore > 30 {
		report.Checks = append(report.Checks, RiskCheckResult{
			Name:     "emulator_enhanced",
			Risk:     "medium",
			Detected: true,
			Score:    int(emulatorResult.RiskScore),
			Reason:   fmt.Sprintf("Emulator risk: %.1f, patterns: %v", emulatorResult.RiskScore, emulatorResult.SuspiciousPatterns),
		})
	}

	report.EnvScore = math.Max(0, math.Min(100, report.EnvScore))

	highRiskChecks := 0
	for _, check := range report.Checks {
		if check.Risk == "high" && check.Detected {
			highRiskChecks++
		}
	}

	if highRiskChecks >= 2 {
		report.RiskLevel = "high"
		report.Action = "block"
	} else if report.EnvScore < 60 {
		report.RiskLevel = "high"
	} else if report.EnvScore < 80 {
		report.RiskLevel = "medium"
	} else {
		report.RiskLevel = "low"
	}

	return report
}
