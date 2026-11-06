import type { Session, State } from '../types'
import { StateDot } from './StateDot'

type SessionGroupSummaryProps = { sessions: Session[]; open: boolean }

const TALLIED: State[] = ['waiting', 'working', 'idle']

export function SessionGroupSummary({ sessions, open }: SessionGroupSummaryProps) {
  const tally = TALLIED.map((state) => ({ state, count: sessions.filter((s) => s.state === state).length })).filter((t) => t.count > 0)
  const waiting = tally.find((t) => t.state === 'waiting')

  return (
    <span className="flex items-center gap-2.5">
      {!open && waiting && (
        <span className="text-label font-medium" data-state="waiting" style={{ color: 'var(--c)' }}>
          {waiting.count} needs you
        </span>
      )}
      {tally
        .filter((t) => open || t.state !== 'waiting')
        .map(({ state, count }) => (
          <span key={state} className="flex items-center gap-1">
            <StateDot state={state} />
            <span className="mono text-label text-dim">{count}</span>
          </span>
        ))}
    </span>
  )
}
