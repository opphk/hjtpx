package service

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/mock"
	"github.com/stretchr/testify/assert"
)

func TestMockDataGenerator_GenerateVerification(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	verification := generator.GenerateVerification()
	
	assert.NotNil(t, verification)
	assert.NotEmpty(t, verification.SessionID)
	assert.Contains(t, []string{"slider", "click", "image", "text", "gesture", "voice"}, verification.CaptchaType)
	assert.Contains(t, []string{"success", "failed", "pending"}, verification.Status)
	assert.NotEmpty(t, verification.IPAddress)
}

func TestMockDataGenerator_GenerateVerificationLog(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	log := generator.GenerateVerificationLog()
	
	assert.NotNil(t, log)
	assert.NotEmpty(t, log.SessionID)
	assert.Contains(t, []string{"slider", "click", "image", "text", "gesture", "voice"}, log.CaptchaType)
	assert.Contains(t, []string{"success", "failed", "pending"}, log.Status)
}

func TestMockDataGenerator_GenerateApplication(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	app := generator.GenerateApplication()
	
	assert.NotNil(t, app)
	assert.NotEmpty(t, app.Name)
	assert.NotEmpty(t, app.AppKey)
	assert.Equal(t, "active", app.Status)
}

func TestMockDataGenerator_GenerateUser(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	user := generator.GenerateUser()
	
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.Username)
	assert.NotEmpty(t, user.Email)
	assert.Contains(t, []string{"user", "admin"}, user.Role)
}

func TestMockDataGenerator_GenerateBehaviorData(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	data := generator.GenerateBehaviorData()
	
	assert.NotNil(t, data)
	assert.Greater(t, len(data), 0)
	for _, d := range data {
		assert.NotEmpty(t, d.Data)
		assert.NotEmpty(t, d.DataType)
	}
}

func TestMockDataGenerator_GenerateCaptchaSession(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	session := generator.GenerateCaptchaSession()
	
	assert.NotNil(t, session)
	assert.Contains(t, session, "session_id")
	assert.Contains(t, session, "type")
	assert.Contains(t, session, "created_at")
	assert.Contains(t, session, "expires_at")
	assert.Contains(t, session, "target_x")
	assert.Contains(t, session, "target_y")
	assert.Contains(t, session, "tolerance")
}

func TestMockDataGenerator_GenerateRiskContext(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	risk := generator.GenerateRiskContext()
	
	assert.NotNil(t, risk)
	assert.NotEmpty(t, risk.IPAddress)
	assert.NotEmpty(t, risk.UserAgent)
	assert.NotEmpty(t, risk.Fingerprint)
	assert.NotEmpty(t, risk.SessionID)
}

func TestMockDataGenerator_GenerateVerificationList(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	verifications := generator.GenerateVerificationList(10)
	
	assert.NotNil(t, verifications)
	assert.Len(t, verifications, 10)
	for _, v := range verifications {
		assert.NotNil(t, v)
	}
}

func TestMockDataGenerator_GenerateVerificationLogList(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	logs := generator.GenerateVerificationLogList(10)
	
	assert.NotNil(t, logs)
	assert.Len(t, logs, 10)
	for _, l := range logs {
		assert.NotNil(t, l)
	}
}

func TestMockDataGenerator_GenerateApplicationList(t *testing.T) {
	generator := mock.NewMockDataGenerator()
	apps := generator.GenerateApplicationList(5)
	
	assert.NotNil(t, apps)
	assert.Len(t, apps, 5)
	for _, a := range apps {
		assert.NotNil(t, a)
	}
}
