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

	bgType := rand.Intn(17)
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
	case 8:
		g.drawBrickTexture(img)
	case 9:
		g.drawWoodTexture(img)
	case 10:
		g.drawMetalTexture(img)
	case 11:
		g.drawGrassTexture(img)
	case 12:
		g.drawDotPatternTexture(img)
	case 13:
		g.drawSpiralTexture(img)
	case 14:
		g.drawGridTexture(img)
	case 15:
		g.drawRadialTexture(img)
	case 16:
		g.drawEnhancedNoiseTexture(img)
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

	result = g.applyEdgeConsistencyEnhancement(result, gap)

	return result
}

func (g *ImageGenerator) applyEdgeConsistencyEnhancement(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	consistencyWindow := 5

	for y := gap.Min.Y; y < gap.Max.Y; y++ {
		for x := gap.Min.X - consistencyWindow; x <= gap.Min.X+consistencyWindow; x++ {
			if x >= 0 && x < g.width {
				consistencyScore := g.calculateEdgeConsistency(img, x, y, gap, true)
				if consistencyScore > 0.7 {
					p := img.RGBAAt(x, y)
					brightness := float64(p.R)*0.299 + float64(p.G)*0.587 + float64(p.B)*0.114
					if brightness < 100 {
						adjustment := 1.15
						r := g.clampUint8(int(float64(p.R) * adjustment))
						gc := g.clampUint8(int(float64(p.G) * adjustment))
						b := g.clampUint8(int(float64(p.B) * adjustment))
						result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
					}
				}
			}
		}
	}

	for y := gap.Min.Y; y < gap.Max.Y; y++ {
		for x := gap.Max.X - consistencyWindow; x <= gap.Max.X+consistencyWindow; x++ {
			if x >= 0 && x < g.width {
				consistencyScore := g.calculateEdgeConsistency(img, x, y, gap, false)
				if consistencyScore > 0.7 {
					p := img.RGBAAt(x, y)
					brightness := float64(p.R)*0.299 + float64(p.G)*0.587 + float64(p.B)*0.114
					if brightness < 100 {
						adjustment := 1.15
						r := g.clampUint8(int(float64(p.R) * adjustment))
						gc := g.clampUint8(int(float64(p.G) * adjustment))
						b := g.clampUint8(int(float64(p.B) * adjustment))
						result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
					}
				}
			}
		}
	}

	return result
}

func (g *ImageGenerator) calculateEdgeConsistency(img *image.RGBA, x, y int, gap image.Rectangle, isLeftEdge bool) float64 {
	brightnessValues := make([]float64, 0)

	searchRange := 10
	for i := -searchRange; i <= searchRange; i++ {
		searchY := y + i
		if searchY >= gap.Min.Y && searchY < gap.Max.Y {
			p := img.RGBAAt(x, searchY)
			brightness := float64(p.R)*0.299 + float64(p.G)*0.587 + float64(p.B)*0.114
			brightnessValues = append(brightnessValues, brightness)
		}
	}

	if len(brightnessValues) < 3 {
		return 0.0
	}

	mean := 0.0
	for _, v := range brightnessValues {
		mean += v
	}
	mean /= float64(len(brightnessValues))

	variance := 0.0
	for _, v := range brightnessValues {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(brightnessValues))

	stdDev := math.Sqrt(variance)

	normalizedVariance := stdDev / mean

	return 1.0 - math.Min(normalizedVariance, 1.0)
}

func (g *ImageGenerator) applyMultiScaleEdgeDetection(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	scales := []int{1, 2, 3}
	scaleWeights := []float64{0.5, 0.3, 0.2}

	for y := gap.Min.Y; y < gap.Max.Y; y++ {
		for x := gap.Min.X; x < gap.Max.X; x++ {
			totalAdjustment := 0.0

			for i, scale := range scales {
				adjustment := g.calculateEdgeAdjustmentAtScale(img, x, y, gap, scale)
				totalAdjustment += adjustment * scaleWeights[i]
			}

			if math.Abs(totalAdjustment) > 0.05 {
				p := img.RGBAAt(x, y)
				if totalAdjustment > 0 {
					r := g.clampUint8(int(float64(p.R) * (1 + totalAdjustment)))
					gc := g.clampUint8(int(float64(p.G) * (1 + totalAdjustment)))
					b := g.clampUint8(int(float64(p.B) * (1 + totalAdjustment)))
					result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
				} else {
					r := g.clampUint8(int(float64(p.R) * (1 + totalAdjustment)))
					gc := g.clampUint8(int(float64(p.G) * (1 + totalAdjustment)))
					b := g.clampUint8(int(float64(p.B) * (1 + totalAdjustment)))
					result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
				}
			}
		}
	}

	return result
}

