package service

import (
	"math"
	"testing"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestBehaviorAnalysisService_NewService(t *testing.T) {
	service := NewBehaviorAnalysisService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.storedPaths)
	assert.Equal(t, 0, len(service.storedPaths))
}

func TestBehaviorAnalysisService_AnalyzeBehavior_HumanLike(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			DataType: "mouse",
			Data:     `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 150, "y": 120, "timestamp": 1150, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 200, "y": 150, "timestamp": 1300, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 250, "y": 180, "timestamp": 1500, "event": "click"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 300, "y": 220, "timestamp": 1700, "event": "move"}`,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.GreaterOrEqual(t, result.Trajectory.TotalDistance, float64(0))
	assert.GreaterOrEqual(t, result.Trajectory.AverageSpeed, float64(0))
	assert.GreaterOrEqual(t, result.Trajectory.DirectionChanges, 0)
	assert.GreaterOrEqual(t, result.Trajectory.PauseCount, 0)

	assert.GreaterOrEqual(t, result.ClickPattern.ClickCount, 0)
	assert.GreaterOrEqual(t, result.ClickPattern.AverageInterval, float64(0))

	assert.GreaterOrEqual(t, result.RiskScore, float64(0))
	assert.LessOrEqual(t, result.RiskScore, float64(100))
	assert.NotNil(t, result.RiskFactors)
}

func TestBehaviorAnalysisService_AnalyzeBehavior_BotLike(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			DataType: "mouse",
			Data:     `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 200, "y": 200, "timestamp": 1050, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 300, "y": 300, "timestamp": 1100, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 400, "y": 400, "timestamp": 1150, "event": "click"}`,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	t.Logf("Bot-like risk score: %.2f", result.RiskScore)
	t.Logf("Risk indicators: %v", result.RiskIndicators)
	t.Logf("Is bot likely: %v", result.IsBotLikely)
}

func TestBehaviorAnalysisService_AnalyzeBehavior_EmptyData(t *testing.T) {
	service := NewBehaviorAnalysisService()

	result, err := service.AnalyzeBehavior([]models.BehaviorData{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, float64(0), result.Trajectory.TotalDistance)
	assert.Equal(t, float64(0), result.Trajectory.AverageSpeed)
	assert.Equal(t, 0, result.ClickPattern.ClickCount)
}

func TestBehaviorAnalysisService_AnalyzeBehavior_KeyboardData(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			DataType: "keyboard",
			Data:     `{"key": "h", "timestamp": 1000, "key_down_time": 1000, "key_up_time": 1050}`,
		},
		{
			DataType: "keyboard",
			Data:     `{"key": "e", "timestamp": 1100, "key_down_time": 1100, "key_up_time": 1150}`,
		},
		{
			DataType: "keyboard",
			Data:     `{"key": "l", "timestamp": 1200, "key_down_time": 1200, "key_up_time": 1250}`,
		},
		{
			DataType: "keyboard",
			Data:     `{"key": "l", "timestamp": 1300, "key_down_time": 1300, "key_up_time": 1350}`,
		},
		{
			DataType: "keyboard",
			Data:     `{"key": "o", "timestamp": 1400, "key_down_time": 1400, "key_up_time": 1450}`,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.GreaterOrEqual(t, result.KeyboardPattern.KeystrokeCount, 0)
	assert.GreaterOrEqual(t, result.KeyboardPattern.TypingSpeed, float64(0))
	assert.GreaterOrEqual(t, result.KeyboardPattern.AverageHoldTime, float64(0))
}

func TestBehaviorAnalysisService_AnalyzeBehavior_MixedData(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			DataType: "mouse",
			Data:     `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 150, "y": 150, "timestamp": 1200, "event": "click"}`,
		},
		{
			DataType: "keyboard",
			Data:     `{"key": "a", "timestamp": 1500, "key_down_time": 1500, "key_up_time": 1550}`,
		},
		{
			DataType: "keyboard",
			Data:     `{"key": "b", "timestamp": 1600, "key_down_time": 1600, "key_up_time": 1650}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 200, "y": 200, "timestamp": 2000, "event": "move"}`,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.Greater(t, result.Trajectory.TotalDistance, float64(0))
	assert.Greater(t, result.ClickPattern.ClickCount, 0)
	assert.Greater(t, result.KeyboardPattern.KeystrokeCount, 0)
}

func TestBehaviorAnalysisService_smoothTrajectory(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "move"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "move"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "move"},
		{X: 140, Y: 140, Timestamp: 1400, Event: "move"},
	}

	smoothed := service.smoothTrajectory(points, 3)
	assert.Equal(t, len(points), len(smoothed))

	for i, p := range smoothed {
		assert.Equal(t, p.Timestamp, points[i].Timestamp)
	}
}

func TestBehaviorAnalysisService_smoothTrajectory_ShortInput(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "move"},
	}

	smoothed := service.smoothTrajectory(points, 5)
	assert.Equal(t, points, smoothed)
}

func TestBehaviorAnalysisService_AdaptiveSmoothTrajectory(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 110, Y: 110, Timestamp: 1150, Event: "move"},
		{X: 120, Y: 120, Timestamp: 1300, Event: "move"},
		{X: 130, Y: 130, Timestamp: 1450, Event: "move"},
		{X: 140, Y: 140, Timestamp: 1600, Event: "move"},
	}

	smoothed := service.AdaptiveSmoothTrajectory(points)
	assert.Equal(t, len(points), len(smoothed))
}

func TestBehaviorAnalysisService_DetermineOptimalWindowSize(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name   string
		speeds []float64
		minWin int
		maxWin int
	}{
		{"very low variance", []float64{1.0, 1.1, 0.9, 1.0, 1.05}, 3, 7},
		{"high variance", []float64{0.5, 2.0, 0.8, 1.5, 0.6}, 3, 5},
		{"very high variance", []float64{0.1, 5.0, 0.2, 3.0, 0.5}, 3, 3},
		{"empty speeds", []float64{}, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			windowSize := service.DetermineOptimalWindowSize(tt.speeds)
			assert.GreaterOrEqual(t, windowSize, tt.minWin)
			assert.LessOrEqual(t, windowSize, tt.maxWin)
		})
	}
}

