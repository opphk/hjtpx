package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AdvancedFingerprintAnalysis struct {
	BaseFingerprint      *FingerprintAnalysis
	MLRiskScore          float64              `json:"ml_risk_score"`
	MLFeatures           *MLFeatures          `json:"ml_features"`
	ChainAnalysis        *ChainAnalysis       `json:"chain_analysis"`
	AdvancedIndicators   *AdvancedIndicators  `json:"advanced_indicators"`
	BehaviorScore        float64              `json:"behavior_score"`
	NetworkAnalysis      *NetworkAnalysis      `json:"network_analysis"`
	ConsistencyScore     float64              `json:"consistency_score"`
	EntropyScore         float64              `json:"entropy_score"`
}

type MLFeatures struct {
	TotalChecks       int       `json:"total_checks"`
	DetectedChecks    int       `json:"detected_checks"`
	AvgScore          float64   `json:"avg_score"`
	MaxScore          float64   `json:"max_score"`
	AutomationScore    float64   `json:"automation_score"`
	FingerprintScore   float64   `json:"fingerprint_score"`
	NetworkScore       float64   `json:"network_score"`
	SystemScore        float64   `json:"system_score"`
	VMScore            float64   `json:"vm_score"`
	SuspiciousPatterns []string  `json:"suspicious_patterns"`
	TimingVariance     float64   `json:"timing_variance"`
	EntropyScore       float64   `json:"entropy_score"`
	ConsistencyScore   float64   `json:"consistency_score"`
}

type ChainAnalysis struct {
	ChainLength       int                 `json:"chain_length"`
	ChainCategories   []string             `json:"chain_categories"`
	ChainResults      map[string]*ChainResult `json:"chain_results"`
	Duration          float64             `json:"duration_ms"`
	ProcessingOrder   []string             `json:"processing_order"`
}

type ChainResult struct {
	Detected     bool     `json:"detected"`
	Score        float64  `json:"score"`
	Duration     float64  `json:"duration_ms"`
	Detections   []string `json:"detections"`
	Error        string   `json:"error,omitempty"`
}

type AdvancedIndicators struct {
	HeadlessIndicators      []string `json:"headless_indicators"`
	AutomationIndicators    []string `json:"automation_indicators"`
	ProxyVPNIndicators      []string `json:"proxy_vpn_indicators"`
	VirtualizationIndicators []string `json:"virtualization_indicators"`
	VMIndicators            []string `json:"vm_indicators"`
	SandboxIndicators       []string `json:"sandbox_indicators"`
	BehavioralIndicators    []string `json:"behavioral_indicators"`
	NetworkIndicators       []string `json:"network_indicators"`
}

type NetworkAnalysis struct {
	WebRTCIPCount    int      `json:"webrtc_ip_count"`
	WebRTCLeakRisk   bool     `json:"webrtc_leak_risk"`
	ConnectionType   string   `json:"connection_type"`
	Latency          float64  `json:"latency"`
	IsProxy          bool     `json:"is_proxy"`
	IsVPN            bool     `json:"is_vpn"`
	IsTor            bool     `json:"is_tor"`
	DatacenterIP     bool     `json:"datacenter_ip"`
	MultiHopProxy    bool     `json:"multi_hop_proxy"`
	IPMismatch       bool     `json:"ip_mismatch"`
	HeadersPresent   []string `json:"headers_present"`
}

type AdvancedFingerprintAnalyzer struct {
	database        *FingerprintDatabase
	mlModel         *MockMLModel
	weights         map[string]float64
	knownBotPatterns map[string]*BotPattern
	knownVPNRanges  []*IPRange
	knownTorNodes   []*IPRange
}

type BotPattern struct {
	Pattern      string
	Weight       float64
	Category     string
	Description  string
}

type IPRange struct {
	Start string
	End   string
	Type  string
}

type MockMLModel struct {
	weights     map[string]float64
	threshold   float64
	trained     bool
}

func NewAdvancedFingerprintAnalyzer() *AdvancedFingerprintAnalyzer {
	analyzer := &AdvancedFingerprintAnalyzer{
		database:  NewFingerprintDatabase(),
		mlModel:   NewMockMLModel(),
		weights:   make(map[string]float64),
		knownBotPatterns: make(map[string]*BotPattern),
		knownVPNRanges:   make([]*IPRange, 0),
		knownTorNodes:    make([]*IPRange, 0),
	}

	analyzer.initWeights()
	analyzer.initBotPatterns()
	analyzer.initVPNRanges()
	analyzer.initTorNodes()

	return analyzer
}

func (a *AdvancedFingerprintAnalyzer) initWeights() {
	a.weights = map[string]float64{
		"canvas":                   14,
		"canvasStable":            12,
		"canvasEntropy":           10,
		"webgl":                   16,
		"webgl2":                  12,
		"webglVendor":             10,
		"webglRenderer":           10,
		"audio":                   13,
		"fonts":                   12,
		"fontEnumeration":         10,
		"fontMetrics":             8,
		"plugins":                 8,
		"pluginFingerprint":       7,
		"webrtc":                  17,
		"webrtcLeak":              14,
		"webdriver":                22,
		"selenium":                 20,
		"puppeteer":                20,
		"playwright":               20,
		"chromeRuntime":            12,
		"headless":                 17,
		"permissions":              8,
		"languages":                6,
		"timezone":                 7,
		"screen":                   5,
		"hardware":                 7,
		"memory":                   6,
		"storage":                  7,
		"navigator":                7,
		"windowProps":              6,
		"iframe":                   8,
		"notification":             4,
		"battery":                  5,
		"mediaDevices":             7,
		"connection":               10,
		"adblock":                  6,
		"math":                     5,
		"gpu":                      9,
		"speech":                   4,
		"proxyVPN":                 20,
		"torExitNode":              17,
		"vpnIndicators":            16,
		"virtualization":           14,
		"sandbox":                  12,
		"automationFrameworks":     18,
		"vmFeatures":               17,
		"sandboxEscape":            16,
		"debuggerDetection":        14,
		"advancedHeadless":         18,
		"stealthMode":              15,
		"browserProfile":           12,
		"timingAnomaly":            10,
		"networkFingerprint":       11,
		"behaviorPattern":          13,
	}
}

