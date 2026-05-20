package tools

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	github.com/hjtpx/hjtpx/internal/model"
)

type DebugDetector struct {
	config         *model.DebugDetectorConfig
	detector       *model.ThreadSafeDebugDetector
	alertHandlers  []DebugAlertHandler
	timingTracker  *TimingTracker
	breakpointMgr  *BreakpointManager
	consoleMonitor *ConsoleMonitor
	stopChan       chan struct{}
	running        bool
	mu             sync.RWMutex
}

type DebugAlertHandler interface {
	HandleAlert(alert *DebugAlert) error
}

type DebugAlertHandlerFunc func(alert *DebugAlert) error

func (f DebugAlertHandlerFunc) HandleAlert(alert *DebugAlert) error {
	return f(alert)
}

type DebugAlert struct {
	ID          string                    `json:"id"`
	Type        model.DebugDetectionType  `json:"type"`
	Severity    model.DebugSeverity      `json:"severity"`
	Message     string                   `json:"message"`
	SessionID   string                   `json:"session_id"`
	IPAddress   string                   `json:"ip_address"`
	Timestamp   time.Time                `json:"timestamp"`
	Evidence    string                   `json:"evidence"`
	Metadata    map[string]interface{}    `json:"metadata,omitempty"`
}

type TimingTracker struct {
	mu           sync.RWMutex
	executions   map[string]*ExecutionRecord
	baselineTime map[string]time.Duration
	maxVariance  time.Duration
}

type ExecutionRecord struct {
	SessionID     string
	OperationName string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	IsAnomalous   bool
	Deviation     float64
}

type BreakpointManager struct {
	mu           sync.RWMutex
	breakpoints  map[string]*BreakpointInfo
	monitorCount map[string]int
}

type BreakpointInfo struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	LineNumber   int       `json:"line_number"`
	FunctionName string    `json:"function_name"`
	HitCount     int       `json:"hit_count"`
	LastHit      time.Time `json:"last_hit"`
	CallStack    []string  `json:"call_stack"`
	IsActive     bool      `json:"is_active"`
}

type ConsoleMonitor struct {
	mu               sync.RWMutex
	consoleMethods   map[string]bool
	overriddenMethods map[string]bool
	activityLog      []ConsoleActivity
	maxLogSize       int
}

type ConsoleActivity struct {
	Method        string
	Timestamp      time.Time
	IsOverridden   bool
	CallCount     int
}

type WebhookDebugAlertHandler struct {
	endpoint   string
	httpClient HTTPClient
	timeout    time.Duration
}

type LogDebugAlertHandler struct {
	logger Logger
}

type Logger interface {
	Error(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Info(format string, args ...interface{})
}

type DefaultLogger struct{}

func (l *DefaultLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

func (l *DefaultLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

type DefaultHTTPClient struct{}

func (c *DefaultHTTPClient) Post(url string, body []byte) error {
	return fmt.Errorf("http client not configured - implement for production use")
}

type HTTPClient interface {
	Post(url string, body []byte) error
}

func NewDebugDetector(config *model.DebugDetectorConfig) *DebugDetector {
	if config == nil {
		config = model.NewDebugDetectorConfig()
	}

	dd := &DebugDetector{
		config:         config,
		detector:       model.NewThreadSafeDebugDetector(),
		alertHandlers:  make([]DebugAlertHandler, 0),
		timingTracker:  NewTimingTracker(),
		breakpointMgr:  NewBreakpointManager(),
		consoleMonitor: NewConsoleMonitor(),
		stopChan:       make(chan struct{}),
	}

	dd.initializeDefaultHandlers()

	return dd
}

func (dd *DebugDetector) initializeDefaultHandlers() {
	dd.AddAlertHandler(NewLogDebugAlertHandler())
}

func NewTimingTracker() *TimingTracker {
	return &TimingTracker{
		executions:   make(map[string]*ExecutionRecord),
		baselineTime: make(map[string]time.Duration),
		maxVariance:  500 * time.Millisecond,
	}
}

func NewBreakpointManager() *BreakpointManager {
	return &BreakpointManager{
		breakpoints:  make(map[string]*BreakpointInfo),
		monitorCount: make(map[string]int),
	}
}

func NewConsoleMonitor() *ConsoleMonitor {
	return &ConsoleMonitor{
		consoleMethods:    map[string]bool{},
		overriddenMethods: map[string]bool{},
		activityLog:       make([]ConsoleActivity, 0),
		maxLogSize:        100,
	}
}

func (dd *DebugDetector) AddAlertHandler(handler DebugAlertHandler) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.alertHandlers = append(dd.alertHandlers, handler)
}

func (dd *DebugDetector) Start() error {
	dd.mu.Lock()
	if dd.running {
		dd.mu.Unlock()
		return fmt.Errorf("debug detector is already running")
	}
	dd.running = true
	dd.stopChan = make(chan struct{})
	dd.mu.Unlock()

	go dd.cleanupLoop()

	return nil
}

func (dd *DebugDetector) Stop() {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if !dd.running {
		return
	}

	dd.running = false
	close(dd.stopChan)
}

func (dd *DebugDetector) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dd.detector.CleanupOldSessions(30 * time.Minute)
			dd.cleanupOldTimingRecords()
		case <-dd.stopChan:
			return
		}
	}
}

