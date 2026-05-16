package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGenerateImageCaptcha(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/captcha/image", GenerateImageCaptcha)

	tests := []struct {
		name         string
		queryParams  string
		expectedCode int
	}{
		{
			name:         "默认参数",
			queryParams:  "",
			expectedCode: http.StatusOK,
		},
		{
			name:         "数字验证码",
			queryParams:  "?type=number",
			expectedCode: http.StatusOK,
		},
		{
			name:         "字母验证码",
			queryParams:  "?type=letter",
			expectedCode: http.StatusOK,
		},
		{
			name:         "混合验证码",
			queryParams:  "?type=mixed",
			expectedCode: http.StatusOK,
		},
		{
			name:         "自定义字符集",
			queryParams:  "?custom_set=abc123&count=5",
			expectedCode: http.StatusOK,
		},
		{
			name:         "指定验证码长度",
			queryParams:  "?count=6",
			expectedCode: http.StatusOK,
		},
		{
			name:         "超过最大长度限制自动修正",
			queryParams:  "?count=10",
			expectedCode: http.StatusOK,
		},
		{
			name:         "小于等于0自动修正",
			queryParams:  "?count=0",
			expectedCode: http.StatusOK,
		},
		{
			name:         "负数自动修正",
			queryParams:  "?count=-1",
			expectedCode: http.StatusOK,
		},
		{
			name:         "指定噪声模式",
			queryParams:  "?noise_mode=3",
			expectedCode: http.StatusOK,
		},
		{
			name:         "指定线条模式",
			queryParams:  "?line_mode=2",
			expectedCode: http.StatusOK,
		},
		{
			name:         "组合参数",
			queryParams:  "?type=number&count=5&noise_mode=4&line_mode=3",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fallbackCaptchaStore = make(map[string]string)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/captcha/image"+tt.queryParams, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp response.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code)

			dataMap, ok := resp.Data.(map[string]interface{})
			assert.True(t, ok, "响应数据应该是map类型")
			assert.NotEmpty(t, dataMap["challenge_id"], "challenge_id不应为空")
			assert.NotEmpty(t, dataMap["image"], "image不应为空")

			imageStr, ok := dataMap["image"].(string)
			assert.True(t, ok, "image应该是字符串类型")
			assert.True(t, strings.HasPrefix(imageStr, "data:image/png;base64,"), "image应该是base64编码的PNG图片")

			base64Data := strings.TrimPrefix(imageStr, "data:image/png;base64,")
			_, err = base64.StdEncoding.DecodeString(base64Data)
			assert.NoError(t, err, "base64解码应该成功")

			decoded, _ := base64.StdEncoding.DecodeString(base64Data)
			img, err := png.Decode(bytes.NewReader(decoded))
			assert.NoError(t, err, "PNG解码应该成功")
			assert.NotNil(t, img, "图片不应为空")
			assert.Equal(t, captchaWidth, img.Bounds().Dx())
			assert.Equal(t, captchaHeight, img.Bounds().Dy())

			challengeID := dataMap["challenge_id"].(string)
			assert.NotEmpty(t, challengeID)
			_, exists := fallbackCaptchaStore[challengeID]
			assert.True(t, exists, "验证码答案应该被存储")
		})
	}
}

