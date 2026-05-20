package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClickAnalysisV2Enhanced_AnalyzeEnhancedClickSequence(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("basic_analysis", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100 + i*10),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Timing)
		assert.NotNil(t, result.Spatial)
		assert.NotNil(t, result.Rhythm)
		assert.GreaterOrEqual(t, result.Coordination, 0.0)
		assert.LessOrEqual(t, result.Coordination, 1.0)
	})

	t.Run("insufficient_data", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 1)
		clicks[0] = EnhancedClickData{
			X:         100,
			Y:         100,
			Timestamp: time.Now(),
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotNil(t, result)
	})

	t.Run("empty_clicks", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 0)

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotNil(t, result)
		assert.True(t, result.IsBot)
	})
}

func TestClickAnalysisV2Enhanced_TimingAnalysis(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("normal_timing", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(300+i*50) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.True(t, result.Timing.Normal)
		assert.Greater(t, result.Timing.AverageInterval, 0.0)
	})

	t.Run("mechanical_timing", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(200) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.False(t, result.Timing.Normal)
	})

	t.Run("timing_statistics", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(250+i*30) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.Greater(t, result.Timing.AverageInterval, 0.0)
		assert.GreaterOrEqual(t, result.Timing.Variance, 0.0)
		assert.GreaterOrEqual(t, result.Timing.StdDev, 0.0)
		assert.GreaterOrEqual(t, result.Timing.CoefficientVariation, 0.0)
	})
}

func TestClickAnalysisV2Enhanced_SpatialAnalysis(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("distributed_clicks", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(50 + (i*37)%300),
				Y:         float64(50 + (i*23)%200),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.Greater(t, result.Spatial.Entropy, 0.0)
		assert.GreaterOrEqual(t, result.Spatial.SpreadMetric, 0.0)
		assert.LessOrEqual(t, result.Spatial.SpreadMetric, 1.0)
	})

	t.Run("centered_clicks", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         195.0,
				Y:         145.0,
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.Greater(t, result.Spatial.CenterDensityRatio, 0.0)
	})

	t.Run("target_accuracy", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 3)
		targets := []struct{ X, Y float64 }{
			{100, 100},
			{200, 150},
			{300, 200},
		}
		baseTime := time.Now()
		for i := 0; i < 3; i++ {
			clicks[i] = EnhancedClickData{
				X:         targets[i].X,
				Y:         targets[i].Y,
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, targets)

		assert.GreaterOrEqual(t, result.Spatial.TargetAccuracy, 0.0)
		assert.LessOrEqual(t, result.Spatial.TargetAccuracy, 1.0)
	})
}

func TestClickAnalysisV2Enhanced_RhythmDetection(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("rhythmic_pattern", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(250) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.Contains(t, []string{"mechanical", "rhythmic", "regular"}, result.Rhythm.Type)
		assert.GreaterOrEqual(t, result.Rhythm.Consistency, 0.0)
		assert.LessOrEqual(t, result.Rhythm.Consistency, 1.0)
	})

	t.Run("random_pattern", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(200+i*100) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotEmpty(t, result.Rhythm.Type)
	})

	t.Run("accelerating_clicks", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			interval := 500 - i*40
			if interval < 100 {
				interval = 100
			}
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(i*interval) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotNil(t, result)
	})
}

func TestClickAnalysisV2Enhanced_RiskAssessment(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("high_risk_mechanical", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(200) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.GreaterOrEqual(t, result.RiskScore, 0.0)
		assert.LessOrEqual(t, result.RiskScore, 1.0)
	})

	t.Run("low_risk_normal", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(50 + (i*37)%300),
				Y:         float64(50 + (i*23)%200),
				Timestamp: baseTime.Add(time.Duration(300+i*50) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.GreaterOrEqual(t, result.RiskScore, 0.0)
		assert.LessOrEqual(t, result.RiskScore, 1.0)
	})

	t.Run("bot_probability", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(200) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.GreaterOrEqual(t, result.BotProbability, 0.0)
		assert.LessOrEqual(t, result.BotProbability, 1.0)
	})

	t.Run("human_likeness", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(50 + (i*37)%300),
				Y:         float64(50 + (i*23)%200),
				Timestamp: baseTime.Add(time.Duration(300+i*50) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.GreaterOrEqual(t, result.HumanLikeness, 0.0)
		assert.LessOrEqual(t, result.HumanLikeness, 1.0)
	})
}

