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

type AIBotDetectionService struct {
	*EnhancedBotDetectionService
	mlModel           *BotMLModel
	featureExtractor  *BotFeatureExtractor
	trainingData      []BotTrainingSample
	modelVersion      string
	lastUpdateTime    time.Time
	updateInterval    time.Duration
}

type BotMLModel struct {
	weights       map[string]float64
	bias          float64
	featureNames  []string
	modelType     string
	trainingEpochs int
	learningRate  float64
}

type BotTrainingSample struct {
	Features      map[string]float64
	Label         bool
	BotType       string
	Severity      string
	CollectionTime time.Time
}

type BotFeatureExtractor struct {
	featureWeights map[string]float64
	normalizers    map[string]float64
	windowSize     int
}

type BotDetectionFeatures struct {
	TimingFeatures     TimingFeatureSet
	BehavioralFeatures BehavioralFeatureSet
	EnvironmentalFeatures EnvironmentalFeatureSet
	NetworkFeatures    NetworkFeatureSet
}

type TimingFeatureSet struct {
	AvgRequestInterval   float64
	IntervalVariance     float64
	MaxInterval          float64
	MinInterval          float64
	RequestRate          float64
	TimingEntropy        float64
	IsPeriodic           bool
}

type BehavioralFeatureSet struct {
	NavigationDiversity  float64
	ClickPatternScore    float64
	MouseMovementEntropy float64
	KeystrokeDynamics    float64
	SessionLength        int
	PageViewDistribution map[string]int
}

type EnvironmentalFeatureSet struct {
	CanvasEntropy       float64
	WebGLConsistency    float64
	AudioFingerprint    string
	FontDiversity       float64
	PluginCount         int
	ScreenResolution    string
	TimezoneConsistency bool
}

type NetworkFeatureSet struct {
	IPReputationScore   float64
	ASNType             string
	RequestOrigin       string
	HeaderCompleteness  float64
	ProtocolVersion     string
}

func NewAIBotDetectionService() *AIBotDetectionService {
	service := &AIBotDetectionService{
		EnhancedBotDetectionService: NewEnhancedBotDetectionService(),
		mlModel:                   NewBotMLModel(),
		featureExtractor:           NewBotFeatureExtractor(),
		trainingData:               []BotTrainingSample{},
		modelVersion:               "v3.0",
		lastUpdateTime:             time.Now(),
		updateInterval:             24 * time.Hour,
	}
	service.initializeMLModel()
	return service
}

func NewBotMLModel() *BotMLModel {
	return &BotMLModel{
		weights:        make(map[string]float64),
		bias:           0.5,
		featureNames:   []string{},
		modelType:      "logistic_regression",
		trainingEpochs: 100,
		learningRate:   0.01,
	}
}

func NewBotFeatureExtractor() *BotFeatureExtractor {
	return &BotFeatureExtractor{
		featureWeights: map[string]float64{
			"timing_regularity":     0.15,
			"request_rate":          0.12,
			"navigation_pattern":    0.14,
			"session_duration":      0.10,
			"click_diversity":        0.11,
			"mouse_entropy":          0.09,
			"keystroke_pattern":      0.08,
			"canvas_fingerprint":     0.10,
			"webgl_consistency":     0.08,
			"audio_fingerprint":     0.06,
			"font_diversity":        0.07,
			"ip_reputation":         0.09,
			"header_completeness":   0.08,
		},
		normalizers: map[string]float64{
			"timing_regularity":     100.0,
			"request_rate":          1000.0,
			"navigation_pattern":    1.0,
			"session_duration":       3600.0,
			"click_diversity":        1.0,
			"mouse_entropy":         10.0,
			"keystroke_pattern":     1000.0,
			"canvas_fingerprint":    1.0,
			"webgl_consistency":     1.0,
			"audio_fingerprint":     1.0,
			"font_diversity":        50.0,
			"ip_reputation":         100.0,
			"header_completeness":   1.0,
		},
		windowSize: 100,
	}
}

