package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGenerateSliderCaptcha(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptchaV2)

	sliderSessionStore = make(map[string]*SliderSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotEmpty(t, resp["session_id"])
	assert.NotEmpty(t, resp["image_url"])
	assert.NotEmpty(t, resp["puzzle_url"])
	assert.NotEmpty(t, resp["hint_url"])

	sessionID := resp["session_id"].(string)
	sliderSessionMu.RLock()
	session, exists := sliderSessionStore[sessionID]
	sliderSessionMu.RUnlock()
	assert.True(t, exists)
	assert.NotNil(t, session)
	assert.False(t, session.Verified)
	assert.Equal(t, 0, session.Attempts)
}

func TestGeneratePuzzleMask(t *testing.T) {
	size := 40

	shapes := []PuzzleShape{ShapeSquare, ShapeCircle, ShapeTriangle, ShapeDiamond, ShapeHexagon}

	for _, shape := range shapes {
		mask := generatePuzzleMask(shape, size)
		assert.Len(t, mask, size)

		hasTrue := false
		hasFalse := false
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if mask[y][x] {
					hasTrue = true
				} else {
					hasFalse = true
				}
			}
		}
		assert.True(t, hasTrue, "mask should have true values")
		assert.True(t, hasFalse, "mask should have false values")
	}
}

func TestCutPuzzleFromImage(t *testing.T) {
	width := 320
	height := 160
	size := 40
	x := 100
	y := 60

	bgImg := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			bgImg.Set(px, py, color.RGBA{
				R: uint8(px % 256),
				G: uint8(py % 256),
				B: uint8((px + py) % 256),
				A: 255,
			})
		}
	}

	puzzleImg := cutPuzzleFromImage(bgImg, x, y, size, ShapeCircle, false)
	assert.NotNil(t, puzzleImg)
	assert.Equal(t, size, puzzleImg.Bounds().Dx())
	assert.Equal(t, size, puzzleImg.Bounds().Dy())
}

func TestCreateHintImage(t *testing.T) {
	width := 320
	height := 160
	puzzleSize := 40
	secretX := 100
	secretY := 60

	bgImg := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			bgImg.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	puzzleImg := image.NewRGBA(image.Rect(0, 0, puzzleSize, puzzleSize))
	draw.Draw(puzzleImg, puzzleImg.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	hintImg := createHintImage(bgImg, puzzleImg, secretX, secretY, puzzleSize, ShapeCircle)
	assert.NotNil(t, hintImg)
	assert.Equal(t, width, hintImg.Bounds().Dx())
	assert.Equal(t, height, hintImg.Bounds().Dy())
}

func TestCreateSlidingGapImage(t *testing.T) {
	width := 320
	height := 160
	puzzleSize := 40
	secretX := 100
	secretY := 60

	bgImg := image.NewRGBA(image.Rect(0, 0, width, height))
	bgColor := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	draw.Draw(bgImg, bgImg.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	gapImg := createSlidingGapImage(bgImg, secretX, secretY, puzzleSize, ShapeSquare)
	assert.NotNil(t, gapImg)
	assert.Equal(t, width, gapImg.Bounds().Dx())
	assert.Equal(t, height, gapImg.Bounds().Dy())

	darkPixels := 0
	for y := secretY - 2; y < secretY+puzzleSize+2; y++ {
		for x := secretX - 2; x < secretX+puzzleSize+2; x++ {
			if x >= 0 && x < width && y >= 0 && y < height {
				r, _, _, _ := gapImg.At(x, y).RGBA()
				if r < 0x4000 {
					darkPixels++
				}
			}
		}
	}
	assert.Greater(t, darkPixels, 0, "gap image should contain dark area")
}

func TestGenerateSliderBackground(t *testing.T) {
	width := 320
	height := 160

	img := generateSliderBackground(width, height)
	assert.NotNil(t, img)
	assert.Equal(t, width, img.Bounds().Dx())
	assert.Equal(t, height, img.Bounds().Dy())

	var coloredPixels int
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				coloredPixels++
			}
		}
	}
	assert.Equal(t, width*height, coloredPixels)
}

func TestImageToDataURL(t *testing.T) {
	width := 100
	height := 50

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	dataURL := imageToDataURL(img)
	assert.NotEmpty(t, dataURL)
	assert.Contains(t, dataURL, "data:image/png;base64,")

	pngData, err := decodeBase64PNG(strings.TrimPrefix(dataURL, "data:image/png;base64,"))
	assert.NoError(t, err)
	assert.NotNil(t, pngData)
	assert.Equal(t, width, pngData.Bounds().Dx())
	assert.Equal(t, height, pngData.Bounds().Dy())
}

func decodeBase64PNG(encoded string) (image.Image, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(data)
	img, err := png.Decode(reader)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func TestGenerateSliderSessionID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateSliderSessionID()
		assert.NotEmpty(t, id)
		assert.True(t, strings.HasPrefix(id, "slider_"))
		assert.False(t, ids[id], "session ID should be unique")
		ids[id] = true
	}
}

