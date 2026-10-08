import { useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import { ago, useNow } from '../time'
import { useArmedConfirm } from '../useArmedConfirm'
import type { Evidence } from '../types'
import { ConfirmRow } from './ConfirmRow'
import { Icon } from './Icon'
import { FIT, ZOOM_STEP, ZoomableImage, zoomTo, type ImageView } from './ZoomableImage'

type EvidenceViewerProps = {
  sessionId: string
  items: Evidence[]
  startId: string
  onClose(): void
}

export function EvidenceViewer({ sessionId, items, startId, onClose }: EvidenceViewerProps) {
  const { report } = useAgentos()
  const now = useNow()
  const root = useRef<HTMLDivElement>(null)
  const [id, setId] = useState(startId)
  const confirm = useArmedConfirm()
  const [copied, setCopied] = useState(false)
  const [view, setView] = useState<ImageView>(FIT)
  const at = items.findIndex((i) => i.id === id)
  const item = items[at]

  useEffect(() => root.current?.focus(), [])

  useEffect(() => {
    if (!item) onClose()
  }, [item, onClose])

  if (!item) return null

  const step = (delta: number) => {
    confirm.disarm()
    setView(FIT)
    setId(items[(at + delta + items.length) % items.length].id)
  }

  const remove = () => {
    const next = items[at + 1] ?? items[at - 1]
    confirm.disarm()
    setView(FIT)
    if (next) setId(next.id)
    report(() => api.deleteEvidence(sessionId, item.id))
  }

  return (
    <div
      ref={root}
      tabIndex={-1}
      role="dialog"
      aria-label="Evidence"
      className="fade-in fixed inset-0 flex flex-col items-center justify-center gap-3 p-4 outline-none"
      style={{ zIndex: 'var(--z-overlay)', background: 'var(--bg-overlay-strong)' }}
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
      onKeyDown={(e) => {
        if (e.key === 'Escape') onClose()
        if (e.key === 'ArrowLeft') step(-1)
        if (e.key === 'ArrowRight') step(1)
        if (e.metaKey) return
        if (e.key === '=' || e.key === '+') setView((v) => zoomTo(v, v.scale * ZOOM_STEP))
        if (e.key === '-') setView((v) => zoomTo(v, v.scale / ZOOM_STEP))
        if (e.key === '0') setView(FIT)
      }}
    >
      <ZoomableImage src={item.url} alt={item.caption || 'Evidence image'} view={view} onView={setView} onDismiss={onClose} />
      <div className="flex max-w-3xl flex-col items-center gap-2 text-center">
        {item.caption && <p className="text-body">{item.caption}</p>}
        <p className="mono text-small text-dim">
          {at + 1} of {items.length} · {item.source === 'agent' ? 'agent' : 'you'} · {ago(item.at, now)} · {Math.round(view.scale * 100)}%
        </p>
        <div className="flex flex-wrap items-center justify-center gap-2">
          <button className="btn" onClick={() => step(-1)} aria-label="Previous image" disabled={items.length < 2}>
            <Icon name="back" /> Previous
          </button>
          <button className="btn" onClick={() => step(1)} aria-label="Next image" disabled={items.length < 2}>
            Next <Icon name="forward" />
          </button>
          <button className="btn" onClick={() => setView((v) => zoomTo(v, v.scale / ZOOM_STEP))} aria-label="Zoom out" disabled={view.scale === 1}>
            −
          </button>
          <button className="btn" onClick={() => setView(FIT)} aria-label="Fit image" disabled={view.scale === 1}>
            Fit
          </button>
          <button className="btn" onClick={() => setView((v) => zoomTo(v, v.scale * ZOOM_STEP))} aria-label="Zoom in">
            +
          </button>
          <button
            className="btn"
            onClick={() => {
              report(() => navigator.clipboard.writeText(item.url))
              setCopied(true)
              setTimeout(() => setCopied(false), 1500)
            }}
          >
            {copied ? 'Copied' : 'Copy path/URL'}
          </button>
          {confirm.armed ? (
            <ConfirmRow inline danger message="Delete this item?" confirmLabel="Delete" cancelLabel="No" onCancel={confirm.disarm} onConfirm={remove} />
          ) : (
            <button className="btn btn-ghost" onClick={() => confirm.arm()}>
              <Icon name="trash" /> Delete
            </button>
          )}
          <button className="btn btn-ghost" onClick={onClose} aria-label="Close">
            <Icon name="close" /> Close
          </button>
        </div>
      </div>
    </div>
  )
}
