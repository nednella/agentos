import { useEffect } from 'react'
import type { RefObject } from 'react'

const EDGE = 48
const MAX_STEP = 16

// WKWebView does not scroll an overflow container under a native drag, so the list scrolls itself
// while the pointer is held in a band at its top or bottom edge, faster nearer the edge.
export function useDragEdgeScroll(list: RefObject<HTMLElement>, dragging: boolean) {
  useEffect(() => {
    const el = list.current
    if (!dragging || !el) return
    let step = 0
    let frame = 0
    const scroll = () => {
      el.scrollTop += step
      frame = step ? requestAnimationFrame(scroll) : 0
    }
    const onDragOver = (e: DragEvent) => {
      const { top, bottom, left, right } = el.getBoundingClientRect()
      const inside = e.clientX >= left && e.clientX <= right && e.clientY >= top && e.clientY <= bottom
      const depth = !inside ? 0 : e.clientY < top + EDGE ? e.clientY - top - EDGE : Math.max(0, e.clientY - bottom + EDGE)
      step = Math.sign(depth) * Math.ceil((Math.abs(depth) / EDGE) * MAX_STEP)
      if (step && !frame) frame = requestAnimationFrame(scroll)
    }
    document.addEventListener('dragover', onDragOver)
    return () => {
      document.removeEventListener('dragover', onDragOver)
      cancelAnimationFrame(frame)
    }
  }, [list, dragging])
}
