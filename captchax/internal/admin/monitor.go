package admin

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"captchax/internal/monitoring"
	"captchax/internal/repository"

	"github.com/gorilla/websocket"
)

type RealtimeHub struct {
	clients     map[*WebSocketClient]bool
	broadcast   chan []byte
	register    chan *WebSocketClient
	unregister  chan *WebSocketClient
	metrics     *monitoring.Metrics
	captchaRepo *repository.CaptchaRepo
	ctx         context.Context
	mu          sync.RWMutex
	running     atomic.Bool
	stopChan    chan struct{}
}

type WebSocketClient struct {
	hub      *RealtimeHub
	conn     *websocket.Conn
	send     chan []byte
	quit     chan struct{}
}

type RealtimeMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Time    int64       `json:"time"`
}

type RealtimeStats struct {
	RequestsPerSecond float64            `json:"requests_per_second"`
	SuccessRate       float64            `json:"success_rate"`
	AvgResponseTime   float64            `json:"avg_response_time_ms"`
	ActiveConnections int                `json:"active_connections"`
	Timestamp         int64              `json:"timestamp"`
}

type ChartDataPoint struct {
	Time        string  `json:"time"`
	Verified    int64   `json:"verified"`
	Rejected    int64   `json:"rejected"`
	Total       int64   `json:"total"`
	SuccessRate float64 `json:"success_rate"`
}

type ResponseTimeDistribution struct {
	Bucket string `json:"bucket"`
	Count  int64  `json:"count"`
}

type ErrorDistribution struct {
	ErrorType string `json:"error_type"`
	Count     int64  `json:"count"`
}

var (
	globalHub   *RealtimeHub
	hubInitOnce sync.Once
)

func GetRealtimeHub() *RealtimeHub {
	hubInitOnce.Do(func() {
		globalHub = &RealtimeHub{
			clients:    make(map[*WebSocketClient]bool),
			broadcast:  make(chan []byte, 256),
			register:   make(chan *WebSocketClient),
			unregister: make(chan *WebSocketClient),
			stopChan:   make(chan struct{}),
		}
	})
	return globalHub
}

func (h *RealtimeHub) Register(client *WebSocketClient) {
	h.register <- client
}

func (h *RealtimeHub) SetCaptchaRepo(repo *repository.CaptchaRepo) {
	h.captchaRepo = repo
}

func (h *RealtimeHub) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *RealtimeHub) SetMetrics(m *monitoring.Metrics) {
	h.metrics = m
}

func (h *RealtimeHub) Run() {
	if h.running.Load() {
		return
	}
	h.running.Store(true)

	go h.broadcastLoop()
	go h.metricsCollector()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mu.Unlock()
			h.broadcastClientCount(clientCount)
		case client := <-h.unregister:
			h.mu.RLock()
			if _, ok := h.clients[client]; ok {
				h.mu.RUnlock()
				h.mu.Lock()
				delete(h.clients, client)
				clientCount := len(h.clients)
				h.mu.Unlock()
				close(client.send)
				h.broadcastClientCount(clientCount)
			} else {
				h.mu.RUnlock()
			}
		case <-h.stopChan:
			h.running.Store(false)
			return
		}
	}
}

func (h *RealtimeHub) Stop() {
	if !h.running.Load() {
		return
	}
	close(h.stopChan)
	h.running.Store(false)
}

func (h *RealtimeHub) broadcastLoop() {
	for {
		select {
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					go func(c *WebSocketClient) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		case <-h.stopChan:
			return
		}
	}
}

