import type { State } from '../types'

type StateDotProps = { state: State }

export function StateDot({ state }: StateDotProps) {
  return <span className="dot" data-state={state} aria-hidden="true" />
}
