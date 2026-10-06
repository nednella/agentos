import { useAgentos } from '../AgentosContext'
import type { Toast } from '../AgentosContext'
import { api, devFlags } from '../api'
import { Icon } from './Icon'
import { StateDot } from './StateDot'

const EDGE: Record<Toast['tone'], string> = {
  error: 'border-l-danger',
  pr: 'border-l-danger',
  waiting: 'border-l-waiting',
  replied: 'border-l-idle',
  opened: 'border-l-idle',
  info: 'border-l-accent',
  evidence: 'border-l-accent',
  done: 'border-l-finished',
}

const ACTION_LABEL: Partial<Record<string, string>> = { replied: 'Open', opened: 'Open', evidence: 'View' }

export function Toasts() {
  const { toasts, select, dismissToast, setSessionView } = useAgentos()
  return (
    <div
      className="pointer-events-none fixed inset-x-0 bottom-12 flex flex-col items-center gap-2"
      style={{ zIndex: 'var(--z-toast)' }}
      aria-live="polite"
    >
      {toasts.map((toast) => {
        if (toast.tone === 'done') return <DoneToast key={toast.key} toast={toast} dismiss={() => dismissToast(toast.key)} />
        const { sessionId } = toast
        const { tone } = toast
        return (
          <div
            key={toast.key}
            className={`toast-in pointer-events-auto flex items-center gap-3 rounded-md border border-l-3 border-line-strong bg-raised py-2 pr-2 pl-3 ${EDGE[tone]}`}
          >
            {tone === 'waiting' && <StateDot state="waiting" />}
            {(tone === 'replied' || tone === 'opened') && <StateDot state="idle" />}
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
            <DismissButton onClick={() => dismissToast(toast.key)} />
          </div>
        )
      })}
    </div>
  )
}

type DoneToastProps = { toast: Toast; dismiss: () => void }

// Three candidates for issue #70; `?done=a|b|c` picks one in the mock. One stays once Ned chooses.
function DoneToast({ toast, dismiss }: DoneToastProps) {
  const variant = devFlags.done ?? 'a'
  const openPR = toast.url && (
    <button className="btn btn-finished" onClick={() => void api.openURL(toast.url!)}>
      <Icon name="external" size={12} />
      PR
    </button>
  )

  if (variant === 'b')
    return (
      <div className="toast-rise pointer-events-auto flex items-center gap-3 rounded-md border border-finished bg-finished-tint py-2 pr-2 pl-3">
        <span className="toast-pop flex h-6 w-6 flex-none items-center justify-center rounded-full bg-finished text-app">
          <Icon name="check" size={14} />
        </span>
        <span className="flex flex-col leading-tight">
          <span className="text-label font-semibold tracking-wide text-finished uppercase">Shipped</span>
          <span className="text-body">{toast.text}</span>
        </span>
        {openPR}
        <DismissButton onClick={dismiss} />
      </div>
    )

  if (variant === 'c')
    return (
      <div className="toast-rise pointer-events-auto relative flex items-center gap-4 overflow-hidden rounded-md border border-finished bg-raised py-3 pr-3 pl-4">
        <span className="toast-burst absolute top-1/2 left-7 h-0 w-0" aria-hidden="true">
          {[...Array(8)].map((_, i) => (
            <i key={i} style={{ ['--i' as string]: i }} />
          ))}
        </span>
        <span className="toast-pop relative flex h-9 w-9 flex-none items-center justify-center rounded-full bg-finished text-app">
          <Icon name="check" size={20} />
        </span>
        <span className="flex flex-col gap-0.5">
          <span className="text-title font-semibold text-finished">Nice one, that's shipped</span>
          <span className="text-small text-soft">
            {toast.text} · {toast.detail}
          </span>
        </span>
        {openPR}
        <DismissButton onClick={dismiss} />
      </div>
    )

  return (
    <div className={`toast-in pointer-events-auto flex items-center gap-3 rounded-md border border-l-3 border-line-strong bg-raised py-2 pr-2 pl-3 ${EDGE.done}`}>
      <span className="flex h-5 w-5 flex-none items-center justify-center rounded-full border-[1.5px] border-finished text-finished">
        <Icon name="check" size={11} />
      </span>
      <span className="flex flex-col leading-tight">
        <span className="text-body font-semibold">{toast.text}</span>
        <span className="text-small text-soft">{toast.detail}</span>
      </span>
      {openPR}
      <DismissButton onClick={dismiss} />
    </div>
  )
}

type DismissButtonProps = { onClick: () => void }

function DismissButton({ onClick }: DismissButtonProps) {
  return (
    <button className="btn btn-ghost w-7 justify-center px-0" aria-label="Dismiss" onClick={onClick}>
      <Icon name="close" size={12} />
    </button>
  )
}
