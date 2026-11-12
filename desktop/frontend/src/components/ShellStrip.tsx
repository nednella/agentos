import { useRef } from 'react'
import type { KeyboardEvent, PointerEvent } from 'react'
import { useLayout } from '../LayoutContext'
import { ShellTabs } from './ShellTabs'
import { ShellTerminal } from './ShellTerminal'

const KEY_STEP_REM = 1
// The strip sits on the window's rounded bottom corners; the terminal pads itself 0.75rem, so this margin
// keeps its text clear of the curve.
const terminalInset = { marginLeft: 'calc(var(--corner-pad) - 0.75rem)', marginRight: 'calc(var(--corner-pad) - 0.75rem)', marginBottom: '0.375rem' }

export function ShellStrip() {
  const { shellOpen, shellRem, setShellHeight, resetShellHeight } = useLayout()
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

  return (
    <div data-panel="shell" tabIndex={-1} className={`relative flex-none border-t border-line ${shellOpen ? 'bg-term' : 'bg-surface'}`}>
      {shellOpen && (
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
      )}
      <ShellTabs />
      {shellOpen && (
        <div className="relative" style={{ ...terminalInset, height: `${shellRem}rem` }}>
          <ShellTerminal visible />
        </div>
      )}
    </div>
  )
}
