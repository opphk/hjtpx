package tools

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewDebugDetector(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	if detector == nil {
		t.Fatal("Expected non-nil DebugDetector")
	}

	if detector.config != config {
		t.Error("Expected config to match")
	}

	if detector.detector == nil {
		t.Error("Expected detector to be initialized")
	}

	if detector.timingTracker == nil {
		t.Error("Expected timingTracker to be initialized")
	}

	if detector.breakpointMgr == nil {
		t.Error("Expected breakpointMgr to be initialized")
	}

	if detector.consoleMonitor == nil {
		t.Error("Expected consoleMonitor to be initialized")
	}
}

func TestDebugDetectorDetectDevTools(t *testing.T) {
	tests := []struct {
		name         string
		headers      map[string]string
		expectDetect bool
		severity     model.DebugSeverity
	}{
		{
			name:         "DevTools emulation header",
			headers:      map[string]string{"X-DevTools-Emulate": "true"},
			expectDetect: true,
			severity:     model.SeverityHigh,
		},
		{
			name:         "DevTools cache manipulation",
			headers:      map[string]string{"Sec-Use-H5cache": "false"},
			expectDetect: true,
			severity:     model.SeverityMedium,
		},
		{
			name:         "DevTools header",
			headers:      map[string]string{"DevTools-Active": "true"},
			expectDetect: true,
			severity:     model.SeverityHigh,
		},
		{
			name:         "Chrome DevTools specific header",
			headers:      map[string]string{"X-Debug-Panel": "open"},
			expectDetect: true,
			severity:     model.SeverityCritical,
		},
		{
			name:         "Normal request",
			headers:      map[string]string{"Accept": "text/html"},
			expectDetect: false,
			severity:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			detector := NewDebugDetector(config)

			headers := http.Header{}
			for k, v := range tt.headers {
				headers.Set(k, v)
			}

			result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

			if tt.expectDetect {
				if !result.IsDetected {
					t.Errorf("Expected detection for %s", tt.name)
				}

				found := false
				for _, detection := range result.Detections {
					if detection.Type == model.DebugTypeDevTools && detection.Severity == tt.severity {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected DevTools detection with severity %d", tt.severity)
				}
			} else {
				if result.IsDetected {
					t.Errorf("Expected no detection for %s", tt.name)
				}
			}
		})
	}
}

func TestDebugDetectorDetectDebugger(t *testing.T) {
	tests := []struct {
		name         string
		headers      map[string]string
		body         []byte
		expectDetect bool
		severity     model.DebugSeverity
	}{
		{
			name:         "Debug mode header",
			headers:      map[string]string{"X-Debug-Mode": "true"},
			body:         nil,
			expectDetect: true,
			severity:     model.SeverityCritical,
		},
		{
			name:         "Debugger statement in body",
			headers:      nil,
			body:         []byte("function test() { debugger; return true; }"),
			expectDetect: true,
			severity:     model.SeverityHigh,
		},
		{
			name:         "Debugger with space",
			headers:      nil,
			body:         []byte("debugger; "),
			expectDetect: true,
			severity:     model.SeverityHigh,
		},
		{
			name:         "Breakpoint header",
			headers:      map[string]string{"X-Breakpoint": "main.js:10"},
			body:         nil,
			expectDetect: true,
			severity:     model.SeverityHigh,
		},
		{
			name:         "Debug break header",
			headers:      map[string]string{"X-Debug-Break": "true"},
			body:         nil,
			expectDetect: true,
			severity:     model.SeverityHigh,
		},
		{
			name:         "Normal body",
			headers:      nil,
			body:         []byte("console.log('hello');"),
			expectDetect: false,
			severity:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			detector := NewDebugDetector(config)

			headers := http.Header{}
			if tt.headers != nil {
				for k, v := range tt.headers {
					headers.Set(k, v)
				}
			}

			result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, tt.body)

			if tt.expectDetect {
				if !result.IsDetected {
					t.Errorf("Expected detection for %s", tt.name)
				}

				found := false
				for _, detection := range result.Detections {
					if detection.Type == model.DebugTypeDebugger && detection.Severity >= tt.severity {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected debugger detection with severity >= %d", tt.severity)
				}
			}
		})
	}
}

