import { useState, useMemo, useCallback } from 'react'
import { clsx } from 'clsx'
import { formatDistanceToNow } from 'date-fns'
import {
  Search,
  Send,
  Hash,
  MessageSquare,
  Database,
  Globe,
  AlertOctagon,
  CheckCircle,
  GitBranch,
  HelpCircle,
  Code2,
  Activity,
  Reply,
  Users,
  Plus,
} from 'lucide-react'
import { useStore } from '@/store'
import { MessageThreadView } from '@/components/MessageThread'
import * as api from '@/api/client'
import type { Message, MessageType, SendMessageRequest } from '@/types'

const messageTypeOptions: Array<{
  value: MessageType
  label: string
  icon: React.ElementType
  color: string
}> = [
  { value: 'chat', label: 'Chat', icon: MessageSquare, color: 'text-surface-400' },
  { value: 'schema', label: 'Schema', icon: Database, color: 'text-purple-400' },
  { value: 'endpoint', label: 'Endpoint', icon: Globe, color: 'text-blue-400' },
  { value: 'blocker', label: 'Blocker', icon: AlertOctagon, color: 'text-red-400' },
  { value: 'review', label: 'Review', icon: CheckCircle, color: 'text-green-400' },
  { value: 'decision', label: 'Decision', icon: GitBranch, color: 'text-amber-400' },
  { value: 'status_update', label: 'Status', icon: Activity, color: 'text-cyan-400' },
  { value: 'code_snippet', label: 'Code', icon: Code2, color: 'text-emerald-400' },
  { value: 'question', label: 'Question', icon: HelpCircle, color: 'text-yellow-400' },
  { value: 'answer', label: 'Answer', icon: Reply, color: 'text-teal-400' },
]

interface ThreadSummary {
  threadId: string
  title: string
  lastMessage: Message
  messageCount: number
  unreadCount: number
  participants: string[]
}

