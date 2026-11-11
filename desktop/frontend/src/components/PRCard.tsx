import { useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useAgentos } from '../AgentosContext'
import type { PR, Session } from '../types'
import { BranchBar } from './BranchBar'
import { Chip } from './Chip'
import { Icon } from './Icon'
import { PRTiles } from './PRTiles'
import type { HoverHandlers } from './useHoverCard'
import { WorkChips } from './WorkChips'

type PRCardProps = { session: Session; anchor: DOMRect; hover: HoverHandlers }

const WIDTH = 300
const MARGIN = 8

const PILL: Record<
  PR['state'],
  {
    icon: 'git-pull-request' | 'git-pull-request-draft' | 'git-merge' | 'git-pull-request-closed'
    label: string
  }
> = {
  open: { icon: 'git-pull-request', label: 'Open' },
  draft: { icon: 'git-pull-request-draft', label: 'Draft' },
  merged: { icon: 'git-merge', label: 'Merged' },
  closed: { icon: 'git-pull-request-closed', label: 'Closed' },
}

export function PRCard({ session, anchor, hover }: PRCardProps) {
  const { openPR, report } = useAgentos()
  const card = useRef<HTMLDivElement>(null)
  const [top, setTop] = useState(anchor.bottom + 4)
  const { pr, git } = session
  const left = Math.max(MARGIN, Math.min(anchor.left, window.innerWidth - WIDTH - MARGIN))
  const open = pr?.state === 'draft' || pr?.state === 'open'

  useLayoutEffect(() => {
    const height = card.current?.offsetHeight ?? 0
    setTop(Math.max(MARGIN, Math.min(anchor.bottom + 4, window.innerHeight - height - MARGIN)))
  }, [anchor])

  return createPortal(
    <div
      ref={card}
      {...hover}
      className="fade-in fixed overflow-hidden rounded-lg border border-line-strong bg-surface text-small text-soft shadow-lg"
      style={{ top, left, width: WIDTH, zIndex: 'var(--z-popup)' }}
    >
      <div className="flex min-w-0 items-center gap-2 px-3 pt-2.5 pb-1.5">
        {pr ? (
          <>
            <span
              className="inline-flex h-[22px] flex-none items-center gap-1 rounded-full pr-2 pl-1.5 text-label font-medium text-white"
              style={{ background: `var(--pr-${pr.state})` }}
            >
              <Icon name={PILL[pr.state].icon} size={14} />
              {PILL[pr.state].label}
            </span>
            <button
              className="min-w-0 truncate text-left font-semibold text-ink hover:underline"
              title={`Open #${pr.number} on GitHub`}
              onClick={() => report(() => openPR(session))}
            >
              {pr.title}
            </button>
            <span className="flex-none text-dim">#{pr.number}</span>
          </>
        ) : (
          <>
            <Icon name="git-branch" size={14} />
            <span className="mono min-w-0 truncate font-medium text-ink">{session.branch}</span>
            <span className="ml-auto flex-none rounded-full border border-line-strong px-2 text-label text-dim">No PR</span>
          </>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 px-3 pb-2.5 text-label text-dim">
        {pr && git && (
          <span className="inline-flex items-center gap-1">
            <Ref>{git.base}</Ref>←<Ref>{session.branch}</Ref>
          </span>
        )}
        {git && (
          <span className="ml-auto">
            <BranchBar ahead={git.ahead} behind={git.behind} />
          </span>
        )}
      </div>
      <div className="flex flex-wrap gap-1.5 px-3 pb-2.5 empty:hidden">
        {open && pr.conflicting && (
          <Chip tone="bad" icon="alert">
            Conflicts
          </Chip>
        )}
        <WorkChips session={session} alwaysWords />
      </div>
      {pr && <PRTiles pr={pr} />}
    </div>,
    document.body,
  )
}

type RefProps = { children: string }

function Ref({ children }: RefProps) {
  return (
    <span className="mono inline-flex h-[18px] max-w-[9rem] items-center truncate rounded-sm bg-hover px-1.5 text-soft">{children}</span>
  )
}
