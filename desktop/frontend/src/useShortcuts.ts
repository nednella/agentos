import { useEffect, useRef } from 'react'
import { useLatestAgentos } from './AgentosContext'
import { actionsFor, matchShortcut } from './actions'
import { useLayout } from './LayoutContext'
import { isEditingText } from './useListNav'

export function useShortcuts(enabled: boolean) {
  const latestAgentos = useLatestAgentos()
  const layout = useLayout()
  const latest = useRef({ layout, enabled })
  latest.current = { layout, enabled }

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (!latest.current.enabled) return
      const agentos = latestAgentos()
      const action = actionsFor(agentos, latest.current.layout).find((x) => x.shortcut && matchShortcut(e, x.shortcut) && !(x.shortcut.unlessTyping && isEditingText(e.target)))
      if (action) {
        e.preventDefault()
        agentos.report(() => action.run([]))
      }
    }
    const onEscape = (e: KeyboardEvent) => {
      if (e.key !== 'Escape' || !latest.current.enabled) return
      const agentos = latestAgentos()
      const lay = latest.current.layout
      if (agentos.overlay) {
        agentos.setOverlay(null)
        return
      }
      if (lay.newsOpen && lay.newsPage !== 'index') {
        lay.closeNewsPage()
        return
      }
      if (lay.statsOpen || lay.newsOpen) {
        lay.closeCentre()
        agentos.focus('terminal')
        return
      }
      if (lay.sidebarPeek || lay.sessionsPeek) {
        lay.returnToTerminal()
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
  }, [latestAgentos])
}
