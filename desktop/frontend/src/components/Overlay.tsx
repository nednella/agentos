import { useEffect, useRef } from 'react'
import type { ReactNode } from 'react'
import { useAgentos } from '../AgentosContext'

type OverlayProps = {
  label: string
  align: 'left' | 'center'
  wide?: boolean
  onClose(): void
  children: ReactNode
}

export function Overlay({ label, align, wide = false, onClose, children }: OverlayProps) {
  const { focus } = useAgentos()
  const previous = useRef(document.activeElement)
  const dialog = useRef<HTMLDivElement>(null)

  useEffect(
    () => () => {
      // Strict Mode's rehearsal unmount leaves the dialog in the page.
      if (dialog.current?.isConnected) return
      const el = previous.current
      if (el instanceof HTMLElement && el.isConnected && el !== document.body) el.focus()
      else focus('terminal')
    },
    [focus],
  )

  return (
    <div
      className="fade-in fixed inset-0"
      style={{ zIndex: 'var(--z-overlay)', background: 'var(--bg-overlay)' }}
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        ref={dialog}
        role="dialog"
        aria-modal="true"
        aria-label={label}
        className={`absolute flex flex-col overflow-hidden rounded-md border border-line-strong bg-surface shadow-2xl ${wide ? 'max-h-[95vh]' : 'max-h-[70vh]'}`}
        style={
          align === 'left'
            ? { top: 'calc(var(--topbar-h) + 4px)', left: '50%', width: 'min(34rem, calc(100vw - 1.5rem))', transform: 'translateX(-50%)' }
            : { top: wide ? '2.5vh' : '12vh', left: '50%', width: wide ? 'min(72rem, 94vw)' : 'min(40rem, 92vw)', transform: 'translateX(-50%)' }
        }
      >
        {children}
      </div>
    </div>
  )
}
