package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agenthub/server/internal/models"
	"github.com/agenthub/server/internal/repository"
	"github.com/google/uuid"
)

// CreateDailyReportInput holds input for creating a daily report.
type CreateDailyReportInput struct {
	ReportDate time.Time `json:"report_date"`
	Summary    string    `json:"summary"`
	Highlights []string  `json:"highlights,omitempty"`
	Blockers   []string  `json:"blockers,omitempty"`
}

// DailyReportService handles daily report business logic.
type DailyReportService struct {
	reportRepo *repository.DailyReportRepository
	taskRepo   *repository.TaskRepository
}

// NewDailyReportService creates a new DailyReportService.
func NewDailyReportService(
	rr *repository.DailyReportRepository,
	tr *repository.TaskRepository,
) *DailyReportService {
	return &DailyReportService{
		reportRepo: rr,
		taskRepo:   tr,
	}
}

// CreateReport creates a daily report with gathered metrics.
func (s *DailyReportService) CreateReport(ctx context.Context, workspaceID uuid.UUID, input CreateDailyReportInput, agentID uuid.UUID) (*models.DailyReport, error) {
	if input.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}

	reportDate := input.ReportDate
	if reportDate.IsZero() {
		reportDate = time.Now().UTC()
	}
	reportDate = time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, time.UTC)

	report := models.NewDailyReport(workspaceID, reportDate, agentID, input.Summary)

	// Gather metrics from the database.
	completed, err := s.reportRepo.CountTasksCompletedOnDate(ctx, workspaceID, reportDate)
	if err == nil {
		report.TasksCompleted = completed
	}

	created, err := s.reportRepo.CountTasksCreatedOnDate(ctx, workspaceID, reportDate)
	if err == nil {
		report.TasksCreated = created
	}

	blocked, err := s.reportRepo.CountBlockedTasks(ctx, workspaceID)
	if err == nil {
		report.TasksBlocked = blocked
	}

	active, err := s.reportRepo.CountActiveAgents(ctx, workspaceID)
	if err == nil {
		report.ActiveAgents = active
	}

	if input.Highlights != nil {
		report.Highlights = input.Highlights
	}
	if input.Blockers != nil {
		report.Blockers = input.Blockers
	}

	report.Metrics["velocity"] = completed
	report.Metrics["backlog_blocked"] = blocked

	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("creating daily report: %w", err)
	}

	return report, nil
}

// GetReport retrieves a daily report by ID.
func (s *DailyReportService) GetReport(ctx context.Context, workspaceID uuid.UUID, reportID uuid.UUID) (*models.DailyReport, error) {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("getting daily report: %w", err)
	}
	if report.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("daily report does not belong to this workspace")
	}
	return report, nil
}

// ListReports lists daily reports for a workspace.
func (s *DailyReportService) ListReports(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]models.DailyReport, error) {
	reports, err := s.reportRepo.ListByWorkspace(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing daily reports: %w", err)
	}
	return reports, nil
}

// GenerateSummary generates a daily summary by querying current workspace state.
func (s *DailyReportService) GenerateSummary(ctx context.Context, workspaceID uuid.UUID, agentID uuid.UUID) (*models.DailyReport, error) {
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	completed, _ := s.reportRepo.CountTasksCompletedOnDate(ctx, workspaceID, today)
	created, _ := s.reportRepo.CountTasksCreatedOnDate(ctx, workspaceID, today)
	blocked, _ := s.reportRepo.CountBlockedTasks(ctx, workspaceID)
	active, _ := s.reportRepo.CountActiveAgents(ctx, workspaceID)

	// Build a summary string from the gathered data.
	var parts []string
	parts = append(parts, fmt.Sprintf("%d task(s) completed today", completed))
	parts = append(parts, fmt.Sprintf("%d new task(s) created", created))
	if blocked > 0 {
		parts = append(parts, fmt.Sprintf("%d task(s) currently blocked", blocked))
	}
	parts = append(parts, fmt.Sprintf("%d active agent(s)", active))
	summary := strings.Join(parts, ". ") + "."

	// Gather blockers from blocked tasks.
	var blockers []string
	blockedTasks, _, _ := s.taskRepo.ListByWorkspace(ctx, workspaceID, repository.TaskFilters{
		Status: models.TaskStatusBlocked,
		Limit:  10,
	})
	for _, t := range blockedTasks {
		reason := ""
		if r, ok := t.Metadata["blocked_reason"]; ok {
			reason = fmt.Sprintf(": %v", r)
		}
		blockers = append(blockers, t.Title+reason)
	}

	// Gather highlights from recently completed tasks.
	var highlights []string
	completedTasks, _, _ := s.taskRepo.ListByWorkspace(ctx, workspaceID, repository.TaskFilters{
		Status: models.TaskStatusCompleted,
		Limit:  10,
	})
	for _, t := range completedTasks {
		if t.CompletedAt != nil && t.CompletedAt.After(today) {
			highlights = append(highlights, t.Title)
		}
	}

	report := models.NewDailyReport(workspaceID, today, agentID, summary)
	report.TasksCompleted = completed
	report.TasksCreated = created
	report.TasksBlocked = blocked
	report.ActiveAgents = active
	report.Highlights = highlights
	report.Blockers = blockers
	report.Metrics["velocity"] = completed
	report.Metrics["backlog_blocked"] = blocked

	if report.Highlights == nil {
		report.Highlights = []string{}
	}
	if report.Blockers == nil {
		report.Blockers = []string{}
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("creating generated daily report: %w", err)
	}

	return report, nil
}
