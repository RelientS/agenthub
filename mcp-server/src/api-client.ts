import axios, { AxiosInstance, AxiosError } from "axios";

export interface ApiConfig {
  serverUrl: string;
  token: string;
  workspaceId: string;
  agentId: string;
}

export interface ApiResponse<T> {
  data: T;
  error: { code: string; message: string } | null;
}

export class AgentHubClient {
  private client: AxiosInstance;
  private workspaceId: string;
  public agentId: string;

  constructor(config: ApiConfig) {
    this.workspaceId = config.workspaceId;
    this.agentId = config.agentId;
    this.client = axios.create({
      baseURL: `${config.serverUrl}/api/v1`,
      headers: {
        Authorization: `Bearer ${config.token}`,
        "Content-Type": "application/json",
      },
      timeout: 30000,
    });
  }

  private url(path: string): string {
    return `/workspaces/${this.workspaceId}${path}`;
  }

  private extractError(err: unknown): string {
    if (err instanceof AxiosError && err.response?.data?.error?.message) {
      return err.response.data.error.message;
    }
    return err instanceof Error ? err.message : String(err);
  }

  // --- Tasks ---

  async listTasks(filters?: {
    status?: string;
    assigned_to?: string;
    priority?: number;
  }): Promise<any[]> {
    const params = new URLSearchParams();
    if (filters?.status) params.set("status", filters.status);
    if (filters?.assigned_to) params.set("assigned_to", filters.assigned_to);
    if (filters?.priority) params.set("priority", String(filters.priority));

    const resp = await this.client.get(this.url("/tasks"), { params });
    return resp.data.data;
  }

  async getTask(taskId: string): Promise<any> {
    const resp = await this.client.get(this.url(`/tasks/${taskId}`));
    return resp.data.data;
  }

  async claimTask(taskId: string): Promise<any> {
    const resp = await this.client.post(this.url(`/tasks/${taskId}/claim`));
    return resp.data.data;
  }

  async completeTask(
    taskId: string,
    result?: string,
    metadata?: Record<string, any>
  ): Promise<any> {
    const resp = await this.client.post(
      this.url(`/tasks/${taskId}/complete`),
      { result, metadata }
    );
    return resp.data.data;
  }

  async updateTask(
    taskId: string,
    updates: Record<string, any>
  ): Promise<any> {
    const resp = await this.client.put(
      this.url(`/tasks/${taskId}`),
      updates
    );
    return resp.data.data;
  }

  async getBoard(): Promise<any> {
    const resp = await this.client.get(this.url("/tasks/board"));
    return resp.data.data;
  }

  // --- Messages ---

  async sendMessage(body: Record<string, any>): Promise<any> {
    const resp = await this.client.post(this.url("/messages"), body);
    return resp.data.data;
  }

  // --- Reports ---

  async generateDailySummary(): Promise<any> {
    const resp = await this.client.post(this.url("/reports/generate"));
    return resp.data.data;
  }

  async listReports(limit?: number): Promise<any[]> {
    const params = limit ? { limit: String(limit) } : {};
    const resp = await this.client.get(this.url("/reports"), { params });
    return resp.data.data;
  }

  // --- Agents ---

  async listAgents(): Promise<any[]> {
    const resp = await this.client.get(this.url("/agents"));
    return resp.data.data;
  }
}
