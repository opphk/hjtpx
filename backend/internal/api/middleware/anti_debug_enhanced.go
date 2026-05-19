package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type AntiDebugConfig struct {
	Enabled              bool
	CheckDevTools         bool
	CheckDebugger        bool
	CheckWebdriver       bool
	CheckAutomation      bool
	CheckTiming          bool
	CheckMemoryTampering bool
	CheckConsole          bool
	BlockOnDetection     bool
	LogDetections        bool
	CustomRules          []AntiDebugRule
}

type AntiDebugRule struct {
	Name        string
	Pattern     string
	Severity    int
	Action      string
	Description string
}

var defaultAntiDebugConfig = AntiDebugConfig{
	Enabled:              true,
	CheckDevTools:         true,
	CheckDebugger:        true,
	CheckWebdriver:       true,
	CheckAutomation:      true,
	CheckTiming:          true,
	CheckMemoryTampering: true,
	CheckConsole:          true,
	BlockOnDetection:     false,
	LogDetections:        true,
}

var (
	detectionLog     = make([]DetectionEvent, 0, 1000)
	detectionMu      sync.RWMutex
	blockedClients   = make(map[string]time.Time)
	blockMu          sync.RWMutex
)

type DetectionEvent struct {
	Timestamp    time.Time
	ClientIP     string
	DetectionType string
	Severity     int
	UserAgent    string
	Details      string
	Blocked      bool
}

type AntiDebugMiddleware struct {
	config   AntiDebugConfig
	patterns []*compiledPattern
}

type compiledPattern struct {
	rule    AntiDebugRule
	regex   *regexp.Regexp
}

func NewAntiDebugMiddleware(config ...AntiDebugConfig) gin.HandlerFunc {
	cfg := defaultAntiDebugConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	middleware := &AntiDebugMiddleware{
		config:   cfg,
		patterns: compilePatterns(cfg.CustomRules),
	}

	go middleware.cleanupLoop()

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		clientIP := getClientIP(c)
		userAgent := c.GetHeader("User-Agent")

		detections := middleware.detectAll(c, clientIP, userAgent)

		if len(detections) > 0 {
			for _, detection := range detections {
				middleware.logDetection(detection)

				if cfg.LogDetections {
					service.GetLogger().Info("Anti-debug detection",
						"type", detection.DetectionType,
						"client_ip", detection.ClientIP,
						"severity", detection.Severity,
						"details", detection.Details,
					)
				}
			}

			if cfg.BlockOnDetection {
				middleware.blockClient(clientIP, detections)
				middleware.writeBlockedResponse(c, detections)
				c.Abort()
				return
			}

			middleware.addSecurityHeaders(c)
			c.Next()
		} else {
			middleware.addSecurityHeaders(c)
			c.Next()
		}
	}
}

func compilePatterns(rules []AntiDebugRule) []*compiledPattern {
	var patterns []*compiledPattern
	for _, rule := range rules {
		if rule.Pattern != "" {
			regex := regexp.MustCompile(rule.Pattern)
			patterns = append(patterns, &compiledPattern{
				rule:  rule,
				regex: regex,
			})
		}
	}
	return patterns
}

func (m *AntiDebugMiddleware) detectAll(c *gin.Context, clientIP, userAgent string) []DetectionEvent {
	var detections []DetectionEvent

	if m.config.CheckDevTools {
		if detection := m.checkDevTools(c, clientIP, userAgent); detection != nil {
			detections = append(detections, *detection)
		}
	}

	if m.config.CheckDebugger {
		if detection := m.checkDebugger(c, clientIP, userAgent); detection != nil {
			detections = append(detections, *detection)
		}
	}

	if m.config.CheckWebdriver {
		if detection := m.checkWebdriver(c, clientIP, userAgent); detection != nil {
			detections = append(detections, *detection)
		}
	}

	if m.config.CheckAutomation {
		if detection := m.checkAutomation(c, clientIP, userAgent); detection != nil {
			detections = append(detections, *detection)
		}
	}

	if m.config.CheckTiming {
		if detection := m.checkTiming(c, clientIP, userAgent); detection != nil {
			detections = append(detections, *detection)
		}
	}

	if m.config.CheckMemoryTampering {
		if detection := m.checkMemoryTampering(c, clientIP, userAgent); detection != nil {
			detections = append(detections, *detection)
		}
	}

	if m.config.CheckConsole {
		if detection := m.checkConsole(c, clientIP, userAgent); detection != nil {
			detections = append(detections, *detection)
		}
	}

	for _, pattern := range m.patterns {
		if detection := m.checkCustomPattern(c, clientIP, userAgent, pattern); detection != nil {
			detections = append(detections, *detection)
		}
	}

	return detections
}

