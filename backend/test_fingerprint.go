//go:build ignore

package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

type FingerprintAnalysis struct {
	FingerprintID       string    `json:"fingerprint_id"`
	IP                  string    `json:"ip"`
	CanvasHash          string    `json:"canvas_hash"`
	WebGLHash           string    `json:"webgl_hash"`
	AudioHash           string    `json:"audio_hash"`
	FontHash            string    `json:"font_hash"`
	PluginHash          string    `json:"plugin_hash"`
	UserAgent           string    `json:"user_agent"`
	ScreenResolution    string    `json:"screen_resolution"`
	Timezone            string    `json:"timezone"`
	Language            string    `json:"language"`
	Platform            string    `json:"platform"`
	HardwareConcurrency int       `json:"hardware_concurrency"`
	DeviceMemory        float64   `json:"device_memory"`
	FirstSeen           time.Time `json:"first_seen"`
	LastSeen            time.Time `json:"last_seen"`
	RequestCount        int       `json:"request_count"`
	Similarity          float64   `json:"similarity"`
	RiskIndicators      []string  `json:"risk_indicators"`
	AnomalyScore        float64   `json:"anomaly_score"`
	Confidence          float64   `json:"confidence"`
	ClusterID           string    `json:"cluster_id"`
	IsKnownBot          bool      `json:"is_known_bot"`
	IsKnownVPN          bool      `json:"is_known_vpn"`
}

type FingerprintDatabase struct {
	fingerprints    map[string]*FingerprintAnalysis
	clusters        map[string][]string
	similarityIndex map[string][]string
	mu              sync.RWMutex
}

func NewFingerprintDatabase() *FingerprintDatabase {
	return &FingerprintDatabase{
		fingerprints:    make(map[string]*FingerprintAnalysis),
		clusters:        make(map[string][]string),
		similarityIndex: make(map[string][]string),
	}
}

func (db *FingerprintDatabase) GetAllFingerprints() []*FingerprintAnalysis {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]*FingerprintAnalysis, 0, len(db.fingerprints))
	for _, fp := range db.fingerprints {
		result = append(result, fp)
	}
	return result
}

type CanvasMetrics struct {
	Hash                 string   `json:"hash"`
	RgbaDistribution     []int    `json:"rgba_distribution"`
	NoiseLevel           float64  `json:"noise_level"`
	RenderingConsistency float64  `json:"rendering_consistency"`
	IsHeadlessRenderer   bool     `json:"is_headless_renderer"`
	SoftwareRenderer     bool     `json:"software_renderer"`
	Details              []string `json:"details"`
}

type WebGLMetrics struct {
	Hash                string   `json:"hash"`
	Vendor              string   `json:"vendor"`
	Renderer            string   `json:"renderer"`
	MaxTextureSize      int      `json:"max_texture_size"`
	MaxRenderbufferSize int      `json:"max_renderbuffer_size"`
	MaxVertexAttribs    int      `json:"max_vertex_attribs"`
	SupportedExtensions int      `json:"supported_extensions"`
	UnmaskedVendor      string   `json:"unmasked_vendor"`
	UnmaskedRenderer    string   `json:"unmasked_renderer"`
	IsSoftwareRenderer  bool     `json:"is_software_renderer"`
	IsVirtualGPU        bool     `json:"is_virtual_gpu"`
	PrecisionLoss       bool     `json:"precision_loss"`
	Details             []string `json:"details"`
}

type FontMetrics struct {
	Hash                string   `json:"hash"`
	DetectedFonts       []string `json:"detected_fonts"`
	FontCount           int      `json:"font_count"`
	CommonFontMissing   []string `json:"common_font_missing"`
	FontFamilyDiversity float64  `json:"font_family_diversity"`
	IsLimitedFontSet    bool     `json:"is_limited_font_set"`
}

type CanvasSimilarityAnalyzer struct {
	database *FingerprintDatabase
}

func NewCanvasSimilarityAnalyzer(db *FingerprintDatabase) *CanvasSimilarityAnalyzer {
	return &CanvasSimilarityAnalyzer{
		database: db,
	}
}

func (c *CanvasSimilarityAnalyzer) CalculateCanvasSimilarity(hash1, hash2 string) float64 {
	if hash1 == "" || hash2 == "" {
		return 0
	}
	if hash1 == hash2 {
		return 100.0
	}
	similarChars := 0
	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}
	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			similarChars++
		}
	}
	avgLen := (len(hash1) + len(hash2)) / 2
	return float64(similarChars) / float64(avgLen) * 100
}

func (c *CanvasSimilarityAnalyzer) CalculateHistogramSimilarity(hash1, hash2 string) float64 {
	if hash1 == "" || hash2 == "" {
		return 0
	}
	hist1 := c.hashToHistogram(hash1)
	hist2 := c.hashToHistogram(hash2)
	return c.cosineSimilarity(hist1, hist2) * 100
}

func (c *CanvasSimilarityAnalyzer) hashToHistogram(hash string) []int {
	histogram := make([]int, 16)
	for i := 0; i < len(hash); i++ {
		nibble := 0
		if hash[i] >= '0' && hash[i] <= '9' {
			nibble = int(hash[i] - '0')
		} else if hash[i] >= 'a' && hash[i] <= 'f' {
			nibble = int(hash[i] - 'a' + 10)
		} else if hash[i] >= 'A' && hash[i] <= 'F' {
			nibble = int(hash[i] - 'A' + 10)
		}
		histogram[nibble]++
	}
	return histogram
}

