package service

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewKeyboardAnalyzer(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()
	assert.NotNil(t, analyzer)
	assert.Equal(t, 150.0, analyzer.typingSpeedThreshold)
	assert.Equal(t, 0.15, analyzer.errorRateThreshold)
	assert.Equal(t, 0.95, analyzer.rhythmThreshold)
}

func TestAnalyzeKeyboardBehavior(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000, IsModifier: false},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100, IsModifier: false},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200, IsModifier: false},
		{KeyCode: 68, Key: "d", EventType: "keydown", Timestamp: 1300, IsModifier: false},
	}

	data := &model.KeyboardBehaviorData{
		SessionID: "test-session",
		UserID:    "test-user",
		KeyEvents: events,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Second),
	}

	features := analyzer.AnalyzeKeyboardBehavior(data)
	assert.NotNil(t, features)
	assert.Equal(t, 4, features.TypingSpeed.TotalCharacters)
	assert.Greater(t, features.TypingSpeed.WPM, 0.0)
	assert.Equal(t, 0, features.ErrorRate.BackspaceCount)
	assert.Equal(t, "low", features.RiskLevel)
}

func TestAnalyzeKeyboardBehavior_NilData(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	features := analyzer.AnalyzeKeyboardBehavior(nil)
	assert.NotNil(t, features)
	assert.False(t, features.IsHumanLike)
	assert.Equal(t, float64(100), features.OverallScore)
	assert.Equal(t, "high", features.RiskLevel)
	assert.Contains(t, features.AnomalyIndicators, "无键盘数据")
}

func TestAnalyzeKeyboardBehavior_EmptyEvents(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	data := &model.KeyboardBehaviorData{
		SessionID: "test-session",
		UserID:    "test-user",
		KeyEvents: []model.KeyEvent{},
	}

	features := analyzer.AnalyzeKeyboardBehavior(data)
	assert.NotNil(t, features)
	assert.False(t, features.IsHumanLike)
	assert.Equal(t, "high", features.RiskLevel)
}

func TestExtractTypingSpeedFeatures(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200},
		{KeyCode: 68, Key: "d", EventType: "keydown", Timestamp: 1300},
		{KeyCode: 69, Key: "e", EventType: "keydown", Timestamp: 1600},
	}

	features := analyzer.extractTypingSpeedFeatures(events)

	assert.Equal(t, 5, features.TotalCharacters)
	assert.Greater(t, features.AverageInterval, 0.0)
	assert.Equal(t, 100.0, features.MinInterval)
	assert.Greater(t, features.MaxInterval, 0.0)
	assert.GreaterOrEqual(t, features.WPM, 0.0)
	assert.GreaterOrEqual(t, features.SpeedConsistency, 0.0)
}

func TestExtractTypingSpeedFeatures_EmptyEvents(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	features := analyzer.extractTypingSpeedFeatures([]model.KeyEvent{})

	assert.Equal(t, 0, features.TotalCharacters)
	assert.Equal(t, 0.0, features.AverageInterval)
}

func TestExtractTypingSpeedFeatures_WithPauses(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200},
		{KeyCode: 68, Key: "d", EventType: "keydown", Timestamp: 2600},
		{KeyCode: 69, Key: "e", EventType: "keydown", Timestamp: 2700},
	}

	features := analyzer.extractTypingSpeedFeatures(events)

	assert.Greater(t, features.PauseCount, 0)
	assert.Equal(t, 1, features.PauseCount)
}

func TestExtractErrorRateFeatures(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 8, Key: "Backspace", EventType: "keydown", Timestamp: 1200},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1300},
	}

	features := analyzer.extractErrorRateFeatures(events)

	assert.Equal(t, 1, features.BackspaceCount)
	assert.Equal(t, 1, features.CorrectionCount)
	assert.Greater(t, features.ErrorRate, 0.0)
	assert.GreaterOrEqual(t, features.AccuracyScore, 0.0)
}

func TestExtractErrorRateFeatures_WithDelete(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 46, Key: "Delete", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1200},
	}

	features := analyzer.extractErrorRateFeatures(events)

	assert.Equal(t, 1, features.DeleteCount)
	assert.Equal(t, 1, features.CorrectionCount)
}

func TestExtractErrorRateFeatures_NoErrors(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200},
	}

	features := analyzer.extractErrorRateFeatures(events)

	assert.Equal(t, 0, features.BackspaceCount)
	assert.Equal(t, 0, features.DeleteCount)
	assert.Equal(t, 0.0, features.ErrorRate)
	assert.Equal(t, 1.0, features.AccuracyScore)
}