func (m *AntiDebugMiddleware) checkDevTools(c *gin.Context, clientIP, userAgent string) *DetectionEvent {
	headers := c.Request.Header

	if c.GetHeader("X-DevTools-Emulate") != "" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "devtools_emulation",
			Severity:      8,
			UserAgent:     userAgent,
			Details:       "DevTools emulation header detected",
		}
	}

	if c.GetHeader("Sec-Use-H5cache") == "false" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "devtools_activity",
			Severity:      6,
			UserAgent:     userAgent,
			Details:       "DevTools cache manipulation detected",
		}
	}

	for key := range headers {
		if strings.Contains(strings.ToLower(key), "devtools") {
			return &DetectionEvent{
				Timestamp:     time.Now(),
				ClientIP:      clientIP,
				DetectionType: "devtools_header",
				Severity:      7,
				UserAgent:     userAgent,
				Details:       fmt.Sprintf("DevTools-related header: %s", key),
			}
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) checkDebugger(c *gin.Context, clientIP, userAgent string) *DetectionEvent {
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	if c.GetHeader("X-Debug-Mode") == "true" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "debugger_enabled",
			Severity:      9,
			UserAgent:     userAgent,
			Details:       "Debug mode explicitly enabled",
		}
	}

	bodyStr := string(body)
	debugPatterns := []string{
		"debugger;",
		"debugger; ",
		"debugger;\n",
		"debugger;\r\n",
	}

	for _, pattern := range debugPatterns {
		if strings.Contains(bodyStr, pattern) {
			return &DetectionEvent{
				Timestamp:     time.Now(),
				ClientIP:      clientIP,
				DetectionType: "debugger_keyword",
				Severity:      8,
				UserAgent:     userAgent,
				Details:       "Debugger keyword in request body",
			}
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) checkWebdriver(c *gin.Context, clientIP, userAgent string) *DetectionEvent {
	webdriverHeader := c.GetHeader("X-Webdriver")
	if webdriverHeader == "true" || webdriverHeader == "1" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "webdriver_detected",
			Severity:      10,
			UserAgent:     userAgent,
			Details:       "WebDriver automation framework detected",
		}
	}

	if c.GetHeader("webdriver") == "true" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "webdriver_attribute",
			Severity:      10,
			UserAgent:     userAgent,
			Details:       "WebDriver attribute in browser",
		}
	}

	if strings.Contains(strings.ToLower(userAgent), "webdriver") {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "webdriver_ua",
			Severity:      9,
			UserAgent:     userAgent,
			Details:       "WebDriver detected in User-Agent",
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) checkAutomation(c *gin.Context, clientIP, userAgent string) *DetectionEvent {
	automationPatterns := []struct {
		pattern  string
		severity int
		desc     string
	}{
		{"HeadlessChrome", 8, "Headless Chrome detected"},
		{"Headless", 8, "Headless browser detected"},
		{"PhantomJS", 9, "PhantomJS detected"},
		{"Selenium", 9, "Selenium WebDriver detected"},
		{"Puppeteer", 8, "Puppeteer detected"},
		{"playwright", 8, "Playwright detected"},
		{"Automation", 7, "Browser automation detected"},
	}

	for _, ap := range automationPatterns {
		if strings.Contains(userAgent, ap.pattern) {
			return &DetectionEvent{
				Timestamp:     time.Now(),
				ClientIP:      clientIP,
				DetectionType: "automation_framework",
				Severity:      ap.severity,
				UserAgent:     userAgent,
				Details:       ap.desc,
			}
		}
	}

	if c.GetHeader("X-Automation-Framework") != "" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "automation_header",
			Severity:      8,
			UserAgent:     userAgent,
			Details:       "Automation framework header detected",
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) checkTiming(c *gin.Context, clientIP, userAgent string) *DetectionEvent {
	timingHeader := c.GetHeader("X-Request-Timing")
	if timingHeader != "" {
		if timing, err := strconv.ParseFloat(timingHeader, 64); err == nil {
			if timing < 10 && !strings.Contains(userAgent, "Chrome-Lighthouse") {
				return &DetectionEvent{
					Timestamp:     time.Now(),
					ClientIP:      clientIP,
					DetectionType: "suspicious_timing",
					Severity:      5,
					UserAgent:     userAgent,
					Details:       fmt.Sprintf("Request completed in %.2fms (suspiciously fast)", timing),
				}
			}
		}
	}

	if c.GetHeader("X-Performance-Mark") == "skipped" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "skipped_performance_marks",
			Severity:      6,
			UserAgent:     userAgent,
			Details:       "Performance marks skipped (possible automation)",
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) checkMemoryTampering(c *gin.Context, clientIP, userAgent string) *DetectionEvent {
	tamperPatterns := []string{
		"__proto__",
		"constructor",
		"prototype",
		"[object Object]",
	}

	for _, pattern := range tamperPatterns {
		if c.GetHeader("X-"+pattern+"-Modified") == "true" {
			return &DetectionEvent{
				Timestamp:     time.Now(),
				ClientIP:      clientIP,
				DetectionType: "memory_tampering",
				Severity:      8,
				UserAgent:     userAgent,
				Details:       fmt.Sprintf("Object %s modification detected", pattern),
			}
		}
	}

	if c.GetHeader("X-Object-Freeze") == "disabled" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "object_protection_disabled",
			Severity:      7,
			UserAgent:     userAgent,
			Details:       "Object freeze/protection disabled",
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) checkConsole(c *gin.Context, clientIP, userAgent string) *DetectionEvent {
	consoleMethods := c.GetHeader("X-Console-Methods")
	if consoleMethods != "" {
		methods := strings.Split(consoleMethods, ",")
		if len(methods) == 0 || (len(methods) == 1 && methods[0] == "") {
			return &DetectionEvent{
				Timestamp:     time.Now(),
				ClientIP:      clientIP,
				DetectionType: "no_console_methods",
				Severity:      6,
				UserAgent:     userAgent,
				Details:       "No console methods available (possible sandbox)",
			}
		}
	}

	if c.GetHeader("X-Console-Overridden") == "true" {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "console_override",
			Severity:      7,
			UserAgent:     userAgent,
			Details:       "Console methods overridden",
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) checkCustomPattern(c *gin.Context, clientIP, userAgent string, pattern *compiledPattern) *DetectionEvent {
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	bodyStr := string(body)
	headersStr := fmt.Sprintf("%v", c.Request.Header)

	if pattern.regex.MatchString(bodyStr) || pattern.regex.MatchString(headersStr) {
		return &DetectionEvent{
			Timestamp:     time.Now(),
			ClientIP:      clientIP,
			DetectionType: "custom_rule_" + pattern.rule.Name,
			Severity:      pattern.rule.Severity,
			UserAgent:     userAgent,
			Details:       pattern.rule.Description,
		}
	}

	return nil
}

func (m *AntiDebugMiddleware) logDetection(event DetectionEvent) {
	detectionMu.Lock()
	defer detectionMu.Unlock()

	detectionLog = append(detectionLog, event)

	if len(detectionLog) > 1000 {
		detectionLog = detectionLog[len(detectionLog)-1000:]
	}
}

func (m *AntiDebugMiddleware) blockClient(clientIP string, detections []DetectionEvent) {
	blockMu.Lock()
	defer blockMu.Unlock()

	maxSeverity := 0
	for _, d := range detections {
		if d.Severity > maxSeverity {
			maxSeverity = d.Severity
		}
	}

	duration := time.Duration(maxSeverity*30) * time.Second
	if duration < 60*time.Second {
		duration = 60 * time.Second
	}

	blockedClients[clientIP] = time.Now().Add(duration)
}

func (m *AntiDebugMiddleware) writeBlockedResponse(c *gin.Context, detections []DetectionEvent) {
	c.Header("X-Blocked-Reason", "anti_debug_detection")
	c.Header("X-Detection-Count", strconv.Itoa(len(detections)))

	detectionTypes := make([]string, len(detections))
	for i, d := range detections {
		detectionTypes[i] = d.DetectionType
	}
	detectionJSON, _ := json.Marshal(detectionTypes)
	c.Header("X-Detection-Types", string(detectionJSON))

	c.JSON(http.StatusForbidden, gin.H{
		"error":       "access_denied",
		"message":     "Security violation detected",
		"detections":  detections,
		"timestamp":   time.Now(),
	})
}

func (m *AntiDebugMiddleware) addSecurityHeaders(c *gin.Context) {
	c.Header("X-Debug-Check", "enabled")
	c.Header("X-Anti-Debug", "active")
	c.Header("X-Security-Level", "enhanced")
}

func (m *AntiDebugMiddleware) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.cleanupBlockedClients()
		m.cleanupDetectionLog()
	}
}

