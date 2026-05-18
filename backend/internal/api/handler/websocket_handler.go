package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	wsService     *service.WebSocketService
	wsServiceOnce sync.Once
	upgrader      = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 消息确认超时时间
	msgAckTimeout = 30 * time.Second
	// 消息重试次数
	maxRetries = 3
)

// GetWebSocketService 获取单例 WebSocket 服务
func GetWebSocketService() *service.WebSocketService {
	wsServiceOnce.Do(func() {
		wsService = service.NewWebSocketService()
	})
	return wsService
}

// 待确认消息存储
var pendingMessages = make(map[string]*PendingMessage)
var pendingMu sync.RWMutex

type PendingMessage struct {
	Message    *service.WebSocketMessage
	SessionID  string
	Retries    int
	Timestamp  time.Time
	Timer      *time.Timer
}

// WebSocketHandler WebSocket 连接处理函数
func WebSocketVerificationHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upgrade WebSocket connection")
		return
	}

	svc := GetWebSocketService()
	session := svc.RegisterSession(conn)

	go svc.WritePump(session)
	go svc.ReadPump(session, handleVerificationMessage)
}

// WebSocketAdminHandler 管理员WebSocket连接处理
func WebSocketAdminHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upgrade WebSocket connection")
		return
	}

	svc := GetWebSocketService()
	session := svc.RegisterSession(conn)

	go adminWritePump(session, svc)
	go adminReadPump(session, svc)
}

