import { useState } from 'react'
import { clsx } from 'clsx'
import { AlertTriangle, Check, X, GitMerge, ArrowLeft, ArrowRight } from 'lucide-react'
import type { WsConflictEvent } from '@/types'

interface ConflictResolverProps {
  conflict: WsConflictEvent
  onResolve: (
    resourceId: string,
    chosen: 'local' | 'remote' | 'merged',
    mergedData?: string
  ) => void
  onDismiss: () => void
}

export function ConflictResolver({
  conflict,
  onResolve,
  onDismiss,
}: ConflictResolverProps) {
  const [mergedContent, setMergedContent] = useState(conflict.remote_data)
  const [activeTab, setActiveTab] = useState<'side-by-side' | 'merged'>(
    'side-by-side'
  )

  const localLines = conflict.local_data.split('\n')
  const remoteLines = conflict.remote_data.split('\n')

  return (
    <div className="flex flex-col rounded-lg border border-amber-500/30 bg-surface-900">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-surface-700 px-4 py-3">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-amber-400" />
          <h3 className="text-sm font-semibold text-amber-200">
            Sync Conflict
          </h3>
          <span className="rounded bg-surface-700 px-1.5 py-0.5 text-[10px] uppercase text-surface-400">
            {conflict.resource_type}
          </span>
        </div>
        <button
          onClick={onDismiss}
          className="rounded p-1 text-surface-400 hover:bg-surface-800 hover:text-surface-200"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Version info */}
      <div className="flex items-center gap-4 border-b border-surface-700 px-4 py-2 text-xs text-surface-400">
        <span>
          Local version: <strong className="text-surface-200">v{conflict.local_version}</strong>
        </span>
        <span>
          Remote version: <strong className="text-surface-200">v{conflict.remote_version}</strong>
        </span>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-surface-700">
        <button
          onClick={() => setActiveTab('side-by-side')}
          className={clsx(
            'px-4 py-2 text-xs font-medium transition-colors',
            activeTab === 'side-by-side'
              ? 'border-b-2 border-primary-500 text-primary-400'
              : 'text-surface-400 hover:text-surface-200'
          )}
        >
          Side by Side
        </button>
        <button
          onClick={() => setActiveTab('merged')}
          className={clsx(
            'px-4 py-2 text-xs font-medium transition-colors',
            activeTab === 'merged'
              ? 'border-b-2 border-primary-500 text-primary-400'
              : 'text-surface-400 hover:text-surface-200'
          )}
        >
          Merged Editor
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {activeTab === 'side-by-side' ? (
          <div className="grid grid-cols-2 divide-x divide-surface-700">
            {/* Local */}
            <div>
              <div className="sticky top-0 flex items-center gap-1 bg-blue-500/10 px-3 py-1.5 text-xs font-medium text-blue-400">
                <ArrowLeft className="h-3 w-3" />
                Local (yours)
              </div>
              <pre className="p-3 text-xs leading-5">
                {localLines.map((line, i) => {
                  const isDiff = i < remoteLines.length && line !== remoteLines[i]
                  const isExtra = i >= remoteLines.length
                  return (
                    <div
                      key={i}
                      className={clsx(
                        'px-1',
                        isDiff && 'bg-blue-500/10 text-blue-300',
                        isExtra && 'bg-blue-500/10 text-blue-300',
                        !isDiff && !isExtra && 'text-surface-300'
                      )}
                    >
                      <span className="mr-3 inline-block w-4 text-right text-surface-600">
                        {i + 1}
                      </span>
                      {line}
                    </div>
                  )
                })}
              </pre>
            </div>

            {/* Remote */}
            <div>
              <div className="sticky top-0 flex items-center gap-1 bg-green-500/10 px-3 py-1.5 text-xs font-medium text-green-400">
                <ArrowRight className="h-3 w-3" />
                Remote (theirs)
              </div>
              <pre className="p-3 text-xs leading-5">
                {remoteLines.map((line, i) => {
                  const isDiff = i < localLines.length && line !== localLines[i]
                  const isExtra = i >= localLines.length
                  return (
                    <div
                      key={i}
                      className={clsx(
                        'px-1',
                        isDiff && 'bg-green-500/10 text-green-300',
                        isExtra && 'bg-green-500/10 text-green-300',
                        !isDiff && !isExtra && 'text-surface-300'
                      )}
                    >
                      <span className="mr-3 inline-block w-4 text-right text-surface-600">
                        {i + 1}
                      </span>
                      {line}
                    </div>
                  )
                })}
              </pre>
            </div>
          </div>
        ) : (
          <div className="p-3">
            <textarea
              value={mergedContent}
              onChange={(e) => setMergedContent(e.target.value)}
              className="h-64 w-full resize-none rounded-md border border-surface-600 bg-surface-950 p-3 font-mono text-xs text-surface-200 outline-none focus:border-primary-500"
              placeholder="Edit the merged content..."
            />
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between border-t border-surface-700 px-4 py-3">
        <p className="text-[11px] text-surface-500">
          Choose which version to keep, or merge manually
        </p>
        <div className="flex items-center gap-2">
          <button
            onClick={() => onResolve(conflict.resource_id, 'local')}
            className="inline-flex items-center gap-1 rounded-md border border-blue-500/30 bg-blue-500/10 px-3 py-1.5 text-xs font-medium text-blue-400 transition-colors hover:bg-blue-500/20"
          >
            <ArrowLeft className="h-3 w-3" />
            Keep Local
          </button>
          <button
            onClick={() => onResolve(conflict.resource_id, 'remote')}
            className="inline-flex items-center gap-1 rounded-md border border-green-500/30 bg-green-500/10 px-3 py-1.5 text-xs font-medium text-green-400 transition-colors hover:bg-green-500/20"
          >
            <ArrowRight className="h-3 w-3" />
            Keep Remote
          </button>
          {activeTab === 'merged' && (
            <button
              onClick={() =>
                onResolve(conflict.resource_id, 'merged', mergedContent)
              }
              className="inline-flex items-center gap-1 rounded-md bg-primary-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-primary-500"
            >
              <GitMerge className="h-3 w-3" />
              Apply Merged
            </button>
          )}
          <button
            onClick={onDismiss}
            className="inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-xs font-medium text-surface-400 transition-colors hover:bg-surface-800 hover:text-surface-200"
          >
            <Check className="h-3 w-3" />
            Dismiss
          </button>
        </div>
      </div>
    </div>
  )
}
