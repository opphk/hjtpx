package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// TimePatternType 时间模式类型
type TimePatternType string

const (
	TimePatternDaily   TimePatternType = "daily"
	TimePatternWeekly  TimePatternType = "weekly"
	TimePatternMonthly TimePatternType = "monthly"
	TimePatternCustom  TimePatternType = "custom"
)

// LocationAccuracy 位置精度
type LocationAccuracy string

const (
	LocationAccuracyCity    LocationAccuracy = "city"
	LocationAccuracyRegion  LocationAccuracy = "region"
	LocationAccuracyCountry LocationAccuracy = "country"
	LocationAccuracyIP      LocationAccuracy = "ip"
)

// SpatioTemporalPoint 时空点
type SpatioTemporalPoint struct {
	Timestamp   int64            `json:"timestamp"`
	Latitude    float64          `json:"latitude"`
	Longitude   float64          `json:"longitude"`
	IPAddress   string           `json:"ip_address,omitempty"`
	UserAgent   string           `json:"user_agent,omitempty"`
	DeviceID    string           `json:"device_id,omitempty"`
	Accuracy    LocationAccuracy `json:"accuracy"`
	Confidence  float64          `json:"confidence"`
}

// SpatioTemporalPattern 时空模式
type SpatioTemporalPattern struct {
	PatternID      string                 `json:"pattern_id"`
	PatternType    TimePatternType        `json:"pattern_type"`
	Points         []SpatioTemporalPoint  `json:"points"`
	Centroid       []float64              `json:"centroid"`
	TimeWindow     TimeWindow             `json:"time_window"`
	BehaviorFeatures map[string]float64    `json:"behavior_features"`
	AnomalyScore   float64                `json:"anomaly_score"`
}

// TimeWindow 时间窗口
type TimeWindow struct {
	StartTime  int64 `json:"start_time"`
	EndTime    int64 `json:"end_time"`
	Duration   int64 `json:"duration"`
}

// SpatioTemporalCaptchaRequest 时空验证请求
type SpatioTemporalCaptchaRequest struct {
	UserID        string                 `json:"user_id"`
	PatternType   TimePatternType        `json:"pattern_type"`
	Difficulty    string                 `json:"difficulty"`
	ClientIP      string                 `json:"client_ip"`
	UserAgent     string                 `json:"user_agent"`
	CurrentLocation *SpatioTemporalPoint `json:"current_location,omitempty"`
}

// SpatioTemporalCaptchaResponse 时空验证响应
type SpatioTemporalCaptchaResponse struct {
	SessionID       string                 `json:"session_id"`
	TargetPattern   *SpatioTemporalPattern `json:"target_pattern"`
	ChallengePoints []SpatioTemporalPoint  `json:"challenge_points"`
	Instructions    string                 `json:"instructions"`
	Options         []ChallengeOption      `json:"options"`
	ExpiresIn       int64                  `json:"expires_in"`
	ExpiresAt       int64                  `json:"expires_at"`
}

// ChallengeOption 挑战选项
type ChallengeOption struct {
	OptionID  string               `json:"option_id"`
	Point     SpatioTemporalPoint  `json:"point"`
	IsCorrect bool                 `json:"is_correct"`
}

// SpatioTemporalVerifyRequest 时空验证请求
type SpatioTemporalVerifyRequest struct {
	SessionID     string                 `json:"session_id"`
	SelectedOption string                 `json:"selected_option"`
	UserLocation   *SpatioTemporalPoint  `json:"user_location"`
	ResponseTime   int64                  `json:"response_time"`
	BehaviorData   map[string]interface{} `json:"behavior_data,omitempty"`
}

// SpatioTemporalVerifyResponse 时空验证响应
type SpatioTemporalVerifyResponse struct {
	Success     bool                      `json:"success"`
	Score       float64                   `json:"score"`
	Message     string                    `json:"message"`
	Details     *SpatioTemporalVerifyDetails `json:"details,omitempty"`
	Analytics   *SpatioTemporalAnalytics `json:"analytics,omitempty"`
}

// SpatioTemporalVerifyDetails 验证详情
type SpatioTemporalVerifyDetails struct {
	LocationMatchScore  float64            `json:"location_match_score"`
	TimePatternScore    float64            `json:"time_pattern_score"`
	BehaviorMatchScore  float64            `json:"behavior_match_score"`
	DistanceToCentroid  float64            `json:"distance_to_centroid"`
	TimeWindowMatch     float64            `json:"time_window_match"`
	AnomalyScore        float64            `json:"anomaly_score"`
}

// SpatioTemporalAnalytics 时空分析
type SpatioTemporalAnalytics struct {
	LocationConfidence   float64            `json:"location_confidence"`
	TimeConsistency      float64            `json:"time_consistency"`
	BehaviorConsistency  float64            `json:"behavior_consistency"`
	RiskLevel            string             `json:"risk_level"`
	RiskFactors          []string           `json:"risk_factors"`
}

