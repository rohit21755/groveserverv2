package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// Constants are now defined in connection.go to avoid duplication

// LeaderboardClient represents a WebSocket client for leaderboard updates
type LeaderboardClient struct {
	conn            *websocket.Conn
	send            chan []byte
	leaderboardType string // "pan-india", "state", "college"
	scopeID         string // state_id or college_id (for state/college leaderboards)
	hub             *LeaderboardHub
}

// LeaderboardHub maintains the set of active clients and broadcasts messages
type LeaderboardHub struct {
	// Registered clients
	clients map[*LeaderboardClient]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *LeaderboardClient

	// Unregister requests from clients
	unregister chan *LeaderboardClient

	// Mutex for thread safety
	mu sync.RWMutex

	// Redis client for pub/sub
	redisClient *db.Redis

	// Postgres for fetching leaderboard data
	postgres *db.Postgres
}

// NewLeaderboardHub creates a new leaderboard hub
func NewLeaderboardHub(redisClient *db.Redis, postgres *db.Postgres) *LeaderboardHub {
	return &LeaderboardHub{
		clients:     make(map[*LeaderboardClient]bool),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *LeaderboardClient),
		unregister:  make(chan *LeaderboardClient),
		redisClient: redisClient,
		postgres:    postgres,
	}
}

// Run starts the hub
func (h *LeaderboardHub) Run() {
	// Subscribe to Redis pub/sub for leaderboard updates
	go h.subscribeToUpdates()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Leaderboard client connected: type=%s, scope=%s", client.leaderboardType, client.scopeID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Leaderboard client disconnected: type=%s, scope=%s", client.leaderboardType, client.scopeID)

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

// subscribeToUpdates subscribes to Redis pub/sub for leaderboard updates
func (h *LeaderboardHub) subscribeToUpdates() {
	ctx := context.Background()
	pubsub := h.redisClient.Client.Subscribe(ctx, "leaderboard:updates")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		// Broadcast the update to all connected clients
		h.broadcast <- []byte(msg.Payload)
	}
}

// BroadcastLeaderboardUpdate publishes a leaderboard update to Redis
func BroadcastLeaderboardUpdate(redisClient *db.Redis, leaderboardType string, scopeID string) {
	ctx := context.Background()
	update := map[string]interface{}{
		"type":      "leaderboard_update",
		"scope":     leaderboardType,
		"scope_id":  scopeID,
		"timestamp": time.Now().Unix(),
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		log.Printf("Error marshaling leaderboard update: %v", err)
		return
	}

	err = redisClient.Client.Publish(ctx, "leaderboard:updates", updateJSON).Err()
	if err != nil {
		log.Printf("Error publishing leaderboard update: %v", err)
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *LeaderboardClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *LeaderboardClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleLeaderboardWS handles WebSocket connections for leaderboard updates
func handleLeaderboardWS(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	// Create hub if it doesn't exist (singleton pattern)
	var hub *LeaderboardHub
	hubOnce.Do(func() {
		hub = NewLeaderboardHub(redisClient, postgres)
		go hub.Run()
	})

	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading to WebSocket: %v", err)
			return
		}

		// Get leaderboard type and scope from query parameters
		leaderboardType := r.URL.Query().Get("type")
		if leaderboardType == "" {
			leaderboardType = "pan-india"
		}

		scopeID := r.URL.Query().Get("scope_id") // state_id or college_id

		// Create client
		client := &LeaderboardClient{
			conn:            conn,
			send:            make(chan []byte, 256),
			leaderboardType: leaderboardType,
			scopeID:         scopeID,
			hub:             hub,
		}

		// Register client
		hub.register <- client

		// Send initial leaderboard data
		go func() {
			leaderboardStore := store.NewLeaderboardStore(postgres)
			var entries []store.LeaderboardEntry
			var err error

			// Get period from query parameter, default to "all"
			period := r.URL.Query().Get("period")
			if period == "" {
				period = "all"
			}
			if period != "all" && period != "weekly" && period != "monthly" {
				period = "all"
			}

			switch leaderboardType {
			case "pan-india":
				entries, err = leaderboardStore.GetPanIndiaLeaderboard(r.Context(), 100, 0, period)
			case "state":
				if scopeID == "" {
					return
				}
				entries, err = leaderboardStore.GetStateLeaderboard(r.Context(), scopeID, 100, 0, period)
			case "college":
				if scopeID == "" {
					return
				}
				entries, err = leaderboardStore.GetCollegeLeaderboard(r.Context(), scopeID, 100, 0, period)
			default:
				return
			}

			if err != nil {
				log.Printf("Error getting initial leaderboard: %v", err)
				return
			}

			// Adjust ranks
			for i := range entries {
				entries[i].Rank = i + 1
			}

			response := map[string]interface{}{
				"type":     "leaderboard_data",
				"scope":    leaderboardType,
				"scope_id": scopeID,
				"entries":  entries,
			}

			responseJSON, err := json.Marshal(response)
			if err == nil {
				select {
				case client.send <- responseJSON:
				case <-time.After(5 * time.Second):
					log.Printf("Timeout sending initial leaderboard data")
				}
			}
		}()

		// Start pumps
		go client.writePump()
		go client.readPump()
	}
}

var (
	hub     *LeaderboardHub
	hubOnce sync.Once
)
