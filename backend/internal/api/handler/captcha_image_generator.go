package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	ImageWidth   = 360
	ImageHeight  = 220
	SliderWidth  = 50
	SliderHeight = 50
)

type CaptchaImageGenerator struct {
	background  *image.RGBA
	sliderImage *image.RGBA
	puzzleMask  *image.RGBA
	targetX     int
	targetY     int
	seed        int64
}

type CaptchaCache struct {
	mu      sync.RWMutex
	items   map[string]*CachedCaptcha
	maxAge  time.Duration
	maxSize int
}

type CachedCaptcha struct {
	Generator   *CaptchaImageGenerator
	CreatedAt   time.Time
	AccessCount int
}

var (
	captchaCache     *CaptchaCache
	captchaCacheOnce sync.Once
)

func GetCaptchaCache() *CaptchaCache {
	captchaCacheOnce.Do(func() {
		captchaCache = &CaptchaCache{
			items:   make(map[string]*CachedCaptcha),
			maxAge:  5 * time.Minute,
			maxSize: 100,
		}
		go captchaCache.cleanup()
	})
	return captchaCache
}

func (c *CaptchaCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.Sub(item.CreatedAt) > c.maxAge {
				delete(c.items, key)
			}
		}
		if len(c.items) > c.maxSize {
			for key := range c.items {
				delete(c.items, key)
				if len(c.items) <= c.maxSize/2 {
					break
				}
			}
		}
		c.mu.Unlock()
	}
}

func NewCaptchaImageGenerator() *CaptchaImageGenerator {
	gen := &CaptchaImageGenerator{
		background:  image.NewRGBA(image.Rect(0, 0, ImageWidth, ImageHeight)),
		sliderImage: image.NewRGBA(image.Rect(0, 0, SliderWidth, SliderHeight)),
		puzzleMask:  image.NewRGBA(image.Rect(0, 0, SliderWidth, SliderHeight)),
		seed:       time.Now().UnixNano(),
	}

	rand.Seed(gen.seed)

	gen.targetX = 100 + rand.Intn(180)
	gen.targetY = 30 + rand.Intn(130)

	gen.generateBackground()
	gen.generatePuzzleMask()
	gen.generateSliderImage()
	gen.applyBackgroundEffects()

	return gen
}

func (g *CaptchaImageGenerator) GetTargetX() int {
	return g.targetX
}

func (g *CaptchaImageGenerator) GetTargetY() int {
	return g.targetY
}

func (g *CaptchaImageGenerator) generateBackground() {
	gradientColors := g.generateGradientColors()

	startColor := gradientColors[0]
	endColor := gradientColors[1]

	for y := 0; y < ImageHeight; y++ {
		ratio := float64(y) / float64(ImageHeight)
		r := uint8(float64(startColor.R) + ratio*float64(endColor.R-startColor.R))
		gr := uint8(float64(startColor.G) + ratio*float64(endColor.G-startColor.G))
		b := uint8(float64(startColor.B) + ratio*float64(endColor.B-startColor.B))

		for x := 0; x < ImageWidth; x++ {
			g.background.Set(x, y, color.RGBA{R: r, G: gr, B: b, A: 255})
		}
	}
}

func (g *CaptchaImageGenerator) generateGradientColors() []color.RGBA {
	palettes := [][]color.RGBA{
		{{R: 102, G: 126, B: 234, A: 255}, {R: 118, G: 75, B: 162, A: 255}},
		{{R: 240, G: 147, B: 251, A: 255}, {R: 245, G: 87, B: 108, A: 255}},
		{{R: 44, G: 162, B: 95, A: 255}, {R: 32, G: 201, B: 151, A: 255}},
		{{R: 255, G: 107, B: 107, A: 255}, {R: 255, G: 159, B: 67, A: 255}},
		{{R: 52, G: 211, B: 153, A: 255}, {R: 44, G: 162, B: 95, A: 255}},
		{{R: 72, G: 99, B: 228, A: 255}, {R: 91, G: 33, B: 182, A: 255}},
		{{R: 251, G: 146, B: 60, A: 255}, {R: 255, G: 196, B: 14, A: 255}},
		{{R: 96, G: 205, B: 255, A: 255}, {R: 192, G: 132, B: 248, A: 255}},
	}

	return palettes[rand.Intn(len(palettes))]
}

