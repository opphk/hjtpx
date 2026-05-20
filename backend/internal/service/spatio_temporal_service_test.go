package service

import (
	"context"
	"math"
	"testing"
	"time"

	github.com/hjtpx/hjtpx/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSpatioTemporalService_Generate(t *testing.T) {
	service := NewSpatioTemporalService()

	req := &model.SpatioTemporalRequest{
		UserID:      "test-user-123",
		PatternType: model.TimePatternDaily,
		Difficulty:  "medium",
		ClientIP:    "192.168.1.1",
		UserAgent:   "Mozilla/5.0",
	}

	resp, err := service.Generate(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotNil(t, resp.TargetPattern)
	assert.Len(t, resp.ChallengePoints, 4)
	assert.Len(t, resp.Options, 4)
	assert.Equal(t, int64(300), resp.ExpiresIn)
}

func TestSpatioTemporalService_Generate_WithLocation(t *testing.T) {
	service := NewSpatioTemporalService()

	currentLocation := &model.SpatioTemporalPoint{
		Timestamp: time.Now().Unix(),
		Latitude:  39.9042,
		Longitude: 116.4074,
		Accuracy:  model.LocationAccuracyGPS,
		Confidence: 0.9,
	}

	req := &model.SpatioTemporalRequest{
		UserID:            "test-user-456",
		PatternType:       model.TimePatternWeekly,
		Difficulty:        "hard",
		CurrentLocation:   currentLocation,
		IncludePredictions: true,
		PredictionWindow:  3600,
	}

	resp, err := service.Generate(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.BehaviorFlow)
	assert.NotNil(t, resp.Prediction)
	assert.NotNil(t, resp.RiskScore)
}

func TestSpatioTemporalService_Generate_DefaultValues(t *testing.T) {
	service := NewSpatioTemporalService()

	req := &model.SpatioTemporalRequest{
		UserID: "test-user-default",
	}

	resp, err := service.Generate(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, model.TimePatternDaily, resp.TargetPattern.PatternType)
	assert.Equal(t, "medium", resp.SessionID[:10])
}

func TestSpatioTemporalService_Generate_EmptyUserID(t *testing.T) {
	service := NewSpatioTemporalService()

	req := &model.SpatioTemporalRequest{}

	resp, err := service.Generate(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestSpatioTemporalService_Generate_DifferentDifficulties(t *testing.T) {
	service := NewSpatioTemporalService()
	difficulties := []string{"easy", "medium", "hard", "expert"}

	for _, diff := range difficulties {
		t.Run(diff, func(t *testing.T) {
			req := &model.SpatioTemporalRequest{
				UserID:     "test-user-" + diff,
				Difficulty: diff,
			}

			resp, err := service.Generate(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestSpatioTemporalService_Verify_CorrectOption(t *testing.T) {
	service := NewSpatioTemporalService()

	session, _ := service.Generate(&model.SpatioTemporalRequest{
		UserID: "verify-test-user",
	})

	resp, err := service.Verify(&model.SpatioTemporalVerifyRequest{
		SessionID:      session.SessionID,
		SelectedOption: session.CorrectOption,
		UserLocation: &model.SpatioTemporalPoint{
			Latitude:  39.9042,
			Longitude: 116.4074,
		},
		ResponseTime: 5000,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	if resp.Score >= 0.7 {
		assert.True(t, resp.Success)
	}
}

func TestSpatioTemporalService_Verify_WrongOption(t *testing.T) {
	service := NewSpatioTemporalService()

	session, _ := service.Generate(&model.SpatioTemporalRequest{
		UserID: "wrong-option-user",
	})

	wrongOption := "option_99"
	if wrongOption == session.CorrectOption {
		wrongOption = "option_0"
		if wrongOption == session.CorrectOption {
			wrongOption = "option_1"
		}
	}

	resp, err := service.Verify(&model.SpatioTemporalVerifyRequest{
		SessionID:      session.SessionID,
		SelectedOption: wrongOption,
		UserLocation: &model.SpatioTemporalPoint{
			Latitude:  39.9042,
			Longitude: 116.4074,
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestSpatioTemporalService_Verify_SessionNotFound(t *testing.T) {
	service := NewSpatioTemporalService()

	resp, err := service.Verify(&model.SpatioTemporalVerifyRequest{
		SessionID:      "nonexistent-session",
		SelectedOption: "option_0",
	})

	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "会话不存在", resp.Message)
}

func TestSpatioTemporalService_Verify_ExpiredSession(t *testing.T) {
	service := NewSpatioTemporalService()

	session, _ := service.Generate(&model.SpatioTemporalRequest{
		UserID: "expired-user",
	})

	service.mu.Lock()
	service.sessions[session.SessionID].ExpiredAt = time.Now().Add(-1 * time.Hour)
	service.mu.Unlock()

	resp, err := service.Verify(&model.SpatioTemporalVerifyRequest{
		SessionID:      session.SessionID,
		SelectedOption: session.CorrectOption,
	})

	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "会话已过期", resp.Message)
}

func TestSpatioTemporalService_Verify_MaxAttempts(t *testing.T) {
	service := NewSpatioTemporalService()

	session, _ := service.Generate(&model.SpatioTemporalRequest{
		UserID: "max-attempts-user",
	})

	service.mu.Lock()
	service.sessions[session.SessionID].VerifyCount = 3
	service.mu.Unlock()

	resp, err := service.Verify(&model.SpatioTemporalVerifyRequest{
		SessionID:      session.SessionID,
		SelectedOption: session.CorrectOption,
	})

	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "验证次数已用完", resp.Message)
}

func TestSpatioTemporalService_GetSession(t *testing.T) {
	service := NewSpatioTemporalService()

	session, _ := service.Generate(&model.SpatioTemporalRequest{
		UserID: "get-session-user",
	})

	retrieved, exists := service.GetSession(session.SessionID)
	assert.True(t, exists)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.SessionID, retrieved.SessionID)
}

func TestSpatioTemporalService_GetSession_NotFound(t *testing.T) {
	service := NewSpatioTemporalService()

	retrieved, exists := service.GetSession("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

func TestSpatioTemporalService_RecordBehaviorFlow(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: time.Now().Unix(), Latitude: 39.9042, Longitude: 116.4074, Velocity: 10},
		{Timestamp: time.Now().Add(1 * time.Second).Unix(), Latitude: 39.9043, Longitude: 116.4075, Velocity: 12},
		{Timestamp: time.Now().Add(2 * time.Second).Unix(), Latitude: 39.9044, Longitude: 116.4076, Velocity: 11},
	}

	flow := service.RecordBehaviorFlow("test-user-flow", points)

	assert.NotNil(t, flow)
	assert.Equal(t, "test-user-flow", flow.UserID)
	assert.Len(t, flow.Points, 3)
	assert.Greater(t, flow.TotalDistance, 0.0)
}

func TestSpatioTemporalService_RecordBehaviorFlow_Empty(t *testing.T) {
	service := NewSpatioTemporalService()

	flow := service.RecordBehaviorFlow("test-user-empty", []model.SpatioTemporalPoint{})

	assert.NotNil(t, flow)
	assert.Len(t, flow.Points, 0)
}

func TestSpatioTemporalService_GetBehaviorFlow(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: time.Now().Unix(), Latitude: 39.9042, Longitude: 116.4074},
	}
	service.RecordBehaviorFlow("flow-user-123", points)

	flow, exists := service.GetBehaviorFlow("flow-user-123")
	assert.True(t, exists)
	assert.NotNil(t, flow)
}

func TestSpatioTemporalService_PredictTrajectory(t *testing.T) {
	service := NewSpatioTemporalService()

	historicalData := []model.SpatioTemporalPoint{
		{Timestamp: time.Now().Add(-10 * time.Second).Unix(), Latitude: 39.9042, Longitude: 116.4074},
		{Timestamp: time.Now().Add(-5 * time.Second).Unix(), Latitude: 39.9043, Longitude: 116.4075},
		{Timestamp: time.Now().Unix(), Latitude: 39.9044, Longitude: 116.4076},
	}

	req := &model.TrajectoryPredictionRequest{
		UserID:           "pred-user-123",
		HistoricalData:   historicalData,
		CurrentLocation:  []float64{39.9044, 116.4076},
		CurrentTime:      time.Now().Unix(),
		PredictionSteps:  3,
		PredictionMethod: "kalman_filter",
	}

	resp, err := service.PredictTrajectory(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Predictions, 3)
	assert.Greater(t, resp.Confidence, 0.0)
}

func TestSpatioTemporalService_PredictTrajectory_InsufficientData(t *testing.T) {
	service := NewSpatioTemporalService()

	req := &model.TrajectoryPredictionRequest{
		UserID:          "pred-user-insufficient",
		HistoricalData:  []model.SpatioTemporalPoint{
			{Timestamp: time.Now().Unix(), Latitude: 39.9042, Longitude: 116.4074},
		},
		PredictionSteps: 3,
	}

	resp, err := service.PredictTrajectory(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSpatioTemporalService_AssessRisk(t *testing.T) {
	service := NewSpatioTemporalService()

	behaviorData := &model.ContinuousBehaviorData{
		UserID:  "risk-user-123",
		SessionID: "risk-session-123",
		Points: []model.SpatioTemporalPoint{
			{Timestamp: time.Now().Add(-10 * time.Second).Unix(), Latitude: 39.9042, Longitude: 116.4074, Velocity: 10},
			{Timestamp: time.Now().Add(-5 * time.Second).Unix(), Latitude: 39.9043, Longitude: 116.4075, Velocity: 15},
			{Timestamp: time.Now().Unix(), Latitude: 39.9044, Longitude: 116.4076, Velocity: 12},
		},
		StartTime:     time.Now().Add(-10 * time.Second).Unix(),
		EndTime:       time.Now().Unix(),
		Duration:      10,
		AvgVelocity:   12.33,
		MaxVelocity:   15,
		MinVelocity:   10,
		TotalDistance: 0.05,
	}

	req := &model.RiskAssessmentRequest{
		UserID:       "risk-user-123",
		BehaviorData: behaviorData,
		Threshold:    0.7,
	}

	resp, err := service.AssessRisk(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.RiskScore)
	assert.NotEmpty(t, resp.RiskScore.RiskLevel)
	assert.Greater(t, resp.RiskScore.OverallScore, 0.0)
}

func TestSpatioTemporalService_ModelBehaviorFlow(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074, Velocity: 10},
		{Timestamp: 2000, Latitude: 39.9043, Longitude: 116.4075, Velocity: 12},
		{Timestamp: 3000, Latitude: 39.9044, Longitude: 116.4076, Velocity: 11},
		{Timestamp: 4000, Latitude: 39.9045, Longitude: 116.4077, Velocity: 13},
	}

	flow := service.modelBehaviorFlow("model-user", points)

	assert.NotNil(t, flow)
	assert.Equal(t, "model-user", flow.UserID)
	assert.Len(t, flow.Points, 4)
	assert.Equal(t, int64(1000), flow.StartTime)
	assert.Equal(t, int64(4000), flow.EndTime)
	assert.Greater(t, flow.TotalDistance, 0.0)
	assert.Greater(t, flow.AvgVelocity, 0.0)
}

func TestSpatioTemporalService_BuildTrajectory(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074, Velocity: 0},
		{Timestamp: 2000, Latitude: 39.9043, Longitude: 116.4075, Velocity: 10},
		{Timestamp: 3000, Latitude: 39.9044, Longitude: 116.4076, Velocity: 12},
	}

	trajectory := service.buildTrajectory(points)

	assert.Len(t, trajectory, 3)
	assert.GreaterOrEqual(t, trajectory[1].Velocity, 0.0)
}

func TestSpatioTemporalService_BuildTrajectory_Insufficient(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074},
	}

	trajectory := service.buildTrajectory(points)
	assert.Nil(t, trajectory)
}

func TestSpatioTemporalService_DetectBehaviorAnomalies(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074},
		{Timestamp: 1100, Latitude: 39.9043, Longitude: 116.4075},
		{Timestamp: 1200, Latitude: 39.9045, Longitude: 116.4080},
	}

	trajectory := []model.TrajectoryPoint{
		{Timestamp: 1000, X: 39.9042, Y: 116.4074, Velocity: 0},
		{Timestamp: 1100, X: 39.9043, Y: 116.4075, Velocity: 100, Acceleration: 1000},
		{Timestamp: 1200, X: 39.9045, Y: 116.4080, Velocity: 500, Acceleration: 4000, Jerk: 30000},
	}

	anomalies := service.detectBehaviorAnomalies(points, trajectory)
	assert.NotNil(t, anomalies)
}

func TestSpatioTemporalService_CalculateTurnAngle(t *testing.T) {
	service := NewSpatioTemporalService()

	p1 := model.TrajectoryPoint{X: 0, Y: 0}
	p2 := model.TrajectoryPoint{X: 1, Y: 1}
	p3 := model.TrajectoryPoint{X: 2, Y: 1}

	angle := service.calculateTurnAngle(p1, p2, p3)
	assert.Greater(t, angle, 0.0)
	assert.Less(t, angle, 180.0)
}

func TestSpatioTemporalService_CalculateTotalDistance(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Latitude: 39.9042, Longitude: 116.4074},
		{Latitude: 39.9142, Longitude: 116.4174},
	}

	distance := service.calculateTotalDistance(points)
	assert.Greater(t, distance, 0.0)
}

func TestSpatioTemporalService_CalculateVelocityStats(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074},
		{Timestamp: 2000, Latitude: 39.9043, Longitude: 116.4075},
		{Timestamp: 3000, Latitude: 39.9044, Longitude: 116.4076},
	}

	avg, max, min := service.calculateVelocityStats(points)
	assert.GreaterOrEqual(t, avg, 0.0)
	assert.GreaterOrEqual(t, max, 0.0)
	assert.GreaterOrEqual(t, min, 0.0)
}

