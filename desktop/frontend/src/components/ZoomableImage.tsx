import { useEffect, useRef } from 'react'

export type ImageView = { scale: number; x: number; y: number }

export const FIT: ImageView = { scale: 1, x: 0, y: 0 }
const MAX_SCALE = 8

type ZoomableImageProps = {
  src: string
  alt: string
  view: ImageView
  onView(view: ImageView): void
  onDismiss(): void
}

export function ZoomableImage({ src, alt, view, onView, onDismiss }: ZoomableImageProps) {
  const box = useRef<HTMLDivElement>(null)
  const img = useRef<HTMLImageElement>(null)
  const shown = clamp(view, img.current)
  const latest = useRef(shown)
  latest.current = shown
  const onViewRef = useRef(onView)
  onViewRef.current = onView

  useEffect(() => {
    const el = box.current
    if (!el) return
    // React attaches wheel listeners as passive, so it cannot stop the page zoom on a pinch.
    const onWheel = (e: WheelEvent) => {
      e.preventDefault()
      const factor = Math.exp(-e.deltaY * (e.ctrlKey ? 0.01 : 0.002))
      latest.current = clamp(zoomAt(latest.current, latest.current.scale * factor, pointFromCentre(el, e)), img.current)
      onViewRef.current(latest.current)
    }
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [])

  const startPan = (e: React.MouseEvent) => {
    if (e.button !== 0) return
    if (shown.scale === 1) {
      if (e.target === e.currentTarget) onDismiss()
      return
    }
    e.preventDefault()
    const start = { mx: e.clientX, my: e.clientY, ...shown }
    const move = (m: MouseEvent) => onView(clamp({ scale: start.scale, x: start.x + m.clientX - start.mx, y: start.y + m.clientY - start.my }, img.current))
    const up = () => {
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
  }

  return (
    <div
      ref={box}
      className="relative flex h-[72vh] w-full items-center justify-center overflow-hidden"
      style={{ cursor: shown.scale > 1 ? 'grab' : 'zoom-in' }}
      onMouseDown={startPan}
      onDoubleClick={(e) => box.current && onView(shown.scale > 1 ? FIT : zoomAt(shown, 2, pointFromCentre(box.current, e)))}
    >
      <img
        ref={img}
        src={src}
        alt={alt}
        draggable={false}
        className="block max-h-full max-w-full select-none rounded-md border border-line-strong object-contain"
        style={{ transform: `translate(${shown.x}px, ${shown.y}px) scale(${shown.scale})` }}
      />
    </div>
  )
}

export function zoomTo(view: ImageView, scale: number): ImageView {
  return zoomAt(view, scale, { x: 0, y: 0 })
}

function pointFromCentre(el: HTMLElement, e: { clientX: number; clientY: number }) {
  const r = el.getBoundingClientRect()
  return { x: e.clientX - r.left - r.width / 2, y: e.clientY - r.top - r.height / 2 }
}

function zoomAt(view: ImageView, scale: number, at: { x: number; y: number }): ImageView {
  const next = Math.min(MAX_SCALE, Math.max(1, scale))
  const k = next / view.scale
  return { scale: next, x: at.x - (at.x - view.x) * k, y: at.y - (at.y - view.y) * k }
}

function clamp(view: ImageView, image: HTMLImageElement | null): ImageView {
  if (!image) return view
  const maxX = ((view.scale - 1) * image.offsetWidth) / 2
  const maxY = ((view.scale - 1) * image.offsetHeight) / 2
  return { scale: view.scale, x: Math.min(maxX, Math.max(-maxX, view.x)), y: Math.min(maxY, Math.max(-maxY, view.y)) }
}
