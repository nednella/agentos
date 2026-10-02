import { useAgentos } from './AgentosContext'
import type { Overlay } from './AgentosContext'
import { isFiltering } from './issueFilter'
import { useLayout } from './LayoutContext'
import type { Project, Session } from './types'

export type Candidate = { value: string; label: string; detail?: string }
export type Shortcut = {
  key: string
  shift?: boolean
  alt?: boolean
  code?: string
  aliases?: string[]
  anyShift?: boolean
  label?: string
}
export type ActionGroup = 'Sessions' | 'Navigate' | 'Projects' | 'Notes' | 'Queue' | 'App'

type Via = 'ui' | 'command'

export type Action = {
  id: string
  label: string
  group: ActionGroup
  shortcut?: Shortcut
  keysLabel?: string
  hidden?: boolean
  command?: {
    name: string
    syntax: string
    summary: string
    complete?(args: string[]): Candidate[]
  }
  palette?: false | { prompt?: string; initial?: string }
  confirm?: boolean
  run(args: string[], via: Via): string | void | Promise<string | void>
}

const KEY_LABELS: Record<string, string> = { arrowleft: '←', arrowright: '→', arrowup: '↑', arrowdown: '↓' }

export function formatShortcut(shortcut: Shortcut): string {
  const key = shortcut.label ?? KEY_LABELS[shortcut.key] ?? shortcut.key.toUpperCase()
  return `⌘${shortcut.alt ? '⌥' : ''}${shortcut.shift ? '⇧' : ''}${key}`
}

export function matchShortcut(e: KeyboardEvent, shortcut: Shortcut): boolean {
  if (!e.metaKey || e.ctrlKey || e.altKey !== Boolean(shortcut.alt)) return false
  if (!shortcut.anyShift && e.shiftKey !== Boolean(shortcut.shift)) return false
  if (shortcut.code) return e.code === shortcut.code
  return [shortcut.key, ...(shortcut.aliases ?? [])].includes(e.key.toLowerCase())
}

export const LIST_KEYS: { keys: string; summary: string }[] = [
  { keys: 'W / S or ↑ / ↓', summary: 'Move the row cursor in a focused list' },
  { keys: 'Enter', summary: 'Act on the row: queue starts or jumps, sessions opens, notes edits' },
  { keys: 'A / D or ← / →', summary: 'Switch Queue and Notes, or fold a lane' },
  { keys: 'P / E in notes', summary: 'Pin or archive the note under the cursor' },
  { keys: 'Esc', summary: 'Close a panel or overlay. In the terminal it goes to the agent; ⌘⇧2 or a click brings you back' },
]

const includes = (text: string, partial: string) => text.toLowerCase().includes(partial.toLowerCase())

