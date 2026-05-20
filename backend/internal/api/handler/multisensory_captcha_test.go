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

func setupMultisensoryTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

func TestCreateMultisensoryCaptcha(t *testing.T) {
	router := setupMultisensoryTestRouter()
	router.POST("/api/v1/captcha/multisensory/create", CreateMultisensoryCaptcha)

	reqBody := MultisensoryCaptchaCreateRequest{
		Types:      []string{"visual", "audio", "tactile"},
		VisualType: "slider",
		Language:   "zh-CN",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/captcha/multisensory/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), response["code"])
}

func TestCreateMultisensoryCaptcha_EmptyTypes(t *testing.T) {
	router := setupMultisensoryTestRouter()
	router.POST("/api/v1/captcha/multisensory/create", CreateMultisensoryCaptcha)

	reqBody := MultisensoryCaptchaCreateRequest{}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/captcha/multisensory/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), response["code"])
}

func TestVerifyMultisensoryCaptcha_MissingSession(t *testing.T) {
	router := setupMultisensoryTestRouter()
	router.POST("/api/v1/captcha/multisensory/verify", VerifyMultisensoryCaptcha)

	reqBody := MultisensoryCaptchaVerifyRequest{
		SessionID: "invalid-session-id",
		Answers:   map[string]string{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/captcha/multisensory/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVerifyMultisensoryCaptcha_InvalidParams(t *testing.T) {
	router := setupMultisensoryTestRouter()
	router.POST("/api/v1/captcha/multisensory/verify", VerifyMultisensoryCaptcha)

	req := httptest.NewRequest("POST", "/api/v1/captcha/multisensory/verify", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetMultisensoryCaptchaStatus_InvalidSession(t *testing.T) {
	router := setupMultisensoryTestRouter()
	router.GET("/api/v1/captcha/multisensory/status/:session_id", GetMultisensoryCaptchaStatus)

	req := httptest.NewRequest("GET", "/api/v1/captcha/multisensory/status/invalid-session", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetMultisensoryCaptchaStatus_MissingSession(t *testing.T) {
	router := setupMultisensoryTestRouter()
	router.GET("/api/v1/captcha/multisensory/status/:session_id", GetMultisensoryCaptchaStatus)

	req := httptest.NewRequest("GET", "/api/v1/captcha/multisensory/status/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
