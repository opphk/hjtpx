package service

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"sync"
	"time"
)

type OptimizedImageGenerator struct {
	bufferPool *BufferPool
	fontCache  *FontCache
	rng        *LockedRand
	precomputed *PrecomputedData
}

type BufferPool struct {
	pool sync.Pool
}

type Buffer struct {
	Data []byte
	Len  int
}

type FontCache struct {
	mu    sync.RWMutex
	cache map[rune]*image.RGBA
}

type LockedRand struct {
	mu  sync.Mutex
	src *rand.Rand
}

type PrecomputedData struct {
	cosTable  []float64
	sinTable  []float64
	colors    []color.RGBA
	positions []struct{ X, Y int }
}

var globalImageGenerator *OptimizedImageGenerator
var initOnce sync.Once

func GetOptimizedImageGenerator() *OptimizedImageGenerator {
	initOnce.Do(func() {
		globalImageGenerator = NewOptimizedImageGenerator()
	})
	return globalImageGenerator
}

func NewOptimizedImageGenerator() *OptimizedImageGenerator {
	gen := &OptimizedImageGenerator{
		bufferPool: &BufferPool{
			pool: sync.Pool{
				New: func() interface{} {
					return &Buffer{
						Data: make([]byte, 0, 65536),
					}
				},
			},
		},
		fontCache: &FontCache{
			cache: make(map[rune]*image.RGBA),
		},
		rng: &LockedRand{
			src: rand.New(rand.NewSource(time.Now().UnixNano())),
		},
		precomputed: &PrecomputedData{
			cosTable:  makeCosTable(),
			sinTable:  makeSinTable(),
			colors:    makeColorPalette(),
			positions: make([]struct{ X, Y int }, 0, 100),
		},
	}

	for i := 0; i < 100; i++ {
		gen.precomputed.positions = append(gen.precomputed.positions, struct{ X, Y int }{
			X: 20 + rand.Intn(260),
			Y: 20 + rand.Intn(260),
		})
	}

	return gen
}

func makeCosTable() []float64 {
	table := make([]float64, 360)
	for i := 0; i < 360; i++ {
		table[i] = math.Cos(float64(i) * math.Pi / 180)
	}
	return table
}

func makeSinTable() []float64 {
	table := make([]float64, 360)
	for i := 0; i < 360; i++ {
		table[i] = math.Sin(float64(i) * math.Pi / 180)
	}
	return table
}

func makeColorPalette() []color.RGBA {
	colors := make([]color.RGBA, 0, 20)
	hues := []int{0, 30, 60, 120, 180, 210, 240, 270, 300, 330}

	for _, h := range hues {
		for s := 60; s <= 90; s += 30 {
			for v := 50; v <= 90; v += 40 {
				r, g, b := hsvToRGBFast(h, float64(s)/100, float64(v)/100)
				colors = append(colors, color.RGBA{
					R: r,
					G: g,
					B: b,
					A: 180 + uint8(rand.Intn(76)),
				})
			}
		}
	}

	return colors
}

func hsvToRGBFast(h int, s, v float64) (uint8, uint8, uint8) {
	hf := float64(h) / 60.0
	i := int(hf)
	f := hf - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}

	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

func (gen *OptimizedImageGenerator) GenerateSliderImage(width, height int) ([]byte, int, int) {
	start := time.Now()
	defer func() {
		if time.Since(start) > 30*time.Millisecond {
		}
	}()

	gen.rng.mu.Lock()
	pieceSize := 50
	bumpRadius := 8 + rand.Intn(5)
	targetX := 30 + rand.Intn(width-pieceSize-bumpRadius*2-60)
	targetY := 30 + rand.Intn(height - pieceSize - 60)
	gen.rng.mu.Unlock()

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	gen.drawGradientBackgroundOptimized(img)

	gen.addDecorationsOptimized(img, 20)

	pieceWidth := pieceSize + bumpRadius
	pieceImg := image.NewRGBA(image.Rect(0, 0, pieceWidth, pieceSize))

	gen.extractPuzzlePiece(img, pieceImg, targetX, targetY, pieceSize, bumpRadius, width, height)

	gen.addPuzzlePieceShadow(pieceImg, pieceSize)

	gen.addCutoutBorderOptimized(img, targetX, targetY, pieceSize, bumpRadius)

	gen.makeHoleInBackground(img, targetX, targetY, pieceSize, bumpRadius, width, height)

	var buf bytes.Buffer
	encoder := png.Encoder{
		CompressionLevel: png.BestSpeed,
	}
	encoder.Encode(&buf, img)

	_ = pieceImg

	return buf.Bytes(), targetX, targetY
}