func TestClickAnalysisV2Enhanced_SequenceMetrics(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("calculate_metrics", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		metrics := analyzer.calculateSequenceMetrics(clicks)

		assert.Equal(t, 10, metrics.TotalClicks)
		assert.Greater(t, metrics.Duration, 0.0)
		assert.Greater(t, metrics.TotalDistance, 0.0)
	})

	t.Run("path_efficiency", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		metrics := analyzer.calculateSequenceMetrics(clicks)

		assert.GreaterOrEqual(t, metrics.PathEfficiency, 0.0)
		assert.LessOrEqual(t, metrics.PathEfficiency, 1.0)
	})

	t.Run("curvature", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			angle := float64(i) * 0.5
			clicks[i] = EnhancedClickData{
				X:         100 + math.Cos(angle)*float64(i)*10,
				Y:         100 + math.Sin(angle)*float64(i)*10,
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		metrics := analyzer.calculateSequenceMetrics(clicks)

		assert.GreaterOrEqual(t, metrics.Curvature, 0.0)
	})
}

func TestClickAnalysisV2Enhanced_PatternClassification(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("classify_linear_pattern", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		metrics := analyzer.calculateSequenceMetrics(clicks)
		signature := analyzer.classifyPattern(clicks, metrics)

		assert.NotEmpty(t, signature.Type)
		assert.NotEmpty(t, signature.Description)
	})

	t.Run("classify_erratic_pattern", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(50 + (i*37)%300),
				Y:         float64(50 + (i*23)%200),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		metrics := analyzer.calculateSequenceMetrics(clicks)
		signature := analyzer.classifyPattern(clicks, metrics)

		assert.NotEmpty(t, signature.Type)
	})
}

func TestClickAnalysisV2Enhanced_SuspiciousPatterns(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("detect_mechanical_timing", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(200) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotEmpty(t, result.SuspiciousPatterns)
	})

	t.Run("detect_perfect_accuracy", func(t *testing.T) {
		targets := []struct{ X, Y float64 }{
			{100, 100},
			{200, 150},
			{300, 200},
		}
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         targets[i%3].X,
				Y:         targets[i%3].Y,
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, targets)

		assert.NotNil(t, result)
	})

	t.Run("detect_simultaneous_clicks", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime,
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotNil(t, result)
	})
}

func TestClickAnalysisV2Enhanced_Recommendations(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("generate_recommendations", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.NotNil(t, result.Recommendations)
		assert.NotEmpty(t, result.Recommendations)
	})
}

func TestClickAnalysisV2Enhanced_Confidence(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("calculate_confidence", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 10)
		baseTime := time.Now()
		for i := 0; i < 10; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)

		assert.GreaterOrEqual(t, result.Confidence, 0.0)
		assert.LessOrEqual(t, result.Confidence, 1.0)
	})
}

func TestClickAnalysisV2Enhanced_SessionAnalysis(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("analyze_multiple_sessions", func(t *testing.T) {
		sessions := make([][]EnhancedClickData, 5)
		for s := 0; s < 5; s++ {
			sessions[s] = make([]EnhancedClickData, 5)
			baseTime := time.Now().Add(time.Duration(s) * time.Minute)
			for i := 0; i < 5; i++ {
				sessions[s][i] = EnhancedClickData{
					X:         float64(100 + i*20),
					Y:         float64(100),
					Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
				}
			}
		}

		analysis := analyzer.AnalyzeSessionWithContext(sessions, nil)

		assert.NotNil(t, analysis)
		assert.Greater(t, analysis["total_sessions"].(int), 0)
		assert.Greater(t, analysis["total_clicks"].(int), 0)
	})

	t.Run("empty_sessions", func(t *testing.T) {
		sessions := make([][]EnhancedClickData, 0)

		analysis := analyzer.AnalyzeSessionWithContext(sessions, nil)

		assert.NotNil(t, analysis)
	})
}

func TestClickAnalysisV2Enhanced_ReportGeneration(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("generate_detailed_report", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			clicks[i] = EnhancedClickData{
				X:         float64(100 + i*20),
				Y:         float64(100),
				Timestamp: baseTime.Add(time.Duration(i*300) * time.Millisecond),
			}
		}

		report := analyzer.GenerateDetailedReport(clicks, nil)

		assert.NotEmpty(t, report)
		assert.Contains(t, report, "高级点选分析报告")
	})

	t.Run("generate_report_empty_clicks", func(t *testing.T) {
		clicks := make([]EnhancedClickData, 0)

		report := analyzer.GenerateDetailedReport(clicks, nil)

		assert.NotEmpty(t, report)
		assert.Contains(t, report, "未检测到点击数据")
	})
}