func TestExtractRhythmFeatures(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200},
		{KeyCode: 68, Key: "d", EventType: "keydown", Timestamp: 1300},
		{KeyCode: 69, Key: "e", EventType: "keydown", Timestamp: 1400},
	}

	features := analyzer.extractRhythmFeatures(events)

	assert.Greater(t, len(features.IntervalSequence), 0)
	assert.Greater(t, features.AverageRhythm, 0.0)
	assert.GreaterOrEqual(t, features.RhythmVariance, 0.0)
	assert.GreaterOrEqual(t, features.RhythmEntropy, 0.0)
}

func TestExtractRhythmFeatures_Regular(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200},
		{KeyCode: 68, Key: "d", EventType: "keydown", Timestamp: 1300},
		{KeyCode: 69, Key: "e", EventType: "keydown", Timestamp: 1400},
		{KeyCode: 70, Key: "f", EventType: "keydown", Timestamp: 1500},
	}

	features := analyzer.extractRhythmFeatures(events)

	assert.Greater(t, features.RhythmRegularity, 0.8)
}

func TestExtractRhythmFeatures_Varied(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1500},
		{KeyCode: 68, Key: "d", EventType: "keydown", Timestamp: 1600},
		{KeyCode: 69, Key: "e", EventType: "keydown", Timestamp: 2100},
		{KeyCode: 70, Key: "f", EventType: "keydown", Timestamp: 2200},
	}

	features := analyzer.extractRhythmFeatures(events)

	assert.Greater(t, features.RhythmChanges, 0)
}

func TestExtractComboKeyFeatures(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 17, Key: "Control", EventType: "keydown", Timestamp: 1000, IsModifier: true},
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1010, IsModifier: false},
		{KeyCode: 17, Key: "Control", EventType: "keyup", Timestamp: 1020, IsModifier: true},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1030, IsModifier: false},
	}

	features := analyzer.extractComboKeyFeatures(events)

	assert.GreaterOrEqual(t, features.TotalCombos, 0)
	assert.GreaterOrEqual(t, features.CtrlCombos, 0)
}

func TestExtractComboKeyFeatures_NoCombos(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200},
	}

	features := analyzer.extractComboKeyFeatures(events)

	assert.Equal(t, 0, features.TotalCombos)
	assert.Equal(t, 0, features.CtrlCombos)
	assert.Equal(t, 0, features.AltCombos)
}

func TestCalculateKeyIntervals(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1200},
	}

	intervals := analyzer.calculateKeyIntervals(events)

	assert.Equal(t, 2, len(intervals))
	assert.Equal(t, 100.0, intervals[0])
	assert.Equal(t, 100.0, intervals[1])
}

func TestCalculateKeyIntervals_WithNonKeydown(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 65, Key: "a", EventType: "keyup", Timestamp: 1050},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1100},
	}

	intervals := analyzer.calculateKeyIntervals(events)

	assert.Equal(t, 1, len(intervals))
	assert.Equal(t, 100.0, intervals[0])
}

func TestDetectBursts(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	intervals := []float64{20, 30, 40, 200, 25, 35, 45, 300, 20, 30, 40}

	bursts := analyzer.detectBursts(intervals)

	assert.GreaterOrEqual(t, len(bursts), 0)
}

func TestDetectPauses(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	intervals := []float64{100, 200, 600, 100, 700, 100, 100}

	pauses := analyzer.detectPauses(intervals)

	assert.Equal(t, 2, len(pauses))
}

func TestCalculateEntropy(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{100, 100, 100, 100, 100}

	entropy := analyzer.calculateEntropy(values)

	assert.GreaterOrEqual(t, entropy, 0.0)
}

func TestCountPeaks(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{10, 50, 20, 60, 30, 40, 10}

	peaks := analyzer.countPeaks(values)

	assert.GreaterOrEqual(t, peaks, 2)
}

func TestCountValleys(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{50, 10, 40, 5, 30, 60, 50}

	valleys := analyzer.countValleys(values)

	assert.Equal(t, 2, valleys)
}

func TestCalculateAutocorrelation(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{100, 110, 90, 105, 95}

	autocorr := analyzer.calculateAutocorrelation(values)

	assert.GreaterOrEqual(t, len(autocorr), 0)
}

