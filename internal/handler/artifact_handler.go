package handler

import (
	"net/http"
	"strings"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ArtifactHandler handles artifact-related HTTP requests.
type ArtifactHandler struct {
	service *service.ArtifactService
}

// NewArtifactHandler creates a new ArtifactHandler.
func NewArtifactHandler(s *service.ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{service: s}
}

// RegisterRoutes registers artifact routes on the given router group.
// All routes are nested under /workspaces/:id/artifacts and expect auth
// middleware to be applied at a higher level.
func (h *ArtifactHandler) RegisterRoutes(rg *gin.RouterGroup) {
	artifacts := rg.Group("/workspaces/:id/artifacts")
	{
		artifacts.POST("", h.CreateArtifact)
		artifacts.GET("", h.ListArtifacts)
		artifacts.GET("/search", h.SearchArtifacts)
		artifacts.GET("/:art_id", h.GetArtifact)
		artifacts.GET("/:art_id/history", h.GetHistory)
	}
}

// CreateArtifact handles POST /api/v1/workspaces/:id/artifacts.
func (h *ArtifactHandler) CreateArtifact(c *gin.Context) {
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

	var input service.CreateArtifactInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.Name == "" {
		models.BadRequestError(c, "name is required")
		return
	}
	if input.ArtifactType == "" {
		models.BadRequestError(c, "artifact_type is required")
		return
	}
	if input.Content == "" {
		models.BadRequestError(c, "content is required")
		return
	}

	artifact, err := h.service.CreateArtifact(c.Request.Context(), workspaceID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to create artifact: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusCreated, artifact)
}

// ListArtifacts handles GET /api/v1/workspaces/:id/artifacts.
// Supports query parameters: type, tags, language.
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	filter := service.ArtifactListFilter{
		ArtifactType: c.Query("type"),
		Language:     c.Query("language"),
	}

	if tagsStr := c.Query("tags"); tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")
	}

	artifacts, err := h.service.ListArtifacts(c.Request.Context(), workspaceID, filter)
	if err != nil {
		models.InternalError(c, "failed to list artifacts: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, artifacts)
}

// GetArtifact handles GET /api/v1/workspaces/:id/artifacts/:art_id.
func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	artifactID, err := parseUUIDParam(c, "art_id")
	if err != nil {
		models.BadRequestError(c, "invalid artifact ID")
		return
	}

	artifact, err := h.service.GetArtifact(c.Request.Context(), workspaceID, artifactID)
	if err != nil {
		models.NotFoundError(c, "artifact not found")
		return
	}

	models.SuccessResponse(c, http.StatusOK, artifact)
}

// GetHistory handles GET /api/v1/workspaces/:id/artifacts/:art_id/history.
func (h *ArtifactHandler) GetHistory(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	artifactID, err := parseUUIDParam(c, "art_id")
	if err != nil {
		models.BadRequestError(c, "invalid artifact ID")
		return
	}

	// First fetch the artifact to get its name, then retrieve history by name.
	artifact, err := h.service.GetArtifact(c.Request.Context(), workspaceID, artifactID)
	if err != nil {
		models.NotFoundError(c, "artifact not found")
		return
	}

	history, err := h.service.GetHistory(c.Request.Context(), workspaceID, artifact.Name)
	if err != nil {
		models.InternalError(c, "failed to get artifact history: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, history)
}

// SearchArtifacts handles GET /api/v1/workspaces/:id/artifacts/search.
// Requires query parameter: q.
func (h *ArtifactHandler) SearchArtifacts(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	query := c.Query("q")
	if query == "" {
		models.BadRequestError(c, "query parameter 'q' is required")
		return
	}

	artifacts, err := h.service.SearchArtifacts(c.Request.Context(), workspaceID, query)
	if err != nil {
		models.InternalError(c, "failed to search artifacts: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, artifacts)
}
