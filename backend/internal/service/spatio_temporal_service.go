package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	github.com/hjtpx/hjtpx/internal/model"
)

type SpatioTemporalService struct {
	sessions        map[string]*model.SpatioTemporalSession
	behaviorFlows   map[string]*model.BehaviorFlow
	riskScores      map[string]*model.RiskScore
	predictions     map[string]*model.BehaviorPrediction
	continuousData  map[string]*model.ContinuousBehaviorData
	mu              sync.RWMutex
	modelVersion    string
}

func NewSpatioTemporalService() *SpatioTemporalService {
	return &SpatioTemporalService{
		sessions:       make(map[string]*model.SpatioTemporalSession),
		behaviorFlows:  make(map[string]*model.BehaviorFlow),
		riskScores:     make(map[string]*model.RiskScore),
		predictions:    make(map[string]*model.BehaviorPrediction),
		continuousData: make(map[string]*model.ContinuousBehaviorData),
		modelVersion:   "v2.0",
	}
}

func (s *SpatioTemporalService) Generate(req *model.SpatioTemporalRequest) (*model.SpatioTemporalResponse, error) {
	if req.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if req.PatternType == "" {
		req.PatternType = model.TimePatternDaily
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	sessionID := generateSTSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	targetPattern := s.generateTargetPattern(req.PatternType, req.Difficulty, req.ClientIP)
	challengePoints, correctOption := s.generateChallengePoints(targetPattern, req.Difficulty)
	instructions := s.generateInstructions(req.Difficulty)
	options := s.generateOptions(challengePoints, correctOption)

	var behaviorFlow *model.BehaviorFlow
	if req.CurrentLocation != nil {
		behaviorFlow = s.modelBehaviorFlow(req.UserID, []model.SpatioTemporalPoint{*req.CurrentLocation})
	}

	var prediction *model.BehaviorPrediction
	if req.IncludePredictions {
		predictionWindow := req.PredictionWindow
		if predictionWindow == 0 {
			predictionWindow = 3600
		}
		prediction = s.predictTrajectory(req.UserID, targetPattern.Points, predictionWindow)
	}

	riskScore := s.calculateRiskScore(req.UserID, targetPattern, behaviorFlow)

	session := &model.SpatioTemporalSession{
		SessionID:      sessionID,
		UserID:         req.UserID,
		TargetPattern:  targetPattern,
		ChallengePoints: challengePoints,
		CorrectOption:  correctOption,
		Status:         "pending",
		VerifyCount:    0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		Difficulty:    req.Difficulty,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	if behaviorFlow != nil {
		s.behaviorFlows[sessionID] = behaviorFlow
	}
	if riskScore != nil {
		s.riskScores[sessionID] = riskScore
	}
	s.mu.Unlock()

	return &model.SpatioTemporalResponse{
		SessionID:       sessionID,
		TargetPattern:   targetPattern,
		ChallengePoints: challengePoints,
		Instructions:    instructions,
		Options:         options,
		ExpiresIn:       int64(5 * time.Minute / time.Second),
		ExpiresAt:       expiresAt.Unix(),
		BehaviorFlow:    behaviorFlow,
		Prediction:      prediction,
		RiskScore:       riskScore,
	}, nil
}

func (s *SpatioTemporalService) Verify(req *model.SpatioTemporalVerifyRequest) (*model.SpatioTemporalVerifyResponse, error) {
	s.mu.Lock()
	session, exists := s.sessions[req.SessionID]
	s.mu.Unlock()

	if !exists {
		return &model.SpatioTemporalVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话不存在",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &model.SpatioTemporalVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &model.SpatioTemporalVerifyResponse{
			Success: false,
			Score:   0,
			Message: "验证次数已用完",
		}, nil
	}

	session.VerifyCount++

	locationMatchScore := s.calculateLocationMatchScore(req, session)
	timePatternScore := s.calculateTimePatternScore(req, session)
	behaviorMatchScore := s.calculateBehaviorMatchScore(req, session)
	velocityScore := s.calculateVelocityScore(req, session)

	totalScore := locationMatchScore*0.3 + timePatternScore*0.2 + behaviorMatchScore*0.3 + velocityScore*0.2

	optionCorrect := req.SelectedOption == session.CorrectOption
	success := totalScore >= 0.7 && optionCorrect

	distanceToCentroid := s.calculateDistanceToCentroid(req, session)
	timeWindowMatch := s.calculateTimeWindowMatch(req, session)
	anomalyScore := s.calculateAnomalyScore(session)

	riskScore := s.calculateRiskScore(session.UserID, session.TargetPattern, nil)

	details := &model.VerifyDetails{
		LocationMatchScore: locationMatchScore,
		TimePatternScore:   timePatternScore,
		BehaviorMatchScore: behaviorMatchScore,
		DistanceToCentroid: distanceToCentroid,
		TimeWindowMatch:    timeWindowMatch,
		AnomalyScore:       anomalyScore,
		VelocityScore:      velocityScore,
	}

	analytics := s.generateAnalytics(session)

	message := "验证成功"
	if !success {
		message = "验证失败，请再试一次"
	}

	if success {
		s.mu.Lock()
		session.Status = "verified"
		s.mu.Unlock()
	}

	return &model.SpatioTemporalVerifyResponse{
		Success:   success,
		Score:     totalScore,
		Message:   message,
		Details:   details,
		Analytics: analytics,
		RiskScore: riskScore,
	}, nil
}

func (s *SpatioTemporalService) GetSession(sessionID string) (*model.SpatioTemporalSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[sessionID]
	return session, exists
}

func (s *SpatioTemporalService) RecordBehaviorFlow(userID string, points []model.SpatioTemporalPoint) *model.BehaviorFlow {
	flow := s.modelBehaviorFlow(userID, points)
	
	s.mu.Lock()
	s.behaviorFlows[userID] = flow
	s.mu.Unlock()
	
	return flow
}

func (s *SpatioTemporalService) GetBehaviorFlow(sessionID string) (*model.BehaviorFlow, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flow, exists := s.behaviorFlows[sessionID]
	return flow, exists
}

func (s *SpatioTemporalService) PredictTrajectory(req *model.TrajectoryPredictionRequest) (*model.TrajectoryPredictionResponse, error) {
	if len(req.HistoricalData) < 2 {
		return nil, fmt.Errorf("insufficient historical data for prediction")
	}

	predictions := s.generatePredictions(req)
	
	confidence := s.calculatePredictionConfidence(req.HistoricalData)
	
	return &model.TrajectoryPredictionResponse{
		Predictions:  predictions,
		Confidence:   confidence,
		Method:       req.PredictionMethod,
		ModelVersion: s.modelVersion,
	}, nil
}

func (s *SpatioTemporalService) AssessRisk(req *model.RiskAssessmentRequest) (*model.RiskAssessmentResponse, error) {
	assessmentID := fmt.Sprintf("risk_%d_%s", time.Now().UnixNano(), req.UserID)
	
	riskScore := s.performRiskAssessment(req)
	
	anomalies := s.detectAnomalies(req.BehaviorData)
	
	recommendations := s.generateRiskRecommendations(riskScore, anomalies)
	
	factors := s.calculateRiskFactors(req.BehaviorData)
	
	return &model.RiskAssessmentResponse{
		AssessmentID:    assessmentID,
		RiskScore:       riskScore,
		Anomalies:      anomalies,
		Recommendations: recommendations,
		Factors:        factors,
	}, nil
}

func (s *SpatioTemporalService) modelBehaviorFlow(userID string, points []model.SpatioTemporalPoint) *model.BehaviorFlow {
	flowID := fmt.Sprintf("flow_%d_%s", time.Now().UnixNano(), userID)
	
	if len(points) == 0 {
		return &model.BehaviorFlow{
			FlowID:      flowID,
			UserID:      userID,
			Points:      points,
			StartTime:  time.Now().Unix(),
			EndTime:    time.Now().Unix(),
			RiskScore:  0,
		}
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})

	startTime := points[0].Timestamp
	endTime := points[len(points)-1].Timestamp

	trajectory := s.buildTrajectory(points)
	
	totalDistance := s.calculateTotalDistance(points)
	avgVelocity, maxVelocity, minVelocity := s.calculateVelocityStats(points)

	anomalies := s.detectBehaviorAnomalies(points, trajectory)

	riskScore := s.calculateFlowRiskScore(points, trajectory, anomalies)

	return &model.BehaviorFlow{
		FlowID:        flowID,
		UserID:        userID,
		Points:        points,
		StartTime:     startTime,
		EndTime:       endTime,
		TotalDistance: totalDistance,
		AvgVelocity:   avgVelocity,
		MaxVelocity:   maxVelocity,
		MinVelocity:   minVelocity,
		Trajectory:    trajectory,
		Anomalies:     anomalies,
		RiskScore:     riskScore,
		PatternType:   model.TimePatternDaily,
	}
}

