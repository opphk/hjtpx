package service

import (
	"encoding/json"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSliderAnalyzer(t *testing.T) {
	analyzer := NewSliderAnalyzer()
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.model)
}

func TestNewSliderMLModel(t *testing.T) {
	model := NewSliderMLModel()
	assert.NotNil(t, model)
	assert.NotNil(t, model.weights)
	assert.False(t, model.isTrained)
	assert.Equal(t, -15.0, model.bias)
}

func TestNewSliderFeatureExtractor(t *testing.T) {
	extractor := NewSliderFeatureExtractor()
	assert.NotNil(t, extractor)
}

func TestAnalyzeSliderTrajectory_BasicValidation(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 205, Timestamp: 1050},
		{X: 120, Y: 210, Timestamp: 1100},
	}

	result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)
}

func TestAnalyzeSliderTrajectory_EmptyTrajectory(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	result, err := analyzer.AnalyzeSliderTrajectory([]SliderPoint{}, 500)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsBot)
	assert.Greater(t, result.Confidence, 0.5)
}

func TestAnalyzeSliderTrajectory_InsufficientPoints(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	result, err := analyzer.AnalyzeSliderTrajectory([]SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 205, Timestamp: 1050},
	}, 500)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAnalyzeTrajectoryBasic(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 210, Timestamp: 1100},
		{X: 300, Y: 220, Timestamp: 1200},
		{X: 400, Y: 230, Timestamp: 1300},
		{X: 500, Y: 240, Timestamp: 1400},
	}

	result := analyzer.analyzeTrajectoryBasic(trajectory, 500)
	assert.NotNil(t, result)
	assert.Equal(t, 5, len(result.Points))
	assert.Greater(t, result.TotalDistance, 0.0)
	assert.Greater(t, result.AverageSpeed, 0.0)
	assert.LessOrEqual(t, result.PathEfficiency, 1.0)
}

func TestAnalyzeTrajectoryBasic_SinglePoint(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	result := analyzer.analyzeTrajectoryBasic([]SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
	}, 500)
	assert.NotNil(t, result)
	assert.True(t, result.IsBot)
}

func TestAnalyzeTrajectoryBasic_StraightLine(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 400, Y: 200, Timestamp: 1300},
		{X: 500, Y: 200, Timestamp: 1400},
	}

	result := analyzer.analyzeTrajectoryBasic(trajectory, 500)
	assert.NotNil(t, result)
	assert.Greater(t, result.PathEfficiency, 0.9)
}

func TestIsTrajectoryBotLike(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	tests := []struct {
		name       string
		trajectory []SliderPoint
		expected   bool
	}{
		{
			name: "perfect straight line",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 200, Y: 200, Timestamp: 1100},
				{X: 300, Y: 200, Timestamp: 1200},
				{X: 400, Y: 200, Timestamp: 1300},
			},
			expected: true,
		},
		{
			name: "normal trajectory",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 150, Y: 210, Timestamp: 1100},
				{X: 200, Y: 205, Timestamp: 1200},
				{X: 300, Y: 220, Timestamp: 1400},
				{X: 400, Y: 210, Timestamp: 1600},
			},
			expected: false,
		},
		{
			name: "high speed",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 900, Y: 200, Timestamp: 1100},
				{X: 1700, Y: 200, Timestamp: 1200},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.analyzeTrajectoryBasic(tt.trajectory, 500)
			isBot := analyzer.isTrajectoryBotLike(tt.trajectory, result)
			assert.Equal(t, tt.expected, isBot)
		})
	}
}

func TestCalculateTrajectoryConfidence(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 300, Y: 220, Timestamp: 1400},
		{X: 400, Y: 210, Timestamp: 1600},
	}

	sliderTraj := &SliderTrajectory{
		Points:         trajectory,
		PathEfficiency: 0.85,
		AverageSpeed:   150,
		SpeedVariance:  0.2,
	}

	confidence := analyzer.calculateTrajectoryConfidence(trajectory, sliderTraj)
	assert.Greater(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 0.99)
}

