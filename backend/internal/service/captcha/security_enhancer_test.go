package captcha

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSliderSecurityEnhancer(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()
	assert.NotNil(t, enhancer)
	assert.Equal(t, 10, enhancer.minTrajectoryPoints)
	assert.Equal(t, float64(2000), enhancer.maxSpeed)
	assert.Equal(t, float64(500), enhancer.maxAcceleration)
	assert.True(t, enhancer.enableHightSampling)
}

func TestSliderSecurityEnhancer_AnalyzeTrajectory_InsufficientPoints(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []SliderPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 10, Timestamp: 100},
	}

	result := enhancer.EnhancedTrajectoryAnalysis(points, 100)
	assert.NotNil(t, result)
	assert.False(t, result.IsHumanLike)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Equal(t, "high", result.RiskLevel)
	assert.Contains(t, result.AnomalyIndicators, "insufficient_trajectory_points")
}

func TestSliderSecurityEnhancer_AnalyzeTrajectory_HumanLike(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := make([]SliderPoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = SliderPoint{
			X:         i * 10,
			Y:         i * 5,
			Timestamp: int64(i * 100),
		}
	}

	result := enhancer.EnhancedTrajectoryAnalysis(points, 200)
	assert.NotNil(t, result)
	assert.Len(t, result.TrajectoryPoints, 20)
	assert.NotNil(t, result.SpeedProfile)
	assert.NotNil(t, result.AccelerationProfile)
	assert.Greater(t, result.TotalDistance, float64(0))
}

func TestSliderSecurityEnhancer_AnalyzeTrajectory_BotLike(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := make([]SliderPoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = SliderPoint{
			X:         i * 20,
			Y:         0,
			Timestamp: int64(i * 10),
		}
	}

	result := enhancer.EnhancedTrajectoryAnalysis(points, 400)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.SpeedProfile.AverageSpeed, float64(0))
}

func TestSliderSecurityEnhancer_convertToTrajectoryPoints(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []SliderPoint{
		{X: 10, Y: 20, Timestamp: 100},
		{X: 30, Y: 40, Timestamp: 200},
	}

	result := enhancer.convertToTrajectoryPoints(points)
	assert.Len(t, result, 2)
	assert.Equal(t, float64(10), result[0].X)
	assert.Equal(t, float64(20), result[0].Y)
	assert.Equal(t, int64(100), result[0].Timestamp)
}

func TestSliderSecurityEnhancer_calculateSpeedAndAcceleration(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 100},
		{X: 200, Y: 0, Timestamp: 200},
	}

	enhancer.calculateSpeedAndAcceleration(points)

	assert.Equal(t, float64(0), points[0].Speed)
	assert.GreaterOrEqual(t, points[1].Speed, float64(0))
	assert.GreaterOrEqual(t, points[2].Acceleration, float64(0))
}

func TestSliderSecurityEnhancer_analyzeSpeedProfile(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []TrajectoryPoint{
		{Speed: 100, Timestamp: 0},
		{Speed: 200, Timestamp: 100},
		{Speed: 150, Timestamp: 200},
	}

	result := enhancer.analyzeSpeedProfile(points)
	assert.Equal(t, float64(100), result.InitialSpeed)
	assert.Equal(t, float64(150), result.FinalSpeed)
	assert.Greater(t, result.AverageSpeed, float64(0))
}

func TestSliderSecurityEnhancer_analyzeSpeedProfile_EmptyPoints(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	result := enhancer.analyzeSpeedProfile([]TrajectoryPoint{})
	assert.Equal(t, float64(0), result.InitialSpeed)
}

func TestSliderSecurityEnhancer_analyzeSpeedProfile_SinglePoint(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	result := enhancer.analyzeSpeedProfile([]TrajectoryPoint{
		{Speed: 100, Timestamp: 0},
		{Speed: 100, Timestamp: 100},
	})
	assert.Equal(t, float64(100), result.InitialSpeed)
}

