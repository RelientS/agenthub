import { useState, useMemo, useCallback } from 'react'
import {
  DndContext,
  DragOverlay,
  closestCorners,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { useDroppable } from '@dnd-kit/core'
import { clsx } from 'clsx'
import {
  Plus,
  Filter,
  Search,
  X,
} from 'lucide-react'
import { useStore } from '@/store'
import { TaskCard, TaskDetailPanel } from '@/components/TaskCard'
import * as api from '@/api/client'
import type { Task, TaskStatus, TaskPriority, TaskColumn, CreateTaskRequest } from '@/types'

const columns: TaskColumn[] = [
  { id: 'pending', title: 'Pending', tasks: [], color: 'border-t-slate-400' },
  { id: 'assigned', title: 'Assigned', tasks: [], color: 'border-t-blue-400' },
  {
    id: 'in_progress',
    title: 'In Progress',
    tasks: [],
    color: 'border-t-amber-400',
  },
  { id: 'review', title: 'Review', tasks: [], color: 'border-t-purple-400' },
  { id: 'blocked', title: 'Blocked', tasks: [], color: 'border-t-red-400' },
  {
    id: 'completed',
    title: 'Completed',
    tasks: [],
    color: 'border-t-emerald-400',
  },
]

interface DroppableColumnProps {
  column: TaskColumn
  tasks: Task[]
  onTaskClick: (task: Task) => void
}

function DroppableColumn({ column, tasks, onTaskClick }: DroppableColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id: column.id })

  return (
    <div
      className={clsx(
        'flex min-h-[200px] flex-col rounded-lg border border-surface-700 border-t-2 bg-surface-900/50',
        column.color,
        isOver && 'border-primary-500/50 bg-primary-500/5'
      )}
    >
      {/* Column Header */}
      <div className="flex items-center justify-between px-3 py-2.5">
        <div className="flex items-center gap-2">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-surface-300">
            {column.title}
          </h3>
          <span className="flex h-5 min-w-[20px] items-center justify-center rounded-full bg-surface-700 px-1.5 text-[10px] font-medium text-surface-400">
            {tasks.length}
          </span>
        </div>
      </div>

      {/* Task List */}
      <div
        ref={setNodeRef}
        className="flex-1 space-y-2 overflow-y-auto px-2 pb-2"
      >
        <SortableContext
          items={tasks.map((t) => t.id)}
          strategy={verticalListSortingStrategy}
        >
          {tasks.map((task) => (
            <TaskCard key={task.id} task={task} onClick={onTaskClick} />
          ))}
        </SortableContext>
        {tasks.length === 0 && (
          <div className="flex h-20 items-center justify-center rounded-lg border border-dashed border-surface-700 text-xs text-surface-500">
            Drop tasks here
          </div>
        )}
      </div>
    </div>
  )
}

