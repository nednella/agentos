import type { ReactNode } from 'react'
import { Icon } from './Icon'
import type { IconName } from './Icon'

export type ChipTone = 'ok' | 'warn' | 'bad' | 'dim' | 'merged'

const COLOR: Record<ChipTone, string> = {
  ok: 'var(--finished)',
  warn: 'var(--idle)',
  bad: 'var(--danger)',
  dim: 'var(--text-dim)',
  merged: 'var(--accent)',
}

type ChipProps = {
  tone: ChipTone
  icon: IconName
  plain?: boolean
  title?: string
  children?: ReactNode
}

export function Chip({ tone, icon, plain = false, title, children }: ChipProps) {
  return (
    <span
      className={`inline-flex h-5 flex-none items-center gap-1 rounded-sm text-label whitespace-nowrap ${plain ? 'px-0.5' : 'px-1.5 font-medium'}`}
      style={{
        color: COLOR[tone],
        background: plain ? undefined : 'color-mix(in srgb, currentColor 14%, transparent)',
      }}
      title={title}
    >
      <Icon name={icon} size={12} />
      {children}
    </span>
  )
}
