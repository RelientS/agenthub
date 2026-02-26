package models

import (
	"time"

	"github.com/google/uuid"
)

// WorkspaceStatus constants define the possible states of a workspace.
const (
	WorkspaceStatusActive   = "active"
	WorkspaceStatusArchived = "archived"
	WorkspaceStatusDeleted  = "deleted"
)

// Workspace represents a collaborative workspace where agents work together.
type Workspace struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Description  string                 `json:"description,omitempty" db:"description"`
	OwnerAgentID uuid.UUID              `json:"owner_agent_id" db:"owner_agent_id"`
	InviteCode   string                 `json:"invite_code" db:"invite_code"`
	Status       string                 `json:"status" db:"status"`
	Settings     map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// NewWorkspace creates a new Workspace with default values.
func NewWorkspace(name, description string, ownerAgentID uuid.UUID) *Workspace {
	now := time.Now().UTC()
	return &Workspace{
		ID:           uuid.New(),
		Name:         name,
		Description:  description,
		OwnerAgentID: ownerAgentID,
		InviteCode:   uuid.New().String()[:8],
		Status:       WorkspaceStatusActive,
		Settings:     make(map[string]interface{}),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
