import { useEffect, useMemo, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useListNav, isTyping } from '../useListNav'
import { useScrollCursorIntoView } from '../useScrollCursorIntoView'
import { Collapse } from './Collapse'
import { Icon } from './Icon'
import { ImageViewer } from './ImageViewer'
import { NoteCard } from './NoteCard'
import { NoteInput } from './NoteInput'
import { Notice } from './Notice'
import type { NavRef } from './Sidebar'

type NotesProps = { nav: NavRef }

export function Notes({ nav }: NotesProps) {
  const a = useAgentos()
  const [query, setQuery] = useState('')
  const [archivedOpen, setArchivedOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [viewing, setViewing] = useState<string | null>(null)
  const list = useRef<HTMLDivElement>(null)

  const { active, archived } = useMemo(() => {
    const matches = a.notes.filter((n) => n.text.toLowerCase().includes(query.trim().toLowerCase()))
    return {
      active: matches.filter((n) => !n.archived).sort((x, y) => Number(y.pinned) - Number(x.pinned)),
      archived: matches.filter((n) => n.archived),
    }
  }, [a.notes, query])

  const visible = archivedOpen || query ? [...active, ...archived] : active
  const showArchived = archivedOpen || Boolean(query)

  const listNav = useListNav(visible.map((n) => n.id), {
    onEnter(index) {
      if (visible[index]) setEditingId(visible[index].id)
    },
  })

  useEffect(() => {
    nav.current = (e) => {
      if (listNav.handle(e)) return true
      const note = visible[listNav.cursor]
      if (!note || isTyping(e.target) || e.metaKey || e.ctrlKey || e.altKey) return false
      if (e.key === 'p' && !note.archived) a.report(() => a.setNotePinned(note.id, !note.pinned))
      else if (e.key === 'e') a.report(() => a.setNoteArchived(note.id, !note.archived))
      else return false
      return true
    }
    return () => {
      nav.current = null
    }
  })

  useScrollCursorIntoView(list, listNav.cursorKey)

  const card = (note: (typeof visible)[number]) => (
    <NoteCard
      key={note.id}
      note={note}
      cursor={listNav.cursorKey === note.id}
      editing={editingId === note.id}
      onEdit={() => {
        listNav.setCursorKey(note.id)
        setEditingId(note.id)
      }}
      onStopEditing={(refocus) => {
        setEditingId(null)
        if (refocus) a.focus('sidebar')
      }}
      onOpenImage={setViewing}
    />
  )

  return (
    <>
      <NoteInput />
      {a.notes.length > 0 && (
        <div className="relative px-3 pb-2">
          <span className="pointer-events-none absolute top-1.5 left-5 text-dim">
            <Icon name="search" size={13} />
          </span>
          <input
            value={query}
            placeholder="Search notes"
            aria-label="Search notes"
            spellCheck={false}
            className="field note-search w-full pl-7 text-small"
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key !== 'Escape') return
              e.stopPropagation()
              if (query) setQuery('')
              else (e.target as HTMLElement).blur()
            }}
          />
        </div>
      )}
      <div ref={list} className="flex min-h-0 flex-1 flex-col gap-2.5 overflow-y-auto px-3 pb-3">
        {a.notes.length === 0 && (
          <Notice title="Nothing jotted yet" hint="Ramble here: half ideas, things to ask, bugs you noticed. Start a session or file an issue from any note later." centered />
        )}
        {a.notes.length > 0 && visible.length === 0 && <Notice title="No notes match" hint="Clear the search to see them all." centered />}
        {active.map(card)}
        {archived.length > 0 && (
          <>
            <button className="flex items-center gap-1.5 py-1 text-small" style={{ color: 'var(--text-note-dim)' }} aria-expanded={showArchived} onClick={() => setArchivedOpen(!archivedOpen)}>
              <span style={{ transform: showArchived ? 'none' : 'rotate(-90deg)' }}>
                <Icon name="chevron" size={12} />
              </span>
              Archived ({archived.length})
            </button>
            <Collapse open={showArchived}>
              <div className="flex flex-col gap-2.5">{archived.map(card)}</div>
            </Collapse>
          </>
        )}
      </div>
      {viewing && <ImageViewer url={viewing} onClose={() => setViewing(null)} />}
    </>
  )
}
