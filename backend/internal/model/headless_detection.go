package model

import (
	"time"
)

type NavigatorProperty struct {
	Name    string      `json:"name"`
	Value   interface{} `json:"value"`
	Present bool        `json:"present"`
	Risk    float64     `json:"risk"`
}

type PluginInfo struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	IsCommon    bool    `json:"is_common"`
	Fingerprint string  `json:"fingerprint"`
	Risk        float64 `json:"risk"`
}

type BrowserEnvironment struct {
	IsHeadless           bool            `json:"is_headless"`
	IsMobile             bool            `json:"is_mobile"`
	IsTablet             bool            `json:"is_tablet"`
	IsTouchDevice        bool            `json:"is_touch_device"`
	OS                   string          `json:"os"`
	BrowserName          string          `json:"browser_name"`
	BrowserVersion       string          `json:"browser_version"`
	Engine               string          `json:"engine"`
	DeviceMemory         float64         `json:"device_memory"`
	HardwareConcurrency  int             `json:"hardware_concurrency"`
	MaxTouchPoints       int             `json:"max_touch_points"`
	ScreenWidth          int             `json:"screen_width"`
	ScreenHeight         int             `json:"screen_height"`
	ColorDepth           int             `json:"color_depth"`
	PixelRatio           float64         `json:"pixel_ratio"`
	Languages            []string        `json:"languages"`
	Timezone             string          `json:"timezone"`
	TimezoneOffset       int             `json:"timezone_offset"`
	Plugins              []PluginInfo    `json:"plugins"`
	Fonts                []string        `json:"fonts"`
	CanvasSupported      bool            `json:"canvas_supported"`
	WebGLSupported       bool            `json:"webgl_supported"`
	WebGLRenderer        string          `json:"webgl_renderer"`
	WebGLVendor          string          `json:"webgl_vendor"`
	AudioContext         bool            `json:"audio_context"`
	SessionStorage       bool            `json:"session_storage"`
	LocalStorage         bool            `json:"local_storage"`
	IndexedDB            bool            `json:"indexed_db"`
	DoNotTrack           string          `json:"do_not_track"`
	CookiesEnabled       bool            `json:"cookies_enabled"`
	UserAgent            string          `json:"user_agent"`
	Vendor               string          `json:"vendor"`
	Platform             string          `json:"platform"`
	AppName              string          `json:"app_name"`
	AppCodeName          string          `json:"app_code_name"`
	Product              string          `json:"product"`
	ProductSub           string          `json:"product_sub"`
	VendorSub            string          `json:"vendor_sub"`
}

type AutomationToolIndicator struct {
	ToolType   string   `json:"tool_type"`
	Name       string   `json:"name"`
	Patterns   []string `json:"patterns"`
	Indicators []string `json:"indicators"`
	Weight     float64  `json:"weight"`
}

type HeadlessDetectionResult struct {
	IsHeadless         bool                    `json:"is_headless"`
	RiskScore          float64                 `json:"risk_score"`
	Confidence         float64                 `json:"confidence"`
	DetectedTool       string                  `json:"detected_tool,omitempty"`
	DetectionMethods   []string                `json:"detection_methods"`
	Indicators         []HeadlessIndicator     `json:"indicators"`
	NavigatorChecks    []NavigatorCheck        `json:"navigator_checks"`
	PluginChecks       []PluginCheck           `json:"plugin_checks"`
	AutomationChecks   []AutomationCheck       `json:"automation_checks"`
	EnvironmentChecks  []EnvironmentCheck      `json:"environment_checks"`
	Recommendations    []string                `json:"recommendations,omitempty"`
	Timestamp          time.Time               `json:"timestamp"`
	SessionID          string                  `json:"session_id,omitempty"`
}

type HeadlessIndicator struct {
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Severity    float64 `json:"severity"`
	Evidence    string  `json:"evidence"`
}

type NavigatorCheck struct {
	Property        string  `json:"property"`
	Expected        string  `json:"expected"`
	Actual          string  `json:"actual"`
	Present         bool    `json:"present"`
	IsSuspicious    bool    `json:"is_suspicious"`
	RiskScore       float64 `json:"risk_score"`
	DetectionMethod string  `json:"detection_method"`
}

type PluginCheck struct {
	PluginName      string  `json:"plugin_name"`
	Present         bool    `json:"present"`
	IsCommon        bool    `json:"is_common"`
	Suspicious      bool    `json:"suspicious"`
	RiskScore       float64 `json:"risk_score"`
	DetectionMethod string  `json:"detection_method"`
}

type AutomationCheck struct {
	ToolName       string   `json:"tool_name"`
	Detected       bool     `json:"detected"`
	Confidence     float64  `json:"confidence"`
	RiskScore      float64  `json:"risk_score"`
	Indicators     []string `json:"indicators"`
	DetectionMethod string  `json:"detection_method"`
}

type EnvironmentCheck struct {
	CheckType   string  `json:"check_type"`
	Name        string  `json:"name"`
	Passed      bool    `json:"passed"`
	RiskScore   float64 `json:"risk_score"`
	Description string  `json:"description"`
	Evidence    string  `json:"evidence"`
}