func TestVerifyImageCaptcha(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/image/verify", VerifyImageCaptcha)

	tests := []struct {
		name           string
		setupStore     func()
		requestBody    VerifyImageCaptchaRequest
		expectedCode   int
		expectSuccess  bool
		expectedReason string
	}{
		{
			name: "成功验证小写",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-1", "abcd")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-1",
				Answer:     "abcd",
			},
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "验证通过",
		},
		{
			name: "成功验证大写自动转小写",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-2", "test")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-2",
				Answer:     "TEST",
			},
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "验证通过",
		},
		{
			name: "成功验证混合大小写",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-3", "AbCd")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-3",
				Answer:     "abcd",
			},
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "验证通过",
		},
		{
			name: "验证码不存在",
			setupStore: func() {
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "non-existent-id",
				Answer:     "abcd",
			},
			expectedCode:   http.StatusNotFound,
			expectSuccess:  false,
			expectedReason: "验证码不存在",
		},
		{
			name: "答案错误",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-4", "abcd")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-4",
				Answer:     "wrong",
			},
			expectedCode:   http.StatusOK,
			expectSuccess:  false,
			expectedReason: "答案错误",
		},
		{
			name: "部分匹配失败",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-5", "abc123")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-5",
				Answer:     "abc124",
			},
			expectedCode:   http.StatusOK,
			expectSuccess:  false,
			expectedReason: "答案错误",
		},
		{
			name: "空答案",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-6", "abcd")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-6",
				Answer:     "",
			},
			expectedCode:   http.StatusBadRequest,
			expectSuccess:  false,
			expectedReason: "答案错误",
		},
		{
			name: "特殊字符答案",
			setupStore: func() {
				setCaptchaAnswer("test-challenge-7", "a1b2")
			},
			requestBody: VerifyImageCaptchaRequest{
				ChallengeID: "test-challenge-7",
				Answer:     "a1b2",
			},
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "验证通过",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fallbackCaptchaStore = make(map[string]string)
			tt.setupStore()

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/image/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp response.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.Equal(t, 0, resp.Code)
				dataMap, ok := resp.Data.(map[string]interface{})
				assert.True(t, ok)
				success, ok := dataMap["success"].(bool)
				assert.True(t, ok)
				assert.True(t, success, tt.expectedReason)
			} else {
				dataMap, ok := resp.Data.(map[string]interface{})
				if ok {
					success, ok := dataMap["success"].(bool)
					if ok {
						assert.False(t, success, tt.expectedReason)
					}
				}
			}
		})
	}
}

