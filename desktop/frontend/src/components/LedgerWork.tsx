import { useState } from 'react'
import { dollars, tokenCount } from '../spend'
import type { WorkRow } from '../types'

type LedgerWorkProps = { work: WorkRow[]; limit?: number }

type SortKey = 'sessions' | 'tokens' | 'cost'

const COLUMNS: { key: SortKey; label: string }[] = [
  { key: 'sessions', label: 'Tasks' },
  { key: 'tokens', label: 'Tokens' },
  { key: 'cost', label: 'Est. cost' },
]

export function LedgerWork({ work, limit }: LedgerWorkProps) {
  const [sortKey, setSortKey] = useState<SortKey>('cost')
  const rows = [...work].sort((a, b) => b[sortKey] - a[sortKey]).slice(0, limit)
  return (
    <table className="w-full text-body">
      <thead>
        <tr className="text-small text-dim">
          <th className="w-full py-1.5 text-left font-medium">Work</th>
          {COLUMNS.map((c) => (
            <th
              key={c.key}
              className="py-1.5 pl-4 text-right font-medium whitespace-nowrap"
              aria-sort={sortKey === c.key ? 'descending' : 'none'}
            >
              <button
                className="font-medium"
                style={{ color: sortKey === c.key ? 'var(--text)' : undefined }}
                title={`Sort by ${c.label.toLowerCase()}, highest first`}
                onClick={() => setSortKey(c.key)}
              >
                {c.label}
              </button>
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((w) => (
          <tr key={`${w.project} ${w.issue} ${w.session}`} className="border-t border-line">
            <td className="max-w-0 py-1.5 pr-3">
              <span className="block truncate" title={w.title}>
                {w.title}
              </span>
              <span className="block text-small text-dim">{w.project}</span>
            </td>
            <td className="mono py-1.5 pl-4 text-right">{w.sessions}</td>
            <td className="mono py-1.5 pl-4 text-right whitespace-nowrap">{tokenCount(w.tokens)}</td>
            <td className="mono py-1.5 pl-4 text-right whitespace-nowrap">~{dollars(w.cost)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
