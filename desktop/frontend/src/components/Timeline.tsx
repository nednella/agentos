import { STATE_LABEL } from '../stateMeta'
import { duration, useNow } from '../time'
import type { Session, State } from '../types'

type TimelineProps = { session: Session; size: 'mini' | 'full' }

type Segment = { state: State; from: number; to: number }

function segmentsOf(session: Session, now: number): Segment[] {
  const { history, createdAt, state } = session
  if (history.length === 0) return [{ state, from: createdAt, to: now }]
  const segments = history.map((entry, i) => ({
    state: entry.state,
    from: entry.at,
    to: history[i + 1]?.at ?? now,
  }))
  if (history[0].at - createdAt > 1000) segments.unshift({ state: 'idle', from: createdAt, to: history[0].at })
  return segments
}

export function Timeline({ session, size }: TimelineProps) {
  const now = useNow()
  const segments = segmentsOf(session, now)
  const bar = (
    <div className={`timeline ${size === 'full' ? 'h-1.5' : 'h-[3px]'}`}>
      {segments.map((seg, i) => (
        <span
          key={`${seg.from}-${i}`}
          data-state={seg.state}
          style={{ flex: `${Math.max(seg.to - seg.from, 1)} 1 0` }}
          title={`${STATE_LABEL[seg.state]} ${duration(seg.to - seg.from)}`}
        />
      ))}
    </div>
  )
  if (size === 'mini') return bar

  const total = (state: State) =>
    segments.filter((s) => s.state === state).reduce((sum, s) => sum + (s.to - s.from), 0)
  return (
    <div className="short:hidden flex items-center gap-4 px-4 pb-2">
      {bar}
      <div className="mono flex flex-none gap-3 text-label whitespace-nowrap text-dim">
        <span>worked {duration(total('working'))}</span>
        <span>waited {duration(total('waiting') + total('finished'))}</span>
      </div>
    </div>
  )
}
