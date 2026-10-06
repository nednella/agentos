import { useRef } from 'react'
import type { KeyboardEvent, RefObject } from 'react'
import { useAgentos } from '../AgentosContext'
import { useFocusRequest } from '../useFocusRequest'
import { useLayout } from '../LayoutContext'
import { useListNav } from '../useListNav'
import { useScrollCursorIntoView } from '../useScrollCursorIntoView'
import { useToggleAnimation } from '../useToggleAnimation'
import { CleanupsMenu } from './CleanupsMenu'
import { EdgeStrip } from './EdgeStrip'
import { Icon } from './Icon'
import { NewSessionRow } from './NewSessionRow'
import { SessionRow } from './SessionRow'

const COMPACT_BELOW_REM = 17
const NEW_SESSION_KEY = 'new-session'

type SessionsListProps = { panel: RefObject<HTMLElement>; full: boolean; overlay: boolean; widthRem: number }

function SessionsList({ panel, full, overlay, widthRem }: SessionsListProps) {
  const { sessions, selectedId, select, focus, setComposing } = useAgentos()
  const { mode, setSessionsOpen, returnToTerminal } = useLayout()
  const list = useRef<HTMLDivElement>(null)

  const nav = useListNav([...sessions.map((s) => s.id), NEW_SESSION_KEY], {
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

  useScrollCursorIntoView(list, nav.cursorKey)

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      returnToTerminal()
      return
    }
    if (nav.handle(e)) e.preventDefault()
  }

  const compact = !full && widthRem < COMPACT_BELOW_REM
  const dense = mode === 'compact' && !overlay

  return (
    <aside
      ref={panel}
      data-panel="sessions"
      tabIndex={-1}
      className={`panel flex min-h-0 flex-col overflow-hidden ${full ? 'min-w-0 flex-1' : 'flex-none'} ${overlay ? 'fade-in absolute top-3 right-3 bottom-3 shadow-2xl' : ''}`}
      style={full ? undefined : { width: `${widthRem}rem`, maxWidth: overlay ? 'calc(100% - 1.5rem)' : undefined, zIndex: overlay ? 'var(--z-sidebar-overlay)' : undefined }}
      onKeyDown={onKeyDown}
      onFocus={(e) => {
        if (e.target !== e.currentTarget) return
        if (selectedId) nav.setCursorKey(selectedId)
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
            title={overlay ? 'Close (Esc)' : 'Collapse sessions (⌘⌥B)'}
            aria-label={overlay ? 'Close sessions' : 'Collapse sessions'}
            onClick={() => setSessionsOpen(false)}
          >
            <Icon name={overlay ? 'close' : 'panel-right'} size={15} />
          </button>
        )}
      </div>
      <div ref={list} className="min-h-0 flex-1 divide-y divide-line overflow-y-auto">
        {sessions.map((s) => (
          <SessionRow key={s.id} session={s} selected={s.id === selectedId} cursor={nav.cursorKey === s.id} compact={compact} dense={dense} />
        ))}
        {sessions.length === 0 && <p className="flex h-full items-center justify-center px-4 text-center text-body font-medium">No sessions yet</p>}
      </div>
      <NewSessionRow cursor={nav.cursorKey === NEW_SESSION_KEY} compact={compact} />
    </aside>
  )
}

export function SessionsPanel() {
  const { mode, sessionsOpen, sessionsPeek, widths, peekWidths, closePeek } = useLayout()
  const shellPanel = useRef<HTMLElement>(null)
  const peekPanel = useRef<HTMLElement>(null)
  const animating = useToggleAnimation(sessionsOpen)

  useFocusRequest('sessions', shellPanel, sessionsOpen)
  useFocusRequest('sessions', peekPanel, sessionsPeek)

  if (mode === 'narrow') return sessionsOpen ? <SessionsList panel={shellPanel} full overlay={false} widthRem={0} /> : null

  return (
    <>
      <div
        className="side-shell"
        data-open={sessionsOpen}
        data-animating={animating}
        style={{ width: sessionsOpen ? `${widths.sessions}rem` : 'var(--edge-strip-w)' }}
      >
        <SessionsList panel={shellPanel} full={false} overlay={false} widthRem={widths.sessions} />
        <EdgeStrip side="right" inShell />
      </div>
      {sessionsPeek && (
        <>
          <div className="absolute inset-0" style={{ zIndex: 'calc(var(--z-sidebar-overlay) - 1)' }} onMouseDown={closePeek} />
          <SessionsList panel={peekPanel} full={false} overlay widthRem={peekWidths.sessions} />
        </>
      )}
    </>
  )
}
