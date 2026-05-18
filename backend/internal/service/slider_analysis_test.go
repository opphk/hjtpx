package service

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func generateTestTrajectory(points int, startX, startY int, withJitter bool) []SliderPoint {
	trajectory := make([]SliderPoint, points)
	trajectory[0] = SliderPoint{X: startX, Y: startY, Timestamp: 0}

	prevX := startX
	prevY := startY

	for i := 1; i < points; i++ {
		dt := int64(10 + (i % 5))
		trajectory[i].Timestamp = trajectory[i-1].Timestamp + dt

		dx := 10 + (i % 3)
		dy := 0
		if i%5 == 0 {
			dy = 2
		} else if i%7 == 0 {
			dy = -2
		}

		if withJitter {
			dx += i % 3
			dy += i % 2
		}

		prevX += dx
		prevY += dy
		trajectory[i].X = prevX
		trajectory[i].Y = prevY
	}

	return trajectory
}

func generateHumanLikeTrajectory(points int) []SliderPoint {
	trajectory := make([]SliderPoint, points)
	trajectory[0] = SliderPoint{X: 100, Y: 200, Timestamp: 0}

	rng := rand.New(rand.NewSource(42))
	prevX := 100
	prevY := 200

	for i := 1; i < points; i++ {
		trajectory[i].Timestamp = trajectory[i-1].Timestamp + int64(10+rng.Intn(20))

		baseDx := 8 + rng.Intn(5)
		baseDy := rng.Intn(5) - 2

		jitter := rng.Intn(3) - 1
		prevX += baseDx + jitter
		prevY += baseDy + jitter

		trajectory[i].X = prevX
		trajectory[i].Y = prevY
	}

	return trajectory
}

func generateRobotTrajectory(points int) []SliderPoint {
	trajectory := make([]SliderPoint, points)
	trajectory[0] = SliderPoint{X: 100, Y: 200, Timestamp: 0}

	for i := 1; i < points; i++ {
		trajectory[i].Timestamp = trajectory[i-1].Timestamp + 10

		dx := 10
		dy := 0

		trajectory[i].X = trajectory[i-1].X + dx
		trajectory[i].Y = trajectory[i-1].Y + dy
	}

	return trajectory
}

func generateBacktrackTrajectory(points int) []SliderPoint {
	trajectory := make([]SliderPoint, points)

	for i := range trajectory {
		trajectory[i].Timestamp = int64(i * 15)
	}

	trajectory[0] = SliderPoint{X: 100, Y: 200, Timestamp: 0}

	for i := 1; i < points/2; i++ {
		trajectory[i].X = trajectory[i-1].X + 15
		trajectory[i].Y = trajectory[i-1].Y
	}

	for i := points / 2; i < points; i++ {
		trajectory[i].X = trajectory[i-1].X - 15
		trajectory[i].Y = trajectory[i-1].Y
	}

	return trajectory
}

func generatePauseTrajectory(points int) []SliderPoint {
	trajectory := make([]SliderPoint, points)

	for i := range trajectory {
		if i > 0 && i%10 == 0 {
			trajectory[i].Timestamp = trajectory[i-1].Timestamp + 500
		} else {
			trajectory[i].Timestamp = trajectory[i-1].Timestamp + 15
		}

		trajectory[i].X = 100 + i*10
		trajectory[i].Y = 200
	}

	return trajectory
}

func TestSliderFeatureExtractor_ExtractFeatures(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name           string
		trajectory     []SliderPoint
		targetPosition int
		validateFunc   func(*testing.T, *SliderFeatures)
	}{
		{
			name:           "Basic trajectory",
			trajectory:     generateTestTrajectory(50, 100, 200, false),
			targetPosition: 400,
			validateFunc: func(t *testing.T, features *SliderFeatures) {
				if features.TotalDistance <= 0 {
					t.Error("TotalDistance should be positive")
				}
				if features.DirectDistance <= 0 {
					t.Error("DirectDistance should be positive")
				}
				if features.PathEfficiency <= 0 || features.PathEfficiency > 1 {
					t.Error("PathEfficiency should be between 0 and 1")
				}
			},
		},
		{
			name:           "Human-like trajectory",
			trajectory:     generateHumanLikeTrajectory(100),
			targetPosition: 800,
			validateFunc: func(t *testing.T, features *SliderFeatures) {
				if features.SmoothnessScore < 0 {
					t.Error("SmoothnessScore should be non-negative")
				}
				if features.HumanLikenessScore < 0 || features.HumanLikenessScore > 1 {
					t.Error("HumanLikenessScore should be between 0 and 1")
				}
			},
		},
		{
			name:           "Robot trajectory",
			trajectory:     generateRobotTrajectory(50),
			targetPosition: 400,
			validateFunc: func(t *testing.T, features *SliderFeatures) {
				if features.SpeedVariance >= 0.1 {
					t.Logf("Robot trajectory speed variance: %f (expected low variance)", features.SpeedVariance)
				}
			},
		},
		{
			name:           "Short trajectory",
			trajectory:     generateTestTrajectory(2, 100, 200, false),
			targetPosition: 110,
			validateFunc: func(t *testing.T, features *SliderFeatures) {
				if features.TotalDistance == 0 && features.DirectDistance == 0 {
					t.Log("Short trajectory handled correctly")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sliderTraj := &SliderTrajectory{Points: tc.trajectory}
			features := extractor.ExtractFeatures(tc.trajectory, tc.targetPosition, sliderTraj)

			if features == nil {
				t.Fatal("ExtractFeatures returned nil")
			}

			tc.validateFunc(t, features)
		})
	}
}

