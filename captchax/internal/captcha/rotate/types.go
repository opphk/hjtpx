package rotate

// CaptchaData 存储旋转验证码的核心数据
type CaptchaData struct {
	ID         string `json:"id"`
	Angle      int    `json:"angle"`      // 正确旋转角度
	CreatedAt  int64  `json:"created_at"` // 创建时间戳
}

// CaptchaResult 返回给前端的验证码结果
type CaptchaResult struct {
	ID          string `json:"id"`
	ImageB64    string `json:"image_b64"`    // 旋转后的图片 base64
	OriginalB64 string `json:"original_b64"` // 原始图片 base64（可选）
}

// VerifyRequest 验证请求
type VerifyRequest struct {
	CaptchaID string `json:"captcha_id"`
	Angle     int    `json:"angle"` // 用户提交的旋转角度
}

// VerifyResult 验证结果
type VerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
