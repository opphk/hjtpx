package service

import (
	"testing"
	"time"
)

func TestExtractEnhancedSpeedFeatures(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 400, Y: 20, Timestamp: 400},
		{X: 500, Y: 25, Timestamp: 500},
	}

	features := extractor.extractEnhancedSpeedFeatures(trajectory)

	if features == nil {
		t.Fatal("期望返回非空特征映射")
	}

	if features["speed_change_rate"] < 0 {
		t.Errorf("速度变化率不应为负数: %f", features["speed_change_rate"])
	}

	t.Logf("速度特征提取测试通过: %+v", features)
}

func TestExtractSpeedSegments(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 50, Y: 2, Timestamp: 50},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 150, Y: 8, Timestamp: 150},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 250, Y: 12, Timestamp: 250},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 350, Y: 18, Timestamp: 350},
		{X: 400, Y: 20, Timestamp: 400},
		{X: 450, Y: 22, Timestamp: 450},
		{X: 500, Y: 25, Timestamp: 500},
	}

	segments := extractor.extractSpeedSegments(trajectory)

	if len(segments) == 0 {
		t.Fatal("应该提取到至少一个速度段")
	}

	for i, segment := range segments {
		if segment.StartIndex >= segment.EndIndex {
			t.Errorf("段 %d: 起始索引应小于结束索引", i)
		}
		if segment.AverageSpeed < 0 {
			t.Errorf("段 %d: 平均速度不应为负数", i)
		}
		t.Logf("段 %d: 起始=%d, 结束=%d, 平均速度=%.2f, 趋势=%s",
			i, segment.StartIndex, segment.EndIndex, segment.AverageSpeed, segment.Trend)
	}
}

func TestDetectSpeedAnomalies(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	normalSpeeds := []float64{500, 520, 480, 510, 490, 505, 495, 515, 505, 490}
	anomalies := extractor.detectSpeedAnomalies(normalSpeeds)
	t.Logf("正常速度序列检测到的异常数: %d", anomalies)

	extremeSpeeds := []float64{500, 510, 490, 500, 510, 2000, 510, 490, 500, 510}
	anomalies = extractor.detectSpeedAnomalies(extremeSpeeds)
	t.Logf("极端速度序列检测到的异常数: %d", anomalies)
	
	hasAnomalyDetection := anomalies > 0
	t.Logf("速度异常检测功能是否工作: %v", hasAnomalyDetection)
}

func TestExtractEnhancedCurvatureFeatures(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 10, Timestamp: 100},
		{X: 200, Y: 5, Timestamp: 200},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 400, Y: 8, Timestamp: 400},
		{X: 500, Y: 20, Timestamp: 500},
	}

	features := extractor.extractEnhancedCurvatureFeatures(trajectory)

	if features == nil {
		t.Fatal("应该返回曲率特征")
	}

	pattern := features["curvature_pattern"]
	if pattern == "" {
		t.Error("曲率模式不应为空")
	}

	t.Logf("曲率特征: %+v", features)
}

func TestCountCurvaturePeaks(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	curvatures := []float64{0.1, 0.15, 0.5, 0.2, 0.1, 0.6, 0.15, 0.1}
	peaks := extractor.countCurvaturePeaks(curvatures)

	if peaks < 1 {
		t.Errorf("应该检测到至少一个曲率峰值，实际: %d", peaks)
	}

	t.Logf("检测到曲率峰值: %d", peaks)
}

func TestClassifyCurvaturePattern(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	testCases := []struct {
		name       string
		curvatures []float64
		expected   string
	}{
		{"均匀曲率", []float64{0.1, 0.1, 0.1, 0.1}, "uniform"},
		{"轻微弯曲", []float64{0.05, 0.06, 0.07, 0.06, 0.05}, "uniform"},
		{"适度弯曲", []float64{0.3, 0.25, 0.35, 0.28, 0.32}, "moderately_curved"},
		{"高度弯曲", []float64{0.7, 0.65, 0.75, 0.68, 0.72}, "highly_curved"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern := extractor.classifyCurvaturePattern(tc.curvatures)
			if pattern != tc.expected {
				t.Errorf("期望模式: %s, 实际: %s", tc.expected, pattern)
			}
		})
	}
}

func TestExtractEnhancedBacktrackFeatures(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 150, Y: 8, Timestamp: 250},
		{X: 180, Y: 12, Timestamp: 300},
		{X: 300, Y: 15, Timestamp: 400},
		{X: 400, Y: 18, Timestamp: 500},
		{X: 350, Y: 20, Timestamp: 550},
		{X: 500, Y: 25, Timestamp: 600},
	}

	features := extractor.extractEnhancedBacktrackFeatures(trajectory)

	if features == nil {
		t.Fatal("应该返回回退特征")
	}

	depth, ok := features["backtrack_depth"].(float64)
	if !ok {
		t.Fatal("回退深度应该是浮点数")
	}
	if depth < 0 {
		t.Errorf("回退深度不应为负数: %f", depth)
	}

	patterns, ok := features["patterns"].([]BacktrackPattern)
	if !ok {
		t.Fatal("回退模式应该是BacktrackPattern数组")
	}

	t.Logf("回退特征 - 深度: %.2f, 模式数: %d", depth, len(patterns))
}