func TestSliderSecurityEnhancer_analyzeAccelerationProfile(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []TrajectoryPoint{
		{Speed: 100, Timestamp: 0},
		{Speed: 200, Timestamp: 100},
		{Speed: 300, Timestamp: 200},
		{Speed: 250, Timestamp: 300},
	}

	result := enhancer.analyzeAccelerationProfile(points)
	assert.Greater(t, result.AverageAcceleration, float64(0))
	assert.GreaterOrEqual(t, result.JerkCount, 0)
	assert.LessOrEqual(t, result.Smoothness, float64(1.0))
}

func TestSliderSecurityEnhancer_analyzeAccelerationProfile_InsufficientPoints(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	result := enhancer.analyzeAccelerationProfile([]TrajectoryPoint{
		{Speed: 100},
		{Speed: 200},
	})
	assert.Equal(t, float64(0), result.AverageAcceleration)
}

func TestSliderSecurityEnhancer_countDirectionChanges(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 0, Timestamp: 100},
		{X: 100, Y: 100, Timestamp: 200},
		{X: 0, Y: 100, Timestamp: 300},
	}

	result := enhancer.countDirectionChanges(points)
	assert.GreaterOrEqual(t, result, 0)
}

func TestSliderSecurityEnhancer_countDirectionChanges_InsufficientPoints(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	result := enhancer.countDirectionChanges([]TrajectoryPoint{
		{X: 0, Y: 0},
		{X: 100, Y: 0},
	})
	assert.Equal(t, 0, result)
}

func TestSliderSecurityEnhancer_calculateTotalDistance(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []TrajectoryPoint{
		{X: 0, Y: 0},
		{X: 100, Y: 0},
		{X: 100, Y: 100},
	}

	result := enhancer.calculateTotalDistance(points)
	assert.Greater(t, result, float64(0))
}

func TestSliderSecurityEnhancer_calculateTotalDistance_InsufficientPoints(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	result := enhancer.calculateTotalDistance([]TrajectoryPoint{{X: 0, Y: 0}})
	assert.Equal(t, float64(0), result)
}

func TestSliderSecurityEnhancer_calculateSpeedVariance(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := []TrajectoryPoint{
		{Speed: 100},
		{Speed: 200},
		{Speed: 150},
	}

	result := enhancer.calculateSpeedVariance(points)
	assert.GreaterOrEqual(t, result, float64(0))
}

func TestSliderSecurityEnhancer_calculateSpeedVariance_InsufficientPoints(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	result := enhancer.calculateSpeedVariance([]TrajectoryPoint{{Speed: 100}})
	assert.Equal(t, float64(0), result)
}

func TestSliderSecurityEnhancer_calculateHumanLikeScore(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := make([]TrajectoryPoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = TrajectoryPoint{
			X:         float64(i * 10),
			Y:         0,
			Timestamp: int64(i * 100),
		}
	}

	speedProfile := SpeedProfile{
		AverageSpeed: 100,
		SpeedFluctuation: 50,
	}

	accelProfile := AccelerationProfile{
		MaxAcceleration: 100,
		Smoothness:      0.8,
	}

	result := enhancer.calculateHumanLikeScore(points, speedProfile, accelProfile, 5)
	assert.GreaterOrEqual(t, result, float64(0))
	assert.LessOrEqual(t, result, float64(1))
}

func TestSliderSecurityEnhancer_calculateHumanLikeScore_TooFast(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := make([]TrajectoryPoint, 5)
	for i := 0; i < 5; i++ {
		points[i] = TrajectoryPoint{
			X:         float64(i * 100),
			Y:         0,
			Timestamp: int64(i * 10),
		}
	}

	speedProfile := SpeedProfile{
		AverageSpeed: 10000,
	}

	accelProfile := AccelerationProfile{}

	result := enhancer.calculateHumanLikeScore(points, speedProfile, accelProfile, 0)
	assert.Less(t, result, float64(0.7))
}

func TestSliderSecurityEnhancer_determineRiskLevel(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	accelProfile := AccelerationProfile{Smoothness: 0.8}

	result := enhancer.determineRiskLevel(0.9, SpeedProfile{}, accelProfile)
	assert.Equal(t, "low", result)

	result = enhancer.determineRiskLevel(0.6, SpeedProfile{}, accelProfile)
	assert.Equal(t, "medium", result)

	result = enhancer.determineRiskLevel(0.3, SpeedProfile{}, accelProfile)
	assert.Equal(t, "high", result)
}

