package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGenerateSessionID(t *testing.T) {
	sessionID1 := generateSessionID()
	sessionID2 := generateSessionID()

	assert.NotEmpty(t, sessionID1)
	assert.NotEmpty(t, sessionID2)
	assert.NotEqual(t, sessionID1, sessionID2)
	assert.Contains(t, sessionID1, "sess_")
}

func TestShuffleInts(t *testing.T) {
	original := []int{1, 2, 3, 4, 5}
	shuffled := shuffleInts(original)

	assert.Len(t, shuffled, len(original))
	assert.ElementsMatch(t, original, shuffled)
}

func TestClampValue(t *testing.T) {
	tests := []struct {
		name     string
		val      int
		min      int
		max      int
		expected int
	}{
		{"below min", 5, 10, 20, 10},
		{"above max", 25, 10, 20, 20},
		{"in range", 15, 10, 20, 15},
		{"equal min", 10, 10, 20, 10},
		{"equal max", 20, 10, 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampValue(tt.val, tt.min, tt.max)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"positive", 5, 5},
		{"negative", -5, 5},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intAbs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImageToBase64(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	result := imageToBase64(img)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "iVBOR") // PNG base64 header
}

func TestClampU8(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected uint8
	}{
		{"negative", -5, 0},
		{"zero", 0, 0},
		{"mid", 128, 128},
		{"max", 255, 255},
		{"overflow", 300, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampU8(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsOverlapping(t *testing.T) {
	placed := []image.Rectangle{
		image.Rect(0, 0, 20, 20),
	}

	assert.True(t, isOverlapping(5, 5, 15, placed))
	assert.False(t, isOverlapping(50, 50, 10, placed))
	assert.True(t, isOverlapping(10, 10, 20, placed))
}

func TestGetCharForIndex(t *testing.T) {
	modes := []CaptchaMode{ModeNumber, ModeLetter, ModeChinese, ModeIcon, ModeMixed}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			for i := 0; i < 20; i++ {
				char := getCharForIndex(i, mode)
				assert.NotEmpty(t, char)
			}
		})
	}
}

func TestGetIconName(t *testing.T) {
	assert.Equal(t, "圆形", getIconName("circle"))
	assert.Equal(t, "方形", getIconName("square"))
	assert.Equal(t, "星形", getIconName("star"))
	assert.Equal(t, "未知图标", getIconName("unknown_icon"))
}

func TestFormatHintOrder(t *testing.T) {
	tests := []struct {
		name     string
		order    []int
		expected string
	}{
		{"empty", []int{}, ""},
		{"single", []int{0}, "1"},
		{"multiple", []int{0, 1, 2}, "1→2→3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHintOrder(tt.order)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzeEnvironmentData(t *testing.T) {
	tests := []struct {
		name     string
		envData  map[string]interface{}
		minScore float64
	}{
		{
			name:     "empty data",
			envData:  map[string]interface{}{},
			minScore: 0,
		},
		{
			name: "high risk webdriver",
			envData: map[string]interface{}{
				"webdriver": "wd:true",
			},
			minScore: 30,
		},
		{
			name: "software renderer",
			envData: map[string]interface{}{
				"webgl": "SwiftShader",
			},
			minScore: 25,
		},
		{
			name: "no webgl",
			envData: map[string]interface{}{
				"webgl": "no_webgl",
			},
			minScore: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzeEnvironmentData(tt.envData)
			assert.GreaterOrEqual(t, score, tt.minScore)
		})
	}
}

func TestGetSliderCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Contains(t, resp, "session_id")
	assert.Contains(t, resp, "image_url")
	assert.Contains(t, resp, "puzzle_image")
	assert.Contains(t, resp, "target_x")
	assert.Contains(t, resp, "target_y")
}

func TestGetClickCaptcha_Modes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	modes := []string{"number", "letter", "chinese", "icon", "mixed"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode="+mode, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "session_id")
			assert.Contains(t, resp, "image_url")
			assert.Contains(t, resp, "hint")
		})
	}
}

