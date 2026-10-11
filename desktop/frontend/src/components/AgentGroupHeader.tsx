type AgentGroupHeaderProps = { label: string; count: number }

export function AgentGroupHeader({ label, count }: AgentGroupHeaderProps) {
  return (
    <div className="flex h-8 items-center gap-2 bg-raised px-4">
      <span className="label">{label}</span>
      <span className="mono text-small text-dim">{count}</span>
    </div>
  )
}
