import { memo, useCallback, useEffect, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { usePaletteItems } from '../usePaletteItems'
import type { PaletteItem } from '../usePaletteItems'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'
import { StateDot } from './StateDot'

export function Palette() {
  const { overlay } = useAgentos()
  if (overlay !== 'palette') return null
  return <PaletteDialog />
}

function PaletteDialog() {
  const { setOverlay, report } = useAgentos()
  const [query, setQuery] = useState('')
  const [cursor, setCursor] = useState(0)
  const [confirming, setConfirming] = useState<string | null>(null)
  const [prompting, setPrompting] = useState<PaletteItem | null>(null)
  const items = usePaletteItems(prompting ? '' : query)
  const results = items.slice(0, 60)
  const list = useRef<HTMLDivElement>(null)
  const close = () => setOverlay(null)

  useEffect(() => {
    list.current?.querySelector('[data-active="true"]')?.scrollIntoView({ block: 'nearest' })
  }, [cursor, results.length])

  const run = (item: PaletteItem, args: string[] = []) => {
    close()
    report(() => item.run(args))
  }

  const activate = (item: PaletteItem) => {
    if (item.prompt) {
      setPrompting(item)
      setQuery(item.prompt.initial)
      return
    }
    if (item.confirm && confirming !== item.key) {
      setConfirming(item.key)
      return
    }
    run(item)
  }

  const latest = useRef({ results, activate })
  latest.current = { results, activate }
  const hover = useCallback((index: number) => setCursor(index), [])
  const pick = useCallback((index: number) => {
    const item = latest.current.results[index]
    if (item) latest.current.activate(item)
  }, [])

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape' && prompting) {
      e.stopPropagation()
      setPrompting(null)
      setQuery('')
      return
    }
    if (prompting) {
      const text = query.trim()
      if (e.key === 'Enter' && text) run(prompting, text.split(/\s+/))
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setCursor((c) => Math.min(c + 1, results.length - 1))
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setCursor((c) => Math.max(c - 1, 0))
    }
    if (e.key === 'Enter' && results[cursor]) activate(results[cursor])
  }

  return (
    <Overlay align="center" label="Command palette" onClose={close}>
      <div className="flex items-center gap-3 border-b border-line px-4 py-3">
        {prompting && <span className="label text-accent">{prompting.prompt?.label || prompting.label}</span>}
        <input
          autoFocus
          value={query}
          placeholder={prompting ? 'Type, then Enter' : 'Sessions, actions, projects, issues (@author label: type:)'}
          className="min-w-0 flex-1 bg-transparent text-title outline-none"
          onChange={(e) => {
            setQuery(e.target.value)
            setCursor(0)
            setConfirming(null)
          }}
          onKeyDown={onKeyDown}
        />
        <Keycap>esc</Keycap>
      </div>
      {!prompting && (
        <div ref={list} className="min-h-0 flex-1 overflow-y-auto py-1">
          {results.length === 0 && <p className="px-4 py-6 text-center text-dim">Nothing matches.</p>}
          {results.map((item, i) => (
            <PaletteRow
              key={item.key}
              item={item}
              active={i === cursor}
              confirming={confirming === item.key}
              showGroup={results[i - 1]?.group !== item.group}
              index={i}
              onHover={hover}
              onPick={pick}
            />
          ))}
        </div>
      )}
    </Overlay>
  )
}

type PaletteRowProps = {
  item: PaletteItem
  active: boolean
  confirming: boolean
  showGroup: boolean
  index: number
  onHover(index: number): void
  onPick(index: number): void
}

function PaletteRowView({ item, active, confirming, showGroup, index, onHover, onPick }: PaletteRowProps) {
  return (
    <>
      {showGroup && <div className="label px-4 pt-3 pb-1">{item.group}</div>}
      <button className="row row-pick h-9 items-center gap-3 px-4" data-selected={active} data-active={active} onMouseEnter={() => onHover(index)} onClick={() => onPick(index)}>
        <span className="flex w-3 flex-none justify-center">{item.state && <StateDot state={item.state} />}</span>
        <span className="truncate">{item.label}</span>
        <span className="mono ml-auto truncate text-small text-dim">{confirming ? 'Enter again to confirm' : item.hint}</span>
        {item.keys && <Keycap>{item.keys}</Keycap>}
      </button>
    </>
  )
}

const PaletteRow = memo(PaletteRowView, (a, b) => a.item.key === b.item.key && a.item.label === b.item.label && a.item.hint === b.item.hint && a.active === b.active && a.confirming === b.confirming && a.showGroup === b.showGroup)
