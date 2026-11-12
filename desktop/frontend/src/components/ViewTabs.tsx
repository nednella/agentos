import { useAgentos } from '../AgentosContext'
import type { SessionView } from '../AgentosContext'
import type { Session } from '../types'

type ViewTabsProps = { session: Session; view: SessionView }

export function ViewTabs({ session, view }: ViewTabsProps) {
  const { setSessionView, browserStates } = useAgentos()
  const browserOpen = browserStates[session.id]?.open ?? session.browser
  const tabs: { id: SessionView; label: string; count?: number; live?: string }[] = [
    { id: 'terminal', label: 'Agent' },
    { id: 'shell', label: 'Shell', live: session.shell ? 'Shell running' : undefined },
    { id: 'browser', label: 'Browser', live: browserOpen ? 'Browser open' : undefined },
    { id: 'evidence', label: 'Media', count: session.evidence },
  ]

  return (
    <div role="tablist" aria-label="Session views" className="flex flex-none items-center gap-1 border-t border-line px-3">
      {tabs.map((tab) => {
        const active = view === tab.id
        return (
          <button
            key={tab.id}
            role="tab"
            aria-selected={active}
            className="flex h-10 items-center gap-1.5 border-b-2 px-2.5 text-body font-medium"
            style={{ borderColor: active ? 'var(--accent)' : 'transparent', color: active ? 'var(--text)' : 'var(--text-soft)' }}
            onClick={() => setSessionView(session.id, tab.id)}
          >
            {tab.label}
            {tab.live && <span className="dot" data-state="working" title={tab.live} />}
            {tab.count !== undefined && tab.count > 0 && <span className="mono text-small text-dim">({tab.count})</span>}
          </button>
        )
      })}
    </div>
  )
}