func TestBehaviorAnalysisService_AnalyzeSpeedAdvanced(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 110, Y: 105, Timestamp: 1100, Event: "move"},
		{X: 125, Y: 115, Timestamp: 1200, Event: "move"},
		{X: 145, Y: 130, Timestamp: 1300, Event: "move"},
		{X: 170, Y: 150, Timestamp: 1400, Event: "move"},
	}

	analysis := service.AnalyzeSpeedAdvanced(points)

	assert.NotNil(t, analysis.Speeds)
	assert.Greater(t, analysis.AverageSpeed, float64(0))
	assert.Greater(t, analysis.SpeedEntropy, float64(0))
	assert.GreaterOrEqual(t, analysis.SpeedBurstiness, float64(0))
}

func TestBehaviorAnalysisService_CalculateSpeedEntropy(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name   string
		speeds []float64
		minEnt float64
		maxEnt float64
	}{
		{"uniform speeds", []float64{1.0, 1.0, 1.0, 1.0}, 0.0, 0.1},
		{"varied speeds", []float64{0.5, 1.0, 1.5, 2.0, 2.5}, 1.0, 2.5},
		{"empty speeds", []float64{}, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := service.CalculateSpeedEntropy(tt.speeds)
			assert.GreaterOrEqual(t, entropy, tt.minEnt)
			assert.LessOrEqual(t, entropy, tt.maxEnt)
		})
	}
}

func TestBehaviorAnalysisService_CalculateSpeedBurstiness(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		speeds   []float64
		minBurst float64
		maxBurst float64
	}{
		{"stable speeds", []float64{1.0, 1.1, 0.9, 1.0, 1.05}, 0.0, 0.5},
		{"bursty speeds", []float64{0.5, 2.0, 0.5, 2.0, 0.5}, 0.5, 5.0},
		{"empty speeds", []float64{}, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			burstiness := service.CalculateSpeedBurstiness(tt.speeds)
			assert.GreaterOrEqual(t, burstiness, tt.minBurst)
			assert.LessOrEqual(t, burstiness, tt.maxBurst)
		})
	}
}

func TestBehaviorAnalysisService_computePathHash(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "move"},
	}

	hash := service.computePathHash(points)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, "|")
}

func TestBehaviorAnalysisService_computeDTWDistance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "move"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 160, Y: 160, Timestamp: 1100, Event: "move"},
		{X: 210, Y: 210, Timestamp: 1200, Event: "move"},
	}

	dtwDist := service.computeDTWDistance(path1, path2)
	assert.Greater(t, dtwDist, float64(0))
}

func TestBehaviorAnalysisService_computeDTWDistance_EmptyPaths(t *testing.T) {
	service := NewBehaviorAnalysisService()

	dtwDist1 := service.computeDTWDistance([]BehaviorDataPoint{}, []BehaviorDataPoint{})
	assert.Equal(t, float64(0), dtwDist1)

	dtwDist2 := service.computeDTWDistance([]BehaviorDataPoint{{X: 100, Y: 100}}, []BehaviorDataPoint{})
	assert.Equal(t, math.MaxFloat64, dtwDist2)
}

func TestBehaviorAnalysisService_computeFrechetDistance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "move"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 160, Y: 160, Timestamp: 1100, Event: "move"},
		{X: 210, Y: 210, Timestamp: 1200, Event: "move"},
	}

	frechetDist := service.computeFrechetDistance(path1, path2)
	assert.Greater(t, frechetDist, float64(0))
}

func TestBehaviorAnalysisService_computePathCorrelation(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "move"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 160, Y: 160, Timestamp: 1100, Event: "move"},
		{X: 210, Y: 210, Timestamp: 1200, Event: "move"},
	}

	correlation := service.computePathCorrelation(path1, path2)
	assert.Greater(t, correlation, float64(0))
	assert.LessOrEqual(t, correlation, float64(1))
}

func TestBehaviorAnalysisService_computePathCorrelation_DifferentLengths(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 160, Y: 160, Timestamp: 1100, Event: "move"},
		{X: 210, Y: 210, Timestamp: 1200, Event: "move"},
	}

	correlation := service.computePathCorrelation(path1, path2)
	assert.Equal(t, float64(0), correlation)
}

func TestBehaviorAnalysisService_pointDistance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	p1 := BehaviorDataPoint{X: 0, Y: 0}
	p2 := BehaviorDataPoint{X: 3, Y: 4}

	dist := service.pointDistance(p1, p2)
	assert.InDelta(t, 5.0, dist, 0.001)
}

func TestBehaviorAnalysisService_ComputeCurvatureStatistics(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 130, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 160, Timestamp: 1300, Event: "move"},
	}

	mean, stdDev, maxCurv := service.ComputeCurvatureStatistics(points)

	assert.GreaterOrEqual(t, mean, float64(0))
	assert.GreaterOrEqual(t, stdDev, float64(0))
	assert.GreaterOrEqual(t, maxCurv, float64(0))
}

func TestBehaviorAnalysisService_ComputeCurvatureStatistics_ShortInput(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
	}

	mean, stdDev, maxCurv := service.ComputeCurvatureStatistics(points)
	assert.Equal(t, float64(0), mean)
	assert.Equal(t, float64(0), stdDev)
	assert.Equal(t, float64(0), maxCurv)
}

func TestBehaviorAnalysisService_ComputeTrajectorySmoothnessMetrics(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 130, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 160, Timestamp: 1300, Event: "move"},
	}

	metrics := service.ComputeTrajectorySmoothnessMetrics(points)

	assert.Contains(t, metrics, "avg_angle_change")
	assert.Contains(t, metrics, "angle_variance")
	assert.Contains(t, metrics, "sharp_turns")
	assert.Contains(t, metrics, "smoothness_score")
}

func TestBehaviorAnalysisService_ComputeTrajectorySmoothnessMetrics_ShortInput(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
	}

	metrics := service.ComputeTrajectorySmoothnessMetrics(points)
	assert.Equal(t, 0, len(metrics))
}

func TestBehaviorAnalysisService_DetectAccelerationAnomalies(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 130, Timestamp: 1200, Event: "move"},
	}

	accelerations := []float64{0.5, 0.6, 0.4, 0.55, 0.45}

	result := service.DetectAccelerationAnomalies(points, accelerations)

	assert.Contains(t, result, "has_anomaly")
	assert.Contains(t, result, "anomaly_count")
}

func TestBehaviorAnalysisService_DetectAccelerationAnomalies_EmptyAccel(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
	}

	accelerations := []float64{}

	result := service.DetectAccelerationAnomalies(points, accelerations)
	assert.Equal(t, false, result["has_anomaly"])
}