func (a *AdvancedFingerprintAnalyzer) initBotPatterns() {
	a.knownBotPatterns = map[string]*BotPattern{
		"headless":       {Pattern: "headless", Weight: 20, Category: "headless", Description: "Headless browser detected"},
		"phantom":        {Pattern: "phantom", Weight: 25, Category: "headless", Description: "PhantomJS detected"},
		"puppeteer":      {Pattern: "puppeteer", Weight: 22, Category: "automation", Description: "Puppeteer automation detected"},
		"playwright":     {Pattern: "playwright", Weight: 22, Category: "automation", Description: "Playwright automation detected"},
		"selenium":       {Pattern: "selenium", Weight: 20, Category: "automation", Description: "Selenium WebDriver detected"},
		"webdriver":      {Pattern: "webdriver", Weight: 23, Category: "automation", Description: "WebDriver property detected"},
		"chrome-headless": {Pattern: "chrome-headless", Weight: 18, Category: "headless", Description: "Chrome headless mode"},
		"firefox-headless": {Pattern: "firefox-headless", Weight: 18, Category: "headless", Description: "Firefox headless mode"},
		"__webdriver_evaluate": {Pattern: "__webdriver_evaluate", Weight: 25, Category: "automation", Description: "WebDriver evaluate function"},
		"$cdc_asdjflasutopfhvcZLmcfl_": {Pattern: "$cdc_asdjflasutopfhvcZLmcfl_", Weight: 28, Category: "puppeteer", Description: "Puppeteer marker detected"},
		"__playwright__": {Pattern: "__playwright__", Weight: 28, Category: "playwright", Description: "Playwright global marker"},
	}
}

func (a *AdvancedFingerprintAnalyzer) initVPNRanges() {
	a.knownVPNRanges = []*IPRange{
		{Start: "45.33.", End: "45.33.", Type: "VPN"},
		{Start: "104.238.", End: "104.238.", Type: "VPN"},
		{Start: "107.170.", End: "107.170.", Type: "VPN"},
		{Start: "142.4.", End: "142.4.", Type: "VPN"},
		{Start: "162.247.", End: "162.247.", Type: "VPN"},
	}
}

func (a *AdvancedFingerprintAnalyzer) initTorNodes() {
	a.knownTorNodes = []*IPRange{
		{Start: "128.31.0.", End: "128.31.0.", Type: "Tor"},
		{Start: "128.93.", End: "128.93.", Type: "Tor"},
		{Start: "131.188.", End: "131.188.", Type: "Tor"},
		{Start: "154.35.", End: "154.35.", Type: "Tor"},
		{Start: "171.25.193.", End: "171.25.193.", Type: "Tor"},
		{Start: "176.10.99.", End: "176.10.99.", Type: "Tor"},
	}
}

func NewMockMLModel() *MockMLModel {
	return &MockMLModel{
		weights:   make(map[string]float64),
		threshold: 0.75,
		trained:   true,
	}
}

func (m *MockMLModel) Predict(features map[string]float64) (float64, error) {
	score := 0.0
	for key, value := range features {
		weight := m.weights[key]
		if weight == 0 {
			weight = 1.0
		}
		score += value * weight
	}

	score = score / float64(len(features)+1)

	return math.Min(math.Max(score, 0), 100), nil
}

func (a *AdvancedFingerprintAnalyzer) AnalyzeAdvancedFingerprint(data map[string]interface{}) (*AdvancedFingerprintAnalysis, error) {
	analysis := &AdvancedFingerprintAnalysis{
		BaseFingerprint:    &FingerprintAnalysis{},
		MLFeatures:        &MLFeatures{},
		ChainAnalysis:     &ChainAnalysis{},
		AdvancedIndicators: &AdvancedIndicators{},
		NetworkAnalysis:   &NetworkAnalysis{},
	}

	a.extractBaseFingerprint(analysis, data)
	a.extractMLFeatures(analysis, data)
	a.extractChainAnalysis(analysis, data)
	a.extractAdvancedIndicators(analysis, data)
	a.extractNetworkAnalysis(analysis, data)
	a.calculateMLRiskScore(analysis)
	a.calculateBehaviorScore(analysis)
	a.calculateConsistencyScore(analysis)
	a.calculateEntropyScore(analysis)

	return analysis, nil
}

func (a *AdvancedFingerprintAnalyzer) extractBaseFingerprint(analysis *AdvancedFingerprintAnalysis, data map[string]interface{}) {
	fp := analysis.BaseFingerprint
	fp.FingerprintID = generateAdvancedFingerprintID(data)
	fp.CanvasHash = getString(data, "canvas_hash")
	fp.WebGLHash = getString(data, "webgl_hash")
	fp.AudioHash = getString(data, "audio_hash")
	fp.FontHash = getString(data, "font_hash")
	fp.PluginHash = getString(data, "plugin_hash")
	fp.UserAgent = getString(data, "user_agent")
	fp.ScreenResolution = getString(data, "screen_resolution")
	fp.Timezone = getString(data, "timezone")
	fp.Language = getString(data, "language")
	fp.Platform = getString(data, "platform")

	if hwConcurrency, ok := data["hardware_concurrency"].(float64); ok {
		fp.HardwareConcurrency = int(hwConcurrency)
	}
	if deviceMemory, ok := data["device_memory"].(float64); ok {
		fp.DeviceMemory = deviceMemory
	}

	fp.FirstSeen = time.Now()
	fp.LastSeen = time.Now()
	fp.RequestCount = 1

	a.detectAdvancedBotIndicators(fp, data)
	a.detectAdvancedVPNIndicators(fp, data)

	anomaly := a.database.DetectAnomaly(fp.FingerprintID)
	fp.AnomalyScore = anomaly.Score
	fp.RiskIndicators = anomaly.Indicators
	fp.Confidence = a.calculateAdvancedConfidence(fp)

	a.database.AddFingerprint(fp)
}

func (a *AdvancedFingerprintAnalyzer) extractMLFeatures(analysis *AdvancedFingerprintAnalysis, data map[string]interface{}) {
	ml := analysis.MLFeatures

	if chainResults, ok := data["chain_results"].(map[string]interface{}); ok {
		ml.TotalChecks = len(chainResults)
		for _, result := range chainResults {
			if resultMap, ok := result.(map[string]interface{}); ok {
				if detected, ok := resultMap["detected"].(bool); ok && detected {
					ml.DetectedChecks++
				}
				if score, ok := resultMap["score"].(float64); ok {
					ml.MaxScore = math.Max(ml.MaxScore, score)
					ml.AvgScore += score
				}
				if detections, ok := resultMap["detections"].([]interface{}); ok {
					for _, d := range detections {
						if detStr, ok := d.(string); ok {
							ml.SuspiciousPatterns = append(ml.SuspiciousPatterns, detStr)
						}
					}
				}
			}
		}
		if ml.TotalChecks > 0 {
			ml.AvgScore = ml.AvgScore / float64(ml.TotalChecks)
		}
	}

	if timingVariance, ok := data["timing_variance"].(float64); ok {
		ml.TimingVariance = timingVariance
	}
}