func (m *AntiDebugMiddleware) cleanupBlockedClients() {
	blockMu.Lock()
	defer blockMu.Unlock()

	now := time.Now()
	for clientIP, expiry := range blockedClients {
		if now.After(expiry) {
			delete(blockedClients, clientIP)
		}
	}
}

func (m *AntiDebugMiddleware) cleanupDetectionLog() {
	detectionMu.Lock()
	defer detectionMu.Unlock()

	if len(detectionLog) > 500 {
		detectionLog = detectionLog[len(detectionLog)-500:]
	}
}

func GetDetectionLog() []DetectionEvent {
	detectionMu.RLock()
	defer detectionMu.RUnlock()
	return detectionLog
}

func GetBlockedClients() map[string]time.Time {
	blockMu.RLock()
	defer blockMu.RUnlock()
	return blockedClients
}

func IsClientBlocked(clientIP string) bool {
	blockMu.RLock()
	defer blockMu.RUnlock()

	if expiry, exists := blockedClients[clientIP]; exists {
		if time.Now().Before(expiry) {
			return true
		}
		delete(blockedClients, clientIP)
	}
	return false
}

func UnblockClient(clientIP string) bool {
	blockMu.Lock()
	defer blockMu.Unlock()

	if _, exists := blockedClients[clientIP]; exists {
		delete(blockedClients, clientIP)
		return true
	}
	return false
}