func TestBehaviorAnalysisService_AnalyzeAccelerationPattern(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 130, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 160, Timestamp: 1300, Event: "move"},
	}

	pattern := service.AnalyzeAccelerationPattern(points)

	assert.Contains(t, pattern, "mean_acceleration")
	assert.Contains(t, pattern, "acceleration_variance")
	assert.Contains(t, pattern, "max_acceleration")
}

func TestBehaviorAnalysisService_AnalyzeAccelerationPattern_ShortInput(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
	}

	pattern := service.AnalyzeAccelerationPattern(points)
	assert.Equal(t, 0, len(pattern))
}

func TestBehaviorAnalysisService_countOscillations(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		minOsc   int
		maxOsc   int
	}{
		{"increasing", []float64{1, 2, 3, 4, 5}, 0, 5},
		{"decreasing", []float64{5, 4, 3, 2, 1}, 0, 5},
		{"oscillating", []float64{1, 3, 2, 4, 3, 5}, 0, 5},
		{"constant", []float64{1, 1, 1, 1}, 0, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oscillations := service.countOscillations(tt.values)
			assert.GreaterOrEqual(t, oscillations, tt.minOsc)
			assert.LessOrEqual(t, oscillations, tt.maxOsc)
		})
	}
}

func TestBehaviorAnalysisService_calculateJerkiness(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name       string
		accels     []float64
		minJerk    float64
		maxJerk    float64
	}{
		{"stable", []float64{0.5, 0.5, 0.5, 0.5}, 0.0, 0.1},
		{"varying", []float64{0.5, 1.0, 0.3, 0.8}, 0.0, 1.0},
		{"empty", []float64{}, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jerkiness := service.calculateJerkiness(tt.accels)
			assert.GreaterOrEqual(t, jerkiness, tt.minJerk)
			assert.LessOrEqual(t, jerkiness, tt.maxJerk)
		})
	}
}

func TestBehaviorAnalysisService_checkPathSimilarity(t *testing.T) {
	service := NewBehaviorAnalysisService()

	currentPath := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 250, Timestamp: 1300, Event: "move"},
		{X: 300, Y: 300, Timestamp: 1400, Event: "move"},
	}

	similarity := service.checkPathSimilarity(currentPath)

	assert.Equal(t, 5, similarity.ComparedPathLength)
	assert.LessOrEqual(t, similarity.SimilarityScore, float64(1))
	assert.GreaterOrEqual(t, similarity.SimilarityScore, float64(0))
}

func TestBehaviorAnalysisService_CalculateRiskScore(t *testing.T) {
	service := NewBehaviorAnalysisService()

	verification := &models.Verification{
		Status: "success",
	}
	behaviorData := []models.BehaviorData{
		{
			DataType: "mouse",
			Data:     `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`,
		},
	}

	score := service.CalculateRiskScore(verification, behaviorData)
	assert.GreaterOrEqual(t, score, float64(0))
	assert.LessOrEqual(t, score, float64(100))
}

func TestBehaviorAnalysisService_GenerateAnalysisReport(t *testing.T) {
	service := NewBehaviorAnalysisService()

	result := &AnalysisResult{
		RiskScore:   45.5,
		IsBotLikely: false,
		Confidence:  0.85,
		RiskIndicators: []string{
			"速度变化较大",
			"路径有轻微曲折",
		},
		Trajectory: MouseTrajectory{
			TotalDistance:   500.0,
			AverageSpeed:    1.5,
			MaxSpeed:        3.0,
			MinSpeed:        0.5,
			PathEfficiency:  0.85,
			SmoothedDistance: 480.0,
			SpeedVariance:  0.25,
			CurvatureAvg:   0.15,
			JitterScore:    0.04,
			DirectionChanges: 5,
			PauseCount:     2,
			MicroCorrections: 3,
		},
		SpeedAnalysis: SpeedAnalysis{
			AverageSpeed:  1.5,
			MedianSpeed:   1.4,
			SpeedStdDev:  0.5,
			SpeedSkewness: 0.2,
		},
		PathSimilarity: PathSimilarity{
			SimilarityScore: 0.75,
		},
		ClickPattern: ClickPattern{
			ClickCount:       3,
			AverageInterval:  250,
			IntervalVariance: 50,
			ClickSpeed:      4.0,
			Regularity:      0.8,
			PositionEntropy: 2.5,
			ClickAreaSize:  10.0,
		},
	}

	report := service.GenerateAnalysisReport(result)
	assert.NotEmpty(t, report)
	assert.Contains(t, report, "风险评分: 45.50")
	assert.Contains(t, report, "疑似机器人: false")
	assert.Contains(t, report, "置信度: 0.85")
}

func TestBehaviorAnalysisService_VerifyWithBehaviorAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			DataType: "mouse",
			Data:     `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 150, "y": 150, "timestamp": 1300, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 200, "y": 200, "timestamp": 1600, "event": "click"}`,
		},
	}

	tests := []struct {
		name         string
		captchaSuccess bool
		minRiskScore float64
		maxRiskScore float64
	}{
		{"captcha success", true, 0, 100},
		{"captcha failure", false, 0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, riskScore, _ := service.VerifyWithBehaviorAnalysis(tt.captchaSuccess, behaviorData)
			assert.GreaterOrEqual(t, riskScore, tt.minRiskScore)
			assert.LessOrEqual(t, riskScore, tt.maxRiskScore)
			t.Logf("Test %s: success=%v, riskScore=%.2f", tt.name, success, riskScore)
		})
	}
}

func TestBehaviorAnalysisService_AnalyzeSpeed(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			DataType: "mouse",
			Data:     `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 150, "y": 150, "timestamp": 1100, "event": "move"}`,
		},
		{
			DataType: "mouse",
			Data:     `{"x": 200, "y": 200, "timestamp": 1200, "event": "move"}`,
		},
	}

	analysis, err := service.AnalyzeSpeed(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Greater(t, analysis.AverageSpeed, float64(0))
}

