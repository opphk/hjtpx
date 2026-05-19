package service

import (
	"encoding/json"
	"fmt"
	"time"
)

type UIMetrics struct {
	LoadTime        int64  `json:"load_time"`
	RenderTime      int64  `json:"render_time"`
	InteractionTime int64  `json:"interaction_time"`
	AnimationFPS    int    `json:"animation_fps"`
	AccessibilityScore float64 `json:"accessibility_score"`
}

type UITheme struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Primary     string                 `json:"primary"`
	Secondary   string                 `json:"secondary"`
	Accent      string                 `json:"accent"`
	Background  string                 `json:"background"`
	Surface     string                 `json:"surface"`
	Text        string                 `json:"text"`
	Border      string                 `json:"border"`
	Success     string                 `json:"success"`
	Warning     string                 `json:"warning"`
	Error       string                 `json:"error"`
	Gradient    []string               `json:"gradient"`
	Animation   map[string]interface{} `json:"animation"`
	Shadow      map[string]interface{} `json:"shadow"`
	BorderRadius string                `json:"border_radius"`
}

type AccessibilityConfig struct {
	HighContrast     bool     `json:"high_contrast"`
	LargeText       bool     `json:"large_text"`
	ReduceMotion    bool     `json:"reduce_motion"`
	ScreenReader    bool     `json:"screen_reader"`
	KeyboardOnly    bool     `json:"keyboard_only"`
	FocusIndicators bool     `json:"focus_indicators"`
	ColorBlindMode  string   `json:"color_blind_mode"`
	FontSize        float64  `json:"font_size"`
	LineHeight      float64  `json:"line_height"`
}

type ResponsiveBreakpoint struct {
	Name    string `json:"name"`
	MinWidth int   `json:"min_width"`
	MaxWidth int   `json:"max_width"`
	Columns  int   `json:"columns"`
	Gap      int   `json:"gap"`
}

type UIService struct {
	themes       map[string]*UITheme
	breakpoints  []ResponsiveBreakpoint
	accessibility *AccessibilityConfig
}

func NewUIService() *UIService {
	return &UIService{
		themes: map[string]*UITheme{
			"modern": {
				ID:          "modern",
				Name:        "Modern",
				Primary:     "#c9a96e",
				Secondary:   "#1a1a2e",
				Accent:      "#0dcaf0",
				Background:  "#ffffff",
				Surface:     "#f8f9fa",
				Text:        "#1a1a2e",
				Border:      "#e9ecef",
				Success:     "#28a745",
				Warning:     "#ffc107",
				Error:       "#dc3545",
				Gradient:    []string{"#c9a96e", "#0dcaf0"},
				Animation:   map[string]interface{}{"duration": "0.3s", "timing": "cubic-bezier(0.4, 0, 0.2, 1)"},
				Shadow:      map[string]interface{}{"sm": "0 2px 4px rgba(0,0,0,0.1)", "md": "0 4px 8px rgba(0,0,0,0.15)", "lg": "0 8px 16px rgba(0,0,0,0.2)"},
				BorderRadius: "8px",
			},
			"elegant": {
				ID:          "elegant",
				Name:        "Elegant",
				Primary:     "#6c5ce7",
				Secondary:   "#2d3436",
				Accent:      "#fd79a8",
				Background:  "#fdfbf7",
				Surface:     "#ffffff",
				Text:        "#2d3436",
				Border:      "#dfe6e9",
				Success:     "#00b894",
				Warning:     "#fdcb6e",
				Error:       "#e17055",
				Gradient:    []string{"#6c5ce7", "#fd79a8"},
				Animation:   map[string]interface{}{"duration": "0.4s", "timing": "ease-in-out"},
				Shadow:      map[string]interface{}{"sm": "0 1px 3px rgba(0,0,0,0.08)", "md": "0 3px 6px rgba(0,0,0,0.12)", "lg": "0 6px 12px rgba(0,0,0,0.16)"},
				BorderRadius: "12px",
			},
			"minimal": {
				ID:          "minimal",
				Name:        "Minimal",
				Primary:     "#0984e3",
				Secondary:   "#636e72",
				Accent:      "#00cec9",
				Background:  "#f5f6fa",
				Surface:     "#ffffff",
				Text:        "#2d3436",
				Border:      "#dcdde1",
				Success:     "#00b894",
				Warning:     "#fdcb6e",
				Error:       "#d63031",
				Gradient:    []string{"#0984e3", "#00cec9"},
				Animation:   map[string]interface{}{"duration": "0.2s", "timing": "linear"},
				Shadow:      map[string]interface{}{"sm": "0 1px 2px rgba(0,0,0,0.05)", "md": "0 2px 4px rgba(0,0,0,0.08)", "lg": "0 4px 8px rgba(0,0,0,0.1)"},
				BorderRadius: "4px",
			},
			"vibrant": {
				ID:          "vibrant",
				Name:        "Vibrant",
				Primary:     "#e84393",
				Secondary:   "#2d3436",
				Accent:      "#fdcb6e",
				Background:  "#ffeaa7",
				Surface:     "#ffffff",
				Text:        "#2d3436",
				Border:      "#fab1a0",
				Success:     "#00b894",
				Warning:     "#fdcb6e",
				Error:       "#d63031",
				Gradient:    []string{"#e84393", "#fdcb6e"},
				Animation:   map[string]interface{}{"duration": "0.5s", "timing": "bounce"},
				Shadow:      map[string]interface{}{"sm": "0 2px 6px rgba(232,67,147,0.2)", "md": "0 4px 12px rgba(232,67,147,0.3)", "lg": "0 8px 24px rgba(232,67,147,0.4)"},
				BorderRadius: "16px",
			},
		},
		breakpoints: []ResponsiveBreakpoint{
			{Name: "xs", MinWidth: 0, MaxWidth: 575, Columns: 1, Gap: 8},
			{Name: "sm", MinWidth: 576, MaxWidth: 767, Columns: 2, Gap: 12},
			{Name: "md", MinWidth: 768, MaxWidth: 991, Columns: 3, Gap: 16},
			{Name: "lg", MinWidth: 992, MaxWidth: 1199, Columns: 4, Gap: 20},
			{Name: "xl", MinWidth: 1200, MaxWidth: 1399, Columns: 6, Gap: 24},
			{Name: "xxl", MinWidth: 1400, MaxWidth: 9999, Columns: 8, Gap: 28},
		},
		accessibility: &AccessibilityConfig{
			HighContrast:     false,
			LargeText:        false,
			ReduceMotion:     false,
			ScreenReader:     false,
			KeyboardOnly:     false,
			FocusIndicators: true,
			ColorBlindMode:   "none",
			FontSize:         1.0,
			LineHeight:       1.5,
		},
	}
}