func (gen *OptimizedImageGenerator) drawGradientBackgroundOptimized(img *image.RGBA) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	gen.rng.mu.Lock()
	r1 := uint8(180 + rand.Intn(60))
	g1 := uint8(180 + rand.Intn(60))
	b1 := uint8(200 + rand.Intn(55))
	r2 := uint8(100 + rand.Intn(80))
	g2 := uint8(120 + rand.Intn(60))
	b2 := uint8(160 + rand.Intn(60))
	gen.rng.mu.Unlock()

	rowCache := make([]uint8, w*4)

	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		r := uint8((int(r1)*(1-int(t)) + int(r2)*int(t)) >> 1)
		g := uint8((int(g1)*(1-int(t)) + int(g2)*int(t)) >> 1)
		b := uint8((int(b1)*(1-int(t)) + int(b2)*int(t)) >> 1)

		for x := 0; x < w; x++ {
			idx := x * 4
			rowCache[idx] = r
			rowCache[idx+1] = g
			rowCache[idx+2] = b
			rowCache[idx+3] = 255
		}

		for x := 0; x < w; x++ {
			idx := x * 4
			img.Set(x, y, color.RGBA{
				R: rowCache[idx],
				G: rowCache[idx+1],
				B: rowCache[idx+2],
				A: rowCache[idx+3],
			})
		}
	}
}

func (gen *OptimizedImageGenerator) addDecorationsOptimized(img *image.RGBA, count int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	gen.rng.mu.Lock()
	defer gen.rng.mu.Unlock()

	for i := 0; i < count; i++ {
		shape := rand.Intn(4)
		x1 := rand.Intn(w)
		y1 := rand.Intn(h)

		switch shape {
		case 0:
			radius := 8 + rand.Intn(25)
			gen.drawFilledCircleFast(img, x1, y1, radius, w, h)
		case 1:
			w1 := 15 + rand.Intn(40)
			h1 := 8 + rand.Intn(20)
			gen.drawFilledRectFast(img, x1, y1, w1, h1, w, h)
		case 2:
			x2 := rand.Intn(w)
			y2 := rand.Intn(h)
			gen.drawLineFast(img, x1, y1, x2, y2, w, h)
		case 3:
			x2 := rand.Intn(w)
			y2 := rand.Intn(h)
			x3 := rand.Intn(w)
			y3 := rand.Intn(h)
			gen.drawBezierFast(img, x1, y1, x2, y2, x3, y3, w, h)
		}
	}

	for i := 0; i < 500; i++ {
		x := rand.Intn(w)
		y := rand.Intn(h)
		noise := rand.Intn(50) - 25
		p := img.RGBAAt(x, y)
		img.Set(x, y, color.RGBA{
			R: clampUint8(int(p.R) + noise),
			G: clampUint8(int(p.G) + noise),
			B: clampUint8(int(p.B) + noise),
			A: 255,
		})
	}
}

