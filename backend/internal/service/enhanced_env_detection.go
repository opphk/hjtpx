package service

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

type EnhancedEnvDetectionService struct {
	config          *EnhancedEnvConfig
	mu              sync.RWMutex
	detectionCache  map[string]*EnhancedEnvDetectionResult
	cacheTTL        time.Duration
}

type EnhancedEnvConfig struct {
	EnableCanvasAnalysis       bool
	EnableWebGLAnalysis       bool
	EnableProxyDetection      bool
	EnableHeadlessDetection   bool
	EnableUserAgentAnalysis   bool
	EnableBehaviorAnalysis    bool
	HighConfidenceThreshold   float64
	MediumConfidenceThreshold float64
	LowConfidenceThreshold    float64
}

type EnhancedEnvDetectionResult struct {
	SessionID        string                     `json:"session_id"`
	Timestamp        time.Time                  `json:"timestamp"`
	RiskLevel        string                     `json:"risk_level"`
	RiskScore        float64                    `json:"risk_score"`
	Confidence       float64                    `json:"confidence"`
	IsAutomated      bool                       `json:"is_automated"`
	DetectedTools    []string                   `json:"detected_tools"`
	DetectionMethods []EnhancedDetectionMethod   `json:"detection_methods"`
	CanvasAnalysis   *CanvasAnalysisResult      `json:"canvas_analysis,omitempty"`
	WebGLAnalysis   *WebGLAnalysisResult       `json:"webgl_analysis,omitempty"`
	ProxyAnalysis   *ProxyAnalysisResult       `json:"proxy_analysis,omitempty"`
	HeadlessAnalysis *HeadlessAnalysisResult    `json:"headless_analysis,omitempty"`
	UAAnalysis       *UserAgentAnalysisResult   `json:"ua_analysis,omitempty"`
	Recommendations []string                   `json:"recommendations"`
}

