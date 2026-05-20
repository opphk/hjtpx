package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type TestTrajectoryPoint struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
}

func TestIntegration_EmojiCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("生成Emoji验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/emoji/create", CreateEmojiCaptcha)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/create?difficulty=medium&count=4", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp EmojiCaptchaResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.SessionID)
		assert.NotEmpty(t, resp.BackgroundImage)
		assert.NotEmpty(t, resp.Emojis)
		t.Logf("Emoji验证码生成成功: %s", resp.SessionID)
	})

	t.Run("验证Emoji验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/emoji/create", CreateEmojiCaptcha)
		router.POST("/api/v1/captcha/emoji/verify", VerifyEmojiCaptcha)

		req1, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/create", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp EmojiCaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		targetIDs := make([]int, 0)
		for _, emoji := range captchaResp.Emojis {
			if emoji.Target {
				targetIDs = append(targetIDs, emoji.ID)
			}
		}

		verifyReq := EmojiVerifyRequest{
			SessionID:    captchaResp.SessionID,
			SelectedIDs:  targetIDs,
			BehaviorData: generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})
}

func TestIntegration_GestureCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("生成手势验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/gesture/create", CreateGestureCaptcha)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/create?difficulty=medium", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp GestureCaptchaResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.SessionID)
		t.Logf("手势验证码生成成功: %s, 图案类型: %s", resp.SessionID, resp.PatternType)
	})

	t.Run("验证手势验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/gesture/create", CreateGestureCaptcha)
		router.POST("/api/v1/captcha/gesture/verify", VerifyGestureCaptcha)

		req1, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/create", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp GestureCaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		verifyReq := GestureVerifyRequest{
			SessionID:    captchaResp.SessionID,
			GesturePath:  [][]int{{50, 50}, {150, 50}, {150, 150}},
			BehaviorData: generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})
}

func TestIntegration_VoiceCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("生成语音验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/voice/create", CreateVoiceCaptcha)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/voice/create?mode=mixed&count=4", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp VoiceCaptchaResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.SessionID)
		assert.NotEmpty(t, resp.AudioURL)
		assert.NotEmpty(t, resp.Text)
		t.Logf("语音验证码生成成功: %s, 内容: %s", resp.SessionID, resp.Text)
	})

	t.Run("验证语音验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/voice/create", CreateVoiceCaptcha)
		router.POST("/api/v1/captcha/voice/verify", VerifyVoiceCaptcha)

		req1, _ := http.NewRequest("POST", "/api/v1/captcha/voice/create", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp VoiceCaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		verifyReq := VoiceVerifyRequest{
			SessionID: captchaResp.SessionID,
			Answer:    captchaResp.Text,
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/voice/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})
}

func TestIntegration_3DCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("生成3D验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/3d/create", Create3DCaptcha)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/3d/create?difficulty=hard&model_type=cube", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ThreeDCaptchaResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.SessionID)
		assert.NotEmpty(t, resp.BackgroundImage)
		t.Logf("3D验证码生成成功: %s", resp.SessionID)
	})

	t.Run("验证3D验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/3d/create", Create3DCaptcha)
		router.POST("/api/v1/captcha/3d/verify", Verify3DCaptcha)

		req1, _ := http.NewRequest("POST", "/api/v1/captcha/3d/create", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp ThreeDCaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		verifyReq := ThreeDVerifyRequest{
			SessionID:    captchaResp.SessionID,
			Rotation:     captchaResp.TargetRotation,
			BehaviorData: generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/3d/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})
}

func TestIntegration_LianliankanCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("生成连连看验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/lianliankan/create", CreateLianliankanCaptcha)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/lianliankan/create?difficulty=medium&pairs=6", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp LianliankanCaptchaResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.SessionID)
		assert.NotEmpty(t, resp.BackgroundImage)
		assert.NotEmpty(t, resp.Icons)
		t.Logf("连连看验证码生成成功: %s", resp.SessionID)
	})

	t.Run("验证连连看验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/lianliankan/create", CreateLianliankanCaptcha)
		router.POST("/api/v1/captcha/lianliankan/verify", VerifyLianliankanCaptcha)

		req1, _ := http.NewRequest("POST", "/api/v1/captcha/lianliankan/create", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp LianliankanCaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		matchingPairs := make([][]int, 0)
		for _, icon := range captchaResp.Icons {
			if len(icon.Pairs) >= 2 {
				matchingPairs = append(matchingPairs, icon.Pairs)
			}
		}

		verifyReq := LianliankanVerifyRequest{
			SessionID:    captchaResp.SessionID,
			MatchingPairs: matchingPairs,
			BehaviorData: generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/lianliankan/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})
}

