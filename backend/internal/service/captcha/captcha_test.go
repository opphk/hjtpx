package captcha

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestImageGenerator_GenerateSliderCaptcha(t *testing.T) {
	generator := NewImageGenerator()

	result, err := generator.GenerateSliderCaptcha()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Background)
	assert.NotNil(t, result.Slider)
	assert.Greater(t, result.GapX, 0)
	assert.Less(t, result.GapX, 300)
	assert.Greater(t, result.GapY, 0)
	assert.Less(t, result.GapY, 160)
}

func TestImageGenerator_SetDimensions(t *testing.T) {
	generator := NewImageGenerator()

	generator.SetDimensions(400, 200, 50, 50)
	assert.Equal(t, 400, generator.width)
	assert.Equal(t, 200, generator.height)
	assert.Equal(t, 50, generator.sliderWidth)
	assert.Equal(t, 50, generator.sliderHeight)
}

func TestImageGenerator_GenerateSliderCaptcha_CustomDimensions(t *testing.T) {
	generator := NewImageGenerator()

	generator.SetDimensions(500, 300, 60, 60)
	result, err := generator.GenerateSliderCaptcha()
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestImageGenerator_ClampUint8(t *testing.T) {
	generator := NewImageGenerator()

	tests := []struct {
		input    int
		expected uint8
	}{
		{0, 0},
		{128, 128},
		{255, 255},
		{-10, 0},
		{-100, 0},
		{300, 255},
		{500, 255},
	}

	for _, tt := range tests {
		result := generator.clampUint8(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestImageGenerator_EncodeToBase64(t *testing.T) {
	generator := NewImageGenerator()

	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	result := generator.EncodeToBase64(img)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "iVBOR")
	assert.Greater(t, len(result), 100)
}

func TestImageGenerator_DrawLine(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	generator.drawLine(img, 0, 0, 50, 50, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	count := 0
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 0 {
				count++
			}
		}
	}
	assert.Greater(t, count, 0)
}

func TestImageGenerator_DrawFilledRect(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	generator.drawFilledRect(img, 10, 10, 30, 20, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	count := 0
	for y := 10; y < 30; y++ {
		for x := 10; x < 40; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 0 {
				count++
			}
		}
	}
	assert.Equal(t, 600, count)
}

func TestImageGenerator_DrawFilledCircle(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	generator.drawFilledCircle(img, 50, 50, 10, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	count := 0
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 0 {
				count++
			}
		}
	}
	assert.Greater(t, count, 0)
}

func TestGenerateSessionID(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateSessionID()
		assert.NotEmpty(t, id)
		assert.True(t, len(id) > 10)
		assert.False(t, ids[id], "session ID should be unique")
		ids[id] = true
	}
}

