package service

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewBehaviorAnalysisService(t *testing.T) {
	service := NewBehaviorAnalysisService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.storedPaths)
	assert.Equal(t, 0, len(service.storedPaths))
}

func TestAnalyzeBehavior(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			Data:      `{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 150, "y": 250, "timestamp": 1100, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 150, "y": 250, "timestamp": 1200, "event": "click"}`,
			DataType:  "click",
			Timestamp: time.Now(),
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.RiskScore, 0.0)
	assert.LessOrEqual(t, result.RiskScore, 100.0)
	assert.NotNil(t, result.RiskIndicators)
	assert.NotNil(t, result.RiskFactors)
}

func TestAnalyzeBehaviorWithKeyboard(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			Data:      `{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 200, "y": 300, "timestamp": 1100, "event": "click"}`,
			DataType:  "click",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"key": "t", "timestamp": 2000, "hold_duration": 100}`,
			DataType:  "keyboard",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"key": "e", "timestamp": 2100, "hold_duration": 80}`,
			DataType:  "keyboard",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"key": "s", "timestamp": 2200, "hold_duration": 90}`,
			DataType:  "keyboard",
			Timestamp: time.Now(),
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.KeyboardPattern.KeystrokeCount)
	assert.Greater(t, result.KeyboardPattern.TypingSpeed, 0.0)
	assert.NotEmpty(t, result.KeyboardPattern.KeyStrokes)
}

func TestAnalyzeBehaviorWithClickPattern(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		createTestBehaviorData(100, 200, 1000, "click"),
		createTestBehaviorData(110, 210, 1150, "click"),
		createTestBehaviorData(120, 220, 1300, "click"),
		createTestBehaviorData(130, 230, 1450, "click"),
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, result.ClickPattern.ClickCount)
	assert.Greater(t, result.ClickPattern.ClickSpeed, 0.0)
	assert.GreaterOrEqual(t, result.ClickPattern.Regularity, 0.0)
	assert.LessOrEqual(t, result.ClickPattern.Regularity, 1.0)
	assert.NotNil(t, result.ClickPattern.XDistribution)
	assert.NotNil(t, result.ClickPattern.YDistribution)
}

func TestCalculateRiskScore(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name         string
		behaviorData []models.BehaviorData
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "empty data",
			behaviorData: []models.BehaviorData{},
			expectedMin:  0.0,
			expectedMax:  100.0,
		},
		{
			name: "normal behavior",
			behaviorData: []models.BehaviorData{
				createTestBehaviorData(100, 200, 1000, "mousemove"),
				createTestBehaviorData(120, 220, 1050, "mousemove"),
				createTestBehaviorData(140, 240, 1100, "mousemove"),
				createTestBehaviorData(140, 240, 1200, "click"),
			},
			expectedMin: 0.0,
			expectedMax: 100.0,
		},
		{
			name: "high speed movement",
			behaviorData: []models.BehaviorData{
				createTestBehaviorData(0, 0, 1000, "mousemove"),
				createTestBehaviorData(1000, 1000, 1050, "mousemove"),
				createTestBehaviorData(2000, 2000, 1100, "mousemove"),
			},
			expectedMin: 0.0,
			expectedMax: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verification := &models.Verification{}
			riskScore := service.CalculateRiskScore(verification, tt.behaviorData)
			assert.GreaterOrEqual(t, riskScore, tt.expectedMin)
			assert.LessOrEqual(t, riskScore, tt.expectedMax)
		})
	}
}

func TestVerifyWithBehaviorAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name           string
		captchaSuccess bool
		behaviorData   []models.BehaviorData
	}{
		{
			name:           "success with low risk",
			captchaSuccess: true,
			behaviorData: []models.BehaviorData{
				createTestBehaviorData(100, 200, 1000, "mousemove"),
				createTestBehaviorData(150, 250, 1100, "click"),
			},
		},
		{
			name:           "fail with captcha fail",
			captchaSuccess: false,
			behaviorData:   []models.BehaviorData{},
		},
		{
			name:           "high risk behavior",
			captchaSuccess: true,
			behaviorData: []models.BehaviorData{
				createTestBehaviorData(0, 0, 1000, "mousemove"),
				createTestBehaviorData(1000, 1000, 1010, "mousemove"),
				createTestBehaviorData(2000, 2000, 1020, "mousemove"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, riskScore, report := service.VerifyWithBehaviorAnalysis(tt.captchaSuccess, tt.behaviorData)
			assert.NotEmpty(t, report)
			assert.GreaterOrEqual(t, riskScore, 0.0)
			assert.LessOrEqual(t, riskScore, 100.0)
			if tt.name == "fail with captcha fail" {
				assert.False(t, passed)
			}
		})
	}
}

func TestGenerateAnalysisReport(t *testing.T) {
	service := NewBehaviorAnalysisService()

	result := &AnalysisResult{
		Trajectory: MouseTrajectory{
			Points: []BehaviorDataPoint{{X: 100, Y: 200, Timestamp: 1000, Event: "click"}},
		},
		ClickPattern: ClickPattern{
			ClickCount: 5,
		},
		KeyboardPattern: KeyboardPattern{
			KeystrokeCount: 10,
			KeyStrokes: []KeyboardDataPoint{
				{Key: "a", Timestamp: 1000},
				{Key: "b", Timestamp: 1100},
			},
		},
		RiskScore:      35.0,
		IsBotLikely:   false,
		Confidence:     0.65,
		RiskIndicators: []string{"test indicator"},
		SpeedAnalysis: SpeedAnalysis{
			AverageSpeed: 1.5,
			MaxSpeed:     3.0,
		},
		PathSimilarity: PathSimilarity{
			SimilarityScore: 0.5,
			ComparedPathLength: 10,
		},
	}

	report := service.GenerateAnalysisReport(result)
	assert.Contains(t, report, "风险评分")
	assert.Contains(t, report, "疑似机器人")
	assert.Contains(t, report, "置信度")
	assert.Contains(t, report, "轨迹分析")
	assert.Contains(t, report, "点击模式")
	assert.Contains(t, report, "键盘模式")
	assert.Contains(t, report, "速度分析")
	assert.Contains(t, report, "路径相似度")
}

func TestAnalyzeSpeedMethod(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		createTestBehaviorData(100, 200, 1000, "mousemove"),
		createTestBehaviorData(110, 210, 1100, "mousemove"),
		createTestBehaviorData(120, 220, 1200, "mousemove"),
	}

	analysis, err := service.AnalyzeSpeed(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Greater(t, analysis.AverageSpeed, 0.0)
	assert.Equal(t, 2, len(analysis.Speeds))
}

func TestBotBehaviorDetection(t *testing.T) {
	service := NewBehaviorAnalysisService()

	botMovement := []models.BehaviorData{}
	startTime := int64(1000)
	for i := 0; i < 30; i++ {
		x := 100 + i*5
		y := 100 + i*5
		timestamp := startTime + int64(i)*10
		bd := createTestBehaviorData(x, y, timestamp, "mousemove")
		botMovement = append(botMovement, bd)
	}

	regularClicks := []models.BehaviorData{}
	for i := 0; i < 5; i++ {
		timestamp := startTime + int64(i)*100
		bd := createTestBehaviorData(200, 200, timestamp, "click")
		regularClicks = append(regularClicks, bd)
	}

	combinedBehavior := append(botMovement, regularClicks...)
	result, err := service.AnalyzeBehavior(combinedBehavior)
	assert.NoError(t, err)

	assert.True(t, len(result.RiskIndicators) > 0 || result.RiskScore > 0)
}

func TestRiskFactorsMap(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		createTestBehaviorData(100, 200, 1000, "mousemove"),
		createTestBehaviorData(500, 600, 1050, "mousemove"),
		createTestBehaviorData(100, 200, 1100, "click"),
		createTestBehaviorData(100, 200, 1200, "click"),
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result.RiskFactors)
	assert.GreaterOrEqual(t, len(result.RiskFactors), 0)
}

func TestEmptyBehaviorData(t *testing.T) {
	service := NewBehaviorAnalysisService()

	result, err := service.AnalyzeBehavior([]models.BehaviorData{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.RiskScore, 0.0)
	assert.LessOrEqual(t, result.RiskScore, 100.0)
}

func TestStraightPathDetection(t *testing.T) {
	service := NewBehaviorAnalysisService()

	straightPath := []models.BehaviorData{}
	for i := 0; i < 50; i++ {
		timestamp := int64(1000 + i*10)
		bd := createTestBehaviorData(100+i*10, 100+i*10, timestamp, "mousemove")
		straightPath = append(straightPath, bd)
	}

	result, err := service.AnalyzeBehavior(straightPath)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50, len(result.Trajectory.Points))
}

func TestRegularClicksDetection(t *testing.T) {
	service := NewBehaviorAnalysisService()

	regularClicks := []models.BehaviorData{}
	for i := 0; i < 10; i++ {
		timestamp := int64(1000 + i*200)
		bd := createTestBehaviorData(200, 200, timestamp, "click")
		regularClicks = append(regularClicks, bd)
	}

	result, err := service.AnalyzeBehavior(regularClicks)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 10, result.ClickPattern.ClickCount)
	assert.Greater(t, result.ClickPattern.Regularity, 0.5)
}

func TestAnalyzePathSimilarity(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "mousemove"},
		{X: 140, Y: 140, Timestamp: 1400, Event: "mousemove"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "mousemove"},
		{X: 140, Y: 140, Timestamp: 1400, Event: "mousemove"},
	}

	similarity := service.AnalyzePathSimilarity(path1, path2)
	assert.NotNil(t, similarity)
	assert.Equal(t, 5, similarity.ComparedPathLength)
	assert.GreaterOrEqual(t, similarity.SimilarityScore, 0.0)
	assert.LessOrEqual(t, similarity.SimilarityScore, 1.0)
}

func TestSmoothTrajectory(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "mousemove"},
		{X: 140, Y: 140, Timestamp: 1400, Event: "mousemove"},
		{X: 150, Y: 150, Timestamp: 1500, Event: "mousemove"},
		{X: 160, Y: 160, Timestamp: 1600, Event: "mousemove"},
	}

	smoothed := service.smoothTrajectory(points, 3)
	assert.Len(t, smoothed, len(points))

	for i, p := range smoothed {
		assert.Equal(t, points[i].Timestamp, p.Timestamp)
		assert.Equal(t, points[i].Event, p.Event)
	}

	shortPoints := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
	}
	shortSmoothed := service.smoothTrajectory(shortPoints, 5)
	assert.Len(t, shortSmoothed, len(shortPoints))

	evenWindowSize := service.smoothTrajectory(points, 4)
	assert.Len(t, evenWindowSize, len(points))
}

func TestSavitzkyGolaySmooth(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "mousemove"},
		{X: 140, Y: 140, Timestamp: 1400, Event: "mousemove"},
	}

	smoothed := service.savitzkyGolaySmooth(points, 3, 2)
	assert.Len(t, smoothed, len(points))

	tooShort := service.savitzkyGolaySmooth(points, 10, 2)
	assert.Len(t, tooShort, len(points))
}

func TestClickPatternEnhanced(t *testing.T) {
	service := NewBehaviorAnalysisService()

	clicks := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "click"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "click"},
		{X: 250, Y: 250, Timestamp: 1300, Event: "click"},
		{X: 300, Y: 300, Timestamp: 1400, Event: "click"},
	}

	pattern := service.analyzeClickPatternEnhanced(clicks, clicks)
	assert.Equal(t, 5, pattern.ClickCount)
	assert.Greater(t, pattern.ClickSpeed, 0.0)
	assert.Greater(t, pattern.PositionEntropy, 0.0)
	assert.Greater(t, pattern.ClickAreaSize, 0.0)
	assert.NotNil(t, pattern.XDistribution)
	assert.NotNil(t, pattern.YDistribution)
	assert.Equal(t, 10, len(pattern.XDistribution))
	assert.Equal(t, 10, len(pattern.YDistribution))
}

func TestKeyboardPatternAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	keyStrokes := []KeyboardDataPoint{
		{Key: "a", Timestamp: 1000, HoldDuration: 100},
		{Key: "b", Timestamp: 1100, HoldDuration: 80},
		{Key: "c", Timestamp: 1200, HoldDuration: 90},
		{Key: "d", Timestamp: 1300, HoldDuration: 85},
	}

	pattern := service.analyzeKeyboardPattern(keyStrokes)
	assert.Equal(t, 4, pattern.KeystrokeCount)
	assert.Greater(t, pattern.TypingSpeed, 0.0)
	assert.GreaterOrEqual(t, pattern.Regularity, 0.0)
	assert.LessOrEqual(t, pattern.Regularity, 1.0)
	assert.NotNil(t, pattern.CommonPairs)
	assert.Greater(t, len(pattern.CommonPairs), 0)
}

func TestKeyboardPatternWithCombos(t *testing.T) {
	service := NewBehaviorAnalysisService()

	keyStrokes := []KeyboardDataPoint{
		{Key: "ctrl", Timestamp: 1000},
		{Key: "c", Timestamp: 1050},
		{Key: "a", Timestamp: 1200},
		{Key: "v", Timestamp: 1300},
		{Key: "ctrl", Timestamp: 1500},
		{Key: "s", Timestamp: 1550},
	}

	pattern := service.analyzeKeyboardPattern(keyStrokes)
	assert.Equal(t, 6, pattern.KeystrokeCount)
	assert.True(t, pattern.ComboDetected)
	assert.Greater(t, len(pattern.ComboPatterns), 0)
}

func TestStatisticalFunctions(t *testing.T) {
	service := NewBehaviorAnalysisService()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	assert.InDelta(t, 3.0, service.mean(values), 0.001)

	emptyMean := service.mean([]float64{})
	assert.Equal(t, 0.0, emptyMean)

	assert.Equal(t, 5.0, service.max(values))
	assert.Equal(t, 1.0, service.min(values))
	assert.Equal(t, 5.0, service.maxAbs([]float64{-5.0, -3.0, -1.0}))

	sorted := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	assert.InDelta(t, 3.0, service.median(sorted), 0.001)

	evenSorted := []float64{1.0, 2.0, 3.0, 4.0}
	assert.InDelta(t, 2.5, service.median(evenSorted), 0.001)

	assert.Greater(t, service.variance(values), 0.0)

	skewTest := []float64{1.0, 2.0, 3.0, 4.0, 100.0}
	skewness := service.skewness(skewTest)
	assert.Greater(t, skewness, 0.0)

	zeroVariance := service.variance([]float64{1.0, 1.0, 1.0})
	assert.Equal(t, 0.0, zeroVariance)
}

func TestComputeEntropy(t *testing.T) {
	service := NewBehaviorAnalysisService()

	counts := []int{2, 2, 2, 2}
	entropy := service.computeEntropy(counts)
	assert.Greater(t, entropy, 0.0)

	emptyEntropy := service.computeEntropy([]int{})
	assert.Equal(t, 0.0, emptyEntropy)

	singleCount := []int{10}
	singleEntropy := service.computeEntropy(singleCount)
	assert.Equal(t, 0.0, singleEntropy)
}

func TestComputeCurvature(t *testing.T) {
	service := NewBehaviorAnalysisService()

	p1 := BehaviorDataPoint{X: 0, Y: 0, Timestamp: 1000, Event: "mousemove"}
	p2 := BehaviorDataPoint{X: 10, Y: 10, Timestamp: 1100, Event: "mousemove"}
	p3 := BehaviorDataPoint{X: 20, Y: 20, Timestamp: 1200, Event: "mousemove"}

	curvature := service.computeCurvature(p1, p2, p3)
	assert.GreaterOrEqual(t, curvature, -3.15)
	assert.LessOrEqual(t, curvature, 3.15)

	p4 := BehaviorDataPoint{X: 10, Y: 0, Timestamp: 1300, Event: "mousemove"}
	curvature2 := service.computeCurvature(p1, p2, p4)
	assert.Greater(t, math.Abs(curvature2), 0.0)
}

func TestComputeDTWDistance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
	}

	path2 := []BehaviorDataPoint{
		{X: 105, Y: 105, Timestamp: 1000, Event: "mousemove"},
		{X: 115, Y: 115, Timestamp: 1100, Event: "mousemove"},
		{X: 125, Y: 125, Timestamp: 1200, Event: "mousemove"},
	}

	distance := service.computeDTWDistance(path1, path2)
	assert.GreaterOrEqual(t, distance, 0.0)

	identicalDistance := service.computeDTWDistance(path1, path1)
	assert.Equal(t, 0.0, identicalDistance)
}

func TestComputeFrechetDistance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
	}

	path2 := []BehaviorDataPoint{
		{X: 105, Y: 105, Timestamp: 1000, Event: "mousemove"},
		{X: 115, Y: 115, Timestamp: 1100, Event: "mousemove"},
		{X: 125, Y: 125, Timestamp: 1200, Event: "mousemove"},
	}

	distance := service.computeFrechetDistance(path1, path2)
	assert.GreaterOrEqual(t, distance, 0.0)
}

func TestComputePathCorrelation(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
	}

	path2 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
	}

	corr := service.computePathCorrelation(path1, path2)
	assert.Equal(t, 1.0, corr)

	emptyCorr := service.computePathCorrelation([]BehaviorDataPoint{}, path2)
	assert.Equal(t, 0.0, emptyCorr)

	differentLengthCorr := service.computePathCorrelation(path1, []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
	})
	assert.Equal(t, 0.0, differentLengthCorr)
}

func TestPearsonCorrelation(t *testing.T) {
	service := NewBehaviorAnalysisService()

	x := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	y := []float64{2.0, 4.0, 6.0, 8.0, 10.0}
	corr := service.pearsonCorrelation(x, y)
	assert.Greater(t, corr, 0.9)

	negativeCorr := service.pearsonCorrelation(x, []float64{10.0, 8.0, 6.0, 4.0, 2.0})
	assert.Less(t, negativeCorr, -0.9)

	zeroCorr := service.pearsonCorrelation([]float64{1.0}, []float64{1.0})
	assert.Equal(t, 0.0, zeroCorr)
}

func TestPointDistance(t *testing.T) {
	service := NewBehaviorAnalysisService()

	p1 := BehaviorDataPoint{X: 0, Y: 0}
	p2 := BehaviorDataPoint{X: 3, Y: 4}

	dist := service.pointDistance(p1, p2)
	assert.InDelta(t, 5.0, dist, 0.001)

	samePoint := BehaviorDataPoint{X: 0, Y: 0}
	zeroDist := service.pointDistance(samePoint, samePoint)
	assert.Equal(t, 0.0, zeroDist)
}

func TestAnalyzeMouseTrajectory(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "mousemove"},
	}

	smoothed := service.smoothTrajectory(points, 3)
	traj := service.analyzeMouseTrajectory(smoothed, points)

	assert.Equal(t, len(points), len(traj.Points))
	assert.Greater(t, traj.TotalDistance, 0.0)
	assert.Greater(t, traj.AverageSpeed, 0.0)
	assert.GreaterOrEqual(t, traj.PathEfficiency, 0.0)
	assert.LessOrEqual(t, traj.PathEfficiency, 1.0)
}

func TestAnalyzeMouseTrajectoryWithFewPoints(t *testing.T) {
	service := NewBehaviorAnalysisService()

	singlePoint := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
	}

	traj := service.analyzeMouseTrajectory(singlePoint, singlePoint)
	assert.Equal(t, 1, len(traj.Points))
	assert.Equal(t, 0.0, traj.TotalDistance)
}

func TestComputePositionDistribution(t *testing.T) {
	service := NewBehaviorAnalysisService()

	clicks := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
		{X: 200, Y: 200, Timestamp: 1100, Event: "click"},
		{X: 300, Y: 300, Timestamp: 1200, Event: "click"},
	}

	xDist := service.computePositionDistribution(clicks, true, 5)
	assert.Len(t, xDist, 5)

	yDist := service.computePositionDistribution(clicks, false, 5)
	assert.Len(t, yDist, 5)

	emptyDist := service.computePositionDistribution([]BehaviorDataPoint{}, true, 5)
	assert.Len(t, emptyDist, 5)
}

func TestCheckPathSimilarity(t *testing.T) {
	service := NewBehaviorAnalysisService()

	path1 := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "mousemove"},
		{X: 140, Y: 140, Timestamp: 1400, Event: "mousemove"},
	}

	similarity := service.checkPathSimilarity(path1)
	assert.Equal(t, 5, similarity.ComparedPathLength)

	path2 := []BehaviorDataPoint{
		{X: 105, Y: 105, Timestamp: 1000, Event: "mousemove"},
		{X: 115, Y: 115, Timestamp: 1100, Event: "mousemove"},
		{X: 125, Y: 125, Timestamp: 1200, Event: "mousemove"},
		{X: 135, Y: 135, Timestamp: 1300, Event: "mousemove"},
		{X: 145, Y: 145, Timestamp: 1400, Event: "mousemove"},
	}

	similarity2 := service.checkPathSimilarity(path2)
	assert.GreaterOrEqual(t, similarity2.SimilarityScore, 0.0)
}

func TestComputePathHash(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "mousemove"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "mousemove"},
	}

	hash := service.computePathHash(points)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, "|")

	hash2 := service.computePathHash(points)
	assert.Equal(t, hash, hash2)

	longPoints := []BehaviorDataPoint{}
	for i := 0; i < 30; i++ {
		longPoints = append(longPoints, BehaviorDataPoint{X: i * 10, Y: i * 10, Timestamp: int64(1000 + i*100)})
	}
	longHash := service.computePathHash(longPoints)
	pointCount := 0
	for _, p := range longPoints[:20] {
		pointCount++
		_ = p
	}
	assert.NotEmpty(t, longHash)
	assert.Greater(t, pointCount, 0)
}

func TestCheckPathHashMatch(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 150, Y: 150, Timestamp: 1100, Event: "mousemove"},
		{X: 200, Y: 200, Timestamp: 1200, Event: "mousemove"},
		{X: 250, Y: 250, Timestamp: 1300, Event: "mousemove"},
		{X: 300, Y: 300, Timestamp: 1400, Event: "mousemove"},
	}

	hash := service.computePathHash(points)
	match := service.checkPathHashMatch(hash)
	assert.False(t, match)

	_ = points
}

func TestInvertMatrix(t *testing.T) {
	service := NewBehaviorAnalysisService()

	matrix := [][]float64{
		{4, 7},
		{2, 6},
	}

	inverse := service.invertMatrix(matrix)
	assert.Len(t, inverse, 2)
	assert.Len(t, inverse[0], 2)

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			sum := 0.0
			for k := 0; k < 2; k++ {
				sum += matrix[i][k] * inverse[k][j]
			}
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			assert.InDelta(t, expected, sum, 0.001)
		}
	}
}

func TestComputeSGCoefficients(t *testing.T) {
	service := NewBehaviorAnalysisService()

	coeffs := service.computeSGCoefficients(5, 2)
	assert.Len(t, coeffs, 5)

	coeffs2 := service.computeSGCoefficients(7, 3)
	assert.Len(t, coeffs2, 7)
}

func TestExtractCoordinates(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 110, Y: 210, Timestamp: 1100},
		{X: 120, Y: 220, Timestamp: 1200},
	}

	x, y := service.extractCoordinates(points)
	assert.Len(t, x, 3)
	assert.Len(t, y, 3)
	assert.Equal(t, 100.0, x[0])
	assert.Equal(t, 200.0, y[0])
}

func TestAnalyzeBehaviorWithInvalidJSON(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			Data:      "invalid json",
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Trajectory.Points), 1)
}

func TestAnalyzeBehaviorWithMixedDataTypes(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			Data:      `{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"}`,
			DataType:  "unknown",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 100, "y": 200, "timestamp": 1100, "event": "click"}`,
			DataType:  "click",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"key": "a", "timestamp": 2000}`,
			DataType:  "keyboard",
			Timestamp: time.Now(),
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.ClickPattern.Clicks))
	assert.Equal(t, 1, len(result.KeyboardPattern.KeyStrokes))
}

