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

func TestCalculateRiskScoreFromFeatures(t *testing.T) {
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

func TestExtractFeatures(t *testing.T) {
	trajectory := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 120, Y: 110, Timestamp: 1050},
		{X: 150, Y: 130, Timestamp: 1110},
		{X: 190, Y: 160, Timestamp: 1180},
		{X: 240, Y: 200, Timestamp: 1260},
		{X: 300, Y: 250, Timestamp: 1350},
		{X: 370, Y: 300, Timestamp: 1450},
		{X: 450, Y: 340, Timestamp: 1560},
	}

	features := ExtractFeatures(trajectory)

	assert.NotNil(t, features)
	assert.Greater(t, features.AvgSpeed, 0.0, "平均速度应大于0")
	assert.Greater(t, features.MaxSpeed, 0.0, "最大速度应大于0")
	assert.GreaterOrEqual(t, features.TrajectorySmoothness, 0.0, "轨迹平滑度应大于等于0")
	assert.LessOrEqual(t, features.TrajectorySmoothness, 1.0, "轨迹平滑度应小于等于1")
	assert.GreaterOrEqual(t, features.PathComplexity, 0.0, "路径复杂度应大于等于0")
	assert.LessOrEqual(t, features.PathComplexity, 1.0, "路径复杂度应小于等于1")
	assert.GreaterOrEqual(t, features.RiskScore, 0.0, "风险评分应大于等于0")
	assert.LessOrEqual(t, features.RiskScore, 100.0, "风险评分应小于等于100")
}

func TestExtractFeaturesEmptyTrajectory(t *testing.T) {
	emptyTrajectory := []TrajectoryPoint{}

	features := ExtractFeatures(emptyTrajectory)

	assert.NotNil(t, features)
	assert.Equal(t, 0.0, features.AvgSpeed, "空轨迹的平均速度应为0")
	assert.Equal(t, 0.0, features.MaxSpeed, "空轨迹的最大速度应为0")
}

func TestCalculateAverageSpeed(t *testing.T) {
	tests := []struct {
		name     string
		points   []TrajectoryPoint
		expected float64
	}{
		{
			name: "正常轨迹",
			points: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 100, Y: 100, Timestamp: 1100},
				{X: 200, Y: 200, Timestamp: 1200},
			},
			expected: 141.42,
		},
		{
			name: "空轨迹",
			points: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
			},
			expected: 0.0,
		},
		{
			name: "单点轨迹",
			points: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 0},
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speed := CalculateAverageSpeed(tt.points)
			if tt.expected == 0.0 {
				assert.Equal(t, tt.expected, speed)
			} else {
				assert.Greater(t, speed, 0.0, "平均速度应大于0")
			}
		})
	}
}

func TestCalculateMaxSpeed(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
		{X: 200, Y: 0, Timestamp: 1150},
		{X: 250, Y: 0, Timestamp: 1200},
	}

	maxSpeed := CalculateMaxSpeed(points)

	assert.Greater(t, maxSpeed, 0.0, "最大速度应大于0")
	t.Logf("最大速度: %.2f", maxSpeed)
}

func TestCalculateMinSpeed(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
		{X: 200, Y: 0, Timestamp: 1500},
		{X: 250, Y: 0, Timestamp: 1600},
	}

	minSpeed := CalculateMinSpeed(points)

	assert.Greater(t, minSpeed, 0.0, "最小速度应大于0")
	t.Logf("最小速度: %.2f", minSpeed)
}

func TestCalculateSpeedVariation(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 100, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
		{X: 300, Y: 300, Timestamp: 1300},
		{X: 400, Y: 400, Timestamp: 1400},
	}

	variation := CalculateSpeedVariation(points)
	assert.GreaterOrEqual(t, variation, 0.0, "速度变化率应大于等于0")
	t.Logf("速度变化率: %.4f", variation)
}

func TestCalculateAcceleration(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 0, Timestamp: 1100},
		{X: 250, Y: 0, Timestamp: 1250},
		{X: 450, Y: 0, Timestamp: 1450},
		{X: 700, Y: 0, Timestamp: 1700},
	}

	acceleration := CalculateAcceleration(points)

	t.Logf("加速度: %.6f", acceleration)
	assert.GreaterOrEqual(t, acceleration, 0.0, "加速度应大于等于0")
}

