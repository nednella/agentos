import { useAgentos } from '../AgentosContext'
import type { Session } from '../types'
import { BrowserEmpty } from './BrowserEmpty'
import { BrowserPage } from './BrowserPage'

type BrowserTabProps = { session: Session }

export function BrowserTab({ session }: BrowserTabProps) {
  const { browserStates } = useAgentos()
  const state = browserStates[session.id]
  if (!state?.open) return <BrowserEmpty session={session} error={state?.error ?? ''} />
  return <BrowserPage session={session} state={state} />
}
