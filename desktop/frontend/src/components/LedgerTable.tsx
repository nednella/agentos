import { dollars, tokenCount } from '../spend'
import type { LedgerRow } from '../types'

type LedgerTableProps = { projects: LedgerRow[]; limit?: number }

const HEADS = ['Prompts', 'Tasks', 'PRs merged', 'Issues closed', 'Tokens', 'Est. cost']

export function LedgerTable({ projects, limit }: LedgerTableProps) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-body">
        <thead>
          <tr className="text-small text-dim">
            <th className="py-1.5 text-left font-medium">Project</th>
            {HEADS.map((head) => (
              <th key={head} className="py-1.5 pl-3 text-right font-medium whitespace-nowrap">{head}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {projects.slice(0, limit).map((p) => (
            <tr key={p.project} className="border-t border-line">
              <td className="py-1.5">{p.project}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">{p.prompts.toLocaleString()}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">{p.sessions.toLocaleString()}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">{p.prsMerged.toLocaleString()}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">{p.issuesClosed.toLocaleString()}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">{tokenCount(p.tokens)}</td>
              <td className="mono py-1.5 pl-3 text-right whitespace-nowrap">~{dollars(p.cost)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
