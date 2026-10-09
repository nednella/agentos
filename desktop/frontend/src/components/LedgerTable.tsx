import type { LedgerRow } from '../types'

type LedgerTableProps = { projects: LedgerRow[] }

const num = (n: number | null) => (n === null ? '—' : n.toLocaleString())
const HEADS = ['Prompts', 'Sessions', 'PRs merged', 'Issues closed']

export function LedgerTable({ projects }: LedgerTableProps) {
  return (
    <section>
      <h3 className="label pb-1">By project</h3>
      <table className="w-full text-body">
        <thead>
          <tr className="text-small text-dim">
            <th className="py-1.5 text-left font-medium">Project</th>
            {HEADS.map((head) => (
              <th key={head} className="py-1.5 text-right font-medium">{head}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {projects.map((p) => (
            <tr key={p.project} className="border-t border-line">
              <td className="py-1.5">{p.project}</td>
              <td className="mono py-1.5 text-right">{num(p.prompts)}</td>
              <td className="mono py-1.5 text-right">{num(p.sessions)}</td>
              <td className="mono py-1.5 text-right">{num(p.prsMerged)}</td>
              <td className="mono py-1.5 text-right">{num(p.issuesClosed)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
