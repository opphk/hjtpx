package model

type EnvInfo struct {
	UserAgent           string   `json:"user_agent"`
	Platform            string   `json:"platform"`
	Language            string   `json:"language"`
	Languages           []string `json:"languages"`
	ScreenWidth         int      `json:"screen_width"`
	ScreenHeight        int      `json:"screen_height"`
	ScreenSize          string   `json:"screen_size"`
	ColorDepth          int      `json:"color_depth"`
	PixelRatio          float64  `json:"pixel_ratio"`
	Timezone            string   `json:"timezone"`
	TimezoneOffset      int      `json:"timezone_offset"`
	CanvasFingerprint   string   `json:"canvas_fingerprint"`
	WebGLRenderer       string   `json:"webgl_renderer"`
	WebGLVendor         string   `json:"webgl_vendor"`
	WebGLSupport        bool     `json:"webgl_support"`
	CanvasSupport       bool     `json:"canvas_support"`
	AudioSupport        bool     `json:"audio_support"`
	CookiesEnabled      bool     `json:"cookies_enabled"`
	DoNotTrack          string   `json:"do_not_track"`
	Plugins             []string `json:"plugins"`
	Fonts               []string `json:"fonts"`
	TouchSupport        bool     `json:"touch_support"`
	MaxTouchPoints      int      `json:"max_touch_points"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	Fingerprint         string   `json:"fingerprint"`
	Referer             string   `json:"referer"`
}

type AutomationResult struct {
	Detected bool
	Risks    []string
}

type EnvRiskResult struct {
	RiskLevel string   `json:"risk_level"`
	Score     float64  `json:"score"`
	Risks     []string `json:"risks"`
	Action    string   `json:"action"`
}

type RiskCheckResult struct {
	Name     string `json:"name"`
	Risk     string `json:"risk"`
	Detected bool   `json:"detected"`
	Score    int    `json:"score"`
	Reason   string `json:"reason,omitempty"`
}

type EnvDetectionReport struct {
	Timestamp     int64             `json:"timestamp"`
	EnvScore      float64           `json:"env_score"`
	IsRisky       bool              `json:"is_risky"`
	RiskLevel     string            `json:"risk_level"`
	DetectedTools []string          `json:"detected_tools"`
	Checks        []RiskCheckResult `json:"checks"`
	Action        string            `json:"action"`
}
