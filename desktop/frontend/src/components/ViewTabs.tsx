import { useAgentos } from '../AgentosContext'
import type { SessionView } from '../AgentosContext'
import type { Session } from '../types'

type ViewTabsProps = { session: Session; view: SessionView }

export function ViewTabs({ session, view }: ViewTabsProps) {
  const { setSessionView, browserStates } = useAgentos()
  const browserOpen = browserStates[session.id]?.open ?? session.browser
  const tabs: { id: SessionView; label: string; count?: number; live?: boolean }[] = [
    { id: 'terminal', label: 'Terminal' },
    { id: 'browser', label: 'Browser', live: browserOpen },
    { id: 'evidence', label: 'Evidence', count: session.evidence },
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
            className="flex h-9 items-center gap-1.5 border-b-2 px-2.5 short:h-8 text-body font-medium"
            style={{ borderColor: active ? 'var(--accent)' : 'transparent', color: active ? 'var(--text)' : 'var(--text-soft)' }}
            onClick={() => setSessionView(session.id, tab.id)}
          >
            {tab.label}
            {tab.live && <span className="dot" data-state="working" title="Browser open" />}
            {tab.count !== undefined && tab.count > 0 && <span className="mono text-small text-dim">({tab.count})</span>}
          </button>
        )
      })}
    </div>
  )
}