func TestCalculateTrajectorySmoothness(t *testing.T) {
	tests := []struct {
		name        string
		points      []TrajectoryPoint
		minExpected float64
		maxExpected float64
	}{
		{
			name: "完全直线",
			points: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 100, Y: 100, Timestamp: 1100},
				{X: 200, Y: 200, Timestamp: 1200},
				{X: 300, Y: 300, Timestamp: 1300},
			},
			minExpected: 0.9,
			maxExpected: 1.0,
		},
		{
			name: "曲线轨迹",
			points: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 50, Y: 100, Timestamp: 1100},
				{X: 150, Y: 150, Timestamp: 1200},
				{X: 100, Y: 250, Timestamp: 1300},
				{X: 200, Y: 300, Timestamp: 1400},
			},
			minExpected: 0.0,
			maxExpected: 1.0,
		},
		{
			name:        "点数不足",
			points:      []TrajectoryPoint{},
			minExpected: 1.0,
			maxExpected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smoothness := CalculateTrajectorySmoothness(tt.points)
			assert.GreaterOrEqual(t, smoothness, tt.minExpected)
			assert.LessOrEqual(t, smoothness, tt.maxExpected)
		})
	}
}

func TestCalculateClickInterval(t *testing.T) {
	clicks := []ClickData{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1150},
		{X: 300, Y: 300, Timestamp: 1350},
		{X: 400, Y: 400, Timestamp: 1600},
	}

	interval := CalculateClickInterval(clicks)

	expectedInterval := 200.0
	assert.InDelta(t, expectedInterval, interval, 10.0, "平均点击间隔应为200ms")
}

func TestCalculateClickPositionVariance(t *testing.T) {
	tests := []struct {
		name     string
		clicks   []ClickData
		expected float64
	}{
		{
			name: "聚集点击",
			clicks: []ClickData{
				{X: 100, Y: 100, Timestamp: 1000},
				{X: 105, Y: 105, Timestamp: 1100},
				{X: 102, Y: 98, Timestamp: 1200},
			},
			expected: 100.0,
		},
		{
			name: "分散点击",
			clicks: []ClickData{
				{X: 100, Y: 100, Timestamp: 1000},
				{X: 500, Y: 500, Timestamp: 1100},
				{X: 100, Y: 500, Timestamp: 1200},
				{X: 500, Y: 100, Timestamp: 1300},
			},
			expected: 40000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variance := CalculateClickPositionVariance(tt.clicks)
			assert.GreaterOrEqual(t, variance, 0.0, "点击位置方差应大于等于0")
		})
	}
}

func TestCalculatePathComplexity(t *testing.T) {
	tests := []struct {
		name        string
		points      []TrajectoryPoint
		minExpected float64
		maxExpected float64
	}{
		{
			name: "完全直线",
			points: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 100, Y: 100, Timestamp: 1100},
				{X: 200, Y: 200, Timestamp: 1200},
				{X: 300, Y: 300, Timestamp: 1300},
			},
			minExpected: 0.0,
			maxExpected: 0.1,
		},
		{
			name: "复杂路径",
			points: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 100, Y: 50, Timestamp: 1100},
				{X: 50, Y: 100, Timestamp: 1200},
				{X: 150, Y: 80, Timestamp: 1300},
				{X: 200, Y: 200, Timestamp: 1400},
				{X: 150, Y: 250, Timestamp: 1500},
				{X: 250, Y: 200, Timestamp: 1600},
			},
			minExpected: 0.3,
			maxExpected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := CalculatePathComplexity(tt.points)
			assert.GreaterOrEqual(t, complexity, tt.minExpected)
			assert.LessOrEqual(t, complexity, tt.maxExpected)
		})
	}
}

