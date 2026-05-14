package puzzle

import (
	"bytes"
	"captchax/config"
	"captchax/pkg/cache"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type Puzzle struct {
	cfg   *config.CaptchaConfig
	redis *cache.RedisClient
}

type PuzzleShape int

const (
	ShapeTriangle PuzzleShape = iota
	ShapeCircle
	ShapeWave
	ShapeDiamond
	ShapeSquare
)

type PuzzlePiece struct {
	Shape    PuzzleShape `json:"shape"`
	CenterX  int         `json:"center_x"`
	CenterY  int         `json:"center_y"`
	Size     int         `json:"size"`
	Angle    int         `json:"angle"`
}

type CaptchaData struct {
	ID           string       `json:"id"`
	TargetX      int          `json:"target_x"`
	TargetY      int          `json:"target_y"`
	TargetWidth  int          `json:"target_width"`
	TargetHeight int          `json:"target_height"`
	PieceShape   PuzzleShape  `json:"piece_shape"`
	PieceSize    int          `json:"piece_size"`
	CreatedAt    int64        `json:"created_at"`
}

type CaptchaResult struct {
	ID             string `json:"id"`
	BackgroundB64  string `json:"background_b64"`
	PuzzlePieceB64 string `json:"puzzle_piece_b64"`
	ShuffledB64    string `json:"shuffled_b64"`
	TargetX        int    `json:"target_x"`
	TargetY        int    `json:"target_y"`
	TargetWidth    int    `json:"target_width"`
	TargetHeight   int    `json:"target_height"`
	HintX          int    `json:"hint_x"`
	HintY          int    `json:"hint_y"`
}

func New(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *Puzzle {
	return &Puzzle{
		cfg:   cfg,
		redis: redisClient,
	}
}

func (p *Puzzle) GenerateCaptcha(ctx context.Context) (*CaptchaResult, error) {
	id := uuid.New().String()

	pieceSize := p.cfg.SliderSize
	if pieceSize == 0 {
		pieceSize = 60
	}
	halfSize := pieceSize / 2

	minX := halfSize + int(math.Floor(float64(p.cfg.Width)*0.1))
	maxX := p.cfg.Width - halfSize - int(math.Floor(float64(p.cfg.Width)*0.1))
	targetX := minX + rand.Intn(maxX-minX+1)

	minY := halfSize + int(math.Floor(float64(p.cfg.Height)*0.1))
	maxY := p.cfg.Height - halfSize - int(math.Floor(float64(p.cfg.Height)*0.3))
	targetY := minY + rand.Intn(maxY-minY+1)

	shape := PuzzleShape(rand.Intn(5))

	backgroundImg := p.generateBackground()
	puzzlePieceImg := p.generatePuzzlePiece(targetX, targetY, pieceSize, shape)
	shuffledImg, hintX, hintY := p.generateShuffled(backgroundImg, puzzlePieceImg, targetX, targetY, pieceSize)

	backgroundB64, err := p.imageToBase64(backgroundImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode background: %w", err)
	}

	pieceB64, err := p.imageToBase64(puzzlePieceImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode puzzle piece: %w", err)
	}

	shuffledB64, err := p.imageToBase64(shuffledImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode shuffled image: %w", err)
	}

	captchaData := CaptchaData{
		ID:           id,
		TargetX:      targetX,
		TargetY:      targetY,
		TargetWidth:  p.cfg.Width,
		TargetHeight: p.cfg.Height,
		PieceShape:   shape,
		PieceSize:    pieceSize,
		CreatedAt:    time.Now().Unix(),
	}

	dataBytes, err := json.Marshal(captchaData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal captcha data: %w", err)
	}

	key := fmt.Sprintf("captcha:puzzle:%s", id)
	expiration := time.Duration(p.cfg.ExpireMinutes) * time.Minute
	if err := p.redis.Set(ctx, key, dataBytes, expiration); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &CaptchaResult{
		ID:             id,
		BackgroundB64:  backgroundB64,
		PuzzlePieceB64: pieceB64,
		ShuffledB64:    shuffledB64,
		TargetX:        targetX,
		TargetY:        targetY,
		TargetWidth:    p.cfg.Width,
		TargetHeight:   p.cfg.Height,
		HintX:          hintX,
		HintY:          hintY,
	}, nil
}

func (p *Puzzle) generateBackground() image.Image {
	bg := image.NewRGBA(image.Rect(0, 0, p.cfg.Width, p.cfg.Height))

	bgColor := color.RGBA{
		R: uint8(180 + rand.Intn(40)),
		G: uint8(180 + rand.Intn(40)),
		B: uint8(180 + rand.Intn(40)),
		A: 255,
	}
	draw.Draw(bg, bg.Bounds(), &solidColor{bgColor.R, bgColor.G, bgColor.B, bgColor.A}, image.ZP, draw.Src)

	p.drawPattern(bg)
	p.drawTargetHint(bg)

	return bg
}

func (p *Puzzle) drawPattern(bg *image.RGBA) {
	for i := 0; i < 40; i++ {
		x := rand.Intn(p.cfg.Width)
		y := rand.Intn(p.cfg.Height)
		size := 3 + rand.Intn(12)

		patternColor := color.RGBA{
			R: uint8(rand.Intn(60)),
			G: uint8(rand.Intn(60)),
			B: uint8(rand.Intn(60)),
			A: uint8(20 + rand.Intn(40)),
		}

		switch rand.Intn(3) {
		case 0:
			p.drawCirclePattern(bg, x, y, size, patternColor)
		case 1:
			p.drawRectPattern(bg, x, y, size, size, patternColor)
		case 2:
			p.drawLinePattern(bg, x, y, x+size, y+size, patternColor)
		}
	}

	for i := 0; i < 8; i++ {
		x1 := rand.Intn(p.cfg.Width)
		y1 := rand.Intn(p.cfg.Height)
		x2 := rand.Intn(p.cfg.Width)
		y2 := rand.Intn(p.cfg.Height)

		patternColor := color.RGBA{
			R: uint8(150 + rand.Intn(50)),
			G: uint8(150 + rand.Intn(50)),
			B: uint8(150 + rand.Intn(50)),
			A: uint8(15 + rand.Intn(25)),
		}
		p.drawLinePattern(bg, x1, y1, x2, y2, patternColor)
	}
}

func (p *Puzzle) drawTargetHint(bg *image.RGBA) {
	pieceSize := p.cfg.SliderSize
	if pieceSize == 0 {
		pieceSize = 60
	}
	halfSize := pieceSize / 2

	hintColor := color.RGBA{
		R: 160,
		G: 160,
		B: 165,
		A: 200,
	}

	p.drawShapeOutline(bg, halfSize, halfSize, pieceSize, ShapeSquare, hintColor)
}

func (p *Puzzle) drawShapeOutline(bg *image.RGBA, cx, cy, size int, shape PuzzleShape, col color.RGBA) {
	switch shape {
	case ShapeTriangle:
		p.drawTriangleOutline(bg, cx, cy, size, col)
	case ShapeCircle:
		p.drawCirclePattern(bg, cx, cy, size/2, col)
	case ShapeWave:
		p.drawWaveOutline(bg, cx, cy, size, col)
	case ShapeDiamond:
		p.drawDiamondOutline(bg, cx, cy, size, col)
	default:
		p.drawRectPattern(bg, cx-size/2, cy-size/2, size, size, col)
	}
}

func (p *Puzzle) drawTriangleOutline(bg *image.RGBA, cx, cy, size int, col color.RGBA) {
	halfSize := size / 2
	points := []struct{ x, y int }{
		{cx, cy - halfSize},
		{cx - halfSize, cy + halfSize},
		{cx + halfSize, cy + halfSize},
	}

	p.drawLinePattern(bg, points[0].x, points[0].y, points[1].x, points[1].y, col)
	p.drawLinePattern(bg, points[1].x, points[1].y, points[2].x, points[2].y, col)
	p.drawLinePattern(bg, points[2].x, points[2].y, points[0].x, points[0].y, col)
}

func (p *Puzzle) drawWaveOutline(bg *image.RGBA, cx, cy, size int, col color.RGBA) {
	halfSize := size / 2
	amplitude := size / 6

	for x := cx - halfSize; x <= cx+halfSize; x++ {
		phase := float64(x-(cx-halfSize)) / float64(size) * 2 * math.Pi
		y := cy + int(float64(amplitude)*math.Sin(phase))
		p.drawPixelSafe(bg, x, y, col)
	}

	for x := cx - halfSize; x <= cx+halfSize; x++ {
		phase := float64(x-(cx-halfSize)) / float64(size) * 2 * math.Pi
		y := cy - int(float64(amplitude) * math.Sin(phase))
		p.drawPixelSafe(bg, x, y, col)
	}

	p.drawLinePattern(bg, cx-halfSize, cy-halfSize, cx-halfSize, cy+halfSize, col)
	p.drawLinePattern(bg, cx+halfSize, cy-halfSize, cx+halfSize, cy+halfSize, col)
}

func (p *Puzzle) drawDiamondOutline(bg *image.RGBA, cx, cy, size int, col color.RGBA) {
	halfSize := size / 2
	points := []struct{ x, y int }{
		{cx, cy - halfSize},
		{cx + halfSize, cy},
		{cx, cy + halfSize},
		{cx - halfSize, cy},
	}

	p.drawLinePattern(bg, points[0].x, points[0].y, points[1].x, points[1].y, col)
	p.drawLinePattern(bg, points[1].x, points[1].y, points[2].x, points[2].y, col)
	p.drawLinePattern(bg, points[2].x, points[2].y, points[3].x, points[3].y, col)
	p.drawLinePattern(bg, points[3].x, points[3].y, points[0].x, points[0].y, col)
}

func (p *Puzzle) generatePuzzlePiece(targetX, targetY, size int, shape PuzzleShape) image.Image {
	pieceImg := image.NewRGBA(image.Rect(0, 0, size*2, size*2))

	fillColor := color.RGBA{
		R: uint8(150 + rand.Intn(40)),
		G: uint8(150 + rand.Intn(40)),
		B: uint8(150 + rand.Intn(40)),
		A: 255,
	}
	draw.Draw(pieceImg, pieceImg.Bounds(), &solidColor{fillColor.R, fillColor.G, fillColor.B, fillColor.A}, image.ZP, draw.Src)

	cx := size
	cy := size

	switch shape {
	case ShapeTriangle:
		p.fillTriangle(pieceImg, cx, cy, size/2, fillColor)
	case ShapeCircle:
		p.fillCircle(pieceImg, cx, cy, size/2, fillColor)
	case ShapeWave:
		p.fillWave(pieceImg, cx, cy, size/2, fillColor)
	case ShapeDiamond:
		p.fillDiamond(pieceImg, cx, cy, size/2, fillColor)
	default:
		p.fillSquare(pieceImg, cx-size/2, cy-size/2, size, size, fillColor)
	}

	shadowColor := color.RGBA{
		R: 50,
		G: 50,
		B: 50,
		A: 100,
	}
	for i := 0; i < 3; i++ {
		switch shape {
		case ShapeTriangle:
			p.drawTriangleOutline(pieceImg, cx+i, cy+i, size/2-i, shadowColor)
		case ShapeCircle:
			p.drawCirclePattern(pieceImg, cx+i, cy+i, size/2-i, shadowColor)
		case ShapeWave:
			p.drawWaveOutline(pieceImg, cx+i, cy+i, size/2-i, shadowColor)
		case ShapeDiamond:
			p.drawDiamondOutline(pieceImg, cx+i, cy+i, size/2-i, shadowColor)
		default:
			p.drawRectPattern(pieceImg, cx-size/2+i, cy-size/2+i, size-i*2, size-i*2, shadowColor)
		}
	}

	return pieceImg
}

func (p *Puzzle) fillTriangle(piece *image.RGBA, cx, cy, size int, col color.RGBA) {
	halfSize := size
	for y := -halfSize; y <= halfSize; y++ {
		rowWidth := int(float64(halfSize-y) * 1.2)
		for x := -rowWidth; x <= rowWidth; x++ {
			if int(math.Sqrt(float64(x*x+y*y))) <= halfSize {
				piece.Set(cx+x, cy+y, col)
			}
		}
	}
}

func (p *Puzzle) fillCircle(piece *image.RGBA, cx, cy, radius int, col color.RGBA) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				piece.Set(cx+x, cy+y, col)
			}
		}
	}
}

