package service

import (
	"context"
	"time"
)

type ABTestingPlatformService struct{}

type ABTest struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ModelID     uint      `json:"modelId"`
	CreatedAt   time.Time `json:"createdAt"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	EndedAt     time.Time `json:"endedAt,omitempty"`
}

type ABTestVariant struct {
	ID            uint      `json:"id"`
	TestID        uint      `json:"testId"`
	Name          string    `json:"name"`
	ModelVersionID uint     `json:"modelVersionId"`
	TrafficWeight float64   `json:"trafficWeight"`
	IsControl     bool      `json:"isControl"`
}

type ABTestMetrics struct {
	TestID        uint    `json:"testId"`
	VariantID     uint    `json:"variantId"`
	Impressions   int64   `json:"impressions"`
	Conversions   int64   `json:"conversions"`
	ConversionRate float64 `json:"conversionRate"`
	AvgLatency    float64 `json:"avgLatency"`
	ErrorRate     float64 `json:"errorRate"`
}

type CreateABTestRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	ModelID     uint                  `json:"modelId"`
	Variants    []*CreateVariantRequest `json:"variants"`
}

type CreateVariantRequest struct {
	Name          string  `json:"name"`
	ModelVersionID uint    `json:"modelVersionId"`
	TrafficWeight float64 `json:"trafficWeight"`
	IsControl     bool    `json:"isControl"`
}

func NewABTestingPlatformService() *ABTestingPlatformService {
	return &ABTestingPlatformService{}
}

func (s *ABTestingPlatformService) ListTests(ctx context.Context) ([]*ABTest, error) {
	tests := []*ABTest{
		{
			ID:          1,
			Name:        "Captcha Model v2.1 vs v2.0",
			Description: "测试新版验证码分类模型性能",
			Status:      "running",
			ModelID:     1,
			CreatedAt:   time.Now().Add(-7 * 24 * time.Hour),
			StartedAt:   time.Now().Add(-5 * 24 * time.Hour),
		},
		{
			ID:          2,
			Name:        "Risk Detector Threshold Test",
			Description: "测试不同阈值下的风险检测效果",
			Status:      "completed",
			ModelID:     2,
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			StartedAt:   time.Now().Add(-25 * 24 * time.Hour),
			EndedAt:     time.Now().Add(-10 * 24 * time.Hour),
		},
	}
	return tests, nil
}

func (s *ABTestingPlatformService) GetTest(ctx context.Context, id uint) (*ABTest, error) {
	test := &ABTest{
		ID:          id,
		Name:        "Captcha Model v2.1 vs v2.0",
		Description: "测试新版验证码分类模型性能",
		Status:      "running",
		ModelID:     1,
		CreatedAt:   time.Now().Add(-7 * 24 * time.Hour),
		StartedAt:   time.Now().Add(-5 * 24 * time.Hour),
	}
	return test, nil
}

func (s *ABTestingPlatformService) CreateTest(ctx context.Context, req *CreateABTestRequest) (*ABTest, error) {
	test := &ABTest{
		ID:          uint(time.Now().Unix()),
		Name:        req.Name,
		Description: req.Description,
		Status:      "draft",
		ModelID:     req.ModelID,
		CreatedAt:   time.Now(),
	}
	return test, nil
}

func (s *ABTestingPlatformService) StartTest(ctx context.Context, id uint) (*ABTest, error) {
	test := &ABTest{
		ID:        id,
		Status:    "running",
		StartedAt: time.Now(),
	}
	return test, nil
}

func (s *ABTestingPlatformService) StopTest(ctx context.Context, id uint) (*ABTest, error) {
	test := &ABTest{
		ID:      id,
		Status:  "completed",
		EndedAt: time.Now(),
	}
	return test, nil
}

func (s *ABTestingPlatformService) DeleteTest(ctx context.Context, id uint) error {
	return nil
}

func (s *ABTestingPlatformService) ListVariants(ctx context.Context, testID uint) ([]*ABTestVariant, error) {
	variants := []*ABTestVariant{
		{
			ID:            1,
			TestID:        testID,
			Name:          "Control (v2.0)",
			ModelVersionID: 2,
			TrafficWeight: 0.5,
			IsControl:     true,
		},
		{
			ID:            2,
			TestID:        testID,
			Name:          "Treatment (v2.1)",
			ModelVersionID: 1,
			TrafficWeight: 0.5,
			IsControl:     false,
		},
	}
	return variants, nil
}

func (s *ABTestingPlatformService) GetTestMetrics(ctx context.Context, testID uint) ([]*ABTestMetrics, error) {
	metrics := []*ABTestMetrics{
		{
			TestID:        testID,
			VariantID:     1,
			Impressions:   100000,
			Conversions:   89500,
			ConversionRate: 0.895,
			AvgLatency:    52.3,
			ErrorRate:     0.002,
		},
		{
			TestID:        testID,
			VariantID:     2,
			Impressions:   100000,
			Conversions:   94200,
			ConversionRate: 0.942,
			AvgLatency:    44.8,
			ErrorRate:     0.001,
		},
	}
	return metrics, nil
}

func (s *ABTestingPlatformService) GetVariantMetrics(ctx context.Context, variantID uint) (*ABTestMetrics, error) {
	metric := &ABTestMetrics{
		VariantID:     variantID,
		Impressions:   100000,
		Conversions:   94200,
		ConversionRate: 0.942,
		AvgLatency:    44.8,
		ErrorRate:     0.001,
	}
	return metric, nil
}
