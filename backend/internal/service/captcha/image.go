package captcha

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"sync"
	"time"
)

var (
	imagePool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
	rgbaPool = sync.Pool{
		New: func() interface{} {
			return image.NewRGBA(image.Rect(0, 0, 320, 160))
		},
	}
)

type ImageGenerator struct {
	width        int
	height       int
	sliderWidth  int
	sliderHeight int
}

type CaptchaResult struct {
	Background []byte
	Slider     []byte
	GapX       int
	GapY       int
}

func NewImageGenerator() *ImageGenerator {
	return &ImageGenerator{
		width:        320,
		height:       160,
		sliderWidth:  40,
		sliderHeight: 40,
	}
}

func (g *ImageGenerator) SetDimensions(width, height, sliderWidth, sliderHeight int) {
	g.width = width
	g.height = height
	g.sliderWidth = sliderWidth
	g.sliderHeight = sliderHeight
}

func (g *ImageGenerator) GenerateSliderCaptcha() (*CaptchaResult, error) {
	g.width = 320
	g.height = 160
	g.sliderWidth = 40
	g.sliderHeight = 40

	background := g.generateBackground()

	gapX := rand.Intn(g.width-g.sliderWidth-20) + 10
	gapY := rand.Intn(g.height-g.sliderHeight-20) + 10

	gap := image.Rect(gapX, gapY, gapX+g.sliderWidth, gapY+g.sliderHeight)

	bgImage := g.applyGap(background, gap)

	sliderImage := g.extractSlider(background, gap)

	bgImage = g.applyEdgeFeather(bgImage, gap)

	bgImage = g.applyAdvancedEdgeDetection(bgImage, gap)

	bgImage = g.applyEnhancedShadowDetection(bgImage, gap)

	bgImage = g.addInterference(bgImage)

	bgData := g.encodePNG(bgImage)
	sliderData := g.encodePNG(sliderImage)

	return &CaptchaResult{
		Background: bgData,
		Slider:     sliderData,
		GapX:       gapX,
		GapY:       gapY,
	}, nil
}

func (g *ImageGenerator) generateBackground() *image.RGBA {
	img := rgbaPool.Get().(*image.RGBA)
	bounds := img.Bounds()
	if bounds.Dx() != g.width || bounds.Dy() != g.height {
		*img = *image.NewRGBA(image.Rect(0, 0, g.width, g.height))
	} else {
		for y := 0; y < g.height; y++ {
			for x := 0; x < g.width; x++ {
				img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			}
		}
	}

	bgType := rand.Intn(8)
	switch bgType {
	case 0:
		g.drawGradientBackground(img)
	case 1:
		g.drawPatternBackground(img)
	case 2:
		g.drawSolidColorBackground(img)
	case 3:
		g.drawNoiseBackground(img)
	case 4:
		g.drawGeometricBackground(img)
	case 5:
		g.drawComplexTextureBackground(img)
	case 6:
		g.drawPerlinLikeNoise(img, uint8(80+rand.Intn(60)), uint8(100+rand.Intn(50)), uint8(120+rand.Intn(40)))
	case 7:
		g.drawMarbleTexture(img, uint8(70+rand.Intn(70)), uint8(90+rand.Intn(50)), uint8(110+rand.Intn(50)))
	default:
		g.drawGradientBackground(img)
	}

	return img
}

func (g *ImageGenerator) recycleBackground(img *image.RGBA) {
	rgbaPool.Put(img)
}

