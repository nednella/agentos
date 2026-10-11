import { stateLabel } from '../stateMeta'
import type { State } from '../types'
import { StateDot } from './StateDot'

type StateBadgeProps = { state: State; chat: boolean }

export function StateBadge({ state, chat }: StateBadgeProps) {
  return (
    <span
      data-state={state}
      className="inline-flex h-6 flex-none items-center gap-1.5 rounded-sm border px-2 text-small font-medium"
      style={{ color: 'var(--c)', borderColor: 'color-mix(in srgb, var(--c) 45%, transparent)' }}
    >
      <StateDot state={state} />
      {stateLabel(state, chat)}
    </span>
  )
}
