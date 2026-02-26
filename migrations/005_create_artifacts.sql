CREATE TABLE artifacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by      UUID NOT NULL REFERENCES agents(id),
    artifact_type   VARCHAR(50) NOT NULL,
    name            VARCHAR(500) NOT NULL,
    description     TEXT,
    content         TEXT,
    content_hash    VARCHAR(64),
    version         INTEGER NOT NULL DEFAULT 1,
    parent_version  UUID REFERENCES artifacts(id) ON DELETE SET NULL,
    file_path       VARCHAR(1000),
    language        VARCHAR(50),
    tags            TEXT[] DEFAULT '{}',
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_artifacts_workspace ON artifacts(workspace_id);
CREATE INDEX idx_artifacts_created_by ON artifacts(created_by);
CREATE INDEX idx_artifacts_type ON artifacts(artifact_type);
CREATE INDEX idx_artifacts_name ON artifacts(workspace_id, name);
CREATE INDEX idx_artifacts_file_path ON artifacts(file_path);
CREATE INDEX idx_artifacts_content_hash ON artifacts(content_hash);
CREATE INDEX idx_artifacts_parent_version ON artifacts(parent_version);
CREATE INDEX idx_artifacts_tags ON artifacts USING GIN(tags);
