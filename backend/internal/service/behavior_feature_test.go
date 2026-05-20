package service

import (
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewMouseBehaviorExtractor(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()
	if extractor == nil {
		t.Error("NewMouseBehaviorExtractor() returned nil")
	}
	if extractor.windowSize != 5 {
		t.Errorf("expected windowSize to be 5, got %d", extractor.windowSize)
	}
	if extractor.threshold != 0.15 {
		t.Errorf("expected threshold to be 0.15, got %f", extractor.threshold)
	}
}

func TestExtractFeatures_WithValidData(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	data := &model.MouseBehaviorData{
		SessionID: "test-session-123",
		UserID:    "user-456",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
	}

	for i := 0; i < 10; i++ {
		data.MousePoints = append(data.MousePoints, model.MousePoint{
			X:         100 + i*10,
			Y:         200 + i*5,
			Timestamp: int64(i * 100),
			Button:    0,
		})
	}

	for i := 0; i < 5; i++ {
		data.ClickPoints = append(data.ClickPoints, model.ClickPoint{
			X:         150 + i*20,
			Y:         250 + i*10,
			Timestamp: int64(i * 200),
			Button:    1,
			ClickType: "single",
		})
	}

	features := extractor.ExtractFeatures(data)

	if features == nil {
		t.Error("ExtractFeatures() returned nil")
		return
	}

	if features.SpeedFeatures.AverageSpeed <= 0 {
		t.Error("expected positive average speed")
	}

	if features.AccelerationFeatures.AverageAcceleration == 0 {
		t.Log("acceleration feature is zero (expected for smooth data)")
	}

	if features.DistributionFeatures.XMean <= 0 {
		t.Error("expected positive X mean for click distribution")
	}

	if features.DoubleClickFeatures.SingleClickCount != 5 {
		t.Errorf("expected 5 single clicks, got %d", features.DoubleClickFeatures.SingleClickCount)
	}

	if features.LatencyFeatures.AverageLatency <= 0 {
		t.Log("latency feature is zero (expected when no movement before clicks)")
	}
}

func TestExtractFeatures_WithEmptyData(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name string
		data *model.MouseBehaviorData
	}{
		{"nil data", nil},
		{"empty data", &model.MouseBehaviorData{}},
		{"no mouse points", &model.MouseBehaviorData{
			MousePoints: []model.MousePoint{},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			features := extractor.ExtractFeatures(tc.data)

			if features == nil {
				t.Error("ExtractFeatures() returned nil")
				return
			}

			if !features.IsHumanLike && features.OverallScore == 100 {
				t.Log("correctly identified as non-human or high risk")
			}
		})
	}
}

func TestExtractSpeedFeatures(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name   string
		points []model.MousePoint
	}{
		{
			name:   "empty points",
			points: []model.MousePoint{},
		},
		{
			name:   "single point",
			points: []model.MousePoint{{X: 100, Y: 200, Timestamp: 0}},
		},
		{
			name: "normal movement",
			points: []model.MousePoint{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 150, Y: 120, Timestamp: 100},
				{X: 200, Y: 150, Timestamp: 200},
				{X: 250, Y: 180, Timestamp: 300},
			},
		},
		{
			name: "variable speed",
			points: []model.MousePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 10, Y: 0, Timestamp: 10},
				{X: 20, Y: 0, Timestamp: 100},
				{X: 30, Y: 0, Timestamp: 110},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			features := extractor.extractSpeedFeatures(tc.points)

			if len(tc.points) >= 2 {
				if features.AverageSpeed < 0 {
					t.Error("expected non-negative average speed")
				}
			} else {
				if features.AverageSpeed != 0 {
					t.Errorf("expected zero average speed for %s", tc.name)
				}
			}
		})
	}
}

func TestExtractAccelerationFeatures(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	points := []model.MousePoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 100, Timestamp: 100},
		{X: 200, Y: 200, Timestamp: 200},
		{X: 300, Y: 300, Timestamp: 300},
	}

	features := extractor.extractAccelerationFeatures(points)

	if len(points) < 3 {
		if features.AverageAcceleration != 0 {
			t.Error("expected zero acceleration for less than 3 points")
		}
	} else {
		t.Logf("acceleration: avg=%.6f, max=%.6f, jerk=%.6f",
			features.AverageAcceleration, features.MaxAcceleration, features.JerkAvg)
	}
}

