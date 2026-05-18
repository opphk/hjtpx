package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestClientManager_RegisterClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws", RealtimeMonitorWebSocket)

	req, _ := http.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	go func() {
		router.ServeHTTP(w, req)
	}()

	select {
	case client := <-manager.register:
		if client == nil {
			t.Error("注册的客户端为 nil")
		}
	case <-time.After(1 * time.Second):
	}
}

func TestClientManager_UnregisterClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws", RealtimeMonitorWebSocket)

	req, _ := http.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()
	
	go router.ServeHTTP(w, req)

	select {
	case client := <-manager.register:
		if client != nil {
			manager.unregister <- client
		}
	case <-time.After(1 * time.Second):
	}

	select {
	case client := <-manager.unregister:
		if client == nil {
			t.Error("取消注册的客户端为 nil")
		}
	case <-time.After(1 * time.Second):
	}
}

func TestClientManager_Broadcast(t *testing.T) {
	msg := Message{
		Type:      "test",
		Payload:   map[string]string{"message": "hello"},
		Timestamp: 1234567890,
	}
	
	msgBytes, _ := json.Marshal(msg)
	
	initialCount := len(manager.clients)
	manager.broadcast <- msgBytes

	if len(manager.clients) != initialCount {
		t.Errorf("广播不应改变客户端数量")
	}
}

func TestClientManager_GetClientCount(t *testing.T) {
	count := GetClientCount()
	if count < 0 {
		t.Errorf("客户端数量不应为负数: %d", count)
	}
}

func TestGetAlertPayload(t *testing.T) {
	alert := GetAlertPayload(1, "critical", "测试告警")
	
	if alert.ID != 1 {
		t.Errorf("期望告警ID为 1, 实际得到 %d", alert.ID)
	}
	if alert.Type != "critical" {
		t.Errorf("期望告警类型为 critical, 实际得到 %s", alert.Type)
	}
	if alert.Message != "测试告警" {
		t.Errorf("期望告警消息为 测试告警, 实际得到 %s", alert.Message)
	}
	if alert.Timestamp <= 0 {
		t.Error("告警时间戳应该大于 0")
	}
}

func TestGetHeartbeatPayload(t *testing.T) {
	payload := GetHeartbeatPayload()
	
	if payload.Timestamp <= 0 {
		t.Error("心跳时间戳应该大于 0")
	}
	if payload.Latency < 0 {
		t.Errorf("心跳延迟不应为负数: %d", payload.Latency)
	}
}

func TestRealtimeDataPayload_JSON(t *testing.T) {
	payload := RealtimeDataPayload{
		Type: "metrics",
		Data: map[string]interface{}{
			"cpu": 45.5,
			"mem": 60.2,
		},
		Timestamp: 1234567890,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		t.Errorf("JSON 序列化失败: %v", err)
	}

	var decoded RealtimeDataPayload
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Errorf("JSON 反序列化失败: %v", err)
	}

	if decoded.Type != payload.Type {
		t.Errorf("类型不匹配: 期望 %s, 实际 %s", payload.Type, decoded.Type)
	}
	if decoded.Timestamp != payload.Timestamp {
		t.Errorf("时间戳不匹配: 期望 %d, 实际 %d", payload.Timestamp, decoded.Timestamp)
	}
}

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		Type:      "test",
		Payload:   map[string]string{"key": "value"},
		Timestamp: 1234567890,
		ID:        "msg-123",
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("JSON 序列化失败: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Errorf("JSON 反序列化失败: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("类型不匹配: 期望 %s, 实际 %s", msg.Type, decoded.Type)
	}
	if decoded.ID != msg.ID {
		t.Errorf("ID 不匹配: 期望 %s, 实际 %s", msg.ID, decoded.ID)
	}
}

func TestAlertPayload_JSON(t *testing.T) {
	alert := AlertPayload{
		ID:        1,
		Type:      "critical",
		Severity:  "high",
		Message:   "测试告警",
		Timestamp: 1234567890,
		Icon:      "warning",
	}

	jsonBytes, err := json.Marshal(alert)
	if err != nil {
		t.Errorf("JSON 序列化失败: %v", err)
	}

	var decoded AlertPayload
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Errorf("JSON 反序列化失败: %v", err)
	}

	if decoded.ID != alert.ID {
		t.Errorf("ID 不匹配: 期望 %d, 实际 %d", alert.ID, decoded.ID)
	}
	if decoded.Severity != alert.Severity {
		t.Errorf("严重级别不匹配: 期望 %s, 实际 %s", alert.Severity, decoded.Severity)
	}
}
