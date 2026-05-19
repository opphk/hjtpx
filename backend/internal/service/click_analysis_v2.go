package service

import (
	"fmt"
	"math"
	"time"
)

type ClickAnalysisV2 struct{}

type ClickDataV2 struct {
	X         float64
	Y         float64
	Timestamp time.Time
}

type ClickAnalysisResultV2 struct {
	Timing       TimingAnalysisV2
	Spatial      SpatialAnalysisV2
	Rhythm       RhythmPatternV2
	Coordination float64
}

type TimingAnalysisV2 struct {
	AverageInterval float64
	Variance        float64
	Normal          bool
}

type SpatialAnalysisV2 struct {
	Grid               [][]int
	Entropy            float64
	CenterDensityRatio float64
}

type RhythmPatternV2 struct {
	Type        string
	Consistency float64
}

func NewClickAnalysisV2() *ClickAnalysisV2 {
	return &ClickAnalysisV2{}
}

func (c *ClickAnalysisV2) AnalyzeClickSequence(clicks []ClickDataV2) *ClickAnalysisResultV2 {
	if len(clicks) == 0 {
		return &ClickAnalysisResultV2{
			Timing:       TimingAnalysisV2{Normal: false},
			Spatial:      SpatialAnalysisV2{},
			Rhythm:       RhythmPatternV2{Type: "insufficient_data"},
			Coordination: 0,
		}
	}

	timingAnalysis := c.analyzeTimingV2(clicks)
	spatialAnalysis := c.analyzeSpatialV2(clicks)
	rhythmPattern := c.detectRhythmPatternV2(clicks)
	coordinationScore := c.detectCoordinationV2(clicks)

	return &ClickAnalysisResultV2{
		Timing:       timingAnalysis,
		Spatial:      spatialAnalysis,
		Rhythm:       rhythmPattern,
		Coordination: coordinationScore,
	}
}

func (c *ClickAnalysisV2) analyzeTimingV2(clicks []ClickDataV2) TimingAnalysisV2 {
	if len(clicks) < 2 {
		return TimingAnalysisV2{Normal: true}
	}

	var intervals []float64
	for i := 1; i < len(clicks); i++ {
		interval := clicks[i].Timestamp.Sub(clicks[i-1].Timestamp).Seconds() * 1000
		intervals = append(intervals, interval)
	}

	avgInterval := calculateAverageV2(intervals)
	variance := calculateVarianceV2(intervals, avgInterval)

	isNormal := true
	for _, interval := range intervals {
		if interval < 100 || interval > 3000 {
			isNormal = false
			break
		}
	}

	return TimingAnalysisV2{
		AverageInterval: avgInterval,
		Variance:        variance,
		Normal:          isNormal,
	}
}

func (c *ClickAnalysisV2) analyzeSpatialV2(clicks []ClickDataV2) SpatialAnalysisV2 {
	gridSize := 10
	grid := make([][]int, gridSize)
	for i := range grid {
		grid[i] = make([]int, gridSize)
	}

	for _, click := range clicks {
		x := int(click.X / 100 * float64(gridSize))
		y := int(click.Y / 100 * float64(gridSize))
		if x >= 0 && x < gridSize && y >= 0 && y < gridSize {
			grid[y][x]++
		}
	}

	entropy := c.calculateEntropyV2(grid)

	centerDensity := 0
	total := 0
	for y := 3; y < 7; y++ {
		for x := 3; x < 7; x++ {
			if y < len(grid) && x < len(grid[y]) {
				centerDensity += grid[y][x]
			}
		}
	}
	for _, row := range grid {
		for _, count := range row {
			total += count
		}
	}

	densityRatio := 0.0
	if total > 0 {
		densityRatio = float64(centerDensity) / float64(total)
	}

	return SpatialAnalysisV2{
		Grid:               grid,
		Entropy:            entropy,
		CenterDensityRatio: densityRatio,
	}
}

func (c *ClickAnalysisV2) detectRhythmPatternV2(clicks []ClickDataV2) RhythmPatternV2 {
	if len(clicks) < 3 {
		return RhythmPatternV2{Type: "insufficient_data"}
	}

	var intervals []float64
	for i := 1; i < len(clicks); i++ {
		interval := clicks[i].Timestamp.Sub(clicks[i-1].Timestamp).Seconds() * 1000
		intervals = append(intervals, interval)
	}

	avgInterval := calculateAverageV2(intervals)
	if avgInterval == 0 {
		return RhythmPatternV2{Type: "random", Consistency: 0}
	}

	variance := calculateVarianceV2(intervals, avgInterval)

	patternType := "random"
	if variance < 100 {
		patternType = "rhythmic"
	} else if variance < 500 {
		patternType = "regular"
	}

	consistency := 1.0 - math.Min(variance/avgInterval, 1.0)

	return RhythmPatternV2{
		Type:        patternType,
		Consistency: consistency,
	}
}