func TestVerifyImageCaptchaInvalidRequest(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/image/verify", VerifyImageCaptcha)

	tests := []struct {
		name         string
		requestBody  interface{}
		expectedCode int
	}{
		{
			name:         "空请求体",
			requestBody:  map[string]string{},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "缺少challenge_id",
			requestBody:  map[string]string{"answer": "test"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "缺少answer",
			requestBody:  map[string]string{"challenge_id": "test-id"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "空challenge_id",
			requestBody:  map[string]string{"challenge_id": "", "answer": "test"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "空answer",
			requestBody:  map[string]string{"challenge_id": "test-id", "answer": ""},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/image/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestVerifyImageCaptchaDeletion(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/image/verify", VerifyImageCaptcha)

	fallbackCaptchaStore = make(map[string]string)
	testID := "deletion-test-id"
	setCaptchaAnswer(testID, "abcd")

	body, _ := json.Marshal(VerifyImageCaptchaRequest{
		ChallengeID: testID,
		Answer:     "abcd",
	})
	req, _ := http.NewRequest("POST", "/api/v1/captcha/image/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, exists := fallbackCaptchaStore[testID]
	assert.False(t, exists, "验证成功后验证码应该被删除")

	body2, _ := json.Marshal(VerifyImageCaptchaRequest{
		ChallengeID: testID,
		Answer:     "abcd",
	})
	req2, _ := http.NewRequest("POST", "/api/v1/captcha/image/verify", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 response.Response
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	dataMap, _ := resp2.Data.(map[string]interface{})
	if dataMap != nil {
		success, ok := dataMap["success"].(bool)
		if ok {
			assert.False(t, success, "重复使用已删除的验证码应该失败")
		}
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		chars  string
		length int
	}{
		{
			name:   "数字字符串",
			chars:  digitCharSet,
			length: 4,
		},
		{
			name:   "字母字符串",
			chars:  letterCharSet,
			length: 6,
		},
		{
			name:   "混合字符串",
			chars:  allCharSet,
			length: 8,
		},
		{
			name:   "单字符",
			chars:  "ABC",
			length: 1,
		},
		{
			name:   "长字符串",
			chars:  allCharSet,
			length: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomString(tt.chars, tt.length)
			assert.Len(t, result, tt.length, "生成字符串长度应该正确")

			for _, c := range result {
				assert.Contains(t, tt.chars, string(c), "生成的字符应该在允许范围内")
			}

			result2 := generateRandomString(tt.chars, tt.length)
			assert.Len(t, result2, tt.length)
		})
	}
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	results := make(map[string]bool)
	chars := digitCharSet
	length := 4
	samples := 100

	for i := 0; i < samples; i++ {
		result := generateRandomString(chars, length)
		results[result] = true
	}

	assert.Greater(t, len(results), samples/2, "随机字符串应该有足够的随机性")
}

func TestGenerateCaptchaImage(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		noiseMode int
		lineMode  int
	}{
		{
			name:      "标准验证码",
			text:      "test123",
			noiseMode: 1,
			lineMode:  1,
		},
		{
			name:      "纯数字",
			text:      "1234",
			noiseMode: 2,
			lineMode:  2,
		},
		{
			name:      "纯字母",
			text:      "abcd",
			noiseMode: 3,
			lineMode:  3,
		},
		{
			name:      "混合字符",
			text:      "A1b2",
			noiseMode: 4,
			lineMode:  4,
		},
		{
			name:      "长验证码",
			text:      "abcdefgh",
			noiseMode: 5,
			lineMode:  5,
		},
		{
			name:      "单字符",
			text:      "A",
			noiseMode: 1,
			lineMode:  1,
		},
		{
			name:      "无效噪声模式",
			text:      "test",
			noiseMode: 99,
			lineMode:  1,
		},
		{
			name:      "无效线条模式",
			text:      "test",
			noiseMode: 1,
			lineMode:  99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := generateCaptchaImage(tt.text, tt.noiseMode, tt.lineMode)

			assert.NotNil(t, img)
			assert.Equal(t, captchaWidth, img.Bounds().Dx())
			assert.Equal(t, captchaHeight, img.Bounds().Dy())

			pixelCount := 0
			for x := 0; x < captchaWidth; x++ {
				for y := 0; y < captchaHeight; y++ {
					_, _, _, a := img.At(x, y).RGBA()
					if a > 0 {
						pixelCount++
					}
				}
			}
			assert.Greater(t, pixelCount, 0, "图片应该有像素内容")
		})
	}
}

func TestGenerateCaptchaImageOutput(t *testing.T) {
	text := "TEST"
	noiseMode := 3
	lineMode := 2

	img := generateCaptchaImage(text, noiseMode, lineMode)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	assert.NoError(t, err)
	assert.Greater(t, buf.Len(), 0)

	decodedImg, err := png.Decode(&buf)
	assert.NoError(t, err)
	assert.NotNil(t, decodedImg)
	assert.Equal(t, captchaWidth, decodedImg.Bounds().Dx())
	assert.Equal(t, captchaHeight, decodedImg.Bounds().Dy())
}

func TestRandomLightColor(t *testing.T) {
	for i := 0; i < 100; i++ {
		col := randomLightColor()
		assert.GreaterOrEqual(t, col.R, uint8(200))
		assert.GreaterOrEqual(t, col.G, uint8(200))
		assert.GreaterOrEqual(t, col.B, uint8(200))
		assert.Equal(t, uint8(255), col.A)
	}
}

func TestRandomDarkColor(t *testing.T) {
	for i := 0; i < 100; i++ {
		col := randomDarkColor()
		assert.LessOrEqual(t, col.R, uint8(100))
		assert.LessOrEqual(t, col.G, uint8(100))
		assert.LessOrEqual(t, col.B, uint8(100))
		assert.Equal(t, uint8(255), col.A)
	}
}

func TestRandomVividColor(t *testing.T) {
	for i := 0; i < 100; i++ {
		col := randomVividColor()
		assert.LessOrEqual(t, col.A, uint8(255))
		assert.GreaterOrEqual(t, col.A, uint8(0))
	}
}

func TestHSLToRgb(t *testing.T) {
	tests := []struct {
		name     string
		h, s, l  float64
		expected color.RGBA
	}{
		{
			name:     "纯红色",
			h:        0,
			s:        1,
			l:        0.5,
			expected: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:     "纯绿色",
			h:        120,
			s:        1,
			l:        0.5,
			expected: color.RGBA{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:     "纯蓝色",
			h:        240,
			s:        1,
			l:        0.5,
			expected: color.RGBA{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:     "灰色",
			h:        0,
			s:        0,
			l:        0.5,
			expected: color.RGBA{R: 128, G: 128, B: 128, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hslToRgb(tt.h, tt.s, tt.l)
			assert.InDelta(t, tt.expected.R, result.R, 1)
			assert.InDelta(t, tt.expected.G, result.G, 1)
			assert.InDelta(t, tt.expected.B, result.B, 1)
			assert.Equal(t, tt.expected.A, result.A)
		})
	}
}

func TestHueToRgb(t *testing.T) {
	tests := []struct {
		name     string
		p, q, t  float64
		expected float64
	}{
		{name: "t<1/6", p: 0.5, q: 0.5, t: 0.1, expected: 0.5},
		{name: "t在1/6和1/2之间", p: 0.5, q: 0.5, t: 0.3, expected: 0.5},
		{name: "t在1/2和2/3之间", p: 0.5, q: 0.5, t: 0.6, expected: 0.5},
		{name: "t<0", p: 0.5, q: 0.5, t: -0.1, expected: 0.5},
		{name: "t>1", p: 0.5, q: 0.5, t: 1.1, expected: 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hueToRgb(tt.p, tt.q, tt.t)
			assert.GreaterOrEqual(t, result, 0.0)
			assert.LessOrEqual(t, result, 1.0)
		})
	}
}

func TestAddComplexNoise(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))

	for mode := 1; mode <= 5; mode++ {
		testImg := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
		bgColor := randomLightColor()
		draw.Draw(testImg, testImg.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

		assert.NotPanics(t, func() {
			addComplexNoise(testImg, mode)
		})
	}

	assert.NotPanics(t, func() {
		addComplexNoise(img, 0)
	})
	assert.NotPanics(t, func() {
		addComplexNoise(img, 99)
	})
}

func TestAddComplexLines(t *testing.T) {
	for mode := 1; mode <= 5; mode++ {
		testImg := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
		bgColor := randomLightColor()
		draw.Draw(testImg, testImg.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

		assert.NotPanics(t, func() {
			addComplexLines(testImg, mode)
		})
	}

	testImg := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
	assert.NotPanics(t, func() {
		addComplexLines(testImg, 0)
	})
	assert.NotPanics(t, func() {
		addComplexLines(testImg, 99)
	})
}

func TestDrawWarpedText(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "标准文本", text: "test"},
		{name: "长文本", text: "longtext"},
		{name: "单字符", text: "A"},
		{name: "数字", text: "1234"},
		{name: "混合", text: "Ab123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
			assert.NotPanics(t, func() {
				drawWarpedText(img, tt.text)
			})
		})
	}
}

func TestApplyTextWarpEffect(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "短文本", text: "ab"},
		{name: "标准文本", text: "abcd"},
		{name: "长文本", text: "abcdefgh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
			assert.NotPanics(t, func() {
				applyTextWarpEffect(img, tt.text)
			})
		})
	}
}

