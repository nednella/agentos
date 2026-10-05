import { useEffect, useRef } from 'react'
import { useAgentos } from '../AgentosContext'
import { useLayout } from '../LayoutContext'
import { Terminal } from './Terminal'

type ShellTerminalProps = { visible: boolean }

export function ShellTerminal({ visible }: ShellTerminalProps) {
  const { shellId, openShell, focusRequest, report, focus, project } = useAgentos()
  const { returnToTerminal } = useLayout()
  const handled = useRef(-1)

  useEffect(() => {
    if (!visible || focusRequest.n === handled.current) return
    handled.current = focusRequest.n
    if (focusRequest.target === 'shell' && !shellId) report(openShell)
  }, [visible, focusRequest, shellId, openShell, report])

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
