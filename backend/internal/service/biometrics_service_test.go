package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBiometricsService_RegisterProfile(t *testing.T) {
	service := NewBiometricsService()

	t.Run("register with empty userID should fail", func(t *testing.T) {
		profile, err := service.RegisterProfile("", nil, nil)
		assert.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "user ID is required")
	})

	t.Run("register new user with keyboard sample", func(t *testing.T) {
		sample := &KeyboardSample{
			KeyEvents: []KeyEvent{
				{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
				{Key: "a", Type: "keyup", Timestamp: 1050, KeyCode: 65},
				{Key: "b", Type: "keydown", Timestamp: 1100, KeyCode: 66},
				{Key: "b", Type: "keyup", Timestamp: 1150, KeyCode: 66},
				{Key: "c", Type: "keydown", Timestamp: 1200, KeyCode: 67},
				{Key: "c", Type: "keyup", Timestamp: 1250, KeyCode: 67},
			},
			Timestamp: 1000,
		}

		profile, err := service.RegisterProfile("user1", sample, nil)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, "user1", profile.UserID)
		assert.Equal(t, 1, profile.VerificationCount)
		assert.Greater(t, profile.ConfidenceScore, 0.0)
	})

	t.Run("register same user twice should update profile", func(t *testing.T) {
		sample := &KeyboardSample{
			KeyEvents: []KeyEvent{
				{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
				{Key: "a", Type: "keyup", Timestamp: 1050, KeyCode: 65},
				{Key: "b", Type: "keydown", Timestamp: 1100, KeyCode: 66},
				{Key: "b", Type: "keyup", Timestamp: 1150, KeyCode: 66},
				{Key: "c", Type: "keydown", Timestamp: 1200, KeyCode: 67},
				{Key: "c", Type: "keyup", Timestamp: 1250, KeyCode: 67},
				{Key: "d", Type: "keydown", Timestamp: 1300, KeyCode: 68},
				{Key: "d", Type: "keyup", Timestamp: 1350, KeyCode: 68},
			},
			Timestamp: 1000,
		}

		profile1, _ := service.RegisterProfile("user2", sample, nil)
		profile2, _ := service.RegisterProfile("user2", sample, nil)

		assert.Equal(t, 2, profile2.VerificationCount)
		assert.Greater(t, profile2.ConfidenceScore, profile1.ConfidenceScore)
	})
}