func TestDTWDistance(t *testing.T) {
	tests := []struct {
		name     string
		seq1     []TrajectoryPoint
		seq2     []TrajectoryPoint
		expected float64
	}{
		{
			name: "完全相同序列",
			seq1: []TrajectoryPoint{
				{X: 100, Y: 100, Timestamp: 1000},
				{X: 150, Y: 150, Timestamp: 1100},
				{X: 200, Y: 200, Timestamp: 1200},
			},
			seq2: []TrajectoryPoint{
				{X: 100, Y: 100, Timestamp: 1000},
				{X: 150, Y: 150, Timestamp: 1100},
				{X: 200, Y: 200, Timestamp: 1200},
			},
			expected: 0.0,
		},
		{
			name: "相似序列",
			seq1: []TrajectoryPoint{
				{X: 100, Y: 100, Timestamp: 1000},
				{X: 150, Y: 150, Timestamp: 1100},
				{X: 200, Y: 200, Timestamp: 1200},
			},
			seq2: []TrajectoryPoint{
				{X: 105, Y: 105, Timestamp: 1000},
				{X: 155, Y: 155, Timestamp: 1100},
				{X: 205, Y: 205, Timestamp: 1200},
			},
			expected: 15.0,
		},
		{
			name:     "空序列",
			seq1:     []TrajectoryPoint{},
			seq2:     []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 0}},
			expected: math.MaxFloat64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := DTWDistance(tt.seq1, tt.seq2)
			if tt.expected == math.MaxFloat64 {
				assert.Equal(t, tt.expected, distance)
			} else {
				assert.InDelta(t, tt.expected, distance, tt.expected*0.5+5.0)
			}
		})
	}
}

func TestCompareWithHumanTrajectory(t *testing.T) {
	tests := []struct {
		name      string
		trajectory []TrajectoryPoint
		minSim    float64
		maxSim    float64
	}{
		{
			name: "人类类似轨迹",
			trajectory: []TrajectoryPoint{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 120, Y: 115, Timestamp: 50},
				{X: 145, Y: 135, Timestamp: 100},
				{X: 175, Y: 160, Timestamp: 160},
				{X: 210, Y: 190, Timestamp: 230},
				{X: 250, Y: 220, Timestamp: 310},
				{X: 290, Y: 255, Timestamp: 400},
				{X: 330, Y: 285, Timestamp: 500},
			},
			minSim: 0.0,
			maxSim: 1.0,
		},
		{
			name: "机器人轨迹",
			trajectory: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 100, Timestamp: 10},
				{X: 200, Y: 200, Timestamp: 20},
				{X: 300, Y: 300, Timestamp: 30},
				{X: 400, Y: 400, Timestamp: 40},
			},
			minSim: 0.0,
			maxSim: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := CompareWithHumanTrajectory(tt.trajectory)
			assert.GreaterOrEqual(t, similarity, tt.minSim)
			assert.LessOrEqual(t, similarity, tt.maxSim)
			t.Logf("轨迹相似度: %.4f", similarity)
		})
	}
}

func TestCalculateRiskScore(t *testing.T) {
	tests := []struct {
		name     string
		features *BehaviorFeatures
		minScore float64
		maxScore float64
	}{
		{
			name: "正常特征",
			features: &BehaviorFeatures{
				AvgSpeed:             500.0,
				TrajectorySmoothness: 0.5,
				Acceleration:        0.5,
				PathComplexity:      0.7,
				PathSimilarity:      0.8,
				SpeedVariation:      0.5,
				ClickInterval:      200.0,
			},
			minScore: 0.0,
			maxScore: 30.0,
		},
		{
			name: "机器人特征",
			features: &BehaviorFeatures{
				AvgSpeed:            2000.0,
				TrajectorySmoothness: 0.98,
				Acceleration:       0.05,
				PathComplexity:     0.1,
				PathSimilarity:     0.3,
				SpeedVariation:     0.05,
				ClickInterval:     30.0,
			},
			minScore: 70.0,
			maxScore: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateRiskScore(tt.features)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
			t.Logf("风险评分: %.2f", score)
		})
	}
}

func TestIsRobot(t *testing.T) {
	tests := []struct {
		name     string
		features *BehaviorFeatures
		expected bool
	}{
		{
			name: "低风险特征",
			features: &BehaviorFeatures{
				RiskScore: 30.0,
			},
			expected: false,
		},
		{
			name: "中风险特征",
			features: &BehaviorFeatures{
				RiskScore: 50.0,
			},
			expected: true,
		},
		{
			name: "高风险特征",
			features: &BehaviorFeatures{
				RiskScore: 85.0,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isBot := IsRobot(tt.features)
			assert.Equal(t, tt.expected, isBot)
		})
	}
}