func (dd *DebugDetector) cleanupOldTimingRecords() {
	dd.timingTracker.mu.Lock()
	defer dd.timingTracker.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for key, record := range dd.timingTracker.executions {
		if record.EndTime.Before(cutoff) {
			delete(dd.timingTracker.executions, key)
		}
	}
}

func (dd *DebugDetector) DetectAll(sessionID, clientIP, userAgent string, headers http.Header, body []byte) *model.DebugDetectionResult {
	result := model.NewDebugDetectionResult()
	result.SessionID = sessionID
	result.IPAddress = clientIP
	result.UserAgent = userAgent

	startTime := time.Now()

	dd.mu.RLock()
	config := dd.config
	dd.mu.RUnlock()

	if config.CheckDevTools {
		if detection := dd.detectDevTools(sessionID, clientIP, userAgent, headers); detection != nil {
			result.AddDetection(*detection)
			dd.handleDetection(sessionID, detection)
		}
	}

	if config.CheckDebugger {
		if detection := dd.detectDebugger(sessionID, clientIP, userAgent, headers, body); detection != nil {
			result.AddDetection(*detection)
			dd.handleDetection(sessionID, detection)
		}
	}

	if config.CheckBreakpoints {
		detections := dd.detectBreakpoints(sessionID, clientIP, userAgent, headers)
		for _, detection := range detections {
			result.AddDetection(detection)
			dd.handleDetection(sessionID, &detection)
		}
	}

	if config.CheckTimingAnomaly {
		if detection := dd.detectTimingAnomaly(sessionID, clientIP, userAgent, headers); detection != nil {
			result.AddDetection(*detection)
			dd.handleDetection(sessionID, detection)
		}
	}

	if config.CheckConsoleActivity {
		if detection := dd.detectConsoleActivity(sessionID, clientIP, userAgent, headers); detection != nil {
			result.AddDetection(*detection)
			dd.handleDetection(sessionID, detection)
		}
	}

	if config.CheckMemoryAccess {
		if detection := dd.detectMemoryAccess(sessionID, clientIP, userAgent, headers); detection != nil {
			result.AddDetection(*detection)
			dd.handleDetection(sessionID, detection)
		}
	}

	if config.CheckCallStackDepth {
		if detection := dd.detectCallStackDepth(sessionID, clientIP, userAgent, headers); detection != nil {
			result.AddDetection(*detection)
			dd.handleDetection(sessionID, detection)
		}
	}

	result.ResponseTimeMs = time.Since(startTime).Milliseconds()
	result.RiskScore = result.CalculateRiskScore()
	result.ShouldBlock = dd.shouldBlockSession(result, config)

	if config.BlockOnDetection && result.ShouldBlock {
		dd.blockIP(clientIP, config.AutoBlockDuration)
		result.Recommendation = fmt.Sprintf("Session blocked for %v due to %d debug detection(s)", 
			config.AutoBlockDuration, result.DetectionCount)
	} else {
		result.Recommendation = dd.generateRecommendation(result)
	}

	return result
}

