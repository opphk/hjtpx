package service

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/hjtpx/hjtpx/internal/model"
)

type KeyboardAnalyzer struct {
	typingSpeedThreshold float64
	errorRateThreshold   float64
	rhythmThreshold      float64
}

func NewKeyboardAnalyzer() *KeyboardAnalyzer {
	return &KeyboardAnalyzer{
		typingSpeedThreshold: 150,
		errorRateThreshold:   0.15,
		rhythmThreshold:      0.95,
	}
}

func (k *KeyboardAnalyzer) AnalyzeKeyboardBehavior(data *model.KeyboardBehaviorData) *model.KeyboardBehaviorFeatures {
	features := &model.KeyboardBehaviorFeatures{
		AnomalyIndicators: make([]string, 0),
	}

	if data == nil || len(data.KeyEvents) == 0 {
		features.AddAnomalyIndicator("无键盘数据")
		features.OverallScore = 100
		features.IsHumanLike = false
		features.Confidence = 0.9
		features.RiskLevel = "high"
		return features
	}

	features.TypingSpeed = k.extractTypingSpeedFeatures(data.KeyEvents)
	features.ErrorRate = k.extractErrorRateFeatures(data.KeyEvents)
	features.Rhythm = k.extractRhythmFeatures(data.KeyEvents)
	features.ComboKeys = k.extractComboKeyFeatures(data.KeyEvents)

	features.CalculateOverallScore()

	return features
}

func (k *KeyboardAnalyzer) extractTypingSpeedFeatures(events []model.KeyEvent) model.TypingSpeedFeature {
	feature := model.TypingSpeedFeature{}

	intervals := k.calculateKeyIntervals(events)
	if len(intervals) == 0 {
		return feature
	}

	sortedIntervals := make([]float64, len(intervals))
	copy(sortedIntervals, intervals)
	sort.Float64s(sortedIntervals)

	feature.TotalCharacters = len(intervals) + 1
	feature.AverageInterval = k.mean(intervals)
	feature.MedianInterval = sortedIntervals[len(sortedIntervals)/2]
	feature.MaxInterval = k.max(intervals)
	feature.MinInterval = k.min(intervals)
	feature.IntervalVariance = k.variance(intervals)
	feature.IntervalStdDev = math.Sqrt(feature.IntervalVariance)
	feature.IntervalSkewness = k.skewness(intervals)
	feature.IntervalKurtosis = k.kurtosis(intervals)

	bursts := k.detectBursts(intervals)
	feature.BurstCount = len(bursts)
	if len(bursts) > 0 {
		sum := 0
		for _, b := range bursts {
			sum += b
		}
		feature.BurstAvgLength = float64(sum) / float64(len(bursts))
	}

	pauses := k.detectPauses(intervals)
	feature.PauseCount = len(pauses)
	if len(pauses) > 0 {
		sum := 0
		for _, p := range pauses {
			sum += p
		}
		feature.PauseAvgDuration = float64(sum) / float64(len(pauses))
	}

	feature.SpeedVariance = feature.IntervalVariance
	feature.SpeedStdDev = feature.IntervalStdDev

	firstHalf := intervals[:len(intervals)/2]
	secondHalf := intervals[len(intervals)/2:]
	firstAvg := k.mean(firstHalf)
	secondAvg := k.mean(secondHalf)

	feature.Accelerating = secondAvg < firstAvg*0.8
	feature.Decelerating = secondAvg > firstAvg*1.2

	feature.SpeedConsistency = 1.0 - (feature.IntervalStdDev / (feature.AverageInterval + 0.001))

	avgIntervalMs := feature.AverageInterval / 1000.0
	if avgIntervalMs > 0 {
		charsPerMinute := 60000.0 / avgIntervalMs
		feature.WPM = charsPerMinute / 5.0
	}

	return feature
}

