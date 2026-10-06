import { useAgentos } from '../AgentosContext'
import { Icon } from './Icon'

export function ShellList() {
  const { shellIds, shellId, selectShell, newShell, closeShell, report, focus } = useAgentos()

  return (
    <div className="flex w-36 flex-none flex-col overflow-y-auto border-l border-line bg-surface py-1" role="tablist" aria-label="Shells" aria-orientation="vertical">
      <div className="flex h-6 items-center justify-between px-3 text-label text-dim">
        <span className="uppercase tracking-wide">Shells</span>
        <button
          className="hover:text-text"
          aria-label="New shell"
          title="New shell"
          onClick={() => {
            report(newShell)
            focus('shell')
          }}
        >
          <Icon name="plus" size={12} />
        </button>
      </div>
      {shellIds.map((id, i) => {
        const active = id === shellId
        return (
          <div
            key={id}
            className="group flex h-6 items-center justify-between pr-3 text-small"
            style={{
              background: active ? 'var(--bg-hover)' : undefined,
              boxShadow: active ? 'inset 2px 0 0 var(--accent)' : undefined,
              color: active ? 'var(--text)' : 'var(--text-soft)',
            }}
          >
            <button
              role="tab"
              aria-selected={active}
              className="h-full flex-1 pl-3 text-left"
              onClick={() => {
                selectShell(id)
                focus('shell')
              }}
            >
              Shell {i + 1}
            </button>
            {shellIds.length > 1 && (
              <button
                className="text-dim opacity-0 hover:text-text group-hover:opacity-100 focus-visible:opacity-100"
                aria-label={`Close shell ${i + 1}`}
                onClick={() => report(() => closeShell(id))}
              >
                <Icon name="close" size={12} />
              </button>
            )}
          </div>
        )
      })}
    </div>
  )
}
