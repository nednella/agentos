import { useEffect, useRef } from 'react'
import { useAgentos } from './AgentosContext'
import { matchShortcut, useActions } from './actions'
import { useLayout } from './LayoutContext'

export function useShortcuts() {
  const a = useAgentos()
  const layout = useLayout()
  const actions = useActions()
  const latest = useRef({ a, layout, actions })
  latest.current = { a, layout, actions }

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const { a: agentos, actions: registry } = latest.current
      const action = registry.find((x) => x.shortcut && matchShortcut(e, x.shortcut))
      if (action) {
        e.preventDefault()
        agentos.report(() => action.run([], 'ui'))
      }
    }
    const onEscape = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      const { a: agentos, layout: lay } = latest.current
      if (agentos.overlay) {
        agentos.setOverlay(null)
        lay.returnToTerminal()
        return
      }
      if (lay.statsOpen || lay.digestOpen) {
        lay.closeCentre()
        agentos.focus('terminal')
        return
      }
      if (lay.mode === 'medium' && lay.sidebarOpen) {
        lay.setSidebarOpen(false)
        agentos.focus('terminal')
        return
      }
      const el = document.activeElement
      if (el instanceof HTMLElement && el.matches('input, textarea') && !el.closest('.xterm')) {
        el.blur()
        agentos.focus('terminal')
      }
    }
    window.addEventListener('keydown', onKeyDown, true)
    window.addEventListener('keydown', onEscape)
    return () => {
      window.removeEventListener('keydown', onKeyDown, true)
      window.removeEventListener('keydown', onEscape)
    }
  }, [])
}
