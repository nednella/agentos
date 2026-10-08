import { useAgentos } from '../AgentosContext'

export function SetupOffer() {
  const { setUpProject, dismissSetup, report } = useAgentos()
  return (
    <div className="mx-3 mt-3 mb-1 flex flex-col gap-2.5 rounded-md border border-line bg-raised p-3">
      <div className="flex flex-col gap-1">
        <p className="text-body font-medium">Set up this project for agentos</p>
        <p className="text-small text-dim">
          Adds an Inbox whose Work action runs <code className="mono">/work</code>, then starts a session to write the command, the issue template and CLAUDE.md with you.
        </p>
      </div>
      <div className="flex justify-end gap-1.5">
        <button className="btn btn-ghost h-7 px-2.5" onClick={() => report(dismissSetup)}>
          Not now
        </button>
        <button className="btn btn-accent h-7 px-2.5" onClick={() => report(setUpProject)}>
          Set up
        </button>
      </div>
    </div>
  )
}
