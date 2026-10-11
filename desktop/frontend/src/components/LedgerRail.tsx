type LedgerRailProps = {
  sections: { id: string; title: string }[]
  active: string
  onPick(id: string): void
}

export function LedgerRail({ sections, active, onPick }: LedgerRailProps) {
  return (
    <nav
      aria-label="Sections"
      className="sticky -top-4 z-10 -mx-4 -mt-4 flex gap-1 overflow-x-auto border-b border-line bg-app px-1.5 py-2 @xl:top-0 @xl:my-0 @xl:mr-0 @xl:-ml-3 @xl:bg-transparent @xl:flex-col @xl:self-start @xl:border-0 @xl:p-0"
    >
      {sections.map((s) => (
        <button
          key={s.id}
          aria-current={active === s.id}
          className="flex-none rounded-sm px-2.5 py-1.5 text-left text-small font-medium whitespace-nowrap @xl:border-l-2"
          style={{
            background: active === s.id ? 'var(--bg-raised)' : 'transparent',
            color: active === s.id ? 'var(--text)' : 'var(--text-soft)',
            borderLeftColor: active === s.id ? 'var(--accent)' : 'transparent',
          }}
          onClick={() => onPick(s.id)}
        >
          {s.title}
        </button>
      ))}
    </nav>
  )
}