func GenerateSecurityToken(clientIP, userAgent string) string {
	data := fmt.Sprintf("%s:%s:%d", clientIP, userAgent, time.Now().Unix()/300)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:32]
}

func ValidateSecurityToken(token, clientIP, userAgent string) bool {
	expectedToken := GenerateSecurityToken(clientIP, userAgent)
	return token == expectedToken
}

func SetupAntiDebugMiddleware(r *gin.Engine) {
	config := AntiDebugConfig{
		Enabled:              true,
		CheckDevTools:         true,
		CheckDebugger:        true,
		CheckWebdriver:       true,
		CheckAutomation:      true,
		CheckTiming:          true,
		CheckMemoryTampering: true,
		CheckConsole:          true,
		BlockOnDetection:     false,
		LogDetections:        true,
		CustomRules: []AntiDebugRule{
			{
				Name:        "automation_tools",
				Pattern:     "(?i)(puppeteer|playwright|selenium|phantomjs|nightmare)",
				Severity:    8,
				Action:      "block",
				Description: "Known automation tool detected",
			},
			{
				Name:        "suspicious_headers",
				Pattern:     "(?i)(x-.*-modified|x-automation|x-devtools|x-webdriver)",
				Severity:    6,
				Action:      "log",
				Description: "Suspicious custom header detected",
			},
		},
	}

	r.Use(NewAntiDebugMiddleware(config))
}

func getClientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}

	return c.ClientIP()
}