func TestSliderFeatureExtractor_EnhancedSmoothnessMetrics(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
		validateFunc func(*testing.T, *SmoothnessMetrics)
	}{
		{
			name:       "Human-like smooth trajectory",
			trajectory: generateHumanLikeTrajectory(100),
			validateFunc: func(t *testing.T, metrics *SmoothnessMetrics) {
				if metrics.PrimarySmoothness < 0 || metrics.PrimarySmoothness > 1 {
					t.Errorf("PrimarySmoothness out of range: %f", metrics.PrimarySmoothness)
				}
				if metrics.SecondarySmoothness < 0 || metrics.SecondarySmoothness > 1 {
					t.Errorf("SecondarySmoothness out of range: %f", metrics.SecondarySmoothness)
				}
				if metrics.JerkScore < 0 || metrics.JerkScore > 1 {
					t.Errorf("JerkScore out of range: %f", metrics.JerkScore)
				}
				if metrics.CurveContinuity < 0 || metrics.CurveContinuity > 1 {
					t.Errorf("CurveContinuity out of range: %f", metrics.CurveContinuity)
				}
				if len(metrics.SegmentSmoothness) != 5 {
					t.Errorf("Expected 5 segment smoothness values, got %d", len(metrics.SegmentSmoothness))
				}
			},
		},
		{
			name:       "Robot trajectory with low smoothness",
			trajectory: generateRobotTrajectory(100),
			validateFunc: func(t *testing.T, metrics *SmoothnessMetrics) {
				if metrics.PrimarySmoothness < 0.5 {
					t.Logf("Robot trajectory has low primary smoothness: %f", metrics.PrimarySmoothness)
				}
				if metrics.TransitionCount != 0 {
					t.Logf("Robot trajectory transition count: %d", metrics.TransitionCount)
				}
			},
		},
		{
			name:       "Short trajectory",
			trajectory: generateTestTrajectory(3, 100, 200, false),
			validateFunc: func(t *testing.T, metrics *SmoothnessMetrics) {
				if metrics.PrimarySmoothness == 0 && metrics.SecondarySmoothness == 0 {
					t.Log("Short trajectory handled correctly (returns zero metrics)")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := extractor.extractEnhancedSmoothnessMetrics(tc.trajectory)

			if metrics == nil {
				t.Fatal("extractEnhancedSmoothnessMetrics returned nil")
			}

			tc.validateFunc(t, metrics)
		})
	}
}

func TestSliderFeatureExtractor_EnhancedAccelerationMetrics(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
		validateFunc func(*testing.T, *EnhancedAccelerationMetrics)
	}{
		{
			name:       "Normal human trajectory",
			trajectory: generateHumanLikeTrajectory(100),
			validateFunc: func(t *testing.T, metrics *EnhancedAccelerationMetrics) {
				if metrics.JerkMean < 0 {
					t.Errorf("JerkMean should be non-negative: %f", metrics.JerkMean)
				}
				if metrics.PositiveAccelRatio < 0 || metrics.PositiveAccelRatio > 1 {
					t.Errorf("PositiveAccelRatio out of range: %f", metrics.PositiveAccelRatio)
				}
				validPatterns := map[string]bool{
					"accelerating": true, "decelerating": true,
					"variable": true, "constant": true,
					"late_acceleration": true, "early_deceleration": true,
					"unknown": true,
				}
				if !validPatterns[metrics.AccelPattern] {
					t.Errorf("Invalid acceleration pattern: %s", metrics.AccelPattern)
				}
				if metrics.AccelerationEntropy < 0 {
					t.Errorf("AccelerationEntropy should be non-negative: %f", metrics.AccelerationEntropy)
				}
			},
		},
		{
			name:       "Constant speed robot trajectory",
			trajectory: generateRobotTrajectory(100),
			validateFunc: func(t *testing.T, metrics *EnhancedAccelerationMetrics) {
				if metrics.AccelPattern == "constant" {
					t.Log("Robot trajectory correctly identified as constant acceleration")
				}
			},
		},
		{
			name:       "Short trajectory",
			trajectory: generateTestTrajectory(2, 100, 200, false),
			validateFunc: func(t *testing.T, metrics *EnhancedAccelerationMetrics) {
				if metrics.MeanAcceleration == 0 {
					t.Log("Short trajectory returns zero acceleration metrics")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := extractor.extractEnhancedAccelerationMetrics(tc.trajectory)

			if metrics == nil {
				t.Fatal("extractEnhancedAccelerationMetrics returned nil")
			}

			tc.validateFunc(t, metrics)
		})
	}
}