func TestBehaviorAnalysisService_AnalyzePathSimilarity(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 250, Timestamp: 1300, Event: "move"},
		{X: 300, Y: 300, Timestamp: 1400, Event: "move"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 155, Y: 155, Timestamp: 1100, Event: "move"},
		{X: 205, Y: 205, Timestamp: 1200, Event: "move"},
		{X: 255, Y: 255, Timestamp: 1300, Event: "move"},
		{X: 305, Y: 305, Timestamp: 1400, Event: "move"},
	}

	similarity := service.AnalyzePathSimilarity(path1, path2)

	assert.GreaterOrEqual(t, similarity.SimilarityScore, float64(0))
	assert.LessOrEqual(t, similarity.SimilarityScore, float64(1))
	assert.GreaterOrEqual(t, similarity.DTWDistance, float64(0))
	assert.GreaterOrEqual(t, similarity.FrechetDistance, float64(0))
}

func TestBehaviorAnalysisService_AnalyzePathSimilarity_ShortPaths(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "move"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 155, Y: 155, Timestamp: 1100, Event: "move"},
	}

	similarity := service.AnalyzePathSimilarity(path1, path2)

	assert.Equal(t, float64(0), similarity.SimilarityScore)
	assert.Equal(t, float64(0), similarity.DTWDistance)
}

func TestBehaviorAnalysisService_mean(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal", []float64{1, 2, 3, 4, 5}, 3.0},
		{"single", []float64{5}, 5.0},
		{"empty", []float64{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.mean(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestBehaviorAnalysisService_variance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		minVar   float64
		maxVar   float64
	}{
		{"uniform", []float64{5, 5, 5, 5}, 0.0, 0.001},
		{"varied", []float64{1, 2, 3, 4, 5}, 1.5, 2.5},
		{"empty", []float64{}, 0.0, 0.0},
		{"single", []float64{5}, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.variance(tt.values)
			assert.GreaterOrEqual(t, result, tt.minVar)
			assert.LessOrEqual(t, result, tt.maxVar)
		})
	}
}

func TestBehaviorAnalysisService_max(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal", []float64{1, 2, 3, 4, 5}, 5.0},
		{"negative", []float64{-5, -3, -1}, -1.0},
		{"empty", []float64{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.max(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestBehaviorAnalysisService_min(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"normal", []float64{1, 2, 3, 4, 5}, 1.0},
		{"negative", []float64{-5, -3, -1}, -5.0},
		{"empty", []float64{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.min(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestBehaviorAnalysisService_maxAbs(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"positive", []float64{1, 2, 3, 4, 5}, 5.0},
		{"negative", []float64{-5, -3, -1}, 5.0},
		{"mixed", []float64{-5, 2, -3, 4}, 5.0},
		{"empty", []float64{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.maxAbs(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestBehaviorAnalysisService_median(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"odd", []float64{1, 2, 3, 4, 5}, 3.0},
		{"even", []float64{1, 2, 3, 4}, 2.5},
		{"single", []float64{5}, 5.0},
		{"empty", []float64{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.median(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestBehaviorAnalysisService_skewness(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		values   []float64
		minSkew  float64
		maxSkew  float64
	}{
		{"normal", []float64{1, 2, 3, 4, 5}, -1.0, 1.0},
		{"symmetric", []float64{1, 2, 2, 2, 3}, -0.5, 0.5},
		{"empty", []float64{}, 0.0, 0.0},
		{"single", []float64{5}, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.skewness(tt.values)
			assert.GreaterOrEqual(t, result, tt.minSkew)
			assert.LessOrEqual(t, result, tt.maxSkew)
		})
	}
}

func TestScoreCard_NewScoreCard(t *testing.T) {
	sc := NewScoreCard()
	assert.NotNil(t, sc)
	assert.NotNil(t, sc.Weights)
	assert.NotNil(t, sc.Thresholds)
	assert.Equal(t, 7, len(sc.Weights))
	assert.Equal(t, 7, len(sc.Thresholds))
}

func TestScoreCard_Evaluate(t *testing.T) {
	sc := NewScoreCard()

	features := &BehaviorFeatures{
		AvgSpeed:           500,
		TrajectorySmoothness: 0.8,
		Acceleration:      0.5,
		PathComplexity:    0.4,
		PathSimilarity:    0.6,
		SpeedVariation:     0.3,
		ClickInterval:     100,
	}

	score := sc.Evaluate(features)
	assert.GreaterOrEqual(t, score, float64(0))
	assert.LessOrEqual(t, score, float64(100))
}

func TestScoreCard_Evaluate_NilFeatures(t *testing.T) {
	sc := NewScoreCard()
	score := sc.Evaluate(nil)
	assert.Equal(t, float64(0), score)
}

func TestExtractFeatures(t *testing.T) {
	trajectory := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 150, Y: 110, Timestamp: 1100},
		{X: 200, Y: 130, Timestamp: 1200},
		{X: 250, Y: 160, Timestamp: 1300},
	}

	features := ExtractFeatures(trajectory)
	assert.NotNil(t, features)
	assert.Greater(t, features.AvgSpeed, float64(0))
}

func TestExtractFeatures_ShortTrajectory(t *testing.T) {
	trajectory := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 1000},
	}

	features := ExtractFeatures(trajectory)
	assert.NotNil(t, features)
	assert.Equal(t, float64(0), features.AvgSpeed)
}

func TestCalculateAverageSpeed(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
	}

	speed := CalculateAverageSpeed(points)
	assert.Greater(t, speed, float64(0))
}

func TestCalculateMaxSpeed(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
		{X: 50, Y: 0, Timestamp: 1200},
	}

	speed := CalculateMaxSpeed(points)
	assert.Greater(t, speed, float64(0))
}

func TestCalculateMinSpeed(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
		{X: 50, Y: 0, Timestamp: 1200},
	}

	speed := CalculateMinSpeed(points)
	assert.GreaterOrEqual(t, speed, float64(0))
}

func TestCalculateSpeedVariation(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
		{X: 200, Y: 0, Timestamp: 1200},
	}

	variation := CalculateSpeedVariation(points)
	assert.GreaterOrEqual(t, variation, float64(0))
}

func TestCalculateAcceleration(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
		{X: 200, Y: 0, Timestamp: 1200},
		{X: 300, Y: 0, Timestamp: 1300},
	}

	accel := CalculateAcceleration(points)
	assert.GreaterOrEqual(t, accel, float64(0))
}

func TestCalculateTrajectorySmoothness(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 50, Y: 50, Timestamp: 1100},
		{X: 100, Y: 100, Timestamp: 1200},
	}

	smoothness := CalculateTrajectorySmoothness(points)
	assert.GreaterOrEqual(t, smoothness, float64(0))
	assert.LessOrEqual(t, smoothness, float64(1))
}

func TestCalculateClickInterval(t *testing.T) {
	clicks := []BehaviorClickData{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 150, Y: 150, Timestamp: 1200},
		{X: 200, Y: 200, Timestamp: 1400},
	}

	interval := CalculateClickInterval(clicks)
	assert.Greater(t, interval, float64(0))
}