func TestClickAnalysisV2Enhanced_ThreadSafety(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("concurrent_analysis", func(t *testing.T) {
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func(id int) {
				clicks := make([]EnhancedClickData, 5)
				baseTime := time.Now()
				for j := 0; j < 5; j++ {
					clicks[j] = EnhancedClickData{
						X:         float64(100 + j*20),
						Y:         float64(100),
						Timestamp: baseTime.Add(time.Duration(j*300) * time.Millisecond),
					}
				}

				result := analyzer.AnalyzeEnhancedClickSequence(clicks, nil)
				assert.NotNil(t, result)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestClickAnalysisV2Enhanced_MathHelpers(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("calculate_mean", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := calculateMean(values)
		assert.Equal(t, 3.0, mean)
	})

	t.Run("calculate_variance", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := calculateMean(values)
		variance := calculateVariance(values, mean)
		assert.Greater(t, variance, 0.0)
	})

	t.Run("calculate_skewness", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 100.0}
		mean := calculateMean(values)
		stdDev := math.Sqrt(calculateVariance(values, mean))
		skewness := calculateSkewness(values, mean, stdDev)
		assert.NotNil(t, skewness)
	})

	t.Run("calculate_kurtosis", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := calculateMean(values)
		stdDev := math.Sqrt(calculateVariance(values, mean))
		kurtosis := calculateKurtosis(values, mean, stdDev)
		assert.NotNil(t, kurtosis)
	})

	t.Run("advanced_entropy", func(t *testing.T) {
		grid := make([][]int, 5)
		for i := range grid {
			grid[i] = make([]int, 5)
			for j := 0; j < 5; j++ {
				grid[i][j] = i*5 + j + 1
			}
		}

		entropy := analyzer.calculateAdvancedEntropy(grid)
		assert.GreaterOrEqual(t, entropy, 0.0)
	})

	t.Run("density_ratios", func(t *testing.T) {
		grid := make([][]int, 15)
		for i := range grid {
			grid[i] = make([]int, 15)
		}
		for i := 0; i < 5; i++ {
			grid[7][7+i] = 1
		}

		centerRatio, totalDensity := analyzer.calculateDensityRatios(grid, 15)
		assert.GreaterOrEqual(t, centerRatio, 0.0)
		assert.LessOrEqual(t, centerRatio, 1.0)
		assert.Equal(t, 5, totalDensity)
	})

	t.Run("spread_metric", func(t *testing.T) {
		grid := make([][]int, 10)
		for i := range grid {
			grid[i] = make([]int, 10)
		}
		grid[0][0] = 1
		grid[9][9] = 1

		spread := analyzer.calculateSpreadMetric(grid, 10)
		assert.Greater(t, spread, 0.0)
		assert.Less(t, spread, 1.0)
	})

	t.Run("clustering_factor", func(t *testing.T) {
		grid := make([][]int, 10)
		for i := range grid {
			grid[i] = make([]int, 10)
		}
		grid[5][5] = 3
		grid[5][6] = 2

		clustering := analyzer.calculateClusteringFactor(grid, 10)
		assert.GreaterOrEqual(t, clustering, 0.0)
	})

	t.Run("autocorrelation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

		autocorr := analyzer.calculateAutocorrelation(values)
		assert.NotNil(t, autocorr)
	})
}

func TestClickAnalysisV2Enhanced_Integration(t *testing.T) {
	analyzer := NewClickAnalysisV2Enhanced()

	t.Run("full_workflow", func(t *testing.T) {
		targets := []struct{ X, Y float64 }{
			{100, 100},
			{200, 150},
			{300, 200},
		}

		clicks := make([]EnhancedClickData, 5)
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			targetIdx := i % 3
			clicks[i] = EnhancedClickData{
				X:         targets[targetIdx].X + float64(i%2)*5,
				Y:         targets[targetIdx].Y + float64(i%2)*5,
				Timestamp: baseTime.Add(time.Duration(300+i*50) * time.Millisecond),
				Pressure:  0.5 + float64(i%3)*0.1,
			}
		}

		result := analyzer.AnalyzeEnhancedClickSequence(clicks, targets)

		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.RiskScore, 0.0)
		assert.LessOrEqual(t, result.RiskScore, 1.0)
		assert.GreaterOrEqual(t, result.BotProbability, 0.0)
		assert.LessOrEqual(t, result.BotProbability, 1.0)
		assert.GreaterOrEqual(t, result.HumanLikeness, 0.0)
		assert.LessOrEqual(t, result.HumanLikeness, 1.0)

		report := analyzer.GenerateDetailedReport(clicks, targets)
		assert.NotEmpty(t, report)
	})
}