func TestScoreCard(t *testing.T) {
	sc := NewScoreCard()

	assert.NotNil(t, sc)
	assert.NotNil(t, sc.Weights)
	assert.NotNil(t, sc.Thresholds)
	assert.Equal(t, 7, len(sc.Weights))
	assert.Equal(t, 7, len(sc.Thresholds))

	normalFeatures := &BehaviorFeatures{
		AvgSpeed:            500.0,
		TrajectorySmoothness: 0.5,
		Acceleration:       0.5,
		PathComplexity:     0.7,
		PathSimilarity:     0.8,
		SpeedVariation:     0.5,
		ClickInterval:     200.0,
	}

	score := sc.Evaluate(normalFeatures)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
	t.Logf("正常特征评分: %.2f", score)

	botFeatures := &BehaviorFeatures{
		AvgSpeed:            2000.0,
		TrajectorySmoothness: 0.98,
		Acceleration:       0.05,
		PathComplexity:     0.1,
		PathSimilarity:     0.3,
		SpeedVariation:     0.05,
		ClickInterval:     30.0,
	}

	botScore := sc.Evaluate(botFeatures)
	assert.GreaterOrEqual(t, botScore, 0.0)
	assert.LessOrEqual(t, botScore, 100.0)
	t.Logf("机器人特征评分: %.2f", botScore)

	assert.Greater(t, botScore, score, "机器人特征评分应高于正常特征评分")
}

func TestScoreCardNilFeatures(t *testing.T) {
	sc := NewScoreCard()
	score := sc.Evaluate(nil)
	assert.Equal(t, 0.0, score)
}

func TestRuleEngine(t *testing.T) {
	engine := NewRuleEngine()

	assert.NotNil(t, engine)
	assert.Equal(t, 0, len(engine.rules))

	rule := Rule{
		Name: "test_rule",
		Condition: func(f *BehaviorFeatures) bool {
			return f.AvgSpeed > 1000
		},
		Weight: 25.0,
	}

	engine.AddRule(rule)
	assert.Equal(t, 1, len(engine.rules))

	lowSpeed := &BehaviorFeatures{AvgSpeed: 500}
	highSpeed := &BehaviorFeatures{AvgSpeed: 1500}

	lowScore := engine.Evaluate(lowSpeed)
	assert.Equal(t, 0.0, lowScore)

	highScore := engine.Evaluate(highSpeed)
	assert.Equal(t, 100.0, highScore)
}

func TestRuleEngineNilFeatures(t *testing.T) {
	engine := NewRuleEngine()
	engine.AddRule(Rule{
		Name:      "test",
		Condition: func(f *BehaviorFeatures) bool { return true },
		Weight:    10,
	})

	score := engine.Evaluate(nil)
	assert.Equal(t, 0.0, score)
}

func TestBotDetectionRules(t *testing.T) {
	engine := NewBotDetectionRuleEngine()

	assert.NotNil(t, engine)
	assert.Greater(t, len(engine.rules), 0)

	botFeatures := &BehaviorFeatures{
		AvgSpeed:            2000.0,
		TrajectorySmoothness: 0.98,
		Acceleration:       0.05,
		PathComplexity:     0.1,
		PathSimilarity:     0.3,
		SpeedVariation:     0.05,
		ClickInterval:     30.0,
	}

	score := engine.Evaluate(botFeatures)
	assert.Greater(t, score, 0.0, "机器人特征应触发规则")
	t.Logf("机器人特征规则评分: %.2f", score)

	humanFeatures := &BehaviorFeatures{
		AvgSpeed:            500.0,
		TrajectorySmoothness: 0.5,
		Acceleration:        0.5,
		PathComplexity:      0.7,
		PathSimilarity:      0.8,
		SpeedVariation:      0.5,
		ClickInterval:      200.0,
	}

	humanScore := engine.Evaluate(humanFeatures)
	assert.Less(t, humanScore, score, "人类特征评分应低于机器人特征评分")
	t.Logf("人类特征规则评分: %.2f", humanScore)

	triggered := engine.GetTriggeredRules(botFeatures)
	assert.Greater(t, len(triggered), 0)
	t.Logf("触发的规则: %v", triggered)
}

