package behavior

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

type BehaviorCaptcha struct {
	cfg        *config.CaptchaConfig
	redis      *cache.RedisClient
}

type CaptchaData struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	TargetX       int             `json:"target_x"`
	TargetY       int             `json:"target_y"`
	TargetIndex   int             `json:"target_index"`
	Targets       []Point         `json:"targets"`
	GuidePoints   []GuidePoint    `json:"guide_points"`
	ChallengeType string          `json:"challenge_type"`
	CreatedAt     int64           `json:"created_at"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type GuidePoint struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Order int    `json:"order"`
	Label string `json:"label"`
}

type CaptchaResult struct {
	ID           string       `json:"id"`
	ImageB64     string       `json:"image_b64"`
	ThumbB64     string       `json:"thumb_b64,omitempty"`
	ChallengeType string      `json:"challenge_type"`
	TargetCount  int          `json:"target_count"`
	GuidePoints  []GuidePoint `json:"guide_points,omitempty"`
	Token        string       `json:"token,omitempty"`
	ExpiresIn    int          `json:"expires_in"`
}

func New(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *BehaviorCaptcha {
	return &BehaviorCaptcha{
		cfg:   cfg,
		redis: redisClient,
	}
}

func (bc *BehaviorCaptcha) GenerateCaptcha(ctx context.Context, challengeType string) (*CaptchaResult, error) {
	id := uuid.New().String()
	token := uuid.New().String()

	width := bc.cfg.Width
	height := bc.cfg.Height
	if width == 0 {
		width = 320
	}
	if height == 0 {
		height = 240
	}

	var targets []Point
	var guidePoints []GuidePoint
	var targetCount int

	switch challengeType {
	case "click_order":
		targetCount = 4
		targets, guidePoints = bc.generateClickOrderTargets(width, height, targetCount)
	case "drag_path":
		targetCount = 3
		targets, guidePoints = bc.generateDragPathTargets(width, height, targetCount)
	case "hover_sequence":
		targetCount = 5
		targets, guidePoints = bc.generateHoverSequenceTargets(width, height, targetCount)
	default:
		targetCount = 4
		targets, guidePoints = bc.generateClickOrderTargets(width, height, targetCount)
		challengeType = "click_order"
	}

	backgroundImg := bc.generateBackground(width, height, challengeType)
	highlightImg := bc.drawTargets(backgroundImg, targets, challengeType)

	imgB64, err := bc.imageToBase64(highlightImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	captchaData := CaptchaData{
		ID:            id,
		Type:          challengeType,
		Targets:       targets,
		GuidePoints:   guidePoints,
		ChallengeType: challengeType,
		CreatedAt:     time.Now().Unix(),
	}

	if err := bc.storeCaptchaData(ctx, token, &captchaData); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &CaptchaResult{
		ID:            id,
		ImageB64:      imgB64,
		ChallengeType: challengeType,
		TargetCount:   targetCount,
		GuidePoints:   guidePoints,
		Token:         token,
		ExpiresIn:     300,
	}, nil
}

func (bc *BehaviorCaptcha) generateClickOrderTargets(width, height, count int) ([]Point, []GuidePoint) {
	targets := make([]Point, count)
	guidePoints := make([]GuidePoint, count)

	margin := 40
	cellWidth := (width - 2*margin) / 2
	cellHeight := (height - 2*margin) / 2

	positions := make([]int, 4)
	for i := range positions {
		positions[i] = i
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(positions) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		positions[i], positions[j] = positions[j], positions[i]
	}

	labels := []string{"1", "2", "3", "4"}

	for i, pos := range positions[:count] {
		row := pos / 2
		col := pos % 2

		baseX := margin + col*cellWidth + cellWidth/2
		baseY := margin + row*cellHeight + cellHeight/2

		offsetX := r.Intn(20) - 10
		offsetY := r.Intn(20) - 10

		targets[i] = Point{
			X: baseX + offsetX,
			Y: baseY + offsetY,
		}

		guidePoints[i] = GuidePoint{
			X:     targets[i].X,
			Y:     targets[i].Y,
			Order: i + 1,
			Label: labels[i],
		}
	}

	return targets, guidePoints
}

func (bc *BehaviorCaptcha) generateDragPathTargets(width, height, count int) ([]Point, []GuidePoint) {
	targets := make([]Point, count)
	guidePoints := make([]GuidePoint, count)

	margin := 60

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	startX := margin + r.Intn(width/4)
	startY := margin + r.Intn(height-2*margin)
	targets[0] = Point{X: startX, Y: startY}
	guidePoints[0] = GuidePoint{X: startX, Y: startY, Order: 1, Label: "1"}

	endX := width - margin - r.Intn(width/4)
	endY := margin + r.Intn(height-2*margin)
	targets[count-1] = Point{X: endX, Y: endY}
	guidePoints[count-1] = GuidePoint{X: endX, Y: endY, Order: count, Label: fmt.Sprintf("%d", count)}

	if count > 2 {
		midX := width/2 + r.Intn(width/4) - width/8
		midY := height/2 + r.Intn(height/4) - height/8
		targets[1] = Point{X: midX, Y: midY}
		guidePoints[1] = GuidePoint{X: midX, Y: midY, Order: 2, Label: "2"}
	}

	return targets, guidePoints
}

func (bc *BehaviorCaptcha) generateHoverSequenceTargets(width, height, count int) ([]Point, []GuidePoint) {
	targets := make([]Point, count)
	guidePoints := make([]GuidePoint, count)

	margin := 30

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	positions := make([]int, 9)
	for i := range positions {
		positions[i] = i
	}
	for i := len(positions) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		positions[i], positions[j] = positions[j], positions[i]
	}

	cellWidth := (width - 2*margin) / 3
	cellHeight := (height - 2*margin) / 3

	labels := []string{"①", "②", "③", "④", "⑤", "⑥", "⑦", "⑧", "⑨"}

	for i := 0; i < count && i < 9; i++ {
		pos := positions[i]
		row := pos / 3
		col := pos % 3

		baseX := margin + col*cellWidth + cellWidth/2
		baseY := margin + row*cellHeight + cellHeight/2

		offsetX := r.Intn(cellWidth/3) - cellWidth/6
		offsetY := r.Intn(cellHeight/3) - cellHeight/6

		targets[i] = Point{
			X: baseX + offsetX,
			Y: baseY + offsetY,
		}

		guidePoints[i] = GuidePoint{
			X:     targets[i].X,
			Y:     targets[i].Y,
			Order: i + 1,
			Label: labels[i],
		}
	}

	return targets, guidePoints
}

func (bc *BehaviorCaptcha) generateBackground(width, height int, challengeType string) image.Image {
	bg := image.NewRGBA(image.Rect(0, 0, width, height))

	bgColor := &imageutil.SolidColor{
		R: uint8(240 + rand.Intn(15)),
		G: uint8(240 + rand.Intn(15)),
		B: uint8(240 + rand.Intn(15)),
		A: 255,
	}
	draw.Draw(bg, bg.Bounds(), bgColor, image.ZP, draw.Src)

	bc.drawBackgroundPattern(bg, width, height)

	return bg
}

func (bc *BehaviorCaptcha) drawBackgroundPattern(bg *image.RGBA, width, height int) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 25; i++ {
		x := r.Intn(width)
		y := r.Intn(height)
		size := 5 + r.Intn(15)

		patternColor := color.RGBA{
			R: uint8(r.Intn(60)),
			G: uint8(r.Intn(60)),
			B: uint8(r.Intn(60)),
			A: uint8(20 + r.Intn(40)),
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

func (bc *BehaviorCaptcha) drawTargets(baseImg image.Image, targets []Point, challengeType string) image.Image {
	width := baseImg.Bounds().Dx()
	height := baseImg.Bounds().Dy()
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(result, result.Bounds(), baseImg, image.ZP, draw.Over)

	targetSize := 32
	if challengeType == "hover_sequence" {
		targetSize = 24
	}

	for i, target := range targets {
		colors := []color.RGBA{
			{R: 255, G: 87, B: 34, A: 255},
			{R: 33, G: 150, B: 243, A: 255},
			{R: 76, G: 175, B: 80, A: 255},
			{R: 156, G: 39, B: 176, A: 255},
			{R: 255, G: 152, B: 0, A: 255},
		}

		targetColor := colors[i%len(colors)]

		highlightColor := color.RGBA{
			R: uint8(math.Min(255, float64(targetColor.R)+60)),
			G: uint8(math.Min(255, float64(targetColor.G)+60)),
			B: uint8(math.Min(255, float64(targetColor.B)+60)),
			A: 200,
		}

		imageutil.DrawCircle(result, target.X, target.Y, targetSize+5, highlightColor)

		imageutil.DrawCircle(result, target.X, target.Y, targetSize, targetColor)

		innerColor := color.RGBA{
			R: uint8(math.Min(255, float64(targetColor.R)+80)),
			G: uint8(math.Min(255, float64(targetColor.G)+80)),
			B: uint8(math.Min(255, float64(targetColor.B)+80)),
			A: 255,
		}
		imageutil.DrawCircle(result, target.X, target.Y, targetSize/2, innerColor)
	}

	return result
}

func (bc *BehaviorCaptcha) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (bc *BehaviorCaptcha) storeCaptchaData(ctx context.Context, token string, data *CaptchaData) error {
	if bc.redis == nil {
		return nil
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal captcha data: %w", err)
	}

	key := fmt.Sprintf("captcha:behavior:%s", token)
	expiration := 5 * time.Minute

	return bc.redis.Set(ctx, key, dataBytes, expiration)
}

func (bc *BehaviorCaptcha) getCaptchaData(ctx context.Context, token string) (*CaptchaData, error) {
	if bc.redis == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("captcha:behavior:%s", token)
	data, err := bc.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("captcha not found or expired: %w", err)
	}

	var captchaData CaptchaData
	if err := json.Unmarshal([]byte(data), &captchaData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal captcha data: %w", err)
	}

	return &captchaData, nil
}

func (bc *BehaviorCaptcha) deleteCaptchaData(ctx context.Context, token string) error {
	if bc.redis == nil {
		return nil
	}

	key := fmt.Sprintf("captcha:behavior:%s", token)
	return bc.redis.Del(ctx, key)
}
