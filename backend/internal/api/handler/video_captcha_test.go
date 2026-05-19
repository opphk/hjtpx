package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVideoCaptchaGenerateRequest_DefaultValues(t *testing.T) {
	req := VideoCaptchaGenerateRequest{}

	assert.Equal(t, 0, req.Width)
	assert.Equal(t, 0, req.Height)
	assert.Equal(t, 0, req.Difficulty)
}

func TestVideoCaptchaGenerateRequest_WithValues(t *testing.T) {
	req := VideoCaptchaGenerateRequest{
		Width:      640,
		Height:     360,
		Difficulty: 2,
	}

	assert.Equal(t, 640, req.Width)
	assert.Equal(t, 360, req.Height)
	assert.Equal(t, 2, req.Difficulty)
}

func TestVideoCaptchaVerifyRequest_Structure(t *testing.T) {
	req := VideoCaptchaVerifyRequest{
		SessionID: "test-session",
		Answer:    "举手",
		BehaviorData: struct {
			StartTime       int64  `json:"start_time"`
			EndTime         int64  `json:"end_time"`
			Duration        int64  `json:"duration"`
			ViewCount       int    `json:"view_count"`
			ReplayCount     int    `json:"replay_count"`
			IsMobile        bool   `json:"is_mobile"`
			DeviceType      string `json:"device_type"`
			NetworkType     string `json:"network_type"`
			Latency         int    `json:"latency"`
			ClickCount      int    `json:"click_count"`
			AnswerTime      int64  `json:"answer_time"`
		}{
			StartTime:   1000,
			EndTime:     9000,
			Duration:    8000,
			ViewCount:   1,
			ReplayCount: 0,
		},
	}

	assert.Equal(t, "test-session", req.SessionID)
	assert.Equal(t, "举手", req.Answer)
	assert.Equal(t, int64(1000), req.BehaviorData.StartTime)
	assert.Equal(t, int64(9000), req.BehaviorData.EndTime)
	assert.Equal(t, int64(8000), req.BehaviorData.Duration)
	assert.Equal(t, 1, req.BehaviorData.ViewCount)
	assert.Equal(t, 0, req.BehaviorData.ReplayCount)
}

func TestInitVideoCaptchaHandler_NilServices(t *testing.T) {
	InitVideoCaptchaHandler(nil, nil)
	assert.Nil(t, videoGeneratorService)
	assert.Nil(t, videoVerifierService)
}

func TestInitVideoCaptchaHandler_WithServices(t *testing.T) {
	InitVideoCaptchaHandler(nil, nil)
	assert.Nil(t, videoGeneratorService)
	assert.Nil(t, videoVerifierService)
}

func TestVideoCaptchaOptions_Response(t *testing.T) {
}
