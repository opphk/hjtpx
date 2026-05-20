package captcha

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	github.com/hjtpx/hjtpx/internal/repository/cache"
	github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type VoiceGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VoiceCaptchaRequest struct {
	Language    string `json:"language"` // "zh-CN" or "en-US"
	Length      int    `json:"length"`   // number of digits, default 4
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type VoiceCaptchaResponse struct {
	SessionID string `json:"session_id"`
	VoiceData string `json:"voice_data"` // base64 encoded audio
	ExpiresIn int64  `json:"expires_in"`
	ExpiresAt int64  `json:"expires_at"`
	Language  string `json:"language"`
}

func NewVoiceGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VoiceGeneratorService {
	return &VoiceGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *VoiceGeneratorService) Generate(ctx context.Context, req *VoiceCaptchaRequest) (*VoiceCaptchaResponse, error) {
	if req.Length <= 0 {
		req.Length = 4
	}
	if req.Language == "" {
		req.Language = "zh-CN"
	}

	code := generateRandomDigits(req.Length)
	audioData := generateVoiceAudio(code, req.Language)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	voiceSession := &models.VoiceCaptchaSession{
		SessionID:   sessionID,
		Code:        code,
		Language:    req.Language,
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
		if err := s.sessionCache.SetVoice(ctx, voiceSession); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.CreateVoiceSession(voiceSession); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	return &VoiceCaptchaResponse{
		SessionID: sessionID,
		VoiceData: base64.StdEncoding.EncodeToString(audioData),
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
		Language:  req.Language,
	}, nil
}

func generateRandomDigits(length int) string {
	digits := ""
	for i := 0; i < length; i++ {
		digits += strconv.Itoa(rand.Intn(10))
	}
	return digits
}

func generateVoiceAudio(code, language string) []byte {
	wav := createWAVHeader()

	sampleRate := 44100
	// bitDepth := 16  // unused
	// channels := 1   // unused

	for _, char := range code {
		digit := int(char - '0')
		wave := generateDigitWave(digit, language, sampleRate)
		wav = append(wav, wave...)

		silence := generateSilence(100, sampleRate)
		wav = append(wav, silence...)
	}

	return wav
}

func createWAVHeader() []byte {
	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	copy(header[8:12], []byte("WAVE"))
	copy(header[12:16], []byte("fmt "))
	copy(header[22:24], []byte{0x01, 0x00})             // Mono
	copy(header[24:28], []byte{0x44, 0xAC, 0x00, 0x00}) // 44100 Hz
	copy(header[28:32], []byte{0x88, 0x58, 0x01, 0x00}) // Byte rate
	copy(header[32:34], []byte{0x02, 0x00})             // Block align
	copy(header[34:36], []byte{0x10, 0x00})             // 16 bits per sample
	copy(header[36:40], []byte("data"))

	return header
}

func generateDigitWave(digit int, language string, sampleRate int) []byte {
	duration := 0.5
	numSamples := int(float64(sampleRate) * duration)
	data := make([]byte, numSamples*2)

	frequency := 440.0 + float64(digit)*100

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		value := int16(32767 * 0.3 * (0.6*mathSin(2*3.14159*frequency*t) + 0.3*mathSin(2*3.14159*frequency*2*t) + 0.1*mathSin(2*3.14159*frequency*3*t)))

		data[i*2] = byte(value & 0xff)
		data[i*2+1] = byte((value >> 8) & 0xff)
	}

	return data
}

func generateSilence(durationMs int, sampleRate int) []byte {
	numSamples := (sampleRate * durationMs) / 1000
	data := make([]byte, numSamples*2)
	return data
}

func (s *VoiceGeneratorService) calculateTone(duration, baseFreq float64) {
	// Calculate tone parameters
}