// SpatioTemporalSession 时空会话
type SpatioTemporalSession struct {
	SessionID       string                 `json:"session_id"`
	TargetPattern   *SpatioTemporalPattern `json:"target_pattern"`
	ChallengePoints []SpatioTemporalPoint  `json:"challenge_points"`
	CorrectOption   string                 `json:"correct_option"`
	UserID          string                 `json:"user_id"`
	Status          string                 `json:"status"`
	VerifyCount     int                    `json:"verify_count"`
	MaxAttempts     int                    `json:"max_attempts"`
	CreatedAt       time.Time              `json:"created_at"`
	ExpiredAt       time.Time              `json:"expired_at"`
	Difficulty      string                 `json:"difficulty"`
	ClientIP        string                 `json:"client_ip"`
	UserAgent       string                 `json:"user_agent"`
}

// SpatioTemporalCaptchaService 时空验证码服务
type SpatioTemporalCaptchaService struct {
	sessions map[string]*SpatioTemporalSession
}

// NewSpatioTemporalCaptchaService 创建新的时空验证码服务
func NewSpatioTemporalCaptchaService() *SpatioTemporalCaptchaService {
	return &SpatioTemporalCaptchaService{
		sessions: make(map[string]*SpatioTemporalSession),
	}
}

// Generate 生成时空验证码
func (s *SpatioTemporalCaptchaService) Generate(req *SpatioTemporalCaptchaRequest) (*SpatioTemporalCaptchaResponse, error) {
	if req.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if req.PatternType == "" {
		req.PatternType = TimePatternDaily
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	sessionID := generateSpatioTemporalSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	targetPattern := s.generateTargetPattern(req.PatternType, req.Difficulty, req.ClientIP)
	challengePoints, correctOption := s.generateChallengePoints(targetPattern, req.Difficulty)
	instructions := s.generateInstructions(req.Difficulty)
	options := s.generateOptions(challengePoints, correctOption)

	session := &SpatioTemporalSession{
		SessionID:       sessionID,
		TargetPattern:   targetPattern,
		ChallengePoints: challengePoints,
		CorrectOption:   correctOption,
		UserID:          req.UserID,
		Status:          "pending",
		VerifyCount:     0,
		MaxAttempts:     3,
		CreatedAt:       time.Now(),
		ExpiredAt:       expiresAt,
		Difficulty:      req.Difficulty,
		ClientIP:        req.ClientIP,
		UserAgent:       req.UserAgent,
	}

	s.sessions[sessionID] = session

	return &SpatioTemporalCaptchaResponse{
		SessionID:       sessionID,
		TargetPattern:   targetPattern,
		ChallengePoints: challengePoints,
		Instructions:    instructions,
		Options:         options,
		ExpiresIn:       int64(5 * time.Minute / time.Second),
		ExpiresAt:       expiresAt.Unix(),
	}, nil
}

// Verify 验证时空验证码
func (s *SpatioTemporalCaptchaService) Verify(req *SpatioTemporalVerifyRequest) (*SpatioTemporalVerifyResponse, error) {
	session, exists := s.sessions[req.SessionID]
	if !exists {
		return &SpatioTemporalVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话不存在",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &SpatioTemporalVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &SpatioTemporalVerifyResponse{
			Success: false,
			Score:   0,
			Message: "验证次数已用完",
		}, nil
	}

	session.VerifyCount++

	locationMatchScore := s.calculateLocationMatchScore(req, session)
	timePatternScore := s.calculateTimePatternScore(req, session)
	behaviorMatchScore := s.calculateBehaviorMatchScore(req, session)

	totalScore := locationMatchScore*0.4 + timePatternScore*0.3 + behaviorMatchScore*0.3

	// 检查是否选择了正确的选项
	optionCorrect := req.SelectedOption == session.CorrectOption

	success := totalScore >= 0.7 && optionCorrect

	distanceToCentroid := s.calculateDistanceToCentroid(req, session)
	timeWindowMatch := s.calculateTimeWindowMatch(req, session)
	anomalyScore := s.calculateAnomalyScore(req, session)

	details := &SpatioTemporalVerifyDetails{
		LocationMatchScore:  locationMatchScore,
		TimePatternScore:    timePatternScore,
		BehaviorMatchScore:  behaviorMatchScore,
		DistanceToCentroid:  distanceToCentroid,
		TimeWindowMatch:     timeWindowMatch,
		AnomalyScore:        anomalyScore,
	}

	analytics := s.generateAnalytics(req, session)

	message := "验证成功"
	if !success {
		message = "验证失败，请再试一次"
	}

	if success {
		session.Status = "verified"
	}

	return &SpatioTemporalVerifyResponse{
		Success:   success,
		Score:     totalScore,
		Message:   message,
		Details:   details,
		Analytics: analytics,
	}, nil
}

