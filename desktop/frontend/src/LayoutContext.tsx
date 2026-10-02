import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { useAgentos } from './AgentosContext'
import type { SidebarTab } from './AgentosContext'
import { devFlags } from './api'
import { readStored, writeStored } from './storage'

export type LayoutMode = 'wide' | 'medium' | 'narrow'
export type Panel = 'sidebar' | 'terminal' | 'sessions' | 'command'
export type MobilePanel = 'queue' | 'notes' | 'session' | 'sessions'
export type SidePanel = 'sidebar' | 'sessions'

export const SCALES = [0.85, 0.92, 1, 1.1, 1.2, 1.35, 1.5]
const DEFAULT_SCALE = 1
const WIDE_MIN = 1250
const MEDIUM_MIN = 900

export const DEFAULT_WIDTH: Record<SidePanel, number> = { sidebar: 21, sessions: 19 }
export const WIDTH_LIMITS: Record<SidePanel, [number, number]> = { sidebar: [15, 32], sessions: [14, 32] }

type Layout = {
  mode: LayoutMode
  width: number
  scale: number
  sidebarOpen: boolean
  sessionsOpen: boolean
  mobilePanel: MobilePanel
  widths: Record<SidePanel, number>
  statsOpen: boolean
  digestOpen: boolean
  zoom(step: -1 | 0 | 1): void
  setMobilePanel(panel: MobilePanel): void
  setWidth(panel: SidePanel, rem: number, commit: boolean): void
  toggleSidebar(): void
  toggleSessions(): void
  setSidebarOpen(open: boolean): void
  setSessionsOpen(open: boolean): void
  showSidebarTab(tab: SidebarTab): void
  toggleStats(): void
  toggleDigest(): void
  closeCentre(): void
  returnToTerminal(): void
  focusPanel(panel: Panel): void
  movePanel(delta: -1 | 1): void
}

const LayoutContext = createContext<Layout | null>(null)

export function useLayout(): Layout {
  const value = useContext(LayoutContext)
  if (!value) throw new Error('useLayout must be used inside LayoutProvider')
  return value
}

function modeFor(width: number): LayoutMode {
  if (width >= WIDE_MIN) return 'wide'
  return width >= MEDIUM_MIN ? 'medium' : 'narrow'
}

function useWindowWidth(): number {
  const [width, setWidth] = useState(window.innerWidth)
  useEffect(() => {
    const update = () => setWidth(window.innerWidth)
    window.addEventListener('resize', update)
    return () => window.removeEventListener('resize', update)
  }, [])
  return width
}

function currentPanel(): Panel {
  const el = document.activeElement
  const panel = el instanceof HTMLElement ? el.closest<HTMLElement>('[data-panel]')?.dataset.panel : undefined
  return (panel as Panel | undefined) ?? 'terminal'
}

type LayoutProviderProps = { children: ReactNode }

