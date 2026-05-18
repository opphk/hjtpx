package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetAlertIcon(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		expected string
	}{
		{"critical", "critical", "exclamation-circle"},
		{"warning", "warning", "exclamation-triangle"},
		{"info", "info", "info-circle"},
		{"unknown", "unknown", "bell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAlertIcon(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConnectedClientsCount(t *testing.T) {
	count := GetConnectedClientsCount()
	assert.GreaterOrEqual(t, count, 0)
}

func TestBroadcastCustomMessage(t *testing.T) {
	err := BroadcastCustomMessage("test", map[string]string{"key": "value"})
	assert.NoError(t, err)
}

func TestTriggerAlert(t *testing.T) {
	tests := []struct {
		name     string
		alertType string
		severity string
		message  string
	}{
		{"critical alert", "system", "critical", "Test critical alert"},
		{"warning alert", "system", "warning", "Test warning alert"},
		{"info alert", "system", "info", "Test info alert"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TriggerAlert(tt.alertType, tt.severity, tt.message)
			assert.NoError(t, err)
		})
	}
}

func TestGetRealtimeMonitoringData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/monitoring/data", GetRealtimeMonitoringData)

	req, _ := http.NewRequest("GET", "/monitoring/data", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "data")
}

func TestGetRealtimeSystemStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/monitoring/status", GetRealtimeSystemStatus)

	req, _ := http.NewRequest("GET", "/monitoring/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "data")
}

func TestGetRealtimeAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/monitoring/alerts", GetRealtimeAlerts)

	req, _ := http.NewRequest("GET", "/monitoring/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "data")
}

func TestMessage_Structure(t *testing.T) {
	msg := Message{
		Type:      "test",
		Payload:   map[string]string{"key": "value"},
		Timestamp: 1234567890,
		ID:        "test-id",
	}

	assert.Equal(t, "test", msg.Type)
	assert.Equal(t, int64(1234567890), msg.Timestamp)
	assert.Equal(t, "test-id", msg.ID)
	assert.NotNil(t, msg.Payload)
}

func TestRealtimeDataPayload_Structure(t *testing.T) {
	payload := RealtimeDataPayload{
		Type:      "metrics",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: 1234567890,
	}

	assert.Equal(t, "metrics", payload.Type)
	assert.Equal(t, int64(1234567890), payload.Timestamp)
	assert.NotNil(t, payload.Data)
}

func TestAlertPayload_Structure(t *testing.T) {
	payload := AlertPayload{
		ID:        1,
		Type:      "system",
		Severity:  "warning",
		Message:   "Test message",
		Timestamp: 1234567890,
		Icon:      "info-circle",
	}

	assert.Equal(t, 1, payload.ID)
	assert.Equal(t, "system", payload.Type)
	assert.Equal(t, "warning", payload.Severity)
	assert.Equal(t, "Test message", payload.Message)
	assert.Equal(t, "info-circle", payload.Icon)
}

func TestHeartbeatPayload_Structure(t *testing.T) {
	payload := HeartbeatPayload{
		Timestamp: 1234567890,
		Latency:   100,
	}

	assert.Equal(t, int64(1234567890), payload.Timestamp)
	assert.Equal(t, int64(100), payload.Latency)
}

func TestSubscriptionPayload_Structure(t *testing.T) {
	payload := SubscriptionPayload{
		Groups: []string{"group1", "group2"},
	}

	assert.Len(t, payload.Groups, 2)
	assert.Contains(t, payload.Groups, "group1")
	assert.Contains(t, payload.Groups, "group2")
}

func TestRealtimeClient_Structure(t *testing.T) {
	client := &RealtimeClient{
		ID:     "test-client-id",
		groups: map[string]bool{"group1": true, "group2": false},
	}

	assert.Equal(t, "test-client-id", client.ID)
	assert.Len(t, client.groups, 2)
}

func TestClientManager_InitialState(t *testing.T) {
	m := &ClientManager{
		clients:    make(map[*RealtimeClient]bool),
		groups:     make(map[string]map[*RealtimeClient]bool),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *RealtimeClient),
		unregister: make(chan *RealtimeClient),
	}

	assert.NotNil(t, m.clients)
	assert.NotNil(t, m.groups)
	assert.NotNil(t, m.broadcast)
	assert.NotNil(t, m.register)
	assert.NotNil(t, m.unregister)
}

func TestMonitoringService_StartStop(t *testing.T) {
	ctx := make(chan struct{})
	
	StartMonitoringService(ctx)
	assert.True(t, IsMonitoringServiceRunning())

	StopMonitoringService()
}

func TestIsMonitoringServiceRunning(t *testing.T) {
	ctx := make(chan struct{})
	
	StartMonitoringService(ctx)
	assert.True(t, IsMonitoringServiceRunning())

	StopMonitoringService()
}

func TestMonitoringService_Structure(t *testing.T) {
	service := &MonitoringService{}
	assert.Nil(t, service.ctx)
	assert.Nil(t, service.cancel)
	assert.False(t, service.isRunning.Load())
}
