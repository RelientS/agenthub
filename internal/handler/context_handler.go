package handler

import (
	"net/http"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ContextHandler handles shared context HTTP requests.
type ContextHandler struct {
	service *service.ContextService
}

// NewContextHandler creates a new ContextHandler.
func NewContextHandler(s *service.ContextService) *ContextHandler {
	return &ContextHandler{service: s}
}

// RegisterRoutes registers context routes on the given router group.
// All routes are nested under /workspaces/:id/contexts and expect auth
// middleware to be applied at a higher level.
func (h *ContextHandler) RegisterRoutes(rg *gin.RouterGroup) {
	contexts := rg.Group("/workspaces/:id/contexts")
	{
		contexts.POST("", h.CreateContext)
		contexts.GET("", h.ListContexts)
		contexts.GET("/snapshot", h.GetSnapshot)
		contexts.GET("/:ctx_id", h.GetContext)
		contexts.PUT("/:ctx_id", h.UpdateContext)
	}
}

// CreateContext handles POST /api/v1/workspaces/:id/contexts.
func (h *ContextHandler) CreateContext(c *gin.Context) {
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

	var input service.CreateContextInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.Title == "" {
		models.BadRequestError(c, "title is required")
		return
	}
	if input.Content == "" {
		models.BadRequestError(c, "content is required")
		return
	}
	if input.ContextType == "" {
		input.ContextType = models.ContextTypePRD // default type
	}
	if !models.IsValidContextType(input.ContextType) {
		models.BadRequestError(c, "invalid context_type: must be one of prd, design_doc, api_contract, architecture, shared_types, env_config, convention")
		return
	}

	ctx, err := h.service.CreateContext(c.Request.Context(), workspaceID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to create context: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusCreated, ctx)
}

// ListContexts handles GET /api/v1/workspaces/:id/contexts.
func (h *ContextHandler) ListContexts(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	contexts, err := h.service.ListContexts(c.Request.Context(), workspaceID)
	if err != nil {
		models.InternalError(c, "failed to list contexts: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, contexts)
}

// GetContext handles GET /api/v1/workspaces/:id/contexts/:ctx_id.
func (h *ContextHandler) GetContext(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	contextID, err := parseUUIDParam(c, "ctx_id")
	if err != nil {
		models.BadRequestError(c, "invalid context ID")
		return
	}

	ctx, err := h.service.GetContext(c.Request.Context(), workspaceID, contextID)
	if err != nil {
		models.NotFoundError(c, "context not found")
		return
	}

	models.SuccessResponse(c, http.StatusOK, ctx)
}

// UpdateContext handles PUT /api/v1/workspaces/:id/contexts/:ctx_id.
func (h *ContextHandler) UpdateContext(c *gin.Context) {
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

	contextID, err := parseUUIDParam(c, "ctx_id")
	if err != nil {
		models.BadRequestError(c, "invalid context ID")
		return
	}

	var input service.UpdateContextInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	ctx, err := h.service.UpdateContext(c.Request.Context(), workspaceID, contextID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to update context: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, ctx)
}

// GetSnapshot handles GET /api/v1/workspaces/:id/contexts/snapshot.
func (h *ContextHandler) GetSnapshot(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	snapshot, err := h.service.GetSnapshot(c.Request.Context(), workspaceID)
	if err != nil {
		models.InternalError(c, "failed to get context snapshot: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, snapshot)
}
