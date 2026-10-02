import { useAgentos } from '../AgentosContext'
import { Icon } from './Icon'
import { StateDot } from './StateDot'

export function Toasts() {
  const { toasts, select, dismissToast, setSessionView } = useAgentos()
  return (
    <div
      className="pointer-events-none fixed right-4 bottom-20 flex flex-col items-end gap-2"
      style={{ zIndex: 'var(--z-toast)' }}
      aria-live="polite"
    >
      {toasts.map((toast) => {
        const { sessionId } = toast
        const { tone } = toast
        const edge = { error: 'var(--danger)', info: 'var(--accent)', waiting: 'var(--waiting)', finished: 'var(--finished)', pr: 'var(--danger)', evidence: 'var(--accent)' }[tone]
        return (
          <div
            key={toast.key}
            className="toast-in pointer-events-auto flex items-center gap-3 rounded-md border border-line-strong bg-raised py-2 pr-2 pl-3"
            style={{ borderLeft: `3px solid ${edge}` }}
          >
            {(tone === 'waiting' || tone === 'finished') && <StateDot state={tone} />}
            {tone === 'pr' && <span className="mono text-small font-semibold text-danger">PR</span>}
            <span className="text-body">{toast.text}</span>
            {sessionId && (
              <button
                className="btn"
                onClick={() => {
                  select(sessionId)
                  if (toast.view) setSessionView(sessionId, toast.view)
                  dismissToast(toast.key)
                }}
              >
                {toast.view ? 'View' : 'Jump'}
              </button>
            )}
            <button className="btn btn-ghost w-7 justify-center px-0" aria-label="Dismiss" onClick={() => dismissToast(toast.key)}>
              <Icon name="close" size={12} />
            </button>
          </div>
        )
      })}
    </div>
  )
}
