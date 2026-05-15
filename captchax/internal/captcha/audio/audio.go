package audio

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type Audio struct {
	cfg           interface{}
	redis         interface{}
	generator     *Generator
	cacheManager  *CacheManager
}

type AudioConfig struct {
	CodeLength    int
	SampleRate    int
	Duration      int
	VoiceType     VoiceType
	EnableNoise   bool
	EnableTremolo bool
}

func New(cfg interface{}, redisClient interface{}) *Audio {
	generator := NewGenerator()
	return &Audio{
		cfg:       cfg,
		redis:     redisClient,
		generator: generator,
	}
}

func (a *Audio) SetCacheManager(cm *CacheManager) {
	a.cacheManager = cm
}

func (a *Audio) GenerateCaptcha(ctx context.Context) (*CaptchaResult, error) {
	id := uuid.New().String()

	codeLength := 4 + rand.Intn(3)

	code := a.generator.GenerateRandomCode(codeLength)

	voiceType := VoiceType(rand.Intn(3))

	audioData, duration, err := a.generator.GenerateWAVAudio(code, voiceType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	audioB64 := base64.StdEncoding.EncodeToString(audioData)

	if a.cacheManager != nil {
		cacheData := &CacheData{
			ID:        id,
			Code:      code,
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}
		if err := a.cacheManager.Set(ctx, id, cacheData); err != nil {
			return nil, fmt.Errorf("failed to store captcha: %w", err)
		}
	}

	return &CaptchaResult{
		ID:       id,
		AudioB64: audioB64,
		Duration: duration,
	}, nil
}

func (a *Audio) GenerateCaptchaWithCode(ctx context.Context, code string) (*CaptchaResult, error) {
	id := uuid.New().String()

	if code == "" {
		code = a.generator.GenerateRandomCode(4 + rand.Intn(3))
	}

	voiceType := VoiceType(rand.Intn(3))

	audioData, duration, err := a.generator.GenerateWAVAudio(code, voiceType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	audioB64 := base64.StdEncoding.EncodeToString(audioData)

	if a.cacheManager != nil {
		cacheData := &CacheData{
			ID:        id,
			Code:      code,
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}
		if err := a.cacheManager.Set(ctx, id, cacheData); err != nil {
			return nil, fmt.Errorf("failed to store captcha: %w", err)
		}
	}

	return &CaptchaResult{
		ID:       id,
		AudioB64: audioB64,
		Duration: duration,
	}, nil
}

func (a *Audio) GetAudioData(code string, voiceType VoiceType) ([]byte, int, error) {
	return a.generator.GenerateWAVAudio(code, voiceType)
}

func (a *Audio) GetAudioB64(code string, voiceType VoiceType) (string, error) {
	audioData, _, err := a.generator.GenerateWAVAudio(code, voiceType)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(audioData), nil
}

func (a *Audio) GetAudioBuffer(code string, voiceType VoiceType) (*bytes.Buffer, error) {
	audioData, _, err := a.generator.GenerateWAVAudio(code, voiceType)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(audioData), nil
}

func (a *Audio) GenerateWithCustomVoice(ctx context.Context, voiceType VoiceType) (*CaptchaResult, error) {
	id := uuid.New().String()

	code := a.generator.GenerateRandomCode(4 + rand.Intn(3))

	audioData, duration, err := a.generator.GenerateWAVAudio(code, voiceType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	audioB64 := base64.StdEncoding.EncodeToString(audioData)

	if a.cacheManager != nil {
		cacheData := &CacheData{
			ID:        id,
			Code:      code,
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}
		if err := a.cacheManager.Set(ctx, id, cacheData); err != nil {
			return nil, fmt.Errorf("failed to store captcha: %w", err)
		}
	}

	return &CaptchaResult{
		ID:       id,
		AudioB64: audioB64,
		Duration: duration,
	}, nil
}

func (a *Audio) GetCacheManager() *CacheManager {
	return a.cacheManager
}

func (a *Audio) GetGenerator() *Generator {
	return a.generator
}

var ErrInvalidCaptchaID = errors.New("invalid captcha ID")
