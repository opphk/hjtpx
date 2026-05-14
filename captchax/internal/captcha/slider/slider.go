package slider

import (
	"bytes"
	"captchax/config"
	"captchax/internal/imageutil"
	"captchax/pkg/cache"
	"context"
	"encoding/base64"
	"encoding/json"
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

type Slider struct {
	cfg   *config.CaptchaConfig
	redis *cache.RedisClient
}

type CaptchaData struct {
	ID        string `json:"id"`
	TargetX   int    `json:"target_x"`
	TargetY   int    `json:"target_y"`
	CreatedAt int64  `json:"created_at"`
}

type CaptchaResult struct {
	ID            string `json:"id"`
	BackgroundB64 string `json:"background_b64"`
	SliderB64     string `json:"slider_b64"`
	TargetX       int    `json:"target_x"`
	TargetY       int    `json:"target_y"`
}

func New(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *Slider {
	return &Slider{
		cfg:   cfg,
		redis: redisClient,
	}
}

func (s *Slider) GenerateCaptcha(ctx context.Context) (*CaptchaResult, error) {
	id := uuid.New().String()

	minX := int(math.Floor(float64(s.cfg.Width) * 0.2))
	maxX := int(math.Floor(float64(s.cfg.Width) * 0.6))
	targetX := minX + rand.Intn(maxX-minX+1)

	minY := int(math.Floor(float64(s.cfg.Height) * 0.3))
	maxY := int(math.Floor(float64(s.cfg.Height) * 0.7))
	targetY := minY + rand.Intn(maxY-minY+1)

	backgroundImg := s.generateBackground(targetX, targetY)
	sliderImg := s.generateSlider(targetX, targetY, backgroundImg)

	backgroundB64, err := s.imageToBase64(backgroundImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode background: %w", err)
	}

	sliderB64, err := s.imageToBase64(sliderImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode slider: %w", err)
	}

	captchaData := CaptchaData{
		ID:        id,
		TargetX:   targetX,
		TargetY:   targetY,
		CreatedAt: time.Now().Unix(),
	}

	dataBytes, err := json.Marshal(captchaData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal captcha data: %w", err)
	}

	key := fmt.Sprintf("captcha:slider:%s", id)
	expiration := time.Duration(s.cfg.ExpireMinutes) * time.Minute
	if s.redis != nil {
		if err := s.redis.Set(ctx, key, dataBytes, expiration); err != nil {
			return nil, fmt.Errorf("failed to store captcha: %w", err)
		}
	}

	return &CaptchaResult{
		ID:            id,
		BackgroundB64: backgroundB64,
		SliderB64:     sliderB64,
		TargetX:       targetX,
		TargetY:       targetY,
	}, nil
}

func (s *Slider) generateBackground(targetX, targetY int) image.Image {
	bg := image.NewRGBA(image.Rect(0, 0, s.cfg.Width, s.cfg.Height))

	bgColor := &imageutil.SolidColor{
		R: uint8(200 + rand.Intn(40)),
		G: uint8(200 + rand.Intn(40)),
		B: uint8(200 + rand.Intn(40)),
		A: 255,
	}
	draw.Draw(bg, bg.Bounds(), bgColor, image.ZP, draw.Src)

	s.drawPattern(bg)
	s.drawPuzzlePiece(bg, targetX, targetY)

	return bg
}

func (s *Slider) drawPattern(bg *image.RGBA) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 30; i++ {
		x := r.Intn(s.cfg.Width)
		y := r.Intn(s.cfg.Height)
		size := 5 + r.Intn(15)

		patternColor := color.RGBA{
			R: uint8(r.Intn(80)),
			G: uint8(r.Intn(80)),
			B: uint8(r.Intn(80)),
			A: uint8(30 + r.Intn(50)),
		}

		switch r.Intn(3) {
		case 0:
			imageutil.DrawCircle(bg, x, y, size, patternColor)
		case 1:
			imageutil.DrawRect(bg, x, y, size, size, patternColor)
		case 2:
			imageutil.DrawLine(bg, x, y, x+size, y+size, patternColor)
		}
	}
}

func (s *Slider) drawPuzzlePiece(bg *image.RGBA, targetX, targetY int) {
	size := s.cfg.SliderSize
	if size == 0 {
		size = 50
	}
	halfSize := size / 2

	pieceColor := color.RGBA{
		R: 180,
		G: 180,
		B: 180,
		A: 255,
	}

	topY := targetY - halfSize
	bottomY := targetY + halfSize
	leftX := targetX - halfSize
	rightX := targetX + halfSize

	if topY >= 0 {
		imageutil.DrawRect(bg, leftX, topY, size, 3, pieceColor)
	}
	if bottomY < s.cfg.Height {
		imageutil.DrawRect(bg, leftX, bottomY-3, size, 3, pieceColor)
	}
	if leftX >= 0 {
		imageutil.DrawRect(bg, leftX, topY, 3, size, pieceColor)
	}
	if rightX < s.cfg.Width {
		imageutil.DrawRect(bg, rightX-3, topY, 3, size, pieceColor)
	}

	arcRadius := 8
	if targetY-arcRadius >= topY && targetY+arcRadius <= bottomY {
		if leftX-arcRadius >= 0 {
			imageutil.DrawCircle(bg, leftX, targetY, arcRadius, pieceColor)
		}
		if rightX+arcRadius < s.cfg.Width {
			imageutil.DrawCircle(bg, rightX, targetY, arcRadius, pieceColor)
		}
	}
}

func (s *Slider) generateSlider(targetX, targetY int, backgroundImg image.Image) image.Image {
	size := s.cfg.SliderSize
	if size == 0 {
		size = 50
	}
	slider := image.NewRGBA(image.Rect(0, 0, size, size))

	sliderBg := &imageutil.SolidColor{R: 240, G: 240, B: 245, A: 255}
	draw.Draw(slider, slider.Bounds(), sliderBg, image.ZP, draw.Src)

	halfSize := size / 2
	minX := targetX - halfSize
	minY := targetY - halfSize

	srcRect := image.Rect(
		max(0, minX),
		max(0, minY),
		min(s.cfg.Width, targetX+halfSize),
		min(s.cfg.Height, targetY+halfSize),
	)

	dstOffset := image.Point{
		X: max(0, -minX),
		Y: max(0, -minY),
	}
	_ = dstOffset

	draw.Draw(slider, slider.Bounds(), backgroundImg, srcRect.Min, draw.Over)

	borderColor := color.RGBA{R: 150, G: 150, B: 155, A: 255}
	imageutil.DrawRect(slider, 0, 0, size, 2, borderColor)
	imageutil.DrawRect(slider, 0, size-2, size, 2, borderColor)
	imageutil.DrawRect(slider, 0, 0, 2, size, borderColor)
	imageutil.DrawRect(slider, size-2, 0, 2, size, borderColor)

	s.drawSliderPattern(slider)

	return slider
}

func (s *Slider) drawSliderPattern(slider *image.RGBA) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	size := s.cfg.SliderSize
	if size == 0 {
		size = 50
	}

	patternColor := color.RGBA{
		R: uint8(200 + r.Intn(40)),
		G: uint8(200 + r.Intn(40)),
		B: uint8(200 + r.Intn(40)),
		A: 100,
	}

	centerX := size / 2
	centerY := size / 2
	radius := size / 4
	imageutil.DrawCircle(slider, centerX, centerY, radius, patternColor)

	innerRadius := radius / 2
	innerColor := color.RGBA{
		R: uint8(220 + r.Intn(30)),
		G: uint8(220 + r.Intn(30)),
		B: uint8(220 + r.Intn(30)),
		A: 80,
	}
	imageutil.DrawCircle(slider, centerX, centerY, innerRadius, innerColor)
}

func (s *Slider) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