func (c *ClickAnalysisV2) detectCoordinationV2(clicks []ClickDataV2) float64 {
	if len(clicks) < 2 {
		return 0
	}

	simultaneousCount := 0
	for i := 1; i < len(clicks); i++ {
		timeDiff := math.Abs(float64(clicks[i].Timestamp.Sub(clicks[i-1].Timestamp).Milliseconds()))
		if timeDiff < 50 {
			simultaneousCount++
		}
	}

	coordination := 0.0
	if len(clicks) > 1 {
		coordination = float64(simultaneousCount) / float64(len(clicks)-1)
	}

	return coordination
}

func (c *ClickAnalysisV2) calculateEntropyV2(grid [][]int) float64 {
	total := 0
	for _, row := range grid {
		for _, count := range row {
			total += count
		}
	}

	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, row := range grid {
		for _, count := range row {
			if count > 0 {
				p := float64(count) / float64(total)
				entropy -= p * math.Log2(p)
			}
		}
	}

	return entropy
}

func calculateAverageV2(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVarianceV2(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func (c *ClickAnalysisV2) GenerateHeatmap(clicks []ClickDataV2, width, height float64) [][]float64 {
	gridSize := 20
	heatmap := make([][]float64, gridSize)
	for i := range heatmap {
		heatmap[i] = make([]float64, gridSize)
	}

	if len(clicks) == 0 {
		return heatmap
	}

	bandwidth := c.estimateBandwidthV2(clicks)
	if bandwidth == 0 {
		bandwidth = 20
	}

	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			gx := width * float64(i) / float64(gridSize-1)
			gy := height * float64(j) / float64(gridSize-1)

			density := 0.0
			for _, click := range clicks {
				dx := click.X - gx
				dy := click.Y - gy
				density += math.Exp(-(dx*dx + dy*dy) / (2 * bandwidth * bandwidth))
			}
			density /= float64(len(clicks)) * bandwidth * math.Sqrt(2*math.Pi)
			heatmap[i][j] = density
		}
	}

	return heatmap
}

func (c *ClickAnalysisV2) estimateBandwidthV2(clicks []ClickDataV2) float64 {
	n := len(clicks)
	if n < 2 {
		return 10.0
	}

	xValues := make([]float64, n)
	yValues := make([]float64, n)
	for i, click := range clicks {
		xValues[i] = click.X
		yValues[i] = click.Y
	}

	stdX := standardDeviationV2(xValues)
	stdY := standardDeviationV2(yValues)
	std := math.Max(stdX, stdY)

	if std == 0 {
		std = 10.0
	}

	return 1.06 * std * math.Pow(float64(n), -0.2)
}

func standardDeviationV2(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := calculateAverageV2(values)
	variance := calculateVarianceV2(values, mean)
	return math.Sqrt(variance)
}

func (c *ClickAnalysisV2) DetectBotPattern(result *ClickAnalysisResultV2) bool {
	botScore := 0.0

	if !result.Timing.Normal {
		botScore += 0.3
	}

	if result.Spatial.CenterDensityRatio > 0.8 {
		botScore += 0.25
	}

	if result.Rhythm.Type == "rhythmic" {
		botScore += 0.25
	}

	if result.Coordination > 0.5 {
		botScore += 0.2
	}

	return botScore > 0.5
}

func (c *ClickAnalysisV2) CalculateRiskScore(result *ClickAnalysisResultV2) float64 {
	riskScore := 0.0

	if !result.Timing.Normal {
		riskScore += result.Timing.Variance / 1000.0
	}

	if result.Spatial.CenterDensityRatio > 0.5 {
		riskScore += result.Spatial.CenterDensityRatio * 0.3
	}

	if result.Rhythm.Type == "rhythmic" {
		riskScore += result.Rhythm.Consistency * 0.3
	}

	riskScore += result.Coordination * 0.2

	return math.Min(riskScore, 1.0)
}