func (k *KeyboardAnalyzer) extractErrorRateFeatures(events []model.KeyEvent) model.KeyboardErrorFeature {
	feature := model.KeyboardErrorFeature{}

	backspaceCount := 0
	deleteCount := 0
	totalKeystrokes := 0
	correctionCount := 0

	for i, event := range events {
		if event.EventType == "keydown" {
			totalKeystrokes++

			if event.Key == "Backspace" {
				backspaceCount++
			} else if event.Key == "Delete" {
				deleteCount++
			}

			if (event.Key == "Backspace" || event.Key == "Delete") && i > 0 {
				for j := i - 1; j >= 0; j-- {
					if events[j].EventType == "keydown" && events[j].Key != "Backspace" && events[j].Key != "Delete" {
						correctionCount++
						break
					}
				}
			}
		}
	}

	feature.BackspaceCount = backspaceCount
	feature.DeleteCount = deleteCount
	feature.TotalKeystrokes = totalKeystrokes
	feature.CorrectionCount = correctionCount

	if totalKeystrokes > 0 {
		feature.ErrorRate = float64(backspaceCount+deleteCount) / float64(totalKeystrokes)
		feature.CorrectionRatio = float64(correctionCount) / float64(totalKeystrokes)
		feature.BackspaceRatio = float64(backspaceCount) / float64(totalKeystrokes)
	}

	errorBursts := k.detectErrorBursts(events)
	feature.ErrorBurstCount = len(errorBursts)
	if len(errorBursts) > 0 {
		sum := 0
		for _, b := range errorBursts {
			sum += b
		}
		feature.ErrorBurstAvgSize = float64(sum) / float64(len(errorBursts))
	}

	feature.ImmediateCorrection = k.countImmediateCorrections(events)
	feature.DelayedCorrection = k.countDelayedCorrections(events)

	feature.AccuracyScore = 1.0 - feature.ErrorRate

	return feature
}

func (k *KeyboardAnalyzer) extractRhythmFeatures(events []model.KeyEvent) model.KeyboardRhythmFeature {
	feature := model.KeyboardRhythmFeature{}

	intervals := k.calculateKeyIntervals(events)
	if len(intervals) == 0 {
		return feature
	}

	intervalsInMs := make([]float64, len(intervals))
	for i, interval := range intervals {
		intervalsInMs[i] = interval / 1000.0
	}

	feature.IntervalSequence = intervalsInMs
	feature.AverageRhythm = k.mean(intervalsInMs)
	feature.RhythmVariance = k.variance(intervalsInMs)
	feature.RhythmStdDev = math.Sqrt(feature.RhythmVariance)

	if feature.AverageRhythm > 0 {
		feature.RhythmRegularity = 1.0 - (feature.RhythmStdDev / feature.AverageRhythm)
	}

	feature.RhythmEntropy = k.calculateEntropy(intervalsInMs)

	feature.PeakCount = k.countPeaks(intervalsInMs)
	feature.ValleyCount = k.countValleys(intervalsInMs)

	feature.PatternComplexity = k.calculatePatternComplexity(intervalsInMs)
	feature.PatternRepetition = k.calculatePatternRepetition(intervalsInMs)

	feature.Autocorrelation = k.calculateAutocorrelation(intervalsInMs)

	feature.FastSegments = k.detectFastSegments(intervalsInMs)
	feature.SlowSegments = k.detectSlowSegments(intervalsInMs)

	feature.RhythmChanges = k.countRhythmChanges(intervalsInMs)

	return feature
}