func (p *Puzzle) fillWave(piece *image.RGBA, cx, cy, size int, col color.RGBA) {
	halfSize := size
	amplitude := size / 3

	for y := -halfSize; y <= halfSize; y++ {
		for x := -halfSize; x <= halfSize; x++ {
			waveY := int(float64(amplitude) * math.Sin(float64(x)/float64(halfSize)*math.Pi))
			if math.Abs(float64(y-waveY)) <= float64(halfSize)/2 {
				piece.Set(cx+x, cy+y, col)
			}
		}
	}
}

func (p *Puzzle) fillDiamond(piece *image.RGBA, cx, cy, size int, col color.RGBA) {
	halfSize := size
	for y := -halfSize; y <= halfSize; y++ {
		rowWidth := halfSize - int(math.Abs(float64(y)))
		for x := -rowWidth; x <= rowWidth; x++ {
			piece.Set(cx+x, cy+y, col)
		}
	}
}

func (p *Puzzle) fillSquare(piece *image.RGBA, x, y, w, h int, col color.RGBA) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			piece.Set(i, j, col)
		}
	}
}

func (p *Puzzle) generateShuffled(background, piece image.Image, targetX, targetY, pieceSize int) (image.Image, int, int) {
	shuffled := image.NewRGBA(image.Rect(0, 0, p.cfg.Width, p.cfg.Height))
	draw.Draw(shuffled, shuffled.Bounds(), background, image.ZP, draw.Src)

	shuffleX := rand.Intn(p.cfg.Width - pieceSize)
	shuffleY := rand.Intn(p.cfg.Height - pieceSize)

	if absInt(shuffleX-targetX) < pieceSize/2 {
		if targetX > p.cfg.Width/2 {
			shuffleX = 10
		} else {
			shuffleX = p.cfg.Width - pieceSize - 10
		}
	}
	if absInt(shuffleY-targetY) < pieceSize/2 {
		shuffleY = rand.Intn(p.cfg.Height - pieceSize)
	}

	cx := pieceSize
	cy := pieceSize
	offsetX := cx - pieceSize/2
	offsetY := cy - pieceSize/2

	draw.Draw(shuffled, image.Rect(shuffleX, shuffleY, shuffleX+pieceSize, shuffleY+pieceSize),
		piece, image.Pt(offsetX, offsetY), draw.Over)

	borderColor := color.RGBA{R: 200, G: 200, B: 200, A: 180}
	p.drawRectPattern(shuffled, shuffleX, shuffleY, pieceSize, 2, borderColor)
	p.drawRectPattern(shuffled, shuffleX, shuffleY+pieceSize-2, pieceSize, 2, borderColor)
	p.drawRectPattern(shuffled, shuffleX, shuffleY, 2, pieceSize, borderColor)
	p.drawRectPattern(shuffled, shuffleX+pieceSize-2, shuffleY, 2, pieceSize, borderColor)

	return shuffled, shuffleX, shuffleY
}

