import { useMemo, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { ConfirmRow } from './ConfirmRow'
import { Icon } from './Icon'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'

export function ProjectPanel() {
  const { overlay } = useAgentos()
  if (overlay !== 'projects') return null
  return <ProjectPanelContent />
}

function ProjectPanelContent() {
  const { project, projects, switchProject, addProject, removeProject, setOverlay, report } = useAgentos()
  const [query, setQuery] = useState('')
  const [cursor, setCursor] = useState(Math.max(0, projects.findIndex((p) => p.name === project?.name)))
  const [removing, setRemoving] = useState<string | null>(null)

  const shown = useMemo(() => projects.filter((p) => p.name.toLowerCase().includes(query.toLowerCase())), [projects, query])
  const close = () => setOverlay(null)

  const choose = (name: string) => {
    close()
    report(() => switchProject(name))
  }

  return (
    <Overlay align="left" label="Projects" onClose={close}>
      <div className="flex items-center gap-2 border-b border-line px-3 py-2">
        <Icon name="search" />
        <input
          autoFocus
          value={query}
          placeholder="Switch project"
          className="min-w-0 flex-1 bg-transparent py-1 outline-none"
          onChange={(e) => {
            setQuery(e.target.value)
            setCursor(0)
          }}
          onKeyDown={(e) => {
            if (e.key === 'ArrowDown') {
              e.preventDefault()
              setCursor((c) => Math.min(c + 1, shown.length - 1))
            }
            if (e.key === 'ArrowUp') {
              e.preventDefault()
              setCursor((c) => Math.max(c - 1, 0))
            }
            if (e.key === 'Enter' && shown[cursor]) choose(shown[cursor].name)
          }}
        />
        <Keycap>esc</Keycap>
      </div>
      <div className="max-h-80 overflow-y-auto py-1">
        {shown.length === 0 && <p className="px-4 py-4 text-small text-dim">No project matches.</p>}
        {shown.map((p, i) => (
          <div key={p.name} className="row group h-11 items-center pr-2 pl-4" data-selected={i === cursor} onMouseEnter={() => setCursor(i)}>
            {removing === p.name ? (
              <>
                <ConfirmRow
                  danger
                  message={`Forget ${p.name}? Its sessions keep running.`}
                  confirmLabel="Remove"
                  onCancel={() => setRemoving(null)}
                  onConfirm={() => {
                    setRemoving(null)
                    report(() => removeProject(p.name))
                  }}
                />
              </>
            ) : (
              <>
                <button className="flex h-full min-w-0 flex-1 items-center gap-3 text-left" onClick={() => choose(p.name)}>
                  <span className="w-3 flex-none text-accent">{p.name === project?.name && <Icon name="check" size={12} />}</span>
                  <span className="min-w-0">
                    <span className="block truncate text-body font-medium">{p.name}</span>
                    <span className="mono block truncate text-label text-dim">{p.dir}</span>
                  </span>
                  <span className="mono ml-auto flex flex-none gap-3 text-small">
                    <span style={{ color: p.needsYou > 0 ? 'var(--waiting)' : 'var(--text-dim)' }} title="need you">
                      {p.needsYou} need
                    </span>
                    <span style={{ color: p.working > 0 ? 'var(--working)' : 'var(--text-dim)' }} title="working">
                      {p.working} work
                    </span>
                    <span className="text-dim" title="sessions">
                      {p.sessions} total
                    </span>
                  </span>
                </button>
                <span className="reveal">
                  <button
                    className="btn btn-ghost ml-1 h-6 w-6 justify-center px-0"
                    aria-label={`Remove ${p.name}`}
                    title="Remove project"
                    onClick={() => setRemoving(p.name)}
                  >
                    <Icon name="trash" size={13} />
                  </button>
                </span>
              </>
            )}
          </div>
        ))}
      </div>
      <button
        className="row h-10 items-center gap-2 border-t border-line px-4 text-soft"
        onClick={() => {
          close()
          report(() => addProject())
        }}
      >
        <Icon name="plus" />
        Add project…
      </button>
    </Overlay>
  )
}
