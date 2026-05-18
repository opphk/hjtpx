package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// 消息类型
	MessageTypeHello     = "hello"
	MessageTypeChallenge = "challenge"
	MessageTypeAnswer    = "answer"
	MessageTypeResult    = "result"
	MessageTypePing      = "ping"
	MessageTypePong      = "pong"
	MessageTypeError     = "error"
	MessageTypeClose     = "close"

	// 验证状态
	VerificationStatusPending = "pending"
	VerificationStatusSuccess = "success"
	VerificationStatusFailed  = "failed"
	VerificationStatusExpired = "expired"

	// 超时设置
	ConnectionTimeout = 5 * time.Minute
	PingInterval      = 30 * time.Second
	ReadDeadline      = 60 * time.Second
	WriteDeadline     = 10 * time.Second
)

var (
	ErrConnectionClosed   = errors.New("connection closed")
	ErrInvalidMessageType = errors.New("invalid message type")
)

// WebSocketMessage 定义通用消息结构
type WebSocketMessage struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// HelloPayload 握手消息
type HelloPayload struct {
	ClientID  string `json:"client_id"`
	UserAgent string `json:"user_agent,omitempty"`
}

// ChallengePayload 验证挑战
type ChallengePayload struct {
	ChallengeID string                 `json:"challenge_id"`
	Type        string                 `json:"type"` // slider, click, gesture 等
	Data        map[string]interface{} `json:"data"`
	ExpiresAt   int64                  `json:"expires_at"`
}

// AnswerPayload 验证答案
type AnswerPayload struct {
	ChallengeID string                 `json:"challenge_id"`
	Data        map[string]interface{} `json:"data"`
}

// ResultPayload 验证结果
type ResultPayload struct {
	ChallengeID string `json:"challenge_id"`
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
	Token       string `json:"token,omitempty"`
	RiskScore   int    `json:"risk_score,omitempty"`
}

// ErrorPayload 错误消息
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WebSocketSession 代表一个WebSocket会话
type WebSocketSession struct {
	ID         string
	Conn       *websocket.Conn
	Send       chan []byte
	CreatedAt  time.Time
	LastActive time.Time
	ClientID   string
	Status     string
	Mu         sync.RWMutex
}

// WebSocketService 管理所有WebSocket连接
type WebSocketService struct {
	sessions   map[string]*WebSocketSession
	clients    map[*WebSocketSession]bool
	mu         sync.RWMutex
	register   chan *WebSocketSession
	unregister chan *WebSocketSession
	broadcast  chan []byte
}

var wsServiceInstance *WebSocketService
var wsServiceOnce sync.Once

// NewWebSocketService 创建WebSocket服务
func NewWebSocketService() *WebSocketService {
	s := &WebSocketService{
		sessions:   make(map[string]*WebSocketSession),
		clients:    make(map[*WebSocketSession]bool),
		register:   make(chan *WebSocketSession),
		unregister: make(chan *WebSocketSession),
		broadcast:  make(chan []byte),
	}
	go s.run()
	return s
}

// GetWebSocketService 获取单例 WebSocket 服务
func GetWebSocketService() *WebSocketService {
	wsServiceOnce.Do(func() {
		wsServiceInstance = NewWebSocketService()
	})
	return wsServiceInstance
}

// run 处理服务生命周期
func (s *WebSocketService) run() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.sessions[client.ID] = client
			s.mu.Unlock()

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				delete(s.sessions, client.ID)
				close(client.Send)
			}
			s.mu.Unlock()

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(s.clients, client)
					delete(s.sessions, client.ID)
				}
			}
			s.mu.RUnlock()

		case <-ticker.C:
			s.cleanupExpiredSessions()
		}
	}
}

// RegisterSession 注册新会话
func (s *WebSocketService) RegisterSession(conn *websocket.Conn) *WebSocketSession {
	sessionID := uuid.New().String()
	session := &WebSocketSession{
		ID:         sessionID,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Status:     VerificationStatusPending,
	}

	s.register <- session
	return session
}

// UnregisterSession 注销会话
func (s *WebSocketService) UnregisterSession(session *WebSocketSession) {
	s.unregister <- session
}

// GetSession 获取会话
func (s *WebSocketService) GetSession(sessionID string) (*WebSocketSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	return session, ok
}

// SendMessage 发送消息到指定会话
func (s *WebSocketService) SendMessage(sessionID string, msg WebSocketMessage) error {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	msg.Timestamp = time.Now().Unix()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case session.Send <- data:
		return nil
	default:
		s.UnregisterSession(session)
		return ErrConnectionClosed
	}
}

// BroadcastMessage 广播消息
func (s *WebSocketService) BroadcastMessage(msg WebSocketMessage) {
	msg.Timestamp = time.Now().Unix()
	data, _ := json.Marshal(msg)
	s.broadcast <- data
}