func (g *ImageGenerator) drawGradientBackground(img *image.RGBA) {
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

func (g *ImageGenerator) drawPatternBackground(img *image.RGBA) {
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

func (g *ImageGenerator) drawSolidColorBackground(img *image.RGBA) {
	baseColor := &image.Uniform{
		C: color.RGBA{
			R: uint8(100 + rand.Intn(100)),
			G: uint8(100 + rand.Intn(100)),
			B: uint8(100 + rand.Intn(100)),
			A: 255,
		},
	}

	draw.Draw(img, img.Bounds(), baseColor, image.Point{}, draw.Src)

	noiseCount := 500
	for i := 0; i < noiseCount; i++ {
		x := rand.Intn(g.width)
		y := rand.Intn(g.height)
		noise := int16(rand.Intn(40) - 20)

		orig := img.RGBAAt(x, y)
		r := g.clampUint8(int(orig.R) + int(noise))
		col := g.clampUint8(int(orig.G) + int(noise))
		b := g.clampUint8(int(orig.B) + int(noise))

		img.Set(x, y, color.RGBA{R: r, G: col, B: b, A: 255})
	}
}

func (g *ImageGenerator) drawNoiseBackground(img *image.RGBA) {
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			r := uint8(rand.Intn(180) + 40)
			gVal := uint8(rand.Intn(180) + 40)
			b := uint8(rand.Intn(180) + 40)

			img.Set(x, y, color.RGBA{R: r, G: gVal, B: b, A: 255})
		}
	}
}

func (g *ImageGenerator) drawGeometricBackground(img *image.RGBA) {
	g.drawGradientBackground(img)

	shapeCount := 8 + rand.Intn(8)
	for i := 0; i < shapeCount; i++ {
		shapeType := rand.Intn(3)
		x := rand.Intn(g.width)
		y := rand.Intn(g.height)
		size := 10 + rand.Intn(30)

		c := color.RGBA{
			R: uint8(rand.Intn(200)),
			G: uint8(rand.Intn(200)),
			B: uint8(rand.Intn(200)),
			A: uint8(30 + rand.Intn(50)),
		}

		switch shapeType {
		case 0:
			g.drawFilledCircle(img, x, y, size, c)
		case 1:
			g.drawFilledRect(img, x-size/2, y-size/2, size, size, c)
		case 2:
			g.drawLine(img, rand.Intn(g.width), rand.Intn(g.height),
				rand.Intn(g.width), rand.Intn(g.height), c)
		}
	}
}

