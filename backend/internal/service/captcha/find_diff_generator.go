package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type FindDifference struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Radius int `json:"radius"`
}

type FindDiffImage struct {
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	DiffCount    int               `json:"diff_count"`
	Differences  []FindDifference `json:"differences"`
	Image1Data   string            `json:"image1_data"`
	Image2Data   string            `json:"image2_data"`
}

type CreateFindDiffRequest struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	DiffCount   int    `json:"diff_count"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateFindDiffResponse struct {
	SessionID string         `json:"session_id"`
	Image     *FindDiffImage `json:"image"`
	ExpiresIn int64          `json:"expires_in"`
	ExpiresAt int64          `json:"expires_at"`
}

type FindDiffGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

var shapeTypes = []string{"circle", "square", "triangle", "star", "diamond"}
var colors = []color.RGBA{
	{R: 255, G: 0, B: 0, A: 255},
	{R: 0, G: 255, B: 0, A: 255},
	{R: 0, G: 0, B: 255, A: 255},
	{R: 255, G: 255, B: 0, A: 255},
	{R: 255, G: 0, B: 255, A: 255},
	{R: 0, G: 255, B: 255, A: 255},
}

func NewFindDiffGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *FindDiffGeneratorService {
	return &FindDiffGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *FindDiffGeneratorService) Create(ctx context.Context, req *CreateFindDiffRequest) (*CreateFindDiffResponse, error) {
	width := req.Width
	height := req.Height
	diffCount := req.DiffCount

	if width <= 0 {
		width = 400
	}
	if height <= 0 {
		height = 400
	}
	if diffCount <= 0 {
		diffCount = 5
	}
	if diffCount > 10 {
		diffCount = 10
	}

	image, err := generateFindDiffImage(width, height, diffCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate find diff image: %w", err)
	}

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	imageData, err := json.Marshal(image)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal image: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(imageData),
		SliderURL:     "",
		GapX:          0,
		GapY:          0,
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

	return &CreateFindDiffResponse{
		SessionID: sessionID,
		Image:     image,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func generateFindDiffImage(width, height, diffCount int) (*FindDiffImage, error) {
	rand.Seed(time.Now().UnixNano())

	img1 := image.NewRGBA(image.Rect(0, 0, width, height))
	img2 := image.NewRGBA(image.Rect(0, 0, width, height))

	drawBackground(img1, width, height)
	copyImage(img1, img2, width, height)

	differences := make([]FindDifference, 0, diffCount)
	usedPositions := make(map[string]bool)

	for i := 0; i < diffCount; i++ {
		var x, y int
		var key string
		for {
			x = rand.Intn(width-100) + 50
			y = rand.Intn(height-100) + 50
			key = fmt.Sprintf("%d-%d", x/50, y/50)
			if !usedPositions[key] {
				usedPositions[key] = true
				break
			}
		}

		radius := rand.Intn(20) + 15
		differences = append(differences, FindDifference{
			X:      x,
			Y:      y,
			Radius: radius,
		})

		shapeType := shapeTypes[rand.Intn(len(shapeTypes))]
		fillColor := colors[rand.Intn(len(colors))]

		drawShape(img1, x, y, radius, shapeType, fillColor)
		drawShape(img2, x, y, radius, shapeType, colors[(rand.Intn(len(colors)-1)+1)%len(colors)])
	}

	img1Data := &strings.Builder{}
	if err := png.Encode(img1Data, img1); err != nil {
		return nil, err
	}

	img2Data := &strings.Builder{}
	if err := png.Encode(img2Data, img2); err != nil {
		return nil, err
	}

	return &FindDiffImage{
		Width:        width,
		Height:       height,
		DiffCount:    diffCount,
		Differences:  differences,
		Image1Data:   "data:image/png;base64," + base64Encode(img1Data.String()),
		Image2Data:   "data:image/png;base64," + base64Encode(img2Data.String()),
	}, nil
}

func drawBackground(img *image.RGBA, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8(200 + rand.Intn(55))
			g := uint8(200 + rand.Intn(55))
			b := uint8(200 + rand.Intn(55))
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func copyImage(src, dst *image.RGBA, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
}

func drawShape(img *image.RGBA, x, y, radius int, shapeType string, fillColor color.Color) {
	switch shapeType {
	case "circle":
		drawCircle(img, x, y, radius, fillColor)
	case "square":
		drawSquare(img, x, y, radius, fillColor)
	case "triangle":
		drawTriangle(img, x, y, radius, fillColor)
	case "star":
		drawStar(img, x, y, radius, fillColor)
	case "diamond":
		drawDiamond(img, x, y, radius, fillColor)
	}
}

func drawCircle(img *image.RGBA, x, y, radius int, c color.Color) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}

func drawSquare(img *image.RGBA, x, y, radius int, c color.Color) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			img.Set(x+dx, y+dy, c)
		}
	}
}

func drawTriangle(img *image.RGBA, x, y, radius int, c color.Color) {
	for dy := 0; dy <= radius; dy++ {
		width := radius - dy
		for dx := -width; dx <= width; dx++ {
			img.Set(x+dx, y-radius+dy, c)
		}
	}
}

func drawStar(img *image.RGBA, x, y, radius int, c color.Color) {
	drawCircle(img, x, y, radius/2, c)
	for i := 0; i < 5; i++ {
		angle := float64(i) * 2 * 3.14159 / 5
		px := x + int(float64(radius)*cos(angle))
		py := y + int(float64(radius)*sin(angle))
		drawCircle(img, px, py, radius/4, c)
	}
}

func drawDiamond(img *image.RGBA, x, y, radius int, c color.Color) {
	for dy := -radius; dy <= radius; dy++ {
		width := radius - findDiffAbs(dy)
		for dx := -width; dx <= width; dx++ {
			img.Set(x+dx, y+dy, c)
		}
	}
}

func cos(theta float64) float64 {
	return cosTable[int(theta*1000)%len(cosTable)]
}

func sin(theta float64) float64 {
	return sinTable[int(theta*1000)%len(sinTable)]
}

func findDiffAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

var cosTable = makeTable(func(x float64) float64 { return x })
var sinTable = makeTable(func(x float64) float64 { return x })

func makeTable(f func(float64) float64) []float64 {
	table := make([]float64, 1000)
	for i := range table {
		table[i] = f(float64(i) / 1000)
	}
	return table
}

func base64Encode(s string) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, (len(s)+2)/3*4)
	for i := 0; i < len(s); i += 3 {
		var n uint32
		count := 0
		for j := 0; j < 3 && i+j < len(s); j++ {
			n |= uint32(s[i+j]) << (16 - j*8)
			count++
		}
		for j := 0; j < count+1; j++ {
			result = append(result, charset[(n>>(18-j*6))&0x3F])
		}
		for j := 0; j < 2-count; j++ {
			result = append(result, '=')
		}
	}
	return string(result)
}

func (s *FindDiffGeneratorService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
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

func (s *FindDiffGeneratorService) DeleteSession(ctx context.Context, sessionID string) error {
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
