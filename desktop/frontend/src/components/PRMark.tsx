import type { PR } from '../types'
import { ChecksMark } from './ChecksMark'

type PRMarkProps = { pr: PR }

export function PRMark({ pr }: PRMarkProps) {
  return (
    <span
      className="mono flex flex-none items-center gap-1 text-label text-soft"
      title={`PR #${pr.number}, ${pr.state}, checks ${pr.checks}`}
    >
      PR
      <ChecksMark checks={pr.checks} />
    </span>
  )
}
