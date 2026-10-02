import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'
import { ChecksMark } from './ChecksMark'

type PRBadgeProps = { session: Session; compact?: boolean }

export function PRBadge({ session, compact = false }: PRBadgeProps) {
  const { openPR, report } = useAgentos()
  const { pr, prAttention } = session
  if (!pr) return null
  const color = { checks: 'var(--danger)', comments: 'var(--waiting)', '': 'var(--text-soft)' }[prAttention]
  const note = { checks: 'failing checks', comments: 'new review comments', '': '' }[prAttention]

  return (
    <button
      className="inline-flex h-6 flex-none items-center gap-1.5 rounded-sm border px-2 text-small font-medium"
      style={{
        color,
        borderColor: prAttention ? color : 'var(--border-strong)',
        background: prAttention ? `color-mix(in srgb, ${color} 12%, transparent)` : 'transparent',
      }}
      title={`Open PR #${pr.number} (${pr.state}${note ? `, ${note}` : ''}) on GitHub`}
      onClick={() => report(() => openPR(session))}
    >
      <span className="mono">{compact ? `#${pr.number}` : `PR #${pr.number}`}</span>
      {!compact && <span className="text-label opacity-80">· {pr.state}</span>}
      <ChecksMark checks={pr.checks} />
      {pr.comments > 0 && <span className="mono text-label">{pr.comments}c</span>}
    </button>
  )
}