func (s *AIBotDetectionService) initializeMLModel() {
	s.mlModel.weights = map[string]float64{
		"timing_regularity":      0.15,
		"request_rate":           0.18,
		"navigation_pattern":     0.14,
		"session_duration":       0.10,
		"click_diversity":        0.11,
		"mouse_entropy":          0.09,
		"keystroke_pattern":      0.08,
		"canvas_fingerprint":     0.10,
		"webgl_consistency":      0.08,
		"audio_fingerprint":      0.06,
		"font_diversity":         0.07,
		"ip_reputation":          0.09,
		"header_completeness":    0.08,
	}
	
	s.mlModel.featureNames = []string{
		"timing_regularity",
		"request_rate",
		"navigation_pattern",
		"session_duration",
		"click_diversity",
		"mouse_entropy",
		"keystroke_pattern",
		"canvas_fingerprint",
		"webgl_consistency",
		"audio_fingerprint",
		"font_diversity",
		"ip_reputation",
		"header_completeness",
	}
}

func (s *AIBotDetectionService) DetectBotV3(r *http.Request, behaviorData *BotBehaviorData, envData map[string]interface{}) *BotDetectionV3Result {
	result := &BotDetectionV3Result{
		Timestamp:         time.Now(),
		DetectionVersion:  "3.0",
		Features:          &BotDetectionFeatures{},
		MLPrediction:      &MLPrediction{},
		Reasons:           []string{},
	}
	
	timingFeatures := s.extractTimingFeatures(r, behaviorData)
	result.Features.TimingFeatures = timingFeatures
	
	behavioralFeatures := s.extractBehavioralFeatures(behaviorData)
	result.Features.BehavioralFeatures = behavioralFeatures
	
	envFeatures := s.extractEnvironmentalFeatures(envData)
	result.Features.EnvironmentalFeatures = envFeatures
	
	networkFeatures := s.extractNetworkFeatures(r)
	result.Features.NetworkFeatures = networkFeatures
	
	features := s.featureExtractor.ExtractFeatureVector(result.Features)
	mlResult := s.mlModel.Predict(features)
	result.MLPrediction = mlResult
	
	classicalResult := s.DetectBot(r, nil)
	if classicalResult.IsBot {
		result.ClassicalDetection = true
		result.ClassicalScore = classicalResult.RiskScore
		result.Reasons = append(result.Reasons, classicalResult.Reasons...)
	}
	
	s.calculateFinalScore(result)
	
	return result
}

type BotDetectionV3Result struct {
	Timestamp          time.Time
	DetectionVersion   string
	Features           *BotDetectionFeatures
	MLPrediction       *MLPrediction
	ClassicalDetection bool
	ClassicalScore     float64
	FinalScore         float64
	IsBot              bool
	ShouldBlock        bool
	ChallengeType      string
	Confidence         float64
	BotType            string
	Reasons            []string
}

type MLPrediction struct {
	Probability     float64
	IsBot           bool
	Confidence      float64
	BotType         string
	AnomalyScore    float64
	FeatureImportance map[string]float64
	Explanation     []string
}

func (f *BotFeatureExtractor) ExtractFeatureVector(detection *BotDetectionFeatures) map[string]float64 {
	features := make(map[string]float64)
	
	features["timing_regularity"] = f.normalize(
		detection.TimingFeatures.IntervalVariance / (detection.TimingFeatures.AvgRequestInterval + 1),
		"timing_regularity")
	
	features["request_rate"] = f.normalize(
		detection.TimingFeatures.RequestRate,
		"request_rate")
	
	features["navigation_pattern"] = f.normalize(
		detection.BehavioralFeatures.NavigationDiversity,
		"navigation_pattern")
	
	features["session_duration"] = f.normalize(
		float64(detection.BehavioralFeatures.SessionLength),
		"session_duration")
	
	features["click_diversity"] = f.normalize(
		detection.BehavioralFeatures.ClickPatternScore,
		"click_diversity")
	
	features["mouse_entropy"] = f.normalize(
		detection.BehavioralFeatures.MouseMovementEntropy,
		"mouse_entropy")
	
	features["keystroke_pattern"] = f.normalize(
		detection.BehavioralFeatures.KeystrokeDynamics,
		"keystroke_pattern")
	
	features["canvas_fingerprint"] = f.normalize(
		detection.EnvironmentalFeatures.CanvasEntropy,
		"canvas_fingerprint")
	
	features["webgl_consistency"] = f.normalize(
		detection.EnvironmentalFeatures.WebGLConsistency,
		"webgl_consistency")
	
	features["font_diversity"] = f.normalize(
		detection.EnvironmentalFeatures.FontDiversity,
		"font_diversity")
	
	features["ip_reputation"] = f.normalize(
		detection.NetworkFeatures.IPReputationScore,
		"ip_reputation")
	
	features["header_completeness"] = f.normalize(
		detection.NetworkFeatures.HeaderCompleteness,
		"header_completeness")
	
	return features
}

