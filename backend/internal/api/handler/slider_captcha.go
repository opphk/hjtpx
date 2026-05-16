package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/crypto"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type PuzzleShape int

const (
	ShapeSquare PuzzleShape = iota
	ShapeCircle
	ShapeTriangle
	ShapeDiamond
	ShapeHexagon
)

type TrajectoryPoint struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	T int64 `json:"t"`
}

type EncryptedTrajectoryPayload struct {
	Timestamp     int64  `json:"timestamp"`
	Salt          string `json:"salt"`
	EncryptedData string `json:"encrypted_data"`
	Signature     string `json:"signature"`
}

type TrajectoryEncryption struct {
	secretKey          []byte
	saltManager        *crypto.SaltManager
	maxTimeDrift       time.Duration
	maxPayloadAge      time.Duration
	compressionEnabled bool
}

var (
	trajectoryEncryptor *TrajectoryEncryption
	encryptorOnce       sync.Once
)

func init() {
	go cleanupExpiredSliderSessions()
}

func getTrajectoryEncryptor() *TrajectoryEncryption {
	encryptorOnce.Do(func() {
		secretKey := []byte("captcha-trajectory-secret-key-2024")
		trajectoryEncryptor = NewTrajectoryEncryption(secretKey)
	})
	return trajectoryEncryptor
}

func NewTrajectoryEncryption(secretKey []byte) *TrajectoryEncryption {
	return &TrajectoryEncryption{
		secretKey:          secretKey,
		saltManager:        crypto.NewSaltManager(16),
		maxTimeDrift:       5 * time.Minute,
		maxPayloadAge:      10 * time.Minute,
		compressionEnabled: true,
	}
}

func (te *TrajectoryEncryption) EncryptTrajectory(trajectory []TrajectoryPoint) (*EncryptedTrajectoryPayload, error) {
	if len(trajectory) == 0 {
		return nil, errors.New("empty trajectory data")
	}

	timestamp := time.Now().UnixMilli()
	salt, err := crypto.GenerateRandomString(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	trajectoryJSON, err := json.Marshal(trajectory)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trajectory: %w", err)
	}

	encryptedData, err := te.encryptData(trajectoryJSON, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt trajectory: %w", err)
	}

	signature := te.generateSignature(timestamp, salt, encryptedData)

	return &EncryptedTrajectoryPayload{
		Timestamp:     timestamp,
		Salt:          salt,
		EncryptedData: encryptedData,
		Signature:     signature,
	}, nil
}

func (te *TrajectoryEncryption) DecryptTrajectory(payload *EncryptedTrajectoryPayload) ([]TrajectoryPoint, error) {
	if payload == nil {
		return nil, errors.New("nil payload")
	}

	if err := te.validateTimestamp(payload.Timestamp); err != nil {
		return nil, err
	}

	if err := te.validateSalt(payload.Salt); err != nil {
		return nil, err
	}

	if err := te.validateSignature(payload.Timestamp, payload.Salt, payload.EncryptedData, payload.Signature); err != nil {
		return nil, err
	}

	decryptedData, err := te.decryptData(payload.EncryptedData, payload.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt trajectory: %w", err)
	}

	var trajectory []TrajectoryPoint
	if err := json.Unmarshal(decryptedData, &trajectory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trajectory: %w", err)
	}

	return trajectory, nil
}