func (g *CaptchaImageGenerator) generatePuzzleMask() {
	g.puzzleMask = image.NewRGBA(image.Rect(0, 0, SliderWidth, SliderHeight))

	centerX := SliderWidth / 2
	centerY := SliderHeight / 2
	edgeSize := 10

	for y := 0; y < SliderHeight; y++ {
		for x := 0; x < SliderWidth; x++ {
			dx := float64(x - centerX)
			dy := float64(y - centerY)
			distance := math.Sqrt(dx*dx + dy*dy)

			maxRadius := float64(SliderWidth) / 2

			if distance > maxRadius {
				continue
			}

			alpha := uint8(255)

			edgeDistance := maxRadius - distance
			if edgeDistance < float64(edgeSize) {
				featherRatio := edgeDistance / float64(edgeSize)
				alpha = uint8(255 * featherRatio)
			}

			side := (x / (SliderWidth / 3)) % 3

			if y >= centerY-12 && y <= centerY+12 {
				if side == 0 && x >= SliderWidth-8 && x <= SliderWidth-2 {
					alpha = 0
				}
				if side == 1 && x >= SliderWidth/2-6 && x <= SliderWidth/2 {
					alpha = 0
				}
			}

			g.puzzleMask.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: alpha})
		}
	}
}

func (g *CaptchaImageGenerator) generateSliderImage() {
	for y := 0; y < SliderHeight; y++ {
		for x := 0; x < SliderWidth; x++ {
			_, _, _, maskAlpha := g.puzzleMask.At(x, y).RGBA()

			if maskAlpha == 0 {
				g.sliderImage.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
				continue
			}

			srcX := g.targetX + x
			srcY := g.targetY + y

			if srcX >= 0 && srcX < ImageWidth && srcY >= 0 && srcY < ImageHeight {
				srcColor := g.background.At(srcX, srcY)
				r, gr, b, _ := srcColor.RGBA()

				overlayColor := color.RGBA{
					R: uint8(r >> 8),
					G: uint8(gr >> 8),
					B: uint8(b >> 8),
					A: 200,
				}

				blendedR := uint8((int(overlayColor.R)*int(overlayColor.A) + int(r>>8)*(255-int(overlayColor.A))) / 255)
				blendedG := uint8((int(overlayColor.G)*int(overlayColor.A) + int(gr>>8)*(255-int(overlayColor.A))) / 255)
				blendedB := uint8((int(overlayColor.B)*int(overlayColor.A) + int(b>>8)*(255-int(overlayColor.A))) / 255)

				g.sliderImage.Set(x, y, color.RGBA{
					R: blendedR,
					G: blendedG,
					B: blendedB,
					A: uint8(maskAlpha >> 8),
				})
			} else {
				g.sliderImage.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: uint8(maskAlpha >> 8)})
			}
		}
	}

	g.addSliderShadow()
	g.addSliderEdgeHighlight()
}

func (g *CaptchaImageGenerator) addSliderShadow() {
	shadow := image.NewRGBA(image.Rect(0, 0, SliderWidth+6, SliderHeight+6))
	draw.Draw(shadow, shadow.Bounds(), image.Transparent, image.ZP, draw.Over)

	for y := 3; y < SliderHeight+3; y++ {
		for x := 3; x < SliderWidth+3; x++ {
			sliderX := x - 3
			sliderY := y - 3
			if sliderX >= 0 && sliderX < SliderWidth && sliderY >= 0 && sliderY < SliderHeight {
				shadow.Set(x, y, g.sliderImage.At(sliderX, sliderY))
			}
		}
	}

	for y := 0; y < SliderHeight+6; y++ {
		for x := 0; x < SliderWidth+6; x++ {
			_, _, _, shadowAlpha := shadow.At(x, y).RGBA()
			if shadowAlpha > 0 {
				continue
			}

			offset := 3
			shadowStrength := uint8(0)

			if x >= offset && x < SliderWidth+offset && y >= offset && y < SliderHeight+offset {
				isEdge := false
				if x == offset || x == SliderWidth+offset-1 {
					isEdge = true
				}
				if y == offset || y == SliderHeight+offset-1 {
					isEdge = true
				}

				if isEdge {
					shadowStrength = 60
				} else {
					shadowStrength = 40
				}
			}

			if shadowStrength > 0 {
				r, gr, b, _ := shadow.At(x, y).RGBA()
				shadow.Set(x, y, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(gr >> 8),
					B: uint8(b >> 8),
					A: shadowStrength,
				})
			}
		}
	}

	g.sliderImage = image.NewRGBA(image.Rect(0, 0, SliderWidth+6, SliderHeight+6))
	draw.Draw(g.sliderImage, g.sliderImage.Bounds(), shadow, image.ZP, draw.Over)
}

