import type { ReactNode } from 'react'

export type Headline = { id: string; title: string; unseen: boolean }

type NewsSourceRowProps = {
  name: string
  sub: string
  error?: string
  unseen: number
  headlines: Headline[]
  empty: ReactNode
  onOpen(): void
}

export function NewsSourceRow({ name, sub, error, unseen, headlines, empty, onOpen }: NewsSourceRowProps) {
  return (
    <button className="flex w-full flex-col gap-2.5 px-3 py-5 text-left hover:bg-hover focus-visible:bg-hover" onClick={onOpen}>
      <span className="flex w-full items-baseline gap-4">
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="text-body font-semibold">{name}</span>
          <span className="text-small text-dim">{sub}</span>
          {error && <span className="text-small text-danger">{error}</span>}
        </span>
        {unseen > 0 && <span className="flex-none text-small text-accent">{unseen} new</span>}
      </span>
      {headlines.length === 0 ? (
        <span className="text-small text-dim">{empty}</span>
      ) : (
        <span className="flex w-full flex-col gap-1">
          {headlines.map((h) => (
            <span key={h.id} className="flex items-center gap-2.5 text-small text-soft">
              <span className="dot" style={{ ['--c' as string]: h.unseen ? 'var(--accent)' : 'var(--border-strong)' }} />
              <span className="min-w-0 flex-1 truncate">{h.title}</span>
            </span>
          ))}
        </span>
      )}
    </button>
  )
}
