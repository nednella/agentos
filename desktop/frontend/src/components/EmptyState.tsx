import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { Icon } from './Icon'
import { Keycap } from './Keycap'
import { Masthead } from './Masthead'

const card = 'flex flex-1 flex-col items-start gap-2 rounded-md border border-line bg-raised p-4 text-left hover:bg-hover'

export function EmptyState() {
  const { report, newSession, newChat, issues } = useAgentos()
  const { showSidebarTab } = useLayout()
  return (
    <div className="grid h-full place-items-center px-6">
      <div className="flex w-full max-w-lg flex-col items-center gap-6 text-center">
        <div className="flex flex-col items-center gap-1">
          <Masthead />
        </div>
        <div className="flex w-full gap-3">
          <button className={card} onClick={() => report(() => newChat())}>
            <span className="flex w-full items-center gap-2 text-body font-semibold text-ink">
              <Icon name="chat" />
              Chat
              <span className="ml-auto">
                <Keycap>⌘M</Keycap>
              </span>
            </span>
            <span className="text-small text-soft">Reads the project and answers questions.</span>
          </button>
          <button className={card} onClick={() => report(() => newSession())}>
            <span className="flex w-full items-center gap-2 text-body font-semibold text-ink">
              <Icon name="branch" />
              Task
              <span className="ml-auto">
                <Keycap>⌘N</Keycap>
              </span>
            </span>
            <span className="text-small text-soft">Works on the project in its own branch and opens a pull request.</span>
          </button>
        </div>
        {issues.length > 0 && (
          <span className="text-small text-dim">
            Or start a task from an issue in the{' '}
            <button className="text-accent underline" onClick={() => showSidebarTab('queue')}>
              queue
            </button>
          </span>
        )}
      </div>
    </div>
  )
}