func TestMLClassifier(t *testing.T) {
	classifier := NewMLClassifier()

	assert.NotNil(t, classifier)
	assert.NotNil(t, classifier.ruleEngine)
	assert.NotNil(t, classifier.scoreCard)

	botFeatures := &BehaviorFeatures{
		AvgSpeed:            2000.0,
		TrajectorySmoothness: 0.98,
		Acceleration:       0.05,
		PathComplexity:     0.1,
		PathSimilarity:     0.3,
		SpeedVariation:     0.05,
		ClickInterval:     30.0,
	}

	isBot, score := classifier.Classify(botFeatures)
	assert.True(t, isBot, "应识别为机器人")
	assert.Greater(t, score, 50.0)
	t.Logf("机器人分类评分: %.2f", score)

	humanFeatures := &BehaviorFeatures{
		AvgSpeed:            500.0,
		TrajectorySmoothness: 0.5,
		Acceleration:        0.5,
		PathComplexity:      0.7,
		PathSimilarity:      0.8,
		SpeedVariation:      0.5,
		ClickInterval:      200.0,
	}

	isBotHuman, scoreHuman := classifier.Classify(humanFeatures)
	assert.False(t, isBotHuman, "不应识别为机器人")
	assert.Less(t, scoreHuman, 50.0)
	t.Logf("人类分类评分: %.2f", scoreHuman)

	confidence := classifier.GetConfidence(botFeatures)
	assert.GreaterOrEqual(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 1.0)
	t.Logf("置信度: %.4f", confidence)
}

func TestMLClassifierNilFeatures(t *testing.T) {
	classifier := NewMLClassifier()

	isBot, score := classifier.Classify(nil)
	assert.False(t, isBot)
	assert.Equal(t, 0.0, score)

	confidence := classifier.GetConfidence(nil)
	assert.Equal(t, 0.0, confidence)
}

func TestMLClassifierDetailedAnalysis(t *testing.T) {
	classifier := NewMLClassifier()

	features := &BehaviorFeatures{
		AvgSpeed:            2000.0,
		TrajectorySmoothness: 0.98,
		Acceleration:       0.05,
		PathComplexity:     0.1,
		PathSimilarity:     0.3,
		SpeedVariation:     0.05,
		ClickInterval:     30.0,
	}

	analysis := classifier.GetDetailedAnalysis(features)

	assert.NotNil(t, analysis)
	assert.Contains(t, analysis, "triggered_rules")
	assert.Contains(t, analysis, "rule_count")
	assert.Contains(t, analysis, "rule_score")
	assert.Contains(t, analysis, "score_card_score")
	assert.Contains(t, analysis, "risk_score")
	assert.Contains(t, analysis, "confidence")
	assert.Contains(t, analysis, "is_bot")
	assert.Contains(t, analysis, "final_score")

	triggered := analysis["triggered_rules"].([]string)
	assert.Greater(t, len(triggered), 0)

	ruleCount := analysis["rule_count"].(int)
	assert.Greater(t, ruleCount, 0)
}

func TestEnsembleClassifier(t *testing.T) {
	ensemble := NewEnsembleClassifier()

	assert.NotNil(t, ensemble)
	assert.Greater(t, len(ensemble.classifiers), 0)

	botFeatures := &BehaviorFeatures{
		AvgSpeed:            2000.0,
		TrajectorySmoothness: 0.98,
		Acceleration:       0.05,
		PathComplexity:     0.1,
		PathSimilarity:     0.3,
		SpeedVariation:     0.05,
		ClickInterval:     30.0,
	}

	isBot, score := ensemble.Classify(botFeatures)
	assert.True(t, isBot)
	assert.Greater(t, score, 50.0)
	t.Logf("集成分类器机器人评分: %.2f", score)

	humanFeatures := &BehaviorFeatures{
		AvgSpeed:            500.0,
		TrajectorySmoothness: 0.5,
		Acceleration:        0.5,
		PathComplexity:      0.7,
		PathSimilarity:      0.8,
		SpeedVariation:      0.5,
		ClickInterval:      200.0,
	}

	isBotHuman, scoreHuman := ensemble.Classify(humanFeatures)
	assert.False(t, isBotHuman)
	assert.Less(t, scoreHuman, 50.0)
	t.Logf("集成分类器人类评分: %.2f", scoreHuman)
}

