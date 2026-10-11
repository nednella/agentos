import type { Session } from '../types'
import type { Action, ActionContext } from './types'

const includes = (text: string, partial: string) => text.toLowerCase().includes(partial.toLowerCase())

function findSession(sessions: Session[], query: string): Session {
  const text = query.trim().replace(/^#?/, '')
  if (!text) throw 'Give an agent number or part of its title'
  if (/^\d+$/.test(text)) {
    const byNumber = sessions.find((s) => s.n === Number(text))
    if (byNumber) return byNumber
  }
  const matches = sessions.filter((s) => includes(s.title, query.trim()))
  if (matches.length === 1) return matches[0]
  if (matches.length > 1) throw `${matches.length} agents match "${query}": ${matches.map((s) => s.n).join(', ')}`
  throw `No agent matches "${query}"`
}

const describe = (s: Session) => `${s.n} ${s.title}`

export const sessionActions = ({ a, current }: ActionContext): Action[] => [
  {
    id: 'new-session',
    label: 'New session',
    group: 'Agents',
    shortcut: { key: 'n' },
    run() {
      a.setComposing('session')
    },
  },
  {
    id: 'new-chat',
    label: 'New chat',
    group: 'Agents',
    shortcut: { key: 'm' },
    run() {
      a.setComposing('chat')
    },
  },
  {
    id: 'open-session',
    label: 'Open agent',
    group: 'Agents',
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
    label: current ? `Detach agent ${describe(current)}` : 'Detach agent',
    group: 'Agents',
    shortcut: { key: 'w' },
    run() {
      if (!current) return
      a.detach()
      return `Detached ${describe(current)}`
    },
  },
  {
    id: 'next-attention',
    label: 'Next that needs you',
    group: 'Agents',
    shortcut: { key: 'e' },
    uiCommand: 'next',
    run() {
      a.nextAttention()
    },
  },
  {
    id: 'kill-session',
    label: current ? `Kill agent ${describe(current)}` : 'Kill agent',
    group: 'Agents',
    confirm: true,
    async run(args) {
      const target = args.length ? findSession(a.sessions, args.join(' ')) : current
      if (!target) throw 'No agent to kill'
      await a.killSession(target.id)
      return `Killed ${describe(target)}`
    },
  },
  {
    id: 'rename-session',
    label: current ? `Rename agent ${describe(current)}` : 'Rename agent',
    group: 'Agents',
    palette: { prompt: 'New title', initial: current?.title },
    async run(args) {
      if (!current) throw 'No agent to rename'
      if (args.length === 0) throw 'Give the new title: rename <title>'
      await a.renameSession(current.id, args.join(' '))
      return `Renamed to "${args.join(' ')}"`
    },
  },
  {
    id: 'open-pr',
    label: current?.pr ? `Open PR #${current.pr.number}` : 'Open the pull request',
    group: 'Agents',
    uiCommand: 'pr',
    async run() {
      if (!current) throw 'No agent selected'
      await a.openPR(current)
      return `Opened PR #${current.pr?.number}`
    },
  },
  {
    id: 'cleanup',
    label: current ? `Clean up session ${describe(current)}` : 'Clean up session',
    group: 'Agents',
    confirm: true,
    async run(args) {
      const target = args.length ? findSession(a.sessions, args.join(' ')) : current
      if (!target) throw 'No session to clean up'
      await a.cleanupSession(target.id, false)
      return `Cleaned up ${describe(target)}`
    },
  },
]
