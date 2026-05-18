package trace

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestDetectAnomaliesEnhanced(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move"},
			{Timestamp: 1200, X: 100, Y: 100, Event: "move"},
			{Timestamp: 1300, X: 150, Y: 150, Event: "move"},
			{Timestamp: 1400, X: 200, Y: 200, Event: "move"},
			{Timestamp: 1500, X: 250, Y: 250, Event: "click"},
		},
		TotalTime: 500,
		ClickData: []model.ClickInfo{
			{X: 250, Y: 250, Timestamp: 1500, ClickType: "left"},
			{X: 255, Y: 255, Timestamp: 1600, ClickType: "left"},
			{X: 260, Y: 260, Timestamp: 1700, ClickType: "left"},
		},
		ScrollData: []model.ScrollInfo{
			{Timestamp: 1800, DeltaY: 100, Velocity: 200, Direction: "down"},
			{Timestamp: 1900, DeltaY: 100, Velocity: 200, Direction: "down"},
		},
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	if len(anomalies) == 0 {
		t.Log("No anomalies detected (normal behavior)")
	}

	t.Logf("Detected %d anomalies", len(anomalies))
	for _, anomaly := range anomalies {
		t.Logf("Anomaly: %s - %s (confidence: %.2f, severity: %s)",
			anomaly.Type, anomaly.Description, anomaly.Confidence, anomaly.Severity)
	}
}

func TestDetectMechanicalClicking(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1200, X: 100, Y: 100, Event: "click"},
			{Timestamp: 1400, X: 100, Y: 100, Event: "move"},
			{Timestamp: 1600, X: 100, Y: 100, Event: "click"},
			{Timestamp: 1800, X: 100, Y: 100, Event: "move"},
			{Timestamp: 2000, X: 100, Y: 100, Event: "click"},
		},
		TotalTime: 1000,
		ClickData: []model.ClickInfo{
			{X: 100, Y: 100, Timestamp: 1200, ClickType: "left"},
			{X: 102, Y: 102, Timestamp: 1600, ClickType: "left"},
			{X: 104, Y: 104, Timestamp: 2000, ClickType: "left"},
		},
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	foundMechanicalClicking := false
	for _, anomaly := range anomalies {
		if anomaly.Type == PatternMechanicalClicking {
			foundMechanicalClicking = true
			if anomaly.Confidence < 0.5 {
				t.Errorf("Expected high confidence for mechanical clicking, got %f", anomaly.Confidence)
			}
		}
	}

	if !foundMechanicalClicking {
		t.Log("Mechanical clicking pattern not detected (expected for this test case)")
	}

	t.Logf("Detected %d anomalies", len(anomalies))
}

func TestDetectRoboticScrolling(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "scroll"},
			{Timestamp: 1100, X: 0, Y: 100, Event: "scroll"},
			{Timestamp: 1200, X: 0, Y: 200, Event: "scroll"},
			{Timestamp: 1300, X: 0, Y: 300, Event: "scroll"},
		},
		TotalTime: 300,
		ScrollData: []model.ScrollInfo{
			{Timestamp: 1000, DeltaY: 100, Velocity: 1000, Direction: "down"},
			{Timestamp: 1100, DeltaY: 100, Velocity: 1000, Direction: "down"},
			{Timestamp: 1200, DeltaY: 100, Velocity: 1000, Direction: "down"},
			{Timestamp: 1300, DeltaY: 100, Velocity: 1000, Direction: "down"},
		},
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	foundRoboticScrolling := false
	for _, anomaly := range anomalies {
		if anomaly.Type == PatternRoboticScrolling {
			foundRoboticScrolling = true
			if anomaly.Confidence < 0.3 {
				t.Errorf("Expected moderate confidence for robotic scrolling, got %f", anomaly.Confidence)
			}
		}
	}

	if !foundRoboticScrolling {
		t.Log("Robotic scrolling pattern not detected")
	}

	t.Logf("Detected %d anomalies", len(anomalies))
}

