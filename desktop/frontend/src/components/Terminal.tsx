import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'
import { Terminal as Xterm } from '@xterm/xterm'
import { useEffect, useRef } from 'react'
import { useAgentos } from '../AgentosContext'
import { api, on } from '../api'
import { decodeBase64 } from '../base64'
import { useLayout } from '../LayoutContext'
import { terminalFont, terminalTheme } from '../terminalTheme'

type TerminalProps = { id: string; active: boolean; kind?: 'session' | 'shell'; onLeave?(): void }

const DOUBLE_ESC_MS = 500

const BASE_FONT_PX = 14
const SHIFT_ENTER = '\x1b[13;2u'
const SYSTEM_KEYS = new Set(['c', 'v', 'x', 'm', 'h', 'w', 'q'])

// A remount of one id must not let the old mount's close land after the new mount's open.
const generations = new Map<string, number>()

// The last focus request a terminal acted on, so a terminal that becomes active later
// (auto-selection after a session ends) does not replay an old request.
let focusedRequest = -1

const DEVICE_REPORT = /^(\x1b\[[?>]?[\d;]*c|\x1bP[^\x1b]*\x1b\\|\x1b\]\d+;[^\x07\x1b]*(\x07|\x1b\\))+$/

export function Terminal({ id, active, kind = 'session', onLeave }: TerminalProps) {
  const focusTarget = kind === 'shell' ? 'shell' : 'terminal'
  const leave = useRef(onLeave)
  leave.current = onLeave
  const lastEsc = useRef(0)
  const { focusRequest } = useAgentos()
  const { scale } = useLayout()
  const host = useRef<HTMLDivElement>(null)
  const xterm = useRef<Xterm | null>(null)
  const fit = useRef<FitAddon | null>(null)
  const opened = useRef(false)
  const renderer = useRef({ attach() {}, detach() {} })

  // A terminal mounted hidden has no size, so the core opens it at the first fit while visible.
  const sync = useRef(() => {})
  sync.current = () => {
    const term = xterm.current
    if (!term || !fit.current || !active) return
    fit.current.fit()
    if (opened.current) {
      void api.termResize(id, term.cols, term.rows)
      return
    }
    opened.current = true
    void api.termOpen(id, term.cols, term.rows)
  }

  useEffect(() => {
    const generation = (generations.get(id) ?? 0) + 1
    generations.set(id, generation)
    opened.current = false
    const term = new Xterm({
      fontFamily: terminalFont,
      fontSize: Math.round(BASE_FONT_PX * scale * 2) / 2,
      lineHeight: 1.18,
      cursorBlink: true,
      scrollback: 5000,
      allowProposedApi: true,
      theme: terminalTheme,
    })
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.loadAddon(new Unicode11Addon())
    term.unicode.activeVersion = '11'
    term.open(host.current!)
    let webgl: WebglAddon | null = null
    renderer.current = {
      attach() {
        if (webgl) return
        try {
          const addon = new WebglAddon()
          addon.onContextLoss(() => {
            addon.dispose()
            if (webgl === addon) webgl = null
          })
          term.loadAddon(addon)
          webgl = addon
        } catch {
          // the default renderer keeps working without WebGL
        }
      },
      detach() {
        webgl?.dispose()
        webgl = null
      },
    }

    term.attachCustomKeyEventHandler((event) => {
      if (kind === 'shell' && event.key === 'Escape' && event.type === 'keydown') {
        const now = performance.now()
        if (now - lastEsc.current < DOUBLE_ESC_MS) {
          lastEsc.current = 0
          leave.current?.()
          return false
        }
        lastEsc.current = now
      }
      const key = event.key.toLowerCase()
      if (event.metaKey && key === 'c' && term.hasSelection()) {
        void navigator.clipboard.writeText(term.getSelection())
        return false
      }
      if (event.metaKey && !SYSTEM_KEYS.has(key)) return false
      if (event.key === 'Enter' && event.shiftKey) {
        if (event.type === 'keydown') {
          event.preventDefault()
          void api.termWrite(id, SHIFT_ENTER)
        }
        return false
      }
      return true
    })

    term.onData((data) => {
      // xterm answers tmux's "what terminal are you" queries (device attributes,
      // version, colours). tmux has stopped waiting by the time the answer
      // crosses the bridge, so it would land in the agent's prompt as typed text.
      if (DEVICE_REPORT.test(data)) return
      void api.termWrite(id, data)
    })
    term.onBinary((data) => void api.termWrite(id, data))

    const offData = on('term:data', (payload) => {
      if (payload.id === id) term.write(decodeBase64(payload.data))
    })
    const offExit = on('term:exit', (payload) => {
      if (payload.id === id) term.write('\r\n\x1b[2m[session ended]\x1b[0m\r\n')
    })

    xterm.current = term
    fit.current = fitAddon

    let timer = 0
    const observer = new ResizeObserver(() => {
      clearTimeout(timer)
      timer = window.setTimeout(() => sync.current(), 80)
    })
    observer.observe(host.current!)

    return () => {
      clearTimeout(timer)
      observer.disconnect()
      offData()
      offExit()
      if (opened.current) {
        // Deferred a tick: an immediate remount of this id bumps the generation and keeps the session open.
        setTimeout(() => {
          if (generations.get(id) !== generation) return
          generations.delete(id)
          void api.termClose(id)
        })
      }
      renderer.current.detach()
      renderer.current = { attach() {}, detach() {} }
      term.dispose()
      xterm.current = null
      fit.current = null
    }
  }, [id])

  useEffect(() => {
    const term = xterm.current
    if (!term) return
    term.options.fontSize = Math.round(BASE_FONT_PX * scale * 2) / 2
    const frame = requestAnimationFrame(() => sync.current())
    return () => cancelAnimationFrame(frame)
  }, [scale])

  useEffect(() => {
    if (!active) return
    const frame = requestAnimationFrame(() => sync.current())
    return () => cancelAnimationFrame(frame)
  }, [active])

  useEffect(() => {
    if (!active) return
    renderer.current.attach()
    return () => renderer.current.detach()
  }, [active, id])

  useEffect(() => {
    if (!active || focusRequest.target !== focusTarget || focusRequest.n <= focusedRequest) return
    focusedRequest = focusRequest.n
    xterm.current?.focus()
  }, [active, focusRequest, focusTarget])

  return <div ref={host} className={`absolute inset-0 px-3 py-2 ${active ? '' : 'hidden'}`} aria-hidden={!active} />
}