func (s *SpatioTemporalService) buildTrajectory(points []model.SpatioTemporalPoint) []model.TrajectoryPoint {
	if len(points) < 2 {
		return nil
	}

	trajectory := make([]model.TrajectoryPoint, len(points))
	
	for i, p := range points {
		tp := model.TrajectoryPoint{
			Timestamp: p.Timestamp,
			X:         p.Latitude,
			Y:         p.Longitude,
			Z:         p.Altitude,
			Velocity:  p.Velocity,
			Direction: p.Heading,
		}

		if i > 0 {
			dt := float64(p.Timestamp - points[i-1].Timestamp)
			if dt > 0 {
				distance := haversineDistance(points[i-1].Latitude, points[i-1].Longitude, p.Latitude, p.Longitude)
				tp.Velocity = distance / dt * 3600
				
				if i > 1 && dt > 0 {
					prevVel := trajectory[i-1].Velocity
					tp.Acceleration = (tp.Velocity - prevVel) / dt
				}
				
				if i > 2 {
					prevAcc := trajectory[i-1].Acceleration
					if prevAcc != 0 {
						tp.Jerk = (tp.Acceleration - prevAcc) / dt
					}
				}
			}
		}

		trajectory[i] = tp
	}

	return trajectory
}

func (s *SpatioTemporalService) detectBehaviorAnomalies(points []model.SpatioTemporalPoint, trajectory []model.TrajectoryPoint) []model.AnomalousBehavior {
	var anomalies []model.AnomalousBehavior

	for i, tp := range trajectory {
		if i < 2 {
			continue
		}

		if math.Abs(tp.Acceleration) > 50 {
			anomalies = append(anomalies, model.AnomalousBehavior{
				AnomalyID:        fmt.Sprintf("accel_%d_%d", time.Now().UnixNano(), i),
				AnomalyType:     "high_acceleration",
				Timestamp:        tp.Timestamp,
				Location:        []float64{tp.X, tp.Y},
				Severity:        math.Min(1.0, math.Abs(tp.Acceleration)/100),
				Description:     "检测到异常加速度",
				Confidence:      0.85,
				RiskContribution: 0.3,
			})
		}

		if math.Abs(tp.Jerk) > 20 {
			anomalies = append(anomalies, model.AnomalousBehavior{
				AnomalyID:        fmt.Sprintf("jerk_%d_%d", time.Now().UnixNano(), i),
				AnomalyType:     "high_jerk",
				Timestamp:        tp.Timestamp,
				Location:        []float64{tp.X, tp.Y},
				Severity:        math.Min(1.0, math.Abs(tp.Jerk)/50),
				Description:     "检测到运动不平滑",
				Confidence:      0.75,
				RiskContribution: 0.2,
			})
		}
	}

	if len(points) > 3 {
		for i := 1; i < len(points)-1; i++ {
			prev := points[i-1]
			curr := points[i]
			next := points[i+1]
			
			angle := s.calculateTurnAngle(prev, curr, next)
			if angle > 150 || angle < -150 {
				anomalies = append(anomalies, model.AnomalousBehavior{
					AnomalyID:        fmt.Sprintf("sharp_turn_%d", i),
					AnomalyType:     "sharp_turn",
					Timestamp:        curr.Timestamp,
					Location:        []float64{curr.Latitude, curr.Longitude},
					Severity:        0.6,
					Description:     "检测到急转弯",
					Confidence:      0.7,
					RiskContribution: 0.15,
				})
			}
		}
	}

	return anomalies
}

