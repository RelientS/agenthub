package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agenthub/server/internal/models"
)

// ArtifactFilters holds optional filters for listing artifacts.
type ArtifactFilters struct {
	ArtifactType string
	CreatedBy    *uuid.UUID
	Language     string
	Tags         []string
	Limit        int
	Offset       int
}

// ArtifactRepository handles database operations for artifacts.
type ArtifactRepository struct {
	pool *pgxpool.Pool
}

// NewArtifactRepository creates a new ArtifactRepository.
func NewArtifactRepository(pool *pgxpool.Pool) *ArtifactRepository {
	return &ArtifactRepository{pool: pool}
}

// scanArtifact scans a single artifact row into a models.Artifact struct.
func scanArtifact(row pgx.Row) (*models.Artifact, error) {
	a := &models.Artifact{}
	err := row.Scan(
		&a.ID,
		&a.WorkspaceID,
		&a.CreatedBy,
		&a.ArtifactType,
		&a.Name,
		&a.Description,
		&a.Content,
		&a.ContentHash,
		&a.Version,
		&a.ParentVersion,
		&a.FilePath,
		&a.Language,
		&a.Tags,
		&a.Metadata,
		&a.CreatedAt,
	)
	return a, err
}

// artifactColumns returns the standard column list for artifact queries.
func artifactColumns() string {
	return `id, workspace_id, created_by, artifact_type, name, description,
	        content, content_hash, version, parent_version, file_path, language,
	        tags, metadata, created_at`
}

// Create inserts a new artifact into the database.
func (r *ArtifactRepository) Create(ctx context.Context, art *models.Artifact) error {
	query := `
		INSERT INTO artifacts (
			id, workspace_id, created_by, artifact_type, name, description,
			content, content_hash, version, parent_version, file_path, language,
			tags, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := r.pool.Exec(ctx, query,
		art.ID,
		art.WorkspaceID,
		art.CreatedBy,
		art.ArtifactType,
		art.Name,
		art.Description,
		art.Content,
		art.ContentHash,
		art.Version,
		art.ParentVersion,
		art.FilePath,
		art.Language,
		art.Tags,
		art.Metadata,
		art.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("artifact create: %w", err)
	}
	return nil
}

// GetByID retrieves an artifact by its ID.
func (r *ArtifactRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Artifact, error) {
	query := fmt.Sprintf(`SELECT %s FROM artifacts WHERE id = $1`, artifactColumns())

	a, err := scanArtifact(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("artifact not found: %w", err)
		}
		return nil, fmt.Errorf("artifact get by id: %w", err)
	}
	return a, nil
}

// ListByWorkspace retrieves artifacts for a workspace with optional filters.
// Returns the artifact list, total count, and any error.
func (r *ArtifactRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, filters ArtifactFilters) ([]models.Artifact, int, error) {
	conditions := []string{"workspace_id = $1"}
	args := []interface{}{workspaceID}
	argIdx := 2

	if filters.ArtifactType != "" {
		conditions = append(conditions, fmt.Sprintf("artifact_type = $%d", argIdx))
		args = append(args, filters.ArtifactType)
		argIdx++
	}
	if filters.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIdx))
		args = append(args, *filters.CreatedBy)
		argIdx++
	}
	if filters.Language != "" {
		conditions = append(conditions, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, filters.Language)
		argIdx++
	}
	if len(filters.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, filters.Tags)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total matching rows.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM artifacts WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("artifact list count: %w", err)
	}

	// Build the data query with pagination.
	dataQuery := fmt.Sprintf(`SELECT %s FROM artifacts WHERE %s ORDER BY created_at DESC`,
		artifactColumns(), whereClause)

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("artifact list query: %w", err)
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var a models.Artifact
		if err := rows.Scan(
			&a.ID,
			&a.WorkspaceID,
			&a.CreatedBy,
			&a.ArtifactType,
			&a.Name,
			&a.Description,
			&a.Content,
			&a.ContentHash,
			&a.Version,
			&a.ParentVersion,
			&a.FilePath,
			&a.Language,
			&a.Tags,
			&a.Metadata,
			&a.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("artifact list scan: %w", err)
		}
		artifacts = append(artifacts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("artifact list rows: %w", err)
	}
	return artifacts, total, nil
}

// GetHistory retrieves all versions of an artifact by name within a workspace,
// ordered from newest to oldest.
func (r *ArtifactRepository) GetHistory(ctx context.Context, workspaceID uuid.UUID, name string) ([]models.Artifact, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM artifacts
		WHERE workspace_id = $1 AND name = $2
		ORDER BY version DESC`, artifactColumns())

	rows, err := r.pool.Query(ctx, query, workspaceID, name)
	if err != nil {
		return nil, fmt.Errorf("artifact get history: %w", err)
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var a models.Artifact
		if err := rows.Scan(
			&a.ID,
			&a.WorkspaceID,
			&a.CreatedBy,
			&a.ArtifactType,
			&a.Name,
			&a.Description,
			&a.Content,
			&a.ContentHash,
			&a.Version,
			&a.ParentVersion,
			&a.FilePath,
			&a.Language,
			&a.Tags,
			&a.Metadata,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("artifact get history scan: %w", err)
		}
		artifacts = append(artifacts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("artifact get history rows: %w", err)
	}
	return artifacts, nil
}

// Search performs a full-text search on artifact names, descriptions, and content
// within a workspace.
func (r *ArtifactRepository) Search(ctx context.Context, workspaceID uuid.UUID, query string) ([]models.Artifact, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT %s
		FROM artifacts
		WHERE workspace_id = $1
		  AND (
		    name ILIKE '%%' || $2 || '%%'
		    OR description ILIKE '%%' || $2 || '%%'
		    OR content ILIKE '%%' || $2 || '%%'
		    OR file_path ILIKE '%%' || $2 || '%%'
		  )
		ORDER BY
			CASE
				WHEN name ILIKE '%%' || $2 || '%%' THEN 0
				WHEN file_path ILIKE '%%' || $2 || '%%' THEN 1
				WHEN description ILIKE '%%' || $2 || '%%' THEN 2
				ELSE 3
			END,
			created_at DESC
		LIMIT 100`, artifactColumns())

	rows, err := r.pool.Query(ctx, sqlQuery, workspaceID, query)
	if err != nil {
		return nil, fmt.Errorf("artifact search: %w", err)
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var a models.Artifact
		if err := rows.Scan(
			&a.ID,
			&a.WorkspaceID,
			&a.CreatedBy,
			&a.ArtifactType,
			&a.Name,
			&a.Description,
			&a.Content,
			&a.ContentHash,
			&a.Version,
			&a.ParentVersion,
			&a.FilePath,
			&a.Language,
			&a.Tags,
			&a.Metadata,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("artifact search scan: %w", err)
		}
		artifacts = append(artifacts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("artifact search rows: %w", err)
	}
	return artifacts, nil
}

// GetByHash retrieves an artifact by its content hash (for deduplication).
func (r *ArtifactRepository) GetByHash(ctx context.Context, hash string) (*models.Artifact, error) {
	query := fmt.Sprintf(`SELECT %s FROM artifacts WHERE content_hash = $1 LIMIT 1`, artifactColumns())

	a, err := scanArtifact(r.pool.QueryRow(ctx, query, hash))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("artifact not found for hash: %w", err)
		}
		return nil, fmt.Errorf("artifact get by hash: %w", err)
	}
	return a, nil
}