func TestEnsembleClassifierDetailedAnalysis(t *testing.T) {
	ensemble := NewEnsembleClassifier()

	features := &BehaviorFeatures{
		AvgSpeed:            2000.0,
		TrajectorySmoothness: 0.98,
		Acceleration:       0.05,
		PathComplexity:     0.1,
		PathSimilarity:     0.3,
		SpeedVariation:     0.05,
		ClickInterval:     30.0,
	}

	analysis := ensemble.GetDetailedAnalysis(features)

	assert.NotNil(t, analysis)
	assert.Contains(t, analysis, "classifier_results")
	assert.Contains(t, analysis, "bot_votes")
	assert.Contains(t, analysis, "total_votes")
	assert.Contains(t, analysis, "vote_ratio")
	assert.Contains(t, analysis, "is_bot")
	assert.Contains(t, analysis, "final_score")

	classifierResults := analysis["classifier_results"].([]map[string]interface{})
	assert.Equal(t, len(ensemble.classifiers), len(classifierResults))

	botVotes := analysis["bot_votes"].(int)
	totalVotes := analysis["total_votes"].(int)
	assert.Greater(t, botVotes, 0)
	assert.Equal(t, len(ensemble.classifiers), totalVotes)
}

func TestValidateTrajectory(t *testing.T) {
	tests := []struct {
		name      string
		trajectory []TrajectoryPoint
		expected  bool
	}{
		{
			name: "有效轨迹",
			trajectory: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 100, Y: 100, Timestamp: 1100},
				{X: 200, Y: 200, Timestamp: 1200},
			},
			expected: true,
		},
		{
			name: "点数不足",
			trajectory: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 100, Y: 100, Timestamp: 1100},
			},
			expected: false,
		},
		{
			name: "时间戳不递增",
			trajectory: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 1000},
				{X: 100, Y: 100, Timestamp: 900},
				{X: 200, Y: 200, Timestamp: 1200},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateTrajectory(tt.trajectory)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreprocessTrajectory(t *testing.T) {
	tests := []struct {
		name          string
		input         []TrajectoryPoint
		targetLength  int
		expectedLen   int
	}{
		{
			name: "下采样",
			input: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 10, Y: 10, Timestamp: 10},
				{X: 20, Y: 20, Timestamp: 20},
				{X: 30, Y: 30, Timestamp: 30},
				{X: 40, Y: 40, Timestamp: 40},
				{X: 50, Y: 50, Timestamp: 50},
				{X: 60, Y: 60, Timestamp: 60},
				{X: 70, Y: 70, Timestamp: 70},
				{X: 80, Y: 80, Timestamp: 80},
				{X: 90, Y: 90, Timestamp: 90},
			},
			targetLength: 5,
			expectedLen:  5,
		},
		{
			name: "上采样",
			input: []TrajectoryPoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 50, Y: 50, Timestamp: 50},
				{X: 100, Y: 100, Timestamp: 100},
			},
			targetLength: 7,
			expectedLen:  7,
		},
		{
			name:         "空轨迹",
			input:        []TrajectoryPoint{},
			targetLength: 5,
			expectedLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreprocessTrajectory(tt.input, tt.targetLength)
			assert.Equal(t, tt.expectedLen, len(result))
		})
	}
}

