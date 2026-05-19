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

type HapticGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type HapticCaptchaRequest struct {
	PatternType string `json:"pattern_type"`
	Difficulty  string `json:"difficulty"`
	GridSize    int    `json:"grid_size"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type HapticCaptchaResponse struct {
	SessionID    string          `json:"session_id"`
	Pattern      *HapticPattern   `json:"pattern"`
	ExpiresIn    int64           `json:"expires_in"`
	ExpiresAt    int64           `json:"expires_at"`
	Instructions string          `json:"instructions"`
	VisualHint   *HapticVisualHint `json:"visual_hint,omitempty"`
}

type HapticPattern struct {
	Type           string            `json:"type"`
	GridSize       int               `json:"grid_size"`
	TargetSequence []int             `json:"target_sequence"`
	Taps           []HapticTapConfig `json:"taps"`
	Duration       int               `json:"duration"`
	Intensity      float64           `json:"intensity"`
	VibrationHz    int               `json:"vibration_hz"`
	SequenceHint   []int             `json:"sequence_hint"`
}

type HapticTapConfig struct {
	Position int     `json:"position"`
	Duration float64 `json:"duration"`
	Pressure float64 `json:"pressure"`
}

type HapticVisualHint struct {
	GridSize int      `json:"grid_size"`
	Positions []Point `json:"positions"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

const (
	HapticPatternSequence   = "sequence"
	HapticPatternGrid      = "grid"
	HapticPatternDirection  = "direction"
	HapticPatternPressure  = "pressure"

	HapticDifficultyEasy   = "easy"
	HapticDifficultyMedium = "medium"
	HapticDifficultyHard   = "hard"
)

func NewHapticGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *HapticGeneratorService {
	return &HapticGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *HapticGeneratorService) Generate(ctx context.Context, req *HapticCaptchaRequest) (*HapticCaptchaResponse, error) {
	gridSize := 3
	if req.GridSize > 0 && req.GridSize <= 6 {
		gridSize = req.GridSize
	}

	difficulty := HapticDifficultyMedium
	switch req.Difficulty {
	case HapticDifficultyEasy, HapticDifficultyHard:
		difficulty = req.Difficulty
	}

	patternType := HapticPatternSequence
	if req.PatternType != "" {
		switch req.PatternType {
		case HapticPatternGrid, HapticPatternDirection, HapticPatternPressure:
			patternType = req.PatternType
		}
	}

	pattern := s.generatePattern(patternType, gridSize, difficulty)
	patternData, err := json.Marshal(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pattern: %w", err)
	}

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	hapticSession := &models.HapticCaptchaSession{
		SessionID:   sessionID,
		Pattern:     string(patternData),
		PatternType: patternType,
		Difficulty:  difficulty,
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   expiresAt,
		ClientIP:    req.ClientIP,
		UserAgent:   req.UserAgent,
		Fingerprint: req.Fingerprint,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.SetHaptic(ctx, hapticSession); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.CreateHapticSession(hapticSession); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	return &HapticCaptchaResponse{
		SessionID:    sessionID,
		Pattern:      pattern,
		ExpiresIn:    int64(5 * time.Minute / time.Second),
		ExpiresAt:    expiresAt.Unix(),
		Instructions: s.getInstructions(patternType, difficulty),
		VisualHint:   s.generateVisualHint(gridSize, pattern.TargetSequence),
	}, nil
}

func (s *HapticGeneratorService) generatePattern(patternType string, gridSize int, difficulty string) *HapticPattern {
	sequenceLength := s.getSequenceLength(difficulty)
	targetSequence := make([]int, sequenceLength)
	for i := 0; i < sequenceLength; i++ {
		targetSequence[i] = rand.Intn(gridSize * gridSize)
		for j := 0; j < i; j++ {
			if targetSequence[j] == targetSequence[i] {
				i--
				break
			}
		}
	}

	taps := s.generateTaps(targetSequence, difficulty)

	intensity := 0.5
	vibrationHz := 200
	switch difficulty {
	case HapticDifficultyEasy:
		intensity = 0.4
		vibrationHz = 150
	case HapticDifficultyHard:
		intensity = 0.8
		vibrationHz = 300
	}

	duration := 3000
	switch difficulty {
	case HapticDifficultyEasy:
		duration = 5000
	case HapticDifficultyHard:
		duration = 2000
	}

	sequenceHint := make([]int, len(targetSequence))
	copy(sequenceHint, targetSequence)

	return &HapticPattern{
		Type:           patternType,
		GridSize:       gridSize,
		TargetSequence: targetSequence,
		Taps:           taps,
		Duration:       duration,
		Intensity:      intensity,
		VibrationHz:    vibrationHz,
		SequenceHint:   sequenceHint,
	}
}

func (s *HapticGeneratorService) getSequenceLength(difficulty string) int {
	switch difficulty {
	case HapticDifficultyEasy:
		return 3
	case HapticDifficultyHard:
		return 6
	default:
		return 4
	}
}

func (s *HapticGeneratorService) generateTaps(sequence []int, difficulty string) []HapticTapConfig {
	taps := make([]HapticTapConfig, len(sequence))
	baseDuration := 150.0
	durationVariance := 50.0

	switch difficulty {
	case HapticDifficultyEasy:
		baseDuration = 200.0
		durationVariance = 30.0
	case HapticDifficultyHard:
		baseDuration = 100.0
		durationVariance = 80.0
	}

	for i, pos := range sequence {
		taps[i] = HapticTapConfig{
			Position: pos,
			Duration: baseDuration + rand.Float64()*durationVariance,
			Pressure: 0.5 + rand.Float64()*0.5,
		}
	}

	return taps
}

func (s *HapticGeneratorService) getInstructions(patternType string, difficulty string) string {
	var patternDesc string
	switch patternType {
	case HapticPatternSequence:
		patternDesc = "按特定顺序点击"
	case HapticPatternGrid:
		patternDesc = "点击网格中的位置"
	case HapticPatternDirection:
		patternDesc = "按指定方向滑动"
	case HapticPatternPressure:
		patternDesc = "按特定力度点击"
	default:
		patternDesc = "完成触觉验证"
	}

	var difficultyDesc string
	switch difficulty {
	case HapticDifficultyEasy:
		difficultyDesc = "简单"
	case HapticDifficultyHard:
		difficultyDesc = "困难"
	default:
		difficultyDesc = "中等"
	}

	return fmt.Sprintf("请%s（难度：%s）", patternDesc, difficultyDesc)
}

func (s *HapticGeneratorService) generateVisualHint(gridSize int, sequence []int) *HapticVisualHint {
	if len(sequence) == 0 {
		return nil
	}

	positions := make([]Point, len(sequence))
	for i, idx := range sequence {
		positions[i] = Point{
			X: idx % gridSize,
			Y: idx / gridSize,
		}
	}

	return &HapticVisualHint{
		GridSize:  gridSize,
		Positions: positions,
	}
}

func (s *HapticGeneratorService) GenerateDemo(ctx context.Context, req *HapticCaptchaRequest) (*HapticCaptchaResponse, error) {
	req.Difficulty = HapticDifficultyEasy
	req.GridSize = 3
	req.PatternType = HapticPatternSequence

	return s.Generate(ctx, req)
}