func TestSliderFeatureExtractor_EnhancedCurvatureMetrics(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
		validateFunc func(*testing.T, *EnhancedCurvatureMetrics)
	}{
		{
			name:       "Curved human trajectory",
			trajectory: generateHumanLikeTrajectory(100),
			validateFunc: func(t *testing.T, metrics *EnhancedCurvatureMetrics) {
				if metrics.MeanCurvature < 0 {
					t.Errorf("MeanCurvature should be non-negative: %f", metrics.MeanCurvature)
				}
				if metrics.StdCurvature < 0 {
					t.Errorf("StdCurvature should be non-negative: %f", metrics.StdCurvature)
				}
				validDirections := map[string]bool{
					"straight": true, "mostly_left": true,
					"mostly_right": true, "mixed": true,
				}
				if !validDirections[metrics.TurnDirection] {
					t.Errorf("Invalid turn direction: %s", metrics.TurnDirection)
				}
				if metrics.CurvatureEntropy < 0 {
					t.Errorf("CurvatureEntropy should be non-negative: %f", metrics.CurvatureEntropy)
				}
			},
		},
		{
			name:       "Straight robot trajectory",
			trajectory: generateRobotTrajectory(100),
			validateFunc: func(t *testing.T, metrics *EnhancedCurvatureMetrics) {
				if metrics.TurningPoints == 0 {
					t.Log("Robot trajectory correctly has no turning points")
				}
				if metrics.TurnDirection == "straight" {
					t.Log("Robot trajectory correctly identified as straight")
				}
			},
		},
		{
			name:       "Backtrack trajectory",
			trajectory: generateBacktrackTrajectory(50),
			validateFunc: func(t *testing.T, metrics *EnhancedCurvatureMetrics) {
				if metrics.CurvatureBursts > 0 {
					t.Logf("Backtrack trajectory has %d curvature bursts", metrics.CurvatureBursts)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			curvatures := extractor.extractCurvatures(tc.trajectory)
			metrics := extractor.extractEnhancedCurvatureMetrics(tc.trajectory, curvatures)

			if metrics == nil {
				t.Fatal("extractEnhancedCurvatureMetrics returned nil")
			}

			tc.validateFunc(t, metrics)
		})
	}
}

func TestSliderFeatureExtractor_EnhancedJitterMetrics(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
		validateFunc func(*testing.T, *EnhancedJitterMetrics)
	}{
		{
			name:       "Human trajectory with natural jitter",
			trajectory: generateHumanLikeTrajectory(100),
			validateFunc: func(t *testing.T, metrics *EnhancedJitterMetrics) {
				if metrics.TotalJitter < 0 {
					t.Errorf("TotalJitter should be non-negative: %f", metrics.TotalJitter)
				}
				if metrics.JitterEntropy < 0 {
					t.Errorf("JitterEntropy should be non-negative: %f", metrics.JitterEntropy)
				}
				if metrics.JitterPeriodicity < 0 || metrics.JitterPeriodicity > 1 {
					t.Errorf("JitterPeriodicity out of range: %f", metrics.JitterPeriodicity)
				}
				if metrics.MicroJitterRatio < 0 || metrics.MicroJitterRatio > 1 {
					t.Errorf("MicroJitterRatio out of range: %f", metrics.MicroJitterRatio)
				}
				if metrics.JitterConsistency < 0 || metrics.JitterConsistency > 1 {
					t.Errorf("JitterConsistency out of range: %f", metrics.JitterConsistency)
				}
			},
		},
		{
			name:       "Robot trajectory with minimal jitter",
			trajectory: generateRobotTrajectory(100),
			validateFunc: func(t *testing.T, metrics *EnhancedJitterMetrics) {
				if metrics.TotalJitter < 1.0 {
					t.Logf("Robot trajectory has minimal jitter: %f", metrics.TotalJitter)
				}
			},
		},
		{
			name:       "Jittery trajectory",
			trajectory: generateTestTrajectory(100, 100, 200, true),
			validateFunc: func(t *testing.T, metrics *EnhancedJitterMetrics) {
				if metrics.TotalJitter > 0 {
					t.Logf("Jittery trajectory has total jitter: %f", metrics.TotalJitter)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := extractor.extractEnhancedJitterMetrics(tc.trajectory)

			if metrics == nil {
				t.Fatal("extractEnhancedJitterMetrics returned nil")
			}

			tc.validateFunc(t, metrics)
		})
	}
}

