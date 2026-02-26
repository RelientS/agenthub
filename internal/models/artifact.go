package models

import (
	"time"

	"github.com/google/uuid"
)

// Artifact type constants.
const (
	ArtifactTypeCodeSnippet    = "code_snippet"
	ArtifactTypeAPISchema      = "api_schema"
	ArtifactTypeTypeDefinition = "type_definition"
	ArtifactTypeTestResult     = "test_result"
	ArtifactTypeMigration      = "migration"
	ArtifactTypeConfig         = "config"
	ArtifactTypeDoc            = "doc"
)

// ValidArtifactTypes contains all valid artifact types for validation.
var ValidArtifactTypes = []string{
	ArtifactTypeCodeSnippet,
	ArtifactTypeAPISchema,
	ArtifactTypeTypeDefinition,
	ArtifactTypeTestResult,
	ArtifactTypeMigration,
	ArtifactTypeConfig,
	ArtifactTypeDoc,
}

// Artifact represents a shared artifact produced by an agent within a workspace.
type Artifact struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	WorkspaceID   uuid.UUID              `json:"workspace_id" db:"workspace_id"`
	CreatedBy     uuid.UUID              `json:"created_by" db:"created_by"`
	ArtifactType  string                 `json:"artifact_type" db:"artifact_type"`
	Name          string                 `json:"name" db:"name"`
	Description   string                 `json:"description,omitempty" db:"description"`
	Content       string                 `json:"content" db:"content"`
	ContentHash   string                 `json:"content_hash" db:"content_hash"`
	Version       int                    `json:"version" db:"version"`
	ParentVersion *int                   `json:"parent_version,omitempty" db:"parent_version"`
	FilePath      string                 `json:"file_path,omitempty" db:"file_path"`
	Language      string                 `json:"language,omitempty" db:"language"`
	Tags          []string               `json:"tags,omitempty" db:"tags"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// NewArtifact creates a new Artifact with default values.
func NewArtifact(workspaceID, createdBy uuid.UUID, artifactType, name, content, contentHash string) *Artifact {
	return &Artifact{
		ID:           uuid.New(),
		WorkspaceID:  workspaceID,
		CreatedBy:    createdBy,
		ArtifactType: artifactType,
		Name:         name,
		Content:      content,
		ContentHash:  contentHash,
		Version:      1,
		Tags:         []string{},
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now().UTC(),
	}
}

// IsValidArtifactType checks whether the given type is a valid artifact type.
func IsValidArtifactType(artType string) bool {
	for _, t := range ValidArtifactTypes {
		if t == artType {
			return true
		}
	}
	return false
}
