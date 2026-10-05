import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { api } from '../api'
import type { Issue } from '../types'
import { Icon } from './Icon'
import { PRMark } from './PRMark'
import { StateDot } from './StateDot'
import { TypeMark } from './TypeMark'

type IssueRowProps = { issue: Issue; cursor: boolean }

export function IssueRow({ issue, cursor }: IssueRowProps) {
  const { sessions, overlay, select, startIssue, setOverlay, report } = useAgentos()
  const { mode } = useLayout()
  const session = sessions.find((s) => s.id === issue.sessionId)
  const dense = mode === 'compact'
  const open = typeof overlay === 'object' && overlay?.issue === issue.number

  return (
    <div className="row row-inset group h-9 items-center gap-1 pr-1.5 pl-3" data-cursor={cursor} data-selected={open}>
      <button
        className="flex h-full min-w-0 flex-1 items-center gap-2 text-left"
        title="Read the issue"
        onClick={() => setOverlay({ issue: issue.number })}
      >
        <span className={`mono flex-none text-small text-dim ${dense ? 'w-8' : 'w-10'}`}>{issue.number}</span>
        <TypeMark type={issue.type} />
        <span className="min-w-0 flex-1 truncate text-body">{issue.title}</span>
      </button>
      <span className="reveal">
        <button
          className="btn btn-ghost h-6 w-6 justify-center px-0"
          title="Open on GitHub"
          aria-label={`Open #${issue.number} on GitHub`}
          onClick={() => report(() => api.openURL(issue.url))}
        >
          <Icon name="external" size={12} />
        </button>
        {!session && (
          <button
            className="btn btn-accent h-6 w-12 flex-none justify-center px-0"
            title="Start a session (hold ⌥ to keep your place)"
            onClick={(e) => report(() => startIssue(issue.number, e.altKey))}
          >
            Start
          </button>
        )}
      </span>
      {session && (
        <button className="flex h-full flex-none items-center justify-end gap-2 pl-1" title={`Open session ${session.n}`} onClick={() => select(session.id)}>
          {session.pr && !dense && <PRMark pr={session.pr} />}
          <StateDot state={session.state} />
        </button>
      )}
    </div>
  )
}