func TestExtractSpeeds(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	speeds := analyzer.extractSpeeds(trajectory)
	assert.Equal(t, 2, len(speeds))
	for _, speed := range speeds {
		assert.Greater(t, speed, 0.0)
	}
}

func TestExtractSpeeds_EmptyTrajectory(t *testing.T) {
	analyzer := NewSliderAnalyzer()
	speeds := analyzer.extractSpeeds([]SliderPoint{})
	assert.Equal(t, 0, len(speeds))
}

func TestSliderFeatureExtractor_ExtractFeatures(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 300, Y: 220, Timestamp: 1400},
		{X: 400, Y: 210, Timestamp: 1600},
	}

	sliderTraj := &SliderTrajectory{
		TotalDistance: 300,
		AverageSpeed:  150,
		MaxSpeed:      200,
		MinSpeed:      100,
	}

	features := extractor.ExtractFeatures(trajectory, 500, sliderTraj)
	assert.NotNil(t, features)
	assert.Greater(t, features.TotalDistance, 0.0)
	assert.Greater(t, features.DirectDistance, 0.0)
}

func TestExtractSpeeds_FeatureExtractor(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	speeds := extractor.extractSpeeds(trajectory)
	assert.Equal(t, 2, len(speeds))
}

func TestExtractAccelerations(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 400, Y: 200, Timestamp: 1300},
	}

	accelerations := extractor.extractAccelerations(trajectory)
	assert.GreaterOrEqual(t, len(accelerations), 0)
}

func TestExtractCurvatures(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 250, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
		{X: 250, Y: 150, Timestamp: 1300},
	}

	curvatures := extractor.extractCurvatures(trajectory)
	assert.GreaterOrEqual(t, len(curvatures), 0)
}

func TestCountDirectionChanges(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 200, Y: 300, Timestamp: 1200},
		{X: 300, Y: 300, Timestamp: 1300},
	}

	changes := extractor.countDirectionChanges(trajectory)
	assert.GreaterOrEqual(t, changes, 0)
}

func TestCountMicroCorrections(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
	}

	corrections := extractor.countMicroCorrections(trajectory)
	assert.GreaterOrEqual(t, corrections, 0)
}

func TestCountBacktrack(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 250, Y: 200, Timestamp: 1300},
		{X: 350, Y: 200, Timestamp: 1400},
	}

	count, distance := extractor.countBacktrack(trajectory)
	assert.GreaterOrEqual(t, count, 0)
	assert.GreaterOrEqual(t, distance, 0.0)
}

func TestCountPauses(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 102, Y: 201, Timestamp: 1200},
		{X: 104, Y: 202, Timestamp: 1400},
		{X: 200, Y: 200, Timestamp: 1600},
	}

	pauses, duration := extractor.countPauses(trajectory)
	assert.GreaterOrEqual(t, pauses, 0)
	assert.GreaterOrEqual(t, duration, 0.0)
}

func TestCountHovers(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 101, Y: 201, Timestamp: 1060},
		{X: 102, Y: 200, Timestamp: 1120},
		{X: 200, Y: 200, Timestamp: 1300},
	}

	hovers, duration := extractor.countHovers(trajectory)
	assert.GreaterOrEqual(t, hovers, 0)
	assert.GreaterOrEqual(t, duration, 0.0)
}

func TestCalculateSpeedDistribution(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	speeds := []float64{100, 150, 200, 250, 300}
	distribution := extractor.calculateSpeedDistribution(speeds, 5)
	assert.Equal(t, 5, len(distribution))

	total := 0.0
	for _, v := range distribution {
		total += v
	}
	assert.InDelta(t, 1.0, total, 0.01)
}

func TestCalculateSpeedConsistency(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	tests := []struct {
		name   string
		speeds []float64
		minVal float64
		maxVal float64
	}{
		{
			name:   "consistent speeds",
			speeds: []float64{100, 101, 102, 100, 99},
			minVal: 0.5,
			maxVal: 1.0,
		},
		{
			name:   "variable speeds",
			speeds: []float64{50, 200, 100, 300, 150},
			minVal: 0.0,
			maxVal: 1.0,
		},
		{
			name:   "empty speeds",
			speeds: []float64{},
			minVal: 0.0,
			maxVal: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consistency := extractor.calculateSpeedConsistency(tt.speeds)
			assert.GreaterOrEqual(t, consistency, tt.minVal)
			assert.LessOrEqual(t, consistency, tt.maxVal)
		})
	}
}