func (a *AdvancedFingerprintAnalyzer) extractChainAnalysis(analysis *AdvancedFingerprintAnalysis, data map[string]interface{}) {
	chain := analysis.ChainAnalysis

	if chainCategories, ok := data["chain_categories"].([]interface{}); ok {
		for _, cat := range chainCategories {
			if catStr, ok := cat.(string); ok {
				chain.ChainCategories = append(chain.ChainCategories, catStr)

				switch catStr {
				case "automation":
					analysis.MLFeatures.AutomationScore += 10
				case "fingerprint":
					analysis.MLFeatures.FingerprintScore += 10
				case "network":
					analysis.MLFeatures.NetworkScore += 10
				case "system":
					analysis.MLFeatures.SystemScore += 10
				case "vm":
					analysis.MLFeatures.VMScore += 10
				}
			}
		}
	}

	chain.ChainLength = len(chain.ChainCategories)

	if chainResults, ok := data["chain_results"].(map[string]interface{}); ok {
		chain.ChainResults = make(map[string]*ChainResult)
		for key, result := range chainResults {
			if resultMap, ok := result.(map[string]interface{}); ok {
				chainResult := &ChainResult{}
				if detected, ok := resultMap["detected"].(bool); ok {
					chainResult.Detected = detected
				}
				if score, ok := resultMap["score"].(float64); ok {
					chainResult.Score = score
				}
				if duration, ok := resultMap["duration_ms"].(float64); ok {
					chainResult.Duration = duration
				}
				if detections, ok := resultMap["detections"].([]interface{}); ok {
					for _, d := range detections {
						if detStr, ok := d.(string); ok {
							chainResult.Detections = append(chainResult.Detections, detStr)
						}
					}
				}
				chain.ChainResults[key] = chainResult
			}
		}
	}

	if duration, ok := data["duration_ms"].(float64); ok {
		chain.Duration = duration
	}
}

func (a *AdvancedFingerprintAnalyzer) extractAdvancedIndicators(analysis *AdvancedFingerprintAnalysis, data map[string]interface{}) {
	indicators := analysis.AdvancedIndicators

	headlessPatterns := []string{
		"headless", "phantom", "chrome-headless", "firefox-headless",
		"zero_outer_dimensions", "no_plugins", "no_languages",
		"canvas_no_content", "zero_window_size",
	}

	automationPatterns := []string{
		"webdriver", "selenium", "puppeteer", "playwright",
		"__webdriver_evaluate", "$cdc_asdjflasutopfhvcZLmcfl_",
		"__playwright__", "automation_framework",
	}

	proxyVPNPatterns := []string{
		"proxy_detected", "vpn_detected", "tor_exit_node", "ip_mismatch",
		"vpn_ip_mismatch", "multi_hop_proxy", "datacenter_ip",
	}

	virtualizationPatterns := []string{
		"vmware", "virtualbox", "parallels", "hyperv", "qemu", "kvm",
		"vm_webgl_renderer", "vm_renderer", "vm_single_core", "vm_low_memory",
	}

	vmPatterns := []string{
		"virtual_machine", "vm_detected", "virtualization_detected",
		"virtual_gpu", "virtual_cpu",
	}

	sandboxPatterns := []string{
		"sandbox_detected", "worker_blocked", "shared_array_buffer_unavailable",
	}

	behavioralPatterns := []string{
		"irregular_timing_patterns", "low_behavioral_entropy",
		"inconsistent_detection_patterns", "too_consistent_rendering",
	}

	networkPatterns := []string{
		"webrtc_ip_count", "ip_leak_detected", "relay_candidate",
		"slow_network_type", "high_latency", "network_fingerprint",
	}

	allPatterns := append(mlFeatures.SuspiciousPatterns, indicators.ExtractAllPatterns()...)

	for _, pattern := range headlessPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.HeadlessIndicators = append(indicators.HeadlessIndicators, p)
				break
			}
		}
	}

	for _, pattern := range automationPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.AutomationIndicators = append(indicators.AutomationIndicators, p)
				break
			}
		}
	}

	for _, pattern := range proxyVPNPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.ProxyVPNIndicators = append(indicators.ProxyVPNIndicators, p)
				break
			}
		}
	}

	for _, pattern := range virtualizationPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.VirtualizationIndicators = append(indicators.VirtualizationIndicators, p)
				break
			}
		}
	}

	for _, pattern := range vmPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.VMIndicators = append(indicators.VMIndicators, p)
				break
			}
		}
	}

	for _, pattern := range sandboxPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.SandboxIndicators = append(indicators.SandboxIndicators, p)
				break
			}
		}
	}

	for _, pattern := range behavioralPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.BehavioralIndicators = append(indicators.BehavioralIndicators, p)
				break
			}
		}
	}

	for _, pattern := range networkPatterns {
		for _, p := range allPatterns {
			if strings.Contains(strings.ToLower(p), pattern) {
				indicators.NetworkIndicators = append(indicators.NetworkIndicators, p)
				break
			}
		}
	}
}

func (i *AdvancedIndicators) ExtractAllPatterns() []string {
	all := make([]string, 0)
	all = append(all, i.HeadlessIndicators...)
	all = append(all, i.AutomationIndicators...)
	all = append(all, i.ProxyVPNIndicators...)
	all = append(all, i.VirtualizationIndicators...)
	all = append(all, i.VMIndicators...)
	all = append(all, i.SandboxIndicators...)
	all = append(all, i.BehavioralIndicators...)
	all = append(all, i.NetworkIndicators...)
	return all
}

