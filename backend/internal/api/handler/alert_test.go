package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestAlertWebSocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/ws/alerts", AlertWebSocketHandler)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/alerts?severity=high"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	alert := AlertEvent{
		ID:        "test-alert-001",
		Type:      "test",
		Severity:  "high",
		Message:   "Test alert message",
		Timestamp: time.Now().Unix(),
	}

	BroadcastAlert(alert)

	time.Sleep(200 * time.Millisecond)
}

func TestBroadcastAlert(t *testing.T) {
	alert := AlertEvent{
		ID:        "test-alert-002",
		Type:      "security",
		Severity:  "critical",
		Message:   "Test critical alert",
		Timestamp: time.Now().Unix(),
	}

	BroadcastAlert(alert)

	time.Sleep(100 * time.Millisecond)
}

func TestGetAlertStatistics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/alerts/statistics", GetAlertStatistics)

	req, err := http.NewRequest("GET", "/api/alerts/statistics", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

func TestGetAlertsBySeverity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/alerts", GetAlertsBySeverity)

	req, err := http.NewRequest("GET", "/api/alerts?severity=high", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

func TestTriggerRiskAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/alerts/trigger", TriggerRiskAlert)

	body := `{"risk_score": 85.5, "event_type": "anomaly", "description": "Test anomaly detected"}`
	req, err := http.NewRequest("POST", "/api/alerts/trigger", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

func TestDismissAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PUT("/api/alerts/:id/dismiss", DismissAlert)

	req, err := http.NewRequest("PUT", "/api/alerts/test-alert-001/dismiss", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

func TestGetAlertHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/alerts/history", GetAlertHistory)

	req, err := http.NewRequest("GET", "/api/alerts/history?start_time=0&end_time="+string(rune(time.Now().Unix())), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}