import type { Session } from '../types'
import { Icon } from './Icon'
import { StateDot } from './StateDot'

type SessionMarkProps = { session: Session }

export function SessionMark({ session }: SessionMarkProps) {
  return (
    <span className="flex flex-none items-center gap-1.5 text-dim">
      <Icon name={session.chat ? 'chat' : 'branch'} size={12} />
      <StateDot state={session.state} />
    </span>
  )
}
