package service

import (
	"context"
	"testing"
	"time"
)

func TestBehaviorFlowService_AnalyzeFlow(t *testing.T) {
	service := NewBehaviorFlowService()

	dataPoints := []FlowDataPoint{
		{X: 100, Y: 100, Z: 0, Timestamp: time.Now(), Velocity: 5.0, EventType: "move"},
		{X: 150, Y: 150, Z: 0, Timestamp: time.Now().Add(100 * time.Millisecond), Velocity: 7.0, EventType: "move"},
		{X: 200, Y: 200, Z: 0, Timestamp: time.Now().Add(200 * time.Millisecond), Velocity: 6.0, EventType: "click"},
		{X: 250, Y: 250, Z: 0, Timestamp: time.Now().Add(300 * time.Millisecond), Velocity: 8.0, EventType: "move"},
		{X: 300, Y: 300, Z: 0, Timestamp: time.Now().Add(400 * time.Millisecond), Velocity: 5.5, EventType: "click"},
	}

	flow, err := service.AnalyzeFlow(context.Background(), "user123", "session456", dataPoints)
	if err != nil {
		t.Fatalf("AnalyzeFlow failed: %v", err)
	}

	if flow.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", flow.UserID)
	}

	if flow.SessionID != "session456" {
		t.Errorf("Expected SessionID 'session456', got '%s'", flow.SessionID)
	}

	if len(flow.DataPoints) != len(dataPoints) {
		t.Errorf("Expected %d data points, got %d", len(dataPoints), len(flow.DataPoints))
	}

	if flow.Features["total_distance"] <= 0 {
		t.Error("Total distance should be positive")
	}

	if flow.Features["avg_velocity"] <= 0 {
		t.Error("Average velocity should be positive")
	}
}

func TestBehaviorFlowService_ComputeTemporalFeatures(t *testing.T) {
	service := NewBehaviorFlowService()
	flow := &SpatioTemporalFlow{}

	dataPoints := []FlowDataPoint{
		{X: 0, Y: 0, Z: 0, Timestamp: time.Now()},
		{X: 10, Y: 10, Z: 0, Timestamp: time.Now().Add(100 * time.Millisecond)},
		{X: 20, Y: 20, Z: 0, Timestamp: time.Now().Add(200 * time.Millisecond)},
		{X: 30, Y: 30, Z: 0, Timestamp: time.Now().Add(300 * time.Millisecond)},
	}
	flow.DataPoints = dataPoints

	service.computeTemporalFeatures(flow)

	if flow.Features["data_point_count"] != 4 {
		t.Errorf("Expected data point count 4, got %.0f", flow.Features["data_point_count"])
	}

	if flow.Features["total_duration"] <= 0 {
		t.Error("Total duration should be positive")
	}
}

func TestBehaviorFlowService_ComputeSpatialFeatures(t *testing.T) {
	service := NewBehaviorFlowService()
	flow := &SpatioTemporalFlow{}

	dataPoints := []FlowDataPoint{
		{X: 0, Y: 0, Z: 0, Timestamp: time.Now()},
		{X: 100, Y: 100, Z: 0, Timestamp: time.Now().Add(1 * time.Second)},
	}
	flow.DataPoints = dataPoints

	service.computeSpatialFeatures(flow)

	if flow.Features["total_distance"] <= 0 {
		t.Error("Total distance should be positive")
	}

	if flow.Features["efficiency"] <= 0 {
		t.Error("Efficiency should be positive")
	}
}

func TestBehaviorFlowService_IdentifyPhase(t *testing.T) {
	service := NewBehaviorFlowService()

	testCases := []struct {
		name           string
		duration       time.Duration
		velocityVar    float64
		anomalyScore   float64
		expectedPhase  string
	}{
		{"short flow", 3 * time.Second, 0.3, 0.2, "initial_stable"},
		{"medium flow dynamic", 60 * time.Second, 0.7, 0.3, "interaction_dynamic"},
		{"long flow stable", 150 * time.Second, 0.2, 0.5, "extended_stable_anomalous"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flow := &SpatioTemporalFlow{
				TotalDuration: tc.duration,
				Features: map[string]float64{
					"velocity_variance": tc.velocityVar,
					"anomaly_score":    tc.anomalyScore,
				},
			}

			service.identifyPhase(flow)

			if flow.Phase != tc.expectedPhase {
				t.Errorf("Expected phase '%s', got '%s'", tc.expectedPhase, flow.Phase)
			}
		})
	}
}

func TestBehaviorFlowService_DetectAnomalies(t *testing.T) {
	service := NewBehaviorFlowService()

	testCases := []struct {
		name           string
		features       map[string]float64
		expectedScore  float64
		expectedRisk   string
	}{
		{
			"normal behavior",
			map[string]float64{
				"velocity_variance":    0.3,
				"interval_variance":   0.3,
				"efficiency":         0.7,
				"avg_velocity":       50,
				"acceleration_variance": 0.3,
				"spatial_entropy":    0.6,
			},
			0.0,
			"low",
		},
		{
			"high velocity anomaly",
			map[string]float64{
				"velocity_variance":    0.5,
				"interval_variance":   0.3,
				"efficiency":         0.98,
				"avg_velocity":       1500,
				"acceleration_variance": 0.01,
				"spatial_entropy":    0.6,
			},
			1.05,
			"critical",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flow := &SpatioTemporalFlow{
				DataPoints: make([]FlowDataPoint, 20),
				Features:   tc.features,
			}

			service.detectAnomalies(flow)

			if flow.AnomalyScore < tc.expectedScore*0.8 || flow.AnomalyScore > tc.expectedScore*1.2 {
				t.Errorf("Expected anomaly score around %.2f, got %.2f", tc.expectedScore, flow.AnomalyScore)
			}
		})
	}
}