func TestUnifiedRiskScorer_CalculateRiskScore(t *testing.T) {
	scorer := NewUnifiedRiskScorer()

	testCases := []struct {
		name     string
		features *SliderFeatures
		validateFunc func(*testing.T, float64)
	}{
		{
			name:     "Nil features",
			features: nil,
			validateFunc: func(t *testing.T, score float64) {
				if score != 0.5 {
					t.Errorf("Expected 0.5 for nil features, got %f", score)
				}
			},
		},
		{
			name: "Human-like features",
			features: &SliderFeatures{
				PathEfficiency:      0.95,
				SpeedConsistency:    0.9,
				SmoothnessScore:     0.92,
				HumanLikenessScore:  0.88,
				AverageSpeed:        1200,
				SpeedVariance:       0.15,
				MicroCorrections:    5,
				PauseCount:          3,
				BacktrackCount:      2,
				FractalDimension:    1.2,
				TrajectoryEntropy:   2.5,
				JitterScore:         0.03,
				CurvatureVariance:   0.05,
				TotalDuration:       2500,
				SmoothnessMetrics: &SmoothnessMetrics{
					PrimarySmoothness:  0.9,
					SecondarySmoothness: 0.85,
					JerkScore:         0.15,
					CurveContinuity:   0.9,
					IsSmoothTransition: true,
					TransitionCount:  3,
				},
				EnhancedAcceleration: &EnhancedAccelerationMetrics{
					MeanAcceleration:   0.5,
					StdAcceleration:    0.3,
					MaxAcceleration:    1.5,
					MinAcceleration:    -1.0,
					JerkMean:           0.2,
					JerkStd:           0.1,
					JerkMax:           0.5,
					PositiveAccelRatio: 0.6,
					AccelerationEntropy: 2.5,
					SustainedAccelTime: 0.3,
					DecelPhaseRatio:   0.4,
					AccelPattern:     "variable",
				},
				EnhancedCurvature: &EnhancedCurvatureMetrics{
					MeanCurvature:    0.1,
					StdCurvature:    0.05,
					MaxCurvature:    0.3,
					CurvatureEntropy: 2.0,
					CurvaturePeaks:  []int{10, 20, 30},
					TurningPoints:   5,
					TurnDirection:   "mixed",
					CurvatureBursts: 3,
					CurvatureTrend:  0.1,
				},
				EnhancedJitter: &EnhancedJitterMetrics{
					HorizontalJitter: 1.2,
					VerticalJitter:   0.8,
					TotalJitter:      1.5,
					JitterEntropy:    2.0,
					JitterPeriodicity: 0.3,
					IsMicroJitter:    false,
					MicroJitterRatio: 0.4,
					JitterConsistency: 0.7,
				},
			},
			validateFunc: func(t *testing.T, score float64) {
				if score < 0 || score > 1 {
					t.Errorf("Score should be between 0 and 1, got %f", score)
				}
				if score < 0.7 {
					t.Logf("Human-like features got risk score: %f (expected high human score)", score)
				}
			},
		},
		{
			name: "Robot-like features",
			features: &SliderFeatures{
				PathEfficiency:      0.99,
				SpeedConsistency:    0.99,
				SmoothnessScore:     0.99,
				HumanLikenessScore:  0.1,
				AverageSpeed:        2000,
				SpeedVariance:       0.001,
				MicroCorrections:    0,
				PauseCount:          0,
				BacktrackCount:      0,
				FractalDimension:    1.0,
				TrajectoryEntropy:   1.0,
				JitterScore:         0.001,
				CurvatureVariance:   0.0001,
				TotalDuration:       500,
				SmoothnessMetrics: &SmoothnessMetrics{
					PrimarySmoothness:  0.99,
					SecondarySmoothness: 0.99,
					JerkScore:         0.01,
					CurveContinuity:   1.0,
					IsSmoothTransition: true,
					TransitionCount:  0,
				},
				EnhancedAcceleration: &EnhancedAccelerationMetrics{
					MeanAcceleration:   0.0,
					StdAcceleration:    0.0,
					MaxAcceleration:    0.0,
					MinAcceleration:    0.0,
					JerkMean:           0.0,
					JerkStd:           0.0,
					JerkMax:           0.0,
					PositiveAccelRatio: 0.5,
					AccelerationEntropy: 0.0,
					SustainedAccelTime: 0.0,
					DecelPhaseRatio:   0.5,
					AccelPattern:     "constant",
				},
				EnhancedCurvature: &EnhancedCurvatureMetrics{
					MeanCurvature:    0.0,
					StdCurvature:    0.0,
					MaxCurvature:    0.0,
					CurvatureEntropy: 0.0,
					CurvaturePeaks:  []int{},
					TurningPoints:   0,
					TurnDirection:   "straight",
					CurvatureBursts: 0,
					CurvatureTrend:  0.0,
				},
				EnhancedJitter: &EnhancedJitterMetrics{
					HorizontalJitter: 0.1,
					VerticalJitter:   0.1,
					TotalJitter:      0.15,
					JitterEntropy:    0.0,
					JitterPeriodicity: 0.0,
					IsMicroJitter:    true,
					MicroJitterRatio: 1.0,
					JitterConsistency: 1.0,
				},
			},
			validateFunc: func(t *testing.T, score float64) {
				if score < 0 || score > 1 {
					t.Errorf("Score should be between 0 and 1, got %f", score)
				}
				if score > 0.3 {
					t.Logf("Robot-like features got risk score: %f (expected low human score)", score)
				}
			},
		},
		{
			name: "Partial features (no enhanced metrics)",
			features: &SliderFeatures{
				PathEfficiency:      0.85,
				SpeedConsistency:    0.75,
				SmoothnessScore:     0.7,
				HumanLikenessScore:  0.5,
				AverageSpeed:        1000,
				SpeedVariance:       0.2,
				MicroCorrections:    3,
				PauseCount:          2,
				BacktrackCount:      1,
				FractalDimension:    1.4,
				TrajectoryEntropy:   3.0,
				JitterScore:         0.05,
				CurvatureVariance:   0.1,
				TotalDuration:       2000,
			},
			validateFunc: func(t *testing.T, score float64) {
				if score < 0 || score > 1 {
					t.Errorf("Score should be between 0 and 1, got %f", score)
				}
				t.Logf("Partial features risk score: %f", score)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := scorer.CalculateRiskScore(tc.features)
			tc.validateFunc(t, score)
		})
	}
}