func (te *TrajectoryEncryption) encryptData(data []byte, salt string) (string, error) {
	key := te.deriveKey(salt)

	ciphertext, err := crypto.AESEncrypt(data, key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (te *TrajectoryEncryption) decryptData(encryptedData string, salt string) ([]byte, error) {
	key := te.deriveKey(salt)

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	return crypto.AESDecrypt(ciphertext, key)
}

func (te *TrajectoryEncryption) deriveKey(salt string) []byte {
	h := hmac.New(sha256.New, te.secretKey)
	h.Write([]byte(salt))
	return h.Sum(nil)[:32]
}

func (te *TrajectoryEncryption) generateSignature(timestamp int64, salt, encryptedData string) string {
	data := fmt.Sprintf("%d:%s:%s", timestamp, salt, encryptedData)
	mac := hmac.New(sha256.New, te.secretKey)
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (te *TrajectoryEncryption) validateTimestamp(timestamp int64) error {
	now := time.Now().UnixMilli()
	drift := time.Duration(now-timestamp) * time.Millisecond

	if drift < 0 {
		drift = -drift
	}

	if drift > te.maxTimeDrift {
		return fmt.Errorf("timestamp drift too large: %v (max: %v)", drift, te.maxTimeDrift)
	}

	return nil
}

func (te *TrajectoryEncryption) validateSalt(salt string) error {
	if len(salt) != 16 {
		return fmt.Errorf("invalid salt length: %d (expected: 16)", len(salt))
	}

	for _, c := range salt {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return errors.New("salt contains invalid characters")
		}
	}

	return nil
}

func (te *TrajectoryEncryption) validateSignature(timestamp int64, salt, encryptedData, signature string) error {
	expectedSignature := te.generateSignature(timestamp, salt, encryptedData)

	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
		return errors.New("signature verification failed")
	}

	return nil
}

func (te *TrajectoryEncryption) RotateSalt() error {
	return te.saltManager.RotateSalt()
}

func (te *TrajectoryEncryption) ShouldRotate() bool {
	return te.saltManager.ShouldRotate()
}

type TrajectoryResult struct {
	Score   int      `json:"score"`
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
}

type SliderCaptchaConfig struct {
	Width            int
	Height           int
	PuzzleSize       int
	MaxAttempts      int
	SessionTTL       time.Duration
	DefaultTolerance int
}

var defaultSliderConfig = SliderCaptchaConfig{
	Width:            320,
	Height:           160,
	PuzzleSize:       40,
	MaxAttempts:      5,
	SessionTTL:       5 * time.Minute,
	DefaultTolerance: 8,
}

type SliderSession struct {
	SessionID       string      `json:"session_id"`
	SecretX         int         `json:"secret_x"`
	SecretY         int         `json:"secret_y"`
	Tolerance       int         `json:"tolerance"`
	Shape           PuzzleShape `json:"shape"`
	BackgroundImage *imageData  `json:"background_image"`
	PuzzleImage     *imageData  `json:"puzzle_image"`
	HintImage       *imageData  `json:"hint_image"`
	Attempts        int         `json:"attempts"`
	Verified        bool        `json:"verified"`
	CreatedAt       time.Time   `json:"created_at"`
	ExpiresAt       time.Time   `json:"expires_at"`
	ClientIP        string      `json:"client_ip"`
	UserAgent       string      `json:"user_agent"`
}

type imageData struct {
	DataURL string `json:"data_url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

type GenerateSliderRequest struct {
	Width     int `form:"width" json:"width"`
	Height    int `form:"height" json:"height"`
	Tolerance int `form:"tolerance" json:"tolerance"`
}

type GenerateSliderResponse struct {
	SessionID   string `json:"session_id"`
	ImageURL    string `json:"image_url"`
	PuzzleURL   string `json:"puzzle_url"`
	HintURL     string `json:"hint_url"`
	Shape       int    `json:"shape"`
	SecretY     int    `json:"secret_y"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
}

type VerifySliderRequest struct {
	SessionID           string                      `json:"session_id" binding:"required"`
	X                   int                         `json:"x" binding:"required"`
	Y                   int                         `json:"y"`
	Trajectory          []TrajectoryPoint           `json:"trajectory,omitempty"`
	EncryptedTrajectory *EncryptedTrajectoryPayload `json:"encrypted_trajectory,omitempty"`
}

type VerifySliderResponse struct {
	Success          bool              `json:"success"`
	Message          string            `json:"message"`
	Remaining        int               `json:"remaining_attempts"`
	TrajectoryResult *TrajectoryResult `json:"trajectory_result,omitempty"`
}

var (
	sliderSessionStore = make(map[string]*SliderSession)
	sliderSessionMu    sync.RWMutex
	sliderPrng         = rand.New(rand.NewSource(time.Now().UnixNano()))
	sliderPrngMu       sync.Mutex
)

func init() {
	go cleanupExpiredSliderSessions()
}

func cleanupExpiredSliderSessions() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		sliderSessionMu.Lock()
		for id, session := range sliderSessionStore {
			if now.After(session.ExpiresAt) || session.Verified {
				delete(sliderSessionStore, id)
			}
		}
		sliderSessionMu.Unlock()
	}
}

func randIntSlider(min, max int) int {
	sliderPrngMu.Lock()
	defer sliderPrngMu.Unlock()
	return min + sliderPrng.Intn(max-min+1)
}

func generateSliderBackground(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	bgStyle := randIntSlider(0, 4)
	switch bgStyle {
	case 0:
		drawGradientBackground(img)
	case 1:
		drawPatternBackground(img, width, height)
	case 2:
		drawNoiseBackground(img, width, height)
	case 3:
		drawGeometricBackground(img, width, height)
	case 4:
		drawTexturedBackground(img, width, height)
	default:
		drawGradientBackground(img)
	}

	addImageNoise(img, width, height)

	return img
}

type gradientColorSet struct {
	start, end color.RGBA
}

func generateGradientColors() gradientColorSet {
	palettes := []gradientColorSet{
		{start: color.RGBA{R: 102, G: 126, B: 234, A: 255}, end: color.RGBA{R: 118, G: 75, B: 162, A: 255}},
		{start: color.RGBA{R: 77, G: 144, B: 142, A: 255}, end: color.RGBA{R: 43, G: 62, B: 80, A: 255}},
		{start: color.RGBA{R: 52, G: 152, B: 219, A: 255}, end: color.RGBA{R: 155, G: 89, B: 182, A: 255}},
		{start: color.RGBA{R: 241, G: 196, B: 15, A: 255}, end: color.RGBA{R: 230, G: 126, B: 34, A: 255}},
		{start: color.RGBA{R: 46, G: 204, B: 113, A: 255}, end: color.RGBA{R: 26, G: 188, B: 156, A: 255}},
		{start: color.RGBA{R: 231, G: 76, B: 60, A: 255}, end: color.RGBA{R: 192, G: 57, B: 43, A: 255}},
	}

	idx := randIntSlider(0, len(palettes)-1)
	return palettes[idx]
}