func TestVerifyWithBehaviorAnalysisHighRisk(t *testing.T) {
	service := NewBehaviorAnalysisService()

	highRiskData := []models.BehaviorData{}
	startTime := int64(1000)
	for i := 0; i < 50; i++ {
		x := int64(100 + i*20)
		y := int64(100 + i*20)
		timestamp := startTime + int64(i)*5
		bd := createTestBehaviorData(int(x), int(y), timestamp, "mousemove")
		highRiskData = append(highRiskData, bd)
	}

	passed, riskScore, report := service.VerifyWithBehaviorAnalysis(true, highRiskData)
	assert.Greater(t, riskScore, 30.0)
	assert.NotEmpty(t, report)
	_ = passed
}

func TestStoredPathsLimit(t *testing.T) {
	service := NewBehaviorAnalysisService()

	for i := 0; i < 150; i++ {
		path := []BehaviorDataPoint{}
		for j := 0; j < 10; j++ {
			path = append(path, BehaviorDataPoint{
				X:         100 + j*10,
				Y:         100 + j*10 + i,
				Timestamp: int64(1000 + j*100),
				Event:     "mousemove",
			})
		}
		service.checkPathSimilarity(path)
	}

	assert.LessOrEqual(t, len(service.storedPaths), 100)
}

func createTestBehaviorData(x, y int, timestamp int64, event string) models.BehaviorData {
	data := BehaviorDataPoint{
		X:         x,
		Y:         y,
		Timestamp: timestamp,
		Event:     event,
	}
	dataJSON, _ := json.Marshal(data)
	return models.BehaviorData{
		Data:      string(dataJSON),
		DataType:  event,
		Timestamp: time.Now(),
	}
}

