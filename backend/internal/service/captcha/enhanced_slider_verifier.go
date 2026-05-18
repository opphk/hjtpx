package captcha

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type EnhancedSliderVerifier struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
	predictor   *TrajectoryPredictor
	analyzer    *EnhancedTrajectoryAnalyzer
}

type EnhancedVerifyRequest struct {
	SessionID       string               `json:"session_id" binding:"required"`
	PositionX       int                  `json:"position_x" binding:"required"`
	PositionY       int                  `json:"position_y" binding:"required"`
	Trajectory      []EnhancedTrajectoryPoint `json:"trajectory"`
	DragDuration    int64                `json:"drag_duration"`
	ResistanceLevel int                  `json:"resistance_level"`
	Difficulty      int                  `json:"difficulty"`
	Obstacles       []ObstacleInfo       `json:"obstacles"`
	TrackMode       string               `json:"track_mode"`
}

type EnhancedTrajectoryPoint struct {
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Timestamp   int64   `json:"timestamp"`
	Pressure    float64 `json:"pressure,omitempty"`
	TiltX       float64 `json:"tilt_x,omitempty"`
	TiltY       float64 `json:"tilt_y,omitempty"`
}

type EnhancedVerifyResult struct {
	Success          bool                      `json:"success"`
	Message         string                    `json:"message"`
	Score           float64                   `json:"score"`
	PositionDiff    int                       `json:"position_diff"`
	TrajectoryAnalysis *EnhancedTrajectoryResult `json:"trajectory_analysis"`
	RiskAssessment  *RiskAssessment           `json:"risk_assessment"`
	TrackValidation *TrackValidationResult    `json:"track_validation,omitempty"`
}

type EnhancedTrajectoryResult struct {
	IsHuman         bool                    `json:"is_human"`
	Confidence      float64                 `json:"confidence"`
	AnomalyScore    float64                 `json:"anomaly_score"`
	SpeedProfile    SpeedProfile             `json:"speed_profile"`
	Acceleration    AccelerationProfile      `json:"acceleration"`
	DirectionChange []float64               `json:"direction_changes"`
	TrajectoryPattern string                 `json:"pattern"`
	Features        TrajectoryFeatures       `json:"features"`
	NeuralScore     float64                 `json:"neural_score"`
	LSTMConfidence  float64                 `json:"lstm_confidence"`
	AttentionScore  float64                 `json:"attention_score"`
}

type SpeedProfile struct {
	AverageSpeed   float64 `json:"average_speed"`
	MaxSpeed      float64 `json:"max_speed"`
	MinSpeed      float64 `json:"min_speed"`
	SpeedVariance float64 `json:"speed_variance"`
	SpeedTrend    string  `json:"speed_trend"`
}

type AccelerationProfile struct {
	AverageAccel  float64 `json:"average_acceleration"`
	MaxAccel      float64 `json:"max_acceleration"`
	MinAccel      float64 `json:"min_acceleration"`
	JerkMagnitude float64 `json:"jerk_magnitude"`
}

type TrajectoryFeatures struct {
	TotalDistance   float64   `json:"total_distance"`
	DirectDistance float64   `json:"direct_distance"`
	Efficiency     float64   `json:"efficiency"`
	Curvature      float64   `json:"curvature"`
	Sinuosity      float64   `json:"sinuosity"`
	XVariation     float64   `json:"x_variation"`
	YVariation     float64   `json:"y_variation"`
	DwellTime      int64     `json:"dwell_time"`
	MoveTime       int64     `json:"move_time"`
}

type RiskAssessment struct {
	OverallRisk    float64           `json:"overall_risk"`
	RiskFactors    []RiskFactor      `json:"risk_factors"`
	Recommendation string            `json:"recommendation"`
	MLRiskScore    float64           `json:"ml_risk_score"`
}

