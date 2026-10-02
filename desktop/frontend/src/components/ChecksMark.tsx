import type { PR } from '../types'

type ChecksMarkProps = { checks: PR['checks'] }

const LABEL: Record<PR['checks'], string> = {
  none: '',
  pending: 'checks running',
  passing: 'checks passing',
  failing: 'checks failing',
}

const COLOR: Record<PR['checks'], string> = {
  none: 'currentColor',
  pending: 'var(--text-dim)',
  passing: 'var(--finished)',
  failing: 'var(--danger)',
}

export function ChecksMark({ checks }: ChecksMarkProps) {
  if (checks === 'none') return null
  return (
    <svg
      width="12"
      height="12"
      viewBox="0 0 12 12"
      fill="none"
      stroke={COLOR[checks]}
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      role="img"
      aria-label={LABEL[checks]}
    >
      <title>{LABEL[checks]}</title>
      <ChecksShape checks={checks} />
    </svg>
  )
}

function ChecksShape({ checks }: ChecksMarkProps) {
  if (checks === 'passing') return <path d="m2 6.5 2.7 2.7L10 3" />
  if (checks === 'failing') return <path d="m3 3 6 6M9 3 3 9" />
  return <circle cx="6" cy="6" r="3.5" strokeDasharray="2.2 2" />
}
