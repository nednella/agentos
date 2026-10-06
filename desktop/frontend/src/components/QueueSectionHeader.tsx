import { Icon } from './Icon'

type QueueSectionHeaderProps = { name: string; count: number; open: boolean; cursor: boolean; locked: boolean; onToggle(): void }

export function QueueSectionHeader({ name, count, open, cursor, locked, onToggle }: QueueSectionHeaderProps) {
  return (
    <button className="row row-inset h-9 items-center gap-2 px-3" aria-expanded={open} data-cursor={cursor} disabled={locked} onClick={onToggle}>
      <span className="text-dim transition-transform duration-150" style={{ transform: open ? 'none' : 'rotate(-90deg)' }}>
        <Icon name="chevron" size={12} />
      </span>
      <span className="label">{name}</span>
      <span className="mono ml-auto text-small text-dim">{count}</span>
    </button>
  )
}
