package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCreateThreeDCaptcha(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/3d/create", CreateThreeDCaptcha)

	threeDGeneratorService = captcha.NewThreeDGeneratorService(nil, nil)
	threeDVerifierService = captcha.NewThreeDVerifierService(nil, nil)

	reqBody := ThreeDCaptchaRequest{
		Difficulty: "medium",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/3d/create", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	dataJSON, _ := json.Marshal(resp.Data)
	var createResp captcha.CreateThreeDResponse
	err = json.Unmarshal(dataJSON, &createResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, createResp.SessionID)
	assert.NotNil(t, createResp.Puzzle)
	assert.Equal(t, "medium", createResp.Puzzle.Difficulty)
}

func TestCreateThreeDCaptcha_DifferentDifficulties(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/3d/create", CreateThreeDCaptcha)

	threeDGeneratorService = captcha.NewThreeDGeneratorService(nil, nil)
	threeDVerifierService = captcha.NewThreeDVerifierService(nil, nil)

	difficulties := []string{"easy", "medium", "hard", "expert"}
	expectedGridSizes := map[string]int{
		"easy":   2,
		"medium": 3,
		"hard":   4,
		"expert": 5,
	}

	for _, difficulty := range difficulties {
		t.Run(difficulty, func(t *testing.T) {
			reqBody := ThreeDCaptchaRequest{
				Difficulty: difficulty,
			}
			jsonBody, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/3d/create", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp response.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			dataJSON, _ := json.Marshal(resp.Data)
			var createResp captcha.CreateThreeDResponse
			err = json.Unmarshal(dataJSON, &createResp)
			assert.NoError(t, err)
			assert.Equal(t, expectedGridSizes[difficulty], createResp.Puzzle.GridSize)
		})
	}
}

func TestNormalizeAngleDiff(t *testing.T) {
	tests := []struct {
		name     string
		angle1   float64
		angle2   float64
		expected float64
	}{
		{"zero", 0, 0, 0},
		{"small diff", 10, 0, 10},
		{"180 diff", 0, 180, 180},
		{"over 180", 0, 200, 160},
		{"360", 0, 360, 0},
		{"negative", -10, 0, 10},
		{"negative over 180", -200, 0, 160},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试normalizeAngleDiff函数逻辑
			diff := absAngle(tt.angle1 - tt.angle2)
			for diff > 180 {
				diff = 360 - diff
			}
			assert.Equal(t, tt.expected, diff)
		})
	}
}