func (g *ImageGenerator) drawComplexTextureBackground(img *image.RGBA) {
	baseR := uint8(60 + rand.Intn(80))
	baseG := uint8(80 + rand.Intn(60))
	baseB := uint8(100 + rand.Intn(60))

	textureType := rand.Intn(4)
	switch textureType {
	case 0:
		g.drawPerlinLikeNoise(img, baseR, baseG, baseB)
	case 1:
		g.drawWaveTexture(img, baseR, baseG, baseB)
	case 2:
		g.drawCellularTexture(img, baseR, baseG, baseB)
	case 3:
		g.drawMarbleTexture(img, baseR, baseG, baseB)
	default:
		g.drawPerlinLikeNoise(img, baseR, baseG, baseB)
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawPerlinLikeNoise(img *image.RGBA, baseR, baseG, baseB uint8) {
	octaves := 3
	persistence := 0.5

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise := 0.0
			amplitude := 1.0
			frequency := 0.05
			maxValue := 0.0

			for o := 0; o < octaves; o++ {
				noiseValue := g.simplexNoise2D(float64(x)*frequency, float64(y)*frequency)
				noise += noiseValue * amplitude
				maxValue += amplitude
				amplitude *= persistence
				frequency *= 2.0
			}

			noise = noise / maxValue
			noise = (noise + 1) / 2

			adjustment := int(noise * 60 - 30)

			r := g.clampUint8(int(baseR) + adjustment)
			gc := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}
}

func (g *ImageGenerator) simplexNoise2D(x, y float64) float64 {
	s := (x + y) * 0.5 * (1.414213562 - 1)
	i := int(math.Floor(x + s))
	j := int(math.Floor(y + s))

	t := float64(i+j) * (3.0 - 1.414213562) / 2.0
	X0 := float64(i) - t
	Y0 := float64(j) - t
	x0 := x - X0
	y0 := y - Y0

	var i1, j1 int
	if x0 > y0 {
		i1 = 1
		j1 = 0
	} else {
		i1 = 0
		j1 = 1
	}

	x1 := x0 - float64(i1) + (3.0-1.414213562)/2.0
	y1 := y0 - float64(j1) + (3.0-1.414213562)/2.0
	x2 := x0 - 1.0 + 2.0*(3.0-1.414213562)/2.0
	y2 := y0 - 1.0 + 2.0*(3.0-1.414213562)/2.0

	gi0 := g.hashIJ(i, j) % 8
	gi1 := g.hashIJ(i+i1, j+j1) % 8
	gi2 := g.hashIJ(i+1, j+1) % 8

	grad3 := [][]float64{
		{1, 1, 0}, {-1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
		{1, 0, 1}, {-1, 0, 1}, {1, 0, -1}, {-1, 0, -1},
		{0, 1, 1}, {0, -1, 1}, {0, 1, -1}, {0, -1, -1},
	}

	n0 := 0.0
	t0 := 0.5 - x0*x0 - y0*y0
	if t0 >= 0 {
		t0 *= t0
		n0 = t0 * t0 * g.dot2(grad3[gi0], x0, y0)
	}

	n1 := 0.0
	t1 := 0.5 - x1*x1 - y1*y1
	if t1 >= 0 {
		t1 *= t1
		n1 = t1 * t1 * g.dot2(grad3[gi1], x1, y1)
	}

	n2 := 0.0
	t2 := 0.5 - x2*x2 - y2*y2
	if t2 >= 0 {
		t2 *= t2
		n2 = t2 * t2 * g.dot2(grad3[gi2], x2, y2)
	}

	return 70.0 * (n0 + n1 + n2)
}

func (g *ImageGenerator) hashIJ(i, j int) int {
	return (i*374761393 + j*668265263) ^ (i*1274126177)
}

func (g *ImageGenerator) dot2(grad []float64, x, y float64) float64 {
	return grad[0]*x + grad[1]*y
}

func (g *ImageGenerator) drawWaveTexture(img *image.RGBA, baseR, baseG, baseB uint8) {
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			wave1 := math.Sin(float64(x)*0.02 + float64(y)*0.01)
			wave2 := math.Cos(float64(x)*0.015 - float64(y)*0.02)
			wave3 := math.Sin(float64(x)*0.01 + float64(y)*0.03)

			combined := (wave1 + wave2*0.7 + wave3*0.5) / 2.2
			adjustment := int(combined * 40)

			r := g.clampUint8(int(baseR) + adjustment)
			gc := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}
}

func (g *ImageGenerator) drawCellularTexture(img *image.RGBA, baseR, baseG, baseB uint8) {
	cells := make([]struct{ x, y float64 }, 15)
	for i := range cells {
		cells[i].x = float64(rand.Intn(g.width))
		cells[i].y = float64(rand.Intn(g.height))
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			minDist1 := float64(g.width + g.height)
			minDist2 := float64(g.width + g.height)

			for _, cell := range cells {
				dx := float64(x) - cell.x
				dy := float64(y) - cell.y
				dist := math.Sqrt(dx*dx + dy*dy)

				if dist < minDist1 {
					minDist2 = minDist1
					minDist1 = dist
				} else if dist < minDist2 {
					minDist2 = dist
				}
			}

			cellValue := minDist2 - minDist1
			normalized := cellValue / 30.0
			if normalized > 1 {
				normalized = 1
			}
			if normalized < 0 {
				normalized = 0
			}

			adjustment := int(normalized * 50 - 25)

			r := g.clampUint8(int(baseR) + adjustment)
			gc := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}
}

func (g *ImageGenerator) drawMarbleTexture(img *image.RGBA, baseR, baseG, baseB uint8) {
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			turbulence := 0.0
			for i := 0; i < 5; i++ {
				scale := math.Pow(2, float64(i))
				turbulence += g.simplexNoise2D(float64(x)/scale, float64(y)/scale) / scale
			}

			marble := math.Sin(float64(x)*0.05 + turbulence*2)

			normalized := (marble + 1) / 2
			adjustment := int(normalized*60 - 30)

			r := g.clampUint8(int(baseR) + adjustment)
			gc := g.clampUint8(int(baseG) + adjustment)
			b := g.clampUint8(int(baseB) + adjustment)

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}
}

