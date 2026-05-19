package service

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestNewPerformancePredictor(t *testing.T) {
	predictor := NewPerformancePredictor()

	if predictor == nil {
		t.Fatal("NewPerformancePredictor returned nil")
	}

	if len(predictor.historicalData) == 0 {
		t.Error("No historical data initialized")
	}

	if len(predictor.models) == 0 {
		t.Error("No models initialized")
	}
}

func TestPerformancePredictor_Predict(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	metrics := OperationalMetrics{
		CPUUsage:          65,
		MemoryUsage:      70,
		DiskUsage:        55,
		NetworkLatency:   25,
		DBLatency:        15,
		CacheHitRate:     85,
		ErrorRate:        2,
		SuccessRate:      98,
		AvgResponseTime:  150,
		RequestThroughput: 1000,
	}

	predictions, err := predictor.Predict(ctx, metrics)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if len(predictions) == 0 {
		t.Error("Predict returned no predictions")
	}

	for _, pred := range predictions {
		if pred.Confidence < 0 || pred.Confidence > 1 {
			t.Errorf("Invalid confidence for %s: %f", pred.MetricName, pred.Confidence)
		}
	}
}

func TestPerformancePredictor_GetForecast(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	forecast, err := predictor.GetForecast(ctx, "cpu_usage", 24*time.Hour)
	if err != nil {
		t.Fatalf("GetForecast failed: %v", err)
	}

	if forecast == nil {
		t.Fatal("GetForecast returned nil")
	}

	if len(forecast.Predictions) == 0 {
		t.Error("GetForecast returned no predictions")
	}

	for _, point := range forecast.Predictions {
		if point.Timestamp.IsZero() {
			t.Error("Forecast point has zero timestamp")
		}
	}
}

func TestPerformancePredictor_GetCapacityPlan(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	plan, err := predictor.GetCapacityPlan(ctx, "cpu_usage")
	if err != nil {
		t.Fatalf("GetCapacityPlan failed: %v", err)
	}

	if plan == nil {
		t.Fatal("GetCapacityPlan returned nil")
	}

	if plan.CurrentUtilization < 0 || plan.CurrentUtilization > 100 {
		t.Errorf("Invalid current utilization: %f", plan.CurrentUtilization)
	}
}

func TestPerformancePredictor_GetOptimizationRecommendations(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	recommendations, err := predictor.GetOptimizationRecommendations(ctx)
	if err != nil {
		t.Fatalf("GetOptimizationRecommendations failed: %v", err)
	}

	if recommendations == nil {
		t.Error("GetOptimizationRecommendations returned nil")
	}
}

func TestPerformancePredictor_AddDataPoint(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	point := TimeSeriesPoint{
		Timestamp: time.Now(),
		Value:     75.5,
	}

	err := predictor.AddDataPoint(ctx, "cpu_usage", point)
	if err != nil {
		t.Errorf("AddDataPoint failed: %v", err)
	}

	predictor.mu.RLock()
	data := predictor.historicalData["cpu_usage"]
	predictor.mu.RUnlock()

	if len(data) == 0 {
		t.Error("Data point not added")
	}
}

