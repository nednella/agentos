import type { BrowserState, Session } from '../types'
import { BrowserConsoleList } from './BrowserConsoleList'
import { BrowserRecentCaptures } from './BrowserRecentCaptures'
import { BrowserStatusHeader } from './BrowserStatusHeader'

type BrowserHeadedProps = { session: Session; state: BrowserState }

export function BrowserHeaded({ session, state }: BrowserHeadedProps) {
  return (
    <div className="absolute inset-0 flex flex-col">
      <BrowserStatusHeader id={session.id} state={state} />
      <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto p-4">
        <BrowserConsoleList lines={state.console} />
        <BrowserRecentCaptures session={session} />
      </div>
    </div>
  )
}
