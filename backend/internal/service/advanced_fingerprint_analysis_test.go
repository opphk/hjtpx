package service

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAdvancedFingerprintAnalyzer_NewAnalyzer(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.database)
	assert.NotNil(t, analyzer.mlModel)
	assert.NotNil(t, analyzer.weights)
	assert.NotNil(t, analyzer.knownBotPatterns)
	assert.NotNil(t, analyzer.knownVPNRanges)
	assert.NotNil(t, analyzer.knownTorNodes)
}

func TestAdvancedFingerprintAnalyzer_InitWeights(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	expectedWeights := map[string]float64{
		"canvas":             14,
		"webgl":              16,
		"audio":              13,
		"fonts":              12,
		"webdriver":          22,
		"headless":           17,
		"proxyVPN":           20,
		"torExitNode":        17,
		"virtualization":     14,
		"automationFrameworks": 18,
	}

	for key, expected := range expectedWeights {
		assert.Equal(t, expected, analyzer.weights[key], "Weight mismatch for %s", key)
	}
}

func TestAdvancedFingerprintAnalyzer_InitBotPatterns(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	expectedPatterns := []struct {
		pattern  string
		category string
		weight   float64
	}{
		{"headless", "headless", 20},
		{"phantom", "headless", 25},
		{"puppeteer", "automation", 22},
		{"playwright", "automation", 22},
		{"selenium", "automation", 20},
		{"webdriver", "automation", 23},
	}

	for _, ep := range expectedPatterns {
		bot, exists := analyzer.knownBotPatterns[ep.pattern]
		assert.True(t, exists, "Bot pattern %s not found", ep.pattern)
		if exists {
			assert.Equal(t, ep.category, bot.Category)
			assert.Equal(t, ep.weight, bot.Weight)
		}
	}
}

func TestAdvancedFingerprintAnalyzer_InitVPNRanges(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	assert.NotEmpty(t, analyzer.knownVPNRanges)

	for _, vpn := range analyzer.knownVPNRanges {
		assert.NotEmpty(t, vpn.Start)
		assert.NotEmpty(t, vpn.End)
		assert.Equal(t, "VPN", vpn.Type)
	}
}

func TestAdvancedFingerprintAnalyzer_InitTorNodes(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	assert.NotEmpty(t, analyzer.knownTorNodes)

	for _, node := range analyzer.knownTorNodes {
		assert.NotEmpty(t, node.Start)
		assert.NotEmpty(t, node.End)
		assert.Equal(t, "Tor", node.Type)
	}
}

func TestMockMLModel_NewModel(t *testing.T) {
	model := NewMockMLModel()

	assert.NotNil(t, model)
	assert.NotNil(t, model.weights)
	assert.Equal(t, 0.75, model.threshold)
	assert.True(t, model.trained)
}

func TestMockMLModel_Predict(t *testing.T) {
	model := NewMockMLModel()

	features := map[string]float64{
		"canvas":     0.8,
		"webgl":     0.9,
		"audio":     0.7,
		"automation": 1.0,
	}

	score, err := model.Predict(features)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestMockMLModel_Predict_EmptyFeatures(t *testing.T) {
	model := NewMockMLModel()

	score, err := model.Predict(map[string]float64{})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, score, 0.0)
}

func TestAdvancedFingerprintAnalyzer_AnalyzeAdvancedFingerprint(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	data := map[string]interface{}{
		"user_agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
		"canvas_hash":      "abc123def456",
		"webgl_hash":       "xyz789",
		"audio_hash":       "audio123",
		"font_hash":        "font456",
		"plugin_hash":      "plugin789",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
		"platform":         "Win32",
		"hardware_concurrency": 8.0,
		"device_memory":     8.0,
		"webdriver":        false,
		"chain_categories": []interface{}{"automation", "fingerprint"},
		"timing_variance":  0.5,
	}

	analysis, err := analyzer.AnalyzeAdvancedFingerprint(data)
	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.NotNil(t, analysis.BaseFingerprint)
	assert.NotNil(t, analysis.MLFeatures)
	assert.NotNil(t, analysis.ChainAnalysis)
	assert.NotNil(t, analysis.AdvancedIndicators)
	assert.NotNil(t, analysis.NetworkAnalysis)
}