func generateHumanLikeTrajectory() []models.BehaviorData {
	data := []models.BehaviorData{}
	startTime := int64(1000)

	segments := []struct {
		startX, startY, endX, endY int
		baseDurationMs             int64
		points                     int
	}{
		{100, 100, 200, 180, 600, 12},
		{200, 180, 280, 220, 500, 10},
		{280, 220, 350, 250, 400, 8},
		{350, 250, 380, 310, 450, 9},
		{380, 310, 420, 280, 300, 6},
	}

	timeOffset := startTime
	for _, seg := range segments {
		for i := 0; i < seg.points; i++ {
			t := float64(i) / float64(seg.points-1)
			t = t*t*(3-2*t)

			speedFactor := 0.5 + 0.5*math.Sin(math.Pi*t)
			segDuration := float64(seg.baseDurationMs) * (0.8 + 0.4*speedFactor)
			pointInterval := segDuration / float64(seg.points)
			pointInterval += float64(i%3-1) * 3

			x := seg.startX + int(float64(seg.endX-seg.startX)*t)
			y := seg.startY + int(float64(seg.endY-seg.startY)*t)

			jitterX := int(math.Sin(float64(i)*1.7)*3 + float64(i%5)-2)
			jitterY := int(math.Cos(float64(i)*1.3)*3 + float64(i%4)-2)
			x += jitterX
			y += jitterY

			timeOffset += int64(math.Max(pointInterval, 10))
			bd := createTestBehaviorData(x, y, timeOffset, "mousemove")
			data = append(data, bd)
		}

		timeOffset += 60 + int64(seg.points%3)*20
		pauseBd := createTestBehaviorData(seg.endX, seg.endY, timeOffset, "mousemove")
		data = append(data, pauseBd)
	}

	timeOffset += 120
	clickBd := createTestBehaviorData(420, 280, timeOffset, "click")
	data = append(data, clickBd)

	return data
}

