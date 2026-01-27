package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rohit21755/groveserverv2/internal/auth"
	"github.com/rohit21755/groveserverv2/internal/env"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - customize in production
		return true
	},
}

// handleWSConnection handles WebSocket connections with JWT authentication
func handleWSConnection(hub *Hub, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from query parameter or Authorization header
		tokenString := r.URL.Query().Get("token")
		
		// If token from query param has "Bearer " prefix, remove it
		if tokenString != "" {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
			tokenString = strings.TrimSpace(tokenString)
		}
		
		if tokenString == "" {
			// Try Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				} else if len(parts) == 1 {
					// Sometimes the header might not have "Bearer " prefix
					tokenString = parts[0]
				}
			}
		}

		if tokenString == "" {
			log.Printf("WebSocket connection rejected: No token provided")
			http.Error(w, "Token required", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
		if err != nil {
			log.Printf("WebSocket JWT validation error: %v, token length: %d", err, len(tokenString))
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
		
		log.Printf("WebSocket connection authenticated: user_id=%s, role=%s", claims.UserID, claims.Role)

		// Upgrade connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Create client
		client := &Client{
			ID:       claims.UserID,
			Conn:     conn,
			Send:     make(chan []byte, 256),
			Hub:      hub,
			UserID:   claims.UserID,
			UserRole: claims.Role,
		}

		// Register client
		hub.register <- client

		// Start goroutines for reading and writing
		go client.writePump()
		go client.readPump()
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages (e.g., chat messages, ping/pong)
		// For now, we just log them
		var wsMessage WSMessage
		if err := json.Unmarshal(message, &wsMessage); err == nil {
			log.Printf("Received message from user %s: type=%s", c.UserID, wsMessage.Type)
			// TODO: Handle different message types (chat, etc.)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