func TestDebugDetectorDetectBreakpoints(t *testing.T) {
	tests := []struct {
		name         string
		headers      map[string]string
		sessionID    string
		bpCount      int
		expectDetect bool
	}{
		{
			name:         "Breakpoint line header",
			headers:      map[string]string{"X-Breakpoint-Line": "42"},
			sessionID:    "session1",
			bpCount:      0,
			expectDetect: true,
		},
		{
			name:         "Debug breakpoint header",
			headers:      map[string]string{"X-Debug-Breakpoint": "function:test"},
			sessionID:    "session2",
			bpCount:      0,
			expectDetect: true,
		},
		{
			name:         "Excessive breakpoint monitoring",
			headers:      nil,
			sessionID:    "session3",
			bpCount:      10,
			expectDetect: true,
		},
		{
			name:         "Normal request",
			headers:      map[string]string{"Accept": "application/json"},
			sessionID:    "session4",
			bpCount:      0,
			expectDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			config.BreakpointThreshold = 3
			detector := NewDebugDetector(config)

			if tt.bpCount > 0 {
				detector.breakpointMgr.mu.Lock()
				detector.breakpointMgr.monitorCount[tt.sessionID] = tt.bpCount
				detector.breakpointMgr.mu.Unlock()
			}

			headers := http.Header{}
			if tt.headers != nil {
				for k, v := range tt.headers {
					headers.Set(k, v)
				}
			}

			result := detector.DetectAll(tt.sessionID, "192.168.1.1", "Mozilla/5.0", headers, nil)

			if tt.expectDetect {
				found := false
				for _, detection := range result.Detections {
					if detection.Type == model.DebugTypeBreakpoint {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected breakpoint detection for %s", tt.name)
				}
			}
		})
	}
}

func TestDebugDetectorDetectTimingAnomaly(t *testing.T) {
	tests := []struct {
		name         string
		timingHeader string
		baseline     time.Duration
		expectDetect bool
	}{
		{
			name:         "Normal timing",
			timingHeader: "100",
			baseline:     100 * time.Millisecond,
			expectDetect: false,
		},
		{
			name:         "Massive delay (500%+ deviation)",
			timingHeader: "1000",
			baseline:     100 * time.Millisecond,
			expectDetect: true,
		},
		{
			name:         "Massive speedup (-80% deviation)",
			timingHeader: "10",
			baseline:     100 * time.Millisecond,
			expectDetect: true,
		},
		{
			name:         "No timing header",
			timingHeader: "",
			baseline:     0,
			expectDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			detector := NewDebugDetector(config)

			sessionID := "timing-test-session"
			detector.timingTracker.mu.Lock()
			detector.timingTracker.baselineTime[sessionID] = tt.baseline
			detector.timingTracker.mu.Unlock()

			headers := http.Header{}
			if tt.timingHeader != "" {
				headers.Set("X-Request-Timing", tt.timingHeader)
			}

			result := detector.DetectAll(sessionID, "192.168.1.1", "Mozilla/5.0", headers, nil)

			if tt.expectDetect {
				found := false
				for _, detection := range result.Detections {
					if detection.Type == model.DebugTypeTimingAnomaly {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected timing anomaly detection for %s", tt.name)
				}
			}
		})
	}
}