func (a *AdvancedFingerprintAnalyzer) extractNetworkAnalysis(analysis *AdvancedFingerprintAnalysis, data map[string]interface{}) {
	network := analysis.NetworkAnalysis

	if webrtcIPs, ok := data["webrtc_ips"].([]interface{}); ok {
		network.WebRTCIPCount = len(webrtcIPs)
		network.WebRTCLeakRisk = network.WebRTCIPCount > 1

		for _, ip := range webrtcIPs {
			if ipStr, ok := ip.(string); ok {
				if a.isTorIP(ipStr) {
					network.IsTor = true
				} else if a.isVPNIP(ipStr) {
					network.IsVPN = true
				} else if a.isDatacenterIP(ipStr) {
					network.DatacenterIP = true
				}
			}
		}
	}

	if connectionType, ok := data["connection_type"].(string); ok {
		network.ConnectionType = connectionType
		network.IsVPN = network.ConnectionType == "vpn" || network.ConnectionType == "cellular"
	}

	if latency, ok := data["network_latency"].(float64); ok {
		network.Latency = latency
	}

	if xff, ok := data["x_forwarded_for"].(string); ok {
		count := len(strings.Split(xff, ","))
		if count > 2 {
			network.MultiHopProxy = true
		}
		network.HeadersPresent = append(network.HeadersPresent, "X-Forwarded-For")
	}

	if _, ok := data["x_real_ip"].(string); ok {
		network.HeadersPresent = append(network.HeadersPresent, "X-Real-IP")
	}

	if _, ok := data["via"].(string); ok {
		network.IsProxy = true
		network.HeadersPresent = append(network.HeadersPresent, "Via")
	}

	if _, ok := data["public_ip"].(string); ok {
		if network.IsTor || network.IsVPN || network.DatacenterIP {
			network.IPMismatch = true
		}
	}
}

func (a *AdvancedFingerprintAnalyzer) isTorIP(ip string) bool {
	for _, node := range a.knownTorNodes {
		if strings.HasPrefix(ip, node.Start) {
			return true
		}
	}
	return false
}

func (a *AdvancedFingerprintAnalyzer) isVPNIP(ip string) bool {
	for _, vpnRange := range a.knownVPNRanges {
		if strings.HasPrefix(ip, vpnRange.Start) {
			return true
		}
	}
	return false
}

func (a *AdvancedFingerprintAnalyzer) isDatacenterIP(ip string) bool {
	datacenterPrefixes := []string{
		"3.", "4.", "8.", "13.", "15.", "16.", "17.", "18.", "20.",
		"23.", "34.", "35.", "40.", "44.", "45.", "47.", "48.", "49.",
		"50.", "52.", "54.", "63.", "64.", "65.", "66.", "67.", "68.",
	}

	for _, prefix := range datacenterPrefixes {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}

	return false
}

func (a *AdvancedFingerprintAnalyzer) calculateMLRiskScore(analysis *AdvancedFingerprintAnalysis) {
	ml := analysis.MLFeatures

	mlScore := 0.0

	mlScore += float64(ml.DetectedChecks) * 5

	mlScore += math.Min(ml.AutomationScore*0.3, 30)
	mlScore += math.Min(ml.FingerprintScore*0.2, 20)
	mlScore += math.Min(ml.NetworkScore*0.25, 25)
	mlScore += math.Min(ml.VMScore*0.25, 20)

	if ml.TimingVariance > 0.8 {
		mlScore += 10
	}

	if ml.EntropyScore < 0.2 {
		mlScore += 8
	}

	if ml.ConsistencyScore < 0.7 {
		mlScore += 7
	}

	highRiskPatterns := []string{
		"headless", "webdriver", "puppeteer", "playwright", "selenium", "tor", "vpn", "proxy", "vm_", "sandbox",
	}

	for _, pattern := range highRiskPatterns {
		for _, suspPattern := range ml.SuspiciousPatterns {
			if strings.Contains(strings.ToLower(suspPattern), pattern) {
				mlScore += 3
				break
			}
		}
	}

	analysis.MLRiskScore = math.Min(math.Max(mlScore, 0), 100)
}

func (a *AdvancedFingerprintAnalyzer) calculateBehaviorScore(analysis *AdvancedFingerprintAnalysis) {
	score := 0.0

	if analysis.MLFeatures.TimingVariance > 0.5 {
		score += 20
	}

	if analysis.MLFeatures.EntropyScore < 0.15 {
		score += 25
	}

	if analysis.MLFeatures.ConsistencyScore < 0.6 {
		score += 20
	}

	if len(analysis.AdvancedIndicators.BehavioralIndicators) > 0 {
		score += float64(len(analysis.AdvancedIndicators.BehavioralIndicators)) * 8
	}

	analysis.BehaviorScore = math.Min(score, 100)
}

func (a *AdvancedFingerprintAnalyzer) calculateConsistencyScore(analysis *AdvancedFingerprintAnalysis) {
	score := 1.0

	canvasChecks := 0
	canvasConsistent := 0

	if analysis.ChainAnalysis.ChainResults != nil {
		for _, result := range analysis.ChainAnalysis.ChainResults {
			for k := range analysis.ChainAnalysis.ChainResults {
				if strings.Contains(strings.ToLower(k), "canvas") {
					canvasChecks++
					if result.Score < 30 {
						canvasConsistent++
					}
				}
			}
		}
	}

	if canvasChecks > 0 && canvasConsistent == canvasChecks {
		score *= 0.7
	}

	if analysis.AdvancedIndicators.HeadlessIndicators != nil &&
		analysis.AdvancedIndicators.AutomationIndicators != nil {
		if len(analysis.AdvancedIndicators.HeadlessIndicators) == 0 ||
			len(analysis.AdvancedIndicators.HeadlessIndicators) > 3 {
			score *= 0.8
		}
	}

	analysis.ConsistencyScore = score
}

func (a *AdvancedFingerprintAnalyzer) calculateEntropyScore(analysis *AdvancedFingerprintAnalysis) {
	score := 1.0

	totalPatterns := len(analysis.MLFeatures.SuspiciousPatterns)
	if totalPatterns == 0 {
		analysis.EntropyScore = score
		return
	}

	patternTypes := make(map[string]bool)
	for _, pattern := range analysis.MLFeatures.SuspiciousPatterns {
		for _, indicator := range analysis.AdvancedIndicators.ExtractAllPatterns() {
			if strings.Contains(pattern, indicator) {
				parts := strings.Split(indicator, "_")
				if len(parts) > 0 {
					patternTypes[parts[0]] = true
				}
			}
		}
	}

	if len(patternTypes) > 0 {
		score = float64(len(patternTypes)) / float64(totalPatterns)
	}

	analysis.EntropyScore = score
}