type HeadlessDetectionConfig struct {
	EnableNavigatorDetection   bool     `json:"enable_navigator_detection"`
	EnablePluginDetection      bool     `json:"enable_plugin_detection"`
	EnableAutomationDetection  bool     `json:"enable_automation_detection"`
	EnableEnvironmentDetection bool     `json:"enable_environment_detection"`
	NavigatorWeight            float64  `json:"navigator_weight"`
	PluginWeight               float64  `json:"plugin_weight"`
	AutomationWeight          float64  `json:"automation_weight"`
	EnvironmentWeight          float64  `json:"environment_weight"`
	HeadlessThreshold          float64  `json:"headless_threshold"`
	ConfidenceThreshold        float64  `json:"confidence_threshold"`
	StrictMode                 bool     `json:"strict_mode"`
	MaxPluginsCheck            int      `json:"max_plugins_check"`
	MaxFontsCheck              int      `json:"max_fonts_check"`
}

type HeadlessDetectionStats struct {
	TotalChecks         int64     `json:"total_checks"`
	HeadlessDetected    int64     `json:"headless_detected"`
	NormalBrowsers      int64     `json:"normal_browsers"`
	DetectionRate       float64   `json:"detection_rate"`
	AvgRiskScore        float64   `json:"avg_risk_score"`
	AvgConfidence       float64   `json:"avg_confidence"`
	TopDetectionMethods []string  `json:"top_detection_methods"`
	TopIndicators       []string  `json:"top_indicators"`
	StartTime           time.Time `json:"start_time"`
	EndTime             time.Time `json:"end_time,omitempty"`
}

type BrowserFingerprint struct {
	Hash              string               `json:"hash"`
	Components        map[string]interface{} `json:"components"`
	NavigatorProps    []NavigatorProperty  `json:"navigator_props"`
	PluginList        []PluginInfo         `json:"plugin_list"`
	CanvasFingerprint string               `json:"canvas_fingerprint"`
	WebGLFingerprint  string               `json:"webgl_fingerprint"`
	AudioFingerprint  string               `json:"audio_fingerprint"`
	FontList          []string             `json:"font_list"`
	CreatedAt         time.Time            `json:"created_at"`
}

func NewHeadlessDetectionConfig() *HeadlessDetectionConfig {
	return &HeadlessDetectionConfig{
		EnableNavigatorDetection:   true,
		EnablePluginDetection:      true,
		EnableAutomationDetection:  true,
		EnableEnvironmentDetection: true,
		NavigatorWeight:            0.30,
		PluginWeight:               0.25,
		AutomationWeight:           0.30,
		EnvironmentWeight:          0.15,
		HeadlessThreshold:          50.0,
		ConfidenceThreshold:        0.60,
		StrictMode:                 false,
		MaxPluginsCheck:            50,
		MaxFontsCheck:              100,
	}
}

func (r *HeadlessDetectionResult) AddIndicator(indicator HeadlessIndicator) {
	r.Indicators = append(r.Indicators, indicator)
	r.RiskScore += indicator.Severity
}

func (r *HeadlessDetectionResult) CalculateConfidence() float64 {
	if len(r.Indicators) == 0 {
		return 0.0
	}

	avgConfidence := 0.0
	for _, indicator := range r.Indicators {
		avgConfidence += indicator.Severity
	}

	avgConfidence /= float64(len(r.Indicators))
	baseConfidence := avgConfidence

	methodCount := float64(len(r.DetectionMethods))
	if methodCount > 3 {
		baseConfidence = min(baseConfidence*1.2, 1.0)
	} else if methodCount > 1 {
		baseConfidence = min(baseConfidence*1.1, 1.0)
	}

	return min(baseConfidence, 1.0)
}

func (r *HeadlessDetectionResult) DetermineHeadlessStatus(config *HeadlessDetectionConfig) bool {
	normalizedScore := min(r.RiskScore, 100.0)

	if config.StrictMode {
		return normalizedScore >= config.HeadlessThreshold && r.Confidence >= config.ConfidenceThreshold
	}

	return normalizedScore >= config.HeadlessThreshold || (normalizedScore >= config.HeadlessThreshold*0.8 && r.Confidence >= 0.7)
}

func (r *HeadlessDetectionResult) GetTopSeverityIndicators(count int) []HeadlessIndicator {
	if len(r.Indicators) <= count {
		return r.Indicators
	}

	sorted := make([]HeadlessIndicator, len(r.Indicators))
	copy(sorted, r.Indicators)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Severity > sorted[i].Severity {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted[:count]
}

func (s *HeadlessDetectionStats) UpdateStats(result *HeadlessDetectionResult) {
	s.TotalChecks++
	if result.IsHeadless {
		s.HeadlessDetected++
	} else {
		s.NormalBrowsers++
	}

	totalScore := s.AvgRiskScore * float64(s.TotalChecks-1)
	s.AvgRiskScore = (totalScore + result.RiskScore) / float64(s.TotalChecks)

	totalConf := s.AvgConfidence * float64(s.TotalChecks-1)
	s.AvgConfidence = (totalConf + result.Confidence) / float64(s.TotalChecks)

	if s.TotalChecks > 0 {
		s.DetectionRate = float64(s.HeadlessDetected) / float64(s.TotalChecks) * 100
	}

	for _, method := range result.DetectionMethods {
		found := false
		for _, m := range s.TopDetectionMethods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			s.TopDetectionMethods = append(s.TopDetectionMethods, method)
		}
	}
}
