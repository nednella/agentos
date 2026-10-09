import { useState, type DragEvent, type ReactNode } from 'react'
import type { IssueDrag } from '../useIssueDrag'

type QueueSectionProps = { name: string; className?: string; drag: IssueDrag; children: ReactNode }

export function QueueSection({ name, className, drag, children }: QueueSectionProps) {
  const [over, setOver] = useState(false)
  const { dragging } = drag
  const state = !dragging || dragging.section === name ? undefined : dragging.moves.includes(name) ? (over ? 'over' : 'target') : 'refused'

  const onDragOver = (e: DragEvent) => {
    if (state !== 'target' && state !== 'over') return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    setOver(true)
  }
  const onDragLeave = (e: DragEvent) => {
    if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setOver(false)
  }
  const onDrop = (e: DragEvent) => {
    if (state !== 'target' && state !== 'over') return
    e.preventDefault()
    setOver(false)
    drag.drop(name)
  }

  return (
    <section className={className} data-drop={state} onDragOver={onDragOver} onDragLeave={onDragLeave} onDrop={onDrop}>
      {children}
    </section>
  )
}