func TestSliderSecurityEnhancer_detectAnomalyIndicators(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := make([]TrajectoryPoint, 5)
	for i := 0; i < 5; i++ {
		points[i] = TrajectoryPoint{
			X:         float64(i * 10),
			Y:         0,
			Timestamp: int64(i * 100),
			Speed:    10000,
		}
	}

	speedProfile := SpeedProfile{AverageSpeed: 10000}
	accelProfile := AccelerationProfile{
		MaxAcceleration: 1000,
		JerkCount:       10,
	}

	result := enhancer.detectAnomalyIndicators(points, speedProfile, accelProfile, 10)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "trajectory_too_short")
}

func TestSliderSecurityEnhancer_detectAnomalyIndicators_Normal(t *testing.T) {
	enhancer := NewSliderSecurityEnhancer()

	points := make([]TrajectoryPoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = TrajectoryPoint{
			X:         float64(i * 10),
			Y:         float64(i * 5),
			Timestamp: int64(i * 100),
			Speed:    100,
		}
	}

	speedProfile := SpeedProfile{AverageSpeed: 100}
	accelProfile := AccelerationProfile{
		MaxAcceleration: 100,
		JerkCount:       1,
	}

	result := enhancer.detectAnomalyIndicators(points, speedProfile, accelProfile, 3)
	assert.NotEmpty(t, result)
}

func TestNewClickCaptchaSecurityEnhancer(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()
	assert.NotNil(t, enhancer)
	assert.Equal(t, 3, enhancer.minClickPoints)
	assert.Equal(t, int64(100), enhancer.maxClickSpeed)
	assert.True(t, enhancer.enableZoneAnalysis)
}

func TestClickCaptchaSecurityEnhancer_AnalyzeClickPattern_InsufficientClicks(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{X: 10, Y: 20, Timestamp: 100},
		{X: 30, Y: 40, Timestamp: 200},
	}

	result := enhancer.AnalyzeClickPattern(clicks)
	assert.NotNil(t, result)
	assert.False(t, result.IsHumanLike)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Equal(t, "high", result.RiskLevel)
	assert.Contains(t, result.AnomalyIndicators, "insufficient_clicks")
}

func TestClickCaptchaSecurityEnhancer_AnalyzeClickPattern_Normal(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{X: 10, Y: 20, Timestamp: 0},
		{X: 30, Y: 40, Timestamp: 500},
		{X: 50, Y: 60, Timestamp: 1000},
		{X: 70, Y: 80, Timestamp: 1500},
	}

	result := enhancer.AnalyzeClickPattern(clicks)
	assert.NotNil(t, result)
	assert.Len(t, result.ClickPoints, 4)
	assert.Equal(t, 4, result.TotalClicks)
	assert.Greater(t, result.ClickTimeSpan, int64(0))
	assert.NotNil(t, result.ZoneDistribution)
}

func TestClickCaptchaSecurityEnhancer_AnalyzeClickPattern_Rapid(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{X: 10, Y: 20, Timestamp: 0},
		{X: 30, Y: 40, Timestamp: 50},
		{X: 50, Y: 60, Timestamp: 100},
		{X: 70, Y: 80, Timestamp: 150},
	}

	result := enhancer.AnalyzeClickPattern(clicks)
	assert.NotNil(t, result)
	assert.Equal(t, "rapid", result.ClickPattern)
}

func TestClickCaptchaSecurityEnhancer_AnalyzeClickPattern_Slow(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{X: 10, Y: 20, Timestamp: 0},
		{X: 30, Y: 40, Timestamp: 2000},
		{X: 50, Y: 60, Timestamp: 4000},
		{X: 70, Y: 80, Timestamp: 6000},
	}

	result := enhancer.AnalyzeClickPattern(clicks)
	assert.NotNil(t, result)
	assert.Equal(t, "slow", result.ClickPattern)
}

func TestClickCaptchaSecurityEnhancer_calculateIntervals(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{Timestamp: 0},
		{Timestamp: 100},
		{Timestamp: 300},
	}

	result := enhancer.calculateIntervals(clicks)
	assert.Len(t, result, 2)
	assert.Equal(t, int64(100), result[0])
	assert.Equal(t, int64(200), result[1])
}

