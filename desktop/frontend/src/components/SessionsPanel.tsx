import { useEffect, useRef } from 'react'
import type { KeyboardEvent, RefObject } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { useListNav } from '../useListNav'
import { useToggleAnimation } from '../useToggleAnimation'
import { CleanupsMenu } from './CleanupsMenu'
import { EdgeStrip } from './EdgeStrip'
import { Icon } from './Icon'
import { NewSessionRow } from './NewSessionRow'
import { SessionRow } from './SessionRow'

const COMPACT_BELOW_REM = 17

type SessionsListProps = { panel: RefObject<HTMLElement>; full: boolean }

function SessionsList({ panel, full }: SessionsListProps) {
  const { sessions, selectedId, select, focus, setComposing } = useAgentos()
  const { widths, setSessionsOpen, returnToTerminal } = useLayout()
  const list = useRef<HTMLDivElement>(null)

  const nav = useListNav(sessions.length + 1, {
    onEnter(index) {
      const session = sessions[index]
      if (!session) {
        setComposing(true)
        return
      }
      select(session.id)
      focus('terminal')
    },
  })

  useEffect(() => {
    list.current?.querySelector('[data-cursor="true"]')?.scrollIntoView({ block: 'nearest' })
  }, [nav.cursor])

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      returnToTerminal()
      return
    }
    if (nav.handle(e)) e.preventDefault()
  }

  const compact = !full && widths.sessions < COMPACT_BELOW_REM

  return (
    <aside
      ref={panel}
      data-panel="sessions"
      tabIndex={-1}
      className={`panel flex min-h-0 flex-col overflow-hidden ${full ? 'min-w-0 flex-1' : 'flex-none'}`}
      style={full ? undefined : { width: `${widths.sessions}rem` }}
      onKeyDown={onKeyDown}
      onFocus={(e) => {
        if (e.target !== e.currentTarget) return
        const at = sessions.findIndex((s) => s.id === selectedId)
        nav.setCursor(Math.max(at, 0))
      }}
    >
      <div className="panel-head flex h-10 flex-none items-center gap-2 border-b border-line pr-[0.3125rem] pl-4">
        <span className="label">Sessions</span>
        <span className="mono text-small text-dim">{sessions.length}</span>
        <span className="ml-auto" />
        <CleanupsMenu />
        {!full && (
          <button
            className="btn btn-ghost h-8 w-8 flex-none justify-center px-0"
            title="Collapse sessions (⌘⌥B)"
            aria-label="Collapse sessions"
            onClick={() => setSessionsOpen(false)}
          >
            <Icon name="panel-right" size={15} />
          </button>
        )}
      </div>
      <div ref={list} className="min-h-0 flex-1 divide-y divide-line overflow-y-auto">
        {sessions.map((s, i) => (
          <SessionRow key={s.id} session={s} selected={s.id === selectedId} cursor={nav.cursor === i} compact={compact} />
        ))}
        {sessions.length === 0 && <p className="px-4 py-6 text-small text-dim">No sessions yet.</p>}
      </div>
      <NewSessionRow cursor={nav.cursor === sessions.length} compact={compact} />
    </aside>
  )
}

export function SessionsPanel() {
  const { focusRequest } = useAgentos()
  const { mode, sessionsOpen, widths } = useLayout()
  const panel = useRef<HTMLElement>(null)
  const animating = useToggleAnimation(sessionsOpen)

  useEffect(() => {
    if (focusRequest.target === 'sessions') panel.current?.focus()
  }, [focusRequest, sessionsOpen])

  if (mode === 'narrow') return sessionsOpen ? <SessionsList panel={panel} full /> : null

  return (
    <div
      className="side-shell"
      data-open={sessionsOpen}
      data-animating={animating}
      style={{ width: sessionsOpen ? `${widths.sessions}rem` : 'var(--edge-strip-w)', maxWidth: '60%' }}
    >
      <SessionsList panel={panel} full={false} />
      <EdgeStrip side="right" inShell />
    </div>
  )
}
