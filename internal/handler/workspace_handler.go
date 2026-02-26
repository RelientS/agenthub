package handler

import (
	"net/http"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WorkspaceHandler handles workspace-related HTTP requests.
type WorkspaceHandler struct {
	service *service.WorkspaceService
}

// NewWorkspaceHandler creates a new WorkspaceHandler.
func NewWorkspaceHandler(s *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{service: s}
}

// RegisterRoutes registers workspace routes on the given router group.
func (h *WorkspaceHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	workspaces := rg.Group("/workspaces")
	workspaces.Use(authMiddleware)
	{
		workspaces.POST("", h.CreateWorkspace)
		workspaces.GET("/:id", h.GetWorkspace)
		workspaces.PUT("/:id", h.UpdateWorkspace)
		workspaces.DELETE("/:id", h.DeleteWorkspace)
		workspaces.POST("/join", h.JoinWorkspace)
		workspaces.POST("/:id/leave", h.LeaveWorkspace)
		workspaces.GET("/:id/agents", h.ListAgents)
	}

	agents := rg.Group("/agents")
	agents.Use(authMiddleware)
	{
		agents.POST("/heartbeat", h.Heartbeat)
	}
}

// CreateWorkspace handles POST /api/v1/workspaces.
func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	agentID, err := getAgentID(c)
	if err != nil {
		models.UnauthorizedError(c, "invalid agent identity")
		return
	}

	var input service.CreateWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.Name == "" {
		models.BadRequestError(c, "name is required")
		return
	}

	workspace, err := h.service.CreateWorkspace(c.Request.Context(), input, agentID)
	if err != nil {
		models.InternalError(c, "failed to create workspace: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusCreated, workspace)
}

// GetWorkspace handles GET /api/v1/workspaces/:id.
func (h *WorkspaceHandler) GetWorkspace(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	workspace, err := h.service.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		models.NotFoundError(c, "workspace not found")
		return
	}

	models.SuccessResponse(c, http.StatusOK, workspace)
}

// UpdateWorkspace handles PUT /api/v1/workspaces/:id.
func (h *WorkspaceHandler) UpdateWorkspace(c *gin.Context) {
	agentID, err := getAgentID(c)
	if err != nil {
		models.UnauthorizedError(c, "invalid agent identity")
		return
	}

	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	var input service.UpdateWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	workspace, err := h.service.UpdateWorkspace(c.Request.Context(), workspaceID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to update workspace: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, workspace)
}

// DeleteWorkspace handles DELETE /api/v1/workspaces/:id.
func (h *WorkspaceHandler) DeleteWorkspace(c *gin.Context) {
	agentID, err := getAgentID(c)
	if err != nil {
		models.UnauthorizedError(c, "invalid agent identity")
		return
	}

	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	if err := h.service.DeleteWorkspace(c.Request.Context(), workspaceID, agentID); err != nil {
		models.InternalError(c, "failed to delete workspace: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, gin.H{"deleted": true})
}

// JoinWorkspace handles POST /api/v1/workspaces/join.
func (h *WorkspaceHandler) JoinWorkspace(c *gin.Context) {
	_, err := getAgentID(c)
	if err != nil {
		models.UnauthorizedError(c, "invalid agent identity")
		return
	}

	var input service.JoinWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.InviteCode == "" {
		models.BadRequestError(c, "invite_code is required")
		return
	}
	if input.AgentName == "" {
		models.BadRequestError(c, "agent_name is required")
		return
	}
	if input.AgentRole == "" {
		models.BadRequestError(c, "agent_role is required")
		return
	}
	if !models.IsValidRole(input.AgentRole) {
		models.BadRequestError(c, "invalid agent_role: must be one of frontend, backend, fullstack, tester, devops")
		return
	}

	agent, err := h.service.JoinWorkspace(c.Request.Context(), input)
	if err != nil {
		models.InternalError(c, "failed to join workspace: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, agent)
}

// LeaveWorkspace handles POST /api/v1/workspaces/:id/leave.
func (h *WorkspaceHandler) LeaveWorkspace(c *gin.Context) {
	agentID, err := getAgentID(c)
	if err != nil {
		models.UnauthorizedError(c, "invalid agent identity")
		return
	}

	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	if err := h.service.LeaveWorkspace(c.Request.Context(), workspaceID, agentID); err != nil {
		models.InternalError(c, "failed to leave workspace: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, gin.H{"left": true})
}

// ListAgents handles GET /api/v1/workspaces/:id/agents.
func (h *WorkspaceHandler) ListAgents(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	agents, err := h.service.ListAgents(c.Request.Context(), workspaceID)
	if err != nil {
		models.InternalError(c, "failed to list agents: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, agents)
}

// Heartbeat handles POST /api/v1/agents/heartbeat.
func (h *WorkspaceHandler) Heartbeat(c *gin.Context) {
	agentID, err := getAgentID(c)
	if err != nil {
		models.UnauthorizedError(c, "invalid agent identity")
		return
	}

	var input service.HeartbeatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.Status != "" && !models.IsValidAgentStatus(input.Status) {
		models.BadRequestError(c, "invalid status: must be one of online, offline, busy")
		return
	}

	agent, err := h.service.Heartbeat(c.Request.Context(), agentID, input)
	if err != nil {
		models.InternalError(c, "failed to update heartbeat: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, agent)
}

// getAgentID extracts the agent ID from the Gin context (set by auth middleware).
// The auth middleware stores agent_id as a uuid.UUID value via c.Set().
func getAgentID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("agent_id")
	if !exists {
		return uuid.Nil, ErrMissingAgentID
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrMissingAgentID
	}
	if id == uuid.Nil {
		return uuid.Nil, ErrMissingAgentID
	}
	return id, nil
}

// parseUUIDParam parses a UUID from a URL path parameter.
func parseUUIDParam(c *gin.Context, param string) (uuid.UUID, error) {
	return uuid.Parse(c.Param(param))
}