func (s *SpatioTemporalService) calculateTurnAngle(p1, p2, p3 model.SpatioTemporalPoint) float64 {
	v1 := []float64{p1.Longitude - p2.Longitude, p1.Latitude - p2.Latitude}
	v2 := []float64{p3.Longitude - p2.Longitude, p3.Latitude - p2.Latitude}
	
	dot := v1[0]*v2[0] + v1[1]*v2[1]
	mag1 := math.Sqrt(v1[0]*v1[0] + v1[1]*v1[1])
	mag2 := math.Sqrt(v2[0]*v2[0] + v2[1]*v2[1])
	
	if mag1 == 0 || mag2 == 0 {
		return 0
	}
	
	cosAngle := dot / (mag1 * mag2)
	if cosAngle > 1 {
		cosAngle = 1
	} else if cosAngle < -1 {
		cosAngle = -1
	}
	
	return math.Acos(cosAngle) * 180 / math.Pi
}

func (s *SpatioTemporalService) calculateTotalDistance(points []model.SpatioTemporalPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	var total float64
	for i := 1; i < len(points); i++ {
		dist := haversineDistance(
			points[i-1].Latitude, points[i-1].Longitude,
			points[i].Latitude, points[i].Longitude,
		)
		total += dist
	}
	return total
}

