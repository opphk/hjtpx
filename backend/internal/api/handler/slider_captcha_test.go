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
	"github.com/hjtpx/hjtpx/pkg/response"
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

func generateHumanLikeTrajectory(startX, endX, startY, endY, numPoints int) []TrajectoryPoint {
	points := make([]TrajectoryPoint, numPoints)
	currentTime := int64(0)

	for i := 0; i < numPoints; i++ {
		progress := float64(i) / float64(numPoints-1)

		easedProgress := progress * progress * (3 - 2*progress)

		x := startX + int(float64(endX-startX)*easedProgress)
		y := startY + int(float64(endY-startY)*easedProgress)

		if i > 0 && i < numPoints-1 {
			x += randIntSlider(-2, 2)
			y += randIntSlider(-1, 1)
		}

		currentTime += int64(randIntSlider(8, 25))
		points[i] = TrajectoryPoint{X: x, Y: y, T: currentTime}
	}

	return points
}

func generateBotLikeTrajectory(startX, endX, startY, numPoints int) []TrajectoryPoint {
	points := make([]TrajectoryPoint, numPoints)
	currentTime := int64(0)
	step := (endX - startX) / numPoints

	for i := 0; i < numPoints; i++ {
		x := startX + step*i
		if x > endX {
			x = endX
		}
		currentTime += 10
		points[i] = TrajectoryPoint{X: x, Y: startY, T: currentTime}
	}

	return points
}

func generateTeleportTrajectory(startX, endX, startY, numPoints int) []TrajectoryPoint {
	points := make([]TrajectoryPoint, numPoints)
	currentTime := int64(0)

	for i := 0; i < numPoints; i++ {
		if i == numPoints/2 {
			points[i] = TrajectoryPoint{X: endX, Y: startY, T: currentTime + 10}
		} else {
			x := startX + (endX-startX)*i/numPoints
			currentTime += 10
			points[i] = TrajectoryPoint{X: x, Y: startY, T: currentTime}
		}
		currentTime += 10
	}

	return points
}

func TestVerifyTrajectory_HumanLike(t *testing.T) {
	points := generateHumanLikeTrajectory(0, 150, 50, 55, 30)
	result := verifyTrajectory(points, 150)

	assert.True(t, result.Passed, "human-like trajectory should pass, got score=%d reasons=%v", result.Score, result.Reasons)
	assert.GreaterOrEqual(t, result.Score, 30)
}

func TestVerifyTrajectory_BotLike(t *testing.T) {
	points := generateBotLikeTrajectory(0, 150, 50, 20)
	result := verifyTrajectory(points, 150)

	assert.False(t, result.Passed, "bot-like trajectory should not pass, got score=%d reasons=%v", result.Score, result.Reasons)
	assert.Less(t, result.Score, 30)
}

func TestVerifyTrajectory_Teleport(t *testing.T) {
	points := generateTeleportTrajectory(0, 150, 50, 10)
	result := verifyTrajectory(points, 150)

	assert.False(t, result.Passed, "teleport trajectory should not pass, got score=%d reasons=%v", result.Score, result.Reasons)
}

func TestVerifyTrajectory_TooFewPoints(t *testing.T) {
	points := []TrajectoryPoint{
		{X: 0, Y: 50, T: 0},
		{X: 100, Y: 50, T: 100},
	}
	result := verifyTrajectory(points, 100)

	assert.False(t, result.Passed)
	assert.Equal(t, 0, result.Score)
	assert.Contains(t, result.Reasons[0], "轨迹点数量不足")
}

func TestVerifyTrajectory_NoTrajectory(t *testing.T) {
	result := verifyTrajectory([]TrajectoryPoint{}, 0)

	assert.False(t, result.Passed)
	assert.Equal(t, 0, result.Score)
}

func TestVerifyTrajectory_NoYVariation(t *testing.T) {
	points := make([]TrajectoryPoint, 10)
	for i := 0; i < 10; i++ {
		points[i] = TrajectoryPoint{X: i * 15, Y: 50, T: int64(i * 10)}
	}
	result := verifyTrajectory(points, 150)

	assert.False(t, result.Passed, "no Y variation should fail")
	hasYReason := false
	for _, r := range result.Reasons {
		if strings.Contains(r, "Y轴无变化") {
			hasYReason = true
			break
		}
	}
	assert.True(t, hasYReason, "should contain Y-axis reason")
}

func TestVerifyTrajectory_StraightLine(t *testing.T) {
	points := make([]TrajectoryPoint, 15)
	for i := 0; i < 15; i++ {
		points[i] = TrajectoryPoint{X: i * 10, Y: 50, T: int64(i * 10)}
	}
	result := verifyTrajectory(points, 150)

	assert.False(t, result.Passed, "straight line trajectory should fail")
}

