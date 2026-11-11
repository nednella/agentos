import type { ReactNode } from 'react'
import type { PR } from '../types'
import { Icon } from './Icon'
import type { IconName } from './Icon'

type PRTilesProps = { pr: PR }

const REVIEW: Record<PR['review'], { icon: IconName; color: string; label: string }> = {
  approved: { icon: 'tick', color: 'var(--finished)', label: 'Approved' },
  changes: {
    icon: 'file-diff',
    color: 'var(--danger)',
    label: 'Changes requested',
  },
  commented: { icon: 'comment', color: 'var(--text-dim)', label: 'Commented' },
  '': { icon: 'eye', color: 'var(--text-dim)', label: 'No review yet' },
}

export function PRTiles({ pr }: PRTilesProps) {
  const failing = pr.failedChecks.length
  const checks: { icon: IconName; color: string; n: number; tinted: boolean } =
    failing > 0
      ? {
          icon: 'x-circle-fill',
          color: 'var(--danger)',
          n: failing,
          tinted: true,
        }
      : pr.runningChecks > 0
        ? {
            icon: 'dot-fill',
            color: 'var(--idle)',
            n: pr.runningChecks,
            tinted: true,
          }
        : {
            icon: 'check-circle-fill',
            color: 'var(--finished)',
            n: pr.passedChecks,
            tinted: false,
          }
  const review = REVIEW[pr.review]

  return (
    <>
      <div className="grid grid-cols-3 border-t border-line">
        <Tile label="Checks">
          <span style={{ color: checks.color }}>
            <Icon name={checks.icon} size={14} />
          </span>
          <Count n={checks.n} color={checks.tinted ? checks.color : ''} />
        </Tile>
        <Tile label="Review" title={review.label} divided>
          <span style={{ color: review.color }}>
            <Icon name={review.icon} size={14} />
          </span>
        </Tile>
        <Tile label="Comments" divided>
          <Icon name="comment-discussion" size={14} />
          <Count n={pr.newComments} color={pr.newComments > 0 ? 'var(--idle)' : ''} />
        </Tile>
      </div>
      {failing > 0 && (
        <div className="flex flex-wrap gap-1.5 px-3 pb-2.5">
          {pr.failedChecks.map((name) => (
            <span
              key={name}
              className="mono inline-flex h-5 items-center gap-1 rounded-full border border-line-strong pr-2 pl-1.5 text-label text-soft"
            >
              <span style={{ color: 'var(--danger)' }}>
                <Icon name="x-circle-fill" size={12} />
              </span>
              {name}
            </span>
          ))}
        </div>
      )}
    </>
  )
}

type TileProps = {
  label: string
  title?: string
  divided?: boolean
  children: ReactNode
}

function Tile({ label, title, divided = false, children }: TileProps) {
  return (
    <div
      className={`flex flex-col items-center gap-0.5 px-0.5 py-2 text-label text-dim ${divided ? 'border-l border-line' : ''}`}
      title={title}
    >
      <span className="inline-flex h-[18px] items-center gap-1 text-soft">{children}</span>
      {label}
    </div>
  )
}

type CountProps = { n: number; color: string }

function Count({ n, color }: CountProps) {
  return (
    <span
      className="mono inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-hover px-1.5 text-label font-medium"
      style={
        color
          ? {
              color,
              background: 'color-mix(in srgb, currentColor 14%, transparent)',
            }
          : undefined
      }
    >
      {n}
    </span>
  )
}
