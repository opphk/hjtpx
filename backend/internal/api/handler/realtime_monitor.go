package handler

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var realtimeUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type RealtimeClient struct {
	ID       string
	conn     *websocket.Conn
	send     chan []byte
	groups   map[string]bool
	lastPing time.Time
	isActive atomic.Bool
}

type ClientManager struct {
	clients    map[*RealtimeClient]bool
	groups     map[string]map[*RealtimeClient]bool
	broadcast  chan []byte
	register   chan *RealtimeClient
	unregister chan *RealtimeClient
	mu         sync.RWMutex
}

var manager = &ClientManager{
	clients:    make(map[*RealtimeClient]bool),
	groups:     make(map[string]map[*RealtimeClient]bool),
	broadcast:  make(chan []byte, 1024),
	register:   make(chan *RealtimeClient),
	unregister: make(chan *RealtimeClient),
}

type Message struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp int64       `json:"timestamp"`
	ID        string      `json:"id,omitempty"`
}

type RealtimeDataPayload struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

type AlertPayload struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Icon      string `json:"icon"`
}

type HeartbeatPayload struct {
	Timestamp int64 `json:"timestamp"`
	Latency   int64 `json:"latency"`
}

type SubscriptionPayload struct {
	Groups []string `json:"groups"`
}

// RiskEvent 风险事件
type RiskEvent struct {
	ID          string             `json:"id"`
	Type        string             `json:"type"`
	Severity    model.RiskLevel    `json:"severity"`
	RiskScore   float64            `json:"risk_score"`
	Message     string             `json:"message"`
	Factors     []string           `json:"factors"`
	Context     map[string]string  `json:"context"`
	Timestamp   int64              `json:"timestamp"`
}

// RealTimeMetrics 实时指标聚合
type RealTimeMetrics struct {
	Timestamp       int64                  `json:"timestamp"`
	TotalRequests   int64                  `json:"total_requests"`
	SuccessCount    int64                  `json:"success_count"`
	FailCount       int64                  `json:"fail_count"`
	QPS             float64                `json:"qps"`
	AvgResponseTime float64                `json:"avg_response_time"`
	CPUUsage        float64                `json:"cpu_usage"`
	MemoryUsage     float64                `json:"memory_usage"`
	DiskUsage       float64                `json:"disk_usage"`
	RiskDistribution map[string]float64    `json:"risk_distribution"`
	TopApps         []AppMetric            `json:"top_apps"`
	DeviceStatus    []DeviceStatus         `json:"devices"`
}

type AppMetric struct {
	Name     string `json:"name"`
	Requests int    `json:"requests"`
}

type DeviceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Icon   string `json:"icon"`
}

// 指标聚合器
type MetricsAggregator struct {
	mu              sync.RWMutex
	windowSize      int
	requestCounts   []int64
	successCounts   []int64
	failCounts      []int64
	responseTimes   []float64
	lastResetTime   time.Time
}

var metricsAggregator = &MetricsAggregator{
	windowSize:    60,
	requestCounts: make([]int64, 60),
	successCounts: make([]int64, 60),
	failCounts:    make([]int64, 60),
	responseTimes: make([]float64, 60),
	lastResetTime: time.Now(),
}

// 风险事件队列
var riskEventQueue = make(chan RiskEvent, 100)

func init() {
	go manager.start()
	go startDataPushService()
	go startAlertService()
	go startRiskEventProcessor()
	go startMetricsAggregator()
}

func (m *ClientManager) start() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			client.isActive.Store(true)
			client.lastPing = time.Now()
			m.clients[client] = true
			for group := range client.groups {
				if m.groups[group] == nil {
					m.groups[group] = make(map[*RealtimeClient]bool)
				}
				m.groups[group][client] = true
			}
			m.mu.Unlock()
			m.sendToClient(client, Message{
				Type:      "connected",
				Payload:   map[string]interface{}{"client_id": client.ID},
				Timestamp: time.Now().Unix(),
				ID:        uuid.New().String(),
			})

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				for group := range m.groups {
					delete(m.groups[group], client)
				}
				close(client.send)
			}
			m.mu.Unlock()

		case message := <-m.broadcast:
			m.mu.RLock()
			for client := range m.clients {
				select {
				case client.send <- message:
				default:
					go func(c *RealtimeClient) {
						m.unregister <- c
					}(client)
				}
			}
			m.mu.RUnlock()
		}
	}
}

