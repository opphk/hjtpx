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
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type EnhancedSliderGenerator struct {
	imageGenerator *EnhancedImageGenerator
	sessionCache   *cache.SessionCache
	captchaRepo    *db.CaptchaRepository
	obstacleGenerator *ObstacleGenerator
	trajectoryGenerator *TrajectoryGenerator
	gapDetector *SmartGapDetector
	resistanceSystem *AdaptiveResistanceSystem
}

type EnhancedSliderResult struct {
	Background     []byte
	Slider          []byte
	GapX            int
	GapY            int
	Obstacles       []ObstacleInfo
	TrajectoryHint  TrajectoryHint
	ResistanceLevel int
	Difficulty      int
}

type ObstacleInfo struct {
	Type     string `json:"type"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Rotation int    `json:"rotation"`
}

type TrajectoryHint struct {
	SuggestedSpeed float64 `json:"suggested_speed"`
	PathComplexity int    `json:"path_complexity"`
	Hints          []Point `json:"hints"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type EnhancedCreateRequest struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	SliderWidth  int    `json:"slider_width"`
	SliderHeight int    `json:"slider_height"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	Fingerprint  string `json:"fingerprint"`
	Difficulty   int    `json:"difficulty"`
	Mode         string `json:"mode"`
}

type EnhancedCreateResponse struct {
	SessionID      string        `json:"session_id"`
	BackgroundURL   string        `json:"background_url"`
	SliderURL       string        `json:"slider_url"`
	GapX            int           `json:"gap_x"`
	GapY            int           `json:"gap_y"`
	ExpiresIn       int64         `json:"expires_in"`
	ExpiresAt       int64         `json:"expires_at"`
	Obstacles       []ObstacleInfo `json:"obstacles"`
	TrajectoryHint  TrajectoryHint `json:"trajectory_hint"`
	ResistanceLevel int           `json:"resistance_level"`
	Difficulty      int           `json:"difficulty"`
	TrackInfo       TrackInfo     `json:"track_info"`
}

type TrackInfo struct {
	UpperTrackY    int `json:"upper_track_y"`
	LowerTrackY    int `json:"lower_track_y"`
	HasObstacles   bool `json:"has_obstacles"`
	TrackWidth     int `json:"track_width"`
}

func NewEnhancedSliderGenerator(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *EnhancedSliderGenerator {
	return &EnhancedSliderGenerator{
		imageGenerator: NewEnhancedImageGenerator(),
		sessionCache:   sessionCache,
		captchaRepo:    captchaRepo,
		obstacleGenerator: NewObstacleGenerator(),
		trajectoryGenerator: NewTrajectoryGenerator(),
		gapDetector: NewSmartGapDetector(),
		resistanceSystem: NewAdaptiveResistanceSystem(),
	}
}

func (s *EnhancedSliderGenerator) Create(ctx context.Context, req *EnhancedCreateRequest) (*EnhancedCreateResponse, error) {
	if req.Width > 0 && req.Height > 0 {
		s.imageGenerator.SetDimensions(req.Width, req.Height, req.SliderWidth, req.SliderHeight)
	}

	difficulty := req.Difficulty
	if difficulty <= 0 {
		difficulty = 1 + rand.Intn(3)
	}

	mode := req.Mode
	if mode == "" {
		modes := []string{"standard", "dual_track", "multi_obstacle", "chaos"}
		mode = modes[rand.Intn(len(modes))]
	}

	result, err := s.imageGenerator.GenerateEnhancedSliderCaptcha(difficulty, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to generate enhanced captcha: %w", err)
	}

	sessionID := generateEnhancedSessionID()

	expiresAt := time.Now().Add(5 * time.Minute)

	resistanceLevel := s.resistanceSystem.CalculateResistanceLevel(req.Fingerprint, difficulty)

	trajectoryHint := s.trajectoryGenerator.GenerateHint(result.GapX, result.GapY, difficulty)

	session := &models.CaptchaSession{
		SessionID:   sessionID,
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		RiskScore:   0,
		TraceScore:  0,
		EnvScore:    0,
		CreatedAt:   time.Now(),
		ExpiredAt:   expiresAt,
		ClientIP:    req.ClientIP,
		UserAgent:   req.UserAgent,
		Fingerprint: req.Fingerprint,
		GapX:        result.GapX,
		GapY:        result.GapY,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	backgroundURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Background)
	sliderURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Slider)

	trackInfo := TrackInfo{
		UpperTrackY:  result.UpperTrackY,
		LowerTrackY:  result.LowerTrackY,
		HasObstacles: len(result.Obstacles) > 0,
		TrackWidth:   s.imageGenerator.height / 2,
	}

	return &EnhancedCreateResponse{
		SessionID:      sessionID,
		BackgroundURL:  backgroundURL,
		SliderURL:      sliderURL,
		GapX:           result.GapX,
		GapY:           result.GapY,
		ExpiresIn:      int64(5 * time.Minute / time.Second),
		ExpiresAt:      expiresAt.Unix(),
		Obstacles:      result.Obstacles,
		TrajectoryHint: trajectoryHint,
		ResistanceLevel: resistanceLevel,
		Difficulty:     difficulty,
		TrackInfo:      trackInfo,
	}, nil
}

func generateEnhancedSessionID() string {
	return fmt.Sprintf("enhanced_%d_%d", time.Now().UnixNano(), time.Now().UnixMicro()%10000)
}

type EnhancedImageGenerator struct {
	width        int
	height       int
	sliderWidth  int
	sliderHeight int
	imagePool    sync.Pool
	rgbaPool     sync.Pool
}

type EnhancedCaptchaResult struct {
	Background   []byte
	Slider       []byte
	GapX         int
	GapY         int
	UpperTrackY  int
	LowerTrackY  int
	Obstacles    []ObstacleInfo
	RawGapX      int
	RawGapY      int
}

func NewEnhancedImageGenerator() *EnhancedImageGenerator {
	return &EnhancedImageGenerator{
		width:        320,
		height:       200,
		sliderWidth:  50,
		sliderHeight: 50,
		imagePool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
		rgbaPool: sync.Pool{
			New: func() interface{} {
				return image.NewRGBA(image.Rect(0, 0, 320, 200))
			},
		},
	}
}

func (g *EnhancedImageGenerator) SetDimensions(width, height, sliderWidth, sliderHeight int) {
	g.width = width
	g.height = height
	g.sliderHeight = sliderHeight
	g.sliderWidth = sliderWidth
}

func (g *EnhancedImageGenerator) GenerateEnhancedSliderCaptcha(difficulty int, mode string) (*EnhancedCaptchaResult, error) {
	g.width = 320
	g.height = 200
	g.sliderWidth = 50
	g.sliderHeight = 50

	background := g.generateEnhancedBackground(difficulty)

	gapX, gapY := g.calculateSmartGap(background, difficulty)
	rawGapX, rawGapY := gapX, gapY

	upperTrackY := rand.Intn(g.height/4) + g.height/8
	lowerTrackY := g.height - rand.Intn(g.height/4) - g.height/8

	gap := image.Rect(gapX, gapY, gapX+g.sliderWidth, gapY+g.sliderHeight)

	obstacles := g.generateObstacles(gap, difficulty, mode)

	bgImage := g.applyDualTrackBackground(background, gap, upperTrackY, lowerTrackY, obstacles)

	bgImage = g.applyObstaclesToBackground(bgImage, obstacles)

	bgImage = g.applyGapWithObstacles(bgImage, gap)

	sliderImage := g.extractEnhancedSlider(background, gap, obstacles)

	bgImage = g.applyAdvancedEdgeDetection(bgImage, gap)

	bgImage = g.applyEnhancedShadowDetection(bgImage, gap)

	bgImage = g.addEnhancedInterference(bgImage, difficulty)

	bgImage = g.applyIntelligentAntiDetection(bgImage, gap, obstacles)

	bgData := g.encodePNG(bgImage)
	sliderData := g.encodePNG(sliderImage)

	return &EnhancedCaptchaResult{
		Background:  bgData,
		Slider:      sliderData,
		GapX:        gapX,
		GapY:        gapY,
		UpperTrackY: upperTrackY,
		LowerTrackY: lowerTrackY,
		Obstacles:   obstacles,
		RawGapX:     rawGapX,
		RawGapY:     rawGapY,
	}, nil
}

func (g *EnhancedImageGenerator) generateEnhancedBackground(difficulty int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, g.width, g.height))

	baseType := rand.Intn(15)
	switch baseType {
	case 0:
		g.drawAdvancedGradientBackground(img)
	case 1:
		g.drawComplexPatternBackground(img)
	case 2:
		g.drawNeuralStyleBackground(img, difficulty)
	case 3:
		g.drawMosaicBackground(img)
	case 4:
		g.drawCircuitBoardBackground(img)
	case 5:
		g.drawHexagonalBackground(img)
	case 6:
		g.drawWovenTextureBackground(img)
	case 7:
		g.drawGeometricMazeBackground(img)
	case 8:
		g.drawFractalNoiseBackground(img, difficulty)
	case 9:
		g.drawBlenderStyleBackground(img)
	case 10:
		g.drawAbstractArtBackground(img, difficulty)
	case 11:
		g.drawDataVisualizationBackground(img)
	case 12:
		g.drawTopographicBackground(img)
	case 13:
		g.drawStainedGlassBackground(img)
	case 14:
		g.drawOrganicTextureBackground(img)
	default:
		g.drawAdvancedGradientBackground(img)
	}

	return img
}

func (g *EnhancedImageGenerator) drawAdvancedGradientBackground(img *image.RGBA) {
	gradientType := rand.Intn(4)
	
	var r1, g1, b1, r2, g2, b2, r3, g3, b3 uint8
	r1 = uint8(120 + rand.Intn(100))
	g1 = uint8(120 + rand.Intn(100))
	b1 = uint8(120 + rand.Intn(100))
	r2 = uint8(80 + rand.Intn(80))
	g2 = uint8(80 + rand.Intn(80))
	b2 = uint8(80 + rand.Intn(80))
	r3 = uint8(100 + rand.Intn(60))
	g3 = uint8(100 + rand.Intn(60))
	b3 = uint8(100 + rand.Intn(60))

	switch gradientType {
	case 0:
		for y := 0; y < g.height; y++ {
			t := float64(y) / float64(g.height)
			for x := 0; x < g.width; x++ {
				t2 := float64(x) / float64(g.width)
				r := uint8(float64(r1)*(1-t) + float64(r2)*t)
				col := uint8(float64(g1)*(1-t) + float64(g2)*t)
				b := uint8(float64(b1)*(1-t) + float64(b2)*t)
				r = uint8(float64(r)*(1-t2) + float64(r3)*t2)
				col = uint8(float64(col)*(1-t2) + float64(g3)*t2)
				b = uint8(float64(b)*(1-t2) + float64(b3)*t2)
				img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
			}
		}
	case 1:
		centerX := g.width / 2
		centerY := g.height / 2
		for y := 0; y < g.height; y++ {
			for x := 0; x < g.width; x++ {
				dx := float64(x - centerX)
				dy := float64(y - centerY)
				dist := math.Sqrt(dx*dx + dy*dy)
				maxDist := math.Sqrt(float64(centerX*centerX + centerY*centerY))
				t := dist / maxDist
				
				r := uint8(float64(r1)*(1-t) + float64(r2)*t)
				col := uint8(float64(g1)*(1-t) + float64(g2)*t)
				b := uint8(float64(b1)*(1-t) + float64(b2)*t)
				img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
			}
		}
	case 2:
		for y := 0; y < g.height; y++ {
			t := float64(y) / float64(g.height)
			t2 := math.Sin(float64(y)*0.02) * 0.3 + 0.5
			for x := 0; x < g.width; x++ {
				t3 := float64(x) / float64(g.width)
				t4 := math.Cos(float64(x)*0.015) * 0.3 + 0.5
				
				combined := t*0.4 + t2*0.3 + t3*0.15 + t4*0.15
				
				r := uint8(float64(r1)*(1-combined) + float64(r2)*combined)
				col := uint8(float64(g1)*(1-combined) + float64(g2)*combined)
				b := uint8(float64(b1)*(1-combined) + float64(b2)*combined)
				img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
			}
		}
	default:
		g.drawGradientBackground(img)
	}
}

func (g *EnhancedImageGenerator) drawGradientBackground(img *image.RGBA) {
	r1 := uint8(180 + rand.Intn(60))
	g1 := uint8(180 + rand.Intn(60))
	b1 := uint8(200 + rand.Intn(55))
	r2 := uint8(100 + rand.Intn(80))
	g2 := uint8(120 + rand.Intn(60))
	b2 := uint8(160 + rand.Intn(60))

	for y := 0; y < g.height; y++ {
		t := float64(y) / float64(g.height)
		r := uint8(float64(r1)*(1-t) + float64(r2)*t)
		col := uint8(float64(g1)*(1-t) + float64(g2)*t)
		b := uint8(float64(b1)*(1-t) + float64(b2)*t)
		for x := 0; x < g.width; x++ {
			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}
}

func (g *EnhancedImageGenerator) drawComplexPatternBackground(img *image.RGBA) {
	r1 := uint8(60 + rand.Intn(40))
	g1 := uint8(80 + rand.Intn(40))
	b1 := uint8(100 + rand.Intn(40))
	r2 := uint8(140 + rand.Intn(60))
	g2 := uint8(160 + rand.Intn(60))
	b2 := uint8(180 + rand.Intn(60))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			patternValue := math.Sin(float64(x)*0.05) * math.Cos(float64(y)*0.05)
			ratio := (patternValue + 1) / 2

			r := uint8(float64(r1)*(1-ratio) + float64(r2)*ratio)
			col := uint8(float64(g1)*(1-ratio) + float64(g2)*ratio)
			b := uint8(float64(b1)*(1-ratio) + float64(b2)*ratio)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}
}

func (g *EnhancedImageGenerator) drawNeuralStyleBackground(img *image.RGBA, difficulty int) {
	baseR := uint8(100 + rand.Intn(80))
	baseG := uint8(100 + rand.Intn(80))
	baseB := uint8(100 + rand.Intn(80))

	style := rand.Intn(5)
	switch style {
	case 0:
		g.drawConvolutionalNeuralStyle(img, baseR, baseG, baseB, difficulty)
	case 1:
		g.drawGenerativeAdversarialStyle(img, baseR, baseG, baseB, difficulty)
	case 2:
		g.drawTransformerStyle(img, baseR, baseG, baseB, difficulty)
	case 3:
		g.drawCapsuleNetworkStyle(img, baseR, baseG, baseB, difficulty)
	case 4:
		g.drawAttentionBasedStyle(img, baseR, baseG, baseB, difficulty)
	default:
		g.drawConvolutionalNeuralStyle(img, baseR, baseG, baseB, difficulty)
	}
}

func (g *EnhancedImageGenerator) drawConvolutionalNeuralStyle(img *image.RGBA, baseR, baseG, baseB uint8, difficulty int) {
	layers := 3 + difficulty
	kernelSize := 3 + difficulty*2

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			featureMaps := make([]float64, layers)
			
			for l := 0; l < layers; l++ {
				offsetX := (l * 7) % kernelSize
				offsetY := (l * 11) % kernelSize
				
				convValue := math.Sin(float64(x+offsetX)*0.03*float64(l+1)) * 
				             math.Cos(float64(y+offsetY)*0.03*float64(l+1))
				
				activation := 1.0 / (1.0 + math.Exp(-convValue))
				featureMaps[l] = activation
			}

			combined := 0.0
			for _, fm := range featureMaps {
				combined += fm
			}
			combined /= float64(layers)

			adjustment := int(combined * 80 - 40)
			r := g.clampUint8(int(baseR) + adjustment)
			col := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}

	g.applyPoolingEffect(img, difficulty)
}

func (g *EnhancedImageGenerator) drawGenerativeAdversarialStyle(img *image.RGBA, baseR, baseG, baseB uint8, difficulty int) {
	noiseSeed := rand.Float64() * 1000

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			latentX := float64(x) / 50.0
			latentY := float64(y) / 50.0
			latentZ := noiseSeed

			ganValue := g.ganForward(latentX, latentY, latentZ)
			
			adjustment := int(ganValue * 100 - 50)
			r := g.clampUint8(int(baseR) + adjustment)
			col := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}

	g.applyBatchNormalization(img)
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (g *EnhancedImageGenerator) ganForward(x, y, z float64) float64 {
	layer1 := math.Tanh(x*0.5 + y*0.3 + z*0.2)
	layer2 := sigmoid(layer1*2.0 + math.Sin(z)*0.5)
	layer3 := math.Tanh(layer2*1.5 + layer1*0.5)
	return layer3
}

func (g *EnhancedImageGenerator) drawTransformerStyle(img *image.RGBA, baseR, baseG, baseB uint8, difficulty int) {
	heads := 4 + difficulty*2
	seqLen := 20

	positionEncoding := make([][]float64, seqLen)
	for i := range positionEncoding {
		positionEncoding[i] = make([]float64, heads)
		for h := 0; h < heads; h++ {
			if h%2 == 0 {
				positionEncoding[i][h] = math.Sin(float64(i) / math.Pow(10000, float64(h)/float64(heads)))
			} else {
				positionEncoding[i][h] = math.Cos(float64(i) / math.Pow(10000, float64(h)/float64(heads)))
			}
		}
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			posX := (x * seqLen) / g.width
			posY := (y * seqLen) / g.height
			
			if posX >= seqLen {
				posX = seqLen - 1
			}
			if posY >= seqLen {
				posY = seqLen - 1
			}

			attentionScore := 0.0
			for h := 0; h < heads; h++ {
				attentionScore += positionEncoding[posX][h] * positionEncoding[posY][h]
			}
			attentionScore /= float64(heads)

			adjustment := int(attentionScore * 80 - 40)
			r := g.clampUint8(int(baseR) + adjustment)
			col := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}

	g.applySoftmaxNormalization(img)
}

func (g *EnhancedImageGenerator) drawCapsuleNetworkStyle(img *image.RGBA, baseR, baseG, baseB uint8, difficulty int) {
	capsules := 8 + difficulty*4

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			primaryCaps := make([][]float64, capsules)
			for i := range primaryCaps {
				primaryCaps[i] = make([]float64, 8)
				for j := range primaryCaps[i] {
					primaryCaps[i][j] = math.Sin(float64(x+i)*0.05+float64(j)*0.3) * 
					                   math.Cos(float64(y+i)*0.05+float64(j)*0.3)
				}
			}

			routingWeight := make([]float64, capsules)
			for i := range routingWeight {
				routingWeight[i] = 1.0 / float64(capsules)
			}

			for iter := 0; iter < 3; iter++ {
				for i := range routingWeight {
					routingWeight[i] = math.Exp(routingWeight[i]) / float64(capsules)
				}
			}

			capsuleOutput := 0.0
			for i := 0; i < capsules; i++ {
				capsuleMag := 0.0
				for j := range primaryCaps[i] {
					capsuleMag += primaryCaps[i][j] * primaryCaps[i][j]
				}
				capsuleMag = math.Sqrt(capsuleMag)
				capsuleOutput += routingWeight[i] * capsuleMag
			}
			capsuleOutput /= float64(capsules)

			adjustment := int(capsuleOutput * 80 - 40)
			r := g.clampUint8(int(baseR) + adjustment)
			col := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}
}

func (g *EnhancedImageGenerator) drawAttentionBasedStyle(img *image.RGBA, baseR, baseG, baseB uint8, difficulty int) {
	queryPoints := make([][]float64, difficulty+2)
	for i := range queryPoints {
		queryPoints[i] = []float64{
			float64(rand.Intn(g.width)),
			float64(rand.Intn(g.height)),
		}
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			attentionSum := 0.0
			weightSum := 0.0

			for _, query := range queryPoints {
				dx := float64(x) - query[0]
				dy := float64(y) - query[1]
				dist := math.Sqrt(dx*dx + dy*dy)
				
				attention := math.Exp(-dist*dist / 2000.0)
				
				attentionSum += attention * math.Sin(query[0]*0.01+query[1]*0.01)
				weightSum += attention
			}

			if weightSum > 0 {
				attentionSum /= weightSum
			}

			adjustment := int(attentionSum * 80 - 40)
			r := g.clampUint8(int(baseR) + adjustment)
			col := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}
}

func (g *EnhancedImageGenerator) applyPoolingEffect(img *image.RGBA, poolSize int) {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	poolStride := poolSize
	for y := 0; y < g.height; y += poolStride {
		for x := 0; x < g.width; x += poolStride {
			maxR, maxG, maxB := 0, 0, 0
			
			for py := 0; py < poolStride && y+py < g.height; py++ {
				for px := 0; px < poolStride && x+px < g.width; px++ {
					p := img.RGBAAt(x+px, y+py)
					if int(p.R) > maxR {
						maxR = int(p.R)
					}
					if int(p.G) > maxG {
						maxG = int(p.G)
					}
					if int(p.B) > maxB {
						maxB = int(p.B)
					}
				}
			}

			for py := 0; py < poolStride && y+py < g.height; py++ {
				for px := 0; px < poolStride && x+px < g.width; px++ {
					result.Set(x+px, y+py, color.RGBA{
						R: uint8(maxR),
						G: uint8(maxG),
						B: uint8(maxB),
						A: 255,
					})
				}
			}
		}
	}

	draw.Draw(img, img.Bounds(), result, result.Bounds().Min, draw.Src)
}

func (g *EnhancedImageGenerator) applyBatchNormalization(img *image.RGBA) {
	var sumR, sumG, sumB float64
	count := float64(g.width * g.height)

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			p := img.RGBAAt(x, y)
			sumR += float64(p.R)
			sumG += float64(p.G)
			sumB += float64(p.B)
		}
	}

	meanR := sumR / count
	meanG := sumG / count
	meanB := sumB / count

	var varR, varG, varB float64
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			p := img.RGBAAt(x, y)
			varR += (float64(p.R) - meanR) * (float64(p.R) - meanR)
			varG += (float64(p.G) - meanG) * (float64(p.G) - meanG)
			varB += (float64(p.B) - meanB) * (float64(p.B) - meanB)
		}
	}

	stdR := math.Sqrt(varR / count)
	stdG := math.Sqrt(varG / count)
	stdB := math.Sqrt(varB / count)

	gamma := 1.0
	beta := 128.0

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			p := img.RGBAAt(x, y)
			
			newR := gamma*(float64(p.R)-meanR)/stdR + beta
			newG := gamma*(float64(p.G)-meanG)/stdG + beta
			newB := gamma*(float64(p.B)-meanB)/stdB + beta

			img.Set(x, y, color.RGBA{
				R: g.clampUint8(int(newR)),
				G: g.clampUint8(int(newG)),
				B: g.clampUint8(int(newB)),
				A: 255,
			})
		}
	}
}

func (g *EnhancedImageGenerator) applySoftmaxNormalization(img *image.RGBA) {
	values := make([]float64, g.width*g.height)
	maxVal := float64(-1e9)

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			p := img.RGBAAt(x, y)
			val := float64(p.R) + float64(p.G) + float64(p.B)
			values[y*g.width+x] = val
			if val > maxVal {
				maxVal = val
			}
		}
	}

	expSum := 0.0
	for i := range values {
		values[i] = math.Exp(values[i] - maxVal)
		expSum += values[i]
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			normalized := values[y*g.width+x] / expSum
			adjusted := int(normalized * 255)
			
			p := img.RGBAAt(x, y)
			img.Set(x, y, color.RGBA{
				R: g.clampUint8(int(float64(p.R)*0.5) + adjusted/3),
				G: g.clampUint8(int(float64(p.G)*0.5) + adjusted/3),
				B: g.clampUint8(int(float64(p.B)*0.5) + adjusted/3),
				A: 255,
			})
		}
	}
}

func (g *EnhancedImageGenerator) drawMosaicBackground(img *image.RGBA) {
	tileSize := 10 + rand.Intn(15)
	tiles := make([][]color.RGBA, (g.height+tileSize-1)/tileSize)
	for i := range tiles {
		tiles[i] = make([]color.RGBA, (g.width+tileSize-1)/tileSize)
		for j := range tiles[i] {
			tiles[i][j] = color.RGBA{
				R: uint8(80 + rand.Intn(120)),
				G: uint8(80 + rand.Intn(120)),
				B: uint8(80 + rand.Intn(120)),
				A: 255,
			}
		}
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			tileX := x / tileSize
			tileY := y / tileSize
			img.Set(x, y, tiles[tileY][tileX])
		}
	}

	g.addMosaicGrout(img, tileSize)
}

func (g *EnhancedImageGenerator) addMosaicGrout(img *image.RGBA, tileSize int) {
	groutColor := color.RGBA{
		R: uint8(40 + rand.Intn(30)),
		G: uint8(40 + rand.Intn(30)),
		B: uint8(40 + rand.Intn(30)),
		A: 255,
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			if x%tileSize == 0 || y%tileSize == 0 {
				img.Set(x, y, groutColor)
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawCircuitBoardBackground(img *image.RGBA) {
	baseColor := color.RGBA{
		R: uint8(20 + rand.Intn(20)),
		G: uint8(60 + rand.Intn(40)),
		B: uint8(20 + rand.Intn(20)),
		A: 255,
	}

	draw.Draw(img, img.Bounds(), &image.Uniform{C: baseColor}, image.Point{}, draw.Src)

	traces := 20 + rand.Intn(30)
	for i := 0; i < traces; i++ {
		x := rand.Intn(g.width)
		y := rand.Intn(g.height)
		length := 20 + rand.Intn(80)
		direction := rand.Intn(4)
		width := 2 + rand.Intn(3)

		traceColor := color.RGBA{
			R: uint8(180 + rand.Intn(50)),
			G: uint8(140 + rand.Intn(60)),
			B: uint8(60 + rand.Intn(40)),
			A: 255,
		}

		switch direction {
		case 0:
			g.drawFilledRect(img, x, y, length, width, traceColor)
		case 1:
			g.drawFilledRect(img, x, y, width, length, traceColor)
		case 2:
			g.drawDiagonalLine(img, x, y, x+length, y+length, width, traceColor)
		case 3:
			g.drawDiagonalLine(img, x+length, y, x, y+length, width, traceColor)
		}
	}

	pads := 10 + rand.Intn(20)
	for i := 0; i < pads; i++ {
		x := rand.Intn(g.width - 10)
		y := rand.Intn(g.height - 10)
		radius := 3 + rand.Intn(5)

		padColor := color.RGBA{
			R: uint8(200 + rand.Intn(30)),
			G: uint8(160 + rand.Intn(50)),
			B: uint8(40 + rand.Intn(40)),
			A: 255,
		}

		g.drawFilledCircle(img, x, y, radius, padColor)
	}
}

func (g *EnhancedImageGenerator) drawHexagonalBackground(img *image.RGBA) {
	hexSize := 15 + rand.Intn(10)
	baseColor := uint8(80 + rand.Intn(60))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			rowOffset := (y / (hexSize * 3)) % 2 * hexSize
			localX := (x + rowOffset) % (hexSize * 2)
			localY := y % (hexSize * 3)

			distToCenter := math.Sqrt(
				math.Pow(float64(localX-hexSize), 2) +
				math.Pow(float64(localY)-float64(hexSize)*1.5, 2),
			)

			hexRadius := float64(hexSize) * 1.1
			if distToCenter < hexRadius {
				brightness := 1.0 - (distToCenter / hexRadius) * 0.3
				r := uint8(float64(baseColor) * brightness * (1.0 + float64(rand.Intn(30))/100))
				gc := uint8(float64(baseColor) * brightness * (1.0 + float64(rand.Intn(30))/100))
				b := uint8(float64(baseColor) * brightness * (1.0 + float64(rand.Intn(30))/100))
				img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
			} else {
				img.Set(x, y, color.RGBA{
					R: baseColor - 30,
					G: baseColor - 20,
					B: baseColor - 10,
					A: 255,
				})
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawWovenTextureBackground(img *image.RGBA) {
	baseR := uint8(120 + rand.Intn(60))
	baseG := uint8(100 + rand.Intn(60))
	baseB := uint8(80 + rand.Intn(60))

	threadWidth := 4 + rand.Intn(4)

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			warpY := y / threadWidth
			weftX := x / threadWidth
			
			warpOver := (warpY + weftX) % 2

			var threadColor color.RGBA
			if warpOver == 0 {
				threadColor = color.RGBA{
					R: baseR,
					G: baseG,
					B: baseB,
					A: 255,
				}
			} else {
				threadColor = color.RGBA{
					R: baseR - 30,
					G: baseG - 20,
					B: baseB - 15,
					A: 255,
				}
			}

			threadBrightness := 1.0
			if x%threadWidth == 0 || x%threadWidth == threadWidth-1 {
				threadBrightness = 0.85
			}
			if y%threadWidth == 0 || y%threadWidth == threadWidth-1 {
				threadBrightness = 0.85
			}

			img.Set(x, y, color.RGBA{
				R: g.clampUint8(int(float64(threadColor.R) * threadBrightness)),
				G: g.clampUint8(int(float64(threadColor.G) * threadBrightness)),
				B: g.clampUint8(int(float64(threadColor.B) * threadBrightness)),
				A: 255,
			})
		}
	}
}

func (g *EnhancedImageGenerator) drawGeometricMazeBackground(img *image.RGBA) {
	cellSize := 20 + rand.Intn(15)
	wallWidth := 2 + rand.Intn(3)

	baseColor := uint8(100 + rand.Intn(80))
	wallColor := color.RGBA{
		R: uint8(baseColor - 40),
		G: uint8(baseColor - 30),
		B: uint8(baseColor - 20),
		A: 255,
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			img.Set(x, y, color.RGBA{
				R: baseColor,
				G: baseColor,
				B: baseColor,
				A: 255,
			})
		}
	}

	for cy := 0; cy < g.height/cellSize; cy++ {
		for cx := 0; cx < g.width/cellSize; cx++ {
			if rand.Float64() > 0.5 {
				g.drawFilledRect(img, cx*cellSize, cy*cellSize, cellSize, wallWidth, wallColor)
			}
			if rand.Float64() > 0.5 {
				g.drawFilledRect(img, cx*cellSize, cy*cellSize, wallWidth, cellSize, wallColor)
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawFractalNoiseBackground(img *image.RGBA, difficulty int) {
	baseR := uint8(100 + rand.Intn(80))
	baseG := uint8(100 + rand.Intn(80))
	baseB := uint8(100 + rand.Intn(80))

	octaves := 4 + difficulty*2
	persistence := 0.5 + float64(difficulty)*0.1

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise := 0.0
			amplitude := 1.0
			frequency := 0.02
			maxValue := 0.0

			for o := 0; o < octaves; o++ {
				noiseValue := g.simplexNoise3D(float64(x)*frequency, float64(y)*frequency, float64(o)*10.5)
				noise += noiseValue * amplitude
				maxValue += amplitude
				amplitude *= persistence
				frequency *= 2.0
			}

			noise = noise / maxValue
			noise = (noise + 1) / 2

			adjustment := int(noise*100 - 50)
			r := g.clampUint8(int(baseR) + adjustment)
			col := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}
}

func (g *EnhancedImageGenerator) simplexNoise3D(x, y, z float64) float64 {
	F3 := 1.0 / 3.0
	G3 := 1.0 / 6.0

	s := (x + y + z) * F3
	i := int(math.Floor(x + s))
	j := int(math.Floor(y + s))
	k := int(math.Floor(z + s))

	t := float64(i+j+k) * G3
	X0 := float64(i) - t
	Y0 := float64(j) - t
	Z0 := float64(k) - t
	x0 := x - X0
	y0 := y - Y0
	z0 := z - Z0

	var i1, j1, k1, i2, j2, k2 int
	if x0 >= y0 {
		if y0 >= z0 {
			i1, j1, k1, i2, j2, k2 = 1, 0, 0, 1, 1, 0
		} else if x0 >= z0 {
			i1, j1, k1, i2, j2, k2 = 1, 0, 0, 1, 0, 1
		} else {
			i1, j1, k1, i2, j2, k2 = 0, 0, 1, 1, 0, 1
		}
	} else {
		if y0 < z0 {
			i1, j1, k1, i2, j2, k2 = 0, 0, 1, 0, 1, 1
		} else if x0 < z0 {
			i1, j1, k1, i2, j2, k2 = 0, 1, 0, 0, 1, 1
		} else {
			i1, j1, k1, i2, j2, k2 = 0, 1, 0, 1, 1, 0
		}
	}

	x1 := x0 - float64(i1) + G3
	y1 := y0 - float64(j1) + G3
	z1 := z0 - float64(k1) + G3
	x2 := x0 - float64(i2) + 2*G3
	y2 := y0 - float64(j2) + 2*G3
	z2 := z0 - float64(k2) + 2*G3
	x3 := x0 - 1 + 3*G3
	y3 := y0 - 1 + 3*G3
	z3 := z0 - 1 + 3*G3

	gi0 := g.hash3D(i, j, k) % 12
	gi1 := g.hash3D(i+i1, j+j1, k+k1) % 12
	gi2 := g.hash3D(i+i2, j+j2, k+k2) % 12
	gi3 := g.hash3D(i+1, j+1, k+1) % 12

	grad3 := [12][]float64{
		{1, 1, 0}, {-1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
		{1, 0, 1}, {-1, 0, 1}, {1, 0, -1}, {-1, 0, -1},
		{0, 1, 1}, {0, -1, 1}, {0, 1, -1}, {0, -1, -1},
	}

	n0, n1, n2, n3 := 0.0, 0.0, 0.0, 0.0

	t0 := 0.6 - x0*x0 - y0*y0 - z0*z0
	if t0 >= 0 {
		t0 *= t0
		n0 = t0 * t0 * g.dot3(grad3[gi0], x0, y0, z0)
	}

	t1 := 0.6 - x1*x1 - y1*y1 - z1*z1
	if t1 >= 0 {
		t1 *= t1
		n1 = t1 * t1 * g.dot3(grad3[gi1], x1, y1, z1)
	}

	t2 := 0.6 - x2*x2 - y2*y2 - z2*z2
	if t2 >= 0 {
		t2 *= t2
		n2 = t2 * t2 * g.dot3(grad3[gi2], x2, y2, z2)
	}

	t3 := 0.6 - x3*x3 - y3*y3 - z3*z3
	if t3 >= 0 {
		t3 *= t3
		n3 = t3 * t3 * g.dot3(grad3[gi3], x3, y3, z3)
	}

	return 32.0 * (n0 + n1 + n2 + n3)
}

func (g *EnhancedImageGenerator) hash3D(i, j, k int) int {
	return (i*574861893 + j*1274126187 + k*2530992493) ^ 
	       (i*2246770379 + j*345658321 + k*987654321)
}

func (g *EnhancedImageGenerator) dot3(grad []float64, x, y, z float64) float64 {
	return grad[0]*x + grad[1]*y + grad[2]*z
}

func (g *EnhancedImageGenerator) drawBlenderStyleBackground(img *image.RGBA) {
	nodes := 3 + rand.Intn(5)
	nodePositions := make([]struct{ x, y int }, nodes)
	nodeColors := make([]color.RGBA, nodes)

	for i := 0; i < nodes; i++ {
		nodePositions[i] = struct{ x, y int }{
			x: rand.Intn(g.width),
			y: rand.Intn(g.height),
		}
		nodeColors[i] = color.RGBA{
			R: uint8(80 + rand.Intn(120)),
			G: uint8(80 + rand.Intn(120)),
			B: uint8(80 + rand.Intn(120)),
			A: 255,
		}
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			var totalR, totalG, totalB float64
			totalWeight := 0.0

			for i := 0; i < nodes; i++ {
				dx := float64(x) - float64(nodePositions[i].x)
				dy := float64(y) - float64(nodePositions[i].y)
				dist := math.Sqrt(dx*dx + dy*dy)

				weight := 1.0 / (1.0 + dist*0.05)
				totalWeight += weight

				totalR += float64(nodeColors[i].R) * weight
				totalG += float64(nodeColors[i].G) * weight
				totalB += float64(nodeColors[i].B) * weight
			}

			r := g.clampUint8(int(totalR / totalWeight))
			col := g.clampUint8(int(totalG / totalWeight))
			b := g.clampUint8(int(totalB / totalWeight))

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}
}

func (g *EnhancedImageGenerator) drawAbstractArtBackground(img *image.RGBA, difficulty int) {
	shapes := 5 + difficulty*3

	for i := 0; i < shapes; i++ {
		shapeType := rand.Intn(4)
		x := rand.Intn(g.width)
		y := rand.Intn(g.height)
		size := 20 + rand.Intn(50)

		shapeColor := color.RGBA{
			R: uint8(rand.Intn(256)),
			G: uint8(rand.Intn(256)),
			B: uint8(rand.Intn(256)),
			A: uint8(100 + rand.Intn(100)),
		}

		switch shapeType {
		case 0:
			g.drawFilledCircle(img, x, y, size, shapeColor)
		case 1:
			g.drawFilledRect(img, x-size/2, y-size/2, size, size, shapeColor)
		case 2:
			g.drawTriangle(img, x, y-size, x-size, y+size, x+size, y+size, shapeColor)
		case 3:
			g.drawEllipse(img, x, y, size, size/2, shapeColor)
		}
	}
}

func (g *EnhancedImageGenerator) drawDataVisualizationBackground(img *image.RGBA) {
	baseColor := uint8(40 + rand.Intn(30))
	
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			img.Set(x, y, color.RGBA{
				R: baseColor,
				G: baseColor + 20,
				B: baseColor + 10,
				A: 255,
			})
		}
	}

	dataPoints := 20 + rand.Intn(30)
	points := make([]struct{ x, y float64 }, dataPoints)
	for i := range points {
		points[i] = struct{ x, y float64 }{
			x: float64(i * g.width / dataPoints),
			y: float64(rand.Intn(g.height)),
		}
	}

	lineColor := color.RGBA{
		R: uint8(100 + rand.Intn(80)),
		G: uint8(180 + rand.Intn(60)),
		B: uint8(80 + rand.Intn(80)),
		A: 255,
	}

	for i := 0; i < len(points)-1; i++ {
		g.drawDiagonalLine(img, int(points[i].x), int(points[i].y), 
			int(points[i+1].x), int(points[i+1].y), 2, lineColor)
	}

	for _, p := range points {
		dotColor := color.RGBA{
			R: uint8(200 + rand.Intn(55)),
			G: uint8(100 + rand.Intn(80)),
			B: uint8(100 + rand.Intn(80)),
			A: 255,
		}
		g.drawFilledCircle(img, int(p.x), int(p.y), 4, dotColor)
	}
}

func (g *EnhancedImageGenerator) drawTopographicBackground(img *image.RGBA) {
	baseColor := uint8(100 + rand.Intn(60))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			elevation := 0.0
			
			for i := 0; i < 3; i++ {
				freq := 0.02 + float64(i)*0.01
				elevation += math.Sin(float64(x)*freq+float64(i)*100) * 
				           math.Cos(float64(y)*freq+float64(i)*50)
			}
			
			elevation = (elevation + 3) / 6
			elevation = math.Max(0, math.Min(1, elevation))

			level := int(elevation * 10)
			brightness := baseColor + uint8(level*8)

			img.Set(x, y, color.RGBA{
				R: brightness,
				G: brightness + 10,
				B: brightness + 5,
				A: 255,
			})
		}
	}
}

func (g *EnhancedImageGenerator) drawStainedGlassBackground(img *image.RGBA) {
	segments := 8 + rand.Intn(8)
	centers := make([]struct{ x, y int }, segments)
	colors := make([]color.RGBA, segments)

	for i := 0; i < segments; i++ {
		centers[i] = struct{ x, y int }{
			x: rand.Intn(g.width),
			y: rand.Intn(g.height),
		}
		colors[i] = color.RGBA{
			R: uint8(80 + rand.Intn(120)),
			G: uint8(80 + rand.Intn(120)),
			B: uint8(80 + rand.Intn(120)),
			A: 255,
		}
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			minDist := float64(g.width + g.height)
			closestIdx := 0

			for i := 0; i < segments; i++ {
				dx := float64(x) - float64(centers[i].x)
				dy := float64(y) - float64(centers[i].y)
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < minDist {
					minDist = dist
					closestIdx = i
				}
			}

			img.Set(x, y, colors[closestIdx])
		}
	}

	leadWidth := 2
	leadColor := color.RGBA{
		R: 30,
		G: 30,
		B: 30,
		A: 255,
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			if x%40 < leadWidth || y%40 < leadWidth {
				img.Set(x, y, leadColor)
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawOrganicTextureBackground(img *image.RGBA) {
	baseR := uint8(80 + rand.Intn(60))
	baseG := uint8(100 + rand.Intn(60))
	baseB := uint8(60 + rand.Intn(60))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise := g.perlinNoise2D(float64(x)*0.05, float64(y)*0.05)
			bubbles := g.cellularNoise(float64(x)*0.03, float64(y)*0.03)

			combined := noise*0.6 + bubbles*0.4
			adjustment := int(combined*60 - 30)

			r := g.clampUint8(int(baseR) + adjustment)
			col := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
		}
	}
}

func (g *EnhancedImageGenerator) perlinNoise2D(x, y float64) float64 {
	return g.simplexNoise3D(x, y, 0)
}

func (g *EnhancedImageGenerator) cellularNoise(x, y float64) float64 {
	cells := 5
	minDist := float64(cells * cells)

	for i := 0; i < cells; i++ {
		for j := 0; j < cells; j++ {
			cellX := float64(i) * 20
			cellY := float64(j) * 20
			dist := math.Sqrt(math.Pow(x-cellX, 2) + math.Pow(y-cellY, 2))
			if dist < minDist {
				minDist = dist
			}
		}
	}

	return minDist / 50.0
}

func (g *EnhancedImageGenerator) drawTriangle(img *image.RGBA, x1, y1, x2, y2, x3, y3 int, c color.RGBA) {
	minX := min(x1, min(x2, x3))
	maxX := max(x1, max(x2, x3))
	minY := min(y1, min(y2, y3))
	maxY := max(y1, max(y2, y3))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if g.pointInTriangle(x, y, x1, y1, x2, y2, x3, y3) {
				if x >= 0 && x < g.width && y >= 0 && y < g.height {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) pointInTriangle(px, py, x1, y1, x2, y2, x3, y3 int) bool {
	d1 := g.sign(px, py, x1, y1, x2, y2)
	d2 := g.sign(px, py, x2, y2, x3, y3)
	d3 := g.sign(px, py, x3, y3, x1, y1)

	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)

	return !(hasNeg && hasPos)
}

func (g *EnhancedImageGenerator) sign(px, py, x1, y1, x2, y2 int) int {
	return (px-x1)*(y2-y1) - (x2-x1)*(py-y1)
}

func (g *EnhancedImageGenerator) drawEllipse(img *image.RGBA, cx, cy, rx, ry int, c color.RGBA) {
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx-rx; x <= cx+rx; x++ {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if dx*dx+dy*dy <= 1 {
				if x >= 0 && x < g.width && y >= 0 && y < g.height {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) calculateSmartGap(background *image.RGBA, difficulty int) (int, int) {
	detector := NewSmartGapDetector()
	
	gapX, gapY := detector.DetectOptimalGap(background, difficulty)
	
	if gapX < g.sliderWidth+10 {
		gapX = g.sliderWidth + 10 + rand.Intn(50)
	}
	if gapX > g.width-g.sliderWidth-10 {
		gapX = g.width - g.sliderWidth - 10 - rand.Intn(50)
	}
	
	if gapY < g.sliderHeight+10 {
		gapY = g.sliderHeight + 10 + rand.Intn(g.height/2)
	}
	if gapY > g.height-g.sliderHeight-10 {
		gapY = g.height - g.sliderHeight - 10 - rand.Intn(g.height/2)
	}

	return gapX, gapY
}

func (g *EnhancedImageGenerator) generateObstacles(gap image.Rectangle, difficulty int, mode string) []ObstacleInfo {
	var obstacles []ObstacleInfo

	obstacleCount := 0
	switch mode {
	case "multi_obstacle":
		obstacleCount = difficulty + 2
	case "chaos":
		obstacleCount = difficulty*2 + 3
	case "dual_track":
		obstacleCount = difficulty
	default:
		obstacleCount = rand.Intn(difficulty + 1)
	}

	for i := 0; i < obstacleCount; i++ {
		obstacle := ObstacleInfo{
			Type:     g.getRandomObstacleType(),
			X:        rand.Intn(g.width - 60),
			Y:        rand.Intn(g.height - 60),
			Width:    30 + rand.Intn(40),
			Height:   30 + rand.Intn(40),
			Rotation: rand.Intn(4) * 90,
		}

		if !g.obstacleOverlapsWithGap(obstacle, gap) {
			obstacles = append(obstacles, obstacle)
		}
	}

	return obstacles
}

func (g *EnhancedImageGenerator) getRandomObstacleType() string {
	types := []string{"barrier", "zigzag", "curve", "bump", "hole", "trap"}
	return types[rand.Intn(len(types))]
}

func (g *EnhancedImageGenerator) obstacleOverlapsWithGap(obstacle ObstacleInfo, gap image.Rectangle) bool {
	obstacleRect := image.Rect(obstacle.X, obstacle.Y, obstacle.X+obstacle.Width, obstacle.Y+obstacle.Height)
	
	margin := 20
	expandedGap := image.Rect(
		gap.Min.X-margin,
		gap.Min.Y-margin,
		gap.Max.X+margin,
		gap.Max.Y+margin,
	)

	return obstacleRect.Overlaps(expandedGap)
}

func (g *EnhancedImageGenerator) applyDualTrackBackground(background *image.RGBA, gap image.Rectangle, upperTrackY, lowerTrackY int, obstacles []ObstacleInfo) *image.RGBA {
	result := image.NewRGBA(background.Bounds())
	draw.Draw(result, result.Bounds(), background, background.Bounds().Min, draw.Src)

	trackWidth := g.height / 4

	g.drawTrackIndicator(result, upperTrackY-trackWidth/2, trackWidth, "upper", obstacles)
	g.drawTrackIndicator(result, lowerTrackY-trackWidth/2, trackWidth, "lower", obstacles)

	return result
}

func (g *EnhancedImageGenerator) drawTrackIndicator(img *image.RGBA, y, width int, trackType string, obstacles []ObstacleInfo) {
	if y < 0 {
		y = 0
	}
	if y+width > g.height {
		width = g.height - y
	}

	for py := y; py < y+width; py++ {
		for px := 0; px < g.width; px++ {
			p := img.RGBAAt(px, py)
			
			distToCenter := math.Abs(float64(py - (y + width/2)))
			normalizedDist := distToCenter / float64(width/2)
			
			alpha := uint8((1.0 - normalizedDist*0.7) * 30)
			
			img.Set(px, py, color.RGBA{
				R: p.R,
				G: p.G,
				B: p.B,
				A: 255 - alpha,
			})
		}
	}

	if trackType == "upper" || trackType == "lower" {
		lineY := y + width/2
		if lineY < g.height {
			for x := 0; x < g.width; x++ {
				p := img.RGBAAt(x, lineY)
				img.Set(x, lineY, color.RGBA{
					R: p.R,
					G: uint8(clampInt(int(p.G)+20, 0, 255)),
					B: uint8(clampInt(int(p.B)+20, 0, 255)),
					A: 255,
				})
			}
		}
	}
}

func (g *EnhancedImageGenerator) applyObstaclesToBackground(img *image.RGBA, obstacles []ObstacleInfo) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	for _, obs := range obstacles {
		switch obs.Type {
		case "barrier":
			g.drawBarrierObstacle(result, obs)
		case "zigzag":
			g.drawZigzagObstacle(result, obs)
		case "curve":
			g.drawCurveObstacle(result, obs)
		case "bump":
			g.drawBumpObstacle(result, obs)
		case "hole":
			g.drawHoleObstacle(result, obs)
		case "trap":
			g.drawTrapObstacle(result, obs)
		}
	}

	return result
}

func (g *EnhancedImageGenerator) drawBarrierObstacle(img *image.RGBA, obs ObstacleInfo) {
	barrierColor := color.RGBA{
		R: uint8(200 + rand.Intn(55)),
		G: uint8(80 + rand.Intn(80)),
		B: uint8(60 + rand.Intn(40)),
		A: 200,
	}

	g.drawFilledRect(img, obs.X, obs.Y, obs.Width, obs.Height, barrierColor)
}

func (g *EnhancedImageGenerator) drawZigzagObstacle(img *image.RGBA, obs ObstacleInfo) {
	zigzagColor := color.RGBA{
		R: uint8(60 + rand.Intn(60)),
		G: uint8(140 + rand.Intn(80)),
		B: uint8(180 + rand.Intn(60)),
		A: 180,
	}

	segments := 4 + rand.Intn(4)
	segmentHeight := obs.Height / segments

	for i := 0; i < segments; i++ {
		offset := 0
		if i%2 == 1 {
			offset = obs.Width / 3
		}
		g.drawFilledRect(img, obs.X+offset, obs.Y+i*segmentHeight, obs.Width/2, segmentHeight, zigzagColor)
	}
}

func (g *EnhancedImageGenerator) drawCurveObstacle(img *image.RGBA, obs ObstacleInfo) {
	curveColor := color.RGBA{
		R: uint8(180 + rand.Intn(75)),
		G: uint8(100 + rand.Intn(60)),
		B: uint8(100 + rand.Intn(80)),
		A: 160,
	}

	for y := 0; y < obs.Height; y++ {
		for x := 0; x < obs.Width; x++ {
			curveX := obs.X + x
			curveY := obs.Y + y

			centerX := obs.X + obs.Width/2
			centerY := obs.Y + obs.Height/2
			
			dx := float64(curveX - centerX)
			dy := float64(curveY - centerY)
			dist := math.Sqrt(dx*dx + dy*dy)
			
			if dist < float64(obs.Width)/2 {
				if curveX >= 0 && curveX < g.width && curveY >= 0 && curveY < g.height {
					img.Set(curveX, curveY, curveColor)
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawBumpObstacle(img *image.RGBA, obs ObstacleInfo) {
	bumpColor := color.RGBA{
		R: uint8(220 + rand.Intn(35)),
		G: uint8(200 + rand.Intn(55)),
		B: uint8(80 + rand.Intn(60)),
		A: 150,
	}

	for y := 0; y < obs.Height; y++ {
		for x := 0; x < obs.Width; x++ {
			nx := float64(x) / float64(obs.Width) * 2 * math.Pi
			ny := math.Sin(nx) * float64(obs.Height/2)

			if math.Abs(float64(y-obs.Height/2)-ny) < 3 {
				bx := obs.X + x
				by := obs.Y + y
				if bx >= 0 && bx < g.width && by >= 0 && by < g.height {
					img.Set(bx, by, bumpColor)
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawHoleObstacle(img *image.RGBA, obs ObstacleInfo) {
	holeColor := color.RGBA{
		R: 30,
		G: 30,
		B: 30,
		A: 255,
	}

	holeRadius := obs.Width / 2

	for y := 0; y < obs.Height; y++ {
		for x := 0; x < obs.Width; x++ {
			dx := float64(x - obs.Width/2)
			dy := float64(y - obs.Height/2)
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist > float64(holeRadius)/2 && dist < float64(holeRadius) {
				hx := obs.X + x
				hy := obs.Y + y
				if hx >= 0 && hx < g.width && hy >= 0 && hy < g.height {
					img.Set(hx, hy, holeColor)
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawTrapObstacle(img *image.RGBA, obs ObstacleInfo) {
	trapColor := color.RGBA{
		R: uint8(180 + rand.Intn(75)),
		G: uint8(60 + rand.Intn(60)),
		B: uint8(60 + rand.Intn(40)),
		A: 170,
	}

	g.drawFilledRect(img, obs.X, obs.Y, obs.Width/3, obs.Height, trapColor)
	g.drawFilledRect(img, obs.X+obs.Width*2/3, obs.Y, obs.Width/3, obs.Height, trapColor)
}

func (g *EnhancedImageGenerator) applyGapWithObstacles(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	darkColor := &image.Uniform{
		C: color.RGBA{
			R: 50,
			G: 50,
			B: 50,
			A: 255,
		},
	}

	draw.Draw(result, gap, darkColor, image.Point{}, draw.Src)

	innerMargin := 3
	innerGap := image.Rect(
		gap.Min.X+innerMargin,
		gap.Min.Y+innerMargin,
		gap.Max.X-innerMargin,
		gap.Max.Y-innerMargin,
	)

	if innerGap.Min.X < innerGap.Max.X && innerGap.Min.Y < innerGap.Max.Y {
		shadowColor := &image.Uniform{
			C: color.RGBA{
				R: 25,
				G: 25,
				B: 25,
				A: 200,
			},
		}
		draw.Draw(result, innerGap, shadowColor, image.Point{}, draw.Src)
	}

	return result
}

func (g *EnhancedImageGenerator) extractEnhancedSlider(background *image.RGBA, gap image.Rectangle, obstacles []ObstacleInfo) *image.RGBA {
	sliderImg := image.NewRGBA(image.Rect(0, 0, g.sliderWidth, g.sliderHeight))

	margin := 4
	extractRect := image.Rect(
		gap.Min.X-margin,
		gap.Min.Y-margin,
		gap.Max.X+margin,
		gap.Max.Y+margin,
	)

	if extractRect.Min.X < 0 {
		extractRect.Min.X = 0
	}
	if extractRect.Min.Y < 0 {
		extractRect.Min.Y = 0
	}
	if extractRect.Max.X > g.width {
		extractRect.Max.X = g.width
	}
	if extractRect.Max.Y > g.height {
		extractRect.Max.Y = g.height
	}

	draw.Draw(sliderImg, sliderImg.Bounds(), &image.Uniform{
		C: color.RGBA{R: 200, G: 200, B: 200, A: 255},
	}, image.Point{}, draw.Src)

	offsetX := -extractRect.Min.X + margin
	offsetY := -extractRect.Min.Y + margin

	for y := 0; y < extractRect.Dy(); y++ {
		for x := 0; x < extractRect.Dx(); x++ {
			srcX := extractRect.Min.X + x
			srcY := extractRect.Min.Y + y

			dstX := x + offsetX
			dstY := y + offsetY

			if dstX >= 0 && dstX < g.sliderWidth && dstY >= 0 && dstY < g.sliderHeight {
				pixel := background.RGBAAt(srcX, srcY)
				sliderImg.SetRGBA(dstX, dstY, pixel)
			}
		}
	}

	g.addSliderBorder(sliderImg)

	sliderImg = g.applyAdvancedSliderProcessing(sliderImg, background, gap)

	return sliderImg
}

func (g *EnhancedImageGenerator) applyAdvancedSliderProcessing(slider *image.RGBA, background *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(slider.Bounds())
	draw.Draw(result, result.Bounds(), slider, slider.Bounds().Min, draw.Src)

	result = g.applyAdaptiveEdgeEnhancement(result)
	result = g.applyIntelligentShadowRecovery(result, background, gap)
	result = g.applyHighQualityAntiAliasing(result)
	result = g.applyColorCorrection(result)

	return result
}

func (g *EnhancedImageGenerator) applyAdaptiveEdgeEnhancement(img *image.RGBA) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	edgeStrength := g.calculateEdgeStrength(img)

	enhancementFactor := 1.0
	if edgeStrength < 15 {
		enhancementFactor = 1.3
	} else if edgeStrength > 40 {
		enhancementFactor = 0.8
	}

	bounds := img.Bounds()
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			edgePixel := g.isEdgePixel(img, x, y)
			if edgePixel {
				p := img.RGBAAt(x, y)

				contrast := 1.0 + (enhancementFactor-1.0)*0.5
				r := g.clampUint8(int(float64(int(p.R)-128)*contrast) + 128)
				gc := g.clampUint8(int(float64(int(p.G)-128)*contrast) + 128)
				b := g.clampUint8(int(float64(int(p.B)-128)*contrast) + 128)

				result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
			}
		}
	}

	return result
}

func (g *EnhancedImageGenerator) calculateEdgeStrength(img *image.RGBA) float64 {
	bounds := img.Bounds()
	totalStrength := 0.0
	count := 0

	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			if g.isEdgePixel(img, x, y) {
				totalStrength += 1.0
			}
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return (totalStrength / float64(count)) * 100
}

func (g *EnhancedImageGenerator) isEdgePixel(img *image.RGBA, x, y int) bool {
	if x < 1 || x >= img.Bounds().Dx()-1 || y < 1 || y >= img.Bounds().Dy()-1 {
		return false
	}

	center := img.RGBAAt(x, y)

	neighbors := [4]struct{ dx, dy int }{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}

	for _, n := range neighbors {
		nx, ny := x+n.dx, y+n.dy
		neighbor := img.RGBAAt(nx, ny)

		diff := float64(abs(int(center.R)-int(neighbor.R))) +
			float64(abs(int(center.G)-int(neighbor.G))) +
			float64(abs(int(center.B)-int(neighbor.B)))

		if diff > 20 {
			return true
		}
	}

	return false
}

func (g *EnhancedImageGenerator) applyIntelligentShadowRecovery(slider *image.RGBA, background *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(slider.Bounds())
	draw.Draw(result, result.Bounds(), slider, slider.Bounds().Min, draw.Src)

	bounds := slider.Bounds()
	sliderMargin := 4

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			isNearEdge := x < sliderMargin || x >= bounds.Dx()-sliderMargin ||
				y < sliderMargin || y >= bounds.Dy()-sliderMargin

			if isNearEdge {
				p := slider.RGBAAt(x, y)
				brightness := float64(p.R)*0.299 + float64(p.G)*0.587 + float64(p.B)*0.114

				if brightness < 80 {
					bgX := gap.Min.X + x - sliderMargin
					bgY := gap.Min.Y + y - sliderMargin

					if bgX >= 0 && bgX < background.Bounds().Dx() &&
						bgY >= 0 && bgY < background.Bounds().Dy() {
						bgPixel := background.RGBAAt(bgX, bgY)
						bgBrightness := float64(bgPixel.R)*0.299 + float64(bgPixel.G)*0.587 + float64(bgPixel.B)*0.114

						if bgBrightness > brightness+20 {
							blendFactor := 0.3
							r := g.clampUint8(int(float64(p.R)*(1-blendFactor) + float64(bgPixel.R)*blendFactor))
							gc := g.clampUint8(int(float64(p.G)*(1-blendFactor) + float64(bgPixel.G)*blendFactor))
							b := g.clampUint8(int(float64(p.B)*(1-blendFactor) + float64(bgPixel.B)*blendFactor))

							result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
						}
					}
				}
			}
		}
	}

	return result
}

func (g *EnhancedImageGenerator) applyHighQualityAntiAliasing(img *image.RGBA) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	bounds := img.Bounds()
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			edgeCount := 0
			neighbors := [8]struct{ dx, dy int }{
				{-1, -1}, {0, -1}, {1, -1},
				{-1, 0}, {1, 0},
				{-1, 1}, {0, 1}, {1, 1},
			}

			for _, n := range neighbors {
				if g.isEdgePixel(img, x+n.dx, y+n.dy) {
					edgeCount++
				}
			}

			if edgeCount >= 3 && edgeCount <= 5 {
				p := img.RGBAAt(x, y)

				avgR, avgG, avgB := 0, 0, 0
				validNeighbors := 0

				for _, n := range neighbors {
					nx, ny := x+n.dx, y+n.dy
					if nx >= 0 && nx < bounds.Dx() && ny >= 0 && ny < bounds.Dy() {
						np := img.RGBAAt(nx, ny)
						avgR += int(np.R)
						avgG += int(np.G)
						avgB += int(np.B)
						validNeighbors++
					}
				}

				if validNeighbors > 0 {
					blendFactor := 0.3
					r := g.clampUint8(int(float64(p.R)*(1-blendFactor) + float64(avgR/validNeighbors)*blendFactor))
					gc := g.clampUint8(int(float64(p.G)*(1-blendFactor) + float64(avgG/validNeighbors)*blendFactor))
					b := g.clampUint8(int(float64(p.B)*(1-blendFactor) + float64(avgB/validNeighbors)*blendFactor))

					result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
				}
			}
		}
	}

	return result
}

func (g *EnhancedImageGenerator) applyColorCorrection(img *image.RGBA) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	bounds := img.Bounds()

	var totalR, totalG, totalB float64
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p := img.RGBAAt(x, y)
			totalR += float64(p.R)
			totalG += float64(p.G)
			totalB += float64(p.B)
			count++
		}
	}

	if count == 0 {
		return result
	}

	avgR := totalR / float64(count)
	avgG := totalG / float64(count)
	avgB := totalB / float64(count)

	avgBrightness := (avgR + avgG + avgB) / 3.0
	targetBrightness := 140.0

	brightnessRatio := targetBrightness / avgBrightness
	if brightnessRatio > 1.3 {
		brightnessRatio = 1.3
	}
	if brightnessRatio < 0.7 {
		brightnessRatio = 0.7
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p := img.RGBAAt(x, y)

			r := g.clampUint8(int(float64(p.R) * brightnessRatio))
			gc := g.clampUint8(int(float64(p.G) * brightnessRatio))
			b := g.clampUint8(int(float64(p.B) * brightnessRatio))

			result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	return result
}

func (g *EnhancedImageGenerator) addSliderBorder(slider *image.RGBA) {
	bounds := slider.Bounds()

	for x := 0; x < bounds.Dx(); x++ {
		for offset := 0; offset < 2; offset++ {
			if bounds.Min.Y+offset < bounds.Max.Y {
				p := slider.RGBAAt(x, bounds.Min.Y+offset)
				slider.Set(x, bounds.Min.Y+offset, color.RGBA{
					R: g.clampUint8(int(p.R) * 70 / 100),
					G: g.clampUint8(int(p.G) * 70 / 100),
					B: g.clampUint8(int(p.B) * 70 / 100),
					A: 255,
				})
			}

			if bounds.Max.Y-1-offset >= bounds.Min.Y {
				p := slider.RGBAAt(x, bounds.Max.Y-1-offset)
				slider.Set(x, bounds.Max.Y-1-offset, color.RGBA{
					R: g.clampUint8(int(p.R) * 70 / 100),
					G: g.clampUint8(int(p.G) * 70 / 100),
					B: g.clampUint8(int(p.B) * 70 / 100),
					A: 255,
				})
			}
		}
	}

	for y := 0; y < bounds.Dy(); y++ {
		for offset := 0; offset < 2; offset++ {
			if bounds.Min.X+offset < bounds.Max.X {
				p := slider.RGBAAt(bounds.Min.X+offset, y)
				slider.Set(bounds.Min.X+offset, y, color.RGBA{
					R: g.clampUint8(int(p.R) * 70 / 100),
					G: g.clampUint8(int(p.G) * 70 / 100),
					B: g.clampUint8(int(p.B) * 70 / 100),
					A: 255,
				})
			}

			if bounds.Max.X-1-offset >= bounds.Min.X {
				p := slider.RGBAAt(bounds.Max.X-1-offset, y)
				slider.Set(bounds.Max.X-1-offset, y, color.RGBA{
					R: g.clampUint8(int(p.R) * 70 / 100),
					G: g.clampUint8(int(p.G) * 70 / 100),
					B: g.clampUint8(int(p.B) * 70 / 100),
					A: 255,
				})
			}
		}
	}
}

func (g *EnhancedImageGenerator) applyAdvancedEdgeDetection(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	edgeWidth := 2

	for y := gap.Min.Y; y < gap.Max.Y; y++ {
		for offset := 0; offset < edgeWidth; offset++ {
			if gap.Min.X-offset >= 0 {
				g.applyEdgePixel(result, gap.Min.X-offset, y, gap, -1, 0)
			}
			if gap.Max.X+offset < g.width {
				g.applyEdgePixel(result, gap.Max.X+offset, y, gap, 1, 0)
			}
		}
	}

	for x := gap.Min.X; x < gap.Max.X; x++ {
		for offset := 0; offset < edgeWidth; offset++ {
			if gap.Min.Y-offset >= 0 {
				g.applyEdgePixel(result, x, gap.Min.Y-offset, gap, 0, -1)
			}
			if gap.Max.Y+offset < g.height {
				g.applyEdgePixel(result, x, gap.Max.Y+offset, gap, 0, 1)
			}
		}
	}

	return result
}

func (g *EnhancedImageGenerator) applyEdgePixel(img *image.RGBA, x, y int, gap image.Rectangle, dirX, dirY int) {
	if x < 0 || x >= g.width || y < 0 || y >= g.height {
		return
	}

	centerX := (gap.Min.X + gap.Max.X) / 2
	centerY := (gap.Min.Y + gap.Max.Y) / 2
	dx := x - centerX
	dy := y - centerY

	var gradientMagnitude float64
	for ny := -1; ny <= 1; ny++ {
		for nx := -1; nx <= 1; nx++ {
			px, py := x+nx, y+ny
			if px < 0 || px >= g.width || py < 0 || py >= g.height {
				continue
			}

			p1 := img.RGBAAt(x, y)
			p2 := img.RGBAAt(px, py)

			edgeDiff := float64(abs(int(p1.R)-int(p2.R))) +
				float64(abs(int(p1.G)-int(p2.G))) +
				float64(abs(int(p1.B)-int(p2.B)))

			gradientMagnitude = math.Max(gradientMagnitude, edgeDiff)
		}
	}

	if gradientMagnitude < 20 {
		highlight := 0.15
		p := img.RGBAAt(x, y)
		r := g.clampUint8(int(float64(p.R)*(1+highlight)) - 10)
		gc := g.clampUint8(int(float64(p.G)*(1+highlight)) - 10)
		b := g.clampUint8(int(float64(p.B)*(1+highlight)) - 10)

		img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
	}

	lightDir := 0.6
	effectiveLight := lightDir - float64(dy)/float64(g.height)*0.3 + float64(dx)/float64(g.width)*0.1
	if effectiveLight > 1 {
		effectiveLight = 1
	}
	if effectiveLight < 0.3 {
		effectiveLight = 0.3
	}

	lightAdjustment := (effectiveLight - 0.5) * 20

	if (dirX > 0 || dirY > 0) && lightAdjustment > 0 {
		p := img.RGBAAt(x, y)
		r := g.clampUint8(int(p.R) + int(lightAdjustment))
		gc := g.clampUint8(int(p.G) + int(lightAdjustment))
		b := g.clampUint8(int(p.B) + int(lightAdjustment))
		img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
	} else if (dirX < 0 || dirY < 0) && lightAdjustment < 0 {
		p := img.RGBAAt(x, y)
		r := g.clampUint8(int(p.R) + int(lightAdjustment))
		gc := g.clampUint8(int(p.G) + int(lightAdjustment))
		b := g.clampUint8(int(p.B) + int(lightAdjustment))
		img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
	}
}

func (g *EnhancedImageGenerator) applyEnhancedShadowDetection(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	shadowBlur := 6
	shadowOffsetX := 3
	shadowOffsetY := 3

	for y := gap.Min.Y - shadowBlur; y <= gap.Max.Y+shadowBlur; y++ {
		for x := gap.Min.X - shadowBlur; x <= gap.Max.X+shadowBlur; x++ {
			if x < 0 || x >= g.width || y < 0 || y >= g.height {
				continue
			}

			isInsideGap := x >= gap.Min.X && x < gap.Max.X && y >= gap.Min.Y && y < gap.Max.Y
			if isInsideGap {
				continue
			}

			distToGap := g.getMinDistanceToRect(x, y, gap)
			if distToGap <= float64(shadowBlur) {
				blurFactor := 1.0 - distToGap/float64(shadowBlur)

				shadowX := float64(x) + float64(shadowOffsetX)
				shadowY := float64(y) + float64(shadowOffsetY)

				if shadowX >= float64(gap.Min.X) && shadowX < float64(gap.Max.X) &&
					shadowY >= float64(gap.Min.Y) && shadowY < float64(gap.Max.Y) {

					p := result.RGBAAt(x, y)
					darkness := blurFactor * 0.4

					r := g.clampUint8(int(float64(p.R) * (1 - darkness)))
					gc := g.clampUint8(int(float64(p.G) * (1 - darkness)))
					b := g.clampUint8(int(float64(p.B) * (1 - darkness)))

					result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
				}
			}
		}
	}

	return result
}

func (g *EnhancedImageGenerator) getMinDistanceToRect(x, y int, rect image.Rectangle) float64 {
	dx := 0
	dy := 0

	if x < rect.Min.X {
		dx = rect.Min.X - x
	} else if x >= rect.Max.X {
		dx = x - rect.Max.X + 1
	}

	if y < rect.Min.Y {
		dy = rect.Min.Y - y
	} else if y >= rect.Max.Y {
		dy = y - rect.Max.Y + 1
	}

	if dx == 0 && dy == 0 {
		insideX := x >= rect.Min.X && x < rect.Max.X
		insideY := y >= rect.Min.Y && y < rect.Max.Y

		if insideX {
			return float64(min(y-rect.Min.Y, rect.Max.Y-1-y))
		}
		if insideY {
			return float64(min(x-rect.Min.X, rect.Max.X-1-x))
		}
		return 0
	}

	return math.Sqrt(float64(dx*dx + dy*dy))
}

func (g *EnhancedImageGenerator) addEnhancedInterference(img *image.RGBA, difficulty int) *image.RGBA {
	noiseCount := 200 + difficulty*100
	g.addNoiseDots(img, noiseCount)

	g.addCracks(img, 2+difficulty)

	g.addBrightnessVariation(img)

	g.addSmallCircles(img, 10+difficulty*5)

	g.addLineInterference(img, difficulty)

	return img
}

func (g *EnhancedImageGenerator) addLineInterference(img *image.RGBA, difficulty int) {
	lineCount := difficulty + rand.Intn(3)

	for i := 0; i < lineCount; i++ {
		startX := rand.Intn(g.width)
		startY := rand.Intn(g.height)
		length := 10 + rand.Intn(30)
		angle := float64(rand.Intn(360)) * math.Pi / 180

		lineColor := color.RGBA{
			R: uint8(rand.Intn(100)),
			G: uint8(rand.Intn(100)),
			B: uint8(rand.Intn(100)),
			A: uint8(30 + rand.Intn(30)),
		}

		endX := startX + int(math.Cos(angle)*float64(length))
		endY := startY + int(math.Sin(angle)*float64(length))

		g.drawDiagonalLine(img, startX, startY, endX, endY, 1, lineColor)
	}
}

func (g *EnhancedImageGenerator) applyIntelligentAntiDetection(img *image.RGBA, gap image.Rectangle, obstacles []ObstacleInfo) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	g.applyHistogramEqualization(result)

	g.applyLocalContrastEnhancement(result, gap)

	g.applyEdgePreservingSmoothing(result)

	return result
}

func (g *EnhancedImageGenerator) applyHistogramEqualization(img *image.RGBA) {
	histogram := make([]int, 256)
	pixelCount := g.width * g.height

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			p := img.RGBAAt(x, y)
			brightness := int(0.299*float64(p.R) + 0.587*float64(p.G) + 0.114*float64(p.B))
			histogram[brightness]++
		}
	}

	cdf := make([]int, 256)
	cdf[0] = histogram[0]
	for i := 1; i < 256; i++ {
		cdf[i] = cdf[i-1] + histogram[i]
	}

	cdfMin := 0
	for i := 0; i < 256; i++ {
		if cdf[i] > 0 {
			cdfMin = cdf[i]
			break
		}
	}

	lookup := make([]uint8, 256)
	for i := 0; i < 256; i++ {
		if pixelCount > cdfMin {
			lookup[i] = uint8(float64(cdf[i]-cdfMin) / float64(pixelCount-cdfMin) * 255)
		} else {
			lookup[i] = uint8(i)
		}
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			p := img.RGBAAt(x, y)
			img.Set(x, y, color.RGBA{
				R: lookup[p.R],
				G: lookup[p.G],
				B: lookup[p.B],
				A: 255,
			})
		}
	}
}

func (g *EnhancedImageGenerator) applyLocalContrastEnhancement(img *image.RGBA, gap image.Rectangle) {
	kernelSize := 5
	halfKernel := kernelSize / 2

	for y := halfKernel; y < g.height-halfKernel; y++ {
		for x := halfKernel; x < g.width-halfKernel; x++ {
			if x >= gap.Min.X-halfKernel && x < gap.Max.X+halfKernel &&
				y >= gap.Min.Y-halfKernel && y < gap.Max.Y+halfKernel {
				continue
			}

			var sumR, sumG, sumB float64
			count := 0

			for ky := -halfKernel; ky <= halfKernel; ky++ {
				for kx := -halfKernel; kx <= halfKernel; kx++ {
					p := img.RGBAAt(x+kx, y+ky)
					sumR += float64(p.R)
					sumG += float64(p.G)
					sumB += float64(p.B)
					count++
				}
			}

			meanR := sumR / float64(count)
			meanG := sumG / float64(count)
			meanB := sumB / float64(count)

			p := img.RGBAAt(x, y)
			
			var localVarR, localVarG, localVarB float64
			for ky := -halfKernel; ky <= halfKernel; ky++ {
				for kx := -halfKernel; kx <= halfKernel; kx++ {
					np := img.RGBAAt(x+kx, y+ky)
					localVarR += (float64(np.R) - meanR) * (float64(np.R) - meanR)
					localVarG += (float64(np.G) - meanG) * (float64(np.G) - meanG)
					localVarB += (float64(np.B) - meanB) * (float64(np.B) - meanB)
				}
			}

			localVarR /= float64(count)
			localVarG /= float64(count)
			localVarB /= float64(count)

			globalVar := (localVarR + localVarG + localVarB) / 3.0
			
			enhancementFactor := 1.0
			if globalVar < 500 {
				enhancementFactor = 1.2
			}

			newR := meanR + (float64(p.R)-meanR)*enhancementFactor
			newG := meanG + (float64(p.G)-meanG)*enhancementFactor
			newB := meanB + (float64(p.B)-meanB)*enhancementFactor

			img.Set(x, y, color.RGBA{
				R: g.clampUint8(int(newR)),
				G: g.clampUint8(int(newG)),
				B: g.clampUint8(int(newB)),
				A: 255,
			})
		}
	}
}

func (g *EnhancedImageGenerator) applyEdgePreservingSmoothing(img *image.RGBA) {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	kernelSize := 3
	halfKernel := kernelSize / 2

	for y := halfKernel; y < g.height-halfKernel; y++ {
		for x := halfKernel; x < g.width-halfKernel; x++ {
			centerPixel := img.RGBAAt(x, y)
			
			var weightedR, weightedG, weightedB float64
			totalWeight := 0.0

			for ky := -halfKernel; ky <= halfKernel; ky++ {
				for kx := -halfKernel; kx <= halfKernel; kx++ {
					neighbor := img.RGBAAt(x+kx, y+ky)
					
					colorDiff := math.Abs(float64(int(centerPixel.R)-int(neighbor.R))) +
						math.Abs(float64(int(centerPixel.G)-int(neighbor.G))) +
						math.Abs(float64(int(centerPixel.B)-int(neighbor.B)))

					spatialDist := math.Sqrt(float64(kx*kx + ky*ky))
					
					sigmaColor := 30.0
					sigmaSpace := 5.0
					
					colorWeight := math.Exp(-colorDiff / (sigmaColor * sigmaColor))
					spatialWeight := math.Exp(-spatialDist / (sigmaSpace * sigmaSpace))
					
					weight := colorWeight * spatialWeight
					
					weightedR += float64(neighbor.R) * weight
					weightedG += float64(neighbor.G) * weight
					weightedB += float64(neighbor.B) * weight
					totalWeight += weight
				}
			}

			if totalWeight > 0 {
				result.Set(x, y, color.RGBA{
					R: g.clampUint8(int(weightedR / totalWeight)),
					G: g.clampUint8(int(weightedG / totalWeight)),
					B: g.clampUint8(int(weightedB / totalWeight)),
					A: 255,
				})
			}
		}
	}

	draw.Draw(img, img.Bounds(), result, result.Bounds().Min, draw.Src)
}

func (g *EnhancedImageGenerator) addNoiseDots(img *image.RGBA, count int) {
	for i := 0; i < count; i++ {
		x := rand.Intn(g.width)
		y := rand.Intn(g.height)
		radius := rand.Intn(2) + 1

		c := color.RGBA{
			R: uint8(rand.Intn(256)),
			G: uint8(rand.Intn(256)),
			B: uint8(rand.Intn(256)),
			A: uint8(20 + rand.Intn(40)),
		}

		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx*dx+dy*dy <= radius*radius {
					px, py := x+dx, y+dy
					if px >= 0 && px < g.width && py >= 0 && py < g.height {
						img.Set(px, py, c)
					}
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) addCracks(img *image.RGBA, count int) {
	for i := 0; i < count; i++ {
		startX := rand.Intn(g.width)
		startY := rand.Intn(g.height)

		steps := 5 + rand.Intn(10)
		prevX, prevY := startX, startY

		crackColor := color.RGBA{
			R: uint8(rand.Intn(100)),
			G: uint8(rand.Intn(100)),
			B: uint8(rand.Intn(100)),
			A: uint8(40 + rand.Intn(40)),
		}

		for j := 0; j < steps; j++ {
			dx := rand.Intn(20) - 10
			dy := rand.Intn(20) - 10

			newX := prevX + dx
			newY := prevY + dy

			if newX < 0 {
				newX = 0
			}
			if newX >= g.width {
				newX = g.width - 1
			}
			if newY < 0 {
				newY = 0
			}
			if newY >= g.height {
				newY = g.height - 1
			}

			g.drawLine(img, prevX, prevY, newX, newY, crackColor)

			prevX, prevY = newX, newY
		}
	}
}

func (g *EnhancedImageGenerator) addBrightnessVariation(img *image.RGBA) {
	brightCount := 3 + rand.Intn(5)

	for i := 0; i < brightCount; i++ {
		centerX := rand.Intn(g.width)
		centerY := rand.Intn(g.height)
		radius := 10 + rand.Intn(20)

		variation := int16(-30 + rand.Intn(60))

		for y := centerY - radius; y <= centerY+radius; y++ {
			for x := centerX - radius; x <= centerX+radius; x++ {
				if x < 0 || x >= g.width || y < 0 || y >= g.height {
					continue
				}

				dx := x - centerX
				dy := y - centerY
				dist := math.Sqrt(float64(dx*dx + dy*dy))

				if dist <= float64(radius) {
					factor := 1.0 - (dist / float64(radius))
					adjustment := int16(float64(variation) * factor)

					p := img.RGBAAt(x, y)
					img.Set(x, y, color.RGBA{
						R: g.clampUint8(int(p.R) + int(adjustment)),
						G: g.clampUint8(int(p.G) + int(adjustment)),
						B: g.clampUint8(int(p.B) + int(adjustment)),
						A: 255,
					})
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) addSmallCircles(img *image.RGBA, count int) {
	for i := 0; i < count; i++ {
		x := rand.Intn(g.width)
		y := rand.Intn(g.height)
		radius := 3 + rand.Intn(5)

		c := color.RGBA{
			R: uint8(rand.Intn(256)),
			G: uint8(rand.Intn(256)),
			B: uint8(rand.Intn(256)),
			A: uint8(60 + rand.Intn(40)),
		}

		g.drawFilledCircle(img, x, y, radius, c)
	}
}

func (g *EnhancedImageGenerator) drawFilledCircle(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < g.width && y >= 0 && y < g.height {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawFilledRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < g.width && py >= 0 && py < g.height {
				img.Set(px, py, c)
			}
		}
	}
}

func (g *EnhancedImageGenerator) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
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

		if x >= 0 && x < g.width && y >= 0 && y < g.height {
			img.Set(x, y, c)
		}
	}
}

func (g *EnhancedImageGenerator) drawDiagonalLine(img *image.RGBA, x1, y1, x2, y2, width int, c color.RGBA) {
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

		for w := -width / 2; w <= width/2; w++ {
			px, py := x+w, y
			if px >= 0 && px < g.width && py >= 0 && py < g.height {
				img.Set(px, py, c)
			}
			px, py = x, y+w
			if px >= 0 && px < g.width && py >= 0 && py < g.height {
				img.Set(px, py, c)
			}
		}
	}
}

func (g *EnhancedImageGenerator) clampUint8(val int) uint8 {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}

func (g *EnhancedImageGenerator) encodePNG(img *image.RGBA) []byte {
	buf := g.imagePool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		g.imagePool.Put(buf)
	}()

	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(buf, img); err != nil {
		return nil
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result
}

type ObstacleGenerator struct{}

func NewObstacleGenerator() *ObstacleGenerator {
	return &ObstacleGenerator{}
}

func (g *ObstacleGenerator) GenerateObstacles(count int, width, height int) []ObstacleInfo {
	var obstacles []ObstacleInfo

	for i := 0; i < count; i++ {
		obstacle := ObstacleInfo{
			Type:     g.getRandomType(),
			X:        rand.Intn(width - 60),
			Y:        rand.Intn(height - 60),
			Width:    30 + rand.Intn(40),
			Height:   30 + rand.Intn(40),
			Rotation: rand.Intn(4) * 90,
		}
		obstacles = append(obstacles, obstacle)
	}

	return obstacles
}

func (g *ObstacleGenerator) getRandomType() string {
	types := []string{"barrier", "zigzag", "curve", "bump", "hole", "trap"}
	return types[rand.Intn(len(types))]
}

type TrajectoryGenerator struct{}

func NewTrajectoryGenerator() *TrajectoryGenerator {
	return &TrajectoryGenerator{}
}

func (g *TrajectoryGenerator) GenerateHint(targetX, targetY int, difficulty int) TrajectoryHint {
	hints := make([]Point, 3+rand.Intn(3))
	
	for i := range hints {
		hints[i] = Point{
			X: float64(targetX) * (float64(i+1) / float64(len(hints)+1)),
			Y: float64(targetY) + float64(rand.Intn(20)-10),
		}
	}

	sort.Slice(hints, func(i, j int) bool {
		return hints[i].X < hints[j].X
	})

	return TrajectoryHint{
		SuggestedSpeed: 0.5 + float64(difficulty)*0.2 + rand.Float64()*0.3,
		PathComplexity: difficulty,
		Hints:          hints,
	}
}

func (g *TrajectoryGenerator) GenerateRandomTrajectory(startX, startY, endX, endY, difficulty int) []Point {
	trajectory := make([]Point, 0)
	
	points := 10 + difficulty*5 + rand.Intn(10)
	
	controlPoints := g.generateControlPoints(startX, startY, endX, endY, difficulty)
	
	for i := 0; i <= points; i++ {
		t := float64(i) / float64(points)
		x, y := g.bezierCurvePoint(controlPoints, t)
		
		jitter := 2 + rand.Intn(5)
		jitterX := float64(rand.Intn(jitter*2) - jitter)
		jitterY := float64(rand.Intn(jitter*2) - jitter)
		
		trajectory = append(trajectory, Point{
			X: x + jitterX,
			Y: y + jitterY,
		})
	}

	return trajectory
}

func (g *TrajectoryGenerator) generateControlPoints(startX, startY, endX, endY, difficulty int) []Point {
	numPoints := 2 + difficulty
	points := make([]Point, numPoints+2)
	
	points[0] = Point{float64(startX), float64(startY)}
	points[numPoints+1] = Point{float64(endX), float64(endY)}
	
	midX := float64(startX+endX) / 2
	midY := float64(startY+endY) / 2
	
	for i := 1; i <= numPoints; i++ {
		t := float64(i) / float64(numPoints+1)
		offsetX := float64(rand.Intn(40)-20) + (float64(endX)-float64(startX))*0.2
		offsetY := float64(rand.Intn(40)-20) + (float64(endY)-float64(startY))*0.2
		
		points[i] = Point{
			X: midX + offsetX + (float64(endX)-midX)*t,
			Y: midY + offsetY + (float64(endY)-midY)*t,
		}
	}
	
	return points
}

func (g *TrajectoryGenerator) bezierCurvePoint(points []Point, t float64) (float64, float64) {
	n := len(points) - 1
	
	var x, y float64
	for i := 0; i <= n; i++ {
		binomial := g.binomialCoefficient(n, i)
		power := math.Pow(1-t, float64(n-i))
		powerT := math.Pow(t, float64(i))
		
		x += binomial * power * powerT * points[i].X
		y += binomial * power * powerT * points[i].Y
	}
	
	return x, y
}

func (g *TrajectoryGenerator) binomialCoefficient(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	
	result := 1.0
	for i := 0; i < k; i++ {
		result *= float64(n-i) / float64(i+1)
	}
	
	return result
}

type SmartGapDetector struct{}

func NewSmartGapDetector() *SmartGapDetector {
	return &SmartGapDetector{}
}

func (d *SmartGapDetector) DetectOptimalGap(img *image.RGBA, difficulty int) (int, int) {
	edges := d.detectEdges(img)
	
	verticalProjection := d.verticalProjection(edges)
	horizontalProjection := d.horizontalProjection(edges)
	
	gapX := d.findOptimalPosition(verticalProjection, difficulty)
	gapY := d.findOptimalPosition(horizontalProjection, difficulty)
	
	return gapX, gapY
}

func (d *SmartGapDetector) detectEdges(img *image.RGBA) [][]float64 {
	edges := make([][]float64, img.Bounds().Dy())
	for i := range edges {
		edges[i] = make([]float64, img.Bounds().Dx())
	}

	sobelX := [][]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY := [][]int{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	for y := 1; y < img.Bounds().Dy()-1; y++ {
		for x := 1; x < img.Bounds().Dx()-1; x++ {
			gx := 0.0
			gy := 0.0

			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					p := img.RGBAAt(x+kx, y+ky)
					brightness := float64(p.R)*0.299 + float64(p.G)*0.587 + float64(p.B)*0.114

					gx += brightness * float64(sobelX[ky+1][kx+1])
					gy += brightness * float64(sobelY[ky+1][kx+1])
				}
			}

			edges[y][x] = math.Sqrt(gx*gx + gy*gy)
		}
	}

	return edges
}

func (d *SmartGapDetector) verticalProjection(edges [][]float64) []float64 {
	height := len(edges)
	if height == 0 {
		return nil
	}
	width := len(edges[0])

	projection := make([]float64, width)
	for x := 0; x < width; x++ {
		sum := 0.0
		for y := 0; y < height; y++ {
			sum += edges[y][x]
		}
		projection[x] = sum / float64(height)
	}

	return projection
}

func (d *SmartGapDetector) horizontalProjection(edges [][]float64) []float64 {
	height := len(edges)
	if height == 0 {
		return nil
	}
	width := len(edges[0])

	projection := make([]float64, height)
	for y := 0; y < height; y++ {
		sum := 0.0
		for x := 0; x < width; x++ {
			sum += edges[y][x]
		}
		projection[y] = sum / float64(width)
	}

	return projection
}

func (d *SmartGapDetector) findOptimalPosition(projection []float64, difficulty int) int {
	if len(projection) == 0 {
		return 50
	}

	smoothed := d.smoothProjection(projection, difficulty)

	minIdx := 0
	minVal := smoothed[0]

	for i := 1; i < len(smoothed); i++ {
		margin := 30 + difficulty*10
		if i < margin || i >= len(smoothed)-margin {
			continue
		}
		
		if smoothed[i] < minVal {
			minVal = smoothed[i]
			minIdx = i
		}
	}

	return minIdx
}

func (d *SmartGapDetector) smoothProjection(projection []float64, windowSize int) []float64 {
	if len(projection) == 0 {
		return nil
	}

	result := make([]float64, len(projection))
	halfWindow := windowSize / 2

	for i := 0; i < len(projection); i++ {
		sum := 0.0
		count := 0

		for j := i - halfWindow; j <= i+halfWindow; j++ {
			if j >= 0 && j < len(projection) {
				sum += projection[j]
				count++
			}
		}

		result[i] = sum / float64(count)
	}

	return result
}

type AdaptiveResistanceSystem struct{}

func NewAdaptiveResistanceSystem() *AdaptiveResistanceSystem {
	return &AdaptiveResistanceSystem{}
}

func (s *AdaptiveResistanceSystem) CalculateResistanceLevel(fingerprint string, difficulty int) int {
	baseResistance := 1 + difficulty

	if len(fingerprint) > 0 {
		fingerprintHash := 0
		for _, c := range fingerprint {
			fingerprintHash += int(c)
		}
		
		if fingerprintHash%3 == 0 {
			baseResistance++
		}
	}

	if baseResistance > 5 {
		baseResistance = 5
	}

	return baseResistance
}

func (s *AdaptiveResistanceSystem) GetResistanceCurve(level int) []float64 {
	curve := make([]float64, 100)
	
	for i := 0; i < 100; i++ {
		t := float64(i) / 100.0
		
		switch level {
		case 1:
			curve[i] = 0.1 + t*0.1
		case 2:
			curve[i] = 0.2 + t*0.15 + math.Sin(t*math.Pi)*0.1
		case 3:
			curve[i] = 0.3 + t*0.2 + math.Sin(t*math.Pi)*0.15
		case 4:
			curve[i] = 0.4 + t*0.25 + math.Sin(t*math.Pi*1.5)*0.15
		case 5:
			curve[i] = 0.5 + t*0.3 + math.Sin(t*math.Pi*2)*0.2
		default:
			curve[i] = 0.2 + t*0.2
		}
	}
	
	return curve
}

func (s *AdaptiveResistanceSystem) CalculateDragResistance(position float64, level int) float64 {
	curve := s.GetResistanceCurve(level)
	
	idx := int(position * 100)
	if idx >= 100 {
		idx = 99
	}
	if idx < 0 {
		idx = 0
	}
	
	return curve[idx]
}

func clampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
