type SegmentedControlProps<T extends string> = {
  label: string
  options: { value: T; label: string }[]
  value: T
  onChange(value: T): void
  dense?: boolean
}

export function SegmentedControl<T extends string>({ label, options, value, onChange, dense }: SegmentedControlProps<T>) {
  return (
    <div role="group" aria-label={label} className="flex flex-none overflow-hidden rounded-md border border-line-strong">
      {options.map((option) => (
        <button
          key={option.value}
          aria-pressed={value === option.value}
          className={`h-7 text-small font-medium ${dense ? 'px-2' : 'px-3'}`}
          style={{ background: value === option.value ? 'var(--bg-hover)' : 'transparent', color: value === option.value ? 'var(--text)' : 'var(--text-soft)' }}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}
