import { useRef } from 'react'
import { useAgentos } from '../AgentosContext'
import { useFocusRequest } from '../useFocusRequest'
import { BrowserTab } from './BrowserTab'
import { DetachedState } from './DetachedState'
import { EmptyState } from './EmptyState'
import { EvidenceTab } from './EvidenceTab'
import { Terminal } from './Terminal'
import { Timeline } from './Timeline'
import { ViewTabs } from './ViewTabs'
import { ViewportActions } from './ViewportActions'
import { ViewportHeader } from './ViewportHeader'

export function Viewport() {
  const { sessions, selectedId, openedIds, viewOf } = useAgentos()
  const panel = useRef<HTMLElement>(null)
  const selected = sessions.find((s) => s.id === selectedId)
  const view = viewOf(selectedId)

  useFocusRequest('terminal', panel, !selected || view !== 'terminal')

  if (!selected) {
    return (
      <section ref={panel} data-panel="terminal" tabIndex={-1} className="panel h-full min-h-0 min-w-0">
        {sessions.length > 0 ? <DetachedState /> : <EmptyState />}
      </section>
    )
  }

  return (
    <section
      ref={panel}
      data-panel="terminal"
      data-state={selected.state}
      tabIndex={-1}
      className="panel flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
    >
      <ViewportHeader key={selected.id} session={selected} />
      <Timeline session={selected} size="full" />
      <ViewportActions session={selected} />
      <ViewTabs session={selected} view={view} />
      <div className="relative min-h-0 flex-1 border-t border-line bg-term">
        <div className={view === 'terminal' ? 'absolute inset-0' : 'hidden'}>
          {openedIds
            .filter((id) => sessions.some((s) => s.id === id))
            .map((id) => (
              <Terminal key={id} id={id} active={id === selectedId} />
            ))}
        </div>
        {view === 'browser' && <BrowserTab key={selected.id} session={selected} />}
        {view === 'evidence' && <EvidenceTab key={selected.id} session={selected} />}
      </div>
    </section>
  )
}
