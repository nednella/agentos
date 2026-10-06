import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import { ago, useNow } from '../time'
import type { Session } from '../types'
import { CleanupControls } from './CleanupControls'
import { Icon } from './Icon'
import { ModelTag } from './ModelTag'
import { PRBadge } from './PRBadge'
import { StateDot } from './StateDot'
import { Timeline } from './Timeline'

type SessionRowProps = { session: Session; selected: boolean; cursor: boolean; compact: boolean; dense: boolean }

export function SessionRow({ session, selected, cursor, compact, dense }: SessionRowProps) {
  const { select, focus, setSessionView, dismissSession, report, issues } = useAgentos()
  const now = useNow()
  const ended = session.state === 'ended'
  const issue = issues.find((i) => i.number === session.issue)
  const showPR = session.pr !== null && !dense
  const showEvidence = session.evidence > 0 && !dense
  const hasFooter = ended || showPR || showEvidence || session.cleanup !== ''

  return (
    <div className="row flex-col" data-state={session.state} data-selected={selected} data-cursor={cursor} style={{ opacity: ended ? 0.6 : 1 }}>
      <button
        className={`flex w-full flex-col gap-1 px-4 text-left ${dense ? 'py-2' : 'pt-2.5 pb-2.5'}`}
        aria-current={selected}
        disabled={ended}
        title={ended ? `${session.title} (ended)` : session.title}
        style={ended ? { cursor: 'default', opacity: 1 } : undefined}
        onMouseDown={(e) => e.preventDefault()}
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
        {!compact && !dense && session.model && (
          <span className="w-full pl-[1.625rem]">
            <ModelTag model={session.model} effort={session.effort} />
          </span>
        )}
        {!dense && (
          <span className="w-full pl-[1.625rem]">
            <Timeline session={session} size="mini" />
          </span>
        )}
      </button>
      {hasFooter && (
        <div className="flex w-full flex-wrap items-center gap-2 pr-3 pb-2.5 pl-[2.625rem]">
          {ended && (
            <span className="inline-flex h-6 items-center rounded-sm border border-line-strong px-2 text-small text-soft">Ended</span>
          )}
          {showPR && <PRBadge session={session} compact={compact} />}
          {ended && session.pr && dense && <PRBadge session={session} compact />}
          {showEvidence && (
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
          {ended && issue && (
            <button
              className="btn btn-ghost h-6 px-2"
              title={`Open #${issue.number} on GitHub`}
              onClick={() => report(() => api.openURL(issue.url))}
            >
              <Icon name="external" size={12} /> #{issue.number}
            </button>
          )}
          {ended && (
            <button className="btn ml-auto h-6 px-2" onClick={() => report(() => dismissSession(session.id))}>
              Dismiss
            </button>
          )}
        </div>
      )}
    </div>
  )
}
