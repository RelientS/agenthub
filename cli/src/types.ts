// ============================================================
// AgentHub CLI - Type Definitions
// Mirrors the Go server models for full API compatibility.
// ============================================================

// -----------------------------------------------------------
// API Response Envelope
// -----------------------------------------------------------

export interface APIResponse<T = unknown> {
  data: T;
  error: APIError | null;
}

export interface APIError {
  code: string;
  message: string;
}

// -----------------------------------------------------------
// Workspace
// -----------------------------------------------------------

export interface Workspace {
  id: string;
  name: string;
  description?: string;
  owner_agent_id: string;
  invite_code: string;
  status: WorkspaceStatus;
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export type WorkspaceStatus = 'active' | 'archived' | 'deleted';

export interface CreateWorkspaceRequest {
  name: string;
  description?: string;
  agent_name: string;
  agent_role: AgentRole;
  agent_capabilities?: string[];
}

export interface CreateWorkspaceResponse {
  workspace: Workspace;
  agent: Agent;
  token: string;
}

export interface JoinWorkspaceRequest {
  invite_code: string;
  agent_name: string;
  agent_role: AgentRole;
  agent_capabilities?: string[];
}

export interface JoinWorkspaceResponse {
  workspace: Workspace;
  agent: Agent;
  token: string;
}

// -----------------------------------------------------------
// Agent
// -----------------------------------------------------------

export interface Agent {
  id: string;
  workspace_id: string;
  name: string;
  role: AgentRole;
  status: AgentStatus;
  capabilities: string[];
  last_heartbeat: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export type AgentRole = 'frontend' | 'backend' | 'fullstack' | 'tester' | 'devops';

export type AgentStatus = 'online' | 'offline' | 'busy';

// -----------------------------------------------------------
// Task
// -----------------------------------------------------------

export interface Task {
  id: string;
  workspace_id: string;
  title: string;
  description?: string;
  status: TaskStatus;
  priority: TaskPriority;
  assigned_to?: string;
  created_by: string;
  parent_id?: string;
  tags?: string[];
  dependencies?: string[];
  metadata?: Record<string, unknown>;
  due_date?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export type TaskStatus =
  | 'pending'
  | 'assigned'
  | 'in_progress'
  | 'blocked'
  | 'review'
  | 'done'
  | 'cancelled';

export type TaskPriority = 'low' | 'medium' | 'high' | 'critical';

export interface CreateTaskRequest {
  title: string;
  description?: string;
  priority?: TaskPriority;
  assigned_to?: string;
  parent_id?: string;
  tags?: string[];
  dependencies?: string[];
}

export interface UpdateTaskRequest {
  title?: string;
  description?: string;
  status?: TaskStatus;
  priority?: TaskPriority;
  assigned_to?: string;
  tags?: string[];
}

export interface TaskFilters {
  status?: TaskStatus;
  priority?: TaskPriority;
  assigned_to?: string;
  created_by?: string;
}

export interface TaskBoardColumn {
  status: string;
  tasks: Task[];
  count: number;
}

export interface TaskBoard {
  workspace_id: string;
  columns: TaskBoardColumn[];
  total_tasks: number;
}

// -----------------------------------------------------------
// Message
// -----------------------------------------------------------

export interface Message {
  id: string;
  workspace_id: string;
  from_agent_id: string;
  to_agent_id?: string;
  thread_id?: string;
  message_type: MessageType;
  payload: Record<string, unknown>;
  ref_task_id?: string;
  ref_artifact_id?: string;
  is_read: boolean;
  created_at: string;
}

export type MessageType =
  | 'request_schema'
  | 'provide_schema'
  | 'request_endpoint'
  | 'provide_endpoint'
  | 'report_blocker'
  | 'resolve_blocker'
  | 'request_review'
  | 'provide_review'
  | 'status_update'
  | 'question'
  | 'answer'
  | 'notification';

export interface SendMessageRequest {
  to_agent_id?: string;
  thread_id?: string;
  message_type: MessageType;
  payload: Record<string, unknown>;
  ref_task_id?: string;
  ref_artifact_id?: string;
}

export interface MessageFilters {
  from_agent_id?: string;
  to_agent_id?: string;
  message_type?: MessageType;
  is_read?: boolean;
  thread_id?: string;
}

// -----------------------------------------------------------
// Artifact
// -----------------------------------------------------------

export interface Artifact {
  id: string;
  workspace_id: string;
  created_by: string;
  artifact_type: ArtifactType;
  name: string;
  description?: string;
  content: string;
  content_hash: string;
  version: number;
  parent_version?: number;
  file_path?: string;
  language?: string;
  tags?: string[];
  metadata?: Record<string, unknown>;
  created_at: string;
}

export type ArtifactType =
  | 'code_snippet'
  | 'api_schema'
  | 'type_definition'
  | 'test_result'
  | 'migration'
  | 'config'
  | 'doc';

export interface CreateArtifactRequest {
  artifact_type: ArtifactType;
  name: string;
  description?: string;
  content: string;
  file_path?: string;
  language?: string;
  tags?: string[];
}

export interface ArtifactFilters {
  artifact_type?: ArtifactType;
  created_by?: string;
  name?: string;
}

// -----------------------------------------------------------
// Context
// -----------------------------------------------------------

export interface Context {
  id: string;
  workspace_id: string;
  context_type: ContextType;
  title: string;
  content: string;
  content_hash: string;
  version: number;
  updated_by: string;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

export type ContextType =
  | 'prd'
  | 'design_doc'
  | 'api_contract'
  | 'architecture'
  | 'shared_types'
  | 'env_config'
  | 'convention';

export interface CreateContextRequest {
  context_type: ContextType;
  title: string;
  content: string;
  tags?: string[];
}

export interface UpdateContextRequest {
  title?: string;
  content?: string;
  tags?: string[];
}

// -----------------------------------------------------------
// Sync
// -----------------------------------------------------------

export interface SyncChange {
  entity_type: SyncEntityType;
  entity_id: string;
  operation: SyncOperation;
  payload: string;
}

export type SyncEntityType = 'task' | 'message' | 'artifact' | 'context' | 'agent';

export type SyncOperation = 'create' | 'update' | 'delete';

export interface SyncLogEntry {
  id: number;
  workspace_id: string;
  entity_type: SyncEntityType;
  entity_id: string;
  operation: SyncOperation;
  agent_id: string;
  payload: string;
  created_at: string;
}

export interface SyncPushResponse {
  sync_id: number;
  accepted: number;
  conflicts: SyncConflict[];
}

export interface SyncConflict {
  entity_type: string;
  entity_id: string;
  message: string;
}

export interface SyncPullResponse {
  changes: SyncLogEntry[];
  last_sync_id: number;
  has_more: boolean;
}

export interface SyncStatusResponse {
  last_sync_id: number;
  pending_changes: number;
  connected_agents: number;
}

// -----------------------------------------------------------
// WebSocket
// -----------------------------------------------------------

export interface WSMessageEnvelope {
  type: string;
  workspace_id: string;
  agent_id?: string;
  payload: unknown;
}

// -----------------------------------------------------------
// Paginated list responses
// -----------------------------------------------------------

export interface PaginatedList<T> {
  items: T[];
  total: number;
}