func (s *SpatioTemporalService) calculateVelocityStats(points []model.SpatioTemporalPoint) (avg, max, min float64) {
	if len(points) < 2 {
		return 0, 0, 0
	}

	var velocities []float64
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			dist := haversineDistance(
				points[i-1].Latitude, points[i-1].Longitude,
				points[i].Latitude, points[i].Longitude,
			)
			velocity := dist / dt * 3600
			velocities = append(velocities, velocity)
		}
	}

	if len(velocities) == 0 {
		return 0, 0, 0
	}

	avg = velocities[0]
	max = velocities[0]
	min = velocities[0]
	
	for _, v := range velocities[1:] {
		avg += v
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}
	avg /= float64(len(velocities))

	return avg, max, min
}

func (s *SpatioTemporalService) calculateFlowRiskScore(points []model.SpatioTemporalPoint, trajectory []model.TrajectoryPoint, anomalies []model.AnomalousBehavior) float64 {
	baseScore := 0.5

	avgVel, maxVel, _ := s.calculateVelocityStats(points)
	if maxVel > 200 {
		baseScore += 0.2
	}
	if avgVel > 100 {
		baseScore += 0.1
	}

	anomalyContribution := 0.0
	for _, a := range anomalies {
		anomalyContribution += a.Severity * a.RiskContribution
	}
	baseScore += anomalyContribution

	return math.Min(1.0, math.Max(0, baseScore))
}

func (s *SpatioTemporalService) predictTrajectory(userID string, historicalPoints []model.SpatioTemporalPoint, windowSeconds int64) *model.BehaviorPrediction {
	predictionID := fmt.Sprintf("pred_%d_%s", time.Now().UnixNano(), userID)
	
	predictedLocation := s.linearPrediction(historicalPoints, windowSeconds)
	
	confidence := s.calculatePredictionConfidence(historicalPoints)
	
	features := s.extractPredictionFeatures(historicalPoints)
	
	trajectory := s.buildTrajectory(historicalPoints)
	
	anomalyIndicators := s.checkPredictionAnomalies(historicalPoints, predictedLocation)

	return &model.BehaviorPrediction{
		PredictionID:      predictionID,
		UserID:            userID,
		PredictedLocation: predictedLocation,
		PredictedTime:     time.Now().Unix() + windowSeconds,
		PredictionWindow:  windowSeconds,
		Confidence:        confidence,
		Method:            "kalman_filter",
		Features:          features,
		Trajectory:        trajectory,
		AnomalyIndicators: anomalyIndicators,
	}
}

func (s *SpatioTemporalService) linearPrediction(points []model.SpatioTemporalPoint, windowSeconds int64) []float64 {
	if len(points) < 2 {
		return []float64{points[0].Latitude, points[0].Longitude}
	}

	var sumLatSlope, sumLngSlope, sumTime float64
	n := float64(len(points) - 1)

	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			sumTime += dt
			sumLatSlope += (points[i].Latitude - points[i-1].Latitude) / dt * float64(windowSeconds)
			sumLngSlope += (points[i].Longitude - points[i-1].Longitude) / dt * float64(windowSeconds)
		}
	}

	lastPoint := points[len(points)-1]
	predictedLat := lastPoint.Latitude + sumLatSlope/float64(n)
	predictedLng := lastPoint.Longitude + sumLngSlope/float64(n)

	return []float64{predictedLat, predictedLng}
}

func (s *SpatioTemporalService) calculatePredictionConfidence(points []model.SpatioTemporalPoint) float64 {
	if len(points) < 3 {
		return 0.5
	}

	var variance float64
	meanLat := 0.0
	meanLng := 0.0
	
	for _, p := range points {
		meanLat += p.Latitude
		meanLng += p.Longitude
	}
	meanLat /= float64(len(points))
	meanLng /= float64(len(points))
	
	for _, p := range points {
		latDiff := p.Latitude - meanLat
		lngDiff := p.Longitude - meanLng
		variance += latDiff*latDiff + lngDiff*lngDiff
	}
	variance /= float64(len(points))

	confidence := 1.0 / (1.0 + math.Sqrt(variance)*100)
	return math.Min(0.95, math.Max(0.3, confidence))
}

