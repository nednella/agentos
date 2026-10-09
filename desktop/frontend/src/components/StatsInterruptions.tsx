import { useEffect, useState } from 'react'
import { api, on } from '../api'
import { useAgentos } from '../AgentosContext'
import type { Stats as StatsData } from '../types'
import { StatsBars } from './StatsBars'
import { StatsFigure } from './StatsFigure'
import { StatsRecent } from './StatsRecent'

type StatsInterruptionsProps = { days: number }

export function StatsInterruptions({ days }: StatsInterruptionsProps) {
  const { report } = useAgentos()
  const [stats, setStats] = useState<StatsData | null>(null)

  useEffect(() => {
    const load = () => report(async () => setStats(await api.stats(days)))
    load()
    return on('stats', load)
  }, [days, report])

  if (stats === null) return <p className="text-small text-dim">Loading…</p>
  if (stats.total === 0) {
    return (
      <div className="flex flex-col gap-1 py-16 text-center">
        <p className="text-body font-medium">No interruptions recorded yet</p>
        <p className="text-small text-dim">Each time a session waits for you, it shows up here with what it asked for.</p>
      </div>
    )
  }
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6">
      <StatsFigure stats={stats} />
      <StatsBars stats={stats} />
      <StatsRecent stats={stats} />
    </div>
  )
}
