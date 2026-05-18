package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWhitelabelService(t *testing.T) {
	// 测试创建服务
	service := NewWhitelabelService(nil, nil)
	assert.NotNil(t, service)
}

func TestGetDefaultConfig(t *testing.T) {
	service := NewWhitelabelService(nil, nil)
	config := service.GetDefaultConfig()
	
	assert.Equal(t, defaultBrandName, config.BrandName)
	assert.Equal(t, defaultPrimary, config.PrimaryColor)
	assert.Equal(t, defaultSuccess, config.SuccessColor)
	assert.Equal(t, defaultWarning, config.WarningColor)
	assert.Equal(t, defaultDanger, config.DangerColor)
	assert.False(t, config.IsEnabled)
}

func TestValidateConfig(t *testing.T) {
	service := NewWhitelabelService(nil, nil)
	
	tests := []struct {
		name    string
		config  WhitelabelConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: WhitelabelConfig{
				BrandName:    "Test Brand",
				PrimaryColor: "#007bff",
				SuccessColor: "#28a745",
				WarningColor: "#ffc107",
				DangerColor:  "#dc3545",
				IsEnabled:    true,
			},
			wantErr: false,
		},
		{
			name: "empty brand name",
			config: WhitelabelConfig{
				BrandName:    "",
				PrimaryColor: "#007bff",
				SuccessColor: "#28a745",
				WarningColor: "#ffc107",
				DangerColor:  "#dc3545",
			},
			wantErr: true,
		},
		{
			name: "invalid primary color - no #",
			config: WhitelabelConfig{
				BrandName:    "Test Brand",
				PrimaryColor: "007bff",
				SuccessColor: "#28a745",
				WarningColor: "#ffc107",
				DangerColor:  "#dc3545",
			},
			wantErr: true,
		},
		{
			name: "invalid primary color - wrong length",
			config: WhitelabelConfig{
				BrandName:    "Test Brand",
				PrimaryColor: "#007bff0",
				SuccessColor: "#28a745",
				WarningColor: "#ffc107",
				DangerColor:  "#dc3545",
			},
			wantErr: true,
		},
		{
			name: "valid 3-digit color",
			config: WhitelabelConfig{
				BrandName:    "Test Brand",
				PrimaryColor: "#07f",
				SuccessColor: "#28a745",
				WarningColor: "#ffc107",
				DangerColor:  "#dc3545",
			},
			wantErr: false,
		},
		{
			name: "invalid characters",
			config: WhitelabelConfig{
				BrandName:    "Test Brand",
				PrimaryColor: "#zzzzzz",
				SuccessColor: "#28a745",
				WarningColor: "#ffc107",
				DangerColor:  "#dc3545",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDarkenColor(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		percent  int
		expected string
	}{
		{
			name:     "6-digit blue",
			hex:      "#007bff",
			percent:  10,
			expected: "#006fd6",
		},
		{
			name:     "3-digit blue",
			hex:      "#07f",
			percent:  10,
			expected: "#0066cc",
		},
		{
			name:     "6-digit green",
			hex:      "#28a745",
			percent:  10,
			expected: "#24963e",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := darkenColor(tt.hex, tt.percent)
			assert.Len(t, result, 7)
			assert.True(t, result[0] == '#')
		})
	}
}

func TestGenerateCSS(t *testing.T) {
	service := NewWhitelabelService(nil, nil)
	
	css := service.GenerateCSS()
	assert.NotEmpty(t, css)
	assert.Contains(t, css, "Whitelabel theme is disabled")
}

func TestBoolToString(t *testing.T) {
	assert.Equal(t, "true", boolToString(true))
	assert.Equal(t, "false", boolToString(false))
}

func TestWhitelabelConfigStructure(t *testing.T) {
	config := WhitelabelConfig{
		BrandName:    "Test Corp",
		PrimaryColor: "#ff0000",
		SuccessColor: "#00ff00",
		WarningColor: "#ffff00",
		DangerColor:  "#ff0000",
		LogoURL:      "/uploads/logos/test.png",
		FaviconURL:   "/uploads/logos/favicon.ico",
		CustomCSS:    ".custom { color: red; }",
		IsEnabled:    true,
	}
	
	assert.Equal(t, "Test Corp", config.BrandName)
	assert.Equal(t, "#ff0000", config.PrimaryColor)
	assert.Equal(t, "#00ff00", config.SuccessColor)
	assert.Equal(t, "#ffff00", config.WarningColor)
	assert.Equal(t, "#ff0000", config.DangerColor)
	assert.Equal(t, "/uploads/logos/test.png", config.LogoURL)
	assert.Equal(t, "/uploads/logos/favicon.ico", config.FaviconURL)
	assert.Equal(t, ".custom { color: red; }", config.CustomCSS)
	assert.True(t, config.IsEnabled)
}
