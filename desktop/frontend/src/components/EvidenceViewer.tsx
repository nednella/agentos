import { useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { api } from '../api'
import { ago, useNow } from '../time'
import { useArmedConfirm } from '../useArmedConfirm'
import type { Evidence } from '../types'
import { ConfirmRow } from './ConfirmRow'
import { Icon } from './Icon'

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
  const at = items.findIndex((i) => i.id === id)
  const item = items[at]

  useEffect(() => root.current?.focus(), [])

  useEffect(() => {
    if (!item) onClose()
  }, [item, onClose])

  if (!item) return null

  const step = (delta: number) => {
    confirm.disarm()
    setId(items[(at + delta + items.length) % items.length].id)
  }

  const remove = () => {
    const next = items[at + 1] ?? items[at - 1]
    confirm.disarm()
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
      }}
    >
      <img src={item.url} alt={item.caption || 'Evidence image'} className="block max-h-[72vh] max-w-full rounded-md border border-line-strong object-contain" />
      <div className="flex max-w-3xl flex-col items-center gap-2 text-center">
        {item.caption && <p className="text-body">{item.caption}</p>}
        <p className="mono text-small text-dim">
          {at + 1} of {items.length} · {item.source === 'agent' ? 'agent' : 'you'} · {ago(item.at, now)}
        </p>
        <div className="flex flex-wrap items-center justify-center gap-2">
          <button className="btn" onClick={() => step(-1)} aria-label="Previous image" disabled={items.length < 2}>
            <Icon name="back" /> Previous
          </button>
          <button className="btn" onClick={() => step(1)} aria-label="Next image" disabled={items.length < 2}>
            Next <Icon name="forward" />
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
