package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	botUserAgentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)bot`),
		regexp.MustCompile(`(?i)crawler`),
		regexp.MustCompile(`(?i)spider`),
		regexp.MustCompile(`(?i)scraper`),
		regexp.MustCompile(`(?i)curl`),
		regexp.MustCompile(`(?i)wget`),
		regexp.MustCompile(`(?i)python-requests`),
		regexp.MustCompile(`(?i)scrapy`),
		regexp.MustCompile(`(?i)selenium`),
		regexp.MustCompile(`(?i)headless`),
		regexp.MustCompile(`(?i)phantom`),
		regexp.MustCompile(`(?i)puppeteer`),
		regexp.MustCompile(`(?i)playwright`),
		regexp.MustCompile(`(?i)googlebot`),
		regexp.MustCompile(`(?i)bingbot`),
		regexp.MustCompile(`(?i)slurp`),
		regexp.MustCompile(`(?i)duckduckbot`),
		regexp.MustCompile(`(?i)baiduspider`),
		regexp.MustCompile(`(?i)yandexbot`),
		regexp.MustCompile(`(?i)sogou`),
		regexp.MustCompile(`(?i)exabot`),
		regexp.MustCompile(`(?i)facebot`),
		regexp.MustCompile(`(?i)ia_archiver`),
	}

	suspiciousHeaderNames = []string{
		"X-Scanner",
		"X-Forwarded-For",
		"Via",
		"X-ProxyUser-Ip",
		"X-Originating-IP",
		"X-Remote-IP",
		"X-Proxy-IP",
		"X-Client-IP",
		"X-Real-IP",
		"X-Forwarded",
		"X-Forwarded-Host",
		"Forwarded-For",
		"X-Cluster-Client-IP",
	}

	automationIndicatorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)webdriver`),
		regexp.MustCompile(`(?i)selenium`),
		regexp.MustCompile(`(?i)chrome-automation`),
		regexp.MustCompile(`(?i)automation`),
		regexp.MustCompile(`(?i)test`),
	}
)

type BotFingerprintData struct {
	FingerprintID string
	FirstSeen     time.Time
	LastSeen      time.Time
	RequestCount  int
	RiskScore     float64
	IsBlacklisted bool
	UserAgent     string
	IP            string
}

type BotBehaviorData struct {
	IP           string
	RequestTimes []time.Time
	RequestPaths []string
	Methods      []string
	RequestCount int
	LastActivity time.Time
	AvgInterval  float64
	IsRegular    bool
}

type BotDetectionResult struct {
	IsBot         bool
	ShouldBlock   bool
	RiskScore     float64
	Reasons       []string
	ChallengeType string
	Confidence    float64
}

type BotDetectionService struct {
	fingerprints    map[string]*BotFingerprintData
	behaviors       map[string]*BotBehaviorData
	mu              sync.RWMutex
	botPatterns     []*regexp.Regexp
	headerPatterns  []string
	autoIndicators  []*regexp.Regexp
	maxFingerprints int
	maxBehaviors    int
}

func NewBotDetectionService() *BotDetectionService {
	return &BotDetectionService{
		fingerprints:    make(map[string]*BotFingerprintData),
		behaviors:       make(map[string]*BotBehaviorData),
		botPatterns:     botUserAgentPatterns,
		headerPatterns:  suspiciousHeaderNames,
		autoIndicators:  automationIndicatorPatterns,
		maxFingerprints: 10000,
		maxBehaviors:    10000,
	}
}