func TestUnifiedRiskScorer_CategoryScores(t *testing.T) {
	scorer := NewUnifiedRiskScorer()

	features := &SliderFeatures{
		PathEfficiency:      0.9,
		SpeedConsistency:    0.85,
		SmoothnessScore:     0.88,
		HumanLikenessScore:  0.8,
		AverageSpeed:        1000,
		SpeedVariance:       0.1,
		FractalDimension:    1.15,
		TrajectoryEntropy:   2.0,
		JitterScore:         0.02,
		MicroCorrections:    4,
		PauseCount:          2,
		BacktrackCount:      1,
		CurvatureVariance:   0.05,
		TotalDuration:       2500,
		SmoothnessMetrics: &SmoothnessMetrics{
			PrimarySmoothness:  0.85,
			SecondarySmoothness: 0.8,
			JerkScore:         0.1,
			CurveContinuity:   0.85,
			IsSmoothTransition: true,
			TransitionCount:  2,
		},
		EnhancedAcceleration: &EnhancedAccelerationMetrics{
			MeanAcceleration:    0.3,
			StdAcceleration:    0.2,
			MaxAcceleration:    1.0,
			MinAcceleration:    -0.8,
			JerkMean:            0.15,
			JerkStd:            0.1,
			JerkMax:            0.4,
			PositiveAccelRatio: 0.55,
			AccelerationEntropy: 2.2,
			SustainedAccelTime: 0.2,
			DecelPhaseRatio:    0.45,
			AccelPattern:      "variable",
		},
		EnhancedCurvature: &EnhancedCurvatureMetrics{
			MeanCurvature:    0.08,
			StdCurvature:    0.04,
			MaxCurvature:    0.25,
			CurvatureEntropy: 1.8,
			CurvaturePeaks:  []int{8, 18},
			TurningPoints:   4,
			TurnDirection:   "mixed",
			CurvatureBursts: 2,
			CurvatureTrend:  0.05,
		},
		EnhancedJitter: &EnhancedJitterMetrics{
			HorizontalJitter: 1.0,
			VerticalJitter:   0.7,
			TotalJitter:      1.2,
			JitterEntropy:    1.8,
			JitterPeriodicity: 0.2,
			IsMicroJitter:    false,
			MicroJitterRatio: 0.3,
			JitterConsistency: 0.65,
		},
	}

	t.Run("Trajectory score", func(t *testing.T) {
		score := scorer.calculateTrajectoryScore(features)
		if score < 0 || score > 1 {
			t.Errorf("Trajectory score out of range: %f", score)
		}
		t.Logf("Trajectory score: %f", score)
	})

	t.Run("Velocity score", func(t *testing.T) {
		score := scorer.calculateVelocityScore(features)
		if score < 0 || score > 1 {
			t.Errorf("Velocity score out of range: %f", score)
		}
		t.Logf("Velocity score: %f", score)
	})

	t.Run("Smoothness score", func(t *testing.T) {
		score := scorer.calculateSmoothnessScoreFromFeatures(features)
		if score < 0 || score > 1 {
			t.Errorf("Smoothness score out of range: %f", score)
		}
		t.Logf("Smoothness score: %f", score)
	})

	t.Run("Anomaly score", func(t *testing.T) {
		score := scorer.calculateAnomalyScore(features)
		if score < 0 || score > 1 {
			t.Errorf("Anomaly score out of range: %f", score)
		}
		t.Logf("Anomaly score: %f", score)
	})

	t.Run("Has enhanced metrics", func(t *testing.T) {
		hasMetrics := scorer.hasEnhancedMetrics(features)
		if !hasMetrics {
			t.Error("Expected enhanced metrics to be present")
		}
	})
}

func TestCalculatePrimarySmoothness(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
		minExpected float64
		maxExpected float64
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
			minExpected: 0.3,
			maxExpected: 1.0,
		},
		{
			name:       "Robot trajectory",
			trajectory: generateRobotTrajectory(100),
			minExpected: 0.5,
			maxExpected: 1.0,
		},
		{
			name:       "Backtrack trajectory",
			trajectory: generateBacktrackTrajectory(50),
			minExpected: 0.0,
			maxExpected: 1.0,
		},
		{
			name:       "Short trajectory",
			trajectory: generateTestTrajectory(2, 100, 200, false),
			minExpected: 0.0,
			maxExpected: 1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			smoothness := extractor.calculatePrimarySmoothness(tc.trajectory)
			if smoothness < tc.minExpected || smoothness > tc.maxExpected {
				t.Errorf("Smoothness %f out of expected range [%f, %f]",
					smoothness, tc.minExpected, tc.maxExpected)
			}
		})
	}
}

func TestCalculateSecondarySmoothness(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
		},
		{
			name:       "Robot trajectory",
			trajectory: generateRobotTrajectory(100),
		},
		{
			name:       "Pause trajectory",
			trajectory: generatePauseTrajectory(50),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			smoothness := extractor.calculateSecondarySmoothness(tc.trajectory)
			if smoothness < 0 || smoothness > 1 {
				t.Errorf("SecondarySmoothness out of range: %f", smoothness)
			}
		})
	}
}

func TestCalculateJerkScore(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
		},
		{
			name:       "Robot trajectory",
			trajectory: generateRobotTrajectory(100),
		},
		{
			name:       "Jittery trajectory",
			trajectory: generateTestTrajectory(100, 100, 200, true),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jerkScore := extractor.calculateJerkScore(tc.trajectory)
			if jerkScore < 0 || jerkScore > 1 {
				t.Errorf("JerkScore out of range: %f", jerkScore)
			}
		})
	}
}

func TestCalculateCurveContinuity(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
		minExpected float64
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
			minExpected: 0.5,
		},
		{
			name:       "Robot trajectory",
			trajectory: generateRobotTrajectory(100),
			minExpected: 0.9,
		},
		{
			name:       "Short trajectory",
			trajectory: generateTestTrajectory(2, 100, 200, false),
			minExpected: 1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			continuity := extractor.calculateCurveContinuity(tc.trajectory)
			if continuity < tc.minExpected {
				t.Errorf("CurveContinuity %f below minimum %f", continuity, tc.minExpected)
			}
		})
	}
}