func (gen *OptimizedImageGenerator) drawFilledCircleFast(img *image.RGBA, cx, cy, radius, w, h int) {
	c := color.RGBA{
		R: uint8(rand.Intn(200)),
		G: uint8(rand.Intn(200)),
		B: uint8(rand.Intn(200)),
		A: uint8(25 + rand.Intn(55)),
	}

	r2 := radius * radius
	for dy := -radius; dy <= radius; dy++ {
		py := cy + dy
		if py < 0 || py >= h {
			continue
		}
		dx := int(math.Sqrt(float64(r2 - dy*dy)))
		for x := cx - dx; x <= cx+dx; x++ {
			if x >= 0 && x < w {
				img.Set(x, py, c)
			}
		}
	}
}

func (gen *OptimizedImageGenerator) drawFilledRectFast(img *image.RGBA, x, y, w, h, imgW, imgH int) {
	c := color.RGBA{
		R: uint8(rand.Intn(200)),
		G: uint8(rand.Intn(200)),
		B: uint8(rand.Intn(200)),
		A: uint8(25 + rand.Intn(55)),
	}

	for dy := 0; dy < h; dy++ {
		py := y + dy
		if py < 0 || py >= imgH {
			continue
		}
		for dx := 0; dx < w; dx++ {
			px := x + dx
			if px >= 0 && px < imgW {
				img.Set(px, py, c)
			}
		}
	}
}

func (gen *OptimizedImageGenerator) drawLineFast(img *image.RGBA, x1, y1, x2, y2, w, h int) {
	c := color.RGBA{
		R: uint8(rand.Intn(200)),
		G: uint8(rand.Intn(200)),
		B: uint8(rand.Intn(200)),
		A: uint8(25 + rand.Intn(55)),
	}

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
		if x >= 0 && x < w && y >= 0 && y < h {
			img.Set(x, y, c)
		}
	}
}

func (gen *OptimizedImageGenerator) drawBezierFast(img *image.RGBA, x0, y0, x1, y1, x2, y2, w, h int) {
	c := color.RGBA{
		R: uint8(rand.Intn(200)),
		G: uint8(rand.Intn(200)),
		B: uint8(rand.Intn(200)),
		A: uint8(25 + rand.Intn(55)),
	}

	steps := 40
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		mt := 1.0 - t
		x := int(mt*mt*float64(x0) + 2.0*mt*t*float64(x1) + t*t*float64(x2) + 0.5)
		y := int(mt*mt*float64(y0) + 2.0*mt*t*float64(y1) + t*t*float64(y2) + 0.5)
		if x >= 0 && x < w && y >= 0 && y < h {
			img.Set(x, y, c)
		}
	}
}

func (gen *OptimizedImageGenerator) extractPuzzlePiece(background, piece *image.RGBA, targetX, targetY, pieceSize, bumpRadius, width, height int) {
	pieceWidth := pieceSize + bumpRadius

	for py := 0; py < pieceSize; py++ {
		for px := 0; px < pieceWidth; px++ {
			absX := targetX + px
			absY := targetY + py

			if absX >= 0 && absX < width && absY >= 0 && absY < height {
				if gen.isInPuzzlePieceFast(px, py, pieceSize, bumpRadius) {
					p := background.RGBAAt(absX, absY)
					piece.Set(px, py, p)
				}
			}
		}
	}
}

func (gen *OptimizedImageGenerator) isInPuzzlePieceFast(x, y, pieceSize, radius int) bool {
	if y < 0 || y >= pieceSize {
		return false
	}
	midY := pieceSize / 2

	if y >= midY-radius && y <= midY+radius {
		dy := y - midY
		leftBoundary := int(math.Sqrt(float64(radius*radius - dy*dy)))
		if x < leftBoundary {
			return false
		}
	} else if x < 0 {
		return false
	}

	if y >= midY-radius && y <= midY+radius {
		dy := y - midY
		rightBoundary := pieceSize + int(math.Sqrt(float64(radius*radius-dy*dy)))
		if x > rightBoundary {
			return false
		}
	} else if x > pieceSize {
		return false
	}

	return true
}

