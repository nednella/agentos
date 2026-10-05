import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'
import { Terminal as Xterm } from '@xterm/xterm'
import { useEffect, useRef } from 'react'
import { useAgentos } from '../AgentosContext'
import { api, on } from '../api'
import { useLayout } from '../LayoutContext'
import { terminalFont, terminalTheme } from '../terminalTheme'

type TerminalProps = { id: string; active: boolean }

const BASE_FONT_PX = 14
const SHIFT_ENTER = '\x1b[13;2u'
const SYSTEM_KEYS = new Set(['c', 'v', 'x', 'm', 'h', 'w', 'q'])

function decode(base64: string): Uint8Array {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes
}

const DEVICE_REPORT = /^(\x1b\[[?>]?[\d;]*c|\x1bP[^\x1b]*\x1b\\|\x1b\]\d+;[^\x07\x1b]*(\x07|\x1b\\))+$/

export function Terminal({ id, active }: TerminalProps) {
  const { focusRequest } = useAgentos()
  const { scale } = useLayout()
  const host = useRef<HTMLDivElement>(null)
  const xterm = useRef<Xterm | null>(null)
  const fit = useRef<FitAddon | null>(null)

  const sync = useRef(() => {})
  sync.current = () => {
    const term = xterm.current
    if (!term || !fit.current || !active) return
    fit.current.fit()
    void api.termResize(id, term.cols, term.rows)
  }

  useEffect(() => {
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
    try {
      const webgl = new WebglAddon()
      webgl.onContextLoss(() => webgl.dispose())
      term.loadAddon(webgl)
    } catch {
      // the default renderer keeps working without WebGL
    }

    term.attachCustomKeyEventHandler((event) => {
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
      if (payload.id === id) term.write(decode(payload.data))
    })
    const offExit = on('term:exit', (payload) => {
      if (payload.id === id) term.write('\r\n\x1b[2m[session ended]\x1b[0m\r\n')
    })

    xterm.current = term
    fit.current = fitAddon
    fitAddon.fit()
    void api.termOpen(id, term.cols, term.rows)

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
      void api.termClose(id)
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
    const frame = requestAnimationFrame(() => {
      sync.current()
      xterm.current?.focus()
    })
    return () => cancelAnimationFrame(frame)
  }, [active])

  useEffect(() => {
    if (active && focusRequest.target === 'terminal') xterm.current?.focus()
  }, [active, focusRequest])

  return <div ref={host} className={`absolute inset-0 px-3 py-2 ${active ? '' : 'hidden'}`} aria-hidden={!active} />
}
