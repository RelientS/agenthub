import { create } from 'zustand'
import type {
  Workspace,
  Agent,
  AgentStatus,
  Task,
  Message,
  Artifact,
  Context,
  SyncLogEntry,
  WsConflictEvent,
} from '@/types'

interface AppState {
  // Auth
  token: string | null
  agentId: string | null
  workspaceId: string | null
  setAuth(token: string, agentId: string, workspaceId: string): void
  clearAuth(): void

  // Workspace
  workspace: Workspace | null
  agents: Agent[]
  setWorkspace(ws: Workspace): void
  setAgents(agents: Agent[]): void
  updateAgentStatus(agentId: string, status: AgentStatus): void
  addOrUpdateAgent(agent: Agent): void
  removeAgent(agentId: string): void

  // Tasks
  tasks: Task[]
  setTasks(tasks: Task[]): void
  updateTask(task: Task): void
  addTask(task: Task): void
  removeTask(taskId: string): void

  // Messages
  messages: Message[]
  unreadCount: number
  setMessages(msgs: Message[]): void
  addMessage(msg: Message): void
  setUnreadCount(count: number): void

  // Artifacts
  artifacts: Artifact[]
  setArtifacts(arts: Artifact[]): void
  addArtifact(art: Artifact): void
  updateArtifact(art: Artifact): void

  // Contexts
  contexts: Context[]
  setContexts(ctxs: Context[]): void

  // Sync
  syncLog: SyncLogEntry[]
  conflicts: WsConflictEvent[]
  setSyncLog(log: SyncLogEntry[]): void
  addConflict(conflict: WsConflictEvent): void
  removeConflict(resourceId: string): void

  // WebSocket
  wsConnected: boolean
  setWsConnected(connected: boolean): void

  // UI
  sidebarCollapsed: boolean
  toggleSidebar(): void
}

export const useStore = create<AppState>((set) => ({
  // ─── Auth ────────────────────────────────────────────────────────────────
  token: localStorage.getItem('agenthub_token'),
  agentId: localStorage.getItem('agenthub_agent_id'),
  workspaceId: localStorage.getItem('agenthub_workspace_id'),

  setAuth(token, agentId, workspaceId) {
    localStorage.setItem('agenthub_token', token)
    localStorage.setItem('agenthub_agent_id', agentId)
    localStorage.setItem('agenthub_workspace_id', workspaceId)
    set({ token, agentId, workspaceId })
  },

  clearAuth() {
    localStorage.removeItem('agenthub_token')
    localStorage.removeItem('agenthub_agent_id')
    localStorage.removeItem('agenthub_workspace_id')
    set({
      token: null,
      agentId: null,
      workspaceId: null,
      workspace: null,
      agents: [],
      tasks: [],
      messages: [],
      artifacts: [],
      contexts: [],
    })
  },

  // ─── Workspace ───────────────────────────────────────────────────────────
  workspace: null,
  agents: [],

  setWorkspace(ws) {
    set({ workspace: ws })
  },

  setAgents(agents) {
    set({ agents })
  },

  updateAgentStatus(agentId, status) {
    set((state) => ({
      agents: state.agents.map((a) =>
        a.id === agentId ? { ...a, status } : a
      ),
    }))
  },

  addOrUpdateAgent(agent) {
    set((state) => {
      const exists = state.agents.find((a) => a.id === agent.id)
      if (exists) {
        return {
          agents: state.agents.map((a) => (a.id === agent.id ? agent : a)),
        }
      }
      return { agents: [...state.agents, agent] }
    })
  },

  removeAgent(agentId) {
    set((state) => ({
      agents: state.agents.filter((a) => a.id !== agentId),
    }))
  },

  // ─── Tasks ───────────────────────────────────────────────────────────────
  tasks: [],

  setTasks(tasks) {
    set({ tasks })
  },

  updateTask(task) {
    set((state) => ({
      tasks: state.tasks.map((t) => (t.id === task.id ? task : t)),
    }))
  },

  addTask(task) {
    set((state) => ({ tasks: [...state.tasks, task] }))
  },

  removeTask(taskId) {
    set((state) => ({
      tasks: state.tasks.filter((t) => t.id !== taskId),
    }))
  },

  // ─── Messages ────────────────────────────────────────────────────────────
  messages: [],
  unreadCount: 0,

  setMessages(msgs) {
    set({ messages: msgs })
  },

  addMessage(msg) {
    set((state) => ({
      messages: [...state.messages, msg],
      unreadCount: state.unreadCount + 1,
    }))
  },

  setUnreadCount(count) {
    set({ unreadCount: count })
  },

  // ─── Artifacts ───────────────────────────────────────────────────────────
  artifacts: [],

  setArtifacts(arts) {
    set({ artifacts: arts })
  },

  addArtifact(art) {
    set((state) => ({ artifacts: [...state.artifacts, art] }))
  },

  updateArtifact(art) {
    set((state) => ({
      artifacts: state.artifacts.map((a) => (a.id === art.id ? art : a)),
    }))
  },

  // ─── Contexts ────────────────────────────────────────────────────────────
  contexts: [],

  setContexts(ctxs) {
    set({ contexts: ctxs })
  },

  // ─── Sync ────────────────────────────────────────────────────────────────
  syncLog: [],
  conflicts: [],

  setSyncLog(log) {
    set({ syncLog: log })
  },

  addConflict(conflict) {
    set((state) => ({ conflicts: [...state.conflicts, conflict] }))
  },

  removeConflict(resourceId) {
    set((state) => ({
      conflicts: state.conflicts.filter((c) => c.resource_id !== resourceId),
    }))
  },

  // ─── WebSocket ───────────────────────────────────────────────────────────
  wsConnected: false,

  setWsConnected(connected) {
    set({ wsConnected: connected })
  },

  // ─── UI ──────────────────────────────────────────────────────────────────
  sidebarCollapsed: false,

  toggleSidebar() {
    set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }))
  },
}))
