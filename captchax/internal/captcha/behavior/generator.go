package behavior

import (
	"bytes"
	"captchax/internal/imageutil"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"time"
)

type Generator struct {
	width        int
	height       int
	targetSize   int
	colorPalette [][]uint8
}

func NewGenerator(width, height int) *Generator {
	if width == 0 {
		width = 320
	}
	if height == 0 {
		height = 240
	}

	palette := [][]uint8{
		{255, 87, 34},    // 橙色
		{33, 150, 243},   // 蓝色
		{76, 175, 80},    // 绿色
		{156, 39, 176},   // 紫色
		{255, 152, 0},    // 深橙色
		{0, 188, 212},    // 青色
		{244, 67, 54},    // 红色
		{103, 58, 183},   // 深紫色
	}

	return &Generator{
		width:        width,
		height:       height,
		targetSize:   32,
		colorPalette: palette,
	}
}

func (g *Generator) GenerateBackground() image.Image {
	bg := image.NewRGBA(image.Rect(0, 0, g.width, g.height))

	baseColor := uint8(235 + rand.Intn(20))
	bgColor := &imageutil.SolidColor{
		R: baseColor,
		G: baseColor,
		B: baseColor,
		A: 255,
	}
	draw.Draw(bg, bg.Bounds(), bgColor, image.ZP, draw.Src)

	g.drawNoisePattern(bg)
	g.drawGeometricShapes(bg)
	g.drawGridLines(bg)

	return bg
}

func (g *Generator) drawNoisePattern(bg *image.RGBA) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		x := r.Intn(g.width)
		y := r.Intn(g.height)
		size := 1 + r.Intn(3)

		noiseColor := color.RGBA{
			R: uint8(r.Intn(40)),
			G: uint8(r.Intn(40)),
			B: uint8(r.Intn(40)),
			A: uint8(10 + r.Intn(20)),
		}

		switch r.Intn(4) {
		case 0:
			imageutil.DrawCircle(bg, x, y, size, noiseColor)
		case 1:
			imageutil.DrawRect(bg, x, y, size, size, noiseColor)
		case 2:
			imageutil.DrawLine(bg, x, y, x+size*2, y+size*2, noiseColor)
		case 3:
			imageutil.DrawLine(bg, x+size*2, y, x, y+size*2, noiseColor)
		}
	}
}

func (g *Generator) drawGeometricShapes(bg *image.RGBA) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	shapeCount := 5 + r.Intn(8)
	for i := 0; i < shapeCount; i++ {
		x := r.Intn(g.width)
		y := r.Intn(g.height)
		size := 10 + r.Intn(25)

		shapeColor := color.RGBA{
			R: uint8(r.Intn(50)),
			G: uint8(r.Intn(50)),
			B: uint8(r.Intn(50)),
			A: uint8(15 + r.Intn(25)),
		}

		switch r.Intn(3) {
		case 0:
			imageutil.DrawCircle(bg, x, y, size, shapeColor)
		case 1:
			imageutil.DrawRect(bg, x, y, size, size, shapeColor)
		case 2:
			points := g.generatePolygonPoints(x, y, size, 6)
			g.drawPolygon(bg, points, shapeColor)
		}
	}
}

func (g *Generator) generatePolygonPoints(cx, cy, radius, sides int) []image.Point {
	points := make([]image.Point, sides)

	for i := 0; i < sides; i++ {
		angle := 2 * math.Pi * float64(i) / float64(sides)
		x := cx + int(float64(radius)*math.Cos(angle))
		y := cy + int(float64(radius)*math.Sin(angle))
		points[i] = image.Point{X: x, Y: y}
	}

	return points
}

func (g *Generator) drawPolygon(bg *image.RGBA, points []image.Point, col color.RGBA) {
	if len(points) < 3 {
		return
	}

	for i := 0; i < len(points); i++ {
		next := (i + 1) % len(points)
		imageutil.DrawLine(bg, points[i].X, points[i].Y, points[next].X, points[next].Y, col)
	}
}

