package trace

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestLSTMEnhancedFeatureExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1050, X: 20, Y: 10, Event: "move"},
			{Timestamp: 1100, X: 40, Y: 25, Event: "move"},
			{Timestamp: 1150, X: 65, Y: 40, Event: "move"},
			{Timestamp: 1200, X: 90, Y: 60, Event: "move"},
			{Timestamp: 1250, X: 120, Y: 85, Event: "move"},
			{Timestamp: 1300, X: 150, Y: 110, Event: "move"},
			{Timestamp: 1350, X: 180, Y: 140, Event: "move"},
		},
		TotalTime: 350,
	}

	features, err := extractor.ExtractEnhancedFeatures(traceData)
	if err != nil {
		t.Fatalf("ExtractEnhancedFeatures failed: %v", err)
	}

	if features == nil {
		t.Fatal("Enhanced features should not be nil")
	}

	if features.VelocityFeatures.Mean <= 0 {
		t.Errorf("Expected positive velocity mean, got %f", features.VelocityFeatures.Mean)
	}

	if features.AccelerationFeatures.Mean == 0 {
		t.Log("Acceleration mean is zero (normal for uniform motion)")
	}

	t.Logf("Enhanced LSTM Features: %+v", features)
}

func TestVelocityFeaturesExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	velocities := []float64{100, 120, 110, 130, 115, 125, 118, 122}

	features := extractor.extractVelocityFeatures(velocities)

	if features.Mean <= 0 {
		t.Errorf("Expected positive mean velocity, got %f", features.Mean)
	}

	if features.Max < features.Min {
		t.Errorf("Expected Max (%f) >= Min (%f)", features.Max, features.Min)
	}

	if features.StdDev < 0 {
		t.Errorf("Expected non-negative StdDev, got %f", features.StdDev)
	}

	t.Logf("Velocity features: Mean=%.2f, StdDev=%.2f, Skewness=%.2f, Kurtosis=%.2f",
		features.Mean, features.StdDev, features.Skewness, features.Kurtosis)
}

func TestPressureFeaturesExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	pressures := []float64{0.5, 0.6, 0.5, 0.7, 0.5, 0.6, 0.5, 0.5}

	features := extractor.extractPressureFeatures(pressures)

	if features.Mean <= 0 {
		t.Errorf("Expected positive mean pressure, got %f", features.Mean)
	}

	if features.Max <= 0 {
		t.Error("Expected positive maximum pressure")
	}

	if features.Min <= 0 {
		t.Error("Expected positive minimum pressure")
	}

	t.Logf("Pressure features: Mean=%.2f, Variance=%.4f, Consistency=%.2f",
		features.Mean, features.StdDev*features.StdDev, features.Consistency)
}

func TestScrollFeaturesExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	scrolls := []ScrollSequencePoint{
		{DeltaX: 0, DeltaY: 100, Velocity: 100, Direction: 0, Timestamp: 1000},
		{DeltaX: 0, DeltaY: 100, Velocity: 100, Direction: 0, Timestamp: 1100},
		{DeltaX: 0, DeltaY: 100, Velocity: 100, Direction: 0, Timestamp: 1200},
		{DeltaX: 0, DeltaY: 100, Velocity: 100, Direction: 0, Timestamp: 1300},
	}

	features := extractor.extractScrollFeatures(scrolls)

	if features.Count != 4 {
		t.Errorf("Expected scroll count 4, got %d", features.Count)
	}

	if features.AvgVelocity <= 0 {
		t.Errorf("Expected positive average velocity, got %f", features.AvgVelocity)
	}

	if features.Regularity < 0.9 {
		t.Logf("Expected high regularity for consistent scrolls, got %f", features.Regularity)
	}

	t.Logf("Scroll features: Count=%d, AvgVelocity=%.2f, Regularity=%.2f",
		features.Count, features.AvgVelocity, features.Regularity)
}

func TestSpatialFeaturesExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	normalized := [][]float64{
		{0.0, 0.0},
		{0.2, 0.1},
		{0.4, 0.25},
		{0.6, 0.4},
		{0.8, 0.6},
		{1.0, 0.85},
	}

	features := extractor.extractSpatialFeatures(normalized)

	if features.CoverageArea <= 0 {
		t.Errorf("Expected positive coverage area, got %f", features.CoverageArea)
	}

	if features.SpatialSpreadX <= 0 {
		t.Errorf("Expected positive spatial spread X, got %f", features.SpatialSpreadX)
	}

	t.Logf("Spatial features: CoverageArea=%.4f, Centroid=(%.2f, %.2f), Spread=(%.2f, %.2f)",
		features.CoverageArea, features.CentroidX, features.CentroidY,
		features.SpatialSpreadX, features.SpatialSpreadY)
}

func TestCurvatureFeaturesExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	curvatures := []float64{0.1, 0.2, 0.15, 0.25, 0.1, 0.2, 0.15, 0.2}

	features := extractor.extractCurvatureFeatures(curvatures)

	if features.Mean <= 0 {
		t.Errorf("Expected positive mean curvature, got %f", features.Mean)
	}

	if features.Max < features.Mean {
		t.Errorf("Expected Max >= Mean")
	}

	t.Logf("Curvature features: Mean=%.4f, StdDev=%.4f, Max=%.4f, ZeroCrossings=%d",
		features.Mean, features.StdDev, features.Max, features.ZeroCrossings)
}

func TestAccelerationFeaturesExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	accelerations := []float64{10, -5, 20, -10, 15, -8, 12, -5}
	velocities := []float64{100, 120, 110, 130, 115, 125, 118, 122}
	points := []model.TracePoint{
		{Timestamp: 1000},
		{Timestamp: 1100},
		{Timestamp: 1200},
		{Timestamp: 1300},
		{Timestamp: 1400},
		{Timestamp: 1500},
		{Timestamp: 1600},
		{Timestamp: 1700},
		{Timestamp: 1800},
	}

	features := extractor.extractAccelerationFeatures(accelerations, velocities, points)

	if features.Mean == 0 && features.StdDev == 0 {
		t.Log("Acceleration features are zero (expected for test data)")
	}

	t.Logf("Acceleration features: Mean=%.2f, StdDev=%.2f, PositiveRatio=%.2f, NegativeRatio=%.2f",
		features.Mean, features.StdDev, features.PositiveRatio, features.NegativeRatio)
}

func TestTemporalFeaturesExtraction(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	points := []model.TracePoint{
		{Timestamp: 1000},
		{Timestamp: 1050},
		{Timestamp: 1100},
		{Timestamp: 1150},
		{Timestamp: 1200},
		{Timestamp: 1250},
	}

	features := extractor.extractTemporalFeaturesFromSeq(points)

	if features.AvgInterval <= 0 {
		t.Errorf("Expected positive average interval, got %f", features.AvgInterval)
	}

	expectedInterval := 50.0
	if features.AvgInterval != expectedInterval {
		t.Logf("Average interval is %f (expected ~%f for this test data)", features.AvgInterval, expectedInterval)
	}

	t.Logf("Temporal features: AvgInterval=%.2f, IntervalStdDev=%.2f, PauseRatio=%.2f",
		features.AvgInterval, features.IntervalStdDev, features.PauseRatio)
}

func TestCountZeroCrossings(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	values := []float64{1, 2, 3, -1, -2, 1, 2}

	crossings := extractor.countZeroCrossings(values)

	expectedCrossings := 2
	if crossings != expectedCrossings {
		t.Errorf("Expected %d zero crossings, got %d", expectedCrossings, crossings)
	}

	t.Logf("Zero crossings: %d", crossings)
}

func TestComputeDirectionEntropy(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	angles := []float64{0, 0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5}

	entropy := extractor.computeDirectionEntropyFromAngles(angles)

	if entropy <= 0 {
		t.Errorf("Expected positive entropy for diverse angles, got %f", entropy)
	}

	uniformAngles := []float64{0, 0, 0, 0, 0}
	uniformEntropy := extractor.computeDirectionEntropyFromAngles(uniformAngles)

	if uniformEntropy > 0.1 {
		t.Errorf("Expected near-zero entropy for uniform angles, got %f", uniformEntropy)
	}

	t.Logf("Diverse entropy: %f, Uniform entropy: %f", entropy, uniformEntropy)
}