func TestCalculateAccelerationProfile(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	accelerations := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
	profile := extractor.calculateAccelerationProfile(accelerations, 3)
	assert.Equal(t, 3, len(profile))
}

func TestCalculateAngleDistribution(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 300, Timestamp: 1200},
		{X: 400, Y: 400, Timestamp: 1300},
	}

	distribution := extractor.calculateAngleDistribution(trajectory, 8)
	assert.Equal(t, 8, len(distribution))
}

func TestCalculateJitterScore(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 210, Timestamp: 1100},
		{X: 120, Y: 205, Timestamp: 1200},
		{X: 130, Y: 215, Timestamp: 1300},
	}

	jitter := extractor.calculateJitterScore(trajectory)
	assert.GreaterOrEqual(t, jitter, 0.0)
}

func TestSliderSmoothTrajectory(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 210, Timestamp: 1100},
		{X: 120, Y: 220, Timestamp: 1200},
		{X: 130, Y: 230, Timestamp: 1300},
	}

	smoothed := extractor.smoothTrajectory(trajectory, 3)
	assert.Equal(t, len(trajectory), len(smoothed))

	for i, p := range smoothed {
		assert.Equal(t, trajectory[i].Timestamp, p.Timestamp)
	}
}

func TestCalculateSmoothnessScore(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	tests := []struct {
		name        string
		trajectory  []SliderPoint
		minExpected float64
		maxExpected float64
	}{
		{
			name: "smooth trajectory",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 110, Y: 210, Timestamp: 1100},
				{X: 120, Y: 220, Timestamp: 1200},
				{X: 130, Y: 230, Timestamp: 1300},
			},
			minExpected: 0.8,
			maxExpected: 1.0,
		},
		{
			name: "noisy trajectory",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 200, Y: 100, Timestamp: 1100},
				{X: 100, Y: 300, Timestamp: 1200},
				{X: 200, Y: 100, Timestamp: 1300},
			},
			minExpected: 0.0,
			maxExpected: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := extractor.calculateSmoothnessScore(tt.trajectory)
			assert.GreaterOrEqual(t, score, tt.minExpected)
			assert.LessOrEqual(t, score, tt.maxExpected)
		})
	}
}

func TestCalculateTrajectoryEntropy(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 200, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
		{X: 250, Y: 200, Timestamp: 1300},
		{X: 300, Y: 200, Timestamp: 1400},
	}

	entropy := extractor.calculateTrajectoryEntropy(trajectory)
	assert.GreaterOrEqual(t, entropy, 0.0)
}

func TestCalculateVelocityProfile(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
		{X: 300, Y: 210, Timestamp: 1400},
		{X: 350, Y: 220, Timestamp: 1500},
	}

	profile := extractor.calculateVelocityProfile(trajectory, 3)
	assert.Equal(t, 3, len(profile))
}

func TestExtractSpeedsFromSegment(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	segment := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 200, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
	}

	speeds := extractor.extractSpeedsFromSegment(segment)
	assert.Equal(t, 2, len(speeds))
}

func TestCalculateFourierFrequency(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
		{X: 300, Y: 210, Timestamp: 1400},
		{X: 350, Y: 220, Timestamp: 1500},
		{X: 400, Y: 215, Timestamp: 1600},
		{X: 450, Y: 225, Timestamp: 1700},
	}

	frequency := extractor.calculateFourierFrequency(trajectory)
	assert.GreaterOrEqual(t, frequency, 0.0)
}

func TestFFT(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	x := []float64{1.0, 2.0, 3.0, 4.0}
	result := extractor.fft(x)
	assert.Equal(t, 4, len(result))
}

