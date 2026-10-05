import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { api, devFlags, errorMessage, on } from './api'
import { readStored, writeStored } from './storage'
import type { Issue, Note, Project, Session } from './types'

export type Overlay = 'palette' | 'projects' | 'shortcuts' | null
export type SidebarTab = 'queue' | 'notes'
export type FocusTarget = 'terminal' | 'shell' | 'sidebar' | 'sessions' | 'queue-filter' | 'note-input'

type Agentos = {
  project: Project | null
  projects: Project[]
  sessions: Session[]
  selectedId: string | null
  issues: Issue[]
  issuesLoading: boolean
  issueFilter: string
  notes: Note[]
  overlay: Overlay
  sidebarTab: SidebarTab
  focusRequest: { target: FocusTarget; n: number }
  select(id: string): void
  startIssue(number: number, background?: boolean): Promise<Session>
  switchProject(name: string): Promise<void>
  addProject(dir?: string): Promise<void>
  removeProject(name: string): Promise<void>
  refreshIssues(): Promise<void>
  setSidebarTab(tab: SidebarTab): void
  setOverlay(overlay: Overlay): void
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
  const [rawIssues, setRawIssues] = useState<Issue[]>([])
  const [issuesLoading, setIssuesLoading] = useState(true)
  const [issueFilter, setIssueFilterState] = useState('')
  const [notes, setNotes] = useState<Note[]>([])
  const [overlay, setOverlay] = useState<Overlay>((devFlags.overlay as Overlay) ?? null)
  const [sidebarTab, setSidebarTab] = useState<SidebarTab>(devFlags.tab === 'notes' ? 'notes' : 'queue')
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
      issues,
      issuesLoading,
      issueFilter,
      notes,
      overlay,
      sidebarTab,
      focusRequest,
      select: selectId,
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
      setSidebarTab,
      setOverlay,
      setIssueFilter(query) {
        setIssueFilterState(query)
        if (projectRef.current) writeStored(filterKey(projectRef.current.name), query)
      },
      focus,
      report,
    }),
    [project, projects, sessions, selectedId, issues, issuesLoading, issueFilter, notes, overlay, sidebarTab, focusRequest, selectId, addSession, applySessions, enterProject, loadIssues, focus, report],
  )

  return <AgentosContext.Provider value={value}>{children}</AgentosContext.Provider>
}
