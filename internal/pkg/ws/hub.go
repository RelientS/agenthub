package ws

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// MessageEnvelope wraps a WebSocket message with routing metadata.
type MessageEnvelope struct {
	Type        string          `json:"type"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	AgentID     *uuid.UUID      `json:"agent_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

// InboundMessage represents a message received from a connected client.
type InboundMessage struct {
	Client  *Client
	Message []byte
}

// MessageHandler is a callback for processing inbound WebSocket messages.
type MessageHandler func(client *Client, message []byte)

// Hub manages WebSocket connections grouped by workspace and agent.
// It supports registering/unregistering clients, broadcasting to workspaces,
// and sending messages to specific agents.
type Hub struct {
	mu sync.RWMutex

	// clients maps workspace_id -> agent_id -> *Client
	clients map[uuid.UUID]map[uuid.UUID]*Client

	// register channel for incoming client registrations
	register chan *Client

	// unregister channel for client disconnections
	unregister chan *Client

	// inbound messages from clients
	inbound chan InboundMessage
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[uuid.UUID]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		inbound:    make(chan InboundMessage, 1024),
	}
}

// Run starts the hub's main loop, processing register, unregister, and inbound
// message events. The optional handler is invoked for each inbound message.
// This method blocks and should be run in a goroutine.
func (h *Hub) Run(handler MessageHandler) {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
			log.Info().
				Str("agent_id", client.AgentID.String()).
				Str("workspace_id", client.WorkspaceID.String()).
				Msg("ws client registered")

		case client := <-h.unregister:
			h.removeClient(client)
			log.Info().
				Str("agent_id", client.AgentID.String()).
				Str("workspace_id", client.WorkspaceID.String()).
				Msg("ws client unregistered")

		case msg := <-h.inbound:
			if handler != nil {
				handler(msg.Client, msg.Message)
			}
		}
	}
}

// Register queues a client for registration.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister queues a client for removal.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Inbound queues an inbound message from a client.
func (h *Hub) Inbound(client *Client, message []byte) {
	h.inbound <- InboundMessage{Client: client, Message: message}
}

// addClient adds a client to the hub's internal map.
func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.WorkspaceID]; !ok {
		h.clients[client.WorkspaceID] = make(map[uuid.UUID]*Client)
	}

	// If there is an existing client for this agent, close its send channel.
	if existing, ok := h.clients[client.WorkspaceID][client.AgentID]; ok {
		close(existing.Send)
	}

	h.clients[client.WorkspaceID][client.AgentID] = client
}

// removeClient removes a client from the hub's internal map and closes its send channel.
func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if agents, ok := h.clients[client.WorkspaceID]; ok {
		if existing, ok := agents[client.AgentID]; ok && existing == client {
			close(client.Send)
			delete(agents, client.AgentID)
			if len(agents) == 0 {
				delete(h.clients, client.WorkspaceID)
			}
		}
	}
}

// SendToAgent delivers a message to a specific agent in a workspace.
// Returns true if the message was queued, false if the agent is not connected.
func (h *Hub) SendToAgent(workspaceID, agentID uuid.UUID, data []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if agents, ok := h.clients[workspaceID]; ok {
		if client, ok := agents[agentID]; ok {
			select {
			case client.Send <- data:
				return true
			default:
				return false
			}
		}
	}
	return false
}

// BroadcastToWorkspace sends a message to all connected agents in a workspace.
func (h *Hub) BroadcastToWorkspace(workspaceID uuid.UUID, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if agents, ok := h.clients[workspaceID]; ok {
		for _, client := range agents {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

// BroadcastToWorkspaceExcept sends a message to all agents in a workspace
// except for the specified agent.
func (h *Hub) BroadcastToWorkspaceExcept(workspaceID, excludeAgentID uuid.UUID, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if agents, ok := h.clients[workspaceID]; ok {
		for id, client := range agents {
			if id == excludeAgentID {
				continue
			}
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

// WorkspaceAgentCount returns the number of connected agents in a workspace.
func (h *Hub) WorkspaceAgentCount(workspaceID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if agents, ok := h.clients[workspaceID]; ok {
		return len(agents)
	}
	return 0
}

// IsAgentConnected returns true if the specified agent is connected in the workspace.
func (h *Hub) IsAgentConnected(workspaceID, agentID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if agents, ok := h.clients[workspaceID]; ok {
		_, connected := agents[agentID]
		return connected
	}
	return false
}
