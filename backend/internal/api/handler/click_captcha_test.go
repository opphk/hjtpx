package handler

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestVerifyClickPoints_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	session := &CaptchaSession{
		ID:        "test-session-basic",
		Type:      "click",
		Mode:      ModeNumber,
		MaxPoints: 3,
		Tolerance: 25,
		TargetPoints: []ClickPoint{
			{X: 50, Y: 50, Index: 0},
			{X: 150, Y: 150, Index: 1},
			{X: 250, Y: 250, Index: 2},
		},
		HintOrder: []int{0, 1, 2},
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name       string
		points     [][2]int
		sequence   []int
		wantPass   bool
		wantReason string
	}{
		{
			name:     "正确顺序点击-无ClickSequence",
			points:   [][2]int{{50, 50}, {150, 150}, {250, 250}},
			wantPass: true,
		},
		{
			name:     "正确顺序点击-有ClickSequence",
			points:   [][2]int{{50, 50}, {150, 150}, {250, 250}},
			sequence: []int{0, 1, 2},
			wantPass: true,
		},
		{
			name:       "点击位置偏差过大",
			points:     [][2]int{{50, 50}, {150, 150}, {500, 500}},
			wantPass:   false,
			wantReason: "无法匹配任何目标点",
		},
		{
			name:       "点击数量不足",
			points:     [][2]int{{50, 50}, {150, 150}},
			wantPass:   false,
			wantReason: "点击数量不匹配",
		},
		{
			name:       "点击数量过多",
			points:     [][2]int{{50, 50}, {150, 150}, {250, 250}, {100, 100}},
			wantPass:   false,
			wantReason: "点击数量不匹配",
		},
		{
			name:       "未提供点击坐标",
			points:     [][2]int{},
			wantPass:   false,
			wantReason: "未提供点击坐标",
		},
		{
			name:     "带容差范围的正确点击",
			points:   [][2]int{{55, 52}, {148, 153}, {247, 249}},
			wantPass: true,
		},
		{
			name:       "容差范围边缘-超出1像素",
			points:     [][2]int{{50, 50}, {150, 150}, {276, 276}},
			wantPass:   false,
			wantReason: "无法匹配任何目标点",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := VerifyRequest{
				SessionID:     session.ID,
				Type:          "click",
				Points:        tt.points,
				ClickSequence: tt.sequence,
			}
			success, reason := verifyClickPoints(session, req)
			assert.Equal(t, tt.wantPass, success)
			if !tt.wantPass && tt.wantReason != "" {
				assert.Contains(t, reason, tt.wantReason)
			}
		})
	}
}

func TestVerifyClickPoints_OrderValidation(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-session-order",
		Type:      "click",
		Mode:      ModeLetter,
		MaxPoints: 3,
		Tolerance: 25,
		TargetPoints: []ClickPoint{
			{X: 50, Y: 50, Index: 0},
			{X: 150, Y: 150, Index: 1},
			{X: 250, Y: 250, Index: 2},
		},
		HintOrder: []int{2, 0, 1},
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name       string
		points     [][2]int
		sequence   []int
		wantPass   bool
		wantReason string
	}{
		{
			name:     "按照HintOrder顺序点击-有ClickSequence",
			points:   [][2]int{{250, 250}, {50, 50}, {150, 150}},
			sequence: []int{0, 1, 2},
			wantPass: true,
		},
		{
			name:     "自然顺序与HintOrder一致",
			points:   [][2]int{{250, 250}, {50, 50}, {150, 150}},
			wantPass: true,
		},
		{
			name:       "错误顺序点击-无ClickSequence",
			points:     [][2]int{{50, 50}, {150, 150}, {250, 250}},
			wantPass:   false,
			wantReason: "点击顺序错误",
		},
		{
			name:       "错误顺序点击-有ClickSequence",
			points:     [][2]int{{250, 250}, {50, 50}, {150, 150}},
			sequence:   []int{2, 1, 0},
			wantPass:   false,
			wantReason: "点击顺序错误",
		},
		{
			name:       "ClickSequence索引无效-超出范围",
			points:     [][2]int{{250, 250}, {50, 50}, {150, 150}},
			sequence:   []int{0, 5, 2},
			wantPass:   false,
			wantReason: "点击时序索引无效",
		},
		{
			name:       "ClickSequence长度不匹配",
			points:     [][2]int{{250, 250}, {50, 50}, {150, 150}},
			sequence:   []int{0, 1},
			wantPass:   false,
			wantReason: "点击时序长度不匹配",
		},
		{
			name:     "带偏差但顺序正确",
			points:   [][2]int{{248, 252}, {52, 48}, {148, 153}},
			sequence: []int{0, 1, 2},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := VerifyRequest{
				SessionID:     session.ID,
				Type:          "click",
				Points:        tt.points,
				ClickSequence: tt.sequence,
			}
			success, reason := verifyClickPoints(session, req)
			assert.Equal(t, tt.wantPass, success)
			if !tt.wantPass && tt.wantReason != "" {
				assert.Contains(t, reason, tt.wantReason)
			}
		})
	}
}

