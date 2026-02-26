package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agenthub/server/internal/models"
)

// WorkspaceRepository handles database operations for workspaces.
type WorkspaceRepository struct {
	pool *pgxpool.Pool
}

// NewWorkspaceRepository creates a new WorkspaceRepository.
func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{pool: pool}
}

// Create inserts a new workspace into the database.
func (r *WorkspaceRepository) Create(ctx context.Context, ws *models.Workspace) error {
	query := `
		INSERT INTO workspaces (id, name, description, owner_agent_id, invite_code, status, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		ws.ID,
		ws.Name,
		ws.Description,
		ws.OwnerAgentID,
		ws.InviteCode,
		ws.Status,
		ws.Settings,
		ws.CreatedAt,
		ws.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("workspace create: %w", err)
	}
	return nil
}

// GetByID retrieves a workspace by its ID.
func (r *WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Workspace, error) {
	query := `
		SELECT id, name, description, owner_agent_id, invite_code, status, settings, created_at, updated_at
		FROM workspaces
		WHERE id = $1`

	ws := &models.Workspace{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ws.ID,
		&ws.Name,
		&ws.Description,
		&ws.OwnerAgentID,
		&ws.InviteCode,
		&ws.Status,
		&ws.Settings,
		&ws.CreatedAt,
		&ws.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workspace not found: %w", err)
		}
		return nil, fmt.Errorf("workspace get by id: %w", err)
	}
	return ws, nil
}

// GetByInviteCode retrieves a workspace by its invite code.
func (r *WorkspaceRepository) GetByInviteCode(ctx context.Context, code string) (*models.Workspace, error) {
	query := `
		SELECT id, name, description, owner_agent_id, invite_code, status, settings, created_at, updated_at
		FROM workspaces
		WHERE invite_code = $1 AND status = $2`

	ws := &models.Workspace{}
	err := r.pool.QueryRow(ctx, query, code, models.WorkspaceStatusActive).Scan(
		&ws.ID,
		&ws.Name,
		&ws.Description,
		&ws.OwnerAgentID,
		&ws.InviteCode,
		&ws.Status,
		&ws.Settings,
		&ws.CreatedAt,
		&ws.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workspace not found for invite code: %w", err)
		}
		return nil, fmt.Errorf("workspace get by invite code: %w", err)
	}
	return ws, nil
}

// Update updates an existing workspace in the database.
func (r *WorkspaceRepository) Update(ctx context.Context, ws *models.Workspace) error {
	query := `
		UPDATE workspaces
		SET name = $1, description = $2, owner_agent_id = $3, invite_code = $4,
		    status = $5, settings = $6, updated_at = $7
		WHERE id = $8`

	result, err := r.pool.Exec(ctx, query,
		ws.Name,
		ws.Description,
		ws.OwnerAgentID,
		ws.InviteCode,
		ws.Status,
		ws.Settings,
		ws.UpdatedAt,
		ws.ID,
	)
	if err != nil {
		return fmt.Errorf("workspace update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("workspace update: no rows affected")
	}
	return nil
}

// Delete soft-deletes a workspace by setting its status to deleted.
func (r *WorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE workspaces
		SET status = $1, updated_at = NOW()
		WHERE id = $2`

	result, err := r.pool.Exec(ctx, query, models.WorkspaceStatusDeleted, id)
	if err != nil {
		return fmt.Errorf("workspace delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("workspace delete: no rows affected")
	}
	return nil
}

// ListAgents retrieves all agents belonging to a workspace.
func (r *WorkspaceRepository) ListAgents(ctx context.Context, workspaceID uuid.UUID) ([]models.Agent, error) {
	query := `
		SELECT id, workspace_id, name, role, status, capabilities, last_heartbeat, metadata, created_at, updated_at
		FROM agents
		WHERE workspace_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace list agents: %w", err)
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(
			&a.ID,
			&a.WorkspaceID,
			&a.Name,
			&a.Role,
			&a.Status,
			&a.Capabilities,
			&a.LastHeartbeat,
			&a.Metadata,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("workspace list agents scan: %w", err)
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("workspace list agents rows: %w", err)
	}
	return agents, nil
}
