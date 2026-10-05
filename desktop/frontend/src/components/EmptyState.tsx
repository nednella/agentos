import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { Keycap } from './Keycap'

export function EmptyState() {
  const { project, report, newSession, issues } = useAgentos()
  const { showSidebarTab } = useLayout()
  return (
    <div className="grid h-full place-items-center">
      <div className="flex max-w-sm flex-col items-center gap-4 text-center">
        <h1 className="text-title font-semibold">No sessions in {project?.name ?? 'this project'}</h1>
        <p className="text-soft">Each session is a Claude Code terminal. The ones that need you show up on the right.</p>
        <button className="btn btn-accent h-9 px-4 text-body" onClick={() => report(() => newSession())}>
          Start a session
        </button>
        <span className="flex items-center gap-1.5 text-small text-dim">
          or press <Keycap>⌘N</Keycap>
          {issues.length > 0 && (
            <>
              , or start one from the{' '}
              <button className="text-accent underline" onClick={() => showSidebarTab('queue')}>
                queue
              </button>
            </>
          )}
        </span>
      </div>
    </div>
  )
}
