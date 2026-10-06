type SegmentedControlProps<T extends string> = {
  label: string
  options: { value: T; label: string }[]
  value: T
  onChange(value: T): void
}

export function SegmentedControl<T extends string>({ label, options, value, onChange }: SegmentedControlProps<T>) {
  return (
    <div role="group" aria-label={label} className="flex flex-none overflow-hidden rounded-md border border-line-strong">
      {options.map((option) => (
        <button
          key={option.value}
          aria-pressed={value === option.value}
          className="h-7 px-3 text-small font-medium"
          style={{ background: value === option.value ? 'var(--bg-hover)' : 'transparent', color: value === option.value ? 'var(--text)' : 'var(--text-soft)' }}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}