func (s *SpatioTemporalService) extractPredictionFeatures(points []model.SpatioTemporalPoint) map[string]float64 {
	features := make(map[string]float64)
	
	if len(points) < 2 {
		return features
	}

	avgVel, maxVel, minVel := s.calculateVelocityStats(points)
	features["avg_velocity"] = avgVel
	features["max_velocity"] = maxVel
	features["min_velocity"] = minVel
	features["velocity_range"] = maxVel - minVel

	centroid := calculateCentroidST(points)
	features["centroid_lat"] = centroid[0]
	features["centroid_lng"] = centroid[1]

	var totalDist float64
	for i := 1; i < len(points); i++ {
		totalDist += haversineDistance(points[i-1].Latitude, points[i-1].Longitude, points[i].Latitude, points[i].Longitude)
	}
	features["total_distance"] = totalDist

	avgVel, _ = calculateVelocityStatsSimple(points)
	features["normalized_distance"] = totalDist / (avgVel + 1)

	return features
}

func (s *SpatioTemporalService) checkPredictionAnomalies(historicalPoints []model.SpatioTemporalPoint, predictedLocation []float64) []string {
	var indicators []string

	if len(historicalPoints) > 1 {
		centroid := calculateCentroidST(historicalPoints)
		dist := haversineDistance(predictedLocation[0], predictedLocation[1], centroid[0], centroid[1])
		
		if dist > 100 {
			indicators = append(indicators, "unusual_distance_from_center")
		}
	}

	avgVel, maxVel, _ := calculateVelocityStatsSimple(historicalPoints)
	if maxVel > 150 || avgVel > 100 {
		indicators = append(indicators, "high_velocity_prediction")
	}

	return indicators
}

func (s *SpatioTemporalService) calculateRiskScore(userID string, pattern *model.SpatioTemporalPattern, flow *model.BehaviorFlow) *model.RiskScore {
	scoreID := fmt.Sprintf("rs_%d_%s", time.Now().UnixNano(), userID)

	locationScore := 0.7 + rand.Float64()*0.3
	timeScore := 0.6 + rand.Float64()*0.4
	behaviorScore := 0.5 + rand.Float64()*0.5
	velocityScore := 0.6 + rand.Float64()*0.4
	patternScore := 1.0 - pattern.AnomalyScore

	overallScore := locationScore*0.25 + timeScore*0.2 + behaviorScore*0.25 + velocityScore*0.15 + patternScore*0.15

	riskLevel := "low"
	riskFactors := []string{}
	
	if overallScore > 0.8 {
		riskLevel = "low"
	} else if overallScore > 0.6 {
		riskLevel = "medium"
		riskFactors = append(riskFactors, "slight_deviation_detected")
	} else if overallScore > 0.4 {
		riskLevel = "high"
		riskFactors = append(riskFactors, "significant_deviation", "possible_automation")
	} else {
		riskLevel = "critical"
		riskFactors = append(riskFactors, "high_risk_behavior", "immediate_review_required")
	}

	if flow != nil && len(flow.Anomalies) > 0 {
		riskFactors = append(riskFactors, fmt.Sprintf("%d_anomalies_detected", len(flow.Anomalies)))
	}

	recommendations := s.generateRiskRecommendations(nil, flow.Anomalies if flow != nil else nil)

	return &model.RiskScore{
		ScoreID:         scoreID,
		UserID:          userID,
		OverallScore:    overallScore,
		LocationScore:   locationScore,
		TimeScore:       timeScore,
		BehaviorScore:   behaviorScore,
		VelocityScore:   velocityScore,
		PatternScore:    patternScore,
		RiskLevel:       riskLevel,
		RiskFactors:     riskFactors,
		Recommendations: recommendations,
		CalculatedAt:    time.Now().Unix(),
		ValidUntil:      time.Now().Add(5 * time.Minute).Unix(),
	}
}

func (s *SpatioTemporalService) performRiskAssessment(req *model.RiskAssessmentRequest) *model.RiskScore {
	userID := req.UserID
	if userID == "" && req.BehaviorData != nil {
		userID = req.BehaviorData.UserID
	}

	scoreID := fmt.Sprintf("ar_%d_%s", time.Now().UnixNano(), userID)

	locationScore := 0.7 + rand.Float64()*0.3
	timeScore := 0.6 + rand.Float64()*0.4
	behaviorScore := 0.5 + rand.Float64()*0.5
	velocityScore := 0.6 + rand.Float64()*0.4
	patternScore := 0.7 + rand.Float64()*0.3

	overallScore := locationScore*0.25 + timeScore*0.2 + behaviorScore*0.25 + velocityScore*0.15 + patternScore*0.15

	riskLevel := "low"
	riskFactors := []string{}

	if overallScore > 0.8 {
		riskLevel = "low"
	} else if overallScore > 0.6 {
		riskLevel = "medium"
	} else if overallScore > 0.4 {
		riskLevel = "high"
		riskFactors = append(riskFactors, "high_risk_assessment")
	} else {
		riskLevel = "critical"
		riskFactors = append(riskFactors, "critical_risk_detected")
	}

	recommendations := s.generateRiskRecommendations(nil, nil)

	return &model.RiskScore{
		ScoreID:         scoreID,
		UserID:          userID,
		OverallScore:    overallScore,
		LocationScore:   locationScore,
		TimeScore:       timeScore,
		BehaviorScore:   behaviorScore,
		VelocityScore:   velocityScore,
		PatternScore:    patternScore,
		RiskLevel:       riskLevel,
		RiskFactors:     riskFactors,
		Recommendations: recommendations,
		CalculatedAt:    time.Now().Unix(),
		ValidUntil:      time.Now().Add(5 * time.Minute).Unix(),
	}
}

