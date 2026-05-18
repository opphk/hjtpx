package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

func TestNewBiometricsHandler(t *testing.T) {
	handler := NewBiometricsHandler()
	if handler == nil {
		t.Error("NewBiometricsHandler 返回了 nil")
	}
	if handler.biometricsService == nil {
		t.Error("biometricsService 未正确初始化")
	}
}

func TestGetBiometricsHandler(t *testing.T) {
	handler := GetBiometricsHandler()
	if handler == nil {
		t.Error("GetBiometricsHandler 返回了 nil")
	}
}

func TestRegisterBiometricProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/register", RegisterBiometricProfile)

	keyboardSample := &service.KeyboardSample{
		KeyEvents: []service.KeyEvent{
			{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
			{Key: "a", Type: "keyup", Timestamp: 1100, KeyCode: 65},
		},
		Timestamp: 1000,
	}

	reqBody := RegisterBiometricProfileRequest{
		UserID:         "user-123",
		KeyboardSample: keyboardSample,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/register", bytes.NewBuffer(body))
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

	if resp["code"].(float64) != 0 {
		t.Errorf("期望响应码 0, 实际得到 %v", resp["code"])
	}
}

func TestRegisterBiometricProfile_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/register", RegisterBiometricProfile)

	reqBody := RegisterBiometricProfileRequest{
		UserID: "",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestRegisterBiometricProfile_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/register", RegisterBiometricProfile)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/register", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyBiometrics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/verify", VerifyBiometrics)

	keyboardSample := &service.KeyboardSample{
		KeyEvents: []service.KeyEvent{
			{Key: "a", Type: "keydown", Timestamp: 1000, KeyCode: 65},
			{Key: "a", Type: "keyup", Timestamp: 1100, KeyCode: 65},
		},
		Timestamp: 1000,
	}

	reqBody := VerifyBiometricsRequest{
		UserID:         "user-123",
		KeyboardSample: keyboardSample,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/verify", bytes.NewBuffer(body))
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

	if resp["code"].(float64) != 0 {
		t.Errorf("期望响应码 0, 实际得到 %v", resp["code"])
	}
}

func TestVerifyBiometrics_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/verify", VerifyBiometrics)

	reqBody := VerifyBiometricsRequest{
		UserID: "",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyBiometrics_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/verify", VerifyBiometrics)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/verify", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}
