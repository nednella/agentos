import { duration, ago, useNow } from '../time'
import { KIND_COLOR, KIND_LABEL } from '../statsMeta'
import type { Stats } from '../types'


type StatsRecentProps = { stats: Stats }

export function StatsRecent({ stats }: StatsRecentProps) {
  const now = useNow()
  return (
    <section>
      <h3 className="label pb-2">Recent</h3>
      <ul className="divide-y divide-line rounded-md border border-line">
        {stats.recent.slice(0, 12).map((w) => (
          <li key={`${w.startedAt}-${w.sessionTitle}-${w.label}`} className="flex items-baseline gap-3 px-3 py-2">
            <span className="mono w-20 flex-none text-label whitespace-nowrap text-dim">{ago(w.startedAt, now)}</span>
            <span className="min-w-0 flex-1 truncate text-body">{w.sessionTitle}</span>
            <span className="mono hidden min-w-0 max-w-[40%] truncate text-small text-soft sm:block">{w.label}</span>
            <span className="flex-none text-label" style={{ color: KIND_COLOR[w.kind] }}>
              {KIND_LABEL[w.kind]}
            </span>
            <span className="mono w-14 flex-none text-right text-small text-soft">{w.waitedMs ? duration(w.waitedMs) : 'waiting'}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}
