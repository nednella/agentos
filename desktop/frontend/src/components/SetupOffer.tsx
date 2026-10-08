import { useAgentos } from '../AgentosContext'

export function SetupOffer() {
  const { setUpProject, dismissSetup, report } = useAgentos()
  return (
    <div className="flex items-center gap-2 border-b border-line px-3 py-2">
      <span className="min-w-0 flex-1 text-small text-soft">Set up this project for agentos?</span>
      <button className="btn h-6 px-2" onClick={() => report(dismissSetup)}>
        Not now
      </button>
      <button className="btn btn-accent h-6 px-2" onClick={() => report(setUpProject)}>
        Set up
      </button>
    </div>
  )
}