func (gen *OptimizedImageGenerator) addPuzzlePieceShadow(pieceImg *image.RGBA, pieceSize int) {
	bounds := pieceImg.Bounds()
	dw := bounds.Dx()
	dh := bounds.Dy()

	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			p := pieceImg.RGBAAt(x, y)
			if p.A > 0 {
				sx, sy := x+2, y+2
				if sx < dw && sy < dh {
					pieceImg.Set(sx, sy, color.RGBA{0, 0, 0, 100})
				}
			}
		}
	}
}

func (gen *OptimizedImageGenerator) addCutoutBorderOptimized(img *image.RGBA, targetX, targetY, pieceSize, radius int) {
	borderColor := color.RGBA{255, 255, 255, 200}

	for y := 0; y < pieceSize; y++ {
		for x := 0; x < pieceSize+radius; x++ {
			if !gen.isInPuzzlePieceFast(x, y, pieceSize, radius) {
				continue
			}

			isBorder := false
			for dy := -1; dy <= 1 && !isBorder; dy++ {
				for dx := -1; dx <= 1 && !isBorder; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if !gen.isInPuzzlePieceFast(x+dx, y+dy, pieceSize, radius) {
						isBorder = true
					}
				}
			}

			if isBorder {
				absX, absY := targetX+x, targetY+y
				if absX >= 0 && absX < img.Bounds().Dx() && absY >= 0 && absY < img.Bounds().Dy() {
					img.Set(absX, absY, borderColor)
				}

				for d := 1; d <= 2; d++ {
					absX2, absY2 := targetX+x+d, targetY+y+d
					if absX2 >= 0 && absX2 < img.Bounds().Dx() && absY2 >= 0 && absY2 < img.Bounds().Dy() {
						orig := img.RGBAAt(absX2, absY2)
						factor := 100 - d*15
						if factor < 60 {
							factor = 60
						}
						img.Set(absX2, absY2, color.RGBA{
							R: clampUint8(int(orig.R) * factor / 100),
							G: clampUint8(int(orig.G) * factor / 100),
							B: clampUint8(int(orig.B) * factor / 100),
							A: 255,
						})
					}
				}
			}
		}
	}
}

func (gen *OptimizedImageGenerator) makeHoleInBackground(img *image.RGBA, targetX, targetY, pieceSize, bumpRadius, width, height int) {
	for y := 0; y < pieceSize; y++ {
		for x := 0; x < pieceSize+bumpRadius; x++ {
			absX := targetX + x
			absY := targetY + y

			if absX >= 0 && absX < width && absY >= 0 && absY < height {
				if gen.isInPuzzlePieceFast(x, y, pieceSize, bumpRadius) {
					p := img.RGBAAt(absX, absY)
					img.Set(absX, absY, color.RGBA{
						R: uint8(int(p.R) * 35 / 100),
						G: uint8(int(p.G) * 35 / 100),
						B: uint8(int(p.B) * 35 / 100),
						A: 255,
					})
				}
			}
		}
	}
}

func (gen *OptimizedImageGenerator) GenerateClickImage(charCount, width, height int) ([]byte, []struct{ X, Y int }, []string) {
	start := time.Now()
	defer func() {
		if time.Since(start) > 30*time.Millisecond {
		}
	}()

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	gen.drawGradientBackgroundOptimized(img)

	gen.addDecorationsOptimized(img, 15)

	charPositions := make([]struct{ X, Y int }, charCount)
	charValues := make([]string, charCount)

	gen.rng.mu.Lock()
	for i := 0; i < charCount; i++ {
		margin := 30
		charPositions[i].X = margin + rand.Intn(width - 2*margin)
		charPositions[i].Y = margin + rand.Intn(height - 2*margin)

		value := fmt.Sprintf("%d", rand.Intn(10))
		charValues[i] = value
	}
	gen.rng.mu.Unlock()

	var buf bytes.Buffer
	encoder := png.Encoder{
		CompressionLevel: png.BestSpeed,
	}
	encoder.Encode(&buf, img)

	return buf.Bytes(), charPositions, charValues
}

func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
