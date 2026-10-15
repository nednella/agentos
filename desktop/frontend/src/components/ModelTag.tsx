type ModelTagProps = { model: string; effort: string; spend?: string; className?: string }

export function ModelTag({ model, effort, spend, className = '' }: ModelTagProps) {
  const text = [model, effort, spend].filter(Boolean).join(' · ')
  if (!text) return null
  return (
    <span
      className={`mono flex-none text-small whitespace-nowrap text-dim ${className}`}
      title={spend ? 'Model, effort, and tokens and cost so far, estimated at API list prices' : 'Model and effort'}
    >
      {text}
    </span>
  )
}