func TestAdvancedFingerprintAnalyzer_ExtractBaseFingerprint(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{}

	data := map[string]interface{}{
		"user_agent":         "TestAgent/1.0",
		"canvas_hash":        "canvas123",
		"webgl_hash":         "webgl456",
		"audio_hash":         "audio789",
		"font_hash":          "font101",
		"plugin_hash":       "plugin102",
		"screen_resolution":  "1366x768",
		"timezone":           "UTC",
		"language":           "en-US",
		"platform":           "Linux",
		"hardware_concurrency": 4.0,
		"device_memory":      4.0,
	}

	analyzer.extractBaseFingerprint(analysis, data)

	assert.NotEmpty(t, analysis.BaseFingerprint.FingerprintID)
	assert.Equal(t, "TestAgent/1.0", analysis.BaseFingerprint.UserAgent)
	assert.Equal(t, "canvas123", analysis.BaseFingerprint.CanvasHash)
	assert.Equal(t, 4, analysis.BaseFingerprint.HardwareConcurrency)
	assert.Equal(t, 4.0, analysis.BaseFingerprint.DeviceMemory)
}

func TestAdvancedFingerprintAnalyzer_ExtractMLFeatures(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		MLFeatures: &MLFeatures{},
	}

	data := map[string]interface{}{
		"chain_results": map[string]interface{}{
			"canvas": map[string]interface{}{
				"detected":   true,
				"score":      75.0,
				"duration_ms": 10.5,
				"detections": []interface{}{"canvas_stable", "canvas_entropy"},
			},
			"webgl": map[string]interface{}{
				"detected":   false,
				"score":      20.0,
				"duration_ms": 8.0,
			},
		},
		"timing_variance": 0.65,
	}

	analyzer.extractMLFeatures(analysis, data)

	assert.Equal(t, 2, analysis.MLFeatures.TotalChecks)
	assert.Equal(t, 1, analysis.MLFeatures.DetectedChecks)
	assert.Equal(t, 75.0, analysis.MLFeatures.MaxScore)
	assert.Equal(t, 47.5, analysis.MLFeatures.AvgScore)
	assert.Len(t, analysis.MLFeatures.SuspiciousPatterns, 2)
	assert.Equal(t, 0.65, analysis.MLFeatures.TimingVariance)
}

func TestAdvancedFingerprintAnalyzer_ExtractChainAnalysis(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		MLFeatures: &MLFeatures{},
	}

	data := map[string]interface{}{
		"chain_categories": []interface{}{"automation", "fingerprint", "network", "vm"},
		"chain_results": map[string]interface{}{
			"automation_check": map[string]interface{}{
				"detected":   true,
				"score":      80.0,
				"duration_ms": 15.0,
				"detections": []interface{}{"selenium_detected"},
			},
		},
		"duration_ms": 150.5,
	}

	analyzer.extractChainAnalysis(analysis, data)

	assert.Len(t, analysis.ChainAnalysis.ChainCategories, 4)
	assert.Equal(t, 4, analysis.ChainAnalysis.ChainLength)
	assert.Equal(t, 150.5, analysis.ChainAnalysis.Duration)
	assert.Contains(t, analysis.ChainAnalysis.ChainCategories, "automation")
	assert.Contains(t, analysis.ChainAnalysis.ChainCategories, "fingerprint")
	assert.Contains(t, analysis.ChainAnalysis.ChainCategories, "network")
	assert.Contains(t, analysis.ChainAnalysis.ChainCategories, "vm")
}

func TestAdvancedFingerprintAnalyzer_ExtractAdvancedIndicators(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		AdvancedIndicators: &AdvancedIndicators{},
		MLFeatures:         &MLFeatures{},
	}

	analysis.MLFeatures.SuspiciousPatterns = []string{
		"headless_detected",
		"webdriver_true",
		"proxy_ip_mismatch",
		"vmware_detected",
	}

	data := map[string]interface{}{}

	analyzer.extractAdvancedIndicators(analysis, data)

	assert.NotEmpty(t, analysis.AdvancedIndicators.HeadlessIndicators)
	assert.NotEmpty(t, analysis.AdvancedIndicators.AutomationIndicators)
	assert.NotEmpty(t, analysis.AdvancedIndicators.ProxyVPNIndicators)
	assert.NotEmpty(t, analysis.AdvancedIndicators.VirtualizationIndicators)
}