func drawPatternBackground(img *image.RGBA, width, height int) {
	bgColor := color.RGBA{
		R: uint8(randIntSlider(40, 120)),
		G: uint8(randIntSlider(40, 120)),
		B: uint8(randIntSlider(40, 120)),
		A: 255,
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	patternType := randIntSlider(0, 3)
	switch patternType {
	case 0:
		drawCirclesPattern(img, width, height)
	case 1:
		drawDotsPattern(img, width, height)
	case 2:
		drawGridPattern(img, width, height)
	case 3:
		drawWavesPattern(img, width, height)
	}
}

func drawCirclesPattern(img *image.RGBA, width, height int) {
	for i := 0; i < 15; i++ {
		cx := randIntSlider(0, width)
		cy := randIntSlider(0, height)
		r := randIntSlider(10, 40)
		circleColor := color.RGBA{
			R: uint8(randIntSlider(150, 255)),
			G: uint8(randIntSlider(150, 255)),
			B: uint8(randIntSlider(150, 255)),
			A: uint8(randIntSlider(10, 40)),
		}

		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				if dx*dx+dy*dy <= r*r {
					px, py := cx+dx, cy+dy
					if px >= 0 && px < width && py >= 0 && py < height {
						img.Set(px, py, circleColor)
					}
				}
			}
		}
	}
}

func drawDotsPattern(img *image.RGBA, width, height int) {
	dotColor := color.RGBA{
		R: uint8(randIntSlider(200, 255)),
		G: uint8(randIntSlider(200, 255)),
		B: uint8(randIntSlider(200, 255)),
		A: uint8(randIntSlider(20, 60)),
	}

	for i := 0; i < 200; i++ {
		x := randIntSlider(0, width-1)
		y := randIntSlider(0, img.Bounds().Dy()-1)
		img.Set(x, y, dotColor)
	}
}

func drawGridPattern(img *image.RGBA, width, height int) {
	lineColor := color.RGBA{
		R: uint8(randIntSlider(100, 180)),
		G: uint8(randIntSlider(100, 180)),
		B: uint8(randIntSlider(100, 180)),
		A: uint8(randIntSlider(15, 35)),
	}

	spacing := randIntSlider(20, 40)

	for x := 0; x < width; x += spacing {
		for y := 0; y < img.Bounds().Dy(); y++ {
			img.Set(x, y, lineColor)
		}
	}

	for y := 0; y < img.Bounds().Dy(); y += spacing {
		for x := 0; x < width; x++ {
			img.Set(x, y, lineColor)
		}
	}
}

func drawWavesPattern(img *image.RGBA, width, height int) {
	for i := 0; i < 5; i++ {
		startY := randIntSlider(0, height)
		amplitude := float64(randIntSlider(5, 20))
		frequency := float64(randIntSlider(1, 3)) * 0.05
		waveColor := color.RGBA{
			R: uint8(randIntSlider(180, 255)),
			G: uint8(randIntSlider(180, 255)),
			B: uint8(randIntSlider(180, 255)),
			A: uint8(randIntSlider(10, 30)),
		}

		for x := 0; x < width; x++ {
			y := startY + int(amplitude*math.Sin(float64(x)*frequency))
			if y >= 0 && y < height {
				for t := 0; t < 2; t++ {
					if y+t < height {
						img.Set(x, y+t, waveColor)
					}
				}
			}
		}
	}
}

func drawNoiseBackground(img *image.RGBA, width, height int) {
	bgR := uint8(randIntSlider(50, 150))
	bgG := uint8(randIntSlider(50, 150))
	bgB := uint8(randIntSlider(50, 150))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			noise := int8(randIntSlider(-30, 30))
			r := clampUint8(int(bgR) + int(noise))
			g := clampUint8(int(bgG) + int(noise))
			b := clampUint8(int(bgB) + int(noise))
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func drawGeometricBackground(img *image.RGBA, width, height int) {
	bgColor := color.RGBA{
		R: uint8(randIntSlider(60, 140)),
		G: uint8(randIntSlider(60, 140)),
		B: uint8(randIntSlider(60, 140)),
		A: 255,
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	for i := 0; i < 8; i++ {
		shapeType := randIntSlider(0, 2)
		x := randIntSlider(0, width)
		y := randIntSlider(0, height)
		size := randIntSlider(20, 60)
		shapeColor := color.RGBA{
			R: uint8(randIntSlider(150, 255)),
			G: uint8(randIntSlider(150, 255)),
			B: uint8(randIntSlider(150, 255)),
			A: uint8(randIntSlider(15, 40)),
		}

		switch shapeType {
		case 0:
			drawGeometricCircle(img, x, y, size, shapeColor)
		case 1:
			drawGeometricTriangle(img, x, y, size, shapeColor)
		case 2:
			drawGeometricRectangle(img, x, y, size, size/2, shapeColor)
		}
	}
}

func drawGeometricCircle(img *image.RGBA, cx, cy, r int, col color.RGBA) {
	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			if dx*dx+dy*dy <= r*r {
				px, py := cx+dx, cy+dy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, col)
				}
			}
		}
	}
}

