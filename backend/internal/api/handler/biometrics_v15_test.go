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

func TestNewBiometricsV15Handler(t *testing.T) {
	handler := NewBiometricsV15Handler()
	if handler == nil {
		t.Error("NewBiometricsV15Handler 返回了 nil")
	}
	if handler.biometricsService == nil {
		t.Error("biometricsService 未正确初始化")
	}
}

func TestGetBiometricsV15Handler(t *testing.T) {
	handler := GetBiometricsV15Handler()
	if handler == nil {
		t.Error("GetBiometricsV15Handler 返回了 nil")
	}
}

func TestRegisterMultimodalBiometricProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/register", RegisterMultimodalBiometricProfile)

	mouseData := &service.MousePressureData{
		PressureData: []service.PressurePoint{
			{Type: "mousedown", X: 100, Y: 200, Timestamp: 1234567890, Pressure: 0.7, Force: 6.87},
			{Type: "mouseup", X: 105, Y: 205, Timestamp: 1234567900, Pressure: 0.3, Force: 2.94},
		},
		PressureAnalysis: &service.PressureAnalysis{
			AveragePressure: 0.5,
			PressureStd:     0.2,
			MaxPressure:     0.8,
			MinPressure:     0.2,
			ForceStd:        1.5,
		},
		ClickAnalysis: &service.ClickAnalysis{
			ClickCount:       10,
			AvgClickDuration: 150.0,
		},
		MovementAnalysis: &service.MovementAnalysis{
			AvgSpeed:        0.5,
			SpeedStd:        0.2,
			MaxSpeed:        1.2,
			MovementEntropy: 3.5,
		},
	}

	reqBody := RegisterMultimodalRequest{
		UserID: "user-test-123",
		BiometricData: &service.MultimodalBiometricData{
			UserID:        "user-test-123",
			MousePressure: mouseData,
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Errorf("期望响应码 0, 实际得到 %v", resp["code"])
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Error("响应数据格式不正确")
	}

	if data["message"] != "多模态生物特征档案注册成功" {
		t.Errorf("期望消息 '多模态生物特征档案注册成功', 实际得到 %v", data["message"])
	}
}

func TestRegisterMultimodalBiometricProfile_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/register", RegisterMultimodalBiometricProfile)

	reqBody := RegisterMultimodalRequest{
		UserID: "",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestRegisterMultimodalBiometricProfile_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/register", RegisterMultimodalBiometricProfile)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/register", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestVerifyMultimodalBiometrics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/verify", VerifyMultimodalBiometrics)

	mouseData := &service.MousePressureData{
		PressureAnalysis: &service.PressureAnalysis{
			AveragePressure: 0.5,
			PressureStd:     0.2,
			MaxPressure:     0.8,
		},
		MovementAnalysis: &service.MovementAnalysis{
			AvgSpeed: 0.5,
			SpeedStd: 0.2,
		},
	}

	reqBody := VerifyMultimodalRequest{
		UserID: "user-test-123",
		BiometricData: &service.MultimodalBiometricData{
			UserID:        "user-test-123",
			MousePressure: mouseData,
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Errorf("期望响应码 0, 实际得到 %v", resp["code"])
	}
}

func TestVerifyMultimodalBiometrics_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/verify", VerifyMultimodalBiometrics)

	reqBody := VerifyMultimodalRequest{
		UserID: "",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestGetBiometricsCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/biometrics/v15/capabilities", GetBiometricsCapabilities)

	req, _ := http.NewRequest("GET", "/api/v1/biometrics/v15/capabilities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Error("响应数据格式不正确")
	}

	if data["version"] != "15.0" {
		t.Errorf("期望版本 '15.0', 实际得到 %v", data["version"])
	}
}

func TestGetMultimodalProfile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/biometrics/v15/profile", GetMultimodalProfile)

	req, _ := http.NewRequest("GET", "/api/v1/biometrics/v15/profile?user_id=nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 实际得到 %d", w.Code)
	}
}

