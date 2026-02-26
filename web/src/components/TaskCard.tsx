import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { clsx } from 'clsx'
import { Clock, AlertTriangle, Link2, Tag, User } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import type { Task, TaskPriority } from '@/types'
import { useStore } from '@/store'

const priorityConfig: Record<
  TaskPriority,
  { color: string; bg: string; label: string }
> = {
  critical: {
    color: 'text-red-300',
    bg: 'bg-red-500/20 border-red-500/30',
    label: 'Critical',
  },
  high: {
    color: 'text-orange-300',
    bg: 'bg-orange-500/20 border-orange-500/30',
    label: 'High',
  },
  medium: {
    color: 'text-yellow-300',
    bg: 'bg-yellow-500/20 border-yellow-500/30',
    label: 'Medium',
  },
  low: {
    color: 'text-blue-300',
    bg: 'bg-blue-500/20 border-blue-500/30',
    label: 'Low',
  },
}

interface TaskCardProps {
  task: Task
  onClick?: (task: Task) => void
  isDragOverlay?: boolean
}

export function TaskCard({ task, onClick, isDragOverlay = false }: TaskCardProps) {
  const agents = useStore((s) => s.agents)
  const assignee = agents.find((a) => a.id === task.assignee_id)
  const priority = priorityConfig[task.priority]

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: task.id,
    data: { task },
    disabled: isDragOverlay,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onClick={() => onClick?.(task)}
      className={clsx(
        'group cursor-pointer rounded-lg border border-surface-700 bg-surface-800 p-3 transition-all',
        'hover:border-surface-500 hover:bg-surface-750',
        isDragging && 'opacity-50',
        isDragOverlay && 'rotate-2 shadow-2xl shadow-primary-500/20'
      )}
    >
      {/* Priority Badge */}
      <div className="mb-2 flex items-center justify-between">
        <span
          className={clsx(
            'inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider',
            priority.bg,
            priority.color
          )}
        >
          {priority.label}
        </span>
        {task.dependencies.length > 0 && (
          <span className="flex items-center gap-0.5 text-xs text-surface-400">
            <Link2 className="h-3 w-3" />
            {task.dependencies.length}
          </span>
        )}
      </div>

      {/* Title */}
      <h4 className="mb-1.5 text-sm font-medium leading-snug text-surface-100 line-clamp-2">
        {task.title}
      </h4>

      {/* Description preview */}
      {task.description && (
        <p className="mb-2 text-xs text-surface-400 line-clamp-2">
          {task.description}
        </p>
      )}

      {/* Tags */}
      {task.tags.length > 0 && (
        <div className="mb-2 flex flex-wrap gap-1">
          {task.tags.slice(0, 3).map((tag) => (
            <span
              key={tag}
              className="inline-flex items-center gap-0.5 rounded bg-surface-700 px-1.5 py-0.5 text-[10px] text-surface-300"
            >
              <Tag className="h-2.5 w-2.5" />
              {tag}
            </span>
          ))}
          {task.tags.length > 3 && (
            <span className="text-[10px] text-surface-500">
              +{task.tags.length - 3}
            </span>
          )}
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between pt-1">
        {assignee ? (
          <div className="flex items-center gap-1">
            <div
              className={clsx(
                'flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-semibold',
                assignee.status === 'online'
                  ? 'bg-primary-600 text-white'
                  : 'bg-surface-600 text-surface-300'
              )}
            >
              {assignee.name[0].toUpperCase()}
            </div>
            <span className="text-xs text-surface-400 truncate max-w-[80px]">
              {assignee.name}
            </span>
          </div>
        ) : (
          <div className="flex items-center gap-1 text-xs text-surface-500">
            <User className="h-3 w-3" />
            Unassigned
          </div>
        )}

        {task.due_date && (
          <div className="flex items-center gap-0.5 text-xs text-surface-400">
            <Clock className="h-3 w-3" />
            {formatDistanceToNow(new Date(task.due_date), { addSuffix: true })}
          </div>
        )}

        {task.status === 'blocked' && (
          <AlertTriangle className="h-3.5 w-3.5 text-red-400" />
        )}
      </div>
    </div>
  )
}

interface TaskDetailPanelProps {
  task: Task
  onClose: () => void
  onUpdate: (updates: Partial<Task>) => void
}

export function TaskDetailPanel({
  task,
  onClose,
  onUpdate,
}: TaskDetailPanelProps) {
  const agents = useStore((s) => s.agents)
  const priority = priorityConfig[task.priority]

  return (
    <div className="flex h-full flex-col border-l border-surface-700 bg-surface-900">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-surface-700 px-4 py-3">
        <h3 className="text-sm font-semibold text-surface-200">Task Detail</h3>
        <button
          onClick={onClose}
          className="rounded p-1 text-surface-400 hover:bg-surface-800 hover:text-surface-200"
        >
          &times;
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        <div>
          <h2 className="text-lg font-semibold text-white">{task.title}</h2>
          <span
            className={clsx(
              'mt-1 inline-flex items-center rounded border px-2 py-0.5 text-xs font-semibold uppercase',
              priority.bg,
              priority.color
            )}
          >
            {priority.label}
          </span>
        </div>

        {task.description && (
          <div>
            <label className="text-xs font-medium uppercase text-surface-500">
              Description
            </label>
            <p className="mt-1 text-sm text-surface-300 whitespace-pre-wrap">
              {task.description}
            </p>
          </div>
        )}

        <div>
          <label className="text-xs font-medium uppercase text-surface-500">
            Assignee
          </label>
          <select
            value={task.assignee_id || ''}
            onChange={(e) =>
              onUpdate({ assignee_id: e.target.value || null } as Partial<Task>)
            }
            className="mt-1 w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-1.5 text-sm text-surface-200 outline-none focus:border-primary-500"
          >
            <option value="">Unassigned</option>
            {agents.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="text-xs font-medium uppercase text-surface-500">
            Status
          </label>
          <select
            value={task.status}
            onChange={(e) =>
              onUpdate({ status: e.target.value } as Partial<Task>)
            }
            className="mt-1 w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-1.5 text-sm text-surface-200 outline-none focus:border-primary-500"
          >
            <option value="pending">Pending</option>
            <option value="assigned">Assigned</option>
            <option value="in_progress">In Progress</option>
            <option value="review">Review</option>
            <option value="blocked">Blocked</option>
            <option value="completed">Completed</option>
          </select>
        </div>

        {task.tags.length > 0 && (
          <div>
            <label className="text-xs font-medium uppercase text-surface-500">
              Tags
            </label>
            <div className="mt-1 flex flex-wrap gap-1">
              {task.tags.map((tag) => (
                <span
                  key={tag}
                  className="rounded bg-surface-700 px-2 py-0.5 text-xs text-surface-300"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}

        {task.dependencies.length > 0 && (
          <div>
            <label className="text-xs font-medium uppercase text-surface-500">
              Dependencies
            </label>
            <div className="mt-1 space-y-1">
              {task.dependencies.map((depId) => (
                <div
                  key={depId}
                  className="flex items-center gap-1 text-xs text-surface-400"
                >
                  <Link2 className="h-3 w-3" />
                  {depId}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-xs font-medium uppercase text-surface-500">
              Created
            </label>
            <p className="mt-0.5 text-xs text-surface-400">
              {new Date(task.created_at).toLocaleString()}
            </p>
          </div>
          <div>
            <label className="text-xs font-medium uppercase text-surface-500">
              Updated
            </label>
            <p className="mt-0.5 text-xs text-surface-400">
              {new Date(task.updated_at).toLocaleString()}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