export function MessageCenter() {
  const workspaceId = useStore((s) => s.workspaceId)
  const agentId = useStore((s) => s.agentId)
  const messages = useStore((s) => s.messages)
  const agents = useStore((s) => s.agents)
  const storeAddMessage = useStore((s) => s.addMessage)
  const setUnreadCount = useStore((s) => s.setUnreadCount)
  const unreadCount = useStore((s) => s.unreadCount)

  const [selectedThread, setSelectedThread] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [messageContent, setMessageContent] = useState('')
  const [messageType, setMessageType] = useState<MessageType>('chat')
  const [sending, setSending] = useState(false)
  const [showTypeSelector, setShowTypeSelector] = useState(false)

  // Group messages into threads
  const threads = useMemo(() => {
    const threadMap = new Map<string, Message[]>()

    // Group messages - use thread_id if available, otherwise group by sender-recipient pair
    for (const msg of messages) {
      const threadId =
        msg.thread_id ||
        [msg.sender_id, msg.recipient_id || 'broadcast'].sort().join(':')
      if (!threadMap.has(threadId)) {
        threadMap.set(threadId, [])
      }
      threadMap.get(threadId)!.push(msg)
    }

    // Build thread summaries
    const summaries: ThreadSummary[] = []
    for (const [threadId, msgs] of threadMap) {
      const sorted = msgs.sort(
        (a, b) =>
          new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      )
      const last = sorted[sorted.length - 1]
      const participantIds = new Set(msgs.map((m) => m.sender_id))
      const unread = msgs.filter(
        (m) => agentId && !m.read_by.includes(agentId)
      ).length

      // Generate thread title
      const participants = Array.from(participantIds)
        .map((id) => agents.find((a) => a.id === id)?.name || 'Unknown')
        .join(', ')

      const isBroadcast = msgs.some((m) => !m.recipient_id)

      summaries.push({
        threadId,
        title: isBroadcast ? 'General' : participants,
        lastMessage: last,
        messageCount: msgs.length,
        unreadCount: unread,
        participants: Array.from(participantIds),
      })
    }

    return summaries.sort(
      (a, b) =>
        new Date(b.lastMessage.created_at).getTime() -
        new Date(a.lastMessage.created_at).getTime()
    )
  }, [messages, agents, agentId])

  const filteredThreads = useMemo(() => {
    if (!searchQuery) return threads
    const q = searchQuery.toLowerCase()
    return threads.filter(
      (t) =>
        t.title.toLowerCase().includes(q) ||
        t.lastMessage.content.toLowerCase().includes(q)
    )
  }, [threads, searchQuery])

  const selectedMessages = useMemo(() => {
    if (!selectedThread) return []
    return messages
      .filter((m) => {
        const threadId =
          m.thread_id ||
          [m.sender_id, m.recipient_id || 'broadcast'].sort().join(':')
        return threadId === selectedThread
      })
      .sort(
        (a, b) =>
          new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      )
  }, [messages, selectedThread])

  const handleSend = useCallback(async () => {
    if (!messageContent.trim() || !workspaceId || sending) return

    setSending(true)
    try {
      const req: SendMessageRequest = {
        message_type: messageType,
        content: messageContent.trim(),
        thread_id: selectedThread || undefined,
      }

      const res = await api.sendMessage(workspaceId, req)
      storeAddMessage(res.data)
      setMessageContent('')
    } catch (err) {
      console.error('Failed to send message:', err)
    } finally {
      setSending(false)
    }
  }, [messageContent, messageType, workspaceId, selectedThread, sending, storeAddMessage])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleMarkRead = useCallback(async () => {
    if (!workspaceId || !selectedThread) return
    const unreadMsgs = selectedMessages.filter(
      (m) => agentId && !m.read_by.includes(agentId)
    )
    for (const msg of unreadMsgs) {
      try {
        await api.markMessageRead(workspaceId, msg.id)
      } catch {
        // Silently fail
      }
    }
    setUnreadCount(Math.max(0, unreadCount - unreadMsgs.length))
  }, [workspaceId, selectedThread, selectedMessages, agentId, setUnreadCount, unreadCount])

  // Mark messages as read when thread is selected
  const selectThread = useCallback(
    (threadId: string) => {
      setSelectedThread(threadId)
      // Defer marking read
      setTimeout(handleMarkRead, 500)
    },
    [handleMarkRead]
  )

  const selectedTypeConfig = messageTypeOptions.find(
    (t) => t.value === messageType
  )!

  return (
    <div className="flex h-full flex-col">
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h1 className="text-xl font-bold text-white">Messages</h1>
          {unreadCount > 0 && (
            <span className="flex h-5 items-center rounded-full bg-purple-500/20 px-2 text-xs font-medium text-purple-300">
              {unreadCount} unread
            </span>
          )}
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden rounded-xl border border-surface-700">
        {/* Thread List */}
        <div className="flex w-80 flex-col border-r border-surface-700 bg-surface-900">
          {/* Search */}
          <div className="border-b border-surface-700 p-3">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-surface-500" />
              <input
                type="text"
                placeholder="Search conversations..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full rounded-md border border-surface-600 bg-surface-800 py-1.5 pl-8 pr-3 text-xs text-surface-200 outline-none focus:border-primary-500"
              />
            </div>
          </div>

          {/* New conversation button */}
          <div className="border-b border-surface-700 p-2">
            <button
              onClick={() => {
                setSelectedThread(null)
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-xs font-medium text-primary-400 transition-colors hover:bg-primary-500/10"
            >
              <Plus className="h-3.5 w-3.5" />
              New Broadcast Message
            </button>
          </div>

          {/* Thread list */}
          <div className="flex-1 overflow-y-auto">
            {filteredThreads.length === 0 ? (
              <div className="flex h-full items-center justify-center text-xs text-surface-500">
                No conversations yet
              </div>
            ) : (
              filteredThreads.map((thread) => {
                const sender = agents.find(
                  (a) => a.id === thread.lastMessage.sender_id
                )
                const isBroadcast = thread.title === 'General'

                return (
                  <div
                    key={thread.threadId}
                    onClick={() => selectThread(thread.threadId)}
                    className={clsx(
                      'cursor-pointer border-b border-surface-800 px-3 py-3 transition-colors',
                      selectedThread === thread.threadId
                        ? 'bg-primary-500/10 border-l-2 border-l-primary-500'
                        : 'hover:bg-surface-800'
                    )}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2 min-w-0">
                        {isBroadcast ? (
                          <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-surface-700">
                            <Users className="h-3 w-3 text-surface-400" />
                          </div>
                        ) : (
                          <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-primary-600 text-[10px] font-semibold text-white">
                            {sender?.name[0]?.toUpperCase() || '?'}
                          </div>
                        )}
                        <div className="min-w-0">
                          <p className="truncate text-xs font-medium text-surface-200">
                            {isBroadcast && (
                              <Hash className="mr-0.5 inline h-3 w-3 text-surface-500" />
                            )}
                            {thread.title}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-1.5 flex-shrink-0">
                        {thread.unreadCount > 0 && (
                          <span className="flex h-4 min-w-[16px] items-center justify-center rounded-full bg-primary-500 px-1 text-[10px] font-medium text-white">
                            {thread.unreadCount}
                          </span>
                        )}
                      </div>
                    </div>
                    <p className="mt-1 truncate text-[11px] text-surface-500 pl-8">
                      {thread.lastMessage.content.slice(0, 80)}
                    </p>
                    <p className="mt-0.5 text-[10px] text-surface-600 pl-8">
                      {formatDistanceToNow(
                        new Date(thread.lastMessage.created_at),
                        { addSuffix: true }
                      )}
                    </p>
                  </div>
                )
              })
            )}
          </div>
        </div>

        {/* Message Area */}
        <div className="flex flex-1 flex-col bg-surface-950">
          {selectedThread || !selectedThread ? (
            <>
              {/* Thread header */}
              <div className="border-b border-surface-700 bg-surface-900 px-4 py-2.5">
                <p className="text-sm font-medium text-surface-200">
                  {selectedThread
                    ? threads.find((t) => t.threadId === selectedThread)
                        ?.title || 'Conversation'
                    : 'Broadcast Channel'}
                </p>
                <p className="text-[11px] text-surface-500">
                  {selectedThread
                    ? `${selectedMessages.length} messages`
                    : 'Send a message to all agents'}
                </p>
              </div>

              {/* Messages */}
              <MessageThreadView
                messages={selectedMessages}
                className="flex-1"
              />

              {/* Compose */}
              <div className="border-t border-surface-700 bg-surface-900 p-3">
                {/* Type selector */}
                <div className="relative mb-2">
                  <button
                    onClick={() => setShowTypeSelector(!showTypeSelector)}
                    className={clsx(
                      'inline-flex items-center gap-1 rounded px-2 py-1 text-[11px] font-medium transition-colors',
                      selectedTypeConfig.color,
                      'bg-surface-800 hover:bg-surface-700'
                    )}
                  >
                    <selectedTypeConfig.icon className="h-3 w-3" />
                    {selectedTypeConfig.label}
                  </button>

                  {showTypeSelector && (
                    <div className="absolute bottom-full left-0 mb-1 z-10 grid grid-cols-2 gap-0.5 rounded-lg border border-surface-600 bg-surface-800 p-1.5 shadow-xl">
                      {messageTypeOptions.map((opt) => {
                        const OptIcon = opt.icon
                        return (
                          <button
                            key={opt.value}
                            onClick={() => {
                              setMessageType(opt.value)
                              setShowTypeSelector(false)
                            }}
                            className={clsx(
                              'flex items-center gap-1.5 rounded px-2 py-1.5 text-[11px] transition-colors',
                              messageType === opt.value
                                ? 'bg-primary-500/20 text-primary-300'
                                : 'text-surface-300 hover:bg-surface-700'
                            )}
                          >
                            <OptIcon className={clsx('h-3 w-3', opt.color)} />
                            {opt.label}
                          </button>
                        )
                      })}
                    </div>
                  )}
                </div>

                {/* Input */}
                <div className="flex items-end gap-2">
                  <textarea
                    value={messageContent}
                    onChange={(e) => setMessageContent(e.target.value)}
                    onKeyDown={handleKeyDown}
                    rows={2}
                    className="flex-1 resize-none rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none placeholder:text-surface-500 focus:border-primary-500"
                    placeholder={`Send a ${messageType.replace('_', ' ')} message...`}
                  />
                  <button
                    onClick={handleSend}
                    disabled={!messageContent.trim() || sending}
                    className="flex h-10 w-10 items-center justify-center rounded-md bg-primary-600 text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
                  >
                    <Send className="h-4 w-4" />
                  </button>
                </div>
                <p className="mt-1 text-[10px] text-surface-600">
                  Press Enter to send, Shift+Enter for new line
                </p>
              </div>
            </>
          ) : (
            <div className="flex flex-1 items-center justify-center">
              <div className="text-center">
                <MessageSquare className="mx-auto mb-2 h-8 w-8 text-surface-600" />
                <p className="text-sm text-surface-400">
                  Select a conversation to view messages
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
