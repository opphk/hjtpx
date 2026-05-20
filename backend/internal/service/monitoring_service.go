package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type MonitoringService struct {
	clients      map[*websocket.Conn]bool
	alerts       []*Alert
	metrics      *SystemMetrics
	wsUpgrader   websocket.Upgrader
	mu           sync.RWMutex
	metricsChan  chan *MetricUpdate
	alertChan    chan *Alert
	broadcastChan chan []byte
	stopChan     chan struct{}
}

type MetricUpdate struct {
	Timestamp    time.Time          `json:"timestamp"`
	CPUUsage     float64            `json:"cpu_usage"`
	MemoryUsage  float64            `json:"memory_usage"`
	DiskUsage    float64            `json:"disk_usage"`
	NetworkIn    int64              `json:"network_in"`
	NetworkOut   int64              `json:"network_out"`
	RequestCount int64              `json:"request_count"`
	ErrorCount   int64              `json:"error_count"`
	ResponseTime float64            `json:"response_time"`
	ActiveConnections int           `json:"active_connections"`
}

type SystemMetrics struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	DiskUsage       float64 `json:"disk_usage"`
	NetworkIn       int64   `json:"network_in"`
	NetworkOut      int64   `json:"network_out"`
	Uptime          int64   `json:"uptime"`
	RequestCount    int64   `json:"request_count"`
	ErrorCount      int64   `json:"error_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime float64 `json:"avg_response_time"`
	ActiveConns     int     `json:"active_connections"`
}

type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Value       float64                `json:"value,omitempty"`
	Threshold   float64                `json:"threshold,omitempty"`
}

type MonitoringConfig struct {
	WebSocketPath    string `json:"web_socket_path"`
	MetricsInterval  int    `json:"metrics_interval"`
	AlertRetention   int    `json:"alert_retention"`
	MaxClients       int    `json:"max_clients"`
	EnableRealtime   bool   `json:"enable_realtime"`
}

func NewMonitoringService() *MonitoringService {
	return &MonitoringService{
		clients:      make(map[*websocket.Conn]bool),
		alerts:       make([]*Alert, 0),
		metrics:      &SystemMetrics{},
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		metricsChan:  make(chan *MetricUpdate, 100),
		alertChan:    make(chan *Alert, 50),
		broadcastChan: make(chan []byte, 200),
		stopChan:     make(chan struct{}),
	}
}

func (s *MonitoringService) Start() {
	go s.metricsCollector()
	go s.alertProcessor()
	go s.broadcaster()
	log.Println("Monitoring service started")
}

func (s *MonitoringService) Stop() {
	close(s.stopChan)
	s.mu.Lock()
	for client := range s.clients {
		client.Close()
	}
	s.mu.Unlock()
	log.Println("Monitoring service stopped")
}

func (s *MonitoringService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	s.mu.Lock()
	if len(s.clients) >= 1000 {
		s.mu.Unlock()
		conn.Close()
		return
	}
	s.clients[conn] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	s.sendInitialData(conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (s *MonitoringService) sendInitialData(conn *websocket.Conn) {
	initialData := map[string]interface{}{
		"type":           "initial",
		"timestamp":       time.Now(),
		"systemMetrics":   s.getCurrentMetrics(),
		"recentAlerts":    s.getRecentAlerts(10),
	}

	data, err := json.Marshal(initialData)
	if err != nil {
		log.Printf("Failed to marshal initial data: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send initial data: %v", err)
	}
}

func (s *MonitoringService) getCurrentMetrics() *SystemMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}

func (s *MonitoringService) getRecentAlerts(limit int) []*Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := limit
	if count > len(s.alerts) {
		count = len(s.alerts)
	}

	result := make([]*Alert, count)
	copy(result, s.alerts[len(s.alerts)-count:])
	return result
}

func (s *MonitoringService) metricsCollector() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			metrics := s.collectMetrics()
			s.mu.Lock()
			s.metrics = metrics
			s.mu.Unlock()

			update := &MetricUpdate{
				Timestamp:    time.Now(),
				CPUUsage:     metrics.CPUUsage,
				MemoryUsage:  metrics.MemoryUsage,
				DiskUsage:    metrics.DiskUsage,
				NetworkIn:    metrics.NetworkIn,
				NetworkOut:   metrics.NetworkOut,
				RequestCount: metrics.RequestCount,
				ErrorCount:   metrics.ErrorCount,
				ResponseTime: metrics.AvgResponseTime,
				ActiveConnections: metrics.ActiveConns,
			}

			select {
			case s.metricsChan <- update:
			default:
			}
		}
	}
}

func (s *MonitoringService) collectMetrics() *SystemMetrics {
	return &SystemMetrics{
		CPUUsage:        45.5,
		MemoryUsage:     62.3,
		DiskUsage:       38.7,
		NetworkIn:       1024 * 1024 * 50,
		NetworkOut:      1024 * 1024 * 30,
		Uptime:          int64(time.Since(time.Now().Add(-24 * time.Hour)).Seconds()),
		RequestCount:    10000,
		ErrorCount:      50,
		SuccessRate:     99.5,
		AvgResponseTime: 125.5,
		ActiveConns:     len(s.clients),
	}
}

func (s *MonitoringService) alertProcessor() {
	for {
		select {
		case <-s.stopChan:
			return
		case alert := <-s.alertChan:
			s.processAlert(alert)
		}
	}
}

func (s *MonitoringService) processAlert(alert *Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert-%d-%s", time.Now().UnixNano(), alert.Name)
	}
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}
	if alert.Status == "" {
		alert.Status = "firing"
	}

	s.alerts = append(s.alerts, alert)

	if len(s.alerts) > 100 {
		s.alerts = s.alerts[len(s.alerts)-100:]
	}

	alertMsg, err := json.Marshal(map[string]interface{}{
		"type":  "alert",
		"alert": alert,
	})
	if err != nil {
		log.Printf("Failed to marshal alert: %v", err)
		return
	}

	select {
	case s.broadcastChan <- alertMsg:
	default:
		log.Println("Broadcast channel full, dropping alert")
	}
}

func (s *MonitoringService) broadcaster() {
	for {
		select {
		case <-s.stopChan:
			return
		case message := <-s.broadcastChan:
			s.broadcastToClients(message)
		case update := <-s.metricsChan:
			metricsMsg, err := json.Marshal(map[string]interface{}{
				"type":    "metrics",
				"metrics": update,
			})
			if err != nil {
				log.Printf("Failed to marshal metrics: %v", err)
				continue
			}
			s.broadcastToClients(metricsMsg)
		}
	}
}

func (s *MonitoringService) broadcastToClients(message []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Failed to send message to client: %v", err)
			client.Close()
			delete(s.clients, client)
		}
	}
}

func (s *MonitoringService) CreateAlert(name, severity, message, source string, value, threshold float64) {
	alert := &Alert{
		Name:        name,
		Severity:    severity,
		Message:     message,
		Source:      source,
		Value:       value,
		Threshold:   threshold,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}

	select {
	case s.alertChan <- alert:
	default:
		log.Println("Alert channel full, dropping alert")
	}
}

func (s *MonitoringService) GetAlerts(status string, limit int) []*Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*Alert
	for _, alert := range s.alerts {
		if status == "" || alert.Status == status {
			filtered = append(filtered, alert)
		}
	}

	if limit > 0 && limit < len(filtered) {
		return filtered[len(filtered)-limit:]
	}
	return filtered
}

func (s *MonitoringService) ResolveAlert(alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, alert := range s.alerts {
		if alert.ID == alertID {
			alert.Status = "resolved"
			return nil
		}
	}
	return fmt.Errorf("alert not found")
}

func (s *MonitoringService) GetMetrics() *SystemMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}

func (s *MonitoringService) RecordRequest(duration time.Duration, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.RequestCount++
	if !success {
		s.metrics.ErrorCount++
	}

	if s.metrics.RequestCount > 0 {
		totalErrors := float64(s.metrics.ErrorCount)
		totalRequests := float64(s.metrics.RequestCount)
		s.metrics.SuccessRate = ((totalRequests - totalErrors) / totalRequests) * 100
	}
}

func (s *MonitoringService) RecordResponseTime(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.metrics.AvgResponseTime == 0 {
		s.metrics.AvgResponseTime = float64(duration.Milliseconds())
	} else {
		currentAvg := s.metrics.AvgResponseTime
		total := float64(s.metrics.RequestCount)
		s.metrics.AvgResponseTime = ((currentAvg * (total - 1)) + float64(duration.Milliseconds())) / total
	}
}

func (s *MonitoringService) GetMonitoringStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_clients":      len(s.clients),
		"total_alerts":        len(s.alerts),
		"active_alerts":       s.countActiveAlerts(),
		"resolved_alerts":     s.countResolvedAlerts(),
		"request_count":       s.metrics.RequestCount,
		"error_count":         s.metrics.ErrorCount,
		"success_rate":        s.metrics.SuccessRate,
		"avg_response_time":   s.metrics.AvgResponseTime,
	}
}

func (s *MonitoringService) countActiveAlerts() int {
	count := 0
	for _, alert := range s.alerts {
		if alert.Status == "firing" {
			count++
		}
	}
	return count
}

func (s *MonitoringService) countResolvedAlerts() int {
	count := 0
	for _, alert := range s.alerts {
		if alert.Status == "resolved" {
			count++
		}
	}
	return count
}

func (s *MonitoringService) ExportMetricsJSON() ([]byte, error) {
	metrics := s.GetMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

func (s *MonitoringService) ExportAlertsJSON() ([]byte, error) {
	alerts := s.GetAlerts("", 100)
	return json.MarshalIndent(alerts, "", "  ")
}

type MetricThreshold struct {
	MetricName  string  `json:"metric_name"`
	Warning     float64 `json:"warning"`
	Critical    float64 `json:"critical"`
	Comparison  string  `json:"comparison"`
}

func (s *MonitoringService) CheckThresholds(thresholds []MetricThreshold) []*Alert {
	s.mu.RLock()
	metrics := s.metrics
	s.mu.RUnlock()

	alerts := make([]*Alert, 0)

	for _, threshold := range thresholds {
		var currentValue float64

		switch threshold.MetricName {
		case "cpu_usage":
			currentValue = metrics.CPUUsage
		case "memory_usage":
			currentValue = metrics.MemoryUsage
		case "disk_usage":
			currentValue = metrics.DiskUsage
		case "error_rate":
			if metrics.RequestCount > 0 {
				currentValue = (float64(metrics.ErrorCount) / float64(metrics.RequestCount)) * 100
			}
		case "response_time":
			currentValue = metrics.AvgResponseTime
		default:
			continue
		}

		shouldAlert := false
		severity := "info"

		switch threshold.Comparison {
		case "gt":
			shouldAlert = currentValue > threshold.Critical
			if shouldAlert {
				severity = "critical"
			} else if currentValue > threshold.Warning {
				severity = "warning"
				shouldAlert = true
			}
		case "lt":
			shouldAlert = currentValue < threshold.Critical
			if shouldAlert {
				severity = "critical"
			}
		}

		if shouldAlert {
			alert := &Alert{
				Name:        fmt.Sprintf("High %s", threshold.MetricName),
				Severity:    severity,
				Message:     fmt.Sprintf("%s is %.2f (threshold: %.2f)", threshold.MetricName, currentValue, threshold.Critical),
				Source:      "monitoring",
				Value:       currentValue,
				Threshold:   threshold.Critical,
				Labels:      map[string]string{"metric": threshold.MetricName},
				Annotations: map[string]string{"description": fmt.Sprintf("Current: %.2f, Warning: %.2f, Critical: %.2f", currentValue, threshold.Warning, threshold.Critical)},
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

func (s *MonitoringService) Subscribe(client *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client] = true
}

func (s *MonitoringService) Unsubscribe(client *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client)
}

func (s *MonitoringService) GetActiveClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

type MetricDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

func (s *MonitoringService) GetMetricHistory(metricName string, duration time.Duration) []MetricDataPoint {
	return []MetricDataPoint{
		{Timestamp: time.Now().Add(-5 * time.Minute), Value: 45.2},
		{Timestamp: time.Now().Add(-4 * time.Minute), Value: 46.8},
		{Timestamp: time.Now().Add(-3 * time.Minute), Value: 44.5},
		{Timestamp: time.Now().Add(-2 * time.Minute), Value: 47.1},
		{Timestamp: time.Now().Add(-1 * time.Minute), Value: 45.5},
	}
}

func (s *MonitoringService) GetMonitoringDashboard() map[string]interface{} {
	s.mu.RLock()
	metrics := s.metrics
	stats := s.GetMonitoringStats()
	s.mu.RUnlock()

	return map[string]interface{}{
		"metrics":         metrics,
		"stats":          stats,
		"recent_alerts":   s.GetAlerts("firing", 5),
		"system_health":   s.getSystemHealth(),
	}
}

func (s *MonitoringService) getSystemHealth() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metrics := s.metrics; metrics != nil {
		if metrics.SuccessRate < 90 {
			return "critical"
		} else if metrics.SuccessRate < 95 {
			return "warning"
		}
	}
	return "healthy"
}

func (s *MonitoringService) HandleMetricsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		metrics := s.GetMetrics()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	case http.MethodPost:
		var update MetricUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.metricsChan <- &update
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *MonitoringService) HandleAlertsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		alerts := s.GetAlerts(status, 100)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alerts)
	case http.MethodPost:
		var alert Alert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.alertChan <- &alert
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *MonitoringService) HandleStatsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.GetMonitoringStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