func (s *BotDetectionService) DetectBot(r *http.Request, additionalData map[string]string) *BotDetectionResult {
	ip := getClientIP(r)
	userAgent := r.UserAgent()

	result := &BotDetectionResult{
		IsBot:         false,
		ShouldBlock:   false,
		RiskScore:     0.0,
		Reasons:       []string{},
		ChallengeType: "",
		Confidence:    0.0,
	}

	score := 0.0
	confidence := 0.0

	if s.checkUserAgent(userAgent, result) {
		score += 0.5
		confidence += 0.4
	}

	if s.checkHeaders(r, result) {
		score += 0.2
		confidence += 0.2
	}

	if s.checkBehavior(ip, r, result) {
		score += 0.2
		confidence += 0.3
	}

	if s.checkFingerprint(ip, userAgent, additionalData, result) {
		score += 0.1
		confidence += 0.1
	}

	// Check if fingerprint is blacklisted
	s.mu.RLock()
	fingerprintID := s.generateFingerprintID(ip, userAgent, additionalData)
	if fp, exists := s.fingerprints[fingerprintID]; exists && fp.IsBlacklisted {
		result.IsBot = true
		result.ShouldBlock = true
		result.RiskScore = 1.0
		result.Confidence = 0.9
		result.Reasons = append(result.Reasons, "Fingerprint blacklisted")
	}
	s.mu.RUnlock()

	// If not already blocked by blacklist, apply threshold
	if !result.IsBot {
		result.RiskScore = math.Min(score, 1.0)
		result.Confidence = math.Min(confidence, 1.0)

		if result.RiskScore >= 0.7 {
			result.IsBot = true
			result.ShouldBlock = true
			result.ChallengeType = "captcha"
		} else if result.RiskScore >= 0.4 {
			result.IsBot = true
			result.ChallengeType = "js_challenge"
		}
	}

	return result
}

func (s *BotDetectionService) checkUserAgent(userAgent string, result *BotDetectionResult) bool {
	for _, pattern := range s.botPatterns {
		if pattern.MatchString(userAgent) {
			result.Reasons = append(result.Reasons, "Suspicious user agent")
			return true
		}
	}
	return false
}

func (s *BotDetectionService) checkHeaders(r *http.Request, result *BotDetectionResult) bool {
	suspicious := false
	for _, header := range s.headerPatterns {
		if r.Header.Get(header) != "" {
			suspicious = true
			result.Reasons = append(result.Reasons, "Suspicious header: "+header)
			break
		}
	}
	return suspicious
}

func (s *BotDetectionService) checkBehavior(ip string, r *http.Request, result *BotDetectionResult) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	behavior, exists := s.behaviors[ip]
	if !exists {
		behavior = &BotBehaviorData{
			IP:           ip,
			RequestTimes: []time.Time{},
			RequestPaths: []string{},
			Methods:      []string{},
			RequestCount: 0,
			LastActivity: now,
		}
		s.behaviors[ip] = behavior
		if len(s.behaviors) > s.maxBehaviors {
			s.cleanupOldBehaviors()
		}
	}

	behavior.RequestTimes = append(behavior.RequestTimes, now)
	behavior.RequestPaths = append(behavior.RequestPaths, r.URL.Path)
	behavior.Methods = append(behavior.Methods, r.Method)
	behavior.RequestCount++
	behavior.LastActivity = now

	if len(behavior.RequestTimes) > 100 {
		behavior.RequestTimes = behavior.RequestTimes[len(behavior.RequestTimes)-100:]
		behavior.RequestPaths = behavior.RequestPaths[len(behavior.RequestPaths)-100:]
		behavior.Methods = behavior.Methods[len(behavior.Methods)-100:]
	}

	if behavior.RequestCount > 50 {
		avgInterval := s.calculateAvgInterval(behavior.RequestTimes)
		if avgInterval > 0 && avgInterval < 100 {
			result.Reasons = append(result.Reasons, "Unusually frequent requests")
			return true
		}
	}

	return false
}

