package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	service *service.TaskService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(s *service.TaskService) *TaskHandler {
	return &TaskHandler{service: s}
}

// RegisterRoutes registers task routes on the given router group.
// All routes are nested under /workspaces/:id/tasks and expect auth middleware
// to be applied at a higher level.
func (h *TaskHandler) RegisterRoutes(rg *gin.RouterGroup) {
	tasks := rg.Group("/workspaces/:id/tasks")
	{
		tasks.POST("", h.CreateTask)
		tasks.GET("", h.ListTasks)
		tasks.GET("/board", h.GetBoard)
		tasks.GET("/:task_id", h.GetTask)
		tasks.PUT("/:task_id", h.UpdateTask)
		tasks.POST("/:task_id/claim", h.ClaimTask)
		tasks.POST("/:task_id/complete", h.CompleteTask)
		tasks.POST("/:task_id/block", h.BlockTask)
	}
}

// CreateTask handles POST /api/v1/workspaces/:id/tasks.
func (h *TaskHandler) CreateTask(c *gin.Context) {
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

	var input service.CreateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.Title == "" {
		models.BadRequestError(c, "title is required")
		return
	}
	if input.Priority != 0 && !models.IsValidPriority(input.Priority) {
		models.BadRequestError(c, "priority must be between 1 and 5")
		return
	}
	if input.Priority == 0 {
		input.Priority = 3 // default priority
	}

	task, err := h.service.CreateTask(c.Request.Context(), workspaceID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to create task: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusCreated, task)
}

// ListTasks handles GET /api/v1/workspaces/:id/tasks.
// Supports query parameters: status, assigned_to, priority, tags.
func (h *TaskHandler) ListTasks(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	filter := service.TaskListFilter{
		Status: c.Query("status"),
	}

	if assignedToStr := c.Query("assigned_to"); assignedToStr != "" {
		parsed, err := uuid.Parse(assignedToStr)
		if err != nil {
			models.BadRequestError(c, "invalid assigned_to: must be a valid UUID")
			return
		}
		filter.AssignedTo = &parsed
	}

	if priorityStr := c.Query("priority"); priorityStr != "" {
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			models.BadRequestError(c, "invalid priority: must be an integer")
			return
		}
		if !models.IsValidPriority(priority) {
			models.BadRequestError(c, "priority must be between 1 and 5")
			return
		}
		filter.Priority = &priority
	}

	if tagsStr := c.Query("tags"); tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")
	}

	if filter.Status != "" && !models.IsValidTaskStatus(filter.Status) {
		models.BadRequestError(c, "invalid status filter")
		return
	}

	tasks, err := h.service.ListTasks(c.Request.Context(), workspaceID, filter)
	if err != nil {
		models.InternalError(c, "failed to list tasks: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, tasks)
}

// GetTask handles GET /api/v1/workspaces/:id/tasks/:task_id.
func (h *TaskHandler) GetTask(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	taskID, err := parseUUIDParam(c, "task_id")
	if err != nil {
		models.BadRequestError(c, "invalid task ID")
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		models.NotFoundError(c, "task not found")
		return
	}

	models.SuccessResponse(c, http.StatusOK, task)
}

// UpdateTask handles PUT /api/v1/workspaces/:id/tasks/:task_id.
func (h *TaskHandler) UpdateTask(c *gin.Context) {
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

	taskID, err := parseUUIDParam(c, "task_id")
	if err != nil {
		models.BadRequestError(c, "invalid task ID")
		return
	}

	var input service.UpdateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.Status != nil && !models.IsValidTaskStatus(*input.Status) {
		models.BadRequestError(c, "invalid task status")
		return
	}
	if input.Priority != nil && !models.IsValidPriority(*input.Priority) {
		models.BadRequestError(c, "priority must be between 1 and 5")
		return
	}

	task, err := h.service.UpdateTask(c.Request.Context(), workspaceID, taskID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to update task: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, task)
}

// ClaimTask handles POST /api/v1/workspaces/:id/tasks/:task_id/claim.
func (h *TaskHandler) ClaimTask(c *gin.Context) {
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

	taskID, err := parseUUIDParam(c, "task_id")
	if err != nil {
		models.BadRequestError(c, "invalid task ID")
		return
	}

	task, err := h.service.ClaimTask(c.Request.Context(), workspaceID, taskID, agentID)
	if err != nil {
		models.ConflictError(c, "failed to claim task: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, task)
}

// CompleteTask handles POST /api/v1/workspaces/:id/tasks/:task_id/complete.
func (h *TaskHandler) CompleteTask(c *gin.Context) {
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

	taskID, err := parseUUIDParam(c, "task_id")
	if err != nil {
		models.BadRequestError(c, "invalid task ID")
		return
	}

	var input service.CompleteTaskInput
	// Body is optional for complete
	_ = c.ShouldBindJSON(&input)

	task, err := h.service.CompleteTask(c.Request.Context(), workspaceID, taskID, agentID, input)
	if err != nil {
		models.InternalError(c, "failed to complete task: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, task)
}

// blockTaskRequest is the request body for blocking a task.
type blockTaskRequest struct {
	Reason string `json:"reason"`
}

// BlockTask handles POST /api/v1/workspaces/:id/tasks/:task_id/block.
func (h *TaskHandler) BlockTask(c *gin.Context) {
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

	taskID, err := parseUUIDParam(c, "task_id")
	if err != nil {
		models.BadRequestError(c, "invalid task ID")
		return
	}

	var req blockTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if req.Reason == "" {
		models.BadRequestError(c, "reason is required")
		return
	}

	task, err := h.service.BlockTask(c.Request.Context(), workspaceID, taskID, agentID, req.Reason)
	if err != nil {
		models.InternalError(c, "failed to block task: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, task)
}

// GetBoard handles GET /api/v1/workspaces/:id/tasks/board.
func (h *TaskHandler) GetBoard(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	board, err := h.service.GetBoard(c.Request.Context(), workspaceID)
	if err != nil {
		models.InternalError(c, "failed to get task board: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, board)
}
