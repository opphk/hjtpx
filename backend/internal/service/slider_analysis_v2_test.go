package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliderAnalysisV2_AnalyzeTrajectory(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("basic_analysis", func(t *testing.T) {
		points := make([]SliderTrajectoryPoint, 50)
		for i := 0; i < 50; i++ {
			points[i] = SliderTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 20),
			}
		}

		result := analyzer.AnalyzeTrajectory(points)

		assert.NotNil(t, result)
		assert.NotNil(t, result.SpeedFeatures)
		assert.Greater(t, result.SpeedFeatures.Average, 0.0)
	})

	t.Run("insufficient_points", func(t *testing.T) {
		points := []SliderTrajectoryPoint{
			{X: 10, Y: 80, Timestamp: 0},
			{X: 20, Y: 80, Timestamp: 20},
		}

		result := analyzer.AnalyzeTrajectory(points)

		assert.NotNil(t, result)
	})
}

func TestSliderAnalysisV2_EnhancedFeatures(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("extract_enhanced_features", func(t *testing.T) {
		points := make([]EnhancedTrajectoryPoint, 30)
		for i := 0; i < 30; i++ {
			points[i] = EnhancedTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 30),
				Velocity:  250.0,
			}
		}

		features := analyzer.extractEnhancedFeatures(points)

		assert.NotNil(t, features)
		assert.Equal(t, 30, features.TotalPoints)
		assert.Greater(t, features.TotalDuration, int64(0))
		assert.Greater(t, features.TotalDistance, 0.0)
		assert.Greater(t, features.AverageVelocity, 0.0)
	})

	t.Run("human_likeness_calculation", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			PathEfficiency:     0.85,
			VelocityVariance:   0.3,
			MicroCorrections:   5,
			PauseCount:        2,
			JitterScore:       0.15,
			SmoothnessScore:   0.7,
			BacktrackCount:    1,
			Entropy:           3.0,
		}

		likeness := analyzer.calculateHumanLikeness(features)

		assert.Greater(t, likeness, 0.5)
		assert.Less(t, likeness, 1.0)
	})

	t.Run("bot_probability_calculation", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			PathEfficiency:     0.9999,
			VelocityVariance:    0.01,
			AverageVelocity:     500.0,
			PauseCount:         0,
			TotalDuration:       2000,
			MicroCorrections:    0,
			HumanLikenessScore:  0.2,
		}

		probability := analyzer.calculateBotProbability(features)

		assert.Greater(t, probability, 0.5)
	})
}

func TestSliderAnalysisV2_BotIndicators(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("detect_perfect_linear", func(t *testing.T) {
		points := make([]EnhancedTrajectoryPoint, 20)
		for i := 0; i < 20; i++ {
			points[i] = EnhancedTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 20),
				Velocity:  250.0,
			}
		}

		features := analyzer.extractEnhancedFeatures(points)
		indicators := analyzer.detectBotIndicators(points, features)

		assert.NotNil(t, indicators)
	})

	t.Run("detect_constant_speed", func(t *testing.T) {
		points := make([]EnhancedTrajectoryPoint, 30)
		for i := 0; i < 30; i++ {
			points[i] = EnhancedTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 20),
				Velocity:  250.0,
			}
		}

		features := analyzer.extractEnhancedFeatures(points)
		indicators := analyzer.detectBotIndicators(points, features)

		assert.NotNil(t, indicators)
		assert.True(t, indicators.ConstantSpeed)
	})

	t.Run("detect_mechanical_movement", func(t *testing.T) {
		points := make([]EnhancedTrajectoryPoint, 30)
		for i := 0; i < 30; i++ {
			points[i] = EnhancedTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 20),
				Velocity:  250.0,
			}
		}

		features := analyzer.extractEnhancedFeatures(points)
		indicators := analyzer.detectBotIndicators(points, features)

		assert.NotNil(t, indicators)
	})
}

func TestSliderAnalysisV2_PatternClassification(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("classify_perfect_linear", func(t *testing.T) {
		points := make([]EnhancedTrajectoryPoint, 20)
		for i := 0; i < 20; i++ {
			points[i] = EnhancedTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 20),
			}
		}

		features := analyzer.extractEnhancedFeatures(points)
		pattern := analyzer.classifyTrajectory(points, features)

		assert.NotNil(t, pattern)
		assert.Contains(t, []string{"perfect_linear", "near_linear"}, pattern.Type)
	})

	t.Run("classify_normal", func(t *testing.T) {
		points := make([]EnhancedTrajectoryPoint, 30)
		for i := 0; i < 30; i++ {
			points[i] = EnhancedTrajectoryPoint{
				X:         float64(10 + i*5 + (i%3-1)*2),
				Y:         80.0 + float64(i%5-2),
				Timestamp: int64(i * 30),
			}
		}

		features := analyzer.extractEnhancedFeatures(points)
		pattern := analyzer.classifyTrajectory(points, features)

		assert.NotNil(t, pattern)
	})
}

func TestSliderAnalysisV2_RiskAssessment(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("assess_high_risk", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			RiskScore:          0.9,
			BotProbability:     0.8,
			PathEfficiency:     0.9999,
			AverageVelocity:    2000.0,
			VelocityVariance:   0.05,
			TotalPauseDuration: 0,
			TotalDuration:      2000,
			FractalDimension:   1.1,
			Entropy:            1.5,
			HumanLikenessScore: 0.2,
		}

		riskLevel := analyzer.assessRiskLevel(features)

		assert.Equal(t, "high", riskLevel)
	})

	t.Run("assess_low_risk", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			RiskScore:          0.15,
			BotProbability:     0.1,
			PathEfficiency:     0.85,
			AverageVelocity:    300.0,
			VelocityVariance:   0.3,
			TotalPauseDuration: 100,
			TotalDuration:      2000,
			FractalDimension:   1.5,
			Entropy:            3.5,
			HumanLikenessScore: 0.8,
		}

		riskLevel := analyzer.assessRiskLevel(features)

		assert.Equal(t, "low", riskLevel)
	})
}

