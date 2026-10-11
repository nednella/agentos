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
  { value: 'work' as const, label: 'Stats' },
  { value: 'interruptions' as const, label: 'Interruptions' },
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
const PROJECT_KEY = 'agentos.statsProject'

export function Stats() {
  const { focusRequest, projects } = useAgentos()
  const { closeCentre } = useLayout()
  const [tab, setTab] = useState<Tab>(() => readStored(TAB_KEY, 'work'))
  const [days, setDays] = useState(() => readStored(RANGE_KEY, 7))
  const [workDays, setWorkDays] = useState(() => readStored(WORK_RANGE_KEY, 30))
  const [picked, setPicked] = useState(() => readStored(PROJECT_KEY, ''))
  const panel = useRef<HTMLElement>(null)
  const work = tab === 'work'
  const project = projects.some((p) => p.key === picked) ? picked : ''

  useEffect(() => {
    if (focusRequest.target === 'terminal') panel.current?.focus()
  }, [focusRequest])

  useEffect(() => panel.current?.focus(), [])

  const pickTab = (next: Tab) => {
    setTab(next)
    writeStored(TAB_KEY, next)
  }
  const pickProject = (next: string) => {
    setPicked(next)
    writeStored(PROJECT_KEY, next)
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
      aria-label={work ? 'Stats' : 'Interruption stats'}
      className="panel @container flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
      onKeyDown={(e) => e.key === 'Escape' && closeCentre()}
    >
      <header className="panel-head grid flex-none grid-cols-[minmax(0,1fr)_auto] items-center justify-items-start gap-x-3 gap-y-2 border-b border-line px-4 py-2.5 @xl:flex">
        <h2 className="sr-only">{work ? 'Stats' : 'What interrupts you'}</h2>
        <SegmentedControl label="View" options={TABS} value={tab} onChange={pickTab} />
        <button className="btn btn-ghost h-7 w-7 justify-center justify-self-end px-0 @xl:order-last" title="Back to the terminal (Esc)" aria-label="Back to the terminal" onClick={closeCentre}>
          <Icon name="close" />
        </button>
        <div className="col-span-2 flex flex-wrap justify-self-stretch items-center justify-between gap-x-3 gap-y-2 @xl:contents">
          <select aria-label="Project" className="btn btn-ghost h-7 px-2" value={project} onChange={(e) => pickProject(e.target.value)}>
            <option value="">All projects</option>
            {projects.map((p) => (
              <option key={p.key} value={p.key}>
                {p.name}
              </option>
            ))}
          </select>
          <div className="@xl:ml-auto">
            <SegmentedControl
              label="Range"
              options={work ? WORK_RANGES : RANGES}
              value={String(work ? workDays : days)}
              onChange={pickRange}
            />
          </div>
        </div>
      </header>
      <div data-scroll className="min-h-0 flex-1 overflow-y-auto px-6 py-4">
        {work ? <LedgerView days={workDays} project={project} /> : <StatsInterruptions days={days} project={project} />}
      </div>
    </section>
  )
}
