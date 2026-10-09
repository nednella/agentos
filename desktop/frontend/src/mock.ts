import { buildIssueDetail, buildIssues, movesFrom, moveIssue, sortBySection } from './mockIssues'
import { clickAt, newPage, normalizeUrl, pageRects, pageTitle, renderPage, typeText } from './mockBrowser'
import type { PageModel } from './mockBrowser'
import * as term from './mockTerminal'
import type { BrowserState, Cleanup, CleanupEvent, CleanupMode, Digest, DigestItem, DigestSchedule, EventMap, Evidence, HistoryEntry, Issue, Ledger, LedgerRow, Note, PR, Project, Release, Session, PromptSend, Settings, Snapshot, State, Stats, ThemeSetting, Wait, WaitKind } from './types'

type Handler = (payload: never) => void

type MockSession = Session & {
  project: string
  buffer: string
  opened: boolean
  nextChangeAt: number
  stream: number
  cursor: number
  tools: number
}

type ProjectData = { name: string; dir: string; repo: string; notes: Note[]; issues: Issue[]; waits: Wait[]; cleanups: Cleanup[]; needsSetup?: boolean; setupDismissed?: boolean }

const minute = 60_000

// A project that is not set up has no queue sections, so every issue sits in the nameless group.
const unsorted = (issues: Issue[]) => issues.map((issue) => ({ ...issue, section: '', actions: ['Start'], moves: [] }))
const noteTexts = [
  'Retry banner when the websocket drops\nShow a thin bar under the header and retry with backoff. Maybe reuse the toast style so it does not feel like a third kind of notice.\nThe backoff should reset on the first successful frame, not on connect, because the proxy accepts and then drops.',
  'Ask Alex about the checkout button copy. They said something about "Buy now" vs "Pay" last week and I forgot which way it went.',
  'Try a keyboard shortcut sheet in the shop, like the one here. `?` opens it, Esc closes. See https://github.com/acme/storefront/issues/436 for the dark mode thread, same overlay could host it.',
  'Shipping: should the postcode match be case-insensitive?\nCheck how the address lookup returns UK postcodes. I think it keeps the space but upper-cases the letters, which is fine, but older orders might differ.',
  'Idea: weekly sales as a sparkline in the stats table',
  'Rename Catalogue to Shop everywhere in the copy. Check the emails too, the order template still says Catalogue.',
  'Write the migration note for the price table: old flat map goes away, priceFor(currency, region). Call out that the payment provider only needs currency + amount.',
  'Check the image upload size limit (10MB on the CDN)',
]

const shot = (hue: number, label: string) =>
  `data:image/svg+xml;utf8,${encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="200" viewBox="0 0 320 200"><rect width="320" height="200" fill="hsl(${hue} 30% 16%)"/><rect x="16" y="16" width="288" height="22" fill="hsl(${hue} 40% 26%)"/><rect x="16" y="52" width="180" height="12" fill="hsl(${hue} 50% 40%)"/><rect x="16" y="76" width="260" height="8" fill="hsl(${hue} 20% 30%)"/><rect x="16" y="94" width="240" height="8" fill="hsl(${hue} 20% 30%)"/><rect x="16" y="112" width="200" height="8" fill="hsl(${hue} 20% 30%)"/><text x="16" y="178" fill="hsl(${hue} 60% 75%)" font-family="monospace" font-size="14">${label}</text></svg>`,
  )}`

function seedNotes(texts: string[]): Note[] {
  const now = Date.now()
  return texts.map((text, i) => ({
    id: `note-${Math.random().toString(36).slice(2, 8)}`,
    text,
    createdAt: now - (i + 1) * 37 * minute,
    updatedAt: now - (i + 1) * 37 * minute,
    pinned: i === 2,
    archived: i >= texts.length - 2,
    images: i === 0 ? [shot(210, 'before: banner overlaps'), shot(150, 'after: thin bar')] : i === 3 ? [shot(30, 'postcode lookup')] : [],
  }))
}

const causes: { kind: WaitKind; label: string; weight: number; waitMin: number }[] = [
  { kind: 'permission', label: 'Bash: npm run test', weight: 18, waitMin: 2 },
  { kind: 'permission', label: 'Bash: npx tsc --noEmit', weight: 14, waitMin: 1 },
  { kind: 'permission', label: 'Edit', weight: 12, waitMin: 1 },
  { kind: 'permission', label: 'Bash: gh pr create', weight: 6, waitMin: 3 },
  { kind: 'permission', label: 'Read outside project', weight: 5, waitMin: 2 },
  { kind: 'question', label: 'Question', weight: 10, waitMin: 7 },
  { kind: 'idle', label: 'Reply landed', weight: 16, waitMin: 11 },
]

function seedWaits(titles: [string, number][]): Wait[] {
  const now = Date.now()
  let seedValue = 7
  const random = () => {
    seedValue = (seedValue * 16807) % 2147483647
    return seedValue / 2147483647
  }
  const total = causes.reduce((sum, c) => sum + c.weight, 0)
  const waits: Wait[] = []
  for (let d = 14; d >= 0; d--) {
    const weekend = new Date(now - d * 86_400_000).getDay() % 6 === 0
    const count = Math.round((weekend ? 2 : 7) + random() * 6)
    for (let i = 0; i < count; i++) {
      let roll = random() * total
      const cause = causes.find((c) => (roll -= c.weight) < 0) ?? causes[0]
      const [sessionTitle, issue] = titles[Math.floor(random() * titles.length)]
      waits.push({
        sessionTitle,
        issue,
        model: 'sonnet',
        kind: cause.kind,
        label: cause.label,
        startedAt: now - d * 86_400_000 - Math.floor(random() * 9 * 3_600_000),
        waitedMs: Math.round((cause.waitMin * (0.4 + random() * 1.6)) * minute),
      })
    }
  }
  return waits.sort((a, b) => b.startedAt - a.startedAt)
}

function pr(number: number, state: PR['state'], checks: PR['checks'], comments = 0): PR {
  return { number, url: `https://github.com/acme/storefront/pull/${number}`, state, checks, comments, updatedAt: Date.now() - 3 * minute }
}

