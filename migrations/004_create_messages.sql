CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    from_agent_id   UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    to_agent_id     UUID REFERENCES agents(id) ON DELETE SET NULL,
    thread_id       UUID,
    message_type    VARCHAR(50) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    ref_task_id     UUID REFERENCES tasks(id) ON DELETE SET NULL,
    ref_artifact_id UUID,
    is_read         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_workspace ON messages(workspace_id);
CREATE INDEX idx_messages_to_agent ON messages(to_agent_id);
CREATE INDEX idx_messages_thread ON messages(thread_id);
CREATE INDEX idx_messages_type ON messages(message_type);
CREATE INDEX idx_messages_unread ON messages(to_agent_id, is_read) WHERE is_read = FALSE;
CREATE INDEX idx_messages_from_agent ON messages(from_agent_id);
CREATE INDEX idx_messages_ref_task ON messages(ref_task_id);
CREATE INDEX idx_messages_created_at ON messages(workspace_id, created_at);