func TestDebugDetectorDetectConsoleActivity(t *testing.T) {
	tests := []struct {
		name         string
		headers      map[string]string
		expectDetect bool
		expectedType string
	}{
		{
			name:         "No console methods",
			headers:      map[string]string{"X-Console-Methods": ""},
			expectDetect: true,
			expectedType: "no_console_methods",
		},
		{
			name:         "Console overridden",
			headers:      map[string]string{"X-Console-Overridden": "true"},
			expectDetect: true,
			expectedType: "console_override",
		},
		{
			name:         "Debug console method",
			headers:      map[string]string{"X-Console-Methods": "log,warn,error,debug"},
			expectDetect: true,
			expectedType: "suspicious_console_method",
		},
		{
			name:         "Trace console method",
			headers:      map[string]string{"X-Console-Methods": "log,trace"},
			expectDetect: true,
			expectedType: "suspicious_console_method",
		},
		{
			name:         "Normal console methods",
			headers:      map[string]string{"X-Console-Methods": "log,warn,error"},
			expectDetect: false,
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			detector := NewDebugDetector(config)

			headers := http.Header{}
			for k, v := range tt.headers {
				headers.Set(k, v)
			}

			result := detector.DetectAll("console-test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

			if tt.expectDetect {
				found := false
				for _, detection := range result.Detections {
					if detection.Type == model.DebugTypeConsoleActivity {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected console activity detection for %s", tt.name)
				}
			}
		})
	}
}

func TestDebugDetectorDetectMemoryAccess(t *testing.T) {
	tests := []struct {
		name         string
		headers      map[string]string
		expectDetect bool
	}{
		{
			name:         "Prototype modification",
			headers:      map[string]string{"X-__proto__-Modified": "true"},
			expectDetect: true,
		},
		{
			name:         "Constructor modification",
			headers:      map[string]string{"X-constructor-Modified": "true"},
			expectDetect: true,
		},
		{
			name:         "Object freeze disabled",
			headers:      map[string]string{"X-Object-Freeze": "disabled"},
			expectDetect: true,
		},
		{
			name:         "Object seal disabled",
			headers:      map[string]string{"X-Object-Seal": "disabled"},
			expectDetect: true,
		},
		{
			name:         "Normal request",
			headers:      map[string]string{"Accept": "text/html"},
			expectDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			detector := NewDebugDetector(config)

			headers := http.Header{}
			for k, v := range tt.headers {
				headers.Set(k, v)
			}

			result := detector.DetectAll("memory-test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

			if tt.expectDetect {
				found := false
				for _, detection := range result.Detections {
					if detection.Type == model.DebugTypeMemoryAccess {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected memory access detection for %s", tt.name)
				}
			}
		})
	}
}

func TestDebugDetectorDetectCallStackDepth(t *testing.T) {
	tests := []struct {
		name         string
		depthHeader  string
		expectDetect bool
	}{
		{
			name:         "Excessive depth",
			depthHeader:  "100",
			expectDetect: true,
		},
		{
			name:         "Normal depth",
			depthHeader:  "10",
			expectDetect: false,
		},
		{
			name:         "No depth header",
			depthHeader:  "",
			expectDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			detector := NewDebugDetector(config)

			headers := http.Header{}
			if tt.depthHeader != "" {
				headers.Set("X-Call-Stack-Depth", tt.depthHeader)
			}

			result := detector.DetectAll("stack-test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

			if tt.expectDetect {
				found := false
				for _, detection := range result.Detections {
					if detection.Type == model.DebugTypeCallStackDepth {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected call stack depth detection for %s", tt.name)
				}
			}
		})
	}
}

func TestDebugDetectorMultipleDetections(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-DevTools-Emulate", "true")
	headers.Set("X-Debug-Mode", "true")
	headers.Set("X-Breakpoint-Line", "42")
	headers.Set("X-Console-Methods", "debug")

	result := detector.DetectAll("multi-detect-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	if result.DetectionCount < 4 {
		t.Errorf("Expected at least 4 detections, got %d", result.DetectionCount)
	}

	if result.RiskScore == 0 {
		t.Error("Expected non-zero risk score")
	}

	if result.HighestSeverity < model.SeverityHigh {
		t.Errorf("Expected highest severity >= %d, got %d", model.SeverityHigh, result.HighestSeverity)
	}
}

func TestDebugDetectorBlockOnDetection(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	config.BlockOnDetection = true
	config.MinSeverityToBlock = model.SeverityMedium
	config.DetectionThreshold = 1

	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-Debug-Mode", "true")

	result := detector.DetectAll("block-test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	if !result.ShouldBlock {
		t.Error("Expected session to be blocked")
	}

	if !detector.IsIPBlocked("192.168.1.1") {
		t.Error("Expected IP to be blocked")
	}
}

func TestDebugDetectorNoBlockOnLowSeverity(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	config.BlockOnDetection = true
	config.MinSeverityToBlock = model.SeverityCritical
	config.DetectionThreshold = 10

	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-Console-Methods", "log,warn")

	result := detector.DetectAll("no-block-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	if result.ShouldBlock {
		t.Error("Expected session not to be blocked")
	}
}

func TestDebugDetectorStats(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-DevTools-Emulate", "true")

	detector.DetectAll("stats-test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	stats := detector.GetStats()

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.TotalDetections == 0 {
		t.Error("Expected at least one detection")
	}

	if stats.ActiveSessions == 0 {
		t.Error("Expected at least one active session")
	}
}

func TestDebugDetectorSessionDetections(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	sessionID := "session-detections-test"

	headers1 := http.Header{}
	headers1.Set("X-DevTools-Emulate", "true")
	detector.DetectAll(sessionID, "192.168.1.1", "Mozilla/5.0", headers1, nil)

	headers2 := http.Header{}
	headers2.Set("X-Debug-Mode", "true")
	detector.DetectAll(sessionID, "192.168.1.1", "Mozilla/5.0", headers2, nil)

	detections := detector.GetSessionDetections(sessionID)

	if len(detections) < 2 {
		t.Errorf("Expected at least 2 detections for session, got %d", len(detections))
	}
}

func TestDebugDetectorConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *model.DebugDetectorConfig
		expectValid bool
	}{
		{
			name: "Valid config",
			config: &model.DebugDetectorConfig{
				MaxDetectionsPerSession: 10,
				DetectionThreshold:      5,
				TimeWindow:              5 * time.Minute,
				AutoBlockDuration:       10 * time.Minute,
			},
			expectValid: true,
		},
		{
			name: "Invalid - zero max detections",
			config: &model.DebugDetectorConfig{
				MaxDetectionsPerSession: 0,
				DetectionThreshold:      5,
				TimeWindow:              5 * time.Minute,
				AutoBlockDuration:       10 * time.Minute,
			},
			expectValid: false,
		},
		{
			name: "Invalid - zero threshold",
			config: &model.DebugDetectorConfig{
				MaxDetectionsPerSession: 10,
				DetectionThreshold:      0,
				TimeWindow:              5 * time.Minute,
				AutoBlockDuration:       10 * time.Minute,
			},
			expectValid: false,
		},
		{
			name: "Invalid - zero time window",
			config: &model.DebugDetectorConfig{
				MaxDetectionsPerSession: 10,
				DetectionThreshold:      5,
				TimeWindow:              0,
				AutoBlockDuration:       10 * time.Minute,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.config.IsValid()
			if isValid != tt.expectValid {
				t.Errorf("Expected IsValid() = %v, got %v", tt.expectValid, isValid)
			}
		})
	}
}

func TestDebugDetectorUpdateConfig(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	newConfig := &model.DebugDetectorConfig{
		Enabled:                 true,
		CheckDevTools:           false,
		CheckDebugger:           true,
		CheckBreakpoints:        true,
		CheckTimingAnomaly:      true,
		CheckConsoleActivity:    true,
		CheckMemoryAccess:       true,
		CheckCallStackDepth:     true,
		BlockOnDetection:        true,
		LogDetections:           true,
		MaxDetectionsPerSession: 5,
		DetectionThreshold:      3,
		TimeWindow:              3 * time.Minute,
		AutoBlockDuration:       5 * time.Minute,
		MinSeverityToBlock:      model.SeverityMedium,
	}

	detector.SetConfig(newConfig)

	retrievedConfig := detector.GetConfig()
	if retrievedConfig.CheckDevTools != false {
		t.Error("Expected CheckDevTools to be updated to false")
	}

	if retrievedConfig.MaxDetectionsPerSession != 5 {
		t.Errorf("Expected MaxDetectionsPerSession to be 5, got %d", retrievedConfig.MaxDetectionsPerSession)
	}
}

func TestDebugDetectorStartStop(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	err := detector.Start()
	if err != nil {
		t.Errorf("Expected Start() to succeed, got error: %v", err)
	}

	err = detector.Start()
	if err == nil {
		t.Error("Expected Start() to fail when already running")
	}

	detector.Stop()

	err = detector.Start()
	if err != nil {
		t.Errorf("Expected Start() to succeed after Stop(), got error: %v", err)
	}

	detector.Stop()
}

func TestDebugDetectorTrackExecution(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	sessionID := "timing-exec-session"
	operationName := "testOperation"

	done := detector.TrackExecution(sessionID, operationName)
	time.Sleep(10 * time.Millisecond)
	done()

	detector.timingTracker.mu.RLock()
	defer detector.timingTracker.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", sessionID, operationName)
	record, exists := detector.timingTracker.executions[key]

	if !exists {
		t.Error("Expected execution record to exist")
	}

	if record.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, record.SessionID)
	}

	if record.Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", record.Duration)
	}
}

