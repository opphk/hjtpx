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
	"math/rand"
	"time"
)

type SliderGenerator struct {
	width  int
	height int
}

type SliderData struct {
	BackgroundImage string `json:"background_image"`
	SliderImage    string `json:"slider_image"`
	SliderX        int    `json:"slider_x"`
	SliderY        int    `json:"slider_y"`
	Pieces         int    `json:"pieces"`
	Difficulty     string `json:"difficulty"`
}

type SliderSolution struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Angle  float64 `json:"angle"`
	Pieces []Piece `json:"pieces"`
}

type Piece struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func NewSliderGenerator() *SliderGenerator {
	return &SliderGenerator{
		width:  320,
		height: 160,
	}
}

func (g *SliderGenerator) Generate(difficulty string) (*SliderData, *SliderSolution, error) {
	diff := g.getDifficultyParams(difficulty)

	pieceCount := diff.pieceCount
	sliderX := g.width/4 + randInt(0, g.width/2-pieceCount)
	sliderY := g.height/4 + randInt(0, g.height/2)

	bgImage, err := g.generateBackground(sliderX, sliderY, pieceCount)
	if err != nil {
		return nil, nil, err
	}

	sliderImage, err := g.generateSlider(sliderX, sliderY, pieceCount)
	if err != nil {
		return nil, nil, err
	}

	data := &SliderData{
		BackgroundImage: base64.StdEncoding.EncodeToString(bgImage),
		SliderImage:    base64.StdEncoding.EncodeToString(sliderImage),
		SliderX:        sliderX,
		SliderY:        sliderY,
		Pieces:         pieceCount,
		Difficulty:     difficulty,
	}

	solution := &SliderSolution{
		X:      sliderX,
		Y:      sliderY,
		Angle:  0,
		Pieces: []Piece{{X: sliderX, Y: sliderY, Width: pieceCount * 10, Height: 40}},
	}

	return data, solution, nil
}

func (g *SliderGenerator) getDifficultyParams(difficulty string) struct{ pieceCount int } {
	switch difficulty {
	case "easy":
		return struct{ pieceCount int }{pieceCount: 3}
	case "hard":
		return struct{ pieceCount int }{pieceCount: 6}
	default:
		return struct{ pieceCount int }{pieceCount: 4}
	}
}

func (g *SliderGenerator) generateBackground(sliderX, sliderY, pieces int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, g.width, g.height))

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			img.Set(x, y, g.getRandomColor())
		}
	}

	g.drawSliderPiece(img, sliderX, sliderY, pieces)

	for i := 0; i < 10+pieces*2; i++ {
		x1 := randInt(0, g.width)
		y1 := randInt(0, g.height)
		g.drawLine(img, x1, y1, x1+randInt(-50, 50), y1+randInt(-20, 20), g.getRandomColor())
	}

	for i := 0; i < 3; i++ {
		g.drawCircle(img, randInt(0, g.width), randInt(0, g.height), randInt(10, 30), g.getRandomColor())
	}

	return g.encodeImage(img)
}

func (g *SliderGenerator) generateSlider(x, y, pieces int) ([]byte, error) {
	pieceWidth := pieces * 10
	pieceHeight := 40

	img := image.NewRGBA(image.Rect(0, 0, pieceWidth, pieceHeight+20))

	for py := 0; py < pieceHeight+20; py++ {
		for px := 0; px < pieceWidth; px++ {
			img.Set(px, py, g.getRandomColor())
		}
	}

	g.drawSliderPiece(img, pieceWidth/2, pieceHeight/2, pieces)

	return g.encodeImage(img)
}

func (g *SliderGenerator) drawSliderPiece(img *image.RGBA, x, y, pieces int) {
	halfWidth := pieces * 5
	halfHeight := 20

	leftX := x - halfWidth
	rightX := x + halfWidth
	topY := y - halfHeight
	bottomY := y + halfHeight

	g.drawLine(img, leftX, topY, leftX+10, topY+10, color.White)
	g.drawLine(img, rightX-10, topY+10, rightX, topY, color.White)
	g.drawLine(img, leftX, bottomY, leftX+10, bottomY-10, color.White)
	g.drawLine(img, rightX-10, bottomY-10, rightX, bottomY, color.White)
}