func (dd *DebugDetector) detectDevTools(sessionID, clientIP, userAgent string, headers http.Header) *model.DebugDetectionEvent {
	if headers.Get("X-DevTools-Emulate") != "" {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeDevTools,
			Severity:      model.SeverityHigh,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "DevTools emulation header detected",
		}
	}

	if headers.Get("Sec-Use-H5cache") == "false" {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeDevTools,
			Severity:      model.SeverityMedium,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "DevTools cache manipulation detected",
		}
	}

	for key := range headers {
		if strings.Contains(strings.ToLower(key), "devtools") {
			return &model.DebugDetectionEvent{
				SessionID:     sessionID,
				Type:          model.DebugTypeDevTools,
				Severity:      model.SeverityHigh,
				IPAddress:     clientIP,
				UserAgent:     userAgent,
				DetectionTime: time.Now(),
				Evidence:      fmt.Sprintf("DevTools-related header: %s", key),
			}
		}
	}

	if headers.Get("X-Debug-Panel") != "" || headers.Get("X-Chrome-Uhlenbrock") != "" {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeDevTools,
			Severity:      model.SeverityCritical,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "Chrome DevTools specific header detected",
		}
	}

	return nil
}

func (dd *DebugDetector) detectDebugger(sessionID, clientIP, userAgent string, headers http.Header, body []byte) *model.DebugDetectionEvent {
	if headers.Get("X-Debug-Mode") == "true" {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeDebugger,
			Severity:      model.SeverityCritical,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "Debug mode explicitly enabled via header",
		}
	}

	if len(body) > 0 {
		bodyStr := string(body)
		debugPatterns := []struct {
			pattern  string
			severity model.DebugSeverity
			desc     string
		}{
			{"debugger;", model.SeverityHigh, "debugger; keyword in request body"},
			{"debugger; ", model.SeverityHigh, "debugger; with space in request body"},
			{"debugger;\n", model.SeverityHigh, "debugger; with newline in request body"},
			{"debugger;\r\n", model.SeverityHigh, "debugger; with CRLF in request body"},
			{`debugger;`, model.SeverityMedium, "compressed debugger statement detected"},
		}

		for _, dp := range debugPatterns {
			if strings.Contains(bodyStr, dp.pattern) {
				return &model.DebugDetectionEvent{
					SessionID:     sessionID,
					Type:          model.DebugTypeDebugger,
					Severity:      dp.severity,
					IPAddress:     clientIP,
					UserAgent:     userAgent,
					DetectionTime: time.Now(),
					Evidence:      dp.desc,
					RequestData:   truncateString(bodyStr, 200),
				}
			}
		}
	}

	if headers.Get("X-Breakpoint") != "" || headers.Get("X-Debug-Break") != "" {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeDebugger,
			Severity:      model.SeverityHigh,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "Breakpoint header detected",
		}
	}

	return nil
}

func (dd *DebugDetector) detectBreakpoints(sessionID, clientIP, userAgent string, headers http.Header) []model.DebugDetectionEvent {
	var detections []model.DebugDetectionEvent

	breakpointHeaders := []string{
		"X-Breakpoint-Line",
		"X-Debug-Breakpoint",
		"X-Breakpoint-ID",
		"X-Debug-Pause",
	}

	for _, header := range breakpointHeaders {
		if value := headers.Get(header); value != "" {
			detections = append(detections, model.DebugDetectionEvent{
				SessionID:     sessionID,
				Type:          model.DebugTypeBreakpoint,
				Severity:      model.SeverityHigh,
				IPAddress:     clientIP,
				UserAgent:     userAgent,
				DetectionTime: time.Now(),
				Evidence:      fmt.Sprintf("Breakpoint %s: %s", header, value),
			})
		}
	}

	dd.mu.RLock()
	bpCount := dd.breakpointMgr.monitorCount[sessionID]
	dd.mu.RUnlock()

	if bpCount > dd.config.BreakpointThreshold {
		detections = append(detections, model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeBreakpoint,
			Severity:      model.SeverityCritical,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      fmt.Sprintf("Excessive breakpoint monitoring: %d times", bpCount),
		})
	}

	return detections
}

