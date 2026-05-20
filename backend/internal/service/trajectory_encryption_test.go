package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrajectoryEncryptionService_EncryptDecrypt(t *testing.T) {
	svc := NewTrajectoryEncryptionService()

	t.Run("encrypt_decrypt_basic", func(t *testing.T) {
		originalData := []byte("test trajectory data")

		encrypted, err := svc.Encrypt(originalData)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, originalData, encrypted)

		decrypted, err := svc.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, originalData, decrypted)
	})

	t.Run("encrypt_different_each_time", func(t *testing.T) {
		data := []byte("same data")

		encrypted1, err := svc.Encrypt(data)
		require.NoError(t, err)

		encrypted2, err := svc.Encrypt(data)
		require.NoError(t, err)

		assert.NotEqual(t, encrypted1, encrypted2)
	})
}

func TestTrajectoryEncryptionService_EncryptDecryptTrajectory(t *testing.T) {
	svc := NewTrajectoryEncryptionService()

	t.Run("encrypt_decrypt_trajectory_features", func(t *testing.T) {
		features := &TrajectoryFeatures{
			TotalDistance:     250.5,
			TotalDuration:     2000,
			AvgVelocity:       350.2,
			MaxVelocity:       800.0,
			VelocityVariance:  0.25,
			PathEfficiency:   0.85,
			DirectionChanges:  5,
			MicroCorrections: 3,
			BacktrackCount:   1,
			BacktrackDistance: 20.0,
			PauseCount:       2,
			TotalPauseDuration: 150.0,
			Smoothness:       0.75,
			Jitter:           0.12,
			Entropy:          3.5,
			HumanLikeness:    0.85,
		}

		encrypted, err := svc.EncryptTrajectory(features)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.Equal(t, "2.0", encrypted.Version)
		assert.NotEmpty(t, encrypted.Encrypted)
		assert.NotEmpty(t, encrypted.Checksum)

		decrypted, err := svc.DecryptTrajectory(encrypted)
		require.NoError(t, err)
		assert.Equal(t, features.TotalDistance, decrypted.TotalDistance)
		assert.Equal(t, features.TotalDuration, decrypted.TotalDuration)
		assert.Equal(t, features.AvgVelocity, decrypted.AvgVelocity)
	})

	t.Run("checksum_validation", func(t *testing.T) {
		features := &TrajectoryFeatures{
			TotalDistance: 250.0,
			TotalDuration: 2000,
		}

		encrypted, err := svc.EncryptTrajectory(features)
		require.NoError(t, err)

		valid := svc.ValidateChecksum([]byte("wrong data"), encrypted.Checksum)
		assert.False(t, valid)

		valid = svc.ValidateChecksum([]byte("{\"total_distance\":250,\"total_duration\":2000}"), encrypted.Checksum)
		assert.True(t, valid)
	})
}

func TestTrajectoryEncryptionService_SetKey(t *testing.T) {
	svc := NewTrajectoryEncryptionService()

	t.Run("set_valid_key", func(t *testing.T) {
		err := svc.SetKey("this-is-a-test-key-that-is-long-enough")
		require.NoError(t, err)

		data := []byte("test data")
		encrypted, err := svc.Encrypt(data)
		require.NoError(t, err)

		decrypted, err := svc.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, data, decrypted)
	})

	t.Run("set_invalid_key", func(t *testing.T) {
		err := svc.SetKey("short")
		assert.Error(t, err)
	})
}