func (g *SliderGenerator) drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
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
		if x1 >= 0 && x1 < g.width && y1 >= 0 && y1 < g.height {
			img.Set(x1, y1, col)
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

func (g *SliderGenerator) drawCircle(img *image.RGBA, x, y, r int, col color.Color) {
	for angle := 0; angle < 360; angle++ {
		rad := float64(angle) * math.Pi / 180
		px := x + int(float64(r)*math.Cos(rad))
		py := y + int(float64(r)*math.Sin(rad))
		if px >= 0 && px < g.width && py >= 0 && py < g.height {
			img.Set(px, py, col)
		}
	}
}

func (g *SliderGenerator) getRandomColor() color.Color {
	return color.RGBA{
		R: uint8(randInt(200, 255)),
		G: uint8(randInt(200, 255)),
		B: uint8(randInt(200, 255)),
		A: 255,
	}
}

func (g *SliderGenerator) encodeImage(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	return buf.Bytes(), nil
}

type SliderVerifier struct {
	toleranceX int
	toleranceY int
}

func NewSliderVerifier() *SliderVerifier {
	return &SliderVerifier{
		toleranceX: 5,
		toleranceY: 10,
	}
}

func (v *SliderVerifier) Verify(solution *SliderSolution, userAnswer json.RawMessage) (bool, float64, error) {
	var answer struct {
		X            int     `json:"x"`
		Y            int     `json:"y"`
		Trajectory   []Point `json:"trajectory"`
		ResponseTime int     `json:"response_time_ms"`
	}

	if err := json.Unmarshal(userAnswer, &answer); err != nil {
		return false, 0, fmt.Errorf("failed to unmarshal answer: %w", err)
	}

	distanceX := math.Abs(float64(solution.X - answer.X))
	distanceY := math.Abs(float64(solution.Y - answer.Y))

	score := 1.0
	if distanceX > float64(v.toleranceX) {
		score -= (distanceX - float64(v.toleranceX)) / float64(solution.X)
		if score < 0 {
			score = 0
		}
	}

	if answer.ResponseTime < 500 {
		score *= 0.3
	} else if answer.ResponseTime > 30000 {
		score *= 0.7
	}

	if len(answer.Trajectory) > 5 {
		trajectoryScore := v.analyzeTrajectory(answer.Trajectory)
		score *= trajectoryScore
	}

	isValid := distanceX <= float64(v.toleranceX) && distanceY <= float64(v.toleranceY) && score >= 0.5

	return isValid, score, nil
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
	T int `json:"t"`
}

func (v *SliderVerifier) analyzeTrajectory(trajectory []Point) float64 {
	if len(trajectory) < 2 {
		return 0.3
	}

	avgSpeed := 0.0
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		dt := float64(trajectory[i].T - trajectory[i-1].T)
		if dt > 0 {
			speed := math.Sqrt(dx*dx + dy*dy) / dt
			avgSpeed += speed
		}
	}
	avgSpeed /= float64(len(trajectory) - 1)

	if avgSpeed < 0.1 || avgSpeed > 5.0 {
		return 0.5
	}

	directionChanges := 0
	for i := 2; i < len(trajectory); i++ {
		prevDir := trajectory[i-1].X - trajectory[i-2].X
		currDir := trajectory[i].X - trajectory[i-1].X
		if prevDir*currDir < 0 {
			directionChanges++
		}
	}

	if directionChanges > len(trajectory)/3 {
		return 1.0
	}

	return 0.8
}

func randInt(min, max int) int {
	if max <= min {
		return min
	}
	n := max - min
	r := rand.Intn(n)
	return min + r
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
