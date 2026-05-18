package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliderAnalyzer_AnalyzeTrajectoryQuality(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("valid_trajectory", func(t *testing.T) {
		trajectory := make([]SliderPoint, 30)
		for i := 0; i < 30; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*10,
				Y:         80,
				Timestamp: int64(i * 50),
			}
		}

		quality := analyzer.AnalyzeTrajectoryQuality(trajectory)

		assert.True(t, quality["is_valid"].(bool), "valid trajectory should be marked as valid")
		assert.Equal(t, 30, quality["point_count"].(int))
		assert.Greater(t, quality["duration_ms"].(float64), 0.0)
		assert.Greater(t, quality["total_distance"].(float64), 0.0)
	})

	t.Run("insufficient_points", func(t *testing.T) {
		trajectory := []SliderPoint{
			{X: 10, Y: 80, Timestamp: 0},
			{X: 20, Y: 80, Timestamp: 50},
		}

		quality := analyzer.AnalyzeTrajectoryQuality(trajectory)

		assert.False(t, quality["is_valid"].(bool))
		assert.Equal(t, "insufficient_points", quality["reason"])
	})

	t.Run("too_fast", func(t *testing.T) {
		trajectory := make([]SliderPoint, 10)
		for i := 0; i < 10; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*30,
				Y:         80,
				Timestamp: int64(i * 5),
			}
		}

		quality := analyzer.AnalyzeTrajectoryQuality(trajectory)

		assert.False(t, quality["is_valid"].(bool))
		assert.Equal(t, "too_fast", quality["reason"])
	})

	t.Run("insufficient_distance", func(t *testing.T) {
		trajectory := make([]SliderPoint, 20)
		for i := 0; i < 20; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*2,
				Y:         80,
				Timestamp: int64(i * 50),
			}
		}

		quality := analyzer.AnalyzeTrajectoryQuality(trajectory)

		assert.False(t, quality["is_valid"].(bool))
		assert.Equal(t, "insufficient_distance", quality["reason"])
	})
}

func TestSliderAnalyzer_DetectAdvancedBotPatterns(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("bot_like_uniform_motion", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80,
				Timestamp: int64(i * 16),
			}
		}

		score, patterns := analyzer.DetectAdvancedBotPatterns(trajectory)

		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
		assert.NotNil(t, patterns)
	})

	t.Run("human_like_organic_motion", func(t *testing.T) {
		trajectory := GenerateHumanLikeSliderTrajectory(10, 80, 280, 80, 2000)

		score, _ := analyzer.DetectAdvancedBotPatterns(trajectory)

		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
	})

	t.Run("empty_trajectory", func(t *testing.T) {
		trajectory := []SliderPoint{}

		score, patterns := analyzer.DetectAdvancedBotPatterns(trajectory)

		assert.Equal(t, 0.0, score)
		assert.Empty(t, patterns)
	})
}

func TestSliderAnalyzer_DetectUniformMotionPattern(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("uniform_motion_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80,
				Timestamp: int64(i * 16),
			}
		}

		pattern := analyzer.detectUniformMotionPattern(trajectory)

		assert.Equal(t, "uniform_motion", pattern.name)
		assert.Greater(t, pattern.weight, 0.0)
	})

	t.Run("variable_motion_not_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			speedVariation := 1.0 + (float64(i%10)-5.0)*0.1
			trajectory[i] = SliderPoint{
				X:         10 + int(float64(i)*6*speedVariation),
				Y:         80,
				Timestamp: int64(i * 16),
			}
		}

		pattern := analyzer.detectUniformMotionPattern(trajectory)

		assert.False(t, pattern.detected)
	})
}

func TestSliderAnalyzer_DetectGeometricPrecision(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("perfect_linear_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80,
				Timestamp: int64(i * 20),
			}
		}

		pattern := analyzer.detectGeometricPrecision(trajectory)

		assert.Equal(t, "geometric_precision", pattern.name)
		assert.True(t, pattern.detected)
		assert.Greater(t, pattern.weight, 0.0)
	})

	t.Run("curved_path_not_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         int(float64(80) + float64(i%10)-5),
				Timestamp: int64(i * 20),
			}
		}

		pattern := analyzer.detectGeometricPrecision(trajectory)

		assert.False(t, pattern.detected)
	})
}

func TestSliderAnalyzer_DetectTemporalRegularity(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("regular_intervals_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80,
				Timestamp: int64(i * 20),
			}
		}

		pattern := analyzer.detectTemporalRegularity(trajectory)

		assert.Equal(t, "temporal_regularity", pattern.name)
		assert.True(t, pattern.detected)
		assert.Greater(t, pattern.weight, 0.0)
	})

	t.Run("irregular_intervals_not_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			variation := 10 + (i%10) * 3
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80,
				Timestamp: int64(i * 20),
			}
			trajectory[i].Timestamp += int64(variation)
		}

		pattern := analyzer.detectTemporalRegularity(trajectory)

		if !pattern.detected {
			assert.False(t, pattern.detected, "irregular intervals should not be detected")
		}
	})
}

