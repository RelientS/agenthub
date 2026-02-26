CREATE TABLE contexts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    context_type    VARCHAR(50) NOT NULL,
    title           VARCHAR(500) NOT NULL,
    content         TEXT,
    content_hash    VARCHAR(64),
    version         INTEGER NOT NULL DEFAULT 1,
    updated_by      UUID REFERENCES agents(id) ON DELETE SET NULL,
    tags            TEXT[] DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_contexts_workspace ON contexts(workspace_id);
CREATE INDEX idx_contexts_type ON contexts(workspace_id, context_type);
CREATE INDEX idx_contexts_title ON contexts(workspace_id, title);
CREATE INDEX idx_contexts_updated_by ON contexts(updated_by);
CREATE INDEX idx_contexts_tags ON contexts USING GIN(tags);
CREATE INDEX idx_contexts_content_hash ON contexts(content_hash);

CREATE TABLE sync_log (
    id              BIGSERIAL PRIMARY KEY,
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    entity_type     VARCHAR(50) NOT NULL,
    entity_id       UUID NOT NULL,
    action          VARCHAR(20) NOT NULL,
    agent_id        UUID REFERENCES agents(id) ON DELETE SET NULL,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload_hash    VARCHAR(64)
);
CREATE INDEX idx_sync_log_workspace ON sync_log(workspace_id);
CREATE INDEX idx_sync_log_entity ON sync_log(entity_type, entity_id);
CREATE INDEX idx_sync_log_agent ON sync_log(agent_id);
CREATE INDEX idx_sync_log_timestamp ON sync_log(workspace_id, timestamp);
CREATE INDEX idx_sync_log_action ON sync_log(action);