func (f *BotFeatureExtractor) normalize(value float64, featureName string) float64 {
	if normalizer, exists := f.normalizers[featureName]; exists && normalizer > 0 {
		return math.Min(value/normalizer, 1.0)
	}
	return value
}

func (m *BotMLModel) Predict(features map[string]float64) *MLPrediction {
	prediction := &MLPrediction{
		FeatureImportance: make(map[string]float64),
		Explanation:      []string{},
	}
	
	score := m.bias
	
	for feature, value := range features {
		if weight, exists := m.weights[feature]; exists {
			contribution := value * weight
			score += contribution
			prediction.FeatureImportance[feature] = contribution
		}
	}
	
	prediction.Probability = 1.0 / (1.0 + math.Exp(-score))
	prediction.IsBot = prediction.Probability > 0.7
	prediction.Confidence = math.Abs(prediction.Probability - 0.5) * 2
	prediction.AnomalyScore = prediction.Probability
	
	if prediction.IsBot {
		prediction.BotType = m.classifyBot(features)
	}
	
	sortFeaturesByImportance(prediction.FeatureImportance, prediction.Explanation)
	
	return prediction
}

func (m *BotMLModel) classifyBot(features map[string]float64) string {
	scores := map[string]float64{
		"automated_scraper":  0,
		"headless_browser":   0,
		"credential_stuffer": 0,
		"ddos_bot":          0,
		"scraping_bot":      0,
	}
	
	if features["timing_regularity"] > 0.8 && features["request_rate"] > 0.5 {
		scores["automated_scraper"] += 0.4
		scores["scraping_bot"] += 0.3
	}
	
	if features["keystroke_pattern"] > 0.9 {
		scores["automated_scraper"] += 0.3
	}
	
	if features["navigation_pattern"] < 0.2 {
		scores["scraping_bot"] += 0.4
	}
	
	if features["webgl_consistency"] < 0.3 || features["canvas_fingerprint"] < 0.2 {
		scores["headless_browser"] += 0.5
	}
	
	if features["request_rate"] > 0.9 {
		scores["ddos_bot"] += 0.5
	}
	
	if features["ip_reputation"] < 0.3 {
		scores["credential_stuffer"] += 0.3
		scores["ddos_bot"] += 0.2
	}
	
	bestType := "unknown_bot"
	bestScore := 0.0
	for botType, score := range scores {
		if score > bestScore {
			bestScore = score
			bestType = botType
		}
	}
	
	return bestType
}

func sortFeaturesByImportance(importance map[string]float64, explanations []string) {
	type kv struct {
		Key   string
		Value float64
	}
	
	var ss []kv
	for k, v := range importance {
		ss = append(ss, kv{k, v})
	}
	
	for i := 0; i < len(ss); i++ {
		for j := i + 1; j < len(ss); j++ {
			if ss[j].Value > ss[i].Value {
				ss[i], ss[j] = ss[j], ss[i]
			}
		}
	}
	
	for i, kv := range ss {
		if i < 3 && kv.Value > 0.05 {
			explanations = append(explanations, fmt.Sprintf("High %s feature contribution: %.2f", kv.Key, kv.Value))
		}
	}
}

