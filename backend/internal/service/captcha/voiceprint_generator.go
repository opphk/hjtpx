package captcha

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type VoiceprintGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VoiceprintCaptchaRequest struct {
	PatternType  string `json:"pattern_type"`
	Complexity   int    `json:"complexity"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	Fingerprint  string `json:"fingerprint"`
}

type VoiceprintCaptchaResponse struct {
	SessionID    string               `json:"session_id"`
	Pattern      *VoiceprintPattern    `json:"pattern"`
	AudioData    string               `json:"audio_data"`
	ExpiresIn    int64                `json:"expires_in"`
	ExpiresAt    int64                `json:"expires_at"`
	Instructions string               `json:"instructions"`
}

type VoiceprintPattern struct {
	TargetPhrase  string   `json:"target_phrase"`
	Frequencies   []float64 `json:"frequencies"`
	Durations     []float64 `json:"durations"`
	Amplitudes    []float64 `json:"amplitudes"`
	Modulation    []float64 `json:"modulation"`
}

var voiceprintPhrases = []string{
	"芝麻开门", "身份验证", "声纹确认", "安全验证", "授权通过",
	"语音识别", "生物认证", "身份确认", "请说密码", "验证通过",
}

func NewVoiceprintGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VoiceprintGeneratorService {
	return &VoiceprintGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *VoiceprintGeneratorService) Generate(ctx context.Context, req *VoiceprintCaptchaRequest) (*VoiceprintCaptchaResponse, error) {
	complexity := 3
	if req.Complexity > 0 && req.Complexity <= 5 {
		complexity = req.Complexity
	}

	pattern := s.generatePattern(complexity)
	patternData, err := json.Marshal(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pattern: %w", err)
	}

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	voiceprintSession := &models.VoiceprintCaptchaSession{
		SessionID:      sessionID,
		Pattern:        string(patternData),
		TargetPhrase:   pattern.TargetPhrase,
		Status:         "pending",
		VerifyCount:    0,
		MaxAttempts:    3,
		CreatedAt:      time.Now(),
		ExpiredAt:      expiresAt,
		ClientIP:       req.ClientIP,
		UserAgent:      req.UserAgent,
		Fingerprint:    req.Fingerprint,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.SetVoiceprint(ctx, voiceprintSession); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.CreateVoiceprintSession(voiceprintSession); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	audioData := s.generateVoiceAudio(pattern)

	return &VoiceprintCaptchaResponse{
		SessionID:    sessionID,
		Pattern:      pattern,
		AudioData:    base64.StdEncoding.EncodeToString(audioData),
		ExpiresIn:    int64(5 * time.Minute / time.Second),
		ExpiresAt:    expiresAt.Unix(),
		Instructions: fmt.Sprintf("请朗读以下内容: %s", pattern.TargetPhrase),
	}, nil
}

func (s *VoiceprintGeneratorService) generatePattern(complexity int) *VoiceprintPattern {
	phraseIndex := rand.Intn(len(voiceprintPhrases))
	targetPhrase := voiceprintPhrases[phraseIndex]

	frequencies := make([]float64, complexity)
	durations := make([]float64, complexity)
	amplitudes := make([]float64, complexity)
	modulation := make([]float64, complexity)

	for i := 0; i < complexity; i++ {
		frequencies[i] = 100 + rand.Float64()*300
		durations[i] = 0.2 + rand.Float64()*0.3
		amplitudes[i] = 0.3 + rand.Float64()*0.5
		modulation[i] = rand.Float64()
	}

	return &VoiceprintPattern{
		TargetPhrase:  targetPhrase,
		Frequencies:   frequencies,
		Durations:     durations,
		Amplitudes:    amplitudes,
		Modulation:    modulation,
	}
}

func (s *VoiceprintGeneratorService) generateVoiceAudio(pattern *VoiceprintPattern) []byte {
	wav := createVoiceprintWAVHeader()
	sampleRate := 44100

	for i := 0; i < len(pattern.Frequencies); i++ {
		freq := pattern.Frequencies[i]
		duration := pattern.Durations[i]
		amplitude := pattern.Amplitudes[i]
		mod := pattern.Modulation[i]

		wave := generateVoiceprintWave(freq, duration, amplitude, mod, sampleRate)
		wav = append(wav, wave...)

		silence := generateSilence(50, sampleRate)
		wav = append(wav, silence...)
	}

	return wav
}

func createVoiceprintWAVHeader() []byte {
	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	copy(header[8:12], []byte("WAVE"))
	copy(header[12:16], []byte("fmt "))
	copy(header[16:20], []byte{0x10, 0x00, 0x00, 0x00})
	copy(header[20:22], []byte{0x01, 0x00})
	copy(header[22:24], []byte{0x01, 0x00})
	copy(header[24:28], []byte{0x44, 0xAC, 0x00, 0x00})
	copy(header[28:32], []byte{0x88, 0x58, 0x01, 0x00})
	copy(header[32:34], []byte{0x02, 0x00})
	copy(header[34:36], []byte{0x10, 0x00})
	copy(header[36:40], []byte("data"))

	return header
}

func generateVoiceprintWave(frequency float64, duration float64, amplitude float64, modulation float64, sampleRate int) []byte {
	numSamples := int(float64(sampleRate) * duration)
	data := make([]byte, numSamples*2)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		modFactor := 1.0 + 0.2*mathSin(2*3.14159*modulation*5*t)
		value := int16(32767 * amplitude * (0.5*mathSin(2*3.14159*frequency*t) +
			0.3*mathSin(2*3.14159*frequency*1.5*t) +
			0.2*mathSin(2*3.14159*frequency*2*t)*modFactor))

		data[i*2] = byte(value & 0xff)
		data[i*2+1] = byte((value >> 8) & 0xff)
	}

	return data
}
