package service

import (
	"testing"
)

func TestMultimodalServiceGetConfig(t *testing.T) {
	service := NewMultimodalService()

	config := service.GetConfig()
	if config == nil {
		t.Error("GetConfig() returned nil")
	}

	if config.ConfidenceThreshold != 0.85 {
		t.Errorf("GetConfig() ConfidenceThreshold = %f, want 0.85", config.ConfidenceThreshold)
	}
}

func TestMultimodalServiceIsModalityEnabled(t *testing.T) {
	service := NewMultimodalService()

	tests := []struct {
		name     string
		modality ModalityType
		want     bool
	}{
		{"Visual modality", ModalityVisual, true},
		{"Voice modality", ModalityVoice, true},
		{"Gesture modality", ModalityGesture, true},
		{"Touch modality", ModalityTouch, true},
		{"AR modality", ModalityAR, true},
		{"Biometric modality", ModalityBiometric, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.IsModalityEnabled(tt.modality); got != tt.want {
				t.Errorf("IsModalityEnabled(%s) = %v, want %v", tt.modality, got, tt.want)
			}
		})
	}
}

func TestMultimodalServiceGenerateVoiceChallenge(t *testing.T) {
	service := NewMultimodalService()

	tests := []struct {
		name      string
		difficulty int
	}{
		{"Easy difficulty", 1},
		{"Medium difficulty", 2},
		{"Hard difficulty", 3},
		{"Expert difficulty", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := service.GenerateVoiceChallenge(tt.difficulty)
			if err != nil {
				t.Errorf("GenerateVoiceChallenge() error = %v", err)
				return
			}

			if challenge == nil {
				t.Error("GenerateVoiceChallenge() returned nil")
				return
			}

			if challenge.ID == "" {
				t.Error("GenerateVoiceChallenge() returned empty ID")
			}

			if challenge.Text == "" {
				t.Error("GenerateVoiceChallenge() returned empty Text")
			}

			if challenge.Difficulty != tt.difficulty {
				t.Errorf("GenerateVoiceChallenge() Difficulty = %d, want %d", challenge.Difficulty, tt.difficulty)
			}
		})
	}
}

func TestMultimodalServiceGenerateGestureChallenge(t *testing.T) {
	service := NewMultimodalService()

	tests := []struct {
		name      string
		difficulty int
	}{
		{"Easy difficulty", 1},
		{"Hard difficulty", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := service.GenerateGestureChallenge(tt.difficulty)
			if err != nil {
				t.Errorf("GenerateGestureChallenge() error = %v", err)
				return
			}

			if challenge == nil {
				t.Error("GenerateGestureChallenge() returned nil")
				return
			}

			if len(challenge.Points) == 0 {
				t.Error("GenerateGestureChallenge() returned empty points")
			}
		})
	}
}

func TestMultimodalServiceGenerateTouchChallenge(t *testing.T) {
	service := NewMultimodalService()

	tests := []struct {
		name      string
		difficulty int
	}{
		{"Easy difficulty", 1},
		{"Hard difficulty", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := service.GenerateTouchChallenge(tt.difficulty)
			if err != nil {
				t.Errorf("GenerateTouchChallenge() error = %v", err)
				return
			}

			if challenge == nil {
				t.Error("GenerateTouchChallenge() returned nil")
				return
			}

			if len(challenge.TouchPattern) == 0 {
				t.Error("GenerateTouchChallenge() returned empty pattern")
			}
		})
	}
}

func TestMultimodalServiceGenerateARChallenge(t *testing.T) {
	service := NewMultimodalService()

	tests := []struct {
		name      string
		difficulty int
	}{
		{"Easy difficulty", 1},
		{"Hard difficulty", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := service.GenerateARChallenge(tt.difficulty)
			if err != nil {
				t.Errorf("GenerateARChallenge() error = %v", err)
				return
			}

			if challenge == nil {
				t.Error("GenerateARChallenge() returned nil")
				return
			}

			if len(challenge.Objects) == 0 {
				t.Error("GenerateARChallenge() returned empty objects")
			}
		})
	}
}