func TestSpatioTemporalService_LinearPrediction(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074},
		{Timestamp: 2000, Latitude: 39.9043, Longitude: 116.4075},
	}

	predicted := service.linearPrediction(points, 3600)

	assert.Len(t, predicted, 2)
	assert.Greater(t, predicted[0], 0.0)
	assert.Greater(t, predicted[1], 0.0)
}

func TestSpatioTemporalService_CalculatePredictionConfidence(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Latitude: 39.9042, Longitude: 116.4074},
		{Latitude: 39.9043, Longitude: 116.4075},
		{Latitude: 39.9044, Longitude: 116.4076},
	}

	confidence := service.calculatePredictionConfidence(points)
	assert.Greater(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 0.95)
}

func TestSpatioTemporalService_CalculatePredictionConfidence_Insufficient(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Latitude: 39.9042, Longitude: 116.4074},
	}

	confidence := service.calculatePredictionConfidence(points)
	assert.Equal(t, 0.5, confidence)
}

func TestSpatioTemporalService_ExtractPredictionFeatures(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074, Velocity: 10},
		{Timestamp: 2000, Latitude: 39.9043, Longitude: 116.4075, Velocity: 12},
	}

	features := service.extractPredictionFeatures(points)
	assert.NotNil(t, features)
	assert.Contains(t, features, "avg_velocity")
	assert.Contains(t, features, "centroid_lat")
}