func (s *AIBotDetectionService) extractTimingFeatures(r *http.Request, behaviorData *BotBehaviorData) TimingFeatureSet {
	features := TimingFeatureSet{}
	
	if behaviorData != nil && len(behaviorData.RequestTimes) >= 2 {
		intervals := []float64{}
		for i := 1; i < len(behaviorData.RequestTimes); i++ {
			interval := behaviorData.RequestTimes[i].Sub(behaviorData.RequestTimes[i-1]).Milliseconds()
			intervals = append(intervals, float64(interval))
		}
		
		features.AvgRequestInterval = calculateMean(intervals)
		features.IntervalVariance = calculateVariance(intervals, features.AvgRequestInterval)
		features.MaxInterval = s.calculateMax(intervals)
		features.MinInterval = s.calculateMin(intervals)
		
		if len(behaviorData.RequestTimes) > 0 {
			totalDuration := behaviorData.RequestTimes[len(behaviorData.RequestTimes)-1].Sub(behaviorData.RequestTimes[0]).Seconds()
			if totalDuration > 0 {
				features.RequestRate = float64(len(behaviorData.RequestTimes)) / totalDuration
			}
		}
		
		features.TimingEntropy = s.calculateEntropy(intervals)
		
		if features.IntervalVariance < 100 && features.AvgRequestInterval < 2000 {
			features.IsPeriodic = true
		}
	}
	
	return features
}

func (s *AIBotDetectionService) extractBehavioralFeatures(behaviorData *BotBehaviorData) BehavioralFeatureSet {
	features := BehavioralFeatureSet{}
	
	if behaviorData != nil {
		features.SessionLength = behaviorData.RequestCount
		
		if len(behaviorData.RequestPaths) > 0 {
			uniquePaths := make(map[string]bool)
			for _, path := range behaviorData.RequestPaths {
				uniquePaths[path] = true
			}
			features.NavigationDiversity = float64(len(uniquePaths)) / float64(len(behaviorData.RequestPaths))
		}
		
		features.PageViewDistribution = make(map[string]int)
		for _, path := range behaviorData.RequestPaths {
			features.PageViewDistribution[path]++
		}
	}
	
	features.ClickPatternScore = 0.5
	features.MouseMovementEntropy = 5.0
	features.KeystrokeDynamics = 100.0
	
	return features
}

func (s *AIBotDetectionService) extractEnvironmentalFeatures(envData map[string]interface{}) EnvironmentalFeatureSet {
	features := EnvironmentalFeatureSet{}
	
	if envData != nil {
		if canvasEntropy, ok := envData["canvas_entropy"].(float64); ok {
			features.CanvasEntropy = canvasEntropy
		} else {
			features.CanvasEntropy = 0.5
		}
		
		if webglConsistency, ok := envData["webgl_consistency"].(float64); ok {
			features.WebGLConsistency = webglConsistency
		} else {
			features.WebGLConsistency = 0.5
		}
		
		if audioFingerprint, ok := envData["audio_fingerprint"].(string); ok {
			features.AudioFingerprint = audioFingerprint
		}
		
		if fontDiversity, ok := envData["font_diversity"].(float64); ok {
			features.FontDiversity = fontDiversity
		} else {
			features.FontDiversity = 10.0
		}
		
		if pluginCount, ok := envData["plugin_count"].(int); ok {
			features.PluginCount = pluginCount
		} else {
			features.PluginCount = 0
		}
		
		if screenRes, ok := envData["screen_resolution"].(string); ok {
			features.ScreenResolution = screenRes
		}
		
		if timezoneConsist, ok := envData["timezone_consistency"].(bool); ok {
			features.TimezoneConsistency = timezoneConsist
		}
	}
	
	return features
}

