package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetAllConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - returns config",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/config", GetAllConfig)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/config", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "system")
			assert.Contains(t, resp, "security")
		})
	}
}

func TestUpdateConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - update config",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/admin/config", UpdateConfig)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/api/v1/admin/config", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestExportConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - export config",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/config/export", ExportConfig)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/config/export", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}

func TestResetConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - reset config",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/config/reset", ResetConfig)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/config/reset", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGenerateJigsawCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - generate jigsaw captcha",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/captcha/jigsaw", GenerateJigsawCaptcha)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/captcha/jigsaw", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestVerifyJigsawCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - verify jigsaw captcha",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/captcha/jigsaw/verify", VerifyJigsawCaptcha)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/captcha/jigsaw/verify", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestVerifyEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - verify email",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/user/verify-email", VerifyEmail)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/user/verify-email", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestVerifyPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - verify phone",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/user/verify-phone", VerifyPhone)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/user/verify-phone", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list users",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/users", ListUsers)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - create user",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/users", CreateUser)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/users", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
	}{
		{
			name:           "success - update user",
			userID:        "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/admin/users/:id", UpdateUser)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/api/v1/admin/users/"+tt.userID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
	}{
		{
			name:           "success - delete user",
			userID:        "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.DELETE("/api/v1/admin/users/:id", DeleteUser)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/v1/admin/users/"+tt.userID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestApproveApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID         string
		expectedStatus int
	}{
		{
			name:           "success - approve application",
			appID:         "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/applications/:id/approve", ApproveApplication)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/applications/"+tt.appID+"/approve", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRejectApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID         string
		expectedStatus int
	}{
		{
			name:           "success - reject application",
			appID:         "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/applications/:id/reject", RejectApplication)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/applications/"+tt.appID+"/reject", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListAPIKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list api keys",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/api-keys", ListAPIKeys)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/api-keys", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCreateAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - create api key",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/api-keys", CreateAPIKey)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/api-keys", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDeleteAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		keyID          string
		expectedStatus int
	}{
		{
			name:           "success - delete api key",
			keyID:         "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.DELETE("/api/v1/admin/api-keys/:id", DeleteAPIKey)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/v1/admin/api-keys/"+tt.keyID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListVerifications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list verifications",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/verifications", ListVerifications)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/verifications", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetVerificationDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		verifID       string
		expectedStatus int
	}{
		{
			name:           "success - get verification detail",
			verifID:       "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/verifications/:id", GetVerificationDetail)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/verifications/"+tt.verifID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestReviewVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		verifID       string
		expectedStatus int
	}{
		{
			name:           "success - review verification",
			verifID:       "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/verifications/:id/review", ReviewVerification)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/verifications/"+tt.verifID+"/review", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAddToBlacklist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - add to blacklist",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/blacklist/add", AddToBlacklist)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/blacklist/add", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRemoveFromBlacklist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - remove from blacklist",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/blacklist/remove", RemoveFromBlacklist)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/blacklist/remove", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - get settings",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/settings", GetSettings)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/settings", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUpdateSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - update settings",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/admin/settings", UpdateSettings)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/api/v1/admin/settings", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListRiskEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list risk events",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/risk-events", ListRiskEvents)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/risk-events", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListTraces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list traces",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/traces", ListTraces)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/traces", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestEnableAlertRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ruleID         string
		expectedStatus int
	}{
		{
			name:           "success - enable alert rule",
			ruleID:        "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/alert-rules/:id/enable", EnableAlertRule)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/alert-rules/"+tt.ruleID+"/enable", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDisableAlertRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ruleID         string
		expectedStatus int
	}{
		{
			name:           "success - disable alert rule",
			ruleID:        "1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/alert-rules/:id/disable", DisableAlertRule)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/admin/alert-rules/"+tt.ruleID+"/disable", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListAlertHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list alert history",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alerts/history", ListAlertHistory)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alerts/history", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestMissingHandlersIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("all missing handlers return success", func(t *testing.T) {
		handlers := map[string]struct {
			method string
			path   string
			fn     gin.HandlerFunc
		}{
			"GetStats":         {"GET", "/api/v1/admin/stats", GetStats},
			"Register":         {"POST", "/api/v1/user/register", Register},
			"GetProfile":       {"GET", "/api/v1/user/profile", GetProfile},
			"UpdateProfile":    {"PUT", "/api/v1/user/profile", UpdateProfile},
			"RefreshToken":     {"POST", "/api/v1/user/refresh-token", RefreshToken},
			"UpdateUserStatus": {"PUT", "/api/v1/admin/users/1/status", UpdateUserStatus},
			"ResetUserPassword":{"POST", "/api/v1/admin/users/1/reset-password", ResetUserPassword},
			"RegenerateAPIKey": {"POST", "/api/v1/admin/api-keys/1/regenerate", RegenerateAPIKey},
			"GetRiskEventDetail": {"GET", "/api/v1/admin/risk-events/1", GetRiskEventDetail},
			"GetTraceDetail":   {"GET", "/api/v1/admin/traces/1", GetTraceDetail},
		}

		for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			r := gin.New()
			switch h.method {
			case "GET":
				r.GET(h.path, h.fn)
			case "POST":
				r.POST(h.path, h.fn)
			case "PUT":
				r.PUT(h.path, h.fn)
			case "DELETE":
				r.DELETE(h.path, h.fn)
			}

			var req *http.Request
			req, _ = http.NewRequest(h.method, h.path, nil)

				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code, "Handler %s should return 200", name)
			})
		}
	})
}