func TestCalculateFourierEnergy(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 210, Timestamp: 1100},
		{X: 200, Y: 205, Timestamp: 1200},
		{X: 250, Y: 215, Timestamp: 1300},
		{X: 300, Y: 210, Timestamp: 1400},
		{X: 350, Y: 220, Timestamp: 1500},
		{X: 400, Y: 215, Timestamp: 1600},
		{X: 450, Y: 225, Timestamp: 1700},
	}

	energy := extractor.calculateFourierEnergy(trajectory)
	assert.GreaterOrEqual(t, energy, 0.0)
}

func TestCalculateFractalDimension(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 210, Timestamp: 1100},
		{X: 120, Y: 205, Timestamp: 1200},
		{X: 130, Y: 215, Timestamp: 1300},
		{X: 140, Y: 210, Timestamp: 1400},
		{X: 150, Y: 220, Timestamp: 1500},
		{X: 160, Y: 215, Timestamp: 1600},
		{X: 170, Y: 225, Timestamp: 1700},
		{X: 180, Y: 220, Timestamp: 1800},
		{X: 190, Y: 230, Timestamp: 1900},
	}

	dimension := extractor.calculateFractalDimension(trajectory)
	assert.Greater(t, dimension, 0.0)
	assert.LessOrEqual(t, dimension, 2.0)
}

func TestLinearRegression(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	x := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	y := []float64{2.0, 4.0, 6.0, 8.0, 10.0}

	slope := extractor.linearRegression(x, y)
	assert.InDelta(t, 2.0, slope, 0.1)
}

func TestLinearRegression_InsufficientData(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	x := []float64{1.0}
	y := []float64{2.0}

	slope := extractor.linearRegression(x, y)
	assert.Equal(t, 1.0, slope)
}

func TestCalculateHumanLikeness(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	tests := []struct {
		name     string
		features *SliderFeatures
		minVal   float64
		maxVal   float64
	}{
		{
			name: "human-like features",
			features: &SliderFeatures{
				PathEfficiency:   0.85,
				SpeedConsistency: 0.6,
				MicroCorrections: 5,
				PauseCount:       2,
				CurvatureAverage: 0.1,
				JitterScore:      0.1,
				SmoothnessScore:  0.7,
				BacktrackCount:   1,
			},
			minVal: 0.5,
			maxVal: 1.0,
		},
		{
			name: "bot-like features",
			features: &SliderFeatures{
				PathEfficiency:   0.99,
				SpeedConsistency: 0.99,
				MicroCorrections: 0,
				PauseCount:       0,
				CurvatureAverage: 0.001,
				JitterScore:      0.001,
				SmoothnessScore:  0.99,
				BacktrackCount:   0,
			},
			minVal: 0.0,
			maxVal: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			likeness := extractor.calculateHumanLikeness(tt.features)
			assert.GreaterOrEqual(t, likeness, tt.minVal)
			assert.LessOrEqual(t, likeness, tt.maxVal)
		})
	}
}

