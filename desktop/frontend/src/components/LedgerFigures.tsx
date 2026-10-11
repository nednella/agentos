import { Figure } from './StatsFigure'

type LedgerFiguresProps = { figures: Parameters<typeof Figure>[0][] }

// Four cards need a wider panel than three before they fit on one row.
const ONE_ROW = {
  three: '@lg:grid-flow-col @lg:auto-cols-fr @lg:grid-cols-none',
  four: '@2xl:grid-flow-col @2xl:auto-cols-fr @2xl:grid-cols-none',
}

export function LedgerFigures({ figures }: LedgerFiguresProps) {
  return (
    <div className="@container">
      <div className={`grid grid-cols-2 gap-3 ${figures.length > 3 ? ONE_ROW.four : ONE_ROW.three}`}>
        {figures.map((f) => (
          <Figure key={f.label} {...f} />
        ))}
      </div>
    </div>
  )
}