func (c *ClickAnalysisV2) AnalyzeMovementPattern(clicks []ClickDataV2) string {
	if len(clicks) < 2 {
		return "insufficient_data"
	}

	var velocities []float64
	for i := 1; i < len(clicks); i++ {
		dx := clicks[i].X - clicks[i-1].X
		dy := clicks[i].Y - clicks[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := clicks[i].Timestamp.Sub(clicks[i-1].Timestamp).Seconds()
		if dt > 0 {
			velocity := distance / dt
			velocities = append(velocities, velocity)
		}
	}

	if len(velocities) == 0 {
		return "stationary"
	}

	avgVelocity := calculateAverageV2(velocities)
	variance := calculateVarianceV2(velocities, avgVelocity)

	if avgVelocity < 10 {
		return "stationary"
	} else if variance < 100 {
		return "uniform"
	} else if variance < 500 {
		return "variable"
	} else {
		return "erratic"
	}
}

func (c *ClickAnalysisV2) DetectSuspiciousPatterns(clicks []ClickDataV2) []string {
	suspiciousPatterns := make([]string, 0)

	if len(clicks) < 2 {
		return suspiciousPatterns
	}

	result := c.AnalyzeClickSequence(clicks)

	if !result.Timing.Normal {
		suspiciousPatterns = append(suspiciousPatterns, "abnormal_timing")
	}

	if result.Spatial.CenterDensityRatio > 0.8 {
		suspiciousPatterns = append(suspiciousPatterns, "high_central_density")
	}

	if result.Rhythm.Type == "rhythmic" && result.Rhythm.Consistency > 0.95 {
		suspiciousPatterns = append(suspiciousPatterns, "mechanical_rhythm")
	}

	if result.Coordination > 0.7 {
		suspiciousPatterns = append(suspiciousPatterns, "simultaneous_clicks")
	}

	movementPattern := c.AnalyzeMovementPattern(clicks)
	if movementPattern == "uniform" && len(clicks) > 3 {
		suspiciousPatterns = append(suspiciousPatterns, "uniform_movement")
	}

	return suspiciousPatterns
}

func (c *ClickAnalysisV2) GenerateReport(result *ClickAnalysisResultV2, clicks []ClickDataV2) string {
	report := "=== 点击分析V2报告 ===\n\n"

	report += "时序分析:\n"
	report += fmt.Sprintf("  平均间隔: %.2f ms\n", result.Timing.AverageInterval)
	report += fmt.Sprintf("  方差: %.2f\n", result.Timing.Variance)
	report += fmt.Sprintf("  正常: %v\n", result.Timing.Normal)

	report += "\n空间分析:\n"
	report += fmt.Sprintf("  熵: %.2f\n", result.Spatial.Entropy)
	report += fmt.Sprintf("  中心密度比: %.2f\n", result.Spatial.CenterDensityRatio)

	report += "\n节奏模式:\n"
	report += fmt.Sprintf("  类型: %s\n", result.Rhythm.Type)
	report += fmt.Sprintf("  一致性: %.2f\n", result.Rhythm.Consistency)

	report += "\n协同检测:\n"
	report += fmt.Sprintf("  协同指数: %.2f\n", result.Coordination)

	report += "\n行为分析:\n"
	report += fmt.Sprintf("  点击次数: %d\n", len(clicks))
	report += fmt.Sprintf("  移动模式: %s\n", c.AnalyzeMovementPattern(clicks))

	suspicious := c.DetectSuspiciousPatterns(clicks)
	if len(suspicious) > 0 {
		report += "\n可疑模式:\n"
		for _, pattern := range suspicious {
			report += fmt.Sprintf("  - %s\n", pattern)
		}
	}

	report += fmt.Sprintf("\n风险评分: %.2f\n", c.CalculateRiskScore(result))
	report += fmt.Sprintf("  机器人检测: %v\n", c.DetectBotPattern(result))

	return report
}

func (c *ClickAnalysisV2) AnalyzeWithContext(clicks []ClickDataV2, context map[string]interface{}) *ClickAnalysisResultV2 {
	result := c.AnalyzeClickSequence(clicks)

	if ctx, ok := context["expected_duration"].(float64); ok {
		totalDuration := 0.0
		if len(clicks) > 1 {
			totalDuration = clicks[len(clicks)-1].Timestamp.Sub(clicks[0].Timestamp).Seconds() * 1000
		}
		if totalDuration > 0 && totalDuration < ctx*0.5 {
			result.Timing.Normal = false
		}
	}

	if ctx, ok := context["expected_targets"].(int); ok {
		if len(clicks) > ctx*2 {
			result.Coordination += 0.2
		}
	}

	return result
}

func (c *ClickAnalysisV2) CompareWithBaseline(result *ClickAnalysisResultV2, baseline *ClickAnalysisResultV2) map[string]float64 {
	comparison := make(map[string]float64)

	comparison["timing_deviation"] = math.Abs(result.Timing.AverageInterval - baseline.Timing.AverageInterval)
	comparison["spatial_deviation"] = math.Abs(result.Spatial.CenterDensityRatio - baseline.Spatial.CenterDensityRatio)
	comparison["rhythm_deviation"] = math.Abs(result.Rhythm.Consistency - baseline.Rhythm.Consistency)
	comparison["coordination_deviation"] = math.Abs(result.Coordination - baseline.Coordination)

	overallDeviation := 0.0
	weight := 1.0 / float64(len(comparison))
	for _, v := range comparison {
		overallDeviation += v * weight
	}
	comparison["overall_deviation"] = overallDeviation

	return comparison
}

func (c *ClickAnalysisV2) DetectAdvancedBotPatterns(clicks []ClickDataV2) map[string]float64 {
	patterns := make(map[string]float64)

	if len(clicks) < 3 {
		return patterns
	}

	result := c.AnalyzeClickSequence(clicks)

	if result.Timing.Variance < 50 && result.Timing.AverageInterval < 300 {
		patterns["mechanical_timing"] = 0.9
	}

	if result.Spatial.CenterDensityRatio > 0.9 {
		patterns["focused_attention"] = 0.85
	}

	angles := c.calculateClickAngles(clicks)
	angleVariance := calculateVarianceV2(angles, calculateAverageV2(angles))
	if angleVariance < 100 {
		patterns["linear_trajectory"] = 0.8
	}

	pressures := c.simulatePressureData(clicks)
	if len(pressures) > 0 {
		pressureVariance := calculateVarianceV2(pressures, calculateAverageV2(pressures))
		if pressureVariance < 10 {
			patterns["constant_pressure"] = 0.75
		}
	}

	return patterns
}

func (c *ClickAnalysisV2) calculateClickAngles(clicks []ClickDataV2) []float64 {
	angles := make([]float64, 0)
	for i := 1; i < len(clicks)-1; i++ {
		dx1 := clicks[i].X - clicks[i-1].X
		dy1 := clicks[i].Y - clicks[i-1].Y
		dx2 := clicks[i+1].X - clicks[i].X
		dy2 := clicks[i+1].Y - clicks[i].Y

		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			angle := math.Acos(math.Max(-1, math.Min(1, cosAngle)))
			angles = append(angles, angle*180/math.Pi)
		}
	}
	return angles
}

