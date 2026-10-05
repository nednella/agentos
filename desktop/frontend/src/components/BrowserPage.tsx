import { useEffect, useRef, useState } from 'react'
import type { KeyboardEvent, MouseEvent } from 'react'
import { useAgentos } from '../AgentosContext'
import { api, on } from '../api'
import { decodeBase64 } from '../base64'
import type { BrowserInput, BrowserState, Session } from '../types'
import { BrowserToolbar } from './BrowserToolbar'

type BrowserPageProps = { session: Session; state: BrowserState }

type Frame = { data: string; width: number; height: number }

const MOVE_INTERVAL_MS = 33
const DOUBLE_ESC_MS = 500
const BUTTONS = ['left', 'middle', 'right'] as const

const textFor = (e: KeyboardEvent): string => {
  if (e.key === 'Enter') return '\r'
  return e.key.length === 1 && !e.ctrlKey ? e.key : ''
}

const modifiers = (e: { altKey: boolean; ctrlKey: boolean; metaKey: boolean; shiftKey: boolean }) =>
  (e.altKey ? 1 : 0) | (e.ctrlKey ? 2 : 0) | (e.metaKey ? 4 : 0) | (e.shiftKey ? 8 : 0)

export function BrowserPage({ session, state }: BrowserPageProps) {
  const { focus } = useAgentos()
  const id = session.id
  const canvas = useRef<HTMLCanvasElement>(null)
  const area = useRef<HTMLDivElement>(null)
  const frameSize = useRef({ width: 0, height: 0 })
  const painting = useRef(false)
  const queued = useRef<Frame | null>(null)
  const lastMove = useRef(0)
  const lastEsc = useRef(0)
  const [focused, setFocused] = useState(false)

  useEffect(() => {
    let alive = true
    painting.current = false
    queued.current = null

    const settle = () => {
      if (!alive) return
      painting.current = false
      const next = queued.current
      queued.current = null
      if (next) paint(next)
    }
    const draw = (source: CanvasImageSource, frame: Frame) => {
      const el = canvas.current
      if (alive && el) {
        if (el.width !== frame.width || el.height !== frame.height) {
          el.width = frame.width
          el.height = frame.height
        }
        el.getContext('2d')?.drawImage(source, 0, 0)
        frameSize.current = { width: frame.width, height: frame.height }
      }
      settle()
    }
    const paintWithImage = (frame: Frame) => {
      const image = new Image()
      image.onload = () => draw(image, frame)
      image.onerror = settle
      image.src = `data:image/jpeg;base64,${frame.data}`
    }
    const paint = (frame: Frame) => {
      painting.current = true
      if (typeof createImageBitmap !== 'function') {
        paintWithImage(frame)
        return
      }
      createImageBitmap(new Blob([decodeBase64(frame.data)], { type: 'image/jpeg' })).then(
        (bitmap) => {
          draw(bitmap, frame)
          bitmap.close()
        },
        () => paintWithImage(frame),
      )
    }
    const off = on('browser:frame', (frame) => {
      if (frame.id !== id) return
      if (painting.current) queued.current = frame
      else paint(frame)
    })
    return () => {
      alive = false
      off()
    }
  }, [id])

  useEffect(() => {
    void api.browserView(id, true)
    const onVisibility = () => void api.browserView(id, !document.hidden)
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      document.removeEventListener('visibilitychange', onVisibility)
      void api.browserView(id, false)
    }
  }, [id])

  useEffect(() => {
    const el = area.current
    if (!el) return
    const send = () => void api.browserResize(id, Math.round(el.clientWidth), Math.round(el.clientHeight))
    send()
    let timer = 0
    const observer = new ResizeObserver(() => {
      clearTimeout(timer)
      timer = window.setTimeout(send, 150)
    })
    observer.observe(el)
    return () => {
      clearTimeout(timer)
      observer.disconnect()
    }
  }, [id])

  const point = (clientX: number, clientY: number) => {
    const rect = canvas.current?.getBoundingClientRect()
    if (!rect || rect.width === 0) return { x: 0, y: 0 }
    const { width, height } = frameSize.current
    return {
      x: ((clientX - rect.left) * (width || rect.width)) / rect.width,
      y: ((clientY - rect.top) * (height || rect.height)) / rect.height,
    }
  }

  const send = (input: BrowserInput) => void api.browserInput(id, input)

  useEffect(() => {
    const el = canvas.current
    if (!el) return
    const onWheel = (e: WheelEvent) => {
      e.preventDefault()
      send({ type: 'wheel', ...point(e.clientX, e.clientY), deltaX: e.deltaX, deltaY: e.deltaY, modifiers: modifiers(e) })
    }
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [id])

  const mouse = (action: 'move' | 'down' | 'up') => (e: MouseEvent) => {
    if (action === 'down') area.current?.focus()
    if (action === 'move') {
      const now = performance.now()
      if (now - lastMove.current < MOVE_INTERVAL_MS) return
      lastMove.current = now
    }
    const button = action === 'move' ? (e.buttons & 1 ? 'left' : 'none') : (BUTTONS[e.button] ?? 'left')
    send({ type: 'mouse', action, ...point(e.clientX, e.clientY), button, clickCount: action === 'move' ? 0 : Math.max(e.detail, 1), modifiers: modifiers(e) })
  }

  const key = (action: 'down' | 'up') => (e: KeyboardEvent) => {
    if (e.metaKey) return
    if (e.key === 'Escape' && action === 'down' && !e.repeat) {
      const now = performance.now()
      if (now - lastEsc.current < DOUBLE_ESC_MS) {
        lastEsc.current = 0
        area.current?.blur()
        focus('terminal')
        return
      }
      lastEsc.current = now
    }
    e.preventDefault()
    send({ type: 'key', action, key: e.key, code: e.code, text: textFor(e), modifiers: modifiers(e) })
  }

  return (
    <div className="absolute inset-0 flex flex-col">
      <BrowserToolbar id={id} state={state} />
      <div
        ref={area}
        tabIndex={0}
        role="application"
        aria-label="Browser page. Keyboard goes to the page while focused."
        className="browser-page relative min-h-0 flex-1 bg-white"
        onKeyDown={key('down')}
        onKeyUp={key('up')}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        onPaste={(e) => {
          e.preventDefault()
          send({ type: 'paste', text: e.clipboardData.getData('text') })
        }}
      >
        <canvas
          ref={canvas}
          className="block h-full w-full"
          onMouseMove={mouse('move')}
          onMouseDown={mouse('down')}
          onMouseUp={mouse('up')}
          onContextMenu={(e) => e.preventDefault()}
        />
        {focused && (
          <span className="pointer-events-none absolute bottom-2 left-2 rounded-sm border border-line-strong bg-raised/90 px-2 py-1 text-label text-soft">
            keyboard goes to the page · Esc Esc to leave
          </span>
        )}
      </div>
    </div>
  )
}