func TestDebugDetectorRecordBreakpointHit(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	sessionID := "bp-hit-session"
	detector.RecordBreakpointHit(sessionID, "debugger", 42, "testFunction")
	detector.RecordBreakpointHit(sessionID, "debugger", 42, "testFunction")
	detector.RecordBreakpointHit(sessionID, "debugger", 50, "anotherFunction")

	detector.breakpointMgr.mu.RLock()
	defer detector.breakpointMgr.mu.RUnlock()

	if detector.breakpointMgr.monitorCount[sessionID] != 3 {
		t.Errorf("Expected monitor count 3, got %d", detector.breakpointMgr.monitorCount[sessionID])
	}

	key := fmt.Sprintf("%s:%d:%s", sessionID, 42, "testFunction")
	info, exists := detector.breakpointMgr.breakpoints[key]

	if !exists {
		t.Error("Expected breakpoint info to exist")
	}

	if info.HitCount != 2 {
		t.Errorf("Expected hit count 2, got %d", info.HitCount)
	}
}

func TestDebugDetectorSecurityToken(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	sessionID := "token-test-session"
	clientIP := "192.168.1.100"
	userAgent := "Mozilla/5.0"

	token := detector.GenerateSecurityToken(sessionID, clientIP, userAgent)

	if token == "" {
		t.Error("Expected non-empty token")
	}

	if len(token) != 32 {
		t.Errorf("Expected token length 32, got %d", len(token))
	}

	valid := detector.ValidateSecurityToken(token, sessionID, clientIP, userAgent)
	if !valid {
		t.Error("Expected token to be valid")
	}

	invalidToken := "invalid-token-length-too-short"
	valid = detector.ValidateSecurityToken(invalidToken, sessionID, clientIP, userAgent)
	if valid {
		t.Error("Expected invalid token to fail validation")
	}
}

