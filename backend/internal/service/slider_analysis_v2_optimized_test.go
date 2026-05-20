package service

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSliderAnalysisV2_OptimizedSmoothing(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("gaussian_smoothing", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		smoothed := analyzer.SmoothTrajectoryGaussian(points, 2.0)

		assert.NotNil(t, smoothed)
		assert.Equal(t, len(points), len(smoothed))

		for i := range smoothed {
			assert.Equal(t, points[i].Timestamp, smoothed[i].Timestamp)
		}

		originalVariance := calculateVarianceSimple(points)
		smoothedVariance := calculateVarianceSimple(smoothed)
		assert.Less(t, smoothedVariance, originalVariance*1.5)
	})

	t.Run("savitzky_golay_smoothing", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		smoothed := analyzer.SmoothTrajectorySavitzkyGolay(points, 5, 2)

		assert.NotNil(t, smoothed)
		assert.Equal(t, len(points), len(smoothed))
	})

	t.Run("exponential_smoothing", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		smoothed := analyzer.SmoothTrajectoryExponential(points, 0.3)

		assert.NotNil(t, smoothed)
		assert.Equal(t, len(points), len(smoothed))
	})

	t.Run("median_smoothing", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		smoothed := analyzer.SmoothTrajectoryMedian(points, 3)

		assert.NotNil(t, smoothed)
		assert.Equal(t, len(points), len(smoothed))
	})
}

func TestSliderAnalysisV2_EnhancedSpeedFeatures(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("extract_enhanced_features", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		features := analyzer.ExtractSpeedFeaturesEnhanced(points)

		assert.Greater(t, features.Average, 0.0)
		assert.GreaterOrEqual(t, features.Max, features.Min)
		assert.GreaterOrEqual(t, features.Max, features.Average)
		assert.GreaterOrEqual(t, features.Average, features.Min)
		assert.GreaterOrEqual(t, features.Range, 0.0)
		assert.GreaterOrEqual(t, features.Percentile75, features.Percentile25)
	})

	t.Run("speed_statistics", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		features := analyzer.ExtractSpeedFeaturesEnhanced(points)

		assert.NotEmpty(t, features.Trend)
		assert.True(t, features.CV >= 0)
	})
}

func TestSliderAnalysisV2_EnhancedJitterDetection(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("human_trajectory_jitter", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		jitter := analyzer.DetectJitterEnhanced(points)

		assert.GreaterOrEqual(t, jitter.TotalJitterScore, 0.0)
		assert.LessOrEqual(t, jitter.TotalJitterScore, 1.0)
	})

	t.Run("bot_trajectory_jitter", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		jitter := analyzer.DetectJitterEnhanced(points)

		assert.Less(t, jitter.TotalJitterScore, 0.2)
	})

	t.Run("jitter_features", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		jitter := analyzer.DetectJitterEnhanced(points)

		assert.GreaterOrEqual(t, jitter.DirectionJitter, 0.0)
		assert.GreaterOrEqual(t, jitter.AmplitudeJitter, 0.0)
		assert.GreaterOrEqual(t, jitter.SpeedJitter, 0.0)
	})
}

func TestSliderAnalysisV2_EnhancedAccelerationDetection(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("detect_acceleration_pattern", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		pattern := analyzer.DetectAccelerationPatternEnhanced(points)

		assert.NotEmpty(t, pattern.Type)
		assert.NotEmpty(t, pattern.Trend)
	})

	t.Run("acceleration_statistics", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		pattern := analyzer.DetectAccelerationPatternEnhanced(points)

		assert.GreaterOrEqual(t, pattern.PositiveRatio, 0.0)
		assert.LessOrEqual(t, pattern.PositiveRatio, 1.0)
		assert.GreaterOrEqual(t, pattern.NegativeRatio, 0.0)
		assert.LessOrEqual(t, pattern.NegativeRatio, 1.0)
	})
}

func TestSliderAnalysisV2_SpeedPatternDetection(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("detect_speed_pattern", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		pattern := analyzer.DetectSpeedPattern(points)

		assert.NotEmpty(t, pattern.Type)
	})

	t.Run("human_speed_pattern", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		pattern := analyzer.DetectSpeedPattern(points)

		assert.True(t, pattern.IsVariable || pattern.Type != "uniform")
	})

	t.Run("bot_speed_pattern", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		pattern := analyzer.DetectSpeedPattern(points)

		assert.True(t, pattern.Type == "uniform" || pattern.IsConsistent)
	})

	t.Run("speed_phases", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		pattern := analyzer.DetectSpeedPattern(points)

		if len(pattern.Phases) > 0 {
			for _, phase := range pattern.Phases {
				assert.Greater(t, phase.EndIndex, phase.StartIndex)
				assert.Greater(t, phase.AvgSpeed, 0.0)
				assert.GreaterOrEqual(t, phase.Duration, int64(0))
			}
		}
	})
}

