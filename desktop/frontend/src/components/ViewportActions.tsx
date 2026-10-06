import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'
import { PRBadge } from './PRBadge'

type ViewportActionsProps = { session: Session }

const PR_ACTION = {
  comments: { label: 'Address review comments', sentence: (n: number) => `Address the review comments on PR #${n}.` },
  checks: { label: 'Fix failing checks', sentence: (n: number) => `Fix the failing checks on PR #${n}.` },
}

export function ViewportActions({ session }: ViewportActionsProps) {
  const { report, focus, typeInto, ackPR } = useAgentos()
  if (!session.pr) return null
  const action = session.prAttention ? PR_ACTION[session.prAttention] : null
  const number = session.pr.number

  return (
    <div className="short:hidden flex flex-wrap items-center gap-x-3 gap-y-1.5 px-4 pb-2">
      <PRBadge session={session} />
      {action && (
        <button
          className="btn btn-accent"
          onClick={() => {
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
    </div>
  )
}