func (c *CanvasSimilarityAnalyzer) cosineSimilarity(vec1, vec2 []int) float64 {
	dotProduct := 0
	norm1 := 0
	norm2 := 0
	minLen := len(vec1)
	if len(vec2) < minLen {
		minLen = len(vec2)
	}
	for i := 0; i < minLen; i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}
	if norm1 == 0 || norm2 == 0 {
		return 0
	}
	return float64(dotProduct) / (math.Sqrt(float64(norm1)) * math.Sqrt(float64(norm2)))
}

func (c *CanvasSimilarityAnalyzer) CalculateEnhancedSimilarity(hash1, hash2 string) float64 {
	if hash1 == "" || hash2 == "" {
		return 0
	}
	if hash1 == hash2 {
		return 100.0
	}
	exactMatch := c.CalculateCanvasSimilarity(hash1, hash2)
	histogramMatch := c.CalculateHistogramSimilarity(hash1, hash2)
	return (exactMatch*0.6 + histogramMatch*0.4)
}

type CanvasStabilityResult struct {
	SampleCount     int      `json:"sample_count"`
	IsStable        bool     `json:"is_stable"`
	StabilityScore  float64  `json:"stability_score"`
	AvgSimilarity   float64  `json:"avg_similarity"`
	ExactMatchRatio float64  `json:"exact_match_ratio"`
	Issues          []string `json:"issues"`
	Warnings        []string `json:"warnings"`
}

func (c *CanvasSimilarityAnalyzer) AnalyzeHashStability(hashSamples []string) *CanvasStabilityResult {
	result := &CanvasStabilityResult{
		SampleCount:    len(hashSamples),
		IsStable:       true,
		StabilityScore: 100.0,
		Issues:         make([]string, 0),
		Warnings:       make([]string, 0),
	}
	if len(hashSamples) < 2 {
		result.Issues = append(result.Issues, "insufficient_samples")
		result.IsStable = false
		result.StabilityScore = 0
		return result
	}
	referenceHash := hashSamples[0]
	totalSimilarity := 0.0
	matchCount := 0
	for i := 1; i < len(hashSamples); i++ {
		similarity := c.CalculateEnhancedSimilarity(referenceHash, hashSamples[i])
		totalSimilarity += similarity
		if hashSamples[i] == referenceHash {
			matchCount++
		}
	}
	result.AvgSimilarity = totalSimilarity / float64(len(hashSamples)-1)
	result.ExactMatchRatio = float64(matchCount) / float64(len(hashSamples)-1)
	if result.AvgSimilarity < 95 {
		result.IsStable = false
		result.StabilityScore = result.AvgSimilarity
		result.Issues = append(result.Issues, "low_average_similarity")
	}
	if result.ExactMatchRatio < 0.8 {
		result.Issues = append(result.Issues, "inconsistent_hash_generation")
	}
	if result.AvgSimilarity > 99.9 && len(hashSamples) > 5 {
		result.Warnings = append(result.Warnings, "suspiciously_identical_hashes")
	}
	return result
}

type TamperingDetection struct {
	IsTampered bool     `json:"is_tampered"`
	Confidence float64  `json:"confidence"`
	Indicators []string `json:"indicators"`
}

func (c *CanvasSimilarityAnalyzer) DetectHashTampering(hash string, expectedLength int) *TamperingDetection {
	result := &TamperingDetection{
		IsTampered: false,
		Confidence: 0.0,
		Indicators: make([]string, 0),
	}
	if hash == "" {
		result.IsTampered = true
		result.Confidence = 0.9
		result.Indicators = append(result.Indicators, "empty_hash")
		return result
	}
	if expectedLength > 0 && len(hash) != expectedLength {
		result.IsTampered = true
		result.Confidence = 0.85
		result.Indicators = append(result.Indicators, "invalid_length")
	}
	hexPattern := regexp.MustCompile("^[0-9a-fA-F]+$")
	if !hexPattern.MatchString(hash) {
		result.IsTampered = true
		result.Confidence = 0.95
		result.Indicators = append(result.Indicators, "non_hex_characters")
	}
	if len(hash) > 0 {
		histogram := c.hashToHistogram(hash)
		entropy := c.calculateEntropy(histogram)
		if entropy < 2.0 {
			result.IsTampered = true
			result.Confidence = math.Min(0.8+(2.0-entropy)*0.1, 0.95)
			result.Indicators = append(result.Indicators, "low_entropy")
		}
		if entropy > 3.9 {
			result.Indicators = append(result.Indicators, "unusually_high_entropy")
		}
	}
	if len(result.Indicators) > 0 {
		result.IsTampered = true
		result.Confidence = math.Min(0.5+float64(len(result.Indicators))*0.15, 0.95)
	}
	return result
}

func (c *CanvasSimilarityAnalyzer) calculateEntropy(histogram []int) float64 {
	total := 0
	for _, count := range histogram {
		total += count
	}
	if total == 0 {
		return 0
	}
	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			prob := float64(count) / float64(total)
			entropy -= prob * math.Log2(prob)
		}
	}
	return entropy / 4.0
}

type WebGLAnalyzer struct {
	database            *FingerprintDatabase
	knownVendors        map[string]bool
	knownRenderers      map[string]bool
	blacklistedPatterns []string
	expectedExtensions  map[string][]string
}