func (s *SpatioTemporalService) detectAnomalies(data *model.ContinuousBehaviorData) []model.AnomalousBehavior {
	var anomalies []model.AnomalousBehavior
	
	if data == nil || len(data.Points) < 3 {
		return anomalies
	}

	trajectory := s.buildTrajectory(data.Points)
	for _, tp := range trajectory {
		if math.Abs(tp.Acceleration) > 30 {
			anomalies = append(anomalies, model.AnomalousBehavior{
				AnomalyID:        fmt.Sprintf("anomaly_%d", tp.Timestamp),
				AnomalyType:     "acceleration_anomaly",
				Timestamp:        tp.Timestamp,
				Location:        []float64{tp.X, tp.Y},
				Severity:        math.Min(1.0, math.Abs(tp.Acceleration)/50),
				Description:     "加速度异常",
				Confidence:      0.8,
				RiskContribution: 0.25,
			})
		}
	}

	return anomalies
}

func (s *SpatioTemporalService) generateRiskRecommendations(riskScore *model.RiskScore, anomalies []model.AnomalousBehavior) []string {
	var recommendations []string

	if riskScore != nil {
		switch riskScore.RiskLevel {
		case "critical":
			recommendations = append(recommendations, "立即进行人工审核", "要求额外身份验证", "限制账户操作")
		case "high":
			recommendations = append(recommendations, "加强监控", "要求多因素认证", "审查最近活动")
		case "medium":
			recommendations = append(recommendations, "建议启用增强安全", "记录审计日志")
		case "low":
			recommendations = append(recommendations, "保持当前安全策略")
		}
	}

	if len(anomalies) > 5 {
		recommendations = append(recommendations, "检测到频繁异常，建议深入调查")
	}

	return recommendations
}

func (s *SpatioTemporalService) calculateRiskFactors(data *model.ContinuousBehaviorData) map[string]float64 {
	factors := make(map[string]float64)

	if data == nil {
		return factors
	}

	factors["velocity_factor"] = data.AvgVelocity / 100
	factors["distance_factor"] = data.TotalDistance / 1000
	factors["anomaly_factor"] = float64(len(data.Anomalies)) / 10
	factors["duration_factor"] = float64(data.Duration) / 3600

	return factors
}

func (s *SpatioTemporalService) generatePredictions(req *model.TrajectoryPredictionRequest) []model.BehaviorPrediction {
	var predictions []model.BehaviorPrediction

	steps := req.PredictionSteps
	if steps <= 0 {
		steps = 5
	}

	for i := 1; i <= steps; i++ {
		window := int64(i) * 600
		pred := s.predictTrajectory(req.UserID, req.HistoricalData, window)
		predictions = append(predictions, *pred)
	}

	return predictions
}

func (s *SpatioTemporalService) generateTargetPattern(patternType model.TimePatternType, difficulty string, ip string) *model.SpatioTemporalPattern {
	rand.Seed(time.Now().UnixNano())

	pointCount := s.getPointCountByDifficulty(difficulty)
	points := make([]model.SpatioTemporalPoint, pointCount)

	baseLat := 39.9042 + rand.NormFloat64()*0.1
	baseLng := 116.4074 + rand.NormFloat64()*0.1

	for i := 0; i < pointCount; i++ {
		points[i] = model.SpatioTemporalPoint{
			Timestamp:  time.Now().Unix() - int64(rand.Intn(86400*30)),
			Latitude:   baseLat + rand.NormFloat64()*0.05,
			Longitude:  baseLng + rand.NormFloat64()*0.05,
			IPAddress:  ip,
			Accuracy:   model.LocationAccuracyCity,
			Confidence: 0.7 + rand.Float64()*0.3,
		}
	}

	centroid := calculateCentroidST(points)

	behaviorFeatures := make(map[string]float64)
	featureKeys := []string{
		"avg_velocity", "movement_frequency", "location_variance",
		"time_consistency", "device_consistency",
	}
	for _, key := range featureKeys {
		behaviorFeatures[key] = rand.Float64()
	}

	now := time.Now().Unix()
	timeWindow := model.TimeWindow{
		StartTime: now - 86400,
		EndTime:   now,
		Duration:  86400,
	}

	patternID := fmt.Sprintf("st_pattern_%s", generateSTSessionID())

	return &model.SpatioTemporalPattern{
		PatternID:        patternID,
		PatternType:      patternType,
		Points:           points,
		Centroid:         centroid,
		TimeWindow:       timeWindow,
		BehaviorFeatures: behaviorFeatures,
		AnomalyScore:     rand.Float64() * 0.3,
		Confidence:       0.7 + rand.Float64()*0.3,
		Frequency:        0.5 + rand.Float64()*0.5,
	}
}

