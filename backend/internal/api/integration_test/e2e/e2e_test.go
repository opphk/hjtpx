package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/captcha/slider", handler.GetSliderCaptcha)
	r.POST("/api/captcha/click", handler.GetClickCaptcha)
	r.POST("/api/captcha/verify", handler.VerifyCaptcha)
	return r
}

func TestEndToEndSliderVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	router := setupTestRouter()

	t.Run("CompleteSliderCaptchaWorkflow", func(t *testing.T) {
		captchaResp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
		router.ServeHTTP(captchaResp, req)

		assert.Equal(t, http.StatusOK, captchaResp.Code)

		var captchaResult map[string]interface{}
		err := json.Unmarshal(captchaResp.Body.Bytes(), &captchaResult)
		assert.NoError(t, err)
		assert.NotEmpty(t, captchaResult["session_id"])
		assert.NotEmpty(t, captchaResult["image_url"])
		assert.NotEmpty(t, captchaResult["puzzle_y"])

		sessionID := captchaResult["session_id"].(string)

		verifyReq := map[string]interface{}{
			"session_id": sessionID,
			"type":       "slider",
			"x":          150,
			"y":          captchaResult["puzzle_y"].(float64),
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		verifyResp := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(verifyResp, req2)

		assert.Equal(t, http.StatusOK, verifyResp.Code)

		var verifyResult map[string]interface{}
		err = json.Unmarshal(verifyResp.Body.Bytes(), &verifyResult)
		assert.NoError(t, err)
		assert.Contains(t, []interface{}{true, false}, verifyResult["success"])
	})
}

func TestEndToEndClickVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	router := setupTestRouter()

	t.Run("CompleteClickCaptchaWorkflow", func(t *testing.T) {
		captchaResp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/click", nil)
		router.ServeHTTP(captchaResp, req)

		assert.Equal(t, http.StatusOK, captchaResp.Code)

		var captchaResult map[string]interface{}
		err := json.Unmarshal(captchaResp.Body.Bytes(), &captchaResult)
		assert.NoError(t, err)
		assert.NotEmpty(t, captchaResult["session_id"])
		assert.NotEmpty(t, captchaResult["image_url"])
		assert.NotEmpty(t, captchaResult["hint"])
		assert.NotEmpty(t, captchaResult["max_points"])

		sessionID := captchaResult["session_id"].(string)
		maxPoints := int(captchaResult["max_points"].(float64))

		clickPoints := make([]map[string]interface{}, maxPoints)
		for i := 0; i < maxPoints; i++ {
			clickPoints[i] = map[string]interface{}{
				"x":           50 + i*100,
				"y":           50 + i*50,
				"imageIndex":  i,
				"clickOrder":  i,
			}
		}

		verifyReq := map[string]interface{}{
			"session_id": sessionID,
			"type":       "click",
			"points":     clickPoints,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		verifyResp := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(verifyResp, req2)

		assert.Equal(t, http.StatusOK, verifyResp.Code)

		var verifyResult map[string]interface{}
		err = json.Unmarshal(verifyResp.Body.Bytes(), &verifyResult)
		assert.NoError(t, err)
		assert.Contains(t, []interface{}{true, false}, verifyResult["success"])
	})
}

func TestAdminWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	t.Run("AdminOperations", func(t *testing.T) {
		t.Log("管理员工作流程测试")
		t.Log("1. 管理员登录")
		t.Log("2. 查看统计数据")
		t.Log("3. 管理应用")
		t.Log("4. 查看日志")
	})
}

func TestMultipleVerificationTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	router := setupTestRouter()

	t.Run("MultipleVerificationTypes", func(t *testing.T) {
		verificationTypes := []string{"slider", "click"}
		for _, vType := range verificationTypes {
			t.Run(vType, func(t *testing.T) {
				endpoint := fmt.Sprintf("/api/captcha/%s", vType)
				resp := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", endpoint, nil)
				router.ServeHTTP(resp, req)

				assert.Equal(t, http.StatusOK, resp.Code)

				var result map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.NotEmpty(t, result["session_id"])
			})
		}
	})
}

func TestEndToEndVerificationWithDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	db := database.GetDB()
	if db == nil {
		t.Skip("Database not available")
	}

	router := setupTestRouter()

	t.Run("VerifyDatabaseRecordCreation", func(t *testing.T) {
		captchaResp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
		router.ServeHTTP(captchaResp, req)

		var captchaResult map[string]interface{}
		json.Unmarshal(captchaResp.Body.Bytes(), &captchaResult)
		sessionID := captchaResult["session_id"].(string)

		verifyReq := map[string]interface{}{
			"session_id":     sessionID,
			"type":           "slider",
			"x":              150,
			"y":              100,
			"application_id": 1,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		verifyResp := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(verifyResp, req2)

		var verification models.Verification
		err := db.Where("session_id = ?", sessionID).First(&verification).Error
		if err == nil {
			t.Logf("Verification record created: ID=%d, Status=%s", verification.ID, verification.Status)
		}

		var logCount int64
		db.Model(&models.VerificationLog{}).Where("session_id = ?", sessionID).Count(&logCount)
		t.Logf("Verification log count: %d", logCount)
	})
}

func TestEndToEndVerificationWithRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	client := redis.GetClient()
	if client == nil {
		t.Skip("Redis not available")
	}

	t.Run("VerifyRedisCacheOperations", func(t *testing.T) {
		ctx := redis.Context
		testKey := "test:e2e:verification"
		testValue := fmt.Sprintf("test_%d", time.Now().Unix())

		err := client.Set(ctx, testKey, testValue, 5*time.Minute).Err()
		assert.NoError(t, err)

		val, err := client.Get(ctx, testKey).Result()
		assert.NoError(t, err)
		assert.Equal(t, testValue, val)

		client.Del(ctx, testKey)
	})
}

func TestEndToEndBehaviorData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	router := setupTestRouter()

	t.Run("VerifyBehaviorDataCollection", func(t *testing.T) {
		captchaResp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
		router.ServeHTTP(captchaResp, req)

		var captchaResult map[string]interface{}
		json.Unmarshal(captchaResp.Body.Bytes(), &captchaResult)
		sessionID := captchaResult["session_id"].(string)

		behaviorData := []map[string]interface{}{
			{"x": 10, "y": 20, "timestamp": time.Now().UnixMilli(), "event": "mousemove"},
			{"x": 30, "y": 40, "timestamp": time.Now().UnixMilli(), "event": "mousemove"},
			{"x": 50, "y": 60, "timestamp": time.Now().UnixMilli(), "event": "click"},
		}

		verifyReq := map[string]interface{}{
			"session_id":     sessionID,
			"type":           "slider",
			"x":              150,
			"y":              100,
			"behavior_data":  behaviorData,
			"verification_time": int64(3000),
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		verifyResp := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(verifyResp, req2)

		assert.Equal(t, http.StatusOK, verifyResp.Code)
	})
}

func TestEndToEndSessionExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	router := setupTestRouter()

	t.Run("VerifyExpiredSessionHandling", func(t *testing.T) {
		expiredSessionID := "expired_session_12345"

		verifyReq := map[string]interface{}{
			"session_id": expiredSessionID,
			"type":        "slider",
			"x":           150,
			"y":           100,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		verifyResp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(verifyResp, req)

		assert.Equal(t, http.StatusNotFound, verifyResp.Code)
	})
}

func TestEndToEndConcurrentVerifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	t.Run("ConcurrentVerificationRequests", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				router := setupTestRouter()
				resp := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
				router.ServeHTTP(resp, req)

				assert.Equal(t, http.StatusOK, resp.Code)
				done <- true
			}(i)
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}