func TestCalculateClickPositionVariance(t *testing.T) {
	clicks := []BehaviorClickData{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 150, Y: 150, Timestamp: 1200},
		{X: 200, Y: 200, Timestamp: 1400},
	}

	variance := CalculateClickPositionVariance(clicks)
	assert.GreaterOrEqual(t, variance, float64(0))
}

func TestCalculatePathComplexity(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 50, Y: 50, Timestamp: 1100},
		{X: 100, Y: 100, Timestamp: 1200},
	}

	complexity := CalculatePathComplexity(points)
	assert.GreaterOrEqual(t, complexity, float64(0))
	assert.LessOrEqual(t, complexity, float64(1))
}

func TestDTWDistance(t *testing.T) {
	seq1 := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 150, Y: 150, Timestamp: 1100},
	}

	seq2 := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 155, Y: 155, Timestamp: 1100},
	}

	dist := DTWDistance(seq1, seq2)
	assert.Greater(t, dist, float64(0))
}

func TestCompareWithHumanTrajectory(t *testing.T) {
	trajectory := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 120, Y: 115, Timestamp: 50},
		{X: 145, Y: 135, Timestamp: 100},
	}

	similarity := CompareWithHumanTrajectory(trajectory)
	assert.GreaterOrEqual(t, similarity, float64(0))
	assert.LessOrEqual(t, similarity, float64(1))
}

func TestNormalizeTrajectory(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
	}

	normalized := normalizeTrajectory(points)
	assert.Equal(t, len(points), len(normalized))
}

func TestCalculateRiskScore_Features(t *testing.T) {
	features := &BehaviorFeatures{
		AvgSpeed:           500,
		TrajectorySmoothness: 0.8,
		Acceleration:      0.5,
		PathComplexity:    0.4,
		PathSimilarity:    0.6,
		SpeedVariation:     0.3,
		ClickInterval:     100,
	}

	score := CalculateRiskScore(features)
	assert.GreaterOrEqual(t, score, float64(0))
	assert.LessOrEqual(t, score, float64(100))
}

func TestIsRobot(t *testing.T) {
	tests := []struct {
		name     string
		features *BehaviorFeatures
		expected bool
	}{
		{
			"low risk",
			&BehaviorFeatures{RiskScore: 30},
			false,
		},
		{
			"medium risk",
			&BehaviorFeatures{RiskScore: 50},
			true,
		},
		{
			"high risk",
			&BehaviorFeatures{RiskScore: 80},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRobot(tt.features)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertToClickData(t *testing.T) {
	trajectory := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 150, Y: 150, Timestamp: 100},
		{X: 200, Y: 200, Timestamp: 200},
	}

	clicks := convertToClickData(trajectory)
	assert.Equal(t, 2, len(clicks))
}

func TestAnalyzeClickPatternEnhanced(t *testing.T) {
	service := NewBehaviorAnalysisService()

	clicks := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
		{X: 150, Y: 150, Timestamp: 1300, Event: "click"},
		{X: 200, Y: 200, Timestamp: 1600, Event: "click"},
	}

	allPoints := append(clicks,
		BehaviorDataPoint{X: 110, Y: 110, Timestamp: 1050, Event: "move"},
		BehaviorDataPoint{X: 160, Y: 160, Timestamp: 1350, Event: "move"},
	)

	pattern := service.analyzeClickPatternEnhanced(clicks, allPoints)

	assert.Equal(t, 3, pattern.ClickCount)
	assert.Greater(t, pattern.AverageInterval, float64(0))
	assert.GreaterOrEqual(t, pattern.Regularity, float64(0))
	assert.LessOrEqual(t, pattern.Regularity, float64(1))
}

func TestAnalyzeClickPatternEnhanced_SingleClick(t *testing.T) {
	service := NewBehaviorAnalysisService()

	clicks := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
	}

	allPoints := []BehaviorDataPoint{
		{X: 50, Y: 50, Timestamp: 500, Event: "move"},
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
	}

	pattern := service.analyzeClickPatternEnhanced(clicks, allPoints)
	assert.Equal(t, 1, pattern.ClickCount)
}

func TestAnalyzeClickPatternEnhanced_EmptyClicks(t *testing.T) {
	service := NewBehaviorAnalysisService()

	clicks := []BehaviorDataPoint{}
	allPoints := []BehaviorDataPoint{}

	pattern := service.analyzeClickPatternEnhanced(clicks, allPoints)
	assert.Equal(t, 0, pattern.ClickCount)
}

func TestAnalyzeKeyboardPattern(t *testing.T) {
	service := NewBehaviorAnalysisService()

	keyStrokes := []KeyboardDataPoint{
		{Key: "a", Timestamp: 1000, KeyDownTime: 1000, KeyUpTime: 1050, HoldDuration: 50},
		{Key: "b", Timestamp: 1150, KeyDownTime: 1150, KeyUpTime: 1200, HoldDuration: 50},
		{Key: "c", Timestamp: 1300, KeyDownTime: 1300, KeyUpTime: 1350, HoldDuration: 50},
	}

	pattern := service.analyzeKeyboardPattern(keyStrokes)

	assert.Equal(t, 3, pattern.KeystrokeCount)
	assert.Greater(t, pattern.TypingSpeed, float64(0))
	assert.GreaterOrEqual(t, pattern.Regularity, float64(0))
}

func TestAnalyzeKeyboardPattern_SingleKeystroke(t *testing.T) {
	service := NewBehaviorAnalysisService()

	keyStrokes := []KeyboardDataPoint{
		{Key: "a", Timestamp: 1000, KeyDownTime: 1000, KeyUpTime: 1050, HoldDuration: 50},
	}

	pattern := service.analyzeKeyboardPattern(keyStrokes)
	assert.Equal(t, 1, pattern.KeystrokeCount)
}

