import { useRef, useEffect } from 'react'
import { clsx } from 'clsx'
import { formatDistanceToNow } from 'date-fns'
import Prism from 'prismjs'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-bash'
import type { Message, MessageType } from '@/types'
import { useStore } from '@/store'
import {
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
} from 'lucide-react'

const messageTypeConfig: Record<
  MessageType,
  { icon: React.ElementType; color: string; bg: string; label: string }
> = {
  chat: {
    icon: MessageSquare,
    color: 'text-surface-400',
    bg: 'bg-surface-700/50',
    label: 'Chat',
  },
  schema: {
    icon: Database,
    color: 'text-purple-400',
    bg: 'bg-purple-500/10',
    label: 'Schema',
  },
  endpoint: {
    icon: Globe,
    color: 'text-blue-400',
    bg: 'bg-blue-500/10',
    label: 'Endpoint',
  },
  blocker: {
    icon: AlertOctagon,
    color: 'text-red-400',
    bg: 'bg-red-500/10',
    label: 'Blocker',
  },
  review: {
    icon: CheckCircle,
    color: 'text-green-400',
    bg: 'bg-green-500/10',
    label: 'Review',
  },
  decision: {
    icon: GitBranch,
    color: 'text-amber-400',
    bg: 'bg-amber-500/10',
    label: 'Decision',
  },
  status_update: {
    icon: Activity,
    color: 'text-cyan-400',
    bg: 'bg-cyan-500/10',
    label: 'Status',
  },
  code_snippet: {
    icon: Code2,
    color: 'text-emerald-400',
    bg: 'bg-emerald-500/10',
    label: 'Code',
  },
  question: {
    icon: HelpCircle,
    color: 'text-yellow-400',
    bg: 'bg-yellow-500/10',
    label: 'Question',
  },
  answer: {
    icon: Reply,
    color: 'text-teal-400',
    bg: 'bg-teal-500/10',
    label: 'Answer',
  },
}

function renderContent(content: string, messageType: MessageType) {
  // Parse code blocks
  const parts = content.split(/(```\w*\n[\s\S]*?\n```)/g)

  return parts.map((part, i) => {
    const codeMatch = part.match(/```(\w*)\n([\s\S]*?)\n```/)
    if (codeMatch) {
      const lang = codeMatch[1] || 'text'
      const code = codeMatch[2]
      let highlighted = code
      try {
        const grammar = Prism.languages[lang]
        if (grammar) {
          highlighted = Prism.highlight(code, grammar, lang)
        }
      } catch {
        // Fall back to plain text
      }

      return (
        <pre
          key={i}
          className="my-2 overflow-x-auto rounded-md bg-surface-950 p-3 text-xs"
        >
          <div className="mb-1 text-[10px] uppercase text-surface-500">{lang}</div>
          <code
            className={`language-${lang}`}
            dangerouslySetInnerHTML={{ __html: highlighted }}
          />
        </pre>
      )
    }

    if (messageType === 'code_snippet' && !content.includes('```')) {
      let highlighted = part
      try {
        highlighted = Prism.highlight(
          part,
          Prism.languages.typescript,
          'typescript'
        )
      } catch {
        // Fall back
      }
      return (
        <pre
          key={i}
          className="my-1 overflow-x-auto rounded-md bg-surface-950 p-3 text-xs"
        >
          <code dangerouslySetInnerHTML={{ __html: highlighted }} />
        </pre>
      )
    }

    return (
      <p key={i} className="whitespace-pre-wrap text-sm text-surface-200">
        {part}
      </p>
    )
  })
}

interface MessageBubbleProps {
  message: Message
  isOwn: boolean
}

function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  const agents = useStore((s) => s.agents)
  const sender = agents.find((a) => a.id === message.sender_id)
  const config = messageTypeConfig[message.message_type]
  const Icon = config.icon

  return (
    <div
      className={clsx('flex gap-2', isOwn ? 'flex-row-reverse' : 'flex-row')}
    >
      {/* Avatar */}
      <div
        className={clsx(
          'flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full text-xs font-semibold',
          sender?.status === 'online'
            ? 'bg-primary-600 text-white'
            : 'bg-surface-700 text-surface-300'
        )}
      >
        {sender?.name[0]?.toUpperCase() || '?'}
      </div>

      {/* Bubble */}
      <div
        className={clsx(
          'max-w-[75%] rounded-lg border px-3 py-2',
          isOwn
            ? 'border-primary-700/50 bg-primary-900/30'
            : 'border-surface-700 bg-surface-800',
          config.bg
        )}
      >
        {/* Header */}
        <div className="mb-1 flex items-center gap-2">
          <span className="text-xs font-medium text-surface-300">
            {sender?.name || 'Unknown'}
          </span>
          <span
            className={clsx(
              'inline-flex items-center gap-0.5 rounded px-1 py-0.5 text-[10px] font-medium',
              config.color
            )}
          >
            <Icon className="h-2.5 w-2.5" />
            {config.label}
          </span>
          <span className="text-[10px] text-surface-500">
            {formatDistanceToNow(new Date(message.created_at), {
              addSuffix: true,
            })}
          </span>
        </div>

        {/* Content */}
        <div>{renderContent(message.content, message.message_type)}</div>

        {/* Metadata */}
        {Object.keys(message.metadata).length > 0 && (
          <div className="mt-2 rounded bg-surface-900/50 p-2">
            {Object.entries(message.metadata).map(([key, value]) => (
              <div key={key} className="flex gap-2 text-[10px]">
                <span className="font-medium text-surface-400">{key}:</span>
                <span className="text-surface-300">{value}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

interface MessageThreadViewProps {
  messages: Message[]
  className?: string
}

export function MessageThreadView({
  messages,
  className,
}: MessageThreadViewProps) {
  const agentId = useStore((s) => s.agentId)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages])

  return (
    <div
      ref={scrollRef}
      className={clsx('flex flex-col gap-3 overflow-y-auto p-4', className)}
    >
      {messages.length === 0 ? (
        <div className="flex flex-1 items-center justify-center text-sm text-surface-500">
          No messages yet. Start a conversation!
        </div>
      ) : (
        messages.map((msg) => (
          <MessageBubble
            key={msg.id}
            message={msg}
            isOwn={msg.sender_id === agentId}
          />
        ))
      )}
    </div>
  )
}
