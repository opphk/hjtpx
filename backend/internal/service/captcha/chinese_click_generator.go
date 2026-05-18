package captcha

import (
	"bytes"
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

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

var (
	chineseChars = []string{
		"你", "我", "他", "她", "它", "这", "那", "哪",
		"上", "下", "左", "右", "前", "后", "中", "间",
		"大", "小", "多", "少", "高", "矮", "长", "短",
		"日", "月", "星", "云", "风", "雨", "雪", "雷",
		"山", "水", "火", "土", "石", "木", "金", "草",
		"人", "口", "手", "足", "耳", "目", "心", "头",
		"天", "地", "国", "家", "学", "校", "工", "作",
		"中", "文", "英", "数", "音", "体", "美", "劳",
	}
)

type ChineseClickTarget struct {
	Char   string `json:"char"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Index  int    `json:"index"`
}

type ChineseClickBoard struct {
	Width       int                  `json:"width"`
	Height      int                  `json:"height"`
	Targets     []ChineseClickTarget `json:"targets"`
	TargetChars string               `json:"target_chars"`
	TotalChars  int                  `json:"total_chars"`
}

type CreateChineseClickRequest struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	TargetCount  int    `json:"target_count"`
	TotalChars   int    `json:"total_chars"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	Fingerprint  string `json:"fingerprint"`
}

type CreateChineseClickResponse struct {
	SessionID     string             `json:"session_id"`
	ImageURL      string             `json:"image_url"`
	Board         *ChineseClickBoard `json:"board"`
	ExpiresIn     int64              `json:"expires_in"`
	ExpiresAt     int64              `json:"expires_at"`
	TargetChars   string             `json:"target_chars"`
}

type ChineseClickGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewChineseClickGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ChineseClickGeneratorService {
	return &ChineseClickGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *ChineseClickGeneratorService) Create(ctx context.Context, req *CreateChineseClickRequest) (*CreateChineseClickResponse, error) {
	width := req.Width
	height := req.Height
	targetCount := req.TargetCount
	totalChars := req.TotalChars

	if width <= 0 {
		width = 400
	}
	if height <= 0 {
		height = 300
	}
	if targetCount <= 0 {
		targetCount = 4
	}
	if targetCount > 8 {
		targetCount = 8
	}
	if totalChars <= 0 {
		totalChars = 12
	}
	if totalChars < targetCount {
		totalChars = targetCount + 4
	}

	board, imageData, err := s.generateChineseClickBoard(width, height, targetCount, totalChars)
	if err != nil {
		return nil, fmt.Errorf("failed to generate board: %w", err)
	}

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	imageURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: imageURL,
		SliderURL:     board.TargetChars,
		GapX:          targetCount,
		GapY:          totalChars,
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
	}

	return &CreateChineseClickResponse{
		SessionID:   sessionID,
		ImageURL:    imageURL,
		Board:       board,
		ExpiresIn:   int64(5 * time.Minute / time.Second),
		ExpiresAt:   expiresAt.Unix(),
		TargetChars: board.TargetChars,
	}, nil
}

func (s *ChineseClickGeneratorService) generateChineseClickBoard(width, height, targetCount, totalChars int) (*ChineseClickBoard, []byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	bgColor := color.RGBA{
		R: uint8(245 + rand.Intn(10)),
		G: uint8(245 + rand.Intn(10)),
		B: uint8(250 + rand.Intn(5)),
		A: 255,
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	s.applyNoiseBackground(img, width, height)

	targetChars := s.pickRandomChars(targetCount)
	allChars := s.generateAllChars(targetChars, totalChars)

	targets, err := s.renderChars(img, allChars, targetChars, width, height)
	if err != nil {
		return nil, nil, err
	}

	s.addInterferenceLines(img, width, height)

	s.addNoiseDots(img, width, height)

	buf := imagePool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		imagePool.Put(buf)
	}()

	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(buf, img); err != nil {
		return nil, nil, err
	}

	board := &ChineseClickBoard{
		Width:       width,
		Height:      height,
		Targets:     targets,
		TargetChars: strings.Join(targetChars, ""),
		TotalChars:  totalChars,
	}

	return board, buf.Bytes(), nil
}

