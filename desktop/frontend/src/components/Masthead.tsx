import { useAgentos } from '../AgentosContext'

export function Masthead() {
  const { version } = useAgentos()
  return (
    <>
      <h1 className="text-title font-semibold">
        agentos {version && <span className="mono text-small font-normal text-dim">{version}</span>}
      </h1>
      <p className="text-soft">Integrate your project's work queue seamlessly with coding-agent sessions</p>
    </>
  )
}