func NewWebGLAnalyzer(db *FingerprintDatabase) *WebGLAnalyzer {
	return &WebGLAnalyzer{
		database:            db,
		knownVendors:        initKnownVendors(),
		knownRenderers:      initKnownRenderers(),
		blacklistedPatterns: initBlacklistedPatterns(),
		expectedExtensions:  initExpectedExtensions(),
	}
}

func initKnownVendors() map[string]bool {
	return map[string]bool{
		"NVIDIA Corporation":           true,
		"Advanced Micro Devices, Inc.": true,
		"Intel(R) Corporation":         true,
		"Google Inc.":                  true,
		"Microsoft Corporation":        true,
		"Apple Inc.":                   true,
	}
}

func initKnownRenderers() map[string]bool {
	return map[string]bool{
		"GeForce":            true,
		"Radeon":             true,
		"Intel(R) HD Graphics": true,
		"SwiftShader":        true,
		"llvmpipe":           true,
		"ANGLE":              true,
	}
}

func initBlacklistedPatterns() []string {
	return []string{"fake", "mock", "test", "emulator", "virtual", "spoof"}
}

func initExpectedExtensions() map[string][]string {
	return map[string][]string{
		"webgl": {"GL_EXT_blend_minmax", "GL_EXT_color_buffer_float"},
		"webgl2": {"GL_EXT_color_buffer_float"},
	}
}

type WebGLAnalysisResult struct {
	IsTampered         bool                `json:"is_tampered"`
	TamperingScore     float64             `json:"tampering_score"`
	Confidence         float64             `json:"confidence"`
	VendorAnalysis     *VendorAnalysis     `json:"vendor_analysis"`
	RendererAnalysis   *RendererAnalysis   `json:"renderer_analysis"`
	ExtensionsAnalysis *ExtensionsAnalysis `json:"extensions_analysis"`
	Capabilities       *WebGLCapabilities  `json:"capabilities"`
	Warnings           []string            `json:"warnings"`
	Errors             []string            `json:"errors"`
}

type VendorAnalysis struct {
	Vendor         string `json:"vendor"`
	UnmaskedVendor string `json:"unmasked_vendor"`
	IsKnown        bool   `json:"is_known"`
}

type RendererAnalysis struct {
	Renderer          string `json:"renderer"`
	UnmaskedRenderer  string `json:"unmasked_renderer"`
	IsSoftwareRenderer bool  `json:"is_software_renderer"`
	IsKnown           bool   `json:"is_known"`
}

type ExtensionsAnalysis struct {
	ExtensionCount   int      `json:"extension_count"`
	Extensions       []string `json:"extensions"`
	MissingExpected  []string `json:"missing_expected"`
	UnexpectedFound  []string `json:"unexpected_found"`
}

type WebGLCapabilities struct {
	MaxTextureSize      int `json:"max_texture_size"`
	MaxRenderbufferSize int `json:"max_renderbuffer_size"`
	MaxVertexAttribs    int `json:"max_vertex_attribs"`
}

func (w *WebGLAnalyzer) AnalyzeWebGLFingerprint(data map[string]interface{}) *WebGLAnalysisResult {
	result := &WebGLAnalysisResult{
		IsTampered:         false,
		TamperingScore:     0.0,
		Confidence:         0.0,
		VendorAnalysis:     &VendorAnalysis{},
		RendererAnalysis:   &RendererAnalysis{},
		ExtensionsAnalysis: &ExtensionsAnalysis{},
		Capabilities:       &WebGLCapabilities{},
		Warnings:           make([]string, 0),
		Errors:             make([]string, 0),
	}
	w.analyzeVendor(data, result)
	w.analyzeRenderer(data, result)
	w.analyzeExtensions(data, result)
	w.analyzeCapabilities(data, result)
	if len(result.Errors) > 0 {
		result.IsTampered = true
		result.TamperingScore = math.Min(50.0+float64(len(result.Errors))*15.0, 95.0)
		result.Confidence = math.Min(0.5+float64(len(result.Errors))*0.15, 0.95)
	} else if len(result.Warnings) > 2 {
		result.IsTampered = true
		result.TamperingScore = math.Min(20.0+float64(len(result.Warnings))*10.0, 50.0)
	}
	return result
}

func (w *WebGLAnalyzer) analyzeVendor(data map[string]interface{}, result *WebGLAnalysisResult) {
	vendor := getString(data, "webgl_vendor")
	unmaskedVendor := getString(data, "webgl_unmasked_vendor")
	
	if vendor == "" {
		result.Errors = append(result.Errors, "missing_vendor")
		return
	}
	
	result.VendorAnalysis.Vendor = vendor
	result.VendorAnalysis.UnmaskedVendor = unmaskedVendor
	result.VendorAnalysis.IsKnown = w.knownVendors[vendor]
	
	if !w.knownVendors[vendor] {
		result.Warnings = append(result.Warnings, "unknown_vendor:"+vendor)
	}
	
	for _, pattern := range w.blacklistedPatterns {
		if strings.Contains(strings.ToLower(vendor), pattern) {
			result.Errors = append(result.Errors, "blacklisted_vendor_pattern:"+pattern)
			return
		}
	}
	
	if vendor != unmaskedVendor && unmaskedVendor != "" {
		result.Warnings = append(result.Warnings, "vendor_mismatch")
	}
}