func TestSliderSessionStorage(t *testing.T) {
	sliderSessionStore = make(map[string]*SliderSession)

	sessionID := "test-storage-session"
	session := &SliderSession{
		SessionID: sessionID,
		SecretX:   150,
		SecretY:   60,
		Tolerance: 8,
		Shape:     ShapeCircle,
		Attempts:  0,
		Verified:  false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	sliderSessionMu.Lock()
	sliderSessionStore[sessionID] = session
	sliderSessionMu.Unlock()

	sliderSessionMu.RLock()
	retrieved, exists := sliderSessionStore[sessionID]
	sliderSessionMu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, session.SessionID, retrieved.SessionID)
	assert.Equal(t, session.SecretX, retrieved.SecretX)
	assert.Equal(t, session.SecretY, retrieved.SecretY)
	assert.Equal(t, session.Tolerance, retrieved.Tolerance)
}

func TestIntAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{100, 100},
		{-100, 100},
	}

	for _, tt := range tests {
		result := intAbs(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestClampUint8(t *testing.T) {
	tests := []struct {
		input    int
		expected uint8
	}{
		{0, 0},
		{128, 128},
		{255, 255},
		{-10, 0},
		{-100, 0},
		{300, 255},
		{500, 255},
	}

	for _, tt := range tests {
		result := clampUint8(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestRandIntSlider(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := randIntSlider(5, 15)
		assert.GreaterOrEqual(t, result, 5)
		assert.LessOrEqual(t, result, 15)
	}
}

func TestSliderCaptchaMultipleShapes(t *testing.T) {
	width := 320
	height := 160
	size := 40
	x := 100
	y := 60

	bgImg := generateSliderBackground(width, height)

	shapes := []PuzzleShape{ShapeSquare, ShapeCircle, ShapeTriangle, ShapeDiamond, ShapeHexagon}

	for _, shape := range shapes {
		puzzleImg := cutPuzzleFromImage(bgImg, x, y, size, shape, true)
		assert.NotNil(t, puzzleImg)

		hintImg := createHintImage(bgImg, puzzleImg, x, y, size, shape)
		assert.NotNil(t, hintImg)

		gapImg := createSlidingGapImage(bgImg, x, y, size, shape)
		assert.NotNil(t, gapImg)

		puzzleDataURL := imageToDataURL(puzzleImg)
		hintDataURL := imageToDataURL(hintImg)
		gapDataURL := imageToDataURL(gapImg)

		assert.NotEmpty(t, puzzleDataURL)
		assert.NotEmpty(t, hintDataURL)
		assert.NotEmpty(t, gapDataURL)
	}
}

func TestSliderCaptchaEdgeCases(t *testing.T) {
	t.Run("min size background", func(t *testing.T) {
		img := generateSliderBackground(100, 50)
		assert.NotNil(t, img)
		assert.Equal(t, 100, img.Bounds().Dx())
		assert.Equal(t, 50, img.Bounds().Dy())
	})

	t.Run("max size background", func(t *testing.T) {
		img := generateSliderBackground(600, 400)
		assert.NotNil(t, img)
		assert.Equal(t, 600, img.Bounds().Dx())
		assert.Equal(t, 400, img.Bounds().Dy())
	})

	t.Run("boundary puzzle position", func(t *testing.T) {
		width := 320
		height := 160
		size := 40

		bgImg := generateSliderBackground(width, height)

		puzzleImg := cutPuzzleFromImage(bgImg, 10, 10, size, ShapeSquare, false)
		assert.NotNil(t, puzzleImg)

		puzzleImg = cutPuzzleFromImage(bgImg, width-size-10, height-size-10, size, ShapeSquare, false)
		assert.NotNil(t, puzzleImg)
	})
}

func TestGetSliderStatus(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/captcha/slider/status", GetSliderStatus)

	sliderSessionStore = make(map[string]*SliderSession)

	sessionID := "test-status-session"
	session := &SliderSession{
		SessionID: sessionID,
		SecretX:   150,
		SecretY:   60,
		Tolerance: 8,
		Shape:     ShapeCircle,
		Attempts:  2,
		Verified:  false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	sliderSessionStore[sessionID] = session

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider/status?session_id="+sessionID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, sessionID, resp["session_id"])
	assert.Equal(t, float64(2), resp["attempts"])
	assert.Equal(t, float64(defaultSliderConfig.MaxAttempts), resp["max_attempts"])
	assert.Equal(t, float64(defaultSliderConfig.MaxAttempts-2), resp["remaining"])
}

func TestDeleteSliderSession(t *testing.T) {
	r := gin.New()
	r.DELETE("/api/v1/captcha/slider", DeleteSliderSession)

	sliderSessionStore = make(map[string]*SliderSession)

	sessionID := "test-delete-session"
	session := &SliderSession{
		SessionID: sessionID,
		SecretX:   150,
		SecretY:   60,
		Tolerance: 8,
		Shape:     ShapeCircle,
		Attempts:  0,
		Verified:  false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	sliderSessionStore[sessionID] = session

	req, _ := http.NewRequest("DELETE", "/api/v1/captcha/slider?session_id="+sessionID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	sliderSessionMu.RLock()
	_, exists := sliderSessionStore[sessionID]
	sliderSessionMu.RUnlock()
	assert.False(t, exists, "session should be deleted")
}
