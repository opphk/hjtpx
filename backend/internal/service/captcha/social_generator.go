package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type SocialBehaviorType string

const (
	SocialTypeTracePattern SocialBehaviorType = "trace_pattern"
	SocialTypeGestureConnect SocialBehaviorType = "gesture_connect"
	SocialTypeTimingSequence SocialBehaviorType = "timing_sequence"
	SocialTypeCollaborative  SocialBehaviorType = "collaborative"
)

type TracePoint struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Pressure  float64 `json:"pressure"`
	Angle     float64 `json:"angle"`
}

type TracePattern struct {
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	TargetShape  string       `json:"target_shape"`
	StartPoint   *TracePoint  `json:"start_point"`
	EndPoint     *TracePoint  `json:"end_point"`
	ControlPoints []TracePoint `json:"control_points"`
	TracePoints  []TracePoint  `json:"trace_points"`
	Difficulty   string       `json:"difficulty"`
}

type SocialPuzzle struct {
	Patterns       []TracePattern     `json:"patterns"`
	BehaviorType   SocialBehaviorType `json:"behavior_type"`
	Instructions   string             `json:"instructions"`
	Difficulty     string             `json:"difficulty"`
	TimeLimit      int                `json:"time_limit"`
	SimilarityThreshold float64        `json:"similarity_threshold"`
}

type CreateSocialRequest struct {
	Difficulty    string `json:"difficulty"`
	BehaviorType  string `json:"behavior_type"`
	PatternCount  int    `json:"pattern_count"`
	ClientIP      string `json:"client_ip"`
	UserAgent     string `json:"user_agent"`
	Fingerprint   string `json:"fingerprint"`
}

type CreateSocialResponse struct {
	SessionID     string        `json:"session_id"`
	Puzzle        *SocialPuzzle `json:"puzzle"`
	ExpiresIn     int64         `json:"expires_in"`
	ExpiresAt     int64         `json:"expires_at"`
}

type VerifySocialRequest struct {
	SessionID     string       `json:"session_id" binding:"required"`
	TraceData     []TracePoint `json:"trace_data" binding:"required"`
	PatternType   string       `json:"pattern_type"`
	StartTime     int64        `json:"start_time"`
	EndTime       int64        `json:"end_time"`
	TouchPoints   []TouchPoint `json:"touch_points"`
	MouseTrail    []TracePoint `json:"mouse_trail"`
	RiskScore     float64      `json:"risk_score"`
}

type VerifySocialResult struct {
	Success            bool    `json:"success"`
	Message            string  `json:"message"`
	Score              float64 `json:"score"`
	ShapeSimilarity    float64 `json:"shape_similarity"`
	SpeedAnalysis      string  `json:"speed_analysis"`
	PressureAnalysis   string  `json:"pressure_analysis"`
	NaturalnessScore   float64 `json:"naturalness_score"`
	SocialScore        float64 `json:"social_score"`
	Feedback           string  `json:"feedback"`
}

type SocialGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type SocialVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

var traceShapes = []string{
	"heart", "star", "circle", "square", "triangle",
	"wave", "zigzag", "spiral", "arrow", "heart2",
	"infinity", "letter_a", "letter_b", "letter_c",
	"smiley", "thumbs_up", "check", "cross", "lightning",
}

var socialInstructions = map[string][]string{
	"trace_pattern": {
		"请沿着虚线轨迹滑动，绘制出相同的图形",
		"跟随指导点完成图形绘制",
		"按照箭头方向完成图案",
	},
	"gesture_connect": {
		"按顺序连接所有圆点",
		"按正确顺序连接数字",
		"将相同颜色的点连接起来",
	},
	"timing_sequence": {
		"按照提示的节奏点击",
		"跟随节拍完成点击序列",
		"按照时间间隔点击指定区域",
	},
	"collaborative": {
		"完成一半后让下一位继续",
		"团队协作完成图案",
		"分步完成复杂图案",
	},
}

func NewSocialGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *SocialGeneratorService {
	return &SocialGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewSocialVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *SocialVerifierService {
	return &SocialVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *SocialGeneratorService) Create(ctx context.Context, req *CreateSocialRequest) (*CreateSocialResponse, error) {
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	behaviorType := SocialBehaviorType(req.BehaviorType)
	if behaviorType == "" {
		types := []SocialBehaviorType{SocialTypeTracePattern, SocialTypeGestureConnect, SocialTypeTimingSequence}
		behaviorType = types[rand.Intn(len(types))]
	}

	patternCount := req.PatternCount
	if patternCount <= 0 {
		patternCount = 1
	}
	if patternCount > 3 {
		patternCount = 3
	}

	puzzle := s.generateSocialPuzzle(behaviorType, difficulty, patternCount)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	puzzleData, err := json.Marshal(puzzle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal puzzle: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(puzzleData),
		SliderURL:     string(puzzleData),
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	return &CreateSocialResponse{
		SessionID: sessionID,
		Puzzle:    puzzle,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *SocialGeneratorService) generateSocialPuzzle(behaviorType SocialBehaviorType, difficulty string, patternCount int) *SocialPuzzle {
	rand.Seed(time.Now().UnixNano())

	patterns := make([]TracePattern, 0, patternCount)

	for i := 0; i < patternCount; i++ {
		pattern := s.generateTracePattern(behaviorType, difficulty)
		patterns = append(patterns, *pattern)
	}

	instructions := s.generateInstructions(behaviorType)

	timeLimit := s.getTimeLimit(difficulty)

	similarityThreshold := s.getSimilarityThreshold(difficulty)

	return &SocialPuzzle{
		Patterns:            patterns,
		BehaviorType:        behaviorType,
		Instructions:        instructions,
		Difficulty:          difficulty,
		TimeLimit:           timeLimit,
		SimilarityThreshold: similarityThreshold,
	}
}

func (s *SocialGeneratorService) generateTracePattern(behaviorType SocialBehaviorType, difficulty string) *TracePattern {
	shapeType := traceShapes[rand.Intn(len(traceShapes))]

	width := 300.0
	height := 300.0
	margin := 50.0

	startPoint := &TracePoint{
		X:         margin + rand.Float64()*(width-2*margin),
		Y:         margin + rand.Float64()*(height-2*margin),
		Timestamp: 0,
		Pressure:  0.5,
		Angle:     0,
	}

	endPoint := &TracePoint{
		X:         margin + rand.Float64()*(width-2*margin),
		Y:         margin + rand.Float64()*(height-2*margin),
		Timestamp: 0,
		Pressure:  0.5,
		Angle:     0,
	}

	controlPoints := s.generateControlPoints(shapeType, width, height, margin)

	tracePoints := s.generateTracePoints(shapeType, startPoint, endPoint, controlPoints, difficulty)

	return &TracePattern{
		ID:           fmt.Sprintf("pattern_%d", rand.Intn(10000)),
		Type:         string(behaviorType),
		TargetShape:  shapeType,
		StartPoint:   startPoint,
		EndPoint:     endPoint,
		ControlPoints: controlPoints,
		TracePoints:  tracePoints,
		Difficulty:   difficulty,
	}
}

func (s *SocialGeneratorService) generateControlPoints(shapeType string, width, height, margin float64) []TracePoint {
	points := make([]TracePoint, 0)

	switch shapeType {
	case "heart":
		points = append(points, TracePoint{X: width / 2, Y: height * 0.3})
		points = append(points, TracePoint{X: width * 0.3, Y: height * 0.2})
		points = append(points, TracePoint{X: width * 0.3, Y: height * 0.5})
		points = append(points, TracePoint{X: width / 2, Y: height * 0.8})
		points = append(points, TracePoint{X: width * 0.7, Y: height * 0.5})
		points = append(points, TracePoint{X: width * 0.7, Y: height * 0.2})
	case "star":
		for i := 0; i < 5; i++ {
			angle := float64(i)*72 - 90
			rad := angle * math.Pi / 180
			points = append(points, TracePoint{
				X: width/2 + float64(math.Cos(rad))*margin*2,
				Y: height/2 + float64(math.Sin(rad))*margin*2,
			})
		}
	case "circle":
		for i := 0; i < 8; i++ {
			angle := float64(i) * 45 * math.Pi / 180
			points = append(points, TracePoint{
				X: width/2 + float64(math.Cos(angle))*margin*1.5,
				Y: height/2 + float64(math.Sin(angle))*margin*1.5,
			})
		}
	case "square":
		points = append(points, TracePoint{X: margin, Y: margin})
		points = append(points, TracePoint{X: width - margin, Y: margin})
		points = append(points, TracePoint{X: width - margin, Y: height - margin})
		points = append(points, TracePoint{X: margin, Y: height - margin})
	case "triangle":
		points = append(points, TracePoint{X: width / 2, Y: margin})
		points = append(points, TracePoint{X: width - margin, Y: height - margin})
		points = append(points, TracePoint{X: margin, Y: height - margin})
	case "wave":
		for i := 0; i < 4; i++ {
			x := margin + float64(i)*((width-2*margin)/3)
			points = append(points, TracePoint{X: x, Y: height / 2})
			points = append(points, TracePoint{X: x + (width-2*margin)/6, Y: height/2 - margin})
			points = append(points, TracePoint{X: x + (width-2*margin)/3, Y: height / 2})
			points = append(points, TracePoint{X: x + (width-2*margin)/6*2, Y: height/2 + margin})
		}
	default:
		points = append(points, TracePoint{X: margin, Y: height / 2})
		points = append(points, TracePoint{X: width - margin, Y: height / 2})
	}

	return points
}

func (s *SocialGeneratorService) generateTracePoints(shapeType string, start, end *TracePoint, controls []TracePoint, difficulty string) []TracePoint {
	points := make([]TracePoint, 0)

	pointCount := 50
	switch difficulty {
	case "easy":
		pointCount = 30
	case "medium":
		pointCount = 50
	case "hard":
		pointCount = 80
	case "expert":
		pointCount = 100
	}

	if len(controls) > 0 {
		for i := 0; i < pointCount; i++ {
			t := float64(i) / float64(pointCount-1)

			segIndex := int(t * float64(len(controls)-1))
			if segIndex >= len(controls)-1 {
				segIndex = len(controls) - 2
			}

			p0 := controls[segIndex]
			p1 := controls[segIndex+1]

			segT := (t - float64(segIndex)/float64(len(controls)-1)) * float64(len(controls)-1)

			x := p0.X + (p1.X-p0.X)*segT + (rand.Float64()-0.5)*5
			y := p0.Y + (p1.Y-p0.Y)*segT + (rand.Float64()-0.5)*5

			points = append(points, TracePoint{
				X:         x,
				Y:         y,
				Timestamp: int64(i * 20),
				Pressure:  0.3 + rand.Float64()*0.4,
				Angle:     rand.Float64() * 360,
			})
		}
	} else {
		for i := 0; i < pointCount; i++ {
			t := float64(i) / float64(pointCount-1)
			x := start.X + (end.X-start.X)*t
			y := start.Y + (end.Y-start.Y)*t

			points = append(points, TracePoint{
				X:         x,
				Y:         y,
				Timestamp: int64(i * 20),
				Pressure:  0.3 + rand.Float64()*0.4,
				Angle:     rand.Float64() * 360,
			})
		}
	}

	return points
}

func (s *SocialGeneratorService) generateInstructions(behaviorType SocialBehaviorType) string {
	templates, ok := socialInstructions[string(behaviorType)]
	if !ok {
		templates = socialInstructions["trace_pattern"]
	}
	return templates[rand.Intn(len(templates))]
}

func (s *SocialGeneratorService) getTimeLimit(difficulty string) int {
	switch difficulty {
	case "easy":
		return 30
	case "medium":
		return 20
	case "hard":
		return 15
	case "expert":
		return 10
	default:
		return 20
	}
}

func (s *SocialGeneratorService) getSimilarityThreshold(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 0.5
	case "medium":
		return 0.65
	case "hard":
		return 0.75
	case "expert":
		return 0.85
	default:
		return 0.65
	}
}

func (s *SocialVerifierService) getSimilarityThreshold(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 0.5
	case "medium":
		return 0.65
	case "hard":
		return 0.75
	case "expert":
		return 0.85
	default:
		return 0.65
	}
}

func (s *SocialVerifierService) Verify(ctx context.Context, req *VerifySocialRequest) (*VerifySocialResult, error) {
	session, err := s.getSession(req.SessionID)
	if err != nil {
		return &VerifySocialResult{
			Success: false,
			Message: "会话不存在",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifySocialResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifySocialResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	s.incrementVerifyCount(req.SessionID)

	var puzzle SocialPuzzle
	if err := json.Unmarshal([]byte(session.BackgroundURL), &puzzle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal puzzle: %w", err)
	}

	if session.Status == "verified" {
		return &VerifySocialResult{
			Success:          true,
			Message:          "验证码已验证通过",
			Score:            100,
			ShapeSimilarity:  100,
			NaturalnessScore: 100,
			SocialScore:      100,
		}, nil
	}

	shapeSimilarity := s.calculateShapeSimilarity(&puzzle, req.TraceData)

	speedAnalysis := s.analyzeSpeed(req.TraceData, req.StartTime, req.EndTime)

	pressureAnalysis := s.analyzePressure(req.TraceData)

	naturalnessScore := s.calculateNaturalness(req.TraceData, req.TouchPoints)

	socialScore := s.calculateSocialScore(&puzzle, req)

	totalScore := shapeSimilarity*0.4 + naturalnessScore*0.3 + socialScore*0.3

	isSuccess := shapeSimilarity >= puzzle.SimilarityThreshold && naturalnessScore >= 0.4

	if isSuccess {
		session.Status = "verified"
		if s.sessionCache != nil {
			_ = s.sessionCache.UpdateStatus(ctx, req.SessionID, "verified")
		}
		if s.captchaRepo != nil {
			_ = s.captchaRepo.UpdateStatus(req.SessionID, "verified")
		}
	}

	return &VerifySocialResult{
		Success:          isSuccess,
		Message:         func() string {
			if isSuccess {
				return "社交行为验证成功"
			}
			return fmt.Sprintf("验证失败，图形匹配度 %.0f%%", shapeSimilarity*100)
		}(),
		Score:            totalScore * 100,
		ShapeSimilarity:  shapeSimilarity * 100,
		SpeedAnalysis:    speedAnalysis,
		PressureAnalysis: pressureAnalysis,
		NaturalnessScore: naturalnessScore * 100,
		SocialScore:      socialScore * 100,
		Feedback:         s.generateFeedback(&puzzle, shapeSimilarity),
	}, nil
}

func (s *SocialVerifierService) calculateShapeSimilarity(puzzle *SocialPuzzle, userTrace []TracePoint) float64 {
	if len(userTrace) < 5 || len(puzzle.Patterns) == 0 {
		return 0.3
	}

	pattern := puzzle.Patterns[0]
	targetTrace := pattern.TracePoints

	if len(targetTrace) == 0 {
		return 0.5
	}

	userNormalized := s.normalizeTrace(userTrace)
	targetNormalized := s.normalizeTrace(targetTrace)

	minLen := len(userNormalized)
	if len(targetNormalized) < minLen {
		minLen = len(targetNormalized)
	}

	var totalDistance float64
	for i := 0; i < minLen; i++ {
		dx := userNormalized[i].X - targetNormalized[i].X
		dy := userNormalized[i].Y - targetNormalized[i].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance
	}

	avgDistance := totalDistance / float64(minLen)

	maxDistance := 200.0
	similarity := math.Max(0, 1-avgDistance/maxDistance)

	return similarity
}

func (s *SocialVerifierService) normalizeTrace(trace []TracePoint) []TracePoint {
	if len(trace) == 0 {
		return trace
	}

	var minX, maxX, minY, maxY float64 = math.MaxFloat64, -math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64

	for _, p := range trace {
		if p.X < minX {
			minX = p.X
		}
	 if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	width := maxX - minX
	height := maxY - minY
	scale := math.Max(width, height)
	if scale == 0 {
		scale = 1
	}

	normalized := make([]TracePoint, len(trace))
	for i, p := range trace {
		normalized[i] = TracePoint{
			X:         (p.X - minX) / scale,
			Y:         (p.Y - minY) / scale,
			Timestamp: p.Timestamp,
			Pressure:  p.Pressure,
			Angle:     p.Angle,
		}
	}

	return normalized
}

func (s *SocialVerifierService) analyzeSpeed(trace []TracePoint, startTime, endTime int64) string {
	if len(trace) < 2 {
		return "轨迹数据不足"
	}

	totalDuration := float64(trace[len(trace)-1].Timestamp - trace[0].Timestamp)
	if totalDuration <= 0 {
		return "时间异常"
	}

	var totalDistance float64
	for i := 1; i < len(trace); i++ {
		dx := trace[i].X - trace[i-1].X
		dy := trace[i].Y - trace[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	avgSpeed := totalDistance / totalDuration * 1000

	if avgSpeed < 0.05 {
		return "速度过慢，可能犹豫不决"
	} else if avgSpeed > 5 {
		return "速度过快，可能是机器操作"
	} else if avgSpeed < 0.2 {
		return "速度较慢，符合人类特征"
	} else if avgSpeed > 2 {
		return "速度较快，但仍可接受"
	}
	return "速度正常"
}

func (s *SocialVerifierService) analyzePressure(trace []TracePoint) string {
	if len(trace) == 0 {
		return "无压力数据"
	}

	var totalPressure, minPressure, maxPressure float64 = 0, 1, 0

	for _, p := range trace {
		if p.Pressure < minPressure {
			minPressure = p.Pressure
		}
		if p.Pressure > maxPressure {
			maxPressure = p.Pressure
		}
		totalPressure += p.Pressure
	}

	avgPressure := totalPressure / float64(len(trace))
	pressureRange := maxPressure - minPressure

	if avgPressure < 0.1 {
		return "压力过低，可能使用触摸笔"
	} else if avgPressure > 0.9 {
		return "压力过大，可能用力过猛"
	} else if pressureRange < 0.1 {
		return "压力稳定，缺乏变化"
	} else if pressureRange > 0.5 {
		return "压力变化大，符合手指特征"
	}
	return "压力正常"
}

func (s *SocialVerifierService) calculateNaturalness(trace []TracePoint, touchPoints []TouchPoint) float64 {
	if len(trace) < 10 {
		return 0.3
	}

	var accelerationChanges int
	for i := 2; i < len(trace); i++ {
		dt1 := float64(trace[i-1].Timestamp - trace[i-2].Timestamp)
		dt2 := float64(trace[i].Timestamp - trace[i-1].Timestamp)

		if dt1 > 0 && dt2 > 0 {
			v1x := (trace[i-1].X - trace[i-2].X) / dt1
			v1y := (trace[i-1].Y - trace[i-2].Y) / dt1
			v2x := (trace[i].X - trace[i-1].X) / dt2
			v2y := (trace[i].Y - trace[i-1].Y) / dt2

			acc1 := math.Sqrt(v1x*v1x + v1y*v1y)
			acc2 := math.Sqrt(v2x*v2x + v2y*v2y)

			accChange := math.Abs(acc2 - acc1) / (acc1 + 0.01)
			if accChange > 0.5 {
				accelerationChanges++
			}
		}
	}

	changeRatio := float64(accelerationChanges) / float64(len(trace)-2)

	naturalness := 0.5
	if changeRatio > 0.1 && changeRatio < 0.5 {
		naturalness += 0.3
	}

	if len(touchPoints) > 0 {
		pressureVariation := 0.0
		for _, tp := range touchPoints {
			pressureVariation += math.Abs(tp.Pressure - 0.5)
		}
		pressureVariation /= float64(len(touchPoints))
		if pressureVariation > 0.1 {
			naturalness += 0.2
		}
	}

	return math.Min(1, naturalness)
}

func (s *SocialVerifierService) calculateSocialScore(puzzle *SocialPuzzle, req *VerifySocialRequest) float64 {
	socialScore := 0.5

	startMatches := false
	if len(req.TraceData) > 0 && len(puzzle.Patterns) > 0 {
		pattern := puzzle.Patterns[0]
		if pattern.StartPoint != nil {
			dx := req.TraceData[0].X - pattern.StartPoint.X
			dy := req.TraceData[0].Y - pattern.StartPoint.Y
			startDist := math.Sqrt(dx*dx + dy*dy)
			if startDist < 50 {
				startMatches = true
				socialScore += 0.2
			}
		}
	}
	_ = startMatches

	responseTime := req.EndTime - req.StartTime
	expectedTime := int64(puzzle.TimeLimit * 1000)

	if responseTime > 0 && responseTime < expectedTime*2 {
		socialScore += 0.15
	} else if responseTime >= expectedTime*2 {
		socialScore -= 0.1
	}

	return math.Min(1, math.Max(0, socialScore))
}

func (s *SocialVerifierService) generateFeedback(puzzle *SocialPuzzle, similarity float64) string {
	if similarity >= 0.8 {
		return "轨迹绘制得非常准确！"
	} else if similarity >= 0.6 {
		return "轨迹基本正确，可以更精确一些"
	} else if similarity >= 0.4 {
		return "轨迹偏差较大，请仔细跟随指导线"
	}
	return "轨迹偏离较多，建议重新尝试"
}

func (s *SocialVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if s.captchaRepo != nil {
		session, err := s.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *SocialVerifierService) incrementVerifyCount(sessionID string) {
	if s.sessionCache != nil {
		_ = s.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if s.captchaRepo != nil {
		_ = s.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (s *SocialVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return s.getSession(sessionID)
}

func (s *SocialVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := s.getSession(sessionID)
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