const details: Record<State, string[]> = {
  working: [
    'Edit route-list.tsx',
    'Bash npx tsc --noEmit',
    'Read Layout.tsx',
    'Grep productContext',
    'Edit Index.tsx',
    'Write ProductGallery.tsx',
  ],
  waiting: ['Claude needs your permission', 'Claude has a question for you'],
  idle: ['Done: 4 files changed', 'Done: tests pass'],
  ended: ['Session ended'],
}

const order: Record<State, number> = { waiting: 0, idle: 1, working: 2, ended: 3 }

const pick = <T,>(items: T[]): T => items[Math.floor(Math.random() * items.length)]

function toBase64(text: string): string {
  const bytes = new TextEncoder().encode(text)
  let binary = ''
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000))
  }
  return btoa(binary)
}

const PATCH_NOTES: Release[] = [
  {
    version: '0.14.0',
    date: '2026-10-07',
    sections: [
      { title: 'Features', changes: [{ scope: 'desktop', text: 'let agents close their browser window', issue: 218, url: 'https://github.com/nednella/agentos/issues/218' }] },
      {
        title: 'Bug Fixes',
        changes: [
          { scope: 'desktop', text: 'forget browser whose connection dropped', issue: 218, url: 'https://github.com/nednella/agentos/issues/218' },
          { scope: 'desktop', text: "start browser without last run's pages", issue: 218, url: 'https://github.com/nednella/agentos/issues/218' },
          { scope: 'desktop', text: 'stop browser once its last window closes', issue: 218, url: 'https://github.com/nednella/agentos/issues/218' },
        ],
      },
    ],
  },
  {
    version: '0.13.0',
    date: '2026-10-07',
    sections: [
      {
        title: 'Features',
        changes: [
          { scope: 'cli', text: 'add --prompt to agentos new', issue: 0, url: '' },
          { scope: 'desktop', text: 'tell claude sessions how to start a session', issue: 0, url: '' },
        ],
      },
    ],
  },
  {
    version: '0.12.0',
    date: '2026-10-07',
    sections: [
      { title: 'Features', changes: [{ scope: 'build', text: 'draw terminal prompt app icon', issue: 0, url: '' }] },
      { title: 'Bug Fixes', changes: [{ scope: 'ui', text: 'pad top of queue list', issue: 0, url: '' }] },
      { title: 'Performance Improvements', changes: [{ scope: 'desktop', text: 'start chrome-devtools-mcp straight from the npx cache', issue: 0, url: '' }] },
    ],
  },
]