func (s *ChineseClickGeneratorService) pickRandomChars(count int) []string {
	result := make([]string, 0, count)
	used := make(map[int]bool)

	for len(result) < count {
		idx := rand.Intn(len(chineseChars))
		if !used[idx] {
			used[idx] = true
			result = append(result, chineseChars[idx])
		}
	}

	return result
}

func (s *ChineseClickGeneratorService) generateAllChars(targetChars []string, totalChars int) []string {
	result := make([]string, 0, totalChars)
	result = append(result, targetChars...)

	targetSet := make(map[string]bool)
	for _, c := range targetChars {
		targetSet[c] = true
	}

	for len(result) < totalChars {
		idx := rand.Intn(len(chineseChars))
		char := chineseChars[idx]
		if !targetSet[char] {
			result = append(result, char)
		}
	}

	for i := range result {
		j := rand.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}

	return result
}

func (s *ChineseClickGeneratorService) renderChars(img *image.RGBA, allChars, targetChars []string, width, height int) ([]ChineseClickTarget, error) {
	charSize := 32 + rand.Intn(8)

	targetSet := make(map[string]bool)
	for _, c := range targetChars {
		targetSet[c] = true
	}

	targets := make([]ChineseClickTarget, 0)
	targetIndex := 0

	padding := 40
	areaWidth := width - padding*2
	areaHeight := height - padding*2

	cols := int(math.Sqrt(float64(len(allChars)) * float64(areaWidth) / float64(areaHeight)))
	if cols < 2 {
		cols = 2
	}
	rows := (len(allChars) + cols - 1) / cols

	cellWidth := areaWidth / cols
	cellHeight := areaHeight / rows

	for i, char := range allChars {
		row := i / cols
		col := i % cols

		baseX := padding + col*cellWidth + cellWidth/2
		baseY := padding + row*cellHeight + cellHeight/2

		offsetX := rand.Intn(cellWidth/3) - cellWidth/6
		offsetY := rand.Intn(cellHeight/3) - cellHeight/6

		x := baseX + offsetX
		y := baseY + offsetY

		isTarget := targetSet[char]

		var textColor color.RGBA
		if isTarget {
			textColor = color.RGBA{
				R: uint8(20 + rand.Intn(30)),
				G: uint8(60 + rand.Intn(40)),
				B: uint8(120 + rand.Intn(40)),
				A: 255,
			}
		} else {
			textColor = color.RGBA{
				R: uint8(80 + rand.Intn(80)),
				G: uint8(90 + rand.Intn(80)),
				B: uint8(100 + rand.Intn(80)),
				A: 255,
			}
		}

		s.drawCharAsShape(img, char, x-charSize/2, y-charSize/2, charSize, textColor, isTarget)

		if isTarget {
			targets = append(targets, ChineseClickTarget{
				Char:   char,
				X:      x - charSize/2,
				Y:      y - charSize/2,
				Width:  charSize,
				Height: charSize,
				Index:  targetIndex,
			})
			targetIndex++
		}
	}

	return targets, nil
}

func (s *ChineseClickGeneratorService) drawCharAsShape(img *image.RGBA, char string, x, y, size int, c color.RGBA, isTarget bool) {
	centerX := x + size/2
	centerY := y + size/2
	radius := size / 3

	if isTarget {
		s.drawFilledCircle(img, centerX, centerY, radius, c)

		borderColor := color.RGBA{
			R: uint8(int(c.R) * 60 / 100),
			G: uint8(int(c.G) * 60 / 100),
			B: uint8(int(c.B) * 60 / 100),
			A: 255,
		}
		s.drawCircleBorder(img, centerX, centerY, radius+2, 2, borderColor)
	} else {
		s.drawFilledRect(img, x+size/4, y+size/4, size/2, size/2, c)
	}

	s.drawCharLabel(img, char, x, y, size, c)
}

func (s *ChineseClickGeneratorService) drawFilledCircle(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				px, py := cx+dx, cy+dy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, c)
				}
			}
		}
	}
}

func (s *ChineseClickGeneratorService) drawCircleBorder(img *image.RGBA, cx, cy, radius, width int, c color.RGBA) {
	for w := 0; w < width; w++ {
		r := radius + w
		for angle := 0; angle < 360; angle += 2 {
			rad := float64(angle) * math.Pi / 180
			x := cx + int(float64(r)*math.Cos(rad))
			y := cy + int(float64(r)*math.Sin(rad))
			if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				img.Set(x, y, c)
			}
		}
	}
}

