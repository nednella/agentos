import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { api, devFlags, errorMessage, on } from './api'
import { readStored, writeStored } from './storage'
import type { BrowserState, Cleanup, Digest, Evidence, Issue, Note, Project, Session } from './types'

export type Toast = {
  key: number
  tone: 'waiting' | 'finished' | 'pr' | 'evidence' | 'error' | 'info'
  text: string
  sessionId?: string
  view?: SessionView
}

export type SessionView = 'terminal' | 'browser' | 'evidence'
export const SESSION_VIEWS: SessionView[] = ['terminal', 'browser', 'evidence']

export type Overlay = 'palette' | 'projects' | 'shortcuts' | null
export type SidebarTab = 'queue' | 'notes'
export type FocusTarget = 'terminal' | 'command' | 'sidebar' | 'sessions' | 'queue-filter' | 'note-input'
export type PendingImage = { base64: string; mime: string }

type Agentos = {
  project: Project | null
  projects: Project[]
  sessions: Session[]
  selectedId: string | null
  openedIds: string[]
  issues: Issue[]
  issuesLoading: boolean
  issueFilter: string
  notes: Note[]
  cleanups: Cleanup[]
  evidence: Record<string, Evidence[]>
  browserStates: Record<string, BrowserState>
  digest: Digest | null
  digestUnseen: boolean
  toasts: Toast[]
  overlay: Overlay
  sidebarTab: SidebarTab
  composing: boolean
  focusRequest: { target: FocusTarget; n: number }
  select(id: string): void
  stepSession(delta: number): void
  newSession(title?: string): Promise<Session>
  killSession(id: string): Promise<void>
  renameSession(id: string, title: string): Promise<void>
  startIssue(number: number, background?: boolean): Promise<Session>
  switchProject(name: string): Promise<void>
  addProject(dir?: string): Promise<void>
  removeProject(name: string): Promise<void>
  refreshIssues(): Promise<void>
  addNote(text: string, images?: PendingImage[]): Promise<void>
  updateNote(id: string, text: string): Promise<void>
  setNotePinned(id: string, pinned: boolean): Promise<void>
  setNoteArchived(id: string, archived: boolean): Promise<void>
  addNoteImage(id: string, image: PendingImage): Promise<void>
  removeNoteImage(id: string, url: string): Promise<void>
  deleteNote(id: string): Promise<void>
  refreshPRs(): Promise<void>
  viewOf(id: string | null): SessionView
  setSessionView(id: string, view: SessionView): void
  cycleSessionView(delta: -1 | 1): void
  openBrowser(id: string, url?: string): Promise<void>
  harnessCheck(): Promise<void>
  runDigest(): Promise<void>
  digestToNote(itemId: string): Promise<void>
  dismissDigestItem(itemId: string): Promise<void>
  markDigestSeen(): void
  ackPR(id: string): Promise<void>
  openPR(session: Session): Promise<void>
  typeInto(id: string, text: string): Promise<void>
  cleanupSession(id: string, force: boolean): Promise<void>
  noteToIssue(id: string): Promise<void>
  noteToSession(id: string): Promise<void>
  nextAttention(): void
  setSidebarTab(tab: SidebarTab): void
  setOverlay(overlay: Overlay): void
  setComposing(open: boolean): void
  setIssueFilter(query: string): void
  focus(target: FocusTarget): void
  report(run: () => unknown): void
  pushToast(toast: Omit<Toast, 'key'>): void
  dismissToast(key: number): void
}

const AgentosContext = createContext<Agentos | null>(null)

export function useAgentos(): Agentos {
  const value = useContext(AgentosContext)
  if (!value) throw new Error('useAgentos must be used inside AgentosProvider')
  return value
}

const cleanupToast = (entry: Cleanup, merged: boolean) =>
  `${entry.issue > 0 ? `#${entry.issue}` : entry.sessionTitle} ${merged ? 'merged, ' : ''}cleaned up`

const digestKey = (project: string) => `agentos.digestSeen.${project}`

const filterKey = (project: string) => `agentos.filter.${project}`

type AgentosProviderProps = { children: ReactNode }

