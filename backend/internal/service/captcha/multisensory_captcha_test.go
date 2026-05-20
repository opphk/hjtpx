package captcha

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMultisensoryGeneratorService_Generate_AllTypes(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	req := &MultisensoryCaptchaRequest{
		Types:      []string{"visual", "audio", "tactile"},
		VisualType: "slider",
		Language:   "zh-CN",
		ClientIP:   "127.0.0.1",
		UserAgent:  "test-agent",
	}

	resp, err := service.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.Len(t, resp.Types, 3)
	assert.Contains(t, resp.Types, "visual")
	assert.Contains(t, resp.Types, "audio")
	assert.Contains(t, resp.Types, "tactile")
	assert.NotNil(t, resp.Visual)
	assert.NotNil(t, resp.Audio)
	assert.NotNil(t, resp.Tactile)
	assert.Equal(t, int64(300), resp.ExpiresIn)
	assert.Greater(t, resp.ExpiresAt, time.Now().Unix())
}

func TestMultisensoryGeneratorService_Generate_VisualOnly(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	req := &MultisensoryCaptchaRequest{
		Types:      []string{"visual"},
		VisualType: "slider",
		Language:   "en-US",
	}

	resp, err := service.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Types, 1)
	assert.NotNil(t, resp.Visual)
	assert.Nil(t, resp.Audio)
	assert.Nil(t, resp.Tactile)
}

func TestMultisensoryGeneratorService_Generate_EmojiVisual(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	req := &MultisensoryCaptchaRequest{
		Types:      []string{"visual"},
		VisualType: "emoji",
		Language:   "zh-CN",
	}

	resp, err := service.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Visual)
	assert.Equal(t, "emoji", resp.Visual.Type)
	assert.Len(t, resp.Visual.Emojis, 8)
	assert.NotEmpty(t, resp.Visual.TargetEmoji)
}

func TestMultisensoryGeneratorService_Generate_AudioOnly(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	req := &MultisensoryCaptchaRequest{
		Types:    []string{"audio"},
		Language: "en-US",
	}

	resp, err := service.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Audio)
	assert.NotEmpty(t, resp.Audio.VoiceData)
	assert.Equal(t, "en-US", resp.Audio.Language)
}

func TestMultisensoryGeneratorService_Generate_TactileOnly(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	req := &MultisensoryCaptchaRequest{
		Types: []string{"tactile"},
	}

	resp, err := service.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Tactile)
	assert.Len(t, resp.Tactile.Pattern, 8)
	assert.Len(t, resp.Tactile.Code, 4)
}

func TestMultisensoryGeneratorService_Generate_DefaultTypes(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	req := &MultisensoryCaptchaRequest{}

	resp, err := service.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Types, 3)
	assert.Equal(t, "slider", resp.Visual.Type)
}

func TestMultisensoryGeneratorService_Generate_SliderVisual(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	visual, answer, err := service.generateSliderVisual()
	assert.NoError(t, err)
	assert.NotNil(t, visual)
	assert.Contains(t, visual.BackgroundURL, "data:image/png;base64,")
	assert.Contains(t, visual.SliderURL, "data:image/png;base64,")
	assert.Contains(t, answer, ",")
}

func TestMultisensoryGeneratorService_Generate_EmojiVisual(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	visual, answer, err := service.generateEmojiVisual()
	assert.NoError(t, err)
	assert.NotNil(t, visual)
	assert.Equal(t, "emoji", visual.Type)
	assert.Len(t, visual.Emojis, 8)
	assert.NotEmpty(t, visual.TargetEmoji)
	assert.Contains(t, visual.Emojis, visual.TargetEmoji)
	assert.Equal(t, visual.TargetEmoji, answer)
}

func TestMultisensoryGeneratorService_Generate_AudioCaptcha(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	audio, code, err := service.generateAudioCaptcha("zh-CN")
	assert.NoError(t, err)
	assert.NotNil(t, audio)
	assert.NotEmpty(t, audio.VoiceData)
	assert.Equal(t, "zh-CN", audio.Language)
	assert.Len(t, code, 4)
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9')
	}
}

func TestMultisensoryGeneratorService_Generate_TactileCaptcha(t *testing.T) {
	service := NewMultisensoryGeneratorServiceSimple()

	tactile, code, err := service.generateTactileCaptcha()
	assert.NoError(t, err)
	assert.NotNil(t, tactile)
	assert.Len(t, tactile.Pattern, 8)
	assert.Len(t, code, 4)
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9')
	}
}

