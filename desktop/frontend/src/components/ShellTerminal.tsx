import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { useFocusRequest } from '../useFocusRequest'
import { Terminal } from './Terminal'

// The last focus request that opened a shell, so a remount after the shell exited does not replay it.
let openedFor = -1

type ShellTerminalProps = { visible: boolean }

export function ShellTerminal({ visible }: ShellTerminalProps) {
  const { shellId, openShell, report, focus, focusRequest, project } = useAgentos()
  const { returnToTerminal } = useLayout()

  useFocusRequest(
    'shell',
    () => {
      if (shellId || focusRequest.n <= openedFor) return
      openedFor = focusRequest.n
      report(openShell)
    },
    visible,
  )

  if (!shellId) {
    return (
      <button
        className="absolute inset-0 flex items-center gap-2 px-3 text-left text-small text-dim"
        onClick={() => focus('shell')}
      >
        <span className="mono text-accent">{project?.name} ❯</span>
        Click or press ⌘S to open a shell here
      </button>
    )
  }
  return <Terminal key={shellId} id={shellId} active={visible} kind="shell" onLeave={returnToTerminal} />
}
