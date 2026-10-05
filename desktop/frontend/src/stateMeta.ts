import type { State } from './types'

export const STATE_LABEL: Record<State, string> = {
  waiting: 'Needs you',
  working: 'Working',
  idle: 'Idle',
  ended: 'Ended',
}
