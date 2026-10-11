type LoadingProps = { label: string }

export function Loading({ label }: LoadingProps) {
  return (
    <div role="status" className="flex h-full flex-1 flex-col items-center justify-center gap-3 px-4 py-16 text-small text-dim">
      <span className="spinner spinner-lg" aria-hidden="true" />
      {label}
    </div>
  )
}