func TestExtractFeaturesFromBehaviorData(t *testing.T) {
	behaviorData := []models.BehaviorData{
		{
			Data:      `{"x": 100, "y": 100, "timestamp": 1000, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 120, "y": 110, "timestamp": 1050, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 150, "y": 130, "timestamp": 1110, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 190, "y": 160, "timestamp": 1180, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 240, "y": 200, "timestamp": 1260, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 300, "y": 250, "timestamp": 1350, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 370, "y": 300, "timestamp": 1450, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 450, "y": 340, "timestamp": 1560, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
	}

	features := ExtractFeaturesFromBehaviorData(behaviorData)

	assert.NotNil(t, features)
	assert.Greater(t, features.AvgSpeed, 0.0)
	assert.Greater(t, features.MaxSpeed, 0.0)
	assert.GreaterOrEqual(t, features.TrajectorySmoothness, 0.0)
	assert.LessOrEqual(t, features.TrajectorySmoothness, 1.0)
}

func TestFullMLPipeline(t *testing.T) {
	service := NewBehaviorAnalysisService()
	mlClassifier := NewMLClassifier()
	ensemble := NewEnsembleClassifier()

	botTrajectory := []models.BehaviorData{}
	startTime := int64(1000)

	for i := 0; i < 50; i++ {
		x := 100 + i*5
		y := 100 + i*5
		timestamp := startTime + int64(i)*10

		bd := createTestBehaviorData(x, y, timestamp, "mousemove")
		botTrajectory = append(botTrajectory, bd)
	}

	humanTrajectory := generateHumanLikeTrajectory()

	botResult, _ := service.AnalyzeBehavior(botTrajectory)
	humanResult, _ := service.AnalyzeBehavior(humanTrajectory)

	t.Logf("传统分析 - 机器人轨迹评分: %.2f", botResult.RiskScore)
	t.Logf("传统分析 - 人类轨迹评分: %.2f", humanResult.RiskScore)

	botFeatures := ExtractFeaturesFromBehaviorData(botTrajectory)
	humanFeatures := ExtractFeaturesFromBehaviorData(humanTrajectory)

	isBotML, mlScore := mlClassifier.Classify(botFeatures)
	isHumanML, mlScoreHuman := mlClassifier.Classify(humanFeatures)

	t.Logf("ML分类器 - 机器人评分: %.2f, 识别为机器人: %v", mlScore, isBotML)
	t.Logf("ML分类器 - 人类评分: %.2f, 识别为机器人: %v", mlScoreHuman, isHumanML)

	isBotEnsemble, ensembleScore := ensemble.Classify(botFeatures)
	isHumanEnsemble, ensembleScoreHuman := ensemble.Classify(humanFeatures)

	t.Logf("集成分类器 - 机器人评分: %.2f, 识别为机器人: %v", ensembleScore, isBotEnsemble)
	t.Logf("集成分类器 - 人类评分: %.2f, 识别为机器人: %v", ensembleScoreHuman, isHumanEnsemble)

	assert.True(t, isBotML, "ML分类器应识别机器人轨迹")
	assert.False(t, isHumanML, "ML分类器不应识别人类轨迹")

	assert.True(t, isBotEnsemble, "集成分类器应识别机器人轨迹")
	assert.False(t, isHumanEnsemble, "集成分类器不应识别人类轨迹")

	assert.Greater(t, mlScore, mlScoreHuman, "机器人评分应高于人类评分")
	assert.Greater(t, ensembleScore, ensembleScoreHuman, "机器人评分应高于人类评分")
}

func TestTrajectoryPointConversion(t *testing.T) {
	behaviorData := []BehaviorDataPoint{
		{X: 100, Y: 200, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 210, Timestamp: 1050, Event: "mousemove"},
		{X: 120, Y: 220, Timestamp: 1100, Event: "mousemove"},
	}

	features := ExtractFeaturesFromDataPoints(behaviorData)

	assert.NotNil(t, features)
	assert.Greater(t, features.AvgSpeed, 0.0)
}

func TestNewBotDetectionRuleEngine(t *testing.T) {
	engine := NewBotDetectionRuleEngine()

	assert.NotNil(t, engine)
	assert.Equal(t, len(BotDetectionRules), len(engine.rules))

	for i, rule := range engine.rules {
		assert.NotEmpty(t, rule.Name)
		assert.NotNil(t, rule.Condition)
		assert.Greater(t, rule.Weight, 0.0)
		assert.Equal(t, BotDetectionRules[i].Name, rule.Name)
		assert.Equal(t, BotDetectionRules[i].Weight, rule.Weight)
	}
}

func TestGetTriggeredRules(t *testing.T) {
	engine := NewBotDetectionRuleEngine()

	features := &BehaviorFeatures{
		AvgSpeed:            2000.0,
		TrajectorySmoothness: 0.98,
		Acceleration:       0.05,
		PathComplexity:     0.1,
		PathSimilarity:     0.3,
		SpeedVariation:     0.05,
		ClickInterval:     30.0,
	}

	triggered := engine.GetTriggeredRules(features)

	assert.Greater(t, len(triggered), 0)

	counts := engine.CountTriggeredRules(features)
	assert.Equal(t, len(triggered), counts)

	for _, ruleName := range triggered {
		found := false
		for _, rule := range BotDetectionRules {
			if rule.Name == ruleName {
				found = true
				assert.True(t, rule.Condition(features))
				break
			}
		}
		assert.True(t, found, "触发规则 %s 应存在于预定义规则中", ruleName)
	}
}

func TestHumanTrajectoryComparisonWithTemplates(t *testing.T) {
	human1 := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 120, Y: 115, Timestamp: 50},
		{X: 145, Y: 135, Timestamp: 100},
		{X: 175, Y: 160, Timestamp: 160},
		{X: 210, Y: 190, Timestamp: 230},
		{X: 250, Y: 220, Timestamp: 310},
		{X: 290, Y: 255, Timestamp: 400},
		{X: 330, Y: 285, Timestamp: 500},
	}

	similarity1 := CompareWithHumanTrajectory(human1)
	t.Logf("模板类似轨迹相似度: %.4f", similarity1)

	bot := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 100, Timestamp: 10},
		{X: 200, Y: 200, Timestamp: 20},
		{X: 300, Y: 300, Timestamp: 30},
		{X: 400, Y: 400, Timestamp: 40},
	}

	similarity2 := CompareWithHumanTrajectory(bot)
	t.Logf("机器人轨迹相似度: %.4f", similarity2)

	assert.GreaterOrEqual(t, similarity1, 0.0)
	assert.LessOrEqual(t, similarity1, 1.0)
	assert.GreaterOrEqual(t, similarity2, 0.0)
	assert.LessOrEqual(t, similarity2, 1.0)
}

func TestNormalizeTrajectory(t *testing.T) {
	trajectory := []TrajectoryPoint{
		{X: 500, Y: 500, Timestamp: 0},
		{X: 600, Y: 600, Timestamp: 100},
		{X: 700, Y: 700, Timestamp: 200},
	}

	normalized := normalizeTrajectory(trajectory)

	assert.Equal(t, len(trajectory), len(normalized))

	minX := normalized[0].X
	minY := normalized[0].Y

	for _, p := range normalized {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
	}

	assert.Equal(t, 0, minX)
	assert.Equal(t, 0, minY)
}

func TestPointDist(t *testing.T) {
	p1 := TrajectoryPoint{X: 0, Y: 0, Timestamp: 0}
	p2 := TrajectoryPoint{X: 3, Y: 4, Timestamp: 0}

	distance := pointDist(p1, p2)

	assert.InDelta(t, 5.0, distance, 0.001)

	p3 := TrajectoryPoint{X: 0, Y: 0, Timestamp: 0}
	p4 := TrajectoryPoint{X: 0, Y: 0, Timestamp: 0}

	distance2 := pointDist(p3, p4)
	assert.Equal(t, 0.0, distance2)
}

func TestConvertToClickData(t *testing.T) {
	trajectory := []TrajectoryPoint{
		{X: 100, Y: 100, Timestamp: 1000},
		{X: 110, Y: 110, Timestamp: 1050},
		{X: 120, Y: 120, Timestamp: 1100},
		{X: 130, Y: 130, Timestamp: 1150},
	}

	clicks := convertToClickData(trajectory)

	assert.Equal(t, len(trajectory), len(clicks))

	for i, click := range clicks {
		assert.Equal(t, trajectory[i].X, click.X)
		assert.Equal(t, trajectory[i].Y, click.Y)
		assert.Equal(t, trajectory[i].Timestamp, click.Timestamp)
	}
}
