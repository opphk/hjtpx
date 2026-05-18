package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWebSocketService(t *testing.T) {
	service := NewWebSocketService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.sessions)
	assert.NotNil(t, service.clients)
	assert.NotNil(t, service.register)
	assert.NotNil(t, service.unregister)
	assert.NotNil(t, service.broadcast)
}

func TestWebSocketMessageTypes(t *testing.T) {
	// 测试消息类型常量
	assert.Equal(t, "hello", MessageTypeHello)
	assert.Equal(t, "challenge", MessageTypeChallenge)
	assert.Equal(t, "answer", MessageTypeAnswer)
	assert.Equal(t, "result", MessageTypeResult)
	assert.Equal(t, "ping", MessageTypePing)
	assert.Equal(t, "pong", MessageTypePong)
	assert.Equal(t, "error", MessageTypeError)
	assert.Equal(t, "close", MessageTypeClose)
}

func TestVerificationStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", VerificationStatusPending)
	assert.Equal(t, "success", VerificationStatusSuccess)
	assert.Equal(t, "failed", VerificationStatusFailed)
	assert.Equal(t, "expired", VerificationStatusExpired)
}

func TestWebSocketMessageStructure(t *testing.T) {
	payload := json.RawMessage(`{"key":"value"}`)
	msg := WebSocketMessage{
		Type:      MessageTypeHello,
		SessionID: "test-session-123",
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}

	assert.Equal(t, MessageTypeHello, msg.Type)
	assert.Equal(t, "test-session-123", msg.SessionID)
	assert.NotNil(t, msg.Payload)
	assert.Greater(t, msg.Timestamp, int64(0))
}

func TestHelloPayload(t *testing.T) {
	payload := HelloPayload{
		ClientID:  "test-client-456",
		UserAgent: "Mozilla/5.0",
	}

	assert.Equal(t, "test-client-456", payload.ClientID)
	assert.Equal(t, "Mozilla/5.0", payload.UserAgent)
}

func TestChallengePayload(t *testing.T) {
	data := map[string]interface{}{
		"puzzle": "some-puzzle-data",
		"image":  "base64-image-data",
	}
	payload := ChallengePayload{
		ChallengeID: "challenge-789",
		Type:        "slider",
		Data:        data,
		ExpiresAt:   time.Now().Add(2 * time.Minute).Unix(),
	}

	assert.Equal(t, "challenge-789", payload.ChallengeID)
	assert.Equal(t, "slider", payload.Type)
	assert.NotNil(t, payload.Data)
	assert.Equal(t, "some-puzzle-data", payload.Data["puzzle"])
}

func TestAnswerPayload(t *testing.T) {
	data := map[string]interface{}{
		"position": 123,
		"trajectory": []int{1, 2, 3},
	}
	payload := AnswerPayload{
		ChallengeID: "challenge-789",
		Data:        data,
	}

	assert.Equal(t, "challenge-789", payload.ChallengeID)
	assert.NotNil(t, payload.Data)
}

func TestResultPayload(t *testing.T) {
	payload := ResultPayload{
		ChallengeID: "challenge-789",
		Success:     true,
		Message:     "Verification successful",
		Token:       "hjtpx_token_abc123",
		RiskScore:   25,
	}

	assert.Equal(t, "challenge-789", payload.ChallengeID)
	assert.True(t, payload.Success)
	assert.Equal(t, "Verification successful", payload.Message)
	assert.Equal(t, "hjtpx_token_abc123", payload.Token)
	assert.Equal(t, 25, payload.RiskScore)
}

func TestErrorPayload(t *testing.T) {
	payload := ErrorPayload{
		Code:    "INVALID_REQUEST",
		Message: "The request was invalid",
	}

	assert.Equal(t, "INVALID_REQUEST", payload.Code)
	assert.Equal(t, "The request was invalid", payload.Message)
}

func TestGenerateVerificationToken(t *testing.T) {
	sessionID := "test-session-123"
	token := GenerateVerificationToken(sessionID)
	
	assert.NotEmpty(t, token)
	assert.Contains(t, token, "hjtpx_")
	assert.Contains(t, token, sessionID)
}

func TestGetSessionCount(t *testing.T) {
	service := NewWebSocketService()
	count := service.GetSessionCount()
	assert.Equal(t, 0, count)
}

func TestGetActiveSessions(t *testing.T) {
	service := NewWebSocketService()
	sessions := service.GetActiveSessions()
	assert.Empty(t, sessions)
	assert.Len(t, sessions, 0)
}

func TestServiceErrors(t *testing.T) {
	assert.Equal(t, "connection closed", ErrConnectionClosed.Error())
	assert.Equal(t, "session not found", ErrSessionNotFound.Error())
	assert.Equal(t, "invalid message type", ErrInvalidMessageType.Error())
}

func TestTimeoutConstants(t *testing.T) {
	assert.Greater(t, ConnectionTimeout, time.Duration(0))
	assert.Greater(t, PingInterval, time.Duration(0))
	assert.Greater(t, ReadDeadline, time.Duration(0))
	assert.Greater(t, WriteDeadline, time.Duration(0))
}

func TestWebSocketMessageJSONSerialization(t *testing.T) {
	msg := WebSocketMessage{
		Type:      MessageTypeChallenge,
		SessionID: "test-session",
		Payload:   json.RawMessage(`{"test":"data"}`),
		Timestamp: time.Now().Unix(),
	}

	// 测试序列化
	data, err := json.Marshal(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// 测试反序列化
	var decoded WebSocketMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.SessionID, decoded.SessionID)
}

func TestChallengePayloadJSON(t *testing.T) {
	payload := ChallengePayload{
		ChallengeID: "ch-123",
		Type:        "slider",
		Data: map[string]interface{}{
			"x": 100,
			"y": 200,
		},
		ExpiresAt: time.Now().Unix(),
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded ChallengePayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.ChallengeID, decoded.ChallengeID)
	assert.Equal(t, payload.Type, decoded.Type)
}

func TestResultPayloadJSON(t *testing.T) {
	payload := ResultPayload{
		ChallengeID: "ch-123",
		Success:     true,
		Message:     "Success",
		Token:       "token-xyz",
		RiskScore:   10,
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded ResultPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.ChallengeID, decoded.ChallengeID)
	assert.Equal(t, payload.Success, decoded.Success)
	assert.Equal(t, payload.Message, decoded.Message)
	assert.Equal(t, payload.Token, decoded.Token)
	assert.Equal(t, payload.RiskScore, decoded.RiskScore)
}
