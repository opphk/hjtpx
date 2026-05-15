package rotate

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
	"time"

	"github.com/google/uuid"
)

// Rotate 旋转验证码生成器
type Rotate struct {
	cfg   *config.CaptchaConfig
	redis *cache.RedisClient
}

// New 创建旋转验证码生成器
func New(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *Rotate {
	return &Rotate{
		cfg:   cfg,
		redis: redisClient,
	}
}

// GenerateCaptcha 生成旋转验证码
func (r *Rotate) GenerateCaptcha(ctx context.Context) (*CaptchaResult, error) {
	id := uuid.New().String()

	// 生成随机旋转角度 (0-359度)
	targetAngle := rand.Intn(360)

	// 生成原始图案和旋转后的图案
	originalImg := r.generateOriginalImage()
	rotatedImg := r.rotateImage(originalImg, targetAngle)

	originalB64, err := r.imageToBase64(originalImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode original image: %w", err)
	}

	imageB64, err := r.imageToBase64(rotatedImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode rotated image: %w", err)
	}

	captchaData := CaptchaData{
		ID:        id,
		Angle:     targetAngle,
		CreatedAt: time.Now().Unix(),
	}

	// 存储到缓存
	cacheData := &CacheData{
		ID:        id,
		Angle:     targetAngle,
		CreatedAt: captchaData.CreatedAt,
		Verified:  false,
	}

	if r.redis != nil {
		if err := NewCacheManager(r.cfg, r.redis).Set(ctx, id, cacheData); err != nil {
			return nil, fmt.Errorf("failed to store captcha: %w", err)
		}
	}

	return &CaptchaResult{
		ID:          id,
		ImageB64:    imageB64,
		OriginalB64: originalB64,
	}, nil
}

// generateOriginalImage 生成原始验证码图案（一个带箭头的圆形）
func (r *Rotate) generateOriginalImage() image.Image {
	width := r.cfg.Width
	height := r.cfg.Height
	if width == 0 {
		width = 300
	}
	if height == 0 {
		height = 300
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 背景色
	bgColor := &imageutil.SolidColor{
		R: uint8(240 + rand.Intn(15)),
		G: uint8(240 + rand.Intn(15)),
		B: uint8(245 + rand.Intn(10)),
		A: 255,
	}
	draw.Draw(img, img.Bounds(), bgColor, image.ZP, draw.Src)

	centerX := width / 2
	centerY := height / 2
	radius := int(math.Min(float64(width), float64(height)) / 2.5)

	// 绘制圆形边框
	circleColor := color.RGBA{
		R: uint8(100 + rand.Intn(50)),
		G: uint8(120 + rand.Intn(50)),
		B: uint8(200 + rand.Intn(50)),
		A: 255,
	}
	imageutil.DrawCircle(img, centerX, centerY, radius, circleColor)

	// 绘制内部圆盘
	innerColor := color.RGBA{
		R: uint8(200 + rand.Intn(40)),
		G: uint8(210 + rand.Intn(30)),
		B: uint8(240 + rand.Intn(15)),
		A: 255,
	}
	imageutil.DrawCircle(img, centerX, centerY, radius-10, innerColor)

	// 绘制指针/箭头（指向0度/正上方）
	r.drawArrow(img, centerX, centerY, radius-25, circleColor)

	// 添加装饰性图案
	r.drawDecorations(img, centerX, centerY, radius, circleColor)

	return img
}

// drawArrow 绘制箭头
func (r *Rotate) drawArrow(img *image.RGBA, centerX, centerY, length int, col color.RGBA) {
	// 箭头主体
	endX := centerX
	endY := centerY - length
	imageutil.DrawLine(img, centerX, centerY, endX, endY, col)

	// 箭头头部
	arrowSize := 15
	leftX := endX - arrowSize
	leftY := endY + arrowSize
	rightX := endX + arrowSize
	rightY := endY + arrowSize

	imageutil.DrawLine(img, endX, endY, leftX, leftY, col)
	imageutil.DrawLine(img, endX, endY, rightX, rightY, col)
	imageutil.DrawLine(img, leftX, leftY, rightX, rightY, col)

	// 中心圆点
	imageutil.DrawCircle(img, centerX, centerY, 8, col)
}

// drawDecorations 添加装饰性图案
func (r *Rotate) drawDecorations(img *image.RGBA, centerX, centerY, radius int, col color.RGBA) {
	// 在圆周上添加标记点
	for i := 0; i < 12; i++ {
		angle := float64(i) * 30 * math.Pi / 180
		x := centerX + int(float64(radius)*math.Sin(angle))
		y := centerY - int(float64(radius)*math.Cos(angle))

		dotSize := 5
		if i%3 == 0 {
			dotSize = 8
		}
		imageutil.DrawCircle(img, x, y, dotSize, col)
	}
}

// rotateImage 旋转图片
func (r *Rotate) rotateImage(src image.Image, angle int) image.Image {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// 计算旋转后的尺寸（使用对角线长度作为新尺寸）
	diagonal := math.Sqrt(float64(srcWidth*srcWidth + srcHeight*srcHeight))
	dstWidth := int(diagonal) + 2
	dstHeight := int(diagonal) + 2

	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))

	// 背景色
	bgColor := &imageutil.SolidColor{R: 245, G: 245, B: 250, A: 255}
	draw.Draw(dst, dst.Bounds(), bgColor, image.ZP, draw.Src)

	// 计算旋转中心
	srcCenterX := float64(srcWidth) / 2
	srcCenterY := float64(srcHeight) / 2
	dstCenterX := float64(dstWidth) / 2
	dstCenterY := float64(dstHeight) / 2

	// 角度转弧度
	theta := -float64(angle) * math.Pi / 180
	cosTheta := math.Cos(theta)
	sinTheta := math.Sin(theta)

	// 旋转绘制
	for dstY := 0; dstY < dstHeight; dstY++ {
		for dstX := 0; dstX < dstWidth; dstX++ {
			// 计算相对于目标中心的坐标
			dx := float64(dstX) - dstCenterX
			dy := float64(dstY) - dstCenterY

			// 反向旋转计算源坐标
			srcX := dx*cosTheta + dy*sinTheta + srcCenterX
			srcY := -dx*sinTheta + dy*cosTheta + srcCenterY

			// 双线性插值采样
			if srcX >= 0 && srcX < float64(srcWidth-1) && srcY >= 0 && srcY < float64(srcHeight-1) {
				x0 := int(math.Floor(srcX))
				y0 := int(math.Floor(srcY))
				x1 := x0 + 1
				y1 := y0 + 1

				dx1 := srcX - float64(x0)
				dy1 := srcY - float64(y0)

				c00 := color.RGBAModel.Convert(src.At(x0, y0)).(color.RGBA)
				c10 := color.RGBAModel.Convert(src.At(x1, y0)).(color.RGBA)
				c01 := color.RGBAModel.Convert(src.At(x0, y1)).(color.RGBA)
				c11 := color.RGBAModel.Convert(src.At(x1, y1)).(color.RGBA)

				r := uint8(float64(c00.R)*(1-dx1)*(1-dy1) + float64(c10.R)*dx1*(1-dy1) + float64(c01.R)*(1-dx1)*dy1 + float64(c11.R)*dx1*dy1)
				g := uint8(float64(c00.G)*(1-dx1)*(1-dy1) + float64(c10.G)*dx1*(1-dy1) + float64(c01.G)*(1-dx1)*dy1 + float64(c11.G)*dx1*dy1)
				b := uint8(float64(c00.B)*(1-dx1)*(1-dy1) + float64(c10.B)*dx1*(1-dy1) + float64(c01.B)*(1-dx1)*dy1 + float64(c11.B)*dx1*dy1)
				a := uint8(float64(c00.A)*(1-dx1)*(1-dy1) + float64(c10.A)*dx1*(1-dy1) + float64(c01.A)*(1-dx1)*dy1 + float64(c11.A)*dx1*dy1)

				dst.Set(dstX, dstY, color.RGBA{r, g, b, a})
			}
		}
	}

	return dst
}

// imageToBase64 将图片转换为 base64 编码
func (r *Rotate) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