func TestTrajectoryIntegrityValidator_Validate(t *testing.T) {
	validator := NewTrajectoryIntegrityValidator()

	t.Run("validate_valid_trajectory", func(t *testing.T) {
		encrypted := &EncryptedTrajectoryData{
			Version:   "2.0",
			Timestamp: 1000000000,
			Features: &TrajectoryFeatures{
				TotalDistance: 200.0,
			},
		}

		err := validator.Validate(encrypted)
		assert.NoError(t, err)
	})

	t.Run("validate_expired_trajectory", func(t *testing.T) {
		encrypted := &EncryptedTrajectoryData{
			Version:   "2.0",
			Timestamp: 1000000000 - 10000000,
			Features: &TrajectoryFeatures{
				TotalDistance: 200.0,
			},
		}

		err := validator.Validate(encrypted)
		assert.Error(t, err)
	})

	t.Run("validate_missing_features", func(t *testing.T) {
		encrypted := &EncryptedTrajectoryData{
			Version:   "2.0",
			Timestamp: 1000000000,
			Features:  nil,
		}

		err := validator.Validate(encrypted)
		assert.Error(t, err)
	})
}

func TestTrajectoryIntegrityValidator_SetConstraints(t *testing.T) {
	validator := NewTrajectoryIntegrityValidator()

	t.Run("set_constraints", func(t *testing.T) {
		constraints := map[string]interface{}{
			"min_points":   20,
			"max_points":   500,
			"min_duration": int64(500),
			"max_duration": int64(20000),
			"min_distance": 100.0,
			"max_distance": 3000.0,
		}

		err := validator.SetConstraints(constraints)
		assert.NoError(t, err)
	})
}

func TestSecureTrajectoryProcessor_ProcessEncryptedTrajectory(t *testing.T) {
	processor := NewSecureTrajectoryProcessor()

	t.Run("process_valid_encrypted_trajectory", func(t *testing.T) {
		features := &TrajectoryFeatures{
			TotalDistance:    250.0,
			TotalDuration:    2000,
			AvgVelocity:      350.0,
			MaxVelocity:      800.0,
			PathEfficiency:  0.85,
			HumanLikeness:   0.85,
		}

		encrypted, err := processor.CreateEncryptedTrajectory(features)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)

		decrypted, err := processor.ProcessEncryptedTrajectory(encrypted.Encrypted)
		require.NoError(t, err)
		assert.Equal(t, features.TotalDistance, decrypted.TotalDistance)
	})
}

