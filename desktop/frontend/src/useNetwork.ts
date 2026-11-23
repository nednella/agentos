import { useCallback, useEffect, useState } from 'react'
import { useAgentos } from './AgentosContext'
import { on } from './api'
import { useMinimumLoading } from './useMinimumLoading'

// The app says it is offline when its GitHub and git calls get no answer, and online at the next answer.
// The web view's own offline signal is left out: no answer from the app's calls could end it.
export function useNetwork() {
  const { refreshIssues, refreshPRs } = useAgentos('refreshIssues', 'refreshPRs')
  const [online, setOnline] = useState(true)
  const [retrying, setRetrying] = useState(false)

  useEffect(() => on('network', (status) => setOnline(status.online)), [])

  const retry = useCallback(async () => {
    setRetrying(true)
    await Promise.allSettled([refreshIssues(), refreshPRs()])
    setRetrying(false)
  }, [refreshIssues, refreshPRs])

  return { online, retrying: useMinimumLoading(retrying), retry }
}