func TestSliderMLModel_Predict(t *testing.T) {
	model := NewSliderMLModel()

	features := &SliderFeatures{
		PathEfficiency:     0.99,
		SpeedConsistency:   0.99,
		AverageSpeed:       2000,
		MaxSpeed:           3000,
		SpeedVariance:      0.001,
		CurvatureAverage:   0.001,
		DirectionChanges:   0,
		MicroCorrections:   0,
		PauseCount:         0,
		HumanLikenessScore: 0.1,
	}

	score := model.Predict(features)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestSliderMLModel_PredictNil(t *testing.T) {
	model := NewSliderMLModel()
	score := model.Predict(nil)
	assert.Equal(t, 0.5, score)
}

func TestSliderMLModel_ExtractFeatureVector(t *testing.T) {
	model := NewSliderMLModel()

	features := &SliderFeatures{
		PathEfficiency:       0.85,
		SpeedConsistency:     0.6,
		AverageSpeed:         500,
		MaxSpeed:             800,
		SpeedVariance:        0.1,
		CurvatureAverage:     0.1,
		CurvatureVariance:    0.05,
		DirectionChanges:     5,
		MicroCorrections:     3,
		BacktrackCount:       1,
		BacktrackDistance:    20,
		PauseCount:           2,
		TotalPauseDuration:   200,
		HoverCount:           1,
		HoverDurationTotal:   100,
		StartDelay:           500,
		ResponseTime:         2000,
		JitterScore:          0.1,
		SmoothnessScore:      0.7,
		TrajectoryEntropy:    3.0,
		FourierFrequency:     2.0,
		FourierEnergy:        1000,
		FractalDimension:     1.5,
		HumanLikenessScore:   0.6,
		AverageAcceleration:  0.1,
		AccelerationVariance: 0.05,
		CurvatureMax:         0.3,
		MinSpeed:             100,
		DirectDistance:       300,
		TotalDistance:        400,
	}

	vector := model.extractFeatureVector(features)
	assert.Equal(t, 30, len(vector))

	for _, v := range vector {
		assert.GreaterOrEqual(t, v, 0.0)
		assert.LessOrEqual(t, v, 1.0)
	}
}

func TestSliderMLModel_SimplePredict(t *testing.T) {
	model := NewSliderMLModel()

	tests := []struct {
		name     string
		features *SliderFeatures
		minScore float64
		maxScore float64
	}{
		{
			name: "bot-like",
			features: &SliderFeatures{
				PathEfficiency:     0.99,
				SpeedConsistency:   0.99,
				AverageSpeed:       2000,
				MicroCorrections:   0,
				PauseCount:         0,
				TotalDuration:      2000,
				HumanLikenessScore: 0.1,
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "human-like",
			features: &SliderFeatures{
				PathEfficiency:     0.85,
				SpeedConsistency:   0.6,
				AverageSpeed:       500,
				MicroCorrections:   5,
				PauseCount:         2,
				TotalDuration:      3000,
				HumanLikenessScore: 0.7,
			},
			minScore: 0.0,
			maxScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := model.simplePredict(tt.features)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSliderDetectAnomalies(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	tests := []struct {
		name     string
		features *SliderFeatures
		minScore float64
		maxScore float64
	}{
		{
			name: "anomalous features",
			features: &SliderFeatures{
				PathEfficiency:     0.99,
				SpeedConsistency:   0.99,
				AverageSpeed:       2500,
				MicroCorrections:   0,
				PauseCount:         0,
				TotalDuration:      3000,
				BacktrackCount:     0,
				CurvatureVariance:  0.0001,
				SmoothnessScore:    0.99,
				HumanLikenessScore: 0.1,
				FractalDimension:   1.05,
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "normal features",
			features: &SliderFeatures{
				PathEfficiency:     0.85,
				SpeedConsistency:   0.6,
				AverageSpeed:       500,
				MicroCorrections:   5,
				PauseCount:         2,
				TotalDuration:      3000,
				BacktrackCount:     1,
				CurvatureVariance:  0.1,
				SmoothnessScore:    0.7,
				HumanLikenessScore: 0.7,
				FractalDimension:   1.4,
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.detectAnomalies(tt.features)
			t.Logf("Test %s: anomaly score = %f", tt.name, score)
			assert.GreaterOrEqual(t, score, 0.0)
			assert.LessOrEqual(t, score, 1.0)
		})
	}
}

func TestClassifyTrajectoryPattern(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	tests := []struct {
		name     string
		features *SliderFeatures
		expected string
	}{
		{
			name: "perfect straight",
			features: &SliderFeatures{
				PathEfficiency: 0.99,
			},
			expected: "perfect_straight",
		},
		{
			name: "near straight",
			features: &SliderFeatures{
				PathEfficiency: 0.92,
			},
			expected: "near_straight",
		},
		{
			name: "erratic",
			features: &SliderFeatures{
				PathEfficiency: 0.7,
				BacktrackCount: 5,
			},
			expected: "erratic",
		},
		{
			name: "hesitant",
			features: &SliderFeatures{
				PathEfficiency: 0.8,
				PauseCount:     8,
			},
			expected: "hesitant",
		},
		{
			name: "curved",
			features: &SliderFeatures{
				PathEfficiency:   0.8,
				BacktrackCount:   2,
				PauseCount:       3,
				DirectionChanges: 15,
			},
			expected: "curved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := analyzer.classifyTrajectoryPattern(tt.features)
			assert.Equal(t, tt.expected, pattern)
		})
	}
}

func TestClassifySpeedProfile(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	tests := []struct {
		name     string
		features *SliderFeatures
		expected string
	}{
		{
			name: "extremely fast",
			features: &SliderFeatures{
				AverageSpeed: 2500,
			},
			expected: "extremely_fast",
		},
		{
			name: "very fast",
			features: &SliderFeatures{
				AverageSpeed: 1700,
			},
			expected: "very_fast",
		},
		{
			name: "fast",
			features: &SliderFeatures{
				AverageSpeed: 900,
			},
			expected: "fast",
		},
		{
			name: "normal",
			features: &SliderFeatures{
				AverageSpeed: 500,
			},
			expected: "normal",
		},
		{
			name: "slow",
			features: &SliderFeatures{
				AverageSpeed: 200,
			},
			expected: "slow",
		},
		{
			name: "very slow",
			features: &SliderFeatures{
				AverageSpeed: 50,
			},
			expected: "very_slow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := analyzer.classifySpeedProfile(tt.features)
			assert.Equal(t, tt.expected, profile)
		})
	}
}

func TestSliderCalculateRiskScore(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	result := &SliderAnalysisResult{
		AnomalyScore: 0.7,
		MLScore:      0.8,
		Trajectory: &SliderTrajectory{
			PathEfficiency: 0.99,
			AverageSpeed:   2000,
		},
		Features: &SliderFeatures{
			SpeedConsistency:   0.99,
			MicroCorrections:   0,
			PauseCount:         0,
			TotalDuration:      3000,
			HumanLikenessScore: 0.1,
		},
	}

	score := analyzer.calculateOverallRiskScore(result)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestSliderConfidence(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	result := &SliderAnalysisResult{
		Trajectory: &SliderTrajectory{
			Points: make([]SliderPoint, 40),
		},
		Features: &SliderFeatures{
			TotalDuration: 3000,
		},
		AnomalyScore: 0.5,
		MLScore:      0.6,
	}

	confidence := analyzer.calculateConfidence(result)
	assert.Greater(t, confidence, 0.7)
	assert.LessOrEqual(t, confidence, 0.99)
}

func TestMeanVarianceMaxMin(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	assert.InDelta(t, 3.0, analyzer.mean(values), 0.01)
	assert.InDelta(t, 2.0, analyzer.variance(values), 0.1)
	assert.InDelta(t, 5.0, analyzer.max(values), 0.01)
	assert.InDelta(t, 1.0, analyzer.min(values), 0.01)
}

func TestMeanVarianceMaxMin_Empty(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	assert.Equal(t, 0.0, analyzer.mean([]float64{}))
	assert.Equal(t, 0.0, analyzer.variance([]float64{}))
	assert.Equal(t, 0.0, analyzer.max([]float64{}))
	assert.Equal(t, 0.0, analyzer.min([]float64{}))
}

func TestSliderTrajectoryValidator(t *testing.T) {
	validator := NewSliderTrajectoryValidator()
	assert.NotNil(t, validator)
	assert.Equal(t, 10, validator.minPoints)
}

func TestSliderTrajectoryValidator_Validate(t *testing.T) {
	validator := NewSliderTrajectoryValidator()

	tests := []struct {
		name        string
		trajectory  []SliderPoint
		expectValid bool
	}{
		{
			name: "valid trajectory",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 200, Y: 200, Timestamp: 1100},
				{X: 300, Y: 200, Timestamp: 1200},
				{X: 400, Y: 200, Timestamp: 1300},
				{X: 500, Y: 200, Timestamp: 1400},
				{X: 600, Y: 200, Timestamp: 1500},
				{X: 700, Y: 200, Timestamp: 1600},
				{X: 800, Y: 200, Timestamp: 1700},
				{X: 900, Y: 200, Timestamp: 1800},
				{X: 1000, Y: 200, Timestamp: 1900},
			},
			expectValid: true,
		},
		{
			name: "insufficient points",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 200, Y: 200, Timestamp: 1100},
				{X: 300, Y: 200, Timestamp: 1200},
			},
			expectValid: false,
		},
		{
			name: "too short duration",
			trajectory: []SliderPoint{
				{X: 100, Y: 200, Timestamp: 1000},
				{X: 200, Y: 200, Timestamp: 1020},
				{X: 300, Y: 200, Timestamp: 1040},
				{X: 400, Y: 200, Timestamp: 1060},
				{X: 500, Y: 200, Timestamp: 1080},
				{X: 600, Y: 200, Timestamp: 1100},
				{X: 700, Y: 200, Timestamp: 1120},
				{X: 800, Y: 200, Timestamp: 1140},
				{X: 900, Y: 200, Timestamp: 1160},
				{X: 1000, Y: 200, Timestamp: 1170},
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := validator.Validate(tt.trajectory)
			assert.Equal(t, tt.expectValid, valid)
		})
	}
}

func TestSliderGenerateReport(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	result := &SliderAnalysisResult{
		Trajectory: &SliderTrajectory{
			TotalDistance:  500,
			PathEfficiency: 0.85,
			AverageSpeed:   200,
			MaxSpeed:       350,
			MinSpeed:       100,
			Points:         []SliderPoint{{X: 100, Y: 200, Timestamp: 1000}},
		},
		Features: &SliderFeatures{
			DirectDistance:     450,
			SpeedVariance:      0.05,
			SpeedConsistency:   0.7,
			DirectionChanges:   5,
			MicroCorrections:   3,
			BacktrackCount:     1,
			BacktrackDistance:  20,
			PauseCount:         2,
			TotalPauseDuration: 200,
			HoverCount:         1,
			CurvatureAverage:   0.05,
			CurvatureVariance:  0.02,
			CurvatureMax:       0.2,
			JitterScore:        0.1,
			SmoothnessScore:    0.7,
			TrajectoryEntropy:  3.5,
			FourierFrequency:   1.5,
			FourierEnergy:      500,
			FractalDimension:   1.3,
			HumanLikenessScore: 0.65,
			TotalDuration:      2500,
		},
		AnomalyScore:      0.3,
		MLScore:           0.4,
		OverallRiskScore:  0.35,
		IsBot:             false,
		Confidence:        0.8,
		TrajectoryPattern: "normal",
		SpeedProfile:      "normal",
		RiskIndicators:    []string{"test indicator"},
		AnomalyDetections: []string{"test detection"},
	}

	report := analyzer.GenerateReport(result)
	assert.NotEmpty(t, report)
	assert.Contains(t, report, "=== 滑块轨迹分析报告 ===")
	assert.Contains(t, report, "总距离")
	assert.Contains(t, report, "风险评估")
}

func TestParseSliderTrajectory(t *testing.T) {
	data := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
	}

	jsonData, _ := json.Marshal(data)
	points, err := ParseSliderTrajectory(jsonData)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(points))
}