export function createMock(params: URLSearchParams) {
  const flags = {
    empty: params.has('empty'),
    overlay: params.get('overlay') ?? undefined,
    tab: params.get('tab') ?? undefined,
    cmd: params.get('cmd') ?? undefined,
    view: params.get('view') ?? undefined,
    toast: params.has('toast'),
    done: params.has('done'),
    warn: params.has('warn'),
    update: params.has('update'),
    patchNotes: params.has('patchnotes'),
    fullDataDir: params.has('fulldata'),
  }
  const handlers = new Map<string, Set<Handler>>()
  const sessions: MockSession[] = []
  const data: ProjectData[] = [
    { name: 'storefront', dir: '~/code/acme/storefront', repo: 'acme/storefront', notes: seedNotes(noteTexts), issues: buildIssues('acme/storefront'), waits: seedWaits([['#454 product page split', 454], ['#444 checkout route', 444], ['#326 cart rounding', 326], ['#389 webhook retries', 389], ['checkout tests scratch', 0]]), cleanups: [{ at: Date.now() - 5 * 3_600_000, sessionTitle: '#371 old banner', issue: 371, pr: 440, merged: true, status: 'done', removed: ['worktree trees/issue-371', 'branch issue-371', 'temp files', 'session'], reason: '' }, { at: Date.now() - 26 * 3_600_000, sessionTitle: '#366 retry copy', issue: 366, pr: 431, merged: true, status: 'done', removed: ['worktree trees/issue-366', 'branch issue-366', 'session'], reason: '' }] },
    { name: 'agentos', dir: '~/code/agentos', repo: 'nednella/agentos', notes: seedNotes(noteTexts.slice(0, 3)), issues: unsorted(buildIssues('nednella/agentos')), waits: seedWaits([['#12 session memory', 12]]).slice(0, 12), cleanups: [], needsSetup: true },
    { name: 'scratch', dir: '~/scratch', repo: '', notes: [], issues: [], waits: [], cleanups: [] },
  ]
  let current = data[0]
  let nextN = 1
  let nextIssue = 460

  function emit<K extends keyof EventMap>(event: K, payload: EventMap[K]) {
    handlers.get(event)?.forEach((handler) => handler(payload as never))
  }

  function on<K extends keyof EventMap>(event: K, handler: (payload: EventMap[K]) => void) {
    const set = handlers.get(event) ?? new Set<Handler>()
    set.add(handler as Handler)
    handlers.set(event, set)
    return () => set.delete(handler as Handler)
  }

  function view(s: MockSession): Session {
    return {
      id: s.id,
      n: s.n,
      title: s.title,
      state: s.state,
      detail: s.detail,
      lastEventAt: s.lastEventAt,
      createdAt: s.createdAt,
      issue: s.issue,
      model: s.model,
      effort: s.effort,
      history: s.history.map((h) => ({ ...h })),
      branch: s.branch,
      worktree: s.worktree,
      pr: s.pr ? { ...s.pr } : null,
      prAttention: s.prAttention,
      cleanup: s.cleanup,
      cleanupReason: s.cleanupReason,
      browser: s.browser,
      evidence: s.evidence,
    }
  }

  function currentSessions(): Session[] {
    return sessions
      .filter((s) => s.project === current.name)
      .map(view)
      .sort((a, b) => order[a.state] - order[b.state] || b.lastEventAt - a.lastEventAt)
  }

  function projectView(p: ProjectData): Project {
    const own = sessions.filter((s) => s.project === p.name)
    return {
      key: p.name,
      name: p.name,
      dir: p.dir,
      repo: p.repo,
      needsYou: own.filter((s) => s.state === 'waiting').length,
      working: own.filter((s) => s.state === 'working').length,
      sessions: own.length,
      needsSetup: !!p.needsSetup,
      setupDismissed: !!p.setupDismissed,
    }
  }

  function currentNotes(): Note[] {
    return [...current.notes].sort(
      (a, b) => Number(a.archived) - Number(b.archived) || Number(b.pinned) - Number(a.pinned) || b.createdAt - a.createdAt,
    )
  }

  const lastMoves = new Map<string, Issue>()

  function currentIssues(): Issue[] {
    return current.issues.map((issue) => ({
      ...issue,
      sessionId: sessions.find((s) => s.project === current.name && s.issue === issue.number)?.id ?? '',
    }))
  }

  type Shell = { id: string; project: string; buffer: string; line: string; opened: boolean }
  const shells: Shell[] = []
  const shellName = (project: string, n: number) => (n === 1 ? `shell-${project}` : `shell-${project}-${n}`)

  const prompt = (project: string) => `\x1b[38;2;167;139;250m${project}\x1b[0m \x1b[2m❯\x1b[0m `

  function shellWrite(shell: Shell, text: string) {
    shell.buffer += text
    if (shell.opened) emit('term:data', { id: shell.id, data: toBase64(text) })
  }

  function shellRun(shell: Shell, line: string) {
    const [name, ...args] = line.trim().split(/\s+/)
    if (!name) return
    if (name === 'agentos') {
      const [command = '', ...rest] = args
      emit('ui:command', { name: command, args: rest })
      shellWrite(shell, `\r\n\x1b[2mok: agentos ${args.join(' ')}\x1b[0m`)
      return
    }
    if (name === 'ls') shellWrite(shell, '\r\napp-frontend  api  e2e  locales  CLAUDE.md  package.json')
    else if (name === 'pwd') shellWrite(shell, `\r\n${current.dir}`)
    else if (name === 'clear') shellWrite(shell, '\x1bc')
    else shellWrite(shell, `\r\nzsh: command not found: ${name}`)
  }

  const shellsOf = (project: string) => shells.filter((x) => x.project === project)
  const findShell = (id: string) => shells.find((x) => x.id === id)

  function startShell(): Shell {
    const taken = shellsOf(current.name).map((x) => x.id)
    let n = 1
    while (taken.includes(shellName(current.name, n))) n++
    const shell: Shell = { id: shellName(current.name, n), project: current.name, buffer: `\x1b[2m${current.dir}\x1b[0m\r\n${prompt(current.name)}`, line: '', opened: false }
    shells.push(shell)
    return shell
  }

  function isShell(id: string) {
    return id.startsWith('shell-')
  }

  function snapshot(): Snapshot {
    return {
      project: projectView(current),
      projects: data.map(projectView),
      sessions: currentSessions(),
      notes: currentNotes(),
      version: 'mock',
      update: flags.update ? '1.4.0' : '',
      shells: shellsOf(current.name).map((x) => x.id),
    }
  }

  function publish() {
    emit('sessions', { project: current.name, items: currentSessions() })
    emit('projects', data.map(projectView))
  }

  function write(s: MockSession, text: string) {
    s.buffer += text
    if (s.opened) emit('term:data', { id: s.id, data: toBase64(text) })
  }

  function find(id: string): MockSession {
    const found = sessions.find((s) => s.id === id)
    if (!found) throw 'No such session'
    return found
  }

  function modelFor(owner: ProjectData, issue: number) {
    const found = owner.issues.find((i) => i.number === issue)
    const labelled = (key: string) => found?.labels.find((l) => l.startsWith(`${key}:`))?.slice(key.length + 1)
    return {
      model: labelled('model') ?? (found?.section === 'Needs plan' ? 'opus' : 'sonnet'),
      effort: labelled('effort') ?? 'medium',
    }
  }

  function create(
    title: string,
    issue: number,
    initial: State,
    history: HistoryEntry[] = [],
    createdAt = Date.now(),
    owner = current,
  ): MockSession {
    const s: MockSession = {
      id: `${owner.name}/${Math.random().toString(36).slice(2, 10)}`,
      n: nextN++,
      title,
      state: initial,
      detail: pick(details[initial]),
      lastEventAt: history.at(-1)?.at ?? 0,
      createdAt,
      issue,
      ...modelFor(owner, issue),
      history,
      branch: issue ? `issue-${issue}` : '',
      worktree: issue ? `${owner.dir}/trees/issue-${issue}` : '',
      pr: null,
      prAttention: '',
      cleanup: '',
      cleanupReason: '',
      browser: false,
      evidence: 0,
      project: owner.name,
      buffer: term.banner(owner.dir) + term.diffBlock('app-frontend/src/js/pages/[slug]/Index.tsx') + term.paragraphs[0] + '\r\n\r\n',
      opened: false,
      nextChangeAt: Date.now() + 8_000 + Math.random() * 25_000,
      stream: 1,
      cursor: 0,
      tools: 0,
    }
    if (initial === 'waiting') s.buffer += term.permissionPrompt('npx tsc --noEmit')
    if (initial === 'idle' && history.length > 0) s.buffer += term.finishedNote(134)
    if (initial === 'idle' && history.length === 0) s.buffer = term.banner(owner.dir) + term.promptLine('')
    if (initial === 'ended') s.buffer += '\r\n\x1b[2m[session ended]\x1b[0m\r\n'
    sessions.push(s)
    return s
  }

  function seed(title: string, issue: number, state: State, trail: [State, number][], owner = data[0]) {
    const now = Date.now()
    const history = trail.map(([st, minutesAgo]) => ({ state: st, at: now - minutesAgo * minute }))
    const s = create(title, issue, state, history, history[0] ? history[0].at - minute : now, owner)
    s.state = state
  }

  if (!flags.empty) {
    seed('#454 product page split', 454, 'working', [['working', 74], ['waiting', 52], ['working', 47], ['idle', 31], ['working', 12]])
    seed('#444 checkout route', 444, 'waiting', [['working', 96], ['idle', 70], ['working', 38], ['waiting', 4]])
    seed('#326 cart rounding', 326, 'idle', [['working', 41], ['waiting', 33], ['working', 30], ['idle', 6]])
    seed('checkout tests scratch', 0, 'working', [['working', 22], ['waiting', 15], ['working', 13]])
    seed('#412 locale sweep', 412, 'idle', [['working', 50], ['idle', 20]])
    seed('#437 coupon strip', 437, 'idle', [['working', 80], ['idle', 25]])
    seed('#430 thumbnail ratio', 430, 'ended', [['working', 60], ['ended', 18]])
    seed('spike: websocket retry', 0, 'ended', [['working', 140], ['idle', 110], ['ended', 95]])
    seed('#389 webhook retries', 389, 'working', [['working', 58], ['idle', 40], ['working', 9]])
    const byIssue = (n: number) => sessions.find((x) => x.issue === n && x.project === 'storefront')!
    byIssue(454).pr = pr(457, 'draft', 'pending')
    Object.assign(byIssue(444), { pr: pr(452, 'open', 'failing'), prAttention: 'checks' })
    Object.assign(byIssue(326), { pr: pr(449, 'open', 'passing', 3), prAttention: 'comments' })
    Object.assign(byIssue(412), { pr: pr(441, 'merged', 'passing'), cleanup: 'blocked', cleanupReason: 'the worktree has uncommitted changes' })
    Object.assign(byIssue(437), { pr: pr(445, 'merged', 'passing'), cleanup: 'pending' })
    Object.assign(byIssue(430), { pr: pr(447, 'closed', 'failing'), cleanup: 'ask' })
    seed('#12 session memory', 12, 'waiting', [['working', 30], ['waiting', 7]], data[1])
    seed('notes sync spike', 0, 'working', [['working', 11]], data[1])
  }

  if (flags.warn) {
    setTimeout(() => emit('warnings', { source: 'github', message: 'gh could not name the repo' }), 1500)
    setTimeout(() => emit('warnings', { source: 'github', message: '' }), 9000)
  }

  if (flags.toast) {
    setTimeout(() => {
      const target = sessions[2]
      if (target) emit('attention', { id: target.id, state: 'replied' })
    }, 1500)
    setTimeout(() => {
      const target = sessions.find((s) => s.pr?.state === 'draft')
      if (target) emit('attention', { id: target.id, state: 'opened' })
    }, 2500)
  }

  function owner(s: MockSession): ProjectData {
    return data.find((p) => p.name === s.project) ?? current
  }

  function closeWait(s: MockSession, at: number) {
    const open = owner(s).waits.find((w) => w.sessionTitle === s.title && w.waitedMs === 0)
    if (open) open.waitedMs = Math.max(at - open.startedAt, 1000)
  }

  function openWait(s: MockSession, at: number) {
    const permission = s.detail.includes('permission')
    const kind: WaitKind = s.state === 'idle' ? 'idle' : permission ? 'permission' : 'question'
    const label = { permission: 'Bash: npx tsc --noEmit', question: 'Question', idle: 'Reply landed' }[kind]
    owner(s).waits.unshift({ sessionTitle: s.title, issue: s.issue, model: s.model, kind, label, startedAt: at, waitedMs: 0 })
  }

  function setState(s: MockSession, state: State) {
    const at = Date.now()
    closeWait(s, at)
    s.state = state
    s.detail = pick(details[state])
    s.lastEventAt = at
    s.history = [...s.history, { state, at }].slice(-200)
    if (state === 'waiting') write(s, term.permissionPrompt('npx tsc --noEmit'))
    if (state === 'idle') write(s, term.finishedNote(Math.round((at - s.createdAt) / 1000) % 600))
    if (state === 'working') write(s, term.toolCall(s.tools++))
    publish()
    if (state === 'waiting' || state === 'idle') {
      openWait(s, at)
      emit('stats', undefined)
      if (s.project === current.name) emit('attention', { id: s.id, state: state === 'idle' ? 'replied' : 'waiting' })
    }
  }

  function nextState(state: State): State {
    if (state === 'working') return Math.random() < 0.6 ? 'waiting' : 'idle'
    return 'working'
  }

  setInterval(() => {
    const now = Date.now()
    sessions.forEach((s) => {
      if (s.state === 'ended' || (s.pr && (s.pr.state === 'merged' || s.pr.state === 'closed'))) return
      if (now >= s.nextChangeAt) {
        s.nextChangeAt = now + 12_000 + Math.random() * 28_000
        setState(s, nextState(s.state))
        return
      }
      if (s.state !== 'working') return
      if (Math.random() < 0.35) {
        s.detail = pick(details.working)
        s.lastEventAt = now
        publish()
      }
    })
  }, 1000)

  setInterval(() => {
    sessions.forEach((s) => {
      if (s.state !== 'working' || !s.opened) return
      const text = term.paragraphs[s.stream % term.paragraphs.length]
      const words = text.split(' ')
      const take = 2 + Math.floor(Math.random() * 3)
      const chunk = words.slice(s.cursor, s.cursor + take).join(' ')
      const done = s.cursor + take >= words.length
      s.cursor = done ? 0 : s.cursor + take
      if (done) s.stream++
      write(s, chunk + (done ? '\r\n\r\n' : ' '))
    })
  }, 140)

  function noteById(id: string): Note {
    const note = current.notes.find((n) => n.id === id)
    if (!note) throw 'No such note'
    return note
  }

  function touchNotes() {
    emit('notes', { project: current.name, items: currentNotes() })
  }

  function startSessionFor(title: string, issue: number, prefill: string): Session {
    const s = create(title, issue, 'idle')
    s.buffer = term.banner(current.dir) + term.promptLine(prefill)
    publish()
    return view(s)
  }

  function switchTo(next: ProjectData): Snapshot {
    current = next
    return snapshot()
  }

  function statsFor(days: number): Stats {
    const since = Date.now() - days * 86_400_000
    const waits = current.waits.filter((w) => w.startedAt >= since)
    const done = waits.filter((w) => w.waitedMs > 0).map((w) => w.waitedMs).sort((a, b) => a - b)
    const groups = new Map<string, { kind: WaitKind; label: string; count: number; totalWaitMs: number }>()
    for (const w of waits) {
      const g = groups.get(w.label) ?? { kind: w.kind, label: w.label, count: 0, totalWaitMs: 0 }
      g.count++
      g.totalWaitMs += w.waitedMs
      groups.set(w.label, g)
    }
    const byDay = Array.from({ length: days }, (_, i) => {
      const day = new Date(Date.now() - (days - 1 - i) * 86_400_000).toISOString().slice(0, 10)
      return { day, count: waits.filter((w) => new Date(w.startedAt).toISOString().slice(0, 10) === day).length }
    })
    return {
      days,
      total: waits.length,
      totalWaitMs: done.reduce((sum, ms) => sum + ms, 0),
      medianWaitMs: done.length ? done[Math.floor(done.length / 2)] : 0,
      byCause: [...groups.values()].sort((a, b) => b.count - a.count),
      byDay,
      recent: waits.slice(0, 50),
    }
  }

  function ledgerFor(days: number): Ledger {
    const dayKey = (ago: number) => new Date(Date.now() - ago * 86_400_000).toISOString().slice(0, 10)
    const heat = Array.from({ length: 364 }, (_, i) => {
      const weekday = (i + 3) % 7
      const wave = Math.sin(i / 9) + 1.2
      const count = weekday > 4 || i % 11 === 0 ? 0 : Math.round(wave * ((i * 37) % 23))
      return { day: dayKey(363 - i), count }
    })
    const scale = days === 0 ? 1 : days === 365 ? 0.8 : 0.12
    const row = (project: string, prompts: number, gh: [number, number, number, number] | null): LedgerRow => ({
      project,
      prompts: Math.round(prompts * scale),
      sessions: Math.round(prompts * scale * 0.08),
      issueSessions: Math.round(prompts * scale * 0.045),
      workMs: Math.round(prompts * scale * 0.13 * 3_600_000),
      prsMerged: gh && Math.round(gh[0] * scale),
      prsClosed: gh && Math.round(gh[1] * scale),
      issuesClosed: gh && Math.round(gh[2] * scale),
      issuesOpen: gh && gh[3],
    })
    const projects = [
      row('agentos', 3104, [181, 12, 170, 41]),
      row('upscope-web', 1209, [52, 5, 49, 14]),
      row('dotfiles', 499, [14, 2, 12, 3]),
      row('scratch', 38, null),
    ]
    const sum = (pick: (r: LedgerRow) => number | null) => {
      const values = projects.map(pick).filter((v): v is number => v !== null)
      return values.length ? values.reduce((a, b) => a + b, 0) : null
    }
    const totals: LedgerRow = {
      project: '',
      prompts: projects.reduce((a, r) => a + r.prompts, 0),
      sessions: projects.reduce((a, r) => a + r.sessions, 0),
      issueSessions: projects.reduce((a, r) => a + r.issueSessions, 0),
      workMs: projects.reduce((a, r) => a + r.workMs, 0),
      prsMerged: sum((r) => r.prsMerged),
      prsClosed: sum((r) => r.prsClosed),
      issuesClosed: sum((r) => r.issuesClosed),
      issuesOpen: sum((r) => r.issuesOpen),
    }
    return { days, since: dayKey(120), totals, projects, heat, github: 'scratch: no GitHub repository' }
  }

  function finishCleanup(s: MockSession) {
    const target = owner(s)
    target.cleanups.unshift({
      at: Date.now(),
      sessionTitle: s.title,
      issue: s.issue,
      pr: s.pr?.number ?? 0,
      merged: s.pr?.state === 'merged',
      status: 'done',
      removed: [`worktree ${s.worktree.replace(`${target.dir}/`, '')}`, `branch ${s.branch}`, 'temp files', 'session'],
      reason: '',
    })
    sessions.splice(sessions.indexOf(s), 1)
    emit('term:exit', { id: s.id })
    publish()
    emit('issues', { project: current.name, items: currentIssues() })
    if (target === current) emit('cleanups', { project: target.name, items: [...target.cleanups] })
  }

  const merging = sessions.find((s) => s.cleanup === 'pending')
  if (merging) setTimeout(() => finishCleanup(merging), flags.done ? 1500 : 9000)

  const failing = sessions.find((s) => s.issue === 389)
  if (failing) {
    setTimeout(() => {
      failing.pr = pr(455, 'open', 'failing')
      failing.prAttention = 'checks'
      publish()
      if (current.name === failing.project) emit('attention', { id: failing.id, state: 'pr' })
    }, 14000)
  }

  const pages = new Map<string, PageModel>()
  const evidence = new Map<string, Evidence[]>()
  const shotCanvas = document.createElement('canvas')
  let evidenceSeq = 0

  function browserStateOf(id: string): BrowserState {
    const page = pages.get(id)
    return {
      id,
      open: Boolean(page),
      url: page?.url ?? '',
      title: page?.title ?? '',
      loading: page?.loading ?? false,
      error: '',
      loadedAt: 0,
      pages: page ? 1 : 0,
      console: [],
    }
  }

  function pushBrowserState(id: string) {
    emit('browser:state', browserStateOf(id))
  }

  function navigate(id: string, url: string) {
    const page = pages.get(id)
    if (!page) throw 'This session has no browser open'
    page.url = url
    page.title = pageTitle(url)
    page.loading = true
    pushBrowserState(id)
    setTimeout(() => {
      page.loading = false
      pushBrowserState(id)
    }, 500)
  }

  function capture(id: string): string {
    const page = pages.get(id)
    if (!page) throw 'This session has no browser open'
    return renderPage(page, shotCanvas)
  }

  function addEvidence(s: MockSession, item: Omit<Evidence, 'id' | 'at'>, at = Date.now()): Evidence {
    const entry: Evidence = { ...item, id: `ev-${++evidenceSeq}`, at }
    const items = [...(evidence.get(s.id) ?? []), entry]
    evidence.set(s.id, items)
    s.evidence = items.length
    publish()
    emit('evidence', { id: s.id, items: items.map((e) => ({ ...e })) })
    if (item.source === 'agent') emit('attention', { id: s.id, state: 'evidence' })
    return entry
  }

  function openPage(s: MockSession, url: string): PageModel {
    const page = newPage(url)
    pages.set(s.id, page)
    s.browser = true
    publish()
    pushBrowserState(s.id)
    return page
  }

  const driven = sessions.find((s) => s.issue === 454)
  if (driven) {
    const page = openPage(driven, 'https://shop.acme.test/products')
    const steps: (() => void)[] = [
      () => navigate(driven.id, 'https://shop.acme.test/products'),
      () => {
        const { field } = pageRects()
        clickAt(page, field.x + 10, field.y + 10)
        typeText(page, 'canvas tote')
      },
      () => {
        const { button } = pageRects()
        clickAt(page, button.x + 10, button.y + 10)
      },
      () => addEvidence(driven, { kind: 'image', url: capture(driven.id), text: '', caption: 'Search results for "canvas tote" after the page split', source: 'agent' }),
      () => {
        page.typed = ''
        page.submitted = ''
        navigate(driven.id, 'https://shop.acme.test/cart')
      },
    ]
    let step = 0
    setTimeout(() => {
      setInterval(() => steps[step++ % steps.length](), 4500)
    }, 6000)
    addEvidence(driven, { kind: 'image', url: capture(driven.id), text: '', caption: 'Before: products list on main', source: 'agent' }, Date.now() - 50 * minute)
    page.submitted = 'tote'
    addEvidence(driven, { kind: 'image', url: capture(driven.id), text: '', caption: 'After: same page on issue-454', source: 'agent' }, Date.now() - 44 * minute)
    page.submitted = ''
    addEvidence(driven, { kind: 'text', url: '', text: 'tsc --noEmit: 0 errors\nvitest: 214 passed, 0 failed\nThe gallery now loads on the client only.', caption: 'Checks before the PR', source: 'agent' }, Date.now() - 20 * minute)
  }
  const second = sessions.find((s) => s.issue === 326)
  if (second) addEvidence(second, { kind: 'text', url: '', text: 'Totals are summed in cents and formatted once. Rounding each line drifted by a cent on large carts.', caption: 'Why cents', source: 'agent' }, Date.now() - 30 * minute)
  sessions.forEach((s) => {
    if (s.evidence > 0) s.nextChangeAt = Number.MAX_SAFE_INTEGER
  })

  const seedDigest = (): DigestItem[] => {
    const now = Date.now()
    const item = (n: number, at: number, source: string, title: string, why: string): DigestItem => ({
      id: `dg-${n}`, title, why, url: `https://example.com/changelog/${n}`, source, at, noteId: '',
    })
    return [
      item(1, now - 2 * 86_400_000, 'Claude Code', 'Hooks can match Bash commands with a pattern', 'Could replace the per-project permission rules in the config.'),
      item(2, now - 2 * 86_400_000, 'wails', 'v2.11 fixes the drag region in full screen', 'The hidden title bar here uses the drag region.'),
      item(3, now - 2 * 86_400_000, 'react', 'ReactDOM.render warnings are louder in 18.3', 'Two entry files still call render.'),
      item(4, now - 2 * 86_400_000, 'tailwind', 'v4.1 ships container queries in core', 'Could unblock the product grid layout work.'),
      item(5, now - 2 * 86_400_000, 'drizzle', 'Batch inserts land in 0.40', 'The customer import inserts rows one by one.'),
      item(6, now - 9 * 86_400_000, 'Claude Code', 'Plan mode now shows a checklist', 'Matches how /investigate presents its plan.'),
      item(7, now - 9 * 86_400_000, 'bullmq', 'Flow producers get a retry option', 'The image pipeline retries by hand today.'),
    ]
  }
  let digest: Digest = { running: false, lastRunAt: Date.now() - 2 * 86_400_000, nextRunAt: Date.now() + 5 * 86_400_000, error: '', project: data[0].name, items: seedDigest() }
  const pushDigest = () => emit('digest', { ...digest, items: digest.items.map((i) => ({ ...i })) })

  const delay = <T,>(value: T, ms = 220) => new Promise<T>((resolve) => setTimeout(() => resolve(value), ms))

  let settings: Settings = { theme: 'system', textScale: 1, keepAwake: true, cleanup: { merge: 'auto', close: 'manual' }, promptSend: 'auto', browserEnabled: true, digestSchedule: 'weekly', dataDir: '~/.local/share/agentos', dataDirFixed: false }

  const backend = {
    Snapshot: async () => snapshot(),
    Awake: async () => data.some((p) => projectView(p).working > 0),
    NewSession: async (title: string, prefill: string) => startSessionFor(title || `session ${nextN}`, 0, prefill),
    DismissSession: async (id: string) => {
      const s = find(id)
      if (s.state !== 'ended') throw 'Only an ended session can be dismissed'
      sessions.splice(sessions.indexOf(s), 1)
      publish()
      emit('issues', { project: current.name, items: currentIssues() })
    },
    KillSession: async (id: string) => {
      const s = find(id)
      if (s.state === 'ended') throw `session ${id} has ended: dismiss it`
      emit('term:exit', { id })
      setState(s, 'ended')
      emit('issues', { project: current.name, items: currentIssues() })
    },
    RenameSession: async (id: string, title: string) => {
      find(id).title = title
      publish()
    },
    SwitchProject: async (name: string) => {
      const next = data.find((p) => p.name === name)
      if (!next) throw 'No such project'
      return switchTo(next)
    },
    AddProject: async () => {
      const name = `picked-folder-${data.length}`
      data.push({ name, dir: `~/code/${name}`, repo: '', notes: [], issues: [], waits: [], cleanups: [], needsSetup: true })
      const snap = switchTo(data[data.length - 1])
      emit('projects', snap.projects)
      return snap
    },
    AddProjectDir: async (dir: string) => {
      if (!/^[~/]/.test(dir) || dir.includes('nope')) throw `Folder does not exist: ${dir}`
      const name = dir.replace(/\/+$/, '').split('/').pop() || dir
      if (data.some((p) => p.name === name)) throw `Project ${name} already exists`
      data.push({ name, dir, repo: '', notes: [], issues: [], waits: [], cleanups: [], needsSetup: true })
      const snap = switchTo(data[data.length - 1])
      emit('projects', snap.projects)
      return snap
    },
    NewProject: async (name: string) => {
      if (!/^[\w.-]+$/.test(name)) throw `${name} is not a project name`
      if (data.some((p) => p.name === name)) throw `Project ${name} already exists`
      data.push({ name, dir: `~/code/${name}`, repo: '', notes: [], issues: [], waits: [], cleanups: [], needsSetup: true })
      const snap = switchTo(data[data.length - 1])
      emit('projects', snap.projects)
      return snap
    },
    CreateRepo: async (name: string) => {
      const p = data.find((p) => p.name === name)
      if (!p) throw `no project called ${name}`
      if (name.includes('taken')) throw `${name} has no GitHub repo: name already exists`
      p.repo = `ned/${name}`
      return snapshot()
    },
    RemoveProject: async (name: string) => {
      const at = data.findIndex((p) => p.name === name)
      if (at < 0) throw 'No such project'
      if (data.length === 1) throw 'Cannot remove the last project'
      data.splice(at, 1)
      const snap = current.name === name ? switchTo(data[0]) : snapshot()
      emit('projects', snap.projects)
      return snap
    },
    SetUpProject: async () => {
      if (current.needsSetup) current.issues = current.issues.map((issue) => ({ ...issue, section: 'Inbox', actions: ['Work'], moves: movesFrom('Inbox') }))
      current.needsSetup = false
      emit('projects', data.map(projectView))
      return startSessionFor('Set up for agentos', 0, 'Set up this project for agentos with me.')
    },
    DismissSetup: async () => {
      current.setupDismissed = true
      const snap = snapshot()
      emit('projects', snap.projects)
      return snap
    },
    AddNote: async (text: string) => {
      const now = Date.now()
      const note: Note = { id: `note-${Math.random().toString(36).slice(2, 8)}`, text, createdAt: now, updatedAt: now, pinned: false, archived: false, images: [] }
      current.notes.push(note)
      touchNotes()
      return { ...note }
    },
    UpdateNote: async (id: string, text: string) => {
      const note = noteById(id)
      note.text = text
      note.updatedAt = Date.now()
      touchNotes()
      return { ...note }
    },
    SetNotePinned: async (id: string, pinned: boolean) => {
      const note = noteById(id)
      note.pinned = pinned
      touchNotes()
      return { ...note }
    },
    SetNoteArchived: async (id: string, archived: boolean) => {
      const note = noteById(id)
      note.archived = archived
      if (archived) note.pinned = false
      touchNotes()
      return { ...note }
    },
    AddNoteImage: async (id: string, base64: string, mime: string) => {
      const note = noteById(id)
      note.images = [...note.images, `data:${mime};base64,${base64}`]
      touchNotes()
      return { ...note }
    },
    RemoveNoteImage: async (id: string, url: string) => {
      const note = noteById(id)
      note.images = note.images.filter((u) => u !== url)
      touchNotes()
      return { ...note }
    },
    DeleteNote: async (id: string) => {
      noteById(id)
      current.notes = current.notes.filter((n) => n.id !== id)
      touchNotes()
    },
    NoteToIssue: async (id: string) => {
      const note = noteById(id)
      if (!current.repo) throw 'This project has no GitHub repo'
      const number = nextIssue++
      const now = Date.now()
      current.issues.unshift({
        number,
        title: note.text.split('\n')[0].slice(0, 90),
        type: '',
        section: 'Inbox',
        actions: ['Plan', 'Investigate', 'Work'],
        moves: movesFrom('Inbox'),
        url: `https://github.com/${current.repo}/issues/${number}`,
        sessionId: '',
        author: 'nednella',
        assignees: [],
        labels: [],
        createdAt: now,
        updatedAt: now,
      })
      current.notes = current.notes.filter((n) => n.id !== id)
      touchNotes()
      emit('issues', { project: current.name, items: currentIssues() })
    },
    NoteToSession: async (id: string) => {
      const note = noteById(id)
      return startSessionFor(note.text.split('\n')[0], 0, note.text)
    },
    Issues: async (refresh: boolean) => delay(currentIssues(), refresh ? 700 : 220),
    StartIssue: async (number: number, action: string) => {
      const issue = current.issues.find((i) => i.number === number)
      if (!issue) throw 'No such issue'
      const existing = sessions.find((s) => s.project === current.name && s.issue === number)
      if (existing) return view(existing)
      if (action && !issue.actions.includes(action)) throw `issue #${number} has no action "${action}"`
      const command = { Work: `/work ${number}`, Plan: `/plan ${number}`, Investigate: `/investigate ${number}` }[action || issue.actions[0]] ?? `Work on issue #${number}: ${issue.title}`
      const short = issue.title.split(' ').slice(0, 3).join(' ')
      const session = startSessionFor(`#${number} ${short}`, number, command)
      emit('issues', { project: current.name, items: currentIssues() })
      return session
    },
    MoveIssue: async (number: number, section: string) => {
      const issue = current.issues.find((i) => i.number === number)
      if (!issue) throw 'No such issue'
      const moved = moveIssue(issue, section)
      await delay(null, 500)
      lastMoves.set(`${current.name}#${number}`, issue)
      current.issues[current.issues.indexOf(issue)] = moved
      sortBySection(current.issues)
      emit('issues', { project: current.name, items: currentIssues() })
    },
    UndoMove: async (number: number) => {
      const before = lastMoves.get(`${current.name}#${number}`)
      const at = current.issues.findIndex((i) => i.number === number)
      if (!before || at < 0) throw `nothing to undo for issue #${number}`
      await delay(null, 500)
      lastMoves.delete(`${current.name}#${number}`)
      current.issues[at] = before
      sortBySection(current.issues)
      emit('issues', { project: current.name, items: currentIssues() })
    },
    IssueDetail: async (number: number) => {
      if (!current.issues.some((i) => i.number === number)) throw `reading issue #${number}: gh api: HTTP 404: Not Found`
      return delay(buildIssueDetail(number), 350)
    },
    ShellOpen: async () => {
      return { id: (shellsOf(current.name)[0] ?? startShell()).id }
    },
    ShellNew: async () => ({ id: startShell().id }),
    ShellClose: async (id: string) => {
      const at = shells.findIndex((x) => x.id === id)
      if (at >= 0) shells.splice(at, 1)
    },
    TermOpen: async (id: string) => {
      if (isShell(id)) {
        const shell = findShell(id)
        if (!shell) throw 'No such shell'
        shell.opened = true
        emit('term:data', { id, data: toBase64(`\x1bc${shell.buffer}`) })
        return
      }
      const s = find(id)
      s.opened = true
      emit('term:data', { id, data: toBase64(`\x1bc${s.buffer}`) })
    },
    TermWrite: async (id: string, data: string) => {
      if (isShell(id)) {
        const shell = findShell(id)
        if (!shell) return
        for (const ch of data) {
          if (ch === '\r') {
            const line = shell.line
            shell.line = ''
            shellRun(shell, line)
            shellWrite(shell, `\r\n${prompt(shell.project)}`)
          } else if (ch === '\x7f') {
            if (shell.line) {
              shell.line = shell.line.slice(0, -1)
              shellWrite(shell, '\b \b')
            }
          } else if (ch >= ' ') {
            shell.line += ch
            shellWrite(shell, ch)
          }
        }
        return
      }
      const echo = data.replace(/\r/g, '\r\n').replace(/\x7f/g, '\b \b')
      write(find(id), echo)
    },
    TermResize: async () => undefined,
    TermClose: async (id: string) => {
      const shell = findShell(id)
      if (shell) shell.opened = false
      const s = sessions.find((x) => x.id === id)
      if (s) s.opened = false
    },
    Stats: async (days: number) => delay(statsFor(days), 150),
    Ledger: async (days: number) => delay(ledgerFor(days), 300),
    RefreshPRs: async () => delay(undefined, 500),
    AckPR: async (id: string) => {
      find(id).prAttention = ''
      publish()
    },
    TypeInto: async (id: string, text: string) => {
      write(find(id), text)
    },
    Cleanup: async (id: string, force: boolean) => {
      const s = find(id)
      if (s.cleanup === 'blocked' && !force) throw `Not cleaned up: ${s.cleanupReason}`
      finishCleanup(s)
    },
    Cleanups: async () => [...current.cleanups],
    BrowserOpen: async (id: string, url: string) => {
      const s = find(id)
      openPage(s, url ? normalizeUrl(url) : 'https://docs.acme.test/')
      return browserStateOf(id)
    },
    BrowserGoto: async (id: string, url: string) => navigate(id, normalizeUrl(url)),
    BrowserShow: async () => {},
    BrowserState: async (id: string) => browserStateOf(id),
    BrowserScreenshot: async (id: string, caption: string) =>
      addEvidence(find(id), { kind: 'image', url: capture(id), text: '', caption, source: 'user' }),
    BrowserClose: async (id: string) => {
      const s = find(id)
      pages.delete(id)
      s.browser = false
      publish()
      pushBrowserState(id)
    },
    Evidence: async (id: string) => (evidence.get(id) ?? []).map((e) => ({ ...e })),
    DeleteEvidence: async (id: string, evidenceId: string) => {
      const s = find(id)
      const items = (evidence.get(id) ?? []).filter((e) => e.id !== evidenceId)
      evidence.set(id, items)
      s.evidence = items.length
      publish()
      emit('evidence', { id, items })
    },
    Digest: async () => ({ ...digest, items: digest.items.map((i) => ({ ...i })) }),
    RunDigest: async () => {
      if (digest.running) throw 'A digest run is already in progress'
      digest = { ...digest, running: true, error: '' }
      pushDigest()
      setTimeout(() => {
        const now = Date.now()
        const fresh: DigestItem[] = [
          ['Claude Code', 'Skills can declare allowed tools', 'Lets /work run without a permission prompt for tsc.'],
          ['wails', 'Native file drop events', 'Notes could take dropped screenshots from Finder.'],
          ['xterm.js', 'Synchronized output mode', 'Would stop flicker while the agent redraws.'],
          ['vite', 'v8 build is faster', 'No change needed, nice to have for the shop.'],
          ['gh', 'pr checks --watch exits on first failure', 'The PR poller could use it.'],
        ].map(([source, title, why], i) => ({ id: `dg-new-${now}-${i}`, title, why, url: 'https://example.com/changelog/new', source, at: now, noteId: '' }))
        digest = { running: false, lastRunAt: now, nextRunAt: now + 7 * 86_400_000, error: '', project: digest.project, items: [...fresh, ...digest.items].slice(0, 30) }
        pushDigest()
      }, 4000)
    },
    DigestToNote: async (itemId: string) => {
      const item = digest.items.find((i) => i.id === itemId)
      if (!item) throw 'No such digest item'
      const now = Date.now()
      const note: Note = { id: `note-${Math.random().toString(36).slice(2, 8)}`, text: `${item.title}\n${item.why}\n${item.url}`, createdAt: now, updatedAt: now, pinned: false, archived: false, images: [] }
      current.notes.push(note)
      item.noteId = note.id
      touchNotes()
      pushDigest()
      return { ...note }
    },
    DismissDigestItem: async (itemId: string) => {
      digest = { ...digest, items: digest.items.filter((i) => i.id !== itemId) }
      pushDigest()
    },
    Settings: async () => settings,
    SetTheme: async (theme: ThemeSetting) => {
      settings = { ...settings, theme }
      return settings
    },
    SetCleanup: async (event: CleanupEvent, mode: CleanupMode) => {
      settings = { ...settings, cleanup: { ...settings.cleanup, [event]: mode } }
      return settings
    },
    SetTextScale: async (textScale: number) => {
      settings = { ...settings, textScale }
      return settings
    },
    SetKeepAwake: async (keepAwake: boolean) => {
      settings = { ...settings, keepAwake }
      return settings
    },
    SetPromptSend: async (promptSend: PromptSend) => {
      settings = { ...settings, promptSend }
      return settings
    },
    SetBrowserEnabled: async (browserEnabled: boolean) => {
      settings = { ...settings, browserEnabled }
      return settings
    },
    SetDigestSchedule: async (digestSchedule: DigestSchedule) => {
      settings = { ...settings, digestSchedule }
      return settings
    },
    PickDataDir: async () => delay({ dir: '~/Library/Mobile Documents/com~apple~CloudDocs/agentos', empty: !flags.fullDataDir }),
    SetDataDir: async (_dir: string, withData: boolean) => {
      await delay(undefined, withData ? 2500 : 400)
    },
    Update: async () => {
      await delay(undefined, 2500)
      throw 'The mock cannot relaunch'
    },
    PatchNotes: async () => (flags.patchNotes ? PATCH_NOTES : []),
    OpenURL: async (url: string) => {
      window.open(url, '_blank')
    },
  }

  return { backend, on, flags }
}
