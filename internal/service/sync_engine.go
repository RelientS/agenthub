package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/pkg/conflict"
	"github.com/agenthub/server/internal/pkg/events"
	"github.com/agenthub/server/internal/pkg/ws"
	"github.com/agenthub/server/internal/repository"
	"github.com/google/uuid"
)

// ---------- request / response types ----------

// PushChangesInput holds the input for pushing sync changes.
type PushChangesInput struct {
	Changes []SyncChange `json:"changes"`
}

// SyncChange represents a single change to push.
type SyncChange struct {
	EntityType  string    `json:"entity_type"`
	EntityID    uuid.UUID `json:"entity_id"`
	Action      string    `json:"action"`
	PayloadHash string    `json:"payload_hash"`
}

// PullChangesInput holds the input for pulling sync changes.
type PullChangesInput struct {
	LastSyncID  int64    `json:"last_sync_id"`
	EntityTypes []string `json:"entity_types,omitempty"`
	Limit       int      `json:"limit,omitempty"`
}

// PullChangesResponse holds the response for a pull operation.
type PullChangesResponse struct {
	Changes    []models.SyncLogEntry `json:"changes"`
	LastSyncID int64                 `json:"last_sync_id"`
	HasMore    bool                  `json:"has_more"`
}

// SyncStatus represents the current sync status for a workspace.
type SyncStatus struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	LastSyncID  int64     `json:"last_sync_id"`
	PendingOps  int       `json:"pending_ops"`
}

// ---------- service ----------

// SyncEngine handles change synchronization between agents.
type SyncEngine struct {
	syncRepo *repository.SyncRepository
	wsHub    *ws.Hub
	eventBus *events.Bus
	resolver *conflict.Resolver
}

// NewSyncEngine creates a new SyncEngine.
func NewSyncEngine(
	sr *repository.SyncRepository,
	wh *ws.Hub,
	eb *events.Bus,
	resolver *conflict.Resolver,
) *SyncEngine {
	return &SyncEngine{
		syncRepo: sr,
		wsHub:    wh,
		eventBus: eb,
		resolver: resolver,
	}
}

// PushChanges receives a batch of changes from an agent, validates them,
// writes each to the sync log, and broadcasts notifications to other agents
// in the workspace via WebSocket.
func (s *SyncEngine) PushChanges(ctx context.Context, workspaceID uuid.UUID, input PushChangesInput, agentID uuid.UUID) ([]models.SyncLogEntry, error) {
	if len(input.Changes) == 0 {
		return nil, fmt.Errorf("no changes to push")
	}

	var entries []models.SyncLogEntry

	for _, change := range input.Changes {
		if !isValidEntityType(change.EntityType) {
			return nil, fmt.Errorf("invalid entity type: %s", change.EntityType)
		}
		if !isValidSyncAction(change.Action) {
			return nil, fmt.Errorf("invalid sync action: %s", change.Action)
		}

		entry := models.NewSyncLogEntry(workspaceID, change.EntityType, change.EntityID, agentID, change.Action, change.PayloadHash)

		if err := s.syncRepo.LogChange(ctx, entry); err != nil {
			return nil, fmt.Errorf("logging change for %s/%s: %w", change.EntityType, change.EntityID, err)
		}

		entries = append(entries, *entry)
	}

	// Broadcast the changes to other agents via WebSocket.
	s.broadcastChange(workspaceID, agentID, entries)

	// Publish a sync push event.
	s.eventBus.Publish(events.NewEvent("sync.push", workspaceID.String(), map[string]interface{}{
		"agent_id":     agentID,
		"change_count": len(entries),
	}))

	return entries, nil
}

