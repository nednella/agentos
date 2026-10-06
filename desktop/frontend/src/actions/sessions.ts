import type { Session } from '../types'
import type { Action, ActionContext } from './types'

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

const describe = (s: Session) => `${s.n} ${s.title}`

export const sessionActions = ({ a, current }: ActionContext): Action[] => [
  {
    id: 'new-session',
    label: 'New session',
    group: 'Sessions',
    shortcut: { key: 'n' },
    run() {
      a.setComposing(true)
    },
  },
  {
    id: 'open-session',
    label: 'Open session',
    group: 'Sessions',
    palette: false,
    uiCommand: 'open',
    run(args) {
      const session = findSession(a.sessions, args.join(' '))
      a.select(session.id)
      return `Opened ${describe(session)}`
    },
  },
  {
    id: 'detach-session',
    label: current ? `Detach session ${describe(current)}` : 'Detach session',
    group: 'Sessions',
    shortcut: { key: 'w' },
    run() {
      if (!current) throw 'No session selected'
      a.detach()
      return `Detached ${describe(current)}`
    },
  },
  {
    id: 'next-attention',
    label: 'Next that needs you',
    group: 'Sessions',
    shortcut: { key: 'e' },
    uiCommand: 'next',
    run() {
      a.nextAttention()
    },
  },
  {
    id: 'kill-session',
    label: current ? `Kill session ${describe(current)}` : 'Kill session',
    group: 'Sessions',
    confirm: true,
    async run(args) {
      const target = args.length ? findSession(a.sessions, args.join(' ')) : current
      if (!target) throw 'No session to kill'
      await a.killSession(target.id)
      return `Killed ${describe(target)}`
    },
  },
  {
    id: 'rename-session',
    label: current ? `Rename session ${describe(current)}` : 'Rename session',
    group: 'Sessions',
    palette: { prompt: 'New title', initial: current?.title },
    async run(args) {
      if (!current) throw 'No session to rename'
      if (args.length === 0) throw 'Give the new title: rename <title>'
      await a.renameSession(current.id, args.join(' '))
      return `Renamed to "${args.join(' ')}"`
    },
  },
  {
    id: 'open-pr',
    label: current?.pr ? `Open PR #${current.pr.number}` : 'Open the pull request',
    group: 'Sessions',
    uiCommand: 'pr',
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
    async run(args) {
      const target = args.length ? findSession(a.sessions, args.join(' ')) : current
      if (!target) throw 'No session to clean up'
      await a.cleanupSession(target.id, false)
      return `Cleaned up ${describe(target)}`
    },
  },
  {
    id: 'harness',
    label: "Check this project's harness",
    group: 'Projects',
    uiCommand: 'harness',
    async run() {
      await a.harnessCheck()
      return 'Started the harness check'
    },
  },
  ...Array.from({ length: 9 }, (_, i): Action => {
    const n = i + 1
    return {
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
    }
  }),
]
