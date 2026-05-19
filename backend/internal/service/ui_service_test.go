package service

import (
	"encoding/json"
	"testing"
)

func TestUIserviceGetTheme(t *testing.T) {
	service := NewUIService()

	tests := []struct {
		name    string
		themeID string
		wantErr bool
	}{
		{
			name:    "Get modern theme",
			themeID: "modern",
			wantErr: false,
		},
		{
			name:    "Get elegant theme",
			themeID: "elegant",
			wantErr: false,
		},
		{
			name:    "Get minimal theme",
			themeID: "minimal",
			wantErr: false,
		},
		{
			name:    "Get vibrant theme",
			themeID: "vibrant",
			wantErr: false,
		},
		{
			name:    "Get non-existent theme",
			themeID: "nonexistent",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme, err := service.GetTheme(tt.themeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTheme() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if theme == nil {
				t.Error("GetTheme() returned nil theme")
			}
		})
	}
}

func TestUIserviceGetAllThemes(t *testing.T) {
	service := NewUIService()

	themes := service.GetAllThemes()
	if len(themes) < 4 {
		t.Errorf("GetAllThemes() returned %d themes, want at least 4", len(themes))
	}
}

func TestUIserviceGetBreakpoints(t *testing.T) {
	service := NewUIService()

	breakpoints := service.GetBreakpoints()
	if len(breakpoints) < 5 {
		t.Errorf("GetBreakpoints() returned %d breakpoints, want at least 5", len(breakpoints))
	}
}

func TestUIserviceGetBreakpointForWidth(t *testing.T) {
	service := NewUIService()

	tests := []struct {
		name  string
		width int
		want  string
	}{
		{"Small screen", 400, "xs"},
		{"Medium small screen", 600, "sm"},
		{"Medium screen", 800, "md"},
		{"Large screen", 1100, "lg"},
		{"Extra large screen", 1300, "xl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp := service.GetBreakpointForWidth(tt.width)
			if bp.Name != tt.want {
				t.Errorf("GetBreakpointForWidth(%d) = %s, want %s", tt.width, bp.Name, tt.want)
			}
		})
	}
}

func TestUIserviceGenerateCSSVariables(t *testing.T) {
	service := NewUIService()

	css, err := service.GenerateCSSVariables("modern")
	if err != nil {
		t.Errorf("GenerateCSSVariables() error = %v", err)
		return
	}

	if css == "" {
		t.Error("GenerateCSSVariables() returned empty CSS")
	}

	expectedVars := []string{
		"--ui-primary:",
		"--ui-secondary:",
		"--ui-accent:",
		"--ui-background:",
		"--ui-border-radius:",
	}

	for _, varName := range expectedVars {
		if !contains(css, varName) {
			t.Errorf("GenerateCSSVariables() missing variable %s", varName)
		}
	}
}

func TestUIserviceAccessibilityConfig(t *testing.T) {
	service := NewUIService()

	config := service.GetAccessibilityConfig()
	if config == nil {
		t.Error("GetAccessibilityConfig() returned nil")
	}

	newConfig := &AccessibilityConfig{
		HighContrast:     true,
		LargeText:        true,
		ReduceMotion:     true,
		ScreenReader:     true,
		KeyboardOnly:     true,
		FocusIndicators: true,
		ColorBlindMode:   "protanopia",
		FontSize:         1.25,
		LineHeight:       1.8,
	}

	service.UpdateAccessibilityConfig(newConfig)
	updated := service.GetAccessibilityConfig()

	if updated.HighContrast != newConfig.HighContrast {
		t.Errorf("UpdateAccessibilityConfig() HighContrast = %v, want %v", updated.HighContrast, newConfig.HighContrast)
	}
	if updated.ColorBlindMode != newConfig.ColorBlindMode {
		t.Errorf("UpdateAccessibilityConfig() ColorBlindMode = %v, want %v", updated.ColorBlindMode, newConfig.ColorBlindMode)
	}
}

func TestUIserviceApplyAccessibilityFixes(t *testing.T) {
	service := NewUIService()

	tests := []struct {
		name   string
		config *AccessibilityConfig
		check  string
	}{
		{
			name:   "High contrast",
			config: &AccessibilityConfig{HighContrast: true},
			check:  "highContrast",
		},
		{
			name:   "Large text",
			config: &AccessibilityConfig{LargeText: true},
			check:  "LargeText",
		},
		{
			name:   "Reduce motion",
			config: &AccessibilityConfig{ReduceMotion: true},
			check:  "ReduceMotion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			css := service.ApplyAccessibilityFixes(tt.config)
			if css == "" {
				t.Errorf("ApplyAccessibilityFixes() returned empty CSS")
			}
		})
	}
}

func TestUIserviceMonitorPerformance(t *testing.T) {
	service := NewUIService()

	metrics, err := service.MonitorPerformance("test-session")
	if err != nil {
		t.Errorf("MonitorPerformance() error = %v", err)
		return
	}

	if metrics == nil {
		t.Error("MonitorPerformance() returned nil metrics")
	}

	if _, ok := metrics["fps"]; !ok {
		t.Error("MonitorPerformance() missing fps metric")
	}
}

func TestUIserviceSerializeConfig(t *testing.T) {
	service := NewUIService()

	data, err := service.SerializeConfig()
	if err != nil {
		t.Errorf("SerializeConfig() error = %v", err)
		return
	}

	if len(data) == 0 {
		t.Error("SerializeConfig() returned empty data")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("SerializeConfig() produced invalid JSON: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
