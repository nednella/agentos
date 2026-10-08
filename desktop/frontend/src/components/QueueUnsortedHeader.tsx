import { Icon } from './Icon'

type QueueUnsortedHeaderProps = { count: number; open: boolean; cursor: boolean; locked: boolean; onToggle(): void }

export function QueueUnsortedHeader({ count, open, cursor, locked, onToggle }: QueueUnsortedHeaderProps) {
  return (
    <button
      className="row row-inset h-9 items-center gap-2 px-3 text-small text-dim"
      aria-expanded={open}
      data-cursor={cursor}
      disabled={locked}
      onClick={onToggle}
    >
      <span className="transition-transform duration-150" style={{ transform: open ? 'none' : 'rotate(-90deg)' }}>
        <Icon name="chevron" size={12} />
      </span>
      <span>{`${count} ${open ? '' : 'more '}${count === 1 ? 'issue matches' : 'issues match'} no section`}</span>
      <span className="ml-auto">{open ? 'Hide' : 'Show'}</span>
    </button>
  )
}
