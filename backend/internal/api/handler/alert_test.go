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

func TestCreateAlertChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "success - create webhook channel",
			requestBody: map[string]interface{}{
				"name":        "Test Channel",
				"type":        "webhook",
				"config":      map[string]interface{}{"url": "https://example.com/webhook"},
				"description": "Test webhook channel",
				"is_enabled":  true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - create email channel",
			requestBody: map[string]interface{}{
				"name":   "Email Channel",
				"type":   "email",
				"config": map[string]interface{}{"smtp_host": "smtp.example.com"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - missing name",
			requestBody: map[string]interface{}{
				"type":   "webhook",
				"config": map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "error - invalid type",
			requestBody: map[string]interface{}{
				"name":   "Test",
				"type":   "invalid",
				"config": map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/alert/channels", CreateAlertChannel)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/admin/alert/channels", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListAlertChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list channels",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alert/channels", ListAlertChannels)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alert/channels", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetAlertChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		channelID      string
		expectedStatus int
	}{
		{
			name:           "success - get existing channel",
			channelID:      "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid channel id",
			channelID:      "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found - non-existent channel",
			channelID:      "999999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alert/channels/:id", GetAlertChannel)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alert/channels/"+tt.channelID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUpdateAlertChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		channelID      string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:      "success - update channel",
			channelID: "1",
			requestBody: map[string]interface{}{
				"name": "Updated Channel",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid channel id",
			channelID:      "invalid",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/admin/alert/channels/:id", UpdateAlertChannel)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/admin/alert/channels/"+tt.channelID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDeleteAlertChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		channelID      string
		expectedStatus int
	}{
		{
			name:           "success - delete channel",
			channelID:      "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid channel id",
			channelID:      "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.DELETE("/api/v1/admin/alert/channels/:id", DeleteAlertChannel)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/v1/admin/alert/channels/"+tt.channelID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCreateAlertRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "success - create alert rule",
			requestBody: map[string]interface{}{
				"name":       "Test Rule",
				"event_type": "high_traffic",
				"severity":   "warning",
				"channel_ids": []uint{1, 2},
				"is_enabled":  true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - missing name",
			requestBody: map[string]interface{}{
				"event_type": "high_traffic",
				"severity":   "warning",
				"channel_ids": []uint{1},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "error - invalid severity",
			requestBody: map[string]interface{}{
				"name":       "Test",
				"event_type": "high_traffic",
				"severity":   "invalid",
				"channel_ids": []uint{1},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/alert/rules", CreateAlertRule)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/admin/alert/rules", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListAlertRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "success - list rules",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alert/rules", ListAlertRules)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alert/rules", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetAlertRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ruleID         string
		expectedStatus int
	}{
		{
			name:           "success - get existing rule",
			ruleID:        "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid rule id",
			ruleID:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alert/rules/:id", GetAlertRule)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alert/rules/"+tt.ruleID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUpdateAlertRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ruleID         string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:    "success - update rule",
			ruleID: "1",
			requestBody: map[string]interface{}{
				"name": "Updated Rule",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid rule id",
			ruleID:        "invalid",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/admin/alert/rules/:id", UpdateAlertRule)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/admin/alert/rules/"+tt.ruleID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDeleteAlertRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ruleID         string
		expectedStatus int
	}{
		{
			name:           "success - delete rule",
			ruleID:        "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid rule id",
			ruleID:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.DELETE("/api/v1/admin/alert/rules/:id", DeleteAlertRule)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/v1/admin/alert/rules/"+tt.ruleID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestListAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryString    string
		expectedStatus int
	}{
		{
			name:           "success - list all alerts",
			queryString:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - list with status filter",
			queryString:    "?status=active",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - list with severity filter",
			queryString:    "?severity=critical",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - list with pagination",
			queryString:    "?page=1&page_size=20",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alerts", ListAlerts)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alerts"+tt.queryString, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		alertID        string
		expectedStatus int
	}{
		{
			name:           "success - get existing alert",
			alertID:        "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid alert id",
			alertID:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alerts/:id", GetAlert)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alerts/"+tt.alertID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestResolveAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		alertID        string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:    "success - resolve alert with note",
			alertID: "1",
			requestBody: map[string]interface{}{
				"note": "Issue resolved",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "success - resolve alert without note",
			alertID: "1",
			requestBody: map[string]interface{}{
				"note": "",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid alert id",
			alertID:        "invalid",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/api/v1/admin/alerts/:id/resolve", ResolveAlert)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/admin/alerts/"+tt.alertID+"/resolve", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetAlertHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		alertID        string
		expectedStatus int
	}{
		{
			name:           "success - get alert history",
			alertID:        "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid alert id",
			alertID:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/admin/alerts/:id/history", GetAlertHistory)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/admin/alerts/"+tt.alertID+"/history", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestSendTestAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "success - send test alert",
			requestBody: map[string]interface{}{
				"event_type": "test",
				"message":    "This is a test alert",
				"context":    map[string]interface{}{"test": true},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - missing event type",
			requestBody: map[string]interface{}{
				"message": "Test message",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "error - missing message",
			requestBody: map[string]interface{}{
				"event_type": "test",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/admin/alerts/test", SendTestAlert)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/admin/alerts/test", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAlertRequestStructures(t *testing.T) {
	t.Run("CreateAlertChannelRequest marshaling", func(t *testing.T) {
		req := CreateAlertChannelRequest{
			Name:        "Test Channel",
			Type:        "webhook",
			Config:      map[string]interface{}{"url": "https://example.com"},
			Description: "Test description",
			IsEnabled:   true,
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		var unmarshaled CreateAlertChannelRequest
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, req.Name, unmarshaled.Name)
		assert.Equal(t, req.Type, unmarshaled.Type)
	})

	t.Run("UpdateAlertChannelRequest marshaling", func(t *testing.T) {
		name := "Updated Name"
		isEnabled := false

		req := UpdateAlertChannelRequest{
			Name:      &name,
			IsEnabled: &isEnabled,
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("CreateAlertRuleRequest marshaling", func(t *testing.T) {
		req := CreateAlertRuleRequest{
			Name:              "Test Rule",
			EventType:         "high_traffic",
			Condition:         "count > 100",
			Severity:          "warning",
			ChannelIDs:        []uint{1, 2},
			IsEnabled:         true,
			AggregationWindow: 300,
			Threshold:         100,
			Description:       "Test rule",
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("UpdateAlertRuleRequest marshaling", func(t *testing.T) {
		name := "Updated Rule"
		enabled := false
		threshold := 50

		req := UpdateAlertRuleRequest{
			Name:       &name,
			IsEnabled:  &enabled,
			Threshold:  &threshold,
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("ResolveAlertRequest marshaling", func(t *testing.T) {
		req := ResolveAlertRequest{
			Note: "Issue resolved",
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("SendTestAlertRequest marshaling", func(t *testing.T) {
		req := SendTestAlertRequest{
			EventType: "test",
			Message:   "Test message",
			Context:   map[string]interface{}{"key": "value"},
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("ListAlertsQuery marshaling", func(t *testing.T) {
		query := ListAlertsQuery{
			Page:     1,
			PageSize: 20,
			Status:   "active",
			Severity: "critical",
		}

		data, err := json.Marshal(query)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		var unmarshaled ListAlertsQuery
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, query.Page, unmarshaled.Page)
		assert.Equal(t, query.PageSize, unmarshaled.PageSize)
	})
}
