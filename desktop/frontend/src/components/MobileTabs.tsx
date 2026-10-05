import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import type { MobilePanel } from '../LayoutContext'

type TabProps = { panel: MobilePanel; label: string; badge?: string; urgent?: boolean }

function Tab({ panel, label, badge, urgent }: TabProps) {
  const { mobilePanel, setMobilePanel, showSidebarTab } = useLayout()
  const active = mobilePanel === panel
  return (
    <button
      role="tab"
      aria-selected={active}
      className="flex h-11 min-w-0 flex-1 flex-col short:h-8 items-center justify-center gap-0.5 border-t-2 text-small font-medium"
      style={{ borderColor: active ? 'var(--accent)' : 'transparent', color: active ? 'var(--text)' : 'var(--text-soft)' }}
      onClick={() => (panel === 'queue' || panel === 'notes' ? showSidebarTab(panel) : setMobilePanel(panel))}
    >
      <span className="flex items-center gap-1.5">
        {label}
        {urgent && <span className="dot" data-state="waiting" />}
      </span>
      {badge && <span className="mono short:hidden text-label text-dim">{badge}</span>}
    </button>
  )
}

export function MobileTabs() {
  const { issues, notes } = useAgentos()
  return (
    <nav role="tablist" aria-label="Panels" className="flex flex-none border-t border-line bg-surface">
      <Tab panel="queue" label="Queue" badge={String(issues.length)} />
      <Tab panel="notes" label="Notes" badge={String(notes.filter((n) => !n.archived).length)} />
    </nav>
  )
}
