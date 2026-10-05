import type { WaitKind } from './types'

export const KIND_COLOR: Record<WaitKind, string> = {
  permission: 'var(--waiting)',
  question: 'var(--accent)',
  idle: 'var(--idle)',
}

export const KIND_LABEL: Record<WaitKind, string> = {
  permission: 'permission',
  question: 'question',
  idle: 'reply landed',
}
