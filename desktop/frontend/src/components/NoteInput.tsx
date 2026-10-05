import { useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { imageFiles, readImage } from '../images'
import type { PreviewImage } from '../images'
import { AutoTextarea } from './AutoTextarea'
import { NoteImages } from './NoteImages'

export function NoteInput() {
  const { addNote, report, focusRequest, noteDraft, setNoteDraft } = useAgentos()
  const { returnToTerminal } = useLayout()
  const input = useRef<HTMLTextAreaElement>(null)
  const [images, setImages] = useState<PreviewImage[]>([])

  useEffect(() => {
    if (focusRequest.target === 'note-input') input.current?.focus()
  }, [focusRequest])

  const attach = (files: File[]) => {
    if (files.length === 0) return false
    report(async () => {
      const read = await Promise.all(files.map(readImage))
      setImages((list) => [...list, ...read])
    })
    return true
  }

  const submit = () => {
    if (!noteDraft.trim() && images.length === 0) return
    setNoteDraft('')
    setImages([])
    report(() => addNote(noteDraft, images))
  }

  return (
    <div className="group px-3 pt-3 pb-2" onDragOver={(e) => e.preventDefault()} onDrop={(e) => attach(imageFiles(e.dataTransfer)) && e.preventDefault()}>
      <div className="composer">
        <AutoTextarea
          taRef={input}
          value={noteDraft}
          onChange={(e) => setNoteDraft(e.target.value)}
          placeholder="Jot something down…"
          aria-label="New note"
          className="px-3 pt-2.5 pb-1 text-body"
          onPaste={(e) => attach(imageFiles(e.clipboardData)) && e.preventDefault()}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              e.stopPropagation()
              returnToTerminal()
            }
            if (e.key !== 'Enter' || e.shiftKey) return
            e.preventDefault()
            submit()
          }}
        />
        {images.length > 0 && (
          <div className="px-3 pb-1">
            <NoteImages urls={images.map((i) => i.preview)} onRemove={(url) => setImages((list) => list.filter((i) => i.preview !== url))} />
          </div>
        )}
        <div className="composer-bar">
          <span className="text-label opacity-0 transition-opacity group-focus-within:opacity-100" style={{ color: 'var(--text-note-dim)' }}>
            Enter adds · Shift+Enter new line · paste an image to attach
          </span>
          <button className="btn btn-accent ml-auto h-6 px-2 text-label" disabled={!noteDraft.trim() && images.length === 0} onMouseDown={(e) => e.preventDefault()} onClick={submit}>
            Add
          </button>
        </div>
      </div>
    </div>
  )
}
