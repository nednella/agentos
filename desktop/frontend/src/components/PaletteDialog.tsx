import { useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useArmedConfirm } from '../useArmedConfirm'
import { usePaletteItems } from '../usePaletteItems'
import type { PaletteItem } from '../usePaletteItems'
import { useScrollCursorIntoView } from '../useScrollCursorIntoView'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'
import { PaletteRow } from './PaletteRow'

export function PaletteDialog() {
  const { setOverlay, report } = useAgentos()
  const [query, setQuery] = useState('')
  const [cursor, setCursor] = useState(0)
  const confirm = useArmedConfirm<string>()
  const [prompting, setPrompting] = useState<PaletteItem | null>(null)
  const items = usePaletteItems(prompting ? '' : query)
  const results = items.slice(0, 60)
  const list = useRef<HTMLDivElement>(null)
  const close = () => setOverlay(null)

  useScrollCursorIntoView(list, results[cursor]?.key)

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
    if (item.confirm && confirm.armed !== item.key) {
      confirm.arm(item.key)
      return
    }
    run(item)
  }

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape' && prompting) {
      e.stopPropagation()
      setPrompting(null)
      setQuery('')
      return
    }
    if (prompting) {
      const text = query.trim()
      if (e.key === 'Enter' && text) run(prompting, [text])
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setCursor((c) => Math.max(0, Math.min(c + 1, results.length - 1)))
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
          role="combobox"
          aria-label={prompting ? prompting.prompt?.label || prompting.label : 'Search sessions, actions, projects and issues'}
          aria-expanded={!prompting}
          aria-controls="palette-list"
          aria-activedescendant={!prompting && results[cursor] ? `palette-option-${cursor}` : undefined}
          value={query}
          placeholder={prompting ? 'Type, then Enter' : 'Sessions, actions, projects, issues (@author label: type:)'}
          className="min-w-0 flex-1 bg-transparent text-title outline-none"
          onChange={(e) => {
            setQuery(e.target.value)
            setCursor(0)
            confirm.disarm()
          }}
          onKeyDown={onKeyDown}
        />
        <Keycap>esc</Keycap>
      </div>
      {!prompting && (
        <div ref={list} id="palette-list" role="listbox" aria-label="Results" className="min-h-0 flex-1 overflow-y-auto py-1">
          {results.length === 0 && <p className="px-4 py-6 text-center text-dim">Nothing matches.</p>}
          {results.map((item, i) => (
            <PaletteRow
              key={item.key}
              item={item}
              active={i === cursor}
              confirming={confirm.armed === item.key}
              showGroup={results[i - 1]?.group !== item.group}
              index={i}
              onHover={setCursor}
              onPick={(index) => activate(results[index])}
            />
          ))}
        </div>
      )}
    </Overlay>
  )
}
