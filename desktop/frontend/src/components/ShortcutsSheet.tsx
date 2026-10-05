import { useAgentos } from '../AgentosContext'
import { formatShortcut, LIST_KEYS, useActions } from '../actions'
import type { Action } from '../actions'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'

type Group = { title: string; ids: string[] }

const GROUPS: Group[] = [
  { title: 'Sessions', ids: ['new-session', 'open-session', 'goto-1', 'next-attention', 'previous-session', 'next-session', 'rename-session', 'kill-session', 'open-pr', 'cleanup', 'harness'] },
  { title: 'Navigation', ids: ['panel-left', 'panel-right', 'panel-down', 'panel-up', 'panel-sidebar', 'panel-terminal', 'panel-sessions', 'toggle-sidebar', 'toggle-sessions', 'switch-project', 'project', 'palette', 'shortcuts'] },
  { title: 'Queue', ids: ['show-queue', 'filter-queue', 'refresh-issues', 'start-issue'] },
  { title: 'Notes', ids: ['show-notes', 'note'] },
  { title: 'Views', ids: ['stats', 'digest', 'show-terminal', 'show-browser', 'show-evidence', 'view-previous', 'view-next', 'zoom-in', 'zoom-out', 'zoom-reset'] },
  { title: 'Command line', ids: ['focus-command', 'clear'] },
]

const listed = new Set(GROUPS.flatMap((g) => g.ids))

type EntryProps = { action: Action }

function Entry({ action }: EntryProps) {
  const keys = action.keysLabel ?? (action.shortcut ? formatShortcut(action.shortcut) : null)
  const title = action.keysLabel ? 'Open session n' : action.label
  return (
    <div className="flex min-h-11 items-start justify-between gap-3 border-b border-line py-2 last:border-b-0">
      <div className="min-w-0">
        {action.shortcut ? (
          <p className="text-body">{title}</p>
        ) : (
          <code className="mono block text-small text-accent">{action.command?.syntax}</code>
        )}
        {action.shortcut && action.command && <code className="mono block text-label text-dim">{action.command.syntax}</code>}
        {!action.shortcut && <p className="text-small text-dim">{action.command?.summary}</p>}
      </div>
      {keys && <Keycap>{keys}</Keycap>}
    </div>
  )
}

type BlockProps = { title: string; children: React.ReactNode }

function Block({ title, children }: BlockProps) {
  return (
    <section className="rounded-md border border-line bg-raised/40 px-3 pt-2.5 pb-1">
      <h3 className="label pb-1">{title}</h3>
      {children}
    </section>
  )
}

const LEFT = ['Sessions', 'Queue']
const RIGHT = ['Navigation', 'Notes', 'Views']

export function ShortcutsSheet() {
  const { overlay, setOverlay } = useAgentos()
  const actions = useActions()
  if (overlay !== 'shortcuts') return null

  const byId = new Map(actions.map((a) => [a.id, a]))
  const rows = (ids: string[]) => ids.map((id) => byId.get(id)).filter((a): a is Action => Boolean(a && (a.shortcut || a.command) && !a.hidden))
  const leftovers = actions.filter((a) => !listed.has(a.id) && (a.shortcut || a.command) && !a.hidden)
  const group = (title: string) => {
    const g = GROUPS.find((x) => x.title === title)!
    return (
      <Block key={title} title={title}>
        {rows(g.ids).map((a) => (
          <Entry key={a.id} action={a} />
        ))}
        {title === 'Queue' && (
          <div className="flex items-start justify-between gap-3 border-b border-line py-2 last:border-b-0">
            <p className="text-body">Start an issue without leaving the queue</p>
            <Keycap>⌥ click</Keycap>
          </div>
        )}
      </Block>
    )
  }

  return (
    <Overlay align="center" wide label="Keyboard shortcuts" onClose={() => setOverlay(null)}>
      <div className="flex flex-none items-center justify-between border-b border-line px-4 py-3">
        <h2 className="text-title font-semibold">Shortcuts and commands</h2>
        <Keycap>esc</Keycap>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
          <div className="flex min-w-0 flex-col gap-4">{LEFT.map(group)}</div>
          <div className="flex min-w-0 flex-col gap-4">{RIGHT.map(group)}</div>
          <div className="grid min-w-0 grid-cols-1 items-start gap-4 lg:col-span-2 lg:grid-cols-2">
            <Block title="Command line">
              {rows(GROUPS.find((g) => g.title === 'Command line')!.ids).map((a) => (
                <Entry key={a.id} action={a} />
              ))}
              <p className="py-2 text-small text-dim">Tab completes, ↑ ↓ walk the history, Esc returns to the terminal. Every command is also in the palette (⌘K).</p>
            </Block>
            <Block title="In a focused list">
              {LIST_KEYS.map((k) => (
                <div key={k.keys} className="border-b border-line py-2 last:border-b-0">
                  <Keycap>{k.keys}</Keycap>
                  <p className="pt-1 text-small text-dim">{k.summary}</p>
                </div>
              ))}
            </Block>
          </div>
          {leftovers.length > 0 && (
            <div className="lg:col-span-2">
              <Block title="More">
                {leftovers.map((a) => (
                  <Entry key={a.id} action={a} />
                ))}
              </Block>
            </div>
          )}
        </div>
      </div>
    </Overlay>
  )
}
