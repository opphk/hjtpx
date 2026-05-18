package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGenerateGestureCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/captcha/gesture", GenerateGestureCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/gesture", nil)
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
	if data["id"] == nil || data["pattern"] == nil || data["hint"] == nil {
		t.Error("响应数据缺少必要字段")
	}
}

func TestVerifyGestureCaptcha_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/gesture/verify", VerifyGestureCaptcha)

	reqBody := map[string]string{
		"id":      "gesture-123",
		"pattern": "1-2-3-5-7",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer(body))
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

	data := resp["data"].(map[string]interface{})
	if data["success"] != true {
		t.Errorf("期望验证成功, 实际得到 %v", data["success"])
	}
}

func TestVerifyGestureCaptcha_WrongPattern(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/gesture/verify", VerifyGestureCaptcha)

	reqBody := map[string]string{
		"id":      "gesture-123",
		"pattern": "1-3-5-7",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer(body))
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

	data := resp["data"].(map[string]interface{})
	if data["success"] != false {
		t.Errorf("期望验证失败, 实际得到 %v", data["success"])
	}
}

func TestVerifyGestureCaptcha_MissingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/gesture/verify", VerifyGestureCaptcha)

	reqBody := map[string]string{
		"id": "gesture-123",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyGestureCaptcha_EmptyPattern(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/gesture/verify", VerifyGestureCaptcha)

	reqBody := map[string]string{
		"id":      "gesture-123",
		"pattern": "",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyGestureCaptcha_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/captcha/gesture/verify", VerifyGestureCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}
