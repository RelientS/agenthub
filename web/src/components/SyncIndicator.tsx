import { clsx } from 'clsx'
import { Wifi, WifiOff, RefreshCw, AlertCircle } from 'lucide-react'
import { useStore } from '@/store'

type SyncStatus = 'synced' | 'syncing' | 'error' | 'disconnected'

interface SyncIndicatorProps {
  className?: string
}

export function SyncIndicator({ className }: SyncIndicatorProps) {
  const wsConnected = useStore((s) => s.wsConnected)
  const conflicts = useStore((s) => s.conflicts)

  let status: SyncStatus = 'synced'
  if (!wsConnected) status = 'disconnected'
  else if (conflicts.length > 0) status = 'error'

  const config: Record<
    SyncStatus,
    {
      icon: React.ElementType
      color: string
      bg: string
      label: string
      animate?: boolean
    }
  > = {
    synced: {
      icon: Wifi,
      color: 'text-emerald-400',
      bg: 'bg-emerald-500/10',
      label: 'Synced',
    },
    syncing: {
      icon: RefreshCw,
      color: 'text-blue-400',
      bg: 'bg-blue-500/10',
      label: 'Syncing...',
      animate: true,
    },
    error: {
      icon: AlertCircle,
      color: 'text-amber-400',
      bg: 'bg-amber-500/10',
      label: `${conflicts.length} conflict${conflicts.length !== 1 ? 's' : ''}`,
    },
    disconnected: {
      icon: WifiOff,
      color: 'text-red-400',
      bg: 'bg-red-500/10',
      label: 'Disconnected',
    },
  }

  const { icon: Icon, color, bg, label, animate } = config[status]

  return (
    <div
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1',
        bg,
        status === 'error'
          ? 'border-amber-500/30'
          : status === 'disconnected'
            ? 'border-red-500/30'
            : 'border-transparent',
        className
      )}
    >
      <Icon
        className={clsx('h-3 w-3', color, animate && 'animate-spin')}
      />
      <span className={clsx('text-[11px] font-medium', color)}>{label}</span>
    </div>
  )
}
