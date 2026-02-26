package ws

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// maxMessageSize is the maximum message size allowed from peer.
	maxMessageSize = 65536

	// sendBufferSize is the buffer size for the client's outbound message channel.
	sendBufferSize = 256
)

// Client represents a single WebSocket connection from an agent.
type Client struct {
	Hub         *Hub
	Conn        *websocket.Conn
	Send        chan []byte
	AgentID     uuid.UUID
	WorkspaceID uuid.UUID

	// PingInterval controls how frequently the server sends ping frames.
	PingInterval time.Duration

	// PongTimeout is how long the server waits for a pong response.
	PongTimeout time.Duration
}

// NewClient creates a new Client with the given parameters and default settings.
func NewClient(hub *Hub, conn *websocket.Conn, agentID, workspaceID uuid.UUID, pingInterval, pongTimeout time.Duration) *Client {
	return &Client{
		Hub:          hub,
		Conn:         conn,
		Send:         make(chan []byte, sendBufferSize),
		AgentID:      agentID,
		WorkspaceID:  workspaceID,
		PingInterval: pingInterval,
		PongTimeout:  pongTimeout,
	}
}

// ReadPump reads messages from the WebSocket connection and forwards them to the hub.
// It runs in its own goroutine per client. When the connection is closed or an error
// occurs, the client is unregistered from the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)

	// Set initial read deadline based on pong timeout.
	if err := c.Conn.SetReadDeadline(time.Now().Add(c.PongTimeout + c.PingInterval)); err != nil {
		log.Error().Err(err).Msg("failed to set initial read deadline")
		return
	}

	// When we receive a pong, extend the read deadline.
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(c.PongTimeout + c.PingInterval))
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Warn().
					Err(err).
					Str("agent_id", c.AgentID.String()).
					Msg("unexpected websocket close")
			}
			return
		}

		// Forward the message to the hub for processing.
		c.Hub.Inbound(c, message)
	}
}

// WritePump pumps messages from the send channel to the WebSocket connection.
// It also sends periodic ping frames to keep the connection alive.
// It runs in its own goroutine per client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(c.PingInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Error().Err(err).Msg("failed to set write deadline")
				return
			}

			if !ok {
				// The hub closed the channel; send a close frame.
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			// Write any queued messages in the send buffer to reduce write syscalls.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				if _, err := w.Write([]byte("\n")); err != nil {
					break
				}
				if _, err := w.Write(<-c.Send); err != nil {
					break
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
