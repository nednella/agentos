import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { api, devFlags, errorMessage, on } from './api'
import { readStored, writeStored } from './storage'
import type { Issue, Note, Project, Session } from './types'

export type Overlay = 'palette' | 'projects' | 'shortcuts' | null
export type SidebarTab = 'queue' | 'notes'
export type FocusTarget = 'terminal' | 'shell' | 'sidebar' | 'sessions' | 'queue-filter' | 'note-input'
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
  overlay: Overlay
  sidebarTab: SidebarTab
  composing: boolean
  focusRequest: { target: FocusTarget; n: number }
  select(id: string): void
  stepSession(delta: number): void
  newSession(title?: string): Promise<Session>
  killSession(id: string): Promise<void>
  dismissSession(id: string): Promise<void>
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
  noteToIssue(id: string): Promise<void>
  noteToSession(id: string): Promise<void>
  nextAttention(): void
  setSidebarTab(tab: SidebarTab): void
  setOverlay(overlay: Overlay): void
  setComposing(open: boolean): void
  setIssueFilter(query: string): void
  focus(target: FocusTarget): void
  report(run: () => unknown): void
}

const AgentosContext = createContext<Agentos | null>(null)

export function useAgentos(): Agentos {
  const value = useContext(AgentosContext)
  if (!value) throw new Error('useAgentos must be used inside AgentosProvider')
  return value
}

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
  const [overlay, setOverlay] = useState<Overlay>((devFlags.overlay as Overlay) ?? null)
  const [sidebarTab, setSidebarTab] = useState<SidebarTab>(devFlags.tab === 'notes' ? 'notes' : 'queue')
  const [composing, setComposing] = useState(false)
  const [focusRequest, setFocusRequest] = useState<Agentos['focusRequest']>({ target: 'terminal', n: 0 })

  const sessionsRef = useRef(sessions)
  const selectedRef = useRef(selectedId)
  const projectRef = useRef(project)
  const lastSelected = useRef(new Map<string, string>())
  sessionsRef.current = sessions
  projectRef.current = project

  const selectId = useCallback((id: string | null) => {
    selectedRef.current = id
    setSelectedId(id)
    if (id) setOpenedIds((ids) => (ids.includes(id) ? ids : [...ids, id]))
  }, [])

  const focus = useCallback((target: FocusTarget) => setFocusRequest((r) => ({ target, n: r.n + 1 })), [])

  const report = useCallback(
    (run: () => unknown) => {
      const fail = (err: unknown) => console.error(errorMessage(err))
      try {
        const result = run()
        if (result instanceof Promise) result.catch(fail)
      } catch (err) {
        fail(err)
      }
    },
    [],
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
    (snap: { project: Project; projects: Project[]; sessions: Session[]; notes: Note[]; shell: string }) => {
      if (projectRef.current && selectedRef.current) lastSelected.current.set(projectRef.current.name, selectedRef.current)
      projectRef.current = snap.project
      selectId(null)
      setProject(snap.project)
      setProjects(snap.projects)
      setNotes(snap.notes)
      setRawIssues([])
      setIssueFilterState(readStored(filterKey(snap.project.name), ''))
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
  useEffect(
    () =>
      on('projects', (list) => {
        setProjects(list)
        setProject((p) => list.find((x) => x.name === p?.name) ?? p)
      }),
    [],
  )

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
      async dismissSession(id) {
        await api.dismissSession(id)
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
        const urgent = [...list.filter((s) => s.state === 'waiting'), ...list.filter((s) => s.state === 'idle')]
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
    }),
    [project, projects, sessions, selectedId, openedIds, issues, issuesLoading, issueFilter, notes, overlay, sidebarTab, composing, focusRequest, selectId, addSession, applySessions, enterProject, loadIssues, focus, report],
  )

  return <AgentosContext.Provider value={value}>{children}</AgentosContext.Provider>
}