func TestAnalyzeKeyboardPattern_Empty(t *testing.T) {
	service := NewBehaviorAnalysisService()

	keyStrokes := []KeyboardDataPoint{}

	pattern := service.analyzeKeyboardPattern(keyStrokes)
	assert.Equal(t, 0, pattern.KeystrokeCount)
}

func TestAnalyzeMouseTrajectory(t *testing.T) {
	service := NewBehaviorAnalysisService()

	originalPoints := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 130, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 160, Timestamp: 1300, Event: "move"},
	}

	smoothedPoints := originalPoints

	traj := service.analyzeMouseTrajectory(smoothedPoints, originalPoints)

	assert.Greater(t, traj.TotalDistance, float64(0))
	assert.Greater(t, traj.AverageSpeed, float64(0))
	assert.Greater(t, traj.MaxSpeed, float64(0))
	assert.GreaterOrEqual(t, traj.PathEfficiency, float64(0))
	assert.LessOrEqual(t, traj.PathEfficiency, float64(1))
}

func TestAnalyzeMouseTrajectory_ShortInput(t *testing.T) {
	service := NewBehaviorAnalysisService()

	originalPoints := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
	}

	smoothedPoints := originalPoints

	traj := service.analyzeMouseTrajectory(smoothedPoints, originalPoints)
	assert.Equal(t, float64(0), traj.TotalDistance)
}

func TestComputeCurvature(t *testing.T) {
	service := NewBehaviorAnalysisService()

	p1 := BehaviorDataPoint{X: 0, Y: 0, Timestamp: 1000}
	p2 := BehaviorDataPoint{X: 100, Y: 0, Timestamp: 1100}
	p3 := BehaviorDataPoint{X: 100, Y: 100, Timestamp: 1200}

	curvature := service.computeCurvature(p1, p2, p3)
	assert.GreaterOrEqual(t, curvature, float64(-3.15))
	assert.LessOrEqual(t, curvature, float64(3.15))
}

func TestComputeCurvature_ZeroVectors(t *testing.T) {
	service := NewBehaviorAnalysisService()

	p1 := BehaviorDataPoint{X: 100, Y: 100, Timestamp: 1000}
	p2 := BehaviorDataPoint{X: 100, Y: 100, Timestamp: 1100}
	p3 := BehaviorDataPoint{X: 100, Y: 100, Timestamp: 1200}

	curvature := service.computeCurvature(p1, p2, p3)
	assert.Equal(t, float64(0), curvature)
}

func BenchmarkBehaviorAnalysis_HumanLike(b *testing.B) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{DataType: "mouse", Data: `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 150, "y": 120, "timestamp": 1150, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 200, "y": 150, "timestamp": 1300, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 250, "y": 180, "timestamp": 1500, "event": "click"}`},
		{DataType: "mouse", Data: `{"x": 300, "y": 220, "timestamp": 1700, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 350, "y": 260, "timestamp": 1900, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 400, "y": 300, "timestamp": 2100, "event": "click"}`},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.AnalyzeBehavior(behaviorData)
	}
}

func BenchmarkBehaviorAnalysis_BotLike(b *testing.B) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{DataType: "mouse", Data: `{"x": 100, "y": 100, "timestamp": 1000, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 200, "y": 200, "timestamp": 1050, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 300, "y": 300, "timestamp": 1100, "event": "move"}`},
		{DataType: "mouse", Data: `{"x": 400, "y": 400, "timestamp": 1150, "event": "click"}`},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.AnalyzeBehavior(behaviorData)
	}
}

func TestBehaviorAnalysisService_AnalyzeSpeedWithAllFields(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 110, Timestamp: 1100, Event: "move"},
		{X: 200, Y: 130, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 160, Timestamp: 1300, Event: "move"},
		{X: 300, Y: 200, Timestamp: 1400, Event: "move"},
		{X: 350, Y: 250, Timestamp: 1500, Event: "move"},
	}

	analysis := service.analyzeSpeed(points)

	assert.NotNil(t, analysis.Speeds)
	assert.Len(t, analysis.Speeds, len(points)-1)
	assert.Greater(t, analysis.AverageSpeed, float64(0))
	assert.Greater(t, analysis.MaxSpeed, float64(0))
	assert.GreaterOrEqual(t, analysis.MinSpeed, float64(0))
	assert.GreaterOrEqual(t, analysis.SpeedVariance, float64(0))
	assert.GreaterOrEqual(t, analysis.SpeedStdDev, float64(0))
	assert.GreaterOrEqual(t, analysis.SpeedSkewness, float64(-10))
	assert.LessOrEqual(t, analysis.SpeedSkewness, float64(10))
}

func TestBehaviorAnalysisService_calculateRiskScoreWithAllFactors(t *testing.T) {
	service := NewBehaviorAnalysisService()

	testCases := []struct {
		name         string
		speedAnalysis SpeedAnalysis
		trajectory    MouseTrajectory
		pathSimilarity PathSimilarity
		clickPattern  ClickPattern
		keyboardPattern KeyboardPattern
		minRiskScore  float64
		maxRiskScore  float64
	}{
		{
			"all low risk",
			SpeedAnalysis{Speeds: []float64{1, 1.1, 1.2}, SpeedOutliers: 0, IsSpeedConsistent: true, AverageSpeed: 1.1, SpeedStdDev: 0.1},
			MouseTrajectory{PathEfficiency: 0.5, JitterScore: 0.1, CurvatureAvg: 0.5, PauseCount: 5, MicroCorrections: 10, AccelerationMagVariance: 0.1, Points: make([]BehaviorDataPoint, 50)},
			PathSimilarity{},
			ClickPattern{Regularity: 0.5, PositionEntropy: 3.0, ClickAreaSize: 50.0, ClickCount: 3},
			KeyboardPattern{},
			0, 30,
		},
		{
			"all high risk",
			SpeedAnalysis{Speeds: []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, SpeedOutliers: 0, IsSpeedConsistent: true, AverageSpeed: 1.0, SpeedStdDev: 0.001},
			MouseTrajectory{PathEfficiency: 0.98, JitterScore: 0.001, CurvatureAvg: 0.001, PauseCount: 0, MicroCorrections: 0, AccelerationMagVariance: 0.00001, Points: make([]BehaviorDataPoint, 50)},
			PathSimilarity{IsPathRepeated: true, PathHashMatch: true, DTWDistance: 10, SimilarityScore: 0.95},
			ClickPattern{Regularity: 0.99, PositionEntropy: 0.5, ClickAreaSize: 1.0, ClickCount: 5},
			KeyboardPattern{},
			50, 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &AnalysisResult{
				SpeedAnalysis:   tc.speedAnalysis,
				Trajectory:       tc.trajectory,
				PathSimilarity:   tc.pathSimilarity,
				ClickPattern:     tc.clickPattern,
				KeyboardPattern: tc.keyboardPattern,
			}

			service.calculateRiskScoreEnhanced(result)

			t.Logf("Test %s: RiskScore = %.2f, Indicators = %v", tc.name, result.RiskScore, result.RiskIndicators)

			assert.GreaterOrEqual(t, result.RiskScore, tc.minRiskScore)
			assert.LessOrEqual(t, result.RiskScore, tc.maxRiskScore)
		})
	}
}

