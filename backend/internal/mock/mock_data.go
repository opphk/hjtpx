package mock

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type MockDataGenerator struct {
	rng *rand.Rand
}

func NewMockDataGenerator() *MockDataGenerator {
	return &MockDataGenerator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *MockDataGenerator) GenerateVerification() *models.Verification {
	return &models.Verification{
		SessionID:     fmt.Sprintf("sess_%d_%d", time.Now().Unix(), g.rng.Intn(10000)),
		CaptchaType:   g.randomCaptchaType(),
		ApplicationID: g.randomUintPtr(),
		UserID:       g.randomUintPtr(),
		Status:       g.randomStatus(),
		IPAddress:    g.randomIPAddress(),
		UserAgent:    g.randomUserAgent(),
		RiskScore:    g.randomFloat64(0, 100),
		Duration:     int64(g.rng.Intn(5000) + 100),
		CreatedAt:    g.randomTime(),
	}
}

func (g *MockDataGenerator) GenerateVerificationLog() *models.VerificationLog {
	return &models.VerificationLog{
		VerificationID: uint(g.rng.Intn(1000) + 1),
		SessionID:       fmt.Sprintf("sess_%d_%d", time.Now().Unix(), g.rng.Intn(10000)),
		ApplicationID:   uint(g.rng.Intn(100) + 1),
		CaptchaType:     g.randomCaptchaType(),
		Status:          g.randomStatus(),
		IPAddress:       g.randomIPAddress(),
		UserAgent:       g.randomUserAgent(),
		RiskScore:       g.randomFloat64(0, 100),
		AnalysisResult:  "Mock analysis result",
		Duration:        int64(g.rng.Intn(5000) + 100),
		CreatedAt:       g.randomTime(),
	}
}

func (g *MockDataGenerator) GenerateApplication() *models.Application {
	return &models.Application{
		Name:            fmt.Sprintf("App_%d", g.rng.Intn(1000)),
		Description:     "Mock application description",
		APIKey:          fmt.Sprintf("key_%d_%d", time.Now().Unix(), g.rng.Intn(10000)),
		IsActive:        true,
	}
}

func (g *MockDataGenerator) GenerateUser() *models.User {
	return &models.User{
		Username:     fmt.Sprintf("user_%d", g.rng.Intn(10000)),
		Email:        fmt.Sprintf("user%d@example.com", g.rng.Intn(10000)),
		PasswordHash: "$2a$10$mockhash",
		Status:       "active",
	}
}

func (g *MockDataGenerator) GenerateBehaviorData() []models.BehaviorData {
	count := g.rng.Intn(10) + 5
	data := make([]models.BehaviorData, count)
	for i := 0; i < count; i++ {
		behaviorJSON, _ := json.Marshal(map[string]interface{}{
			"event":      g.randomEventType(),
			"timestamp":  time.Now().UnixMilli() - int64(count-i)*100,
			"x":          g.randomFloat64(0, 1920),
			"y":          g.randomFloat64(0, 1080),
			"velocity":   g.randomFloat64(0, 100),
			"acceleration": g.randomFloat64(-10, 10),
		})
		data[i] = models.BehaviorData{
			Data:      string(behaviorJSON),
			DataType:  g.randomEventType(),
			Timestamp: g.randomTime(),
		}
	}
	return data
}

func (g *MockDataGenerator) GenerateCaptchaSession() map[string]interface{} {
	return map[string]interface{}{
		"session_id": fmt.Sprintf("sess_%d_%d", time.Now().Unix(), g.rng.Intn(10000)),
		"type":       g.randomCaptchaType(),
		"created_at": g.randomTime().Format(time.RFC3339),
		"expires_at": g.randomTime().Add(5 * time.Minute).Format(time.RFC3339),
		"target_x":   g.rng.Intn(300) + 50,
		"target_y":   g.rng.Intn(150) + 30,
		"tolerance":  10,
	}
}

func (g *MockDataGenerator) GenerateVerificationList(count int) []*models.Verification {
	verifications := make([]*models.Verification, count)
	for i := 0; i < count; i++ {
		verifications[i] = g.GenerateVerification()
	}
	return verifications
}

func (g *MockDataGenerator) GenerateVerificationLogList(count int) []*models.VerificationLog {
	logs := make([]*models.VerificationLog, count)
	for i := 0; i < count; i++ {
		logs[i] = g.GenerateVerificationLog()
	}
	return logs
}

func (g *MockDataGenerator) GenerateApplicationList(count int) []*models.Application {
	apps := make([]*models.Application, count)
	for i := 0; i < count; i++ {
		apps[i] = g.GenerateApplication()
	}
	return apps
}

func (g *MockDataGenerator) randomCaptchaType() string {
	types := []string{"slider", "click", "image", "text", "gesture", "voice"}
	return types[g.rng.Intn(len(types))]
}

func (g *MockDataGenerator) randomStatus() string {
	statuses := []string{"success", "failed", "pending"}
	return statuses[g.rng.Intn(len(statuses))]
}

func (g *MockDataGenerator) randomIPAddress() string {
	return fmt.Sprintf("%d.%d.%d.%d", 
		g.rng.Intn(256), 
		g.rng.Intn(256), 
		g.rng.Intn(256), 
		g.rng.Intn(256))
}

func (g *MockDataGenerator) randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15",
	}
	return userAgents[g.rng.Intn(len(userAgents))]
}

func (g *MockDataGenerator) randomEventType() string {
	events := []string{"mousemove", "mousedown", "mouseup", "click", "touchstart", "touchend"}
	return events[g.rng.Intn(len(events))]
}

func (g *MockDataGenerator) randomFloat64(min, max float64) float64 {
	return min + g.rng.Float64()*(max-min)
}

func (g *MockDataGenerator) randomUintPtr() *uint {
	val := uint(g.rng.Intn(100) + 1)
	return &val
}

func (g *MockDataGenerator) randomTime() time.Time {
	now := time.Now()
	offset := time.Duration(g.rng.Intn(7*24*60*60)) * time.Second
	return now.Add(-offset)
}
