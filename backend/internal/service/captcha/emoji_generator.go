package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type EmojiGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type CreateEmojiCaptchaRequest struct {
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateEmojiCaptchaResponse struct {
	SessionID      string   `json:"session_id"`
	TargetEmojis   []string `json:"target_emojis"`
	ShuffledEmojis []string `json:"shuffled_emojis"`
	ExpiresIn      int64    `json:"expires_in"`
	ExpiresAt      int64    `json:"expires_at"`
}

// EmojiCaptchaSession 表情验证码会话（用于缓存存储）
type EmojiCaptchaSession struct {
	SessionID      string
	TargetEmojis   []string
	ShuffledEmojis []string
	Status         string
	VerifyCount    int
	MaxAttempts    int
	RiskScore      float64
	TraceScore     float64
	EnvScore       float64
	CreatedAt      time.Time
	ExpiredAt      time.Time
	ClientIP       string
	UserAgent      string
	Fingerprint    string
}

// 表情库 - 包含常见的Unicode表情
var emojiLibrary = []string{
	"😀", "😁", "😂", "😃", "😄", "😅", "😆", "😇", "😈", "😉",
	"😊", "😋", "😌", "😍", "😎", "😏", "😐", "😑", "😒", "😓",
	"😔", "😕", "😖", "😗", "😘", "😙", "😚", "😛", "😜", "😝",
	"😞", "😟", "😠", "😡", "😢", "😣", "😤", "😥", "😦", "😧",
	"😨", "😩", "😪", "😫", "😬", "😭", "😮", "😯", "😰", "😱",
	"😲", "😳", "😴", "😵", "😶", "😷", "😸", "😹", "😺", "😻",
	"😼", "😽", "😾", "😿", "🙀", "🙁", "🙂", "🙃", "🙄", "🙅",
	"🙆", "🙇", "🙈", "🙉", "🙊", "🙋", "🙌", "🙍", "🙎", "🙏",
}

func NewEmojiGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *EmojiGeneratorService {
	return &EmojiGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *EmojiGeneratorService) Create(ctx context.Context, req *CreateEmojiCaptchaRequest) (*CreateEmojiCaptchaResponse, error) {
	sessionID := generateEmojiSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	// 生成目标表情序列 (4个表情)
	targetCount := 4
	targetEmojis := make([]string, targetCount)
	usedIndices := make(map[int]bool)
	for i := 0; i < targetCount; i++ {
		for {
			idx := rand.Intn(len(emojiLibrary))
			if !usedIndices[idx] {
				usedIndices[idx] = true
				targetEmojis[i] = emojiLibrary[idx]
				break
			}
		}
	}

	// 生成打乱后的表情列表 (目标 + 6个干扰 = 10个)
	shuffledEmojis := make([]string, 0, 10)
	shuffledEmojis = append(shuffledEmojis, targetEmojis...)
	for i := 0; i < 6; i++ {
		for {
			idx := rand.Intn(len(emojiLibrary))
			if !usedIndices[idx] {
				usedIndices[idx] = true
				shuffledEmojis = append(shuffledEmojis, emojiLibrary[idx])
				break
			}
		}
	}

	// Fisher-Yates 打乱
	for i := len(shuffledEmojis) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffledEmojis[i], shuffledEmojis[j] = shuffledEmojis[j], shuffledEmojis[i]
	}

	// 创建表情验证码会话
	session := &EmojiCaptchaSession{
		SessionID:      sessionID,
		TargetEmojis:   targetEmojis,
		ShuffledEmojis: shuffledEmojis,
		Status:         "pending",
		VerifyCount:    0,
		MaxAttempts:    3,
		RiskScore:      0,
		TraceScore:     0,
		EnvScore:       0,
		CreatedAt:      time.Now(),
		ExpiredAt:      expiresAt,
		ClientIP:       req.ClientIP,
		UserAgent:      req.UserAgent,
		Fingerprint:    req.Fingerprint,
	}

	// 将表情数据序列化为JSON并存储到缓存中
	if s.sessionCache != nil {
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session: %w", err)
		}
		if err := s.sessionCache.SetRaw(ctx, sessionID, string(sessionJSON), 5*time.Minute); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	return &CreateEmojiCaptchaResponse{
		SessionID:      sessionID,
		TargetEmojis:   targetEmojis,
		ShuffledEmojis: shuffledEmojis,
		ExpiresIn:      int64(5 * time.Minute / time.Second),
		ExpiresAt:      expiresAt.Unix(),
	}, nil
}

func (s *EmojiGeneratorService) GetSession(ctx context.Context, sessionID string) (*EmojiCaptchaSession, error) {
	if s.sessionCache != nil {
		sessionData, err := s.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session EmojiCaptchaSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				return &session, nil
			}
		}
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *EmojiGeneratorService) UpdateSession(ctx context.Context, session *EmojiCaptchaSession) error {
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

func generateEmojiSessionID() string {
	return fmt.Sprintf("emoji_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
