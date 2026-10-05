import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { useAgentos } from './AgentosContext'
import type { SidebarTab } from './AgentosContext'
import { devFlags } from './api'
import { readStored, writeStored } from './storage'

export type Band = 'compact' | 'medium' | 'wide'
export type LayoutMode = 'narrow' | Band
export type Panel = 'sidebar' | 'terminal' | 'sessions' | 'shell'
export type MobilePanel = 'queue' | 'notes' | 'session' | 'sessions' | 'shell'
export type SidePanel = 'sidebar' | 'sessions'

export const SCALES = [0.85, 0.92, 1, 1.1, 1.2, 1.35, 1.5]
const DEFAULT_SCALE = 1
const TABS_BELOW = 640
const COMPACT_BELOW = 900
const MEDIUM_BELOW = 1250
const TERMINAL_MIN_PX = 280
const MAIN_PAD_PX = 24
const GAPS_PX = 24
const STRIP_REM = 2.75
const SHARE: Record<SidePanel, number> = { sidebar: 0.34, sessions: 0.24 }
const DEFAULT_RANGE: Record<SidePanel, [number, number]> = { sidebar: [11, 26], sessions: [9.5, 22] }
const MIN_REM: Record<SidePanel, number> = { sidebar: 11, sessions: 9.5 }
const MAX_REM = 32
const LAYOUT_KEY = 'agentos.layout.v2'

type BandPrefs = { sidebarOpen: boolean; sessionsOpen: boolean; sidebar: number | null; sessions: number | null }
type AllPrefs = Record<Band, BandPrefs>

const fresh = (band: Band): BandPrefs => ({ sidebarOpen: true, sessionsOpen: band !== 'compact', sidebar: null, sessions: null })

type Layout = {
  mode: LayoutMode
  width: number
  scale: number
  sidebarOpen: boolean
  sessionsOpen: boolean
  sidebarPeek: boolean
  sessionsPeek: boolean
  mobilePanel: MobilePanel
  widths: Record<SidePanel, number>
  peekWidths: Record<SidePanel, number>
  statsOpen: boolean
  zoom(step: -1 | 0 | 1): void
  setMobilePanel(panel: MobilePanel): void
  setWidth(panel: SidePanel, rem: number, commit: boolean): void
  resetWidth(panel: SidePanel): void
  toggleSidebar(): void
  toggleSessions(): void
  setSidebarOpen(open: boolean): void
  setSessionsOpen(open: boolean): void
  showSidebarTab(tab: SidebarTab): void
  toggleStats(): void
  closeCentre(): void
  closePeek(): void
  returnToTerminal(): void
  focusPanel(panel: Panel): void
}

const LayoutContext = createContext<Layout | null>(null)

export function useLayout(): Layout {
  const value = useContext(LayoutContext)
  if (!value) throw new Error('useLayout must be used inside LayoutProvider')
  return value
}