func (h *RealtimeHub) metricsCollector() {
	if h.captchaRepo == nil {
		return
	}

	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	prevTotal := int64(0)

	for {
		select {
		case <-ticker.C:
			if h.captchaRepo == nil {
				continue
			}

			now := time.Now()
			windowStart := now.Add(-1 * time.Minute)

			stats, err := h.captchaRepo.GetStats(ctx, windowStart, now)
			if err != nil {
				continue
			}

			rps := float64(stats.TotalCount)
			if prevTotal > 0 {
				rps = float64(stats.TotalCount - prevTotal)
			}
			prevTotal = stats.TotalCount

			successRate := stats.SuccessRate

			chartData := h.collectChartData(ctx, windowStart, now)
			responseTimeDist := h.collectResponseTimeDistribution()
			errorDist := h.collectErrorDistribution()

			msg := RealtimeMessage{
				Type: "stats_update",
				Payload: map[string]interface{}{
					"requests_per_second":        rps,
					"success_rate":               successRate,
					"total_verifications":        stats.TotalCount,
					"verified":                   stats.SuccessCount,
					"rejected":                   stats.FailCount,
					"active_connections":         h.getClientCount(),
					"chart_data":                chartData,
					"response_time_distribution": responseTimeDist,
					"error_distribution":         errorDist,
				},
				Time: time.Now().Unix(),
			}

			data, err := json.Marshal(msg)
			if err == nil {
				select {
				case h.broadcast <- data:
				default:
				}
			}
		case <-h.stopChan:
			return
		}
	}
}

func (h *RealtimeHub) collectChartData(ctx context.Context, start, end time.Time) []ChartDataPoint {
	data := make([]ChartDataPoint, 0, 60)

	hourlyStats, err := h.captchaRepo.GetHourlyTrend(ctx, start, end)
	if err != nil || len(hourlyStats) == 0 {
		for i := 59; i >= 0; i-- {
			t := time.Now().Add(time.Duration(-i) * time.Second)
			data = append(data, ChartDataPoint{
				Time:        t.Format("15:04:05"),
				Verified:    0,
				Rejected:    0,
				Total:       0,
				SuccessRate: 0,
			})
		}
		return data
	}

	for _, point := range hourlyStats {
		successRate := 0.0
		if point.TotalCount > 0 {
			successRate = float64(point.SuccessCount) / float64(point.TotalCount) * 100
		}
		data = append(data, ChartDataPoint{
			Time:        point.Hour.Format("15:04:05"),
			Verified:    point.SuccessCount,
			Rejected:    point.FailCount,
			Total:       point.TotalCount,
			SuccessRate: successRate,
		})
	}

	return data
}

func (h *RealtimeHub) collectResponseTimeDistribution() []ResponseTimeDistribution {
	dist := make([]ResponseTimeDistribution, 0, 11)

	buckets := []string{"<10ms", "10-20ms", "20-30ms", "30-40ms", "40-50ms",
		"50-100ms", "100-200ms", "200-300ms", "300-500ms", "500ms-1s", ">1s"}

	counts := make([]int64, 11)
	if h.metrics != nil {
		counts = h.metrics.GetResponseTimeDistribution()
	}

	for i, bucket := range buckets {
		dist = append(dist, ResponseTimeDistribution{
			Bucket: bucket,
			Count:  counts[i],
		})
	}

	return dist
}

func (h *RealtimeHub) collectErrorDistribution() []ErrorDistribution {
	return []ErrorDistribution{
		{ErrorType: "timeout", Count: 0},
		{ErrorType: "invalid_captcha", Count: 0},
		{ErrorType: "rate_limit", Count: 0},
		{ErrorType: "system_error", Count: 0},
	}
}

func (h *RealtimeHub) broadcastClientCount(count int) {
	msg := RealtimeMessage{
		Type: "client_count",
		Payload: map[string]interface{}{
			"active_connections": count,
		},
		Time: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err == nil {
		select {
		case h.broadcast <- data:
		default:
		}
	}
}

func (h *RealtimeHub) BroadcastStats() {
	if h.captchaRepo == nil {
		return
	}

	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now()
	windowStart := now.Add(-1 * time.Minute)

	stats, err := h.captchaRepo.GetStats(ctx, windowStart, now)
	if err != nil {
		return
	}

	msg := RealtimeMessage{
		Type: "stats_update",
		Payload: map[string]interface{}{
			"total_verifications": stats.TotalCount,
			"verified":            stats.SuccessCount,
			"rejected":            stats.FailCount,
			"success_rate":        stats.SuccessRate,
			"active_connections":  h.getClientCount(),
		},
		Time: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err == nil {
		select {
		case h.broadcast <- data:
		default:
		}
	}
}

func (h *RealtimeHub) getClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
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
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.quit:
			return
		}
	}
}

func (c *WebSocketClient) Close() {
	close(c.quit)
}
