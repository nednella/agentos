import type { CSSProperties } from 'react'
import { useAgentos } from '../AgentosContext'
import { useActions, formatShortcut } from '../actions'
import { useLayout } from '../LayoutContext'
import type { State } from '../types'
import { Icon } from './Icon'
import { Keycap } from './Keycap'
import { StateDot } from './StateDot'

const drag = { '--wails-draggable': 'drag' } as CSSProperties
const noDrag = { '--wails-draggable': 'no-drag' } as CSSProperties

const FULL_MIN = 1100
const DOTS_MIN = 640
const NEXT_FULL_MIN = 780
const HELP_MIN = 560
const MORE_MIN = 480

type CountProps = { state: State; count: number; label: string; condensed: boolean }

function Count({ state, count, label, condensed }: CountProps) {
  return (
    <span className="flex items-center gap-1.5 text-body" style={{ opacity: count > 0 ? 1 : 0.5 }} title={`${count} ${label}`}>
      <StateDot state={state} />
      <span className="mono font-semibold" style={{ color: count > 0 ? 'var(--c)' : undefined }} data-state={state}>
        {count}
      </span>
      {!condensed && <span className="text-soft">{label}</span>}
    </span>
  )
}

export function TopBar() {
  const { project, projects, sessions, setOverlay, digestUnseen, report } = useAgentos()
  const { width, statsOpen, digestOpen } = useLayout()
  const actions = useActions()
  const count = (state: State) => sessions.filter((s) => s.state === state).length
  const waiting = count('waiting')
  const idle = count('idle')
  const othersNeedYou = projects.some((p) => p.name !== project?.name && p.needsYou > 0)
  const find = (id: string) => actions.find((a) => a.id === id)!
  const keys = (id: string) => formatShortcut(find(id).shortcut!)
  const iconButton = 'btn btn-ghost h-8 w-8 justify-center px-0'
  const full = width >= FULL_MIN
  const badge = width < DOTS_MIN

  return (
    <header
      className="relative flex flex-none items-center border-b border-line"
      style={{ ...drag, height: 'var(--topbar-h)', paddingLeft: 'var(--traffic-light-zone)', paddingRight: 'var(--corner-pad)' }}
    >
      <div className="flex min-w-0 items-center gap-3" style={noDrag}>
        {badge ? (
          <span
            className="mono grid h-6 min-w-6 place-items-center rounded-sm border px-1.5 text-small font-semibold"
            data-state="waiting"
            style={{ color: waiting > 0 ? 'var(--c)' : 'var(--text-dim)', borderColor: waiting > 0 ? 'var(--c)' : 'var(--border-strong)' }}
            title={`${waiting} need you, ${count('working')} working, ${idle} idle`}
          >
            {waiting}
          </span>
        ) : (
          <>
            <Count state="waiting" count={waiting} label="need you" condensed={!full} />
            <Count state="working" count={count('working')} label="working" condensed={!full} />
            <Count state="idle" count={idle} label="idle" condensed={!full} />
          </>
        )}
      </div>
      <div className="absolute top-1/2 left-1/2 flex -translate-x-1/2 -translate-y-1/2 items-center gap-1" style={noDrag}>
        <button
          className="btn btn-ghost h-8 min-w-0 gap-2 px-2 text-body font-semibold text-ink"
          aria-haspopup="dialog"
          title={`Switch project (${keys('switch-project')})`}
          onClick={() => setOverlay('projects')}
        >
          {width >= 520 && <Icon name="folder" />}
          <span className={`truncate ${width < 520 ? 'max-w-[5.5rem]' : 'max-w-[10rem]'}`}>{project?.name ?? '…'}</span>
          {othersNeedYou && <span className="dot" data-state="waiting" title="Another project needs you" />}
          <Icon name="chevron" size={12} />
        </button>
        {project?.repo && (
          <button className={iconButton} title={`Open ${project.repo} on GitHub`} aria-label="Open repository on GitHub" onClick={() => report(() => find('open-repo').run([]))}>
            <Icon name="external" />
          </button>
        )}
      </div>
      <div className="ml-auto flex items-center justify-end gap-1" style={noDrag}>
        {width >= NEXT_FULL_MIN ? (
          <button className="btn" onClick={() => find('next-attention').run([])}>
            Next
            <Keycap>{keys('next-attention')}</Keycap>
          </button>
        ) : (
          <button className={iconButton} title={`Next that needs you (${keys('next-attention')})`} aria-label="Next that needs you" onClick={() => find('next-attention').run([])}>
            <Icon name="next" />
          </button>
        )}
        {width >= MORE_MIN && (
          <>
            <button
              className={iconButton}
              style={{ color: statsOpen ? 'var(--accent)' : undefined }}
              title={`Stats (${keys('stats')})`}
              aria-label="Stats"
              aria-pressed={statsOpen}
              onClick={() => find('stats').run([])}
            >
              <Icon name="chart" />
            </button>
            <button
              className={`${iconButton} relative`}
              style={{ color: digestOpen ? 'var(--accent)' : undefined }}
              title={`Weekly digest (${keys('digest')})${digestUnseen ? ', new items' : ''}`}
              aria-label={digestUnseen ? 'Weekly digest, new items' : 'Weekly digest'}
              aria-pressed={digestOpen}
              onClick={() => find('digest').run([])}
            >
              <Icon name="digest" />
              {digestUnseen && <span className="dot absolute top-1.5 right-1.5" style={{ ['--c' as string]: 'var(--accent)' }} />}
            </button>
          </>
        )}
        <button className={iconButton} onClick={() => setOverlay('palette')} title={`Command palette (${keys('palette')})`} aria-label="Command palette">
          <Icon name="search" />
        </button>
        {width >= HELP_MIN && (
          <button className={iconButton} onClick={() => setOverlay('shortcuts')} title={`Keyboard shortcuts (${keys('shortcuts')})`} aria-label="Keyboard shortcuts">
            <Icon name="help" />
          </button>
        )}
      </div>
    </header>
  )
}