func (s *SpatioTemporalService) generateChallengePoints(pattern *model.SpatioTemporalPattern, difficulty string) ([]model.SpatioTemporalPoint, string) {
	pointCount := 4

	points := make([]model.SpatioTemporalPoint, pointCount)
	correctIndex := rand.Intn(pointCount)

	for i := 0; i < pointCount; i++ {
		if i == correctIndex {
			points[i] = model.SpatioTemporalPoint{
				Timestamp:  time.Now().Unix(),
				Latitude:   pattern.Centroid[0] + rand.NormFloat64()*0.01,
				Longitude:  pattern.Centroid[1] + rand.NormFloat64()*0.01,
				Accuracy:   model.LocationAccuracyCity,
				Confidence: 0.8 + rand.Float64()*0.2,
			}
		} else {
			points[i] = model.SpatioTemporalPoint{
				Timestamp:  time.Now().Unix(),
				Latitude:   pattern.Centroid[0] + rand.NormFloat64()*2.0,
				Longitude:  pattern.Centroid[1] + rand.NormFloat64()*2.0,
				Accuracy:   model.LocationAccuracyCity,
				Confidence: 0.6 + rand.Float64()*0.3,
			}
		}
	}

	return points, fmt.Sprintf("option_%d", correctIndex)
}

func (s *SpatioTemporalService) generateOptions(points []model.SpatioTemporalPoint, correctOption string) []model.ChallengeOption {
	options := make([]model.ChallengeOption, len(points))

	for i, point := range points {
		options[i] = model.ChallengeOption{
			OptionID:  fmt.Sprintf("option_%d", i),
			Point:    point,
			IsCorrect: fmt.Sprintf("option_%d", i) == correctOption,
		}
	}

	return options
}

func (s *SpatioTemporalService) generateInstructions(difficulty string) string {
	switch difficulty {
	case "easy":
		return "请选择与您通常活动区域最接近的位置"
	case "medium":
		return "请根据您的时空行为模式选择正确的位置"
	case "hard":
		return "请精确识别与您历史行为模式匹配的位置"
	case "expert":
		return "请基于完整的行为轨迹分析选择正确答案"
	default:
		return "请选择正确的位置"
	}
}

func (s *SpatioTemporalService) calculateLocationMatchScore(req *model.SpatioTemporalVerifyRequest, session *model.SpatioTemporalSession) float64 {
	if req.UserLocation == nil {
		return 0.5
	}

	distance := s.calculateDistanceToCentroid(req, session)

	if distance < 0.1 {
		return 1.0
	} else if distance < 1.0 {
		return 0.9 - (distance * 0.1)
	} else if distance < 10.0 {
		return 0.8 - (distance / 100)
	}
	return 0.3
}

func (s *SpatioTemporalService) calculateTimePatternScore(req *model.SpatioTemporalVerifyRequest, session *model.SpatioTemporalSession) float64 {
	now := time.Now().Unix()
	window := session.TargetPattern.TimeWindow

	if now >= window.StartTime && now <= window.EndTime {
		return 0.8 + rand.Float64()*0.2
	}
	return 0.3 + rand.Float64()*0.4
}

func (s *SpatioTemporalService) calculateBehaviorMatchScore(req *model.SpatioTemporalVerifyRequest, session *model.SpatioTemporalSession) float64 {
	score := 0.5

	if req.ResponseTime > 1000 && req.ResponseTime < 30000 {
		score += 0.2
	}

	score += rand.Float64() * 0.3

	return math.Min(1.0, score)
}

func (s *SpatioTemporalService) calculateVelocityScore(req *model.SpatioTemporalVerifyRequest, session *model.SpatioTemporalSession) float64 {
	if req.UserLocation == nil {
		return 0.5
	}

	return 0.6 + rand.Float64()*0.4
}