func TestAdvancedEncryptionService_GenerateSessionKey(t *testing.T) {
	svc := NewAdvancedEncryptionService()

	t.Run("generate_session_key", func(t *testing.T) {
		sessionID := "test-session-123"

		key, err := svc.GenerateSessionKey(sessionID)
		require.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("encrypt_decrypt_with_session_key", func(t *testing.T) {
		sessionID := "test-session-456"
		originalData := []byte("sensitive trajectory data")

		key, err := svc.GenerateSessionKey(sessionID)
		require.NoError(t, err)
		assert.Len(t, key, 32)

		encrypted, err := svc.EncryptWithSessionKey(sessionID, originalData)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		decrypted, err := svc.DecryptWithSessionKey(sessionID, encrypted)
		require.NoError(t, err)
		assert.Equal(t, originalData, decrypted)
	})

	t.Run("decrypt_without_key", func(t *testing.T) {
		sessionID := "non-existent-session"

		_, err := svc.DecryptWithSessionKey(sessionID, "encrypted_data")
		assert.Error(t, err)
	})
}

func TestAdvancedEncryptionService_CleanupSessionKey(t *testing.T) {
	svc := NewAdvancedEncryptionService()

	t.Run("cleanup_session_key", func(t *testing.T) {
		sessionID := "test-session-789"

		_, err := svc.GenerateSessionKey(sessionID)
		require.NoError(t, err)

		svc.CleanupSessionKey(sessionID)

		_, err = svc.DecryptWithSessionKey(sessionID, "encrypted_data")
		assert.Error(t, err)
	})
}

func TestAdvancedEncryptionService_EncryptDecryptData(t *testing.T) {
	svc := NewAdvancedEncryptionService()

	t.Run("encrypt_decrypt_complex_structure", func(t *testing.T) {
		data := map[string]interface{}{
			"total_distance":   250.5,
			"total_duration":   2000,
			"avg_velocity":    350.2,
			"max_velocity":    800.0,
			"path_efficiency": 0.85,
			"human_likeness":  0.85,
			"trajectory_points": []map[string]float64{
				{"x": 10.0, "y": 80.0, "t": 0.0},
				{"x": 20.0, "y": 80.0, "t": 20.0},
			},
		}

		encrypted, err := svc.EncryptData(data)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		var decrypted map[string]interface{}
		err = svc.DecryptData(encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, data["total_distance"], decrypted["total_distance"])
	})
}

func TestAdvancedEncryptionService_ThreadSafety(t *testing.T) {
	svc := NewAdvancedEncryptionService()

	t.Run("concurrent_encrypt_decrypt", func(t *testing.T) {
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func(id int) {
				sessionID := "concurrent-session"
				_, _ = svc.GenerateSessionKey(sessionID)

				data := []byte("concurrent test data")
				encrypted, err := svc.EncryptWithSessionKey(sessionID, data)
				if err != nil {
					done <- false
					return
				}

				decrypted, err := svc.DecryptWithSessionKey(sessionID, encrypted)
				if err != nil || string(decrypted) != string(data) {
					done <- false
					return
				}

				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			assert.True(t, <-done)
		}
	})
}

func TestTrajectoryFeatures_Serialization(t *testing.T) {
	t.Run("serialize_deserialize", func(t *testing.T) {
		features := &TrajectoryFeatures{
			TotalDistance:     250.5,
			TotalDuration:     2000,
			AvgVelocity:       350.2,
			MaxVelocity:       800.0,
			VelocityVariance:  0.25,
			PathEfficiency:   0.85,
			DirectionChanges:  5,
			MicroCorrections:  3,
			BacktrackCount:    1,
			BacktrackDistance: 20.0,
			PauseCount:        2,
			Smoothness:        0.75,
			Jitter:            0.12,
			Entropy:           3.5,
			HumanLikeness:    0.85,
		}

		encrypted, err := NewTrajectoryEncryptionService().EncryptTrajectory(features)
		require.NoError(t, err)

		decrypted, err := NewTrajectoryEncryptionService().DecryptTrajectory(encrypted)
		require.NoError(t, err)

		assert.Equal(t, features.TotalDistance, decrypted.TotalDistance)
		assert.Equal(t, features.TotalDuration, decrypted.TotalDuration)
		assert.Equal(t, features.AvgVelocity, decrypted.AvgVelocity)
		assert.Equal(t, features.MaxVelocity, decrypted.MaxVelocity)
		assert.Equal(t, features.PathEfficiency, decrypted.PathEfficiency)
	})
}

func TestDeviceInfo_Serialization(t *testing.T) {
	t.Run("serialize_deserialize_device_info", func(t *testing.T) {
		info := &DeviceInfo{
			UserAgent:    "Mozilla/5.0",
			Platform:    "Linux x86_64",
			ScreenWidth:  1920,
			ScreenHeight: 1080,
			TouchSupport: true,
			PixelRatio:  1.0,
			Language:    "en-US",
			Timezone:    "America/New_York",
		}

		assert.NotEmpty(t, info.UserAgent)
		assert.NotEmpty(t, info.Platform)
		assert.Greater(t, info.ScreenWidth, 0)
		assert.Greater(t, info.ScreenHeight, 0)
	})
}

func TestTrajectorySummary_Serialization(t *testing.T) {
	t.Run("serialize_deserialize_summary", func(t *testing.T) {
		summary := &TrajectorySummary{
			PointCount: 50,
			Duration:   2000,
			Distance:   250.0,
			IsValid:    true,
		}

		assert.Equal(t, 50, summary.PointCount)
		assert.Equal(t, int64(2000), summary.Duration)
		assert.Equal(t, 250.0, summary.Distance)
		assert.True(t, summary.IsValid)
	})
}
