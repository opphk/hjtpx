package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var monitoringUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var (
	clients   = make(map[*Client]bool)
	clientsMu sync.RWMutex
	broadcast = make(chan []byte)
)

func init() {
	go handleMessages()
	go generateMockData()
}

func handleMessages() {
	for {
		msg := <-broadcast
		clientsMu.RLock()
		for client := range clients {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
		clientsMu.RUnlock()
	}
}

func generateMockData() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		totalRequests := 100000 + rand.Intn(100000)
		successCount := 95000 + rand.Intn(95000)
		failCount := totalRequests - successCount
		if failCount < 0 {
			failCount = 0
		}

		metrics := map[string]interface{}{
			"type": "metrics",
			"payload": map[string]interface{}{
				"total_requests":    totalRequests,
				"success_count":     successCount,
				"fail_count":        failCount,
				"qps":               100 + rand.Float64()*200,
				"avg_response_time": 50 + rand.Float64()*200,
				"cpu_usage":         40 + rand.Float64()*30,
				"memory_usage":      55 + rand.Float64()*20,
				"disk_usage":        30 + rand.Float64()*10,
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
		}

		msg, _ := json.Marshal(metrics)
		broadcast <- msg

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
			alert := map[string]interface{}{
				"type": "alert",
				"payload": map[string]interface{}{
					"id":        rand.Intn(1000),
					"severity":  severities[idx],
					"message":   messages[rand.Intn(len(messages))],
					"timestamp": time.Now().Unix(),
					"icon":      icons[idx],
				},
			}
			alertMsg, _ := json.Marshal(alert)
			broadcast <- alertMsg
		}
	}
}

// WebSocketHandler WebSocket处理函数
func WebSocketHandler(c *gin.Context) {
	conn, err := monitoringUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upgrade WebSocket connection")
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	clientsMu.Lock()
	clients[client] = true
	clientsMu.Unlock()

	go client.writePump()
	client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
		clientsMu.Lock()
		delete(clients, c)
		clientsMu.Unlock()
		close(c.send)
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			}
			break
		}
	}
}

func (c *Client) writePump() {
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
		}
	}
}

// GetMonitoringData 获取监控数据
func GetMonitoringData(c *gin.Context) {
	totalRequests := 123456
	successCount := 118500
	failCount := 4956

	response.Success(c, gin.H{
		"timestamp": time.Now().Unix(),
		"requests": gin.H{
			"total":   totalRequests,
			"success": successCount,
			"failed":  failCount,
		},
		"system": gin.H{
			"cpu_usage":    45.2,
			"memory_usage": 62.8,
			"disk_usage":   35.1,
		},
	})
}

// GetAlerts 获取告警列表
func GetAlerts(c *gin.Context) {
	response.Success(c, []gin.H{
		{
			"id":           1,
			"type":         "high_cpu",
			"message":      "Redis 内存使用率较高",
			"severity":     "warning",
			"timestamp":    time.Now().Add(-5 * time.Minute).Unix(),
			"acknowledged": false,
			"icon":         "memory",
		},
		{
			"id":           2,
			"type":         "info",
			"message":      "系统自动备份完成",
			"severity":     "info",
			"timestamp":    time.Now().Add(-10 * time.Minute).Unix(),
			"acknowledged": false,
			"icon":         "check-circle",
		},
	})
}

// GetSystemMetrics 获取系统指标
func GetSystemMetrics(c *gin.Context) {
	response.Success(c, gin.H{
		"cpu":    []float64{42.3, 45.1, 47.8, 44.5, 46.2},
		"memory": []float64{60.2, 62.5, 63.1, 61.8, 62.8},
		"disk":   []float64{34.5, 34.7, 34.9, 35.0, 35.1},
		"network": gin.H{
			"in":  125000,
			"out": 98000,
		},
	})
}

// GetRequestMetrics 获取请求指标
func GetRequestMetrics(c *gin.Context) {
	response.Success(c, gin.H{
		"total_requests":        123456,
		"requests_per_second":   156,
		"average_response_time": 123,
		"error_rate":            2.8,
		"status_codes": gin.H{
			"200": 118000,
			"400": 3000,
			"401": 1000,
			"500": 456,
		},
	})
}

// GetApiStats 获取API统计
func GetApiStats(c *gin.Context) {
	response.Success(c, gin.H{
		"endpoints": []gin.H{
			{
				"path":       "/api/v1/captcha/slider",
				"method":     "GET",
				"requests":   50000,
				"avg_time":   85,
				"error_rate": 1.2,
			},
			{
				"path":       "/api/v1/captcha/click",
				"method":     "GET",
				"requests":   45000,
				"avg_time":   92,
				"error_rate": 1.5,
			},
		},
	})
}