func (dd *DebugDetector) detectTimingAnomaly(sessionID, clientIP, userAgent string, headers http.Header) *model.DebugDetectionEvent {
	timingHeader := headers.Get("X-Request-Timing")
	if timingHeader == "" {
		return nil
	}

	timing, err := strconv.ParseInt(timingHeader, 10, 64)
	if err != nil {
		return nil
	}

	dd.timingTracker.mu.Lock()
	baseline, exists := dd.timingTracker.baselineTime[sessionID]
	if !exists {
		dd.timingTracker.baselineTime[sessionID] = time.Duration(timing) * time.Millisecond
		dd.timingTracker.mu.Unlock()
		return nil
	}
	dd.timingTracker.mu.Unlock()

	currentDuration := time.Duration(timing) * time.Millisecond
	deviation := float64(currentDuration-baseline) / float64(baseline) * 100

	if deviation > 500 || deviation < -80 {
		violation := &model.TimingViolation{
			Type:       "execution_time_anomaly",
			ExpectedMs:  baseline.Milliseconds(),
			ActualMs:    timing,
			Deviation:  deviation,
			Severity:   7,
			Timestamp:  time.Now(),
			Evidence:   fmt.Sprintf("Timing deviation: %.2f%% (baseline: %dms, current: %dms)", 
				deviation, baseline.Milliseconds(), timing),
		}

		metadata, _ := json.Marshal(violation)

		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeTimingAnomaly,
			Severity:      model.SeverityHigh,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      violation.Evidence,
			Metadata:      string(metadata),
			ResponseDelay: timing,
		}
	}

	return nil
}

func (dd *DebugDetector) detectConsoleActivity(sessionID, clientIP, userAgent string, headers http.Header) *model.DebugDetectionEvent {
	consoleMethods := headers.Get("X-Console-Methods")
	if consoleMethods == "" {
		return nil
	}

	methods := strings.Split(consoleMethods, ",")
	if len(methods) == 0 || (len(methods) == 1 && methods[0] == "") {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeConsoleActivity,
			Severity:      model.SeverityMedium,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "No console methods available (possible sandbox)",
		}
	}

	if headers.Get("X-Console-Overridden") == "true" {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeConsoleActivity,
			Severity:      model.SeverityHigh,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "Console methods overridden",
		}
	}

	suspiciousMethods := []string{"debug", "table", "trace", "profile", "profileEnd"}
	for _, method := range suspiciousMethods {
		if containsString(methods, method) {
			return &model.DebugDetectionEvent{
				SessionID:     sessionID,
				Type:          model.DebugTypeConsoleActivity,
				Severity:      model.SeverityWarning,
				IPAddress:     clientIP,
				UserAgent:     userAgent,
				DetectionTime: time.Now(),
				Evidence:      fmt.Sprintf("Suspicious console method used: %s", method),
			}
		}
	}

	return nil
}