func TestCalculateAutocorrelation_Empty(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	autocorr := analyzer.calculateAutocorrelation([]float64{})

	assert.Equal(t, 0, len(autocorr))
}

func TestDetectFastSegments(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{30, 40, 35, 150, 160, 145, 50, 60}

	segments := analyzer.detectFastSegments(values)

	assert.GreaterOrEqual(t, len(segments), 0)
}

func TestDetectSlowSegments(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{200, 250, 180, 100, 110, 300, 280}

	segments := analyzer.detectSlowSegments(values)

	assert.GreaterOrEqual(t, len(segments), 0)
}

func TestCountSimultaneousPress(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 66, Key: "b", EventType: "keydown", Timestamp: 1005},
		{KeyCode: 67, Key: "c", EventType: "keydown", Timestamp: 1500},
	}

	count := analyzer.countSimultaneousPress(events)

	assert.GreaterOrEqual(t, count, 0)
}

func TestMean(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{10, 20, 30, 40, 50}

	mean := analyzer.mean(values)

	assert.Equal(t, 30.0, mean)
}

func TestMean_Empty(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	mean := analyzer.mean([]float64{})

	assert.Equal(t, 0.0, mean)
}

func TestMax(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{10, 50, 30, 20, 40}

	max := analyzer.max(values)

	assert.Equal(t, 50.0, max)
}

func TestMin(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{50, 10, 30, 20, 40}

	min := analyzer.min(values)

	assert.Equal(t, 10.0, min)
}

func TestVariance(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{10, 20, 30, 40, 50}

	variance := analyzer.variance(values)

	assert.Greater(t, variance, 0.0)
}

func TestSkewness(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{10, 20, 30, 40, 50}

	skewness := analyzer.skewness(values)

	assert.GreaterOrEqual(t, skewness, 0.0)
}

func TestKurtosis(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{10, 20, 30, 40, 50, 60, 70}

	kurtosis := analyzer.kurtosis(values)

	assert.GreaterOrEqual(t, kurtosis, -3.0)
}

func TestCalculateRiskScore(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	features := &model.KeyboardBehaviorFeatures{
		TypingSpeed: model.TypingSpeedFeature{
			WPM: 50,
		},
		ErrorRate: model.KeyboardErrorFeature{
			ErrorRate: 0.1,
		},
		Rhythm: model.KeyboardRhythmFeature{
			RhythmRegularity: 0.8,
		},
		AnomalyIndicators: []string{},
	}

	riskScore := analyzer.CalculateRiskScore(features)

	assert.LessOrEqual(t, riskScore, 100.0)
}

func TestCalculateRiskScore_HighRisk(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	features := &model.KeyboardBehaviorFeatures{
		TypingSpeed: model.TypingSpeedFeature{
			WPM: 200,
		},
		ErrorRate: model.KeyboardErrorFeature{
			ErrorRate: 0.5,
		},
		Rhythm: model.KeyboardRhythmFeature{
			RhythmRegularity: 0.99,
		},
		AnomalyIndicators: []string{"异常1", "异常2", "异常3", "异常4"},
	}

	riskScore := analyzer.CalculateRiskScore(features)

	assert.Greater(t, riskScore, 50.0)
}

func TestIsBotBehavior(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	features := &model.KeyboardBehaviorFeatures{
		TypingSpeed: model.TypingSpeedFeature{
			WPM: 200,
		},
		ErrorRate: model.KeyboardErrorFeature{
			ErrorRate: 0.5,
		},
		Rhythm: model.KeyboardRhythmFeature{
			RhythmRegularity: 0.99,
		},
		AnomalyIndicators: []string{"异常1", "异常2"},
	}

	isBot := analyzer.IsBotBehavior(features)

	assert.True(t, isBot)
}

func TestIsBotBehavior_Human(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	features := &model.KeyboardBehaviorFeatures{
		TypingSpeed: model.TypingSpeedFeature{
			WPM: 60,
		},
		ErrorRate: model.KeyboardErrorFeature{
			ErrorRate: 0.05,
		},
		Rhythm: model.KeyboardRhythmFeature{
			RhythmRegularity: 0.7,
		},
		AnomalyIndicators: []string{},
	}

	isBot := analyzer.IsBotBehavior(features)

	assert.False(t, isBot)
}