func TestExtractDistributionFeatures(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name   string
		clicks []model.ClickPoint
	}{
		{
			name:   "empty clicks",
			clicks: []model.ClickPoint{},
		},
		{
			name: "single click",
			clicks: []model.ClickPoint{
				{X: 100, Y: 200, Timestamp: 0},
			},
		},
		{
			name: "distributed clicks",
			clicks: []model.ClickPoint{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 200, Y: 200, Timestamp: 100},
				{X: 300, Y: 300, Timestamp: 200},
				{X: 400, Y: 400, Timestamp: 300},
				{X: 500, Y: 500, Timestamp: 400},
			},
		},
		{
			name: "clustered clicks",
			clicks: []model.ClickPoint{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 105, Y: 105, Timestamp: 100},
				{X: 95, Y: 98, Timestamp: 200},
				{X: 110, Y: 102, Timestamp: 300},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			features := extractor.extractDistributionFeatures(tc.clicks)

			if len(tc.clicks) == 0 {
				if features.XMean != 0 || features.YMean != 0 {
					t.Error("expected zero means for empty clicks")
				}
			} else if len(tc.clicks) == 1 {
				if features.XMean != float64(tc.clicks[0].X) {
					t.Errorf("expected X mean to be %d", tc.clicks[0].X)
				}
			} else {
				if features.XMean <= 0 {
					t.Error("expected positive X mean for distributed clicks")
				}
				if features.Clusters < 1 {
					t.Error("expected at least one cluster")
				}
			}
		})
	}
}

func TestExtractDoubleClickFeatures(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name         string
		clicks       []model.ClickPoint
		expectedMin  int
		expectedMax  int
	}{
		{
			name:        "no clicks",
			clicks:      []model.ClickPoint{},
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name: "single click",
			clicks: []model.ClickPoint{
				{X: 100, Y: 100, Timestamp: 0, ClickType: "single"},
			},
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name: "double clicks",
			clicks: []model.ClickPoint{
				{X: 100, Y: 100, Timestamp: 0, ClickType: "single"},
				{X: 100, Y: 100, Timestamp: 150, ClickType: "single"},
				{X: 200, Y: 200, Timestamp: 500, ClickType: "single"},
				{X: 200, Y: 200, Timestamp: 600, ClickType: "single"},
			},
			expectedMin: 150,
			expectedMax: 600,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			features := extractor.extractDoubleClickFeatures(tc.clicks)

			if len(tc.clicks) >= 2 {
				if features.MinInterval < float64(tc.expectedMin)-1 {
					t.Errorf("expected min interval around %d, got %.2f", tc.expectedMin, features.MinInterval)
				}
				if features.MaxInterval > float64(tc.expectedMax)+1 {
					t.Errorf("expected max interval around %d, got %.2f", tc.expectedMax, features.MaxInterval)
				}
			} else {
				if features.AverageInterval != 0 {
					t.Error("expected zero average interval for less than 2 clicks")
				}
			}
		})
	}
}

func TestExtractLatencyFeatures(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	points := []model.MousePoint{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 110, Y: 110, Timestamp: 50},
		{X: 120, Y: 120, Timestamp: 100},
		{X: 130, Y: 130, Timestamp: 150},
	}

	clicks := []model.ClickPoint{
		{X: 140, Y: 140, Timestamp: 180},
		{X: 150, Y: 150, Timestamp: 300},
	}

	features := extractor.extractLatencyFeatures(points, clicks)

	if len(clicks) > 0 && len(points) > 0 {
		if features.AverageLatency < 0 {
			t.Error("expected non-negative average latency")
		}
		t.Logf("latency: avg=%.2f, median=%.2f, hesitation=%d",
			features.AverageLatency, features.MedianLatency, features.HesitationCount)
	}
}

func TestCalculateSpeeds(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	points := []model.MousePoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 100},
		{X: 200, Y: 0, Timestamp: 200},
	}

	speeds := extractor.calculateSpeeds(points)

	expectedCount := len(points) - 1
	if len(speeds) != expectedCount {
		t.Errorf("expected %d speeds, got %d", expectedCount, len(speeds))
	}

	for _, speed := range speeds {
		if speed < 0 {
			t.Error("expected non-negative speed")
		}
	}
}

func TestCalculateAccelerations(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	points := []model.MousePoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 100},
		{X: 200, Y: 0, Timestamp: 200},
		{X: 300, Y: 0, Timestamp: 300},
	}

	speeds := extractor.calculateSpeeds(points)
	accelerations := extractor.calculateAccelerations(speeds, points)

	expectedCount := len(speeds) - 2
	if len(accelerations) != expectedCount {
		t.Errorf("expected %d accelerations, got %d", expectedCount, len(accelerations))
	}
}

