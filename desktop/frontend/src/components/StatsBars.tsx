import { useAgentos } from '../AgentosContext'
import { duration } from '../time'
import { KIND_COLOR, KIND_LABEL } from '../statsMeta'
import type { Stats } from '../types'


type StatsBarsProps = { stats: Stats }

export function StatsBars({ stats }: StatsBarsProps) {
  const { harnessCheck, report } = useAgentos()
  const top = Math.max(...stats.byCause.map((c) => c.count), 1)
  const topDay = Math.max(...stats.byDay.map((d) => d.count), 1)
  const labelEvery = Math.ceil(stats.byDay.length / 7)

  return (
    <>
      <section>
        <div className="flex flex-wrap items-center justify-between gap-2 pb-2">
          <h3 className="label">Causes, most frequent first</h3>
          <button className="btn h-7" onClick={() => report(harnessCheck)}>
            See what the harness could fix
          </button>
        </div>
        <ul className="flex flex-col gap-2.5">
          {stats.byCause.map((cause) => (
            <li key={cause.label} className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-1">
              <span className="flex min-w-0 items-baseline gap-2">
                <span className="mono truncate text-body">{cause.label}</span>
                <span className="flex-none text-label" style={{ color: KIND_COLOR[cause.kind] }}>
                  {KIND_LABEL[cause.kind]}
                </span>
              </span>
              <span className="mono text-small text-soft">
                {cause.count} · {Math.round((cause.count / stats.total) * 100)}% · {duration(cause.totalWaitMs)}
              </span>
              <span className="col-span-2 h-2 rounded-sm bg-raised" role="img" aria-label={`${cause.count} of ${stats.total}`}>
                <span
                  className="block h-full rounded-sm"
                  style={{ width: `${(cause.count / top) * 100}%`, background: KIND_COLOR[cause.kind] }}
                />
              </span>
            </li>
          ))}
        </ul>
      </section>
      <section>
        <h3 className="label pb-2">Per day</h3>
        <div className="flex h-28 items-end gap-1" role="img" aria-label="Interruptions per day">
          {stats.byDay.map((d) => (
            <div key={d.day} className="flex h-full min-w-0 flex-1 flex-col justify-end" title={`${d.day}: ${d.count}`}>
              <span className="mono pb-0.5 text-center text-label text-dim">{d.count || ''}</span>
              <span className="block rounded-sm" style={{ height: `${(d.count / topDay) * 80}%`, minHeight: d.count ? 2 : 0, background: 'var(--accent)' }} />
            </div>
          ))}
        </div>
        <div className="mt-1 flex gap-1">
          {stats.byDay.map((d, i) => (
            <span key={d.day} className="mono min-w-0 flex-1 overflow-visible text-center text-label whitespace-nowrap text-dim">
              {i % labelEvery === 0 ? d.day.slice(5) : ''}
            </span>
          ))}
        </div>
      </section>
    </>
  )
}