func (g *CaptchaImageGenerator) addSliderEdgeHighlight() {
	bounds := g.sliderImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			_, _, _, a := g.sliderImage.At(x, y).RGBA()
			if a == 0 {
				continue
			}

			isEdge := false
			if x > 0 {
				_, _, _, leftA := g.sliderImage.At(x-1, y).RGBA()
				if leftA == 0 {
					isEdge = true
				}
			}
			if x < width-1 {
				_, _, _, rightA := g.sliderImage.At(x+1, y).RGBA()
				if rightA == 0 {
					isEdge = true
				}
			}
			if y > 0 {
				_, _, _, topA := g.sliderImage.At(x, y-1).RGBA()
				if topA == 0 {
					isEdge = true
				}
			}
			if y < height-1 {
				_, _, _, bottomA := g.sliderImage.At(x, y+1).RGBA()
				if bottomA == 0 {
					isEdge = true
				}
			}

			if isEdge {
				r, gr, b, _ := g.sliderImage.At(x, y).RGBA()
				newR := uint8(math.Min(255, float64(uint8(r>>8))+50))
				newG := uint8(math.Min(255, float64(uint8(gr>>8))+50))
				newB := uint8(math.Min(255, float64(uint8(b>>8))+50))
				g.sliderImage.Set(x, y, color.RGBA{
					R: newR,
					G: newG,
					B: newB,
					A: uint8(a >> 8),
				})
			}
		}
	}
}

func (g *CaptchaImageGenerator) applyBackgroundEffects() {
	g.addNoiseEffect()
	g.addInterferenceLines()
	g.addInterferenceBlocks()
	g.addTextOverlay()
	g.addPuzzleGap()
}

func (g *CaptchaImageGenerator) addNoiseEffect() {
	noiseIntensity := 15

	for i := 0; i < 2000; i++ {
		x := rand.Intn(ImageWidth)
		y := rand.Intn(ImageHeight)

		r, gr, b, a := g.background.At(x, y).RGBA()

		noise := int8(rand.Intn(noiseIntensity*2) - noiseIntensity)

		newR := uint8(min(255, max(0, int(r>>8)+int(noise))))
		newG := uint8(min(255, max(0, int(gr>>8)+int(noise))))
		newB := uint8(min(255, max(0, int(b>>8)+int(noise))))

		g.background.Set(x, y, color.RGBA{R: newR, G: newG, B: newB, A: uint8(a >> 8)})
	}

	for i := 0; i < 500; i++ {
		x := rand.Intn(ImageWidth)
		y := rand.Intn(ImageHeight)

		r, gr, b, _ := g.background.At(x, y).RGBA()

		g.background.Set(x, y, color.RGBA{
			R: uint8(r >> 8),
			G: uint8(gr >> 8),
			B: uint8(b >> 8),
			A: 40,
		})
	}
}

func (g *CaptchaImageGenerator) addInterferenceLines() {
	lineCount := 8 + rand.Intn(8)

	for i := 0; i < lineCount; i++ {
		startX := rand.Intn(ImageWidth)
		startY := rand.Intn(ImageHeight)
		length := 30 + rand.Intn(100)

		isHorizontal := rand.Float32() < 0.5

		if isHorizontal {
			endX := startX + length
			endY := startY
			g.drawLine(startX, startY, endX, endY, 1+rand.Intn(2))
		} else {
			endX := startX
			endY := startY + length
			g.drawLine(startX, startY, endX, endY, 1+rand.Intn(2))
		}
	}
}

