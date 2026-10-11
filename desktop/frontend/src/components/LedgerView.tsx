import { useEffect, useRef, useState, type ReactNode } from 'react'
import { api } from '../api'
import { useAgentos } from '../AgentosContext'
import { dollars, priceNote, tokenCount } from '../spend'
import { duration } from '../time'
import { useMinimumLoading } from '../useMinimumLoading'
import type { Ledger } from '../types'
import { Icon } from './Icon'
import { LedgerFigures } from './LedgerFigures'
import { LedgerHeatmap } from './LedgerHeatmap'
import { LedgerRail } from './LedgerRail'
import { LedgerSection } from './LedgerSection'
import { LedgerShowAll } from './LedgerShowAll'
import { LedgerTable } from './LedgerTable'
import { LedgerUsage } from './LedgerUsage'
import { LedgerWork } from './LedgerWork'
import { Loading } from './Loading'

type LedgerViewProps = { days: number; project: string }

type Table = 'projects' | 'spend' | 'work'

const SECTIONS = [
  { id: 'activity', title: 'Activity' },
  { id: 'delivery', title: 'Delivery' },
  { id: 'projects', title: 'Projects' },
  { id: 'spend', title: 'Spend' },
  { id: 'work', title: 'Work' },
]

const ROWS = 10

const average = (ms: number) => (ms ? duration(ms) : '—')
const hours = (ms: number) => {
  const h = Math.round(ms / 3_600_000)
  return h ? `${h.toLocaleString()}h` : duration(ms)
}