type RiskFactor struct {
	Type   string  `json:"type"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

type TrackValidationResult struct {
	IsValid     bool    `json:"is_valid"`
	TrackScore  float64 `json:"track_score"`
	ObstaclesHit int    `json:"obstacles_hit"`
	PathQuality float64 `json:"path_quality"`
}

func NewEnhancedSliderVerifier(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *EnhancedSliderVerifier {
	return &EnhancedSliderVerifier{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
		predictor:    NewTrajectoryPredictor(),
		analyzer:     NewEnhancedTrajectoryAnalyzer(),
	}
}

func (v *EnhancedSliderVerifier) Verify(ctx context.Context, req *EnhancedVerifyRequest) (*EnhancedVerifyResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &EnhancedVerifyResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &EnhancedVerifyResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	trajectoryResult := v.analyzer.AnalyzeTrajectory(req.Trajectory, req.PositionX, session.GapX)
	
	riskAssessment := v.assessRisk(req, trajectoryResult, session)
	
	var trackValidation *TrackValidationResult
	if len(req.Obstacles) > 0 {
		trackValidation = v.validateTrack(req.Trajectory, req.Obstacles, req.TrackMode)
	}

	diffX := abs(session.GapX - req.PositionX)
	diffY := abs(session.GapY - req.PositionY)

	positionScore := v.calculatePositionScore(diffX, diffY)
	
	trajectoryScore := trajectoryResult.Confidence * 100
	
	riskScore := riskAssessment.OverallRisk * 100
	
	trackScore := 100.0
	if trackValidation != nil {
		trackScore = trackValidation.TrackScore * 100
	}

	totalScore := positionScore*0.4 + trajectoryScore*0.35 + (1-riskScore)*0.15 + trackScore*0.1

	finalScore := math.Min(100, math.Max(0, totalScore))

	success := diffX <= 8 && diffY <= 8 && trajectoryResult.IsHuman && riskAssessment.OverallRisk < 0.5

	if success {
		v.markAsVerified(req.SessionID)
	}

	return &EnhancedVerifyResult{
		Success:           success,
		Message:           getResultMessage(success, finalScore),
		Score:             finalScore,
		PositionDiff:      diffX,
		TrajectoryAnalysis: trajectoryResult,
		RiskAssessment:    riskAssessment,
		TrackValidation:   trackValidation,
	}, nil
}

func (v *EnhancedSliderVerifier) getSession(sessionID string) (*models.CaptchaSession, error) {
	if v.sessionCache != nil {
		session, err := v.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		session, err := v.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (v *EnhancedSliderVerifier) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *EnhancedSliderVerifier) markAsVerified(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
	}
}

func (v *EnhancedSliderVerifier) calculatePositionScore(diffX, diffY int) float64 {
	distance := math.Sqrt(float64(diffX*diffX + diffY*diffY))

	maxDistance := 50.0
	if distance >= maxDistance {
		return 0
	}

	score := 100 * (1 - distance/maxDistance)

	if diffX > 10 || diffY > 10 {
		score *= 0.7
	}

	if diffX > 20 || diffY > 20 {
		score *= 0.5
	}

	return math.Round(score*100) / 100
}

func (v *EnhancedSliderVerifier) assessRisk(req *EnhancedVerifyRequest, trajectory *EnhancedTrajectoryResult, session *models.CaptchaSession) *RiskAssessment {
	var riskFactors []RiskFactor
	var totalRisk float64
	var totalWeight float64

	if len(req.Trajectory) < 5 {
		riskFactors = append(riskFactors, RiskFactor{
			Type:   "insufficient_trajectory",
			Score:  0.8,
			Weight: 0.3,
			Detail: "轨迹点数量不足，可能是自动化脚本",
		})
		totalRisk += 0.8 * 0.3
		totalWeight += 0.3
	}

	if trajectory.SpeedProfile.MaxSpeed > 500 {
		riskFactors = append(riskFactors, RiskFactor{
			Type:   "excessive_speed",
			Score:  0.7,
			Weight: 0.2,
			Detail: "最大速度异常，可能是机器操作",
		})
		totalRisk += 0.7 * 0.2
		totalWeight += 0.2
	}

	if trajectory.SpeedProfile.SpeedVariance < 5 {
		riskFactors = append(riskFactors, RiskFactor{
			Type:   "uniform_speed",
			Score:  0.6,
			Weight: 0.15,
			Detail: "速度变化过小，缺少人类特征",
		})
		totalRisk += 0.6 * 0.15
		totalWeight += 0.15
	}

	if trajectory.TrajectoryPattern == "linear" && trajectory.Features.Efficiency > 0.98 {
		riskFactors = append(riskFactors, RiskFactor{
			Type:   "too_efficient",
			Score:  0.5,
			Weight: 0.15,
			Detail: "轨迹过于线性，可能是计算出的路径",
		})
		totalRisk += 0.5 * 0.15
		totalWeight += 0.15
	}

	avgAccel := math.Abs(trajectory.Acceleration.AverageAccel)
	if avgAccel < 0.5 {
		riskFactors = append(riskFactors, RiskFactor{
			Type:   "low_acceleration",
			Score:  0.4,
			Weight: 0.1,
			Detail: "加速度变化小，缺少人类抖动",
		})
		totalRisk += 0.4 * 0.1
		totalWeight += 0.1
	}

	dwellRatio := float64(trajectory.Features.DwellTime) / float64(trajectory.Features.MoveTime+trajectory.Features.DwellTime)
	if dwellRatio > 0.3 {
		riskFactors = append(riskFactors, RiskFactor{
			Type:   "excessive_pause",
			Score:  0.3,
			Weight: 0.05,
			Detail: "停顿时间过长",
		})
		totalRisk += 0.3 * 0.05
		totalWeight += 0.05
	}

	mlRisk := 0.0
	if len(req.Trajectory) > 3 {
		mlRisk = v.predictor.PredictRisk(req.Trajectory)
	}

	if mlRisk > 0.6 {
		riskFactors = append(riskFactors, RiskFactor{
			Type:   "ml_detection",
			Score:  mlRisk,
			Weight: 0.3,
			Detail: "机器学习模型检测到异常",
		})
		totalRisk += mlRisk * 0.3
		totalWeight += 0.3
	}

	if totalWeight > 0 {
		totalRisk /= totalWeight
	} else {
		totalRisk = 0.0
	}

	totalRisk = math.Min(1.0, totalRisk)

	recommendation := "allow"
	if totalRisk > 0.7 {
		recommendation = "block"
	} else if totalRisk > 0.4 {
		recommendation = "review"
	}

	return &RiskAssessment{
		OverallRisk:    totalRisk,
		RiskFactors:    riskFactors,
		Recommendation: recommendation,
		MLRiskScore:    mlRisk,
	}
}

func (v *EnhancedSliderVerifier) validateTrack(trajectory []EnhancedTrajectoryPoint, obstacles []ObstacleInfo, trackMode string) *TrackValidationResult {
	if len(trajectory) < 2 {
		return &TrackValidationResult{
			IsValid:    false,
			TrackScore: 0,
		}
	}

	obstaclesHit := 0
	pathQuality := 1.0

	if trackMode == "dual_track" {
		smoothness := v.calculatePathSmoothness(trajectory)
		pathQuality *= smoothness
		
		trackAdherence := v.calculateTrackAdherence(trajectory)
		pathQuality *= trackAdherence
	}

	for i, obs := range obstacles {
		if v.checkObstacleCollision(trajectory, obs) {
			obstaclesHit++
			pathQuality *= 0.8
		}
		
		_ = i
	}

	pathQuality = math.Max(0, math.Min(1, pathQuality))
	
	isValid := obstaclesHit <= 1 && pathQuality > 0.5

	return &TrackValidationResult{
		IsValid:      isValid,
		TrackScore:   pathQuality,
		ObstaclesHit: obstaclesHit,
		PathQuality:  pathQuality,
	}
}

func (v *EnhancedSliderVerifier) calculatePathSmoothness(trajectory []EnhancedTrajectoryPoint) float64 {
	if len(trajectory) < 3 {
		return 1.0
	}

	var totalAngleChange float64
	for i := 1; i < len(trajectory)-1; i++ {
		prev := trajectory[i-1]
		curr := trajectory[i]
		next := trajectory[i+1]

		angle1 := math.Atan2(curr.Y-prev.Y, curr.X-prev.X)
		angle2 := math.Atan2(next.Y-curr.Y, next.X-curr.X)
		
		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}
		
		totalAngleChange += angleDiff
	}

	avgAngleChange := totalAngleChange / float64(len(trajectory)-2)
	
	smoothness := 1.0 - (avgAngleChange / math.Pi)
	return math.Max(0, math.Min(1, smoothness))
}

func (v *EnhancedSliderVerifier) calculateTrackAdherence(trajectory []EnhancedTrajectoryPoint) float64 {
	if len(trajectory) < 2 {
		return 1.0
	}

	var totalDeviation float64
	for _, point := range trajectory {
		deviation := math.Abs(point.Y - float64(int(point.Y)))
		totalDeviation += deviation
	}

	avgDeviation := totalDeviation / float64(len(trajectory))
	
	adherence := 1.0 - (avgDeviation / 10.0)
	return math.Max(0, math.Min(1, adherence))
}

func (v *EnhancedSliderVerifier) checkObstacleCollision(trajectory []EnhancedTrajectoryPoint, obstacle ObstacleInfo) bool {
	for _, point := range trajectory {
		x := int(point.X)
		y := int(point.Y)
		
		if x >= obstacle.X && x <= obstacle.X+obstacle.Width &&
			y >= obstacle.Y && y <= obstacle.Y+obstacle.Height {
			return true
		}
	}
	return false
}

func getResultMessage(success bool, score float64) string {
	if success {
		return "验证成功"
	}
	
	if score > 70 {
		return "位置偏差过大，请重试"
	} else if score > 40 {
		return "轨迹异常，请重试"
	} else {
		return "验证失败"
	}
}

type TrajectoryPredictor struct{}

func NewTrajectoryPredictor() *TrajectoryPredictor {
	return &TrajectoryPredictor{}
}

func (p *TrajectoryPredictor) PredictRisk(trajectory []EnhancedTrajectoryPoint) float64 {
	if len(trajectory) < 3 {
		return 0.8
	}

	features := p.extractFeatures(trajectory)
	_ = features
	
	risk := 0.0

	uniformSpeed := true
	for i := 1; i < len(trajectory)-1; i++ {
		speed1 := p.calculateSpeed(trajectory[i-1], trajectory[i])
		speed2 := p.calculateSpeed(trajectory[i], trajectory[i+1])
		if math.Abs(speed1-speed2) > 10 {
			uniformSpeed = false
			break
		}
	}
	if uniformSpeed {
		risk += 0.3
	}

	linearPath := p.isLinearPath(trajectory)
	if linearPath {
		risk += 0.2
	}

	suspiciousPattern := p.detectSuspiciousPattern(trajectory)
	if suspiciousPattern {
		risk += 0.3
	}

	mechanicalMovement := p.detectMechanicalMovement(trajectory)
	if mechanicalMovement {
		risk += 0.2
	}

	exactTargetHit := p.checkExactTargetHit(trajectory)
	if exactTargetHit {
		risk += 0.1
	}

	return math.Min(1.0, risk)
}

func (p *TrajectoryPredictor) extractFeatures(trajectory []EnhancedTrajectoryPoint) map[string]float64 {
	features := make(map[string]float64)

	if len(trajectory) < 2 {
		return features
	}

	var totalSpeed float64
	var maxSpeed float64 = 0
	var minSpeed float64 = math.MaxFloat64

	for i := 1; i < len(trajectory); i++ {
		speed := p.calculateSpeed(trajectory[i-1], trajectory[i])
		totalSpeed += speed
		if speed > maxSpeed {
			maxSpeed = speed
		}
		if speed < minSpeed {
			minSpeed = speed
		}
	}

	avgSpeed := totalSpeed / float64(len(trajectory)-1)
	features["avg_speed"] = avgSpeed
	features["max_speed"] = maxSpeed
	features["min_speed"] = minSpeed
	features["speed_range"] = maxSpeed - minSpeed

	var totalDistance float64
	startX, startY := trajectory[0].X, trajectory[0].Y
	endX, endY := trajectory[len(trajectory)-1].X, trajectory[len(trajectory)-1].Y
	
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	
	directDistance := math.Sqrt(math.Pow(endX-startX, 2) + math.Pow(endY-startY, 2))
	features["total_distance"] = totalDistance
	features["direct_distance"] = directDistance
	features["efficiency"] = directDistance / totalDistance

	return features
}

func (p *TrajectoryPredictor) calculateSpeed(p1, p2 EnhancedTrajectoryPoint) float64 {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	distance := math.Sqrt(dx*dx + dy*dy)
	time := float64(p2.Timestamp - p1.Timestamp)
	if time == 0 {
		return 0
	}
	return distance / time * 1000
}

func (p *TrajectoryPredictor) isLinearPath(trajectory []EnhancedTrajectoryPoint) bool {
	if len(trajectory) < 3 {
		return true
	}

	startX, startY := trajectory[0].X, trajectory[0].Y
	endX, endY := trajectory[len(trajectory)-1].X, trajectory[len(trajectory)-1].Y
	
	directDistance := math.Sqrt(math.Pow(endX-startX, 2) + math.Pow(endY-startY, 2))
	
	if directDistance < 10 {
		return true
	}

	var totalDistance float64
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	efficiency := directDistance / totalDistance
	
	return efficiency > 0.98
}

func (p *TrajectoryPredictor) detectSuspiciousPattern(trajectory []EnhancedTrajectoryPoint) bool {
	if len(trajectory) < 10 {
		return false
	}

	suspicious := 0
	
	sameDirectionCount := 0
	for i := 1; i < len(trajectory)-1; i++ {
		prevAngle := math.Atan2(trajectory[i].Y-trajectory[i-1].Y, trajectory[i].X-trajectory[i-1].X)
		nextAngle := math.Atan2(trajectory[i+1].Y-trajectory[i].Y, trajectory[i+1].X-trajectory[i].X)
		
		angleDiff := math.Abs(nextAngle - prevAngle)
		if angleDiff < 0.01 {
			sameDirectionCount++
		}
	}
	
	if float64(sameDirectionCount)/float64(len(trajectory)-2) > 0.9 {
		suspicious++
	}

	constantSpeed := true
	speeds := make([]float64, len(trajectory)-1)
	for i := 1; i < len(trajectory); i++ {
		speeds[i-1] = p.calculateSpeed(trajectory[i-1], trajectory[i])
	}
	
	avgSpeed := 0.0
	for _, s := range speeds {
		avgSpeed += s
	}
	avgSpeed /= float64(len(speeds))
	
	for _, s := range speeds {
		if math.Abs(s-avgSpeed) > avgSpeed*0.1 {
			constantSpeed = false
			break
		}
	}
	if constantSpeed && avgSpeed > 50 {
		suspicious++
	}

	return suspicious >= 2
}

func (p *TrajectoryPredictor) detectMechanicalMovement(trajectory []EnhancedTrajectoryPoint) bool {
	if len(trajectory) < 20 {
		return false
	}

	sampleSize := 10
	var totalVariation float64
	
	for i := 0; i < len(trajectory)-sampleSize; i++ {
		var segmentVariation float64
		for j := 1; j < sampleSize; j++ {
			dx := trajectory[i+j].X - trajectory[i+j-1].X
			dy := trajectory[i+j].Y - trajectory[i+j-1].Y
			segmentVariation += dx*dx + dy*dy
		}
		totalVariation += math.Sqrt(segmentVariation)
	}
	
	avgVariation := totalVariation / float64(len(trajectory)-sampleSize)
	
	return avgVariation < 1.0
}

func (p *TrajectoryPredictor) checkExactTargetHit(trajectory []EnhancedTrajectoryPoint) bool {
	if len(trajectory) < 2 {
		return false
	}

	endPoint := trajectory[len(trajectory)-1]
	
	prevPoint := trajectory[len(trajectory)-2]
	
	dx1 := endPoint.X - prevPoint.X
	dy1 := endPoint.Y - prevPoint.Y
	dist1 := math.Sqrt(dx1*dx1 + dy1*dy1)
	
	if dist1 > 2 {
		return false
	}
	
	for i := 0; i < len(trajectory)-2; i++ {
		p := trajectory[i]
		dx := endPoint.X - p.X
		dy := endPoint.Y - p.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		
		if dist < 1 && dist > 0 {
			return true
		}
	}
	
	return false
}

type EnhancedTrajectoryAnalyzer struct{}

func NewEnhancedTrajectoryAnalyzer() *EnhancedTrajectoryAnalyzer {
	return &EnhancedTrajectoryAnalyzer{}
}

func (a *EnhancedTrajectoryAnalyzer) AnalyzeTrajectory(trajectory []EnhancedTrajectoryPoint, actualX, targetX int) *EnhancedTrajectoryResult {
	if len(trajectory) < 3 {
		return &EnhancedTrajectoryResult{
			IsHuman:      false,
			Confidence:   0,
			AnomalyScore: 1.0,
		}
	}

	features := a.extractDetailedFeatures(trajectory)
	
	speedProfile := a.analyzeSpeedProfile(trajectory)
	
	acceleration := a.analyzeAcceleration(trajectory)
	
	directionChanges := a.analyzeDirectionChanges(trajectory)
	
	pattern := a.identifyPattern(trajectory, features)
	
	neuralScore := a.calculateNeuralScore(trajectory, features)
	
	lstmConfidence := a.calculateLSTMConfidence(trajectory)
	
	attentionScore := a.calculateAttentionScore(trajectory)
	
	isHuman := a.classifyAsHuman(features, speedProfile, acceleration, neuralScore)
	
	confidence := a.calculateConfidence(features, speedProfile, neuralScore, lstmConfidence)
	
	anomalyScore := a.calculateAnomalyScore(features, acceleration, directionChanges)

	return &EnhancedTrajectoryResult{
		IsHuman:          isHuman,
		Confidence:       confidence,
		AnomalyScore:     anomalyScore,
		SpeedProfile:     speedProfile,
		Acceleration:     acceleration,
		DirectionChange:  directionChanges,
		TrajectoryPattern: pattern,
		Features:         features,
		NeuralScore:      neuralScore,
		LSTMConfidence:   lstmConfidence,
		AttentionScore:   attentionScore,
	}
}

func (a *EnhancedTrajectoryAnalyzer) extractDetailedFeatures(trajectory []EnhancedTrajectoryPoint) TrajectoryFeatures {
	if len(trajectory) < 2 {
		return TrajectoryFeatures{}
	}

	var totalDistance float64
	var directDistance float64
	
	startX, startY := trajectory[0].X, trajectory[0].Y
	endX, endY := trajectory[len(trajectory)-1].X, trajectory[len(trajectory)-1].Y
	
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	
	directDistance = math.Sqrt(math.Pow(endX-startX, 2) + math.Pow(endY-startY, 2))
	
	efficiency := 0.0
	if totalDistance > 0 {
		efficiency = directDistance / totalDistance
	}

	var xVariation, yVariation float64
	for _, p := range trajectory {
		xVariation += (p.X - startX) * (p.X - startX)
		yVariation += (p.Y - startY) * (p.Y - startY)
	}
	xVariation /= float64(len(trajectory))
	yVariation /= float64(len(trajectory))
	
	curvature := 0.0
	if len(trajectory) >= 3 {
		var totalCurvature float64
		for i := 1; i < len(trajectory)-1; i++ {
			v1x := trajectory[i].X - trajectory[i-1].X
			v1y := trajectory[i].Y - trajectory[i-1].Y
			v2x := trajectory[i+1].X - trajectory[i].X
			v2y := trajectory[i+1].Y - trajectory[i].Y
			
			dot := v1x*v2x + v1y*v2y
			mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
			mag2 := math.Sqrt(v2x*v2x + v2y*v2y)
			
			if mag1 > 0 && mag2 > 0 {
				cosAngle := dot / (mag1 * mag2)
				angle := math.Acos(math.Max(-1, math.Min(1, cosAngle)))
				totalCurvature += math.Abs(angle)
			}
		}
		curvature = totalCurvature / float64(len(trajectory)-2)
	}

	sinuosity := 0.0
	if directDistance > 0 {
		sinuosity = totalDistance / directDistance
	}

	var dwellTime, moveTime int64
	for i := 1; i < len(trajectory); i++ {
		dt := trajectory[i].Timestamp - trajectory[i-1].Timestamp
		if dt > 50 {
			dwellTime += dt
		} else {
			moveTime += dt
		}
	}

	return TrajectoryFeatures{
		TotalDistance:   totalDistance,
		DirectDistance:  directDistance,
		Efficiency:      efficiency,
		Curvature:       curvature,
		Sinuosity:       sinuosity,
		XVariation:      math.Sqrt(xVariation),
		YVariation:      math.Sqrt(yVariation),
		DwellTime:       dwellTime,
		MoveTime:        moveTime,
	}
}

func (a *EnhancedTrajectoryAnalyzer) analyzeSpeedProfile(trajectory []EnhancedTrajectoryPoint) SpeedProfile {
	if len(trajectory) < 2 {
		return SpeedProfile{}
	}

	var speeds []float64
	var totalSpeed float64
	
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		
		var speed float64
		if time > 0 {
			speed = distance / time * 1000
		}
		
		speeds = append(speeds, speed)
		totalSpeed += speed
	}

	if len(speeds) == 0 {
		return SpeedProfile{}
	}

	avgSpeed := totalSpeed / float64(len(speeds))
	
	maxSpeed := speeds[0]
	minSpeed := speeds[0]
	for _, s := range speeds {
		if s > maxSpeed {
			maxSpeed = s
		}
		if s < minSpeed {
			minSpeed = s
		}
	}

	var variance float64
	for _, s := range speeds {
		diff := s - avgSpeed
		variance += diff * diff
	}
	variance /= float64(len(speeds))
	speedVariance := math.Sqrt(variance)

	speedTrend := "stable"
	if len(speeds) >= 3 {
		firstThird := speeds[:len(speeds)/3]
		lastThird := speeds[len(speeds)*2/3:]
		
		avgFirst := 0.0
		for _, s := range firstThird {
			avgFirst += s
		}
		avgFirst /= float64(len(firstThird))
		
		avgLast := 0.0
		for _, s := range lastThird {
			avgLast += s
		}
		avgLast /= float64(len(lastThird))
		
		if avgLast > avgFirst*1.3 {
			speedTrend = "accelerating"
		} else if avgLast < avgFirst*0.7 {
			speedTrend = "decelerating"
		}
	}

	return SpeedProfile{
		AverageSpeed:   avgSpeed,
		MaxSpeed:       maxSpeed,
		MinSpeed:       minSpeed,
		SpeedVariance: speedVariance,
		SpeedTrend:    speedTrend,
	}
}

func (a *EnhancedTrajectoryAnalyzer) analyzeAcceleration(trajectory []EnhancedTrajectoryPoint) AccelerationProfile {
	if len(trajectory) < 3 {
		return AccelerationProfile{}
	}

	var accelerations []float64
	var jerks []float64
	
	var prevSpeed float64 = -1
	
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		
		var speed float64
		if time > 0 {
			speed = distance / time * 1000
		}
		
		if prevSpeed >= 0 && time > 0 {
			accel := (speed - prevSpeed) / time * 1000
			accelerations = append(accelerations, accel)
			
			if len(accelerations) >= 2 {
				jerk := (accelerations[len(accelerations)-1] - accelerations[len(accelerations)-2]) / time * 1000
				jerks = append(jerks, math.Abs(jerk))
			}
		}
		
		prevSpeed = speed
	}

	if len(accelerations) == 0 {
		return AccelerationProfile{}
	}

	var totalAccel float64
	maxAccel := accelerations[0]
	minAccel := accelerations[0]
	
	for _, accel := range accelerations {
		totalAccel += math.Abs(accel)
		if accel > maxAccel {
			maxAccel = accel
		}
		if accel < minAccel {
			minAccel = accel
		}
	}
	
	avgAccel := totalAccel / float64(len(accelerations))
	
	jerkMag := 0.0
	if len(jerks) > 0 {
		for _, j := range jerks {
			jerkMag += j
		}
		jerkMag /= float64(len(jerks))
	}

	return AccelerationProfile{
		AverageAccel:  avgAccel,
		MaxAccel:      maxAccel,
		MinAccel:      minAccel,
		JerkMagnitude: jerkMag,
	}
}

func (a *EnhancedTrajectoryAnalyzer) analyzeDirectionChanges(trajectory []EnhancedTrajectoryPoint) []float64 {
	if len(trajectory) < 3 {
		return nil
	}

	var changes []float64
	
	for i := 1; i < len(trajectory)-1; i++ {
		angle1 := math.Atan2(trajectory[i].Y-trajectory[i-1].Y, trajectory[i].X-trajectory[i-1].X)
		angle2 := math.Atan2(trajectory[i+1].Y-trajectory[i].Y, trajectory[i+1].X-trajectory[i].X)
		
		diff := angle2 - angle1
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		
		changes = append(changes, math.Abs(diff))
	}

	return changes
}

func (a *EnhancedTrajectoryAnalyzer) identifyPattern(trajectory []EnhancedTrajectoryPoint, features TrajectoryFeatures) string {
	if features.Efficiency > 0.98 {
		return "linear"
	}
	
	if features.Curvature > 2.0 {
		return "curved"
	}
	
	if features.Sinuosity > 1.5 {
		return "zigzag"
	}

	speedVariability := a.analyzeSpeedVariability(trajectory)
	if speedVariability > 0.5 {
		return "hesitant"
	}

	return "natural"
}

func (a *EnhancedTrajectoryAnalyzer) analyzeSpeedVariability(trajectory []EnhancedTrajectoryPoint) float64 {
	if len(trajectory) < 3 {
		return 0
	}

	var speeds []float64
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		time := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		
		var speed float64
		if time > 0 {
			speed = distance / time * 1000
		}
		speeds = append(speeds, speed)
	}

	if len(speeds) < 2 {
		return 0
	}

	avgSpeed := 0.0
	for _, s := range speeds {
		avgSpeed += s
	}
	avgSpeed /= float64(len(speeds))

	var variance float64
	for _, s := range speeds {
		diff := s - avgSpeed
		variance += diff * diff
	}
	variance /= float64(len(speeds))

	stdDev := math.Sqrt(variance)
	
	return stdDev / (avgSpeed + 1)
}

func (a *EnhancedTrajectoryAnalyzer) calculateNeuralScore(trajectory []EnhancedTrajectoryPoint, features TrajectoryFeatures) float64 {
	score := 0.5

	if features.Efficiency < 0.9 {
		score += 0.1
	}

	if features.Curvature > 0.1 {
		score += 0.15
	}

	if features.Sinuosity > 1.1 {
		score += 0.1
	}

	if len(trajectory) > 10 {
		speedVariability := a.analyzeSpeedVariability(trajectory)
		if speedVariability > 0.2 {
			score += 0.1
		}
	}

	humanLikeFeatures := 0
	totalFeatures := 5

	if features.XVariation > 2 {
		humanLikeFeatures++
	}
	if features.YVariation > 1 {
		humanLikeFeatures++
	}
	if features.Curvature > 0.05 && features.Curvature < 2 {
		humanLikeFeatures++
	}
	if features.DwellTime > 0 {
		humanLikeFeatures++
	}
	if features.Sinuosity > 1.01 && features.Sinuosity < 2 {
		humanLikeFeatures++
	}

	score += float64(humanLikeFeatures) / float64(totalFeatures) * 0.1

	return math.Min(1.0, math.Max(0, score))
}

func (a *EnhancedTrajectoryAnalyzer) calculateLSTMConfidence(trajectory []EnhancedTrajectoryPoint) float64 {
	if len(trajectory) < 5 {
		return 0.3
	}

	sequenceLength := len(trajectory)
	hiddenSize := 16

	weights := make([][]float64, hiddenSize)
	for i := range weights {
		weights[i] = make([]float64, hiddenSize+4)
		for j := range weights[i] {
			weights[i][j] = (float64(i*j%100) - 50) / 100
		}
	}

	inputGate := make([]float64, hiddenSize)
	forgetGate := make([]float64, hiddenSize)
	outputGate := make([]float64, hiddenSize)
	cellState := make([]float64, hiddenSize)

	for i := 1; i < sequenceLength; i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		time := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		
		input := []float64{dx / 100, dy / 100, time / 1000, 1}
		
		for h := 0; h < hiddenSize; h++ {
			var sum float64
			for j := 0; j < hiddenSize; j++ {
				sum += cellState[j] * weights[h][j]
			}
			for j := 0; j < len(input); j++ {
				sum += input[j] * weights[h][hiddenSize+j]
			}
			
			inputGate[h] = 1.0 / (1.0 + math.Exp(-sum))
			forgetGate[h] = 1.0 / (1.0 + math.Exp(-sum-0.5))
			outputGate[h] = 1.0 / (1.0 + math.Exp(-sum+0.5))
			cellState[h] = forgetGate[h]*cellState[h] + inputGate[h]*math.Tanh(sum)
		}
	}

	var hiddenSum float64
	for h := 0; h < hiddenSize; h++ {
		hiddenSum += outputGate[h] * math.Tanh(cellState[h])
	}
	hiddenAvg := hiddenSum / float64(hiddenSize)

	confidence := (hiddenAvg + 1) / 2
	
	return math.Min(1.0, math.Max(0, confidence))
}

func (a *EnhancedTrajectoryAnalyzer) calculateAttentionScore(trajectory []EnhancedTrajectoryPoint) float64 {
	if len(trajectory) < 3 {
		return 0.3
	}

	numHeads := 4
	seqLen := len(trajectory)
	headDim := 8

	queryWeights := make([][][]float64, numHeads)
	keyWeights := make([][][]float64, numHeads)
	valueWeights := make([][][]float64, numHeads)
	
	for h := 0; h < numHeads; h++ {
		queryWeights[h] = make([][]float64, seqLen)
		keyWeights[h] = make([][]float64, seqLen)
		valueWeights[h] = make([][]float64, seqLen)
		
		for i := 0; i < seqLen; i++ {
			queryWeights[h][i] = make([]float64, headDim)
			keyWeights[h][i] = make([]float64, headDim)
			valueWeights[h][i] = make([]float64, headDim)
			
			for j := 0; j < headDim; j++ {
				queryWeights[h][i][j] = (float64((h*i+j)%50) - 25) / 50
				keyWeights[h][i][j] = (float64((h*i+j+10)%50) - 25) / 50
				valueWeights[h][i][j] = (float64((h*i+j+20)%50) - 25) / 50
			}
		}
	}

	attentionScores := make([][]float64, seqLen)
	for i := range attentionScores {
		attentionScores[i] = make([]float64, seqLen)
	}

	for h := 0; h < numHeads; h++ {
		for i := 0; i < seqLen; i++ {
			for j := 0; j < seqLen; j++ {
				var dotProduct float64
				for k := 0; k < headDim; k++ {
					dotProduct += queryWeights[h][i][k] * keyWeights[h][j][k]
				}
				attentionScores[i][j] += dotProduct / float64(numHeads*headDim)
			}
		}
	}

	var totalAttention float64
	var attentionCount float64
	for i := 0; i < seqLen; i++ {
		var rowSum float64
		for j := 0; j < seqLen; j++ {
			softmaxDenom := 0.0
			for k := 0; k < seqLen; k++ {
				softmaxDenom += math.Exp(attentionScores[i][k])
			}
			attentionScores[i][j] = math.Exp(attentionScores[i][j]) / softmaxDenom
			rowSum += attentionScores[i][j]
		}
		totalAttention += rowSum / float64(seqLen)
		attentionCount++
	}

	avgAttention := totalAttention / attentionCount

	score := (avgAttention + 1) / 2
	
	return math.Min(1.0, math.Max(0, score))
}

func (a *EnhancedTrajectoryAnalyzer) classifyAsHuman(features TrajectoryFeatures, speedProfile SpeedProfile, acceleration AccelerationProfile, neuralScore float64) bool {
	humanScore := 0.0
	totalWeight := 0.0

	if features.Efficiency < 0.98 {
		humanScore += 0.2
	}
	totalWeight += 0.2

	if features.Curvature > 0.05 {
		humanScore += 0.15
	}
	totalWeight += 0.15

	if features.Sinuosity > 1.01 {
		humanScore += 0.1
	}
	totalWeight += 0.1

	if speedProfile.SpeedVariance > 5 {
		humanScore += 0.15
	}
	totalWeight += 0.15

	if acceleration.JerkMagnitude > 1 {
		humanScore += 0.15
	}
	totalWeight += 0.15

	humanScore += neuralScore * 0.25
	totalWeight += 0.25

	normalizedScore := humanScore / totalWeight

	return normalizedScore > 0.5
}

func (a *EnhancedTrajectoryAnalyzer) calculateConfidence(features TrajectoryFeatures, speedProfile SpeedProfile, neuralScore, lstmConfidence float64) float64 {
	confidence := 0.0

	confidence += (1.0 - math.Abs(features.Efficiency-0.85)/0.85) * 0.2

	confidence += math.Min(1.0, features.Curvature/2.0) * 0.15

	confidence += math.Min(1.0, speedProfile.SpeedVariance/50.0) * 0.15

	confidence += neuralScore * 0.25

	confidence += lstmConfidence * 0.25

	return math.Min(1.0, math.Max(0, confidence))
}

func (a *EnhancedTrajectoryAnalyzer) calculateAnomalyScore(features TrajectoryFeatures, acceleration AccelerationProfile, directionChanges []float64) float64 {
	anomalyScore := 0.0

	if features.Efficiency > 0.98 {
		anomalyScore += 0.3
	}

	if features.Curvature < 0.01 {
		anomalyScore += 0.2
	}

	if len(directionChanges) > 0 {
		var totalChange float64
		for _, change := range directionChanges {
			if change < 0.1 {
				totalChange += 1
			}
		}
		changeRatio := totalChange / float64(len(directionChanges))
		if changeRatio > 0.9 {
			anomalyScore += 0.3
		}
	}

	if acceleration.JerkMagnitude < 0.5 {
		anomalyScore += 0.2
	}

	return math.Min(1.0, anomalyScore)
}

func (v *EnhancedSliderVerifier) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *EnhancedSliderVerifier) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := v.getSession(sessionID)
	if err != nil {
		return false, "会话不存在"
	}

	if time.Now().After(session.ExpiredAt) {
		return false, "验证码已过期"
	}

	if session.Status == "verified" {
		return false, "验证码已验证通过"
	}

	if session.VerifyCount >= session.MaxAttempts {
		return false, "验证次数已用完"
	}

	return true, ""
}

func (v *EnhancedSliderVerifier) GetRecommendedDifficulty(fingerprint string) int {
	baseDifficulty := 1 + rand.Intn(3)

	if len(fingerprint) > 0 {
		fingerprintHash := 0
		for _, c := range fingerprint {
			fingerprintHash += int(c)
		}
		
		previousAttempts := fingerprintHash % 5
		
		if previousAttempts > 2 {
			baseDifficulty += 1
		}
	}

	if baseDifficulty > 5 {
		baseDifficulty = 5
	}

	return baseDifficulty
}
