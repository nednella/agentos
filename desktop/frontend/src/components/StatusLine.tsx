import type { ReactNode } from 'react'
import { useAgentos } from '../AgentosContext'
import type { PR, Session } from '../types'
import { Chip } from './Chip'
import type { ChipTone } from './Chip'
import { PRCard } from './PRCard'
import { useHoverCard } from './useHoverCard'
import { WorkChips } from './WorkChips'

type StatusLineProps = { session: Session; children?: ReactNode }

const PR_ICON = {
  draft: 'git-pull-request-draft',
  open: 'git-pull-request',
  merged: 'git-merge',
  closed: 'git-pull-request-closed',
} as const

const PR_TONE: Record<PR['state'], ChipTone> = {
  draft: 'dim',
  open: 'ok',
  merged: 'merged',
  closed: 'bad',
}

export function StatusLine({ session, children }: StatusLineProps) {
  const { openPR, report } = useAgentos()
  const { anchor, hover } = useHoverCard()
  const { pr, git } = session
  if (!pr && !git) return null
  const live = pr?.state === 'draft' || pr?.state === 'open'
  const passing = pr?.checks === 'passing' || pr?.checks === 'none'
  const ready = live && passing && !pr.conflicting && pr.newComments === 0 && git?.unpushed === 0 && git.uncommitted === 0

  return (
    <div className="flex w-full flex-wrap items-center gap-x-2 gap-y-1" {...hover}>
      {pr ? (
        <button className="flex-none" title={`Open PR #${pr.number} (${pr.state}) on GitHub`} onClick={() => report(() => openPR(session))}>
          <Chip tone={PR_TONE[pr.state]} icon={PR_ICON[pr.state]} plain>
            <span className="mono">#{pr.number}</span>
          </Chip>
        </button>
      ) : (
        <Chip tone="dim" icon="git-pull-request" plain>
          no PR
        </Chip>
      )}
      {live && pr.checks === 'passing' && <Chip tone="ok" icon="tick" plain title="Checks pass" />}
      {live && pr.checks === 'pending' && <Chip tone="dim" icon="circle" plain title="Checks running" />}
      {live && pr.checks === 'failing' && (
        <Chip tone="bad" icon="cross" title="Checks fail">
          checks
        </Chip>
      )}
      {pr && pr.newComments > 0 && (
        <Chip tone="warn" icon="comment" title={`${pr.newComments} new comments`}>
          {pr.newComments}
        </Chip>
      )}
      {live && pr.conflicting && (
        <Chip tone="bad" icon="alert" title="The branch conflicts with the base">
          conflict
        </Chip>
      )}
      {ready && (
        <Chip tone="ok" icon="check-circle-fill" title="Ready to merge. This is not an approval.">
          ready
        </Chip>
      )}
      <WorkChips session={session} />
      {git && git.behind > 0 && pr?.state !== 'merged' && pr?.state !== 'closed' && (
        <Chip tone="dim" icon="arrow-down" plain title={`${git.behind} commits behind ${git.base}`}>
          <span className="mono">{git.behind}</span>
        </Chip>
      )}
      {children}
      {anchor && <PRCard session={session} anchor={anchor} hover={hover} />}
    </div>
  )
}