func TestSliderAnalysisV2_QualityAssessment(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("assess_excellent_quality", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			TotalPoints:      40,
			TotalDuration:    2000,
			TotalDistance:    200,
			PathEfficiency:   0.85,
			AverageVelocity:  400.0,
		}

		quality := analyzer.assessQuality(features)

		assert.Equal(t, "excellent", quality)
	})

	t.Run("assess_poor_quality", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			TotalPoints:      5,
			TotalDuration:    100,
			TotalDistance:    20,
			PathEfficiency:   0.4,
			AverageVelocity:  50.0,
		}

		quality := analyzer.assessQuality(features)

		assert.Equal(t, "poor", quality)
	})
}

func TestSliderAnalysisV2_ExtendedAnalysis(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("perform_extended_analysis", func(t *testing.T) {
		points := make([]SliderTrajectoryPoint, 30)
		for i := 0; i < 30; i++ {
			points[i] = SliderTrajectoryPoint{
				X:         float64(10 + i*5 + (i%3-1)*2),
				Y:         80.0 + float64(i%5-2),
				Timestamp: int64(i * 30),
			}
		}

		result := analyzer.PerformExtendedAnalysis(points)

		assert.NotNil(t, result)
		assert.NotNil(t, result.ExtendedAnalysisResult)
		assert.NotNil(t, result.EnhancedFeatures)
		assert.NotNil(t, result.BotIndicators)
		assert.NotNil(t, result.Pattern)
		assert.Contains(t, []string{"high", "medium", "low", "minimal"}, result.RiskLevel)
		assert.Contains(t, []string{"excellent", "good", "fair", "poor"}, result.Quality)
	})

	t.Run("extended_analysis_with_recommendations", func(t *testing.T) {
		points := make([]SliderTrajectoryPoint, 30)
		for i := 0; i < 30; i++ {
			points[i] = SliderTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 20),
			}
		}

		result := analyzer.PerformExtendedAnalysis(points)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Recommendations)
	})
}

func TestSliderAnalysisV2_Validation(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("validate_valid_trajectory", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			TotalPoints:  30,
			TotalDuration: 2000,
			TotalDistance: 200,
			MaxVelocity:  1500.0,
			QualityIssues: make([]string, 0),
		}

		isValid := analyzer.validateTrajectory(features)

		assert.True(t, isValid)
		assert.Empty(t, features.QualityIssues)
	})

	t.Run("validate_invalid_trajectory", func(t *testing.T) {
		features := &EnhancedTrajectoryFeatures{
			TotalPoints:   5,
			TotalDuration: 100,
			TotalDistance: 20,
			MaxVelocity:  6000.0,
			QualityIssues: make([]string, 0),
		}

		isValid := analyzer.validateTrajectory(features)

		assert.False(t, isValid)
		assert.NotEmpty(t, features.QualityIssues)
	})
}

func TestSliderAnalysisV2_MathHelpers(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("calculate_mean", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := analyzer.mean(values)
		assert.Equal(t, 3.0, mean)
	})

	t.Run("calculate_variance", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := analyzer.mean(values)
		variance := analyzer.variance(values, mean)
		assert.Greater(t, variance, 0.0)
	})

	t.Run("calculate_skewness", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 100.0}
		mean := analyzer.mean(values)
		skewness := analyzer.skewness(values, mean)
		assert.NotNil(t, skewness)
	})

	t.Run("calculate_kurtosis", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := analyzer.mean(values)
		kurtosis := analyzer.kurtosis(values, mean)
		assert.NotNil(t, kurtosis)
	})
}

func TestSliderAnalysisV2_EntropyCalculation(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("calculate_entropy", func(t *testing.T) {
		values := make([]float64, 100)
		for i := 0; i < 100; i++ {
			values[i] = float64(i % 10)
		}

		entropy := analyzer.calculateEntropy(values)

		assert.Greater(t, entropy, 0.0)
		assert.Less(t, entropy, 5.0)
	})

	t.Run("calculate_fractal_dimension", func(t *testing.T) {
		points := make([]EnhancedTrajectoryPoint, 30)
		for i := 0; i < 30; i++ {
			points[i] = EnhancedTrajectoryPoint{
				X:         float64(10 + i*5),
				Y:         80.0,
				Timestamp: int64(i * 20),
			}
		}

		dimension := analyzer.calculateFractalDimension(points)

		assert.GreaterOrEqual(t, dimension, 1.0)
		assert.LessOrEqual(t, dimension, 2.0)
	})
}

func TestSliderAnalysisV2_ThreadSafety(t *testing.T) {
	analyzer := NewSliderAnalysisV2()

	t.Run("concurrent_analysis", func(t *testing.T) {
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func() {
				points := make([]SliderTrajectoryPoint, 30)
				for j := 0; j < 30; j++ {
					points[j] = SliderTrajectoryPoint{
						X:         float64(10 + j*5),
						Y:         80.0,
						Timestamp: int64(j * 20),
					}
				}

				result := analyzer.AnalyzeTrajectory(points)
				assert.NotNil(t, result)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
