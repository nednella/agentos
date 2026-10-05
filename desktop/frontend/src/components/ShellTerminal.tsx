import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { useFocusRequest } from '../useFocusRequest'
import { Terminal } from './Terminal'

type ShellTerminalProps = { visible: boolean }

export function ShellTerminal({ visible }: ShellTerminalProps) {
  const { shellId, openShell, report, focus, project } = useAgentos()
  const { returnToTerminal } = useLayout()

  useFocusRequest('shell', () => !shellId && report(openShell), visible)

  if (!shellId) {
    return (
      <button
        className="absolute inset-0 flex items-center gap-2 px-4 text-left text-small text-dim"
        onClick={() => {
          report(openShell)
          focus('shell')
        }}
      >
        <span className="mono text-accent">{project?.name} ❯</span>
        Click or press ⌘S to open a shell here
      </button>
    )
  }
  return <Terminal key={shellId} id={shellId} active={visible} kind="shell" onLeave={returnToTerminal} />
}