func TestPerformancePredictor_GetAllMetrics(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	metrics, err := predictor.GetAllMetrics(ctx)
	if err != nil {
		t.Fatalf("GetAllMetrics failed: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("GetAllMetrics returned no metrics")
	}
}

func TestPerformancePredictor_GetModelInfo(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	model, err := predictor.GetModelInfo(ctx, "cpu_usage")
	if err != nil {
		t.Fatalf("GetModelInfo failed: %v", err)
	}

	if model == nil {
		t.Fatal("GetModelInfo returned nil")
	}

	if model.Accuracy < 0 || model.Accuracy > 1 {
		t.Errorf("Invalid model accuracy: %f", model.Accuracy)
	}

	_, err = predictor.GetModelInfo(ctx, "non_existent")
	if err == nil {
		t.Error("Expected error for non-existent model")
	}
}

func TestPerformancePredictor_RetrainModel(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	err := predictor.RetrainModel(ctx, "cpu_usage")
	if err != nil {
		t.Errorf("RetrainModel failed: %v", err)
	}

	err = predictor.RetrainModel(ctx, "non_existent")
	if err == nil {
		t.Error("Expected error for non-existent model")
	}
}

func TestPerformancePredictor_SetConfidenceThreshold(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	err := predictor.SetConfidenceThreshold(ctx, 0.8)
	if err != nil {
		t.Errorf("SetConfidenceThreshold failed: %v", err)
	}

	err = predictor.SetConfidenceThreshold(ctx, -0.5)
	if err == nil {
		t.Error("Expected error for negative threshold")
	}

	err = predictor.SetConfidenceThreshold(ctx, 1.5)
	if err == nil {
		t.Error("Expected error for threshold > 1")
	}
}

func TestPerformancePredictor_ExportForecasts(t *testing.T) {
	predictor := NewPerformancePredictor()
	ctx := context.Background()

	data, err := predictor.ExportForecasts(ctx, "json")
	if err != nil {
		t.Fatalf("ExportForecasts failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportForecasts returned empty data")
	}
}

func TestPredictMetric(t *testing.T) {
	predictor := NewPerformancePredictor()

	tests := []struct {
		name         string
		metricName   string
		currentValue float64
	}{
		{"CPU normal", "cpu_usage", 50},
		{"CPU high", "cpu_usage", 85},
		{"Memory normal", "memory_usage", 60},
		{"Memory critical", "memory_usage", 92},
		{"Error rate normal", "error_rate", 2},
		{"Error rate high", "error_rate", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prediction := predictor.predictMetric(tt.metricName, tt.currentValue)

			if prediction.MetricName != tt.metricName {
				t.Errorf("Metric name mismatch: got %s, want %s", prediction.MetricName, tt.metricName)
			}

			if prediction.CurrentValue != tt.currentValue {
				t.Errorf("Current value mismatch: got %f, want %f", prediction.CurrentValue, tt.currentValue)
			}

			if prediction.Confidence < 0 || prediction.Confidence > 1 {
				t.Errorf("Invalid confidence: %f", prediction.Confidence)
			}

			if prediction.TimeHorizon == "" {
				t.Error("Time horizon is empty")
			}
		})
	}
}

func TestCalculateTrend(t *testing.T) {
	predictor := NewPerformancePredictor()

	increasingData := make([]TimeSeriesPoint, 24)
	for i := 0; i < 24; i++ {
		increasingData[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(-time.Duration(24-i) * time.Hour),
			Value:     30 + float64(i)*2,
		}
	}

	trend := predictor.calculateTrend(increasingData)
	if trend != "increasing" {
		t.Errorf("Expected increasing trend, got %s", trend)
	}

	decreasingData := make([]TimeSeriesPoint, 24)
	for i := 0; i < 24; i++ {
		decreasingData[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(-time.Duration(24-i) * time.Hour),
			Value:     80 - float64(i)*2,
		}
	}

	trend = predictor.calculateTrend(decreasingData)
	if trend != "decreasing" {
		t.Errorf("Expected decreasing trend, got %s", trend)
	}

	stableData := make([]TimeSeriesPoint, 24)
	for i := 0; i < 24; i++ {
		stableData[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(-time.Duration(24-i) * time.Hour),
			Value:     50,
		}
	}

	trend = predictor.calculateTrend(stableData)
	if trend != "stable" {
		t.Errorf("Expected stable trend, got %s", trend)
	}
}

func TestCalculateAverage(t *testing.T) {
	predictor := NewPerformancePredictor()

	tests := []struct {
		name     string
		points   []TimeSeriesPoint
		expected float64
	}{
		{
			name:     "Normal",
			points:   []TimeSeriesPoint{{Value: 10}, {Value: 20}, {Value: 30}},
			expected: 20.0,
		},
		{
			name:     "Empty",
			points:   []TimeSeriesPoint{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predictor.calculateAverage(tt.points)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("calculateAverage = %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestDetectOutliers(t *testing.T) {
	predictor := NewPerformancePredictor()

	data := make([]TimeSeriesPoint, 20)
	for i := 0; i < 20; i++ {
		data[i] = TimeSeriesPoint{
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Value:     50,
		}
	}

	data[0].Value = 100

	outliers := predictor.detectOutliers(data)

	if len(outliers) == 0 {
		t.Error("Expected outliers")
	}
}

func TestPredictAnomalies(t *testing.T) {
	predictor := NewPerformancePredictor()

	forecastPoints := []ForecastPoint{
		{Timestamp: time.Now().Add(1 * time.Hour), Value: 50},
		{Timestamp: time.Now().Add(2 * time.Hour), Value: 100},
		{Timestamp: time.Now().Add(3 * time.Hour), Value: 90},
	}

	anomalies := predictor.predictAnomalies("cpu_usage", forecastPoints)

	if len(anomalies) == 0 {
		t.Error("Expected anomalies from sudden change")
	}
}
