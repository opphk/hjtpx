package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type CaptchaType string

const (
	CaptchaTypeNumber CaptchaType = "number"
	CaptchaTypeLetter CaptchaType = "letter"
	CaptchaTypeMixed  CaptchaType = "mixed"
)

type DifficultyLevel int

const (
	Easy DifficultyLevel = iota + 1
	Medium
	Hard
	Expert
)

type CharSetType int

const (
	Numeric CharSetType = iota + 1
	Alphabetic
	Alphanumeric
	Chinese
)

type ImageConfig struct {
	Length     int
	Width      int
	Height     int
	Difficulty DifficultyLevel
	CharSet    CharSetType
}

var (
	numericCharSet      = "0123456789"
	alphabeticCharSet   = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
	alphanumericCharSet = numericCharSet + alphabeticCharSet
	chineseCharSet      = "的一是在不了有和人这中大为上个国我以要他时来"
	fullCharSet         = numericCharSet + alphabeticCharSet
)

type GenerateImageCaptchaRequest struct {
	Type       CaptchaType `form:"type" json:"type"`
	Count      int         `form:"count" json:"count"`
	CustomSet  string      `form:"custom_set" json:"custom_set"`
	NoiseMode  int         `form:"noise_mode" json:"noise_mode"`
	LineMode   int         `form:"line_mode" json:"line_mode"`
}

type GenerateImageCaptchaResponse struct {
	ChallengeID string `json:"challenge_id"`
	Image       string `json:"image"`
}

type VerifyImageCaptchaRequest struct {
	ChallengeID string `json:"challenge_id" binding:"required"`
	Answer      string `json:"answer" binding:"required"`
}

type VerifyImageCaptchaResponse struct {
	Success bool `json:"success"`
}

const (
	captchaWidth  = 140
	captchaHeight = 50
	captchaTTL    = 5 * time.Minute
)

var (
	digitCharSet    = "0123456789"
	letterCharSet   = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
	allCharSet      = digitCharSet + letterCharSet
	r               *rand.Rand
	rMu             sync.Mutex
)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GenerateImageCaptcha(c *gin.Context) {
	var req GenerateImageCaptchaRequest
	if err := c.ShouldBind(&req); err != nil {
		req.Type = CaptchaTypeMixed
		req.Count = 4
	}

	if req.Count <= 0 || req.Count > 8 {
		req.Count = 4
	}

	var chars string
	if req.CustomSet != "" {
		chars = req.CustomSet
	} else {
		switch req.Type {
		case CaptchaTypeNumber:
			chars = digitCharSet
		case CaptchaTypeLetter:
			chars = letterCharSet
		default:
			chars = allCharSet
		}
	}

	if req.NoiseMode <= 0 {
		req.NoiseMode = randInt(1, 5)
	}
	if req.LineMode <= 0 {
		req.LineMode = randInt(1, 5)
	}

	answer := generateRandomString(chars, req.Count)
	challengeID := uuid.New().String()

	img := generateCaptchaImage(answer, req.NoiseMode, req.LineMode)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		response.InternalServerError(c, "failed to generate captcha image")
		return
	}

	imageBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	setCaptchaAnswer(challengeID, answer)

	response.Success(c, GenerateImageCaptchaResponse{
		ChallengeID: challengeID,
		Image:       imageBase64,
	})
}

var fallbackCaptchaStore = make(map[string]string)

func setCaptchaAnswer(challengeID, answer string) {
	if redis.Client != nil {
		ctx := context.Background()
		redis.Client.Set(ctx, "captcha:"+challengeID, strings.ToLower(answer), captchaTTL)
	} else {
		fallbackCaptchaStore[challengeID] = strings.ToLower(answer)
	}
}

func getCaptchaAnswer(challengeID string) (string, bool) {
	if redis.Client != nil {
		ctx := context.Background()
		answer, err := redis.Client.Get(ctx, "captcha:"+challengeID).Result()
		if err == nil {
			return answer, true
		}
		return "", false
	}
	answer, ok := fallbackCaptchaStore[challengeID]
	return answer, ok
}

func deleteCaptchaAnswer(challengeID string) {
	if redis.Client != nil {
		ctx := context.Background()
		redis.Client.Del(ctx, "captcha:"+challengeID)
	} else {
		delete(fallbackCaptchaStore, challengeID)
	}
}

func VerifyImageCaptcha(c *gin.Context) {
	var req VerifyImageCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	storedAnswer, found := getCaptchaAnswer(req.ChallengeID)
	if !found {
		response.NotFound(c, "captcha expired or not found")
		return
	}

	success := strings.ToLower(req.Answer) == storedAnswer

	if success {
		deleteCaptchaAnswer(req.ChallengeID)
	}

	response.Success(c, VerifyImageCaptchaResponse{
		Success: success,
	})
}

func randInt(min, max int) int {
	rMu.Lock()
	defer rMu.Unlock()
	return min + r.Intn(max-min+1)
}

