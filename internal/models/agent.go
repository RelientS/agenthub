package models

import (
	"time"

	"github.com/google/uuid"
)

// Agent role constants.
const (
	AgentRoleFrontend  = "frontend"
	AgentRoleBackend   = "backend"
	AgentRoleFullstack = "fullstack"
	AgentRoleTester    = "tester"
	AgentRoleDevops    = "devops"
)

// Agent status constants.
const (
	AgentStatusOnline  = "online"
	AgentStatusOffline = "offline"
	AgentStatusBusy    = "busy"
)

// ValidAgentRoles contains all valid agent roles for validation.
var ValidAgentRoles = []string{
	AgentRoleFrontend,
	AgentRoleBackend,
	AgentRoleFullstack,
	AgentRoleTester,
	AgentRoleDevops,
}

// ValidAgentStatuses contains all valid agent statuses for validation.
var ValidAgentStatuses = []string{
	AgentStatusOnline,
	AgentStatusOffline,
	AgentStatusBusy,
}

// Agent represents an AI agent participating in a workspace.
type Agent struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	WorkspaceID   uuid.UUID              `json:"workspace_id" db:"workspace_id"`
	Name          string                 `json:"name" db:"name"`
	Role          string                 `json:"role" db:"role"`
	Status        string                 `json:"status" db:"status"`
	Capabilities  []string               `json:"capabilities" db:"capabilities"`
	LastHeartbeat time.Time              `json:"last_heartbeat" db:"last_heartbeat"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// NewAgent creates a new Agent with default values.
func NewAgent(workspaceID uuid.UUID, name, role string, capabilities []string) *Agent {
	now := time.Now().UTC()
	return &Agent{
		ID:            uuid.New(),
		WorkspaceID:   workspaceID,
		Name:          name,
		Role:          role,
		Status:        AgentStatusOnline,
		Capabilities:  capabilities,
		LastHeartbeat: now,
		Metadata:      make(map[string]interface{}),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// IsValidRole checks whether the given role is a valid agent role.
func IsValidRole(role string) bool {
	for _, r := range ValidAgentRoles {
		if r == role {
			return true
		}
	}
	return false
}

// IsValidAgentStatus checks whether the given status is a valid agent status.
func IsValidAgentStatus(status string) bool {
	for _, s := range ValidAgentStatuses {
		if s == status {
			return true
		}
	}
	return false
}