func TestMultimodalServiceCreateCrossDeviceSession(t *testing.T) {
	service := NewMultimodalService()

	session, err := service.CreateCrossDeviceSession("desktop", "mobile", ModalityVisual)
	if err != nil {
		t.Errorf("CreateCrossDeviceSession() error = %v", err)
		return
	}

	if session == nil {
		t.Error("CreateCrossDeviceSession() returned nil")
	}

	if session.PrimaryDevice != "desktop" {
		t.Errorf("CreateCrossDeviceSession() PrimaryDevice = %s, want desktop", session.PrimaryDevice)
	}

	if session.SecondaryDevice != "mobile" {
		t.Errorf("CreateCrossDeviceSession() SecondaryDevice = %s, want mobile", session.SecondaryDevice)
	}

	if session.Status != "pending" {
		t.Errorf("CreateCrossDeviceSession() Status = %s, want pending", session.Status)
	}
}

func TestMultimodalServiceVerifyVoiceResponse(t *testing.T) {
	service := NewMultimodalService()

	challenge, _ := service.GenerateVoiceChallenge(2)

	result, err := service.VerifyVoiceResponse(challenge.ID, "audio_data_here")
	if err != nil {
		t.Errorf("VerifyVoiceResponse() error = %v", err)
		return
	}

	if result == nil {
		t.Error("VerifyVoiceResponse() returned nil")
		return
	}

	if result.Modality != ModalityVoice {
		t.Errorf("VerifyVoiceResponse() Modality = %s, want %s", result.Modality, ModalityVoice)
	}
}

func TestMultimodalServiceVerifyGestureResponse(t *testing.T) {
	service := NewMultimodalService()

	challenge, _ := service.GenerateGestureChallenge(2)

	points := []GesturePoint{
		{X: 0.5, Y: 0.5, Pressure: 0.8, Timestamp: 1000},
		{X: 0.55, Y: 0.45, Pressure: 0.8, Timestamp: 1050},
		{X: 0.6, Y: 0.4, Pressure: 0.8, Timestamp: 1100},
	}

	result, err := service.VerifyGestureResponse(challenge.ID, points)
	if err != nil {
		t.Errorf("VerifyGestureResponse() error = %v", err)
		return
	}

	if result == nil {
		t.Error("VerifyGestureResponse() returned nil")
		return
	}

	if result.Modality != ModalityGesture {
		t.Errorf("VerifyGestureResponse() Modality = %s, want %s", result.Modality, ModalityGesture)
	}
}

func TestMultimodalServiceVerifyTouchResponse(t *testing.T) {
	service := NewMultimodalService()

	challenge, _ := service.GenerateTouchChallenge(2)

	points := []TouchPoint{
		{X: 0.5, Y: 0.5, Timestamp: 1000, Pressure: 0.8, Fingers: 1},
		{X: 0.55, Y: 0.45, Timestamp: 1500, Pressure: 0.8, Fingers: 1},
		{X: 0.6, Y: 0.4, Timestamp: 2000, Pressure: 0.8, Fingers: 1},
	}

	result, err := service.VerifyTouchResponse(challenge.ID, points)
	if err != nil {
		t.Errorf("VerifyTouchResponse() error = %v", err)
		return
	}

	if result == nil {
		t.Error("VerifyTouchResponse() returned nil")
		return
	}

	if result.Modality != ModalityTouch {
		t.Errorf("VerifyTouchResponse() Modality = %s, want %s", result.Modality, ModalityTouch)
	}
}

func TestMultimodalServiceVerifyARResponse(t *testing.T) {
	service := NewMultimodalService()

	challenge, _ := service.GenerateARChallenge(2)

	objects := []ARObject{
		{
			ID:       "obj_0",
			Type:     "cube",
			Position: []float64{0.3, 0, -2},
			Rotation: []float64{0, 0, 0},
			Scale:    1.0,
			Target:   true,
		},
	}

	result, err := service.VerifyARResponse(challenge.ID, objects)
	if err != nil {
		t.Errorf("VerifyARResponse() error = %v", err)
		return
	}

	if result == nil {
		t.Error("VerifyARResponse() returned nil")
		return
	}

	if result.Modality != ModalityAR {
		t.Errorf("VerifyARResponse() Modality = %s, want %s", result.Modality, ModalityAR)
	}
}

func TestMultimodalServiceExportConfig(t *testing.T) {
	service := NewMultimodalService()

	data, err := service.ExportConfig()
	if err != nil {
		t.Errorf("ExportConfig() error = %v", err)
		return
	}

	if len(data) == 0 {
		t.Error("ExportConfig() returned empty data")
	}
}