func drawGeometricTriangle(img *image.RGBA, cx, cy, size int, col color.RGBA) {
	for i := 0; i < size; i++ {
		halfWidth := i * size / size
		for j := -halfWidth; j <= halfWidth; j++ {
			px, py := cx+j, cy-size/2+i
			if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
				img.Set(px, py, col)
			}
		}
	}
}

func drawGeometricRectangle(img *image.RGBA, cx, cy, w, h int, col color.RGBA) {
	for dx := -w / 2; dx < w/2; dx++ {
		for dy := -h / 2; dy < h/2; dy++ {
			px, py := cx+dx, cy+dy
			if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
				img.Set(px, py, col)
			}
		}
	}
}

func drawTexturedBackground(img *image.RGBA, width, height int) {
	bgR := uint8(randIntSlider(80, 160))
	bgG := uint8(randIntSlider(80, 160))
	bgB := uint8(randIntSlider(80, 160))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			texture := int(float64(x)*0.05*math.Sin(float64(y)*0.1)) +
				int(float64(y)*0.05*math.Cos(float64(x)*0.1))
			noise := randIntSlider(-20, 20)
			r := clampUint8(int(bgR) + texture + noise)
			g := clampUint8(int(bgG) + texture + noise)
			b := clampUint8(int(bgB) + texture + noise)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func addImageNoise(img *image.RGBA, width, height int) {
	noiseCount := randIntSlider(500, 1500)
	for i := 0; i < noiseCount; i++ {
		x := randIntSlider(0, width-1)
		y := randIntSlider(0, height-1)
		noiseColor := color.RGBA{
			R: uint8(randIntSlider(0, 255)),
			G: uint8(randIntSlider(0, 255)),
			B: uint8(randIntSlider(0, 255)),
			A: uint8(randIntSlider(5, 30)),
		}
		img.Set(x, y, noiseColor)
	}
}

func clampUint8(val int) uint8 {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}

func generatePuzzleMask(shape PuzzleShape, size int) [][]bool {
	mask := make([][]bool, size)
	for i := range mask {
		mask[i] = make([]bool, size)
	}

	center := size / 2

	switch shape {
	case ShapeSquare:
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				margin := size / 8
				if x >= margin && x < size-margin && y >= margin && y < size-margin {
					mask[y][x] = true
				}
			}
		}

	case ShapeCircle:
		radius := size / 2
		innerRadius := size / 4
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := x - center
				dy := y - center
				dist := math.Sqrt(float64(dx*dx + dy*dy))
				if dist >= float64(innerRadius) && dist <= float64(radius) {
					mask[y][x] = true
				}
			}
		}

	case ShapeTriangle:
		for y := 0; y < size; y++ {
			rowWidth := y * size / size
			startX := (size - rowWidth) / 2
			for x := 0; x < size; x++ {
				if x >= startX && x < startX+rowWidth && y < size*3/4 {
					mask[y][x] = true
				}
			}
		}

	case ShapeDiamond:
		for y := 0; y < size; y++ {
			distFromCenter := int(math.Abs(float64(y - center)))
			rowWidth := size - 2*distFromCenter
			startX := center - rowWidth/2
			for x := 0; x < size; x++ {
				if x >= startX && x < startX+rowWidth {
					mask[y][x] = true
				}
			}
		}

	case ShapeHexagon:
		radius := size / 2
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := math.Abs(float64(x - center))
				dy := math.Abs(float64(y - center))
				hexCondition := dx*0.866+dy*0.5 <= float64(radius)*0.866
				if hexCondition && dx <= float64(radius)*0.8 && dy <= float64(radius)*0.8 {
					mask[y][x] = true
				}
			}
		}

	default:
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				mask[y][x] = true
			}
		}
	}

	return mask
}

func cutPuzzleFromImage(bgImg image.Image, x, y, size int, shape PuzzleShape, isGlow bool) image.Image {
	mask := generatePuzzleMask(shape, size)
	puzzleImg := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(puzzleImg, puzzleImg.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			if mask[py][px] {
				srcX := x + px
				srcY := y + py

				if srcX >= 0 && srcX < bgImg.Bounds().Dx() && srcY >= 0 && srcY < bgImg.Bounds().Dy() {
					pixel := bgImg.At(srcX, srcY)
					puzzleImg.Set(px, py, pixel)
				}
			}
		}
	}

	if isGlow {
		addGlowEffect(puzzleImg, mask)
	}

	return puzzleImg
}

func addGlowEffect(img *image.RGBA, mask [][]bool) {
	size := img.Bounds().Dx()
	borderColor := color.RGBA{
		R: 255,
		G: 255,
		B: 255,
		A: 200,
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if !mask[y][x] {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < size && ny >= 0 && ny < size && mask[ny][nx] {
							img.Set(x, y, borderColor)
							break
						}
					}
				}
			}
		}
	}
}