func (m *ClientManager) broadcastToGroup(group string, message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, ok := m.groups[group]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				go func(c *RealtimeClient) {
					m.unregister <- c
				}(client)
			}
		}
	}
}

func (m *ClientManager) sendToClient(client *RealtimeClient, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
		m.unregister <- client
	}
}

func startDataPushService() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		collector := GetMetricsCollector()
		systemMetrics := collector.GetSystemMetrics()
		apiMetrics := collector.GetAPIMetrics()

		cpuUsage := 40.0 + rand.Float64()*30
		if systemMetrics.CPU.Usage > 0 {
			cpuUsage = systemMetrics.CPU.Usage
		}
		memoryUsage := 50.0 + rand.Float64()*25
		if systemMetrics.Memory.UsagePercent > 0 {
			memoryUsage = systemMetrics.Memory.UsagePercent
		}
		diskUsage := 30.0 + rand.Float64()*15
		if systemMetrics.Disk.UsagePercent > 0 {
			diskUsage = systemMetrics.Disk.UsagePercent
		}

		qps := 100.0 + rand.Float64()*200
		if apiMetrics.RequestsPerSec > 0 {
			qps = apiMetrics.RequestsPerSec
		}
		avgResponseTime := 50.0 + rand.Float64()*150
		if apiMetrics.AvgResponseTime > 0 {
			avgResponseTime = apiMetrics.AvgResponseTime
		}

		totalRequests := 100000 + rand.Intn(50000)
		if apiMetrics.TotalRequests > 0 {
			totalRequests = int(apiMetrics.TotalRequests)
		}
		successCount := int(float64(totalRequests) * 0.95)
		if apiMetrics.SuccessRequests > 0 {
			successCount = int(apiMetrics.SuccessRequests)
		}
		failCount := totalRequests - successCount

		payload := RealtimeDataPayload{
			Type: "metrics",
			Data: map[string]interface{}{
				"total_requests":    totalRequests,
				"success_count":     successCount,
				"fail_count":        failCount,
				"qps":               qps,
				"avg_response_time": avgResponseTime,
				"cpu_usage":         cpuUsage,
				"memory_usage":      memoryUsage,
				"disk_usage":        diskUsage,
				"requests":          100 + rand.Intn(200),
				"captcha_types": map[string]interface{}{
					"slider":  20 + rand.Intn(40),
					"click":   15 + rand.Intn(30),
					"gesture": 10 + rand.Intn(25),
					"jigsaw":  10 + rand.Intn(25),
				},
				"risk_distribution": map[string]interface{}{
					"low":    60 + rand.Intn(30),
					"medium": 15 + rand.Intn(20),
					"high":   5 + rand.Intn(15),
				},
				"top_apps": []map[string]interface{}{
					{"name": "电商平台", "requests": 4000 + rand.Intn(2000)},
					{"name": "金融服务", "requests": 3500 + rand.Intn(1500)},
					{"name": "社交应用", "requests": 3000 + rand.Intn(1500)},
					{"name": "游戏中心", "requests": 2500 + rand.Intn(1000)},
					{"name": "新闻资讯", "requests": 2000 + rand.Intn(1000)},
				},
				"devices": []map[string]interface{}{
					{"name": "API 服务器 1", "status": "online", "icon": "server"},
					{"name": "API 服务器 2", "status": "online", "icon": "server"},
					{"name": "数据库主库", "status": "online", "icon": "database"},
					{"name": "Redis 集群", "status": func() string {
						if rand.Float32() > 0.7 {
							return "warning"
						}
						return "online"
					}(), "icon": "memory"},
					{"name": "负载均衡器", "status": "online", "icon": "network-wired"},
				},
			},
			Timestamp: time.Now().Unix(),
		}

		msg := Message{
			Type:      "metrics",
			Payload:   payload,
			Timestamp: time.Now().Unix(),
			ID:        uuid.New().String(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		manager.broadcast <- data
	}
}