func (s *AIBotDetectionService) extractNetworkFeatures(r *http.Request) NetworkFeatureSet {
	features := NetworkFeatureSet{}
	
	features.IPReputationScore = 1.0
	features.ASNType = "normal"
	features.RequestOrigin = "unknown"
	
	requiredHeaders := []string{"User-Agent", "Accept", "Accept-Language", "Accept-Encoding"}
	presentHeaders := 0
	for _, header := range requiredHeaders {
		if r.Header.Get(header) != "" {
			presentHeaders++
		}
	}
	features.HeaderCompleteness = float64(presentHeaders) / float64(len(requiredHeaders))
	
	if r.Header.Get("X-Forwarded-For") != "" {
		features.RequestOrigin = "proxy"
		features.IPReputationScore -= 0.2
	}
	
	if strings.Contains(strings.ToLower(r.UserAgent()), "bot") ||
		strings.Contains(strings.ToLower(r.UserAgent()), "crawler") ||
		strings.Contains(strings.ToLower(r.UserAgent()), "spider") {
		features.IPReputationScore -= 0.3
	}
	
	return features
}

func (s *AIBotDetectionService) calculateFinalScore(result *BotDetectionV3Result) {
	mlWeight := 0.6
	classicalWeight := 0.4
	
	mlScore := result.MLPrediction.Probability * 100
	finalScore := mlScore * mlWeight
	
	if result.ClassicalDetection {
		finalScore += result.ClassicalScore * classicalWeight
	}
	
	result.FinalScore = math.Min(finalScore, 100)
	result.Confidence = result.MLPrediction.Confidence
	
	if result.FinalScore >= 70 {
		result.IsBot = true
		result.ShouldBlock = true
		result.ChallengeType = "captcha"
	} else if result.FinalScore >= 50 {
		result.IsBot = true
		result.ChallengeType = "js_challenge"
	} else if result.FinalScore >= 30 {
		result.ChallengeType = "monitoring"
	}
	
	result.BotType = result.MLPrediction.BotType
	
	if result.ClassicalDetection && !result.IsBot {
		for _, reason := range result.Reasons {
			if strings.Contains(strings.ToLower(reason), "blacklist") {
				result.IsBot = true
				result.ShouldBlock = true
				result.ChallengeType = "captcha"
				break
			}
		}
	}
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return sumSq / float64(len(values))
}

func (s *AIBotDetectionService) calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (s *AIBotDetectionService) calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (s *AIBotDetectionService) calculateEntropy(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	freq := make(map[float64]int)
	for _, v := range values {
		freq[v]++
	}
	
	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / float64(len(values))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	
	return entropy
}

func (s *AIBotDetectionService) TrainModel(samples []BotTrainingSample) error {
	s.trainingData = append(s.trainingData, samples...)
	
	for _, sample := range samples {
		features := s.convertSampleToFeatures(sample)
		s.mlModel.Train(features, sample.Label)
	}
	
	s.lastUpdateTime = time.Now()
	return nil
}

func (s *BotMLModel) Train(features map[string]float64, label bool) {
	featureValue := 0.0
	for feature, value := range features {
		if weight, exists := s.weights[feature]; exists {
			featureValue += value * weight
		}
	}
	
	featureValue += s.bias
	prediction := 1.0 / (1.0 + math.Exp(-featureValue))
	
	labelValue := 0.0
	if label {
		labelValue = 1.0
	}
	
	error := labelValue - prediction
	
	for feature, value := range features {
		if _, exists := s.weights[feature]; exists {
			s.weights[feature] += s.learningRate * error * value
		}
	}
	
	s.bias += s.learningRate * error
}

func (s *AIBotDetectionService) convertSampleToFeatures(sample BotTrainingSample) map[string]float64 {
	features := make(map[string]float64)
	for k, v := range sample.Features {
		features[k] = v
	}
	return features
}

func (s *AIBotDetectionService) ShouldUpdateModel() bool {
	return time.Since(s.lastUpdateTime) > s.updateInterval && len(s.trainingData) > 100
}

func (s *AIBotDetectionService) UpdateModel() error {
	if !s.ShouldUpdateModel() {
		return nil
	}
	
	s.mlModel.trainingEpochs += 10
	
	s.lastUpdateTime = time.Now()
	return nil
}
