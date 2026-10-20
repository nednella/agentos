type ModelTagProps = { agent?: string; model: string; effort: string; spend?: string; className?: string }

// agent is the session's provider when it is not the default, else ''.
export function ModelTag({ agent = '', model, effort, spend, className = '' }: ModelTagProps) {
  const text = [agent, model, effort, spend].filter(Boolean).join(' · ')
  if (!text) return null
  const what = agent ? 'Agent, model, effort' : 'Model, effort'
  return (
    <span
      className={`mono flex-none text-small whitespace-nowrap text-dim ${className}`}
      title={spend ? `${what}, and tokens and cost so far, estimated at API list prices` : what.replace(', effort', ' and effort')}
    >
      {text}
    </span>
  )
}