func TestSpatioTemporalService_CheckPredictionAnomalies(t *testing.T) {
	service := NewSpatioTemporalService()

	points := []model.SpatioTemporalPoint{
		{Latitude: 39.9042, Longitude: 116.4074},
		{Latitude: 39.9043, Longitude: 116.4075},
	}

	predictedLocation := []float64{39.9042, 116.4074}

	indicators := service.checkPredictionAnomalies(points, predictedLocation)
	assert.NotNil(t, indicators)
}

func TestSpatioTemporalService_CalculateRiskScore(t *testing.T) {
	service := NewSpatioTemporalService()

	pattern := &model.SpatioTemporalPattern{
		PatternID:    "test-pattern",
		PatternType:  model.TimePatternDaily,
		AnomalyScore: 0.2,
	}

	riskScore := service.calculateRiskScore("risk-user", pattern, nil)

	assert.NotNil(t, riskScore)
	assert.Equal(t, "risk-user", riskScore.UserID)
	assert.Greater(t, riskScore.OverallScore, 0.0)
	assert.Less(t, riskScore.OverallScore, 1.0)
	assert.NotEmpty(t, riskScore.RiskLevel)
}

func TestSpatioTemporalService_GenerateRiskRecommendations(t *testing.T) {
	service := NewSpatioTemporalService()

	recommendations := service.generateRiskRecommendations(nil, nil)
	assert.NotNil(t, recommendations)
	assert.Greater(t, len(recommendations), 0)
}