function findSession(sessions: Session[], query: string): Session {
  const text = query.trim().replace(/^#?/, '')
  if (!text) throw 'Give a session number or part of its title'
  if (/^\d+$/.test(text)) {
    const byNumber = sessions.find((s) => s.n === Number(text))
    if (byNumber) return byNumber
  }
  const matches = sessions.filter((s) => includes(s.title, query.trim()))
  if (matches.length === 1) return matches[0]
  if (matches.length > 1) throw `${matches.length} sessions match "${query}": ${matches.map((s) => s.n).join(', ')}`
  throw `No session matches "${query}"`
}

function findProject(projects: Project[], query: string): Project {
  const name = query.trim().toLowerCase()
  if (!name) throw 'Give a project name'
  const exact = projects.find((p) => p.name.toLowerCase() === name)
  if (exact) return exact
  const matches = projects.filter((p) => p.name.toLowerCase().startsWith(name))
  if (matches.length === 1) return matches[0]
  if (matches.length > 1) throw `${matches.length} projects match "${query}"`
  throw `No project named "${query}"`
}

const describe = (s: Session) => `${s.n} ${s.title}`

export function useActions(): Action[] {
  const a = useAgentos()
  const layout = useLayout()
  const current = a.sessions.find((s) => s.id === a.selectedId)
  const openOverlay = (overlay: Overlay) => () => a.setOverlay(overlay)

  const sessionCandidates = (args: string[]): Candidate[] =>
    a.sessions
      .filter((s) => includes(describe(s), args[args.length - 1] ?? ''))
      .map((s) => ({ value: String(s.n), label: s.title, detail: `#${s.n}` }))

  const projectCandidates = (partial: string, extra: string[] = []): Candidate[] => [
    ...extra.filter((e) => e.startsWith(partial)).map((e) => ({ value: e, label: e, detail: 'subcommand' })),
    ...a.projects
      .filter((p) => p.name.toLowerCase().startsWith(partial.toLowerCase()))
      .map((p) => ({ value: p.name, label: p.name, detail: `${p.sessions} sessions` })),
  ]

  const actions: Action[] = [
    {
      id: 'new-session',
      label: 'New session',
      group: 'Sessions',
      shortcut: { key: 'n' },
      command: { name: 'new', syntax: 'new [title]', summary: 'Start a session' },
      run(args, via) {
        if (via === 'ui') {
          a.setComposing(true)
          return
        }
        return a.newSession(args.join(' ')).then((s) => `Started session ${describe(s)}`)
      },
    },
    {
      id: 'open-session',
      label: 'Open session',
      group: 'Sessions',
      palette: false,
      command: { name: 'open', syntax: 'open <n|title>', summary: 'Jump to a session', complete: sessionCandidates },
      run(args) {
        const session = findSession(a.sessions, args.join(' '))
        a.select(session.id)
        return `Opened ${describe(session)}`
      },
    },
    {
      id: 'next-attention',
      label: 'Next that needs you',
      group: 'Sessions',
      shortcut: { key: 'e' },
      command: { name: 'next', syntax: 'next', summary: 'Jump to the next session that needs you' },
      run() {
        a.nextAttention()
      },
    },
    {
      id: 'stop-session',
      label: current ? `Stop session ${describe(current)}` : 'Stop session',
      group: 'Sessions',
      confirm: true,
      command: { name: 'stop', syntax: 'stop [n]', summary: 'Stop a session (default: the current one)', complete: sessionCandidates },
      async run(args) {
        const target = args.length ? findSession(a.sessions, args.join(' ')) : current
        if (!target) throw 'No session to stop'
        await a.killSession(target.id)
        return `Stopped ${describe(target)}`
      },
    },
    {
      id: 'rename-session',
      label: current ? `Rename session ${describe(current)}` : 'Rename session',
      group: 'Sessions',
      palette: { prompt: 'New title', initial: current?.title },
      command: { name: 'rename', syntax: 'rename <title>', summary: 'Rename the current session' },
      async run(args) {
        if (!current) throw 'No session to rename'
        if (args.length === 0) throw 'Give the new title: rename <title>'
        await a.renameSession(current.id, args.join(' '))
        return `Renamed to "${args.join(' ')}"`
      },
    },
    {
      id: 'start-issue',
      label: 'Start issue',
      group: 'Queue',
      palette: false,
      command: {
        name: 'issue',
        syntax: 'issue <number…>',
        summary: 'Start a session from one or more issues',
        complete: (args) =>
          a.issues
            .filter((i) => !i.sessionId && includes(`${i.number} ${i.title}`, args[args.length - 1] ?? ''))
            .slice(0, 12)
            .map((i) => ({ value: String(i.number), label: i.title, detail: `#${i.number}` })),
      },
      async run(args) {
        const numbers = args.map((x) => Number(x.replace(/^#/, '')))
        if (numbers.length === 0 || numbers.some((n) => !Number.isInteger(n))) throw 'Give issue numbers: issue 394 393'
        for (const [i, number] of numbers.entries()) {
          try {
            await a.startIssue(number, i < numbers.length - 1)
          } catch (err) {
            throw `Issue #${number}: ${typeof err === 'string' ? err : 'could not start'}`
          }
        }
        return `Started ${numbers.map((n) => `#${n}`).join(', ')}`
      },
    },
    {
      id: 'switch-project',
      label: 'Switch project…',
      group: 'Projects',
      shortcut: { key: 'p' },
      run: openOverlay('projects'),
    },
    {
      id: 'project',
      label: 'Project',
      group: 'Projects',
      palette: false,
      command: {
        name: 'project',
        syntax: 'project <name> | add [path] | remove <name>',
        summary: 'Switch, add or remove a project',
        complete: (args) =>
          args[0] === 'remove' || args.length > 2
            ? projectCandidates(args[args.length - 1] ?? '')
            : projectCandidates(args[0] ?? '', ['add', 'remove']),
      },
      async run(args) {
        if (args.length === 0) {
          a.setOverlay('projects')
          return
        }
        if (args[0] === 'add') {
          await a.addProject(args.slice(1).join(' ') || undefined)
          return 'Added project'
        }
        if (args[0] === 'remove') {
          const target = findProject(a.projects, args.slice(1).join(' '))
          await a.removeProject(target.name)
          return `Removed project ${target.name}`
        }
        const target = findProject(a.projects, args.join(' '))
        await a.switchProject(target.name)
        return `Switched to ${target.name}`
      },
    },
    {
      id: 'add-project',
      label: 'Add project…',
      group: 'Projects',
      async run() {
        await a.addProject()
      },
    },
    {
      id: 'note',
      label: 'New note',
      group: 'Notes',
      shortcut: { key: 'n', shift: true },
      palette: { prompt: 'Note text' },
      command: { name: 'note', syntax: 'note <text>', summary: 'Capture a note' },
      async run(args) {
        if (args.length === 0) {
          layout.showSidebarTab('notes')
          a.focus('note-input')
          return
        }
        await a.addNote(args.join(' '))
        return 'Note added'
      },
    },
    {
      id: 'show-notes',
      label: 'Show notes',
      group: 'Notes',
      command: { name: 'notes', syntax: 'notes', summary: 'Show the notes tab' },
      run() {
        layout.showSidebarTab('notes')
      },
    },
    {
      id: 'show-queue',
      label: 'Search the queue',
      group: 'Queue',
      shortcut: { key: 'q', shift: true },
      command: { name: 'queue', syntax: 'queue', summary: 'Show the queue tab and focus its search' },
      run() {
        layout.showSidebarTab('queue')
        a.focus('queue-filter')
      },
    },
    {
      id: 'filter-queue',
      label: 'Filter the queue…',
      group: 'Queue',
      palette: { prompt: 'Filter, e.g. @mariam-k label:idea -type:bug', initial: a.issueFilter },
      command: { name: 'filter', syntax: 'filter [query]', summary: 'Filter the queue (no query clears it)' },
      run(args) {
        layout.showSidebarTab('queue')
        a.setIssueFilter(args.join(' '))
        return isFiltering(args.join(' ')) ? 'Filter set' : 'Filter cleared'
      },
    },
    {
      id: 'refresh-issues',
      label: 'Refresh issues and PRs',
      group: 'Queue',
      shortcut: { key: 'r' },
      command: { name: 'refresh', syntax: 'refresh', summary: 'Reload issues and pull requests from GitHub' },
      async run() {
        await Promise.all([a.refreshIssues(), a.refreshPRs()])
        return 'Issues and pull requests refreshed'
      },
    },
    {
      id: 'toggle-sidebar',
      label: 'Toggle queue and notes panel',
      group: 'Navigate',
      shortcut: { key: 'b' },
      run: layout.toggleSidebar,
    },
    {
      id: 'toggle-sessions',
      label: 'Toggle sessions panel',
      group: 'Navigate',
      shortcut: { key: 'b', code: 'KeyB', alt: true },
      run: layout.toggleSessions,
    },
    {
      id: 'focus-command',
      label: 'Focus command line',
      group: 'Navigate',
      shortcut: { key: 'l' },
      run: () => layout.focusPanel('command'),
    },
    {
      id: 'panel-left',
      label: 'Focus the panel on the left',
      group: 'Navigate',
      shortcut: { key: 'arrowleft', alt: true },
      palette: false,
      run: () => layout.movePanel(-1),
    },
    {
      id: 'panel-right',
      label: 'Focus the panel on the right',
      group: 'Navigate',
      shortcut: { key: 'arrowright', alt: true },
      palette: false,
      run: () => layout.movePanel(1),
    },
    {
      id: 'panel-sidebar',
      label: 'Focus queue and notes',
      group: 'Navigate',
      shortcut: { key: '1', code: 'Digit1', shift: true, label: '1' },
      run: () => layout.focusPanel('sidebar'),
    },
    {
      id: 'panel-terminal',
      label: 'Focus terminal',
      group: 'Navigate',
      shortcut: { key: '2', code: 'Digit2', shift: true, label: '2' },
      run: () => layout.focusPanel('terminal'),
    },
    {
      id: 'panel-sessions',
      label: 'Focus sessions',
      group: 'Navigate',
      shortcut: { key: '3', code: 'Digit3', shift: true, label: '3' },
      run: () => layout.focusPanel('sessions'),
    },
    {
      id: 'panel-command',
      label: 'Focus command line (panel 4)',
      group: 'Navigate',
      shortcut: { key: '4', code: 'Digit4', shift: true, label: '4' },
      palette: false,
      run: () => layout.focusPanel('command'),
    },
    {
      id: 'zoom-in',
      label: 'Larger text',
      group: 'App',
      shortcut: { key: '=', aliases: ['+'], anyShift: true, label: '+' },
      run: () => layout.zoom(1),
    },
    {
      id: 'zoom-out',
      label: 'Smaller text',
      group: 'App',
      shortcut: { key: '-', aliases: ['_'], anyShift: true },
      run: () => layout.zoom(-1),
    },
    {
      id: 'zoom-reset',
      label: 'Reset text size',
      group: 'App',
      shortcut: { key: '0' },
      run: () => layout.zoom(0),
    },
    {
      id: 'stats',
      label: 'Stats: what interrupts you',
      group: 'App',
      shortcut: { key: 's', shift: true },
      command: { name: 'stats', syntax: 'stats', summary: 'Show or hide the interruption stats' },
      run: layout.toggleStats,
    },
    {
      id: 'show-terminal',
      label: 'Show terminal',
      group: 'Sessions',
      command: { name: 'term', syntax: 'term', summary: "Show the session's terminal" },
      run() {
        if (!current) throw 'No session selected'
        layout.closeCentre()
        a.setSessionView(current.id, 'terminal')
      },
    },
    {
      id: 'show-browser',
      label: 'Show browser',
      group: 'Sessions',
      command: { name: 'browser', syntax: 'browser [url]', summary: "Show the session's browser, or open a page in it" },
      async run(args) {
        if (!current) throw 'No session selected'
        layout.closeCentre()
        if (args.length === 0) {
          a.setSessionView(current.id, 'browser')
          return
        }
        await a.openBrowser(current.id, args.join(' '))
        return `Opened ${args.join(' ')}`
      },
    },
    {
      id: 'show-evidence',
      label: 'Show evidence',
      group: 'Sessions',
      command: { name: 'evidence', syntax: 'evidence', summary: "Show what the session has captured" },
      run() {
        if (!current) throw 'No session selected'
        layout.closeCentre()
        a.setSessionView(current.id, 'evidence')
      },
    },
    {
      id: 'view-previous',
      label: 'Previous view (terminal, browser, evidence)',
      group: 'Navigate',
      shortcut: { key: '[', code: 'BracketLeft', shift: true, label: '[' },
      palette: false,
      run: () => a.cycleSessionView(-1),
    },
    {
      id: 'view-next',
      label: 'Next view (terminal, browser, evidence)',
      group: 'Navigate',
      shortcut: { key: ']', code: 'BracketRight', shift: true, label: ']' },
      palette: false,
      run: () => a.cycleSessionView(1),
    },
    {
      id: 'digest',
      label: 'Weekly digest',
      group: 'App',
      shortcut: { key: 'd', shift: true },
      command: { name: 'digest', syntax: 'digest', summary: 'Show or hide the weekly digest' },
      run: layout.toggleDigest,
    },
    {
      id: 'harness',
      label: "Check this project's harness",
      group: 'Projects',
      command: { name: 'harness', syntax: 'harness', summary: "Start a session that reviews the project's harness" },
      async run() {
        await a.harnessCheck()
        return 'Started the harness check'
      },
    },
    {
      id: 'open-pr',
      label: current?.pr ? `Open PR #${current.pr.number}` : 'Open the pull request',
      group: 'Sessions',
      command: { name: 'pr', syntax: 'pr', summary: "Open the current session's pull request" },
      async run() {
        if (!current) throw 'No session selected'
        await a.openPR(current)
        return `Opened PR #${current.pr?.number}`
      },
    },
    {
      id: 'cleanup',
      label: current ? `Clean up session ${describe(current)}` : 'Clean up session',
      group: 'Sessions',
      confirm: true,
      command: { name: 'cleanup', syntax: 'cleanup [n]', summary: 'Remove the worktree, branch and session once its PR is done', complete: sessionCandidates },
      async run(args) {
        const target = args.length ? findSession(a.sessions, args.join(' ')) : current
        if (!target) throw 'No session to clean up'
        await a.cleanupSession(target.id, false)
        return `Cleaned up ${describe(target)}`
      },
    },
    {
      id: 'previous-session',
      label: 'Previous session',
      group: 'Navigate',
      shortcut: { key: '[' },
      run: () => a.stepSession(-1),
    },
    {
      id: 'next-session',
      label: 'Next session',
      group: 'Navigate',
      shortcut: { key: ']' },
      run: () => a.stepSession(1),
    },
    {
      id: 'palette',
      label: 'Command palette',
      group: 'Navigate',
      shortcut: { key: 'k' },
      palette: false,
      run: () => a.setOverlay(a.overlay === 'palette' ? null : 'palette'),
    },
    {
      id: 'shortcuts',
      label: 'Keyboard shortcuts',
      group: 'App',
      shortcut: { key: '/' },
      command: { name: 'help', syntax: 'help', summary: 'Show the shortcuts' },
      run: openOverlay('shortcuts'),
    },
    {
      id: 'clear',
      label: 'Clear',
      group: 'App',
      palette: false,
      command: { name: 'clear', syntax: 'clear', summary: 'Clear the message line' },
      run: () => '',
    },
  ]

  for (let n = 1; n <= 9; n++) {
    actions.push({
      id: `goto-${n}`,
      label: `Open session ${n}`,
      group: 'Sessions',
      shortcut: { key: String(n) },
      keysLabel: n === 1 ? '⌘1–9' : undefined,
      hidden: n > 1,
      palette: false,
      run() {
        const target = a.sessions.find((s) => s.n === n)
        if (!target) throw `No session ${n}`
        a.select(target.id)
      },
    })
  }
  return actions
}
