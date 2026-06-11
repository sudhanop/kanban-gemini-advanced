package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
)

// MessageType defines the type of realtime message
type MessageType string

const (
	MsgTaskMoved      MessageType = "task_moved"
	MsgTaskCreated    MessageType = "task_created"
	MsgTaskUpdated    MessageType = "task_updated"
	MsgTaskDeleted    MessageType = "task_deleted"
	MsgCommentAdded   MessageType = "comment_added"
	MsgTyping         MessageType = "typing"
	MsgPresence       MessageType = "presence"
	MsgNotification   MessageType = "notification"
	MsgChatMessage    MessageType = "chat_message"
	MsgBoardSync      MessageType = "board_sync"
	MsgUserJoined     MessageType = "user_joined"
	MsgUserLeft       MessageType = "user_left"
	MsgColumnReordered MessageType = "column_reordered"
)

// Message is the envelope for all WebSocket messages
type Message struct {
	Type      MessageType    `json:"type"`
	RoomID    string         `json:"room_id"`
	UserID    string         `json:"user_id"`
	UserName  string         `json:"user_name"`
	UserAvatar string        `json:"user_avatar,omitempty"`
	Payload   interface{}    `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID      string
	UserID  string
	Email   string
	Name    string
	Avatar  string
	Rooms   map[string]bool
	Conn    *websocket.Conn
	Send    chan []byte
	mu      sync.Mutex
}

// Hub manages all WebSocket connections
type Hub struct {
	clients    map[string]*Client
	rooms      map[string]map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	logger     *zap.Logger
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

type BroadcastMessage struct {
	RoomID  string
	Message []byte
	Exclude string // client ID to exclude
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]map[string]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *BroadcastMessage, 1024),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			h.logger.Debug("Client connected", zap.String("client", client.ID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				for roomID := range client.Rooms {
					h.leaveRoom(client, roomID)
				}
				close(client.Send)
			}
			h.mu.Unlock()
			h.logger.Debug("Client disconnected", zap.String("client", client.ID))

		case msg := <-h.broadcast:
			h.mu.RLock()
			if room, ok := h.rooms[msg.RoomID]; ok {
				for _, client := range room {
					if client.ID == msg.Exclude {
						continue
					}
					select {
					case client.Send <- msg.Message:
					default:
						// Buffer full, skip
					}
				}
			}
			h.mu.RUnlock()

		case <-h.ctx.Done():
			return
		}
	}
}

func (h *Hub) Stop() {
	h.cancel()
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[string]*Client)
	}
	h.rooms[roomID][client.ID] = client
	client.Rooms[roomID] = true

	// Notify room about new presence
	h.broadcastToRoom(roomID, &Message{
		Type:       MsgUserJoined,
		RoomID:     roomID,
		UserID:     client.UserID,
		UserName:   client.Name,
		UserAvatar: client.Avatar,
		Timestamp:  time.Now(),
	}, "")
}

// LeaveRoom removes a client from a room
func (h *Hub) leaveRoom(client *Client, roomID string) {
	if room, ok := h.rooms[roomID]; ok {
		delete(room, client.ID)
		delete(client.Rooms, roomID)
		if len(room) == 0 {
			delete(h.rooms, roomID)
		}

		// Notify room about departure
		h.broadcastToRoom(roomID, &Message{
			Type:      MsgUserLeft,
			RoomID:    roomID,
			UserID:    client.UserID,
			UserName:  client.Name,
			Timestamp: time.Now(),
		}, client.ID)
	}
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(roomID string, msg *Message, excludeClientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcastToRoom(roomID, msg, excludeClientID)
}

func (h *Hub) broadcastToRoom(roomID string, msg *Message, excludeClientID string) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal message", zap.Error(err))
		return
	}



	// Also send locally
	if room, ok := h.rooms[roomID]; ok {
		for _, client := range room {
			if client.ID == excludeClientID {
				continue
			}
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

// SendToUser sends a message directly to a specific user
func (h *Hub) SendToUser(userID string, msg *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, _ := json.Marshal(msg)
	for _, client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

// GetRoomPresence returns online users in a room
func (h *Hub) GetRoomPresence(roomID string) []map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var users []map[string]string
	if room, ok := h.rooms[roomID]; ok {
		for _, client := range room {
			users = append(users, map[string]string{
				"user_id": client.UserID,
				"name":    client.Name,
				"avatar":  client.Avatar,
			})
		}
	}
	return users
}



// HandleConnection is the Fiber WebSocket handler
func (h *Hub) HandleConnection(c *websocket.Conn) {
	userID := c.Locals("user_id").(string)
	userName := c.Locals("user_name").(string)
	userAvatar := c.Locals("user_avatar").(string)

	client := &Client{
		ID:      userID + "-" + time.Now().Format("20060102150405"),
		UserID:  userID,
		Name:    userName,
		Avatar:  userAvatar,
		Rooms:   make(map[string]bool),
		Conn:    c,
		Send:    make(chan []byte, 256),
	}

	h.register <- client

	// Start write pump
	go client.writePump()

	// Read pump (blocking)
	client.readPump(h)
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	type IncomingMsg struct {
		Type   string          `json:"type"`
		RoomID string          `json:"room_id"`
		Data   json.RawMessage `json:"data"`
	}

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var incoming IncomingMsg
		if err := json.Unmarshal(msgBytes, &incoming); err != nil {
			continue
		}

		switch incoming.Type {
		case "join_room":
			hub.JoinRoom(c, incoming.RoomID)
		case "leave_room":
			hub.mu.Lock()
			hub.leaveRoom(c, incoming.RoomID)
			hub.mu.Unlock()
		case "ping":
			c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		case "chat_message":
			hub.BroadcastToRoom(incoming.RoomID, &Message{
				Type:      MsgChatMessage,
				RoomID:    incoming.RoomID,
				UserID:    c.UserID,
				UserName:  c.Name,
				Payload:   incoming.Data,
				Timestamp: time.Now(),
			}, "")
		case "typing":
			hub.BroadcastToRoom(incoming.RoomID, &Message{
				Type:     MsgTyping,
				RoomID:   incoming.RoomID,
				UserID:   c.UserID,
				UserName: c.Name,
				Timestamp: time.Now(),
			}, c.ID)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
