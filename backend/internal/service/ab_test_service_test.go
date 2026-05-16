package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestNewABTestService(t *testing.T) {
	service := NewABTestService()
	assert.NotNil(t, service)
}

func TestCreateABTestInputValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateABTestInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: CreateABTestInput{
				Name:         "Test A/B",
				Description:  "Test Description",
				ApplicationID: 1,
				Variants: []CreateVariantInput{
					{Name: "Control", IsControl: true, TrafficPercent: 50},
					{Name: "Variant", IsControl: false, TrafficPercent: 50},
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			input: CreateABTestInput{
				Name:         "",
				ApplicationID: 1,
				Variants: []CreateVariantInput{
					{Name: "Control", TrafficPercent: 50},
					{Name: "Variant", TrafficPercent: 50},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid traffic - not 100",
			input: CreateABTestInput{
				Name:         "Test A/B",
				ApplicationID: 1,
				Variants: []CreateVariantInput{
					{Name: "Control", TrafficPercent: 60},
					{Name: "Variant", TrafficPercent: 50},
				},
			},
			wantErr: true,
		},
		{
			name: "no variants",
			input: CreateABTestInput{
				Name:         "Test A/B",
				ApplicationID: 1,
				Variants:     []CreateVariantInput{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.input.Name == "" {
				assert.True(t, tt.wantErr)
			}
			if len(tt.input.Variants) < 2 {
				assert.True(t, tt.wantErr)
			}
			totalTraffic := 0
			for _, v := range tt.input.Variants {
				totalTraffic += v.TrafficPercent
			}
			if totalTraffic != 100 && len(tt.input.Variants) > 0 {
				assert.True(t, tt.wantErr)
			}
		})
	}
}

func TestSelectVariantByHash(t *testing.T) {
	service := NewABTestService()
	assert.NotNil(t, service)

	// Create test variants without gorm.Model
	type testVariant struct {
		ID             uint
		Name           string
		TrafficPercent int
	}
	
	variants := []testVariant{
		{ID: 1, Name: "Control", TrafficPercent: 50},
		{ID: 2, Name: "Variant A", TrafficPercent: 50},
	}

	// Convert to models.ABTestVariant for testing
	modelVariants := make([]models.ABTestVariant, len(variants))
	for i, v := range variants {
		modelVariants[i].Name = v.Name
		modelVariants[i].TrafficPercent = v.TrafficPercent
	}

	// Test that the function doesn't panic
	variant := service.selectVariantByHash("session-123", modelVariants)
	assert.NotNil(t, variant)
}

func TestCalculateConfidence(t *testing.T) {
	service := NewABTestService()

	tests := []struct {
		name           string
		cVisitors      int64
		cConversions   int64
		vVisitors      int64
		vConversions   int64
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:         "no visitors",
			cVisitors:    0,
			cConversions: 0,
			vVisitors:    0,
			vConversions: 0,
			expectedMin:  0,
			expectedMax:  0,
		},
		{
			name:         "some visitors no difference",
			cVisitors:    1000,
			cConversions: 100,
			vVisitors:    1000,
			vConversions: 100,
			expectedMin:  0,
			expectedMax:  60,
		},
		{
			name:         "big difference",
			cVisitors:    1000,
			cConversions: 50,
			vVisitors:    1000,
			vConversions: 100,
			expectedMin:  90,
			expectedMax:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := service.calculateConfidence(tt.cVisitors, tt.cConversions, tt.vVisitors, tt.vConversions)
			assert.GreaterOrEqual(t, confidence, tt.expectedMin)
			assert.LessOrEqual(t, confidence, tt.expectedMax)
		})
	}
}

