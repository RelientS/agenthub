package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/pkg/events"
	"github.com/agenthub/server/internal/pkg/ws"
	"github.com/agenthub/server/internal/repository"
	"github.com/google/uuid"
)

// ---------- request / response types ----------

// SendMessageInput holds the input for sending a message.
type SendMessageInput struct {
	ToAgentID     *uuid.UUID             `json:"to_agent_id,omitempty"`
	ThreadID      *uuid.UUID             `json:"thread_id,omitempty"`
	MessageType   string                 `json:"message_type"`
	Payload       map[string]interface{} `json:"payload"`
	RefTaskID     *uuid.UUID             `json:"ref_task_id,omitempty"`
	RefArtifactID *uuid.UUID             `json:"ref_artifact_id,omitempty"`
}

// MessageListFilter holds filter parameters for listing messages.
type MessageListFilter struct {
	FromAgentID *uuid.UUID
	ToAgentID   *uuid.UUID
	MessageType string
	ThreadID    *uuid.UUID
	Limit       int
	Offset      int
}

// ---------- service ----------

// MessagingService handles messaging business logic.
type MessagingService struct {
	messageRepo *repository.MessageRepository
	syncRepo    *repository.SyncRepository
	eventBus    *events.Bus
	wsHub       *ws.Hub
}

// NewMessagingService creates a new MessagingService.
func NewMessagingService(
	mr *repository.MessageRepository,
	sr *repository.SyncRepository,
	eb *events.Bus,
	wh *ws.Hub,
) *MessagingService {
	return &MessagingService{
		messageRepo: mr,
		syncRepo:    sr,
		eventBus:    eb,
		wsHub:       wh,
	}
}

// SendMessage creates a message, logs the sync entry, publishes a domain event,
// and sends a WebSocket notification to the recipient (or broadcasts if no
// specific recipient is set).
func (s *MessagingService) SendMessage(ctx context.Context, workspaceID uuid.UUID, input SendMessageInput, fromAgentID uuid.UUID) (*models.Message, error) {
	if !models.IsValidMessageType(input.MessageType) {
		return nil, fmt.Errorf("invalid message type: %s", input.MessageType)
	}
	if input.Payload == nil {
		return nil, fmt.Errorf("message payload is required")
	}

	msg := models.NewMessage(workspaceID, fromAgentID, input.MessageType, input.Payload)
	msg.ToAgentID = input.ToAgentID
	msg.ThreadID = input.ThreadID
	msg.RefTaskID = input.RefTaskID
	msg.RefArtifactID = input.RefArtifactID

	if err := s.messageRepo.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}

	// Log sync entry.
	s.logChange(ctx, workspaceID, models.SyncEntityMessage, msg.ID, fromAgentID, models.SyncActionCreate, msg)

	// Publish domain event.
	s.eventBus.Publish(events.NewEvent(events.EventMessageNew, workspaceID.String(), map[string]interface{}{
		"message_id":   msg.ID,
		"from_agent":   msg.FromAgentID,
		"to_agent":     msg.ToAgentID,
		"message_type": msg.MessageType,
	}))

	// Send WebSocket notification.
	wsPayload, err := json.Marshal(map[string]interface{}{
		"type":         "message",
		"message_id":   msg.ID,
		"workspace_id": msg.WorkspaceID,
		"from_agent":   msg.FromAgentID,
		"message_type": msg.MessageType,
		"payload":      msg.Payload,
	})
	if err == nil {
		if msg.ToAgentID != nil {
			s.wsHub.SendToAgent(msg.WorkspaceID, *msg.ToAgentID, wsPayload)
		} else {
			// Broadcast to all agents except the sender.
			s.wsHub.BroadcastToWorkspaceExcept(msg.WorkspaceID, msg.FromAgentID, wsPayload)
		}
	}

	return msg, nil
}

// ListMessages returns messages matching the given filter criteria.
func (s *MessagingService) ListMessages(ctx context.Context, workspaceID uuid.UUID, filter MessageListFilter) ([]models.Message, error) {
	repoFilter := repository.MessageFilters{
		FromAgentID: filter.FromAgentID,
		ToAgentID:   filter.ToAgentID,
		MessageType: filter.MessageType,
		ThreadID:    filter.ThreadID,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	}

	messages, _, err := s.messageRepo.ListByWorkspace(ctx, workspaceID, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}
	return messages, nil
}

// GetUnread returns all unread messages for an agent in a workspace.
func (s *MessagingService) GetUnread(ctx context.Context, workspaceID uuid.UUID, agentID uuid.UUID) ([]models.Message, error) {
	messages, err := s.messageRepo.GetUnread(ctx, workspaceID, agentID)
	if err != nil {
		return nil, fmt.Errorf("getting unread messages: %w", err)
	}
	return messages, nil
}

// MarkAsRead marks a specific message as read.
func (s *MessagingService) MarkAsRead(ctx context.Context, workspaceID uuid.UUID, messageID uuid.UUID, agentID uuid.UUID) error {
	// Validate the message exists and belongs to the workspace.
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("getting message: %w", err)
	}
	if msg.WorkspaceID != workspaceID {
		return fmt.Errorf("message does not belong to this workspace")
	}

	if err := s.messageRepo.MarkAsRead(ctx, messageID); err != nil {
		return fmt.Errorf("marking message as read: %w", err)
	}

	return nil
}

// GetThread returns all messages belonging to a given thread.
func (s *MessagingService) GetThread(ctx context.Context, workspaceID uuid.UUID, threadID uuid.UUID) ([]models.Message, error) {
	messages, err := s.messageRepo.GetThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("getting thread: %w", err)
	}

	// Filter to only messages belonging to the requested workspace.
	var filtered []models.Message
	for _, m := range messages {
		if m.WorkspaceID == workspaceID {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// BroadcastToWorkspace sends a notification-type message from the given agent
// to all other agents in the workspace.
func (s *MessagingService) BroadcastToWorkspace(ctx context.Context, workspaceID uuid.UUID, fromAgentID uuid.UUID, payload map[string]interface{}) (*models.Message, error) {
	input := SendMessageInput{
		MessageType: models.MsgTypeNotification,
		Payload:     payload,
	}
	return s.SendMessage(ctx, workspaceID, input, fromAgentID)
}

// ---------- helpers ----------

// logChange marshals the payload and writes a sync log entry.
func (s *MessagingService) logChange(ctx context.Context, workspaceID uuid.UUID, entityType string, entityID, agentID uuid.UUID, action string, payload interface{}) {
	data, _ := json.Marshal(payload)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	entry := models.NewSyncLogEntry(workspaceID, entityType, entityID, agentID, action, hash)
	_ = s.syncRepo.LogChange(ctx, entry)
}
