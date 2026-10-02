import { useEffect, useRef } from 'react'
import type { KeyboardEvent, MutableRefObject, RefObject } from 'react'
import { useAgentos } from '../AgentosContext'
import type { SidebarTab } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { isTyping } from '../useListNav'
import { useToggleAnimation } from '../useToggleAnimation'
import { EdgeStrip } from './EdgeStrip'
import { Icon } from './Icon'
import { Notes } from './Notes'
import { Queue } from './Queue'

export type NavHandler = (e: KeyboardEvent) => boolean
export type NavRef = MutableRefObject<NavHandler | null>

type TabButtonProps = { tab: SidebarTab; label: string; count: number }

function TabButton({ tab, label, count }: TabButtonProps) {
  const { sidebarTab } = useAgentos()
  const { showSidebarTab } = useLayout()
  const active = sidebarTab === tab
  return (
    <button
      role="tab"
      aria-selected={active}
      className="flex h-10 items-center gap-2 border-b-2 px-3 text-body font-medium"
      style={{ borderColor: active ? 'var(--accent)' : 'transparent', color: active ? 'var(--text)' : 'var(--text-soft)' }}
      onClick={() => showSidebarTab(tab)}
    >
      {label}
      <span className="mono text-small text-dim">{count}</span>
    </button>
  )
}

const TABS: SidebarTab[] = ['queue', 'notes']

type SidebarPanelProps = { panel: RefObject<HTMLElement>; nav: NavRef; overlay: boolean; full: boolean }

function SidebarPanel({ panel, nav, overlay, full }: SidebarPanelProps) {
  const { sidebarTab, issues, notes } = useAgentos()
  const { widths, setSidebarOpen, showSidebarTab, returnToTerminal } = useLayout()

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
        ...(full ? {} : { width: `${widths.sidebar}rem`, maxWidth: overlay ? 'calc(100% - 1.5rem)' : undefined }),
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
            <TabButton tab="queue" label="Queue" count={issues.length} />
            <TabButton tab="notes" label="Notes" count={notes.filter((n) => !n.archived).length} />
          </>
        )}
      </div>
      {sidebarTab === 'queue' ? <Queue nav={nav} /> : <Notes nav={nav} />}
    </aside>
  )
}

export function Sidebar() {
  const { focusRequest } = useAgentos()
  const { mode, sidebarOpen, widths, setSidebarOpen } = useLayout()
  const panel = useRef<HTMLElement>(null)
  const nav: NavRef = useRef(null)
  const animating = useToggleAnimation(sidebarOpen)
  const overlayOpen = mode === 'medium' && sidebarOpen

  useEffect(() => {
    if (focusRequest.target === 'sidebar') panel.current?.focus()
  }, [focusRequest, sidebarOpen])

  useEffect(() => {
    if (overlayOpen) panel.current?.focus()
  }, [overlayOpen])

  if (mode === 'narrow') return sidebarOpen ? <SidebarPanel panel={panel} nav={nav} overlay={false} full /> : null

  if (mode === 'medium') {
    return (
      <>
        <EdgeStrip side="left" />
        {sidebarOpen && (
          <>
            <div className="absolute inset-0" style={{ zIndex: 'calc(var(--z-sidebar-overlay) - 1)' }} onMouseDown={() => setSidebarOpen(false)} />
            <SidebarPanel panel={panel} nav={nav} overlay full={false} />
          </>
        )}
      </>
    )
  }

  return (
    <div
      className="side-shell"
      data-open={sidebarOpen}
      data-animating={animating}
      style={{ width: sidebarOpen ? `${widths.sidebar}rem` : 'var(--edge-strip-w)', maxWidth: '60%' }}
    >
      <SidebarPanel panel={panel} nav={nav} overlay={false} full={false} />
      <EdgeStrip side="left" inShell />
    </div>
  )
}
