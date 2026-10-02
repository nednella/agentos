import { useEffect, useRef } from 'react'
import { useAgentos } from '../AgentosContext'
import { BrowserTab } from './BrowserTab'
import { EmptyState } from './EmptyState'
import { EvidenceTab } from './EvidenceTab'
import { Terminal } from './Terminal'
import { Timeline } from './Timeline'
import { ViewTabs } from './ViewTabs'
import { ViewportHeader } from './ViewportHeader'

export function Viewport() {
  const { sessions, selectedId, openedIds, focusRequest, viewOf } = useAgentos()
  const panel = useRef<HTMLElement>(null)
  const selected = sessions.find((s) => s.id === selectedId)
  const view = viewOf(selectedId)

  const handled = useRef(focusRequest.n)
  useEffect(() => {
    if (focusRequest.n === handled.current) return
    handled.current = focusRequest.n
    if ((!selected || view !== 'terminal') && focusRequest.target === 'terminal') panel.current?.focus()
  }, [selected, view, focusRequest])

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
      <ViewportHeader session={selected} />
      <Timeline session={selected} size="full" />
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
