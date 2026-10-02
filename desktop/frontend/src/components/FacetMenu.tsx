import { useEffect, useRef, useState } from 'react'
import type { Facet } from '../issueFilter'
import { Icon } from './Icon'

type FacetMenuProps = {
  label: string
  facets: Facet[]
  active: string | null
  onPick(value: string | null): void
}

export function FacetMenu({ label, facets, active, onPick }: FacetMenuProps) {
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false)
    }
    window.addEventListener('mousedown', close)
    return () => window.removeEventListener('mousedown', close)
  }, [open])

  return (
    <div ref={root} className="relative">
      <button className="btn btn-ghost h-6 gap-1 px-2" aria-expanded={open} onClick={() => setOpen(!open)}>
        <span style={{ color: active ? 'var(--accent)' : undefined }}>{active ? `${label}: ${active}` : label}</span>
        <Icon name="chevron" size={11} />
      </button>
      {open && (
        <div
          className="fade-in absolute top-7 left-0 max-h-64 min-w-40 overflow-y-auto rounded-md border border-line-strong bg-raised py-1"
          style={{ zIndex: 'var(--z-popup)' }}
          role="menu"
        >
          {active && (
            <button
              className="row h-7 items-center px-3 text-soft"
              onClick={() => {
                setOpen(false)
                onPick(null)
              }}
            >
              Any {label.toLowerCase()}
            </button>
          )}
          {facets.map((f) => (
            <button
              key={f.value}
              role="menuitem"
              className="row h-7 items-center gap-3 px-3"
              data-selected={f.value === active}
              onClick={() => {
                setOpen(false)
                onPick(f.value)
              }}
            >
              <span className="truncate">{f.value}</span>
              <span className="mono ml-auto text-small text-dim">{f.count}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
