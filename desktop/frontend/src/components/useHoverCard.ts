import { useEffect, useRef, useState } from 'react'

const OPEN_DELAY = 300
const CLOSE_DELAY = 150

export type HoverHandlers = {
  onMouseEnter(e: React.MouseEvent<HTMLElement>): void
  onMouseLeave(): void
}

export function useHoverCard() {
  const [anchor, setAnchor] = useState<DOMRect | null>(null)
  const timer = useRef<number>(0)
  const open = anchor !== null

  useEffect(() => () => window.clearTimeout(timer.current), [])

  useEffect(() => {
    if (!open) return
    const close = () => setAnchor(null)
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && close()
    window.addEventListener('keydown', onKey)
    window.addEventListener('scroll', close, true)
    window.addEventListener('blur', close)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('scroll', close, true)
      window.removeEventListener('blur', close)
    }
  }, [open])

  const hover: HoverHandlers = {
    onMouseEnter(e) {
      window.clearTimeout(timer.current)
      if (open) return
      const rect = e.currentTarget.getBoundingClientRect()
      timer.current = window.setTimeout(() => setAnchor(rect), OPEN_DELAY)
    },
    onMouseLeave() {
      window.clearTimeout(timer.current)
      timer.current = window.setTimeout(() => setAnchor(null), CLOSE_DELAY)
    },
  }
  return { anchor, hover }
}