func (s *ChineseClickGeneratorService) drawFilledRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
				img.Set(px, py, c)
			}
		}
	}
}

func (s *ChineseClickGeneratorService) drawCharLabel(img *image.RGBA, char string, x, y, size int, c color.RGBA) {
	charIndex := 0
	for i, ch := range chineseChars {
		if ch == char {
			charIndex = i
			break
		}
	}

	gridSize := 5
	cellSize := size / gridSize

	for gy := 0; gy < gridSize; gy++ {
		for gx := 0; gx < gridSize; gx++ {
			if s.getCharPattern(charIndex, gx, gy, gridSize) {
				px := x + gx*cellSize + cellSize/4
				py := y + gy*cellSize + cellSize/4
				s.drawFilledRect(img, px, py, cellSize/2, cellSize/2, c)
			}
		}
	}
}

func (s *ChineseClickGeneratorService) getCharPattern(index, x, y, gridSize int) bool {
	patterns := []uint32{
		0x018081808, 0x010001001, 0x010101011, 0x018181818, 0x001001001,
		0x018080818, 0x000101000, 0x018080810, 0x001101100, 0x010101000,
		0x000101011, 0x011101111, 0x011000000, 0x010101010, 0x011100000,
		0x000000111, 0x010010010, 0x001000000, 0x001010100, 0x010000001,
		0x010100000, 0x000010101, 0x010000101, 0x000110000, 0x000100010,
		0x000001000, 0x010001000, 0x011111110, 0x001100000, 0x000001100,
		0x001001000, 0x010010011, 0x000000010, 0x000001001, 0x010000010,
		0x000100100, 0x001000101, 0x001101110, 0x011011011, 0x010000100,
	}

	if index >= len(patterns) {
		index = index % len(patterns)
	}

	pattern := patterns[index]
	bitPos := y*gridSize + x
	return (pattern & (1 << bitPos)) != 0
}

func (s *ChineseClickGeneratorService) applyNoiseBackground(img *image.RGBA, width, height int) {
	for i := 0; i < width*height/50; i++ {
		x := rand.Intn(width)
		y := rand.Intn(height)
		gray := uint8(rand.Intn(30))
		img.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 20})
	}
}

func (s *ChineseClickGeneratorService) addInterferenceLines(img *image.RGBA, width, height int) {
	lineCount := 3 + rand.Intn(5)

	for i := 0; i < lineCount; i++ {
		x1 := rand.Intn(width)
		y1 := rand.Intn(height)
		x2 := rand.Intn(width)
		y2 := rand.Intn(height)

		alpha := uint8(20 + rand.Intn(30))
		gray := uint8(150 + rand.Intn(50))

		s.drawLine(img, x1, y1, x2, y2, color.RGBA{R: gray, G: gray, B: gray, A: alpha})
	}
}

func (s *ChineseClickGeneratorService) addNoiseDots(img *image.RGBA, width, height int) {
	dotCount := 50 + rand.Intn(50)

	for i := 0; i < dotCount; i++ {
		x := rand.Intn(width)
		y := rand.Intn(height)
		radius := rand.Intn(2) + 1

		gray := uint8(rand.Intn(100) + 100)
		alpha := uint8(30 + rand.Intn(40))

		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx*dx+dy*dy <= radius*radius {
					px, py := x+dx, y+dy
					if px >= 0 && px < width && py >= 0 && py < height {
						img.Set(px, py, color.RGBA{R: gray, G: gray, B: gray, A: alpha})
					}
				}
			}
		}
	}
}

func (s *ChineseClickGeneratorService) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := x2 - x1
	dy := y2 - y1
	steps := int(math.Sqrt(float64(dx*dx + dy*dy)))
	if steps < 1 {
		steps = 1
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := x1 + int(float64(dx)*t+0.5)
		y := y1 + int(float64(dy)*t+0.5)

		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
	}
}

func (s *ChineseClickGeneratorService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(ctx, sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if s.captchaRepo != nil {
		session, err := s.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *ChineseClickGeneratorService) DeleteSession(ctx context.Context, sessionID string) error {
	if s.sessionCache != nil {
		if err := s.sessionCache.Delete(ctx, sessionID); err != nil {
			return err
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Delete(sessionID); err != nil {
			return err
		}
	}

	return nil
}