func TestAdvancedFingerprintAnalyzer_ExtractNetworkAnalysis(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		NetworkAnalysis: &NetworkAnalysis{},
	}

	data := map[string]interface{}{
		"webrtc_ips":     []interface{}{"8.8.8.8", "8.8.4.4"},
		"connection_type": "vpn",
		"network_latency": 45.5,
		"x_forwarded_for": "10.0.0.1, 10.0.0.2, 10.0.0.3",
		"via":            "1.1 proxy",
	}

	analyzer.extractNetworkAnalysis(analysis, data)

	assert.Equal(t, 2, analysis.NetworkAnalysis.WebRTCIPCount)
	assert.True(t, analysis.NetworkAnalysis.WebRTCLeakRisk)
	assert.Equal(t, "vpn", analysis.NetworkAnalysis.ConnectionType)
	assert.Equal(t, 45.5, analysis.NetworkAnalysis.Latency)
	assert.True(t, analysis.NetworkAnalysis.MultiHopProxy)
	assert.True(t, analysis.NetworkAnalysis.IsProxy)
	assert.Contains(t, analysis.NetworkAnalysis.HeadersPresent, "X-Forwarded-For")
	assert.Contains(t, analysis.NetworkAnalysis.HeadersPresent, "Via")
}

func TestAdvancedFingerprintAnalyzer_IsTorIP(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	assert.True(t, analyzer.isTorIP("128.31.0.1"))
	assert.True(t, analyzer.isTorIP("131.188.1.1"))
	assert.False(t, analyzer.isTorIP("8.8.8.8"))
	assert.False(t, analyzer.isTorIP("192.168.1.1"))
}

func TestAdvancedFingerprintAnalyzer_IsVPNIP(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	assert.True(t, analyzer.isVPNIP("45.33.1.1"))
	assert.True(t, analyzer.isVPNIP("104.238.1.1"))
	assert.False(t, analyzer.isVPNIP("8.8.8.8"))
	assert.False(t, analyzer.isVPNIP("192.168.1.1"))
}

func TestAdvancedFingerprintAnalyzer_IsDatacenterIP(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	assert.True(t, analyzer.isDatacenterIP("3.1.1.1"))
	assert.True(t, analyzer.isDatacenterIP("45.33.1.1"))
	assert.True(t, analyzer.isDatacenterIP("52.1.1.1"))
	assert.False(t, analyzer.isDatacenterIP("8.8.8.8"))
	assert.False(t, analyzer.isDatacenterIP("192.168.1.1"))
}

func TestAdvancedFingerprintAnalyzer_CalculateMLRiskScore(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		MLFeatures: &MLFeatures{
			DetectedChecks:    5,
			AutomationScore:   30,
			FingerprintScore:  20,
			NetworkScore:      25,
			VMScore:           20,
			TimingVariance:    0.9,
			EntropyScore:      0.15,
			ConsistencyScore: 0.6,
			SuspiciousPatterns: []string{
				"headless_detected",
				"puppeteer_marker",
				"tor_exit_node",
			},
		},
	}

	analyzer.calculateMLRiskScore(analysis)

	assert.Greater(t, analysis.MLRiskScore, 0.0)
	assert.LessOrEqual(t, analysis.MLRiskScore, 100.0)
}

func TestAdvancedFingerprintAnalyzer_CalculateBehaviorScore(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		MLFeatures: &MLFeatures{
			TimingVariance:      0.7,
			EntropyScore:        0.1,
			ConsistencyScore:   0.5,
		},
		AdvancedIndicators: &AdvancedIndicators{
			BehavioralIndicators: []string{"irregular_timing", "low_entropy"},
		},
	}

	analyzer.calculateBehaviorScore(analysis)

	assert.Greater(t, analysis.BehaviorScore, 0.0)
	assert.LessOrEqual(t, analysis.BehaviorScore, 100.0)
}

