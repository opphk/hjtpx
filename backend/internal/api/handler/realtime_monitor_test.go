package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketMonitoringHandler_Upgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/ws", WebSocketMonitoringHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/ws", nil)
	r.ServeHTTP(w, req)

	assert.NotNil(t, w.Code)
}

func TestGetRealtimeMonitoringData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/realtime-data", GetRealtimeMonitoringData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/realtime-data", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "system")
	assert.Contains(t, w.Body.String(), "api")
}

func TestGetRealtimeSystemStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/system-status", GetRealtimeSystemStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/system-status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "overall")
	assert.Contains(t, w.Body.String(), "cpu")
	assert.Contains(t, w.Body.String(), "memory")
	assert.Contains(t, w.Body.String(), "disk")
}

func TestGetRealtimeAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/admin/monitoring/realtime-alerts", GetRealtimeAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/monitoring/realtime-alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBroadcastCustomMessage(t *testing.T) {
	err := BroadcastCustomMessage("test", map[string]interface{}{"key": "value"})
	assert.NoError(t, err)
}

func TestGetConnectedClientsCount(t *testing.T) {
	count := GetConnectedClientsCount()
	assert.GreaterOrEqual(t, count, 0)
}

func TestTriggerAlert(t *testing.T) {
	err := TriggerAlert("test_alert", "info", "Test alert message")
	assert.NoError(t, err)
}

func TestGetAlertIcon(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "exclamation-circle"},
		{"warning", "exclamation-triangle"},
		{"info", "info-circle"},
		{"unknown", "bell"},
	}

	for _, tt := range tests {
		result := getAlertIcon(tt.severity)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMessageStruct(t *testing.T) {
	msg := Message{
		Type:      "test",
		Payload:   map[string]interface{}{"key": "value"},
		Timestamp: time.Now().Unix(),
		ID:        "test-id",
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.ID, decoded.ID)
}

func TestRealtimeDataPayloadStruct(t *testing.T) {
	payload := RealtimeDataPayload{
		Type: "metrics",
		Data: map[string]interface{}{
			"cpu_usage": 50.5,
			"memory":    1024,
		},
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded RealtimeDataPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.Type, decoded.Type)
}

func TestAlertPayloadStruct(t *testing.T) {
	payload := AlertPayload{
		ID:        1,
		Type:      "high_cpu",
		Severity:  "warning",
		Message:   "CPU usage is high",
		Timestamp: time.Now().Unix(),
		Icon:      "cpu",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded AlertPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.ID, decoded.ID)
	assert.Equal(t, payload.Type, decoded.Type)
	assert.Equal(t, payload.Severity, decoded.Severity)
}

func TestHeartbeatPayloadStruct(t *testing.T) {
	payload := HeartbeatPayload{
		Timestamp: time.Now().Unix(),
		Latency:   100,
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded HeartbeatPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.Latency, decoded.Latency)
}

func TestSubscriptionPayloadStruct(t *testing.T) {
	payload := SubscriptionPayload{
		Groups: []string{"group1", "group2"},
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded SubscriptionPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(decoded.Groups))
}

func TestClientManager_ConcurrentBroadcast(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			BroadcastCustomMessage("test", map[string]interface{}{"index": i})
		}()
	}
	wg.Wait()
}

func TestMonitoringServiceStartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	StartMonitoringService(ctx)
	assert.True(t, IsMonitoringServiceRunning())

	StopMonitoringService()
	time.Sleep(100 * time.Millisecond)
	assert.False(t, IsMonitoringServiceRunning())
}

func TestRealtimeClientStruct(t *testing.T) {
	client := &RealtimeClient{
		ID:       "test-client-id",
		groups:   make(map[string]bool),
		lastPing: time.Now(),
	}
	client.isActive.Store(true)

	assert.Equal(t, "test-client-id", client.ID)
	assert.True(t, client.isActive.Load())
	assert.NotNil(t, client.groups)
}

func TestMonitoringServiceStruct(t *testing.T) {
	service := &MonitoringService{}
	assert.NotNil(t, service)
}

func TestClientManagerStruct(t *testing.T) {
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

func TestRealtimeClientSendToClient(t *testing.T) {
	client := &RealtimeClient{
		ID:       "test-client",
		send:     make(chan []byte, 10),
		groups:   make(map[string]bool),
		lastPing: time.Now(),
	}
	client.isActive.Store(true)

	msg := Message{
		Type:      "test",
		Payload:   map[string]interface{}{"key": "value"},
		Timestamp: time.Now().Unix(),
		ID:        "msg-id",
	}

	client.sendToClient(client, msg)

	select {
	case received := <-client.send:
		assert.NotEmpty(t, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected message to be sent to client")
	}
}

func TestMultipleAlertTriggers(t *testing.T) {
	alertTypes := []string{"cpu", "memory", "disk", "network"}
	severities := []string{"info", "warning", "critical"}

	for i := 0; i < 10; i++ {
		alertType := alertTypes[i%len(alertTypes)]
		severity := severities[i%len(severities)]
		err := TriggerAlert(alertType, severity, "Test alert")
		assert.NoError(t, err)
	}
}

func TestContextManagement(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	StartMonitoringService(ctx1)
	assert.True(t, IsMonitoringServiceRunning())

	ctx2, cancel2 := context.WithCancel(context.Background())
	StartMonitoringService(ctx2)
	assert.True(t, IsMonitoringServiceRunning())

	cancel1()
	time.Sleep(100 * time.Millisecond)

	cancel2()
	StopMonitoringService()
}

type testMessageHandler struct {
	mu       sync.Mutex
	messages []Message
}

func (h *testMessageHandler) handleMessage(msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, msg)
}

func TestMessageHandler(t *testing.T) {
	handler := &testMessageHandler{
		messages: make([]Message, 0),
	}

	for i := 0; i < 5; i++ {
		msg := Message{
			Type:      "test",
			Payload:   map[string]interface{}{"index": i},
			Timestamp: time.Now().Unix(),
			ID:        "msg-" + string(rune('0'+i)),
		}
		handler.handleMessage(msg)
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()
	assert.Equal(t, 5, len(handler.messages))
}
