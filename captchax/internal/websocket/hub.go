package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	Clients    map[uint]map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
	onMessage  func(client *Client, msg *Message)
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[uint]map[*Client]bool),
		Broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) SetOnMessage(handler func(client *Client, msg *Message)) {
	h.onMessage = handler
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; !ok {
				h.Clients[client.ID] = make(map[*Client]bool)
			}
			h.Clients[client.ID][client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.Clients[client.ID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.Clients, client.ID)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, clients := range h.Clients {
				for client := range clients {
					select {
					case client.Send <- message:
					default:
						go func(c *Client) {
							h.Unregister <- c
						}(client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastMessage(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.Broadcast <- data
}

func (h *Hub) SendToUser(userID uint, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.Clients[userID]; ok {
		for client := range clients {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

func (h *Hub) SendToUserIDs(userIDs []uint, msg *Message) {
	for _, userID := range userIDs {
		h.SendToUser(userID, msg)
	}
}

func (h *Hub) GetOnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.Clients {
		count += len(clients)
	}
	return count
}

func (h *Hub) GetOnlineUsers() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]uint, 0, len(h.Clients))
	for userID := range h.Clients {
		users = append(users, userID)
	}
	return users
}

func (h *Hub) IsUserOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.Clients[userID]
	return ok && len(clients) > 0
}

func (h *Hub) handleIncomingMessage(client *Client, msg *Message) {
	if h.onMessage != nil {
		h.onMessage(client, msg)
	}
}

func (h *Hub) RegisterClient(userID uint, conn *websocket.Conn) *Client {
	client := &Client{
		ID:   userID,
		Hub:  h,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	h.Register <- client

	go client.WritePump()
	go client.ReadPump()

	return client
}