func (g *ImageGenerator) applySubtleVignette(img *image.RGBA) {
	centerX := float64(g.width) / 2
	centerY := float64(g.height) / 2
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	vignetteStrength := 0.3

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx + dy*dy)

			vignetteFactor := 1.0 - (dist/maxDist)*vignetteStrength
			if vignetteFactor < 0.7 {
				vignetteFactor = 0.7
			}

			p := img.RGBAAt(x, y)
			r := g.clampUint8(int(float64(p.R) * vignetteFactor))
			gc := g.clampUint8(int(float64(p.G) * vignetteFactor))
			b := g.clampUint8(int(float64(p.B) * vignetteFactor))

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}
}

func (g *ImageGenerator) applyEnhancedShadowDetection(img *image.RGBA, gap image.Rectangle) *image.RGBA {
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

func (g *ImageGenerator) getMinDistanceToRect(x, y int, rect image.Rectangle) float64 {
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

func (g *ImageGenerator) applyAdvancedEdgeDetection(img *image.RGBA, gap image.Rectangle) *image.RGBA {
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

func (g *ImageGenerator) applyEdgePixel(img *image.RGBA, x, y int, gap image.Rectangle, dirX, dirY int) {
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (g *ImageGenerator) applyGap(background *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(background.Bounds())

	draw.Draw(result, result.Bounds(), background, background.Bounds().Min, draw.Src)

	darkColor := &image.Uniform{
		C: color.RGBA{
			R: 40,
			G: 40,
			B: 40,
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
				R: 20,
				G: 20,
				B: 20,
				A: 200,
			},
		}
		draw.Draw(result, innerGap, shadowColor, image.Point{}, draw.Src)
	}

	return result
}

func (g *ImageGenerator) extractSlider(background *image.RGBA, gap image.Rectangle) *image.RGBA {
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

	return sliderImg
}

func (g *ImageGenerator) addSliderBorder(slider *image.RGBA) {
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

func (g *ImageGenerator) applyEdgeFeather(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())

	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	featherRadius := 2.0

	for y := gap.Min.Y - int(featherRadius); y <= gap.Max.Y+int(featherRadius); y++ {
		for x := gap.Min.X - int(featherRadius); x <= gap.Max.X+int(featherRadius); x++ {
			if x < 0 || x >= g.width || y < 0 || y >= g.height {
				continue
			}

			distToEdge := g.getDistanceToRectEdge(x, y, gap)

			if distToEdge < featherRadius && distToEdge >= 0 {
				factor := distToEdge / featherRadius
				alpha := uint8(float64(255) * factor)

				pixel := img.RGBAAt(x, y)
				blended := color.RGBA{
					R: pixel.R,
					G: pixel.G,
					B: pixel.B,
					A: alpha,
				}
				result.SetRGBA(x, y, blended)
			}
		}
	}

	return result
}

func (g *ImageGenerator) getDistanceToRectEdge(x, y int, rect image.Rectangle) float64 {
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
		return 0
	}

	return math.Sqrt(float64(dx*dx + dy*dy))
}

func (g *ImageGenerator) addInterference(img *image.RGBA) *image.RGBA {
	g.addNoiseDots(img, 300)

	g.addCracks(img, 3)

	g.addBrightnessVariation(img)

	g.addSmallCircles(img, 15)

	return img
}

func (g *ImageGenerator) addNoiseDots(img *image.RGBA, count int) {
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

func (g *ImageGenerator) addCracks(img *image.RGBA, count int) {
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

func (g *ImageGenerator) addBrightnessVariation(img *image.RGBA) {
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

func (g *ImageGenerator) addSmallCircles(img *image.RGBA, count int) {
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

func (g *ImageGenerator) drawFilledCircle(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
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

func (g *ImageGenerator) drawFilledRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < g.width && py >= 0 && py < g.height {
				img.Set(px, py, c)
			}
		}
	}
}

func (g *ImageGenerator) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
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

func (g *ImageGenerator) clampUint8(val int) uint8 {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}

func (g *ImageGenerator) encodePNG(img *image.RGBA) []byte {
	buf := imagePool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		imagePool.Put(buf)
	}()

	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(buf, img); err != nil {
		return nil
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result
}

func (g *ImageGenerator) EncodeToBase64(img *image.RGBA) string {
	data := g.encodePNG(img)
	return base64.StdEncoding.EncodeToString(data)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