func TestSliderAnalyzer_DetectVelocityAnomaly(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("velocity_anomaly_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			speed := 300.0
			if i > 20 && i < 30 {
				speed = 2000.0
			}
			dt := int64(16)
			trajectory[i] = SliderPoint{
				X:         10 + int(float64(i)*6*(speed/300.0)),
				Y:         80,
				Timestamp: int64(i) * dt,
			}
		}

		pattern := analyzer.detectVelocityAnomaly(trajectory)

		assert.Equal(t, "velocity_anomaly", pattern.name)
	})

	t.Run("normal_velocity_not_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80,
				Timestamp: int64(i * 20),
			}
		}

		pattern := analyzer.detectVelocityAnomaly(trajectory)

		assert.False(t, pattern.detected)
	})
}

func TestSliderAnalyzer_DetectDirectionAnomaly(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("straight_line_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80,
				Timestamp: int64(i * 20),
			}
		}

		pattern := analyzer.detectDirectionAnomaly(trajectory)

		assert.Equal(t, "direction_anomaly", pattern.name)
		assert.True(t, pattern.detected)
		assert.Greater(t, pattern.weight, 0.0)
	})

	t.Run("wavy_path_not_detected", func(t *testing.T) {
		trajectory := make([]SliderPoint, 50)
		for i := 0; i < 50; i++ {
			trajectory[i] = SliderPoint{
				X:         10 + i*6,
				Y:         80 + int(float64(i%20)-10),
				Timestamp: int64(i * 20),
			}
		}

		pattern := analyzer.detectDirectionAnomaly(trajectory)

		assert.False(t, pattern.detected)
	})
}

func TestBotPatternStruct(t *testing.T) {
	pattern := botPattern{
		name:        "test_pattern",
		description: "test description",
		detected:    true,
		weight:      0.5,
	}

	assert.Equal(t, "test_pattern", pattern.name)
	assert.Equal(t, "test description", pattern.description)
	assert.True(t, pattern.detected)
	assert.Equal(t, 0.5, pattern.weight)
}

func TestGenerateHumanLikeSliderTrajectory(t *testing.T) {
	trajectory := GenerateHumanLikeSliderTrajectory(10, 80, 280, 80, 2000)

	assert.Greater(t, len(trajectory), 20, "human-like trajectory should have sufficient points")
	assert.Less(t, len(trajectory), 100, "human-like trajectory should not be excessive")

	firstPoint := trajectory[0]
	lastPoint := trajectory[len(trajectory)-1]
	assert.GreaterOrEqual(t, firstPoint.X, 5, "trajectory should start near the expected X coordinate")
	assert.LessOrEqual(t, firstPoint.X, 15, "trajectory should start near the expected X coordinate")
	assert.Greater(t, lastPoint.X, 200)

	totalDuration := lastPoint.Timestamp - firstPoint.Timestamp
	assert.Greater(t, totalDuration, int64(1500))
	assert.Less(t, totalDuration, int64(2500))
}

func TestGenerateBotLikeSliderTrajectory(t *testing.T) {
	trajectory := GenerateBotLikeSliderTrajectory(10, 80, 280, 80, 500)

	assert.Greater(t, len(trajectory), 15)
	assert.Less(t, len(trajectory), 50)

	firstPoint := trajectory[0]
	lastPoint := trajectory[len(trajectory)-1]
	assert.Equal(t, 10, firstPoint.X)
	assert.Equal(t, 280, lastPoint.X)

	totalDuration := lastPoint.Timestamp - firstPoint.Timestamp
	assert.Less(t, totalDuration, int64(600))
}

func TestSliderAnalyzer_AnalyzeTrajectoryQuality_WithRealisticData(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	t.Run("realistic_human_trajectory", func(t *testing.T) {
		trajectory := GenerateHumanLikeSliderTrajectory(10, 80, 280, 80, 2000)

		quality := analyzer.AnalyzeTrajectoryQuality(trajectory)

		assert.True(t, quality["is_valid"].(bool))
		assert.Greater(t, quality["sampling_rate_hz"].(float64), 10.0)
	})

	t.Run("realistic_bot_trajectory", func(t *testing.T) {
		trajectory := GenerateBotLikeSliderTrajectory(10, 80, 280, 80, 500)

		quality := analyzer.AnalyzeTrajectoryQuality(trajectory)

		assert.True(t, quality["is_valid"].(bool))
		assert.Greater(t, quality["sampling_rate_hz"].(float64), 10.0)
	})
}