func TestVerifyClickPoints_ChineseMode(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-session-chinese",
		Type:      "click",
		Mode:      ModeChinese,
		MaxPoints: 2,
		Tolerance: 30,
		TargetPoints: []ClickPoint{
			{X: 80, Y: 80, Index: 0},
			{X: 200, Y: 200, Index: 1},
		},
		HintOrder: []int{1, 0},
		CreatedAt: time.Now(),
	}

	req := VerifyRequest{
		SessionID: session.ID,
		Type:      "click",
		Points:    [][2]int{{200, 200}, {80, 80}},
	}
	success, reason := verifyClickPoints(session, req)
	assert.True(t, success, "中文模式正确顺序应验证通过")
	assert.Empty(t, reason)

	req2 := VerifyRequest{
		SessionID: session.ID,
		Type:      "click",
		Points:    [][2]int{{80, 80}, {200, 200}},
	}
	success2, reason2 := verifyClickPoints(session, req2)
	assert.False(t, success2, "中文模式错误顺序应验证失败")
	assert.Contains(t, reason2, "点击顺序错误")
}

func TestVerifyClickPoints_MixedMode(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-session-mixed",
		Type:      "click",
		Mode:      ModeMixed,
		MaxPoints: 4,
		Tolerance: 20,
		TargetPoints: []ClickPoint{
			{X: 30, Y: 30, Index: 0},
			{X: 100, Y: 100, Index: 1},
			{X: 170, Y: 170, Index: 2},
			{X: 240, Y: 240, Index: 3},
		},
		HintOrder: []int{3, 1, 2, 0},
		CreatedAt: time.Now(),
	}

	req := VerifyRequest{
		SessionID:     session.ID,
		Type:          "click",
		Points:        [][2]int{{240, 240}, {100, 100}, {170, 170}, {30, 30}},
		ClickSequence: []int{0, 1, 2, 3},
	}
	success, reason := verifyClickPoints(session, req)
	assert.True(t, success, "混合模式正确顺序应验证通过")
	assert.Empty(t, reason)
}

func TestVerifyClickPoints_IconMode(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-session-icon",
		Type:      "click",
		Mode:      ModeIcon,
		MaxPoints: 2,
		Tolerance: 25,
		TargetPoints: []ClickPoint{
			{X: 60, Y: 60, Index: 0},
			{X: 180, Y: 180, Index: 1},
		},
		HintOrder: []int{0, 1},
		CreatedAt: time.Now(),
	}

	req := VerifyRequest{
		SessionID: session.ID,
		Type:      "click",
		Points:    [][2]int{{60, 60}, {180, 180}},
	}
	success, reason := verifyClickPoints(session, req)
	assert.True(t, success, "图标模式正确顺序应验证通过")
	assert.Empty(t, reason)
}

