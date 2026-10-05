import type { PaletteItem } from '../usePaletteItems'
import { Keycap } from './Keycap'
import { StateDot } from './StateDot'

type PaletteRowProps = {
  item: PaletteItem
  active: boolean
  confirming: boolean
  showGroup: boolean
  index: number
  onHover(index: number): void
  onPick(index: number): void
}

export function PaletteRow({ item, active, confirming, showGroup, index, onHover, onPick }: PaletteRowProps) {
  return (
    <>
      {showGroup && <div className="label px-4 pt-3 pb-1">{item.group}</div>}
      <button
        id={`palette-option-${index}`}
        role="option"
        aria-selected={active}
        tabIndex={-1}
        className="row row-pick h-9 items-center gap-3 px-4"
        data-selected={active}
        data-cursor={active}
        onMouseEnter={() => onHover(index)}
        onClick={() => onPick(index)}
      >
        <span className="flex w-3 flex-none justify-center">{item.state && <StateDot state={item.state} />}</span>
        <span className="truncate">{item.label}</span>
        <span className="mono ml-auto truncate text-small text-dim">{confirming ? 'Enter again to confirm' : item.hint}</span>
        {item.keys && <Keycap>{item.keys}</Keycap>}
      </button>
    </>
  )
}