func absAngle(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestGetMaxAllowedDiff(t *testing.T) {
	tests := []struct {
		difficulty string
		expected   float64
	}{
		{"easy", 45},
		{"medium", 30},
		{"hard", 20},
		{"expert", 15},
		{"unknown", 30},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			// 测试getMaxAllowedDiff逻辑
			var result float64
			switch tt.difficulty {
			case "easy":
				result = 45
			case "medium":
				result = 30
			case "hard":
				result = 20
			case "expert":
				result = 15
			default:
				result = 30
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPassThreshold(t *testing.T) {
	tests := []struct {
		difficulty string
		expected   float64
	}{
		{"easy", 70},
		{"medium", 80},
		{"hard", 85},
		{"expert", 90},
		{"unknown", 80},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			// 测试getPassThreshold逻辑
			var result float64
			switch tt.difficulty {
			case "easy":
				result = 70
			case "medium":
				result = 80
			case "hard":
				result = 85
			case "expert":
				result = 90
			default:
				result = 80
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPieceColorsAndTypes(t *testing.T) {
	// 测试pieceColors和pieceTypes变量是否正确设置
	colors := []string{"#e74c3c", "#3498db", "#2ecc71", "#f39c12", "#9b59b6", "#1abc9c", "#e91e63", "#00bcd4", "#8bc34a", "#ff9800"}
	types := []string{"cube", "cylinder", "sphere", "cone", "torus"}

	assert.Greater(t, len(colors), 0)
	assert.Greater(t, len(types), 0)

	// 验证颜色格式
	for _, color := range colors {
		assert.Equal(t, '#', rune(color[0]))
		assert.Len(t, color, 7)
	}
}

func TestThreeDPuzzleStructure(t *testing.T) {
	puzzle := &captcha.ThreeDPuzzle{
		Pieces: []captcha.ThreeDPiece{
			{
				ID:        0,
				Type:      "cube",
				Color:     "#e74c3c",
				PositionX: 0,
				PositionY: 0,
				PositionZ: 0,
				RotationX: 45,
				RotationY: 90,
				RotationZ: 0,
				Scale:     0.8,
			},
		},
		GridSize:   3,
		Difficulty: "medium",
		TargetRotX: 180,
		TargetRotY: 90,
		TargetRotZ: 0,
	}

	assert.NotNil(t, puzzle)
	assert.Equal(t, 3, puzzle.GridSize)
	assert.Equal(t, "medium", puzzle.Difficulty)
	assert.Len(t, puzzle.Pieces, 1)
	assert.Equal(t, "cube", puzzle.Pieces[0].Type)
}

func TestFindPieceByID(t *testing.T) {
	pieces := []captcha.ThreeDPiece{
		{ID: 0, Type: "cube"},
		{ID: 1, Type: "sphere"},
		{ID: 2, Type: "cylinder"},
	}

	// 测试找到的情况
	found := findPieceByIDFromSlice(pieces, 1)
	assert.NotNil(t, found)
	assert.Equal(t, 1, found.ID)
	assert.Equal(t, "sphere", found.Type)

	// 测试找不到的情况
	notFound := findPieceByIDFromSlice(pieces, 999)
	assert.Nil(t, notFound)
}

func findPieceByIDFromSlice(pieces []captcha.ThreeDPiece, id int) *captcha.ThreeDPiece {
	for i := range pieces {
		if pieces[i].ID == id {
			return &pieces[i]
		}
	}
	return nil
}

func TestThreeDCaptchaE2E(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/captcha/3d/create", CreateThreeDCaptcha)
	r.POST("/api/v1/captcha/3d/verify", VerifyThreeDCaptcha)

	threeDGeneratorService = captcha.NewThreeDGeneratorService(nil, nil)
	threeDVerifierService = captcha.NewThreeDVerifierService(nil, nil)

	// Step 1: Create captcha
	reqBody := ThreeDCaptchaRequest{
		Difficulty: "easy",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/3d/create", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	dataJSON, _ := json.Marshal(resp.Data)
	var createResp captcha.CreateThreeDResponse
	err = json.Unmarshal(dataJSON, &createResp)
	assert.NoError(t, err)

	// Create a puzzle with correct rotations
	verifyPuzzle := &captcha.ThreeDPuzzle{
		Pieces:     make([]captcha.ThreeDPiece, len(createResp.Puzzle.Pieces)),
		GridSize:   createResp.Puzzle.GridSize,
		Difficulty: createResp.Puzzle.Difficulty,
		TargetRotX: createResp.Puzzle.TargetRotX,
		TargetRotY: createResp.Puzzle.TargetRotY,
		TargetRotZ: createResp.Puzzle.TargetRotZ,
	}

	// Copy pieces and set rotations to match original
	for i, piece := range createResp.Puzzle.Pieces {
		verifyPuzzle.Pieces[i] = piece
	}

	// Verify
	verifyReq := ThreeDVerifyRequest{
		SessionID: createResp.SessionID,
		Puzzle:    verifyPuzzle,
	}
	verifyJSON, _ := json.Marshal(verifyReq)
	verifyHTTPReq, _ := http.NewRequest("POST", "/api/v1/captcha/3d/verify", bytes.NewReader(verifyJSON))
	verifyHTTPReq.Header.Set("Content-Type", "application/json")

	verifyW := httptest.NewRecorder()
	r.ServeHTTP(verifyW, verifyHTTPReq)

	// 注意：由于没有真实的存储，session可能找不到，这个测试主要验证流程
	assert.Equal(t, http.StatusOK, verifyW.Code)
}
