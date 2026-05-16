package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
	jwt.InitJWT("test-integration-secret")
}

func TestCaptchaFlow(t *testing.T) {
	r := gin.New()

	r.GET("/captcha/slider", handler.GetSliderCaptcha)
	r.GET("/captcha/click", handler.GetClickCaptcha)
	r.POST("/captcha/verify", handler.VerifyCaptcha)

	sliderReq, _ := http.NewRequest("GET", "/captcha/slider", nil)
	sliderW := httptest.NewRecorder()
	r.ServeHTTP(sliderW, sliderReq)

	assert.Equal(t, http.StatusOK, sliderW.Code)

	var sliderResp map[string]interface{}
	err := json.Unmarshal(sliderW.Body.Bytes(), &sliderResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, sliderResp["session_id"])

	sessionID := sliderResp["session_id"].(string)

	verifyReq := handler.VerifyRequest{
		SessionID: sessionID,
		Type:      "slider",
		X:         100,
		Y:         100,
	}

	body, _ := json.Marshal(verifyReq)
	verifyW := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(verifyW, req)

	assert.Equal(t, http.StatusOK, verifyW.Code)

	var verifyResp map[string]interface{}
	err = json.Unmarshal(verifyW.Body.Bytes(), &verifyResp)
	assert.NoError(t, err)
	assert.Contains(t, verifyResp, "success")
}

func TestClickCaptchaFlow(t *testing.T) {
	r := gin.New()

	r.GET("/captcha/click", handler.GetClickCaptcha)
	r.POST("/captcha/verify", handler.VerifyCaptcha)

	clickReq, _ := http.NewRequest("GET", "/captcha/click", nil)
	clickW := httptest.NewRecorder()
	clickHttpReq, _ := http.NewRequest("GET", "/captcha/click", nil)
	r.ServeHTTP(clickW, clickHttpReq)

	var clickResp map[string]interface{}
	err := json.Unmarshal(clickW.Body.Bytes(), &clickResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, clickResp["session_id"])

	sessionID := clickResp["session_id"].(string)
	maxPoints := int(clickResp["max_points"].(float64))

	points := make([]handler.ClickPoint, maxPoints)
	for i := 0; i < maxPoints; i++ {
		points[i] = handler.ClickPoint{
			X:          100 + i*50,
			Y:          100 + i*50,
			ImageIndex: i,
			ClickOrder: i,
		}
	}

	verifyReq := handler.VerifyRequest{
		SessionID: sessionID,
		Type:      "click",
		Points:    points,
	}

	body, _ := json.Marshal(verifyReq)
	verifyW := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(verifyW, req)

	assert.Equal(t, http.StatusOK, verifyW.Code)
}

func TestCaptchaVerificationInvalidSession(t *testing.T) {
	r := gin.New()
	r.POST("/captcha/verify", handler.VerifyCaptcha)

	verifyReq := handler.VerifyRequest{
		SessionID: "invalid-session",
		Type:      "slider",
		X:         100,
		Y:         100,
	}

	body, _ := json.Marshal(verifyReq)
	verifyW := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(verifyW, req)

	var resp map[string]interface{}
	json.Unmarshal(verifyW.Body.Bytes(), &resp)
	assert.Equal(t, http.StatusNotFound, verifyW.Code)
}

func TestMultipleCaptchaSessions(t *testing.T) {
	r := gin.New()
	r.GET("/captcha/slider", handler.GetSliderCaptcha)

	sessions := make([]string, 5)
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/captcha/slider", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		sessions[i] = resp["session_id"].(string)
	}

	sessionMap := make(map[string]bool)
	for _, sessionID := range sessions {
		if sessionMap[sessionID] {
			t.Errorf("Duplicate session ID found: %s", sessionID)
		}
		sessionMap[sessionID] = true
	}
}

