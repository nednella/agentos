import { useEffect, useMemo, useRef } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { ago, until, useNow } from '../time'
import type { DigestItem } from '../types'
import { DigestItemRow } from './DigestItemRow'
import { Icon } from './Icon'

const dayLabel = (at: number) => new Date(at).toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' })

function groupByDay(items: DigestItem[]): { day: string; items: DigestItem[] }[] {
  const groups: { day: string; items: DigestItem[] }[] = []
  for (const item of items) {
    const day = dayLabel(item.at)
    const last = groups[groups.length - 1]
    if (last?.day === day) last.items.push(item)
    else groups.push({ day, items: [item] })
  }
  return groups
}

export function DigestView() {
  const { digest, runDigest, markDigestSeen, report, focusRequest, project } = useAgentos()
  const { closeCentre } = useLayout()
  const now = useNow()
  const panel = useRef<HTMLElement>(null)
  const groups = useMemo(() => groupByDay(digest?.items ?? []), [digest?.items])

  useEffect(() => panel.current?.focus(), [])
  useEffect(() => {
    if (focusRequest.target === 'terminal') panel.current?.focus()
  }, [focusRequest])
  useEffect(() => markDigestSeen(), [digest?.lastRunAt, markDigestSeen])

  return (
    <section
      ref={panel}
      data-panel="terminal"
      tabIndex={-1}
      aria-label="Weekly digest"
      className="panel flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
      onKeyDown={(e) => e.key === 'Escape' && closeCentre()}
    >
      <header className="panel-head flex flex-none flex-wrap items-center gap-x-3 gap-y-2 border-b border-line px-4 py-2.5">
        <h2 className="text-title font-semibold">Weekly digest</h2>
        <span className="text-small text-dim">{project?.name}</span>
        <span className="mono text-small text-dim">
          {digest && digest.lastRunAt > 0 ? `last run ${ago(digest.lastRunAt, now)}` : 'never run'}
          {digest && digest.nextRunAt > 0 ? ` · next ${until(digest.nextRunAt, now)}` : ' · automatic runs off'}
        </span>
        <span className="ml-auto flex items-center gap-2">
          <button className="btn" disabled={!digest || digest.running} onClick={() => report(runDigest)}>
            {digest?.running ? (
              <>
                <span className="spinner" /> Running…
              </>
            ) : (
              'Run now'
            )}
          </button>
          <button className="btn btn-ghost h-7 w-7 justify-center px-0" title="Back (Esc)" aria-label="Close digest" onClick={closeCentre}>
            <Icon name="close" />
          </button>
        </span>
      </header>
      {digest?.error && <p className="flex-none border-b border-line px-4 py-2 text-small text-danger">{digest.error}</p>}
      <div className="min-h-0 flex-1 overflow-y-auto">
        {groups.length === 0 && (
          <div className="flex flex-col gap-1 px-6 py-16 text-center">
            <p className="text-body font-medium">Nothing in the digest yet</p>
            <p className="text-small text-dim">Once a week an agent checks what changed in the tools this project depends on. Run it now to see how it reads.</p>
          </div>
        )}
        <div className="mx-auto max-w-3xl">
          {groups.map((group) => (
            <section key={group.day}>
              <h3 className="label sticky top-0 border-b border-line bg-surface px-4 py-2">{group.day}</h3>
              <ul className="divide-y divide-line">
                {group.items.map((item) => (
                  <DigestItemRow key={item.id} item={item} />
                ))}
              </ul>
            </section>
          ))}
        </div>
      </div>
    </section>
  )
}