// PullChanges retrieves all sync log entries after the caller's last known
// sync ID. It returns at most `limit` entries and indicates whether more
// entries remain via the HasMore field.
func (s *SyncEngine) PullChanges(ctx context.Context, workspaceID uuid.UUID, input PullChangesInput) (*PullChangesResponse, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	// Request one extra entry to determine if there are more results.
	changes, err := s.syncRepo.GetChangesSince(ctx, workspaceID, input.LastSyncID, input.EntityTypes, limit+1)
	if err != nil {
		return nil, fmt.Errorf("pulling changes: %w", err)
	}

	hasMore := len(changes) > limit
	if hasMore {
		changes = changes[:limit]
	}

	var lastID int64
	if len(changes) > 0 {
		lastID = changes[len(changes)-1].ID
	} else {
		lastID = input.LastSyncID
	}

	s.eventBus.Publish(events.NewEvent("sync.pull", workspaceID.String(), map[string]interface{}{
		"since_id":     input.LastSyncID,
		"result_count": len(changes),
	}))

	return &PullChangesResponse{
		Changes:    changes,
		LastSyncID: lastID,
		HasMore:    hasMore,
	}, nil
}

// GetSyncStatus returns the latest sync ID for a workspace. Agents can use
// this to determine how far behind they are.
func (s *SyncEngine) GetSyncStatus(ctx context.Context, workspaceID uuid.UUID) (*SyncStatus, error) {
	latestID, err := s.syncRepo.GetLatestSyncID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("getting latest sync id: %w", err)
	}

	return &SyncStatus{
		WorkspaceID: workspaceID,
		LastSyncID:  latestID,
	}, nil
}

// GetSyncStatusSince returns the sync status relative to a given sync ID,
// including the count of pending operations.
func (s *SyncEngine) GetSyncStatusSince(ctx context.Context, workspaceID uuid.UUID, sinceID int64) (*SyncStatus, error) {
	latestID, err := s.syncRepo.GetLatestSyncID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("getting latest sync id: %w", err)
	}

	changes, pullErr := s.syncRepo.GetChangesSince(ctx, workspaceID, sinceID, nil, 0)
	pendingCount := 0
	if pullErr == nil {
		pendingCount = len(changes)
	}

	return &SyncStatus{
		WorkspaceID: workspaceID,
		LastSyncID:  latestID,
		PendingOps:  pendingCount,
	}, nil
}

// LogChange is a convenience helper that writes a single sync log entry with
// automatic payload hashing.
func (s *SyncEngine) LogChange(ctx context.Context, workspaceID uuid.UUID, entityType string, entityID, agentID uuid.UUID, action string, payload interface{}) error {
	data, _ := json.Marshal(payload)
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(data))
	entry := models.NewSyncLogEntry(workspaceID, entityType, entityID, agentID, action, payloadHash)

	if err := s.syncRepo.LogChange(ctx, entry); err != nil {
		return fmt.Errorf("logging sync change: %w", err)
	}
	return nil
}

// BroadcastChange sends a sync notification to all agents in the workspace
// except the originating agent.
func (s *SyncEngine) BroadcastChange(workspaceID, excludeAgentID uuid.UUID, entries []models.SyncLogEntry) {
	s.broadcastChange(workspaceID, excludeAgentID, entries)
}

// ---------- helpers ----------

// broadcastChange serializes entries and sends them via WebSocket.
func (s *SyncEngine) broadcastChange(workspaceID, excludeAgentID uuid.UUID, entries []models.SyncLogEntry) {
	payload, err := json.Marshal(map[string]interface{}{
		"type":         "sync",
		"workspace_id": workspaceID,
		"changes":      entries,
	})
	if err != nil {
		return
	}
	s.wsHub.BroadcastToWorkspaceExcept(workspaceID, excludeAgentID, payload)
}

// isValidEntityType checks whether the given entity type is known.
func isValidEntityType(entityType string) bool {
	switch entityType {
	case models.SyncEntityTask,
		models.SyncEntityMessage,
		models.SyncEntityArtifact,
		models.SyncEntityContext,
		models.SyncEntityAgent:
		return true
	default:
		return false
	}
}

// isValidSyncAction checks whether the given sync action is known.
func isValidSyncAction(action string) bool {
	switch action {
	case models.SyncActionCreate,
		models.SyncActionUpdate,
		models.SyncActionDelete:
		return true
	default:
		return false
	}
}
