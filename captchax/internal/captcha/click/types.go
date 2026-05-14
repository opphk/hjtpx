package click

import "time"

type ClickPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type CaptchaData struct {
	ID            string          `json:"id"`
	Image         string          `json:"image"`
	TargetChars   []string        `json:"target_chars"`
	CharPositions []CharPosition `json:"char_positions"`
	CreatedAt     time.Time       `json:"created_at"`
}

type CharPosition struct {
	Char  string `json:"char"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Width int    `json:"width"`
	Height int   `json:"height"`
}

type VerifyRequest struct {
	CaptchaID string          `json:"captcha_id"`
	Clicks    []ClickPosition `json:"clicks"`
}

type VerifyResponse struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
	Message string  `json:"message"`
}

const (
	DefaultImageWidth  = 300
	DefaultImageHeight = 200
	DefaultCharCount   = 4
	ClickTolerance     = 10
	CacheExpireMinutes = 5
)
