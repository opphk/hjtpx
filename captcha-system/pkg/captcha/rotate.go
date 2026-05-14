package captcha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
)

type RotateGenerator struct {
	size int
}

type RotateData struct {
	BackgroundImage string `json:"background_image"`
	RotatedImage    string `json:"rotated_image"`
	TargetAngle     int    `json:"target_angle"`
	Difficulty      string `json:"difficulty"`
}

type RotateSolution struct {
	TargetAngle int `json:"target_angle"`
	Tolerance   int `json:"tolerance"`
}

func NewRotateGenerator() *RotateGenerator {
	return &RotateGenerator{size: 200}
}

func (g *RotateGenerator) Generate(difficulty string) (*RotateData, *RotateSolution, error) {
	diff := g.getDifficultyParams(difficulty)

	targetAngle := randInt(-180+diff.rangeDegrees, -diff.rangeDegrees)
	if randInt(0, 2) == 1 {
		targetAngle = -targetAngle
	}

	targetAngle = (targetAngle + 360) % 360

	bgImage, err := g.generateBackground()
	if err != nil {
		return nil, nil, err
	}

	rotatedImage, err := g.rotateImage(bgImage, targetAngle)
	if err != nil {
		return nil, nil, err
	}

	data := &RotateData{
		BackgroundImage: base64.StdEncoding.EncodeToString(bgImage),
		RotatedImage:    base64.StdEncoding.EncodeToString(rotatedImage),
		TargetAngle:     targetAngle,
		Difficulty:      difficulty,
	}

	solution := &RotateSolution{
		TargetAngle: targetAngle,
		Tolerance:   diff.tolerance,
	}

	return data, solution, nil
}

type rotateDifficulty struct {
	tolerance          int
	rangeDegrees       int
	patternComplexity  int
}

func (g *RotateGenerator) getDifficultyParams(difficulty string) rotateDifficulty {
	switch difficulty {
	case "easy":
		return rotateDifficulty{tolerance: 15, rangeDegrees: 90, patternComplexity: 1}
	case "hard":
		return rotateDifficulty{tolerance: 5, rangeDegrees: 150, patternComplexity: 3}
	default:
		return rotateDifficulty{tolerance: 10, rangeDegrees: 120, patternComplexity: 2}
	}
}

func (g *RotateGenerator) generateBackground() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, g.size, g.size))

	bgColor := color.RGBA{
		R: uint8(randInt(240, 255)),
		G: uint8(randInt(240, 255)),
		B: uint8(randInt(240, 255)),
		A: 255,
	}

	for y := 0; y < g.size; y++ {
		for x := 0; x < g.size; x++ {
			img.Set(x, y, bgColor)
		}
	}

	centerX := g.size / 2
	centerY := g.size / 2

	colors := []color.RGBA{
		{255, 100, 100, 255},
		{100, 150, 255, 255},
		{100, 200, 100, 255},
		{255, 200, 100, 255},
		{200, 100, 255, 255},
	}

	for i := 0; i < 8; i++ {
		angle := float64(i) * 45.0 * math.Pi / 180.0
		innerR := 30
		outerR := 70

		for r := innerR; r < outerR; r++ {
			px := centerX + int(float64(r)*math.Cos(angle))
			py := centerY + int(float64(r)*math.Sin(angle))
			if px >= 0 && px < g.size && py >= 0 && py < g.size {
				img.Set(px, py, colors[i%len(colors)])
			}
		}
	}

	for i := 0; i < 12; i++ {
		angle := float64(i) * 30.0 * math.Pi / 180.0
		for r := 75; r < 95; r++ {
			px := centerX + int(float64(r)*math.Cos(angle))
			py := centerY + int(float64(r)*math.Sin(angle))
			if px >= 0 && px < g.size && py >= 0 && py < g.size {
				img.Set(px, py, color.Black)
			}
		}
	}

	for i := 0; i < 4; i++ {
		x := centerX - 20 + randInt(0, 40)
		y := centerY - 20 + randInt(0, 40)
		g.drawSmallShape(img, x, y, colors[i%len(colors)])
	}

	return g.encodeImage(img)
}

func (g *RotateGenerator) drawSmallShape(img *image.RGBA, x, y int, col color.Color) {
	for dy := -5; dy <= 5; dy++ {
		for dx := -5; dx <= 5; dx++ {
			if dx*dx+dy*dy <= 25 {
				px := x + dx
				py := y + dy
				if px >= 0 && px < g.size && py >= 0 && py < g.size {
					img.Set(px, py, col)
				}
			}
		}
	}
}

func (g *RotateGenerator) rotateImage(src []byte, angle int) ([]byte, error) {
	srcImg := image.NewRGBA(image.Rect(0, 0, g.size, g.size))
	for i := 0; i < len(src); i += 4 {
		if i+3 < len(src) {
			x := (i / 4) % g.size
			y := (i / 4) / g.size
			srcImg.SetRGBA(x, y, color.RGBA{
				R: src[i],
				G: src[i+1],
				B: src[i+2],
				A: src[i+3],
			})
		}
	}

	angleRad := float64(angle) * math.Pi / 180.0
	cosA := math.Cos(angleRad)
	sinA := math.Sin(angleRad)

	dstImg := image.NewRGBA(image.Rect(0, 0, g.size, g.size))

	centerX := float64(g.size) / 2
	centerY := float64(g.size) / 2

	bgColor := color.RGBA{200, 200, 200, 255}
	for y := 0; y < g.size; y++ {
		for x := 0; x < g.size; x++ {
			dstImg.Set(x, y, bgColor)
		}
	}

	for y := 0; y < g.size; y++ {
		for x := 0; x < g.size; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY

			srcX := int(dx*cosA - dy*sinA + centerX)
			srcY := int(dx*sinA + dy*cosA + centerY)

			if srcX >= 0 && srcX < g.size && srcY >= 0 && srcY < g.size {
				dstImg.Set(x, y, srcImg.At(srcX, srcY))
			}
		}
	}

	return g.encodeImage(dstImg)
}

func (g *RotateGenerator) encodeImage(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	return buf.Bytes(), nil
}

type RotateVerifier struct {
	defaultTolerance int
}

func NewRotateVerifier() *RotateVerifier {
	return &RotateVerifier{defaultTolerance: 10}
}

type RotateAnswer struct {
	Angle        int `json:"angle"`
	ResponseTime int `json:"response_time_ms"`
}

func (v *RotateVerifier) Verify(solution *RotateSolution, userAnswer json.RawMessage) (bool, float64, error) {
	var answer RotateAnswer
	if err := json.Unmarshal(userAnswer, &answer); err != nil {
		return false, 0, fmt.Errorf("failed to unmarshal answer: %w", err)
	}

	tolerance := solution.Tolerance
	if tolerance == 0 {
		tolerance = v.defaultTolerance
	}

	diff := v.calculateAngleDiff(solution.TargetAngle, answer.Angle)

	score := 1.0 - float64(diff)/180.0

	if answer.ResponseTime < 500 {
		score *= 0.3
	} else if answer.ResponseTime > 15000 {
		score *= 0.7
	}

	isValid := diff <= tolerance && score >= 0.5

	return isValid, score, nil
}

func (v *RotateVerifier) calculateAngleDiff(target, actual int) int {
	diff := intAbs(target - actual)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}

func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
