import { useAgentos } from '../AgentosContext'
import { STATE_LABEL } from '../stateMeta'
import { ago, useNow } from '../time'
import type { Session } from '../types'
import { Icon } from './Icon'
import { CleanupControls } from './CleanupControls'
import { PRBadge } from './PRBadge'
import { StateDot } from './StateDot'
import { Timeline } from './Timeline'

type SessionRowProps = { session: Session; selected: boolean; cursor: boolean; compact: boolean }

export function SessionRow({ session, selected, cursor, compact }: SessionRowProps) {
  const { select, focus, setSessionView } = useAgentos()
  const now = useNow()
  const waiting = session.state === 'waiting'
  const hasFooter = session.pr !== null || session.cleanup !== '' || session.evidence > 0

  return (
    <div className="row flex-col" data-state={session.state} data-selected={selected} data-cursor={cursor}>
      <button
        className="flex w-full flex-col gap-1 px-4 pt-2.5 pb-2.5 text-left"
        aria-current={selected}
        title={session.title}
        onClick={() => {
          select(session.id)
          focus('terminal')
        }}
      >
        <span className="flex w-full items-center gap-2">
          <StateDot state={session.state} />
          <span className="mono w-4 flex-none text-small text-dim">{session.n}</span>
          <span className="min-w-0 flex-1 truncate text-body font-medium">{session.title}</span>
          <span className="mono flex-none text-label text-dim">{ago(session.lastEventAt, now)}</span>
        </span>
        {!compact && (
          <span
            className="mono w-full truncate pl-[1.625rem] text-small"
            style={{ color: waiting ? 'var(--waiting)' : 'var(--text-dim)' }}
          >
            {session.detail || STATE_LABEL[session.state]}
          </span>
        )}
        <span className="w-full pl-[1.625rem]">
          <Timeline session={session} size="mini" />
        </span>
      </button>
      {hasFooter && (
        <div className="flex w-full flex-wrap items-center gap-2 pr-4 pb-2.5 pl-[2.625rem]">
          <PRBadge session={session} compact={compact} />
          {session.evidence > 0 && (
            <button
              className="inline-flex h-6 flex-none items-center gap-1 rounded-sm border border-line-strong px-2 text-small text-soft"
              title={`${session.evidence} evidence items`}
              onClick={() => {
                select(session.id)
                setSessionView(session.id, 'evidence')
              }}
            >
              <Icon name="image" size={12} />
              <span className="mono">{session.evidence}</span>
            </button>
          )}
          {session.cleanup !== '' && <CleanupControls session={session} />}
        </div>
      )}
    </div>
  )
}
