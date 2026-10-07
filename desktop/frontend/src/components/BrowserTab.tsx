import { useEffect } from 'react'
import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'
import { BrowserEmpty } from './BrowserEmpty'
import { BrowserHeaded } from './BrowserHeaded'
import { BrowserPage } from './BrowserPage'

type BrowserTabProps = { session: Session }

export function BrowserTab({ session }: BrowserTabProps) {
  const { browserStates, loadBrowserState } = useAgentos()
  const state = browserStates[session.id]
  const known = Boolean(state)

  useEffect(() => {
    if (session.browser && !known) void loadBrowserState(session.id)
  }, [session.id, session.browser, known, loadBrowserState])

  if (!state?.open) return <BrowserEmpty session={session} error={state?.error ?? ''} />
  if (state.headed) return <BrowserHeaded session={session} state={state} />
  return <BrowserPage session={session} state={state} />
}