func createHintImage(bgImg image.Image, puzzleImg image.Image, puzzleX, puzzleY, puzzleSize int, shape PuzzleShape) image.Image {
	width := bgImg.Bounds().Dx()
	height := bgImg.Bounds().Dy()
	hintImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(hintImg, hintImg.Bounds(), bgImg, image.Point{}, draw.Src)

	mask := generatePuzzleMask(shape, puzzleSize)
	borderWidth := 3
	borderColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	for py := 0; py < puzzleSize; py++ {
		for px := 0; px < puzzleSize; px++ {
			if mask[py][px] {
				for b := 0; b < borderWidth; b++ {
					isBorder := false
					if px-b < 0 || px-b >= puzzleSize || !mask[py][px-b] {
						isBorder = true
					}
					if px+b >= puzzleSize || px+b < 0 || !mask[py][px+b] {
						isBorder = true
					}
					if py-b < 0 || py-b >= puzzleSize || !mask[py-b][px] {
						isBorder = true
					}
					if py+b >= puzzleSize || py+b < 0 || !mask[py+b][px] {
						isBorder = true
					}

					if isBorder {
						hx, hy := puzzleX+px, puzzleY+py
						if hx >= 0 && hx < width && hy >= 0 && hy < height {
							hintImg.Set(hx, hy, borderColor)
						}
					}
				}
			}
		}
	}

	return hintImg
}

func createSlidingGapImage(bgImg image.Image, puzzleX, puzzleY, puzzleSize int, shape PuzzleShape) image.Image {
	width := bgImg.Bounds().Dx()
	height := bgImg.Bounds().Dy()
	gapImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(gapImg, gapImg.Bounds(), bgImg, image.Point{}, draw.Src)

	mask := generatePuzzleMask(shape, puzzleSize)
	gapPadding := 2

	for y := puzzleY - gapPadding; y < puzzleY+puzzleSize+gapPadding; y++ {
		for x := puzzleX - gapPadding; x < puzzleX+puzzleSize+gapPadding; x++ {
			isInsidePuzzle := false
			mx, my := x-puzzleX, y-puzzleY
			if mx >= 0 && mx < puzzleSize && my >= 0 && my < puzzleSize {
				isInsidePuzzle = mask[my][mx]
			}

			if x >= 0 && x < width && y >= 0 && y < height {
				if isInsidePuzzle {
					darkColor := color.RGBA{
						R: 30,
						G: 30,
						B: 30,
						A: 255,
					}
					gapImg.Set(x, y, darkColor)

					if x+1 < width {
						gapImg.Set(x+1, y, color.RGBA{R: 60, G: 60, B: 60, A: 255})
					}
					if y+1 < height {
						gapImg.Set(x, y+1, color.RGBA{R: 60, G: 60, B: 60, A: 255})
					}
				}
			}
		}
	}

	return gapImg
}

