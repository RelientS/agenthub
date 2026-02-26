package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/repository"
	"github.com/google/uuid"
)

// OrchestratorService provides higher-level orchestration across tasks,
// messaging, and context. It runs periodic background checks and exposes
// utility methods for dependency analysis and progress reporting.
type OrchestratorService struct {
	taskService      *TaskService
	messagingService *MessagingService
	contextService   *ContextService
	agentRepo        *repository.AgentRepository

	// Background goroutine management.
	checkInterval  time.Duration
	staleTaskHours int
	workspaceIDs   []uuid.UUID

	stopCh chan struct{}
	once   sync.Once
}

// NewOrchestratorService creates a new OrchestratorService.
func NewOrchestratorService(
	ts *TaskService,
	ms *MessagingService,
	cs *ContextService,
	ar *repository.AgentRepository,
	checkInterval time.Duration,
	staleTaskHours int,
) *OrchestratorService {
	return &OrchestratorService{
		taskService:      ts,
		messagingService: ms,
		contextService:   cs,
		agentRepo:        ar,
		checkInterval:    checkInterval,
		staleTaskHours:   staleTaskHours,
		stopCh:           make(chan struct{}),
	}
}

// SetWorkspaces configures which workspaces the orchestrator should monitor.
func (s *OrchestratorService) SetWorkspaces(workspaceIDs []uuid.UUID) {
	s.workspaceIDs = workspaceIDs
}