func TestClickCaptchaSecurityEnhancer_calculateAverage(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	result := enhancer.calculateAverage([]int64{100, 200, 300})
	assert.Equal(t, int64(200), result)
}

func TestClickCaptchaSecurityEnhancer_calculateAverage_Empty(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	result := enhancer.calculateAverage([]int64{})
	assert.Equal(t, int64(0), result)
}

func TestClickCaptchaSecurityEnhancer_calculateVariance(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	result := enhancer.calculateVariance([]int64{100, 200, 300})
	assert.GreaterOrEqual(t, result, float64(0))
}

func TestClickCaptchaSecurityEnhancer_calculateVariance_SingleValue(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	result := enhancer.calculateVariance([]int64{100})
	assert.Equal(t, float64(0), result)
}

func TestClickCaptchaSecurityEnhancer_analyzeZoneDistribution(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{X: 50, Y: 50},
		{X: 150, Y: 50},
		{X: 50, Y: 150},
	}

	result := enhancer.analyzeZoneDistribution(clicks)
	assert.Len(t, result, 3)
	assert.Equal(t, 1, result["zone_0_0"])
	assert.Equal(t, 1, result["zone_1_0"])
	assert.Equal(t, 1, result["zone_0_1"])
}

func TestClickCaptchaSecurityEnhancer_getZone(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	result := enhancer.getZone(50, 50)
	assert.Equal(t, "zone_0_0", result)

	result = enhancer.getZone(150, 250)
	assert.Equal(t, "zone_1_2", result)
}

func TestClickCaptchaSecurityEnhancer_identifyClickPattern(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	result := enhancer.identifyClickPattern([]ClickPoint{{}, {}}, 100)
	assert.Equal(t, "rapid", result)

	result = enhancer.identifyClickPattern([]ClickPoint{{}, {}}, 500)
	assert.Equal(t, "normal", result)

	result = enhancer.identifyClickPattern([]ClickPoint{{}, {}}, 1500)
	assert.Equal(t, "slow", result)

	result = enhancer.identifyClickPattern([]ClickPoint{{}}, 500)
	assert.Equal(t, "insufficient_data", result)
}

func TestClickCaptchaSecurityEnhancer_calculateHumanLikeScore(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{Timestamp: 0},
		{Timestamp: 500},
		{Timestamp: 1000},
		{Timestamp: 1500},
	}

	result := enhancer.calculateHumanLikeScore(clicks, 500, 1000)
	assert.GreaterOrEqual(t, result, float64(0))
	assert.LessOrEqual(t, result, float64(1))
}

func TestClickCaptchaSecurityEnhancer_calculateHumanLikeScore_TooFast(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{Timestamp: 0},
		{Timestamp: 50},
		{Timestamp: 100},
		{Timestamp: 150},
	}

	result := enhancer.calculateHumanLikeScore(clicks, 50, 50)
	assert.Less(t, result, float64(0.5))
}

func TestClickCaptchaSecurityEnhancer_determineRiskLevel(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	result := enhancer.determineRiskLevel(0.8)
	assert.Equal(t, "low", result)

	result = enhancer.determineRiskLevel(0.5)
	assert.Equal(t, "medium", result)

	result = enhancer.determineRiskLevel(0.3)
	assert.Equal(t, "high", result)
}

func TestClickCaptchaSecurityEnhancer_detectAnomalyIndicators(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{X: 50, Y: 50, Timestamp: 0},
		{X: 60, Y: 60, Timestamp: 50},
	}

	result := enhancer.detectAnomalyIndicators(clicks, 50, 50)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "too_few_clicks")
}

func TestClickCaptchaSecurityEnhancer_detectAnomalyIndicators_Normal(t *testing.T) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := []ClickPoint{
		{X: 50, Y: 50, Timestamp: 0},
		{X: 60, Y: 60, Timestamp: 500},
		{X: 70, Y: 70, Timestamp: 1000},
		{X: 80, Y: 80, Timestamp: 1500},
	}

	result := enhancer.detectAnomalyIndicators(clicks, 500, 1000)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "normal_pattern")
}
