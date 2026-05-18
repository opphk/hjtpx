package service

import (
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AutomationDetectionService struct {
	seleniumPatterns     []*regexp.Regexp
	phantomJSPatterns    []*regexp.Regexp
	puppeteerPatterns    []*regexp.Regexp
	playwrightPatterns   []*regexp.Regexp
	headlessPatterns     []*regexp.Regexp
	webdriverPatterns    []*regexp.Regexp
	automationIndicators []*regexp.Regexp

	mu                    sync.RWMutex
	behaviorPatterns      map[string]*AutomationBehavior
	debuggerDetection     map[string]*DebuggerDetectionRecord
	sessionBehavior       map[string]*SessionBehaviorAnalysis
}

type AutomationBehavior struct {
	IP                  string
	RequestTimestamps   []time.Time
	RequestPaths        []string
	RequestMethods      []string
	IntervalVariance    float64
	RequestCount        int
	LastActivity        time.Time
}

type DebuggerDetectionRecord struct {
	IP           string
	DetectedAt   time.Time
	Evidence     []string
	Confidence   float64
}

type SessionBehaviorAnalysis struct {
	SessionID             string
	StartTime             time.Time
	TotalRequests         int
	UniquePaths           map[string]bool
	RequestIntervals      []time.Duration
	PathTransitionMatrix  map[string]map[string]int
	IsSuspicious          bool
	SuspiciousScore       float64
}

type AutoDetectionResult struct {
	IsAutomated           bool                `json:"is_automated"`
	ToolType              string              `json:"tool_type"`
	Confidence            float64             `json:"confidence"`
	Evidence              []string            `json:"evidence"`
	RiskScore             float64             `json:"risk_score"`
	AutoBehavioralIndicators *AutoBehavioralIndicators `json:"behavioral_indicators"`
	DebuggerDetected      bool                `json:"debugger_detected"`
	HeadlessDetected      bool                `json:"headless_detected"`
}

type AutoBehavioralIndicators struct {
	RequestPattern     string  `json:"request_pattern"`
	RequestFrequency   float64 `json:"request_frequency"`
	IntervalRegularity float64 `json:"interval_regularity"`
	PathDiversity     float64 `json:"path_diversity"`
	IsHumanLike       bool    `json:"is_human_like"`
	TimingAnomalies   bool    `json:"timing_anomalies"`
	SuspiciousScore   float64 `json:"suspicious_score"`
}

func NewAutomationDetectionService() *AutomationDetectionService {
	service := &AutomationDetectionService{
		seleniumPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)selenium`),
			regexp.MustCompile(`(?i)webdriver.*selenium`),
			regexp.MustCompile(`(?i)selenium::webdriver`),
			regexp.MustCompile(`(?i)__selenium`),
			regexp.MustCompile(`(?i)callSelenium`),
			regexp.MustCompile(`(?i)window\.__selenium`),
			regexp.MustCompile(`(?i)document\.__selenium`),
			regexp.MustCompile(`(?i)seleniumObject`),
			regexp.MustCompile(`(?i)selenium-webdriver`),
			regexp.MustCompile(`(?i)Selenium\.prototype`),
		},
		phantomJSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)phantomjs`),
			regexp.MustCompile(`(?i)phantom\.js`),
			regexp.MustCompile(`(?i)callPhantom`),
			regexp.MustCompile(`(?i)window\._phantom`),
			regexp.MustCompile(`(?i)window\.phantom`),
			regexp.MustCompile(`(?i)__phantom`),
		},
		puppeteerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)puppeteer`),
			regexp.MustCompile(`(?i)headless.*chrome`),
			regexp.MustCompile(`(?i)chrome-headless`),
			regexp.MustCompile(`(?i)window\.__puppeteer`),
			regexp.MustCompile(`(?i)\$cdc_asdjflasutopfhvcZLmcfl_`),
			regexp.MustCompile(`(?i)\$chrome_asyncScriptInfo`),
			regexp.MustCompile(`(?i)__puppeteer_evaluation_script`),
			regexp.MustCompile(`(?i)puppeteer-core`),
			regexp.MustCompile(`(?i)pptr:`),
		},
		playwrightPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)playwright`),
			regexp.MustCompile(`(?i)window\.__playwright__`),
			regexp.MustCompile(`(?i)__pw_api_hooks__`),
			regexp.MustCompile(`(?i)__pw_resume__`),
			regexp.MustCompile(`(?i)__pw_timeout__`),
			regexp.MustCompile(`(?i)playwright-core`),
			regexp.MustCompile(`(?i)pw:`),
		},
		headlessPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)headless`),
			regexp.MustCompile(`(?i)chrome-headless`),
			regexp.MustCompile(`(?i)firefox-headless`),
			regexp.MustCompile(`(?i)headlessfirefox`),
			regexp.MustCompile(`(?i)headlessbrowser`),
		},
		webdriverPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)webdriver`),
			regexp.MustCompile(`(?i)chrome-automation`),
			regexp.MustCompile(`(?i)navigator\.webdriver`),
			regexp.MustCompile(`(?i)__webdriver_evaluate`),
			regexp.MustCompile(`(?i)__driver_evaluate`),
			regexp.MustCompile(`(?i)__wd_`),
		},
		automationIndicators: []*regexp.Regexp{
			regexp.MustCompile(`(?i)automation`),
			regexp.MustCompile(`(?i)test.*automation`),
			regexp.MustCompile(`(?i)bot.*framework`),
			regexp.MustCompile(`(?i)crawler`),
			regexp.MustCompile(`(?i)scraper`),
		},
		behaviorPatterns:    make(map[string]*AutomationBehavior),
		debuggerDetection:   make(map[string]*DebuggerDetectionRecord),
		sessionBehavior:     make(map[string]*SessionBehaviorAnalysis),
	}
	return service
}