func TestAdvancedFingerprintAnalyzer_CalculateConsistencyScore(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		ChainAnalysis: &ChainAnalysis{
			ChainResults: map[string]*ChainResult{
				"canvas_stable": {Score: 20.0, Detected: true},
				"canvas_entropy": {Score: 25.0, Detected: true},
			},
		},
		AdvancedIndicators: &AdvancedIndicators{
			HeadlessIndicators:    []string{},
			AutomationIndicators:   []string{"selenium"},
		},
	}

	analyzer.calculateConsistencyScore(analysis)

	assert.GreaterOrEqual(t, analysis.ConsistencyScore, 0.0)
	assert.LessOrEqual(t, analysis.ConsistencyScore, 1.0)
}

func TestAdvancedFingerprintAnalyzer_CalculateEntropyScore(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	analysis := &AdvancedFingerprintAnalysis{
		MLFeatures: &MLFeatures{
			SuspiciousPatterns: []string{
				"headless",
				"webdriver",
				"puppeteer",
				"vpn",
			},
		},
		AdvancedIndicators: &AdvancedIndicators{
			HeadlessIndicators:      []string{"headless"},
			AutomationIndicators:    []string{"webdriver"},
			ProxyVPNIndicators:      []string{"vpn"},
		},
	}

	analyzer.calculateEntropyScore(analysis)

	assert.GreaterOrEqual(t, analysis.EntropyScore, 0.0)
	assert.LessOrEqual(t, analysis.EntropyScore, 1.0)
}

func TestAdvancedFingerprintAnalyzer_DetectAdvancedBotIndicators(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	fp := &FingerprintAnalysis{
		UserAgent: "Mozilla/5.0 (HeadlessChrome)",
	}

	data := map[string]interface{}{
		"navigator.webdriver": true,
		"$cdc_asdjflasutopfhvcZLmcfl_": true,
		"plugins_count": 0.0,
		"languages_count": 0.0,
		"window.outerWidth": 0.0,
		"webgl_renderer": "llvmpipe Software",
	}

	analyzer.detectAdvancedBotIndicators(fp, data)

	assert.True(t, fp.IsKnownBot)
	assert.NotEmpty(t, fp.RiskIndicators)
	assert.Greater(t, fp.AnomalyScore, 0.0)
}

func TestAdvancedFingerprintAnalyzer_DetectAdvancedVPNIndicators(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()
	fp := &FingerprintAnalysis{}

	data := map[string]interface{}{
		"webrtc_ips":     []interface{}{"192.168.1.1", "8.8.8.8"},
		"connection_type": "vpn",
		"public_ip":      "45.33.1.1",
		"x_forwarded_for": "10.0.0.1, 10.0.0.2, 10.0.0.3, 10.0.0.4",
	}

	analyzer.detectAdvancedVPNIndicators(fp, data)

	assert.True(t, fp.IsKnownVPN)
	assert.NotEmpty(t, fp.RiskIndicators)
}

func TestAdvancedFingerprintAnalyzer_CalculateAdvancedConfidence(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	fp := &FingerprintAnalysis{
		CanvasHash:       "hash1",
		WebGLHash:        "hash2",
		AudioHash:        "hash3",
		FontHash:         "hash4",
		UserAgent:        "UA",
		ScreenResolution: "1920x1080",
	}

	confidence := analyzer.calculateAdvancedConfidence(fp)
	assert.Greater(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 1.0)
}

func TestGenerateAdvancedFingerprintID(t *testing.T) {
	data := map[string]interface{}{
		"user_agent":    "Mozilla/5.0",
		"canvas_hash":   "abc123",
		"webgl_hash":    "def456",
		"screen_resolution": "1920x1080",
	}

	id := generateAdvancedFingerprintID(data)
	assert.NotEmpty(t, id)
	assert.Len(t, id, 21)
}

func TestEnhancedRiskScorer_NewScorer(t *testing.T) {
	scorer := NewEnhancedRiskScorer()

	assert.NotNil(t, scorer)
	assert.NotNil(t, scorer.weights)
	assert.NotNil(t, scorer.categories)
}

