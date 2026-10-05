import { useEffect, useRef, useState } from 'react'
import { api, errorMessage } from '../api'
import { useAgentos } from '../AgentosContext'
import { ago, useNow } from '../time'
import type { IssueDetail } from '../types'
import { Icon } from './Icon'
import { IssueBody } from './IssueBody'
import { Overlay } from './Overlay'
import { StateDot } from './StateDot'
import { TypeMark } from './TypeMark'

const dateOf = (at: number) => new Date(at).toLocaleString()

export function IssueDialog() {
  const { overlay } = useAgentos()
  if (typeof overlay !== 'object' || !overlay) return null
  return <IssueDialogContent key={overlay.issue} number={overlay.issue} />
}

type IssueDialogContentProps = { number: number }

function IssueDialogContent({ number }: IssueDialogContentProps) {
  const { issues, sessions, startIssue, select, setOverlay, report } = useAgentos()
  const now = useNow()
  const body = useRef<HTMLDivElement>(null)
  const [detail, setDetail] = useState<IssueDetail | null>(null)
  const [error, setError] = useState('')
  const issue = issues.find((i) => i.number === number)
  const session = sessions.find((s) => s.id === issue?.sessionId)

  useEffect(() => {
    api.issueDetail(number).then(setDetail, (err) => setError(errorMessage(err)))
  }, [number])

  useEffect(() => body.current?.focus(), [])

  const close = () => setOverlay(null)

  return (
    <Overlay align="center" size="reading" label={`Issue #${number}`} onClose={close}>
      <header className="flex flex-none flex-col gap-1.5 border-b border-line px-4 py-2.5">
        <div className="flex items-start gap-3">
          <span className="mono flex-none pt-px text-title font-semibold text-soft">#{number}</span>
          <h2 className="min-w-0 flex-1 text-title font-semibold break-words">{issue?.title ?? 'No longer in the queue'}</h2>
          <span className="flex flex-none items-center gap-2">
            {issue && session && (
              <button
                className="btn"
                title={`Open session ${session.n}`}
                onClick={() => {
                  close()
                  select(session.id)
                }}
              >
                <StateDot state={session.state} />
                Session {session.n}
              </button>
            )}
            {issue && !session && (
              <button
                className="btn btn-accent"
                title="Start a session for this issue"
                onClick={() => {
                  close()
                  report(() => startIssue(number))
                }}
              >
                Start
              </button>
            )}
            {issue && (
              <button className="btn btn-ghost h-7 w-7 justify-center px-0" title="Open on GitHub" aria-label="Open on GitHub" onClick={() => report(() => api.openURL(issue.url))}>
                <Icon name="external" />
              </button>
            )}
            <button className="btn btn-ghost h-7 w-7 justify-center px-0" title="Close (Esc)" aria-label="Close" onClick={close}>
              <Icon name="close" />
            </button>
          </span>
        </div>
        {issue && (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-small text-dim">
            <TypeMark type={issue.type} />
            <span title={dateOf(issue.createdAt)}>
              {issue.author ? `${issue.author} opened this` : 'Opened'} {ago(issue.createdAt, now)}
            </span>
            {issue.labels.map((label) => (
              <span key={label} className="mono rounded-sm border border-line-strong px-1.5 text-label text-soft">
                {label}
              </span>
            ))}
          </div>
        )}
      </header>
      <div ref={body} tabIndex={-1} className="min-h-0 flex-1 overflow-y-auto p-4 outline-none">
        <div className="flex flex-col gap-6">
          {error && <p className="text-small text-danger">{error}</p>}
          {!detail && !error && <p className="text-small text-dim">Loading…</p>}
          {detail && (detail.bodyHTML ? <IssueBody html={detail.bodyHTML} /> : <p className="text-small text-dim">No description.</p>)}
          {detail && detail.comments.length > 0 && (
            <section className="flex flex-col gap-4 border-t border-line pt-4">
              <h3 className="label">{detail.comments.length === 1 ? '1 comment' : `${detail.comments.length} comments`}</h3>
              {detail.comments.map((comment) => (
                <article key={comment.createdAt} className="flex flex-col gap-1.5">
                  <div className="flex items-baseline gap-2 text-small">
                    <span className="font-medium">{comment.author}</span>
                    <span className="text-dim" title={dateOf(comment.createdAt)}>
                      {ago(comment.createdAt, now)}
                    </span>
                  </div>
                  <IssueBody html={comment.bodyHTML} />
                </article>
              ))}
            </section>
          )}
        </div>
      </div>
    </Overlay>
  )
}