func (s *BotDetectionService) checkFingerprint(ip string, userAgent string, additionalData map[string]string, result *BotDetectionResult) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	fingerprintID := s.generateFingerprintID(ip, userAgent, additionalData)
	fingerprint, exists := s.fingerprints[fingerprintID]
	if !exists {
		fingerprint = &BotFingerprintData{
			FingerprintID: fingerprintID,
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
			RequestCount:  0,
			RiskScore:     0.0,
			IsBlacklisted: false,
			UserAgent:     userAgent,
			IP:            ip,
		}
		s.fingerprints[fingerprintID] = fingerprint
		if len(s.fingerprints) > s.maxFingerprints {
			s.cleanupOldFingerprints()
		}
	}

	fingerprint.RequestCount++
	fingerprint.LastSeen = time.Now()

	if fingerprint.IsBlacklisted {
		result.Reasons = append(result.Reasons, "Fingerprint blacklisted")
		return true
	}

	return false
}

func (s *BotDetectionService) generateFingerprintID(ip string, userAgent string, additionalData map[string]string) string {
	hasher := sha256.New()
	hasher.Write([]byte(ip))
	hasher.Write([]byte(userAgent))
	for k, v := range additionalData {
		hasher.Write([]byte(k + ":" + v))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *BotDetectionService) calculateAvgInterval(times []time.Time) float64 {
	if len(times) < 2 {
		return 0
	}
	total := 0.0
	for i := 1; i < len(times); i++ {
		total += float64(times[i].Sub(times[i-1]).Milliseconds())
	}
	return total / float64(len(times)-1)
}

func (s *BotDetectionService) cleanupOldFingerprints() {
	cutoff := time.Now().Add(-24 * time.Hour)
	for id, fp := range s.fingerprints {
		if fp.LastSeen.Before(cutoff) {
			delete(s.fingerprints, id)
		}
	}
}

func (s *BotDetectionService) cleanupOldBehaviors() {
	cutoff := time.Now().Add(-24 * time.Hour)
	for ip, bh := range s.behaviors {
		if bh.LastActivity.Before(cutoff) {
			delete(s.behaviors, ip)
		}
	}
}

func (s *BotDetectionService) AddToBlacklist(ip string, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fp := range s.fingerprints {
		if fp.IP == ip {
			fp.IsBlacklisted = true
		}
	}
}

func (s *BotDetectionService) RemoveFromBlacklist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fp := range s.fingerprints {
		if fp.IP == ip {
			fp.IsBlacklisted = false
		}
	}
}

type AutomationToolType string

const (
	ToolSelenium   AutomationToolType = "selenium"
	ToolPhantomJS  AutomationToolType = "phantomjs"
	ToolPuppeteer  AutomationToolType = "puppeteer"
	ToolPlaywright AutomationToolType = "playwright"
	ToolHeadless   AutomationToolType = "headless"
	ToolWebDriver  AutomationToolType = "webdriver"
	ToolGeneric    AutomationToolType = "generic_bot"
)

type AutomationDetectionResult struct {
	ToolType             AutomationToolType    `json:"tool_type"`
	IsAutomated          bool                  `json:"is_automated"`
	Confidence           float64               `json:"confidence"`
	DetectionMethods     []string              `json:"detection_methods"`
	Indicators           []string              `json:"indicators"`
	Score                float64               `json:"score"`
	BehavioralIndicators *BehavioralIndicators `json:"behavioral_indicators"`
}

type BehavioralIndicators struct {
	RequestPattern     string   `json:"request_pattern"`
	NavigationFlow     []string `json:"navigation_flow"`
	MouseMovement      bool     `json:"mouse_movement"`
	KeyboardPatterns   bool     `json:"keyboard_patterns"`
	TimingAnomalies    bool     `json:"timing_anomalies"`
	SessionConsistency float64  `json:"session_consistency"`
	IsHumanLike        bool     `json:"is_human_like"`
}

type EnhancedBotDetectionService struct {
	*BotDetectionService
	behaviorPatterns     map[string][]time.Time
	sessionData          map[string]*BotSessionInfo
	knownAutomationTools map[AutomationToolType]*AutomationToolSignature
}

type BotSessionInfo struct {
	SessionID        string
	StartTime        time.Time
	RequestCount     int
	RequestIntervals []time.Duration
	LastRequestTime  time.Time
	NavigationPaths  []string
}