func (c *ClickAnalysisV2) simulatePressureData(clicks []ClickDataV2) []float64 {
	pressures := make([]float64, len(clicks))
	for i := range pressures {
		pressures[i] = 0.5 + (float64(i%3) * 0.1)
	}
	return pressures
}

func (c *ClickAnalysisV2) AnalyzeSessionPatterns(sessions [][]ClickDataV2) map[string]interface{} {
	sessionAnalysis := make(map[string]interface{})

	if len(sessions) == 0 {
		return sessionAnalysis
	}

	var avgClicksPerSession float64
	var avgDurationPerSession float64
	var totalRhythmicSessions int

	for _, session := range sessions {
		avgClicksPerSession += float64(len(session))

		if len(session) > 1 {
			duration := session[len(session)-1].Timestamp.Sub(session[0].Timestamp).Seconds()
			avgDurationPerSession += duration
		}

		result := c.AnalyzeClickSequence(session)
		if result.Rhythm.Type == "rhythmic" {
			totalRhythmicSessions++
		}
	}

	avgClicksPerSession /= float64(len(sessions))
	avgDurationPerSession /= float64(len(sessions))
	rhythmicRatio := float64(totalRhythmicSessions) / float64(len(sessions))

	sessionAnalysis["avg_clicks_per_session"] = avgClicksPerSession
	sessionAnalysis["avg_duration_per_session"] = avgDurationPerSession
	sessionAnalysis["rhythmic_session_ratio"] = rhythmicRatio
	sessionAnalysis["total_sessions"] = len(sessions)

	if rhythmicRatio > 0.7 {
		sessionAnalysis["bot_session_warning"] = true
	} else {
		sessionAnalysis["bot_session_warning"] = false
	}

	return sessionAnalysis
}