func generateRandomString(chars string, length int) string {
	rMu.Lock()
	defer rMu.Unlock()
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}

func generateCaptchaImage(text string, noiseMode, lineMode int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))

	bgColor := randomLightColor()
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	addComplexNoise(img, noiseMode)
	addComplexLines(img, lineMode)

	drawWarpedText(img, text)

	return img
}

func randomLightColor() color.RGBA {
	return color.RGBA{
		R: uint8(200 + randInt(0, 55)),
		G: uint8(200 + randInt(0, 55)),
		B: uint8(200 + randInt(0, 55)),
		A: 255,
	}
}

func randomDarkColor() color.RGBA {
	return color.RGBA{
		R: uint8(randInt(10, 100)),
		G: uint8(randInt(10, 100)),
		B: uint8(randInt(10, 100)),
		A: 255,
	}
}

func randomVividColor() color.RGBA {
	h := float64(randInt(0, 360))
	s := float64(randInt(50, 100)) / 100.0
	l := float64(randInt(30, 60)) / 100.0

	return hslToRgb(h, s, l)
}

func hslToRgb(h, s, l float64) color.RGBA {
	var r, g, b float64

	if s == 0 {
		r, g, b = l, l, l
	} else {
		h = h / 360
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = hueToRgb(p, q, h+1.0/3.0)
		g = hueToRgb(p, q, h)
		b = hueToRgb(p, q, h-1.0/3.0)
	}

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func addComplexNoise(img *image.RGBA, mode int) {
	switch mode {
	case 1:
		addDotNoise(img)
	case 2:
		addLineNoise(img)
	case 3:
		addGridNoise(img)
	case 4:
		addWaveNoise(img)
	case 5:
		addSpiralNoise(img)
	default:
		addDotNoise(img)
	}
}

func addDotNoise(img *image.RGBA) {
	for i := 0; i < 120; i++ {
		x := randInt(0, captchaWidth)
		y := randInt(0, captchaHeight)
		img.Set(x, y, randomDarkColor())
	}
	for i := 0; i < 60; i++ {
		x := randInt(0, captchaWidth)
		y := randInt(0, captchaHeight)
		size := randInt(1, 2)
		for dx := 0; dx < size; dx++ {
			for dy := 0; dy < size; dy++ {
				if x+dx < captchaWidth && y+dy < captchaHeight {
					img.Set(x+dx, y+dy, randomDarkColor())
				}
			}
		}
	}
}

func addLineNoise(img *image.RGBA) {
	for i := 0; i < 30; i++ {
		x1 := randInt(0, captchaWidth)
		y1 := randInt(0, captchaHeight)
		length := randInt(5, 20)
		angle := float64(randInt(0, 360)) * math.Pi / 180
		x2 := x1 + int(float64(length)*math.Cos(angle))
		y2 := y1 + int(float64(length)*math.Sin(angle))
		drawThickLine(img, x1, y1, x2, y2, randInt(1, 2), randomDarkColor())
	}
}

func addGridNoise(img *image.RGBA) {
	for x := 0; x < captchaWidth; x += randInt(8, 15) {
		for y := 0; y < captchaHeight; y += randInt(8, 15) {
			if randInt(0, 10) > 7 {
				size := randInt(1, 3)
				for dx := 0; dx < size; dx++ {
					for dy := 0; dy < size; dy++ {
						if x+dx < captchaWidth && y+dy < captchaHeight {
							img.Set(x+dx, y+dy, randomDarkColor())
						}
					}
				}
			}
		}
	}
}

func addWaveNoise(img *image.RGBA) {
	for i := 0; i < 8; i++ {
		startX := randInt(0, captchaWidth)
		startY := randInt(0, captchaHeight)
		amplitude := float64(randInt(3, 10))
		frequency := float64(randInt(1, 5)) * 0.1
		length := randInt(30, 80)
		phase := float64(randInt(0, 628)) / 100.0

		for x := 0; x < length && startX+x < captchaWidth; x++ {
			y := startY + int(amplitude*math.Sin(float64(x)*frequency+phase))
			if y >= 0 && y < captchaHeight {
				img.Set(startX+x, y, randomDarkColor())
				if y+1 < captchaHeight {
					img.Set(startX+x, y+1, randomDarkColor())
				}
			}
		}
	}
}

func addSpiralNoise(img *image.RGBA) {
	centerX := randInt(captchaWidth/4, captchaWidth*3/4)
	centerY := randInt(captchaHeight/4, captchaHeight*3/4)
	maxRadius := randInt(15, 30)
	turns := float64(randInt(1, 3))

	for radius := 0; radius < maxRadius; radius++ {
		for angle := 0.0; angle < turns*2*math.Pi; angle += 0.1 {
			x := centerX + int(float64(radius)*math.Cos(angle))
			y := centerY + int(float64(radius)*math.Sin(angle))
			if x >= 0 && x < captchaWidth && y >= 0 && y < captchaHeight {
				img.Set(x, y, randomDarkColor())
			}
		}
	}
}

func addComplexLines(img *image.RGBA, mode int) {
	switch mode {
	case 1:
		addSimpleCurvedLines(img)
	case 2:
		addBezierLines(img)
	case 3:
		addWavyLines(img)
	case 4:
		addArcLines(img)
	case 5:
		addMixedLines(img)
	default:
		addSimpleCurvedLines(img)
	}
}

func addSimpleCurvedLines(img *image.RGBA) {
	for i := 0; i < 5; i++ {
		x1 := randInt(0, captchaWidth)
		y1 := randInt(0, captchaHeight)
		x2 := randInt(0, captchaWidth)
		y2 := randInt(0, captchaHeight)
		ctrlX := randInt(0, captchaWidth)
		ctrlY := randInt(0, captchaHeight)
		drawQuadraticBezier(img, x1, y1, ctrlX, ctrlY, x2, y2, randomDarkColor())
	}
}

func addBezierLines(img *image.RGBA) {
	for i := 0; i < 4; i++ {
		x1 := randInt(0, captchaWidth)
		y1 := randInt(0, captchaHeight)
		x2 := randInt(0, captchaWidth)
		y2 := randInt(0, captchaHeight)
		ctrlX1 := randInt(0, captchaWidth)
		ctrlY1 := randInt(0, captchaHeight)
		ctrlX2 := randInt(0, captchaWidth)
		ctrlY2 := randInt(0, captchaHeight)
		drawCubicBezier(img, x1, y1, ctrlX1, ctrlY1, ctrlX2, ctrlY2, x2, y2, randomDarkColor())
	}
}

func addWavyLines(img *image.RGBA) {
	for i := 0; i < 3; i++ {
		startY := randInt(5, captchaHeight-5)
		amplitude := float64(randInt(5, 15))
		frequency := float64(randInt(2, 5)) * 0.05
		phase := float64(randInt(0, 628)) / 100.0
		thickness := randInt(1, 3)

		for x := 0; x < captchaWidth; x++ {
			y := startY + int(amplitude*math.Sin(float64(x)*frequency+phase))
			if y >= 0 && y < captchaHeight {
				for t := 0; t < thickness; t++ {
					if y+t < captchaHeight {
						img.Set(x, y+t, randomDarkColor())
					}
				}
			}
		}
	}
}

func addArcLines(img *image.RGBA) {
	for i := 0; i < 4; i++ {
		centerX := randInt(captchaWidth/4, captchaWidth*3/4)
		centerY := randInt(captchaHeight/4, captchaHeight*3/4)
		radius := randInt(20, 50)
		startAngle := float64(randInt(0, 360)) * math.Pi / 180
		endAngle := startAngle + float64(randInt(60, 180))*math.Pi/180
		thickness := randInt(1, 2)

		for angle := startAngle; angle < endAngle; angle += 0.05 {
			x := centerX + int(float64(radius)*math.Cos(angle))
			y := centerY + int(float64(radius)*math.Sin(angle))
			if x >= 0 && x < captchaWidth && y >= 0 && y < captchaHeight {
				for t := 0; t < thickness; t++ {
					if x+t < captchaWidth {
						img.Set(x+t, y, randomDarkColor())
					}
				}
			}
		}
	}
}

func addMixedLines(img *image.RGBA) {
	addSimpleCurvedLines(img)
	addWavyLines(img)
	addArcLines(img)
}

func drawQuadraticBezier(img *image.RGBA, x0, y0, cx, cy, x1, y1 int, col color.Color) {
	points := calculateQuadraticBezierPoints(x0, y0, cx, cy, x1, y1, 50)
	for _, p := range points {
		if p.X >= 0 && p.X < captchaWidth && p.Y >= 0 && p.Y < captchaHeight {
			img.Set(p.X, p.Y, col)
		}
	}
}

func drawCubicBezier(img *image.RGBA, x0, y0, cx1, cy1, cx2, cy2, x1, y1 int, col color.Color) {
	points := calculateCubicBezierPoints(x0, y0, cx1, cy1, cx2, cy2, x1, y1, 50)
	for _, p := range points {
		if p.X >= 0 && p.X < captchaWidth && p.Y >= 0 && p.Y < captchaHeight {
			img.Set(p.X, p.Y, col)
		}
	}
}

type point struct {
	X, Y int
}

func calculateQuadraticBezierPoints(x0, y0, cx, cy, x1, y1, steps int) []point {
	points := make([]point, 0, steps)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		mt := 1 - t
		x := int(mt*mt*float64(x0) + 2*mt*t*float64(cx) + t*t*float64(x1))
		y := int(mt*mt*float64(y0) + 2*mt*t*float64(cy) + t*t*float64(y1))
		points = append(points, point{X: x, Y: y})
	}
	return points
}