func TestDebugDetectorAnalyzeSessionPattern(t *testing.T) {
	tests := []struct {
		name             string
		detections       []model.DebugDetectionEvent
		expectSuspicious bool
	}{
		{
			name: "High frequency single type",
			detections: []model.DebugDetectionEvent{
				{Type: model.DebugTypeDevTools, Severity: model.SeverityMedium},
				{Type: model.DebugTypeDevTools, Severity: model.SeverityMedium},
				{Type: model.DebugTypeDevTools, Severity: model.SeverityMedium},
				{Type: model.DebugTypeDevTools, Severity: model.SeverityMedium},
				{Type: model.DebugTypeDevTools, Severity: model.SeverityMedium},
			},
			expectSuspicious: true,
		},
		{
			name: "Escalating with multiple types",
			detections: []model.DebugDetectionEvent{
				{Type: model.DebugTypeDevTools, Severity: model.SeverityLow},
				{Type: model.DebugTypeDebugger, Severity: model.SeverityMedium},
				{Type: model.DebugTypeBreakpoint, Severity: model.SeverityHigh},
				{Type: model.DebugTypeConsoleActivity, Severity: model.SeverityMedium},
			},
			expectSuspicious: true,
		},
		{
			name: "Normal pattern",
			detections: []model.DebugDetectionEvent{
				{Type: model.DebugTypeDevTools, Severity: model.SeverityLow},
				{Type: model.DebugTypeDebugger, Severity: model.SeverityMedium},
			},
			expectSuspicious: false,
		},
		{
			name:             "Insufficient data",
			detections:       []model.DebugDetectionEvent{},
			expectSuspicious: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := model.NewDebugDetectorConfig()
			detector := NewDebugDetector(config)

			sessionID := "pattern-test-session"

			for _, detection := range tt.detections {
				detection.SessionID = sessionID
				detector.detector.AddDetection(sessionID, detection)
			}

			suspicious, _ := detector.AnalyzeSessionPattern(sessionID)

			if suspicious != tt.expectSuspicious {
				t.Errorf("Expected suspicious = %v, got %v", tt.expectSuspicious, suspicious)
			}
		})
	}
}

