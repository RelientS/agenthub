package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agenthub/server/internal/models"
)

// DailyReportRepository handles database operations for daily reports.
type DailyReportRepository struct {
	pool *pgxpool.Pool
}

// NewDailyReportRepository creates a new DailyReportRepository.
func NewDailyReportRepository(pool *pgxpool.Pool) *DailyReportRepository {
	return &DailyReportRepository{pool: pool}
}

func dailyReportColumns() string {
	return `id, workspace_id, report_date, generated_by, summary,
	        tasks_completed, tasks_created, tasks_blocked, active_agents,
	        highlights, blockers, metrics, created_at`
}

func scanDailyReport(row pgx.Row) (*models.DailyReport, error) {
	r := &models.DailyReport{}
	var highlightsJSON, blockersJSON, metricsJSON []byte
	err := row.Scan(
		&r.ID,
		&r.WorkspaceID,
		&r.ReportDate,
		&r.GeneratedBy,
		&r.Summary,
		&r.TasksCompleted,
		&r.TasksCreated,
		&r.TasksBlocked,
		&r.ActiveAgents,
		&highlightsJSON,
		&blockersJSON,
		&metricsJSON,
		&r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if highlightsJSON != nil {
		_ = json.Unmarshal(highlightsJSON, &r.Highlights)
	}
	if r.Highlights == nil {
		r.Highlights = []string{}
	}
	if blockersJSON != nil {
		_ = json.Unmarshal(blockersJSON, &r.Blockers)
	}
	if r.Blockers == nil {
		r.Blockers = []string{}
	}
	if metricsJSON != nil {
		_ = json.Unmarshal(metricsJSON, &r.Metrics)
	}
	if r.Metrics == nil {
		r.Metrics = make(map[string]interface{})
	}
	return r, nil
}

// Create inserts a new daily report.
func (r *DailyReportRepository) Create(ctx context.Context, report *models.DailyReport) error {
	highlightsJSON, _ := json.Marshal(report.Highlights)
	blockersJSON, _ := json.Marshal(report.Blockers)
	metricsJSON, _ := json.Marshal(report.Metrics)

	query := `
		INSERT INTO daily_reports (
			id, workspace_id, report_date, generated_by, summary,
			tasks_completed, tasks_created, tasks_blocked, active_agents,
			highlights, blockers, metrics, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.pool.Exec(ctx, query,
		report.ID,
		report.WorkspaceID,
		report.ReportDate,
		report.GeneratedBy,
		report.Summary,
		report.TasksCompleted,
		report.TasksCreated,
		report.TasksBlocked,
		report.ActiveAgents,
		highlightsJSON,
		blockersJSON,
		metricsJSON,
		report.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("daily report create: %w", err)
	}
	return nil
}

// GetByID retrieves a daily report by its ID.
func (r *DailyReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DailyReport, error) {
	query := fmt.Sprintf(`SELECT %s FROM daily_reports WHERE id = $1`, dailyReportColumns())
	report, err := scanDailyReport(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("daily report not found: %w", err)
		}
		return nil, fmt.Errorf("daily report get by id: %w", err)
	}
	return report, nil
}

// GetByDate retrieves a daily report for a specific workspace and date.
func (r *DailyReportRepository) GetByDate(ctx context.Context, workspaceID uuid.UUID, date time.Time) (*models.DailyReport, error) {
	query := fmt.Sprintf(`SELECT %s FROM daily_reports WHERE workspace_id = $1 AND report_date = $2`, dailyReportColumns())
	report, err := scanDailyReport(r.pool.QueryRow(ctx, query, workspaceID, date))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("daily report not found for date: %w", err)
		}
		return nil, fmt.Errorf("daily report get by date: %w", err)
	}
	return report, nil
}

// ListByWorkspace retrieves daily reports for a workspace, ordered by date descending.
func (r *DailyReportRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]models.DailyReport, error) {
	if limit <= 0 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`SELECT %s FROM daily_reports WHERE workspace_id = $1 ORDER BY report_date DESC LIMIT $2 OFFSET $3`,
		dailyReportColumns())

	rows, err := r.pool.Query(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("daily report list: %w", err)
	}
	defer rows.Close()

	var reports []models.DailyReport
	for rows.Next() {
		var report models.DailyReport
		var highlightsJSON, blockersJSON, metricsJSON []byte
		if err := rows.Scan(
			&report.ID,
			&report.WorkspaceID,
			&report.ReportDate,
			&report.GeneratedBy,
			&report.Summary,
			&report.TasksCompleted,
			&report.TasksCreated,
			&report.TasksBlocked,
			&report.ActiveAgents,
			&highlightsJSON,
			&blockersJSON,
			&metricsJSON,
			&report.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("daily report scan: %w", err)
		}
		if highlightsJSON != nil {
			_ = json.Unmarshal(highlightsJSON, &report.Highlights)
		}
		if report.Highlights == nil {
			report.Highlights = []string{}
		}
		if blockersJSON != nil {
			_ = json.Unmarshal(blockersJSON, &report.Blockers)
		}
		if report.Blockers == nil {
			report.Blockers = []string{}
		}
		if metricsJSON != nil {
			_ = json.Unmarshal(metricsJSON, &report.Metrics)
		}
		if report.Metrics == nil {
			report.Metrics = make(map[string]interface{})
		}
		reports = append(reports, report)
	}

	return reports, rows.Err()
}

// CountTasksCompletedOnDate counts tasks completed in a workspace on a given date.
func (r *DailyReportRepository) CountTasksCompletedOnDate(ctx context.Context, workspaceID uuid.UUID, date time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND status = 'completed'
	          AND completed_at >= $2 AND completed_at < $3`
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)
	var count int
	err := r.pool.QueryRow(ctx, query, workspaceID, startOfDay, endOfDay).Scan(&count)
	return count, err
}

// CountTasksCreatedOnDate counts tasks created in a workspace on a given date.
func (r *DailyReportRepository) CountTasksCreatedOnDate(ctx context.Context, workspaceID uuid.UUID, date time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE workspace_id = $1
	          AND created_at >= $2 AND created_at < $3`
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)
	var count int
	err := r.pool.QueryRow(ctx, query, workspaceID, startOfDay, endOfDay).Scan(&count)
	return count, err
}

// CountBlockedTasks counts currently blocked tasks in a workspace.
func (r *DailyReportRepository) CountBlockedTasks(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE workspace_id = $1 AND status = 'blocked'`
	var count int
	err := r.pool.QueryRow(ctx, query, workspaceID).Scan(&count)
	return count, err
}

// CountActiveAgents counts agents with recent heartbeat in a workspace.
func (r *DailyReportRepository) CountActiveAgents(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM agents WHERE workspace_id = $1 AND status IN ('online', 'busy')`
	var count int
	err := r.pool.QueryRow(ctx, query, workspaceID).Scan(&count)
	return count, err
}
