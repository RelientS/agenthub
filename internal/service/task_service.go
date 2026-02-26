package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/pkg/events"
	"github.com/agenthub/server/internal/repository"
	"github.com/google/uuid"
)

// ---------- request / response types ----------

// CreateTaskInput holds the input for creating a task.
type CreateTaskInput struct {
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Priority       int                    `json:"priority"`
	ParentID       *uuid.UUID             `json:"parent_id,omitempty"`
	AssignedTo     *uuid.UUID             `json:"assigned_to,omitempty"`
	DependsOn      []uuid.UUID            `json:"depends_on,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	EstimatedHours float64                `json:"estimated_hours,omitempty"`
	BranchName     string                 `json:"branch_name,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTaskInput holds the input for updating a task.
type UpdateTaskInput struct {
	Title          *string                `json:"title,omitempty"`
	Description    *string                `json:"description,omitempty"`
	Status         *string                `json:"status,omitempty"`
	Priority       *int                   `json:"priority,omitempty"`
	AssignedTo     *uuid.UUID             `json:"assigned_to,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	EstimatedHours *float64               `json:"estimated_hours,omitempty"`
	BranchName     *string                `json:"branch_name,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// CompleteTaskInput holds the input for completing a task.
type CompleteTaskInput struct {
	Result   string                 `json:"result,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TaskListFilter holds filter parameters for listing tasks.
type TaskListFilter struct {
	Status     string
	AssignedTo *uuid.UUID
	Priority   *int
	Tags       []string
	ParentID   *uuid.UUID
	Limit      int
	Offset     int
}

// ---------- service ----------

// TaskService handles task business logic.
type TaskService struct {
	taskRepo *repository.TaskRepository
	syncRepo *repository.SyncRepository
	eventBus *events.Bus
}

// NewTaskService creates a new TaskService.
func NewTaskService(
	tr *repository.TaskRepository,
	sr *repository.SyncRepository,
	eb *events.Bus,
) *TaskService {
	return &TaskService{
		taskRepo: tr,
		syncRepo: sr,
		eventBus: eb,
	}
}

// CreateTask creates a new task in a workspace.
func (s *TaskService) CreateTask(ctx context.Context, workspaceID uuid.UUID, input CreateTaskInput, createdBy uuid.UUID) (*models.Task, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}
	if !models.IsValidPriority(input.Priority) {
		return nil, fmt.Errorf("invalid priority %d: must be between 1 and 5", input.Priority)
	}

	now := time.Now().UTC()
	task := &models.Task{
		ID:             uuid.New(),
		WorkspaceID:    workspaceID,
		ParentID:       input.ParentID,
		Title:          input.Title,
		Description:    input.Description,
		Status:         models.TaskStatusPending,
		Priority:       input.Priority,
		AssignedTo:     input.AssignedTo,
		CreatedBy:      createdBy,
		DependsOn:      input.DependsOn,
		BranchName:     input.BranchName,
		EstimatedHours: input.EstimatedHours,
		Tags:           input.Tags,
		Metadata:       input.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if task.DependsOn == nil {
		task.DependsOn = []uuid.UUID{}
	}
	if task.Tags == nil {
		task.Tags = []string{}
	}
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}

	s.logChange(ctx, workspaceID, models.SyncEntityTask, task.ID, createdBy, models.SyncActionCreate, task)

	s.eventBus.Publish(events.NewEvent(events.EventTaskCreated, workspaceID.String(), map[string]interface{}{
		"task_id": task.ID,
		"title":   task.Title,
	}))

	return task, nil
}

// GetTask retrieves a task by ID. The workspaceID is used for authorization scoping.
func (s *TaskService) GetTask(ctx context.Context, workspaceID uuid.UUID, taskID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	if task.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("task does not belong to this workspace")
	}
	return task, nil
}

// UpdateTask applies partial updates to a task. If a status change is included,
// the state machine is validated.
func (s *TaskService) UpdateTask(ctx context.Context, workspaceID uuid.UUID, taskID uuid.UUID, input UpdateTaskInput, agentID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	if task.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("task does not belong to this workspace")
	}

	if input.Title != nil {
		if *input.Title == "" {
			return nil, fmt.Errorf("task title cannot be empty")
		}
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Status != nil {
		if err := validateStatusTransition(task.Status, *input.Status); err != nil {
			return nil, fmt.Errorf("status transition: %w", err)
		}
		task.Status = *input.Status
		if *input.Status == models.TaskStatusCompleted {
			now := time.Now().UTC()
			task.CompletedAt = &now
		}
	}
	if input.Priority != nil {
		if !models.IsValidPriority(*input.Priority) {
			return nil, fmt.Errorf("invalid priority %d: must be between 1 and 5", *input.Priority)
		}
		task.Priority = *input.Priority
	}
	if input.AssignedTo != nil {
		task.AssignedTo = input.AssignedTo
	}
	if input.Tags != nil {
		task.Tags = input.Tags
	}
	if input.EstimatedHours != nil {
		task.EstimatedHours = *input.EstimatedHours
	}
	if input.BranchName != nil {
		task.BranchName = *input.BranchName
	}
	if input.Metadata != nil {
		for k, v := range input.Metadata {
			task.Metadata[k] = v
		}
	}
	task.UpdatedAt = time.Now().UTC()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("updating task: %w", err)
	}

	s.logChange(ctx, workspaceID, models.SyncEntityTask, task.ID, agentID, models.SyncActionUpdate, task)

	s.eventBus.Publish(events.NewEvent(events.EventTaskUpdated, workspaceID.String(), map[string]interface{}{
		"task_id": task.ID,
		"status":  task.Status,
	}))

	return task, nil
}

// DeleteTask removes a task by ID.
func (s *TaskService) DeleteTask(ctx context.Context, workspaceID uuid.UUID, taskID uuid.UUID, agentID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("getting task: %w", err)
	}
	if task.WorkspaceID != workspaceID {
		return fmt.Errorf("task does not belong to this workspace")
	}

	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}

	s.logChange(ctx, workspaceID, models.SyncEntityTask, taskID, agentID, models.SyncActionDelete, nil)

	return nil
}

// ClaimTask assigns a task to an agent. The task must be in a status that allows
// transition to assigned, and all dependencies must be completed.
func (s *TaskService) ClaimTask(ctx context.Context, workspaceID uuid.UUID, taskID uuid.UUID, agentID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	if task.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("task does not belong to this workspace")
	}

	if err := validateStatusTransition(task.Status, models.TaskStatusAssigned); err != nil {
		return nil, fmt.Errorf("cannot claim task: %w", err)
	}

	depsMet, err := s.checkDependenciesMet(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("checking dependencies: %w", err)
	}
	if !depsMet {
		return nil, fmt.Errorf("cannot claim task: not all dependencies are completed")
	}

	task.AssignedTo = &agentID
	task.Status = models.TaskStatusAssigned
	task.UpdatedAt = time.Now().UTC()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("claiming task: %w", err)
	}

	s.logChange(ctx, workspaceID, models.SyncEntityTask, task.ID, agentID, models.SyncActionUpdate, task)

	s.eventBus.Publish(events.NewEvent(events.EventTaskAssigned, workspaceID.String(), map[string]interface{}{
		"task_id":  task.ID,
		"agent_id": agentID,
	}))

	return task, nil
}

// CompleteTask marks a task as completed and checks whether dependent tasks
// can now proceed.
func (s *TaskService) CompleteTask(ctx context.Context, workspaceID uuid.UUID, taskID uuid.UUID, agentID uuid.UUID, input CompleteTaskInput) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	if task.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("task does not belong to this workspace")
	}

	if err := validateStatusTransition(task.Status, models.TaskStatusCompleted); err != nil {
		return nil, fmt.Errorf("cannot complete task: %w", err)
	}

	now := time.Now().UTC()
	task.Status = models.TaskStatusCompleted
	task.CompletedAt = &now
	task.UpdatedAt = now
	if input.Result != "" {
		task.Metadata["result"] = input.Result
	}
	if input.Metadata != nil {
		for k, v := range input.Metadata {
			task.Metadata[k] = v
		}
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("completing task: %w", err)
	}

	s.logChange(ctx, workspaceID, models.SyncEntityTask, task.ID, agentID, models.SyncActionUpdate, task)

	s.eventBus.Publish(events.NewEvent(events.EventTaskUpdated, workspaceID.String(), map[string]interface{}{
		"task_id": task.ID,
		"status":  models.TaskStatusCompleted,
	}))

	// Notify dependent tasks that dependencies may now be met.
	dependents, depErr := s.taskRepo.GetDependents(ctx, taskID)
	if depErr == nil {
		for i := range dependents {
			met, checkErr := s.checkDependenciesMet(ctx, &dependents[i])
			if checkErr == nil && met {
				s.eventBus.Publish(events.NewEvent(events.EventTaskUpdated, workspaceID.String(), map[string]interface{}{
					"task_id":          dependents[i].ID,
					"dependencies_met": true,
				}))
			}
		}
	}

	return task, nil
}

// BlockTask marks a task as blocked with a reason stored in metadata.
func (s *TaskService) BlockTask(ctx context.Context, workspaceID uuid.UUID, taskID uuid.UUID, agentID uuid.UUID, reason string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	if task.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("task does not belong to this workspace")
	}

	if err := validateStatusTransition(task.Status, models.TaskStatusBlocked); err != nil {
		return nil, fmt.Errorf("cannot block task: %w", err)
	}

	now := time.Now().UTC()
	task.Status = models.TaskStatusBlocked
	task.Metadata["blocked_reason"] = reason
	task.Metadata["blocked_by"] = agentID.String()
	task.Metadata["blocked_at"] = now.Format(time.RFC3339)
	task.UpdatedAt = now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("blocking task: %w", err)
	}

	s.logChange(ctx, workspaceID, models.SyncEntityTask, task.ID, agentID, models.SyncActionUpdate, task)

	s.eventBus.Publish(events.NewEvent(events.EventTaskUpdated, workspaceID.String(), map[string]interface{}{
		"task_id": task.ID,
		"status":  models.TaskStatusBlocked,
		"reason":  reason,
	}))

	return task, nil
}

// GetBoard returns a kanban board view of all tasks in a workspace grouped by status.
func (s *TaskService) GetBoard(ctx context.Context, workspaceID uuid.UUID) (*models.TaskBoardResponse, error) {
	tasks, _, err := s.taskRepo.ListByWorkspace(ctx, workspaceID, repository.TaskFilters{})
	if err != nil {
		return nil, fmt.Errorf("listing tasks for board: %w", err)
	}

	board := models.GroupTasksByStatus(workspaceID, tasks)
	return board, nil
}

// ListTasks returns tasks matching the given filter criteria.
func (s *TaskService) ListTasks(ctx context.Context, workspaceID uuid.UUID, filter TaskListFilter) ([]models.Task, error) {
	repoFilter := repository.TaskFilters{
		Status:     filter.Status,
		AssignedTo: filter.AssignedTo,
		Priority:   filter.Priority,
		Tags:       filter.Tags,
		ParentID:   filter.ParentID,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}

	tasks, _, err := s.taskRepo.ListByWorkspace(ctx, workspaceID, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	return tasks, nil
}

// ValidateStatusTransition checks whether transitioning from currentStatus
// to targetStatus is allowed by the state machine.
func (s *TaskService) ValidateStatusTransition(currentStatus, targetStatus string) error {
	return validateStatusTransition(currentStatus, targetStatus)
}

// CheckDependenciesMet verifies that all tasks in a task's DependsOn list
// have status completed.
func (s *TaskService) CheckDependenciesMet(ctx context.Context, task *models.Task) (bool, error) {
	return s.checkDependenciesMet(ctx, task)
}

// ---------- helpers ----------

// validateStatusTransition is a package-level helper for state machine validation.
func validateStatusTransition(currentStatus, targetStatus string) error {
	allowed, exists := models.TaskStatusTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("unknown current status: %s", currentStatus)
	}

	for _, s := range allowed {
		if s == targetStatus {
			return nil
		}
	}

	return fmt.Errorf("transition from %q to %q is not allowed", currentStatus, targetStatus)
}

// checkDependenciesMet verifies all dependency tasks are completed.
func (s *TaskService) checkDependenciesMet(ctx context.Context, task *models.Task) (bool, error) {
	if len(task.DependsOn) == 0 {
		return true, nil
	}

	for _, depID := range task.DependsOn {
		depTask, err := s.taskRepo.GetByID(ctx, depID)
		if err != nil {
			return false, fmt.Errorf("getting dependency task %s: %w", depID, err)
		}
		if depTask.Status != models.TaskStatusCompleted {
			return false, nil
		}
	}
	return true, nil
}

// logChange marshals the payload and writes a sync log entry.
func (s *TaskService) logChange(ctx context.Context, workspaceID uuid.UUID, entityType string, entityID, agentID uuid.UUID, action string, payload interface{}) {
	data, _ := json.Marshal(payload)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	entry := models.NewSyncLogEntry(workspaceID, entityType, entityID, agentID, action, hash)
	_ = s.syncRepo.LogChange(ctx, entry)
}
