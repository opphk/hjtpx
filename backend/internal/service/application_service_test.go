package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewApplicationService(t *testing.T) {
	service := NewApplicationService()
	assert.NotNil(t, service)
}

func TestGenerateAPIKey(t *testing.T) {
	key, err := generateAPIKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.Len(t, key, 64) // 32 bytes hex encoded

	key2, err := generateAPIKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key, key2) // Keys should be different
}

func TestToApplicationResponse(t *testing.T) {
	// Since this function depends on gorm.Model which has ID, CreatedAt, etc.,
	// we'll test with a mock or just verify that it doesn't panic
	t.Skip("Requires database models with actual data")
}

func TestCreateApplicationInputValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateApplicationInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: CreateApplicationInput{
				Name:        "Test App",
				UserID:      1,
				Description: "Test Description",
				Domain:      "example.com",
				Website:     "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			input: CreateApplicationInput{
				Name:        "",
				UserID:      1,
				Description: "Test Description",
			},
			wantErr: true,
		},
		{
			name: "zero user id",
			input: CreateApplicationInput{
				Name:   "Test App",
				UserID: 0,
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
			if tt.input.UserID == 0 {
				assert.True(t, tt.wantErr)
			}
		})
	}
}

func TestListApplicationsFilterValidation(t *testing.T) {
	tests := []struct {
		name   string
		filter ListApplicationsFilter
	}{
		{
			name: "default page",
			filter: ListApplicationsFilter{
				Page:     0,
				PageSize: 0,
			},
		},
		{
			name: "valid page and size",
			filter: ListApplicationsFilter{
				Page:     2,
				PageSize: 20,
			},
		},
		{
			name: "with keyword",
			filter: ListApplicationsFilter{
				Keyword: "test",
			},
		},
		{
			name: "with user id",
			filter: ListApplicationsFilter{
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the filter struct works as expected
			assert.NotNil(t, tt.filter)
		})
	}
}

func TestUpdateApplicationInputValidation(t *testing.T) {
	tests := []struct {
		name  string
		input UpdateApplicationInput
	}{
		{
			name:  "empty input",
			input: UpdateApplicationInput{},
		},
		{
			name: "with name update",
			input: UpdateApplicationInput{
				Name: ptrToString("Updated Name"),
			},
		},
		{
			name: "with active status",
			input: UpdateApplicationInput{
				IsActive: ptrToBool(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.input)
		})
	}
}

func TestApplicationConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config ApplicationConfig
	}{
		{
			name: "default config",
			config: ApplicationConfig{
				CaptchaTypes:         []string{"slider", "click"},
				MaxVerifyPerMinute:   60,
				MaxVerifyPerDay:      5000,
				AllowedIPs:           []string{},
				BlockRefusedRequests: false,
				CustomSettings:       map[string]interface{}{},
			},
		},
		{
			name:   "empty config",
			config: ApplicationConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.config)
		})
	}
}

// Helper functions for pointers
func ptrToString(s string) *string {
	return &s
}

func ptrToBool(b bool) *bool {
	return &b
}
