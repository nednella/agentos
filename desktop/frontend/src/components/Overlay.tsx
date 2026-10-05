import type { ReactNode } from 'react'

type OverlayProps = {
  label: string
  align: 'left' | 'center'
  wide?: boolean
  onClose(): void
  children: ReactNode
}

export function Overlay({ label, align, wide = false, onClose, children }: OverlayProps) {
  return (
    <div
      className="fade-in fixed inset-0"
      style={{ zIndex: 'var(--z-overlay)', background: 'var(--bg-overlay)' }}
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
      onKeyDown={(e) => e.key === 'Escape' && onClose()}
    >
      <div
        role="dialog"
        aria-label={label}
        className={`absolute flex flex-col overflow-hidden rounded-md border border-line-strong bg-surface shadow-2xl ${wide ? 'max-h-[88vh]' : 'max-h-[70vh]'}`}
        style={
          align === 'left'
            ? { top: 'calc(var(--topbar-h) + 4px)', left: '50%', width: 'min(34rem, calc(100vw - 1.5rem))', transform: 'translateX(-50%)' }
            : { top: wide ? '5vh' : '12vh', left: '50%', width: wide ? 'min(72rem, 94vw)' : 'min(40rem, 92vw)', transform: 'translateX(-50%)' }
        }
      >
        {children}
      </div>
    </div>
  )
}