func TestBehaviorAnalysisService_computeEntropy(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name   string
		counts []int
		minEnt float64
		maxEnt float64
	}{
		{"uniform", []int{10, 10, 10, 10}, 1.8, 2.2},
		{"skewed", []int{100, 0, 0, 0}, 0.0, 0.5},
		{"empty", []int{}, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := service.computeEntropy(tt.counts)
			assert.GreaterOrEqual(t, entropy, tt.minEnt)
			assert.LessOrEqual(t, entropy, tt.maxEnt)
		})
	}
}

func TestBehaviorAnalysisService_computePositionDistribution(t *testing.T) {
	service := NewBehaviorAnalysisService()

	clicks := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
		{X: 200, Y: 200, Timestamp: 1100, Event: "click"},
		{X: 300, Y: 300, Timestamp: 1200, Event: "click"},
		{X: 400, Y: 400, Timestamp: 1300, Event: "click"},
	}

	xDist := service.computePositionDistribution(clicks, true, 10)
	yDist := service.computePositionDistribution(clicks, false, 10)

	assert.Len(t, xDist, 10)
	assert.Len(t, yDist, 10)
}

func TestBehaviorAnalysisService_computePreClickHesitation(t *testing.T) {
	service := NewBehaviorAnalysisService()

	click := BehaviorDataPoint{X: 300, Y: 300, Timestamp: 1500, Event: "click"}
	allPoints := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
		{X: 200, Y: 200, Timestamp: 1300, Event: "move"},
		{X: 300, Y: 300, Timestamp: 1500, Event: "click"},
	}

	hesitation := service.computePreClickHesitation(click, allPoints)
	assert.GreaterOrEqual(t, hesitation, float64(0))
}

func TestBehaviorAnalysisService_computePreClickHesitation_NoMove(t *testing.T) {
	service := NewBehaviorAnalysisService()

	click := BehaviorDataPoint{X: 100, Y: 100, Timestamp: 1000, Event: "click"}
	allPoints := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
	}

	hesitation := service.computePreClickHesitation(click, allPoints)
	assert.Equal(t, float64(0), hesitation)
}

func TestBehaviorAnalysisService_AnalyzeClickRhythmAdvanced(t *testing.T) {
	service := NewBehaviorAnalysisService()

	clicks := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
		{X: 150, Y: 150, Timestamp: 1300, Event: "click"},
		{X: 200, Y: 200, Timestamp: 1600, Event: "click"},
		{X: 250, Y: 250, Timestamp: 1900, Event: "click"},
	}

	allPoints := make([]BehaviorDataPoint, 0, len(clicks)*2)
	for _, c := range clicks {
		allPoints = append(allPoints, c)
		allPoints = append(allPoints, BehaviorDataPoint{X: c.X - 10, Y: c.Y - 10, Timestamp: c.Timestamp - 50, Event: "move"})
	}

	pattern := service.AnalyzeClickRhythmAdvanced(clicks, allPoints)

	assert.Equal(t, 4, pattern.ClickCount)
	assert.GreaterOrEqual(t, pattern.ClickIntervalCV, float64(0))
	assert.GreaterOrEqual(t, pattern.ClickBurstiness, float64(0))
	assert.GreaterOrEqual(t, pattern.ClickRhythmConsistency, float64(0))
	assert.LessOrEqual(t, pattern.ClickRhythmConsistency, float64(1))
	assert.NotEmpty(t, pattern.ClickTimingPattern)
}

func TestBehaviorAnalysisService_calculateClickIntervalCoefficientOfVariation(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name       string
		clicks     []BehaviorDataPoint
		minCV      float64
		maxCV      float64
	}{
		{"regular", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1100},
			{X: 200, Y: 200, Timestamp: 1200},
			{X: 250, Y: 250, Timestamp: 1300},
		}, 0.0, 0.2},
		{"irregular", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1200},
			{X: 200, Y: 200, Timestamp: 1600},
			{X: 250, Y: 250, Timestamp: 2500},
		}, 0.5, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := service.calculateClickIntervalCoefficientOfVariation(tt.clicks)
			assert.GreaterOrEqual(t, cv, tt.minCV)
			assert.LessOrEqual(t, cv, tt.maxCV)
		})
	}
}

func TestBehaviorAnalysisService_calculateClickBurstiness(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name       string
		clicks     []BehaviorDataPoint
		minBurst   float64
		maxBurst   float64
	}{
		{"uniform", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1100},
			{X: 200, Y: 200, Timestamp: 1200},
		}, 0.0, 0.5},
		{"bursty", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1050},
			{X: 200, Y: 200, Timestamp: 2000},
		}, 0.5, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			burstiness := service.calculateClickBurstiness(tt.clicks)
			assert.GreaterOrEqual(t, burstiness, tt.minBurst)
			assert.LessOrEqual(t, burstiness, tt.maxBurst)
		})
	}
}

func TestBehaviorAnalysisService_calculateClickRhythmConsistency(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name       string
		clicks     []BehaviorDataPoint
		minConsist float64
		maxConsist float64
	}{
		{"consistent", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1100},
			{X: 200, Y: 200, Timestamp: 1200},
			{X: 250, Y: 250, Timestamp: 1300},
		}, 0.7, 1.0},
		{"inconsistent", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1500},
			{X: 200, Y: 200, Timestamp: 1550},
			{X: 250, Y: 250, Timestamp: 3000},
		}, 0.0, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consistency := service.calculateClickRhythmConsistency(tt.clicks)
			assert.GreaterOrEqual(t, consistency, tt.minConsist)
			assert.LessOrEqual(t, consistency, tt.maxConsist)
		})
	}
}

