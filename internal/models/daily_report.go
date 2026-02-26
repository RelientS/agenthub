package models

import (
	"time"

	"github.com/google/uuid"
)

// DailyReport represents a daily workspace summary report.
type DailyReport struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	WorkspaceID    uuid.UUID              `json:"workspace_id" db:"workspace_id"`
	ReportDate     time.Time              `json:"report_date" db:"report_date"`
	GeneratedBy    uuid.UUID              `json:"generated_by" db:"generated_by"`
	Summary        string                 `json:"summary" db:"summary"`
	TasksCompleted int                    `json:"tasks_completed" db:"tasks_completed"`
	TasksCreated   int                    `json:"tasks_created" db:"tasks_created"`
	TasksBlocked   int                    `json:"tasks_blocked" db:"tasks_blocked"`
	ActiveAgents   int                    `json:"active_agents" db:"active_agents"`
	Highlights     []string               `json:"highlights" db:"highlights"`
	Blockers       []string               `json:"blockers" db:"blockers"`
	Metrics        map[string]interface{} `json:"metrics" db:"metrics"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// NewDailyReport creates a new DailyReport with default values.
func NewDailyReport(workspaceID uuid.UUID, reportDate time.Time, generatedBy uuid.UUID, summary string) *DailyReport {
	return &DailyReport{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		ReportDate:  reportDate,
		GeneratedBy: generatedBy,
		Summary:     summary,
		Highlights:  []string{},
		Blockers:    []string{},
		Metrics:     make(map[string]interface{}),
		CreatedAt:   time.Now().UTC(),
	}
}
