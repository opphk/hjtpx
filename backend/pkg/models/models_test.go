package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserModel(t *testing.T) {
	user := User{
		Username:    "testuser",
		Email:       "test@example.com",
		PasswordHash: "hashedpassword",
		IsVerified:  true,
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.True(t, user.IsVerified)
}

func TestAdminModel(t *testing.T) {
	admin := Admin{
		Username:     "admin",
		PasswordHash: "hashedpassword",
		IsSuperAdmin: true,
	}

	assert.Equal(t, "admin", admin.Username)
	assert.True(t, admin.IsSuperAdmin)
}

func TestApplicationModel(t *testing.T) {
	app := Application{
		Name:        "Test Application",
		UserID:      1,
		Description: "Test description",
		APIKey:      "test-api-key-123",
		IsActive:    true,
	}

	assert.Equal(t, "Test Application", app.Name)
	assert.Equal(t, uint(1), app.UserID)
	assert.NotEmpty(t, app.APIKey)
	assert.True(t, app.IsActive)
}

func TestVerificationModel(t *testing.T) {
	verification := Verification{
		ApplicationID: 1,
		UserID:       1,
		SessionID:    "session-123",
		CaptchaType:  "slider",
		Status:       "success",
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		RiskScore:    25.5,
	}

	assert.Equal(t, "slider", verification.CaptchaType)
	assert.Equal(t, "success", verification.Status)
	assert.Equal(t, 25.5, verification.RiskScore)
}

func TestVerificationModelWithStatus(t *testing.T) {
	statuses := []string{"pending", "success", "failed", "expired"}

	for _, status := range statuses {
		verification := Verification{Status: status}
		assert.Equal(t, status, verification.Status)
	}
}

func TestBehaviorDataModel(t *testing.T) {
	data := BehaviorData{
		SessionID:      "session-123",
		VerificationID: 1,
		Data:           `{"x": 100, "y": 200, "timestamp": 1000}`,
		DataType:       "mousemove",
	}

	assert.Equal(t, "session-123", data.SessionID)
	assert.NotEmpty(t, data.Data)
	assert.Equal(t, "mousemove", data.DataType)
}

func TestVerificationLogModel(t *testing.T) {
	log := VerificationLog{
		VerificationID: 1,
		SessionID:     "session-123",
		ApplicationID: 1,
		CaptchaType:   "click",
		Status:        "success",
		IPAddress:     "192.168.1.1",
		UserAgent:     "Mozilla/5.0",
		RiskScore:     30.0,
		AnalysisResult: "Risk score: 30.0",
		Duration:      1500,
	}

	assert.Equal(t, "click", log.CaptchaType)
	assert.Equal(t, "success", log.Status)
	assert.Equal(t, int64(1500), log.Duration)
}

func TestSilentVerificationModel(t *testing.T) {
	sv := SilentVerification{
		Token:              "token-123",
		SessionID:          "session-456",
		UserID:             1,
		ApplicationID:      1,
		DeviceFingerprint:  "device-fp-789",
		RiskLevel:          "low",
		RiskScore:          15.0,
		Status:             "verified",
		NeedCaptcha:        false,
	}

	assert.Equal(t, "token-123", sv.Token)
	assert.Equal(t, "low", sv.RiskLevel)
	assert.Equal(t, 15.0, sv.RiskScore)
	assert.False(t, sv.NeedCaptcha)
}

func TestSilentVerificationRiskLevels(t *testing.T) {
	levels := []string{"low", "medium", "high"}

	for _, level := range levels {
		sv := SilentVerification{RiskLevel: level}
		assert.Equal(t, level, sv.RiskLevel)
	}
}

func TestDeviceTrustHistoryModel(t *testing.T) {
	history := DeviceTrustHistory{
		DeviceFingerprint: "fp-123",
		UserID:           1,
		TrustScore:       85.0,
		LoginCount:       10,
		IPAddress:        "192.168.1.1",
		IsTrusted:        true,
	}

	assert.Equal(t, "fp-123", history.DeviceFingerprint)
	assert.Equal(t, 85.0, history.TrustScore)
	assert.Equal(t, 10, history.LoginCount)
	assert.True(t, history.IsTrusted)
}

func TestUserBehaviorProfileModel(t *testing.T) {
	profile := UserBehaviorProfile{
		UserID:           1,
		AvgTypingSpeed:   5.5,
		AvgMouseSpeed:    2.3,
		ClickFrequency:   1.2,
		ScrollFrequency:  0.8,
		CommonLoginTime:  "09:00-17:00",
		CommonLocation:   "Shanghai",
		SampleSize:       100,
	}

	assert.Equal(t, uint(1), profile.UserID)
	assert.Equal(t, 5.5, profile.AvgTypingSpeed)
	assert.Equal(t, 2.3, profile.AvgMouseSpeed)
	assert.Equal(t, 100, profile.SampleSize)
}

func TestDeviceFingerprintModel(t *testing.T) {
	fp := DeviceFingerprint{
		UserID:          1,
		FingerprintHash: "hash-abc-123",
		UserAgent:       "Mozilla/5.0",
		ScreenInfo:      "1920x1080",
		BrowserInfo:     "Chrome",
		PlatformInfo:    "Windows",
		CanvasHash:      "canvas-hash",
		WebGLHash:       "webgl-hash",
		AudioHash:       "audio-hash",
		VisitCount:      5,
		IsTrusted:       false,
		RiskLevel:       "medium",
	}

	assert.Equal(t, "hash-abc-123", fp.FingerprintHash)
	assert.Equal(t, 5, fp.VisitCount)
	assert.False(t, fp.IsTrusted)
	assert.Equal(t, "medium", fp.RiskLevel)
}

func TestDeviceHistoryModel(t *testing.T) {
	history := DeviceHistory{
		FingerprintID: 1,
		IPAddress:     "192.168.1.1",
		Location:      "Shanghai, China",
		LoginSuccess:  true,
		UserAgent:     "Mozilla/5.0",
	}

	assert.Equal(t, uint(1), history.FingerprintID)
	assert.Equal(t, "Shanghai, China", history.Location)
	assert.True(t, history.LoginSuccess)
}

func TestVerificationModelAssociations(t *testing.T) {
	verification := Verification{
		ApplicationID: 1,
		UserID:       1,
	}

	assert.Equal(t, uint(1), verification.ApplicationID)
	assert.Equal(t, uint(1), verification.UserID)
}

func TestApplicationModelAssociations(t *testing.T) {
	app := Application{
		Name:   "Test App",
		UserID: 1,
	}

	user := User{
		Username: "testuser",
		Email:   "test@example.com",
	}
	app.User = user

	assert.Equal(t, "testuser", app.User.Username)
}
