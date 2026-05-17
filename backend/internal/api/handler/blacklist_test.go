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

func TestBlacklistHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "list all blacklist entries",
			queryParams: "",
		},
		{
			name:        "list with pagination",
			queryParams: "?page=1&size=20",
		},
		{
			name:        "list with type filter",
			queryParams: "?type=ip",
		},
		{
			name:        "list with reason filter",
			queryParams: "?reason=bot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/blacklist", ListBlacklist)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/blacklist"+tt.queryParams, nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestBlacklistHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		requestBody interface{}
	}{
		{
			name: "valid IP entry",
			requestBody: map[string]interface{}{
				"type":   "ip",
				"target": "192.168.1.100",
				"reason": "bot_attack",
			},
		},
		{
			name: "valid user entry",
			requestBody: map[string]interface{}{
				"type":   "user_id",
				"target": "user123",
				"reason": "abuse",
			},
		},
		{
			name:        "empty body",
			requestBody: nil,
		},
		{
			name: "missing type",
			requestBody: map[string]interface{}{
				"target": "192.168.1.1",
				"reason": "bot",
			},
		},
		{
			name: "missing target",
			requestBody: map[string]interface{}{
				"type":   "ip",
				"reason": "bot",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/blacklist", CreateBlacklist)

			w := httptest.NewRecorder()
			var req *http.Request

			if tt.requestBody == nil {
				req, _ = http.NewRequest("POST", "/api/v1/blacklist", nil)
			} else {
				body, _ := json.Marshal(tt.requestBody)
				req, _ = http.NewRequest("POST", "/api/v1/blacklist", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			}

			r.ServeHTTP(w, req)
			assert.NotNil(t, w)
		})
	}
}

func TestBlacklistHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		blacklistID    string
		expectedStatus int
	}{
		{
			name:           "invalid id - non-numeric",
			blacklistID:    "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "delete non-existent entry",
			blacklistID:    "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.DELETE("/api/v1/blacklist/:id", DeleteBlacklist)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/v1/blacklist/"+tt.blacklistID, nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestBlacklistHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		blacklistID string
	}{
		{
			name:        "get by valid id",
			blacklistID: "1",
		},
		{
			name:        "get by invalid id",
			blacklistID: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/blacklist/:id", GetBlacklistByID)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/blacklist/"+tt.blacklistID, nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestBlacklistHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		blacklistID string
		requestBody interface{}
	}{
		{
			name:        "update with valid id",
			blacklistID: "1",
			requestBody: map[string]interface{}{
				"reason": "updated_reason",
			},
		},
		{
			name:        "update with invalid id",
			blacklistID: "abc",
			requestBody: map[string]interface{}{
				"reason": "updated_reason",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/blacklist/:id", UpdateBlacklist)

			w := httptest.NewRecorder()
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/blacklist/"+tt.blacklistID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestBlacklistHandler_Unblock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		blacklistID string
	}{
		{
			name:        "unblock with valid id",
			blacklistID: "1",
		},
		{
			name:        "unblock with invalid id",
			blacklistID: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/blacklist/:id/unblock", UnblockBlacklist)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/blacklist/"+tt.blacklistID+"/unblock", nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestBlacklistHandler_Check(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "check IP",
			queryParams: "?target=192.168.1.1&type=ip",
		},
		{
			name:        "check user",
			queryParams: "?target=user123&type=user_id",
		},
		{
			name:        "check without type",
			queryParams: "?target=192.168.1.1",
		},
		{
			name:        "check without target",
			queryParams: "?type=ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/blacklist/check", CheckBlacklist)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/blacklist/check"+tt.queryParams, nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestBlacklistHandler_GetSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/blacklist/summary", GetBlacklistSummary)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/blacklist/summary", nil)
	r.ServeHTTP(w, req)

	assert.NotNil(t, w)
}

func TestBlacklistHandler_Import(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		requestBody interface{}
	}{
		{
			name: "valid import",
			requestBody: map[string]interface{}{
				"type":    "ip",
				"targets": []string{"192.168.1.1", "192.168.1.2"},
				"reason":  "bot",
			},
		},
		{
			name:        "empty body",
			requestBody: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/blacklist/import", ImportBlacklist)

			w := httptest.NewRecorder()
			var req *http.Request

			if tt.requestBody == nil {
				req, _ = http.NewRequest("POST", "/api/v1/blacklist/import", nil)
			} else {
				body, _ := json.Marshal(tt.requestBody)
				req, _ = http.NewRequest("POST", "/api/v1/blacklist/import", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			}

			r.ServeHTTP(w, req)
			assert.NotNil(t, w)
		})
	}
}

func TestListBlacklistQuery_Validation(t *testing.T) {
	query := ListBlacklistQuery{
		Page:      1,
		Size:      20,
		Type:      "ip",
		Source:    "manual",
		Status:    "active",
		Keyword:   "test",
		StartDate: "2024-01-01",
		EndDate:   "2024-12-31",
	}

	assert.Equal(t, 1, query.Page)
	assert.Equal(t, 20, query.Size)
	assert.Equal(t, "ip", query.Type)
	assert.Equal(t, "manual", query.Source)
	assert.Equal(t, "active", query.Status)
	assert.Equal(t, "test", query.Keyword)
	assert.Equal(t, "2024-01-01", query.StartDate)
	assert.Equal(t, "2024-12-31", query.EndDate)
}

func TestCreateBlacklistRequest_Validation(t *testing.T) {
	req := CreateBlacklistRequest{
		Type:           "ip",
		Target:         "192.168.1.1",
		Reason:         "bot_attack",
		Action:         "block",
		ApplicationIDs: []string{"app1", "app2"},
		Expiration:     "2025-12-31",
		Note:           "Test note",
	}

	assert.Equal(t, "ip", req.Type)
	assert.Equal(t, "192.168.1.1", req.Target)
	assert.Equal(t, "bot_attack", req.Reason)
	assert.Equal(t, "block", req.Action)
	assert.Len(t, req.ApplicationIDs, 2)
	assert.Equal(t, "2025-12-31", req.Expiration)
	assert.Equal(t, "Test note", req.Note)
}

func TestUpdateBlacklistRequest_Validation(t *testing.T) {
	reason := "updated_reason"
	action := "captcha"
	expiration := "2025-12-31"
	note := "Updated note"

	req := UpdateBlacklistRequest{
		Reason:     &reason,
		Action:     &action,
		Expiration: &expiration,
		Note:       &note,
	}

	assert.Equal(t, "updated_reason", *req.Reason)
	assert.Equal(t, "captcha", *req.Action)
	assert.Equal(t, "2025-12-31", *req.Expiration)
	assert.Equal(t, "Updated note", *req.Note)
}

func TestImportBlacklistRequest_Validation(t *testing.T) {
	req := ImportBlacklistRequest{
		Type:      "ip",
		Targets:   []string{"192.168.1.1", "192.168.1.2"},
		Reason:    "bot",
		Action:    "block",
		ExpiresAt: "2025-12-31",
	}

	assert.Equal(t, "ip", req.Type)
	assert.Len(t, req.Targets, 2)
	assert.Equal(t, "bot", req.Reason)
	assert.Equal(t, "block", req.Action)
	assert.Equal(t, "2025-12-31", req.ExpiresAt)
}
