import { memo } from 'react'
import type { Project } from '../types'
import { ConfirmRow } from './ConfirmRow'
import { Icon } from './Icon'

type ProjectRowProps = {
  project: Project
  index: number
  current: boolean
  active: boolean
  removing: boolean
  onHover(index: number): void
  onChoose(name: string): void
  onAskRemove(name: string): void
  onCancelRemove(): void
  onRemove(name: string): void
}

function ProjectRowView({ project: p, index, current, active, removing, onHover, onChoose, onAskRemove, onCancelRemove, onRemove }: ProjectRowProps) {
  return (
    <div className="row row-pick group h-11 items-center pr-2 pl-4" data-selected={active} onMouseEnter={() => onHover(index)}>
      {removing ? (
        <ConfirmRow
          danger
          message={`Forget ${p.name}? Its sessions keep running.`}
          confirmLabel="Remove"
          onCancel={onCancelRemove}
          onConfirm={() => onRemove(p.name)}
        />
      ) : (
        <>
          <button className="flex h-full min-w-0 flex-1 items-center gap-3 text-left" onClick={() => onChoose(p.name)}>
            <span className="w-3 flex-none text-accent">{current && <Icon name="check" size={12} />}</span>
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
          <button
            className="btn btn-ghost ml-1 h-6 w-6 flex-none justify-center px-0 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100"
            aria-label={`Remove ${p.name}`}
            title="Remove project"
            onClick={() => onAskRemove(p.name)}
          >
            <Icon name="trash" size={13} />
          </button>
        </>
      )}
    </div>
  )
}

const sameProject = (a: Project, b: Project) =>
  a.name === b.name && a.dir === b.dir && a.needsYou === b.needsYou && a.working === b.working && a.sessions === b.sessions

export const ProjectRow = memo(
  ProjectRowView,
  (a, b) =>
    sameProject(a.project, b.project) &&
    a.index === b.index &&
    a.current === b.current &&
    a.active === b.active &&
    a.removing === b.removing &&
    a.onChoose === b.onChoose &&
    a.onRemove === b.onRemove,
)
