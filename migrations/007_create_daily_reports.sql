CREATE TABLE IF NOT EXISTS daily_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    report_date     DATE NOT NULL,
    generated_by    UUID NOT NULL REFERENCES agents(id),
    summary         TEXT NOT NULL,
    tasks_completed INTEGER NOT NULL DEFAULT 0,
    tasks_created   INTEGER NOT NULL DEFAULT 0,
    tasks_blocked   INTEGER NOT NULL DEFAULT 0,
    active_agents   INTEGER NOT NULL DEFAULT 0,
    highlights      JSONB DEFAULT '[]',
    blockers        JSONB DEFAULT '[]',
    metrics         JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, report_date)
);
CREATE INDEX IF NOT EXISTS idx_daily_reports_workspace ON daily_reports(workspace_id);
CREATE INDEX IF NOT EXISTS idx_daily_reports_date ON daily_reports(workspace_id, report_date DESC);