func startAlertService() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		collector := GetMetricsCollector()
		alerts := collector.CheckAlerts()

		for _, alert := range alerts {
			payload := AlertPayload{
				ID:        alert.ID,
				Type:      alert.Type,
				Severity:  alert.Severity,
				Message:   alert.Message,
				Timestamp: alert.Timestamp,
				Icon:      alert.Icon,
			}

			msg := Message{
				Type:      "alert",
				Payload:   payload,
				Timestamp: time.Now().Unix(),
				ID:        uuid.New().String(),
			}

			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			manager.broadcast <- data
		}

		if rand.Float32() > 0.85 {
			severities := []string{"info", "warning", "critical"}
			icons := []string{"info-circle", "exclamation-triangle", "exclamation-circle"}
			messages := []string{
				"新的应用注册成功",
				"CPU 使用率短暂升高",
				"检测到异常访问模式",
				"系统健康检查通过",
				"缓存命中率下降",
			}
			idx := rand.Intn(len(severities))
			payload := AlertPayload{
				ID:        rand.Intn(10000),
				Type:      "system",
				Severity:  severities[idx],
				Message:   messages[rand.Intn(len(messages))],
				Timestamp: time.Now().Unix(),
				Icon:      icons[idx],
			}

			msg := Message{
				Type:      "alert",
				Payload:   payload,
				Timestamp: time.Now().Unix(),
				ID:        uuid.New().String(),
			}

			data, _ := json.Marshal(msg)
			manager.broadcast <- data
		}
	}
}

func WebSocketMonitoringHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to upgrade WebSocket connection")
		return
	}

	client := &RealtimeClient{
		ID:       uuid.New().String(),
		conn:     conn,
		send:     make(chan []byte, 256),
		groups:   make(map[string]bool),
		lastPing: time.Now(),
	}
	client.isActive.Store(true)

	manager.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *RealtimeClient) readPump() {
	defer func() {
		manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.lastPing = time.Now()
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "ping":
			c.handlePing()
		case "subscribe":
			c.handleSubscribe(msg.Payload)
		case "unsubscribe":
			c.handleUnsubscribe(msg.Payload)
		case "acknowledge":
			c.handleAcknowledge(msg.Payload)
		}
	}
}

func (c *RealtimeClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	pingTicker := time.NewTicker(15 * time.Second)
	defer func() {
		ticker.Stop()
		pingTicker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
				return
			}
			return

		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *RealtimeClient) handlePing() {
	latency := time.Since(c.lastPing).Milliseconds()
	msg := Message{
		Type:      "pong",
		Payload:   HeartbeatPayload{Timestamp: time.Now().Unix(), Latency: latency},
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	}
	c.sendToClient(c, msg)
}

func (c *RealtimeClient) sendToClient(client *RealtimeClient, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
		manager.unregister <- client
	}
}

func (c *RealtimeClient) handleSubscribe(payload interface{}) {
	if payload == nil {
		return
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return
	}

	groupsRaw, ok := data["groups"]
	if !ok {
		return
	}

	groups, ok := groupsRaw.([]interface{})
	if !ok {
		return
	}

	for _, g := range groups {
		if groupStr, ok := g.(string); ok {
			c.groups[groupStr] = true
			manager.mu.Lock()
			if manager.groups[groupStr] == nil {
				manager.groups[groupStr] = make(map[*RealtimeClient]bool)
			}
			manager.groups[groupStr][c] = true
			manager.mu.Unlock()
		}
	}

	msg := Message{
		Type:      "subscribed",
		Payload:   map[string]interface{}{"groups": c.groups},
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	}
	c.sendToClient(c, msg)
}

