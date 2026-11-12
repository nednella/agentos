import { Icon } from './Icon'

export const tabRowButton = 'btn btn-ghost h-[1.625rem] w-[1.625rem] flex-none justify-center px-0 text-dim hover:text-ink'

type ShellTabProps = {
  label: string
  selected: boolean
  pinned?: boolean
  onSelect(): void
  onClose?(): void
}

export function ShellTab({ label, selected, pinned = false, onSelect, onClose }: ShellTabProps) {
  return (
    <div
      className={`group flex h-full flex-none scroll-mx-6 items-center border-r ${pinned ? 'border-line-strong' : 'border-line'} ${selected ? 'bg-term text-ink' : 'text-soft hover:bg-hover'}`}
      style={selected ? { boxShadow: 'inset 0 2px 0 var(--accent)' } : undefined}
      data-selected={selected}
    >
      <button
        role="tab"
        aria-selected={selected}
        className={`flex h-full items-center gap-2 whitespace-nowrap pl-[1.125rem] ${onClose ? 'pr-1' : 'pr-[1.125rem]'}`}
        onClick={onSelect}
      >
        {pinned && <Icon name="folder" size={12} />}
        {label}
      </button>
      {onClose && (
        <button
          className={`${tabRowButton} mr-1.5 focus-visible:opacity-100 group-hover:opacity-100 ${selected ? '' : 'opacity-0'}`}
          title="Close shell"
          aria-label={`Close ${label}`}
          onClick={onClose}
        >
          <Icon name="cross" size={14} />
        </button>
      )}
    </div>
  )
}