func (k *KeyboardAnalyzer) extractComboKeyFeatures(events []model.KeyEvent) model.ComboKeyFeature {
	feature := model.ComboKeyFeature{}

	feature.CommonCombos = make(map[string]int)
	feature.HoldDuration = make(map[string]float64)

	modifierKeys := map[string]bool{
		"Control": true,
		"Alt":     true,
		"Shift":   true,
		"Meta":    true,
	}

	for i := 0; i < len(events); i++ {
		event := events[i]
		if event.EventType != "keydown" {
			continue
		}

		if !modifierKeys[event.Key] {
			modifiers := k.getActiveModifiers(events, i)
			if len(modifiers) > 0 {
				combo := strings.Join(append(modifiers, event.Key), "+")
				feature.CommonCombos[combo]++

				switch {
				case strings.Contains(combo, "Control"):
					feature.CtrlCombos++
				case strings.Contains(combo, "Alt"):
					feature.AltCombos++
				case strings.Contains(combo, "Shift"):
					feature.ShiftCombos++
				case strings.Contains(combo, "Meta"):
					feature.MetaCombos++
				}

				feature.TotalCombos++

				pattern := model.ComboPattern{
					Pattern: combo,
					Count:   feature.CommonCombos[combo],
				}
				feature.ComboPatterns = append(feature.ComboPatterns, pattern)
			}
		}

		if modifierKeys[event.Key] {
			holdDuration := k.calculateHoldDuration(events, i, event.Key)
			feature.HoldDuration[event.Key] = holdDuration
		}
	}

	if len(events) > 0 {
		feature.ComboFrequency = float64(feature.TotalCombos) / float64(len(events))
	}

	modifierCount := 0
	for _, event := range events {
		if modifierKeys[event.Key] && event.EventType == "keydown" {
			modifierCount++
		}
	}
	if len(events) > 0 {
		feature.ModifierUsageRate = float64(modifierCount) / float64(len(events))
	}

	feature.SimultaneousPress = k.countSimultaneousPress(events)
	feature.SequentialPress = k.countSequentialPress(events)

	return feature
}

func (k *KeyboardAnalyzer) calculateKeyIntervals(events []model.KeyEvent) []float64 {
	var intervals []float64
	var lastKeydownTime int64

	for _, event := range events {
		if event.EventType == "keydown" {
			if lastKeydownTime > 0 {
				intervals = append(intervals, float64(event.Timestamp-lastKeydownTime))
			}
			lastKeydownTime = event.Timestamp
		}
	}

	return intervals
}

func (k *KeyboardAnalyzer) detectBursts(intervals []float64) []int {
	var bursts []int
	currentBurst := 0
	burstThreshold := 50.0

	for _, interval := range intervals {
		if interval < burstThreshold {
			currentBurst++
		} else {
			if currentBurst >= 3 {
				bursts = append(bursts, currentBurst)
			}
			currentBurst = 0
		}
	}

	if currentBurst >= 3 {
		bursts = append(bursts, currentBurst)
	}

	return bursts
}

func (k *KeyboardAnalyzer) detectPauses(intervals []float64) []int {
	var pauses []int
	pauseThreshold := 500.0

	for _, interval := range intervals {
		if interval > pauseThreshold {
			pauses = append(pauses, int(interval))
		}
	}

	return pauses
}

func (k *KeyboardAnalyzer) detectErrorBursts(events []model.KeyEvent) []int {
	var bursts []int
	currentBurst := 0

	for _, event := range events {
		if event.EventType == "keydown" {
			if event.Key == "Backspace" || event.Key == "Delete" {
				currentBurst++
			} else {
				if currentBurst >= 2 {
					bursts = append(bursts, currentBurst)
				}
				currentBurst = 0
			}
		}
	}

	if currentBurst >= 2 {
		bursts = append(bursts, currentBurst)
	}

	return bursts
}

func (k *KeyboardAnalyzer) countImmediateCorrections(events []model.KeyEvent) int {
	count := 0

	for i := 0; i < len(events)-1; i++ {
		if events[i].EventType == "keydown" && events[i].Key != "Backspace" && events[i].Key != "Delete" {
			if events[i+1].EventType == "keydown" && (events[i+1].Key == "Backspace" || events[i+1].Key == "Delete") {
				count++
			}
		}
	}

	return count
}

