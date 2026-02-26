package models

import (
	"time"

	"github.com/google/uuid"
)

// Context type constants.
const (
	ContextTypePRD         = "prd"
	ContextTypeDesignDoc   = "design_doc"
	ContextTypeAPIContract = "api_contract"
	ContextTypeArchitect   = "architecture"
	ContextTypeSharedTypes = "shared_types"
	ContextTypeEnvConfig   = "env_config"
	ContextTypeConvention  = "convention"
)

// ValidContextTypes contains all valid context types for validation.
var ValidContextTypes = []string{
	ContextTypePRD,
	ContextTypeDesignDoc,
	ContextTypeAPIContract,
	ContextTypeArchitect,
	ContextTypeSharedTypes,
	ContextTypeEnvConfig,
	ContextTypeConvention,
}

// Context represents a piece of shared context within a workspace.
type Context struct {
	ID          uuid.UUID `json:"id" db:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id" db:"workspace_id"`
	ContextType string    `json:"context_type" db:"context_type"`
	Title       string    `json:"title" db:"title"`
	Content     string    `json:"content" db:"content"`
	ContentHash string    `json:"content_hash" db:"content_hash"`
	Version     int       `json:"version" db:"version"`
	UpdatedBy   uuid.UUID `json:"updated_by" db:"updated_by"`
	Tags        []string  `json:"tags,omitempty" db:"tags"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// NewContext creates a new Context with default values.
func NewContext(workspaceID uuid.UUID, contextType, title, content, contentHash string, updatedBy uuid.UUID) *Context {
	now := time.Now().UTC()
	return &Context{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		ContextType: contextType,
		Title:       title,
		Content:     content,
		ContentHash: contentHash,
		Version:     1,
		UpdatedBy:   updatedBy,
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsValidContextType checks whether the given type is a valid context type.
func IsValidContextType(ctxType string) bool {
	for _, t := range ValidContextTypes {
		if t == ctxType {
			return true
		}
	}
	return false
}

// SyncLogAction constants.
const (
	SyncActionCreate = "create"
	SyncActionUpdate = "update"
	SyncActionDelete = "delete"
)

// SyncEntity type constants identify the entity being tracked in the sync log.
const (
	SyncEntityTask     = "task"
	SyncEntityMessage  = "message"
	SyncEntityArtifact = "artifact"
	SyncEntityContext  = "context"
	SyncEntityAgent    = "agent"
)

// SyncLogEntry represents a record of a synchronization event for change tracking.
type SyncLogEntry struct {
	ID          int64     `json:"id" db:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id" db:"workspace_id"`
	EntityType  string    `json:"entity_type" db:"entity_type"`
	EntityID    uuid.UUID `json:"entity_id" db:"entity_id"`
	Action      string    `json:"action" db:"action"`
	AgentID     uuid.UUID `json:"agent_id" db:"agent_id"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	PayloadHash string    `json:"payload_hash" db:"payload_hash"`
}

// NewSyncLogEntry creates a new SyncLogEntry.
func NewSyncLogEntry(workspaceID uuid.UUID, entityType string, entityID, agentID uuid.UUID, action, payloadHash string) *SyncLogEntry {
	return &SyncLogEntry{
		WorkspaceID: workspaceID,
		EntityType:  entityType,
		EntityID:    entityID,
		Action:      action,
		AgentID:     agentID,
		Timestamp:   time.Now().UTC(),
		PayloadHash: payloadHash,
	}
}