func TestSliderAnalysisV2_AdvancedAnalysis(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("human_trajectory_analysis", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		result := analyzer.PerformAdvancedAnalysis(points)

		assert.NotNil(t, result)
		assert.NotNil(t, result.AdvancedFeatures)
		assert.Greater(t, result.AdvancedFeatures.HumanLikenessScore, 0.3)
	})

	t.Run("bot_trajectory_analysis", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		result := analyzer.PerformAdvancedAnalysis(points)

		assert.NotNil(t, result)
		assert.NotNil(t, result.AdvancedFeatures)
		assert.Less(t, result.AdvancedFeatures.BotProbability, 0.7)
	})

	t.Run("validation_result", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		result := analyzer.PerformAdvancedAnalysis(points)

		assert.NotNil(t, result.ValidationResult)
		assert.NotEmpty(t, result.ValidationResult.RiskLevel)
	})
}

func TestSliderAnalysisV2_TrajectoryComplexity(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("human_trajectory_complexity", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		complexity := analyzer.CalculateTrajectoryComplexity(points)

		assert.Greater(t, complexity, 0.0)
	})

	t.Run("bot_trajectory_complexity", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		complexity := analyzer.CalculateTrajectoryComplexity(points)

		assert.LessOrEqual(t, complexity, 1.0)
	})
}

func TestSliderAnalysisV2_HumanLikeness(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("human_likeness_calculation", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		smoothed := analyzer.SmoothTrajectoryGaussian(points, 2.0)
		speedFeatures := analyzer.ExtractSpeedFeaturesEnhanced(smoothed)
		jitterFeatures := analyzer.DetectJitterEnhanced(smoothed)
		pathEfficiency := analyzer.CalculatePathEfficiency(smoothed)

		likeness := analyzer.CalculateHumanLikenessEnhanced(speedFeatures, jitterFeatures, pathEfficiency)

		assert.Greater(t, likeness, 0.3)
		assert.LessOrEqual(t, likeness, 1.0)
	})
}

func TestSliderAnalysisV2_BotProbability(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("bot_probability_calculation", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		smoothed := analyzer.SmoothTrajectoryGaussian(points, 2.0)
		speedFeatures := analyzer.ExtractSpeedFeaturesEnhanced(smoothed)
		jitterFeatures := analyzer.DetectJitterEnhanced(smoothed)
		speedPattern := analyzer.DetectSpeedPattern(smoothed)

		probability := analyzer.CalculateBotProbabilityEnhanced(speedFeatures, jitterFeatures, speedPattern)

		assert.GreaterOrEqual(t, probability, 0.0)
		assert.LessOrEqual(t, probability, 1.0)
	})
}

func TestSliderAnalysisV2_Validation(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("human_trajectory_validation", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)
		smoothed := analyzer.SmoothTrajectoryGaussian(points, 2.0)
		speedFeatures := analyzer.ExtractSpeedFeaturesEnhanced(smoothed)
		jitterFeatures := analyzer.DetectJitterEnhanced(smoothed)
		speedPattern := analyzer.DetectSpeedPattern(smoothed)
		pathEfficiency := analyzer.CalculatePathEfficiency(smoothed)

		validation := analyzer.ValidateTrajectoryAdvanced(speedFeatures, jitterFeatures, speedPattern, pathEfficiency)

		assert.NotNil(t, validation)
		assert.NotEmpty(t, validation.RiskLevel)
	})

	t.Run("bot_trajectory_validation", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)
		smoothed := analyzer.SmoothTrajectoryGaussian(points, 2.0)
		speedFeatures := analyzer.ExtractSpeedFeaturesEnhanced(smoothed)
		jitterFeatures := analyzer.DetectJitterEnhanced(smoothed)
		speedPattern := analyzer.DetectSpeedPattern(smoothed)
		pathEfficiency := analyzer.CalculatePathEfficiency(smoothed)

		validation := analyzer.ValidateTrajectoryAdvanced(speedFeatures, jitterFeatures, speedPattern, pathEfficiency)

		assert.NotNil(t, validation)
	})
}

func TestGenerateTestTrajectory(t *testing.T) {
	t.Run("generate_human_trajectory", func(t *testing.T) {
		points := GenerateTestTrajectory(50, true)

		assert.Equal(t, 50, len(points))
		assert.Greater(t, points[len(points)-1].X, points[0].X)

		for i := 1; i < len(points); i++ {
			assert.GreaterOrEqual(t, points[i].Timestamp, points[i-1].Timestamp)
		}
	})

	t.Run("generate_bot_trajectory", func(t *testing.T) {
		points := GenerateTestTrajectory(50, false)

		assert.Equal(t, 50, len(points))
		assert.Greater(t, points[len(points)-1].X, points[0].X)
	})
}