func (g *ImageGenerator) calculateEdgeAdjustmentAtScale(img *image.RGBA, x, y int, gap image.Rectangle, scale int) float64 {
	isVerticalEdge := (x >= gap.Min.X-scale && x < gap.Min.X) ||
		(x >= gap.Max.X && x < gap.Max.X+scale)
	isHorizontalEdge := (y >= gap.Min.Y-scale && y < gap.Min.Y) ||
		(y >= gap.Max.Y && y < gap.Max.Y+scale)

	if !isVerticalEdge && !isHorizontalEdge {
		return 0.0
	}

	var gradientMagnitude float64
	for ny := -scale; ny <= scale; ny++ {
		for nx := -scale; nx <= scale; nx++ {
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

	adjustment := 0.0
	if gradientMagnitude < 20 {
		adjustment = 0.15
	} else if gradientMagnitude < 40 {
		adjustment = 0.1
	}

	centerX := (gap.Min.X + gap.Max.X) / 2
	centerY := (gap.Min.Y + gap.Max.Y) / 2
	dx := x - centerX
	dy := y - centerY

	lightDir := 0.6
	effectiveLight := lightDir - float64(dy)/float64(g.height)*0.3 + float64(dx)/float64(g.width)*0.1
	if effectiveLight > 1 {
		effectiveLight = 1
	}
	if effectiveLight < 0.3 {
		effectiveLight = 0.3
	}

	lightAdjustment := (effectiveLight - 0.5) * 0.4

	if isVerticalEdge {
		if dx > 0 {
			adjustment += lightAdjustment
		} else {
			adjustment -= lightAdjustment
		}
	}

	if isHorizontalEdge {
		if dy > 0 {
			adjustment += lightAdjustment * 0.8
		} else {
			adjustment -= lightAdjustment * 0.8
		}
	}

	return math.Max(-0.3, math.Min(0.3, adjustment))
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

	sliderImg = g.applyAdvancedSliderProcessing(sliderImg, background, gap)

	return sliderImg
}

func (g *ImageGenerator) applyAdvancedSliderProcessing(slider *image.RGBA, background *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(slider.Bounds())
	draw.Draw(result, result.Bounds(), slider, slider.Bounds().Min, draw.Src)

	result = g.applyAdaptiveEdgeEnhancement(result)

	result = g.applyIntelligentShadowRecovery(result, background, gap)

	result = g.applyHighQualityAntiAliasing(result)

	result = g.applyColorCorrection(result)

	return result
}

func (g *ImageGenerator) applyAdaptiveEdgeEnhancement(img *image.RGBA) *image.RGBA {
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

func (g *ImageGenerator) calculateEdgeStrength(img *image.RGBA) float64 {
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

func (g *ImageGenerator) isEdgePixel(img *image.RGBA, x, y int) bool {
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

func (g *ImageGenerator) applyIntelligentShadowRecovery(slider *image.RGBA, background *image.RGBA, gap image.Rectangle) *image.RGBA {
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

func (g *ImageGenerator) applyHighQualityAntiAliasing(img *image.RGBA) *image.RGBA {
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

func (g *ImageGenerator) applyColorCorrection(img *image.RGBA) *image.RGBA {
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

func (g *ImageGenerator) drawBrickTexture(img *image.RGBA) {
	brickWidth := 40
	brickHeight := 20
	mortarWidth := 3

	baseR := uint8(120 + rand.Intn(40))
	baseG := uint8(80 + rand.Intn(40))
	baseB := uint8(60 + rand.Intn(30))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			row := y / brickHeight
			offset := 0
			if row%2 == 1 {
				offset = brickWidth / 2
			}

			brickX := (x + offset) % brickWidth
			brickY := y % brickHeight

			isMortar := brickX < mortarWidth || brickY < mortarWidth

			var r, gc, b uint8
			if isMortar {
				mortarVariation := int16(rand.Intn(20) - 10)
				r = g.clampUint8(int(baseR) - 60 + int(mortarVariation))
				gc = g.clampUint8(int(baseG) - 60 + int(mortarVariation))
				b = g.clampUint8(int(baseB) - 40 + int(mortarVariation))
			} else {
				brickVariation := int16(rand.Intn(30) - 15)
				r = g.clampUint8(int(baseR) + int(brickVariation))
				gc = g.clampUint8(int(baseG) + int(brickVariation))
				b = g.clampUint8(int(baseB) + int(brickVariation))

				noise := int16(rand.Intn(10) - 5)
				r = g.clampUint8(int(r) + int(noise))
				gc = g.clampUint8(int(gc) + int(noise))
				b = g.clampUint8(int(b) + int(noise))
			}

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawWoodTexture(img *image.RGBA) {
	baseR := uint8(120 + rand.Intn(60))
	baseG := uint8(80 + rand.Intn(40))
	baseB := uint8(50 + rand.Intn(30))

	ringFrequency := 0.05 + rand.Float64()*0.03

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise := g.simplexNoise2D(float64(x)*0.1, float64(y)*0.1) * 20

			woodPattern := math.Sin(float64(y)*ringFrequency+noise) * 15

			grain := g.simplexNoise2D(float64(x)*0.3, float64(y)*0.3) * 10

			r := g.clampUint8(int(baseR) + int(woodPattern) + int(grain))
			gc := g.clampUint8(int(baseG) + int(float64(woodPattern)*0.8) + int(float64(grain)*0.7))
			b := g.clampUint8(int(baseB) + int(float64(woodPattern)*0.5) + int(float64(grain)*0.5))

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawMetalTexture(img *image.RGBA) {
	baseR := uint8(100 + rand.Intn(60))
	baseG := uint8(100 + rand.Intn(60))
	baseB := uint8(120 + rand.Intn(60))

	isBrushed := rand.Float64() > 0.5

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise := g.simplexNoise2D(float64(x)*0.15, float64(y)*0.15) * 30

			var r, gc, b uint8

			if isBrushed {
				brushedLines := math.Sin(float64(y)*0.8) * 10
				r = g.clampUint8(int(baseR) + int(brushedLines) + int(noise))
				gc = g.clampUint8(int(baseG) + int(brushedLines) + int(noise))
				b = g.clampUint8(int(baseB) + int(brushedLines) + int(noise))
			} else {
				brushedLines := math.Sin(float64(x)*0.8) * 10
				r = g.clampUint8(int(baseR) + int(brushedLines) + int(noise))
				gc = g.clampUint8(int(baseG) + int(brushedLines) + int(noise))
				b = g.clampUint8(int(baseB) + int(brushedLines) + int(noise))
			}

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawGrassTexture(img *image.RGBA) {
	baseR := uint8(40 + rand.Intn(30))
	baseG := uint8(80 + rand.Intn(50))
	baseB := uint8(30 + rand.Intn(30))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise1 := g.simplexNoise2D(float64(x)*0.2, float64(y)*0.2) * 25
			noise2 := g.simplexNoise2D(float64(x)*0.5, float64(y)*0.5) * 10

			grassVariation := math.Sin(float64(x)*0.3+float64(y)*0.2) * 8

			r := g.clampUint8(int(baseR) + int(noise1) + int(grassVariation))
			gc := g.clampUint8(int(baseG) + int(noise1) + int(noise2) + int(grassVariation))
			b := g.clampUint8(int(baseB) + int(noise2) + int(grassVariation))

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	bladeCount := 50 + rand.Intn(50)
	for i := 0; i < bladeCount; i++ {
		x := rand.Intn(g.width)
		startY := rand.Intn(g.height)
		length := 5 + rand.Intn(15)
		angle := -0.3 + rand.Float64()*0.6

		for j := 0; j < length; j++ {
			bladeX := x + int(float64(j)*angle)
			bladeY := startY - j

			if bladeX >= 0 && bladeX < g.width && bladeY >= 0 && bladeY < g.height {
				p := img.RGBAAt(bladeX, bladeY)
				darkness := 0.7 + rand.Float64()*0.3
				r := g.clampUint8(int(float64(p.R) * darkness * 0.8))
				gc := g.clampUint8(int(float64(p.G) * darkness))
				b := g.clampUint8(int(float64(p.B) * darkness * 0.8))
				img.Set(bladeX, bladeY, color.RGBA{R: r, G: gc, B: b, A: 255})
			}
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawDotPatternTexture(img *image.RGBA) {
	baseR := uint8(80 + rand.Intn(60))
	baseG := uint8(100 + rand.Intn(50))
	baseB := uint8(120 + rand.Intn(40))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise := g.simplexNoise2D(float64(x)*0.1, float64(y)*0.1) * 30

			dotSpacing := 8 + rand.Intn(4)
			dotRadius := 2 + rand.Intn(2)

			isDot := false
			for dy := -dotRadius; dy <= dotRadius; dy++ {
				for dx := -dotRadius; dx <= dotRadius; dx++ {
					if dx*dx+dy*dy <= dotRadius*dotRadius {
						cx := x + dx
						cy := y + dy
						if cx >= 0 && cx < g.width && cy >= 0 && cy < g.height {
							gridX := cx % dotSpacing
							gridY := cy % dotSpacing
							if gridX >= dotRadius && gridX < dotSpacing-dotRadius &&
								gridY >= dotRadius && gridY < dotSpacing-dotRadius {
								isDot = true
								break
							}
						}
					}
				}
				if isDot {
					break
				}
			}

			var r, gc, b uint8
			if isDot {
				variation := int16(rand.Intn(20) - 10)
				r = g.clampUint8(int(baseR) + int(noise) - 30 + int(variation))
				gc = g.clampUint8(int(baseG) + int(noise) - 30 + int(variation))
				b = g.clampUint8(int(baseB) + int(noise) - 30 + int(variation))
			} else {
				variation := int16(rand.Intn(15))
				r = g.clampUint8(int(baseR) + int(noise) + int(variation))
				gc = g.clampUint8(int(baseG) + int(noise) + int(variation))
				b = g.clampUint8(int(baseB) + int(noise) + int(variation))
			}

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawSpiralTexture(img *image.RGBA) {
	baseR := uint8(90 + rand.Intn(50))
	baseG := uint8(110 + rand.Intn(40))
	baseB := uint8(130 + rand.Intn(30))

	centerX := float64(g.width) / 2
	centerY := float64(g.height) / 2

	spiralCount := 2 + rand.Intn(2)
	spiralTightness := 0.1 + rand.Float64()*0.1

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx + dy*dy)
			angle := math.Atan2(dy, dx)

			spiralValue := 0.0
			for s := 0; s < spiralCount; s++ {
				offset := float64(s) * 2 * math.Pi / float64(spiralCount)
				spiral := math.Sin(angle*float64(spiralCount) + dist*spiralTightness + offset)
				spiralValue += spiral
			}
			spiralValue /= float64(spiralCount)

			noise := g.simplexNoise2D(float64(x)*0.08, float64(y)*0.08) * 20

			adjustment := spiralValue*30 + noise

			r := g.clampUint8(int(baseR) + int(adjustment))
			gc := g.clampUint8(int(baseG) + int(adjustment))
			b := g.clampUint8(int(baseB) + int(adjustment))

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawGridTexture(img *image.RGBA) {
	baseR := uint8(70 + rand.Intn(60))
	baseG := uint8(90 + rand.Intn(50))
	baseB := uint8(110 + rand.Intn(40))

	gridSize := 20 + rand.Intn(20)
	lineWidth := 1 + rand.Intn(2)

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			isGridLine := (x%gridSize < lineWidth) || (y%gridSize < lineWidth)

			noise := g.simplexNoise2D(float64(x)*0.1, float64(y)*0.1) * 25

			var r, gc, b uint8
			if isGridLine {
				variation := int16(rand.Intn(15))
				r = g.clampUint8(int(baseR) + int(noise) + 20 + int(variation))
				gc = g.clampUint8(int(baseG) + int(noise) + 20 + int(variation))
				b = g.clampUint8(int(baseB) + int(noise) + 20 + int(variation))
			} else {
				variation := int16(rand.Intn(10))
				r = g.clampUint8(int(baseR) + int(noise) + int(variation))
				gc = g.clampUint8(int(baseG) + int(noise) + int(variation))
				b = g.clampUint8(int(baseB) + int(noise) + int(variation))
			}

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawRadialTexture(img *image.RGBA) {
	baseR := uint8(85 + rand.Intn(55))
	baseG := uint8(105 + rand.Intn(45))
	baseB := uint8(125 + rand.Intn(35))

	centerX := float64(g.width) / 2
	centerY := float64(g.height) / 2

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx + dy*dy)
			angle := math.Atan2(dy, dx)

			maxDist := math.Sqrt(centerX*centerX + centerY*centerY)
			normalizedDist := dist / maxDist

			wave1 := math.Sin(normalizedDist * 10 * math.Pi)
			wave2 := math.Sin(angle * 6)
			wave3 := math.Sin(angle*8 + normalizedDist*5)

			combined := (wave1*0.5 + wave2*0.3 + wave3*0.2)

			noise := g.simplexNoise2D(float64(x)*0.12, float64(y)*0.12) * 25

			adjustment := combined*35 + noise - normalizedDist*20

			r := g.clampUint8(int(baseR) + int(adjustment))
			gc := g.clampUint8(int(baseG) + int(adjustment))
			b := g.clampUint8(int(baseB) + int(adjustment))

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) drawEnhancedNoiseTexture(img *image.RGBA) {
	baseR := uint8(75 + rand.Intn(65))
	baseG := uint8(95 + rand.Intn(55))
	baseB := uint8(115 + rand.Intn(45))

	octaves := 4
	persistence := 0.5

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			noise := 0.0
			amplitude := 1.0
			frequency := 0.03
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

			highFreqNoise := g.simplexNoise2D(float64(x)*0.2, float64(y)*0.2)
			highFreqNoise = (highFreqNoise + 1) / 2

			combinedNoise := noise*0.7 + highFreqNoise*0.3

			// Increase adjustment range for more variance
			adjustment := combinedNoise*150 - 75

			r := g.clampUint8(int(baseR) + int(adjustment))
			gc := g.clampUint8(int(baseG) + int(adjustment))
			b := g.clampUint8(int(baseB) + int(adjustment))

			img.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
		}
	}

	g.applySubtleVignette(img)
}

func (g *ImageGenerator) applyMultiLayerShadowDetection(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	result = g.applySoftShadow(result, gap, 8, 0.3)

	result = g.applyMediumShadow(result, gap, 4, 0.5)

	result = g.applyHardShadow(result, gap, 2, 0.7)

	result = g.applyAmbientOcclusion(result, gap)

	return result
}

func (g *ImageGenerator) applySoftShadow(img *image.RGBA, gap image.Rectangle, blur int, intensity float64) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	for y := gap.Min.Y - blur*2; y <= gap.Max.Y+blur*2; y++ {
		for x := gap.Min.X - blur*2; x <= gap.Max.X+blur*2; x++ {
			if x < 0 || x >= g.width || y < 0 || y >= g.height {
				continue
			}

			isInsideGap := x >= gap.Min.X && x < gap.Max.X && y >= gap.Min.Y && y < gap.Max.Y
			if isInsideGap {
				continue
			}

			distToGap := g.getMinDistanceToRect(x, y, gap)
			if distToGap <= float64(blur*2) {
				blurFactor := 1.0 - distToGap/float64(blur*2)

				offsetX := 2
				offsetY := 2
				shadowX := x + offsetX
				shadowY := y + offsetY

				if shadowX >= gap.Min.X && shadowX < gap.Max.X &&
					shadowY >= gap.Min.Y && shadowY < gap.Max.Y {
					p := result.RGBAAt(x, y)

					shadowColor := g.calculateShadowColorFromRGBA(p.R, p.G, p.B, intensity*blurFactor)

					result.Set(x, y, shadowColor)
				}
			}
		}
	}

	return result
}

func (g *ImageGenerator) applyMediumShadow(img *image.RGBA, gap image.Rectangle, blur int, intensity float64) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	for y := gap.Min.Y - blur; y <= gap.Max.Y+blur; y++ {
		for x := gap.Min.X - blur; x <= gap.Max.X+blur; x++ {
			if x < 0 || x >= g.width || y < 0 || y >= g.height {
				continue
			}

			isInsideGap := x >= gap.Min.X && x < gap.Max.X && y >= gap.Min.Y && y < gap.Max.Y
			if isInsideGap {
				continue
			}

			distToGap := g.getMinDistanceToRect(x, y, gap)
			if distToGap <= float64(blur) {
				blurFactor := 1.0 - distToGap/float64(blur)
				effectiveIntensity := blurFactor * intensity

				offsetX := 1
				offsetY := 1
				shadowX := x + offsetX
				shadowY := y + offsetY

				if shadowX >= gap.Min.X && shadowX < gap.Max.X &&
					shadowY >= gap.Min.Y && shadowY < gap.Max.Y {
					p := result.RGBAAt(x, y)

					shadowColor := g.calculateShadowColorFromRGBA(p.R, p.G, p.B, effectiveIntensity)

					result.Set(x, y, shadowColor)
				}
			}
		}
	}

	return result
}

func (g *ImageGenerator) applyHardShadow(img *image.RGBA, gap image.Rectangle, offset int, intensity float64) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	for y := gap.Min.Y; y < gap.Max.Y; y++ {
		for x := gap.Min.X; x < gap.Max.X; x++ {
			shadowX := x + offset
			shadowY := y + offset

			if shadowX >= 0 && shadowX < g.width && shadowY >= 0 && shadowY < g.height {
				p := result.RGBAAt(shadowX, shadowY)
				shadowColor := g.calculateShadowColorFromRGBA(p.R, p.G, p.B, intensity)
				result.Set(shadowX, shadowY, shadowColor)
			}
		}
	}

	return result
}

func (g *ImageGenerator) applyAmbientOcclusion(img *image.RGBA, gap image.Rectangle) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	radius := 3

	for y := gap.Min.Y - radius; y <= gap.Max.Y+radius; y++ {
		for x := gap.Min.X - radius; x <= gap.Max.X+radius; x++ {
			if x < 0 || x >= g.width || y < 0 || y >= g.height {
				continue
			}

			isInsideGap := x >= gap.Min.X && x < gap.Max.X && y >= gap.Min.Y && y < gap.Max.Y
			if isInsideGap {
				continue
			}

			distToGap := g.getMinDistanceToRect(x, y, gap)
			if distToGap <= float64(radius) {
				aoFactor := 1.0 - (distToGap / float64(radius)) * 0.3

				p := result.RGBAAt(x, y)
				r := g.clampUint8(int(float64(p.R) * aoFactor))
				gc := g.clampUint8(int(float64(p.G) * aoFactor))
				b := g.clampUint8(int(float64(p.B) * aoFactor))

				result.Set(x, y, color.RGBA{R: r, G: gc, B: b, A: 255})
			}
		}
	}

	return result
}

func (g *ImageGenerator) calculateShadowColor(p color.RGBA, intensity float64) color.RGBA {
	shadowFactor := 1.0 - intensity

	brightness := float64(p.R)*0.299 + float64(p.G)*0.587 + float64(p.B)*0.114

	var shadowR, shadowG, shadowB uint8
	if brightness > 128 {
		shadowR = g.clampUint8(int(float64(p.R) * shadowFactor))
		shadowG = g.clampUint8(int(float64(p.G) * shadowFactor))
		shadowB = g.clampUint8(int(float64(p.B) * shadowFactor))
	} else {
		shadowR = g.clampUint8(int(float64(p.R) * (1.0 + intensity * 0.2)))
		shadowG = g.clampUint8(int(float64(p.G) * (1.0 + intensity * 0.2)))
		shadowB = g.clampUint8(int(float64(p.B) * (1.0 + intensity * 0.2)))
	}

	return color.RGBA{R: shadowR, G: shadowG, B: shadowB, A: 255}
}

func (g *ImageGenerator) calculateShadowColorFromRGBA(r, gv, b uint8, intensity float64) color.RGBA {
	shadowFactor := 1.0 - intensity

	brightness := float64(r)*0.299 + float64(gv)*0.587 + float64(b)*0.114

	var shadowR, shadowGv, shadowB uint8
	if brightness > 128 {
		shadowR = g.clampUint8(int(float64(r) * shadowFactor))
		shadowGv = g.clampUint8(int(float64(gv) * shadowFactor))
		shadowB = g.clampUint8(int(float64(b) * shadowFactor))
	} else {
		shadowR = g.clampUint8(int(float64(r) * (1.0 + intensity * 0.2)))
		shadowGv = g.clampUint8(int(float64(gv) * (1.0 + intensity * 0.2)))
		shadowB = g.clampUint8(int(float64(b) * (1.0 + intensity * 0.2)))
	}

	return color.RGBA{R: shadowR, G: shadowGv, B: shadowB, A: 255}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