func calculateCubicBezierPoints(x0, y0, cx1, cy1, cx2, cy2, x1, y1, steps int) []point {
	points := make([]point, 0, steps)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		mt := 1 - t
		mt2 := mt * mt
		t2 := t * t
		x := int(mt2*mt*float64(x0) + 3*mt2*t*float64(cx1) + 3*mt*t2*float64(cx2) + t2*t*float64(x1))
		y := int(mt2*mt*float64(y0) + 3*mt2*t*float64(cy1) + 3*mt*t2*float64(cy2) + t2*t*float64(y1))
		points = append(points, point{X: x, Y: y})
	}
	return points
}

func drawThickLine(img *image.RGBA, x1, y1, x2, y2, thickness int, col color.Color) {
	dx := imageAbs(x2 - x1)
	dy := imageAbs(y2 - y1)
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
		for t := 0; t < thickness; t++ {
			if x1 >= 0 && x1 < captchaWidth && y1+t >= 0 && y1+t < captchaHeight {
				img.Set(x1, y1+t, col)
			}
			if x1+t >= 0 && x1+t < captchaWidth && y1 >= 0 && y1 < captchaHeight {
				img.Set(x1+t, y1, col)
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

func drawWarpedText(img *image.RGBA, text string) {
	charWidth := captchaWidth / len(text)
	face := basicfont.Face7x13

	textColor := randomDarkColor()

	for i, char := range text {
		baseX := i*charWidth + (charWidth-7)/2
		baseY := captchaHeight/2 + 5

		offsetX := randInt(-3, 3)
		offsetY := randInt(-4, 4)
		baseX += offsetX
		baseY += offsetY

		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(textColor),
			Face: face,
			Dot: fixed.Point26_6{
				X: fixed.I(baseX),
				Y: fixed.I(baseY),
			},
		}
		d.DrawString(string(char))
	}

	applyTextWarpEffect(img, text)
}

func applyTextWarpEffect(img *image.RGBA, text string) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	copyBounds := newImg.Bounds()
	draw.Draw(newImg, copyBounds, &image.Uniform{C: randomLightColor()}, image.Point{}, draw.Src)

	charWidth := width / len(text)
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			charIndex := i / charWidth
			charCenter := charIndex*charWidth + charWidth/2
			_ = charCenter
			waveAmplitude := float64(charWidth) * 0.1
			waveFrequency := 0.1
			phase := float64(charIndex) * 0.5
			offset := int(waveAmplitude * math.Sin(float64(j)*waveFrequency+phase))

			srcY := j - offset
			if srcY >= 0 && srcY < height {
				newImg.Set(i, j, img.At(i, srcY))
			}
		}
	}

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, newImg.At(x, y))
		}
	}
}

func imageAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func getCharSetByType(charSetType CharSetType) string {
	switch charSetType {
	case Numeric:
		return numericCharSet
	case Alphabetic:
		return alphabeticCharSet
	case Alphanumeric:
		return alphanumericCharSet
	case Chinese:
		return chineseCharSet
	default:
		return alphanumericCharSet
	}
}

func getImageDifficultyConfig(difficulty DifficultyLevel) (lineCount, noiseCount, distortionLevel int) {
	switch difficulty {
	case Easy:
		return 3, 50, 1
	case Medium:
		return 8, 120, 2
	case Hard:
		return 15, 200, 3
	case Expert:
		return 25, 300, 4
	default:
		return 5, 100, 2
	}
}

func GenerateDistortedChar(f *sfnt.Font, char rune, angle float64, size float64) *image.RGBA {
	width := 40
	height := 40
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	dpi := 72.0
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		face = basicfont.Face7x13
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(width / 2),
			Y: fixed.I(height / 2),
		},
	}
	d.DrawString(string(char))

	if angle != 0 || size != 13 {
		img = applyWaveDistortion(img, angle, size)
	}

	return img
}

func applyWaveDistortion(src *image.RGBA, angle, size float64) *image.RGBA {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	centerX := width / 2
	centerY := height / 2

	cosA := math.Cos(angle * math.Pi / 180)
	sinA := math.Sin(angle * math.Pi / 180)

	distortionFactor := size / 20.0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := float64(x - centerX)
			dy := float64(y - centerY)

			dist := math.Sqrt(dx*dx + dy*dy)
			waveOffset := dist * distortionFactor * 0.1

			newX := cosA*float64(x-centerX) - sinA*float64(y-centerY) + waveOffset
			newY := sinA*float64(x-centerX) + cosA*float64(y-centerY) + waveOffset

			srcX := int(newX) + centerX
			srcY := int(newY) + centerY

			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				dst.Set(x, y, src.At(srcX, srcY))
			} else {
				dst.Set(x, y, image.Transparent)
			}
		}
	}

	return dst
}