// Start launches a background goroutine that runs periodic checks at the
// configured interval. It checks for stale tasks and generates progress
// notifications.
func (s *OrchestratorService) Start() {
	go func() {
		ticker := time.NewTicker(s.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runPeriodicChecks()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Stop signals the background goroutine to exit. It is safe to call multiple
// times.
func (s *OrchestratorService) Stop() {
	s.once.Do(func() {
		close(s.stopCh)
	})
}

// CheckDependencies examines a completed task and finds all dependent tasks
// whose dependencies are now fully met. For each such task it broadcasts a
// notification to the workspace.
func (s *OrchestratorService) CheckDependencies(ctx context.Context, workspaceID uuid.UUID, completedTaskID uuid.UUID) error {
	completedTask, err := s.taskService.GetTask(ctx, workspaceID, completedTaskID)
	if err != nil {
		return fmt.Errorf("getting completed task: %w", err)
	}

	if completedTask.Status != models.TaskStatusCompleted {
		return fmt.Errorf("task %s is not completed", completedTaskID)
	}

	// Get all tasks in the workspace to find dependents.
	allTasks, err := s.taskService.ListTasks(ctx, workspaceID, TaskListFilter{Limit: 1000})
	if err != nil {
		return fmt.Errorf("listing tasks: %w", err)
	}

	for i := range allTasks {
		task := &allTasks[i]
		if task.Status == models.TaskStatusCompleted {
			continue
		}

		// Check if this task depends on the completed task.
		dependsOnCompleted := false
		for _, depID := range task.DependsOn {
			if depID == completedTaskID {
				dependsOnCompleted = true
				break
			}
		}
		if !dependsOnCompleted {
			continue
		}

		// Check if ALL dependencies are now met.
		met, err := s.taskService.CheckDependenciesMet(ctx, task)
		if err != nil {
			continue
		}

		if met {
			payload := map[string]interface{}{
				"event":   "dependencies_met",
				"task_id": task.ID,
				"title":   task.Title,
				"message": fmt.Sprintf("All dependencies for task %q are now completed. It is ready to be claimed.", task.Title),
			}
			_, _ = s.messagingService.BroadcastToWorkspace(ctx, workspaceID, completedTask.CreatedBy, payload)
		}
	}

	return nil
}

// GenerateProgressReport creates a markdown-formatted summary of all tasks in
// a workspace and broadcasts it as a notification message.
func (s *OrchestratorService) GenerateProgressReport(ctx context.Context, workspaceID uuid.UUID, requestedBy uuid.UUID) (string, error) {
	board, err := s.taskService.GetBoard(ctx, workspaceID)
	if err != nil {
		return "", fmt.Errorf("getting task board: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# Workspace Progress Report\n\n")

	totalTasks := len(board.Pending) + len(board.Assigned) + len(board.InProgress) +
		len(board.Review) + len(board.Blocked) + len(board.Completed)
	completedCount := len(board.Completed)

	if totalTasks > 0 {
		pct := float64(completedCount) / float64(totalTasks) * 100
		sb.WriteString(fmt.Sprintf("**Overall Progress:** %d/%d tasks completed (%.0f%%)\n\n", completedCount, totalTasks, pct))
	} else {
		sb.WriteString("**No tasks found in this workspace.**\n\n")
	}

	writeSection := func(title string, tasks []models.Task) {
		if len(tasks) == 0 {
			return
		}
		sb.WriteString(fmt.Sprintf("## %s (%d)\n", title, len(tasks)))
		for _, t := range tasks {
			assignee := "unassigned"
			if t.AssignedTo != nil {
				assignee = t.AssignedTo.String()
			}
			sb.WriteString(fmt.Sprintf("- [P%d] **%s** (assigned: %s)\n", t.Priority, t.Title, assignee))
		}
		sb.WriteString("\n")
	}

	writeSection("Blocked", board.Blocked)
	writeSection("In Progress", board.InProgress)
	writeSection("In Review", board.Review)
	writeSection("Assigned", board.Assigned)
	writeSection("Pending", board.Pending)
	writeSection("Completed", board.Completed)

	report := sb.String()

	// Broadcast the report to the workspace.
	_, _ = s.messagingService.BroadcastToWorkspace(ctx, workspaceID, requestedBy, map[string]interface{}{
		"event":  "progress_report",
		"report": report,
	})

	return report, nil
}

// CheckStaleTasks finds tasks that have been in progress for longer than the
// configured stale threshold and sends reminder notifications to their
// assignees.
func (s *OrchestratorService) CheckStaleTasks(ctx context.Context, workspaceID uuid.UUID) ([]models.Task, error) {
	allTasks, err := s.taskService.ListTasks(ctx, workspaceID, TaskListFilter{
		Status: models.TaskStatusInProgress,
		Limit:  500,
	})
	if err != nil {
		return nil, fmt.Errorf("listing in-progress tasks: %w", err)
	}

	staleCutoff := time.Now().UTC().Add(-time.Duration(s.staleTaskHours) * time.Hour)
	var staleTasks []models.Task

	for _, task := range allTasks {
		if task.UpdatedAt.Before(staleCutoff) {
			staleTasks = append(staleTasks, task)

			// Send a reminder notification if the task is assigned.
			if task.AssignedTo != nil {
				input := SendMessageInput{
					ToAgentID:   task.AssignedTo,
					MessageType: models.MsgTypeNotification,
					Payload: map[string]interface{}{
						"event":   "stale_task_reminder",
						"task_id": task.ID,
						"title":   task.Title,
						"message": fmt.Sprintf("Task %q has been in progress for over %d hours. Please update its status.", task.Title, s.staleTaskHours),
					},
				}

				_, _ = s.messagingService.SendMessage(ctx, workspaceID, input, task.CreatedBy)
			}
		}
	}

	return staleTasks, nil
}

// DecomposeTasksWebhook is a placeholder for integration with an external AI
// service that can decompose a high-level task description into subtasks.
// When implemented, it would call an external API and create the returned
// subtasks in the workspace.
func (s *OrchestratorService) DecomposeTasksWebhook(ctx context.Context, workspaceID uuid.UUID, parentTaskID uuid.UUID, agentID uuid.UUID) ([]models.Task, error) {
	parentTask, err := s.taskService.GetTask(ctx, workspaceID, parentTaskID)
	if err != nil {
		return nil, fmt.Errorf("getting parent task: %w", err)
	}

	// Placeholder: In a real implementation, this would call an external AI
	// service with the parent task's description and receive back a list of
	// subtask definitions.
	_ = parentTask

	return nil, fmt.Errorf("task decomposition webhook not yet configured")
}

// ---------- internal ----------

// runPeriodicChecks iterates over all configured workspaces and runs stale
// task detection.
func (s *OrchestratorService) runPeriodicChecks() {
	ctx := context.Background()
	for _, wsID := range s.workspaceIDs {
		_, _ = s.CheckStaleTasks(ctx, wsID)
	}
}
