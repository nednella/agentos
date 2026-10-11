import { dollars, priceNote, tokenCount } from '../spend'
import type { Ledger } from '../types'

type LedgerUsageProps = { usage: Ledger['usage'] }

const HEADS = ['Tasks', 'Tokens', 'Per task', 'Cost']

export function LedgerUsage({ usage }: LedgerUsageProps) {
  const { totals, byModel, pricesAsOf } = usage
  return (
    <section>
      <h3 className="label pb-1">Tokens by model and effort</h3>
      <p className="pb-2 text-small text-dim">
        {tokenCount(totals.input)} input, {tokenCount(totals.output)} output ({tokenCount(totals.thinking)} thinking),{' '}
        {tokenCount(totals.cacheRead)} cache read, {tokenCount(totals.cacheWrite)} cache write. {priceNote(pricesAsOf)}; subagents included.
      </p>
      <table className="w-full text-body">
        <thead>
          <tr className="text-small text-dim">
            <th className="py-1.5 text-left font-medium">Model</th>
            <th className="py-1.5 pl-3 text-left font-medium">Effort</th>
            {HEADS.map((head) => (
              <th key={head} className="py-1.5 pl-3 text-right font-medium whitespace-nowrap">
                {head}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {byModel.map((r) => (
            <tr key={`${r.model} ${r.effort}`} className="border-t border-line">
              <td className="mono py-1.5 whitespace-nowrap" title={r.model}>
                {r.model.replace(/^claude-/, '') || 'unknown'}
              </td>
              <td className="py-1.5 pl-3 text-soft">{r.effort || 'default'}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">{r.sessions.toLocaleString()}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">{tokenCount(r.tokens)}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">
                {r.sessions ? tokenCount(Math.round(r.tokens / r.sessions)) : '—'}
              </td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">~{dollars(r.cost)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