func TestDetectBacktrackPatterns(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 150, Y: 8, Timestamp: 300},
		{X: 180, Y: 12, Timestamp: 400},
		{X: 300, Y: 15, Timestamp: 500},
	}

	patterns := extractor.detectBacktrackPatterns(trajectory)

	if len(patterns) == 0 {
		t.Error("应该检测到至少一个回退模式")
	}

	for i, pattern := range patterns {
		if pattern.MaxDepth <= 0 {
			t.Errorf("模式 %d: 最大深度应大于0", i)
		}
		if pattern.StartIndex >= pattern.EndIndex {
			t.Errorf("模式 %d: 起始索引应小于结束索引", i)
		}
		t.Logf("回退模式 %d: 深度=%.2f, 类型=%s, 距离=%.2f",
			i, pattern.MaxDepth, pattern.PatternType, pattern.Distance)
	}
}

func TestAnalyzeSliderTrajectoryWithLogs(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 50, Y: 2, Timestamp: 50},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 150, Y: 8, Timestamp: 150},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 250, Y: 12, Timestamp: 250},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 350, Y: 18, Timestamp: 350},
		{X: 400, Y: 20, Timestamp: 400},
		{X: 450, Y: 22, Timestamp: 450},
		{X: 500, Y: 25, Timestamp: 500},
	}

	result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
	if err != nil {
		t.Fatalf("分析失败: %v", err)
	}

	if result == nil {
		t.Fatal("结果不应为空")
	}

	if result.AnalysisLogs == nil || len(result.AnalysisLogs) == 0 {
		t.Error("应该包含分析日志")
	}

	t.Logf("分析完成，判定为机器人: %v, 置信度: %.4f, 日志数: %d",
		result.IsBot, result.Confidence, len(result.AnalysisLogs))

	for i, log := range result.AnalysisLogs {
		t.Logf("日志 %d: [%s] %s - %s", i, log.Level, log.Message, log.Description)
	}
}

func TestSpeedChangeRateCalculation(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 250, Y: 10, Timestamp: 200},
		{X: 450, Y: 15, Timestamp: 300},
		{X: 700, Y: 20, Timestamp: 400},
	}

	speeds := extractor.extractSpeeds(trajectory)
	changeRate := extractor.calculateSpeedChangeRate(speeds, trajectory)

	if changeRate < 0 {
		t.Errorf("速度变化率不应为负数: %f", changeRate)
	}

	t.Logf("速度变化率: %f", changeRate)
}

func TestSpeedSkewnessCalculation(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	speeds := []float64{500, 520, 480, 510, 490, 505, 495, 515, 505, 490}
	skewness := extractor.calculateSpeedSkewness(speeds)

	t.Logf("速度偏度: %f", skewness)

	if skewness < -3 || skewness > 3 {
		t.Errorf("速度偏度应在合理范围内: %f", skewness)
	}
}

func TestSpeedKurtosisCalculation(t *testing.T) {
	extractor := NewSliderFeatureExtractor()

	speeds := []float64{500, 520, 480, 510, 490, 505, 495, 515, 505, 490}
	kurtosis := extractor.calculateSpeedKurtosisEnhanced(speeds)

	t.Logf("速度峰度: %f", kurtosis)
}

func TestBacktrackTypeClassification(t *testing.T) {
	testCases := []struct {
		name     string
		depth    float64
		duration int64
		expected string
	}{
		{"微回退", 5.0, 100, "micro"},
		{"小回退", 20.0, 100, "small"},
		{"快速回退", 40.0, 80, "quick"},
		{"犹豫回退", 40.0, 600, "hesitant"},
		{"正常回退", 40.0, 300, "normal"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := classifyBacktrackType(tc.depth, tc.duration)
			if result != tc.expected {
				t.Errorf("期望类型: %s, 实际: %s", tc.expected, result)
			}
		})
	}
}

func TestAnalyzeAdvancedFeatures(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 10, Timestamp: 100},
		{X: 200, Y: 5, Timestamp: 200},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 400, Y: 8, Timestamp: 400},
		{X: 500, Y: 20, Timestamp: 500},
	}

	features := analyzer.AnalyzeAdvancedFeatures(trajectory, 500)

	if features == nil {
		t.Fatal("高级特征不应为空")
	}

	if features["acceleration_mean"] == 0 && features["acceleration_std"] == 0 {
		t.Log("注意: 加速度特征可能未计算")
	}

	t.Logf("高级特征: %+v", features)
}