type AutomationToolSignature struct {
	Type       AutomationToolType
	Name       string
	Patterns   []*regexp.Regexp
	Indicators []string
	Weight     float64
}

func NewEnhancedBotDetectionService() *EnhancedBotDetectionService {
	service := &EnhancedBotDetectionService{
		BotDetectionService:  NewBotDetectionService(),
		behaviorPatterns:     make(map[string][]time.Time),
		sessionData:          make(map[string]*BotSessionInfo),
		knownAutomationTools: make(map[AutomationToolType]*AutomationToolSignature),
	}

	service.initializeAutomationSignatures()
	return service
}

func (s *EnhancedBotDetectionService) initializeAutomationSignatures() {
	s.knownAutomationTools[ToolSelenium] = &AutomationToolSignature{
		Type: ToolSelenium,
		Name: "Selenium",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)selenium`),
			regexp.MustCompile(`(?i)webdriver.*selenium`),
			regexp.MustCompile(`(?i)selenium::webdriver`),
		},
		Indicators: []string{
			"__selenium_evaluate",
			"__webdriver_script_fn",
			"Selenium.prototype",
			"selenium_webdriver",
		},
		Weight: 0.85,
	}

	s.knownAutomationTools[ToolPhantomJS] = &AutomationToolSignature{
		Type: ToolPhantomJS,
		Name: "PhantomJS",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)phantomjs`),
			regexp.MustCompile(`(?i)phantom\.js`),
		},
		Indicators: []string{
			"phantom",
			"callPhantom",
			"_phantom",
		},
		Weight: 0.90,
	}

	s.knownAutomationTools[ToolPuppeteer] = &AutomationToolSignature{
		Type: ToolPuppeteer,
		Name: "Puppeteer",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)puppeteer`),
			regexp.MustCompile(`(?i)headless.*chrome`),
			regexp.MustCompile(`(?i)chrome\-headless`),
		},
		Indicators: []string{
			"$cdc_asdjflasutopfhvcZLmcfl_",
			"$chrome_asyncScriptInfo",
			"__puppeteer_evaluation_script",
		},
		Weight: 0.88,
	}

	s.knownAutomationTools[ToolPlaywright] = &AutomationToolSignature{
		Type: ToolPlaywright,
		Name: "Playwright",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)playwright`),
		},
		Indicators: []string{
			"__playwright__",
			"__pw_api_hooks__",
			"__pw_resume__",
			"__pw_timeout__",
		},
		Weight: 0.87,
	}

	s.knownAutomationTools[ToolHeadless] = &AutomationToolSignature{
		Type: ToolHeadless,
		Name: "Headless Browser",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)headless`),
			regexp.MustCompile(`(?i)chrome\-headless`),
			regexp.MustCompile(`(?i)firefox\-headless`),
		},
		Indicators: []string{
			"navigator.webdriver",
			"headless_detected",
		},
		Weight: 0.75,
	}

	s.knownAutomationTools[ToolWebDriver] = &AutomationToolSignature{
		Type: ToolWebDriver,
		Name: "WebDriver",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)webdriver`),
			regexp.MustCompile(`(?i)chrome\-automation`),
		},
		Indicators: []string{
			"webdriver",
			"__webdriver_evaluate",
			"__driver_evaluate",
		},
		Weight: 0.82,
	}
}

func (s *EnhancedBotDetectionService) DetectAutomationTool(r *http.Request, additionalData map[string]interface{}) *AutomationDetectionResult {
	result := &AutomationDetectionResult{
		DetectionMethods:     make([]string, 0),
		Indicators:           make([]string, 0),
		BehavioralIndicators: &BehavioralIndicators{},
	}

	userAgent := r.UserAgent()
	ip := getClientIP(r)

	s.analyzeHeadersForAutomation(r, result)
	s.analyzeUserAgentForAutomation(userAgent, result)
	s.analyzeAdditionalData(additionalData, result)
	s.analyzeBehavior(ip, result)
	s.calculateFinalScore(result)

	return result
}

