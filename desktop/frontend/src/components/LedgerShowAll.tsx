type LedgerShowAllProps = { count: number; limit: number; onShow(): void }

export function LedgerShowAll({ count, limit, onShow }: LedgerShowAllProps) {
  if (count <= limit) return null
  return (
    <button className="btn btn-ghost h-7 self-start px-2" onClick={onShow}>
      Show all {count.toLocaleString()}
    </button>
  )
}
