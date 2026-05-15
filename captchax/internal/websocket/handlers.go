package websocket

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	hub       *Hub
	jwtSecret []byte
}

func NewHandler(hub *Hub, jwtSecret string) *Handler {
	return &Handler{
		hub:       hub,
		jwtSecret: []byte(jwtSecret),
	}
}

type wsClaims struct {
	AdminID  uint   `json:"admin_id"`
	Username string `json:"username"`
	UserID   uint   `json:"user_id"`
	jwt.RegisteredClaims
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	userID := h.authenticate(c)
	if userID == 0 {
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.hub.RegisterClient(userID, conn)
}

func (h *Handler) authenticate(c *gin.Context) uint {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		authHeader := c.GetHeader("Sec-WebSocket-Protocol")
		if authHeader != "" {
			parts := strings.Split(authHeader, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "token.") {
					tokenStr = strings.TrimPrefix(part, "token.")
					break
				}
			}
		}
	}

	if tokenStr == "" {
		http.Error(c.Writer, "authentication required", http.StatusUnauthorized)
		return 0
	}

	token, err := jwt.ParseWithClaims(tokenStr, &wsClaims{}, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(c.Writer, "invalid token", http.StatusUnauthorized)
		return 0
	}

	claims, ok := token.Claims.(*wsClaims)
	if !ok {
		http.Error(c.Writer, "invalid token claims", http.StatusUnauthorized)
		return 0
	}

	if claims.AdminID > 0 {
		return claims.AdminID
	}
	if claims.UserID > 0 {
		return claims.UserID
	}

	return 0
}

func (h *Handler) GetHub() *Hub {
	return h.hub
}

func (h *Handler) NotifyUser(userID uint, notificationType string, data interface{}) {
	dataBytes, _ := json.Marshal(data)
	h.hub.SendToUser(userID, &Message{
		Type: notificationType,
		Data: dataBytes,
	})
}