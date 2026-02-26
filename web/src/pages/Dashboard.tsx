import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { clsx } from 'clsx'
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
} from 'recharts'
import { formatDistanceToNow } from 'date-fns'
import {
  Users,
  ListTodo,
  MessageSquare,
  FileCode,
  Plus,
  ArrowRight,
  Activity,
  CheckCircle2,
  Clock,
  AlertTriangle,
  Zap,
} from 'lucide-react'
import { useStore } from '@/store'
import { AgentStatusIndicator } from '@/components/AgentStatus'
import type { TaskStatus } from '@/types'

const taskStatusColors: Record<TaskStatus, string> = {
  pending: '#94a3b8',
  assigned: '#60a5fa',
  in_progress: '#f59e0b',
  review: '#a78bfa',
  blocked: '#ef4444',
  completed: '#10b981',
}

const taskStatusLabels: Record<TaskStatus, string> = {
  pending: 'Pending',
  assigned: 'Assigned',
  in_progress: 'In Progress',
  review: 'Review',
  blocked: 'Blocked',
  completed: 'Completed',
}

export function Dashboard() {
  const navigate = useNavigate()
  const workspace = useStore((s) => s.workspace)
  const agents = useStore((s) => s.agents)
  const tasks = useStore((s) => s.tasks)
  const messages = useStore((s) => s.messages)
  const unreadCount = useStore((s) => s.unreadCount)
  const artifacts = useStore((s) => s.artifacts)
  const conflicts = useStore((s) => s.conflicts)

  const onlineAgents = useMemo(
    () => agents.filter((a) => a.status === 'online' || a.status === 'busy'),
    [agents]
  )

  const taskStats = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const task of tasks) {
      counts[task.status] = (counts[task.status] || 0) + 1
    }
    return Object.entries(counts).map(([status, value]) => ({
      name: taskStatusLabels[status as TaskStatus] || status,
      value,
      color: taskStatusColors[status as TaskStatus] || '#64748b',
    }))
  }, [tasks])

  const completedTasks = tasks.filter((t) => t.status === 'completed').length
  const inProgressTasks = tasks.filter((t) => t.status === 'in_progress').length
  const blockedTasks = tasks.filter((t) => t.status === 'blocked').length

  const recentActivity = useMemo(() => {
    const items: Array<{
      id: string
      type: string
      description: string
      timestamp: string
      icon: React.ElementType
      color: string
    }> = []

    // Recent tasks
    tasks
      .slice()
      .sort(
        (a, b) =>
          new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
      )
      .slice(0, 8)
      .forEach((task) => {
        items.push({
          id: `task-${task.id}`,
          type: 'task',
          description: `Task "${task.title}" is ${task.status.replace('_', ' ')}`,
          timestamp: task.updated_at,
          icon: ListTodo,
          color: 'text-blue-400',
        })
      })

    // Recent messages
    messages
      .slice()
      .sort(
        (a, b) =>
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      )
      .slice(0, 6)
      .forEach((msg) => {
        const sender = agents.find((a) => a.id === msg.sender_id)
        items.push({
          id: `msg-${msg.id}`,
          type: 'message',
          description: `${sender?.name || 'Unknown'} sent a ${msg.message_type.replace('_', ' ')} message`,
          timestamp: msg.created_at,
          icon: MessageSquare,
          color: 'text-purple-400',
        })
      })

    // Recent artifacts
    artifacts
      .slice()
      .sort(
        (a, b) =>
          new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
      )
      .slice(0, 6)
      .forEach((art) => {
        items.push({
          id: `art-${art.id}`,
          type: 'artifact',
          description: `Artifact "${art.name}" updated to v${art.version}`,
          timestamp: art.updated_at,
          icon: FileCode,
          color: 'text-green-400',
        })
      })

    return items
      .sort(
        (a, b) =>
          new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
      )
      .slice(0, 20)
  }, [tasks, messages, artifacts, agents])

  return (
    <div className="space-y-6">
      {/* Workspace Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">
          {workspace?.name || 'Workspace'}
        </h1>
        <p className="mt-1 text-sm text-surface-400">
          {workspace?.description || 'Multi-agent collaboration workspace'}
        </p>
      </div>

      {/* Conflict Banner */}
      {conflicts.length > 0 && (
        <div className="flex items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3">
          <AlertTriangle className="h-5 w-5 text-amber-400" />
          <div className="flex-1">
            <p className="text-sm font-medium text-amber-200">
              {conflicts.length} sync conflict{conflicts.length !== 1 ? 's' : ''}{' '}
              detected
            </p>
            <p className="text-xs text-amber-300/70">
              Review and resolve conflicts to keep data in sync
            </p>
          </div>
          <button className="rounded-md bg-amber-500/20 px-3 py-1.5 text-xs font-medium text-amber-300 hover:bg-amber-500/30">
            Resolve
          </button>
        </div>
      )}

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Online Agents */}
        <div
          onClick={() => {}}
          className="group cursor-pointer rounded-xl border border-surface-700 bg-surface-800/50 p-5 transition-all hover:border-primary-500/30 hover:bg-surface-800"
        >
          <div className="mb-3 flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-500/10">
              <Users className="h-5 w-5 text-primary-400" />
            </div>
            <span className="text-2xl font-bold text-white">
              {onlineAgents.length}
              <span className="text-sm font-normal text-surface-500">
                /{agents.length}
              </span>
            </span>
          </div>
          <p className="text-sm font-medium text-surface-300">Online Agents</p>
          <div className="mt-3 flex -space-x-1.5">
            {onlineAgents.slice(0, 5).map((agent) => (
              <div
                key={agent.id}
                className={clsx(
                  'flex h-6 w-6 items-center justify-center rounded-full border-2 border-surface-800 text-[10px] font-semibold',
                  agent.status === 'busy'
                    ? 'bg-amber-600 text-white'
                    : 'bg-primary-600 text-white'
                )}
                title={agent.name}
              >
                {agent.name[0]}
              </div>
            ))}
          </div>
        </div>

        {/* Task Progress */}
        <div
          onClick={() => navigate('/tasks')}
          className="group cursor-pointer rounded-xl border border-surface-700 bg-surface-800/50 p-5 transition-all hover:border-emerald-500/30 hover:bg-surface-800"
        >
          <div className="mb-3 flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/10">
              <CheckCircle2 className="h-5 w-5 text-emerald-400" />
            </div>
            <span className="text-2xl font-bold text-white">
              {completedTasks}
              <span className="text-sm font-normal text-surface-500">
                /{tasks.length}
              </span>
            </span>
          </div>
          <p className="text-sm font-medium text-surface-300">Completed Tasks</p>
          <div className="mt-3 flex items-center gap-3 text-xs text-surface-400">
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3 text-amber-400" />
              {inProgressTasks} in progress
            </span>
            {blockedTasks > 0 && (
              <span className="flex items-center gap-1 text-red-400">
                <AlertTriangle className="h-3 w-3" />
                {blockedTasks} blocked
              </span>
            )}
          </div>
        </div>

        {/* Messages */}
        <div
          onClick={() => navigate('/messages')}
          className="group cursor-pointer rounded-xl border border-surface-700 bg-surface-800/50 p-5 transition-all hover:border-purple-500/30 hover:bg-surface-800"
        >
          <div className="mb-3 flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10">
              <MessageSquare className="h-5 w-5 text-purple-400" />
            </div>
            <span className="text-2xl font-bold text-white">
              {messages.length}
            </span>
          </div>
          <p className="text-sm font-medium text-surface-300">Messages</p>
          {unreadCount > 0 && (
            <div className="mt-3 flex items-center gap-1">
              <span className="flex h-5 items-center rounded-full bg-purple-500/20 px-2 text-xs font-medium text-purple-300">
                {unreadCount} unread
              </span>
            </div>
          )}
        </div>

        {/* Artifacts */}
        <div
          onClick={() => navigate('/artifacts')}
          className="group cursor-pointer rounded-xl border border-surface-700 bg-surface-800/50 p-5 transition-all hover:border-cyan-500/30 hover:bg-surface-800"
        >
          <div className="mb-3 flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-cyan-500/10">
              <FileCode className="h-5 w-5 text-cyan-400" />
            </div>
            <span className="text-2xl font-bold text-white">
              {artifacts.length}
            </span>
          </div>
          <p className="text-sm font-medium text-surface-300">Artifacts</p>
          <p className="mt-3 text-xs text-surface-400">
            {artifacts.filter((a) => a.type === 'code').length} code,{' '}
            {artifacts.filter((a) => a.type === 'config').length} config,{' '}
            {artifacts.filter((a) => a.type === 'schema').length} schema
          </p>
        </div>
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Task Progress Chart */}
        <div className="rounded-xl border border-surface-700 bg-surface-800/50 p-5">
          <h3 className="mb-4 text-sm font-semibold text-surface-200">
            Task Distribution
          </h3>
          {taskStats.length > 0 ? (
            <div className="flex items-center gap-4">
              <div className="h-40 w-40">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={taskStats}
                      cx="50%"
                      cy="50%"
                      innerRadius={35}
                      outerRadius={65}
                      paddingAngle={2}
                      dataKey="value"
                    >
                      {taskStats.map((entry, i) => (
                        <Cell key={i} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1e293b',
                        border: '1px solid #334155',
                        borderRadius: '8px',
                        fontSize: '12px',
                        color: '#e2e8f0',
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="space-y-1.5">
                {taskStats.map((entry) => (
                  <div key={entry.name} className="flex items-center gap-2">
                    <div
                      className="h-2.5 w-2.5 rounded-full"
                      style={{ backgroundColor: entry.color }}
                    />
                    <span className="text-xs text-surface-300">
                      {entry.name}
                    </span>
                    <span className="text-xs font-semibold text-surface-200">
                      {entry.value}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="flex h-40 items-center justify-center text-sm text-surface-500">
              No tasks yet
            </div>
          )}
        </div>

        {/* Agents Panel */}
        <div className="rounded-xl border border-surface-700 bg-surface-800/50 p-5">
          <h3 className="mb-4 text-sm font-semibold text-surface-200">
            Agents
          </h3>
          <div className="space-y-3">
            {agents.length === 0 ? (
              <p className="text-sm text-surface-500">No agents connected</p>
            ) : (
              agents
                .slice()
                .sort((a, b) => {
                  const order = { online: 0, busy: 1, idle: 2, offline: 3 }
                  return (order[a.status] ?? 4) - (order[b.status] ?? 4)
                })
                .map((agent) => (
                  <div
                    key={agent.id}
                    className="flex items-center justify-between rounded-lg bg-surface-800 px-3 py-2"
                  >
                    <AgentStatusIndicator
                      agent={agent}
                      size="sm"
                      showType
                    />
                    <div className="flex items-center gap-1.5">
                      {agent.capabilities.slice(0, 2).map((cap) => (
                        <span
                          key={cap}
                          className="rounded bg-surface-700 px-1.5 py-0.5 text-[10px] text-surface-400"
                        >
                          {cap}
                        </span>
                      ))}
                    </div>
                  </div>
                ))
            )}
          </div>
        </div>

        {/* Quick Actions */}
        <div className="rounded-xl border border-surface-700 bg-surface-800/50 p-5">
          <h3 className="mb-4 text-sm font-semibold text-surface-200">
            Quick Actions
          </h3>
          <div className="space-y-2">
            <button
              onClick={() => navigate('/tasks')}
              className="flex w-full items-center gap-3 rounded-lg bg-primary-600/10 px-4 py-3 text-left transition-colors hover:bg-primary-600/20"
            >
              <Plus className="h-4 w-4 text-primary-400" />
              <span className="text-sm font-medium text-primary-300">
                Create Task
              </span>
              <ArrowRight className="ml-auto h-4 w-4 text-primary-500" />
            </button>
            <button
              onClick={() => navigate('/messages')}
              className="flex w-full items-center gap-3 rounded-lg bg-purple-600/10 px-4 py-3 text-left transition-colors hover:bg-purple-600/20"
            >
              <MessageSquare className="h-4 w-4 text-purple-400" />
              <span className="text-sm font-medium text-purple-300">
                Send Message
              </span>
              <ArrowRight className="ml-auto h-4 w-4 text-purple-500" />
            </button>
            <button
              onClick={() => navigate('/artifacts')}
              className="flex w-full items-center gap-3 rounded-lg bg-cyan-600/10 px-4 py-3 text-left transition-colors hover:bg-cyan-600/20"
            >
              <FileCode className="h-4 w-4 text-cyan-400" />
              <span className="text-sm font-medium text-cyan-300">
                Browse Artifacts
              </span>
              <ArrowRight className="ml-auto h-4 w-4 text-cyan-500" />
            </button>
            <button
              onClick={() => {}}
              className="flex w-full items-center gap-3 rounded-lg bg-emerald-600/10 px-4 py-3 text-left transition-colors hover:bg-emerald-600/20"
            >
              <Zap className="h-4 w-4 text-emerald-400" />
              <span className="text-sm font-medium text-emerald-300">
                View Sync Log
              </span>
              <ArrowRight className="ml-auto h-4 w-4 text-emerald-500" />
            </button>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="rounded-xl border border-surface-700 bg-surface-800/50 p-5">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-surface-200">
            Recent Activity
          </h3>
          <Activity className="h-4 w-4 text-surface-500" />
        </div>
        {recentActivity.length === 0 ? (
          <p className="text-sm text-surface-500">No recent activity</p>
        ) : (
          <div className="space-y-0">
            {recentActivity.map((item, index) => {
              const Icon = item.icon
              return (
                <div
                  key={item.id}
                  className="relative flex items-start gap-3 py-2.5"
                >
                  {/* Timeline line */}
                  {index < recentActivity.length - 1 && (
                    <div className="absolute left-[11px] top-9 h-full w-px bg-surface-700" />
                  )}
                  <div
                    className={clsx(
                      'relative z-10 flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-surface-800',
                      item.color
                    )}
                  >
                    <Icon className="h-3 w-3" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm text-surface-300">
                      {item.description}
                    </p>
                    <p className="text-[11px] text-surface-500">
                      {formatDistanceToNow(new Date(item.timestamp), {
                        addSuffix: true,
                      })}
                    </p>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