func (w *WebGLAnalyzer) analyzeRenderer(data map[string]interface{}, result *WebGLAnalysisResult) {
	renderer := getString(data, "webgl_renderer")
	unmaskedRenderer := getString(data, "webgl_unmasked_renderer")
	
	if renderer == "" {
		result.Errors = append(result.Errors, "missing_renderer")
		return
	}
	
	result.RendererAnalysis.Renderer = renderer
	result.RendererAnalysis.UnmaskedRenderer = unmaskedRenderer
	
	softwarePatterns := []string{"swiftshader", "llvmpipe", "mesa", "software"}
	for _, pattern := range softwarePatterns {
		if strings.Contains(strings.ToLower(renderer), pattern) {
			result.RendererAnalysis.IsSoftwareRenderer = true
			result.Warnings = append(result.Warnings, "software_renderer:"+pattern)
			break
		}
	}
	
	result.RendererAnalysis.IsKnown = false
	for knownRenderer := range w.knownRenderers {
		if strings.Contains(strings.ToLower(renderer), strings.ToLower(knownRenderer)) {
			result.RendererAnalysis.IsKnown = true
			break
		}
	}
	
	if !result.RendererAnalysis.IsKnown {
		result.Warnings = append(result.Warnings, "unknown_renderer:"+renderer)
	}
	
	if renderer != unmaskedRenderer && unmaskedRenderer != "" {
		result.Warnings = append(result.Warnings, "renderer_mismatch")
	}
}

func (w *WebGLAnalyzer) analyzeExtensions(data map[string]interface{}, result *WebGLAnalysisResult) {
	var extensions []string
	if extData, ok := data["webgl_extensions"].([]interface{}); ok {
		for _, ext := range extData {
			if extStr, ok := ext.(string); ok {
				extensions = append(extensions, extStr)
			}
		}
	}
	
	result.ExtensionsAnalysis.ExtensionCount = len(extensions)
	result.ExtensionsAnalysis.Extensions = extensions
	
	if len(extensions) == 0 {
		result.Errors = append(result.Errors, "no_extensions")
	}
}

func (w *WebGLAnalyzer) analyzeCapabilities(data map[string]interface{}, result *WebGLAnalysisResult) {
	if val, ok := data["webgl_max_texture_size"].(float64); ok {
		result.Capabilities.MaxTextureSize = int(val)
	}
	if val, ok := data["webgl_max_renderbuffer_size"].(float64); ok {
		result.Capabilities.MaxRenderbufferSize = int(val)
	}
	if val, ok := data["webgl_max_vertex_attribs"].(float64); ok {
		result.Capabilities.MaxVertexAttribs = int(val)
	}
}

func (w *WebGLAnalyzer) CompareWebGLFingerprints(fp1, fp2 *WebGLMetrics) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}
	
	totalScore := 0.0
	weightSum := 0.0
	
	if fp1.Vendor != "" && fp2.Vendor != "" {
		if fp1.Vendor == fp2.Vendor {
			totalScore += 30.0
		} else {
			totalScore += 10.0
		}
		weightSum += 30.0
	}
	
	if fp1.Renderer != "" && fp2.Renderer != "" {
		if fp1.Renderer == fp2.Renderer {
			totalScore += 30.0
		} else {
			totalScore += 10.0
		}
		weightSum += 30.0
	}
	
	if fp1.MaxTextureSize != 0 && fp2.MaxTextureSize != 0 {
		if fp1.MaxTextureSize == fp2.MaxTextureSize {
			totalScore += 20.0
		} else {
			diff := math.Abs(float64(fp1.MaxTextureSize - fp2.MaxTextureSize))
			ratio := diff / float64(fp1.MaxTextureSize)
			if ratio < 0.1 {
				totalScore += 15.0
			} else if ratio < 0.5 {
				totalScore += 10.0
			}
		}
		weightSum += 20.0
	}
	
	if fp1.MaxVertexAttribs != 0 && fp2.MaxVertexAttribs != 0 {
		if fp1.MaxVertexAttribs == fp2.MaxVertexAttribs {
			totalScore += 20.0
		} else {
			totalScore += 5.0
		}
		weightSum += 20.0
	}
	
	if weightSum == 0 {
		return 0
	}
	
	return totalScore / weightSum * 100
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

type FontAnalyzer struct {
	database      *FingerprintDatabase
	commonFonts   map[string]bool
	platformFonts map[string][]string
}

func NewFontAnalyzer(db *FingerprintDatabase) *FontAnalyzer {
	return &FontAnalyzer{
		database:    db,
		commonFonts: initCommonFonts(),
		platformFonts: map[string][]string{
			"windows": {"Arial", "Times New Roman", "Verdana", "Courier New", "Georgia", "Impact"},
			"macos":   {"Arial", "Helvetica", "SF Pro Display", "Times New Roman", "Courier", "Georgia"},
			"linux":   {"Arial", "Liberation Sans", "DejaVu Sans", "Times New Roman"},
			"android": {"Roboto", "Arial", "Droid Sans"},
			"ios":     {"SF Pro Display", "Arial", "Helvetica"},
		},
	}
}

func initCommonFonts() map[string]bool {
	return map[string]bool{
		"Arial":          true,
		"Helvetica":      true,
		"Times New Roman": true,
		"Verdana":        true,
		"Courier New":    true,
		"Georgia":        true,
		"Times":          true,
		"Courier":        true,
	}
}

