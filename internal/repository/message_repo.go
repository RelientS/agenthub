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

// MessageFilters holds optional filters for listing messages.
type MessageFilters struct {
	FromAgentID *uuid.UUID
	ToAgentID   *uuid.UUID
	MessageType string
	ThreadID    *uuid.UUID
	Limit       int
	Offset      int
}

// messageColumns returns the standard column list for message queries.
func messageColumns() string {
	return `id, workspace_id, from_agent_id, to_agent_id, thread_id, message_type,
	        payload, ref_task_id, ref_artifact_id, is_read, created_at`
}

// scanMessageRows scans multiple message rows into a slice.
func scanMessageRows(rows pgx.Rows) ([]models.Message, error) {
	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(
			&m.ID,
			&m.WorkspaceID,
			&m.FromAgentID,
			&m.ToAgentID,
			&m.ThreadID,
			&m.MessageType,
			&m.Payload,
			&m.RefTaskID,
			&m.RefArtifactID,
			&m.IsRead,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// MessageRepository handles database operations for messages.
type MessageRepository struct {
	pool *pgxpool.Pool
}

// NewMessageRepository creates a new MessageRepository.
func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

// Create inserts a new message into the database.
func (r *MessageRepository) Create(ctx context.Context, msg *models.Message) error {
	query := `
		INSERT INTO messages (
			id, workspace_id, from_agent_id, to_agent_id, thread_id, message_type,
			payload, ref_task_id, ref_artifact_id, is_read, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.pool.Exec(ctx, query,
		msg.ID,
		msg.WorkspaceID,
		msg.FromAgentID,
		msg.ToAgentID,
		msg.ThreadID,
		msg.MessageType,
		msg.Payload,
		msg.RefTaskID,
		msg.RefArtifactID,
		msg.IsRead,
		msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("message create: %w", err)
	}
	return nil
}

// GetByID retrieves a message by its ID.
func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	query := fmt.Sprintf(`SELECT %s FROM messages WHERE id = $1`, messageColumns())

	m := &models.Message{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.WorkspaceID,
		&m.FromAgentID,
		&m.ToAgentID,
		&m.ThreadID,
		&m.MessageType,
		&m.Payload,
		&m.RefTaskID,
		&m.RefArtifactID,
		&m.IsRead,
		&m.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("message not found: %w", err)
		}
		return nil, fmt.Errorf("message get by id: %w", err)
	}
	return m, nil
}

// ListByWorkspace retrieves messages for a workspace with optional filters.
// Returns the message list, total count, and any error.
func (r *MessageRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, filters MessageFilters) ([]models.Message, int, error) {
	conditions := []string{"workspace_id = $1"}
	args := []interface{}{workspaceID}
	argIdx := 2

	if filters.FromAgentID != nil {
		conditions = append(conditions, fmt.Sprintf("from_agent_id = $%d", argIdx))
		args = append(args, *filters.FromAgentID)
		argIdx++
	}
	if filters.ToAgentID != nil {
		conditions = append(conditions, fmt.Sprintf("to_agent_id = $%d", argIdx))
		args = append(args, *filters.ToAgentID)
		argIdx++
	}
	if filters.MessageType != "" {
		conditions = append(conditions, fmt.Sprintf("message_type = $%d", argIdx))
		args = append(args, filters.MessageType)
		argIdx++
	}
	if filters.ThreadID != nil {
		conditions = append(conditions, fmt.Sprintf("thread_id = $%d", argIdx))
		args = append(args, *filters.ThreadID)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total matching rows.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM messages WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("message list count: %w", err)
	}

	// Build the data query with pagination.
	dataQuery := fmt.Sprintf(`SELECT %s FROM messages WHERE %s ORDER BY created_at DESC`,
		messageColumns(), whereClause)

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
		return nil, 0, fmt.Errorf("message list query: %w", err)
	}
	defer rows.Close()

	messages, err := scanMessageRows(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("message list scan: %w", err)
	}
	return messages, total, nil
}

// GetUnread retrieves all unread messages for a specific agent in a workspace.
// A message is unread if is_read is false and is directed to the agent
// (either via to_agent_id or as a broadcast with NULL to_agent_id).
func (r *MessageRepository) GetUnread(ctx context.Context, workspaceID, agentID uuid.UUID) ([]models.Message, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM messages
		WHERE workspace_id = $1
		  AND is_read = false
		  AND from_agent_id != $2
		  AND (to_agent_id = $2 OR to_agent_id IS NULL)
		ORDER BY created_at ASC`, messageColumns())

	rows, err := r.pool.Query(ctx, query, workspaceID, agentID)
	if err != nil {
		return nil, fmt.Errorf("message get unread: %w", err)
	}
	defer rows.Close()

	messages, err := scanMessageRows(rows)
	if err != nil {
		return nil, fmt.Errorf("message get unread scan: %w", err)
	}
	return messages, nil
}

// GetUnreadCount returns the number of unread messages for a specific agent in a workspace.
func (r *MessageRepository) GetUnreadCount(ctx context.Context, workspaceID, agentID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM messages
		WHERE workspace_id = $1
		  AND is_read = false
		  AND from_agent_id != $2
		  AND (to_agent_id = $2 OR to_agent_id IS NULL)`

	var count int
	err := r.pool.QueryRow(ctx, query, workspaceID, agentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("message get unread count: %w", err)
	}
	return count, nil
}

// MarkAsRead marks a message as read by setting is_read to true.
func (r *MessageRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE messages
		SET is_read = true
		WHERE id = $1 AND is_read = false`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("message mark as read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("message mark as read: no rows affected (already read or not found)")
	}
	return nil
}

// GetThread retrieves all messages in a thread, ordered chronologically.
func (r *MessageRepository) GetThread(ctx context.Context, threadID uuid.UUID) ([]models.Message, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM messages
		WHERE thread_id = $1 OR id = $1
		ORDER BY created_at ASC`, messageColumns())

	rows, err := r.pool.Query(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("message get thread: %w", err)
	}
	defer rows.Close()

	messages, err := scanMessageRows(rows)
	if err != nil {
		return nil, fmt.Errorf("message get thread scan: %w", err)
	}
	return messages, nil
}
