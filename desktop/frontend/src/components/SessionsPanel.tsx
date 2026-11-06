import { useRef, useState } from 'react'
import type { KeyboardEvent, RefObject } from 'react'
import { useAgentos } from '../AgentosContext'
import { readStored, writeStored } from '../storage'
import type { Session } from '../types'
import { useFocusRequest } from '../useFocusRequest'
import { useLayout } from '../LayoutContext'
import { useListNav } from '../useListNav'
import { useScrollCursorIntoView } from '../useScrollCursorIntoView'
import { useToggleAnimation } from '../useToggleAnimation'
import { SessionGroupEmpty } from './SessionGroupEmpty'
import { SessionGroupSummary } from './SessionGroupSummary'
import { CleanupsMenu } from './CleanupsMenu'
import { Collapse } from './Collapse'
import { EdgeStrip } from './EdgeStrip'
import { Icon } from './Icon'
import { NewSessionRow } from './NewSessionRow'
import { SectionHeader } from './SectionHeader'
import { SessionRow } from './SessionRow'

const COMPACT_BELOW_REM = 17
const NEW_SESSION_KEY = 'new-session'
const CLOSED_KEY = 'agentos.sessions.closed'

type GroupKey = 'chats' | 'tasks'
type Group = { key: GroupKey; name: string; sessions: Session[]; open: boolean }
type Item = { kind: 'group'; group: Group } | { kind: 'session'; session: Session }

const itemKey = (item: Item) => (item.kind === 'group' ? `group:${item.group.key}` : item.session.id)

type SessionsListProps = { panel: RefObject<HTMLElement>; full: boolean; overlay: boolean; widthRem: number }

function SessionsList({ panel, full, overlay, widthRem }: SessionsListProps) {
  const { sessions, selectedId, select, focus, setComposing } = useAgentos()
  const { mode, setSessionsOpen, returnToTerminal } = useLayout()
  const list = useRef<HTMLDivElement>(null)
  const [closed, setClosed] = useState<Set<GroupKey>>(() => new Set(readStored<GroupKey[]>(CLOSED_KEY, [])))

  const groups: Group[] = [
    { key: 'chats', name: 'Chats', sessions: sessions.filter((s) => s.chat), open: !closed.has('chats') },
    { key: 'tasks', name: 'Tasks', sessions: sessions.filter((s) => !s.chat), open: !closed.has('tasks') },
  ]
  const items: Item[] = groups.flatMap((group) => [
    { kind: 'group' as const, group },
    ...(group.open ? group.sessions.map((session) => ({ kind: 'session' as const, session })) : []),
  ])

  const toggle = (key: GroupKey, open: boolean) => {
    setClosed((set) => {
      const next = new Set(set)
      if (open) next.delete(key)
      else next.add(key)
      writeStored(CLOSED_KEY, [...next])
      return next
    })
  }

  const groupAt = (index: number) => {
    const item = items[index]
    return item?.kind === 'group' ? item.group : null
  }

  const nav = useListNav([...items.map(itemKey), NEW_SESSION_KEY], {
    onEnter(index) {
      const item = items[index]
      if (!item) {
        setComposing('session')
        return
      }
      if (item.kind === 'group') {
        toggle(item.group.key, !item.group.open)
        return
      }
      select(item.session.id)
      focus('terminal')
    },
    onSpace(index) {
      const group = groupAt(index)
      if (!group) return false
      toggle(group.key, !group.open)
      return true
    },
    onLeft(index) {
      const group = groupAt(index)
      if (!group?.open) return false
      toggle(group.key, false)
      return true
    },
    onRight(index) {
      const group = groupAt(index)
      if (!group || group.open) return false
      toggle(group.key, true)
      return true
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
            title={overlay ? 'Close (Esc)' : 'Collapse sessions (⌘D)'}
            aria-label={overlay ? 'Close sessions' : 'Collapse sessions'}
            onClick={() => setSessionsOpen(false)}
          >
            <Icon name={overlay ? 'close' : 'panel-right'} size={15} />
          </button>
        )}
      </div>
      <div ref={list} className="min-h-0 flex-1 overflow-y-auto">
        {groups.map((group) => (
          <section key={group.key}>
            <SectionHeader
              name={group.name}
              count={group.sessions.length}
              open={group.open}
              cursor={nav.cursorKey === `group:${group.key}`}
              summary={<SessionGroupSummary sessions={group.sessions} open={group.open} />}
              onToggle={() => {
                nav.setCursorKey(`group:${group.key}`)
                toggle(group.key, !group.open)
              }}
            />
            <Collapse open={group.open}>
              <div className="flex flex-col gap-1.5 py-1.5">
                {group.sessions.map((s) => (
                  <SessionRow key={s.id} session={s} selected={s.id === selectedId} cursor={nav.cursorKey === s.id} compact={compact} dense={dense} />
                ))}
                {group.sessions.length === 0 &&
                  (group.key === 'chats' ? (
                    <SessionGroupEmpty icon="chat" title="No chats yet" hint="Ask about the project without changing it." />
                  ) : (
                    <SessionGroupEmpty icon="branch" title="No tasks yet" hint="Work on an issue or an idea, in its own branch." />
                  ))}
              </div>
            </Collapse>
          </section>
        ))}
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