func TestDebugDetectionResult(t *testing.T) {
	result := model.NewDebugDetectionResult()

	if result.IsDetected {
		t.Error("Expected IsDetected to be false initially")
	}

	if result.DetectionCount != 0 {
		t.Errorf("Expected DetectionCount to be 0, got %d", result.DetectionCount)
	}

	event1 := model.DebugDetectionEvent{
		Type:     model.DebugTypeDevTools,
		Severity: model.SeverityMedium,
	}
	event2 := model.DebugDetectionEvent{
		Type:     model.DebugTypeDebugger,
		Severity: model.SeverityHigh,
	}

	result.AddDetection(event1)
	result.AddDetection(event2)

	if !result.IsDetected {
		t.Error("Expected IsDetected to be true after adding detections")
	}

	if result.DetectionCount != 2 {
		t.Errorf("Expected DetectionCount to be 2, got %d", result.DetectionCount)
	}

	if result.HighestSeverity != model.SeverityHigh {
		t.Errorf("Expected HighestSeverity to be %d, got %d", model.SeverityHigh, result.HighestSeverity)
	}

	if result.TotalSeverity != int(model.SeverityMedium)+int(model.SeverityHigh) {
		t.Errorf("Expected TotalSeverity to be %d, got %d", 
			int(model.SeverityMedium)+int(model.SeverityHigh), result.TotalSeverity)
	}

	riskScore := result.CalculateRiskScore()
	if riskScore == 0 {
		t.Error("Expected non-zero risk score")
	}
}

func TestDebugDetectionResultShouldBlock(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	config.DetectionThreshold = 3

	result := model.NewDebugDetectionResult()

	if result.ShouldBlockSession(config.DetectionThreshold) {
		t.Error("Expected ShouldBlockSession to be false with 0 detections")
	}

	for i := 0; i < 3; i++ {
		result.AddDetection(model.DebugDetectionEvent{
			Type:     model.DebugTypeDevTools,
			Severity: model.SeverityMedium,
		})
	}

	if !result.ShouldBlockSession(config.DetectionThreshold) {
		t.Error("Expected ShouldBlockSession to be true with >= threshold detections")
	}
}

func TestDebugDetectionResultGetMostSevere(t *testing.T) {
	result := model.NewDebugDetectionResult()

	mostSevere := result.GetMostSevereDetection()
	if mostSevere != nil {
		t.Error("Expected nil when no detections exist")
	}

	events := []model.DebugDetectionEvent{
		{Type: model.DebugTypeDevTools, Severity: model.SeverityLow},
		{Type: model.DebugTypeDebugger, Severity: model.SeverityCritical},
		{Type: model.DebugTypeBreakpoint, Severity: model.SeverityMedium},
	}

	for _, event := range events {
		result.AddDetection(event)
	}

	mostSevere = result.GetMostSevereDetection()
	if mostSevere == nil {
		t.Fatal("Expected non-nil most severe detection")
	}

	if mostSevere.Type != model.DebugTypeDebugger {
		t.Errorf("Expected most severe type to be debugger, got %s", mostSevere.Type)
	}

	if mostSevere.Severity != model.SeverityCritical {
		t.Errorf("Expected severity to be critical, got %d", mostSevere.Severity)
	}
}

func TestDebugDetectionEventMetadata(t *testing.T) {
	event := model.NewDebugDetectionEvent(model.DebugTypeDevTools, model.SeverityMedium)

	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	err := event.SetMetadata(metadata)
	if err != nil {
		t.Errorf("Expected SetMetadata to succeed, got error: %v", err)
	}

	retrieved, err := event.GetMetadata()
	if err != nil {
		t.Errorf("Expected GetMetadata to succeed, got error: %v", err)
	}

	if retrieved["key1"] != "value1" {
		t.Error("Expected key1 to be 'value1'")
	}

	if retrieved["key2"] != 123 {
		t.Error("Expected key2 to be 123")
	}
}

func TestThreadSafeDebugDetector(t *testing.T) {
	detector := model.NewThreadSafeDebugDetector()

	sessionID := "concurrent-test-session"

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := model.DebugDetectionEvent{
				SessionID: sessionID,
				Type:      model.DebugTypeDevTools,
				Severity:  model.DebugSeverity(id % 10),
			}
			detector.AddDetection(sessionID, event)
		}(i)
	}

	wg.Wait()

	count := detector.GetSessionDetectionCount(sessionID)
	if count != 100 {
		t.Errorf("Expected 100 detections, got %d", count)
	}

	detections := detector.GetSessionDetections(sessionID)
	if len(detections) != 100 {
		t.Errorf("Expected 100 detections in slice, got %d", len(detections))
	}
}

func TestThreadSafeDebugDetectorBlockIP(t *testing.T) {
	detector := model.NewThreadSafeDebugDetector()

	ip := "192.168.1.200"
	duration := 5 * time.Minute

	if detector.IsIPBlocked(ip) {
		t.Error("Expected IP not to be blocked initially")
	}

	detector.BlockIP(ip, duration)

	if !detector.IsIPBlocked(ip) {
		t.Error("Expected IP to be blocked after BlockIP()")
	}
}