func TestDetectSmoothTransitions(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
		},
		{
			name:       "Robot trajectory",
			trajectory: generateRobotTrajectory(100),
		},
		{
			name:       "Short trajectory",
			trajectory: generateTestTrajectory(2, 100, 200, false),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.detectSmoothTransitions(tc.trajectory)
			if result.count < 0 {
				t.Error("Transition count should be non-negative")
			}
			t.Logf("Smooth transitions: isSmooth=%v, count=%d", result.isSmooth, result.count)
		})
	}
}

func TestCalculateJitterEntropy(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name        string
		xDeviations []float64
		yDeviations []float64
	}{
		{
			name:        "Normal deviations",
			xDeviations: []float64{1.0, 2.0, 1.5, 2.5, 1.8},
			yDeviations: []float64{0.8, 1.2, 1.0, 1.5, 0.9},
		},
		{
			name:        "Empty deviations",
			xDeviations: []float64{},
			yDeviations: []float64{},
		},
		{
			name:        "Constant deviations",
			xDeviations: []float64{1.0, 1.0, 1.0, 1.0},
			yDeviations: []float64{1.0, 1.0, 1.0, 1.0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entropy := extractor.calculateJitterEntropy(tc.xDeviations, tc.yDeviations)
			if entropy < 0 {
				t.Error("Entropy should be non-negative")
			}
			t.Logf("Jitter entropy: %f", entropy)
		})
	}
}

func TestCalculateJitterPeriodicity(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		deviations []float64
	}{
		{
			name:       "Human-like deviations",
			deviations: []float64{1.0, 1.5, 2.0, 1.8, 1.2, 1.6, 2.2, 1.9, 1.4, 1.7},
		},
		{
			name:       "Robot-like deviations",
			deviations: []float64{0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1},
		},
		{
			name:       "Short deviations",
			deviations: []float64{1.0, 2.0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			periodicity := extractor.calculateJitterPeriodicity(tc.deviations)
			if periodicity < 0 || periodicity > 1 {
				t.Errorf("Periodicity out of range: %f", periodicity)
			}
			t.Logf("Jitter periodicity: %f", periodicity)
		})
	}
}

func TestCalculateJitterConsistency(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name        string
		xDeviations []float64
		yDeviations []float64
	}{
		{
			name:        "Consistent deviations",
			xDeviations: []float64{1.0, 1.1, 0.9, 1.0, 1.0},
			yDeviations: []float64{0.8, 0.8, 0.8, 0.8, 0.8},
		},
		{
			name:        "Inconsistent deviations",
			xDeviations: []float64{0.5, 2.0, 1.5, 0.1, 3.0},
			yDeviations: []float64{1.0, 0.1, 2.5, 0.5, 3.0},
		},
		{
			name:        "Empty deviations",
			xDeviations: []float64{},
			yDeviations: []float64{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			consistency := extractor.calculateJitterConsistency(tc.xDeviations, tc.yDeviations)
			if consistency < 0 || consistency > 1 {
				t.Errorf("Consistency out of range: %f", consistency)
			}
			t.Logf("Jitter consistency: %f", consistency)
		})
	}
}

func TestClassifyAccelerationPattern(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name          string
		accelerations []float64
		expected      []string
	}{
		{
			name:          "Constant acceleration",
			accelerations: []float64{1.0, 1.0, 1.0, 1.0, 1.0},
			expected:      []string{"constant"},
		},
		{
			name:          "Mostly accelerating",
			accelerations: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			expected:      []string{"accelerating", "late_acceleration"},
		},
		{
			name:          "Mostly decelerating",
			accelerations: []float64{5.0, 4.0, 3.0, 2.0, 1.0},
			expected:      []string{"decelerating", "early_deceleration"},
		},
		{
			name:          "Variable",
			accelerations: []float64{1.0, -1.0, 2.0, -2.0, 1.5},
			expected:      []string{"variable"},
		},
		{
			name:          "Short acceleration",
			accelerations: []float64{1.0, 2.0},
			expected:      []string{"unknown"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern := extractor.classifyAccelerationPattern(tc.accelerations)
			found := false
			for _, exp := range tc.expected {
				if pattern == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected pattern from %v, got %s", tc.expected, pattern)
			}
		})
	}
}

func TestDetectCurvaturePeaks(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		curvatures []float64
	}{
		{
			name:       "Normal curvatures",
			curvatures: []float64{0.1, 0.2, 0.5, 0.2, 0.3, 0.6, 0.2, 0.1},
		},
		{
			name:       "No peaks",
			curvatures: []float64{0.1, 0.15, 0.12, 0.18, 0.14},
		},
		{
			name:       "Short curvatures",
			curvatures: []float64{0.1, 0.2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			peaks := extractor.detectCurvaturePeaks(tc.curvatures)
			for _, peak := range peaks {
				if peak < 1 || peak >= len(tc.curvatures)-1 {
					t.Errorf("Invalid peak index: %d", peak)
				}
			}
			t.Logf("Detected %d curvature peaks at indices: %v", len(peaks), peaks)
		})
	}
}

func TestCountTurningPoints(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
		},
		{
			name:       "Straight robot trajectory",
			trajectory: generateRobotTrajectory(100),
		},
		{
			name:       "Backtrack trajectory",
			trajectory: generateBacktrackTrajectory(50),
		},
		{
			name:       "Short trajectory",
			trajectory: generateTestTrajectory(2, 100, 200, false),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			turningPoints := extractor.countTurningPoints(tc.trajectory)
			if turningPoints < 0 {
				t.Error("Turning points should be non-negative")
			}
			t.Logf("Turning points: %d", turningPoints)
		})
	}
}

