import type { Session } from '../types'
import { Chip } from './Chip'
import type { IconName } from './Icon'

type WorkChipsProps = { session: Session; alwaysWords?: boolean }

export function WorkChips({ session, alwaysWords = false }: WorkChipsProps) {
  const { git, pr } = session
  if (!git) return null
  const merged = pr?.state === 'merged'
  const working = session.state === 'working'
  const tone = merged ? 'bad' : working ? 'dim' : 'warn'
  const words = alwaysWords || !working || merged
  const items: { n: number; icon: IconName; label: string }[] = [
    merged ? { n: git.notInMain, icon: 'alert', label: 'not in main' } : { n: git.unpushed, icon: 'repo-push', label: 'unpushed' },
    { n: git.uncommitted, icon: 'file-diff', label: 'uncommitted' },
  ]

  return items
    .filter((item) => item.n > 0)
    .map(({ n, icon, label }) => (
      <Chip key={label} tone={tone} icon={icon} plain={!words} title={`${n} ${label}`}>
        <span className={words ? '' : 'mono'}>{words ? `${n} ${label}` : n}</span>
      </Chip>
    ))
}