export function LayoutProvider({ children }: LayoutProviderProps) {
  const { selectedId, sidebarTab, setSidebarTab, focus } = useAgentos()
  const width = useWindowWidth()
  const mode = modeFor(width)
  const [scale, setScale] = useState(() => readStored('agentos.scale', DEFAULT_SCALE))
  const [openWide, setOpenWide] = useState(() => readStored('agentos.sidebarOpen', true))
  const [sessionsPersisted, setSessionsPersisted] = useState(() => readStored('agentos.sessionsOpen', true))
  const [overlayOpen, setOverlayOpen] = useState(false)
  const [mobilePanel, setMobilePanel] = useState<MobilePanel>('session')
  const [widths, setWidths] = useState(() => ({ ...DEFAULT_WIDTH, ...readStored<Partial<Record<SidePanel, number>>>('agentos.widths', {}) }))
  const [statsOpen, setStatsOpen] = useState(devFlags.view === 'stats')
  const [digestOpen, setDigestOpen] = useState(devFlags.view === 'digest')

  useLayoutEffect(() => {
    document.documentElement.style.setProperty('--ui-scale', String(scale))
  }, [scale])

  const previousSelected = useRef(selectedId)
  useEffect(() => {
    if (previousSelected.current === selectedId) return
    const first = previousSelected.current === null
    previousSelected.current = selectedId
    if (first) return
    setStatsOpen(false)
    setDigestOpen(false)
    if (mode === 'medium') setOverlayOpen(false)
    if (mode === 'narrow') setMobilePanel('session')
  }, [selectedId, mode])

  const sidebarMobile = mobilePanel === 'queue' || mobilePanel === 'notes'
  const sidebarOpen = { wide: openWide, medium: overlayOpen, narrow: sidebarMobile }[mode]
  const sessionsOpen = mode === 'narrow' ? mobilePanel === 'sessions' : sessionsPersisted

  const setSidebarOpen = useCallback(
    (open: boolean) => {
      if (mode === 'wide') {
        setOpenWide(open)
        writeStored('agentos.sidebarOpen', open)
      } else if (mode === 'medium') setOverlayOpen(open)
      else setMobilePanel(open ? sidebarTab : 'session')
    },
    [mode, sidebarTab],
  )

  const setSessionsOpen = useCallback(
    (open: boolean) => {
      if (mode === 'narrow') {
        setMobilePanel(open ? 'sessions' : 'session')
        return
      }
      setSessionsPersisted(open)
      writeStored('agentos.sessionsOpen', open)
    },
    [mode],
  )

  const showSidebarTab = useCallback(
    (tab: SidebarTab) => {
      setSidebarTab(tab)
      if (mode === 'narrow') setMobilePanel(tab)
      else setSidebarOpen(true)
    },
    [mode, setSidebarOpen, setSidebarTab],
  )

  const visiblePanels = useCallback((): Panel[] => {
    if (mode === 'narrow') {
      if (sidebarMobile) return ['sidebar', 'command']
      return mobilePanel === 'session' ? ['terminal', 'command'] : ['sessions', 'command']
    }
    return [...(sidebarOpen ? (['sidebar'] as const) : []), 'terminal', ...(sessionsOpen ? (['sessions'] as const) : []), 'command']
  }, [mode, sidebarMobile, mobilePanel, sidebarOpen, sessionsOpen])

  const focusPanel = useCallback(
    (panel: Panel) => {
      if (panel === 'sidebar') setSidebarOpen(true)
      if (panel === 'sessions') setSessionsOpen(true)
      if (panel === 'terminal' && mode === 'narrow') setMobilePanel('session')
      focus(panel)
    },
    [focus, mode, setSessionsOpen, setSidebarOpen],
  )

  const value = useMemo<Layout>(
    () => ({
      mode,
      width,
      scale,
      sidebarOpen,
      sessionsOpen,
      mobilePanel,
      widths,
      statsOpen,
      digestOpen,
      zoom(step) {
        const at = SCALES.indexOf(scale)
        const next = step === 0 ? DEFAULT_SCALE : SCALES[Math.min(Math.max((at < 0 ? 2 : at) + step, 0), SCALES.length - 1)]
        setScale(next)
        writeStored('agentos.scale', next)
      },
      setMobilePanel,
      setWidth(panel, rem, commit) {
        const [min, max] = WIDTH_LIMITS[panel]
        const next = { ...widths, [panel]: Math.min(Math.max(rem, min), max) }
        setWidths(next)
        if (commit) writeStored('agentos.widths', next)
      },
      toggleSidebar: () => setSidebarOpen(!sidebarOpen),
      toggleSessions: () => setSessionsOpen(!sessionsOpen),
      setSidebarOpen,
      setSessionsOpen,
      showSidebarTab,
      toggleStats() {
        setStatsOpen(!statsOpen)
        setDigestOpen(false)
        if (!statsOpen && mode === 'narrow') setMobilePanel('session')
        if (!statsOpen && mode === 'medium') setOverlayOpen(false)
      },
      toggleDigest() {
        setDigestOpen(!digestOpen)
        setStatsOpen(false)
        if (!digestOpen && mode === 'narrow') setMobilePanel('session')
        if (!digestOpen && mode === 'medium') setOverlayOpen(false)
      },
      closeCentre() {
        setStatsOpen(false)
        setDigestOpen(false)
      },
      returnToTerminal() {
        if (mode === 'medium') setOverlayOpen(false)
        focus('terminal')
      },
      focusPanel,
      movePanel(delta) {
        const panels = visiblePanels()
        const at = Math.max(panels.indexOf(currentPanel()), 0)
        focusPanel(panels[(at + delta + panels.length) % panels.length])
      },
    }),
    [mode, width, scale, sidebarOpen, sessionsOpen, mobilePanel, widths, statsOpen, digestOpen, setSidebarOpen, setSessionsOpen, showSidebarTab, focusPanel, visiblePanels, focus],
  )

  return <LayoutContext.Provider value={value}>{children}</LayoutContext.Provider>
}
