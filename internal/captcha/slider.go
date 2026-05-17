package captcha

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	mrand "math/rand"
	"sync"
)

const (
	SliderWidth   = 300
	SliderHeight  = 150
	BlockWidth    = 50
	BlockHeight   = 50
	CanvasWidth   = SliderWidth + 20
	CanvasHeight  = SliderHeight + 20
)

type SliderGenerator struct {
	mu sync.Mutex
}

type SliderResult struct {
	BackgroundImage string
	SliderImage    string
	X              int
	Y              int
	Token          string
}

func NewSliderGenerator() *SliderGenerator {
	return &SliderGenerator{}
}

func (g *SliderGenerator) Generate() (*SliderResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	x := randRange(30, CanvasWidth-BlockWidth-30)
	y := randRange(20, CanvasHeight-BlockHeight-20)

	bgImage, sliderImage, err := g.createSliderImages(x, y)
	if err != nil {
		return nil, err
	}

	bgBase64 := g.encodeToBase64(bgImage)
	sliderBase64 := g.encodeToBase64(sliderImage)

	tokenBytes := make([]byte, 16)
	cryptorand.Read(tokenBytes)
	token := fmt.Sprintf("%x", tokenBytes)

	return &SliderResult{
		BackgroundImage: bgBase64,
		SliderImage:     sliderBase64,
		X:               x,
		Y:               y,
		Token:           token,
	}, nil
}

func (g *SliderGenerator) createSliderImages(targetX, targetY int) (image.Image, image.Image, error) {
	bg := image.NewRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))
	slider := image.NewRGBA(image.Rect(0, 0, BlockWidth+6, BlockHeight+6))

	g.drawBackground(bg)
	g.cutBlock(bg, slider, targetX, targetY)

	return bg, slider, nil
}

func (g *SliderGenerator) drawBackground(img *image.RGBA) {
	baseColor := color.RGBA{
		R: uint8(randRange(180, 220)),
		G: uint8(randRange(180, 220)),
		B: uint8(randRange(180, 220)),
		A: 255,
	}

	draw.Draw(img, img.Bounds(), &image.Uniform{baseColor}, image.Point{}, draw.Src)

	for i := 0; i < 50; i++ {
		x := randRange(0, CanvasWidth)
		y := randRange(0, CanvasHeight)
		size := randRange(2, 8)
		circleColor := color.RGBA{
			R: uint8(randRange(150, 200)),
			G: uint8(randRange(150, 200)),
			B: uint8(randRange(150, 200)),
			A: uint8(randRange(50, 150)),
		}
		g.drawCircle(img, x, y, size, circleColor)
	}

	for i := 0; i < 20; i++ {
		x1 := randRange(0, CanvasWidth)
		y1 := randRange(0, CanvasHeight)
		x2 := randRange(0, CanvasWidth)
		y2 := randRange(0, CanvasHeight)
		lineColor := color.RGBA{
			R: uint8(randRange(100, 150)),
			G: uint8(randRange(100, 150)),
			B: uint8(randRange(100, 150)),
			A: uint8(randRange(30, 80)),
		}
		g.drawLine(img, x1, y1, x2, y2, 1, lineColor)
	}
}

func (g *SliderGenerator) cutBlock(bg *image.RGBA, slider *image.RGBA, x, y int) {
	sliderStartX := x - 3
	sliderStartY := y - 3

	draw.Draw(slider, slider.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	for py := 0; py < BlockHeight+6; py++ {
		for px := 0; px < BlockWidth+6; px++ {
			srcX := sliderStartX + px
			srcY := sliderStartY + py

			if srcX >= 0 && srcX < CanvasWidth && srcY >= 0 && srcY < CanvasHeight {
				if px < 3 || px >= BlockWidth+3 || py < 3 || py >= BlockHeight+3 {
					maskColor := color.RGBA{
						R: uint8(randRange(160, 200)),
						G: uint8(randRange(160, 200)),
						B: uint8(randRange(160, 200)),
						A: 255,
					}
					slider.Set(px, py, maskColor)
				} else {
					slider.Set(px, py, bg.At(srcX, srcY))
				}
			}
		}
	}

	padding := 3
	blockTop := y
	blockBottom := y + BlockHeight
	blockLeft := x
	blockRight := x + BlockWidth

	for py := blockTop - padding; py < blockBottom+padding; py++ {
		for px := blockLeft - padding; px < blockRight+padding; px++ {
			if px >= 0 && px < CanvasWidth && py >= 0 && py < CanvasHeight {
				edgeColor := color.RGBA{
					R: uint8(randRange(100, 140)),
					G: uint8(randRange(100, 140)),
					B: uint8(randRange(100, 140)),
					A: 200,
				}

				isEdge := px == blockLeft-padding || px == blockRight+padding-1 ||
					py == blockTop-padding || py == blockBottom+padding-1

				if isEdge {
					bg.Set(px, py, edgeColor)
				} else {
					bg.Set(px, py, &image.Uniform{color.RGBA{
						R: uint8(randRange(230, 250)),
						G: uint8(randRange(230, 250)),
						B: uint8(randRange(230, 250)),
						A: 255,
					}})
				}
			}
		}
	}
}

func (g *SliderGenerator) drawCircle(img *image.RGBA, x, y, radius int, col color.RGBA) {
	for angle := 0.0; angle < 360; angle++ {
		rad := angle * math.Pi / 180
		px := int(float64(x) + float64(radius)*math.Cos(rad))
		py := int(float64(y) + float64(radius)*math.Sin(rad))
		if px >= 0 && px < CanvasWidth && py >= 0 && py < CanvasHeight {
			img.Set(px, py, col)
		}
	}
}

func (g *SliderGenerator) drawLine(img *image.RGBA, x1, y1, x2, y2, thickness int, col color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)

	var sx, sy int
	if x1 < x2 {
		sx = 1
	} else {
		sx = -1
	}
	if y1 < y2 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy

	for {
		for t := -thickness / 2; t <= thickness/2; t++ {
			if x1 >= 0 && x1 < CanvasWidth && y1+t >= 0 && y1+t < CanvasHeight {
				img.Set(x1, y1+t, col)
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

func (g *SliderGenerator) encodeToBase64(img image.Image) string {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func randRange(min, max int) int {
	return min + int(mrand.Intn(max-min))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
