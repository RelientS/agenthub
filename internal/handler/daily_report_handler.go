package handler

import (
	"net/http"
	"strconv"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/service"
	"github.com/gin-gonic/gin"
)

// DailyReportHandler handles daily report HTTP requests.
type DailyReportHandler struct {
	service *service.DailyReportService
}

// NewDailyReportHandler creates a new DailyReportHandler.
func NewDailyReportHandler(s *service.DailyReportService) *DailyReportHandler {
	return &DailyReportHandler{service: s}
}

// RegisterRoutes registers daily report routes on the given router group.
func (h *DailyReportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	reports := rg.Group("/workspaces/:id/reports")
	{
		reports.POST("", h.CreateReport)
		reports.GET("", h.ListReports)
		reports.GET("/:report_id", h.GetReport)
		reports.POST("/generate", h.GenerateSummary)
	}
}

// CreateReport handles POST /api/v1/workspaces/:id/reports.
func (h *DailyReportHandler) CreateReport(c *gin.Context) {
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

	var input service.CreateDailyReportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.Summary == "" {
		models.BadRequestError(c, "summary is required")
		return
	}

	report, err := h.service.CreateReport(c.Request.Context(), workspaceID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to create report: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusCreated, report)
}

// ListReports handles GET /api/v1/workspaces/:id/reports.
func (h *DailyReportHandler) ListReports(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	limit := 30
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	reports, err := h.service.ListReports(c.Request.Context(), workspaceID, limit, offset)
	if err != nil {
		models.InternalError(c, "failed to list reports: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, reports)
}

// GetReport handles GET /api/v1/workspaces/:id/reports/:report_id.
func (h *DailyReportHandler) GetReport(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	reportID, err := parseUUIDParam(c, "report_id")
	if err != nil {
		models.BadRequestError(c, "invalid report ID")
		return
	}

	report, err := h.service.GetReport(c.Request.Context(), workspaceID, reportID)
	if err != nil {
		models.NotFoundError(c, "report not found")
		return
	}

	models.SuccessResponse(c, http.StatusOK, report)
}

// GenerateSummary handles POST /api/v1/workspaces/:id/reports/generate.
func (h *DailyReportHandler) GenerateSummary(c *gin.Context) {
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

	report, err := h.service.GenerateSummary(c.Request.Context(), workspaceID, agentID)
	if err != nil {
		models.InternalError(c, "failed to generate summary: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusCreated, report)
}
