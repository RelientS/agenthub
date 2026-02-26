package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/pkg/conflict"
	"github.com/agenthub/server/internal/pkg/events"
	"github.com/agenthub/server/internal/repository"
	"github.com/google/uuid"
)

// ---------- request / response types ----------

// CreateContextInput holds the input for creating a context entry.
type CreateContextInput struct {
	ContextType string   `json:"context_type"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateContextInput holds the input for updating a context entry.
type UpdateContextInput struct {
	Title       *string  `json:"title,omitempty"`
	Content     *string  `json:"content,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	BaseVersion int      `json:"base_version"`
}

// ContextSnapshot represents a snapshot of all context entries in a workspace.
type ContextSnapshot struct {
	WorkspaceID uuid.UUID        `json:"workspace_id"`
	Contexts    []models.Context `json:"contexts"`
	GeneratedAt time.Time        `json:"generated_at"`
}

// ---------- service ----------

// ContextService handles shared context business logic.
type ContextService struct {
	contextRepo *repository.ContextRepository
	syncRepo    *repository.SyncRepository
	eventBus    *events.Bus
	resolver    *conflict.Resolver
}

// NewContextService creates a new ContextService.
func NewContextService(
	cr *repository.ContextRepository,
	sr *repository.SyncRepository,
	eb *events.Bus,
	resolver *conflict.Resolver,
) *ContextService {
	return &ContextService{
		contextRepo: cr,
		syncRepo:    sr,
		eventBus:    eb,
		resolver:    resolver,
	}
}

// CreateContext creates a new context entry. It computes a SHA-256 hash of the
// content and sets the initial version to 1.
func (s *ContextService) CreateContext(ctx context.Context, workspaceID uuid.UUID, input CreateContextInput, updatedBy uuid.UUID) (*models.Context, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("context title is required")
	}
	if input.Content == "" {
		return nil, fmt.Errorf("context content is required")
	}
	if !models.IsValidContextType(input.ContextType) {
		return nil, fmt.Errorf("invalid context type: %s", input.ContextType)
	}

	contentHash := computeHash(input.Content)

	entry := models.NewContext(workspaceID, input.ContextType, input.Title, input.Content, contentHash, updatedBy)
	if input.Tags != nil {
		entry.Tags = input.Tags
	}

	if err := s.contextRepo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("creating context: %w", err)
	}

	s.logContextChange(ctx, workspaceID, entry.ID, updatedBy, models.SyncActionCreate, entry)

	s.eventBus.Publish(events.NewEvent(events.EventContextUpdated, workspaceID.String(), map[string]interface{}{
		"context_id": entry.ID,
		"title":      entry.Title,
		"action":     "created",
	}))

	return entry, nil
}

// GetContext retrieves a context entry by ID, validating workspace membership.
func (s *ContextService) GetContext(ctx context.Context, workspaceID uuid.UUID, contextID uuid.UUID) (*models.Context, error) {
	entry, err := s.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, fmt.Errorf("getting context: %w", err)
	}
	if entry.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("context does not belong to this workspace")
	}
	return entry, nil
}

// UpdateContext applies updates to a context entry using optimistic locking.
// The caller must provide a BaseVersion matching the current version; otherwise
// the conflict resolver returns a version mismatch error.
func (s *ContextService) UpdateContext(ctx context.Context, workspaceID uuid.UUID, contextID uuid.UUID, input UpdateContextInput, agentID uuid.UUID) (*models.Context, error) {
	entry, err := s.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, fmt.Errorf("getting context: %w", err)
	}
	if entry.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("context does not belong to this workspace")
	}

	// Build the new content for hash comparison.
	newContent := entry.Content
	if input.Content != nil {
		newContent = *input.Content
	}
	newHash := computeHash(newContent)

	// Use the conflict resolver for version checking (optimistic locking).
	resolution, resolveErr := s.resolver.ResolveContext(
		contextID.String(),
		entry.Version,
		input.BaseVersion,
		entry.ContentHash,
		newHash,
	)
	if resolveErr != nil {
		return nil, fmt.Errorf("context version conflict: %w", resolveErr)
	}

	if input.Title != nil {
		if *input.Title == "" {
			return nil, fmt.Errorf("context title cannot be empty")
		}
		entry.Title = *input.Title
	}
	if input.Content != nil {
		entry.Content = *input.Content
	}
	if input.Tags != nil {
		entry.Tags = input.Tags
	}

	entry.ContentHash = newHash
	entry.Version = resolution.NewVersion
	entry.UpdatedBy = agentID
	entry.UpdatedAt = time.Now().UTC()

	if err := s.contextRepo.Update(ctx, entry); err != nil {
		return nil, fmt.Errorf("updating context: %w", err)
	}

	s.logContextChange(ctx, workspaceID, entry.ID, agentID, models.SyncActionUpdate, entry)

	s.eventBus.Publish(events.NewEvent(events.EventContextUpdated, workspaceID.String(), map[string]interface{}{
		"context_id": entry.ID,
		"title":      entry.Title,
		"version":    entry.Version,
		"action":     "updated",
	}))

	return entry, nil
}

// ListContexts returns all context entries for a workspace.
func (s *ContextService) ListContexts(ctx context.Context, workspaceID uuid.UUID) ([]models.Context, error) {
	contexts, err := s.contextRepo.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing contexts: %w", err)
	}
	return contexts, nil
}

// GetSnapshot returns a full snapshot of all active (non-expired) context entries
// in a workspace. This uses the repository's DISTINCT ON query to get the latest
// value for each key.
func (s *ContextService) GetSnapshot(ctx context.Context, workspaceID uuid.UUID) (*ContextSnapshot, error) {
	contexts, err := s.contextRepo.GetSnapshot(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("getting context snapshot: %w", err)
	}

	return &ContextSnapshot{
		WorkspaceID: workspaceID,
		Contexts:    contexts,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// ---------- helpers ----------

// logContextChange marshals the payload and writes a sync log entry.
func (s *ContextService) logContextChange(ctx context.Context, workspaceID uuid.UUID, entityID, agentID uuid.UUID, action string, payload interface{}) {
	data, _ := json.Marshal(payload)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	entry := models.NewSyncLogEntry(workspaceID, models.SyncEntityContext, entityID, agentID, action, hash)
	_ = s.syncRepo.LogChange(ctx, entry)
}