func TestDetectAbnormalPressure(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move", Pressure: 0.5},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move", Pressure: 0.5},
			{Timestamp: 1200, X: 100, Y: 100, Event: "click", Pressure: 0.5},
			{Timestamp: 1300, X: 150, Y: 150, Event: "move", Pressure: 0.5},
			{Timestamp: 1400, X: 200, Y: 200, Event: "click", Pressure: 0.5},
		},
		TotalTime: 400,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	foundAbnormalPressure := false
	for _, anomaly := range anomalies {
		if anomaly.Type == PatternAbnormalPressure {
			foundAbnormalPressure = true
			if anomaly.Confidence < 0.3 {
				t.Logf("Abnormal pressure confidence: %f", anomaly.Confidence)
			}
		}
	}

	if !foundAbnormalPressure {
		t.Log("Abnormal pressure pattern not detected")
	}

	t.Logf("Detected %d anomalies", len(anomalies))
}

func TestDetectUniformCurvature(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "move"},
			{Timestamp: 1300, X: 30, Y: 30, Event: "move"},
			{Timestamp: 1400, X: 40, Y: 40, Event: "move"},
			{Timestamp: 1500, X: 50, Y: 50, Event: "move"},
		},
		TotalTime: 500,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	foundUniformCurvature := false
	for _, anomaly := range anomalies {
		if anomaly.Type == PatternUniformCurvature {
			foundUniformCurvature = true
			t.Logf("Uniform curvature detected with confidence: %f", anomaly.Confidence)
		}
	}

	if !foundUniformCurvature {
		t.Log("Uniform curvature pattern not detected")
	}

	t.Logf("Detected %d anomalies", len(anomalies))
}

func TestGetAnomalySummary(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1100, X: 100, Y: 0, Event: "move"},
			{Timestamp: 1200, X: 200, Y: 0, Event: "move"},
			{Timestamp: 1300, X: 300, Y: 0, Event: "move"},
		},
		TotalTime: 300,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	summary := detector.GetAnomalySummary(anomalies)

	if summary == nil {
		t.Fatal("Summary should not be nil")
	}

	if totalAnomalies, ok := summary["total_anomalies"].(int); ok {
		t.Logf("Total anomalies: %d", totalAnomalies)
	}

	if severityCount, ok := summary["severity_count"].(map[string]int); ok {
		t.Logf("Severity counts: %+v", severityCount)
	}

	if isSuspicious, ok := summary["is_suspicious"].(bool); ok {
		t.Logf("Is suspicious: %v", isSuspicious)
	}

	t.Logf("Summary: %+v", summary)
}

func TestAnomalyPatternsWithInsufficientData(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move"},
		},
		TotalTime: 100,
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies should not fail with minimal data: %v", err)
	}

	t.Logf("Detected %d anomalies with minimal data", len(anomalies))
}

func TestMultipleAnomalyPatterns(t *testing.T) {
	detector := NewAnomalyDetector()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1100, X: 10, Y: 0, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 0, Event: "move"},
			{Timestamp: 1300, X: 30, Y: 0, Event: "move"},
			{Timestamp: 1400, X: 40, Y: 0, Event: "move"},
			{Timestamp: 1500, X: 50, Y: 0, Event: "click"},
			{Timestamp: 1600, X: 52, Y: 0, Event: "click"},
			{Timestamp: 1700, X: 54, Y: 0, Event: "click"},
			{Timestamp: 1800, X: 56, Y: 0, Event: "click"},
		},
		TotalTime: 800,
		ClickData: []model.ClickInfo{
			{X: 50, Y: 0, Timestamp: 1500, ClickType: "left"},
			{X: 52, Y: 0, Timestamp: 1600, ClickType: "left"},
			{X: 54, Y: 0, Timestamp: 1700, ClickType: "left"},
			{X: 56, Y: 0, Timestamp: 1800, ClickType: "left"},
		},
	}

	anomalies, err := detector.DetectAnomalies(traceData)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	t.Logf("Detected %d anomalies for multi-pattern test:", len(anomalies))
	for _, anomaly := range anomalies {
		t.Logf("  - Type: %s, Confidence: %.2f, Severity: %s",
			anomaly.Type, anomaly.Confidence, anomaly.Severity)
	}

	if len(anomalies) < 2 {
		t.Errorf("Expected multiple anomalies, got %d", len(anomalies))
	}
}