func TestSpatioTemporalService_GetRiskScore(t *testing.T) {
	service := NewSpatioTemporalService()

	_, _ = service.Generate(&model.SpatioTemporalRequest{
		UserID: "risk-score-user",
	})

	session, _ := service.GetSession("")
	_ = session

	score, exists := service.GetRiskScore("")
	assert.False(t, exists)
	assert.Nil(t, score)
}

func TestSpatioTemporalService_GetPrediction(t *testing.T) {
	service := NewSpatioTemporalService()

	pred, exists := service.GetPrediction("")
	assert.False(t, exists)
	assert.Nil(t, pred)
}

func TestHaversineDistance(t *testing.T) {
	lat1, lon1 := 39.9042, 116.4074
	lat2, lon2 := 39.9142, 116.4174

	distance := haversineDistance(lat1, lon1, lat2, lon2)

	assert.Greater(t, distance, 0.0)
	assert.Less(t, distance, 50.0)
}

func TestCalculateCentroidST(t *testing.T) {
	points := []model.SpatioTemporalPoint{
		{Latitude: 39.9042, Longitude: 116.4074},
		{Latitude: 39.9142, Longitude: 116.4174},
		{Latitude: 39.9242, Longitude: 116.4274},
	}

	centroid := calculateCentroidST(points)

	assert.Len(t, centroid, 2)
	assert.Greater(t, centroid[0], 39.9)
	assert.Greater(t, centroid[1], 116.4)
}

