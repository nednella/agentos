import { useEffect, useRef, useState } from 'react'
import { api, on } from '../api'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { readStored, writeStored } from '../storage'
import type { Stats as StatsData } from '../types'
import { StatsBars } from './StatsBars'
import { StatsFigure } from './StatsFigure'
import { StatsRecent } from './StatsRecent'
import { Icon } from './Icon'

const RANGES = [7, 14, 30]
const RANGE_KEY = 'agentos.statsDays'

export function Stats() {
  const { report, focusRequest, project } = useAgentos()
  const { closeCentre } = useLayout()
  const [days, setDays] = useState(() => readStored(RANGE_KEY, 7))
  const [stats, setStats] = useState<StatsData | null>(null)
  const panel = useRef<HTMLElement>(null)

  useEffect(() => {
    const load = () => report(async () => setStats(await api.stats(days)))
    load()
    return on('stats', load)
  }, [days, report])

  useEffect(() => {
    if (focusRequest.target === 'terminal') panel.current?.focus()
  }, [focusRequest])

  useEffect(() => panel.current?.focus(), [])

  const pickRange = (next: number) => {
    setDays(next)
    writeStored(RANGE_KEY, next)
  }

  return (
    <section
      ref={panel}
      data-panel="terminal"
      tabIndex={-1}
      aria-label="Interruption stats"
      className="panel flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
      onKeyDown={(e) => e.key === 'Escape' && closeCentre()}
    >
      <header className="panel-head flex flex-none flex-wrap items-center gap-3 border-b border-line px-4 py-2.5">
        <h2 className="text-title font-semibold">What interrupts you</h2>
        <span className="text-small text-dim">{project?.name}</span>
        <div role="group" aria-label="Range" className="ml-auto flex overflow-hidden rounded-md border border-line-strong">
          {RANGES.map((range) => (
            <button
              key={range}
              aria-pressed={days === range}
              className="h-7 px-3 text-small font-medium"
              style={{ background: days === range ? 'var(--bg-hover)' : 'transparent', color: days === range ? 'var(--text)' : 'var(--text-soft)' }}
              onClick={() => pickRange(range)}
            >
              {range} days
            </button>
          ))}
        </div>
        <button className="btn btn-ghost h-7 w-7 justify-center px-0" title="Back to the terminal (Esc)" aria-label="Back to the terminal" onClick={closeCentre}>
          <Icon name="close" />
        </button>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {stats === null && <p className="text-small text-dim">Loading…</p>}
        {stats !== null && stats.total === 0 && (
          <div className="flex flex-col gap-1 py-16 text-center">
            <p className="text-body font-medium">No interruptions recorded yet</p>
            <p className="text-small text-dim">Each time a session waits for you, it shows up here with what it asked for.</p>
          </div>
        )}
        {stats !== null && stats.total > 0 && <StatsBody stats={stats} />}
      </div>
    </section>
  )
}

type StatsBodyProps = { stats: StatsData }

function StatsBody({ stats }: StatsBodyProps) {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6">
      <StatsFigure stats={stats} />
      <StatsBars stats={stats} />
      <StatsRecent stats={stats} />
    </div>
  )
}