func TestGeneratorService_Create(t *testing.T) {
	generator := NewGeneratorService(nil, nil)

	req := &CreateCaptchaRequest{
		Width:        320,
		Height:       160,
		SliderWidth:  40,
		SliderHeight: 40,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.Contains(t, resp.BackgroundURL, "data:image/png;base64,")
	assert.Contains(t, resp.SliderURL, "data:image/png;base64,")
	assert.Greater(t, resp.GapX, 0)
	assert.Greater(t, resp.GapY, 0)
	assert.Equal(t, int64(300), resp.ExpiresIn)
}

func TestGeneratorService_Create_CustomDimensions(t *testing.T) {
	generator := NewGeneratorService(nil, nil)

	req := &CreateCaptchaRequest{
		Width:        400,
		Height:       200,
		SliderWidth:  50,
		SliderHeight: 50,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGeneratorService_Create_WithoutDimensions(t *testing.T) {
	generator := NewGeneratorService(nil, nil)

	req := &CreateCaptchaRequest{}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGeneratorService_GetSession_NotFound(t *testing.T) {
	generator := NewGeneratorService(nil, nil)

	session, err := generator.GetSession(context.Background(), "nonexistent-session")
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestGeneratorService_DeleteSession(t *testing.T) {
	generator := NewGeneratorService(nil, nil)

	req := &CreateCaptchaRequest{}
	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	err = generator.DeleteSession(context.Background(), resp.SessionID)
	assert.NoError(t, err)
}

func TestVerifierService_Verify_ExpiredSession(t *testing.T) {
	verifierService := &mockVerifierService{
		session: &models.CaptchaSession{
			SessionID:   "test-expired",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(-1 * time.Hour),
			GapX:        100,
			GapY:        50,
		},
	}

	req := &VerifyRequest{
		SessionID: "test-expired",
		PositionX: 100,
		PositionY: 50,
	}

	result, err := verifierService.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证码已过期", result.Message)
}

func TestVerifierService_Verify_MaxAttempts(t *testing.T) {
	verifierService := &mockVerifierService{
		session: &models.CaptchaSession{
			SessionID:   "test-max-attempts",
			Status:      "pending",
			VerifyCount: 3,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
			GapX:        100,
			GapY:        50,
		},
	}

	req := &VerifyRequest{
		SessionID: "test-max-attempts",
		PositionX: 100,
		PositionY: 50,
	}

	result, err := verifierService.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证次数已用完", result.Message)
}

func TestVerifierService_Verify_AlreadyVerified(t *testing.T) {
	verifierService := &mockVerifierService{
		session: &models.CaptchaSession{
			SessionID:   "test-verified",
			Status:      "verified",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
			GapX:        100,
			GapY:        50,
		},
	}

	req := &VerifyRequest{
		SessionID: "test-verified",
		PositionX: 100,
		PositionY: 50,
	}

	result, err := verifierService.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "验证码已验证通过", result.Message)
	assert.Equal(t, float64(100), result.Score)
}

func TestVerifierService_Verify_CorrectPosition(t *testing.T) {
	verifierService := &mockVerifierService{
		session: &models.CaptchaSession{
			SessionID:   "test-correct",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
			GapX:        100,
			GapY:        50,
		},
	}

	req := &VerifyRequest{
		SessionID:  "test-correct",
		PositionX:  100,
		PositionY:  50,
		RiskScore:  80,
		TraceScore: 85,
		EnvScore:   90,
	}

	result, err := verifierService.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "验证成功", result.Message)
	assert.Equal(t, float64(100), result.Score)
	assert.Equal(t, 0, result.PositionDiff)
}

func TestVerifierService_Verify_WrongPosition(t *testing.T) {
	verifierService := &mockVerifierService{
		session: &models.CaptchaSession{
			SessionID:   "test-wrong",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
			GapX:        100,
			GapY:        50,
		},
	}

	req := &VerifyRequest{
		SessionID: "test-wrong",
		PositionX: 50,
		PositionY: 25,
	}

	result, err := verifierService.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证失败", result.Message)
	assert.Less(t, result.Score, float64(100))
	assert.Greater(t, result.PositionDiff, 0)
}

func TestVerifierService_CheckSessionValid(t *testing.T) {
	verifierService := &mockVerifierService{
		session: &models.CaptchaSession{
			SessionID:   "test-valid",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
		},
	}

	valid, msg := verifierService.CheckSessionValid(context.Background(), "test-valid")
	assert.True(t, valid)
	assert.Empty(t, msg)
}

func TestVerifierService_CheckSessionValid_Expired(t *testing.T) {
	verifierService := &mockVerifierService{
		session: &models.CaptchaSession{
			SessionID:   "test-expired",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(-1 * time.Hour),
		},
	}

	valid, msg := verifierService.CheckSessionValid(context.Background(), "test-expired")
	assert.False(t, valid)
	assert.Equal(t, "验证码已过期", msg)
}

func TestCalculatePartialScore(t *testing.T) {
	tests := []struct {
		name     string
		diffX    int
		diffY    int
		minScore float64
		maxScore float64
	}{
		{"exact match", 0, 0, 100, 100},
		{"small diff", 5, 5, 90, 100},
		{"medium diff", 20, 20, 50, 80},
		{"large diff", 50, 50, 0, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculatePartialScore(tt.diffX, tt.diffY)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{100, 100},
		{-100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

type mockVerifierService struct {
	session *models.CaptchaSession
}

func (m *mockVerifierService) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	session := m.session

	if time.Now().After(session.ExpiredAt) {
		return &VerifyResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	if session.Status == "verified" {
		return &VerifyResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	diffX := abs(session.GapX - req.PositionX)
	diffY := abs(session.GapY - req.PositionY)

	tolerance := 5
	if diffX <= tolerance && diffY <= tolerance {
		return &VerifyResult{
			Success:      true,
			Message:      "验证成功",
			Score:        100,
			PositionDiff: diffX,
		}, nil
	}

	score := calculatePartialScore(diffX, diffY)

	return &VerifyResult{
		Success:      false,
		Message:      "验证失败",
		Score:        score,
		PositionDiff: diffX,
	}, nil
}

func (m *mockVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session := m.session

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

func TestGenerateRandomDigits(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"default length (4)", 4},
		{"short length (2)", 2},
		{"long length (6)", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digits := generateRandomDigits(tt.length)
			assert.NotEmpty(t, digits)
			assert.Equal(t, tt.length, len(digits))
			for _, c := range digits {
				assert.GreaterOrEqual(t, c, '0')
				assert.LessOrEqual(t, c, '9')
			}
		})
	}
}

func TestCreateWAVHeader(t *testing.T) {
	header := createWAVHeader()
	assert.NotNil(t, header)
	assert.Greater(t, len(header), 0)
	assert.Contains(t, string(header), "RIFF")
	assert.Contains(t, string(header), "WAVE")
}

func TestGenerateVoiceAudio(t *testing.T) {
	audio := generateVoiceAudio("1234", "zh-CN")
	assert.NotNil(t, audio)
	assert.Greater(t, len(audio), 100)
}

func TestGenerateDigitWave(t *testing.T) {
	wave := generateDigitWave(5, "zh-CN", 44100)
	assert.NotNil(t, wave)
	assert.Greater(t, len(wave), 0)
}

func TestGenerateSilence(t *testing.T) {
	silence := generateSilence(100, 44100)
	assert.NotNil(t, silence)
	assert.Greater(t, len(silence), 0)
}

func TestVoiceMathSin(t *testing.T) {
	tests := []struct {
		name   string
		input  float64
		minVal float64
		maxVal float64
	}{
		{"zero", 0, -1, 1},
		{"pi/2", 3.14159 / 2, -1, 1},
		{"pi", 3.14159, -1, 1},
		{"2pi", 2 * 3.14159, -1, 1},
		{"negative", -3.14159, -1, 1},
		{"large", 100, -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := voiceMathSin(tt.input)
			assert.GreaterOrEqual(t, result, tt.minVal)
			assert.LessOrEqual(t, result, tt.maxVal)
		})
	}
}

func TestVoiceGeneratorService_Generate(t *testing.T) {
	generator := NewVoiceGeneratorService(nil, nil)

	req := &VoiceCaptchaRequest{
		Language: "zh-CN",
		Length:   4,
	}

	resp, err := generator.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotEmpty(t, resp.VoiceData)
	assert.Equal(t, "zh-CN", resp.Language)
	assert.Equal(t, int64(300), resp.ExpiresIn)
}

func TestVoiceGeneratorService_Generate_English(t *testing.T) {
	generator := NewVoiceGeneratorService(nil, nil)

	req := &VoiceCaptchaRequest{
		Language: "en-US",
		Length:   6,
	}

	resp, err := generator.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "en-US", resp.Language)
}

func TestVoiceGeneratorService_Generate_DefaultParams(t *testing.T) {
	generator := NewVoiceGeneratorService(nil, nil)

	req := &VoiceCaptchaRequest{}

	resp, err := generator.Generate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "zh-CN", resp.Language)
}

type mockVoiceVerifierService struct {
	session *models.VoiceCaptchaSession
}

func (m *mockVoiceVerifierService) Verify(ctx context.Context, req *VoiceVerifyRequest) (*VoiceVerifyResult, error) {
	session := m.session

	if time.Now().After(session.ExpiredAt) {
		return &VoiceVerifyResult{
			Success: false,
			Message: "验证码已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VoiceVerifyResult{
			Success: false,
			Message: "验证次数已用完",
		}, nil
	}

	if session.Status == "verified" {
		return &VoiceVerifyResult{
			Success: true,
			Message: "验证码已验证通过",
		}, nil
	}

	if session.Code == req.Code {
		return &VoiceVerifyResult{
			Success: true,
			Message: "验证成功",
		}, nil
	}

	return &VoiceVerifyResult{
		Success: false,
		Message: "验证码错误",
	}, nil
}

func TestVoiceVerifierService_Verify_CorrectCode(t *testing.T) {
	verifier := &mockVoiceVerifierService{
		session: &models.VoiceCaptchaSession{
			SessionID:   "test-correct",
			Code:        "1234",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
		},
	}

	req := &VoiceVerifyRequest{
		SessionID: "test-correct",
		Code:      "1234",
	}

	result, err := verifier.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "验证成功", result.Message)
}

func TestVoiceVerifierService_Verify_WrongCode(t *testing.T) {
	verifier := &mockVoiceVerifierService{
		session: &models.VoiceCaptchaSession{
			SessionID:   "test-wrong",
			Code:        "1234",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
		},
	}

	req := &VoiceVerifyRequest{
		SessionID: "test-wrong",
		Code:      "4321",
	}

	result, err := verifier.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证码错误", result.Message)
}

func TestVoiceVerifierService_Verify_ExpiredSession(t *testing.T) {
	verifier := &mockVoiceVerifierService{
		session: &models.VoiceCaptchaSession{
			SessionID:   "test-expired",
			Code:        "1234",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(-1 * time.Hour),
		},
	}

	req := &VoiceVerifyRequest{
		SessionID: "test-expired",
		Code:      "1234",
	}

	result, err := verifier.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证码已过期", result.Message)
}

func TestVoiceVerifierService_Verify_MaxAttempts(t *testing.T) {
	verifier := &mockVoiceVerifierService{
		session: &models.VoiceCaptchaSession{
			SessionID:   "test-max-attempts",
			Code:        "1234",
			Status:      "pending",
			VerifyCount: 3,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
		},
	}

	req := &VoiceVerifyRequest{
		SessionID: "test-max-attempts",
		Code:      "1234",
	}

	result, err := verifier.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证次数已用完", result.Message)
}

func TestVoiceVerifierService_Verify_AlreadyVerified(t *testing.T) {
	verifier := &mockVoiceVerifierService{
		session: &models.VoiceCaptchaSession{
			SessionID:   "test-verified",
			Code:        "1234",
			Status:      "verified",
			VerifyCount: 0,
			MaxAttempts: 3,
			ExpiredAt:   time.Now().Add(5 * time.Minute),
		},
	}

	req := &VoiceVerifyRequest{
		SessionID: "test-verified",
		Code:      "1234",
	}

	result, err := verifier.Verify(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "验证码已验证通过", result.Message)
}
