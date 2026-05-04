package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yibaiba/dramora/internal/realtime"
	"github.com/yibaiba/dramora/internal/service"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// ClientConn represents a WebSocket client connection
type ClientConn struct {
	id      string
	userID  string
	orgID   string
	ws      *websocket.Conn
	send    chan *WebSocketMessage
	done    chan struct{}
	manager *WebSocketManager
}

// WebSocketManager manages all WebSocket connections
type WebSocketManager struct {
	clients    map[string]*ClientConn
	clientsMu  sync.RWMutex
	broadcast  chan *broadcastMessage
	register   chan *ClientConn
	unregister chan *ClientConn
	logger     *slog.Logger
}

type broadcastMessage struct {
	orgID   string
	message *WebSocketMessage
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper CORS checking
		return true
	},
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(logger *slog.Logger) *WebSocketManager {
	manager := &WebSocketManager{
		clients:    make(map[string]*ClientConn),
		broadcast:  make(chan *broadcastMessage, 256),
		register:   make(chan *ClientConn, 256),
		unregister: make(chan *ClientConn, 256),
		logger:     logger,
	}
	go manager.run()
	return manager
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract user from context (set by auth middleware)
	auth, ok := service.RequestAuthFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	// Create client connection
	client := &ClientConn{
		id:      generateClientID(),
		userID:  auth.UserID,
		orgID:   auth.OrganizationID,
		ws:      ws,
		send:    make(chan *WebSocketMessage, 256),
		done:    make(chan struct{}),
		manager: h.wsManager,
	}

	h.wsManager.register <- client
	h.logger.Info("WebSocket client connected",
		"client_id", client.id,
		"user_id", auth.UserID,
		"org_id", auth.OrganizationID,
	)

	// Start reading and writing goroutines
	go client.readPump()
	go client.writePump()
}

// readPump reads messages from client (for future use, heartbeat, etc.)
func (c *ClientConn) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.ws.Close()
	}()

	c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.manager.logger.Error("WebSocket error", "error", err)
			}
			return
		}

		// Handle client messages (future use)
		_ = message
	}
}

// writePump writes messages to client
func (c *ClientConn) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case msg := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteJSON(msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.manager.logger.Error("WebSocket write error", "error", err)
				}
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// run manages client registration, unregistration, and broadcasting
func (m *WebSocketManager) run() {
	for {
		select {
		case client := <-m.register:
			m.clientsMu.Lock()
			m.clients[client.id] = client
			m.clientsMu.Unlock()

		case client := <-m.unregister:
			m.clientsMu.Lock()
			if _, ok := m.clients[client.id]; ok {
				delete(m.clients, client.id)
				close(client.send)
				m.logger.Info("WebSocket client disconnected", "client_id", client.id)
			}
			m.clientsMu.Unlock()

		case broadcast := <-m.broadcast:
			m.broadcastToOrg(broadcast.orgID, broadcast.message)
		}
	}
}

// broadcastToOrg broadcasts a message to all clients in an organization
func (m *WebSocketManager) broadcastToOrg(orgID string, msg *WebSocketMessage) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	for _, client := range m.clients {
		if client.orgID == orgID {
			select {
			case client.send <- msg:
			default:
				// Client's send channel is full, skip (don't block)
				m.logger.Warn("Client send channel full, skipping message",
					"client_id", client.id,
				)
			}
		}
	}
}

// BroadcastEvent broadcasts a realtime event to WebSocket clients
func (m *WebSocketManager) BroadcastEvent(orgID string, event *realtime.Event) {
	msg := &WebSocketMessage{
		Type: string(event.Type),
		Data: event.Data,
	}

	select {
	case m.broadcast <- &broadcastMessage{
		orgID:   orgID,
		message: msg,
	}:
	default:
		// Broadcast channel is full, skip (don't block caller)
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return time.Now().Format("2006-01-02T15:04:05") + "-" + strconv.Itoa(int(time.Now().UnixNano()%1000000))
}

// WebSocketHandler is the HTTP handler for WebSocket endpoints
type WebSocketHandler struct {
	wsManager *WebSocketManager
	logger    *slog.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(wsManager *WebSocketManager, logger *slog.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager: wsManager,
		logger:    logger,
	}
}
