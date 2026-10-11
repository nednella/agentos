import { Figure } from './StatsFigure'

type LedgerFiguresProps = { figures: Parameters<typeof Figure>[0][] }

export function LedgerFigures({ figures }: LedgerFiguresProps) {
  return (
    <div className="@container">
      <div className="grid grid-cols-2 gap-3 @lg:grid-flow-col @lg:auto-cols-fr @lg:grid-cols-none">
        {figures.map((f) => (
          <Figure key={f.label} {...f} />
        ))}
      </div>
    </div>
  )
}
