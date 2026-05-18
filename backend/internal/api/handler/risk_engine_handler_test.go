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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestRealTimeRiskAssessment(t *testing.T) {
	router := setupTestRouter()
	router.POST("/api/v1/risk/assess", RealTimeRiskAssessment)

	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "正常请求",
			requestBody: map[string]interface{}{
				"fingerprint": "test_fingerprint_001",
				"ip_address": "192.168.1.100",
				"session_id": "session_001",
				"device_info": map[string]interface{}{
					"user_agent": "Mozilla/5.0 Chrome/91.0",
				},
				"behavior_data": map[string]interface{}{
					"score": 85.0,
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				data, ok := response["data"].(map[string]interface{})
				assert.True(t, ok)
				assert.NotEmpty(t, data["request_id"])
				assert.NotEmpty(t, data["action"])
				assert.GreaterOrEqual(t, data["risk_score"].(float64), 0.0)
				assert.LessOrEqual(t, data["risk_score"].(float64), 100.0)
			},
		},
		{
			name: "缺少必需参数",
			requestBody: map[string]interface{}{
				"fingerprint": "test_fingerprint_001",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/risk/assess", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.checkResponse != nil && w.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				tc.checkResponse(t, response)
			}
		})
	}
}

func TestGetRiskProfile(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/profile", GetRiskProfile)

	testCases := []struct {
		name           string
		queryParams     string
		expectedStatus int
	}{
		{
			name:           "查询统一画像-缺少参数",
			queryParams:    "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/risk/profile"
			if tc.queryParams != "" {
				url += "?" + tc.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestGetDRLPolicyStatus(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/drl/status", GetDRLPolicyStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/risk/drl/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), response["code"])

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, data["current_performance"])
	assert.NotNil(t, data["outcomes_summary"])
}

func TestGetMonitoringMetrics(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/monitoring/metrics", GetMonitoringMetrics)

	testCases := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "查询风险指标",
			queryParams: "type=risk&range=1h",
		},
		{
			name:        "查询所有类型",
			queryParams: "range=24h",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/risk/monitoring/metrics"
			if tc.queryParams != "" {
				url += "?" + tc.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
		})
	}
}

func TestGetActiveAlerts(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/monitoring/alerts", GetActiveAlerts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/risk/monitoring/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), response["code"])
}

func TestRecordDRLOutcome(t *testing.T) {
	router := setupTestRouter()
	router.POST("/api/v1/risk/drl/outcome", RecordDRLOutcome)

	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "正常记录",
			requestBody: map[string]interface{}{
				"session_id": "test_session_001",
				"action":     "allow",
				"success":    true,
				"latency_ms": 5,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "缺少必需参数",
			requestBody: map[string]interface{}{
				"action": "allow",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/risk/drl/outcome", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestEvaluateRiskRules(t *testing.T) {
	router := setupTestRouter()
	router.POST("/api/v1/risk/strategy/evaluate", EvaluateRiskRules)

	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "正常评估",
			requestBody: map[string]interface{}{
				"risk_context": map[string]interface{}{
					"ip_request_count": 150.0,
					"mouse_speed":      2500.0,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "缺少风险上下文",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/risk/strategy/evaluate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				data, ok := response["data"].(map[string]interface{})
				assert.True(t, ok)
				assert.NotEmpty(t, data["action"])
			}
		})
	}
}

func TestGetStrategyVersion(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/strategy/version", GetStrategyVersion)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/risk/strategy/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), response["code"])

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, data["version"])
}

func TestGetRiskMetrics(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/monitoring/risk-metrics", GetRiskMetrics)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/risk/monitoring/risk-metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, data["metrics"])
}

func TestGetStrategyPerformance(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/monitoring/strategy-performance", GetStrategyPerformance)

	testCases := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "查询所有策略性能",
			queryParams: "",
		},
		{
			name:        "查询特定策略",
			queryParams: "strategy_name=test_strategy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/risk/monitoring/strategy-performance"
			if tc.queryParams != "" {
				url += "?" + tc.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestGetModelPerformance(t *testing.T) {
	router := setupTestRouter()
	router.GET("/api/v1/risk/monitoring/model-performance", GetModelPerformance)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/risk/monitoring/model-performance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTrainDRLModel(t *testing.T) {
	router := setupTestRouter()
	router.POST("/api/v1/risk/drl/train", TrainDRLModel)

	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "默认批次大小训练",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusOK,
		},
		{
			name: "自定义批次大小训练",
			requestBody: map[string]interface{}{
				"batch_size": 64,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/risk/drl/train", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