func TestEnhancedRiskScorer_CalculateScore(t *testing.T) {
	scorer := NewEnhancedRiskScorer()

	analysis := &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{
			AnomalyScore: 50.0,
		},
		AdvancedIndicators: &AdvancedIndicators{
			AutomationIndicators:  []string{"a", "b", "c", "d"},
			ProxyVPNIndicators:   []string{"vpn1", "vpn2", "vpn3"},
			VirtualizationIndicators: []string{"vm1", "vm2", "vm3"},
		},
		MLRiskScore:   60.0,
		BehaviorScore: 40.0,
	}

	score := scorer.CalculateScore(analysis)

	assert.Greater(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestPatternMatcher_NewMatcher(t *testing.T) {
	matcher := NewPatternMatcher()

	assert.NotNil(t, matcher)
	assert.True(t, matcher.compiled)
}

func TestPatternMatcher_Match(t *testing.T) {
	matcher := NewPatternMatcher()

	text := "This browser is running in headless mode with puppeteer automation"

	results := matcher.Match(text)

	assert.NotEmpty(t, results)
}

func TestPatternMatcher_Match_NoMatch(t *testing.T) {
	matcher := NewPatternMatcher()

	text := "This is a normal browser without any automation"

	results := matcher.Match(text)

	assert.Empty(t, results)
}

func TestMatchResult_GetCategory(t *testing.T) {
	pattern := &CompiledPattern{
		Category:    "automation",
		Weight:      20.0,
		Description: "Test pattern",
	}

	result := &MatchResult{
		Pattern:     pattern,
		MatchedText: "matched",
	}

	assert.Equal(t, "automation", result.GetCategory())
}

func TestMatchResult_GetWeight(t *testing.T) {
	pattern := &CompiledPattern{
		Category:    "automation",
		Weight:      25.0,
		Description: "Test pattern",
	}

	result := &MatchResult{
		Pattern:     pattern,
		MatchedText: "matched",
	}

	assert.Equal(t, 25.0, result.GetWeight())
}

func TestMatchResult_GetDescription(t *testing.T) {
	pattern := &CompiledPattern{
		Category:    "automation",
		Weight:      20.0,
		Description: "Automation framework detected",
	}

	result := &MatchResult{
		Pattern:     pattern,
		MatchedText: "selenium",
	}

	assert.Equal(t, "Automation framework detected", result.GetDescription())
}

func TestCalculateRiskLevel(t *testing.T) {
	tests := []struct {
		score    float64
		expected RiskLevel
	}{
		{90.0, RiskLevelCritical},
		{80.0, RiskLevelCritical},
		{65.0, RiskLevelHigh},
		{60.0, RiskLevelHigh},
		{45.0, RiskLevelMedium},
		{40.0, RiskLevelMedium},
		{30.0, RiskLevelLow},
		{10.0, RiskLevelLow},
	}

	for _, tt := range tests {
		level := CalculateRiskLevel(tt.score)
		assert.Equal(t, tt.expected, level, "Score: %f", tt.score)
	}
}

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskLevelLow, "low"},
		{RiskLevelMedium, "medium"},
		{RiskLevelHigh, "high"},
		{RiskLevelCritical, "critical"},
		{RiskLevel(100), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.String())
	}
}

func TestAdvancedFingerprintAnalyzer_GenerateRiskReport(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	analysis := &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{
			AnomalyScore: 85.0,
		},
		MLRiskScore:     70.0,
		BehaviorScore:   60.0,
		ConsistencyScore: 0.5,
		EntropyScore:    0.3,
		AdvancedIndicators: &AdvancedIndicators{
			AutomationIndicators:    []string{"headless", "puppeteer"},
			ProxyVPNIndicators:      []string{"vpn"},
			VirtualizationIndicators: []string{"vmware"},
		},
	}

	report := analyzer.GenerateRiskReport(analysis)

	assert.NotNil(t, report)
	assert.Equal(t, RiskLevelCritical, report.RiskLevel)
	assert.Greater(t, report.FinalScore, 0.0)
	assert.NotEmpty(t, report.Indicators)
}

