import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { imageFiles, readImage } from '../images'
import { ago, useNow } from '../time'
import { useArmedConfirm } from '../useArmedConfirm'
import type { Note } from '../types'
import { AutoTextarea } from './AutoTextarea'
import { ConfirmRow } from './ConfirmRow'
import { Icon } from './Icon'
import { NoteImages } from './NoteImages'
import { NoteText } from './NoteText'

type NoteCardProps = {
  note: Note
  cursor: boolean
  editing: boolean
  onEdit(): void
  onStopEditing(refocus: boolean): void
  onOpenImage(url: string): void
}

type Confirm = 'file' | 'delete'

const FOLD_LINES = 10
const AUTOSAVE_MS = 700

export function NoteCard({ note, cursor, editing, onEdit, onStopEditing, onOpenImage }: NoteCardProps) {
  const a = useAgentos()
  const now = useNow()
  const confirm = useArmedConfirm<Confirm>()
  const [expanded, setExpanded] = useState(false)
  const [overflowing, setOverflowing] = useState(false)
  const body = useRef<HTMLDivElement>(null)
  const timer = useRef(0)
  const cancelled = useRef(false)
  const finishing = useRef(false)
  const title = note.text.split('\n')[0]
  const canFile = Boolean(a.project?.repo) && note.text.trim() !== ''

  useLayoutEffect(() => {
    const el = body.current
    if (el && !expanded) setOverflowing(el.scrollHeight > el.clientHeight + 1)
  }, [note.text, expanded, editing])

  const save = (text: string) => {
    if ((text.trim() || note.images.length > 0) && text.trim() !== note.text) a.report(() => a.updateNote(note.id, text))
  }

  useEffect(() => {
    if (editing) finishing.current = false
  }, [editing])

  const finish = (el: HTMLTextAreaElement, refocus: boolean) => {
    if (finishing.current) return
    finishing.current = true
    clearTimeout(timer.current)
    if (!cancelled.current) save(el.value)
    cancelled.current = false
    onStopEditing(refocus)
  }

  const attach = (files: File[]) => {
    if (files.length === 0) return false
    a.report(async () => {
      for (const image of await Promise.all(files.map(readImage))) await a.addNoteImage(note.id, image)
    })
    return true
  }

  if (editing) {
    return (
      <article className="note-card flex flex-col gap-2 p-3" data-cursor={cursor} onDragOver={(e) => e.preventDefault()} onDrop={(e) => attach(imageFiles(e.dataTransfer)) && e.preventDefault()}>
        <AutoTextarea
          autoFocus
          defaultValue={note.text}
          aria-label="Edit note"
          className="note-jot rounded-md border px-2.5 text-body"
          style={{ background: 'var(--bg-note)', borderColor: 'var(--border-note)' }}
          onFocus={(e) => e.currentTarget.setSelectionRange(note.text.length, note.text.length)}
          onPaste={(e) => attach(imageFiles(e.clipboardData)) && e.preventDefault()}
          onInput={(e) => {
            const el = e.currentTarget
            clearTimeout(timer.current)
            timer.current = window.setTimeout(() => save(el.value), AUTOSAVE_MS)
          }}
          onBlur={(e) => finish(e.currentTarget, false)}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              e.stopPropagation()
              cancelled.current = true
              finish(e.currentTarget, true)
            }
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              finish(e.currentTarget, true)
            }
          }}
        />
        <NoteImages urls={note.images} onOpen={onOpenImage} onRemove={(url) => a.report(() => a.removeNoteImage(note.id, url))} />
        <p className="text-label" style={{ color: 'var(--text-note-dim)' }}>
          Saves as you type. Enter finishes, Esc discards the pending change, paste an image to attach it.
        </p>
      </article>
    )
  }

  const iconButton = 'btn btn-ghost h-6 w-6 justify-center px-0'
  return (
    <article className="note-card group p-3" data-cursor={cursor} style={{ opacity: note.archived ? 0.6 : 1 }}>
      <div className="mb-1 flex min-h-6 items-center gap-2">
        <span className="mono text-label" style={{ color: 'var(--text-note-dim)' }}>
          {ago(note.createdAt, now)}
        </span>
        <span className="ml-auto flex items-center">
          <span className="reveal">
            <button className={iconButton} title="Start a session from this note" aria-label="Start session" onClick={() => a.report(() => a.noteToSession(note.id))}>
              <Icon name="play" size={12} />
            </button>
            {canFile && (
              <button className={iconButton} title="File as an issue" aria-label="File as issue" onClick={() => confirm.arm('file')}>
                <Icon name="issue" size={13} />
              </button>
            )}
            <button
              className={iconButton}
              title={note.archived ? 'Restore from the archive (E)' : 'Archive (E)'}
              aria-label={note.archived ? 'Restore note' : 'Archive note'}
              onClick={() => a.report(() => a.setNoteArchived(note.id, !note.archived))}
            >
              <Icon name="archive" size={13} />
            </button>
            <button className={iconButton} title="Delete" aria-label="Delete note" onClick={() => confirm.arm('delete')}>
              <Icon name="trash" size={13} />
            </button>
          </span>
          {!note.archived && (
            <button
              className={`${iconButton} -mr-1.5`}
              style={{ color: note.pinned ? 'var(--accent)' : 'var(--text-note-dim)' }}
              title={note.pinned ? 'Pinned. Click to unpin (P)' : 'Pin to the top (P)'}
              aria-label={note.pinned ? 'Unpin note' : 'Pin note'}
              aria-pressed={note.pinned}
              onClick={() => a.report(() => a.setNotePinned(note.id, !note.pinned))}
            >
              <Icon name="pin" size={13} />
            </button>
          )}
        </span>
      </div>
      <div
        role="button"
        tabIndex={0}
        aria-label="Edit note"
        title="Click to edit"
        className="cursor-text text-left"
        onClick={onEdit}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && e.target === e.currentTarget) {
            e.preventDefault()
            onEdit()
          }
        }}
      >
        <div
          ref={body}
          className="text-body leading-relaxed break-words whitespace-pre-wrap"
          style={expanded ? undefined : { display: '-webkit-box', WebkitLineClamp: FOLD_LINES, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}
          hidden={note.text === ''}
        >
          <div>
            <NoteText text={note.text} />
          </div>
        </div>
      </div>
      {(overflowing || expanded) && (
        <button className="mt-1 text-small" style={{ color: 'var(--text-note-dim)' }} onClick={() => setExpanded(!expanded)}>
          {expanded ? 'less' : 'more'}
        </button>
      )}
      {note.images.length > 0 && (
        <div className="mt-2">
          <NoteImages urls={note.images} onOpen={onOpenImage} />
        </div>
      )}
      {confirm.armed === 'file' && (
        <div className="mt-2.5 border-t pt-2.5" style={{ borderColor: 'var(--border-note)' }}>
          <ConfirmRow
            message={`File as issue “${title}”${note.images.length > 0 ? ' with its pictures' : ''}? The note is deleted afterwards.`}
            confirmLabel="File"
            onCancel={confirm.disarm}
            onConfirm={() => {
              confirm.disarm()
              a.report(() => a.noteToIssue(note.id))
            }}
          />
        </div>
      )}
      {confirm.armed === 'delete' && (
        <div className="mt-2.5 border-t pt-2.5" style={{ borderColor: 'var(--border-note)' }}>
          <ConfirmRow
            danger
            message="Delete this note?"
            confirmLabel="Delete"
            onCancel={confirm.disarm}
            onConfirm={() => {
              confirm.disarm()
              a.report(() => a.deleteNote(note.id))
            }}
          />
        </div>
      )}
    </article>
  )
}
