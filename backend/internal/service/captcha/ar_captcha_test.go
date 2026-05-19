package captcha

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewARGeneratorService(t *testing.T) {
	service := NewARGeneratorService(nil, nil)
	assert.NotNil(t, service)
}

func TestNewARVerifierService(t *testing.T) {
	service := NewARVerifierService(nil, nil)
	assert.NotNil(t, service)
}

func TestGetObjectCountByDifficulty(t *testing.T) {
	tests := []struct {
		difficulty string
		expected   int
	}{
		{"easy", 3},
		{"medium", 5},
		{"hard", 7},
		{"expert", 10},
		{"unknown", 5},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			service := NewARGeneratorService(nil, nil)
			result := service.getObjectCount(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateObjects(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	difficulties := []string{"easy", "medium", "hard", "expert"}

	for _, difficulty := range difficulties {
		t.Run(difficulty, func(t *testing.T) {
			objectCount := service.getObjectCount(difficulty)
			objects := service.generateObjects(objectCount, difficulty)

			assert.NotNil(t, objects)
			assert.Len(t, objects, objectCount)

			targetCount := 0
			for _, obj := range objects {
				assert.NotEmpty(t, obj.Type)
				assert.NotEmpty(t, obj.Color)
				assert.GreaterOrEqual(t, obj.Scale, 0.0)
				if obj.IsTarget {
					targetCount++
				}
			}
			assert.Equal(t, 1, targetCount, "Should have exactly one target object")
		})
	}
}

func TestGenerateGesturePath(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	sceneTypes := []string{"gesture_recognition", "object_placement", "spatial_puzzle", "unknown"}
	difficulties := []string{"easy", "medium", "hard", "expert"}

	for _, sceneType := range sceneTypes {
		for _, difficulty := range difficulties {
			t.Run(sceneType+"_"+difficulty, func(t *testing.T) {
				path := service.generateGesturePath(sceneType, difficulty)

				assert.NotNil(t, path)
				assert.Greater(t, len(path), 0)

				// 验证点的时间戳递增
				for i := 1; i < len(path); i++ {
					assert.GreaterOrEqual(t, path[i].Timestamp, path[i-1].Timestamp)
				}
			})
		}
	}
}

func TestGenerateAnnotations(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	difficulties := []string{"easy", "medium", "hard", "expert"}
	expectedCounts := map[string]int{
		"easy":   1,
		"medium": 2,
		"hard":   3,
		"expert": 4,
	}

	for _, difficulty := range difficulties {
		t.Run(difficulty, func(t *testing.T) {
			annotations := service.generateAnnotations(difficulty)
			assert.NotNil(t, annotations)
			assert.Len(t, annotations, expectedCounts[difficulty])

			for _, annotation := range annotations {
				assert.NotEmpty(t, annotation.Type)
				assert.NotEmpty(t, annotation.Color)
				assert.GreaterOrEqual(t, annotation.PositionX, 0.0)
				assert.LessOrEqual(t, annotation.PositionX, 1.0)
			}
		})
	}
}

func TestGenerateCameraConfig(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	difficulties := []string{"easy", "medium", "hard", "expert"}
	expectedFOVs := map[string]float64{
		"easy":   50.0,
		"medium": 60.0,
		"hard":   70.0,
		"expert": 80.0,
	}

	for _, difficulty := range difficulties {
		t.Run(difficulty, func(t *testing.T) {
			config := service.generateCameraConfig(difficulty)

			assert.NotNil(t, config)
			assert.Equal(t, expectedFOVs[difficulty], config.FOV)
			assert.Greater(t, config.NearClip, 0.0)
			assert.Greater(t, config.FarClip, config.NearClip)
			assert.Equal(t, float64(5), config.PositionZ)
		})
	}
}

func TestGenerateLightingConfig(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	difficulties := []string{"easy", "medium", "hard", "expert"}

	for _, difficulty := range difficulties {
		t.Run(difficulty, func(t *testing.T) {
			config := service.generateLightingConfig(difficulty)

			assert.NotNil(t, config)
			assert.Greater(t, config.AmbientIntensity, 0.0)
			assert.Greater(t, config.DirectionalIntensity, 0.0)
			assert.Greater(t, config.PointIntensity, 0.0)
			assert.NotEmpty(t, config.AmbientColor)

			if difficulty == "hard" || difficulty == "expert" {
				assert.True(t, config.ShadowEnabled)
			}
		})
	}
}

func TestGetTimeLimit(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	tests := []struct {
		difficulty string
		expected   int
	}{
		{"easy", 30},
		{"medium", 20},
		{"hard", 15},
		{"expert", 10},
		{"unknown", 20},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := service.getTimeLimit(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSuccessCriteria(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	tests := []struct {
		sceneType string
		expected  string
	}{
		{"gesture_recognition", "accurate_gesture"},
		{"object_placement", "correct_position"},
		{"spatial_puzzle", "spatial_accuracy"},
		{"object_tracking", "tracking_completeness"},
		{"depth_estimation", "depth_accuracy"},
		{"unknown", "gesture_accuracy"},
	}

	for _, tt := range tests {
		t.Run(tt.sceneType, func(t *testing.T) {
			result := service.getSuccessCriteria(tt.sceneType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateARScene(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	sceneTypes := sceneTypes
	difficulties := []string{"easy", "medium", "hard", "expert"}

	for _, sceneType := range sceneTypes {
		for _, difficulty := range difficulties {
			t.Run(sceneType+"_"+difficulty, func(t *testing.T) {
				scene := service.generateScene(sceneType, difficulty)

				assert.NotNil(t, scene)
				assert.Equal(t, sceneType, scene.SceneType)
				assert.Equal(t, difficulty, scene.Difficulty)
				assert.NotNil(t, scene.CameraConfig)
				assert.NotNil(t, scene.LightingConfig)
				assert.Greater(t, len(scene.Objects), 0)
				assert.Greater(t, len(scene.GesturePath), 0)
				assert.Greater(t, scene.TimeLimit, 0)
				assert.NotEmpty(t, scene.SuccessCriteria)
			})
		}
	}
}

func TestCreateARSession(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &CreateARRequest{
		SceneType:  "gesture_recognition",
		Difficulty: "medium",
	}

	result, err := service.Create(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.SessionID)
	assert.NotNil(t, result.Scene)
	assert.Greater(t, result.ExpiresIn, int64(0))
	assert.Greater(t, result.ExpiresAt, int64(0))

	result2, err := service.Create(nil, req)
	assert.NoError(t, err)
	assert.NotEqual(t, result.SessionID, result2.SessionID)
}

func TestGetPassThreshold(t *testing.T) {
	verifier := NewARVerifierService(nil, nil)

	tests := []struct {
		difficulty string
		expected  float64
	}{
		{"easy", 60},
		{"medium", 70},
		{"hard", 80},
		{"expert", 85},
		{"unknown", 70},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := verifier.getPassThreshold(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGestureTypes(t *testing.T) {
	assert.Greater(t, len(gestureTypes), 0)
	expectedGestures := []string{"tap", "swipe_left", "swipe_right", "circle", "triangle"}
	for _, gesture := range expectedGestures {
		assert.Contains(t, gestureTypes, gesture)
	}
}

func TestObjectTypes(t *testing.T) {
	assert.Greater(t, len(objectTypes), 0)
	expectedObjects := []string{"cube", "sphere", "pyramid", "cylinder", "torus"}
	for _, obj := range expectedObjects {
		assert.Contains(t, objectTypes, obj)
	}
}

func TestSceneTypes(t *testing.T) {
	assert.Greater(t, len(sceneTypes), 0)
	expectedScenes := []string{"object_placement", "gesture_recognition", "spatial_puzzle"}
	for _, scene := range expectedScenes {
		assert.Contains(t, sceneTypes, scene)
	}
}

func TestCalculateGestureAmplitude(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	tests := []struct {
		difficulty string
		minExpected float64
		maxExpected float64
	}{
		{"easy", 0.2, 0.4},
		{"medium", 0.4, 0.6},
		{"hard", 0.6, 0.8},
		{"expert", 0.8, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := service.getGestureAmplitude(tt.difficulty)
			assert.GreaterOrEqual(t, result, tt.minExpected)
			assert.LessOrEqual(t, result, tt.maxExpected)
		})
	}
}

func TestGesturePointCount(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	tests := []struct {
		difficulty string
		expected   int
	}{
		{"easy", 10},
		{"medium", 20},
		{"hard", 30},
		{"expert", 40},
		{"unknown", 20},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := service.getGesturePointCount(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGesturePhase(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	tests := []struct {
		t        float64
		expected string
	}{
		{0.05, "start"},
		{0.5, "middle"},
		{0.95, "end"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := service.getGesturePhase(tt.t)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRandomHelpers(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	color := service.getRandomColor()
	assert.NotEmpty(t, color)
	assert.Equal(t, '#', rune(color[0]))

	texture := service.getRandomTexture()
	assert.NotEmpty(t, texture)

	animation := service.getRandomAnimation()
	assert.NotEmpty(t, animation)

	emissiveColor := service.getEmissiveColor()
	assert.NotEmpty(t, emissiveColor)

	bgColor := service.getBackgroundColor()
	assert.NotEmpty(t, bgColor)
}
