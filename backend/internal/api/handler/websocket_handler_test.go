package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(WebSocketHandler))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	err = conn.WriteJSON(WebSocketMessage{
		Type:    "subscribe",
		Payload: []byte(`{"group":"test"}`),
	})
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}
}

func TestRegisterToGroup(t *testing.T) {
	session := &WebSocketSession{
		conn:   nil,
		Groups: make(map[string]bool),
	}

	registerToGroup(session, "test_group")

	if !session.Groups["test_group"] {
		t.Error("Expected session to be registered to test_group")
	}
}

func TestUnregisterFromGroup(t *testing.T) {
	session := &WebSocketSession{
		conn:   nil,
		Groups: map[string]bool{"test_group": true},
	}

	unregisterFromGroup(session, "test_group")

	if session.Groups["test_group"] {
		t.Error("Expected session to be unregistered from test_group")
	}
}

func TestBroadcastToGroup(t *testing.T) {
	msg := service.WebSocketMessage{
		MessageType: "test",
		Data:        []byte(`{"test":"data"}`),
	}

	BroadcastToGroup("test_group", msg)

	time.Sleep(100 * time.Millisecond)
}

func TestSendWithRetry(t *testing.T) {
	session := &WebSocketSession{
		conn: nil,
	}

	msg := &service.WebSocketMessage{
		MessageType: "test",
		Data:        []byte(`{"test":"data"}`),
	}

	sendWithRetry(session, msg)
}