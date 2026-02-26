import { useEffect, useRef } from 'react'
import { clsx } from 'clsx'
import Prism from 'prismjs'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-yaml'
import 'prismjs/components/prism-sql'
import 'prismjs/components/prism-css'
import {
  Copy,
  Download,
  FileCode,
  FileText,
  Settings,
  Database,
  TestTube,
  Globe,
  ArrowUpDown,
  X,
} from 'lucide-react'
import type { Artifact, ArtifactType } from '@/types'

const typeConfig: Record<
  ArtifactType,
  { icon: React.ElementType; color: string; label: string }
> = {
  code: { icon: FileCode, color: 'text-blue-400', label: 'Code' },
  config: { icon: Settings, color: 'text-yellow-400', label: 'Config' },
  schema: { icon: Database, color: 'text-purple-400', label: 'Schema' },
  document: { icon: FileText, color: 'text-green-400', label: 'Document' },
  test: { icon: TestTube, color: 'text-orange-400', label: 'Test' },
  api_spec: { icon: Globe, color: 'text-cyan-400', label: 'API Spec' },
  migration: { icon: ArrowUpDown, color: 'text-red-400', label: 'Migration' },
}

const languageMap: Record<string, string> = {
  ts: 'typescript',
  tsx: 'typescript',
  js: 'javascript',
  jsx: 'javascript',
  py: 'python',
  go: 'go',
  json: 'json',
  yaml: 'yaml',
  yml: 'yaml',
  sql: 'sql',
  css: 'css',
  sh: 'bash',
  bash: 'bash',
}

interface ArtifactViewerProps {
  artifact: Artifact
  onClose?: () => void
  className?: string
}

export function ArtifactViewer({
  artifact,
  onClose,
  className,
}: ArtifactViewerProps) {
  const codeRef = useRef<HTMLElement>(null)
  const config = typeConfig[artifact.type]
  const Icon = config.icon
  const lang = languageMap[artifact.language] || artifact.language || 'text'

  useEffect(() => {
    if (codeRef.current) {
      const grammar = Prism.languages[lang]
      if (grammar) {
        codeRef.current.innerHTML = Prism.highlight(
          artifact.content,
          grammar,
          lang
        )
      } else {
        codeRef.current.textContent = artifact.content
      }
    }
  }, [artifact.content, lang])

  const copyContent = () => {
    navigator.clipboard.writeText(artifact.content)
  }

  const downloadContent = () => {
    const blob = new Blob([artifact.content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = artifact.name
    a.click()
    URL.revokeObjectURL(url)
  }

  const lineCount = artifact.content.split('\n').length

  return (
    <div
      className={clsx(
        'flex flex-col rounded-lg border border-surface-700 bg-surface-900',
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b border-surface-700 px-4 py-2.5">
        <div className="flex items-center gap-2">
          <Icon className={clsx('h-4 w-4', config.color)} />
          <span className="text-sm font-medium text-surface-100">
            {artifact.name}
          </span>
          <span className="rounded bg-surface-700 px-1.5 py-0.5 text-[10px] uppercase text-surface-400">
            {lang}
          </span>
          <span className="text-xs text-surface-500">v{artifact.version}</span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={copyContent}
            className="rounded p-1.5 text-surface-400 hover:bg-surface-700 hover:text-surface-200"
            title="Copy content"
          >
            <Copy className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={downloadContent}
            className="rounded p-1.5 text-surface-400 hover:bg-surface-700 hover:text-surface-200"
            title="Download"
          >
            <Download className="h-3.5 w-3.5" />
          </button>
          {onClose && (
            <button
              onClick={onClose}
              className="rounded p-1.5 text-surface-400 hover:bg-surface-700 hover:text-surface-200"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Code */}
      <div className="relative flex-1 overflow-auto">
        <div className="flex">
          {/* Line numbers */}
          <div className="sticky left-0 flex-shrink-0 select-none border-r border-surface-800 bg-surface-900 px-3 py-3 text-right">
            {Array.from({ length: lineCount }, (_, i) => (
              <div
                key={i}
                className="text-xs leading-5 text-surface-600"
              >
                {i + 1}
              </div>
            ))}
          </div>
          {/* Code content */}
          <pre className="flex-1 overflow-x-auto p-3">
            <code ref={codeRef} className={`language-${lang} text-xs leading-5`}>
              {artifact.content}
            </code>
          </pre>
        </div>
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between border-t border-surface-700 px-4 py-1.5 text-[10px] text-surface-500">
        <div className="flex items-center gap-3">
          <span>{lineCount} lines</span>
          <span>{artifact.content.length} chars</span>
        </div>
        <div className="flex items-center gap-2">
          {artifact.tags.map((tag) => (
            <span
              key={tag}
              className="rounded bg-surface-800 px-1.5 py-0.5 text-surface-400"
            >
              {tag}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}

interface ArtifactCardProps {
  artifact: Artifact
  onClick?: (artifact: Artifact) => void
  viewMode?: 'grid' | 'list'
}

export function ArtifactCard({
  artifact,
  onClick,
  viewMode = 'grid',
}: ArtifactCardProps) {
  const config = typeConfig[artifact.type]
  const Icon = config.icon

  if (viewMode === 'list') {
    return (
      <div
        onClick={() => onClick?.(artifact)}
        className="flex cursor-pointer items-center gap-4 rounded-lg border border-surface-700 bg-surface-800 px-4 py-3 transition-colors hover:border-surface-500 hover:bg-surface-750"
      >
        <Icon className={clsx('h-5 w-5 flex-shrink-0', config.color)} />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-surface-100">
            {artifact.name}
          </p>
          <p className="text-xs text-surface-400">
            {artifact.language} &middot; v{artifact.version} &middot;{' '}
            {artifact.content.split('\n').length} lines
          </p>
        </div>
        <div className="flex items-center gap-2">
          {artifact.tags.slice(0, 2).map((tag) => (
            <span
              key={tag}
              className="rounded bg-surface-700 px-1.5 py-0.5 text-[10px] text-surface-400"
            >
              {tag}
            </span>
          ))}
          <span className="rounded bg-surface-700 px-1.5 py-0.5 text-[10px] uppercase text-surface-400">
            {config.label}
          </span>
        </div>
      </div>
    )
  }

  return (
    <div
      onClick={() => onClick?.(artifact)}
      className="cursor-pointer rounded-lg border border-surface-700 bg-surface-800 p-4 transition-colors hover:border-surface-500 hover:bg-surface-750"
    >
      <div className="mb-3 flex items-center gap-2">
        <Icon className={clsx('h-5 w-5', config.color)} />
        <span className="rounded bg-surface-700 px-1.5 py-0.5 text-[10px] uppercase text-surface-400">
          {config.label}
        </span>
      </div>
      <h4 className="mb-1 truncate text-sm font-medium text-surface-100">
        {artifact.name}
      </h4>
      <p className="mb-3 text-xs text-surface-400">
        {artifact.language} &middot; v{artifact.version} &middot;{' '}
        {artifact.content.split('\n').length} lines
      </p>
      {/* Code preview */}
      <div className="overflow-hidden rounded bg-surface-950 p-2">
        <pre className="text-[10px] leading-4 text-surface-400 line-clamp-4">
          {artifact.content.slice(0, 300)}
        </pre>
      </div>
      {artifact.tags.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {artifact.tags.slice(0, 3).map((tag) => (
            <span
              key={tag}
              className="rounded bg-surface-700 px-1.5 py-0.5 text-[10px] text-surface-400"
            >
              {tag}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
