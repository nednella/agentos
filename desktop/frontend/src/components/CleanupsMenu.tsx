import { useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { ago, useNow } from '../time'
import { Icon } from './Icon'

export function CleanupsMenu() {
  const { cleanups } = useAgentos()
  const now = useNow()
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLDivElement>(null)
  const [anchor, setAnchor] = useState({ top: 0, right: 0 })

  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false)
    }
    const escape = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      e.stopPropagation()
      setOpen(false)
    }
    window.addEventListener('mousedown', close)
    window.addEventListener('keydown', escape, true)
    return () => {
      window.removeEventListener('mousedown', close)
      window.removeEventListener('keydown', escape, true)
    }
  }, [open])

  return (
    <div ref={root} className="relative">
      <button
        className="btn btn-ghost h-7 gap-1 px-2"
        aria-expanded={open}
        title="Cleaned up sessions"
        aria-label={`Cleaned up sessions, ${cleanups.length}`}
        onClick={(e) => {
          const rect = e.currentTarget.getBoundingClientRect()
          setAnchor({ top: rect.bottom + 4, right: window.innerWidth - rect.right })
          setOpen(!open)
        }}
      >
        <Icon name="broom" size={13} />
        <span className="mono text-small">{cleanups.length}</span>
      </button>
      {open && (
        <div
          className="fade-in fixed max-h-96 w-[min(24rem,92vw)] overflow-y-auto rounded-md border border-line-strong bg-raised shadow-2xl"
          style={{ zIndex: 'var(--z-popup)', top: anchor.top, right: Math.max(anchor.right, 8) }}
          role="dialog"
          aria-label="Cleaned up sessions"
        >
          <div className="label border-b border-line px-4 py-2">Cleaned up</div>
          {cleanups.length === 0 && <p className="px-4 py-4 text-small text-dim">Nothing cleaned up yet.</p>}
          {cleanups.map((c) => (
            <div key={`${c.at}-${c.sessionTitle}`} className="border-b border-line px-4 py-2.5 last:border-b-0">
              <div className="flex items-baseline gap-2">
                <span className="min-w-0 flex-1 truncate text-body font-medium">{c.sessionTitle}</span>
                {c.pr > 0 && <span className="mono text-small text-dim">PR #{c.pr}</span>}
                <span className="mono text-label text-dim">{ago(c.at, now)}</span>
              </div>
              {c.status === 'blocked' ? (
                <p className="text-small text-danger">Blocked: {c.reason}</p>
              ) : (
                <p className="text-small text-dim">Removed {c.removed.join(', ')}</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
