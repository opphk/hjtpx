package captcha

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

// MultisensoryCaptchaSession 多感官验证码会话
type MultisensoryCaptchaSession struct {
	SessionID     string
	Types         []string
	VisualAnswer  string
	AudioAnswer   string
	TactileAnswer string
	Status        string
	VerifyCount   int
	MaxAttempts   int
	CreatedAt     time.Time
	ExpiredAt     time.Time
	ClientIP      string
	UserAgent     string
	Fingerprint   string
	Verified      map[string]bool
}

var (
	multisensorySessions = make(map[string]*MultisensoryCaptchaSession)
	multisensoryMu       sync.RWMutex
)

type MultisensoryGeneratorService struct {
	imageGenerator *ImageGenerator
}

type MultisensoryCaptchaRequest struct {
	Types       []string `json:"types"`       // visual, audio, tactile
	VisualType  string   `json:"visual_type"` // slider, emoji
	Language    string   `json:"language"`    // zh-CN, en-US
	ClientIP    string   `json:"client_ip"`
	UserAgent   string   `json:"user_agent"`
	Fingerprint string   `json:"fingerprint"`
}

type MultisensoryCaptchaResponse struct {
	SessionID string                `json:"session_id"`
	Visual    *VisualCaptchaData    `json:"visual,omitempty"`
	Audio     *AudioCaptchaData     `json:"audio,omitempty"`
	Tactile   *TactileCaptchaData   `json:"tactile,omitempty"`
	ExpiresIn int64                 `json:"expires_in"`
	ExpiresAt int64                 `json:"expires_at"`
	Types     []string              `json:"types"`
}

type VisualCaptchaData struct {
	Type          string `json:"type"`
	BackgroundURL string `json:"background_url,omitempty"`
	SliderURL     string `json:"slider_url,omitempty"`
	GapX          int    `json:"gap_x,omitempty"`
	GapY          int    `json:"gap_y,omitempty"`
	Emojis        []string `json:"emojis,omitempty"`
	TargetEmoji   string `json:"target_emoji,omitempty"`
}

type AudioCaptchaData struct {
	VoiceData string `json:"voice_data"`
	Language  string `json:"language"`
}

type TactileCaptchaData struct {
	Pattern []int `json:"pattern"` // vibration pattern in ms
	Code    string `json:"code"`    // corresponding code for tactile
}

func NewMultisensoryGeneratorServiceSimple() *MultisensoryGeneratorService {
	return &MultisensoryGeneratorService{
		imageGenerator: NewImageGenerator(),
	}
}

func (s *MultisensoryGeneratorService) Generate(ctx context.Context, req *MultisensoryCaptchaRequest) (*MultisensoryCaptchaResponse, error) {
	if len(req.Types) == 0 {
		req.Types = []string{"visual", "audio", "tactile"}
	}
	if req.Language == "" {
		req.Language = "zh-CN"
	}
	if req.VisualType == "" {
		req.VisualType = "slider"
	}

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	response := &MultisensoryCaptchaResponse{
		SessionID: sessionID,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
		Types:     req.Types,
	}

	session := &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       req.Types,
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   expiresAt,
		ClientIP:    req.ClientIP,
		UserAgent:   req.UserAgent,
		Fingerprint: req.Fingerprint,
		Verified:    make(map[string]bool),
	}

	for _, t := range req.Types {
		switch t {
		case "visual":
			visualData, visualAnswer, err := s.generateVisualCaptcha(req.VisualType)
			if err != nil {
				return nil, err
			}
			response.Visual = visualData
			session.VisualAnswer = visualAnswer
		case "audio":
			audioData, audioAnswer, err := s.generateAudioCaptcha(req.Language)
			if err != nil {
				return nil, err
			}
			response.Audio = audioData
			session.AudioAnswer = audioAnswer
		case "tactile":
			tactileData, tactileAnswer, err := s.generateTactileCaptcha()
			if err != nil {
				return nil, err
			}
			response.Tactile = tactileData
			session.TactileAnswer = tactileAnswer
		}
	}

	multisensoryMu.Lock()
	multisensorySessions[sessionID] = session
	multisensoryMu.Unlock()

	return response, nil
}

func (s *MultisensoryGeneratorService) generateVisualCaptcha(visualType string) (*VisualCaptchaData, string, error) {
	if visualType == "emoji" {
		return s.generateEmojiVisual()
	}
	return s.generateSliderVisual()
}

func (s *MultisensoryGeneratorService) generateSliderVisual() (*VisualCaptchaData, string, error) {
	result, err := s.imageGenerator.GenerateSliderCaptcha()
	if err != nil {
		return nil, "", err
	}

	backgroundURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Background)
	sliderURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Slider)

	return &VisualCaptchaData{
		Type:          "slider",
		BackgroundURL: backgroundURL,
		SliderURL:     sliderURL,
	}, fmt.Sprintf("%d,%d", result.GapX, result.GapY), nil
}

func (s *MultisensoryGeneratorService) generateEmojiVisual() (*VisualCaptchaData, string, error) {
	allEmojis := []string{"😊", "😂", "😍", "🥳", "😎", "🤔", "😴", "🥺", "😤", "😱", "🎉", "🔥", "⭐", "❤️", "👍", "👏"}
	rand.Shuffle(len(allEmojis), func(i, j int) {
		allEmojis[i], allEmojis[j] = allEmojis[j], allEmojis[i]
	})

	selectedEmojis := allEmojis[:8]
	targetEmoji := selectedEmojis[rand.Intn(len(selectedEmojis))]

	return &VisualCaptchaData{
		Type:        "emoji",
		Emojis:      selectedEmojis,
		TargetEmoji: targetEmoji,
	}, targetEmoji, nil
}

func (s *MultisensoryGeneratorService) generateAudioCaptcha(language string) (*AudioCaptchaData, string, error) {
	code := generateRandomDigits(4)
	audioData := generateVoiceAudio(code, language)

	return &AudioCaptchaData{
		VoiceData: base64.StdEncoding.EncodeToString(audioData),
		Language:  language,
	}, code, nil
}

func (s *MultisensoryGeneratorService) generateTactileCaptcha() (*TactileCaptchaData, string, error) {
	pattern := []int{}
	code := ""
	
	for i := 0; i < 4; i++ {
		digit := rand.Intn(10)
		code += strconv.Itoa(digit)
		
		shortVib := 100 + rand.Intn(100)
		longVib := 300 + rand.Intn(200)
		pause := 200 + rand.Intn(200)
		
		if digit < 5 {
			pattern = append(pattern, shortVib, pause)
		} else {
			pattern = append(pattern, longVib, pause)
		}
	}

	return &TactileCaptchaData{
		Pattern: pattern,
		Code:    code,
	}, code, nil
}