func TestCaptchaWithBehaviorData(t *testing.T) {
	r := gin.New()
	r.GET("/captcha/slider", handler.GetSliderCaptcha)
	r.POST("/captcha/verify", handler.VerifyCaptcha)

	sliderReq, _ := http.NewRequest("GET", "/captcha/slider", nil)
	sliderW := httptest.NewRecorder()
	r.ServeHTTP(sliderW, sliderReq)

	var sliderResp map[string]interface{}
	json.Unmarshal(sliderW.Body.Bytes(), &sliderResp)
	sessionID := sliderResp["session_id"].(string)

	behaviorData := []map[string]interface{}{
		{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"},
		{"x": 110, "y": 210, "timestamp": 1100, "event": "mousemove"},
		{"x": 120, "y": 220, "timestamp": 1200, "event": "click"},
	}

	verifyReq := handler.VerifyRequest{
		SessionID:      sessionID,
		Type:           "slider",
		X:              100,
		Y:              100,
		BehaviorData:   convertToBehaviorData(behaviorData),
		VerificationTime: 500,
	}

	body, _ := json.Marshal(verifyReq)
	verifyW := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(verifyW, req)

	assert.Equal(t, http.StatusOK, verifyW.Code)
}

func convertToBehaviorData(data []map[string]interface{}) []handler.BehaviorDataPoint {
	result := make([]handler.BehaviorDataPoint, len(data))
	for i, d := range data {
		result[i] = handler.BehaviorDataPoint{
			X:         int(d["x"].(float64)),
			Y:         int(d["y"].(float64)),
			Timestamp: int64(d["timestamp"].(float64)),
			Event:     d["event"].(string),
		}
	}
	return result
}

func TestApplicationManagementFlow(t *testing.T) {
	r := gin.New()

	r.GET("/admin/applications", handler.ListApplications)
	r.POST("/admin/applications", handler.CreateApplication)
	r.PUT("/admin/applications/:id", handler.UpdateApplication)
	r.DELETE("/admin/applications/:id", handler.DeleteApplication)

	listReq, _ := http.NewRequest("GET", "/admin/applications", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, listW.Code)

	invalidUpdateReq, _ := http.NewRequest("PUT", "/admin/applications/invalid", nil)
	invalidUpdateW := httptest.NewRecorder()
	r.ServeHTTP(invalidUpdateW, invalidUpdateReq)

	assert.Equal(t, http.StatusBadRequest, invalidUpdateW.Code)

	invalidDeleteReq, _ := http.NewRequest("DELETE", "/admin/applications/invalid", nil)
	invalidDeleteW := httptest.NewRecorder()
	r.ServeHTTP(invalidDeleteW, invalidDeleteReq)

	assert.Equal(t, http.StatusBadRequest, invalidDeleteW.Code)
}

func TestLoginLogoutFlow(t *testing.T) {
	r := gin.New()
	r.POST("/admin/login", handler.Login)
	r.POST("/admin/logout", handler.Logout)

	logoutReq, _ := http.NewRequest("POST", "/admin/logout", nil)
	logoutW := httptest.NewRecorder()
	r.ServeHTTP(logoutW, logoutReq)

	assert.Equal(t, http.StatusOK, logoutW.Code)
}

func TestJWTTokenGenerationAndValidation(t *testing.T) {
	token, err := jwt.GenerateToken(1, "testadmin")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwt.ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.AdminID)
	assert.Equal(t, "testadmin", claims.Username)
}

func TestJWTTokenWithDifferentUsers(t *testing.T) {
	users := []struct {
		id       uint
		username string
	}{
		{1, "admin1"},
		{2, "admin2"},
		{3, "admin3"},
	}

	tokens := make([]string, len(users))
	for i, user := range users {
		token, err := jwt.GenerateToken(user.id, user.username)
		assert.NoError(t, err)
		tokens[i] = token
	}

	for i, user := range users {
		claims, err := jwt.ParseToken(tokens[i])
		assert.NoError(t, err)
		assert.Equal(t, user.id, claims.AdminID)
		assert.Equal(t, user.username, claims.Username)
	}
}

func TestInvalidJWTToken(t *testing.T) {
	invalidTokens := []string{
		"",
		"invalid-token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
	}

	for _, token := range invalidTokens {
		_, err := jwt.ParseToken(token)
		assert.Error(t, err, fmt.Sprintf("Expected error for token: %s", token))
	}
}

func TestProtectedEndpointWithoutAuth(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	})
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProtectedEndpointWithValidAuth(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	})
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	token, _ := jwt.GenerateToken(1, "testadmin")

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