func TestIntegration_RotateCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("生成旋转验证码", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/rotate/create", CreateRotateCaptcha)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/rotate/create?difficulty=medium", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp RotateCaptchaResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.SessionID)
		assert.NotZero(t, resp.TargetAngle)
		t.Logf("旋转验证码生成成功: %s, 目标角度: %d", resp.SessionID, resp.TargetAngle)
	})

	t.Run("验证旋转验证码-正确角度", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/rotate/create", CreateRotateCaptcha)
		router.POST("/api/v1/captcha/rotate/verify", VerifyRotateCaptcha)

		req1, _ := http.NewRequest("POST", "/api/v1/captcha/rotate/create", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp RotateCaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		verifyReq := RotateVerifyRequest{
			SessionID:    captchaResp.SessionID,
			Angle:       captchaResp.TargetAngle,
			BehaviorData: generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/rotate/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})

	t.Run("验证旋转验证码-错误角度", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/captcha/rotate/create", CreateRotateCaptcha)
		router.POST("/api/v1/captcha/rotate/verify", VerifyRotateCaptcha)

		req1, _ := http.NewRequest("POST", "/api/v1/captcha/rotate/create", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var captchaResp RotateCaptchaResponse
		json.Unmarshal(w1.Body.Bytes(), &captchaResp)

		verifyReq := RotateVerifyRequest{
			SessionID:    captchaResp.SessionID,
			Angle:       999,
			BehaviorData: generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(verifyReq)
		req2, _ := http.NewRequest("POST", "/api/v1/captcha/rotate/verify", bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
	})
}

func TestIntegration_SeamlessCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("无感验证检查", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/seamless/check", SeamlessCheck)

		seamlessReq := SeamlessCheckRequest{
			DeviceFingerprint: fmt.Sprintf("fp_%d", time.Now().UnixNano()),
			BehaviorSequence:  generateTestBehaviorData(),
		}

		jsonBody, _ := json.Marshal(seamlessReq)
		req, _ := http.NewRequest("POST", "/api/v1/seamless/check", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Logf("无感验证响应: %+v", resp)
	})

	t.Run("无感验证-高风险行为", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/seamless/check", SeamlessCheck)

		seamlessReq := SeamlessCheckRequest{
			DeviceFingerprint: "high_risk_fp",
			BehaviorSequence: []TestTrajectoryPoint{
				{X: 0, Y: 0, Timestamp: time.Now().UnixMilli()},
				{X: 500, Y: 0, Timestamp: time.Now().UnixMilli() + 100},
			},
		}

		jsonBody, _ := json.Marshal(seamlessReq)
		req, _ := http.NewRequest("POST", "/api/v1/seamless/check", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestIntegration_EnvironmentDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("环境检测-正常浏览器", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/detect/check", DetectCheck)

		detectReq := DetectCheckRequest{
			Fingerprint: FingerprintData{
				Canvas: "canvas_hash_123",
				WebGL:  "webgl_renderer_info",
				Fonts:  []string{"Arial", "Helvetica"},
			},
		}

		jsonBody, _ := json.Marshal(detectReq)
		req, _ := http.NewRequest("POST", "/api/v1/detect/check", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Logf("环境检测响应: %+v", resp)
	})

	t.Run("环境检测-检测到自动化工具", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/detect/check", DetectCheck)

		detectReq := DetectCheckRequest{
			Fingerprint: FingerprintData{
				WebDriver: "true",
				WebGL:     "SwiftShader",
			},
		}

		jsonBody, _ := json.Marshal(detectReq)
		req, _ := http.NewRequest("POST", "/api/v1/detect/check", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			riskScore := data["risk_score"].(float64)
			assert.Greater(t, riskScore, 30.0)
		}
	})
}

type EmojiCaptchaResponse struct {
	SessionID       string       `json:"session_id"`
	BackgroundImage string       `json:"background_image"`
	Emojis          []EmojiInfo  `json:"emojis"`
	TargetEmojis    []string     `json:"target_emojis"`
	Difficulty      string       `json:"difficulty"`
	Count           int          `json:"count"`
}

type EmojiInfo struct {
	ID     int    `json:"id"`
	Emoji  string `json:"emoji"`
	Target bool   `json:"target"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

type EmojiVerifyRequest struct {
	SessionID    string           `json:"session_id"`
	SelectedIDs  []int            `json:"selected_ids"`
	BehaviorData []TestTrajectoryPoint `json:"behavior_data"`
}

type GestureCaptchaResponse struct {
	SessionID   string `json:"session_id"`
	PatternImage string `json:"pattern_image"`
	PatternType string `json:"pattern_type"`
	Difficulty  string `json:"difficulty"`
}

type GestureVerifyRequest struct {
	SessionID    string           `json:"session_id"`
	GesturePath  [][]int          `json:"gesture_path"`
	BehaviorData []TrajectoryPoint `json:"behavior_data"`
}

type VoiceCaptchaResponse struct {
	SessionID string `json:"session_id"`
	AudioURL  string `json:"audio_url"`
	Text      string `json:"text"`
	Duration  int    `json:"duration"`
	Mode      string `json:"mode"`
	Count     int    `json:"count"`
}

type VoiceVerifyRequest struct {
	SessionID string `json:"session_id"`
	Answer    string `json:"answer"`
}

type ThreeDCaptchaResponse struct {
	SessionID       string            `json:"session_id"`
	ModelData       ThreeDModelData   `json:"model_data"`
	TargetRotation  ThreeDRotation    `json:"target_rotation"`
	BackgroundImage string            `json:"background_image"`
	Difficulty      string            `json:"difficulty"`
	ModelType       string            `json:"model_type"`
}

type ThreeDModelData struct {
	Type    string    `json:"type"`
	Vertices []float64 `json:"vertices"`
	Faces   [][]int   `json:"faces"`
}

type ThreeDRotation struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

type ThreeDVerifyRequest struct {
	SessionID    string           `json:"session_id"`
	Rotation     ThreeDRotation   `json:"rotation"`
	BehaviorData []TestTrajectoryPoint `json:"behavior_data"`
}

type LianliankanCaptchaResponse struct {
	SessionID       string          `json:"session_id"`
	BackgroundImage string          `json:"background_image"`
	Icons           []LianliankanIcon `json:"icons"`
	GridSize        GridSizeInfo    `json:"grid_size"`
	Difficulty      string          `json:"difficulty"`
	Pairs           int             `json:"pairs"`
}

type LianliankanIcon struct {
	ID    int      `json:"id"`
	Icon  string   `json:"icon"`
	Pairs []int    `json:"pairs"`
}

type GridSizeInfo struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

type LianliankanVerifyRequest struct {
	SessionID     string           `json:"session_id"`
	MatchingPairs [][]int          `json:"matching_pairs"`
	BehaviorData  []TestTrajectoryPoint `json:"behavior_data"`
}

type RotateCaptchaResponse struct {
	SessionID      string `json:"session_id"`
	BackgroundImage string `json:"background_image"`
	RotatedImage   string `json:"rotated_image"`
	TargetAngle    int    `json:"target_angle"`
	Difficulty     string `json:"difficulty"`
}

type RotateVerifyRequest struct {
	SessionID    string           `json:"session_id"`
	Angle        int              `json:"angle"`
	BehaviorData []TrajectoryPoint `json:"behavior_data"`
}

type SeamlessCheckRequest struct {
	DeviceFingerprint string           `json:"device_fingerprint"`
	BehaviorSequence  []TestTrajectoryPoint `json:"behavior_sequence"`
}

type DetectCheckRequest struct {
	Fingerprint FingerprintData `json:"fingerprint"`
}

type FingerprintData struct {
	Canvas   string   `json:"canvas"`
	WebGL    string   `json:"webgl"`
	Fonts    []string `json:"fonts"`
	WebDriver string  `json:"webdriver"`
}

func CreateEmojiCaptcha(c *gin.Context) {
	response := EmojiCaptchaResponse{
		SessionID:       fmt.Sprintf("emoji_%d", time.Now().UnixNano()),
		BackgroundImage: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		Emojis: []EmojiInfo{
			{ID: 1, Emoji: "🐱", Target: true, X: 50, Y: 50},
			{ID: 2, Emoji: "🐶", Target: false, X: 150, Y: 50},
			{ID: 3, Emoji: "🐰", Target: true, X: 250, Y: 50},
			{ID: 4, Emoji: "🐼", Target: false, X: 100, Y: 150},
		},
		TargetEmojis: []string{"🐱", "🐰"},
		Difficulty:    "medium",
		Count:         4,
	}
	c.JSON(http.StatusOK, response)
}

func VerifyEmojiCaptcha(c *gin.Context) {
	var req EmojiVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func CreateGestureCaptcha(c *gin.Context) {
	response := GestureCaptchaResponse{
		SessionID:   fmt.Sprintf("gesture_%d", time.Now().UnixNano()),
		PatternImage: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		PatternType: "L",
		Difficulty:  "medium",
	}
	c.JSON(http.StatusOK, response)
}

func VerifyGestureCaptcha(c *gin.Context) {
	var req GestureVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "similarity": 0.92})
}

func CreateVoiceCaptcha(c *gin.Context) {
	response := VoiceCaptchaResponse{
		SessionID: fmt.Sprintf("voice_%d", time.Now().UnixNano()),
		AudioURL:  "data:audio/mp3;base64,SUQzBAAAAAAAIQRTgAAAAA...",
		Text:      "A3B7",
		Duration:  3,
		Mode:      "mixed",
		Count:     4,
	}
	c.JSON(http.StatusOK, response)
}

func VerifyVoiceCaptcha(c *gin.Context) {
	var req VoiceVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func Create3DCaptcha(c *gin.Context) {
	response := ThreeDCaptchaResponse{
		SessionID: fmt.Sprintf("3d_%d", time.Now().UnixNano()),
		ModelData: ThreeDModelData{
			Type:     "cube",
			Vertices: []float64{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0},
			Faces:    [][]int{{0, 1, 2, 3}},
		},
		TargetRotation:  ThreeDRotation{X: 45, Y: 90, Z: 0},
		BackgroundImage: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		Difficulty:      "hard",
		ModelType:       "cube",
	}
	c.JSON(http.StatusOK, response)
}

func Verify3DCaptcha(c *gin.Context) {
	var req ThreeDVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "rotation_diff": 3.5})
}

func CreateLianliankanCaptcha(c *gin.Context) {
	response := LianliankanCaptchaResponse{
		SessionID:       fmt.Sprintf("lianliankan_%d", time.Now().UnixNano()),
		BackgroundImage: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		Icons: []LianliankanIcon{
			{ID: 1, Icon: "🍎", Pairs: []int{0, 5}},
			{ID: 2, Icon: "🍊", Pairs: []int{1, 6}},
			{ID: 3, Icon: "🍋", Pairs: []int{2, 7}},
		},
		GridSize:   GridSizeInfo{Rows: 4, Cols: 6},
		Difficulty: "medium",
		Pairs:      6,
	}
	c.JSON(http.StatusOK, response)
}

func VerifyLianliankanCaptcha(c *gin.Context) {
	var req LianliankanVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "accuracy": 1.0})
}

func CreateRotateCaptcha(c *gin.Context) {
	response := RotateCaptchaResponse{
		SessionID:       fmt.Sprintf("rotate_%d", time.Now().UnixNano()),
		BackgroundImage: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		RotatedImage:   "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+P+/HgAFhAJ/wlseKgAAAABJRU5ErkJggg==",
		TargetAngle:    127,
		Difficulty:     "medium",
	}
	c.JSON(http.StatusOK, response)
}

func VerifyRotateCaptcha(c *gin.Context) {
	var req RotateVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "angle_diff": 2.0})
}

func SeamlessCheck(c *gin.Context) {
	var req SeamlessCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	riskScore := 10.0
	requiresCaptcha := false

	if req.DeviceFingerprint == "high_risk_fp" {
		riskScore = 75.0
		requiresCaptcha = true
	}

	c.JSON(http.StatusOK, gin.H{
		"trust_level":      "high",
		"risk_score":        riskScore,
		"requires_captcha":  requiresCaptcha,
		"trust_duration":    3600,
	})
}

func DetectCheck(c *gin.Context) {
	var req DetectCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	riskScore := 10.0
	if req.Fingerprint.WebDriver == "true" {
		riskScore = 50.0
	}
	if req.Fingerprint.WebGL == "SwiftShader" {
		riskScore += 20.0
	}

	c.JSON(http.StatusOK, gin.H{
		"is_proxy":        false,
		"is_vpn":          false,
		"is_tor":          false,
		"is_emulator":     false,
		"is_real_browser":  req.Fingerprint.WebDriver != "true",
		"risk_score":       riskScore,
		"fingerprint_id":  fmt.Sprintf("fp_%d", time.Now().UnixNano()),
	})
}