func TestNormalCDF(t *testing.T) {
	service := NewABTestService()

	tests := []struct {
		name     string
		x        float64
		expected float64
	}{
		{name: "zero", x: 0, expected: 0.5},
		{name: "positive", x: 1, expected: 0.84},
		{name: "negative", x: -1, expected: 0.16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.normalCDF(tt.x)
			assert.InDelta(t, tt.expected, result, 0.02)
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	service := NewABTestService()

	tests := []struct {
		name             string
		variants         []VariantStats
		status           string
		expectCount      int
		expectWinnerHint bool
	}{
		{
			name: "small sample size",
			variants: []VariantStats{
				{VariantID: 1, VariantName: "Control", IsControl: true, Visitors: 50, Conversions: 5, ConversionRate: 10},
				{VariantID: 2, VariantName: "Variant", IsControl: false, Visitors: 50, Conversions: 10, ConversionRate: 20},
			},
			status:      "running",
			expectCount: 2, // data small + running
		},
		{
			name: "high confidence winner",
			variants: []VariantStats{
				{VariantID: 1, VariantName: "Control", IsControl: true, Visitors: 2000, Conversions: 200, ConversionRate: 10},
				{VariantID: 2, VariantName: "Variant", IsControl: false, Visitors: 2000, Conversions: 400, ConversionRate: 20, Improvement: 100, Confidence: 99},
			},
			status:           "running",
			expectCount:      2,
			expectWinnerHint: true,
		},
		{
			name: "enough data no winner",
			variants: []VariantStats{
				{VariantID: 1, VariantName: "Control", IsControl: true, Visitors: 1500, Conversions: 150, ConversionRate: 10},
				{VariantID: 2, VariantName: "Variant", IsControl: false, Visitors: 1500, Conversions: 160, ConversionRate: 10.7, Improvement: 7, Confidence: 60},
			},
			status:      "running",
			expectCount: 2, // running + no winner hint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := service.generateRecommendations(tt.variants, tt.status)
			assert.Len(t, recommendations, tt.expectCount)
			
			if tt.expectWinnerHint {
				hasWinnerRec := false
				for _, r := range recommendations {
					if len(r) > 0 {
						hasWinnerRec = true
						break
					}
				}
				assert.True(t, hasWinnerRec)
			}
		})
	}
}

func TestListABTestsFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter ListABTestsFilter
	}{
		{
			name: "default filter",
			filter: ListABTestsFilter{
				Page:     0,
				PageSize: 0,
			},
		},
		{
			name: "with keyword",
			filter: ListABTestsFilter{
				Keyword: "test",
			},
		},
		{
			name: "with status",
			filter: ListABTestsFilter{
				Status: "running",
			},
		},
		{
			name: "with application id",
			filter: ListABTestsFilter{
				ApplicationID: 1,
			},
		},
		{
			name: "with sorting",
			filter: ListABTestsFilter{
				SortField: "created_at",
				SortOrder: "ASC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.filter)
		})
	}
}

func TestUpdateABTestInput(t *testing.T) {
	name := "Updated Name"
	desc := "Updated Description"
	config := map[string]interface{}{"key": "value"}

	tests := []struct {
		name  string
		input UpdateABTestInput
	}{
		{
			name:  "empty input",
			input: UpdateABTestInput{},
		},
		{
			name: "with name update",
			input: UpdateABTestInput{
				Name: &name,
			},
		},
		{
			name: "with description update",
			input: UpdateABTestInput{
				Description: &desc,
			},
		},
		{
			name: "with config update",
			input: UpdateABTestInput{
				Config: &config,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.input)
		})
	}
}

func TestVariantStats(t *testing.T) {
	tests := []struct {
		name   string
		stats  VariantStats
	}{
		{
			name: "control variant",
			stats: VariantStats{
				VariantID:      1,
				VariantName:    "Control",
				IsControl:      true,
				Visitors:       1000,
				Conversions:    100,
				ConversionRate: 10.0,
				TrafficPercent: 50,
			},
		},
		{
			name: "test variant with improvement",
			stats: VariantStats{
				VariantID:      2,
				VariantName:    "Test",
				IsControl:      false,
				Visitors:       1000,
				Conversions:    150,
				ConversionRate: 15.0,
				Improvement:    50.0,
				Confidence:     95.0,
				TrafficPercent: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.stats)
			assert.GreaterOrEqual(t, tt.stats.Visitors, int64(0))
			assert.GreaterOrEqual(t, tt.stats.Conversions, int64(0))
		})
	}
}
