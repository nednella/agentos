type ActionTagProps = { action: string }

export function ActionTag({ action }: ActionTagProps) {
  if (!action) return null
  return (
    <span
      className="flex-none rounded-sm border border-line-strong px-1.5 text-label whitespace-nowrap text-soft"
      title="The queue action this session was started with"
    >
      {action}
    </span>
  )
}
