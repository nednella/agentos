import type { LedgerRow } from '../types'
import { duration } from '../time'
import { Figure } from './StatsFigure'

type LedgerFiguresProps = { totals: LedgerRow; since: string }

export function LedgerFigures({ totals, since }: LedgerFiguresProps) {
  const hours = Math.round(totals.workMs / 3_600_000)
  const agentTime = hours ? `${hours.toLocaleString()}h` : duration(totals.workMs)
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] gap-3">
      <Figure label="Prompts" value={totals.prompts.toLocaleString()} hint={since ? `since ${since}` : 'none yet'} />
      <Figure label="Sessions" value={totals.sessions.toLocaleString()} hint={`${totals.issueSessions.toLocaleString()} from an issue`} />
      <Figure label="Agent time" value={agentTime} hint="time agents spent working" />
    </div>
  )
}
