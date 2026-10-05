import { ago, useNow } from '../time'
import type { Evidence } from '../types'
import { Icon } from './Icon'

type EvidenceCardProps = {
  item: Evidence
  selectable: boolean
  selected: boolean
  onOpen(): void
  onToggle(): void
}

export function EvidenceCard({ item, selectable, selected, onOpen, onToggle }: EvidenceCardProps) {
  const now = useNow()
  const image = item.kind === 'image'

  return (
    <article
      className="flex min-w-0 flex-col overflow-hidden rounded-md border bg-surface"
      style={{ borderColor: selected ? 'var(--accent)' : 'var(--border)' }}
    >
      {image ? (
        <button
          className="relative block aspect-[16/10] w-full bg-term"
          title={selectable ? 'Select for comparison' : 'Open larger'}
          aria-pressed={selectable ? selected : undefined}
          onClick={selectable ? onToggle : onOpen}
        >
          <img src={item.url} alt={item.caption || 'Evidence image'} className="h-full w-full object-cover" />
          {selectable && (
            <span
              className="absolute top-2 right-2 grid h-5 w-5 place-items-center rounded-sm border"
              style={{ background: selected ? 'var(--accent)' : 'var(--bg-app)', borderColor: selected ? 'var(--accent)' : 'var(--border-strong)', color: 'var(--bg-app)' }}
            >
              {selected && <Icon name="check" size={12} />}
            </span>
          )}
        </button>
      ) : (
        <div className="max-h-48 overflow-y-auto px-3 pt-3 text-body break-words whitespace-pre-wrap">{item.text}</div>
      )}
      <div className="flex flex-1 flex-col gap-1.5 px-3 py-2.5">
        {item.caption && <p className="text-small text-soft">{item.caption}</p>}
        <p className="mt-auto flex items-center gap-2 text-label text-dim">
          <span
            className="rounded-sm border px-1.5"
            style={{ color: item.source === 'agent' ? 'var(--accent)' : 'var(--text-soft)', borderColor: item.source === 'agent' ? 'var(--accent)' : 'var(--border-strong)' }}
          >
            {item.source === 'agent' ? 'agent' : 'you'}
          </span>
          <span className="mono">{ago(item.at, now)}</span>
        </p>
      </div>
    </article>
  )
}
