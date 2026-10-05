import { useRef } from 'react'
import type { KeyboardEvent, PointerEvent } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { Icon } from './Icon'
import { ShellTerminal } from './ShellTerminal'

const HEADER_REM = 1.75
const KEY_STEP_REM = 1

export function ShellStrip() {
  const { project, focus } = useAgentos()
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
          className="flex h-9 w-full items-center gap-2 px-4 text-left"
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
      <div className="panel-head flex items-center gap-2 border-b border-line px-4 select-none" style={{ height: `${HEADER_REM}rem` }}>
        <span className="mono text-small text-accent">{project?.name ?? ''} ❯</span>
        <span className="text-small text-dim">Shell</span>
        <button
          className="btn btn-ghost ml-auto h-6 w-6 justify-center px-0"
          title="Collapse the shell"
          aria-label="Collapse the shell"
          onClick={() => setShellOpen(false)}
        >
          <Icon name="chevron" size={13} />
        </button>
      </div>
      <div className="relative" style={{ height: `${shellRem}rem` }}>
        <ShellTerminal visible />
      </div>
    </div>
  )
}