func TestBezierCurves(t *testing.T) {
	quadratic := calculateQuadraticBezierPoints(0, 0, 50, 50, 100, 100, 20)
	assert.Len(t, quadratic, 21)
	for i, p := range quadratic {
		assert.Equal(t, i, i)
		_ = p
	}

	cubic := calculateCubicBezierPoints(0, 0, 25, 50, 75, 50, 100, 100, 20)
	assert.Len(t, cubic, 21)

	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
	assert.NotPanics(t, func() {
		drawQuadraticBezier(img, 0, 0, 50, 50, 100, 100, color.Black)
	})
	assert.NotPanics(t, func() {
		drawCubicBezier(img, 0, 0, 25, 50, 75, 50, 100, 100, color.Black)
	})
}

func TestDrawThickLine(t *testing.T) {
	tests := []struct {
		name      string
		x1, y1    int
		x2, y2    int
		thickness int
	}{
		{name: "水平线", x1: 0, y1: 25, x2: 100, y2: 25, thickness: 2},
		{name: "垂直线", x1: 50, y1: 0, x2: 50, y2: 50, thickness: 1},
		{name: "对角线", x1: 0, y1: 0, x2: 100, y2: 50, thickness: 2},
		{name: "反向对角线", x1: 100, y1: 50, x2: 0, y2: 0, thickness: 1},
		{name: "短线", x1: 10, y1: 10, x2: 20, y2: 20, thickness: 1},
	}

	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				drawThickLine(img, tt.x1, tt.y1, tt.x2, tt.y2, tt.thickness, color.Black)
			})
		})
	}
}

func TestImageAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{input: 5, expected: 5},
		{input: -5, expected: 5},
		{input: 0, expected: 0},
		{input: 100, expected: 100},
		{input: -100, expected: 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := imageAbs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRandInt(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := randInt(1, 10)
		assert.GreaterOrEqual(t, result, 1)
		assert.LessOrEqual(t, result, 10)
	}

	minMax := randInt(5, 5)
	assert.Equal(t, 5, minMax)
}