func (s *EnhancedBotDetectionService) analyzeHeadersForAutomation(r *http.Request, result *AutomationDetectionResult) {
	type headerInfo struct {
		indicator string
		weight    float64
	}

	automationHeaders := map[string]headerInfo{
		"X-WDAuthToken":      {indicator: "automation_token", weight: 0.9},
		"X-Crawlera-Profile": {indicator: "crawlera_profile", weight: 0.85},
		"X-Crawlera-UA":      {indicator: "crawlera_ua", weight: 0.80},
		"X-Amzn-Trace-Id":    {indicator: "amazon_trace", weight: 0.30},
		"X-Forwarded-For":    {indicator: "proxy_header", weight: 0.40},
		"Via":                {indicator: "proxy_header", weight: 0.35},
		"X-ProxyID":          {indicator: "proxy_id", weight: 0.60},
	}

	for header, info := range automationHeaders {
		value := r.Header.Get(header)
		if value != "" {
			result.Indicators = append(result.Indicators, info.indicator)
			result.Score += info.weight * 100
			result.DetectionMethods = append(result.DetectionMethods, "header:"+header)
		}
	}

	if r.Header.Get("Sec-Ch-Ua-Platform") == "" && r.Header.Get("Sec-Ch-Ua") != "" {
		result.Indicators = append(result.Indicators, "missing_platform_header")
		result.Score += 20
		result.DetectionMethods = append(result.DetectionMethods, "missing_sec_ch_ua_platform")
	}

	if r.Header.Get("Sec-Fetch-Site") == "" {
		result.Indicators = append(result.Indicators, "missing_fetch_site")
		result.Score += 15
	}

	if r.Header.Get("Accept-Language") == "" {
		result.Indicators = append(result.Indicators, "missing_accept_language")
		result.Score += 10
	}
}

func (s *EnhancedBotDetectionService) analyzeUserAgentForAutomation(ua string, result *AutomationDetectionResult) {
	result.DetectionMethods = append(result.DetectionMethods, "user_agent_analysis")

	for toolType, signature := range s.knownAutomationTools {
		for _, pattern := range signature.Patterns {
			if pattern.MatchString(ua) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("ua_match:%s", toolType))
				result.Score += signature.Weight * 100
				result.ToolType = toolType
				result.DetectionMethods = append(result.DetectionMethods, fmt.Sprintf("pattern:%s", toolType))
				break
			}
		}
	}

	commonBotPatterns := []string{
		`curl/\d+\.\d+`,
		`wget/\d+\.\d+`,
		`python-requests/\d+\.\d+`,
		`scrapy/\d+\.\d+`,
		`apache-httpclient/\d+`,
		`java/\d+\.\d+`,
		`go-http-client`,
		`node-fetch/\d+`,
		`axios/\d+`,
	}

	for _, pattern := range commonBotPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(ua) {
			result.Indicators = append(result.Indicators, "known_library")
			result.Score += 60
			result.DetectionMethods = append(result.DetectionMethods, "known_library:"+pattern)
		}
	}

	versionPatterns := regexp.MustCompile(`(?:chrome|firefox|safari|edge)/(\d+)\.(\d+)\.(\d+)`)
	matches := versionPatterns.FindStringSubmatch(ua)
	if matches != nil {
		majorVersion := 0
		fmt.Sscanf(matches[1], "%d", &majorVersion)
		if majorVersion > 0 && majorVersion < 50 {
			result.Indicators = append(result.Indicators, "old_browser_version")
			result.Score += 25
		}
	}
}