func generateBotLikeTrajectory() []models.BehaviorData {
	data := []models.BehaviorData{}
	startTime := int64(1000)

	for i := 0; i < 40; i++ {
		x := 100 + i*10
		y := 100 + i*6
		timestamp := startTime + int64(i)*20
		bd := createTestBehaviorData(x, y, timestamp, "mousemove")
		data = append(data, bd)
	}

	timestamp := startTime + int64(40)*20
	clickBd := createTestBehaviorData(500, 340, timestamp, "click")
	data = append(data, clickBd)

	return data
}

func TestHumanTrajectoryLowRisk(t *testing.T) {
	service := NewBehaviorAnalysisService()

	humanData := generateHumanLikeTrajectory()
	result, err := service.AnalyzeBehavior(humanData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	t.Logf("人类轨迹风险评分: %.2f", result.RiskScore)
	t.Logf("人类轨迹风险指标: %v", result.RiskIndicators)
	t.Logf("人类轨迹路径效率: %.4f", result.Trajectory.PathEfficiency)
	t.Logf("人类轨迹抖动: %.4f", result.Trajectory.JitterScore)
	t.Logf("人类轨迹曲率: %.6f", result.Trajectory.CurvatureAvg)
	t.Logf("人类轨迹停顿: %d", result.Trajectory.PauseCount)
	t.Logf("人类轨迹微修正: %d", result.Trajectory.MicroCorrections)

	assert.Less(t, result.RiskScore, 50.0, "人类轨迹的风险评分应低于50")
	assert.False(t, result.IsBotLikely, "人类轨迹不应被判定为机器人")
}

func TestBotTrajectoryHighRisk(t *testing.T) {
	service := NewBehaviorAnalysisService()

	botData := generateBotLikeTrajectory()
	result, err := service.AnalyzeBehavior(botData)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	t.Logf("机器人轨迹风险评分: %.2f", result.RiskScore)
	t.Logf("机器人轨迹风险指标: %v", result.RiskIndicators)
	t.Logf("机器人轨迹路径效率: %.4f", result.Trajectory.PathEfficiency)
	t.Logf("机器人轨迹抖动: %.4f", result.Trajectory.JitterScore)
	t.Logf("机器人轨迹曲率: %.6f", result.Trajectory.CurvatureAvg)
	t.Logf("机器人轨迹停顿: %d", result.Trajectory.PauseCount)
	t.Logf("机器人轨迹微修正: %d", result.Trajectory.MicroCorrections)

	assert.GreaterOrEqual(t, result.RiskScore, 50.0, "机器人轨迹的风险评分应不低于50")
	assert.True(t, result.IsBotLikely, "机器人轨迹应被判定为机器人")
	assert.Contains(t, result.RiskIndicators, "路径过于笔直", "机器人轨迹应包含\"路径过于笔直\"指标")
}

func TestStraightLineLowScore(t *testing.T) {
	service := NewBehaviorAnalysisService()

	straightPath := []models.BehaviorData{}
	for i := 0; i < 30; i++ {
		timestamp := int64(1000 + i*10)
		bd := createTestBehaviorData(100+i*10, 100+i*10, timestamp, "mousemove")
		straightPath = append(straightPath, bd)
	}

	result, err := service.AnalyzeBehavior(straightPath)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	t.Logf("直线轨迹风险评分: %.2f", result.RiskScore)
	t.Logf("直线轨迹风险指标: %v", result.RiskIndicators)
	t.Logf("直线轨迹路径效率: %.4f", result.Trajectory.PathEfficiency)
	t.Logf("直线轨迹曲率: %.6f", result.Trajectory.CurvatureAvg)
	t.Logf("直线轨迹抖动: %.4f", result.Trajectory.JitterScore)

	assert.GreaterOrEqual(t, result.RiskScore, 50.0, "直线轨迹风险评分应不低于50")
	assert.True(t, result.IsBotLikely, "直线轨迹应被判定为机器人")
}

func TestPreClickHesitation(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		createTestBehaviorData(100, 100, 1000, "mousemove"),
		createTestBehaviorData(200, 200, 1100, "mousemove"),
		createTestBehaviorData(300, 300, 1200, "mousemove"),
		createTestBehaviorData(300, 300, 1350, "click"),
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.ClickPattern.PreClickHesitation, 0.0, "点击前犹豫应大于0")
	assert.InDelta(t, 150.0, result.ClickPattern.PreClickHesitation, 50.0, "点击前犹豫应在150ms左右")
}

