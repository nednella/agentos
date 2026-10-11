import { Icon } from './Icon'

type SessionGroupEmptyProps = { icon: 'chat' | 'branch'; title: string; hint: string }

export function SessionGroupEmpty({ icon, title, hint }: SessionGroupEmptyProps) {
  return (
    <div className="flex items-start gap-3 px-4 py-4 text-dim">
      <span className="mt-0.5 flex-none">
        <Icon name={icon} size={16} />
      </span>
      <div className="flex min-w-0 flex-col gap-1">
        <span className="text-small text-soft">{title}</span>
        <span className="text-small">{hint}</span>
      </div>
    </div>
  )
}