func TestCalculateCentroidST_Empty(t *testing.T) {
	centroid := calculateCentroidST([]model.SpatioTemporalPoint{})
	assert.Equal(t, []float64{0, 0}, centroid)
}

func TestCalculateVelocityStatsSimple(t *testing.T) {
	points := []model.SpatioTemporalPoint{
		{Timestamp: 1000, Latitude: 39.9042, Longitude: 116.4074},
		{Timestamp: 2000, Latitude: 39.9043, Longitude: 116.4075},
	}

	avg, max := calculateVelocityStatsSimple(points)
	assert.GreaterOrEqual(t, avg, 0.0)
	assert.GreaterOrEqual(t, max, 0.0)
}

func TestSerializeSession(t *testing.T) {
	session := &model.SpatioTemporalSession{
		SessionID: "test-session-123",
		UserID:    "test-user",
		Status:    "pending",
	}

	data, err := SerializeSession(session)
	assert.NoError(t, err)
	assert.Greater(t, len(data), 0)

	deserialized, err := DeserializeSpatioTemporalSession(data)
	assert.NoError(t, err)
	assert.Equal(t, session.SessionID, deserialized.SessionID)
}

func TestDeserializeSpatioTemporalSession_Invalid(t *testing.T) {
	_, err := DeserializeSpatioTemporalSession([]byte("invalid json"))
	assert.Error(t, err)
}

