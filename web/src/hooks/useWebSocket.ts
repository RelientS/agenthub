import { useEffect, useRef, useCallback } from 'react'
import ReconnectingWebSocket from 'reconnecting-websocket'
import { useStore } from '@/store'
import type {
  WsEvent,
  WsAgentEvent,
  WsTaskEvent,
  WsMessageEvent,
  WsArtifactEvent,
  WsConflictEvent,
} from '@/types'

export function useWebSocket() {
  const wsRef = useRef<ReconnectingWebSocket | null>(null)
  const {
    token,
    workspaceId,
    setWsConnected,
    updateAgentStatus,
    addOrUpdateAgent,
    addTask,
    updateTask,
    removeTask,
    addMessage,
    addArtifact,
    updateArtifact,
    addConflict,
  } = useStore()

  const handleEvent = useCallback(
    (event: WsEvent) => {
      switch (event.type) {
        case 'agent_online':
        case 'agent_status_changed': {
          const payload = event.payload as WsAgentEvent
          if (payload.agent) {
            addOrUpdateAgent(payload.agent)
          } else {
            updateAgentStatus(payload.agent_id, payload.status)
          }
          break
        }
        case 'agent_offline': {
          const payload = event.payload as WsAgentEvent
          updateAgentStatus(payload.agent_id, 'offline')
          break
        }
        case 'task_created': {
          const payload = event.payload as WsTaskEvent
          addTask(payload.task)
          break
        }
        case 'task_updated': {
          const payload = event.payload as WsTaskEvent
          updateTask(payload.task)
          break
        }
        case 'task_deleted': {
          const payload = event.payload as { task_id: string }
          removeTask(payload.task_id)
          break
        }
        case 'message_received': {
          const payload = event.payload as WsMessageEvent
          addMessage(payload.message)
          break
        }
        case 'artifact_created': {
          const payload = event.payload as WsArtifactEvent
          addArtifact(payload.artifact)
          break
        }
        case 'artifact_updated': {
          const payload = event.payload as WsArtifactEvent
          updateArtifact(payload.artifact)
          break
        }
        case 'sync_conflict': {
          const payload = event.payload as WsConflictEvent
          addConflict(payload)
          break
        }
        case 'heartbeat':
          break
        default:
          console.warn('Unknown WS event type:', event.type)
      }
    },
    [
      addOrUpdateAgent,
      updateAgentStatus,
      addTask,
      updateTask,
      removeTask,
      addMessage,
      addArtifact,
      updateArtifact,
      addConflict,
    ]
  )

  useEffect(() => {
    if (!token || !workspaceId) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/ws?token=${encodeURIComponent(token)}&workspace_id=${workspaceId}`

    const ws = new ReconnectingWebSocket(url, [], {
      maxRetries: 20,
      connectionTimeout: 5000,
      maxReconnectionDelay: 10000,
    })

    ws.onopen = () => {
      console.log('[WS] Connected')
      setWsConnected(true)
    }

    ws.onclose = () => {
      console.log('[WS] Disconnected')
      setWsConnected(false)
    }

    ws.onerror = (err) => {
      console.error('[WS] Error:', err)
    }

    ws.onmessage = (evt) => {
      try {
        const event: WsEvent = JSON.parse(evt.data as string)
        handleEvent(event)
      } catch (err) {
        console.error('[WS] Failed to parse message:', err)
      }
    }

    wsRef.current = ws

    return () => {
      ws.close()
      wsRef.current = null
      setWsConnected(false)
    }
  }, [token, workspaceId, handleEvent, setWsConnected])

  const send = useCallback((data: Record<string, unknown>) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    }
  }, [])

  return { send }
}
