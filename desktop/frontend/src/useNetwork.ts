import { useCallback, useEffect, useState } from 'react'
import { useAgentos } from './AgentosContext'
import { on } from './api'
import { useMinimumLoading } from './useMinimumLoading'

// The app says it is offline when its GitHub and git calls get no answer, and online at the next answer.
export function useNetwork() {
  const { refreshIssues, refreshPRs } = useAgentos()
  const [online, setOnline] = useState(navigator.onLine)
  const [retrying, setRetrying] = useState(false)

  useEffect(() => on('network', (status) => setOnline(status.online)), [])

  useEffect(() => {
    const goOffline = () => setOnline(false)
    window.addEventListener('offline', goOffline)
    return () => window.removeEventListener('offline', goOffline)
  }, [])

  const retry = useCallback(async () => {
    setRetrying(true)
    await Promise.allSettled([refreshIssues(), refreshPRs()])
    setRetrying(false)
  }, [refreshIssues, refreshPRs])

  return { online, retrying: useMinimumLoading(retrying), retry }
}