// GetSession 获取会话
func (s *SpatioTemporalCaptchaService) GetSession(sessionID string) (*SpatioTemporalSession, bool) {
	session, exists := s.sessions[sessionID]
	return session, exists
}

// generateTargetPattern 生成目标模式
func (s *SpatioTemporalCaptchaService) generateTargetPattern(patternType TimePatternType, difficulty string, ip string) *SpatioTemporalPattern {
	rand.Seed(time.Now().UnixNano())

	pointCount := s.getPointCountByDifficulty(difficulty)
	points := make([]SpatioTemporalPoint, pointCount)

	// 生成一个中心点（模拟北京坐标）
	baseLat := 39.9042 + rand.NormFloat64()*0.1
	baseLng := 116.4074 + rand.NormFloat64()*0.1

	for i := 0; i < pointCount; i++ {
		points[i] = SpatioTemporalPoint{
			Timestamp:  time.Now().Unix() - int64(rand.Intn(86400*30)),
			Latitude:   baseLat + rand.NormFloat64()*0.05,
			Longitude:  baseLng + rand.NormFloat64()*0.05,
			IPAddress:  ip,
			Accuracy:   LocationAccuracyCity,
			Confidence: 0.7 + rand.Float64()*0.3,
		}
	}

	// 计算质心
	centroid := calculateCentroid(points)

	behaviorFeatures := make(map[string]float64)
	featureKeys := []string{
		"avg_velocity", "movement_frequency", "location_variance",
		"time_consistency", "device_consistency",
	}
	for _, key := range featureKeys {
		behaviorFeatures[key] = rand.Float64()
	}

	now := time.Now().Unix()
	timeWindow := TimeWindow{
		StartTime: now - 86400,
		EndTime:   now,
		Duration:  86400,
	}

	return &SpatioTemporalPattern{
		PatternID:        fmt.Sprintf("st_pattern_%s", generateSpatioTemporalSessionID()),
		PatternType:      patternType,
		Points:           points,
		Centroid:         centroid,
		TimeWindow:       timeWindow,
		BehaviorFeatures: behaviorFeatures,
		AnomalyScore:     rand.Float64() * 0.3,
	}
}

// generateChallengePoints 生成挑战点
func (s *SpatioTemporalCaptchaService) generateChallengePoints(pattern *SpatioTemporalPattern, difficulty string) ([]SpatioTemporalPoint, string) {
	pointCount := 4 // 4个选项

	points := make([]SpatioTemporalPoint, pointCount)
	correctIndex := rand.Intn(pointCount)

	for i := 0; i < pointCount; i++ {
		if i == correctIndex {
			// 正确的点：接近质心
			points[i] = SpatioTemporalPoint{
				Timestamp:  time.Now().Unix(),
				Latitude:   pattern.Centroid[0] + rand.NormFloat64()*0.01,
				Longitude:  pattern.Centroid[1] + rand.NormFloat64()*0.01,
				Accuracy:   LocationAccuracyCity,
				Confidence: 0.8 + rand.Float64()*0.2,
			}
		} else {
			// 错误的点：远离质心
			points[i] = SpatioTemporalPoint{
				Timestamp:  time.Now().Unix(),
				Latitude:   pattern.Centroid[0] + rand.NormFloat64()*2.0,
				Longitude:  pattern.Centroid[1] + rand.NormFloat64()*2.0,
				Accuracy:   LocationAccuracyCity,
				Confidence: 0.6 + rand.Float64()*0.3,
			}
		}
	}

	return points, fmt.Sprintf("option_%d", correctIndex)
}

// generateOptions 生成选项
func (s *SpatioTemporalCaptchaService) generateOptions(points []SpatioTemporalPoint, correctOption string) []ChallengeOption {
	options := make([]ChallengeOption, len(points))

	for i, point := range points {
		options[i] = ChallengeOption{
			OptionID:  fmt.Sprintf("option_%d", i),
			Point:     point,
			IsCorrect: fmt.Sprintf("option_%d", i) == correctOption,
		}
	}

	return options
}

// generateInstructions 生成指令
func (s *SpatioTemporalCaptchaService) generateInstructions(difficulty string) string {
	switch difficulty {
	case "easy":
		return "请选择与您通常活动区域最接近的位置"
	case "medium":
		return "请根据您的时空行为模式选择正确的位置"
	case "hard":
		return "请精确识别与您历史行为模式匹配的位置"
	default:
		return "请选择正确的位置"
	}
}

