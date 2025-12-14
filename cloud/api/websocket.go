// Package api provides WebSocket support for real-time updates
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`    // event type: instance_update, log, notification
	Payload interface{} `json:"payload"` // event data
}

// Client represents a connected WebSocket client
type Client struct {
	conn   *websocket.Conn
	userID string
	send   chan []byte
}

// WSHub maintains active WebSocket connections
type WSHub struct {
	clients     map[*Client]bool
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	userClients map[string][]*Client // Map userID to their clients
	mu          sync.RWMutex
}

// NewWSHub creates a new WebSocket hub
func NewWSHub() *WSHub {
	return &WSHub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		userClients: make(map[string][]*Client),
	}
}

// Run starts the hub's event loop
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.userClients[client.userID] = append(h.userClients[client.userID], client)
			h.mu.Unlock()
			log.Printf("WebSocket client connected: user=%s", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Remove from user clients
				clients := h.userClients[client.userID]
				for i, c := range clients {
					if c == client {
						h.userClients[client.userID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected: user=%s", client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToUser sends a message to all clients of a specific user
func (h *WSHub) SendToUser(userID string, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal WS message: %v", err)
		return
	}

	h.mu.RLock()
	clients := h.userClients[userID]
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			// Client buffer full, skip
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WSHub) Broadcast(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal WS message: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		// Buffer full
	}
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(c echo.Context) error {
	// Authenticate via query param or header
	token := c.QueryParam("token")
	if token == "" {
		token = c.Request().Header.Get("Authorization")
	}

	userID := "demo" // Default for demo mode
	if token != "" && token != "cm_demo" {
		// Validate JWT token
		claims, err := s.validateJWT(token)
		if err == nil {
			userID = claims.UserID
		}
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return err
	}

	client := &Client{
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, 256),
	}

	s.wsHub.register <- client

	// Start client goroutines
	go s.wsWritePump(client)
	go s.wsReadPump(client)

	return nil
}

// wsWritePump pumps messages from hub to client
func (s *Server) wsWritePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Drain any queued messages
			n := len(client.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// wsReadPump reads messages from client
func (s *Server) wsReadPump(client *Client) {
	defer func() {
		s.wsHub.unregister <- client
		client.conn.Close()
	}()

	client.conn.SetReadLimit(512)
	_ = client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		_ = client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages (e.g., subscribe to specific instance)
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Process based on message type
		switch msg.Type {
		case "subscribe_instance":
			// Client wants to subscribe to an instance's logs
			// Could implement per-instance subscription here
		case "ping":
			// Respond with pong
			s.wsHub.SendToUser(client.userID, WSMessage{Type: "pong"})
		}
	}
}

// NotifyInstanceUpdate sends an instance update to the owner
func (s *Server) NotifyInstanceUpdate(userID string, instanceID string, status string, details map[string]interface{}) {
	if s.wsHub == nil {
		return
	}

	payload := map[string]interface{}{
		"instance_id": instanceID,
		"status":      status,
		"timestamp":   time.Now().UTC(),
	}
	for k, v := range details {
		payload[k] = v
	}

	s.wsHub.SendToUser(userID, WSMessage{
		Type:    "instance_update",
		Payload: payload,
	})
}