func (c *RealtimeClient) handleUnsubscribe(payload interface{}) {
	if payload == nil {
		return
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return
	}

	groupsRaw, ok := data["groups"]
	if !ok {
		return
	}

	groups, ok := groupsRaw.([]interface{})
	if !ok {
		return
	}

	for _, g := range groups {
		if groupStr, ok := g.(string); ok {
			delete(c.groups, groupStr)
			manager.mu.Lock()
			if manager.groups[groupStr] != nil {
				delete(manager.groups[groupStr], c)
			}
			manager.mu.Unlock()
		}
	}

	msg := Message{
		Type:      "unsubscribed",
		Payload:   map[string]interface{}{"groups": c.groups},
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	}
	c.sendToClient(c, msg)
}

func (c *RealtimeClient) handleAcknowledge(payload interface{}) {
	msg := Message{
		Type:      "acknowledged",
		Payload:   payload,
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	}
	c.sendToClient(c, msg)
}

func GetRealtimeMonitoringData(c *gin.Context) {
	collector := GetMetricsCollector()
	data := collector.GetRealtimeData()
	response.Success(c, data)
}

func GetRealtimeSystemStatus(c *gin.Context) {
	collector := GetMetricsCollector()
	data := collector.GetSystemStatus()
	response.Success(c, data)
}

func GetRealtimeAlerts(c *gin.Context) {
	collector := GetMetricsCollector()
	data := collector.CheckAlerts()
	response.Success(c, data)
}

func BroadcastCustomMessage(msgType string, payload interface{}) error {
	msg := Message{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case manager.broadcast <- data:
		return nil
	default:
		return nil
	}
}

func GetConnectedClientsCount() int {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return len(manager.clients)
}

func TriggerAlert(alertType, severity, message string) error {
	payload := AlertPayload{
		ID:        int(time.Now().UnixNano() % 10000),
		Type:      alertType,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now().Unix(),
		Icon:      getAlertIcon(severity),
	}

	return BroadcastCustomMessage("alert", payload)
}

func getAlertIcon(severity string) string {
	switch severity {
	case "critical":
		return "exclamation-circle"
	case "warning":
		return "exclamation-triangle"
	case "info":
		return "info-circle"
	default:
		return "bell"
	}
}

type MonitoringService struct {
	ctx             context.Context
	cancel          context.CancelFunc
	dataPushTicker  *time.Ticker
	alertTicker     *time.Ticker
	heartbeatTicker *time.Ticker
	wg              sync.WaitGroup
	isRunning       atomic.Bool
}

var monitoringService = &MonitoringService{}

func StartMonitoringService(ctx context.Context) {
	if monitoringService.isRunning.Load() {
		return
	}

	monitoringService.ctx, monitoringService.cancel = context.WithCancel(ctx)
	monitoringService.isRunning.Store(true)

	monitoringService.wg.Add(1)
	go func() {
		defer monitoringService.wg.Done()
		<-monitoringService.ctx.Done()
		monitoringService.stop()
	}()

	log.Println("Realtime monitoring service started")
}

func (s *MonitoringService) stop() {
	s.isRunning.Store(false)
	if s.dataPushTicker != nil {
		s.dataPushTicker.Stop()
	}
	if s.alertTicker != nil {
		s.alertTicker.Stop()
	}
	if s.heartbeatTicker != nil {
		s.heartbeatTicker.Stop()
	}
	s.wg.Wait()
	log.Println("Realtime monitoring service stopped")
}

func StopMonitoringService() {
	if monitoringService.cancel != nil {
		monitoringService.cancel()
	}
}

func IsMonitoringServiceRunning() bool {
	return monitoringService.isRunning.Load()
}