func TestBiometricsService_Verify(t *testing.T) {
	service := NewBiometricsService()

	t.Run("verify non-existent user should return no profile", func(t *testing.T) {
		result, err := service.Verify("nonexistent", nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsVerified)
		assert.Equal(t, 0, result.Confidence)
		assert.Contains(t, result.Details, "No profile found")
	})

	t.Run("verify with keyboard sample", func(t *testing.T) {
		sample := &KeyboardSample{
			KeyEvents: []KeyEvent{
				{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
				{Key: "a", Type: "keyup", Timestamp: 1050, KeyCode: 65},
				{Key: "b", Type: "keydown", Timestamp: 1100, KeyCode: 66},
				{Key: "b", Type: "keyup", Timestamp: 1150, KeyCode: 66},
				{Key: "c", Type: "keydown", Timestamp: 1200, KeyCode: 67},
				{Key: "c", Type: "keyup", Timestamp: 1250, KeyCode: 67},
				{Key: "d", Type: "keydown", Timestamp: 1300, KeyCode: 68},
				{Key: "d", Type: "keyup", Timestamp: 1350, KeyCode: 68},
			},
			Timestamp: 1000,
		}

		service.RegisterProfile("user3", sample, nil)
		result, err := service.Verify("user3", sample, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Confidence, 0.5)
	})
}

func TestBiometricsService_extractKeyboardFeatures(t *testing.T) {
	service := NewBiometricsService()

	t.Run("extract features with insufficient events", func(t *testing.T) {
		sample := &KeyboardSample{
			KeyEvents: []KeyEvent{
				{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
				{Key: "a", Type: "keyup", Timestamp: 1050, KeyCode: 65},
			},
		}

		features := service.extractKeyboardFeatures(sample)
		assert.Equal(t, float64(0), features.AverageHoldTime)
	})

	t.Run("extract features with valid events", func(t *testing.T) {
		sample := &KeyboardSample{
			KeyEvents: []KeyEvent{
				{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
				{Key: "a", Type: "keyup", Timestamp: 1050, KeyCode: 65},
				{Key: "b", Type: "keydown", Timestamp: 1100, KeyCode: 66},
				{Key: "b", Type: "keyup", Timestamp: 1150, KeyCode: 66},
				{Key: "c", Type: "keydown", Timestamp: 1200, KeyCode: 67},
				{Key: "c", Type: "keyup", Timestamp: 1250, KeyCode: 67},
			},
		}

		features := service.extractKeyboardFeatures(sample)
		assert.Greater(t, features.AverageHoldTime, float64(0))
		assert.Greater(t, features.TypingSpeed, float64(0))
	})
}

func TestBiometricsService_extractMouseFeatures(t *testing.T) {
	service := NewBiometricsService()

	t.Run("extract features with insufficient events", func(t *testing.T) {
		sample := &MouseSample{
			MouseEvents: []MouseEvent{
				{Type: "mousemove", X: 100, Y: 100, Timestamp: 1000},
				{Type: "mousemove", X: 101, Y: 101, Timestamp: 1001},
			},
		}

		features := service.extractMouseFeatures(sample)
		assert.Equal(t, float64(0), features.AverageSpeed)
	})

	t.Run("extract features with valid events", func(t *testing.T) {
		sample := &MouseSample{
			MouseEvents: []MouseEvent{
				{Type: "mousemove", X: 100, Y: 100, Timestamp: 1000},
				{Type: "mousemove", X: 110, Y: 110, Timestamp: 1010},
				{Type: "mousemove", X: 120, Y: 120, Timestamp: 1020},
				{Type: "mousemove", X: 130, Y: 130, Timestamp: 1030},
				{Type: "click", X: 130, Y: 130, Timestamp: 1040},
			},
		}

		features := service.extractMouseFeatures(sample)
		assert.Greater(t, features.AverageSpeed, float64(0))
	})
}

func TestBiometricsService_calculateSimilarityScore(t *testing.T) {
	service := NewBiometricsService()

	t.Run("identical values should return 1.0", func(t *testing.T) {
		score := service.calculateSimilarityScore(100, 100, 0.3)
		assert.Equal(t, 1.0, score)
	})

	t.Run("zero or negative values should return 0.5", func(t *testing.T) {
		score := service.calculateSimilarityScore(0, 100, 0.3)
		assert.Equal(t, 0.5, score)
		score = service.calculateSimilarityScore(100, 0, 0.3)
		assert.Equal(t, 0.5, score)
	})

	t.Run("values within max diff ratio", func(t *testing.T) {
		score := service.calculateSimilarityScore(100, 110, 0.3)
		assert.Greater(t, score, 0.0)
		assert.Less(t, score, 1.0)
	})
}

func TestBiometricsService_compareKeyPairTimings(t *testing.T) {
	service := NewBiometricsService()

	t.Run("empty maps should return 0.5", func(t *testing.T) {
		score := service.compareKeyPairTimings(map[string]float64{}, map[string]float64{})
		assert.Equal(t, 0.5, score)
	})

	t.Run("matching pairs should return high score", func(t *testing.T) {
		pairs1 := map[string]float64{"a→b": 100, "b→c": 100}
		pairs2 := map[string]float64{"a→b": 105, "b→c": 95}
		score := service.compareKeyPairTimings(pairs1, pairs2)
		assert.Greater(t, score, 0.5)
	})
}

func TestBiometricsService_calculateCurvature(t *testing.T) {
	service := NewBiometricsService()

	t.Run("straight line should return 0", func(t *testing.T) {
		p1 := MouseEvent{X: 0, Y: 0, Timestamp: 1000}
		p2 := MouseEvent{X: 10, Y: 0, Timestamp: 1010}
		p3 := MouseEvent{X: 20, Y: 0, Timestamp: 1020}
		curvature := service.calculateCurvature(p1, p2, p3)
		assert.Equal(t, 0.0, curvature)
	})
}

func TestBiometricsService_calculateMotionEntropy(t *testing.T) {
	service := NewBiometricsService()

	t.Run("insufficient events should return 0", func(t *testing.T) {
		events := []MouseEvent{{X: 100, Y: 100, Timestamp: 1000}}
		entropy := service.calculateMotionEntropy(events)
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("valid events should return positive entropy", func(t *testing.T) {
		events := []MouseEvent{
			{X: 100, Y: 100, Timestamp: 1000},
			{X: 200, Y: 200, Timestamp: 1010},
			{X: 300, Y: 300, Timestamp: 1020},
		}
		entropy := service.calculateMotionEntropy(events)
		assert.Greater(t, entropy, 0.0)
	})
}

func TestBiometricProfile_SerializeProfile(t *testing.T) {
	profile := &BiometricProfile{
		UserID:    "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		KeyboardProfile: KeyboardBiometrics{
			AverageHoldTime:   50.0,
			AverageFlightTime: 100.0,
		},
		VerificationCount: 5,
		ConfidenceScore:   0.5,
	}

	data, err := profile.SerializeProfile()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var deserialized BiometricProfile
	err = json.Unmarshal(data, &deserialized)
	assert.NoError(t, err)
	assert.Equal(t, profile.UserID, deserialized.UserID)
}

func TestBiometricsService_DeserializeProfile(t *testing.T) {
	service := NewBiometricsService()

	t.Run("valid data should deserialize", func(t *testing.T) {
		data := []byte(`{"user_id":"test","verification_count":3}`)
		profile, err := service.DeserializeProfile(data)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, "test", profile.UserID)
		assert.Equal(t, 3, profile.VerificationCount)
	})

	t.Run("invalid data should fail", func(t *testing.T) {
		data := []byte(`{invalid json}`)
		profile, err := service.DeserializeProfile(data)
		assert.Error(t, err)
		assert.Nil(t, profile)
	})
}

func TestMean(t *testing.T) {
	t.Run("empty slice should return 0", func(t *testing.T) {
		result := mean([]float64{})
		assert.Equal(t, 0.0, result)
	})

	t.Run("single value", func(t *testing.T) {
		result := mean([]float64{5.0})
		assert.Equal(t, 5.0, result)
	})

	t.Run("multiple values", func(t *testing.T) {
		result := mean([]float64{1.0, 2.0, 3.0, 4.0, 5.0})
		assert.Equal(t, 3.0, result)
	})
}

func TestStdDev(t *testing.T) {
	t.Run("insufficient values should return 0", func(t *testing.T) {
		result := stdDev([]float64{5.0})
		assert.Equal(t, 0.0, result)
	})

	t.Run("standard deviation calculation", func(t *testing.T) {
		result := stdDev([]float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0})
		assert.Greater(t, result, 0.0)
	})
}