func TestVerifyClickPoints_EdgeCases(t *testing.T) {
	t.Run("Tolerance为0时使用默认值", func(t *testing.T) {
		session := &CaptchaSession{
			ID:        "test-edge-1",
			Type:      "click",
			MaxPoints: 1,
			Tolerance: 0,
			TargetPoints: []ClickPoint{
				{X: 100, Y: 100, Index: 0},
			},
			HintOrder: []int{0},
			CreatedAt: time.Now(),
		}
		req := VerifyRequest{
			SessionID: session.ID,
			Type:      "click",
			Points:    [][2]int{{100, 100}},
		}
		success, reason := verifyClickPoints(session, req)
		assert.True(t, success)
		assert.Empty(t, reason)
	})

	t.Run("HintOrder为空时使用默认顺序", func(t *testing.T) {
		session := &CaptchaSession{
			ID:        "test-edge-2",
			Type:      "click",
			MaxPoints: 2,
			Tolerance: 25,
			TargetPoints: []ClickPoint{
				{X: 50, Y: 50, Index: 0},
				{X: 150, Y: 150, Index: 1},
			},
			CreatedAt: time.Now(),
		}
		req := VerifyRequest{
			SessionID: session.ID,
			Type:      "click",
			Points:    [][2]int{{50, 50}, {150, 150}},
		}
		success, reason := verifyClickPoints(session, req)
		assert.True(t, success)
		assert.Empty(t, reason)
	})

	t.Run("5个点的多点验证", func(t *testing.T) {
		session := &CaptchaSession{
			ID:        "test-edge-3",
			Type:      "click",
			MaxPoints: 5,
			Tolerance: 20,
			TargetPoints: []ClickPoint{
				{X: 20, Y: 20, Index: 0},
				{X: 80, Y: 80, Index: 1},
				{X: 140, Y: 140, Index: 2},
				{X: 200, Y: 200, Index: 3},
				{X: 260, Y: 260, Index: 4},
			},
			HintOrder: []int{4, 3, 2, 1, 0},
			CreatedAt: time.Now(),
		}
		req := VerifyRequest{
			SessionID:     session.ID,
			Type:          "click",
			Points:        [][2]int{{260, 260}, {200, 200}, {140, 140}, {80, 80}, {20, 20}},
			ClickSequence: []int{0, 1, 2, 3, 4},
		}
		success, reason := verifyClickPoints(session, req)
		assert.True(t, success, "5个点正确顺序应验证通过")
		assert.Empty(t, reason)
	})

	t.Run("最佳匹配算法测试-点击在两个目标之间", func(t *testing.T) {
		session := &CaptchaSession{
			ID:        "test-edge-4",
			Type:      "click",
			MaxPoints: 2,
			Tolerance: 50,
			TargetPoints: []ClickPoint{
				{X: 50, Y: 50, Index: 0},
				{X: 100, Y: 100, Index: 1},
			},
			HintOrder: []int{0, 1},
			CreatedAt: time.Now(),
		}
		req := VerifyRequest{
			SessionID: session.ID,
			Type:      "click",
			Points:    [][2]int{{48, 48}, {102, 102}},
		}
		success, reason := verifyClickPoints(session, req)
		assert.True(t, success, "最佳匹配算法应正确匹配最近的目标")
		assert.Empty(t, reason)
	})
}

func TestGetClickCaptchaEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	captchaSessions = make(map[string]*CaptchaSession)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantMode   string
	}{
		{
			name:       "默认数字模式",
			query:      "",
			wantStatus: http.StatusOK,
			wantMode:   "number",
		},
		{
			name:       "字母模式",
			query:      "?mode=letter",
			wantStatus: http.StatusOK,
			wantMode:   "letter",
		},
		{
			name:       "中文模式",
			query:      "?mode=chinese",
			wantStatus: http.StatusOK,
			wantMode:   "chinese",
		},
		{
			name:       "混合模式",
			query:      "?mode=mixed",
			wantStatus: http.StatusOK,
			wantMode:   "mixed",
		},
		{
			name:       "图标模式",
			query:      "?mode=icon",
			wantStatus: http.StatusOK,
			wantMode:   "icon",
		},
		{
			name:       "指定点数",
			query:      "?points=4",
			wantStatus: http.StatusOK,
			wantMode:   "number",
		},
		{
			name:       "不随机顺序",
			query:      "?shuffle=false",
			wantStatus: http.StatusOK,
			wantMode:   "number",
		},
	}

	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/captcha/click"+tt.query, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			assert.NotEmpty(t, resp["session_id"])
			assert.NotEmpty(t, resp["image_url"])
			assert.NotEmpty(t, resp["hint"])
			assert.NotNil(t, resp["hint_order"])
			assert.NotNil(t, resp["max_points"])
			assert.Equal(t, tt.wantMode, resp["mode"])

			assert.True(t, strings.HasPrefix(resp["image_url"].(string), "data:image/png;base64,"))
		})
	}
}

