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

// TaskFilters holds optional filters for listing tasks.
type TaskFilters struct {
	Status     string
	AssignedTo *uuid.UUID
	Priority   *int
	Tags       []string
	ParentID   *uuid.UUID
	Limit      int
	Offset     int
}

// taskColumns returns the standard column list for task queries.
func taskColumns() string {
	return `id, workspace_id, parent_id, title, description, status, priority,
	        assigned_to, created_by, depends_on, branch_name, estimated_hours,
	        tags, metadata, created_at, updated_at, completed_at`
}

// scanTask scans a single task row into a models.Task struct.
func scanTask(row pgx.Row) (*models.Task, error) {
	t := &models.Task{}
	err := row.Scan(
		&t.ID,
		&t.WorkspaceID,
		&t.ParentID,
		&t.Title,
		&t.Description,
		&t.Status,
		&t.Priority,
		&t.AssignedTo,
		&t.CreatedBy,
		&t.DependsOn,
		&t.BranchName,
		&t.EstimatedHours,
		&t.Tags,
		&t.Metadata,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.CompletedAt,
	)
	return t, err
}

// scanTaskRows scans multiple task rows into a slice.
func scanTaskRows(rows pgx.Rows) ([]models.Task, error) {
	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(
			&t.ID,
			&t.WorkspaceID,
			&t.ParentID,
			&t.Title,
			&t.Description,
			&t.Status,
			&t.Priority,
			&t.AssignedTo,
			&t.CreatedBy,
			&t.DependsOn,
			&t.BranchName,
			&t.EstimatedHours,
			&t.Tags,
			&t.Metadata,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.CompletedAt,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// TaskRepository handles database operations for tasks.
type TaskRepository struct {
	pool *pgxpool.Pool
}

// NewTaskRepository creates a new TaskRepository.
func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

// Create inserts a new task into the database.
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	query := `
		INSERT INTO tasks (
			id, workspace_id, parent_id, title, description, status, priority,
			assigned_to, created_by, depends_on, branch_name, estimated_hours,
			tags, metadata, created_at, updated_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	_, err := r.pool.Exec(ctx, query,
		task.ID,
		task.WorkspaceID,
		task.ParentID,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.AssignedTo,
		task.CreatedBy,
		task.DependsOn,
		task.BranchName,
		task.EstimatedHours,
		task.Tags,
		task.Metadata,
		task.CreatedAt,
		task.UpdatedAt,
		task.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("task create: %w", err)
	}
	return nil
}

// GetByID retrieves a task by its ID.
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	query := fmt.Sprintf(`SELECT %s FROM tasks WHERE id = $1`, taskColumns())

	t, err := scanTask(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task not found: %w", err)
		}
		return nil, fmt.Errorf("task get by id: %w", err)
	}
	return t, nil
}

// Update updates an existing task in the database.
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	query := `
		UPDATE tasks
		SET parent_id = $1, title = $2, description = $3, status = $4, priority = $5,
		    assigned_to = $6, depends_on = $7, branch_name = $8, estimated_hours = $9,
		    tags = $10, metadata = $11, updated_at = $12, completed_at = $13
		WHERE id = $14`

	result, err := r.pool.Exec(ctx, query,
		task.ParentID,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.AssignedTo,
		task.DependsOn,
		task.BranchName,
		task.EstimatedHours,
		task.Tags,
		task.Metadata,
		task.UpdatedAt,
		task.CompletedAt,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("task update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task update: no rows affected")
	}
	return nil
}

// Delete removes a task from the database.
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("task delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task delete: no rows affected")
	}
	return nil
}

// ListByWorkspace retrieves tasks for a workspace with optional filters.
// Returns the task list, total count, and any error.
func (r *TaskRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, filters TaskFilters) ([]models.Task, int, error) {
	// Build the WHERE clause dynamically based on filters.
	conditions := []string{"workspace_id = $1"}
	args := []interface{}{workspaceID}
	argIdx := 2

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argIdx))
		args = append(args, *filters.AssignedTo)
		argIdx++
	}
	if filters.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *filters.Priority)
		argIdx++
	}
	if len(filters.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, filters.Tags)
		argIdx++
	}
	if filters.ParentID != nil {
		conditions = append(conditions, fmt.Sprintf("parent_id = $%d", argIdx))
		args = append(args, *filters.ParentID)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total matching rows.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("task list count: %w", err)
	}

	// Build the data query with pagination.
	dataQuery := fmt.Sprintf(`SELECT %s FROM tasks WHERE %s ORDER BY priority DESC, created_at DESC`,
		taskColumns(), whereClause)

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
		return nil, 0, fmt.Errorf("task list query: %w", err)
	}
	defer rows.Close()

	tasks, err := scanTaskRows(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("task list scan: %w", err)
	}
	return tasks, total, nil
}

// GetBoard retrieves all tasks for a workspace and groups them by status
// into a TaskBoardResponse.
func (r *TaskRepository) GetBoard(ctx context.Context, workspaceID uuid.UUID) (*models.TaskBoardResponse, error) {
	query := fmt.Sprintf(`SELECT %s FROM tasks WHERE workspace_id = $1 ORDER BY priority DESC, created_at ASC`,
		taskColumns())

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("task get board: %w", err)
	}
	defer rows.Close()

	tasks, err := scanTaskRows(rows)
	if err != nil {
		return nil, fmt.Errorf("task get board scan: %w", err)
	}

	return models.GroupTasksByStatus(workspaceID, tasks), nil
}

// GetDependencies retrieves the tasks that a given task depends on.
func (r *TaskRepository) GetDependencies(ctx context.Context, taskID uuid.UUID) ([]models.Task, error) {
	// First get the dependency IDs from the task itself.
	var depIDs []uuid.UUID
	idQuery := `SELECT depends_on FROM tasks WHERE id = $1`
	if err := r.pool.QueryRow(ctx, idQuery, taskID).Scan(&depIDs); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task not found: %w", err)
		}
		return nil, fmt.Errorf("task get dependencies ids: %w", err)
	}

	if len(depIDs) == 0 {
		return []models.Task{}, nil
	}

	// Fetch all dependency tasks using ANY.
	query := fmt.Sprintf(`SELECT %s FROM tasks WHERE id = ANY($1) ORDER BY created_at ASC`, taskColumns())

	rows, err := r.pool.Query(ctx, query, depIDs)
	if err != nil {
		return nil, fmt.Errorf("task get dependencies: %w", err)
	}
	defer rows.Close()

	tasks, err := scanTaskRows(rows)
	if err != nil {
		return nil, fmt.Errorf("task get dependencies scan: %w", err)
	}
	return tasks, nil
}

// GetDependents retrieves tasks that depend on the given task.
func (r *TaskRepository) GetDependents(ctx context.Context, taskID uuid.UUID) ([]models.Task, error) {
	query := fmt.Sprintf(`SELECT %s FROM tasks WHERE $1 = ANY(depends_on) ORDER BY created_at ASC`, taskColumns())

	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("task get dependents: %w", err)
	}
	defer rows.Close()

	tasks, err := scanTaskRows(rows)
	if err != nil {
		return nil, fmt.Errorf("task get dependents scan: %w", err)
	}
	return tasks, nil
}
