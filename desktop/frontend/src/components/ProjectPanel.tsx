import { useCallback, useMemo, useRef, useState } from 'react'
import { useAgentos } from '../AgentosContext'
import { useArmedConfirm } from '../useArmedConfirm'
import { ProjectRow } from './ProjectRow'
import { Icon } from './Icon'
import { InlineInput } from './InlineInput'
import { Keycap } from './Keycap'
import { Overlay } from './Overlay'

export function ProjectPanel() {
  const { overlay } = useAgentos()
  if (overlay !== 'projects') return null
  return <ProjectPanelContent />
}

function ProjectPanelContent() {
  const { project, projects, switchProject, addProject, newProject, removeProject, setOverlay, report } = useAgentos()
  const [query, setQuery] = useState('')
  const [cursor, setCursor] = useState(Math.max(0, projects.findIndex((p) => p.name === project?.name)))
  const [naming, setNaming] = useState(false)
  const removal = useArmedConfirm<string>()

  const shown = useMemo(() => projects.filter((p) => p.name.toLowerCase().includes(query.toLowerCase())), [projects, query])
  const close = () => setOverlay(null)

  const latest = useRef({ switchProject, removeProject, report, setOverlay, disarm: removal.disarm })
  latest.current = { switchProject, removeProject, report, setOverlay, disarm: removal.disarm }
  const choose = useCallback((name: string) => {
    latest.current.setOverlay(null)
    latest.current.report(() => latest.current.switchProject(name))
  }, [])
  const remove = useCallback((name: string) => {
    latest.current.disarm()
    latest.current.report(() => latest.current.removeProject(name))
  }, [])

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
        <Keycap>Esc</Keycap>
      </div>
      <div className="max-h-80 overflow-y-auto py-1">
        {shown.length === 0 && <p className="px-4 py-4 text-small text-dim">No project matches.</p>}
        {shown.map((p, i) => (
          <ProjectRow
            key={p.name}
            project={p}
            index={i}
            current={p.name === project?.name}
            active={i === cursor}
            removing={removal.armed === p.name}
            onHover={setCursor}
            onChoose={choose}
            onAskRemove={removal.arm}
            onCancelRemove={removal.disarm}
            onRemove={remove}
          />
        ))}
      </div>
      {naming ? (
        <div className="border-t border-line p-3">
          <InlineInput
            placeholder="Name of the new project"
            blur="cancel"
            action="Create"
            icon={<Icon name="plus" />}
            onSubmit={(name) => {
              close()
              report(() => newProject(name))
            }}
            onCancel={() => setNaming(false)}
          />
        </div>
      ) : (
        <div className="flex border-t border-line">
          <button
            className="row h-10 flex-1 items-center gap-2 px-4 text-soft"
            onClick={() => {
              close()
              report(() => addProject())
            }}
          >
            <Icon name="plus" />
            Add project…
          </button>
          <button className="row h-10 flex-1 items-center gap-2 border-l border-line px-4 text-soft" onClick={() => setNaming(true)}>
            <Icon name="plus" />
            New project…
          </button>
        </div>
      )}
    </Overlay>
  )
}