func (a *AdvancedFingerprintAnalyzer) detectAdvancedBotIndicators(fp *FingerprintAnalysis, data map[string]interface{}) {
	uaLower := strings.ToLower(fp.UserAgent)

	for _, bot := range a.knownBotPatterns {
		if strings.Contains(uaLower, bot.Pattern) {
			fp.RiskIndicators = append(fp.RiskIndicators, bot.Description)
			fp.IsKnownBot = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, bot.Weight)
		}
	}

	if webdriver, ok := data["navigator.webdriver"]; ok {
		if webdriverBool, ok := webdriver.(bool); ok && webdriverBool {
			fp.RiskIndicators = append(fp.RiskIndicators, "navigator.webdriver is true")
			fp.IsKnownBot = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 23)
		}
	}

	if _, ok := data["$cdc_asdjflasutopfhvcZLmcfl_"]; ok {
		fp.RiskIndicators = append(fp.RiskIndicators, "Puppeteer CDC marker detected")
		fp.IsKnownBot = true
		fp.AnomalyScore = math.Max(fp.AnomalyScore, 28)
	}

	if _, ok := data["__playwright__"]; ok {
		fp.RiskIndicators = append(fp.RiskIndicators, "Playwright global marker detected")
		fp.IsKnownBot = true
		fp.AnomalyScore = math.Max(fp.AnomalyScore, 28)
	}

	if plugins, ok := data["plugins_count"].(float64); ok {
		if plugins == 0 {
			fp.RiskIndicators = append(fp.RiskIndicators, "No plugins detected")
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 15)
		}
	}

	if languages, ok := data["languages_count"].(float64); ok {
		if languages == 0 {
			fp.RiskIndicators = append(fp.RiskIndicators, "No languages detected")
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 15)
		}
	}

	if outerWidth, ok := data["window.outerWidth"].(float64); ok {
		if outerWidth == 0 {
			fp.RiskIndicators = append(fp.RiskIndicators, "Zero outer width detected")
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 18)
		}
	}

	if webglRenderer, ok := data["webgl_renderer"].(string); ok {
		softwarePatterns := []string{"swiftshader", "llvmpipe", "mesa", "virtual", "software"}
		for _, pattern := range softwarePatterns {
			if strings.Contains(strings.ToLower(webglRenderer), pattern) {
				fp.RiskIndicators = append(fp.RiskIndicators, "Software WebGL renderer detected: "+pattern)
				fp.AnomalyScore = math.Max(fp.AnomalyScore, 25)
				break
			}
		}
	}
}

func (a *AdvancedFingerprintAnalyzer) detectAdvancedVPNIndicators(fp *FingerprintAnalysis, data map[string]interface{}) {
	if webrtcIPs, ok := data["webrtc_ips"].([]interface{}); ok {
		if len(webrtcIPs) == 0 {
			fp.RiskIndicators = append(fp.RiskIndicators, "No WebRTC IPs detected")
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 10)
		}

		privateIPs := 0
		publicIPs := 0
		for _, ip := range webrtcIPs {
			if ipStr, ok := ip.(string); ok {
				if isPrivateIP(ipStr) {
					privateIPs++
				} else {
					publicIPs++
				}
			}
		}

		if privateIPs > 0 && publicIPs > 0 {
			fp.RiskIndicators = append(fp.RiskIndicators, "VPN/TOR indicator: Private and public IPs both detected")
			fp.IsKnownVPN = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 30)
		}
	}

	if connectionType, ok := data["connection_type"].(string); ok {
		if connectionType == "vpn" || connectionType == "cellular" {
			fp.RiskIndicators = append(fp.RiskIndicators, "Connection type: "+connectionType)
			fp.IsKnownVPN = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 20)
		}
	}

	if publicIP, ok := data["public_ip"].(string); ok {
		if a.isVPNIP(publicIP) {
			fp.RiskIndicators = append(fp.RiskIndicators, "Known VPN IP range detected")
			fp.IsKnownVPN = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 25)
		}

		if a.isTorIP(publicIP) {
			fp.RiskIndicators = append(fp.RiskIndicators, "Known Tor exit node detected")
			fp.IsKnownVPN = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 30)
		}

		if a.isDatacenterIP(publicIP) {
			fp.RiskIndicators = append(fp.RiskIndicators, "Datacenter IP detected")
			fp.IsKnownVPN = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 20)
		}
	}

	if xff, ok := data["x_forwarded_for"].(string); ok {
		count := len(strings.Split(xff, ","))
		if count > 2 {
			fp.RiskIndicators = append(fp.RiskIndicators, "Multi-hop proxy detected")
			fp.IsKnownVPN = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 20)
		}
	}
}

func (a *AdvancedFingerprintAnalyzer) calculateAdvancedConfidence(fp *FingerprintAnalysis) float64 {
	fields := 0
	complete := 0

	checks := []struct {
		value    interface{}
		weight   float64
	}{
		{fp.CanvasHash, 20},
		{fp.WebGLHash, 20},
		{fp.AudioHash, 15},
		{fp.FontHash, 15},
		{fp.UserAgent, 15},
		{fp.ScreenResolution, 15},
	}

	for _, check := range checks {
		fields += 1
		if check.value != nil && check.value != "" {
			complete++
		}
	}

	if fields == 0 {
		return 0
	}

	return float64(complete) / float64(fields)
}

func generateAdvancedFingerprintID(data map[string]interface{}) string {
	hasher := sha256.New()

	fields := []string{
		"user_agent", "canvas_hash", "webgl_hash", "screen_resolution",
		"timezone", "webgl_vendor", "webgl_renderer", "audio_hash",
		"font_hash", "plugin_hash", "platform",
	}

	// Sort fields for consistent hashing
	sort.Strings(fields)

	for _, field := range fields {
		if val, ok := data[field].(string); ok && val != "" {
			hasher.Write([]byte(val))
		}
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)[:16] + fmt.Sprintf("_%d", rand.Intn(10000))
}

type EnhancedRiskScorer struct {
	weights     map[string]float64
	categories  map[string][]string
}

func NewEnhancedRiskScorer() *EnhancedRiskScorer {
	return &EnhancedRiskScorer{
		weights: map[string]float64{
			"automation":        2.0,
			"fingerprint":       1.5,
			"network":           1.8,
			"system":            1.0,
			"environment":       1.3,
			"vm":                1.6,
			"sandbox":           1.4,
			"debugger":          1.2,
			"behavior":          1.7,
		},
		categories: map[string][]string{
			"automation":    {"headless", "webdriver", "puppeteer", "playwright", "selenium"},
			"fingerprint":   {"canvas", "webgl", "audio", "fonts"},
			"network":       {"proxy", "vpn", "tor", "webrtc"},
			"vm":            {"virtualization", "vm_features"},
			"behavior":     {"timing", "behavior"},
		},
	}
}

