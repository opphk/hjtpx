package text

// CaptchaData 存储文字验证码的核心数据
type CaptchaData struct {
	ID        string `json:"id"`
	Code      string `json:"code"`       // 验证码字符串
	CreatedAt int64  `json:"created_at"` // 创建时间戳
}

// CaptchaResult 返回给前端的验证码结果
type CaptchaResult struct {
	ID       string `json:"id"`
	ImageB64 string `json:"image_b64"` // 验证码图片 base64
}

// VerifyRequest 验证请求
type VerifyRequest struct {
	CaptchaID string `json:"captcha_id"`
	Code      string `json:"code"` // 用户提交的验证码
}

// VerifyResult 验证结果
type VerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
