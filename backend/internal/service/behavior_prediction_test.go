package service

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestBehaviorPredictionRealtime(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	traceData := generatePredictionTestData(150, 50)

	startTime := time.Now()
	result, err := prediction.Predict(traceData)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if result == nil {
		t.Fatal("Prediction result is nil")
	}

	if result.RiskScore < 0 || result.RiskScore > 1 {
		t.Errorf("RiskScore out of range [0,1]: %f", result.RiskScore)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence out of range [0,1]: %f", result.Confidence)
	}

	if elapsed.Milliseconds() > 80 {
		t.Logf("WARNING: Prediction took %dms, exceeding 80ms target", elapsed.Milliseconds())
	} else {
		t.Logf("Prediction completed in %dms", elapsed.Milliseconds())
	}

	t.Logf("Prediction: risk=%.2f, confidence=%.2f, anomalies=%d",
		result.RiskScore, result.Confidence, len(result.Anomalies))
}

func TestBehaviorPredictionRiskAssessment(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	traceData := generatePredictionTestData(120, 40)

	assessment, err := prediction.AssessRisk(traceData)
	if err != nil {
		t.Fatalf("AssessRisk failed: %v", err)
	}

	if assessment == nil {
		t.Fatal("Risk assessment is nil")
	}

	expectedLevels := map[string]bool{
		"minimal": true,
		"low":     true,
		"medium":  true,
		"high":    true,
	}

	if !expectedLevels[assessment.RiskLevel] {
		t.Errorf("Invalid risk level: %s", assessment.RiskLevel)
	}

	if assessment.OverallRisk < 0 || assessment.OverallRisk > 1 {
		t.Errorf("OverallRisk out of range [0,1]: %f", assessment.OverallRisk)
	}

	if len(assessment.Recommendations) == 0 {
		t.Log("WARNING: No recommendations generated")
	}

	t.Logf("Risk assessment: level=%s, overall=%.2f, factors=%d, recommendations=%d",
		assessment.RiskLevel, assessment.OverallRisk, len(assessment.RiskFactors), len(assessment.Recommendations))
}

func TestBehaviorPredictionBatch(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	traces := make([]*model.TraceData, 20)
	for i := 0; i < 20; i++ {
		traces[i] = generatePredictionTestData(100+i*2, 30+i)
	}

	startTime := time.Now()
	results, err := prediction.PredictBatch(traces)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("PredictBatch failed: %v", err)
	}

	if len(results) != 20 {
		t.Errorf("Expected 20 results, got %d", len(results))
	}

	if elapsed.Milliseconds() > 1600 {
		t.Logf("WARNING: Batch prediction took %dms for 20 traces", elapsed.Milliseconds())
	} else {
		t.Logf("Batch prediction completed in %dms", elapsed.Milliseconds())
	}

	var avgRisk float64
	for _, result := range results {
		avgRisk += result.RiskScore
	}
	avgRisk /= float64(len(results))

	t.Logf("Batch results: avg_risk=%.2f, time=%dms", avgRisk, elapsed.Milliseconds())
}

func TestBehaviorPredictionMetrics(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	for i := 0; i < 10; i++ {
		traceData := generatePredictionTestData(100+i*5, 35+i*2)
		prediction.Predict(traceData)
	}

	metrics := prediction.GetMetrics()

	if metrics == nil {
		t.Fatal("Metrics is nil")
	}

	if metrics.TotalPredictions != 10 {
		t.Errorf("Expected 10 predictions, got %d", metrics.TotalPredictions)
	}

	if metrics.AvgRiskScore < 0 || metrics.AvgRiskScore > 1 {
		t.Errorf("AvgRiskScore out of range: %f", metrics.AvgRiskScore)
	}

	t.Logf("Metrics: total=%d, avg_risk=%.2f, high_risk=%d",
		metrics.TotalPredictions, metrics.AvgRiskScore, metrics.HighRiskCount)
}

func TestBehaviorPredictionSequence(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	traces := make([]*model.TraceData, 5)
	for i := 0; i < 5; i++ {
		traces[i] = generatePredictionTestData(80+i*10, 30+i*3)
	}

	analysis, err := prediction.AnalyzeSequence(traces)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("Sequence analysis is nil")
	}

	if sequenceLength, ok := analysis["sequence_length"].(int); ok {
		if sequenceLength != 5 {
			t.Errorf("Expected sequence length 5, got %d", sequenceLength)
		}
	}

	t.Logf("Sequence analysis: trend=%v, mean_risk=%.2f",
		analysis["trend"], analysis["mean_risk_score"])
}

func TestBehaviorPredictionBuffer(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	for i := 0; i < 15; i++ {
		traceData := generatePredictionTestData(90+i*3, 25+i*2)
		prediction.Predict(traceData)
	}

	recent := prediction.GetRecentPredictions(10)
	if len(recent) != 10 {
		t.Errorf("Expected 10 recent predictions, got %d", len(recent))
	}

	prediction.ClearBuffer()
	recentAfterClear := prediction.GetRecentPredictions(10)
	if len(recentAfterClear) != 0 {
		t.Errorf("Expected 0 predictions after clear, got %d", len(recentAfterClear))
	}

	t.Logf("Buffer test passed: recent=%d, after_clear=%d", len(recent), len(recentAfterClear))
}

func TestBehaviorPredictionCompare(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	trace1 := generatePredictionTestData(100, 40)
	trace2 := generatePredictionTestData(100, 40)

	similarity, err := prediction.CompareTraces(trace1, trace2)
	if err != nil {
		t.Fatalf("CompareTraces failed: %v", err)
	}

	if similarity < 0 {
		t.Error("Similarity should be non-negative")
	}

	t.Logf("Trace similarity: %f", similarity)
}

func TestBehaviorPredictionReset(t *testing.T) {
	prediction := NewRealtimeBehaviorPrediction()

	for i := 0; i < 5; i++ {
		traceData := generatePredictionTestData(85+i*5, 28+i*2)
		prediction.Predict(traceData)
	}

	metricsBefore := prediction.GetMetrics()

	prediction.Reset()

	metricsAfter := prediction.GetMetrics()

	if metricsAfter.TotalPredictions != 0 {
		t.Errorf("Expected 0 predictions after reset, got %d", metricsAfter.TotalPredictions)
	}

	t.Logf("Reset test: before=%d predictions, after=%d predictions",
		metricsBefore.TotalPredictions, metricsAfter.TotalPredictions)
}

func generatePredictionTestData(pointCount, spacing int) *model.TraceData {
	traceData := &model.TraceData{
		Points: make([]model.TracePoint, pointCount),
	}

	baseTime := int64(4000000)
	for i := 0; i < pointCount; i++ {
		traceData.Points[i] = model.TracePoint{
			X:         150 + i*spacing + (i%5)*10,
			Y:         150 + (i/2)*spacing - (i%4)*spacing/3,
			Timestamp: baseTime + int64(i*14),
			Pressure:  0.55 + float64(i%9)*0.04,
			TouchSize: 11.0 + float64(i%7)*1.8,
			Event:     "move",
		}
	}

	return traceData
}
