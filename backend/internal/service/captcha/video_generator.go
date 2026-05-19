package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

type VideoGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VideoCaptchaRequest struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Difficulty  int    `json:"difficulty"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type VideoCaptchaResponse struct {
	SessionID   string                 `json:"session_id"`
	VideoURL    string                 `json:"video_url"`
	VideoData   string                 `json:"video_data,omitempty"`
	Question    string                 `json:"question"`
	Options     []string               `json:"options"`
	TargetAction string                `json:"target_action"`
	Duration    int                    `json:"duration"`
	ExpiresIn   int64                 `json:"expires_in"`
	ExpiresAt   int64                 `json:"expires_at"`
	Difficulty  int                    `json:"difficulty"`
}

type VideoCaptchaSession struct {
	SessionID     string    `json:"session_id"`
	VideoURL      string    `json:"video_url"`
	VideoData     string    `json:"video_data"`
	Question      string    `json:"question"`
	Options       []string  `json:"options"`
	TargetAction  string    `json:"target_action"`
	Answer        string    `json:"answer"`
	Duration      int       `json:"duration"`
	Difficulty    int       `json:"difficulty"`
	Status        string    `json:"status"`
	VerifyCount   int       `json:"verify_count"`
	MaxAttempts   int       `json:"max_attempts"`
	RiskScore     float64   `json:"risk_score"`
	TraceScore    float64   `json:"trace_score"`
	EnvScore      float64   `json:"env_score"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiredAt     time.Time `json:"expired_at"`
	ClientIP      string    `json:"client_ip"`
	UserAgent     string    `json:"user_agent"`
	Fingerprint   string    `json:"fingerprint"`
}

var videoActionLibrary = []string{
	"举手", "挥手", "点头", "摇头", "眨眼", "张嘴",
	"抬手", "放下", "向左看", "向右看", "向上看", "向下看",
}

var videoActionLabels = map[string][]string{
	"举手":   {"举手", "举手过头顶", "抬起手"},
	"挥手":   {"挥手", "挥动手臂", "摆动手"},
	"点头":   {"点头", "向下点头", "头部向下"},
	"摇头":   {"摇头", "向左摇头", "向右摇头"},
	"眨眼":   {"眨眼", "快速眨眼", "眨眼睛"},
	"张嘴":   {"张嘴", "张开嘴巴", "张大嘴"},
	"抬手":   {"抬手", "举起手", "抬起手臂"},
	"放下":   {"放下", "放下手", "放下手臂"},
	"向左看": {"向左看", "眼睛向左", "头部左转"},
	"向右看": {"向右看", "眼睛向右", "头部右转"},
	"向上看": {"向上看", "眼睛向上", "抬头"},
	"向下看": {"向下看", "眼睛向下", "低头"},
}

func NewVideoGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VideoGeneratorService {
	return &VideoGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewVideoGeneratorServiceSimple() *VideoGeneratorService {
	return &VideoGeneratorService{}
}

func (s *VideoGeneratorService) Create(ctx context.Context, req *VideoCaptchaRequest) (*VideoCaptchaResponse, error) {
	sessionID := generateVideoSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	if req.Width <= 0 {
		req.Width = 640
	}
	if req.Height <= 0 {
		req.Height = 360
	}
	if req.Difficulty <= 0 {
		req.Difficulty = 2
	}

	targetAction := s.selectTargetAction(req.Difficulty)
	question := s.generateQuestion(targetAction)
	options := s.generateOptions(targetAction)
	videoURL := fmt.Sprintf("/static/video/action_%d.mp4", rand.Intn(5)+1)

	duration := s.calculateDuration(req.Difficulty)

	session := &VideoCaptchaSession{
		SessionID:    sessionID,
		VideoURL:     videoURL,
		VideoData:    "",
		Question:     question,
		Options:      options,
		TargetAction: targetAction,
		Answer:       targetAction,
		Duration:     duration,
		Difficulty:   req.Difficulty,
		Status:       "pending",
		VerifyCount:  0,
		MaxAttempts:  3,
		RiskScore:    0,
		TraceScore:   0,
		EnvScore:     0,
		CreatedAt:    time.Now(),
		ExpiredAt:    expiresAt,
		ClientIP:     req.ClientIP,
		UserAgent:    req.UserAgent,
		Fingerprint:  req.Fingerprint,
	}

	if s.sessionCache != nil {
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session: %w", err)
		}
		if err := s.sessionCache.SetRaw(ctx, sessionID, string(sessionJSON), 5*time.Minute); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	return &VideoCaptchaResponse{
		SessionID:    sessionID,
		VideoURL:     videoURL,
		Question:     question,
		Options:      options,
		TargetAction: targetAction,
		Duration:     duration,
		ExpiresIn:     int64(5 * time.Minute / time.Second),
		ExpiresAt:    expiresAt.Unix(),
		Difficulty:   req.Difficulty,
	}, nil
}

func (s *VideoGeneratorService) GetSession(ctx context.Context, sessionID string) (*VideoCaptchaSession, error) {
	if s.sessionCache != nil {
		sessionData, err := s.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session VideoCaptchaSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				return &session, nil
			}
		}
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *VideoGeneratorService) UpdateSession(ctx context.Context, session *VideoCaptchaSession) error {
	if s.sessionCache != nil {
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return fmt.Errorf("failed to marshal session: %w", err)
		}
		remainingTime := time.Until(session.ExpiredAt)
		if remainingTime <= 0 {
			return fmt.Errorf("session expired")
		}
		if err := s.sessionCache.SetRaw(ctx, session.SessionID, string(sessionJSON), remainingTime); err != nil {
			return fmt.Errorf("failed to update session cache: %w", err)
		}
	}
	return nil
}

func (s *VideoGeneratorService) selectTargetAction(difficulty int) string {
	if difficulty == 1 {
		simpleActions := []string{"举手", "挥手", "点头", "摇头"}
		return simpleActions[rand.Intn(len(simpleActions))]
	}
	return videoActionLibrary[rand.Intn(len(videoActionLibrary))]
}

func (s *VideoGeneratorService) generateQuestion(action string) string {
	questions := []string{
		"请做出 %s 的动作",
		"视频中的人物正在%s，请选择正确的动作",
		"识别视频中的动作：%s",
		"请回答视频中执行的动作是什么",
	}
	questionTemplate := questions[rand.Intn(len(questions))]
	return fmt.Sprintf(questionTemplate, action)
}

func (s *VideoGeneratorService) generateOptions(correctAction string) []string {
	options := make([]string, 0, 4)
	options = append(options, correctAction)

	usedActions := map[string]bool{correctAction: true}
	attempts := 0
	for len(options) < 4 && attempts < 20 {
		action := videoActionLibrary[rand.Intn(len(videoActionLibrary))]
		if !usedActions[action] {
			options = append(options, action)
			usedActions[action] = true
		}
		attempts++
	}

	for i := len(options) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		options[i], options[j] = options[j], options[i]
	}

	return options
}

func (s *VideoGeneratorService) calculateDuration(difficulty int) int {
	switch difficulty {
	case 1:
		return 5
	case 2:
		return 8
	case 3:
		return 12
	default:
		return 8
	}
}

func generateVideoSessionID() string {
	return fmt.Sprintf("video_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