func TestBehaviorAnalysisService_analyzeClickTimingPattern(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name     string
		clicks  []BehaviorDataPoint
		patterns []string
	}{
		{"mechanical", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1010},
			{X: 200, Y: 200, Timestamp: 1020},
		}, []string{"mechanical", "regular"}},
		{"natural", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1100},
			{X: 200, Y: 200, Timestamp: 1600},
		}, []string{"natural", "erratic"}},
		{"erratic", []BehaviorDataPoint{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 150, Y: 150, Timestamp: 1500},
			{X: 200, Y: 200, Timestamp: 1600},
		}, []string{"natural", "erratic"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := service.analyzeClickTimingPattern(tt.clicks)
			found := false
			for _, p := range tt.patterns {
				if pattern == p {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected one of %v, got %s", tt.patterns, pattern)
		})
	}
}

func TestAnalyzeBehavior_WithJSONParsingErrors(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{DataType: "mouse", Data: `invalid json`},
		{DataType: "keyboard", Data: `{"key": "a", "timestamp": 1000}`},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateAnalysisReport_EmptyResult(t *testing.T) {
	service := NewBehaviorAnalysisService()

	result := &AnalysisResult{}

	report := service.GenerateAnalysisReport(result)
	assert.NotEmpty(t, report)
}

func TestAnalyzeSpeed_EmptyBehaviorData(t *testing.T) {
	service := NewBehaviorAnalysisService()

	analysis, err := service.AnalyzeSpeed([]models.BehaviorData{})
	assert.NoError(t, err)
	assert.NotNil(t, analysis)
}

func TestCheckPathSimilarity_EmptyPath(t *testing.T) {
	service := NewBehaviorAnalysisService()

	similarity := service.checkPathSimilarity([]BehaviorDataPoint{})
	assert.Equal(t, 0, similarity.ComparedPathLength)
}

func TestCheckPathHashMatch_EmptyStoredPaths(t *testing.T) {
	service := NewBehaviorAnalysisService()

	result := service.checkPathHashMatch("test|hash|path")
	assert.False(t, result)
}

func TestInvertMatrix(t *testing.T) {
	service := NewBehaviorAnalysisService()

	matrix := [][]float64{
		{4, 7},
		{2, 6},
	}

	inverse := service.invertMatrix(matrix)
	assert.NotNil(t, inverse)
	assert.Len(t, inverse, 2)
	assert.Len(t, inverse[0], 2)
}

func TestComputeSGCoefficients(t *testing.T) {
	service := NewBehaviorAnalysisService()

	coeffs := service.computeSGCoefficients(5, 2)
	assert.Len(t, coeffs, 5)

	sum := float64(0)
	for _, c := range coeffs {
		sum += c
	}
	assert.True(t, sum > 0)
}

func TestSavitzkyGolaySmooth(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 110, Y: 105, Timestamp: 1100},
		{X: 120, Y: 115, Timestamp: 1200},
		{X: 130, Y: 125, Timestamp: 1300},
		{X: 140, Y: 135, Timestamp: 1400},
	}

	smoothed := service.SavitzkyGolaySmooth(points, 3, 2)
	assert.Equal(t, len(points), len(smoothed))
}

func TestSavitzkyGolaySmooth_InvalidInput(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 110, Y: 105, Timestamp: 1100},
	}

	smoothed := service.SavitzkyGolaySmooth(points, 5, 2)
	assert.Equal(t, points, smoothed)

	smoothed2 := service.SavitzkyGolaySmooth(points, 3, 5)
	assert.Equal(t, points, smoothed2)
}

func TestCalculateSpeedRampIndex(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := make([]BehaviorDataPoint, 20)
	for i := range points {
		points[i] = BehaviorDataPoint{
			X:         100 + i*10,
			Y:         100 + i*10,
			Timestamp: int64(1000 + i*100),
		}
	}

	index := service.CalculateSpeedRampIndex(points)
	t.Logf("Speed ramp index: %f", index)
}

func TestCalculateAccelerationTrend(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name   string
		accels []float64
	}{
		{"increasing", []float64{1, 2, 3, 4, 5, 6}},
		{"decreasing", []float64{6, 5, 4, 3, 2, 1}},
		{"stable", []float64{3, 3, 3, 3, 3}},
		{"empty", []float64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trend := service.CalculateAccelerationTrend(tt.accels)
			t.Logf("Acceleration trend for %s: %f", tt.name, trend)
		})
	}
}

func TestCalculateNormalizedSpeedVariance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name   string
		speeds []float64
	}{
		{"uniform", []float64{1, 1, 1, 1, 1}},
		{"varied", []float64{0.5, 1, 1.5, 2, 2.5}},
		{"empty", []float64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variance := service.CalculateNormalizedSpeedVariance(tt.speeds)
			t.Logf("Normalized speed variance for %s: %f", tt.name, variance)
		})
	}
}

func TestComputeEntropyFromFloat(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name    string
		values  []float64
		minEnt  float64
		maxEnt  float64
	}{
		{"normal", []float64{1.0, 2.0, 3.0, 4.0, 5.0}, 1.5, 2.5},
		{"empty", []float64{}, 0.0, 0.0},
		{"single", []float64{5.0}, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := service.ComputeEntropyFromFloat(tt.values)
			assert.GreaterOrEqual(t, entropy, tt.minEnt)
			assert.LessOrEqual(t, entropy, tt.maxEnt)
		})
	}
}

func TestPearsonCorrelation(t *testing.T) {
	service := NewBehaviorAnalysisService()

	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 6, 8, 10}

	corr := service.pearsonCorrelation(x, y)
	assert.Greater(t, corr, 0.9)
}

func TestPearsonCorrelation_DifferentLengths(t *testing.T) {
	service := NewBehaviorAnalysisService()

	x := []float64{1, 2, 3}
	y := []float64{2, 4}

	corr := service.pearsonCorrelation(x, y)
	assert.Equal(t, float64(0), corr)
}

func TestExtractCoordinates(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 250, Timestamp: 1100},
	}

	x, y := service.extractCoordinates(points)

	assert.Len(t, x, 2)
	assert.Len(t, y, 2)
	assert.Equal(t, float64(100), x[0])
	assert.Equal(t, float64(200), y[0])
}