func TestParseSliderTrajectory_InvalidJSON(t *testing.T) {
	data := []byte("invalid json")
	points, err := ParseSliderTrajectory(data)

	assert.Error(t, err)
	assert.Nil(t, points)
}

func TestGenerateHumanLikeSliderTrajectory(t *testing.T) {
	trajectory := GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000)

	assert.Greater(t, len(trajectory), 0)
	assert.InDelta(t, 100, float64(trajectory[0].X), 10)
	assert.Greater(t, trajectory[len(trajectory)-1].X, 400)
}

func TestGenerateBotLikeSliderTrajectory(t *testing.T) {
	trajectory := GenerateBotLikeSliderTrajectory(100, 200, 500, 200, 1000)

	assert.Greater(t, len(trajectory), 0)

	for i := 1; i < len(trajectory); i++ {
		assert.Greater(t, trajectory[i].X, trajectory[i-1].X)
	}
}

func TestHumanVsBotSliderDetection(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	humanTrajectory := GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000)
	humanResult, _ := analyzer.AnalyzeSliderTrajectory(humanTrajectory, 500)

	t.Logf("人类轨迹风险分数: %.4f", humanResult.OverallRiskScore)
	t.Logf("人类轨迹判定为机器人: %v", humanResult.IsBot)

	botTrajectory := GenerateBotLikeSliderTrajectory(100, 200, 500, 200, 1000)
	botResult, _ := analyzer.AnalyzeSliderTrajectory(botTrajectory, 500)

	t.Logf("机器人轨迹风险分数: %.4f", botResult.OverallRiskScore)
	t.Logf("机器人轨迹判定为机器人: %v", botResult.IsBot)

	assert.Less(t, humanResult.OverallRiskScore, botResult.OverallRiskScore,
		"人类轨迹风险分数应低于机器人轨迹")
}