func (s *AutomationDetectionService) DetectAutomationTool(r *http.Request, frontendData map[string]interface{}) *AutoDetectionResult {
	result := &AutoDetectionResult{
		IsAutomated:           false,
		ToolType:              "",
		Confidence:            0.0,
		Evidence:              []string{},
		RiskScore:             0.0,
		AutoBehavioralIndicators: &AutoBehavioralIndicators{IsHumanLike: true},
		DebuggerDetected:      false,
		HeadlessDetected:      false,
	}

	userAgent := r.UserAgent()
	ip := getClientIP(r)

	score := 0.0
	confidence := 0.0

	if s.detectSelenium(userAgent, frontendData, result) {
		score += 35
		confidence += 0.35
	}

	if s.detectPhantomJS(userAgent, frontendData, result) {
		score += 40
		confidence += 0.40
	}

	if s.detectPuppeteer(userAgent, frontendData, result) {
		score += 38
		confidence += 0.38
	}

	if s.detectPlaywright(userAgent, frontendData, result) {
		score += 37
		confidence += 0.37
	}

	if s.detectHeadlessBrowser(userAgent, frontendData, result) {
		score += 30
		confidence += 0.30
		result.HeadlessDetected = true
	}

	if s.detectWebDriver(userAgent, frontendData, result) {
		score += 32
		confidence += 0.32
	}

	if s.detectDebugger(r, frontendData, ip, result) {
		score += 25
		confidence += 0.25
		result.DebuggerDetected = true
	}

	behaviorResult := s.analyzeBehavioralPatterns(ip, r)
	result.AutoBehavioralIndicators = behaviorResult
	if !behaviorResult.IsHumanLike {
		score += behaviorResult.SuspiciousScore * 0.5
	}

	result.RiskScore = math.Min(score, 100)
	result.Confidence = math.Min(confidence, 1.0)
	result.IsAutomated = result.RiskScore >= 60

	if result.ToolType == "" && result.IsAutomated {
		result.ToolType = "generic_automation"
	}

	return result
}

func (s *AutomationDetectionService) detectSelenium(userAgent string, data map[string]interface{}, result *AutoDetectionResult) bool {
	found := false

	for _, pattern := range s.seleniumPatterns {
		if pattern.MatchString(userAgent) {
			result.Evidence = append(result.Evidence, "Selenium detected in User-Agent")
			result.ToolType = "selenium"
			found = true
			break
		}
	}

	if data != nil {
		if navigator, ok := data["navigator"].(map[string]interface{}); ok {
			if webdriver, ok := navigator["webdriver"].(bool); ok && webdriver {
				result.Evidence = append(result.Evidence, "navigator.webdriver is true")
				result.ToolType = "selenium"
				found = true
			}
		}

		if seleniumObj, ok := data["selenium_object"]; ok && seleniumObj != nil {
			result.Evidence = append(result.Evidence, "Selenium object detected")
			result.ToolType = "selenium"
			found = true
		}
	}

	return found
}

