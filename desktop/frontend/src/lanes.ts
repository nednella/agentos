import type { Lane } from './types'

export const LANES: { id: Lane; label: string }[] = [
  { id: 'ready', label: 'Ready' },
  { id: 'plan', label: 'Needs plan' },
  { id: 'you', label: 'Needs you' },
  { id: 'inbox', label: 'Inbox' },
  { id: 'idea', label: 'Ideas' },
]