func TestRiskReport_ToJSON(t *testing.T) {
	report := &RiskReport{
		Timestamp:       testTime(),
		FinalScore:      75.5,
		RiskLevel:       RiskLevelHigh,
		BaseScore:       60.0,
		MLScore:         70.0,
		BehaviorScore:   55.0,
		ConsistencyScore: 0.6,
		EntropyScore:    0.4,
		Indicators:      []string{"headless", "vpn"},
		Categories:      map[string]int{"automation": 1, "network": 1},
		Recommendations: []string{"Add extra verification", "Monitor closely"},
	}

	jsonData, err := report.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
}

func TestRiskReport_GetSummary(t *testing.T) {
	report := &RiskReport{
		FinalScore:  75.5,
		RiskLevel:   RiskLevelHigh,
		Indicators:  []string{"a", "b", "c"},
		Categories:  map[string]int{"automation": 2, "network": 1},
	}

	summary := report.GetSummary()
	assert.Contains(t, summary, "Risk Level: high")
	assert.Contains(t, summary, "Score: 75.5")
	assert.Contains(t, summary, "3 indicators")
}

func TestExtractMLFeaturesFromData(t *testing.T) {
	data := map[string]interface{}{
		"chain_results": map[string]interface{}{
			"check1": map[string]interface{}{
				"detected":   true,
				"score":      80.0,
				"detections": []interface{}{"pattern1"},
			},
		},
		"chain_categories": []interface{}{"automation", "network"},
		"timing_variance":  0.75,
	}

	ml := extractMLFeaturesFromData(data)

	assert.Equal(t, 1, ml.TotalChecks)
	assert.Equal(t, 1, ml.DetectedChecks)
	assert.Equal(t, 0.75, ml.TimingVariance)
	assert.NotEmpty(t, ml.AutomationScore)
	assert.NotEmpty(t, ml.NetworkScore)
}

func TestCalculateEntropy(t *testing.T) {
	data := []float64{1.0, 1.0, 2.0, 3.0, 3.0, 3.0}

	entropy := calculateEntropy(data)
	assert.Greater(t, entropy, 0.0)
}

func TestCalculateEntropy_EmptyData(t *testing.T) {
	entropy := calculateEntropy([]float64{})
	assert.Equal(t, 0.0, entropy)
}

func TestAdvancedFingerprintAnalyzer_AnalyzeTemporalPattern(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	data := map[string]interface{}{
		"request_timestamps": []interface{}{
			1000.0, 1100.0, 1200.0, 1300.0, 1400.0,
		},
	}

	analysis := analyzer.AnalyzeTemporalPattern(data)

	assert.NotNil(t, analysis)
	assert.Equal(t, 5, analysis.RequestCount)
	assert.True(t, analysis.MinInterval < 1.0)
	assert.True(t, analysis.SuspiciousPattern)
}

func TestAdvancedFingerprintAnalyzer_AnalyzeTemporalPattern_TooRegular(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	timestamps := make([]interface{}, 60)
	for i := 0; i < 60; i++ {
		timestamps[i] = float64(1000 + i*100)
	}

	data := map[string]interface{}{
		"request_timestamps": timestamps,
	}

	analysis := analyzer.AnalyzeTemporalPattern(data)

	assert.True(t, analysis.SuspiciousPattern)
	assert.Equal(t, "too_regular", analysis.PatternType)
}

func TestCalculateAverage(t *testing.T) {
	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}

	avg := calculateAverage(values)
	assert.Equal(t, 30.0, avg)
}

func TestCalculateAverage_Empty(t *testing.T) {
	avg := calculateAverage([]float64{})
	assert.Equal(t, 0.0, avg)
}

func TestCalculateMin(t *testing.T) {
	values := []float64{50.0, 30.0, 10.0, 40.0, 20.0}

	min := calculateMin(values)
	assert.Equal(t, 10.0, min)
}

func TestCalculateMax(t *testing.T) {
	values := []float64{10.0, 30.0, 50.0, 40.0, 20.0}

	max := calculateMax(values)
	assert.Equal(t, 50.0, max)
}

func TestVariance(t *testing.T) {
	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}

	v := variance(values)
	assert.Greater(t, v, 0.0)
}