func (s *EnhancedBotDetectionService) analyzeAdditionalData(data map[string]interface{}, result *AutomationDetectionResult) {
	if data == nil {
		return
	}

	result.DetectionMethods = append(result.DetectionMethods, "additional_data_analysis")

	if navigatorProps, ok := data["navigator_properties"].(map[string]interface{}); ok {
		if webdriver, ok := navigatorProps["webdriver"].(bool); ok && webdriver {
			result.Indicators = append(result.Indicators, "navigator.webdriver=true")
			result.Score += 80
			result.DetectionMethods = append(result.DetectionMethods, "navigator_webdriver")
		}

		if plugins, ok := navigatorProps["plugins"].([]interface{}); ok && len(plugins) == 0 {
			result.Indicators = append(result.Indicators, "no_plugins")
			result.Score += 30
		}

		if languages, ok := navigatorProps["languages"].([]interface{}); ok && len(languages) == 0 {
			result.Indicators = append(result.Indicators, "no_languages")
			result.Score += 25
		}
	}

	if canvasData, ok := data["canvas_fingerprint"].(string); ok && canvasData != "" {
		result.DetectionMethods = append(result.DetectionMethods, "canvas_analysis")
		canvasHash := fmt.Sprintf("%x", sha256.Sum256([]byte(canvasData)))
		if s.isCommonCanvasHash(canvasHash) {
			result.Indicators = append(result.Indicators, "common_canvas_hash")
			result.Score += 20
		}
	}

	if webglData, ok := data["webgl_renderer"].(string); ok {
		if strings.Contains(strings.ToLower(webglData), "swiftshader") ||
			strings.Contains(strings.ToLower(webglData), "llvmpipe") ||
			strings.Contains(strings.ToLower(webglData), "software") {
			result.Indicators = append(result.Indicators, "software_renderer")
			result.Score += 45
			result.DetectionMethods = append(result.DetectionMethods, "software_webgl")
		}
	}

	if timingData, ok := data["timing_data"].(map[string]interface{}); ok {
		if loadTime, ok := timingData["load_time"].(float64); ok {
			if loadTime < 100 {
				result.Indicators = append(result.Indicators, "fast_load_time")
				result.Score += 35
			}
		}
	}
}

func (s *EnhancedBotDetectionService) analyzeBehavior(ip string, result *AutomationDetectionResult) {
	result.DetectionMethods = append(result.DetectionMethods, "behavior_analysis")

	s.mu.Lock()
	defer s.mu.Unlock()

	timingData := s.behaviorPatterns[ip]
	if len(timingData) < 5 {
		result.BehavioralIndicators.IsHumanLike = true
		return
	}

	intervals := make([]time.Duration, 0)
	for i := 1; i < len(timingData); i++ {
		interval := timingData[i].Sub(timingData[i-1])
		intervals = append(intervals, interval)
	}

	avgInterval := s.calculateAverageInterval(intervals)
	variance := s.calculateIntervalVariance(intervals, avgInterval)

	if variance < 50*time.Millisecond && avgInterval < 2*time.Second {
		result.BehavioralIndicators.TimingAnomalies = true
		result.BehavioralIndicators.RequestPattern = "too_regular"
		result.Score += 40
		result.Indicators = append(result.Indicators, "regular_timing_pattern")
	}

	if avgInterval < 500*time.Millisecond {
		result.Score += 30
		result.Indicators = append(result.Indicators, "fast_requests")
	}

	if len(intervals) > 0 {
		maxInterval := intervals[0]
		minInterval := intervals[0]
		for _, interval := range intervals {
			if interval > maxInterval {
				maxInterval = interval
			}
			if interval < minInterval {
				minInterval = interval
			}
		}

		ratio := float64(maxInterval) / float64(minInterval)
		if ratio < 1.5 && avgInterval < 3*time.Second {
			result.BehavioralIndicators.IsHumanLike = false
			result.Score += 35
			result.Indicators = append(result.Indicators, "mechanical_behavior")
		}
	}

	result.BehavioralIndicators.IsHumanLike = result.Score < 50
}

