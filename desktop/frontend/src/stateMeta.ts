import type { State } from './types'

export const STATE_LABEL: Record<State, string> = {
  waiting: 'Needs you',
  finished: 'Finished',
  working: 'Working',
  idle: 'Idle',
}