func (dd *DebugDetector) detectMemoryAccess(sessionID, clientIP, userAgent string, headers http.Header) *model.DebugDetectionEvent {
	tamperPatterns := []struct {
		pattern  string
		severity model.DebugSeverity
		desc     string
	}{
		{"__proto__", model.SeverityHigh, "Object prototype modification detected"},
		{"constructor", model.SeverityHigh, "Constructor modification detected"},
		{"prototype", model.SeverityMedium, "Prototype access detected"},
		{"Object-Freeze", model.SeverityMedium, "Object freeze disabled"},
		{"Object-Seal", model.SeverityMedium, "Object seal disabled"},
	}

	for _, tp := range tamperPatterns {
		if headers.Get("X-"+tp.pattern+"-Modified") == "true" {
			return &model.DebugDetectionEvent{
				SessionID:     sessionID,
				Type:          model.DebugTypeMemoryAccess,
				Severity:      tp.severity,
				IPAddress:     clientIP,
				UserAgent:     userAgent,
				DetectionTime: time.Now(),
				Evidence:      tp.desc,
			}
		}
	}

	if headers.Get("X-Object-Freeze") == "disabled" || headers.Get("X-Object-Seal") == "disabled" {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeMemoryAccess,
			Severity:      model.SeverityHigh,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      "Object protection disabled",
		}
	}

	return nil
}

func (dd *DebugDetector) detectCallStackDepth(sessionID, clientIP, userAgent string, headers http.Header) *model.DebugDetectionEvent {
	depthHeader := headers.Get("X-Call-Stack-Depth")
	if depthHeader == "" {
		return nil
	}

	depth, err := strconv.Atoi(depthHeader)
	if err != nil {
		return nil
	}

	maxReasonableDepth := 50
	if depth > maxReasonableDepth {
		return &model.DebugDetectionEvent{
			SessionID:     sessionID,
			Type:          model.DebugTypeCallStackDepth,
			Severity:      model.SeverityHigh,
			IPAddress:     clientIP,
			UserAgent:     userAgent,
			DetectionTime: time.Now(),
			Evidence:      fmt.Sprintf("Excessive call stack depth: %d (max reasonable: %d)", 
				depth, maxReasonableDepth),
		}
	}

	return nil
}

func (dd *DebugDetector) handleDetection(sessionID string, detection *model.DebugDetectionEvent) {
	dd.mu.RLock()
	config := dd.config
	dd.mu.RUnlock()

	dd.detector.AddDetection(sessionID, *detection)

	if config.LogDetections {
		dd.logDetection(detection)
	}

	if detection.Severity >= config.MinSeverityToBlock {
		dd.triggerAlert(detection)
	}
}

func (dd *DebugDetector) logDetection(detection *model.DebugDetectionEvent) {
	dd.mu.RLock()
	handlers := dd.alertHandlers
	dd.mu.RUnlock()

	for _, handler := range handlers {
		if logHandler, ok := handler.(*LogDebugAlertHandler); ok {
			logHandler.logger.Warn("Debug detection: Type=%s, Severity=%d, SessionID=%s, IP=%s, Evidence=%s",
				detection.Type, detection.Severity, detection.SessionID, detection.IPAddress, detection.Evidence)
		}
	}
}

func (dd *DebugDetector) triggerAlert(detection *model.DebugDetectionEvent) {
	dd.mu.RLock()
	handlers := dd.alertHandlers
	dd.mu.RUnlock()

	alert := &DebugAlert{
		ID:        fmt.Sprintf("alert-%d-%s", time.Now().UnixNano(), detection.Type),
		Type:      detection.Type,
		Severity:  detection.Severity,
		Message:   fmt.Sprintf("High severity debug detection: %s", detection.Evidence),
		SessionID: detection.SessionID,
		IPAddress: detection.IPAddress,
		Timestamp: time.Now(),
		Evidence:  detection.Evidence,
	}

	for _, handler := range handlers {
		go func(h DebugAlertHandler) {
			_ = h.HandleAlert(alert)
		}(handler)
	}
}

func (dd *DebugDetector) shouldBlockSession(result *model.DebugDetectionResult, config *model.DebugDetectorConfig) bool {
	if !config.BlockOnDetection {
		return false
	}

	if result.DetectionCount >= config.DetectionThreshold {
		return true
	}

	if result.HighestSeverity >= config.MinSeverityToBlock {
		return true
	}

	sessionCount := dd.detector.GetSessionDetectionCount(result.SessionID)
	if sessionCount >= config.MaxDetectionsPerSession {
		return true
	}

	return false
}

func (dd *DebugDetector) blockIP(ip string, duration time.Duration) {
	dd.detector.BlockIP(ip, duration)
}

