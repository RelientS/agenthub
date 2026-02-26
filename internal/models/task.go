package models

import (
	"time"

	"github.com/google/uuid"
)

// Task status constants.
const (
	TaskStatusPending    = "pending"
	TaskStatusAssigned   = "assigned"
	TaskStatusInProgress = "in_progress"
	TaskStatusReview     = "review"
	TaskStatusBlocked    = "blocked"
	TaskStatusCompleted  = "completed"
)

// ValidTaskStatuses contains all valid task statuses for validation.
var ValidTaskStatuses = []string{
	TaskStatusPending,
	TaskStatusAssigned,
	TaskStatusInProgress,
	TaskStatusReview,
	TaskStatusBlocked,
	TaskStatusCompleted,
}

// TaskStatusTransitions defines the valid state transitions for task status.
// Key is the current status, value is a list of statuses it can transition to.
var TaskStatusTransitions = map[string][]string{
	TaskStatusPending:    {TaskStatusAssigned, TaskStatusCompleted},
	TaskStatusAssigned:   {TaskStatusInProgress, TaskStatusPending, TaskStatusCompleted},
	TaskStatusInProgress: {TaskStatusReview, TaskStatusBlocked, TaskStatusCompleted},
	TaskStatusReview:     {TaskStatusInProgress, TaskStatusCompleted},
	TaskStatusBlocked:    {TaskStatusInProgress, TaskStatusPending},
	TaskStatusCompleted:  {},
}

// Task represents a unit of work within a workspace.
type Task struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	WorkspaceID    uuid.UUID              `json:"workspace_id" db:"workspace_id"`
	ParentID       *uuid.UUID             `json:"parent_id,omitempty" db:"parent_id"`
	Title          string                 `json:"title" db:"title"`
	Description    string                 `json:"description,omitempty" db:"description"`
	Status         string                 `json:"status" db:"status"`
	Priority       int                    `json:"priority" db:"priority"`
	AssignedTo     *uuid.UUID             `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedBy      uuid.UUID              `json:"created_by" db:"created_by"`
	DependsOn      []uuid.UUID            `json:"depends_on,omitempty" db:"depends_on"`
	BranchName     string                 `json:"branch_name,omitempty" db:"branch_name"`
	EstimatedHours float64                `json:"estimated_hours,omitempty" db:"estimated_hours"`
	Tags           []string               `json:"tags,omitempty" db:"tags"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// TaskBoardResponse groups tasks by their status for task board display.
type TaskBoardResponse struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Pending     []Task    `json:"pending"`
	Assigned    []Task    `json:"assigned"`
	InProgress  []Task    `json:"in_progress"`
	Review      []Task    `json:"review"`
	Blocked     []Task    `json:"blocked"`
	Completed   []Task    `json:"completed"`
}

// NewTask creates a new Task with default values.
func NewTask(workspaceID uuid.UUID, title, description string, priority int, createdBy uuid.UUID) *Task {
	now := time.Now().UTC()
	return &Task{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Title:       title,
		Description: description,
		Status:      TaskStatusPending,
		Priority:    priority,
		CreatedBy:   createdBy,
		DependsOn:   []uuid.UUID{},
		Tags:        []string{},
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsValidTaskStatus checks whether the given status is a valid task status.
func IsValidTaskStatus(status string) bool {
	for _, s := range ValidTaskStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// CanTransitionTo checks whether a task can transition from its current status to the target status.
func (t *Task) CanTransitionTo(target string) bool {
	allowed, exists := TaskStatusTransitions[t.Status]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

// IsValidPriority checks whether the given priority is within the valid range (1-5).
func IsValidPriority(priority int) bool {
	return priority >= 1 && priority <= 5
}

// GroupTasksByStatus takes a slice of tasks and returns a TaskBoardResponse.
func GroupTasksByStatus(workspaceID uuid.UUID, tasks []Task) *TaskBoardResponse {
	board := &TaskBoardResponse{
		WorkspaceID: workspaceID,
		Pending:     []Task{},
		Assigned:    []Task{},
		InProgress:  []Task{},
		Review:      []Task{},
		Blocked:     []Task{},
		Completed:   []Task{},
	}

	for _, task := range tasks {
		switch task.Status {
		case TaskStatusPending:
			board.Pending = append(board.Pending, task)
		case TaskStatusAssigned:
			board.Assigned = append(board.Assigned, task)
		case TaskStatusInProgress:
			board.InProgress = append(board.InProgress, task)
		case TaskStatusReview:
			board.Review = append(board.Review, task)
		case TaskStatusBlocked:
			board.Blocked = append(board.Blocked, task)
		case TaskStatusCompleted:
			board.Completed = append(board.Completed, task)
		}
	}

	return board
}
