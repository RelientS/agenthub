import { useState, useMemo } from 'react'
import { clsx } from 'clsx'
import {
  Search,
  Grid3X3,
  List,
  Plus,
  X,
  Filter,
  FileCode,
  Settings,
  Database,
  FileText,
  TestTube,
  Globe,
  ArrowUpDown,
  Clock,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { useStore } from '@/store'
import { ArtifactViewer, ArtifactCard } from '@/components/ArtifactViewer'
import * as api from '@/api/client'
import type { Artifact, ArtifactType, CreateArtifactRequest } from '@/types'

const artifactTypeOptions: Array<{
  value: ArtifactType
  label: string
  icon: React.ElementType
  color: string
}> = [
  { value: 'code', label: 'Code', icon: FileCode, color: 'text-blue-400' },
  { value: 'config', label: 'Config', icon: Settings, color: 'text-yellow-400' },
  { value: 'schema', label: 'Schema', icon: Database, color: 'text-purple-400' },
  { value: 'document', label: 'Document', icon: FileText, color: 'text-green-400' },
  { value: 'test', label: 'Test', icon: TestTube, color: 'text-orange-400' },
  { value: 'api_spec', label: 'API Spec', icon: Globe, color: 'text-cyan-400' },
  { value: 'migration', label: 'Migration', icon: ArrowUpDown, color: 'text-red-400' },
]

const languageOptions = [
  'typescript',
  'javascript',
  'go',
  'python',
  'json',
  'yaml',
  'sql',
  'css',
  'bash',
  'text',
]

export function ArtifactBrowser() {
  const workspaceId = useStore((s) => s.workspaceId)
  const artifacts = useStore((s) => s.artifacts)
  const agents = useStore((s) => s.agents)
  const storeAddArtifact = useStore((s) => s.addArtifact)

  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [searchQuery, setSearchQuery] = useState('')
  const [filterType, setFilterType] = useState<string>('')
  const [filterLanguage, setFilterLanguage] = useState<string>('')
  const [filterTag, setFilterTag] = useState<string>('')
  const [showFilters, setShowFilters] = useState(false)
  const [selectedArtifact, setSelectedArtifact] = useState<Artifact | null>(null)
  const [showCreateForm, setShowCreateForm] = useState(false)

  const allTags = useMemo(() => {
    const tagSet = new Set<string>()
    artifacts.forEach((a) => a.tags.forEach((tag) => tagSet.add(tag)))
    return Array.from(tagSet).sort()
  }, [artifacts])

  const allLanguages = useMemo(() => {
    const langSet = new Set<string>()
    artifacts.forEach((a) => {
      if (a.language) langSet.add(a.language)
    })
    return Array.from(langSet).sort()
  }, [artifacts])

  const filteredArtifacts = useMemo(() => {
    let result = artifacts

    if (filterType) {
      result = result.filter((a) => a.type === filterType)
    }
    if (filterLanguage) {
      result = result.filter((a) => a.language === filterLanguage)
    }
    if (filterTag) {
      result = result.filter((a) => a.tags.includes(filterTag))
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      result = result.filter(
        (a) =>
          a.name.toLowerCase().includes(q) ||
          a.content.toLowerCase().includes(q) ||
          a.tags.some((t) => t.toLowerCase().includes(q))
      )
    }

    return result.sort(
      (a, b) =>
        new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    )
  }, [artifacts, filterType, filterLanguage, filterTag, searchQuery])

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h1 className="text-xl font-bold text-white">Artifacts</h1>
          <span className="rounded bg-surface-700 px-2 py-0.5 text-xs text-surface-400">
            {artifacts.length} total
          </span>
        </div>
        <div className="flex items-center gap-2">
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-surface-500" />
            <input
              type="text"
              placeholder="Search artifacts..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-8 w-56 rounded-md border border-surface-600 bg-surface-800 pl-8 pr-3 text-xs text-surface-200 outline-none focus:border-primary-500"
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
          {/* View mode toggle */}
          <div className="flex rounded-md border border-surface-600">
            <button
              onClick={() => setViewMode('grid')}
              className={clsx(
                'flex h-8 w-8 items-center justify-center rounded-l-md transition-colors',
                viewMode === 'grid'
                  ? 'bg-surface-700 text-surface-200'
                  : 'text-surface-500 hover:text-surface-300'
              )}
            >
              <Grid3X3 className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={clsx(
                'flex h-8 w-8 items-center justify-center rounded-r-md transition-colors',
                viewMode === 'list'
                  ? 'bg-surface-700 text-surface-200'
                  : 'text-surface-500 hover:text-surface-300'
              )}
            >
              <List className="h-3.5 w-3.5" />
            </button>
          </div>
          {/* Create */}
          <button
            onClick={() => setShowCreateForm(true)}
            className="flex h-8 items-center gap-1.5 rounded-md bg-primary-600 px-3 text-xs font-medium text-white transition-colors hover:bg-primary-500"
          >
            <Plus className="h-3.5 w-3.5" />
            New Artifact
          </button>
        </div>
      </div>

      {/* Filter Bar */}
      {showFilters && (
        <div className="mb-4 flex items-center gap-3 rounded-lg border border-surface-700 bg-surface-800/50 px-4 py-2.5">
          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value)}
            className="h-7 rounded border border-surface-600 bg-surface-800 px-2 text-xs text-surface-300 outline-none"
          >
            <option value="">All Types</option>
            {artifactTypeOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
          <select
            value={filterLanguage}
            onChange={(e) => setFilterLanguage(e.target.value)}
            className="h-7 rounded border border-surface-600 bg-surface-800 px-2 text-xs text-surface-300 outline-none"
          >
            <option value="">All Languages</option>
            {allLanguages.map((lang) => (
              <option key={lang} value={lang}>
                {lang}
              </option>
            ))}
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
          {(filterType || filterLanguage || filterTag) && (
            <button
              onClick={() => {
                setFilterType('')
                setFilterLanguage('')
                setFilterTag('')
              }}
              className="flex items-center gap-1 text-xs text-surface-400 hover:text-surface-200"
            >
              <X className="h-3 w-3" />
              Clear
            </button>
          )}
          <span className="ml-auto text-xs text-surface-500">
            {filteredArtifacts.length} result{filteredArtifacts.length !== 1 ? 's' : ''}
          </span>
        </div>
      )}

      {/* Content */}
      <div className="flex flex-1 gap-4 overflow-hidden">
        {/* Artifact List */}
        <div
          className={clsx(
            'flex-1 overflow-y-auto',
            selectedArtifact ? 'max-w-[50%]' : ''
          )}
        >
          {filteredArtifacts.length === 0 ? (
            <div className="flex h-64 items-center justify-center rounded-xl border border-surface-700 bg-surface-800/50">
              <div className="text-center">
                <FileCode className="mx-auto mb-2 h-8 w-8 text-surface-600" />
                <p className="text-sm text-surface-400">
                  {artifacts.length === 0
                    ? 'No artifacts yet'
                    : 'No artifacts match your filters'}
                </p>
              </div>
            </div>
          ) : viewMode === 'grid' ? (
            <div
              className={clsx(
                'grid gap-3',
                selectedArtifact
                  ? 'grid-cols-1'
                  : 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3'
              )}
            >
              {filteredArtifacts.map((artifact) => (
                <ArtifactCard
                  key={artifact.id}
                  artifact={artifact}
                  onClick={setSelectedArtifact}
                  viewMode="grid"
                />
              ))}
            </div>
          ) : (
            <div className="space-y-2">
              {filteredArtifacts.map((artifact) => (
                <ArtifactCard
                  key={artifact.id}
                  artifact={artifact}
                  onClick={setSelectedArtifact}
                  viewMode="list"
                />
              ))}
            </div>
          )}
        </div>

        {/* Artifact Detail */}
        {selectedArtifact && (
          <div className="flex w-[50%] flex-col gap-3">
            <ArtifactViewer
              artifact={selectedArtifact}
              onClose={() => setSelectedArtifact(null)}
              className="flex-1"
            />
            {/* Version History */}
            <div className="rounded-lg border border-surface-700 bg-surface-900 p-3">
              <h4 className="mb-2 text-xs font-semibold uppercase text-surface-400">
                Details
              </h4>
              <div className="space-y-1.5 text-xs">
                <div className="flex justify-between">
                  <span className="text-surface-500">Creator</span>
                  <span className="text-surface-300">
                    {agents.find((a) => a.id === selectedArtifact.creator_id)
                      ?.name || selectedArtifact.creator_id}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-surface-500">Version</span>
                  <span className="text-surface-300">
                    v{selectedArtifact.version}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-surface-500">Updated</span>
                  <span className="flex items-center gap-1 text-surface-300">
                    <Clock className="h-3 w-3" />
                    {formatDistanceToNow(
                      new Date(selectedArtifact.updated_at),
                      { addSuffix: true }
                    )}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-surface-500">Created</span>
                  <span className="text-surface-300">
                    {new Date(selectedArtifact.created_at).toLocaleDateString()}
                  </span>
                </div>
                {selectedArtifact.dependencies.length > 0 && (
                  <div>
                    <span className="text-surface-500">Dependencies</span>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {selectedArtifact.dependencies.map((dep) => (
                        <span
                          key={dep}
                          className="rounded bg-surface-800 px-1.5 py-0.5 text-surface-400"
                        >
                          {dep}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Create Artifact Modal */}
      {showCreateForm && (
        <CreateArtifactModal
          onClose={() => setShowCreateForm(false)}
          onCreate={async (req) => {
            if (!workspaceId) return
            try {
              const res = await api.createArtifact(workspaceId, req)
              storeAddArtifact(res.data)
              setShowCreateForm(false)
            } catch (err) {
              console.error('Failed to create artifact:', err)
            }
          }}
        />
      )}
    </div>
  )
}

interface CreateArtifactModalProps {
  onClose: () => void
  onCreate: (req: CreateArtifactRequest) => Promise<void>
}

function CreateArtifactModal({ onClose, onCreate }: CreateArtifactModalProps) {
  const [name, setName] = useState('')
  const [type, setType] = useState<ArtifactType>('code')
  const [language, setLanguage] = useState('typescript')
  const [content, setContent] = useState('')
  const [tags, setTags] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || !content.trim()) return
    setSubmitting(true)
    await onCreate({
      name: name.trim(),
      type,
      language,
      content,
      tags: tags
        ? tags.split(',').map((t) => t.trim()).filter(Boolean)
        : undefined,
    })
    setSubmitting(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-2xl rounded-xl border border-surface-700 bg-surface-900 shadow-2xl">
        <div className="flex items-center justify-between border-b border-surface-700 px-5 py-3.5">
          <h2 className="text-sm font-semibold text-white">
            Create New Artifact
          </h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-surface-400 hover:bg-surface-800 hover:text-surface-200"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4 p-5">
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-surface-400">
                Name *
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
                placeholder="artifact-name.ts"
                autoFocus
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-surface-400">
                Type
              </label>
              <select
                value={type}
                onChange={(e) => setType(e.target.value as ArtifactType)}
                className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
              >
                {artifactTypeOptions.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-surface-400">
                Language
              </label>
              <select
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2 text-sm text-surface-200 outline-none focus:border-primary-500"
              >
                {languageOptions.map((lang) => (
                  <option key={lang} value={lang}>
                    {lang}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-surface-400">
              Content *
            </label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              rows={12}
              className="w-full resize-none rounded-md border border-surface-600 bg-surface-800 px-3 py-2 font-mono text-xs text-surface-200 outline-none focus:border-primary-500"
              placeholder="// Paste or write your code here..."
            />
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
              placeholder="api, auth, middleware..."
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
              disabled={!name.trim() || !content.trim() || submitting}
              className="rounded-md bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
            >
              {submitting ? 'Creating...' : 'Create Artifact'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
