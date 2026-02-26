package handler

import (
	"net/http"
	"strconv"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/service"
	"github.com/gin-gonic/gin"
)

// SyncHandler handles sync-related HTTP requests.
type SyncHandler struct {
	engine *service.SyncEngine
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(e *service.SyncEngine) *SyncHandler {
	return &SyncHandler{engine: e}
}

// RegisterRoutes registers sync routes on the given router group.
// All routes are nested under /workspaces/:id/sync and expect auth
// middleware to be applied at a higher level.
func (h *SyncHandler) RegisterRoutes(rg *gin.RouterGroup) {
	sync := rg.Group("/workspaces/:id/sync")
	{
		sync.POST("/push", h.PushChanges)
		sync.POST("/pull", h.PullChanges)
		sync.GET("/status", h.GetSyncStatus)
	}
}

// PushChanges handles POST /api/v1/workspaces/:id/sync/push.
func (h *SyncHandler) PushChanges(c *gin.Context) {
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

	var input service.PushChangesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if len(input.Changes) == 0 {
		models.BadRequestError(c, "changes array must not be empty")
		return
	}

	// Validate each change entry.
	for i, change := range input.Changes {
		if change.EntityType == "" {
			models.BadRequestError(c, "changes["+itoa(i)+"]: entity_type is required")
			return
		}
		if change.Action == "" {
			models.BadRequestError(c, "changes["+itoa(i)+"]: operation is required")
			return
		}
		if change.Action != models.SyncActionCreate && change.Action != models.SyncActionUpdate && change.Action != models.SyncActionDelete {
			models.BadRequestError(c, "changes["+itoa(i)+"]: operation must be one of create, update, delete")
			return
		}
	}

	entries, err := h.engine.PushChanges(c.Request.Context(), workspaceID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to push changes: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, entries)
}

// PullChanges handles POST /api/v1/workspaces/:id/sync/pull.
func (h *SyncHandler) PullChanges(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	var input service.PullChangesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.LastSyncID < 0 {
		models.BadRequestError(c, "last_sync_id must be non-negative")
		return
	}

	resp, err := h.engine.PullChanges(c.Request.Context(), workspaceID, input)
	if err != nil {
		models.InternalError(c, "failed to pull changes: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, resp)
}

// GetSyncStatus handles GET /api/v1/workspaces/:id/sync/status.
func (h *SyncHandler) GetSyncStatus(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	status, err := h.engine.GetSyncStatus(c.Request.Context(), workspaceID)
	if err != nil {
		models.InternalError(c, "failed to get sync status: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, status)
}

// itoa is a small helper to convert int to string for error messages.
func itoa(i int) string {
	return strconv.Itoa(i)
}
