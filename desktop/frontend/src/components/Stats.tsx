import { useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { readStored, writeStored } from '../storage'
import { Icon } from './Icon'
import { LedgerView } from './LedgerView'
import { SegmentedControl } from './SegmentedControl'
import { StatsInterruptions } from './StatsInterruptions'

type Tab = 'interruptions' | 'work'

const TABS = [
  { value: 'interruptions' as const, label: 'Interruptions' },
  { value: 'work' as const, label: 'Your work' },
]
const RANGES = [7, 14, 30].map((days) => ({ value: String(days), label: `${days} days` }))
const WORK_RANGES = [
  { value: '30', label: '30 days' },
  { value: '365', label: 'Year' },
  { value: '0', label: 'Lifetime' },
]
const TAB_KEY = 'agentos.statsTab'
const RANGE_KEY = 'agentos.statsDays'
const WORK_RANGE_KEY = 'agentos.statsWorkDays'

export function Stats() {
  const { focusRequest, project } = useAgentos()
  const { closeCentre } = useLayout()
  const [tab, setTab] = useState<Tab>(() => readStored(TAB_KEY, 'interruptions'))
  const [days, setDays] = useState(() => readStored(RANGE_KEY, 7))
  const [workDays, setWorkDays] = useState(() => readStored(WORK_RANGE_KEY, 30))
  const panel = useRef<HTMLElement>(null)
  const work = tab === 'work'

  useEffect(() => {
    if (focusRequest.target === 'terminal') panel.current?.focus()
  }, [focusRequest])

  useEffect(() => panel.current?.focus(), [])

  const pickTab = (next: Tab) => {
    setTab(next)
    writeStored(TAB_KEY, next)
  }
  const pickRange = (next: string) => {
    if (work) {
      setWorkDays(Number(next))
      writeStored(WORK_RANGE_KEY, Number(next))
    } else {
      setDays(Number(next))
      writeStored(RANGE_KEY, Number(next))
    }
  }

  return (
    <section
      ref={panel}
      data-panel="terminal"
      tabIndex={-1}
      aria-label={work ? 'Your work' : 'Interruption stats'}
      className="panel flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
      onKeyDown={(e) => e.key === 'Escape' && closeCentre()}
    >
      <header className="panel-head flex flex-none flex-wrap items-center gap-3 border-b border-line px-4 py-2.5">
        <h2 className="sr-only">{work ? 'Your work' : 'What interrupts you'}</h2>
        <SegmentedControl label="View" options={TABS} value={tab} onChange={pickTab} />
        <span className="text-small text-dim">{work ? 'all projects' : project?.name}</span>
        <div className="ml-auto">
          <SegmentedControl
            label="Range"
            options={work ? WORK_RANGES : RANGES}
            value={String(work ? workDays : days)}
            onChange={pickRange}
          />
        </div>
        <button className="btn btn-ghost h-7 w-7 justify-center px-0" title="Back to the terminal (Esc)" aria-label="Back to the terminal" onClick={closeCentre}>
          <Icon name="close" />
        </button>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {work ? <LedgerView days={workDays} /> : <StatsInterruptions days={days} />}
      </div>
    </section>
  )
}
