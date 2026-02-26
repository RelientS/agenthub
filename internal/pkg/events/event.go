package events

import (
	"time"
)

// Event type constants used across the system.
const (
	EventAgentOnline  = "agent.online"
	EventAgentOffline = "agent.offline"

	EventTaskCreated  = "task.created"
	EventTaskUpdated  = "task.updated"
	EventTaskAssigned = "task.assigned"

	EventMessageNew = "message.new"

	EventArtifactNew     = "artifact.new"
	EventArtifactUpdated = "artifact.updated"

	EventContextUpdated = "context.updated"

	EventConflictDetected = "conflict.detected"
)

// Event represents a domain event that flows through the event bus.
type Event struct {
	Type        string      `json:"type"`
	WorkspaceID string      `json:"workspace_id"`
	Data        interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
}

// NewEvent creates a new Event with the current timestamp.
func NewEvent(eventType, workspaceID string, data interface{}) Event {
	return Event{
		Type:        eventType,
		WorkspaceID: workspaceID,
		Data:        data,
		Timestamp:   time.Now().UTC(),
	}
}
