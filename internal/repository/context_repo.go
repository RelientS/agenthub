package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agenthub/server/internal/models"
)

// contextColumns returns the standard column list for context queries.
func contextColumns() string {
	return `id, workspace_id, context_type, title, content, content_hash,
	        version, updated_by, tags, created_at, updated_at`
}

// scanContextRows scans multiple context rows into a slice.
func scanContextRows(rows pgx.Rows) ([]models.Context, error) {
	var contexts []models.Context
	for rows.Next() {
		var c models.Context
		if err := rows.Scan(
			&c.ID,
			&c.WorkspaceID,
			&c.ContextType,
			&c.Title,
			&c.Content,
			&c.ContentHash,
			&c.Version,
			&c.UpdatedBy,
			&c.Tags,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contexts = append(contexts, c)
	}
	return contexts, rows.Err()
}

// ContextRepository handles database operations for shared context entries.
type ContextRepository struct {
	pool *pgxpool.Pool
}

// NewContextRepository creates a new ContextRepository.
func NewContextRepository(pool *pgxpool.Pool) *ContextRepository {
	return &ContextRepository{pool: pool}
}

// Create inserts a new context entry into the database.
func (r *ContextRepository) Create(ctx context.Context, c *models.Context) error {
	query := `
		INSERT INTO contexts (
			id, workspace_id, context_type, title, content, content_hash,
			version, updated_by, tags, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.pool.Exec(ctx, query,
		c.ID,
		c.WorkspaceID,
		c.ContextType,
		c.Title,
		c.Content,
		c.ContentHash,
		c.Version,
		c.UpdatedBy,
		c.Tags,
		c.CreatedAt,
		c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("context create: %w", err)
	}
	return nil
}

// GetByID retrieves a context entry by its ID.
func (r *ContextRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Context, error) {
	query := fmt.Sprintf(`SELECT %s FROM contexts WHERE id = $1`, contextColumns())

	c := &models.Context{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.WorkspaceID,
		&c.ContextType,
		&c.Title,
		&c.Content,
		&c.ContentHash,
		&c.Version,
		&c.UpdatedBy,
		&c.Tags,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("context not found: %w", err)
		}
		return nil, fmt.Errorf("context get by id: %w", err)
	}
	return c, nil
}

// Update updates an existing context entry in the database.
func (r *ContextRepository) Update(ctx context.Context, c *models.Context) error {
	query := `
		UPDATE contexts
		SET context_type = $1, title = $2, content = $3, content_hash = $4,
		    version = $5, updated_by = $6, tags = $7, updated_at = $8
		WHERE id = $9`

	result, err := r.pool.Exec(ctx, query,
		c.ContextType,
		c.Title,
		c.Content,
		c.ContentHash,
		c.Version,
		c.UpdatedBy,
		c.Tags,
		c.UpdatedAt,
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("context update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("context update: no rows affected")
	}
	return nil
}

// ListByWorkspace retrieves all context entries for a workspace.
func (r *ContextRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Context, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM contexts
		WHERE workspace_id = $1
		ORDER BY title ASC`, contextColumns())

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("context list by workspace: %w", err)
	}
	defer rows.Close()

	contexts, err := scanContextRows(rows)
	if err != nil {
		return nil, fmt.Errorf("context list by workspace scan: %w", err)
	}
	return contexts, nil
}

// GetSnapshot retrieves the current context snapshot for a workspace.
// Returns the latest version of each context entry (by title), ordered by title.
func (r *ContextRepository) GetSnapshot(ctx context.Context, workspaceID uuid.UUID) ([]models.Context, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT ON (title)
			%s
		FROM contexts
		WHERE workspace_id = $1
		ORDER BY title ASC, version DESC`, contextColumns())

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("context get snapshot: %w", err)
	}
	defer rows.Close()

	contexts, err := scanContextRows(rows)
	if err != nil {
		return nil, fmt.Errorf("context get snapshot scan: %w", err)
	}
	return contexts, nil
}
