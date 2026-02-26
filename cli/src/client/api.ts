import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import {
  APIResponse,
  Workspace,
  Agent,
  Task,
  TaskBoard,
  TaskFilters,
  Message,
  MessageFilters,
  Artifact,
  ArtifactFilters,
  Context,
  CreateWorkspaceRequest,
  CreateWorkspaceResponse,
  JoinWorkspaceRequest,
  JoinWorkspaceResponse,
  CreateTaskRequest,
  UpdateTaskRequest,
  CreateArtifactRequest,
  CreateContextRequest,
  UpdateContextRequest,
  SendMessageRequest,
  SyncChange,
  SyncPushResponse,
  SyncPullResponse,
  SyncStatusResponse,
} from '../types';

/**
 * ApiClient provides typed access to every AgentHub REST endpoint.
 *
 * All public methods unwrap the standard APIResponse envelope and throw
 * on server-side errors so callers can rely on simple try/catch handling.
 */
export class ApiClient {
  private http: AxiosInstance;

  constructor(private baseUrl: string, private token?: string) {
    this.http = axios.create({
      baseURL: this.baseUrl,
      timeout: 30_000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Attach bearer token when available.
    this.http.interceptors.request.use((config) => {
      if (this.token) {
        config.headers.Authorization = `Bearer ${this.token}`;
      }
      return config;
    });

    // Unwrap error messages from the standard envelope.
    this.http.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.data?.error) {
          const apiErr = error.response.data.error;
          const wrapped = new Error(`[${apiErr.code}] ${apiErr.message}`);
          (wrapped as any).code = apiErr.code;
          throw wrapped;
        }
        throw error;
      },
    );
  }

  /** Replace the current auth token (e.g. after join/create). */
  setToken(token: string): void {
    this.token = token;
  }

  // ================================================================
  // Private helpers
  // ================================================================

  private async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const res = await this.http.get<APIResponse<T>>(url, config);
    return res.data.data;
  }

  private async post<T>(url: string, body?: unknown): Promise<T> {
    const res = await this.http.post<APIResponse<T>>(url, body);
    return res.data.data;
  }

  private async put<T>(url: string, body?: unknown): Promise<T> {
    const res = await this.http.put<APIResponse<T>>(url, body);
    return res.data.data;
  }

  private async patch<T>(url: string, body?: unknown): Promise<T> {
    const res = await this.http.patch<APIResponse<T>>(url, body);
    return res.data.data;
  }

  private async del<T = void>(url: string): Promise<T> {
    const res = await this.http.delete<APIResponse<T>>(url);
    return res.data.data;
  }

  // ================================================================
  // Workspace
  // ================================================================

  async createWorkspace(req: CreateWorkspaceRequest): Promise<CreateWorkspaceResponse> {
    return this.post<CreateWorkspaceResponse>('/api/v1/workspaces', req);
  }

  async joinWorkspace(req: JoinWorkspaceRequest): Promise<JoinWorkspaceResponse> {
    return this.post<JoinWorkspaceResponse>('/api/v1/workspaces/join', req);
  }

  async getWorkspace(id: string): Promise<Workspace> {
    return this.get<Workspace>(`/api/v1/workspaces/${id}`);
  }

  async leaveWorkspace(workspaceId: string): Promise<void> {
    await this.post<void>(`/api/v1/workspaces/${workspaceId}/leave`);
  }

  async listAgents(workspaceId: string): Promise<Agent[]> {
    return this.get<Agent[]>(`/api/v1/workspaces/${workspaceId}/agents`);
  }

  // ================================================================
  // Tasks
  // ================================================================

  async createTask(workspaceId: string, req: CreateTaskRequest): Promise<Task> {
    return this.post<Task>(`/api/v1/workspaces/${workspaceId}/tasks`, req);
  }

  async listTasks(
    workspaceId: string,
    filters?: TaskFilters,
  ): Promise<{ tasks: Task[]; total: number }> {
    const params: Record<string, string> = {};
    if (filters?.status) params.status = filters.status;
    if (filters?.priority) params.priority = filters.priority;
    if (filters?.assigned_to) params.assigned_to = filters.assigned_to;
    if (filters?.created_by) params.created_by = filters.created_by;
    return this.get<{ tasks: Task[]; total: number }>(
      `/api/v1/workspaces/${workspaceId}/tasks`,
      { params },
    );
  }

  async getTask(workspaceId: string, taskId: string): Promise<Task> {
    return this.get<Task>(`/api/v1/workspaces/${workspaceId}/tasks/${taskId}`);
  }

  async updateTask(
    workspaceId: string,
    taskId: string,
    req: UpdateTaskRequest,
  ): Promise<Task> {
    return this.patch<Task>(`/api/v1/workspaces/${workspaceId}/tasks/${taskId}`, req);
  }

  async claimTask(workspaceId: string, taskId: string): Promise<Task> {
    return this.post<Task>(`/api/v1/workspaces/${workspaceId}/tasks/${taskId}/claim`);
  }

  async completeTask(workspaceId: string, taskId: string): Promise<Task> {
    return this.post<Task>(`/api/v1/workspaces/${workspaceId}/tasks/${taskId}/complete`);
  }

  async blockTask(workspaceId: string, taskId: string, reason: string): Promise<Task> {
    return this.post<Task>(`/api/v1/workspaces/${workspaceId}/tasks/${taskId}/block`, {
      reason,
    });
  }

  async getBoard(workspaceId: string): Promise<TaskBoard> {
    return this.get<TaskBoard>(`/api/v1/workspaces/${workspaceId}/tasks/board`);
  }

  // ================================================================
  // Messages
  // ================================================================

  async sendMessage(workspaceId: string, req: SendMessageRequest): Promise<Message> {
    return this.post<Message>(`/api/v1/workspaces/${workspaceId}/messages`, req);
  }

  async listMessages(
    workspaceId: string,
    filters?: MessageFilters,
  ): Promise<{ messages: Message[]; total: number }> {
    const params: Record<string, string> = {};
    if (filters?.from_agent_id) params.from_agent_id = filters.from_agent_id;
    if (filters?.to_agent_id) params.to_agent_id = filters.to_agent_id;
    if (filters?.message_type) params.message_type = filters.message_type;
    if (filters?.thread_id) params.thread_id = filters.thread_id;
    if (filters?.is_read !== undefined) params.is_read = String(filters.is_read);
    return this.get<{ messages: Message[]; total: number }>(
      `/api/v1/workspaces/${workspaceId}/messages`,
      { params },
    );
  }

  async getUnread(workspaceId: string): Promise<Message[]> {
    return this.get<Message[]>(`/api/v1/workspaces/${workspaceId}/messages/unread`);
  }

  async markAsRead(workspaceId: string, msgId: string): Promise<void> {
    await this.post<void>(
      `/api/v1/workspaces/${workspaceId}/messages/${msgId}/read`,
    );
  }

  async getThread(workspaceId: string, threadId: string): Promise<Message[]> {
    return this.get<Message[]>(
      `/api/v1/workspaces/${workspaceId}/messages/thread/${threadId}`,
    );
  }

  // ================================================================
  // Artifacts
  // ================================================================

  async createArtifact(
    workspaceId: string,
    req: CreateArtifactRequest,
  ): Promise<Artifact> {
    return this.post<Artifact>(`/api/v1/workspaces/${workspaceId}/artifacts`, req);
  }

  async listArtifacts(
    workspaceId: string,
    filters?: ArtifactFilters,
  ): Promise<{ artifacts: Artifact[]; total: number }> {
    const params: Record<string, string> = {};
    if (filters?.artifact_type) params.artifact_type = filters.artifact_type;
    if (filters?.created_by) params.created_by = filters.created_by;
    if (filters?.name) params.name = filters.name;
    return this.get<{ artifacts: Artifact[]; total: number }>(
      `/api/v1/workspaces/${workspaceId}/artifacts`,
      { params },
    );
  }

  async getArtifact(workspaceId: string, artifactId: string): Promise<Artifact> {
    return this.get<Artifact>(
      `/api/v1/workspaces/${workspaceId}/artifacts/${artifactId}`,
    );
  }

  async getArtifactHistory(
    workspaceId: string,
    artifactId: string,
  ): Promise<Artifact[]> {
    return this.get<Artifact[]>(
      `/api/v1/workspaces/${workspaceId}/artifacts/${artifactId}/history`,
    );
  }

  async searchArtifacts(workspaceId: string, query: string): Promise<Artifact[]> {
    return this.get<Artifact[]>(
      `/api/v1/workspaces/${workspaceId}/artifacts/search`,
      { params: { q: query } },
    );
  }

  // ================================================================
  // Contexts
  // ================================================================

  async createContext(
    workspaceId: string,
    req: CreateContextRequest,
  ): Promise<Context> {
    return this.post<Context>(`/api/v1/workspaces/${workspaceId}/contexts`, req);
  }

  async listContexts(workspaceId: string): Promise<Context[]> {
    return this.get<Context[]>(`/api/v1/workspaces/${workspaceId}/contexts`);
  }

  async getContext(workspaceId: string, contextId: string): Promise<Context> {
    return this.get<Context>(
      `/api/v1/workspaces/${workspaceId}/contexts/${contextId}`,
    );
  }

  async updateContext(
    workspaceId: string,
    contextId: string,
    req: UpdateContextRequest,
  ): Promise<Context> {
    return this.patch<Context>(
      `/api/v1/workspaces/${workspaceId}/contexts/${contextId}`,
      req,
    );
  }

  async getSnapshot(workspaceId: string): Promise<Context[]> {
    return this.get<Context[]>(
      `/api/v1/workspaces/${workspaceId}/contexts/snapshot`,
    );
  }

  // ================================================================
  // Sync
  // ================================================================

  async syncPush(
    workspaceId: string,
    changes: SyncChange[],
  ): Promise<SyncPushResponse> {
    return this.post<SyncPushResponse>(
      `/api/v1/workspaces/${workspaceId}/sync/push`,
      { changes },
    );
  }

  async syncPull(
    workspaceId: string,
    lastSyncId: number,
    entityTypes?: string[],
  ): Promise<SyncPullResponse> {
    const params: Record<string, string> = {
      last_sync_id: String(lastSyncId),
    };
    if (entityTypes?.length) {
      params.entity_types = entityTypes.join(',');
    }
    return this.get<SyncPullResponse>(
      `/api/v1/workspaces/${workspaceId}/sync/pull`,
      { params },
    );
  }

  async syncStatus(workspaceId: string): Promise<SyncStatusResponse> {
    return this.get<SyncStatusResponse>(
      `/api/v1/workspaces/${workspaceId}/sync/status`,
    );
  }

  // ================================================================
  // Heartbeat
  // ================================================================

  async heartbeat(): Promise<void> {
    await this.post<void>('/api/v1/heartbeat');
  }
}
