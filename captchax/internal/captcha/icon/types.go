package icon

import "time"

// IconInfo 图标信息
type IconInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	SVG    string `json:"svg"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// CaptchaData 验证码核心数据
type CaptchaData struct {
	ID             string    `json:"id"`
	TargetIconIDs  []string  `json:"target_icon_ids"`
	AllIcons       []IconInfo `json:"all_icons"`
	GridCols       int       `json:"grid_cols"`
	GridRows       int       `json:"grid_rows"`
	CreatedAt      time.Time `json:"created_at"`
}

// CaptchaResult 返回给前端的验证码结果
type CaptchaResult struct {
	ID             string    `json:"id"`
	TargetIcons    []IconInfo `json:"target_icons"`
	AllIcons       []IconInfo `json:"all_icons"`
	GridCols       int       `json:"grid_cols"`
	GridRows       int       `json:"grid_rows"`
	IconSize       int       `json:"icon_size"`
}

// VerifyRequest 验证请求
type VerifyRequest struct {
	CaptchaID string   `json:"captcha_id"`
	IconIDs   []string `json:"icon_ids"`
}

// VerifyResult 验证结果
type VerifyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

const (
	// DefaultIconSize 默认图标尺寸
	DefaultIconSize = 64
	// DefaultGridCols 默认网格列数
	DefaultGridCols = 5
	// DefaultGridRows 默认网格行数
	DefaultGridRows = 3
	// MinTargetIcons 最小目标图标数
	MinTargetIcons = 3
	// MaxTargetIcons 最大目标图标数
	MaxTargetIcons = 5
	// MinTotalIcons 最小总图标数
	MinTotalIcons = 9
	// MaxTotalIcons 最大总图标数
	MaxTotalIcons = 15
	// CacheExpireMinutes 缓存过期时间（分钟）
	CacheExpireMinutes = 5
)