func TestCalculateClickIntervalsBehavior(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	clicks := []model.ClickPoint{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 100, Y: 100, Timestamp: 200},
		{X: 200, Y: 200, Timestamp: 500},
	}

	intervals := extractor.calculateClickIntervals(clicks)

	if len(intervals) != len(clicks)-1 {
		t.Errorf("expected %d intervals, got %d", len(clicks)-1, len(intervals))
	}

	if len(intervals) >= 1 {
		if intervals[0] != 200 {
			t.Errorf("expected first interval to be 200, got %.2f", intervals[0])
		}
	}
}

func TestCalculateLatencies(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	points := []model.MousePoint{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 110, Y: 110, Timestamp: 50},
		{X: 120, Y: 120, Timestamp: 100},
	}

	clicks := []model.ClickPoint{
		{X: 130, Y: 130, Timestamp: 120},
	}

	latencies := extractor.calculateLatencies(points, clicks)

	if len(latencies) != len(clicks) {
		t.Errorf("expected %d latencies, got %d", len(clicks), len(latencies))
	}

	if len(latencies) > 0 && latencies[0] < 0 {
		t.Error("expected non-negative latency")
	}
}

func TestMeanBehavior(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 5.0},
		{"normal", []float64{1.0, 2.0, 3.0}, 2.0},
		{"negative", []float64{-1.0, 0.0, 1.0}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.mean(tc.values)
			if result != tc.expected {
				t.Errorf("expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestMedian(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 5.0},
		{"odd length", []float64{1.0, 2.0, 3.0}, 2.0},
		{"even length", []float64{1.0, 2.0, 3.0, 4.0}, 2.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.median(tc.values)
			if result != tc.expected {
				t.Errorf("expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestVarianceBehavior(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name        string
		values      []float64
		expectZero  bool
	}{
		{"empty", []float64{}, true},
		{"single", []float64{5.0}, true},
		{"constant", []float64{3.0, 3.0, 3.0}, true},
		{"variable", []float64{1.0, 2.0, 3.0}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.variance(tc.values)
			if tc.expectZero && result != 0 {
				t.Errorf("expected 0, got %.2f", result)
			}
			if !tc.expectZero && result <= 0 {
				t.Error("expected positive variance")
			}
		})
	}
}

func TestMaxBehavior(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 5.0},
		{"normal", []float64{1.0, 5.0, 3.0}, 5.0},
		{"negative", []float64{-5.0, -1.0, -3.0}, -1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.max(tc.values)
			if result != tc.expected {
				t.Errorf("expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestMinBehavior(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 5.0},
		{"normal", []float64{1.0, 5.0, 3.0}, 1.0},
		{"negative", []float64{-5.0, -1.0, -3.0}, -5.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.min(tc.values)
			if result != tc.expected {
				t.Errorf("expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestFindPeaks(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	values := []float64{1.0, 3.0, 2.0, 5.0, 4.0, 6.0, 3.0}
	peaks := extractor.findPeaks(values)

	expectedPeaks := []float64{3.0, 5.0, 6.0}
	if len(peaks) != len(expectedPeaks) {
		t.Errorf("expected %d peaks, got %d", len(expectedPeaks), len(peaks))
	}
}

func TestFindValleys(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	values := []float64{3.0, 1.0, 2.0, -1.0, 0.0, 4.0, 2.0}
	valleys := extractor.findValleys(values)

	expectedValleys := []float64{1.0, -1.0, 0.0}
	if len(valleys) != len(expectedValleys) {
		t.Errorf("expected %d valleys, got %d", len(expectedValleys), len(valleys))
	}
}

func TestCalculatePositiveRatio(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"all positive", []float64{1.0, 2.0, 3.0}, 1.0},
		{"all negative", []float64{-1.0, -2.0, -3.0}, 0.0},
		{"mixed", []float64{1.0, -2.0, 3.0}, 2.0 / 3.0},
		{"with zero", []float64{1.0, 0.0, -1.0}, 1.0 / 3.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.calculatePositiveRatio(tc.values)
			if len(tc.values) == 0 && result != 0 {
				t.Errorf("expected 0 for empty, got %.2f", result)
			} else if len(tc.values) > 0 && math.Abs(result-tc.expected) > 0.01 {
				t.Errorf("expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestCountDirectionChanges(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name        string
		points      []model.MousePoint
		minChanges  int
	}{
		{
			name:        "empty",
			points:      []model.MousePoint{},
			minChanges:  0,
		},
		{
			name: "single point",
			points: []model.MousePoint{
				{X: 100, Y: 100, Timestamp: 0},
			},
			minChanges: 0,
		},
		{
			name: "straight line",
			points: []model.MousePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 0, Timestamp: 100},
				{X: 200, Y: 0, Timestamp: 200},
				{X: 300, Y: 0, Timestamp: 300},
			},
			minChanges: 0,
		},
		{
			name: "zigzag",
			points: []model.MousePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 100, Timestamp: 100},
				{X: 200, Y: 0, Timestamp: 200},
				{X: 300, Y: 100, Timestamp: 300},
			},
			minChanges: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			changes := extractor.countDirectionChanges(tc.points)
			if changes < tc.minChanges {
				t.Errorf("expected at least %d changes, got %d", tc.minChanges, changes)
			}
		})
	}
}

func TestCalculateEntropyBehavior(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name      string
		values    []float64
		bins      int
		expectZero bool
	}{
		{"empty", []float64{}, 10, true},
		{"single", []float64{5.0}, 10, true},
		{"uniform", []float64{1.0, 2.0, 3.0, 4.0, 5.0}, 5, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.calculateEntropy(tc.values, tc.bins)
			if tc.expectZero && result != 0 {
				t.Errorf("expected 0, got %.2f", result)
			}
			if !tc.expectZero && result < 0 {
				t.Error("expected non-negative entropy")
			}
		})
	}
}

func TestEstimateClusterCount(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name        string
		xValues     []float64
		yValues     []float64
		minClusters int
	}{
		{
			name:        "empty",
			xValues:     []float64{},
			yValues:     []float64{},
			minClusters: 1,
		},
		{
			name:        "single",
			xValues:     []float64{100},
			yValues:     []float64{100},
			minClusters: 1,
		},
		{
			name:        "clustered",
			xValues:     []float64{100, 105, 110, 200, 205, 210},
			yValues:     []float64{100, 105, 110, 200, 205, 210},
			minClusters: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			count := extractor.estimateClusterCount(tc.xValues, tc.yValues)
			if count < tc.minClusters {
				t.Errorf("expected at least %d clusters, got %d", tc.minClusters, count)
			}
		})
	}
}

