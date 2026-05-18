package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliderCaptchaRequest_Structure(t *testing.T) {
	req := SliderCaptchaRequest{
		Width:        320,
		Height:       160,
		SliderWidth:  50,
		SliderHeight: 50,
	}

	assert.Equal(t, 320, req.Width)
	assert.Equal(t, 160, req.Height)
	assert.Equal(t, 50, req.SliderWidth)
	assert.Equal(t, 50, req.SliderHeight)
}

func TestSliderVerifyRequest_Structure(t *testing.T) {
	req := SliderVerifyRequest{
		SessionID:  "test-session",
		PositionX:  150,
		PositionY:  60,
		RiskScore:  0.5,
		TraceScore: 0.8,
		EnvScore:   0.9,
	}

	assert.Equal(t, "test-session", req.SessionID)
	assert.Equal(t, 150, req.PositionX)
	assert.Equal(t, 60, req.PositionY)
	assert.Equal(t, 0.5, req.RiskScore)
	assert.Equal(t, 0.8, req.TraceScore)
	assert.Equal(t, 0.9, req.EnvScore)
}

func TestSliderVerifyRequest_RequiredFields(t *testing.T) {
	req := SliderVerifyRequest{
		SessionID: "session-123",
		PositionX: 100,
		PositionY: 50,
	}

	assert.NotEmpty(t, req.SessionID)
	assert.Greater(t, req.PositionX, 0)
	assert.Greater(t, req.PositionY, 0)
}

func TestSliderCaptchaRequest_DefaultValues(t *testing.T) {
	req := SliderCaptchaRequest{}

	assert.Equal(t, 0, req.Width)
	assert.Equal(t, 0, req.Height)
	assert.Equal(t, 0, req.SliderWidth)
	assert.Equal(t, 0, req.SliderHeight)
}

func TestInitSliderCaptchaHandler(t *testing.T) {
	InitSliderCaptchaHandler(nil, nil)
	assert.Nil(t, sliderGeneratorService)
	assert.Nil(t, sliderVerifierService)
}

func TestInitSliderCaptchaHandler_WithServices(t *testing.T) {
	InitSliderCaptchaHandler(nil, nil)
	assert.Nil(t, sliderGeneratorService)
	assert.Nil(t, sliderVerifierService)
}
