package trace

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestExtractEnhancedFeatures(t *testing.T) {
	extractor := NewTraceExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move", Pressure: 0.5},
			{Timestamp: 1050, X: 20, Y: 20, Event: "move", Pressure: 0.6},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move", Pressure: 0.5},
			{Timestamp: 1150, X: 90, Y: 90, Event: "move", Pressure: 0.7},
			{Timestamp: 1200, X: 140, Y: 140, Event: "move", Pressure: 0.5},
			{Timestamp: 1300, X: 200, Y: 200, Event: "click", Pressure: 0.8},
			{Timestamp: 1500, X: 210, Y: 210, Event: "move", Pressure: 0.5},
		},
		TotalTime: 500,
		ClickData: []model.ClickInfo{
			{X: 140, Y: 140, Timestamp: 1200, Pressure: 0.8, ClickType: "left"},
			{X: 200, Y: 200, Timestamp: 1600, Pressure: 0.7, ClickType: "left"},
		},
		ScrollData: []model.ScrollInfo{
			{Timestamp: 1700, DeltaY: 100, Velocity: 200, Direction: "down"},
			{Timestamp: 1800, DeltaY: 100, Velocity: 200, Direction: "down"},
		},
	}

	features, err := extractor.ExtractEnhancedFeatures(traceData)
	if err != nil {
		t.Fatalf("ExtractEnhancedFeatures failed: %v", err)
	}

	if features == nil {
		t.Fatal("Enhanced features should not be nil")
	}

	if features.AvgPressure <= 0 {
		t.Errorf("Expected positive AvgPressure, got %f", features.AvgPressure)
	}

	if features.ClickCount != 2 {
		t.Errorf("Expected ClickCount 2, got %d", features.ClickCount)
	}

	if features.ScrollCount != 2 {
		t.Errorf("Expected ScrollCount 2, got %d", features.ScrollCount)
	}

	t.Logf("Enhanced Features: %+v", features)
}

func TestEnhancedFeaturesWithNoClickData(t *testing.T) {
	extractor := NewTraceExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move"},
			{Timestamp: 1200, X: 100, Y: 100, Event: "move"},
		},
		TotalTime: 200,
	}

	features, err := extractor.ExtractEnhancedFeatures(traceData)
	if err != nil {
		t.Fatalf("ExtractEnhancedFeatures failed: %v", err)
	}

	if features == nil {
		t.Fatal("Enhanced features should not be nil")
	}

	if features.ClickCount != 0 {
		t.Errorf("Expected ClickCount 0, got %d", features.ClickCount)
	}

	if features.ScrollCount != 0 {
		t.Errorf("Expected ScrollCount 0, got %d", features.ScrollCount)
	}

	t.Logf("Enhanced Features (no clicks/scrolls): %+v", features)
}

func TestCalculateClickRegularity(t *testing.T) {
	extractor := NewTraceExtractor()

	clicks := []model.ClickInfo{
		{Timestamp: 1000},
		{Timestamp: 1200},
		{Timestamp: 1400},
		{Timestamp: 1600},
	}

	regularity := extractor.calculateClickRegularity(clicks)

	if regularity < 0.9 || regularity > 1.0 {
		t.Errorf("Expected high regularity for evenly spaced clicks, got %f", regularity)
	}

	clicksIrregular := []model.ClickInfo{
		{Timestamp: 1000},
		{Timestamp: 1100},
		{Timestamp: 2000},
		{Timestamp: 2100},
	}

	irregularity := extractor.calculateClickRegularity(clicksIrregular)

	if irregularity > 0.5 {
		t.Errorf("Expected low regularity for irregular clicks, got %f", irregularity)
	}

	t.Logf("Regular click regularity: %f, Irregular click regularity: %f", regularity, irregularity)
}

func TestCalculateClickAreaSize(t *testing.T) {
	extractor := NewTraceExtractor()

	clicks := []model.ClickInfo{
		{X: 100, Y: 100},
		{X: 150, Y: 150},
		{X: 200, Y: 200},
	}

	areaSize := extractor.calculateClickAreaSize(clicks)

	expectedArea := (200.0 - 100.0) * (200.0 - 100.0) / 10000.0

	if areaSize != expectedArea {
		t.Errorf("Expected area size %f, got %f", expectedArea, areaSize)
	}

	t.Logf("Click area size: %f", areaSize)
}

