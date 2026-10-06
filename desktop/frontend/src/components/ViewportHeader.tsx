import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { ago, useNow } from '../time'
import { useArmedConfirm } from '../useArmedConfirm'
import type { Session } from '../types'
import { InlineInput } from './InlineInput'
import { ModelTag } from './ModelTag'
import { StateBadge } from './StateBadge'

type ViewportHeaderProps = { session: Session }

export function ViewportHeader({ session }: ViewportHeaderProps) {
  const { renameSession, report, focus, killSession } = useAgentos()
  const now = useNow()
  const [renaming, setRenaming] = useState(false)
  const kill = useArmedConfirm()

  const finishRename = () => {
    setRenaming(false)
    focus('terminal')
  }

  return (
    <header className="panel-head flex flex-none flex-col gap-1 px-4 py-2.5 short:py-1.5">
      <div className="flex items-center gap-3">
        <span className="mono text-title font-semibold text-soft">{session.n}</span>
        <div className="min-w-0 flex-1">
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
        </div>
        <span className="mono flex-none text-small whitespace-nowrap text-dim">{ago(session.lastEventAt, now)}</span>
        {kill.armed ? (
          <span
            className="contents"
            role="alertdialog"
            aria-label="Kill this session"
            onKeyDown={(e) => {
              if (e.key !== 'Escape') return
              e.stopPropagation()
              kill.disarm()
            }}
          >
            <button className="btn" onClick={kill.disarm}>
              Cancel
            </button>
            <button
              className="btn btn-danger"
              autoFocus
              onClick={() => {
                kill.disarm()
                report(() => killSession(session.id))
              }}
            >
              Kill
            </button>
          </span>
        ) : (
          <button className="btn btn-ghost" onClick={() => kill.arm()} title="Kill this session">
            Kill
          </button>
        )}
      </div>
      <div className="short:hidden flex flex-wrap items-center gap-x-3 gap-y-1.5 pl-[1.625rem]">
        <StateBadge state={session.state} />
        <ModelTag model={session.model} effort={session.effort} />
      </div>
      {kill.armed && (
        <p className="pt-1.5 pb-2 text-center text-small" style={{ color: 'var(--danger)' }}>
          Kill this session? The agent stops now. Its conversation stays on disk and can be resumed from a terminal with{' '}
          <code className="mono">claude --resume</code>.
        </p>
      )}
    </header>
  )
}