func TestFormatHintOrder(t *testing.T) {
	tests := []struct {
		name     string
		order    []int
		expected string
	}{
		{"空数组", []int{}, ""},
		{"单个元素", []int{0}, "1"},
		{"两个元素", []int{0, 1}, "1→2"},
		{"三个元素", []int{2, 0, 1}, "3→1→2"},
		{"乱序", []int{3, 1, 4, 2}, "4→2→5→3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHintOrder(tt.order)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIconName(t *testing.T) {
	tests := []struct {
		name     string
		icon     string
		expected string
	}{
		{"圆形", "circle", "圆形"},
		{"方形", "square", "方形"},
		{"三角形", "triangle", "三角形"},
		{"星形", "star", "星形"},
		{"菱形", "diamond", "菱形"},
		{"心形", "heart", "心形"},
		{"箭头", "arrow", "箭头"},
		{"十字", "cross", "十字"},
		{"月牙", "moon", "月牙"},
		{"圆环", "ring", "圆环"},
		{"未知图标返回原值", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIconName(tt.icon)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderIcon(t *testing.T) {
	icons := []IconType{
		IconCircle, IconSquare, IconTriangle, IconStar,
		IconDiamond, IconHeart, IconArrow, IconCross,
		IconMoon, IconRing,
	}

	for _, icon := range icons {
		t.Run(string(icon), func(t *testing.T) {
			img := renderIcon(icon, 60, color.RGBA{R: 255, G: 100, B: 50, A: 255})
			assert.NotNil(t, img)
			assert.Equal(t, 60, img.Bounds().Dx())
			assert.Equal(t, 60, img.Bounds().Dy())

			hasContent := false
			for x := 0; x < 60; x++ {
				for y := 0; y < 60; y++ {
					_, _, _, a := img.At(x, y).RGBA()
					if a > 0 {
						hasContent = true
						break
					}
				}
				if hasContent {
					break
				}
			}
			assert.True(t, hasContent, "图标 %s 应有像素内容", icon)
		})
	}
}

func TestVerifyClickPoints_BestMatch(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-bestmatch",
		Type:      "click",
		MaxPoints: 2,
		Tolerance: 30,
		TargetPoints: []ClickPoint{
			{X: 50, Y: 50, Index: 0},
			{X: 100, Y: 100, Index: 1},
		},
		HintOrder: []int{0, 1},
		CreatedAt: time.Now(),
	}

	req := VerifyRequest{
		SessionID: session.ID,
		Type:      "click",
		Points:    [][2]int{{55, 55}, {95, 95}},
	}
	success, reason := verifyClickPoints(session, req)
	assert.True(t, success, "最佳匹配应正确分配目标")
	assert.Empty(t, reason)
}

func TestVerifyClickPoints_DuplicateTargetRejection(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-duplicate",
		Type:      "click",
		MaxPoints: 3,
		Tolerance: 30,
		TargetPoints: []ClickPoint{
			{X: 50, Y: 50, Index: 0},
			{X: 150, Y: 150, Index: 1},
			{X: 250, Y: 250, Index: 2},
		},
		HintOrder: []int{0, 1, 2},
		CreatedAt: time.Now(),
	}

	req := VerifyRequest{
		SessionID: session.ID,
		Type:      "click",
		Points:    [][2]int{{50, 50}, {55, 55}, {250, 250}},
	}
	success, reason := verifyClickPoints(session, req)
	assert.False(t, success, "重复点击同一目标应验证失败")
	assert.Contains(t, reason, "无法匹配任何目标点")
}

func TestGenerateClickImageWithBackground(t *testing.T) {
	modes := []CaptchaMode{ModeNumber, ModeLetter, ModeChinese, ModeMixed, ModeIcon}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			session := &CaptchaSession{
				ID:           fmt.Sprintf("test-gen-%s", mode),
				Type:         "click",
				Mode:         mode,
				MaxPoints:    3,
				AllowShuffle: false,
				CreatedAt:    time.Now(),
				ImageSeed:    time.Now().UnixNano(),
			}

			imageURL, targetPoints, hintOrder, hint := generateClickImageWithBackground(session)
			assert.NotEmpty(t, imageURL)
			assert.True(t, strings.HasPrefix(imageURL, "data:image/png;base64,"))
			assert.Len(t, targetPoints, 3)
			assert.Len(t, hintOrder, 3)
			assert.NotEmpty(t, hint)
			assert.Contains(t, hint, "依次点击")

			for _, pt := range targetPoints {
				assert.Greater(t, pt.X, 0)
				assert.Greater(t, pt.Y, 0)
				assert.Less(t, pt.X, session.ImageWidth)
				assert.Less(t, pt.Y, session.ImageHeight)
			}
		})
	}
}