func adminWritePump(session *service.WebSocketSession, svc *service.WebSocketService) {
	ticker := time.NewTicker(15 * time.Second)
	defer func() {
		ticker.Stop()
		session.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-session.Send:
			session.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				session.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := session.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(session.Send)
			for i := 0; i < n; i++ {
				w.Write(<-session.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			session.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := session.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func adminReadPump(session *service.WebSocketSession, svc *service.WebSocketService) {
	defer func() {
		svc.UnregisterSession(session)
		session.Conn.Close()
	}()

	session.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	session.Conn.SetPongHandler(func(string) error {
		session.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := session.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			}
			break
		}

		var msg service.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		handleAdminMessage(session, msg, svc)
	}
}

func handleAdminMessage(session *service.WebSocketSession, msg service.WebSocketMessage, svc *service.WebSocketService) {
	switch msg.Type {
	case service.MessageTypePing:
		handlePing(session, svc)
	case "subscribe":
		handleSubscribe(session, msg)
	case "unsubscribe":
		handleUnsubscribe(session, msg)
	case "ack":
		handleAck(session, msg)
	default:
		sendErrorResponse(session, "UNSUPPORTED_TYPE", "Unsupported message type", svc)
	}
}

// 订阅分组
func handleSubscribe(session *service.WebSocketSession, msg service.WebSocketMessage) {
	var payload struct {
		Groups []string `json:"groups"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	session.Mu.Lock()
	if session.ClientID == "" {
		session.ClientID = uuid.New().String()
	}
	session.Mu.Unlock()

	for _, group := range payload.Groups {
		registerToGroup(group, session)
	}

	ackMsg := service.WebSocketMessage{
		Type:      "subscribed",
		SessionID: session.ID,
		Timestamp: time.Now().Unix(),
	}
	payloadBytes, _ := json.Marshal(map[string]interface{}{"groups": payload.Groups})
	ackMsg.Payload = payloadBytes
	svc := GetWebSocketService()
	svc.SendMessage(session.ID, ackMsg)
}

// 取消订阅
func handleUnsubscribe(session *service.WebSocketSession, msg service.WebSocketMessage) {
	var payload struct {
		Groups []string `json:"groups"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	for _, group := range payload.Groups {
		unregisterFromGroup(group, session)
	}

	ackMsg := service.WebSocketMessage{
		Type:      "unsubscribed",
		SessionID: session.ID,
		Timestamp: time.Now().Unix(),
	}
	payloadBytes, _ := json.Marshal(map[string]interface{}{"groups": payload.Groups})
	ackMsg.Payload = payloadBytes
	svc := GetWebSocketService()
	svc.SendMessage(session.ID, ackMsg)
}

// 处理消息确认
func handleAck(session *service.WebSocketSession, msg service.WebSocketMessage) {
	var payload struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	pendingMu.Lock()
	if pendingMsg, ok := pendingMessages[payload.MessageID]; ok {
		if pendingMsg.Timer != nil {
			pendingMsg.Timer.Stop()
		}
		delete(pendingMessages, payload.MessageID)
	}
	pendingMu.Unlock()
}

// 分组管理
var groups = make(map[string][]*service.WebSocketSession)
var groupsMu sync.RWMutex

func registerToGroup(group string, session *service.WebSocketSession) {
	groupsMu.Lock()
	if groups[group] == nil {
		groups[group] = []*service.WebSocketSession{}
	}
	groups[group] = append(groups[group], session)
	groupsMu.Unlock()
}

func unregisterFromGroup(group string, session *service.WebSocketSession) {
	groupsMu.Lock()
	if sessions, ok := groups[group]; ok {
		for i, s := range sessions {
			if s.ID == session.ID {
				groups[group] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		}
	}
	groupsMu.Unlock()
}

// BroadcastToGroup 向指定分组广播消息
func BroadcastToGroup(group string, msg service.WebSocketMessage) {
	groupsMu.RLock()
	sessions, ok := groups[group]
	groupsMu.RUnlock()

	if !ok {
		return
	}

	svc := GetWebSocketService()
	for _, session := range sessions {
		go sendWithRetry(session.ID, msg, svc)
	}
}

// 带重试的消息发送
func sendWithRetry(sessionID string, msg service.WebSocketMessage, svc *service.WebSocketService) {
	msgID := uuid.New().String()
	
	pendingMu.Lock()
	pendingMsg := &PendingMessage{
		Message:   &msg,
		SessionID: sessionID,
		Retries:   0,
		Timestamp: time.Now(),
	}
	
	pendingMsg.Timer = time.AfterFunc(msgAckTimeout, func() {
		pendingMu.Lock()
		if pendingMsg.Retries < maxRetries {
			pendingMsg.Retries++
			pendingMu.Unlock()
			sendWithRetry(sessionID, msg, svc)
		} else {
			delete(pendingMessages, msgID)
			pendingMu.Unlock()
		}
	})
	
	pendingMessages[msgID] = pendingMsg
	pendingMu.Unlock()

	svc.SendMessage(sessionID, msg)
}

// handleVerificationMessage 处理验证消息
func handleVerificationMessage(session *service.WebSocketSession, msg service.WebSocketMessage) {
	svc := GetWebSocketService()

	switch msg.Type {
	case service.MessageTypeHello:
		handleHello(session, msg, svc)
	case service.MessageTypeAnswer:
		handleAnswer(session, msg, svc)
	case service.MessageTypePing:
		handlePing(session, svc)
	case service.MessageTypeClose:
		svc.UnregisterSession(session)
	default:
		sendErrorResponse(session, "UNSUPPORTED_TYPE", "Unsupported message type", svc)
	}
}

// handleHello 处理握手消息
func handleHello(session *service.WebSocketSession, msg service.WebSocketMessage, svc *service.WebSocketService) {
	var helloPayload service.HelloPayload
	if err := json.Unmarshal(msg.Payload, &helloPayload); err != nil {
		sendErrorResponse(session, "INVALID_PAYLOAD", "Invalid hello payload", svc)
		return
	}

	// 更新会话信息
	session.ClientID = helloPayload.ClientID

	// 发送第一个验证挑战
	sendNewChallenge(session, svc)
}

// handleAnswer 处理验证答案
func handleAnswer(session *service.WebSocketSession, msg service.WebSocketMessage, svc *service.WebSocketService) {
	var answerPayload service.AnswerPayload
	if err := json.Unmarshal(msg.Payload, &answerPayload); err != nil {
		sendErrorResponse(session, "INVALID_PAYLOAD", "Invalid answer payload", svc)
		return
	}

	// 模拟验证逻辑
	success := verifyAnswer(answerPayload)
	riskScore := calculateRiskScore(answerPayload)
	var message string

	if success {
		message = "Verification successful"
	} else {
		message = "Verification failed"
	}

	// 生成验证令牌
	token := ""
	if success {
		token = service.GenerateVerificationToken(session.ID)
	}

	// 发送验证结果
	err := svc.SendResult(
		session.ID,
		answerPayload.ChallengeID,
		success,
		message,
		token,
		riskScore,
	)
	if err != nil {
		return
	}

	// 如果验证成功，结束会话；否则发送新挑战
	if !success {
		time.Sleep(500 * time.Millisecond)
		sendNewChallenge(session, svc)
	}
}

// handlePing 处理 ping 消息
func handlePing(session *service.WebSocketSession, svc *service.WebSocketService) {
	pongMsg := service.WebSocketMessage{
		Type:      service.MessageTypePong,
		Timestamp: time.Now().Unix(),
	}
	svc.SendMessage(session.ID, pongMsg)
}

// sendNewChallenge 发送新的验证挑战
func sendNewChallenge(session *service.WebSocketSession, svc *service.WebSocketService) {
	challengeTypes := []string{"slider", "click", "rotation", "gesture"}
	challengeType := challengeTypes[rand.Intn(len(challengeTypes))]

	challengeData := generateChallengeData(challengeType)

	_, err := svc.SendChallenge(session.ID, challengeType, challengeData)
	if err != nil {
		sendErrorResponse(session, "CHALLENGE_FAILED", "Failed to send challenge", svc)
	}
}

// generateChallengeData 生成挑战数据
func generateChallengeData(challengeType string) map[string]interface{} {
	data := make(map[string]interface{})

	switch challengeType {
	case "slider":
		data["target_x"] = 100 + rand.Intn(200)
		data["puzzle_y"] = 50 + rand.Intn(100)
		data["image_url"] = "/api/v1/captcha/slider/image"
		data["tolerance"] = 5

	case "click":
		points := make([]map[string]int, 0)
		numPoints := 3 + rand.Intn(2)
		for i := 0; i < numPoints; i++ {
			points = append(points, map[string]int{
				"x": 50 + rand.Intn(250),
				"y": 50 + rand.Intn(150),
			})
		}
		data["points"] = points
		data["hint"] = "Click on the marked points in order"
		data["image_url"] = "/api/v1/captcha/click/image"

	case "rotation":
		data["target_angle"] = rand.Intn(360)
		data["image_url"] = "/api/v1/captcha/rotation/image"
		data["tolerance"] = 10

	case "gesture":
		gestures := []string{"checkmark", "circle", "triangle", "line-v", "line-h"}
		data["gesture"] = gestures[rand.Intn(len(gestures))]
		data["hint"] = "Draw the gesture shown"
	}

	data["expires_in"] = 120
	return data
}

// verifyAnswer 模拟验证答案
func verifyAnswer(answer service.AnswerPayload) bool {
	// 在真实场景中，这里会进行实际的验证逻辑
	// 这里我们使用简单的模拟，成功率为 80%
	return rand.Float32() < 0.8
}

// calculateRiskScore 计算风险分数
func calculateRiskScore(answer service.AnswerPayload) int {
	// 模拟风险评估，范围 0-100，分数越低越安全
	baseScore := rand.Intn(50)

	if answer.Data != nil {
		if speed, ok := answer.Data["speed"].(float64); ok && speed > 1000 {
			baseScore += 30
		}
	}

	if baseScore > 100 {
		baseScore = 100
	}
	return baseScore
}

// sendErrorResponse 发送错误响应
func sendErrorResponse(session *service.WebSocketSession, code string, message string, svc *service.WebSocketService) {
	errorPayload := service.ErrorPayload{
		Code:    code,
		Message: message,
	}
	payloadBytes, _ := json.Marshal(errorPayload)

	msg := service.WebSocketMessage{
		Type:      service.MessageTypeError,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	svc.SendMessage(session.ID, msg)
}

// GetWebSocketStats 获取 WebSocket 统计信息
func GetWebSocketStats(c *gin.Context) {
	svc := GetWebSocketService()
	sessions := svc.GetActiveSessions()

	response.Success(c, gin.H{
		"active_sessions": len(sessions),
		"session_ids":     sessions,
		"timestamp":       time.Now().Unix(),
	})
}

// BroadcastWebSocketMessage 广播消息
func BroadcastWebSocketMessage(c *gin.Context) {
	var req struct {
		Type    string      `json:"type" binding:"required"`
		Payload interface{} `json:"payload" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	svc := GetWebSocketService()
	payloadBytes, _ := json.Marshal(req.Payload)

	msg := service.WebSocketMessage{
		Type:      req.Type,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	svc.BroadcastMessage(msg)

	response.Success(c, gin.H{
		"message": "Broadcast message sent",
		"type":    req.Type,
	})
}
