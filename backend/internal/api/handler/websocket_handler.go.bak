package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	wsService     *service.WebSocketService
	wsServiceOnce sync.Once
	upgrader      = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// GetWebSocketService 获取单例 WebSocket 服务
func GetWebSocketService() *service.WebSocketService {
	wsServiceOnce.Do(func() {
		wsService = service.NewWebSocketService()
	})
	return wsService
}

// WebSocketHandler WebSocket 连接处理函数
func WebSocketVerificationHandler(c *gin.Context) {
	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upgrade WebSocket connection")
		return
	}

	// 获取 WebSocket 服务
	svc := GetWebSocketService()

	// 注册新会话
	session := svc.RegisterSession(conn)

	// 启动读写协程
	go svc.WritePump(session)
	go svc.ReadPump(session, handleVerificationMessage)
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
