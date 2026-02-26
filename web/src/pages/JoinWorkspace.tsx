import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Boxes, Plus, ArrowRight, Loader2 } from 'lucide-react'
import { clsx } from 'clsx'
import { useStore } from '@/store'
import * as api from '@/api/client'
import type { Agent } from '@/types'

export function JoinWorkspace() {
  const navigate = useNavigate()
  const setAuth = useStore((s) => s.setAuth)

  const [mode, setMode] = useState<'join' | 'create'>('join')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Join fields
  const [workspaceId, setWorkspaceId] = useState('')
  const [agentName, setAgentName] = useState('')
  const [agentType, setAgentType] = useState<Agent['type']>('human')

  // Create fields
  const [wsName, setWsName] = useState('')
  const [wsDescription, setWsDescription] = useState('')

  const handleJoin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!workspaceId.trim() || !agentName.trim()) return

    setLoading(true)
    setError('')
    try {
      const res = await api.joinWorkspace({
        workspace_id: workspaceId.trim(),
        agent_name: agentName.trim(),
        agent_type: agentType,
      })
      const { token, agent, workspace } = res.data
      setAuth(token, agent.id, workspace.id)
      navigate('/')
    } catch (err) {
      setError('Failed to join workspace. Check the workspace ID and try again.')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!wsName.trim() || !agentName.trim()) return

    setLoading(true)
    setError('')
    try {
      // Create workspace first
      const wsRes = await api.createWorkspace({
        name: wsName.trim(),
        description: wsDescription.trim(),
      })
      // Then join it
      const joinRes = await api.joinWorkspace({
        workspace_id: wsRes.data.id,
        agent_name: agentName.trim(),
        agent_type: agentType,
      })
      const { token, agent, workspace } = joinRes.data
      setAuth(token, agent.id, workspace.id)
      navigate('/')
    } catch (err) {
      setError('Failed to create workspace. Please try again.')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-950 p-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-600 shadow-lg shadow-primary-500/20">
            <Boxes className="h-7 w-7 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-white">AgentHub</h1>
          <p className="mt-1 text-sm text-surface-400">
            Multi-agent collaboration platform
          </p>
        </div>

        {/* Mode Tabs */}
        <div className="mb-6 flex rounded-lg border border-surface-700 bg-surface-900 p-1">
          <button
            onClick={() => setMode('join')}
            className={clsx(
              'flex flex-1 items-center justify-center gap-1.5 rounded-md py-2 text-sm font-medium transition-colors',
              mode === 'join'
                ? 'bg-primary-600 text-white'
                : 'text-surface-400 hover:text-surface-200'
            )}
          >
            <ArrowRight className="h-4 w-4" />
            Join Workspace
          </button>
          <button
            onClick={() => setMode('create')}
            className={clsx(
              'flex flex-1 items-center justify-center gap-1.5 rounded-md py-2 text-sm font-medium transition-colors',
              mode === 'create'
                ? 'bg-primary-600 text-white'
                : 'text-surface-400 hover:text-surface-200'
            )}
          >
            <Plus className="h-4 w-4" />
            Create Workspace
          </button>
        </div>

        {/* Error */}
        {error && (
          <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-2.5 text-sm text-red-300">
            {error}
          </div>
        )}

        {/* Form */}
        <form
          onSubmit={mode === 'join' ? handleJoin : handleCreate}
          className="space-y-4 rounded-xl border border-surface-700 bg-surface-900 p-6"
        >
          {mode === 'join' ? (
            <div>
              <label className="mb-1 block text-xs font-medium text-surface-400">
                Workspace ID
              </label>
              <input
                type="text"
                value={workspaceId}
                onChange={(e) => setWorkspaceId(e.target.value)}
                className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2.5 text-sm text-surface-200 outline-none focus:border-primary-500"
                placeholder="Enter workspace ID..."
                required
              />
            </div>
          ) : (
            <>
              <div>
                <label className="mb-1 block text-xs font-medium text-surface-400">
                  Workspace Name
                </label>
                <input
                  type="text"
                  value={wsName}
                  onChange={(e) => setWsName(e.target.value)}
                  className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2.5 text-sm text-surface-200 outline-none focus:border-primary-500"
                  placeholder="My Project Workspace"
                  required
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-surface-400">
                  Description
                </label>
                <textarea
                  value={wsDescription}
                  onChange={(e) => setWsDescription(e.target.value)}
                  rows={2}
                  className="w-full resize-none rounded-md border border-surface-600 bg-surface-800 px-3 py-2.5 text-sm text-surface-200 outline-none focus:border-primary-500"
                  placeholder="What is this workspace for?"
                />
              </div>
            </>
          )}

          <hr className="border-surface-700" />

          <div>
            <label className="mb-1 block text-xs font-medium text-surface-400">
              Your Agent Name
            </label>
            <input
              type="text"
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              className="w-full rounded-md border border-surface-600 bg-surface-800 px-3 py-2.5 text-sm text-surface-200 outline-none focus:border-primary-500"
              placeholder="e.g., Claude-Backend, GPT-Frontend..."
              required
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-surface-400">
              Agent Type
            </label>
            <div className="grid grid-cols-4 gap-2">
              {(['claude', 'gpt', 'human', 'custom'] as const).map((t) => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setAgentType(t)}
                  className={clsx(
                    'rounded-md border py-2 text-xs font-medium capitalize transition-colors',
                    agentType === t
                      ? 'border-primary-500 bg-primary-500/20 text-primary-300'
                      : 'border-surface-600 text-surface-400 hover:border-surface-500 hover:text-surface-300'
                  )}
                >
                  {t}
                </button>
              ))}
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="flex w-full items-center justify-center gap-2 rounded-md bg-primary-600 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : mode === 'join' ? (
              <>
                <ArrowRight className="h-4 w-4" />
                Join Workspace
              </>
            ) : (
              <>
                <Plus className="h-4 w-4" />
                Create & Join
              </>
            )}
          </button>
        </form>

        <p className="mt-6 text-center text-xs text-surface-600">
          AgentHub v1.0 &mdash; Multi-Agent Collaboration Platform
        </p>
      </div>
    </div>
  )
}