func (s *AutomationDetectionService) detectPhantomJS(userAgent string, data map[string]interface{}, result *AutoDetectionResult) bool {
	found := false

	for _, pattern := range s.phantomJSPatterns {
		if pattern.MatchString(userAgent) {
			result.Evidence = append(result.Evidence, "PhantomJS detected in User-Agent")
			result.ToolType = "phantomjs"
			found = true
			break
		}
	}

	if data != nil {
		if phantom, ok := data["phantom"]; ok && phantom != nil {
			result.Evidence = append(result.Evidence, "PhantomJS global object detected")
			result.ToolType = "phantomjs"
			found = true
		}
	}

	return found
}

func (s *AutomationDetectionService) detectPuppeteer(userAgent string, data map[string]interface{}, result *AutoDetectionResult) bool {
	found := false

	for _, pattern := range s.puppeteerPatterns {
		if pattern.MatchString(userAgent) {
			result.Evidence = append(result.Evidence, "Puppeteer detected in User-Agent")
			result.ToolType = "puppeteer"
			found = true
			break
		}
	}

	if data != nil {
		if cdcObj, ok := data["cdc_object"]; ok && cdcObj != nil {
			result.Evidence = append(result.Evidence, "Puppeteer CDP object detected ($cdc_asdjflasutopfhvcZLmcfl_)")
			result.ToolType = "puppeteer"
			found = true
		}

		if puppeteerObj, ok := data["puppeteer_object"]; ok && puppeteerObj != nil {
			result.Evidence = append(result.Evidence, "Puppeteer global object detected")
			result.ToolType = "puppeteer"
			found = true
		}
	}

	return found
}

func (s *AutomationDetectionService) detectPlaywright(userAgent string, data map[string]interface{}, result *AutoDetectionResult) bool {
	found := false

	for _, pattern := range s.playwrightPatterns {
		if pattern.MatchString(userAgent) {
			result.Evidence = append(result.Evidence, "Playwright detected in User-Agent")
			result.ToolType = "playwright"
			found = true
			break
		}
	}

	if data != nil {
		if playwrightObj, ok := data["playwright_object"]; ok && playwrightObj != nil {
			result.Evidence = append(result.Evidence, "Playwright global object detected")
			result.ToolType = "playwright"
			found = true
		}
	}

	return found
}

func (s *AutomationDetectionService) detectHeadlessBrowser(userAgent string, data map[string]interface{}, result *AutoDetectionResult) bool {
	found := false

	for _, pattern := range s.headlessPatterns {
		if pattern.MatchString(userAgent) {
			result.Evidence = append(result.Evidence, "Headless browser detected in User-Agent")
			found = true
			break
		}
	}

	if data != nil {
		if plugins, ok := data["plugins"].([]interface{}); ok && len(plugins) == 0 {
			result.Evidence = append(result.Evidence, "No browser plugins (headless indicator)")
			found = true
		}

		if screenSize, ok := data["screen_size"].(string); ok {
			if strings.Contains(screenSize, "0x0") || strings.Contains(screenSize, "1x1") {
				result.Evidence = append(result.Evidence, "Abnormal screen size (headless indicator)")
				found = true
			}
		}

		if webglRenderer, ok := data["webgl_renderer"].(string); ok {
			if strings.Contains(strings.ToLower(webglRenderer), "swiftshader") ||
				strings.Contains(strings.ToLower(webglRenderer), "llvmpipe") {
				result.Evidence = append(result.Evidence, "Software renderer detected (headless indicator)")
				found = true
			}
		}
	}

	return found
}