func TestMultisensoryVerifierService_Verify_ExpiredSession(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-expired-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       []string{"visual"},
		VisualAnswer: "100,50",
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now().Add(-10 * time.Minute),
		ExpiredAt:   time.Now().Add(-5 * time.Minute),
		Verified:    make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"visual": "100,50"},
	})
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证码已过期", result.Message)
}

func TestMultisensoryVerifierService_Verify_MaxAttempts(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-max-attempts-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       []string{"visual"},
		VisualAnswer: "100,50",
		Status:      "pending",
		VerifyCount: 3,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   time.Now().Add(5 * time.Minute),
		Verified:    make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"visual": "100,50"},
	})
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证次数已用完", result.Message)
}

func TestMultisensoryVerifierService_Verify_AlreadyVerified(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-verified-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       []string{"visual"},
		VisualAnswer: "100,50",
		Status:      "verified",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   time.Now().Add(5 * time.Minute),
		Verified:    map[string]bool{"visual": true},
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"visual": "100,50"},
	})
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.AllPassed)
	assert.Equal(t, "验证码已验证通过", result.Message)
}

func TestMultisensoryVerifierService_Verify_CorrectVisualSlider(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-correct-slider-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       []string{"visual"},
		VisualAnswer: "150,80",
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   time.Now().Add(5 * time.Minute),
		Verified:    make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"visual": "150,80"},
	})
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.AllPassed)
	assert.True(t, result.Verified["visual"])
}

func TestMultisensoryVerifierService_Verify_WrongVisualSlider(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-wrong-slider-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       []string{"visual"},
		VisualAnswer: "150,80",
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   time.Now().Add(5 * time.Minute),
		Verified:    make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"visual": "50,30"},
	})
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.False(t, result.AllPassed)
}

func TestMultisensoryVerifierService_Verify_VisualSliderTolerance(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-tolerance-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       []string{"visual"},
		VisualAnswer: "100,50",
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   time.Now().Add(5 * time.Minute),
		Verified:    make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"visual": "103,53"},
	})
	assert.NoError(t, err)
	assert.True(t, result.Success)
}

func TestMultisensoryVerifierService_Verify_CorrectAudio(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-correct-audio-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:    sessionID,
		Types:        []string{"audio"},
		AudioAnswer:  "1234",
		Status:       "pending",
		VerifyCount:  0,
		MaxAttempts:  3,
		CreatedAt:    time.Now(),
		ExpiredAt:    time.Now().Add(5 * time.Minute),
		Verified:     make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"audio": "1234"},
	})
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Verified["audio"])
}

func TestMultisensoryVerifierService_Verify_WrongAudio(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-wrong-audio-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:    sessionID,
		Types:        []string{"audio"},
		AudioAnswer:  "5678",
		Status:       "pending",
		VerifyCount:  0,
		MaxAttempts:  3,
		CreatedAt:    time.Now(),
		ExpiredAt:    time.Now().Add(5 * time.Minute),
		Verified:     make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"audio": "1234"},
	})
	assert.NoError(t, err)
	assert.False(t, result.Success)
}

func TestMultisensoryVerifierService_Verify_CorrectTactile(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-correct-tactile-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:     sessionID,
		Types:         []string{"tactile"},
		TactileAnswer: "0123",
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     time.Now().Add(5 * time.Minute),
		Verified:      make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: sessionID,
		Answers:   map[string]string{"tactile": "0123"},
	})
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Verified["tactile"])
}

func TestMultisensoryVerifierService_Verify_MultipleTypes_PartialSuccess(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-partial-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:     sessionID,
		Types:         []string{"visual", "audio"},
		VisualAnswer:  "100,50",
		AudioAnswer:   "5678",
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     time.Now().Add(5 * time.Minute),
		Verified:      make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID:  sessionID,
		Answers:    map[string]string{"visual": "100,50"},
		RequireAll: false,
	})
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, result.AllPassed)
	assert.True(t, result.Verified["visual"])
	assert.False(t, result.Verified["audio"])
}

func TestMultisensoryVerifierService_Verify_MultipleTypes_AllSuccess(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-all-success-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:     sessionID,
		Types:         []string{"visual", "audio"},
		VisualAnswer:  "100,50",
		AudioAnswer:   "5678",
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     time.Now().Add(5 * time.Minute),
		Verified:      make(map[string]bool),
	}
	multisensoryMu.Unlock()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID:  sessionID,
		Answers:    map[string]string{"visual": "100,50", "audio": "5678"},
		RequireAll: true,
	})
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.AllPassed)
	assert.True(t, result.Verified["visual"])
	assert.True(t, result.Verified["audio"])
}