func GenerateConnectedChars(chars []rune, f *sfnt.Font, charSize float64) []*image.RGBA {
	if len(chars) < 2 {
		result := make([]*image.RGBA, len(chars))
		for i, char := range chars {
			result[i] = GenerateDistortedChar(f, char, 0, charSize)
		}
		return result
	}

	connectedCount := randInt(2, 3)
	if connectedCount > len(chars) {
		connectedCount = len(chars)
	}

	startIdx := randInt(0, len(chars)-connectedCount+1)

	result := make([]*image.RGBA, len(chars))
	charWidth := 40

	for i := 0; i < len(chars); i++ {
		offsetX := 0
		if i >= startIdx && i < startIdx+connectedCount && i > startIdx {
			overlap := randInt(-5, -2)
			offsetX = overlap
		}

		charImg := GenerateDistortedChar(f, chars[i], float64(randInt(-15, 15)), charSize+float64(randInt(-2, 2)))

		if offsetX != 0 && charImg != nil {
			newImg := image.NewRGBA(image.Rect(0, 0, charWidth, charImg.Bounds().Dy()))
			for y := 0; y < charImg.Bounds().Dy(); y++ {
				for x := offsetX; x < charWidth && x-offsetX < charImg.Bounds().Dx(); x++ {
					if x >= 0 && x < charWidth {
						newImg.Set(x, y, charImg.At(x-offsetX, y))
					}
				}
			}
			charImg = newImg
		}

		result[i] = charImg
	}

	return result
}

func GenerateInterferenceLines(img *image.RGBA, difficulty DifficultyLevel) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	lineCount, _, _ := getImageDifficultyConfig(difficulty)

	for i := 0; i < lineCount; i++ {
		lineType := randInt(0, 3)
		color := randomDarkColor()

		switch lineType {
		case 0:
			drawRandomStraightLine(img, width, height, color)
		case 1:
			drawRandomCurveLine(img, width, height, color)
		case 2:
			drawRandomBezierLine(img, width, height, color)
		case 3:
			drawRandomWavyLine(img, width, height, color)
		}
	}

	return nil
}

func drawRandomStraightLine(img *image.RGBA, width, height int, col color.RGBA) {
	x1 := randInt(0, width)
	y1 := randInt(0, height)
	x2 := randInt(0, width)
	y2 := randInt(0, height)
	thickness := randInt(1, 2)
	drawThickLine(img, x1, y1, x2, y2, thickness, col)
}

func drawRandomCurveLine(img *image.RGBA, width, height int, col color.RGBA) {
	x1 := randInt(0, width)
	y1 := randInt(0, height)
	x2 := randInt(0, width)
	y2 := randInt(0, height)
	ctrlX := randInt(0, width)
	ctrlY := randInt(0, height)
	drawQuadraticBezier(img, x1, y1, ctrlX, ctrlY, x2, y2, col)
}

func drawRandomBezierLine(img *image.RGBA, width, height int, col color.RGBA) {
	x1 := randInt(0, width)
	y1 := randInt(0, height)
	x2 := randInt(0, width)
	y2 := randInt(0, height)
	ctrlX1 := randInt(0, width)
	ctrlY1 := randInt(0, height)
	ctrlX2 := randInt(0, width)
	ctrlY2 := randInt(0, height)
	drawCubicBezier(img, x1, y1, ctrlX1, ctrlY1, ctrlX2, ctrlY2, x2, y2, col)
}

func drawRandomWavyLine(img *image.RGBA, width, height int, col color.RGBA) {
	startY := randInt(5, height-5)
	amplitude := float64(randInt(5, 15))
	frequency := float64(randInt(2, 5)) * 0.05
	phase := float64(randInt(0, 628)) / 100.0
	thickness := randInt(1, 2)

	for x := 0; x < width; x++ {
		y := startY + int(amplitude*math.Sin(float64(x)*frequency+phase))
		if y >= 0 && y < height {
			for t := 0; t < thickness; t++ {
				if y+t < height {
					img.Set(x, y+t, col)
				}
			}
		}
	}
}

func AddNoise(img *image.RGBA, density float64) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	noiseCount := int(float64(width*height) * density)

	for i := 0; i < noiseCount; i++ {
		x := randInt(0, width)
		y := randInt(0, height)

		noiseType := randInt(0, 2)
		switch noiseType {
		case 0:
			img.Set(x, y, randomDarkColor())
		case 1:
			size := randInt(1, 2)
			for dx := 0; dx < size; dx++ {
				for dy := 0; dy < size; dy++ {
					if x+dx < width && y+dy < height {
						img.Set(x+dx, y+dy, randomDarkColor())
					}
				}
			}
		case 2:
			c := randomVividColor()
			img.Set(x, y, c)
		}
	}

	return nil
}

