import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { ago, useNow } from '../time'
import { useArmedConfirm } from '../useArmedConfirm'
import type { Session } from '../types'
import { InlineInput } from './InlineInput'
import { ChatTag } from './ChatTag'
import { ModelTag } from './ModelTag'
import { StateBadge } from './StateBadge'
import { Timeline } from './Timeline'

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
    <header className="flex flex-none flex-col">
      <div className="panel-head flex items-center gap-3 px-4 py-2.5 short:py-1.5">
        <span className="mono w-4 flex-none text-title font-semibold text-soft">{session.n}</span>
        <div className="min-w-0">
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
        <span className="mono ml-auto flex-none text-small whitespace-nowrap text-dim">{ago(session.lastEventAt, now)}</span>
        {session.state === 'ended' ? null : kill.armed ? (
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
          <button className="btn btn-ghost text-danger" onClick={() => kill.arm()} title="Kill this session">
            Kill
          </button>
        )}
      </div>
      <div className="short:hidden flex items-center gap-3 px-4 pt-2.5 pb-0.5">
        <StateBadge state={session.state} />
        {session.chat && <ChatTag />}
        <ModelTag model={session.model} effort={session.effort} />
      </div>
      {!session.chat && <Timeline session={session} size="full" />}
      {kill.armed && (
        <p className="px-4 pb-2 text-center text-small" style={{ color: 'var(--danger)' }}>
          Kill this session? The agent stops now. Its conversation stays on disk and can be resumed from a terminal with{' '}
          <code className="mono">claude --resume</code>.
        </p>
      )}
    </header>
  )
}
