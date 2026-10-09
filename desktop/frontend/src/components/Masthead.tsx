import { useAgentos } from '../AgentosContext'

export function Masthead() {
  const { version, setOverlay } = useAgentos()
  return (
    <>
      <h1 className="text-title font-semibold">
        agentos {version && (
          <button className="mono text-small font-normal text-dim hover:text-ink" title="Patch notes" onClick={() => setOverlay('changelog')}>
            {version}
          </button>
        )}
      </h1>
      <p className="text-soft">One desktop window for every coding-agent session you run</p>
    </>
  )
}