type EnhancedDetectionMethod struct {
	Method      string  `json:"method"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Score       float64 `json:"score"`
	Detected    bool    `json:"detected"`
}

type CanvasAnalysisResult struct {
	Hash              string             `json:"hash"`
	Entropy           float64            `json:"entropy"`
	IsSuspicious      bool               `json:"is_suspicious"`
	SuspiciousReasons []string           `json:"suspicious_reasons"`
	Uniqueness        float64            `json:"uniqueness"`
	Stability         float64            `json:"stability"`
	Features          CanvasFeatures     `json:"features"`
	AntiSpoofScore    float64            `json:"anti_spoof_score"`
}

type CanvasFeatures struct {
	HasText      bool    `json:"has_text"`
	HasGradient  bool    `json:"has_gradient"`
	HasBezier    bool    `json:"has_bezier"`
	HasArc       bool    `json:"has_arc"`
	HasShadow    bool    `json:"has_shadow"`
	HasComposite bool    `json:"has_composite"`
	HasEmoji     bool    `json:"has_emoji"`
	HasUnicode   bool    `json:"has_unicode"`
	TextLength   int     `json:"text_length"`
	Complexity   float64 `json:"complexity"`
}

type WebGLAnalysisResult struct {
	Vendor             string              `json:"vendor"`
	Renderer           string              `json:"renderer"`
	RendererType      string              `json:"renderer_type"`
	IsSoftware        bool                `json:"is_software"`
	IsVirtual         bool                `json:"is_virtual"`
	IsSuspicious      bool                `json:"is_suspicious"`
	SuspiciousReasons []string            `json:"suspicious_reasons"`
	Extensions        []string            `json:"extensions"`
	Uniqueness        float64             `json:"uniqueness"`
	Parameters        WebGLParameters     `json:"parameters"`
	AntiSpoofScore    float64             `json:"anti_spoof_score"`
}

type WebGLParameters struct {
	MaxTextureSize        int    `json:"max_texture_size"`
	MaxRenderbufferSize  int    `json:"max_renderbuffer_size"`
	MaxVertexAttribs     int    `json:"max_vertex_attribs"`
	MaxViewportDims      []int  `json:"max_viewport_dims"`
	MaxCubeMapTextureSize int   `json:"max_cube_map_texture_size"`
}

type ProxyAnalysisResult struct {
	IsProxy               bool     `json:"is_proxy"`
	IsVPN                 bool     `json:"is_vpn"`
	IsTor                 bool     `json:"is_tor"`
	IsDatacenter          bool     `json:"is_datacenter"`
	IsHosting             bool     `json:"is_hosting"`
	Country               string   `json:"country"`
	ISP                   string   `json:"isp"`
	ASN                   int      `json:"asn"`
	RiskScore             float64  `json:"risk_score"`
	DetectionIndicators   []string `json:"detection_indicators"`
}

type HeadlessAnalysisResult struct {
	IsHeadless          bool                       `json:"is_headless"`
	DetectedTool       string                     `json:"detected_tool"`
	RiskScore          float64                    `json:"risk_score"`
	Checks             []HeadlessCheckResult      `json:"checks"`
	NavigatorProps     NavigatorPropertyCheck     `json:"navigator_props"`
	EnvironmentProps   EnvironmentCheckResult     `json:"environment_props"`
}

type HeadlessCheckResult struct {
	CheckType string  `json:"check_type"`
	Passed    bool    `json:"passed"`
	RiskScore float64 `json:"risk_score"`
	Evidence  string  `json:"evidence"`
}

type NavigatorPropertyCheck struct {
	Webdriver           string  `json:"webdriver"`
	Languages           string  `json:"languages"`
	Plugins             string  `json:"plugins"`
	HardwareConcurrency int     `json:"hardware_concurrency"`
	DeviceMemory        float64 `json:"device_memory"`
	Platform            string  `json:"platform"`
	Vendor              string  `json:"vendor"`
	MaxTouchPoints      int     `json:"max_touch_points"`
}

type EnvironmentCheckResult struct {
	ScreenSize       string `json:"screen_size"`
	CanvasSupported  bool   `json:"canvas_supported"`
	WebGLSupported   bool   `json:"webgl_supported"`
	StorageAvailable bool   `json:"storage_available"`
}

type UserAgentAnalysisResult struct {
	RawUA                  string   `json:"raw_ua"`
	Browser                string   `json:"browser"`
	BrowserVer             string   `json:"browser_version"`
	OS                     string   `json:"os"`
	OSVersion              string   `json:"os_version"`
	DeviceType             string   `json:"device_type"`
	IsMobile               bool     `json:"is_mobile"`
	IsSuspicious           bool     `json:"is_suspicious"`
	SuspiciousReasons      []string `json:"suspicious_reasons"`
	AutomationIndicators   []string `json:"automation_indicators"`
	BotScore               float64  `json:"bot_score"`
}

func NewEnhancedEnvDetectionService() *EnhancedEnvDetectionService {
	return &EnhancedEnvDetectionService{
		config: &EnhancedEnvConfig{
			EnableCanvasAnalysis:       true,
			EnableWebGLAnalysis:       true,
			EnableProxyDetection:      true,
			EnableHeadlessDetection:   true,
			EnableUserAgentAnalysis:   true,
			EnableBehaviorAnalysis:    true,
			HighConfidenceThreshold:   70.0,
			MediumConfidenceThreshold: 50.0,
			LowConfidenceThreshold:    30.0,
		},
		detectionCache: make(map[string]*EnhancedEnvDetectionResult),
		cacheTTL:       5 * time.Minute,
	}
}

func (s *EnhancedEnvDetectionService) Detect(req *EnhancedEnvRequest) *EnhancedEnvDetectionResult {
	result := &EnhancedEnvDetectionResult{
		SessionID:        req.SessionID,
		Timestamp:        time.Now(),
		RiskLevel:        "low",
		RiskScore:        0,
		Confidence:       100,
		IsAutomated:      false,
		DetectedTools:    []string{},
		DetectionMethods: []EnhancedDetectionMethod{},
		Recommendations:  []string{},
	}

	s.mu.Lock()
	if cached, exists := s.detectionCache[req.SessionID]; exists {
		if time.Since(cached.Timestamp) < s.cacheTTL {
			s.mu.Unlock()
			return cached
		}
	}
	s.mu.Unlock()

	if s.config.EnableUserAgentAnalysis {
		uaResult := s.analyzeUserAgent(req.UserAgent, req.Headers)
		result.UAAnalysis = uaResult
		result.addDetectionMethod("user_agent", "User-Agent 分析", 0.15, uaResult.BotScore, uaResult.IsSuspicious)
		if uaResult.IsSuspicious {
			result.RiskScore += uaResult.BotScore * 0.15
			result.Confidence -= uaResult.BotScore * 0.1
		}
	}

	if s.config.EnableCanvasAnalysis {
		canvasResult := s.analyzeCanvas(req.CanvasHash, req.CanvasData)
		result.CanvasAnalysis = canvasResult
		result.addDetectionMethod("canvas", "Canvas 指纹分析", 0.2, canvasResult.AntiSpoofScore, canvasResult.IsSuspicious)
		if canvasResult.IsSuspicious {
			result.RiskScore += canvasResult.AntiSpoofScore * 0.2
			result.Confidence -= canvasResult.AntiSpoofScore * 0.15
		}
	}

	if s.config.EnableWebGLAnalysis {
		webglResult := s.analyzeWebGL(req.WebGLVendor, req.WebGLRenderer, req.WebGLParams)
		result.WebGLAnalysis = webglResult
		result.addDetectionMethod("webgl", "WebGL 指纹分析", 0.15, webglResult.AntiSpoofScore, webglResult.IsSuspicious)
		if webglResult.IsSuspicious {
			result.RiskScore += webglResult.AntiSpoofScore * 0.15
			result.Confidence -= webglResult.AntiSpoofScore * 0.1
		}
	}

	if s.config.EnableProxyDetection {
		proxyResult := s.analyzeProxy(req.IPAddress, req.Headers)
		result.ProxyAnalysis = proxyResult
		result.addDetectionMethod("proxy", "代理/VPN 检测", 0.25, proxyResult.RiskScore, proxyResult.IsProxy || proxyResult.IsVPN || proxyResult.IsTor)
		if proxyResult.IsProxy || proxyResult.IsVPN || proxyResult.IsTor {
			result.RiskScore += proxyResult.RiskScore * 0.25
			result.Confidence -= proxyResult.RiskScore * 0.2
		}
	}

	if s.config.EnableHeadlessDetection {
		headlessResult := s.analyzeHeadless(req)
		result.HeadlessAnalysis = headlessResult
		result.addDetectionMethod("headless", "Headless 浏览器检测", 0.25, headlessResult.RiskScore, headlessResult.IsHeadless)
		if headlessResult.IsHeadless {
			result.RiskScore += headlessResult.RiskScore * 0.25
			result.Confidence -= headlessResult.RiskScore * 0.2
			result.IsAutomated = true
			if headlessResult.DetectedTool != "" {
				result.DetectedTools = append(result.DetectedTools, headlessResult.DetectedTool)
			}
		}
	}

	s.calculateFinalRisk(result)

	s.mu.Lock()
	s.detectionCache[req.SessionID] = result
	s.mu.Unlock()

	return result
}

func (r *EnhancedEnvDetectionResult) addDetectionMethod(method, desc string, weight, score float64, detected bool) {
	r.DetectionMethods = append(r.DetectionMethods, EnhancedDetectionMethod{
		Method:      method,
		Description: desc,
		Weight:      weight,
		Score:       score,
		Detected:    detected,
	})
}

func (s *EnhancedEnvDetectionService) analyzeUserAgent(ua string, headers map[string]string) *UserAgentAnalysisResult {
	result := &UserAgentAnalysisResult{
		RawUA:                ua,
		IsSuspicious:         false,
		SuspiciousReasons:    []string{},
		AutomationIndicators: []string{},
		BotScore:             0,
	}

	if ua == "" {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "User-Agent 为空")
		result.BotScore += 30
	}

	uaLower := strings.ToLower(ua)

	browserPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"Chrome", regexp.MustCompile(`Chrome/(\d+)`)},
		{"Firefox", regexp.MustCompile(`Firefox/(\d+)`)},
		{"Safari", regexp.MustCompile(`Safari/(\d+)`)},
		{"Edge", regexp.MustCompile(`Edg/(\d+)`)},
		{"Opera", regexp.MustCompile(`OPR/(\d+)`)},
		{"IE", regexp.MustCompile(`MSIE\s?(\d+)`)},
		{"Edge Legacy", regexp.MustCompile(`Edge/(\d+)`)},
	}

	for _, bp := range browserPatterns {
		if matches := bp.pattern.FindStringSubmatch(ua); len(matches) > 1 {
			result.Browser = bp.name
			result.BrowserVer = matches[1]
			break
		}
	}

	if result.Browser == "" {
		result.Browser = "Unknown"
	}

	osPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"Windows", regexp.MustCompile(`Windows\s+NT\s?([\d.]+)`)},
		{"macOS", regexp.MustCompile(`Macintosh.*Mac\s+OS\s+X\s?([\d_]+)`)},
		{"Linux", regexp.MustCompile(`Linux`)},
		{"Android", regexp.MustCompile(`Android\s([\d.]+)`)},
		{"iOS", regexp.MustCompile(`iPhone\s+OS\s([\d_]+)`)},
		{"Chrome OS", regexp.MustCompile(`CrOS`)},
	}

	for _, op := range osPatterns {
		if op.pattern.MatchString(ua) {
			result.OS = op.name
			if matches := op.pattern.FindStringSubmatch(ua); len(matches) > 1 {
				result.OSVersion = matches[1]
			}
			break
		}
	}

	mobilePatterns := []string{"mobile", "android", "iphone", "ipad", "ipod", "blackberry", "windows phone"}
	for _, pattern := range mobilePatterns {
		if strings.Contains(uaLower, pattern) {
			result.IsMobile = true
			result.DeviceType = "mobile"
			break
		}
	}

	if !result.IsMobile {
		result.DeviceType = "desktop"
	}

	automationPatterns := []struct {
		pattern string
		tool    string
		score   float64
	}{
		{"headlesschrome", "Headless Chrome", 85},
		{"headless", "Headless Browser", 75},
		{"puppeteer", "Puppeteer", 88},
		{"playwright", "Playwright", 85},
		{"selenium", "Selenium", 80},
		{"webdriver", "WebDriver", 75},
		{"phantomjs", "PhantomJS", 90},
		{"phantom", "PhantomJS", 85},
		{"slimerjs", "SlimerJS", 80},
		{"geckodriver", "GeckoDriver", 70},
		{"chromedriver", "ChromeDriver", 65},
		{"firefox headless", "Headless Firefox", 80},
		{"cypress", "Cypress", 75},
		{"nightmare", "Nightmare", 70},
		{"testcafe", "TestCafe", 65},
		{"webdriverio", "WebDriverIO", 60},
		{"appium", "Appium", 55},
	}

	for _, ap := range automationPatterns {
		if strings.Contains(uaLower, ap.pattern) {
			result.AutomationIndicators = append(result.AutomationIndicators, ap.tool)
			result.BotScore += ap.score
			result.IsSuspicious = true
		}
	}

	botPatterns := []struct {
		pattern string
		bot     string
		score   float64
	}{
		{"googlebot", "Google Bot", 90},
		{"bingbot", "Bing Bot", 85},
		{"slurp", "Yahoo Slurp", 80},
		{"duckduckbot", "DuckDuckBot", 75},
		{"baiduspider", "Baidu Spider", 85},
		{"yandexbot", "Yandex Bot", 80},
		{"python-requests", "Python Bot", 70},
		{"python-urllib", "Python Bot", 70},
		{"curl", "cURL", 50},
		{"wget", "Wget", 50},
		{"httpie", "HTTPie", 45},
		{"axios", "Axios Bot", 55},
		{"node-fetch", "Node.js Bot", 60},
		{"java/", "Java Bot", 60},
		{"go-http-client", "Go Bot", 55},
		{"ruby", "Ruby Bot", 50},
		{"perl", "Perl Bot", 50},
		{"php", "PHP Bot", 55},
	}

	for _, bp := range botPatterns {
		if strings.Contains(uaLower, bp.pattern) {
			result.AutomationIndicators = append(result.AutomationIndicators, bp.bot)
			result.BotScore += bp.score
			result.IsSuspicious = true
		}
	}

	if len(ua) < 30 {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "User-Agent 过短")
		result.BotScore += 20
	}

	if result.BotScore > 100 {
		result.BotScore = 100
	}

	return result
}

func (s *EnhancedEnvDetectionService) analyzeCanvas(canvasHash string, canvasData map[string]interface{}) *CanvasAnalysisResult {
	result := &CanvasAnalysisResult{
		Hash:              canvasHash,
		IsSuspicious:      false,
		SuspiciousReasons: []string{},
		Features:          CanvasFeatures{},
		AntiSpoofScore:    0,
	}

	if canvasHash == "" {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "Canvas 指纹为空")
		result.AntiSpoofScore += 40
		return result
	}

	result.Entropy = s.calculateStringEntropy(canvasHash)
	result.Uniqueness = s.calculateUniqueness(canvasHash)

	if result.Entropy < 3.0 {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, fmt.Sprintf("Canvas 熵值过低: %.2f", result.Entropy))
		result.AntiSpoofScore += 25
	}

	if result.Entropy > 5.5 {
		result.Uniqueness = 0.9
	}

	hexRatio := s.calculateHexRatio(canvasHash)
	if hexRatio > 0.95 {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "Canvas 指纹几乎全为十六进制，可能缺乏渲染特征")
		result.AntiSpoofScore += 20
	}

	if len(canvasHash) < 32 {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "Canvas 指纹过短")
		result.AntiSpoofScore += 30
	}

	repeatScore := s.calculateRepeatScore(canvasHash)
	if repeatScore > 50 {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "Canvas 指纹存在大量重复字符")
		result.AntiSpoofScore += 25
	}

	if canvasData != nil {
		s.extractCanvasFeatures(canvasData, &result.Features)
		result.Features.Complexity = s.calculateCanvasComplexity(&result.Features)
	}

	commonHashes := map[string]bool{
		"a1b2c3d4e5f6":      true,
		"1234567890abcdef":   true,
		"ffffffffffffffff":   true,
		"0000000000000000":   true,
		"deadbeefdeadbeef":   true,
	}
	if commonHashes[canvasHash] {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "Canvas 指纹为常见测试值")
		result.AntiSpoofScore += 50
	}

	if result.AntiSpoofScore > 100 {
		result.AntiSpoofScore = 100
	}

	return result
}

func (s *EnhancedEnvDetectionService) extractCanvasFeatures(data map[string]interface{}, features *CanvasFeatures) {
	if data == nil {
		return
	}

	if val, ok := data["has_text"]; ok {
		features.HasText, _ = val.(bool)
	}
	if val, ok := data["has_gradient"]; ok {
		features.HasGradient, _ = val.(bool)
	}
	if val, ok := data["has_bezier"]; ok {
		features.HasBezier, _ = val.(bool)
	}
	if val, ok := data["has_arc"]; ok {
		features.HasArc, _ = val.(bool)
	}
	if val, ok := data["has_shadow"]; ok {
		features.HasShadow, _ = val.(bool)
	}
	if val, ok := data["has_composite"]; ok {
		features.HasComposite, _ = val.(bool)
	}
	if val, ok := data["has_emoji"]; ok {
		features.HasEmoji, _ = val.(bool)
	}
	if val, ok := data["has_unicode"]; ok {
		features.HasUnicode, _ = val.(bool)
	}
	if val, ok := data["text_length"]; ok {
		if v, ok := val.(float64); ok {
			features.TextLength = int(v)
		}
	}
}

func (s *EnhancedEnvDetectionService) calculateCanvasComplexity(features *CanvasFeatures) float64 {
	complexity := 0.0
	if features.HasText {
		complexity += 15
	}
	if features.HasGradient {
		complexity += 20
	}
	if features.HasBezier {
		complexity += 25
	}
	if features.HasArc {
		complexity += 15
	}
	if features.HasShadow {
		complexity += 10
	}
	if features.HasComposite {
		complexity += 20
	}
	if features.HasEmoji {
		complexity += 25
	}
	if features.HasUnicode {
		complexity += 15
	}
	if features.TextLength > 50 {
		complexity += 10
	}
	return math.Min(complexity, 100)
}

func (s *EnhancedEnvDetectionService) analyzeWebGL(vendor, renderer string, params map[string]interface{}) *WebGLAnalysisResult {
	result := &WebGLAnalysisResult{
		Vendor:             vendor,
		Renderer:           renderer,
		IsSuspicious:       false,
		SuspiciousReasons:  []string{},
		Extensions:         []string{},
		Parameters:         WebGLParameters{},
		AntiSpoofScore:     0,
	}

	if vendor == "" && renderer == "" {
		result.IsSuspicious = true
		result.SuspiciousReasons = append(result.SuspiciousReasons, "WebGL 信息完全缺失")
		result.AntiSpoofScore += 50
		return result
	}

	rendererLower := strings.ToLower(renderer)
	vendorLower := strings.ToLower(vendor)

	softwarePatterns := []string{
		"swiftshader",
		"llvmpipe",
		"mesa",
		"software",
		"emulated",
		"virtual",
	}

	for _, pattern := range softwarePatterns {
		if strings.Contains(rendererLower, pattern) {
			result.IsSoftware = true
			result.RendererType = "software"
			result.IsSuspicious = true
			result.SuspiciousReasons = append(result.SuspiciousReasons, fmt.Sprintf("检测到软件渲染器: %s", pattern))
			result.AntiSpoofScore += 40
			break
		}
	}

	virtualPatterns := []string{
		"virtualbox",
		"vmware",
		"qemu",
		"kvm",
		"parallels",
		"hyper-v",
		"xen",
	}

	for _, pattern := range virtualPatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			result.IsVirtual = true
			result.RendererType = "virtual"
			result.IsSuspicious = true
			result.SuspiciousReasons = append(result.SuspiciousReasons, fmt.Sprintf("检测到虚拟机: %s", pattern))
			result.AntiSpoofScore += 45
			break
		}
	}

	if !result.IsSoftware && !result.IsVirtual {
		result.RendererType = "hardware"
	}

	suspiciousPatterns := []string{
		"fake", "mock", "test", "spoof",
		"none", "unknown", "undefined", "null",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(vendorLower, pattern) {
			result.IsSuspicious = true
			result.SuspiciousReasons = append(result.SuspiciousReasons, fmt.Sprintf("厂商名称包含可疑关键词: %s", pattern))
			result.AntiSpoofScore += 35
		}
		if strings.Contains(rendererLower, pattern) {
			result.IsSuspicious = true
			result.SuspiciousReasons = append(result.SuspiciousReasons, fmt.Sprintf("渲染器名称包含可疑关键词: %s", pattern))
			result.AntiSpoofScore += 35
		}
	}

	if params != nil {
		if val, ok := params["max_texture_size"]; ok {
			if v, ok := val.(float64); ok {
				result.Parameters.MaxTextureSize = int(v)
				if v < 2048 {
					result.IsSuspicious = true
					result.SuspiciousReasons = append(result.SuspiciousReasons, "纹理尺寸限制异常")
					result.AntiSpoofScore += 15
				}
			}
		}
		if val, ok := params["extensions"]; ok {
			if exts, ok := val.([]interface{}); ok {
				for _, ext := range exts {
					if extStr, ok := ext.(string); ok {
						result.Extensions = append(result.Extensions, extStr)
					}
				}
				if len(result.Extensions) < 3 {
					result.IsSuspicious = true
					result.SuspiciousReasons = append(result.SuspiciousReasons, fmt.Sprintf("WebGL 扩展数量过少: %d", len(result.Extensions)))
					result.AntiSpoofScore += 20
				}
			}
		}
	}

	gpuFamilies := []string{"nvidia", "geforce", "amd", "radeon", "ati", "intel", "apple", "adreno", "mali", "powervr"}
	for _, family := range gpuFamilies {
		if strings.Contains(rendererLower, family) {
			result.Uniqueness = 0.8
			break
		}
	}

	if result.AntiSpoofScore > 100 {
		result.AntiSpoofScore = 100
	}

	return result
}

func (s *EnhancedEnvDetectionService) analyzeProxy(ip string, headers map[string]string) *ProxyAnalysisResult {
	result := &ProxyAnalysisResult{
		RiskScore:           0,
		DetectionIndicators: []string{},
	}

	if xff, ok := headers["X-Forwarded-For"]; ok && xff != "" {
		result.DetectionIndicators = append(result.DetectionIndicators, "X-Forwarded-For header present")
		result.RiskScore += 25
		ips := strings.Split(xff, ",")
		if len(ips) > 2 {
			result.IsProxy = true
			result.RiskScore += 15
		}
	}

	if xri, ok := headers["X-Real-IP"]; ok && xri != "" {
		result.DetectionIndicators = append(result.DetectionIndicators, "X-Real-IP header present")
		result.RiskScore += 15
	}

	if via, ok := headers["Via"]; ok && via != "" {
		viaLower := strings.ToLower(via)
		result.DetectionIndicators = append(result.DetectionIndicators, "Via header present")
		result.RiskScore += 20

		proxyKeywords := []string{"proxy", "squid", "nginx", "apache", "varnish", "haproxy", "traefik"}
		for _, keyword := range proxyKeywords {
			if strings.Contains(viaLower, keyword) {
				result.IsProxy = true
				result.DetectionIndicators = append(result.DetectionIndicators, fmt.Sprintf("Known proxy: %s", keyword))
				result.RiskScore += 15
				break
			}
		}
	}

	for header := range headers {
		headerLower := strings.ToLower(header)
		if strings.Contains(headerLower, "vpn") || strings.Contains(headerLower, "proxy") || strings.Contains(headerLower, "tor") {
			result.DetectionIndicators = append(result.DetectionIndicators, fmt.Sprintf("Suspicious header: %s", header))
			result.RiskScore += 20
		}
	}

	torRanges := []string{
		"128.31.", "199.87.", "199.58.", "171.25.",
		"162.247.", "45.33.", "104.244.", "77.247.",
		"93.95.", "185.220.", "192.95.", "193.11.",
		"199.249.", "204.13.", "209.141.", "23.129.",
		"45.154.", "62.210.", "66.111.", "72.14.",
		"78.142.", "86.59.", "91.250.", "94.140.",
		"95.211.", "131.188.", "154.35.", "176.10.",
	}

	for _, torRange := range torRanges {
		if strings.HasPrefix(ip, torRange) {
			result.IsTor = true
			result.RiskScore += 50
			result.DetectionIndicators = append(result.DetectionIndicators, "Known Tor exit node IP range")
			break
		}
	}

	vpnProviders := map[string][]string{
		"NordVPN":      {"45.33.", "45.45.", "45.67.", "45.89."},
		"ExpressVPN":   {"23.", "104.", "132."},
		"Surfshark":    {"172.104.", "185.220.", "188.172."},
		"CyberGhost":   {"37.", "82.", "85.", "89."},
		"PIA":          {"104.238.", "107.170.", "172.104."},
		"ProtonVPN":    {"185.195.", "185.220."},
		"Mullvad":      {"185.195.", "194.132."},
		"Windscribe":   {"35.182.", "45.33."},
		"HideMyAss":    {"185.183.", "212.83."},
		"IPVanish":     {"107.170.", "172.104."},
	}

	for provider, prefixes := range vpnProviders {
		for _, prefix := range prefixes {
			if strings.HasPrefix(ip, prefix) {
				result.IsVPN = true
				result.RiskScore += 35
				result.DetectionIndicators = append(result.DetectionIndicators, fmt.Sprintf("Known VPN provider: %s", provider))
				break
			}
		}
	}

	datacenterPrefixes := []string{
		"3.", "4.", "8.", "13.", "15.", "16.", "17.", "18.", "20.",
		"23.", "34.", "35.", "40.", "44.", "45.", "47.", "48.", "49.",
		"50.", "52.", "54.", "63.", "64.", "65.", "66.", "67.", "68.",
	}

	for _, prefix := range datacenterPrefixes {
		if strings.HasPrefix(ip, prefix) {
			result.IsDatacenter = true
			result.RiskScore += 20
			result.DetectionIndicators = append(result.DetectionIndicators, "Datacenter IP range")
			break
		}
	}

	hostingProviders := []string{
		"digitalocean", "linode", "vultr", "aws", "azure", "gce",
		"ovh", "hetrix", "hostwinds", "ramnode", "contabo",
	}

	if ua, ok := headers["User-Agent"]; ok {
		uaLower := strings.ToLower(ua)
		for _, provider := range hostingProviders {
			if strings.Contains(uaLower, provider) {
				result.IsHosting = true
				result.RiskScore += 15
				result.DetectionIndicators = append(result.DetectionIndicators, fmt.Sprintf("Hosting provider detected: %s", provider))
				break
			}
		}
	}

	if result.RiskScore > 100 {
		result.RiskScore = 100
	}

	return result
}

func (s *EnhancedEnvDetectionService) analyzeHeadless(req *EnhancedEnvRequest) *HeadlessAnalysisResult {
	result := &HeadlessAnalysisResult{
		Checks:           []HeadlessCheckResult{},
		NavigatorProps:   NavigatorPropertyCheck{},
		EnvironmentProps: EnvironmentCheckResult{
			CanvasSupported:  true,
			WebGLSupported:   true,
			StorageAvailable: true,
		},
	}

	uaLower := strings.ToLower(req.UserAgent)

	result.Checks = append(result.Checks, s.checkUAHeadless(uaLower))

	if req.NavigatorProps != nil {
		if webdriver, ok := req.NavigatorProps["webdriver"]; ok {
			if webdriver == "true" {
				result.Checks = append(result.Checks, HeadlessCheckResult{
					CheckType: "navigator_webdriver",
					Passed:    false,
					RiskScore: 85,
					Evidence:  "navigator.webdriver is true",
				})
				result.RiskScore += 85
				result.IsHeadless = true
			}
		}

		if languages, ok := req.NavigatorProps["languages"]; ok {
			if languages == "" || languages == "[]" {
				result.Checks = append(result.Checks, HeadlessCheckResult{
					CheckType: "navigator_languages",
					Passed:    false,
					RiskScore: 30,
					Evidence:  "navigator.languages is empty",
				})
				result.RiskScore += 30
			}
		}

		if plugins, ok := req.NavigatorProps["plugins"]; ok {
			if plugins == "" || plugins == "[]" {
				result.Checks = append(result.Checks, HeadlessCheckResult{
					CheckType: "navigator_plugins",
					Passed:    false,
					RiskScore: 25,
					Evidence:  "navigator.plugins is empty",
				})
				result.RiskScore += 25
			}
		}

		if hwConcurrency, ok := req.NavigatorProps["hardware_concurrency"]; ok {
			if hwConcurrency == "1" || hwConcurrency == "2" {
				result.Checks = append(result.Checks, HeadlessCheckResult{
					CheckType: "hardware_concurrency",
					Passed:    false,
					RiskScore: 15,
					Evidence:  fmt.Sprintf("Low CPU cores: %s", hwConcurrency),
				})
				result.RiskScore += 15
			}
		}
	}

	if req.WebGLRenderer != "" {
		rendererLower := strings.ToLower(req.WebGLRenderer)
		softwareIndicators := []string{"swiftshader", "llvmpipe", "mesa", "software", "virtual"}

		for _, indicator := range softwareIndicators {
			if strings.Contains(rendererLower, indicator) {
				result.Checks = append(result.Checks, HeadlessCheckResult{
					CheckType: "webgl_software_renderer",
					Passed:    false,
					RiskScore: 50,
					Evidence:  fmt.Sprintf("Software WebGL renderer: %s", req.WebGLRenderer),
				})
				result.RiskScore += 50
				break
			}
		}
	}

	if req.ScreenWidth == 0 || req.ScreenHeight == 0 {
		result.Checks = append(result.Checks, HeadlessCheckResult{
			CheckType: "screen_size",
			Passed:    false,
			RiskScore: 40,
			Evidence:  "Screen size is zero",
		})
		result.RiskScore += 40
		result.EnvironmentProps.CanvasSupported = false
	} else if req.ScreenWidth == 800 && req.ScreenHeight == 600 {
		result.Checks = append(result.Checks, HeadlessCheckResult{
			CheckType: "screen_size",
			Passed:    false,
			RiskScore: 30,
			Evidence:  "Common headless resolution: 800x600",
		})
		result.RiskScore += 30
	}

	if req.HeadlessIndicators != nil {
		for _, indicator := range req.HeadlessIndicators {
			indicatorLower := strings.ToLower(indicator)
			if strings.Contains(indicatorLower, "automation") ||
				strings.Contains(indicatorLower, "headless") ||
				strings.Contains(indicatorLower, "puppeteer") ||
				strings.Contains(indicatorLower, "playwright") ||
				strings.Contains(indicatorLower, "selenium") {
				result.RiskScore += 40
				result.IsHeadless = true
				result.DetectedTool = indicator
			}
		}
	}

	if result.RiskScore > 70 {
		result.IsHeadless = true
	}

	return result
}

func (s *EnhancedEnvDetectionService) checkUAHeadless(uaLower string) HeadlessCheckResult {
	headlessPatterns := []struct {
		pattern string
		tool    string
	}{
		{"headlesschrome", "Headless Chrome"},
		{"headless chrome", "Headless Chrome"},
		{"headless", "Headless Browser"},
		{"puppeteer", "Puppeteer"},
		{"playwright", "Playwright"},
		{"selenium", "Selenium"},
		{"webdriver", "WebDriver"},
		{"phantomjs", "PhantomJS"},
		{"phantom", "PhantomJS"},
		{"cypress", "Cypress"},
		{"nightmare", "Nightmare"},
	}

	for _, hp := range headlessPatterns {
		if strings.Contains(uaLower, hp.pattern) {
			return HeadlessCheckResult{
				CheckType: "ua_headless",
				Passed:    false,
				RiskScore: 45,
				Evidence:  fmt.Sprintf("User-Agent contains: %s", hp.tool),
			}
		}
	}

	return HeadlessCheckResult{
		CheckType: "ua_headless",
		Passed:    true,
		RiskScore: 0,
		Evidence:  "No headless indicators in User-Agent",
	}
}

func (s *EnhancedEnvDetectionService) calculateFinalRisk(result *EnhancedEnvDetectionResult) {
	result.RiskScore = math.Min(result.RiskScore, 100)
	result.Confidence = math.Max(0, math.Min(100, result.Confidence))

	if result.RiskScore >= 70 {
		result.RiskLevel = "critical"
		result.Recommendations = append(result.Recommendations, "阻止访问，要求额外验证")
	} else if result.RiskScore >= 50 {
		result.RiskLevel = "high"
		result.Recommendations = append(result.Recommendations, "添加额外验证步骤")
	} else if result.RiskScore >= 30 {
		result.RiskLevel = "medium"
		result.Recommendations = append(result.Recommendations, "启用增强监控")
	} else {
		result.RiskLevel = "low"
	}

	if result.IsAutomated {
		result.Recommendations = append(result.Recommendations, "检测到自动化工具，添加验证码验证")
	}

	if len(result.DetectedTools) > 0 {
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("检测到的自动化工具: %s", strings.Join(result.DetectedTools, ", ")))
	}
}

func (s *EnhancedEnvDetectionService) calculateStringEntropy(str string) float64 {
	if len(str) == 0 {
		return 0.0
	}

	charCounts := make(map[byte]int)
	for _, c := range str {
		charCounts[byte(c)]++
	}

	entropy := 0.0
	for _, count := range charCounts {
		p := float64(count) / float64(len(str))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (s *EnhancedEnvDetectionService) calculateUniqueness(str string) float64 {
	if len(str) == 0 {
		return 0.0
	}

	uniqueChars := make(map[byte]bool)
	for _, c := range str {
		uniqueChars[byte(c)] = true
	}

	return float64(len(uniqueChars)) / float64(len(str))
}

func (s *EnhancedEnvDetectionService) calculateHexRatio(str string) float64 {
	if len(str) == 0 {
		return 0.0
	}

	hexCount := 0
	for _, c := range str {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			hexCount++
		}
	}

	return float64(hexCount) / float64(len(str))
}

func (s *EnhancedEnvDetectionService) calculateRepeatScore(str string) float64 {
	if len(str) < 2 {
		return 0.0
	}

	repeats := 0
	maxRepeatLen := 0
	currentRepeatLen := 1

	for i := 1; i < len(str); i++ {
		if str[i] == str[i-1] {
			currentRepeatLen++
			if currentRepeatLen > maxRepeatLen {
				maxRepeatLen = currentRepeatLen
			}
		} else {
			if currentRepeatLen > 1 {
				repeats += currentRepeatLen - 1
			}
			currentRepeatLen = 1
		}
	}

	if currentRepeatLen > 1 {
		repeats += currentRepeatLen - 1
	}

	return float64(repeats) / float64(len(str)) * 100.0
}

type EnhancedEnvRequest struct {
	SessionID          string
	UserAgent          string
	IPAddress          string
	Headers            map[string]string
	CanvasHash         string
	CanvasData         map[string]interface{}
	WebGLVendor        string
	WebGLRenderer      string
	WebGLParams        map[string]interface{}
	NavigatorProps     map[string]string
	ScreenWidth        int
	ScreenHeight       int
	HeadlessIndicators []string
}

func (s *EnhancedEnvDetectionService) BatchDetect(requests []*EnhancedEnvRequest) []*EnhancedEnvDetectionResult {
	results := make([]*EnhancedEnvDetectionResult, len(requests))
	for i, req := range requests {
		results[i] = s.Detect(req)
	}
	return results
}

func (s *EnhancedEnvDetectionService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detectionCache = make(map[string]*EnhancedEnvDetectionResult)
}

func (s *EnhancedEnvDetectionService) UpdateConfig(config *EnhancedEnvConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if config.EnableCanvasAnalysis {
		s.config.EnableCanvasAnalysis = config.EnableCanvasAnalysis
	}
	if config.EnableWebGLAnalysis {
		s.config.EnableWebGLAnalysis = config.EnableWebGLAnalysis
	}
	if config.EnableProxyDetection {
		s.config.EnableProxyDetection = config.EnableProxyDetection
	}
	if config.EnableHeadlessDetection {
		s.config.EnableHeadlessDetection = config.EnableHeadlessDetection
	}
	if config.EnableUserAgentAnalysis {
		s.config.EnableUserAgentAnalysis = config.EnableUserAgentAnalysis
	}
	if config.HighConfidenceThreshold > 0 {
		s.config.HighConfidenceThreshold = config.HighConfidenceThreshold
	}
	if config.MediumConfidenceThreshold > 0 {
		s.config.MediumConfidenceThreshold = config.MediumConfidenceThreshold
	}
	if config.LowConfidenceThreshold > 0 {
		s.config.LowConfidenceThreshold = config.LowConfidenceThreshold
	}
}

func (s *EnhancedEnvDetectionService) GetConfig() *EnhancedEnvConfig {
	return s.config
}