// ReadPump 读取消息循环
func (s *WebSocketService) ReadPump(session *WebSocketSession, handler func(*WebSocketSession, WebSocketMessage)) {
	defer func() {
		s.UnregisterSession(session)
		session.Conn.Close()
	}()

	session.Conn.SetReadDeadline(time.Now().Add(ReadDeadline))
	session.Conn.SetPongHandler(func(string) error {
		session.Conn.SetReadDeadline(time.Now().Add(ReadDeadline))
		return nil
	})

	for {
		_, message, err := session.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// 记录错误但不中断
			}
			break
		}

		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			s.sendError(session, "INVALID_JSON", "Invalid JSON format")
			continue
		}

		wsMsg.SessionID = session.ID
		session.Mu.Lock()
		session.LastActive = time.Now()
		session.Mu.Unlock()

		if handler != nil {
			go handler(session, wsMsg)
		} else {
			s.handleDefaultMessage(session, wsMsg)
		}
	}
}

// WritePump 写入消息循环
func (s *WebSocketService) WritePump(session *WebSocketSession) {
	ticker := time.NewTicker(PingInterval)
	defer func() {
		ticker.Stop()
		session.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-session.Send:
			session.Conn.SetWriteDeadline(time.Now().Add(WriteDeadline))
			if !ok {
				session.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := session.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
			n := len(session.Send)
			for i := 0; i < n; i++ {
				w.Write(<-session.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			session.Conn.SetWriteDeadline(time.Now().Add(WriteDeadline))
			if err := session.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleDefaultMessage 默认消息处理
func (s *WebSocketService) handleDefaultMessage(session *WebSocketSession, msg WebSocketMessage) {
	switch msg.Type {
	case MessageTypeHello:
		s.handleHello(session, msg)
	case MessageTypePing:
		s.sendPong(session)
	case MessageTypeClose:
		s.UnregisterSession(session)
	default:
		s.sendError(session, "UNKNOWN_TYPE", "Unknown message type")
	}
}

// handleHello 处理握手消息
func (s *WebSocketService) handleHello(session *WebSocketSession, msg WebSocketMessage) {
	var payload HelloPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		s.sendError(session, "INVALID_PAYLOAD", "Invalid hello payload")
		return
	}

	session.Mu.Lock()
	session.ClientID = payload.ClientID
	session.Mu.Unlock()

	// 发送响应
	response := WebSocketMessage{
		Type:      "hello_ack",
		SessionID: session.ID,
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(map[string]interface{}{
		"session_id":  session.ID,
		"server_time": time.Now().Unix(),
	})
	response.Payload = data
	s.SendMessage(session.ID, response)
}

// sendPong 发送pong
func (s *WebSocketService) sendPong(session *WebSocketSession) {
	msg := WebSocketMessage{
		Type:      MessageTypePong,
		Timestamp: time.Now().Unix(),
	}
	s.SendMessage(session.ID, msg)
}

// sendError 发送错误
func (s *WebSocketService) sendError(session *WebSocketSession, code, message string) {
	errorPayload := ErrorPayload{
		Code:    code,
		Message: message,
	}
	payload, _ := json.Marshal(errorPayload)
	msg := WebSocketMessage{
		Type:      MessageTypeError,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(msg)
	select {
	case session.Send <- data:
	default:
	}
}

// SendChallenge 发送验证挑战
func (s *WebSocketService) SendChallenge(sessionID string, challengeType string, data map[string]interface{}) (string, error) {
	_, ok := s.GetSession(sessionID)
	if !ok {
		return "", ErrSessionNotFound
	}

	challengeID := uuid.New().String()
	payload := ChallengePayload{
		ChallengeID: challengeID,
		Type:        challengeType,
		Data:        data,
		ExpiresAt:   time.Now().Add(2 * time.Minute).Unix(),
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := WebSocketMessage{
		Type:      MessageTypeChallenge,
		SessionID: sessionID,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	return challengeID, s.SendMessage(sessionID, msg)
}

// SendResult 发送验证结果
func (s *WebSocketService) SendResult(sessionID string, challengeID string, success bool, message string, token string, riskScore int) error {
	payload := ResultPayload{
		ChallengeID: challengeID,
		Success:     success,
		Message:     message,
		Token:       token,
		RiskScore:   riskScore,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := WebSocketMessage{
		Type:      MessageTypeResult,
		SessionID: sessionID,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	return s.SendMessage(sessionID, msg)
}

// cleanupExpiredSessions 清理过期会话
func (s *WebSocketService) cleanupExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.LastActive) > ConnectionTimeout {
			session.Conn.Close()
			close(session.Send)
			delete(s.clients, session)
			delete(s.sessions, id)
		}
	}
}

// GetSessionCount 获取当前会话数
func (s *WebSocketService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// GetActiveSessions 获取所有活跃会话
func (s *WebSocketService) GetActiveSessions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionIDs := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	return sessionIDs
}

// GenerateVerificationToken 生成验证令牌
func GenerateVerificationToken(sessionID string) string {
	return fmt.Sprintf("hjtpx_%s_%d", sessionID, time.Now().Unix())
}
