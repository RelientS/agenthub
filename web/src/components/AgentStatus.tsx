import { clsx } from 'clsx'
import type { Agent, AgentStatus as AgentStatusType } from '@/types'

const statusConfig: Record<AgentStatusType, { color: string; label: string }> = {
  online: { color: 'bg-emerald-400', label: 'Online' },
  offline: { color: 'bg-gray-500', label: 'Offline' },
  busy: { color: 'bg-amber-400', label: 'Busy' },
  idle: { color: 'bg-blue-400', label: 'Idle' },
}

const typeIcons: Record<Agent['type'], string> = {
  claude: 'C',
  gpt: 'G',
  human: 'H',
  custom: 'X',
}

interface AgentStatusProps {
  agent: Agent
  size?: 'sm' | 'md' | 'lg'
  showName?: boolean
  showType?: boolean
}

export function AgentStatusIndicator({
  agent,
  size = 'md',
  showName = true,
  showType = false,
}: AgentStatusProps) {
  const config = statusConfig[agent.status]
  const sizeClasses = {
    sm: 'h-6 w-6 text-xs',
    md: 'h-8 w-8 text-sm',
    lg: 'h-10 w-10 text-base',
  }
  const dotSizes = {
    sm: 'h-2 w-2',
    md: 'h-2.5 w-2.5',
    lg: 'h-3 w-3',
  }

  return (
    <div className="flex items-center gap-2">
      <div className="relative flex-shrink-0">
        <div
          className={clsx(
            'flex items-center justify-center rounded-full font-semibold',
            sizeClasses[size],
            agent.status === 'online'
              ? 'bg-primary-600 text-white'
              : agent.status === 'busy'
                ? 'bg-amber-600 text-white'
                : 'bg-surface-700 text-surface-300'
          )}
        >
          {typeIcons[agent.type]}
        </div>
        <span
          className={clsx(
            'absolute -bottom-0.5 -right-0.5 rounded-full border-2 border-surface-900',
            dotSizes[size],
            config.color,
            agent.status === 'online' && 'animate-pulse-slow'
          )}
        />
      </div>
      {showName && (
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-surface-100">
            {agent.name}
          </p>
          {showType && (
            <p className="text-xs text-surface-400">
              {agent.type} &middot; {config.label}
            </p>
          )}
        </div>
      )}
    </div>
  )
}

interface AgentAvatarGroupProps {
  agents: Agent[]
  max?: number
}

export function AgentAvatarGroup({ agents, max = 5 }: AgentAvatarGroupProps) {
  const visible = agents.slice(0, max)
  const remaining = agents.length - max

  return (
    <div className="flex -space-x-2">
      {visible.map((agent) => (
        <div
          key={agent.id}
          className={clsx(
            'relative flex h-8 w-8 items-center justify-center rounded-full border-2 border-surface-900 text-xs font-semibold',
            agent.status === 'online'
              ? 'bg-primary-600 text-white'
              : 'bg-surface-700 text-surface-300'
          )}
          title={`${agent.name} (${agent.status})`}
        >
          {typeIcons[agent.type]}
        </div>
      ))}
      {remaining > 0 && (
        <div className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-surface-900 bg-surface-600 text-xs font-medium text-surface-200">
          +{remaining}
        </div>
      )}
    </div>
  )
}