func TestVerifyClickPoints_DistancePrecision(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-precision",
		Type:      "click",
		MaxPoints: 1,
		Tolerance: 25,
		TargetPoints: []ClickPoint{
			{X: 100, Y: 100, Index: 0},
		},
		HintOrder: []int{0},
		CreatedAt: time.Now(),
	}

	distances := []struct {
		name string
		x, y int
		pass bool
	}{
		{"正好在目标点", 100, 100, true},
		{"容差范围内-水平", 120, 100, true},
		{"容差范围内-垂直", 100, 120, true},
		{"容差范围内-对角线", 117, 117, true},
		{"容差边界-距离25", 125, 100, true},
		{"超出容差-距离26", 126, 100, false},
		{"超出容差-对角线", 118, 118, false},
	}

	for _, d := range distances {
		t.Run(d.name, func(t *testing.T) {
			req := VerifyRequest{
				SessionID: session.ID,
				Type:      "click",
				Points:    [][2]int{{d.x, d.y}},
			}
			success, _ := verifyClickPoints(session, req)
			expectedDist := math.Sqrt(float64((d.x-100)*(d.x-100) + (d.y-100)*(d.y-100)))
			assert.Equal(t, d.pass, success, "距离=%.2f, 容差=%d", expectedDist, session.Tolerance)
		})
	}
}

func TestVerifyClickPoints_MaxPoints6(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-max6",
		Type:      "click",
		MaxPoints: 6,
		Tolerance: 20,
		TargetPoints: []ClickPoint{
			{X: 20, Y: 20, Index: 0},
			{X: 70, Y: 70, Index: 1},
			{X: 120, Y: 120, Index: 2},
			{X: 170, Y: 170, Index: 3},
			{X: 220, Y: 220, Index: 4},
			{X: 270, Y: 270, Index: 5},
		},
		HintOrder: []int{0, 1, 2, 3, 4, 5},
		CreatedAt: time.Now(),
	}

	req := VerifyRequest{
		SessionID: session.ID,
		Type:      "click",
		Points: [][2]int{
			{20, 20}, {70, 70}, {120, 120},
			{170, 170}, {220, 220}, {270, 270},
		},
	}
	success, reason := verifyClickPoints(session, req)
	assert.True(t, success, "6个点正确顺序应验证通过")
	assert.Empty(t, reason)

	req2 := VerifyRequest{
		SessionID: session.ID,
		Type:      "click",
		Points: [][2]int{
			{270, 270}, {220, 220}, {170, 170},
			{120, 120}, {70, 70}, {20, 20},
		},
	}
	success2, reason2 := verifyClickPoints(session, req2)
	assert.False(t, success2, "6个点逆序应验证失败")
	assert.Contains(t, reason2, "点击顺序错误")
}

func TestVerifyClickPoints_ClickSequenceReverseOrder(t *testing.T) {
	session := &CaptchaSession{
		ID:        "test-reverse",
		Type:      "click",
		MaxPoints: 3,
		Tolerance: 25,
		TargetPoints: []ClickPoint{
			{X: 50, Y: 50, Index: 0},
			{X: 150, Y: 150, Index: 1},
			{X: 250, Y: 250, Index: 2},
		},
		HintOrder: []int{2, 1, 0},
		CreatedAt: time.Now(),
	}

	req := VerifyRequest{
		SessionID:     session.ID,
		Type:          "click",
		Points:        [][2]int{{50, 50}, {150, 150}, {250, 250}},
		ClickSequence: []int{2, 1, 0},
	}
	success, reason := verifyClickPoints(session, req)
	assert.True(t, success, "通过ClickSequence指定倒序点击应验证通过")
	assert.Empty(t, reason)
}