func (k *KeyboardAnalyzer) countDelayedCorrections(events []model.KeyEvent) int {
	count := 0
	lastCharTime := int64(0)

	for _, event := range events {
		if event.EventType == "keydown" && event.Key != "Backspace" && event.Key != "Delete" {
			lastCharTime = event.Timestamp
		}
		if event.EventType == "keydown" && (event.Key == "Backspace" || event.Key == "Delete") {
			if event.Timestamp-lastCharTime > 300 {
				count++
			}
		}
	}

	return count - count/2
}

func (k *KeyboardAnalyzer) calculateEntropy(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	buckets := 10
	bucketSize := (k.max(values) - k.min(values)) / float64(buckets)
	if bucketSize == 0 {
		return 0
	}

	freq := make([]int, buckets)
	for _, v := range values {
		bucket := int((v - k.min(values)) / bucketSize)
		if bucket >= buckets {
			bucket = buckets - 1
		}
		freq[bucket]++
	}

	entropy := 0.0
	n := float64(len(values))
	for _, f := range freq {
		if f > 0 {
			p := float64(f) / n
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (k *KeyboardAnalyzer) countPeaks(values []float64) int {
	if len(values) < 3 {
		return 0
	}

	peaks := 0
	threshold := k.mean(values)

	for i := 1; i < len(values)-1; i++ {
		if values[i] > values[i-1] && values[i] > values[i+1] && values[i] > threshold {
			peaks++
		}
	}

	return peaks
}

func (k *KeyboardAnalyzer) countValleys(values []float64) int {
	if len(values) < 3 {
		return 0
	}

	valleys := 0
	threshold := k.mean(values)

	for i := 1; i < len(values)-1; i++ {
		if values[i] < values[i-1] && values[i] < values[i+1] && values[i] < threshold {
			valleys++
		}
	}

	return valleys
}

func (k *KeyboardAnalyzer) calculatePatternComplexity(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	uniqueValues := make(map[float64]bool)
	for _, v := range values {
		uniqueValues[math.Round(v*100)/100] = true
	}

	return float64(len(uniqueValues)) / float64(len(values))
}

func (k *KeyboardAnalyzer) calculatePatternRepetition(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}

	repetitions := 0
	windowSize := 3

	for i := 0; i < len(values)-windowSize*2; i++ {
		match := true
		for j := 0; j < windowSize; j++ {
			if math.Abs(values[i+j]-values[i+windowSize+j]) > 5 {
				match = false
				break
			}
		}
		if match {
			repetitions++
		}
	}

	return float64(repetitions) / float64(len(values)-windowSize*2+1)
}

func (k *KeyboardAnalyzer) calculateAutocorrelation(values []float64) []float64 {
	if len(values) < 2 {
		return []float64{}
	}

	mean := k.mean(values)
	var correlations []float64

	for lag := 1; lag <= int(float64(len(values))*0.5); lag++ {
		sum := 0.0
		count := 0
		for i := 0; i < len(values)-lag; i++ {
			sum += (values[i] - mean) * (values[i+lag] - mean)
			count++
		}
		if count > 0 {
			correlations = append(correlations, sum/float64(count))
		}
	}

	return correlations
}

func (k *KeyboardAnalyzer) detectFastSegments(values []float64) []model.SegmentInfo {
	var segments []model.SegmentInfo
	threshold := k.mean(values) * 0.7

	start := -1
	for i, v := range values {
		if v < threshold {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 && i-start >= 2 {
				segmentValues := values[start:i]
				segments = append(segments, model.SegmentInfo{
					StartIndex:  start,
					EndIndex:    i - 1,
					AvgInterval: k.mean(segmentValues),
					Type:        "fast",
				})
			}
			start = -1
		}
	}

	if start != -1 && len(values)-start >= 2 {
		segmentValues := values[start:]
		segments = append(segments, model.SegmentInfo{
			StartIndex:  start,
			EndIndex:    len(values) - 1,
			AvgInterval: k.mean(segmentValues),
			Type:        "fast",
		})
	}

	return segments
}

func (k *KeyboardAnalyzer) detectSlowSegments(values []float64) []model.SegmentInfo {
	var segments []model.SegmentInfo
	threshold := k.mean(values) * 1.5

	start := -1
	for i, v := range values {
		if v > threshold {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 && i-start >= 2 {
				segmentValues := values[start:i]
				segments = append(segments, model.SegmentInfo{
					StartIndex:  start,
					EndIndex:    i - 1,
					AvgInterval: k.mean(segmentValues),
					Type:        "slow",
				})
			}
			start = -1
		}
	}

	if start != -1 && len(values)-start >= 2 {
		segmentValues := values[start:]
		segments = append(segments, model.SegmentInfo{
			StartIndex:  start,
			EndIndex:    len(values) - 1,
			AvgInterval: k.mean(segmentValues),
			Type:        "slow",
		})
	}

	return segments
}

func (k *KeyboardAnalyzer) countRhythmChanges(values []float64) int {
	if len(values) < 3 {
		return 0
	}

	changes := 0
	prevTrend := 0

	for i := 1; i < len(values)-1; i++ {
		trend := 0
		if values[i] > values[i-1] {
			trend = 1
		} else if values[i] < values[i-1] {
			trend = -1
		}

		if prevTrend != 0 && trend != 0 && trend != prevTrend {
			changes++
		}
		prevTrend = trend
	}

	return changes
}

func (k *KeyboardAnalyzer) getActiveModifiers(events []model.KeyEvent, currentIndex int) []string {
	var modifiers []string
	currentTime := events[currentIndex].Timestamp

	modifierKeys := map[string]bool{
		"Control": true,
		"Alt":     true,
		"Shift":   true,
		"Meta":    true,
	}

	for i := currentIndex - 1; i >= 0 && i >= currentIndex-5; i-- {
		event := events[i]
		if !modifierKeys[event.Key] {
			continue
		}

		if currentTime-event.Timestamp > 500 {
			break
		}

		if event.EventType == "keydown" {
			modifiers = append(modifiers, event.Key)
		} else if event.EventType == "keyup" {
			break
		}
	}

	return modifiers
}

func (k *KeyboardAnalyzer) calculateHoldDuration(events []model.KeyEvent, keyIndex int, keyName string) float64 {
	keydownTime := events[keyIndex].Timestamp
	var keyupTime int64

	for i := keyIndex + 1; i < len(events); i++ {
		if events[i].Key == keyName && events[i].EventType == "keyup" {
			keyupTime = events[i].Timestamp
			break
		}
	}

	if keyupTime == 0 {
		return 0
	}

	return float64(keyupTime - keydownTime)
}

func (k *KeyboardAnalyzer) countSimultaneousPress(events []model.KeyEvent) int {
	count := 0
	timeWindow := 30.0

	for i := 0; i < len(events)-1; i++ {
		if events[i].EventType != "keydown" {
			continue
		}

		for j := i + 1; j < len(events) && float64(events[j].Timestamp-events[i].Timestamp) < timeWindow; j++ {
			if events[j].EventType == "keydown" && events[j].Key != events[i].Key {
				count++
				break
			}
		}
	}

	return count
}

func (k *KeyboardAnalyzer) countSequentialPress(events []model.KeyEvent) int {
	count := 0
	prevKey := ""

	for _, event := range events {
		if event.EventType == "keydown" && event.Key != prevKey {
			count++
			prevKey = event.Key
		}
	}

	return count
}

func (k *KeyboardAnalyzer) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (k *KeyboardAnalyzer) max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (k *KeyboardAnalyzer) min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (k *KeyboardAnalyzer) variance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := k.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - m) * (v - m)
	}
	return sum / float64(len(values))
}