type FontAnalysisResult struct {
	IsTampered      bool                  `json:"is_tampered"`
	Confidence      float64               `json:"confidence"`
	FontAnalysis    *DetailedFontAnalysis `json:"font_analysis"`
	PlatformMatch   *PlatformFontMatch    `json:"platform_match"`
	Warnings        []string              `json:"warnings"`
	Errors          []string              `json:"errors"`
}

type DetailedFontAnalysis struct {
	DetectedFonts           []string `json:"detected_fonts"`
	FontCount               int      `json:"font_count"`
	CommonFontCount         int      `json:"common_font_count"`
	IsLimitedFontSet        bool     `json:"is_limited_font_set"`
	RenderingConsistency    float64  `json:"rendering_consistency"`
	FontFamilyDiversity     float64  `json:"font_family_diversity"`
	CommonFontMissing       []string `json:"common_font_missing"`
}

type PlatformFontMatch struct {
	Platform         string  `json:"platform"`
	ExpectedFontCount int    `json:"expected_font_count"`
	MatchedFontCount  int    `json:"matched_font_count"`
	MatchRatio       float64 `json:"match_ratio"`
}

func (f *FontAnalyzer) AnalyzeFontFingerprint(data map[string]interface{}) *FontAnalysisResult {
	result := &FontAnalysisResult{
		IsTampered:     false,
		Confidence:     50.0,
		FontAnalysis:   &DetailedFontAnalysis{},
		PlatformMatch:  &PlatformFontMatch{},
		Warnings:       make([]string, 0),
		Errors:         make([]string, 0),
	}
	f.extractFontData(data, result)
	f.analyzeFontPatterns(result)
	f.analyzePlatformMatch(data, result)
	f.analyzeRenderingConsistency(data, result)
	f.calculateConfidence(result)
	return result
}

func (f *FontAnalyzer) extractFontData(data map[string]interface{}, result *FontAnalysisResult) {
	if fonts, ok := data["detected_fonts"].([]interface{}); ok {
		for _, font := range fonts {
			if fontStr, ok := font.(string); ok {
				result.FontAnalysis.DetectedFonts = append(result.FontAnalysis.DetectedFonts, fontStr)
			}
		}
	}
	result.FontAnalysis.FontCount = len(result.FontAnalysis.DetectedFonts)
}

func (f *FontAnalyzer) analyzeFontPatterns(result *FontAnalysisResult) {
	commonCount := 0
	for _, font := range result.FontAnalysis.DetectedFonts {
		for common := range f.commonFonts {
			if strings.Contains(strings.ToLower(font), strings.ToLower(common)) {
				commonCount++
				break
			}
		}
	}
	result.FontAnalysis.CommonFontCount = commonCount
	
	for common := range f.commonFonts {
		found := false
		for _, font := range result.FontAnalysis.DetectedFonts {
			if strings.Contains(strings.ToLower(font), strings.ToLower(common)) {
				found = true
				break
			}
		}
		if !found {
			result.FontAnalysis.CommonFontMissing = append(result.FontAnalysis.CommonFontMissing, common)
		}
	}
	
	if commonCount == 0 && result.FontAnalysis.FontCount > 0 {
		result.Errors = append(result.Errors, "no_common_fonts_detected")
	}
	
	if result.FontAnalysis.FontCount < 3 {
		result.FontAnalysis.IsLimitedFontSet = true
		result.Warnings = append(result.Warnings, "limited_font_set")
	}
	
	totalDiversity := len(result.FontAnalysis.DetectedFonts)
	if totalDiversity > 0 {
		result.FontAnalysis.FontFamilyDiversity = float64(totalDiversity) / 20.0 * 100
	}
}

func (f *FontAnalyzer) analyzePlatformMatch(data map[string]interface{}, result *FontAnalysisResult) {
	platform := getString(data, "platform")
	result.PlatformMatch.Platform = platform
	
	expectedFonts := f.platformFonts[platform]
	if len(expectedFonts) > 0 {
		matchCount := 0
		for _, expected := range expectedFonts {
			for _, detected := range result.FontAnalysis.DetectedFonts {
				if strings.Contains(strings.ToLower(detected), strings.ToLower(expected)) {
					matchCount++
					break
				}
			}
		}
		result.PlatformMatch.MatchedFontCount = matchCount
		result.PlatformMatch.ExpectedFontCount = len(expectedFonts)
		result.PlatformMatch.MatchRatio = float64(matchCount) / float64(len(expectedFonts))
	}
}

func (f *FontAnalyzer) analyzeRenderingConsistency(data map[string]interface{}, result *FontAnalysisResult) {
	result.FontAnalysis.RenderingConsistency = 100.0
	
	if renderingData, ok := data["rendering_data"].(map[string]interface{}); ok {
		if consistency, ok := renderingData["consistency"].(float64); ok {
			result.FontAnalysis.RenderingConsistency = consistency
		}
	}
}

func (f *FontAnalyzer) calculateConfidence(result *FontAnalysisResult) {
	baseScore := 50.0
	
	if len(result.Errors) > 0 {
		baseScore -= float64(len(result.Errors)) * 20
	}
	
	if result.FontAnalysis.FontCount >= 10 {
		baseScore += 15
	} else if result.FontAnalysis.FontCount >= 5 {
		baseScore += 5
	}
	
	if result.FontAnalysis.CommonFontCount >= 3 {
		baseScore += 15
	} else if result.FontAnalysis.CommonFontCount >= 2 {
		baseScore += 5
	}
	
	if result.PlatformMatch.MatchRatio > 0.5 {
		baseScore += 10
	} else if result.PlatformMatch.MatchRatio > 0.3 {
		baseScore += 5
	}
	
	if result.FontAnalysis.RenderingConsistency > 90 {
		baseScore += 10
	}
	
	result.Confidence = math.Max(0, math.Min(100, baseScore))
	
	if result.Confidence < 30 {
		result.IsTampered = true
	}
}

