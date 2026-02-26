// ─── Core Models ─────────────────────────────────────────────────────────────

export interface Workspace {
  id: string
  name: string
  description: string
  owner_id: string
  settings: WorkspaceSettings
  created_at: string
  updated_at: string
}

export interface WorkspaceSettings {
  max_agents: number
  sync_interval: number
  conflict_strategy: 'last_write_wins' | 'manual' | 'merge'
  allowed_artifact_types: string[]
}

export interface Agent {
  id: string
  workspace_id: string
  name: string
  type: 'claude' | 'gpt' | 'human' | 'custom'
  status: AgentStatus
  capabilities: string[]
  metadata: Record<string, string>
  last_heartbeat: string
  created_at: string
  updated_at: string
}

export type AgentStatus = 'online' | 'offline' | 'busy' | 'idle'

export interface Task {
  id: string
  workspace_id: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  assignee_id: string | null
  creator_id: string
  parent_id: string | null
  dependencies: string[]
  tags: string[]
  artifacts: string[]
  due_date: string | null
  started_at: string | null
  completed_at: string | null
  created_at: string
  updated_at: string
}

export type TaskStatus =
  | 'pending'
  | 'assigned'
  | 'in_progress'
  | 'review'
  | 'blocked'
  | 'completed'

export type TaskPriority = 'critical' | 'high' | 'medium' | 'low'

export interface Message {
  id: string
  workspace_id: string
  sender_id: string
  recipient_id: string | null
  thread_id: string | null
  message_type: MessageType
  content: string
  metadata: Record<string, string>
  read_by: string[]
  created_at: string
}

export type MessageType =
  | 'chat'
  | 'schema'
  | 'endpoint'
  | 'blocker'
  | 'review'
  | 'decision'
  | 'status_update'
  | 'code_snippet'
  | 'question'
  | 'answer'

export interface Artifact {
  id: string
  workspace_id: string
  name: string
  type: ArtifactType
  language: string
  content: string
  version: number
  creator_id: string
  tags: string[]
  dependencies: string[]
  metadata: Record<string, string>
  created_at: string
  updated_at: string
}

export type ArtifactType =
  | 'code'
  | 'config'
  | 'schema'
  | 'document'
  | 'test'
  | 'api_spec'
  | 'migration'

export interface Context {
  id: string
  workspace_id: string
  key: string
  value: string
  scope: 'global' | 'agent' | 'task'
  scope_id: string | null
  version: number
  updated_by: string
  created_at: string
  updated_at: string
}

export interface SyncLogEntry {
  id: string
  workspace_id: string
  agent_id: string
  action: 'create' | 'update' | 'delete' | 'resolve'
  resource_type: 'task' | 'artifact' | 'context' | 'message'
  resource_id: string
  version: number
  data: string
  conflict: boolean
  resolved: boolean
  created_at: string
}

// ─── API Types ───────────────────────────────────────────────────────────────

export interface ApiResponse<T> {
  data: T
  error?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  per_page: number
}

export interface CreateWorkspaceRequest {
  name: string
  description: string
  settings?: Partial<WorkspaceSettings>
}

export interface JoinWorkspaceRequest {
  workspace_id: string
  agent_name: string
  agent_type: Agent['type']
  capabilities?: string[]
}

export interface JoinWorkspaceResponse {
  token: string
  agent: Agent
  workspace: Workspace
}

export interface CreateTaskRequest {
  title: string
  description: string
  priority: TaskPriority
  assignee_id?: string
  parent_id?: string
  dependencies?: string[]
  tags?: string[]
  due_date?: string
}

export interface UpdateTaskRequest {
  title?: string
  description?: string
  status?: TaskStatus
  priority?: TaskPriority
  assignee_id?: string | null
  tags?: string[]
  due_date?: string | null
}

export interface SendMessageRequest {
  recipient_id?: string
  thread_id?: string
  message_type: MessageType
  content: string
  metadata?: Record<string, string>
}

export interface CreateArtifactRequest {
  name: string
  type: ArtifactType
  language: string
  content: string
  tags?: string[]
  dependencies?: string[]
  metadata?: Record<string, string>
}

export interface UpdateArtifactRequest {
  content: string
  metadata?: Record<string, string>
}

// ─── WebSocket Types ─────────────────────────────────────────────────────────

export interface WsEvent {
  type: WsEventType
  payload: unknown
  timestamp: string
}

export type WsEventType =
  | 'agent_online'
  | 'agent_offline'
  | 'agent_status_changed'
  | 'task_created'
  | 'task_updated'
  | 'task_deleted'
  | 'message_received'
  | 'artifact_created'
  | 'artifact_updated'
  | 'context_updated'
  | 'sync_conflict'
  | 'heartbeat'

export interface WsAgentEvent {
  agent_id: string
  status: AgentStatus
  agent?: Agent
}

export interface WsTaskEvent {
  task: Task
}

export interface WsMessageEvent {
  message: Message
}

export interface WsArtifactEvent {
  artifact: Artifact
}

export interface WsContextEvent {
  context: Context
}

export interface WsConflictEvent {
  resource_type: string
  resource_id: string
  local_version: number
  remote_version: number
  local_data: string
  remote_data: string
}

// ─── Board Types ─────────────────────────────────────────────────────────────

export interface TaskBoard {
  pending: Task[]
  assigned: Task[]
  in_progress: Task[]
  review: Task[]
  blocked: Task[]
  completed: Task[]
}

export interface TaskColumn {
  id: TaskStatus
  title: string
  tasks: Task[]
  color: string
}

// ─── UI State Types ──────────────────────────────────────────────────────────

export interface MessageThread {
  thread_id: string
  participants: string[]
  last_message: Message
  unread_count: number
  messages: Message[]
}

export interface ArtifactVersion {
  version: number
  content: string
  updated_by: string
  updated_at: string
}

export interface ConflictResolution {
  resource_type: string
  resource_id: string
  chosen: 'local' | 'remote' | 'merged'
  merged_data?: string
}
