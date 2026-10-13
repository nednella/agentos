import type { Ledger } from '../types'

type LedgerHeatmapProps = { heat: Ledger['heat']; days: number }

const LEVELS = 4

function cellColor(count: number, top: number) {
  if (count === 0) return 'var(--bg-raised)'
  const level = Math.ceil((count / top) * LEVELS)
  return `color-mix(in srgb, var(--finished) ${level * 25}%, var(--bg-raised))`
}

export function LedgerHeatmap({ heat, days }: LedgerHeatmapProps) {
  const firstInRange = days ? heat.length - days : 0
  const top = Math.max(...heat.map((d) => d.count), 1)
  const lead = heat.length ? new Date(`${heat[0].day}T00:00`).getDay() : 0
  return (
    <figure>
      <div
        className="grid grid-flow-col grid-rows-7 auto-cols-fr gap-[2px]"
        role="img"
        aria-label="Prompts per day"
      >
        {Array.from({ length: lead }, (_, i) => (
          <i key={i} />
        ))}
        {heat.map((d, i) => (
          <i
            key={d.day}
            title={`${d.day}: ${d.count} prompts`}
            className={`block aspect-square rounded-[2px] ${i < firstInRange ? 'opacity-40' : ''}`}
            style={{ background: cellColor(d.count, top) }}
          />
        ))}
      </div>
      <figcaption className="pt-1.5 text-small text-dim">Prompts per day, last 52 weeks{firstInRange > 0 && ', days outside the range dimmed'}</figcaption>
    </figure>
  )
}
