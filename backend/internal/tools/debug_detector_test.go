package tools

import (
	"net/http"
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

	if detector.detector == nil {
		t.Error("Expected detector to be initialized")
	}
}

func TestDebugDetectorDetectDevTools(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-DevTools-Emulate", "true")

	result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	if !result.IsDetected {
		t.Error("Expected detection")
	}

	found := false
	for _, detection := range result.Detections {
		if detection.Type == model.DebugTypeDevTools && detection.Severity == model.SeverityHigh {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected DevTools detection with severity High")
	}
}

func TestDebugDetectorDetectDebugger(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-Debug-Mode", "true")

	result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	if !result.IsDetected {
		t.Error("Expected debugger detection")
	}
}

func TestDebugDetectorDetectBreakpoints(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-Breakpoint-Line", "42")

	result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	found := false
	for _, detection := range result.Detections {
		if detection.Type == model.DebugTypeBreakpoint {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected breakpoint detection")
	}
}

func TestDebugDetectorDetectConsoleActivity(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-Console-Overridden", "true")

	result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	found := false
	for _, detection := range result.Detections {
		if detection.Type == model.DebugTypeConsoleActivity {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected console activity detection")
	}
}

func TestDebugDetectorDetectMemoryAccess(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-Object-Freeze", "disabled")

	result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	found := false
	for _, detection := range result.Detections {
		if detection.Type == model.DebugTypeMemoryAccess {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected memory access detection")
	}
}

func TestDebugDetectorDetectCallStackDepth(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	headers := http.Header{}
	headers.Set("X-Call-Stack-Depth", "100")

	result := detector.DetectAll("test-session", "192.168.1.1", "Mozilla/5.0", headers, nil)

	found := false
	for _, detection := range result.Detections {
		if detection.Type == model.DebugTypeCallStackDepth {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected call stack depth detection")
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
}

func TestDebugDetectorStartStop(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	err := detector.Start()
	if err != nil {
		t.Errorf("Expected Start() to succeed, got error: %v", err)
	}

	detector.Stop()
}

func TestDebugDetectorTrackExecution(t *testing.T) {
	config := model.NewDebugDetectorConfig()
	detector := NewDebugDetector(config)

	done := detector.TrackExecution("session1", "testOp")
	time.Sleep(10 * time.Millisecond)
	done()

	detector.timingTracker.mu.RLock()
	defer detector.timingTracker.mu.RUnlock()

	key := "session1:testOp"
	record, exists := detector.timingTracker.executions[key]

	if !exists {
		t.Error("Expected execution record to exist")
	}

	if record.Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", record.Duration)
	}
}

func TestDebugDetectionResult(t *testing.T) {
	result := model.NewDebugDetectionResult()

	if result.IsDetected {
		t.Error("Expected IsDetected to be false initially")
	}

	event := model.DebugDetectionEvent{
		Type:     model.DebugTypeDevTools,
		Severity: model.SeverityMedium,
	}

	result.AddDetection(event)

	if !result.IsDetected {
		t.Error("Expected IsDetected to be true after adding detections")
	}

	if result.DetectionCount != 1 {
		t.Errorf("Expected DetectionCount to be 1, got %d", result.DetectionCount)
	}
}

func TestThreadSafeDebugDetector(t *testing.T) {
	detector := model.NewThreadSafeDebugDetector()

	sessionID := "concurrent-test-session"

	event := model.DebugDetectionEvent{
		SessionID: sessionID,
		Type:      model.DebugTypeDevTools,
		Severity:  model.SeverityMedium,
	}

	detector.AddDetection(sessionID, event)

	count := detector.GetSessionDetectionCount(sessionID)
	if count != 1 {
		t.Errorf("Expected 1 detection, got %d", count)
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