func TestSliderAnalysisV2_StatisticalHelpers(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("mean_calculation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := analyzer.calculateMean(values)
		assert.InDelta(t, mean, 3.0, 0.001)
	})

	t.Run("variance_calculation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := analyzer.calculateMean(values)
		variance := analyzer.calculateVariance(values, mean)
		assert.Greater(t, variance, 0.0)
	})

	t.Run("median_calculation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		median := analyzer.calculateMedian(values)
		assert.InDelta(t, median, 3.0, 0.001)
	})

	t.Run("skewness_calculation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 100.0}
		mean := analyzer.calculateMean(values)
		stdDev := math.Sqrt(analyzer.calculateVariance(values, mean))
		skewness := analyzer.calculateSkewness(values, mean, stdDev)
		assert.Greater(t, skewness, 0.0)
	})

	t.Run("kurtosis_calculation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := analyzer.calculateMean(values)
		stdDev := math.Sqrt(analyzer.calculateVariance(values, mean))
		kurtosis := analyzer.calculateKurtosis(values, mean, stdDev)
		assert.False(t, math.IsNaN(kurtosis))
	})

	t.Run("percentile_calculation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
		p25 := analyzer.calculatePercentile(values, 25)
		p75 := analyzer.calculatePercentile(values, 75)

		assert.LessOrEqual(t, p25, p75)
	})
}

func BenchmarkSliderAnalysisV2_GaussianSmoothing(b *testing.B) {
	analyzer := NewSliderAnalysisV2()
	points := GenerateTestTrajectory(100, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.SmoothTrajectoryGaussian(points, 2.0)
	}
}

func BenchmarkSliderAnalysisV2_SavitzkyGolaySmoothing(b *testing.B) {
	analyzer := NewSliderAnalysisV2()
	points := GenerateTestTrajectory(100, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.SmoothTrajectorySavitzkyGolay(points, 5, 2)
	}
}

func BenchmarkSliderAnalysisV2_ExtractSpeedFeatures(b *testing.B) {
	analyzer := NewSliderAnalysisV2()
	points := GenerateTestTrajectory(100, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.ExtractSpeedFeaturesEnhanced(points)
	}
}

func BenchmarkSliderAnalysisV2_DetectJitter(b *testing.B) {
	analyzer := NewSliderAnalysisV2()
	points := GenerateTestTrajectory(100, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.DetectJitterEnhanced(points)
	}
}

func BenchmarkSliderAnalysisV2_PerformAdvancedAnalysis(b *testing.B) {
	analyzer := NewSliderAnalysisV2()
	points := GenerateTestTrajectory(100, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.PerformAdvancedAnalysis(points)
	}
}

func calculateVarianceSimple(points []SliderTrajectoryPoint) float64 {
	if len(points) == 0 {
		return 0
	}

	meanX := 0.0
	meanY := 0.0
	for _, p := range points {
		meanX += p.X
		meanY += p.Y
	}
	meanX /= float64(len(points))
	meanY /= float64(len(points))

	variance := 0.0
	for _, p := range points {
		variance += (p.X - meanX) * (p.X - meanX)
		variance += (p.Y - meanY) * (p.Y - meanY)
	}

	return variance / float64(len(points)*2)
}

func ExampleSliderAnalysisV2() {
	analyzer := NewSliderAnalysisV2()

	humanTrajectory := GenerateTestTrajectory(50, true)
	botTrajectory := GenerateTestTrajectory(50, false)

	humanResult := analyzer.PerformAdvancedAnalysis(humanTrajectory)
	botResult := analyzer.PerformAdvancedAnalysis(botTrajectory)

	fmt.Printf("Human trajectory:\n")
	fmt.Printf("  Human likeness: %.2f\n", humanResult.AdvancedFeatures.HumanLikenessScore)
	fmt.Printf("  Bot probability: %.2f\n", humanResult.AdvancedFeatures.BotProbability)
	fmt.Printf("  Risk level: %s\n", humanResult.ValidationResult.RiskLevel)

	fmt.Printf("\nBot trajectory:\n")
	fmt.Printf("  Human likeness: %.2f\n", botResult.AdvancedFeatures.HumanLikenessScore)
	fmt.Printf("  Bot probability: %.2f\n", botResult.AdvancedFeatures.BotProbability)
	fmt.Printf("  Risk level: %s\n", botResult.ValidationResult.RiskLevel)

	time.Sleep(100 * time.Millisecond)
}