func TestVariance_LessThanTwo(t *testing.T) {
	v := variance([]float64{10.0})
	assert.Equal(t, 0.0, v)
}

func TestDetectionChain_NewChain(t *testing.T) {
	chain := NewDetectionChain("test-chain-123")

	assert.NotNil(t, chain)
	assert.Equal(t, "test-chain-123", chain.ID)
	assert.Empty(t, chain.Methods)
	assert.Empty(t, chain.Results)
	assert.NotZero(t, chain.StartTime)
}

func TestDetectionChain_AddMethod(t *testing.T) {
	chain := NewDetectionChain("test")

	chain.AddMethod("canvas")
	chain.AddMethod("webgl")

	assert.Len(t, chain.Methods, 2)
	assert.Contains(t, chain.Methods, "canvas")
	assert.Contains(t, chain.Methods, "webgl")
}

func TestDetectionChain_AddResult(t *testing.T) {
	chain := NewDetectionChain("test")

	result := &ChainResult{
		Detected:   true,
		Score:      75.0,
		Duration:   10.5,
		Detections: []string{"pattern1"},
	}

	chain.AddResult("canvas", result)

	assert.Contains(t, chain.Results, "canvas")
	assert.True(t, chain.Results["canvas"].Detected)
}

func TestDetectionChain_Complete(t *testing.T) {
	chain := NewDetectionChain("test")

	chain.AddResult("canvas", &ChainResult{Detected: true, Score: 80.0})
	chain.AddResult("webgl", &ChainResult{Detected: false, Score: 20.0})

	chain.Complete()

	assert.NotZero(t, chain.EndTime)
	assert.NotZero(t, chain.Duration)
	assert.Greater(t, chain.Score, 0.0)
}

func TestDetectionChain_GetDetectedMethods(t *testing.T) {
	chain := NewDetectionChain("test")

	chain.AddResult("canvas", &ChainResult{Detected: true, Score: 80.0})
	chain.AddResult("webgl", &ChainResult{Detected: false, Score: 20.0})
	chain.AddResult("audio", &ChainResult{Detected: true, Score: 70.0})

	detected := chain.getDetectedMethods()

	assert.Len(t, detected, 2)
	assert.Contains(t, detected, "canvas")
	assert.Contains(t, detected, "audio")
}

func TestDetectionChain_ToReport(t *testing.T) {
	chain := NewDetectionChain("test")
	chain.AddMethod("canvas")
	chain.AddMethod("webgl")
	chain.AddResult("canvas", &ChainResult{Detected: true, Score: 80.0})
	chain.AddResult("webgl", &ChainResult{Detected: true, Score: 75.0})
	chain.Complete()

	report := chain.ToReport()

	assert.Equal(t, "test", report.ID)
	assert.Equal(t, 2, report.MethodCount)
	assert.Equal(t, 2, report.DetectedCount)
	assert.True(t, report.Suspicious)
}

func TestAdvancedFingerprintDatabase_NewDatabase(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	assert.NotNil(t, db)
	assert.NotNil(t, db.advancedData)
}

func TestAdvancedFingerprintDatabase_AddAndGetAnalysis(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	analysis := &AdvancedFingerprintAnalysis{
		MLRiskScore: 75.0,
	}

	db.AddAdvancedAnalysis("fp-123", analysis)

	retrieved, exists := db.GetAdvancedAnalysis("fp-123")
	assert.True(t, exists)
	assert.Equal(t, 75.0, retrieved.MLRiskScore)
}

func TestAdvancedFingerprintDatabase_GetAllAnalyses(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	db.AddAdvancedAnalysis("fp-1", &AdvancedFingerprintAnalysis{})
	db.AddAdvancedAnalysis("fp-2", &AdvancedFingerprintAnalysis{})
	db.AddAdvancedAnalysis("fp-3", &AdvancedFingerprintAnalysis{})

	analyses := db.GetAllAdvancedAnalyses()
	assert.Len(t, analyses, 3)
}

func TestAdvancedFingerprintDatabase_CleanupOldData(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	analysis := &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{
			LastSeen:    testTime().Add(-48 * time.Hour),
			RequestCount: 2,
		},
	}
	db.AddAdvancedAnalysis("old-fp", analysis)

	removed := db.CleanupOldAdvancedData(24 * time.Hour)
	assert.Equal(t, 1, removed)

	_, exists := db.GetAdvancedAnalysis("old-fp")
	assert.False(t, exists)
}