function modeFor(width: number): LayoutMode {
  if (width < TABS_BELOW) return 'narrow'
  if (width < COMPACT_BELOW) return 'compact'
  return width < MEDIUM_BELOW ? 'medium' : 'wide'
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

const clamp = (value: number, [min, max]: [number, number]) => Math.min(Math.max(value, min), max)

type Fitted = { widths: Record<SidePanel, number>; full: Record<SidePanel, number>; sessionsForcedClosed: boolean }

function fit(width: number, prefs: BandPrefs, root: number): Fitted {
  const preferred = (panel: SidePanel) => prefs[panel] ?? clamp((SHARE[panel] * width) / root, DEFAULT_RANGE[panel])
  const strip = STRIP_REM * root
  let sidebar = prefs.sidebarOpen ? preferred('sidebar') * root : strip
  let sessions = prefs.sessionsOpen ? preferred('sessions') * root : strip
  const over = () => MAIN_PAD_PX + GAPS_PX + sidebar + sessions + TERMINAL_MIN_PX - width
  if (over() > 0 && prefs.sessionsOpen) sessions = Math.max(MIN_REM.sessions * root, sessions - over())
  if (over() > 0 && prefs.sidebarOpen) sidebar = Math.max(MIN_REM.sidebar * root, sidebar - over())
  const forced = over() > 0 && prefs.sessionsOpen
  if (forced) sessions = strip
  const full = { sidebar: preferred('sidebar'), sessions: preferred('sessions') }
  return { widths: { sidebar: sidebar / root, sessions: sessions / root }, full, sessionsForcedClosed: forced }
}

type LayoutProviderProps = { children: ReactNode }

export function LayoutProvider({ children }: LayoutProviderProps) {
  const { selectedId, sidebarTab, setSidebarTab, focus } = useAgentos()
  const width = useWindowWidth()
  const mode = modeFor(width)
  const band: Band = mode === 'narrow' ? 'compact' : mode
  const [scale, setScale] = useState(() => readStored('agentos.scale', DEFAULT_SCALE))
  const [stored, setStored] = useState(() => readStored<Partial<AllPrefs>>(LAYOUT_KEY, {}))
  const [peek, setPeek] = useState<SidePanel | null>(null)
  const [mobilePanel, setMobilePanel] = useState<MobilePanel>('session')
  const [statsOpen, setStatsOpen] = useState(devFlags.view === 'stats')

  useLayoutEffect(() => {
    document.documentElement.style.setProperty('--ui-scale', String(scale))
  }, [scale])

  const prefs: BandPrefs = { ...fresh(band), ...stored[band] }
  const fitted = fit(width, prefs, 16 * scale)
  const narrow = mode === 'narrow'
  const sidebarMobile = mobilePanel === 'queue' || mobilePanel === 'notes'
  const sidebarOpen = narrow ? sidebarMobile : prefs.sidebarOpen
  const sessionsOpen = narrow ? mobilePanel === 'sessions' : prefs.sessionsOpen && !fitted.sessionsForcedClosed

  const updatePrefs = useCallback(
    (patch: Partial<BandPrefs>) =>
      setStored((all) => {
        const next = { ...all, [band]: { ...fresh(band), ...all[band], ...patch } }
        writeStored(LAYOUT_KEY, next)
        return next
      }),
    [band],
  )

  const previousSelected = useRef(selectedId)
  useEffect(() => {
    if (previousSelected.current === selectedId) return
    const first = previousSelected.current === null
    previousSelected.current = selectedId
    if (first) return
    setStatsOpen(false)
    setPeek(null)
    if (narrow) setMobilePanel('session')
  }, [selectedId, narrow])

  const setSidebarOpen = useCallback(
    (open: boolean) => {
      setPeek(null)
      if (narrow) setMobilePanel(open ? sidebarTab : 'session')
      else updatePrefs({ sidebarOpen: open })
    },
    [narrow, sidebarTab, updatePrefs],
  )

  const setSessionsOpen = useCallback(
    (open: boolean) => {
      setPeek(null)
      if (narrow) setMobilePanel(open ? 'sessions' : 'session')
      else updatePrefs({ sessionsOpen: open })
    },
    [narrow, updatePrefs],
  )

  const showSidebarTab = useCallback(
    (tab: SidebarTab) => {
      setSidebarTab(tab)
      if (narrow) setMobilePanel(tab)
      else if (!prefs.sidebarOpen) setPeek('sidebar')
    },
    [narrow, prefs.sidebarOpen, setSidebarTab],
  )

  const focusPanel = useCallback(
    (panel: Panel) => {
      if (narrow) {
        if (panel === 'sidebar') setMobilePanel(sidebarTab)
        if (panel === 'sessions') setMobilePanel('sessions')
        if (panel === 'terminal') setMobilePanel('session')
      } else if (panel === 'sidebar') setPeek(prefs.sidebarOpen ? null : 'sidebar')
      else if (panel === 'sessions') setPeek(sessionsOpen ? null : 'sessions')
      else {
        setPeek(null)
      }
      focus(panel)
    },
    [focus, narrow, prefs.sidebarOpen, sessionsOpen, sidebarTab, updatePrefs],
  )

  const value = useMemo<Layout>(
    () => ({
      mode,
      width,
      scale,
      sidebarOpen,
      sessionsOpen,
      sidebarPeek: !narrow && !prefs.sidebarOpen && peek === 'sidebar',
      sessionsPeek: !narrow && !sessionsOpen && peek === 'sessions',
      mobilePanel,
      widths: fitted.widths,
      peekWidths: {
        sidebar: Math.min(Math.max(fitted.full.sidebar, 20), (0.8 * width) / (16 * scale)),
        sessions: Math.min(Math.max(fitted.full.sessions, 18), (0.8 * width) / (16 * scale)),
      },
      statsOpen,
      zoom(step) {
        const at = SCALES.indexOf(scale)
        const next = step === 0 ? DEFAULT_SCALE : SCALES[Math.min(Math.max((at < 0 ? 2 : at) + step, 0), SCALES.length - 1)]
        setScale(next)
        writeStored('agentos.scale', next)
      },
      setMobilePanel,
      setWidth(panel, rem, commit) {
        const next = Math.min(Math.max(rem, MIN_REM[panel]), MAX_REM)
        if (commit) updatePrefs({ [panel]: next })
        else setStored((all) => ({ ...all, [band]: { ...fresh(band), ...all[band], [panel]: next } }))
      },
      resetWidth: (panel) => updatePrefs({ [panel]: null }),
      toggleSidebar: () => setSidebarOpen(!sidebarOpen),
      toggleSessions: () => setSessionsOpen(!sessionsOpen),
      setSidebarOpen,
      setSessionsOpen,
      showSidebarTab,
      toggleStats() {
        setStatsOpen(!statsOpen)
        setPeek(null)
        if (!statsOpen && narrow) setMobilePanel('session')
      },
      closeCentre() {
        setStatsOpen(false)
      },
      closePeek: () => setPeek(null),
      returnToTerminal() {
        setPeek(null)
        focus('terminal')
      },
      focusPanel,
    }),
    [mode, width, scale, sidebarOpen, sessionsOpen, narrow, prefs.sidebarOpen, peek, mobilePanel, fitted.widths, fitted.full, statsOpen, band, setSidebarOpen, setSessionsOpen, showSidebarTab, focusPanel, updatePrefs, focus],
  )

  return <LayoutContext.Provider value={value}>{children}</LayoutContext.Provider>
}
