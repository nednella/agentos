import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { readStored, writeStored } from '../storage'
import { ConfirmRow } from './ConfirmRow'
import type { Session } from '../types'

type CleanupControlsProps = { session: Session }

const KEPT_KEY = 'agentos.keptCleanups'

function tail(path: string): string {
  return path.split('/').slice(-2).join('/')
}

export function CleanupControls({ session }: CleanupControlsProps) {
  const { cleanupSession, report } = useAgentos()
  const [confirming, setConfirming] = useState(false)
  const [kept, setKept] = useState(() => readStored<string[]>(KEPT_KEY, []).includes(session.id))
  const { cleanup, cleanupReason, worktree, branch } = session

  if (cleanup === 'pending') return <span className="text-small text-dim">cleaning up…</span>
  if (cleanup === 'ask' && kept) return null

  if (confirming) {
    const what = `${worktree ? `the worktree ${tail(worktree)}, ` : ''}${branch ? `the branch ${branch}, ` : ''}temp files and the session`
    return (
      <ConfirmRow
        danger
        message={`Remove ${what}?${cleanup === 'blocked' ? ' Uncommitted work is lost.' : ''}`}
        confirmLabel="Remove"
        onCancel={() => setConfirming(false)}
        onConfirm={() => {
          setConfirming(false)
          report(() => cleanupSession(session.id, true))
        }}
      />
    )
  }

  if (cleanup === 'blocked') {
    return (
      <div className="flex w-full flex-wrap items-center gap-x-2 gap-y-1">
        <span className="min-w-0 text-small text-danger">Not cleaned up: {cleanupReason}</span>
        <button className="btn h-6" onClick={() => setConfirming(true)}>
          Clean up anyway
        </button>
      </div>
    )
  }

  return (
    <div className="flex w-full flex-wrap items-center gap-x-2 gap-y-1">
      <span className="text-small text-waiting">
        {session.pr?.state === 'merged' ? 'PR was already merged' : 'PR closed without merging'}
      </span>
      <button className="btn h-6" onClick={() => setConfirming(true)}>
        Clean up
      </button>
      <button
        className="btn btn-ghost h-6"
        onClick={() => {
          writeStored(KEPT_KEY, [...readStored<string[]>(KEPT_KEY, []), session.id])
          setKept(true)
        }}
      >
        Keep
      </button>
    </div>
  )
}