func (s *UIService) GetTheme(themeID string) (*UITheme, error) {
	theme, exists := s.themes[themeID]
	if !exists {
		return s.themes["modern"], nil
	}
	return theme, nil
}

func (s *UIService) GetAllThemes() []*UITheme {
	themes := make([]*UITheme, 0, len(s.themes))
	for _, theme := range s.themes {
		themes = append(themes, theme)
	}
	return themes
}

func (s *UIService) CreateTheme(theme *UITheme) error {
	if theme.ID == "" {
		return fmt.Errorf("theme ID is required")
	}
	s.themes[theme.ID] = theme
	return nil
}

func (s *UIService) UpdateTheme(themeID string, theme *UITheme) error {
	if _, exists := s.themes[themeID]; !exists {
		return fmt.Errorf("theme not found: %s", themeID)
	}
	theme.ID = themeID
	s.themes[themeID] = theme
	return nil
}

func (s *UIService) DeleteTheme(themeID string) error {
	if _, exists := s.themes[themeID]; !exists {
		return fmt.Errorf("theme not found: %s", themeID)
	}
	delete(s.themes, themeID)
	return nil
}

func (s *UIService) GetBreakpoints() []ResponsiveBreakpoint {
	return s.breakpoints
}

func (s *UIService) GetBreakpointForWidth(width int) *ResponsiveBreakpoint {
	for i := len(s.breakpoints) - 1; i >= 0; i-- {
		bp := s.breakpoints[i]
		if width >= bp.MinWidth && width <= bp.MaxWidth {
			return &bp
		}
	}
	return &s.breakpoints[0]
}

func (s *UIService) GetAccessibilityConfig() *AccessibilityConfig {
	return s.accessibility
}

func (s *UIService) UpdateAccessibilityConfig(config *AccessibilityConfig) {
	s.accessibility = config
}

func (s *UIService) GenerateCSSVariables(themeID string) (string, error) {
	theme, err := s.GetTheme(themeID)
	if err != nil {
		return "", err
	}

	cssVars := fmt.Sprintf(`:root {
  --ui-primary: %s;
  --ui-secondary: %s;
  --ui-accent: %s;
  --ui-background: %s;
  --ui-surface: %s;
  --ui-text: %s;
  --ui-border: %s;
  --ui-success: %s;
  --ui-warning: %s;
  --ui-error: %s;
  --ui-gradient: linear-gradient(135deg, %s);
  --ui-animation-duration: %s;
  --ui-animation-timing: %s;
  --ui-border-radius: %s;
`, theme.Primary, theme.Secondary, theme.Accent, theme.Background, theme.Surface,
		theme.Text, theme.Border, theme.Success, theme.Warning, theme.Error,
		theme.Gradient[0], theme.Animation["duration"], theme.Animation["timing"], theme.BorderRadius)

	if shadowSm, ok := theme.Shadow["sm"]; ok {
		cssVars += fmt.Sprintf("  --ui-shadow-sm: %s;\n", shadowSm)
	}
	if shadowMd, ok := theme.Shadow["md"]; ok {
		cssVars += fmt.Sprintf("  --ui-shadow-md: %s;\n", shadowMd)
	}
	if shadowLg, ok := theme.Shadow["lg"]; ok {
		cssVars += fmt.Sprintf("  --ui-shadow-lg: %s;\n", shadowLg)
	}

	cssVars += "}\n"
	return cssVars, nil
}

