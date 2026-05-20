package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

type FingerprintAdvanced struct {
	proxyPatterns   *ProxyPatternDetector
	canvasAnalyzer *CanvasAnalyzer
	webglAnalyzer  *WebGLAnalyzer
}

type ProxyPatternDetector struct {
	SuspiciousHeaders []string
	SuspiciousIPs    map[string]bool
}

type CanvasAnalyzer struct {
	noisePatterns    map[string]float64
	toleranceLevel   float64
}

type WebGLAnalyzer struct {
	knownBotRenderers []string
	suspiciousParams []string
}

func NewFingerprintAdvanced() *FingerprintAdvanced {
	return &FingerprintAdvanced{
		proxyPatterns: &ProxyPatternDetector{
			SuspiciousHeaders: []string{
				"x-forwarded-for",
				"x-real-ip",
				"via",
				"proxy-connection",
				"x-proxyid",
			},
			SuspiciousIPs: make(map[string]bool),
		},
		canvasAnalyzer: &CanvasAnalyzer{
			noisePatterns: map[string]float64{
				"uniform":   0.0,
				"low_var":   0.3,
				"high_var":  0.7,
				"patterned": 0.9,
			},
			toleranceLevel: 0.05,
		},
		webglAnalyzer: &WebGLAnalyzer{
			knownBotRenderers: []string{
				"SwiftShader",
				"llvmpipe",
				"Mesa",
				"Software",
				"virtualbox",
				"vmware",
			},
			suspiciousParams: []string{
				"ALIASED_LINE_WIDTH_RANGE",
				"ALIASED_POINT_SIZE_RANGE",
			},
		},
	}
}

func (f *FingerprintAdvanced) AnalyzeCanvasV2(canvasData string) CanvasFingerprintResult {
	result := CanvasFingerprintResult{}

	if canvasData == "" {
		result.Valid = false
		return result
	}

	result.Valid = true
	result.Hash = f.calculateHash(canvasData)
	result.Length = len(canvasData)

	entropy := f.calculateEntropy([]byte(canvasData))
	result.Entropy = entropy
	result.Quality = f.evaluateQuality(entropy, len(canvasData))

	return result
}

func (f *FingerprintAdvanced) AnalyzeWebGL(webglData map[string]interface{}) WebGLFingerprintResult {
	result := WebGLFingerprintResult{}

	if vendor, ok := webglData["vendor"].(string); ok {
		result.Vendor = vendor
	}
	if renderer, ok := webglData["renderer"].(string); ok {
		result.Renderer = renderer
	}
	if extensions, ok := webglData["extensions"].([]string); ok {
		result.ExtensionCount = len(extensions)
		result.Extensions = extensions
	}
	if params, ok := webglData["params"].(map[string]interface{}); ok {
		result.Params = params
	}

	fingerprintStr := result.Vendor + result.Renderer +
		strings.Join(result.Extensions, ",")
	result.Hash = f.calculateHash(fingerprintStr)

	result.Uniqueness = f.evaluateUniqueness(result)

	return result
}

func (f *FingerprintAdvanced) AnalyzeAudioContext(audioData map[string]interface{}) AudioFingerprintResult {
	result := AudioFingerprintResult{}

	if frequencyData, ok := audioData["frequencyData"].([]float64); ok {
		result.FrequencyData = frequencyData
		result.Hash = f.calculateHashFromFloatArray(frequencyData)
	}

	if waveformData, ok := audioData["waveformData"].([]float64); ok {
		result.WaveformData = waveformData
	}

	result.Characteristics = f.extractAudioCharacteristics(result.FrequencyData)

	return result
}

func (f *FingerprintAdvanced) AnalyzeFonts(fonts []string, baseFonts []string) FontFingerprintResult {
	result := FontFingerprintResult{}

	fontSet := make(map[string]bool)
	baseSet := make(map[string]bool)

	for _, font := range fonts {
		fontSet[strings.ToLower(font)] = true
	}
	for _, font := range baseFonts {
		baseSet[strings.ToLower(font)] = true
	}

	uniqueFonts := []string{}
	for font := range fontSet {
		if !baseSet[font] {
			uniqueFonts = append(uniqueFonts, font)
		}
	}

	result.TotalFonts = len(fonts)
	result.UniqueFonts = uniqueFonts
	result.UniqueCount = len(uniqueFonts)
	result.Hash = f.calculateHash(strings.Join(uniqueFonts, ","))

	result.Risk = f.evaluateFontRisk(uniqueFonts, len(fonts))

	return result
}

