import type { ReactNode } from 'react'
import { Icon } from './Icon'

type SectionHeaderProps = { name: string; count: number; open: boolean; cursor: boolean; locked?: boolean; summary?: ReactNode; onToggle(): void }

export function SectionHeader({ name, count, open, cursor, locked = false, summary, onToggle }: SectionHeaderProps) {
  return (
    <button className="section-head" aria-expanded={open} data-cursor={cursor} disabled={locked} onClick={onToggle}>
      <span className="label">{name}</span>
      <span className="mono text-small text-dim">{count}</span>
      <span className="ml-auto flex items-center gap-2.5">
        {summary}
        <span data-chevron className="transition-transform duration-150" style={{ transform: open ? 'none' : 'rotate(-90deg)' }}>
          <Icon name="chevron" size={12} />
        </span>
      </span>
    </button>
  )
}
