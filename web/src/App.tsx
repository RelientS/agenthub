import { Routes, Route, Navigate, NavLink, useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
import {
  LayoutDashboard,
  Kanban,
  MessageSquare,
  FileCode,
  Boxes,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Settings,
} from 'lucide-react'
import { useStore } from '@/store'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useWorkspace } from '@/hooks/useWorkspace'
import { SyncIndicator } from '@/components/SyncIndicator'
import { ConflictResolver } from '@/components/ConflictResolver'
import { Dashboard } from '@/pages/Dashboard'
import { TaskBoard } from '@/pages/TaskBoard'
import { MessageCenter } from '@/pages/MessageCenter'
import { ArtifactBrowser } from '@/pages/ArtifactBrowser'
import { JoinWorkspace } from '@/pages/JoinWorkspace'
import * as api from '@/api/client'
import './App.css'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/tasks', icon: Kanban, label: 'Tasks' },
  { to: '/messages', icon: MessageSquare, label: 'Messages' },
  { to: '/artifacts', icon: FileCode, label: 'Artifacts' },
]

function AuthenticatedLayout() {
  const { refetch } = useWorkspace()
  useWebSocket()

  const workspace = useStore((s) => s.workspace)
  const agentId = useStore((s) => s.agentId)
  const agents = useStore((s) => s.agents)
  const unreadCount = useStore((s) => s.unreadCount)
  const conflicts = useStore((s) => s.conflicts)
  const removeConflict = useStore((s) => s.removeConflict)
  const clearAuth = useStore((s) => s.clearAuth)
  const sidebarCollapsed = useStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useStore((s) => s.toggleSidebar)
  const workspaceId = useStore((s) => s.workspaceId)
  const location = useLocation()

  const currentAgent = agents.find((a) => a.id === agentId)

  const handleResolveConflict = async (
    resourceId: string,
    chosen: 'local' | 'remote' | 'merged',
    mergedData?: string
  ) => {
    if (!workspaceId) return
    try {
      await api.resolveConflict(workspaceId, resourceId, { chosen, merged_data: mergedData })
      removeConflict(resourceId)
      refetch()
    } catch (err) {
      console.error('Failed to resolve conflict:', err)
    }
  }

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside
        className={clsx(
          'flex flex-col border-r border-surface-700 bg-surface-900 transition-all duration-200',
          sidebarCollapsed ? 'w-16' : 'w-56'
        )}
      >
        {/* Logo */}
        <div className="flex h-14 items-center gap-2.5 border-b border-surface-700 px-4">
          <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-primary-600">
            <Boxes className="h-4 w-4 text-white" />
          </div>
          {!sidebarCollapsed && (
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold text-white">
                AgentHub
              </p>
              <p className="truncate text-[10px] text-surface-500">
                {workspace?.name || 'Loading...'}
              </p>
            </div>
          )}
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-0.5 p-2">
          {navItems.map((item) => {
            const Icon = item.icon
            const isActive =
              item.to === '/'
                ? location.pathname === '/'
                : location.pathname.startsWith(item.to)

            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={clsx(
                  'group flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-primary-600/15 text-primary-400'
                    : 'text-surface-400 hover:bg-surface-800 hover:text-surface-200'
                )}
              >
                <div className="relative">
                  <Icon className="h-4.5 w-4.5 flex-shrink-0" />
                  {item.label === 'Messages' && unreadCount > 0 && (
                    <span className="absolute -right-1.5 -top-1.5 flex h-3.5 min-w-[14px] items-center justify-center rounded-full bg-primary-500 px-1 text-[8px] font-bold text-white">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  )}
                </div>
                {!sidebarCollapsed && <span>{item.label}</span>}
              </NavLink>
            )
          })}
        </nav>

        {/* Footer */}
        <div className="border-t border-surface-700 p-2 space-y-1">
          {/* Sync Status */}
          <div className="flex justify-center px-2 py-1">
            <SyncIndicator />
          </div>

          {/* Current Agent */}
          {currentAgent && !sidebarCollapsed && (
            <div className="flex items-center gap-2 rounded-lg bg-surface-800 px-3 py-2">
              <div
                className={clsx(
                  'flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full text-[10px] font-semibold',
                  currentAgent.status === 'online'
                    ? 'bg-primary-600 text-white'
                    : 'bg-surface-600 text-surface-300'
                )}
              >
                {currentAgent.name[0].toUpperCase()}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-medium text-surface-200">
                  {currentAgent.name}
                </p>
                <p className="text-[10px] capitalize text-surface-500">
                  {currentAgent.type}
                </p>
              </div>
            </div>
          )}

          {/* Collapse & Logout */}
          <div className="flex items-center gap-1">
            <button
              onClick={toggleSidebar}
              className="flex flex-1 items-center justify-center rounded-md py-1.5 text-surface-500 hover:bg-surface-800 hover:text-surface-300"
              title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            >
              {sidebarCollapsed ? (
                <ChevronRight className="h-4 w-4" />
              ) : (
                <ChevronLeft className="h-4 w-4" />
              )}
            </button>
            {!sidebarCollapsed && (
              <>
                <button
                  onClick={() => {}}
                  className="flex flex-1 items-center justify-center rounded-md py-1.5 text-surface-500 hover:bg-surface-800 hover:text-surface-300"
                  title="Settings"
                >
                  <Settings className="h-4 w-4" />
                </button>
                <button
                  onClick={() => {
                    clearAuth()
                    window.location.href = '/join'
                  }}
                  className="flex flex-1 items-center justify-center rounded-md py-1.5 text-surface-500 hover:bg-red-500/10 hover:text-red-400"
                  title="Leave workspace"
                >
                  <LogOut className="h-4 w-4" />
                </button>
              </>
            )}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex flex-1 flex-col overflow-hidden">
        {/* Conflict Banners */}
        {conflicts.length > 0 && (
          <div className="space-y-2 border-b border-surface-700 bg-surface-950 p-3">
            {conflicts.slice(0, 3).map((conflict) => (
              <ConflictResolver
                key={conflict.resource_id}
                conflict={conflict}
                onResolve={handleResolveConflict}
                onDismiss={() => removeConflict(conflict.resource_id)}
              />
            ))}
          </div>
        )}

        {/* Page Content */}
        <div className="flex-1 overflow-auto p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/tasks" element={<TaskBoard />} />
            <Route path="/messages" element={<MessageCenter />} />
            <Route path="/artifacts" element={<ArtifactBrowser />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </main>
    </div>
  )
}

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useStore((s) => s.token)
  const workspaceId = useStore((s) => s.workspaceId)

  if (!token || !workspaceId) {
    return <Navigate to="/join" replace />
  }

  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/join" element={<JoinWorkspace />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <AuthenticatedLayout />
          </RequireAuth>
        }
      />
    </Routes>
  )
}
