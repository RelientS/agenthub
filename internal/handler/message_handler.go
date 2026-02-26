package handler

import (
	"net/http"
	"strconv"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/service"
	"github.com/gin-gonic/gin"
)

// MessageHandler handles message-related HTTP requests.
type MessageHandler struct {
	service *service.MessagingService
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(s *service.MessagingService) *MessageHandler {
	return &MessageHandler{service: s}
}

// RegisterRoutes registers message routes on the given router group.
// All routes are nested under /workspaces/:id and expect auth middleware
// to be applied at a higher level.
func (h *MessageHandler) RegisterRoutes(rg *gin.RouterGroup) {
	messages := rg.Group("/workspaces/:id/messages")
	{
		messages.POST("", h.SendMessage)
		messages.GET("", h.ListMessages)
		messages.GET("/unread", h.GetUnread)
		messages.POST("/:msg_id/read", h.MarkAsRead)
	}

	threads := rg.Group("/workspaces/:id/threads")
	{
		threads.GET("/:thread_id", h.GetThread)
	}
}

// SendMessage handles POST /api/v1/workspaces/:id/messages.
func (h *MessageHandler) SendMessage(c *gin.Context) {
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

	var input service.SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		models.BadRequestError(c, "invalid request body: "+err.Error())
		return
	}

	if input.MessageType == "" {
		models.BadRequestError(c, "message_type is required")
		return
	}
	if !models.IsValidMessageType(input.MessageType) {
		models.BadRequestError(c, "invalid message_type")
		return
	}
	if input.Payload == nil {
		models.BadRequestError(c, "payload is required")
		return
	}

	message, err := h.service.SendMessage(c.Request.Context(), workspaceID, input, agentID)
	if err != nil {
		models.InternalError(c, "failed to send message: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusCreated, message)
}

// ListMessages handles GET /api/v1/workspaces/:id/messages.
// Supports query parameters: limit, offset.
func (h *MessageHandler) ListMessages(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	filter := service.MessageListFilter{
		Limit:  50, // default
		Offset: 0,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			models.BadRequestError(c, "invalid limit: must be a positive integer")
			return
		}
		if limit > 200 {
			limit = 200
		}
		filter.Limit = limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			models.BadRequestError(c, "invalid offset: must be a non-negative integer")
			return
		}
		filter.Offset = offset
	}

	messages, err := h.service.ListMessages(c.Request.Context(), workspaceID, filter)
	if err != nil {
		models.InternalError(c, "failed to list messages: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, messages)
}

// GetUnread handles GET /api/v1/workspaces/:id/messages/unread.
func (h *MessageHandler) GetUnread(c *gin.Context) {
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

	messages, err := h.service.GetUnread(c.Request.Context(), workspaceID, agentID)
	if err != nil {
		models.InternalError(c, "failed to get unread messages: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, messages)
}

// MarkAsRead handles POST /api/v1/workspaces/:id/messages/:msg_id/read.
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
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

	messageID, err := parseUUIDParam(c, "msg_id")
	if err != nil {
		models.BadRequestError(c, "invalid message ID")
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), workspaceID, messageID, agentID); err != nil {
		models.InternalError(c, "failed to mark message as read: "+err.Error())
		return
	}

	models.SuccessResponse(c, http.StatusOK, gin.H{"marked": true})
}

// GetThread handles GET /api/v1/workspaces/:id/threads/:thread_id.
func (h *MessageHandler) GetThread(c *gin.Context) {
	workspaceID, err := parseUUIDParam(c, "id")
	if err != nil {
		models.BadRequestError(c, "invalid workspace ID")
		return
	}

	threadID, err := parseUUIDParam(c, "thread_id")
	if err != nil {
		models.BadRequestError(c, "invalid thread ID")
		return
	}

	messages, err := h.service.GetThread(c.Request.Context(), workspaceID, threadID)
	if err != nil {
		models.NotFoundError(c, "thread not found")
		return
	}

	models.SuccessResponse(c, http.StatusOK, messages)
}