func TestBehaviorFlowService_ClusterFlowPoints(t *testing.T) {
	service := NewBehaviorFlowService()
	flow := &SpatioTemporalFlow{}

	dataPoints := []FlowDataPoint{
		{X: 0, Y: 0, Z: 0, Timestamp: time.Now()},
		{X: 10, Y: 10, Z: 0, Timestamp: time.Now()},
		{X: 20, Y: 20, Z: 0, Timestamp: time.Now()},
		{X: 100, Y: 100, Z: 0, Timestamp: time.Now()},
		{X: 110, Y: 110, Z: 0, Timestamp: time.Now()},
		{X: 200, Y: 200, Z: 0, Timestamp: time.Now()},
		{X: 210, Y: 210, Z: 0, Timestamp: time.Now()},
		{X: 300, Y: 300, Z: 0, Timestamp: time.Now()},
	}
	flow.DataPoints = dataPoints

	service.clusterFlowPoints(flow)

	if len(flow.Clusters) != len(dataPoints) {
		t.Errorf("Expected %d cluster assignments, got %d", len(dataPoints), len(flow.Clusters))
	}

	clusterCount := make(map[int]int)
	for _, c := range flow.Clusters {
		clusterCount[c]++
	}

	if len(clusterCount) < 2 {
		t.Error("Should identify at least 2 clusters")
	}
}

func TestBehaviorFlowService_ComputeContinuityScore(t *testing.T) {
	service := NewBehaviorFlowService()
	flow := &SpatioTemporalFlow{
		DataPoints: make([]FlowDataPoint, 5),
		Features: map[string]float64{
			"interval_variance":    0.2,
			"efficiency":          0.8,
			"velocity_variance":   0.3,
		},
	}

	service.computeContinuityScore(flow)

	if flow.Continuity < 0 || flow.Continuity > 1 {
		t.Errorf("Continuity should be between 0 and 1, got %.2f", flow.Continuity)
	}
}

func TestBehaviorFlowService_GetFlowHistory(t *testing.T) {
	service := NewBehaviorFlowService()

	dataPoints := []FlowDataPoint{
		{X: 0, Y: 0, Z: 0, Timestamp: time.Now()},
		{X: 10, Y: 10, Z: 0, Timestamp: time.Now().Add(100 * time.Millisecond)},
	}

	_, _ = service.AnalyzeFlow(context.Background(), "user123", "session1", dataPoints)
	_, _ = service.AnalyzeFlow(context.Background(), "user123", "session2", dataPoints)

	history, err := service.GetFlowHistory(context.Background(), "user123")
	if err != nil {
		t.Fatalf("GetFlowHistory failed: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 flows in history, got %d", len(history))
	}
}

func TestBehaviorFlowService_GenerateFlowReport(t *testing.T) {
	service := NewBehaviorFlowService()

	dataPoints := []FlowDataPoint{
		{X: 0, Y: 0, Z: 0, Timestamp: time.Now()},
		{X: 100, Y: 100, Z: 0, Timestamp: time.Now().Add(1 * time.Second)},
	}

	flow, _ := service.AnalyzeFlow(context.Background(), "user123", "session456", dataPoints)

	report := service.GenerateFlowReport(flow)

	if len(report) == 0 {
		t.Error("Report should not be empty")
	}

	if report == "" {
		t.Error("Report should be generated")
	}
}

func TestMeanFunction(t *testing.T) {
	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal values", []float64{1, 2, 3, 4, 5}, 3.0},
		{"single value", []float64{5}, 5.0},
		{"empty", []float64{}, 0.0},
		{"large values", []float64{100, 200, 300}, 200.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mean(tc.values)
			if result != tc.expected {
				t.Errorf("Expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestVarianceFunction(t *testing.T) {
	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"constant values", []float64{5, 5, 5, 5}, 0.0},
		{"varying values", []float64{1, 3, 5, 7, 9}, 10.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := variance(tc.values)
			if result < tc.expected*0.9 || result > tc.expected*1.1 {
				t.Errorf("Expected around %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestEntropyFunction(t *testing.T) {
	testCases := []struct {
		name        string
		values      []float64
		minExpected float64
		maxExpected float64
	}{
		{"uniform distribution", []float64{1, 1, 1, 1}, 0.8, 1.2},
		{"random distribution", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0.5, 1.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := entropy(tc.values)
			if result < tc.minExpected || result > tc.maxExpected {
				t.Errorf("Expected between %.2f and %.2f, got %.2f", tc.minExpected, tc.maxExpected, result)
			}
		})
	}
}

func BenchmarkBehaviorFlowService_AnalyzeFlow(b *testing.B) {
	service := NewBehaviorFlowService()

	dataPoints := make([]FlowDataPoint, 100)
	baseTime := time.Now()
	for i := 0; i < 100; i++ {
		dataPoints[i] = FlowDataPoint{
			X:         float64(i * 10),
			Y:         float64(i * 10),
			Z:         0,
			Timestamp: baseTime.Add(time.Duration(i*10) * time.Millisecond),
			Velocity:  50 + float64(i%10),
			EventType: "move",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.AnalyzeFlow(context.Background(), "bench_user", "bench_session", dataPoints)
	}
}