func TestGetMultimodalProfile_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/biometrics/v15/profile", GetMultimodalProfile)

	req, _ := http.NewRequest("GET", "/api/v1/biometrics/v15/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestDeleteMultimodalProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/api/v1/biometrics/v15/profile", DeleteMultimodalProfile)

	handler := GetBiometricsV15Handler()
	handler.biometricsService.RegisterMultimodalProfile("user-to-delete", &service.MultimodalBiometricData{
		UserID: "user-to-delete",
	})

	req, _ := http.NewRequest("DELETE", "/api/v1/biometrics/v15/profile?user_id=user-to-delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestDeleteMultimodalProfile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/api/v1/biometrics/v15/profile", DeleteMultimodalProfile)

	req, _ := http.NewRequest("DELETE", "/api/v1/biometrics/v15/profile?user_id=nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 实际得到 %d", w.Code)
	}
}

func TestFusionVerifyBiometrics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/fusion/verify", FusionVerifyBiometrics)

	reqBody := FusionVerifyRequest{
		UserID: "user-test-456",
		BiometricData: &service.MultimodalBiometricData{
			UserID: "user-test-456",
			MousePressure: &service.MousePressureData{
				PressureAnalysis: &service.PressureAnalysis{
					AveragePressure: 0.5,
				},
			},
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/fusion/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d, body: %s", w.Code, w.Body.String())
	}
}

func TestAnalyzeBiometricData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/analyze", AnalyzeBiometricData)

	mouseData := &service.MousePressureData{
		PressureAnalysis: &service.PressureAnalysis{
			AveragePressure: 0.5,
			PressureStd:     0.2,
		},
	}

	reqBody := struct {
		BiometricData *service.MultimodalBiometricData `json:"biometric_data"`
	}{
		BiometricData: &service.MultimodalBiometricData{
			SessionID:     "session-123",
			MousePressure: mouseData,
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestCompareBiometricProfiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/compare", CompareBiometricProfiles)

	handler := GetBiometricsV15Handler()
	handler.biometricsService.RegisterMultimodalProfile("user-compare-1", &service.MultimodalBiometricData{
		UserID: "user-compare-1",
		MousePressure: &service.MousePressureData{
			PressureAnalysis: &service.PressureAnalysis{AveragePressure: 0.5},
		},
	})
	handler.biometricsService.RegisterMultimodalProfile("user-compare-2", &service.MultimodalBiometricData{
		UserID: "user-compare-2",
		MousePressure: &service.MousePressureData{
			PressureAnalysis: &service.PressureAnalysis{AveragePressure: 0.6},
		},
	})

	reqBody := struct {
		UserID1 string `json:"user_id_1"`
		UserID2 string `json:"user_id_2"`
	}{
		UserID1: "user-compare-1",
		UserID2: "user-compare-2",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/compare", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestRegisterMousePressureProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/register/mouse", RegisterMousePressureProfile)

	mouseData := &service.MousePressureData{
		PressureAnalysis: &service.PressureAnalysis{
			AveragePressure: 0.5,
		},
	}

	reqBody := RegisterMousePressureRequest{
		UserID:    "user-mouse-test",
		MouseData: mouseData,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/register/mouse", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestRegisterTouchForceProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/register/touch", RegisterTouchForceProfile)

	touchData := &service.TouchForceData{
		ForceAnalysis: &service.TouchForceAnalysis{
			TouchCount: 20,
			AvgForce:   0.6,
		},
		SwipeAnalysis: &service.SwipeAnalysis{
			SwipeCount: 5,
		},
	}

	reqBody := RegisterTouchForceRequest{
		UserID:    "user-touch-test",
		TouchData: touchData,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/register/touch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestRegisterEyeTrackingProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/biometrics/v15/register/eye", RegisterEyeTrackingProfile)

	eyeData := &service.EyeTrackingData{
		GazeAnalysis: &service.GazeAnalysis{
			GazeCount: 100,
			AvgX:      500,
			AvgY:      300,
		},
		BlinkAnalysis: &service.BlinkAnalysis{
			BlinkCount:       15,
			BlinkRate:        12.0,
			AvgBlinkDuration: 150.0,
		},
	}

	reqBody := RegisterEyeTrackingRequest{
		UserID:          "user-eye-test",
		EyeTrackingData: eyeData,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/biometrics/v15/register/eye", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}
