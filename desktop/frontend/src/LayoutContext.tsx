import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { useAgentos } from './AgentosContext'
import type { SidebarTab } from './AgentosContext'
import { devFlags } from './api'
import { readStored, writeStored } from './storage'

export type Theme = 'dark' | 'light'
const DARK_QUERY = '(prefers-color-scheme: dark)'
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
const SHELL_ROW_REM = 1.0325
const SHELL_PAD_REM = 0.5
export const shellRemForRows = (rows: number) => rows * SHELL_ROW_REM + SHELL_PAD_REM
const SHELL_DEFAULT_REM = shellRemForRows(6)
const SHELL_MIN_REM = shellRemForRows(3)

type BandPrefs = { sidebarOpen: boolean; sessionsOpen: boolean; shellOpen: boolean; sidebar: number | null; sessions: number | null; shell: number | null }
type AllPrefs = Record<Band, BandPrefs>

const fresh = (band: Band): BandPrefs => ({ sidebarOpen: true, sessionsOpen: band !== 'compact', shellOpen: true, sidebar: null, sessions: null, shell: null })

export type NewsPage = 'index' | 'tldr' | 'digest'

export type Layout = {
  mode: LayoutMode
  width: number
  scale: number
  theme: Theme
  sidebarOpen: boolean
  sessionsOpen: boolean
  shellOpen: boolean
  shellRem: number
  sidebarPeek: boolean
  sessionsPeek: boolean
  mobilePanel: MobilePanel
  widths: Record<SidePanel, number>
  peekWidths: Record<SidePanel, number>
  statsOpen: boolean
  newsOpen: boolean
  newsPage: NewsPage
  zoom(step: -1 | 0 | 1): void
  setMobilePanel(panel: MobilePanel): void
  setWidth(panel: SidePanel, rem: number, commit: boolean): void
  resetWidth(panel: SidePanel): void
  setShellOpen(open: boolean): void
  setShellHeight(rem: number, commit: boolean): void
  resetShellHeight(): void
  toggleSidebar(): void
  toggleSessions(): void
  setSidebarOpen(open: boolean): void
  setSessionsOpen(open: boolean): void
  showSidebarTab(tab: SidebarTab): void
  toggleStats(): void
  toggleDigest(): void
  toggleNews(): void
  openNews(page: NewsPage): void
  closeNewsPage(): void
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

function useWindowSize(): { w: number; h: number } {
  const [size, setSize] = useState({ w: window.innerWidth, h: window.innerHeight })
  useEffect(() => {
    const update = () => setSize({ w: window.innerWidth, h: window.innerHeight })
    window.addEventListener('resize', update)
    return () => window.removeEventListener('resize', update)
  }, [])
  return size
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

function useSystemTheme(): Theme {
  const [theme, setTheme] = useState<Theme>(() => (matchMedia(DARK_QUERY).matches ? 'dark' : 'light'))
  useEffect(() => {
    const query = matchMedia(DARK_QUERY)
    const onChange = (e: MediaQueryListEvent) => setTheme(e.matches ? 'dark' : 'light')
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  }, [])
  return theme
}

export function LayoutProvider({ children }: LayoutProviderProps) {
  const { selectedId, settings, setTextScale, sidebarTab, setSidebarTab, focus, report, pushToast, dismissToast } = useAgentos()
  const { w: width, h: height } = useWindowSize()
  const mode = modeFor(width)
  const band: Band = mode === 'narrow' ? 'compact' : mode
  const scale = settings.textScale
  const systemTheme = useSystemTheme()
  const theme = settings.theme === 'system' ? systemTheme : settings.theme
  const [stored, setStored] = useState(() => readStored<Partial<AllPrefs>>(LAYOUT_KEY, {}))
  const [peek, setPeek] = useState<SidePanel | null>(null)
  const [mobilePanel, setMobilePanel] = useState<MobilePanel>('session')
  const [statsOpen, setStatsOpen] = useState(devFlags.view === 'stats')
  const [newsOpen, setNewsOpen] = useState(devFlags.view === 'news' || devFlags.view === 'digest')
  const [newsPage, setNewsPage] = useState<NewsPage>(devFlags.view === 'digest' ? 'digest' : 'index')
  const zoomToast = useRef<number | null>(null)

  useLayoutEffect(() => {
    document.documentElement.style.setProperty('--ui-scale', String(scale))
  }, [scale])

  useLayoutEffect(() => {
    document.documentElement.dataset.theme = theme
  }, [theme])

  const prefs: BandPrefs = { ...fresh(band), ...stored[band] }
  const { widths: fittedWidths, full: fullWidths, sessionsForcedClosed } = fit(width, prefs, 16 * scale)
  const narrow = mode === 'narrow'
  const sidebarMobile = mobilePanel === 'queue' || mobilePanel === 'notes'
  const sidebarOpen = narrow ? sidebarMobile : prefs.sidebarOpen
  const sessionsOpen = narrow ? mobilePanel === 'sessions' : prefs.sessionsOpen && !sessionsForcedClosed

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
    setNewsOpen(false)
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
        if (panel === 'shell') setMobilePanel('shell')
      } else if (panel === 'sidebar') setPeek(prefs.sidebarOpen ? null : 'sidebar')
      else if (panel === 'sessions') setPeek(sessionsOpen ? null : 'sessions')
      else {
        setPeek(null)
        if (panel === 'shell') updatePrefs({ shellOpen: true })
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
      theme,
      sidebarOpen,
      sessionsOpen,
      shellOpen: narrow ? mobilePanel === 'shell' : prefs.shellOpen,
      shellRem: Math.min(Math.max(prefs.shell ?? SHELL_DEFAULT_REM, SHELL_MIN_REM), (0.6 * height) / (16 * scale)),
      sidebarPeek: !narrow && !prefs.sidebarOpen && peek === 'sidebar',
      sessionsPeek: !narrow && !sessionsOpen && peek === 'sessions',
      mobilePanel,
      widths: { sidebar: fittedWidths.sidebar, sessions: fittedWidths.sessions },
      peekWidths: {
        sidebar: Math.min(Math.max(fullWidths.sidebar, 20), (0.8 * width) / (16 * scale)),
        sessions: Math.min(Math.max(fullWidths.sessions, 18), (0.8 * width) / (16 * scale)),
      },
      statsOpen,
      newsOpen,
      newsPage,
      zoom(step) {
        const at = SCALES.indexOf(scale)
        const next = step === 0 ? DEFAULT_SCALE : SCALES[Math.min(Math.max((at < 0 ? 2 : at) + step, 0), SCALES.length - 1)]
        report(() => setTextScale(next))
        if (zoomToast.current !== null) dismissToast(zoomToast.current)
        zoomToast.current = pushToast({ tone: 'info', text: `Text size ${Math.round(next * 100)}%` })
      },
      setMobilePanel,
      setWidth(panel, rem, commit) {
        const next = Math.min(Math.max(rem, MIN_REM[panel]), MAX_REM)
        if (commit) updatePrefs({ [panel]: next })
        else setStored((all) => ({ ...all, [band]: { ...fresh(band), ...all[band], [panel]: next } }))
      },
      resetWidth: (panel) => updatePrefs({ [panel]: null }),
      setShellOpen(open) {
        if (narrow) setMobilePanel(open ? 'shell' : 'session')
        else updatePrefs({ shellOpen: open })
      },
      setShellHeight(rem, commit) {
        const next = Math.max(rem, SHELL_MIN_REM)
        if (commit) updatePrefs({ shell: next })
        else setStored((all) => ({ ...all, [band]: { ...fresh(band), ...all[band], shell: next } }))
      },
      resetShellHeight: () => updatePrefs({ shell: null }),
      toggleSidebar: () => setSidebarOpen(!sidebarOpen),
      toggleSessions: () => setSessionsOpen(!sessionsOpen),
      setSidebarOpen,
      setSessionsOpen,
      showSidebarTab,
      toggleStats() {
        setStatsOpen(!statsOpen)
        setNewsOpen(false)
        setPeek(null)
        if (!statsOpen && narrow) setMobilePanel('session')
      },
      toggleNews() {
        const close = newsOpen && newsPage === 'index'
        setNewsPage('index')
        setNewsOpen(!close)
        setStatsOpen(false)
        setPeek(null)
        if (!close && narrow) setMobilePanel('session')
      },
      toggleDigest() {
        const close = newsOpen && newsPage === 'digest'
        setNewsPage('digest')
        setNewsOpen(!close)
        setStatsOpen(false)
        setPeek(null)
        if (!close && narrow) setMobilePanel('session')
      },
      openNews(page) {
        setNewsPage(page)
        setNewsOpen(true)
        setStatsOpen(false)
        setPeek(null)
        if (narrow) setMobilePanel('session')
      },
      closeNewsPage: () => setNewsPage('index'),
      closeCentre() {
        setStatsOpen(false)
        setNewsOpen(false)
      },
      closePeek: () => setPeek(null),
      returnToTerminal() {
        setPeek(null)
        focus('terminal')
      },
      focusPanel,
    }),
    [mode, width, scale, theme, sidebarOpen, sessionsOpen, narrow, prefs.sidebarOpen, prefs.shellOpen, prefs.shell, peek, mobilePanel, fittedWidths.sidebar, fittedWidths.sessions, fullWidths.sidebar, fullWidths.sessions, height, statsOpen, newsOpen, newsPage, band, setSidebarOpen, setSessionsOpen, showSidebarTab, focusPanel, updatePrefs, focus, report, setTextScale],
  )

  return <LayoutContext.Provider value={value}>{children}</LayoutContext.Provider>
}