func (f *FingerprintAdvanced) AnalyzeHardwareConcurrency(concurrency int) HardwareFingerprintResult {
	result := HardwareFingerprintResult{}

	result.Concurrency = concurrency

	if concurrency < 1 || concurrency > 64 {
		result.Suspicious = true
		result.Risk = "high"
	} else if concurrency > 32 {
		result.Suspicious = true
		result.Risk = "medium"
	} else {
		result.Suspicious = false
		result.Risk = "low"
	}

	return result
}

func (f *FingerprintAdvanced) CalculateDeviceFingerprint(fingerprints map[string]string) string {
	combined := ""
	for key, value := range fingerprints {
		combined += key + ":" + value + "|"
	}
	return f.calculateHash(combined)
}

func (f *FingerprintAdvanced) calculateHash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (f *FingerprintAdvanced) calculateHashFromFloatArray(data []float64) string {
	str := ""
	for _, v := range data {
		str += string(rune(int(v * 1000)))
	}
	return f.calculateHash(str)
}

func (f *FingerprintAdvanced) calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	frequency := make(map[byte]int)
	for _, b := range data {
		frequency[b]++
	}

	entropy := 0.0
	for _, count := range frequency {
		p := float64(count) / float64(len(data))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (f *FingerprintAdvanced) evaluateQuality(entropy float64, length int) string {
	maxEntropy := 8.0
	normalizedEntropy := entropy / maxEntropy
	normalizedLength := math.Min(float64(length)/10000.0, 1.0)

	score := (normalizedEntropy + normalizedLength) / 2.0

	if score > 0.8 {
		return "high"
	} else if score > 0.5 {
		return "medium"
	}
	return "low"
}

func (f *FingerprintAdvanced) evaluateUniqueness(result WebGLFingerprintResult) float64 {
	score := 0.0

	if result.Vendor != "" && result.Renderer != "" {
		score += 0.3
	}

	if result.ExtensionCount > 50 {
		score += 0.3
	} else if result.ExtensionCount > 20 {
		score += 0.2
	}

	paramsLen := 0
	for _, v := range result.Params {
		paramsLen += len(fmt.Sprintf("%v", v))
	}
	if paramsLen > 1000 {
		score += 0.4
	} else if paramsLen > 500 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

func (f *FingerprintAdvanced) extractAudioCharacteristics(frequencyData []float64) map[string]float64 {
	characteristics := make(map[string]float64)

	if len(frequencyData) == 0 {
		return characteristics
	}

	sum := 0.0
	max := 0.0
	for _, v := range frequencyData {
		sum += v
		if v > max {
			max = v
		}
	}

	characteristics["mean"] = sum / float64(len(frequencyData))
	characteristics["max"] = max
	characteristics["variance"] = f.calculateVariance(frequencyData, characteristics["mean"])

	return characteristics
}

func (f *FingerprintAdvanced) evaluateFontRisk(uniqueFonts []string, totalFonts int) string {
	uniqueRatio := float64(len(uniqueFonts)) / float64(totalFonts)

	if uniqueRatio > 0.3 {
		return "high"
	} else if uniqueRatio > 0.15 {
		return "medium"
	}
	return "low"
}

func (f *FingerprintAdvanced) calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

type CanvasFingerprintResult struct {
	Valid   bool
	Hash    string
	Length  int
	Entropy float64
	Quality string
}

type WebGLFingerprintResult struct {
	Vendor         string
	Renderer       string
	ExtensionCount int
	Extensions     []string
	Params         map[string]interface{}
	Hash           string
	Uniqueness     float64
}

type AudioFingerprintResult struct {
	FrequencyData   []float64
	WaveformData    []float64
	Hash            string
	Characteristics map[string]float64
}

type FontFingerprintResult struct {
	TotalFonts  int
	UniqueFonts []string
	UniqueCount int
	Hash        string
	Risk        string
}

type HardwareFingerprintResult struct {
	Concurrency int
	Suspicious  bool
	Risk        string
}

func (f *FingerprintAdvanced) DetectProxyVPN(headers map[string]string, ip string, networkData map[string]interface{}) ProxyVPNResult {
	result := ProxyVPNResult{}

	result.IP = ip
	result.Headers = headers

	result.HeaderScore = f.analyzeProxyHeaders(headers)
	result.IPScore = f.analyzeIPAddress(ip)
	result.NetworkScore = f.analyzeNetworkPatterns(networkData)

	result.TotalScore = (result.HeaderScore + result.IPScore + result.NetworkScore) / 3.0
	result.IsProxy = result.TotalScore > 0.5
	result.IsVPN = f.detectVPNPatterns(networkData)

	return result
}

func (f *FingerprintAdvanced) analyzeProxyHeaders(headers map[string]string) float64 {
	score := 0.0
	suspiciousCount := 0

	for _, header := range f.proxyPatterns.SuspiciousHeaders {
		if _, exists := headers[strings.ToLower(header)]; exists {
			suspiciousCount++
		}
	}

	if suspiciousCount > 0 {
		score = float64(suspiciousCount) / float64(len(f.proxyPatterns.SuspiciousHeaders))
	}

	return math.Min(score, 1.0)
}

func (f *FingerprintAdvanced) analyzeIPAddress(ip string) float64 {
	if ip == "" {
		return 0.0
	}

	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return 0.0
	}

	firstOctet := 0
	fmt.Sscanf(parts[0], "%d", &firstOctet)

	if firstOctet >= 10 || firstOctet == 127 {
		return 0.0
	}

	if ip == "0.0.0.0" || ip == "127.0.0.1" {
		return 0.0
	}

	score := 0.3

	if f.proxyPatterns.SuspiciousIPs[ip] {
		score = 1.0
	}

	return score
}

func (f *FingerprintAdvanced) analyzeNetworkPatterns(networkData map[string]interface{}) float64 {
	score := 0.0

	if latency, ok := networkData["latency"].(float64); ok {
		if latency > 500 {
			score += 0.2
		}
		if latency < 10 {
			score += 0.3
		}
	}

	if asn, ok := networkData["asn"].(string); ok {
		if strings.Contains(strings.ToLower(asn), "hosting") ||
			strings.Contains(strings.ToLower(asn), "vpn") ||
			strings.Contains(strings.ToLower(asn), "proxy") {
			score += 0.5
		}
	}

	if rtt, ok := networkData["rtt_variance"].(float64); ok {
		if rtt > 100 {
			score += 0.2
		}
	}

	return math.Min(score, 1.0)
}

func (f *FingerprintAdvanced) detectVPNPatterns(networkData map[string]interface{}) bool {
	if country, ok := networkData["country"].(string); ok {
		highRiskCountries := []string{"RU", "CN", "KP", "IR"}
		for _, riskCountry := range highRiskCountries {
			if country == riskCountry {
				return true
			}
		}
	}

	if isp, ok := networkData["isp"].(string); ok {
		vpnKeywords := []string{"vpn", "proxy", "hosting", "datacenter"}
		for _, keyword := range vpnKeywords {
			if strings.Contains(strings.ToLower(isp), keyword) {
				return true
			}
		}
	}

	return false
}

func (f *FingerprintAdvanced) AnalyzeBrowserEnvironment(browserData map[string]interface{}) BrowserEnvironmentResult {
	result := BrowserEnvironmentResult{}

	if userAgent, ok := browserData["userAgent"].(string); ok {
		result.UserAgent = userAgent
		result.IsMobile = f.detectMobileBrowser(userAgent)
		result.BrowserInfo = f.parseUserAgent(userAgent)
	}

	if plugins, ok := browserData["plugins"].([]string); ok {
		result.PluginCount = len(plugins)
		result.HasFlash = f.checkPlugin(plugins, "flash")
		result.HasSilverlight = f.checkPlugin(plugins, "silverlight")
		result.IsHeadless = f.detectHeadlessBrowser(browserData)
	}

	if timezone, ok := browserData["timezone"].(string); ok {
		result.Timezone = timezone
		result.TimezoneOffset = f.analyzeTimezoneOffset(timezone, browserData)
	}

	result.CanvasScore = f.analyzeCanvasNoise(browserData)
	result.AudioScore = f.analyzeAudioFingerprint(browserData)
	result.WebGLScore = f.analyzeWebGLSuspicious(browserData)

	result.OverallRisk = f.calculateBrowserRiskScore(result)

	return result
}

func (f *FingerprintAdvanced) detectMobileBrowser(userAgent string) bool {
	mobileKeywords := []string{"mobile", "android", "iphone", "ipad", "tablet"}
	userAgentLower := strings.ToLower(userAgent)

	for _, keyword := range mobileKeywords {
		if strings.Contains(userAgentLower, keyword) {
			return true
		}
	}
	return false
}

func (f *FingerprintAdvanced) parseUserAgent(userAgent string) BrowserInfo {
	info := BrowserInfo{}

	if strings.Contains(userAgent, "Chrome") && !strings.Contains(userAgent, "Edg") {
		info.Browser = "Chrome"
	} else if strings.Contains(userAgent, "Firefox") {
		info.Browser = "Firefox"
	} else if strings.Contains(userAgent, "Safari") && !strings.Contains(userAgent, "Chrome") {
		info.Browser = "Safari"
	} else if strings.Contains(userAgent, "Edg") {
		info.Browser = "Edge"
	}

	if strings.Contains(userAgent, "Windows") {
		info.OS = "Windows"
	} else if strings.Contains(userAgent, "Mac") {
		info.OS = "macOS"
	} else if strings.Contains(userAgent, "Linux") {
		info.OS = "Linux"
	} else if strings.Contains(userAgent, "Android") {
		info.OS = "Android"
	} else if strings.Contains(userAgent, "iPhone") || strings.Contains(userAgent, "iPad") {
		info.OS = "iOS"
	}

	return info
}

func (f *FingerprintAdvanced) checkPlugin(plugins []string, name string) bool {
	for _, plugin := range plugins {
		if strings.Contains(strings.ToLower(plugin), strings.ToLower(name)) {
			return true
		}
	}
	return false
}

func (f *FingerprintAdvanced) detectHeadlessBrowser(browserData map[string]interface{}) bool {
	if webdriver, ok := browserData["webdriver"].(bool); ok && webdriver {
		return true
	}

	if languages, ok := browserData["languages"].([]string); ok {
		if len(languages) == 0 {
			return true
		}
		if len(languages) == 1 && languages[0] == "en-US" {
			return true
		}
	}

	if screenRes, ok := browserData["screenResolution"].(string); ok {
		commonHeadlessRes := []string{"800x600", "0x0"}
		for _, res := range commonHeadlessRes {
			if screenRes == res {
				return true
			}
		}
	}

	if automation, ok := browserData["automation"].(bool); ok && automation {
		return true
	}

	return false
}

func (f *FingerprintAdvanced) analyzeTimezoneOffset(timezone string, browserData map[string]interface{}) float64 {
	if offset, ok := browserData["timezoneOffset"].(int); ok {
		if offset == 0 {
			return 0.3
		}
	}

	if screenTimezone, ok := browserData["screenTimezone"].(string); ok {
		if screenTimezone != timezone {
			return 0.6
		}
	}

	return 0.0
}

func (f *FingerprintAdvanced) analyzeCanvasNoise(browserData map[string]interface{}) float64 {
	if canvasData, ok := browserData["canvasFingerprint"].(string); ok {
		if len(canvasData) < 100 {
			return 0.8
		}

		entropy := f.calculateEntropy([]byte(canvasData))
		if entropy < 3.0 {
			return 0.7
		}
	}

	return 0.0
}

func (f *FingerprintAdvanced) analyzeAudioFingerprint(browserData map[string]interface{}) float64 {
	if audioData, ok := browserData["audioFingerprint"].(string); ok {
		if len(audioData) < 50 {
			return 0.6
		}

		entropy := f.calculateEntropy([]byte(audioData))
		if entropy < 2.0 {
			return 0.5
		}
	}

	return 0.0
}

func (f *FingerprintAdvanced) analyzeWebGLSuspicious(browserData map[string]interface{}) float64 {
	score := 0.0

	if renderer, ok := browserData["webglRenderer"].(string); ok {
		rendererLower := strings.ToLower(renderer)
		for _, botRenderer := range f.webglAnalyzer.knownBotRenderers {
			if strings.Contains(rendererLower, strings.ToLower(botRenderer)) {
				score = 1.0
				break
			}
		}
	}

	if params, ok := browserData["webglParams"].(map[string]interface{}); ok {
		for _, suspiciousParam := range f.webglAnalyzer.suspiciousParams {
			if _, exists := params[suspiciousParam]; exists {
				score += 0.3
			}
		}
	}

	return math.Min(score, 1.0)
}

func (f *FingerprintAdvanced) calculateBrowserRiskScore(result BrowserEnvironmentResult) float64 {
	score := 0.0
	factorCount := 0.0

	if result.IsMobile {
		score += 0.1
	}
	factorCount++

	if result.PluginCount == 0 {
		score += 0.2
	}
	factorCount++

	if result.HasFlash || result.HasSilverlight {
		score += 0.1
	}
	factorCount++

	if result.IsHeadless {
		score += 0.8
	}
	factorCount++

	score += result.CanvasScore * 0.3
	factorCount++

	score += result.AudioScore * 0.2
	factorCount++

	score += result.WebGLScore * 0.4
	factorCount++

	if result.TimezoneOffset > 0.3 {
		score += 0.2
	}
	factorCount++

	return math.Min(score/factorCount*2, 1.0)
}

func (f *FingerprintAdvanced) GenerateEnhancedCanvasFingerprint(canvasContext map[string]interface{}) EnhancedCanvasResult {
	result := EnhancedCanvasResult{}

	if imageData, ok := canvasContext["imageData"].(string); ok {
		result.BaseFingerprint = f.calculateHash(imageData)
		result.ImageLength = len(imageData)

		result.NoiseLevel = f.calculateCanvasNoiseLevel(imageData)
		result.Consistency = f.analyzeCanvasConsistency(canvasContext)

		result.HiddenElements = f.detectHiddenCanvasElements(canvasContext)
		result.RenderingQuality = f.assessRenderingQuality(imageData)

		result.Risk = f.evaluateCanvasRisk(result)
	}

	return result
}

func (f *FingerprintAdvanced) calculateCanvasNoiseLevel(imageData string) string {
	if len(imageData) == 0 {
		return "unknown"
	}

	bytes := []byte(imageData)
	uniqueBytes := make(map[byte]bool)
	for _, b := range bytes {
		uniqueBytes[b] = true
	}

	entropy := f.calculateEntropy(bytes)
	variance := f.calculateByteVariance(bytes)

	if entropy < 3.0 || variance < 10.0 {
		return "low"
	} else if entropy > 6.0 && variance > 100.0 {
		return "high"
	}
	return "medium"
}

func (f *FingerprintAdvanced) calculateByteVariance(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, b := range data {
		sum += float64(b)
	}
	mean := sum / float64(len(data))

	variance := 0.0
	for _, b := range data {
		diff := float64(b) - mean
		variance += diff * diff
	}

	return variance / float64(len(data))
}

func (f *FingerprintAdvanced) analyzeCanvasConsistency(canvasContext map[string]interface{}) float64 {
	if attempts, ok := canvasContext["attempts"].(int); ok && attempts > 1 {
		if previousHash, ok := canvasContext["previousHash"].(string); ok {
			if currentHash, ok := canvasContext["currentHash"].(string); ok {
				if previousHash == currentHash {
					return 0.0
				}
			}
		}
	}

	return 1.0
}

func (f *FingerprintAdvanced) detectHiddenCanvasElements(canvasContext map[string]interface{}) bool {
	if hiddenElements, ok := canvasContext["hiddenElements"].(int); ok && hiddenElements > 0 {
		return true
	}
	return false
}

func (f *FingerprintAdvanced) assessRenderingQuality(imageData string) float64 {
	if len(imageData) == 0 {
		return 0.0
	}

	bytes := []byte(imageData)

	contrastScore := f.calculateContrastScore(bytes)

	resolution := len(bytes) / 4
	resolutionScore := 0.0
	if resolution > 10000 {
		resolutionScore = 1.0
	} else if resolution > 1000 {
		resolutionScore = 0.7
	} else if resolution > 100 {
		resolutionScore = 0.4
	}

	return (contrastScore + resolutionScore) / 2.0
}

func (f *FingerprintAdvanced) calculateContrastScore(data []byte) float64 {
	if len(data) < 3 {
		return 0.0
	}

	minVal := float64(data[0])
	maxVal := float64(data[0])

	for _, b := range data {
		if float64(b) < minVal {
			minVal = float64(b)
		}
		if float64(b) > maxVal {
			maxVal = float64(b)
		}
	}

	contrast := (maxVal - minVal) / 255.0

	return contrast
}

func (f *FingerprintAdvanced) evaluateCanvasRisk(result EnhancedCanvasResult) float64 {
	score := 0.0

	switch result.NoiseLevel {
	case "low":
		score += 0.4
	case "unknown":
		score += 0.3
	}

	if result.Consistency < 0.5 {
		score += 0.3
	}

	if result.HiddenElements {
		score += 0.2
	}

	if result.RenderingQuality < 0.3 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

type ProxyVPNResult struct {
	IP          string
	Headers     map[string]string
	HeaderScore float64
	IPScore     float64
	NetworkScore float64
	TotalScore  float64
	IsProxy     bool
	IsVPN       bool
}

type BrowserEnvironmentResult struct {
	UserAgent      string
	IsMobile       bool
	BrowserInfo    BrowserInfo
	PluginCount    int
	HasFlash       bool
	HasSilverlight bool
	IsHeadless     bool
	Timezone       string
	TimezoneOffset float64
	CanvasScore    float64
	AudioScore     float64
	WebGLScore     float64
	OverallRisk    float64
}

type BrowserInfo struct {
	Browser string
	OS      string
}

type EnhancedCanvasResult struct {
	BaseFingerprint   string
	ImageLength       int
	NoiseLevel        string
	Consistency       float64
	HiddenElements    bool
	RenderingQuality  float64
	Risk              float64
}
