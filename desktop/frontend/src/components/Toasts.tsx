import { useAgentos } from '../AgentosContext'
import type { Toast } from '../AgentosContext'
import { Icon } from './Icon'
import { StateDot } from './StateDot'

const EDGE: Record<Toast['tone'], string> = {
  error: 'border-l-danger',
  pr: 'border-l-danger',
  waiting: 'border-l-waiting',
  replied: 'border-l-idle',
  info: 'border-l-accent',
  evidence: 'border-l-accent',
}

const ACTION_LABEL: Partial<Record<string, string>> = { replied: 'Open', evidence: 'View' }

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
        return (
          <div
            key={toast.key}
            className={`toast-in pointer-events-auto flex items-center gap-3 rounded-md border border-l-3 border-line-strong bg-raised py-2 pr-2 pl-3 ${EDGE[tone]}`}
          >
            {tone === 'waiting' && <StateDot state="waiting" />}
            {tone === 'replied' && <StateDot state="idle" />}
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
                {ACTION_LABEL[toast.tone] ?? 'Jump'}
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