func GenerateGradientBackground(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	gradientType := randInt(0, 2)

	switch gradientType {
	case 0:
		drawLinearGradient(img, width, height)
	case 1:
		drawRadialGradient(img, width, height)
	case 2:
		drawDiagonalGradient(img, width, height)
	default:
		drawLinearGradient(img, width, height)
	}

	return img
}

func drawLinearGradient(img *image.RGBA, width, height int) {
	startColor := randomLightColor()
	endColor := randomLightColor()

	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		r := uint8(float64(startColor.R)*(1-ratio) + float64(endColor.R)*ratio)
		g := uint8(float64(startColor.G)*(1-ratio) + float64(endColor.G)*ratio)
		b := uint8(float64(startColor.B)*(1-ratio) + float64(endColor.B)*ratio)

		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func drawRadialGradient(img *image.RGBA, width, height int) {
	centerX := width / 2
	centerY := height / 2
	maxRadius := int(math.Sqrt(float64(centerX*centerX + centerY*centerY)))

	startColor := randomLightColor()
	endColor := randomLightColor()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := x - centerX
			dy := y - centerY
			dist := int(math.Sqrt(float64(dx*dx + dy*dy)))
			ratio := float64(dist) / float64(maxRadius)
			if ratio > 1 {
				ratio = 1
			}

			r := uint8(float64(startColor.R)*(1-ratio) + float64(endColor.R)*ratio)
			g := uint8(float64(startColor.G)*(1-ratio) + float64(endColor.G)*ratio)
			b := uint8(float64(startColor.B)*(1-ratio) + float64(endColor.B)*ratio)

			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func drawDiagonalGradient(img *image.RGBA, width, height int) {
	startColor := randomLightColor()
	endColor := randomLightColor()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ratio := (float64(x) + float64(y)) / float64(width+height)
			r := uint8(float64(startColor.R)*(1-ratio) + float64(endColor.R)*ratio)
			g := uint8(float64(startColor.G)*(1-ratio) + float64(endColor.G)*ratio)
			b := uint8(float64(startColor.B)*(1-ratio) + float64(endColor.B)*ratio)

			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func GenerateTexture(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	baseColor := randomLightColor()
	draw.Draw(img, img.Bounds(), &image.Uniform{C: baseColor}, image.Point{}, draw.Src)

	textureType := randInt(0, 3)

	switch textureType {
	case 0:
		addDotPattern(img, width, height)
	case 1:
		addLinePattern(img, width, height)
	case 2:
		addCirclePattern(img, width, height)
	case 3:
		addCheckerboardPattern(img, width, height)
	}

	return img
}

func addDotPattern(img *image.RGBA, width, height int) {
	dotColor := randomDarkColor()
	spacing := randInt(8, 15)

	for x := 0; x < width; x += spacing {
		for y := 0; y < height; y += spacing {
			if randInt(0, 10) > 7 {
				size := randInt(1, 3)
				for dx := 0; dx < size; dx++ {
					for dy := 0; dy < size; dy++ {
						if x+dx < width && y+dy < height {
							img.Set(x+dx, y+dy, dotColor)
						}
					}
				}
			}
		}
	}
}

func addLinePattern(img *image.RGBA, width, height int) {
	lineColor := randomDarkColor()

	for i := 0; i < 10; i++ {
		x1 := randInt(0, width)
		y1 := randInt(0, height)
		x2 := randInt(0, width)
		y2 := randInt(0, height)
		drawThickLine(img, x1, y1, x2, y2, 1, lineColor)
	}
}

func addCirclePattern(img *image.RGBA, width, height int) {
	circleColor := randomDarkColor()

	for i := 0; i < 8; i++ {
		cx := randInt(0, width)
		cy := randInt(0, height)
		radius := randInt(3, 8)

		for angle := 0.0; angle < 2*math.Pi; angle += 0.1 {
			x := cx + int(float64(radius)*math.Cos(angle))
			y := cy + int(float64(radius)*math.Sin(angle))
			if x >= 0 && x < width && y >= 0 && y < height {
				img.Set(x, y, circleColor)
			}
		}
	}
}

func addCheckerboardPattern(img *image.RGBA, width, height int) {
	patternColor := randomDarkColor()
	blockSize := randInt(6, 12)

	for x := 0; x < width; x += blockSize {
		for y := 0; y < height; y += blockSize {
			if (x/blockSize+y/blockSize)%2 == 0 {
				for dx := 0; dx < blockSize && x+dx < width; dx++ {
					for dy := 0; dy < blockSize && y+dy < height; dy++ {
						if randInt(0, 10) > 8 {
							img.Set(x+dx, y+dy, patternColor)
						}
					}
				}
			}
		}
	}
}

func GenerateEnhancedCaptchaImage(text string, config ImageConfig) *image.RGBA {
	width := config.Width
	height := config.Height

	var background *image.RGBA
	if randInt(0, 1) == 0 {
		background = GenerateGradientBackground(width, height)
	} else {
		background = GenerateTexture(width, height)
	}

	_, noiseCount, _ := getImageDifficultyConfig(config.Difficulty)
	noiseDensity := float64(noiseCount) / float64(width*height)
	AddNoise(background, noiseDensity)

	GenerateInterferenceLines(background, config.Difficulty)

	chars := []rune(text)

	face := basicfont.Face7x13
	charWidth := width / len(chars)

	textColor := randomDarkColor()

	for i, char := range chars {
		baseX := i*charWidth + (charWidth-7)/2
		baseY := height/2 + 5

		offsetX := randInt(-5, 5)
		offsetY := randInt(-4, 4)
		baseX += offsetX
		baseY += offsetY

		_ = float64(randInt(-30, 30))

		d := &font.Drawer{
			Dst:  background,
			Src:  image.NewUniform(textColor),
			Face: face,
			Dot: fixed.Point26_6{
				X: fixed.I(baseX),
				Y: fixed.I(baseY),
			},
		}
		d.DrawString(string(char))
	}

	applyAdvancedTextWarpEffect(background, text)

	return background
}

func applyAdvancedTextWarpEffect(img *image.RGBA, text string) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	copyBounds := newImg.Bounds()

	baseColor := randomLightColor()
	draw.Draw(newImg, copyBounds, &image.Uniform{C: baseColor}, image.Point{}, draw.Src)

	charWidth := width / len(text)

	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			charIndex := i / charWidth
			charCenter := charIndex*charWidth + charWidth/2

			waveAmplitude := float64(charWidth) * 0.15
			waveFrequency := 0.08
			phase := float64(charIndex) * 0.8

			distanceFromCenter := math.Abs(float64(i - charCenter))
			amplitudeFactor := 1.0 - (distanceFromCenter / float64(width))

			offset := int(waveAmplitude * amplitudeFactor * math.Sin(float64(j)*waveFrequency+phase))

			srcY := j - offset
			if srcY >= 0 && srcY < height {
				newImg.Set(i, j, img.At(i, srcY))
			}
		}
	}

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, newImg.At(x, y))
		}
	}
}