func (s *AutomationDetectionService) detectWebDriver(userAgent string, data map[string]interface{}, result *AutoDetectionResult) bool {
	found := false

	for _, pattern := range s.webdriverPatterns {
		if pattern.MatchString(userAgent) {
			result.Evidence = append(result.Evidence, "WebDriver detected in User-Agent")
			if result.ToolType == "" {
				result.ToolType = "webdriver"
			}
			found = true
			break
		}
	}

	if data != nil {
		if webdriver, ok := data["webdriver"].(bool); ok && webdriver {
			result.Evidence = append(result.Evidence, "navigator.webdriver is true")
			if result.ToolType == "" {
				result.ToolType = "webdriver"
			}
			found = true
		}
	}

	return found
}

func (s *AutomationDetectionService) detectDebugger(r *http.Request, data map[string]interface{}, ip string, result *AutoDetectionResult) bool {
	found := false

	if data != nil {
		if debuggerDetected, ok := data["debugger_detected"].(bool); ok && debuggerDetected {
			result.Evidence = append(result.Evidence, "Debugger detected by frontend")
			found = true
		}

		if devtoolsOpen, ok := data["devtools_open"].(bool); ok && devtoolsOpen {
			result.Evidence = append(result.Evidence, "DevTools is open")
			found = true
		}

		if debuggerStatement, ok := data["debugger_statement"].(bool); ok && debuggerStatement {
			result.Evidence = append(result.Evidence, "Debugger statement executed")
			found = true
		}

		if timingAnomaly, ok := data["timing_anomaly"].(float64); ok && timingAnomaly > 50 {
			result.Evidence = append(result.Evidence, "Execution timing anomaly detected")
			found = true
		}

		if callStackDepth, ok := data["call_stack_depth"].(int); ok && callStackDepth > 100 {
			result.Evidence = append(result.Evidence, "Abnormal call stack depth")
			found = true
		}
	}

	if found {
		s.mu.Lock()
		s.debuggerDetection[ip] = &DebuggerDetectionRecord{
			IP:         ip,
			DetectedAt: time.Now(),
			Evidence:   result.Evidence,
			Confidence: result.Confidence,
		}
		s.mu.Unlock()
	}

	return found
}

func (s *AutomationDetectionService) analyzeBehavioralPatterns(ip string, r *http.Request) *AutoBehavioralIndicators {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	if _, exists := s.behaviorPatterns[ip]; !exists {
		s.behaviorPatterns[ip] = &AutomationBehavior{
			IP:                ip,
			RequestTimestamps: []time.Time{},
			RequestPaths:      []string{},
			RequestMethods:    []string{},
			RequestCount:      0,
			LastActivity:      now,
		}
	}

	behavior := s.behaviorPatterns[ip]
	behavior.RequestTimestamps = append(behavior.RequestTimestamps, now)
	behavior.RequestPaths = append(behavior.RequestPaths, r.URL.Path)
	behavior.RequestMethods = append(behavior.RequestMethods, r.Method)
	behavior.RequestCount++
	behavior.LastActivity = now

	maxRecords := 100
	if len(behavior.RequestTimestamps) > maxRecords {
		behavior.RequestTimestamps = behavior.RequestTimestamps[len(behavior.RequestTimestamps)-maxRecords:]
		behavior.RequestPaths = behavior.RequestPaths[len(behavior.RequestPaths)-maxRecords:]
		behavior.RequestMethods = behavior.RequestMethods[len(behavior.RequestMethods)-maxRecords:]
	}

	indicators := &AutoBehavioralIndicators{
		IsHumanLike: true,
	}

	if len(behavior.RequestTimestamps) >= 5 {
		intervals := make([]float64, 0)
		for i := 1; i < len(behavior.RequestTimestamps); i++ {
			interval := behavior.RequestTimestamps[i].Sub(behavior.RequestTimestamps[i-1]).Seconds()
			intervals = append(intervals, interval)
		}

		avgInterval := autoAverage(intervals)
		stdDev := autoStandardDeviation(intervals)
		coefficientVariation := stdDev / avgInterval

		indicators.IntervalRegularity = coefficientVariation
		indicators.RequestFrequency = float64(len(intervals)) / avgInterval

		if coefficientVariation < 0.1 && avgInterval < 0.5 {
			indicators.RequestPattern = "too_regular"
			indicators.TimingAnomalies = true
			indicators.IsHumanLike = false
		} else if avgInterval < 0.1 {
			indicators.RequestPattern = "too_fast"
			indicators.TimingAnomalies = true
			indicators.IsHumanLike = false
		} else {
			indicators.RequestPattern = "normal"
		}
	}

	if len(behavior.RequestPaths) > 0 {
		uniquePaths := make(map[string]bool)
		for _, path := range behavior.RequestPaths {
			uniquePaths[path] = true
		}
		indicators.PathDiversity = float64(len(uniquePaths)) / float64(len(behavior.RequestPaths))

		if indicators.PathDiversity < 0.1 {
			indicators.IsHumanLike = false
		}
	}

	return indicators
}

