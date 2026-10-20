type ModelTagProps = { agent?: string; model: string; effort: string; spend?: string; className?: string }

export function ModelTag({ agent = '', model, effort, spend, className = '' }: ModelTagProps) {
  const text = [agent, model, effort, spend].filter(Boolean).join(' · ')
  if (!text) return null
  return (
    <span
      className={`mono flex-none text-small whitespace-nowrap text-dim ${className}`}
      title={spend ? 'Agent, model, effort, and tokens and cost so far, estimated at API list prices' : 'Agent, model and effort'}
    >
      {text}
    </span>
  )
}