// Rotation CAPTCHA constants
const (
	rotationCaptchaSize    = 200
	rotationCaptchaTTL     = 5 * time.Minute
	rotationAngleTolerance = 8
)

type GenerateRotationCaptchaRequest struct{}

type GenerateRotationCaptchaResponse struct {
	ChallengeID string `json:"challenge_id"`
	Image       string `json:"image"`
}

type VerifyRotationCaptchaRequest struct {
	ChallengeID string `json:"challenge_id" binding:"required"`
	Angle       *int   `json:"angle" binding:"required"`
}

type VerifyRotationCaptchaResponse struct {
	Success bool `json:"success"`
}

var (
	rotationCaptchaStore = make(map[string]int)
	rotationStoreMu      sync.Mutex
)

func setRotationCaptchaAnswer(challengeID string, angle int) {
	rotationStoreMu.Lock()
	defer rotationStoreMu.Unlock()
	if redis.Client != nil {
		ctx := context.Background()
		redis.Client.Set(ctx, "rotation:"+challengeID, angle, rotationCaptchaTTL)
	} else {
		rotationCaptchaStore[challengeID] = angle
	}
}

func getRotationCaptchaAnswer(challengeID string) (int, bool) {
	rotationStoreMu.Lock()
	defer rotationStoreMu.Unlock()
	if redis.Client != nil {
		ctx := context.Background()
		val, err := redis.Client.Get(ctx, "rotation:"+challengeID).Int()
		if err == nil {
			return val, true
		}
		return 0, false
	}
	angle, ok := rotationCaptchaStore[challengeID]
	return angle, ok
}

func deleteRotationCaptchaAnswer(challengeID string) {
	rotationStoreMu.Lock()
	defer rotationStoreMu.Unlock()
	if redis.Client != nil {
		ctx := context.Background()
		redis.Client.Del(ctx, "rotation:"+challengeID)
	} else {
		delete(rotationCaptchaStore, challengeID)
	}
}

func generateRandomAngle() int {
	return randInt(0, 359)
}

func verifyRotationAngle(stored, submitted int) bool {
	diff := submitted - stored
	if diff < 0 {
		diff = -diff
	}
	if diff > 180 {
		diff = 360 - diff
	}
	return diff <= rotationAngleTolerance
}

func GenerateRotationCaptcha(c *gin.Context) {
	angle := generateRandomAngle()
	challengeID := uuid.New().String()

	img := generateRotationCaptchaImage(angle)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		response.InternalServerError(c, "failed to generate rotation captcha image")
		return
	}

	imageBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	setRotationCaptchaAnswer(challengeID, angle)

	response.Success(c, GenerateRotationCaptchaResponse{
		ChallengeID: challengeID,
		Image:       imageBase64,
	})
}