func (s *AutomationDetectionService) RecordSessionBehavior(sessionID string, path string, timestamp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessionBehavior[sessionID]; !exists {
		s.sessionBehavior[sessionID] = &SessionBehaviorAnalysis{
			SessionID:            sessionID,
			StartTime:            timestamp,
			TotalRequests:        0,
			UniquePaths:          make(map[string]bool),
			RequestIntervals:     []time.Duration{},
			PathTransitionMatrix: make(map[string]map[string]int),
			IsSuspicious:         false,
			SuspiciousScore:      0,
		}
	}

	session := s.sessionBehavior[sessionID]
	session.TotalRequests++
	session.UniquePaths[path] = true

	if len(session.RequestIntervals) > 0 {
		lastTime := session.RequestIntervals[len(session.RequestIntervals)-1]
		interval := timestamp.Sub(time.Unix(0, lastTime.Nanoseconds()))
		session.RequestIntervals = append(session.RequestIntervals, interval)
	} else {
		session.RequestIntervals = append(session.RequestIntervals, timestamp.Sub(session.StartTime))
	}

	if len(session.RequestIntervals) > 50 {
		session.RequestIntervals = session.RequestIntervals[len(session.RequestIntervals)-50:]
	}

	session.IsSuspicious = s.analyzeSessionSuspiciousness(session)
}

func (s *AutomationDetectionService) analyzeSessionSuspiciousness(session *SessionBehaviorAnalysis) bool {
	score := 0.0

	if session.TotalRequests > 100 {
		duration := time.Since(session.StartTime).Minutes()
		if duration > 0 {
			requestsPerMinute := float64(session.TotalRequests) / duration
			if requestsPerMinute > 50 {
				score += 30
			} else if requestsPerMinute > 20 {
				score += 15
			}
		}
	}

	pathDiversity := float64(len(session.UniquePaths)) / float64(session.TotalRequests)
	if pathDiversity < 0.2 {
		score += 40
	} else if pathDiversity < 0.4 {
		score += 20
	}

	if len(session.RequestIntervals) > 10 {
		intervals := make([]float64, 0)
		for _, interval := range session.RequestIntervals {
			intervals = append(intervals, interval.Seconds())
		}

		avgInterval := autoAverage(intervals)
		stdDev := autoStandardDeviation(intervals)
		cv := stdDev / avgInterval

		if cv < 0.05 && avgInterval < 1.0 {
			score += 45
		} else if cv < 0.1 && avgInterval < 2.0 {
			score += 25
		}
	}

	session.SuspiciousScore = score
	return score >= 60
}

func (s *AutomationDetectionService) GetSessionAnalysis(sessionID string) *SessionBehaviorAnalysis {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if session, exists := s.sessionBehavior[sessionID]; exists {
		return session
	}
	return nil
}

func (s *AutomationDetectionService) CleanupOldRecords() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)

	for ip, behavior := range s.behaviorPatterns {
		if behavior.LastActivity.Before(cutoff) {
			delete(s.behaviorPatterns, ip)
		}
	}

	for ip, record := range s.debuggerDetection {
		if record.DetectedAt.Before(cutoff) {
			delete(s.debuggerDetection, ip)
		}
	}

	for sessionID, session := range s.sessionBehavior {
		if session.StartTime.Before(cutoff) {
			delete(s.sessionBehavior, sessionID)
		}
	}
}

func autoAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func autoStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	avg := autoAverage(values)
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - avg
		sumSquaredDiff += diff * diff
	}
	return math.Sqrt(sumSquaredDiff / float64(len(values)))
}