func (dd *DebugDetector) generateRecommendation(result *model.DebugDetectionResult) string {
	if result.DetectionCount == 0 {
		return "No debug activity detected. Session appears normal."
	}

	recommendations := []string{
		fmt.Sprintf("Detected %d debug-related activities with highest severity %d", 
			result.DetectionCount, result.HighestSeverity),
		fmt.Sprintf("Risk score: %.2f%%", result.RiskScore),
	}

	if result.HighestSeverity >= model.SeverityCritical {
		recommendations = append(recommendations, "CRITICAL: Immediate action recommended - possible sophisticated debugging attempt")
	} else if result.HighestSeverity >= model.SeverityHigh {
		recommendations = append(recommendations, "HIGH: Consider blocking or additional verification")
	} else if result.HighestSeverity >= model.SeverityMedium {
		recommendations = append(recommendations, "MEDIUM: Monitor closely for escalation")
	} else {
		recommendations = append(recommendations, "LOW: Log for future analysis")
	}

	return strings.Join(recommendations, ". ")
}

func (dd *DebugDetector) RecordBreakpointHit(sessionID, bpType string, lineNumber int, functionName string) {
	dd.breakpointMgr.mu.Lock()
	defer dd.breakpointMgr.mu.Unlock()

	key := fmt.Sprintf("%s:%d:%s", sessionID, lineNumber, functionName)
	if info, exists := dd.breakpointMgr.breakpoints[key]; exists {
		info.HitCount++
		info.LastHit = time.Now()
	} else {
		dd.breakpointMgr.breakpoints[key] = &BreakpointInfo{
			ID:           key,
			Type:         bpType,
			LineNumber:   lineNumber,
			FunctionName: functionName,
			HitCount:     1,
			LastHit:      time.Now(),
			IsActive:     true,
		}
	}

	dd.breakpointMgr.monitorCount[sessionID]++
}

func (dd *DebugDetector) TrackExecution(sessionID, operationName string) func() {
	startTime := time.Now()

	return func() {
		duration := time.Since(startTime)

		dd.timingTracker.mu.Lock()
		defer dd.timingTracker.mu.Unlock()

		key := fmt.Sprintf("%s:%s", sessionID, operationName)
		record := &ExecutionRecord{
			SessionID:     sessionID,
			OperationName: operationName,
			StartTime:     startTime,
			EndTime:       time.Now(),
			Duration:      duration,
		}

		if baseline, exists := dd.timingTracker.baselineTime[operationName]; exists {
			deviation := float64(duration-baseline) / float64(baseline) * 100
			if deviation > 100 || deviation < -50 {
				record.IsAnomalous = true
				record.Deviation = deviation
			}
		} else {
			dd.timingTracker.baselineTime[operationName] = duration
		}

		dd.timingTracker.executions[key] = record
	}
}

func (dd *DebugDetector) GetStats() *model.DebugStats {
	return dd.detector.GetStats()
}

func (dd *DebugDetector) GetSessionDetections(sessionID string) []model.DebugDetectionEvent {
	return dd.detector.GetSessionDetections(sessionID)
}

func (dd *DebugDetector) IsIPBlocked(ip string) bool {
	return dd.detector.IsIPBlocked(ip)
}

func (dd *DebugDetector) SetConfig(config *model.DebugDetectorConfig) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.config = config
	dd.detector.SetConfig(config)
}

func (dd *DebugDetector) GetConfig() *model.DebugDetectorConfig {
	dd.mu.RLock()
	defer dd.mu.RUnlock()
	return dd.config
}

func (dd *DebugDetector) GenerateSecurityToken(sessionID, clientIP, userAgent string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", sessionID, clientIP, userAgent, time.Now().Unix()/300)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:32]
}

func (dd *DebugDetector) ValidateSecurityToken(token, sessionID, clientIP, userAgent string) bool {
	expectedToken := dd.GenerateSecurityToken(sessionID, clientIP, userAgent)
	return hmac.Equal([]byte(token), []byte(expectedToken))
}