func imageToDataURL(img image.Image) string {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func GenerateSliderCaptcha(c *gin.Context) {
	var req GenerateSliderRequest
	if err := c.ShouldBind(&req); err != nil {
		req.Width = 0
		req.Height = 0
		req.Tolerance = 0
	}

	config := defaultSliderConfig
	if req.Width > 0 && req.Width <= 600 {
		config.Width = req.Width
	}
	if req.Height > 0 && req.Height <= 400 {
		config.Height = req.Height
	}

	tolerance := config.DefaultTolerance
	if req.Tolerance > 0 && req.Tolerance <= 20 {
		tolerance = req.Tolerance
	}

	sessionID := generateSliderSessionID()

	puzzleSize := config.PuzzleSize
	minX := puzzleSize + 10
	maxX := config.Width - puzzleSize - 10
	secretX := randIntSlider(minX, maxX)
	secretY := randIntSlider(10, config.Height-puzzleSize-10)

	shape := PuzzleShape(randIntSlider(0, 4))

	bgImg := generateSliderBackground(config.Width, config.Height)

	puzzleImg := cutPuzzleFromImage(bgImg, secretX, secretY, puzzleSize, shape, true)

	hintImg := createHintImage(bgImg, puzzleImg, secretX, secretY, puzzleSize, shape)

	gapImg := createSlidingGapImage(bgImg, secretX, secretY, puzzleSize, shape)

	session := &SliderSession{
		SessionID: sessionID,
		SecretX:   secretX,
		SecretY:   secretY,
		Tolerance: tolerance,
		Shape:     shape,
		BackgroundImage: &imageData{
			DataURL: imageToDataURL(gapImg),
			Width:   config.Width,
			Height:  config.Height,
		},
		PuzzleImage: &imageData{
			DataURL: imageToDataURL(puzzleImg),
			Width:   puzzleSize,
			Height:  puzzleSize,
		},
		HintImage: &imageData{
			DataURL: imageToDataURL(hintImg),
			Width:   config.Width,
			Height:  config.Height,
		},
		Attempts:  0,
		Verified:  false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(config.SessionTTL),
		ClientIP:  c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	sliderSessionMu.Lock()
	sliderSessionStore[sessionID] = session
	sliderSessionMu.Unlock()

	saveSliderSessionToRedis(session)

	c.JSON(http.StatusOK, GenerateSliderResponse{
		SessionID:   sessionID,
		ImageURL:    imageToDataURL(gapImg),
		PuzzleURL:   imageToDataURL(puzzleImg),
		HintURL:     imageToDataURL(hintImg),
		Shape:       int(shape),
		SecretY:     secretY,
		ImageWidth:  config.Width,
		ImageHeight: config.Height,
	})
}

func generateSliderSessionID() string {
	bytes, err := generateRandomBytes(16)
	if err != nil {
		return fmt.Sprintf("slider_%d_%d", time.Now().UnixNano(), randIntSlider(1000, 9999))
	}
	return fmt.Sprintf("slider_%x", bytes)
}

const (
	minTrajectoryPoints     = 3
	maxSingleStepDistance   = 50
	minAccelerationVariance = 0.5
	minYVariation           = 1
	maxStraightLineRatio    = 0.85
	maxTeleportRatio        = 0.6
)

func verifyTrajectory(points []TrajectoryPoint, totalDistance int) *TrajectoryResult {
	result := &TrajectoryResult{
		Score:   100,
		Passed:  true,
		Reasons: []string{},
	}

	if len(points) < minTrajectoryPoints {
		result.Score = 0
		result.Passed = false
		result.Reasons = append(result.Reasons, "轨迹点数量不足")
		return result
	}

	reasons := []string{}
	score := 100

	score, reasons = checkPointCount(points, score, reasons)
	score, reasons = checkTeleportation(points, score, reasons, totalDistance)
	score, reasons = checkYVariation(points, score, reasons)
	score, reasons = checkAccelerationConsistency(points, score, reasons)
	score, reasons = checkJitter(points, score, reasons)
	score, reasons = checkStraightLine(points, score, reasons)

	if score < 0 {
		score = 0
	}

	result.Score = score
	result.Passed = score >= 30
	result.Reasons = reasons
	return result
}

func checkPointCount(points []TrajectoryPoint, score int, reasons []string) (int, []string) {
	if len(points) < 5 {
		score -= 20
		reasons = append(reasons, "轨迹点较少")
	}
	if len(points) > 200 {
		score -= 10
		reasons = append(reasons, "轨迹点过多")
	}
	return score, reasons
}

func checkTeleportation(points []TrajectoryPoint, score int, reasons []string, totalDistance int) (int, []string) {
	maxStep := 0
	for i := 1; i < len(points); i++ {
		step := intAbs(points[i].X - points[i-1].X)
		if step > maxStep {
			maxStep = step
		}
	}

	if maxStep > maxSingleStepDistance {
		score -= 30
		reasons = append(reasons, "存在异常瞬移轨迹")
	}

	if totalDistance > 0 {
		teleportThreshold := int(float64(totalDistance) * maxTeleportRatio)
		if maxStep > teleportThreshold {
			score -= 20
			if len(reasons) == 0 || reasons[len(reasons)-1] != "存在异常瞬移轨迹" {
				reasons = append(reasons, "单步距离占比过大")
			}
		}
	}

	return score, reasons
}

func checkYVariation(points []TrajectoryPoint, score int, reasons []string) (int, []string) {
	if len(points) < 2 {
		return score, reasons
	}

	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	yRange := maxY - minY
	if yRange < minYVariation {
		score -= 25
		reasons = append(reasons, "Y轴无变化，疑似机器操作")
	}

	return score, reasons
}

func checkAccelerationConsistency(points []TrajectoryPoint, score int, reasons []string) (int, []string) {
	if len(points) < 4 {
		return score, reasons
	}

	velocities := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].T - points[i-1].T)
		if dt <= 0 {
			dt = 1
		}
		dx := float64(points[i].X - points[i-1].X)
		velocities = append(velocities, dx/dt)
	}

	if len(velocities) < 2 {
		return score, reasons
	}

	accelerations := make([]float64, 0, len(velocities)-1)
	for i := 1; i < len(velocities); i++ {
		accelerations = append(accelerations, velocities[i]-velocities[i-1])
	}

	if len(accelerations) == 0 {
		return score, reasons
	}

	mean := 0.0
	for _, a := range accelerations {
		mean += a
	}
	mean /= float64(len(accelerations))

	variance := 0.0
	for _, a := range accelerations {
		diff := a - mean
		variance += diff * diff
	}
	variance /= float64(len(accelerations))

	if variance < minAccelerationVariance {
		score -= 20
		reasons = append(reasons, "加速度变化异常，疑似机器操作")
	}

	return score, reasons
}

func checkJitter(points []TrajectoryPoint, score int, reasons []string) (int, []string) {
	if len(points) < 6 {
		return score, reasons
	}

	jitterCount := 0
	for i := 2; i < len(points); i++ {
		prevDir := points[i-1].X - points[i-2].X
		currDir := points[i].X - points[i-1].X
		if prevDir > 0 && currDir < 0 || prevDir < 0 && currDir > 0 {
			jitterCount++
		}
	}

	if jitterCount == 0 {
		score -= 15
		reasons = append(reasons, "轨迹过于平滑，无自然抖动")
	}

	return score, reasons
}