func (k *KeyboardAnalyzer) skewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	m := k.mean(values)
	stdDev := math.Sqrt(k.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-m)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func (k *KeyboardAnalyzer) kurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}
	m := k.mean(values)
	stdDev := math.Sqrt(k.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-m)/stdDev, 4)
	}
	return sum/float64(len(values)) - 3
}

func (k *KeyboardAnalyzer) CalculateRiskScore(features *model.KeyboardBehaviorFeatures) float64 {
	riskScore := 0.0

	if features.TypingSpeed.WPM < 20 || features.TypingSpeed.WPM > 150 {
		riskScore += 30
	}

	if features.ErrorRate.ErrorRate > 0.15 {
		riskScore += 25
	}

	if features.Rhythm.RhythmRegularity > 0.95 {
		riskScore += 20
	}

	if len(features.AnomalyIndicators) > 3 {
		riskScore += 15
	}

	return math.Min(100, riskScore)
}

func (k *KeyboardAnalyzer) IsBotBehavior(features *model.KeyboardBehaviorFeatures) bool {
	riskScore := k.CalculateRiskScore(features)
	return riskScore > 50
}

func (k *KeyboardAnalyzer) GenerateReport(features *model.KeyboardBehaviorFeatures) string {
	var report strings.Builder

	report.WriteString(fmt.Sprintf("键盘行为分析报告\n"))
	report.WriteString(fmt.Sprintf("=================\n\n"))

	report.WriteString(fmt.Sprintf("打字速度分析:\n"))
	report.WriteString(fmt.Sprintf("  - WPM: %.2f\n", features.TypingSpeed.WPM))
	report.WriteString(fmt.Sprintf("  - 平均间隔: %.2f ms\n", features.TypingSpeed.AverageInterval))
	report.WriteString(fmt.Sprintf("  - 速度一致性: %.2f%%\n", features.TypingSpeed.SpeedConsistency*100))

	report.WriteString(fmt.Sprintf("\n错误率分析:\n"))
	report.WriteString(fmt.Sprintf("  - 回退键次数: %d\n", features.ErrorRate.BackspaceCount))
	report.WriteString(fmt.Sprintf("  - 错误率: %.2f%%\n", features.ErrorRate.ErrorRate*100))
	report.WriteString(fmt.Sprintf("  - 准确度: %.2f%%\n", features.ErrorRate.AccuracyScore*100))

	report.WriteString(fmt.Sprintf("\n节奏分析:\n"))
	report.WriteString(fmt.Sprintf("  - 节奏规律性: %.2f%%\n", features.Rhythm.RhythmRegularity*100))
	report.WriteString(fmt.Sprintf("  - 节奏熵: %.2f\n", features.Rhythm.RhythmEntropy))
	report.WriteString(fmt.Sprintf("  - 节奏变化次数: %d\n", features.Rhythm.RhythmChanges))

	report.WriteString(fmt.Sprintf("\n组合键分析:\n"))
	report.WriteString(fmt.Sprintf("  - Ctrl组合: %d\n", features.ComboKeys.CtrlCombos))
	report.WriteString(fmt.Sprintf("  - Alt组合: %d\n", features.ComboKeys.AltCombos))
	report.WriteString(fmt.Sprintf("  - Shift组合: %d\n", features.ComboKeys.ShiftCombos))

	report.WriteString(fmt.Sprintf("\n综合评分:\n"))
	report.WriteString(fmt.Sprintf("  - 总体评分: %.2f\n", features.OverallScore))
	report.WriteString(fmt.Sprintf("  - 置信度: %.2f%%\n", features.Confidence*100))
	report.WriteString(fmt.Sprintf("  - 风险等级: %s\n", features.RiskLevel))

	if len(features.AnomalyIndicators) > 0 {
		report.WriteString(fmt.Sprintf("\n异常指标:\n"))
		for _, indicator := range features.AnomalyIndicators {
			report.WriteString(fmt.Sprintf("  - %s\n", indicator))
		}
	}

	return report.String()
}