func TestDetermineTurnDirection(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		trajectory []SliderPoint
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
		},
		{
			name:       "Straight robot trajectory",
			trajectory: generateRobotTrajectory(100),
		},
		{
			name:       "Short trajectory",
			trajectory: generateTestTrajectory(2, 100, 200, false),
		},
	}

	validDirections := map[string]bool{
		"straight": true, "mostly_left": true,
		"mostly_right": true, "mixed": true,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			direction := extractor.determineTurnDirection(tc.trajectory)
			if !validDirections[direction] {
				t.Errorf("Invalid turn direction: %s", direction)
			}
			t.Logf("Turn direction: %s", direction)
		})
	}
}

func TestCalculateCurvatureTrend(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		curvatures []float64
	}{
		{
			name:       "Increasing trend",
			curvatures: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		},
		{
			name:       "Decreasing trend",
			curvatures: []float64{0.5, 0.4, 0.3, 0.2, 0.1},
		},
		{
			name:       "Stable",
			curvatures: []float64{0.3, 0.3, 0.3, 0.3, 0.3},
		},
		{
			name:       "Random",
			curvatures: []float64{0.2, 0.5, 0.1, 0.4, 0.3},
		},
		{
			name:       "Short curvatures",
			curvatures: []float64{0.1, 0.2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			trend := extractor.calculateCurvatureTrend(tc.curvatures)
			if trend < -1 || trend > 1 {
				t.Errorf("Trend out of range [-1, 1]: %f", trend)
			}
			t.Logf("Curvature trend: %f", trend)
		})
	}
}

func TestCalculateAccelerationEntropy(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name          string
		accelerations []float64
	}{
		{
			name:          "Normal accelerations",
			accelerations: []float64{0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 2.5, 2.0, 1.5, 1.0},
		},
		{
			name:          "Constant accelerations",
			accelerations: []float64{1.0, 1.0, 1.0, 1.0, 1.0},
		},
		{
			name:          "Empty accelerations",
			accelerations: []float64{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entropy := extractor.calculateAccelerationEntropy(tc.accelerations)
			if entropy < 0 {
				t.Error("Entropy should be non-negative")
			}
			t.Logf("Acceleration entropy: %f", entropy)
		})
	}
}

func TestCalculateCurvatureEntropy(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		curvatures []float64
	}{
		{
			name:       "Normal curvatures",
			curvatures: []float64{0.1, 0.2, 0.3, 0.2, 0.1, 0.3, 0.2, 0.1},
		},
		{
			name:       "Constant curvatures",
			curvatures: []float64{0.2, 0.2, 0.2, 0.2, 0.2},
		},
		{
			name:       "Empty curvatures",
			curvatures: []float64{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entropy := extractor.calculateCurvatureEntropy(tc.curvatures)
			if entropy < 0 {
				t.Error("Entropy should be non-negative")
			}
			t.Logf("Curvature entropy: %f", entropy)
		})
	}
}

func TestCalculateSegmentSmoothness(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name        string
		trajectory  []SliderPoint
		numSegments int
	}{
		{
			name:        "Human-like trajectory",
			trajectory:  generateHumanLikeTrajectory(100),
			numSegments: 5,
		},
		{
			name:        "Robot trajectory",
			trajectory:  generateRobotTrajectory(100),
			numSegments: 5,
		},
		{
			name:        "Too short trajectory",
			trajectory:  generateTestTrajectory(5, 100, 200, false),
			numSegments: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			smoothness := extractor.calculateSegmentSmoothness(tc.trajectory, tc.numSegments)
			if len(tc.trajectory) >= tc.numSegments*2 {
				if len(smoothness) != tc.numSegments {
					t.Errorf("Expected %d smoothness values, got %d", tc.numSegments, len(smoothness))
				}
				for i, s := range smoothness {
					if s < 0 || s > 1 {
						t.Errorf("Segment %d smoothness out of range: %f", i, s)
					}
				}
			} else {
				if len(smoothness) != 0 {
					t.Errorf("Expected empty smoothness for short trajectory, got %d values", len(smoothness))
				}
			}
		})
	}
}

func TestSmoothnessQualityScore(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name    string
		metrics *SmoothnessMetrics
	}{
		{
			name: "High quality metrics",
			metrics: &SmoothnessMetrics{
				PrimarySmoothness:   0.95,
				SecondarySmoothness: 0.9,
				JerkScore:          0.05,
				CurveContinuity:    0.95,
				IsSmoothTransition: true,
				TransitionCount:    1,
			},
		},
		{
			name: "Low quality metrics",
			metrics: &SmoothnessMetrics{
				PrimarySmoothness:   0.3,
				SecondarySmoothness: 0.2,
				JerkScore:          0.8,
				CurveContinuity:    0.3,
				IsSmoothTransition: false,
				TransitionCount:    10,
			},
		},
		{
			name: "Zero metrics",
			metrics: &SmoothnessMetrics{
				PrimarySmoothness:   0,
				SecondarySmoothness: 0,
				JerkScore:          0,
				CurveContinuity:    0,
				IsSmoothTransition: false,
				TransitionCount:    0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := extractor.calculateSmoothnessQualityScore(tc.metrics)
			if score < 0 || score > 1 {
				t.Errorf("Quality score out of range: %f", score)
			}
			t.Logf("Quality score: %f", score)
		})
	}
}