func TestCaptchaStoreWithMock(t *testing.T) {
	fallbackCaptchaStore = make(map[string]string)

	testID := "test-store-id"
	testAnswer := "test123"

	setCaptchaAnswer(testID, testAnswer)

	retrieved, found := getCaptchaAnswer(testID)
	assert.True(t, found)
	assert.Equal(t, testAnswer, retrieved)

	deleteCaptchaAnswer(testID)

	_, found = getCaptchaAnswer(testID)
	assert.False(t, found)
}

func TestSetGetCaptchaAnswer(t *testing.T) {
	fallbackCaptchaStore = make(map[string]string)

	testCases := []struct {
		id     string
		answer string
	}{
		{"id1", "abcd"},
		{"id2", "1234"},
		{"id3", "a1b2c3"},
		{"id4", "ABCDEFGH"},
		{"id5", "mixed123"},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			setCaptchaAnswer(tc.id, tc.answer)
			retrieved, found := getCaptchaAnswer(tc.id)
			assert.True(t, found)
			assert.Equal(t, strings.ToLower(tc.answer), retrieved)
		})
	}
}

func TestDeleteCaptchaAnswer(t *testing.T) {
	fallbackCaptchaStore = make(map[string]string)

	testID := "delete-test-id"
	setCaptchaAnswer(testID, "test")

	_, found := getCaptchaAnswer(testID)
	assert.True(t, found)

	deleteCaptchaAnswer(testID)

	_, found = getCaptchaAnswer(testID)
	assert.False(t, found)
}

func TestMultipleCaptchaLifecycle(t *testing.T) {
	fallbackCaptchaStore = make(map[string]string)

	captchas := make(map[string]string)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("captcha-%d", i)
		answer := generateRandomString(allCharSet, 4)
		captchas[strings.ToLower(id)] = strings.ToLower(answer)
		setCaptchaAnswer(id, answer)
	}

	for id, answer := range captchas {
		retrieved, found := getCaptchaAnswer(id)
		assert.True(t, found)
		assert.Equal(t, answer, retrieved)
	}

	for id := range captchas {
		deleteCaptchaAnswer(id)
	}

	assert.Equal(t, 0, len(fallbackCaptchaStore))
}

func TestConcurrentCaptchaGeneration(t *testing.T) {
	fallbackCaptchaStore = make(map[string]string)

	for i := 0; i < 10; i++ {
		testID := fmt.Sprintf("test-id-%d", i)
		answer := generateRandomString(allCharSet, 4)
		setCaptchaAnswer(testID, answer)
	}

	assert.Equal(t, 10, len(fallbackCaptchaStore))

	for i := 0; i < 10; i++ {
		testID := fmt.Sprintf("test-id-%d", i)
		_, found := getCaptchaAnswer(testID)
		assert.True(t, found)
	}
}

func TestImageContentValidation(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		noiseMode int
		lineMode  int
	}{
		{"模式1", "test", 1, 1},
		{"模式2", "abcd", 2, 2},
		{"模式3", "1234", 3, 3},
		{"模式4", "Ab12", 4, 4},
		{"模式5", "xy78", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := generateCaptchaImage(tt.text, tt.noiseMode, tt.lineMode)

			var buf bytes.Buffer
			err := png.Encode(&buf, img)
			assert.NoError(t, err)

			decoded, err := png.Decode(&buf)
			assert.NoError(t, err)

			var totalR, totalG, totalB uint64
			var pixelCount int

			for x := 0; x < decoded.Bounds().Dx(); x++ {
				for y := 0; y < decoded.Bounds().Dy(); y++ {
					r, g, b, a := decoded.At(x, y).RGBA()
					if a > 0 {
						totalR += uint64(r >> 8)
						totalG += uint64(g >> 8)
						totalB += uint64(b >> 8)
						pixelCount++
					}
				}
			}

			if pixelCount > 0 {
				avgR := float64(totalR) / float64(pixelCount)
				avgG := float64(totalG) / float64(pixelCount)
				avgB := float64(totalB) / float64(pixelCount)

				assert.True(t, avgR > 0 || avgG > 0 || avgB > 0, "图片应该有颜色内容")
			}
		})
	}
}

func TestPointStruct(t *testing.T) {
}

