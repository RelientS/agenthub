package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agenthub/server/internal/models"
)

// AgentRepository handles database operations for agents.
type AgentRepository struct {
	pool *pgxpool.Pool
}

// NewAgentRepository creates a new AgentRepository.
func NewAgentRepository(pool *pgxpool.Pool) *AgentRepository {
	return &AgentRepository{pool: pool}
}

// Create inserts a new agent into the database.
func (r *AgentRepository) Create(ctx context.Context, agent *models.Agent) error {
	query := `
		INSERT INTO agents (id, workspace_id, name, role, status, capabilities, last_heartbeat, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		agent.ID,
		agent.WorkspaceID,
		agent.Name,
		agent.Role,
		agent.Status,
		agent.Capabilities,
		agent.LastHeartbeat,
		agent.Metadata,
		agent.CreatedAt,
		agent.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("agent create: %w", err)
	}
	return nil
}

// GetByID retrieves an agent by its ID.
func (r *AgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	query := `
		SELECT id, workspace_id, name, role, status, capabilities, last_heartbeat, metadata, created_at, updated_at
		FROM agents
		WHERE id = $1`

	a := &models.Agent{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
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
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("agent not found: %w", err)
		}
		return nil, fmt.Errorf("agent get by id: %w", err)
	}
	return a, nil
}

// UpdateStatus updates the status of an agent.
func (r *AgentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE agents
		SET status = $1, updated_at = NOW()
		WHERE id = $2`

	result, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("agent update status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent update status: no rows affected")
	}
	return nil
}

// UpdateHeartbeat updates the last heartbeat timestamp for an agent.
func (r *AgentRepository) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE agents
		SET last_heartbeat = NOW(), updated_at = NOW()
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("agent update heartbeat: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent update heartbeat: no rows affected")
	}
	return nil
}

// Delete removes an agent from the database.
func (r *AgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM agents WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("agent delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent delete: no rows affected")
	}
	return nil
}

// ListByWorkspace retrieves all agents in a workspace.
func (r *AgentRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Agent, error) {
	query := `
		SELECT id, workspace_id, name, role, status, capabilities, last_heartbeat, metadata, created_at, updated_at
		FROM agents
		WHERE workspace_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("agent list by workspace: %w", err)
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
			return nil, fmt.Errorf("agent list by workspace scan: %w", err)
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("agent list by workspace rows: %w", err)
	}
	return agents, nil
}
