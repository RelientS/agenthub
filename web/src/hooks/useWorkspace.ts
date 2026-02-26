import { useEffect, useCallback, useRef } from 'react'
import { useStore } from '@/store'
import * as api from '@/api/client'

export function useWorkspace() {
  const {
    workspaceId,
    agentId,
    setWorkspace,
    setAgents,
    setTasks,
    setMessages,
    setArtifacts,
    setContexts,
    setUnreadCount,
  } = useStore()

  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchAll = useCallback(async () => {
    if (!workspaceId) return

    try {
      const [wsRes, agentsRes, tasksRes, msgsRes, artsRes, ctxRes] =
        await Promise.allSettled([
          api.getWorkspace(workspaceId),
          api.getAgents(workspaceId),
          api.getTasks(workspaceId),
          api.getMessages(workspaceId),
          api.getArtifacts(workspaceId),
          api.getContexts(workspaceId),
        ])

      if (wsRes.status === 'fulfilled') setWorkspace(wsRes.value.data)
      if (agentsRes.status === 'fulfilled') setAgents(agentsRes.value.data)
      if (tasksRes.status === 'fulfilled') setTasks(tasksRes.value.data)
      if (msgsRes.status === 'fulfilled') {
        const msgs = msgsRes.value.data
        setMessages(msgs)
        const unread = msgs.filter(
          (m) => agentId && !m.read_by.includes(agentId)
        ).length
        setUnreadCount(unread)
      }
      if (artsRes.status === 'fulfilled') setArtifacts(artsRes.value.data)
      if (ctxRes.status === 'fulfilled') setContexts(ctxRes.value.data)
    } catch (err) {
      console.error('Failed to fetch workspace data:', err)
    }
  }, [
    workspaceId,
    agentId,
    setWorkspace,
    setAgents,
    setTasks,
    setMessages,
    setArtifacts,
    setContexts,
    setUnreadCount,
  ])

  // Initial data fetch
  useEffect(() => {
    fetchAll()
  }, [fetchAll])

  // Heartbeat
  useEffect(() => {
    if (!workspaceId || !agentId) return

    const sendHeartbeat = () => {
      api.sendHeartbeat(workspaceId, agentId).catch(() => {
        // Silently fail - WS reconnect will handle recovery
      })
    }

    sendHeartbeat()
    heartbeatRef.current = setInterval(sendHeartbeat, 30000)

    return () => {
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current)
      }
    }
  }, [workspaceId, agentId])

  return { refetch: fetchAll }
}