func TestHumanVsBotScoreDifference(t *testing.T) {
	service := NewBehaviorAnalysisService()

	humanData := generateHumanLikeTrajectory()
	botData := generateBotLikeTrajectory()

	humanResult, _ := service.AnalyzeBehavior(humanData)
	botResult, _ := service.AnalyzeBehavior(botData)

	t.Logf("人类轨迹评分: %.2f, 机器人轨迹评分: %.2f", humanResult.RiskScore, botResult.RiskScore)
	t.Logf("评分差距: %.2f", botResult.RiskScore-humanResult.RiskScore)

	assert.Greater(t, botResult.RiskScore, humanResult.RiskScore,
		"机器人轨迹评分应高于人类轨迹评分")
	assert.Greater(t, botResult.RiskScore-humanResult.RiskScore, 20.0,
		"机器人轨迹与人类轨迹的评分差距应大于20分")
}

func TestMicroCorrectionsDetection(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := []BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 108, Timestamp: 1050, Event: "mousemove"},
		{X: 118, Y: 115, Timestamp: 1100, Event: "mousemove"},
		{X: 125, Y: 125, Timestamp: 1150, Event: "mousemove"},
		{X: 128, Y: 127, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 129, Timestamp: 1250, Event: "mousemove"},
		{X: 131, Y: 130, Timestamp: 1300, Event: "mousemove"},
	}

	smoothed := service.smoothTrajectory(points, 3)
	traj := service.analyzeMouseTrajectory(smoothed, points)

	t.Logf("微修正次数: %d", traj.MicroCorrections)
	t.Logf("停顿次数: %d", traj.PauseCount)

	assert.GreaterOrEqual(t, traj.MicroCorrections, 0)
}