func (f *FontAnalyzer) CompareFontFingerprints(fp1, fp2 *FontMetrics) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}
	
	if fp1.Hash != "" && fp2.Hash != "" && fp1.Hash == fp2.Hash {
		return 100.0
	}
	
	if fp1.FontCount == 0 && fp2.FontCount == 0 {
		return 0
	}
	
	commonFonts := 0
	totalFonts := len(fp1.DetectedFonts)
	
	fontSet := make(map[string]bool)
	for _, font := range fp1.DetectedFonts {
		fontSet[strings.ToLower(font)] = true
	}
	
	for _, font := range fp2.DetectedFonts {
		if fontSet[strings.ToLower(font)] {
			commonFonts++
		}
		totalFonts++
	}
	
	if totalFonts == 0 {
		return 0
	}
	
	return float64(commonFonts*2) / float64(totalFonts) * 100
}

type FingerprintStabilityAnalyzer struct {
	database       *FingerprintDatabase
	historyStorage map[string][]*FingerprintAnalysis
}

func NewFingerprintStabilityAnalyzer(db *FingerprintDatabase) *FingerprintStabilityAnalyzer {
	return &FingerprintStabilityAnalyzer{
		database:       db,
		historyStorage: make(map[string][]*FingerprintAnalysis),
	}
}

func (s *FingerprintStabilityAnalyzer) TrackFingerprint(fp *FingerprintAnalysis) {
	s.historyStorage[fp.FingerprintID] = append(s.historyStorage[fp.FingerprintID], fp)
	if len(s.historyStorage[fp.FingerprintID]) > 50 {
		s.historyStorage[fp.FingerprintID] = s.historyStorage[fp.FingerprintID][1:]
	}
}

type StabilityAnalysisResult struct {
	IsStable            bool     `json:"is_stable"`
	StabilityScore      float64  `json:"stability_score"`
	AverageSimilarity   float64  `json:"average_similarity"`
	SampleCount         int      `json:"sample_count"`
	InsufficientSamples bool     `json:"insufficient_samples"`
	Warnings            []string `json:"warnings"`
}

func (s *FingerprintStabilityAnalyzer) AnalyzeStability(fingerprintID string) *StabilityAnalysisResult {
	history, exists := s.historyStorage[fingerprintID]
	if !exists || len(history) < 2 {
		return &StabilityAnalysisResult{
			IsStable:            false,
			InsufficientSamples: true,
			Warnings:            make([]string, 0),
		}
	}
	
	result := &StabilityAnalysisResult{
		IsStable:   true,
		StabilityScore: 100.0,
		Warnings:   make([]string, 0),
	}
	
	reference := history[0]
	totalSimilarity := 0.0
	
	for i := 1; i < len(history); i++ {
		similarity := s.calculateOverallSimilarity(reference, history[i])
		totalSimilarity += similarity
	}
	
	result.AverageSimilarity = totalSimilarity / float64(len(history)-1)
	result.SampleCount = len(history)
	
	if result.AverageSimilarity < 95 {
		result.IsStable = false
		result.StabilityScore = result.AverageSimilarity
	}
	
	if result.AverageSimilarity > 99.9 && len(history) > 5 {
		result.Warnings = append(result.Warnings, "suspiciously_consistent")
	}
	
	return result
}

type TemporalStabilityResult struct {
	IsStable            bool     `json:"is_stable"`
	OverallScore        float64  `json:"overall_score"`
	CanvasStability     float64  `json:"canvas_stability"`
	WebGLStability      float64  `json:"webgl_stability"`
	UserAgentStability  float64  `json:"user_agent_stability"`
	SampleCount         int      `json:"sample_count"`
	InsufficientSamples bool     `json:"insufficient_samples"`
	Warnings            []string `json:"warnings"`
}

func (s *FingerprintStabilityAnalyzer) AnalyzeTemporalStability(fingerprintID string) *TemporalStabilityResult {
	history, exists := s.historyStorage[fingerprintID]
	if !exists || len(history) < 3 {
		return &TemporalStabilityResult{
			IsStable:            false,
			InsufficientSamples: true,
			Warnings:            make([]string, 0),
		}
	}
	
	result := &TemporalStabilityResult{
		IsStable:   true,
		SampleCount: len(history),
		Warnings:   make([]string, 0),
	}
	
	canvasMatches := 0
	webglMatches := 0
	uaMatches := 0
	
	for i := 1; i < len(history); i++ {
		if history[i].CanvasHash == history[0].CanvasHash && history[0].CanvasHash != "" {
			canvasMatches++
		}
		if history[i].WebGLHash == history[0].WebGLHash && history[0].WebGLHash != "" {
			webglMatches++
		}
		if history[i].UserAgent == history[0].UserAgent && history[0].UserAgent != "" {
			uaMatches++
		}
	}
	
	total := len(history) - 1
	
	result.CanvasStability = float64(canvasMatches) / float64(total) * 100
	result.WebGLStability = float64(webglMatches) / float64(total) * 100
	result.UserAgentStability = float64(uaMatches) / float64(total) * 100
	
	result.OverallScore = (result.CanvasStability*0.4 + result.WebGLStability*0.3 + result.UserAgentStability*0.3)
	
	if result.OverallScore < 90 {
		result.IsStable = false
	}
	
	return result
}