func (s *EnhancedRiskScorer) CalculateScore(analysis *AdvancedFingerprintAnalysis) float64 {
	baseScore := analysis.BaseFingerprint.AnomalyScore

	automationMultiplier := 1.0
	if len(analysis.AdvancedIndicators.AutomationIndicators) >= 4 {
		automationMultiplier = 2.0
	} else if len(analysis.AdvancedIndicators.AutomationIndicators) >= 3 {
		automationMultiplier = 1.8
	} else if len(analysis.AdvancedIndicators.AutomationIndicators) >= 2 {
		automationMultiplier = 1.5
	} else if len(analysis.AdvancedIndicators.AutomationIndicators) >= 1 {
		automationMultiplier = 1.3
	}

	vpnMultiplier := 1.0
	vpnIndicators := len(analysis.AdvancedIndicators.ProxyVPNIndicators)
	if vpnIndicators >= 3 {
		vpnMultiplier = 1.6
	} else if vpnIndicators >= 2 {
		vpnMultiplier = 1.4
	} else if vpnIndicators >= 1 {
		vpnMultiplier = 1.2
	}

	vmMultiplier := 1.0
	if len(analysis.AdvancedIndicators.VirtualizationIndicators) >= 3 {
		vmMultiplier = 1.4
	} else if len(analysis.AdvancedIndicators.VirtualizationIndicators) >= 2 {
		vmMultiplier = 1.3
	}

	behaviorMultiplier := 1.0
	if analysis.BehaviorScore > 50 {
		behaviorMultiplier = 1.15
	}

	finalScore := baseScore * automationMultiplier * vpnMultiplier * vmMultiplier * behaviorMultiplier

	finalScore += analysis.MLRiskScore * 0.2

	return math.Min(math.Max(finalScore, 0), 100)
}

type PatternMatcher struct {
	patterns     []*CompiledPattern
	compiled     bool
}

type CompiledPattern struct {
	Regex        *regexp.Regexp
	Category     string
	Weight       float64
	Description  string
}

func NewPatternMatcher() *PatternMatcher {
	matcher := &PatternMatcher{
		patterns: make([]*CompiledPattern, 0),
		compiled: false,
	}

	matcher.initPatterns()

	return matcher
}

func (m *PatternMatcher) initPatterns() {
	patterns := []struct {
		pattern     string
		category    string
		weight      float64
		description string
	}{
		{`headless`, "automation", 20, "Headless browser detected"},
		{`phantom`, "automation", 25, "PhantomJS detected"},
		{`puppeteer`, "automation", 22, "Puppeteer detected"},
		{`playwright`, "automation", 22, "Playwright detected"},
		{`selenium`, "automation", 20, "Selenium detected"},
		{`webdriver`, "automation", 23, "WebDriver detected"},
		{`chrome-headless`, "automation", 18, "Chrome headless mode"},
		{`firefox-headless`, "automation", 18, "Firefox headless mode"},
		{`swiftshader|llvmpipe|mesa`, "fingerprint", 25, "Software renderer detected"},
		{`virtualbox|vmware|parallels`, "vm", 30, "Virtual machine detected"},
		{`tor|onion`, "network", 25, "Tor network detected"},
		{`vpn|proxy`, "network", 20, "VPN/Proxy detected"},
		{`no_plugins|no_plugins`, "system", 15, "No plugins"},
		{`no_languages`, "system", 15, "No languages"},
		{`zero_outer`, "environment", 18, "Zero outer dimensions"},
	}

	for _, p := range patterns {
		regex, err := regexp.Compile("(?i)" + p.pattern)
		if err == nil {
			m.patterns = append(m.patterns, &CompiledPattern{
				Regex:        regex,
				Category:     p.category,
				Weight:       p.weight,
				Description:  p.description,
			})
		}
	}

	m.compiled = true
}

func (m *PatternMatcher) Match(text string) []*MatchResult {
	results := make([]*MatchResult, 0)

	if !m.compiled {
		return results
	}

	for _, pattern := range m.patterns {
		if pattern.Regex.MatchString(text) {
			results = append(results, &MatchResult{
				Pattern:     pattern,
				MatchedText: pattern.Regex.FindString(text),
			})
		}
	}

	return results
}

type MatchResult struct {
	Pattern     *CompiledPattern
	MatchedText string
}

func (r *MatchResult) GetCategory() string {
	return r.Pattern.Category
}

func (r *MatchResult) GetWeight() float64 {
	return r.Pattern.Weight
}

func (r *MatchResult) GetDescription() string {
	return r.Pattern.Description
}

type RiskLevel int

