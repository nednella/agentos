import { useEffect } from 'react'
import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'
import { BrowserEmpty } from './BrowserEmpty'
import { BrowserStatus } from './BrowserStatus'

type BrowserTabProps = { session: Session }

export function BrowserTab({ session }: BrowserTabProps) {
  const { browserStates, loadBrowserState } = useAgentos()
  const state = browserStates[session.id]
  const known = Boolean(state)

  useEffect(() => {
    if (session.browser && !known) void loadBrowserState(session.id)
  }, [session.id, session.browser, known, loadBrowserState])

  if (!state?.open) return <BrowserEmpty session={session} error={state?.error ?? ''} />
  return <BrowserStatus session={session} state={state} />
}