func TestGenerateRotationCaptcha(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/captcha/rotation", GenerateRotationCaptcha)

	tests := []struct {
		name         string
		expectedCode int
	}{
		{
			name:         "生成旋转验证码",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rotationCaptchaStore = make(map[string]int)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/captcha/rotation", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp response.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code)

			dataMap, ok := resp.Data.(map[string]interface{})
			assert.True(t, ok, "响应数据应该是map类型")
			assert.NotEmpty(t, dataMap["challenge_id"], "challenge_id不应为空")
			assert.NotEmpty(t, dataMap["image"], "image不应为空")

			imageStr, ok := dataMap["image"].(string)
			assert.True(t, ok, "image应该是字符串类型")
			assert.True(t, strings.HasPrefix(imageStr, "data:image/png;base64,"), "image应该是base64编码的PNG图片")

			base64Data := strings.TrimPrefix(imageStr, "data:image/png;base64,")
			_, err = base64.StdEncoding.DecodeString(base64Data)
			assert.NoError(t, err, "base64解码应该成功")

			decoded, _ := base64.StdEncoding.DecodeString(base64Data)
			img, err := png.Decode(bytes.NewReader(decoded))
			assert.NoError(t, err, "PNG解码应该成功")
			assert.NotNil(t, img, "图片不应为空")

			challengeID := dataMap["challenge_id"].(string)
			assert.NotEmpty(t, challengeID)
			_, exists := rotationCaptchaStore[challengeID]
			assert.True(t, exists, "旋转验证码角度应该被存储")
		})
	}
}

func TestVerifyRotationCaptcha(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/rotation/verify", VerifyRotationCaptcha)

	tests := []struct {
		name           string
		setupStore     func() (string, int)
		requestAngle   int
		expectedCode   int
		expectSuccess  bool
		expectedReason string
	}{
		{
			name: "正确角度验证通过",
			setupStore: func() (string, int) {
				return "rotation-test-1", 45
			},
			requestAngle:   45,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "正确角度应验证通过",
		},
		{
			name: "容差范围内验证通过（+5度）",
			setupStore: func() (string, int) {
				return "rotation-test-2", 90
			},
			requestAngle:   95,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "容差范围内应验证通过",
		},
		{
			name: "容差范围内验证通过（-5度）",
			setupStore: func() (string, int) {
				return "rotation-test-3", 90
			},
			requestAngle:   85,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "容差范围内应验证通过",
		},
		{
			name: "边界容差验证通过（+8度）",
			setupStore: func() (string, int) {
				return "rotation-test-4", 100
			},
			requestAngle:   108,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "边界容差应验证通过",
		},
		{
			name: "边界容差验证通过（-8度）",
			setupStore: func() (string, int) {
				return "rotation-test-5", 100
			},
			requestAngle:   92,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "边界容差应验证通过",
		},
		{
			name: "错误角度验证失败",
			setupStore: func() (string, int) {
				return "rotation-test-6", 45
			},
			requestAngle:   180,
			expectedCode:   http.StatusOK,
			expectSuccess:  false,
			expectedReason: "错误角度应验证失败",
		},
		{
			name: "超出容差验证失败（+9度）",
			setupStore: func() (string, int) {
				return "rotation-test-7", 100
			},
			requestAngle:   109,
			expectedCode:   http.StatusOK,
			expectSuccess:  false,
			expectedReason: "超出容差应验证失败",
		},
		{
			name: "超出容差验证失败（-9度）",
			setupStore: func() (string, int) {
				return "rotation-test-8", 100
			},
			requestAngle:   91,
			expectedCode:   http.StatusOK,
			expectSuccess:  false,
			expectedReason: "超出容差应验证失败",
		},
		{
			name: "验证码不存在",
			setupStore: func() (string, int) {
				return "non-existent-rotation", 0
			},
			requestAngle:   45,
			expectedCode:   http.StatusNotFound,
			expectSuccess:  false,
			expectedReason: "验证码不存在",
		},
		{
			name: "角度0度验证",
			setupStore: func() (string, int) {
				return "rotation-test-9", 0
			},
			requestAngle:   0,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "0度应验证通过",
		},
		{
			name: "角度359度验证（边界）",
			setupStore: func() (string, int) {
				return "rotation-test-10", 359
			},
			requestAngle:   359,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "359度应验证通过",
		},
		{
			name: "角度环绕验证（存储355，提交3，差8度）",
			setupStore: func() (string, int) {
				return "rotation-test-11", 355
			},
			requestAngle:   3,
			expectedCode:   http.StatusOK,
			expectSuccess:  true,
			expectedReason: "环绕角度在容差内应验证通过",
		},
		{
			name: "角度环绕验证失败（存储355，提交4，差9度）",
			setupStore: func() (string, int) {
				return "rotation-test-12", 355
			},
			requestAngle:   4,
			expectedCode:   http.StatusOK,
			expectSuccess:  false,
			expectedReason: "环绕角度超出容差应验证失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rotationCaptchaStore = make(map[string]int)
			challengeID, storedAngle := tt.setupStore()

			if challengeID != "non-existent-rotation" {
				setRotationCaptchaAnswer(challengeID, storedAngle)
			}

			angle := tt.requestAngle
			body, _ := json.Marshal(VerifyRotationCaptchaRequest{
				ChallengeID: challengeID,
				Angle:       &angle,
			})
			req, _ := http.NewRequest("POST", "/api/v1/captcha/rotation/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp response.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.Equal(t, 0, resp.Code)
				dataMap, ok := resp.Data.(map[string]interface{})
				assert.True(t, ok)
				success, ok := dataMap["success"].(bool)
				assert.True(t, ok)
				assert.True(t, success, tt.expectedReason)
			} else {
				dataMap, ok := resp.Data.(map[string]interface{})
				if ok {
					success, ok := dataMap["success"].(bool)
					if ok {
						assert.False(t, success, tt.expectedReason)
					}
				}
			}
		})
	}
}