const (
	RiskLevelLow RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

func (l RiskLevel) String() string {
	switch l {
	case RiskLevelLow:
		return "low"
	case RiskLevelMedium:
		return "medium"
	case RiskLevelHigh:
		return "high"
	case RiskLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func CalculateRiskLevel(score float64) RiskLevel {
	switch {
	case score >= 80:
		return RiskLevelCritical
	case score >= 60:
		return RiskLevelHigh
	case score >= 40:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

func (a *AdvancedFingerprintAnalyzer) GenerateRiskReport(analysis *AdvancedFingerprintAnalysis) *RiskReport {
	scorer := NewEnhancedRiskScorer()
	finalScore := scorer.CalculateScore(analysis)

	report := &RiskReport{
		Timestamp:       time.Now(),
		FinalScore:      finalScore,
		RiskLevel:       CalculateRiskLevel(finalScore),
		BaseScore:       analysis.BaseFingerprint.AnomalyScore,
		MLScore:         analysis.MLRiskScore,
		BehaviorScore:   analysis.BehaviorScore,
		ConsistencyScore: analysis.ConsistencyScore,
		EntropyScore:    analysis.EntropyScore,
		Indicators:      analysis.AdvancedIndicators.ExtractAllPatterns(),
		Categories:      make(map[string]int),
		Recommendations: make([]string, 0),
	}

	categoryCount := make(map[string]int)
	for _, indicator := range report.Indicators {
		matcher := NewPatternMatcher()
		matches := matcher.Match(indicator)
		for _, match := range matches {
			categoryCount[match.GetCategory()]++
		}
	}
	report.Categories = categoryCount

	if report.RiskLevel == RiskLevelCritical {
		report.Recommendations = append(report.Recommendations,
			"立即阻止访问并记录日志",
			"通知安全团队进行人工审查",
			"收集完整的环境信息用于分析",
		)
	} else if report.RiskLevel == RiskLevelHigh {
		report.Recommendations = append(report.Recommendations,
			"添加额外的验证步骤",
			"限制敏感操作",
			"增加监控频率",
		)
	} else if report.RiskLevel == RiskLevelMedium {
		report.Recommendations = append(report.Recommendations,
			"启用增强日志记录",
			"考虑添加验证码",
		)
	}

	return report
}

type RiskReport struct {
	Timestamp        time.Time            `json:"timestamp"`
	FinalScore       float64              `json:"final_score"`
	RiskLevel        RiskLevel            `json:"risk_level"`
	BaseScore        float64              `json:"base_score"`
	MLScore          float64              `json:"ml_score"`
	BehaviorScore    float64              `json:"behavior_score"`
	ConsistencyScore float64              `json:"consistency_score"`
	EntropyScore     float64              `json:"entropy_score"`
	Indicators       []string             `json:"indicators"`
	Categories       map[string]int       `json:"categories"`
	Recommendations  []string             `json:"recommendations"`
}

func (r *RiskReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r *RiskReport) GetSummary() string {
	return fmt.Sprintf("Risk Level: %s (Score: %.1f) - %d indicators across %d categories",
		r.RiskLevel.String(),
		r.FinalScore,
		len(r.Indicators),
		len(r.Categories),
	)
}

var mlFeatures *MLFeatures

func init() {
	mlFeatures = &MLFeatures{}
}

func extractMLFeaturesFromData(data map[string]interface{}) *MLFeatures {
	ml := &MLFeatures{}

	if chainResults, ok := data["chain_results"].(map[string]interface{}); ok {
		ml.TotalChecks = len(chainResults)
		for _, result := range chainResults {
			if resultMap, ok := result.(map[string]interface{}); ok {
				if detected, ok := resultMap["detected"].(bool); ok && detected {
					ml.DetectedChecks++
				}
				if score, ok := resultMap["score"].(float64); ok {
					ml.MaxScore = math.Max(ml.MaxScore, score)
					ml.AvgScore += score
				}
				if detections, ok := resultMap["detections"].([]interface{}); ok {
					for _, d := range detections {
						if detStr, ok := d.(string); ok {
							ml.SuspiciousPatterns = append(ml.SuspiciousPatterns, detStr)
						}
					}
				}
			}
		}
		if ml.TotalChecks > 0 {
			ml.AvgScore = ml.AvgScore / float64(ml.TotalChecks)
		}
	}

	if chainCategories, ok := data["chain_categories"].([]interface{}); ok {
		for _, cat := range chainCategories {
			if catStr, ok := cat.(string); ok {
				switch catStr {
				case "automation":
					ml.AutomationScore += 10
				case "fingerprint":
					ml.FingerprintScore += 10
				case "network":
					ml.NetworkScore += 10
				case "system":
					ml.SystemScore += 10
				case "vm":
					ml.VMScore += 10
				}
			}
		}
	}

	if timingVariance, ok := data["timing_variance"].(float64); ok {
		ml.TimingVariance = timingVariance
	}

	return ml
}

func calculateEntropy(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	frequency := make(map[float64]int)
	for _, val := range data {
		frequency[val]++
	}

	entropy := 0.0
	for _, count := range frequency {
		prob := float64(count) / float64(len(data))
		if prob > 0 {
			entropy -= prob * math.Log2(prob)
		}
	}

	return entropy
}

func (a *AdvancedFingerprintAnalyzer) AnalyzeTemporalPattern(data map[string]interface{}) *TemporalPatternAnalysis {
	analysis := &TemporalPatternAnalysis{}

	if timestamps, ok := data["request_timestamps"].([]interface{}); ok {
		analysis.RequestCount = len(timestamps)

		intervals := make([]float64, 0)
		for i := 1; i < len(timestamps); i++ {
			if t1, ok1 := timestamps[i-1].(float64); ok1 {
				if t2, ok2 := timestamps[i].(float64); ok2 {
					intervals = append(intervals, t2-t1)
				}
			}
		}

		if len(intervals) > 0 {
			analysis.AvgInterval = calculateAverage(intervals)
			analysis.MinInterval = calculateMin(intervals)
			analysis.MaxInterval = calculateMax(intervals)
			analysis.IntervalVariance = variance(intervals)

			if analysis.MinInterval < 0.5 {
				analysis.SuspiciousPattern = true
				analysis.PatternType = "high_frequency"
			}

			if analysis.IntervalVariance < 0.1 && analysis.RequestCount > 50 {
				analysis.SuspiciousPattern = true
				analysis.PatternType = "too_regular"
			}
		}
	}

	return analysis
}

type TemporalPatternAnalysis struct {
	RequestCount      int     `json:"request_count"`
	AvgInterval       float64 `json:"avg_interval"`
	MinInterval       float64 `json:"min_interval"`
	MaxInterval       float64 `json:"max_interval"`
	IntervalVariance  float64 `json:"interval_variance"`
	SuspiciousPattern bool    `json:"suspicious_pattern"`
	PatternType       string  `json:"pattern_type,omitempty"`
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := calculateAverage(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - avg) * (v - avg)
	}
	return sum / float64(len(values)-1)
}

func (a *AdvancedFingerprintAnalyzer) GetDatabase() *FingerprintDatabase {
	return a.database
}

func (a *AdvancedFingerprintAnalyzer) ExportAnalysis(analysis *AdvancedFingerprintAnalysis) ([]byte, error) {
	return json.MarshalIndent(analysis, "", "  ")
}

func (a *AdvancedFingerprintAnalyzer) ImportAnalysis(data []byte) (*AdvancedFingerprintAnalysis, error) {
	analysis := &AdvancedFingerprintAnalysis{}
	err := json.Unmarshal(data, analysis)
	return analysis, err
}

func parsePort(portStr string) int {
	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil {
		return 0
	}
	return port
}

type DetectionChain struct {
	ID          string
	Methods     []string
	Results     map[string]*ChainResult
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Score       float64
	Suspicious  bool
}

func NewDetectionChain(id string) *DetectionChain {
	return &DetectionChain{
		ID:        id,
		Methods:   make([]string, 0),
		Results:   make(map[string]*ChainResult),
		StartTime: time.Now(),
	}
}

func (c *DetectionChain) AddMethod(method string) {
	c.Methods = append(c.Methods, method)
}

func (c *DetectionChain) AddResult(method string, result *ChainResult) {
	c.Results[method] = result
}

func (c *DetectionChain) Complete() {
	c.EndTime = time.Now()
	c.Duration = c.EndTime.Sub(c.StartTime)
	c.calculateScore()
}

func (c *DetectionChain) calculateScore() {
	totalScore := 0.0
	methodCount := 0

	for _, result := range c.Results {
		if result != nil {
			totalScore += result.Score
			methodCount++
		}
	}

	if methodCount > 0 {
		c.Score = totalScore / float64(methodCount)
	}

	c.Suspicious = c.Score > 50 || len(c.getDetectedMethods()) >= 3
}

func (c *DetectionChain) getDetectedMethods() []string {
	detected := make([]string, 0)
	for method, result := range c.Results {
		if result != nil && result.Detected {
			detected = append(detected, method)
		}
	}
	return detected
}

func (c *DetectionChain) ToReport() *ChainReport {
	return &ChainReport{
		ID:           c.ID,
		Duration:     c.Duration,
		Score:        c.Score,
		Suspicious:   c.Suspicious,
		MethodCount:  len(c.Methods),
		DetectedCount: len(c.getDetectedMethods()),
		Methods:      c.Methods,
		Detected:     c.getDetectedMethods(),
	}
}

type ChainReport struct {
	ID            string    `json:"id"`
	Duration      time.Duration `json:"duration"`
	Score         float64   `json:"score"`
	Suspicious    bool      `json:"suspicious"`
	MethodCount   int       `json:"method_count"`
	DetectedCount int       `json:"detected_count"`
	Methods       []string  `json:"methods"`
	Detected      []string  `json:"detected_methods"`
}

func (c *DetectionChain) ExportJSON() ([]byte, error) {
	return json.MarshalIndent(c.ToReport(), "", "  ")
}

type AdvancedFingerprintDatabase struct {
	*FingerprintDatabase
	advancedData map[string]*AdvancedFingerprintAnalysis
	mu           sync.RWMutex
}

func NewAdvancedFingerprintDatabase() *AdvancedFingerprintDatabase {
	return &AdvancedFingerprintDatabase{
		FingerprintDatabase: NewFingerprintDatabase(),
		advancedData:         make(map[string]*AdvancedFingerprintAnalysis),
	}
}

func (db *AdvancedFingerprintDatabase) AddAdvancedAnalysis(id string, analysis *AdvancedFingerprintAnalysis) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.advancedData[id] = analysis
}

func (db *AdvancedFingerprintDatabase) GetAdvancedAnalysis(id string) (*AdvancedFingerprintAnalysis, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	analysis, exists := db.advancedData[id]
	return analysis, exists
}

func (db *AdvancedFingerprintDatabase) GetAllAdvancedAnalyses() []*AdvancedFingerprintAnalysis {
	db.mu.RLock()
	defer db.mu.RUnlock()

	analyses := make([]*AdvancedFingerprintAnalysis, 0, len(db.advancedData))
	for _, analysis := range db.advancedData {
		analyses = append(analyses, analysis)
	}

	return analyses
}

func (db *AdvancedFingerprintDatabase) CleanupOldAdvancedData(maxAge time.Duration) int {
	db.mu.Lock()
	defer db.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, analysis := range db.advancedData {
		if analysis.BaseFingerprint != nil {
			if analysis.BaseFingerprint.LastSeen.Before(cutoff) &&
				analysis.BaseFingerprint.RequestCount < 5 {
				delete(db.advancedData, id)
				removed++
			}
		}
	}

	return removed
}

func (db *AdvancedFingerprintDatabase) ExportAll() ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	data := map[string]interface{}{
		"fingerprints":  db.FingerprintDatabase.fingerprints,
		"advancedData":  db.advancedData,
		"stats":         db.FingerprintDatabase.stats,
		"exported_at":   time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

func (db *AdvancedFingerprintDatabase) ImportAll(data []byte) error {
	type ImportData struct {
		Fingerprints  map[string]*FingerprintAnalysis    `json:"fingerprints"`
		AdvancedData  map[string]*AdvancedFingerprintAnalysis `json:"advancedData"`
	}

	var importData ImportData
	if err := json.Unmarshal(data, &importData); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	for id, fp := range importData.Fingerprints {
		fp.FingerprintID = id
		db.fingerprints[id] = fp
	}

	for id, analysis := range importData.AdvancedData {
		db.advancedData[id] = analysis
	}

	return nil
}

func (db *AdvancedFingerprintDatabase) GetRiskDistribution() map[string]int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	distribution := map[string]int{
		"low":      0,
		"medium":   0,
		"high":     0,
		"critical": 0,
	}

	for _, analysis := range db.advancedData {
		level := CalculateRiskLevel(analysis.BaseFingerprint.AnomalyScore)
		distribution[level.String()]++
	}

	return distribution
}

func (db *AdvancedFingerprintDatabase) GetTopRiskFactors(n int) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	factorCounts := make(map[string]int)

	for _, analysis := range db.advancedData {
		for _, indicator := range analysis.AdvancedIndicators.ExtractAllPatterns() {
			factorCounts[indicator]++
		}
	}

	factors := make([]string, 0, len(factorCounts))
	for factor := range factorCounts {
		factors = append(factors, factor)
	}

	sort.Slice(factors, func(i, j int) bool {
		return factorCounts[factors[i]] > factorCounts[factors[j]]
	})

	if len(factors) > n {
		factors = factors[:n]
	}

	return factors
}

func (db *AdvancedFingerprintDatabase) GetAnalytics() *DatabaseAnalytics {
	db.mu.RLock()
	defer db.mu.RUnlock()

	analytics := &DatabaseAnalytics{
		TotalFingerprints:  len(db.fingerprints),
		TotalAnalyses:      len(db.advancedData),
		BotCount:           0,
		VPNCount:           0,
		RiskDistribution:   db.GetRiskDistribution(),
		TopRiskFactors:     db.GetTopRiskFactors(10),
	}

	for _, fp := range db.fingerprints {
		if fp.IsKnownBot {
			analytics.BotCount++
		}
		if fp.IsKnownVPN {
			analytics.VPNCount++
		}
	}

	var totalScore float64
	for _, analysis := range db.advancedData {
		totalScore += analysis.BaseFingerprint.AnomalyScore
	}

	if analytics.TotalAnalyses > 0 {
		analytics.AvgRiskScore = totalScore / float64(analytics.TotalAnalyses)
	}

	return analytics
}

type DatabaseAnalytics struct {
	TotalFingerprints int              `json:"total_fingerprints"`
	TotalAnalyses     int              `json:"total_analyses"`
	BotCount          int              `json:"bot_count"`
	VPNCount          int              `json:"vpn_count"`
	AvgRiskScore      float64          `json:"avg_risk_score"`
	RiskDistribution  map[string]int   `json:"risk_distribution"`
	TopRiskFactors    []string         `json:"top_risk_factors"`
}

func (a *DatabaseAnalytics) ToJSON() ([]byte, error) {
	return json.MarshalIndent(a, "", "  ")
}
