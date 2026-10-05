import { useAgentos } from './AgentosContext'
import { formatShortcut, useActions } from './actions'
import { fuzzyScore } from './fuzzy'
import { useLayout } from './LayoutContext'
import { matchIssue, parseQuery } from './issueFilter'
import type { State } from './types'

export type PaletteItem = {
  key: string
  group: 'Sessions' | 'Actions' | 'Projects' | 'Notes' | 'Issues'
  label: string
  hint: string
  state?: State
  keys?: string
  confirm?: boolean
  prompt?: { label: string; initial: string }
  run(args: string[]): unknown
}

const ISSUE_LIMIT = 30

export function usePaletteItems(query: string): PaletteItem[] {
  const a = useAgentos()
  const actions = useActions()
  const layout = useLayout()
  const parsed = parseQuery(query)

  const sessions: PaletteItem[] = a.sessions
    .filter((s) => s.state !== 'ended')
    .map((s) => ({
    key: `s-${s.id}`,
    group: 'Sessions',
    label: `${s.n}  ${s.title}`,
    hint: s.detail,
    state: s.state,
    keys: s.n <= 9 ? `⌘${s.n}` : undefined,
    run: () => a.select(s.id),
  }))

  const commands: PaletteItem[] = actions
    .filter((action) => action.palette !== false && !action.hidden)
    .map((action) => ({
      key: `a-${action.id}`,
      group: 'Actions',
      label: action.label,
      hint: '',
      keys: action.shortcut ? formatShortcut(action.shortcut) : undefined,
      confirm: action.confirm,
      prompt: action.palette ? { label: action.palette.prompt ?? '', initial: action.palette.initial ?? '' } : undefined,
      run: (args) => action.run(args),
    }))

  const projects: PaletteItem[] = a.projects
    .filter((p) => p.name !== a.project?.name)
    .map((p) => ({
      key: `p-${p.name}`,
      group: 'Projects',
      label: `Switch to ${p.name}`,
      hint: p.needsYou > 0 ? `${p.needsYou} need you` : `${p.sessions} sessions`,
      run: () => a.switchProject(p.name),
    }))

  const notes: PaletteItem[] = a.notes
    .filter((n) => !n.archived)
    .map((n) => ({
      key: `n-${n.id}`,
      group: 'Notes',
      label: n.text.split('\n')[0],
      hint: 'note',
      run: () => layout.showSidebarTab('notes'),
    }))

  const issues: PaletteItem[] = a.issues
    .filter((i) => !i.sessionId && matchIssue(i, parsed))
    .slice(0, ISSUE_LIMIT)
    .map((i) => ({
      key: `i-${i.number}`,
      group: 'Issues',
      label: `#${i.number} ${i.title}`,
      hint: `start · ${i.lane}`,
      run: () => a.startIssue(i.number),
    }))

  if (parsed.tokens.length > 0) return issues
  const rest = [...sessions, ...commands, ...projects, ...notes]
  if (!query.trim()) return [...rest.filter((i) => i.group !== 'Notes'), ...issues.slice(0, 6)]
  return [
    ...rest
      .map((item) => ({ item, score: fuzzyScore(query.trim(), item.label) }))
      .filter((r) => r.score > 0)
      .sort((x, y) => y.score - x.score)
      .map((r) => r.item),
    ...issues,
  ]
}
