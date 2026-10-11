import { dollars, priceNote, tokenCount } from '../spend'
import { duration } from '../time'
import type { Ledger, LedgerRow } from '../types'
import { Figure } from './StatsFigure'

type LedgerFiguresProps = { totals: LedgerRow; since: string; usage: Ledger['usage'] }

const average = (ms: number) => (ms ? duration(ms) : '—')

export function LedgerFigures({ totals, since, usage }: LedgerFiguresProps) {
  const spent = usage.totals
  const perTask = spent.sessions ? spent.tokens / spent.sessions : 0
  const hours = Math.round(totals.workMs / 3_600_000)
  const agentTime = hours ? `${hours.toLocaleString()}h` : duration(totals.workMs)
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] gap-3">
      <Figure label="Prompts" value={totals.prompts.toLocaleString()} hint={since ? `since ${since}` : 'none yet'} />
      <Figure label="Tasks" value={totals.sessions.toLocaleString()} hint={`${totals.issueSessions.toLocaleString()} from an issue`} />
      <Figure label="Agent time" value={agentTime} hint="time agents spent working" />
      <Figure
        label="PRs merged"
        value={totals.prsMerged.toLocaleString()}
        hint={`${totals.prsOpened.toLocaleString()} opened, ${totals.prsClosed.toLocaleString()} closed`}
      />
      <Figure label="Issues closed" value={totals.issuesClosed.toLocaleString()} hint={`${totals.issuesFiled.toLocaleString()} filed`} />
      <Figure label="Task length" value={average(totals.avgSessionMs)} hint="on average" />
      <Figure label="Lead time" value={average(totals.avgLeadMs)} hint="PR open to merge, on average" />
      <Figure
        label="Tokens"
        value={tokenCount(spent.tokens)}
        hint={`~${dollars(spent.cost)} estimated`}
        title={priceNote(usage.pricesAsOf)}
      />
      <Figure
        label="Tokens per task"
        value={perTask ? tokenCount(Math.round(perTask)) : '—'}
        hint={perTask ? `~${dollars(spent.cost / spent.sessions)} estimated` : 'no usage yet'}
        title={priceNote(usage.pricesAsOf)}
      />
    </div>
  )
}
