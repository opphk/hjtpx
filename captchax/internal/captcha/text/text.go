package text

import (
	"bytes"
	"captchax/config"
	"captchax/internal/imageutil"
	"captchax/pkg/cache"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Text 文字验证码生成器
type Text struct {
	cfg   *config.CaptchaConfig
	redis *cache.RedisClient
}

// New 创建文字验证码生成器
func New(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *Text {
	return &Text{
		cfg:   cfg,
		redis: redisClient,
	}
}

// 字符集：数字 + 大小写字母（排除易混淆字符如 0, O, l, I, 1）
const charset = "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

// 简单的像素字体数据（10x14 像素）
var pixelFont = map[rune][][]bool{
	'0': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, true, true, true},
		{true, true, false, false, true, false, true, true},
		{true, true, false, true, false, false, true, true},
		{true, true, true, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'1': {
		{false, false, true, true, false, false, false, false},
		{false, true, true, true, false, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, false, true, true, false, false, false, false},
		{true, true, true, true, true, true, true, true},
	},
	'2': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{false, false, false, false, false, true, true, false},
		{false, false, false, false, true, true, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, true, true, false, false, false, false, false},
		{true, true, true, true, true, true, true, true},
	},
	'3': {
		{true, true, true, true, true, true, true, true},
		{false, false, false, false, false, true, true, false},
		{false, false, false, false, true, true, false, false},
		{false, false, true, true, true, true, false, false},
		{false, false, false, false, false, true, true, false},
		{false, false, false, false, false, true, true, false},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'4': {
		{false, false, false, false, true, true, false, false},
		{false, false, false, true, true, true, false, false},
		{false, false, true, true, false, true, false, false},
		{false, true, true, false, false, true, false, false},
		{true, true, false, false, false, true, false, false},
		{true, true, true, true, true, true, true, true},
		{false, false, false, false, false, true, false, false},
		{false, false, false, false, false, true, false, false},
	},
	'5': {
		{true, true, true, true, true, true, true, true},
		{true, true, false, false, false, false, false, false},
		{true, true, true, true, true, true, true, false},
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'6': {
		{false, false, true, true, true, true, true, false},
		{false, true, true, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'7': {
		{true, true, true, true, true, true, true, true},
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, true, true, false},
		{false, false, false, false, true, true, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, false, true, true, false, false, false, false},
	},
	'8': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'9': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, true},
		{false, false, false, false, false, true, true, false},
		{false, false, false, false, true, true, false, false},
		{false, true, true, true, true, false, false, false},
	},
	'A': {
		{false, false, true, true, true, true, false, false},
		{false, true, true, false, false, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, true, true, true, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
	},
	'B': {
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, true, true, true, true, false},
	},
	'C': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'D': {
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, true, true, true, true, false},
	},
	'E': {
		{true, true, true, true, true, true, true, true},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, true, true, true, true, true, true},
	},
	'F': {
		{true, true, true, true, true, true, true, true},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
	},
	'G': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, true, true, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'H': {
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, true, true, true, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
	},
	'J': {
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'K': {
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, true, true, false},
		{true, true, false, false, true, true, false, false},
		{true, true, true, true, true, false, false, false},
		{true, true, false, false, true, true, false, false},
		{true, true, false, false, false, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
	},
	'L': {
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, true, true, true, true, true, true},
	},
	'M': {
		{true, true, false, false, false, false, true, true},
		{true, true, true, false, false, true, true, true},
		{true, true, true, true, true, true, true, true},
		{true, true, false, true, true, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
	},
	'N': {
		{true, true, false, false, false, false, true, true},
		{true, true, true, false, false, false, true, true},
		{true, true, true, true, false, false, true, true},
		{true, true, false, true, true, false, true, true},
		{true, true, false, false, true, true, true, true},
		{true, true, false, false, false, true, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
	},
	'P': {
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
		{true, true, false, false, false, false, false, false},
	},
	'Q': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, true, false, false, true, true},
		{true, true, false, false, true, false, true, true},
		{true, true, false, false, false, true, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'R': {
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, true, true, true, true, false},
		{true, true, false, false, true, true, false, false},
		{true, true, false, false, false, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
	},
	'S': {
		{false, true, true, true, true, true, true, false},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, false, false},
		{false, true, true, true, true, true, true, false},
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'T': {
		{true, true, true, true, true, true, true, true},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
	},
	'U': {
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, true, true, true, true, false},
	},
	'V': {
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{false, true, true, false, false, true, true, false},
		{false, true, true, false, false, true, true, false},
		{false, false, true, true, true, true, false, false},
		{false, false, false, true, true, false, false, false},
	},
	'W': {
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, true, true, false, true, true},
		{true, true, true, true, true, true, true, true},
		{true, true, false, true, true, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
	},
	'X': {
		{true, true, false, false, false, false, true, true},
		{false, true, true, false, false, true, true, false},
		{false, false, true, true, true, true, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, true, true, true, true, false, false},
		{false, true, true, false, false, true, true, false},
		{true, true, false, false, false, false, true, true},
	},
	'Y': {
		{true, true, false, false, false, false, true, true},
		{false, true, true, false, false, true, true, false},
		{false, false, true, true, true, true, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, false, true, true, false, false, false},
	},
	'Z': {
		{true, true, true, true, true, true, true, true},
		{false, false, false, false, false, false, true, true},
		{false, false, false, false, false, true, true, false},
		{false, false, false, false, true, true, false, false},
		{false, false, false, true, true, false, false, false},
		{false, false, true, true, false, false, false, false},
		{false, true, true, false, false, false, false, false},
		{true, true, true, true, true, true, true, true},
	},
}

// GenerateCaptcha 生成文字验证码
func (t *Text) GenerateCaptcha(ctx context.Context) (*CaptchaResult, error) {
	id := uuid.New().String()

	// 生成 4-6 位随机字符串
	code := t.generateRandomCode(4 + rand.Intn(3))

	// 生成验证码图片
	img := t.generateCaptchaImage(code)

	// 转换为 base64
	imgB64, err := t.imageToBase64(img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	// 存储到缓存
	if t.redis != nil {
		cacheData := &CacheData{
			ID:        id,
			Code:      code,
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}
		if err := NewCacheManager(t.cfg, t.redis).Set(ctx, id, cacheData); err != nil {
			return nil, fmt.Errorf("failed to store captcha: %w", err)
		}
	}

	return &CaptchaResult{
		ID:       id,
		ImageB64: imgB64,
	}, nil
}

// generateRandomCode 生成指定长度的随机验证码
func (t *Text) generateRandomCode(length int) string {
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

// generateCaptchaImage 生成验证码图片
func (t *Text) generateCaptchaImage(code string) *image.RGBA {
	width := t.cfg.Width
	height := t.cfg.Height
	if width == 0 {
		width = 200
	}
	if height == 0 {
		height = 80
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 1. 绘制背景渐变
	t.drawGradientBackground(img, width, height)

	// 2. 绘制干扰线
	t.drawInterferenceLines(img, width, height)

	// 3. 绘制噪点
	t.drawNoise(img, width, height)

	// 4. 绘制验证码文字
	t.drawCode(img, code, width, height)

	// 5. 应用扭曲效果
	t.distortImage(img, width, height)

	return img
}

// drawGradientBackground 绘制渐变背景
func (t *Text) drawGradientBackground(img *image.RGBA, width, height int) {
	// 随机选择渐变起点和终点颜色
	r1 := uint8(200 + rand.Intn(55))
	g1 := uint8(200 + rand.Intn(55))
	b1 := uint8(220 + rand.Intn(35))
	r2 := uint8(230 + rand.Intn(25))
	g2 := uint8(230 + rand.Intn(25))
	b2 := uint8(240 + rand.Intn(15))

	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		r := uint8(float64(r1)*(1-ratio) + float64(r2)*ratio)
		g := uint8(float64(g1)*(1-ratio) + float64(g2)*ratio)
		b := uint8(float64(b1)*(1-ratio) + float64(b2)*ratio)

		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
}

// drawInterferenceLines 绘制干扰线
func (t *Text) drawInterferenceLines(img *image.RGBA, width, height int) {
	numLines := 3 + rand.Intn(3)
	for i := 0; i < numLines; i++ {
		col := color.RGBA{
			uint8(100 + rand.Intn(100)),
			uint8(100 + rand.Intn(100)),
			uint8(100 + rand.Intn(100)),
			uint8(100 + rand.Intn(100)),
		}

		x1 := rand.Intn(width)
		y1 := rand.Intn(height)
		x2 := rand.Intn(width)
		y2 := rand.Intn(height)

		// 绘制曲线
		t.drawCurve(img, x1, y1, x2, y2, col)
	}
}

// drawCurve 绘制贝塞尔曲线
func (t *Text) drawCurve(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA) {
	cx1 := x1 + rand.Intn(50) - 25
	cy1 := y1 + rand.Intn(30) - 15
	cx2 := x2 + rand.Intn(50) - 25
	cy2 := y2 + rand.Intn(30) - 15

	for t := 0.0; t <= 1.0; t += 0.01 {
		t2 := t * t
		t3 := t2 * t
		mt := 1 - t
		mt2 := mt * mt
		mt3 := mt2 * mt

		x := int(mt3*float64(x1) + 3*mt2*t*float64(cx1) + 3*mt*t2*float64(cx2) + t3*float64(x2))
		y := int(mt3*float64(y1) + 3*mt2*t*float64(cy1) + 3*mt*t2*float64(cy2) + t3*float64(y2))

		imageutil.DrawPixelSafe(img, x, y, col)
	}
}

// drawNoise 绘制噪点
func (t *Text) drawNoise(img *image.RGBA, width, height int) {
	numNoise := width * height / 30
	for i := 0; i < numNoise; i++ {
		x := rand.Intn(width)
		y := rand.Intn(height)
		col := color.RGBA{
			uint8(rand.Intn(255)),
			uint8(rand.Intn(255)),
			uint8(rand.Intn(255)),
			uint8(80 + rand.Intn(100)),
		}
		imageutil.DrawPixelSafe(img, x, y, col)
	}

	// 绘制一些小圆圈
	for i := 0; i < 10; i++ {
		x := rand.Intn(width)
		y := rand.Intn(height)
		radius := 1 + rand.Intn(3)
		col := color.RGBA{
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(60 + rand.Intn(80)),
		}
		imageutil.DrawCircle(img, x, y, radius, col)
	}
}

// drawCode 绘制验证码文字
func (t *Text) drawCode(img *image.RGBA, code string, width, height int) {
	codeLen := len(code)
	charWidth := 20
	charHeight := 10
	totalWidth := codeLen * (charWidth + 8)
	startX := (width - totalWidth) / 2
	startY := (height - charHeight*2) / 2

	for i, c := range code {
		// 随机文字颜色
		textColor := color.RGBA{
			uint8(30 + rand.Intn(80)),
			uint8(30 + rand.Intn(80)),
			uint8(30 + rand.Intn(80)),
			255,
		}

		// 随机旋转角度
		angle := (rand.Float64() - 0.5) * 40 // ±20度

		// 计算字符位置
		x := startX + i*(charWidth+8) + rand.Intn(4) - 2
		y := startY + rand.Intn(4) - 2

		// 绘制字符
		t.drawChar(img, c, x, y, angle, textColor)
	}
}

// drawChar 绘制单个字符（带旋转）
func (t *Text) drawChar(img *image.RGBA, c rune, x, y int, angle float64, col color.RGBA) {
	fontData, exists := pixelFont[c]
	if !exists {
		// 如果字符不在字体库中，尝试使用小写或大写
		if c >= 'A' && c <= 'Z' {
			fontData, exists = pixelFont[c-'A'+'a']
		} else if c >= 'a' && c <= 'z' {
			fontData, exists = pixelFont[c-'a'+'A']
		}
		if !exists {
			return
		}
	}

	fontHeight := len(fontData)
	if fontHeight == 0 {
		return
	}
	fontWidth := len(fontData[0])

	// 计算旋转中心
	centerX := x + fontWidth/2
	centerY := y + fontHeight/2

	// 将角度转换为弧度
	rad := angle * math.Pi / 180
	cosAngle := math.Cos(rad)
	sinAngle := math.Sin(rad)

	// 绘制旋转后的字符
	for fy := 0; fy < fontHeight; fy++ {
		for fx := 0; fx < fontWidth; fx++ {
			if fontData[fy][fx] {
				// 计算相对于中心的坐标
				dx := float64(fx - fontWidth/2)
				dy := float64(fy - fontHeight/2)

				// 旋转
				newDx := dx*cosAngle - dy*sinAngle
				newDy := dx*sinAngle + dy*cosAngle

				// 计算最终位置
				px := int(math.Round(float64(centerX) + newDx))
				py := int(math.Round(float64(centerY) + newDy))

				imageutil.DrawPixelSafe(img, px, py, col)
				// 添加一点阴影/加粗效果
				imageutil.DrawPixelSafe(img, px+1, py, col)
				imageutil.DrawPixelSafe(img, px, py+1, col)
			}
		}
	}
}

// distortImage 应用扭曲效果
func (t *Text) distortImage(img *image.RGBA, width, height int) {
	tempImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(tempImg, tempImg.Bounds(), img, image.ZP, draw.Src)

	// 正弦波扭曲参数
	amplitude := 3.0 + rand.Float64()*3.0
	period := float64(height) / (2 + rand.Float64()*2)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 计算偏移量
			offsetX := int(math.Sin(float64(y)/period) * amplitude)
			offsetY := int(math.Cos(float64(x)/period) * amplitude * 0.5)

			srcX := x + offsetX
			srcY := y + offsetY

			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				img.Set(x, y, tempImg.At(srcX, srcY))
			}
		}
	}
}

// imageToBase64 将图片转换为 base64 编码
func (t *Text) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
