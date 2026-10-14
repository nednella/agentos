import { useEffect } from 'react'

// Marks the panel that holds the focus with data-focused, for the focus styling. WebKit does not
// restyle a panel that lost the focus while the window was not key, so :focus-within left two
// panels looking focused after the click that brings the window to the front.
export function useFocusedPanel() {
  useEffect(() => {
    let focused: Element | null = null
    const update = () => {
      const panel = document.hasFocus() ? (document.activeElement?.closest('[data-panel]') ?? null) : null
      if (panel === focused) return
      focused?.removeAttribute('data-focused')
      panel?.setAttribute('data-focused', '')
      focused = panel
    }
    update()
    document.addEventListener('focusin', update)
    document.addEventListener('focusout', update)
    window.addEventListener('focus', update)
    window.addEventListener('blur', update)
    return () => {
      document.removeEventListener('focusin', update)
      document.removeEventListener('focusout', update)
      window.removeEventListener('focus', update)
      window.removeEventListener('blur', update)
    }
  }, [])
}
