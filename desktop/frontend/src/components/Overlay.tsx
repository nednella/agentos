import { useEffect, useRef } from 'react'
import type { CSSProperties, ReactNode } from 'react'
import { useAgentos } from '../AgentosContext'

type Size = 'default' | 'reading' | 'wide'

const CENTRED: Record<Size, CSSProperties> = {
  default: { top: '12vh', width: 'min(40rem, 92vw)', maxHeight: '70vh', transform: 'translateX(-50%)' },
  reading: { top: '50%', width: 'min(44rem, 92vw)', maxHeight: '84vh', transform: 'translate(-50%, -50%)' },
  wide: { top: '2.5vh', width: 'min(72rem, 94vw)', maxHeight: '95vh', transform: 'translateX(-50%)' },
}

type OverlayProps = {
  label: string
  align: 'left' | 'center'
  size?: Size
  onClose(): void
  children: ReactNode
}

export function Overlay({ label, align, size = 'default', onClose, children }: OverlayProps) {
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
        className="absolute flex flex-col overflow-hidden rounded-md border border-line-strong bg-surface shadow-2xl"
        style={
          align === 'left'
            ? { top: 'calc(var(--topbar-h) + 4px)', left: '50%', width: 'min(34rem, calc(100vw - 1.5rem))', maxHeight: '70vh', transform: 'translateX(-50%)' }
            : { ...CENTRED[size], left: '50%' }
        }
      >
        {children}
      </div>
    </div>
  )
}
