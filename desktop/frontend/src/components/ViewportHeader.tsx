import { useEffect, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { ago, useNow } from '../time'
import type { Session } from '../types'
import { InlineInput } from './InlineInput'
import { ConfirmRow } from './ConfirmRow'
import { StateBadge } from './StateBadge'

type ViewportHeaderProps = { session: Session }

export function ViewportHeader({ session }: ViewportHeaderProps) {
  const { renameSession, report, focus, killSession } = useAgentos()
  const now = useNow()
  const [renaming, setRenaming] = useState(false)
  const [killing, setKilling] = useState(false)

  useEffect(() => setKilling(false), [session.id])

  const finishRename = () => {
    setRenaming(false)
    focus('terminal')
  }

  return (
    <header className="panel-head flex flex-none flex-col gap-1 px-4 py-2.5 short:py-1.5">
      <div className="flex items-center gap-3">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1.5">
          <span className="mono text-title font-semibold text-soft">{session.n}</span>
          {renaming ? (
            <InlineInput
              initial={session.title}
              blur="submit"
              className="w-64 max-w-full"
              onSubmit={(title) => {
                report(() => renameSession(session.id, title))
                finishRename()
              }}
              onCancel={finishRename}
            />
          ) : (
            <h2
              className="min-w-0 truncate text-title font-semibold"
              title="Double-click to rename"
              onDoubleClick={() => setRenaming(true)}
            >
              {session.title}
            </h2>
          )}
          <span className="contents short:hidden">
            <StateBadge state={session.state} />
          </span>
        </div>
        <span className="mono flex-none text-small whitespace-nowrap text-dim">{ago(session.lastEventAt, now)}</span>
        <button className="btn btn-ghost" onClick={() => setKilling(true)} title="Kill this session" disabled={killing}>
          Kill
        </button>
      </div>
      {killing && (
        <div className="pt-1.5">
          <ConfirmRow
            danger
            label="Kill this session"
            message={
              <>
                Kill this session? The agent stops now. Its conversation stays on disk and can be resumed from a terminal with{' '}
                <code className="mono">claude --resume</code>.
              </>
            }
            confirmLabel="Kill"
            onCancel={() => setKilling(false)}
            onConfirm={() => {
              setKilling(false)
              report(() => killSession(session.id))
            }}
          />
        </div>
      )}
      {session.detail && (
        <span
          className="mono short:hidden truncate text-small"
          style={{ color: session.state === 'waiting' ? 'var(--waiting)' : 'var(--text-soft)' }}
        >
          {session.detail}
        </span>
      )}
    </header>
  )
}