func TestAdvancedFingerprintDatabase_ExportImport(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	db.AddAdvancedAnalysis("fp-export", &AdvancedFingerprintAnalysis{
		MLRiskScore: 85.0,
	})

	data, err := db.ExportAll()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	newDB := NewAdvancedFingerprintDatabase()
	err = newDB.ImportAll(data)
	assert.NoError(t, err)

	retrieved, exists := newDB.GetAdvancedAnalysis("fp-export")
	assert.True(t, exists)
	assert.Equal(t, 85.0, retrieved.MLRiskScore)
}

func TestAdvancedFingerprintDatabase_GetRiskDistribution(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	db.AddAdvancedAnalysis("fp1", &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{AnomalyScore: 85.0},
	})
	db.AddAdvancedAnalysis("fp2", &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{AnomalyScore: 65.0},
	})
	db.AddAdvancedAnalysis("fp3", &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{AnomalyScore: 35.0},
	})

	distribution := db.GetRiskDistribution()

	assert.Greater(t, distribution["critical"], 0)
	assert.Greater(t, distribution["high"], 0)
	assert.Greater(t, distribution["medium"], 0)
}

func TestAdvancedFingerprintDatabase_GetTopRiskFactors(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	db.AddAdvancedAnalysis("fp1", &AdvancedFingerprintAnalysis{
		AdvancedIndicators: &AdvancedIndicators{
			HeadlessIndicators: []string{"headless_detected"},
		},
	})
	db.AddAdvancedAnalysis("fp2", &AdvancedFingerprintAnalysis{
		AdvancedIndicators: &AdvancedIndicators{
			HeadlessIndicators: []string{"headless_detected"},
			AutomationIndicators: []string{"puppeteer"},
		},
	})

	factors := db.GetTopRiskFactors(5)
	assert.NotEmpty(t, factors)
}

func TestAdvancedFingerprintDatabase_GetAnalytics(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	db.AddAdvancedAnalysis("fp1", &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{IsKnownBot: true, AnomalyScore: 80.0},
	})
	db.AddAdvancedAnalysis("fp2", &AdvancedFingerprintAnalysis{
		BaseFingerprint: &FingerprintAnalysis{IsKnownVPN: true, AnomalyScore: 60.0},
	})

	analytics := db.GetAnalytics()

	assert.Equal(t, 2, analytics.TotalAnalyses)
	assert.Equal(t, 1, analytics.BotCount)
	assert.Equal(t, 1, analytics.VPNCount)
}

func TestDatabaseAnalytics_ToJSON(t *testing.T) {
	analytics := &DatabaseAnalytics{
		TotalFingerprints: 100,
		TotalAnalyses:     50,
		BotCount:          10,
		VPNCount:          5,
		AvgRiskScore:      45.5,
		RiskDistribution:  map[string]int{"low": 20, "medium": 15, "high": 10, "critical": 5},
		TopRiskFactors:    []string{"headless", "puppeteer"},
	}

	jsonData, err := analytics.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
}

func TestAdvancedIndicators_ExtractAllPatterns(t *testing.T) {
	indicators := &AdvancedIndicators{
		HeadlessIndicators:        []string{"h1", "h2"},
		AutomationIndicators:      []string{"a1", "a2", "a3"},
		ProxyVPNIndicators:        []string{"p1"},
		VirtualizationIndicators:  []string{"v1", "v2"},
		VMIndicators:              []string{"vm1"},
		SandboxIndicators:         []string{"s1"},
		BehavioralIndicators:      []string{"b1"},
		NetworkIndicators:         []string{"n1", "n2"},
	}

	all := indicators.ExtractAllPatterns()

	assert.Len(t, all, 14)
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"8080", 8080},
		{"  443  ", 443},
		{"3000", 3000},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		port := parsePort(tt.input)
		assert.Equal(t, tt.expected, port, "Input: %s", tt.input)
	}
}

func testTime() time.Time {
	return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
}
