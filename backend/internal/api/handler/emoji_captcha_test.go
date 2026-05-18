package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateEmojiCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/emoji/create", CreateEmojiCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/create", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Errorf("期望响应码 0, 实际得到 %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["sessionId"] == nil || data["targetEmojis"] == nil || data["shuffledEmojis"] == nil {
		t.Error("响应数据缺少必要字段")
	}
}

func TestVerifyEmojiCaptcha_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/emoji/verify", VerifyEmojiCaptcha)

	reqBody := map[string]interface{}{
		"sessionId": "test-session-id",
		"selectedEmojis": []string{"😊", "🎉", "❤️"},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}
}

func TestVerifyEmojiCaptcha_MissingSessionId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/emoji/verify", VerifyEmojiCaptcha)

	reqBody := map[string]interface{}{
		"selectedEmojis": []string{"😊", "🎉", "❤️"},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyEmojiCaptcha_MissingSelectedEmojis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/emoji/verify", VerifyEmojiCaptcha)

	reqBody := map[string]interface{}{
		"sessionId": "test-session-id",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyEmojiCaptcha_EmptySelectedEmojis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/emoji/verify", VerifyEmojiCaptcha)

	reqBody := map[string]interface{}{
		"sessionId":       "test-session-id",
		"selectedEmojis":  []string{},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyEmojiCaptcha_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/emoji/verify", VerifyEmojiCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/emoji/verify", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}
