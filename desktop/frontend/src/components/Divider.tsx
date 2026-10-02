import { useRef } from 'react'
import type { KeyboardEvent, PointerEvent } from 'react'
import { DEFAULT_WIDTH, useLayout } from '../LayoutContext'
import type { SidePanel } from '../LayoutContext'

type DividerProps = { panel: SidePanel }

const KEY_STEP_REM = 1

export function Divider({ panel }: DividerProps) {
  const { widths, setWidth } = useLayout()
  const drag = useRef<{ x: number; width: number; rem: number } | null>(null)
  const direction = panel === 'sidebar' ? 1 : -1

  const rootRem = () => parseFloat(getComputedStyle(document.documentElement).fontSize)

  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    e.currentTarget.setPointerCapture(e.pointerId)
    e.currentTarget.dataset.dragging = 'true'
    drag.current = { x: e.clientX, width: widths[panel], rem: rootRem() }
  }

  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    const start = drag.current
    if (start) setWidth(panel, start.width + (direction * (e.clientX - start.x)) / start.rem, false)
  }

  const onPointerUp = (e: PointerEvent<HTMLDivElement>) => {
    const start = drag.current
    if (!start) return
    drag.current = null
    delete e.currentTarget.dataset.dragging
    setWidth(panel, start.width + (direction * (e.clientX - start.x)) / start.rem, true)
  }

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return
    e.preventDefault()
    const sign = e.key === 'ArrowRight' ? 1 : -1
    setWidth(panel, widths[panel] + direction * sign * KEY_STEP_REM, true)
  }

  return (
    <div
      className="divider"
      role="separator"
      aria-orientation="vertical"
      aria-label={`Resize ${panel === 'sidebar' ? 'queue and notes' : 'sessions'} panel`}
      tabIndex={0}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onDoubleClick={() => setWidth(panel, DEFAULT_WIDTH[panel], true)}
      onKeyDown={onKeyDown}
    />
  )
}
