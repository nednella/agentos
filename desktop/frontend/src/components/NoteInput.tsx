import { useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { imageFiles, readImage } from '../images'
import type { PreviewImage } from '../images'
import { AutoTextarea, growTextarea } from './AutoTextarea'
import { NoteImages } from './NoteImages'

export function NoteInput() {
  const { addNote, report, focusRequest } = useAgentos()
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
    const el = input.current
    if (!el) return
    const text = el.value
    if (!text.trim() && images.length === 0) return
    const attached = images
    el.value = ''
    growTextarea(el)
    setImages([])
    report(() => addNote(text, attached))
  }

  return (
    <div className="group flex flex-col gap-1.5 px-3 pt-3 pb-2" onDragOver={(e) => e.preventDefault()} onDrop={(e) => attach(imageFiles(e.dataTransfer)) && e.preventDefault()}>
      <AutoTextarea
        taRef={input}
        placeholder="Jot something down…"
        aria-label="New note"
        className="note-jot text-body"
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
      <p className="px-1 text-label opacity-0 transition-opacity group-focus-within:opacity-100" style={{ color: 'var(--text-note-dim)' }}>
        Enter adds · Shift+Enter new line
      </p>
      <NoteImages urls={images.map((i) => i.preview)} onRemove={(url) => setImages((list) => list.filter((i) => i.preview !== url))} />
    </div>
  )
}