func (p *Puzzle) drawCirclePattern(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	x, y, d := 0, radius, 3-2*radius
	for x <= y {
		p.drawPixelSafe(img, cx+x, cy+y, col)
		p.drawPixelSafe(img, cx+y, cy+x, col)
		p.drawPixelSafe(img, cx-y, cy+x, col)
		p.drawPixelSafe(img, cx-x, cy+y, col)
		p.drawPixelSafe(img, cx+x, cy-y, col)
		p.drawPixelSafe(img, cx+y, cy-x, col)
		p.drawPixelSafe(img, cx-y, cy-x, col)
		p.drawPixelSafe(img, cx-x, cy-y, col)
		if d < 0 {
			d = d + 4*x + 6
		} else {
			d = d + 4*(x-y) + 10
			y--
		}
		x++
	}
}

func (p *Puzzle) drawRectPattern(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	for i := x; i < x+w && i < img.Bounds().Dx(); i++ {
		for j := y; j < y+h && j < img.Bounds().Dy(); j++ {
			if i >= 0 && j >= 0 {
				img.Set(i, j, col)
			}
		}
	}
}

func (p *Puzzle) drawLinePattern(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA) {
	dx := absInt(x2 - x1)
	dy := absInt(y2 - y1)
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
		p.drawPixelSafe(img, x1, y1, col)
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

func (p *Puzzle) drawPixelSafe(img *image.RGBA, x, y int, col color.RGBA) {
	if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
		img.Set(x, y, col)
	}
}

func (p *Puzzle) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

type solidColor struct {
	r, g, b, a uint8
}

func (sc *solidColor) RGBA() (r, g, b, a uint32) {
	return uint32(sc.r) * 0x101, uint32(sc.g) * 0x101, uint32(sc.b) * 0x101, uint32(sc.a) * 0x101
}

func (sc *solidColor) ColorModel() color.Model {
	return color.RGBAModel
}

func (sc *solidColor) Bounds() image.Rectangle {
	return image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 0xffffffff, Y: 0xffffffff}}
}

func (sc *solidColor) At(x, y int) color.Color {
	return sc
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
