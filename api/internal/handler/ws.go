package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"

	"openfms/api/internal/model"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins in development, configure for production
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// Heartbeat interval
	pingInterval = 30 * time.Second
	// Write timeout
	writeTimeout = 10 * time.Second
)

// LocationMessage represents a location update message sent to clients
type LocationMessage struct {
	DeviceID  string  `json:"device_id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Speed     float64 `json:"speed"`
	Direction float64 `json:"direction"`
	Timestamp int64   `json:"timestamp"`
	Status    int     `json:"status,omitempty"`
}

// WSMessage represents a WebSocket message from client
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *WSHub
	DeviceID string // Filter by device ID (empty means all devices)
}

// WSHub manages WebSocket clients and broadcasts messages
type WSHub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	natsConn   *nats.Conn
	sub        *nats.Subscription
	alarmSub   *nats.Subscription
	mu         sync.RWMutex
}

// NewWSHub creates a new WebSocket hub
func NewWSHub(nc *nats.Conn) *WSHub {
	return &WSHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		natsConn:   nc,
	}
}

// Run starts the hub's event loop
func (h *WSHub) Run() {
	// Subscribe to NATS location topic
	sub, err := h.natsConn.Subscribe("fms.uplink.LOCATION", func(msg *nats.Msg) {
		var locMsg LocationMessage
		if err := json.Unmarshal(msg.Data, &locMsg); err != nil {
			log.Printf("[WS] Failed to unmarshal location message: %v", err)
			return
		}

		// Broadcast to all connected WebSocket clients
		data, err := json.Marshal(map[string]interface{}{
			"type": "location",
			"data": locMsg,
		})
		if err != nil {
			log.Printf("[WS] Failed to marshal broadcast message: %v", err)
			return
		}

		h.broadcast <- data
	})
	if err != nil {
		log.Printf("[WS] Failed to subscribe to NATS: %v", err)
		return
	}
	h.sub = sub

	// Subscribe to NATS alarm topic
	alarmSub, err := h.natsConn.Subscribe("fms.alarm.*", func(msg *nats.Msg) {
		var alarmMsg model.AlarmMessage
		if err := json.Unmarshal(msg.Data, &alarmMsg); err != nil {
			log.Printf("[WS] Failed to unmarshal alarm message: %v", err)
			return
		}

		// Broadcast alarm to all connected clients
		data, err := json.Marshal(map[string]interface{}{
			"type": "alarm",
			"data": alarmMsg,
		})
		if err != nil {
			log.Printf("[WS] Failed to marshal alarm broadcast message: %v", err)
			return
		}

		h.broadcast <- data
	})
	if err != nil {
		log.Printf("[WS] Failed to subscribe to NATS alarm: %v", err)
		return
	}
	h.alarmSub = alarmSub

	log.Println("[WS] Hub started, subscribed to NATS location and alarm updates")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS] Client connected: %s, total clients: %d", client.ID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[WS] Client disconnected: %s, total clients: %d", client.ID, len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.Send <- message:
				default:
					// Client send buffer is full, close connection
					h.unregister <- client
				}
			}
		}
	}
}

// Stop stops the hub and cleans up resources
func (h *WSHub) Stop() {
	if h.sub != nil {
		h.sub.Unsubscribe()
	}
	if h.alarmSub != nil {
		h.alarmSub.Unsubscribe()
	}
	h.mu.Lock()
	for client := range h.clients {
		close(client.Send)
		client.Conn.Close()
		delete(h.clients, client)
	}
	h.mu.Unlock()
}

// GetClientCount returns the number of connected clients
func (h *WSHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastAlarm broadcasts an alarm message to all connected clients
func (h *WSHub) BroadcastAlarm(msg *model.AlarmMessage) error {
	data, err := json.Marshal(map[string]interface{}{
		"type": "alarm",
		"data": msg,
	})
	if err != nil {
		return err
	}

	select {
	case h.broadcast <- data:
		return nil
	default:
		return nil // Don't block if channel is full
	}
}

// ReadPump handles incoming messages from the client
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Client %s read error: %v", c.ID, err)
			}
			break
		}

		// Handle client messages (e.g., subscribe to specific device)
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err == nil {
			switch wsMsg.Type {
			case "subscribe":
				// Client wants to subscribe to specific device
				var data struct {
					DeviceID string `json:"device_id"`
				}
				if err := json.Unmarshal(wsMsg.Data, &data); err == nil && data.DeviceID != "" {
					c.DeviceID = data.DeviceID
					log.Printf("[WS] Client %s subscribed to device %s", c.ID, c.DeviceID)
				}
			case "ping":
				// Client ping, respond with pong
				select {
				case c.Send <- []byte(`{"type":"pong"}`):
				default:
				}
			}
		}
	}
}

// WritePump handles outgoing messages to the client
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WSHandler handles WebSocket connections
type WSHandler struct {
	hub *WSHub
}

// NewWSHandler creates a new WebSocket handler
func NewWSHandler(hub *WSHub) *WSHandler {
	return &WSHandler{hub: hub}
}

// HandleLocation handles WebSocket connections for location updates
func (h *WSHandler) HandleLocation(c *gin.Context) {
	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}

	// Generate client ID
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = generateClientID()
	}

	// Optional: filter by device ID
	deviceID := c.Query("device_id")

	client := &Client{
		ID:       clientID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      h.hub,
		DeviceID: deviceID,
	}

	// Register client
	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()

	// Send welcome message
	welcome := map[string]interface{}{
		"type":    "connected",
		"message": "Connected to OpenFMS location stream",
		"client_id": clientID,
	}
	if data, err := json.Marshal(welcome); err == nil {
		select {
		case client.Send <- data:
		default:
		}
	}
}

// GetStats returns WebSocket hub statistics
func (h *WSHandler) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"connected_clients": h.hub.GetClientCount(),
	})
}

// BroadcastLocation broadcasts a location message to all connected clients
func (h *WSHandler) BroadcastLocation(ctx context.Context, msg *LocationMessage) error {
	data, err := json.Marshal(map[string]interface{}{
		"type": "location",
		"data": msg,
	})
	if err != nil {
		return err
	}

	select {
	case h.hub.broadcast <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(b)
}
