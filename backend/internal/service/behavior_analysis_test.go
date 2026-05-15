package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewBehaviorAnalysisService(t *testing.T) {
	service := NewBehaviorAnalysisService()
	assert.NotNil(t, service)
}

func TestAnalyzeBehavior(t *testing.T) {
	service := NewBehaviorAnalysisService()

	// 创建测试行为数据
	behaviorData := []models.BehaviorData{
		{
			Data:      `{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 150, "y": 250, "timestamp": 1100, "event": "mousemove"}`,
			DataType:  "mousemove",
			Timestamp: time.Now(),
		},
		{
			Data:      `{"x": 150, "y": 250, "timestamp": 1200, "event": "click"}`,
			DataType:  "click",
			Timestamp: time.Now(),
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.RiskScore, 0.0)
	assert.LessOrEqual(t, result.RiskScore, 100.0)
}

func TestCalculateRiskScore(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name           string
		behaviorData   []models.BehaviorData
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:         "empty data",
			behaviorData: []models.BehaviorData{},
			expectedMin:  0.0,
			expectedMax:  100.0,
		},
		{
			name: "normal behavior",
			behaviorData: []models.BehaviorData{
				createTestBehaviorData(100, 200, 1000, "mousemove"),
				createTestBehaviorData(120, 220, 1050, "mousemove"),
				createTestBehaviorData(140, 240, 1100, "mousemove"),
				createTestBehaviorData(140, 240, 1200, "click"),
			},
			expectedMin: 0.0,
			expectedMax: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verification := &models.Verification{}
			riskScore := service.CalculateRiskScore(verification, tt.behaviorData)
			assert.GreaterOrEqual(t, riskScore, tt.expectedMin)
			assert.LessOrEqual(t, riskScore, tt.expectedMax)
		})
	}
}

func TestVerifyWithBehaviorAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	tests := []struct {
		name           string
		captchaSuccess bool
		behaviorData   []models.BehaviorData
	}{
		{
			name:           "success with low risk",
			captchaSuccess: true,
			behaviorData: []models.BehaviorData{
				createTestBehaviorData(100, 200, 1000, "mousemove"),
				createTestBehaviorData(150, 250, 1100, "click"),
			},
		},
		{
			name:           "fail with captcha fail",
			captchaSuccess: false,
			behaviorData:   []models.BehaviorData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finalSuccess, riskScore, report := service.VerifyWithBehaviorAnalysis(tt.captchaSuccess, tt.behaviorData)
			assert.NotEmpty(t, report)
			assert.GreaterOrEqual(t, riskScore, 0.0)
			assert.LessOrEqual(t, riskScore, 100.0)
			if tt.captchaSuccess {
				assert.True(t, finalSuccess)
			}
		})
	}
}

func createTestBehaviorData(x, y int, timestamp int64, event string) models.BehaviorData {
	data := BehaviorDataPoint{
		X:         x,
		Y:         y,
		Timestamp: timestamp,
		Event:     event,
	}
	dataJSON, _ := json.Marshal(data)
	return models.BehaviorData{
		Data:      string(dataJSON),
		DataType:  event,
		Timestamp: time.Now(),
	}
}
