package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/pkg/ws"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// defaultPingInterval is used when no config override is provided.
	defaultPingInterval = 30 * time.Second

	// defaultPongTimeout is used when no config override is provided.
	defaultPongTimeout = 10 * time.Second
)

// wsUpgrader is the default WebSocket upgrader with permissive CheckOrigin for development.
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins; restrict in production via config.
	},
}

// WSHandler handles WebSocket upgrade requests.
type WSHandler struct {
	hub          *ws.Hub
	jwtSecret    string
	pingInterval time.Duration
	pongTimeout  time.Duration
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *ws.Hub, jwtSecret string) *WSHandler {
	return &WSHandler{
		hub:          hub,
		jwtSecret:    jwtSecret,
		pingInterval: defaultPingInterval,
		pongTimeout:  defaultPongTimeout,
	}
}

// NewWSHandlerWithConfig creates a new WSHandler with custom ping/pong settings.
func NewWSHandlerWithConfig(hub *ws.Hub, jwtSecret string, pingInterval, pongTimeout time.Duration) *WSHandler {
	return &WSHandler{
		hub:          hub,
		jwtSecret:    jwtSecret,
		pingInterval: pingInterval,
		pongTimeout:  pongTimeout,
	}
}

// RegisterRoutes registers the WebSocket route on the Gin engine.
func (h *WSHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/ws", h.HandleWebSocket)
}

// wsAgentClaims represents the JWT claims for an agent connecting via WebSocket.
// This mirrors middleware.AgentClaims but is used for query-parameter token
// validation rather than Authorization-header validation.
type wsAgentClaims struct {
	AgentID     uuid.UUID `json:"agent_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	jwt.RegisteredClaims
}

// HandleWebSocket handles GET /ws.
// Query parameters:
//   - token: JWT authentication token (required)
//   - workspace_id: the workspace to connect to (required)
//
// The handler validates the JWT, extracts agent identity, upgrades the
// connection to WebSocket, and registers the client with the hub. The
// client's ReadPump and WritePump goroutines are started automatically.
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// Extract and validate query parameters.
	token := c.Query("token")
	if token == "" {
		models.UnauthorizedError(c, "token query parameter is required")
		return
	}

	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		models.BadRequestError(c, "workspace_id query parameter is required")
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		models.BadRequestError(c, "invalid workspace_id: must be a valid UUID")
		return
	}

	// Parse and validate JWT token.
	claims := &wsAgentClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !parsedToken.Valid {
		models.UnauthorizedError(c, "invalid or expired token")
		return
	}

	agentID := claims.AgentID
	if agentID == uuid.Nil {
		models.UnauthorizedError(c, "invalid agent_id in token")
		return
	}

	// Optionally verify that the workspace_id in the token matches the query param.
	if claims.WorkspaceID != uuid.Nil && claims.WorkspaceID != workspaceID {
		models.ForbiddenError(c, "token workspace_id does not match requested workspace")
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection.
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade already wrote an HTTP error response; do not write again.
		return
	}

	// Create the client using the ws package's constructor and register it.
	client := ws.NewClient(h.hub, conn, agentID, workspaceID, h.pingInterval, h.pongTimeout)
	h.hub.Register(client)

	// Start read and write pumps in goroutines. These methods are provided
	// by the ws.Client and handle ping/pong, read deadlines, and message
	// forwarding to the hub.
	go client.WritePump()
	go client.ReadPump()
}