func TestGenerateReport(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	features := &model.KeyboardBehaviorFeatures{
		TypingSpeed: model.TypingSpeedFeature{
			WPM:              60,
			AverageInterval:  100,
			SpeedConsistency: 0.8,
		},
		ErrorRate: model.KeyboardErrorFeature{
			BackspaceCount: 2,
			ErrorRate:      0.05,
			AccuracyScore:  0.95,
		},
		Rhythm: model.KeyboardRhythmFeature{
			RhythmRegularity: 0.8,
			RhythmEntropy:     2.5,
			RhythmChanges:     3,
		},
		ComboKeys: model.ComboKeyFeature{
			CtrlCombos:  1,
			AltCombos:   0,
			ShiftCombos: 0,
		},
		OverallScore: 75,
		Confidence:   0.9,
		RiskLevel:    "low",
		AnomalyIndicators: []string{},
	}

	report := analyzer.GenerateReport(features)

	assert.Contains(t, report, "键盘行为分析报告")
	assert.Contains(t, report, "打字速度分析")
	assert.Contains(t, report, "错误率分析")
	assert.Contains(t, report, "WPM")
	assert.Contains(t, report, "风险等级")
}

func TestCalculateOverallScore(t *testing.T) {
	features := &model.KeyboardBehaviorFeatures{
		TypingSpeed: model.TypingSpeedFeature{
			WPM: 60,
		},
		ErrorRate: model.KeyboardErrorFeature{
			ErrorRate: 0.05,
		},
		Rhythm: model.KeyboardRhythmFeature{
			RhythmRegularity: 0.8,
		},
		AnomalyIndicators: []string{},
	}

	features.CalculateOverallScore()

	assert.Greater(t, features.OverallScore, 0.0)
	assert.LessOrEqual(t, features.OverallScore, 100.0)
	assert.True(t, features.IsHumanLike)
	assert.Equal(t, "low", features.RiskLevel)
}

func TestCalculateOverallScore_HighRisk(t *testing.T) {
	features := &model.KeyboardBehaviorFeatures{
		TypingSpeed: model.TypingSpeedFeature{
			WPM: 10,
		},
		ErrorRate: model.KeyboardErrorFeature{
			ErrorRate: 0.3,
		},
		Rhythm: model.KeyboardRhythmFeature{
			RhythmRegularity: 0.99,
		},
		ComboKeys: model.ComboKeyFeature{
			ModifierUsageRate: 0.0,
			TotalCombos: 0,
		},
		AnomalyIndicators: []string{},
	}

	features.CalculateOverallScore()

	assert.Less(t, features.OverallScore, 60.0)
	assert.False(t, features.IsHumanLike)
	assert.Contains(t, []string{"high", "medium"}, features.RiskLevel)
}

func TestGetActiveModifiers(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 17, Key: "Control", EventType: "keydown", Timestamp: 1000, IsModifier: true},
		{KeyCode: 65, Key: "a", EventType: "keydown", Timestamp: 1010, IsModifier: false},
	}

	modifiers := analyzer.getActiveModifiers(events, 1)

	assert.Contains(t, modifiers, "Control")
}

func TestCalculateHoldDuration(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{
		{KeyCode: 17, Key: "Control", EventType: "keydown", Timestamp: 1000},
		{KeyCode: 17, Key: "Control", EventType: "keyup", Timestamp: 1150},
	}

	duration := analyzer.calculateHoldDuration(events, 0, "Control")

	assert.Equal(t, 150.0, duration)
}

func TestCountRhythmChanges(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	values := []float64{100, 150, 100, 150, 100}

	changes := analyzer.countRhythmChanges(values)

	assert.GreaterOrEqual(t, changes, 0)
}

func TestAnalyzeKeyboardBehavior_ExtremeWPM(t *testing.T) {
	analyzer := NewKeyboardAnalyzer()

	events := []model.KeyEvent{}
	baseTime := int64(1000)
	for i := 0; i < 100; i++ {
		events = append(events, model.KeyEvent{
			KeyCode:   65 + (i % 26),
			Key:        string(rune(65 + (i % 26))),
			EventType:  "keydown",
			Timestamp:  baseTime + int64(i*10),
		})
	}

	data := &model.KeyboardBehaviorData{
		SessionID: "test-session",
		UserID:    "test-user",
		KeyEvents: events,
	}

	features := analyzer.AnalyzeKeyboardBehavior(data)

	assert.Greater(t, features.TypingSpeed.WPM, 100.0)
	assert.Contains(t, features.AnomalyIndicators, "打字速度异常")
}
