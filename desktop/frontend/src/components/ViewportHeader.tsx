import { useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { ago, useNow } from '../time'
import { useArmedConfirm } from '../useArmedConfirm'
import type { Session } from '../types'
import { InlineInput } from './InlineInput'
import { ConfirmRow } from './ConfirmRow'
import { PRBadge } from './PRBadge'
import { StateBadge } from './StateBadge'

type ViewportHeaderProps = { session: Session }

const PR_ACTION = {
  comments: { label: 'Address review comments', sentence: (n: number) => `Address the review comments on PR #${n}.` },
  checks: { label: 'Fix failing checks', sentence: (n: number) => `Fix the failing checks on PR #${n}.` },
}

export function ViewportHeader({ session }: ViewportHeaderProps) {
  const { renameSession, report, focus, typeInto, ackPR, killSession } = useAgentos()
  const now = useNow()
  const [renaming, setRenaming] = useState(false)
  const kill = useArmedConfirm()
  const action = session.pr && session.prAttention ? PR_ACTION[session.prAttention] : null

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
            <PRBadge session={session} />
            {action && session.pr && (
              <button
                className="btn btn-accent"
                onClick={() => {
                  const number = session.pr?.number ?? 0
                  report(async () => {
                    await typeInto(session.id, action.sentence(number))
                    await ackPR(session.id)
                  })
                  focus('terminal')
                }}
              >
                {action.label}
              </button>
            )}
          </span>
        </div>
        <span className="mono flex-none text-small whitespace-nowrap text-dim">{ago(session.lastEventAt, now)}</span>
        <button className="btn btn-ghost" onClick={() => kill.arm()} title="Kill this session" disabled={Boolean(kill.armed)}>
          Kill
        </button>
      </div>
      {kill.armed && (
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
            onCancel={kill.disarm}
            onConfirm={() => {
              kill.disarm()
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