func TestSliderAnalysisIntegration(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectories := []struct {
		name        string
		trajectory  []SliderPoint
		expectedBot bool
	}{
		{
			name:        "straight fast line",
			trajectory:  generateStraightFastLine(),
			expectedBot: true,
		},
		{
			name:        "normal human-like",
			trajectory:  GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000),
			expectedBot: false,
		},
	}

	for _, tt := range trajectories {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.AnalyzeSliderTrajectory(tt.trajectory, 500)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.expectedBot {
				assert.GreaterOrEqual(t, result.OverallRiskScore, 0.5)
			}
		})
	}
}

func generateStraightFastLine() []SliderPoint {
	points := make([]SliderPoint, 0)
	for i := 0; i < 30; i++ {
		points = append(points, SliderPoint{
			X:         100 + i*20,
			Y:         200,
			Timestamp: int64(1000 + i*20),
		})
	}
	return points
}

func TestSliderAnalyzer_HighAccuracy(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	successCount := 0
	totalTests := 100

	for i := 0; i < totalTests; i++ {
		isHuman := i%2 == 0
		var trajectory []SliderPoint

		if isHuman {
			trajectory = GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000)
		} else {
			trajectory = GenerateBotLikeSliderTrajectory(100, 200, 500, 200, 1000)
		}

		result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
		assert.NoError(t, err)

		correctlyIdentified := (isHuman && !result.IsBot) || (!isHuman && result.IsBot)
		if correctlyIdentified {
			successCount++
		}
	}

	accuracy := float64(successCount) / float64(totalTests)
	t.Logf("滑块验证准确率: %.2f%% (%d/%d)", accuracy*100, successCount, totalTests)

	assert.Greater(t, accuracy, 0.80, "准确率应大于80%")
}

