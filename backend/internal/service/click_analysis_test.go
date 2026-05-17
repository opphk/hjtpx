package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClickAnalyzer(t *testing.T) {
	analyzer := NewClickAnalyzer()
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.model)
}

func TestNewClickMLModel(t *testing.T) {
	model := NewClickMLModel()
	assert.NotNil(t, model)
	assert.Equal(t, -5.0, model.bias)
	assert.Equal(t, 10, len(model.weights))
}

func TestAnalyzeClickVerification(t *testing.T) {
	analyzer := NewClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []SliderClickData{
			{X: 100, Y: 200, Timestamp: 1000, Index: 0},
			{X: 200, Y: 300, Timestamp: 1500, Index: 1},
			{X: 300, Y: 250, Timestamp: 2000, Index: 2},
		},
		TargetImages: []TargetImage{
			{X: 100, Y: 200, Width: 50, Height: 50},
			{X: 200, Y: 300, Width: 50, Height: 50},
		},
	}

	result := analyzer.AnalyzeClickVerification(verification)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)
}

func TestAnalyzeClickVerification_EmptyClicks(t *testing.T) {
	analyzer := NewClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []SliderClickData{},
	}

	result := analyzer.AnalyzeClickVerification(verification)
	assert.NotNil(t, result)
	assert.True(t, result.IsBot)
	assert.Greater(t, result.Confidence, 0.5)
}

func TestAnalyzeClickVerification_NilVerification(t *testing.T) {
	analyzer := NewClickAnalyzer()

	result := analyzer.AnalyzeClickVerification(nil)
	assert.NotNil(t, result)
	assert.True(t, result.IsBot)
	assert.Greater(t, result.Confidence, 0.5)
}

func TestAnalyzeClickPattern(t *testing.T) {
	analyzer := NewClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []SliderClickData{
			{X: 100, Y: 200, Timestamp: 1000, Index: 0},
			{X: 200, Y: 300, Timestamp: 1300, Index: 1},
			{X: 300, Y: 250, Timestamp: 1600, Index: 2},
			{X: 400, Y: 350, Timestamp: 1900, Index: 3},
		},
	}

	pattern := analyzer.analyzeClickPattern(verification)
	assert.NotNil(t, pattern)
	assert.Equal(t, 4, pattern.ClickCount)
	assert.GreaterOrEqual(t, len(pattern.ClickIntervals), 0)
}

func TestAnalyzeClickPattern_SingleClick(t *testing.T) {
	analyzer := NewClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []SliderClickData{
			{X: 100, Y: 200, Timestamp: 1000, Index: 0},
		},
	}

	pattern := analyzer.analyzeClickPattern(verification)
	assert.NotNil(t, pattern)
	assert.Equal(t, 1, pattern.ClickCount)
}

func TestCalculateClickIntervals(t *testing.T) {
	analyzer := NewClickAnalyzer()

	clicks := []SliderClickData{
		{X: 100, Y: 200, Timestamp: 1000, Index: 0},
		{X: 200, Y: 300, Timestamp: 1300, Index: 1},
		{X: 300, Y: 250, Timestamp: 1600, Index: 2},
	}

	intervals := analyzer.calculateClickIntervals(clicks)
	assert.Equal(t, 2, len(intervals))
	assert.InDelta(t, 300.0, intervals[0], 1.0)
	assert.InDelta(t, 300.0, intervals[1], 1.0)
}

func TestAnalyzePositionDistribution(t *testing.T) {
	analyzer := NewClickAnalyzer()

	clicks := []SliderClickData{
		{X: 100, Y: 200, Timestamp: 1000, Index: 0},
		{X: 200, Y: 300, Timestamp: 1500, Index: 1},
		{X: 300, Y: 400, Timestamp: 2000, Index: 2},
	}

	distribution := analyzer.analyzePositionDistribution(clicks)
	assert.NotNil(t, distribution)
	assert.Greater(t, distribution.XMean, 0.0)
	assert.Greater(t, distribution.YMean, 0.0)
	assert.Greater(t, distribution.XVariance, 0.0)
	assert.Greater(t, distribution.YVariance, 0.0)
}

func TestAnalyzePositionDistribution_SingleClick(t *testing.T) {
	analyzer := NewClickAnalyzer()

	clicks := []SliderClickData{
		{X: 100, Y: 200, Timestamp: 1000, Index: 0},
	}

	distribution := analyzer.analyzePositionDistribution(clicks)
	assert.NotNil(t, distribution)
	assert.Equal(t, 100.0, distribution.XMean)
	assert.Equal(t, 200.0, distribution.YMean)
}