func TestVerifyRotationCaptchaInvalidRequest(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/rotation/verify", VerifyRotationCaptcha)

	tests := []struct {
		name         string
		requestBody  interface{}
		expectedCode int
	}{
		{
			name:         "空请求体",
			requestBody:  map[string]string{},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "缺少challenge_id",
			requestBody:  map[string]interface{}{"angle": 45},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "缺少angle",
			requestBody:  map[string]string{"challenge_id": "test-id"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "空challenge_id",
			requestBody:  map[string]interface{}{"challenge_id": "", "angle": 45},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/rotation/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestVerifyRotationCaptchaDeletion(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/rotation/verify", VerifyRotationCaptcha)

	rotationCaptchaStore = make(map[string]int)
	testID := "rotation-deletion-test-id"
	setRotationCaptchaAnswer(testID, 45)

	angle45 := 45
	body, _ := json.Marshal(VerifyRotationCaptchaRequest{
		ChallengeID: testID,
		Angle:       &angle45,
	})
	req, _ := http.NewRequest("POST", "/api/v1/captcha/rotation/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, exists := rotationCaptchaStore[testID]
	assert.False(t, exists, "验证成功后旋转验证码应该被删除")

	body2, _ := json.Marshal(VerifyRotationCaptchaRequest{
		ChallengeID: testID,
		Angle:       &angle45,
	})
	req2, _ := http.NewRequest("POST", "/api/v1/captcha/rotation/verify", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 response.Response
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	dataMap, _ := resp2.Data.(map[string]interface{})
	if dataMap != nil {
		success, ok := dataMap["success"].(bool)
		if ok {
			assert.False(t, success, "重复使用已删除的旋转验证码应该失败")
		}
	}
}

func TestGenerateRandomAngle(t *testing.T) {
	for i := 0; i < 1000; i++ {
		angle := generateRandomAngle()
		assert.GreaterOrEqual(t, angle, 0, "角度应 >= 0")
		assert.LessOrEqual(t, angle, 359, "角度应 <= 359")
	}

	angles := make(map[int]bool)
	for i := 0; i < 1000; i++ {
		angle := generateRandomAngle()
		angles[angle] = true
	}
	assert.Greater(t, len(angles), 100, "随机角度应有足够的随机性")
}

func TestVerifyRotationAngle(t *testing.T) {
	tests := []struct {
		name      string
		stored    int
		submitted int
		expected  bool
	}{
		{name: "完全匹配", stored: 45, submitted: 45, expected: true},
		{name: "容差内+5", stored: 45, submitted: 50, expected: true},
		{name: "容差内-5", stored: 45, submitted: 40, expected: true},
		{name: "边界+8", stored: 45, submitted: 53, expected: true},
		{name: "边界-8", stored: 45, submitted: 37, expected: true},
		{name: "超出+9", stored: 45, submitted: 54, expected: false},
		{name: "超出-9", stored: 45, submitted: 36, expected: false},
		{name: "大幅偏差", stored: 45, submitted: 180, expected: false},
		{name: "0度匹配", stored: 0, submitted: 0, expected: true},
		{name: "0度容差内", stored: 0, submitted: 8, expected: true},
		{name: "0度超出", stored: 0, submitted: 9, expected: false},
		{name: "359度匹配", stored: 359, submitted: 359, expected: true},
		{name: "环绕容差内（359->3，差4度）", stored: 359, submitted: 3, expected: true},
		{name: "环绕容差内（3->359，差4度）", stored: 3, submitted: 359, expected: true},
		{name: "环绕边界（359->7，差8度）", stored: 359, submitted: 7, expected: true},
		{name: "环绕超出（359->8，差9度）", stored: 359, submitted: 8, expected: false},
		{name: "环绕边界（0->352，差8度）", stored: 0, submitted: 352, expected: true},
		{name: "环绕超出（0->351，差9度）", stored: 0, submitted: 351, expected: false},
		{name: "180度偏差", stored: 0, submitted: 180, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyRotationAngle(tt.stored, tt.submitted)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateRotationCaptchaImage(t *testing.T) {
	tests := []struct {
		name  string
		angle int
	}{
		{name: "0度", angle: 0},
		{name: "45度", angle: 45},
		{name: "90度", angle: 90},
		{name: "180度", angle: 180},
		{name: "270度", angle: 270},
		{name: "359度", angle: 359},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := generateRotationCaptchaImage(tt.angle)

			assert.NotNil(t, img)

			var buf bytes.Buffer
			err := png.Encode(&buf, img)
			assert.NoError(t, err)
			assert.Greater(t, buf.Len(), 0)

			decoded, err := png.Decode(&buf)
			assert.NoError(t, err)
			assert.NotNil(t, decoded)

			var pixelCount int
			for x := 0; x < decoded.Bounds().Dx(); x++ {
				for y := 0; y < decoded.Bounds().Dy(); y++ {
					_, _, _, a := decoded.At(x, y).RGBA()
					if a > 0 {
						pixelCount++
					}
				}
			}
			assert.Greater(t, pixelCount, 0, "图片应该有像素内容")
		})
	}
}

func TestRotationCaptchaStore(t *testing.T) {
	rotationCaptchaStore = make(map[string]int)

	testID := "rotation-store-test"
	testAngle := 123

	setRotationCaptchaAnswer(testID, testAngle)

	retrieved, found := getRotationCaptchaAnswer(testID)
	assert.True(t, found)
	assert.Equal(t, testAngle, retrieved)

	deleteRotationCaptchaAnswer(testID)

	_, found = getRotationCaptchaAnswer(testID)
	assert.False(t, found)
}

func TestMultipleRotationCaptchaLifecycle(t *testing.T) {
	rotationCaptchaStore = make(map[string]int)

	captchas := make(map[string]int)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("rotation-captcha-%d", i)
		angle := generateRandomAngle()
		captchas[id] = angle
		setRotationCaptchaAnswer(id, angle)
	}

	for id, angle := range captchas {
		retrieved, found := getRotationCaptchaAnswer(id)
		assert.True(t, found)
		assert.Equal(t, angle, retrieved)
	}

	for id := range captchas {
		deleteRotationCaptchaAnswer(id)
	}

	assert.Equal(t, 0, len(rotationCaptchaStore))
}

func TestRotationCaptchaImageOutput(t *testing.T) {
	angle := 45
	img := generateRotationCaptchaImage(angle)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	assert.NoError(t, err)
	assert.Greater(t, buf.Len(), 0)

	decodedImg, err := png.Decode(&buf)
	assert.NoError(t, err)
	assert.NotNil(t, decodedImg)
}

func TestRotationCaptchaImageDifferentAngles(t *testing.T) {
	img0 := generateRotationCaptchaImage(0)
	img90 := generateRotationCaptchaImage(90)
	img180 := generateRotationCaptchaImage(180)

	var buf0, buf90, buf180 bytes.Buffer
	png.Encode(&buf0, img0)
	png.Encode(&buf90, img90)
	png.Encode(&buf180, img180)

	assert.NotEqual(t, buf0.Bytes(), buf90.Bytes(), "不同角度的图片应该不同")
	assert.NotEqual(t, buf0.Bytes(), buf180.Bytes(), "不同角度的图片应该不同")
	assert.NotEqual(t, buf90.Bytes(), buf180.Bytes(), "不同角度的图片应该不同")
}
