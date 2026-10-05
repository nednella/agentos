import { duration } from '../time'
import type { Stats } from '../types'

type StatsFigureProps = { stats: Stats }

type FigureProps = { label: string; value: string; hint: string }

function Figure({ label, value, hint }: FigureProps) {
  return (
    <div className="flex min-w-0 flex-1 basis-36 flex-col gap-0.5 rounded-md border border-line bg-surface px-4 py-3">
      <span className="label">{label}</span>
      <span className="mono text-[1.75rem] leading-tight font-semibold">{value}</span>
      <span className="text-small text-dim">{hint}</span>
    </div>
  )
}

export function StatsFigure({ stats }: StatsFigureProps) {
  const perDay = (stats.total / stats.days).toFixed(1)
  return (
    <div className="flex flex-wrap gap-3">
      <Figure label="Interruptions" value={String(stats.total)} hint={`${perDay} a day`} />
      <Figure label="Time waiting on you" value={duration(stats.totalWaitMs)} hint="all sessions together" />
      <Figure label="Median wait" value={duration(stats.medianWaitMs)} hint="before you answered" />
    </div>
  )
}
