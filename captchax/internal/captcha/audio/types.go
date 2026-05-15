package audio

type CaptchaData struct {
	ID        string   `json:"id"`
	Code      string   `json:"code"`
	CreatedAt int64    `json:"created_at"`
	Verified  bool     `json:"verified"`
}

type CaptchaResult struct {
	ID       string `json:"id"`
	AudioB64 string `json:"audio_b64"`
	Duration int    `json:"duration"`
}

type VerifyRequest struct {
	CaptchaID string `json:"captcha_id"`
	Code      string `json:"code"`
}

type VerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