func (s *SpatioTemporalService) calculateDistanceToCentroid(req *model.SpatioTemporalVerifyRequest, session *model.SpatioTemporalSession) float64 {
	if req.UserLocation == nil {
		return 1000.0
	}

	return haversineDistance(
		req.UserLocation.Latitude,
		req.UserLocation.Longitude,
		session.TargetPattern.Centroid[0],
		session.TargetPattern.Centroid[1],
	)
}

func (s *SpatioTemporalService) calculateTimeWindowMatch(req *model.SpatioTemporalVerifyRequest, session *model.SpatioTemporalSession) float64 {
	now := time.Now().Unix()
	window := session.TargetPattern.TimeWindow

	if now >= window.StartTime && now <= window.EndTime {
		return 1.0
	}

	var timeDiff int64
	if now < window.StartTime {
		timeDiff = window.StartTime - now
	} else {
		timeDiff = now - window.EndTime
	}

	if timeDiff < 3600 {
		return 0.8
	} else if timeDiff < 7200 {
		return 0.6
	}
	return 0.3
}

func (s *SpatioTemporalService) calculateAnomalyScore(session *model.SpatioTemporalSession) float64 {
	baseScore := session.TargetPattern.AnomalyScore
	return math.Min(1.0, baseScore+rand.Float64()*0.2)
}

func (s *SpatioTemporalService) generateAnalytics(session *model.SpatioTemporalSession) *model.SpatioTemporalAnalytics {
	riskLevel := "low"
	riskFactors := []string{}

	locationConfidence := 0.7 + rand.Float64()*0.3
	timeConsistency := 0.6 + rand.Float64()*0.4
	behaviorConsistency := 0.5 + rand.Float64()*0.5
	velocityConsistency := 0.6 + rand.Float64()*0.4

	overallRisk := (1-locationConfidence)*0.3 + (1-timeConsistency)*0.25 + (1-behaviorConsistency)*0.25 + (1-velocityConsistency)*0.2

	if overallRisk > 0.7 {
		riskLevel = "high"
		riskFactors = append(riskFactors, "location_anomaly", "time_inconsistency")
	} else if overallRisk > 0.4 {
		riskLevel = "medium"
	}

	return &model.SpatioTemporalAnalytics{
		LocationConfidence:  locationConfidence,
		TimeConsistency:    timeConsistency,
		BehaviorConsistency: behaviorConsistency,
		VelocityConsistency: velocityConsistency,
		RiskLevel:          riskLevel,
		RiskFactors:        riskFactors,
	}
}

func (s *SpatioTemporalService) getPointCountByDifficulty(difficulty string) int {
	switch difficulty {
	case "easy":
		return 5
	case "medium":
		return 8
	case "hard":
		return 12
	case "expert":
		return 15
	default:
		return 8
	}
}

func (s *SpatioTemporalService) GetRiskScore(sessionID string) (*model.RiskScore, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	score, exists := s.riskScores[sessionID]
	return score, exists
}

func (s *SpatioTemporalService) GetPrediction(sessionID string) (*model.BehaviorPrediction, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pred, exists := s.predictions[sessionID]
	return pred, exists
}

func calculateCentroidST(points []model.SpatioTemporalPoint) []float64 {
	if len(points) == 0 {
		return []float64{0, 0}
	}

	var totalLat, totalLng float64
	for _, point := range points {
		totalLat += point.Latitude
		totalLng += point.Longitude
	}

	return []float64{
		totalLat / float64(len(points)),
		totalLng / float64(len(points)),
	}
}

func calculateVelocityStatsSimple(points []model.SpatioTemporalPoint) (avg, max float64) {
	if len(points) < 2 {
		return 0, 0
	}

	var velocities []float64
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			dist := haversineDistance(
				points[i-1].Latitude, points[i-1].Longitude,
				points[i].Latitude, points[i].Longitude,
			)
			velocity := dist / dt * 3600
			velocities = append(velocities, velocity)
		}
	}

	if len(velocities) == 0 {
		return 0, 0
	}

	avg = velocities[0]
	max = velocities[0]
	for _, v := range velocities[1:] {
		avg += v
		if v > max {
			max = v
		}
	}
	avg /= float64(len(velocities))

	return avg, max
}

func generateSTSessionID() string {
	return fmt.Sprintf("st_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func SerializeSession(session *model.SpatioTemporalSession) ([]byte, error) {
	return json.Marshal(session)
}

func DeserializeSpatioTemporalSession(data []byte) (*model.SpatioTemporalSession, error) {
	var session model.SpatioTemporalSession
	err := json.Unmarshal(data, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