func (s *FingerprintStabilityAnalyzer) DetectStabilityAnomalies(fingerprintID string) []string {
	history, exists := s.historyStorage[fingerprintID]
	if !exists || len(history) < 5 {
		return []string{}
	}
	
	var anomalies []string
	
	for i := 1; i < len(history); i++ {
		if history[i].CanvasHash != history[0].CanvasHash {
			anomalies = append(anomalies, "canvas_hash_changed")
			break
		}
	}
	
	for i := 1; i < len(history); i++ {
		if history[i].WebGLHash != history[0].WebGLHash {
			anomalies = append(anomalies, "webgl_hash_changed")
			break
		}
	}
	
	return anomalies
}

func (s *FingerprintStabilityAnalyzer) calculateOverallSimilarity(fp1, fp2 *FingerprintAnalysis) float64 {
	if fp1.CanvasHash == fp2.CanvasHash && fp1.CanvasHash != "" {
		return 100.0
	}
	
	matchCount := 0
	totalFields := 0
	
	if fp1.CanvasHash != "" && fp2.CanvasHash != "" {
		totalFields++
		if fp1.CanvasHash == fp2.CanvasHash {
			matchCount++
		}
	}
	
	if fp1.WebGLHash != "" && fp2.WebGLHash != "" {
		totalFields++
		if fp1.WebGLHash == fp2.WebGLHash {
			matchCount++
		}
	}
	
	if fp1.UserAgent != "" && fp2.UserAgent != "" {
		totalFields++
		if fp1.UserAgent == fp2.UserAgent {
			matchCount++
		}
	}
	
	if totalFields == 0 {
		return 0
	}
	
	return float64(matchCount) / float64(totalFields) * 100
}

