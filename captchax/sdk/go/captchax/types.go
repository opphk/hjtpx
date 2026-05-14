package captchax

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type SliderCaptchaResult struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	SliderB64     string `json:"slider_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type SliderVerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type CharPosition struct {
	Char string `json:"char"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

type ClickCaptchaResult struct {
	ID            string         `json:"id"`
	Image         string         `json:"image"`
	TargetChars   []string       `json:"target_chars"`
	CharPositions []CharPosition `json:"char_positions"`
}

type ClickVerifyResult struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
	Message string  `json:"message"`
}

type PuzzleCaptchaResult struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	PuzzleB64     string `json:"puzzle_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

type PuzzleVerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Scenario struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Difficulty  string                 `json:"difficulty,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	CreatedAt   string                 `json:"created_at,omitempty"`
	UpdatedAt   string                 `json:"updated_at,omitempty"`
}

type Webhook struct {
	ID        string            `json:"id,omitempty"`
	AppID     string            `json:"app_id"`
	URL       string            `json:"url"`
	Secret    string            `json:"secret,omitempty"`
	Events    []string          `json:"events"`
	Headers   map[string]string `json:"headers,omitempty"`
	Enabled   *bool             `json:"enabled,omitempty"`
	CreatedAt string            `json:"created_at,omitempty"`
	UpdatedAt string            `json:"updated_at,omitempty"`
}

type BatchVerifyItem struct {
	CaptchaID string         `json:"captcha_id"`
	Type      string         `json:"type"`
	TargetX   int            `json:"target_x"`
	TargetY   *int           `json:"target_y,omitempty"`
	Clicks    []CharPosition `json:"clicks,omitempty"`
}

type BatchVerifyResult struct {
	CaptchaID string  `json:"captcha_id"`
	Success   bool    `json:"success"`
	Message   string  `json:"message"`
	Score     *float64 `json:"score,omitempty"`
}

type BatchVerifySummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type BatchVerifyResponse struct {
	Results []BatchVerifyResult `json:"results"`
	Summary BatchVerifySummary  `json:"summary"`
}

type HealthStatus struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

type ScenarioListResponse struct {
	Scenarios []Scenario `json:"scenarios"`
	Total     int        `json:"total"`
}

type WebhookListResponse struct {
	Webhooks []Webhook `json:"webhooks"`
	Total    int        `json:"total"`
}

type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}
