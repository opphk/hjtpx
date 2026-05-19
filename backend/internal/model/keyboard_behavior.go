package model

import (
	"math"
	"time"
)

type KeyboardBehaviorData struct {
	SessionID     string     `json:"session_id"`
	UserID        string     `json:"user_id"`
	KeyEvents     []KeyEvent `json:"key_events"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       time.Time  `json:"end_time"`
	TotalDuration int64      `json:"total_duration"`
}

type KeyEvent struct {
	KeyCode    int      `json:"key_code"`
	Key        string   `json:"key"`
	EventType  string   `json:"event_type"`
	Timestamp  int64    `json:"timestamp"`
	IsModifier bool     `json:"is_modifier"`
	Modifiers  []string `json:"modifiers"`
	Location   int      `json:"location"`
	IsRepeat   bool     `json:"is_repeat"`
}

type TypingSpeedFeature struct {
	TotalCharacters  int     `json:"total_characters"`
	WPM              float64 `json:"wpm"`
	AverageInterval  float64 `json:"average_interval"`
	MedianInterval   float64 `json:"median_interval"`
	MaxInterval      float64 `json:"max_interval"`
	MinInterval      float64 `json:"min_interval"`
	IntervalVariance float64 `json:"interval_variance"`
	IntervalStdDev   float64 `json:"interval_std_dev"`
	IntervalSkewness float64 `json:"interval_skewness"`
	IntervalKurtosis float64 `json:"interval_kurtosis"`
	BurstCount       int     `json:"burst_count"`
	BurstAvgLength   float64 `json:"burst_avg_length"`
	PauseCount       int     `json:"pause_count"`
	PauseAvgDuration float64 `json:"pause_avg_duration"`
	SpeedVariance    float64 `json:"speed_variance"`
	SpeedStdDev      float64 `json:"speed_std_dev"`
	Accelerating     bool    `json:"accelerating"`
	Decelerating     bool    `json:"decelerating"`
	SpeedConsistency float64 `json:"speed_consistency"`
}

type KeyboardErrorFeature struct {
	BackspaceCount      int      `json:"backspace_count"`
	DeleteCount         int      `json:"delete_count"`
	TotalKeystrokes     int      `json:"total_keystrokes"`
	ErrorRate           float64  `json:"error_rate"`
	CorrectionCount     int      `json:"correction_count"`
	CorrectionRatio     float64  `json:"correction_ratio"`
	BackspaceRatio      float64  `json:"backspace_ratio"`
	ErrorBurstCount     int      `json:"error_burst_count"`
	ErrorBurstAvgSize   float64  `json:"error_burst_avg_size"`
	ImmediateCorrection int      `json:"immediate_correction"`
	DelayedCorrection   int      `json:"delayed_correction"`
	ErrorPatterns       []string `json:"error_patterns"`
	AccuracyScore       float64  `json:"accuracy_score"`
}

type KeyboardRhythmFeature struct {
	IntervalSequence   []float64     `json:"interval_sequence"`
	AverageRhythm      float64       `json:"average_rhythm"`
	RhythmVariance     float64       `json:"rhythm_variance"`
	RhythmStdDev       float64       `json:"rhythm_std_dev"`
	RhythmRegularity   float64       `json:"rhythm_regularity"`
	RhythmEntropy      float64       `json:"rhythm_entropy"`
	PeakCount          int           `json:"peak_count"`
	ValleyCount        int           `json:"valley_count"`
	PatternComplexity  float64       `json:"pattern_complexity"`
	PatternRepetition  float64       `json:"pattern_repetition"`
	Autocorrelation    []float64     `json:"autocorrelation"`
	FastSegments       []SegmentInfo `json:"fast_segments"`
	SlowSegments       []SegmentInfo `json:"slow_segments"`
	RhythmChanges      int           `json:"rhythm_changes"`
}

type SegmentInfo struct {
	StartIndex  int     `json:"start_index"`
	EndIndex    int     `json:"end_index"`
	AvgInterval float64 `json:"avg_interval"`
	Type        string  `json:"type"`
}

type ComboKeyFeature struct {
	TotalCombos       int                  `json:"total_combos"`
	CtrlCombos       int                  `json:"ctrl_combos"`
	AltCombos        int                  `json:"alt_combos"`
	ShiftCombos      int                  `json:"shift_combos"`
	MetaCombos       int                  `json:"meta_combos"`
	ComboPatterns    []ComboPattern       `json:"combo_patterns"`
	CommonCombos     map[string]int       `json:"common_combos"`
	ComboFrequency   float64              `json:"combo_frequency"`
	AvgComboInterval float64              `json:"avg_combo_interval"`
	ModifierUsageRate float64             `json:"modifier_usage_rate"`
	HoldDuration     map[string]float64   `json:"hold_duration"`
	SimultaneousPress int                 `json:"simultaneous_press"`
	SequentialPress   int                 `json:"sequential_press"`
}

type ComboPattern struct {
	Pattern   string  `json:"pattern"`
	Count     int     `json:"count"`
	Frequency float64 `json:"frequency"`
}

type KeyboardBehaviorFeatures struct {
	TypingSpeed       TypingSpeedFeature   `json:"typing_speed"`
	ErrorRate         KeyboardErrorFeature `json:"error_rate"`
	Rhythm            KeyboardRhythmFeature `json:"rhythm"`
	ComboKeys         ComboKeyFeature       `json:"combo_keys"`
	OverallScore      float64              `json:"overall_score"`
	IsHumanLike       bool                 `json:"is_human_like"`
	Confidence        float64              `json:"confidence"`
	AnomalyIndicators []string             `json:"anomaly_indicators"`
	RiskLevel         string               `json:"risk_level"`
}

func (k *KeyboardBehaviorFeatures) CalculateOverallScore() {
	var score float64 = 100

	if k.TypingSpeed.WPM < 20 || k.TypingSpeed.WPM > 150 {
		score -= 20
		k.AnomalyIndicators = append(k.AnomalyIndicators, "打字速度异常")
	}

	if k.ErrorRate.ErrorRate > 0.15 {
		score -= 15
		k.AnomalyIndicators = append(k.AnomalyIndicators, "错误率较高")
	}

	if k.Rhythm.RhythmRegularity > 0.95 {
		score -= 10
		k.AnomalyIndicators = append(k.AnomalyIndicators, "节奏过于规律")
	}

	if k.ComboKeys.ModifierUsageRate < 0.01 && k.ComboKeys.TotalCombos < 2 {
		score -= 5
	}

	k.OverallScore = math.Max(0, score)
	k.Confidence = 0.9
	k.IsHumanLike = score > 50

	if score < 30 {
		k.RiskLevel = "high"
	} else if score < 60 {
		k.RiskLevel = "medium"
	} else {
		k.RiskLevel = "low"
	}
}

func (k *KeyboardBehaviorFeatures) AddAnomalyIndicator(indicator string) {
	if k.AnomalyIndicators == nil {
		k.AnomalyIndicators = make([]string, 0)
	}
	k.AnomalyIndicators = append(k.AnomalyIndicators, indicator)
}