export function LedgerView({ days, project }: LedgerViewProps) {
  const { report } = useAgentos()
  const [ledger, setLedger] = useState<Ledger | null>(null)
  const [active, setActive] = useState(SECTIONS[0].id)
  const [full, setFull] = useState<Table | null>(null)
  const content = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let current = true
    report(async () => {
      const next = await api.ledger(days, project)
      if (current) setLedger(next)
    })
    return () => {
      current = false
    }
  }, [days, project, report])
  const loading = useMinimumLoading(ledger === null)

  const loaded = !loading && ledger !== null && ledger.since !== '' && full === null
  useEffect(() => {
    const scroller = content.current?.closest('[data-scroll]')
    if (!loaded || !scroller) return
    // The active section is the last one whose top has passed a third of the way down, or the last
    // section once the panel is scrolled to the end, since it may be too short to reach the top.
    const spy = () => {
      const sections = [...scroller.querySelectorAll<HTMLElement>('[data-section]')]
      const box = scroller.getBoundingClientRect()
      const line = box.top + box.height / 3
      const atEnd = scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - 2
      const passed = sections.filter((s) => s.getBoundingClientRect().top <= line)
      const hit = atEnd ? sections.at(-1) : passed.at(-1)
      setActive(hit?.dataset.section ?? SECTIONS[0].id)
    }
    spy()
    scroller.addEventListener('scroll', spy, { passive: true })
    return () => scroller.removeEventListener('scroll', spy)
  }, [loaded])

  useEffect(() => {
    if (full !== null) content.current?.closest('[data-scroll]')?.scrollTo({ top: 0 })
  }, [full])

  if (loading || ledger === null) return <Loading />
  if (ledger.since === '') {
    return (
      <div className="flex flex-col gap-1 py-16 text-center">
        <p className="text-body font-medium">Nothing recorded yet</p>
        <p className="text-small text-dim">Chats, tasks, prompts, session time and tokens are counted from now on.</p>
      </div>
    )
  }

  const { totals, usage } = ledger
  const spent = usage.totals
  const perTask = spent.sessions ? spent.tokens / spent.sessions : 0
  const sections = project ? SECTIONS.filter((s) => s.id !== 'projects') : SECTIONS
  const pick = (id: string) => {
    document.getElementById(`ledger-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
  const back = (table: Table) => {
    setFull(null)
    requestAnimationFrame(() => document.getElementById(`ledger-${table}`)?.scrollIntoView({ block: 'start' }))
  }

  const tables: Record<Table, { title: string; count: number; table(limit?: number): ReactNode }> = {
    projects: { title: 'Projects', count: ledger.projects.length, table: (limit) => <LedgerTable projects={ledger.projects} limit={limit} /> },
    spend: { title: 'Spend by model', count: usage.byModel.length, table: (limit) => <LedgerUsage usage={usage} limit={limit} /> },
    work: { title: 'Work', count: ledger.work.length, table: (limit) => <LedgerWork work={ledger.work} limit={limit} /> },
  }
  const showAll = (table: Table) => <LedgerShowAll count={tables[table].count} limit={ROWS} onShow={() => setFull(table)} />

  if (full !== null) {
    return (
      <div ref={content} className="mx-auto flex max-w-5xl flex-col gap-4">
        <div className="flex items-center gap-3">
          <button className="btn btn-ghost h-7 gap-1.5 px-2" onClick={() => back(full)}>
            <Icon name="back" /> Back
          </button>
          <h2 className="text-title font-semibold">{tables[full].title}</h2>
          <span className="text-small text-dim">{tables[full].count.toLocaleString()} rows</span>
        </div>
        {tables[full].table()}
      </div>
    )
  }

  return (
    <div className="@container">
      <div className="mx-auto flex max-w-5xl flex-col gap-6 @xl:grid @xl:grid-cols-[7.5rem_minmax(0,1fr)] @xl:gap-6">
        <LedgerRail sections={sections} active={active} onPick={pick} />
        <div ref={content} className="flex min-w-0 flex-col gap-8">
          <LedgerSection id="activity" title="Activity" hint={`since ${ledger.since}`}>
            <LedgerFigures
              figures={[
                {
                  label: 'Chats',
                  value: totals.chats.toLocaleString(),
                  hint: 'in no other figure',
                },
                {
                  label: 'Tasks',
                  value: totals.sessions.toLocaleString(),
                  hint: `${totals.issueSessions.toLocaleString()} from an issue`,
                },
                {
                  label: 'Prompts',
                  value: totals.prompts.toLocaleString(),
                  hint: 'sent to agents',
                },
                {
                  label: 'Session time',
                  value: hours(totals.sessionMs),
                  hint: 'open, start to end',
                },
              ]}
            />
            <LedgerHeatmap heat={ledger.heat} />
          </LedgerSection>
          <LedgerSection id="delivery" title="Delivery" hint="what the work shipped">
            <LedgerFigures
              figures={[
                {
                  label: 'PRs merged',
                  value: totals.prsMerged.toLocaleString(),
                  hint: `${totals.prsOpened.toLocaleString()} opened, ${totals.prsClosed.toLocaleString()} closed`,
                },
                {
                  label: 'Issues closed',
                  value: totals.issuesClosed.toLocaleString(),
                  hint: `${totals.issuesFiled.toLocaleString()} filed`,
                },
                {
                  label: 'Task length',
                  value: average(totals.avgSessionMs),
                  hint: 'average',
                },
                {
                  label: 'Lead time',
                  value: average(totals.avgLeadMs),
                  hint: 'PR open to merge',
                },
              ]}
            />
          </LedgerSection>
          {!project && (
            <LedgerSection id="projects" title="Projects" hint={`${ledger.projects.length.toLocaleString()} with activity`}>
              {tables.projects.table(ROWS)}
              {showAll('projects')}
            </LedgerSection>
          )}
          <LedgerSection id="spend" title="Spend" hint="estimated">
            <LedgerFigures
              figures={[
                {
                  label: 'Tokens',
                  value: tokenCount(spent.tokens),
                  hint: `~${dollars(spent.cost)} estimated`,
                  title: priceNote(usage.pricesAsOf),
                },
                {
                  label: 'Tokens per task',
                  value: perTask ? tokenCount(Math.round(perTask)) : '—',
                  hint: perTask ? `~${dollars(spent.cost / spent.sessions)} estimated` : 'no usage yet',
                  title: priceNote(usage.pricesAsOf),
                },
              ]}
            />
            {usage.byModel.length > 0 && tables.spend.table(ROWS)}
            {showAll('spend')}
          </LedgerSection>
          <LedgerSection id="work" title="Work" hint="tokens by issue and task">
            {ledger.work.length > 0 ? tables.work.table(ROWS) : <p className="text-small text-dim">No usage yet.</p>}
            {showAll('work')}
          </LedgerSection>
        </div>
      </div>
    </div>
  )
}
