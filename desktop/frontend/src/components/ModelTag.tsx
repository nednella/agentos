import { shortModel } from '../model'

type ModelTagProps = { agent?: string; model: string; effort: string; spend?: string; problem?: string; className?: string }

export function ModelTag({ agent = '', model, effort, spend, problem = '', className = '' }: ModelTagProps) {
  const text = [agent, shortModel(model), effort, spend].filter(Boolean).join(' · ')
  if (!text) return null
  const title = spend ? 'Agent, model, effort, and tokens and cost so far, estimated at API list prices' : 'Agent, model and effort'
  return (
    <span className={`mono flex-none text-small whitespace-nowrap ${problem ? 'text-danger' : 'text-dim'} ${className}`} title={problem || title}>
      {text}
    </span>
  )
}