func TestGetClickCaptcha_ShuffleAndPoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	tests := []struct {
		name    string
		shuffle string
		points  string
	}{
		{"shuffle true", "true", "4"},
		{"shuffle false", "false", "5"},
		{"default points", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/captcha/click"
			if tt.shuffle != "" || tt.points != "" {
				url += "?"
				if tt.shuffle != "" {
					url += "shuffle=" + tt.shuffle
				}
				if tt.points != "" {
					if tt.shuffle != "" {
						url += "&"
					}
					url += "points=" + tt.points
				}
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestVerifyCaptcha_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyCaptcha_SessionNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	verifyReq := VerifyRequest{
		SessionID: "nonexistent-session",
		Type:      "slider",
		X:         100,
		Y:         50,
	}

	jsonBody, _ := json.Marshal(verifyReq)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVerifyCaptcha_TypeMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	getReq, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	var createResp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &createResp)
	sessionID := createResp["session_id"].(string)

	verifyReq := VerifyRequest{
		SessionID: sessionID,
		Type:      "click",
		Points:    [][2]int{{100, 100}},
	}

	jsonBody, _ := json.Marshal(verifyReq)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyClickPoints_NoPoints(t *testing.T) {
	session := &CaptchaSession{
		MaxPoints: 3,
	}

	req := VerifyRequest{
		Points: [][2]int{},
	}

	success, reason := verifyClickPoints(session, req)
	assert.False(t, success)
	assert.Contains(t, reason, "未提供点击坐标")
}

func TestVerifyClickPoints_CountMismatch(t *testing.T) {
	session := &CaptchaSession{
		MaxPoints: 3,
	}

	req := VerifyRequest{
		Points: [][2]int{{100, 100}, {150, 150}},
	}

	success, reason := verifyClickPoints(session, req)
	assert.False(t, success)
	assert.Contains(t, reason, "点击数量不匹配")
}

func TestVerifyClickPoints_InvalidSequence(t *testing.T) {
	session := &CaptchaSession{
		MaxPoints: 2,
	}

	req := VerifyRequest{
		Points:        [][2]int{{100, 100}, {150, 150}},
		ClickSequence: []int{0, 1, 2},
	}

	success, reason := verifyClickPoints(session, req)
	assert.False(t, success)
	assert.Contains(t, reason, "点击时序长度不匹配")
}

func TestVerifyClickPoints_Success(t *testing.T) {
	session := &CaptchaSession{
		MaxPoints: 2,
		Tolerance: 35,
		TargetPoints: []ClickPoint{
			{X: 100, Y: 100},
			{X: 200, Y: 200},
		},
		HintOrder: []int{0, 1},
	}

	req := VerifyRequest{
		Points:        [][2]int{{100, 100}, {200, 200}},
		ClickSequence: []int{0, 1},
	}

	success, reason := verifyClickPoints(session, req)
	assert.True(t, success)
	assert.Empty(t, reason)
}

func TestCaptchaSession_Structure(t *testing.T) {
	session := &CaptchaSession{
		ID:           "test-session",
		Type:         "slider",
		Mode:         ModeNumber,
		TargetPoints: []ClickPoint{{X: 100, Y: 100}},
		HintOrder:    []int{0},
		Points:       [][2]int{{100, 100}},
		Hint:         "测试提示",
		MaxPoints:    1,
		Tolerance:    10,
		ImageWidth:   300,
		ImageHeight:  300,
		TargetX:      150,
		TargetY:      150,
	}

	assert.Equal(t, "test-session", session.ID)
	assert.Equal(t, "slider", session.Type)
	assert.Equal(t, ModeNumber, session.Mode)
	assert.Len(t, session.TargetPoints, 1)
	assert.Equal(t, 1, session.MaxPoints)
}

func TestClickPoint_Structure(t *testing.T) {
	point := ClickPoint{
		X:     100,
		Y:     200,
		Index: 0,
	}

	assert.Equal(t, 100, point.X)
	assert.Equal(t, 200, point.Y)
	assert.Equal(t, 0, point.Index)
}

func TestVerifyRequest_Structure(t *testing.T) {
	req := VerifyRequest{
		SessionID:     "test-session",
		Type:          "click",
		X:             100,
		Y:             50,
		Points:        [][2]int{{100, 100}},
		ClickSequence: []int{0},
		ApplicationID: 1,
	}

	assert.Equal(t, "test-session", req.SessionID)
	assert.Equal(t, "click", req.Type)
	assert.Equal(t, 100, req.X)
	assert.Equal(t, 50, req.Y)
	assert.Len(t, req.Points, 1)
	assert.Len(t, req.ClickSequence, 1)
	assert.Equal(t, uint(1), req.ApplicationID)
}