func (dd *DebugDetector) ExportDetectionReport(sessionID string) (string, error) {
	result := dd.DetectAll(sessionID, "", "", nil, nil)
	
	detections := dd.GetSessionDetections(sessionID)
	result.Detections = detections

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (dd *DebugDetector) AnalyzeSessionPattern(sessionID string) (bool, string) {
	detections := dd.GetSessionDetections(sessionID)
	
	if len(detections) < 3 {
		return false, "insufficient_data"
	}

	typeCount := make(map[model.DebugDetectionType]int)
	for _, d := range detections {
		typeCount[d.Type]++
	}

	for detectionType, count := range typeCount {
		if count >= 5 {
			return true, fmt.Sprintf("high_frequency_%s", detectionType)
		}
	}

	severityEscalation := false
	for i := 1; i < len(detections); i++ {
		if detections[i].Severity > detections[i-1].Severity {
			severityEscalation = true
			break
		}
	}

	if severityEscalation && len(typeCount) >= 3 {
		return true, "escalating_pattern_with_multiple_types"
	}

	return false, "normal"
}

func NewWebhookDebugAlertHandler(endpoint string) *WebhookDebugAlertHandler {
	return &WebhookDebugAlertHandler{
		endpoint:   endpoint,
		httpClient: &DefaultHTTPClient{},
		timeout:    10 * time.Second,
	}
}

func (h *WebhookDebugAlertHandler) HandleAlert(alert *DebugAlert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	return h.httpClient.Post(h.endpoint, data)
}

func NewLogDebugAlertHandler() *LogDebugAlertHandler {
	return &LogDebugAlertHandler{
		logger: &DefaultLogger{},
	}
}

func (h *LogDebugAlertHandler) HandleAlert(alert *DebugAlert) error {
	h.logger.Error("Debug Alert: ID=%s, Type=%s, Severity=%d, SessionID=%s, IP=%s, Message=%s",
		alert.ID, alert.Type, alert.Severity, alert.SessionID, alert.IPAddress, alert.Message)
	return nil
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen] + "..."
}

func GenerateDebugDetectionToken(sessionID, clientIP, userAgent string, timestamp int64) string {
	data := fmt.Sprintf("%s:%s:%s:%d", sessionID, clientIP, userAgent, timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func VerifyDebugDetectionToken(token, sessionID, clientIP, userAgent string, maxAge time.Duration) bool {
	if len(token) != 64 {
		return false
	}

	data := fmt.Sprintf("%s:%s:%s:", sessionID, clientIP, userAgent)
	hash := sha256.Sum256([]byte(data))
	expectedPrefix := hex.EncodeToString(hash[:])

	if !strings.HasPrefix(token, expectedPrefix) {
		return false
	}

	timestampStr := token[64:]
	if len(timestampStr) < 10 {
		return false
	}

	timestampPart := timestampStr[:10]
	timestamp, err := strconv.ParseInt(timestampPart, 10, 64)
	if err != nil {
		return false
	}

	tokenTime := time.Unix(timestamp, 0)
	if time.Since(tokenTime) > maxAge {
		return false
	}

	return true
}

type HTTPRequestData struct {
	SessionID string
	ClientIP  string
	UserAgent string
	Headers   http.Header
	Body      []byte
}

func ParseHTTPRequestData(req *http.Request) *HTTPRequestData {
	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	clientIP := getClientIP(req)
	userAgent := req.Header.Get("User-Agent")

	return &HTTPRequestData{
		SessionID: req.Header.Get("X-Session-ID"),
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Headers:   req.Header,
		Body:      body,
	}
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}

func DetectDebugFromRequest(req *http.Request) *model.DebugDetectionResult {
	data := ParseHTTPRequestData(req)
	
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	sessionID := data.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("auto-%s-%d", data.ClientIP, time.Now().Unix())
	}

	return detector.DetectAll(sessionID, data.ClientIP, data.UserAgent, data.Headers, data.Body)
}