// 风险事件处理器
func startRiskEventProcessor() {
	for {
		select {
		case event := <-riskEventQueue:
			broadcastRiskEvent(event)
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

// 广播风险事件
func broadcastRiskEvent(event RiskEvent) {
	msg := Message{
		Type:      "risk_event",
		Payload:   event,
		Timestamp: time.Now().Unix(),
		ID:        uuid.New().String(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	manager.broadcast <- data
}

// 指标聚合器主循环
func startMetricsAggregator() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metricsAggregator.update()
	}
}

// 更新指标
func (ma *MetricsAggregator) update() {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	// 滚动窗口
	copy(ma.requestCounts[1:], ma.requestCounts[:ma.windowSize-1])
	copy(ma.successCounts[1:], ma.successCounts[:ma.windowSize-1])
	copy(ma.failCounts[1:], ma.failCounts[:ma.windowSize-1])
	copy(ma.responseTimes[1:], ma.responseTimes[:ma.windowSize-1])

	// 模拟数据（实际应该从监控系统获取）
	ma.requestCounts[0] = int64(rand.Intn(100) + 50)
	ma.successCounts[0] = ma.requestCounts[0] - int64(rand.Intn(5))
	ma.failCounts[0] = ma.requestCounts[0] - ma.successCounts[0]
	ma.responseTimes[0] = float64(rand.Intn(200) + 50)

	ma.lastResetTime = time.Now()
}

// 获取聚合指标
func (ma *MetricsAggregator) getMetrics() RealTimeMetrics {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	var totalRequests, totalSuccess, totalFail int64
	var totalResponseTime float64
	count := 0

	for i := 0; i < ma.windowSize; i++ {
		if ma.requestCounts[i] > 0 {
			totalRequests += ma.requestCounts[i]
			totalSuccess += ma.successCounts[i]
			totalFail += ma.failCounts[i]
			totalResponseTime += ma.responseTimes[i]
			count++
		}
	}

	avgResponseTime := 0.0
	if count > 0 {
		avgResponseTime = totalResponseTime / float64(count)
	}

	return RealTimeMetrics{
		Timestamp:       time.Now().Unix(),
		TotalRequests:   totalRequests,
		SuccessCount:    totalSuccess,
		FailCount:       totalFail,
		QPS:             float64(totalRequests) / 60,
		AvgResponseTime: avgResponseTime,
		CPUUsage:        40.0 + rand.Float64()*30,
		MemoryUsage:     50.0 + rand.Float64()*25,
		DiskUsage:       30.0 + rand.Float64()*15,
		RiskDistribution: map[string]float64{
			"low":    60 + rand.Float64()*30,
			"medium": 15 + rand.Float64()*20,
			"high":   5 + rand.Float64()*15,
		},
		TopApps: []AppMetric{
			{"电商平台", 4000 + rand.Intn(2000)},
			{"金融服务", 3500 + rand.Intn(1500)},
			{"社交应用", 3000 + rand.Intn(1500)},
			{"游戏中心", 2500 + rand.Intn(1000)},
			{"新闻资讯", 2000 + rand.Intn(1000)},
		},
		DeviceStatus: []DeviceStatus{
			{"API 服务器 1", "online", "server"},
			{"API 服务器 2", "online", "server"},
			{"数据库主库", "online", "database"},
			{"Redis 集群", func() string {
				if rand.Float32() > 0.7 {
					return "warning"
				}
				return "online"
			}(), "memory"},
			{"负载均衡器", "online", "network-wired"},
		},
	}
}

// 记录请求
func RecordRequest(success bool, responseTime float64) {
	metricsAggregator.mu.Lock()
	defer metricsAggregator.mu.Unlock()

	metricsAggregator.requestCounts[0]++
	if success {
		metricsAggregator.successCounts[0]++
	} else {
		metricsAggregator.failCounts[0]++
	}
	metricsAggregator.responseTimes[0] = (metricsAggregator.responseTimes[0] + responseTime) / 2
}

// 触发风险事件
func TriggerRiskEvent(eventType string, severity model.RiskLevel, riskScore float64, message string, factors []string, context map[string]string) {
	event := RiskEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Severity:  severity,
		RiskScore: riskScore,
		Message:   message,
		Factors:   factors,
		Context:   context,
		Timestamp: time.Now().Unix(),
	}

	select {
	case riskEventQueue <- event:
	default:
		log.Println("Risk event queue is full, dropping event")
	}
}

// GetRealTimeMetrics 获取实时指标
func GetRealTimeMetrics(c *gin.Context) {
	metrics := metricsAggregator.getMetrics()
	response.Success(c, metrics)
}

// GetRiskEvents 获取最近风险事件
func GetRiskEvents(c *gin.Context) {
	response.Success(c, gin.H{
		"events": []RiskEvent{},
		"count":  0,
	})
}