func TestVerifyTrajectory_SmoothNoJitter(t *testing.T) {
	points := make([]TrajectoryPoint, 10)
	for i := 0; i < 10; i++ {
		x := i * 15
		y := 50 + i%3
		points[i] = TrajectoryPoint{X: x, Y: y, T: int64(i * 10)}
	}
	result := verifyTrajectory(points, 150)

	hasJitterReason := false
	for _, r := range result.Reasons {
		if strings.Contains(r, "无自然抖动") {
			hasJitterReason = true
			break
		}
	}
	assert.True(t, hasJitterReason, "smooth trajectory should have jitter reason")
}

func TestVerifyTrajectory_ConstantAcceleration(t *testing.T) {
	points := make([]TrajectoryPoint, 10)
	for i := 0; i < 10; i++ {
		x := i * 15
		y := 50 + i%2
		points[i] = TrajectoryPoint{X: x, Y: y, T: int64(i * 10)}
	}
	result := verifyTrajectory(points, 150)

	hasAccelReason := false
	for _, r := range result.Reasons {
		if strings.Contains(r, "加速度变化异常") {
			hasAccelReason = true
			break
		}
	}
	assert.True(t, hasAccelReason, "constant acceleration should be detected")
}

func TestVerifyTrajectory_EdgeCases(t *testing.T) {
	t.Run("single point", func(t *testing.T) {
		points := []TrajectoryPoint{{X: 0, Y: 50, T: 0}}
		result := verifyTrajectory(points, 0)
		assert.False(t, result.Passed)
		assert.Equal(t, 0, result.Score)
	})

	t.Run("exactly 3 points with variation", func(t *testing.T) {
		points := []TrajectoryPoint{
			{X: 0, Y: 50, T: 0},
			{X: 75, Y: 52, T: 100},
			{X: 150, Y: 51, T: 200},
		}
		result := verifyTrajectory(points, 150)
		assert.NotNil(t, result)
	})

	t.Run("many points human-like", func(t *testing.T) {
		points := generateHumanLikeTrajectory(0, 200, 50, 55, 100)
		result := verifyTrajectory(points, 200)
		assert.True(t, result.Passed, "human-like with many points should pass, score=%d", result.Score)
	})
}

func TestVerifySliderCaptcha_WithTrajectory(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/slider/verify", VerifySliderCaptcha)

	sliderSessionStore = make(map[string]*SliderSession)

	sessionID := "test-traj-session"
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

	t.Run("correct position with human trajectory", func(t *testing.T) {
		traj := generateHumanLikeTrajectory(0, 150, 60, 62, 30)
		body := VerifySliderRequest{
			SessionID:  sessionID,
			X:          150,
			Y:          60,
			Trajectory: traj,
		}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/verify", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var apiResp response.Response
		err := json.Unmarshal(w.Body.Bytes(), &apiResp)
		assert.NoError(t, err)

		dataJSON, _ := json.Marshal(apiResp.Data)
		var resp VerifySliderResponse
		err = json.Unmarshal(dataJSON, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success, "human trajectory with correct position should succeed, msg=%s", resp.Message)
		assert.NotNil(t, resp.TrajectoryResult)
		assert.True(t, resp.TrajectoryResult.Passed, "trajectory should pass, score=%d reasons=%v", resp.TrajectoryResult.Score, resp.TrajectoryResult.Reasons)
	})

	t.Run("correct position with bot trajectory", func(t *testing.T) {
		sessionID2 := "test-traj-session-2"
		session2 := &SliderSession{
			SessionID: sessionID2,
			SecretX:   150,
			SecretY:   60,
			Tolerance: 8,
			Shape:     ShapeCircle,
			Attempts:  0,
			Verified:  false,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		sliderSessionStore[sessionID2] = session2

		traj := generateBotLikeTrajectory(0, 150, 60, 15)
		body := VerifySliderRequest{
			SessionID:  sessionID2,
			X:          150,
			Y:          60,
			Trajectory: traj,
		}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/verify", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var apiResp response.Response
		err := json.Unmarshal(w.Body.Bytes(), &apiResp)
		assert.NoError(t, err)

		dataJSON, _ := json.Marshal(apiResp.Data)
		var resp VerifySliderResponse
		err = json.Unmarshal(dataJSON, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success, "bot trajectory should be rejected even with correct position")
		assert.NotNil(t, resp.TrajectoryResult)
		assert.False(t, resp.TrajectoryResult.Passed)
	})
}

func TestSliderSessionExpiry(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/slider/verify", VerifySliderCaptcha)

	sliderSessionStore = make(map[string]*SliderSession)

	sessionID := "test-expired-session"
	session := &SliderSession{
		SessionID: sessionID,
		SecretX:   150,
		SecretY:   60,
		Tolerance: 8,
		Shape:     ShapeCircle,
		Attempts:  0,
		Verified:  false,
		CreatedAt: time.Now().Add(-10 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	sliderSessionStore[sessionID] = session

	body := VerifySliderRequest{
		SessionID: sessionID,
		X:         150,
		Y:         60,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/verify", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	sliderSessionMu.RLock()
	_, exists := sliderSessionStore[sessionID]
	sliderSessionMu.RUnlock()
	assert.False(t, exists, "expired session should be deleted")
}