func TestSpatioTemporalPoint_Fields(t *testing.T) {
	point := model.SpatioTemporalPoint{
		Timestamp:  1000,
		Latitude:   39.9042,
		Longitude:  116.4074,
		Altitude:   50.0,
		IPAddress:  "192.168.1.1",
		DeviceID:   "device-123",
		Accuracy:   model.LocationAccuracyGPS,
		Confidence: 0.9,
		Velocity:   10.5,
		Heading:   90.0,
	}

	assert.Equal(t, int64(1000), point.Timestamp)
	assert.Equal(t, 39.9042, point.Latitude)
	assert.Equal(t, model.LocationAccuracyGPS, point.Accuracy)
}

func TestBehaviorFlow_Fields(t *testing.T) {
	flow := model.BehaviorFlow{
		FlowID:        "flow-123",
		UserID:        "user-456",
		StartTime:     1000,
		EndTime:       2000,
		TotalDistance: 5.5,
		AvgVelocity:   20.0,
		MaxVelocity:   30.0,
		MinVelocity:   10.0,
		RiskScore:     0.3,
		PatternType:   model.TimePatternDaily,
	}

	assert.Equal(t, "flow-123", flow.FlowID)
	assert.Equal(t, int64(1000), flow.StartTime)
	assert.Equal(t, 5.5, flow.TotalDistance)
}

func TestRiskScore_Fields(t *testing.T) {
	score := model.RiskScore{
		ScoreID:         "rs-123",
		UserID:          "user-456",
		OverallScore:    0.75,
		LocationScore:   0.8,
		TimeScore:       0.7,
		BehaviorScore:   0.75,
		VelocityScore:   0.85,
		PatternScore:    0.9,
		RiskLevel:       "medium",
		RiskFactors:     []string{"factor1", "factor2"},
		Recommendations: []string{"rec1", "rec2"},
		CalculatedAt:    1000,
		ValidUntil:      2000,
	}

	assert.Equal(t, "rs-123", score.ScoreID)
	assert.Equal(t, 0.75, score.OverallScore)
	assert.Equal(t, "medium", score.RiskLevel)
	assert.Len(t, score.RiskFactors, 2)
}

func TestBehaviorPrediction_Fields(t *testing.T) {
	pred := model.BehaviorPrediction{
		PredictionID:      "pred-123",
		UserID:            "user-456",
		PredictedLocation: []float64{39.9042, 116.4074},
		PredictedTime:     2000,
		PredictionWindow:  3600,
		Confidence:        0.85,
		Method:            "kalman_filter",
	}

	assert.Equal(t, "pred-123", pred.PredictionID)
	assert.Len(t, pred.PredictedLocation, 2)
	assert.Equal(t, 0.85, pred.Confidence)
}

func TestSpatioTemporalAnalytics_Fields(t *testing.T) {
	analytics := model.SpatioTemporalAnalytics{
		LocationConfidence:   0.9,
		TimeConsistency:      0.8,
		BehaviorConsistency:  0.75,
		VelocityConsistency: 0.85,
		RiskLevel:           "low",
		RiskFactors:         []string{},
	}

	assert.Equal(t, 0.9, analytics.LocationConfidence)
	assert.Equal(t, "low", analytics.RiskLevel)
}

func TestTimeWindow_Fields(t *testing.T) {
	window := model.TimeWindow{
		StartTime: 1000,
		EndTime:   2000,
		Duration:  1000,
	}

	assert.Equal(t, int64(1000), window.StartTime)
	assert.Equal(t, int64(2000), window.EndTime)
	assert.Equal(t, int64(1000), window.Duration)
}