func TestCalculateDensity(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name      string
		xValues   []float64
		yValues   []float64
		expectZero bool
	}{
		{"empty", []float64{}, []float64{}, true},
		{"single", []float64{100}, []float64{100}, true},
		{"distributed", []float64{0, 100, 200}, []float64{0, 100, 200}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			density := extractor.calculateDensity(tc.xValues, tc.yValues)
			if tc.expectZero && density != 0 {
				t.Errorf("expected 0 density, got %.6f", density)
			}
			if !tc.expectZero && density <= 0 {
				t.Error("expected positive density")
			}
		})
	}
}

func TestAnalyzeHumanLikelihood(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name      string
		features  *model.MouseBehaviorFeatures
		expectHuman bool
	}{
		{
			name:      "nil features",
			features:  nil,
			expectHuman: false,
		},
		{
			name: "high consistency (bot-like)",
			features: &model.MouseBehaviorFeatures{
				SpeedFeatures: model.MouseSpeedFeature{
					AverageSpeed: 1.0,
					SpeedStdDev:  0.05,
				},
				AccelerationFeatures: model.MouseAccelerationFeature{
					AccelerationStdDev: 0.005,
				},
				DoubleClickFeatures: model.DoubleClickFeature{
					FastDoubleClickRatio: 0.95,
				},
				LatencyFeatures: model.ClickLatencyFeature{
					FastClickRatio: 0.95,
					HesitationCount: 0,
				},
			},
			expectHuman: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isHuman, likelihood := extractor.AnalyzeHumanLikelihood(tc.features)
			if tc.features == nil {
				if isHuman != false {
					t.Error("expected false for nil features")
				}
				if likelihood != 0.0 {
					t.Error("expected 0 likelihood for nil features")
				}
			} else {
				t.Logf("isHuman=%v, likelihood=%.2f", isHuman, likelihood)
			}
		})
	}
}

