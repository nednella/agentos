import { useRef } from 'react'
import type { KeyboardEvent, PointerEvent } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { ShellList } from './ShellList'
import { ShellTerminal } from './ShellTerminal'

const HEADER_REM = 1.75
const KEY_STEP_REM = 1
// The strip sits on the window's rounded bottom corners; the terminal pads itself 0.75rem, so this margin
// puts its text in line with the header and clear of the curve.
const cornerPad = { paddingLeft: 'var(--corner-pad)', paddingRight: 'var(--corner-pad)' }
const terminalInset = { marginLeft: 'calc(var(--corner-pad) - 0.75rem)', marginRight: 'calc(var(--corner-pad) - 0.75rem)', marginBottom: '0.375rem' }

export function ShellStrip() {
  const { project, shellIds, focus } = useAgentos()
  const { shellOpen, shellRem, setShellOpen, setShellHeight, resetShellHeight } = useLayout()
  const drag = useRef<{ y: number; rem: number; root: number } | null>(null)

  const root = () => parseFloat(getComputedStyle(document.documentElement).fontSize)

  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    e.currentTarget.setPointerCapture(e.pointerId)
    e.currentTarget.dataset.dragging = 'true'
    drag.current = { y: e.clientY, rem: shellRem, root: root() }
  }
  const heightAt = (e: PointerEvent<HTMLDivElement>) => {
    const start = drag.current!
    return Math.min(start.rem + (start.y - e.clientY) / start.root, (0.6 * window.innerHeight) / start.root)
  }
  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    if (drag.current) setShellHeight(heightAt(e), false)
  }
  const onPointerUp = (e: PointerEvent<HTMLDivElement>) => {
    if (!drag.current) return
    setShellHeight(heightAt(e), true)
    drag.current = null
    delete e.currentTarget.dataset.dragging
  }

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return
    e.preventDefault()
    const step = e.key === 'ArrowUp' ? KEY_STEP_REM : -KEY_STEP_REM
    setShellHeight(Math.min(shellRem + step, (0.6 * window.innerHeight) / root()), true)
  }

  if (!shellOpen) {
    return (
      <div data-panel="shell" className="flex-none border-t border-line bg-surface">
        <button
          className="flex h-9 w-full items-center gap-2 text-left"
          style={cornerPad}
          aria-label="Open the shell"
          onClick={() => {
            setShellOpen(true)
            focus('shell')
          }}
        >
          <span className="mono text-small text-accent">{project?.name ?? ''} ❯</span>
          <span className="text-small text-dim">Shell</span>
        </button>
      </div>
    )
  }

  return (
    <div data-panel="shell" tabIndex={-1} className="relative flex-none border-t border-line bg-term">
      <div
        className="divider"
        role="separator"
        aria-orientation="horizontal"
        aria-label="Resize shell"
        tabIndex={0}
        title="Drag to resize, double-click to reset"
        onKeyDown={onKeyDown}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onDoubleClick={resetShellHeight}
      />
      <button
        className="panel-head flex w-full items-center gap-2 border-b border-line text-left"
        style={{ ...cornerPad, height: `${HEADER_REM}rem` }}
        aria-label="Collapse the shell"
        onClick={() => setShellOpen(false)}
      >
        <span className="mono text-small text-accent">{project?.name ?? ''} ❯</span>
        <span className="text-small text-dim">Shell</span>
      </button>
      <div className="flex" style={{ height: `${shellRem}rem`, marginBottom: '0.375rem' }}>
        <div className="relative min-w-0 flex-1" style={{ marginLeft: terminalInset.marginLeft }}>
          <ShellTerminal visible />
        </div>
        {shellIds.length > 0 && <ShellList />}
      </div>
    </div>
  )
}
