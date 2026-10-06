type ModelTagProps = { model: string; effort: string }

export function ModelTag({ model, effort }: ModelTagProps) {
  const text = [model, effort].filter(Boolean).join(' · ')
  if (!text) return null
  return (
    <span className="mono flex-none text-small whitespace-nowrap text-dim" title="Model and effort">
      {text}
    </span>
  )
}
