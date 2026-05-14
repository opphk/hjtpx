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

type ClickGenerator struct {
	width  int
	height int
}

type ClickData struct {
	BackgroundImage string    `json:"background_image"`
	TargetText      string    `json:"target_text"`
	Distractors     []string  `json:"distractors"`
	TargetPosition  Position  `json:"target_position"`
	Hotspots        []Hotspot `json:"hotspots"`
	Difficulty      string    `json:"difficulty"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Hotspot struct {
	X int `json:"x"`
	Y int `json:"y"`
	R int `json:"r"`
}

type ClickSolution struct {
	TargetIndex int      `json:"target_index"`
	TargetPos   Position `json:"target_pos"`
	ClickOrder  []int    `json:"click_order"`
	TimeLimit   int      `json:"time_limit_ms"`
}

func NewClickGenerator() *ClickGenerator {
	return &ClickGenerator{
		width:  400,
		height: 300,
	}
}

func (g *ClickGenerator) Generate(difficulty string) (*ClickData, *ClickSolution, error) {
	diff := g.getDifficultyParams(difficulty)

	chineseChars := []string{"大", "中", "小", "上", "下", "左", "右", "红", "蓝", "绿", "黄", "白", "黑"}
	targetIdx := randInt(0, len(chineseChars))
	targetChar := chineseChars[targetIdx]

	targetX := diff.padding + randInt(0, g.width-2*diff.padding-diff.charSize)
	targetY := diff.padding + randInt(0, g.height-2*diff.padding-diff.charSize)

	distractors := make([]string, 0)
	for i := 0; i < diff.numDistractors; i++ {
		distIdx := randInt(0, len(chineseChars))
		if distIdx != targetIdx {
			distractors = append(distractors, chineseChars[distIdx])
		}
	}

	hotspots := make([]Hotspot, 0)
	hotspots = append(hotspots, Hotspot{
		X: targetX + diff.charSize/2,
		Y: targetY + diff.charSize/2,
		R: diff.charSize,
	})

	for i := 0; i < diff.numDistractors; i++ {
		hx := diff.padding + randInt(0, g.width-2*diff.padding-diff.charSize)
		hy := diff.padding + randInt(0, g.height-2*diff.padding-diff.charSize)
		hotspots = append(hotspots, Hotspot{
			X: hx + diff.charSize/2,
			Y: hy + diff.charSize/2,
			R: diff.charSize,
		})
	}

	bgImage, err := g.generateBackground()
	if err != nil {
		return nil, nil, err
	}

	markedImage, err := g.markTextOnImage(bgImage, targetChar, targetX, targetY, targetIdx, distractors)
	if err != nil {
		return nil, nil, err
	}

	data := &ClickData{
		BackgroundImage: base64.StdEncoding.EncodeToString(markedImage),
		TargetText:      targetChar,
		Distractors:    distractors,
		TargetPosition:  Position{X: targetX, Y: targetY, W: diff.charSize, H: diff.charSize},
		Hotspots:        hotspots,
		Difficulty:      difficulty,
	}

	solution := &ClickSolution{
		TargetIndex: targetIdx,
		TargetPos:   Position{X: targetX, Y: targetY, W: diff.charSize, H: diff.charSize},
		ClickOrder:  []int{0},
		TimeLimit:   diff.timeLimit,
	}

	return data, solution, nil
}

type clickDifficulty struct {
	padding        int
	charSize       int
	numDistractors int
	timeLimit      int
}

func (g *ClickGenerator) getDifficultyParams(difficulty string) clickDifficulty {
	switch difficulty {
	case "easy":
		return clickDifficulty{padding: 50, charSize: 40, numDistractors: 2, timeLimit: 10000}
	case "hard":
		return clickDifficulty{padding: 30, charSize: 30, numDistractors: 5, timeLimit: 5000}
	default:
		return clickDifficulty{padding: 40, charSize: 35, numDistractors: 3, timeLimit: 8000}
	}
}

func (g *ClickGenerator) generateBackground() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, g.width, g.height))

	bgColor := color.RGBA{
		R: uint8(randInt(240, 250)),
		G: uint8(randInt(240, 250)),
		B: uint8(randInt(240, 250)),
		A: 255,
	}

	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	for i := 0; i < 20; i++ {
		x := randInt(0, g.width)
		y := randInt(0, g.height)
		r := randInt(5, 15)
		g.drawCircle(img, x, y, r, color.Gray{Y: uint8(randInt(180, 220))})
	}

	for i := 0; i < 5; i++ {
		x1 := randInt(0, g.width)
		y1 := randInt(0, g.height)
		x2 := randInt(0, g.width)
		y2 := randInt(0, g.height)
		g.drawLine(img, x1, y1, x2, y2, color.Gray{Y: uint8(randInt(180, 220))})
	}

	return g.encodeImage(img)
}

func (g *ClickGenerator) markTextOnImage(bgImage []byte, target string, tx, ty, targetIdx int, distractors []string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, g.width, g.height))

	for i := 0; i < len(bgImage); i += 4 {
		if i+3 < len(bgImage) {
			img.SetRGBA(i/4%g.width, i/4/g.width, color.RGBA{
				R: bgImage[i],
				G: bgImage[i+1],
				B: bgImage[i+2],
				A: bgImage[i+3],
			})
		}
	}

	colors := []color.RGBA{
		{255, 0, 0, 255},
		{0, 128, 255, 255},
		{0, 200, 0, 255},
		{255, 165, 0, 255},
	}

	charSize := 35
	positions := make([]Position, 0)
	positions = append(positions, Position{X: tx, Y: ty, W: charSize, H: charSize})

	for i := 0; i < len(distractors); i++ {
		px := 40 + randInt(0, g.width-80-charSize)
		py := 40 + randInt(0, g.height-80-charSize)
		positions = append(positions, Position{X: px, Y: py, W: charSize, H: charSize})
	}

	for i, pos := range positions {
		c := colors[i%len(colors)]
		if i == 0 {
			c = color.RGBA{255, 0, 0, 255}
		}
		g.drawRect(img, pos.X-2, pos.Y-2, pos.X+pos.W+2, pos.Y+pos.H+2, c)
	}

	return g.encodeImage(img)
}

func (g *ClickGenerator) drawRect(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	for x := x1; x < x2 && x < g.width; x++ {
		if y1 >= 0 && y1 < g.height {
			img.Set(x, y1, col)
		}
		if y2 >= 0 && y2 < g.height {
			img.Set(x, y2, col)
		}
	}
	for y := y1; y < y2 && y < g.height; y++ {
		if x1 >= 0 && x1 < g.width {
			img.Set(x1, y, col)
		}
		if x2 >= 0 && x2 < g.width {
			img.Set(x2, y, col)
		}
	}
}

func (g *ClickGenerator) drawCircle(img *image.RGBA, x, y, r int, col color.Color) {
	for angle := 0; angle < 360; angle++ {
		rad := float64(angle) * math.Pi / 180
		px := x + int(float64(r)*math.Cos(rad))
		py := y + int(float64(r)*math.Sin(rad))
		if px >= 0 && px < g.width && py >= 0 && py < g.height {
			img.Set(px, py, col)
		}
	}
}

func (g *ClickGenerator) drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
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

func (g *ClickGenerator) encodeImage(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	return buf.Bytes(), nil
}

type ClickVerifier struct {
	toleranceRadius int
}

func NewClickVerifier() *ClickVerifier {
	return &ClickVerifier{toleranceRadius: 20}
}

type ClickAnswer struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	ClickTime  int64   `json:"click_time_ms"`
	Trajectory []Point `json:"trajectory"`
}

func (v *ClickVerifier) Verify(solution *ClickSolution, userAnswer json.RawMessage) (bool, float64, error) {
	var answers []ClickAnswer
	if err := json.Unmarshal(userAnswer, &answers); err != nil {
		return false, 0, fmt.Errorf("failed to unmarshal answer: %w", err)
	}

	if len(answers) == 0 {
		return false, 0.0, nil
	}

	score := 1.0

	targetX := solution.TargetPos.X + solution.TargetPos.W/2
	targetY := solution.TargetPos.Y + solution.TargetPos.H/2

	firstClick := answers[0]
	distance := math.Sqrt(math.Pow(float64(firstClick.X-targetX), 2) + math.Pow(float64(firstClick.Y-targetY), 2))

	if distance > float64(v.toleranceRadius) {
		return false, 0.0, nil
	}

	if firstClick.ClickTime < 300 {
		score *= 0.3
	} else if firstClick.ClickTime > int64(solution.TimeLimit) {
		score *= 0.5
	}

	if len(answers) > 1 {
		trajectoryScore := v.analyzeTrajectory(answers)
		score *= trajectoryScore
	}

	isValid := distance <= float64(v.toleranceRadius) && score >= 0.5

	return isValid, score, nil
}

func (v *ClickVerifier) analyzeTrajectory(answers []ClickAnswer) float64 {
	if len(answers) < 2 {
		return 0.8
	}

	totalDistance := 0.0
	for i := 1; i < len(answers); i++ {
		dx := float64(answers[i].X - answers[i-1].X)
		dy := float64(answers[i].Y - answers[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	directDistance := math.Sqrt(math.Pow(float64(answers[len(answers)-1].X-answers[0].X), 2) +
		math.Pow(float64(answers[len(answers)-1].Y-answers[0].Y), 2))

	if directDistance > 0 {
		ratio := totalDistance / directDistance
		if ratio > 5.0 {
			return 0.6
		}
		if ratio > 3.0 {
			return 0.8
		}
	}

	return 1.0
}
