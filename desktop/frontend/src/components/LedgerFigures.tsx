import type { LedgerRow } from '../types'
import { duration } from '../time'
import { Figure } from './StatsFigure'

type LedgerFiguresProps = { totals: LedgerRow; since: string }

const count = (n: number | null) => (n === null ? '—' : n.toLocaleString())
const hint = (n: number | null, text: string) => (n === null ? '' : `${n.toLocaleString()} ${text}`)

export function LedgerFigures({ totals, since }: LedgerFiguresProps) {
  const hours = Math.round(totals.workMs / 3_600_000)
  const agentTime = hours ? `${hours.toLocaleString()}h` : duration(totals.workMs)
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] gap-3">
      <Figure label="Prompts" value={count(totals.prompts)} hint={since ? `since ${since}` : 'none yet'} />
      <Figure label="Sessions" value={count(totals.sessions)} hint={`${totals.issueSessions.toLocaleString()} from an issue`} />
      <Figure label="PRs merged" value={count(totals.prsMerged)} hint={hint(totals.prsClosed, 'closed unmerged')} />
      <Figure label="Issues closed" value={count(totals.issuesClosed)} hint={hint(totals.issuesOpen, 'still open')} />
      <Figure label="Agent time" value={agentTime} hint="time agents spent working" />
    </div>
  )
}