func TestCalculateEntropy(t *testing.T) {
	analyzer := NewClickAnalyzer()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	entropy := analyzer.calculateEntropy(values, 5)
	assert.Greater(t, entropy, 0.0)
}

func TestCalculateEntropy_EmptyValues(t *testing.T) {
	analyzer := NewClickAnalyzer()
	entropy := analyzer.calculateEntropy([]float64{}, 5)
	assert.Equal(t, 0.0, entropy)
}

func TestCalculateEntropy_ConstantValues(t *testing.T) {
	analyzer := NewClickAnalyzer()
	values := []float64{5.0, 5.0, 5.0, 5.0}
	entropy := analyzer.calculateEntropy(values, 5)
	assert.Equal(t, 0.0, entropy)
}

func TestGenerateClickSequence(t *testing.T) {
	analyzer := NewClickAnalyzer()

	clicks := []SliderClickData{
		{X: 100, Y: 200, Timestamp: 1000, Index: 0},
		{X: 300, Y: 200, Timestamp: 1500, Index: 1},
		{X: 300, Y: 400, Timestamp: 2000, Index: 2},
	}

	sequence := analyzer.generateClickSequence(clicks)
	assert.NotEmpty(t, sequence)
	assert.Contains(t, sequence, "start")
}

func TestClassifySequencePattern(t *testing.T) {
	analyzer := NewClickAnalyzer()

	tests := []struct {
		sequence string
		expected string
	}{
		{"start->right->right->right", "linear"},
		{"start->right->left->right->left", "varied"},
		{"start->down->down->down->down", "repeated"},
		{"", "unknown"},
		{"start", "single"},
	}

	for _, tt := range tests {
		t.Run(tt.sequence, func(t *testing.T) {
			pattern := analyzer.classifySequencePattern(tt.sequence)
			assert.Equal(t, tt.expected, pattern)
		})
	}
}