func VerifyRotationCaptcha(c *gin.Context) {
	var req VerifyRotationCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if req.Angle == nil {
		response.BadRequest(c, "angle is required")
		return
	}

	angle := *req.Angle
	if angle < 0 || angle > 359 {
		response.BadRequest(c, "angle must be between 0 and 359")
		return
	}

	storedAngle, found := getRotationCaptchaAnswer(req.ChallengeID)
	if !found {
		response.NotFound(c, "rotation captcha expired or not found")
		return
	}

	success := verifyRotationAngle(storedAngle, angle)

	if success {
		deleteRotationCaptchaAnswer(req.ChallengeID)
	}

	response.Success(c, VerifyRotationCaptchaResponse{
		Success: success,
	})
}

func generateRotationCaptchaImage(angle int) *image.RGBA {
	size := rotationCaptchaSize
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	bgColor := color.RGBA{245, 245, 250, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	cx, cy := size/2, size/2
	radius := size/2 - 12

	drawRotationCircleOutline(img, cx, cy, radius, color.RGBA{80, 80, 90, 255})
	drawRotationCircleOutline(img, cx, cy, radius-25, color.RGBA{180, 180, 190, 255})

	drawRotationTickMark(img, cx, cy, radius, 0, color.RGBA{220, 50, 50, 255})
	drawRotationTickMark(img, cx, cy, radius, 90, color.RGBA{50, 150, 50, 255})
	drawRotationTickMark(img, cx, cy, radius, 180, color.RGBA{50, 50, 200, 255})
	drawRotationTickMark(img, cx, cy, radius, 270, color.RGBA{200, 160, 30, 255})

	drawRotationArrow(img, cx, cy, radius-8, color.RGBA{220, 50, 50, 255})

	for i := 0; i < 12; i++ {
		a := i * 30
		drawRotationDot(img, cx, cy, radius-15, a, color.RGBA{120, 120, 130, 200})
	}

	drawRotationCenterCircle(img, cx, cy, 8, color.RGBA{80, 80, 90, 255})

	rotated := rotateImageRGBA(img, float64(angle))
	return rotated
}

func drawRotationCircleOutline(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	for a := 0; a < 360; a++ {
		rad := float64(a) * math.Pi / 180
		x := cx + int(float64(radius)*math.Cos(rad))
		y := cy + int(float64(radius)*math.Sin(rad))
		if x >= 0 && x < rotationCaptchaSize && y >= 0 && y < rotationCaptchaSize {
			img.Set(x, y, col)
		}
	}
}

func drawRotationTickMark(img *image.RGBA, cx, cy, radius, angleDeg int, col color.RGBA) {
	rad := float64(angleDeg) * math.Pi / 180
	innerRadius := radius - 12
	for r := innerRadius; r <= radius; r++ {
		x := cx + int(float64(r)*math.Cos(rad))
		y := cy + int(float64(r)*math.Sin(rad))
		if x >= 0 && x < rotationCaptchaSize && y >= 0 && y < rotationCaptchaSize {
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					px, py := x+dx, y+dy
					if px >= 0 && px < rotationCaptchaSize && py >= 0 && py < rotationCaptchaSize {
						img.Set(px, py, col)
					}
				}
			}
		}
	}
}

func drawRotationArrow(img *image.RGBA, cx, cy, length int, col color.RGBA) {
	arrowHeadSize := 10
	shaftLength := length - arrowHeadSize

	for i := 0; i < shaftLength; i++ {
		x := cx
		y := cy - i
		if y >= 0 && y < rotationCaptchaSize {
			img.Set(x, y, col)
			img.Set(x-1, y, col)
			img.Set(x+1, y, col)
		}
	}

	tipY := cy - length
	for dx := -arrowHeadSize; dx <= arrowHeadSize; dx++ {
		for dy := -arrowHeadSize; dy <= 0; dy++ {
			absDx := dx
			if absDx < 0 {
				absDx = -absDx
			}
			if absDx+(-dy) <= arrowHeadSize {
				x := cx + dx
				y := tipY + dy
				if x >= 0 && x < rotationCaptchaSize && y >= 0 && y < rotationCaptchaSize {
					img.Set(x, y, col)
				}
			}
		}
	}
}

func drawRotationDot(img *image.RGBA, cx, cy, radius, angleDeg int, col color.RGBA) {
	rad := float64(angleDeg) * math.Pi / 180
	x := cx + int(float64(radius)*math.Cos(rad))
	y := cy + int(float64(radius)*math.Sin(rad))
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			if dx*dx+dy*dy <= 4 {
				px, py := x+dx, y+dy
				if px >= 0 && px < rotationCaptchaSize && py >= 0 && py < rotationCaptchaSize {
					img.Set(px, py, col)
				}
			}
		}
	}
}

func drawRotationCenterCircle(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < rotationCaptchaSize && y >= 0 && y < rotationCaptchaSize {
					img.Set(x, y, col)
				}
			}
		}
	}
}