// calculateLocationMatchScore 计算位置匹配分数
func (s *SpatioTemporalCaptchaService) calculateLocationMatchScore(req *SpatioTemporalVerifyRequest, session *SpatioTemporalSession) float64 {
	if req.UserLocation == nil {
		return 0.5
	}

	distance := s.calculateDistanceToCentroid(req, session)

	// 距离越近，分数越高
	if distance < 0.1 {
		return 1.0
	} else if distance < 1.0 {
		return 0.9 - (distance * 0.1)
	} else if distance < 10.0 {
		return 0.8 - (distance / 100)
	}
	return 0.3
}

// calculateTimePatternScore 计算时间模式分数
func (s *SpatioTemporalCaptchaService) calculateTimePatternScore(req *SpatioTemporalVerifyRequest, session *SpatioTemporalSession) float64 {
	now := time.Now().Unix()
	window := session.TargetPattern.TimeWindow

	// 检查当前时间是否在正常活动时间窗口内
	if now >= window.StartTime && now <= window.EndTime {
		return 0.8 + rand.Float64()*0.2
	}
	return 0.3 + rand.Float64()*0.4
}

// calculateBehaviorMatchScore 计算行为匹配分数
func (s *SpatioTemporalCaptchaService) calculateBehaviorMatchScore(req *SpatioTemporalVerifyRequest, session *SpatioTemporalSession) float64 {
	score := 0.5

	// 检查响应时间
	if req.ResponseTime > 1000 && req.ResponseTime < 30000 {
		score += 0.2
	}

	// 模拟其他行为特征
	score += rand.Float64() * 0.3

	return math.Min(1.0, score)
}

// calculateDistanceToCentroid 计算到质心的距离
func (s *SpatioTemporalCaptchaService) calculateDistanceToCentroid(req *SpatioTemporalVerifyRequest, session *SpatioTemporalSession) float64 {
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

// calculateTimeWindowMatch 计算时间窗口匹配
func (s *SpatioTemporalCaptchaService) calculateTimeWindowMatch(req *SpatioTemporalVerifyRequest, session *SpatioTemporalSession) float64 {
	now := time.Now().Unix()
	window := session.TargetPattern.TimeWindow

	if now >= window.StartTime && now <= window.EndTime {
		return 1.0
	}

	// 计算距离窗口的时间差
	var timeDiff int64
	if now < window.StartTime {
		timeDiff = window.StartTime - now
	} else {
		timeDiff = now - window.EndTime
	}

	// 距离越近，分数越高
	if timeDiff < 3600 {
		return 0.8
	} else if timeDiff < 7200 {
		return 0.6
	}
	return 0.3
}

// calculateAnomalyScore 计算异常分数
func (s *SpatioTemporalCaptchaService) calculateAnomalyScore(req *SpatioTemporalVerifyRequest, session *SpatioTemporalSession) float64 {
	baseScore := session.TargetPattern.AnomalyScore

	// 添加随机变化
	return math.Min(1.0, baseScore+rand.Float64()*0.2)
}

// generateAnalytics 生成分析数据
func (s *SpatioTemporalCaptchaService) generateAnalytics(req *SpatioTemporalVerifyRequest, session *SpatioTemporalSession) *SpatioTemporalAnalytics {
	riskLevel := "low"
	riskFactors := []string{}

	locationConfidence := 0.7 + rand.Float64()*0.3
	timeConsistency := 0.6 + rand.Float64()*0.4
	behaviorConsistency := 0.5 + rand.Float64()*0.5

	overallRisk := (1 - locationConfidence) * 0.4 + (1 - timeConsistency) * 0.3 + (1 - behaviorConsistency) * 0.3

	if overallRisk > 0.7 {
		riskLevel = "high"
		riskFactors = append(riskFactors, "location_anomaly", "time_inconsistency")
	} else if overallRisk > 0.4 {
		riskLevel = "medium"
	}

	return &SpatioTemporalAnalytics{
		LocationConfidence:   locationConfidence,
		TimeConsistency:      timeConsistency,
		BehaviorConsistency:  behaviorConsistency,
		RiskLevel:            riskLevel,
		RiskFactors:          riskFactors,
	}
}

// getPointCountByDifficulty 根据难度获取点数量
func (s *SpatioTemporalCaptchaService) getPointCountByDifficulty(difficulty string) int {
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

// 辅助函数

// calculateCentroid 计算质心
func calculateCentroid(points []SpatioTemporalPoint) []float64 {
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

// haversineDistance 计算两个坐标点之间的Haversine距离（公里）
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // 地球半径（公里）

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// generateSpatioTemporalSessionID 生成会话ID
func generateSpatioTemporalSessionID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("st_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

// SerializeSession 序列化会话
func (s *SpatioTemporalSession) SerializeSession() ([]byte, error) {
	return json.Marshal(s)
}

// DeserializeSession 反序列化会话
func DeserializeSpatioTemporalSession(data []byte) (*SpatioTemporalSession, error) {
	var session SpatioTemporalSession
	err := json.Unmarshal(data, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
