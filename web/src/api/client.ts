import axios from 'axios'
import type {
  ApiResponse,
  Workspace,
  Agent,
  Task,
  Message,
  Artifact,
  Context,
  CreateWorkspaceRequest,
  JoinWorkspaceRequest,
  JoinWorkspaceResponse,
  CreateTaskRequest,
  UpdateTaskRequest,
  SendMessageRequest,
  CreateArtifactRequest,
  UpdateArtifactRequest,
  SyncLogEntry,
} from '@/types'

const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('agenthub_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('agenthub_token')
      localStorage.removeItem('agenthub_agent_id')
      localStorage.removeItem('agenthub_workspace_id')
      window.location.href = '/join'
    }
    return Promise.reject(error)
  }
)

// ─── Workspace ───────────────────────────────────────────────────────────────

export async function createWorkspace(
  req: CreateWorkspaceRequest
): Promise<ApiResponse<Workspace>> {
  const { data } = await api.post('/workspaces', req)
  return data
}

export async function getWorkspace(id: string): Promise<ApiResponse<Workspace>> {
  const { data } = await api.get(`/workspaces/${id}`)
  return data
}

export async function joinWorkspace(
  req: JoinWorkspaceRequest
): Promise<ApiResponse<JoinWorkspaceResponse>> {
  const { data } = await api.post('/workspaces/join', req)
  return data
}

// ─── Agents ──────────────────────────────────────────────────────────────────

export async function getAgents(workspaceId: string): Promise<ApiResponse<Agent[]>> {
  const { data } = await api.get(`/workspaces/${workspaceId}/agents`)
  return data
}

export async function getAgent(
  workspaceId: string,
  agentId: string
): Promise<ApiResponse<Agent>> {
  const { data } = await api.get(`/workspaces/${workspaceId}/agents/${agentId}`)
  return data
}

export async function sendHeartbeat(
  workspaceId: string,
  agentId: string
): Promise<void> {
  await api.post(`/workspaces/${workspaceId}/agents/${agentId}/heartbeat`)
}

// ─── Tasks ───────────────────────────────────────────────────────────────────

export async function getTasks(workspaceId: string): Promise<ApiResponse<Task[]>> {
  const { data } = await api.get(`/workspaces/${workspaceId}/tasks`)
  return data
}

export async function getTask(
  workspaceId: string,
  taskId: string
): Promise<ApiResponse<Task>> {
  const { data } = await api.get(`/workspaces/${workspaceId}/tasks/${taskId}`)
  return data
}

export async function createTask(
  workspaceId: string,
  req: CreateTaskRequest
): Promise<ApiResponse<Task>> {
  const { data } = await api.post(`/workspaces/${workspaceId}/tasks`, req)
  return data
}

export async function updateTask(
  workspaceId: string,
  taskId: string,
  req: UpdateTaskRequest
): Promise<ApiResponse<Task>> {
  const { data } = await api.patch(`/workspaces/${workspaceId}/tasks/${taskId}`, req)
  return data
}

export async function deleteTask(
  workspaceId: string,
  taskId: string
): Promise<void> {
  await api.delete(`/workspaces/${workspaceId}/tasks/${taskId}`)
}

// ─── Messages ────────────────────────────────────────────────────────────────

export async function getMessages(
  workspaceId: string,
  threadId?: string
): Promise<ApiResponse<Message[]>> {
  const params = threadId ? { thread_id: threadId } : {}
  const { data } = await api.get(`/workspaces/${workspaceId}/messages`, { params })
  return data
}

export async function sendMessage(
  workspaceId: string,
  req: SendMessageRequest
): Promise<ApiResponse<Message>> {
  const { data } = await api.post(`/workspaces/${workspaceId}/messages`, req)
  return data
}

export async function markMessageRead(
  workspaceId: string,
  messageId: string
): Promise<void> {
  await api.post(`/workspaces/${workspaceId}/messages/${messageId}/read`)
}

// ─── Artifacts ───────────────────────────────────────────────────────────────

export async function getArtifacts(
  workspaceId: string
): Promise<ApiResponse<Artifact[]>> {
  const { data } = await api.get(`/workspaces/${workspaceId}/artifacts`)
  return data
}

export async function getArtifact(
  workspaceId: string,
  artifactId: string
): Promise<ApiResponse<Artifact>> {
  const { data } = await api.get(
    `/workspaces/${workspaceId}/artifacts/${artifactId}`
  )
  return data
}

export async function createArtifact(
  workspaceId: string,
  req: CreateArtifactRequest
): Promise<ApiResponse<Artifact>> {
  const { data } = await api.post(`/workspaces/${workspaceId}/artifacts`, req)
  return data
}

export async function updateArtifact(
  workspaceId: string,
  artifactId: string,
  req: UpdateArtifactRequest
): Promise<ApiResponse<Artifact>> {
  const { data } = await api.put(
    `/workspaces/${workspaceId}/artifacts/${artifactId}`,
    req
  )
  return data
}

// ─── Contexts ────────────────────────────────────────────────────────────────

export async function getContexts(
  workspaceId: string
): Promise<ApiResponse<Context[]>> {
  const { data } = await api.get(`/workspaces/${workspaceId}/contexts`)
  return data
}

// ─── Sync ────────────────────────────────────────────────────────────────────

export async function getSyncLog(
  workspaceId: string
): Promise<ApiResponse<SyncLogEntry[]>> {
  const { data } = await api.get(`/workspaces/${workspaceId}/sync/log`)
  return data
}

export async function resolveConflict(
  workspaceId: string,
  entryId: string,
  resolution: { chosen: 'local' | 'remote' | 'merged'; merged_data?: string }
): Promise<void> {
  await api.post(
    `/workspaces/${workspaceId}/sync/resolve/${entryId}`,
    resolution
  )
}

export default api