func TestCalculateMovementFluidity(t *testing.T) {
	extractor := NewTraceExtractor()

	traceDataSmooth := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1050, X: 20, Y: 20, Event: "move"},
			{Timestamp: 1100, X: 40, Y: 40, Event: "move"},
			{Timestamp: 1150, X: 60, Y: 60, Event: "move"},
		},
		TotalTime: 150,
	}

	fluiditySmooth, err := extractor.ExtractEnhancedFeatures(traceDataSmooth)
	if err != nil {
		t.Fatalf("ExtractEnhancedFeatures failed: %v", err)
	}

	if fluiditySmooth.MovementFluidity < 0.5 {
		t.Errorf("Expected high fluidity for smooth movement, got %f", fluiditySmooth.MovementFluidity)
	}

	traceDataJerky := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move"},
			{Timestamp: 1100, X: 100, Y: 0, Event: "move"},
			{Timestamp: 1200, X: 0, Y: 100, Event: "move"},
			{Timestamp: 1300, X: 100, Y: 100, Event: "move"},
		},
		TotalTime: 300,
	}

	fluidityJerky, err := extractor.ExtractEnhancedFeatures(traceDataJerky)
	if err != nil {
		t.Fatalf("ExtractEnhancedFeatures failed: %v", err)
	}

	if fluidityJerky.MovementFluidity > fluiditySmooth.MovementFluidity {
		t.Errorf("Expected lower fluidity for jerky movement, got jerky: %f, smooth: %f",
			fluidityJerky.MovementFluidity, fluiditySmooth.MovementFluidity)
	}

	t.Logf("Smooth fluidity: %f, Jerky fluidity: %f", fluiditySmooth.MovementFluidity, fluidityJerky.MovementFluidity)
}

func TestPressureSequenceExtraction(t *testing.T) {
	extractor := NewTraceExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "move", Pressure: 0.3},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move", Pressure: 0.5},
			{Timestamp: 1200, X: 100, Y: 100, Event: "click", Pressure: 0.8},
			{Timestamp: 1300, X: 150, Y: 150, Event: "move"},
			{Timestamp: 1400, X: 200, Y: 200, Event: "move", Pressure: 0.4},
		},
		TotalTime: 400,
	}

	features, err := extractor.ExtractEnhancedFeatures(traceData)
	if err != nil {
		t.Fatalf("ExtractEnhancedFeatures failed: %v", err)
	}

	if features.AvgPressure <= 0 {
		t.Error("Expected positive average pressure")
	}

	if features.PressureVariance <= 0 {
		t.Error("Expected positive pressure variance")
	}

	t.Logf("Pressure stats - Mean: %f, Variance: %f, Max: %f, Min: %f",
		features.AvgPressure, features.PressureVariance, features.MaxPressure, features.MinPressure)
}

func TestScrollRegularityCalculation(t *testing.T) {
	extractor := NewTraceExtractor()

	scrollsRegular := []model.ScrollInfo{
		{Timestamp: 1000, DeltaY: 100, Velocity: 100},
		{Timestamp: 1100, DeltaY: 100, Velocity: 100},
		{Timestamp: 1200, DeltaY: 100, Velocity: 100},
		{Timestamp: 1300, DeltaY: 100, Velocity: 100},
	}

	regularity := extractor.calculateScrollRegularity(scrollsRegular)

	if regularity < 0.9 {
		t.Errorf("Expected high regularity for consistent scrolls, got %f", regularity)
	}

	scrollsIrregular := []model.ScrollInfo{
		{Timestamp: 1000, DeltaY: 100, Velocity: 50},
		{Timestamp: 1100, DeltaY: 200, Velocity: 200},
		{Timestamp: 1200, DeltaY: 50, Velocity: 50},
		{Timestamp: 1300, DeltaY: 300, Velocity: 300},
	}

	irregularity := extractor.calculateScrollRegularity(scrollsIrregular)

	if irregularity > 0.5 {
		t.Errorf("Expected low regularity for inconsistent scrolls, got %f", irregularity)
	}

	t.Logf("Regular scroll regularity: %f, Irregular scroll regularity: %f", regularity, irregularity)
}

func TestScrollDirectionEntropy(t *testing.T) {
	extractor := NewTraceExtractor()

	scrolls := []model.ScrollInfo{
		{Timestamp: 1000, Direction: "up"},
		{Timestamp: 1100, Direction: "down"},
		{Timestamp: 1200, Direction: "left"},
		{Timestamp: 1300, Direction: "right"},
		{Timestamp: 1400, Direction: "up"},
		{Timestamp: 1500, Direction: "down"},
		{Timestamp: 1600, Direction: "left"},
		{Timestamp: 1700, Direction: "right"},
	}

	entropy := extractor.calculateScrollDirectionEntropy(scrolls)

	if entropy < 1.5 {
		t.Errorf("Expected high entropy for diverse directions, got %f", entropy)
	}

	scrollsUniform := []model.ScrollInfo{
		{Timestamp: 1000, Direction: "down"},
		{Timestamp: 1100, Direction: "down"},
		{Timestamp: 1200, Direction: "down"},
	}

	uniformEntropy := extractor.calculateScrollDirectionEntropy(scrollsUniform)

	if uniformEntropy > 0.5 {
		t.Errorf("Expected low entropy for uniform direction, got %f", uniformEntropy)
	}

	t.Logf("Diverse entropy: %f, Uniform entropy: %f", entropy, uniformEntropy)
}