func TestCalculateAdvancedBotScore(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	botTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 50},
		{X: 200, Y: 0, Timestamp: 100},
		{X: 300, Y: 0, Timestamp: 150},
		{X: 400, Y: 0, Timestamp: 200},
		{X: 500, Y: 0, Timestamp: 250},
	}

	score, indicators := analyzer.CalculateAdvancedBotScore(botTrajectory, 500)

	if score < 0.3 {
		t.Errorf("机器人轨迹应该有较高的bot分数: %f", score)
	}

	if len(indicators) == 0 {
		t.Error("应该返回至少一个指标")
	}

	t.Logf("Bot分数: %.4f, 指标: %v", score, indicators)
}

func TestSliderTrajectoryValidator(t *testing.T) {
	validator := NewSliderTrajectoryValidator()

	validTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 50, Y: 2, Timestamp: 50},
		{X: 100, Y: 5, Timestamp: 100},
		{X: 150, Y: 8, Timestamp: 150},
		{X: 200, Y: 10, Timestamp: 200},
		{X: 250, Y: 12, Timestamp: 250},
		{X: 300, Y: 15, Timestamp: 300},
		{X: 350, Y: 18, Timestamp: 350},
		{X: 400, Y: 20, Timestamp: 400},
		{X: 450, Y: 22, Timestamp: 450},
		{X: 500, Y: 25, Timestamp: 500},
	}

	valid, msg := validator.Validate(validTrajectory)
	if !valid {
		t.Errorf("有效轨迹应该通过验证: %s", msg)
	}

	shortTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 50, Y: 2, Timestamp: 50},
	}

	valid, msg = validator.Validate(shortTrajectory)
	if valid {
		t.Error("过短的轨迹应该被拒绝")
	} else {
		t.Logf("短轨迹验证失败（预期）: %s", msg)
	}

	fastTrajectory := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1000, Y: 0, Timestamp: 10},
		{X: 2000, Y: 0, Timestamp: 20},
	}

	valid, msg = validator.Validate(fastTrajectory)
	if valid {
		t.Error("超快速轨迹应该被拒绝")
	} else {
		t.Logf("快速轨迹验证失败（预期）: %s", msg)
	}
}

func TestHumanLikeTrajectoryGeneration(t *testing.T) {
	trajectory := GenerateHumanLikeSliderTrajectory(0, 100, 500, 100, 2000)

	if len(trajectory) < 20 {
		t.Errorf("生成的轨迹点数太少: %d", len(trajectory))
	}

	t.Logf("生成的人类轨迹点数: %d, 起点X: %d, 终点X: %d",
		len(trajectory), trajectory[0].X, trajectory[len(trajectory)-1].X)
}

func TestBotLikeTrajectoryGeneration(t *testing.T) {
	trajectory := GenerateBotLikeSliderTrajectory(0, 100, 500, 100, 500)

	if len(trajectory) < 10 {
		t.Errorf("生成的轨迹点数太少: %d", len(trajectory))
	}

	if trajectory[len(trajectory)-1].X != 500 {
		t.Errorf("终点X坐标应为500，实际: %d", trajectory[len(trajectory)-1].X)
	}

	t.Logf("生成的机器人轨迹点数: %d", len(trajectory))
}

func TestConcurrentAnalysis(t *testing.T) {
	analyzer := NewSliderAnalyzer()

	trajectories := make([][]SliderPoint, 10)
	for i := range trajectories {
		trajectories[i] = GenerateHumanLikeSliderTrajectory(0, 100, 500, 100, 2000)
	}

	done := make(chan bool, len(trajectories))

	startTime := time.Now()

	for i, traj := range trajectories {
		go func(idx int, trajectory []SliderPoint) {
			result, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
			if err != nil {
				t.Errorf("并发分析失败: %v", err)
			}
			if result == nil {
				t.Error("结果不应为空")
			}
			done <- true
		}(i, traj)
	}

	for i := 0; i < len(trajectories); i++ {
		<-done
	}

	elapsed := time.Since(startTime)

	t.Logf("并发分析10条轨迹耗时: %v", elapsed)

	if elapsed > 5*time.Second {
		t.Errorf("并发分析耗时过长: %v", elapsed)
	}
}

func BenchmarkAnalyzeSliderTrajectory(b *testing.B) {
	analyzer := NewSliderAnalyzer()
	trajectory := GenerateHumanLikeSliderTrajectory(0, 100, 500, 100, 2000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeSliderTrajectory(trajectory, 500)
		if err != nil {
			b.Fatalf("分析失败: %v", err)
		}
	}
}

func BenchmarkExtractEnhancedSpeedFeatures(b *testing.B) {
	extractor := NewSliderFeatureExtractor()
	trajectory := GenerateHumanLikeSliderTrajectory(0, 100, 500, 100, 2000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		features := extractor.extractEnhancedSpeedFeatures(trajectory)
		if features == nil {
			b.Fatal("特征提取失败")
		}
	}
}

func BenchmarkDetectBacktrackPatterns(b *testing.B) {
	extractor := NewSliderFeatureExtractor()
	trajectory := GenerateHumanLikeSliderTrajectory(0, 100, 500, 100, 2000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		patterns := extractor.detectBacktrackPatterns(trajectory)
		if patterns == nil {
			b.Fatal("回退检测失败")
		}
	}
}