func BenchmarkSliderAnalysis(b *testing.B) {
	analyzer := NewSliderAnalyzer()
	trajectory := GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.AnalyzeSliderTrajectory(trajectory, 500)
	}
}

func TestSliderAnalyzerWithEdgeCases(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("nil trajectory", func(t *testing.T) {
		result, err := analyzer.AnalyzeSliderTrajectory(nil, 500)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("very long trajectory", func(t *testing.T) {
		trajectory := make([]SliderPoint, 1000)
		for i := range trajectory {
			trajectory[i] = SliderPoint{
				X:         100 + i%100,
				Y:         200 + i%50,
				Timestamp: int64(1000 + i*10),
			}
		}
		result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("zero timestamp", func(t *testing.T) {
		trajectory := []SliderPoint{
			{X: 100, Y: 200, Timestamp: 0},
			{X: 200, Y: 200, Timestamp: 100},
			{X: 300, Y: 200, Timestamp: 200},
		}
		result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestSliderAnalysisFullPipeline(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	validator := NewSliderTrajectoryValidator()

	humanTrajectory := GenerateHumanLikeSliderTrajectory(100, 200, 500, 200, 3000)

	valid, msg := validator.Validate(humanTrajectory)
	assert.True(t, valid, msg)

	result, err := analyzer.AnalyzeSliderTrajectory(humanTrajectory, 500)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.NotNil(t, result.Trajectory)
	assert.NotNil(t, result.Features)
	assert.Greater(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)

	report := analyzer.GenerateReport(result)
	assert.NotEmpty(t, report)
}
