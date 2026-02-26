package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/pkg/conflict"
	"github.com/agenthub/server/internal/pkg/events"
	"github.com/agenthub/server/internal/repository"
	"github.com/google/uuid"
)

// ---------- request / response types ----------

// CreateArtifactInput holds the input for creating an artifact.
type CreateArtifactInput struct {
	Name         string                 `json:"name"`
	ArtifactType string                 `json:"artifact_type"`
	Description  string                 `json:"description,omitempty"`
	Content      string                 `json:"content"`
	FilePath     string                 `json:"file_path,omitempty"`
	Language     string                 `json:"language,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ArtifactListFilter holds filter parameters for listing artifacts.
type ArtifactListFilter struct {
	ArtifactType string
	CreatedBy    *uuid.UUID
	Language     string
	Tags         []string
	Limit        int
	Offset       int
}

// ---------- service ----------

// ArtifactService handles artifact business logic.
type ArtifactService struct {
	artifactRepo *repository.ArtifactRepository
	syncRepo     *repository.SyncRepository
	eventBus     *events.Bus
	resolver     *conflict.Resolver
}

// NewArtifactService creates a new ArtifactService.
func NewArtifactService(
	ar *repository.ArtifactRepository,
	sr *repository.SyncRepository,
	eb *events.Bus,
	resolver *conflict.Resolver,
) *ArtifactService {
	return &ArtifactService{
		artifactRepo: ar,
		syncRepo:     sr,
		eventBus:     eb,
		resolver:     resolver,
	}
}

// CreateArtifact creates a new artifact in a workspace. It computes the SHA-256
// content hash and checks for conflicts with existing versions of the same name.
// If a version already exists with a different hash, a new version is created
// using the last-write-wins conflict resolution strategy.
func (s *ArtifactService) CreateArtifact(ctx context.Context, workspaceID uuid.UUID, input CreateArtifactInput, createdBy uuid.UUID) (*models.Artifact, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("artifact name is required")
	}
	if input.Content == "" {
		return nil, fmt.Errorf("artifact content is required")
	}
	if !models.IsValidArtifactType(input.ArtifactType) {
		return nil, fmt.Errorf("invalid artifact type: %s", input.ArtifactType)
	}

	// Compute content hash.
	contentHash := computeHash(input.Content)

	// Check for existing version of this artifact name.
	existing, err := s.artifactRepo.GetHistory(ctx, workspaceID, input.Name)
	if err != nil {
		return nil, fmt.Errorf("checking existing artifact versions: %w", err)
	}

	version := 1
	var parentVersion *int

	if len(existing) > 0 {
		latest := existing[0] // History is ordered newest first.

		// Use the conflict resolver to determine the new version.
		resolution, conflictErr := s.resolver.ResolveArtifact(
			latest.ID.String(),
			latest.ContentHash,
			contentHash,
			latest.Version,
		)

		if conflictErr != nil {
			// Conflict detected but LWW accepts it. Log an event.
			s.eventBus.Publish(events.NewEvent(events.EventConflictDetected, workspaceID.String(), map[string]interface{}{
				"artifact_name": input.Name,
				"conflict":      conflictErr.Error(),
			}))
		}

		version = resolution.NewVersion
		pv := latest.Version
		parentVersion = &pv

		// If the resolution indicates no change was detected, return the existing artifact.
		if resolution.NewVersion == latest.Version && conflictErr == nil {
			return &latest, nil
		}
	}

	tags := input.Tags
	if tags == nil {
		tags = []string{}
	}
	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	artifact := models.NewArtifact(workspaceID, createdBy, input.ArtifactType, input.Name, input.Content, contentHash)
	artifact.Description = input.Description
	artifact.FilePath = input.FilePath
	artifact.Language = input.Language
	artifact.Tags = tags
	artifact.Metadata = metadata
	artifact.Version = version
	artifact.ParentVersion = parentVersion

	if err := s.artifactRepo.Create(ctx, artifact); err != nil {
		return nil, fmt.Errorf("creating artifact: %w", err)
	}

	s.logChange(ctx, workspaceID, models.SyncEntityArtifact, artifact.ID, createdBy, models.SyncActionCreate, artifact)

	eventType := events.EventArtifactNew
	if parentVersion != nil {
		eventType = events.EventArtifactUpdated
	}
	s.eventBus.Publish(events.NewEvent(eventType, workspaceID.String(), map[string]interface{}{
		"artifact_id": artifact.ID,
		"name":        artifact.Name,
		"version":     artifact.Version,
	}))

	return artifact, nil
}

// GetArtifact retrieves an artifact by ID.
func (s *ArtifactService) GetArtifact(ctx context.Context, workspaceID uuid.UUID, artifactID uuid.UUID) (*models.Artifact, error) {
	artifact, err := s.artifactRepo.GetByID(ctx, artifactID)
	if err != nil {
		return nil, fmt.Errorf("getting artifact: %w", err)
	}
	if artifact.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("artifact does not belong to this workspace")
	}
	return artifact, nil
}

// ListArtifacts returns artifacts matching the given filter criteria.
func (s *ArtifactService) ListArtifacts(ctx context.Context, workspaceID uuid.UUID, filter ArtifactListFilter) ([]models.Artifact, error) {
	repoFilter := repository.ArtifactFilters{
		ArtifactType: filter.ArtifactType,
		CreatedBy:    filter.CreatedBy,
		Language:     filter.Language,
		Tags:         filter.Tags,
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	}

	artifacts, _, err := s.artifactRepo.ListByWorkspace(ctx, workspaceID, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("listing artifacts: %w", err)
	}
	return artifacts, nil
}

// GetHistory returns the version history for an artifact identified by name.
func (s *ArtifactService) GetHistory(ctx context.Context, workspaceID uuid.UUID, name string) ([]models.Artifact, error) {
	history, err := s.artifactRepo.GetHistory(ctx, workspaceID, name)
	if err != nil {
		return nil, fmt.Errorf("getting artifact history: %w", err)
	}
	return history, nil
}

// SearchArtifacts performs a text search across artifact names, descriptions,
// and content within a workspace.
func (s *ArtifactService) SearchArtifacts(ctx context.Context, workspaceID uuid.UUID, query string) ([]models.Artifact, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	artifacts, err := s.artifactRepo.Search(ctx, workspaceID, query)
	if err != nil {
		return nil, fmt.Errorf("searching artifacts: %w", err)
	}
	return artifacts, nil
}

// CheckConflict compares the content hash of incoming content against the latest
// version of the named artifact and returns any detected conflict.
func (s *ArtifactService) CheckConflict(ctx context.Context, workspaceID uuid.UUID, name string, incomingContent string) (*conflict.Resolution, *conflict.ConflictError) {
	incomingHash := computeHash(incomingContent)

	existing, err := s.artifactRepo.GetHistory(ctx, workspaceID, name)
	if err != nil || len(existing) == 0 {
		// No existing version, no conflict.
		return &conflict.Resolution{
			Strategy:   conflict.StrategyLastWriteWins,
			Accepted:   true,
			NewVersion: 1,
			Message:    "no existing version found",
		}, nil
	}

	latest := existing[0]
	return s.resolver.ResolveArtifact(latest.ID.String(), latest.ContentHash, incomingHash, latest.Version)
}

// ---------- helpers ----------

// computeHash returns the hex-encoded SHA-256 hash of the given content.
func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// logChange marshals the payload and writes a sync log entry.
func (s *ArtifactService) logChange(ctx context.Context, workspaceID uuid.UUID, entityType string, entityID, agentID uuid.UUID, action string, payload interface{}) {
	data, _ := json.Marshal(payload)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	entry := models.NewSyncLogEntry(workspaceID, entityType, entityID, agentID, action, hash)
	_ = s.syncRepo.LogChange(ctx, entry)
}