func (s *UIService) RecordUIMetrics(sessionID string, metrics *UIMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}
	return nil
}

func (s *UIService) GetUIMetrics(sessionID string) (*UIMetrics, error) {
	return &UIMetrics{
		LoadTime:        150,
		RenderTime:      50,
		InteractionTime: 200,
		AnimationFPS:    60,
		AccessibilityScore: 95.5,
	}, nil
}

func (s *UIService) GenerateAnimation(keyframes string, duration string) (string, error) {
	animation := fmt.Sprintf(`@keyframes %s {
  0%% { transform: translateY(0); opacity: 0; }
  100%% { transform: translateY(-10px); opacity: 1; }
}
.ui-animation-%s {
  animation: %s %s ease-out;
}`, keyframes, keyframes, keyframes, duration)
	return animation, nil
}

func (s *UIService) GenerateResponsiveStyles() string {
	styles := ".ui-container { display: grid; gap: var(--ui-gap, 16px); padding: 16px; }\n"
	for _, bp := range s.breakpoints {
		styles += fmt.Sprintf(`@media (min-width: %dpx) and (max-width: %dpx) {
  .ui-container { grid-template-columns: repeat(%d, 1fr); }
}
`, bp.MinWidth, bp.MaxWidth, bp.Columns)
	}
	return styles
}

type UIComponent struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Styles     map[string]interface{} `json:"styles"`
	Props      map[string]interface{} `json:"props"`
	Children   []UIComponent          `json:"children,omitempty"`
	Accessible bool                   `json:"accessible"`
	ARIA       map[string]string      `json:"aria"`
}

func (s *UIService) CreateComponent(component *UIComponent) error {
	if component.Type == "" || component.Name == "" {
		return fmt.Errorf("component type and name are required")
	}
	return nil
}

func (s *UIService) RenderComponent(component *UIComponent) (string, error) {
	if component == nil {
		return "", fmt.Errorf("component cannot be nil")
	}

	styles := ""
	for key, value := range component.Styles {
		styles += fmt.Sprintf("%s: %v; ", key, value)
	}

	ariaAttrs := ""
	for key, value := range component.ARIA {
		ariaAttrs += fmt.Sprintf("aria-%s=\"%s\" ", key, value)
	}

	html := fmt.Sprintf("<div class=\"ui-component ui-%s\" style=\"%s\" %s>",
		component.Name, styles, ariaAttrs)

	if len(component.Children) > 0 {
		for _, child := range component.Children {
			childHTML, _ := s.RenderComponent(&child)
			html += childHTML
		}
		html += "</div>"
	} else {
		html += "</div>"
	}

	return html, nil
}

func (s *UIService) OptimizeForScreenReader(config *AccessibilityConfig) string {
	if config.ScreenReader {
		return `.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}`
	}
	return ""
}

func (s *UIService) ApplyAccessibilityFixes(config *AccessibilityConfig) string {
	css := ""

	if config.HighContrast {
		css += `
.ui-component {
  border: 2px solid #000 !important;
  outline: 3px solid #0000ff !important;
}`
	}

	if config.LargeText {
		css += `
body {
  font-size: 125% !important;
}`
	}

	if config.ReduceMotion {
		css += `
* {
  animation-duration: 0.001ms !important;
  transition-duration: 0.001ms !important;
}`
	}

	if config.ColorBlindMode != "none" {
		switch config.ColorBlindMode {
		case "protanopia":
			css += `html { filter: url(#protanopia); }`
		case "deuteranopia":
			css += `html { filter: url(#deuteranopia); }`
		case "tritanopia":
			css += `html { filter: url(#tritanopia); }`
		}
	}

	if config.FocusIndicators {
		css += `
:focus-visible {
  outline: 3px solid var(--ui-primary);
  outline-offset: 2px;
}`
	}

	return css
}

func (s *UIService) MonitorPerformance(sessionID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"fps":              60,
		"memory_usage":    50,
		"dom_depth":       15,
		"elements_count":  250,
		"load_time":       150,
		"interactive_time": 300,
		"timestamp":       time.Now().Unix(),
	}, nil
}

func (s *UIService) SerializeConfig() ([]byte, error) {
	config := map[string]interface{}{
		"themes":       s.themes,
		"breakpoints":  s.breakpoints,
		"accessibility": s.accessibility,
	}
	return json.Marshal(config)
}
