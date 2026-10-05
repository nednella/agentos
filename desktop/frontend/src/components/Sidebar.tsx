import { useRef } from 'react'
import type { KeyboardEvent, MutableRefObject, RefObject } from 'react'
import { useAgentos } from '../AgentosContext'
import type { SidebarTab } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { isTyping } from '../useListNav'
import { useFocusRequest } from '../useFocusRequest'
import { useToggleAnimation } from '../useToggleAnimation'
import { EdgeStrip } from './EdgeStrip'
import { Icon } from './Icon'

export type NavHandler = (e: KeyboardEvent) => boolean
export type NavRef = MutableRefObject<NavHandler | null>

type TabButtonProps = { tab: SidebarTab; label: string; count: number; tight: boolean }

function TabButton({ tab, label, count, tight }: TabButtonProps) {
  const { sidebarTab } = useAgentos()
  const { showSidebarTab } = useLayout()
  const active = sidebarTab === tab
  return (
    <button
      role="tab"
      aria-selected={active}
      className={`flex h-10 items-center gap-2 border-b-2 text-body font-medium ${tight ? 'px-2' : 'px-3'}`}
      title={`${label} (${count})`}
      style={{ borderColor: active ? 'var(--accent)' : 'transparent', color: active ? 'var(--text)' : 'var(--text-soft)' }}
      onClick={() => showSidebarTab(tab)}
    >
      {label}
      {!tight && <span className="mono text-small text-dim">{count}</span>}
    </button>
  )
}

const TABS: SidebarTab[] = ['queue', 'notes']

type SidebarPanelProps = { panel: RefObject<HTMLElement>; nav: NavRef; overlay: boolean; full: boolean; widthRem: number }

function SidebarPanel({ panel, nav, overlay, full, widthRem }: SidebarPanelProps) {
  const { sidebarTab, issues, notes } = useAgentos()
  const tight = widthRem < 15
  const { setSidebarOpen, showSidebarTab, returnToTerminal } = useLayout()

  const onKeyDown = (e: KeyboardEvent) => {
    if (isTyping(e.target)) return
    if (e.key === 'Escape') {
      e.preventDefault()
      returnToTerminal()
      return
    }
    if (nav.current?.(e)) {
      e.preventDefault()
      return
    }
    if (e.metaKey || e.ctrlKey || e.altKey) return
    if (['ArrowLeft', 'ArrowRight', 'a', 'd'].includes(e.key)) {
      e.preventDefault()
      showSidebarTab(TABS[(TABS.indexOf(sidebarTab) + 1) % TABS.length])
    }
  }

  return (
    <aside
      ref={panel}
      data-panel="sidebar"
      data-tab={sidebarTab}
      tabIndex={-1}
      className={`panel flex min-h-0 flex-col overflow-hidden ${full ? 'min-w-0 flex-1' : 'flex-none'} ${overlay ? 'fade-in absolute top-3 bottom-3 left-3 shadow-2xl' : ''}`}
      style={{
        ...(full ? {} : { width: `${widthRem}rem`, maxWidth: overlay ? 'calc(100% - 1.5rem)' : undefined }),
        zIndex: overlay ? 'var(--z-sidebar-overlay)' : undefined,
      }}
      onKeyDown={onKeyDown}
    >
      <div role="tablist" className="panel-head flex flex-none items-center border-b border-line">
        {!full && (
          <button
            className="btn btn-ghost my-1 mr-1 ml-[0.3125rem] h-8 w-8 flex-none justify-center px-0"
            title={overlay ? 'Close (Esc)' : 'Collapse queue and notes (⌘B)'}
            aria-label={overlay ? 'Close queue and notes' : 'Collapse queue and notes'}
            onClick={() => setSidebarOpen(false)}
          >
            <Icon name={overlay ? 'close' : 'panel-left'} size={15} />
          </button>
        )}
        {full ? (
          <span className="label px-4">{sidebarTab}</span>
        ) : (
          <>
            <TabButton tab="queue" label="Queue" count={issues.length} tight={tight} />
            <TabButton tab="notes" label="Notes" count={notes.filter((n) => !n.archived).length} tight={tight} />
          </>
        )}
      </div>
    </aside>
  )
}

export function Sidebar() {
  const { mode, sidebarOpen, sidebarPeek, widths, peekWidths, closePeek } = useLayout()
  const shellPanel = useRef<HTMLElement>(null)
  const peekPanel = useRef<HTMLElement>(null)
  const shellNav: NavRef = useRef(null)
  const peekNav: NavRef = useRef(null)
  const animating = useToggleAnimation(sidebarOpen)

  useFocusRequest('sidebar', shellPanel, sidebarOpen)
  useFocusRequest('sidebar', peekPanel, sidebarPeek)

  if (mode === 'narrow') return sidebarOpen ? <SidebarPanel panel={shellPanel} nav={shellNav} overlay={false} full widthRem={0} /> : null

  return (
    <>
      <div
        className="side-shell"
        data-open={sidebarOpen}
        data-animating={animating}
        style={{ width: sidebarOpen ? `${widths.sidebar}rem` : 'var(--edge-strip-w)' }}
      >
        <SidebarPanel panel={shellPanel} nav={shellNav} overlay={false} full={false} widthRem={widths.sidebar} />
        <EdgeStrip side="left" inShell />
      </div>
      {sidebarPeek && (
        <>
          <div className="absolute inset-0" style={{ zIndex: 'calc(var(--z-sidebar-overlay) - 1)' }} onMouseDown={closePeek} />
          <SidebarPanel panel={peekPanel} nav={peekNav} overlay full={false} widthRem={peekWidths.sidebar} />
        </>
      )}
    </>
  )
}