func TestMultisensoryVerifierService_Verify_SessionNotFound(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	result, err := verifier.Verify(context.Background(), &MultisensoryVerifyRequest{
		SessionID: "nonexistent-session",
		Answers:   map[string]string{"visual": "100,50"},
	})
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "Session not found", result.Message)
}

func TestMultisensoryVerifierService_GetSessionStatus(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-status-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		Types:       []string{"visual"},
		VisualAnswer: "100,50",
		Status:      "pending",
		VerifyCount: 1,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   time.Now().Add(5 * time.Minute),
		Verified:    make(map[string]bool),
	}
	multisensoryMu.Unlock()

	session, err := verifier.GetSessionStatus(context.Background(), sessionID)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, sessionID, session.SessionID)
	assert.Equal(t, 1, session.VerifyCount)
}

func TestMultisensoryVerifierService_GetSessionStatus_NotFound(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	session, err := verifier.GetSessionStatus(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestMultisensoryVerifierService_VerifyType_UnknownType(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	session := &MultisensoryCaptchaSession{
		SessionID: "test-unknown",
	}

	correct, err := verifier.verifyType(session, "unknown", "test")
	assert.Error(t, err)
	assert.False(t, correct)
}

func TestMultisensoryVerifierService_VerifyVisual_NoAnswer(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	session := &MultisensoryCaptchaSession{
		SessionID:    "test-no-answer",
		VisualAnswer: "",
	}

	correct, err := verifier.verifyVisual(session, "100,50")
	assert.Error(t, err)
	assert.False(t, correct)
}

func TestMultisensoryVerifierService_VerifyAudio_NoAnswer(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	session := &MultisensoryCaptchaSession{
		SessionID:   "test-no-answer",
		AudioAnswer: "",
	}

	correct, err := verifier.verifyAudio(session, "1234")
	assert.Error(t, err)
	assert.False(t, correct)
}

func TestMultisensoryVerifierService_VerifyTactile_NoAnswer(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	session := &MultisensoryCaptchaSession{
		SessionID:     "test-no-answer",
		TactileAnswer: "",
	}

	correct, err := verifier.verifyTactile(session, "0123")
	assert.Error(t, err)
	assert.False(t, correct)
}

func TestMultisensoryVerifierService_IncrementVerifyCount(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-increment-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID:   sessionID,
		VerifyCount: 0,
	}
	multisensoryMu.Unlock()

	verifier.incrementVerifyCount(sessionID)

	multisensoryMu.RLock()
	session := multisensorySessions[sessionID]
	multisensoryMu.RUnlock()

	assert.Equal(t, 1, session.VerifyCount)
}

func TestMultisensoryVerifierService_MarkAsVerified(t *testing.T) {
	verifier := NewMultisensoryVerifierServiceSimple()

	sessionID := "test-mark-session"
	multisensoryMu.Lock()
	multisensorySessions[sessionID] = &MultisensoryCaptchaSession{
		SessionID: sessionID,
		Status:    "pending",
		Verified:  make(map[string]bool),
	}
	multisensoryMu.Unlock()

	verified := map[string]bool{"visual": true, "audio": true}
	verifier.markAsVerified(sessionID, verified)

	multisensoryMu.RLock()
	session := multisensorySessions[sessionID]
	multisensoryMu.RUnlock()

	assert.Equal(t, "verified", session.Status)
	assert.True(t, session.Verified["visual"])
	assert.True(t, session.Verified["audio"])
}

func TestMultisensoryCaptchaSession_Fields(t *testing.T) {
	session := &MultisensoryCaptchaSession{
		SessionID:     "test-session",
		Types:         []string{"visual", "audio", "tactile"},
		VisualAnswer:  "100,50",
		AudioAnswer:   "1234",
		TactileAnswer: "5678",
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     time.Now().Add(5 * time.Minute),
		ClientIP:      "192.168.1.1",
		UserAgent:     "Mozilla/5.0",
		Fingerprint:   "abc123",
		Verified:      make(map[string]bool),
	}

	assert.Equal(t, "test-session", session.SessionID)
	assert.Len(t, session.Types, 3)
	assert.Equal(t, "100,50", session.VisualAnswer)
	assert.Equal(t, "1234", session.AudioAnswer)
	assert.Equal(t, "5678", session.TactileAnswer)
	assert.Equal(t, "pending", session.Status)
	assert.Equal(t, 0, session.VerifyCount)
	assert.Equal(t, 3, session.MaxAttempts)
	assert.Equal(t, "192.168.1.1", session.ClientIP)
	assert.Equal(t, "Mozilla/5.0", session.UserAgent)
	assert.Equal(t, "abc123", session.Fingerprint)
}

func TestVisualCaptchaData_Fields(t *testing.T) {
	visual := &VisualCaptchaData{
		Type:          "slider",
		BackgroundURL: "data:image/png;base64,abc",
		SliderURL:     "data:image/png;base64,xyz",
		GapX:          100,
		GapY:          50,
		Emojis:        []string{"😊", "😂", "😍"},
		TargetEmoji:   "😊",
	}

	assert.Equal(t, "slider", visual.Type)
	assert.Equal(t, "data:image/png;base64,abc", visual.BackgroundURL)
	assert.Equal(t, "data:image/png;base64,xyz", visual.SliderURL)
	assert.Equal(t, 100, visual.GapX)
	assert.Equal(t, 50, visual.GapY)
	assert.Len(t, visual.Emojis, 3)
	assert.Equal(t, "😊", visual.TargetEmoji)
}

func TestAudioCaptchaData_Fields(t *testing.T) {
	audio := &AudioCaptchaData{
		VoiceData: "base64-encoded-audio",
		Language:  "en-US",
	}

	assert.Equal(t, "base64-encoded-audio", audio.VoiceData)
	assert.Equal(t, "en-US", audio.Language)
}

func TestTactileCaptchaData_Fields(t *testing.T) {
	tactile := &TactileCaptchaData{
		Pattern: []int{100, 200, 300, 400, 100, 200, 300, 400},
		Code:    "0123",
	}

	assert.Len(t, tactile.Pattern, 8)
	assert.Equal(t, "0123", tactile.Code)
}

func TestMultisensoryCaptchaRequest_Fields(t *testing.T) {
	req := &MultisensoryCaptchaRequest{
		Types:       []string{"visual", "audio"},
		VisualType:  "emoji",
		Language:    "zh-CN",
		ClientIP:    "10.0.0.1",
		UserAgent:   "TestAgent",
		Fingerprint: "fp123",
	}

	assert.Len(t, req.Types, 2)
	assert.Equal(t, "emoji", req.VisualType)
	assert.Equal(t, "zh-CN", req.Language)
	assert.Equal(t, "10.0.0.1", req.ClientIP)
	assert.Equal(t, "TestAgent", req.UserAgent)
	assert.Equal(t, "fp123", req.Fingerprint)
}

func TestMultisensoryCaptchaResponse_Fields(t *testing.T) {
	resp := &MultisensoryCaptchaResponse{
		SessionID: "session-123",
		Visual: &VisualCaptchaData{
			Type: "slider",
		},
		Audio: &AudioCaptchaData{
			VoiceData: "audio",
		},
		Tactile: &TactileCaptchaData{
			Pattern: []int{100, 200},
		},
		ExpiresIn: 300,
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		Types:     []string{"visual", "audio", "tactile"},
	}

	assert.Equal(t, "session-123", resp.SessionID)
	assert.NotNil(t, resp.Visual)
	assert.NotNil(t, resp.Audio)
	assert.NotNil(t, resp.Tactile)
	assert.Equal(t, int64(300), resp.ExpiresIn)
	assert.Len(t, resp.Types, 3)
}

func TestMultisensoryVerifyRequest_Fields(t *testing.T) {
	req := &MultisensoryVerifyRequest{
		SessionID:  "verify-session",
		Answers:    map[string]string{"visual": "100,50", "audio": "1234"},
		RequireAll: true,
	}

	assert.Equal(t, "verify-session", req.SessionID)
	assert.Len(t, req.Answers, 2)
	assert.True(t, req.RequireAll)
}

func TestMultisensoryVerifyResult_Fields(t *testing.T) {
	result := &MultisensoryVerifyResult{
		Success:   true,
		Message:   "验证成功",
		Verified:  map[string]bool{"visual": true, "audio": true},
		AllPassed: true,
	}

	assert.True(t, result.Success)
	assert.Equal(t, "验证成功", result.Message)
	assert.Len(t, result.Verified, 2)
	assert.True(t, result.AllPassed)
}
