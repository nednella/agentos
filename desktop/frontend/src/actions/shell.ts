import type { Action, ActionContext } from './types'

export const shellActions = ({ layout }: ActionContext): Action[] => [
  {
    id: 'focus-shell',
    label: 'Focus the shell',
    group: 'Navigate',
    run: () => layout.focusPanel('shell'),
  },
]
