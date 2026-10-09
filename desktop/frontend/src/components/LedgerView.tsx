import { useEffect, useState } from 'react'
import { api } from '../api'
import { useAgentos } from '../AgentosContext'
import type { Ledger } from '../types'
import { LedgerFigures } from './LedgerFigures'
import { LedgerHeatmap } from './LedgerHeatmap'
import { LedgerTable } from './LedgerTable'

type LedgerViewProps = { days: number }

const isEmpty = (ledger: Ledger) =>
  ledger.since === '' && ledger.totals.prsMerged === null && ledger.totals.issuesClosed === null

export function LedgerView({ days }: LedgerViewProps) {
  const { report } = useAgentos()
  const [ledger, setLedger] = useState<Ledger | null>(null)

  useEffect(() => {
    let current = true
    setLedger(null)
    report(async () => {
      const next = await api.ledger(days)
      if (current) setLedger(next)
    })
    return () => {
      current = false
    }
  }, [days, report])

  if (ledger === null) return <p className="text-small text-dim">Loading…</p>
  if (isEmpty(ledger)) {
    return (
      <div className="flex flex-col gap-1 py-16 text-center">
        <p className="text-body font-medium">Nothing recorded yet</p>
        <p className="text-small text-dim">Prompts, sessions and agent time are counted from now on. {ledger.github}</p>
      </div>
    )
  }
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6">
      <LedgerFigures totals={ledger.totals} since={ledger.since} />
      {ledger.github && <p className="-mt-3 text-small text-dim">{ledger.github}</p>}
      <LedgerHeatmap heat={ledger.heat} />
      <LedgerTable projects={ledger.projects} />
    </div>
  )
}