func (s *EnhancedBotDetectionService) calculateAverageInterval(intervals []time.Duration) time.Duration {
	if len(intervals) == 0 {
		return 0
	}

	var sum time.Duration
	for _, interval := range intervals {
		sum += interval
	}

	return sum / time.Duration(len(intervals))
}

func (s *EnhancedBotDetectionService) calculateIntervalVariance(intervals []time.Duration, avg time.Duration) time.Duration {
	if len(intervals) == 0 {
		return 0
	}

	var sumSq float64
	for _, interval := range intervals {
		diff := float64(interval - avg)
		sumSq += diff * diff
	}

	variance := sumSq / float64(len(intervals))
	return time.Duration(math.Sqrt(variance))
}

func (s *EnhancedBotDetectionService) calculateFinalScore(result *AutomationDetectionResult) {
	result.Score = math.Min(result.Score, 100)
	result.Confidence = result.Score / 100.0
	result.IsAutomated = result.Score >= 60

	if len(result.Indicators) >= 5 && result.Score >= 40 {
		result.IsAutomated = true
		result.Confidence = math.Min(result.Confidence+0.2, 1.0)
	}

	if result.ToolType == "" && result.IsAutomated {
		result.ToolType = ToolGeneric
	}
}

func (s *EnhancedBotDetectionService) isCommonCanvasHash(hash string) bool {
	commonHashes := map[string]bool{
		"a1b2c3d4e5f6": true,
		"1234567890ab": true,
		"ffffffffffff": true,
		"000000000000": true,
	}
	return commonHashes[hash]
}

func (s *EnhancedBotDetectionService) RecordRequest(ip string, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.behaviorPatterns[ip] = append(s.behaviorPatterns[ip], now)

	if len(s.behaviorPatterns[ip]) > 1000 {
		s.behaviorPatterns[ip] = s.behaviorPatterns[ip][len(s.behaviorPatterns[ip])-500:]
	}

	if _, exists := s.sessionData[ip]; !exists {
		s.sessionData[ip] = &BotSessionInfo{
			SessionID:        fmt.Sprintf("session_%s_%d", ip, now.Unix()),
			StartTime:        now,
			RequestCount:     0,
			RequestIntervals: []time.Duration{},
			NavigationPaths:  []string{},
		}
	}

	session := s.sessionData[ip]
	session.RequestCount++
	session.LastRequestTime = now
	session.NavigationPaths = append(session.NavigationPaths, path)

	if len(session.NavigationPaths) > 100 {
		session.NavigationPaths = session.NavigationPaths[len(session.NavigationPaths)-50:]
	}
}

func (s *EnhancedBotDetectionService) GetSessionInfo(ip string) *BotSessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if session, exists := s.sessionData[ip]; exists {
		return session
	}

	return nil
}

func (s *EnhancedBotDetectionService) DetectAutomatedScriptPattern(ip string) (bool, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session := s.sessionData[ip]
	if session == nil || session.RequestCount < 10 {
		return false, 0
	}

	score := 0.0

	if len(session.NavigationPaths) > 0 {
		uniquePaths := make(map[string]bool)
		for _, path := range session.NavigationPaths {
			uniquePaths[path] = true
		}

		pathDiversity := float64(len(uniquePaths)) / float64(len(session.NavigationPaths))
		if pathDiversity < 0.1 {
			score += 40
		} else if pathDiversity < 0.3 {
			score += 20
		}
	}

	if len(session.RequestIntervals) > 5 {
		avgInterval := s.calculateAverageInterval(session.RequestIntervals)
		if avgInterval < 1*time.Second {
			score += 35
		} else if avgInterval < 3*time.Second {
			score += 15
		}
	}

	sessionDuration := time.Since(session.StartTime)
	if sessionDuration > 0 && session.RequestCount > 0 {
		requestsPerMinute := float64(session.RequestCount) / sessionDuration.Minutes()
		if requestsPerMinute > 100 {
			score += 40
		} else if requestsPerMinute > 50 {
			score += 20
		}
	}

	return score >= 60, math.Min(score, 100)
}