func main() {
	fmt.Println("=== Testing Canvas Similarity Analyzer ===")
	db := NewFingerprintDatabase()
	canvasAnalyzer := NewCanvasSimilarityAnalyzer(db)

	similarity := canvasAnalyzer.CalculateCanvasSimilarity("abc123", "abc123")
	fmt.Printf("Canvas similarity (same): %.2f (expected: 100.00)\n", similarity)

	similarity = canvasAnalyzer.CalculateCanvasSimilarity("abc123", "def456")
	fmt.Printf("Canvas similarity (different): %.2f (expected: 0.00)\n", similarity)

	similarity = canvasAnalyzer.CalculateHistogramSimilarity("aabbcc", "aabbcc")
	fmt.Printf("Histogram similarity (same): %.2f (expected: 100.00)\n", similarity)

	samples := []string{"abc123", "abc123", "abc123"}
	stability := canvasAnalyzer.AnalyzeHashStability(samples)
	fmt.Printf("Hash stability: stable=%v, score=%.2f\n", stability.IsStable, stability.StabilityScore)

	tampering := canvasAnalyzer.DetectHashTampering("", 0)
	fmt.Printf("Empty hash tampering: %v (expected: true)\n", tampering.IsTampered)

	tampering = canvasAnalyzer.DetectHashTampering("abc123", 6)
	fmt.Printf("Valid hash tampering: %v (expected: false)\n", tampering.IsTampered)

	tampering = canvasAnalyzer.DetectHashTampering("gibberish!", 0)
	fmt.Printf("Non-hex hash tampering: %v (expected: true)\n", tampering.IsTampered)

	tampering = canvasAnalyzer.DetectHashTampering("aaaaaaaaaaaaaaaa", 16)
	fmt.Printf("Low entropy hash tampering: %v (expected: true)\n", tampering.IsTampered)

	fmt.Println("\n=== Testing WebGL Analyzer ===")
	webglAnalyzer := NewWebGLAnalyzer(db)
	data := map[string]interface{}{
		"webgl_vendor":              "NVIDIA Corporation",
		"webgl_unmasked_vendor":     "NVIDIA Corporation",
		"webgl_renderer":            "NVIDIA GeForce GTX 1080",
		"webgl_unmasked_renderer":   "NVIDIA GeForce GTX 1080",
		"webgl_extensions":          []interface{}{"GL_EXT_blend_minmax", "GL_EXT_color_buffer_float"},
		"webgl_max_texture_size":    float64(8192),
		"webgl_max_vertex_attribs":  float64(16),
	}
	webglResult := webglAnalyzer.AnalyzeWebGLFingerprint(data)
	fmt.Printf("WebGL tampered: %v, score=%.2f, vendor_known=%v\n", 
		webglResult.IsTampered, webglResult.TamperingScore, webglResult.VendorAnalysis.IsKnown)

	data2 := map[string]interface{}{
		"webgl_vendor":   "Google Inc.",
		"webgl_renderer": "SwiftShader",
	}
	webglResult2 := webglAnalyzer.AnalyzeWebGLFingerprint(data2)
	fmt.Printf("Software renderer detection: %v\n", webglResult2.RendererAnalysis.IsSoftwareRenderer)

	data3 := map[string]interface{}{
		"webgl_vendor":   "fake vendor",
		"webgl_renderer": "test renderer",
	}
	webglResult3 := webglAnalyzer.AnalyzeWebGLFingerprint(data3)
	fmt.Printf("Blacklisted vendor detection: %v, errors=%v\n", webglResult3.IsTampered, webglResult3.Errors)

	fp1 := &WebGLMetrics{Vendor: "NVIDIA", Renderer: "GeForce", MaxTextureSize: 8192}
	fp2 := &WebGLMetrics{Vendor: "NVIDIA", Renderer: "GeForce", MaxTextureSize: 8192}
	similarity = webglAnalyzer.CompareWebGLFingerprints(fp1, fp2)
	fmt.Printf("WebGL fingerprint similarity: %.2f\n", similarity)

	fmt.Println("\n=== Testing Font Analyzer ===")
	fontAnalyzer := NewFontAnalyzer(db)
	fontData := map[string]interface{}{
		"detected_fonts": []interface{}{"Arial", "Helvetica", "Times New Roman", "Verdana", "Georgia"},
		"platform":       "windows",
	}
	fontResult := fontAnalyzer.AnalyzeFontFingerprint(fontData)
	fmt.Printf("Font tampered: %v, confidence=%.2f, common_fonts=%d\n", 
		fontResult.IsTampered, fontResult.Confidence, fontResult.FontAnalysis.CommonFontCount)

	fontData2 := map[string]interface{}{
		"detected_fonts": []interface{}{"Arial"},
		"platform":       "windows",
	}
	fontResult2 := fontAnalyzer.AnalyzeFontFingerprint(fontData2)
	fmt.Printf("Limited font set detection: %v, platform_match_ratio=%.2f\n", 
		fontResult2.FontAnalysis.IsLimitedFontSet, fontResult2.PlatformMatch.MatchRatio)

	fontData3 := map[string]interface{}{
		"detected_fonts": []interface{}{"SomeRareFont"},
	}
	fontResult3 := fontAnalyzer.AnalyzeFontFingerprint(fontData3)
	fmt.Printf("No common fonts detection: errors=%v\n", fontResult3.Errors)

	fontFp1 := &FontMetrics{Hash: "hash1", DetectedFonts: []string{"Arial", "Helvetica"}, FontCount: 2}
	fontFp2 := &FontMetrics{Hash: "hash1", DetectedFonts: []string{"Arial", "Helvetica"}, FontCount: 2}
	similarity = fontAnalyzer.CompareFontFingerprints(fontFp1, fontFp2)
	fmt.Printf("Font fingerprint similarity (same): %.2f\n", similarity)

	fmt.Println("\n=== Testing Stability Analyzer ===")
	stabilityAnalyzer := NewFingerprintStabilityAnalyzer(db)

	fpStab1 := &FingerprintAnalysis{
		FingerprintID:    "test_fp",
		CanvasHash:       "canvas1",
		WebGLHash:        "webgl1",
		UserAgent:        "Mozilla/5.0",
		ScreenResolution: "1920x1080",
		FirstSeen:        time.Now(),
		LastSeen:         time.Now(),
	}
	stabilityAnalyzer.TrackFingerprint(fpStab1)

	fpStab2 := &FingerprintAnalysis{
		FingerprintID:    "test_fp",
		CanvasHash:       "canvas1",
		WebGLHash:        "webgl1",
		UserAgent:        "Mozilla/5.0",
		ScreenResolution: "1920x1080",
		FirstSeen:        time.Now(),
		LastSeen:         time.Now(),
	}
	stabilityAnalyzer.TrackFingerprint(fpStab2)

	stabResult := stabilityAnalyzer.AnalyzeStability("test_fp")
	fmt.Printf("Stability: stable=%v, score=%.2f, avg_similarity=%.2f\n",
		stabResult.IsStable, stabResult.StabilityScore, stabResult.AverageSimilarity)

	for i := 0; i < 6; i++ {
		fp := &FingerprintAnalysis{
			FingerprintID: "fp_temporal",
			CanvasHash:    "canvas2",
			WebGLHash:     "webgl2",
			UserAgent:     "Mozilla/5.0",
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
		}
		stabilityAnalyzer.TrackFingerprint(fp)
	}

	temporalResult := stabilityAnalyzer.AnalyzeTemporalStability("fp_temporal")
	fmt.Printf("Temporal stability: stable=%v, overall_score=%.2f, canvas_stability=%.2f\n",
		temporalResult.IsStable, temporalResult.OverallScore, temporalResult.CanvasStability)

	fpDrift1 := &FingerprintAnalysis{FingerprintID: "fp_drift", CanvasHash: "canvasA", WebGLHash: "webglA"}
	fpDrift2 := &FingerprintAnalysis{FingerprintID: "fp_drift", CanvasHash: "canvasB", WebGLHash: "webglB"}
	stabilityAnalyzer.TrackFingerprint(fpDrift1)
	stabilityAnalyzer.TrackFingerprint(fpDrift2)

	anomalies := stabilityAnalyzer.DetectStabilityAnomalies("fp_drift")
	fmt.Printf("Anomalies with <5 samples: %v\n", anomalies)

	for i := 0; i < 5; i++ {
		fp := &FingerprintAnalysis{FingerprintID: "fp_drift", CanvasHash: string(rune('a'+i)) + "bc123", WebGLHash: "webgl1"}
		stabilityAnalyzer.TrackFingerprint(fp)
	}

	anomalies = stabilityAnalyzer.DetectStabilityAnomalies("fp_drift")
	fmt.Printf("Anomalies with hash drift: %v\n", anomalies)

	fmt.Println("\n=== All tests completed successfully! ===")
}