export function AgentosProvider({ children }: AgentosProviderProps) {
  const [project, setProject] = useState<Project | null>(null)
  const [projects, setProjects] = useState<Project[]>([])
  const [sessions, setSessions] = useState<Session[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [openedIds, setOpenedIds] = useState<string[]>([])
  const [rawIssues, setRawIssues] = useState<Issue[]>([])
  const [issuesLoading, setIssuesLoading] = useState(true)
  const [issueFilter, setIssueFilterState] = useState('')
  const [notes, setNotes] = useState<Note[]>([])
  const [cleanups, setCleanups] = useState<Cleanup[]>([])
  const [evidence, setEvidence] = useState<Record<string, Evidence[]>>({})
  const [browserStates, setBrowserStates] = useState<Record<string, BrowserState>>({})
  const [views, setViews] = useState<Record<string, SessionView>>({})
  const [digest, setDigest] = useState<Digest | null>(null)
  const [digestSeen, setDigestSeen] = useState(0)
  const [toasts, setToasts] = useState<Toast[]>([])
  const [overlay, setOverlay] = useState<Overlay>((devFlags.overlay as Overlay) ?? null)
  const [sidebarTab, setSidebarTab] = useState<SidebarTab>(devFlags.tab === 'notes' ? 'notes' : 'queue')
  const [composing, setComposing] = useState(false)
  const [focusRequest, setFocusRequest] = useState<Agentos['focusRequest']>({ target: 'terminal', n: 0 })

  const sessionsRef = useRef(sessions)
  const selectedRef = useRef(selectedId)
  const projectRef = useRef(project)
  const lastSelected = useRef(new Map<string, string>())
  const toastKey = useRef(0)
  sessionsRef.current = sessions
  projectRef.current = project

  const selectId = useCallback((id: string | null) => {
    selectedRef.current = id
    setSelectedId(id)
    if (id) setOpenedIds((ids) => (ids.includes(id) ? ids : [...ids, id]))
  }, [])

  const focus = useCallback((target: FocusTarget) => setFocusRequest((r) => ({ target, n: r.n + 1 })), [])

  const pushToast = useCallback((toast: Omit<Toast, 'key'>) => {
    const key = ++toastKey.current
    setToasts((list) => [...list.slice(-2), { ...toast, key }])
    const ms = toast.tone === 'error' ? 6000 : 7000
    setTimeout(() => setToasts((list) => list.filter((t) => t.key !== key)), ms)
  }, [])

  const report = useCallback(
    (run: () => unknown) => {
      const fail = (err: unknown) => pushToast({ tone: 'error', text: errorMessage(err) })
      try {
        const result = run()
        if (result instanceof Promise) result.catch(fail)
      } catch (err) {
        fail(err)
      }
    },
    [pushToast],
  )

  const applySessions = useCallback(
    (list: Session[]) => {
      setSessions(list)
      setOpenedIds((ids) => ids.filter((id) => list.some((s) => s.id === id)))
      const current = selectedRef.current
      if (current && list.some((s) => s.id === current)) return
      const remembered = list.find((s) => s.id === lastSelected.current.get(projectRef.current?.name ?? ''))
      const next = remembered ?? list[0]
      selectId(next ? next.id : null)
    },
    [selectId],
  )

  const loadIssues = useCallback(async (refresh: boolean) => {
    setIssuesLoading(true)
    try {
      setRawIssues(await api.issues(refresh))
    } finally {
      setIssuesLoading(false)
    }
  }, [])

  const enterProject = useCallback(
    (snap: { project: Project; projects: Project[]; sessions: Session[]; notes: Note[] }) => {
      if (projectRef.current && selectedRef.current) lastSelected.current.set(projectRef.current.name, selectedRef.current)
      projectRef.current = snap.project
      selectId(null)
      setProject(snap.project)
      setProjects(snap.projects)
      setNotes(snap.notes)
      setRawIssues([])
      setIssueFilterState(readStored(filterKey(snap.project.name), ''))
      report(async () => setCleanups(await api.cleanups()))
      report(async () => setDigest(await api.digest()))
      setDigestSeen(readStored(digestKey(snap.project.name), 0))
      applySessions(snap.sessions)
      report(() => loadIssues(false))
    },
    [applySessions, loadIssues, report, selectId],
  )

  useEffect(() => {
    report(async () => enterProject(await api.snapshot()))
  }, [enterProject, report])

  useEffect(() => on('sessions', applySessions), [applySessions])
  useEffect(() => on('notes', setNotes), [])
  useEffect(() => on('issues', setRawIssues), [])
  useEffect(() => on('evidence', ({ id, items }) => setEvidence((map) => ({ ...map, [id]: items }))), [])
  useEffect(() => on('browser:state', (state) => setBrowserStates((map) => ({ ...map, [state.id]: state }))), [])
  useEffect(() => on('digest', setDigest), [])
  useEffect(
    () =>
      on('cleanups', (list) => {
        setCleanups(list)
        const entry = list[0]
        if (!entry || entry.status !== 'done') return
        const merged = sessionsRef.current.find((s) => s.title === entry.sessionTitle)?.pr?.state === 'merged'
        pushToast({ tone: 'info', text: cleanupToast(entry, merged) })
      }),
    [pushToast],
  )
  useEffect(
    () =>
      on('projects', (list) => {
        setProjects(list)
        setProject((p) => list.find((x) => x.name === p?.name) ?? p)
      }),
    [],
  )

  useEffect(
    () =>
      on('attention', ({ id, state }) => {
        if (id === selectedRef.current) return
        const session = sessionsRef.current.find((s) => s.id === id)
        if (!session) return
        const describe = (current: Session) => {
          if (state === 'waiting') return 'needs you'
          if (state === 'finished') return 'finished'
          if (state === 'evidence') return 'has something to show'
          return current.prAttention === 'checks' ? 'has failing checks' : 'has new review comments'
        }
        // The sessions event that carries prAttention lands in the same tick as this one.
        setTimeout(() => {
          const current = sessionsRef.current.find((s) => s.id === id) ?? session
          pushToast({ tone: state, text: `${current.title} ${describe(current)}`, sessionId: id, view: state === 'evidence' ? 'evidence' : undefined })
        }, 50)
      }),
    [pushToast],
  )

  useEffect(() => {
    if (!selectedId || evidence[selectedId]) return
    report(async () => {
      const items = await api.evidence(selectedId)
      setEvidence((map) => (map[selectedId] ? map : { ...map, [selectedId]: items }))
    })
  }, [selectedId, evidence, report])

  useEffect(() => {
    const target = sessions.find((s) => s.id === selectedId)
    if (!target?.browser || browserStates[target.id]) return
    report(async () => {
      const state = await api.browserState(target.id)
      setBrowserStates((map) => (map[target.id] ? map : { ...map, [target.id]: state }))
    })
  }, [sessions, selectedId, browserStates, report])

  const digestUnseen = digest?.items.some((i) => !i.noteId && i.at > digestSeen) ?? false

  const issues = useMemo(
    () => rawIssues.map((issue) => ({ ...issue, sessionId: sessions.find((s) => s.issue === issue.number)?.id ?? '' })),
    [rawIssues, sessions],
  )

  const addSession = useCallback(
    (created: Session, background = false) => {
      setSessions((list) => (list.some((s) => s.id === created.id) ? list : [...list, created]))
      if (!background) selectId(created.id)
      return created
    },
    [selectId],
  )

  const value = useMemo<Agentos>(
    () => ({
      project,
      projects,
      sessions,
      selectedId,
      openedIds,
      issues,
      issuesLoading,
      issueFilter,
      notes,
      cleanups,
      evidence,
      browserStates,
      digest,
      digestUnseen,
      toasts,
      overlay,
      sidebarTab,
      composing,
      focusRequest,
      select: selectId,
      stepSession(delta) {
        const list = sessionsRef.current
        if (list.length === 0) return
        const at = list.findIndex((s) => s.id === selectedRef.current)
        selectId(list[(at + delta + list.length) % list.length].id)
      },
      async newSession(title = '') {
        return addSession(await api.newSession(title, ''))
      },
      async killSession(id) {
        await api.killSession(id)
        applySessions(sessionsRef.current.filter((s) => s.id !== id))
      },
      async renameSession(id, title) {
        const trimmed = title.trim()
        if (!trimmed) throw 'A title cannot be empty'
        await api.renameSession(id, trimmed)
        setSessions((list) => list.map((s) => (s.id === id ? { ...s, title: trimmed } : s)))
      },
      async startIssue(number, background = false) {
        return addSession(await api.startIssue(number), background)
      },
      async switchProject(name) {
        if (name === projectRef.current?.name) return
        enterProject(await api.switchProject(name))
      },
      async addProject(dir) {
        enterProject(await (dir ? api.addProjectDir(dir) : api.addProject()))
      },
      async removeProject(name) {
        const snap = await api.removeProject(name)
        if (snap.project.name === projectRef.current?.name) setProjects(snap.projects)
        else enterProject(snap)
      },
      refreshIssues: () => loadIssues(true),
      viewOf: (id) => (id && views[id]) || 'terminal',
      setSessionView(id, view) {
        setViews((map) => ({ ...map, [id]: view }))
        if (view === 'terminal') focus('terminal')
      },
      cycleSessionView(delta) {
        const id = selectedRef.current
        if (!id) return
        const at = SESSION_VIEWS.indexOf(views[id] ?? 'terminal')
        const next = SESSION_VIEWS[(at + delta + SESSION_VIEWS.length) % SESSION_VIEWS.length]
        setViews((map) => ({ ...map, [id]: next }))
        if (next === 'terminal') focus('terminal')
      },
      async openBrowser(id, url) {
        setViews((map) => ({ ...map, [id]: 'browser' }))
        const state = browserStates[id] ?? (await api.browserState(id))
        if (!state.open) {
          const opened = await api.browserOpen(id, url ?? '')
          setBrowserStates((map) => ({ ...map, [id]: opened }))
        } else if (url) {
          await api.browserGoto(id, url)
        }
      },
      async harnessCheck() {
        addSession(await api.harnessCheck())
      },
      async runDigest() {
        await api.runDigest()
      },
      async digestToNote(itemId) {
        await api.digestToNote(itemId)
      },
      async dismissDigestItem(itemId) {
        await api.dismissDigestItem(itemId)
      },
      markDigestSeen() {
        const now = Date.now()
        setDigestSeen(now)
        if (projectRef.current) writeStored(digestKey(projectRef.current.name), now)
      },
      async addNote(text, images = []) {
        const trimmed = text.trim()
        if (!trimmed && images.length === 0) throw 'A note cannot be empty'
        let note: Note
        try {
          note = await api.addNote(trimmed)
        } catch (err) {
          if (trimmed) throw err
          note = await api.addNote('(screenshot)')
        }
        for (const image of images) note = await api.addNoteImage(note.id, image.base64, image.mime)
        const saved = note
        setNotes((list) => (list.some((n) => n.id === saved.id) ? list.map((n) => (n.id === saved.id ? saved : n)) : [saved, ...list]))
      },
      async updateNote(id, text) {
        const trimmed = text.trim()
        if (!trimmed && !notes.find((n) => n.id === id)?.images.length) throw 'A note cannot be empty'
        const note = await api.updateNote(id, trimmed)
        setNotes((list) => list.map((n) => (n.id === id ? note : n)))
      },
      async setNotePinned(id, pinned) {
        const note = await api.setNotePinned(id, pinned)
        setNotes((list) => list.map((n) => (n.id === id ? note : n)))
      },
      async setNoteArchived(id, archived) {
        const note = await api.setNoteArchived(id, archived)
        setNotes((list) => list.map((n) => (n.id === id ? note : n)))
      },
      async addNoteImage(id, image) {
        const note = await api.addNoteImage(id, image.base64, image.mime)
        setNotes((list) => list.map((n) => (n.id === id ? note : n)))
      },
      async removeNoteImage(id, url) {
        const note = await api.removeNoteImage(id, url)
        setNotes((list) => list.map((n) => (n.id === id ? note : n)))
      },
      async refreshPRs() {
        await api.refreshPRs()
      },
      async ackPR(id) {
        await api.ackPR(id)
      },
      async openPR(session) {
        if (!session.pr) throw 'This session has no pull request yet'
        await api.openURL(session.pr.url)
        if (session.prAttention) await api.ackPR(session.id)
      },
      async typeInto(id, text) {
        await api.typeInto(id, text)
      },
      async cleanupSession(id, force) {
        await api.cleanup(id, force)
      },
      async deleteNote(id) {
        await api.deleteNote(id)
        setNotes((list) => list.filter((n) => n.id !== id))
      },
      async noteToIssue(id) {
        const note = await api.noteToIssue(id)
        setNotes((list) => list.map((n) => (n.id === id ? note : n)))
      },
      async noteToSession(id) {
        addSession(await api.noteToSession(id))
      },
      nextAttention() {
        const list = sessionsRef.current
        const urgent = [...list.filter((s) => s.state === 'waiting'), ...list.filter((s) => s.state === 'finished')]
        if (urgent.length === 0) throw 'Nothing needs you right now'
        const at = urgent.findIndex((s) => s.id === selectedRef.current)
        selectId(urgent[(at + 1) % urgent.length].id)
      },
      setSidebarTab,
      setOverlay,
      setComposing,
      setIssueFilter(query) {
        setIssueFilterState(query)
        if (projectRef.current) writeStored(filterKey(projectRef.current.name), query)
      },
      focus,
      report,
      pushToast,
      dismissToast: (key) => setToasts((list) => list.filter((t) => t.key !== key)),
    }),
    [project, projects, sessions, selectedId, openedIds, issues, issuesLoading, issueFilter, notes, cleanups, evidence, browserStates, digest, digestUnseen, views, toasts, overlay, sidebarTab, composing, focusRequest, selectId, addSession, applySessions, enterProject, loadIssues, focus, report, pushToast],
  )

  return <AgentosContext.Provider value={value}>{children}</AgentosContext.Provider>
}
