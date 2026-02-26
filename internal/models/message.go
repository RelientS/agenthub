package models

import (
	"time"

	"github.com/google/uuid"
)

// Message type constants defining the kinds of inter-agent messages.
const (
	MsgTypeRequestSchema   = "request_schema"
	MsgTypeProvideSchema   = "provide_schema"
	MsgTypeRequestEndpoint = "request_endpoint"
	MsgTypeProvideEndpoint = "provide_endpoint"
	MsgTypeReportBlocker   = "report_blocker"
	MsgTypeResolveBlocker  = "resolve_blocker"
	MsgTypeRequestReview   = "request_review"
	MsgTypeProvideReview   = "provide_review"
	MsgTypeStatusUpdate    = "status_update"
	MsgTypeQuestion        = "question"
	MsgTypeAnswer          = "answer"
	MsgTypeNotification    = "notification"
)

// ValidMessageTypes contains all valid message types for validation.
var ValidMessageTypes = []string{
	MsgTypeRequestSchema,
	MsgTypeProvideSchema,
	MsgTypeRequestEndpoint,
	MsgTypeProvideEndpoint,
	MsgTypeReportBlocker,
	MsgTypeResolveBlocker,
	MsgTypeRequestReview,
	MsgTypeProvideReview,
	MsgTypeStatusUpdate,
	MsgTypeQuestion,
	MsgTypeAnswer,
	MsgTypeNotification,
}

// Message represents a communication between agents in a workspace.
type Message struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	WorkspaceID   uuid.UUID              `json:"workspace_id" db:"workspace_id"`
	FromAgentID   uuid.UUID              `json:"from_agent_id" db:"from_agent_id"`
	ToAgentID     *uuid.UUID             `json:"to_agent_id,omitempty" db:"to_agent_id"`
	ThreadID      *uuid.UUID             `json:"thread_id,omitempty" db:"thread_id"`
	MessageType   string                 `json:"message_type" db:"message_type"`
	Payload       map[string]interface{} `json:"payload" db:"payload"`
	RefTaskID     *uuid.UUID             `json:"ref_task_id,omitempty" db:"ref_task_id"`
	RefArtifactID *uuid.UUID             `json:"ref_artifact_id,omitempty" db:"ref_artifact_id"`
	IsRead        bool                   `json:"is_read" db:"is_read"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// NewMessage creates a new Message with default values.
func NewMessage(workspaceID, fromAgentID uuid.UUID, messageType string, payload map[string]interface{}) *Message {
	return &Message{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		FromAgentID: fromAgentID,
		MessageType: messageType,
		Payload:     payload,
		IsRead:      false,
		CreatedAt:   time.Now().UTC(),
	}
}

// IsBroadcast returns true if the message has no specific recipient (broadcast to workspace).
func (m *Message) IsBroadcast() bool {
	return m.ToAgentID == nil
}

// IsValidMessageType checks whether the given type is a valid message type.
func IsValidMessageType(msgType string) bool {
	for _, t := range ValidMessageTypes {
		if t == msgType {
			return true
		}
	}
	return false
}
