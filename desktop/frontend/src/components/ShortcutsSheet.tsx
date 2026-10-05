import { useMemo, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { formatShortcut, LIST_KEYS, useActions } from '../actions'
import type { Action } from '../actions'
import { Icon } from './Icon'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'

type Row = { id: string; label: string; keys?: string; mono?: boolean }
type Group = { title: string; ids: string[]; extra?: Row[] }

const GROUPS: Group[] = [
  { title: 'Sessions', ids: ['new-session', 'goto-1', 'next-attention', 'previous-session', 'next-session', 'rename-session', 'kill-session', 'open-pr', 'cleanup', 'harness'] },
  { title: 'Navigate', ids: ['panel-sidebar', 'panel-sessions', 'toggle-sidebar', 'toggle-sessions', 'switch-project', 'palette', 'shortcuts'] },
  { title: 'Queue', ids: ['show-queue', 'filter-queue', 'refresh-issues'], extra: [{ id: 'alt-click', label: 'Start without leaving the queue', keys: '⌥ click' }] },
  { title: 'Notes', ids: ['show-notes', 'note'] },
  { title: 'Views', ids: ['stats', 'digest', 'show-terminal', 'show-browser', 'show-evidence', 'view-previous', 'view-next', 'zoom-in', 'zoom-out', 'zoom-reset'] },
  {
    title: 'Shell',
    ids: [],
    extra: [
      { id: 'shell-keys', label: 'Back to the terminal', keys: 'Esc Esc' },
      { id: 'shell-ex-1', label: 'agentos issue 394 393', mono: true },
      { id: 'shell-ex-2', label: 'agentos open 2', mono: true },
      { id: 'shell-ex-3', label: 'agentos filter @nednella type:bug', mono: true },
      { id: 'shell-ex-4', label: 'agentos next', mono: true },
    ],
  },
  { title: 'In a list', ids: [], extra: LIST_KEYS.map((k) => ({ id: k.keys, label: k.summary, keys: k.keys })) },
]

const LEFT = ['Sessions', 'Queue', 'In a list']
const RIGHT = ['Navigate', 'Notes', 'Views', 'Shell']
const listed = new Set(GROUPS.flatMap((g) => g.ids))

function toRow(action: Action): Row {
  const keys = action.keysLabel ?? (action.shortcut ? formatShortcut(action.shortcut) : undefined)
  return { id: action.id, label: action.keysLabel ? 'Open session n' : action.label, keys }
}

const matches = (row: Row, query: string) => [row.label, row.keys].some((text) => text?.toLowerCase().includes(query))

type RowViewProps = { row: Row }

function RowView({ row }: RowViewProps) {
  return (
    <li className="flex h-7 items-center gap-3 border-b border-line last:border-b-0">
      <span className={`min-w-0 flex-1 truncate ${row.mono ? 'mono text-small text-accent' : 'text-body'}`}>{row.label}</span>
      {row.keys && <Keycap>{row.keys}</Keycap>}
    </li>
  )
}

export function ShortcutsSheet() {
  const { overlay, setOverlay } = useAgentos()
  const actions = useActions()
  const [query, setQuery] = useState('')

  const groups = useMemo(() => {
    const byId = new Map(actions.map((a) => [a.id, a]))
    const needle = query.trim().toLowerCase()
    const build = (group: Group) => {
      const rows = [
        ...group.ids.map((id) => byId.get(id)).filter((a): a is Action => Boolean(a && a.shortcut && !a.hidden)).map(toRow),
        ...(group.extra ?? []),
      ]
      return { title: group.title, rows: needle ? rows.filter((r) => matches(r, needle)) : rows }
    }
    const all = GROUPS.map(build)
    const leftovers = actions.filter((a) => !listed.has(a.id) && a.shortcut && !a.hidden).map(toRow)
    if (leftovers.length > 0) all.push({ title: 'More', rows: needle ? leftovers.filter((r) => matches(r, needle)) : leftovers })
    return all.filter((g) => g.rows.length > 0)
  }, [actions, query])

  if (overlay !== 'shortcuts') return null

  const column = (titles: string[]) => groups.filter((g) => titles.includes(g.title) || (titles === RIGHT && g.title === 'More'))
  const blocks = (list: typeof groups) =>
    list.map((g) => (
      <section key={g.title} className="min-w-0">
        <h3 className="label pb-0.5">{g.title}</h3>
        {g.title === 'Shell' && <p className="pb-1 text-small text-dim">A zsh in the project folder. Run agentos &lt;command&gt; here, or in any terminal, to drive the app.</p>}
        <ul>
          {g.rows.map((row) => (
            <RowView key={row.id} row={row} />
          ))}
        </ul>
      </section>
    ))

  return (
    <Overlay align="center" wide label="Keyboard shortcuts" onClose={() => setOverlay(null)}>
      <div className="flex flex-none items-center gap-3 border-b border-line px-4 py-2.5">
        <h2 className="flex-none text-title font-semibold">Shortcuts</h2>
        <div className="relative min-w-0 flex-1">
          <span className="pointer-events-none absolute top-2 left-2 text-dim">
            <Icon name="search" size={13} />
          </span>
          <input
            autoFocus
            value={query}
            spellCheck={false}
            aria-label="Filter shortcuts"
            placeholder="Filter shortcuts and commands"
            className="field w-full pl-7 text-small"
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <Keycap>esc</Keycap>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-2">
        {groups.length === 0 && <p className="py-8 text-center text-small text-dim">Nothing matches.</p>}
        <div className="grid grid-cols-1 items-start gap-x-10 gap-y-4 lg:grid-cols-2">
          <div className="flex min-w-0 flex-col gap-4">{blocks(column(LEFT))}</div>
          <div className="flex min-w-0 flex-col gap-4">{blocks(column(RIGHT))}</div>
        </div>
      </div>
    </Overlay>
  )
}
