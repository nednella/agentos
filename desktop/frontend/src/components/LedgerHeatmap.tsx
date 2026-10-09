import type { Ledger } from '../types'

type LedgerHeatmapProps = { heat: Ledger['heat'] }

const LEVELS = 4

function cellColor(count: number, top: number) {
  if (count === 0) return 'var(--bg-raised)'
  const level = Math.ceil((count / top) * LEVELS)
  return `color-mix(in srgb, var(--finished) ${level * 25}%, var(--bg-raised))`
}

export function LedgerHeatmap({ heat }: LedgerHeatmapProps) {
  const top = Math.max(...heat.map((d) => d.count), 1)
  const lead = heat.length ? new Date(`${heat[0].day}T00:00`).getDay() : 0
  return (
    <section>
      <h3 className="label pb-2">Prompts per day, last 52 weeks</h3>
      <div
        className="grid grid-flow-col grid-rows-7 auto-cols-fr gap-[2px]"
        role="img"
        aria-label="Prompts per day"
      >
        {Array.from({ length: lead }, (_, i) => (
          <i key={i} />
        ))}
        {heat.map((d) => (
          <i key={d.day} title={`${d.day}: ${d.count} prompts`} className="block aspect-square rounded-[2px]" style={{ background: cellColor(d.count, top) }} />
        ))}
      </div>
    </section>
  )
}
