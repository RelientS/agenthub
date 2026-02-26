package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agenthub/server/internal/models"
)

// SyncRepository handles database operations for the sync log.
type SyncRepository struct {
	pool *pgxpool.Pool
}

// NewSyncRepository creates a new SyncRepository.
func NewSyncRepository(pool *pgxpool.Pool) *SyncRepository {
	return &SyncRepository{pool: pool}
}

// LogChange inserts a new sync log entry into the database.
// The ID field is populated by the database (auto-increment) and set on the entry upon return.
func (r *SyncRepository) LogChange(ctx context.Context, entry *models.SyncLogEntry) error {
	query := `
		INSERT INTO sync_log (workspace_id, entity_type, entity_id, action, agent_id, timestamp, payload_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := r.pool.QueryRow(ctx, query,
		entry.WorkspaceID,
		entry.EntityType,
		entry.EntityID,
		entry.Action,
		entry.AgentID,
		entry.Timestamp,
		entry.PayloadHash,
	).Scan(&entry.ID)
	if err != nil {
		return fmt.Errorf("sync log change: %w", err)
	}
	return nil
}

// GetChangesSince retrieves sync log entries after a given sync ID for a workspace.
// Results can be filtered by entity types and limited to a maximum number of entries.
func (r *SyncRepository) GetChangesSince(ctx context.Context, workspaceID uuid.UUID, sinceID int64, entityTypes []string, limit int) ([]models.SyncLogEntry, error) {
	conditions := []string{"workspace_id = $1", "id > $2"}
	args := []interface{}{workspaceID, sinceID}
	argIdx := 3

	if len(entityTypes) > 0 {
		conditions = append(conditions, fmt.Sprintf("entity_type = ANY($%d)", argIdx))
		args = append(args, entityTypes)
		argIdx++
	}

	if limit <= 0 {
		limit = 100
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT id, workspace_id, entity_type, entity_id, action, agent_id, timestamp, payload_hash
		FROM sync_log
		WHERE %s
		ORDER BY id ASC
		LIMIT $%d`, whereClause, argIdx)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sync get changes since: %w", err)
	}
	defer rows.Close()

	var entries []models.SyncLogEntry
	for rows.Next() {
		var e models.SyncLogEntry
		if err := rows.Scan(
			&e.ID,
			&e.WorkspaceID,
			&e.EntityType,
			&e.EntityID,
			&e.Action,
			&e.AgentID,
			&e.Timestamp,
			&e.PayloadHash,
		); err != nil {
			return nil, fmt.Errorf("sync get changes since scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sync get changes since rows: %w", err)
	}
	return entries, nil
}

// GetLatestSyncID retrieves the most recent sync log entry ID for a workspace.
// Returns 0 if no entries exist.
func (r *SyncRepository) GetLatestSyncID(ctx context.Context, workspaceID uuid.UUID) (int64, error) {
	query := `
		SELECT COALESCE(MAX(id), 0)
		FROM sync_log
		WHERE workspace_id = $1`

	var latestID int64
	err := r.pool.QueryRow(ctx, query, workspaceID).Scan(&latestID)
	if err != nil {
		return 0, fmt.Errorf("sync get latest id: %w", err)
	}
	return latestID, nil
}

// CleanOldEntries deletes sync log entries older than the specified retention period.
func (r *SyncRepository) CleanOldEntries(ctx context.Context, retentionDays int) error {
	query := `
		DELETE FROM sync_log
		WHERE timestamp < NOW() - make_interval(days => $1)`

	_, err := r.pool.Exec(ctx, query, retentionDays)
	if err != nil {
		return fmt.Errorf("sync clean old entries: %w", err)
	}
	return nil
}
