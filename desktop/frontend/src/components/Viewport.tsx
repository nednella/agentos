import { useRef } from 'react'
import { useAgentos, useAgentosSelector } from '../AgentosContext'
import { useFocusRequest } from '../useFocusRequest'
import { BrowserTab } from './BrowserTab'
import { EmptyState } from './EmptyState'
import { EvidenceTab } from './EvidenceTab'
import { Terminal } from './Terminal'
import { ViewTabs } from './ViewTabs'
import { ViewportActions } from './ViewportActions'
import { ViewportHeader } from './ViewportHeader'

export function Viewport() {
  const { selectedId, viewOf } = useAgentos('selectedId', 'viewOf')
  const selected = useAgentosSelector((a) => a.sessions.find((s) => s.id === a.selectedId))
  const panel = useRef<HTMLElement>(null)
  const view = viewOf(selectedId)

  useFocusRequest('terminal', panel, !selected || view !== 'terminal')

  if (!selected) {
    return (
      <section ref={panel} data-panel="terminal" tabIndex={-1} className="panel h-full min-h-0 min-w-0">
        <EmptyState />
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
      <ViewportActions session={selected} />
      <ViewTabs session={selected} view={view} />
      <div className="relative min-h-0 flex-1 border-t border-line bg-term">
        <div className={view === 'terminal' ? 'absolute inset-0' : 'hidden'}>
          <Terminal key={selected.id} id={selected.id} active />
        </div>
        {view === 'browser' && <BrowserTab key={selected.id} session={selected} />}
        {view === 'evidence' && <EvidenceTab key={selected.id} session={selected} />}
      </div>
    </section>
  )
}