func checkStraightLine(points []TrajectoryPoint, score int, reasons []string) (int, []string) {
	if len(points) < 3 {
		return score, reasons
	}

	straightSegments := 0
	totalSegments := len(points) - 1

	for i := 1; i < len(points); i++ {
		if points[i].Y == points[i-1].Y {
			straightSegments++
		}
	}

	straightRatio := float64(straightSegments) / float64(totalSegments)
	if straightRatio > maxStraightLineRatio {
		score -= 20
		reasons = append(reasons, "轨迹近似直线，疑似机器操作")
	}

	return score, reasons
}

func saveSliderSessionToRedis(session *SliderSession) {
	if redis.Client == nil {
		return
	}

	ctx := context.Background()
	data, err := json.Marshal(session)
	if err != nil {
		return
	}

	key := fmt.Sprintf("slider_session:%s", session.SessionID)
	redis.Client.Set(ctx, key, data, defaultSliderConfig.SessionTTL)
}

func getSliderSessionFromRedis(sessionID string) (*SliderSession, bool) {
	if redis.Client == nil {
		sliderSessionMu.RLock()
		defer sliderSessionMu.RUnlock()
		session, exists := sliderSessionStore[sessionID]
		return session, exists
	}

	ctx := context.Background()
	key := fmt.Sprintf("slider_session:%s", sessionID)
	data, err := redis.Client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var session SliderSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, false
	}

	return &session, true
}

