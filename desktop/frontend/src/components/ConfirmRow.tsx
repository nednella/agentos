import type { ReactNode } from 'react'

type ConfirmRowProps = {
  message: ReactNode
  label?: string
  confirmLabel: string
  cancelLabel?: string
  danger?: boolean
  inline?: boolean
  onConfirm(): void
  onCancel(): void
}

export function ConfirmRow({ message, label = 'Confirm', confirmLabel, cancelLabel = 'Cancel', danger = false, inline = false, onConfirm, onCancel }: ConfirmRowProps) {
  return (
    <div
      className={`flex items-center gap-2 ${inline ? '' : 'w-full'}`}
      role="alertdialog"
      aria-label={label}
      onKeyDown={(e) => {
        if (e.key !== 'Escape') return
        e.stopPropagation()
        onCancel()
      }}
    >
      <span className={`min-w-0 text-small ${inline ? '' : 'flex-1'}`} style={{ color: danger ? 'var(--danger)' : 'var(--text-soft)' }}>
        {message}
      </span>
      <span className="ml-auto flex flex-none items-center gap-1.5">
        <button className="btn h-6 px-2" onClick={onCancel}>
          {cancelLabel}
        </button>
        <button className={`btn h-6 px-2 ${danger ? 'btn-danger' : 'btn-accent'}`} autoFocus onClick={onConfirm}>
          {confirmLabel}
        </button>
      </span>
    </div>
  )
}
