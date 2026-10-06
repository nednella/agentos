export type State = 'waiting' | 'working' | 'idle' | 'ended'

export type HistoryEntry = { state: State; at: number }

export type PR = {
  number: number
  url: string
  state: 'draft' | 'open' | 'merged' | 'closed'
  checks: 'none' | 'pending' | 'passing' | 'failing'
  comments: number
  updatedAt: number
}

export type Session = {
  id: string
  n: number
  title: string
  state: State
  detail: string
  lastEventAt: number
  createdAt: number
  issue: number
  history: HistoryEntry[]
  branch: string
  worktree: string
  pr: PR | null
  prAttention: '' | 'checks' | 'comments'
  cleanup: '' | 'pending' | 'blocked' | 'ask'
  cleanupReason: string
  browser: boolean
  evidence: number
}

export type Project = {
  key: string
  name: string
  dir: string
  repo: string
  needsYou: number
  working: number
  sessions: number
}

export type Note = {
  id: string
  text: string
  createdAt: number
  updatedAt: number
  pinned: boolean
  archived: boolean
  images: string[]
}

export type IssueType = 'bug' | 'feature' | 'refactor' | 'chore' | ''

export type Lane = 'ready' | 'plan' | 'you' | 'idea' | 'inbox'

export type Issue = {
  number: number
  title: string
  type: IssueType
  lane: Lane
  url: string
  sessionId: string
  author: string
  assignees: string[]
  labels: string[]
  createdAt: number
  updatedAt: number
}

export type IssueComment = {
  author: string
  createdAt: number
  bodyHTML: string
}

export type IssueDetail = {
  number: number
  bodyHTML: string
  comments: IssueComment[]
}

export type Snapshot = {
  project: Project
  projects: Project[]
  sessions: Session[]
  notes: Note[]
  version: string
  shell: string
}

export type WaitKind = 'permission' | 'question' | 'idle'

export type Wait = {
  sessionTitle: string
  issue: number
  kind: WaitKind
  label: string
  startedAt: number
  waitedMs: number
}

export type Stats = {
  days: number
  total: number
  totalWaitMs: number
  medianWaitMs: number
  byCause: { kind: WaitKind; label: string; count: number; totalWaitMs: number }[]
  byDay: { day: string; count: number }[]
  recent: Wait[]
}

export type Cleanup = {
  at: number
  sessionTitle: string
  issue: number
  pr: number
  merged: boolean
  status: 'done' | 'blocked'
  removed: string[]
  reason: string
}

export type BrowserState = {
  id: string
  open: boolean
  url: string
  title: string
  loading: boolean
  canGoBack: boolean
  canGoForward: boolean
  error: string
}

export type BrowserInput =
  | { type: 'mouse'; action: 'move' | 'down' | 'up'; x: number; y: number; button: 'left' | 'middle' | 'right' | 'none'; clickCount: number; modifiers: number }
  | { type: 'wheel'; x: number; y: number; deltaX: number; deltaY: number; modifiers: number }
  | { type: 'key'; action: 'down' | 'up'; key: string; code: string; text: string; modifiers: number }
  | { type: 'paste'; text: string }

export type Evidence = {
  id: string
  kind: 'image' | 'text'
  url: string
  text: string
  caption: string
  source: 'agent' | 'user'
  at: number
}

export type DigestItem = {
  id: string
  title: string
  why: string
  url: string
  source: string
  at: number
  noteId: string
}

export type Digest = {
  running: boolean
  lastRunAt: number
  nextRunAt: number
  error: string
  project: string
  items: DigestItem[]
}

export type ProjectList<T> = { project: string; items: T[] }

export type Warning = {
  source: 'tmux' | 'github' | 'pull requests' | 'worktrees'
  message: string
}

export type EventMap = {
  sessions: ProjectList<Session>
  projects: Project[]
  notes: ProjectList<Note>
  issues: ProjectList<Issue>
  cleanups: ProjectList<Cleanup>
  warnings: Warning
  stats: undefined
  'ui:command': { name: string; args: string[] }
  evidence: { id: string; items: Evidence[] }
  digest: Digest
  'browser:frame': { id: string; data: string; width: number; height: number }
  'browser:state': BrowserState
  'term:data': { id: string; data: string }
  'term:exit': { id: string }
  attention: { id: string; state: 'waiting' | 'replied' | 'opened' | 'pr' | 'evidence' }
}