func deleteSliderSessionFromRedis(sessionID string) {
	if redis.Client == nil {
		sliderSessionMu.Lock()
		delete(sliderSessionStore, sessionID)
		sliderSessionMu.Unlock()
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("slider_session:%s", sessionID)
	redis.Client.Del(ctx, key)
}

func VerifySliderCaptcha(c *gin.Context) {
	var req VerifySliderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	session, exists := getSliderSessionFromRedis(req.SessionID)
	if !exists {
		response.NotFound(c, "验证码会话不存在或已过期")
		return
	}

	if time.Now().After(session.ExpiresAt) {
		deleteSliderSessionFromRedis(req.SessionID)
		response.NotFound(c, "验证码已过期，请重新获取")
		return
	}

	if session.Verified {
		response.BadRequest(c, "验证码已验证通过")
		return
	}

	session.Attempts++
	if session.Attempts > defaultSliderConfig.MaxAttempts {
		deleteSliderSessionFromRedis(req.SessionID)
		response.BadRequest(c, "验证次数过多，请重新获取验证码")
		return
	}

	if req.Y == 0 {
		req.Y = session.SecretY
	}

	tolerance := session.Tolerance
	if tolerance <= 0 {
		tolerance = defaultSliderConfig.DefaultTolerance
	}

	distance := intAbs(req.X - session.SecretX)
	yDistance := intAbs(req.Y - session.SecretY)

	var trajResult *TrajectoryResult
	var trajectory []TrajectoryPoint

	if req.EncryptedTrajectory != nil {
		encryptor := getTrajectoryEncryptor()

		if encryptor.ShouldRotate() {
			encryptor.RotateSalt()
		}

		decryptedTrajectory, err := encryptor.DecryptTrajectory(req.EncryptedTrajectory)
		if err != nil {
			remaining := defaultSliderConfig.MaxAttempts - session.Attempts
			response.Success(c, VerifySliderResponse{
				Success:          false,
				Message:          "轨迹解密失败: " + err.Error(),
				Remaining:        remaining,
				TrajectoryResult: nil,
			})
			return
		}
		trajectory = decryptedTrajectory
	} else if len(req.Trajectory) > 0 {
		trajectory = req.Trajectory
	}

	if len(trajectory) > 0 {
		trajResult = verifyTrajectory(trajectory, distance)
	}

	if distance <= tolerance && yDistance <= tolerance {
		if trajResult != nil && !trajResult.Passed {
			remaining := defaultSliderConfig.MaxAttempts - session.Attempts
			response.Success(c, VerifySliderResponse{
				Success:          false,
				Message:          "位置正确但轨迹异常，请使用自然手势滑动",
				Remaining:        remaining,
				TrajectoryResult: trajResult,
			})
			return
		}

		session.Verified = true
		deleteSliderSessionFromRedis(req.SessionID)

		response.Success(c, VerifySliderResponse{
			Success:          true,
			Message:          "验证成功",
			Remaining:        defaultSliderConfig.MaxAttempts - session.Attempts,
			TrajectoryResult: trajResult,
		})
		return
	}

	remaining := defaultSliderConfig.MaxAttempts - session.Attempts
	if remaining <= 0 {
		deleteSliderSessionFromRedis(req.SessionID)
		response.BadRequest(c, "验证次数已用完，请重新获取验证码")
		return
	}

	accuracy := 100 - (distance * 100 / (defaultSliderConfig.Width / 2))
	response.Success(c, VerifySliderResponse{
		Success:          false,
		Message:          fmt.Sprintf("位置偏差较大，准确度约%d%%", accuracy),
		Remaining:        remaining,
		TrajectoryResult: trajResult,
	})
}

func GetSliderCaptchaV2(c *gin.Context) {
	var req GenerateSliderRequest
	if err := c.ShouldBind(&req); err != nil {
		req.Width = 0
		req.Height = 0
		req.Tolerance = 0
	}

	config := defaultSliderConfig
	if req.Width > 0 && req.Width <= 600 {
		config.Width = req.Width
	}
	if req.Height > 0 && req.Height <= 400 {
		config.Height = req.Height
	}

	tolerance := config.DefaultTolerance
	if req.Tolerance > 0 && req.Tolerance <= 20 {
		tolerance = req.Tolerance
	}

	sessionID := generateSliderSessionID()

	puzzleSize := config.PuzzleSize
	minX := puzzleSize + 10
	maxX := config.Width - puzzleSize - 10
	secretX := randIntSlider(minX, maxX)
	secretY := randIntSlider(10, config.Height-puzzleSize-10)

	shape := PuzzleShape(randIntSlider(0, 4))

	bgImg := generateSliderBackground(config.Width, config.Height)

	puzzleImg := cutPuzzleFromImage(bgImg, secretX, secretY, puzzleSize, shape, true)

	hintImg := createHintImage(bgImg, puzzleImg, secretX, secretY, puzzleSize, shape)

	gapImg := createSlidingGapImage(bgImg, secretX, secretY, puzzleSize, shape)

	session := &SliderSession{
		SessionID: sessionID,
		SecretX:   secretX,
		SecretY:   secretY,
		Tolerance: tolerance,
		Shape:     shape,
		BackgroundImage: &imageData{
			DataURL: imageToDataURL(gapImg),
			Width:   config.Width,
			Height:  config.Height,
		},
		PuzzleImage: &imageData{
			DataURL: imageToDataURL(puzzleImg),
			Width:   puzzleSize,
			Height:  puzzleSize,
		},
		HintImage: &imageData{
			DataURL: imageToDataURL(hintImg),
			Width:   config.Width,
			Height:  config.Height,
		},
		Attempts:  0,
		Verified:  false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(config.SessionTTL),
		ClientIP:  c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	sliderSessionMu.Lock()
	sliderSessionStore[sessionID] = session
	sliderSessionMu.Unlock()

	saveSliderSessionToRedis(session)

	c.JSON(http.StatusOK, gin.H{
		"session_id":   sessionID,
		"image_url":    imageToDataURL(gapImg),
		"puzzle_url":   imageToDataURL(puzzleImg),
		"hint_url":     imageToDataURL(hintImg),
		"shape":        int(shape),
		"secret_y":     secretY,
		"image_width":  config.Width,
		"image_height": config.Height,
	})
}

func TestImageConversion(c *gin.Context) {
	width := 320
	height := 160
	img := generateSliderBackground(width, height)

	puzzleSize := 40
	secretX := 100
	secretY := 60
	shape := ShapeCircle

	puzzleImg := cutPuzzleFromImage(img, secretX, secretY, puzzleSize, shape, true)
	hintImg := createHintImage(img, puzzleImg, secretX, secretY, puzzleSize, shape)
	gapImg := createSlidingGapImage(img, secretX, secretY, puzzleSize, shape)

	c.JSON(http.StatusOK, gin.H{
		"background": imageToDataURL(img),
		"puzzle":     imageToDataURL(puzzleImg),
		"hint":       imageToDataURL(hintImg),
		"gap":        imageToDataURL(gapImg),
	})
}

func TestPuzzleMask(c *gin.Context) {
	size := 40

	shapes := []PuzzleShape{ShapeSquare, ShapeCircle, ShapeTriangle, ShapeDiamond, ShapeHexagon}
	results := make(map[string]string)

	for _, shape := range shapes {
		mask := generatePuzzleMask(shape, size)
		maskImg := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.Draw(maskImg, maskImg.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if mask[y][x] {
					maskImg.Set(x, y, color.RGBA{255, 255, 255, 255})
				}
			}
		}

		shapeName := []string{"square", "circle", "triangle", "diamond", "hexagon"}[shape]
		results[shapeName] = imageToDataURL(maskImg)
	}

	c.JSON(http.StatusOK, results)
}

func GetSliderStatus(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		response.BadRequest(c, "缺少session_id参数")
		return
	}

	session, exists := getSliderSessionFromRedis(sessionID)
	if !exists {
		response.NotFound(c, "会话不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":   session.SessionID,
		"attempts":     session.Attempts,
		"max_attempts": defaultSliderConfig.MaxAttempts,
		"verified":     session.Verified,
		"remaining":    defaultSliderConfig.MaxAttempts - session.Attempts,
		"expires_at":   session.ExpiresAt.Unix(),
		"created_at":   session.CreatedAt.Unix(),
	})
}

func DeleteSliderSession(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		response.BadRequest(c, "缺少session_id参数")
		return
	}

	deleteSliderSessionFromRedis(sessionID)
	response.Success(c, gin.H{"message": "会话已删除"})
}