func TestGenerateReportBehavior(t *testing.T) {
	extractor := NewMouseBehaviorExtractor()

	testCases := []struct {
		name     string
		features *model.MouseBehaviorFeatures
	}{
		{
			name:     "nil features",
			features: nil,
		},
		{
			name: "valid features",
			features: &model.MouseBehaviorFeatures{
				SpeedFeatures: model.MouseSpeedFeature{
					AverageSpeed:  1.5,
					MaxSpeed:      3.0,
					SpeedStdDev:   0.5,
					SpeedVariance: 0.25,
					ZeroSpeedCount: 0,
				},
				AccelerationFeatures: model.MouseAccelerationFeature{
					AverageAcceleration: 0.1,
					MaxAcceleration:     0.5,
					JerkAvg:            0.01,
					DirectionChanges:   5,
				},
				DistributionFeatures: model.ClickDistributionFeature{
					XMean:     200.0,
					YMean:     300.0,
					XStdDev:   50.0,
					YStdDev:   50.0,
					XEntropy:  2.5,
					YEntropy:  2.5,
					Clusters:  2,
				},
				DoubleClickFeatures: model.DoubleClickFeature{
					DoubleClickCount:    3,
					SingleClickCount:    10,
					AverageInterval:     200.0,
					FastDoubleClickRatio: 0.3,
				},
				LatencyFeatures: model.ClickLatencyFeature{
					AverageLatency:  150.0,
					MedianLatency:   140.0,
					HesitationCount: 2,
					FastClickRatio:  0.6,
				},
				OverallScore: 35.0,
				IsHumanLike:   true,
				Confidence:    0.85,
				AnomalyIndicators: []string{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			report := extractor.GenerateReport(tc.features)
			if len(report) == 0 {
				t.Error("expected non-empty report")
			}
			t.Logf("Report length: %d chars", len(report))
		})
	}
}

func TestModelFunctions(t *testing.T) {
	t.Run("MouseSpeedFeature.CalculateSpeedRange", func(t *testing.T) {
		feature := model.MouseSpeedFeature{
			MaxSpeed: 5.0,
			MinSpeed: 1.0,
		}
		range_ := feature.CalculateSpeedRange()
		if range_ != 4.0 {
			t.Errorf("expected 4.0, got %.2f", range_)
		}
	})

	t.Run("MouseSpeedFeature.CalculateSpeedConsistency", func(t *testing.T) {
		feature := model.MouseSpeedFeature{
			AverageSpeed: 2.0,
			SpeedStdDev:  0.2,
		}
		consistency := feature.CalculateSpeedConsistency()
		if consistency < 0 || consistency > 1 {
			t.Errorf("expected value between 0 and 1, got %.2f", consistency)
		}
	})

	t.Run("MouseAccelerationFeature.CalculateAccelerationConsistency", func(t *testing.T) {
		feature := model.MouseAccelerationFeature{
			AverageAcceleration:  0.1,
			AccelerationStdDev:  0.01,
		}
		consistency := feature.CalculateAccelerationConsistency()
		if consistency < 0 || consistency > 1 {
			t.Errorf("expected value between 0 and 1, got %.2f", consistency)
		}
	})

	t.Run("ClickDistributionFeature.CalculateDistanceFromCenter", func(t *testing.T) {
		feature := model.ClickDistributionFeature{
			CenterX: 100.0,
			CenterY: 100.0,
		}
		distance := feature.CalculateDistanceFromCenter(100, 100)
		if distance != 0 {
			t.Errorf("expected 0, got %.2f", distance)
		}

		distance = feature.CalculateDistanceFromCenter(200, 100)
		expected := 100.0
		if math.Abs(distance-expected) > 0.01 {
			t.Errorf("expected %.2f, got %.2f", expected, distance)
		}
	})

	t.Run("DoubleClickFeature.CalculateFastClickRatio", func(t *testing.T) {
		feature := model.DoubleClickFeature{
			FastDoubleClickRatio: 3,
			SingleClickCount:     5,
			TripleClickCount:     2,
		}
		ratio := feature.CalculateFastClickRatio()
		expected := 3.0 / 10.0
		if math.Abs(ratio-expected) > 0.01 {
			t.Errorf("expected %.2f, got %.2f", expected, ratio)
		}
	})

	t.Run("ClickLatencyFeature.CalculateHesitationRate", func(t *testing.T) {
		feature := model.ClickLatencyFeature{
			AverageLatency:  200.0,
			HesitationCount: 5,
		}
		rate := feature.CalculateHesitationRate()
		expected := 5.0 / 200.0
		if math.Abs(rate-expected) > 0.01 {
			t.Errorf("expected %.2f, got %.2f", expected, rate)
		}
	})

	t.Run("MouseBehaviorFeatures.AddAnomalyIndicator", func(t *testing.T) {
		features := &model.MouseBehaviorFeatures{
			AnomalyIndicators: []string{},
		}

		features.AddAnomalyIndicator("test")
		if len(features.AnomalyIndicators) != 1 {
			t.Error("expected 1 indicator")
		}

		features.AddAnomalyIndicator("test")
		if len(features.AnomalyIndicators) != 1 {
			t.Error("expected 1 indicator (no duplicates)")
		}

		features.AddAnomalyIndicator("test2")
		if len(features.AnomalyIndicators) != 2 {
			t.Error("expected 2 indicators")
		}
	})
}
