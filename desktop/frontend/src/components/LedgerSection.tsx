import type { ReactNode } from 'react'

type LedgerSectionProps = {
  id: string
  title: string
  hint?: string
  children: ReactNode
}

export function LedgerSection({ id, title, hint, children }: LedgerSectionProps) {
  return (
    <section
      id={`ledger-${id}`}
      data-section={id}
      className="flex scroll-mt-14 flex-col gap-4 border-b border-line pb-8 last:border-b-0 @xl:scroll-mt-0"
    >
      <div className="flex items-baseline gap-2.5">
        <h2 className="text-title font-semibold">{title}</h2>
        {hint && <span className="text-small text-dim">{hint}</span>}
      </div>
      {children}
    </section>
  )
}