func (g *CaptchaImageGenerator) drawLine(x1, y1, x2, y2, width int) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)

	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}

	err := dx - dy

	for {
		for w := -width / 2; w <= width/2; w++ {
			px := x1
			py := y1 + w

			if px >= 0 && px < ImageWidth && py >= 0 && py < ImageHeight {
				r, gr, b, _ := g.background.At(px, py).RGBA()
				alpha := uint8(80 + rand.Intn(40))

				newR := uint8(r >> 8)
				newG := uint8(gr >> 8)
				newB := uint8(b >> 8)

				if abs(px-g.targetX) < SliderWidth+10 && abs(py-g.targetY) < SliderHeight+10 {
					continue
				}

				g.background.Set(px, py, color.RGBA{
					R: newR,
					G: newG,
					B: newB,
					A: alpha,
				})
			}
		}

		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func (g *CaptchaImageGenerator) addInterferenceBlocks() {
	blockCount := 3 + rand.Intn(4)

	for i := 0; i < blockCount; i++ {
		x := rand.Intn(ImageWidth - 20)
		y := rand.Intn(ImageHeight - 20)
		width := 15 + rand.Intn(25)
		height := 15 + rand.Intn(25)

		if x+width > g.targetX-5 && x < g.targetX+SliderWidth+5 &&
			y+height > g.targetY-5 && y < g.targetY+SliderHeight+5 {
			continue
		}

		for by := y; by < y+height && by < ImageHeight; by++ {
			for bx := x; bx < x+width && bx < ImageWidth; bx++ {
				r, gr, b, _ := g.background.At(bx, by).RGBA()
				alpha := uint8(30 + rand.Intn(30))

				g.background.Set(bx, by, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(gr >> 8),
					B: uint8(b >> 8),
					A: alpha,
				})
			}
		}
	}
}

func (g *CaptchaImageGenerator) addTextOverlay() {
	fontPatterns := []string{"验证", "安全", "通过", "完成"}

	pattern := fontPatterns[rand.Intn(len(fontPatterns))]

	for i, char := range pattern {
		offsetX := 50 + i*70
		offsetY := 40 + rand.Intn(100)

		charOffset := int(char) % 26

		for dy := 0; dy < 25; dy++ {
			for dx := 0; dx < 25; dx++ {
				px := offsetX + dx
				py := offsetY + dy

				if px >= 0 && px < ImageWidth && py >= 0 && py < ImageHeight {
					patternX := (dx + charOffset*3) % 5
					patternY := (dy + charOffset*2) % 5

					if (patternX+patternY)%3 == 0 {
						r, gr, b, a := g.background.At(px, py).RGBA()

						overlayAlpha := uint8(60)
						overlayR := uint8(255)
						overlayG := uint8(255)
						overlayB := uint8(255)

						newR := uint8((int(overlayR)*int(overlayAlpha) + int(r>>8)*(255-int(overlayAlpha))) / 255)
						newG := uint8((int(overlayG)*int(overlayAlpha) + int(gr>>8)*(255-int(overlayAlpha))) / 255)
						newB := uint8((int(overlayB)*int(overlayAlpha) + int(b>>8)*(255-int(overlayAlpha))) / 255)

						g.background.Set(px, py, color.RGBA{
							R: newR,
							G: newG,
							B: newB,
							A: uint8(a >> 8),
						})
					}
				}
			}
		}
	}
}

func (g *CaptchaImageGenerator) addPuzzleGap() {
	for y := g.targetY - 2; y < g.targetY+SliderHeight+2; y++ {
		for x := g.targetX - 2; x < g.targetX+SliderWidth+2; x++ {
			if x >= 0 && x < ImageWidth && y >= 0 && y < ImageHeight {
				g.background.Set(x, y, color.RGBA{R: 20, G: 20, B: 30, A: 180})
			}
		}
	}
}

func (g *CaptchaImageGenerator) EncodeBackgroundToBase64() string {
	var buf bytes.Buffer
	png.Encode(&buf, g.background)
	return "data:image/png;base64," + buf.String()
}

func (g *CaptchaImageGenerator) EncodeSliderToBase64() string {
	var buf bytes.Buffer
	png.Encode(&buf, g.sliderImage)
	return "data:image/png;base64," + buf.String()
}

func (g *CaptchaImageGenerator) GetBackgroundImage() image.Image {
	return g.background
}

func (g *CaptchaImageGenerator) GetSliderImage() image.Image {
	return g.sliderImage
}

func (g *CaptchaImageGenerator) CacheKey() string {
	data := []byte{
		byte(g.targetX),
		byte(g.targetY),
		byte(g.seed & 0xFF),
		byte((g.seed >> 8) & 0xFF),
		byte((g.seed >> 16) & 0xFF),
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
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