func TestThreadSafeDebugDetectorCleanup(t *testing.T) {
	detector := model.NewThreadSafeDebugDetector()

	sessionID := "cleanup-test-session"
	detector.AddDetection(sessionID, model.DebugDetectionEvent{
		SessionID:     sessionID,
		Type:          model.DebugTypeDevTools,
		Severity:      model.SeverityMedium,
		DetectionTime: time.Now().Add(-1 * time.Hour),
	})

	detector.CleanupOldSessions(30 * time.Minute)

	count := detector.GetSessionDetectionCount(sessionID)
	if count != 0 {
		t.Errorf("Expected 0 detections after cleanup, got %d", count)
	}
}

func TestParseHTTPRequestData(t *testing.T) {
	reqBody := []byte("test body content")
	
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
	req.Header.Set("X-Session-ID", "test-session-123")
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	req.Header.Set("X-DevTools-Emulate", "true")

	data := ParseHTTPRequestData(req)

	if data.SessionID != "test-session-123" {
		t.Errorf("Expected SessionID 'test-session-123', got '%s'", data.SessionID)
	}

	if data.UserAgent != "TestBrowser/1.0" {
		t.Errorf("Expected UserAgent 'TestBrowser/1.0', got '%s'", data.UserAgent)
	}

	if data.Headers.Get("X-DevTools-Emulate") != "true" {
		t.Error("Expected X-DevTools-Emulate header to be present")
	}

	if string(data.Body) != string(reqBody) {
		t.Error("Expected body to match")
	}
}

func TestDetectDebugFromRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-DevTools-Emulate", "true")
	req.Header.Set("X-Debug-Mode", "true")

	result := DetectDebugFromRequest(req)

	if !result.IsDetected {
		t.Error("Expected debug detection")
	}

	if result.DetectionCount < 2 {
		t.Errorf("Expected at least 2 detections, got %d", result.DetectionCount)
	}
}

func TestGenerateDebugDetectionToken(t *testing.T) {
	sessionID := "token-session"
	clientIP := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	timestamp := time.Now().Unix()

	token := GenerateDebugDetectionToken(sessionID, clientIP, userAgent, timestamp)

	if token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestVerifyDebugDetectionToken(t *testing.T) {
	sessionID := "verify-token-session"
	clientIP := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	timestamp := time.Now().Unix()

	token := GenerateDebugDetectionToken(sessionID, clientIP, userAgent, timestamp)

	maxAge := 5 * time.Minute

	if !VerifyDebugDetectionToken(token, sessionID, clientIP, userAgent, maxAge) {
		t.Error("Expected token to be valid")
	}

	if VerifyDebugDetectionToken("short", sessionID, clientIP, userAgent, maxAge) {
		t.Error("Expected short token to be invalid")
	}
}

func TestDebugDetectionResultJSON(t *testing.T) {
	result := model.NewDebugDetectionResult()
	result.SessionID = "json-test-session"
	result.IPAddress = "192.168.1.1"

	result.AddDetection(model.DebugDetectionEvent{
		SessionID: "json-test-session",
		Type:      model.DebugTypeDevTools,
		Severity:  model.SeverityHigh,
	})

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Errorf("Expected ToJSON to succeed, got error: %v", err)
	}

	if !strings.Contains(jsonStr, "json-test-session") {
		t.Error("Expected JSON to contain session ID")
	}

	parsed, err := model.ParseDebugDetectionResult(jsonStr)
	if err != nil {
		t.Errorf("Expected ParseDebugDetectionResult to succeed, got error: %v", err)
	}

	if parsed.SessionID != "json-test-session" {
		t.Errorf("Expected parsed SessionID 'json-test-session', got '%s'", parsed.SessionID)
	}
}

func BenchmarkDebugDetectorDetect(b *testing.B) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-DevTools-Emulate", "true")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectAll("bench-session", "192.168.1.1", "Mozilla/5.0", headers, nil)
	}
}

func BenchmarkDebugDetectorMultipleChecks(b *testing.B) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-DevTools-Emulate", "true")
	headers.Set("X-Debug-Mode", "true")
	headers.Set("X-Breakpoint-Line", "42")
	headers.Set("X-Console-Methods", "log,warn,error,debug")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectAll("bench-session", "192.168.1.1", "Mozilla/5.0", headers, nil)
	}
}