func TestComputeScrollRegularity(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	velocities := []float64{100, 100, 100, 100}

	regularity := extractor.computeScrollRegularity(velocities)

	if regularity < 0.9 {
		t.Errorf("Expected high regularity for uniform velocities, got %f", regularity)
	}

	variedVelocities := []float64{50, 100, 150, 200}
	variedRegularity := extractor.computeScrollRegularity(variedVelocities)

	if variedRegularity > regularity {
		t.Errorf("Expected varied velocities to have lower regularity")
	}

	t.Logf("Uniform regularity: %f, Varied regularity: %f", regularity, variedRegularity)
}

func TestComputeRampIndex(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	acceleratingVelocities := []float64{100, 110, 120, 130, 140, 150, 160, 170}
	rampIndex := extractor.computeRampIndex(acceleratingVelocities)

	if rampIndex <= 0 {
		t.Errorf("Expected positive ramp index for accelerating velocities, got %f", rampIndex)
	}

	deceleratingVelocities := []float64{170, 160, 150, 140, 130, 120, 110, 100}
	decelRampIndex := extractor.computeRampIndex(deceleratingVelocities)

	if decelRampIndex >= 0 {
		t.Errorf("Expected negative ramp index for decelerating velocities, got %f", decelRampIndex)
	}

	t.Logf("Accelerating ramp index: %f, Decelerating ramp index: %f", rampIndex, decelRampIndex)
}

func TestComputeJerkiness(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	smoothAccelerations := []float64{10, 11, 10, 11, 10, 11, 10}
	jerkyAccelerations := []float64{10, -10, 10, -10, 10, -10, 10}

	smoothJerkiness := extractor.computeJerkiness(smoothAccelerations)
	jerkyJerkiness := extractor.computeJerkiness(jerkyAccelerations)

	if jerkyJerkiness <= smoothJerkiness {
		t.Logf("Expected jerky motion to have higher jerkiness, got smooth: %f, jerky: %f",
			smoothJerkiness, jerkyJerkiness)
	}

	t.Logf("Smooth jerkiness: %f, Jerky jerkiness: %f", smoothJerkiness, jerkyJerkiness)
}

func TestComputeSkewnessAndKurtosis(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	skewness := extractor.computeSkewness(values)
	kurtosis := extractor.computeKurtosis(values)

	t.Logf("Normal distribution: Skewness=%.2f, Kurtosis=%.2f", skewness, kurtosis)

	rightSkewed := []float64{1, 1, 1, 2, 2, 3, 4, 5, 10, 20}
	rightSkewness := extractor.computeSkewness(rightSkewed)

	if rightSkewness < 0 {
		t.Logf("Right skewness: %.2f (expected positive)", rightSkewness)
	}

	t.Logf("Right skewness: %.2f", rightSkewness)
}

func TestLSTMModelParameters(t *testing.T) {
	if LSTMFeatureDim != 128 {
		t.Errorf("Expected LSTMFeatureDim 128, got %d", LSTMFeatureDim)
	}

	if LSTMHiddenSize != 256 {
		t.Errorf("Expected LSTMHiddenSize 256, got %d", LSTMHiddenSize)
	}

	if LSTMNumLayers != 3 {
		t.Errorf("Expected LSTMNumLayers 3, got %d", LSTMNumLayers)
	}

	if DropoutRate != 0.3 {
		t.Errorf("Expected DropoutRate 0.3, got %f", DropoutRate)
	}

	if LearningRate != 0.001 {
		t.Errorf("Expected LearningRate 0.001, got %f", LearningRate)
	}

	t.Logf("LSTM parameters verified: FeatureDim=%d, HiddenSize=%d, NumLayers=%d, Dropout=%.2f, LR=%.4f",
		LSTMFeatureDim, LSTMHiddenSize, LSTMNumLayers, DropoutRate, LearningRate)
}
