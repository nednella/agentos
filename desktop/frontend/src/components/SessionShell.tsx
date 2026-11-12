import { useRef } from 'react'
import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'
import { useFocusRequest } from '../useFocusRequest'
import { Terminal } from './Terminal'

type SessionShellProps = { session: Session }

export function SessionShell({ session }: SessionShellProps) {
  const { project, openSessionShell, setSessionView, report } = useAgentos()
  const hint = useRef<HTMLButtonElement>(null)
  useFocusRequest('session-shell', hint, !session.shell)

  if (!session.shell) {
    return (
      <button
        ref={hint}
        className="absolute inset-0 flex items-center gap-2 px-3 text-left text-small text-dim"
        onClick={() => report(() => openSessionShell(session.id))}
      >
        <span className="mono truncate text-accent">{session.worktree || project?.dir} ❯</span>
        <span className="flex-none">Click or press Enter to open a shell here</span>
      </button>
    )
  }
  return <Terminal key={session.shell} id={session.shell} active kind="session-shell" onLeave={() => setSessionView(session.id, 'terminal')} />
}