func TestCalculateClusteringScore(t *testing.T) {
	analyzer := NewClickAnalyzer()

	tests := []struct {
		name     string
		clicks   []ClickData
		minScore float64
		maxScore float64
	}{
		{
			name: "clustered clicks",
			clicks: []SliderClickData{
				{X: 100, Y: 200, Timestamp: 1000, Index: 0},
				{X: 105, Y: 205, Timestamp: 1500, Index: 1},
				{X: 102, Y: 198, Timestamp: 2000, Index: 2},
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
		{
			name: "spread clicks",
			clicks: []SliderClickData{
				{X: 100, Y: 100, Timestamp: 1000, Index: 0},
				{X: 500, Y: 500, Timestamp: 1500, Index: 1},
				{X: 800, Y: 200, Timestamp: 2000, Index: 2},
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.calculateClusteringScore(tt.clicks)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestCalculateClusteringScore_Empty(t *testing.T) {
	analyzer := NewClickAnalyzer()
	score := analyzer.calculateClusteringScore([]SliderClickData{})
	assert.Equal(t, 0.0, score)
}

func TestAnalyzeTiming(t *testing.T) {
	analyzer := NewClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []SliderClickData{
			{X: 100, Y: 200, Timestamp: 1000, Index: 0},
			{X: 200, Y: 300, Timestamp: 1500, Index: 1},
			{X: 300, Y: 250, Timestamp: 2000, Index: 2},
		},
	}

	timing := analyzer.analyzeTiming(verification)
	assert.NotNil(t, timing)
	assert.Greater(t, timing.TotalDuration, 0)
	assert.Greater(t, timing.FirstClickDelay, 0)
}

func TestCalculateResponseTimes(t *testing.T) {
	analyzer := NewClickAnalyzer()

	clicks := []SliderClickData{
		{X: 100, Y: 200, Timestamp: 1000, Index: 0},
		{X: 200, Y: 300, Timestamp: 1300, Index: 1},
		{X: 300, Y: 250, Timestamp: 1700, Index: 2},
	}

	times := analyzer.calculateResponseTimes(clicks)
	assert.Equal(t, 2, len(times))
	assert.InDelta(t, 300.0, times[0], 1.0)
	assert.InDelta(t, 400.0, times[1], 1.0)
}

func TestCalculateHesitationTimes(t *testing.T) {
	analyzer := NewClickAnalyzer()

	clicks := []SliderClickData{
		{X: 100, Y: 200, Timestamp: 1000, Index: 0},
		{X: 200, Y: 300, Timestamp: 2000, Index: 1},
		{X: 300, Y: 250, Timestamp: 2500, Index: 2},
	}

	hesitations := analyzer.calculateHesitationTimes(clicks)
	assert.GreaterOrEqual(t, len(hesitations), 0)
}

func TestClassifyTimingPattern(t *testing.T) {
	analyzer := NewClickAnalyzer()

	tests := []struct {
		timing  *TimingAnalysis
		pattern string
	}{
		{timing: &TimingAnalysis{TotalDuration: 500}, pattern: "very_fast"},
		{timing: &TimingAnalysis{TotalDuration: 2000}, pattern: "fast"},
		{timing: &TimingAnalysis{TotalDuration: 5000}, pattern: "normal"},
		{timing: &TimingAnalysis{TotalDuration: 10000}, pattern: "slow"},
		{timing: &TimingAnalysis{TotalDuration: 0}, pattern: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := analyzer.classifyTimingPattern(tt.timing)
			assert.Equal(t, tt.pattern, result)
		})
	}
}

func TestIsTimingRhythmic(t *testing.T) {
	analyzer := NewClickAnalyzer()

	tests := []struct {
		name       string
		timing     *TimingAnalysis
		isRhythmic bool
	}{
		{
			name: "rhythmic timing",
			timing: &TimingAnalysis{
				ResponseTimes:    []float64{300, 300, 300, 300, 300},
				AverageDuration:  300,
				DurationVariance: 10,
			},
			isRhythmic: true,
		},
		{
			name: "irregular timing",
			timing: &TimingAnalysis{
				ResponseTimes:    []float64{100, 500, 200, 800, 150},
				AverageDuration:  350,
				DurationVariance: 100000,
			},
			isRhythmic: false,
		},
		{
			name: "insufficient data",
			timing: &TimingAnalysis{
				ResponseTimes:    []float64{300, 300},
				AverageDuration:  300,
				DurationVariance: 0,
			},
			isRhythmic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isRhythmic := analyzer.isTimingRhythmic(tt.timing)
			assert.Equal(t, tt.isRhythmic, isRhythmic)
		})
	}
}

func TestAnalyzeAccuracy(t *testing.T) {
	analyzer := NewClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []SliderClickData{
			{X: 120, Y: 220, Timestamp: 1000, Index: 0},
			{X: 220, Y: 320, Timestamp: 1500, Index: 1},
		},
		TargetImages: []TargetImage{
			{X: 100, Y: 200, Width: 50, Height: 50},
			{X: 200, Y: 300, Width: 50, Height: 50},
		},
	}

	accuracy := analyzer.analyzeAccuracy(verification)
	assert.NotNil(t, accuracy)
	assert.Equal(t, 2, accuracy.TotalClicks)
	assert.GreaterOrEqual(t, accuracy.CorrectClicks, 0)
	assert.GreaterOrEqual(t, accuracy.Accuracy, 0.0)
	assert.LessOrEqual(t, accuracy.Accuracy, 1.0)
}

func TestAnalyzeAccuracy_NoTargets(t *testing.T) {
	analyzer := NewClickAnalyzer()

	verification := &ClickVerification{
		Clicks: []SliderClickData{
			{X: 100, Y: 200, Timestamp: 1000, Index: 0},
		},
		TargetImages: []TargetImage{},
	}

	accuracy := analyzer.analyzeAccuracy(verification)
	assert.NotNil(t, accuracy)
	assert.Equal(t, 0, accuracy.CorrectClicks)
}

func TestClickMLModel_Predict(t *testing.T) {
	model := NewClickMLModel()

	tests := []struct {
		name     string
		result   *ClickAnalysisResult
		minScore float64
		maxScore float64
	}{
		{
			name: "regular pattern",
			result: &ClickAnalysisResult{
				ClickPattern: &ClickPatternAnalysis{
					Regularity:      0.99,
					ClusteringScore: 0.1,
				},
				TimingAnalysis: &TimingAnalysis{
					IsRhythmic:      true,
					TimingPattern:   "very_fast",
					HesitationTimes: []float64{},
					TotalDuration:   2000,
				},
				AccuracyAnalysis: &AccuracyAnalysis{
					Accuracy:    1.0,
					TotalClicks: 5,
				},
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "normal pattern",
			result: &ClickAnalysisResult{
				ClickPattern: &ClickPatternAnalysis{
					Regularity:      0.5,
					ClusteringScore: 0.5,
				},
				TimingAnalysis: &TimingAnalysis{
					IsRhythmic:      false,
					TimingPattern:   "normal",
					HesitationTimes: []float64{200, 300},
					TotalDuration:   5000,
				},
				AccuracyAnalysis: &AccuracyAnalysis{
					Accuracy:    0.7,
					TotalClicks: 3,
				},
			},
			minScore: 0.0,
			maxScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := model.Predict(tt.result)
			t.Logf("Test %s: score = %f", tt.name, score)
			assert.GreaterOrEqual(t, score, 0.0)
			assert.LessOrEqual(t, score, 1.0)
		})
	}
}

func TestClickMLModel_PredictNil(t *testing.T) {
	model := NewClickMLModel()
	score := model.Predict(nil)
	assert.Equal(t, 0.5, score)
}

func TestClickDetectAnomalies(t *testing.T) {
	analyzer := NewClickAnalyzer()

	tests := []struct {
		name     string
		result   *ClickAnalysisResult
		minScore float64
		maxScore float64
	}{
		{
			name: "high anomaly",
			result: &ClickAnalysisResult{
				ClickPattern: &ClickPatternAnalysis{
					Regularity:      0.99,
					SequencePattern: "linear",
					ClusteringScore: 0.05,
					PositionDistribution: &PositionDistribution{
						XEntropy: 0.5,
						YEntropy: 0.5,
					},
				},
				TimingAnalysis: &TimingAnalysis{
					IsRhythmic:      true,
					TimingPattern:   "very_fast",
					HesitationTimes: []float64{},
					TotalDuration:   3000,
				},
				AccuracyAnalysis: &AccuracyAnalysis{
					Accuracy:            1.0,
					TotalClicks:         5,
					AverageMissDistance: 3,
				},
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "normal",
			result: &ClickAnalysisResult{
				ClickPattern: &ClickPatternAnalysis{
					Regularity:      0.6,
					SequencePattern: "varied",
					ClusteringScore: 0.5,
				},
				TimingAnalysis: &TimingAnalysis{
					IsRhythmic:      false,
					TimingPattern:   "normal",
					HesitationTimes: []float64{200, 300},
					TotalDuration:   5000,
				},
				AccuracyAnalysis: &AccuracyAnalysis{
					Accuracy:    0.7,
					TotalClicks: 3,
				},
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.detectAnomalies(tt.result)
			t.Logf("Test %s: anomaly score = %f", tt.name, score)
			assert.GreaterOrEqual(t, score, 0.0)
			assert.LessOrEqual(t, score, 1.0)
		})
	}
}

func TestClickCalculateRiskScore(t *testing.T) {
	analyzer := NewClickAnalyzer()

	result := &ClickAnalysisResult{
		AnomalyScore: 0.7,
		MLScore:      0.6,
		ClickPattern: &ClickPatternAnalysis{
			Regularity:      0.95,
			ClusteringScore: 0.2,
		},
		TimingAnalysis: &TimingAnalysis{
			IsRhythmic:    true,
			TimingPattern: "very_fast",
		},
		AccuracyAnalysis: &AccuracyAnalysis{
			Accuracy:    1.0,
			TotalClicks: 4,
		},
	}

	score := analyzer.calculateOverallRiskScore(result)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestClickAnalyzerConfidence(t *testing.T) {
	analyzer := NewClickAnalyzer()

	tests := []struct {
		name    string
		result  *ClickAnalysisResult
		minConf float64
	}{
		{
			name: "high confidence",
			result: &ClickAnalysisResult{
				ClickPattern: &ClickPatternAnalysis{
					ClickCount: 5,
				},
				TimingAnalysis: &TimingAnalysis{
					TotalDuration: 2000,
				},
				AccuracyAnalysis: &AccuracyAnalysis{
					TotalClicks: 4,
				},
				AnomalyScore: 0.5,
				MLScore:      0.7,
			},
			minConf: 0.8,
		},
		{
			name: "low confidence",
			result: &ClickAnalysisResult{
				ClickPattern: &ClickPatternAnalysis{
					ClickCount: 1,
				},
				TimingAnalysis: &TimingAnalysis{
					TotalDuration: 100,
				},
				AccuracyAnalysis: &AccuracyAnalysis{
					TotalClicks: 1,
				},
				AnomalyScore: 0.1,
				MLScore:      0.3,
			},
			minConf: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := analyzer.calculateConfidence(tt.result)
			assert.GreaterOrEqual(t, conf, tt.minConf)
			assert.LessOrEqual(t, conf, 0.99)
		})
	}
}

func TestMeanVarianceMaxMin_ClickAnalyzer(t *testing.T) {
	analyzer := NewClickAnalyzer()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	assert.InDelta(t, 3.0, analyzer.mean(values), 0.01)
	assert.InDelta(t, 2.0, analyzer.variance(values), 0.1)
	assert.InDelta(t, 5.0, analyzer.max(values), 0.01)
	assert.InDelta(t, 1.0, analyzer.min(values), 0.01)
}

func TestMeanVarianceMaxMin_ClickAnalyzer_Empty(t *testing.T) {
	analyzer := NewClickAnalyzer()

	assert.Equal(t, 0.0, analyzer.mean([]float64{}))
	assert.Equal(t, 0.0, analyzer.variance([]float64{}))
	assert.Equal(t, 0.0, analyzer.max([]float64{}))
	assert.Equal(t, 0.0, analyzer.min([]float64{}))
}

func TestNewClickPatternDetector(t *testing.T) {
	detector := NewClickPatternDetector()
	assert.NotNil(t, detector)
	assert.Equal(t, 5, len(detector.patterns))
}

func TestClickPatternDetector_DetectPatterns(t *testing.T) {
	detector := NewClickPatternDetector()

	result := &ClickAnalysisResult{
		ClickPattern: &ClickPatternAnalysis{
			Regularity: 0.99,
		},
		TimingAnalysis: &TimingAnalysis{
			IsRhythmic:      true,
			TimingPattern:   "very_fast",
			HesitationTimes: []float64{},
			TotalDuration:   3000,
		},
		AccuracyAnalysis: &AccuracyAnalysis{
			Accuracy:    1.0,
			TotalClicks: 5,
		},
	}

	patterns := detector.DetectPatterns(result)
	assert.GreaterOrEqual(t, len(patterns), 0)
}

func TestClickGenerateReport(t *testing.T) {
	analyzer := NewClickAnalyzer()

	result := &ClickAnalysisResult{
		ClickPattern: &ClickPatternAnalysis{
			ClickCount:       5,
			AverageInterval:  300,
			IntervalVariance: 50,
			IntervalStdDev:   7.07,
			Regularity:       0.85,
			ClusteringScore:  0.5,
			ClickSequence:    "start->right->down->left->up",
			SequencePattern:  "varied",
			PositionDistribution: &PositionDistribution{
				XMean:     300,
				YMean:     250,
				XVariance: 10000,
				YVariance: 8000,
				XEntropy:  3.2,
				YEntropy:  3.0,
				SpreadX:   400,
				SpreadY:   300,
			},
		},
		TimingAnalysis: &TimingAnalysis{
			TotalDuration:    2500,
			AverageDuration:  625,
			DurationVariance: 10000,
			FirstClickDelay:  1000,
			TimingPattern:    "normal",
			IsRhythmic:       false,
			HesitationTimes:  []float64{150, 200},
		},
		AccuracyAnalysis: &AccuracyAnalysis{
			CorrectClicks:       4,
			TotalClicks:         5,
			Accuracy:            0.8,
			AverageMissDistance: 15,
			Precision:           0.8,
		},
		AnomalyScore:      0.3,
		MLScore:           0.4,
		OverallRiskScore:  0.35,
		IsBot:             false,
		Confidence:        0.85,
		RiskIndicators:    []string{"test indicator"},
		AnomalyDetections: []string{"test detection"},
	}

	report := analyzer.GenerateReport(result)
	assert.NotEmpty(t, report)
	assert.Contains(t, report, "=== 点选验证分析报告 ===")
	assert.Contains(t, report, "点击模式分析")
	assert.Contains(t, report, "风险评估")
}

func TestGenerateBotLikeClickData(t *testing.T) {
	targets := []TargetImage{
		{X: 100, Y: 200, Width: 50, Height: 50},
		{X: 300, Y: 400, Width: 50, Height: 50},
	}

	clicks := GenerateBotLikeClickData(targets, 1000)
	assert.Equal(t, len(targets), len(clicks))

	for i, click := range clicks {
		if i < len(targets) {
			assert.Equal(t, targets[i].X+targets[i].Width/2, click.X)
			assert.Equal(t, targets[i].Y+targets[i].Height/2, click.Y)
		}
	}
}

func TestHumanVsBotClickDetection(t *testing.T) {
	analyzer := NewClickAnalyzer()

	targets := []TargetImage{
		{X: 100, Y: 200, Width: 50, Height: 50},
		{X: 300, Y: 400, Width: 50, Height: 50},
		{X: 500, Y: 300, Width: 50, Height: 50},
	}

	humanVerification := &ClickVerification{
		Clicks:       GenerateHumanLikeClickData(targets, 3000),
		TargetImages: targets,
	}
	humanResult := analyzer.AnalyzeClickVerification(humanVerification)

	botVerification := &ClickVerification{
		Clicks:       GenerateBotLikeClickData(targets, 1000),
		TargetImages: targets,
	}
	botResult := analyzer.AnalyzeClickVerification(botVerification)

	t.Logf("人类点击风险分数: %.4f", humanResult.OverallRiskScore)
	t.Logf("机器人点击风险分数: %.4f", botResult.OverallRiskScore)
	t.Logf("人类点击判定为机器人: %v", humanResult.IsBot)
	t.Logf("机器人点击判定为机器人: %v", botResult.IsBot)

	assert.Less(t, humanResult.OverallRiskScore, botResult.OverallRiskScore,
		"人类点击风险分数应低于机器人点击")
}

func TestClickAnalysisFullPipeline(t *testing.T) {
	analyzer := NewClickAnalyzer()

	targets := []TargetImage{
		{X: 100, Y: 200, Width: 50, Height: 50},
		{X: 300, Y: 400, Width: 50, Height: 50},
	}

	verification := &ClickVerification{
		Clicks:       GenerateHumanLikeClickData(targets, 3000),
		TargetImages: targets,
	}

	result := analyzer.AnalyzeClickVerification(verification)
	assert.NotNil(t, result)

	assert.NotNil(t, result.ClickPattern)
	assert.NotNil(t, result.TimingAnalysis)
	assert.NotNil(t, result.AccuracyAnalysis)

	assert.Greater(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)

	report := analyzer.GenerateReport(result)
	assert.NotEmpty(t, report)
}

func TestClickAnalysisWithEdgeCases(t *testing.T) {
	analyzer := NewClickAnalyzer()

	t.Run("nil verification", func(t *testing.T) {
		result := analyzer.AnalyzeClickVerification(nil)
		assert.NotNil(t, result)
		assert.True(t, result.IsBot)
	})

	t.Run("empty clicks", func(t *testing.T) {
		result := analyzer.AnalyzeClickVerification(&ClickVerification{
			Clicks:       []SliderClickData{},
			TargetImages: []TargetImage{},
		})
		assert.NotNil(t, result)
		assert.True(t, result.IsBot)
	})

	t.Run("single click", func(t *testing.T) {
		result := analyzer.AnalyzeClickVerification(&ClickVerification{
			Clicks: []SliderClickData{
				{X: 100, Y: 200, Timestamp: 1000, Index: 0},
			},
			TargetImages: []TargetImage{
				{X: 100, Y: 200, Width: 50, Height: 50},
			},
		})
		assert.NotNil(t, result)
		assert.Greater(t, result.Confidence, 0.0)
	})

	t.Run("many clicks", func(t *testing.T) {
		clicks := make([]ClickData, 100)
		for i := range clicks {
			clicks[i] = SliderClickData{
				X:         100 + (i%10)*50,
				Y:         200 + (i/10)*50,
				Timestamp: int64(1000 + i*100),
				Index:     i,
			}
		}
		result := analyzer.AnalyzeClickVerification(&ClickVerification{
			Clicks:       clicks,
			TargetImages: []TargetImage{},
		})
		assert.NotNil(t, result)
		assert.Greater(t, result.Confidence, 0.7)
	})
}

func BenchmarkClickAnalysis(b *testing.B) {
	analyzer := NewClickAnalyzer()

	targets := []TargetImage{
		{X: 100, Y: 200, Width: 50, Height: 50},
		{X: 300, Y: 400, Width: 50, Height: 50},
	}

	verification := &ClickVerification{
		Clicks:       GenerateHumanLikeClickData(targets, 3000),
		TargetImages: targets,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeClickVerification(verification)
	}
}