export function TaskBoard() {
  const workspaceId = useStore((s) => s.workspaceId)
  const tasks = useStore((s) => s.tasks)
  const agents = useStore((s) => s.agents)
  const storeUpdateTask = useStore((s) => s.updateTask)
  const addTask = useStore((s) => s.addTask)

  const [activeTask, setActiveTask] = useState<Task | null>(null)
  const [selectedTask, setSelectedTask] = useState<Task | null>(null)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [filterAgent, setFilterAgent] = useState<string>('')
  const [filterPriority, setFilterPriority] = useState<string>('')
  const [filterTag, setFilterTag] = useState<string>('')
  const [searchQuery, setSearchQuery] = useState('')
  const [showFilters, setShowFilters] = useState(false)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor)
  )

  const filteredTasks = useMemo(() => {
    let result = tasks

    if (filterAgent) {
      result = result.filter((t) => t.assignee_id === filterAgent)
    }
    if (filterPriority) {
      result = result.filter((t) => t.priority === filterPriority)
    }
    if (filterTag) {
      result = result.filter((t) => t.tags.includes(filterTag))
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      result = result.filter(
        (t) =>
          t.title.toLowerCase().includes(q) ||
          t.description.toLowerCase().includes(q)
      )
    }

    return result
  }, [tasks, filterAgent, filterPriority, filterTag, searchQuery])

  const tasksByColumn = useMemo(() => {
    const grouped: Record<TaskStatus, Task[]> = {
      pending: [],
      assigned: [],
      in_progress: [],
      review: [],
      blocked: [],
      completed: [],
    }
    for (const task of filteredTasks) {
      if (grouped[task.status]) {
        grouped[task.status].push(task)
      }
    }
    // Sort by priority within each column
    const priorityOrder: Record<TaskPriority, number> = {
      critical: 0,
      high: 1,
      medium: 2,
      low: 3,
    }
    for (const status of Object.keys(grouped) as TaskStatus[]) {
      grouped[status].sort(
        (a, b) => priorityOrder[a.priority] - priorityOrder[b.priority]
      )
    }
    return grouped
  }, [filteredTasks])

  const allTags = useMemo(() => {
    const tagSet = new Set<string>()
    tasks.forEach((t) => t.tags.forEach((tag) => tagSet.add(tag)))
    return Array.from(tagSet).sort()
  }, [tasks])

  const handleDragStart = (event: DragStartEvent) => {
    const task = tasks.find((t) => t.id === event.active.id)
    if (task) setActiveTask(task)
  }

  const handleDragEnd = useCallback(
    async (event: DragEndEvent) => {
      setActiveTask(null)
      const { active, over } = event
      if (!over || !workspaceId) return

      const task = tasks.find((t) => t.id === active.id)
      if (!task) return

      // Determine target column
      let targetStatus: TaskStatus | null = null

      // Check if dropped on a column
      const colIds: TaskStatus[] = [
        'pending',
        'assigned',
        'in_progress',
        'review',
        'blocked',
        'completed',
      ]
      if (colIds.includes(over.id as TaskStatus)) {
        targetStatus = over.id as TaskStatus
      } else {
        // Dropped on another task - find which column that task is in
        const overTask = tasks.find((t) => t.id === over.id)
        if (overTask) {
          targetStatus = overTask.status
        }
      }

      if (targetStatus && targetStatus !== task.status) {
        // Optimistic update
        const updated = { ...task, status: targetStatus }
        storeUpdateTask(updated)

        try {
          await api.updateTask(workspaceId, task.id, {
            status: targetStatus,
          })
        } catch (err) {
          console.error('Failed to update task status:', err)
          // Revert
          storeUpdateTask(task)
        }
      }
    },
    [tasks, workspaceId, storeUpdateTask]
  )

  const handleTaskUpdate = async (updates: Partial<Task>) => {
    if (!workspaceId || !selectedTask) return
    const updated = { ...selectedTask, ...updates } as Task
    storeUpdateTask(updated)
    setSelectedTask(updated)
    try {
      await api.updateTask(workspaceId, selectedTask.id, updates)
    } catch (err) {
      console.error('Failed to update task:', err)
      storeUpdateTask(selectedTask)
      setSelectedTask(selectedTask)
    }
  }

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h1 className="text-xl font-bold text-white">Task Board</h1>
          <span className="rounded bg-surface-700 px-2 py-0.5 text-xs text-surface-400">
            {tasks.length} tasks
          </span>
        </div>
        <div className="flex items-center gap-2">
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-surface-500" />
            <input
              type="text"
              placeholder="Search tasks..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-8 rounded-md border border-surface-600 bg-surface-800 pl-8 pr-3 text-xs text-surface-200 outline-none focus:border-primary-500"
            />
          </div>
          {/* Filter toggle */}
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={clsx(
              'flex h-8 items-center gap-1.5 rounded-md border px-3 text-xs font-medium transition-colors',
              showFilters
                ? 'border-primary-500/50 bg-primary-500/10 text-primary-400'
                : 'border-surface-600 text-surface-400 hover:bg-surface-800'
            )}
          >
            <Filter className="h-3 w-3" />
            Filters
          </button>
          {/* Create */}
          <button
            onClick={() => setShowCreateForm(true)}
            className="flex h-8 items-center gap-1.5 rounded-md bg-primary-600 px-3 text-xs font-medium text-white transition-colors hover:bg-primary-500"
          >
            <Plus className="h-3.5 w-3.5" />
            New Task
          </button>
        </div>
      </div>

      {/* Filter Bar */}
      {showFilters && (
        <div className="mb-4 flex items-center gap-3 rounded-lg border border-surface-700 bg-surface-800/50 px-4 py-2.5">
          <select
            value={filterAgent}
            onChange={(e) => setFilterAgent(e.target.value)}
            className="h-7 rounded border border-surface-600 bg-surface-800 px-2 text-xs text-surface-300 outline-none"
          >
            <option value="">All Agents</option>
            {agents.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
          <select
            value={filterPriority}
            onChange={(e) => setFilterPriority(e.target.value)}
            className="h-7 rounded border border-surface-600 bg-surface-800 px-2 text-xs text-surface-300 outline-none"
          >
            <option value="">All Priorities</option>
            <option value="critical">Critical</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>
          <select
            value={filterTag}
            onChange={(e) => setFilterTag(e.target.value)}
            className="h-7 rounded border border-surface-600 bg-surface-800 px-2 text-xs text-surface-300 outline-none"
          >
            <option value="">All Tags</option>
            {allTags.map((tag) => (
              <option key={tag} value={tag}>
                {tag}
              </option>
            ))}
          </select>
          {(filterAgent || filterPriority || filterTag) && (
            <button
              onClick={() => {
                setFilterAgent('')
                setFilterPriority('')
                setFilterTag('')
              }}
              className="flex items-center gap-1 text-xs text-surface-400 hover:text-surface-200"
            >
              <X className="h-3 w-3" />
              Clear
            </button>
          )}
        </div>
      )}

      {/* Board */}
      <div className="flex flex-1 gap-3 overflow-x-auto pb-4">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          {columns.map((col) => (
            <div key={col.id} className="w-72 flex-shrink-0">
              <DroppableColumn
                column={col}
                tasks={tasksByColumn[col.id]}
                onTaskClick={setSelectedTask}
              />
            </div>
          ))}
          <DragOverlay>
            {activeTask ? (
              <div className="w-72">
                <TaskCard task={activeTask} isDragOverlay />
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      </div>

      {/* Task Detail Sidebar */}
      {selectedTask && (
        <div className="fixed inset-y-0 right-0 z-50 w-96 shadow-2xl">
          <TaskDetailPanel
            task={selectedTask}
            onClose={() => setSelectedTask(null)}
            onUpdate={handleTaskUpdate}
          />
        </div>
      )}

      {/* Create Task Modal */}
      {showCreateForm && (
        <CreateTaskModal
          onClose={() => setShowCreateForm(false)}
          onCreate={async (req) => {
            if (!workspaceId) return
            try {
              const res = await api.createTask(workspaceId, req)
              addTask(res.data)
              setShowCreateForm(false)
            } catch (err) {
              console.error('Failed to create task:', err)
            }
          }}
        />
      )}
    </div>
  )
}

interface CreateTaskModalProps {
  onClose: () => void
  onCreate: (req: CreateTaskRequest) => Promise<void>
}

function CreateTaskModal({ onClose, onCreate }: CreateTaskModalProps) {
  const agents = useStore((s) => s.agents)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<TaskPriority>('medium')
  const [assigneeId, setAssigneeId] = useState('')
  const [tags, setTags] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) return
    setSubmitting(true)
    await onCreate({
      title: title.trim(),
      description: description.trim(),
      priority,
      assignee_id: assigneeId || undefined,
      tags: tags
        ? tags.split(',').map((t) => t.trim()).filter(Boolean)
        : undefined,
    })
    setSubmitting(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-lg rounded-xl border border-surface-700 bg-surface-900 shadow-2xl">
        <div className="flex items-center justify-between border-b border-surface-700 px-5 py-3.5">
          <h2 className="text-sm font-semibold text-white">Create New Task</h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-surface-400 hover:bg-surface-800 hover:text-surface-200"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4 p-5">
          <div>
            <label className="mb-1 block text-xs font-medium text-surface-400">
              Title *
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
              placeholder="Task title..."
              autoFocus
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-surface-400">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full resize-none rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
              placeholder="Describe the task..."
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-surface-400">
                Priority
              </label>
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value as TaskPriority)}
                className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
              >
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-surface-400">
                Assignee
              </label>
              <select
                value={assigneeId}
                onChange={(e) => setAssigneeId(e.target.value)}
                className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
              >
                <option value="">Unassigned</option>
                {agents.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-surface-400">
              Tags (comma-separated)
            </label>
            <input
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
              placeholder="frontend, api, bug..."
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-4 py-2 text-sm text-surface-400 hover:bg-surface-800 hover:text-surface-200"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!title.trim() || submitting}
              className="rounded-md bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
            >
              {submitting ? 'Creating...' : 'Create Task'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
