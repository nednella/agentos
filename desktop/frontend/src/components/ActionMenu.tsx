import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Icon } from './Icon'

type ActionMenuProps = { issue: number; actions: string[]; onPick(action: string, e: React.MouseEvent): void }

// The popup sits in the body, not in the row: a row clips what it reveals on hover.
export function ActionMenu({ issue, actions, onPick }: ActionMenuProps) {
  const [at, setAt] = useState<{ top: number; right: number } | null>(null)
  const popup = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!at) return
    const close = (e: Event) => {
      if (!(e instanceof MouseEvent) || !popup.current?.contains(e.target as Node)) setAt(null)
    }
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setAt(null)
    window.addEventListener('mousedown', close)
    window.addEventListener('keydown', onKey)
    window.addEventListener('blur', close)
    return () => {
      window.removeEventListener('mousedown', close)
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('blur', close)
    }
  }, [at])

  return (
    <>
      <button
        className="btn btn-ghost h-6 w-6 justify-center px-0"
        title="Other actions"
        aria-label={`Other actions for #${issue}`}
        aria-expanded={at !== null}
        onClick={(e) => {
          const box = e.currentTarget.getBoundingClientRect()
          setAt(at ? null : { top: box.bottom + 4, right: window.innerWidth - box.right })
        }}
      >
        <Icon name="chevron" size={11} />
      </button>
      {at &&
        createPortal(
          <div
            ref={popup}
            className="fade-in fixed min-w-32 rounded-md border border-line-strong bg-raised py-1"
            style={{ top: at.top, right: at.right, zIndex: 'var(--z-popup)' }}
            role="menu"
          >
            {actions.map((name) => (
              <button
                key={name}
                role="menuitem"
                className="row h-7 items-center px-3"
                onClick={(e) => {
                  setAt(null)
                  onPick(name, e)
                }}
              >
                {name}
              </button>
            ))}
          </div>,
          document.body,
        )}
    </>
  )
}