func (g *Generator) drawGridLines(bg *image.RGBA) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	if r.Intn(2) == 0 {
		return
	}

	gridColor := color.RGBA{
		R: 180,
		G: 180,
		B: 180,
		A: 15,
	}

	gridSpacing := 40 + r.Intn(20)

	for x := gridSpacing; x < g.width; x += gridSpacing {
		imageutil.DrawLine(bg, x, 0, x, g.height, gridColor)
	}

	for y := gridSpacing; y < g.height; y += gridSpacing {
		imageutil.DrawLine(bg, 0, y, g.width, y, gridColor)
	}
}

func (g *Generator) GenerateTargetWithHighlight(baseImg image.Image, targets []Point, challengeType string) image.Image {
	width := baseImg.Bounds().Dx()
	height := baseImg.Bounds().Dy()
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(result, result.Bounds(), baseImg, image.ZP, draw.Over)

	targetSize := g.targetSize
	if challengeType == "hover_sequence" {
		targetSize = 24
	}

	for i, target := range targets {
		colorIdx := i % len(g.colorPalette)
		targetColor := g.colorPalette[colorIdx]

		outerGlow := color.RGBA{
			R: targetColor[0],
			G: targetColor[1],
			B: targetColor[2],
			A: 60,
		}
		imageutil.DrawCircle(result, target.X, target.Y, targetSize+10, outerGlow)

		midGlow := color.RGBA{
			R: targetColor[0],
			G: targetColor[1],
			B: targetColor[2],
			A: 120,
		}
		imageutil.DrawCircle(result, target.X, target.Y, targetSize+5, midGlow)

		mainColor := color.RGBA{
			R: targetColor[0],
			G: targetColor[1],
			B: targetColor[2],
			A: 255,
		}
		imageutil.DrawCircle(result, target.X, target.Y, targetSize, mainColor)

		innerHighlight := color.RGBA{
			R: uint8(math.Min(255, float64(targetColor[0])+60)),
			G: uint8(math.Min(255, float64(targetColor[1])+60)),
			B: uint8(math.Min(255, float64(targetColor[2])+60)),
			A: 255,
		}
		imageutil.DrawCircle(result, target.X, target.Y, targetSize/2, innerHighlight)

		centerDot := color.RGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 200,
		}
		imageutil.DrawCircle(result, target.X, target.Y, targetSize/4, centerDot)
	}

	return result
}

func (g *Generator) GenerateWithGuide(baseImg image.Image, guidePoints []GuidePoint, challengeType string) image.Image {
	result := g.GenerateTargetWithHighlight(baseImg, pointsFromGuide(guidePoints), challengeType)

	return result
}

func (g *Generator) GenerateThumb(img image.Image, maxWidth, maxHeight int) image.Image {
	srcBounds := img.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	if srcWidth <= maxWidth && srcHeight <= maxHeight {
		return img
	}

	ratio := math.Min(
		float64(maxWidth)/float64(srcWidth),
		float64(maxHeight)/float64(srcHeight),
	)

	newWidth := int(float64(srcWidth) * ratio)
	newHeight := int(float64(srcHeight) * ratio)

	thumb := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.Draw(thumb, thumb.Bounds(), img, srcBounds.Min, draw.Src)

	return thumb
}

func (g *Generator) ImageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func pointsFromGuide(guidePoints []GuidePoint) []Point {
	points := make([]Point, len(guidePoints))
	for i, gp := range guidePoints {
		points[i] = Point{X: gp.X, Y: gp.Y}
	}
	return points
}

func (g *Generator) SetTargetSize(size int) {
	if size > 0 && size <= 100 {
		g.targetSize = size
	}
}

func (g *Generator) AddCustomColor(color []uint8) {
	if len(color) == 3 {
		g.colorPalette = append(g.colorPalette, color)
	}
}

func (g *Generator) SetColorPalette(palette [][]uint8) {
	if len(palette) > 0 {
		g.colorPalette = palette
	}
}
