import { STATE_LABEL } from '../stateMeta'
import { duration, useNow } from '../time'
import type { Session, State } from '../types'

type TimelineProps = { session: Session; size: 'mini' | 'full' }

type Segment = { state: State; from: number; to: number }

const SUMMARY: { state: State; word: string }[] = [
  { state: 'working', word: 'working' },
  { state: 'waiting', word: 'waiting' },
  { state: 'idle', word: 'idle' },
]

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
    <div className="timeline h-[2px]">
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
  const summary = SUMMARY.map(({ state, word }) => ({ word, ms: total(state) }))
    .filter((part) => part.ms >= 1000)
    .map((part) => `${duration(part.ms)} ${part.word}`)
    .join(' · ')

  return (
    <div className="short:hidden flex flex-wrap items-center gap-x-4 gap-y-1 px-4 pb-2">
      <div className="min-w-[8rem] flex-1">{bar}</div>
      <span className="mono flex-none text-label whitespace-nowrap text-dim">{summary || 'just started'}</span>
    </div>
  )
}