func TestMathRng(t *testing.T) {
	t.Run("Consistent random values", func(t *testing.T) {
		rng1 := rand.New(rand.NewSource(42))
		rng2 := rand.New(rand.NewSource(42))

		for i := 0; i < 100; i++ {
			if rng1.Intn(1000) != rng2.Intn(1000) {
				t.Error("Same seed should produce same random values")
			}
		}
	})

	t.Run("Different seeds produce different values", func(t *testing.T) {
		rng1 := rand.New(rand.NewSource(42))
		rng2 := rand.New(rand.NewSource(43))

		foundDifference := false
		for i := 0; i < 100; i++ {
			if rng1.Intn(1000) != rng2.Intn(1000) {
				foundDifference = true
				break
			}
		}
		if !foundDifference {
			t.Log("Different seeds produced same sequence (statistically possible but unlikely)")
		}
	})

	t.Run("Intn bounds", func(t *testing.T) {
		rng := rand.New(rand.NewSource(42))
		for i := 0; i < 100; i++ {
			val := rng.Intn(100)
			if val < 0 || val >= 100 {
				t.Errorf("Intn(100) returned value out of bounds: %d", val)
			}
		}
	})

	t.Run("Float64 range", func(t *testing.T) {
		rng := rand.New(rand.NewSource(42))
		for i := 0; i < 100; i++ {
			val := rng.Float64()
			if val < 0 || val >= 1 {
				t.Errorf("Float64() returned value out of range [0, 1): %f", val)
			}
		}
	})
}

func TestSliderAnalyzer_Analyze(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	testCases := []struct {
		name        string
		trajectory  []SliderPoint
		targetPos   int
	}{
		{
			name:       "Human-like trajectory",
			trajectory: generateHumanLikeTrajectory(100),
			targetPos:  800,
		},
		{
			name:       "Robot trajectory",
			trajectory: generateRobotTrajectory(100),
			targetPos:  1000,
		},
		{
			name:       "Backtrack trajectory",
			trajectory: generateBacktrackTrajectory(50),
			targetPos:  400,
		},
		{
			name:       "Pause trajectory",
			trajectory: generatePauseTrajectory(50),
			targetPos:  400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := analyzer.AnalyzeSliderTrajectory(tc.trajectory, tc.targetPos)

			if err != nil {
				t.Fatalf("AnalyzeSliderTrajectory returned error: %v", err)
			}

			if result == nil {
				t.Fatal("AnalyzeSliderTrajectory returned nil")
			}

			if result.OverallRiskScore < 0 || result.OverallRiskScore > 1 {
				t.Errorf("OverallRiskScore out of range: %f", result.OverallRiskScore)
			}

			t.Logf("Risk score: %f, Confidence: %f, IsBot: %v",
				result.OverallRiskScore, result.Confidence, result.IsBot)
		})
	}
}

func BenchmarkSliderFeatureExtractor_ExtractFeatures(b *testing.B) {
	extractor := NewSliderFeatureExtractor()
	trajectory := generateHumanLikeTrajectory(200)
	sliderTraj := &SliderTrajectory{Points: trajectory}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.ExtractFeatures(trajectory, 1600, sliderTraj)
	}
}

func BenchmarkUnifiedRiskScorer_CalculateRiskScore(b *testing.B) {
	scorer := NewUnifiedRiskScorer()
	features := &SliderFeatures{
		PathEfficiency:      0.95,
		SpeedConsistency:    0.9,
		SmoothnessScore:     0.92,
		HumanLikenessScore:  0.88,
		AverageSpeed:        1200,
		SpeedVariance:       0.15,
		FractalDimension:    1.2,
		TrajectoryEntropy:   2.5,
		JitterScore:         0.03,
		CurvatureVariance:   0.05,
		TotalDuration:       2500,
		SmoothnessMetrics: &SmoothnessMetrics{
			PrimarySmoothness:   0.9,
			SecondarySmoothness: 0.85,
			JerkScore:          0.15,
			CurveContinuity:    0.9,
			IsSmoothTransition: true,
			TransitionCount:   3,
		},
		EnhancedAcceleration: &EnhancedAccelerationMetrics{
			MeanAcceleration:    0.5,
			StdAcceleration:     0.3,
			PositiveAccelRatio:  0.6,
			AccelerationEntropy: 2.5,
			AccelPattern:       "variable",
		},
		EnhancedCurvature: &EnhancedCurvatureMetrics{
			MeanCurvature:    0.1,
			StdCurvature:    0.05,
			TurningPoints:   5,
			TurnDirection:   "mixed",
			CurvatureBursts: 3,
		},
		EnhancedJitter: &EnhancedJitterMetrics{
			TotalJitter:       1.5,
			JitterEntropy:     2.0,
			JitterPeriodicity: 0.3,
			IsMicroJitter:    false,
			MicroJitterRatio: 0.4,
			JitterConsistency: 0.7,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scorer.CalculateRiskScore(features)
	}
}

func BenchmarkEnhancedSmoothnessMetrics(b *testing.B) {
	extractor := NewSliderFeatureExtractor()
	trajectory := generateHumanLikeTrajectory(200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.extractEnhancedSmoothnessMetrics(trajectory)
	}
}

func BenchmarkEnhancedJitterMetrics(b *testing.B) {
	extractor := NewSliderFeatureExtractor()
	trajectory := generateHumanLikeTrajectory(200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.extractEnhancedJitterMetrics(trajectory)
	}
}

var _ = time.Sleep
var _ = math.Pow
