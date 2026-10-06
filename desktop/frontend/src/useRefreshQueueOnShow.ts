import { useEffect, useRef } from 'react'
import { useAgentos } from './AgentosContext'
import { useLayout } from './LayoutContext'

const MIN_GAP_MS = 5000

// Issues change on GitHub with no event the app hears, so a queue on screen is read again when it is
// shown, when the project changes and when the window comes back into focus.
export function useRefreshQueueOnShow() {
  const { project, sidebarTab, refreshIssues, report } = useAgentos()
  const { sidebarOpen, sidebarPeek } = useLayout()
  const shown = (sidebarOpen || sidebarPeek) && sidebarTab === 'queue'
  const refresh = useRef<(force?: boolean) => void>(() => {})
  const last = useRef(Date.now())
  refresh.current = (force = false) => {
    if (!force && Date.now() - last.current < MIN_GAP_MS) return
    last.current = Date.now()
    report(refreshIssues)
  }

  const projectKey = project?.key
  const seen = useRef(projectKey)
  useEffect(() => {
    const switched = seen.current !== undefined && seen.current !== projectKey
    seen.current = projectKey
    if (shown) refresh.current(switched)
  }, [shown, projectKey])

  useEffect(() => {
    if (!shown) return
    const onFocus = () => document.visibilityState === 'visible' && refresh.current()
    window.addEventListener('focus', onFocus)
    document.addEventListener('visibilitychange', onFocus)
    return () => {
      window.removeEventListener('focus', onFocus)
      document.removeEventListener('visibilitychange', onFocus)
    }
  }, [shown])
}
