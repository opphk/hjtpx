package captcha

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVRARGeneratorService(t *testing.T) {
	service := NewVRARGeneratorService(nil, nil)
	assert.NotNil(t, service)
}

func TestNewVRARVerifierService(t *testing.T) {
	service := NewVRARVerifierService(nil, nil)
	assert.NotNil(t, service)
}

func TestVRARModeConstants(t *testing.T) {
	assert.Equal(t, VRARMode("vr"), VRARModeVR)
	assert.Equal(t, VRARMode("ar"), VRARModeAR)
	assert.Equal(t, VRARMode("hybrid"), VRARModeHybrid)
	assert.Equal(t, VRARMode("interactive"), VRARModeInteractive)
}

func TestVRARTypeConstants(t *testing.T) {
	assert.Equal(t, VRARType("3d_placement"), VRARType3DPlacement)
	assert.Equal(t, VRARType("gesture"), VRARTypeGesture)
	assert.Equal(t, VRARType("eye_tracking"), VRARTypeEyeTracking)
	assert.Equal(t, VRARType("object_rotation"), VRARTypeObjectRotation)
	assert.Equal(t, VRARType("spatial_puzzle"), VRARTypeSpatialPuzzle)
	assert.Equal(t, VRARType("sequential"), VRARTypeSequential)
}

func TestGetObjectCountByDifficulty(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()

	tests := []struct {
		difficulty string
		expected   int
	}{
		{"easy", 2},
		{"medium", 3},
		{"hard", 4},
		{"expert", 5},
		{"unknown", 3},
		{"", 3},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := service.getObjectCountByDifficulty(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetToleranceByDifficulty(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()

	tests := []struct {
		difficulty string
		expected   float64
	}{
		{"easy", 0.3},
		{"medium", 0.2},
		{"hard", 0.15},
		{"expert", 0.1},
		{"unknown", 0.2},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := service.getToleranceByDifficulty(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAngleToleranceByDifficulty(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()

	tests := []struct {
		difficulty string
		expected   float64
	}{
		{"easy", 30.0},
		{"medium", 20.0},
		{"hard", 15.0},
		{"expert", 10.0},
		{"unknown", 20.0},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := service.getAngleToleranceByDifficulty(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateVRARSessionID(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()
	sessionID := service.generateSessionID()
	assert.NotEmpty(t, sessionID)
	assert.Contains(t, sessionID, "vrar_")

	sessionID2 := service.generateSessionID()
	assert.NotEmpty(t, sessionID2)
	assert.NotEqual(t, sessionID, sessionID2)
}

func TestVRARCaptchaRequestStructure(t *testing.T) {
	req := &VRARCaptchaRequest{
		Mode:        VRARModeVR,
		Type:        VRARType3DPlacement,
		Difficulty:  "medium",
		ClientIP:    "127.0.0.1",
		UserAgent:   "Test User Agent",
		Fingerprint: "test-fingerprint",
	}

	assert.Equal(t, VRARModeVR, req.Mode)
	assert.Equal(t, VRARType3DPlacement, req.Type)
	assert.Equal(t, "medium", req.Difficulty)
	assert.Equal(t, "127.0.0.1", req.ClientIP)
}

func TestVRARVerifyRequestStructure(t *testing.T) {
	req := &VRARVerifyRequest{
		SessionID: "test-session-id",
		Interaction: &VRInteractionData{
			ObjectPositions: map[string][]float64{
				"obj1": {1.0, 2.0, 3.0},
			},
			TimeSpent: 10.5,
		},
	}

	assert.Equal(t, "test-session-id", req.SessionID)
	assert.NotNil(t, req.Interaction)
	assert.Len(t, req.Interaction.ObjectPositions, 1)
}

func TestVRARSceneConfigStructure(t *testing.T) {
	config := &VRARSceneConfig{
		Mode:        VRARModeVR,
		Type:        VRARType3DPlacement,
		Environment: "simple_room",
		Objects: []*VRObject{
			{
				ID:       "obj1",
				Type:     "cube",
				Position: []float64{0, 0, 0},
			},
		},
		Physics: true,
	}

	assert.Equal(t, VRARModeVR, config.Mode)
	assert.Equal(t, VRARType3DPlacement, config.Type)
	assert.Len(t, config.Objects, 1)
	assert.True(t, config.Physics)
}

func TestVRARSessionStructure(t *testing.T) {
	session := &VRARSession{
		SessionID:     "test-session",
		Mode:          VRARModeVR,
		Type:          VRARTypeGesture,
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
	}

	assert.Equal(t, "test-session", session.SessionID)
	assert.Equal(t, VRARModeVR, session.Mode)
	assert.Equal(t, VRARTypeGesture, session.Type)
	assert.Equal(t, "pending", session.Status)
	assert.Equal(t, 0, session.VerifyCount)
	assert.Equal(t, 3, session.MaxAttempts)
}

func TestSelectDefaultMode(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()
	mode := service.selectDefaultMode()
	assert.NotEmpty(t, mode)
	assert.Contains(t, []VRARMode{VRARModeVR, VRARModeAR, VRARModeHybrid}, mode)
}

func TestSelectDefaultType(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()
	
	vrType := service.selectDefaultType(VRARModeVR)
	assert.Equal(t, VRARType3DPlacement, vrType)
	
	arType := service.selectDefaultType(VRARModeAR)
	assert.Equal(t, VRARTypeObjectRotation, arType)
}

func TestExtractCorrectActions(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()
	
	config := &VRARSceneConfig{
		Objects: []*VRObject{
			{
				ID:             "obj1",
				TargetPosition: []float64{1, 0, 0},
			},
		},
		TargetGesture: "pinch",
	}
	
	actions := service.extractCorrectActions(config)
	assert.NotEmpty(t, actions)
	assert.Contains(t, actions, "place:obj1")
	assert.Contains(t, actions, "gesture:pinch")
}

func TestGenerateSimple(t *testing.T) {
	service := NewVRARGeneratorServiceSimple()
	
	req := &VRARCaptchaRequest{
		Mode:       VRARModeVR,
		Type:       VRARType3DPlacement,
		Difficulty: "easy",
	}
	
	result, err := service.Generate(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.SessionID)
	assert.NotNil(t, result.SceneConfig)
	assert.NotEmpty(t, result.Instructions)
}

func TestCalculateDistance3DVR(t *testing.T) {
	tests := []struct {
		name     string
		pos1     []float64
		pos2     []float64
		expected float64
	}{
		{"zero distance", []float64{0, 0, 0}, []float64{0, 0, 0}, 0},
		{"x axis", []float64{0, 0, 0}, []float64{3, 0, 0}, 3},
		{"diagonal", []float64{0, 0, 0}, []float64{1, 1, 1}, 1.732},
		{"invalid input", []float64{0}, []float64{0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDistance3D_vr(tt.pos1, tt.pos2)
			if tt.expected == 0 {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.InDelta(t, tt.expected, result, 0.01)
			}
		})
	}
}

func TestCalculateAngleDifferenceVR(t *testing.T) {
	tests := []struct {
		name     string
		rot1     []float64
		rot2     []float64
	}{
		{"zero diff", []float64{0, 0, 0}, []float64{0, 0, 0}},
		{"small diff", []float64{0, 10, 0}, []float64{0, 0, 0}},
		{"180 diff", []float64{0, 0, 0}, []float64{0, 180, 0}},
		{"over 180", []float64{0, 0, 0}, []float64{0, 200, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAngleDifference_vr(tt.rot1, tt.rot2)
			assert.GreaterOrEqual(t, result, 0.0)
		})
	}
}
