import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { Icon } from './Icon'
import { StateDot } from './StateDot'

type EdgeStripProps = { side: 'left' | 'right'; inShell?: boolean }

export function EdgeStrip({ side, inShell = false }: EdgeStripProps) {
  const { sessions, selectedId, select, issues, notes } = useAgentos()
  const { setSidebarOpen, setSessionsOpen, showSidebarTab } = useLayout()
  const left = side === 'left'

  return (
    <div
      className={`panel flex min-h-0 flex-col items-center gap-2 overflow-y-auto pt-1 pb-2 ${inShell ? 'edge-strip' : 'flex-none'}`}
      style={inShell ? undefined : { width: 'var(--edge-strip-w)' }}
    >
      <button
        className="btn btn-ghost h-8 w-8 flex-none justify-center px-0"
        title={left ? 'Show queue and notes (⌘B)' : 'Show sessions (⌘⌥B)'}
        aria-label={left ? 'Show queue and notes' : 'Show sessions'}
        onClick={() => (left ? setSidebarOpen(true) : setSessionsOpen(true))}
      >
        <Icon name={left ? 'panel-left' : 'panel-right'} size={15} />
      </button>
      {left ? (
        <>
          <button className="mono text-small text-soft hover:text-ink" title="Queue" onClick={() => showSidebarTab('queue')}>
            Q {issues.length}
          </button>
          <button className="mono text-small text-soft hover:text-ink" title="Notes" onClick={() => showSidebarTab('notes')}>
            N {notes.filter((n) => !n.archived).length}
          </button>
        </>
      ) : (
        sessions.map((s) => (
          <button
            key={s.id}
            className="flex w-full flex-col items-center gap-1 py-1"
            data-state={s.state}
            title={`${s.n} ${s.title}`}
            aria-label={`Open session ${s.n}, ${s.title}`}
            style={{ boxShadow: s.id === selectedId ? 'inset 2px 0 var(--accent)' : undefined }}
            onClick={() => select(s.id)}
          >
            <StateDot state={s.state} />
            <span className="mono text-label text-dim">{s.n}</span>
          </button>
        ))
      )}
    </div>
  )
}
