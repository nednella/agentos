import { useAgentos } from '../AgentosContext'
import { formatShortcut, LIST_KEYS, useActions } from '../actions'
import type { Action } from '../actions'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'

type Group = { title: string; ids: string[] }

const GROUPS: Group[] = [
  { title: 'Sessions', ids: ['new-session', 'open-session', 'goto-1', 'next-attention', 'previous-session', 'next-session', 'rename-session', 'stop-session', 'open-pr', 'cleanup', 'harness'] },
  { title: 'Navigation', ids: ['panel-left', 'panel-right', 'panel-down', 'panel-up', 'panel-terminal', 'panel-sessions', 'toggle-sidebar', 'toggle-sessions', 'switch-project', 'project', 'palette', 'shortcuts'] },
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
    <div className="flex items-start justify-between gap-3 border-b border-line py-2 last:border-b-0">
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
    <section className="mb-4 break-inside-avoid rounded-md border border-line bg-raised/40 px-3 pt-2.5 pb-1.5">
      <h3 className="label pb-1">{title}</h3>
      {children}
    </section>
  )
}

export function ShortcutsSheet() {
  const { overlay, setOverlay } = useAgentos()
  const actions = useActions()
  if (overlay !== 'shortcuts') return null

  const byId = new Map(actions.map((a) => [a.id, a]))
  const rows = (ids: string[]) => ids.map((id) => byId.get(id)).filter((a): a is Action => Boolean(a && (a.shortcut || a.command) && !a.hidden))
  const leftovers = actions.filter((a) => !listed.has(a.id) && (a.shortcut || a.command) && !a.hidden)

  return (
    <Overlay align="center" wide label="Keyboard shortcuts" onClose={() => setOverlay(null)}>
      <div className="flex flex-none items-center justify-between border-b border-line px-4 py-3">
        <h2 className="text-title font-semibold">Shortcuts and commands</h2>
        <Keycap>esc</Keycap>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <div className="columns-1 gap-4 md:columns-2 xl:columns-3">
          {GROUPS.map((group) => (
            <Block key={group.title} title={group.title}>
              {rows(group.ids).map((a) => (
                <Entry key={a.id} action={a} />
              ))}
              {group.title === 'Queue' && (
                <div className="flex items-start justify-between gap-3 border-b border-line py-2 last:border-b-0">
                  <p className="text-body">Start an issue without leaving the queue</p>
                  <Keycap>⌥ click</Keycap>
                </div>
              )}
              {group.title === 'Command line' && <p className="py-2 text-small text-dim">Tab completes, ↑ ↓ walk the history, Esc returns to the terminal. Every command above is also in the palette (⌘K).</p>}
            </Block>
          ))}
          <Block title="In a focused list">
            {LIST_KEYS.map((k) => (
              <div key={k.keys} className="border-b border-line py-2 last:border-b-0">
                <Keycap>{k.keys}</Keycap>
                <p className="pt-1 text-small text-dim">{k.summary}</p>
              </div>
            ))}
          </Block>
          {leftovers.length > 0 && (
            <Block title="More">
              {leftovers.map((a) => (
                <Entry key={a.id} action={a} />
              ))}
            </Block>
          )}
        </div>
      </div>
    </Overlay>
  )
}
